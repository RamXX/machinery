// Package processcontrol runs subprocesses with cancellation scoped to their
// complete process tree rather than only the direct child.
package processcontrol

import (
	"context"
	"errors"
	"os/exec"
	"time"
)

// DefaultWaitDelay bounds how long a finished direct child may leave output
// pipes held open by descendants before the complete process tree is closed.
const DefaultWaitDelay = 2 * time.Second

func Run(ctx context.Context, cmd *exec.Cmd) error {
	// CommandContext satisfies static call-site guarantees, while tree
	// cancellation remains owned here. Its direct-child Cancel would otherwise
	// race the process-group/job termination below and strand descendants.
	if cmd.Cancel != nil {
		cmd.Cancel = func() error { return nil }
	}
	if cmd.WaitDelay == 0 {
		cmd.WaitDelay = DefaultWaitDelay
	}
	control, err := prepare(cmd)
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return errors.Join(err, control.close())
	}
	if err := control.attach(cmd); err != nil {
		// A Windows child starts suspended and is not yet owned by the job
		// object when opening or assigning it fails. Terminating the still-empty
		// job cannot release cmd.Wait in that case, so always terminate the
		// direct child as an independent cleanup authority. Keep both attempts:
		// attach can also fail after assignment, when the job remains responsible
		// for any descendants.
		directKillErr := cmd.Process.Kill()
		treeKillErr := control.terminate(cmd)
		waitErr := cmd.Wait()
		return errors.Join(err, directKillErr, treeKillErr, waitErr, control.close())
	}
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	select {
	case err := <-wait:
		return errors.Join(err, control.close())
	case <-ctx.Done():
		terminateErr := control.terminate(cmd)
		waitErr := <-wait
		return errors.Join(waitErr, terminateErr, control.close())
	}
}
