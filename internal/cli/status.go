package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/evidence"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/landing"
	"github.com/RamXX/machinery/internal/tags"
)

func init() {
	Register(&Command{Name: "status", Summary: "Print a summary of obligations and evidence.", Run: runStatus})
}

var statusKinds = []string{"tc", "sc", "inv", "fm", "fv", "cx", "ck"}

func runStatus(e *Env, args []string) int {
	fs := e.Flags("status")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery status takes no arguments")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	d := design.Load(p)
	w := e.Stdout
	if red, err := landing.LoadRed(p); err == nil && red != nil {
		fmt.Fprintf(w, "red: since %s: repair or revert first\n", red.What())
	}
	fmt.Fprintf(w, "project: %s\n", filepath.Base(p.Root))
	kinds := map[string]int{}
	tiers := map[string]int{}
	for _, o := range d.Obligations {
		kinds[o.Kind]++
		t, _ := evidence.TierOf(d, o.ID)
		tiers[t]++
	}
	var parts []string
	for _, k := range statusKinds {
		if kinds[k] > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", k, kinds[k]))
		}
	}
	if len(parts) > 0 {
		fmt.Fprintf(w, "obligations: %d (%s)\n", len(d.Obligations), strings.Join(parts, ", "))
	} else {
		fmt.Fprintf(w, "obligations: 0\n")
	}
	vas := d.VitalAreas()
	if d.HasRisk {
		line := fmt.Sprintf("tiers: routine %d, consequential %d, critical %d",
			tiers[design.Routine], tiers[design.Consequential], tiers[design.Critical])
		if len(vas) > 0 {
			line += fmt.Sprintf(", vital %d", tiers[design.Vital])
		}
		fmt.Fprintln(w, line)
	}
	proposed, owed, proven, retired := d.StatusCounts()
	fmt.Fprintf(w, "ids: %d proposed, %d owed, %d proven, %d retired\n", proposed, owed, proven, retired)
	ev, err := evidence.Evaluate(p, d, true)
	if err != nil {
		return e.Fail(err)
	}
	fmt.Fprintf(w, "proven: %d  partial: %d  failed: %d  owed: %d  unproven and not owed: %d\n",
		ev.NProven, ev.NPartial, ev.NFailed, ev.NOwed, ev.NUnowed)
	if len(vas) > 0 {
		for _, l := range evidence.VitalOf(ev, vas, nil).Lines() { // section 23.2
			fmt.Fprintln(w, l)
		}
	}
	noStory := 0
	for _, o := range d.Owed {
		if o.Story == "" {
			noStory++
		}
	}
	fmt.Fprintf(w, "owed without a story: %d\n", noStory)
	if design.Adopted(d.Owed) { // section 14: an adopted project sees its owed ids by claim
		claims, err := tags.Read(p, d)
		if err != nil {
			return e.Fail(err)
		}
		proven := func(id string) bool { return ev.State(id) == evidence.Proven }
		for _, l := range claims.SplitOwed(d, proven).Lines("owed, ") {
			fmt.Fprintln(w, l)
		}
	}
	if !ev.HasEvidence {
		fmt.Fprintf(w, "evidence: none at %s\n", gitx.Short(ev.Head))
	} else {
		state := "clean"
		for _, r := range ev.Runs {
			if r.Dirty {
				state = "dirty"
			}
		}
		if ev.Dirty {
			state = "dirty"
		}
		fmt.Fprintf(w, "evidence: %s %s, %d run(s), %d receipt(s)\n", gitx.Short(ev.Head), state, len(ev.Runs), ev.NReceipts)
	}
	statusIntakes(e, p)
	return 0
}
