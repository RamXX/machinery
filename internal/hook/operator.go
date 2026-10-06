package hook

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// agentSessionMarkers are environment variables agent hosts set for the tools
// they run. Their presence means a command is running inside an agent session
// rather than in an operator's own terminal.
var agentSessionMarkers = []string{
	"CLAUDECODE", "CLAUDE_CODE_SESSION_ID", "CLAUDE_CODE_ENTRYPOINT", "AI_AGENT",
	"CODEX_SANDBOX", "CODEX_SANDBOX_NETWORK_DISABLED", "CODEX_THREAD_ID", "CODEX_COMPANION_SESSION_ID",
	"OPENCODE", "OPENCODE_SESSION_ID",
}

// operatorGate decides whether a hook-state transition (adopt, release) may
// run. It is a variable only so the package's own tests, which run under an
// agent host, can exercise the transitions; production always uses
// defaultOperatorGate.
var operatorGate = defaultOperatorGate

func defaultOperatorGate() error {
	return requireOperator(os.Environ(), stdinIsTerminal())
}

// requireOperator refuses a hook-state transition that does not come from an
// operator's interactive terminal. An agent tool call carries its host's
// session markers and has no terminal on stdin, so an agent cannot release its
// own tokens or rebind its own store through a tool call. Like the PreToolUse
// guard on the command text, an agent that deliberately scrubs its
// environment and fakes a terminal can evade this; the transitions still
// never clear an obligation and are journaled with what ran them.
func requireOperator(environ []string, interactive bool) error {
	var found []string
	for _, entry := range environ {
		name, value, _ := strings.Cut(entry, "=")
		if value == "" {
			continue
		}
		for _, marker := range agentSessionMarkers {
			if name == marker {
				found = append(found, name)
			}
		}
	}
	if len(found) > 0 {
		sort.Strings(found)
		return fmt.Errorf("machinery hook-state is an operator command and refuses to run inside an agent session (%s is set); run it from your own terminal", strings.Join(found, ", "))
	}
	if !interactive {
		return fmt.Errorf("machinery hook-state is an operator command and requires an interactive terminal on stdin; run it from your own terminal")
	}
	return nil
}

func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
