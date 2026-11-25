package evidence

import (
	"errors"
	"fmt"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
)

// SyncResult counts what owed sync did.
type SyncResult struct{ Kept, Added, Removed int }

// ErrNoRun is returned by Sync when no complete run is valid for the head.
var ErrNoRun = errors.New("no complete run")

// Sync rewrites the owed file from the evidence (section 11.2).
func Sync(p *project.Project, d *design.Design, ev *Evaluation, pruneOnly bool) (SyncResult, error) {
	var res SyncResult
	if len(d.OwedProblems) > 0 {
		return res, fmt.Errorf("%s", d.OwedProblems[0])
	}
	if !ev.HasEvidence {
		return res, ErrNoRun
	}
	listed := map[string]bool{}
	var out []design.OwedEntry
	for _, e := range d.Owed {
		listed[e.ID] = true
		if ev.State(e.ID) == Proven {
			res.Removed++
			continue
		}
		res.Kept++
		out = append(out, e)
	}
	if !pruneOnly {
		reason := "unproven at " + gitx.Short(ev.Head)
		for _, o := range d.Obligations {
			if !listed[o.ID] && ev.State(o.ID) != Proven {
				out = append(out, design.OwedEntry{ID: o.ID, Reason: reason})
				res.Added++
			}
		}
	}
	if res.Added+res.Removed == 0 {
		return res, nil
	}
	return res, design.WriteOwed(p, out)
}
