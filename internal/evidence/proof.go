package evidence

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gitx"
)

// SelectIDs returns the obligations named by --ids and --prefix (section 13),
// sorted; all obligations when both are empty. A selector that matches nothing
// is an error (exit 2).
func SelectIDs(d *design.Design, ids, prefixes []string) ([]string, error) {
	if len(ids) == 0 && len(prefixes) == 0 {
		var all []string
		for _, o := range d.Obligations {
			all = append(all, o.ID)
		}
		return all, nil
	}
	set := map[string]bool{}
	for _, id := range ids {
		if d.IsProof(id) {
			return nil, fmt.Errorf("--ids: %s is a proof obligation: it awaits a machine-checked proof and has no evidence to show", id)
		}
		if d.Obligation(id) == nil {
			return nil, fmt.Errorf("--ids: %s is not an obligation", id)
		}
		set[id] = true
	}
	for _, pre := range prefixes {
		pre = strings.TrimSuffix(pre, "*")
		n := 0
		for _, o := range d.Obligations {
			if strings.HasPrefix(o.ID, pre) {
				set[o.ID] = true
				n++
			}
		}
		if n == 0 {
			return nil, fmt.Errorf("--prefix: %s matches no obligation", pre)
		}
	}
	var res []string
	for id := range set {
		res = append(res, id)
	}
	sort.Strings(res)
	return res, nil
}

// TierOf returns the tier of an id and the names of the areas giving it. An
// fv id has the tier of the critical area it names.
func TierOf(d *design.Design, id string) (string, []string) {
	tier, in := d.Tier(id)
	names := map[string]string{}
	for _, a := range in {
		names[a.Name] = a.Tier
	}
	if strings.HasPrefix(id, "fv:") {
		for _, a := range d.Areas {
			if "fv:"+a.Name == id {
				names[a.Name] = a.Tier
				if design.TierRank(a.Tier) > design.TierRank(tier) {
					tier = a.Tier
				}
			}
		}
	}
	var res []string
	for n, t := range names {
		if t == tier {
			res = append(res, n)
		}
	}
	sort.Strings(res)
	return tier, res
}

// HighestTier is the highest tier among ids and the areas giving it.
func HighestTier(d *design.Design, ids []string) (string, []string) {
	best := design.Routine
	areas := map[string]bool{}
	for _, id := range ids {
		t, names := TierOf(d, id)
		if design.TierRank(t) > design.TierRank(best) {
			best = t
			areas = map[string]bool{}
		}
		if t == best {
			for _, n := range names {
				areas[n] = true
			}
		}
	}
	var res []string
	for n := range areas {
		res = append(res, n)
	}
	sort.Strings(res)
	return best, res
}

// Notes are the parts of a proof block its author writes (section 13).
type Notes struct {
	Limits       []string
	Learnings    []string
	Observations []string // "<kind>: <text>"
}

// ProofFor writes the proof block of section 13 for the given obligation ids
// and returns 0 when every one is proven, 1 otherwise. The tier line is
// computed over tierIDs: the ids of a story include the actions it serves,
// which are not obligations but do carry a tier. extra are more lines under
// not proven (the unproven assumptions of the formal models covering the ids).
func ProofFor(w io.Writer, d *design.Design, ev *Evaluation, ids, tierIDs, extra []string, notes Notes) int {
	fmt.Fprintln(w, "PROOF")
	fmt.Fprintf(w, "commit:   %s\n", ev.Head)
	tier, areas := HighestTier(d, tierIDs)
	if len(areas) > 0 {
		fmt.Fprintf(w, "tier:     %s (%s)\n", tier, strings.Join(areas, ", "))
	} else {
		fmt.Fprintf(w, "tier:     %s\n", tier)
	}
	for _, r := range ev.Runs {
		fmt.Fprintf(w, "command:  %s\n", strings.Join(r.Command, " "))
		if r.Adapter {
			fmt.Fprintln(w, EnvLine(r.File, r.Env)) // section 13
		}
	}
	var p, f, s int
	for _, id := range ids {
		for _, r := range ev.Receipts(id) {
			switch r.Outcome {
			case Pass:
				p++
			case Fail:
				f++
			case Skip:
				s++
			}
		}
	}
	exit := 0
	if n := len(ev.Refused); n > 0 {
		// what a file that is not used holds is unknown: no count stands (section 13)
		exit = 1
		fmt.Fprintf(w, "result:   %d receipts file(s) refused: delete them and run the tests through machinery run\n", n)
	} else if ev.HasEvidence {
		fmt.Fprintf(w, "result:   %d passed / %d failed / %d skipped%s\n", p, f, s, against(ev.Runs))
	} else {
		fmt.Fprintf(w, "result:   no evidence at %s\n", gitx.Short(ev.Head))
	}
	if parts := adapterProven(ev, ids); len(parts) > 0 { // section 10.3
		fmt.Fprintf(w, "adapter:  %s\n", strings.Join(parts, ", "))
	}
	owed := d.OwedSet()
	fmt.Fprintln(w, "criteria:")
	var notProven []string
	for _, id := range ids {
		st := ev.State(id)
		_, isOwed := owed[id]
		if st != Proven {
			exit = 1
		}
		if line := criterion(d, ev, id, st); line != "" {
			fmt.Fprintf(w, "  %s\n", line)
		}
		if st == Partial || st == Unproven || st == PartlyRun || isOwed {
			label := st
			if pr, ok := ev.Partly(id); ok {
				label += ": " + pr.Text()
			}
			if isOwed {
				label += ", owed"
			}
			if _, held := ev.AdapterNotCounted(id); held {
				label += ", adapter pass not counted"
			}
			if _, none := ev.NoClaimant(id); none {
				label += ", no claimant can run here"
			}
			notProven = append(notProven, fmt.Sprintf("%s (%s)", id, label))
		}
	}
	fmt.Fprintln(w, "not proven:")
	for _, l := range append(notProven, extra...) {
		fmt.Fprintf(w, "  %s\n", l)
	}
	for _, part := range []struct {
		name  string
		lines []string
	}{{"limits", notes.Limits}, {"learnings", notes.Learnings}, {"observations", notes.Observations}} {
		fmt.Fprintf(w, "%s:\n", part.name)
		if len(part.lines) == 0 {
			fmt.Fprintln(w, "  none")
		}
		for _, l := range part.lines {
			fmt.Fprintf(w, "  %s\n", l)
		}
	}
	return exit
}

func against(runs []RunInfo) string {
	seen := map[string]bool{}
	var stacks []string
	for _, r := range runs {
		if r.Stack != "" && !seen[r.Stack] {
			seen[r.Stack] = true
			stacks = append(stacks, r.Stack)
		}
	}
	if len(stacks) == 0 {
		return ""
	}
	return ", against " + strings.Join(stacks, "; ")
}

// criterion renders the criteria line of one id, or "" when it has no receipt.
func criterion(d *design.Design, ev *Evaluation, id, st string) string {
	o := d.Obligation(id)
	if o.Kind == "fv" {
		if st != Proven {
			return ""
		}
		runs, det := 0, 0
		if FormalCounts != nil {
			runs, det = FormalCounts(d, id)
		}
		return fmt.Sprintf("%s -> formal -> pass: %d run(s), %d mutation(s) detected", id, runs, det)
	}
	rs := ev.Receipts(id)
	if len(rs) == 0 {
		return ""
	}
	// a receipt a test emitted explains the state before one the adapter wrote
	rs = append([]Receipt(nil), rs...)
	sort.SliceStable(rs, func(i, j int) bool { return rs[i].Source == "" && rs[j].Source != "" })
	r, label := pick(d, o, st, rs)
	line := fmt.Sprintf("%s -> %s -> %s: %s", id, r.Test, label, r.Observed)
	if g := ev.ByAdapter(id); g != "" && st == Proven {
		if n := ev.ExcusedOf(id); n > 0 {
			line += fmt.Sprintf(" (by adapter at %s granularity, %d claimant(s) excused)", g, n)
		} else {
			line += fmt.Sprintf(" (by adapter at %s granularity)", g)
		}
	}
	if st == Proven && d.IsOver(id) {
		line += " " + design.ProofAwaited // section 23.2
	}
	return line
}

// pick chooses the receipt that best explains the state.
func pick(d *design.Design, o *design.Obligation, st string, rs []Receipt) (Receipt, string) {
	expect := ""
	for _, t := range d.Traces {
		if t.ID == o.ID {
			expect = t.Expect
		}
	}
	for _, r := range rs {
		switch st {
		case Failed:
			if r.Outcome == Fail || (o.Kind == "cx" && r.Replay != "" && (r.Replay != expect || expect == Undetermined)) {
				return r, "FAIL"
			}
		case Proven:
			if o.Kind == "cx" && r.Replay == expect && r.Outcome != Skip {
				return r, "pass"
			}
			if o.Kind != "cx" && FullyObserved(o, r) {
				return r, "pass"
			}
		default:
			if r.Outcome == Pass {
				return r, "pass"
			}
		}
	}
	return rs[0], rs[0].Outcome
}

// adapterProven renders how many of the ids are proven by adapter.
func adapterProven(ev *Evaluation, ids []string) []string {
	function, file := ev.AdapterCounts(ids)
	return AdapterParts(function, file, "proven ")
}
