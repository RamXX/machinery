package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckWarningsAsErrors(t *testing.T) {
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "notes.md"), []byte("hand-written em dash — warning\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, codes := withCapturedIO(t)
	cmd := newCheckCmd()
	cmd.SetArgs([]string{design, "--gate", "gl", "--warnings-as-errors"})
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*codes) != 1 || (*codes)[0] != 1 {
		t.Fatalf("warning did not block: %v", *codes)
	}
}

func TestCheckCompleteRequiresImplAndFullSuite(t *testing.T) {
	design := t.TempDir()
	for _, args := range [][]string{{design, "--complete"}, {design, "--complete", "--gate", "g2", "--impl", design}} {
		_, errB, codes := withCapturedIO(t)
		cmd := newCheckCmd()
		cmd.SetArgs(args)
		if err := executeCapturedCommand(cmd); err != nil {
			t.Fatal(err)
		}
		if len(*codes) != 1 || (*codes)[0] != 1 {
			t.Fatalf("args %v did not fail: %v", args, *codes)
		}
		if !strings.Contains(errB.String(), "--complete") {
			t.Fatalf("args %v diagnostic = %q", args, errB.String())
		}
	}
}

func TestCheckCompleteRejectsIncompleteFinalHandoffWithImpl(t *testing.T) {
	design := t.TempDir()
	copyDirInto(t, "../../examples/go-crm/design", design)
	for _, name := range []string{"attestations.yaml", "domain.modelith.md"} {
		if err := os.Remove(filepath.Join(design, name)); err != nil {
			t.Fatal(err)
		}
	}
	buildPath := filepath.Join(design, "BUILD.md")
	build, err := os.ReadFile(buildPath)
	if err != nil {
		t.Fatal(err)
	}
	writeText(t, buildPath, strings.ReplaceAll(string(build), "Status: closed\n", ""))
	// This is a full, implementation-backed design, not the empty-input or
	// gate-subset validation exercised above. Supplying --impl still cannot
	// earn final handoff while closing artifacts are absent and its declared
	// milestones have not been marked accepted.
	out, _, codes := withCapturedIO(t)
	cmd := newCheckCmd()
	cmd.SetArgs([]string{design, "--complete", "--impl", "../../examples/go-crm/impl"})
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*codes) != 1 || (*codes)[0] != 1 {
		t.Fatalf("incomplete handoff exited %v, want [1]", *codes)
	}
	got := out.String()
	for _, want := range []string{
		"G!-complete  final-handoff completeness",
		"attestations.yaml is required for final handoff",
		"domain.modelith.md is required for final handoff",
		"BUILD.md milestone M0 is not Status: closed",
		"warnings are errors",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("complete output missing %q:\n%s", want, got)
		}
	}
}
