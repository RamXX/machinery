package gates

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeclaredReadsRejectsArtifactOnlyCommit(t *testing.T) {
	root, design, impl, _ := declaredReadRepo(t)
	matrix := filepath.Join(design, "machines", "Principal.matrix.md")
	raw, err := os.ReadFile(matrix)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteReadFile(t, matrix, string(raw)+"| RESIDUAL | newly designed token |\n")
	mustRunReadGit(t, root, "add", "design")
	mustRunReadGit(t, root, "commit", "-qm", "change parsed design only")

	g := CheckDeclaredReads(design, impl, design, impl)
	if !hasErr(g, "changed without reader lib/principal_reader.ex in the same commit") {
		t.Fatalf("artifact-only commit was not rejected: errs=%v warns=%v", g.Errs, g.Warns)
	}
}

func TestDeclaredReadsAcceptsArtifactAndReaderInSameCommit(t *testing.T) {
	root, design, impl, _ := declaredReadRepo(t)
	matrix := filepath.Join(design, "machines", "Principal.matrix.md")
	raw, err := os.ReadFile(matrix)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteReadFile(t, matrix, string(raw)+"| RESIDUAL | newly designed token |\n")
	reader := filepath.Join(impl, "lib", "principal_reader.ex")
	raw, err = os.ReadFile(reader)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteReadFile(t, reader, string(raw)+"# handles RESIDUAL\n")
	mustRunReadGit(t, root, "add", ".")
	mustRunReadGit(t, root, "commit", "-qm", "change parsed design with reader")

	g := CheckDeclaredReads(design, impl, design, impl)
	if len(g.Errs) != 0 || g.Counts["paired artifact changes"] != 1 {
		t.Fatalf("paired commit failed: errs=%v counts=%v", g.Errs, g.Counts)
	}
}

func TestDeclaredReadsWarnsWithoutImplementation(t *testing.T) {
	_, design, _, _ := declaredReadRepo(t)
	g := CheckDeclaredReads(design, "", design, "")
	if len(g.Errs) != 0 || len(g.Warns) != 1 || !strings.Contains(g.Warns[0], "an implementation reads this file") {
		t.Fatalf("design-only run did not state the obligation: errs=%v warns=%v", g.Errs, g.Warns)
	}
}

func declaredReadRepo(t *testing.T) (root, design, impl, reviewed string) {
	t.Helper()
	root = t.TempDir()
	design = filepath.Join(root, "design")
	impl = filepath.Join(root, "impl")
	mustCopyReadFixture(t, "Principal.matrix.md", filepath.Join(design, "machines", "Principal.matrix.md"), "")
	mustCopyReadFixture(t, "principal_reader.ex", filepath.Join(impl, "lib", "principal_reader.ex"), "")
	mustWriteReadFile(t, filepath.Join(design, "ARCHITECTURE.md"), "# Architecture\n")
	mustRunReadGit(t, root, "init", "-q")
	mustRunReadGit(t, root, "config", "user.email", "fixture@example.invalid")
	mustRunReadGit(t, root, "config", "user.name", "Declared Reads Fixture")
	mustRunReadGit(t, root, "add", ".")
	mustRunReadGit(t, root, "commit", "-qm", "reviewed baseline")
	reviewed = strings.TrimSpace(mustRunReadGit(t, root, "rev-parse", "HEAD"))
	mustCopyReadFixture(t, "ARCHITECTURE.md.tmpl", filepath.Join(design, "ARCHITECTURE.md"), reviewed)
	mustRunReadGit(t, root, "add", "design/ARCHITECTURE.md")
	mustRunReadGit(t, root, "commit", "-qm", "declare compile-time design read")
	return root, design, impl, reviewed
}

func mustCopyReadFixture(t *testing.T, name, dst, reviewed string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "declared-reads", name))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ReplaceAll(string(raw), "REVIEW_COMMIT", reviewed)
	mustWriteReadFile(t, dst, text)
}

func mustWriteReadFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRunReadGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}
