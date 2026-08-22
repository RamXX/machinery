package formal

import (
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

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
	stale := "---- MODULE Stale ----\nVARIABLE x\nInit == x = 0\nNext == x' = x\nSpec == Init /\\ [][Next]_x\n====\n"
	if err := os.WriteFile(filepath.Join(fdir, "Stale.tla"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fdir, "Stale.cfg"), []byte("SPECIFICATION Spec\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var rc int
	stdout, stderr := captureOutput(t, func() { rc = VerifyFormal(design, true) })
	if rc != 0 {
		t.Fatalf("gen-only returned %d, want 0 (an orphan pair warns, it does not fail)", rc)
	}
	if !strings.Contains(stdout, "1 spec pair(s) regenerated") {
		t.Errorf("gen-only counted committed pairs as regenerated:\n%s", stdout)
	}
	if !strings.Contains(stderr, "Stale") || !strings.Contains(stderr, "not regenerated (no source)") {
		t.Errorf("orphan pair not warned about:\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestVerifyFormalNothingToGenerateIsHardError(t *testing.T) {
	// zero machines AND zero relational annotations: even with a committed
	// orphan pair on disk, there is nothing to generate; exit hard.
	design := t.TempDir()
	fdir := filepath.Join(design, "formal")
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fdir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := "---- MODULE Stale ----\nVARIABLE x\nInit == x = 0\nNext == x' = x\nSpec == Init /\\ [][Next]_x\n====\n"
	if err := os.WriteFile(filepath.Join(fdir, "Stale.tla"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fdir, "Stale.cfg"), []byte("SPECIFICATION Spec\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, genOnly := range []bool{true, false} {
		var rc int
		_, stderr := captureOutput(t, func() { rc = VerifyFormal(design, genOnly) })
		if rc != 1 {
			t.Fatalf("genOnly=%v: returned %d, want 1", genOnly, rc)
		}
		if !strings.Contains(stderr, "nothing to generate") {
			t.Errorf("genOnly=%v: stderr does not say nothing to generate:\n%s", genOnly, stderr)
		}
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
	t.Setenv("PATH", filepath.Join(design, "empty-path"))
	var rc int
	stdout, _ := captureOutput(t, func() { rc = VerifyFormal(design, false) })
	if rc != 1 {
		t.Fatalf("returned %d, want 1", rc)
	}
	if !strings.Contains(stdout, "FAIL") {
		t.Fatalf("no FAIL line:\n%s", stdout)
	}
	if !strings.Contains(stdout, "error:") || !strings.Contains(stdout, "java") {
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
