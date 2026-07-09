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
	if version != wantDev {
		t.Fatalf("binary dev version = %q, plugin metadata requires %q", version, wantDev)
	}
	makefile := mustRepositoryFile(t, filepath.Join(root, "Makefile"))
	if !strings.Contains(makefile, "INTERNAL_VERSION := "+wantDev) {
		t.Fatalf("Makefile INTERNAL_VERSION must be %s", wantDev)
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
