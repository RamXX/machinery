package cli

import (
	"fmt"
	"strings"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/pm"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/stages"
)

func init() {
	Register(&Command{Name: "intake new", Summary: "Record a requirement as an intake item, before any story.", Run: runIntakeNew})
	Register(&Command{Name: "intake status", Summary: "Print where a requirement stands, one line per step.", Run: runIntakeStatus})
	Register(&Command{Name: "intake plan", Summary: "Print the stories a design change owes; --apply creates them at once.", Run: runIntakePlan})
}

func runIntakeNew(e *Env, args []string) int {
	fs := e.Flags("intake new")
	intent := fs.String("intent", "", "who needs what, and the observable result")
	var refuses StringList
	fs.Var(&refuses, "refuses", "what must be refused (repeatable)")
	parent := fs.String("parent", "", "parent of the intake item in the backlog")
	pos, ok := parseMixed(e, fs, args)
	if !ok {
		return 2
	}
	switch {
	case len(pos) != 1 || strings.TrimSpace(pos[0]) == "":
		return e.Usage("machinery intake new takes one title")
	case strings.TrimSpace(*intent) == "":
		return e.Usage("machinery intake new needs --intent")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	c, err := backlog.New(p, "")
	if err != nil {
		return e.Fail(err)
	}
	head, err := gitx.Head(p.Root)
	if err != nil {
		return e.Fail(fmt.Errorf("cannot read the current commit: %v", err))
	}
	d := design.Load(p)
	dirty, err := gitx.StatusPaths(p.Root)
	if err != nil {
		return e.Fail(err)
	}
	if u := pm.UncommittedDesign(d, dirty); len(u) > 0 {
		fmt.Fprintf(e.Stderr, "machinery: uncommitted design change: %s; a requirement starts from a committed design\n", strings.Join(u, ", "))
		return 1
	}
	desc := pm.IntakeDescription(*intent, refuses, head, pm.ContractOf(d).Hash)
	id, err := c.Create(pm.OneLine(pos[0]), "feature", desc, *parent, []string{pm.IntakeLabel})
	if err != nil {
		return e.Fail(err)
	}
	fmt.Fprintln(e.Stdout, id)
	return 0
}

func runIntakeStatus(e *Env, args []string) int {
	fs := e.Flags("intake status")
	asJSON := fs.Bool("json", false, "print the status as one JSON document")
	pos, ok := parseMixed(e, fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		return e.Usage("machinery intake status takes one intake id")
	}
	p, c, in, code := openIntake(e, pos[0])
	if in == nil {
		return code
	}
	ds, issues, err := intakeState(p, c)
	if err != nil {
		return e.Fail(err)
	}
	st := ds.Assess(in, issues)
	if *asJSON {
		return st.JSON(e.Stdout)
	}
	return st.Print(e.Stdout)
}

func runIntakePlan(e *Env, args []string) int {
	fs := e.Flags("intake plan")
	apply := fs.Bool("apply", false, "create the stories and edges in the backlog")
	pos, ok := parseMixed(e, fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		return e.Usage("machinery intake plan takes one intake id")
	}
	p, c, in, code := openIntake(e, pos[0])
	if in == nil {
		return code
	}
	_, frozen, err := pm.LoadWave(p)
	if err != nil {
		return e.Fail(err)
	}
	if frozen {
		fmt.Fprintln(e.Stderr, "machinery: a wave is frozen: land or thaw it; a contract change lands alone")
		return 1
	}
	ds, issues, err := intakeState(p, c)
	if err != nil {
		return e.Fail(err)
	}
	st := ds.Assess(in, issues)
	if refusals := st.Refusals(); len(refusals) > 0 {
		for _, r := range refusals {
			fmt.Fprintln(e.Stderr, "machinery: "+r)
		}
		return 1
	}
	if probs := ds.D.OwedProblems; len(probs) > 0 {
		return e.Fail(fmt.Errorf("%s", probs[0]))
	}
	plan := st.Plan(ds.D)
	if len(st.NewIDs) == 0 {
		fmt.Fprintf(e.Stdout, "intake %s: no new id: the design change adds none; only the capstone is planned\n", in.ID)
	}
	if len(plan.Stories) == 0 {
		fmt.Fprintf(e.Stdout, "intake %s: every new id has a story; nothing to create\n", in.ID)
	}
	if !*apply {
		plan.Markdown(e.Stdout)
		return 0
	}
	res, err := pm.ApplyIntake(c, ds, st, plan)
	if err != nil {
		return e.Fail(err)
	}
	fmt.Fprintf(e.Stdout, "plan: %d created, %d existing, %d edge(s)\n", res.Created, res.Existing, res.Edges)
	fmt.Fprintf(e.Stdout, "owed: %d added\n", res.Owed)
	if res.Planned {
		fmt.Fprintf(e.Stdout, "intake %s: planned at %s, contract %s\n", in.ID, gitx.Short(ds.Head), ds.Contract.Short())
	}
	return 0
}

// openIntake loads the project and the backlog and reads one intake item. On
// failure in is nil and code is the exit code.
func openIntake(e *Env, id string) (p *project.Project, c *backlog.Client, in *pm.Intake, code int) {
	p, ok := e.Project()
	if !ok {
		return nil, nil, nil, 2
	}
	c, err := backlog.New(p, "")
	if err != nil {
		return nil, nil, nil, e.Fail(err)
	}
	is, code := showStory(e, c, id)
	if is == nil {
		return nil, nil, nil, code
	}
	in, ok = pm.ParseIntake(is)
	if !ok {
		return nil, nil, nil, e.Usage("%s is not an intake item: it has no machinery:intake line", id)
	}
	return p, c, in, 0
}

// intakeState runs the stages the intake steps read and reads the backlog.
func intakeState(p *project.Project, c *backlog.Client) (*pm.DesignState, []backlog.Issue, error) {
	ctx := &stages.Context{P: p}
	lint := stages.Run(ctx, stages.LintStages)
	drift := stages.RunOne(ctx, "gen-drift")
	ds, err := pm.NewDesignState(ctx.Design(), lint, drift)
	if err != nil {
		return nil, nil, err
	}
	issues, err := c.List()
	if err != nil {
		return nil, nil, err
	}
	return ds, issues, nil
}

// statusIntakes prints the summary line of every open intake item, when the
// backlog can be read (section 14).
func statusIntakes(e *Env, p *project.Project) {
	if p.Cfg.Backlog == nil {
		return
	}
	c, err := backlog.New(p, "")
	if err != nil {
		return
	}
	issues, err := c.List()
	if err != nil {
		return
	}
	open := pm.OpenIntakes(issues)
	if len(open) == 0 {
		return
	}
	ds, _, err := intakeState(p, c)
	if err != nil {
		return
	}
	for _, in := range open {
		fmt.Fprintln(e.Stdout, ds.Assess(in, issues).Summary())
	}
}
