package cli

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/formal"
	"github.com/RamXX/machinery/internal/gen"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
)

func init() {
	Register(&Command{Name: "formal new", Summary: "Scaffold a spec manifest and a property module.", Run: runFormalNew})
	Register(&Command{Name: "formal gen", Summary: "Write the generated skeletons of the formal models.", Run: runFormalGen})
	Register(&Command{Name: "formal check", Summary: "Run the model checkers and record results.", Run: runFormalCheck})
	Register(&Command{Name: "formal baseline", Summary: "Record the baseline of properties, fairness and bounds.", Run: runFormalBaseline})
	Register(&Command{Name: "formal sync", Summary: "Show or classify source drift of a formal model.", Run: runFormalSync})
	Register(&Command{Name: "formal tools", Summary: "Fetch the pinned formal tools by checksum.", Run: runFormalTools})
}

func selectSpecs(specs []*formal.Spec, names []string) ([]*formal.Spec, error) {
	if len(names) == 0 {
		return specs, nil
	}
	var res []*formal.Spec
	for _, n := range names {
		found := false
		for _, s := range specs {
			if s.Spec == n {
				res = append(res, s)
				found = true
			}
		}
		if !found {
			return nil, project.Errf("no spec %q in %s", n, "the formal directory")
		}
	}
	return res, nil
}

func runFormalNew(e *Env, args []string) int {
	fs := e.Flags("formal new")
	tool := fs.String("tool", "", "tla or alloy")
	machineName := fs.String("machine", "", "tla: base name of the machine file")
	model := fs.String("model", "", "alloy: the model file")
	area := fs.String("area", "", "the risk area the spec models")
	pos, ok := e.ParseMixed(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 || *area == "" || (*tool != "tla" && *tool != "alloy") {
		return e.Usage("machinery formal new <spec> --tool tla --machine <Name> --area <area> | --tool alloy --model <path> --area <area>")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	head, err := gitx.Head(p.Root)
	if err != nil {
		return e.Fail(project.Errf("formal new needs a git repository with a commit: sources.read_at is a commit"))
	}
	d := design.Load(p)
	name := pos[0]
	s := &formal.Spec{Dir: formal.Dir(p), Path: formal.Dir(p) + "/" + name + formal.Suffix}
	s.Spec, s.Tool = name, *tool
	var questions []string
	for _, a := range d.Areas {
		if a.Name == *area {
			for _, q := range a.Questions {
				if design.Questions[q] == *tool {
					questions = append(questions, q)
				}
			}
		}
	}
	var source, props, run string
	var propsText []byte
	if *tool == "tla" {
		if *machineName == "" {
			return e.Usage("--tool tla needs --machine")
		}
		s.Machine = *machineName
		for _, m := range d.Machines {
			if m.Base == *machineName {
				source = m.Path
			}
		}
		if source == "" {
			return e.Fail(project.Errf("no machine file %s.machine.json in %s", *machineName, p.Cfg.Design.Machines))
		}
		props = *machineName + "Props"
		propsText = formal.TLAPropsTemplate(props, *machineName)
		if len(questions) == 0 {
			questions = []string{"ordering"}
		}
		run = "  - name: strict\n    constants: {Instances: 1, Callers: 2, Atomic: true}\n    invariants: [TypeOK]\n    reach: [SomeFinal]\n    expect: pass\n"
	} else {
		if *model == "" {
			return e.Usage("--tool alloy needs --model")
		}
		s.Model = *model
		source = *model
		module := formal.AlloyModule(*model)
		props = module + "_props"
		propsText = formal.AlloyPropsTemplate(module)
		questions = []string{"structure"}
		run = "  - name: strict\n    constants: {scope: 4}\n    invariants: [ReplaceMe]\n    reach: [NonEmpty]\n    expect: pass\n"
	}
	s.Properties = props
	if p.Exists(s.Path) || p.Exists(s.PropsPath()) {
		return e.Fail(fmt.Errorf("%s or %s exists; formal new never overwrites", s.Path, s.PropsPath()))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "version: 1\nspec: %s\ntool: %s\nareas: [%s]\nquestions: [%s]\n", name, *tool, *area, strings.Join(questions, ", "))
	if *tool == "tla" {
		fmt.Fprintf(&b, "machine: %s\n", s.Machine)
	} else {
		fmt.Fprintf(&b, "model: %s\n", s.Model)
	}
	fmt.Fprintf(&b, "properties: %s\nsources:\n  read_at: %s\n  paths: [%q]\nassumptions: []\nruns:\n%smutations: []\n", props, head, source, run)
	if err := p.WriteAtomic(s.Path, []byte(b.String())); err != nil {
		return e.Fail(err)
	}
	if err := p.WriteAtomic(s.PropsPath(), propsText); err != nil {
		return e.Fail(err)
	}
	fmt.Fprintf(e.Stdout, "formal: wrote %s and %s\nnext: machinery formal gen, then add properties, reach predicates and mutations\n", s.Path, s.PropsPath())
	return 0
}

func runFormalGen(e *Env, args []string) int {
	fs := e.Flags("formal gen")
	check := fs.Bool("check", false, "report drift and write nothing")
	if !e.Parse(fs, args) {
		return 2
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	d := design.Load(p)
	specs, err := formal.Load(p)
	if err != nil {
		return e.Fail(err)
	}
	st := report.New("formal-gen")
	var written []string
	done := map[string]bool{}
	for _, s := range specs {
		if len(s.Problems) > 0 {
			for _, prob := range s.Problems {
				st.Errorf("%s", prob)
			}
			continue
		}
		if done[s.SkeletonPath()] {
			continue
		}
		done[s.SkeletonPath()] = true
		want, serr := formal.Skeleton(d, s)
		if serr != nil {
			st.Errorf("%v", serr)
			continue
		}
		got, rerr := p.Read(s.SkeletonPath())
		if rerr == nil && bytes.Equal(got, want) {
			continue
		}
		if *check {
			if rerr != nil {
				st.Driftf("%s: missing", s.SkeletonPath())
			} else {
				st.Driftf("%s: stale", s.SkeletonPath())
			}
			continue
		}
		if werr := p.WriteAtomic(s.SkeletonPath(), want); werr != nil {
			return e.Fail(werr)
		}
		written = append(written, s.SkeletonPath())
	}
	st.Count(len(done), "skeletons")
	if len(specs) == 0 {
		st.NotApplicable("no formal model")
	}
	if *check || st.Blocking() > 0 {
		return report.PrintAll(e.Stdout, "formal gen", []*report.Stage{st})
	}
	sort.Strings(written)
	fmt.Fprintf(e.Stdout, "formal gen: wrote %d file(s)\n", len(written))
	for _, w := range written {
		fmt.Fprintf(e.Stdout, "  %s\n", w)
	}
	return 0
}

func runFormalCheck(e *Env, args []string) int {
	fs := e.Flags("formal check")
	workers := fs.Int("workers", 2, "TLC worker threads")
	verify := fs.Bool("verify", false, "run the tools, write nothing, and compare with the committed results")
	pos, ok := e.ParseMixed(fs, args)
	if !ok {
		return 2
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	d := design.Load(p)
	all, err := formal.Load(p)
	if err != nil {
		return e.Fail(err)
	}
	specs, err := selectSpecs(all, pos)
	if err != nil {
		return e.Fail(err)
	}
	if *verify {
		return report.PrintAll(e.Stdout, "formal check", []*report.Stage{formal.VerifyStage(p, d, specs, *workers)})
	}
	var runErrs []string
	for _, s := range specs {
		if _, cerr := formal.Check(p, d, s, *workers); cerr != nil {
			var te *formal.ToolError
			if errors.As(cerr, &te) {
				st := report.New("formal").NotDemonstrated(te.Reason)
				return report.PrintAll(e.Stdout, "formal check", []*report.Stage{st})
			}
			runErrs = append(runErrs, cerr.Error())
		}
	}
	// new counterexamples become cx obligations at once
	if _, werr := gen.Write(design.Load(p)); werr != nil {
		return e.Fail(werr)
	}
	j, err := formal.Judge(design.Load(p))
	if err != nil {
		return e.Fail(err)
	}
	st := j.Stage()
	for _, m := range runErrs {
		st.Errorf("%s", m)
	}
	return report.PrintAll(e.Stdout, "formal check", []*report.Stage{st})
}

func runFormalBaseline(e *Env, args []string) int {
	fs := e.Flags("formal baseline")
	accept := fs.Bool("accept", false, "accept weakenings")
	reason := fs.String("reason", "", "why a weakening is accepted")
	if !e.Parse(fs, args) {
		return 2
	}
	if *accept && strings.TrimSpace(*reason) == "" {
		return e.Usage("--accept needs --reason")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	specs, err := formal.Load(p)
	if err != nil {
		return e.Fail(err)
	}
	at := ""
	if head, herr := gitx.Head(p.Root); herr == nil {
		at = gitx.Short(head)
	}
	refused, recorded, err := formal.Record(p, specs, *accept, *reason, at)
	if err != nil {
		return e.Fail(err)
	}
	if len(refused) > 0 {
		st := report.New("formal-baseline").Count(len(specs), "specs")
		for name, ws := range refused {
			for _, w := range ws {
				st.Errorf("weakened: %s: %s; accept with machinery formal baseline --accept --reason", name, w)
			}
		}
		return report.PrintAll(e.Stdout, "formal baseline", []*report.Stage{st})
	}
	fmt.Fprintf(e.Stdout, "baseline: %d spec(s), %d updated\n", len(specs), recorded)
	return 0
}

func runFormalSync(e *Env, args []string) int {
	fs := e.Flags("formal sync")
	class := fs.String("class", "", "no-effect, model-drift, promise-change or possible-bug")
	evidence := fs.String("evidence", "", "what shows the classification is right")
	story := fs.String("story", "", "tracker id of the investigation (possible-bug)")
	pos, ok := e.ParseMixed(fs, args)
	if !ok {
		return 2
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	all, err := formal.Load(p)
	if err != nil {
		return e.Fail(err)
	}
	specs, err := selectSpecs(all, pos)
	if err != nil {
		return e.Fail(err)
	}
	if *class != "" {
		if len(pos) != 1 {
			return e.Usage("machinery formal sync <spec> --class <class>")
		}
		s := specs[0]
		old := s.Sources.ReadAt
		if cerr := formal.Classify(p, s, *class, *evidence, *story); cerr != nil {
			return e.Fail(cerr)
		}
		head, _ := gitx.Head(p.Root)
		if *class == "possible-bug" {
			fmt.Fprintf(e.Stdout, "sync: %s: possible-bug recorded (%s); read_at stays at %s\n", s.Spec, *story, gitx.Short(old))
		} else {
			fmt.Fprintf(e.Stdout, "sync: %s: %s recorded; read_at %s -> %s\n", s.Spec, *class, gitx.Short(old), gitx.Short(head))
		}
		return 0
	}
	code := 0
	for _, s := range specs {
		commits, problem := formal.Drift(p, s)
		switch {
		case problem != "":
			fmt.Fprintf(e.Stdout, "%s: %s\n", s.Spec, problem)
			code = 1
		case len(commits) == 0:
			fmt.Fprintf(e.Stdout, "%s: in sync at %s\n", s.Spec, gitx.Short(s.Sources.ReadAt))
		default:
			fmt.Fprintf(e.Stdout, "%s: %d commit(s) touched its sources since %s\n", s.Spec, len(commits), gitx.Short(s.Sources.ReadAt))
			for _, c := range commits {
				files, _ := gitx.FilesOfCommit(p.Root, c)
				var touched []string
				for _, f := range files {
					if project.Match(s.Sources.Paths, f) {
						touched = append(touched, f)
					}
				}
				fmt.Fprintf(e.Stdout, "  %s  %s\n", gitx.Short(c), strings.Join(touched, " "))
			}
			code = 1
		}
	}
	return code
}

func runFormalTools(e *Env, args []string) int {
	fs := e.Flags("formal tools")
	check := fs.Bool("check", false, "verify only; download nothing")
	if !e.Parse(fs, args) {
		return 2
	}
	root := "."
	configured := ""
	if p, err := project.Load(e.ProjectFlag); err == nil {
		root, configured = p.Root, p.Cfg.Tools.Dir
	}
	dir := project.ToolsDir(configured, root)
	code := 0
	for _, t := range formal.Tools {
		err := t.Verify(dir)
		if err != nil && !*check && !strings.Contains(err.Error(), "checksum mismatch") {
			if ferr := t.Fetch(dir); ferr != nil {
				err = ferr
			} else {
				fmt.Fprintf(e.Stdout, "%s: fetched (%s)\n", t.File, t.Version)
				continue
			}
		}
		if err != nil {
			fmt.Fprintf(e.Stdout, "ERROR  %s\n", err)
			code = 1
			continue
		}
		fmt.Fprintf(e.Stdout, "%s: ok (%s)\n", t.File, t.Version)
	}
	return code
}
