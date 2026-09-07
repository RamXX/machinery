//go:build !unix

package processscope

import (
	"net"
	"os"
	"syscall"
)

func platformSupported() bool { return false }

func isSocketFD(fd int) bool { return false }

func isPipeFD(fd int) bool { return false }

func signalGroup(pgid int, sig int) error {
	return errf(CodeUnsupportedPlatform, "signalGroup", "platform has no native guardian support")
}

func groupExists(pgid int) bool { return false }

func signalProbe(pid int) bool { return false }

func reapPID(pid int) (bool, error) {
	return false, errf(CodeUnsupportedPlatform, "reapPID", "platform has no native guardian support")
}

func newChannelPair() (a, b *os.File, err error) {
	return nil, nil, errf(CodeUnsupportedPlatform, "channel", "platform has no native guardian support")
}

func connFromFile(f *os.File) (*net.UnixConn, error) {
	return nil, errf(CodeUnsupportedPlatform, "channel", "platform has no native guardian support")
}

func writeFrame(c *net.UnixConn, payload []byte, files ...*os.File) error {
	return errf(CodeUnsupportedPlatform, "frame", "platform has no native guardian support")
}

func readFrame(c *net.UnixConn) (payload []byte, files []*os.File, err error) {
	return nil, nil, errf(CodeUnsupportedPlatform, "frame", "platform has no native guardian support")
}

func newGroupAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

func sigKillCode() int { return 9 }

func runGuardian(io InternalIO, args []string) int {
	return 1
}
