package evidence

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/project"
	"github.com/RamXX/machinery/v2/internal/report"
)

// NotCounted is an adapter pass the policy does not let count (section 10.3).
type NotCounted struct {
	Granularity string
	Why         string
}

// adapterWhy returns "" when the policy evidence.adapter lets an adapter
// pass of the granularity count for the obligation, and otherwise why not.
func adapterWhy(p *project.Project, d *design.Design, o *design.Obligation, granularity string) string {
	kinds, tiers, min := p.Cfg.AdapterPolicy()
	tier, areas := TierOf(d, o.ID)
	need := min.Default
	for _, key := range []string{o.Kind, tier} {
		if min.By[key] == project.GranularityFunction {
			need = project.GranularityFunction
		}
	}
	switch {
	case !contains(project.AdapterKinds, o.Kind):
		return fmt.Sprintf("a %s id is never proven by the adapter", o.Kind)
	case !contains(kinds, o.Kind):
		return "evidence.adapter.allow does not list " + o.Kind
	case !contains(tiers, tier) && tier == design.Routine:
		return "evidence.adapter.tiers does not list routine"
	case !contains(tiers, tier):
		return fmt.Sprintf("it is in a %s area (%s): it needs a receipt the test emits after its own assertions", tier, strings.Join(areas, ", "))
	case need == project.GranularityFunction && granularity != project.GranularityFunction:
		return "evidence.adapter.min_granularity asks for function granularity"
	}
	return ""
}

// admitted drops the adapter passes the policy does not let count and
// returns them by id, the function-granularity one when an id has both. A
// fail and a skip of the adapter are kept: a fail fails whatever the policy.
func admitted(p *project.Project, d *design.Design, recs []Receipt) ([]Receipt, map[string]NotCounted) {
	var res []Receipt
	out := map[string]NotCounted{}
	for _, r := range recs {
		if o := d.Obligation(r.ID); o != nil && r.Source == SourceAdapter && r.Outcome == Pass {
			if why := adapterWhy(p, d, o, r.Granularity); why != "" {
				if cur, seen := out[r.ID]; !seen || (cur.Granularity != project.GranularityFunction && r.Granularity == project.GranularityFunction) {
					out[r.ID] = NotCounted{Granularity: r.Granularity, Why: why}
				}
				continue
			}
		}
		res = append(res, r)
	}
	return res, out
}

// byAdapter returns the granularity at which a proven obligation is proven
// by adapter, or "" when a receipt a test emitted proves it on its own.
func byAdapter(o *design.Obligation, rs []Receipt) string {
	gran := ""
	for _, r := range rs {
		switch {
		case r.Outcome != Pass:
		case r.Source != SourceAdapter:
			if FullyObserved(o, r) {
				return ""
			}
		case r.Granularity == project.GranularityFunction || gran == "":
			gran = r.Granularity
		}
	}
	return gran
}

// ByAdapter returns "function" or "file" for an id that is proven by
// adapter evidence alone, and "" for any other id.
func (e *Evaluation) ByAdapter(id string) string { return e.adapter[id] }

// AdapterNotCounted returns the adapter pass of an id that is not proven and
// that the policy does not let count; ok is false when there is none.
func (e *Evaluation) AdapterNotCounted(id string) (n NotCounted, ok bool) {
	n, ok = e.notCounted[id]
	return n, ok
}

// AdapterNotCountedIDs returns the ids AdapterNotCounted knows, sorted.
func (e *Evaluation) AdapterNotCountedIDs() []string {
	ids := make([]string, 0, len(e.notCounted))
	for id := range e.notCounted {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// AdapterParts renders how many ids are proven by adapter, by granularity:
// "<n> <prefix>by adapter at function granularity" and the same for file,
// each only when it counts something.
func AdapterParts(function, file int, prefix string) []string {
	var parts []string
	if function > 0 {
		parts = append(parts, fmt.Sprintf("%d %sby adapter at function granularity", function, prefix))
	}
	if file > 0 {
		parts = append(parts, fmt.Sprintf("%d %sby adapter at file granularity", file, prefix))
	}
	return parts
}

// CountAdapter adds to the checked line of a stage how many of the proven
// are proven by adapter, each part only when it counts something.
func CountAdapter(s *report.Stage, function, file int) {
	if function > 0 {
		s.Count(function, "by adapter at function granularity")
	}
	if file > 0 {
		s.Count(file, "by adapter at file granularity")
	}
}

// AdapterCounts counts, among ids, those proven by adapter, by granularity.
func (e *Evaluation) AdapterCounts(ids []string) (function, file int) {
	for _, id := range ids {
		switch e.adapter[id] {
		case project.GranularityFunction:
			function++
		case project.GranularityFile:
			file++
		}
	}
	return function, file
}
