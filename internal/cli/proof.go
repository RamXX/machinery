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
	if !e.Parse(fs, args) {
		return 2
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
		tierIDs = backlog.ParseTags(is).IDs
		if sel = pm.ObligationIDs(d, tierIDs); len(sel) == 0 {
			return e.Usage("story %s names no obligation id", *story)
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
	code := evidence.ProofFor(io.MultiWriter(e.Stdout, &block), d, ev, sel, tierIDs, assumptionLines(d, sel))
	if *post {
		if c.Actor == "" {
			return e.Usage("no actor: pass --actor or set BEADS_ACTOR")
		}
		note := fmt.Sprintf("machinery:proof %s %s\n%s", ev.Head, c.Actor, strings.TrimRight(block.String(), "\n"))
		if err := c.Note(*story, note); err != nil {
			return e.Fail(err)
		}
	}
	return code
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
