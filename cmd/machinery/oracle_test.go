package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMachine(t *testing.T, dir, name, src string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOracleRefusesLintFailingMachine(t *testing.T) {
	// IR-F14: `machinery oracle` generated from machines that fail lint
	// (silently narrowing array targets to their first element, among others).
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Bad.machine.json", `{"id":"m14","initial":"Idle","states":{
		"Idle":{"on":{"GO":{"target":["Done","Other"]}}},
		"Done":{"type":"final"},
		"Other":{"type":"final"}}}`)
	_ = oracleRun(d)
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "fails lint") {
		t.Fatalf("stderr %q", errB.String())
	}
	if leftovers, _ := filepath.Glob(filepath.Join(d, "*.oracle.md")); len(leftovers) != 0 {
		t.Fatalf("oracle written for a lint-failing machine: %v", leftovers)
	}
}

func TestOracleTagCollisionIsError(t *testing.T) {
	// IR-F12: Deal and DealAggregate both derive the tag DEAL; identical
	// stable ids across machines in one design must be a hard error.
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Deal.machine.json", `{"id":"deal","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Won":{"type":"final"}}}`)
	writeMachine(t, d, "DealAggregate.machine.json", `{"id":"dealAggregate","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Closed","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Closed":{"type":"final"}}}`)
	_ = oracleRun(d)
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "_oracle_tag") {
		t.Fatalf("stderr should point at _oracle_tag, got %q", errB.String())
	}
}

func TestOracleTagOverrideDisambiguates(t *testing.T) {
	out, _, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Deal.machine.json", `{"id":"deal","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Won":{"type":"final"}}}`)
	writeMachine(t, d, "DealAggregate.machine.json", `{"id":"dealAggregate","_oracle_tag":"DEALAGG","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Closed","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Closed":{"type":"final"}}}`)
	if err := oracleRun(d); err != nil {
		t.Fatalf("oracleRun: %v (stdout %q)", err, out.String())
	}
	if len(*codes) != 0 {
		t.Fatalf("exit codes %v, want none", *codes)
	}
	body, err := os.ReadFile(filepath.Join(d, "DealAggregate.oracle.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "DEALAGG-") {
		t.Fatalf("override tag missing from generated oracle:\n%s", body)
	}
}

const validMachineA = `{"id":"alpha","initial":"Lead","states":{
	"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
	"Won":{"type":"final"}}}`

const validMachineB = `{"id":"beta","initial":"Open","states":{
	"Open":{"on":{"close":{"target":"Closed","guard":"canClose"}},"_refusal":{"close":"fixture: the command boundary refuses when canClose is false"}},
	"Closed":{"type":"final"}}}`

func TestOracleDirectoryRegeneratesValidPastBroken(t *testing.T) {
	// S12: one half-written machine must not block the whole directory. The
	// valid machines regenerate, the broken one is named, the run exits 1.
	out, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Alpha.machine.json", validMachineA)
	writeMachine(t, d, "Beta.machine.json", validMachineB)
	writeMachine(t, d, "Broken.machine.json", `{"id":"broken","initial":`)
	_ = oracleRun(d)
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	for _, want := range []string{"Alpha.oracle.md", "Beta.oracle.md"} {
		if _, err := os.Stat(filepath.Join(d, want)); err != nil {
			t.Fatalf("%s not generated past the broken sibling (stdout %q, stderr %q)", want, out.String(), errB.String())
		}
	}
	if _, err := os.Stat(filepath.Join(d, "Broken.oracle.md")); err == nil {
		t.Fatal("oracle written for an unparseable machine")
	}
	if !strings.Contains(errB.String(), "Broken") && !strings.Contains(errB.String(), "broken") {
		t.Fatalf("broken machine not named on stderr: %q", errB.String())
	}
	if !strings.Contains(errB.String(), "1 machine(s) failed") {
		t.Fatalf("failure summary missing: %q", errB.String())
	}
}

func TestOracleSingleFileMode(t *testing.T) {
	// S12: a named file regenerates exactly that file, leaving siblings alone.
	out, _, codes := withCapturedIO(t)
	d := t.TempDir()
	fa := writeMachine(t, d, "Alpha.machine.json", validMachineA)
	writeMachine(t, d, "Beta.machine.json", validMachineB)
	if err := oracleRunFiles([]string{fa}); err != nil {
		t.Fatalf("oracleRunFiles: %v (stdout %q)", err, out.String())
	}
	if len(*codes) != 0 {
		t.Fatalf("exit codes %v, want none", *codes)
	}
	if _, err := os.Stat(filepath.Join(d, "Alpha.oracle.md")); err != nil {
		t.Fatal("named file's oracle not generated")
	}
	if _, err := os.Stat(filepath.Join(d, "Beta.oracle.md")); err == nil {
		t.Fatal("unnamed sibling's oracle generated by a per-file run")
	}
}

func TestOracleSingleFileReservesSiblingTags(t *testing.T) {
	// S12: a per-file run must not silently mint a stable-id tag a directory
	// sibling already owns; both ids share the derived tag here.
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	fa := writeMachine(t, d, "Deal.machine.json", `{"id":"deal","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Won":{"type":"final"}}}`)
	writeMachine(t, d, "DealAggregate.machine.json", `{"id":"dealAggregate","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Closed","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Closed":{"type":"final"}}}`)
	_ = oracleRunFiles([]string{fa})
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "_oracle_tag") {
		t.Fatalf("stderr should point at _oracle_tag, got %q", errB.String())
	}
	if _, err := os.Stat(filepath.Join(d, "Deal.oracle.md")); err == nil {
		t.Fatal("oracle generated for a machine whose tag a sibling contests")
	}
}

func TestOracleLintFailureSkipsOnlyThatMachine(t *testing.T) {
	// S12: a lint-failing machine is refused without blocking a valid sibling.
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Alpha.machine.json", validMachineA)
	writeMachine(t, d, "Bad.machine.json", `{"id":"m14","initial":"Idle","states":{
		"Idle":{"on":{"GO":{"target":["Done","Other"]}}},
		"Done":{"type":"final"},
		"Other":{"type":"final"}}}`)
	_ = oracleRun(d)
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if _, err := os.Stat(filepath.Join(d, "Alpha.oracle.md")); err != nil {
		t.Fatal("valid sibling not generated past the lint-failing machine")
	}
	if _, err := os.Stat(filepath.Join(d, "Bad.oracle.md")); err == nil {
		t.Fatal("oracle written for a lint-failing machine")
	}
	if !strings.Contains(errB.String(), "fails lint") {
		t.Fatalf("stderr %q", errB.String())
	}
}

func TestOracleNonMachineFileArgIsError(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	p := writeMachine(t, d, "notes.txt", "hello")
	_ = oracleRunFiles([]string{p})
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "not a *.machine.json") {
		t.Fatalf("stderr %q", errB.String())
	}
}
