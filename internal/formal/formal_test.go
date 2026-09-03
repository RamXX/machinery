package formal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/alloy"
	"github.com/RamXX/machinery/internal/runtimeclosure"
)

func supportedJavaScript(engineBody string) string {
	return "#!/bin/sh\nif [ \"$1\" = '-XshowSettings:properties' ]; then\n  java_home=$(CDPATH= cd -- \"$(dirname -- \"$0\")/..\" && pwd -P)\n  echo \"    java.home = $java_home\" >&2\n  echo '    java.runtime.version = 21.0.12.1+1-LTS' >&2\n  echo '    java.vendor = Eclipse Adoptium' >&2\n  echo '    java.version = 21.0.12.1' >&2\n  echo '    java.vm.name = OpenJDK 64-Bit Server VM' >&2\n  exit 0\nfi\n" + engineBody
}

func TestOptionalFormalDiscoveryTreatsOnlyENOENTAsAbsent(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	present, err := optionalPathExists(missing)
	if err != nil || present {
		t.Fatalf("missing path = present %t, err %v", present, err)
	}

	dangling := filepath.Join(t.TempDir(), "dangling")
	if err := os.Symlink(filepath.Join(t.TempDir(), "absent"), dangling); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	present, err = optionalPathExists(dangling)
	if err != nil || !present {
		t.Fatalf("dangling symlink was silently treated as absence: present %t, err %v", present, err)
	}

	notDir := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(notDir, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := globExt(filepath.Join(notDir, "child"), ".tla"); err == nil {
		t.Fatal("non-ENOENT discovery failure was silently treated as an empty inventory")
	}
}

func writeJavaRuntime(t *testing.T, javaPath, body string) {
	t.Helper()
	root := filepath.Dir(filepath.Dir(javaPath))
	for _, name := range []string{"bin", "conf", "lib"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "lib", "modules"), []byte("modules"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(javaPath, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := runtimeclosure.JavaClosureDigest(javaPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(runtimeclosure.JavaEnv, javaPath)
	t.Setenv(runtimeclosure.JavaClosureSHAEnv, digest)
}

// captureOutput runs f with os.Stdout and os.Stderr redirected and returns
// what was written to each.
func captureOutput(t *testing.T, f func()) (stdout, stderr string) {
	t.Helper()
	or, ow, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	er, ew, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = ow, ew
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()
	outCh := make(chan string)
	errCh := make(chan string)
	go func() { b, _ := io.ReadAll(or); outCh <- string(b) }()
	go func() { b, _ := io.ReadAll(er); errCh <- string(b) }()
	f()
	ow.Close()
	ew.Close()
	return <-outCh, <-errCh
}

// None of these tests needs java: the failure cases fail before any TLC
// invocation, and gen-only never invokes TLC at all.

func TestVerifyFormalFailsWhenNothingToCheck(t *testing.T) {
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := VerifyFormal(design, false); got != 1 {
		t.Fatalf("empty formal suite returned %d, want 1 (nothing to check is a failure)", got)
	}
	if got := VerifyFormal(design, true); got != 1 {
		t.Fatalf("empty formal suite returned %d in gen-only mode, want 1 (nothing to generate is a failure)", got)
	}
}

func TestVerifyFormalRejectsSymlinkedFormalDirectory(t *testing.T) {
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(design, "formal")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	var rc int
	_, stderr := captureOutput(t, func() { rc = VerifyFormal(design, true) })
	if rc != 1 || !strings.Contains(stderr, "must be a real directory") {
		t.Fatalf("symlinked formal directory accepted: rc=%d stderr=%s", rc, stderr)
	}
}

func TestVerifyFormalRejectsSymlinkedSourceAndArtifact(t *testing.T) {
	for _, tc := range []struct {
		name string
		dir  string
		file string
		body string
	}{
		{"machine source", "machines", "Toy.machine.json", `{"id":"Toy","initial":"A","states":{"A":{"type":"final"}}}`},
		{"committed artifact", "formal", "Toy.tla", "---- MODULE Toy ----\n====\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			for _, dir := range []string{"machines", "formal"} {
				if err := os.MkdirAll(filepath.Join(design, dir), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			outside := filepath.Join(t.TempDir(), tc.file)
			if err := os.WriteFile(outside, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(design, tc.dir, tc.file)); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			var rc int
			_, stderr := captureOutput(t, func() { rc = VerifyFormal(design, true) })
			if rc != 1 || !strings.Contains(stderr, "is a symlink") {
				t.Fatalf("symlinked formal input accepted: rc=%d stderr=%s", rc, stderr)
			}
		})
	}
}

func TestVerifyFormalFailsOnGeneratorError(t *testing.T) {
	// A machine tla_gen cannot translate (nested states) must fail the run,
	// never be skipped while stale committed specs are checked as fresh.
	design := t.TempDir()
	mdir := filepath.Join(design, "machines")
	if err := os.MkdirAll(mdir, 0o755); err != nil {
		t.Fatal(err)
	}
	nested := `{"id":"broken","initial":"A","states":{"A":{"initial":"B","states":{"B":{"type":"final"}}}}}`
	if err := os.WriteFile(filepath.Join(mdir, "Broken.machine.json"), []byte(nested), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := VerifyFormal(design, false); got != 1 {
		t.Fatalf("generator failure returned %d, want 1", got)
	}
	if got := VerifyFormal(design, true); got != 1 {
		t.Fatalf("generator failure returned %d in gen-only mode, want 1", got)
	}
}

func TestVerifyFormalGenOnlyRegeneratesWithoutTLC(t *testing.T) {
	// gen-only must succeed with no java in the loop: specs regenerated, TLC
	// skipped, exit 0. This is the nightly regen gate's code path.
	design := t.TempDir()
	mdir := filepath.Join(design, "machines")
	if err := os.MkdirAll(mdir, 0o755); err != nil {
		t.Fatal(err)
	}
	flat := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	if err := os.WriteFile(filepath.Join(mdir, "Toy.machine.json"), []byte(flat), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := VerifyFormal(design, true); got != 0 {
		t.Fatalf("gen-only on a valid machine returned %d, want 0", got)
	}
	for _, f := range []string{"Toy.tla", "Toy.cfg"} {
		if _, err := os.Stat(filepath.Join(design, "formal", f)); err != nil {
			t.Fatalf("gen-only did not regenerate %s: %v", f, err)
		}
	}
}

func TestVerifyFormalDoesNotGenerateFromPostSnapshotABA(t *testing.T) {
	design := t.TempDir()
	mdir := filepath.Join(design, "machines")
	if err := os.MkdirAll(mdir, 0o755); err != nil {
		t.Fatal(err)
	}
	machine := filepath.Join(mdir, "Toy.machine.json")
	original := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	transient := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"C"}}},"C":{"type":"final"}}}`
	if err := os.WriteFile(machine, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	prior := formalAfterDesignSourceSnapshot
	formalAfterDesignSourceSnapshot = func() {
		if err := os.WriteFile(machine, []byte(transient), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(machine, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { formalAfterDesignSourceSnapshot = prior }()
	if got := VerifyFormal(design, true); got != 0 {
		t.Fatalf("gen-only after live A→B→A returned %d", got)
	}
	body, err := os.ReadFile(filepath.Join(design, "formal", "Toy.tla"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), `"C"`) || !strings.Contains(string(body), `"B"`) {
		t.Fatalf("formal output was derived from transient B input:\n%s", body)
	}
}

func TestVerifyFormalRejectsSourceArtifactCollisionBeforeOverwrite(t *testing.T) {
	design := t.TempDir()
	mdir, fdir := filepath.Join(design, "machines"), filepath.Join(design, "formal")
	if err := os.MkdirAll(mdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fdir, 0o755); err != nil {
		t.Fatal(err)
	}
	flat := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	for _, name := range []string{"One.machine.json", "Two.machine.json"} {
		if err := os.WriteFile(filepath.Join(mdir, name), []byte(flat), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sentinel := []byte("committed sentinel\n")
	if err := os.WriteFile(filepath.Join(fdir, "Toy.tla"), sentinel, 0o644); err != nil {
		t.Fatal(err)
	}
	var rc int
	_, stderr := captureOutput(t, func() { rc = VerifyFormal(design, true) })
	if rc != 1 || !strings.Contains(stderr, "ownership collision") {
		t.Fatalf("collision was not a hard pre-write failure: rc=%d stderr=%s", rc, stderr)
	}
	got, err := os.ReadFile(filepath.Join(fdir, "Toy.tla"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(sentinel) {
		t.Fatalf("collision overwrote committed artifact: %q", got)
	}
}

func TestVerifyFormalBlocksAndPreservesForeignOrphanHalf(t *testing.T) {
	design := t.TempDir()
	fdir := filepath.Join(design, "formal")
	if err := os.MkdirAll(fdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fdir, "Half.tla"), []byte("---- MODULE Half ----\n====\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var rc int
	_, stderr := captureOutput(t, func() { rc = VerifyFormal(design, true) })
	if rc != 1 || !strings.Contains(stderr, "not canonically Machinery-generated") {
		t.Fatalf("foreign orphan half was not blocked: rc=%d stderr=%s", rc, stderr)
	}
	if got, err := os.ReadFile(filepath.Join(fdir, "Half.tla")); err != nil || string(got) != "---- MODULE Half ----\n====\n" {
		t.Fatalf("foreign orphan half changed: %q, %v", got, err)
	}
}

func TestVerifyFormalDeclaredManualPairIsFullyChecked(t *testing.T) {
	design := t.TempDir()
	fdir := filepath.Join(design, "formal")
	if err := os.MkdirAll(fdir, 0o755); err != nil {
		t.Fatal(err)
	}
	tlaBody := manualTLAMarker + "\n---- MODULE Manual ----\nVARIABLE x\nInit == x = 0\nNext == x' = x\nSpec == Init /\\ [][Next]_x\n====\n"
	if err := os.WriteFile(filepath.Join(fdir, "Manual.tla"), []byte(tlaBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fdir, "Manual.cfg"), []byte("SPECIFICATION Spec\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := VerifyFormal(design, true); rc != 0 {
		t.Fatalf("declared manual pair rejected in gen-only: %d", rc)
	}

	dummyJar := filepath.Join(design, "dummy.jar")
	if err := os.WriteFile(dummyJar, []byte("jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha, _ := fileSHA256(dummyJar)
	t.Setenv("TLA_TOOLS_JAR", dummyJar)
	t.Setenv("TLA_TOOLS_JAR_SHA256", sha)
	bin := filepath.Join(design, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	java := filepath.Join(bin, "java")
	writeJavaRuntime(t, java, supportedJavaScript("echo 'No error has been found'\n"))
	t.Setenv("PATH", bin)
	if rc := VerifyFormal(design, false); rc != 0 {
		t.Fatalf("declared manual pair was not TLC checked: %d", rc)
	}
}

func TestVerifyFormalControlFlowOnlySemanticsIsStrictAndCounted(t *testing.T) {
	makeDesign := func(t *testing.T, sem string) string {
		t.Helper()
		design := t.TempDir()
		mdir, fdir := filepath.Join(design, "machines"), filepath.Join(design, "formal")
		if err := os.MkdirAll(mdir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(fdir, 0o755); err != nil {
			t.Fatal(err)
		}
		flat := `{"id":"order","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
		if err := os.WriteFile(filepath.Join(mdir, "Order.machine.json"), []byte(flat), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(fdir, "Order.semantics.yaml"), []byte(sem), 0o644); err != nil {
			t.Fatal(err)
		}
		return design
	}
	valid := "machine: order\npattern: control-flow-only\nreason: lifecycle has event-driven overlays outside the data-refinement algebras\n"
	design := makeDesign(t, valid)
	var rc int
	stdout, stderr := captureOutput(t, func() { rc = VerifyFormal(design, true) })
	if rc != 0 || !strings.Contains(stdout, "1 control-flow-only declaration(s)") {
		t.Fatalf("valid control-flow-only declaration not counted: rc=%d stdout=%s stderr=%s", rc, stdout, stderr)
	}
	for name, sem := range map[string]string{
		"machine mismatch": strings.Replace(valid, "machine: order", "machine: Order", 1),
		"empty reason":     strings.Replace(valid, "reason: lifecycle has event-driven overlays outside the data-refinement algebras", "reason: ''", 1),
		"unknown key":      valid + "stages: [A, B]\n",
	} {
		t.Run(name, func(t *testing.T) {
			design := makeDesign(t, sem)
			var rc int
			_, stderr := captureOutput(t, func() { rc = VerifyFormal(design, true) })
			if rc != 1 || !strings.Contains(stderr, "control-flow-only") {
				t.Fatalf("invalid declaration accepted: rc=%d stderr=%s", rc, stderr)
			}
		})
	}
}

func TestFetchJarVerifiesCachedBytesEveryUse(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "tool.jar")
	if err := os.WriteFile(dest, []byte("trusted"), 0o644); err != nil {
		t.Fatal(err)
	}
	want, _ := fileSHA256(dest)
	if _, err := fetchJar(dest, "unused", "test jar", want); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("mutated"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := fetchJar(dest, "unused", "test jar", want); err == nil || !strings.Contains(err.Error(), "cached") {
		t.Fatalf("mutated cached jar was trusted: %v", err)
	}
}

func TestFetchJarRecoversInterruptedPrivateDownloadResidue(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "tool.jar")
	if err := os.WriteFile(dest, []byte("verified jar"), 0o600); err != nil {
		t.Fatal(err)
	}
	want, _ := fileSHA256(dest)
	residue := filepath.Join(dir, ".machinery-jar-123456")
	if err := os.WriteFile(residue, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := fetchJar(dest, "unused", "test jar", want)
	if err != nil || got != dest {
		t.Fatalf("recover cached jar: got %q, err %v", got, err)
	}
	if _, err := os.Lstat(residue); !os.IsNotExist(err) {
		t.Fatalf("interrupted jar residue remains: %v", err)
	}
}

func TestPinnedJarResolversPropagateUserCacheDirErrors(t *testing.T) {
	t.Setenv("TLA_TOOLS_JAR", "")
	t.Setenv("ALLOY_TOOLS_JAR", "")
	want := errors.New("cache unavailable")
	original := formalUserCacheDir
	formalUserCacheDir = func() (string, error) { return "", want }
	t.Cleanup(func() { formalUserCacheDir = original })

	if _, err := jarPath(); !errors.Is(err, want) {
		t.Fatalf("TLA+ cache resolver error was hidden: %v", err)
	}
	if _, err := alloyJarPath(); !errors.Is(err, want) {
		t.Fatalf("Alloy cache resolver error was hidden: %v", err)
	}
}

func TestFetchJarRejectsMissingAndOversizeContentLength(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    string
	}{
		{
			name: "missing",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.(http.Flusher).Flush()
				_, _ = w.Write([]byte("jar"))
			},
			want: "Content-Length must be present",
		},
		{
			name: "oversize",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", fmt.Sprint(formalJarDownloadLimit+1))
				w.WriteHeader(http.StatusOK)
			},
			want: "exceeds",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()
			dest := filepath.Join(t.TempDir(), "tool.jar")
			if _, err := fetchJar(dest, server.URL, "test jar", strings.Repeat("0", 64)); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid response length was accepted: %v", err)
			}
			if _, err := os.Lstat(dest); !os.IsNotExist(err) {
				t.Fatalf("invalid response published a cache artifact: %v", err)
			}
		})
	}
}

func TestFetchJarSyncsParentDirectoryAfterRename(t *testing.T) {
	body := []byte("verified jar body")
	sum := sha256.Sum256(body)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		_, _ = w.Write(body)
	}))
	defer server.Close()
	dir := t.TempDir()
	dest := filepath.Join(dir, "tool.jar")
	original := syncFormalPathDirectory
	var synced []string
	syncFormalPathDirectory = func(path string) error {
		synced = append(synced, path)
		return original(path)
	}
	t.Cleanup(func() { syncFormalPathDirectory = original })

	got, err := fetchJar(dest, server.URL, "test jar", hex.EncodeToString(sum[:]))
	if err != nil || got != dest {
		t.Fatalf("fetch jar = %q, %v", got, err)
	}
	if len(synced) != 1 || synced[0] != dir {
		t.Fatalf("cache directory syncs = %v, want [%s]", synced, dir)
	}
}

func TestFetchJarPropagatesParentDirectorySyncFailure(t *testing.T) {
	body := []byte("verified jar body")
	sum := sha256.Sum256(body)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		_, _ = w.Write(body)
	}))
	defer server.Close()
	want := errors.New("injected directory sync failure")
	original := syncFormalPathDirectory
	syncFormalPathDirectory = func(string) error { return want }
	t.Cleanup(func() { syncFormalPathDirectory = original })

	dest := filepath.Join(t.TempDir(), "tool.jar")
	if _, err := fetchJar(dest, server.URL, "test jar", hex.EncodeToString(sum[:])); !errors.Is(err, want) {
		t.Fatalf("cache directory sync failure was hidden: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("atomically installed cache artifact changed after reported sync failure: %q, %v", got, err)
	}
}

func TestOverrideChecksumPolicyIsExplicitAndValidated(t *testing.T) {
	t.Setenv("TLA_TOOLS_JAR", "/tmp/custom.jar")
	t.Setenv("TLA_TOOLS_JAR_SHA256", "not-a-checksum")
	if _, err := ensureJar(); err == nil || !strings.Contains(err.Error(), "64 lowercase") {
		t.Fatalf("invalid override checksum accepted: %v", err)
	}
}

func TestTLCSuccessDiagnosticsRejectExitZeroWarnings(t *testing.T) {
	if err := validateTLCSuccessOutput("Model checking completed. No error has been found.\n"); err != nil {
		t.Fatalf("canonical success rejected: %v", err)
	}
	for _, output := range []string{
		"WARNING: liveness was skipped\nNo error has been found\n",
		"FATAL: worker terminated\nNo error has been found\n",
		"ERROR: checkpoint corrupt\nNo error has been found\n",
		"SEVERE: solver degraded\nNo error has been found\n",
		"Assertion failed while evaluating\nNo error has been found\n",
		"FATAL No error has been found\n",
		"No error has been found\nNo error has been found\n",
	} {
		if err := validateTLCSuccessOutput(output); err == nil {
			t.Fatalf("ambiguous exit-zero output accepted: %q", output)
		}
	}
}

func TestTLCSemanticFailureSummaryOmitsDynamicEngineData(t *testing.T) {
	first := strings.Join([]string{
		"TLC2 Version 2.19 on host-a at 2026-01-01 10:11:12 with 8 workers and 512MB heap",
		"Error: Invariant TypeOK is violated.",
		"Error: The behavior up to this point is:",
		"Progress(42) at 2026-01-01: 100 states generated, 91 distinct, 13,456 states/min",
	}, "\n")
	second := strings.Join([]string{
		"TLC2 Version 2.19 on host-z at 2035-12-31 23:59:59 with 64 workers and 64GB heap",
		"Error: Invariant TypeOK is violated.",
		"Error: The behavior up to this point is:",
		"Progress(99) at 2035-12-31: 999999 states generated, 7 distinct, 1 states/min",
	}, "\n")
	want := strings.Join(tlcSemanticFailureSummary(first), "\n")
	if got := strings.Join(tlcSemanticFailureSummary(second), "\n"); got != want {
		t.Fatalf("dynamic TLC data changed diagnostic:\nwant %q\n got %q", want, got)
	}
	for _, private := range []string{"host-", "2026-", "2035-", "states/min", "workers", "heap"} {
		if strings.Contains(want, private) {
			t.Fatalf("semantic diagnostic leaked dynamic token %q: %q", private, want)
		}
	}
}

// FORMAL-F5: gen-only counted every committed .tla/.cfg pair as "regenerated
// from source"; an orphan pair with no source inflated the count and a
// zero-machine design exited 0 claiming a pair was regenerated.

func TestVerifyFormalGenOnlyReportsOnlyWrittenPairs(t *testing.T) {
	design := t.TempDir()
	mdir := filepath.Join(design, "machines")
	fdir := filepath.Join(design, "formal")
	if err := os.MkdirAll(mdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fdir, 0o755); err != nil {
		t.Fatal(err)
	}
	flat := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	if err := os.WriteFile(filepath.Join(mdir, "Toy.machine.json"), []byte(flat), 0o644); err != nil {
		t.Fatal(err)
	}
	// an orphan committed pair no generator produces (reviewer fixture exp-c)
	stale := "---- MODULE Stale ----\n\\* machinery-version: v1\n\\* GENERATED by Machinery test fixture\nVARIABLE x\nInit == x = 0\nNext == x' = x\nSpec == Init /\\ [][Next]_x\n====\n"
	if err := os.WriteFile(filepath.Join(fdir, "Stale.tla"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fdir, "Stale.cfg"), []byte("\\* machinery-version: v1\nSPECIFICATION Spec\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fdir, alloy.OutputName), []byte("// Code generated from stale inputs by machinery alloy. DO NOT EDIT.\n// machinery-version: v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var rc int
	stdout, stderr := captureOutput(t, func() { rc = VerifyFormal(design, true) })
	if rc != 0 {
		t.Fatalf("gen-only returned %d, want successful convergence; stderr=%s", rc, stderr)
	}
	if !strings.Contains(stdout, "1 spec pair(s) regenerated") {
		t.Errorf("gen-only counted committed pairs as regenerated:\n%s", stdout)
	}
	for _, name := range []string{"Stale.tla", "Stale.cfg", alloy.OutputName} {
		if _, err := os.Lstat(filepath.Join(fdir, name)); !os.IsNotExist(err) {
			t.Errorf("stale generated artifact %s survived: %v", name, err)
		}
	}
	if rc2 := VerifyFormal(design, true); rc2 != 0 {
		t.Fatalf("second convergence run failed: %d", rc2)
	}
}

func TestVerifyFormalNothingToGenerateIsHardError(t *testing.T) {
	// A stale pair is converged away once. With no source or declared manual
	// pair left, the following run remains a hard nothing-to-verify error.
	design := t.TempDir()
	fdir := filepath.Join(design, "formal")
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fdir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := "---- MODULE Stale ----\n\\* machinery-version: v1\n\\* GENERATED by Machinery test fixture\nVARIABLE x\nInit == x = 0\nNext == x' = x\nSpec == Init /\\ [][Next]_x\n====\n"
	if err := os.WriteFile(filepath.Join(fdir, "Stale.tla"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fdir, "Stale.cfg"), []byte("\\* machinery-version: v1\nSPECIFICATION Spec\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := VerifyFormal(design, true); rc != 0 {
		t.Fatalf("stale-only cleanup returned %d", rc)
	}
	var rc int
	_, stderr := captureOutput(t, func() { rc = VerifyFormal(design, true) })
	if rc != 1 || !strings.Contains(stderr, "nothing to verify") {
		t.Fatalf("empty post-convergence design was not rejected: rc=%d stderr=%s", rc, stderr)
	}
}

// FORMAL-F6: runTLC's error was discarded; an infrastructure failure (missing
// java, missing jar, timeout) printed a bare FAIL with zero diagnostics.
func TestVerifyFormalPrintsTLCInfrastructureError(t *testing.T) {
	design := t.TempDir()
	mdir := filepath.Join(design, "machines")
	if err := os.MkdirAll(mdir, 0o755); err != nil {
		t.Fatal(err)
	}
	flat := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	if err := os.WriteFile(filepath.Join(mdir, "Toy.machine.json"), []byte(flat), 0o644); err != nil {
		t.Fatal(err)
	}
	// a dummy jar so ensureJar succeeds without a network fetch, and an empty
	// PATH so the java invocation itself fails: the pure infrastructure case
	dummyJar := filepath.Join(design, "dummy.jar")
	if err := os.WriteFile(dummyJar, []byte("not a jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TLA_TOOLS_JAR", dummyJar)
	gotSHA, _ := fileSHA256(dummyJar)
	t.Setenv("TLA_TOOLS_JAR_SHA256", gotSHA)
	t.Setenv("PATH", filepath.Join(design, "empty-path"))
	var rc int
	stdout, _ := captureOutput(t, func() { rc = VerifyFormal(design, false) })
	if rc != 1 {
		t.Fatalf("returned %d, want 1", rc)
	}
	if !strings.Contains(stdout, "FAIL") {
		t.Fatalf("no FAIL line:\n%s", stdout)
	}
	if !strings.Contains(stdout, "error:") {
		t.Errorf("infrastructure error not printed alongside the FAIL:\n%s", stdout)
	}
}

// TLC names its scratch directory from the wall clock at second resolution
// and, until -metadir was routed elsewhere, defaulted it to
// <spec-dir>/states/. Concurrent verify-formal runs on one design (or
// per-machine runs starting within the same second) collided on that name:
// "TLC writes its files to a directory whose name is generated from the
// current time", FileNotFoundException on states/<ts>/nodes_0, and one run's
// cleanup deleting another's state files. Every invocation now gets a private
// metadir under the OS temp dir, removed when TLC exits.

// realPath resolves symlinks so paths under the OS temp dir compare equal
// whichever alias (e.g. /var vs /private/var on macOS) a caller used.
func realPath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func isUnder(t *testing.T, child, parent string) bool {
	t.Helper()
	rel, err := filepath.Rel(realPath(t, parent), realPath(t, child))
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// tlcScratchDirs lists the per-invocation TLC metadirs currently present
// under the OS temp dir.
func tlcScratchDirs(t *testing.T) map[string]bool {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "machinery-tlc-*"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, m := range matches {
		out[m] = true
	}
	return out
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTLCMetaDirIsPrivatePerInvocation(t *testing.T) {
	design := t.TempDir()
	a, err := newTLCMetaDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(a)
	b, err := newTLCMetaDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(b)
	if a == b {
		t.Fatalf("two invocations share a metadir: %s", a)
	}
	for _, d := range []string{a, b} {
		fi, err := os.Stat(d)
		if err != nil || !fi.IsDir() {
			t.Fatalf("metadir %s is not a directory: %v", d, err)
		}
		if !isUnder(t, d, os.TempDir()) {
			t.Errorf("metadir %s is not under the OS temp dir %s", d, os.TempDir())
		}
		if isUnder(t, d, design) {
			t.Errorf("metadir %s is under the design dir %s", d, design)
		}
	}
	args := tlcArgs("x.jar", a,
		filepath.Join(design, "formal", "M.cfg"), filepath.Join(design, "formal", "M.tla"))
	routed, tmpdir := false, false
	for i, arg := range args {
		if arg == "-metadir" && i+1 < len(args) && args[i+1] == a {
			routed = true
			continue
		}
		if arg == "-Djava.io.tmpdir="+a {
			tmpdir = true
			continue
		}
		if arg != a && strings.Contains(arg, "states") {
			t.Errorf("TLC argument %q still names a states/ directory", arg)
		}
	}
	if !routed {
		t.Errorf("TLC arguments do not carry -metadir %s: %v", a, args)
	}
	if !tmpdir {
		t.Errorf("TLC arguments do not point java.io.tmpdir at %s: %v", a, args)
	}
}

func TestRunTLCRemovesItsMetaDir(t *testing.T) {
	// the infrastructure-failure setup from FORMAL-F6: a dummy jar and an
	// empty PATH, so runTLC goes through its whole lifecycle without java
	// Each suite owns its temp root. Looking under the process-global temp root
	// mistakes another concurrently running suite's live metadir for a leak.
	scratchRoot := t.TempDir()
	t.Setenv("TMPDIR", scratchRoot)
	t.Setenv("TMP", scratchRoot)
	t.Setenv("TEMP", scratchRoot)
	design := t.TempDir()
	fdir := filepath.Join(design, "formal")
	if err := os.MkdirAll(fdir, 0o755); err != nil {
		t.Fatal(err)
	}
	tla := filepath.Join(fdir, "Toy.tla")
	cfg := filepath.Join(fdir, "Toy.cfg")
	spec := "---- MODULE Toy ----\nVARIABLE x\nInit == x = 0\nNext == x' = x\nSpec == Init /\\ [][Next]_x\n====\n"
	if err := os.WriteFile(tla, []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("SPECIFICATION Spec\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dummyJar := filepath.Join(design, "dummy.jar")
	if err := os.WriteFile(dummyJar, []byte("not a jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TLA_TOOLS_JAR", dummyJar)
	gotSHA, _ := fileSHA256(dummyJar)
	t.Setenv("TLA_TOOLS_JAR_SHA256", gotSHA)
	t.Setenv("PATH", filepath.Join(design, "empty-path"))
	before := tlcScratchDirs(t)
	if _, err := runTLC(tla, cfg); err == nil {
		t.Fatal("runTLC succeeded with no java on PATH")
	}
	for d := range tlcScratchDirs(t) {
		if !before[d] {
			t.Errorf("metadir %s left behind after the run", d)
		}
	}
	if _, err := os.Stat(filepath.Join(fdir, "states")); !os.IsNotExist(err) {
		t.Errorf("formal/states exists after the run (stat err: %v)", err)
	}
}

func formalTestEnvWithTempRoot(root string) []string {
	keys := []string{"TMPDIR", "TMP", "TEMP"}
	env := make([]string, 0, len(os.Environ())+len(keys))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		overridden := false
		for _, candidate := range keys {
			if strings.EqualFold(key, candidate) {
				overridden = true
				break
			}
		}
		if !overridden {
			env = append(env, entry)
		}
	}
	for _, key := range keys {
		env = append(env, key+"="+root)
	}
	return env
}

func TestTwoConcurrentFormalSuitesKeepTLCMetaCleanupIsolated(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		name   string
		output []byte
		err    error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	for _, name := range []string{"suite-a", "suite-b"} {
		root := filepath.Join(t.TempDir(), name)
		if err := os.Mkdir(root, 0o700); err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(t.Context(), executable,
			"-test.run=^TestRunTLCRemovesItsMetaDir$", "-test.count=1")
		cmd.Env = formalTestEnvWithTempRoot(root)
		go func(name string, cmd *exec.Cmd) {
			<-start
			output, err := cmd.CombinedOutput()
			results <- result{name: name, output: output, err: err}
		}(name, cmd)
	}
	close(start)
	for range 2 {
		result := <-results
		if result.err != nil {
			t.Errorf("%s failed while the other formal suite was active: %v\n%s", result.name, result.err, result.output)
		}
	}
}

func TestRunTLCRejectsJarSwapBetweenResolutionAndUse(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "tool.jar")
	if err := os.WriteFile(jar, []byte("verified"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha, err := fileSHA256(jar)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TLA_TOOLS_JAR", jar)
	t.Setenv("TLA_TOOLS_JAR_SHA256", sha)
	oldHook := formalAfterJarResolved
	formalAfterJarResolved = func(string) {
		if err := os.WriteFile(jar, []byte("swapped"), 0o644); err != nil {
			t.Errorf("swap jar: %v", err)
		}
	}
	t.Cleanup(func() { formalAfterJarResolved = oldHook })
	_, err = runTLC(filepath.Join(dir, "Toy.tla"), filepath.Join(dir, "Toy.cfg"))
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("swapped jar was used: %v", err)
	}
}

func TestRunTLCRejectsExitZeroWarningAlongsideSuccess(t *testing.T) {
	for _, diagnostic := range []string{
		"WARNING: possible liveness error",
		"FATAL: worker terminated",
		"ERROR: checkpoint corrupt",
		"SEVERE: solver degraded",
		"Assertion failed while evaluating",
	} {
		t.Run(strings.Fields(diagnostic)[0], func(t *testing.T) {
			dir := t.TempDir()
			jar := filepath.Join(dir, "tool.jar")
			if err := os.WriteFile(jar, []byte("verified"), 0o644); err != nil {
				t.Fatal(err)
			}
			sha, _ := fileSHA256(jar)
			t.Setenv("TLA_TOOLS_JAR", jar)
			t.Setenv("TLA_TOOLS_JAR_SHA256", sha)
			javaPath := filepath.Join(dir, "runtime", "bin", "java")
			writeJavaRuntime(t, javaPath, supportedJavaScript("echo '"+diagnostic+"'\necho 'No error has been found'\nexit 0\n"))
			t.Setenv("MACHINERY_JAVA", javaPath)
			output, err := runTLC(filepath.Join(dir, "Toy.tla"), filepath.Join(dir, "Toy.cfg"))
			if err == nil || !strings.Contains(err.Error(), "unexpected") || !strings.Contains(output, "No error has been found") {
				t.Fatalf("exit-zero TLC diagnostic was accepted: output=%q err=%v", output, err)
			}
		})
	}
}

func TestRunTLCRedactsPrivateWorkdirFromEngineOutput(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "tool.jar")
	if err := os.WriteFile(jar, []byte("verified"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha, _ := fileSHA256(jar)
	t.Setenv("TLA_TOOLS_JAR", jar)
	t.Setenv("TLA_TOOLS_JAR_SHA256", sha)
	t.Setenv("JAVA_TOOL_OPTIONS", "-javaagent:/hostile.jar")
	t.Setenv("JDK_JAVA_OPTIONS", "-Dhostile=true")
	t.Setenv("CLASSPATH", "/hostile/classpath")
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	engineBody := "printf '%s\\n' \"$@\"\nprintf 'HOME=%s JAVA_TOOL_OPTIONS=%s JDK_JAVA_OPTIONS=%s CLASSPATH=%s TZ=%s LC_ALL=%s\\n' \"$HOME\" \"$JAVA_TOOL_OPTIONS\" \"$JDK_JAVA_OPTIONS\" \"$CLASSPATH\" \"$TZ\" \"$LC_ALL\"\nexit 9\n"
	javaPath := filepath.Join(bin, "java")
	writeJavaRuntime(t, javaPath, supportedJavaScript(engineBody))
	t.Setenv("MACHINERY_JAVA", javaPath)
	var want string
	for i := 0; i < 10; i++ {
		out, err := runTLC(filepath.Join(dir, "Toy.tla"), filepath.Join(dir, "Toy.cfg"))
		if err == nil || strings.Contains(out+err.Error(), "machinery-tlc-") || !strings.Contains(out, "<tlc-workdir>") {
			t.Fatalf("private workdir leaked or stable placeholder absent: out=%q err=%v", out, err)
		}
		for _, forbidden := range []string{"hostile.jar", "hostile=true", "/hostile/classpath"} {
			if strings.Contains(out, forbidden) {
				t.Fatalf("hostile environment reached TLC: %s", out)
			}
		}
		for _, required := range []string{"TZ=UTC", "LC_ALL=C.UTF-8"} {
			if !strings.Contains(out, required) {
				t.Fatalf("deterministic environment lacks %s: %s", required, out)
			}
		}
		got := out + "\n" + err.Error()
		if i == 0 {
			want = got
		} else if got != want {
			t.Fatalf("run %d diagnostic changed:\nwant %s\n got %s", i, want, got)
		}
	}
}

func TestRunTLCRejectsUnsupportedJavaPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, vendor, version, want string
	}{
		{"wrong major", "Eclipse Adoptium", "17.0.12", "require exact pinned Temurin build"},
		{"missing vendor", "", "21.0.12.1", "java.vendor"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			jar := filepath.Join(dir, "tool.jar")
			if err := os.WriteFile(jar, []byte("verified"), 0o644); err != nil {
				t.Fatal(err)
			}
			sha, _ := fileSHA256(jar)
			t.Setenv("TLA_TOOLS_JAR", jar)
			t.Setenv("TLA_TOOLS_JAR_SHA256", sha)
			java := filepath.Join(dir, "bin", "java")
			script := "#!/bin/sh\njava_home=$(CDPATH= cd -- \"$(dirname -- \"$0\")/..\" && pwd -P)\necho \"    java.home = $java_home\" >&2\necho '    java.runtime.version = 21.0.12.1+1-LTS' >&2\necho '    java.vendor = " + tc.vendor + "' >&2\necho '    java.version = " + tc.version + "' >&2\necho '    java.vm.name = OpenJDK 64-Bit Server VM' >&2\n"
			writeJavaRuntime(t, java, script)
			t.Setenv("MACHINERY_JAVA", java)
			if _, err := runTLC(filepath.Join(dir, "Toy.tla"), filepath.Join(dir, "Toy.cfg")); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unsupported Java accepted: %v", err)
			}
		})
	}
}

func TestRunTLCRejectsJavaPathSwapAfterProbe(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "tool.jar")
	if err := os.WriteFile(jar, []byte("verified"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha, _ := fileSHA256(jar)
	t.Setenv("TLA_TOOLS_JAR", jar)
	t.Setenv("TLA_TOOLS_JAR_SHA256", sha)
	java := filepath.Join(dir, "bin", "java")
	writeJavaRuntime(t, java, supportedJavaScript("exit 0\n"))
	t.Setenv("MACHINERY_JAVA", java)
	oldHook := formalAfterJavaProbe
	formalAfterJavaProbe = func(string) {
		if err := os.Rename(java, java+".old"); err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(java, []byte(supportedJavaScript("exit 0\n")), 0o755); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() { formalAfterJavaProbe = oldHook })
	if _, err := runTLC(filepath.Join(dir, "Toy.tla"), filepath.Join(dir, "Toy.cfg")); err == nil || !strings.Contains(err.Error(), "changed identity") {
		t.Fatalf("Java path swap accepted: %v", err)
	}
}

func TestEnsureJarRejectsSymlinkOverride(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.jar")
	link := filepath.Join(dir, "link.jar")
	if err := os.WriteFile(real, []byte("verified"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	sha, _ := fileSHA256(real)
	t.Setenv("TLA_TOOLS_JAR", link)
	t.Setenv("TLA_TOOLS_JAR_SHA256", sha)
	if _, err := ensureJar(); err == nil || !strings.Contains(err.Error(), "non-symlink") {
		t.Fatalf("symlink jar accepted: %v", err)
	}
}

func TestVerifyFormalConcurrentRunsDoNotShareTLCScratch(t *testing.T) {
	if _, err := exec.LookPath("java"); err != nil {
		t.Skip("java not on PATH; TLC cannot run")
	}
	// the jar is a precondition, not the subject: fetch it once up front so a
	// cold cache never turns two concurrent fetches into the test's result
	if _, err := ensureJar(); err != nil {
		t.Skipf("tla2tools.jar unavailable: %v", err)
	}
	src := filepath.Join("..", "..", "examples", "portfolio-engine", "design")
	const runs = 2
	designs := make([]string, runs)
	for i := range designs {
		designs[i] = filepath.Join(t.TempDir(), "design")
		copyTree(t, src, designs[i])
	}
	// watch every design tree for the whole run: a formal/states directory
	// that appears even transiently means TLC scratch landed in the tree
	var seen sync.Map
	stop := make(chan struct{})
	var watch sync.WaitGroup
	watch.Add(1)
	go func() {
		defer watch.Done()
		for {
			for _, d := range designs {
				if _, err := os.Stat(filepath.Join(d, "formal", "states")); err == nil {
					seen.Store(d, true)
				}
			}
			select {
			case <-stop:
				return
			case <-time.After(5 * time.Millisecond):
			}
		}
	}()
	rcs := make([]int, runs)
	stdout, stderr := captureOutput(t, func() {
		var wg sync.WaitGroup
		for i := range designs {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				rcs[i] = VerifyFormal(designs[i], false)
			}(i)
		}
		wg.Wait()
	})
	close(stop)
	watch.Wait()
	for i, rc := range rcs {
		if rc != 0 {
			t.Errorf("concurrent run %d returned %d, want 0", i, rc)
		}
	}
	if strings.Contains(stdout, "FAIL") || strings.Contains(stdout+stderr, "states/") {
		t.Errorf("concurrent runs reported a failure or a states/ path")
	}
	if got := strings.Count(stdout, " passed, 0 failed"); got != runs {
		t.Errorf("%d clean summaries, want %d", got, runs)
	}
	if t.Failed() {
		t.Logf("stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	for _, d := range designs {
		if _, ok := seen.Load(d); ok {
			t.Errorf("TLC scratch appeared under %s/formal/states during the run", d)
		}
		if _, err := os.Stat(filepath.Join(d, "formal", "states")); !os.IsNotExist(err) {
			t.Errorf("%s/formal/states exists after the run (stat err: %v)", d, err)
		}
	}
}

func TestDesignLockRejectsConcurrentVerifier(t *testing.T) {
	fdir := t.TempDir()
	first, err := acquireDesignLock(fdir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acquireDesignLock(fdir); err == nil || !strings.Contains(err.Error(), "already active") {
		t.Fatalf("concurrent verifier acquired the same design lock: %v", err)
	}
	if err := first.releaseAll(); err != nil {
		t.Fatal(err)
	}
	second, err := acquireDesignLock(fdir)
	if err != nil {
		t.Fatalf("released design lock remained stuck: %v", err)
	}
	if err := second.releaseAll(); err != nil {
		t.Fatal(err)
	}
}

func TestArtifactCollectorRejectsPortableCaseAlias(t *testing.T) {
	collector, err := newArtifactCollector()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(collector.root)
	if err := collector.add("first", "Foo.tla", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := collector.add("second", "foo.tla", []byte("b")); err == nil || !strings.Contains(err.Error(), "case-insensitive") {
		t.Fatalf("portable formal alias accepted: %v", err)
	}
}

func TestArtifactCollectorScratchIsOutsideGovernedDesign(t *testing.T) {
	design := t.TempDir()
	collector, err := newArtifactCollector()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(collector.root)
	rel, err := filepath.Rel(design, collector.root)
	if err != nil {
		t.Fatal(err)
	}
	if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		t.Fatalf("pre-publication collector scratch escaped into governed design: %s", collector.root)
	}
}

func TestGeneratedTargetAliasDiagnosticIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.CFG", "a.TLA"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("existing"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]generatedArtifact{
		"A.tla": {body: []byte("new"), owner: "test"},
		"B.cfg": {body: []byte("new"), owner: "test"},
	}
	const want = `existing formal artifact "a.TLA" aliases generated target "A.tla" on case-insensitive filesystems`
	for i := 0; i < 100; i++ {
		err := commitGeneratedArtifacts(dir, files)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("run %d diagnostic = %v, want %q", i, err, want)
		}
	}
}

func TestCommitGeneratedArtifactsRollsBackWholeSet(t *testing.T) {
	dir := t.TempDir()
	original := map[string]string{"A.tla": "old-a", "B.cfg": "old-b"}
	for name, body := range original {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]generatedArtifact{
		"A.tla": {body: []byte("new-a"), owner: "test"},
		"B.cfg": {body: []byte("new-b"), owner: "test"},
	}
	installs := 0
	rename := func(root *os.Root, old, new string) error {
		base := filepath.Base(old)
		if strings.HasPrefix(base, ".machinery-formal-stage-") {
			installs++
			if installs == 2 {
				return fmt.Errorf("injected second-install failure")
			}
		}
		return root.Rename(old, new)
	}
	if err := commitGeneratedArtifactsWithRename(dir, files, rename); err == nil {
		t.Fatal("injected transaction failure succeeded")
	}
	for name, want := range original {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(got) != want {
			t.Errorf("%s after rollback = %q, %v; want %q", name, got, err, want)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(original) {
		t.Fatalf("transaction left scratch artifacts: %v", entries)
	}
}

func TestFormalParkingRenameSuccessBeforeWitnessRecordRecoversOldObject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "A.tla"), []byte("old-a"), 0o644); err != nil {
		t.Fatal(err)
	}
	injected := errors.New("simulated crash after parking rename")
	rename := func(root *os.Root, old, new string) error {
		if old == "A.tla" && new == formalScratchName("backup", "A.tla") {
			if err := root.Rename(old, new); err != nil {
				return err
			}
			if err := syncFormalDir(root); err != nil {
				return err
			}
			return injected
		}
		return root.Rename(old, new)
	}
	err := commitGeneratedArtifactsWithRename(dir, map[string]generatedArtifact{
		"A.tla": {body: []byte("new-a"), owner: "test"},
	}, rename)
	if !errors.Is(err, injected) || strings.Contains(err.Error(), "rollback also failed") {
		t.Fatalf("parking rename crash gap did not recover the same old object: %v", err)
	}
	got, readErr := os.ReadFile(filepath.Join(dir, "A.tla"))
	if readErr != nil || string(got) != "old-a" {
		t.Fatalf("old object was not restored after parking crash gap: %q, %v", got, readErr)
	}
	entries, readDirErr := os.ReadDir(dir)
	if readDirErr != nil || len(entries) != 1 || entries[0].Name() != "A.tla" {
		t.Fatalf("parking crash-gap recovery left residue: %v, %v", entries, readDirErr)
	}
}

func TestCommitGeneratedArtifactsJoinsCommitAndRollbackErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "A.tla"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := map[string]generatedArtifact{"A.tla": {body: []byte("new"), owner: "test"}}
	commitErr := errors.New("injected install failure")
	rollbackErr := errors.New("injected restore failure")
	rename := func(root *os.Root, old, new string) error {
		base := filepath.Base(old)
		if strings.HasPrefix(base, ".machinery-formal-backup-") {
			return rollbackErr
		}
		if strings.HasPrefix(base, ".machinery-formal-stage-") {
			return commitErr
		}
		return root.Rename(old, new)
	}
	err := commitGeneratedArtifactsWithRename(dir, files, rename)
	if !errors.Is(err, commitErr) || !errors.Is(err, rollbackErr) {
		t.Fatalf("combined failure does not expose both causes: %v", err)
	}
}

func writeFormalRecoveryFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func formalRecoveryEntry(target string, existed bool, oldBody, newBody string) formalJournalEntry {
	stageKind := "stage"
	newIdentity := formalAbsentIdentity
	if newBody == "" {
		stageKind = "stage-delete"
	} else {
		newIdentity = formalBodyIdentity([]byte(newBody))
	}
	oldIdentity := formalAbsentIdentity
	oldWitness := formalAbsentIdentity
	if existed {
		oldIdentity = formalBodyIdentity([]byte(oldBody))
		oldWitness = "unix:01:02:03:04"
	}
	oldMode := uint32(0)
	if existed {
		oldMode = 0o644
	}
	newWitness := formalAbsentIdentity
	newMode := uint32(0)
	if newBody != "" {
		newWitness = "unix:05:06:07:08"
		newMode = 0o644
	}
	return formalJournalEntry{
		Target: target, Stage: formalScratchName(stageKind, target), Backup: formalScratchName("backup", target),
		Existed: existed, OldIdentity: oldIdentity, NewIdentity: newIdentity,
		OldWitness: oldWitness, NewWitness: newWitness,
		OldMode: oldMode, NewMode: newMode,
	}
}

func hydrateFormalRecoveryWitnesses(t *testing.T, dir string, entries []formalJournalEntry) {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for i := range entries {
		if entries[i].Existed {
			oldPath := entries[i].Target
			if _, err := root.Lstat(entries[i].Backup); err == nil {
				oldPath = entries[i].Backup
			} else if !os.IsNotExist(err) {
				t.Fatal(err)
			}
			_, exists, info, err := formalRegularSnapshot(root, oldPath, "test old witness")
			if err != nil || !exists {
				t.Fatalf("inspect test old witness: exists=%t err=%v", exists, err)
			}
			entries[i].OldWitness, err = formalNativeWitness(root, oldPath, "test old witness", info)
			if err != nil {
				t.Fatal(err)
			}
			entries[i].OldMode = uint32(info.Mode())
		}
		if !formalEntryDeletes(entries[i]) {
			newPath := entries[i].Stage
			if _, err := root.Lstat(newPath); os.IsNotExist(err) {
				newPath = entries[i].Target
			} else if err != nil {
				t.Fatal(err)
			}
			_, exists, info, err := formalRegularSnapshot(root, newPath, "test new witness")
			if err != nil || !exists {
				t.Fatalf("inspect test new witness: exists=%t err=%v", exists, err)
			}
			entries[i].NewWitness, err = formalNativeWitness(root, newPath, "test new witness", info)
			if err != nil {
				t.Fatal(err)
			}
			entries[i].NewMode = uint32(info.Mode())
		}
	}
}

func seedFormalCrash(t *testing.T, dir, phase string) []formalJournalEntry {
	t.Helper()
	entries := []formalJournalEntry{
		formalRecoveryEntry("A.tla", true, "old-a", "new-a"),
		formalRecoveryEntry("B.cfg", true, "old-b", "new-b"),
		formalRecoveryEntry("C.tla", false, "", "new-c"),
	}
	newBodies := map[string]string{"A.tla": "new-a", "B.cfg": "new-b", "C.tla": "new-c"}
	switch phase {
	case "prepared":
		writeFormalRecoveryFile(t, dir, "A.tla", "old-a")
		writeFormalRecoveryFile(t, dir, "B.cfg", "old-b")
		for _, entry := range entries {
			writeFormalRecoveryFile(t, dir, entry.Stage, newBodies[entry.Target])
		}
	case "parking":
		writeFormalRecoveryFile(t, dir, entries[0].Backup, "old-a")
		writeFormalRecoveryFile(t, dir, "B.cfg", "old-b")
		for _, entry := range entries {
			writeFormalRecoveryFile(t, dir, entry.Stage, newBodies[entry.Target])
		}
	case "installing":
		writeFormalRecoveryFile(t, dir, "A.tla", "new-a")
		writeFormalRecoveryFile(t, dir, "B.cfg", "new-b")
		writeFormalRecoveryFile(t, dir, "C.tla", "new-c")
		writeFormalRecoveryFile(t, dir, entries[0].Backup, "old-a")
		writeFormalRecoveryFile(t, dir, entries[1].Backup, "old-b")
	case "committed":
		writeFormalRecoveryFile(t, dir, "A.tla", "new-a")
		writeFormalRecoveryFile(t, dir, "B.cfg", "new-b")
		writeFormalRecoveryFile(t, dir, "C.tla", "new-c")
		writeFormalRecoveryFile(t, dir, entries[0].Backup, "old-a")
		writeFormalRecoveryFile(t, dir, entries[1].Backup, "old-b")
	default:
		t.Fatalf("unknown crash phase %s", phase)
	}
	hydrateFormalRecoveryWitnesses(t, dir, entries)
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := createFormalJournal(root, entries); err != nil {
		t.Fatal(err)
	}
	for _, recorded := range []string{"parking", "installing", "committed"} {
		if phase == "prepared" {
			break
		}
		if err := appendFormalPhase(root, recorded); err != nil {
			t.Fatal(err)
		}
		if phase == recorded {
			break
		}
	}
	return entries
}

func TestFormalJournalRecoversEveryCrashPhaseOnLockAcquisition(t *testing.T) {
	for _, phase := range []string{"prepared", "parking", "installing", "committed"} {
		t.Run(phase, func(t *testing.T) {
			dir := t.TempDir()
			entries := seedFormalCrash(t, dir, phase)
			lock, err := acquireDesignLock(dir)
			if err != nil {
				t.Fatal(err)
			}
			if err := lock.releaseAll(); err != nil {
				t.Fatal(err)
			}
			want := map[string]string{"A.tla": "old-a", "B.cfg": "old-b"}
			if phase == "committed" {
				want = map[string]string{"A.tla": "new-a", "B.cfg": "new-b", "C.tla": "new-c"}
			}
			for name, body := range want {
				got, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil || string(got) != body {
					t.Fatalf("%s after %s recovery = %q, %v; want %q", name, phase, got, err, body)
				}
			}
			if phase != "committed" {
				if _, err := os.Lstat(filepath.Join(dir, "C.tla")); !os.IsNotExist(err) {
					t.Fatalf("new target survived uncommitted %s recovery: %v", phase, err)
				}
			}
			for _, entry := range entries {
				for _, scratch := range []string{entry.Stage, entry.Backup} {
					if _, err := os.Lstat(filepath.Join(dir, scratch)); !os.IsNotExist(err) {
						t.Fatalf("%s survived recovery: %v", scratch, err)
					}
				}
			}
			if _, err := os.Lstat(filepath.Join(dir, formalJournalName)); !os.IsNotExist(err) {
				t.Fatalf("journal survived recovery: %v", err)
			}
		})
	}
}

func TestFormalCrashRecoveryPreservesLateTargetMutationAndABA(t *testing.T) {
	tests := []struct {
		name     string
		phase    string
		lateBody string
	}{
		{name: "installing user edit", phase: "installing", lateBody: "post-crash-user-edit"},
		{name: "installing old-image ABA", phase: "installing", lateBody: "old-a"},
		{name: "committed user edit", phase: "committed", lateBody: "post-commit-user-edit"},
		{name: "committed old-image ABA", phase: "committed", lateBody: "old-a"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			entries := seedFormalCrash(t, dir, tc.phase)
			if err := os.WriteFile(filepath.Join(dir, "A.tla"), []byte(tc.lateBody), 0o644); err != nil {
				t.Fatal(err)
			}

			if _, err := acquireDesignLock(dir); err == nil || !strings.Contains(err.Error(), "preserving") {
				t.Fatalf("late target mutation was not held for manual recovery: %v", err)
			}
			got, err := os.ReadFile(filepath.Join(dir, "A.tla"))
			if err != nil || string(got) != tc.lateBody {
				t.Fatalf("late target mutation was changed: %q, %v", got, err)
			}
			for _, name := range []string{formalJournalName, entries[0].Backup} {
				if _, err := os.Lstat(filepath.Join(dir, name)); err != nil {
					t.Fatalf("recovery evidence %s was removed after identity mismatch: %v", name, err)
				}
			}
		})
	}
}

func TestFormalCrashRecoveryRejectsSameContentNativeIdentityABA(t *testing.T) {
	tests := []struct {
		name  string
		phase string
		setup func(*testing.T, string, []formalJournalEntry)
		body  string
	}{
		{name: "prepared old target", phase: "prepared", body: "old-a"},
		{name: "installing new target", phase: "installing", body: "new-a"},
		{
			name:  "interrupted rollback old target",
			phase: "installing",
			body:  "old-a",
			setup: func(t *testing.T, dir string, entries []formalJournalEntry) {
				t.Helper()
				if err := os.Remove(filepath.Join(dir, entries[0].Target)); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(filepath.Join(dir, entries[0].Backup), filepath.Join(dir, entries[0].Target)); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			entries := seedFormalCrash(t, dir, tc.phase)
			if tc.setup != nil {
				tc.setup(t, dir, entries)
			}
			replacement := filepath.Join(dir, "replacement")
			if err := os.WriteFile(replacement, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			replacementInfo, err := os.Lstat(replacement)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(dir, entries[0].Target)); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(replacement, filepath.Join(dir, entries[0].Target)); err != nil {
				t.Fatal(err)
			}

			if _, err := acquireDesignLock(dir); err == nil || !strings.Contains(err.Error(), "preserving") {
				t.Fatalf("same-content native identity ABA was accepted: %v", err)
			}
			got, readErr := os.ReadFile(filepath.Join(dir, entries[0].Target))
			currentInfo, statErr := os.Lstat(filepath.Join(dir, entries[0].Target))
			if readErr != nil || statErr != nil || string(got) != tc.body || !os.SameFile(replacementInfo, currentInfo) {
				t.Fatalf("foreign same-content replacement was not preserved: body=%q read=%v stat=%v", got, readErr, statErr)
			}
			if _, err := os.Lstat(filepath.Join(dir, formalJournalName)); err != nil {
				t.Fatalf("failed-closed recovery removed its journal: %v", err)
			}
		})
	}
}

func TestFormalCrashRecoveryPreservesModeMutation(t *testing.T) {
	dir := t.TempDir()
	seedFormalCrash(t, dir, "installing")
	if err := os.Chmod(filepath.Join(dir, "A.tla"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := acquireDesignLock(dir); err == nil || !strings.Contains(err.Error(), "preserving") {
		t.Fatalf("durable mode mutation was accepted: %v", err)
	}
	info, err := os.Lstat(filepath.Join(dir, "A.tla"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("foreign mode mutation was not preserved: mode=%v err=%v", info, err)
	}
	if _, err := os.Lstat(filepath.Join(dir, formalJournalName)); err != nil {
		t.Fatalf("failed-closed mode recovery removed its journal: %v", err)
	}
}

func TestFormalOrdinaryRollbackPreservesLateInstalledMutationAndABA(t *testing.T) {
	for _, lateBody := range []string{"post-install-user-edit", "old-a"} {
		t.Run(lateBody, func(t *testing.T) {
			dir := t.TempDir()
			for target, body := range map[string]string{"A.tla": "old-a", "B.cfg": "old-b"} {
				if err := os.WriteFile(filepath.Join(dir, target), []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			files := map[string]generatedArtifact{
				"A.tla": {body: []byte("new-a"), owner: "test"},
				"B.cfg": {body: []byte("new-b"), owner: "test"},
			}
			injected := errors.New("injected second-install failure")
			rename := func(root *os.Root, old, new string) error {
				if old == formalScratchName("stage", "B.cfg") && new == "B.cfg" {
					if err := os.WriteFile(filepath.Join(dir, "A.tla"), []byte(lateBody), 0o644); err != nil {
						return err
					}
					return injected
				}
				return root.Rename(old, new)
			}

			err := commitGeneratedArtifactsWithRename(dir, files, rename)
			if !errors.Is(err, injected) || !strings.Contains(err.Error(), "rollback also failed") {
				t.Fatalf("late mutation did not stop rollback with both errors: %v", err)
			}
			got, readErr := os.ReadFile(filepath.Join(dir, "A.tla"))
			if readErr != nil || string(got) != lateBody {
				t.Fatalf("late installed mutation was changed: %q, %v", got, readErr)
			}
			for _, evidence := range []string{formalJournalName, formalScratchName("backup", "A.tla")} {
				if _, statErr := os.Lstat(filepath.Join(dir, evidence)); statErr != nil {
					t.Fatalf("rollback removed evidence %s after mismatch: %v", evidence, statErr)
				}
			}
		})
	}
}

func TestFormalOrdinaryRollbackPreservesBehindCursorABA(t *testing.T) {
	dir := t.TempDir()
	for target, body := range map[string]string{"A.tla": "old-a", "B.cfg": "old-b"} {
		if err := os.WriteFile(filepath.Join(dir, target), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]generatedArtifact{
		"A.tla": {body: []byte("new-a"), owner: "test"},
		"B.cfg": {body: []byte("new-b"), owner: "test"},
	}
	injected := errors.New("injected second-install failure")
	installFailed := false
	mutated := false
	var replacementInfo os.FileInfo
	rename := func(root *os.Root, old, new string) error {
		if old == formalScratchName("stage", "B.cfg") && new == "B.cfg" {
			installFailed = true
			return injected
		}
		if installFailed && !mutated && old == formalScratchName("backup", "B.cfg") && new == "B.cfg" {
			if err := root.Rename(old, new); err != nil {
				return err
			}
			replacement := filepath.Join(dir, "A.replacement")
			if err := os.WriteFile(replacement, []byte("new-a"), 0o644); err != nil {
				return err
			}
			var err error
			replacementInfo, err = os.Lstat(replacement)
			if err != nil {
				return err
			}
			if err := os.Remove(filepath.Join(dir, "A.tla")); err != nil {
				return err
			}
			if err := os.Rename(replacement, filepath.Join(dir, "A.tla")); err != nil {
				return err
			}
			mutated = true
			return nil
		}
		return root.Rename(old, new)
	}

	err := commitGeneratedArtifactsWithRename(dir, files, rename)
	if !errors.Is(err, injected) || !strings.Contains(err.Error(), "changed identity, mode, or content since preflight") {
		t.Fatalf("behind-cursor ABA was not rejected: %v", err)
	}
	if !mutated || replacementInfo == nil {
		t.Fatal("behind-cursor ABA hook did not run")
	}
	got, readErr := os.ReadFile(filepath.Join(dir, "A.tla"))
	currentInfo, statErr := os.Lstat(filepath.Join(dir, "A.tla"))
	if readErr != nil || statErr != nil || string(got) != "new-a" || !os.SameFile(replacementInfo, currentInfo) {
		t.Fatalf("foreign same-content replacement was not preserved: body=%q read=%v stat=%v", got, readErr, statErr)
	}
	if _, statErr := os.Lstat(filepath.Join(dir, formalJournalName)); statErr != nil {
		t.Fatalf("failed-closed rollback removed its journal: %v", statErr)
	}
}

func TestFormalRecoveryStaysOnOpenedRootDuringParentSwap(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "formal")
	held := filepath.Join(base, "held-formal")
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	seedFormalCrash(t, dir, "parking")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	swapped := false
	rename := func(opened *os.Root, old, new string) error {
		if !swapped {
			swapped = true
			if err := os.Rename(dir, held); err != nil {
				return err
			}
			if err := os.Symlink(outside, dir); err != nil {
				return err
			}
		}
		return opened.Rename(old, new)
	}
	if err := recoverFormalTransaction(root, rename); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(held, "A.tla"))
	if err != nil || string(got) != "old-a" {
		t.Fatalf("rooted recovery restored %q, %v", got, err)
	}
	got, err = os.ReadFile(sentinel)
	if err != nil || string(got) != "outside" {
		t.Fatalf("outside sentinel changed: %q, %v", got, err)
	}
}

func TestFormalJournalBytesAreDeterministic(t *testing.T) {
	entries := []formalJournalEntry{
		formalRecoveryEntry("A.tla", true, "old-a", "new-a"),
		formalRecoveryEntry("B.cfg", false, "", "new-b"),
	}
	var bodies [][]byte
	for i := 0; i < 2; i++ {
		dir := t.TempDir()
		root, err := os.OpenRoot(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := createFormalJournal(root, entries); err != nil {
			t.Fatal(err)
		}
		if err := appendFormalPhase(root, "parking"); err != nil {
			t.Fatal(err)
		}
		if err := root.Close(); err != nil {
			t.Fatal(err)
		}
		body, err := os.ReadFile(filepath.Join(dir, formalJournalName))
		if err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, body)
	}
	if !bytes.Equal(bodies[0], bodies[1]) {
		t.Fatalf("same formal transaction produced different journals:\n%s\n%s", bodies[0], bodies[1])
	}
}

func TestFormalJournalRecoversPastTornPhaseRecord(t *testing.T) {
	dir := t.TempDir()
	seedFormalCrash(t, dir, "prepared")
	f, err := os.OpenFile(filepath.Join(dir, formalJournalName), os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"phase":"park`); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	lock, err := acquireDesignLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.releaseAll(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "A.tla"))
	if err != nil || string(got) != "old-a" {
		t.Fatalf("torn record recovery restored %q, %v", got, err)
	}
}

func TestFormalJournalRejectsUnsafeAuthority(t *testing.T) {
	t.Run("symlink journal", func(t *testing.T) {
		dir := t.TempDir()
		outside := filepath.Join(t.TempDir(), "sentinel")
		writeFormalRecoveryFile(t, filepath.Dir(outside), filepath.Base(outside), "outside")
		if err := os.Symlink(outside, filepath.Join(dir, formalJournalName)); err != nil {
			t.Fatal(err)
		}
		if _, err := acquireDesignLock(dir); err == nil || !strings.Contains(err.Error(), "non-symlink") {
			t.Fatalf("symlink journal accepted: %v", err)
		}
		got, _ := os.ReadFile(outside)
		if string(got) != "outside" {
			t.Fatalf("outside sentinel changed: %q", got)
		}
	})
	t.Run("special journal", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, formalJournalName), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := acquireDesignLock(dir); err == nil || !strings.Contains(err.Error(), "regular") {
			t.Fatalf("directory journal accepted: %v", err)
		}
	})
	t.Run("malformed journal", func(t *testing.T) {
		dir := t.TempDir()
		writeFormalRecoveryFile(t, dir, formalJournalName, "not-json\n")
		if _, err := acquireDesignLock(dir); err == nil || !strings.Contains(err.Error(), "malformed") {
			t.Fatalf("malformed journal accepted: %v", err)
		}
	})
	t.Run("duplicate journal key", func(t *testing.T) {
		dir := t.TempDir()
		body := `{"version":1,"version":1,"phase":"prepared","entries":[]}` + "\n"
		writeFormalRecoveryFile(t, dir, formalJournalName, body)
		if _, err := acquireDesignLock(dir); err == nil || !strings.Contains(err.Error(), "malformed") {
			t.Fatalf("duplicate journal key accepted: %v", err)
		}
	})
	t.Run("case-aliased journal key", func(t *testing.T) {
		dir := t.TempDir()
		body := `{"Version":1,"phase":"prepared","entries":[]}` + "\n"
		writeFormalRecoveryFile(t, dir, formalJournalName, body)
		if _, err := acquireDesignLock(dir); err == nil || !strings.Contains(err.Error(), "malformed") {
			t.Fatalf("case-aliased journal key accepted: %v", err)
		}
	})
	t.Run("portable aliases", func(t *testing.T) {
		dir := t.TempDir()
		entries := []formalJournalEntry{
			formalRecoveryEntry("A.tla", true, "old-a", "new-a"),
			formalRecoveryEntry("a.TLA", true, "old-b", "new-b"),
		}
		body, _ := encodeFormalRecord(formalJournalHeader{Version: 2, Phase: "prepared", Entries: entries})
		if err := os.WriteFile(filepath.Join(dir, formalJournalName), body, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := acquireDesignLock(dir); err == nil || !strings.Contains(err.Error(), "alias") {
			t.Fatalf("portable alias journal accepted: %v", err)
		}
	})
	t.Run("outside path", func(t *testing.T) {
		base := t.TempDir()
		dir := filepath.Join(base, "formal")
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		outside := filepath.Join(base, "outside")
		if err := os.WriteFile(outside, []byte("sentinel"), 0o644); err != nil {
			t.Fatal(err)
		}
		target := "../outside"
		entry := formalRecoveryEntry(target, true, "old", "new")
		body, _ := encodeFormalRecord(formalJournalHeader{Version: 2, Phase: "prepared", Entries: []formalJournalEntry{entry}})
		if err := os.WriteFile(filepath.Join(dir, formalJournalName), body, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := acquireDesignLock(dir); err == nil || !strings.Contains(err.Error(), "unsafe") {
			t.Fatalf("outside journal path accepted: %v", err)
		}
		got, _ := os.ReadFile(outside)
		if string(got) != "sentinel" {
			t.Fatalf("outside sentinel changed: %q", got)
		}
	})
}

func TestCommitGeneratedArtifactsRejectsSymlinkTargetBeforeMutation(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("sentinel"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "A.tla")); err != nil {
		t.Fatal(err)
	}
	files := map[string]generatedArtifact{"A.tla": {body: []byte("new"), owner: "test"}}
	if err := commitGeneratedArtifacts(dir, files); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink target accepted: %v", err)
	}
	got, err := os.ReadFile(outside)
	if err != nil || string(got) != "sentinel" {
		t.Fatalf("outside target changed: %q, %v", got, err)
	}
}

func TestCommitGeneratedArtifactsStaysOnOpenedRootDuringParentSwap(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "formal")
	held := filepath.Join(base, "held-formal")
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "A.tla"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	swapped := false
	rename := func(root *os.Root, old, new string) error {
		if !swapped {
			swapped = true
			if err := os.Rename(dir, held); err != nil {
				return err
			}
			if err := os.Symlink(outside, dir); err != nil {
				return err
			}
		}
		return root.Rename(old, new)
	}
	files := map[string]generatedArtifact{"A.tla": {body: []byte("new"), owner: "test"}}
	if err := commitGeneratedArtifactsWithRename(dir, files, rename); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(held, "A.tla"))
	if err != nil || string(got) != "new" {
		t.Fatalf("opened root received %q, %v", got, err)
	}
	got, err = os.ReadFile(sentinel)
	if err != nil || string(got) != "outside" {
		t.Fatalf("outside sentinel changed: %q, %v", got, err)
	}
	if _, err := os.Lstat(filepath.Join(outside, "A.tla")); !os.IsNotExist(err) {
		t.Fatalf("ambient replacement received transaction output: %v", err)
	}
}

func TestApplyFormalReleaseMakesSuccessFail(t *testing.T) {
	releaseErr := errors.New("injected formal lock close failure")
	code, err := applyFormalRelease(0, func() error { return releaseErr })
	if code != 1 || !errors.Is(err, releaseErr) {
		t.Fatalf("release failure was hidden: code=%d err=%v", code, err)
	}
}

func TestFormalDeletionRollsBackWithLaterInstallFailure(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "A.tla"), []byte("old-a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Stale.cfg"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	injected := errors.New("install failed")
	rename := func(opened *os.Root, old, new string) error {
		if old == formalScratchName("stage", "A.tla") && new == "A.tla" {
			return injected
		}
		return opened.Rename(old, new)
	}
	err = commitGeneratedArtifactsRoot(root, map[string]generatedArtifact{
		"A.tla": {body: []byte("new-a"), owner: "test"},
	}, []string{"Stale.cfg"}, rename)
	if !errors.Is(err, injected) {
		t.Fatalf("install failure hidden: %v", err)
	}
	for name, want := range map[string]string{"A.tla": "old-a", "Stale.cfg": "stale"} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(got) != want {
			t.Fatalf("rollback %s = %q, %v; want %q", name, got, err, want)
		}
	}
}
