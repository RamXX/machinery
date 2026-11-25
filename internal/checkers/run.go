package checkers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/project"
)

// Outcome is the result of running one checker.
type Outcome struct {
	Checker   string
	DidNotRun string   // why the command did not produce a document
	Problems  []string // invalid evidence, not installed
	Evidence  *Evidence
	Installed bool
}

// OK reports whether the checker ran, produced valid evidence and passed.
func (o *Outcome) OK() bool {
	return o.DidNotRun == "" && len(o.Problems) == 0 && o.Evidence != nil && o.Evidence.Verdict == "pass"
}

// Run runs the command of one checker against the projection (whose bytes are
// on disk at the projection path), validates its evidence and installs it.
func Run(d *design.Design, ck project.Checker, entry Entry, proj *Projection, projBytes []byte) *Outcome {
	p := d.P
	out := &Outcome{Checker: ck.ID}
	dir, err := os.MkdirTemp("", "machinery-checker-")
	if err != nil {
		out.DidNotRun = err.Error()
		return out
	}
	defer os.RemoveAll(dir)
	tmp := filepath.Join(dir, "evidence.json")
	repl := strings.NewReplacer("{projection}", p.Abs(ProjectionPath(d)), "{evidence}", tmp, "{project}", p.Root)
	args := make([]string, len(entry.Command))
	for i, a := range entry.Command {
		args[i] = repl.Replace(a)
	}
	secs := entry.TimeoutSeconds
	if secs == 0 {
		secs = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(secs)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = p.Root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.WaitDelay = 2 * time.Second
	runErr := cmd.Run()
	var exit *exec.ExitError
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		out.DidNotRun = fmt.Sprintf("timed out after %ds", secs)
		return out
	case errors.As(runErr, &exit):
		out.DidNotRun = fmt.Sprintf("exited with status %d", exit.ExitCode())
		if line := firstLine(stderr.String()); line != "" {
			out.DidNotRun += ": " + line
		}
		return out
	case runErr != nil:
		out.DidNotRun = lastPart(runErr)
		return out
	}
	data, err := os.ReadFile(tmp)
	if err != nil {
		out.DidNotRun = "no evidence file was written"
		return out
	}
	ev, stale, problems := Parse(data, ck.ID, SHA256(projBytes), proj.IDs())
	if stale {
		problems = []string{"input_sha256 is not the hash of the projection the checker was given"}
	}
	if len(problems) > 0 {
		out.Problems = problems
		return out
	}
	if err := p.WriteAtomic(ck.Evidence, data); err != nil {
		out.DidNotRun = "cannot install the evidence: " + lastPart(err)
		return out
	}
	out.Evidence, out.Installed = ev, true
	return out
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
