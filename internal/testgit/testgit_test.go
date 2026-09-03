package testgit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunIgnoresHostConfigSigningAndHooks(t *testing.T) {
	repo := t.TempDir()
	global := filepath.Join(t.TempDir(), "global.gitconfig")
	hooks := t.TempDir()
	if err := os.WriteFile(filepath.Join(hooks, "pre-commit"), []byte("#!/bin/sh\nexit 91\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := "[commit]\n\tgpgsign = true\n[core]\n\thooksPath = " + filepath.ToSlash(hooks) + "\n"
	if err := os.WriteFile(global, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.name", "Fixture"},
		{"config", "user.email", "fixture@example.invalid"},
	} {
		if output, err := Run(t.Context(), repo, args...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "fixture"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "fixture"}, {"commit", "-qm", "fixture"}} {
		if output, err := Run(context.Background(), repo, args...); err != nil {
			t.Fatalf("closed Git command %v inherited host behavior: %v: %s", args, err, output)
		}
	}
}
