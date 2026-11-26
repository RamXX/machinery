package evidence

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
)

// RunOptions configures Run.
type RunOptions struct {
	Stack string
	// Env holds extra KEY=VALUE variables for the command.
	Env []string
	// Only lists the ids exported as MACHINERY_ONLY (a targeted run).
	Only []string
	// ReceiptsFile, when set, is written instead of a new file in the receipts
	// directory, and no receipts file is deleted. The probe uses it.
	ReceiptsFile   string
	Stdout, Stderr io.Writer
	// Adapter, when set, reads the command's standard output and gives the
	// receipts machinery writes itself (section 10.3).
	Adapter RunAdapter
	// Streams: the adapter does not read standard output; it reads the
	// files the command leaves in a directory machinery makes and names in
	// MACHINERY_GO_TEST_JSON.
	Streams bool
}

// RunAdapter writes receipts from what a test command prints.
type RunAdapter interface {
	// Name is recorded in the run header.
	Name() string
	// Consume reads the command's standard output to its end and prints
	// what the user should see of it.
	Consume(stream io.Reader, out io.Writer) error
	// Receipts returns the receipt lines to write, once the command is done.
	Receipts() [][]byte
	// ReadStreams reads the stream files of a directory instead of Consume.
	ReadStreams(dir string) error
}

// ExitNotStarted is the exit code when the command could not be started.
const ExitNotStarted = 127

// The variables machinery owns; inherited values are dropped.
var ownVars = []string{"MACHINERY_RECEIPTS", "MACHINERY_COMMIT", "MACHINERY_ONLY", "MACHINERY_TABLES_DIR", "MACHINERY_PROBE_CASE", StreamsEnv}

type header struct {
	V       int      `json:"v"`
	Type    string   `json:"type"`
	Commit  string   `json:"commit"`
	Dirty   bool     `json:"dirty"`
	Command []string `json:"command"`
	Stack   string   `json:"stack"`
	Started string   `json:"started"`
	Adapter string   `json:"adapter,omitempty"`
}

type endLine struct {
	V        int    `json:"v"`
	Type     string `json:"type"`
	Exit     int    `json:"exit"`
	Finished string `json:"finished"`
}

func jsonLine(v any) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return buf.Bytes()
}

// Run implements section 10: it records a receipts file around one run of
// argv and returns the command's exit code and the file written (absolute).
func Run(p *project.Project, opts RunOptions, argv []string) (exit int, file string, err error) {
	if len(argv) == 0 {
		return 2, "", errors.New("no command to run")
	}
	head, err := gitx.Head(p.Root)
	if err != nil {
		return 1, "", fmt.Errorf("cannot read the current commit (machinery run needs a git repository with a commit): %v", err)
	}
	dirty, err := TreeDirty(p)
	if err != nil {
		return 1, "", err
	}
	now := time.Now().UTC()
	if opts.ReceiptsFile != "" {
		if file, err = filepath.Abs(opts.ReceiptsFile); err != nil {
			return 1, "", err
		}
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			return 1, "", err
		}
	} else {
		if err := pruneStale(p, head); err != nil {
			return 1, "", err
		}
		dir := p.Abs(Dir(p))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return 1, "", err
		}
		file, err = newFileName(dir, now, gitx.Short(head))
		if err != nil {
			return 1, "", err
		}
	}
	f, err := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 1, "", err
	}
	h := header{V: 1, Type: "run", Commit: head, Dirty: dirty, Command: argv,
		Stack: opts.Stack, Started: now.Format(time.RFC3339)}
	if opts.Adapter != nil {
		h.Adapter = opts.Adapter.Name()
	}
	head1 := jsonLine(h)
	_, err = f.Write(head1)
	f.Close()
	if err != nil {
		return 1, file, err
	}
	// the ledger binds the file to this run (section 10.2); a probe run is not evidence
	ledger := opts.ReceiptsFile == ""
	if ledger {
		if err := writeEntry(p, file, ledgerEntry{Commit: head, Dir: ledgerDir(p)}); err != nil {
			return 1, file, err
		}
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = p.Root
	cmd.Stderr = opts.Stderr
	if opts.Adapter == nil {
		cmd.Stdout = opts.Stdout
	}
	env := []string{}
	for _, kv := range os.Environ() {
		if !isOwn(kv) {
			env = append(env, kv)
		}
	}
	env = append(env, "MACHINERY_RECEIPTS="+file, "MACHINERY_COMMIT="+head)
	if len(opts.Only) > 0 {
		env = append(env, "MACHINERY_ONLY="+strings.Join(opts.Only, ","))
	}
	cmd.Env = append(env, opts.Env...)
	var stream io.ReadCloser
	streamsDir := ""
	if opts.Adapter != nil && opts.Streams {
		if streamsDir, err = os.MkdirTemp("", "machinery-streams-"); err != nil {
			os.Remove(file)
			if ledger {
				forget(p, file)
			}
			return 1, "", err
		}
		defer os.RemoveAll(streamsDir) // machinery made it; it is gone after reading
		cmd.Env = append(cmd.Env, StreamsEnv+"="+streamsDir)
		cmd.Stdout = opts.Stdout
	} else if opts.Adapter != nil {
		if stream, err = cmd.StdoutPipe(); err != nil {
			os.Remove(file)
			if ledger {
				forget(p, file)
			}
			return 1, "", err
		}
	}
	if err := cmd.Start(); err != nil {
		os.Remove(file)
		if ledger {
			forget(p, file)
		}
		return ExitNotStarted, "", fmt.Errorf("cannot start %s: %v", argv[0], err)
	}
	var streamErr error
	if stream != nil {
		streamErr = opts.Adapter.Consume(stream, opts.Stdout)
		io.Copy(io.Discard, stream) // never leave the command blocked on a full pipe
	}
	exit = exitCode(cmd.Wait())
	if streamsDir != "" {
		if err := opts.Adapter.ReadStreams(streamsDir); err != nil {
			// nothing of this run counts: no receipt, no end line (section 10.3)
			if exit == 0 {
				exit = 1
			}
			return exit, file, err
		}
	}
	f, err = os.OpenFile(file, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return exit, file, err
	}
	if opts.Adapter != nil && streamErr == nil {
		// the adapter's receipts go in before the end line, each in one write
		for _, l := range opts.Adapter.Receipts() {
			if _, err = f.Write(l); err != nil {
				f.Close()
				return exit, file, err
			}
		}
	}
	_, err = f.Write(jsonLine(endLine{V: 1, Type: "end", Exit: exit, Finished: time.Now().UTC().Format(time.RFC3339)}))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && ledger {
		err = finishEntry(p, file, ledgerEntry{Commit: head, Dir: ledgerDir(p)})
	}
	if err == nil && streamErr != nil {
		err = fmt.Errorf("reading the event stream: %v", streamErr)
	}
	return exit, file, err
}

func isOwn(kv string) bool {
	for _, k := range ownVars {
		if strings.HasPrefix(kv, k+"=") {
			return true
		}
	}
	return false
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal())
		}
		return ee.ExitCode()
	}
	return 1
}

// newFileName picks <UTC timestamp>-<short sha>.jsonl, unique in dir. The
// timestamp has millisecond precision so that names sort by start time.
func newFileName(dir string, now time.Time, short string) (string, error) {
	for i := 0; i < 1000; i++ {
		t := now.Add(time.Duration(i) * time.Millisecond)
		name := filepath.Join(dir, t.Format("20060102T150405.000Z")+"-"+short+".jsonl")
		f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			f.Close()
			return name, nil
		}
		if !os.IsExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("cannot pick a receipts file name in %s", dir)
}

// pruneStale deletes receipts files that are not valid for the head (step 2
// of section 10).
func pruneStale(p *project.Project, head string) error {
	if err := forgetGone(p); err != nil {
		return err
	}
	files, err := receiptFiles(p)
	if err != nil {
		return err
	}
	v := newValidity(p, head)
	for _, rel := range files {
		fd, err := parseFile(p.Abs(rel), rel)
		if err != nil {
			return err
		}
		if fd.Header == nil || !v.valid(fd.Header.Commit) {
			if err := os.Remove(p.Abs(rel)); err != nil {
				return err
			}
			forget(p, rel)
		}
	}
	return nil
}
