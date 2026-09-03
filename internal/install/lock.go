package install

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/filelock"
)

const installLockCapabilityEnv = "MACHINERY_INTERNAL_INSTALL_LOCK_CAPABILITY"
const activationCanonicalExecutableEnv = "MACHINERY_INTERNAL_CANONICAL_EXECUTABLE"

var (
	installControlDirOwned = installDirectoryOwnedByCurrentUser
	installFileLockRelease = (*filelock.Lock).Release
)

// installOperationLock serializes every install/update mutation belonging to
// one installation receipt. The advisory lock lives outside the install-owned
// artifact set and is released by the kernel if the process dies.
type installOperationLock struct {
	lock *filelock.Lock
}

// ActivationRecoveryError means recovery restored the executable image that
// launched this process. The in-memory program is now newer than the durable
// companion artifacts, so no caller logic may continue in this process.
type ActivationRecoveryError struct {
	Executable     string
	Identity       string
	file           *os.File
	lock           *filelock.Lock
	activationPath string
}

func (e *ActivationRecoveryError) Error() string {
	return fmt.Sprintf("interrupted update restored executable %s; re-exec is required before continuing", e.Executable)
}

// ActivationRecoveryExecutable extracts the restored executable that must be
// re-executed before any command or host-governance logic may continue.
func ActivationRecoveryExecutable(err error) (string, bool) {
	var activation *ActivationRecoveryError
	if !errors.As(err, &activation) {
		return "", false
	}
	return activation.Executable, true
}

// ValidateActivationRecovery binds a recovery signal to the exact restored
// executable bytes and metadata before the CLI replaces its process image.
func ValidateActivationRecovery(err error) error {
	var activation *ActivationRecoveryError
	if !errors.As(err, &activation) {
		return fmt.Errorf("error is not an executable activation recovery signal")
	}
	if activation.Executable == "" || activation.Identity == "" || activation.file == nil || activation.lock == nil {
		return fmt.Errorf("activation recovery signal is incomplete")
	}
	identity, identityErr := activationOpenFileIdentity(activation.file)
	if identityErr != nil {
		return identityErr
	}
	if identity != activation.Identity {
		return fmt.Errorf("restored executable identity changed before re-exec")
	}
	if err := validateActivationExecutablePath(activation); err != nil {
		return err
	}
	return nil
}

// Close releases a retained activation without executing it. Production CLI
// startup uses ReexecActivationRecovery; Close exists for fail-closed library
// callers and tests that elect to terminate instead.
func (e *ActivationRecoveryError) Close() error {
	if e == nil {
		return nil
	}
	var errs []error
	if e.file != nil {
		errs = append(errs, closeInstallFile(e.file))
		e.file = nil
	}
	if e.activationPath != "" {
		errs = append(errs, cleanupActivationExecutable())
		e.activationPath = ""
	}
	if e.lock != nil {
		errs = append(errs, e.lock.Release())
		e.lock = nil
	}
	return errors.Join(errs...)
}

// CloseActivationRecovery releases a retained activation signal without
// executing it.
func CloseActivationRecovery(err error) error {
	var activation *ActivationRecoveryError
	if !errors.As(err, &activation) {
		return nil
	}
	return activation.Close()
}

func acquireInstallOperationLock() (*installOperationLock, error) {
	if err := prepareExistingInstallControlDirectory(); err != nil {
		return nil, err
	}
	scope, err := installOperationScope()
	if err != nil {
		return nil, err
	}
	capability := os.Getenv(installLockCapabilityEnv)
	lock, lockErr := filelock.Acquire(scope)
	if lockErr == nil {
		if err := cleanupActivationExecutable(); err != nil {
			return nil, errors.Join(fmt.Errorf("clean prior executable activation: %w", err), installFileLockRelease(lock))
		}
		recovery, err := recoverInstallTransaction()
		if err != nil {
			return nil, errors.Join(fmt.Errorf("recover interrupted install transaction: %w", err), installFileLockRelease(lock))
		}
		if recovery.restoredExecutable != "" {
			return nil, &ActivationRecoveryError{
				Executable:     recovery.restoredExecutable,
				Identity:       recovery.executableIdentity,
				file:           recovery.executableFile,
				lock:           lock,
				activationPath: recovery.activationPath,
			}
		}
		// A child carrying a delegated capability may proceed only while the
		// parent demonstrably owns the real lock. Acquiring it here proves the
		// parent has exited or released early; recovery is complete, but letting
		// the child continue would mix rolled-back parent state with a subset of
		// newly refreshed artifacts.
		if capability != "" {
			return nil, errors.Join(fmt.Errorf("delegated install operation lost its parent lock"), installFileLockRelease(lock))
		}
		return &installOperationLock{lock: lock}, nil
	}

	// Update holds the operation lock while asking the newly installed binary
	// to refresh direct harnesses. Only that child receives a short-lived,
	// parent-bound capability. If the parent died, the advisory lock is free and
	// the acquisition above succeeds, so an abandoned capability cannot bypass
	// serialization.
	if err := validateInstallLockCapability(capability, scope); err == nil {
		return &installOperationLock{}, nil
	}
	return nil, lockErr
}

// EnsureActivationConsistency is the process-wide startup barrier. Every CLI
// command calls it before parsing or executing command logic, so a process
// launched from a partially updated binary either observes a committed
// activation or receives ActivationRecoveryError and re-executes the restored
// image. Library entry points retain the same barrier through their operation
// lock acquisition.
func EnsureActivationConsistency() error {
	lock, err := acquireInstallOperationLock()
	if err != nil {
		return err
	}
	return lock.Release()
}

// prepareExistingInstallControlDirectory establishes the owned capability
// boundary before a journal can be created beneath it. Public ancestors (for
// example macOS /var) are outside this boundary; only the configured leaf is
// required to be a private real directory. An absent leaf is created as 0700
// later, after the operation lock is held. Legacy 0755 leaves are migrated
// only after an owner and stable-identity check through a retained root.
func prepareExistingInstallControlDirectory() (retErr error) {
	receipt, err := installationReceiptPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(receipt)
	info, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("install control directory %s is not a real directory", dir)
	}
	if err := filelock.ValidatePrivateDir(dir, info); err == nil {
		return nil
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("open legacy install control directory: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return errors.Join(fmt.Errorf("install control directory changed while opening"), err)
	}
	if !installControlDirOwned(opened) {
		return fmt.Errorf("install control directory %s is not owned by the current user", dir)
	}
	if err := root.Chmod(".", 0o700); err != nil {
		return fmt.Errorf("migrate legacy install control directory permissions: %w", err)
	}
	if err := syncRootRelativeDir(root, "."); err != nil {
		return fmt.Errorf("persist migrated install control directory permissions: %w", err)
	}
	current, err := os.Lstat(dir)
	if err != nil || !os.SameFile(opened, current) {
		return errors.Join(fmt.Errorf("install control directory changed during permission migration"), err)
	}
	if err := filelock.ValidatePrivateDir(dir, current); err != nil {
		return fmt.Errorf("install control directory remains unconfined after migration: %w", err)
	}
	return nil
}

func acquireInstallOperationLockWait() (*installOperationLock, error) {
	deadline := time.Now().Add(10 * time.Second)
	for {
		lock, err := acquireInstallOperationLock()
		if err == nil {
			return lock, nil
		}
		if !strings.Contains(err.Error(), "another operation holds the lock") || time.Now().After(deadline) {
			return nil, err
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (l *installOperationLock) Release() error {
	if l == nil || l.lock == nil {
		return nil
	}
	return l.lock.Release()
}

// WithInstallInspectionLock recovers any interrupted artifact transaction and
// holds the same operation lock used by install/update/uninstall for the full
// inspection callback. Doctor and preflight therefore cannot observe a
// PREPARED transaction's partial state or race a concurrent mutation.
func WithInstallInspectionLock(inspect func() error) (retErr error) {
	lock, err := acquireInstallOperationLock()
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, lock.Release()) }()
	return inspect()
}

func installOperationScope() (string, error) {
	path, err := installationReceiptPath()
	if err != nil {
		return "", fmt.Errorf("resolve install operation lock scope: %w", err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve install operation lock scope: %w", err)
	}
	// Resolve the nearest existing prefix even when the receipt itself does not
	// exist yet. Otherwise an alias such as /var -> /private/var can change the
	// lock identity after the first install creates the receipt directory,
	// allowing two in-process operations to hold different locks for one receipt.
	resolved, err := evalInstallPath(filepath.Clean(abs))
	if err != nil {
		return "", fmt.Errorf("resolve install operation lock scope: %w", err)
	}
	return resolved, nil
}

type installLockCapability struct {
	path  string
	token string
	pid   int
}

func createInstallLockCapability(scope string) (installLockCapability, func(), error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return installLockCapability{}, func() {}, fmt.Errorf("create install lock capability: %w", err)
	}
	token := hex.EncodeToString(random[:])
	f, err := os.CreateTemp("", ".machinery-install-lock-capability-*")
	if err != nil {
		return installLockCapability{}, func() {}, fmt.Errorf("create install lock capability: %w", err)
	}
	path := f.Name()
	cleanup := func() { _ = os.Remove(path) }
	fail := func(err error) (installLockCapability, func(), error) {
		_ = f.Close()
		cleanup()
		return installLockCapability{}, func() {}, err
	}
	if err := f.Chmod(0o600); err != nil {
		return fail(fmt.Errorf("secure install lock capability: %w", err))
	}
	payload := strings.Join([]string{scope, token, strconv.Itoa(os.Getpid())}, "\n") + "\n"
	if _, err := f.WriteString(payload); err != nil {
		return fail(fmt.Errorf("write install lock capability: %w", err))
	}
	if err := f.Sync(); err != nil {
		return fail(fmt.Errorf("sync install lock capability: %w", err))
	}
	if err := f.Close(); err != nil {
		cleanup()
		return installLockCapability{}, func() {}, fmt.Errorf("close install lock capability: %w", err)
	}
	return installLockCapability{path: path, token: token, pid: os.Getpid()}, cleanup, nil
}

func (c installLockCapability) String() string {
	return strings.Join([]string{c.path, c.token, strconv.Itoa(c.pid)}, "\n")
}

func validateInstallLockCapability(encoded, scope string) error {
	parts := strings.Split(encoded, "\n")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return fmt.Errorf("invalid install lock capability")
	}
	pid, err := strconv.Atoi(parts[2])
	if err != nil || pid <= 0 || pid != os.Getppid() {
		return fmt.Errorf("install lock capability parent does not match")
	}
	info, err := os.Lstat(parts[0])
	if err != nil {
		return fmt.Errorf("inspect install lock capability: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || !privateFilePermissionsOK(info) {
		return fmt.Errorf("install lock capability is not a private regular file")
	}
	raw, err := readPrivateRegularFile(parts[0], 4096)
	if err != nil {
		return fmt.Errorf("read install lock capability: %w", err)
	}
	want := strings.Join([]string{scope, parts[1], parts[2]}, "\n") + "\n"
	if string(raw) != want {
		return fmt.Errorf("install lock capability does not match")
	}
	return nil
}

func runCombinedWithInstallLockCapability(capability installLockCapability) commandRunner {
	return func(name string, args ...string) (string, error) {
		environment := setActivationEnvironment(os.Environ(), installLockCapabilityEnv, capability.String())
		return runBoundedPluginCommand(environment, name, args...)
	}
}
