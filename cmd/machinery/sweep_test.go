package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSweepFile(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestSweepWholeTokenAndExclusions(t *testing.T) {
	out, _, codes := withCapturedIO(t)
	d := t.TempDir()
	writeSweepFile(t, d, "BUILD.md", "the `guardFoo` contract\nguardFooBar is another unit\nbare guardFoo again\n")
	writeSweepFile(t, d, "machines/X.matrix.md", "| guardFoo | guard |\n")
	writeSweepFile(t, d, "machines/X.oracle.md", "| T-X-01 | guardFoo |\n")
	writeSweepFile(t, d, "formal/X.tla", "guardFoo\n")
	writeSweepFile(t, d, "packs/p.md", "guardFoo\n")
	if err := sweepRun("guardFoo", d, 0); err != nil {
		t.Fatalf("sweepRun: %v", err)
	}
	if len(*codes) != 0 {
		t.Fatalf("exit codes %v, want none", *codes)
	}
	s := out.String()
	if !strings.Contains(s, "BUILD.md  (2)") {
		t.Fatalf("BUILD.md hits wrong:\n%s", s)
	}
	if !strings.Contains(s, filepath.ToSlash("machines/X.matrix.md")+"  (1)") {
		t.Fatalf("matrix hit missing:\n%s", s)
	}
	if strings.Contains(s, "oracle.md") || strings.Contains(s, "formal") || strings.Contains(s, "packs") {
		t.Fatalf("generated artifacts scanned:\n%s", s)
	}
	if strings.Contains(s, "guardFooBar is another unit") {
		t.Fatalf("substring matched as a token:\n%s", s)
	}
	if !strings.Contains(s, "3 mention(s) of 'guardFoo' across 2 file(s)") {
		t.Fatalf("summary wrong:\n%s", s)
	}
}

func TestSweepDottedNameAndNoMentions(t *testing.T) {
	out, _, _ := withCapturedIO(t)
	d := t.TempDir()
	writeSweepFile(t, d, "ARCHITECTURE.md", "| `audit.append` | ops |\nthe audit.appendix is unrelated\n")
	if err := sweepRun("audit.append", d, 0); err != nil {
		t.Fatalf("sweepRun: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "ARCHITECTURE.md  (1)") {
		t.Fatalf("dotted-name hit wrong:\n%s", s)
	}
	out.Reset()
	if err := sweepRun("guardNowhere", d, 0); err != nil {
		t.Fatalf("sweepRun: %v", err)
	}
	if !strings.Contains(out.String(), "no mentions of 'guardNowhere'") {
		t.Fatalf("no-mentions message wrong: %q", out.String())
	}
}

func TestSweepContext(t *testing.T) {
	out, _, _ := withCapturedIO(t)
	d := t.TempDir()
	writeSweepFile(t, d, "notes.md", "before\nthe guardFoo line\nafter\n")
	if err := sweepRun("guardFoo", d, 1); err != nil {
		t.Fatalf("sweepRun: %v", err)
	}
	s := out.String()
	for _, want := range []string{"  1: before", "> 2: the guardFoo line", "  3: after"} {
		if !strings.Contains(s, want) {
			t.Fatalf("context output missing %q:\n%s", want, s)
		}
	}
}
