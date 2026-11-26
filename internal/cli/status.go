package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/evidence"
	"github.com/RamXX/machinery/internal/gitx"
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
	if d.HasRisk {
		fmt.Fprintf(w, "tiers: routine %d, consequential %d, critical %d\n",
			tiers[design.Routine], tiers[design.Consequential], tiers[design.Critical])
	}
	ev, err := evidence.Evaluate(p, d, true)
	if err != nil {
		return e.Fail(err)
	}
	fmt.Fprintf(w, "proven: %d  partial: %d  failed: %d  owed: %d  unproven and not owed: %d\n",
		ev.NProven, ev.NPartial, ev.NFailed, ev.NOwed, ev.NUnowed)
	noStory := 0
	for _, o := range d.Owed {
		if o.Story == "" {
			noStory++
		}
	}
	fmt.Fprintf(w, "owed without a story: %d\n", noStory)
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
