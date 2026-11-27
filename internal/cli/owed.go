package cli

import (
	"errors"
	"fmt"

	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/evidence"
	"github.com/RamXX/machinery/v2/internal/gitx"
)

func init() {
	Register(&Command{Name: "owed list", Summary: "Print the owed list.", Run: runOwedList})
	Register(&Command{Name: "owed sync", Summary: "Rewrite owed.yaml from the evidence; --prune-only never adds.", Run: runOwedSync})
}

func runOwedList(e *Env, args []string) int {
	fs := e.Flags("owed list")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery owed list takes no arguments")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	d := design.Load(p)
	entries := append([]design.OwedEntry(nil), d.Owed...)
	sortOwed(entries)
	for _, o := range entries {
		if o.Story != "" {
			fmt.Fprintf(e.Stdout, "%s  %s  %s\n", o.ID, o.Reason, o.Story)
		} else {
			fmt.Fprintf(e.Stdout, "%s  %s\n", o.ID, o.Reason)
		}
	}
	noun := "entries"
	if len(entries) == 1 {
		noun = "entry"
	}
	fmt.Fprintf(e.Stdout, "owed: %d %s\n", len(entries), noun)
	problems := append(append([]string(nil), d.OwedProblems...), d.OwedIssues()...)
	for _, pr := range problems {
		fmt.Fprintf(e.Stderr, "ERROR  %s\n", pr)
	}
	if len(problems) > 0 {
		return 1
	}
	return 0
}

func sortOwed(es []design.OwedEntry) {
	for i := 1; i < len(es); i++ {
		for j := i; j > 0 && es[j].ID < es[j-1].ID; j-- {
			es[j], es[j-1] = es[j-1], es[j]
		}
	}
}

func runOwedSync(e *Env, args []string) int {
	fs := e.Flags("owed sync")
	pruneOnly := fs.Bool("prune-only", false, "only remove proven entries")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery owed sync takes no arguments")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	d := design.Load(p)
	ev, err := evidence.Evaluate(p, d, false)
	if err != nil {
		return e.Fail(err)
	}
	// a receipts file that is refused or incomplete may hold a failure: nothing is pruned (section 11.2)
	if refused := ev.RefusedFindings(); len(refused) > 0 {
		for _, f := range refused {
			fmt.Fprintf(e.Stdout, "ERROR  %s\n", f.Msg)
		}
		if !ev.HasEvidence {
			fmt.Fprintf(e.Stdout, "ERROR  no complete run valid for %s: run the tests through machinery run first\n", gitx.Short(ev.Head))
		}
		return 1
	}
	res, err := evidence.Sync(p, d, ev, *pruneOnly)
	if errors.Is(err, evidence.ErrNoRun) {
		fmt.Fprintf(e.Stdout, "ERROR  no complete run valid for %s: run the tests through machinery run first\n", gitx.Short(ev.Head))
		return 1
	}
	if err != nil {
		fmt.Fprintf(e.Stdout, "ERROR  %v\n", err)
		return 1
	}
	fmt.Fprintf(e.Stdout, "owed: %d kept, %d added, %d removed\n", res.Kept, res.Added, res.Removed)
	return 0
}
