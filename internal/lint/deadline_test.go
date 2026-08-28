package lint

import (
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

const deadlineMachine = `{"id":"m","initial":"Received",
"_delays":{"WORK_REMAINING":"derived remaining of the stamped absolute deadline","B":"1000 ms - backoff"},
"context":{},
"states":{
 "Received":{"on":{"go":{"target":"Working"}},"after":{"WORK_REMAINING":{"target":"Failed"}}},
 "Working":{"invoke":{"src":"work","onDone":{"target":"Reviewing"},"onError":{"target":"workRetry"}},"after":{"WORK_REMAINING":{"target":"Failed"}}},
 "workRetry":{"always":[{"target":"Failed","guard":"guardRetriesExhausted"}],"after":{"B":{"target":"Working","actions":"incrementRetries"}}},
 "Reviewing":{"on":{"approve":{"target":"Done"}}},
 "routing":{"always":[{"target":"Done"}]},
 "Done":{"type":"final"},
 "Failed":{"type":"final"}}}`

func TestDerivedDeadlineHole(t *testing.T) {
	// Reviewing dwells (an on handler), sits between two consumers'
	// reachability... it is downstream-only here, so make it a hole by
	// letting it route back into a consumer.
	src := strings.Replace(deadlineMachine, `"Reviewing":{"on":{"approve":{"target":"Done"}}}`,
		`"Reviewing":{"on":{"approve":{"target":"Done"},"rework":{"target":"Working"}}}`, 1)
	m, err := ir.LoadMachineJSONStr("t", src)
	if err != nil {
		t.Fatal(err)
	}
	warns := LintDerivedDeadlines(m, "m.machine.json")
	found := false
	for _, w := range warns {
		if strings.Contains(w, "Reviewing") && strings.Contains(w, "WORK_REMAINING") {
			found = true
		}
	}
	if !found {
		t.Fatalf("hole not flagged: %v", warns)
	}
}

func TestDerivedDeadlineExemptions(t *testing.T) {
	// The retry sibling (owns a backoff after) and the pure-always router
	// must not flag; neither must finals or downstream-only states.
	m, err := ir.LoadMachineJSONStr("t", deadlineMachine)
	if err != nil {
		t.Fatal(err)
	}
	warns := LintDerivedDeadlines(m, "m.machine.json")
	for _, w := range warns {
		for _, bad := range []string{"workRetry", "routing", "Done", "Failed", "Reviewing"} {
			if strings.Contains(w, "state "+bad+" ") {
				t.Fatalf("exempt state flagged: %v", warns)
			}
		}
	}
}

func TestNonDeadlineDelaysIgnored(t *testing.T) {
	src := strings.ReplaceAll(deadlineMachine, "WORK_REMAINING", "SOME_TIMEOUT")
	src = strings.Replace(src, `"derived remaining of the stamped absolute deadline"`, `"9000 ms - plain window"`, 1)
	m, err := ir.LoadMachineJSONStr("t", src)
	if err != nil {
		t.Fatal(err)
	}
	if warns := LintDerivedDeadlines(m, "m.machine.json"); len(warns) != 0 {
		t.Fatalf("plain delay flagged: %v", warns)
	}
}
