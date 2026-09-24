// The consistency-layer adoption baseline: how an existing design records
// today's Gy-rules findings and Gl-ledger undeclared-fact warnings in
// design/ratchet.json, beside G4's baselined edges, and burns them down.
//
// A recorded finding is reported as a NOTE ("baselined: ..."), never an ERROR
// or a warning, and counted on its gate's checked line; a finding the ratchet
// does not record reports exactly as it does without one; a recorded entry
// no longer observed is a NOTE ("... resolved; run machinery baseline to
// shrink the ratchet"), so burning down debt never breaks a build. A later
// `machinery baseline --gate gy,gl` keeps only the recorded entries that are
// still observed (the ratchet only shrinks) unless --grow accepts new ones.

package gates

import (
	"fmt"
	"strings"
)

// BaselinedCount is the checked-line label of the findings a gate reports as
// baselined notes.
const BaselinedCount = "baselined"

const (
	resolvedCount   = "baselined entries resolved"
	baselinedPrefix = "baselined: "
	resolvedSuffix  = " resolved; run machinery baseline to shrink the ratchet"
)

// DebtRecord is what one `machinery baseline --gate gy|gl` did for one gate,
// in occurrences (a Gl key recorded with count 2 is two).
type DebtRecord struct {
	Gate        string // "Gy-rules" or "Gl-ledger"
	Observed    int    // findings the gate reports today
	Recorded    int    // occurrences the ratchet now tolerates
	NotRecorded int    // observed but left out: new since the last baseline (no --grow); they keep blocking
	Dropped     int    // previously recorded occurrences no longer observed (the shrink)
	First       bool   // the section was absent, so every observed finding was recorded
}

// RecordConsistencyDebt rewrites r's rule_findings (gy) and undeclared_facts
// (gl) sections from the design's current findings. A gate whose section is
// absent records every current finding (adoption). Otherwise the new section
// is the current set intersected with the recorded one (counts take the
// minimum), so the ratchet only shrinks, unless grow is set, which records the
// current set as it is. Nothing in r changes when an error is returned: Gy
// refuses while the design has projection errors, which are not debt.
func RecordConsistencyDebt(design, impl string, r *Ratchet, gy, gl, grow bool) ([]DebtRecord, error) {
	var recs []DebtRecord
	var rules []RuleDebt
	var undeclared []UndeclaredDebt
	if gy {
		current, err := ObserveRuleDebt(design, impl)
		if err != nil {
			return nil, err
		}
		rec := DebtRecord{Gate: "Gy-rules", Observed: len(current), First: r.Rules == nil}
		prior := map[string]bool{}
		for _, d := range r.Rules {
			prior[d.key()] = true
		}
		kept := map[string]bool{}
		rules = []RuleDebt{}
		for _, d := range current {
			if rec.First || grow || prior[d.key()] {
				rules = append(rules, d)
				kept[d.key()] = true
			} else {
				rec.NotRecorded++
			}
		}
		for k := range prior {
			if !kept[k] {
				rec.Dropped++
			}
		}
		rec.Recorded = len(rules)
		recs = append(recs, rec)
	}
	if gl {
		current := ObserveUndeclaredDebt(design)
		rec := DebtRecord{Gate: "Gl-ledger", First: r.Undeclared == nil}
		prior := map[string]int{}
		for _, d := range r.Undeclared {
			prior[d.key()] = d.Count
		}
		undeclared = []UndeclaredDebt{}
		for _, d := range current {
			rec.Observed += d.Count
			keep := d.Count
			if !rec.First && !grow {
				keep = min(d.Count, prior[d.key()])
			}
			rec.NotRecorded += d.Count - keep
			if keep > 0 {
				d.Count = keep
				undeclared = append(undeclared, d)
			}
		}
		recorded := map[string]int{}
		for _, d := range undeclared {
			recorded[d.key()] = d.Count
			rec.Recorded += d.Count
		}
		for k, n := range prior {
			rec.Dropped += max(0, n-recorded[k])
		}
		recs = append(recs, rec)
	}
	if gy {
		r.Rules = rules
	}
	if gl {
		r.Undeclared = undeclared
	}
	return recs, nil
}

// requireNoBaselinedDebt closes final handoff over the consistency baseline:
// --complete refuses while any Gy-rules or Gl-ledger finding is still
// tolerated by ratchet.json, the way it refuses an open milestone, and prints
// the count per gate. G4's baselined edges are judged by G4 alone.
func requireNoBaselinedDebt(final *Gate, run []*Gate) {
	counts := map[string]int{}
	for _, g := range run {
		if name := strings.Fields(g.Title); len(name) > 0 {
			counts[name[0]] += g.Counts[BaselinedCount]
		}
	}
	total := 0
	var parts []string
	for _, name := range []string{"Gy-rules", "Gl-ledger"} {
		if n := counts[name]; n > 0 {
			total += n
			parts = append(parts, fmt.Sprintf("%s %d", name, n))
		}
	}
	if total == 0 {
		return
	}
	final.Errs = append(final.Errs, fmt.Sprintf("%d baselined consistency finding(s) remain (%s); final handoff requires the Gy/Gl debt recorded in %s burned down: fix each, then rerun machinery baseline to shrink the ratchet",
		total, strings.Join(parts, ", "), RatchetFile))
}
