package formal

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/alloy"
)

// The fixture receipt was produced by `alloy exec` on the go-crm Policy.als
// generated from the FAITHFUL annotation (the invariants as originally
// written), so it carries both failure modes: the teamless-Manager
// counterexample on CapableWritesOwn and the outsider handoff on
// ReassignRetainsAuthority.
func loadReceipt(t *testing.T) alloyReceipt {
	t.Helper()
	raw, err := os.ReadFile("testdata/policy-receipt.json")
	if err != nil {
		t.Fatal(err)
	}
	var r alloyReceipt
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestVerdicts(t *testing.T) {
	r := loadReceipt(t)
	commands := []alloy.Command{
		{Kind: "run", Name: "SomeWorld"},
		{Kind: "check", Name: "WriteImpliesRead"},
		{Kind: "check", Name: "CapableWritesOwn"},
		{Kind: "check", Name: "ReassignRetainsAuthority"},
		{Kind: "run", Name: "Possible_Rep_update"},
	}
	vs, err := verdicts(r, commands, func(name string) string { return "counterexample: rendered-" + name })
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"SomeWorld":                true,  // run, SAT: world exists
		"WriteImpliesRead":         true,  // check, UNSAT: holds
		"CapableWritesOwn":         false, // check, SAT: the teamless Manager
		"ReassignRetainsAuthority": false, // check, SAT: the outsider handoff
		"Possible_Rep_update":      true,  // run, SAT: grant exercisable
	}
	for _, v := range vs {
		if v.Pass != want[v.Command.Name] {
			t.Errorf("%s: pass = %v, want %v", v.Command.Name, v.Pass, want[v.Command.Name])
		}
	}
	// failing checks carry the rendered counterexample
	for _, v := range vs {
		if v.Command.Name == "CapableWritesOwn" && v.Detail != "counterexample: rendered-CapableWritesOwn" {
			t.Errorf("CapableWritesOwn detail = %q", v.Detail)
		}
	}
}

func TestVerdictsMissingCommand(t *testing.T) {
	r := loadReceipt(t)
	_, err := verdicts(r, []alloy.Command{{Kind: "check", Name: "NotThere"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "no result for command") {
		t.Errorf("want missing-command error, got %v", err)
	}
}

func TestVerdictFailedRunDetail(t *testing.T) {
	// a run with no solution is a vacuity failure, with an explanatory detail
	r := alloyReceipt{Commands: map[string]alloyCommandResult{
		"Possible_X_read": {Type: "run"},
	}}
	vs, err := verdicts(r, []alloy.Command{{Kind: "run", Name: "Possible_X_read"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if vs[0].Pass || !strings.Contains(vs[0].Detail, "no instance within scope") {
		t.Errorf("verdict = %+v; want failed run with vacuity detail", vs[0])
	}
}

// The text fixture is the real solver output for the go-crm faithful policy's
// CapableWritesOwn counterexample: six users, a teamless Manager (User$5),
// and Record$0 owned by that Manager.
func TestRenderSolutionText(t *testing.T) {
	raw, err := os.ReadFile("testdata/policy-solution.txt")
	if err != nil {
		t.Fatal(err)
	}
	got := renderSolutionText(string(raw))
	for _, want := range []string{
		"counterexample: ",
		"User$5{role=Manager$0, team=(none)}",
		"Record$0{owner=User$5}",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered %q missing %q", got, want)
		}
	}
}

func TestRenderSolutionTextEmpty(t *testing.T) {
	if got := renderSolutionText("no relations here"); got != "" {
		t.Errorf("want empty render, got %q", got)
	}
}

func TestAlloyJarPathOverride(t *testing.T) {
	t.Setenv("ALLOY_TOOLS_JAR", "/tmp/custom.jar")
	if alloyJarPath() != "/tmp/custom.jar" {
		t.Error("env override ignored")
	}
	t.Setenv("ALLOY_TOOLS_JAR", "")
	if !strings.Contains(alloyJarPath(), "alloy-dist-"+alloyVersion) {
		t.Errorf("default path = %q", alloyJarPath())
	}
}
