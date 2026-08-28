package lint

import (
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

func counterMachine(t *testing.T, src string) *ir.Value {
	t.Helper()
	m, err := ir.LoadMachineJSONStr("test", src)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

const noResetMachine = `{"id":"m","initial":"Working","_delays":{"B":"1000 ms - backoff"},
"context":{"workRetries":0},
"states":{
 "Working":{"invoke":{"src":"work","onDone":{"target":"Done"},"onError":{"target":"workRetry"}}},
 "workRetry":{"always":[{"target":"Failed","guard":"guardWorkRetriesExhausted"}],
   "after":{"B":{"target":"Working","actions":"incrementWorkRetries"}}},
 "Done":{"type":"final"},
 "Failed":{"type":"final"}}}`

func TestCountersWarnOnMissingReset(t *testing.T) {
	m := counterMachine(t, noResetMachine)
	errs, warns := LintCounters(m, "m.machine.json")
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "workRetries") || !strings.Contains(warns[0], "no reset action") {
		t.Fatalf("warns: %v", warns)
	}
}

func TestCountersWaiverSilencesWithProof(t *testing.T) {
	src := strings.Replace(noResetMachine, `"context"`, `"_counters":{"workRetries":"one budget per instance: the leg exhausts into the final Failed and a re-run is a new row"},"context"`, 1)
	m := counterMachine(t, src)
	errs, warns := LintCounters(m, "m.machine.json")
	if len(errs) != 0 || len(warns) != 0 {
		t.Fatalf("waived machine flagged: errs %v warns %v", errs, warns)
	}
}

func TestCountersWaiverWithoutProofErrors(t *testing.T) {
	src := strings.Replace(noResetMachine, `"context"`, `"_counters":{"workRetries":"  "},"context"`, 1)
	m := counterMachine(t, src)
	errs, _ := LintCounters(m, "m.machine.json")
	if len(errs) != 1 || !strings.Contains(errs[0], "no proof") {
		t.Fatalf("errs: %v", errs)
	}
}

func TestCountersWaiverUnknownCounterErrors(t *testing.T) {
	src := strings.Replace(noResetMachine, `"context"`, `"_counters":{"ghostRetries":"whatever"},"context"`, 1)
	m := counterMachine(t, src)
	errs, _ := LintCounters(m, "m.machine.json")
	if len(errs) != 1 || !strings.Contains(errs[0], "ghostRetries") {
		t.Fatalf("errs: %v", errs)
	}
}

func TestCountersResetSilences(t *testing.T) {
	src := strings.Replace(noResetMachine,
		`"invoke":{"src":"work","onDone":{"target":"Done"},"onError":{"target":"workRetry"}}`,
		`"on":{"go":{"target":"Running","actions":"clearWorkRetries"}}`, 1)
	src = strings.Replace(src, `"Done":{"type":"final"},`, `"Done":{"type":"final"},"Running":{"invoke":{"src":"work","onDone":{"target":"Done"},"onError":{"target":"workRetry"}}},`, 1)
	m := counterMachine(t, src)
	errs, warns := LintCounters(m, "m.machine.json")
	if len(errs) != 0 || len(warns) != 0 {
		t.Fatalf("reset machine flagged: errs %v warns %v", errs, warns)
	}
}

func TestCountersStaleWaiverWarns(t *testing.T) {
	src := strings.Replace(noResetMachine,
		`"after":{"B":{"target":"Working","actions":"incrementWorkRetries"}}`,
		`"after":{"B":{"target":"Working","actions":["incrementWorkRetries","clearWorkRetries"]}}`, 1)
	src = strings.Replace(src, `"context"`, `"_counters":{"workRetries":"one budget per instance: stale claim"},"context"`, 1)
	m := counterMachine(t, src)
	_, warns := LintCounters(m, "m.machine.json")
	if len(warns) != 1 || !strings.Contains(warns[0], "stale") {
		t.Fatalf("warns: %v", warns)
	}
}
