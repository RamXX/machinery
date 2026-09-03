//go:build !windows

package testgit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func installFakeGit(t *testing.T) {
	t.Helper()
	bin := t.TempDir()
	script := `#!/bin/sh
case "${MACHINERY_TESTGIT_MODE:-}" in
  stdout-overflow)
    printf '0123456789'
    ;;
  stderr-overflow)
    printf 'abcdefghij' >&2
    ;;
  warning)
    printf 'warning from fake git\n' >&2
    ;;
  descendant-pipe)
    (sleep 30) &
    ;;
  *)
    exit 64
    ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestRunInputRejectsEachSuccessfulOutputOverflow(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode string
		want string
	}{
		{name: "stdout", mode: "stdout-overflow", want: "stdout exceeded 4-byte capture limit"},
		{name: "stderr", mode: "stderr-overflow", want: "stderr exceeded 4-byte capture limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			installFakeGit(t)
			t.Setenv("MACHINERY_TESTGIT_MODE", tc.mode)
			output, err := runInputWithLimits(t.Context(), "", nil, 4, 4, "status")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("overflow error = %v", err)
			}
			if len(output) > 2*(4+len("\n[output truncated at 4 bytes]\n")) {
				t.Fatalf("bounded output length = %d: %q", len(output), output)
			}
			if !strings.Contains(string(output), "[output truncated at 4 bytes]") {
				t.Fatalf("bounded output omitted truncation marker: %q", output)
			}
		})
	}
}

func TestRunInputRejectsSuccessfulGitDiagnostics(t *testing.T) {
	installFakeGit(t)
	t.Setenv("MACHINERY_TESTGIT_MODE", "warning")
	output, err := runInputWithLimits(t.Context(), "", nil, 64, 64, "status")
	if err == nil || !strings.Contains(err.Error(), "emitted stderr on success") || !strings.Contains(err.Error(), "warning from fake git") {
		t.Fatalf("successful warning error = %v", err)
	}
	if string(output) != "warning from fake git\n" {
		t.Fatalf("warning output = %q", output)
	}
}

func TestRunInputBoundsDescendantHoldingOutputPipes(t *testing.T) {
	installFakeGit(t)
	t.Setenv("MACHINERY_TESTGIT_MODE", "descendant-pipe")
	started := time.Now()
	_, err := runInputWithLimits(t.Context(), "", nil, 64, 64, "status")
	if err == nil {
		t.Fatal("descendant holding Git output pipes was reported as success")
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("descendant held Git output descriptors for %s: %v", elapsed, err)
	}
}
