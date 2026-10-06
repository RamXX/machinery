package hook

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"
)

// releaseTokenPrefixMin is the shortest token prefix an operator may name.
// The Stop and doctor notices print 12 characters.
const releaseTokenPrefixMin = 12

// ReleaseState is the audited operator recovery for in-flight tool tokens no
// session can complete any more: a host killed mid tool call, a plugin
// disabled between PreToolUse and PostToolUse, or a ledger written before
// tokens recorded their owner. It removes only the named tokens. The project
// design/impl obligation stays armed, so the next Stop in the project still
// runs the gates; releasing a token never discharges or skips a check. Every
// release is journaled (time, operator, host, process, root, tokens) in the
// store's handoff journal next to the independent initialization marker
// before the ledger changes.
func ReleaseState(w io.Writer, root string, tokens []string, orphaned bool) error {
	return ReleaseStateWith(w, root, ReleaseOptions{Tokens: tokens, Orphaned: orphaned})
}

// ReleaseOptions selects what a release removes. Tokens and Orphaned are
// exclusive; Routes may accompany either or stand alone.
type ReleaseOptions struct {
	Tokens   []string
	Orphaned bool
	// Routes forgets the routing configurations the obligation was armed
	// under, accepting the current .machinery.json as the one the next Stop
	// runs the gates under, even where it narrows what they check. The
	// obligation itself stays armed.
	Routes bool
}

// ReleaseStateWith is ReleaseState with every selector.
func ReleaseStateWith(w io.Writer, root string, opts ReleaseOptions) (retErr error) {
	if err := operatorGate(); err != nil {
		return err
	}
	tokens, orphaned := opts.Tokens, opts.Orphaned
	if orphaned && len(tokens) > 0 {
		return fmt.Errorf("name --orphaned or --token values, not both")
	}
	if !orphaned && len(tokens) == 0 && !opts.Routes {
		return fmt.Errorf("name --orphaned, one or more --token values, or --routes")
	}
	root, err := canonicalHookRoot(root)
	if err != nil {
		return err
	}
	stateFile, err := statePathExact(root)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(stateFile); errors.Is(err, os.ErrNotExist) {
		_, err := fmt.Fprintf(w, "No recorded gate obligation for %s; nothing to release.\n", root)
		return err
	} else if err != nil {
		return err
	}
	if err := requireStateDir(); err != nil {
		return err
	}
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		return err
	}
	lock, err := acquireHookStateLock(stateFile)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, lock.release()) }()
	raw, err := readStateFile(stateFile)
	if err != nil {
		return err
	}
	record, err := parseHookStateRecord(raw)
	if err != nil {
		return err
	}
	var selected []string
	if orphaned || len(tokens) > 0 {
		if selected, err = selectReleasedTokens(record.pending, tokens, orphaned); err != nil {
			return err
		}
	}
	routes := opts.Routes && len(record.routes) > 0
	if len(selected) == 0 && !routes {
		_, err := fmt.Fprintf(w, "No in-flight tool tokens or recorded routes match for %s; nothing to release.\n", root)
		return err
	}
	// The journal is shared with adoption and guarded by the marker lock.
	// Hold it only for the append: ledger publication revalidates the store
	// binding under that same lock.
	markerLock, err := acquireHookFileLock(marker, "hook state release journal")
	if err != nil {
		return err
	}
	journalErr := journalRelease(marker, root, record, selected, routes)
	if err := errors.Join(journalErr, markerLock.Release()); err != nil {
		return err
	}
	if err := mutateStateLocked(stateFile, root, false, func(r *hookStateRecord) bool {
		removed := false
		for _, token := range selected {
			removed = r.removePending(token) || removed
		}
		if routes {
			removed = removed || len(r.routes) > 0
			r.routes = nil
		}
		return removed
	}); err != nil {
		return err
	}
	fmt.Fprintf(w, "Released %d in-flight tool token(s) for %s; recorded in %s.\n", len(selected), root, marker+".handoffs")
	if routes {
		fmt.Fprintf(w, "  forgot %d recorded routing configuration(s); the next Stop runs the gates under the current %s\n", len(record.routes), ConfigName)
	}
	for _, token := range selected {
		owner := "owner unrecorded (written before owners were recorded)"
		if o := record.pendingOwners[token]; o.known() {
			owner = "session " + o.session[:12]
		}
		fmt.Fprintf(w, "  released %s (%s)\n", token, owner)
	}
	classes := []string{}
	if record.design {
		classes = append(classes, "design")
	}
	if record.impl {
		classes = append(classes, "impl")
	}
	_, err = fmt.Fprintf(w, "The %s gate obligation stays armed: the next Stop in this project runs the gates before it can clear.\n", strings.Join(classes, "+"))
	return err
}

func selectReleasedTokens(pending, named []string, all bool) ([]string, error) {
	if all {
		return append([]string(nil), pending...), nil
	}
	seen := map[string]bool{}
	var selected []string
	for _, name := range named {
		name = strings.ToLower(strings.TrimSpace(name))
		if len(name) < releaseTokenPrefixMin || len(name) > 64 || !validLowerHex(name) {
			return nil, fmt.Errorf("token %q must be %d to 64 lowercase hex characters", name, releaseTokenPrefixMin)
		}
		var match string
		for _, token := range pending {
			if strings.HasPrefix(token, name) {
				if match != "" {
					return nil, fmt.Errorf("token prefix %s matches more than one in-flight token; name more characters", name)
				}
				match = token
			}
		}
		if match == "" {
			return nil, fmt.Errorf("no in-flight tool token %s is recorded for this project", name)
		}
		if !seen[match] {
			seen[match] = true
			selected = append(selected, match)
		}
	}
	return selected, nil
}

func journalRelease(marker, root string, record hookStateRecord, selected []string, routes bool) error {
	journal := marker + ".handoffs"
	prior, err := readBoundedHookFile(journal, "hook state handoff journal", 1<<20)
	if err != nil {
		return err
	}
	var history []byte
	if prior != nil {
		history = prior.body
	}
	operator := os.Getenv("USER")
	if current, err := user.Current(); err == nil && current.Username != "" {
		operator = current.Username
	}
	host, _ := os.Hostname()
	owners := make([]string, len(selected))
	for i, token := range selected {
		owners[i] = record.pendingOwners[token].session
	}
	event, err := json.Marshal(map[string]any{
		"time":      time.Now().UTC().Format(time.RFC3339Nano),
		"event":     "release",
		"root":      root,
		"tokens":    selected,
		"routes":    forgottenRoutes(record, routes),
		"owners":    owners,
		"revision":  strconv.FormatUint(record.revision, 10),
		"design":    record.design,
		"impl":      record.impl,
		"operator":  operator,
		"uid":       os.Getuid(),
		"host":      host,
		"pid":       os.Getpid(),
		"ppid":      os.Getppid(),
		"agent_env": agentEnvironment(),
	})
	if err != nil {
		return err
	}
	history = append(history, append(event, '\n')...)
	if len(history) > 1<<20 {
		return fmt.Errorf("handoff journal %s exceeds its bounded capacity; archive it before releasing", journal)
	}
	return writeAdoptionFile(journal, history)
}

// agentEnvironment names the agent host a release ran under, when its
// environment says so, so the journal tells an operator's terminal from a
// release an agent ran through its own shell tool.
func agentEnvironment() []string {
	var found []string
	for _, key := range []string{"CLAUDECODE", "CLAUDE_PROJECT_DIR", "CODEX_SANDBOX", "CODEX_THREAD_ID", "OPENCODE"} {
		if value, ok := os.LookupEnv(key); ok && value != "" {
			found = append(found, key)
		}
	}
	return found
}

func forgottenRoutes(record hookStateRecord, routes bool) []string {
	if !routes {
		return []string{}
	}
	return append([]string{}, record.routes...)
}
