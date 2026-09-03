//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package cachestage

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestRecoverRejectsSpecialEntryWithoutRemovingStage(t *testing.T) {
	base := t.TempDir()
	stage := filepath.Join(base, ".java-stage-123")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	pipe := filepath.Join(stage, "partial-download-pipe")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Recover(base, ".java-stage-"); err == nil || !strings.Contains(err.Error(), "special entry") {
		t.Fatalf("special crash residue accepted: %v", err)
	}
	if info, err := os.Lstat(pipe); err != nil || info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("fail-closed validation mutated special residue: %v, %v", info, err)
	}
}
