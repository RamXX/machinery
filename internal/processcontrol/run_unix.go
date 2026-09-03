//go:build !windows

package processcontrol

import (
	"errors"
	"os/exec"
	"syscall"
)

type treeControl struct{ pgid int }

func prepare(cmd *exec.Cmd) (*treeControl, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return &treeControl{}, nil
}

func (c *treeControl) attach(cmd *exec.Cmd) error {
	c.pgid = cmd.Process.Pid
	return nil
}

func (c *treeControl) close() error {
	if c.pgid == 0 {
		return nil
	}
	err := syscall.Kill(-c.pgid, syscall.SIGKILL)
	c.pgid = 0
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

func (c *treeControl) terminate(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	pgid := c.pgid
	if pgid == 0 {
		pgid = cmd.Process.Pid
	}
	err := syscall.Kill(-pgid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
