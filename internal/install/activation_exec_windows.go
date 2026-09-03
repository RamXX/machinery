//go:build windows

package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func reexecActivationCapability(recovery *ActivationRecoveryError, args, env []string, stdin, stdout, stderr *os.File) (int, error) {
	var childArgs []string
	if len(args) > 1 {
		childArgs = args[1:]
	}
	command := exec.CommandContext(context.Background(), recovery.Executable, childArgs...)
	command.Env = env
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	// openActivationExecutable denies delete/write sharing. CreateProcess must
	// map this same pathname while the exact verified handle and operation lock
	// remain held; only then may the parent release them and let child startup
	// acquire the consistency barrier.
	if err := command.Start(); err != nil {
		return -1, errors.Join(fmt.Errorf("start restored machinery image: %w", err), recovery.Close())
	}
	if err := recovery.Close(); err != nil {
		_ = command.Process.Kill()
		return -1, fmt.Errorf("release activation capability after child start: %w", err)
	}
	err := command.Wait()
	if err == nil {
		return 0, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	return -1, err
}
