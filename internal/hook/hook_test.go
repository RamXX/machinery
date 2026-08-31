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
	"time"

	"github.com/RamXX/machinery/internal/gates"
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

func codexPatchEvent(event, sessionID, patch string) Input {
	return Input{
		SessionID:     sessionID,
		HookEventName: event,
		ToolName:      "apply_patch",
		ToolInput:     toolInput{Command: patch},
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
		{"design/ratchet.json", "Edit", true, "machinery baseline"},
		{"design/ratchet.json", "Write", true, "defeats the ratchet"},
		{"design/formal/Policy.als", "Write", true, "machinery alloy"},
		{"design/formal/Deal.semantics.yaml", "Edit", false, ""},    // annotation source
		{"design/formal/policy.relational.yaml", "Edit", false, ""}, // annotation source
		{"design/machines/Deal.machine.json", "Edit", false, ""},    // machine source
		{"design/machines/Deal.matrix.md", "Edit", false, ""},       // hand matrix
		{"design/domain.modelith.md", "Edit", false, ""},            // rendered, but post-processed by hand
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

func TestCodexApplyPatchDeniesAnyGeneratedArtifact(t *testing.T) {
	root := managedRoot(t)
	patch := "*** Begin Patch\n" +
		"*** Update File: src/main.go\n" +
		"@@\n-old\n+new\n" +
		"*** Update File: design/machines/Deal.oracle.md\n" +
		"@@\n-old\n+new\n" +
		"*** End Patch"
	out := runEvent(t, root, codexPatchEvent("PreToolUse", "s-codex-pre", patch))
	var got preOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Codex deny output is not JSON: %v (%q)", err, out)
	}
	if got.HookSpecificOutput.PermissionDecision != "deny" {
		t.Fatalf("a multi-file patch touching a generated artifact must be denied: %+v", got)
	}
	if !strings.Contains(got.HookSpecificOutput.PermissionDecisionReason, "machinery oracle") {
		t.Fatalf("deny reason must identify the regeneration command: %+v", got)
	}
}

func TestEditedPathsParsesCodexPatchOperations(t *testing.T) {
	in := codexPatchEvent("PreToolUse", "s-paths", "*** Begin Patch\n"+
		"*** Add File: src/new.go\n"+
		"*** Update File: src/old.go\n"+
		"*** Move to: src/moved.go\n"+
		"*** Delete File: src/gone.go\n"+
		"*** Update File: src/old.go\n"+
		"*** End Patch")
	got := editedPaths(in)
	want := []string{"src/new.go", "src/old.go", "src/gone.go", "src/moved.go"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("editedPaths = %v, want %v", got, want)
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

func TestCodexApplyPatchRecordsAllTouchedClasses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"impl":"."}`)
	sid := "s-codex-post"
	t.Cleanup(func() { clearState(root, sid) })
	patch := "*** Begin Patch\n" +
		"*** Update File: design/machines/Deal.machine.json\n" +
		"@@\n-old\n+new\n" +
		"*** Add File: src/main.go\n" +
		"+package main\n" +
		"*** End Patch"
	runEvent(t, root, codexPatchEvent("PostToolUse", sid, patch))
	if d, i := readState(root, sid); !d || !i {
		t.Fatalf("Codex multi-file patch: got design=%v impl=%v", d, i)
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
	// Phase 1 skeleton: the conventional marker exists, so gc applies from
	// turn one exactly as the CLI default suite arms it. An empty model is a
	// finding ("an empty check is a failure, not a pass"), surfaced as the
	// informational mid-phase message, never a block (no DRIFT, not strict).
	root := managedRoot(t)
	sid := "s-phase1"
	appendState(root, sid, "design")
	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision != "" {
		t.Fatalf("a mid-phase ERROR must not block: %+v", got)
	}
	if !strings.Contains(got.SystemMessage, "ERROR finding(s)") {
		t.Fatalf("the empty-model carrier finding must surface in the message: %+v", got)
	}
	if d, _ := readState(root, sid); d {
		t.Fatal("state must clear after a non-blocking stop")
	}
}

// TestSelectGatesProgressiveOptional locks progressive opt-in behavior: each
// relational annotation and the migration contract turns on its own gate at
// stop time without requiring configuration.
func TestSelectGatesProgressiveOptional(t *testing.T) {
	dir := t.TempDir()
	formal := filepath.Join(dir, "formal")
	if err := os.MkdirAll(formal, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "legacy"), 0o755); err != nil {
		t.Fatal(err)
	}
	sel, _ := selectGates(dir, Config{})
	for _, g := range []string{"gm", "gs", "gp", "gi", "gn"} {
		if sel.Run[g] {
			t.Errorf("%s must not run before its annotation exists", g)
		}
	}
	writeFile(t, filepath.Join(dir, "migration.yaml"), "contract_version: 1\n")
	writeFile(t, filepath.Join(dir, "legacy", "surface.yaml"), "surface_version: 1\n")
	writeFile(t, filepath.Join(formal, "policy.relational.yaml"), "subjects: {}\n")
	writeFile(t, filepath.Join(formal, "integrity.relational.yaml"), "entities: []\n")
	writeFile(t, filepath.Join(formal, "isolation.relational.yaml"), "tenant: {}\n")
	sel, _ = selectGates(dir, Config{})
	for _, g := range []string{"gm", "gs", "gp", "gi", "gn"} {
		if !sel.Run[g] {
			t.Errorf("%s must run once its opt-in artifact exists", g)
		}
	}
}

// The stop hook mirrors the CLI default suite for every checkable-from-design
// gate: gc arms on the domain model, gd on machines, gk on the external-
// checker layer. Omitting gk once let checker DRIFT (a stale committed
// projection after a mid-session model edit) pass the turn end green while
// the CLI reported it and exited 1.
func TestSelectGatesArmsCarrierIdciteCheckers(t *testing.T) {
	dir := t.TempDir()
	sel, _ := selectGates(dir, Config{})
	for _, g := range []string{"gc", "gd", "gk"} {
		if sel.Run[g] {
			t.Errorf("%s must not run before its artifact exists: %v", g, sel.Run)
		}
	}
	writeFile(t, filepath.Join(dir, "domain.modelith.yaml"), "entities: {}\n")
	writeFile(t, filepath.Join(dir, "machines", "Order.machine.json"), "{}\n")
	writeFile(t, filepath.Join(dir, "checkers", "pii.checker.yaml"), "checker_version: 1\n")
	sel, _ = selectGates(dir, Config{})
	for _, g := range []string{"gc", "gd", "gk"} {
		if !sel.Run[g] {
			t.Errorf("%s must run once its artifact exists (CLI parity): %v", g, sel.Run)
		}
	}
}

// A machine-less decomposed parent (decomposition.yaml, BUILD.md, an empty
// machines/ directory) must not select Gx at stop time: its behavior layer is
// the children's, and Gx against the parent's BUILD.md would fail it for
// phases that live in the child designs. The artifact-activated gates
// (gm/gs/gp/gi/gn) keep their auto-activation on that same parent (the v0.3.0
// CLI narrowing regression dropped gp/gi/gn; the hook path must never copy
// that). Once machines exist the full selection returns.
func TestSelectGatesSkipsGxOnMachinelessDecomposedParent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "checkout.modelith.yaml"), "entities: {}\n")
	writeFile(t, filepath.Join(dir, "decomposition.yaml"), "decomposition_version: 1\n")
	writeFile(t, filepath.Join(dir, "BUILD.md"), "Mode: full\n")
	writeFile(t, filepath.Join(dir, "ARCHITECTURE.md"), "# arch\n")
	writeFile(t, filepath.Join(dir, "migration.yaml"), "contract_version: 1\n")
	writeFile(t, filepath.Join(dir, "legacy", "surface.yaml"), "surface_version: 1\n")
	writeFile(t, filepath.Join(dir, "formal", "policy.relational.yaml"), "subjects: {}\n")
	writeFile(t, filepath.Join(dir, "formal", "integrity.relational.yaml"), "entities: []\n")
	writeFile(t, filepath.Join(dir, "formal", "isolation.relational.yaml"), "tenant: {}\n")
	if err := os.MkdirAll(filepath.Join(dir, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	sel, _ := selectGates(dir, Config{})
	if sel.Run["gx"] || sel.Run["g3"] {
		t.Errorf("machine-less decomposed parent must not select g3/gx: %v", sel.Run)
	}
	if !sel.Run["g2"] || !sel.Run["g5"] {
		t.Errorf("machine-less decomposed parent must select g2,g5: %v", sel.Run)
	}
	for _, g := range []string{"gm", "gs", "gp", "gi", "gn"} {
		if !sel.Run[g] {
			t.Errorf("machine-less decomposed parent dropped artifact-activated gate %s: %v", g, sel.Run)
		}
	}
	writeFile(t, filepath.Join(dir, "machines", "Order.machine.json"), "{}\n")
	sel, _ = selectGates(dir, Config{})
	if !sel.Run["gx"] || !sel.Run["g3"] {
		t.Errorf("with machines present g3,gx must return: %v", sel.Run)
	}
}

// The stop hook selects Gx as soon as the model and machines exist: BUILD.md
// is not a precondition, or phase-3 Gx DRIFT (a stale maps-to reference)
// escapes the drift-blocking contract until Phase 4 (GATE-6).
func TestSelectGatesGxWithoutBuildDoc(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "domain.modelith.yaml"), "entities: {}\n")
	writeFile(t, filepath.Join(dir, "machines", "Order.machine.json"), "{}\n")
	sel, _ := selectGates(dir, Config{})
	if !sel.Run["gx"] {
		t.Errorf("modelith + machines must select gx even without BUILD.md: %v", sel.Run)
	}
	if sel.Run["gb"] {
		t.Errorf("gb still needs BUILD.md: %v", sel.Run)
	}
}

// Machine detection must survive glob metacharacters in the project path: a
// design under "pr[1]" once defeated filepath.Glob, silently dropping g3 and
// letting committed-oracle DRIFT pass at stop time (GATE-3/GATE-7 hook half).
func TestSelectGatesMetacharDesignPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pr[1]", "design")
	writeFile(t, filepath.Join(dir, "domain.modelith.yaml"), "entities: {}\n")
	writeFile(t, filepath.Join(dir, "machines", "Order.machine.json"), "{}\n")
	sel, _ := selectGates(dir, Config{})
	if !sel.Run["g3"] || !sel.Run["gx"] {
		t.Errorf("a metachar path must not defeat machine detection: %v", sel.Run)
	}
}

// The governance configuration itself is agent-read-only: a Write of
// {"hooks": false} to .machinery.json is how an agent would switch machinery
// off (GATE-10).
func TestPreDeniesGovernanceConfigEdits(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), "{}\n")
	for _, tool := range []string{"Write", "Edit"} {
		out := runEvent(t, root, editEvent("PreToolUse", tool, "s-gov", filepath.Join(root, ConfigName)))
		var got preOut
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("%s deny output is not JSON: %v (%q)", tool, err, out)
		}
		if got.HookSpecificOutput.PermissionDecision != "deny" {
			t.Fatalf("%s of %s must be denied: %+v", tool, ConfigName, got)
		}
		if !strings.Contains(got.HookSpecificOutput.PermissionDecisionReason, "governance") {
			t.Fatalf("the reason must say why: %+v", got)
		}
	}
}

// Deleting a governance marker via a Codex apply_patch switches machinery off
// for the whole repo; updating the domain model is Phase 1 authoring and
// stays allowed (GATE-10).
func TestCodexDeleteOfGovernanceMarkerDenied(t *testing.T) {
	root := managedRoot(t)
	deny := codexPatchEvent("PreToolUse", "s-del", "*** Begin Patch\n"+
		"*** Delete File: design/domain.modelith.yaml\n"+
		"*** End Patch")
	out := runEvent(t, root, deny)
	var got preOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("deny output is not JSON: %v (%q)", err, out)
	}
	if got.HookSpecificOutput.PermissionDecision != "deny" {
		t.Fatalf("deleting the design marker must be denied: %+v", got)
	}
	allow := codexPatchEvent("PreToolUse", "s-upd", "*** Begin Patch\n"+
		"*** Update File: design/domain.modelith.yaml\n"+
		"@@\n-old\n+new\n"+
		"*** End Patch")
	if out := runEvent(t, root, allow); out != "" {
		t.Fatalf("updating the domain model is Phase 1 authoring and must stay allowed, got %q", out)
	}
}

// The wave sentinel defers stop-time gating while it is fresh, so an agent
// that could touch it would defer the gates indefinitely. It is operator-only:
// file-tool creates and edits are denied wherever the base name appears, while
// deleting it (the documented way to close a wave) stays allowed.
func TestPreDeniesWaveSentinelWrites(t *testing.T) {
	root := managedRoot(t)
	cases := []struct {
		name string
		rel  string
		tool string
		deny bool
	}{
		{"root design dir, Write", "design/.machinery-wave", "Write", true},
		{"root design dir, Edit", "design/.machinery-wave", "Edit", true},
		{"root design dir, MultiEdit", "design/.machinery-wave", "MultiEdit", true},
		{"nested child design", "design/children/billing/.machinery-wave", "Write", true},
		{"outside the design dir", "ops/.machinery-wave", "Write", true},
		{"lookalike suffix is not the sentinel", "design/.machinery-wave.bak", "Write", false},
		{"lookalike prefix is not the sentinel", "design/wave.md", "Write", false},
		{"Bash is not a governed file tool", "design/.machinery-wave", "Bash", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := runEvent(t, root, editEvent("PreToolUse", c.tool, "s-wave", filepath.Join(root, c.rel)))
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
				t.Fatalf("writing the wave sentinel must be denied: %+v", got)
			}
			reason := got.HookSpecificOutput.PermissionDecisionReason
			for _, want := range []string{c.rel, "wave sentinel", "operator-created"} {
				if !strings.Contains(reason, want) {
					t.Fatalf("reason %q missing %q", reason, want)
				}
			}
		})
	}
}

// The filesystems this hook guards on (APFS, NTFS) resolve names case-
// insensitively, so the enforcement reads (os.Stat of the sentinel,
// os.ReadFile of the config) find a case-variant spelling; the deny must
// fold case too or the guard is bypassable by writing .MACHINERY-WAVE.
func TestPreDeniesCaseVariants(t *testing.T) {
	root := managedRoot(t)
	cases := []struct{ name, rel, want string }{
		{"wave sentinel upper", "design/.MACHINERY-WAVE", "wave sentinel"},
		{"governance config mixed", ".Machinery.json", "governance configuration"},
		{"ratchet mixed", "design/Ratchet.json", "machinery baseline"},
		{"oracle mixed suffix", "design/machines/Thing.Oracle.MD", "machinery oracle"},
		{"frozen pack mixed", "design/Pack/events.md", "frozen pack"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := runEvent(t, root, editEvent("PreToolUse", "Write", "s-fold", filepath.Join(root, c.rel)))
			var got preOut
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("deny output is not JSON: %v (%q)", err, out)
			}
			if got.HookSpecificOutput.PermissionDecision != "deny" {
				t.Fatalf("case variant %s must be denied: %+v", c.rel, got)
			}
			if !strings.Contains(got.HookSpecificOutput.PermissionDecisionReason, c.want) {
				t.Fatalf("reason %q missing %q", got.HookSpecificOutput.PermissionDecisionReason, c.want)
			}
		})
	}
}

func TestCodexPatchWaveSentinel(t *testing.T) {
	root := managedRoot(t)
	add := codexPatchEvent("PreToolUse", "s-wave-add", "*** Begin Patch\n"+
		"*** Add File: design/.machinery-wave\n"+
		"+240\n"+
		"*** End Patch")
	out := runEvent(t, root, add)
	var got preOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("deny output is not JSON: %v (%q)", err, out)
	}
	if got.HookSpecificOutput.PermissionDecision != "deny" {
		t.Fatalf("an apply_patch that creates the sentinel must be denied: %+v", got)
	}
	del := codexPatchEvent("PreToolUse", "s-wave-del", "*** Begin Patch\n"+
		"*** Delete File: design/.machinery-wave\n"+
		"*** End Patch")
	if out := runEvent(t, root, del); out != "" {
		t.Fatalf("deleting the sentinel closes the wave and must stay allowed, got %q", out)
	}
}

// A delete of the sentinel is allowed because it closes the wave, but the
// delete must not launder any other write to the same path in the same tool
// call: a patch that deletes and re-adds .machinery-wave reopens a fresh
// full-TTL wave in one call and must be denied.
func TestCodexPatchWaveSentinelDeleteDoesNotLaunderRewrite(t *testing.T) {
	root := managedRoot(t)
	cases := []struct {
		name  string
		patch string
	}{
		{"delete then add", "*** Begin Patch\n" +
			"*** Delete File: design/.machinery-wave\n" +
			"*** Add File: design/.machinery-wave\n" +
			"+240\n" +
			"*** End Patch"},
		{"add then delete", "*** Begin Patch\n" +
			"*** Add File: design/.machinery-wave\n" +
			"+240\n" +
			"*** Delete File: design/.machinery-wave\n" +
			"*** End Patch"},
		{"delete then update", "*** Begin Patch\n" +
			"*** Delete File: design/.machinery-wave\n" +
			"*** Update File: design/.machinery-wave\n" +
			"@@\n-45\n+240\n" +
			"*** End Patch"},
		{"delete one sentinel, add another", "*** Begin Patch\n" +
			"*** Delete File: design/.machinery-wave\n" +
			"*** Add File: design/children/billing/.machinery-wave\n" +
			"+240\n" +
			"*** End Patch"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := runEvent(t, root, codexPatchEvent("PreToolUse", "s-wave-relaunder", c.patch))
			var got preOut
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("deny output is not JSON: %v (%q)", err, out)
			}
			if got.HookSpecificOutput.PermissionDecision != "deny" {
				t.Fatalf("re-creating the sentinel in the same patch that deletes it must be denied: %+v", got)
			}
			reason := got.HookSpecificOutput.PermissionDecisionReason
			for _, want := range []string{"wave sentinel", "operator-created"} {
				if !strings.Contains(reason, want) {
					t.Fatalf("reason %q missing %q", reason, want)
				}
			}
		})
	}
}

// editedOps is the substrate the per-operation deny rests on: a single patch
// that deletes and re-adds one path must report both operations, and a plain
// file-tool write must never be classified as a delete.
func TestEditedOpsReportsOperationPerPath(t *testing.T) {
	patch := "*** Begin Patch\n" +
		"*** Delete File: design/.machinery-wave\n" +
		"*** Add File: design/.machinery-wave\n" +
		"+240\n" +
		"*** Update File: design/notes.md\n" +
		"*** End Patch"
	got := editedOps(Input{ToolName: "apply_patch", ToolInput: toolInput{Command: patch}})
	want := []editedPath{
		{Path: "design/.machinery-wave", Op: opDelete},
		{Path: "design/.machinery-wave", Op: opAdd},
		{Path: "design/notes.md", Op: opUpdate},
	}
	if len(got) != len(want) {
		t.Fatalf("editedOps = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("editedOps[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	if paths := editedPaths(Input{ToolName: "apply_patch", ToolInput: toolInput{Command: patch}}); len(paths) != 2 {
		t.Fatalf("editedPaths must stay deduplicated by path, got %v", paths)
	}
	write := editedOps(Input{ToolName: "Write", ToolInput: toolInput{FilePath: "design/.machinery-wave"}})
	if len(write) != 1 || write[0].Op != opWrite {
		t.Fatalf("a file-tool write must report opWrite, got %v", write)
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

// writeG4Fixture builds a managed root whose impl has one undeclared
// cross-boundary import (alpha -> beta), so G4 is red at stop time.
func writeG4Fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"impl":"."}`)
	arch := "# Architecture\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: alpha\n    code: [\"alpha/**\"]\n  - id: beta\n    code: [\"beta/**\"]\n```\n"
	writeFile(t, filepath.Join(root, "design", "ARCHITECTURE.md"), arch)
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/m\n")
	writeFile(t, filepath.Join(root, "alpha", "a.go"), "package alpha\n\nimport \"example.com/m/beta\"\n")
	writeFile(t, filepath.Join(root, "beta", "b.go"), "package beta\n")
	return root
}

func TestStopImportFindingsDisarmedThenArmed(t *testing.T) {
	root := writeG4Fixture(t)
	sid := "s-arming"
	t.Cleanup(func() { clearState(root, sid) })

	// no ratchet.json: pre-baseline debt warns with the arming instruction
	appendState(root, sid, "impl")
	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision == "block" {
		t.Fatalf("import findings must not block before a baseline exists: %+v", got)
	}
	if !strings.Contains(got.SystemMessage, "disarmed") || !strings.Contains(got.SystemMessage, "machinery baseline") {
		t.Fatalf("the warning must name the arming step: %+v", got)
	}

	// an empty snapshot (greenfield arming) makes the same finding block
	writeFile(t, filepath.Join(root, "design", "ratchet.json"), `{"date":"2026-07","edges":{}}`)
	appendState(root, sid, "impl")
	out = runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	got = stopOut{}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision != "block" {
		t.Fatalf("with the baseline recorded an import finding must block: %+v", got)
	}
	if !strings.Contains(got.Reason, "undeclared cross-boundary edge") {
		t.Fatalf("the block must carry the gate output: %q", got.Reason)
	}
}

// A staged gates list naming the impl-facing gates (gt, g4) with no impl
// configured must not fail the stop, but the drop has to stay visible: a
// silently skipped gate is a configured-but-never-run gate.
func TestStopWarnsWhenStagedImplGatesLackImpl(t *testing.T) {
	// green design, otherwise-silent stop: the warning must still surface
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"gates":"g2,g3,gt"}`)
	copyTree(t, crmDesign, filepath.Join(root, "design"))
	sid := "s-dropped"
	t.Cleanup(func() { clearState(root, sid) })
	appendState(root, sid, "design")
	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision == "block" {
		t.Fatalf("a config gap must warn, never block: %+v", got)
	}
	if !strings.Contains(got.SystemMessage, "gt") || !strings.Contains(got.SystemMessage, "impl") {
		t.Fatalf("the dropped gate and the missing impl setting must be named: %+v", got)
	}

	// the whole staged list impl-facing: nothing runs, the warning names both
	writeFile(t, filepath.Join(root, ConfigName), `{"gates":"g4,gt"}`)
	appendState(root, sid, "design")
	out = runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	got = stopOut{}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if !strings.Contains(got.SystemMessage, "g4,gt") {
		t.Fatalf("an all-dropped list must still warn: %+v", got)
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
			if (ev == "PreToolUse" || ev == "PostToolUse") && !strings.Contains(e.Matcher, "apply_patch") {
				t.Fatalf("%s matcher must include the Codex apply_patch tool, got %q", ev, e.Matcher)
			}
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
		Skills  string `json:"skills"`
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
	claudeVersion := plugin.Version

	raw, err = os.ReadFile(repoPath(".codex-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &plugin); err != nil {
		t.Fatalf("Codex plugin.json does not parse: %v", err)
	}
	if plugin.Name != "machinery" || plugin.Version != claudeVersion || plugin.Skills != "./skills/" {
		t.Fatalf("Codex manifest must reuse the shared skill and match the Claude version, got %+v", plugin)
	}
	skillRaw, err := os.ReadFile(repoPath("skills", "machinery", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(skillRaw), "version: \""+claudeVersion+"\"") {
		t.Fatalf("plugin version %s and skill metadata version diverge", claudeVersion)
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

func TestShimFindsGitRootWithoutClaudeEnvironment(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for Codex root discovery")
	}
	root := t.TempDir()
	cmd := exec.CommandContext(t.Context(), "git", "init", "-q", root)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	writeFile(t, filepath.Join(root, ConfigName), `{}`)
	subdir := filepath.Join(root, "nested", "work")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	fake := filepath.Join(binDir, "machinery")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nprintf '%s' \"$*\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	shim, err := filepath.Abs(repoPath("hooks", "machinery-hook.sh"))
	if err != nil {
		t.Fatal(err)
	}
	hookCmd := exec.CommandContext(t.Context(), "/bin/sh", shim)
	hookCmd.Dir = subdir
	hookCmd.Env = withoutEnv(os.Environ(), "CLAUDE_PROJECT_DIR")
	hookCmd.Env = append(hookCmd.Env, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	hookCmd.Stdin = strings.NewReader(`{"hook_event_name":"SessionStart"}`)
	out, err := hookCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shim: %v: %s", err, out)
	}
	rootCmd := exec.CommandContext(t.Context(), "git", "-C", subdir, "rev-parse", "--show-toplevel")
	rootOut, err := rootCmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	want := "hook --root " + strings.TrimSpace(string(rootOut))
	if string(out) != want {
		t.Fatalf("shim invocation = %q, want %q", out, want)
	}
}

func withoutEnv(env []string, key string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env))
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			out = append(out, entry)
		}
	}
	return out
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

// The stop-time selection matches `machinery check`'s default activation for
// Ga: neither artifact, no gate; the acceptance directory or a milestone
// marked closed, gate.
func TestSelectGatesActivatesGaOnAcceptanceArtifacts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "BUILD.md"),
		"# B\n\nMode: full\n\n## Build plan\n\n**M0 - Walking skeleton.** DoD: T-X-01 green.\n")
	sel, _ := selectGates(dir, Config{})
	if sel.Run["ga"] {
		t.Error("ga must not run before a milestone is closed or evidence exists")
	}

	writeFile(t, filepath.Join(dir, "acceptance", "M0.yaml"), "milestone: 0\n")
	sel, _ = selectGates(dir, Config{})
	if !sel.Run["ga"] {
		t.Error("committed acceptance evidence must activate ga")
	}

	bare := t.TempDir()
	writeFile(t, filepath.Join(bare, "BUILD.md"),
		"# B\n\nMode: full\n\n## Build plan\n\n**M0 - Walking skeleton.** DoD: T-X-01 green.\nStatus: closed\n")
	sel, _ = selectGates(bare, Config{})
	if !sel.Run["ga"] {
		t.Error("a milestone marked closed must activate ga on its own")
	}
}

// The stop-time selection matches `machinery check`'s default activation for
// Gv: no evidence file, no gate; a committed attestations.yaml, gate. The
// stop hook is where staleness must surface, because the turn that edited a
// covered artifact is the turn that invalidated the judgment over it.
func TestSelectGatesActivatesGvOnAttestationEvidence(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ARCHITECTURE.md"), "# A\n")
	sel, _ := selectGates(dir, Config{})
	if sel.Run["gv"] {
		t.Error("gv must not run before attestation evidence is committed")
	}

	writeFile(t, filepath.Join(dir, gates.AttestationsFileName), "attestation_version: 1\nattestations: []\n")
	sel, _ = selectGates(dir, Config{})
	if !sel.Run["gv"] {
		t.Error("committed attestation evidence must activate gv")
	}
}

func TestWaveSentinel(t *testing.T) {
	d := t.TempDir()
	if _, _, active := waveSentinel(d); active {
		t.Fatal("absent sentinel reported active")
	}
	p := filepath.Join(d, ".machinery-wave")
	if err := os.WriteFile(p, []byte("wave open\n"), 0644); err != nil {
		t.Fatal(err)
	}
	left, stale, active := waveSentinel(d)
	if !active || stale || left == "" {
		t.Fatalf("fresh sentinel: left=%q stale=%v active=%v", left, stale, active)
	}
	if err := os.WriteFile(p, []byte("1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}
	_, stale, active = waveSentinel(d)
	if !active || !stale {
		t.Fatalf("expired sentinel: stale=%v active=%v", stale, active)
	}
}

// The upgrade protocol forbids mixing a binary upgrade with a design change;
// the warning fires exactly when a regenerated artifact's machinery-version
// stamp changed AND a hand-written design file changed in the same tree.
func TestUpgradeMixWarning(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q")
	writeFile(t, filepath.Join(root, "design", "domain.modelith.yaml"), "entities: {}\n")
	writeFile(t, filepath.Join(root, "design", "machines", "Order.oracle.md"),
		"<!-- machinery-version: v0.1.0 -->\n| test id | stable id |\n|---|---|\n| T-1 | ORDE-aaa |\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")

	if w := upgradeMixWarning(root, "design"); w != "" {
		t.Fatalf("clean tree must not warn: %q", w)
	}
	// stamp change alone: an upgrade in flight, no mixing
	writeFile(t, filepath.Join(root, "design", "machines", "Order.oracle.md"),
		"<!-- machinery-version: v0.2.0 -->\n| test id | stable id |\n|---|---|\n| T-1 | ORDE-aaa |\n")
	if w := upgradeMixWarning(root, "design"); w != "" {
		t.Fatalf("an upgrade alone must not warn: %q", w)
	}
	// plus a hand-written edit: the mix the protocol forbids
	writeFile(t, filepath.Join(root, "design", "domain.modelith.yaml"), "entities: {A: {}}\n")
	w := upgradeMixWarning(root, "design")
	if !strings.Contains(w, "mixes a binary upgrade") || !strings.Contains(w, "machines/Order.oracle.md") {
		t.Fatalf("mixed change set must warn naming the artifact: %q", w)
	}
	// hand edit alone (stamp reverted): no warning
	writeFile(t, filepath.Join(root, "design", "machines", "Order.oracle.md"),
		"<!-- machinery-version: v0.1.0 -->\n| test id | stable id |\n|---|---|\n| T-1 | ORDE-aaa |\n")
	if w := upgradeMixWarning(root, "design"); w != "" {
		t.Fatalf("a design edit alone must not warn: %q", w)
	}
}

// The plain dialog register swaps only the USER-FACING stop messages and adds
// a register reminder to session start; deny reasons, block reasons, and the
// default strings stay byte-identical (every other test in this file runs
// with the default register and pins that).
func TestDialogPlainRegister(t *testing.T) {
	t.Run("unknown value warns and clears", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ConfigName), `{"dialog":"terse"}`)
		cfg, ok, warn := Load(root)
		if !ok || cfg.Dialog != "" || !strings.Contains(warn, "dialog value") {
			t.Fatalf("cfg=%+v ok=%v warn=%q", cfg, ok, warn)
		}
	})
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"dialog":"plain"}`)
	writeFile(t, filepath.Join(root, "design", "domain.modelith.yaml"), "model: {}\n")
	t.Run("mid-phase stop message is plain", func(t *testing.T) {
		sid := "s-plain"
		appendState(root, sid, "design")
		out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
		var got stopOut
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("stop output is not JSON: %v (%q)", err, out)
		}
		if got.Decision != "" {
			t.Fatalf("plain register never changes blocking behavior: %+v", got)
		}
		if !strings.Contains(got.SystemMessage, "design-check item(s) are still open") {
			t.Fatalf("plain message expected: %+v", got)
		}
		for _, jargon := range []string{"gate ERROR", "DRIFT", "machinery check"} {
			if strings.Contains(got.SystemMessage, jargon) {
				t.Fatalf("plain message leaks %q: %+v", jargon, got)
			}
		}
	})
	t.Run("session start carries the register reminder", func(t *testing.T) {
		out := runEvent(t, root, Input{HookEventName: "SessionStart"})
		if !strings.Contains(out, "Dialog register: PLAIN") {
			t.Fatalf("session start must remind the conductor of the register, got %q", out)
		}
	})
}
