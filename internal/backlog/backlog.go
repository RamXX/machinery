// Package backlog is the only part of machinery that talks to the backlog
// tool. It runs the real bd executable and reads its JSON
// (docs/formats.md section 15.1).
package backlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/v2/internal/gitx"
	"github.com/RamXX/machinery/v2/internal/project"
)

// Issue is a backlog story as machinery sees it.
type Issue struct {
	ID          string
	Title       string
	Description string
	Notes       string
	Status      string
	Type        string
	Priority    int
	Labels      []string
	Parent      string
	Blockers    []string // ids this issue is blocked by
}

// Open reports whether the issue is neither closed nor deferred.
func (i *Issue) Open() bool { return i.Status != "closed" && i.Status != "deferred" }

// Client runs bd in the project directory.
type Client struct {
	p     *project.Project
	exe   string
	Actor string
}

// New returns a client. It is a config error when the project has no backlog
// configured or bd cannot be run.
func New(p *project.Project, actor string) (*Client, error) {
	if p.Cfg.Backlog == nil {
		return nil, project.Errf("no backlog configured: set backlog.tool: bd in %s", project.ConfigName)
	}
	c := &Client{p: p, exe: p.Cfg.Tools.BD, Actor: ResolveActor(p, actor)}
	if out, err := exec.Command(c.exe, "--version").CombinedOutput(); err != nil {
		reason := err.Error()
		if msg := strings.TrimSpace(string(out)); msg != "" {
			reason = firstLine(msg) + ": " + reason
		}
		return nil, project.Errf("bd not runnable: %s", reason)
	}
	return c, nil
}

// ResolveActor implements the actor rule of section 15.1.
func ResolveActor(p *project.Project, flag string) string {
	if flag != "" {
		return flag
	}
	if env := os.Getenv("BEADS_ACTOR"); env != "" {
		return env
	}
	return gitx.UserName(p.Root)
}

func (c *Client) run(args ...string) ([]byte, error) {
	cmd := exec.Command(c.exe, args...)
	cmd.Dir = c.p.Root
	cmd.Env = append(os.Environ(), "BEADS_ACTOR="+c.Actor, "BD_NON_INTERACTIVE=1")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = strings.TrimSpace(out.String())
		}
		return nil, fmt.Errorf("bd %s: %s", args[0], firstLine(msg))
	}
	return out.Bytes(), nil
}

type rawIssue struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Notes        string   `json:"notes"`
	Status       string   `json:"status"`
	IssueType    string   `json:"issue_type"`
	Priority     int      `json:"priority"`
	Labels       []string `json:"labels"`
	Parent       string   `json:"parent"`
	Dependencies []struct {
		IssueID     string `json:"issue_id"`
		DependsOnID string `json:"depends_on_id"`
		Type        string `json:"type"`
		// bd show reports dependencies as full issues with a dependency_type
		ID             string `json:"id"`
		DependencyType string `json:"dependency_type"`
	} `json:"dependencies"`
}

func (r rawIssue) issue() Issue {
	is := Issue{ID: r.ID, Title: r.Title, Description: r.Description, Notes: r.Notes, Status: r.Status,
		Type: r.IssueType, Priority: r.Priority, Labels: r.Labels, Parent: r.Parent}
	for _, d := range r.Dependencies {
		switch {
		case d.Type == "blocks" && d.DependsOnID != "":
			is.Blockers = append(is.Blockers, d.DependsOnID)
		case d.DependencyType == "blocks" && d.ID != "":
			is.Blockers = append(is.Blockers, d.ID)
		}
	}
	sort.Strings(is.Blockers)
	sort.Strings(is.Labels)
	return is
}

// List returns every issue, closed ones included, sorted by id, with the
// notes bd list reports.
func (c *Client) List() ([]Issue, error) {
	out, err := c.run("list", "--all", "--json", "--limit", "0")
	if err != nil {
		return nil, err
	}
	var raws []rawIssue
	if err := json.Unmarshal(out, &raws); err != nil {
		return nil, fmt.Errorf("bd list: unreadable JSON: %v", err)
	}
	res := make([]Issue, 0, len(raws))
	for _, r := range raws {
		res = append(res, r.issue())
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res, nil
}

// Show returns one issue with its notes.
func (c *Client) Show(id string) (*Issue, error) {
	out, err := c.run("show", id, "--json")
	if err != nil {
		return nil, err
	}
	var raws []rawIssue
	if err := json.Unmarshal(out, &raws); err != nil || len(raws) == 0 {
		return nil, fmt.Errorf("bd show %s: unreadable JSON", id)
	}
	is := raws[0].issue()
	return &is, nil
}

// Note appends a note to an issue.
func (c *Client) Note(id, text string) error {
	_, err := c.run("update", id, "--append-notes", text)
	return err
}

// Supersede closes an issue as superseded by a newer one, with bd's link.
func (c *Client) Supersede(old, by string) error {
	_, err := c.run("supersede", old, "--with", by)
	return err
}

// Close closes an issue with a reason.
func (c *Client) Close(id, reason string) error {
	_, err := c.run("close", id, "--reason", reason)
	return err
}

// Node and Edge describe a graph plan for bd create --graph.
type Node struct {
	Key         string   `json:"key"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Priority    int      `json:"priority"`
	Labels      []string `json:"labels,omitempty"`
	Description string   `json:"description"`
	Acceptance  string   `json:"acceptance_criteria,omitempty"`
	ParentID    string   `json:"parent_id,omitempty"`
}

// Edge says that From is blocked by To. An end is a key of the plan, or the
// id of an issue that exists already.
type Edge struct {
	FromKey string `json:"from_key,omitempty"`
	ToKey   string `json:"to_key,omitempty"`
	FromID  string `json:"from_id,omitempty"`
	ToID    string `json:"to_id,omitempty"`
	Type    string `json:"type"`
}

// Graph is a bd graph plan.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// CreateGraph creates the stories and edges and returns the id of each key.
func (c *Client) CreateGraph(g Graph) (map[string]string, error) {
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return nil, err
	}
	f, err := os.CreateTemp("", "machinery-plan-*.json")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return nil, err
	}
	f.Close()
	out, err := c.run("create", "--graph", f.Name(), "--json")
	if err != nil {
		return nil, err
	}
	var res struct {
		IDs map[string]string `json:"ids"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, fmt.Errorf("bd create --graph: unreadable JSON: %v", err)
	}
	return res.IDs, nil
}

// Create creates one issue and returns its id.
func (c *Client) Create(title, typ, description, parent string, labels []string) (string, error) {
	return c.CreateItem(Item{Title: title, Type: typ, Description: description, Parent: parent, Labels: labels})
}

// Item is one issue to create.
type Item struct {
	Title, Type, Description, Parent string
	Labels                           []string
	Deps                             []string // "<type>:<id>", as bd create --deps takes them
}

// CreateItem creates one issue and returns its id.
func (c *Client) CreateItem(it Item) (string, error) {
	args := []string{"create", "--title", it.Title, "--type", it.Type, "--description", it.Description, "--silent"}
	if len(it.Labels) > 0 {
		args = append(args, "--labels", strings.Join(it.Labels, ","))
	}
	if it.Parent != "" {
		args = append(args, "--parent", it.Parent)
	}
	if len(it.Deps) > 0 {
		args = append(args, "--deps", strings.Join(it.Deps, ","))
	}
	out, err := c.run(args...)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(out))
	if id == "" {
		return "", fmt.Errorf("bd create: no id returned")
	}
	return firstLine(id), nil
}

// Tags are the machine-readable lines of a story (section 15.1).
type Tags struct {
	IDs         []string
	Scope       []string
	Proofs      []Proof
	Predictions []Stamp
	Verdicts    []Stamp

	// intake (section 15.7)
	Intake         bool     // a bare machinery:intake line: the item is an intake item
	IntakeOf       string   // machinery:intake <id>: the intake a story belongs to
	Base           *Mark    // machinery:intake-base
	Planned        []Mark   // machinery:intake-planned
	NoDesignChange string   // machinery:no-design-change <reason>
	Capstone       bool     // machinery:capstone
	IntakeDesign   bool     // machinery:intake-design: the story lands the design of its intake
	Demos          []string // machinery:demo <command>

	// the records of the work (section 15.8)
	Learnings    []string      // machinery:learning <text>
	Observations []Observation // machinery:observation <kind> <text>
	Filed        []Observation // machinery:observation-filed <item id> <kind> <text>
	Repairs      []string      // machinery:repair <short sha>
	Decision     bool          // a machinery:decision line: the item is a decision
	DecidedBy    string        // the actor of the machinery:decision line
	Sources      []string      // machinery:decision-source <path>#<12 hex>
	Supersedes   []string      // machinery:supersedes <decision id>
	Vetoable     bool          // machinery:vetoable
	Confirmed    []string      // machinery:confirmed <actor>
	Overruled    []string      // machinery:overruled <actor>

	// questions for the owner (section 15.9)
	Question  bool     // a machinery:question line: the item is a question
	AskedBy   string   // the actor of the machinery:question line
	AskedIn   Who      // the session and the agent of the machinery:question line
	Options   []string // machinery:option <n> <text>, option n at index n-1
	Recommend int      // machinery:recommend <n>
	Default   string   // machinery:default <text>
	About     []string // machinery:about <topic>...
	Amount    string   // machinery:amount <number> <unit>
	Unit      string
	StoryOf   string   // machinery:story <id>
	Answers   []Answer // machinery:answer lines, in order

	// the checker's reviews (section 15.10)
	Reviews []Review // machinery:review lines, in order
}

// Review is a machinery:review line (section 15.10).
type Review struct {
	Kind     string // backlog | design | intake
	Verdict  string // approved | rejected
	Commit   string
	Contract string
	Actor    string
	Who      Who
	Stories  []string // the wave-1 ids of a backlog review, sorted
	Issues   []string
	Seq      int
}

// Answer is a machinery:answer line (section 15.9).
type Answer struct {
	What   string // an option number, "refuse" or "hold"
	Actor  string
	Clause string // the clause of an answer on the owner's behalf; "" for the owner's
	Reason string
	Seq    int
}

// Option returns the option number of the answer, 0 for a refusal or a hold.
func (a Answer) Option() int {
	n, err := strconv.Atoi(a.What)
	if err != nil || n < 1 {
		return 0
	}
	return n
}

// Delegated reports whether the answer was given on the owner's behalf.
func (a Answer) Delegated() bool { return a.Clause != "" }

// Observation is a machinery:observation line, or with Item set a
// machinery:observation-filed line.
type Observation struct {
	Item string // the item it was filed as
	Kind string
	Text string
}

// Mark is a commit and a contract hash (machinery:intake-base and
// machinery:intake-planned), with the planner who wrote the line when it
// records one.
type Mark struct {
	Commit   string
	Contract string
	Actor    string
	Who      Who
}

// Proof is a machinery:proof line.
type Proof struct {
	Commit  string
	Actor   string
	Session string // the session=<value> word, "" when none
	Agent   string // the agent=<value> word, "" when none
	Seq     int    // position among all tag lines of the story
}

// Stamp is a machinery:prediction or machinery:verdict line.
type Stamp struct {
	Verdict string // accept | reject, for verdicts
	Commit  string
	Actor   string
	Fields  map[string]string
	Seq     int
}

var kvRe = regexp.MustCompile(`(\w+)=("([^"\\]|\\.)*"|\S+)`)

// ParseTags reads the tag lines of a story's description and notes.
func ParseTags(is *Issue) Tags {
	var t Tags
	seq := 0
	seen := map[string]bool{}
	for _, line := range strings.Split(is.Description+"\n"+is.Notes, "\n") {
		i := strings.Index(line, "machinery:")
		if i < 0 {
			continue
		}
		rest := line[i+len("machinery:"):]
		word, args, _ := strings.Cut(strings.TrimSpace(rest), " ")
		seq++
		switch word {
		case "ids":
			for _, id := range strings.Fields(args) {
				if !seen[id] {
					seen[id] = true
					t.IDs = append(t.IDs, id)
				}
			}
		case "scope":
			t.Scope = append(t.Scope, strings.Fields(args)...)
		case "intake":
			if f := strings.Fields(args); len(f) == 0 {
				t.Intake = true
			} else if t.IntakeOf == "" {
				t.IntakeOf = f[0]
			}
		case "intake-base", "intake-planned":
			f := strings.Fields(args)
			if len(f) < 2 {
				continue
			}
			kv := parseKV(strings.Join(f[2:], " "))
			m := Mark{Commit: f[0], Contract: f[1], Actor: kv["actor"], Who: Who{kv["session"], kv["agent"]}}
			if word == "intake-planned" {
				t.Planned = append(t.Planned, m)
			} else if t.Base == nil {
				t.Base = &m
			}
		case "no-design-change":
			if r := strings.TrimSpace(args); r != "" && t.NoDesignChange == "" {
				t.NoDesignChange = r
			}
		case "capstone":
			t.Capstone = true
		case "intake-design":
			t.IntakeDesign = true
		case "decision-source":
			if f := strings.Fields(args); len(f) > 0 {
				t.Sources = append(t.Sources, f[0])
			}
		case "review":
			t.Reviews = append(t.Reviews, parseReview(args, seq))
		case "demo":
			if cmd := strings.TrimSpace(args); cmd != "" {
				t.Demos = append(t.Demos, cmd)
			}
		case "learning":
			if l := strings.TrimSpace(args); l != "" {
				t.Learnings = append(t.Learnings, l)
			}
		case "observation":
			kind, text, _ := strings.Cut(strings.TrimSpace(args), " ")
			if text = strings.TrimSpace(text); kind != "" && text != "" {
				t.Observations = append(t.Observations, Observation{Kind: kind, Text: text})
			}
		case "observation-filed":
			f := strings.Fields(args)
			if len(f) >= 3 {
				t.Filed = append(t.Filed, Observation{Item: f[0], Kind: f[1], Text: strings.Join(f[2:], " ")})
			}
		case "repair":
			if f := strings.Fields(args); len(f) > 0 {
				t.Repairs = append(t.Repairs, f[0])
			}
		case "decision":
			if !t.Decision {
				t.Decision, t.DecidedBy = true, strings.TrimSpace(args)
			}
		case "supersedes":
			t.Supersedes = append(t.Supersedes, strings.Fields(args)...)
		case "vetoable":
			t.Vetoable = true
		case "confirmed":
			t.Confirmed = append(t.Confirmed, strings.TrimSpace(args))
		case "overruled":
			t.Overruled = append(t.Overruled, strings.TrimSpace(args))
		case "question":
			if !t.Question {
				t.Question = true
				t.AskedBy, t.AskedIn = actorAndWho(strings.Fields(args))
			}
		case "option":
			num, text, _ := strings.Cut(strings.TrimSpace(args), " ")
			if n, err := strconv.Atoi(num); err == nil && n >= 1 && n <= maxOptions {
				for len(t.Options) < n {
					t.Options = append(t.Options, "")
				}
				t.Options[n-1] = strings.TrimSpace(text)
			}
		case "recommend":
			if n, err := strconv.Atoi(strings.TrimSpace(args)); err == nil && t.Recommend == 0 {
				t.Recommend = n
			}
		case "default":
			if t.Default == "" {
				t.Default = strings.TrimSpace(args)
			}
		case "about":
			t.About = append(t.About, strings.Fields(args)...)
		case "amount":
			if f := strings.Fields(args); len(f) == 2 && t.Amount == "" {
				t.Amount, t.Unit = f[0], f[1]
			}
		case "story":
			if f := strings.Fields(args); len(f) > 0 && t.StoryOf == "" {
				t.StoryOf = f[0]
			}
		case "answer":
			what, more, _ := strings.Cut(strings.TrimSpace(args), " ")
			kv := parseKV(more)
			t.Answers = append(t.Answers, Answer{What: what, Actor: kv["actor"], Clause: kv["clause"], Reason: kv["reason"], Seq: seq})
		case "proof":
			f := strings.Fields(args)
			if len(f) < 2 {
				continue
			}
			pr := Proof{Commit: f[0], Seq: seq}
			var who Who
			pr.Actor, who = actorAndWho(f[1:])
			pr.Session, pr.Agent = who.Session, who.Agent
			t.Proofs = append(t.Proofs, pr)
		case "prediction", "verdict":
			st := Stamp{Fields: map[string]string{}, Seq: seq}
			if word == "verdict" {
				v, more, _ := strings.Cut(strings.TrimSpace(args), " ")
				st.Verdict, args = v, more
			}
			st.Fields = parseKV(args)
			st.Commit, st.Actor = st.Fields["commit"], st.Fields["actor"]
			if word == "verdict" {
				t.Verdicts = append(t.Verdicts, st)
			} else {
				t.Predictions = append(t.Predictions, st)
			}
		}
	}
	sort.Strings(t.IDs)
	return t
}

// actorAndWho reads the words of a line that end with optional session= and
// agent= words: the words before them are the actor.
func actorAndWho(f []string) (string, Who) {
	var w Who
	for len(f) >= 2 {
		last := f[len(f)-1]
		if v, ok := strings.CutPrefix(last, "session="); ok && w.Session == "" {
			w.Session = v
		} else if v, ok := strings.CutPrefix(last, "agent="); ok && w.Agent == "" {
			w.Agent = v
		} else {
			break
		}
		f = f[:len(f)-1]
	}
	return strings.Join(f, " "), w
}

// parseReview reads the words after machinery:review.
func parseReview(args string, seq int) Review {
	kind, rest, _ := strings.Cut(strings.TrimSpace(args), " ")
	verdict, rest, _ := strings.Cut(strings.TrimSpace(rest), " ")
	kv := parseKV(rest)
	r := Review{Kind: kind, Verdict: verdict, Commit: kv["commit"], Contract: kv["contract"], Actor: kv["actor"],
		Who: Who{kv["session"], kv["agent"]}, Seq: seq}
	if s := kv["stories"]; s != "" {
		r.Stories = strings.Split(s, ",")
		sort.Strings(r.Stories)
	}
	for n := 1; n <= 5; n++ {
		if v, ok := kv[fmt.Sprintf("issue%d", n)]; ok {
			r.Issues = append(r.Issues, v)
		}
	}
	return r
}

// maxOptions bounds the option numbers a question line may carry.
const maxOptions = 99

// parseKV reads the key=value fields of a tag line; a quoted value is unquoted.
func parseKV(s string) map[string]string {
	res := map[string]string{}
	for _, m := range kvRe.FindAllStringSubmatch(s, -1) {
		val := m[2]
		if strings.HasPrefix(val, "\"") {
			val = strings.NewReplacer("\\\"", "\"", "\\\\", "\\").Replace(val[1 : len(val)-1])
		}
		res[m[1]] = val
	}
	return res
}

// Value renders a value for a key=value tag field: bare when it is one word,
// quoted otherwise.
func Value(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\r\n\"\\") {
		return s
	}
	return Quote(s)
}

// SessionEnv names the environment variable that holds the harness session
// (section 15.1).
const SessionEnv = "MACHINERY_SESSION"

// AgentEnv names the environment variable that holds the agent, the worker
// context within a session (section 15.1).
const AgentEnv = "MACHINERY_AGENT"

// Session returns the session of section 15.1, "" when it is unset or empty.
// A value that is not one word is an error that names it.
func Session() (string, error) { return oneWord(SessionEnv) }

// Agent returns the agent of section 15.1, read like the session.
func Agent() (string, error) { return oneWord(AgentEnv) }

func oneWord(env string) (string, error) {
	v := strings.TrimSpace(os.Getenv(env))
	if strings.ContainsAny(v, " \t\r\n\"") {
		return "", fmt.Errorf("%s must be one word, got %q", env, v)
	}
	return v, nil
}

// Who is the session and the agent a tag line records.
type Who struct{ Session, Agent string }

// WhoFromEnv reads the session and the agent; the first value that is not
// one word is the error.
func WhoFromEnv() (Who, error) {
	s, err := Session()
	if err != nil {
		return Who{}, err
	}
	a, err := Agent()
	return Who{s, a}, err
}

// Part renders the tag line part: " session=<value> agent=<value>", each only
// when set, "" when neither is.
func (w Who) Part() string {
	var s string
	if w.Session != "" {
		s += " session=" + w.Session
	}
	if w.Agent != "" {
		s += " agent=" + w.Agent
	}
	return s
}

// Quote renders a value for a key="value" tag field.
func Quote(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\\", "\\\\")
	return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\""
}

// HasLabel reports whether the issue carries a label.
func (i *Issue) HasLabel(l string) bool {
	for _, x := range i.Labels {
		if x == l {
			return true
		}
	}
	return false
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
