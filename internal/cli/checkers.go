package cli

import (
	"bytes"
	"flag"
	"fmt"
	"strings"

	"github.com/RamXX/machinery/internal/checkers"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gen"
	"github.com/RamXX/machinery/internal/project"
)

func init() {
	Register(&Command{Name: "checkers run", Summary: "Run the registered external checkers and record their evidence.", Run: runCheckersRun})
	Register(&Command{Name: "checkers list", Summary: "List the configured checkers, their registration and evidence.", Run: runCheckersList})
}

func runCheckersRun(e *Env, args []string) int {
	fs := e.Flags("checkers run")
	registry := fs.String("registry", "", "registry file (default "+checkers.DefaultRegistry+")")
	ids, ok := parseInterspersed(e, fs, args)
	if !ok {
		return 2
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	selected, err := selectCheckers(p, ids)
	if err != nil {
		return e.Fail(err)
	}
	d := design.Load(p)
	proj, factProblems := checkers.Build(d)
	data, err := proj.Bytes()
	if err != nil {
		return e.Fail(err)
	}
	if len(factProblems) > 0 {
		for _, pr := range factProblems {
			fmt.Fprintf(e.Stdout, "ERROR  %s: %s\n", p.Cfg.Design.Facts, pr)
		}
		return 1
	}
	if got, rerr := p.Read(checkers.ProjectionPath(d)); rerr != nil || !bytes.Equal(got, data) {
		fmt.Fprintln(e.Stdout, "ERROR  run machinery gen first")
		return 1
	}
	reg, err := checkers.LoadRegistry(p, *registry)
	if err != nil {
		return e.Fail(err)
	}
	entries := make([]checkers.Entry, len(selected))
	for i, ck := range selected {
		if entries[i], err = reg.Command(ck.ID); err != nil {
			return e.Fail(err)
		}
	}
	failed := false
	for i, ck := range selected {
		out := checkers.Run(d, ck, entries[i], proj, data)
		switch {
		case out.DidNotRun != "":
			fmt.Fprintf(e.Stdout, "ERROR  checker %s did not run: %s\n", ck.ID, out.DidNotRun)
			failed = true
		case len(out.Problems) > 0:
			for _, pr := range out.Problems {
				fmt.Fprintf(e.Stdout, "ERROR  %s: %s\n", ck.Evidence, pr)
			}
			failed = true
		default:
			ev := out.Evidence
			for _, f := range ev.Findings {
				label := "warn  "
				if f.Severity == "error" {
					label = "ERROR "
				}
				fmt.Fprintf(e.Stdout, "%s %s\n", label, checkers.FindingLine(ck.ID, f))
			}
			fmt.Fprintf(e.Stdout, "checker %s: %s, %d finding(s), %d obligation(s), %d covered\n",
				ck.ID, ev.Verdict, len(ev.Findings), len(ev.Obligations), len(ev.Covers))
			failed = failed || ev.Verdict != "pass"
		}
	}
	if err := refreshObligations(p); err != nil {
		return e.Fail(err)
	}
	if failed {
		return 1
	}
	return 0
}

// parseInterspersed parses flags that may follow the checker ids.
func parseInterspersed(e *Env, fs *flag.FlagSet, args []string) (ids []string, ok bool) {
	for {
		if !e.Parse(fs, args) {
			return nil, false
		}
		if fs.NArg() == 0 {
			return ids, true
		}
		ids = append(ids, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

// refreshObligations rewrites obligations.json so new ck obligations appear.
func refreshObligations(p *project.Project) error {
	d := design.Load(p)
	data, err := gen.Obligations(d)
	if err != nil {
		return err
	}
	rel := gen.Dir(d) + "/obligations.json"
	if got, rerr := p.Read(rel); rerr == nil && bytes.Equal(got, data) {
		return nil
	}
	return p.WriteAtomic(rel, data)
}

// selectCheckers returns the configured checkers named by ids, or all of them.
func selectCheckers(p *project.Project, ids []string) ([]project.Checker, error) {
	all := p.Cfg.Checkers
	if len(all) == 0 {
		return nil, project.Errf("no checkers configured: add checkers to machinery.yaml")
	}
	if len(ids) == 0 {
		return all, nil
	}
	var res []project.Checker
	for _, id := range ids {
		found := false
		for _, ck := range all {
			if ck.ID == id {
				res = append(res, ck)
				found = true
			}
		}
		if !found {
			var names []string
			for _, ck := range all {
				names = append(names, ck.ID)
			}
			return nil, project.Errf("no checker %s in machinery.yaml (configured: %s)", id, strings.Join(names, ", "))
		}
	}
	return res, nil
}

func runCheckersList(e *Env, args []string) int {
	fs := e.Flags("checkers list")
	registry := fs.String("registry", "", "registry file (default "+checkers.DefaultRegistry+")")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery checkers list takes no arguments")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	reg, err := checkers.LoadRegistry(p, *registry)
	if err != nil {
		return e.Fail(err)
	}
	d := design.Load(p)
	proj, _ := checkers.Build(d)
	data, err := proj.Bytes()
	if err != nil {
		return e.Fail(err)
	}
	sha, ids := checkers.SHA256(data), proj.IDs()
	if len(p.Cfg.Checkers) == 0 {
		fmt.Fprintln(e.Stdout, "no checkers configured")
		return 0
	}
	for _, ck := range p.Cfg.Checkers {
		state := "not registered"
		if reg.Registered(ck.ID) {
			state = "registered"
		}
		a := checkers.Assess(p, ck, sha, ids)
		fmt.Fprintf(e.Stdout, "%s  %s  evidence: %s\n", ck.ID, state, a.Status)
	}
	return 0
}
