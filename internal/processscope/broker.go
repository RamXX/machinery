package processscope

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type scopeNode struct {
	id      string
	parent  *scopeNode
	closing bool
}

func (n *scopeNode) inSubtree(other *scopeNode) bool {
	for cur := n; cur != nil; cur = cur.parent {
		if cur == other {
			return true
		}
	}
	return false
}

type job struct {
	id           string
	scope        *scopeNode
	guardianPID  int
	pgid         int
	registered   bool
	terminated   bool
	reaped       bool
	started      bool
	completed    bool
	terminal     bool
	retired      bool
	exitCode     int
	signal       string
	cancelCode   string
	budgetExceed bool
	guardian     *os.Process
	gconn        *net.UnixConn
	client       *channel
	resultCh     chan struct{}
	once         sync.Once
}

type containerRec struct {
	id     string
	daemon string
	scope  *scopeNode
}

type attachRec struct {
	scopeID  string
	digest   string
	deadline int64
	cap      bool
}

type channel struct {
	c     *net.UnixConn
	wmu   sync.Mutex
	fr    *frameReader
	bound string
}

func (ch *channel) send(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	ch.wmu.Lock()
	defer ch.wmu.Unlock()
	return writeFrame(ch.c, b)
}

func (ch *channel) sendWith(v any, files ...*os.File) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	ch.wmu.Lock()
	defer ch.wmu.Unlock()
	return writeFrame(ch.c, b, files...)
}

type broker struct {
	mu             sync.Mutex
	rootID         string
	dir            string
	selfExe        string
	limits         Limits
	wallDeadline   time.Time
	cleanupArmed   bool
	cleanupPending time.Time
	scopes         map[string]*scopeNode
	jobs           map[string]*job
	jobOrder       []string
	containers     map[string]*containerRec
	nonces         map[string]*attachRec
	ledgerOnce     sync.Once
	authorize      func(bound, claimed string) error
	register       func(b *broker, j *job) error
	groupSignal    func(pgid int, sig int) error
}

var hookAuthorizeRequest = func(bound, claimed string) error {
	if bound != "" && bound != claimed {
		return errf(CodeAuthFailed, "authorize", "claim does not match channel binding")
	}
	return nil
}

var hookRegisterJob = func(b *broker, j *job) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, dup := b.jobs[j.id]; dup {
		return errf(CodeInternalError, "register", "duplicate job id")
	}
	b.jobs[j.id] = j
	b.jobOrder = append(b.jobOrder, j.id)
	return nil
}

var hookGroupSignal = func(pgid int, sig int) error {
	return signalGroup(pgid, sig)
}

func (b *broker) scopeByID(id string) *scopeNode {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.scopes[id]
}

func (b *broker) openScope(id string) *scopeNode {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := b.scopes[id]
	if n == nil || n.closing {
		return nil
	}
	return n
}

func (b *broker) liveJobsLocked() int {
	n := 0
	for _, j := range b.jobs {
		if !j.retired {
			n++
		}
	}
	return n
}

func (b *broker) markClosingLocked(n *scopeNode) {
	if n == nil || n.closing {
		return
	}
	n.closing = true
	for _, other := range b.scopes {
		if other.parent == n {
			b.markClosingLocked(other)
		}
	}
}

func (b *broker) newScope(parent *scopeNode) *scopeNode {
	n := &scopeNode{id: "scope-" + mustRandomHex(8), parent: parent}
	b.mu.Lock()
	b.scopes[n.id] = n
	b.mu.Unlock()
	return n
}

func (b *broker) armCleanupLocked() {
	b.cleanupPending = NextCleanupDeadline(time.Now(), b.cleanupArmed, b.cleanupPending, b.limits.CleanupMS)
	b.cleanupArmed = true
}

func runBroker(io InternalIO, args []string) int {
	if !io.consume() {
		return 1
	}
	st := io.state
	if st == nil || st.ctl == nil || st.owner == nil {
		return 1
	}
	dir := ""
	for i := 1; i+1 < len(args); i++ {
		if args[i] == "--dir" {
			dir = args[i+1]
		}
	}
	conn, err := connFromFile(st.ctl)
	st.ctl.Close()
	if err != nil {
		st.owner.Close()
		return 2
	}
	self, err := selfDigest()
	if err != nil {
		conn.Close()
		st.owner.Close()
		return 2
	}
	_ = conn.SetReadDeadline(time.Now().Add(authWindow))
	payload, _, err := readFrame(conn)
	if err != nil {
		conn.Close()
		st.owner.Close()
		return 2
	}
	var boot msgBootstrap
	if err := json.Unmarshal(payload, &boot); err != nil || boot.T != "bootstrap" || !isHex64(boot.Digest) {
		conn.Close()
		st.owner.Close()
		return 1
	}
	if boot.Digest != self {
		_ = writeFrame(conn, mustMarshal(msgRefused{T: "refused", Code: CodeAuthFailed, Subject: "bootstrap", Message: "helper digest mismatch"}))
		conn.Close()
		st.owner.Close()
		return 1
	}
	wall := time.Unix(0, boot.Deadline)
	if wall.IsZero() || !wall.After(time.Now()) {
		conn.Close()
		st.owner.Close()
		return 1
	}
	exe, _ := os.Executable()
	limits, err := NormalizeLimits(boot.Limits)
	if err != nil {
		conn.Close()
		st.owner.Close()
		return 1
	}
	b := &broker{
		rootID:       "root-" + mustRandomHex(8),
		dir:          dir,
		selfExe:      exe,
		limits:       limits,
		wallDeadline: wall,
		scopes:       map[string]*scopeNode{},
		jobs:         map[string]*job{},
		containers:   map[string]*containerRec{},
		nonces:       map[string]*attachRec{},
		authorize:    hookAuthorizeRequest,
		register:     hookRegisterJob,
		groupSignal:  hookGroupSignal,
	}
	b.scopes["root"] = &scopeNode{id: "root"}
	_ = conn.SetReadDeadline(time.Time{})
	if err := writeFrame(conn, mustMarshal(msgReady{T: "ready", Root: b.rootID, Ack: boot.Challenge, Self: self})); err != nil {
		st.owner.Close()
		return 2
	}

	liveness := make(chan struct{})
	var once sync.Once
	watch := func(f *os.File) {
		if f == nil {
			return
		}
		go func() {
			buf := make([]byte, 1)
			for {
				if _, err := f.Read(buf); err != nil {
					once.Do(func() { close(liveness) })
					return
				}
			}
		}()
	}
	watch(st.owner)
	watch(st.parent)

	sigCh := make(chan os.Signal, 4)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	rootCh := &channel{c: conn, fr: newFrameReader(conn), bound: "root"}
	served := make(chan struct{})
	go func() {
		b.serveChannel(rootCh)
		close(served)
	}()

	select {
	case <-liveness:
	case <-sigCh:
	case <-served:
	}
	b.mu.Lock()
	b.markClosingLocked(b.scopes["root"])
	b.armCleanupLocked()
	b.mu.Unlock()
	b.retireAll()
	b.writeLedger()
	conn.Close()
	st.owner.Close()
	if st.parent != nil {
		st.parent.Close()
	}
	return 0
}

func mustRandomHex(n int) string {
	s, err := randomHex(n)
	if err != nil {
		return "00000000000000000000000000000000"[:n*2]
	}
	return s
}

func (b *broker) serveChannel(ch *channel) {
	for {
		payload, files, err := ch.fr.read()
		if err != nil {
			return
		}
		if quit := b.dispatch(ch, payload, files); quit {
			return
		}
	}
}

func (b *broker) dispatch(ch *channel, payload []byte, files []*os.File) bool {
	var head struct {
		T string `json:"t"`
	}
	if err := json.Unmarshal(payload, &head); err != nil {
		_ = ch.send(msgRefused{T: "refused", Code: CodeInvalidSchema, Subject: "control", Message: "malformed control record"})
		return false
	}
	switch head.T {
	case "hello":
		var m msgHello
		if err := json.Unmarshal(payload, &m); err != nil {
			return false
		}
		if err := b.authorize(ch.bound, m.Scope); err != nil {
			_ = ch.send(msgRefused{T: "refused", Code: CodeAuthFailed, Subject: "hello", Message: "scope claim rejected"})
			return false
		}
		_ = ch.send(msgWelcome{T: "welcome", Root: b.rootID, Deadline: b.wallDeadline.UnixNano()})
	case "attach":
		var m msgAttach
		if err := json.Unmarshal(payload, &m); err != nil {
			return false
		}
		if b.openScope(m.Scope) == nil {
			_ = ch.send(msgRefused{T: "refused", Code: CodeScopeClosed, Subject: "attach", Message: "scope is closing or unknown"})
			return false
		}
		if RemainingWallMS(time.Now(), b.wallDeadline) <= 0 {
			_ = ch.send(msgRefused{T: "refused", Code: CodeBudgetExhausted, Subject: "attach", Message: "remaining wall budget is not positive"})
			return false
		}
		nonce, err := randomHex(32)
		if err != nil {
			_ = ch.send(msgRefused{T: "refused", Code: CodeInternalError, Subject: "attach", Message: "nonce generation failed"})
			return false
		}
		b.mu.Lock()
		b.nonces[nonce] = &attachRec{scopeID: m.Scope, digest: m.Digest, deadline: m.Deadline, cap: m.Cap}
		b.mu.Unlock()
		_ = ch.send(msgAttached{T: "attached", Nonce: nonce})
	case "run":
		var m msgRun
		if err := json.Unmarshal(payload, &m); err != nil {
			return false
		}
		b.handleRun(ch, m, files)
	case "cancel":
		var m msgCancel
		if err := json.Unmarshal(payload, &m); err != nil {
			return false
		}
		auth := b.scopeByID(m.Scope)
		b.mu.Lock()
		j := b.jobs[m.Job]
		b.mu.Unlock()
		if j == nil || auth == nil || !j.scope.inSubtree(auth) {
			_ = ch.send(msgRefused{T: "refused", Code: CodeStaleCapability, Subject: "cancel", Message: "unknown or foreign job"})
			return false
		}
		if j.retired {
			return false
		}
		j.once.Do(func() {
			switch m.Reason {
			case "timeout":
				j.cancelCode = CodeTimeout
			case "overflow":
				j.cancelCode = CodeOutputLimit
			default:
				j.cancelCode = CodeCanceled
			}
		})
		go b.retireJob(j)
	case "close":
		var m msgClose
		if err := json.Unmarshal(payload, &m); err != nil {
			return false
		}
		node := b.scopeByID(m.Scope)
		if node == nil {
			_ = ch.send(msgRefused{T: "refused", Code: CodeScopeClosed, Subject: "close", Message: "unknown scope"})
			return false
		}
		b.mu.Lock()
		b.markClosingLocked(node)
		b.armCleanupLocked()
		b.mu.Unlock()
		b.retireSubtree(node)
		report := b.buildReport(node)
		_ = ch.send(msgClosed{T: "closed", Report: report})
		return node == b.scopes["root"]
	case "childreq":
		var m msgChildReq
		if err := json.Unmarshal(payload, &m); err != nil {
			return false
		}
		node := b.openScope(m.Scope)
		if node == nil {
			_ = ch.send(msgRefused{T: "refused", Code: CodeScopeClosed, Subject: "child", Message: "scope is closing or unknown"})
			return false
		}
		child := b.newScope(node)
		a, bb, err := newChannelPair()
		if err != nil {
			_ = ch.send(msgRefused{T: "refused", Code: CodeInternalError, Subject: "child", Message: "channel creation failed"})
			return false
		}
		childCh := &channel{c: mustConn(a), bound: child.id}
		childCh.fr = newFrameReader(childCh.c)
		go b.serveChannel(childCh)
		if err := ch.sendWith(msgChildOK{T: "childok", Scope: child.id}, bb); err != nil {
			bb.Close()
			return false
		}
		bb.Close()
	case "regcontainer":
		var m msgRegContainer
		if err := json.Unmarshal(payload, &m); err != nil {
			return false
		}
		node := b.openScope(m.Scope)
		if node == nil {
			_ = ch.send(msgRefused{T: "refused", Code: CodeScopeClosed, Subject: "regcontainer", Message: "scope is closing or unknown"})
			return false
		}
		if m.ID == "" || m.Daemon == "" {
			_ = ch.send(msgRefused{T: "refused", Code: CodeInvalidSchema, Subject: "regcontainer", Message: "container id and daemon are required"})
			return false
		}
		b.mu.Lock()
		b.containers[m.ID] = &containerRec{id: m.ID, daemon: m.Daemon, scope: node}
		b.mu.Unlock()
		_ = ch.send(map[string]string{"t": "regcontainerok"})
	default:
		_ = ch.send(msgRefused{T: "refused", Code: CodeInvalidSchema, Subject: "control", Message: "unknown control record"})
	}
	return false
}

func mustConn(f *os.File) *net.UnixConn {
	if f == nil {
		return nil
	}
	c, err := connFromFile(f)
	if err != nil {
		return nil
	}
	f.Close()
	return c
}

func (b *broker) handleRun(ch *channel, m msgRun, files []*os.File) {
	node := b.openScope(m.Scope)
	b.mu.Lock()
	rec := b.nonces[m.Nonce]
	if rec != nil && (rec.scopeID != m.Scope || rec.digest != m.Digest) {
		rec = nil
	}
	live := b.liveJobsLocked()
	b.mu.Unlock()

	if node == nil {
		_ = ch.send(msgRefused{T: "refused", Code: CodeScopeClosed, Subject: "run", Message: "scope is closing or unknown"})
		return
	}
	if rec == nil {
		_ = ch.send(msgRefused{T: "refused", Code: CodeInvalidCommand, Subject: "run", Message: "stale, forged, or spent attachment"})
		return
	}
	if time.Now().UnixNano() >= rec.deadline {
		_ = ch.send(msgRefused{T: "refused", Code: CodeBudgetExhausted, Subject: "run", Message: "attachment deadline expired"})
		return
	}
	if live >= int(b.limits.Jobs) {
		_ = ch.send(msgRefused{T: "refused", Code: CodeRegistrationFailed, Subject: "run", Message: "concurrent job limit reached"})
		return
	}
	if RemainingWallMS(time.Now(), b.wallDeadline) <= 0 {
		_ = ch.send(msgRefused{T: "refused", Code: CodeBudgetExhausted, Subject: "run", Message: "remaining wall budget is not positive"})
		return
	}

	spec := m.Spec
	spec.RootID = b.rootID
	spec.JobID = "job-" + mustRandomHex(8)
	if remaining := RemainingWallMS(time.Now(), b.wallDeadline); remaining < spec.DeadlineMS {
		spec.DeadlineMS = remaining
	}
	spec.DrainGraceMS = b.limits.CleanupMS
	if spec.DrainGraceMS > drainGraceMS {
		spec.DrainGraceMS = drainGraceMS
	}
	if hasChildRequest(spec.Env) != rec.cap {
		_ = ch.send(msgRefused{T: "refused", Code: CodeInvalidCommand, Subject: "run", Message: "attachment capability flag does not match declared environment"})
		return
	}

	jobScope := node
	var capConn *os.File
	if rec.cap {
		child := b.newScope(node)
		spec.ScopeID = child.id
		ca, cb, err := newChannelPair()
		if err != nil {
			_ = ch.send(msgRefused{T: "refused", Code: CodeInternalError, Subject: "run", Message: "capability channel creation failed"})
			return
		}
		spec.CapEnv = encodeCapabilityEnv(3, child.id, b.rootID, b.wallDeadline)
		spec.FDCap = true
		capCh := &channel{c: mustConn(ca), bound: child.id}
		capCh.fr = newFrameReader(capCh.c)
		go b.serveChannel(capCh)
		capConn = cb
		env := make([]string, 0, len(spec.Env))
		for _, e := range spec.Env {
			if e == EnvChildRequest+"=join" {
				env = append(env, EnvCapability+"="+spec.CapEnv)
				continue
			}
			env = append(env, e)
		}
		spec.Env = env
		jobScope = child
	} else {
		env := make([]string, 0, len(spec.Env))
		for _, e := range spec.Env {
			if e == EnvChildRequest+"=join" {
				continue
			}
			env = append(env, e)
		}
		spec.Env = env
	}

	want := 0
	for _, f := range files {
		if f != nil {
			want++
		}
	}
	need := 0
	for _, ok := range []bool{spec.FDStdin, spec.FDStdout, spec.FDStderr} {
		if ok {
			need++
		}
	}
	if want != need {
		if capConn != nil {
			capConn.Close()
		}
		_ = ch.send(msgRefused{T: "refused", Code: CodeInvalidCommand, Subject: "run", Message: "stdio descriptor set does not match streams"})
		return
	}
	if capConn != nil {
		files = append(files, capConn)
	}

	j := &job{
		id:         spec.JobID,
		scope:      jobScope,
		registered: true,
		client:     ch,
		resultCh:   make(chan struct{}),
	}
	if err := b.register(b, j); err != nil {
		_ = ch.send(msgRefused{T: "refused", Code: CodeRegistrationFailed, Subject: "run", Message: "registration failed: " + err.Error()})
		return
	}
	b.mu.Lock()
	delete(b.nonces, m.Nonce)
	b.mu.Unlock()

	ga, gb, err := newChannelPair()
	if err != nil {
		j.retired = true
		j.terminal = true
		_ = ch.send(msgRefused{T: "refused", Code: CodeInternalError, Subject: "run", Message: "guardian channel creation failed"})
		return
	}
	var logf *os.File
	if b.dir != "" {
		logf, _ = os.OpenFile(filepath.Join(b.dir, "guardian-"+spec.JobID+".log"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	}
	gcmd := exec.Command(b.selfExe, InternalMarker, "guardian")
	gcmd.Env = []string{
		EnvInternalCtl + "=3",
		EnvInternalDeadline + "=" + fmt.Sprintf("%d", b.wallDeadline.UnixNano()),
	}
	gcmd.ExtraFiles = []*os.File{gb}
	gcmd.SysProcAttr = newGroupAttr()
	if logf != nil {
		gcmd.Stdout = logf
		gcmd.Stderr = logf
	}
	if err := gcmd.Start(); err != nil {
		ga.Close()
		gb.Close()
		if logf != nil {
			logf.Close()
		}
		j.retired = true
		j.terminal = true
		_ = ch.send(msgRefused{T: "refused", Code: CodeInternalError, Subject: "run", Message: "guardian launch failed: " + err.Error()})
		return
	}
	gb.Close()
	if logf != nil {
		logf.Close()
	}
	j.guardian = gcmd.Process
	j.guardianPID = gcmd.Process.Pid
	j.pgid = gcmd.Process.Pid

	gch := &channel{c: mustConn(ga), bound: ""}
	gch.fr = newFrameReader(gch.c)
	j.gconn = gch.c

	if err := gch.sendWith(msgJob{T: "job", Spec: spec}, files...); err != nil {
		j.once.Do(func() { j.cancelCode = CodeInternalError })
		b.retireJob(j)
		_ = ch.send(msgRefused{T: "refused", Code: CodeInternalError, Subject: "run", Message: "guardian dispatch failed"})
		return
	}
	for _, f := range files {
		f.Close()
	}

	go func() {
		payload, _, err := gch.fr.read()
		if err != nil {
			j.once.Do(func() { j.cancelCode = CodeCustodyError })
			closeTerminal(j)
			return
		}
		var head struct {
			T string `json:"t"`
		}
		if err := json.Unmarshal(payload, &head); err != nil {
			closeTerminal(j)
			return
		}
		if head.T == "gstartfail" {
			var gf msgGStartFail
			_ = json.Unmarshal(payload, &gf)
			j.once.Do(func() { j.cancelCode = gf.Code })
			_ = ch.send(msgRefused{T: "refused", Code: gf.Code, Subject: "run", Message: "target start failed: " + gf.Message})
			b.retireJob(j)
			return
		}
		var gs msgGStarted
		_ = json.Unmarshal(payload, &gs)
		j.started = true
		_ = ch.send(msgStarted{T: "started", Job: j.id, Pid: gs.Pid})
		for {
			payload, _, err := gch.fr.read()
			if err != nil {
				j.once.Do(func() { j.cancelCode = CodeCustodyError })
				closeTerminal(j)
				return
			}
			var ge msgGExit
			if err := json.Unmarshal(payload, &ge); err != nil || ge.T != "gexit" {
				continue
			}
			j.completed = ge.Completed
			j.exitCode = ge.ExitCode
			j.signal = ge.Signal
			closeTerminal(j)
			if !ge.Terminal {
				b.forwardResult(j)
			}
			return
		}
	}()
}

func closeTerminal(j *job) {
	select {
	case <-j.resultCh:
	default:
		close(j.resultCh)
	}
}

func (b *broker) forwardResult(j *job) {
	if j.client == nil {
		return
	}
	cleanup := CleanupReport{Status: StatusCleaned, Jobs: []ResourceState{{ID: j.id, Registered: j.registered, Terminated: j.terminated, Reaped: j.reaped}}}
	if !j.terminated || !j.reaped {
		if j.cancelCode != "" {
			cleanup.Status = StatusCleanupFailed
		}
	}
	_ = j.client.send(msgResult{
		T: "result", Job: j.id, Started: j.started, Completed: j.completed,
		ExitCode: j.exitCode, Signal: j.signal, Code: j.cancelCode, Cleanup: cleanup,
	})
}

func (b *broker) retireJob(j *job) {
	b.mu.Lock()
	if j.retired {
		b.mu.Unlock()
		return
	}
	j.retired = true
	armed := b.cleanupArmed
	pending := b.cleanupPending
	cleanupMS := b.limits.CleanupMS
	b.mu.Unlock()

	var waitUntil time.Time
	if armed {
		waitUntil = pending
	} else {
		w := cleanupMS
		if w > limitsCaps.CleanupMS {
			w = limitsCaps.CleanupMS
		}
		if w < 1 {
			w = 1
		}
		waitUntil = time.Now().Add(time.Duration(w) * time.Millisecond)
	}
	if cap := time.Now().Add(3 * time.Second); waitUntil.After(cap) {
		waitUntil = cap
	}

	_ = b.groupSignal(j.pgid, int(syscall.SIGTERM))
	select {
	case <-j.resultCh:
	case <-time.After(time.Until(waitUntil)):
	}
	_ = b.groupSignal(j.pgid, int(syscall.SIGKILL))
	j.terminated = true
	if j.guardianPID > 0 {
		reapDeadline := time.Now().Add(hardReapWindow)
		for time.Now().Before(reapDeadline) {
			ok, err := reapPID(j.guardianPID)
			if err == nil && ok {
				j.reaped = true
				break
			}
			if err != nil && !errors.Is(err, syscall.EINTR) {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if !j.reaped {
			j.terminated = false
		} else if groupExists(j.pgid) {
			j.terminated = false
		}
		if j.guardian != nil {
			j.guardian.Release()
		}
	} else {
		j.terminated = false
	}
	if time.Now().After(waitUntil) {
		j.budgetExceed = true
	}
	j.terminal = true
	closeTerminal(j)
	if j.gconn != nil {
		j.gconn.Close()
	}
	b.forwardResult(j)
}

func (b *broker) retireAll() {
	b.mu.Lock()
	jobs := make([]*job, 0, len(b.jobs))
	for _, id := range b.jobOrder {
		if j := b.jobs[id]; !j.retired {
			jobs = append(jobs, j)
		}
	}
	b.mu.Unlock()
	for _, j := range jobs {
		b.retireJob(j)
	}
}

func (b *broker) retireSubtree(node *scopeNode) {
	b.mu.Lock()
	jobs := make([]*job, 0)
	for _, id := range b.jobOrder {
		if j := b.jobs[id]; !j.retired && j.scope.inSubtree(node) {
			jobs = append(jobs, j)
		}
	}
	b.mu.Unlock()
	for _, j := range jobs {
		b.retireJob(j)
	}
}

func (b *broker) buildReport(node *scopeNode) CleanupReport {
	b.mu.Lock()
	defer b.mu.Unlock()
	root := b.scopes["root"]
	rep := CleanupReport{Status: StatusCleaned}
	for _, id := range b.jobOrder {
		j := b.jobs[id]
		if j == nil || (node != root && !j.scope.inSubtree(node)) {
			continue
		}
		rep.Jobs = append(rep.Jobs, ResourceState{ID: j.id, Registered: j.registered, Terminated: j.terminated, Reaped: j.reaped})
		if !j.registered || !j.terminated || !j.reaped {
			rep.Status = StatusCleanupFailed
		}
		if j.budgetExceed {
			rep.Diagnostics = append(rep.Diagnostics, Diagnostic{Code: CodeTimeout, Subject: j.id, Message: "shared cleanup budget exhausted before retirement completed"})
			rep.Status = StatusCleanupFailed
		}
	}
	for _, c := range b.containers {
		if node != root && !c.scope.inSubtree(node) {
			continue
		}
		rep.Containers = append(rep.Containers, ResourceState{ID: c.id, Registered: true, Terminated: false, Reaped: false})
		rep.Diagnostics = append(rep.Diagnostics, Diagnostic{Code: CodeCustodyError, Subject: c.id, Message: "container removal requires live daemon cleanup wiring; process death never implies container removal"})
		rep.Status = StatusCleanupFailed
	}
	return rep
}

func (b *broker) writeLedger() {
	b.ledgerOnce.Do(func() {
		if b.dir == "" {
			return
		}
		b.mu.Lock()
		root := b.scopes["root"]
		rec := ledgerRecord{
			Schema:       LedgerDomain,
			Root:         b.rootID,
			WallDeadline: b.wallDeadline.UnixNano(),
			Status:       StatusCleaned,
			FinishedAt:   time.Now().UnixNano(),
		}
		if b.cleanupArmed {
			rec.CleanupArmed = b.cleanupPending.UnixNano()
		}
		for _, id := range b.jobOrder {
			j := b.jobs[id]
			if j == nil {
				continue
			}
			_ = root
			rec.Jobs = append(rec.Jobs, ledgerJob{Job: j.id, Scope: j.scope.id, Guardian: j.guardianPID, PGID: j.pgid, Registered: j.registered, Terminated: j.terminated, Reaped: j.reaped})
			if !j.terminated || !j.reaped {
				rec.Status = StatusCleanupFailed
			}
		}
		for _, c := range b.containers {
			rec.Containers = append(rec.Containers, ledgerContainer{ID: c.id, Registered: true, Terminated: false, Reaped: false})
		}
		b.mu.Unlock()
		out, err := json.MarshalIndent(rec, "", " ")
		if err != nil {
			return
		}
		tmp := filepath.Join(b.dir, ".ledger.tmp")
		final := filepath.Join(b.dir, "ledger.json")
		if err := os.WriteFile(tmp, out, 0600); err != nil {
			return
		}
		_ = os.Rename(tmp, final)
	})
}
