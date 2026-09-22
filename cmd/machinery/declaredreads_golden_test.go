package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoldenDeclaredReads pins the design-only Gr warning over the smallest
// fixture that exhibits a compile-time design-file consumer. The impl-backed
// history failure is exercised in internal/gates/reads_gate_test.go.
func TestGoldenDeclaredReads(t *testing.T) {
	root := repoRootDir(t)
	source := filepath.Join(root, "internal", "gates", "testdata", "declared-reads")
	design := filepath.Join(t.TempDir(), "design")
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	arch, err := os.ReadFile(filepath.Join(source, "ARCHITECTURE.md.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	arch = []byte(strings.ReplaceAll(string(arch), "REVIEW_COMMIT", strings.Repeat("0", 40)))
	if err := os.WriteFile(filepath.Join(design, "ARCHITECTURE.md"), arch, 0o644); err != nil {
		t.Fatal(err)
	}
	matrix, err := os.ReadFile(filepath.Join(source, "Principal.matrix.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "machines", "Principal.matrix.md"), matrix, 0o644); err != nil {
		t.Fatal(err)
	}
	out, errS, code := runBin(t, "check", design, "--gate", "gr")
	g := goldenDir(t, "check-declared-reads")
	compareOrUpdate(t, filepath.Join(g, "stdout.txt"), out)
	compareOrUpdate(t, filepath.Join(g, "stderr.txt"), errS)
	compareOrUpdate(t, filepath.Join(g, "exitcode.txt"), fmt.Sprintf("%d\n", code))
}
