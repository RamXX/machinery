package pm

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/formal"
	"github.com/RamXX/machinery/internal/gen"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
)

// IntakeLabel is the label of an intake item (section 15.7).
const IntakeLabel = "machinery-intake"

// Intake is an intake item: the record of one requirement (section 15.7).
type Intake struct {
	ID      string
	Title   string
	Intent  string
	Refuses []string
	Tags    backlog.Tags
}

// ParseIntake reads an intake item; ok is false when the issue is not one.
func ParseIntake(is *backlog.Issue) (*Intake, bool) {
	t := backlog.ParseTags(is)
	if !t.Intake {
		return nil, false
	}
	in := &Intake{ID: is.ID, Title: is.Title, Tags: t}
	in.Intent, in.Refuses = intakeText(is.Description)
	return in, true
}

// intakeText reads the Intent and Refuses sections of an intake description.
func intakeText(desc string) (intent string, refuses []string) {
	section := ""
	var lines []string
	for _, line := range strings.Split(desc, "\n") {
		trim := strings.TrimSpace(line)
		switch {
		case trim == "Intent:" || trim == "Refuses:":
			section = trim
			continue
		case trim == "" || strings.Contains(trim, "machinery:"):
			section = ""
			continue
		}
		switch section {
		case "Intent:":
			lines = append(lines, trim)
		case "Refuses:":
			if r, ok := strings.CutPrefix(trim, "- "); ok && r != "none" {
				refuses = append(refuses, r)
			}
		}
	}
	return strings.Join(lines, " "), refuses
}

// OneLine makes a value one line: a newline becomes a space.
func OneLine(s string) string {
	return strings.TrimSpace(strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(s))
}

// IntakeDescription renders the description of a new intake item.
func IntakeDescription(intent string, refuses []string, head, contract string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Intent:\n%s\n\nRefuses:\n", OneLine(intent))
	if len(refuses) == 0 {
		b.WriteString("- none\n")
	}
	for _, r := range refuses {
		fmt.Fprintf(&b, "- %s\n", OneLine(r))
	}
	fmt.Fprintf(&b, "\nmachinery:intake\nmachinery:intake-base %s %s\n", head, contract)
	return b.String()
}

// IntakeStories returns the stories of an intake: the items that carry
// machinery:intake <id> or are its children, sorted by id.
func IntakeStories(issues []backlog.Issue, id string) []backlog.Issue {
	var res []backlog.Issue
	for _, is := range issues {
		if is.ID == id {
			continue
		}
		if is.Parent == id || backlog.ParseTags(&is).IntakeOf == id {
			res = append(res, is)
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res
}

// IntakeOf returns the intake a story belongs to, or "".
func IntakeOf(is *backlog.Issue) string {
	if id := backlog.ParseTags(is).IntakeOf; id != "" {
		return id
	}
	return is.Parent
}

// CapstoneSiblings returns the other stories of a capstone's intake.
func CapstoneSiblings(is *backlog.Issue, issues []backlog.Issue) []backlog.Issue {
	var res []backlog.Issue
	for _, s := range IntakeStories(issues, IntakeOf(is)) {
		if s.ID != is.ID {
			res = append(res, s)
		}
	}
	return res
}

// CapstoneIDs returns the ids of a capstone: the ids of the other stories of
// its intake, sorted (section 15.6).
func CapstoneIDs(is *backlog.Issue, issues []backlog.Issue) []string {
	set := map[string]bool{}
	for _, s := range CapstoneSiblings(is, issues) {
		for _, id := range backlog.ParseTags(&s).IDs {
			set[id] = true
		}
	}
	return sortedKeys(set)
}

// IntakeSteps are the steps of an intake, in order (section 15.7).
var IntakeSteps = []string{"design changed", "design sound", "generated", "new ids", "tiers", "formal", "planned"}

// planSteps are the steps intake plan requires.
var planSteps = map[string]bool{"design changed": true, "design sound": true, "generated": true, "new ids": true, "tiers": true, "formal": true}

// Step is one step of an intake status.
type Step struct {
	Name    string `json:"step"`
	Done    bool   `json:"done"`
	Reason  string `json:"reason"`
	details []string
}

// NewID is an id the design change added, with its tier.
type NewID struct {
	ID    string   `json:"id"`
	Tier  string   `json:"tier"`
	Areas []string `json:"areas"`
}

// DesignState is what every intake of one command is judged against: the
// design at the current commit, its contract, the stage outcomes the steps
// read and the uncommitted paths.
type DesignState struct {
	D        *design.Design
	Contract *Contract
	Head     string
	Lint     []*report.Stage
	GenDrift *report.Stage
	dirty    []string
	specs    []*formal.Spec
	proven   map[string]bool
}

// NewDesignState gathers what the intake steps read. lint holds the lint
// stages and genDrift stage gen-drift, run by the caller.
func NewDesignState(d *design.Design, lint []*report.Stage, genDrift *report.Stage) (*DesignState, error) {
	head, err := gitx.Head(d.P.Root)
	if err != nil {
		return nil, fmt.Errorf("cannot read the current commit: %v", err)
	}
	dirty, err := gitx.StatusPaths(d.P.Root)
	if err != nil {
		return nil, err
	}
	specs, err := formal.Load(d.P)
	if err != nil {
		return nil, err
	}
	return &DesignState{D: d, Contract: ContractOf(d), Head: head, Lint: lint, GenDrift: genDrift,
		dirty: dirty, specs: specs, proven: formal.ProvenAreas(d)}, nil
}

// UncommittedDesign lists the design sources with uncommitted changes.
func UncommittedDesign(d *design.Design, dirty []string) []string {
	files := setOf(d.ContractFiles())
	var res []string
	for _, p := range dirty {
		if files[p] {
			res = append(res, p)
		}
	}
	sort.Strings(res)
	return res
}

// IntakeStatus is where one requirement stands (section 15.7).
type IntakeStatus struct {
	In               *Intake
	Steps            []Step
	NewIDs           []NewID
	RoutineByDefault []string
	Proposed         []string        // new ids that are only proposed (section 4.1): never planned
	Stories          []backlog.Issue // the stories of the intake, the capstone included
	Capstone         *backlog.Issue
	without          []string // new ids that no story of the intake lists
	newKnown         bool
	contract         string
}

// Assess judges the steps of one intake.
func (ds *DesignState) Assess(in *Intake, issues []backlog.Issue) *IntakeStatus {
	st := &IntakeStatus{In: in, contract: ds.Contract.Hash}
	newReason := ""
	if in.Tags.Base == nil {
		newReason = "the item has no machinery:intake-base line; record requirements with machinery intake new"
	} else {
		st.NewIDs, st.Proposed, newReason = ds.newIDs(in.Tags.Base.Commit)
		st.newKnown = newReason == ""
	}
	st.Steps = []Step{
		ds.designChanged(st),
		ds.designSound(),
		ds.generated(),
		st.newIDsStep(newReason),
		ds.tiers(st),
		ds.formal(st),
		st.planned(issues),
	}
	return st
}

func stepDone(name, reason string) Step    { return Step{Name: name, Done: true, Reason: reason} }
func stepMissing(name, reason string) Step { return Step{Name: name, Reason: reason} }

func (ds *DesignState) designChanged(st *IntakeStatus) Step {
	const name = "design changed"
	in := st.In
	if in.Tags.Base == nil {
		return stepMissing(name, "the item has no machinery:intake-base line; record requirements with machinery intake new")
	}
	if dirty := UncommittedDesign(ds.D, ds.dirty); len(dirty) > 0 {
		return stepMissing(name, "uncommitted design change: "+strings.Join(dirty, ", ")+"; commit it")
	}
	base := in.Tags.Base.Contract
	if ds.Contract.Hash != base {
		return stepDone(name, fmt.Sprintf("contract %s, base %s", ds.Contract.Short(), short12(base)))
	}
	if r := in.Tags.NoDesignChange; r != "" && st.newKnown {
		if len(st.NewIDs) == 0 {
			return stepDone(name, "no design change: "+r+"; the design already covers it")
		}
		return stepMissing(name, fmt.Sprintf("machinery:no-design-change needs a plan without new ids: %d id(s) are new", len(st.NewIDs)))
	}
	return stepMissing(name, "the contract is the base contract: change the design first, or record machinery:no-design-change <reason>")
}

func (ds *DesignState) designSound() Step {
	var parts []string
	for _, s := range ds.Lint {
		if !s.OK() {
			parts = append(parts, s.Name+": "+OutcomeLine(s))
		}
	}
	if len(parts) > 0 {
		return stepMissing("design sound", "machinery lint: "+strings.Join(parts, "; "))
	}
	return stepDone("design sound", "every lint stage passes or is not applicable")
}

func (ds *DesignState) generated() Step {
	const name = "generated"
	if !ds.GenDrift.OK() {
		return stepMissing(name, "machinery gen --check: "+OutcomeLine(ds.GenDrift)+"; run machinery gen and commit")
	}
	prefix := gen.Dir(ds.D) + "/"
	var dirty []string
	for _, p := range ds.dirty {
		if strings.HasPrefix(p, prefix) {
			dirty = append(dirty, p)
		}
	}
	sort.Strings(dirty)
	if len(dirty) > 0 {
		return stepMissing(name, "uncommitted generated file: "+strings.Join(dirty, ", ")+"; commit it")
	}
	return stepDone(name, "machinery gen --check: no drift")
}

func (st *IntakeStatus) newIDsStep(reason string) Step {
	const name = "new ids"
	if reason != "" {
		return stepMissing(name, reason)
	}
	base := gitx.Short(st.In.Tags.Base.Commit)
	s := stepDone(name, fmt.Sprintf("%d new id(s) since %s", len(st.NewIDs), base))
	if len(st.NewIDs) == 0 {
		s.Reason = "none since " + base + ": only the capstone is planned"
	}
	for _, n := range st.NewIDs {
		line := n.ID + "  " + n.Tier
		if len(n.Areas) > 0 {
			line += " (" + strings.Join(n.Areas, ", ") + ")"
		}
		s.details = append(s.details, line)
	}
	if len(st.Proposed) > 0 {
		s.details = append(s.details, "proposed, not planned: "+strings.Join(st.Proposed, " "))
	}
	return s
}

func (ds *DesignState) tiers(st *IntakeStatus) Step {
	const name = "tiers"
	for _, n := range st.NewIDs {
		if ds.D.IsAct(n.ID) && len(n.Areas) == 0 {
			st.RoutineByDefault = append(st.RoutineByDefault, n.ID)
		}
	}
	var s Step
	risk := ds.stage("risk")
	switch {
	case risk == nil || risk.Outcome() == report.NotApplicable:
		s = stepDone(name, "no design.risk configured: every new id is routine")
	case risk.Outcome() == report.Passed:
		s = stepDone(name, "stage risk passed")
	default:
		s = stepMissing(name, "stage risk: "+OutcomeLine(risk)+"; run machinery lint --stage risk")
	}
	if len(st.RoutineByDefault) > 0 {
		s.details = []string{"routine by default: " + strings.Join(st.RoutineByDefault, " ")}
	}
	return s
}

func (ds *DesignState) stage(name string) *report.Stage {
	for _, s := range ds.Lint {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func (ds *DesignState) formal(st *IntakeStatus) Step {
	const name = "formal"
	var ids []string
	for _, n := range st.NewIDs {
		ids = append(ids, n.ID)
	}
	areas := CriticalAreas(ds.D, ids)
	if len(areas) == 0 {
		return stepDone(name, "no critical area holds a new id")
	}
	var gaps []string
	for _, a := range ds.D.Areas {
		if !setOf(areas)[a.Name] || ds.proven["fv:"+a.Name] {
			continue
		}
		gaps = append(gaps, formal.Unanswered(ds.specs, a)...)
	}
	if len(gaps) > 0 {
		sort.Strings(gaps)
		return stepMissing(name, strings.Join(gaps, "; ")+"; scaffold it with machinery formal new")
	}
	return stepDone(name, "modeled: "+strings.Join(areas, ", "))
}

func (st *IntakeStatus) planned(issues []backlog.Issue) Step {
	const name = "planned"
	st.Stories = IntakeStories(issues, st.In.ID)
	covered := map[string]bool{}
	for i := range st.Stories {
		t := backlog.ParseTags(&st.Stories[i])
		if t.Capstone {
			if st.Capstone == nil {
				st.Capstone = &st.Stories[i]
			}
			continue
		}
		for _, id := range t.IDs {
			covered[id] = true
		}
	}
	for _, n := range st.NewIDs {
		if !covered[n.ID] {
			st.without = append(st.without, n.ID)
		}
	}
	run := "run machinery intake plan " + st.In.ID + " --apply"
	switch {
	case !st.newKnown:
		return stepMissing(name, "the new ids are not known; "+run)
	case len(st.without) > 0:
		return stepMissing(name, fmt.Sprintf("%d new id(s) without a story; %s", len(st.without), run))
	case st.Capstone == nil:
		return stepMissing(name, "no capstone; "+run)
	}
	return stepDone(name, "every new id has a story; capstone "+st.Capstone.ID)
}

// newIDs lists the ids now that the generated obligation list at the base
// commit does not hold, and apart from them the proposed ids that are new the
// same way and were not proposed at the base either. reason is set when the
// base cannot be read.
func (ds *DesignState) newIDs(commit string) ([]NewID, []string, string) {
	d := ds.D
	root := d.P.Root
	if !gitx.Exists(root, commit) {
		return nil, nil, fmt.Sprintf("base commit %s is not in this repository", gitx.Short(commit))
	}
	had := map[string]bool{}
	genDir, statusFile := d.P.Cfg.Design.Gen, d.P.Cfg.Design.Status
	if raw, ok := gitx.Show(root, commit, project.ConfigName); ok {
		if bcfg, err := project.Parse(raw); err == nil {
			genDir, statusFile = bcfg.Design.Gen, bcfg.Design.Status
		}
	}
	wasProposed := map[string]bool{}
	if data, ok := gitx.Show(root, commit, statusFile); ok {
		l, _ := design.ParseStatus(statusFile, data)
		for _, e := range l.Proposed {
			for _, id := range d.DeclaredIDs() {
				if design.MatchID(strings.TrimSpace(e.ID), id) {
					wasProposed[id] = true
				}
			}
		}
	}
	if data, ok := gitx.Show(root, commit, strings.TrimSuffix(genDir, "/")+"/obligations.json"); ok {
		var inv struct {
			Obligations []struct {
				ID string `json:"id"`
			} `json:"obligations"`
			Served []struct {
				ID string `json:"id"`
			} `json:"served"`
		}
		if err := json.Unmarshal(data, &inv); err != nil {
			return nil, nil, fmt.Sprintf("obligations.json at %s cannot be read: %v", gitx.Short(commit), err)
		}
		for _, o := range inv.Obligations {
			had[o.ID] = true
		}
		for _, a := range inv.Served {
			had[a.ID] = true
		}
	}
	var res []NewID
	for _, id := range d.AllIDs() {
		if had[id] {
			continue
		}
		tier, in := d.Tier(id)
		areas := []string{}
		for _, a := range in {
			areas = append(areas, a.Name)
		}
		sort.Strings(areas)
		res = append(res, NewID{ID: id, Tier: tier, Areas: areas})
	}
	var proposed []string
	for _, id := range d.WithStatus(design.Proposed) {
		if !had[id] && !wasProposed[id] {
			proposed = append(proposed, id)
		}
	}
	return res, proposed, ""
}

func short12(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

// OutcomeLine is the last line of a stage block.
func OutcomeLine(s *report.Stage) string { return StageDoc(s).Outcome }

// Done counts the steps done.
func (st *IntakeStatus) Done() int {
	n := 0
	for _, s := range st.Steps {
		if s.Done {
			n++
		}
	}
	return n
}

// Next is the first missing step, or "".
func (st *IntakeStatus) Next() string {
	for _, s := range st.Steps {
		if !s.Done {
			return s.Name
		}
	}
	return ""
}

// Summary is the last line of intake status.
func (st *IntakeStatus) Summary() string {
	line := fmt.Sprintf("intake %s: %d of %d steps done", st.In.ID, st.Done(), len(st.Steps))
	if next := st.Next(); next != "" {
		line += "; next: " + next
	}
	return line
}

// Print writes the status and returns the exit code.
func (st *IntakeStatus) Print(w io.Writer) int {
	fmt.Fprintf(w, "intake %s: %s\n", st.In.ID, st.In.Title)
	if b := st.In.Tags.Base; b != nil {
		fmt.Fprintf(w, "base: %s, contract %s\n", gitx.Short(b.Commit), short12(b.Contract))
	} else {
		fmt.Fprintln(w, "base: none")
	}
	for _, s := range st.Steps {
		state := "missing"
		if s.Done {
			state = "done"
		}
		fmt.Fprintf(w, "  %-14s  %-7s  %s\n", s.Name, state, s.Reason)
		for _, l := range s.details {
			fmt.Fprintf(w, "    %s\n", l)
		}
	}
	fmt.Fprintln(w, st.Summary())
	return st.exit()
}

func (st *IntakeStatus) exit() int {
	if st.Next() != "" {
		return 1
	}
	return 0
}

// JSON writes the status as one document and returns the exit code.
func (st *IntakeStatus) JSON(w io.Writer) int {
	type base struct {
		Commit   string `json:"commit"`
		Contract string `json:"contract"`
	}
	doc := struct {
		Intake           string   `json:"intake"`
		Title            string   `json:"title"`
		Base             *base    `json:"base"`
		Contract         string   `json:"contract"`
		Steps            []Step   `json:"steps"`
		NewIDs           []NewID  `json:"new_ids"`
		Proposed         []string `json:"proposed"`
		RoutineByDefault []string `json:"routine_by_default"`
		Done             int      `json:"done"`
		Next             string   `json:"next"`
	}{Intake: st.In.ID, Title: st.In.Title, Contract: st.contract, Steps: st.Steps,
		NewIDs: append([]NewID{}, st.NewIDs...), Proposed: append([]string{}, st.Proposed...),
		RoutineByDefault: append([]string{}, st.RoutineByDefault...),
		Done:             st.Done(), Next: st.Next()}
	if b := st.In.Tags.Base; b != nil {
		doc.Base = &base{b.Commit, b.Contract}
	}
	w.Write(JSON(doc))
	return st.exit()
}

// Refusals lists the steps intake plan requires that are missing, as
// messages (section 15.7).
func (st *IntakeStatus) Refusals() []string {
	var res []string
	for _, s := range st.Steps {
		if planSteps[s.Name] && !s.Done {
			res = append(res, fmt.Sprintf("cannot plan intake %s: %s is missing: %s", st.In.ID, s.Name, s.Reason))
		}
	}
	return res
}

// Capstone texts (section 15.7).
const (
	capstoneIntent  = "Show the requirement on the running system through its real interface."
	capstoneExclude = "- No new behavior: the capstone shows what the other stories built."
	capstoneNeed    = "the behavior shown"
)

// Plan builds the stories an intake still owes: one per group of new ids
// without a story, and the capstone when the intake has none (section 15.7).
func (st *IntakeStatus) Plan(d *design.Design) *Plan {
	plan := BuildPlanFor(d, st.without, st.In)
	if st.Capstone != nil {
		for _, s := range plan.Stories {
			plan.Edges = append(plan.Edges, backlog.Edge{FromID: st.Capstone.ID, ToKey: s.Key, Type: "blocks"})
		}
		sortEdges(plan.Edges)
		return plan
	}
	used := map[string]bool{}
	for _, s := range plan.Stories {
		used[s.Key] = true
	}
	in := st.In
	c := &Story{Title: "Demonstrate: " + in.Title, capstone: true}
	c.Key = uniqueKey(slug(c.Title), used)
	tier := design.Consequential
	for _, n := range st.NewIDs {
		if design.TierRank(n.Tier) > design.TierRank(tier) {
			tier = n.Tier
		}
	}
	c.Tier, c.Priority = tier, Priority(tier)
	c.Labels = []string{"machinery", "tier:" + tier}
	for _, s := range plan.Stories {
		c.needs = append(c.needs, need{s.Key, s.Title, capstoneNeed})
		plan.Edges = append(plan.Edges, backlog.Edge{FromKey: c.Key, ToKey: s.Key, Type: "blocks"})
	}
	for _, s := range st.Stories {
		c.needs = append(c.needs, need{"", s.Title, capstoneNeed})
		plan.Edges = append(plan.Edges, backlog.Edge{FromKey: c.Key, ToID: s.ID, Type: "blocks"})
	}
	sort.SliceStable(c.needs, func(i, j int) bool { return c.needs[i].title < c.needs[j].title })
	sortEdges(plan.Edges)
	acc := []string{"- The intent is observed: " + in.Intent}
	crit := []string{"- The intent is shown on the running system through its real interface: " + in.Intent}
	for _, r := range in.Refuses {
		acc = append(acc, "- Refused: "+r)
		crit = append(crit, "- Refused on the running system: "+r)
	}
	c.Description = storyText(capstoneIntent, acc, []string{"machinery:capstone"}, []string{capstoneExclude, excludeDesign}, c.needs, in)
	c.Acceptance = strings.Join(append(crit,
		"- The commands that show it are recorded on this story, one machinery:demo <command> line each.",
		"- Every other story of the intake is closed.",
		"- machinery story passes for this story."), "\n") + "\n"
	plan.Stories = append(plan.Stories, c)
	return plan
}

// IntakeResult counts what ApplyIntake did.
type IntakeResult struct {
	Created, Existing, Edges, Owed int
	Planned                        bool
}

// ApplyIntake creates the plan's stories and edges in one backlog call, then
// owes their ids and records the planned line (section 15.7). Nothing is
// written when the backlog call fails.
func ApplyIntake(c *backlog.Client, ds *DesignState, st *IntakeStatus, plan *Plan) (IntakeResult, error) {
	res := IntakeResult{Existing: len(st.Stories)}
	d := ds.D
	if len(plan.Stories) > 0 {
		g := plan.Graph(nil, st.In.ID)
		ids, err := c.CreateGraph(g)
		if err != nil {
			return res, err
		}
		res.Created, res.Edges = len(g.Nodes), len(g.Edges)
		entries := append([]design.OwedEntry(nil), d.Owed...)
		at := map[string]int{}
		for i, e := range entries {
			at[e.ID] = i
		}
		for _, s := range plan.Stories {
			if s.capstone {
				continue
			}
			story := ids[s.Key]
			if story == "" {
				return res, fmt.Errorf("bd create --graph returned no id for %s", s.Title)
			}
			for _, id := range s.IDs {
				if i, ok := at[id]; ok {
					entries[i].Story = story
					continue
				}
				at[id] = len(entries)
				entries = append(entries, design.OwedEntry{ID: id, Reason: "owed by intake " + st.In.ID, Story: story})
				res.Owed++
			}
		}
		if err := design.WriteOwed(d.P, entries); err != nil {
			return res, err
		}
	}
	for _, m := range st.In.Tags.Planned {
		if m.Contract == ds.Contract.Hash {
			return res, nil
		}
	}
	if err := c.Note(st.In.ID, fmt.Sprintf("machinery:intake-planned %s %s", ds.Head, ds.Contract.Hash)); err != nil {
		return res, err
	}
	res.Planned = true
	return res, nil
}

// IntakeFindings are the warnings of stage intake for one open intake item
// (section 15.7).
func IntakeFindings(in *Intake, contract string, issues []backlog.Issue) []string {
	var res []string
	if b := in.Tags.Base; b != nil && b.Contract != contract {
		planned := false
		for _, m := range in.Tags.Planned {
			planned = planned || m.Contract == contract
		}
		if !planned {
			res = append(res, fmt.Sprintf("intake %s: the design changed and no stories were planned from it; run machinery intake plan %s", in.ID, in.ID))
		}
	}
	stories := IntakeStories(issues, in.ID)
	capstone, open := false, false
	for i := range stories {
		if backlog.ParseTags(&stories[i]).Capstone && stories[i].Status == "closed" {
			capstone = true
		}
		if stories[i].Status != "closed" {
			open = true
		}
	}
	if capstone && !open {
		res = append(res, fmt.Sprintf("intake %s is delivered and still open; close it", in.ID))
	}
	return res
}

// OpenIntakes returns the open intake items, sorted by id.
func OpenIntakes(issues []backlog.Issue) []*Intake {
	var res []*Intake
	for i := range issues {
		if !issues[i].Open() {
			continue
		}
		if in, ok := ParseIntake(&issues[i]); ok {
			res = append(res, in)
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res
}
