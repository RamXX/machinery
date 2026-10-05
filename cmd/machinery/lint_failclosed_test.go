package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLintFailsClosedMachineEndToEnd drives the real binary: a machine whose
// not-checked state can reach caller-facing success must fail `machinery
// lint` (the same LintMachine G3 runs) with the fail-closed ERROR, and the
// same machine routed to a failing outcome must pass.
func TestLintFailsClosedMachineEndToEnd(t *testing.T) {
	const violating = `{"id":"gate","initial":"Checking","states":{
		"Checking":{"on":{"pass":{"target":"Passed"},"skip":{"target":"Skipped"},"report":{"target":"Skipped"}}},
		"Skipped":{"tags":["unchecked"],"on":{"report":{"target":"Passed"},"pass":{"target":"Passed"},"skip":{"target":"Passed"}}},
		"Passed":{"type":"final","tags":["accepting"]}}}`
	dir := t.TempDir()
	path := filepath.Join(dir, "Gate.machine.json")
	if err := os.WriteFile(path, []byte(violating), 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, code := runBin(t, "lint", dir)
	if code == 0 {
		t.Fatalf("a fail-open machine must fail lint:\n%s", out)
	}
	if !strings.Contains(out, "ERROR  Gate.machine.json: fail-closed: on:report transition from unchecked state Skipped enters accepting state Passed") {
		t.Fatalf("missing the fail-closed ERROR:\n%s", out)
	}

	fixed := strings.ReplaceAll(violating, `"Skipped":{"tags":["unchecked"],"on":{"report":{"target":"Passed"},"pass":{"target":"Passed"},"skip":{"target":"Passed"}}}`,
		`"Skipped":{"tags":["unchecked"],"on":{"report":{"target":"NotChecked"},"pass":{"target":"NotChecked"},"skip":{"target":"NotChecked"}}},
		"NotChecked":{"type":"final","tags":["omitted"]}`)
	if err := os.WriteFile(path, []byte(fixed), 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, code = runBin(t, "lint", dir)
	if code != 0 || strings.Contains(out, "fail-closed") {
		t.Fatalf("the fail-closed machine must lint clean (exit %d):\n%s", code, out)
	}
}
