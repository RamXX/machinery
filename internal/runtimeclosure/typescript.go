package runtimeclosure

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// The exact approved TypeScript closure of the first-release catalog
// (docs/test-assurance-contract.md section 6): node-test-typescript/v1
// executes only under Node 26.8.1 with the TypeScript compiler 7.0.2 on the
// pinned native platforms darwin/arm64 and linux/amd64. OpenTypeScript
// binds the complete closure by exact bytes without launching anything: the
// node executable, the TypeScript package tree (CLI shim, API surface,
// vendored libraries and the platform declaration libraries) and the
// platform native compiler binary. Validate performs the runtime's own
// identity probes (node --version, the native compiler's --version) under a
// live processscope scope; Close is the final process-free
// byte/topology/root-identity revalidation. There is no user-selectable
// adapter runtime API.
const (
	// TypeScriptProfile is the runtime profile identity of the
	// node-test-typescript/v1 closure.
	TypeScriptProfile = tddRuntimeProfile
	// RequiredNodeVersion is the exact first-release Node runtime version.
	RequiredNodeVersion = "v26.8.1"
	// RequiredTypeScriptVersion is the exact first-release TypeScript
	// compiler version.
	RequiredTypeScriptVersion = "7.0.2"
	// TypeScriptIdentityVersion is the RuntimeRef version spelling of the
	// two-part pinned closure identity.
	TypeScriptIdentityVersion = "26.8.1/7.0.2"

	typeScriptClosureDomain     = "machinery.tdd.runtime.node-test-typescript/v1"
	typeScriptProbeDeadlineMS   = int64(30000)
	typeScriptProbeOutputLimit  = int64(1 << 20)
	typeScriptBinaryMaxBytes    = int64(256 << 20)
	typeScriptTreeMaxEntries    = 20000
	typeScriptTreeMaxDepth      = 24
	typeScriptTreeMaxBytes      = int64(768 << 20)
	nodeIdentityProbeArg        = "--version"
	compilerIdentityExpected    = "Version " + RequiredTypeScriptVersion
	nodeIdentityExpected        = RequiredNodeVersion
	typeScriptTypescriptPkgName = "typescript"
)

// tddRuntimeProfile is unexported so no other package can conjure the
// identity without this file's pinned constants.
const tddRuntimeProfile = "node-test-typescript/v1"

// TypeScriptRequest is the closed open request of the pinned TypeScript
// runtime closure.
type TypeScriptRequest struct {
	// NodePath optionally names the node executable. Empty resolves the
	// host runtime from PATH through its real symlink chain; the compiler
	// closure is always resolved from the host tsc installation.
	NodePath string
	// ExpectedClosure, when nonempty, is an exact lowercase sha256 the
	// opened closure must reproduce; a mismatch fails closed.
	ExpectedClosure string
}

// TypeScript is the pinned Node 26.8.1 / TypeScript 7.0.2 runtime handle
// implementing tdd.RuntimeHandle. nodeLibrary binds the node runtime's own
// shared library (libnode) when the installation links one; it is part of
// the closure exactly like the executables.
type TypeScript struct {
	nodeSource     string
	nodePath       string
	nodeInfo       os.FileInfo
	nodeFile       *os.File
	nodeBody       []byte
	nodeDigest     string
	nodeLibPath    string
	nodeLibInfo    os.FileInfo
	nodeLibFile    *os.File
	nodeLibBody    []byte
	nodeLibDigest  string
	compilerPath   string
	compilerInfo   os.FileInfo
	compilerFile   *os.File
	compilerBody   []byte
	compilerDigest string
	pkgRoot        string
	pkgRootInfo    os.FileInfo
	pkgRootHandle  *os.Root
	treeDigest     string
	closureDigest  string
	identity       tdd.RuntimeRef
	closed         bool
}

// Compile-time contract check: the closed RuntimeHandle boundary.
var _ tdd.RuntimeHandle = (*TypeScript)(nil)

// typeScriptPlatformPackage maps the pinned native platforms to the
// compiler's platform package spelling.
func typeScriptPlatformPackage() (string, error) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/arm64":
		return "typescript-darwin-arm64", nil
	case "linux/amd64":
		return "typescript-linux-x64", nil
	default:
		return "", fmt.Errorf("UNSUPPORTED_PLATFORM: %s/%s is not a pinned native assurance platform for the TypeScript runtime closure", runtime.GOOS, runtime.GOARCH)
	}
}

// OpenTypeScript opens and binds the pinned TypeScript runtime closure. It
// never launches a process: the TypeScript version is pre-bound from the
// package's own static identity, both binaries are retained by exact bytes
// and the complete package tree is fingerprinted into the closure digest.
// The node version identity is proven by the scoped probes in Validate.
func OpenTypeScript(ctx context.Context, req TypeScriptRequest) (*TypeScript, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: cannot open the pinned TypeScript %s runtime closure: %w", RequiredTypeScriptVersion, err)
	}
	if _, err := typeScriptPlatformPackage(); err != nil {
		return nil, err
	}
	nodeSource := req.NodePath
	if nodeSource == "" {
		discovered, err := exec.LookPath("node")
		if err != nil {
			return nil, fmt.Errorf("UNSUPPORTED_VERSION: the pinned Node %s runtime is absent from PATH: %w", RequiredNodeVersion, err)
		}
		nodeSource = discovered
	}
	nodeAbs, err := filepath.Abs(nodeSource)
	if err != nil {
		return nil, err
	}
	nodeReal, err := filepath.EvalSymlinks(nodeAbs)
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: resolve node executable: %w", err)
	}
	nodeInfo, nodeFile, nodeBody, err := openTypeScriptBinary(nodeReal)
	if err != nil {
		return nil, err
	}
	handle := &TypeScript{
		nodeSource: nodeAbs, nodePath: nodeReal, nodeInfo: nodeInfo, nodeFile: nodeFile, nodeBody: nodeBody,
	}
	if libPath, err := discoverNodeSharedLibrary(nodeReal); err != nil {
		return nil, errors.Join(err, nodeFile.Close())
	} else if libPath != "" {
		libInfo, libFile, libBody, err := openTypeScriptLibrary(libPath)
		if err != nil {
			return nil, errors.Join(err, nodeFile.Close())
		}
		handle.nodeLibPath, handle.nodeLibInfo, handle.nodeLibFile, handle.nodeLibBody = libPath, libInfo, libFile, libBody
		handle.nodeLibDigest = typeScriptDigestBytes(libBody)
	}
	shimDiscovered, err := exec.LookPath("tsc")
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: the pinned TypeScript %s compiler is absent from PATH: %w", RequiredTypeScriptVersion, err)
	}
	shimReal, err := filepath.EvalSymlinks(shimDiscovered)
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: resolve tsc entry: %w", err)
	}
	if filepath.Base(filepath.Dir(shimReal)) != "bin" {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: tsc entry %s does not live in its package bin directory; unsupported installation layout", shimDiscovered)
	}
	pkgRoot := filepath.Dir(filepath.Dir(shimReal))
	pkgIdentity, err := readTypeScriptPackageIdentity(pkgRoot)
	if err != nil {
		return nil, err
	}
	if pkgIdentity.Name != typeScriptTypescriptPkgName || pkgIdentity.Version != RequiredTypeScriptVersion {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: TypeScript package %q version %q is not the exact pinned compiler %s", pkgIdentity.Name, pkgIdentity.Version, RequiredTypeScriptVersion)
	}
	platformPkg, _ := typeScriptPlatformPackage()
	compilerPath := filepath.Join(pkgRoot, "node_modules", "@typescript", platformPkg, "lib", "tsc")
	platformIdentity, err := readTypeScriptPackageIdentity(filepath.Dir(filepath.Dir(compilerPath)))
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: the pinned platform compiler package is absent or malformed: %w", err)
	}
	if platformIdentity.Name != "@typescript/"+platformPkg || platformIdentity.Version != RequiredTypeScriptVersion {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: platform compiler package %q version %q is not the exact pinned %s", platformIdentity.Name, platformIdentity.Version, RequiredTypeScriptVersion)
	}
	compilerInfo, compilerFile, compilerBody, err := openTypeScriptBinary(compilerPath)
	if err != nil {
		return nil, err
	}
	pkgRootInfo, err := os.Lstat(pkgRoot)
	if err != nil || !pkgRootInfo.IsDir() {
		return nil, errors.Join(err, fmt.Errorf("UNSUPPORTED_VERSION: TypeScript package root must be a real directory"))
	}
	pkgRootHandle, err := os.OpenRoot(pkgRoot)
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: open TypeScript package root: %w", err)
	}
	openedRoot, err := pkgRootHandle.Lstat(".")
	if err != nil || !os.SameFile(pkgRootInfo, openedRoot) {
		return nil, errors.Join(err, pkgRootHandle.Close(), fmt.Errorf("UNSUPPORTED_VERSION: TypeScript package root changed identity while opening"))
	}
	treeDigest, err := fingerprintTypeScriptPackage(pkgRootHandle)
	if err != nil {
		return nil, errors.Join(pkgRootHandle.Close(), compilerFile.Close(), nodeFile.Close(), fmt.Errorf("UNSUPPORTED_VERSION: fingerprint TypeScript package closure: %w", err))
	}
	handle.compilerPath, handle.compilerInfo, handle.compilerFile, handle.compilerBody = compilerPath, compilerInfo, compilerFile, compilerBody
	handle.pkgRoot, handle.pkgRootInfo, handle.pkgRootHandle, handle.treeDigest = pkgRoot, pkgRootInfo, pkgRootHandle, treeDigest
	handle.nodeDigest = typeScriptDigestBytes(nodeBody)
	handle.compilerDigest = typeScriptDigestBytes(compilerBody)
	handle.closureDigest = typeScriptClosureDigest(handle.nodeDigest, handle.nodeLibDigest, handle.compilerDigest, treeDigest)
	handle.identity = tdd.RuntimeRef{
		Profile: TypeScriptProfile, Version: TypeScriptIdentityVersion,
		Platform: runtime.GOOS + "/" + runtime.GOARCH, Closure: handle.closureDigest,
	}
	if req.ExpectedClosure != "" && req.ExpectedClosure != handle.closureDigest {
		_ = handle.Close()
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: explicit TypeScript runtime closure sha256:%s does not match paired trust root sha256:%s", handle.closureDigest, req.ExpectedClosure)
	}
	return handle, nil
}

// Identity implements tdd.RuntimeHandle.
func (ts *TypeScript) Identity() tdd.RuntimeRef { return ts.identity }

// NodePath returns the resolved real node executable path.
func (ts *TypeScript) NodePath() string { return ts.nodePath }

// CompilerPath returns the resolved real native TypeScript compiler binary.
func (ts *TypeScript) CompilerPath() string { return ts.compilerPath }

// NodeDigest returns the sha256 of the verified node executable bytes.
func (ts *TypeScript) NodeDigest() string { return ts.nodeDigest }

// CompilerDigest returns the sha256 of the verified native compiler bytes.
func (ts *TypeScript) CompilerDigest() string { return ts.compilerDigest }

// Validate implements tdd.RuntimeHandle: it performs the runtime's own
// identity probes (node --version, the native compiler's --version) as real
// scoped subprocesses while the supplied scope is live, then re-checks the
// retained byte identities.
func (ts *TypeScript) Validate(ctx context.Context, scope processscope.Scope) error {
	if ts.closed {
		return fmt.Errorf("UNSUPPORTED_VERSION: the TypeScript runtime closure handle is closed")
	}
	if scope == nil {
		return fmt.Errorf("INVALID_SCHEMA: TypeScript closure validation requires a live processscope scope")
	}
	probeEnv := []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + os.TempDir(),
		"TMPDIR=" + os.TempDir(),
		"LC_ALL=C.UTF-8",
	}
	probes := []struct {
		name string
		cmd  processscope.Command
		want string
	}{
		{"node", processscope.Command{
			Executable: ts.nodePath, Args: []string{nodeIdentityProbeArg}, Env: probeEnv,
			RuntimeDigest: "sha256:" + ts.nodeDigest, DeadlineMS: typeScriptProbeDeadlineMS,
		}, nodeIdentityExpected},
		{"tsc", processscope.Command{
			Executable: ts.compilerPath, Args: []string{nodeIdentityProbeArg}, Env: probeEnv,
			RuntimeDigest: "sha256:" + ts.compilerDigest, DeadlineMS: typeScriptProbeDeadlineMS,
		}, compilerIdentityExpected},
	}
	for _, probe := range probes {
		if err := typeScriptProbe(ctx, scope, probe.cmd, probe.want, probe.name); err != nil {
			return err
		}
	}
	return ts.revalidate()
}

// typeScriptProbe runs one identity probe in its own child scope of the
// supplied live scope and closes the child with verified terminal cleanup,
// so probes never consume the caller's job budget with unretired guardians.
func typeScriptProbe(ctx context.Context, scope processscope.Scope, cmd processscope.Command, want, name string) error {
	child, err := scope.Child(ctx)
	if err != nil {
		return fmt.Errorf("CUSTODY_ERROR: open the %s identity probe custody child: %w", name, err)
	}
	var stdout, stderr bytes.Buffer
	result, runErr := func() (processscope.Result, error) {
		attached, err := child.Attach(cmd)
		if err != nil {
			return processscope.Result{}, fmt.Errorf("CUSTODY_ERROR: attach the %s identity probe: %w", name, err)
		}
		return child.Run(ctx, attached, processscope.Streams{
			Stdout: &stdout, Stderr: &stderr, StdoutLimit: typeScriptProbeOutputLimit, StderrLimit: typeScriptProbeOutputLimit,
		})
	}()
	closeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	report, closeErr := child.Close(closeCtx)
	if closeErr != nil || report.Status != processscope.StatusCleaned {
		return errors.Join(runErr, closeErr, fmt.Errorf("CUSTODY_ERROR: the %s probe custody child cleanup did not verify: %+v", name, report))
	}
	if runErr != nil {
		return fmt.Errorf("CUSTODY_ERROR: the %s identity probe failed under custody: %w (stderr: %s)", name, runErr, stderr.String())
	}
	if !result.Completed || result.ExitCode != 0 || result.Signal != "" {
		return fmt.Errorf("UNSUPPORTED_VERSION: the %s identity probe exited abnormally (completed=%v exit=%d signal=%q)", name, result.Completed, result.ExitCode, result.Signal)
	}
	if got := strings.TrimSpace(stdout.String()); got != want {
		return fmt.Errorf("UNSUPPORTED_VERSION: %s runtime identity %q is not the exact pinned pin %q", name, got, want)
	}
	return nil
}

// Close implements tdd.RuntimeHandle: the final process-free
// byte/topology/root-identity revalidation before releasing the retained
// handles. It never launches a process.
func (ts *TypeScript) Close() error {
	if ts.closed {
		return fmt.Errorf("UNSUPPORTED_VERSION: the TypeScript runtime closure handle is already closed")
	}
	ts.closed = true
	err := ts.revalidate()
	if ts.nodeLibFile != nil {
		err = errors.Join(err, ts.nodeLibFile.Close())
	}
	return errors.Join(err, ts.nodeFile.Close(), ts.compilerFile.Close(), ts.pkgRootHandle.Close())
}

// revalidate re-digests the retained runtime bytes and re-fingerprints the
// package tree without launching anything.
func (ts *TypeScript) revalidate() error {
	nodeInfo, err := os.Lstat(ts.nodePath)
	if err != nil || !sameJavaFileSnapshot(ts.nodeInfo, nodeInfo) {
		return fmt.Errorf("UNSUPPORTED_VERSION: the retained node executable changed identity after verification")
	}
	if ts.nodeLibPath != "" {
		libInfo, err := os.Lstat(ts.nodeLibPath)
		if err != nil || !sameJavaFileSnapshot(ts.nodeLibInfo, libInfo) {
			return fmt.Errorf("UNSUPPORTED_VERSION: the retained node shared library changed identity after verification")
		}
		if typeScriptDigestBytes(ts.nodeLibBody) != ts.nodeLibDigest {
			return fmt.Errorf("UNSUPPORTED_VERSION: the retained node shared library changed content after verification")
		}
	}
	compilerInfo, err := os.Lstat(ts.compilerPath)
	if err != nil || !sameJavaFileSnapshot(ts.compilerInfo, compilerInfo) {
		return fmt.Errorf("UNSUPPORTED_VERSION: the retained native compiler changed identity after verification")
	}
	if typeScriptDigestBytes(ts.nodeBody) != ts.nodeDigest || typeScriptDigestBytes(ts.compilerBody) != ts.compilerDigest {
		return fmt.Errorf("UNSUPPORTED_VERSION: the retained runtime bytes changed content after verification")
	}
	pkgRootInfo, err := os.Lstat(ts.pkgRoot)
	if err != nil || !os.SameFile(ts.pkgRootInfo, pkgRootInfo) {
		return fmt.Errorf("UNSUPPORTED_VERSION: the TypeScript package root changed identity after verification")
	}
	treeDigest, err := fingerprintTypeScriptPackage(ts.pkgRootHandle)
	if err != nil {
		return fmt.Errorf("UNSUPPORTED_VERSION: revalidate the TypeScript package closure: %w", err)
	}
	if treeDigest != ts.treeDigest {
		return fmt.Errorf("UNSUPPORTED_VERSION: the TypeScript package closure changed after verification (was sha256:%s, now sha256:%s)", ts.treeDigest, treeDigest)
	}
	return nil
}

type typeScriptPkgIdentity struct{ Name, Version string }

func readTypeScriptPackageIdentity(pkgRoot string) (typeScriptPkgIdentity, error) {
	var identity typeScriptPkgIdentity
	raw, err := os.ReadFile(filepath.Join(pkgRoot, "package.json"))
	if err != nil || len(raw) > 1<<20 {
		return identity, fmt.Errorf("package identity file must be a bounded readable package.json: %w", err)
	}
	var doc struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil || doc.Name == "" || doc.Version == "" {
		return identity, fmt.Errorf("malformed package identity")
	}
	return typeScriptPkgIdentity{Name: doc.Name, Version: doc.Version}, nil
}

// discoverNodeSharedLibrary resolves the node runtime's own shared library
// (libnode) when the installation links one; an empty result means a static
// node whose executable is the complete runtime. Ambiguous or multiple
// libnode candidates fail closed instead of binding an unverified library.
func discoverNodeSharedLibrary(nodePath string) (string, error) {
	libDir := filepath.Join(filepath.Dir(filepath.Dir(nodePath)), "lib")
	entries, err := os.ReadDir(libDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var candidates []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			continue
		}
		if strings.HasPrefix(name, "libnode.") && (strings.HasSuffix(name, ".dylib") || strings.HasSuffix(name, ".so") || strings.Contains(name, ".so.")) {
			candidates = append(candidates, filepath.Join(libDir, name))
		}
	}
	if len(candidates) == 0 {
		return "", nil
	}
	if len(candidates) > 1 {
		return "", fmt.Errorf("UNSUPPORTED_VERSION: ambiguous node shared library closure %v", candidates)
	}
	return candidates[0], nil
}

func openTypeScriptLibrary(path string) (os.FileInfo, *os.File, []byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, nil, nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, nil, nil, fmt.Errorf("node shared library %s must be a regular non-symlink file", path)
	}
	if before.Size() <= 0 || before.Size() > typeScriptBinaryMaxBytes {
		return nil, nil, nil, fmt.Errorf("node shared library %s exceeds the %d-byte closure limit", path, typeScriptBinaryMaxBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}
	opened, statErr := file.Stat()
	if statErr != nil || !sameJavaFileSnapshot(before, opened) {
		return nil, nil, nil, errors.Join(statErr, file.Close(), fmt.Errorf("node shared library %s changed identity while opening", path))
	}
	body, readErr := io.ReadAll(io.LimitReader(file, before.Size()+1))
	afterInfo, statErr2 := file.Stat()
	if err := errors.Join(readErr, statErr2); err != nil {
		return nil, nil, nil, errors.Join(err, file.Close())
	}
	if int64(len(body)) != before.Size() || !sameJavaFileSnapshot(before, afterInfo) {
		return nil, nil, nil, errors.Join(fmt.Errorf("node shared library %s changed size while reading", path), file.Close())
	}
	return before, file, body, nil
}

func openTypeScriptBinary(path string) (os.FileInfo, *os.File, []byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, nil, nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Mode().Perm()&0111 == 0 {
		return nil, nil, nil, fmt.Errorf("runtime binary %s must be a regular non-symlink executable", path)
	}
	if before.Size() <= 0 || before.Size() > typeScriptBinaryMaxBytes {
		return nil, nil, nil, fmt.Errorf("runtime binary %s exceeds the %d-byte closure limit", path, typeScriptBinaryMaxBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}
	opened, statErr := file.Stat()
	if statErr != nil || !sameJavaFileSnapshot(before, opened) {
		return nil, nil, nil, errors.Join(statErr, file.Close(), fmt.Errorf("runtime binary %s changed identity while opening", path))
	}
	body, readErr := io.ReadAll(io.LimitReader(file, before.Size()+1))
	afterInfo, statErr2 := file.Stat()
	if err := errors.Join(readErr, statErr2); err != nil {
		return nil, nil, nil, errors.Join(err, file.Close())
	}
	if int64(len(body)) != before.Size() || !sameJavaFileSnapshot(before, afterInfo) {
		return nil, nil, nil, errors.Join(fmt.Errorf("runtime binary %s changed size while reading", path), file.Close())
	}
	return before, file, body, nil
}

// fingerprintTypeScriptPackage hashes the complete TypeScript package tree
// topologically: sorted directory and file identities, exact file bytes,
// bounded entries/bytes/depth, with a stable-identity double census. It
// reuses the Java closure's bounded-tree accounting.
func fingerprintTypeScriptPackage(root *os.Root) (string, error) {
	names, infos, _, err := inventoryTypeScriptRoot(root)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	for _, name := range names {
		info := infos[name]
		slashName := filepath.ToSlash(name)
		if info.IsDir() {
			_, _ = fmt.Fprintf(hash, "d\x00%s\x00", slashName)
			continue
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("TypeScript package closure entry %s must be a regular file or directory", slashName)
		}
		_, _ = fmt.Fprintf(hash, "f\x00%s\x00%d\x00", slashName, info.Size())
		if err := hashTypeScriptFile(root, name, info, hash); err != nil {
			return "", fmt.Errorf("hash TypeScript package closure entry %s: %w", slashName, err)
		}
	}
	finalNames, _, _, err := inventoryTypeScriptRoot(root)
	if err != nil {
		return "", err
	}
	if len(finalNames) != len(names) {
		return "", fmt.Errorf("TypeScript package closure inventory changed while hashing")
	}
	for i, name := range names {
		if finalNames[i] != name {
			return "", fmt.Errorf("TypeScript package closure inventory changed while hashing at %s", filepath.ToSlash(name))
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func inventoryTypeScriptRoot(root *os.Root) ([]string, map[string]os.FileInfo, int64, error) {
	limits := javaTreeLimits{maxDepth: typeScriptTreeMaxDepth, maxEntries: typeScriptTreeMaxEntries, maxBytes: typeScriptTreeMaxBytes}
	if err := validateJavaTreeLimits("TypeScript package closure", limits); err != nil {
		return nil, nil, 0, err
	}
	budget := javaTreeBudget{label: "TypeScript package closure", limits: limits}
	names := make([]string, 0, 1024)
	infos := map[string]os.FileInfo{}
	var walk func(dir string) error
	walk = func(dir string) error {
		info, err := root.Lstat(dir)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("TypeScript package entry %s must be a real directory", dir)
		}
		if err := budget.addEntry(dir, 0); err != nil {
			return err
		}
		names = append(names, dir)
		infos[dir] = info
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
			if child.IsDir() {
				if err := walk(name); err != nil {
					return err
				}
				continue
			}
			if child.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("TypeScript package closure entry %s must not be a symlink", filepath.ToSlash(name))
			}
			size := int64(0)
			if child.Mode().IsRegular() {
				size = child.Size()
			}
			if err := budget.addEntry(name, size); err != nil {
				return err
			}
			names = append(names, name)
			infos[name] = child
		}
		return nil
	}
	if err := walk("."); err != nil {
		return nil, nil, 0, err
	}
	sort.Strings(names)
	return names, infos, budget.bytes, nil
}

func hashTypeScriptFile(root *os.Root, name string, before os.FileInfo, hash io.Writer) error {
	if before.Size() < 0 {
		return fmt.Errorf("negative size")
	}
	file, err := root.Open(name)
	if err != nil {
		return err
	}
	opened, statErr := file.Stat()
	if statErr != nil || !sameJavaFileSnapshot(before, opened) {
		return errors.Join(statErr, file.Close(), fmt.Errorf("changed identity while opening"))
	}
	written, copyErr := io.Copy(hash, io.LimitReader(file, before.Size()+1))
	closeErr := file.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return err
	}
	if written != before.Size() {
		return fmt.Errorf("changed size while hashing")
	}
	return nil
}

func typeScriptDigestBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func typeScriptClosureDigest(nodeDigest, nodeLibDigest, compilerDigest, treeDigest string) string {
	sum := sha256.Sum256([]byte(typeScriptClosureDomain + "\x00" + nodeDigest + "\x00" + nodeLibDigest + "\x00" + compilerDigest + "\x00" + treeDigest))
	return hex.EncodeToString(sum[:])
}
