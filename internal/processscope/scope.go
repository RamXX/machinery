package processscope

import (
	"context"
	"net"
	"os"
	"os/exec"
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
	logFile   *os.File
	closed    bool
}

type scope struct {
	mu       sync.Mutex
	conn     *net.UnixConn
	wmu      sync.Mutex
	id       string
	rootID   string
	deadline time.Time
	limits   Limits
	isRoot   bool
	root     *rootHandle
	closed   bool
	report   CleanupReport
}

func (s *scope) Run(ctx context.Context, cmd Command, streams Streams) (Result, error) {
	return Result{}, errf(CodeInternalError, "run", "not implemented")
}

func (s *scope) Child(ctx context.Context) (Scope, error) {
	return nil, errf(CodeInternalError, "child", "not implemented")
}

func (s *scope) Attach(cmd Command) (Command, error) {
	return Command{}, errf(CodeInternalError, "attach", "not implemented")
}

func (s *scope) Close(ctx context.Context) (CleanupReport, error) {
	return CleanupReport{}, errf(CodeInternalError, "close", "not implemented")
}

func (s *scope) registerContainer(ctx context.Context, id, daemon string) error {
	return errf(CodeInternalError, "registerContainer", "not implemented")
}

func (s *scope) dir() string {
	if s.root != nil {
		return s.root.dir
	}
	return ""
}

func Open(ctx context.Context, opts Options) (Scope, error) {
	return nil, errf(CodeInternalError, "open", "not implemented")
}

func Join(ctx context.Context, cap Capability) (Scope, error) {
	return nil, errf(CodeInternalError, "join", "not implemented")
}

func ServeInternal(args []string, io InternalIO) (handled bool, exitCode int) {
	return false, 0
}

func InheritedInternalIO(ctx context.Context) (InternalIO, bool, error) {
	return InternalIO{}, false, nil
}

func InheritedCapability(ctx context.Context) (Capability, error) {
	return Capability{}, errf(CodeCustodyError, "inherited-capability", "absent required attachment")
}
