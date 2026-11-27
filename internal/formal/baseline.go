package formal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/v2/internal/project"
)

// Baseline is baseline.json (section 16.6).
type Baseline struct {
	Version int                      `json:"version"`
	Specs   map[string]*SpecBaseline `json:"specs"`
	Removed []Acceptance             `json:"removed,omitempty"`
}

// SpecBaseline records what a spec promised when the baseline was taken.
type SpecBaseline struct {
	Properties map[string]string      `json:"properties"`
	Fairness   string                 `json:"fairness"`
	Runs       map[string]RunBaseline `json:"runs"`
	Accepted   []Acceptance           `json:"accepted,omitempty"`
}

// RunBaseline records a run's expectation and bounds.
type RunBaseline struct {
	Expect    string         `json:"expect"`
	Constants map[string]any `json:"constants"`
}

// Acceptance records a deliberate weakening.
type Acceptance struct {
	At     string `json:"at"`
	Change string `json:"change"`
	Reason string `json:"reason"`
}

// BaselinePath is the project-relative path of the baseline file.
func BaselinePath(p *project.Project) string { return Dir(p) + "/baseline.json" }

// LoadBaseline reads the baseline; a missing file is an empty baseline.
func LoadBaseline(p *project.Project) (*Baseline, error) {
	return ParseBaseline(p.Read(BaselinePath(p)))
}

// ParseBaseline parses baseline bytes; a read error is an empty baseline.
func ParseBaseline(data []byte, err error) (*Baseline, error) {
	b := &Baseline{Version: 1, Specs: map[string]*SpecBaseline{}}
	if err != nil {
		return b, nil
	}
	if err := json.Unmarshal(data, b); err != nil {
		return nil, fmt.Errorf("baseline.json: %v", err)
	}
	if b.Specs == nil {
		b.Specs = map[string]*SpecBaseline{}
	}
	return b, nil
}

var wsRe = regexp.MustCompile(`\s+`)

func hashText(s string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(wsRe.ReplaceAllString(s, " "))))
	return hex.EncodeToString(sum[:])
}

var tlaDefRe = regexp.MustCompile(`(?m)^([A-Za-z_][A-Za-z0-9_]*)(\([^)]*\))?\s*==`)

// tlaDefinition returns the text of a top-level definition, or "" when absent.
func tlaDefinition(text, name string) string {
	locs := tlaDefRe.FindAllStringSubmatchIndex(text, -1)
	for i, loc := range locs {
		if text[loc[2]:loc[3]] != name {
			continue
		}
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		body := text[loc[0]:end]
		if j := strings.Index(body, "\n===="); j >= 0 {
			body = body[:j]
		}
		return body
	}
	return ""
}

// alloyBlock returns the text of an assert or pred block, or "" when absent.
func alloyBlock(text, name string) string {
	re := regexp.MustCompile(`(?m)^\s*(assert|pred)\s+` + regexp.QuoteMeta(name) + `\b`)
	loc := re.FindStringIndex(text)
	if loc == nil {
		return ""
	}
	open := strings.Index(text[loc[0]:], "{")
	if open < 0 {
		return ""
	}
	depth := 0
	for i := loc[0] + open; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[loc[0] : i+1]
			}
		}
	}
	return ""
}

// definition finds a property's text in the property file, then the skeleton.
func (s *Spec) definition(props, skeleton, name string) string {
	for _, text := range []string{props, skeleton} {
		var def string
		if s.Tool == "alloy" {
			def = alloyBlock(text, name)
		} else {
			def = tlaDefinition(text, name)
		}
		if def != "" {
			return def
		}
	}
	return ""
}

func normConstants(c map[string]any) map[string]any {
	res := map[string]any{}
	for k, v := range c {
		switch x := v.(type) {
		case int:
			res[k] = float64(x)
		case int64:
			res[k] = float64(x)
		default:
			res[k] = v
		}
	}
	return res
}

// Current computes what a spec promises now. missing lists property names
// that are not defined in its files.
func (s *Spec) Current(p *project.Project) (cur *SpecBaseline, missing []string) {
	propsB, _ := p.Read(s.PropsPath())
	skelB, _ := p.Read(s.SkeletonPath())
	props, skel := string(propsB), string(skelB)
	cur = &SpecBaseline{Properties: map[string]string{}, Runs: map[string]RunBaseline{}}
	for _, name := range s.PropertyNames() {
		def := s.definition(props, skel, name)
		if def == "" {
			missing = append(missing, name)
			continue
		}
		cur.Properties[name] = hashText(def)
	}
	if s.Tool == "tla" {
		if def := s.definition(props, skel, "Fairness"); def != "" {
			cur.Fairness = hashText(def)
		}
	}
	for _, r := range s.Runs {
		cur.Runs[r.Name] = RunBaseline{Expect: r.Expect, Constants: normConstants(r.Constants)}
	}
	return cur, missing
}

// Diff compares a recorded baseline with the current state. additions need
// recording; weakenings need a deliberate acceptance.
func Diff(rec, cur *SpecBaseline) (additions, weakenings []string) {
	for name, h := range cur.Properties {
		old, ok := rec.Properties[name]
		switch {
		case !ok:
			additions = append(additions, "property "+name)
		case old != h:
			weakenings = append(weakenings, "property "+name+" changed")
		}
	}
	for name := range rec.Properties {
		if _, ok := cur.Properties[name]; !ok {
			weakenings = append(weakenings, "property "+name+" removed")
		}
	}
	if rec.Fairness != cur.Fairness {
		weakenings = append(weakenings, "fairness changed")
	}
	for name, cr := range cur.Runs {
		rr, ok := rec.Runs[name]
		if !ok {
			additions = append(additions, "run "+name)
			continue
		}
		switch {
		case rr.Expect == "pass" && cr.Expect == "finding":
			weakenings = append(weakenings, fmt.Sprintf("run %s: expect changed from pass to finding", name))
		case rr.Expect != cr.Expect:
			additions = append(additions, fmt.Sprintf("run %s: expect now %s", name, cr.Expect))
		}
		keys := map[string]bool{}
		for k := range rr.Constants {
			keys[k] = true
		}
		for k := range cr.Constants {
			keys[k] = true
		}
		for k := range keys {
			ov, ook := rr.Constants[k]
			nv, nok := cr.Constants[k]
			of, oIsNum := ov.(float64)
			nf, nIsNum := nv.(float64)
			switch {
			case ook && nok && oIsNum && nIsNum:
				if nf < of {
					weakenings = append(weakenings, fmt.Sprintf("run %s: constant %s lowered from %v to %v", name, k, ov, nv))
				} else if nf > of {
					additions = append(additions, fmt.Sprintf("run %s: constant %s raised to %v", name, k, nv))
				}
			case !ook || !nok || fmt.Sprint(ov) != fmt.Sprint(nv):
				weakenings = append(weakenings, fmt.Sprintf("run %s: constant %s changed", name, k))
			}
		}
	}
	for name := range rec.Runs {
		if _, ok := cur.Runs[name]; !ok {
			weakenings = append(weakenings, fmt.Sprintf("run %s: removed", name))
		}
	}
	sort.Strings(additions)
	sort.Strings(weakenings)
	return
}

// Record updates the baseline file from the tree. Without accept it refuses
// weakenings and returns them per spec; nothing is written then.
func Record(p *project.Project, specs []*Spec, accept bool, reason, at string) (refused map[string][]string, recorded int, err error) {
	b, err := LoadBaseline(p)
	if err != nil {
		return nil, 0, err
	}
	refused = map[string][]string{}
	live := map[string]bool{}
	next := map[string]*SpecBaseline{}
	for _, s := range specs {
		live[s.Spec] = true
		cur, _ := s.Current(p)
		rec, ok := b.Specs[s.Spec]
		if !ok {
			next[s.Spec] = cur
			recorded++
			continue
		}
		adds, weak := Diff(rec, cur)
		if len(weak) > 0 && !accept {
			refused[s.Spec] = weak
			continue
		}
		cur.Accepted = rec.Accepted
		for _, w := range weak {
			cur.Accepted = append(cur.Accepted, Acceptance{At: at, Change: w, Reason: reason})
		}
		if len(adds)+len(weak) > 0 {
			recorded++
		}
		next[s.Spec] = cur
	}
	for name, rec := range b.Specs {
		if live[name] {
			continue
		}
		if !accept {
			refused[name] = []string{"spec removed"}
			next[name] = rec
			continue
		}
		b.Removed = append(b.Removed, Acceptance{At: at, Change: "spec " + name + " removed", Reason: reason})
		recorded++
	}
	if len(refused) > 0 {
		return refused, 0, nil
	}
	b.Specs = next
	return nil, recorded, writeJSON(p, BaselinePath(p), b)
}
