package hook

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- helpers ---

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// managedRoot returns a temp project root marked machinery-managed by
// convention (design/domain.modelith.yaml).
func managedRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "design", "domain.modelith.yaml"), "model: {}\n")
	return root
}

// copyTree copies the go-crm example design into dst for gate-level tests.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// runEvent pipes one synthesized event through Run and returns stdout.
func runEvent(t *testing.T, root string, in Input) string {
	t.Helper()
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(bytes.NewReader(raw), &out, root); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return out.String()
}

func editEvent(event, tool, sessionID, file string) Input {
	return Input{
		SessionID:     sessionID,
		HookEventName: event,
		ToolName:      tool,
		ToolInput:     toolInput{FilePath: file},
	}
}

// --- detection: the no-op guarantee ---

func TestLoadDetection(t *testing.T) {
	t.Run("unmanaged dir is not managed", func(t *testing.T) {
		_, ok, _ := Load(t.TempDir())
		if ok {
			t.Fatal("bare directory must not count as machinery-managed")
		}
	})
	t.Run("conventional design marks managed", func(t *testing.T) {
		root := managedRoot(t)
		cfg, ok, warn := Load(root)
		if !ok || cfg.Design != "design" || warn != "" {
			t.Fatalf("got cfg=%+v ok=%v warn=%q", cfg, ok, warn)
		}
	})
	t.Run("config marks managed with custom design dir", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ConfigName), `{"design":"blueprint","gates":"g2,g3","impl":".","strict":true}`)
		cfg, ok, _ := Load(root)
		if !ok || cfg.Design != "blueprint" || cfg.Gates != "g2,g3" || cfg.Impl != "." || !cfg.Strict {
			t.Fatalf("got cfg=%+v ok=%v", cfg, ok)
		}
	})
	t.Run("hooks false opts out", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ConfigName), `{"hooks": false}`)
		if _, ok, _ := Load(root); ok {
			t.Fatal("hooks:false must disable governance")
		}
	})
	t.Run("unparseable config stays managed and warns", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ConfigName), `{not json`)
		cfg, ok, warn := Load(root)
		if !ok || warn == "" || cfg.Design != "design" {
			t.Fatalf("a config typo must degrade loudly, not disable governance: cfg=%+v ok=%v warn=%q", cfg, ok, warn)
		}
	})
	t.Run("unknown gate in list warns and clears", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ConfigName), `{"gates":"g2,g9"}`)
		cfg, ok, warn := Load(root)
		if !ok || warn == "" || cfg.Gates != "" {
			t.Fatalf("got cfg=%+v ok=%v warn=%q", cfg, ok, warn)
		}
	})
	t.Run("design dir escaping the root falls back to default", func(t *testing.T) {
		for _, d := range []string{"..", "../elsewhere", ".", "/abs"} {
			if got := designRel(Config{Design: d}); got != "design" {
				t.Fatalf("designRel(%q) = %q, want design", d, got)
			}
		}
	})
}

func TestRunIsNoopWhenUnmanaged(t *testing.T) {
	root := t.TempDir()
	for _, ev := range []string{"PreToolUse", "PostToolUse", "Stop", "SubagentStop", "SessionStart"} {
		out := runEvent(t, root, editEvent(ev, "Write", "s1", filepath.Join(root, "design", "machines", "X.oracle.md")))
		if out != "" {
			t.Fatalf("%s in an unmanaged repo must produce no output, got %q", ev, out)
		}
	}
}

// --- PreToolUse: generated artifacts are read-only ---

func TestPreDeniesGeneratedArtifacts(t *testing.T) {
	root := managedRoot(t)
	cases := []struct {
		rel  string
		tool string
		deny bool
		hint string
	}{
		{"design/machines/Deal.oracle.md", "Edit", true, "machinery oracle"},
		{"design/machines/Deal.oracle.md", "Write", true, "machinery oracle"},
		{"design/formal/Deal.tla", "Write", true, "verify-formal"},
		{"design/formal/Deal.cfg", "MultiEdit", true, "verify-formal"},
		{"design/packs/billing.pack/domain.yaml", "Write", true, "pack generate"},
		{"design/pack/contract.machine.json", "Edit", true, "frozen pack"},
		{"design/formal/Deal.semantics.yaml", "Edit", false, ""}, // annotation source
		{"design/machines/Deal.machine.json", "Edit", false, ""}, // machine source
		{"design/machines/Deal.matrix.md", "Edit", false, ""},    // hand matrix
		{"design/domain.modelith.md", "Edit", false, ""},         // rendered, but post-processed by hand
		{"src/main.go", "Write", false, ""},
		{"design/machines/Deal.oracle.md", "Bash", false, ""}, // not a file tool: G3 DRIFT catches it at stop
	}
	for _, c := range cases {
		t.Run(c.tool+" "+c.rel, func(t *testing.T) {
			out := runEvent(t, root, editEvent("PreToolUse", c.tool, "s-pre", filepath.Join(root, c.rel)))
			if !c.deny {
				if out != "" {
					t.Fatalf("expected allow (no output), got %q", out)
				}
				return
			}
			var got preOut
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("deny output is not JSON: %v (%q)", err, out)
			}
			if got.HookSpecificOutput.PermissionDecision != "deny" {
				t.Fatalf("expected deny, got %+v", got)
			}
			if !strings.Contains(got.HookSpecificOutput.PermissionDecisionReason, c.hint) {
				t.Fatalf("reason %q missing hint %q", got.HookSpecificOutput.PermissionDecisionReason, c.hint)
			}
		})
	}
}

func TestPreIgnoresPathsOutsideRoot(t *testing.T) {
	root := managedRoot(t)
	other := t.TempDir()
	out := runEvent(t, root, editEvent("PreToolUse", "Edit", "s-out", filepath.Join(other, "design", "machines", "X.oracle.md")))
	if out != "" {
		t.Fatalf("a path outside the project root is not ours to police, got %q", out)
	}
}

// --- PostToolUse: the touched ledger ---

func TestPostRecordsTouches(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"impl":"."}`)
	sid := "s-post"
	t.Cleanup(func() { clearState(root, sid) })

	runEvent(t, root, editEvent("PostToolUse", "Write", sid, filepath.Join(root, "design", "machines", "Deal.machine.json")))
	if d, i := readState(root, sid); !d || i {
		t.Fatalf("design edit: got design=%v impl=%v", d, i)
	}
	runEvent(t, root, editEvent("PostToolUse", "Edit", sid, filepath.Join(root, "src", "main.go")))
	if d, i := readState(root, sid); !d || !i {
		t.Fatalf("source edit: got design=%v impl=%v", d, i)
	}
}

func TestPostIgnoresNonSourceAndUnwatched(t *testing.T) {
	root := managedRoot(t) // no impl configured
	sid := "s-post2"
	t.Cleanup(func() { clearState(root, sid) })
	runEvent(t, root, editEvent("PostToolUse", "Write", sid, filepath.Join(root, "README.md")))
	runEvent(t, root, editEvent("PostToolUse", "Write", sid, filepath.Join(root, "src", "main.go"))) // impl not set
	if d, i := readState(root, sid); d || i {
		t.Fatalf("nothing watched was edited: got design=%v impl=%v", d, i)
	}
}

// --- Stop: gates run before the turn ends ---

const crmDesign = "../../examples/go-crm/design"

func TestStopSilentWhenNothingTouched(t *testing.T) {
	root := managedRoot(t)
	out := runEvent(t, root, Input{SessionID: "s-idle", HookEventName: "Stop"})
	if out != "" {
		t.Fatalf("a session that touched nothing must stop silently, got %q", out)
	}
}

func TestStopGreenDesignClearsStateSilently(t *testing.T) {
	root := t.TempDir()
	copyTree(t, crmDesign, filepath.Join(root, "design"))
	sid := "s-green"
	appendState(root, sid, "design")
	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	if out != "" {
		t.Fatalf("green gates must be silent, got %q", out)
	}
	if d, i := readState(root, sid); d || i {
		t.Fatal("state must clear after a green pass")
	}
}

func TestStopDriftBlocks(t *testing.T) {
	root := t.TempDir()
	copyTree(t, crmDesign, filepath.Join(root, "design"))
	oracle := filepath.Join(root, "design", "machines", "Deal.oracle.md")
	raw, err := os.ReadFile(oracle)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, oracle, string(raw)+"\ntampered\n")
	sid := "s-drift"
	t.Cleanup(func() { clearState(root, sid) })
	appendState(root, sid, "design")

	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision != "block" {
		t.Fatalf("a stale oracle must block the stop, got %+v", got)
	}
	if !strings.Contains(got.Reason, "DRIFT") {
		t.Fatalf("block reason must carry the gate output, got %q", got.Reason)
	}
	if d, _ := readState(root, sid); !d {
		t.Fatal("state must survive a block so the re-check runs after the fix")
	}

	// the continuation already happened once: surface, never loop
	out = runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop", StopHookActive: true})
	got = stopOut{}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision == "block" || got.SystemMessage == "" {
		t.Fatalf("with stop_hook_active the hook must warn, not block again: %+v", got)
	}
	if d, i := readState(root, sid); d || i {
		t.Fatal("state must clear once the block gives way to a warning")
	}
}

func TestStopMidPhaseErrorsWarnOnly(t *testing.T) {
	root := managedRoot(t)
	// Phase 2 in flight: an ARCHITECTURE.md with no parseable contract is an
	// ERROR, but no machines and no BUILD.md exist, so g3/gx do not apply.
	writeFile(t, filepath.Join(root, "design", "ARCHITECTURE.md"), "# Architecture\n(draft)\n")
	sid := "s-midphase"
	t.Cleanup(func() { clearState(root, sid) })
	appendState(root, sid, "design")

	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision == "block" {
		t.Fatalf("mid-phase ERRORs must not block a non-strict stop: %+v", got)
	}
	if !strings.Contains(got.SystemMessage, "ERROR") {
		t.Fatalf("the warning must still surface the red gates: %+v", got)
	}
}

func TestStopStrictBlocksOnAnyFinding(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"strict": true}`)
	writeFile(t, filepath.Join(root, "design", "domain.modelith.yaml"), "model: {}\n")
	writeFile(t, filepath.Join(root, "design", "ARCHITECTURE.md"), "# Architecture\n(draft)\n")
	sid := "s-strict"
	t.Cleanup(func() { clearState(root, sid) })
	appendState(root, sid, "design")

	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision != "block" {
		t.Fatalf("strict mode must block on any blocking finding: %+v", got)
	}
}

func TestStopBeforeAnyGateApplies(t *testing.T) {
	root := managedRoot(t) // Phase 1 only: nothing for any gate to check yet
	sid := "s-phase1"
	appendState(root, sid, "design")
	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	if out != "" {
		t.Fatalf("with no applicable gate the stop must be silent, got %q", out)
	}
	if d, _ := readState(root, sid); d {
		t.Fatal("state must clear when no gate applies")
	}
}

func TestStopMissingDesignDirWarns(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"blueprint"}`)
	sid := "s-nodir"
	appendState(root, sid, "design")
	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision == "block" || !strings.Contains(got.SystemMessage, "blueprint") {
		t.Fatalf("a missing design dir warns and skips, got %+v", got)
	}
}

// --- SessionStart: the governance announcement ---

func TestSessionStartAnnouncesGovernance(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"gates":"g2,g4","impl":"."}`)
	writeFile(t, filepath.Join(root, "design", "STATE.md"), "Phase 1: gate-passed\nPhase 2: in-progress\n")
	out := runEvent(t, root, Input{HookEventName: "SessionStart"})
	for _, want := range []string{"machinery-managed", "g2,g4", "oracle.md", "STATE.md", "Phase 2: in-progress"} {
		if !strings.Contains(out, want) {
			t.Fatalf("session context missing %q:\n%s", want, out)
		}
	}
}

func TestSessionStartSilentWhenUnmanaged(t *testing.T) {
	out := runEvent(t, t.TempDir(), Input{HookEventName: "SessionStart"})
	if out != "" {
		t.Fatalf("unmanaged repos get no session context, got %q", out)
	}
}

// --- state ledger isolation ---

func TestStatePathIsolatesSessionsAndRoots(t *testing.T) {
	rootA, rootB := t.TempDir(), t.TempDir()
	if statePath(rootA, "s1") == statePath(rootB, "s1") {
		t.Fatal("different roots must not share a ledger")
	}
	if statePath(rootA, "s1") == statePath(rootA, "s2") {
		t.Fatal("different sessions must not share a ledger")
	}
	if p := statePath(rootA, "../../etc/passwd"); strings.Contains(filepath.Base(p), "/") {
		t.Fatalf("session id must be sanitized into the filename, got %q", p)
	}
}

// --- the plugin wiring itself: a regression net over the shipped files ---

func repoPath(parts ...string) string {
	return filepath.Join(append([]string{"..", ".."}, parts...)...)
}

func TestPluginHooksJSONWiring(t *testing.T) {
	raw, err := os.ReadFile(repoPath("hooks", "hooks.json"))
	if err != nil {
		t.Fatalf("hooks/hooks.json must ship with the plugin: %v", err)
	}
	var doc struct {
		Description string `json:"description"`
		Hooks       map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
				Timeout int    `json:"timeout"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("hooks.json does not parse: %v", err)
	}
	if len(doc.Hooks) == 0 {
		t.Fatal("hooks.json must use the plugin wrapper format with a hooks key")
	}
	for _, ev := range []string{"PreToolUse", "PostToolUse", "Stop", "SubagentStop", "SessionStart"} {
		entries, ok := doc.Hooks[ev]
		if !ok || len(entries) == 0 {
			t.Fatalf("hooks.json missing event %s", ev)
		}
		for _, e := range entries {
			for _, h := range e.Hooks {
				if h.Type != "command" {
					t.Fatalf("%s: only command hooks are shipped, got %q", ev, h.Type)
				}
				if h.Command != "${CLAUDE_PLUGIN_ROOT}/hooks/machinery-hook.sh" {
					t.Fatalf("%s: every hook must route through the shim, got %q", ev, h.Command)
				}
				if h.Timeout <= 0 {
					t.Fatalf("%s: hooks must carry an explicit timeout", ev)
				}
			}
		}
	}
	fi, err := os.Stat(repoPath("hooks", "machinery-hook.sh"))
	if err != nil {
		t.Fatalf("the shim must exist: %v", err)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Fatal("hooks/machinery-hook.sh must be executable")
	}
}

func TestPluginManifests(t *testing.T) {
	var plugin struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	raw, err := os.ReadFile(repoPath(".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &plugin); err != nil {
		t.Fatalf("plugin.json does not parse: %v", err)
	}
	if plugin.Name != "machinery" || plugin.Version == "" {
		t.Fatalf("plugin.json must name and version the plugin, got %+v", plugin)
	}

	var mkt struct {
		Name    string `json:"name"`
		Plugins []struct {
			Name   string `json:"name"`
			Source string `json:"source"`
		} `json:"plugins"`
	}
	raw, err = os.ReadFile(repoPath(".claude-plugin", "marketplace.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &mkt); err != nil {
		t.Fatalf("marketplace.json does not parse: %v", err)
	}
	if len(mkt.Plugins) != 1 || mkt.Plugins[0].Name != "machinery" || mkt.Plugins[0].Source != "./" {
		t.Fatalf("marketplace must list the repo root as the machinery plugin, got %+v", mkt)
	}

	// the plugin reuses the repo's own skill, agents, and commands
	for _, p := range [][]string{
		{"skills", "machinery", "SKILL.md"},
		{"agents", "machinery-fsm-author.md"},
		{"agents", "machinery-build-writer.md"},
		{"commands", "design.md"},
		{"commands", "check.md"},
		{"commands", "init.md"},
		{"commands", "status.md"},
	} {
		if _, err := os.Stat(repoPath(p...)); err != nil {
			t.Fatalf("plugin component missing: %s", filepath.Join(p...))
		}
	}
}

// TestShimNoopContract documents the shim's stdin-independence: for an
// unmanaged root the shim must exit before it ever reads stdin or looks for
// the binary. Exercised here by running the shim when sh is available.
func TestShimNoopContract(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no /bin/sh")
	}
	root := t.TempDir()
	out, errOut, code := runShim(t, root, `{"hook_event_name":"Stop"}`)
	if code != 0 || out != "" || errOut != "" {
		t.Fatalf("unmanaged root: shim must be a silent no-op, got code=%d out=%q err=%q", code, out, errOut)
	}
}

func runShim(t *testing.T, projectDir, stdin string) (stdout, stderr string, code int) {
	t.Helper()
	shim, err := filepath.Abs(repoPath("hooks", "machinery-hook.sh"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "/bin/sh", shim)
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
	cmd.Stdin = strings.NewReader(stdin)
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	err = cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
	default:
		t.Fatalf("shim: %v", err)
	}
	return so.String(), se.String(), code
}
