package adopt

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/RamXX/machinery/internal/project"
)

// Default file names adopt writes (section 17.2).
const (
	DefaultE2E       = "e2e/**"
	DesignDir        = "design/" // what adopt writes lives here, or in the gen directory
	TestsBaseline    = "design/tests-baseline.yaml"
	ArchBaseline     = "design/arch-baseline.yaml"
	defaultScopeGlob = "**"
)

// adoptionHead is the comment at the top of design/adoption.yaml.
const adoptionHead = "machinery adopt writes this file whole: what adoption added to the configuration. machinery.yaml is yours; adopt never edits it."

// plan collects what adoption adds to the configuration (section 17.2, steps
// 1, 4 and 5). It never touches the text of a file: view is the
// configuration as the files wrote it, decoded, with each addition made to it
// as it is planned, and add holds the additions alone. A new project gets
// them as its machinery.yaml; a project that has one gets them in
// design/adoption.yaml (section 2.1).
type plan struct {
	view  project.Config
	add   project.Adoption
	notes []string
}

func (pl *plan) note(format string, a ...any) {
	pl.notes = append(pl.notes, fmt.Sprintf(format, a...))
}

// list appends to a list of the view, and to the same list of the additions,
// the values the view does not hold yet.
func (pl *plan) list(view, add *[]string, values []string, what string) {
	have := map[string]bool{}
	for _, v := range *view {
		have[v] = true
	}
	var added []string
	for _, v := range values {
		if !have[v] {
			have[v] = true
			added = append(added, v)
		}
	}
	if len(added) == 0 {
		return
	}
	*view = append(append([]string{}, *view...), added...)
	*add = append(*add, added...)
	pl.note("%s +%s", what, strings.Join(added, ", "))
}

// set gives a scalar of the view, and of the additions, its value.
func (pl *plan) set(view, add *string, value, what string) {
	*view, *add = value, value
	pl.note("%s", what)
}

// covered drops the globs an existing glob already matches.
func covered(existing, globs []string) []string {
	var res []string
	for _, g := range globs {
		if !project.Match(existing, g) {
			res = append(res, g)
		}
	}
	return res
}

// step1 is step 1 of section 17.2: what the configuration is missing. A key
// that is absent, or holds an empty list, is absent.
func (pl *plan) step1(existing bool, o Options, src Sources, backlogFound bool) {
	v, add := &pl.view, &pl.add
	switch {
	case !existing:
		paths := o.Paths
		if len(paths) == 0 {
			paths = []string{defaultScopeGlob}
		}
		pl.list(&v.Scope, &add.Scope, paths, "scope")
	case len(v.Scope) > 0:
		pl.list(&v.Scope, &add.Scope, covered(v.Scope, o.Paths), "scope")
	}
	effScope := append([]string{}, v.Scope...)
	if len(effScope) == 0 {
		effScope = []string{defaultScopeGlob}
	}

	// design sources found now, including ones written after the first adoption
	pl.list(&v.Design.Models, &add.Design.Models, src.Models, "design.models")
	imported := v.ImportedMachines() != nil
	if len(src.Machines) > 0 && v.Design.Machines == "" && !imported {
		dir := src.machinesDir()
		pl.set(&v.Design.Machines, &add.Design.Machines, dir, "design.machines +"+dir)
	}
	have := map[string]bool{}
	for _, c := range v.Contracts {
		have[c.Path] = true
	}
	for _, path := range src.Contracts {
		if !have[path] {
			c := project.Contract{Kind: "openapi", Path: path}
			v.Contracts = append(append([]project.Contract{}, v.Contracts...), c)
			add.Contracts = append(add.Contracts, c)
			pl.note("contracts +%s", path)
		}
	}
	hasDesign := len(v.Design.Models) > 0 || v.Design.Machines != "" || imported

	traceExisted, testsExisted := v.Trace != nil, v.Tests != nil
	e2e := o.E2E
	if len(e2e) == 0 && testsExisted {
		e2e = v.Tests.E2E
	}
	if len(e2e) == 0 && traceExisted {
		e2e = v.Trace.Tests
	}
	if len(e2e) == 0 {
		e2e = []string{DefaultE2E}
	}
	e2e = append([]string{}, e2e...)
	switch {
	case !traceExisted && hasDesign:
		v.Trace = &project.Trace{}
		pl.list(&v.Trace.Code, &add.Trace.Code, effScope, "trace.code")
		pl.list(&v.Trace.Tests, &add.Trace.Tests, e2e, "trace.tests")
	case traceExisted:
		if len(v.Trace.Code) > 0 {
			pl.list(&v.Trace.Code, &add.Trace.Code, covered(v.Trace.Code, o.Paths), "trace.code")
		}
		if len(v.Trace.Tests) > 0 {
			pl.list(&v.Trace.Tests, &add.Trace.Tests, covered(v.Trace.Tests, o.E2E), "trace.tests")
		}
	}

	if !testsExisted {
		v.Tests = &project.Tests{}
		if !traceExisted {
			pl.list(&v.Tests.E2E, &add.Tests.E2E, e2e, "tests.e2e")
		}
	} else if len(o.E2E) > 0 {
		switch {
		case len(v.Tests.E2E) > 0:
			pl.list(&v.Tests.E2E, &add.Tests.E2E, covered(v.Tests.E2E, o.E2E), "tests.e2e")
		case v.Trace == nil || len(v.Trace.Tests) == 0:
			pl.list(&v.Tests.E2E, &add.Tests.E2E, o.E2E, "tests.e2e")
		}
	}
	if v.Tests.Baseline == "" {
		pl.set(&v.Tests.Baseline, &add.Tests.Baseline, TestsBaseline, "tests.baseline")
	}
	if v.Constraints != nil && v.Constraints.Baseline == "" {
		pl.set(&v.Constraints.Baseline, &add.Constraints.Baseline, ArchBaseline, "constraints.baseline")
	}
	if backlogFound && v.Backlog == nil {
		v.Backlog = &project.Backlog{}
		pl.set(&v.Backlog.Tool, &add.Backlog.Tool, "bd", "backlog.tool: bd")
	}
}

// waive adds one waiver per entry, each with the reason (section 17.2, steps
// 4 and 5), and notes how many. It never removes or rewrites a waiver.
func (pl *plan) waive(ws []project.Waiver, reason, note string) {
	for _, w := range ws {
		pl.add.Waivers = append(pl.add.Waivers, project.Waiver{Rule: w.Rule, ID: w.ID, Reason: reason})
	}
	pl.note(note, len(ws))
}

// render encodes the keys adopt writes, in their order, as a new
// machinery.yaml (version 2) or as design/adoption.yaml (version 1, with its
// head comment): lists in block style, globs quoted, one waiver per line.
func render(version int, a *project.Adoption) ([]byte, error) {
	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	put := func(m *yaml.Node, key string, v *yaml.Node) {
		m.Content = append(m.Content, scalar(key, false), v)
	}
	str := func(m *yaml.Node, key, v string) {
		if v != "" {
			put(m, key, scalar(v, false))
		}
	}
	seq := func(m *yaml.Node, key string, vs []string, quoted bool) {
		if len(vs) == 0 {
			return
		}
		s := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, v := range vs {
			s.Content = append(s.Content, scalar(v, quoted))
		}
		put(m, key, s)
	}
	block := func(key string, fill func(m *yaml.Node)) {
		m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		if fill(m); len(m.Content) > 0 {
			put(root, key, m)
		}
	}
	put(root, "version", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprint(version)})
	if version == 1 {
		root.Content[0].HeadComment = adoptionHead
	}
	seq(root, "scope", a.Scope, true)
	block("design", func(m *yaml.Node) {
		seq(m, "models", a.Design.Models, false)
		str(m, "machines", a.Design.Machines)
	})
	if len(a.Contracts) > 0 {
		s := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, c := range a.Contracts {
			item := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			put(item, "kind", scalar(c.Kind, false))
			put(item, "path", scalar(c.Path, false))
			s.Content = append(s.Content, item)
		}
		put(root, "contracts", s)
	}
	block("trace", func(m *yaml.Node) {
		seq(m, "code", a.Trace.Code, true)
		seq(m, "tests", a.Trace.Tests, true)
	})
	block("tests", func(m *yaml.Node) {
		seq(m, "e2e", a.Tests.E2E, true)
		str(m, "baseline", a.Tests.Baseline)
	})
	block("constraints", func(m *yaml.Node) { str(m, "baseline", a.Constraints.Baseline) })
	block("backlog", func(m *yaml.Node) { str(m, "tool", a.Backlog.Tool) })
	if len(a.Waivers) > 0 {
		s := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, w := range a.Waivers {
			item := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Style: yaml.FlowStyle}
			put(item, "rule", scalar(w.Rule, false))
			put(item, "id", scalar(w.ID, true))
			put(item, "reason", scalar(w.Reason, true))
			s.Content = append(s.Content, item)
		}
		put(root, "waivers", s)
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func scalar(v string, quoted bool) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
	if quoted {
		n.Style = yaml.DoubleQuotedStyle
	}
	return n
}

// config is the configuration adoption leaves behind, and the file that
// holds the additions: for a new project its machinery.yaml, for one that
// has a machinery.yaml (cfg, never written) design/adoption.yaml, which is
// what it held (old, its bytes in oldBytes) and then the additions. data is
// nil when there is nothing to write.
func (pl *plan) config(existing bool, cfg, oldBytes []byte, old *project.Adoption) (c *project.Config, data []byte, err error) {
	switch {
	case !existing:
		if data, err = render(2, &pl.add); err != nil {
			return nil, nil, err
		}
		c, err = project.Parse(data)
	case len(pl.notes) == 0:
		c, err = project.ParseWith(cfg, oldBytes)
	default:
		whole := *old
		if err = whole.Add(&pl.add, project.AdoptionName, "adopt"); err != nil {
			return nil, nil, err
		}
		if data, err = render(1, &whole); err != nil {
			return nil, nil, err
		}
		c, err = project.ParseWith(cfg, data)
	}
	return c, data, err
}
