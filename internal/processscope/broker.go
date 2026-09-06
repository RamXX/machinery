package processscope

import (
	"net"
	"os"
	"sync"
	"time"
)

type scopeNode struct {
	id      string
	parent  *scopeNode
	closing bool
}

type job struct {
	id          string
	scope       *scopeNode
	guardianPID int
	pgid        int
	registered  bool
	terminated  bool
	reaped      bool
	started     bool
	completed   bool
	terminal    bool
	exitCode    int
	signal      string
	retired     bool
	guardian    *os.Process
	gconn       *net.UnixConn
	client      *net.UnixConn
	resultCh    chan struct{}
}

type containerRec struct {
	id     string
	daemon string
	scope  *scopeNode
}

type broker struct {
	mu             sync.Mutex
	rootID         string
	dir            string
	limits         Limits
	wallDeadline   time.Time
	cleanupArmed   bool
	cleanupPending time.Time
	scopes         map[string]*scopeNode
	rootNode       *scopeNode
	jobs           map[string]*job
	jobOrder       []string
	containers     map[string]*containerRec
	nonces         map[string]bool
	quitting       bool
}

var hookAuthorizeRequest = func(bound, claimed string) error {
	return errf(CodeAuthFailed, "authorize", "claim does not match channel binding")
}

var hookRegisterJob = func(b *broker, j *job) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.jobs[j.id] = j
	b.jobOrder = append(b.jobOrder, j.id)
	return nil
}

var hookGroupSignal = func(pgid int, sig int) error {
	return signalGroup(pgid, sig)
}

type msgBootstrap struct {
	T         string `json:"t"`
	Challenge string `json:"challenge"`
	Digest    string `json:"digest"`
	Deadline  int64  `json:"deadline"`
	Limits    Limits `json:"limits"`
}

type msgReady struct {
	T     string `json:"t"`
	Root  string `json:"root"`
	Ack   string `json:"ack"`
	Self  string `json:"self"`
}

type msgHello struct {
	T     string `json:"t"`
	Scope string `json:"scope"`
	Role  string `json:"role"`
}

type msgWelcome struct {
	T        string `json:"t"`
	Root     string `json:"root"`
	Deadline int64  `json:"deadline"`
}

type msgAttach struct {
	T        string `json:"t"`
	Scope    string `json:"scope"`
	Digest   string `json:"digest"`
	Deadline int64  `json:"deadline"`
	Cap      bool   `json:"cap"`
}

type msgAttached struct {
	T     string `json:"t"`
	Nonce string `json:"nonce"`
}

type msgRun struct {
	T      string  `json:"t"`
	Scope  string  `json:"scope"`
	Nonce  string  `json:"nonce"`
	Digest string  `json:"digest"`
	Spec   jobSpec `json:"spec"`
}

type jobSpec struct {
	Executable   string   `json:"exe"`
	Args         []string `json:"args"`
	Dir          string   `json:"dir"`
	Env          []string `json:"env"`
	RuntimeDigest string  `json:"runtime"`
	DeadlineMS   int64    `json:"deadlineMS"`
	DrainGraceMS int64    `json:"drainGraceMS"`
	JobID        string   `json:"job"`
	RootID       string   `json:"root"`
	ScopeID      string   `json:"scope"`
	FDStdin      bool     `json:"fdstdin"`
	FDStdout     bool     `json:"fdstdout"`
	FDStderr     bool     `json:"fdstderr"`
	FDCap        bool     `json:"fdcap"`
	CapEnv       string   `json:"capenv"`
}

type msgStarted struct {
	T   string `json:"t"`
	Job string `json:"job"`
	Pid int    `json:"pid"`
}

type msgRefused struct {
	T       string `json:"t"`
	Code    string `json:"code"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type msgResult struct {
	T         string        `json:"t"`
	Job       string        `json:"job"`
	Started   bool          `json:"started"`
	Completed bool          `json:"completed"`
	ExitCode  int           `json:"exit"`
	Signal    string        `json:"signal"`
	Code      string        `json:"code"`
	Cleanup   CleanupReport `json:"cleanup"`
}

type msgCancel struct {
	T      string `json:"t"`
	Scope  string `json:"scope"`
	Job    string `json:"job"`
	Reason string `json:"reason"`
}

type msgClose struct {
	T     string `json:"t"`
	Scope string `json:"scope"`
}

type msgClosed struct {
	T      string        `json:"t"`
	Report CleanupReport `json:"report"`
}

type msgChildReq struct {
	T     string `json:"t"`
	Scope string `json:"scope"`
}

type msgChildOK struct {
	T     string `json:"t"`
	Scope string `json:"scope"`
}

type msgRegContainer struct {
	T      string `json:"t"`
	Scope  string `json:"scope"`
	ID     string `json:"id"`
	Daemon string `json:"daemon"`
}

type msgJob struct {
	T    string  `json:"t"`
	Spec jobSpec `json:"spec"`
}

type msgGStarted struct {
	T   string `json:"t"`
	Pid int    `json:"pid"`
}

type msgGStartFail struct {
	T       string `json:"t"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type msgGExit struct {
	T         string `json:"t"`
	Completed bool   `json:"completed"`
	ExitCode  int    `json:"exit"`
	Signal    string `json:"signal"`
	Terminal  bool   `json:"terminal"`
}

type ledgerJob struct {
	Job        string `json:"job"`
	Scope      string `json:"scope"`
	Guardian   int    `json:"guardian"`
	PGID       int    `json:"pgid"`
	Registered bool   `json:"registered"`
	Terminated bool   `json:"terminated"`
	Reaped     bool   `json:"reaped"`
}

type ledgerContainer struct {
	ID         string `json:"id"`
	Registered bool   `json:"registered"`
	Terminated bool   `json:"terminated"`
	Reaped     bool   `json:"reaped"`
}

type ledgerRecord struct {
	Schema        string            `json:"schema"`
	Root          string            `json:"root"`
	WallDeadline  int64             `json:"wallDeadline"`
	CleanupArmed  int64             `json:"cleanupDeadline"`
	FinishedAt    int64             `json:"finishedAt"`
	Status        string            `json:"status"`
	Jobs          []ledgerJob       `json:"jobs"`
	Containers    []ledgerContainer `json:"containers"`
}

func runBroker(io InternalIO, args []string) int {
	return 1
}
