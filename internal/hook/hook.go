// Package hook handles the shared Claude Code and Codex hook events behind
// `machinery hook`: it reads one hook event as JSON on stdin and answers on
// stdout using their compatible deny/block/context JSON contract (exit code
// always 0 on a handled event). OpenCode's adapter translates its native
// plugin events into the same input shape.
//
// Every event is a strict no-op unless the project is machinery-managed: a
// .machinery.json at the project root, or the conventional
// design/domain.modelith.yaml. The plugin's shell shim always invokes an
// available binary so Post/Stop can recover durable pre-shell routing even
// after a command deletes both mutable markers; unmanaged projects still
// produce no hook output.
//
// Division of labor: the hooks enforce only what is deterministic and never
// legitimate to violate mid-work (hand-edits to generated artifacts, DRIFT
// at turn end, import-boundary violations). Gate ERRORs on a half-built
// design are a normal interrogation state and only warn, unless the config
// asks for strict mode. CI remains the outer wall; these hooks are the
// inner-loop tripwire.
package hook

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/checker"
	"github.com/RamXX/machinery/internal/dirscan"
	"github.com/RamXX/machinery/internal/filelock"
	"github.com/RamXX/machinery/internal/fsatomic"
	"github.com/RamXX/machinery/internal/gates"
	"github.com/RamXX/machinery/internal/gitcontrol"
	"github.com/RamXX/machinery/internal/pack"
	"github.com/RamXX/machinery/internal/portablepath"
	"github.com/RamXX/machinery/internal/processcontrol"
	"github.com/RamXX/machinery/internal/safefile"
)

// ConfigName is the project-root marker and configuration file.
const ConfigName = ".machinery.json"

const (
	hookInputMaxBytes         int64 = 16 << 20
	hookConfigMaxBytes        int64 = 64 << 10
	hookMarkerMaxBytes        int64 = 16 << 20
	hookWaveMaxBytes          int64 = 4 << 10
	hookStateMaxBytes         int64 = 1 << 20
	hookRouteMaxBytes         int64 = 64 << 10
	hookStateMarkerMaxBytes   int64 = 4 << 10
	hookStateIdentityMaxBytes int64 = 4 << 10
	hookDesignMaxEntries            = 100_000
	hookStateDirMaxEntries          = 4_096
	hookStateLockWaitLimit          = 10 * time.Second
)

// Input is the compatible subset of the Claude Code and Codex hook stdin JSON.
type Input struct {
	SessionID       string    `json:"session_id"`
	PromptID        string    `json:"prompt_id"`
	ToolUseID       string    `json:"tool_use_id"`
	Cwd             string    `json:"cwd"`
	HookEventName   string    `json:"hook_event_name"`
	ToolName        string    `json:"tool_name"`
	ToolInput       toolInput `json:"tool_input"`
	StopHookActive  bool      `json:"stop_hook_active"`
	BackgroundTasks int       `json:"-"`
}

type hookJSONFieldKind uint8

const (
	hookJSONString hookJSONFieldKind = iota
	hookJSONBool
)

type toolInput struct {
	FilePath     string `json:"file_path"`
	NotebookPath string `json:"notebook_path"`
	Command      string `json:"command"`
	Patch        string `json:"patch"`
}

// Config is the .machinery.json shape. Every field is optional; an absent
// file falls back to convention (design/ with domain.modelith.yaml).
type Config struct {
	Design string `json:"design"` // design directory relative to the root (default "design")
	Gates  string `json:"gates"`  // staged --gate list; empty selects by which artifacts exist
	Impl   string `json:"impl"`   // implementation dir for G4-import; empty disables it
	Hooks  *bool  `json:"hooks"`  // explicit opt-out: {"hooks": false}
	Strict bool   `json:"strict"` // block the stop on any blocking finding, not only DRIFT
	// Dialog selects the register of the USER-FACING hook messages (the stop
	// systemMessage lines): "" keeps the operator strings, "plain" swaps them
	// for plain-language equivalents and adds a register reminder to the
	// session-start context. Model-facing text (deny reasons, block reasons,
	// the governance contract in session start) keeps full machinery
	// vocabulary in both modes: the conductor needs it, and the skill makes
	// translating it at relay time the conductor's job.
	Dialog    string `json:"dialog"`
	loadError string
	// snapshotDesign is the immutable design capability installed only while
	// a routed hook event holds gates.Snapshot.
	snapshotDesign string
}

// plainDialog reports whether the config selects the plain user-facing
// register.
func (c Config) plainDialog() bool { return c.Dialog == "plain" }

func decodeConfig(raw []byte) (Config, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	start, err := dec.Token()
	if err != nil {
		return Config{}, err
	}
	if delim, ok := start.(json.Delim); !ok || delim != '{' {
		return Config{}, fmt.Errorf("root must be a JSON object")
	}
	cfg := Config{Design: "design"}
	seen := map[string]bool{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return Config{}, err
		}
		key, ok := tok.(string)
		if !ok {
			return Config{}, fmt.Errorf("config key must be a string")
		}
		if seen[key] {
			return Config{}, fmt.Errorf("duplicate config key %q", key)
		}
		seen[key] = true
		switch key {
		case "design":
			cfg.Design, err = decodeConfigString(dec, key)
		case "gates":
			cfg.Gates, err = decodeConfigString(dec, key)
		case "impl":
			cfg.Impl, err = decodeConfigString(dec, key)
		case "hooks":
			var hooks bool
			hooks, err = decodeConfigBool(dec, key)
			cfg.Hooks = &hooks
		case "strict":
			cfg.Strict, err = decodeConfigBool(dec, key)
		case "dialog":
			cfg.Dialog, err = decodeConfigString(dec, key)
		default:
			return Config{}, fmt.Errorf("unknown config key %q (supported: design, gates, impl, hooks, strict, dialog)", key)
		}
		if err != nil {
			return Config{}, err
		}
	}
	if _, err := dec.Token(); err != nil {
		return Config{}, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Config{}, fmt.Errorf("trailing JSON value after root object")
		}
		return Config{}, err
	}
	return cfg, nil
}

func decodeConfigString(dec *json.Decoder, key string) (string, error) {
	tok, err := dec.Token()
	if err != nil {
		return "", err
	}
	value, ok := tok.(string)
	if !ok {
		return "", fmt.Errorf("config key %q must be a string", key)
	}
	return value, nil
}

func decodeConfigBool(dec *json.Decoder, key string) (bool, error) {
	tok, err := dec.Token()
	if err != nil {
		return false, err
	}
	value, ok := tok.(bool)
	if !ok {
		return false, fmt.Errorf("config key %q must be a boolean", key)
	}
	return value, nil
}

// decodeInput reads exactly one hook-event object. The outer protocol is
// closed because a misspelled routing key can otherwise turn a governed event
// into a zero-valued no-op. Tool payloads have their own closed vocabulary of
// the fields emitted by the supported Claude/Codex/OpenCode file and shell
// tools; values that governance does not inspect are still decoded so their
// shape cannot interfere with framing.
func decodeInput(r io.Reader) (Input, error) {
	rawInput, err := io.ReadAll(io.LimitReader(r, hookInputMaxBytes+1))
	if err != nil {
		return Input{}, fmt.Errorf("read hook-event JSON: %w", err)
	}
	if int64(len(rawInput)) > hookInputMaxBytes {
		return Input{}, fmt.Errorf("hook-event JSON exceeds %d-byte limit", hookInputMaxBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(rawInput))
	start, err := dec.Token()
	if err != nil {
		return Input{}, err
	}
	if delim, ok := start.(json.Delim); !ok || delim != '{' {
		return Input{}, fmt.Errorf("root must be a JSON object")
	}
	var in Input
	seen := map[string]bool{}
	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return Input{}, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return Input{}, fmt.Errorf("hook-event key must be a string")
		}
		if seen[key] {
			return Input{}, fmt.Errorf("duplicate hook-event key %q", key)
		}
		seen[key] = true
		switch key {
		case "session_id":
			in.SessionID, err = decodeInputString(dec, key)
		case "prompt_id":
			in.PromptID, err = decodeInputString(dec, key)
		case "cwd":
			in.Cwd, err = decodeInputString(dec, key)
		case "hook_event_name":
			in.HookEventName, err = decodeInputString(dec, key)
		case "tool_name":
			in.ToolName, err = decodeInputString(dec, key)
		case "tool_use_id":
			in.ToolUseID, err = decodeInputString(dec, key)
		case "tool_input":
			var raw json.RawMessage
			if err = dec.Decode(&raw); err == nil {
				in.ToolInput, err = decodeToolInput(raw)
			}
		case "stop_hook_active":
			in.StopHookActive, err = decodeInputBool(dec, key)
		case "effort":
			var raw json.RawMessage
			if err = dec.Decode(&raw); err == nil {
				err = decodeHookEffort(raw)
			}
		case "duration_ms":
			err = decodeNonnegativeInputNumber(dec, key)
		case "seconds_since_last_response", "context_tokens", "estimated_cache_write_usd":
			err = decodeNonnegativeInputNumber(dec, key)
		case "error":
			_, err = decodeInputString(dec, key)
		case "is_interrupt":
			_, err = decodeInputBool(dec, key)
		case "prompt_cache_likely_expired":
			_, err = decodeInputBool(dec, key)
		case "background_tasks":
			var raw json.RawMessage
			if err = dec.Decode(&raw); err == nil {
				in.BackgroundTasks, err = decodeClosedObjectArray(raw, key, map[string]hookJSONFieldKind{
					"id": hookJSONString, "type": hookJSONString, "status": hookJSONString,
					"description": hookJSONString, "command": hookJSONString, "agent_type": hookJSONString,
					"server": hookJSONString, "tool": hookJSONString, "name": hookJSONString,
				})
			}
		case "session_crons":
			var raw json.RawMessage
			if err = dec.Decode(&raw); err == nil {
				_, err = decodeClosedObjectArray(raw, key, map[string]hookJSONFieldKind{
					"id": hookJSONString, "schedule": hookJSONString, "recurring": hookJSONBool,
					"prompt": hookJSONString,
				})
			}
		// Official adapter metadata is accepted but never allowed to steer
		// machinery's routing. Keeping the list explicit catches typos.
		case "transcript_path", "scratchpad_dir", "permission_mode", "source", "model", "agent_id", "agent_type", "agent_transcript_path", "last_assistant_message", "session_title":
			_, err = decodeInputString(dec, key)
		case "tool_response":
			var ignored json.RawMessage
			err = dec.Decode(&ignored)
		default:
			return Input{}, fmt.Errorf("unknown hook-event key %q", key)
		}
		if err != nil {
			return Input{}, err
		}
	}
	if _, err := dec.Token(); err != nil {
		return Input{}, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Input{}, fmt.Errorf("trailing JSON value after hook-event object")
		}
		return Input{}, err
	}
	return in, nil
}

func decodeToolInput(raw []byte) (toolInput, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return toolInput{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	start, err := dec.Token()
	if err != nil {
		return toolInput{}, err
	}
	if delim, ok := start.(json.Delim); !ok || delim != '{' {
		return toolInput{}, fmt.Errorf("hook-event key %q must be an object", "tool_input")
	}
	var input toolInput
	seen := map[string]bool{}
	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return toolInput{}, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return toolInput{}, fmt.Errorf("tool_input key must be a string")
		}
		if seen[key] {
			return toolInput{}, fmt.Errorf("duplicate tool_input key %q", key)
		}
		seen[key] = true
		switch key {
		case "file_path":
			input.FilePath, err = decodeInputString(dec, "tool_input."+key)
		case "notebook_path":
			input.NotebookPath, err = decodeInputString(dec, "tool_input."+key)
		case "command":
			input.Command, err = decodeInputString(dec, "tool_input."+key)
		case "patch":
			input.Patch, err = decodeInputString(dec, "tool_input."+key)
		case "content", "old_string", "new_string", "replace_all", "edits", "cell_id", "new_source", "cell_type", "edit_mode", "description", "timeout", "run_in_background", "dangerouslyDisableSandbox":
			var ignored json.RawMessage
			err = dec.Decode(&ignored)
		default:
			return toolInput{}, fmt.Errorf("unknown tool_input key %q", key)
		}
		if err != nil {
			return toolInput{}, err
		}
	}
	if _, err := dec.Token(); err != nil {
		return toolInput{}, err
	}
	return input, nil
}

func decodeInputString(dec *json.Decoder, key string) (string, error) {
	var value string
	if err := dec.Decode(&value); err != nil {
		return "", fmt.Errorf("hook-event key %q must be a string: %w", key, err)
	}
	return value, nil
}

func decodeInputBool(dec *json.Decoder, key string) (bool, error) {
	var value bool
	if err := dec.Decode(&value); err != nil {
		return false, fmt.Errorf("hook-event key %q must be a boolean: %w", key, err)
	}
	return value, nil
}

func decodeNonnegativeInputNumber(dec *json.Decoder, key string) error {
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return err
	}
	numberDecoder := json.NewDecoder(bytes.NewReader(raw))
	numberDecoder.UseNumber()
	var value any
	if err := numberDecoder.Decode(&value); err != nil {
		return fmt.Errorf("hook-event key %q must be a nonnegative finite number: %w", key, err)
	}
	number, ok := value.(json.Number)
	parsed, parseErr := number.Float64()
	if !ok || parseErr != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) || parsed < 0 || strings.HasPrefix(number.String(), "-") {
		return fmt.Errorf("hook-event key %q must be a nonnegative finite number", key)
	}
	return nil
}

func decodeHookEffort(raw []byte) error {
	values, err := decodeClosedObject(raw, "effort", map[string]hookJSONFieldKind{"level": hookJSONString})
	if err != nil {
		return err
	}
	level, ok := values["level"].(string)
	if !ok {
		return fmt.Errorf("hook-event key %q requires string field %q", "effort", "level")
	}
	switch level {
	case "low", "medium", "high", "xhigh", "max":
		return nil
	default:
		return fmt.Errorf("hook-event key %q has unsupported level %q", "effort", level)
	}
}

func decodeClosedObjectArray(raw []byte, key string, fields map[string]hookJSONFieldKind) (int, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	start, err := dec.Token()
	if err != nil {
		return 0, err
	}
	if delim, ok := start.(json.Delim); !ok || delim != '[' {
		return 0, fmt.Errorf("hook-event key %q must be an array", key)
	}
	count := 0
	for dec.More() {
		var item json.RawMessage
		if err := dec.Decode(&item); err != nil {
			return 0, err
		}
		if _, err := decodeClosedObject(item, key+" item", fields); err != nil {
			return 0, err
		}
		count++
	}
	if _, err := dec.Token(); err != nil {
		return 0, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return 0, fmt.Errorf("hook-event key %q has trailing JSON", key)
		}
		return 0, err
	}
	return count, nil
}

func decodeClosedObject(raw []byte, label string, fields map[string]hookJSONFieldKind) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	start, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := start.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("hook-event %s must be an object", label)
	}
	values := make(map[string]any, len(fields))
	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, fmt.Errorf("hook-event %s key must be a string", label)
		}
		kind, ok := fields[key]
		if !ok {
			return nil, fmt.Errorf("unknown hook-event %s key %q", label, key)
		}
		if _, duplicate := values[key]; duplicate {
			return nil, fmt.Errorf("duplicate hook-event %s key %q", label, key)
		}
		switch kind {
		case hookJSONString:
			values[key], err = decodeInputString(dec, label+"."+key)
		case hookJSONBool:
			values[key], err = decodeInputBool(dec, label+"."+key)
		default:
			err = fmt.Errorf("unsupported hook-event %s field kind", label)
		}
		if err != nil {
			return nil, err
		}
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("hook-event %s has trailing JSON", label)
		}
		return nil, err
	}
	return values, nil
}

// Load resolves the machinery hook configuration for root. ok is false when
// the project is not machinery-managed (every hook no-ops). A present but
// invalid config still counts as managed and carries a hard loadError; no
// event is routed through guessed defaults.
func Load(root string) (cfg Config, ok bool, warn string) {
	cfg = Config{Design: "design"}
	raw, present, err := readConfinedRegular(root, ConfigName, hookConfigMaxBytes)
	if err != nil {
		cfg.loadError = fmt.Sprintf("%s is invalid or unreadable: %v", ConfigName, err)
		return cfg, true, "machinery: " + cfg.loadError
	}
	if !present {
		_, markerPresent, markerErr := readConfinedRegular(root, conventionalMarker, hookMarkerMaxBytes)
		if markerErr != nil {
			cfg.loadError = fmt.Sprintf("%s is invalid or unreadable: %v", conventionalMarker, markerErr)
			return cfg, true, "machinery: " + cfg.loadError
		}
		if markerPresent {
			return cfg, true, ""
		}
		return cfg, false, ""
	}
	parsed, jerr := decodeConfig(raw)
	if jerr != nil {
		cfg = Config{Design: "design"}
		cfg.loadError = fmt.Sprintf("%s does not parse: %v", ConfigName, jerr)
		return cfg, true, "machinery: " + cfg.loadError
	}
	cfg = parsed
	if cfg.Design == "" {
		cfg.Design = "design"
	}
	if err := portablepath.ValidateRelative(cfg.Design); err != nil {
		cfg.loadError = fmt.Sprintf("%s design path is invalid: %v", ConfigName, err)
		return cfg, true, "machinery: " + cfg.loadError
	}
	if cfg.Impl != "" && cfg.Impl != "." {
		if err := portablepath.ValidateRelative(cfg.Impl); err != nil {
			cfg.loadError = fmt.Sprintf("%s impl path is invalid: %v", ConfigName, err)
			return cfg, true, "machinery: " + cfg.loadError
		}
	}
	if cfg.Gates != "" {
		for _, tok := range strings.Split(strings.ToLower(cfg.Gates), ",") {
			t := strings.TrimSpace(tok)
			if !gates.KnownGate(t) {
				cfg.loadError = fmt.Sprintf("%s gates list has unknown gate %q", ConfigName, t)
				return cfg, true, "machinery: " + cfg.loadError
			}
		}
	}
	if cfg.Dialog != "" && cfg.Dialog != "plain" {
		cfg.loadError = fmt.Sprintf("%s dialog value %q is not supported (use \"plain\" or omit it)", ConfigName, cfg.Dialog)
		return cfg, true, "machinery: " + cfg.loadError
	}
	// An explicit opt-out is honored only after the entire present config has
	// passed the closed schema and semantic vocabulary. Otherwise a typo in a
	// disabled-looking config could silently turn governance off.
	if cfg.Hooks != nil && !*cfg.Hooks {
		return cfg, false, ""
	}
	return cfg, true, warn
}

func readConfinedRegular(rootPath, name string, maxBytes int64) (body []byte, present bool, retErr error) {
	if maxBytes <= 0 {
		return nil, false, fmt.Errorf("%s read limit must be positive", name)
	}
	abs, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, false, err
	}
	before, err := os.Lstat(abs)
	if err != nil {
		return nil, false, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, false, fmt.Errorf("root %s must be a real directory", abs)
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, false, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	after, err := root.Stat(".")
	if err != nil {
		return nil, false, err
	}
	if !os.SameFile(before, after) {
		return nil, false, fmt.Errorf("root %s changed while it was being opened", abs)
	}
	entry, err := root.Lstat(name)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, err
	}
	if entry.Mode()&os.ModeSymlink != 0 || !entry.Mode().IsRegular() {
		return nil, true, fmt.Errorf("%s must be a regular file inside the project; symlinks and special entries are rejected", name)
	}
	if entry.Size() > maxBytes {
		return nil, true, fmt.Errorf("%s exceeds %d-byte limit", name, maxBytes)
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, true, err
	}
	defer func() { retErr = errors.Join(retErr, f.Close()) }()
	info, err := f.Stat()
	if err != nil {
		return nil, true, err
	}
	if !info.Mode().IsRegular() || !os.SameFile(entry, info) {
		return nil, true, fmt.Errorf("%s changed identity or type while being opened", name)
	}
	if info.Size() > maxBytes {
		return nil, true, fmt.Errorf("%s exceeds %d-byte limit", name, maxBytes)
	}
	body, err = io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, true, err
	}
	if int64(len(body)) > maxBytes {
		return nil, true, fmt.Errorf("%s exceeds %d-byte limit", name, maxBytes)
	}
	openedAfter, statErr := f.Stat()
	pathAfter, pathErr := root.Lstat(name)
	if err := errors.Join(statErr, pathErr); err != nil {
		return nil, true, err
	}
	if !openedAfter.Mode().IsRegular() || pathAfter.Mode()&os.ModeSymlink != 0 || !pathAfter.Mode().IsRegular() ||
		!os.SameFile(info, openedAfter) || !os.SameFile(info, pathAfter) || openedAfter.Mode() != info.Mode() ||
		openedAfter.Size() != info.Size() || !openedAfter.ModTime().Equal(info.ModTime()) {
		return nil, true, fmt.Errorf("%s changed while being read", name)
	}
	return body, true, nil
}

// Run dispatches one hook event read from r and writes the answer to w.
// root overrides project-root resolution (flag > $CLAUDE_PROJECT_DIR > the
// event's cwd). A nil return with no output means "nothing to say": the
// event was either not machinery's business or clean.
func Run(r io.Reader, w io.Writer, root string) error {
	in, err := decodeInput(r)
	if err != nil {
		return fmt.Errorf("machinery hook: stdin is not hook-event JSON: %w", err)
	}
	switch in.HookEventName {
	case "PreToolUse", "PostToolUse", "PostToolUseFailure", "Stop", "SubagentStop", "SessionStart":
	default:
		return fmt.Errorf("machinery hook: unsupported hook event %q", in.HookEventName)
	}
	if root == "" {
		root = os.Getenv("CLAUDE_PROJECT_DIR")
	}
	if root == "" {
		root = in.Cwd
	}
	if root == "" {
		root = "."
	}
	root = resolveEventPath(in.Cwd, root)
	root, err = canonicalHookRoot(root)
	if err != nil {
		reason := "machinery governance cannot resolve a canonical project root: " + err.Error()
		switch in.HookEventName {
		case "PreToolUse":
			return emitJSON(w, preOut{HookSpecificOutput: preSpecific{HookEventName: "PreToolUse", PermissionDecision: "deny", PermissionDecisionReason: reason}})
		case "Stop", "SubagentStop":
			return emitJSON(w, stopOut{Decision: "block", Reason: reason})
		default:
			return errors.New(reason)
		}
	}
	cfg, ok, warn := Load(root)
	if in.HookEventName == "PostToolUse" || in.HookEventName == "PostToolUseFailure" || in.HookEventName == "Stop" || in.HookEventName == "SubagentStop" {
		if !ok {
			durable, durableErr := durableProjectStatePresent(root)
			if durableErr != nil {
				reason := "machinery governance cannot inspect pre-event durable routing state: " + durableErr.Error()
				if in.HookEventName == "Stop" || in.HookEventName == "SubagentStop" {
					return emitJSON(w, stopOut{Decision: "block", Reason: reason})
				}
				return errors.New(reason)
			}
			if !durable {
				return nil
			}
		}
		routeCfg, routePresent, routeErr := loadRouteSnapshot(root, in.SessionID)
		if routeErr != nil {
			reason := "machinery governance cannot recover pre-shell routing state: " + routeErr.Error()
			if in.HookEventName == "Stop" || in.HookEventName == "SubagentStop" {
				return emitJSON(w, stopOut{Decision: "block", Reason: reason})
			}
			return errors.New(reason)
		}
		if routePresent {
			cfg, ok, warn = routeCfg, true, ""
		}
	}
	if !ok {
		return nil
	}
	if err := requireStateDir(); err != nil {
		reason := "machinery governance cannot access its durable project-obligation store: " + err.Error()
		switch in.HookEventName {
		case "PreToolUse":
			return emitJSON(w, preOut{HookSpecificOutput: preSpecific{HookEventName: "PreToolUse", PermissionDecision: "deny", PermissionDecisionReason: reason}})
		case "Stop", "SubagentStop":
			return emitJSON(w, stopOut{Decision: "block", Reason: reason})
		default:
			return errors.New(reason)
		}
	}
	if cfg.loadError != "" {
		reason := "machinery governance configuration is unusable; refusing to route this event through guessed defaults: " + cfg.loadError
		switch in.HookEventName {
		case "PreToolUse":
			return emitJSON(w, preOut{HookSpecificOutput: preSpecific{
				HookEventName:            "PreToolUse",
				PermissionDecision:       "deny",
				PermissionDecisionReason: reason,
			}})
		case "Stop", "SubagentStop":
			return emitJSON(w, stopOut{Decision: "block", Reason: reason})
		default:
			return errors.New(reason)
		}
	}
	switch in.HookEventName {
	case "PreToolUse":
		return withRoutingSnapshot(root, cfg, func(current Config) error { return pre(w, root, current, in) })
	case "PostToolUse", "PostToolUseFailure":
		return withRoutingSnapshot(root, cfg, func(current Config) error { return post(root, current, in) })
	case "Stop", "SubagentStop":
		return stop(w, root, cfg, in, warn)
	case "SessionStart":
		return withRoutingSnapshot(root, cfg, func(current Config) error { return sessionStart(w, root, current, warn) })
	}
	return nil
}

// canonicalHookRoot deterministically ascends the real filesystem hierarchy
// to the nearest machinery marker. It never consults Git, so missing tools,
// injected GIT_DIR/GIT_WORK_TREE, corrupt repositories, and localized Git
// diagnostics cannot reroute governance or turn a managed subdirectory into
// an unmanaged project.
func canonicalHookRoot(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(real)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("hook root %s must be a real directory", real)
	}
	for candidate := real; ; candidate = filepath.Dir(candidate) {
		for _, marker := range []string{ConfigName, conventionalMarker} {
			_, err := os.Lstat(filepath.Join(candidate, filepath.FromSlash(marker)))
			if err == nil {
				return candidate, nil
			}
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("inspect governance marker %s: %w", filepath.Join(candidate, filepath.FromSlash(marker)), err)
			}
		}
		dirty, err := durableProjectStatePresent(candidate)
		if err != nil {
			return "", fmt.Errorf("inspect durable governance state for %s: %w", candidate, err)
		}
		if dirty {
			return candidate, nil
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return real, nil
		}
	}
}

func withRoutingSnapshot(root string, cfg Config, fn func(Config) error) (retErr error) {
	designDir := filepath.Join(root, filepath.FromSlash(designRel(cfg)))
	if _, err := os.Lstat(designDir); os.IsNotExist(err) {
		return fn(cfg)
	} else if err != nil {
		return err
	}
	snapshot, err := gates.AcquireSnapshot(designDir)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, snapshot.Release()) }()
	configPath := filepath.Join(root, ConfigName)
	if _, err := os.Lstat(configPath); err == nil {
		if err := snapshot.TrackExternal(configPath); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	current, ok, _ := Load(root)
	if !ok || !sameConfig(cfg, current) {
		return fmt.Errorf("machinery hook routing config changed while acquiring the design snapshot; retry the hook event")
	}
	current.snapshotDesign = snapshot.DesignPath()
	if err := fn(current); err != nil {
		return snapshot.LogicalError(err)
	}
	return snapshot.CheckUnchanged()
}

// --- PreToolUse: generated artifacts are read-only ---

var fileTools = map[string]bool{
	"Edit":         true,
	"Write":        true,
	"MultiEdit":    true,
	"NotebookEdit": true,
	"apply_patch":  true,
	"edit":         true,
	"write":        true,
	"patch":        true,
}

var shellTools = map[string]bool{
	"Bash": true, "bash": true, "Shell": true, "shell": true,
}

type preOut struct {
	HookSpecificOutput preSpecific `json:"hookSpecificOutput"`
}

type preSpecific struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision"`
	PermissionDecisionReason string `json:"permissionDecisionReason"`
}

// conventionalMarker is the fallback governance marker: with no ConfigName
// present, its existence alone is what keeps the hooks armed.
const conventionalMarker = "design/domain.modelith.yaml"

func pre(w io.Writer, root string, cfg Config, in Input) error {
	if !fileTools[in.ToolName] && !shellTools[in.ToolName] {
		return nil
	}
	deny := func(reason string) error {
		return emitJSON(w, preOut{HookSpecificOutput: preSpecific{
			HookEventName:            "PreToolUse",
			PermissionDecision:       "deny",
			PermissionDecisionReason: reason,
		}})
	}
	if shellTools[in.ToolName] {
		reason, err := shellProtectedMutation(root, cfg, in.ToolInput.Command)
		if err != nil {
			return deny("protected-artifact inventory is unreadable or invalid; refusing a shell command that could bypass generated-file governance: " + err.Error())
		}
		if reason != "" {
			return deny(reason)
		}
		if err := armShellState(root, in, cfg); err != nil {
			return deny("machinery governance could not durably arm pre-shell design/implementation tracking; refusing shell execution: " + err.Error())
		}
		return nil
	}
	checkerOutputs, checkerErr := checkerGeneratedOutputsFrom(root, designRel(cfg), cfg.snapshotDesign)
	if checkerErr != nil {
		return deny("external-checker generated-output inventory is unreadable or invalid; refusing an edit that could overwrite protected evidence: " + logicalSnapshotError(root, cfg, checkerErr))
	}
	// governance must not be switchable from inside a session (GATE-10):
	// a Write of {"hooks": false} to the config, or an apply_patch DELETE of
	// either marker, turns every hook off. Editing the domain model itself
	// stays allowed: design/domain.modelith.yaml is the Phase 1 source, and
	// only its deletion (which disarms detection) is denied. Shell commands
	// are handled above and their Post event always arms the stop ledger.
	// every path comparison below folds case: the filesystems this hook
	// guards on (APFS, NTFS) resolve names case-insensitively, so an exact-
	// case guard is bypassable by writing .MACHINERY-WAVE or .Machinery.json
	// and letting os.Stat/os.ReadFile find it under the canonical spelling.
	// On a case-sensitive filesystem the folded deny over-covers only
	// near-case variants of reserved names, which no legitimate edit uses.
	for _, deleted := range deletedPaths(in) {
		rel := relToRoot(root, resolveEventPath(in.Cwd, deleted))
		if strings.EqualFold(rel, ConfigName) || strings.EqualFold(rel, conventionalMarker) {
			return deny("deleting " + rel + " switches machinery governance off for this repository. " +
				"If governance must be disabled, a human sets {\"hooks\": false} in " + ConfigName + ".")
		}
	}
	// Exemptions are per operation, never per path: one apply_patch that both
	// deletes and re-adds the sentinel must still be denied for the add, so
	// the delete cannot launder a fresh full-TTL wave through the same call.
	for _, edited := range editedOps(in) {
		rel := relToRoot(root, resolveEventPath(in.Cwd, edited.Path))
		if rel == "" {
			continue
		}
		if strings.EqualFold(rel, ConfigName) {
			return deny(rel + " is the machinery governance configuration; an agent edit here can switch " +
				"governance off ({\"hooks\": false}) or silently reroute the gates. A human maintains this file " +
				"(or 'machinery init' regenerates it).")
		}
		if strings.EqualFold(path.Base(rel), waveSentinelName) && edited.Op != opDelete {
			return deny(rel + " is the wave sentinel, and it is operator-created: while it is fresh the stop " +
				"gates surface red findings as a message instead of blocking, so an agent that creates or " +
				"re-touches it defers gating for as long as it likes. Ask the human running the wave to open " +
				"or extend it. Deleting it (which closes the wave and re-arms the gates) stays allowed.")
		}
		reason := generatedReason(designRel(cfg), rel)
		if kind := checkerOutputs[strings.ToLower(filepath.ToSlash(rel))]; kind != "" {
			command := "machinery project " + designRel(cfg)
			if kind != "projection" {
				command = "machinery verify-checkers " + designRel(cfg)
			}
			reason = rel + " is generated external-checker " + kind + " output. Run '" + command + "'; never edit checker results in place."
		}
		if reason == "" {
			continue
		}
		return deny(reason)
	}
	if err := armFileState(root, cfg, in); err != nil {
		return deny("machinery governance could not durably arm project tracking before the file edit; refusing execution: " + err.Error())
	}
	return nil
}

func shellProtectedMutation(root string, cfg Config, command string) (string, error) {
	folded := strings.ToLower(filepath.ToSlash(command))
	for _, reserved := range []string{strings.ToLower(ConfigName), strings.ToLower(waveSentinelName), strings.ToLower(conventionalMarker)} {
		if strings.Contains(folded, reserved) {
			return reserved + " is protected machinery governance state; shell commands may not reference it because command text cannot prove read-only intent or a confined target", nil
		}
	}
	for _, marker := range []string{"ratchet.json", ".oracle.md", ".tla", ".cfg", ".als", "/packs/", "/pack/"} {
		if strings.Contains(folded, marker) {
			return marker + " identifies generated or frozen machinery output; shell commands may not reference protected output regardless of verb (use the owning machinery generator)", nil
		}
	}
	checkerOutputs, err := checkerGeneratedOutputsFrom(root, designRel(cfg), cfg.snapshotDesign)
	if err != nil {
		return "", errors.New(logicalSnapshotError(root, cfg, err))
	}
	checkerPaths := make([]string, 0, len(checkerOutputs))
	for rel := range checkerOutputs {
		checkerPaths = append(checkerPaths, rel)
	}
	sort.Strings(checkerPaths)
	for _, rel := range checkerPaths {
		kind := checkerOutputs[rel]
		if strings.Contains(folded, strings.ToLower(rel)) {
			return rel + " is generated external-checker " + kind + " output; mutate its sources and regenerate it instead of writing it from a shell", nil
		}
	}
	for _, field := range strings.Fields(command) {
		candidate := strings.Trim(field, `"'\`+"`"+`;|&<>(){}[]`)
		if candidate == "" || strings.HasPrefix(candidate, "-") {
			continue
		}
		rel := relToRoot(root, resolveEventPath("", candidate))
		if reason := generatedReason(designRel(cfg), rel); reason != "" {
			return reason, nil
		}
	}
	return "", nil
}

func logicalSnapshotError(root string, cfg Config, err error) string {
	if err == nil {
		return ""
	}
	if cfg.snapshotDesign == "" {
		return err.Error()
	}
	logical := filepath.Join(root, filepath.FromSlash(designRel(cfg)))
	return strings.ReplaceAll(err.Error(), cfg.snapshotDesign, logical)
}

func checkerGeneratedOutputsFrom(root, design, sourceDesign string) (map[string]string, error) {
	out := map[string]string{}
	designDir := filepath.Join(root, filepath.FromSlash(design))
	if sourceDesign != "" {
		designDir = sourceDesign
	}
	manifestPaths, err := checker.ManifestPaths(designDir)
	if err != nil {
		return nil, err
	}
	for _, manifestPath := range manifestPaths {
		manifest, err := checker.LoadManifest(manifestPath)
		if err != nil {
			return nil, err
		}
		for _, spec := range []struct {
			rel, kind string
		}{
			{manifest.Evidence.ProjectionOut, "projection"},
			{manifest.Evidence.EvidenceIn, "evidence"},
		} {
			full, err := checker.ConfinedPath(designDir, spec.rel)
			if err != nil {
				return nil, err
			}
			designRelPath, err := filepath.Rel(designDir, full)
			if err != nil {
				return nil, err
			}
			rootRel := filepath.Join(filepath.FromSlash(design), designRelPath)
			out[strings.ToLower(filepath.ToSlash(rootRel))] = spec.kind
		}

		evidenceFull, err := checker.ConfinedPath(designDir, manifest.Evidence.EvidenceIn)
		if err != nil {
			return nil, err
		}
		if _, err := os.Lstat(evidenceFull); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		evidence, _, err := checker.LoadEvidenceConfinedBytes(designDir, manifest.Evidence.EvidenceIn)
		if err != nil {
			return nil, err
		}
		if evidence.TraceRef != "" {
			traceRel := filepath.ToSlash(filepath.Join(filepath.Dir(manifest.Evidence.EvidenceIn), filepath.FromSlash(evidence.TraceRef)))
			traceFull, err := checker.ConfinedPath(designDir, traceRel)
			if err != nil {
				return nil, err
			}
			designRelPath, err := filepath.Rel(designDir, traceFull)
			if err != nil {
				return nil, err
			}
			rootRel := filepath.Join(filepath.FromSlash(design), designRelPath)
			out[strings.ToLower(filepath.ToSlash(rootRel))] = "trace"
		}
	}
	return out, nil
}

// generatedReason classifies rel (a root-relative, slash-separated path) as
// a generated design artifact and returns the refusal reason, or "".
func generatedReason(design, rel string) string {
	if !strings.EqualFold(rel, design) && !foldHasPrefix(rel, design+"/") {
		return ""
	}
	sub := rel
	if foldHasPrefix(rel, design+"/") {
		sub = rel[len(design)+1:]
	}
	base := path.Base(sub)
	switch {
	case strings.EqualFold(sub, "ratchet.json"):
		return rel + " is generated by 'machinery baseline' from the observed import graph; a hand edit defeats the ratchet. " +
			"Rerun 'machinery baseline " + design + " --impl <dir>' to regenerate it."
	case foldHasPrefix(sub, "formal/") && foldHasSuffix(base, ".oracle.md"):
		return rel + " is generated by 'machinery alloy' from the domain model + " +
			"formal/" + alloySource(base) + ". Edit those sources, run 'machinery alloy " + design + "', " +
			"and commit the regenerated oracle."
	case foldHasSuffix(base, ".oracle.md"):
		return rel + " is generated by 'machinery oracle'; a hand edit is DRIFT by definition. " +
			"Edit the machine JSON (and its matrix), run 'machinery oracle " + design + "/machines', " +
			"and commit the regenerated oracle."
	case foldHasPrefix(sub, "formal/") && (foldHasSuffix(base, ".tla") || foldHasSuffix(base, ".cfg")):
		return rel + " is generated by 'machinery verify-formal'. Edit the machine JSON or the " +
			"formal/*.yaml annotations, run 'machinery verify-formal " + design + "' " +
			"(--gen-only without Java), and commit the regenerated files."
	case foldHasPrefix(sub, "formal/") && foldHasSuffix(base, ".als"):
		return rel + " is generated by 'machinery alloy' from the domain model + " +
			"formal/" + alloySource(base) + ". Edit those sources, run 'machinery alloy " + design + "', " +
			"and commit the regenerated model."
	case foldHasPrefix(sub, "packs/"):
		return rel + " is generated by 'machinery pack generate'. Edit the parent design sources " +
			"and regenerate the packs; a boundary change is a parent edit."
	case foldHasPrefix(sub, "pack/"):
		return rel + " is the frozen pack this child design was built against. It changes only when " +
			"the parent regenerates it; copy the new pack in, never edit it in place."
	}
	return ""
}

// foldHasPrefix and foldHasSuffix are the case-folded spellings of
// strings.HasPrefix/HasSuffix, for path guards on case-insensitive
// filesystems (see the comment in pre).
func foldHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

func foldHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && strings.EqualFold(s[len(s)-len(suffix):], suffix)
}

// alloySource maps a generated alloy artifact base name to the relational
// annotation it was compiled from, so the deny message points at the right
// source. Each layer has its own annotation; the default is the policy layer.
func alloySource(base string) string {
	switch {
	case strings.HasPrefix(base, "Integrity."):
		return "integrity.relational.yaml"
	case strings.HasPrefix(base, "Isolation."):
		return "isolation.relational.yaml"
	default:
		return "policy.relational.yaml"
	}
}

// waveSentinelName is the operator-owned wave sentinel. Its explicit state is
// what downgrades a red stop from a block to a message, so opening it is a
// human act. Deleting it closes the wave and re-arms the gates.
const waveSentinelName = ".machinery-wave"

// waveSentinel is deliberately clock-free: canonical content "open" is the
// complete active state; deletion ends it. Any other content fails closed as
// stale rather than changing decisions with mtime or wall-clock progress.
func waveSentinel(designDir string) (left string, stale, active bool) {
	body, present, err := readConfinedRegular(designDir, waveSentinelName, hookWaveMaxBytes)
	if !present {
		return "", false, false
	}
	if err != nil {
		return "", true, true
	}
	if strings.TrimSpace(string(body)) != "open" {
		return "", true, true
	}
	return "open", false, true
}

// --- PostToolUse: record what the session touched (the stop gates read it) ---

var sourceExt = map[string]bool{
	".go": true, ".py": true,
	".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".mjs": true, ".cjs": true,
	".ex": true, ".exs": true,
	".rs": true,
}

func post(root string, cfg Config, in Input) error {
	if !fileTools[in.ToolName] && !shellTools[in.ToolName] {
		return nil
	}
	design := designRel(cfg)
	// A shell command can compute paths dynamically, so its target set cannot
	// be reconstructed soundly from text. Conservatively retain both ledgers;
	// Stop then inventories the actual trees and cannot be skipped by aliases,
	// variables, subshells, or tool-specific command syntax.
	touchedDesign, touchedImpl := shellTools[in.ToolName], shellTools[in.ToolName] && cfg.Impl != ""
	for _, edited := range editedPaths(in) {
		rel := relToRoot(root, resolveEventPath(in.Cwd, edited))
		if rel == "" {
			continue
		}
		switch {
		case rel == design || strings.HasPrefix(rel, design+"/"):
			touchedDesign = true
		case cfg.Impl != "" && sourceExt[path.Ext(rel)] && underImpl(cfg, rel):
			touchedImpl = true
		}
	}
	if touchedDesign || touchedImpl {
		if err := completeToolState(root, cfg, in, touchedDesign, touchedImpl); err != nil {
			return fmt.Errorf("machinery hook: complete durable tool tracking for stop-time governance: %w", err)
		}
	}
	return nil
}

func underImpl(cfg Config, rel string) bool {
	impl := path.Clean(filepath.ToSlash(cfg.Impl))
	if impl == "." {
		return true
	}
	return rel == impl || strings.HasPrefix(rel, impl+"/")
}

// --- Stop / SubagentStop: the gates run before the turn may end ---

type stopOut struct {
	Decision      string `json:"decision,omitempty"`
	Reason        string `json:"reason,omitempty"`
	SystemMessage string `json:"systemMessage,omitempty"`
}

// reasonCap bounds the gate output fed back into the model on a block.
const reasonCap = 8000

func stop(w io.Writer, root string, cfg Config, in Input, warn string) (retErr error) {
	state, stateErr := readStateRecord(root, in.SessionID)
	if stateErr != nil {
		return emitJSON(w, stopOut{Decision: "block", Reason: "machinery governance cannot read its touched-file state; refusing to end the turn without running the required checks: " + stateErr.Error()})
	}
	if len(state.routes) > 0 {
		routeBody, err := routeSnapshotBody(cfg)
		if err != nil {
			return emitJSON(w, stopOut{Decision: "block", Reason: "machinery governance cannot bind the dirty obligation to its routing identity: " + err.Error()})
		}
		want := routeSnapshotDigest(routeBody)
		for _, bound := range state.routes {
			if bound != want {
				return emitJSON(w, stopOut{Decision: "block", Reason: "machinery governance dirty obligation was armed under a different routing configuration; refusing to clear it using fallback or changed configuration"})
			}
		}
	}
	if len(state.pending) > 0 {
		return emitJSON(w, stopOut{Decision: "block", Reason: fmt.Sprintf("machinery governance has %d in-flight tool operation(s) whose PostToolUse completion was not durably recorded; refusing to discharge or clear the project gate obligation while a mutation may still be running", len(state.pending))})
	}
	touchedDesign, touchedImpl := state.design, state.impl
	if !touchedDesign && !touchedImpl {
		return nil
	}
	if in.BackgroundTasks > 0 {
		return emitJSON(w, stopOut{Decision: "block", Reason: fmt.Sprintf("machinery governance sees %d background task(s) still running; refusing to discharge or clear the project gate obligation while a process may still mutate the design or implementation", in.BackgroundTasks)})
	}
	design := designRel(cfg)
	designDir := filepath.Join(root, filepath.FromSlash(design))
	if fi, err := os.Stat(designDir); err != nil || !fi.IsDir() {
		return emitJSON(w, stopOut{Decision: "block", Reason: "machinery governance cannot run because the configured design directory " +
			design + "/ is missing or is not a directory. Restore it or correct the operator-owned configuration; the touched state is retained."})
	}
	snapshot, snapshotErr := gates.AcquireSnapshot(designDir)
	if snapshotErr != nil {
		return emitJSON(w, stopOut{Decision: "block", Reason: "machinery governance cannot acquire a consistent design snapshot: " + snapshotErr.Error()})
	}
	defer func() { retErr = errors.Join(retErr, snapshot.Release()) }()
	sourceDesignDir := snapshot.DesignPath()
	configPath := filepath.Join(root, ConfigName)
	if _, err := os.Lstat(configPath); err == nil {
		if err := snapshot.TrackExternal(configPath); err != nil {
			return emitJSON(w, stopOut{Decision: "block", Reason: "machinery governance cannot bind its routing config to the design snapshot: " + err.Error()})
		}
	} else if !os.IsNotExist(err) {
		return emitJSON(w, stopOut{Decision: "block", Reason: "machinery governance cannot inspect its routing config: " + err.Error()})
	}
	freshCfg, freshOK, freshWarn := Load(root)
	if !freshOK || !sameConfig(cfg, freshCfg) {
		return emitJSON(w, stopOut{Decision: "block", Reason: "machinery governance routing config changed while acquiring the design snapshot; retry the stop so selection and design use one config revision"})
	}
	// The same config bytes should reproduce the same warning. Preserve an
	// earlier safety warning if the reload unexpectedly loses it; silently
	// dropping operator guidance would make the hook less conservative.
	if warn != "" && freshWarn == "" {
		freshWarn = warn
	}
	cfg, warn = freshCfg, freshWarn
	sel, selWarn, selErr := selectGatesCheckedInSnapshot(snapshot, sourceDesignDir, cfg)
	if selErr != nil {
		reason := "machinery design inventory is invalid; stop-time gates cannot be selected safely: " + selErr.Error()
		if warn != "" {
			reason = warn + "\n" + reason
		}
		return emitJSON(w, stopOut{Decision: "block", Reason: reason})
	}
	baselineCommit, baselineErr := resolveUpgradeCommit(root)
	if baselineErr != nil {
		return emitJSON(w, stopOut{Decision: "block", Reason: "machinery governance cannot bind the repository baseline to the design snapshot: " + baselineErr.Error()})
	}
	mix, mixErr := upgradeMixWarningAt(root, design, sourceDesignDir, baselineCommit)
	if mixErr != nil {
		return emitJSON(w, stopOut{Decision: "block", Reason: "machinery governance cannot prove that this change set keeps binary upgrades separate from design changes: " + mixErr.Error()})
	}
	if mix != "" {
		if selWarn != "" {
			selWarn += "\n"
		}
		selWarn += mix
	}
	if len(sel.Run) == 0 {
		if err := snapshot.CheckUnchanged(); err != nil {
			return emitJSON(w, stopOut{Decision: "block", Reason: "machinery design changed while the empty gate decision was being derived; retry the stop: " + err.Error()})
		}
		if err := clearCheckedState(root, in.SessionID, state.revision); err != nil {
			return emitJSON(w, stopOut{Decision: "block", Reason: "machinery gates could not durably clear the hook state ledger: " + err.Error()})
		}
		if selWarn != "" {
			// nothing ran, but the dropped-gates gap must stay visible
			return emitJSON(w, stopOut{SystemMessage: selWarn})
		}
		return nil
	}
	implDir := ""
	if cfg.Impl != "" {
		implDir = filepath.Join(root, filepath.FromSlash(cfg.Impl))
	}

	var buf bytes.Buffer
	blocking, drift, g4Blocking := 0, 0, 0
	// A stop-time run binds no commit: the working tree is mid-change and the
	// commit under review does not exist yet, so Ga states that non-check
	// rather than guessing. CI passes --commit and stays the outer wall.
	for _, g := range snapshot.RunSelected(implDir, sel, gates.RunOptions{}) {
		n := g.Emit(&buf)
		blocking += n
		drift += len(g.Drift)
		if strings.Contains(g.Title, "G4") {
			g4Blocking += n
		}
	}
	fmt.Fprintf(&buf, "\n%d blocking (ERROR/DRIFT) finding(s)\n", blocking)

	// Import findings block only once a baseline snapshot exists (Stage 1
	// done, or a greenfield ran `machinery baseline` when enabling impl).
	// Before that they warn: blocking a session on pre-existing boundary
	// debt it did not create invites the model to "fix" the debt by adding
	// allow rules, which is silent amnesty. Strict mode overrides.
	armed := fileExists(filepath.Join(sourceDesignDir, "ratchet.json"))
	shouldBlock := drift > 0 || (g4Blocking > 0 && armed) || (cfg.Strict && blocking > 0)
	// S10 (wave sentinel): canonical content "open" is the explicit operator
	// state that defers red gates during a multi-agent wave. Deleting it closes
	// the wave. No mtime or wall clock participates in the decision.
	left, waveStale, waveActive := waveSentinel(sourceDesignDir)
	if err := snapshot.CheckUnchanged(); err != nil {
		return emitJSON(w, stopOut{Decision: "block", Reason: "machinery design changed while the stop decision was being derived; retry the stop: " + err.Error()})
	}
	if shouldBlock {
		if stale, active := waveStale, waveActive; active {
			if stale {
				fmt.Fprintf(&buf, "\nmachinery: wave sentinel %s/.machinery-wave has invalid state; gating normally. Delete it, or replace it with canonical content 'open' as an explicit operator action.\n", design)
			} else {
				msg := fmt.Sprintf("machinery: wave sentinel state is %s; %d blocking finding(s), %d DRIFT are deferred to wave close. "+
					"Delete %s/.machinery-wave to close the wave and gate.", left, blocking, drift, design)
				if cfg.plainDialog() {
					// plain register: the user sees the state, not the plumbing
					plain := fmt.Sprintf("machinery: a review window is %s; %d design-check item(s) are held until it closes.", left, blocking)
					return emitJSON(w, stopOut{SystemMessage: plain})
				}
				return emitJSON(w, stopOut{SystemMessage: capString(msg+"\n"+buf.String(), reasonCap)})
			}
		}
	}
	switch {
	case shouldBlock:
		// keep the state so the re-check runs when the fix attempt finishes
		reason := "machinery gates are red for this session's edits.\n\n" + capString(buf.String(), reasonCap) +
			"\nFix the sources and regenerate the derived artifacts " +
			"(machinery oracle | machinery verify-formal --gen-only | machinery pack generate); " +
			"never hand-edit generated files."
		if in.StopHookActive {
			reason = "machinery gates remain red after a fix attempt; the blocking state is retained until a green check.\n\n" + capString(buf.String(), reasonCap)
		}
		if selWarn != "" {
			reason = selWarn + "\n" + reason
		}
		if warn != "" {
			reason = warn + "\n" + reason
		}
		return emitJSON(w, stopOut{Decision: "block", Reason: reason})
	case blocking > 0:
		// ERRORs without DRIFT: normal for a design mid-interrogation
		if err := clearCheckedState(root, in.SessionID, state.revision); err != nil {
			return emitJSON(w, stopOut{Decision: "block", Reason: "machinery gates ran, but the hook state ledger could not be durably cleared: " + err.Error()})
		}
		msg := fmt.Sprintf("machinery: %d gate ERROR finding(s) remain (no DRIFT); normal mid-phase. "+
			"'machinery check %s' lists them.", blocking, design)
		if g4Blocking > 0 && !armed {
			msg = fmt.Sprintf("machinery: %d gate ERROR finding(s) remain, %d of them import findings. "+
				"Import blocking is disarmed: %s/ratchet.json does not exist. Complete Stage 1 with "+
				"'machinery baseline %s --impl <dir>' (paste the printed rules, commit the ratchet) to arm enforcement.",
				blocking, g4Blocking, design, design)
		}
		if cfg.plainDialog() {
			msg = fmt.Sprintf("machinery: %d design-check item(s) are still open; normal while the design is in progress. The assistant can list and explain them.", blocking)
		}
		if selWarn != "" {
			msg = selWarn + "\n" + msg
		}
		return emitJSON(w, stopOut{SystemMessage: msg})
	default:
		if err := clearCheckedState(root, in.SessionID, state.revision); err != nil {
			return emitJSON(w, stopOut{Decision: "block", Reason: "machinery gates passed, but the hook state ledger could not be durably cleared: " + err.Error()})
		}
		if selWarn != "" {
			// a green run still surfaces the config gap, or it never surfaces
			return emitJSON(w, stopOut{SystemMessage: selWarn})
		}
		return nil
	}
}

func sameConfig(a, b Config) bool {
	hookValue := func(value *bool) (present, enabled bool) {
		if value == nil {
			return false, false
		}
		return true, *value
	}
	ap, av := hookValue(a.Hooks)
	bp, bv := hookValue(b.Hooks)
	return a.Design == b.Design && a.Gates == b.Gates && a.Impl == b.Impl && a.Strict == b.Strict && a.Dialog == b.Dialog && a.loadError == b.loadError && ap == bp && av == bv
}

// versionStampRe matches the machinery-version stamp line every generated
// artifact carries in its header.
var versionStampRe = regexp.MustCompile(`machinery-version:\s*(v?\d+\.\d+\.\d+)`)

// upgradeMixWarning reports when the working tree mixes a binary upgrade with
// a design change: at least one generated design artifact was regenerated
// under a DIFFERENT machinery-version than its committed copy, while
// hand-written design files also changed. The upgrade protocol requires the
// two causes in separate change sets, so the diff attributes to exactly one;
// the rule was stated in the skill and held by nobody. Never blocks: the
// working tree is mid-change by definition, so this stays a message. Outside
// a git repository (or with git absent) it stays silent.
var upgradeGitTimeout = 10 * time.Second

const upgradeGitOutputLimit = 16 << 20

type boundedGitBuffer struct {
	bytes.Buffer
	exceeded bool
}

func (b *boundedGitBuffer) Write(p []byte) (int, error) {
	remaining := upgradeGitOutputLimit - b.Len()
	if remaining <= 0 {
		b.exceeded = true
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = b.Buffer.Write(p[:remaining])
		b.exceeded = true
		return len(p), nil
	}
	return b.Buffer.Write(p)
}

func upgradeMixWarning(root string) (string, error) {
	const design = "design"
	commit, err := resolveUpgradeCommit(root)
	if err != nil {
		return "", err
	}
	return upgradeMixWarningAt(root, design, filepath.Join(root, filepath.FromSlash(design)), commit)
}

func resolveUpgradeCommit(root string) (string, error) {
	inside, err := hasGitMetadata(root)
	if err != nil {
		return "", fmt.Errorf("inspect repository membership: %w", err)
	}
	if !inside {
		return "", nil
	}
	out, err := runUpgradeGit(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve repository baseline commit: %w", err)
	}
	commit := strings.TrimSpace(string(out))
	if (len(commit) != 40 && len(commit) != 64) || strings.ToLower(commit) != commit {
		return "", fmt.Errorf("repository baseline resolved to noncanonical object id %q", commit)
	}
	for _, ch := range commit {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return "", fmt.Errorf("repository baseline resolved to noncanonical object id %q", commit)
		}
	}
	return commit, nil
}

// upgradeMixWarningAt compares one immutable design materialization directly
// with blobs from one exact commit. The index and ambient working tree are not
// inputs, so moving refs or transient A→B→A edits cannot create a hybrid
// decision after commit and source acquisition.
func upgradeMixWarningAt(root, design, sourceDesign, commit string) (string, error) {
	if commit == "" {
		return "", nil
	}
	out, err := runUpgradeGit(root, "ls-tree", "-r", "-z", "--name-only", commit, "--", design)
	if err != nil {
		return "", fmt.Errorf("inventory committed design at %s: %w", commit, err)
	}
	if len(out) > 0 && out[len(out)-1] != 0 {
		return "", fmt.Errorf("git ls-tree returned a non-NUL-terminated design inventory")
	}
	if count := bytes.Count(out, []byte{0}); count > hookDesignMaxEntries {
		return "", fmt.Errorf("committed design inventory exceeds %d-entry limit", hookDesignMaxEntries)
	}
	headPaths := bytes.Split(out, []byte{0})
	if len(headPaths) > 0 && len(headPaths[len(headPaths)-1]) == 0 {
		headPaths = headPaths[:len(headPaths)-1]
	}
	paths := map[string]bool{}
	headSet := map[string]bool{}
	for _, raw := range headPaths {
		rel := string(raw)
		if rel == "" || (rel != design && !strings.HasPrefix(rel, design+"/")) {
			return "", fmt.Errorf("git ls-tree returned malformed design path %q", rel)
		}
		paths[rel] = true
		headSet[rel] = true
	}
	err = dirscan.Walk(sourceDesign, hookDesignMaxEntries, func(walkPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return fmt.Errorf("immutable design entry %s is not regular", walkPath)
		}
		rel, relErr := filepath.Rel(sourceDesign, walkPath)
		if relErr != nil {
			return relErr
		}
		paths[path.Join(design, filepath.ToSlash(rel))] = true
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("inventory immutable design: %w", err)
	}
	ordered := make([]string, 0, len(paths))
	for rel := range paths {
		ordered = append(ordered, rel)
	}
	sort.Strings(ordered)
	var upgraded []string
	handWritten := 0
	for _, rel := range ordered {
		designRelPath := strings.TrimPrefix(strings.TrimPrefix(rel, design), "/")
		currentB, currentErr := safefile.Read(filepath.Join(sourceDesign, filepath.FromSlash(designRelPath)), "immutable design path", hookMarkerMaxBytes)
		currentExists := currentErr == nil
		if currentErr != nil && !errors.Is(currentErr, fs.ErrNotExist) {
			return "", fmt.Errorf("read immutable design path %s: %w", rel, currentErr)
		}
		var oldB []byte
		oldExists := headSet[rel]
		if oldExists {
			var oldErr error
			oldB, oldErr = runUpgradeGit(root, "show", commit+":"+rel)
			if oldErr != nil {
				return "", fmt.Errorf("read committed form of %s: %w", rel, oldErr)
			}
		}
		if currentExists == oldExists && bytes.Equal(currentB, oldB) {
			continue
		}
		current := string(currentB)
		cur := versionStampRe.FindString(current)
		if generatedReason(design, rel) == "" && cur == "" {
			handWritten++
			continue
		}
		if cur == "" || !oldExists {
			continue
		}
		if old := versionStampRe.FindString(string(oldB)); old != "" && old != cur {
			upgraded = append(upgraded, rel)
		}
	}
	if len(upgraded) == 0 || handWritten == 0 {
		return "", nil
	}
	sort.Strings(upgraded)
	return fmt.Sprintf("machinery: this change set mixes a binary upgrade (%s regenerated under a new machinery-version) with %d hand-written design edit(s); never mix an upgrade with a design change, the diff must attribute to exactly one cause (commit the upgrade alone first)", strings.Join(upgraded, ", "), handWritten), nil
}

func hasGitMetadata(root string) (bool, error) {
	dir, err := filepath.Abs(root)
	if err != nil {
		return false, err
	}
	for {
		marker := filepath.Join(dir, ".git")
		info, statErr := os.Lstat(marker)
		if statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
				return false, fmt.Errorf("%s must be a real directory or regular gitdir file", marker)
			}
			return true, nil
		}
		if !os.IsNotExist(statErr) {
			return false, statErr
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false, nil
		}
		dir = parent
	}
}

func runUpgradeGit(root string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), upgradeGitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	cmd.Env = upgradeGitEnvironment(os.Environ())
	var stdout, stderr boundedGitBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := processcontrol.Run(ctx, cmd)
	if stdout.exceeded || stderr.exceeded {
		return nil, fmt.Errorf("git %s output exceeds %d bytes", strings.Join(args, " "), upgradeGitOutputLimit)
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("git %s timed out after %s: %w", strings.Join(args, " "), upgradeGitTimeout, ctx.Err())
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		if len(detail) > 1200 {
			detail = detail[:1200] + "..."
		}
		if detail == "" {
			return nil, fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
		}
		return nil, fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, detail)
	}
	if detail := strings.TrimSpace(stderr.String()); detail != "" {
		return nil, fmt.Errorf("git %s emitted stderr on success: %s", strings.Join(args, " "), capString(detail, 1200))
	}
	return stdout.Bytes(), nil
}

func upgradeGitEnvironment(environ []string) []string {
	return gitcontrol.Environment(environ)
}

// selectGates picks the suite for a stop-time check: the staged list from
// the config when present, otherwise whichever gates have artifacts to
// check, so a half-built design is never failed for phases not yet reached.
// The returned warning names any impl-facing gates (g4, gt) the staged list
// requested but no impl setting backs: a stop-time hook must not hard-fail
// the whole check for a config gap, but the drop has to stay visible.
func selectGates(designDir string, cfg Config) (gates.Selection, string) {
	sel, warn, _ := selectGatesChecked(designDir, cfg)
	return sel, warn
}

func selectGatesChecked(designDir string, cfg Config) (gates.Selection, string, error) {
	snapshot, err := gates.AcquireSnapshot(designDir)
	if err != nil {
		return gates.Selection{}, "", err
	}
	sel, warn, selectErr := selectGatesCheckedInSnapshot(snapshot, designDir, cfg)
	return sel, warn, errors.Join(selectErr, snapshot.Release())
}

func selectGatesCheckedInSnapshot(snapshot *gates.Snapshot, designDir string, cfg Config) (gates.Selection, string, error) {
	// Always cross the CLI's universal inventory boundary first, including in
	// progressive auto-selection mode. Otherwise a symlinked or unreadable
	// activation marker can make its own gate disappear before RunSelected
	// ever has a chance to report it.
	if _, err := snapshot.Select("", cfg.Impl); err != nil {
		return gates.Selection{}, "", err
	}
	if cfg.Gates != "" {
		sel, err := snapshot.Select(cfg.Gates, cfg.Impl)
		if err != nil {
			return gates.Selection{}, "", err
		}
		warn := ""
		if cfg.Impl == "" {
			var dropped []string
			for _, gate := range []string{"g4", "gt"} {
				if sel.Run[gate] {
					delete(sel.Run, gate)
					dropped = append(dropped, gate)
				}
			}
			if len(dropped) > 0 {
				warn = fmt.Sprintf("machinery: the %s gates list names %s but no impl is configured; those gates were skipped (set \"impl\" in %s to run them)",
					ConfigName, strings.Join(dropped, ","), ConfigName)
			}
		}
		return sel, warn, nil
	}
	run := map[string]bool{}
	if fileExists(filepath.Join(designDir, "migration.yaml")) {
		run["gm"] = true
	}
	if fileExists(filepath.Join(designDir, "legacy", "surface.yaml")) {
		run["gs"] = true
	}
	if fileExists(filepath.Join(designDir, gates.TargetSurfacesName)) {
		run["gu"] = true
	}
	if fileExists(filepath.Join(designDir, "formal", "policy.relational.yaml")) {
		run["gp"] = true
	}
	if fileExists(filepath.Join(designDir, "formal", "integrity.relational.yaml")) {
		run["gi"] = true
	}
	if fileExists(filepath.Join(designDir, "formal", "isolation.relational.yaml")) {
		run["gn"] = true
	}
	if gates.HasModelith(designDir) {
		// the carrier gate is checkable from the domain model alone, and the
		// CLI default suite arms it the same way; omitting it here let a
		// carrier defect introduced mid-session pass the stop unexamined
		run["gc"] = true
	}
	if fileExists(filepath.Join(designDir, "workspace.dsl")) || fileExists(filepath.Join(designDir, "ARCHITECTURE.md")) {
		run["g2"] = true
	}
	// ledger-format and house-style findings are checkable from turn one and
	// never block on their own (warn tier plus rare format errors)
	run["gl"] = true
	if gates.AdjudicationActive(designDir) {
		run["gj"] = true
	}
	// machine detection never uses filepath.Glob: a project path carrying
	// glob metacharacters ([ ] * ?) once defeated it, silently dropping g3
	// and letting committed-oracle DRIFT pass at stop time (GATE-3)
	hasMachines := gates.HasMachines(designDir)
	if hasMachines {
		run["g3"] = true
		// stable-id citations are checkable as soon as oracles exist; the CLI
		// default suite arms gd on machines, and the stop hook must match
		run["gd"] = true
	}
	if hasMachines && gates.HasModelith(designDir) {
		// Gx activates on its sources (model + machines), NOT on BUILD.md:
		// waiting for Phase 4 left a window where phase-3 Gx DRIFT (a stale
		// maps-to reference) escaped the drift-blocking contract (GATE-6).
		// A machine-less decomposed parent still skips it: the children
		// carry G3/Gx.
		run["gx"] = true
	}
	if fileExists(filepath.Join(designDir, "BUILD.md")) {
		// unlike Gx, the plan-shape gate applies even on a machine-less
		// decomposed parent: the manifest BUILD.md is still its artifact
		run["gb"] = true
	}
	if gates.EmbedActive(designDir) {
		// a declared embed is checkable from the documents alone, so the
		// stop hook holds it at every turn end: a copy edited on one side is
		// exactly the drift that survives a review
		run["ge"] = true
	}
	if gates.AcceptanceActive(designDir) {
		// the acceptance directory, or a milestone marked closed: either is a
		// claim that a milestone was discharged, and the claim is checkable
		run["ga"] = true
	}
	if gates.AttestationActive(designDir) {
		// committed attestation evidence is checkable from the design tree
		// alone, and its whole value is freshness: an artifact edited this
		// turn invalidates the judgment recorded over it, so the stop hook is
		// exactly where that must surface
		run["gv"] = true
	}
	if gates.HasCheckers(designDir) {
		// the external-checker layer's whole stop-time value is freshness:
		// a session that edits the domain model makes the committed projection
		// and evidence stale (DRIFT at the CLI), and the omission of gk here
		// once let exactly that DRIFT pass the turn end green
		run["gk"] = true
	}
	if cfg.Impl != "" {
		run["g4"] = true
		run["gt"] = true
	}
	if pack.HasDecomposition(designDir) || pack.HasPack(designDir) {
		run["g5"] = true
	}
	return gates.Selection{Run: run, Explicit: true}, "", nil
}

// --- SessionStart: announce governance so every session knows the contract ---

func sessionStart(w io.Writer, root string, cfg Config, warn string) error {
	design := designRel(cfg)
	designDir := filepath.Join(root, filepath.FromSlash(design))
	if cfg.snapshotDesign != "" {
		designDir = cfg.snapshotDesign
	}
	var b strings.Builder
	if warn != "" {
		b.WriteString(warn + "\n")
	}
	b.WriteString("This repository is machinery-managed: design governance is active (machinery plugin).\n")
	fmt.Fprintf(&b, "- Design directory: %s/\n", design)
	if cfg.Gates != "" {
		fmt.Fprintf(&b, "- Staged gate list (from %s): %s\n", ConfigName, cfg.Gates)
	}
	if cfg.Impl != "" {
		state := "no ratchet.json baseline yet, so import findings warn only; run 'machinery baseline' to arm blocking"
		if fileExists(filepath.Join(designDir, "ratchet.json")) {
			state = "baseline recorded, violations block at turn end"
		}
		fmt.Fprintf(&b, "- Import-boundary gate G4 watches source edits under %s (%s).\n", cfg.Impl, state)
	}
	fmt.Fprintf(&b, "- Generated artifacts are read-only and hooks deny edits to them: %s/**/*.oracle.md, %s/formal/*.tla, *.cfg and *.als, %s/packs/**, %s/pack/**, %s/ratchet.json. Edit the sources, then regenerate (machinery oracle | machinery verify-formal --gen-only | machinery alloy | machinery pack generate | machinery baseline).\n",
		design, design, design, design, design)
	fmt.Fprintf(&b, "- The wave sentinel %s/%s is operator-created: hooks deny agent writes to it, because a session that could touch it would defer its own gates. Deleting it (closing the wave) stays allowed.\n",
		design, waveSentinelName)
	mode := "stale generated artifacts (DRIFT) or import-boundary violations block"
	if cfg.Strict {
		mode = "strict mode: any blocking finding blocks"
	}
	fmt.Fprintf(&b, "- When a turn edits the design or watched sources, 'machinery check' runs before the turn can end; %s.\n", mode)
	b.WriteString("- Design work runs through the 'machinery' skill: four phases, each behind a gate.\n")
	if cfg.plainDialog() {
		b.WriteString("- Dialog register: PLAIN (set in " + ConfigName + "). Everything in this notice is conductor context, " +
			"never language to repeat to the user: speak in plain step names, translate findings to their meaning, and keep " +
			"gate ids, phase numbers, and CLI invocations out of the conversation unless the user uses them first.\n")
	}
	raw, present, readErr := readConfinedRegular(designDir, "STATE.md", hookMarkerMaxBytes)
	if readErr != nil {
		return fmt.Errorf("read session ledger: %w", readErr)
	}
	if present {
		const maxLines = 30
		lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
		trunc := ""
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			trunc = "\n(... truncated; read the file for the rest)"
		}
		fmt.Fprintf(&b, "- Session ledger %s/STATE.md:\n%s%s\n", design, strings.Join(lines, "\n"), trunc)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// --- session state ledger (what this session touched) ---

// statePath keys the dirty obligation by canonical project root, deliberately
// not by host session. A crashed process, reboot, or replacement session must
// inherit the unfinished gate obligation; only a successful stop-time check
// clears it. The session parameter remains in this internal API so old hosts
// and focused tests cannot accidentally select a second storage scheme.
func statePath(root, _ string) string {
	p, err := statePathExact(root)
	if err == nil {
		return p
	}
	absRoot := filepath.Clean(root)
	dir := stateDirPath()
	return filepath.Join(dir, stateFileName(absRoot))
}

func statePathExact(root string) (string, error) {
	absRoot, err := filelock.ScopeIdentity(root)
	if err != nil {
		return "", err
	}
	dir, err := stateDirPathExact()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stateFileName(absRoot)), nil
}

func stateFileName(absRoot string) string {
	h := sha256.New()
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(absRoot)))
	_, _ = h.Write(size[:])
	_, _ = h.Write([]byte(absRoot))
	return fmt.Sprintf("%x.state", h.Sum(nil))
}

func routeStatePrefix(root string) string { return statePath(root, "") + ".route-" }

func routeStatePath(root, sessionID string) string {
	h := sha256.New()
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(sessionID)))
	_, _ = h.Write(size[:])
	_, _ = h.Write([]byte(sessionID))
	return routeStatePrefix(root) + fmt.Sprintf("%x", h.Sum(nil)) + ".json"
}

func routeStatePaths(root string) ([]string, error) {
	dir := stateDirPath()
	prefix := filepath.Base(routeStatePrefix(root))
	return boundedHookStateMatches(dir, "hook route snapshot", func(name string) (bool, error) {
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".json") {
			return false, nil
		}
		digest := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".json")
		if !validHookHexDigest(digest) {
			return false, fmt.Errorf("hook route snapshot %q has a noncanonical filename", name)
		}
		return true, nil
	})
}

func routeStateTemps(root string) ([]string, error) {
	dir := stateDirPath()
	prefix := "." + filepath.Base(routeStatePrefix(root))
	return boundedHookStateMatches(dir, "hook route temp", func(name string) (bool, error) {
		if !strings.HasPrefix(name, prefix) {
			return false, nil
		}
		rest := strings.TrimPrefix(name, prefix)
		separator := strings.Index(rest, ".json.tmp-")
		if separator < 0 {
			return false, nil
		}
		if !validHookHexDigest(rest[:separator]) || rest[separator+len(".json.tmp-"):] == "" {
			return false, fmt.Errorf("hook route temp %q has a noncanonical filename", name)
		}
		return true, nil
	})
}

func validHookHexDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func boundedHookStateMatches(dir, kind string, match func(string) (bool, error)) ([]string, error) {
	entries, err := dirscan.Read(dir, hookStateDirMaxEntries)
	if err != nil {
		return nil, fmt.Errorf("inventory %s files: %w", kind, err)
	}
	paths := make([]string, 0)
	for _, entry := range entries {
		matched, err := match(entry.Name())
		if err != nil {
			return nil, err
		}
		if !matched {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s %s must be a regular file, not a symlink or special file", kind, path)
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func stateDirPath() string {
	dir, _ := stateDirPathExact()
	return dir
}

const (
	// stateInitializationMarkerBody is the pre-directory-binding marker kept
	// solely for fail-closed migration of existing installations.
	stateInitializationMarkerBody = "machinery-hook-state-v1\n"
	stateDirectoryIdentityName    = ".store-identity"
)

type stateDirectoryBinding struct {
	native     string
	generation string
}

func (b stateDirectoryBinding) markerBody() []byte {
	return []byte("machinery-hook-state-v2\ndirectory " + b.native + "\ngeneration " + b.generation + "\n")
}

func (b stateDirectoryBinding) identityBody() []byte {
	return []byte("machinery-hook-state-directory-v1\ndirectory " + b.native + "\ngeneration " + b.generation + "\n")
}

func parseStateDirectoryBinding(body []byte, header string) (stateDirectoryBinding, error) {
	lines := strings.Split(string(body), "\n")
	if len(lines) != 4 || lines[0] != header || !strings.HasPrefix(lines[1], "directory ") || !strings.HasPrefix(lines[2], "generation ") || lines[3] != "" {
		return stateDirectoryBinding{}, fmt.Errorf("expected canonical %s record", header)
	}
	binding := stateDirectoryBinding{
		native:     strings.TrimPrefix(lines[1], "directory "),
		generation: strings.TrimPrefix(lines[2], "generation "),
	}
	if !validHookNativeIdentity(binding.native) {
		return stateDirectoryBinding{}, fmt.Errorf("directory identity is not canonical")
	}
	if len(binding.generation) != 64 || binding.generation != strings.ToLower(binding.generation) {
		return stateDirectoryBinding{}, fmt.Errorf("directory generation is not canonical")
	}
	if _, err := hex.DecodeString(binding.generation); err != nil {
		return stateDirectoryBinding{}, fmt.Errorf("directory generation is not canonical")
	}
	return binding, nil
}

func validHookNativeIdentity(identity string) bool {
	if len(identity) < 8 || len(identity) > 256 {
		return false
	}
	for _, char := range identity {
		if char != ':' && (char < '0' || char > '9') && (char < 'a' || char > 'z') {
			return false
		}
	}
	return true
}

func stateInitializationMarkerPath() (string, error) {
	home, key, err := hookStateHomeIdentity()
	if err != nil {
		return "", err
	}
	// The loss sentinel must not share a parent with the store it protects.
	// Removing the complete config directory must leave durable proof that
	// this is not a first initialization.
	return filepath.Join(home, ".machinery-hook-state-"+key+".initialized"), nil
}

func legacyStateInitializationMarkerPath() (string, error) {
	dir, err := stateDirPathExact()
	if err != nil {
		return "", err
	}
	return dir + ".initialized", nil
}

func hookStateHomeIdentity() (string, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("resolve absolute user home for hook state: %w", err)
	}
	if home == "" || !filepath.IsAbs(home) {
		return "", "", fmt.Errorf("resolve absolute user home for hook state: no absolute home is configured")
	}
	homeInfo, err := os.Lstat(home)
	if err != nil {
		return "", "", fmt.Errorf("inspect user home for hook state: %w", err)
	}
	if homeInfo.Mode()&os.ModeSymlink != 0 || !homeInfo.IsDir() {
		return "", "", fmt.Errorf("user home for hook state must be a real directory")
	}
	sum := sha256.Sum256([]byte("machinery-hook-state\x00" + home))
	return home, fmt.Sprintf("%x", sum[:12]), nil
}

func stateDirPathExact() (string, error) {
	home, key, err := hookStateHomeIdentity()
	if err != nil {
		return "", err
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve absolute user config directory for hook state: %w", err)
	}
	if configDir == "" || !filepath.IsAbs(configDir) {
		return "", fmt.Errorf("resolve absolute user config directory for hook state: no absolute config directory is configured")
	}
	if !stateMarkerHasIndependentParent(home, configDir) {
		return "", fmt.Errorf("user config directory for hook state cannot contain the user home; no independent durable loss sentinel is possible")
	}
	configInfo, err := os.Lstat(configDir)
	if os.IsNotExist(err) {
		return filepath.Join(configDir, "machinery-hook-state-"+key), nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect user config directory for hook state: %w", err)
	}
	if configInfo.Mode()&os.ModeSymlink != 0 || !configInfo.IsDir() {
		return "", fmt.Errorf("user config directory for hook state must be a real directory")
	}
	return filepath.Join(configDir, "machinery-hook-state-"+key), nil
}

func stateMarkerHasIndependentParent(home, configDir string) bool {
	rel, err := filepath.Rel(filepath.Clean(configDir), filepath.Clean(home))
	if err != nil {
		return true
	}
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func ensureStateDir() (returnErr error) {
	dir, err := stateDirPathExact()
	if err != nil {
		return err
	}
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		return err
	}
	legacyMarker, err := legacyStateInitializationMarkerPath()
	if err != nil {
		return err
	}
	configDir := filepath.Dir(dir)
	// Check loss before acquiring the advisory lock: its implementation uses
	// the user cache, which an operator may have configured beneath the config
	// directory. Lock setup must not recreate the very missing parent whose
	// absence the independent marker proves.
	preexistingMarker, err := inspectStateInitializationMarker(marker)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(configDir); os.IsNotExist(err) && preexistingMarker {
		return fmt.Errorf("durable hook state parent directory %s is missing after prior initialization; refusing to recreate it as empty", configDir)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect user config directory for hook state: %w", err)
	}
	lock, err := acquireHookFileLock(marker, "hook state initialization marker")
	if err != nil {
		return err
	}
	defer func() { returnErr = errors.Join(returnErr, lock.Release()) }()
	markerPresent, markerLegacy, markerBinding, err := readStateInitializationMarker(marker)
	if err != nil {
		return err
	}
	legacyMarkerPresent, _, _, err := readStateInitializationMarker(legacyMarker)
	if err != nil {
		return err
	}
	initialized := markerPresent || legacyMarkerPresent
	configInfo, err := os.Lstat(configDir)
	if os.IsNotExist(err) {
		if initialized {
			return fmt.Errorf("durable hook state parent directory %s is missing after prior initialization; refusing to recreate it as empty", configDir)
		}
		if err := os.MkdirAll(configDir, 0o700); err != nil {
			return fmt.Errorf("create user config directory for hook state: %w", err)
		}
		configInfo, err = os.Lstat(configDir)
	}
	if err != nil {
		return fmt.Errorf("inspect user config directory for hook state: %w", err)
	}
	if configInfo.Mode()&os.ModeSymlink != 0 || !configInfo.IsDir() {
		return fmt.Errorf("user config directory for hook state must be a real directory")
	}
	info, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		if initialized {
			return fmt.Errorf("durable hook state directory %s is missing after prior initialization; refusing to recreate it as empty", dir)
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		info, err = os.Lstat(dir)
	}
	if err != nil {
		return err
	}
	if err := filelock.ValidatePrivateDir(dir, info); err != nil {
		return err
	}
	binding, err := ensureStateDirectoryIdentity(dir, markerPresent && !markerLegacy, markerBinding)
	if err != nil {
		return err
	}
	if markerPresent && !markerLegacy {
		return nil
	}
	// Migrate a legacy constant marker only while its real private state
	// directory is present and has been rebound to a durable native identity
	// plus a store-local generation. The independent marker remains the source
	// of truth after the legacy sibling is left in place for rollback safety.
	return writeStateInitializationMarker(marker, binding, markerPresent)
}

// requireStateDir distinguishes first safe initialization from loss of a
// previously initialized durable store. The marker lives at the user-home
// root, outside the config parent it protects, so deleting or replacing that
// entire parent cannot turn a dirty project into an untouched one. Existing
// sibling-marker installations are migrated only while their real private
// directory is still present.
func requireStateDir() error {
	return ensureStateDir()
}

func ensureStateDirectoryIdentity(dir string, requireExisting bool, expected stateDirectoryBinding) (stateDirectoryBinding, error) {
	native, err := captureStateDirectoryIdentity(dir)
	if err != nil {
		return stateDirectoryBinding{}, err
	}
	identityPath := filepath.Join(dir, stateDirectoryIdentityName)
	witness, err := readBoundedHookFile(identityPath, "hook state directory identity", hookStateIdentityMaxBytes)
	if err != nil {
		return stateDirectoryBinding{}, err
	}
	if witness == nil {
		if requireExisting {
			return stateDirectoryBinding{}, fmt.Errorf("durable hook state directory %s no longer contains its bound identity; refusing to accept a replacement store", dir)
		}
		generationBytes := make([]byte, 32)
		if _, err := io.ReadFull(cryptorand.Reader, generationBytes); err != nil {
			return stateDirectoryBinding{}, fmt.Errorf("generate hook state directory identity: %w", err)
		}
		binding := stateDirectoryBinding{native: native, generation: hex.EncodeToString(generationBytes)}
		file, err := os.OpenFile(identityPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return stateDirectoryBinding{}, err
		}
		if _, err := file.Write(binding.identityBody()); err != nil {
			return stateDirectoryBinding{}, errors.Join(err, file.Close())
		}
		if err := file.Sync(); err != nil {
			return stateDirectoryBinding{}, errors.Join(err, file.Close())
		}
		if err := file.Close(); err != nil {
			return stateDirectoryBinding{}, err
		}
		if err := syncStateDirectory(dir); err != nil {
			return stateDirectoryBinding{}, err
		}
		return binding, nil
	}
	binding, err := parseStateDirectoryBinding(witness.body, "machinery-hook-state-directory-v1")
	if err != nil {
		return stateDirectoryBinding{}, fmt.Errorf("hook state directory identity %s is corrupt or noncanonical: %w", identityPath, err)
	}
	if binding.native != native {
		return stateDirectoryBinding{}, fmt.Errorf("durable hook state directory %s changed native identity; refusing to accept a replacement store", dir)
	}
	if requireExisting && binding != expected {
		return stateDirectoryBinding{}, fmt.Errorf("durable hook state directory %s does not match its independent initialization marker; refusing to accept a replacement store", dir)
	}
	return binding, nil
}

func captureStateDirectoryIdentity(dir string) (native string, returnErr error) {
	before, err := os.Lstat(dir)
	if err != nil {
		return "", err
	}
	if err := filelock.ValidatePrivateDir(dir, before); err != nil {
		return "", err
	}
	file, err := os.Open(dir)
	if err != nil {
		return "", err
	}
	defer func() { returnErr = errors.Join(returnErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil || !opened.IsDir() || !os.SameFile(before, opened) || before.Mode() != opened.Mode() {
		return "", errors.Join(err, fmt.Errorf("durable hook state directory %s changed identity while opening", dir))
	}
	native, err = hookNativeDirectoryWitness(file, opened)
	if err != nil {
		return "", err
	}
	hookStateDirectoryPhase("after-open", dir)
	openedAfter, openedErr := file.Stat()
	pathAfter, pathErr := os.Lstat(dir)
	if err := errors.Join(openedErr, pathErr); err != nil {
		return "", err
	}
	afterNative, nativeErr := hookNativeDirectoryWitness(file, openedAfter)
	if nativeErr != nil || !openedAfter.IsDir() || !pathAfter.IsDir() || pathAfter.Mode()&os.ModeSymlink != 0 ||
		!os.SameFile(opened, openedAfter) || !os.SameFile(opened, pathAfter) || opened.Mode() != openedAfter.Mode() || opened.Mode() != pathAfter.Mode() || native != afterNative {
		return "", errors.Join(nativeErr, fmt.Errorf("durable hook state directory %s changed identity during validation", dir))
	}
	return native, nil
}

func writeStateInitializationMarker(marker string, binding stateDirectoryBinding, replace bool) (returnErr error) {
	tmp, err := os.CreateTemp(filepath.Dir(marker), "."+filepath.Base(marker)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		if err := removeOwnedHookFile(tmpPath, "hook state initialization marker temp", hookStateMarkerMaxBytes); err != nil {
			returnErr = errors.Join(returnErr, err)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if _, err := tmp.Write(binding.markerBody()); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Sync(); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	hookStateMarkerPhase("created")
	if !replace {
		if _, err := os.Lstat(marker); err == nil {
			return fmt.Errorf("hook state initialization marker %s appeared during creation", marker)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return replaceStateFile(tmpPath, marker)
}

func validateStateInitializationMarker(marker string) (bool, error) {
	present, _, _, err := readStateInitializationMarker(marker)
	return present, err
}

func readStateInitializationMarker(marker string) (present, legacy bool, binding stateDirectoryBinding, returnErr error) {
	witness, err := readBoundedHookFile(marker, "hook state initialization marker", hookStateMarkerMaxBytes)
	// The legacy marker is rooted next to the state directory, whose complete
	// parent is legitimately absent before the first initialization. The
	// independent home-rooted marker is checked separately and makes that same
	// absence fail closed after initialization.
	if errors.Is(err, os.ErrNotExist) {
		return false, false, stateDirectoryBinding{}, nil
	}
	if err != nil {
		return true, false, stateDirectoryBinding{}, err
	}
	if witness == nil {
		return false, false, stateDirectoryBinding{}, nil
	}
	if string(witness.body) == stateInitializationMarkerBody {
		return true, true, stateDirectoryBinding{}, nil
	}
	binding, err = parseStateDirectoryBinding(witness.body, "machinery-hook-state-v2")
	if err != nil {
		return true, false, stateDirectoryBinding{}, fmt.Errorf("hook state initialization marker %s is corrupt or noncanonical: %w", marker, err)
	}
	return true, false, binding, nil
}

// inspectStateInitializationMarker binds lock routing and loss detection to
// the marker pathname without reading bytes that another lock holder may
// still be writing. Its content is validated only after acquiring that same
// marker-scoped lock.
func inspectStateInitializationMarker(marker string) (bool, error) {
	info, err := os.Lstat(marker)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return true, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return true, fmt.Errorf("hook state initialization marker %s must be a regular file, not a symlink or special file", marker)
	}
	return true, nil
}

func durableProjectStatePresent(root string) (bool, error) {
	stateFile, err := statePathExact(root)
	if err != nil {
		// Without an addressable store there can be no candidate-specific
		// durable evidence. Mutable markers are checked before this function;
		// they still make managed events fail closed in requireStateDir. Do not
		// turn a missing HOME/config root into output in every unrelated repo.
		return unavailableDurableProjectState()
	}
	// The store-wide initialization marker must not disturb an unrelated,
	// unmanaged project. Establish candidate-specific evidence first; only a
	// ledger, route, or crash temp whose filename is bound to this canonical
	// root authorizes validating the global store on this ascent step.
	candidatePresent := false
	if _, err := os.Lstat(stateFile); err == nil {
		candidatePresent = true
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if _, err := os.Lstat(filepath.Dir(stateFile)); errors.Is(err, os.ErrNotExist) {
		return candidatePresent, nil
	} else if err != nil {
		return false, err
	}
	for _, inventory := range []func() ([]string, error){
		func() ([]string, error) { return routeStatePaths(root) },
		func() ([]string, error) { return hookStateTemps(stateFile) },
		func() ([]string, error) { return routeStateTemps(root) },
		func() ([]string, error) { return hookProjectQuarantinePaths(stateFile) },
	} {
		matches, inventoryErr := inventory()
		if inventoryErr != nil {
			return false, inventoryErr
		}
		candidatePresent = candidatePresent || len(matches) > 0
	}
	if !candidatePresent {
		return false, nil
	}
	if err := requireStateDir(); err != nil {
		return false, err
	}
	if raw, err := readStateFile(stateFile); err != nil {
		return false, err
	} else if raw != nil {
		if _, err := parseHookStateRecord(raw); err != nil {
			return false, err
		}
		return true, nil
	}
	for _, inventory := range []func() ([]string, error){
		func() ([]string, error) { return routeStatePaths(root) },
		func() ([]string, error) { return hookStateTemps(stateFile) },
		func() ([]string, error) { return routeStateTemps(root) },
		func() ([]string, error) { return hookProjectQuarantinePaths(stateFile) },
	} {
		matches, inventoryErr := inventory()
		if inventoryErr != nil {
			return false, inventoryErr
		}
		if len(matches) > 0 {
			return true, nil
		}
	}
	return false, nil
}

// unavailableDurableProjectState deliberately maps an unaddressable global
// store to absence while walking otherwise-unmanaged ancestors. A mutable
// managed marker is checked before this path and still fails closed.
func unavailableDurableProjectState() (bool, error) { return false, nil }

func readStateFile(path string) ([]byte, error) {
	witness, err := readBoundedHookStateFile(path, "hook state", hookStateMaxBytes)
	if witness == nil || err != nil {
		return nil, err
	}
	return witness.body, nil
}

func readRouteStateFile(path string) ([]byte, error) {
	witness, err := readBoundedHookStateFile(path, "hook route snapshot", hookRouteMaxBytes)
	if witness == nil || err != nil {
		return nil, err
	}
	return witness.body, nil
}

type hookFileWitness struct {
	body     []byte
	info     os.FileInfo
	mode     os.FileMode
	size     int64
	modTime  time.Time
	changeID string
}

func readBoundedHookFile(path, kind string, limit int64) (witness *hookFileWitness, retErr error) {
	return readBoundedHookFileExpectedParent(path, kind, limit, "")
}

func readBoundedHookStateFile(path, kind string, limit int64) (*hookFileWitness, error) {
	binding, err := validatedStateDirectoryBinding()
	if err != nil {
		return nil, err
	}
	hookStateBindingPhase("validated", stateDirPath())
	return readBoundedHookFileExpectedParent(path, kind, limit, binding.native)
}

func validatedStateDirectoryBinding() (binding stateDirectoryBinding, returnErr error) {
	dir, err := stateDirPathExact()
	if err != nil {
		return stateDirectoryBinding{}, err
	}
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		return stateDirectoryBinding{}, err
	}
	lock, err := acquireHookFileLock(marker, "hook state initialization marker")
	if err != nil {
		return stateDirectoryBinding{}, err
	}
	defer func() { returnErr = errors.Join(returnErr, lock.Release()) }()
	present, legacy, expected, err := readStateInitializationMarker(marker)
	if err != nil {
		return stateDirectoryBinding{}, err
	}
	if !present || legacy {
		return stateDirectoryBinding{}, fmt.Errorf("hook state initialization marker is missing its bound directory identity")
	}
	return ensureStateDirectoryIdentity(dir, true, expected)
}

func readBoundedHookFileExpectedParent(path, kind string, limit int64, expectedParentNative string) (witness *hookFileWitness, retErr error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%s read limit must be positive", kind)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	parent, name := filepath.Dir(abs), filepath.Base(abs)
	parentBefore, err := os.Lstat(parent)
	if err != nil {
		return nil, err
	}
	if parentBefore.Mode()&os.ModeSymlink != 0 || !parentBefore.IsDir() {
		return nil, fmt.Errorf("%s parent %s must be a real directory", kind, parent)
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	parentInside, err := root.Lstat(".")
	if err != nil || !os.SameFile(parentBefore, parentInside) || parentBefore.Mode() != parentInside.Mode() {
		return nil, errors.Join(err, fmt.Errorf("%s parent %s changed while opening", kind, parent))
	}
	if expectedParentNative != "" {
		parentHandle, err := root.Open(".")
		if err != nil {
			return nil, err
		}
		openedParent, statErr := parentHandle.Stat()
		if statErr != nil {
			return nil, errors.Join(statErr, parentHandle.Close())
		}
		parentNative, nativeErr := hookNativeDirectoryWitness(parentHandle, openedParent)
		closeErr := parentHandle.Close()
		if err := errors.Join(statErr, nativeErr, closeErr); err != nil {
			return nil, err
		}
		if !os.SameFile(parentInside, openedParent) || parentNative != expectedParentNative {
			return nil, fmt.Errorf("%s parent %s does not match the bound hook state directory identity", kind, parent)
		}
	}
	before, err := root.Lstat(name)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("%s %s must be a regular file, not a symlink or special file", kind, abs)
	}
	if before.Size() > limit {
		return nil, fmt.Errorf("%s %s exceeds %d-byte limit", kind, abs, limit)
	}
	hookStateReadPhase("after-lstat", abs)
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) || before.Mode() != opened.Mode() || before.Size() != opened.Size() || !before.ModTime().Equal(opened.ModTime()) {
		return nil, errors.Join(err, fmt.Errorf("%s %s changed identity or metadata while opening", kind, abs))
	}
	beforeChangeID := hookFileChangeID(before)
	openedChangeID := hookFileChangeID(opened)
	if beforeChangeID != "" && openedChangeID != "" && beforeChangeID != openedChangeID {
		return nil, fmt.Errorf("%s %s changed modification identity while opening", kind, abs)
	}
	if opened.Size() > limit {
		return nil, fmt.Errorf("%s %s exceeds %d-byte limit", kind, abs, limit)
	}
	hookStateReadPhase("after-open", abs)
	body, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%s %s exceeds %d-byte limit", kind, abs, limit)
	}
	hookStateReadPhase("after-read", abs)
	openedAfter, openedErr := file.Stat()
	pathAfter, pathErr := root.Lstat(name)
	parentAfter, parentErr := os.Lstat(parent)
	if err := errors.Join(openedErr, pathErr, parentErr); err != nil {
		return nil, err
	}
	afterChangeID := hookFileChangeID(openedAfter)
	pathChangeID := hookFileChangeID(pathAfter)
	if !openedAfter.Mode().IsRegular() || pathAfter.Mode()&os.ModeSymlink != 0 || !pathAfter.Mode().IsRegular() ||
		!os.SameFile(opened, openedAfter) || !os.SameFile(opened, pathAfter) || !os.SameFile(parentInside, parentAfter) ||
		opened.Mode() != openedAfter.Mode() || opened.Mode() != pathAfter.Mode() || opened.Size() != openedAfter.Size() || opened.Size() != pathAfter.Size() ||
		!opened.ModTime().Equal(openedAfter.ModTime()) || !opened.ModTime().Equal(pathAfter.ModTime()) ||
		(openedChangeID != "" && afterChangeID != "" && openedChangeID != afterChangeID) ||
		(openedChangeID != "" && pathChangeID != "" && openedChangeID != pathChangeID) {
		return nil, fmt.Errorf("%s %s changed identity, metadata, or content while being read", kind, abs)
	}
	return &hookFileWitness{body: body, info: pathAfter, mode: pathAfter.Mode(), size: pathAfter.Size(), modTime: pathAfter.ModTime(), changeID: pathChangeID}, nil
}

func hookFileChangeID(info os.FileInfo) string {
	if info == nil || info.Sys() == nil {
		return ""
	}
	value := reflect.ValueOf(info.Sys())
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return ""
	}
	for _, name := range []string{"Ctim", "Ctimespec"} {
		field := value.FieldByName(name)
		if field.IsValid() && field.Kind() == reflect.Struct {
			sec, nsec := field.FieldByName("Sec"), field.FieldByName("Nsec")
			if sec.IsValid() && nsec.IsValid() && sec.CanInt() && nsec.CanInt() {
				return fmt.Sprintf("%d:%d", sec.Int(), nsec.Int())
			}
		}
	}
	ctime, ctimeNsec := value.FieldByName("Ctime"), value.FieldByName("Ctimensec")
	if ctime.IsValid() && ctimeNsec.IsValid() && ctime.CanInt() && ctimeNsec.CanInt() {
		return fmt.Sprintf("%d:%d", ctime.Int(), ctimeNsec.Int())
	}
	return ""
}

func armShellState(root string, in Input, cfg Config) error {
	return armProjectState(root, in, cfg, true, cfg.Impl != "")
}

// armFileState closes the PreToolUse -> tool execution -> PostToolUse crash
// window. Once an allowed file edit is about to touch a governed tree, the
// project-wide obligation exists before the tool starts and survives a lost
// Post event or a replacement host session.
func armFileState(root string, cfg Config, in Input) error {
	design := designRel(cfg)
	designTouched, implTouched := false, false
	for _, edited := range editedPaths(in) {
		rel := relToRoot(root, resolveEventPath(in.Cwd, edited))
		switch {
		case rel == design || strings.HasPrefix(rel, design+"/"):
			designTouched = true
		case cfg.Impl != "" && sourceExt[path.Ext(rel)] && underImpl(cfg, rel):
			implTouched = true
		}
	}
	if !designTouched && !implTouched {
		return nil
	}
	return armProjectState(root, in, cfg, designTouched, implTouched)
}

func armProjectState(root string, in Input, cfg Config, designTouched, implTouched bool) (returnErr error) {
	operation, err := toolOperationToken(in)
	if err != nil {
		return err
	}
	raw, err := routeSnapshotBody(cfg)
	if err != nil {
		return err
	}
	if err := ensureStateDir(); err != nil {
		return err
	}
	p := statePath(root, in.SessionID)
	lock, err := acquireHookStateLock(p)
	if err != nil {
		return err
	}
	defer func() { returnErr = errors.Join(returnErr, lock.release()) }()
	if temps, err := routeStateTemps(root); err != nil {
		return err
	} else if len(temps) > 0 {
		return fmt.Errorf("incomplete hook route transaction: durable temp %s exists; refusing to overwrite crash evidence", temps[0])
	}
	if err := updateStateLocked(p, designTouched, implTouched, operation, "", routeSnapshotDigest(raw)); err != nil {
		return err
	}
	return writeRouteSnapshotLocked(routeStatePath(root, in.SessionID), raw)
}

func routeSnapshotBody(cfg Config) ([]byte, error) {
	route := map[string]any{
		"design": cfg.Design, "gates": cfg.Gates, "impl": cfg.Impl,
		"strict": cfg.Strict, "dialog": cfg.Dialog,
	}
	if cfg.Hooks != nil {
		route["hooks"] = *cfg.Hooks
	}
	raw, err := json.Marshal(route)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func completeToolState(root string, cfg Config, in Input, designTouched, implTouched bool) (returnErr error) {
	operation, err := toolOperationToken(in)
	if err != nil {
		return err
	}
	raw, err := routeSnapshotBody(cfg)
	if err != nil {
		return err
	}
	if err := requireStateDir(); err != nil {
		return err
	}
	p := statePath(root, in.SessionID)
	lock, err := acquireHookStateLock(p)
	if err != nil {
		return err
	}
	defer func() { returnErr = errors.Join(returnErr, lock.release()) }()
	if temps, err := routeStateTemps(root); err != nil {
		return err
	} else if len(temps) > 0 {
		return fmt.Errorf("incomplete hook route transaction: durable temp %s exists; refusing to overwrite crash evidence", temps[0])
	}
	if err := updateStateLocked(p, designTouched, implTouched, "", operation, routeSnapshotDigest(raw)); err != nil {
		return err
	}
	return writeRouteSnapshotLocked(routeStatePath(root, in.SessionID), raw)
}

func toolOperationToken(in Input) (string, error) {
	if strings.TrimSpace(in.ToolUseID) == "" {
		return "", fmt.Errorf("hook event has no tool_use_id; an in-flight tool cannot be tracked safely")
	}
	h := sha256.New()
	for _, value := range []string{in.SessionID, in.ToolUseID} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		_, _ = h.Write(size[:])
		_, _ = h.Write([]byte(value))
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func routeSnapshotDigest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum[:])
}

func writeRouteSnapshot(root, sessionID string, body []byte) (returnErr error) {
	if err := ensureStateDir(); err != nil {
		return err
	}
	p := routeStatePath(root, sessionID)
	lock, err := acquireStateLock(statePath(root, sessionID))
	if err != nil {
		return err
	}
	defer func() { returnErr = errors.Join(returnErr, lock.release()) }()
	return writeRouteSnapshotLocked(p, body)
}

func writeRouteSnapshotLocked(p string, body []byte) error {
	return writeHookStateBody(p, body, false)
}

func writeHookStateBody(p string, body []byte, announcePhase bool) (returnErr error) {
	limit, kind := hookRouteMaxBytes, "hook route snapshot"
	if announcePhase {
		limit, kind = hookStateMaxBytes, "hook state"
	}
	if int64(len(body)) > limit {
		return fmt.Errorf("%s %s exceeds %d-byte limit", kind, p, limit)
	}
	tmp, err := os.CreateTemp(stateDirPath(), "."+filepath.Base(p)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		if err := removeOwnedHookStateTemp(tmpPath, kind, limit); err != nil {
			returnErr = errors.Join(returnErr, err)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := syncStateDirectory(stateDirPath()); err != nil {
		return err
	}
	if announcePhase {
		hookStatePhase("temp-synced")
	} else {
		hookRoutePhase("temp-synced")
	}
	return replaceHookStateFile(tmpPath, p)
}

func removeOwnedHookStateTemp(path, kind string, limit int64) error {
	witness, err := readBoundedHookStateFile(path, kind+" temp", limit)
	if err != nil {
		return err
	}
	if witness == nil {
		return nil
	}
	return removeHookFileWitness(path, kind+" temp", limit, witness)
}

func removeOwnedHookFile(path, kind string, limit int64) error {
	witness, err := readBoundedHookFile(path, kind, limit)
	if err != nil {
		return err
	}
	if witness == nil {
		return nil
	}
	return removeHookFileWitnessWithPrefix(path, kind, limit, witness, "d")
}

func loadRouteSnapshot(root, sessionID string) (Config, bool, error) {
	if err := requireStateDir(); err != nil {
		return Config{}, false, err
	}
	if temps, err := routeStateTemps(root); err != nil {
		return Config{}, false, err
	} else if len(temps) > 0 {
		return Config{}, false, fmt.Errorf("incomplete hook route transaction: durable temp %s exists", temps[0])
	}
	raw, err := readRouteStateFile(routeStatePath(root, sessionID))
	if err != nil {
		return Config{}, false, err
	}
	if raw != nil {
		cfg, err := decodeConfig(raw)
		if err != nil {
			return Config{}, false, fmt.Errorf("pre-shell route snapshot is corrupt: %w", err)
		}
		if cfg.Design == "" {
			cfg.Design = "design"
		}
		return cfg, true, nil
	}
	// A replacement session has no exact route filename. Recover the sole
	// project route, or one common route shared by every unfinished session;
	// divergent route revisions are ambiguous and therefore block.
	paths, err := routeStatePaths(root)
	if err != nil || len(paths) == 0 {
		return Config{}, false, err
	}
	var recovered Config
	for i, routePath := range paths {
		routeRaw, err := readRouteStateFile(routePath)
		if err != nil {
			return Config{}, false, err
		}
		cfg, err := decodeConfig(routeRaw)
		if err != nil {
			return Config{}, false, fmt.Errorf("pre-shell route snapshot %s is corrupt: %w", routePath, err)
		}
		if cfg.Design == "" {
			cfg.Design = "design"
		}
		if i == 0 {
			recovered = cfg
			continue
		}
		if !sameConfig(recovered, cfg) {
			return Config{}, false, fmt.Errorf("unfinished project obligation has conflicting pre-event routing snapshots; retry the originating sessions or restore one operator configuration")
		}
	}
	return recovered, true, nil
}

type stateLock struct{ releaseFn func() error }

func acquireHookFileLock(scope, kind string) (*filelock.Lock, error) {
	ctx, cancel := context.WithTimeout(context.Background(), hookStateLockWaitLimit)
	defer cancel()
	return acquireHookFileLockContext(ctx, scope, kind)
}

func acquireHookFileLockContext(ctx context.Context, scope, kind string) (*filelock.Lock, error) {
	lock, err := filelock.AcquireWaitContext(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("%s remained locked within %s: %w", kind, hookStateLockWaitLimit, err)
	}
	return lock, nil
}

func acquireStateLock(stateFile string) (*stateLock, error) {
	lock, err := acquireHookFileLock(stateFile+".session", "hook state for this session")
	if err != nil {
		return nil, err
	}
	return &stateLock{releaseFn: lock.Release}, nil
}

func acquireStateLockContext(ctx context.Context, stateFile string) (*stateLock, error) {
	lock, err := acquireHookFileLockContext(ctx, stateFile+".session", "hook state for this session")
	if err != nil {
		return nil, err
	}
	return &stateLock{releaseFn: lock.Release}, nil
}

func (l *stateLock) release() error { return l.releaseFn() }

var (
	acquireHookStateLock    = acquireStateLock
	replaceHookStateFile    = replaceStateFile
	hookStatePhase          = func(string) {}
	hookRoutePhase          = func(string) {}
	hookStateMarkerPhase    = func(string) {}
	hookStateReadPhase      = func(string, string) {}
	hookStateReplacePhase   = func(string, string) {}
	hookStateDirectoryPhase = func(string, string) {}
	hookStateBindingPhase   = func(string, string) {}
)

func replaceStateFileAtomic(temp, target string) (retErr error) {
	tempAbs, err := filepath.Abs(temp)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if filepath.Dir(tempAbs) != filepath.Dir(targetAbs) {
		return fmt.Errorf("hook state replacement source and target must share one directory authority")
	}
	parent := filepath.Dir(targetAbs)
	root, err := os.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	tempName, targetName := filepath.Base(tempAbs), filepath.Base(targetAbs)
	if err := recoverHookReplacementQuarantines(root, parent, targetName, hookStateMaxBytes); err != nil {
		return err
	}
	newWitness, err := readBoundedHookRootFile(root, tempName, tempAbs, "hook state replacement", hookStateMaxBytes)
	if err != nil {
		return err
	}
	oldWitness, err := readBoundedHookRootFile(root, targetName, targetAbs, "hook state replacement target", hookStateMaxBytes)
	if errors.Is(err, os.ErrNotExist) {
		hookStateReplacePhase("before-create", targetAbs)
		if err := fsatomic.RenameNoReplaceBetween(root, tempName, root, targetName); err != nil {
			return err
		}
		if err := syncStateDirectory(parent); err != nil {
			return err
		}
		published, err := readBoundedHookRootFile(root, targetName, targetAbs, "hook state replacement target", hookStateMaxBytes)
		if err != nil || !sameHookFileAfterRename(newWitness, published) {
			return errors.Join(err, fmt.Errorf("hook state replacement target %s changed at creation boundary", targetAbs))
		}
		return nil
	}
	if err != nil {
		return err
	}
	hookStateReplacePhase("before-quarantine", targetAbs)
	quarantined, err := fsatomic.Quarantine(root, targetName, "r")
	if err != nil {
		return err
	}
	isolated, err := readBoundedHookRootFile(quarantined.Root(), quarantined.Name(), targetAbs, "hook state replacement target", hookStateMaxBytes)
	if err != nil || !sameHookFileAfterRename(oldWitness, isolated) {
		cause := errors.Join(err, fmt.Errorf("hook state replacement target %s changed while entering private authority", targetAbs))
		return errors.Join(cause, restoreHookReplacementQuarantine(quarantined), quarantined.Close())
	}
	if err := writeHookReplacementWitness(quarantined.Root(), newWitness); err != nil {
		return errors.Join(err, restoreHookReplacementQuarantine(quarantined), quarantined.Close())
	}
	hookStateReplacePhase("after-quarantine", targetAbs)
	if err := fsatomic.RenameNoReplaceBetween(root, tempName, root, targetName); err != nil {
		return errors.Join(err, restoreHookReplacementQuarantine(quarantined), quarantined.Close())
	}
	if err := syncStateDirectory(parent); err != nil {
		return errors.Join(err, quarantined.Close())
	}
	published, err := readBoundedHookRootFile(root, targetName, targetAbs, "hook state replacement target", hookStateMaxBytes)
	if err != nil || !sameHookFileAfterRename(newWitness, published) {
		return errors.Join(err, fmt.Errorf("hook state replacement target %s changed after publication; preserving private prior state", targetAbs), quarantined.Close())
	}
	if err := finishHookReplacementQuarantine(quarantined, true); err != nil {
		return errors.Join(err, quarantined.Close())
	}
	published, err = readBoundedHookRootFile(root, targetName, targetAbs, "hook state replacement target", hookStateMaxBytes)
	if err != nil || !sameHookFileAfterRename(newWitness, published) {
		return errors.Join(err, fmt.Errorf("hook state replacement target %s changed while retiring prior state", targetAbs))
	}
	return syncStateDirectory(parent)
}

func recoverHookReplacementQuarantines(root *os.Root, parent, target string, limit int64) error {
	entries, err := dirscan.Read(parent, hookStateDirMaxEntries)
	if err != nil {
		return err
	}
	const prefix = "r"
	matches := make([]string, 0, 1)
	for _, entry := range entries {
		name := entry.Name()
		if !looksLikeHookQuarantine(name, prefix) {
			continue
		}
		q, err := fsatomic.ResumeQuarantine(root, name, "")
		if err != nil {
			return err
		}
		if q.Source() != target {
			if err := q.Close(); err != nil {
				return err
			}
			continue
		}
		matches = append(matches, name)
		if err := q.Close(); err != nil {
			return err
		}
	}
	if len(matches) > 1 {
		return fmt.Errorf("multiple interrupted hook state replacements exist for %s", filepath.Join(parent, target))
	}
	if len(matches) == 0 {
		return nil
	}
	q, err := fsatomic.ResumeQuarantine(root, matches[0], target)
	if err != nil {
		return err
	}
	name := matches[0]
	_, objectErr := q.Root().Lstat(q.Name())
	newWitness, newErr := readBoundedHookRootFile(q.Root(), "new", filepath.Join(parent, name, "new"), "hook state replacement witness", limit)
	if errors.Is(newErr, os.ErrNotExist) {
		newWitness, newErr = nil, nil
	}
	if newErr != nil {
		return errors.Join(newErr, q.Close())
	}
	public, publicErr := readBoundedHookRootFile(root, target, filepath.Join(parent, target), "hook state replacement target", limit)
	if errors.Is(publicErr, os.ErrNotExist) {
		if errors.Is(objectErr, os.ErrNotExist) {
			return errors.Join(fmt.Errorf("interrupted hook state replacement lost both public and private state for %s", target), q.Close())
		}
		if objectErr != nil {
			return errors.Join(objectErr, q.Close())
		}
		if newWitness != nil {
			if err := q.Root().Remove("new"); err != nil {
				return errors.Join(err, q.Close())
			}
			if err := syncHookRoot(q.Root()); err != nil {
				return errors.Join(err, q.Close())
			}
		}
		if err := q.Restore(); err != nil {
			return errors.Join(err, q.Close())
		}
		return nil
	}
	if publicErr != nil {
		return errors.Join(publicErr, q.Close())
	}
	if newWitness == nil {
		if errors.Is(objectErr, os.ErrNotExist) {
			if err := q.FinishEmpty(); err != nil {
				return errors.Join(err, q.Close())
			}
			return nil
		}
		return errors.Join(fmt.Errorf("interrupted hook state replacement of %s has no exact new-state witness; preserving both authorities", target), q.Close())
	}
	if !sameHookFileContent(newWitness, public) {
		return errors.Join(fmt.Errorf("public hook state changed during interrupted replacement of %s; preserving both authorities", target), q.Close())
	}
	if objectErr != nil {
		if !errors.Is(objectErr, os.ErrNotExist) {
			return errors.Join(objectErr, q.Close())
		}
	}
	if err := finishHookReplacementQuarantine(q, !errors.Is(objectErr, os.ErrNotExist)); err != nil {
		return errors.Join(err, q.Close())
	}
	return nil
}

func looksLikeHookQuarantine(name, prefix string) bool {
	const marker = ".fsatomic."
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	rest := strings.TrimPrefix(name, prefix)
	return len(rest) > 32+len(marker) && validLowerHex(rest[:32]) && strings.HasPrefix(rest[32:], marker)
}

func writeHookReplacementWitness(root *os.Root, witness *hookFileWitness) error {
	if witness == nil {
		return fmt.Errorf("hook state replacement has no new-state witness")
	}
	file, err := root.OpenFile("new", os.O_WRONLY|os.O_CREATE|os.O_EXCL, witness.mode.Perm())
	if err != nil {
		return err
	}
	written, writeErr := file.Write(witness.body)
	if writeErr == nil && written != len(witness.body) {
		writeErr = io.ErrShortWrite
	}
	if err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return err
	}
	return syncHookRoot(root)
}

func finishHookReplacementQuarantine(q *fsatomic.Quarantined, objectExists bool) error {
	if objectExists {
		if err := q.Root().Remove(q.Name()); err != nil {
			return err
		}
		if err := syncHookRoot(q.Root()); err != nil {
			return err
		}
	}
	if err := q.Root().Remove("new"); err != nil {
		return err
	}
	if err := syncHookRoot(q.Root()); err != nil {
		return err
	}
	return q.FinishEmpty()
}

func restoreHookReplacementQuarantine(q *fsatomic.Quarantined) error {
	if err := q.Root().Remove("new"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := syncHookRoot(q.Root()); err != nil {
		return err
	}
	return q.Restore()
}

func syncHookRoot(root *os.Root) error {
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

func hookStateTemps(stateFile string) ([]string, error) {
	prefix := "." + filepath.Base(stateFile) + ".tmp-"
	return boundedHookStateMatches(filepath.Dir(stateFile), "hook state temp", func(name string) (bool, error) {
		if !strings.HasPrefix(name, prefix) {
			return false, nil
		}
		if strings.TrimPrefix(name, prefix) == "" {
			return false, fmt.Errorf("hook state temp %q has a noncanonical filename", name)
		}
		return true, nil
	})
}

func hookProjectBase(stateFile string) (string, error) {
	base := filepath.Base(stateFile)
	base = strings.TrimPrefix(base, ".")
	index := strings.Index(base, ".state")
	if index != sha256.Size*2 || !validHookHexDigest(base[:index]) {
		return "", fmt.Errorf("hook state path %s has a noncanonical project identity", stateFile)
	}
	return base[:index+len(".state")], nil
}

func hookDeletionQuarantinePrefix(stateFile string) (string, error) {
	if _, err := hookProjectBase(stateFile); err != nil {
		return "", err
	}
	return "d", nil
}

func hookProjectQuarantinePaths(stateFile string) ([]string, error) {
	base, err := hookProjectBase(stateFile)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(stateFile)
	entries, err := dirscan.Read(dir, hookStateDirMaxEntries)
	if err != nil {
		return nil, fmt.Errorf("inventory hook state retirement directories: %w", err)
	}
	paths := make([]string, 0)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	for _, entry := range entries {
		name := entry.Name()
		if !looksLikeHookQuarantine(name, "d") && !looksLikeHookQuarantine(name, "r") {
			continue
		}
		info, err := os.Lstat(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, fmt.Errorf("hook state retirement %s must be a real directory", filepath.Join(dir, name))
		}
		q, err := fsatomic.ResumeQuarantine(root, name, "")
		if err != nil {
			return nil, err
		}
		related := hookProjectSourceMatches(filepath.Join(dir, base), q.Source())
		if err := q.Close(); err != nil {
			return nil, err
		}
		if !related {
			continue
		}
		paths = append(paths, filepath.Join(dir, name))
	}
	return paths, nil
}

func recoverHookDeletionQuarantines(stateFile string) (retErr error) {
	prefix, err := hookDeletionQuarantinePrefix(stateFile)
	if err != nil {
		return err
	}
	dir := filepath.Dir(stateFile)
	entries, err := dirscan.Read(dir, hookStateDirMaxEntries)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	for _, entry := range entries {
		name := entry.Name()
		if !looksLikeHookQuarantine(name, prefix) {
			continue
		}
		q, err := fsatomic.ResumeQuarantine(root, name, "")
		if err != nil {
			return err
		}
		if !hookProjectSourceMatches(stateFile, q.Source()) {
			if err := q.Close(); err != nil {
				return err
			}
			continue
		}
		if _, err := q.Root().Lstat(q.Name()); errors.Is(err, os.ErrNotExist) {
			if err := q.FinishEmpty(); err != nil {
				return errors.Join(err, q.Close())
			}
			continue
		} else if err != nil {
			return errors.Join(err, q.Close())
		}
		if err := q.Restore(); err != nil {
			return errors.Join(fmt.Errorf("restore interrupted hook state deletion %q: %w", name, err), q.Close())
		}
	}
	return nil
}

func validLowerHex(value string) bool {
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return value != ""
}

func hookProjectSourceMatches(stateFile, source string) bool {
	base, err := hookProjectBase(stateFile)
	if err != nil {
		return false
	}
	if source == base || strings.HasPrefix(source, "."+base+".tmp-") {
		return true
	}
	routePrefix := base + ".route-"
	if strings.HasPrefix(source, routePrefix) && strings.HasSuffix(source, ".json") {
		return validHookHexDigest(strings.TrimSuffix(strings.TrimPrefix(source, routePrefix), ".json"))
	}
	routeTempPrefix := "." + routePrefix
	if strings.HasPrefix(source, routeTempPrefix) {
		rest := strings.TrimPrefix(source, routeTempPrefix)
		index := strings.Index(rest, ".json.tmp-")
		return index == sha256.Size*2 && validHookHexDigest(rest[:index]) && rest[index+len(".json.tmp-"):] != ""
	}
	return false
}

func cleanupHookStateTemps(stateFile string) error {
	if err := recoverHookDeletionQuarantines(stateFile); err != nil {
		return err
	}
	matches, err := hookStateTemps(stateFile)
	if err != nil {
		return err
	}
	removed := false
	for _, temp := range matches {
		witness, err := readBoundedHookStateFile(temp, "hook state temp", hookStateMaxBytes)
		if err != nil {
			return err
		}
		if witness == nil {
			return fmt.Errorf("hook state temp %s disappeared during cleanup", temp)
		}
		if err := removeHookFileWitness(temp, "hook state temp", hookStateMaxBytes, witness); err != nil {
			return err
		}
		removed = true
	}
	if removed {
		return syncStateDirectory(filepath.Dir(stateFile))
	}
	return nil
}

// appendState records kind ("design" or "impl"). Failure is blocking: losing
// this write would make the later Stop event incorrectly believe no governed
// file changed and silently skip its checks.
func appendState(root, sessionID, kind string) (returnErr error) {
	if kind != "design" && kind != "impl" {
		return fmt.Errorf("unsupported hook state kind %q", kind)
	}
	if err := ensureStateDir(); err != nil {
		return err
	}
	p := statePath(root, sessionID)
	lock, err := acquireHookStateLock(p)
	if err != nil {
		return err
	}
	defer func() {
		returnErr = errors.Join(returnErr, lock.release())
	}()
	return updateStateLocked(p, kind == "design", kind == "impl", "", "", "")
}

type hookStateRecord struct {
	revision uint64
	design   bool
	impl     bool
	pending  []string
	routes   []string
}

func updateStateLocked(p string, addDesign, addImpl bool, addPending, removePending, addRoute string) error {
	temps, err := hookStateTemps(p)
	if err != nil {
		return err
	}
	if len(temps) > 0 {
		return fmt.Errorf("incomplete hook state transaction: durable temp %s exists; refusing to overwrite crash evidence", temps[0])
	}
	raw, err := readStateFile(p)
	if err != nil {
		return err
	}
	record, err := parseHookStateRecord(raw)
	if err != nil {
		return err
	}
	if record.revision == ^uint64(0) {
		return fmt.Errorf("hook state revision overflow")
	}
	record.revision++
	record.design = record.design || addDesign
	record.impl = record.impl || addImpl
	pending := make(map[string]bool, len(record.pending)+1)
	for _, token := range record.pending {
		pending[token] = true
	}
	if addPending != "" {
		pending[addPending] = true
	}
	if removePending != "" {
		delete(pending, removePending)
	}
	record.pending = record.pending[:0]
	for token := range pending {
		record.pending = append(record.pending, token)
	}
	sort.Strings(record.pending)
	if addRoute != "" {
		found := false
		for _, digest := range record.routes {
			found = found || digest == addRoute
		}
		if !found {
			record.routes = append(record.routes, addRoute)
			sort.Strings(record.routes)
		}
	}
	var body strings.Builder
	fmt.Fprintf(&body, "revision %d\n", record.revision)
	if record.design {
		body.WriteString("design\n")
	}
	if record.impl {
		body.WriteString("impl\n")
	}
	for _, digest := range record.routes {
		body.WriteString("route " + digest + "\n")
	}
	for _, token := range record.pending {
		body.WriteString("pending " + token + "\n")
	}
	return writeHookStateBody(p, []byte(body.String()), true)
}

var hookStateIdentityRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

func parseHookStateRecord(raw []byte) (hookStateRecord, error) {
	if raw == nil {
		return hookStateRecord{}, nil
	}
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(lines) < 2 || !strings.HasSuffix(string(raw), "\n") || !strings.HasPrefix(lines[0], "revision ") {
		return hookStateRecord{}, fmt.Errorf("hook state ledger is corrupt or noncanonical: expected revision <positive integer> followed by design and/or impl")
	}
	revision, err := strconv.ParseUint(strings.TrimPrefix(lines[0], "revision "), 10, 64)
	if err != nil || revision == 0 {
		return hookStateRecord{}, fmt.Errorf("hook state ledger is corrupt or noncanonical: revision must be a positive canonical integer")
	}
	if strconv.FormatUint(revision, 10) != strings.TrimPrefix(lines[0], "revision ") {
		return hookStateRecord{}, fmt.Errorf("hook state ledger is corrupt or noncanonical: revision must be a positive canonical integer")
	}
	record := hookStateRecord{revision: revision}
	classIndex := 0
	routeStarted := false
	pendingStarted := false
	for _, line := range lines[1:] {
		switch {
		case !routeStarted && !pendingStarted && classIndex == 0 && line == "design":
			record.design = true
			classIndex++
		case !routeStarted && !pendingStarted && (classIndex == 0 || classIndex == 1) && line == "impl" && !record.impl:
			record.impl = true
			classIndex = 2
		case !pendingStarted && strings.HasPrefix(line, "route "):
			routeStarted = true
			digest := strings.TrimPrefix(line, "route ")
			if !hookStateIdentityRe.MatchString(digest) {
				return hookStateRecord{}, fmt.Errorf("hook state ledger is corrupt or noncanonical: route identity must be 64 lowercase hex characters")
			}
			if len(record.routes) > 0 && digest <= record.routes[len(record.routes)-1] {
				return hookStateRecord{}, fmt.Errorf("hook state ledger is corrupt or noncanonical: route identities must be unique and sorted")
			}
			record.routes = append(record.routes, digest)
		case strings.HasPrefix(line, "pending "):
			pendingStarted = true
			token := strings.TrimPrefix(line, "pending ")
			if !hookStateIdentityRe.MatchString(token) {
				return hookStateRecord{}, fmt.Errorf("hook state ledger is corrupt or noncanonical: pending operation identity must be 64 lowercase hex characters")
			}
			if len(record.pending) > 0 && token <= record.pending[len(record.pending)-1] {
				return hookStateRecord{}, fmt.Errorf("hook state ledger is corrupt or noncanonical: pending operations must be unique and sorted")
			}
			record.pending = append(record.pending, token)
		default:
			return hookStateRecord{}, fmt.Errorf("hook state ledger is corrupt or noncanonical: touch classes must be exactly design, impl, or design then impl")
		}
	}
	if !record.design && !record.impl {
		return hookStateRecord{}, fmt.Errorf("hook state ledger is corrupt or noncanonical: at least one touch class is required")
	}
	return record, nil
}

func readStateErr(root, sessionID string) (design, impl bool, err error) {
	record, err := readStateRecord(root, sessionID)
	return record.design, record.impl, err
}

func readStateRecord(root, sessionID string) (record hookStateRecord, returnErr error) {
	if err := requireStateDir(); err != nil {
		return hookStateRecord{}, err
	}
	p := statePath(root, sessionID)
	lock, err := acquireHookStateLock(p)
	if err != nil {
		return hookStateRecord{}, err
	}
	defer func() { returnErr = errors.Join(returnErr, lock.release()) }()
	temps, err := hookStateTemps(p)
	if err != nil {
		return hookStateRecord{}, err
	}
	if len(temps) > 0 {
		return hookStateRecord{}, fmt.Errorf("incomplete hook state transaction for this project: durable temp %s exists; refusing to treat it as untouched", temps[0])
	}
	if routeTemps, err := routeStateTemps(root); err != nil {
		return hookStateRecord{}, err
	} else if len(routeTemps) > 0 {
		return hookStateRecord{}, fmt.Errorf("incomplete hook route transaction: durable temp %s exists", routeTemps[0])
	}
	raw, err := readStateFile(p)
	if err != nil {
		return hookStateRecord{}, err
	}
	if raw == nil {
		routes, err := routeStatePaths(root)
		if err != nil {
			return hookStateRecord{}, err
		}
		if len(routes) > 0 {
			return hookStateRecord{}, fmt.Errorf("incomplete hook state transaction: %d route snapshot(s) exist without the project dirty ledger", len(routes))
		}
		return hookStateRecord{}, nil
	}
	return parseHookStateRecord(raw)
}

func readState(root, sessionID string) (design, impl bool) {
	design, impl, _ = readStateErr(root, sessionID)
	return design, impl
}

func clearState(root, sessionID string) (returnErr error) {
	_, returnErr = clearStateRevision(root, sessionID, 0)
	return returnErr
}

func clearCheckedState(root, sessionID string, revision uint64) error {
	cleared, err := clearStateRevision(root, sessionID, revision)
	if err != nil {
		return err
	}
	if !cleared {
		return fmt.Errorf("new governed edits arrived while the gates were running; the newer project obligation is retained and must be checked again")
	}
	return nil
}

func clearStateRevision(root, sessionID string, expectedRevision uint64) (cleared bool, returnErr error) {
	p := statePath(root, sessionID)
	if err := requireStateDir(); err != nil {
		return false, err
	}
	lock, err := acquireHookStateLock(p)
	if err != nil {
		return false, err
	}
	defer func() {
		returnErr = errors.Join(returnErr, lock.release())
	}()
	if err := recoverHookDeletionQuarantines(p); err != nil {
		return false, err
	}
	stateWitness, err := readBoundedHookStateFile(p, "hook state", hookStateMaxBytes)
	if err != nil {
		return false, err
	}
	var raw []byte
	if stateWitness != nil {
		raw = stateWitness.body
	}
	if temps, err := routeStateTemps(root); err != nil {
		return false, err
	} else if len(temps) > 0 {
		return false, fmt.Errorf("incomplete hook route transaction: durable temp %s exists", temps[0])
	}
	if raw != nil && expectedRevision != 0 {
		record, err := parseHookStateRecord(raw)
		if err != nil {
			return false, err
		}
		if record.revision != expectedRevision {
			return false, nil
		}
	}
	routes, err := routeStatePaths(root)
	if err != nil {
		return false, err
	}
	routeWitnesses := make([]*hookFileWitness, len(routes))
	for i, route := range routes {
		witness, err := readBoundedHookStateFile(route, "hook route snapshot", hookRouteMaxBytes)
		if err != nil {
			return false, err
		}
		if witness == nil {
			return false, fmt.Errorf("hook route snapshot %s disappeared during clear", route)
		}
		routeWitnesses[i] = witness
	}
	removed := false
	// Routes go first and the dirty ledger goes last. Any mismatch therefore
	// retains the project obligation rather than discharging it partially.
	for index, route := range routes {
		if err := removeHookFileWitness(route, "hook route snapshot", hookRouteMaxBytes, routeWitnesses[index]); err != nil {
			return false, err
		}
		removed = true
	}
	remainingRoutes, err := routeStatePaths(root)
	if err != nil {
		return false, err
	}
	if len(remainingRoutes) != 0 {
		return false, fmt.Errorf("hook route snapshot inventory changed during clear; preserving dirty ledger")
	}
	if stateWitness != nil {
		if err := removeHookFileWitness(p, "hook state", hookStateMaxBytes, stateWitness); err != nil {
			return false, err
		}
		removed = true
	}
	if !removed {
		return true, nil
	}
	return true, syncStateDirectory(filepath.Dir(p))
}

func removeHookFileWitness(path, kind string, limit int64, want *hookFileWitness) (retErr error) {
	prefix, err := hookDeletionQuarantinePrefix(path)
	if err != nil {
		return err
	}
	return removeHookFileWitnessWithPrefix(path, kind, limit, want, prefix)
}

func removeHookFileWitnessWithPrefix(path, kind string, limit int64, want *hookFileWitness, prefix string) (retErr error) {
	if want == nil {
		return fmt.Errorf("%s %s has no deletion witness", kind, path)
	}
	for pass := 1; pass <= 2; pass++ {
		got, err := readBoundedHookFile(path, kind, limit)
		if err != nil {
			return err
		}
		if !sameHookFileWitness(want, got) {
			return fmt.Errorf("%s %s changed before deletion; preserving it", kind, path)
		}
		if pass == 1 {
			hookStateReadPhase("before-remove", path)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	parent, name := filepath.Dir(abs), filepath.Base(abs)
	root, err := os.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	quarantined, err := fsatomic.Quarantine(root, name, prefix)
	if err != nil {
		return err
	}
	hookStateReadPhase("after-quarantine", path)
	isolated, err := readBoundedHookRootFile(quarantined.Root(), quarantined.Name(), path, kind, limit)
	if err != nil || !sameHookFileAfterRename(want, isolated) {
		cause := errors.Join(err, fmt.Errorf("%s %s changed while entering private deletion authority; preserving it", kind, path))
		if restoreErr := quarantined.Restore(); restoreErr != nil {
			return errors.Join(cause, restoreErr, quarantined.Close())
		}
		return cause
	}
	if err := quarantined.Remove(); err != nil {
		return errors.Join(err, quarantined.Close())
	}
	if _, err := root.Lstat(name); err == nil {
		return fmt.Errorf("%s %s was replaced during deletion; preserving the replacement", kind, path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func readBoundedHookRootFile(root *os.Root, name, display, kind string, limit int64) (witness *hookFileWitness, retErr error) {
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("%s %s must be a regular file, not a symlink or special file", kind, display)
	}
	if before.Size() > limit {
		return nil, fmt.Errorf("%s %s exceeds %d-byte limit", kind, display, limit)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) || before.Mode() != opened.Mode() || before.Size() != opened.Size() || !before.ModTime().Equal(opened.ModTime()) {
		return nil, errors.Join(err, fmt.Errorf("%s %s changed while opening private deletion authority", kind, display))
	}
	body, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%s %s exceeds %d-byte limit", kind, display, limit)
	}
	after, statErr := file.Stat()
	pathAfter, pathErr := root.Lstat(name)
	if err := errors.Join(statErr, pathErr); err != nil {
		return nil, err
	}
	beforeChangeID, afterChangeID, pathChangeID := hookFileChangeID(before), hookFileChangeID(after), hookFileChangeID(pathAfter)
	if !os.SameFile(opened, after) || !os.SameFile(opened, pathAfter) || opened.Mode() != after.Mode() || opened.Mode() != pathAfter.Mode() ||
		opened.Size() != after.Size() || opened.Size() != pathAfter.Size() || !opened.ModTime().Equal(after.ModTime()) || !opened.ModTime().Equal(pathAfter.ModTime()) ||
		(beforeChangeID != "" && afterChangeID != "" && beforeChangeID != afterChangeID) || (beforeChangeID != "" && pathChangeID != "" && beforeChangeID != pathChangeID) {
		return nil, fmt.Errorf("%s %s changed inside private deletion authority", kind, display)
	}
	return &hookFileWitness{body: body, info: pathAfter, mode: pathAfter.Mode(), size: pathAfter.Size(), modTime: pathAfter.ModTime(), changeID: pathChangeID}, nil
}

func sameHookFileWitness(want, got *hookFileWitness) bool {
	return want != nil && got != nil && want.info != nil && got.info != nil && os.SameFile(want.info, got.info) &&
		want.mode == got.mode && want.size == got.size && want.modTime.Equal(got.modTime) && bytes.Equal(want.body, got.body) &&
		(want.changeID == "" || got.changeID == "" || want.changeID == got.changeID)
}

// A rename updates change time on several platforms while preserving the
// file object. Isolation comparisons therefore bind native identity, bytes,
// mode, size, and modification time, but deliberately exclude change time.
func sameHookFileAfterRename(want, got *hookFileWitness) bool {
	return want != nil && got != nil && want.info != nil && got.info != nil && os.SameFile(want.info, got.info) &&
		want.mode == got.mode && want.size == got.size && want.modTime.Equal(got.modTime) && bytes.Equal(want.body, got.body)
}

func sameHookFileContent(want, got *hookFileWitness) bool {
	return want != nil && got != nil && want.mode == got.mode && want.size == got.size && bytes.Equal(want.body, got.body)
}

// --- small helpers ---

var patchPathLine = regexp.MustCompile(`(?m)^\*\*\* (Add|Update|Delete) File: (.+)$`)
var patchMoveLine = regexp.MustCompile(`(?m)^\*\*\* Move to: (.+)$`)
var patchDeleteLine = regexp.MustCompile(`(?m)^\*\*\* Delete File: (.+)$`)

// deletedPaths returns the paths an apply_patch-style tool input deletes.
// Claude's file tools (Write/Edit) never delete, so only the patch protocols
// contribute here.
func deletedPaths(in Input) []string {
	patchText := in.ToolInput.Command
	if patchText == "" {
		patchText = in.ToolInput.Patch
	}
	if patchText == "" {
		return nil
	}
	var paths []string
	for _, match := range patchDeleteLine.FindAllStringSubmatch(patchText, -1) {
		if p := strings.TrimSpace(match[1]); p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}

// patchOp is the kind of edit a single tool input performs on a single path.
// A path can carry more than one op in one apply_patch (a delete plus an add
// of the same file), which is why the deny rules key on the op and not on the
// set of paths the patch happens to touch.
type patchOp string

const (
	opWrite  patchOp = "write"
	opAdd    patchOp = "add"
	opUpdate patchOp = "update"
	opDelete patchOp = "delete"
	opMove   patchOp = "move"
)

// editedPath is one path together with the operation that reached it.
type editedPath struct {
	Path string
	Op   patchOp
}

var patchOpByKeyword = map[string]patchOp{"Add": opAdd, "Update": opUpdate, "Delete": opDelete}

// editedOps normalizes the two supported hook protocols into per-operation
// entries. Claude file tools provide file_path/notebook_path and only ever
// write. Codex apply_patch provides the complete patch in tool_input.command;
// the OpenCode adapter may use command or patch. Entries are deduplicated by
// path AND op, so a patch that deletes and then re-adds the same file yields
// both operations in the order the patch states them.
func editedOps(in Input) []editedPath {
	if in.ToolInput.FilePath != "" {
		return []editedPath{{Path: in.ToolInput.FilePath, Op: opWrite}}
	}
	if in.ToolInput.NotebookPath != "" {
		return []editedPath{{Path: in.ToolInput.NotebookPath, Op: opWrite}}
	}
	patchText := in.ToolInput.Command
	if patchText == "" {
		patchText = in.ToolInput.Patch
	}
	if patchText == "" {
		return nil
	}

	seen := map[editedPath]bool{}
	var edits []editedPath
	add := func(p string, op patchOp) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		e := editedPath{Path: p, Op: op}
		if seen[e] {
			return
		}
		seen[e] = true
		edits = append(edits, e)
	}
	for _, match := range patchPathLine.FindAllStringSubmatch(patchText, -1) {
		add(match[2], patchOpByKeyword[match[1]])
	}
	for _, match := range patchMoveLine.FindAllStringSubmatch(patchText, -1) {
		add(match[1], opMove)
	}
	return edits
}

// editedPaths is the path-only view of editedOps, deduplicated by path and
// preserving the order the operations appear in. Callers that must reason
// about what an operation means (a delete is not a create) use editedOps.
func editedPaths(in Input) []string {
	seen := map[string]bool{}
	var paths []string
	for _, e := range editedOps(in) {
		if seen[e.Path] {
			continue
		}
		seen[e.Path] = true
		paths = append(paths, e.Path)
	}
	return paths
}

// designRel returns the configured design directory as a clean, slash-
// separated, root-relative path; a value that escapes the root falls back
// to the default rather than widening the read-only net to the whole repo.
func designRel(cfg Config) string {
	d := path.Clean(filepath.ToSlash(cfg.Design))
	if d == "" || d == "." || d == ".." || strings.HasPrefix(d, "../") || path.IsAbs(d) {
		return "design"
	}
	return d
}

// relToRoot returns p relative to root with forward slashes, or "" when p
// lies outside root or cannot be resolved.
func relToRoot(root, p string) string {
	if p == "" {
		return ""
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return ""
	}
	absPath, err := filepath.Abs(p)
	if err != nil {
		return ""
	}
	if resolved, rerr := filepath.EvalSymlinks(absRoot); rerr == nil {
		absRoot = resolved
	}
	if resolved, rerr := evalPathWithMissingTail(absPath); rerr == nil {
		absPath = resolved
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return filepath.ToSlash(rel)
}

// resolveEventPath interprets relative hook paths from the event's own cwd,
// not the hook process cwd. Hosts commonly launch hooks from a plugin cache.
func resolveEventPath(cwd, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	base := cwd
	if base == "" {
		base = "."
	}
	return filepath.Clean(filepath.Join(base, p))
}

// evalPathWithMissingTail resolves symlink aliases in the longest existing
// prefix while preserving a not-yet-created file suffix.
func evalPathWithMissingTail(p string) (string, error) {
	cur := filepath.Clean(p)
	var tail []string
	for {
		resolved, err := filepath.EvalSymlinks(cur)
		if err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return resolved, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", err
		}
		tail = append(tail, filepath.Base(cur))
		cur = parent
	}
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func capString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n(... gate output truncated)"
}

func emitJSON(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}
