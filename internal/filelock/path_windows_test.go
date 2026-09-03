//go:build windows

package filelock

import (
	"os"
	"testing"
)

func TestWindowsLockDirectoryDoesNotRequireSyntheticPOSIXMode(t *testing.T) {
	info, err := os.Lstat(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := validateLockDir("test-cache", info); err != nil {
		t.Fatalf("real Windows directory rejected by POSIX permission semantics: %v", err)
	}
}
