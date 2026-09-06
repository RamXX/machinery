package processscope

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

const (
	InternalMarker          = "__machinery_internal__"
	EnvInternalCtl          = "MACHINERY_INTERNAL_CTL"
	EnvInternalOwner        = "MACHINERY_INTERNAL_OWNER"
	EnvInternalParent       = "MACHINERY_INTERNAL_PARENT"
	EnvInternalDeadline     = "MACHINERY_INTERNAL_DEADLINE"
	EnvCapability           = "MACHINERY_PROCESSSCOPE_CAP"
	EnvChildRequest         = "MACHINERY_PROCESSSCOPE_CHILD"
	EnvAttachment           = "MACHINERY_PROCESSSCOPE_ATTACHMENT"
	AttachmentDomain        = "machinery.processscope.attachment/v1"
	CommandDomain           = "machinery.processscope.command/v1"
	LedgerDomain            = "machinery.processscope.ledger/v1"
	StatusCleaned           = "cleaned"
	StatusCleanupFailed     = "cleanup-failed"
	CodeInvalidSchema       = "INVALID_SCHEMA"
	CodeInvalidOptions      = "INVALID_OPTIONS"
	CodeInvalidCommand      = "INVALID_COMMAND"
	CodeUnsupportedPlatform = "UNSUPPORTED_PLATFORM"
	CodeUnsupportedFeature  = "UNSUPPORTED_FEATURE"
	CodeAuthFailed          = "AUTH_FAILED"
	CodeStaleCapability     = "STALE_CAPABILITY"
	CodeScopeClosed         = "SCOPE_CLOSED"
	CodeRegistrationFailed  = "REGISTRATION_FAILED"
	CodeBudgetExhausted     = "BUDGET_EXHAUSTED"
	CodeTimeout             = "TIMEOUT"
	CodeOutputLimit         = "OUTPUT_LIMIT"
	CodeCanceled            = "CANCELED"
	CodeCustodyError        = "CUSTODY_ERROR"
	CodeInternalError       = "INTERNAL_ERROR"
	RoleRoot                = "root"
	RoleJoined              = "joined"
	RoleBroker              = "broker"
	RoleGuardian            = "guardian"
	drainGraceMS            = int64(1250)
	authWindow              = 5 * time.Second
	frameLimit              = 1 << 20
	hardReapWindow          = 2 * time.Second
	diagLimit               = 4096
)

type Error struct {
	Code    string
	Subject string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("processscope: %s: %s: %s", e.Code, e.Subject, e.Message)
}

func errf(code, subject, format string, a ...any) *Error {
	return &Error{Code: code, Subject: subject, Message: fmt.Sprintf(format, a...)}
}

func errFromContext(ctx context.Context, subject string) *Error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errf(CodeTimeout, subject, "context deadline exceeded")
	}
	return errf(CodeCanceled, subject, "context canceled")
}

type Diagnostic struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type Limits struct {
	WallMS      int64 `json:"wall_ms"`
	CleanupMS   int64 `json:"cleanup_ms"`
	StdoutBytes int64 `json:"stdout_bytes"`
	StderrBytes int64 `json:"stderr_bytes"`
	EventBytes  int64 `json:"event_bytes"`
	EventCount  int64 `json:"event_count"`
	Jobs        int64 `json:"jobs"`
	BundleBytes int64 `json:"bundle_bytes"`
	Entries     int64 `json:"entries"`
	Depth       int64 `json:"depth"`
}

var limitsDefaults = Limits{
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

var limitsCaps = Limits{
	WallMS:      3600000,
	CleanupMS:   30000,
	StdoutBytes: 67108864,
	StderrBytes: 67108864,
	EventBytes:  67108864,
	EventCount:  1000000,
	Jobs:        4,
	BundleBytes: 4294967296,
	Entries:     1000000,
	Depth:       128,
}

func NormalizeLimits(l Limits) (Limits, error) {
	return Limits{}, errf(CodeInvalidSchema, "limits", "not implemented")
}

func RemainingWallMS(now, deadline time.Time) int64 {
	return 0
}

func EffectiveDeadlineMS(now time.Time, cmdDeadlineMS int64, ctx context.Context, scopeDeadline time.Time) int64 {
	return 0
}

func NextCleanupDeadline(now time.Time, armed bool, current time.Time, cleanupMS int64) time.Time {
	return time.Time{}
}

type Command struct {
	Executable   string
	Args         []string
	Dir          string
	Env          []string
	RuntimeDigest string
	DeadlineMS   int64
}

type Streams struct {
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	StdoutLimit int64
	StderrLimit int64
}

type Result struct {
	JobID     string
	Started   bool
	Completed bool
	ExitCode  int
	Signal    string
	Cleanup   CleanupReport
}

type Options struct {
	HelperExecutable string
	HelperDigest     string
	ScratchRoot      string
	Limits           Limits
	OwnerLiveness    *os.File
}

type ResourceState struct {
	ID         string `json:"id"`
	Registered bool   `json:"registered"`
	Terminated bool   `json:"terminated"`
	Reaped     bool   `json:"reaped"`
}

type CleanupReport struct {
	Status      string          `json:"status"`
	Jobs        []ResourceState `json:"jobs"`
	Containers  []ResourceState `json:"containers"`
	Diagnostics []Diagnostic    `json:"diagnostics"`
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func isHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func fileDigest(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func commandDigest(cmd Command) string {
	return ""
}

type attachment struct {
	Nonce    string
	Digest   string
	Deadline int64
	Cap      bool
}

func stripReservedEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, EnvAttachment+"=") {
			continue
		}
		out = append(out, e)
	}
	return out
}

func hasChildRequest(env []string) bool {
	n := 0
	for _, e := range env {
		if e == EnvChildRequest+"=join" {
			n++
		}
	}
	return n == 1
}

func encodeAttachment(a attachment) string {
	c := "0"
	if a.Cap {
		c = "1"
	}
	return fmt.Sprintf("v1;%s;%s;%d;%s", a.Nonce, a.Digest, a.Deadline, c)
}

func parseAttachment(v string) (attachment, bool) {
	parts := strings.Split(v, ";")
	if len(parts) != 5 || parts[0] != "v1" {
		return attachment{}, false
	}
	if !isHex64(parts[1]) || !isHex64(parts[2]) {
		return attachment{}, false
	}
	d, err := strconvParseInt(parts[3])
	if err != nil || d <= 0 {
		return attachment{}, false
	}
	if parts[4] != "0" && parts[4] != "1" {
		return attachment{}, false
	}
	return attachment{Nonce: parts[1], Digest: parts[2], Deadline: d, Cap: parts[4] == "1"}, true
}

func strconvParseInt(s string) (int64, error) {
	var n int64
	if s == "" {
		return 0, errors.New("empty")
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, errors.New("not decimal")
		}
		n = n*10 + int64(c-'0')
		if n < 0 {
			return 0, errors.New("overflow")
		}
	}
	return n, nil
}

type internalClaim struct {
	ctl      int
	owner    int
	parent   int
	deadline time.Time
}

func parseInternalClaim(env func(string) string) (internalClaim, bool) {
	var c internalClaim
	raw := env(EnvInternalCtl)
	if raw == "" {
		return c, false
	}
	n, err := strconvParseInt(raw)
	if err != nil || n < 3 {
		return c, true
	}
	c.ctl = int(n)
	if raw := env(EnvInternalOwner); raw != "" {
		if n, err := strconvParseInt(raw); err == nil && n >= 3 {
			c.owner = int(n)
		}
	}
	if raw := env(EnvInternalParent); raw != "" {
		if n, err := strconvParseInt(raw); err == nil && n >= 3 {
			c.parent = int(n)
		}
	}
	if raw := env(EnvInternalDeadline); raw != "" {
		if n, err := strconvParseInt(raw); err == nil && n > 0 {
			c.deadline = time.Unix(0, n)
		}
	}
	return c, true
}

type capabilityClaim struct {
	fd       int
	scopeID  string
	rootID   string
	deadline time.Time
}

func parseCapabilityClaim(env func(string) string) (capabilityClaim, bool) {
	var c capabilityClaim
	raw := env(EnvCapability)
	if raw == "" {
		return c, false
	}
	for _, kv := range strings.Split(raw, ";") {
		i := strings.IndexByte(kv, '=')
		if i < 0 {
			continue
		}
		k, v := kv[:i], kv[i+1:]
		switch k {
		case "fd":
			if n, err := strconvParseInt(v); err == nil && n >= 3 {
				c.fd = int(n)
			}
		case "scope":
			c.scopeID = v
		case "root":
			c.rootID = v
		case "deadline":
			if n, err := strconvParseInt(v); err == nil && n > 0 {
				c.deadline = time.Unix(0, n)
			}
		}
	}
	return c, true
}

func encodeCapabilityEnv(fd int, scopeID, rootID string, deadline time.Time) string {
	return fmt.Sprintf("fd=%d;scope=%s;root=%s;deadline=%d", fd, scopeID, rootID, deadline.UnixNano())
}

const (
	handleUnused   = int32(0)
	handleConsumed = int32(1)
	handleClosed   = int32(2)
)

type ioState struct {
	ctl      *os.File
	owner    *os.File
	parent   *os.File
	deadline time.Time
	diag     io.Writer
	status   atomic.Int32
}

type InternalIO struct {
	state *ioState
}

func (io InternalIO) Close() error {
	if io.state == nil {
		return nil
	}
	if !io.state.status.CompareAndSwap(handleUnused, handleClosed) {
		return nil
	}
	var err1, err2, err3 error
	if io.state.ctl != nil {
		err1 = io.state.ctl.Close()
	}
	if io.state.owner != nil {
		err2 = io.state.owner.Close()
	}
	if io.state.parent != nil {
		err3 = io.state.parent.Close()
	}
	return errors.Join(err1, err2, err3)
}

func (io InternalIO) consume() bool {
	return io.state != nil && io.state.status.CompareAndSwap(handleUnused, handleConsumed)
}

func (io InternalIO) writeDiag(s string) {
	if io.state == nil || io.state.diag == nil {
		return
	}
	if len(s) > diagLimit {
		s = s[:diagLimit]
	}
	fmt.Fprint(io.state.diag, s)
}

type capState struct {
	conn     *os.File
	scopeID  string
	rootID   string
	deadline time.Time
	status   atomic.Int32
}

type Capability struct {
	state *capState
}

func (c Capability) Close() error {
	if c.state == nil {
		return nil
	}
	if !c.state.status.CompareAndSwap(handleUnused, handleClosed) {
		return nil
	}
	return c.state.conn.Close()
}

func (c Capability) consume() bool {
	return c.state != nil && c.state.status.CompareAndSwap(handleUnused, handleConsumed)
}
