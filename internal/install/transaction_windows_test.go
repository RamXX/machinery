//go:build windows

package install

import (
	"os"
	"path/filepath"
	"testing"
)

// This test is intentionally native-only: it proves that retained-root
// removal/copy and ReOpenFile+FlushFileBuffers directory durability work in a
// prepared transaction, then that the same rooted path recovers after the
// process-equivalent handle loss.
func TestWindowsRootedTransactionMutationAndRecovery(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	target := filepath.Join(t.TempDir(), "artifact")
	source := filepath.Join(t.TempDir(), "replacement")
	write(t, target, "old")
	write(t, source, "new")
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := beginArtifactTransaction([]string{target})
	if err != nil {
		t.Fatal(err)
	}
	if err := durableRemoveAll(target); err != nil {
		t.Fatal(err)
	}
	if err := copyEntryNoFollow(source, target); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "new" {
		t.Fatalf("mutated target = %q, %v", got, err)
	}
	if err := tx.closeAnchors(); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	recovered, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.Release()
	if got, err := os.ReadFile(target); err != nil || string(got) != "old" {
		t.Fatalf("recovered target = %q, %v", got, err)
	}
}
