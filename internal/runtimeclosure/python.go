package runtimeclosure

// The exact approved CPython runtime closure of the first-release catalog
// (docs/test-assurance-contract.md section 6): python-unittest/v1 executes
// only under CPython 3.14.7 on the pinned native platforms darwin/arm64 and
// linux/amd64. OpenPython binds the complete closure by exact bytes without
// launching anything: the interpreter executable, the interpreter's own
// stdlib library tree (including any installed site-packages under that
// tree — a runtime-library change changes the closure identity) and, when
// the installation ships it, the static patchlevel.h version identity of
// the interpreter's own prefix. Validate performs the runtime's own
// identity probe (python3 --version) under a live processscope scope; Close
// is the final process-free byte/topology/root-identity revalidation. There
// is no user-selectable adapter runtime API.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/tdd"
)

const (
	// PythonProfile is the runtime profile identity of the
	// python-unittest/v1 closure.
	PythonProfile = pythonRuntimeProfile
	// RequiredPythonVersion is the exact first-release CPython version.
	RequiredPythonVersion = "3.14.7"

	pythonClosureDomain    = "machinery.runtime.python/v1"
	pythonProbeDeadlineMS  = int64(30000)
	pythonProbeOutputLimit = int64(1 << 20)
	pythonBinaryMaxBytes   = int64(256 << 20)
	pythonTreeMaxEntries   = 200000
	pythonTreeMaxDepth     = int64(24)
	pythonTreeMaxBytes     = int64(2 << 30)
	pythonIdentityExpected = "Python " + RequiredPythonVersion
)

// pythonRuntimeProfile is unexported so no other package can conjure the
// identity without this file's pinned constants.
const pythonRuntimeProfile = "python-unittest/v1"

var (
	pythonLauncherPattern = regexp.MustCompile(`^python3(\.\d+)?$`)
	pythonStdlibPattern   = regexp.MustCompile(`^python3\.\d+$`)
)

// PythonRequest is the closed open request of the pinned CPython runtime
// closure.
type PythonRequest struct {
	// PythonPath optionally names the python3 interpreter. Empty resolves
	// the host runtime from PATH through its real symlink chain.
	PythonPath string
	// ExpectedClosure, when nonempty, is an exact lowercase sha256 the
	// opened closure must reproduce; a mismatch fails closed.
	ExpectedClosure string
}

// Python is the pinned CPython 3.14.7 runtime handle implementing
// tdd.RuntimeHandle.
type Python struct {
	source     string
	binPath    string
	binInfo    os.FileInfo
	binFile    *os.File
	binBody    []byte
	binDigest  string
	stdlibPath string
	stdlibInfo os.FileInfo
	stdlibRoot *os.Root
	treeDigest string
	// patchlevelPinned records whether the interpreter's own prefix ships
	// the static patchlevel.h identity; when it does, its exact version
	// was verified at open. When it does not, the exact version proof is
	// the scoped identity probe of Validate (disclosed residual, never a
	// guess: the bytes are always bound).
	patchlevelPinned bool
	closureDigest    string
	identity         tdd.RuntimeRef
	closed           bool
}

// Compile-time contract check: the closed RuntimeHandle boundary.
var _ tdd.RuntimeHandle = (*Python)(nil)

// OpenPython opens and binds the pinned CPython runtime closure. It never
// launches a process: the interpreter bytes are retained under an open
// handle, the complete stdlib library tree is fingerprinted into the
// closure digest, and the exact version is pinned statically from the
// interpreter's own patchlevel.h whenever the installation ships it.
func OpenPython(ctx context.Context, req PythonRequest) (*Python, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: cannot open the pinned CPython %s runtime closure: %w", RequiredPythonVersion, err)
	}
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if platform != "darwin/arm64" && platform != "linux/amd64" {
		return nil, fmt.Errorf("UNSUPPORTED_PLATFORM: %s is not a pinned native assurance platform for the CPython runtime closure", platform)
	}
	source := req.PythonPath
	if source == "" {
		discovered, err := exec.LookPath("python3")
		if err != nil {
			return nil, fmt.Errorf("UNSUPPORTED_VERSION: the pinned CPython %s runtime is absent from PATH: %w", RequiredPythonVersion, err)
		}
		source = discovered
	}
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return nil, err
	}
	binPath, err := filepath.EvalSymlinks(sourceAbs)
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: resolve the python3 interpreter: %w", err)
	}
	if !pythonLauncherPattern.MatchString(filepath.Base(binPath)) {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: interpreter %s is not a python3 launcher of a supported installation layout", binPath)
	}
	binInfo, binFile, binBody, err := openTypeScriptBinary(binPath)
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: bind the CPython interpreter: %w", err)
	}
	stdlibPath, err := discoverPythonStdlib(binPath)
	if err != nil {
		return nil, errors.Join(err, binFile.Close())
	}
	stdlibInfo, err := os.Lstat(stdlibPath)
	if err != nil || !stdlibInfo.IsDir() {
		return nil, errors.Join(err, binFile.Close(), fmt.Errorf("UNSUPPORTED_VERSION: the CPython stdlib root %s must be a real directory", stdlibPath))
	}
	stdlibRoot, err := os.OpenRoot(stdlibPath)
	if err != nil {
		return nil, errors.Join(err, binFile.Close(), fmt.Errorf("UNSUPPORTED_VERSION: open the CPython stdlib root: %w", err))
	}
	fail := func(err error) (*Python, error) {
		return nil, errors.Join(err, stdlibRoot.Close(), binFile.Close())
	}
	openedStdlib, err := stdlibRoot.Lstat(".")
	if err != nil || !os.SameFile(stdlibInfo, openedStdlib) {
		return fail(fmt.Errorf("UNSUPPORTED_VERSION: the CPython stdlib root changed identity while opening"))
	}
	// The stdlib tree must carry the native unittest framework the strict
	// suite executes; an interpreter without its stdlib is not the closed
	// runtime closure.
	unitInfo, err := stdlibRoot.Lstat("unittest")
	if err != nil || !unitInfo.IsDir() {
		return fail(fmt.Errorf("UNSUPPORTED_VERSION: the CPython stdlib root %s lacks its stdlib unittest framework", stdlibPath))
	}
	patchlevelPinned := false
	if version, err := readPythonPatchlevel(stdlibPath); err == nil {
		if version != RequiredPythonVersion {
			return fail(fmt.Errorf("UNSUPPORTED_VERSION: the interpreter's static patchlevel identity is %q, want exactly %s", version, RequiredPythonVersion))
		}
		patchlevelPinned = true
	}
	treeDigest, err := fingerprintPythonStdlib(stdlibRoot, stdlibPath)
	if err != nil {
		return fail(fmt.Errorf("UNSUPPORTED_VERSION: fingerprint the CPython stdlib closure: %w", err))
	}
	binDigest := typeScriptDigestBytes(binBody)
	handle := &Python{
		source: sourceAbs, binPath: binPath, binInfo: binInfo, binFile: binFile, binBody: binBody,
		binDigest: binDigest, stdlibPath: stdlibPath, stdlibInfo: stdlibInfo, stdlibRoot: stdlibRoot,
		treeDigest: treeDigest, patchlevelPinned: patchlevelPinned,
	}
	handle.closureDigest = pythonClosureDigest(binDigest, treeDigest)
	handle.identity = tdd.RuntimeRef{
		Profile: PythonProfile, Version: RequiredPythonVersion,
		Platform: platform, Closure: handle.closureDigest,
	}
	if req.ExpectedClosure != "" {
		if len(req.ExpectedClosure) != sha256.Size*2 || req.ExpectedClosure != strings.ToLower(req.ExpectedClosure) {
			return fail(fmt.Errorf("INVALID_SCHEMA: PythonRequest.ExpectedClosure requires an exact lowercase sha256"))
		}
		if req.ExpectedClosure != handle.closureDigest {
			return fail(fmt.Errorf("UNSUPPORTED_VERSION: explicit CPython runtime closure sha256:%s does not match the paired trust root sha256:%s", handle.closureDigest, req.ExpectedClosure))
		}
	}
	return handle, nil
}

// Identity implements tdd.RuntimeHandle.
func (py *Python) Identity() tdd.RuntimeRef { return py.identity }

// Binary returns the resolved absolute python3 interpreter of the closure.
func (py *Python) Binary() string { return py.binPath }

// StdlibRoot returns the resolved absolute stdlib library root.
func (py *Python) StdlibRoot() string { return py.stdlibPath }

// Validate implements tdd.RuntimeHandle: it performs the runtime's own
// identity probe (python3 --version) as a real scoped subprocess while the
// supplied scope is live, then re-checks the retained byte identities.
func (py *Python) Validate(ctx context.Context, scope processscope.Scope) error {
	if py.closed {
		return fmt.Errorf("UNSUPPORTED_VERSION: the CPython runtime closure handle is closed")
	}
	if scope == nil {
		return fmt.Errorf("INVALID_SCHEMA: CPython closure validation requires a live processscope scope")
	}
	probeEnv := []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + os.TempDir(),
		"TMPDIR=" + os.TempDir(),
		"LC_ALL=C.UTF-8",
	}
	cmd := processscope.Command{
		Executable: py.binPath, Args: []string{"--version"}, Env: probeEnv,
		RuntimeDigest: "sha256:" + py.binDigest, DeadlineMS: pythonProbeDeadlineMS,
	}
	if err := pythonProbe(ctx, scope, cmd, pythonIdentityExpected); err != nil {
		return err
	}
	return py.revalidate()
}

// Close implements tdd.RuntimeHandle: the final process-free
// byte/topology/root-identity revalidation before releasing the retained
// handles. It never launches a process.
func (py *Python) Close() error {
	if py.closed {
		return fmt.Errorf("UNSUPPORTED_VERSION: the CPython runtime closure handle is already closed")
	}
	py.closed = true
	err := py.revalidate()
	return errors.Join(err, py.binFile.Close(), py.stdlibRoot.Close())
}

// revalidate re-digests the retained interpreter bytes and re-fingerprints
// the stdlib tree without launching anything.
func (py *Python) revalidate() error {
	binInfo, err := os.Lstat(py.binPath)
	if err != nil || !sameJavaFileSnapshot(py.binInfo, binInfo) {
		return fmt.Errorf("UNSUPPORTED_VERSION: the retained CPython interpreter changed identity after verification")
	}
	if typeScriptDigestBytes(py.binBody) != py.binDigest {
		return fmt.Errorf("UNSUPPORTED_VERSION: the retained CPython interpreter changed content after verification")
	}
	stdlibInfo, err := os.Lstat(py.stdlibPath)
	if err != nil || !os.SameFile(py.stdlibInfo, stdlibInfo) {
		return fmt.Errorf("UNSUPPORTED_VERSION: the CPython stdlib root changed identity after verification")
	}
	treeDigest, err := fingerprintPythonStdlib(py.stdlibRoot, py.stdlibPath)
	if err != nil {
		return fmt.Errorf("UNSUPPORTED_VERSION: revalidate the CPython stdlib closure: %w", err)
	}
	if treeDigest != py.treeDigest {
		return fmt.Errorf("UNSUPPORTED_VERSION: the CPython stdlib closure changed after verification (was sha256:%s, now sha256:%s)", py.treeDigest, treeDigest)
	}
	return nil
}

// pythonProbe runs the identity probe in its own child scope of the
// supplied live scope and closes the child with verified terminal cleanup.
func pythonProbe(ctx context.Context, scope processscope.Scope, cmd processscope.Command, want string) error {
	child, err := scope.Child(ctx)
	if err != nil {
		return fmt.Errorf("CUSTODY_ERROR: open the python identity probe custody child: %w", err)
	}
	var stdout bytes.Buffer
	result, runErr := func() (processscope.Result, error) {
		attached, err := child.Attach(cmd)
		if err != nil {
			return processscope.Result{}, fmt.Errorf("CUSTODY_ERROR: attach the python identity probe: %w", err)
		}
		return child.Run(ctx, attached, processscope.Streams{
			Stdout: &stdout, Stderr: io.Discard, StderrLimit: pythonProbeOutputLimit, StdoutLimit: pythonProbeOutputLimit,
		})
	}()
	closeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	report, closeErr := child.Close(closeCtx)
	if closeErr != nil || report.Status != processscope.StatusCleaned {
		return errors.Join(runErr, closeErr, fmt.Errorf("CUSTODY_ERROR: the python probe custody child cleanup did not verify: %+v", report))
	}
	if runErr != nil {
		return fmt.Errorf("CUSTODY_ERROR: the python identity probe failed under custody: %w (output: %s)", runErr, boundedPythonString(stdout.String(), 2048))
	}
	if !result.Completed || result.ExitCode != 0 || result.Signal != "" {
		return fmt.Errorf("UNSUPPORTED_VERSION: the python identity probe exited abnormally (completed=%v exit=%d signal=%q)", result.Completed, result.ExitCode, result.Signal)
	}
	if got := strings.TrimSpace(stdout.String()); got != want {
		return fmt.Errorf("UNSUPPORTED_VERSION: python runtime identity %q is not the exact pinned pin %q", got, want)
	}
	return nil
}

// discoverPythonStdlib resolves the interpreter's own stdlib library tree
// from the real launcher path (the framework prefix layout of macOS
// distributions and the /usr layout of Linux distributions): the unique
// lib/python3.X sibling directory of the interpreter's prefix, matching the
// interpreter's own version suffix when its name carries one.
func discoverPythonStdlib(binPath string) (string, error) {
	prefix := filepath.Dir(filepath.Dir(binPath))
	libDir := filepath.Join(prefix, "lib")
	entries, err := os.ReadDir(libDir)
	if err != nil {
		return "", fmt.Errorf("UNSUPPORTED_VERSION: the CPython installation prefix %s carries no lib directory: %w", prefix, err)
	}
	versionSuffix := strings.TrimPrefix(filepath.Base(binPath), "python")
	var candidates []string
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || !pythonStdlibPattern.MatchString(name) {
			continue
		}
		if versionSuffix != "3" && versionSuffix != "" && name != "python"+versionSuffix {
			continue
		}
		candidates = append(candidates, filepath.Join(libDir, name))
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("UNSUPPORTED_VERSION: no stdlib library tree of the pinned layout exists under %s", libDir)
	}
	if len(candidates) > 1 {
		return "", fmt.Errorf("UNSUPPORTED_VERSION: ambiguous CPython stdlib closure %v", candidates)
	}
	return candidates[0], nil
}

// readPythonPatchlevel reads the interpreter's own static version identity
// from the prefix include directory when the installation ships it.
func readPythonPatchlevel(stdlibPath string) (string, error) {
	prefix := filepath.Dir(filepath.Dir(stdlibPath))
	includeDir := filepath.Join(prefix, "include")
	entries, err := os.ReadDir(includeDir)
	if err != nil {
		return "", err
	}
	var header string
	for _, entry := range entries {
		if entry.IsDir() && pythonStdlibPattern.MatchString(entry.Name()) {
			if header != "" {
				return "", fmt.Errorf("ambiguous include directories under %s", includeDir)
			}
			header = filepath.Join(includeDir, entry.Name(), "patchlevel.h")
		}
	}
	if header == "" {
		return "", fmt.Errorf("no python include directory under %s", includeDir)
	}
	body, err := os.ReadFile(header)
	if err != nil || len(body) > 1<<20 {
		return "", err
	}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 3 && fields[0] == "#define" && fields[1] == "PY_VERSION" {
			return strings.Trim(fields[2], `"`), nil
		}
	}
	return "", fmt.Errorf("patchlevel.h carries no PY_VERSION identity")
}

// fingerprintPythonStdlib hashes the complete stdlib tree topologically:
// sorted directory and file identities, exact file bytes, and symlink
// members bound by their target string plus the resolved content of
// file-target links (CPython framework layouts legitimately carry both);
// bounded entries/bytes/depth with a stable-identity double census. It
// reuses the Java closure's bounded-tree accounting.
func fingerprintPythonStdlib(root *os.Root, absPath string) (string, error) {
	limits := javaTreeLimits{maxDepth: int(pythonTreeMaxDepth), maxEntries: int(pythonTreeMaxEntries), maxBytes: pythonTreeMaxBytes}
	if err := validateJavaTreeLimits("CPython stdlib closure", limits); err != nil {
		return "", err
	}
	budget := javaTreeBudget{label: "CPython stdlib closure", limits: limits}
	hash := sha256.New()
	hash.Write([]byte(pythonClosureDomain))
	var walked int
	var walk func(dir string, depth int64) error
	walk = func(dir string, depth int64) error {
		if depth > int64(limits.maxDepth) {
			return fmt.Errorf("depth bound exceeded at %s", dir)
		}
		info, err := root.Lstat(dir)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("CPython stdlib closure entry %s must be a real directory", dir)
		}
		if err := budget.addEntry(dir, 0); err != nil {
			return err
		}
		walked++
		opened, err := root.Open(dir)
		if err != nil {
			return err
		}
		entries, readErr := readJavaDirEntriesBounded(opened, budget.remainingEntries(), limits.maxEntries)
		closeErr := opened.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		for _, entry := range entries {
			name := filepath.Join(dir, entry.Name())
			child, err := root.Lstat(name)
			if err != nil {
				return err
			}
			slashName := filepath.ToSlash(name)
			switch {
			case child.IsDir():
				_, _ = fmt.Fprintf(hash, "d\x00%s\x00", slashName)
				if err := walk(name, depth+1); err != nil {
					return err
				}
			case child.Mode()&os.ModeSymlink != 0:
				target, err := root.Readlink(name)
				if err != nil {
					return fmt.Errorf("CPython stdlib closure link %s: %w", slashName, err)
				}
				_, _ = fmt.Fprintf(hash, "l\x00%s\x00%s\x00", slashName, target)
				if resolved, statErr := os.Stat(filepath.Join(absPath, filepath.FromSlash(name))); statErr == nil && resolved.Mode().IsRegular() && resolved.Size() > 0 && resolved.Size() <= pythonBinaryMaxBytes {
					if body, readErr := os.ReadFile(filepath.Join(absPath, filepath.FromSlash(name))); readErr == nil {
						sum := sha256.Sum256(body)
						_, _ = hash.Write(sum[:])
					}
				}
			case child.Mode().IsRegular():
				if err := budget.addEntry(name, child.Size()); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(hash, "f\x00%s\x00%d\x00", slashName, child.Size())
				if err := hashTypeScriptFile(root, name, child, hash); err != nil {
					return fmt.Errorf("hash CPython stdlib closure entry %s: %w", slashName, err)
				}
			default:
				return fmt.Errorf("CPython stdlib closure entry %s is not a regular file, directory or bound symlink", slashName)
			}
		}
		return nil
	}
	if err := walk(".", 0); err != nil {
		return "", err
	}
	finalWalked := 0
	var recount func(dir string, depth int64) error
	recount = func(dir string, depth int64) error {
		if depth > int64(limits.maxDepth) {
			return fmt.Errorf("depth bound exceeded at %s", dir)
		}
		info, err := root.Lstat(dir)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("CPython stdlib closure entry %s must be a real directory", dir)
		}
		finalWalked++
		opened, err := root.Open(dir)
		if err != nil {
			return err
		}
		entries, readErr := readJavaDirEntriesBounded(opened, limits.maxEntries, limits.maxEntries)
		closeErr := opened.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				if err := recount(filepath.Join(dir, entry.Name()), depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := recount(".", 0); err != nil {
		return "", err
	}
	if finalWalked != walked {
		return "", fmt.Errorf("CPython stdlib closure inventory changed while hashing")
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func boundedPythonString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(bounded)"
}

// pythonClosureDigest binds the interpreter bytes and the stdlib tree
// fingerprint under the closure domain.
func pythonClosureDigest(binDigest, treeDigest string) string {
	sum := sha256.Sum256([]byte(pythonClosureDomain + "\x00" + binDigest + "\x00" + treeDigest))
	return hex.EncodeToString(sum[:])
}
