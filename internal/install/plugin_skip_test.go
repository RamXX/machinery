package install

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallSkipsHomeServedByPlugin: when the machinery Claude Code plugin
// is already cached under ~/.claude, the default install must not lay a
// duplicate skill there; ~/.agents (and any explicit --home) still get one.
func TestInstallSkipsHomeServedByPlugin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// the plugin cache layout: <home>/.claude/plugins/cache/<marketplace>/<plugin>
	if err := os.MkdirAll(filepath.Join(home, ".claude", "plugins", "cache", "machinery", "machinery"), 0o755); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := Install(Options{From: "../..", Out: &out}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "machinery", "SKILL.md")); err != nil {
		t.Fatalf("~/.agents must still receive the canonical copy: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "machinery")); !os.IsNotExist(err) {
		t.Fatalf("~/.claude must be skipped when the plugin serves it (err=%v)", err)
	}
	if !strings.Contains(out.String(), "skipping") {
		t.Fatalf("the skip must be announced, got %q", out.String())
	}
}

// TestInstallExplicitHomeWinsOverPluginSkip: an explicit --home is honored
// even when the plugin is present; the filter applies to defaults only.
func TestInstallExplicitHomeWinsOverPluginSkip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	claude := filepath.Join(home, ".claude")
	if err := os.MkdirAll(filepath.Join(claude, "plugins", "cache", "machinery", "machinery"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Install(Options{Homes: []string{claude}, From: "../.."}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(claude, "skills", "machinery", "SKILL.md")); err != nil {
		t.Fatalf("an explicit home must be honored: %v", err)
	}
}
