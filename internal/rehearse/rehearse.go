// Package rehearse adopts a snapshot of a project outside it and reports what
// adoption would take and what blocks it (docs/formats.md section 17.3). It
// writes only below its out directory and never runs the backlog tool.
package rehearse

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/adopt"
	"github.com/RamXX/machinery/internal/arch"
	"github.com/RamXX/machinery/internal/assess"
	"github.com/RamXX/machinery/internal/baseline"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/pm"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/stages"
	"github.com/RamXX/machinery/internal/testpolicy"
)

const verb = "adopt --rehearse"

// noBacklog is why stage intake does not run in a rehearsal.
const noBacklog = "a rehearsal never reads the backlog"

// firstFindings is how many findings of a blocking stage the report lists.
const firstFindings = 5

// fixFirstStages are the stages whose ERROR no waiver covers.
var fixFirstStages = map[string]bool{"model": true, "machines": true, "failure-modes": true, "risk": true}

// owedKinds orders the kinds of the owed list.
var owedKinds = []string{"tc", "sc", "inv", "fm", "fv", "cx", "ck", "act"}

// Options are the arguments of machinery adopt --rehearse.
type Options struct {
	ProjectFlag string
	Config      string
	Out         string
	Overlay     string
	Paths, E2E  []string
	Stdout      io.Writer
	Gen         func(*stages.Context) (written []string, blocked []*report.Stage, err error)
}

type file struct {
	Path  string
	Bytes int64
}

type count struct {
	Key   string
	Count int
}

type finding struct {
	Severity, Message, Disposition, Waiver string
}

type block struct {
	Stage, Outcome string
	Findings       []finding
	More           int
}

type rehearsal struct {
	name, head, cfgLabel string
	changed, untracked   int
	outside              []string
	overlay              *assess.Overlay
	files                []file
	waivers              []count
	baselines            []count
	owed                 []count
	owedTotal            int
	blocked              []string
	checkLine            string
	stageOutcomes        []count // Key: stage, outcome in waiverOf
	outcomes             map[string]string
	planStories          []planCount
	planTotal, planEdges int
	planError            string
	blocks               []block
	fixFirst             [][2]string
	adoptLine            string
	backlogLine          string
}

type planCount struct {
	Kind, Tier string
	Count      int
}

// Run rehearses the adoption. A *UsageError or a config error means nothing
// was written; any other error means the snapshot could not be taken.
func Run(o Options) error {
	os.Setenv("GIT_OPTIONAL_LOCKS", "0")
	root, err := assess.ResolveRoot(o.ProjectFlag)
	if err != nil {
		return err
	}
	for _, g := range append(append([]string{}, o.Paths...), o.E2E...) {
		if err := project.CheckPath("glob", g); err != nil {
			return err
		}
		if !project.ValidGlob(g) {
			return project.Errf("%q is not a valid glob", g)
		}
	}
	r := &rehearsal{name: filepath.Base(root), cfgLabel: "none", outcomes: map[string]string{}}
	if r.overlay, err = assess.ReadOverlay(o.Overlay); err != nil {
		return err
	}
	var cfg []byte
	if oc := r.overlay.Config(); oc != nil {
		if o.Config != "" {
			return &assess.UsageError{Msg: fmt.Sprintf("--overlay %s holds %s and --config is given; keep one", o.Overlay, project.ConfigName)}
		}
		cfg, r.cfgLabel = oc, "--overlay "+project.ConfigName
	} else if o.Config != "" {
		data, rerr := os.ReadFile(o.Config)
		if rerr != nil {
			return project.Errf("--config %s: cannot read: %v", o.Config, shortErr(rerr))
		}
		cfg, r.cfgLabel = data, "--config "+filepath.Base(o.Config)
	} else if data, rerr := os.ReadFile(filepath.Join(root, project.ConfigName)); rerr == nil {
		cfg, r.cfgLabel = data, project.ConfigName
	}
	var parsed *project.Config
	if cfg != nil {
		if parsed, err = project.Parse(cfg); err != nil {
			return err
		}
	}
	if r.head, err = assess.HeadOf(root, verb); err != nil {
		return err
	}
	var external *project.Config
	if o.Config != "" {
		external = parsed
	}
	if err := r.overlay.Check(root, external, r.cfgLabel, parsed, r.cfgLabel); err != nil {
		return err
	}
	out, err := assess.PrepareOut(root, o.Out, verb)
	if err != nil {
		return err
	}
	w := o.Stdout
	fmt.Fprintf(w, "out: %s\n", out)
	snap, outside, err := assess.TakeSnapshot(root, out, r.head, cfg, r.overlay)
	if err != nil {
		return fmt.Errorf("%s: cannot take the snapshot: %v", verb, err)
	}
	r.outside = outside
	if r.changed, r.untracked, err = gitx.StatusCounts(root); err != nil {
		return err
	}
	if err := r.rehearse(o, root, out, snap); err != nil {
		return fmt.Errorf("%s: %v", verb, err)
	}
	fmt.Fprintf(w, "project: %s at %s\n", r.name, r.head)
	if r.changed > 0 || r.untracked > 0 {
		fmt.Fprintf(w, "dirty: %d changed, %d untracked file(s) are not rehearsed\n", r.changed, r.untracked)
	}
	if len(r.outside) > 0 {
		fmt.Fprintf(w, "outside links: %d symbolic link(s) leave the project and are not in the snapshot: %s\n", len(r.outside), strings.Join(r.outside, ", "))
	}
	if line := r.overlay.Line(); line != "" {
		fmt.Fprintln(w, line)
	}
	fmt.Fprintln(w, r.adoptLine)
	fmt.Fprintf(w, "check: %s\n", r.checkLine)
	if r.planError != "" {
		fmt.Fprintf(w, "plan: %s\n", r.planError)
	} else {
		fmt.Fprintf(w, "plan: %d stories, %d edge(s)\n", r.planTotal, r.planEdges)
	}
	if len(r.blocks) == 0 {
		fmt.Fprintln(w, "blocks adoption: nothing")
	} else {
		var names []string
		for _, b := range r.blocks {
			names = append(names, b.Stage)
		}
		fmt.Fprintf(w, "blocks adoption: %d stage(s): %s\n", len(r.blocks), strings.Join(names, ", "))
	}
	if len(r.fixFirst) > 0 {
		fmt.Fprintf(w, "fix in the project first: %d finding(s)\n", len(r.fixFirst))
	}
	if err := r.write(out); err != nil {
		return fmt.Errorf("%s: cannot write the report: %v", verb, err)
	}
	fmt.Fprintln(w, "rehearse: rehearsal.md, rehearsal.json")
	return nil
}

func shortErr(err error) string {
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i >= 0 {
		return s[i+2:]
	}
	return s
}

// rehearse adopts the snapshot, checks it and builds the plan.
func (r *rehearsal) rehearse(o Options, root, out, snap string) error {
	before := waiverCounts(snap)
	res, err := adopt.Run(adopt.Options{Root: snap, Paths: o.Paths, E2E: o.E2E, Write: true, Gen: o.Gen, NoBacklog: true})
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	res.Print(&buf)
	if err := save(out, "stages/adopt.txt", buf.Bytes()); err != nil {
		return err
	}
	for _, f := range res.Written {
		info, serr := os.Stat(filepath.Join(snap, filepath.FromSlash(f)))
		if serr == nil {
			r.files = append(r.files, file{f, info.Size()})
		}
	}
	after := waiverCounts(snap)
	added := 0
	for _, rule := range sortedKeys(after) {
		if n := after[rule] - before[rule]; n > 0 {
			r.waivers = append(r.waivers, count{rule, n})
			added += n
		}
	}
	r.blocked = res.BlockedLines()
	r.backlogLine = res.Backlog.Line()
	if err := gitx.CommitAll(snap, "rehearsed adoption"); err != nil {
		return err
	}
	p, err := project.Load(snap)
	if err != nil {
		return err
	}
	if d := p.Cfg.Tools.Dir; d != "" && !filepath.IsAbs(d) {
		p.Cfg.Tools.Dir = filepath.Join(root, filepath.FromSlash(d))
	}
	ctx := &stages.Context{P: p, NoBacklog: noBacklog}
	sts := stages.Run(ctx, stages.CheckStages)
	buf.Reset()
	report.PrintAll(&buf, "check", sts)
	if err := save(out, "stages/check.txt", buf.Bytes()); err != nil {
		return err
	}
	var sum bytes.Buffer
	report.Summary(&sum, "check", sts)
	r.checkLine = strings.TrimSpace(sum.String())
	d := ctx.Design()
	r.findings(sts)
	r.counts(p, d)
	if len(r.blocked) > 0 {
		r.adoptLine = fmt.Sprintf("adopt: blocked: machinery gen refuses while %d finding(s) stand; nothing is owed", blockedCount(r.blocked))
	} else {
		r.adoptLine = fmt.Sprintf("adopt: wrote %d file(s), %d waiver(s), %d owed", len(r.files), added, r.owedTotal)
	}
	return r.plan(out, d)
}

// blockedCount reads the count of the first line of adopt's blocked report.
func blockedCount(lines []string) int {
	return len(lines) - 1 - boolInt(strings.HasPrefix(lines[len(lines)-1], "fix in the project first"))
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// waiverCounts counts the waivers of the snapshot's machinery.yaml by rule.
func waiverCounts(snap string) map[string]int {
	res := map[string]int{}
	data, err := os.ReadFile(filepath.Join(snap, project.ConfigName))
	if err != nil {
		return res
	}
	cfg, err := project.Parse(data)
	if err != nil {
		return res
	}
	for _, w := range cfg.Waivers {
		res[w.Rule]++
	}
	return res
}

// counts fills the baselines and the owed list by kind.
func (r *rehearsal) counts(p *project.Project, d *design.Design) {
	for _, rel := range []string{testpolicy.BaselinePath(p), arch.BaselinePath(p)} {
		if rel == "" || !p.Exists(rel) {
			continue
		}
		entries, _ := baseline.Load(p, rel)
		r.baselines = append(r.baselines, count{rel, len(entries)})
	}
	byKind := map[string]int{}
	for _, o := range d.Owed {
		kind, _, _ := strings.Cut(o.ID, ":")
		byKind[kind]++
		r.owedTotal++
	}
	for _, k := range owedKinds {
		if byKind[k] > 0 {
			r.owed = append(r.owed, count{k, byKind[k]})
		}
	}
}

// findings records the outcome of each stage, what blocks and what to fix in
// the project first.
func (r *rehearsal) findings(sts []*report.Stage) {
	for _, s := range sts {
		outcome := pm.OutcomeLine(s)
		r.stageOutcomes = append(r.stageOutcomes, count{s.Name, 0})
		r.outcomes[s.Name] = outcome
		if fixFirstStages[s.Name] {
			for _, f := range s.Sorted() {
				if f.Sev == report.Error {
					r.fixFirst = append(r.fixFirst, [2]string{s.Name, f.Msg})
				}
			}
		}
		if s.OK() {
			continue
		}
		b := block{Stage: s.Name, Outcome: outcome}
		for _, f := range s.Sorted() {
			if f.Sev == report.Warn {
				continue
			}
			if len(b.Findings) == firstFindings {
				b.More++
				continue
			}
			fd := finding{Severity: report.Label(f.Sev), Message: f.Msg}
			fd.Disposition, fd.Waiver = disposition(s.Name, f, r.head)
			b.Findings = append(b.Findings, fd)
		}
		r.blocks = append(r.blocks, b)
	}
}

// disposition says what to do about a blocking finding: fix it in the
// project, or the waiver a cross-reference rule takes (offered, never written).
func disposition(stage string, f report.Finding, sha string) (string, string) {
	if f.Sev != report.Error {
		return "", ""
	}
	if fixFirstStages[stage] {
		return "fix in the project first", ""
	}
	parts := strings.SplitN(f.Msg, ": ", 3)
	if len(parts) == 3 && project.WaivableRules[parts[0]] {
		waiver := fmt.Sprintf("- {rule: %s, id: %q, reason: \"found at adoption (%s)\"}", parts[0], parts[1], sha)
		return "waivable: " + waiver, waiver
	}
	return "", ""
}

// plan builds what machinery plan would print, writes it to plan.md and
// counts its stories.
func (r *rehearsal) plan(out string, d *design.Design) error {
	plan, err := pm.BuildPlan(d)
	if err != nil {
		r.planError = "refused: " + err.Error()
		return nil
	}
	var buf bytes.Buffer
	plan.Markdown(&buf)
	if err := save(out, "plan.md", buf.Bytes()); err != nil {
		return err
	}
	counts := map[[2]string]int{}
	for _, s := range plan.Stories {
		kind := ""
		if len(s.IDs) > 0 {
			kind, _, _ = strings.Cut(s.IDs[0], ":")
		}
		counts[[2]string{kind, s.Tier}]++
	}
	for k, n := range counts {
		r.planStories = append(r.planStories, planCount{k[0], k[1], n})
	}
	rank := map[string]int{}
	for i, k := range owedKinds {
		rank[k] = i
	}
	sort.Slice(r.planStories, func(i, j int) bool {
		a, b := r.planStories[i], r.planStories[j]
		if a.Kind != b.Kind {
			return rank[a.Kind] < rank[b.Kind]
		}
		return design.TierRank(a.Tier) < design.TierRank(b.Tier)
	})
	r.planTotal, r.planEdges = len(plan.Stories), len(plan.Edges)
	return nil
}

func save(out, rel string, data []byte) error {
	path := filepath.Join(out, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func sortedKeys(m map[string]int) []string {
	var res []string
	for k := range m {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}
