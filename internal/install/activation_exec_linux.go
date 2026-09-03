//go:build linux

package install

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"syscall"
)

// beforeActivationExec is used only by the Linux descriptor-bound subprocess
// proof to replace the staged pathname after final validation. Production
// leaves it nil.
var beforeActivationExec func(string)

func reexecActivationCapability(recovery *ActivationRecoveryError, args, env []string, _, _, _ *os.File) (int, error) {
	argv := []string{recovery.Executable}
	if len(args) > 1 {
		argv = append(argv, args[1:]...)
	}
	if beforeActivationExec != nil {
		beforeActivationExec(recovery.activationPath)
	}
	// /proc/self/fd/N is resolved by the kernel to the already-open verified
	// executable description. Replacing either the restored destination or the
	// private activation pathname after validation cannot redirect this exec to
	// a different inode. The operation lock remains held until successful exec
	// closes it through CLOEXEC (or recovery.Close releases it on failure).
	capabilityPath := "/proc/self/fd/" + strconv.FormatUint(uint64(recovery.file.Fd()), 10)
	if err := syscall.Exec(capabilityPath, argv, env); err != nil {
		return -1, errors.Join(fmt.Errorf("re-exec retained restored machinery image: %w", err), recovery.Close())
	}
	return -1, nil
}
