// Package hook is the harness side of machinery hook (docs/formats.md section
// 21.1): the hook JSON a harness writes on stdin and reads on stdout, the git
// commands found in a shell command line, and the text of the findings.
package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/RamXX/machinery/v2/internal/report"
)

// Input is the part of a harness's hook input machinery reads.
type Input struct {
	SessionID      string          `json:"session_id"`
	Cwd            string          `json:"cwd"`
	HookEventName  string          `json:"hook_event_name"`
	ToolName       string          `json:"tool_name"`
	ToolInput      json.RawMessage `json:"tool_input"`
	StopHookActive bool            `json:"stop_hook_active"`
}

// ParseInput reads the hook input; an error says the bytes are not hook JSON.
func ParseInput(data []byte) (*Input, error) {
	var in Input
	if err := json.Unmarshal(data, &in); err != nil {
		return nil, fmt.Errorf("stdin is not hook JSON: %v", err)
	}
	return &in, nil
}

// Command returns tool_input.command, "" when there is none.
func (in *Input) Command() (string, error) {
	if len(in.ToolInput) == 0 {
		return "", nil
	}
	var ti struct {
		Command *string `json:"command"`
	}
	if err := json.Unmarshal(in.ToolInput, &ti); err != nil {
		return "", fmt.Errorf("tool_input.command is not a string: %v", err)
	}
	if ti.Command == nil {
		return "", nil
	}
	return *ti.Command, nil
}

var sessionRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// SafeSession reports whether a session id may name a file and an
// environment value as it is.
func SafeSession(id string) bool { return sessionRe.MatchString(id) }

// Output is the hook output machinery writes.
type Output struct {
	Decision      string    `json:"decision,omitempty"`
	Reason        string    `json:"reason,omitempty"`
	SystemMessage string    `json:"systemMessage,omitempty"`
	Specific      *Specific `json:"hookSpecificOutput,omitempty"`
}

// Specific is hookSpecificOutput.
type Specific struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
	AdditionalContext        string `json:"additionalContext,omitempty"`
}

// Context adds context for the model at an event.
func Context(event, text string) *Output {
	return &Output{Specific: &Specific{HookEventName: event, AdditionalContext: text}}
}

// Deny refuses a tool call; the reason is shown to the model.
func Deny(reason string) *Output {
	return &Output{Specific: &Specific{HookEventName: "PreToolUse", PermissionDecision: "deny", PermissionDecisionReason: reason}}
}

// Block continues the turn at a stop; the reason is shown to the model.
func Block(reason string) *Output { return &Output{Decision: "block", Reason: reason} }

// Message is shown to the user and changes nothing.
func Message(text string) *Output { return &Output{SystemMessage: text} }

// Write prints the output as one JSON object on one line.
func Write(w io.Writer, out *Output) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(out)
}

// MaxFindings is how many findings a hook text lists.
const MaxFindings = 40

// Text renders a header, the first MaxFindings findings and a count of the
// rest, which the more line tells how to see.
func Text(header string, findings []string, more string) string {
	lines := []string{header}
	if len(findings) > MaxFindings {
		lines = append(lines, findings[:MaxFindings]...)
		lines = append(lines, fmt.Sprintf("%d more finding(s): %s", len(findings)-MaxFindings, more))
	} else {
		lines = append(lines, findings...)
	}
	return strings.Join(lines, "\n")
}

// Cap keeps at most max lines; past it, the first max-1 and a count of the
// rest, which the more line tells how to see.
func Cap(lines []string, max int, more string) []string {
	if len(lines) <= max {
		return lines
	}
	return append(append([]string{}, lines[:max-1]...), fmt.Sprintf("%d more line(s): %s", len(lines)-max+1, more))
}

// StageFindings lists what blocks in a stage: each ERROR and DRIFT line of a
// stage that failed, or the reason of a stage that was not demonstrated.
func StageFindings(s *report.Stage) []string {
	switch s.Outcome() {
	case report.Failed:
		var res []string
		for _, f := range s.Sorted() {
			if f.Sev != report.Warn {
				res = append(res, fmt.Sprintf("%s: %s  %s", s.Name, report.Label(f.Sev), f.Msg))
			}
		}
		return res
	case report.NotDemonstrated:
		reason := s.ND
		if reason == "" {
			reason = "nothing to check"
		}
		return []string{fmt.Sprintf("%s: not demonstrated: %s", s.Name, reason)}
	}
	return nil
}

// GitVerbs reports whether a shell command line runs git commit or git push.
func GitVerbs(line string) (commit, push bool) {
	for _, words := range simpleCommands(line) {
		switch gitSubcommand(words) {
		case "commit":
			commit = true
		case "push":
			push = true
		}
	}
	return commit, push
}

// simpleCommands splits a command line into the words of its simple
// commands, at ; & | newlines, parentheses and backquotes outside quotes.
func simpleCommands(s string) [][]string {
	var cmds [][]string
	var words []string
	var cur strings.Builder
	inWord := false
	var quote rune
	endWord := func() {
		if inWord {
			words = append(words, cur.String())
			cur.Reset()
			inWord = false
		}
	}
	endCommand := func() {
		endWord()
		if len(words) > 0 {
			cmds = append(cmds, words)
			words = nil
		}
	}
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, inWord = r, true
		case r == ' ' || r == '\t':
			endWord()
		case strings.ContainsRune(";&|\n()`", r):
			endCommand()
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	endCommand()
	return cmds
}

var assignRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

var wrappers = map[string]bool{"env": true, "command": true, "exec": true, "nohup": true, "time": true, "sudo": true}

// gitOptsWithValue are git's own options that take the next word.
var gitOptsWithValue = map[string]bool{"-C": true, "-c": true, "--git-dir": true, "--work-tree": true, "--namespace": true, "--super-prefix": true}

// gitSubcommand returns the git subcommand a simple command runs, or "".
func gitSubcommand(words []string) string {
	i := 0
	skipAssignments := func() {
		for i < len(words) && assignRe.MatchString(words[i]) {
			i++
		}
	}
	skipAssignments()
	if i < len(words) && wrappers[words[i]] {
		i++
		skipAssignments()
	}
	if i >= len(words) || path.Base(words[i]) != "git" {
		return ""
	}
	for i++; i < len(words) && strings.HasPrefix(words[i], "-"); i++ {
		if gitOptsWithValue[words[i]] {
			i++
		}
	}
	if i >= len(words) {
		return ""
	}
	return words[i]
}
