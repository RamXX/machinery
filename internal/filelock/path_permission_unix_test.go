//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package filelock

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestScopeIdentityPropagatesFilesystemPermissionFailure(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "restricted")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0); err != nil {
		t.Skipf("cannot create permission fixture: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })
	_, err := ScopeIdentity(filepath.Join(parent, "missing"))
	if err == nil {
		t.Skip("current credentials bypass directory permissions")
	}
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("filesystem permission failure was not propagated: %v", err)
	}
}
