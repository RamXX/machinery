package checker

import (
	"encoding/json"
	"fmt"
	"os"
)

// Finding is one issue a checker surfaced. Machinery renders blocking findings
// as ERRORs on a fail verdict, advisory as warns, info as notes.
type Finding struct {
	Severity string `json:"severity"`
	Code     string `json:"code,omitempty"`
	Element  string `json:"element,omitempty"`
	Message  string `json:"message"`
	Locator  string `json:"locator,omitempty"`
}

// CoverageRow records one design element the checker actually decided.
type CoverageRow struct {
	Element string `json:"element"`
	Verdict string `json:"verdict"`
	Detail  string `json:"detail,omitempty"`
}

// Evidence is what a checker (or its adapter) writes back. Machinery's pure phase
// reads only this file and never the checker's native output; attestation and
// trace_ref are carried opaquely for the external verify phase.
type Evidence struct {
	EvidenceSchema string `json:"evidence_schema"`
	Checker        struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	} `json:"checker"`
	InputHash string        `json:"input_hash"`
	Verdict   string        `json:"verdict"`
	Coverage  []CoverageRow `json:"coverage"`
	Findings  []Finding     `json:"findings,omitempty"`
	// Attestation and TraceRef are opaque to the pure phase; kept as raw so a
	// checker that "emits what it emits" carries its own provenance untouched.
	Attestation json.RawMessage `json:"attestation,omitempty"`
	TraceRef    string          `json:"trace_ref,omitempty"`
}

// LoadEvidence reads and shallow-validates an evidence file. Absence is an error
// (the caller turns it into a gate ERROR); a wrong schema or verdict token is an
// error too, so a malformed verdict cannot masquerade as a pass.
func LoadEvidence(path string) (*Evidence, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var e Evidence
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	switch e.Verdict {
	case "pass", "fail":
	default:
		return nil, fmt.Errorf("%s: verdict must be \"pass\" or \"fail\", got %q", path, e.Verdict)
	}
	return &e, nil
}
