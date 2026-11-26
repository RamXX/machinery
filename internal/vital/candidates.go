package vital

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/project"
)

// The default word lists of section 23.3.
var (
	DefaultNames   = []string{"pay", "charge", "refund", "transfer", "debit", "credit", "sign", "approve", "destroy", "delete", "erase", "grant", "revoke", "seal", "audit", "ledger"}
	DefaultFailure = []string{"lost", "twice", "duplicate", "double", "replay"}
	DefaultEffects = []string{"charge", "debit", "credit", "emit", "send", "publish"}
)

// Candidate is an id that looks as if it belongs in a vital area and is in
// none, with the reason of the first rule that matched it.
type Candidate struct {
	ID     string
	Reason string
}

// Line renders a candidate as machinery vital candidates prints it.
func (c Candidate) Line() string { return c.ID + "  " + c.Reason }

// Summary is the last line of the list.
func Summary(n, dismissed int) string {
	return fmt.Sprintf("vital candidates: %d, dismissed: %d", n, dismissed)
}

func lists(c *project.Config) (names, failure, effects []string) {
	names, failure, effects = DefaultNames, DefaultFailure, DefaultEffects
	if v := c.Vital; v != nil {
		if v.Names != nil {
			names = *v.Names
		}
		if v.Failure != nil {
			failure = *v.Failure
		}
		if v.Effects != nil {
			effects = *v.Effects
		}
	}
	return lower(names), lower(failure), lower(effects)
}

func lower(ws []string) []string {
	res := make([]string, 0, len(ws))
	for _, w := range ws {
		res = append(res, strings.ToLower(strings.TrimSpace(w)))
	}
	return res
}

// Candidates applies the rules of section 23.3 to the design, in their order,
// and returns the candidates that are not dismissed, sorted by id, and how
// many are dismissed.
func Candidates(d *design.Design) (cands []Candidate, dismissed int) {
	names, failure, effects := lists(&d.P.Cfg)
	reasons := map[string]string{}
	add := func(id, reason string) {
		if _, done := reasons[id]; done || !d.Known(id) || d.InVitalArea(id) {
			return
		}
		reasons[id] = reason
	}
	var entities []design.Entity
	for _, m := range d.Models {
		entities = append(entities, m.Entities...)
	}
	for _, e := range entities { // action
		for _, a := range e.Actions {
			if w := match(a.Name, names); w != "" {
				add(design.ActID(e.Name, a.Name), fmt.Sprintf("action %s matches %q", a.Name, w))
			}
		}
	}
	for _, e := range entities { // entity
		if w := match(e.Name, names); w != "" {
			for _, a := range e.Actions {
				add(design.ActID(e.Name, a.Name), fmt.Sprintf("entity %s matches %q", e.Name, w))
			}
		}
	}
	for _, m := range d.Machines { // event
		for _, tr := range m.Table.Transitions {
			if tr.Kind == "on" {
				if w := match(tr.Event, names); w != "" {
					add(tr.ID, fmt.Sprintf("event %s matches %q", tr.Event, w))
				}
			}
		}
	}
	for _, m := range d.Machines { // event entity
		ent := d.LifecycleEntity(m)
		w := match(ent, names)
		if ent == "" || w == "" {
			continue
		}
		for _, tr := range m.Table.Transitions {
			if tr.Kind == "on" {
				add(tr.ID, fmt.Sprintf("entity %s matches %q", ent, w))
			}
		}
	}
	for _, fm := range d.FailureModes { // failure
		if w := match(fm.Failure+" "+fm.Behavior, failure); w != "" {
			add("fm:"+fm.ID, fmt.Sprintf("failure mode text matches %q", w))
		}
	}
	for _, m := range d.Machines { // effect
		for _, tr := range m.Table.Transitions {
			for _, e := range tr.Effects {
				if w := match(e, effects); w != "" {
					add(tr.ID, fmt.Sprintf("effect %s matches %q", e, w))
					break
				}
			}
		}
	}
	for id, reason := range reasons {
		if d.Dismissed(id) {
			dismissed++
			continue
		}
		cands = append(cands, Candidate{ID: id, Reason: reason})
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].ID < cands[j].ID })
	return cands, dismissed
}

// match returns the first word of the list that a word of the text starts
// with, or "".
func match(text string, list []string) string {
	ws := Words(text)
	for _, w := range list {
		if w == "" {
			continue
		}
		for _, x := range ws {
			if strings.HasPrefix(x, w) {
				return w
			}
		}
	}
	return ""
}

// Words splits a name or a text into lower-cased words (section 23.3): at
// every character that is not a letter or a digit, between a lower-case
// letter or a digit and an upper-case letter, and before the last upper-case
// letter of a run that a lower-case letter follows.
func Words(s string) []string {
	var words []string
	rs := []rune(s)
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, strings.ToLower(string(cur)))
			cur = nil
		}
	}
	for i, r := range rs {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if len(cur) > 0 && unicode.IsUpper(r) {
			prev := rs[i-1]
			nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				flush()
			}
		}
		cur = append(cur, r)
	}
	flush()
	return words
}
