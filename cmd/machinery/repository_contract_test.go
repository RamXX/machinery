package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestRepositoryVersionContracts prevents the documentation/tool metadata
// drift that is otherwise easy to introduce during a release or Go bump.
func TestRepositoryVersionContracts(t *testing.T) {
	root := repoRootDir(t)
	goMod := mustRepositoryFile(t, filepath.Join(root, "go.mod"))
	match := regexp.MustCompile(`(?m)^go ([0-9]+\.[0-9]+\.[0-9]+)$`).FindStringSubmatch(goMod)
	if len(match) != 2 {
		t.Fatal("go.mod must pin a full Go patch version")
	}
	readme := mustRepositoryFile(t, filepath.Join(root, "README.md"))
	if !strings.Contains(readme, "`go.mod` pins "+match[1]) {
		t.Fatalf("README Go pin is stale; want %s", match[1])
	}

	var plugin struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(mustRepositoryFile(t, filepath.Join(root, ".claude-plugin", "plugin.json"))), &plugin); err != nil {
		t.Fatal(err)
	}
	wantDev := "v" + plugin.Version + "-dev"
	var codexPlugin struct {
		Version string `json:"version"`
		Skills  string `json:"skills"`
	}
	if err := json.Unmarshal([]byte(mustRepositoryFile(t, filepath.Join(root, ".codex-plugin", "plugin.json"))), &codexPlugin); err != nil {
		t.Fatal(err)
	}
	if codexPlugin.Version != plugin.Version || codexPlugin.Skills != "./skills/" {
		t.Fatalf("Codex manifest = %+v, want Claude version %s and shared skills path", codexPlugin, plugin.Version)
	}
	if version != wantDev {
		t.Fatalf("binary dev version = %q, plugin metadata requires %q", version, wantDev)
	}
	makefile := mustRepositoryFile(t, filepath.Join(root, "Makefile"))
	if !strings.Contains(makefile, "INTERNAL_VERSION := "+wantDev) {
		t.Fatalf("Makefile INTERNAL_VERSION must be %s", wantDev)
	}
}

func TestOpenCodeAdapterContracts(t *testing.T) {
	root := repoRootDir(t)
	plugin := mustRepositoryFile(t, filepath.Join(root, "adapters", "opencode", "plugins", "machinery.js"))
	for _, required := range []string{
		`"tool.execute.before"`,
		`"tool.execute.after"`,
		`"session.idle"`,
		`machinery hook --root`,
		`args.patchText`,
		`input.args`,
		`client.tui.showToast`,
		`stop_hook_active: true`,
		`throw new Error(reason)`,
	} {
		if !strings.Contains(plugin, required) {
			t.Errorf("OpenCode adapter is missing protocol behavior %q", required)
		}
	}
	for _, command := range []string{"design.md", "check.md", "init.md", "status.md"} {
		doc := mustRepositoryFile(t, filepath.Join(root, "adapters", "opencode", "commands", command))
		if !strings.HasPrefix(doc, "---\n") || !strings.Contains(doc, "description:") {
			t.Errorf("OpenCode command %s has no valid frontmatter", command)
		}
	}
}

func TestAgentPortabilityDocumentationContracts(t *testing.T) {
	root := repoRootDir(t)
	readme := mustRepositoryFile(t, filepath.Join(root, "README.md"))
	if !strings.Contains(readme, "[agent portability guide](docs/agent-portability.md)") {
		t.Fatal("README must link the agent portability guide")
	}
	guide := mustRepositoryFile(t, filepath.Join(root, "docs", "agent-portability.md"))
	for _, required := range []string{
		"machinery install --target all",
		"machinery update --version",
		"machinery doctor --target all",
		"machinery uninstall --target all",
		"Missing subagent support falls back",
		"OpenCode's event API",
		"CI remains authoritative",
	} {
		if !strings.Contains(guide, required) {
			t.Errorf("agent portability guide is missing %q", required)
		}
	}
}

func mustRepositoryFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
