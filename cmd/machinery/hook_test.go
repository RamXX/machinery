package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runHookCmd pipes one hook event (JSON) through the hidden hook command.
func runHookCmd(t *testing.T, root, event string) string {
	t.Helper()
	oldIn, oldOut := stdinR, stdoutW
	defer func() { stdinR, stdoutW = oldIn, oldOut }()
	stdinR = strings.NewReader(event)
	var out bytes.Buffer
	stdoutW = &out
	cmd := newHookCmd()
	cmd.SetArgs([]string{"--root", root})
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatalf("hook command: %v", err)
	}
	return out.String()
}

func TestHookCmdNoopOutsideMachineryRepos(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateTestConfigDir(t))
	out := runHookCmd(t, t.TempDir(), `{"hook_event_name":"Stop","session_id":"s"}`)
	if out != "" {
		t.Fatalf("hook must be silent in a non-machinery repo, got %q", out)
	}
}

func TestHookCmdDeniesGeneratedEdit(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateTestConfigDir(t))
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "design"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "design", "domain.modelith.yaml"), []byte("model: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oracle := filepath.Join(root, "design", "machines", "Deal.oracle.md")
	event := `{"hook_event_name":"PreToolUse","tool_name":"Edit","session_id":"s","tool_input":{"file_path":` + jsonString(oracle) + `}}`
	out := runHookCmd(t, root, event)
	if !strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Fatalf("expected a deny for a generated-oracle edit, got %q", out)
	}
}

func TestHookCmdFailsClosedWhenInterruptedInstallCannotRecover(t *testing.T) {
	config := t.TempDir()
	t.Setenv("MACHINERY_CONFIG_DIR", config)
	journal := filepath.Join(config, ".machinery-install-journal")
	if err := os.MkdirAll(journal, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(journal, "journal.json"), []byte("{corrupt\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	oldIn, oldOut := stdinR, stdoutW
	defer func() { stdinR, stdoutW = oldIn, oldOut }()
	stdinR = strings.NewReader(`{"hook_event_name":"Stop","session_id":"s"}`)
	var out bytes.Buffer
	stdoutW = &out
	cmd := newHookCmd()
	cmd.SetArgs([]string{"--root", t.TempDir()})
	err := executeCapturedCommand(cmd)
	if err == nil || !strings.Contains(err.Error(), "recover interrupted install transaction") {
		t.Fatalf("hook recovery error = %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("governance ran before recovery failed: %q", out.String())
	}
}

func jsonString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func TestHookStateReleaseRequiresExactlyOneSelector(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{
		{"release", "--root", root},
		{"release", "--root", root, "--orphaned", "--token", strings.Repeat("a", 64)},
		{"release", "--orphaned"},
	} {
		cmd := newHookStateCmd()
		cmd.SetArgs(args)
		if err := executeCapturedCommand(cmd); err == nil {
			t.Fatalf("hook-state %v was accepted", args)
		}
	}
}

func TestHookStateAdoptOffersRebindIdentity(t *testing.T) {
	cmd := newHookStateCmd()
	adopt, _, err := cmd.Find([]string{"adopt"})
	if err != nil || adopt.Flags().Lookup("rebind-identity") == nil {
		t.Fatalf("adopt must offer --rebind-identity: %v", err)
	}
}

func TestHookStateRefusesInsideAnAgentSession(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	for _, args := range [][]string{
		{"release", "--root", t.TempDir(), "--orphaned"},
		{"adopt", "--root", t.TempDir(), "--rebind-identity"},
	} {
		cmd := newHookStateCmd()
		cmd.SetArgs(args)
		err := executeCapturedCommand(cmd)
		if err == nil || !strings.Contains(err.Error(), "operator command") {
			t.Fatalf("hook-state %v ran inside an agent session: %v", args, err)
		}
	}
}
