package install

import (
	"errors"
	"fmt"
	"os"
)

// afterActivationValidation is a deterministic adversarial test hook. It runs
// after the retained executable handle has been identity-verified and before
// the OS process-image activation primitive.
var afterActivationValidation func(string)

// ReexecActivationRecovery executes the exact retained restored image while
// the install operation lock is still held. The platform implementation uses
// the retained file descriptor/handle, so replacing the pathname after
// validation cannot redirect activation to different bytes.
func ReexecActivationRecovery(recoveryErr error, args, env []string, stdin, stdout, stderr *os.File) (int, error) {
	var recovery *ActivationRecoveryError
	if !errors.As(recoveryErr, &recovery) {
		return -1, fmt.Errorf("error is not an executable activation recovery signal")
	}
	if err := ValidateActivationRecovery(recoveryErr); err != nil {
		return -1, errors.Join(err, recovery.Close())
	}
	if afterActivationValidation != nil {
		afterActivationValidation(recovery.activationPath)
	}
	// Detect an injected/noncooperative swap of the private activation image at
	// the last boundary available on platforms (notably Darwin) without
	// fexecve/execveat. The operation lock remains held through the subsequent
	// Exec/CreateProcess call; same-user hostile processes are outside the
	// cooperative installer threat model and can subvert the process itself.
	if err := ValidateActivationRecovery(recoveryErr); err != nil {
		return -1, errors.Join(fmt.Errorf("activation image changed after final validation: %w", err), recovery.Close())
	}
	env = setActivationEnvironment(env, activationCanonicalExecutableEnv, recovery.Executable)
	return reexecActivationCapability(recovery, args, env, stdin, stdout, stderr)
}

func setActivationEnvironment(env []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix {
			continue
		}
		result = append(result, entry)
	}
	return append(result, prefix+value)
}
