package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/datalog"
)

// runProjectCmd runs the command and reports failure as an error: a typed
// non-zero exit is captured, not returned, at the test process boundary.
func runProjectCmd(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	out, errOut, codes := withCapturedIO(t)
	cmd := newProjectCmd()
	var cobraOut bytes.Buffer
	cmd.SetOut(&cobraOut)
	cmd.SetErr(&cobraOut)
	cmd.SetArgs(args)
	err = executeCapturedCommand(cmd)
	if err == nil && len(*codes) > 0 {
		err = fmt.Errorf("exit codes %v", *codes)
	}
	return out.String(), errOut.String() + cobraOut.String(), err
}

func factsDirSnapshot(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		b.WriteString("== " + name + "\n" + string(body))
	}
	return b.String()
}

func TestProjectFactsWritesAReproducibleDirectory(t *testing.T) {
	design := filepath.Join("..", "..", "examples", "fulfillment", "design")
	dir := filepath.Join(t.TempDir(), "out facts")
	stdout, stderr, err := runProjectCmd(t, "--facts", dir, design)
	if err != nil {
		t.Fatalf("project --facts failed: %v\n%s", err, stderr)
	}
	if !strings.Contains(stdout, "wrote "+dir) {
		t.Fatalf("stdout = %q", stdout)
	}
	first := factsDirSnapshot(t, dir)
	for _, want := range []string{"== relations.txt\n", "== transition.facts\n", "== unit_writes.facts\n", "== supersedes.facts\n"} {
		if !strings.Contains(first, want) {
			t.Fatalf("facts directory lacks %q", want)
		}
	}
	in, err := datalog.ReadFacts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(in["transition"]) == 0 || len(in["unit_writes"]) == 0 {
		t.Fatal("the fulfillment facts must carry transitions and WRITES declarations")
	}
	if _, stderr, err := runProjectCmd(t, "--facts", dir, design); err != nil {
		t.Fatalf("second run failed: %v\n%s", err, stderr)
	}
	if second := factsDirSnapshot(t, dir); second != first {
		t.Fatal("a second run over an unchanged design must write byte-identical files")
	}
	if _, err := os.Stat(filepath.Join(design, "checkers")); !os.IsNotExist(err) {
		t.Fatal("--facts must not write checker projections into the design")
	}
}

func TestProjectFactsRefusesAnUnrelatedDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := runProjectCmd(t, "--facts", dir, filepath.Join("..", "..", "examples", "pii-flow", "design"))
	if err == nil || !strings.Contains(stderr, "keep.txt") {
		t.Fatalf("an unrelated directory must be refused, err=%v stderr=%q", err, stderr)
	}
	if body, err := os.ReadFile(filepath.Join(dir, "keep.txt")); err != nil || string(body) != "mine" {
		t.Fatal("the unrelated directory must be left intact")
	}
}
