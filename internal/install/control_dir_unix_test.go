//go:build !windows

package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyInstallControlDirectoryMigrationIsOwnerVerified(t *testing.T) {
	config := privateConfigDir(t)
	t.Setenv("MACHINERY_CONFIG_DIR", config)
	if err := os.Chmod(config, 0o755); err != nil {
		t.Fatal(err)
	}
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(config); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("migrated control directory mode = %v, %v", info, err)
	}

	if err := os.Chmod(config, 0o755); err != nil {
		t.Fatal(err)
	}
	previous := installControlDirOwned
	installControlDirOwned = func(os.FileInfo) bool { return false }
	t.Cleanup(func() { installControlDirOwned = previous })
	if _, err := acquireInstallOperationLock(); err == nil || !strings.Contains(err.Error(), "not owned") {
		t.Fatalf("non-owner migration error = %v", err)
	}
	if info, err := os.Stat(config); err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("non-owner directory was mutated: %v, %v", info, err)
	}
}

func TestLegacyInstallControlDirectoryMigrationRejectsIdentitySwap(t *testing.T) {
	base := privateConfigDir(t)
	config := filepath.Join(base, "config")
	if err := os.Mkdir(config, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MACHINERY_CONFIG_DIR", config)
	outside := privateConfigDir(t)
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	parked := filepath.Join(base, "parked")
	previous := installControlDirOwned
	installControlDirOwned = func(os.FileInfo) bool {
		installControlDirOwned = previous
		if err := os.Rename(config, parked); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, config); err != nil {
			t.Fatal(err)
		}
		return true
	}
	t.Cleanup(func() {
		installControlDirOwned = previous
		_ = os.Remove(config)
		_ = os.Rename(parked, config)
	})
	if _, err := acquireInstallOperationLock(); err == nil || !strings.Contains(err.Error(), "changed during") {
		t.Fatalf("identity-swap migration error = %v", err)
	}
	if got, err := os.ReadFile(sentinel); err != nil || string(got) != "outside" {
		t.Fatalf("outside sentinel changed: %q, %v", got, err)
	}
}
