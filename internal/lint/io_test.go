package lint

import (
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

const ioDeadlockMachine = `{"id":"m","initial":"Pending",
"_io":{
 "guardEvidenceRecorded":{"reads":["evidenceRef"]},
 "recordEvidence":{"writes":["evidenceRef"]}
},
"context":{"evidenceRef":null},
"states":{
 "Pending":{"on":{"admit":{"target":"Running","guard":"guardEvidenceRecorded","actions":"recordEvidence"}},"_refusal":{"admit":"fixture: refused without recorded evidence"}},
 "Running":{"type":"final"}}}`

func TestIODeadlockWarns(t *testing.T) {
	// The only writer rides the edge the guard itself gates: first instance
	// refuses forever.
	m, err := ir.LoadMachineJSONStr("t", ioDeadlockMachine)
	if err != nil {
		t.Fatal(err)
	}
	errs, warns := LintIO(m, "m.machine.json")
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "guardEvidenceRecorded") || !strings.Contains(warns[0], "deadlock") {
		t.Fatalf("warns: %v", warns)
	}
}

func TestIOUpstreamWriterSilences(t *testing.T) {
	src := strings.Replace(ioDeadlockMachine,
		`"Pending":{"on":{"admit"`,
		`"Pending":{"on":{"record":{"target":"Pending","actions":"recordEvidence"},"admit"`, 1)
	m, err := ir.LoadMachineJSONStr("t", src)
	if err != nil {
		t.Fatal(err)
	}
	_, warns := LintIO(m, "m.machine.json")
	if len(warns) != 0 {
		t.Fatalf("upstream writer still warned: %v", warns)
	}
}

func TestIOCreationCounts(t *testing.T) {
	src := strings.Replace(ioDeadlockMachine, `"context":{"evidenceRef":null}`, `"context":{"evidenceRef":"seeded"}`, 1)
	m, err := ir.LoadMachineJSONStr("t", src)
	if err != nil {
		t.Fatal(err)
	}
	if _, warns := LintIO(m, "m.machine.json"); len(warns) != 0 {
		t.Fatalf("non-null initial still warned: %v", warns)
	}
	src2 := strings.Replace(ioDeadlockMachine, `"recordEvidence":{"writes":["evidenceRef"]}`,
		`"recordEvidence":{"writes":["evidenceRef"]},"create":{"writes":["evidenceRef"]}`, 1)
	m2, err := ir.LoadMachineJSONStr("t", src2)
	if err != nil {
		t.Fatal(err)
	}
	if _, warns := LintIO(m2, "m.machine.json"); len(warns) != 0 {
		t.Fatalf("create.writes still warned: %v", warns)
	}
}

func TestIODeclarationValidation(t *testing.T) {
	src := strings.Replace(ioDeadlockMachine, `"recordEvidence":{"writes":["evidenceRef"]}`,
		`"ghostUnit":{"writes":["evidenceRef"]},"recordEvidence":{"writes":["nope"]}`, 1)
	m, err := ir.LoadMachineJSONStr("t", src)
	if err != nil {
		t.Fatal(err)
	}
	errs, _ := LintIO(m, "m.machine.json")
	var ghost, nofield bool
	for _, e := range errs {
		if strings.Contains(e, "ghostUnit") {
			ghost = true
		}
		if strings.Contains(e, "'nope'") {
			nofield = true
		}
	}
	if !ghost || !nofield {
		t.Fatalf("declaration validation incomplete: %v", errs)
	}
}
