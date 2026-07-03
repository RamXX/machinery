package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func withCapturedIO(t *testing.T) (*bytes.Buffer, *bytes.Buffer, *[]int) {
	t.Helper()
	var out, errB bytes.Buffer
	var codes []int
	stdoutW, stderrW = &out, &errB
	exitFunc = func(c int) { codes = append(codes, c) }
	t.Cleanup(func() {
		stdoutW, stderrW = os.Stdout, os.Stderr
		exitFunc = os.Exit
	})
	return &out, &errB, &codes
}

func TestCheckGateG4RequiresImplCaseInsensitive(t *testing.T) {
	// Regression: `--gate G4` (uppercase) used to skip the requires-impl error
	// AND every gate, exiting 0 having verified nothing.
	_, errB, codes := withCapturedIO(t)
	cmd := newCheckCmd()
	cmd.SetArgs([]string{"../../examples/go-crm/design", "--gate", "G4"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "--gate g4 requires --impl") {
		t.Fatalf("stderr %q", errB.String())
	}
}

func TestCheckUnknownGateStillErrors(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	cmd := newCheckCmd()
	cmd.SetArgs([]string{"../../examples/go-crm/design", "--gate", "g9"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "unknown gate") {
		t.Fatalf("stderr %q", errB.String())
	}
}
