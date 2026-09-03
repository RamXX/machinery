//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGitRunBoundsDescendantRetainingOutputPipes(t *testing.T) {
	bin := t.TempDir()
	git := filepath.Join(bin, "git")
	script := "#!/bin/sh\n(sleep 30) &\nprintf 'direct-child-finished\\n'\nexit 0\n"
	if err := os.WriteFile(git, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	started := time.Now()
	stdout, _, err := gitRun(t.TempDir(), "status")
	if err == nil {
		t.Fatal("a descendant retaining git output pipes was reported as clean success")
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("git output-holder blocked for %s: %v", elapsed, err)
	}
	if !strings.Contains(stdout, "direct-child-finished") {
		t.Fatalf("bounded git output lost direct child output: %q", stdout)
	}
}

func TestGitRunRejectsExitZeroStderr(t *testing.T) {
	bin := t.TempDir()
	git := filepath.Join(bin, "git")
	if err := os.WriteFile(git, []byte("#!/bin/sh\nprintf 'unexpected diagnostic\\n' >&2\nprintf 'apparently-successful\\n'\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	stdout, stderr, err := gitRun(t.TempDir(), "status")
	if err == nil || stdout != "" || !strings.Contains(stderr, "unexpected diagnostic") {
		t.Fatalf("exit-zero git stderr was accepted: stdout=%q stderr=%q err=%v", stdout, stderr, err)
	}
}
