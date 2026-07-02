package tla

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ramirosalas/machinery/internal/ir"
)

func writeMachine(t *testing.T, m *ir.Value) string {
	t.Helper()
	d := t.TempDir()
	p := filepath.Join(d, "Widget.machine.json")
	// re-serialize the ordered value to JSON
	b := ir.DumpJSON
	_ = b
	os.WriteFile(p, []byte(m.AsObject().GetString("__src")), 0644)
	return p
}

func writeSrc(t *testing.T, src string) string {
	d := t.TempDir()
	p := filepath.Join(d, "Widget.machine.json")
	os.WriteFile(p, []byte(src), 0644)
	return p
}

const minimalSrc = `{"id":"widget","initial":"Draft","states":{
  "Draft":{"on":{"publish":[{"target":"persisting","guard":"guardCanPublish","actions":"setPending"},{"actions":"recordDenied"}]}},
  "Published":{"type":"final"},
  "persisting":{"invoke":{"src":"saveWidget","onDone":{"target":"Published","actions":"commit"},"onError":{"target":"Draft","actions":"recordError"}},"after":{"persistTimeout":{"target":"Draft","actions":"recordTimeout"}}}}}`

func TestTLAHasAssumptionsBlock(t *testing.T) {
	_, tla, _, err := Generate(writeSrc(t, minimalSrc))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ASSUMPTIONS", "Guards are erased", "Single machine instance"} {
		if !strings.Contains(tla, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestTLARejectsNestedStates(t *testing.T) {
	src := `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"x":{"target":"Wrapper"}}},
		"Wrapper":{"initial":"Inner","states":{"Inner":{"type":"final"}}},
		"Done":{"type":"final"}}}`
	_, _, _, err := Generate(writeSrc(t, src))
	if err == nil || !strings.Contains(err.Error(), "nested states") {
		t.Fatalf("expected nested-states error, got %v", err)
	}
}

func TestTLACarriesExhaustiveNotes(t *testing.T) {
	src := `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"x":{"target":"router"},"done":{"target":"Done"}}},
		"Done":{"type":"final"},
		"router":{"always":[{"target":"Draft","guard":"priorIsDraft"}],"_exhaustive":"prior ranges over {Draft} by construction"}}}`
	_, tla, _, err := Generate(writeSrc(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tla, "prior ranges over {Draft} by construction") {
		t.Fatal("exhaustive note missing")
	}
}

func TestTLAGivesEachRetryStateItsOwnCounter(t *testing.T) {
	src := `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"poke":{"target":"pokeWait"}}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"s","onDone":{"target":"Published"},"onError":[{"target":"persistRetry","guard":"isLocked"},{"target":"Draft"}]},"after":{"t":{"target":"Draft"}}},
		"persistRetry":{"always":[{"target":"Draft","guard":"retriesExhausted"}],"after":{"backoff":{"target":"persisting"}}},
		"pokeWait":{"invoke":{"src":"poker","onDone":{"target":"Published"},"onError":{"target":"pokeRetry"}},"after":{"pokeTimeout":{"target":"pokeRetry"}}},
		"pokeRetry":{"always":[{"target":"Draft","guard":"pokesExhausted"}],"after":{"pokeBackoff":{"target":"pokeWait"}}}}}`
	_, tla, _, err := Generate(writeSrc(t, src))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"rc1", "rc2", "RetryAgain_persistRetry", "RetryAgain_pokeRetry"} {
		if !strings.Contains(tla, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestTLAModelsMultiTargetRetryResume(t *testing.T) {
	src := `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"lock":{"target":"locked"}},
		         "done":{"target":"Done"}},
		"Done":{"type":"final"},
		"persisting":{"invoke":{"src":"s","onDone":{"target":"Published"},"onError":{"target":"Draft"}},"after":{"t":{"target":"Draft"}}},
		"Published":{"type":"final"},
		"locked":{"always":[{"target":"Draft","guard":"retriesExhausted"}],"after":{"backoff":[{"target":"persisting","guard":"phaseA"},{"target":"Draft","guard":"phaseB"}]}}}`
	_, tla, _, err := Generate(writeSrc(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tla, `st' \in {"Draft", "persisting"}`) {
		t.Fatalf("multi-target resume not modeled:\n%s", tla)
	}
}
