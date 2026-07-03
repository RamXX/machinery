package tla

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSrc(t *testing.T, src string) string {
	t.Helper()
	d := t.TempDir()
	p := filepath.Join(d, "Widget.machine.json")
	if err := os.WriteFile(p, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
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
	src := `{"id":"widget","initial":"Draft","states":{"Draft":{"on":{"x":{"target":"locked"},"d":{"target":"Done"}}},"Done":{"type":"final"},"locked":{"always":[{"target":"Draft","guard":"retriesExhausted"}],"after":{"backoff":[{"target":"Draft","guard":"phaseA"},{"target":"persisting","guard":"phaseB"}]}},"persisting":{"invoke":{"src":"s","onDone":{"target":"Done"},"onError":{"target":"Draft"}},"after":{"t":{"target":"Draft"}}}}}`
	_, tla, _, err := Generate(writeSrc(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tla, `st' \in {"Draft", "persisting"}`) {
		t.Fatalf("multi-target resume not modeled:\n%s", tla)
	}
}

func TestTLARejectsRetryStateWithOnHandlers(t *testing.T) {
	// A retry-shaped state's on: handlers used to be silently dropped from the
	// model (and the state reclassified out of the liveness property).
	src := `{"id":"widget","initial":"Draft","states":{
	  "Draft":{"invoke":{"src":"save","onDone":{"target":"Done"},"onError":{"target":"retrying"}},"after":{"t":{"target":"retrying"}}},
	  "retrying":{"always":[{"guard":"retriesLeft","target":"Draft"}],"after":{"backoff":{"target":"Failed"}},"on":{"CANCEL":{"target":"Failed"}},"_exhaustive":"retriesLeft is total"},
	  "Done":{"type":"final"},
	  "Failed":{"type":"final"}}}`
	_, _, _, err := Generate(writeSrc(t, src))
	if err == nil || !strings.Contains(err.Error(), "retry state") {
		t.Fatalf("expected retry-state on: handlers to be rejected, got %v", err)
	}
}

func TestTLARejectsMissingInitial(t *testing.T) {
	src := `{"id":"widget","states":{"Draft":{"on":{"x":{"target":"Done"}}},"Done":{"type":"final"}}}`
	_, _, _, err := Generate(writeSrc(t, src))
	if err == nil || !strings.Contains(err.Error(), "no initial state") {
		t.Fatalf("expected missing-initial rejection, got %v", err)
	}
}

func TestTLARejectsEmptyStates(t *testing.T) {
	src := `{"id":"widget","initial":"Draft","states":{}}`
	_, _, _, err := Generate(writeSrc(t, src))
	if err == nil || !strings.Contains(err.Error(), "no states") {
		t.Fatalf("expected empty-states rejection, got %v", err)
	}
}

func TestTLAMaxRetriesAnnotation(t *testing.T) {
	src := strings.Replace(minimalSrc, `{"id":"widget",`, `{"id":"widget","_max_retries":7,`, 1)
	_, _, cfg, err := Generate(writeSrc(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cfg, "MaxRetries = 7") {
		t.Errorf("cfg does not honor _max_retries: %s", cfg)
	}
	bad := strings.Replace(minimalSrc, `{"id":"widget",`, `{"id":"widget","_max_retries":0,`, 1)
	if _, _, _, err := Generate(writeSrc(t, bad)); err == nil {
		t.Error("nonpositive _max_retries accepted")
	}
}
