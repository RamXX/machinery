//go:build unix

package processscope

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	io, ok, err := InheritedInternalIO(ctx)
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, "machinery: invalid internal channel claim:", err)
		os.Exit(2)
	}
	if ok {
		handled, code := ServeInternal(os.Args[1:], io)
		if !handled {
			fmt.Fprintln(os.Stderr, "machinery: internal protocol failure")
			os.Exit(2)
		}
		os.Exit(code)
	}
	os.Exit(m.Run())
}

func asCode(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %s, got nil", want)
	}
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("expected *processscope.Error, got %T: %v", err, err)
	}
	if e.Code != want {
		t.Fatalf("expected error code %s, got %s (%v)", want, e.Code, err)
	}
}

func mustCtx() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_ = cancel
	return ctx
}

func TestNormalizeLimits(t *testing.T) {
	got, err := NormalizeLimits(Limits{})
	if err != nil {
		t.Fatalf("defaults: %v", err)
	}
	want := Limits{
		WallMS:      600000,
		CleanupMS:   10000,
		StdoutBytes: 4194304,
		StderrBytes: 4194304,
		EventBytes:  16777216,
		EventCount:  100000,
		Jobs:        1,
		BundleBytes: 536870912,
		Entries:     100000,
		Depth:       64,
	}
	if got != want {
		t.Fatalf("defaults mismatch: got %+v want %+v", got, want)
	}
	got, err = NormalizeLimits(Limits{WallMS: 1000, Jobs: 2})
	if err != nil {
		t.Fatalf("partial: %v", err)
	}
	if got.WallMS != 1000 || got.Jobs != 2 || got.CleanupMS != 10000 {
		t.Fatalf("partial mismatch: %+v", got)
	}
	if got, err = NormalizeLimits(Limits{WallMS: 3600000}); err != nil || got.WallMS != 3600000 {
		t.Fatalf("wall cap edge: %v %+v", err, got)
	}
	asCode(t, mustErr(NormalizeLimits(Limits{WallMS: 3600001})), CodeInvalidSchema)
	asCode(t, mustErr(NormalizeLimits(Limits{CleanupMS: 30001})), CodeInvalidSchema)
	asCode(t, mustErr(NormalizeLimits(Limits{Jobs: 5})), CodeInvalidSchema)
	asCode(t, mustErr(NormalizeLimits(Limits{StdoutBytes: -1})), CodeInvalidSchema)
	asCode(t, mustErr(NormalizeLimits(Limits{Depth: 129})), CodeInvalidSchema)
	asCode(t, mustErr(NormalizeLimits(Limits{EventCount: 1000001})), CodeInvalidSchema)
	asCode(t, mustErr(NormalizeLimits(Limits{BundleBytes: 4294967297})), CodeInvalidSchema)
}

func mustErr(_ Limits, err error) error { return err }

func TestBudgetArithmetic(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	if r := RemainingWallMS(now, now.Add(1500*time.Millisecond)); r != 1500 {
		t.Fatalf("remaining: %d", r)
	}
	if r := RemainingWallMS(now, now.Add(-500*time.Millisecond)); r != -500 {
		t.Fatalf("negative remaining: %d", r)
	}
	scopeDeadline := now.Add(10 * time.Second)
	if r := EffectiveDeadlineMS(now, 0, context.Background(), scopeDeadline); r != 10000 {
		t.Fatalf("effective scope: %d", r)
	}
	if r := EffectiveDeadlineMS(now, 3000, context.Background(), scopeDeadline); r != 3000 {
		t.Fatalf("effective cmd: %d", r)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	cancel()
	if r := EffectiveDeadlineMS(now, 0, ctx, scopeDeadline); r > 0 {
		t.Fatalf("effective cancelled ctx must be nonpositive: %d", r)
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	realNow := time.Now()
	if r := EffectiveDeadlineMS(realNow, 0, ctx2, realNow.Add(10*time.Second)); r < 1500 || r > 2000 {
		t.Fatalf("effective ctx: %d", r)
	}
	first := NextCleanupDeadline(now, false, time.Time{}, 8000)
	if !first.Equal(now.Add(8 * time.Second)) {
		t.Fatalf("arm: %v", first)
	}
	later := now.Add(5 * time.Second)
	if d := NextCleanupDeadline(later, true, first, 8000); !d.Equal(first) {
		t.Fatalf("renewal refused: %v", d)
	}
	if d := NextCleanupDeadline(later, true, first, 30000); !d.Equal(first) {
		t.Fatalf("renewal refused 2: %v", d)
	}
	if d := NextCleanupDeadline(later, false, time.Time{}, 30000); !d.Equal(later.Add(30 * time.Second)) {
		t.Fatalf("cap: %v", d)
	}
}

func TestServeInternalRefusals(t *testing.T) {
	if handled, code := ServeInternal([]string{"check", "design"}, InternalIO{}); handled || code != 0 {
		t.Fatalf("non-marker: %v %d", handled, code)
	}
	if handled, code := ServeInternal([]string{InternalMarker, "broker"}, InternalIO{}); !handled || code != 1 {
		t.Fatalf("zero io: %v %d", handled, code)
	}
	st := &ioState{deadline: time.Now().Add(time.Hour)}
	st.status.Store(handleConsumed)
	if handled, code := ServeInternal([]string{InternalMarker, "guardian"}, InternalIO{state: st}); !handled || code != 1 {
		t.Fatalf("consumed io: %v %d", handled, code)
	}
	closed := InternalIO{state: &ioState{}}
	closed.Close()
	if handled, code := ServeInternal([]string{InternalMarker, "broker"}, closed); !handled || code != 1 {
		t.Fatalf("closed io: %v %d", handled, code)
	}
}

func TestInheritedInternalIOContextAndClaims(t *testing.T) {
	asCode(t, func() error { _, _, err := InheritedInternalIO(nil); return err }(), CodeInvalidSchema)
	asCode(t, func() error { _, _, err := InheritedInternalIO(context.Background()); return err }(), CodeInvalidSchema)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	asCode(t, func() error { _, _, err := InheritedInternalIO(ctx); return err }(), CodeCanceled)

	t.Setenv(EnvInternalCtl, "")
	if _, ok, err := InheritedInternalIO(mustCtx()); err != nil || ok {
		t.Fatalf("absence: %v %v", ok, err)
	}

	f, err := os.Create(filepath.Join(t.TempDir(), "regular"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	t.Setenv(EnvInternalCtl, fmt.Sprintf("%d", f.Fd()))
	if _, ok, err := InheritedInternalIO(mustCtx()); !ok || err == nil {
		t.Fatalf("regular-file ctl must be present-invalid: %v %v", ok, err)
	} else {
		asCode(t, err, CodeInvalidSchema)
	}

	a, b, err := newChannelPair()
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	defer b.Close()
	t.Setenv(EnvInternalCtl, fmt.Sprintf("%d", a.Fd()))
	t.Setenv(EnvInternalDeadline, fmt.Sprintf("%d", time.Now().Add(-time.Second).UnixNano()))
	if _, ok, err := InheritedInternalIO(mustCtx()); !ok || err == nil {
		t.Fatalf("expired deadline must be present-invalid: %v %v", ok, err)
	} else {
		asCode(t, err, CodeTimeout)
	}
	t.Setenv(EnvInternalDeadline, fmt.Sprintf("%d", time.Now().Add(time.Hour).UnixNano()))
	io, ok, err := InheritedInternalIO(mustCtx())
	if err != nil || !ok {
		t.Fatalf("present-valid: %v %v", ok, err)
	}
	if err := io.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestInheritedCapabilityClaims(t *testing.T) {
	asCode(t, func() error { _, err := InheritedCapability(nil); return err }(), CodeInvalidSchema)
	asCode(t, func() error { _, err := InheritedCapability(context.Background()); return err }(), CodeInvalidSchema)
	t.Setenv(EnvCapability, "")
	asCode(t, func() error { _, err := InheritedCapability(mustCtx()); return err }(), CodeCustodyError)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	t.Setenv(EnvCapability, fmt.Sprintf("fd=%d;scope=s;root=r;deadline=%d", w.Fd(), time.Now().Add(time.Hour).UnixNano()))
	asCode(t, func() error { _, err := InheritedCapability(mustCtx()); return err }(), CodeInvalidSchema)
	a, b, err := newChannelPair()
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	defer b.Close()
	t.Setenv(EnvCapability, fmt.Sprintf("fd=%d;scope=s;root=r;deadline=%d", a.Fd(), time.Now().Add(-time.Second).UnixNano()))
	asCode(t, func() error { _, err := InheritedCapability(mustCtx()); return err }(), CodeTimeout)
	t.Setenv(EnvCapability, fmt.Sprintf("fd=%d;scope=s;root=r;deadline=%d", a.Fd(), time.Now().Add(time.Hour).UnixNano()))
	cap, err := InheritedCapability(mustCtx())
	if err != nil {
		t.Fatal(err)
	}
	if err := cap.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenOptionsValidation(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dg, err := fileDigest(exe)
	if err != nil {
		t.Fatal(err)
	}
	scratch := t.TempDir()
	good := Options{HelperExecutable: exe, HelperDigest: dg, ScratchRoot: scratch, Limits: Limits{Jobs: 2}}
	asCode(t, func() error { _, err := Open(nil, good); return err }(), CodeInvalidSchema)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	asCode(t, func() error { _, err := Open(ctx, good); return err }(), CodeCanceled)
	asCode(t, func() error {
		_, err := Open(mustCtx(), Options{HelperExecutable: exe, HelperDigest: "", ScratchRoot: scratch})
		return err
	}(), CodeInvalidOptions)
	asCode(t, func() error {
		_, err := Open(mustCtx(), Options{HelperExecutable: "machinery-helper", HelperDigest: dg, ScratchRoot: scratch})
		return err
	}(), CodeInvalidOptions)
	asCode(t, func() error {
		_, err := Open(mustCtx(), Options{HelperExecutable: filepath.Join(scratch, "missing-helper"), HelperDigest: dg, ScratchRoot: scratch})
		return err
	}(), CodeInvalidOptions)
	wrongDigest := dg[:63] + flipHex(dg[63])
	asCode(t, func() error {
		_, err := Open(mustCtx(), Options{HelperExecutable: exe, HelperDigest: wrongDigest, ScratchRoot: scratch})
		return err
	}(), CodeInvalidOptions)
	asCode(t, func() error {
		_, err := Open(mustCtx(), Options{HelperExecutable: exe, HelperDigest: dg, ScratchRoot: ""})
		return err
	}(), CodeInvalidOptions)
	asCode(t, func() error {
		_, err := Open(mustCtx(), Options{HelperExecutable: exe, HelperDigest: dg, ScratchRoot: "relative/scratch"})
		return err
	}(), CodeInvalidOptions)
	asCode(t, func() error {
		_, err := Open(mustCtx(), Options{HelperExecutable: exe, HelperDigest: dg, ScratchRoot: scratch, Limits: Limits{Jobs: 9}})
		return err
	}(), CodeInvalidSchema)
}

func flipHex(c byte) string {
	if c == '0' {
		return "1"
	}
	return "0"
}

func TestAttachmentCodecRoundTrip(t *testing.T) {
	cmd := Command{Executable: "/bin/true", Args: []string{"a", "b"}, Dir: "/tmp", Env: []string{"A=1", "B=2"}, RuntimeDigest: "sha256:" + repeatHex(64)}
	dg := commandDigest(cmd)
	if dg == "" || !isHex64(dg) {
		t.Fatalf("digest not produced: %q", dg)
	}
	tampered := cmd
	tampered.Args = []string{"a", "c"}
	if commandDigest(tampered) == dg {
		t.Fatal("arg tamper not detected by digest")
	}
	tamperedEnv := cmd
	tamperedEnv.Env = []string{"A=1", "B=3"}
	if commandDigest(tamperedEnv) == dg {
		t.Fatal("env tamper not detected by digest")
	}
	stripped := cmd
	stripped.Env = append([]string{"A=1", "B=2"}, EnvAttachment+"=v1;junk")
	if commandDigest(stripped) != dg {
		t.Fatal("reserved attachment entry must be excluded from digest")
	}
	enc := encodeAttachment(attachment{Nonce: dg, Digest: dg, Deadline: 42, Cap: true})
	parsed, ok := parseAttachment(enc)
	if !ok || parsed.Nonce != dg || parsed.Digest != dg || parsed.Deadline != 42 || !parsed.Cap {
		t.Fatalf("attachment round trip: %v %v", enc, parsed)
	}
	if _, ok := parseAttachment("v2;x;y;1;0"); ok {
		t.Fatal("malformed attachment accepted")
	}
	if _, ok := parseAttachment("v1;" + dg + ";" + dg + ";0;1"); ok {
		t.Fatal("nonpositive deadline accepted")
	}
	if !hasChildRequest([]string{"A=1", EnvChildRequest + "=join"}) || hasChildRequest([]string{EnvChildRequest + "=join", EnvChildRequest + "=join"}) {
		t.Fatal("child request detection")
	}
}

func repeatHex(n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = '0'
	}
	return string(out)
}

func startTestBroker(t *testing.T, limits Limits) (*scope, func()) {
	t.Helper()
	dir := t.TempDir()
	a, b, err := newChannelPair()
	if err != nil {
		t.Fatal(err)
	}
	lr, lw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dg, err := fileDigest(exe)
	if err != nil {
		t.Fatal(err)
	}
	norm, err := NormalizeLimits(limits)
	if err != nil {
		t.Fatal(err)
	}
	wall := time.Now().Add(time.Duration(norm.WallMS) * time.Millisecond)
	st := &ioState{ctl: b, owner: lr, deadline: wall}
	go func() { runBroker(InternalIO{state: st}, []string{InternalMarker, "broker", "--dir", dir}) }()
	connA, err := connFromFile(a)
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := randomHex(32)
	if err != nil {
		t.Fatal(err)
	}
	boot := msgBootstrap{T: "bootstrap", Challenge: challenge, Digest: dg, Deadline: wall.UnixNano(), Limits: norm}
	if err := writeJSONFrame(connA, boot); err != nil {
		t.Fatal(err)
	}
	var ready msgReady
	if err := readJSONFrame(connA, &ready); err != nil {
		t.Fatal(err)
	}
	if ready.Ack != challenge || ready.Self != dg || ready.Root == "" {
		t.Fatalf("bootstrap handshake: %+v", ready)
	}
	s := &scope{conn: connA, fr: newFrameReader(connA), id: "root", rootID: ready.Root, deadline: wall, limits: norm, isRoot: true, root: &rootHandle{dir: dir, livenessW: lw}}
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, _ = s.Close(ctx)
		connA.Close()
		lw.Close()
	}
	return s, cleanup
}

func writeJSONFrame(c *net.UnixConn, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return writeFrame(c, b)
}

func readJSONFrame(c *net.UnixConn, v any) error {
	b, _, err := readFrame(c)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func trueCommand() Command {
	return Command{Executable: "/usr/bin/true", Args: nil, Dir: "", Env: []string{}, RuntimeDigest: "sha256:" + repeatHex(64)}
}

func TestChallengeAuthorizeUnsafeAcceptsForgedScope(t *testing.T) {
	defaultHook := hookAuthorizeRequest
	defer func() { hookAuthorizeRequest = defaultHook }()

	s, cleanup := startTestBroker(t, Limits{})
	defer cleanup()
	if _, err := s.Run(mustCtx(), trueCommand(), Streams{}); err == nil {
		t.Fatal("run without attachment must be rejected")
	}
	hello := msgHello{T: "hello", Scope: "forged-scope", Role: RoleJoined}
	if err := writeJSONFrame(s.conn, hello); err != nil {
		t.Fatal(err)
	}
	var rep map[string]any
	if err := readJSONFrame(s.conn, &rep); err != nil {
		t.Fatal(err)
	}
	if rep["t"] != "refused" || rep["code"] != CodeAuthFailed {
		t.Fatalf("control: forged scope must be refused with AUTH_FAILED: %v", rep)
	}

	hookAuthorizeRequest = func(bound, claimed string) error { return nil }
	s2, cleanup2 := startTestBroker(t, Limits{})
	defer cleanup2()
	if err := writeJSONFrame(s2.conn, hello); err != nil {
		t.Fatal(err)
	}
	var rep2 map[string]any
	if err := readJSONFrame(s2.conn, &rep2); err != nil {
		t.Fatal(err)
	}
	if rep2["t"] != "welcome" {
		t.Fatalf("challenge: unsafe authorize must accept forged scope: %v", rep2)
	}
}

func TestChallengeRegistrationSkipObserved(t *testing.T) {
	defaultHook := hookRegisterJob
	defer func() { hookRegisterJob = defaultHook }()

	s, cleanup := startTestBroker(t, Limits{})
	defer cleanup()
	attached, err := s.Attach(trueCommand())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Run(mustCtx(), attached, Streams{}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rep, err := s.Close(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Jobs) != 1 || !rep.Jobs[0].Registered || !rep.Jobs[0].Terminated || !rep.Jobs[0].Reaped {
		t.Fatalf("control: report must show registered job: %+v", rep)
	}

	hookRegisterJob = func(b *broker, j *job) error { return nil }
	s2, cleanup2 := startTestBroker(t, Limits{})
	defer cleanup2()
	attached2, err := s2.Attach(trueCommand())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s2.Run(mustCtx(), attached2, Streams{}); err != nil {
		t.Fatal(err)
	}
	rep2, err := s2.Close(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep2.Jobs) != 0 {
		t.Fatalf("challenge: skipped registration must be observable as an unregistered job: %+v", rep2)
	}
}

func TestChallengeKillSkipLeaksGroup(t *testing.T) {
	defaultHook := hookGroupSignal
	defer func() { hookGroupSignal = defaultHook }()

	runCase := func(unsafe bool) (CleanupReport, int) {
		if unsafe {
			hookGroupSignal = func(pgid int, sig int) error { return nil }
		} else {
			hookGroupSignal = defaultHook
		}
		s, cleanup := startTestBroker(t, Limits{})
		defer cleanup()
		f := filepath.Join(t.TempDir(), "grandchild")
		exe, _ := os.Executable()
		cmd := Command{Executable: exe, Args: []string{"-test.run=^TestHelperTarget$", "-test.timeout=120s"}, Env: []string{"PATH=/bin:/usr/bin", "MACHINERY_QLW2_ROLE=grandparent", "MACHINERY_QLW2_FILE=" + f, "MACHINERY_QLW2_SECS=30"}, RuntimeDigest: "sha256:" + repeatHex(64)}
		attached, err := s.Attach(cmd)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.Run(mustCtx(), attached, Streams{}); err != nil {
			t.Fatal(err)
		}
		pid := waitPidFile(t, f, 5*time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		rep, err := s.Close(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return rep, pid
	}

	rep, pid := runCase(false)
	if rep.Status != StatusCleaned {
		t.Fatalf("control: cleaned: %+v", rep)
	}
	if !waitGonePid(pid, 5*time.Second) {
		t.Fatalf("control: grandchild %d must be terminated", pid)
	}

	rep2, pid2 := runCase(true)
	if len(rep2.Jobs) != 1 || rep2.Jobs[0].Reaped {
		t.Fatalf("challenge: unreaped guardian must be reported: %+v", rep2)
	}
	deadline := time.Now().Add(700 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !processAlive(pid2) {
			t.Fatalf("challenge: kill skip must leak the grandchild (it died unexpectedly)")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func processAlive(pid int) bool {
	return signalProbe(pid)
}

func waitPidFile(t *testing.T, path string, timeout time.Duration) int {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		b, err := os.ReadFile(path)
		if err == nil {
			n := 0
			ok := len(b) > 0
			for i := 0; i < len(b); i++ {
				if b[i] < '0' || b[i] > '9' {
					ok = false
					break
				}
				n = n*10 + int(b[i]-'0')
			}
			if ok && n > 0 {
				return n
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("pid file %s never appeared", path)
	return 0
}

func waitGonePid(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return !processAlive(pid)
}

func TestContainerRegistrationReport(t *testing.T) {
	s, cleanup := startTestBroker(t, Limits{})
	defer cleanup()
	if err := s.registerContainer(mustCtx(), "containerabc123", "/var/run/docker.sock"); err != nil {
		t.Fatal(err)
	}
	attached, err := s.Attach(trueCommand())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Run(mustCtx(), attached, Streams{}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rep, err := s.Close(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Containers) != 1 || rep.Containers[0].ID != "containerabc123" || !rep.Containers[0].Registered {
		t.Fatalf("container registration state: %+v", rep.Containers)
	}
	if rep.Containers[0].Terminated || rep.Containers[0].Reaped {
		t.Fatalf("container must never be assumed terminated by process death: %+v", rep.Containers)
	}
	if rep.Status != StatusCleanupFailed {
		t.Fatalf("daemonless container cleanup must be cleanup-failed: %s", rep.Status)
	}
	found := false
	for _, d := range rep.Diagnostics {
		if d.Code == CodeCustodyError {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected custody diagnostic: %+v", rep.Diagnostics)
	}
}

func TestLedgerNotCleanupAuthority(t *testing.T) {
	s, cleanup := startTestBroker(t, Limits{})
	defer cleanup()
	sentinelCmd := exec.Command("/bin/sleep", "120")
	if err := sentinelCmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer sentinelCmd.Wait()
	defer sentinelCmd.Process.Kill()
	attached, err := s.Attach(trueCommand())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Run(mustCtx(), attached, Streams{}); err != nil {
		t.Fatal(err)
	}
	forge := map[string]any{"schema": LedgerDomain, "jobs": []map[string]any{{"job": "fake", "pgid": sentinelCmd.Process.Pid, "registered": true}}}
	b, _ := json.Marshal(forge)
	if err := os.WriteFile(filepath.Join(s.dir(), "ledger.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rep, err := s.Close(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Status != StatusCleaned {
		t.Fatalf("forged ledger must not poison cleanup: %+v", rep)
	}
	if !processAlive(sentinelCmd.Process.Pid) {
		t.Fatal("historic ledger PID must never be used as cleanup authority")
	}
}
