package runtimeclosure

// The exact approved Go runtime closure of the first-release catalog
// (docs/test-assurance-contract.md section 6): go-testing/v1 executes only
// under Go 1.27.1 on the pinned native platforms darwin/arm64 and
// linux/amd64. OpenGo binds the complete GOROOT tree by exact bytes without
// launching anything; Validate performs the runtime's own identity probes
// (go version, go env GOROOT) under a live processscope scope; Close is the
// final process-free byte/topology/root-identity revalidation. There is no
// user-selectable adapter runtime API.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/tdd"
)

const (
	// GoProfile is the runtime profile identity of the Go toolchain closure.
	GoProfile = "go"
	// RequiredGoVersion is the exact first-release Go toolchain version.
	RequiredGoVersion = "1.27.1"

	goClosureDomain     = "machinery.runtime.go/v1"
	goToolchainMaxBytes = int64(256 << 20)
	goProbeDeadlineMS   = int64(30000)
	goProbeOutputLimit  = int64(1 << 20)
	goTreeEntries       = int64(1000000)
	goTreeDepth         = int64(128)
	goTreeBytes         = int64(4 << 30)
)

// GoRequest is the closed open request of the pinned Go runtime closure.
type GoRequest struct {
	// RuntimeRoot names the GOROOT directory that owns bin/go. Empty
	// resolves the host toolchain from PATH through its real symlink chain.
	RuntimeRoot string
	// ExpectedClosure, when nonempty, is an exact lowercase sha256 the
	// opened closure must reproduce; a mismatch fails closed.
	ExpectedClosure string
}

// Go is the pinned Go 1.27.1 runtime handle implementing tdd.RuntimeHandle.
type Go struct {
	source      string // the caller-supplied or PATH-discovered launcher path
	binPath     string // resolved absolute real path of bin/go
	rootPath    string // resolved absolute real GOROOT
	rootInfo    os.FileInfo
	binInfo     os.FileInfo
	binFile     *os.File
	root        *os.Root
	binBody     []byte
	closureHash [sha256.Size]byte
	identity    tdd.RuntimeRef
	closed      bool
}

// Compile-time contract check: the closed RuntimeHandle boundary.
var _ tdd.RuntimeHandle = (*Go)(nil)

// OpenGo opens and binds the pinned Go runtime closure. It never launches a
// process: the version is pre-bound from the toolchain's own VERSION file,
// the launcher bytes are retained under an open root handle, and the
// complete GOROOT tree is fingerprinted with the closure digest.
func OpenGo(ctx context.Context, req GoRequest) (*Go, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: cannot open the pinned Go %s runtime closure: %w", RequiredGoVersion, err)
	}
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if platform != "darwin/arm64" && platform != "linux/amd64" {
		return nil, fmt.Errorf("UNSUPPORTED_PLATFORM: %s is not a pinned native assurance platform for the Go runtime closure", platform)
	}
	source := req.RuntimeRoot
	explicit := source != ""
	if !explicit {
		discovered, err := exec.LookPath("go")
		if err != nil {
			return nil, fmt.Errorf("UNSUPPORTED_VERSION: the pinned Go %s runtime is absent from PATH: %w", RequiredGoVersion, err)
		}
		source = discovered
	}
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return nil, err
	}
	var rootPath string
	if explicit {
		rootPath, err = filepath.EvalSymlinks(sourceAbs)
		if err != nil {
			return nil, fmt.Errorf("UNSUPPORTED_VERSION: the declared Go runtime root %s is unavailable: %w", sourceAbs, err)
		}
	} else {
		realBin, err := filepath.EvalSymlinks(sourceAbs)
		if err != nil {
			return nil, fmt.Errorf("UNSUPPORTED_VERSION: the discovered go launcher %s is unavailable: %w", sourceAbs, err)
		}
		rootPath = filepath.Dir(filepath.Dir(realBin))
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: cannot open the Go runtime root %s: %w", rootPath, err)
	}
	fail := func(err error) (*Go, error) {
		return nil, errors.Join(err, root.Close())
	}
	rootInfo, err := os.Lstat(rootPath)
	if err != nil || !rootInfo.IsDir() {
		return fail(fmt.Errorf("UNSUPPORTED_VERSION: the Go runtime root %s must be a real directory", rootPath))
	}
	openedRoot, err := root.Lstat(".")
	if err != nil || !os.SameFile(rootInfo, openedRoot) {
		return fail(fmt.Errorf("STALE_INPUT: the Go runtime root %s changed identity while opening", rootPath))
	}
	binRel := filepath.ToSlash(filepath.Join("bin", "go"))
	binStat, err := root.Lstat(binRel)
	if err != nil || binStat.Mode()&os.ModeSymlink != 0 || !binStat.Mode().IsRegular() || binStat.Mode().Perm()&0111 == 0 {
		return fail(fmt.Errorf("UNSUPPORTED_VERSION: %s must be a regular executable bin/go of the Go runtime root", rootPath))
	}
	binFile, err := root.Open(binRel)
	if err != nil {
		return fail(fmt.Errorf("UNSUPPORTED_VERSION: cannot retain the Go launcher: %w", err))
	}
	binInfo, err := binFile.Stat()
	if err != nil || !binInfo.Mode().IsRegular() || binInfo.Size() > goToolchainMaxBytes {
		return fail(errors.Join(fmt.Errorf("UNSUPPORTED_VERSION: the retained Go launcher is not a bounded regular file"), binFile.Close()))
	}
	binBody, err := io.ReadAll(io.LimitReader(binFile, binInfo.Size()+1))
	if err != nil {
		return fail(errors.Join(fmt.Errorf("UNSUPPORTED_VERSION: cannot read the Go launcher bytes: %w", err), binFile.Close()))
	}
	if int64(len(binBody)) != binInfo.Size() {
		return fail(errors.Join(fmt.Errorf("STALE_INPUT: the Go launcher changed size while reading"), binFile.Close()))
	}
	binPath := filepath.Join(rootPath, "bin", "go")
	if !explicit {
		resolved, err := filepath.EvalSymlinks(sourceAbs)
		if err != nil || resolved != binPath {
			return fail(errors.Join(fmt.Errorf("STALE_INPUT: the go launcher symlink chain changed after verification"), binFile.Close()))
		}
	}
	versionBytes, err := root.ReadFile("VERSION")
	if err != nil || len(versionBytes) > 4096 {
		return fail(errors.Join(fmt.Errorf("UNSUPPORTED_VERSION: the Go runtime root %s carries no readable VERSION identity", rootPath), binFile.Close()))
	}
	versionLine, _, _ := strings.Cut(string(versionBytes), "\n")
	if versionLine != "go"+RequiredGoVersion {
		return fail(errors.Join(fmt.Errorf("UNSUPPORTED_VERSION: the Go runtime root %s is %q, want exactly go%s", rootPath, versionLine, RequiredGoVersion), binFile.Close()))
	}
	if srcStat, err := root.Lstat("src"); err != nil || !srcStat.IsDir() {
		return fail(errors.Join(fmt.Errorf("UNSUPPORTED_VERSION: the Go runtime root %s lacks its src/ toolchain tree", rootPath), binFile.Close()))
	}
	closureHash, fpErr := fingerprintGoRoot(root)
	if fpErr != nil {
		return fail(errors.Join(fmt.Errorf("UNSUPPORTED_VERSION: cannot fingerprint the Go runtime closure: %w", fpErr), binFile.Close()))
	}
	if req.ExpectedClosure != "" {
		got := fmt.Sprintf("%x", closureHash)
		if len(req.ExpectedClosure) != sha256.Size*2 || req.ExpectedClosure != strings.ToLower(req.ExpectedClosure) {
			return fail(errors.Join(fmt.Errorf("INVALID_SCHEMA: GoRequest.ExpectedClosure requires an exact lowercase sha256"), binFile.Close()))
		}
		if req.ExpectedClosure != got {
			return fail(errors.Join(fmt.Errorf("UNSUPPORTED_VERSION: explicit Go runtime closure sha256:%s does not match the pinned trust root sha256:%s", got, req.ExpectedClosure), binFile.Close()))
		}
	}
	return &Go{
		source: sourceAbs, binPath: binPath, rootPath: rootPath,
		rootInfo: rootInfo, binInfo: binInfo, binFile: binFile, root: root,
		binBody: binBody, closureHash: closureHash,
		identity: tdd.RuntimeRef{
			Profile: GoProfile, Version: RequiredGoVersion,
			Platform: platform, Closure: "sha256:" + fmt.Sprintf("%x", closureHash),
		},
	}, nil
}

// Identity returns the pinned runtime identity of the opened closure.
func (g *Go) Identity() tdd.RuntimeRef { return g.identity }

// Binary returns the resolved absolute go executable of the closure.
func (g *Go) Binary() string { return g.binPath }

// Validate performs the runtime's own identity probes under the supplied
// live custody scope (never unowned), then revalidates the closure bytes.
// A missing scope, a wrong native version or a disagreeing GOROOT fails
// closed before any suite work happens on this handle.
func (g *Go) Validate(ctx context.Context, scope processscope.Scope) error {
	if g.closed {
		return fmt.Errorf("STALE_INPUT: the Go runtime handle is already closed")
	}
	if scope == nil {
		return fmt.Errorf("INVALID_SCHEMA: Go runtime validation requires a live custody scope")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("UNSUPPORTED_VERSION: go runtime validation canceled: %w", err)
	}
	versionOut, err := g.probe(ctx, scope, "version")
	if err != nil {
		return err
	}
	wantVersion := fmt.Sprintf("go version go%s %s/%s", RequiredGoVersion, runtime.GOOS, runtime.GOARCH)
	if versionOut != wantVersion {
		return fmt.Errorf("UNSUPPORTED_VERSION: native go version probe reported %q, want %q", versionOut, wantVersion)
	}
	rootOut, err := g.probe(ctx, scope, "env", "GOROOT")
	if err != nil {
		return err
	}
	reported, err := filepath.EvalSymlinks(strings.TrimSpace(rootOut))
	if err != nil || reported != g.rootPath {
		return fmt.Errorf("UNSUPPORTED_VERSION: native go env GOROOT reported %q, want the verified runtime root %q", strings.TrimSpace(rootOut), g.rootPath)
	}
	return g.revalidate()
}

// probe runs one bounded launcher invocation as its own guarded child scope
// of the supplied live scope: the child is closed with verified terminal
// retirement before returning, so completed guardians never leak against
// the scope's concurrent job budget.
func (g *Go) probe(ctx context.Context, scope processscope.Scope, args ...string) (string, error) {
	child, err := scope.Child(ctx)
	if err != nil {
		return "", fmt.Errorf("CUSTODY_ERROR: open the probe custody child: %w", err)
	}
	var stdout, stderr bytes.Buffer
	result, runErr := func() (processscope.Result, error) {
		attached, err := child.Attach(processscope.Command{
			Executable:    g.binPath,
			Args:          args,
			Dir:           g.rootPath,
			Env:           []string{"PATH=" + filepath.Join(g.rootPath, "bin")},
			RuntimeDigest: g.identity.Closure,
			DeadlineMS:    goProbeDeadlineMS,
		})
		if err != nil {
			return processscope.Result{}, fmt.Errorf("CUSTODY_ERROR: attach go identity probe: %w", err)
		}
		return child.Run(ctx, attached, processscope.Streams{
			Stdout: &stdout, Stderr: &stderr,
			StdoutLimit: goProbeOutputLimit, StderrLimit: goProbeOutputLimit,
		})
	}()
	closeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	report, closeErr := child.Close(closeCtx)
	if closeErr != nil || report.Status != processscope.StatusCleaned {
		return "", errors.Join(runErr, closeErr, fmt.Errorf("CUSTODY_ERROR: probe custody child cleanup did not verify: %+v", report))
	}
	if runErr != nil {
		return "", fmt.Errorf("CUSTODY_ERROR: go identity probe failed under custody: %w (stderr: %s)", runErr, stderr.String())
	}
	if !result.Completed || result.ExitCode != 0 || result.Signal != "" {
		return "", fmt.Errorf("UNSUPPORTED_VERSION: go identity probe exited abnormally (completed=%v exit=%d signal=%q)", result.Completed, result.ExitCode, result.Signal)
	}
	output := strings.TrimRight(stdout.String(), "\n")
	if output == "" {
		return "", fmt.Errorf("UNSUPPORTED_VERSION: go identity probe produced no identity output")
	}
	return output, nil
}

// Close performs the final process-free byte/topology/root-identity
// revalidation, then releases the retained handles. It never launches a
// subprocess and fails closed on every drift or double close.
func (g *Go) Close() error {
	if g.closed {
		return fmt.Errorf("STALE_INPUT: the Go runtime handle is already closed")
	}
	if err := g.revalidate(); err != nil {
		_ = errors.Join(g.binFile.Close(), g.root.Close())
		g.closed = true
		return err
	}
	err := errors.Join(g.binFile.Close(), g.root.Close())
	g.closed = true
	return err
}

// revalidate re-checks root identity, launcher identity and bytes, and the
// complete closure fingerprint without launching anything.
func (g *Go) revalidate() error {
	ambientRoot, ambientErr := os.Lstat(g.rootPath)
	retainedRoot, retainedErr := g.root.Lstat(".")
	if ambientErr != nil || retainedErr != nil || !ambientRoot.IsDir() || !retainedRoot.IsDir() || !os.SameFile(g.rootInfo, ambientRoot) || !os.SameFile(g.rootInfo, retainedRoot) {
		return errors.Join(ambientErr, retainedErr, fmt.Errorf("STALE_INPUT: the Go runtime root %s changed identity after verification", g.rootPath))
	}
	binRel := filepath.ToSlash(filepath.Join("bin", "go"))
	ambientBin, ambientErr := os.Lstat(g.binPath)
	rootBin, rootErr := g.root.Lstat(binRel)
	openedBin, openedErr := g.binFile.Stat()
	if ambientErr != nil || rootErr != nil || openedErr != nil ||
		!g.binInfo.ModTime().Equal(ambientBin.ModTime()) || g.binInfo.Mode() != ambientBin.Mode() || g.binInfo.Size() != ambientBin.Size() || !os.SameFile(g.binInfo, ambientBin) ||
		!g.binInfo.ModTime().Equal(rootBin.ModTime()) || g.binInfo.Mode() != rootBin.Mode() || g.binInfo.Size() != rootBin.Size() ||
		!g.binInfo.ModTime().Equal(openedBin.ModTime()) || g.binInfo.Size() != openedBin.Size() {
		return errors.Join(ambientErr, rootErr, openedErr, fmt.Errorf("STALE_INPUT: the Go launcher %s changed identity after verification", g.binPath))
	}
	if _, err := g.binFile.Seek(0, io.SeekStart); err != nil {
		return err
	}
	again, readErr := io.ReadAll(io.LimitReader(g.binFile, g.binInfo.Size()+1))
	if readErr != nil || int64(len(again)) != g.binInfo.Size() || !bytes.Equal(g.binBody, again) {
		return errors.Join(readErr, fmt.Errorf("STALE_INPUT: the Go launcher changed content after verification"))
	}
	closureHash, err := fingerprintGoRoot(g.root)
	if err != nil {
		return fmt.Errorf("STALE_INPUT: cannot re-fingerprint the Go runtime closure: %w", err)
	}
	if closureHash != g.closureHash {
		return fmt.Errorf("STALE_INPUT: the Go runtime closure changed after verification (was sha256:%x, now sha256:%x)", g.closureHash, closureHash)
	}
	return nil
}

// fingerprintGoRoot hashes the complete runtime tree topology and file
// bytes under the domain machinery.runtime.go/v1 with the shared entry,
// depth and byte bounds of the shipped custody limits.
func fingerprintGoRoot(root *os.Root) ([sha256.Size]byte, error) {
	hash := sha256.New()
	hash.Write([]byte(goClosureDomain))
	var entries, totalBytes int64
	var walk func(rel string, depth int64) error
	walk = func(rel string, depth int64) error {
		if depth > goTreeDepth {
			return fmt.Errorf("depth bound exceeded at %s", rel)
		}
		dir, err := root.Open(rel)
		if err != nil {
			return err
		}
		dirents, err := dir.ReadDir(-1)
		dir.Close()
		if err != nil {
			return err
		}
		names := make([]string, 0, len(dirents))
		byName := map[string]os.DirEntry{}
		for _, entry := range dirents {
			names = append(names, entry.Name())
			byName[entry.Name()] = entry
		}
		sort.Strings(names)
		for _, name := range names {
			entries++
			if entries > goTreeEntries {
				return fmt.Errorf("entry bound exceeded")
			}
			child := name
			if rel != "." {
				child = rel + "/" + name
			}
			entry := byName[name]
			info, err := entry.Info()
			if err != nil {
				return err
			}
			switch {
			case info.IsDir():
				writeClosureEntry(hash, child, true, 0, info.Mode().Perm(), [sha256.Size]byte{})
				if err := walk(child, depth+1); err != nil {
					return err
				}
			case info.Mode().IsRegular():
				if info.Size() > goTreeMaxFile {
					return fmt.Errorf("file %s exceeds the closure byte bound", child)
				}
				totalBytes += info.Size()
				if totalBytes > goTreeBytes {
					return fmt.Errorf("closure byte bound exceeded")
				}
				body, err := root.ReadFile(child)
				if err != nil {
					return err
				}
				sum := sha256.Sum256(body)
				writeClosureEntry(hash, child, false, info.Size(), info.Mode().Perm(), sum)
			default:
				return fmt.Errorf("closure member %s is not a regular file or directory", child)
			}
		}
		return nil
	}
	if err := walk(".", 0); err != nil {
		return [sha256.Size]byte{}, err
	}
	var out [sha256.Size]byte
	copy(out[:], hash.Sum(nil))
	return out, nil
}

const goTreeMaxFile = int64(512 << 20)

func writeClosureEntry(hash io.Writer, path string, isDir bool, size int64, perm os.FileMode, digest [sha256.Size]byte) {
	kind := byte(1)
	if isDir {
		kind = 0
	}
	_, _ = hash.Write([]byte{kind})
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(path)))
	_, _ = hash.Write(length[:])
	_, _ = hash.Write([]byte(path))
	var mode [4]byte
	binary.BigEndian.PutUint32(mode[:], uint32(perm.Perm()))
	_, _ = hash.Write(mode[:])
	binary.BigEndian.PutUint64(length[:], uint64(size))
	_, _ = hash.Write(length[:])
	_, _ = hash.Write(digest[:])
}
