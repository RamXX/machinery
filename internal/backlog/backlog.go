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
	"strings"

	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
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

// Edge says that From is blocked by To.
type Edge struct {
	FromKey string `json:"from_key"`
	ToKey   string `json:"to_key"`
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

// Tags are the machine-readable lines of a story (section 15.1).
type Tags struct {
	IDs         []string
	Scope       []string
	Proofs      []Proof
	Predictions []Stamp
	Verdicts    []Stamp
}

// Proof is a machinery:proof line.
type Proof struct {
	Commit string
	Actor  string
	Seq    int // position among all tag lines of the story
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
		case "proof":
			f := strings.Fields(args)
			if len(f) >= 2 {
				t.Proofs = append(t.Proofs, Proof{Commit: f[0], Actor: strings.Join(f[1:], " "), Seq: seq})
			}
		case "prediction", "verdict":
			st := Stamp{Fields: map[string]string{}, Seq: seq}
			if word == "verdict" {
				v, more, _ := strings.Cut(strings.TrimSpace(args), " ")
				st.Verdict, args = v, more
			}
			for _, m := range kvRe.FindAllStringSubmatch(args, -1) {
				val := m[2]
				if strings.HasPrefix(val, "\"") {
					val = strings.NewReplacer("\\\"", "\"", "\\\\", "\\").Replace(val[1 : len(val)-1])
				}
				st.Fields[m[1]] = val
			}
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
