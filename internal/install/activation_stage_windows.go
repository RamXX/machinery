//go:build windows

package install

import "os"

func cleanupActivationExecutable() error { return nil }

func stageActivationExecutable(restored string, _ *os.File, _ string) (string, error) {
	// openActivationExecutable retains a handle without delete/write sharing.
	// CreateProcess maps this exact pathname before that handle is released.
	return restored, nil
}

func validateActivationExecutablePath(*ActivationRecoveryError) error { return nil }
