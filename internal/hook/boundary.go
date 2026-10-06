package hook

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
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
type boundaryInput struct {
	Input
	// batchToolUseIDs are the exact calls a PostToolBatch reports resolved.
	batchToolUseIDs []string
}

func decodeBoundaryInput(raw []byte) (boundaryInput, bool, error) {
	fields, scanErr := scanHookObject(raw)
	var event string
	if rawEvent, ok := fields["hook_event_name"]; ok && json.Unmarshal(rawEvent, &event) == nil && boundaryEvents[event] {
		// a boundary event: from here every ambiguity is refused
	} else if scanErr == nil || !mentionsBoundaryEvent(raw) {
		return boundaryInput{}, false, nil // the strict decoder owns it
	}
	if scanErr != nil {
		return boundaryInput{}, true, scanErr
	}
	// The routing fields go through the enforcing decoder itself, so a
	// boundary reads exactly the session, agent, and cwd values that
	// PreToolUse recorded its owners from. Every routing value must be a
	// canonical non-null JSON string.
	routing := map[string]json.RawMessage{}
	for _, key := range []string{"hook_event_name", "session_id", "agent_id", "cwd"} {
		value, ok := fields[key]
		if !ok {
			continue
		}
		if err := requireJSONString(key, value); err != nil {
			return boundaryInput{}, true, err
		}
		routing[key] = value
	}
	routingRaw, err := json.Marshal(routing)
	if err != nil {
		return boundaryInput{}, true, err
	}
	in, err := decodeInput(bytes.NewReader(routingRaw))
	if err != nil {
		return boundaryInput{}, true, err
	}
	if in.HookEventName != event {
		return boundaryInput{}, true, fmt.Errorf("boundary event name did not survive canonical decoding")
	}
	if strings.TrimSpace(in.SessionID) == "" {
		return boundaryInput{}, true, fmt.Errorf("boundary event %s has no session_id; it cannot be bound to the tokens it would close", event)
	}
	if _, present := fields["agent_id"]; present && strings.TrimSpace(in.AgentID) == "" {
		return boundaryInput{}, true, fmt.Errorf("boundary event %s carries an empty agent_id; refusing to read it as the main thread", event)
	}
	batch := boundaryInput{Input: in}
	if event == "PostToolBatch" {
		ids, err := decodeBatchToolUseIDs(fields["tool_calls"])
		if err != nil {
			return boundaryInput{}, true, err
		}
		batch.batchToolUseIDs = ids
	} else if _, present := fields["tool_calls"]; present {
		return boundaryInput{}, true, fmt.Errorf("boundary event %s must not carry tool_calls", event)
	}
	return batch, true, nil
}

// requireJSONString accepts only a JSON string token, never null, a number,
// or a container, so no decoder can read a different value from it.
func requireJSONString(key string, raw json.RawMessage) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '"' {
		return fmt.Errorf("hook-event key %q must be a JSON string", key)
	}
	var value string
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return fmt.Errorf("hook-event key %q must be a JSON string: %w", key, err)
	}
	return nil
}

// decodeBatchToolUseIDs reads a PostToolBatch call list strictly: an array of
// objects, each scanned like the event itself (no duplicate keys, no
// case-folded alias of tool_use_id), each with exactly one non-empty string
// tool_use_id. A missing list closes nothing; anything else is refused.
func decodeBatchToolUseIDs(raw json.RawMessage) ([]string, error) {
	if raw == nil {
		return nil, nil
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("PostToolBatch tool_calls must be an array of tool calls")
	}
	var calls []json.RawMessage
	if err := json.Unmarshal(trimmed, &calls); err != nil {
		return nil, fmt.Errorf("PostToolBatch tool_calls must be an array of tool calls: %w", err)
	}
	ids := make([]string, 0, len(calls))
	for i, call := range calls {
		fields, err := scanJSONObject(call, []string{"tool_use_id"})
		if err != nil {
			return nil, fmt.Errorf("PostToolBatch tool_calls[%d]: %w", i, err)
		}
		value, ok := fields["tool_use_id"]
		if !ok {
			return nil, fmt.Errorf("PostToolBatch tool_calls[%d] has no tool_use_id", i)
		}
		if err := requireJSONString("tool_use_id", value); err != nil {
			return nil, fmt.Errorf("PostToolBatch tool_calls[%d]: %w", i, err)
		}
		var id string
		_ = json.Unmarshal(value, &id)
		if strings.TrimSpace(id) == "" {
			return nil, fmt.Errorf("PostToolBatch tool_calls[%d] has an empty tool_use_id", i)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// boundaryCloses reports whether a boundary event of in proves that the token
// owned by owner can no longer complete. Tokens without a recorded owner are
// never closed here: nothing proves which session they belong to.
func boundaryCloses(in boundaryInput, token string, owner pendingOwner) bool {
	if !owner.known() {
		return false
	}
	switch in.HookEventName {
	case "SessionEnd":
		return owner.session == hookSessionDigest(in.SessionID)
	case "PostToolBatch":
		// Only the exact calls the host reports resolved, in this lane.
		if owner.lane != hookLaneDigest(in.SessionID, in.AgentID) {
			return false
		}
		for _, id := range in.batchToolUseIDs {
			if expected, err := toolOperationToken(Input{SessionID: in.SessionID, ToolUseID: id}); err == nil && expected == token {
				return true
			}
		}
		return false
	default:
		return owner.lane == hookLaneDigest(in.SessionID, in.AgentID)
	}
}

func runBoundary(in boundaryInput, root string) (retErr error) {
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
		marked := false
		for _, token := range record.pending {
			owner := record.pendingOwners[token]
			if !owner.ended && boundaryCloses(in, token, owner) {
				owner.ended = true
				record.pendingOwners[token] = owner
				marked = true
			}
		}
		return marked
	})
	if err != nil {
		return fmt.Errorf("machinery hook: %s could not close this session's stranded tool tokens: %w", in.HookEventName, err)
	}
	return nil
}

// partitionPending splits a ledger's in-flight tokens into those owned by the
// session of in, which block its Stop; unclassified tokens with no recorded
// owner, which block every Stop because nothing proves whose they are; and
// orphaned tokens armed by another identified session. The Stop of one
// session cannot complete another session's tool call, so an orphaned token
// never blocks it; it keeps the project obligation armed. A Stop without a
// session id cannot tell its own tokens from anyone's, so every token blocks
// it.
//
// A token a boundary event marked ended stops blocking its owning session's
// main-thread Stop, which may then discharge; for a subagent Stop of that
// session and for every other session it is orphaned (never blocks, always
// withholds discharge). The rule rests on the host's Stop contract, not on
// any guard against forged boundary events.
func partitionPending(record hookStateRecord, in Input) (own, unclassified, orphaned []string) {
	if strings.TrimSpace(in.SessionID) == "" {
		return append([]string(nil), record.pending...), nil, nil
	}
	session := hookSessionDigest(in.SessionID)
	mainLaneStop := in.HookEventName == "Stop" && in.AgentID == ""
	mainLane := hookLaneDigest(in.SessionID, "")
	for _, token := range record.pending {
		owner := record.pendingOwners[token]
		switch {
		case !owner.known():
			unclassified = append(unclassified, token)
		case owner.session == session && owner.ended && mainLaneStop && owner.lane == mainLane:
			// the owner's main-thread Stop over a main-thread call: the host
			// fires it only after every foreground call resolved. A subagent
			// lane's ended call may belong to a background agent, so it
			// stays orphaned below.
		case owner.session == session && owner.ended:
			// a subagent Stop of the owning session cannot vouch for the main
			// lane or a sibling lane; the ended token withholds discharge
			orphaned = append(orphaned, token)
		case owner.session == session:
			own = append(own, token)
		default:
			orphaned = append(orphaned, token)
		}
	}
	return own, unclassified, orphaned
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
	compared := map[string]bool{}
	for _, path := range paths {
		raw, err := readRouteStateFile(path)
		if err != nil {
			return nil, err
		}
		if raw == nil || !recorded[routeSnapshotDigest(raw)] {
			continue
		}
		compared[routeSnapshotDigest(raw)] = true
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
		if earlier.Gates != "" && cfg.Gates == "" {
			add("the staged gate list " + earlier.Gates + " became progressive selection")
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
	// A recorded route whose snapshot is gone (retention, a session that
	// re-armed under a newer route, a ledger from an older binary) cannot be
	// compared; unknown is treated as narrowing, so it never discharges
	// without an operator acceptance.
	unknown := 0
	for route := range recorded {
		if !compared[route] {
			unknown++
		}
	}
	if unknown > 0 {
		add(fmt.Sprintf("%d earlier routing configuration(s) can no longer be compared because their snapshots are gone", unknown))
	}
	return narrowed, nil
}

// shellJoiners are the characters a shell removes while joining one word
// (quotes and backslash escapes): two quoted halves of a name, or a name with
// a backslash inside it, both run machinery. Deleting them first reads words the way the shell does.
var shellJoiners = strings.NewReplacer(`"`, "", `'`, "", `\`, "")

// shellWordSeparators split words where the shell does.
var shellWordSeparators = regexp.MustCompile("[|&;<>(){}`\\s]+")

// canonicalShellWords is the single reading of a command the operator-only
// guard decides on.
func canonicalShellWords(folded string) []string {
	return shellWordSeparators.Split(shellJoiners.Replace(folded), -1)
}

// ambiguousShellExpansion matches what the canonical reading cannot resolve
// without running the shell: parameter and command substitution, brace
// expansion, globs, and ANSI-C quoting.
var ambiguousShellExpansion = regexp.MustCompile("[$`{}*?\\[\\]]")

// hookWord matches hook as a whole word, including inside a brace list.
var hookWord = regexp.MustCompile(`(^|[^a-z0-9_-])hook([^a-z0-9_-]|$)`)

// invokesMachineryHook reports whether the word hook follows any word that
// names the machinery binary, or appears in a command whose words the guard
// cannot resolve (an expansion could produce the binary's name). It over-covers on purpose: flags and their
// values may sit between the binary and the subcommand, and an agent loses
// nothing it needs when a rare harmless command is refused.
func invokesMachineryHook(folded string) bool {
	words := canonicalShellWords(folded)
	if ambiguousShellExpansion.MatchString(folded) && hookWord.MatchString(shellJoiners.Replace(folded)) {
		return true // fail closed: an expansion may name the binary
	}
	seenBinary := false
	for _, word := range words {
		if name := path.Base(word); name == "machinery" || name == "machinery.exe" {
			seenBinary = true
			continue
		}
		if seenBinary && word == "hook" {
			return true
		}
	}
	return false
}

// operatorOnlyCommand names why a shell command or file edit may not run from
// a governed agent session: the durable hook store, its operator commands, and
// the hook entry point are operator and host surfaces. Like the wave sentinel,
// this is a literal guard; command text assembled dynamically can evade it,
// which the operator gate on hook-state itself and CI's machinery check back.
func operatorOnlyCommand(folded string) string {
	joined := shellJoiners.Replace(folded)
	switch {
	case operatorOnlyPath(joined) != "":
		return operatorOnlyPath(joined)
	case strings.Contains(joined, "hook-state"):
		return "'machinery hook-state' (adopt, release) is an operator command; it may not run from an agent session. Ask the human operator to run it in their own terminal."
	case strings.Contains(joined, "machinery-hook") || invokesMachineryHook(folded):
		return "'machinery hook' is the host's hook entry point; an agent may not send hook events on its own behalf."
	}
	return ""
}

// closeArmedToken removes exactly the token in armed, if the ledger holds
// it. It never creates a ledger and never changes a touch class.
func closeArmedToken(root string, in Input) (retErr error) {
	operation, err := toolOperationToken(in)
	if err != nil {
		return err
	}
	stateFile, err := statePathExact(root)
	if err != nil {
		return nil // no addressable store, so nothing was armed
	}
	if _, err := os.Lstat(stateFile); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := requireStateDir(); err != nil {
		return err
	}
	lock, err := acquireHookStateLock(stateFile)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, lock.release()) }()
	return mutateStateLocked(stateFile, root, false, func(record *hookStateRecord) bool {
		return record.removePending(operation)
	})
}

// boundaryRoutingKeys are the fields a boundary decision reads. A key that
// matches one of them only after case folding is refused, because Go's
// struct decoding matches keys case-insensitively and another consumer could
// read the variant where this one reads the exact key.
var boundaryRoutingKeys = []string{"hook_event_name", "session_id", "agent_id", "cwd", "tool_calls"}

// scanHookObject reads one top-level JSON object exactly once, in order,
// refusing duplicate keys, case-folded aliases of the routing keys, and
// trailing content, so the boundary decision and the strict decoder can never
// see different values for the same field.
func scanHookObject(raw []byte) (map[string]json.RawMessage, error) {
	return scanJSONObject(raw, boundaryRoutingKeys)
}

// scanJSONObject is scanHookObject's reader with the routing keys whose
// case-folded aliases it refuses.
func scanJSONObject(raw []byte, routingKeys []string) (map[string]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	start, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := start.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("root must be a JSON object")
	}
	fields := map[string]json.RawMessage{}
	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, fmt.Errorf("hook-event key must be a string")
		}
		if _, dup := fields[key]; dup {
			return nil, fmt.Errorf("duplicate hook-event key %q", key)
		}
		for _, routing := range routingKeys {
			if key != routing && strings.EqualFold(key, routing) {
				return nil, fmt.Errorf("hook-event key %q is a case variant of %q", key, routing)
			}
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, err
		}
		fields[key] = value
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("trailing JSON value after hook-event object")
	}
	return fields, nil
}

// mentionsBoundaryEvent reports whether an unparseable or ambiguous payload
// names a boundary event anywhere, so it is refused as a boundary instead of
// being handed to a decoder that might read it differently.
func mentionsBoundaryEvent(raw []byte) bool {
	for event := range boundaryEvents {
		if bytes.Contains(raw, []byte(event)) {
			return true
		}
	}
	return false
}

// operatorOnlyPath names why a file tool or shell command may not touch a
// path: the durable hook store and its initialization marker are both named
// machinery-hook-state-<key>.
func operatorOnlyPath(folded string) string {
	if strings.Contains(folded, "machinery-hook-state") {
		return "the machinery hook state store and its initialization marker are governance evidence; agent tools may not read, write, or move them. An operator inspects them with 'machinery doctor'."
	}
	return ""
}
