//go:build unix

package processscope

import (
	"errors"
	"net"
	"os"
	"syscall"
)

func platformSupported() bool { return true }

func isSocketFD(fd int) bool { return false }

func isPipeFD(fd int) bool { return false }

func signalGroup(pgid int, sig int) error {
	return errf(CodeUnsupportedFeature, "signalGroup", "not implemented")
}

func groupExists(pgid int) bool { return false }

func signalProbe(pid int) bool { return false }

func reapPID(pid int) (bool, error) {
	return false, errf(CodeUnsupportedFeature, "reapPID", "not implemented")
}

func newGroupAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func newChannelPair() (a, b *os.File, err error) {
	return nil, nil, errf(CodeUnsupportedFeature, "channel", "not implemented")
}

func connFromFile(f *os.File) (*net.UnixConn, error) {
	return nil, errf(CodeUnsupportedFeature, "channel", "not implemented")
}

func writeFrame(c *net.UnixConn, payload []byte, files ...*os.File) error {
	return errf(CodeUnsupportedFeature, "frame", "not implemented")
}

func readFrame(c *net.UnixConn) (payload []byte, files []*os.File, err error) {
	return nil, nil, errf(CodeUnsupportedFeature, "frame", "not implemented")
}

func runGuardian(io InternalIO, args []string) int {
	return 1
}

var _ = errors.New
var _ = syscall.Kill
