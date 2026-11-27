package stages

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/evidence"
	"github.com/RamXX/machinery/v2/internal/report"
	"github.com/RamXX/machinery/v2/internal/tags"
)

func init() { Bind("trace", Trace) }

// claimed are the obligation kinds a test must verify (section 8). fv ids are
// proven by the formal stage.
var claimed = map[string]bool{"inv": true, "sc": true, "tc": true, "fm": true, "cx": true, "ck": true}

// NonAction is one id that a serves tag names and that is not an action
// (section 8).
type NonAction struct {
	File string
	Line int
	ID   string
}

// Trace is stage trace (section 8).
func Trace(ctx *Context) *report.Stage {
	s, _, _, _ := trace(ctx)
	return s
}

// NonActionServes returns what machinery trace --non-action-serves lists:
// every id a serves tag names that is not an action, once per tag, by file,
// line and id, and the paths of the symbolic links the stage reports, whose
// tags are not read. unread says why the files could not all be listed or
// read; the list is then not known, and is never given as if it were whole.
func NonActionServes(ctx *Context) (list []NonAction, links, unread []string) {
	_, list, links, unread = trace(ctx)
	if len(unread) > 0 {
		return nil, nil, unread
	}
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.ID < b.ID
	})
	return list, links, nil
}

// trace runs the stage and returns, with its result, the serves tags that
// name an id which is not an action, the symbolic links it reports, and what
// kept it from listing or reading every file.
func trace(ctx *Context) (*report.Stage, []NonAction, []string, []string) {
	s := report.New("trace")
	p := ctx.P
	if p.Cfg.Trace == nil {
		return s.NotApplicable("no trace configured"), nil, nil, nil
	}
	var unread []string
	cannot := func(format string, err error) {
		unread = append(unread, fmt.Sprintf(format, err))
		s.Errorf(format, err)
	}
	d := ctx.Design()
	code, err := tags.CodeFiles(p)
	if err != nil {
		cannot("cannot list code files: %v", err)
	}
	tests, err := tags.TestFiles(p)
	if err != nil {
		cannot("cannot list test files: %v", err)
	}
	if len(p.Cfg.Trace.Code) > 0 && len(code) == 0 {
		s.Errorf("trace.code matches no file")
	}
	if len(p.Cfg.Trace.Tests) > 0 && len(tests) == 0 {
		s.Errorf("trace.tests matches no file")
	}
	links, err := tags.Links(p)
	if err != nil {
		cannot("cannot list symbolic links: %v", err)
	}
	var linkPaths []string
	for _, l := range links {
		linkPaths = append(linkPaths, l.Path)
		s.Errorf("trace: %s is a symbolic link; markers behind it are not read: replace it with the files, or leave it out of %s", l.Path, strings.Join(l.Lists, " and "))
	}
	found, err := tags.Scan(p, append(append([]string{}, code...), tests...))
	if err != nil {
		cannot("cannot read tags: %v", err)
	}

	layout, err := evidence.GoLayout(p, d, found) // the layout of the Go modules (section 10.3)
	if err != nil {
		cannot("cannot list go.mod files: %v", err)
	}
	for _, msg := range layout {
		s.Errorf("adapter: %s", msg)
	}

	for _, msg := range d.OwedProblems {
		s.Errorf("%s", msg)
	}
	owed := d.OwedSet()
	for _, msg := range d.OwedIssues() {
		s.Errorf("%s", msg)
	}

	// a proposed or retired id is an id of the design, which nothing may claim
	ids := d.DeclaredIDs()
	scenarios := d.ScenarioIDs()
	claims := tags.ClaimsOf(d, found)
	nonAction, warned, nonActionTags := map[string]bool{}, map[string]bool{}, map[string]bool{}
	var nonActions []NonAction
	claimedHeld := map[string]bool{}
	proofs := d.ProofIDs()
	for _, t := range found {
		for _, tok := range t.IDs {
			if !t.Scenario && strings.HasPrefix(tok, "pf:") {
				// a proof obligation is matched against the proof obligations alone (section 8)
				hits := tags.Expand(tok, proofs)
				if len(hits) == 0 {
					s.Errorf("unknown-id: %s:%d: %s", t.File, t.Line, tok)
				}
				for _, id := range hits {
					if key := id + " " + t.File; !claimedHeld[key] {
						claimedHeld[key] = true
						s.Errorf("proof: %s is claimed by %s; tests do not discharge a proof obligation: it awaits a machine-checked proof", id, t.File)
					}
				}
				continue
			}
			hits := tags.Resolve(t, tok, ids, scenarios)
			if len(hits) == 0 {
				if t.Exact {
					continue // text that is no scenario name claims nothing (section 8.1)
				}
				if t.Scenario {
					s.Errorf("unknown-id: %s:%d: scenario %q", t.File, t.Line, tok)
				} else {
					s.Errorf("unknown-id: %s:%d: %s", t.File, t.Line, tok)
				}
				continue
			}
			for _, id := range hits {
				if st := d.Status(id); st != "" {
					// a pattern claims only the ids that yield an obligation
					if key := id + " " + t.File; !strings.HasSuffix(tok, "*") && !claimedHeld[key] {
						claimedHeld[key] = true
						if st == design.Proposed {
							s.Errorf("proposed: %s is claimed by %s; commit it first", id, t.File)
						} else {
							s.Errorf("retired: %s is claimed by %s; remove the claim, or un-retire it", id, t.File)
						}
					}
					continue
				}
				if key := fmt.Sprintf("%s:%d: %s", t.File, t.Line, id); t.Kind == tags.Serves && !d.IsAct(id) && !warned[key] {
					warned[key] = true
					nonAction[id] = true
					nonActionTags[fmt.Sprintf("%s:%d", t.File, t.Line)] = true
					nonActions = append(nonActions, NonAction{File: t.File, Line: t.Line, ID: id})
				}
			}
		}
	}
	for _, a := range d.Served {
		if _, isOwed := owed[a.ID]; !claims.Served[a.ID] && !isOwed {
			s.Errorf("unserved: %s", a.ID)
		}
	}
	for _, o := range d.Obligations {
		if _, isOwed := owed[o.ID]; claimed[o.Kind] && len(claims.Tests[o.ID]) == 0 && !isOwed {
			s.Errorf("unverified: %s", o.ID)
		}
	}

	s.Count(len(code), "code files").Count(len(tests), "test files").
		Count(len(d.Served), "act ids").Count(len(d.Obligations), "obligations")
	if len(nonAction) > 0 {
		// one counted line for the run, whatever the number of tags: a
		// project whose convention is to serve invariants has hundreds
		s.Warnf("serves: %d tag(s) name %d id(s) that are not actions; machinery trace --non-action-serves lists them", len(nonActionTags), len(nonAction))
		s.Count(len(nonAction), "non-action serves")
	}
	if len(d.Served)+len(d.Obligations) == 0 && s.Blocking() == 0 {
		s.NotDemonstrated("nothing to check")
	}
	return s, nonActions, linkPaths, unread
}
