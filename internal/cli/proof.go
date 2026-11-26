package cli

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/evidence"
	"github.com/RamXX/machinery/internal/formal"
	"github.com/RamXX/machinery/internal/pm"
)

func init() {
	Register(&Command{Name: "proof", Summary: "Print a proof block for the selected obligations.", Run: runProof})
}

func runProof(e *Env, args []string) int {
	fs := e.Flags("proof")
	var ids, prefixes StringList
	fs.Var(&ids, "ids", "comma-separated obligation ids (repeatable)")
	fs.Var(&prefixes, "prefix", "id prefix (repeatable)")
	story := fs.String("story", "", "select the ids of a backlog story")
	post := fs.Bool("post", false, "append the block to the story's notes (with --story)")
	actor := fs.String("actor", "", "who posts (default BEADS_ACTOR, else git user.name)")
	var limits, learnings, observations StringList
	fs.Var(&limits, "limit", "a claim still unproven (repeatable)")
	fs.Var(&learnings, "learning", "a rule the next story or the planner should know (repeatable)")
	fs.Var(&observations, "observation", "<kind>: <text>, something seen outside the story's scope (repeatable)")
	if !e.Parse(fs, args) {
		return 2
	}
	notes, msg := proofNotes(limits, learnings, observations)
	if msg != "" {
		return e.Usage("%s", msg)
	}
	switch {
	case fs.NArg() > 0:
		return e.Usage("machinery proof takes no arguments")
	case *story != "" && (len(ids) > 0 || len(prefixes) > 0):
		return e.Usage("--story does not go with --ids or --prefix")
	case *post && *story == "":
		return e.Usage("--post needs --story")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	d := design.Load(p)
	var sel, tierIDs []string
	var c *backlog.Client
	if *story != "" {
		var err error
		if c, err = backlog.New(p, *actor); err != nil {
			return e.Fail(err)
		}
		is, code := showStory(e, c, *story)
		if is == nil {
			return code
		}
		ids, capstone, err := storyIDs(c, is)
		if err != nil {
			return e.Fail(err)
		}
		tierIDs = ids
		if sel = pm.ObligationIDs(d, tierIDs); len(sel) == 0 && !capstone {
			return e.Usage("story %s names no obligation id", *story)
		}
		if tierIDs == nil {
			tierIDs = []string{}
		}
	} else {
		var err error
		if sel, err = evidence.SelectIDs(d, splitIDs(ids), prefixes); err != nil {
			return e.Usage("%v", err)
		}
	}
	ev, err := evidence.Evaluate(p, d, false)
	if err != nil {
		return e.Fail(err)
	}
	var block bytes.Buffer
	if tierIDs == nil {
		tierIDs = sel
	}
	extra := append(assumptionLines(d, sel), vitalLines(d, tierIDs)...)
	code := evidence.ProofFor(io.MultiWriter(e.Stdout, &block), d, ev, sel, tierIDs, extra, notes)
	if *post {
		if c.Actor == "" {
			return e.Usage("no actor: pass --actor or set BEADS_ACTOR")
		}
		note := fmt.Sprintf("machinery:proof %s %s\n%s", ev.Head, c.Actor, strings.TrimRight(block.String(), "\n"))
		for _, l := range notes.Learnings {
			note += "\nmachinery:learning " + l
		}
		for _, o := range notes.Observations {
			kind, text, _ := pm.ParseObservation(o)
			note += "\nmachinery:observation " + kind + " " + text
		}
		if err := c.Note(*story, note); err != nil {
			return e.Fail(err)
		}
	}
	return code
}

// proofNotes checks the author's parts of the block (section 13) and makes
// each value one line. msg is the usage error.
func proofNotes(limits, learnings, observations []string) (evidence.Notes, string) {
	var n evidence.Notes
	for _, part := range []struct {
		flag string
		in   []string
		out  *[]string
	}{{"limit", limits, &n.Limits}, {"learning", learnings, &n.Learnings}, {"observation", observations, &n.Observations}} {
		for _, v := range part.in {
			line := pm.OneLine(v)
			if line == "" {
				return n, fmt.Sprintf("--%s: a value must not be empty", part.flag)
			}
			if part.flag == "observation" {
				kind, text, ok := pm.ParseObservation(line)
				if !ok {
					return n, fmt.Sprintf("--observation %q: write <kind>: <text>, the kind one of %s", v, strings.Join(pm.ObservationKinds, ", "))
				}
				line = kind + ": " + text
			}
			*part.out = append(*part.out, line)
		}
	}
	return n, ""
}

// assumptionLines lists the unproven assumptions of every formal model whose
// areas contain a selected id (section 13).
func assumptionLines(d *design.Design, ids []string) []string {
	j, err := formal.Judge(d)
	if err != nil || len(j.Unproven) == 0 {
		return nil
	}
	areas := map[string]bool{}
	for _, id := range ids {
		_, in := d.Tier(id)
		for _, a := range in {
			areas[a.Name] = true
		}
		if name, ok := strings.CutPrefix(id, "fv:"); ok {
			areas[name] = true
		}
	}
	covers := map[string]bool{}
	for _, s := range j.Specs {
		for _, a := range s.Areas {
			covers[s.Spec] = covers[s.Spec] || areas[a]
		}
		for _, id := range ids {
			covers[s.Spec] = covers[s.Spec] || strings.HasPrefix(id, "cx:"+s.Spec+":")
		}
	}
	var res []string
	for _, u := range j.Unproven {
		if covers[u.Spec] {
			res = append(res, fmt.Sprintf("assumption %s %s: %s", u.Spec, u.ID, u.Text))
		}
	}
	sort.Strings(res)
	return res
}

// vitalLines lists, for the vital areas the ids belong to, the proof
// obligations awaiting proof and the assumptions whose evidence is owed
// (section 13).
func vitalLines(d *design.Design, ids []string) []string {
	areas := d.VitalAreasOf(ids)
	var res []string
	for _, id := range design.ProofsOf(areas) {
		res = append(res, id+" ("+design.AwaitingProof+")")
	}
	for _, u := range d.Unevidenced(areas) {
		res = append(res, u.Line())
	}
	return res
}

// splitIDs splits comma-separated flag values.
func splitIDs(vals []string) []string {
	var res []string
	for _, v := range vals {
		for _, id := range strings.Split(v, ",") {
			if id = strings.TrimSpace(id); id != "" {
				res = append(res, id)
			}
		}
	}
	return res
}
