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
		"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}}},
		"Won":{"type":"final"}}}`)
	writeMachine(t, d, "DealAggregate.machine.json", `{"id":"dealAggregate","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Closed","guard":"canAdvance"}}},
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
		"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}}},
		"Won":{"type":"final"}}}`)
	writeMachine(t, d, "DealAggregate.machine.json", `{"id":"dealAggregate","_oracle_tag":"DEALAGG","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Closed","guard":"canAdvance"}}},
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
