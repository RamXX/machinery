package hook

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Boundary events are host notices that a lane or a whole session can no
// longer complete a tool call it started: the user interrupted the turn, the
// next prompt arrived, a tool batch resolved, or the session ended. Hosts do
// not send PostToolUse or PostToolUseFailure for a cancelled or manually
// denied call, so without these notices that call's token would stay in
// flight forever and block every later Stop of its own session.
//
// A boundary closes tokens exactly as a failed completion does: the token
// goes, the project design/impl obligation stays armed, and the next Stop
// runs the gates. A boundary never discharges an obligation and never runs
// or skips a gate.
var boundaryEvents = map[string]bool{
	"UserPromptSubmit": true, // Claude Code and Codex: a new turn starts in the main lane
	"Interrupt":        true, // Codex: the user interrupted the main lane
	"PostToolBatch":    true, // Claude Code: every call of this lane's batch has resolved
	"SessionEnd":       true, // Claude Code and Codex: nothing of this session runs any more
}

// decodeBoundaryInput recognizes a boundary event and reads only the fields
// that select which tokens it closes. These events carry host-specific
// payloads (prompt text, tool results, end reasons) that machinery never
// routes on, so unlike the strict tool and Stop decoder it ignores fields it
// does not use: a new host field must never turn an interrupt notice into a
// stranded token. A payload that is not a boundary event is left to the
// strict decoder.
func decodeBoundaryInput(raw []byte) (Input, bool, error) {
	event := boundaryEventName(raw)
	if !boundaryEvents[event] {
		return Input{}, false, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return Input{}, true, err
	}
	in := Input{HookEventName: event}
	for key, target := range map[string]*string{"session_id": &in.SessionID, "agent_id": &in.AgentID, "cwd": &in.Cwd} {
		value, ok := fields[key]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			continue
		}
		if err := json.Unmarshal(value, target); err != nil {
			return Input{}, true, fmt.Errorf("boundary event field %q must be a string", key)
		}
	}
	if in.SessionID == "" {
		return Input{}, true, fmt.Errorf("boundary event %s has no session_id; it cannot be bound to the tokens it would close", event)
	}
	return in, true, nil
}

// boundaryEventName reads only the event name; anything unreadable is not a
// boundary event and goes to the strict decoder, which reports it.
func boundaryEventName(raw []byte) string {
	var probe struct {
		HookEventName string `json:"hook_event_name"`
	}
	if json.Unmarshal(raw, &probe) != nil {
		return ""
	}
	return probe.HookEventName
}

// boundaryCloses reports whether a boundary event of in proves that the token
// owned by owner can no longer complete. Tokens without a recorded owner are
// never closed here: nothing proves which session they belong to.
func boundaryCloses(in Input, owner pendingOwner) bool {
	if !owner.known() {
		return false
	}
	if in.HookEventName == "SessionEnd" {
		return owner.session == hookSessionDigest(in.SessionID)
	}
	return owner.lane == hookLaneDigest(in.SessionID, in.AgentID)
}

func runBoundary(in Input, root string) (retErr error) {
	if root == "" {
		root = os.Getenv("CLAUDE_PROJECT_DIR")
	}
	if root == "" {
		root = in.Cwd
	}
	if root == "" {
		root = "."
	}
	root, err := canonicalHookRoot(resolveEventPath(in.Cwd, root))
	if err != nil {
		return fmt.Errorf("machinery hook: %s cannot resolve a canonical project root: %w", in.HookEventName, err)
	}
	stateFile, err := statePathExact(root)
	if err != nil {
		return nil // no addressable store, so no token to close
	}
	if _, err := os.Lstat(stateFile); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := requireStateDir(); err != nil {
		return fmt.Errorf("machinery hook: %s cannot access the durable project-obligation store: %w", in.HookEventName, err)
	}
	lock, err := acquireHookStateLock(stateFile)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, lock.release()) }()
	err = mutateStateLocked(stateFile, root, false, func(record *hookStateRecord) bool {
		closed := false
		for _, token := range append([]string(nil), record.pending...) {
			if boundaryCloses(in, record.pendingOwners[token]) {
				closed = record.removePending(token) || closed
			}
		}
		return closed
	})
	if err != nil {
		return fmt.Errorf("machinery hook: %s could not close this session's stranded tool tokens: %w", in.HookEventName, err)
	}
	return nil
}

// partitionPending splits a ledger's in-flight tokens into those owned by the
// session of in, which still block its Stop, and orphaned tokens armed by
// another session or recorded before owners were (those carry no owner).
// The Stop of one session cannot complete another session's tool call, so an
// orphaned token never blocks it; it keeps the project obligation armed.
func partitionPending(record hookStateRecord, in Input) (own, orphaned []string) {
	session := hookSessionDigest(in.SessionID)
	for _, token := range record.pending {
		if owner := record.pendingOwners[token]; owner.known() && owner.session == session {
			own = append(own, token)
		} else {
			orphaned = append(orphaned, token)
		}
	}
	return own, orphaned
}

func shortTokens(tokens []string) string {
	const shown = 4
	var b bytes.Buffer
	for i, token := range tokens {
		if i == shown {
			fmt.Fprintf(&b, ", and %d more", len(tokens)-shown)
			break
		}
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(token[:12])
	}
	return b.String()
}

func releaseCommand(root string) string {
	return "machinery hook-state release --root " + shellQuote(root) + " --orphaned"
}

func orphanedTokenNote(root string, orphaned []string, plain bool) string {
	if plain {
		return fmt.Sprintf("machinery: an earlier or parallel session left %d unfinished tool operation(s); the design checks rerun at every turn end until they finish or an operator releases them with '%s'.", len(orphaned), releaseCommand(root))
	}
	return fmt.Sprintf("machinery: %d tool operation(s) armed by another or an ended session never reported completion (%s). They do not block this session, but the project gate obligation stays armed and the gates run again at every Stop until they complete or an operator releases them. If no agent session in this project is still running a tool, run: %s", len(orphaned), shortTokens(orphaned), releaseCommand(root))
}

// narrowedRoutes compares the current configuration with every earlier route
// the ledger records whose snapshot is still readable, and names each way the
// current one checks less: another design or implementation tree, the
// implementation tree dropped, strict mode dropped, or staged gates removed.
// A route whose snapshot retention already reclaimed cannot be compared.
func narrowedRoutes(root string, routes []string, current string, cfg Config) ([]string, error) {
	paths, err := routeStatePaths(root)
	if err != nil {
		return nil, err
	}
	recorded := map[string]bool{}
	for _, route := range routes {
		if route != current {
			recorded[route] = true
		}
	}
	seen := map[string]bool{}
	var narrowed []string
	add := func(reason string) {
		if !seen[reason] {
			seen[reason] = true
			narrowed = append(narrowed, reason)
		}
	}
	for _, path := range paths {
		raw, err := readRouteStateFile(path)
		if err != nil {
			return nil, err
		}
		if raw == nil || !recorded[routeSnapshotDigest(raw)] {
			continue
		}
		earlier, err := decodeConfig(raw)
		if err != nil {
			return nil, fmt.Errorf("route snapshot %s is corrupt: %w", path, err)
		}
		if earlier.Design == "" {
			earlier.Design = "design"
		}
		if earlier.Design != cfg.Design {
			add(fmt.Sprintf("design directory %q is now %q", earlier.Design, cfg.Design))
		}
		switch {
		case earlier.Impl != "" && cfg.Impl == "":
			add(fmt.Sprintf("implementation tree %q is no longer checked", earlier.Impl))
		case earlier.Impl != "" && earlier.Impl != cfg.Impl:
			add(fmt.Sprintf("implementation tree %q is now %q", earlier.Impl, cfg.Impl))
		}
		if earlier.Strict && !cfg.Strict {
			add("strict mode was switched off")
		}
		if cfg.Gates != "" {
			now := map[string]bool{}
			for _, gate := range strings.Split(strings.ToLower(cfg.Gates), ",") {
				now[strings.TrimSpace(gate)] = true
			}
			if earlier.Gates == "" {
				add("the progressive gate selection became the staged list " + cfg.Gates)
			} else {
				for _, gate := range strings.Split(strings.ToLower(earlier.Gates), ",") {
					if gate = strings.TrimSpace(gate); gate != "" && !now[gate] {
						add("staged gate " + gate + " was removed")
					}
				}
			}
		}
	}
	return narrowed, nil
}
