package formal

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// tlcOutcome is what one TLC invocation established.
type tlcOutcome struct {
	Outcome  string // passed | violated | error
	Violated string // property name, "Deadlock", or the first error line
	Kind     string // invariant | property | action | deadlock
	States   []tlcState
	Loop     int
}

type tlcState struct {
	Stuttering bool
	Vars       map[string]string
}

func tlaValue(v any) string {
	switch x := v.(type) {
	case bool:
		if x {
			return "TRUE"
		}
		return "FALSE"
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.Itoa(int(x))
	case string:
		return strconv.Quote(x)
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = tlaValue(e)
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return fmt.Sprint(v)
}

func intOf(v any, def int) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	}
	return def
}

func nameSet(prefix string, n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf("%q", prefix+strconv.Itoa(i+1))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// tlcConfig renders a TLC configuration file for a run.
func tlcConfig(r *Run, invariants, properties []string) string {
	var b strings.Builder
	spec := r.Spec
	if spec == "" {
		spec = "Spec"
	}
	fmt.Fprintf(&b, "SPECIFICATION %s\nCONSTANTS\n", spec)
	consts := map[string]any{"Instances": 1, "Callers": 2, "Atomic": true}
	for k, v := range r.Constants {
		consts[k] = v
	}
	var keys []string
	for k := range consts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		switch k {
		case "Instances":
			fmt.Fprintf(&b, "  Instances = %s\n", nameSet("i", intOf(consts[k], 1)))
		case "Callers":
			fmt.Fprintf(&b, "  Callers = %s\n", nameSet("c", intOf(consts[k], 2)))
		default:
			fmt.Fprintf(&b, "  %s = %s\n", k, tlaValue(consts[k]))
		}
	}
	if len(invariants) > 0 {
		b.WriteString("INVARIANT\n")
		for _, i := range invariants {
			fmt.Fprintf(&b, "  %s\n", i)
		}
	}
	if len(properties) > 0 {
		b.WriteString("PROPERTY\n")
		for _, p := range properties {
			fmt.Fprintf(&b, "  %s\n", p)
		}
	}
	return b.String()
}

var (
	invViolRe  = regexp.MustCompile(`Error: Invariant (\S+) is violated`)
	actViolRe  = regexp.MustCompile(`Error: Action property (\S+) is violated`)
	stateHdrRe = regexp.MustCompile(`^State ([0-9]+): (.*)$`)
	backRe     = regexp.MustCompile(`^Back to state ([0-9]+)`)
	fieldRe    = regexp.MustCompile(`(\w+) \|-> "([^"]*)"`)
)

type tlcRunner struct {
	java    string
	jar     string
	workers int
}

// run executes TLC on a module in dir with the given config text.
func (t *tlcRunner) run(dir, module, cfg string, deadlock bool, timeout time.Duration) tlcOutcome {
	cfgPath := filepath.Join(dir, "run.cfg")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		return tlcOutcome{Outcome: "error", Violated: err.Error()}
	}
	meta, err := os.MkdirTemp(dir, "meta-")
	if err != nil {
		return tlcOutcome{Outcome: "error", Violated: err.Error()}
	}
	defer os.RemoveAll(meta)
	args := []string{"-XX:+UseParallelGC", "-cp", t.jar, "tlc2.TLC", "-workers", strconv.Itoa(t.workers), "-metadir", meta, "-config", "run.cfg"}
	if !deadlock {
		args = append(args, "-deadlock")
	}
	args = append(args, module)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, t.java, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	runErr := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return tlcOutcome{Outcome: "error", Violated: fmt.Sprintf("timed out after %s", timeout)}
	}
	return parseTLC(out.String(), runErr)
}

func parseTLC(out string, runErr error) tlcOutcome {
	res := tlcOutcome{}
	lines := strings.Split(out, "\n")
	switch {
	case invViolRe.MatchString(out):
		res.Outcome, res.Kind = "violated", "invariant"
		res.Violated = strings.TrimSuffix(invViolRe.FindStringSubmatch(out)[1], ".")
	case actViolRe.MatchString(out):
		res.Outcome, res.Kind = "violated", "action"
		res.Violated = strings.TrimSuffix(actViolRe.FindStringSubmatch(out)[1], ".")
	case strings.Contains(out, "Error: Temporal properties were violated"):
		res.Outcome, res.Kind = "violated", "property"
	case strings.Contains(out, "Error: Deadlock reached"):
		res.Outcome, res.Kind, res.Violated = "violated", "deadlock", "Deadlock"
	case strings.Contains(out, "Model checking completed. No error has been found."):
		res.Outcome = "passed"
		return res
	default:
		res.Outcome = "error"
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "Error:") || strings.HasPrefix(l, "***") || strings.Contains(l, "Semantic error") || strings.Contains(l, "Parsing or semantic analysis failed") || strings.HasPrefix(l, "Was expecting") || strings.HasPrefix(l, "Encountered") {
				res.Violated = l
				break
			}
		}
		if res.Violated == "" {
			res.Violated = "the model checker did not finish"
			if runErr != nil {
				res.Violated += ": " + runErr.Error()
			}
		}
		return res
	}
	// trace
	var cur *tlcState
	var curVar string
	flush := func() {
		if cur != nil {
			res.States = append(res.States, *cur)
			cur = nil
		}
	}
	for _, l := range lines {
		if m := stateHdrRe.FindStringSubmatch(l); m != nil {
			flush()
			cur = &tlcState{Vars: map[string]string{}, Stuttering: strings.HasPrefix(m[2], "Stuttering")}
			curVar = ""
			continue
		}
		if m := backRe.FindStringSubmatch(l); m != nil {
			res.Loop, _ = strconv.Atoi(m[1])
			continue
		}
		if cur == nil {
			continue
		}
		if strings.HasPrefix(l, "/\\ ") {
			rest := strings.TrimPrefix(l, "/\\ ")
			if i := strings.Index(rest, " = "); i > 0 {
				curVar = rest[:i]
				cur.Vars[curVar] = rest[i+3:]
			}
			continue
		}
		if strings.TrimSpace(l) == "" {
			flush()
			continue
		}
		if curVar != "" {
			cur.Vars[curVar] += " " + strings.TrimSpace(l)
		}
	}
	flush()
	return res
}

// steps turns parsed states into trace steps (section 16.10).
func (o tlcOutcome) steps() []Step {
	var steps []Step
	for i, s := range o.States {
		if i == 0 || s.Stuttering {
			continue
		}
		last := map[string]string{}
		for _, m := range fieldRe.FindAllStringSubmatch(s.Vars["last"], -1) {
			last[m[1]] = m[2]
		}
		state := map[string]string{}
		for _, m := range fieldRe.FindAllStringSubmatch(s.Vars["st"], -1) {
			state[m[1]] = m[2]
		}
		steps = append(steps, Step{N: len(steps) + 1, Case: last["id"], Instance: last["i"], Caller: last["c"], Step: last["step"], State: state})
	}
	return steps
}

// loopStep maps TLC's "Back to state N" to a step number (state 1 is the initial state).
func (o tlcOutcome) loopStep() int {
	if o.Loop <= 1 {
		return 0
	}
	return o.Loop - 1
}
