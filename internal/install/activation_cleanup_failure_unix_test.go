//go:build !windows

package install

import (
	"os"
	"testing"
)

func createInvalidActivationStageForTest(t *testing.T) bool {
	t.Helper()
	activation, err := activationStagingPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("invalid-activation", activation); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	return true
}
