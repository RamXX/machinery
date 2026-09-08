//go:build unix

package processscope

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
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
	// A stream socket accepts only what fits in its send buffer, so one send
	// of a record larger than that buffer is a short write, not a failure.
	// The record is written until it is complete; the descriptor rights ride
	// with the first send, because they belong to the record rather than to
	// any byte of it. Treating the short write as a failure truncated exactly
	// the records that grow with the work: a cleanup report naming every
	// retired job of a large run.
	// The rights carry raw descriptor numbers, which are only meaningful for
	// as long as the files that own them are alive: a file collected between
	// the number being read and the send that transfers it takes its
	// descriptor with it, and the send transfers a number that now names
	// something else or nothing at all.
	defer runtime.KeepAlive(files)
	for off := 0; off < len(msg); {
		n, _, err := c.WriteMsgUnix(msg[off:], oob, nil)
		if err != nil {
			return err
		}
		if n <= 0 {
			return errf(CodeInternalError, "frame", "control write made no progress")
		}
		off += n
		oob = nil
	}
	return nil
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
	openNull := func() *os.File {
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

	// The job's custody is owned by the guardian drain protocol (SIGTERM,
	// grace, SIGKILL with event reporting); no ambient context exists inside
	// the guardian, and the direct-child Cancel of CommandContext must never
	// race the drain, so the background context is used with Cancel neutered.
	cmd := exec.CommandContext(context.Background(), spec.Executable, spec.Args...)
	cmd.Cancel = func() error { return nil }
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
	}
	if cmd.Stdin == nil {
		cmd.Stdin = openNull()
	}
	if cmd.Stdout == nil {
		cmd.Stdout = openNull()
	}
	if cmd.Stderr == nil {
		cmd.Stderr = openNull()
	}
	defer func() {
		for i := len(files) - 1; i >= 0; i-- {
			// Best-effort close of inherited job descriptors at guardian exit.
			_ = files[i].Close()
		}
	}()
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
	origPPID := os.Getppid()
	orphaned := make(chan struct{})
	go func() {
		for {
			if os.Getppid() != origPPID {
				close(orphaned)
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()
	release := func() {
		select {
		case <-eof:
		case <-orphaned:
		}
	}

	for {
		select {
		case err := <-done:
			if !reported {
				reported = true
				report(false, err)
			}
			release()
			return 0
		case <-sigCh:
			drain(true)
			release()
			return 0
		case <-eof:
			if !reported {
				drain(true)
			}
			return 0
		case <-orphaned:
			if !reported {
				drain(true)
			}
			return 0
		}
	}
}
