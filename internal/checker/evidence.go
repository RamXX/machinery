package checker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	InputHash      string        `json:"input_hash"`
	RuntimeClosure string        `json:"runtime_closure"`
	Verdict        string        `json:"verdict"`
	Coverage       []CoverageRow `json:"coverage"`
	Findings       []Finding     `json:"findings,omitempty"`
	// Attestation and TraceRef are opaque to the pure phase; kept as raw so a
	// checker that "emits what it emits" carries its own provenance untouched.
	Attestation    json.RawMessage `json:"attestation,omitempty"`
	InputSignature json.RawMessage `json:"input_signature,omitempty"`
	TraceRef       string          `json:"trace_ref,omitempty"`
	Generated      json.RawMessage `json:"generated,omitempty"`
}

// LoadEvidence reads and fully validates the implemented evidence contract.
// Absence, schema drift, ambiguous JSON, inconsistent verdicts, unsafe proof
// references, and unsupported reserved fields all fail closed.
func LoadEvidence(path string) (*Evidence, error) {
	data, err := readCheckerStructuredFile(path, "evidence")
	if err != nil {
		return nil, err
	}
	return parseEvidence(path, data, func(traceRef string) error {
		tracePath, err := ConfinedPath(filepath.Dir(path), traceRef)
		if err != nil {
			return fmt.Errorf("trace_ref is unsafe: %w", err)
		}
		traceInfo, err := os.Lstat(tracePath)
		if err != nil {
			return fmt.Errorf("trace_ref %q is unreadable: %w", traceRef, err)
		}
		if traceInfo.Mode()&os.ModeSymlink != 0 || !traceInfo.Mode().IsRegular() {
			return fmt.Errorf("trace_ref %q must name a regular, non-symlink file", traceRef)
		}
		return nil
	})
}

// LoadEvidenceConfined reads evidence and any trace reference through one
// design root, closing the validation/read race exposed by ConfinedPath plus a
// later ambient LoadEvidence call.
func LoadEvidenceConfined(design, rel string) (evidence *Evidence, retErr error) {
	evidence, _, retErr = LoadEvidenceConfinedBytes(design, rel)
	return evidence, retErr
}

// LoadEvidenceConfinedBytes returns the exact bytes validated as evidence so a
// caller can pass a trusted snapshot to an external adapter without reopening
// the ambient path.
func LoadEvidenceConfinedBytes(design, rel string) (evidence *Evidence, data []byte, retErr error) {
	root, err := openDesignRoot(design)
	if err != nil {
		return nil, nil, err
	}
	defer closeRoot(&retErr, root)
	clean, err := root.confinedRel(rel)
	if err != nil {
		return nil, nil, err
	}
	data, err = root.readRegularBounded(clean, "evidence", checkerStructuredFileMaxBytes)
	if err != nil {
		return nil, nil, err
	}
	evidence, err = parseEvidence(root.display(clean), data, func(traceRef string) error {
		traceRel, err := root.confinedRel(filepath.ToSlash(filepath.Join(filepath.Dir(clean), filepath.FromSlash(traceRef))))
		if err != nil {
			return fmt.Errorf("trace_ref is unsafe: %w", err)
		}
		exists, err := root.lstatRegular(traceRel, "trace_ref", false)
		if err != nil {
			return fmt.Errorf("trace_ref %q is unreadable: %w", traceRef, err)
		}
		if !exists {
			return fmt.Errorf("trace_ref %q is unreadable: %w", traceRef, os.ErrNotExist)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return evidence, data, nil
}

func parseEvidence(path string, data []byte, validateTrace func(string) error) (*Evidence, error) {
	var e Evidence
	if err := decodeStrictJSON(data, &e); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, arrayField := range []string{"coverage", "findings"} {
		if raw, present := fields[arrayField]; present {
			trimmed := bytes.TrimSpace(raw)
			if len(trimmed) == 0 || trimmed[0] != '[' {
				return nil, fmt.Errorf("%s: %s must be a JSON array", path, arrayField)
			}
		}
	}
	if e.EvidenceSchema != SchemaVersion {
		return nil, fmt.Errorf("%s: evidence_schema must be %q, got %q", path, SchemaVersion, e.EvidenceSchema)
	}
	if strings.TrimSpace(e.Checker.ID) == "" {
		return nil, fmt.Errorf("%s: checker.id must be non-empty", path)
	}
	if strings.TrimSpace(e.Checker.Version) == "" {
		return nil, fmt.Errorf("%s: checker.version must be non-empty", path)
	}
	if !validSHA256(e.InputHash) {
		return nil, fmt.Errorf("%s: input_hash must be a lowercase sha256:<64 hex> digest", path)
	}
	if !validSHA256(e.RuntimeClosure) {
		return nil, fmt.Errorf("%s: runtime_closure must be a lowercase sha256:<64 hex> OCI closure digest", path)
	}
	switch e.Verdict {
	case "pass", "fail":
	default:
		return nil, fmt.Errorf("%s: verdict must be \"pass\" or \"fail\", got %q", path, e.Verdict)
	}
	if e.Coverage == nil {
		return nil, fmt.Errorf("%s: coverage must be an array, not null or absent", path)
	}
	seenCoverage := map[string]bool{}
	failedElement := ""
	for i, row := range e.Coverage {
		if strings.TrimSpace(row.Element) == "" {
			return nil, fmt.Errorf("%s: coverage[%d].element must be non-empty", path, i)
		}
		switch row.Verdict {
		case "pass", "fail":
		default:
			return nil, fmt.Errorf("%s: coverage[%d].verdict must be \"pass\" or \"fail\", got %q", path, i, row.Verdict)
		}
		if seenCoverage[row.Element] {
			return nil, fmt.Errorf("%s: coverage element %q appears more than once", path, row.Element)
		}
		seenCoverage[row.Element] = true
		if row.Verdict == "fail" && failedElement == "" {
			failedElement = row.Element
		}
	}
	hasBlockingFinding := false
	for i, finding := range e.Findings {
		switch finding.Severity {
		case "blocking", "advisory", "info":
		default:
			return nil, fmt.Errorf("%s: findings[%d].severity must be blocking, advisory, or info, got %q", path, i, finding.Severity)
		}
		if strings.TrimSpace(finding.Message) == "" {
			return nil, fmt.Errorf("%s: findings[%d].message must be non-empty", path, i)
		}
		if finding.Severity == "blocking" {
			hasBlockingFinding = true
		}
	}
	if e.Verdict == "pass" && failedElement != "" {
		return nil, fmt.Errorf("%s: pass verdict conflicts with coverage verdict: fail [%s]", path, failedElement)
	}
	if e.Verdict == "pass" && hasBlockingFinding {
		return nil, fmt.Errorf("%s: pass verdict conflicts with a blocking finding", path)
	}
	if e.Verdict == "fail" && failedElement == "" && !hasBlockingFinding {
		return nil, fmt.Errorf("%s: fail verdict requires a failed coverage row or blocking finding", path)
	}
	if len(e.InputSignature) > 0 {
		return nil, fmt.Errorf("%s: input_signature is reserved and unsupported; use opaque attestation plus the registry verify command", path)
	}
	if e.TraceRef != "" {
		if err := validatePortableRelativePath(e.TraceRef); err != nil {
			return nil, fmt.Errorf("%s: trace_ref is not portable: %w", path, err)
		}
		if !strings.HasPrefix(e.TraceRef, "generated/") {
			return nil, fmt.Errorf("%s: trace_ref must be owned below the evidence file's generated/ directory", path)
		}
		if err := validateTrace(e.TraceRef); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	for _, opaque := range []struct {
		name string
		raw  json.RawMessage
	}{{"attestation", e.Attestation}, {"generated", e.Generated}} {
		name, raw := opaque.name, opaque.raw
		if len(raw) == 0 {
			continue
		}
		if first := bytes.TrimSpace(raw); len(first) == 0 || first[0] != '{' {
			return nil, fmt.Errorf("%s: %s must be a JSON object", path, name)
		}
		var object map[string]json.RawMessage
		if err := decodeStrictJSON(raw, &object); err != nil {
			return nil, fmt.Errorf("%s: %s: %w", path, name, err)
		}
	}
	return &e, nil
}
