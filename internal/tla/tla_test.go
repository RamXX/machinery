package tla

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/version"
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

func TestTLARejectsRetryStateWithInvokeHandlers(t *testing.T) {
	// Adapted from the reviewer fixture exp-a-retry-invoke: a retry-shaped
	// state carrying an invoke whose onDone/onError routes rung 3 would
	// silently drop (the sibling of the on: handler case).
	src := `{"id":"retryinvoke","initial":"Idle","states":{
	  "Idle":{"on":{"start":{"target":"working"}}},
	  "working":{"invoke":{"src":"doWork","onDone":{"target":"Done"},"onError":{"target":"retryWait"}},"after":{"timeout":{"target":"retryWait"}}},
	  "retryWait":{"always":[{"target":"Failed","guard":"retriesExhausted"}],"after":{"backoff":{"target":"working"}},"invoke":{"src":"probeHealth","onDone":{"target":"Done"},"onError":{"target":"Failed"}}},
	  "Done":{"type":"final"},
	  "Failed":{"type":"final"}}}`
	_, _, _, err := Generate(writeSrc(t, src))
	if err == nil || !strings.Contains(err.Error(), "retry state") || !strings.Contains(err.Error(), "invoke") {
		t.Fatalf("expected retry-state invoke handlers to be rejected, got %v", err)
	}
}

func TestTLARejectsRetryStateWithStateLevelOnDone(t *testing.T) {
	src := `{"id":"widget","initial":"Idle","states":{
	  "Idle":{"on":{"start":{"target":"working"}}},
	  "working":{"invoke":{"src":"doWork","onDone":{"target":"Done"},"onError":{"target":"retryWait"}},"after":{"timeout":{"target":"retryWait"}}},
	  "retryWait":{"always":[{"target":"Failed","guard":"retriesExhausted"}],"after":{"backoff":{"target":"working"}},"onDone":{"target":"Done"}},
	  "Done":{"type":"final"},
	  "Failed":{"type":"final"}}}`
	_, _, _, err := Generate(writeSrc(t, src))
	if err == nil || !strings.Contains(err.Error(), "retry state") || !strings.Contains(err.Error(), "onDone") {
		t.Fatalf("expected retry-state state-level onDone to be rejected, got %v", err)
	}
}

func TestTLAAssumptionsDocumentRetryCounterSemantics(t *testing.T) {
	// The retry-counter model assumptions (counters reset on any domain
	// transition; the guarded always list replaced by the concrete
	// rc >= MaxRetries test) must be stated in the generated header.
	src := `{"id":"widget","initial":"Draft","states":{
	  "Draft":{"on":{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}]}},
	  "Published":{"type":"final"},
	  "persisting":{"invoke":{"src":"s","onDone":{"target":"Published"},"onError":[{"target":"persistRetry","guard":"isLocked"},{"target":"Draft"}]},"after":{"t":{"target":"Draft"}}},
	  "persistRetry":{"always":[{"target":"Draft","guard":"retriesExhausted"}],"after":{"backoff":{"target":"persisting"}}}}}`
	_, tla, _, err := Generate(writeSrc(t, src))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Retry counters",
		"reset to 0",
		"rc >= MaxRetries",
		"rc < MaxRetries",
	} {
		if !strings.Contains(tla, want) {
			t.Errorf("assumptions block missing %q", want)
		}
	}
	// a machine with no retry states must not carry the retry assumptions
	_, plain, _, err := Generate(writeSrc(t, minimalSrc))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain, "Retry counters") {
		t.Error("retry-counter assumptions emitted for a machine with no retry states")
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

// P-F10: files WRITTEN for committal (design/formal) carry a version stamp in
// each format's comment syntax; the in-memory Generate output stays unstamped
// because the pack generator embeds it in hash-covered pack files (a stamp
// there would churn the pack content hash every release).
func TestRunWrittenStampsGeneratorVersion(t *testing.T) {
	src := writeSrc(t, minimalSrc)
	outdir := t.TempDir()
	names, err := RunWritten(src, outdir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("names = %v", names)
	}
	for _, n := range names {
		data, rerr := os.ReadFile(filepath.Join(outdir, n))
		if rerr != nil {
			t.Fatal(rerr)
		}
		body := string(data)
		if !strings.Contains(body, version.TLAStamp()) {
			t.Errorf("%s carries no version stamp:\n%s", n, body)
		}
		if got := strings.Count(body, "machinery-version:"); got != 1 {
			t.Errorf("%s carries %d stamp lines, want exactly 1", n, got)
		}
		if strings.HasSuffix(n, ".tla") && !strings.HasPrefix(body, "---- MODULE ") {
			t.Errorf("%s no longer opens with the MODULE line:\n%s", n, body)
		}
	}
}

func TestGenerateOutputIsUnstamped(t *testing.T) {
	_, tlaOut, cfgOut, err := Generate(writeSrc(t, minimalSrc))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(tlaOut, "machinery-version:") || strings.Contains(cfgOut, "machinery-version:") {
		t.Error("Generate output must stay unstamped: packs embed it under the content hash")
	}
}
