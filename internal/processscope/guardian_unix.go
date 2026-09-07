//go:build unix

package processscope

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func platformSupported() bool { return true }

func isSocketFD(fd int) bool {
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		return false
	}
	return st.Mode&syscall.S_IFMT == syscall.S_IFSOCK
}

func isPipeFD(fd int) bool {
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		return false
	}
	return st.Mode&syscall.S_IFMT == syscall.S_IFIFO
}

func signalGroup(pgid int, sig int) error {
	err := syscall.Kill(-pgid, syscall.Signal(sig))
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

func groupExists(pgid int) bool {
	return syscall.Kill(-pgid, 0) == nil
}

func signalProbe(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

func reapPID(pid int) (bool, error) {
	var ws syscall.WaitStatus
	wpid, err := syscall.Wait4(pid, &ws, syscall.WNOHANG, nil)
	if err != nil {
		if errors.Is(err, syscall.ECHILD) {
			return true, nil
		}
		if errors.Is(err, syscall.EINTR) {
			return false, nil
		}
		return false, err
	}
	return wpid == pid, nil
}

func selfDigest() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return fileDigest(exe)
}

func newGroupAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func sigKillCode() int { return int(syscall.SIGKILL) }

func newChannelPair() (a, b *os.File, err error) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, err
	}
	a = os.NewFile(uintptr(fds[0]), "processscope-channel-a")
	b = os.NewFile(uintptr(fds[1]), "processscope-channel-b")
	return a, b, nil
}

func connFromFile(f *os.File) (*net.UnixConn, error) {
	if f == nil {
		return nil, errf(CodeInternalError, "channel", "nil channel file")
	}
	c, err := net.FileConn(f)
	if err != nil {
		return nil, err
	}
	uc, ok := c.(*net.UnixConn)
	if !ok {
		c.Close()
		return nil, errf(CodeInternalError, "channel", "not a unix connection")
	}
	return uc, nil
}

func writeFrame(c *net.UnixConn, payload []byte, files ...*os.File) error {
	if uint64(len(payload)) > frameLimit {
		return errf(CodeInvalidSchema, "frame", "control record exceeds bound")
	}
	msg := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(msg[:4], uint32(len(payload)))
	copy(msg[4:], payload)
	var oob []byte
	if len(files) > 0 {
		fds := make([]int, 0, len(files))
		for _, f := range files {
			if f != nil {
				fds = append(fds, int(f.Fd()))
			}
		}
		oob = syscall.UnixRights(fds...)
	}
	n, _, err := c.WriteMsgUnix(msg, oob, nil)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return errf(CodeInternalError, "frame", "short control write")
	}
	return nil
}

type frameReader struct {
	c       *net.UnixConn
	buf     []byte
	pending []*os.File
}

func newFrameReader(c *net.UnixConn) *frameReader {
	return &frameReader{c: c}
}

func (fr *frameReader) read() ([]byte, []*os.File, error) {
	for {
		if len(fr.buf) >= 4 {
			n := binary.BigEndian.Uint32(fr.buf[:4])
			if uint64(n) > frameLimit {
				return nil, nil, errf(CodeInvalidSchema, "frame", "control record exceeds bound")
			}
			if uint32(len(fr.buf)-4) >= n {
				payload := append([]byte(nil), fr.buf[4:4+n]...)
				rest := append([]byte(nil), fr.buf[4+n:]...)
				fr.buf = rest
				files := fr.pending
				fr.pending = nil
				return payload, files, nil
			}
		}
		chunk := make([]byte, 65536)
		oob := make([]byte, 1024)
		rn, oobn, _, _, err := fr.c.ReadMsgUnix(chunk, oob)
		if rn > 0 {
			fr.buf = append(fr.buf, chunk[:rn]...)
		}
		if oobn > 0 {
			if cms, perr := syscall.ParseSocketControlMessage(oob[:oobn]); perr == nil {
				for _, cm := range cms {
					for i := 0; i+4 <= len(cm.Data); i += 4 {
						fd := int(binary.NativeEndian.Uint32(cm.Data[i : i+4]))
						fr.pending = append(fr.pending, os.NewFile(uintptr(fd), "processscope-fd-"+strconv.Itoa(os.Getpid())+"-"+strconv.Itoa(i/4)))
					}
				}
			}
		}
		if err != nil {
			return nil, nil, err
		}
	}
}

func readFrame(c *net.UnixConn) ([]byte, []*os.File, error) {
	return newFrameReader(c).read()
}

func exitStatus(err error) (completed bool, exitCode int, sig string) {
	if err == nil {
		return true, 0, ""
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		return false, -1, ""
	}
	ws, ok := ee.Sys().(syscall.WaitStatus)
	if !ok {
		return true, -1, ""
	}
	if ws.Exited() {
		return true, ws.ExitStatus(), ""
	}
	if ws.Signaled() {
		return true, -1, ws.Signal().String()
	}
	return true, -1, ""
}

func runGuardian(io InternalIO, args []string) int {
	if !io.consume() {
		return 1
	}
	st := io.state
	if st == nil || st.ctl == nil {
		return 1
	}
	conn, err := connFromFile(st.ctl)
	st.ctl.Close()
	if err != nil {
		return 2
	}
	defer conn.Close()
	fr := newFrameReader(conn)
	payload, files, err := fr.read()
	if err != nil {
		return 0
	}
	var jm msgJob
	if err := json.Unmarshal(payload, &jm); err != nil || jm.T != "job" {
		io.writeDiag("machinery: guardian received a malformed job record\n")
		return 2
	}
	spec := jm.Spec

	sigCh := make(chan os.Signal, 4)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	var devnull *os.File
	openNull := func(write bool) *os.File {
		if devnull != nil {
			return devnull
		}
		f, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
		if err != nil {
			return nil
		}
		devnull = f
		return f
	}
	defer func() {
		if devnull != nil {
			devnull.Close()
		}
	}()

	cmd := exec.Command(spec.Executable, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Env = spec.Env
	idx := 0
	if spec.FDStdin && idx < len(files) {
		cmd.Stdin = files[idx]
		idx++
	}
	if spec.FDStdout && idx < len(files) {
		cmd.Stdout = files[idx]
		idx++
	}
	if spec.FDStderr && idx < len(files) {
		cmd.Stderr = files[idx]
		idx++
	}
	if spec.FDCap && idx < len(files) {
		cmd.ExtraFiles = []*os.File{files[idx]}
		idx++
	}
	if cmd.Stdin == nil {
		cmd.Stdin = openNull(false)
	}
	if cmd.Stdout == nil {
		cmd.Stdout = openNull(true)
	}
	if cmd.Stderr == nil {
		cmd.Stderr = openNull(true)
	}
	for _, f := range files {
		defer f.Close()
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{}

	done := make(chan error, 1)
	reported := false
	grace := time.Duration(spec.DrainGraceMS) * time.Millisecond
	if grace <= 0 {
		grace = time.Duration(drainGraceMS) * time.Millisecond
	}
	report := func(terminal bool, err error) {
		completed, code, sig := exitStatus(err)
		_ = writeFrame(conn, mustMarshal(msgGExit{T: "gexit", Completed: completed, ExitCode: code, Signal: sig, Terminal: terminal}))
	}
	drain := func(terminal bool) {
		if reported {
			return
		}
		reported = true
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
		select {
		case err := <-done:
			report(terminal, err)
			return
		case <-time.After(grace):
		}
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		select {
		case err := <-done:
			report(terminal, err)
		case <-time.After(hardReapWindow):
			_ = writeFrame(conn, mustMarshal(msgGExit{T: "gexit", Completed: false, ExitCode: -1, Signal: "", Terminal: terminal}))
		}
	}

	if err := cmd.Start(); err != nil {
		_ = writeFrame(conn, mustMarshal(msgGStartFail{T: "gstartfail", Code: CodeInvalidCommand, Message: err.Error()}))
		for {
			if _, _, err := fr.read(); err != nil {
				return 0
			}
		}
	}
	for _, f := range files {
		f.Close()
	}
	_ = writeFrame(conn, mustMarshal(msgGStarted{T: "gstarted", Pid: cmd.Process.Pid}))
	go func() { done <- cmd.Wait() }()

	eof := make(chan struct{})
	go func() {
		for {
			if _, _, err := fr.read(); err != nil {
				close(eof)
				return
			}
		}
	}()

	for {
		select {
		case err := <-done:
			if !reported {
				reported = true
				report(false, err)
			}
			<-eof
			return 0
		case <-sigCh:
			drain(true)
			<-eof
			return 0
		case <-eof:
			if !reported {
				drain(true)
			}
			return 0
		}
	}
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"t":"gstartfail","code":"INTERNAL_ERROR","message":"marshal"}`)
	}
	return b
}
