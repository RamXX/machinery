// Package hook handles the Claude Code plugin hook events behind
// `machinery hook`: it reads one hook event as JSON on stdin and answers on
// stdout per the Claude Code hook contract (deny/block/context as JSON, exit
// code always 0 on a handled event).
//
// Every event is a strict no-op unless the project is machinery-managed: a
// .machinery.json at the project root, or the conventional
// design/domain.modelith.yaml. The plugin's shell shim performs the same
// detection before invoking the binary, so a non-machinery repo never pays
// more than two stat calls and never sees output from these hooks.
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
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RamXX/machinery/internal/gates"
	"github.com/RamXX/machinery/internal/pack"
)

// ConfigName is the project-root marker and configuration file.
const ConfigName = ".machinery.json"

// Input mirrors the Claude Code hook stdin JSON (only the fields used here).
type Input struct {
	SessionID      string    `json:"session_id"`
	Cwd            string    `json:"cwd"`
	HookEventName  string    `json:"hook_event_name"`
	ToolName       string    `json:"tool_name"`
	ToolInput      toolInput `json:"tool_input"`
	StopHookActive bool      `json:"stop_hook_active"`
}

type toolInput struct {
	FilePath     string `json:"file_path"`
	NotebookPath string `json:"notebook_path"`
}

// Config is the .machinery.json shape. Every field is optional; an absent
// file falls back to convention (design/ with domain.modelith.yaml).
type Config struct {
	Design string `json:"design"` // design directory relative to the root (default "design")
	Gates  string `json:"gates"`  // staged --gate list; empty selects by which artifacts exist
	Impl   string `json:"impl"`   // implementation dir for G4-import; empty disables it
	Hooks  *bool  `json:"hooks"`  // explicit opt-out: {"hooks": false}
	Strict bool   `json:"strict"` // block the stop on any blocking finding, not only DRIFT
}

// Load resolves the machinery hook configuration for root. ok is false when
// the project is not machinery-managed (every hook no-ops). A present but
// unparseable config still counts as managed, with defaults plus a warning,
// so a typo degrades loudly instead of silently disabling governance.
func Load(root string) (cfg Config, ok bool, warn string) {
	cfg = Config{Design: "design"}
	raw, err := os.ReadFile(filepath.Join(root, ConfigName))
	if err != nil {
		if _, serr := os.Stat(filepath.Join(root, "design", "domain.modelith.yaml")); serr == nil {
			return cfg, true, ""
		}
		return cfg, false, ""
	}
	if jerr := json.Unmarshal(raw, &cfg); jerr != nil {
		cfg = Config{Design: "design"}
		return cfg, true, fmt.Sprintf("machinery: %s does not parse (%v); governance runs with defaults", ConfigName, jerr)
	}
	if cfg.Hooks != nil && !*cfg.Hooks {
		return cfg, false, ""
	}
	if cfg.Design == "" {
		cfg.Design = "design"
	}
	if cfg.Gates != "" {
		for _, tok := range strings.Split(strings.ToLower(cfg.Gates), ",") {
			t := strings.TrimSpace(tok)
			if t != "g2" && t != "g3" && t != "gx" && t != "g4" && t != "g5" {
				warn = fmt.Sprintf("machinery: %s gates list has unknown gate %q; selecting gates automatically", ConfigName, t)
				cfg.Gates = ""
				break
			}
		}
	}
	return cfg, true, warn
}

// Run dispatches one hook event read from r and writes the answer to w.
// root overrides project-root resolution (flag > $CLAUDE_PROJECT_DIR > the
// event's cwd). A nil return with no output means "nothing to say": the
// event was either not machinery's business or clean.
func Run(r io.Reader, w io.Writer, root string) error {
	var in Input
	if err := json.NewDecoder(r).Decode(&in); err != nil {
		return fmt.Errorf("machinery hook: stdin is not hook-event JSON: %w", err)
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
	cfg, ok, warn := Load(root)
	if !ok {
		return nil
	}
	switch in.HookEventName {
	case "PreToolUse":
		return pre(w, root, cfg, in)
	case "PostToolUse":
		return post(root, cfg, in)
	case "Stop", "SubagentStop":
		return stop(w, root, cfg, in, warn)
	case "SessionStart":
		return sessionStart(w, root, cfg, warn)
	}
	return nil
}

// --- PreToolUse: generated artifacts are read-only ---

var fileTools = map[string]bool{
	"Edit":         true,
	"Write":        true,
	"MultiEdit":    true,
	"NotebookEdit": true,
}

type preOut struct {
	HookSpecificOutput preSpecific `json:"hookSpecificOutput"`
}

type preSpecific struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision"`
	PermissionDecisionReason string `json:"permissionDecisionReason"`
}

func pre(w io.Writer, root string, cfg Config, in Input) error {
	if !fileTools[in.ToolName] {
		return nil
	}
	rel := relToRoot(root, editedPath(in))
	if rel == "" {
		return nil
	}
	reason := generatedReason(designRel(cfg), rel)
	if reason == "" {
		return nil
	}
	return emitJSON(w, preOut{HookSpecificOutput: preSpecific{
		HookEventName:            "PreToolUse",
		PermissionDecision:       "deny",
		PermissionDecisionReason: reason,
	}})
}

// generatedReason classifies rel (a root-relative, slash-separated path) as
// a generated design artifact and returns the refusal reason, or "".
func generatedReason(design, rel string) string {
	if rel != design && !strings.HasPrefix(rel, design+"/") {
		return ""
	}
	sub := strings.TrimPrefix(rel, design+"/")
	base := path.Base(sub)
	switch {
	case sub == "ratchet.json":
		return rel + " is generated by 'machinery baseline' from the observed import graph; a hand edit defeats the ratchet. " +
			"Rerun 'machinery baseline " + design + " --impl <dir>' to regenerate it."
	case strings.HasSuffix(base, ".oracle.md"):
		return rel + " is generated by 'machinery oracle'; a hand edit is DRIFT by definition. " +
			"Edit the machine JSON (and its matrix), run 'machinery oracle " + design + "/machines', " +
			"and commit the regenerated oracle."
	case strings.HasPrefix(sub, "formal/") && (strings.HasSuffix(base, ".tla") || strings.HasSuffix(base, ".cfg")):
		return rel + " is generated by 'machinery verify-formal'. Edit the machine JSON or the " +
			"formal/*.yaml annotations, run 'machinery verify-formal " + design + "' " +
			"(--gen-only without Java), and commit the regenerated files."
	case strings.HasPrefix(sub, "packs/"):
		return rel + " is generated by 'machinery pack generate'. Edit the parent design sources " +
			"and regenerate the packs; a boundary change is a parent edit."
	case strings.HasPrefix(sub, "pack/"):
		return rel + " is the frozen pack this child design was built against. It changes only when " +
			"the parent regenerates it; copy the new pack in, never edit it in place."
	}
	return ""
}

// --- PostToolUse: record what the session touched (the stop gates read it) ---

var sourceExt = map[string]bool{
	".go": true, ".py": true,
	".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".mjs": true, ".cjs": true,
	".ex": true, ".exs": true,
	".rs": true,
}

func post(root string, cfg Config, in Input) error {
	if !fileTools[in.ToolName] {
		return nil
	}
	rel := relToRoot(root, editedPath(in))
	if rel == "" {
		return nil
	}
	design := designRel(cfg)
	switch {
	case rel == design || strings.HasPrefix(rel, design+"/"):
		appendState(root, in.SessionID, "design")
	case cfg.Impl != "" && sourceExt[path.Ext(rel)] && underImpl(cfg, rel):
		appendState(root, in.SessionID, "impl")
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

func stop(w io.Writer, root string, cfg Config, in Input, warn string) error {
	touchedDesign, touchedImpl := readState(root, in.SessionID)
	if !touchedDesign && !touchedImpl {
		return nil
	}
	design := designRel(cfg)
	designDir := filepath.Join(root, filepath.FromSlash(design))
	if fi, err := os.Stat(designDir); err != nil || !fi.IsDir() {
		clearState(root, in.SessionID)
		return emitJSON(w, stopOut{SystemMessage: "machinery: project is machinery-managed but the design directory " +
			design + "/ does not exist; gates skipped."})
	}
	sel := selectGates(designDir, cfg)
	if len(sel.Run) == 0 {
		clearState(root, in.SessionID)
		return nil
	}
	implDir := ""
	if cfg.Impl != "" {
		implDir = filepath.Join(root, filepath.FromSlash(cfg.Impl))
	}

	var buf bytes.Buffer
	blocking, drift, g4Blocking := 0, 0, 0
	for _, g := range gates.RunSelected(designDir, implDir, sel) {
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
	armed := fileExists(filepath.Join(designDir, "ratchet.json"))
	shouldBlock := drift > 0 || (g4Blocking > 0 && armed) || (cfg.Strict && blocking > 0)
	switch {
	case shouldBlock && !in.StopHookActive:
		// keep the state so the re-check runs when the fix attempt finishes
		reason := "machinery gates are red for this session's edits.\n\n" + capString(buf.String(), reasonCap) +
			"\nFix the sources and regenerate the derived artifacts " +
			"(machinery oracle | machinery verify-formal --gen-only | machinery pack generate); " +
			"never hand-edit generated files."
		if warn != "" {
			reason = warn + "\n" + reason
		}
		return emitJSON(w, stopOut{Decision: "block", Reason: reason})
	case shouldBlock:
		// already continued once for this stop: surface it, do not loop
		clearState(root, in.SessionID)
		return emitJSON(w, stopOut{SystemMessage: fmt.Sprintf(
			"machinery: gates still red after a fix attempt (%d blocking finding(s), %d DRIFT). "+
				"Stopping anyway; run 'machinery check %s' to review.", blocking, drift, design)})
	case blocking > 0:
		// ERRORs without DRIFT: normal for a design mid-interrogation
		clearState(root, in.SessionID)
		msg := fmt.Sprintf("machinery: %d gate ERROR finding(s) remain (no DRIFT); normal mid-phase. "+
			"'machinery check %s' lists them.", blocking, design)
		if g4Blocking > 0 && !armed {
			msg = fmt.Sprintf("machinery: %d gate ERROR finding(s) remain, %d of them import findings. "+
				"Import blocking is disarmed: %s/ratchet.json does not exist. Complete Stage 1 with "+
				"'machinery baseline %s --impl <dir>' (paste the printed rules, commit the ratchet) to arm enforcement.",
				blocking, g4Blocking, design, design)
		}
		return emitJSON(w, stopOut{SystemMessage: msg})
	default:
		clearState(root, in.SessionID)
		return nil
	}
}

// selectGates picks the suite for a stop-time check: the staged list from
// the config when present, otherwise whichever gates have artifacts to
// check, so a half-built design is never failed for phases not yet reached.
func selectGates(designDir string, cfg Config) gates.Selection {
	if cfg.Gates != "" {
		if sel, err := gates.Select(designDir, cfg.Gates); err == nil {
			if cfg.Impl == "" {
				delete(sel.Run, "g4")
			}
			return sel
		}
	}
	run := map[string]bool{}
	if fileExists(filepath.Join(designDir, "workspace.dsl")) || fileExists(filepath.Join(designDir, "ARCHITECTURE.md")) {
		run["g2"] = true
	}
	if ms, _ := filepath.Glob(filepath.Join(designDir, "machines", "*.machine.json")); len(ms) > 0 {
		run["g3"] = true
	}
	if fileExists(filepath.Join(designDir, "BUILD.md")) {
		run["gx"] = true
	}
	if cfg.Impl != "" {
		run["g4"] = true
	}
	if pack.HasDecomposition(designDir) || pack.HasPack(designDir) {
		run["g5"] = true
	}
	return gates.Selection{Run: run, Explicit: true}
}

// --- SessionStart: announce governance so every session knows the contract ---

func sessionStart(w io.Writer, root string, cfg Config, warn string) error {
	design := designRel(cfg)
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
		if fileExists(filepath.Join(root, filepath.FromSlash(design), "ratchet.json")) {
			state = "baseline recorded, violations block at turn end"
		}
		fmt.Fprintf(&b, "- Import-boundary gate G4 watches source edits under %s (%s).\n", cfg.Impl, state)
	}
	fmt.Fprintf(&b, "- Generated artifacts are read-only and hooks deny edits to them: %s/**/*.oracle.md, %s/formal/*.tla and *.cfg, %s/packs/**, %s/pack/**, %s/ratchet.json. Edit the sources, then regenerate (machinery oracle | machinery verify-formal --gen-only | machinery pack generate | machinery baseline).\n",
		design, design, design, design, design)
	mode := "stale generated artifacts (DRIFT) or import-boundary violations block"
	if cfg.Strict {
		mode = "strict mode: any blocking finding blocks"
	}
	fmt.Fprintf(&b, "- When a turn edits the design or watched sources, 'machinery check' runs before the turn can end; %s.\n", mode)
	b.WriteString("- Design work runs through the 'machinery' skill: four phases, each behind a gate.\n")
	if raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(design), "STATE.md")); err == nil {
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

// statePath keys the touched-files ledger by session and project root under
// the OS temp dir, so parallel sessions and projects never share state.
func statePath(root, sessionID string) string {
	h := fnv.New32a()
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}
	_, _ = h.Write([]byte(absRoot))
	sid := sanitizeID(sessionID)
	if sid == "" {
		sid = "nosession"
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("machinery-hook-%s-%08x", sid, h.Sum32()))
}

var idRe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func sanitizeID(s string) string { return idRe.ReplaceAllString(s, "") }

// appendState records kind ("design" or "impl"). The ledger is advisory: a
// failure to write must never fail the user's tool call, so errors are
// dropped and the worst case is a skipped stop-time check.
func appendState(root, sessionID, kind string) {
	f, err := os.OpenFile(statePath(root, sessionID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	_, _ = f.WriteString(kind + "\n")
	_ = f.Close()
}

func readState(root, sessionID string) (design, impl bool) {
	raw, err := os.ReadFile(statePath(root, sessionID))
	if err != nil {
		return false, false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		switch strings.TrimSpace(line) {
		case "design":
			design = true
		case "impl":
			impl = true
		}
	}
	return design, impl
}

func clearState(root, sessionID string) { _ = os.Remove(statePath(root, sessionID)) }

// --- small helpers ---

func editedPath(in Input) string {
	if in.ToolInput.FilePath != "" {
		return in.ToolInput.FilePath
	}
	return in.ToolInput.NotebookPath
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
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return filepath.ToSlash(rel)
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
