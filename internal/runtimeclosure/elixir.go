package runtimeclosure

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
	"sort"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/tdd"
)

// The exact approved Elixir closure of the first-release catalog
// (docs/test-assurance-contract.md section 6): elixir-exunit/v1 executes
// only under Elixir/ExUnit/Mix 1.20.4 with Erlang/OTP 29.0.6 (ERTS 17.0.6)
// on the pinned native platforms darwin/arm64 and linux/amd64. OpenElixir
// binds the complete closure by exact bytes without launching anything: the
// elixir and mix launchers and the complete Elixir lib tree (ExUnit and Mix
// libraries included), plus the erl launcher, the pinned ERTS directory and
// the complete Erlang/OTP installation tree. The runtime version identity
// is proven by the scoped probes in Validate; Close is the final
// process-free byte/topology/root-identity revalidation. There is no
// user-selectable adapter runtime API.
const (
	// ElixirProfile is the runtime profile identity of the
	// elixir-exunit/v1 closure.
	ElixirProfile = elixirRuntimeProfile
	// RequiredElixirVersion is the exact first-release Elixir/Mix/ExUnit
	// version.
	RequiredElixirVersion = "1.20.4"
	// RequiredOTPVersion is the exact first-release OTP version.
	RequiredOTPVersion = "29.0.6"
	// RequiredErtsVersion is the exact first-release ERTS version.
	RequiredErtsVersion = "17.0.6"
	// RequiredOTPMajor is the OTP release-series identity of the catalog.
	RequiredOTPMajor = "29"
	// ElixirIdentityVersion is the RuntimeRef version spelling of the
	// three-part pinned closure identity.
	ElixirIdentityVersion = "1.20.4/29.0.6/17.0.6"

	elixirClosureDomain      = "machinery.tdd.runtime.elixir-exunit/v1"
	elixirLauncherVersionTag = "ELIXIR_VERSION=" + RequiredElixirVersion
	elixirProbeDeadlineMS    = int64(45000)
	elixirProbeOutputLimit   = int64(1 << 20)
)

// elixirRuntimeProfile is unexported so no other package can conjure the
// identity without this file's pinned constants.
const elixirRuntimeProfile = "elixir-exunit/v1"

// elixirVersionIdentity matches the OTP release-series identity inside the
// runtime's own version output.
var elixirVersionIdentity = regexp.MustCompile(`Erlang/OTP ` + RequiredOTPMajor + `([^\d.]|$)`)

// ElixirRequest is the closed open request of the pinned Elixir runtime
// closure.
type ElixirRequest struct {
	// ElixirPath optionally names the elixir launcher. Empty resolves the
	// host runtime from PATH through its real symlink chain.
	ElixirPath string
	// ErlangPath optionally names the erl launcher. Empty resolves the
	// host runtime from PATH through its real symlink chain.
	ErlangPath string
	// ExpectedClosure, when nonempty, is an exact lowercase sha256 the
	// opened closure must reproduce; a mismatch fails closed.
	ExpectedClosure string
}

// Elixir is the pinned Elixir 1.20.4 / OTP 29.0.6 (ERTS 17.0.6) runtime
// handle implementing tdd.RuntimeHandle. The retained handles cover the
// Elixir runtime root (launchers plus lib tree) and the Erlang/OTP
// installation root (launcher, ERTS and lib tree); the closure digest
// binds both trees by exact bytes and topology.
type Elixir struct {
	elixirBin      string
	mixBin         string
	erlBin         string
	elixirRootPath string
	erlangRootPath string
	elixirRootInfo os.FileInfo
	erlangRootInfo os.FileInfo
	elixirRoot     *os.Root
	erlangRoot     *os.Root
	elixirBody     []byte
	mixBody        []byte
	erlBody        []byte
	elixirDigest   string
	mixDigest      string
	erlDigest      string
	elixirTree     string
	erlangTree     string
	closureDigest  string
	identity       tdd.RuntimeRef
	closed         bool
}

// Compile-time contract check: the closed RuntimeHandle boundary.
var _ tdd.RuntimeHandle = (*Elixir)(nil)

// OpenElixir opens and binds the pinned Elixir runtime closure. It never
// launches a process: the Elixir version is pre-bound from the launcher's
// own embedded version tag, the OTP/ERTS identity from the installation's
// own releases data, both launcher scripts are retained by exact bytes,
// and the complete Elixir and Erlang/OTP trees are fingerprinted into the
// closure digest.
func OpenElixir(ctx context.Context, req ElixirRequest) (*Elixir, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: cannot open the pinned Elixir %s runtime closure: %w", RequiredElixirVersion, err)
	}
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if platform != "darwin/arm64" && platform != "linux/amd64" {
		return nil, fmt.Errorf("UNSUPPORTED_PLATFORM: %s is not a pinned native assurance platform for the Elixir runtime closure", platform)
	}
	elixirSource := req.ElixirPath
	if elixirSource == "" {
		discovered, err := exec.LookPath("elixir")
		if err != nil {
			return nil, fmt.Errorf("UNSUPPORTED_VERSION: the pinned Elixir %s runtime is absent from PATH: %w", RequiredElixirVersion, err)
		}
		elixirSource = discovered
	}
	elixirSourceAbs, err := filepath.Abs(elixirSource)
	if err != nil {
		return nil, err
	}
	elixirReal, err := filepath.EvalSymlinks(elixirSourceAbs)
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: resolve elixir launcher: %w", err)
	}
	mixSource := filepath.Join(filepath.Dir(elixirReal), "mix")
	mixReal, err := filepath.EvalSymlinks(mixSource)
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: the mix launcher is not installed beside elixir (%s): %w", mixSource, err)
	}
	elixirRootPath := filepath.Dir(filepath.Dir(elixirReal))
	if err := validateElixirLayout(elixirRootPath); err != nil {
		return nil, err
	}
	elixirBinBody, err := os.ReadFile(elixirReal)
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: read the elixir launcher: %w", err)
	}
	mixBody, err := os.ReadFile(mixReal)
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: read the mix launcher: %w", err)
	}
	if !strings.Contains(string(elixirBinBody), elixirLauncherVersionTag) {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: the elixir launcher at %s does not carry its own %s identity", elixirReal, elixirLauncherVersionTag)
	}
	erlangSource := req.ErlangPath
	if erlangSource == "" {
		discovered, err := exec.LookPath("erl")
		if err != nil {
			return nil, fmt.Errorf("UNSUPPORTED_VERSION: the pinned Erlang/OTP %s runtime is absent from PATH: %w", RequiredOTPVersion, err)
		}
		erlangSource = discovered
	}
	erlangSourceAbs, err := filepath.Abs(erlangSource)
	if err != nil {
		return nil, err
	}
	erlReal, err := filepath.EvalSymlinks(erlangSourceAbs)
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: resolve erl launcher: %w", err)
	}
	erlangRootPath := filepath.Dir(filepath.Dir(erlReal))
	if resolved := filepath.Join(erlangRootPath, "lib", "erlang"); dirExists(resolved) {
		erlangRootPath = resolved
	}
	if err := validateErlangLayout(erlangRootPath); err != nil {
		return nil, err
	}
	erlBody, err := os.ReadFile(erlReal)
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: read the erl launcher: %w", err)
	}
	handle := &Elixir{
		elixirBin: elixirReal, mixBin: mixReal, erlBin: erlReal,
		elixirRootPath: elixirRootPath, erlangRootPath: erlangRootPath,
		elixirBody: elixirBinBody, mixBody: mixBody, erlBody: erlBody,
	}
	elixirDigest := typeScriptDigestBytes(elixirBinBody)
	mixDigest := typeScriptDigestBytes(mixBody)
	erlDigest := typeScriptDigestBytes(erlBody)
	handle.elixirDigest, handle.mixDigest, handle.erlDigest = elixirDigest, mixDigest, erlDigest

	elixirRootInfo, err := os.Lstat(elixirRootPath)
	if err != nil || !elixirRootInfo.IsDir() {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: the Elixir runtime root %s must be a real directory", elixirRootPath)
	}
	elixirRoot, err := os.OpenRoot(elixirRootPath)
	if err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: open the Elixir runtime root: %w", err)
	}
	openedElixir, err := elixirRoot.Lstat(".")
	if err != nil || !os.SameFile(elixirRootInfo, openedElixir) {
		return nil, errors.Join(err, elixirRoot.Close(), fmt.Errorf("UNSUPPORTED_VERSION: the Elixir runtime root changed identity while opening"))
	}
	erlangRootInfo, err := os.Lstat(erlangRootPath)
	if err != nil || !erlangRootInfo.IsDir() {
		return nil, errors.Join(elixirRoot.Close(), fmt.Errorf("UNSUPPORTED_VERSION: the Erlang/OTP root %s must be a real directory", erlangRootPath))
	}
	erlangRoot, err := os.OpenRoot(erlangRootPath)
	if err != nil {
		return nil, errors.Join(elixirRoot.Close(), fmt.Errorf("UNSUPPORTED_VERSION: open the Erlang/OTP root: %w", err))
	}
	openedErlang, err := erlangRoot.Lstat(".")
	if err != nil || !os.SameFile(erlangRootInfo, openedErlang) {
		return nil, errors.Join(err, elixirRoot.Close(), erlangRoot.Close(), fmt.Errorf("UNSUPPORTED_VERSION: the Erlang/OTP root changed identity while opening"))
	}
	elixirTree, err := fingerprintElixirTree(elixirRootPath, elixirRoot, "Elixir runtime tree")
	if err != nil {
		return nil, errors.Join(elixirRoot.Close(), erlangRoot.Close(), fmt.Errorf("UNSUPPORTED_VERSION: fingerprint the Elixir runtime tree: %w", err))
	}
	erlangTree, err := fingerprintElixirTree(erlangRootPath, erlangRoot, "Erlang/OTP tree")
	if err != nil {
		return nil, errors.Join(elixirRoot.Close(), erlangRoot.Close(), fmt.Errorf("UNSUPPORTED_VERSION: fingerprint the Erlang/OTP tree: %w", err))
	}
	handle.elixirRootInfo, handle.erlangRootInfo = elixirRootInfo, erlangRootInfo
	handle.elixirRoot, handle.erlangRoot = elixirRoot, erlangRoot
	handle.elixirTree, handle.erlangTree = elixirTree, erlangTree
	handle.closureDigest = elixirClosureDigest(elixirTree, erlangTree)
	handle.identity = tdd.RuntimeRef{
		Profile: ElixirProfile, Version: ElixirIdentityVersion,
		Platform: platform, Closure: handle.closureDigest,
	}
	if req.ExpectedClosure != "" {
		if req.ExpectedClosure != handle.closureDigest {
			_ = handle.Close()
			return nil, fmt.Errorf("UNSUPPORTED_VERSION: explicit Elixir runtime closure sha256:%s does not match paired trust root sha256:%s", handle.closureDigest, req.ExpectedClosure)
		}
	}
	return handle, nil
}

// Identity implements tdd.RuntimeHandle.
func (e *Elixir) Identity() tdd.RuntimeRef { return e.identity }

// ElixirPath returns the resolved real elixir launcher of the closure.
func (e *Elixir) ElixirPath() string { return e.elixirBin }

// MixPath returns the resolved real mix launcher of the closure.
func (e *Elixir) MixPath() string { return e.mixBin }

// ErlangPath returns the resolved real erl launcher of the closure.
func (e *Elixir) ErlangPath() string { return e.erlBin }

// ElixirBinDir returns the directory holding the elixir/mix launchers.
func (e *Elixir) ElixirBinDir() string { return filepath.Dir(e.elixirBin) }

// ErlangBinDir returns the directory holding the erl launcher.
func (e *Elixir) ErlangBinDir() string { return filepath.Dir(e.erlBin) }

// Validate implements tdd.RuntimeHandle: it performs the runtime's own
// identity probes (elixir --version, mix --version) as real scoped
// subprocesses while the supplied scope is live, then re-checks the
// retained byte identities.
func (e *Elixir) Validate(ctx context.Context, scope processscope.Scope) error {
	if e.closed {
		return fmt.Errorf("UNSUPPORTED_VERSION: the Elixir runtime closure handle is closed")
	}
	if scope == nil {
		return fmt.Errorf("INVALID_SCHEMA: Elixir closure validation requires a live processscope scope")
	}
	probeEnv := []string{
		"PATH=" + e.ErlangBinDir() + string(os.PathListSeparator) + e.ElixirBinDir() + string(os.PathListSeparator) + "/usr/bin" + string(os.PathListSeparator) + "/bin",
		"HOME=" + os.TempDir(),
		"TMPDIR=" + os.TempDir(),
		"LC_ALL=C.UTF-8",
		"NO_COLOR=1",
	}
	if err := elixirProbe(ctx, scope, e, e.elixirBin, probeEnv, "elixir", func(out string) error {
		if !strings.Contains(out, "Elixir "+RequiredElixirVersion) ||
			!strings.Contains(out, "erts-"+RequiredErtsVersion) ||
			!elixirVersionIdentity.MatchString(out) {
			return fmt.Errorf("elixir runtime identity %q does not bind Elixir %s / OTP %s / erts-%s", firstLine(out), RequiredElixirVersion, RequiredOTPMajor, RequiredErtsVersion)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := elixirProbe(ctx, scope, e, e.mixBin, probeEnv, "mix", func(out string) error {
		if !strings.Contains(out, "Mix "+RequiredElixirVersion) {
			return fmt.Errorf("mix runtime identity %q is not the exact pinned Mix %s", firstLine(out), RequiredElixirVersion)
		}
		return nil
	}); err != nil {
		return err
	}
	return e.revalidate()
}

// Close implements tdd.RuntimeHandle: the final process-free
// byte/topology/root-identity revalidation before releasing the retained
// handles. It never launches a process.
func (e *Elixir) Close() error {
	if e.closed {
		return fmt.Errorf("UNSUPPORTED_VERSION: the Elixir runtime closure handle is already closed")
	}
	e.closed = true
	err := e.revalidate()
	return errors.Join(err, e.elixirRoot.Close(), e.erlangRoot.Close())
}

// revalidate re-digests the retained launcher bytes and re-fingerprints
// both runtime trees without launching anything.
func (e *Elixir) revalidate() error {
	for _, member := range []struct {
		path    string
		body    []byte
		digest  string
		subject string
	}{
		{e.elixirBin, e.elixirBody, e.elixirDigest, "elixir launcher"},
		{e.mixBin, e.mixBody, e.mixDigest, "mix launcher"},
		{e.erlBin, e.erlBody, e.erlDigest, "erl launcher"},
	} {
		info, err := os.Lstat(member.path)
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("UNSUPPORTED_VERSION: the retained %s changed identity after verification", member.subject)
		}
		body, err := os.ReadFile(member.path)
		if err != nil || typeScriptDigestBytes(body) != member.digest {
			return fmt.Errorf("UNSUPPORTED_VERSION: the retained %s changed content after verification", member.subject)
		}
	}
	elixirRootInfo, err := os.Lstat(e.elixirRootPath)
	if err != nil || !os.SameFile(e.elixirRootInfo, elixirRootInfo) {
		return fmt.Errorf("UNSUPPORTED_VERSION: the Elixir runtime root changed identity after verification")
	}
	erlangRootInfo, err := os.Lstat(e.erlangRootPath)
	if err != nil || !os.SameFile(e.erlangRootInfo, erlangRootInfo) {
		return fmt.Errorf("UNSUPPORTED_VERSION: the Erlang/OTP root changed identity after verification")
	}
	elixirTree, err := fingerprintElixirTree(e.elixirRootPath, e.elixirRoot, "Elixir runtime tree")
	if err != nil {
		return fmt.Errorf("UNSUPPORTED_VERSION: revalidate the Elixir runtime tree: %w", err)
	}
	erlangTree, err := fingerprintElixirTree(e.erlangRootPath, e.erlangRoot, "Erlang/OTP tree")
	if err != nil {
		return fmt.Errorf("UNSUPPORTED_VERSION: revalidate the Erlang/OTP tree: %w", err)
	}
	if elixirTree != e.elixirTree || erlangTree != e.erlangTree {
		return fmt.Errorf("UNSUPPORTED_VERSION: the Elixir runtime closure changed after verification (was sha256:%s, now sha256:%s)", e.closureDigest, elixirClosureDigest(elixirTree, erlangTree))
	}
	return nil
}

// elixirProbe runs one identity probe in its own child scope of the
// supplied live scope and closes the child with verified terminal cleanup.
func elixirProbe(ctx context.Context, scope processscope.Scope, handle *Elixir, executable string, env []string, name string, check func(string) error) error {
	child, err := scope.Child(ctx)
	if err != nil {
		return fmt.Errorf("CUSTODY_ERROR: open the %s identity probe custody child: %w", name, err)
	}
	var stdout, stderr bytes.Buffer
	result, runErr := func() (processscope.Result, error) {
		attached, err := child.Attach(processscope.Command{
			Executable:    executable,
			Args:          []string{"--version"},
			Dir:           os.TempDir(),
			Env:           env,
			RuntimeDigest: "sha256:" + handle.launcherDigest(executable),
			DeadlineMS:    elixirProbeDeadlineMS,
		})
		if err != nil {
			return processscope.Result{}, fmt.Errorf("CUSTODY_ERROR: attach the %s identity probe: %w", name, err)
		}
		return child.Run(ctx, attached, processscope.Streams{
			Stdout: &stdout, Stderr: &stderr, StdoutLimit: elixirProbeOutputLimit, StderrLimit: elixirProbeOutputLimit,
		})
	}()
	closeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	report, closeErr := child.Close(closeCtx)
	if closeErr != nil || report.Status != processscope.StatusCleaned {
		return errors.Join(runErr, closeErr, fmt.Errorf("CUSTODY_ERROR: the %s probe custody child cleanup did not verify: %+v", name, report))
	}
	if runErr != nil {
		return fmt.Errorf("CUSTODY_ERROR: the %s identity probe failed under custody: %w (stderr: %s)", name, runErr, boundedElixir(stderr.String()))
	}
	if !result.Completed || result.ExitCode != 0 || result.Signal != "" {
		return fmt.Errorf("UNSUPPORTED_VERSION: the %s identity probe exited abnormally (completed=%v exit=%d signal=%q)", name, result.Completed, result.ExitCode, result.Signal)
	}
	if err := check(stdout.String()); err != nil {
		return fmt.Errorf("UNSUPPORTED_VERSION: %w", err)
	}
	return nil
}

func boundedElixir(s string) string {
	if len(s) > 2048 {
		return s[:2048] + "...(bounded)"
	}
	return s
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

func dirExists(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir()
}

// validateElixirLayout validates the Elixir runtime root layout: its bin
// launchers and the compiled lib tree carrying ExUnit and Mix.
func validateElixirLayout(root string) error {
	if filepath.Base(root) != "elixir" {
		return fmt.Errorf("UNSUPPORTED_VERSION: the resolved Elixir runtime root %s does not carry the elixir installation layout", root)
	}
	if !dirExists(filepath.Join(root, "bin")) || !dirExists(filepath.Join(root, "lib")) {
		return fmt.Errorf("UNSUPPORTED_VERSION: the resolved Elixir runtime root %s lacks its bin/lib tree", root)
	}
	for _, member := range []string{"ex_unit", "mix"} {
		if !dirExists(filepath.Join(root, "lib", member, "ebin")) {
			return fmt.Errorf("UNSUPPORTED_VERSION: the resolved Elixir runtime root %s lacks its compiled %s library", root, member)
		}
	}
	return nil
}

// validateErlangLayout validates the Erlang/OTP installation layout by its
// own release records: the pinned ERTS directory and the start_erl.data
// binding of ERTS and OTP release identities.
func validateErlangLayout(root string) error {
	ertsDir := filepath.Join(root, "erts-"+RequiredErtsVersion)
	if !dirExists(ertsDir) {
		return fmt.Errorf("UNSUPPORTED_VERSION: the Erlang/OTP root %s lacks its pinned erts-%s directory", root, RequiredErtsVersion)
	}
	data, err := os.ReadFile(filepath.Join(root, "releases", "start_erl.data"))
	if err != nil || len(data) > 256 {
		return fmt.Errorf("UNSUPPORTED_VERSION: the Erlang/OTP root %s lacks its start_erl.data release identity", root)
	}
	if want := RequiredErtsVersion + " " + RequiredOTPMajor + "\n"; string(data) != want {
		return fmt.Errorf("UNSUPPORTED_VERSION: the Erlang/OTP root %s reports start_erl.data %q, want %q", root, string(data), want)
	}
	otpVersion, err := os.ReadFile(filepath.Join(root, "releases", RequiredOTPMajor, "OTP_VERSION"))
	if err != nil || len(otpVersion) > 64 {
		return fmt.Errorf("UNSUPPORTED_VERSION: the Erlang/OTP root %s lacks its OTP_VERSION release record", root)
	}
	if strings.TrimSpace(string(otpVersion)) != RequiredOTPVersion {
		return fmt.Errorf("UNSUPPORTED_VERSION: the Erlang/OTP root %s reports OTP %q, want exactly %s", root, strings.TrimSpace(string(otpVersion)), RequiredOTPVersion)
	}
	return nil
}

// fingerprintElixirTree hashes the complete runtime tree topologically
// under the elixir closure domain: sorted directory and file identities,
// exact file bytes, file symlinks resolved and bound by their target
// identity plus the resolved bytes, bounded entries/bytes/depth, with a
// stable-identity double census. The pinned trees are small (Elixir ~0.5k
// entries, OTP ~4.5k) against the shipped bounds.
func fingerprintElixirTree(rootPath string, root *os.Root, label string) (string, error) {
	budget := &elixirTreeBudget{label: label, entries: 0, bytes: 0}
	// The root spelling the caller holds may itself traverse symlinks; every
	// containment decision below is taken against its fully resolved form so
	// an in-tree link is never mistaken for an escape.
	resolvedRoot, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		return "", fmt.Errorf("resolve the runtime tree root %s: %w", rootPath, err)
	}
	type entry struct {
		name  string
		kind  byte
		size  int64
		extra string
	}
	entries := make([]entry, 0, 1024)
	var walk func(rel string, depth int) error
	walk = func(rel string, depth int) error {
		if depth > elixirTreeMaxDepth {
			return fmt.Errorf("depth bound exceeded at %s", rel)
		}
		info, err := root.Lstat(rel)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			bound, err := bindElixirLink(rootPath, resolvedRoot, rel)
			if err != nil {
				return err
			}
			if err := budget.add(1, bound.size); err != nil {
				return err
			}
			entries = append(entries, entry{name: rel, kind: bound.kind, size: bound.size, extra: bound.target + "\x00" + bound.digest})
			return nil
		}
		if !info.IsDir() {
			return fmt.Errorf("entry %s must be a directory, regular file or in-tree symlink", rel)
		}
		if err := budget.add(1, 0); err != nil {
			return err
		}
		entries = append(entries, entry{name: rel, kind: 'd'})
		opened, err := root.Open(rel)
		if err != nil {
			return err
		}
		dirents, readErr := opened.ReadDir(-1)
		closeErr := opened.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		names := make([]string, 0, len(dirents))
		for _, dirent := range dirents {
			names = append(names, dirent.Name())
		}
		sortStrings(names)
		for _, name := range names {
			child := name
			if rel != "." {
				child = rel + "/" + name
			}
			childInfo, err := root.Lstat(child)
			if err != nil {
				return err
			}
			switch {
			case childInfo.Mode()&os.ModeSymlink != 0:
				if err := walk(child, depth+1); err != nil {
					return err
				}
			case childInfo.IsDir():
				if err := walk(child, depth+1); err != nil {
					return err
				}
			case childInfo.Mode().IsRegular():
				if childInfo.Size() > elixirTreeMaxFile {
					return fmt.Errorf("file %s exceeds the closure byte bound", child)
				}
				if err := budget.add(1, childInfo.Size()); err != nil {
					return err
				}
				digest, err := hashElixirFile(root, child, childInfo)
				if err != nil {
					return err
				}
				entries = append(entries, entry{name: child, kind: 'f', size: childInfo.Size(), extra: digest})
			default:
				return fmt.Errorf("closure member %s is not a regular file, directory or symlink", child)
			}
		}
		return nil
	}
	if err := walk(".", 0); err != nil {
		return "", err
	}
	hash := sha256.New()
	hash.Write([]byte(elixirClosureDomain + "\x00" + label + "\x00"))
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	for _, item := range entries {
		slashName := filepath.ToSlash(item.name)
		switch item.kind {
		case 'd':
			_, _ = fmt.Fprintf(hash, "d\x00%s\x00", slashName)
		case 'l':
			_, _ = fmt.Fprintf(hash, "l\x00%s\x00%d\x00%s\x00", slashName, item.size, item.extra)
		case 'x':
			_, _ = fmt.Fprintf(hash, "x\x00%s\x00%d\x00%s\x00", slashName, item.size, item.extra)
		default:
			_, _ = fmt.Fprintf(hash, "f\x00%s\x00%d\x00%s\x00", slashName, item.size, item.extra)
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// elixirLinkBinding is one symlink's contribution to the tree fingerprint:
// its topological kind, the identity of what it points at, and the sha256 of
// the exact bytes it delivers.
type elixirLinkBinding struct {
	kind   byte
	target string
	digest string
	size   int64
}

// elixirOutsideTarget is the path-independent identity recorded for a link
// whose resolved target lies outside the runtime tree. The bytes are still
// bound exactly by digest; only the absolute install location, which differs
// per host, is deliberately left out of the closure identity.
const elixirOutsideTarget = "<outside-tree>"

// bindElixirLink resolves one symlink and binds it by content.
//
// A link resolving inside the tree keeps its root-relative target identity
// (kind 'l'). A link resolving outside is bound by its exact bytes under a
// distinct kind ('x') and a path-independent target marker, so it can never
// collide with an in-tree binding and any change to the bytes it delivers
// still changes the closure digest.
//
// Rejecting out-of-tree targets outright is not sound as a completeness
// claim and is wrong in practice: erlef/setup-beam installs Erlang/OTP by
// copying the tool cache with Node's fs.cpSync, which rewrites every relative
// symlink to an absolute path into the source tree. On a hosted macOS runner
// $RUNNER_TEMP/.setup-beam/otp/bin/epmd therefore points at
// $RUNNER_TOOL_CACHE/otp/<version>/<arch>/erts-*/bin/epmd. That is a real,
// unmodified vendor layout, and the bytes it delivers are bound here exactly
// as the in-tree copy's are. Only an unresolvable, non-regular or unbounded
// target fails closed, and the diagnostic names both the link and what it
// resolved to.
func bindElixirLink(rootPath, resolvedRoot, rel string) (elixirLinkBinding, error) {
	absolute := filepath.Join(rootPath, filepath.FromSlash(rel))
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return elixirLinkBinding{}, fmt.Errorf("resolve symlink %s: %w", rel, err)
	}
	binding := elixirLinkBinding{kind: 'x', target: elixirOutsideTarget}
	relative, relErr := filepath.Rel(resolvedRoot, resolved)
	if relErr == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative) {
		binding.kind, binding.target = 'l', filepath.ToSlash(relative)
	}
	info, err := os.Lstat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return elixirLinkBinding{}, fmt.Errorf("symlink %s resolves to %s, which is not a regular file", rel, resolved)
	}
	if info.Size() > elixirTreeMaxFile {
		return elixirLinkBinding{}, fmt.Errorf("symlink %s resolves to %s, which exceeds the closure byte bound", rel, resolved)
	}
	body, err := os.ReadFile(resolved)
	if err != nil || int64(len(body)) != info.Size() {
		return elixirLinkBinding{}, fmt.Errorf("symlink %s target %s changed while reading", rel, resolved)
	}
	sum := sha256.Sum256(body)
	binding.digest, binding.size = hex.EncodeToString(sum[:]), int64(len(body))
	return binding, nil
}

// hashElixirFile hashes one regular file's exact bytes under the root.
func hashElixirFile(root *os.Root, name string, before os.FileInfo) (string, error) {
	if before.Size() < 0 {
		return "", fmt.Errorf("negative size")
	}
	file, err := root.Open(name)
	if err != nil {
		return "", err
	}
	opened, statErr := file.Stat()
	if statErr != nil || !sameJavaFileSnapshot(before, opened) {
		return "", errors.Join(statErr, file.Close(), fmt.Errorf("changed identity while opening"))
	}
	hash := sha256.New()
	written, copyErr := io.Copy(hash, io.LimitReader(file, before.Size()+1))
	closeErr := file.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return "", err
	}
	if written != before.Size() {
		return "", fmt.Errorf("changed size while hashing")
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

type elixirTreeBudget struct {
	label   string
	entries int64
	bytes   int64
}

func (b *elixirTreeBudget) add(entries, bytes int64) error {
	b.entries += entries
	b.bytes += bytes
	if b.entries > elixirTreeMaxEntries {
		return fmt.Errorf("%s entry bound exceeded", b.label)
	}
	if b.bytes > elixirTreeMaxBytes {
		return fmt.Errorf("%s byte bound exceeded", b.label)
	}
	return nil
}

const (
	elixirTreeMaxEntries = int64(20000)
	elixirTreeMaxDepth   = 24
	elixirTreeMaxBytes   = int64(768 << 20)
	elixirTreeMaxFile    = int64(512 << 20)
)

func sortStrings(names []string) { sort.Strings(names) }

func elixirClosureDigest(elixirTree, erlangTree string) string {
	sum := sha256.Sum256([]byte(elixirClosureDomain + "\x00" + elixirTree + "\x00" + erlangTree))
	return hex.EncodeToString(sum[:])
}

// launcherDigest returns the retained byte digest of one launcher.
func (e *Elixir) launcherDigest(executable string) string {
	switch executable {
	case e.mixBin:
		return e.mixDigest
	case e.erlBin:
		return e.erlDigest
	default:
		return e.elixirDigest
	}
}
