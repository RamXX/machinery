// Package receipt lets a Go end-to-end suite emit machinery receipts
// (docs/formats.md section 10). It uses the standard library only.
//
// A test asserts first and emits afterwards:
//
//	if receipt.Wanted("tc:ORDE-2e883d") {
//		// drive the real system, assert ...
//		receipt.Pass("tc:ORDE-2e883d", t.Name(), "Pending -> paying", "stampAttempt")
//	}
//
// When MACHINERY_RECEIPTS is not set every emit is a no-op, so the suite runs
// the same outside machinery run.
package receipt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// Outcomes of a receipt.
const (
	OutcomePass = "pass"
	OutcomeFail = "fail"
	OutcomeSkip = "skip"
)

// Replay values for cx ids (section 16.10).
const (
	Reproduced = "reproduced"
	Refuted    = "refuted"
)

// MaxObserved is the longest observed sentence; longer text is cut.
const MaxObserved = 300

// Environment variables machinery run sets.
const (
	EnvReceipts = "MACHINERY_RECEIPTS"
	EnvOnly     = "MACHINERY_ONLY"
	EnvCommit   = "MACHINERY_COMMIT"
)

// Receipt is one observation made by a test.
type Receipt struct {
	ID       string   // an obligation id
	Outcome  string   // OutcomePass, OutcomeFail or OutcomeSkip
	Test     string   // the name of the test that observed it
	Effects  []string // for tc ids: the effects actually observed
	Observed string   // one short sentence
	Replay   string   // for cx ids: Reproduced or Refuted
}

type wire struct {
	V        int      `json:"v"`
	Type     string   `json:"type"`
	ID       string   `json:"id"`
	Outcome  string   `json:"outcome"`
	Test     string   `json:"test"`
	Effects  []string `json:"effects,omitempty"`
	Observed string   `json:"observed"`
	Replay   string   `json:"replay,omitempty"`
}

// Enabled reports whether receipts are being recorded.
func Enabled() bool { return os.Getenv(EnvReceipts) != "" }

// Emit appends one receipt line to the file named by MACHINERY_RECEIPTS with
// a single write, so parallel tests do not interleave. It does nothing when
// the variable is not set.
func Emit(r Receipt) error {
	path := os.Getenv(EnvReceipts)
	if path == "" {
		return nil
	}
	if r.ID == "" {
		return fmt.Errorf("receipt: id is required")
	}
	switch r.Outcome {
	case OutcomePass, OutcomeFail, OutcomeSkip:
	default:
		return fmt.Errorf("receipt: %s: outcome must be pass, fail or skip, got %q", r.ID, r.Outcome)
	}
	if r.Replay != "" && r.Replay != Reproduced && r.Replay != Refuted {
		return fmt.Errorf("receipt: %s: replay must be reproduced or refuted, got %q", r.ID, r.Replay)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(wire{V: 1, Type: "receipt", ID: r.ID, Outcome: r.Outcome, Test: r.Test,
		Effects: r.Effects, Observed: cut(r.Observed), Replay: r.Replay}); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	_, werr := f.Write(buf.Bytes())
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	return werr
}

// Pass emits a pass receipt with the effects the test observed.
func Pass(id, test, observed string, effects ...string) error {
	return Emit(Receipt{ID: id, Outcome: OutcomePass, Test: test, Effects: effects, Observed: observed})
}

// Fail emits a fail receipt.
func Fail(id, test, observed string) error {
	return Emit(Receipt{ID: id, Outcome: OutcomeFail, Test: test, Observed: observed})
}

// Skip emits a skip receipt; skips never count as evidence.
func Skip(id, test, observed string) error {
	return Emit(Receipt{ID: id, Outcome: OutcomeSkip, Test: test, Observed: observed})
}

// Replay emits the receipt of a counterexample replay.
func Replay(id, test, replay, observed string) error {
	return Emit(Receipt{ID: id, Outcome: OutcomePass, Test: test, Observed: observed, Replay: replay})
}

// Only returns the ids of MACHINERY_ONLY, or nil for a full run.
func Only() []string {
	var ids []string
	for _, id := range strings.Split(os.Getenv(EnvOnly), ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// Wanted reports whether the run asks for id: always true in a full run.
func Wanted(id string) bool {
	only := Only()
	if len(only) == 0 {
		return true
	}
	for _, o := range only {
		if o == id {
			return true
		}
	}
	return false
}

func cut(s string) string {
	if utf8.RuneCountInString(s) <= MaxObserved {
		return s
	}
	r := []rune(s)
	return string(r[:MaxObserved])
}
