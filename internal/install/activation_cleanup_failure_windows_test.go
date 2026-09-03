//go:build windows

package install

import "testing"

func createInvalidActivationStageForTest(t *testing.T) bool {
	t.Helper()
	return false
}
