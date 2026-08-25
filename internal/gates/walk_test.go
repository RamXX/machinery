package gates

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// skipIfPermsUnenforceable skips tests that rely on chmod 0o000 actually
// denying access: Windows has no such mode, and root ignores it.
func skipIfPermsUnenforceable(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("chmod 0o000 does not deny access on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions")
	}
}

// unreadableDir creates dir with mode 0o000 and restores it on cleanup so
// t.TempDir removal succeeds.
func unreadableDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
}

func relFiles(t *testing.T, root string, files []string) []string {
	t.Helper()
	var out []string
	for _, f := range files {
		rel, err := filepath.Rel(root, f)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, rel)
	}
	sort.Strings(out)
	return out
}

// The prune glob semantics match the post-walk file filters: a directory is
// pruned when it IS the static prefix of an ignore glob or lies under it,
// never when it merely contains the prefix.
func TestDirIgnoredGlobVsDirPrefix(t *testing.T) {
	cases := []struct {
		rel, glob string
		want      bool
	}{
		{"mock-ui", "mock-ui/**", true},
		{"test", "test/**", true},
		{"lib", "lib/tixx/**", false},
		{"lib/tixx", "lib/tixx/**", true},
		{"lib/tixx/sub", "lib/tixx/**", true},
		{"mock-ui", "mix.exs", false},
		{"deps", "deps", true},
	}
	for _, c := range cases {
		if got := dirIgnored(c.rel, []string{c.glob}); got != c.want {
			t.Errorf("dirIgnored(%q, [%q]) = %v, want %v", c.rel, c.glob, got, c.want)
		}
	}
}

// An ignored directory is pruned DURING the walk (its unreadable subtree is
// never touched, so no warning), while a non-ignored symlink into it stays
// followable: the prune keys on the walk path, not the resolved target.
func TestWalkPrunesIgnoredDirsDuringWalk(t *testing.T) {
	skipIfPermsUnenforceable(t)
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "real", "code", "pkg.ex"), "defmodule P do\nend\n")
	unreadableDir(t, filepath.Join(root, "real", "unreadable"))
	if err := os.Symlink(filepath.Join(root, "real", "code"), filepath.Join(root, "mirror")); err != nil {
		t.Fatal(err)
	}

	files, pruned, warns, err := walkSourceFiles(root, []string{"real/**"})
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("pruning must happen during the walk (the ignored unreadable subtree is never read), got warns %v", warns)
	}
	if got := relFiles(t, root, files); len(got) != 1 || got[0] != filepath.Join("mirror", "pkg.ex") {
		t.Fatalf("the symlink into the ignored dir must stay followable and the dir itself pruned, got %v", got)
	}
	if len(pruned) != 1 || pruned[0] != "real" {
		t.Fatalf("the pruned dir must be reported, got %v", pruned)
	}

	// without the ignore glob the unreadable subtree IS reached and warned
	_, _, warns, err = walkSourceFiles(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "unreadable") {
		t.Fatalf("the unpruned walk must warn about the unreadable subtree, got %v", warns)
	}
}

// One unreadable directory between two readable ones: both readable subtrees
// are still scanned and the failure is named, not silently truncated.
func TestWalkContinuesPastUnreadableDir(t *testing.T) {
	skipIfPermsUnenforceable(t)
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a", "a.go"), "package a\n")
	unreadableDir(t, filepath.Join(root, "b"))
	mustWrite(t, filepath.Join(root, "c", "c.go"), "package c\n")

	files, _, warns, err := walkSourceFiles(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join("a", "a.go"), filepath.Join("c", "c.go")}
	if got := relFiles(t, root, files); len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("files from BOTH readable dirs must survive, got %v", got)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], filepath.Join(root, "b")) {
		t.Fatalf("exactly one warning naming the unreadable dir, got %v", warns)
	}
}

func TestWalkSymlinkCycleTerminates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs privileges on Windows")
	}
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "d", "f.go"), "package d\n")
	if err := os.Symlink(root, filepath.Join(root, "d", "loop")); err != nil {
		t.Fatal(err)
	}
	files, _, warns, err := walkSourceFiles(root, nil)
	if err != nil || len(warns) != 0 {
		t.Fatalf("err %v warns %v", err, warns)
	}
	if got := relFiles(t, root, files); len(got) != 1 || got[0] != filepath.Join("d", "f.go") {
		t.Fatalf("the cycle must terminate with each file collected once, got %v", got)
	}
}

func TestWalkDanglingSymlinkSkipped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs privileges on Windows")
	}
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.go"), "package a\n")
	if err := os.Symlink(filepath.Join(root, "missing"), filepath.Join(root, "gone")); err != nil {
		t.Fatal(err)
	}
	files, _, warns, err := walkSourceFiles(root, nil)
	if err != nil || len(warns) != 0 {
		t.Fatalf("a dangling symlink is skipped silently, got err %v warns %v", err, warns)
	}
	if got := relFiles(t, root, files); len(got) != 1 || got[0] != "a.go" {
		t.Fatalf("got %v", got)
	}
}

func TestWalkRootFailureIsFatal(t *testing.T) {
	if _, _, _, err := walkSourceFiles(filepath.Join(t.TempDir(), "missing"), nil); err == nil {
		t.Fatal("an unreadable root must stay a fatal error")
	}
}

// Gate-level wiring: an unreadable subtree that sorts BEFORE the real code
// no longer erases it from the corpus; G4 still judges the edge and reports
// the skipped subtree loudly.
func TestG4PartialWalkIsLoudAndScansRest(t *testing.T) {
	skipIfPermsUnenforceable(t)
	design, impl := writeElixirFixture(t, "  deny: [\"core -> web\"]", "  def build, do: Web.Endpoint.url()\n")
	unreadableDir(t, filepath.Join(impl, "aaa"))
	g := CheckImports(design, impl)
	if !hasErr(g, "core -> web") {
		t.Fatalf("code sorting after the unreadable dir must still be judged, got %v", g.Errs)
	}
	if !hasErr(g, "walk incomplete, subtree skipped") {
		t.Fatalf("the partial walk must be loud, got %v", g.Errs)
	}
}

// Gt corpus wiring: test files on both sides of an unreadable directory are
// scanned, and the truncation is reported on the gate.
func TestGtCorpusSurvivesUnreadableDir(t *testing.T) {
	skipIfPermsUnenforceable(t)
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a", "a_test.go"), "package a\n")
	unreadableDir(t, filepath.Join(root, "m"))
	mustWrite(t, filepath.Join(root, "z", "z_test.exs"), "defmodule ZTest do\nend\n")
	g := NewGate("gt")
	corpus := testCorpus(filepath.Join(root, "no-design"), root, g)
	if len(corpus.files) != 2 {
		t.Fatalf("test files from both readable dirs must be scanned, got %+v", corpus.files)
	}
	if !hasErr(g, "walk incomplete, subtree skipped") {
		t.Fatalf("the partial corpus must be loud, got %v", g.Errs)
	}
}
