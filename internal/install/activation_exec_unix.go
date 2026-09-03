//go:build !windows && !linux

package install

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func reexecActivationCapability(recovery *ActivationRecoveryError, args, env []string, _, _, _ *os.File) (int, error) {
	argv := []string{recovery.Executable}
	if len(args) > 1 {
		argv = append(argv, args[1:]...)
	}
	// Darwin and the remaining supported Unix targets do not expose a portable
	// fexecve/execveat primitive to pure Go. The private activation directory is
	// mode 0500 and the cooperative install operation lock remains held through
	// Exec. A same-UID process that deliberately chmods and rewrites that private
	// directory is outside this boundary; Linux and Windows use descriptor/
	// handle-bound activation and do not have this pathname gap.
	if err := syscall.Exec(recovery.activationPath, argv, env); err != nil {
		return -1, errors.Join(fmt.Errorf("re-exec restored machinery image: %w", err), recovery.Close())
	}
	return -1, nil
}
