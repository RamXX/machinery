// Package landing is the end gate (docs/formats.md section 12.2): check with
// the ratchet, the whole suite through run, then verify.
package landing

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/RamXX/machinery/internal/evidence"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/stages"
)

// MaxFixForward is how many fix-forwards a landing gets before it must stop.
const MaxFixForward = 2

// Options configures Run.
type Options struct {
	Base           string
	Stack          string
	Command        []string // after --; empty means commands.e2e
	Stdout, Stderr io.Writer
}

// Attempt is the count of one landing attempt.
type Attempt struct{ N, FixForward int }

type record struct {
	Attempts   int    `json:"attempts"`
	FixForward int    `json:"fix_forward"`
	Head       string `json:"head"`
	Outcome    string `json:"outcome"`
}

type store struct {
	Version  int               `json:"version"`
	Branches map[string]record `json:"branches"`
}

func storePath(p *project.Project) (string, error) {
	dir, err := gitx.CommonDir(p.Root)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "machinery", "landing.json"), nil
}

func load(path string) (*store, error) {
	st := &store{Version: 1, Branches: map[string]record{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return st, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, st); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	if st.Branches == nil {
		st.Branches = map[string]record{}
	}
	return st, nil
}

func branchKey(p *project.Project) string {
	if b := gitx.Branch(p.Root); b != "" {
		return b
	}
	return "(detached)"
}

// Begin counts the attempt about to run at head: an attempt at a new head
// after a failed one is a fix-forward.
func Begin(p *project.Project, head string) (Attempt, error) {
	path, err := storePath(p)
	if err != nil {
		return Attempt{}, err
	}
	st, err := load(path)
	if err != nil {
		return Attempt{}, err
	}
	rec, ok := st.Branches[branchKey(p)]
	if !ok {
		return Attempt{N: 1}, nil
	}
	a := Attempt{N: rec.Attempts + 1, FixForward: rec.FixForward}
	if rec.Outcome == "failed" && rec.Head != head {
		a.FixForward++
	}
	return a, nil
}

// Finish records the outcome; a passed landing resets the count.
func Finish(p *project.Project, a Attempt, head string, passed bool) error {
	path, err := storePath(p)
	if err != nil {
		return err
	}
	st, err := load(path)
	if err != nil {
		return err
	}
	key := branchKey(p)
	if passed {
		delete(st.Branches, key)
	} else {
		st.Branches[key] = record{Attempts: a.N, FixForward: a.FixForward, Head: head, Outcome: "failed"}
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return project.WriteFileAtomic(path, append(data, '\n'))
}

// Run is machinery landing. It returns the exit code; err is a config or
// usage error (exit 2) found before anything ran.
func Run(p *project.Project, o Options) (exit int, err error) {
	argv := o.Command
	if len(argv) == 0 {
		argv = p.Cfg.Commands.E2E
	}
	if len(argv) == 0 {
		return 2, project.Errf("no end-to-end command: pass one after -- or set commands.e2e in %s", project.ConfigName)
	}
	head, err := gitx.Head(p.Root)
	if err != nil {
		return 1, fmt.Errorf("cannot read the current commit (landing needs a git repository with a commit): %v", err)
	}
	att, err := Begin(p, head)
	if err != nil {
		return 1, err
	}
	passed := gate(p, o, argv)
	if err := Finish(p, att, head, passed); err != nil {
		return 1, err
	}
	state := "failed"
	if passed {
		state = "passed"
	}
	fmt.Fprintf(o.Stdout, "landing: %s at %s (attempt %d, fix-forward %d of %d)\n", state, gitx.Short(head), att.N, att.FixForward, MaxFixForward)
	if !passed && att.FixForward > MaxFixForward {
		fmt.Fprintln(o.Stdout, "stop: two fix-forwards used; return the failing stories to the backlog")
	}
	if !passed {
		return 1, nil
	}
	return 0, nil
}

// gate runs the three steps in order and stops at the first that fails.
func gate(p *project.Project, o Options, argv []string) bool {
	names := append(append([]string{}, stages.CheckStages...), "ratchet")
	if report.PrintAll(o.Stdout, "check", stages.Run(&stages.Context{P: p, Base: o.Base}, names)) != 0 {
		return false
	}
	exit, file, err := evidence.Run(p, evidence.RunOptions{Stack: o.Stack, Stdout: o.Stdout, Stderr: o.Stderr}, argv)
	if err != nil {
		fmt.Fprintf(o.Stderr, "machinery: run: %v\n", err)
		return false
	}
	fmt.Fprintf(o.Stderr, "machinery run: %s (exit %d)\n", p.Rel(file), exit)
	if exit != 0 {
		return false
	}
	ctx := &stages.Context{P: p}
	return report.PrintAll(o.Stdout, "verify", []*report.Stage{stages.Verify(ctx, false)}) == 0
}
