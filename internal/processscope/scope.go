// Package processscope owns native process custody: the broker and guardian
// supervision protocol, wall-deadline enforcement, and the scoped execution
// surface consumed by processcontrol. Lifecycle authority lives here, never
// in incidental caller contexts.
package processscope

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Scope interface {
	Run(context.Context, Command, Streams) (Result, error)
	Child(context.Context) (Scope, error)
	Attach(Command) (Command, error)
	Close(context.Context) (CleanupReport, error)
}

type rootHandle struct {
	cmd       *exec.Cmd
	dir       string
	livenessW *os.File
	cleanup   sync.Once
}

type scope struct {
	mu       sync.Mutex
	conn     *net.UnixConn
	wmu      sync.Mutex
	fr       *frameReader
	id       string
	rootID   string
	deadline time.Time
	limits   Limits
	isRoot   bool
	root     *rootHandle
	closed   bool
	report   CleanupReport

	dmxOnce  sync.Once
	replyCh  chan envFiles
	eofCh    chan struct{}
	eofOnce  sync.Once
	jobsMu   sync.Mutex
	jobChans map[string]chan anyEnv
}

type envFiles struct {
	env   anyEnv
	files []*os.File
}

type anyEnv struct {
	T         string        `json:"t"`
	Job       string        `json:"job"`
	Pid       int           `json:"pid"`
	Root      string        `json:"root"`
	Ack       string        `json:"ack"`
	Self      string        `json:"self"`
	Nonce     string        `json:"nonce"`
	Code      string        `json:"code"`
	Subject   string        `json:"subject"`
	Message   string        `json:"message"`
	Started   bool          `json:"started"`
	Completed bool          `json:"completed"`
	ExitCode  int           `json:"exit"`
	Signal    string        `json:"signal"`
	Scope     string        `json:"scope"`
	Deadline  int64         `json:"deadline"`
	Report    CleanupReport `json:"report"`
}

func (s *scope) ensureDemux() {
	s.dmxOnce.Do(func() {
		s.replyCh = make(chan envFiles, 8)
		s.eofCh = make(chan struct{})
		s.jobChans = map[string]chan anyEnv{}
		go func() {
			for {
				payload, files, err := s.fr.read()
				if err != nil {
					s.eofOnce.Do(func() { close(s.eofCh) })
					return
				}
				var m anyEnv
				if err := json.Unmarshal(payload, &m); err != nil {
					continue
				}
				switch m.T {
				case "result":
					// The registration belongs to Run, which retires it when
					// it is done with the job. Deleting it here would let a
					// Run that has not yet reached jobChanFor register a
					// second, empty channel and then wait out its whole
					// deadline on it while the delivered result sat in the
					// abandoned one: a short job whose result overtakes its
					// own caller. The buffered channel holds the first
					// terminal result; a later duplicate (the retirement of an
					// already-reported job) finds the buffer full and is
					// dropped rather than blocking this reader, which every
					// other frame on the connection depends on.
					s.jobsMu.Lock()
					ch := s.jobChans[m.Job]
					s.jobsMu.Unlock()
					if ch != nil {
						select {
						case ch <- m:
						default:
						}
					}
				case "started":
					// Frames on one connection are read sequentially by this
					// goroutine, so registering the job's result channel here,
					// before the started reply is delivered to Run, guarantees
					// the job's terminal result frame, whenever it arrives, finds
					// its channel instead of being dropped as unreadable.
					s.jobsMu.Lock()
					if s.jobChans[m.Job] == nil {
						s.jobChans[m.Job] = make(chan anyEnv, 1)
					}
					s.jobsMu.Unlock()
					s.replyCh <- envFiles{env: m, files: files}
				case "refused", "attached", "childok", "closed", "welcome", "regcontainerok":
					s.replyCh <- envFiles{env: m, files: files}
				}
			}
		}()
	})
}

func (s *scope) send(v any, files ...*os.File) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	return writeFrame(s.conn, b, files...)
}

func (s *scope) request(ctx context.Context, v any, files ...*os.File) (envFiles, error) {
	s.ensureDemux()
	if err := s.send(v, files...); err != nil {
		return envFiles{}, err
	}
	wait := authWindow
	if dl, ok := ctx.Deadline(); ok {
		if d := time.Until(dl); d < wait {
			wait = d
		}
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case ew := <-s.replyCh:
		return replyOrRefusal(ew)
	case <-s.eofCh:
		// The broker answers the root close and then exits, so the end of the
		// control channel follows its own last reply by a few milliseconds.
		// The single demux goroutine delivers that reply to replyCh strictly
		// before it reads the end of the channel and closes eofCh, so once
		// eofCh is closed any reply the broker did send is already buffered:
		// draining it here is a settled read, not a second race. Without the
		// drain, a caller that had not yet reached this select when both
		// became ready got whichever case the runtime picked, and half the
		// time it reported a delivered close report as a stale capability.
		select {
		case ew := <-s.replyCh:
			return replyOrRefusal(ew)
		default:
		}
		return envFiles{}, errf(CodeStaleCapability, s.id, "broker channel closed")
	case <-ctx.Done():
		return envFiles{}, errFromContext(ctx, s.id)
	case <-timer.C:
		return envFiles{}, errf(CodeTimeout, s.id, "control channel timed out")
	}
}

func replyOrRefusal(ew envFiles) (envFiles, error) {
	if ew.env.T == "refused" {
		return envFiles{}, &Error{Code: ew.env.Code, Subject: ew.env.Subject, Message: ew.env.Message}
	}
	return ew, nil
}

func (s *scope) jobChanFor(id string) chan anyEnv {
	ch := make(chan anyEnv, 1)
	s.jobsMu.Lock()
	if cur := s.jobChans[id]; cur != nil {
		s.jobsMu.Unlock()
		return cur
	}
	s.jobChans[id] = ch
	s.jobsMu.Unlock()
	return ch
}

func (s *scope) dropJobChan(id string, ch chan anyEnv) {
	s.jobsMu.Lock()
	if cur := s.jobChans[id]; cur == ch {
		delete(s.jobChans, id)
	}
	s.jobsMu.Unlock()
}

func (s *scope) checkUsable() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errf(CodeScopeClosed, s.id, "scope is closed")
	}
	if !time.Now().Before(s.deadline) {
		return errf(CodeBudgetExhausted, s.id, "inherited wall deadline has passed")
	}
	return nil
}

func (s *scope) dir() string {
	if s.root != nil {
		return s.root.dir
	}
	return ""
}

func (s *scope) Attach(cmd Command) (Command, error) {
	if err := s.checkUsable(); err != nil {
		return Command{}, err
	}
	if !filepath.IsAbs(cmd.Executable) {
		return Command{}, errf(CodeInvalidCommand, "attach", "executable must be an absolute path")
	}
	st, err := os.Stat(cmd.Executable)
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0111 == 0 {
		return Command{}, errf(CodeInvalidCommand, "attach", "executable must be an absolute regular executable file")
	}
	if cmd.RuntimeDigest == "" {
		return Command{}, errf(CodeInvalidCommand, "attach", "runtime identity digest is required")
	}
	if cmd.Env == nil {
		return Command{}, errf(CodeInvalidCommand, "attach", "environment must be explicitly declared")
	}
	if cmd.Dir != "" && !filepath.IsAbs(cmd.Dir) {
		return Command{}, errf(CodeInvalidCommand, "attach", "working directory must be absolute")
	}
	n := 0
	for _, e := range cmd.Env {
		if e == EnvChildRequest+"=join" {
			n++
		}
	}
	if n > 1 {
		return Command{}, errf(CodeInvalidCommand, "attach", "duplicate capability request entries")
	}
	remaining := RemainingWallMS(time.Now(), s.deadline)
	if cmd.DeadlineMS > 0 && cmd.DeadlineMS < remaining {
		remaining = cmd.DeadlineMS
	}
	if remaining <= 0 {
		return Command{}, errf(CodeBudgetExhausted, s.id, "remaining wall budget is not positive")
	}
	digest := commandDigest(cmd)
	deadlineNS := time.Now().Add(time.Duration(remaining) * time.Millisecond).UnixNano()
	ctx, cancel := context.WithTimeout(context.Background(), authWindow)
	defer cancel()
	ew, err := s.request(ctx, msgAttach{T: "attach", Scope: s.id, Digest: digest, Deadline: deadlineNS, Cap: n == 1})
	if err != nil {
		return Command{}, err
	}
	out := cmd
	out.DeadlineMS = remaining
	out.Env = append(append([]string{}, stripReservedEnv(cmd.Env)...), EnvAttachment+"="+encodeAttachment(attachment{Nonce: ew.env.Nonce, Digest: digest, Deadline: deadlineNS, Cap: n == 1}))
	return out, nil
}

func (s *scope) Run(ctx context.Context, cmd Command, streams Streams) (Result, error) {
	if ctx == nil {
		return Result{}, errf(CodeInvalidSchema, "run", "nil context")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, errFromContext(ctx, "run")
	}
	if err := s.checkUsable(); err != nil {
		return Result{}, err
	}
	var att attachment
	found := 0
	for _, e := range cmd.Env {
		if v, ok := stringsCutPrefix(e, EnvAttachment+"="); ok {
			found++
			var ok bool
			att, ok = parseAttachment(v)
			if !ok {
				return Result{}, errf(CodeInvalidCommand, "run", "malformed reserved attachment entry")
			}
		}
	}
	if found != 1 {
		return Result{}, errf(CodeInvalidCommand, "run", "exactly one reserved attachment entry is required")
	}
	if commandDigest(cmd) != att.Digest {
		return Result{}, errf(CodeInvalidCommand, "run", "command no longer matches its attachment")
	}
	now := time.Now()
	if now.UnixNano() >= att.Deadline {
		return Result{}, errf(CodeBudgetExhausted, "run", "attachment deadline expired")
	}
	eff := EffectiveDeadlineMS(now, 0, ctx, s.deadline)
	if d := (att.Deadline - now.UnixNano()) / int64(time.Millisecond); d < eff {
		eff = d
	}
	if eff <= 0 {
		return Result{}, errf(CodeBudgetExhausted, "run", "remaining budget is not positive")
	}
	stdoutLimit := streams.StdoutLimit
	if stdoutLimit == 0 {
		stdoutLimit = s.limits.StdoutBytes
	}
	stderrLimit := streams.StderrLimit
	if stderrLimit == 0 {
		stderrLimit = s.limits.StderrBytes
	}
	if stdoutLimit < 0 || stderrLimit < 0 {
		return Result{}, errf(CodeInvalidCommand, "run", "stream limits must be nonnegative")
	}

	spec := jobSpec{
		Executable:    cmd.Executable,
		Args:          cmd.Args,
		Dir:           cmd.Dir,
		Env:           stripReservedEnv(cmd.Env),
		RuntimeDigest: cmd.RuntimeDigest,
		DeadlineMS:    eff,
	}
	var sendFiles []*os.File
	var stdinW, stdoutR, stderrR *os.File
	if streams.Stdin != nil {
		r, w, err := os.Pipe()
		if err != nil {
			return Result{}, errf(CodeInternalError, "run", "stdin pipe: %v", err)
		}
		stdinW = w
		spec.FDStdin = true
		sendFiles = append(sendFiles, r)
	}
	if streams.Stdout != nil {
		r, w, err := os.Pipe()
		if err != nil {
			return Result{}, errf(CodeInternalError, "run", "stdout pipe: %v", err)
		}
		stdoutR = r
		spec.FDStdout = true
		sendFiles = append(sendFiles, w)
	}
	if streams.Stderr != nil {
		r, w, err := os.Pipe()
		if err != nil {
			return Result{}, errf(CodeInternalError, "run", "stderr pipe: %v", err)
		}
		stderrR = r
		spec.FDStderr = true
		sendFiles = append(sendFiles, w)
	}
	ew, err := s.request(ctx, msgRun{T: "run", Scope: s.id, Nonce: att.Nonce, Digest: att.Digest, Spec: spec}, sendFiles...)
	for _, f := range sendFiles {
		f.Close()
	}
	if err != nil {
		return Result{}, err
	}
	if ew.env.T != "started" {
		return Result{}, errf(CodeInternalError, "run", "unexpected control reply %q", ew.env.T)
	}
	jobID := ew.env.Job
	// The demux registered this job's channel when it read the started frame
	// and holds the registration until this call retires it, so a result that
	// arrives before this line lands in the channel returned here.
	jobCh := s.jobChanFor(jobID)
	defer s.dropJobChan(jobID, jobCh)

	cancelCh := make(chan string, 1)
	var cancelOnce sync.Once
	trigger := func(code, reason string) {
		cancelOnce.Do(func() {
			cancelCh <- code
			_ = s.send(msgCancel{T: "cancel", Scope: s.id, Job: jobID, Reason: reason})
		})
	}

	var pumpWG sync.WaitGroup
	if stdinW != nil {
		pumpWG.Add(1)
		go func() {
			defer pumpWG.Done()
			_, _ = io.Copy(stdinW, streams.Stdin)
			stdinW.Close()
		}()
	}
	pump := func(r *os.File, w io.Writer, limit int64) {
		if r == nil {
			return
		}
		pumpWG.Add(1)
		go func() {
			defer pumpWG.Done()
			buf := make([]byte, 32*1024)
			count := int64(0)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					remain := limit - count
					if remain <= 0 {
						trigger(CodeOutputLimit, "overflow")
						return
					}
					if int64(n) > remain {
						n = int(remain)
					}
					if _, werr := w.Write(buf[:n]); werr != nil {
						return
					}
					count += int64(n)
					if count >= limit {
						trigger(CodeOutputLimit, "overflow")
						return
					}
				}
				if err != nil {
					return
				}
			}
		}()
	}
	pump(stdoutR, streams.Stdout, stdoutLimit)
	pump(stderrR, streams.Stderr, stderrLimit)

	timer := time.NewTimer(time.Duration(eff) * time.Millisecond)
	defer timer.Stop()
	watchDone := make(chan struct{})
	watchStopped := make(chan struct{})
	go func() {
		defer close(watchStopped)
		select {
		case <-ctx.Done():
			code := errFromContext(ctx, "run").Code
			reason := "cancel"
			if code == CodeTimeout {
				reason = "timeout"
			}
			trigger(code, reason)
		case <-timer.C:
			trigger(CodeTimeout, "timeout")
		case <-watchDone:
		}
	}()

	// The give-up bound is budget-derived per the custody contract: the job's
	// own effective wall deadline, plus the cleanup grace the broker may use
	// to terminate and reap after that deadline, plus one fixed operational
	// slack for result delivery. A fixed window measured from wait entry
	// would abandon a still-running long job (a job may legitimately run for
	// its whole deadline; eff is already bounded by the scope wall cap).
	hardLimit := time.Duration(eff+s.limits.CleanupMS+10000) * time.Millisecond
	var res anyEnv
	select {
	case res = <-jobCh:
	case <-time.After(hardLimit):
		close(watchDone)
		<-watchStopped
		forceClosePipes(stdinW, stdoutR, stderrR)
		return Result{JobID: jobID, Started: true}, errf(CodeCustodyError, jobID, "broker did not deliver a terminal result")
	}
	close(watchDone)
	<-watchStopped
	drainDone := make(chan struct{})
	go func() {
		pumpWG.Wait()
		close(drainDone)
	}()
	select {
	case <-drainDone:
	case <-time.After(2 * time.Second):
		forceClosePipes(stdinW, stdoutR, stderrR)
	}
	result := Result{JobID: jobID, Started: res.Started, Completed: res.Completed, ExitCode: res.ExitCode, Signal: res.Signal, Cleanup: CleanupReport{Status: StatusCleaned}}
	var rerr error
	if res.Code != "" {
		rerr = &Error{Code: res.Code, Subject: jobID, Message: "job terminated by scope cancellation"}
	}
	return result, rerr
}

func forceClosePipes(files ...*os.File) {
	for _, f := range files {
		if f != nil {
			f.Close()
		}
	}
}

func stringsCutPrefix(s, p string) (string, bool) {
	if len(s) >= len(p) && s[:len(p)] == p {
		return s[len(p):], true
	}
	return "", false
}

func (s *scope) Child(ctx context.Context) (Scope, error) {
	if ctx == nil {
		return nil, errf(CodeInvalidSchema, "child", "nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, errFromContext(ctx, "child")
	}
	if err := s.checkUsable(); err != nil {
		return nil, err
	}
	ew, err := s.request(ctx, msgChildReq{T: "childreq", Scope: s.id})
	if err != nil {
		return nil, err
	}
	if len(ew.files) < 1 || ew.files[0] == nil {
		return nil, errf(CodeInternalError, "child", "broker did not deliver a child channel")
	}
	conn, err := connFromFile(ew.files[0])
	ew.files[0].Close()
	if err != nil {
		return nil, errf(CodeInternalError, "child", "child channel is not a unix connection")
	}
	return &scope{conn: conn, fr: newFrameReader(conn), id: ew.env.Scope, rootID: s.rootID, deadline: s.deadline, limits: s.limits}, nil
}

func (s *scope) registerContainer(ctx context.Context, id, daemon string) error {
	if ctx == nil {
		return errf(CodeInvalidSchema, "registerContainer", "nil context")
	}
	if err := ctx.Err(); err != nil {
		return errFromContext(ctx, "registerContainer")
	}
	if err := s.checkUsable(); err != nil {
		return err
	}
	_, err := s.request(ctx, msgRegContainer{T: "regcontainer", Scope: s.id, ID: id, Daemon: daemon})
	return err
}

func (s *scope) Close(ctx context.Context) (CleanupReport, error) {
	if ctx == nil {
		return CleanupReport{}, errf(CodeInvalidSchema, "close", "nil context")
	}
	s.mu.Lock()
	if s.closed {
		rep := s.report
		s.mu.Unlock()
		return rep, nil
	}
	s.closed = true
	s.mu.Unlock()

	ew, err := s.request(ctx, msgClose{T: "close", Scope: s.id})
	rep := CleanupReport{Status: StatusCleanupFailed, Diagnostics: []Diagnostic{{Code: CodeCustodyError, Subject: s.id, Message: "close did not complete"}}}
	if err == nil && ew.env.T == "closed" {
		rep = ew.env.Report
	}
	if s.isRoot && s.root != nil {
		s.root.cleanup.Do(func() {
			brokerExit := time.NewTimer(time.Duration(s.limits.CleanupMS)*time.Millisecond + 5*time.Second)
			defer brokerExit.Stop()
			select {
			case <-s.eofCh:
			case <-brokerExit.C:
			}
			if s.root.cmd != nil && s.root.cmd.Process != nil {
				done := make(chan error, 1)
				go func() { done <- s.root.cmd.Wait() }()
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					_ = signalGroup(s.root.cmd.Process.Pid, sigKillCode())
					<-done
					rep.Diagnostics = append(rep.Diagnostics, Diagnostic{Code: CodeCustodyError, Subject: "broker", Message: "broker had to be force-killed after close"})
				}
			}
			if rep.Status == StatusCleaned && s.root.dir != "" {
				_ = os.RemoveAll(s.root.dir)
			}
			if s.root.livenessW != nil {
				s.root.livenessW.Close()
			}
		})
	}
	s.mu.Lock()
	s.report = rep
	s.mu.Unlock()
	return rep, err
}

func Open(ctx context.Context, opts Options) (Scope, error) {
	if ctx == nil {
		return nil, errf(CodeInvalidSchema, "open", "nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, errFromContext(ctx, "open")
	}
	if !platformSupported() {
		return nil, errf(CodeUnsupportedPlatform, "open", "no native custody support on this platform")
	}
	if !filepath.IsAbs(opts.HelperExecutable) {
		return nil, errf(CodeInvalidOptions, "open", "helper executable must be an absolute path")
	}
	st, err := os.Stat(opts.HelperExecutable)
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0111 == 0 {
		return nil, errf(CodeInvalidOptions, "open", "helper executable must be an absolute regular executable file")
	}
	if !isHex64(opts.HelperDigest) {
		return nil, errf(CodeInvalidOptions, "open", "helper digest must be 64 lowercase hex digits")
	}
	actual, err := fileDigest(opts.HelperExecutable)
	if err != nil || actual != opts.HelperDigest {
		return nil, errf(CodeInvalidOptions, "open", "helper executable digest does not match the declared closure identity")
	}
	if opts.ScratchRoot == "" || !filepath.IsAbs(opts.ScratchRoot) {
		return nil, errf(CodeInvalidOptions, "open", "scratch root must be an absolute path")
	}
	if err := os.MkdirAll(opts.ScratchRoot, 0700); err != nil {
		return nil, errf(CodeInvalidOptions, "open", "scratch root: %v", err)
	}
	dir, err := os.MkdirTemp(opts.ScratchRoot, "processscope-")
	if err != nil {
		return nil, errf(CodeInternalError, "open", "private authority directory: %v", err)
	}
	if err := os.Chmod(dir, 0700); err != nil {
		os.RemoveAll(dir)
		return nil, errf(CodeInternalError, "open", "private authority directory mode: %v", err)
	}
	limits, err := NormalizeLimits(opts.Limits)
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	wall := time.Now().Add(time.Duration(limits.WallMS) * time.Millisecond)
	if d, ok := ctx.Deadline(); ok && d.Before(wall) {
		wall = d
	}
	if !wall.After(time.Now()) {
		os.RemoveAll(dir)
		return nil, errf(CodeBudgetExhausted, "open", "remaining budget is not positive")
	}

	ca, cb, err := newChannelPair()
	if err != nil {
		os.RemoveAll(dir)
		return nil, errf(CodeInternalError, "open", "control channel: %v", err)
	}
	lr, lw, err := os.Pipe()
	if err != nil {
		ca.Close()
		cb.Close()
		os.RemoveAll(dir)
		return nil, errf(CodeInternalError, "open", "owner liveness channel: %v", err)
	}
	logf, lerr := os.OpenFile(filepath.Join(dir, "broker.log"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if lerr != nil {
		logf = nil
	}
	brokerEnv := []string{
		EnvInternalCtl + "=3",
		EnvInternalOwner + "=4",
		EnvInternalDeadline + "=" + fmt.Sprintf("%d", wall.UnixNano()),
	}
	extra := []*os.File{cb, lr}
	if opts.OwnerLiveness != nil {
		brokerEnv = append(brokerEnv, EnvInternalParent+"=5")
		extra = append(extra, opts.OwnerLiveness)
	}
	// CommandContext satisfies static call-site guarantees; broker custody is
	// owned by the wall-deadline protocol above, so its direct-child Cancel is
	// neutered exactly like processcontrol.Run does for scoped launches.
	cmd := exec.CommandContext(ctx, opts.HelperExecutable, InternalMarker, "broker", "--dir", dir)
	cmd.Cancel = func() error { return nil }
	cmd.Env = brokerEnv
	cmd.ExtraFiles = extra
	cmd.SysProcAttr = newGroupAttr()
	if logf != nil {
		cmd.Stdout = logf
		cmd.Stderr = logf
	}
	if err := cmd.Start(); err != nil {
		ca.Close()
		cb.Close()
		lr.Close()
		lw.Close()
		if logf != nil {
			logf.Close()
		}
		os.RemoveAll(dir)
		return nil, errf(CodeInternalError, "open", "broker launch failed: %v", err)
	}
	cb.Close()
	lr.Close()
	if logf != nil {
		logf.Close()
	}
	conn, err := connFromFile(ca)
	ca.Close()
	if err != nil {
		abortBootstrap(cmd, lw)
		os.RemoveAll(dir)
		return nil, errf(CodeInternalError, "open", "control channel: %v", err)
	}
	challenge, err := randomHex(32)
	if err != nil {
		conn.Close()
		abortBootstrap(cmd, lw)
		os.RemoveAll(dir)
		return nil, errf(CodeInternalError, "open", "challenge: %v", err)
	}
	if err := writeFrame(conn, mustMarshal(msgBootstrap{T: "bootstrap", Challenge: challenge, Digest: opts.HelperDigest, Deadline: wall.UnixNano(), Limits: limits})); err != nil {
		conn.Close()
		abortBootstrap(cmd, lw)
		os.RemoveAll(dir)
		return nil, errf(CodeInternalError, "open", "bootstrap send: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(authWindow))
	payload, _, err := readFrame(conn)
	_ = conn.SetReadDeadline(time.Time{})
	if err != nil {
		conn.Close()
		abortBootstrap(cmd, lw)
		os.RemoveAll(dir)
		return nil, errf(CodeInternalError, "open", "bootstrap handshake failed: %v", err)
	}
	var ready msgReady
	if err := json.Unmarshal(payload, &ready); err != nil || ready.T != "ready" || ready.Ack != challenge || ready.Self != opts.HelperDigest {
		conn.Close()
		abortBootstrap(cmd, lw)
		os.RemoveAll(dir)
		return nil, errf(CodeAuthFailed, "open", "broker failed the authenticated bootstrap exchange")
	}
	s := &scope{conn: conn, fr: newFrameReader(conn), id: "root", rootID: ready.Root, deadline: wall, limits: limits, isRoot: true, root: &rootHandle{cmd: cmd, dir: dir, livenessW: lw}}
	return s, nil
}

func abortBootstrap(cmd *exec.Cmd, lw *os.File) {
	if lw != nil {
		lw.Close()
	}
	if cmd != nil && cmd.Process != nil {
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
			return
		case <-time.After(3 * time.Second):
		}
		_ = signalGroup(cmd.Process.Pid, sigKillCode())
		<-done
	}
}

func Join(ctx context.Context, cap Capability) (Scope, error) {
	if ctx == nil {
		return nil, errf(CodeInvalidSchema, "join", "nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, errFromContext(ctx, "join")
	}
	if cap.state == nil {
		return nil, errf(CodeInvalidSchema, "join", "capability is absent")
	}
	if !cap.consume() {
		return nil, errf(CodeStaleCapability, "join", "capability is closed or already consumed")
	}
	conn, err := connFromFile(cap.state.conn)
	cap.state.conn.Close()
	if err != nil {
		return nil, errf(CodeInternalError, "join", "capability transport is not a unix connection")
	}
	deadline := cap.state.deadline
	scopeID := cap.state.scopeID
	if err := writeFrame(conn, mustMarshal(msgHello{T: "hello", Scope: scopeID, Role: RoleJoined})); err != nil {
		conn.Close()
		return nil, errf(CodeStaleCapability, "join", "live broker handshake failed: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(authWindow))
	payload, _, err := readFrame(conn)
	_ = conn.SetReadDeadline(time.Time{})
	if err != nil {
		conn.Close()
		return nil, errf(CodeStaleCapability, "join", "live broker handshake failed: %v", err)
	}
	var head msgRefused
	if err := json.Unmarshal(payload, &head); err == nil && head.T == "refused" {
		conn.Close()
		return nil, &Error{Code: head.Code, Subject: "join", Message: head.Message}
	}
	var welcome msgWelcome
	if err := json.Unmarshal(payload, &welcome); err != nil || welcome.T != "welcome" || welcome.Root == "" {
		conn.Close()
		return nil, errf(CodeAuthFailed, "join", "broker did not complete the verified attachment handshake")
	}
	if welcome.Deadline > 0 {
		if d := time.Unix(0, welcome.Deadline); d.Before(deadline) {
			deadline = d
		}
	}
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	return &scope{conn: conn, fr: newFrameReader(conn), id: scopeID, rootID: welcome.Root, deadline: deadline, limits: Limits{CleanupMS: limitsDefaults.CleanupMS, StdoutBytes: limitsDefaults.StdoutBytes, StderrBytes: limitsDefaults.StderrBytes}}, nil
}

func ServeInternal(args []string, io InternalIO) (handled bool, exitCode int) {
	if len(args) == 0 || args[0] != InternalMarker {
		return false, 0
	}
	if !platformSupported() {
		io.writeDiag("machinery: internal execution is unsupported on this platform\n")
		return true, 1
	}
	if io.state == nil {
		return true, 1
	}
	role := ""
	if len(args) > 1 {
		role = args[1]
	}
	switch role {
	case RoleBroker:
		if io.state.ctl == nil || io.state.owner == nil {
			return true, 1
		}
		return true, runBroker(io, args)
	case RoleGuardian:
		if io.state.ctl == nil {
			return true, 1
		}
		return true, runGuardian(io, args)
	default:
		io.writeDiag("machinery: unknown internal role\n")
		return true, 1
	}
}

func InheritedInternalIO(ctx context.Context) (InternalIO, bool, error) {
	if ctx == nil {
		return InternalIO{}, false, errf(CodeInvalidSchema, "inherited-io", "context is nil")
	}
	if err := ctx.Err(); err != nil {
		return InternalIO{}, false, errFromContext(ctx, "inherited-io")
	}
	if _, ok := ctx.Deadline(); !ok {
		return InternalIO{}, false, errf(CodeInvalidSchema, "inherited-io", "acquisition context requires a finite deadline")
	}
	claim, present := parseInternalClaim(os.Getenv)
	if !present {
		return InternalIO{}, false, nil
	}
	if claim.ctl < 3 || !isSocketFD(claim.ctl) {
		return InternalIO{}, true, errf(CodeInvalidSchema, "inherited-io", "control descriptor claim is not a live inherited socket")
	}
	if claim.owner != 0 && (claim.owner < 3 || !isPipeFD(claim.owner)) {
		return InternalIO{}, true, errf(CodeInvalidSchema, "inherited-io", "owner liveness descriptor claim is not a live inherited pipe")
	}
	if claim.parent != 0 && (claim.parent < 3 || !isPipeFD(claim.parent)) {
		return InternalIO{}, true, errf(CodeInvalidSchema, "inherited-io", "parent liveness descriptor claim is not a live inherited pipe")
	}
	if !claim.deadline.IsZero() && !claim.deadline.After(time.Now()) {
		return InternalIO{}, true, errf(CodeTimeout, "inherited-io", "authenticated deadline already expired")
	}
	st := &ioState{
		ctl:      os.NewFile(uintptr(claim.ctl), "inherited-ctl"),
		deadline: claim.deadline,
		diag:     os.Stderr,
	}
	if claim.owner >= 3 {
		st.owner = os.NewFile(uintptr(claim.owner), "inherited-owner")
	}
	if claim.parent >= 3 {
		st.parent = os.NewFile(uintptr(claim.parent), "inherited-parent")
	}
	return InternalIO{state: st}, true, nil
}

func InheritedCapability(ctx context.Context) (Capability, error) {
	if ctx == nil {
		return Capability{}, errf(CodeInvalidSchema, "inherited-capability", "context is nil")
	}
	if err := ctx.Err(); err != nil {
		return Capability{}, errFromContext(ctx, "inherited-capability")
	}
	if _, ok := ctx.Deadline(); !ok {
		return Capability{}, errf(CodeInvalidSchema, "inherited-capability", "acquisition context requires a finite deadline")
	}
	claim, present := parseCapabilityClaim(os.Getenv)
	if !present {
		return Capability{}, errf(CodeCustodyError, "inherited-capability", "absent required attachment")
	}
	if claim.fd < 3 || !isSocketFD(claim.fd) {
		return Capability{}, errf(CodeInvalidSchema, "inherited-capability", "attachment descriptor claim is not a live inherited socket")
	}
	if claim.scopeID == "" || claim.rootID == "" {
		return Capability{}, errf(CodeInvalidSchema, "inherited-capability", "attachment claim lacks scope identity")
	}
	if !claim.deadline.IsZero() && !claim.deadline.After(time.Now()) {
		return Capability{}, errf(CodeTimeout, "inherited-capability", "authenticated deadline already expired")
	}
	return Capability{state: &capState{
		conn:     os.NewFile(uintptr(claim.fd), "inherited-capability"),
		scopeID:  claim.scopeID,
		rootID:   claim.rootID,
		deadline: claim.deadline,
	}}, nil
}
