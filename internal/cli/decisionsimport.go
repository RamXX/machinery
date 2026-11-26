package cli

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/pm"
	"github.com/RamXX/machinery/internal/project"
)

func init() {
	Register(&Command{Name: "decisions import", Summary: "Turn the records of an existing decision file into backlog decisions; --apply creates them.", Run: runDecisionsImport})
}

func runDecisionsImport(e *Env, args []string) int {
	fs := e.Flags("decisions import")
	path := fs.String("file", "", "the decision file, relative to the project root")
	record := fs.String("record", "", "a regular expression matching the heading line of one record")
	apply := fs.Bool("apply", false, "create the new decisions in the backlog, in one call")
	actor := fs.String("actor", "", "who records them (default BEADS_ACTOR, else git user.name)")
	pos, ok := parseMixed(e, fs, args)
	if !ok {
		return 2
	}
	switch {
	case len(pos) > 0:
		return e.Usage("machinery decisions import takes no arguments")
	case *path == "" || *record == "":
		return e.Usage("machinery decisions import needs --file <path> and --record \"<regex>\"")
	}
	re, err := regexp.Compile(*record)
	if err != nil {
		return e.Usage("--record is not a regular expression: %v", err)
	}
	if err := project.CheckPath("--file", *path); err != nil {
		var ce *project.ConfigError
		if errors.As(err, &ce) {
			return e.Usage("%s", ce.Msg)
		}
		return e.Usage("%v", err)
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	data, err := p.Read(*path)
	if err != nil {
		return e.Usage("--file %s: cannot read", *path)
	}
	recs := pm.DecisionRecords(string(data), re, *path, design.Load(p))
	if len(recs) == 0 {
		return e.Usage("--record %q matches no line of %s", *record, *path)
	}
	imported := map[string]string{}
	var c *backlog.Client
	notRead := ""
	if *apply {
		if c, err = backlog.New(p, *actor); err != nil {
			return e.Fail(err)
		}
		if c.Actor == "" {
			return e.Usage("no actor: pass --actor or set BEADS_ACTOR")
		}
		issues, err := c.List()
		if err != nil {
			return e.Fail(err)
		}
		imported = pm.ImportedSources(issues)
	} else if p.Cfg.Backlog == nil {
		notRead = "no backlog configured"
	} else if c, err = backlog.New(p, *actor); err != nil {
		notRead = "bd not runnable"
	} else if issues, err := c.List(); err != nil {
		notRead = "the backlog cannot be read"
	} else {
		imported = pm.ImportedSources(issues)
	}
	var fresh []pm.DecisionRecord
	naming := 0
	for _, r := range recs {
		if len(r.IDs) > 0 {
			naming++
		}
		if imported[r.Source] == "" {
			fresh = append(fresh, r)
		}
	}
	created := map[string]string{}
	if *apply && len(fresh) > 0 {
		g := backlog.Graph{Nodes: []backlog.Node{}, Edges: []backlog.Edge{}}
		for i, r := range fresh {
			g.Nodes = append(g.Nodes, backlog.Node{Key: fmt.Sprintf("d%d", i+1), Title: r.Title, Type: "decision",
				Priority: 2, Description: pm.ImportDescription(r, c.Actor)})
		}
		ids, err := c.CreateGraph(g)
		if err != nil {
			return e.Fail(err)
		}
		for i, r := range fresh {
			created[r.Source] = ids[fmt.Sprintf("d%d", i+1)]
		}
	}
	w := e.Stdout
	for _, r := range recs {
		governs := "none"
		if len(r.IDs) > 0 {
			governs = strings.Join(r.IDs, " ")
		}
		where := "new"
		switch {
		case imported[r.Source] != "":
			where = imported[r.Source]
		case created[r.Source] != "":
			where = created[r.Source]
		}
		fmt.Fprintf(w, "%d: %s\n  source: %s\n  governs: %s\n  backlog: %s\n", r.Line, r.Title, r.Source, governs, where)
	}
	if notRead != "" {
		fmt.Fprintf(w, "backlog: not read: %s; every record counts as new\n", notRead)
	}
	verb := "new"
	if *apply {
		verb = "created"
	}
	fmt.Fprintf(w, "decisions import: %d record(s) in %s, %d naming design ids, %d %s, %d imported already\n",
		len(recs), *path, naming, len(fresh), verb, len(recs)-len(fresh))
	n := pm.CountLines(string(data))
	fmt.Fprintf(w, "shed: %d line(s): an instructions file that loads %s is that much shorter once the file is removed from what agents load\n", n, *path)
	return 0
}
