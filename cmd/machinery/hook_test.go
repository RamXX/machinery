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
	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook command: %v", err)
	}
	return out.String()
}

func TestHookCmdNoopOutsideMachineryRepos(t *testing.T) {
	out := runHookCmd(t, t.TempDir(), `{"hook_event_name":"Stop","session_id":"s"}`)
	if out != "" {
		t.Fatalf("hook must be silent in a non-machinery repo, got %q", out)
	}
}

func TestHookCmdDeniesGeneratedEdit(t *testing.T) {
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
