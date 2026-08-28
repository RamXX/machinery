package gates

import (
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

const branchOrderMachine = `{"id":"m","initial":"Deciding","states":{
 "Deciding":{"always":[
   {"target":"A","guard":"guardComplete"},
   {"target":"B","guard":"guardBlocked"},
   {"target":"C"}]},
 "A":{"type":"final"},"B":{"type":"final"},"C":{"type":"final"}}}`

const branchOrderMatrix = "| name | kind | signature | contract | maps to | test | fixture |\n" +
	"|---|---|---|---|---|---|---|\n" +
	"| `guardComplete` | guard | sig | c | inv `cascade-proven` | unit | f |\n" +
	"| `guardBlocked` | guard | sig | c | inv `cascade-proven` | unit | f |\n"

func TestBranchOrderOverlapWarns(t *testing.T) {
	m, err := ir.LoadMachineJSONStr("t", branchOrderMachine)
	if err != nil {
		t.Fatal(err)
	}
	g := NewGate("t")
	CheckBranchOrder(g, m, branchOrderMatrix, "m.machine.json")
	if len(g.Warns) != 1 || !strings.Contains(g.Warns[0], "cascade-proven") || !strings.Contains(g.Warns[0], "_branch_order") {
		t.Fatalf("warns: %v", g.Warns)
	}
}

func TestBranchOrderNoteSilences(t *testing.T) {
	src := strings.Replace(branchOrderMachine, `"Deciding":{"always"`,
		`"Deciding":{"_branch_order":"complete is checked first because a fully blocked cascade is complete too, and certifying it would launder the block","always"`, 1)
	m, err := ir.LoadMachineJSONStr("t", src)
	if err != nil {
		t.Fatal(err)
	}
	g := NewGate("t")
	CheckBranchOrder(g, m, branchOrderMatrix, "m.machine.json")
	if len(g.Warns) != 0 {
		t.Fatalf("annotated state still warned: %v", g.Warns)
	}
}

func TestBranchOrderAmbientInvariantExcluded(t *testing.T) {
	mx := branchOrderMatrix +
		"| `guardX` | guard | sig | c | inv `cascade-proven` | unit | f |\n"
	m, err := ir.LoadMachineJSONStr("t", branchOrderMachine)
	if err != nil {
		t.Fatal(err)
	}
	g := NewGate("t")
	CheckBranchOrder(g, m, mx, "m.machine.json")
	if len(g.Warns) != 0 {
		t.Fatalf("ambient invariant (3 citing guards) still counted: %v", g.Warns)
	}
}

func TestBranchOrderDisjointGuardsSilent(t *testing.T) {
	mx := "| name | kind | signature | contract | maps to | test | fixture |\n" +
		"|---|---|---|---|---|---|---|\n" +
		"| `guardComplete` | guard | sig | c | inv `cascade-proven` | unit | f |\n" +
		"| `guardBlocked` | guard | sig | c | inv `hold-disclosed` | unit | f |\n"
	m, err := ir.LoadMachineJSONStr("t", branchOrderMachine)
	if err != nil {
		t.Fatal(err)
	}
	g := NewGate("t")
	CheckBranchOrder(g, m, mx, "m.machine.json")
	if len(g.Warns) != 0 {
		t.Fatalf("disjoint guards warned: %v", g.Warns)
	}
}
