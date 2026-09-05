package install

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestUpdateVerifiesReleaseAndRefreshesRecordedHarnesses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("running-executable replacement semantics differ on Windows")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	t.Setenv("HOME", home)
	receipt := installReceipt{
		SchemaVersion: receiptSchema,
		HomeInstalls:  []homeInstall{{Homes: []string{filepath.Join(home, "a"), filepath.Join(home, "b")}}},
		Targets:       []targetInstall{{Target: "codex"}, {Target: "opencode", Copy: true}},
	}
	writeLegacyReceipt(t, receipt)

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

func TestUpdateRollsBackAllHomesBinaryAndReceiptOnLaterFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("running-executable replacement semantics differ on Windows")
	}
	config := privateConfigDir(t)
	t.Setenv("MACHINERY_CONFIG_DIR", config)
	home := t.TempDir()
	t.Setenv("HOME", home)
	first, second := filepath.Join(home, "first"), filepath.Join(home, "second")
	for _, harness := range []string{first, second} {
		write(t, filepath.Join(harness, "skills", "machinery", "SKILL.md"), "old "+filepath.Base(harness)+"\n")
	}
	receipt := installReceipt{
		SchemaVersion: receiptSchema,
		HomeInstalls: []homeInstall{
			{Homes: []string{first}, Copy: true},
			{Homes: []string{second}, Copy: true},
		},
	}
	writeLegacyReceipt(t, receipt)
	receiptPath, err := installationReceiptPath()
	if err != nil {
		t.Fatal(err)
	}
	receiptBefore, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}

	const tag = "v9.9.4"
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
	installCalls := 0
	runner := func(_ string, args ...string) (string, error) {
		if len(args) == 1 && args[0] == "version" {
			return "machinery version " + tag + "\n", nil
		}
		installCalls++
		var harness string
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "--home" {
				harness = args[i+1]
				break
			}
		}
		if harness == "" {
			return "", fmt.Errorf("test runner received no --home: %v", args)
		}
		write(t, filepath.Join(harness, "skills", "machinery", "SKILL.md"), "new\n")
		if err := declareInstallPresentPostImage(filepath.Join(harness, "skills", "machinery"), filepath.Join(harness, "skills", "machinery")); err != nil {
			t.Fatal(err)
		}
		if installCalls == 1 {
			if err := os.WriteFile(receiptPath, []byte("mutated receipt\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := declareInstallPresentPostImage(receiptPath, receiptPath); err != nil {
				t.Fatal(err)
			}
			return "first refreshed\n", nil
		}
		return "second failed\n", errors.New("injected later-home refresh failure")
	}
	_, err = Update(UpdateOptions{
		Version:     tag,
		Repo:        "acme/machinery",
		Executable:  destination,
		SkipPlugins: true,
		run:         runner,
	})
	if err == nil || !strings.Contains(err.Error(), "injected later-home refresh failure") {
		t.Fatalf("Update error = %v", err)
	}
	for _, harness := range []string{first, second} {
		got, readErr := os.ReadFile(filepath.Join(harness, "skills", "machinery", "SKILL.md"))
		want := "old " + filepath.Base(harness) + "\n"
		if readErr != nil || string(got) != want {
			t.Errorf("%s was not restored: got %q, err %v, want %q", harness, got, readErr, want)
		}
	}
	if got, _ := os.ReadFile(destination); string(got) != "old binary\n" {
		t.Fatalf("binary was not restored: %q", got)
	}
	if got, readErr := os.ReadFile(receiptPath); readErr != nil || !bytes.Equal(got, receiptBefore) {
		t.Fatalf("receipt was not restored: got %q, err %v, want %q", got, readErr, receiptBefore)
	}
}

func TestUpdateExecutesDownloadedBinaryForHarnessRefresh(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
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

func TestBinaryOnlyUpdateCandidateUsesDelegatedParentLock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	t.Setenv("HOME", t.TempDir())
	const tag = "v9.9.3"
	testBinary := strings.ReplaceAll(os.Args[0], "'", "'\\''")
	script := "#!/bin/sh\n" +
		"[ \"$1\" = version ] || exit 2\n" +
		"exec env MACHINERY_TEST_UPDATE_CANDIDATE_TAG=" + tag + " '" + testBinary + "' -test.run=^TestUpdateCandidateStartupHelper$\n"
	server := updateReleaseServer(t, tag, []byte(script), false)
	defer server.Close()
	oldGH := githubBase
	githubBase = server.URL
	defer func() { githubBase = oldGH }()

	destination := filepath.Join(t.TempDir(), "machinery")
	result, err := Update(UpdateOptions{
		Version:     tag,
		Repo:        "acme/machinery",
		Executable:  destination,
		SkipPlugins: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != tag || result.Executable != destination || result.HomeInstalls != 0 || result.TargetInstalls != 0 {
		t.Fatalf("binary-only update result = %+v", result)
	}
}

func TestUpdateCandidateStartupHelper(t *testing.T) {
	tag := os.Getenv("MACHINERY_TEST_UPDATE_CANDIDATE_TAG")
	if tag == "" {
		return
	}
	if err := EnsureActivationConsistency(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("machinery version " + tag)
	os.Exit(0)
}

func TestUpdateChecksumMismatchPreservesExistingBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fixture targets POSIX release assets")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
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
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
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
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
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
		case strings.HasSuffix(r.URL.Path, "/releases/tags/"+tag):
			_, _ = fmt.Fprintf(w, `{"tag_name":%q}`, tag)
		case strings.HasSuffix(r.URL.Path, "/"+asset):
			_, _ = w.Write(candidate)
		case strings.HasSuffix(r.URL.Path, "/checksums-sha256.txt"):
			_, _ = fmt.Fprintf(w, "%s  %s\n", sum, asset)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	oldGH, oldAPI := githubBase, apiBase
	githubBase, apiBase = server.URL, server.URL
	defer func() { githubBase, apiBase = oldGH, oldAPI }()
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
	if err == nil || !strings.Contains(err.Error(), "source asset") {
		t.Fatalf("source failure = %v", err)
	}
	got, _ := os.ReadFile(destination)
	if string(got) != "known-good" {
		t.Fatalf("existing binary changed before source was staged: %q", got)
	}
}

func TestUpdateRefreshFailureRollsBackBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fixture targets POSIX release assets")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	t.Setenv("HOME", t.TempDir())
	const tag = "v9.9.4"
	server := updateReleaseServer(t, tag, []byte("new-binary"), false)
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
		Homes:       []string{filepath.Join(t.TempDir(), ".agents")},
		SkipPlugins: true,
		run: func(_ string, args ...string) (string, error) {
			if len(args) == 1 && args[0] == "version" {
				return "machinery version " + tag, nil
			}
			return "refresh failed", errors.New("synthetic refresh failure")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "direct harness refresh failed") {
		t.Fatalf("Update error = %v, want refresh failure", err)
	}
	got, readErr := os.ReadFile(destination)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "known-good" {
		t.Fatalf("binary after failed refresh = %q, want rollback", got)
	}
}

// TestUpdateRefreshesExistingDefaultInstallInPlace pins the field contract
// that every default install depends on: a binary that already exists next to
// a recorded home group and native targets updates in place, with the binary
// swapped and every placement refreshed from the same release. Refusing this
// shape would make the documented `machinery update` (and a re-run of the
// bootstrap installer) unusable for the default topology.
func TestUpdateRefreshesExistingDefaultInstallInPlace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("running-executable replacement semantics differ on Windows")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	t.Setenv("HOME", home)
	agents := filepath.Join(home, ".agents")
	claude := filepath.Join(home, ".claude")
	writeLegacyReceipt(t, installReceipt{
		SchemaVersion: receiptSchema,
		HomeInstalls:  []homeInstall{{Homes: []string{agents, claude}}},
		Targets:       []targetInstall{{Target: "claude"}, {Target: "codex"}, {Target: "opencode"}},
	})
	role := filepath.Join(agents, "agents", "machinery-fsm-author.md")
	write(t, role, "old-role")

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
		t.Fatalf("existing default install refused an in-place update: %v", err)
	}
	if result.Version != tag || result.HomeInstalls != 1 || result.TargetInstalls != 3 {
		t.Fatalf("result = %+v", result)
	}
	if got, err := os.ReadFile(destination); err != nil || string(got) != string(candidate) {
		t.Fatalf("binary after update = %q, %v; want the release candidate", got, err)
	}
	joined := fmt.Sprint(calls)
	for _, required := range []string{"install --from", "--home " + agents, "--home " + claude, "--target claude", "--target codex", "--target opencode"} {
		if !strings.Contains(joined, required) {
			t.Errorf("refresh calls missing %q: %v", required, calls)
		}
	}
	resolvedDestination, err := filepath.EvalSymlinks(destination)
	if err != nil {
		t.Fatal(err)
	}
	for _, call := range calls {
		if call[0] != resolvedDestination {
			t.Errorf("harness refresh ran %s, want the updated binary %s", call[0], resolvedDestination)
		}
	}
	assertNoInstallJournal(t)
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

func TestReleaseAssetMatrixMatchesPublishedTuples(t *testing.T) {
	tests := []struct {
		goos, goarch, want string
		ok                 bool
	}{
		{"linux", "amd64", "machinery-linux-amd64", true},
		{"linux", "arm64", "machinery-linux-arm64", true},
		{"darwin", "amd64", "machinery-darwin-amd64", true},
		{"darwin", "arm64", "machinery-darwin-arm64", true},
		{"windows", "amd64", "", false},
		{"windows", "arm64", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.goos+"-"+tc.goarch, func(t *testing.T) {
			got, err := releaseAssetNameFor(tc.goos, tc.goarch)
			if got != tc.want || (err == nil) != tc.ok {
				t.Fatalf("asset=%q err=%v, want %q ok=%v", got, err, tc.want, tc.ok)
			}
		})
	}
}

func TestBootstrapDefaultPlanIsPluginAware(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override does not steer os.UserHomeDir on Windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	plan, err := updatePlan(UpdateOptions{BootstrapDefaults: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.HomeInstalls) != 1 || len(plan.HomeInstalls[0].Homes) != 2 {
		t.Fatalf("fresh bootstrap plan = %+v", plan)
	}
	seedCachedMachineryPlugin(t, filepath.Join(home, ".claude"), "machinery")
	plan, err = updatePlan(UpdateOptions{BootstrapDefaults: true})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.ClaudePlugin || len(plan.HomeInstalls) != 1 || len(plan.HomeInstalls[0].Homes) != 1 || filepath.Base(plan.HomeInstalls[0].Homes[0]) != ".agents" {
		t.Fatalf("plugin-aware bootstrap plan = %+v", plan)
	}
}

func TestPluginRefreshFailuresAreAggregatedAndNeverWriteCaches(t *testing.T) {
	plan := refreshPlan{ClaudePlugin: true}
	var calls []string
	run := func(name string, args ...string) (string, error) {
		call := name + " " + strings.Join(args, " ")
		calls = append(calls, call)
		if strings.Contains(call, "claude plugin list --json") {
			return `[{"id":"machinery@machinery","scope":"user"}]`, nil
		}
		if strings.Contains(call, "claude plugin marketplace update") {
			return claudeMarketplaceSuccessOutput(), nil
		}
		if strings.Contains(call, "claude plugin update") {
			return "managed scope", errors.New("denied")
		}
		if strings.Contains(call, "codex plugin list --json") {
			return codexInventory("machinery@machinery", true, true), nil
		}
		if strings.Contains(call, "codex plugin add") {
			return codexPluginAddSuccessOutput(t, "machinery"), nil
		}
		return "", nil
	}
	lookup := func(name string) (string, error) { return "/usr/bin/" + name, nil }
	updates, warnings, obligations, refreshErr := refreshHostPlugins(plan, run, lookup, io.Discard)
	var aggregate *PluginRefreshError
	if !errors.As(refreshErr, &aggregate) || updates != 1 || len(warnings) != 1 || !slices.Contains(obligations, "codex") {
		t.Fatalf("updates=%d warnings=%v obligations=%v error=%v calls=%v", updates, warnings, obligations, refreshErr, calls)
	}
	joined := strings.Join(calls, "\n")
	if !strings.Contains(joined, "claude plugin update machinery@machinery") || !strings.Contains(joined, "codex plugin add machinery@machinery") {
		t.Fatalf("plugin managers were not used: %s", joined)
	}
	if strings.Contains(joined, "/cache/") {
		t.Fatalf("plugin cache was addressed directly: %s", joined)
	}
}

func TestPluginRefreshRejectsExitZeroWarningsForEveryMutation(t *testing.T) {
	tests := []struct {
		name       string
		plan       refreshPlan
		warnCall   string
		wantCalled string
	}{
		{name: "Claude marketplace", plan: refreshPlan{ClaudePlugin: true}, warnCall: "claude plugin marketplace update machinery", wantCalled: "claude plugin marketplace update machinery"},
		{name: "Claude plugin", plan: refreshPlan{ClaudePlugin: true}, warnCall: "claude plugin update machinery@machinery --scope user", wantCalled: "claude plugin update machinery@machinery --scope user"},
		{name: "Codex plugin", plan: refreshPlan{CodexPlugin: true}, warnCall: "codex plugin add machinery@machinery --json", wantCalled: "codex plugin add machinery@machinery --json"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			run := func(name string, args ...string) (string, error) {
				call := filepath.Base(name) + " " + strings.Join(args, " ")
				calls = append(calls, call)
				switch call {
				case tc.warnCall:
					return "warning: deprecated partial refresh\n", nil
				case "claude plugin list --json":
					return `[{"id":"machinery@machinery","scope":"user"}]`, nil
				case "claude plugin marketplace update machinery":
					return claudeMarketplaceSuccessOutput(), nil
				case "claude plugin update machinery@machinery --scope user":
					return claudePluginSuccessOutput("user"), nil
				case "codex plugin list --json":
					return codexInventory("machinery@machinery", true, true), nil
				case "codex plugin add machinery@machinery --json":
					return codexPluginAddSuccessOutput(t, "machinery"), nil
				default:
					return "", fmt.Errorf("unexpected plugin command %q", call)
				}
			}
			lookup := func(name string) (string, error) {
				if (name == "claude" && tc.plan.ClaudePlugin) || (name == "codex" && tc.plan.CodexPlugin) {
					return filepath.Join("/usr/bin", name), nil
				}
				return "", errors.New("not installed")
			}
			updates, warnings, _, err := refreshHostPlugins(tc.plan, run, lookup, io.Discard)
			if err == nil || updates != 0 || len(warnings) != 1 {
				t.Fatalf("updates=%d warnings=%v error=%v calls=%v", updates, warnings, err, calls)
			}
			if !slices.Contains(calls, tc.wantCalled) {
				t.Fatalf("warning command was not called: %v", calls)
			}
			if tc.name == "Claude marketplace" && slices.Contains(calls, "claude plugin update machinery@machinery --scope user") {
				t.Fatalf("plugin scope update continued after untrusted marketplace result: %v", calls)
			}
		})
	}
}

func TestPluginMutationSuccessOutputContracts(t *testing.T) {
	if err := validateClaudeMarketplaceUpdateOutput(claudeMarketplaceSuccessOutput()); err != nil {
		t.Fatal(err)
	}
	for _, scope := range []string{"managed", "user", "project", "local"} {
		if err := validateClaudePluginUpdateOutput(claudePluginSuccessOutput(scope), scope); err != nil {
			t.Fatalf("scope %s: %v", scope, err)
		}
	}
	if err := validateCodexPluginAddOutput(codexPluginAddSuccessOutput(t, "machinery")); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{
		"",
		"warning: success\n",
		`{"pluginId":"machinery@machinery","name":"machinery","marketplaceName":"machinery","version":"0.6.2","installedPath":"relative","authPolicy":"ON_USE"}`,
		`{"pluginId":"other@machinery","name":"machinery","marketplaceName":"machinery","version":"0.6.2","installedPath":"/tmp/plugin","authPolicy":"ON_USE"}`,
	} {
		if err := validateCodexPluginAddOutput(invalid); err == nil {
			t.Fatalf("invalid Codex success output accepted: %q", invalid)
		}
	}
}

func claudeMarketplaceSuccessOutput() string {
	return "Updating marketplace: machinery...Validating local marketplace\n✔ Successfully updated marketplace: machinery\n"
}

func claudePluginSuccessOutput(scope string) string {
	return "Checking for updates for plugin \"machinery@machinery\" at " + scope + " scope…\n✔ machinery is already at the latest version (0.6.2).\n"
}

func codexPluginAddSuccessOutput(t *testing.T, name string) string {
	t.Helper()
	return fmt.Sprintf(`{"pluginId":%q,"name":%q,"marketplaceName":"machinery","version":"0.6.2","installedPath":%q,"authPolicy":"ON_USE"}`, name+"@machinery", name, filepath.Join(t.TempDir(), name))
}

func TestPluginRefreshFailsForZeroClaudeScopesAndUnavailableObligatedCodex(t *testing.T) {
	t.Run("zero Claude scopes", func(t *testing.T) {
		plan := refreshPlan{ClaudePlugin: true}
		run := func(string, ...string) (string, error) { return `[{"id":"other@market","scope":"user"}]`, nil }
		lookup := func(name string) (string, error) {
			if name == "claude" {
				return "/usr/bin/claude", nil
			}
			return "", errors.New("missing")
		}
		_, _, _, err := refreshHostPlugins(plan, run, lookup, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "host plugin refresh failed") {
			t.Fatalf("zero-scope refresh error = %v", err)
		}
	})

	t.Run("missing obligated Codex CLI", func(t *testing.T) {
		plan := refreshPlan{CodexPlugin: true}
		lookup := func(string) (string, error) { return "", errors.New("not found") }
		_, warnings, _, err := refreshHostPlugins(plan, nil, lookup, io.Discard)
		if err == nil || len(warnings) != 1 || !strings.Contains(warnings[0], "not on PATH") {
			t.Fatalf("missing Codex CLI warnings=%v error=%v", warnings, err)
		}
	})
}

func TestBuildRefreshPlanRetainsRecordedHostPluginObligation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override does not steer os.UserHomeDir on Windows")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	t.Setenv("HOME", home)
	preserveInstallDiscoveryHooks(t)
	readPluginCache = func(path string) ([]fs.DirEntry, error) {
		if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(home)+string(os.PathSeparator)) {
			return nil, fmt.Errorf("plugin discovery escaped private test home: %s", path)
		}
		return os.ReadDir(path)
	}
	if err := saveReceipt(installReceipt{SchemaVersion: receiptSchema, HostPlugins: []string{"codex"}}); err != nil {
		t.Fatal(err)
	}
	plan, err := buildRefreshPlan()
	if err != nil {
		t.Fatal(err)
	}
	if !plan.CodexPlugin {
		t.Fatalf("recorded Codex obligation lost: %+v", plan)
	}
}

func TestClaudePluginInventoryScopes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"absent", `[{"id":"other@market","scope":"user"}]`, nil},
		{"user only", `[{"id":"machinery@machinery","scope":"user"}]`, []string{"user"}},
		{"managed only", `[{"id":"machinery@machinery","scope":"managed"}]`, []string{"managed"}},
		{"canonical multi scope", `[{"id":"machinery@machinery","scope":"local"},{"id":"machinery@machinery","scope":"managed"},{"id":"machinery@machinery","scope":"project"},{"id":"machinery@machinery","scope":"user"}]`, []string{"managed", "user", "project", "local"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := claudeMachineryScopes(tc.raw)
			if err != nil || !slices.Equal(got, tc.want) {
				t.Fatalf("scopes=%v, err=%v, want %v", got, err, tc.want)
			}
		})
	}
	if _, err := claudeMachineryScopes(`[{"id":"machinery@machinery","scope":"workspace"}]`); err == nil {
		t.Fatal("unknown Claude scope was accepted")
	}
	invalid := []string{
		`[{"id":"machinery@machinery","id":"other@market","scope":"user"}]`,
		`[{"id":"machinery@machinery","scope":"user","extra":true}]`,
		`[{"id":"machinery@machinery","scope":"user"},{"id":"machinery@machinery","scope":"user"}]`,
		`[{"id":"machinery@machinery","scope":"user"}] {}`,
	}
	for _, raw := range invalid {
		if _, err := claudeMachineryScopes(raw); err == nil {
			t.Fatalf("invalid Claude inventory was accepted: %s", raw)
		}
	}
}

func TestCodexPluginInventoryInstalledStatus(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		installed bool
		wantErr   bool
	}{
		{"absent", `{"installed":[],"available":[]}`, false, false},
		{"not installed", codexInventory("machinery@machinery", false, false), false, false},
		{"disabled", codexInventory("machinery@machinery", true, false), false, false},
		{"installed", codexInventory("machinery@machinery", true, true), true, false},
		{"unknown schema", `{"plugins":[]}`, false, true},
		{"unknown status", `{"installed":[{"pluginId":"machinery@machinery","installed":"yes"}]}`, false, true},
		{"duplicate root", `{"installed":[],"installed":[]}`, false, true},
		{"duplicate entry key", `{"installed":[{"pluginId":"machinery@machinery","pluginId":"other@market","installed":true}]}`, false, true},
		{"duplicate identity", `{"installed":[{"pluginId":"machinery@machinery","installed":true},{"pluginId":"machinery@machinery","installed":false}]}`, false, true},
		{"unknown entry field", `{"installed":[{"pluginId":"machinery@machinery","installed":true,"state":"ready"}]}`, false, true},
		{"wrong available type", `{"installed":[],"available":{}}`, false, true},
		{"valid available inventory", `{"installed":[],"available":[{"pluginId":"other@market","marketplaceName":"market","installed":false,"enabled":false,"marketplaceSource":{"sourceType":"local","source":"/tmp/market"},"installPolicy":"AVAILABLE","authPolicy":"ON_USE"}]}`, false, false},
		{"available unknown field", `{"installed":[],"available":[{"pluginId":"other@market","marketplaceName":"market","installed":false,"enabled":false,"marketplaceSource":{"sourceType":"local","source":"/tmp/market"},"installPolicy":"AVAILABLE","authPolicy":"ON_USE","future":true}]}`, false, true},
		{"available claims installed", `{"installed":[],"available":[{"pluginId":"other@market","marketplaceName":"market","installed":true,"enabled":false,"marketplaceSource":{"sourceType":"local","source":"/tmp/market"},"installPolicy":"AVAILABLE","authPolicy":"ON_USE"}]}`, false, true},
		{"trailing document", `{"installed":[]} {"installed":[]}`, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := codexMachineryInstalled(tc.raw)
			if got != tc.installed || (err != nil) != tc.wantErr {
				t.Fatalf("installed=%v, err=%v", got, err)
			}
		})
	}
}

func codexInventory(id string, installed, enabled bool) string {
	return fmt.Sprintf(`{"installed":[{"pluginId":%q,"name":"machinery","marketplaceName":"machinery","version":"0.6.2","installed":%t,"enabled":%t,"source":{"source":"local","path":"/tmp/machinery"},"installPolicy":"AVAILABLE","authPolicy":"ON_USE"}],"available":[]}`,
		id, installed, enabled)
}

func TestPluginInventoryUnknownFieldDiagnosticsAreCanonical(t *testing.T) {
	want := `Codex plugin inventory has unknown fields ["alpha" "zebra"]`
	for i := 0; i < 100; i++ {
		_, err := codexMachineryInstalled(`{"zebra":true,"installed":[],"alpha":true}`)
		if err == nil || err.Error() != want {
			t.Fatalf("iteration %d diagnostic = %v, want %q", i, err, want)
		}
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
	tarball := sourceTarball(t, "machinery")
	sourceSum := fmt.Sprintf("%x", sha256.Sum256(tarball))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/tags/"+tag):
			_, _ = fmt.Fprintf(w, `{"tag_name":%q}`, tag)
		case strings.HasSuffix(r.URL.Path, "/releases/download/"+tag+"/"+asset):
			_, _ = w.Write(candidate)
		case strings.HasSuffix(r.URL.Path, "/releases/download/"+tag+"/checksums-sha256.txt"):
			_, _ = fmt.Fprintf(w, "%s  %s\n%s  machinery-source.tar.gz\n", sum, asset, sourceSum)
		case strings.HasSuffix(r.URL.Path, "/releases/download/"+tag+"/machinery-source.tar.gz"):
			_, _ = w.Write(tarball)
		default:
			http.NotFound(w, r)
		}
	}))
	oldAPI := apiBase
	apiBase = server.URL
	t.Cleanup(func() { apiBase = oldAPI })
	return server
}

// Codex writes `source` as a tagged union. A local plugin carries the path it
// came from; a remote one carries a connector id and no path. Machinery knew
// only the local arm, so a single curated-marketplace plugin anywhere in the
// inventory aborted the whole update. Shapes below are taken from a real
// `codex plugin list --json` (openai-curated-remote entries).
func TestCodexInventoryAcceptsRemoteSourceArm(t *testing.T) {
	remote := `{"installed":[` +
		`{"pluginId":"machinery@machinery","name":"machinery","marketplaceName":"machinery","version":"0.6.8",` +
		`"installed":true,"enabled":true,"source":{"source":"local","path":"/tmp/machinery"},` +
		`"installPolicy":"AVAILABLE","authPolicy":"ON_USE"},` +
		`{"pluginId":"github@openai-curated-remote","name":"github","marketplaceName":"openai-curated-remote",` +
		`"version":"0.1.12-5f7cd798dc99","installed":true,"enabled":true,` +
		`"source":{"source":"remote","id":"plugin_connector_1p_1a69035c238881919c4190932b2df699"},` +
		`"installPolicy":"AVAILABLE","authPolicy":"ON_INSTALL"}],"available":[]}`
	got, err := codexMachineryInstalled(remote)
	if err != nil {
		t.Fatalf("remote source arm rejected: %v", err)
	}
	if !got {
		t.Fatal("machinery entry not reported installed alongside a remote-source plugin")
	}
}

// Machinery's own entry is the record the update acts on, so a source kind
// this binary does not know is fatal there.
func TestCodexInventoryRejectsUnknownSourceKindOnMachineryEntry(t *testing.T) {
	raw := `{"installed":[{"pluginId":"machinery@machinery","name":"machinery","marketplaceName":"machinery",` +
		`"version":"0.6.8","installed":true,"enabled":true,"source":{"source":"quantum","path":"/tmp/x"},` +
		`"installPolicy":"AVAILABLE","authPolicy":"ON_USE"}],"available":[]}`
	if _, err := codexMachineryInstalled(raw); err == nil ||
		!strings.Contains(err.Error(), `unsupported source kind "quantum"`) {
		t.Fatalf("machinery entry accepted an unknown source kind: %v", err)
	}
}

// Somebody else's plugin growing a shape this binary predates must not abort
// machinery's update. The entry's identity fields are still validated.
func TestCodexInventoryToleratesUnknownSourceKindOnForeignEntry(t *testing.T) {
	raw := `{"installed":[` +
		`{"pluginId":"machinery@machinery","name":"machinery","marketplaceName":"machinery","version":"0.6.8",` +
		`"installed":true,"enabled":true,"source":{"source":"local","path":"/tmp/machinery"},` +
		`"installPolicy":"AVAILABLE","authPolicy":"ON_USE"},` +
		`{"pluginId":"future@somewhere","name":"future","marketplaceName":"somewhere","version":"9.9.9",` +
		`"installed":true,"enabled":true,"source":{"source":"orbital","satellite":"n7","beam":3},` +
		`"installPolicy":"AVAILABLE","authPolicy":"ON_USE"}],"available":[]}`
	got, err := codexMachineryInstalled(raw)
	if err != nil {
		t.Fatalf("a foreign plugin's unknown source kind aborted the update: %v", err)
	}
	if !got {
		t.Fatal("machinery entry not reported installed")
	}
}

// A known arm still has a closed schema: the companion field is required and
// must be a nonempty string, and no extra fields are allowed.
func TestCodexInventoryKnownSourceArmStaysClosed(t *testing.T) {
	for name, source := range map[string]string{
		"remote missing id":   `{"source":"remote"}`,
		"remote empty id":     `{"source":"remote","id":"  "}`,
		"remote extra field":  `{"source":"remote","id":"c1","path":"/tmp/x"}`,
		"local missing path":  `{"source":"local"}`,
		"local wrong compan.": `{"source":"local","id":"c1"}`,
	} {
		t.Run(name, func(t *testing.T) {
			raw := `{"installed":[{"pluginId":"machinery@machinery","name":"machinery",` +
				`"marketplaceName":"machinery","version":"0.6.8","installed":true,"enabled":true,` +
				`"source":` + source + `,"installPolicy":"AVAILABLE","authPolicy":"ON_USE"}],"available":[]}`
			if _, err := codexMachineryInstalled(raw); err == nil {
				t.Fatalf("closed schema accepted %s", source)
			}
		})
	}
}
