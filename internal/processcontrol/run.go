// Package processcontrol runs subprocesses with cancellation scoped to their
// complete process tree rather than only the direct child.
package processcontrol

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// DefaultWaitDelay bounds how long a finished direct child may leave output
// pipes held open by descendants before the complete process tree is closed.
const DefaultWaitDelay = 2 * time.Second

const cancellationReapLimit = DefaultWaitDelay + time.Second

var (
	attachProcessTree = func(control *treeControl, cmd *exec.Cmd) error {
		return control.attach(cmd)
	}
	terminateProcessTree = func(control *treeControl, cmd *exec.Cmd) error {
		return control.terminate(cmd)
	}
	killDirectProcess = func(cmd *exec.Cmd) error {
		if cmd == nil || cmd.Process == nil {
			return nil
		}
		err := cmd.Process.Kill()
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return err
	}
	closeProcessTree = func(control *treeControl) error {
		return control.close()
	}
)

func Run(ctx context.Context, cmd *exec.Cmd) error {
	if ctx == nil {
		return fmt.Errorf("run subprocess: nil context")
	}
	// CommandContext satisfies static call-site guarantees, while tree
	// cancellation remains owned here. Its direct-child Cancel would otherwise
	// race the process-group/job termination below and strand descendants.
	if cmd.Cancel != nil {
		cmd.Cancel = func() error { return nil }
	}
	if cmd.WaitDelay <= 0 || cmd.WaitDelay > DefaultWaitDelay {
		cmd.WaitDelay = DefaultWaitDelay
	}
	control, err := prepare(cmd)
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return errors.Join(err, closeProcessTree(control))
	}
	if err := attachProcessTree(control, cmd); err != nil {
		// A Windows child starts suspended and is not yet owned by the job
		// object when opening or assigning it fails. Terminating the still-empty
		// job cannot release cmd.Wait in that case, so always terminate the
		// direct child as an independent cleanup authority. Keep both attempts:
		// attach can also fail after assignment, when the job remains responsible
		// for any descendants.
		wait := waitForCommand(cmd)
		waitErr, terminateErr, directKillErr, closeErr, reapErr := terminateAndReap(control, cmd, wait)
		return errors.Join(err, waitErr, terminateErr, directKillErr, closeErr, reapErr)
	}
	wait := waitForCommand(cmd)
	select {
	case err := <-wait:
		return errors.Join(err, closeProcessTree(control))
	case <-ctx.Done():
		// Tree termination and direct-child termination are independent
		// authorities. Always attempt both: a failed job/process-group kill must
		// not leave the direct child alive, because WaitDelay does not begin
		// until that child exits. Closing the tree controller is a third attempt
		// (job close with kill-on-close on Windows, process-group kill on Unix)
		// and also releases every native handle before Run returns.
		waitErr, terminateErr, directKillErr, closeErr, reapErr := terminateAndReap(control, cmd, wait)
		return errors.Join(ctx.Err(), waitErr, terminateErr, directKillErr, closeErr, reapErr)
	}
}

func waitForCommand(cmd *exec.Cmd) <-chan error {
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	return wait
}

func terminateAndReap(control *treeControl, cmd *exec.Cmd, wait <-chan error) (waitErr, terminateErr, directKillErr, closeErr, reapErr error) {
	terminateErr = terminateProcessTree(control, cmd)
	directKillErr = killDirectProcess(cmd)
	closeErr = closeProcessTree(control)
	waitErr, reapErr = awaitCanceledProcess(wait, cmd)
	return waitErr, terminateErr, directKillErr, closeErr, reapErr
}

func awaitCanceledProcess(wait <-chan error, cmd *exec.Cmd) (waitErr, reapErr error) {
	timer := time.NewTimer(cancellationReapLimit)
	defer timer.Stop()
	select {
	case waitErr = <-wait:
		return waitErr, nil
	case <-timer.C:
	}

	// A successful direct kill should make Cmd.Wait return within WaitDelay,
	// even when descendants retain inherited descriptors. Retry independently
	// before the final bounded reap window so transient platform failures do
	// not leak the child.
	retryErr := killDirectProcess(cmd)
	retry := time.NewTimer(DefaultWaitDelay)
	defer retry.Stop()
	select {
	case waitErr = <-wait:
		return waitErr, errors.Join(
			fmt.Errorf("subprocess reap exceeded %s after cancellation", cancellationReapLimit),
			retryErr,
		)
	case <-retry.C:
		return nil, errors.Join(
			fmt.Errorf("subprocess did not reap within %s after cancellation", cancellationReapLimit+DefaultWaitDelay),
			retryErr,
		)
	}
}
