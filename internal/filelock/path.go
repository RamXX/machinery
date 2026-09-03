// Package filelock provides process-safe advisory locks keyed by a canonical
// filesystem scope identity.
package filelock

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

type lockLocation struct {
	root *os.Root
	dir  string
	name string
	info os.FileInfo
}

type acquireHooks struct {
	afterLockOpen func(string) error
}

const lockOpenAttempts = 32
const filelockTestRootEnv = "MACHINERY_INTERNAL_TEST_LOCK_ROOT"

var (
	filelockUserCacheDir = os.UserCacheDir
	filelockExecutable   = os.Executable
	filelockTesting      = func() bool { return flag.Lookup("test.v") != nil }
	filelockOpenFile     = func(root *os.Root, name string) (*os.File, error) {
		return root.OpenFile(name, os.O_CREATE|os.O_RDWR, 0o600)
	}
)

func openLockLocation(scope string) (*lockLocation, error) {
	base, err := lockCacheBase()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(base, "machinery", "locks")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if err := ValidatePrivateDir(dir, info); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	inside, err := root.Lstat(".")
	if err != nil || !os.SameFile(info, inside) || info.Mode() != inside.Mode() {
		return nil, errors.Join(err, root.Close(), fmt.Errorf("lock directory %s changed identity while opening", dir))
	}
	if err := ValidatePrivateDir(dir, inside); err != nil {
		return nil, errors.Join(err, root.Close())
	}
	identity, err := ScopeIdentity(scope)
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	sum := sha256.Sum256([]byte(identity))
	return &lockLocation{root: root, dir: dir, name: fmt.Sprintf("%x.lock", sum[:]), info: inside}, nil
}

func lockCacheBase() (string, error) {
	// Every go test binary registers the test.* flag set. Keeping randomized
	// test scopes beside that ephemeral executable prevents repeated suites from
	// leaking permanent zero-byte lock identities into the user's real cache;
	// helper subprocesses execute the same binary and therefore share the root.
	if filelockTesting() {
		if inherited := os.Getenv(filelockTestRootEnv); inherited != "" {
			return inherited, nil
		}
		executable, err := filelockExecutable()
		if err != nil || executable == "" {
			return "", errors.Join(fmt.Errorf("resolve isolated file-lock test root"), err)
		}
		base := filepath.Join(filepath.Dir(executable), ".machinery-test-cache")
		// Activation tests may re-exec an exact staged binary from a read-only
		// directory. Preserve the original writable test root across that exec;
		// the override is honored only by binaries carrying Go's test flags.
		if err := os.Setenv(filelockTestRootEnv, base); err != nil {
			return "", fmt.Errorf("preserve isolated file-lock test root: %w", err)
		}
		return base, nil
	}
	base, err := filelockUserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache directory for file locks: %w", err)
	}
	if strings.TrimSpace(base) == "" {
		return "", fmt.Errorf("resolve user cache directory for file locks: empty path")
	}
	return base, nil
}

func (location *lockLocation) validatePathIdentity() error {
	pathInfo, err := os.Lstat(location.dir)
	if err != nil {
		return fmt.Errorf("reinspect lock directory %s: %w", location.dir, err)
	}
	inside, err := location.root.Lstat(".")
	if err != nil {
		return fmt.Errorf("reinspect rooted lock directory %s: %w", location.dir, err)
	}
	if !os.SameFile(location.info, inside) || !os.SameFile(inside, pathInfo) || location.info.Mode() != inside.Mode() || inside.Mode() != pathInfo.Mode() {
		return fmt.Errorf("lock directory %s changed identity during acquisition", location.dir)
	}
	if err := ValidatePrivateDir(location.dir, pathInfo); err != nil {
		return err
	}
	return nil
}

// openFile tolerates only the transient ENOENT that Darwin/APFS may report
// when O_CREATE lookups race unrelated creates in a very large shared lock
// directory. Before every retry it proves that the retained directory and its
// pathname still name the same private directory, so a removed or replaced
// directory remains a hard error rather than silently creating a split lock.
func (location *lockLocation) openFile() (*os.File, error) {
	var lastErr error
	for attempt := 0; attempt < lockOpenAttempts; attempt++ {
		file, err := filelockOpenFile(location.root, location.name)
		if err == nil {
			return file, nil
		}
		lastErr = err
		if !errors.Is(err, os.ErrNotExist) {
			break
		}
		if identityErr := location.validatePathIdentity(); identityErr != nil {
			return nil, errors.Join(err, identityErr)
		}
		runtime.Gosched()
	}
	return nil, lastErr
}

// ValidatePrivateDir applies the native directory confinement contract used
// by lock and hook state. Unix enforces private permission bits; Windows uses
// the user's cache/temp ACL and therefore must not interpret synthesized
// POSIX bits.
func ValidatePrivateDir(path string, info os.FileInfo) error {
	return validateLockDir(path, info)
}

// ScopeIdentity returns the canonical lock/state identity for scope on the
// running platform.
func ScopeIdentity(scope string) (string, error) {
	return scopeIdentity(scope, runtime.GOOS)
}

func scopeIdentity(scope, goos string) (string, error) {
	abs, err := filepath.Abs(scope)
	if err != nil {
		return "", err
	}
	abs, err = resolveScopePath(abs, filepath.EvalSymlinks, os.Lstat)
	if err != nil {
		return "", err
	}
	return normalizeScopeIdentity(abs, goos), nil
}

func resolveScopePath(abs string, eval func(string) (string, error), lstat func(string) (os.FileInfo, error)) (string, error) {
	current := abs
	var missing []string
	const transitionAttempts = 32
	transitions := 0
	for {
		resolved, err := eval(current)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("resolve filesystem scope %s: %w", abs, err)
		}
		if _, statErr := lstat(current); statErr == nil {
			// A missing leaf may become present between EvalSymlinks and Lstat
			// during first-use initialization. Re-resolve it rather than treating
			// that safe transition as corruption, but bound a dangling-symlink or
			// adversarial ABA loop.
			transitions++
			if transitions >= transitionAttempts {
				return "", fmt.Errorf("resolve filesystem scope %s: identity did not stabilize after %d attempts", current, transitionAttempts)
			}
			runtime.Gosched()
			continue
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return "", fmt.Errorf("inspect filesystem scope %s: %w", current, statErr)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("resolve filesystem scope %s: no existing ancestor", abs)
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

// normalizeScopeIdentity turns the OS spelling of an already-absolute scope
// into the stable key hashed by openLockLocation. It never changes the path used for
// filesystem access. Darwin and Windows filesystems commonly alias case; a
// lock must alias in exactly the same way. Windows additionally admits slash,
// drive-letter, UNC, and extended-length spellings for one volume path.
func normalizeScopeIdentity(abs, goos string) string {
	switch goos {
	case "darwin":
		return strings.ToLower(filepath.Clean(abs))
	case "windows":
		return strings.ToLower(normalizeWindowsPath(abs))
	default:
		return filepath.Clean(abs)
	}
}

func normalizeWindowsPath(value string) string {
	s := strings.ReplaceAll(value, `\`, "/")
	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "//?/unc/"):
		s = "//" + s[len("//?/unc/"):]
	case strings.HasPrefix(lower, "//?/"):
		s = s[len("//?/"):]
	}
	if strings.HasPrefix(s, "//") {
		return "unc:" + path.Clean("/"+strings.TrimLeft(s, "/"))
	}
	if len(s) >= 2 && s[1] == ':' {
		volume := s[:2]
		rest := s[2:]
		if rest == "" {
			rest = "/"
		} else if !strings.HasPrefix(rest, "/") {
			rest = "/" + rest
		}
		return volume + path.Clean(rest)
	}
	return path.Clean(s)
}
