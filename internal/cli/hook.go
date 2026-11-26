package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/hook"
	"github.com/RamXX/machinery/internal/landing"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/stages"
)

func init() {
	Register(&Command{Name: "hook", Summary: "Answer a harness hook (session-start, pre-tool, stop): hook JSON on stdin and stdout.", Run: runHook})
}

// runHook always exits 0: a hook never fails the harness (section 21.1).
func runHook(e *Env, args []string) (code int) {
	name := "machinery hook"
	if len(args) > 0 {
		name += " " + args[0]
	}
	defer func() {
		if r := recover(); r != nil {
			hook.Write(e.Stdout, hook.Message(fmt.Sprintf("%s could not run: internal error: %v", name, r)))
			code = 0
		}
	}()
	out, err := hookEvent(e, args)
	if err != nil {
		var ce *project.ConfigError
		msg := err.Error()
		if errors.As(err, &ce) {
			msg = "config: " + ce.Msg
		}
		out = hook.Message(name + " could not run: " + msg)
	}
	if out != nil {
		hook.Write(e.Stdout, out)
	}
	return 0
}

func hookEvent(e *Env, args []string) (*hook.Output, error) {
	if len(args) == 0 {
		return nil, errors.New("name the event: session-start, pre-tool or stop")
	}
	handlers := map[string]func(*Env, *hook.Input) (*hook.Output, error){
		"session-start": hookSessionStart, "pre-tool": hookPreTool, "stop": hookStop,
	}
	h, ok := handlers[args[0]]
	switch {
	case !ok:
		return nil, errors.New("unknown event; the events are session-start, pre-tool and stop")
	case len(args) > 1:
		return nil, errors.New("takes one event and its input on stdin")
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("cannot read stdin: %v", err)
	}
	in, err := hook.ParseInput(data)
	if err != nil {
		return nil, err
	}
	return h(e, in)
}

// hookProject finds the project from -C, else the input's cwd, else the
// working directory. nil without an error: not in a machinery project.
func hookProject(e *Env, in *hook.Input) (*project.Project, error) {
	var root string
	if e.ProjectFlag != "" {
		if _, err := os.Stat(filepath.Join(e.ProjectFlag, project.ConfigName)); err != nil {
			return nil, nil
		}
		root = e.ProjectFlag
	} else {
		dir := in.Cwd
		if dir == "" {
			dir = "."
		}
		r, err := project.Find(dir)
		if err != nil {
			return nil, nil
		}
		root = r
	}
	return project.Load(root)
}

// summaryPrefixes are the lines of status a session starts with (section 21.1).
var summaryPrefixes = []string{"red: ", "ids: ", "proofs awaiting: ", "intake ", "questions: "}

func hookSessionStart(e *Env, in *hook.Input) (*hook.Output, error) {
	p, err := hookProject(e, in)
	if err != nil || p == nil {
		return nil, err
	}
	var out, errb bytes.Buffer
	if runStatus(&Env{Stdout: &out, Stderr: &errb, ProjectFlag: p.Root}, nil) != 0 {
		return nil, errors.New(strings.TrimSpace(strings.SplitN(errb.String(), "\n", 2)[0]))
	}
	lines := []string{fmt.Sprintf("machinery: %s is a machinery project; machinery status prints the full summary.", filepath.Base(p.Root))}
	for _, l := range strings.Split(out.String(), "\n") {
		for _, pre := range summaryPrefixes {
			if strings.HasPrefix(l, pre) {
				lines = append(lines, l)
				break
			}
		}
	}
	lines = hook.Cap(lines, 30, "run machinery status")
	if hook.SafeSession(in.SessionID) && gitx.IsRepo(p.Root) {
		if err := recordSession(p, in.SessionID); err != nil {
			return nil, err
		}
		if env := os.Getenv("CLAUDE_ENV_FILE"); env != "" {
			if err := appendLine(env, "export MACHINERY_SESSION="+in.SessionID); err != nil {
				return nil, fmt.Errorf("CLAUDE_ENV_FILE: %v", err)
			}
		}
	}
	return hook.Context("SessionStart", strings.Join(lines, "\n")), nil
}

// sessionRecord is <git common dir>/machinery/sessions/<id>.json.
type sessionRecord struct {
	Version int    `json:"version"`
	Session string `json:"session"`
	Head    string `json:"head"`
}

func sessionPath(p *project.Project, id string) (string, error) {
	dir, err := gitx.CommonDir(p.Root)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "machinery", "sessions", id+".json"), nil
}

// recordSession writes the start of a session unless it is recorded already.
func recordSession(p *project.Project, id string) error {
	path, err := sessionPath(p, id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	head, _ := gitx.Head(p.Root)
	sid, _ := json.Marshal(id)
	sha, _ := json.Marshal(head)
	return project.WriteFileAtomic(path, []byte(fmt.Sprintf("{\"version\": 1, \"session\": %s, \"head\": %s}\n", sid, sha)))
}

func appendLine(path, line string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(f, line); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func hookPreTool(e *Env, in *hook.Input) (*hook.Output, error) {
	if in.ToolName != "Bash" {
		return nil, nil
	}
	line, err := in.Command()
	if err != nil {
		return nil, err
	}
	commit, push := hook.GitVerbs(line)
	if !commit && !push {
		return nil, nil
	}
	p, err := hookProject(e, in)
	if err != nil || p == nil {
		return nil, err
	}
	verb := "commit"
	if push {
		verb = "push"
	}
	mode := p.Cfg.HookMode(verb)
	if mode == project.HookOff {
		return nil, nil
	}
	base := "main"
	if p.Cfg.Backlog != nil {
		base = p.Cfg.Backlog.Base
	}
	var findings []string
	if push {
		red, err := landing.LoadRed(p)
		if err != nil {
			return nil, err
		}
		if red != nil {
			findings = append(findings, redFinding(p, red, base))
		}
	}
	findings = append(findings, hookCheck(p, base)...)
	if len(findings) == 0 {
		return nil, nil
	}
	check := "machinery check --base " + base
	if mode == project.HookBlock {
		header := fmt.Sprintf("machinery: git %s refused: %s found %d finding(s)", verb, check, len(findings))
		return hook.Deny(hook.Text(header, findings, "run "+check)), nil
	}
	header := fmt.Sprintf("machinery: git %s goes ahead (hooks.%s: warn): %s found %d finding(s)", verb, verb, check, len(findings))
	return hook.Context("PreToolUse", hook.Text(header, findings, "run "+check)), nil
}

// redFinding names the red marker and the repair path (section 21.1).
func redFinding(p *project.Project, red *landing.Red, base string) string {
	marker, _ := landing.RedPath(p)
	if rel, err := filepath.Rel(p.Root, marker); err == nil {
		marker = filepath.ToSlash(rel)
	}
	return fmt.Sprintf("red: the integration branch is red since %s; marker %s; repair: fix forward and run machinery landing --base %s --repair, or revert and run machinery landing --clear-red --reason \"<what was reverted>\"", red.What(), marker, base)
}

// hookCheck runs the stages of machinery check, with the ratchet against base
// when base is not empty, and lists what blocks.
func hookCheck(p *project.Project, base string) []string {
	names := append([]string{}, stages.CheckStages...)
	ctx := &stages.Context{P: p}
	var tail []string
	if base != "" {
		if _, err := gitx.Head(p.Root); err == nil {
			if _, err := gitx.Resolve(p.Root, base); err == nil {
				names, ctx.Base = append(names, "ratchet"), base
			} else {
				tail = append(tail, "ratchet: not demonstrated: base "+base+" is not a commit; set backlog.base to the integration branch")
			}
		}
	}
	var findings []string
	for _, s := range stages.Run(ctx, names) {
		findings = append(findings, hook.StageFindings(s)...)
	}
	return append(findings, tail...)
}

func hookStop(e *Env, in *hook.Input) (*hook.Output, error) {
	p, err := hookProject(e, in)
	if err != nil || p == nil {
		return nil, err
	}
	mode := p.Cfg.HookMode("stop")
	if mode == project.HookOff {
		return nil, nil
	}
	changed, err := changedThisSession(p, in.SessionID)
	if err != nil || !changed {
		return nil, err
	}
	findings := hookCheck(p, "")
	if len(findings) == 0 {
		return nil, nil
	}
	text := hook.Text(fmt.Sprintf("machinery: machinery check found %d finding(s) in the files changed this session", len(findings)), findings, "run machinery check")
	if mode == project.HookBlock && !in.StopHookActive {
		return hook.Block(text), nil
	}
	return hook.Message(text), nil
}

// changedThisSession reports whether git status is not empty, or HEAD moved
// since the session's recorded start.
func changedThisSession(p *project.Project, id string) (bool, error) {
	paths, err := gitx.StatusPaths(p.Root)
	if err != nil {
		return false, fmt.Errorf("git status: %v", err)
	}
	if len(paths) > 0 {
		return true, nil
	}
	if !hook.SafeSession(id) {
		return false, nil
	}
	path, err := sessionPath(p, id)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var rec sessionRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return false, fmt.Errorf("%s: %v", filepath.Base(path), err)
	}
	head, _ := gitx.Head(p.Root)
	return head != rec.Head, nil
}
