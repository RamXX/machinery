package pm

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/machine"
)

// Story is one story of a plan (section 15.2).
type Story struct {
	Key         string
	Title       string
	Tier        string
	IDs         []string
	Priority    int
	Labels      []string
	Description string
	Acceptance  string
	needs       []need
	capstone    bool
	design      bool // the design story of an intake (section 15.7)
}

type need struct {
	key, title, output string
}

// Plan is the stories and edges for the owed ids.
type Plan struct {
	Stories []*Story // sorted by title
	Edges   []backlog.Edge
}

// Edge outputs (section 15.2).
const (
	needModel   = "the formal model the tier requires"
	needInvoke  = "the invoke path the failure is induced on"
	needReplay  = "the transitions the replay drives"
	storyType   = "task"
	internalEvt = "internal transitions"
)

// BuildPlan groups the owed ids into stories and derives the edges.
func BuildPlan(d *design.Design) (*Plan, error) {
	if len(d.OwedProblems) > 0 {
		return nil, fmt.Errorf("%s", d.OwedProblems[0])
	}
	var ids []string
	for _, o := range d.Owed {
		if d.Known(o.ID) {
			ids = append(ids, o.ID)
		}
	}
	return BuildPlanFor(d, ids, nil), nil
}

// BuildPlanFor groups the given ids into stories and derives the edges. For
// an intake every story also carries the intake's intent and tag (section
// 15.7).
func BuildPlanFor(d *design.Design, ids []string, in *Intake) *Plan {
	byTitle := map[string]*Story{}
	ids = append([]string(nil), ids...)
	sort.Strings(ids)
	for _, id := range ids {
		title := groupTitle(d, id)
		s := byTitle[title]
		if s == nil {
			s = &Story{Title: title}
			byTitle[title] = s
		}
		s.IDs = append(s.IDs, id)
	}
	plan := &Plan{}
	keys := map[string]bool{}
	for _, s := range byTitle {
		plan.Stories = append(plan.Stories, s)
	}
	sort.Slice(plan.Stories, func(i, j int) bool { return plan.Stories[i].Title < plan.Stories[j].Title })
	for _, s := range plan.Stories {
		s.Key = uniqueKey(slug(s.Title), keys)
		s.Tier, _ = StoryTier(d, &backlog.Issue{}, s.IDs)
		s.Priority = Priority(s.Tier)
		s.Labels = []string{"machinery", "tier:" + s.Tier}
	}
	plan.edges(d)
	for _, s := range plan.Stories {
		s.Description = description(d, s, in)
		s.Acceptance = acceptance(d, s)
	}
	return plan
}

// groupTitle is the title of the story an owed id belongs to.
func groupTitle(d *design.Design, id string) string {
	kind, name, _ := strings.Cut(id, ":")
	switch kind {
	case "tc":
		if m, event := caseGroup(d, id); m != nil {
			if event == "" {
				return "Prove " + m.ID + " " + internalEvt
			}
			return "Prove " + m.ID + " " + event
		}
	case "fv":
		return fmt.Sprintf("Model %s (%s)", name, areaTools(d, name))
	case "act":
		entity, _, _ := strings.Cut(name, ".")
		return "Serve " + entity + " actions"
	}
	return "Prove " + id
}

// caseGroup returns the machine of a tc id and its event when the case is
// about a caller event (an on transition, an ignore, a clause of an on
// transition, a refusal whose handler names the event), else "".
func caseGroup(d *design.Design, id string) (*machine.Machine, string) {
	m := d.MachineOfCase(id)
	if m == nil {
		return nil, ""
	}
	t := m.Table
	byID := map[string]machine.Transition{}
	for _, tr := range t.Transitions {
		byID[tr.ID] = tr
	}
	if tr, ok := byID[id]; ok {
		return m, onEvent(tr)
	}
	for _, ig := range t.Ignores {
		if ig.ID == id {
			return m, ig.Event
		}
	}
	for _, cl := range t.Clauses {
		if cl.ID == id {
			return m, onEvent(byID[cl.Transition])
		}
	}
	for _, rf := range t.Refusals {
		if rf.ID == id {
			return m, handlerEvent(rf.Handler)
		}
	}
	return m, ""
}

// handlerEvent returns the event a refusal handler names, or "" for a
// handler with no caller event.
func handlerEvent(h string) string {
	if strings.Contains(h, ":") || h == "always" || h == "stateDone" || h == "onDone" || h == "onError" {
		return ""
	}
	return h
}

func onEvent(tr machine.Transition) string {
	if tr.Kind == "on" {
		return tr.Event
	}
	return ""
}

// areaTools names the tools that answer an area's questions (section 16.1).
func areaTools(d *design.Design, area string) string {
	set := map[string]bool{}
	for _, a := range d.Areas {
		if a.Name == area {
			for _, q := range a.Questions {
				if t := design.Questions[q]; t != "" {
					set[t] = true
				}
			}
		}
	}
	if len(set) == 0 {
		return "none"
	}
	return strings.Join(sortedKeys(set), ", ")
}

func (plan *Plan) edges(d *design.Design) {
	byID := map[string]*Story{}
	byTitle := map[string]*Story{}
	for _, s := range plan.Stories {
		byTitle[s.Title] = s
		for _, id := range s.IDs {
			byID[id] = s
		}
	}
	type pair struct{ from, to string }
	seen := map[pair]bool{}
	add := func(from, to *Story, output string) {
		if from == nil || to == nil || from == to || seen[pair{from.Key, to.Key}] {
			return
		}
		seen[pair{from.Key, to.Key}] = true
		from.needs = append(from.needs, need{to.Key, to.Title, output})
		plan.Edges = append(plan.Edges, backlog.Edge{FromKey: from.Key, ToKey: to.Key, Type: "blocks"})
	}
	for _, s := range plan.Stories {
		if !strings.HasPrefix(s.Title, "Prove ") {
			continue
		}
		for _, area := range CriticalAreas(d, s.IDs) {
			add(s, byID["fv:"+area], needModel)
		}
		for _, id := range s.IDs {
			switch {
			case strings.HasPrefix(id, "fm:"):
				if mid := invokeMachine(d, strings.TrimPrefix(id, "fm:")); mid != "" {
					add(s, byTitle["Prove "+mid+" "+internalEvt], needInvoke)
				}
			case strings.HasPrefix(id, "cx:"):
				for _, c := range traceCases(d, id) {
					add(s, byID[c], needReplay)
				}
			}
		}
	}
	sortEdges(plan.Edges)
	for _, s := range plan.Stories {
		sort.Slice(s.needs, func(i, j int) bool { return s.needs[i].title < s.needs[j].title })
	}
}

// sortEdges orders edges by their ends, keys before ids.
func sortEdges(es []backlog.Edge) {
	sort.Slice(es, func(i, j int) bool {
		a, b := es[i], es[j]
		ka := []string{a.FromKey, a.FromID, a.ToKey, a.ToID}
		kb := []string{b.FromKey, b.FromID, b.ToKey, b.ToID}
		for n := range ka {
			if ka[n] != kb[n] {
				return ka[n] < kb[n]
			}
		}
		return false
	})
}

// invokeMachine returns the machine id of a failure mode with an invoke
// subject, or "".
func invokeMachine(d *design.Design, fmID string) string {
	for _, fm := range d.FailureModes {
		if fm.ID == fmID {
			if rest, ok := strings.CutPrefix(fm.Subject, "invoke:"); ok {
				mid, _, _ := strings.Cut(rest, ".")
				return mid
			}
		}
	}
	return ""
}

// traceCases lists the tc ids the steps of a counterexample trace name.
func traceCases(d *design.Design, cxID string) []string {
	for _, t := range d.Traces {
		if t.ID != cxID {
			continue
		}
		data, err := d.P.Read(t.Path)
		if err != nil {
			return nil
		}
		var tf struct {
			Steps []struct {
				Case string `json:"case"`
			} `json:"steps"`
		}
		if json.Unmarshal(data, &tf) != nil {
			return nil
		}
		set := map[string]bool{}
		for _, st := range tf.Steps {
			if st.Case != "" {
				set[st.Case] = true
			}
		}
		return sortedKeys(set)
	}
	return nil
}

// IDTitle is the title of a design id for descriptions and briefs.
func IDTitle(d *design.Design, id string) string {
	if o := d.Obligation(id); o != nil {
		return o.Title
	}
	if d.IsAct(id) {
		entity, action, _ := strings.Cut(strings.TrimPrefix(id, "act:"), ".")
		return fmt.Sprintf("action %s of %s", action, entity)
	}
	return "(unknown id)"
}

// excludeDesign is the exclusion every story carries (section 15.2).
const excludeDesign = "- No design change: a design change is its own story."

// description renders the story text of section 15.2.
func description(d *design.Design, s *Story, in *Intake) string {
	var acc []string
	for _, id := range s.IDs {
		acc = append(acc, "- "+id+": "+observed(d, id))
	}
	tags := []string{"machinery:ids " + strings.Join(s.IDs, " ")}
	return storyText(storyIntent(d, s), acc, tags, []string{excludeDesign}, s.needs, in)
}

// storyText writes the five sections of a story description.
func storyText(intent string, acceptance, tags, excludes []string, needs []need, in *Intake) string {
	var b strings.Builder
	b.WriteString("Intent:\n" + intent + "\n")
	if in != nil {
		fmt.Fprintf(&b, "Requirement %s: %s\n", in.ID, in.Intent)
	}
	b.WriteString("\nAcceptance:\n")
	for _, l := range acceptance {
		b.WriteString(l + "\n")
	}
	b.WriteString("\nIds:\n")
	for _, t := range tags {
		b.WriteString(t + "\n")
	}
	if in != nil {
		b.WriteString("machinery:intake " + in.ID + "\n")
	}
	b.WriteString("\nExcludes:\n")
	for _, x := range excludes {
		b.WriteString(x + "\n")
	}
	b.WriteString("\nNeeds:\n")
	if len(needs) == 0 {
		b.WriteString("- none\n")
	}
	for _, n := range needs {
		fmt.Fprintf(&b, "- %s: %s\n", n.title, n.output)
	}
	return b.String()
}

// storyIntent is the intent line of a generated story (section 15.2).
func storyIntent(d *design.Design, s *Story) string {
	title := s.Title
	switch {
	case strings.HasPrefix(title, "Model "):
		area, _, _ := strings.Cut(strings.TrimPrefix(title, "Model "), " ")
		return "The formal model of " + area + " answers its questions and is proven."
	case strings.HasPrefix(title, "Serve "):
		entity := strings.TrimSuffix(strings.TrimPrefix(title, "Serve "), " actions")
		return "Code serves the actions of " + entity + " through their operations."
	case strings.HasSuffix(title, " "+internalEvt):
		m := strings.TrimSuffix(strings.TrimPrefix(title, "Prove "), " "+internalEvt)
		return "Machine " + m + " makes its internal transitions on the running system as its table says."
	}
	id := strings.TrimPrefix(title, "Prove ")
	switch kind, _, _ := strings.Cut(id, ":"); kind {
	case "sc":
		return "The scenario runs end to end on the running system."
	case "inv":
		return "The invariant holds on the running system."
	case "fm":
		return "The failure gets its required behavior on the running system."
	case "cx":
		return "The counterexample of the formal model cannot happen on the running system."
	case "ck":
		return "The external checker holds the obligation for this design."
	case "tc":
		return "The transition case holds on the running system."
	}
	m, event, _ := strings.Cut(id, " ")
	return "Machine " + m + " answers " + event + " on the running system as its table says."
}

// observed is the acceptance line of one id: what is observed, and for a
// refusal case what is refused (section 15.2).
func observed(d *design.Design, id string) string {
	kind, name, _ := strings.Cut(id, ":")
	switch kind {
	case "tc":
		return caseObserved(d, id)
	case "inv":
		return "holds on the running system: " + IDTitle(d, id)
	case "sc":
		return "runs end to end on the running system: " + IDTitle(d, id)
	case "fm":
		for _, fm := range d.FailureModes {
			if fm.ID == name {
				if fm.Required == "refuse" {
					return fmt.Sprintf("refused: %s Required behavior: %s", fm.Failure, fm.Behavior)
				}
				return fmt.Sprintf("%s Required behavior (%s): %s", fm.Failure, fm.Required, fm.Behavior)
			}
		}
	case "fv":
		for _, a := range d.Areas {
			if a.Name == name {
				return "machinery formal check proves the model: " + a.Why
			}
		}
	case "cx":
		for _, t := range d.Traces {
			if t.ID == id {
				return fmt.Sprintf("refused on the running system: the replay of %s run %s keeps %s", t.Spec, t.Run, t.Violates)
			}
		}
	case "ck":
		return "the checker's evidence for this design holds it: " + IDTitle(d, id)
	case "act":
		entity, action, _ := strings.Cut(name, ".")
		return fmt.Sprintf("served by code and reachable through its operation: action %s of %s", action, entity)
	}
	return IDTitle(d, id)
}

// caseObserved is the acceptance line of a transition case.
func caseObserved(d *design.Design, id string) string {
	m := d.MachineOfCase(id)
	if m == nil {
		return IDTitle(d, id)
	}
	t := m.Table
	byID := map[string]machine.Transition{}
	for _, tr := range t.Transitions {
		byID[tr.ID] = tr
	}
	for _, ig := range t.Ignores {
		if ig.ID == id {
			return fmt.Sprintf("refused: %s in %s changes nothing and has no effect (%s)", ig.Event, ig.State, ig.Reason)
		}
	}
	for _, rf := range t.Refusals {
		if rf.ID == id {
			return fmt.Sprintf("refused: in %s, with every guard of %s false, nothing changes and no effect happens", rf.State, rf.Handler)
		}
	}
	for _, cl := range t.Clauses {
		if cl.ID == id {
			tr := byID[cl.Transition]
			return fmt.Sprintf("refused: with only %s of %s false, %s %s does not fire", cl.Clause, cl.Guard, tr.Source, machine.Trigger(tr.Kind, tr.Event))
		}
	}
	effects := "none"
	if o := d.Obligation(id); o != nil && len(o.Effects) > 0 {
		effects = strings.Join(o.Effects, ", ")
	}
	return fmt.Sprintf("%s, effects: %s; observed on the running system", IDTitle(d, id), effects)
}

func acceptance(d *design.Design, s *Story) string {
	acts, obligations := 0, 0
	for _, id := range s.IDs {
		if d.IsAct(id) {
			acts++
		} else {
			obligations++
		}
	}
	a := ""
	if obligations > 0 {
		a += "- Each listed obligation is proven by machinery verify at the story's commit and removed from owed.yaml.\n"
	}
	if acts > 0 {
		a += "- Each listed action is served by code (stage trace) and removed from owed.yaml.\n"
	}
	if s.Tier != design.Routine {
		a += "- machinery story passes for this story.\n"
	}
	return a
}

// Markdown writes the plan as bd create -f Markdown.
func (plan *Plan) Markdown(w io.Writer) {
	for i, s := range plan.Stories {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "## %s\n\n### Type\n%s\n\n### Priority\n%d\n\n### Labels\n%s\n\n### Description\n%s\n### Acceptance Criteria\n%s",
			s.Title, storyType, s.Priority, strings.Join(s.Labels, ", "), s.Description, s.Acceptance)
	}
}

// Graph renders the stories as a bd create --graph plan. keep selects the
// stories to include (nil: all); edges are kept only between included ones.
func (plan *Plan) Graph(keep map[string]bool, parent string) backlog.Graph {
	g := backlog.Graph{Nodes: []backlog.Node{}, Edges: []backlog.Edge{}}
	for _, s := range plan.Stories {
		if keep != nil && !keep[s.Key] {
			continue
		}
		g.Nodes = append(g.Nodes, backlog.Node{Key: s.Key, Title: s.Title, Type: storyType, Priority: s.Priority,
			Labels: s.Labels, Description: s.Description, Acceptance: s.Acceptance, ParentID: parent})
	}
	kept := func(key string) bool { return key == "" || keep[key] }
	for _, e := range plan.Edges {
		if keep == nil || (kept(e.FromKey) && kept(e.ToKey)) {
			g.Edges = append(g.Edges, e)
		}
	}
	return g
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func uniqueKey(k string, used map[string]bool) string {
	key := k
	for i := 2; used[key]; i++ {
		key = fmt.Sprintf("%s-%d", k, i)
	}
	used[key] = true
	return key
}

// ApplyResult counts what Apply did.
type ApplyResult struct{ Created, Existing, Edges int }

// Apply creates the stories that are not linked to an existing story yet and
// writes the new story ids into the owed list (section 15.2).
func Apply(c *backlog.Client, d *design.Design, plan *Plan, parent string) (ApplyResult, error) {
	var res ApplyResult
	issues, err := c.List()
	if err != nil {
		return res, err
	}
	exists := map[string]bool{}
	for _, is := range issues {
		exists[is.ID] = true
	}
	owed := d.OwedSet()
	keep := map[string]bool{}
	for _, s := range plan.Stories {
		linked := true
		for _, id := range s.IDs {
			if st := owed[id].Story; st == "" || !exists[st] {
				linked = false
			}
		}
		if linked {
			res.Existing++
			continue
		}
		keep[s.Key] = true
	}
	if len(keep) == 0 {
		return res, nil
	}
	g := plan.Graph(keep, parent)
	ids, err := c.CreateGraph(g)
	if err != nil {
		return res, err
	}
	res.Created, res.Edges = len(g.Nodes), len(g.Edges)
	storyOf := map[string]string{}
	for _, s := range plan.Stories {
		if keep[s.Key] {
			if ids[s.Key] == "" {
				return res, fmt.Errorf("bd create --graph returned no id for %s", s.Title)
			}
			for _, id := range s.IDs {
				storyOf[id] = ids[s.Key]
			}
		}
	}
	return res, setStories(d, storyOf)
}

// Link attaches ids to an existing story (section 15.2).
func Link(c *backlog.Client, d *design.Design, story string, ids []string) error {
	if err := c.Note(story, "machinery:ids "+strings.Join(ids, " ")); err != nil {
		return err
	}
	storyOf := map[string]string{}
	for _, id := range ids {
		storyOf[id] = story
	}
	return setStories(d, storyOf)
}

func setStories(d *design.Design, storyOf map[string]string) error {
	entries := append([]design.OwedEntry(nil), d.Owed...)
	changed := false
	for i := range entries {
		if st, ok := storyOf[entries[i].ID]; ok && entries[i].Story != st {
			entries[i].Story = st
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return design.WriteOwed(d.P, entries)
}
