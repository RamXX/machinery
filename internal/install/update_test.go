package install

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUpdateVerifiesReleaseAndRefreshesRecordedHarnesses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("running-executable replacement semantics differ on Windows")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	receipt := installReceipt{
		SchemaVersion: receiptSchema,
		HomeInstalls:  []homeInstall{{Homes: []string{filepath.Join(home, "a"), filepath.Join(home, "b")}}},
		Targets:       []targetInstall{{Target: "codex"}, {Target: "opencode", Copy: true}},
	}
	if err := saveReceipt(receipt); err != nil {
		t.Fatal(err)
	}

	const tag = "v9.9.9"
	candidate := []byte("new machinery binary\n")
	server := updateReleaseServer(t, tag, candidate, false)
	defer server.Close()
	oldGH := githubBase
	githubBase = server.URL
	defer func() { githubBase = oldGH }()

	destination := filepath.Join(t.TempDir(), "machinery")
	if err := os.WriteFile(destination, []byte("old binary\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	runner := func(name string, args ...string) (string, error) {
		if len(args) == 1 && args[0] == "version" {
			return "machinery version " + tag + "\n", nil
		}
		calls = append(calls, append([]string{name}, args...))
		return "refreshed\n", nil
	}
	result, err := Update(UpdateOptions{
		Version:     tag,
		Repo:        "acme/machinery",
		Executable:  destination,
		SkipPlugins: true,
		run:         runner,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != tag || result.HomeInstalls != 1 || result.TargetInstalls != 2 {
		t.Fatalf("result = %+v", result)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(candidate) {
		t.Fatalf("updated binary = %q, want %q", got, candidate)
	}
	if len(calls) != 3 {
		t.Fatalf("refresh calls = %v, want home + two copy-mode target groups", calls)
	}
	joined := fmt.Sprint(calls)
	for _, required := range []string{"--home", filepath.Join(home, "a"), "--target codex", "--target opencode", "--copy"} {
		if !strings.Contains(joined, required) {
			t.Errorf("refresh calls missing %q: %v", required, calls)
		}
	}
}

func TestUpdateExecutesDownloadedBinaryForHarnessRefresh(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	const tag = "v9.9.5"
	logPath := filepath.Join(t.TempDir(), "candidate.log")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = version ]; then printf 'machinery version " + tag + "\\n'; exit 0; fi\n" +
		"printf '%s\\n' \"$*\" >> '" + strings.ReplaceAll(logPath, "'", "'\\''") + "'\n"
	server := updateReleaseServer(t, tag, []byte(script), false)
	defer server.Close()
	oldGH := githubBase
	githubBase = server.URL
	defer func() { githubBase = oldGH }()
	destination := filepath.Join(t.TempDir(), "machinery")
	if err := os.WriteFile(destination, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	harnessHome := filepath.Join(t.TempDir(), ".agents")
	if _, err := Update(UpdateOptions{
		Version:     tag,
		Repo:        "acme/machinery",
		Executable:  destination,
		Homes:       []string{harnessHome},
		SkipPlugins: true,
	}); err != nil {
		t.Fatal(err)
	}
	logRaw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	log := string(logRaw)
	for _, required := range []string{"install --from", "--home " + harnessHome} {
		if !strings.Contains(log, required) {
			t.Errorf("candidate invocation missing %q: %s", required, log)
		}
	}
}

func TestUpdateChecksumMismatchPreservesExistingBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fixture targets POSIX release assets")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	const tag = "v9.9.8"
	server := updateReleaseServer(t, tag, []byte("untrusted"), true)
	defer server.Close()
	oldGH := githubBase
	githubBase = server.URL
	defer func() { githubBase = oldGH }()
	destination := filepath.Join(t.TempDir(), "machinery")
	if err := os.WriteFile(destination, []byte("known-good"), 0o755); err != nil {
		t.Fatal(err)
	}
	called := false
	_, err := Update(UpdateOptions{
		Version:     tag,
		Repo:        "acme/machinery",
		Executable:  destination,
		SkipPlugins: true,
		run: func(string, ...string) (string, error) {
			called = true
			return "", nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("checksum error = %v", err)
	}
	if called {
		t.Fatal("an unverified candidate must never execute")
	}
	got, _ := os.ReadFile(destination)
	if string(got) != "known-good" {
		t.Fatalf("existing binary changed after checksum failure: %q", got)
	}
}

func TestUpdateRejectsCandidateVersionBeforeReplacement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fixture targets POSIX release assets")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	const tag = "v9.9.7"
	server := updateReleaseServer(t, tag, []byte("wrong-version-binary"), false)
	defer server.Close()
	oldGH := githubBase
	githubBase = server.URL
	defer func() { githubBase = oldGH }()
	destination := filepath.Join(t.TempDir(), "machinery")
	if err := os.WriteFile(destination, []byte("known-good"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Update(UpdateOptions{
		Version:     tag,
		Repo:        "acme/machinery",
		Executable:  destination,
		SkipPlugins: true,
		run: func(string, ...string) (string, error) {
			return "machinery version v0.0.1", nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "downloaded binary reports") {
		t.Fatalf("version validation error = %v", err)
	}
	got, _ := os.ReadFile(destination)
	if string(got) != "known-good" {
		t.Fatalf("existing binary changed after version validation failure: %q", got)
	}
}

func TestUpdateSourceFailurePreservesExistingBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fixture targets POSIX release assets")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	const tag = "v9.9.6"
	candidate := []byte("candidate")
	asset, err := releaseAssetName()
	if err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(candidate))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/"+asset):
			_, _ = w.Write(candidate)
		case strings.HasSuffix(r.URL.Path, "/checksums-sha256.txt"):
			_, _ = fmt.Fprintf(w, "%s  %s\n", sum, asset)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	oldGH := githubBase
	githubBase = server.URL
	defer func() { githubBase = oldGH }()
	destination := filepath.Join(t.TempDir(), "machinery")
	if err := os.WriteFile(destination, []byte("known-good"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = Update(UpdateOptions{
		Version:     tag,
		Repo:        "acme/machinery",
		Executable:  destination,
		Homes:       []string{filepath.Join(t.TempDir(), ".agents")},
		SkipPlugins: true,
		run: func(string, ...string) (string, error) {
			return "machinery version " + tag, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "download source tarball") {
		t.Fatalf("source failure = %v", err)
	}
	got, _ := os.ReadFile(destination)
	if string(got) != "known-good" {
		t.Fatalf("existing binary changed before source was staged: %q", got)
	}
}

func TestValidateReleaseBinaryRunsCandidate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	candidate := filepath.Join(t.TempDir(), "machinery")
	if err := os.WriteFile(candidate, []byte("#!/bin/sh\nprintf 'machinery version v1.2.3\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := validateReleaseBinary(candidate, "v1.2.3", runCombined); err != nil {
		t.Fatal(err)
	}
}

func TestPluginRefreshIsBestEffortAndNeverWritesCaches(t *testing.T) {
	plan := refreshPlan{ClaudePlugin: true}
	var calls []string
	run := func(name string, args ...string) (string, error) {
		call := name + " " + strings.Join(args, " ")
		calls = append(calls, call)
		if strings.Contains(call, "claude plugin update") {
			return "managed scope", errors.New("denied")
		}
		if strings.Contains(call, "codex plugin list") {
			return "machinery@machinery", nil
		}
		return "", nil
	}
	lookup := func(name string) (string, error) { return "/usr/bin/" + name, nil }
	updates, warnings := refreshHostPlugins(plan, run, lookup, io.Discard)
	if updates != 1 || len(warnings) != 1 {
		t.Fatalf("updates=%d warnings=%v calls=%v", updates, warnings, calls)
	}
	joined := strings.Join(calls, "\n")
	if !strings.Contains(joined, "claude plugin update machinery@machinery") || !strings.Contains(joined, "codex plugin add machinery@machinery") {
		t.Fatalf("plugin managers were not used: %s", joined)
	}
	if strings.Contains(joined, "/cache/") {
		t.Fatalf("plugin cache was addressed directly: %s", joined)
	}
}

func updateReleaseServer(t *testing.T, tag string, candidate []byte, badChecksum bool) *httptest.Server {
	t.Helper()
	asset, err := releaseAssetName()
	if err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(candidate))
	if badChecksum {
		sum = strings.Repeat("0", 64)
	}
	tarball := sourceTarball(t, "machinery-"+strings.TrimPrefix(tag, "v"))
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/download/"+tag+"/"+asset):
			_, _ = w.Write(candidate)
		case strings.HasSuffix(r.URL.Path, "/releases/download/"+tag+"/checksums-sha256.txt"):
			_, _ = fmt.Fprintf(w, "%s  %s\n", sum, asset)
		case strings.HasSuffix(r.URL.Path, "/archive/refs/tags/"+tag+".tar.gz"):
			_, _ = w.Write(tarball)
		default:
			http.NotFound(w, r)
		}
	}))
}
