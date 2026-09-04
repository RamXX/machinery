package filelock

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExclusiveAndReacquirable(t *testing.T) {
	scope := t.TempDir()
	first, err := Acquire(scope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(scope); err == nil {
		t.Fatal("concurrent lock acquired")
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	second, err := Acquire(scope)
	if err != nil {
		t.Fatalf("released lock was not reacquirable: %v", err)
	}
	if err := second.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestSharedReadersCoexistAndExcludeWriter(t *testing.T) {
	scope := t.TempDir()
	first, err := AcquireShared(scope)
	if err != nil {
		t.Fatal(err)
	}
	second, err := AcquireShared(scope)
	if err != nil {
		_ = first.Release()
		t.Fatalf("second shared reader was serialized: %v", err)
	}
	if writer, err := Acquire(scope); writer != nil || !IsContended(err) {
		_ = second.Release()
		_ = first.Release()
		t.Fatalf("writer entered while shared readers were active: lock=%v err=%v", writer, err)
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	if writer, err := Acquire(scope); writer != nil || !IsContended(err) {
		_ = second.Release()
		t.Fatalf("writer entered while one shared reader remained: lock=%v err=%v", writer, err)
	}
	if err := second.Release(); err != nil {
		t.Fatal(err)
	}
	writer, err := Acquire(scope)
	if err != nil {
		t.Fatalf("writer did not enter after every shared reader left: %v", err)
	}
	if err := writer.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestSharedReaderWaitsForWriterAndThenSucceeds(t *testing.T) {
	scope := t.TempDir()
	writer, err := Acquire(scope)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	acquired := make(chan *Lock, 1)
	errs := make(chan error, 1)
	go func() {
		reader, err := AcquireSharedWaitContext(ctx, scope)
		if err != nil {
			errs <- err
			return
		}
		acquired <- reader
	}()
	select {
	case reader := <-acquired:
		_ = reader.Release()
		t.Fatal("shared reader entered while the writer was active")
	case err := <-errs:
		t.Fatalf("shared reader failed instead of waiting: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if err := writer.Release(); err != nil {
		t.Fatal(err)
	}
	select {
	case reader := <-acquired:
		if err := reader.Release(); err != nil {
			t.Fatal(err)
		}
	case err := <-errs:
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatalf("shared reader did not enter after writer release: %v", ctx.Err())
	}
}

func TestLockCacheBaseFailsClosedWithoutUserCache(t *testing.T) {
	originalTesting := filelockTesting
	originalCache := filelockUserCacheDir
	t.Cleanup(func() {
		filelockTesting = originalTesting
		filelockUserCacheDir = originalCache
	})
	filelockTesting = func() bool { return false }

	sentinel := errors.New("cache discovery denied")
	filelockUserCacheDir = func() (string, error) { return "", sentinel }
	if _, err := lockCacheBase(); !errors.Is(err, sentinel) || !strings.Contains(err.Error(), "user cache directory") {
		t.Fatalf("cache discovery failure was not preserved: %v", err)
	}
	filelockUserCacheDir = func() (string, error) { return "", nil }
	if _, err := lockCacheBase(); err == nil || !strings.Contains(err.Error(), "empty path") {
		t.Fatalf("empty cache path did not fail closed: %v", err)
	}
}

func TestLockCacheBaseIsIsolatedForTestBinaries(t *testing.T) {
	originalTesting := filelockTesting
	originalExecutable := filelockExecutable
	t.Cleanup(func() {
		filelockTesting = originalTesting
		filelockExecutable = originalExecutable
	})
	filelockTesting = func() bool { return true }
	t.Setenv(filelockTestRootEnv, "")
	executable := filepath.Join(t.TempDir(), "filelock.test")
	filelockExecutable = func() (string, error) { return executable, nil }
	base, err := lockCacheBase()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(filepath.Dir(executable), ".machinery-test-cache")
	if base != want {
		t.Fatalf("test lock cache = %q, want %q", base, want)
	}
	if inherited := os.Getenv(filelockTestRootEnv); inherited != want {
		t.Fatalf("inherited test lock cache = %q, want %q", inherited, want)
	}
	filelockExecutable = func() (string, error) { return "", errors.New("must use inherited root") }
	if inherited, err := lockCacheBase(); err != nil || inherited != want {
		t.Fatalf("re-exec did not retain test lock cache: root=%q err=%v", inherited, err)
	}
}

func TestLockOpenRetriesOnlyTransientENOENT(t *testing.T) {
	originalOpen := filelockOpenFile
	t.Cleanup(func() { filelockOpenFile = originalOpen })
	location, err := openLockLocation(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	filelockOpenFile = func(root *os.Root, name string) (*os.File, error) {
		calls++
		if calls == 1 {
			return nil, &os.PathError{Op: "openat", Path: name, Err: os.ErrNotExist}
		}
		return originalOpen(root, name)
	}
	file, err := location.openFile()
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("lock open calls = %d, want 2", calls)
	}
	if err := errors.Join(file.Close(), location.root.Close()); err != nil {
		t.Fatal(err)
	}
}

func TestAcquireWaitBlocksUntilRelease(t *testing.T) {
	scope := t.TempDir()
	first, err := Acquire(scope)
	if err != nil {
		t.Fatal(err)
	}
	acquired := make(chan *Lock, 1)
	errs := make(chan error, 1)
	go func() {
		lock, err := AcquireWait(scope)
		if err != nil {
			errs <- err
			return
		}
		acquired <- lock
	}()
	select {
	case <-acquired:
		t.Fatal("AcquireWait returned while the first lock was held")
	case err := <-errs:
		t.Fatalf("AcquireWait failed instead of waiting: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	select {
	case lock := <-acquired:
		if err := lock.Release(); err != nil {
			t.Fatal(err)
		}
	case err := <-errs:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("AcquireWait did not acquire after release")
	}
}

func TestAcquireWaitContextFailsClosedOnStuckHolder(t *testing.T) {
	scope := t.TempDir()
	first, err := Acquire(scope)
	if err != nil {
		t.Fatal(err)
	}
	released := false
	t.Cleanup(func() {
		if !released {
			_ = first.Release()
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	started := time.Now()
	lock, err := AcquireWaitContext(ctx, scope)
	if lock != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("stuck-holder wait = lock %v, err %v; want deadline failure", lock, err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("stuck-holder wait exceeded deterministic bound: %v", elapsed)
	}
	if err := first.Release(); err != nil {
		t.Fatalf("release stuck holder: %v", err)
	}
	released = true
	reacquired, err := Acquire(scope)
	if err != nil {
		t.Fatalf("deadline left a hidden waiter or retained resource: %v", err)
	}
	if err := reacquired.Release(); err != nil {
		t.Fatalf("release reacquired lock: %v", err)
	}
}

func TestAcquireWaitContextCancellationClosesEveryResource(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var opened *Lock
	lock, err := acquireWithContext(ctx, t.TempDir(), false, acquireHooks{tryLock: func(candidate *Lock) (bool, error) {
		opened = candidate
		cancel()
		return false, nil
	}})
	if lock != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled injected wait = lock %v, err %v; want cancellation", lock, err)
	}
	if opened == nil {
		t.Fatal("injected lock attempt did not run")
	}
	if _, statErr := opened.file.Stat(); !errors.Is(statErr, os.ErrClosed) {
		t.Fatalf("lock file remained open after cancellation: %v", statErr)
	}
	if _, statErr := opened.root.Lstat("."); !errors.Is(statErr, os.ErrClosed) {
		t.Fatalf("lock root remained open after cancellation: %v", statErr)
	}
}

func TestAcquireRemainsSingleAttemptAndClosesEveryResourceOnContention(t *testing.T) {
	var opened *Lock
	attempts := 0
	lock, err := acquireWithContext(context.Background(), t.TempDir(), true, acquireHooks{tryLock: func(candidate *Lock) (bool, error) {
		opened = candidate
		attempts++
		return false, nil
	}})
	if lock != nil || !errors.Is(err, errLockHeld) {
		t.Fatalf("injected nonblocking acquisition = lock %v, err %v", lock, err)
	}
	if attempts != 1 {
		t.Fatalf("nonblocking lock attempts = %d, want exactly 1", attempts)
	}
	if opened == nil {
		t.Fatal("injected lock attempt did not run")
	}
	if _, statErr := opened.file.Stat(); !errors.Is(statErr, os.ErrClosed) {
		t.Fatalf("contended lock file remained open: %v", statErr)
	}
	if _, statErr := opened.root.Lstat("."); !errors.Is(statErr, os.ErrClosed) {
		t.Fatalf("contended lock root remained open: %v", statErr)
	}
}

func TestAcquireWaitContextRejectsNilAndPreCanceledContextBeforeOpen(t *testing.T) {
	originalOpen := filelockOpenFile
	t.Cleanup(func() { filelockOpenFile = originalOpen })
	filelockOpenFile = func(*os.Root, string) (*os.File, error) {
		t.Fatal("canceled acquisition opened a lock file")
		return nil, errors.New("unreachable")
	}

	var nilContext context.Context
	if lock, err := AcquireWaitContext(nilContext, t.TempDir()); lock != nil || err == nil || !strings.Contains(err.Error(), "nil context") {
		t.Fatalf("nil-context wait = lock %v, err %v", lock, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if lock, err := AcquireWaitContext(ctx, t.TempDir()); lock != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled wait = lock %v, err %v", lock, err)
	}
}

func TestScopeIdentityPlatformAliases(t *testing.T) {
	tests := []struct {
		name, goos, left, right string
		equal                   bool
	}{
		{"darwin case", "darwin", "/Users/Alice/Work", "/users/alice/work", true},
		{"linux case distinct", "linux", "/srv/Work", "/srv/work", false},
		{"windows drive and slash", "windows", `C:\Users\Alice\Work`, `c:/users/alice/work/`, true},
		{"windows extended drive", "windows", `\\?\C:\Users\Alice\Work`, `c:/users/alice/work`, true},
		{"windows UNC", "windows", `\\Server\Share\Work`, `\\?\UNC\server\share\work\`, true},
		{"windows clean", "windows", `D:\work\.\child\..\scope`, `d:/work/scope`, true},
		{"windows volumes distinct", "windows", `C:\work`, `D:\work`, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			left := normalizeScopeIdentity(tc.left, tc.goos)
			right := normalizeScopeIdentity(tc.right, tc.goos)
			if got := left == right; got != tc.equal {
				t.Fatalf("identities equal = %v, want %v:\nleft  %q\nright %q", got, tc.equal, left, right)
			}
		})
	}
}

func TestLockPathUsesDarwinCaseFoldedIdentity(t *testing.T) {
	base := t.TempDir()
	left, err := scopeIdentity(filepath.Join(base, "Scope"), "darwin")
	if err != nil {
		t.Fatal(err)
	}
	right, err := scopeIdentity(filepath.Join(base, "scope"), "darwin")
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("Darwin case aliases produced distinct identities: %q != %q", left, right)
	}
}

func TestScopeIdentityResolvesMissingLeafThroughSymlinkedParent(t *testing.T) {
	base := t.TempDir()
	realParent := filepath.Join(base, "real-parent")
	if err := os.Mkdir(realParent, 0o700); err != nil {
		t.Fatal(err)
	}
	aliasParent := filepath.Join(base, "alias-parent")
	if err := os.Symlink(realParent, aliasParent); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	realIdentity, err := ScopeIdentity(filepath.Join(realParent, "missing", "leaf"))
	if err != nil {
		t.Fatal(err)
	}
	aliasIdentity, err := ScopeIdentity(filepath.Join(aliasParent, "missing", "leaf"))
	if err != nil {
		t.Fatal(err)
	}
	if realIdentity != aliasIdentity {
		t.Fatalf("missing leaves below one physical parent split lock identity:\nreal  %q\nalias %q", realIdentity, aliasIdentity)
	}
}

func TestAcquireMissingLeafParentAliasesShareOneLock(t *testing.T) {
	cacheHome := t.TempDir()
	t.Setenv("HOME", cacheHome)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(cacheHome, "cache"))
	t.Setenv("LOCALAPPDATA", filepath.Join(cacheHome, "local-cache"))
	base := t.TempDir()
	realParent := filepath.Join(base, "real-parent")
	if err := os.Mkdir(realParent, 0o700); err != nil {
		t.Fatal(err)
	}
	aliasParent := filepath.Join(base, "alias-parent")
	if err := os.Symlink(realParent, aliasParent); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	first, err := Acquire(filepath.Join(aliasParent, "missing", "leaf"))
	if err != nil {
		t.Fatal(err)
	}
	if second, err := Acquire(filepath.Join(realParent, "missing", "leaf")); err == nil {
		_ = second.Release()
		t.Fatal("physical parent aliases acquired distinct locks for one missing leaf")
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestResolveScopePathPropagatesPermissionFailure(t *testing.T) {
	permission := &fs.PathError{Op: "lstat", Path: "restricted", Err: fs.ErrPermission}
	_, err := resolveScopePath(filepath.Join(t.TempDir(), "restricted", "leaf"), func(string) (string, error) {
		return "", permission
	}, os.Lstat)
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("permission failure was treated as a missing leaf: %v", err)
	}
}

func TestResolveScopePathRetriesMissingToExistingTransition(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scope")
	if err := os.WriteFile(path, []byte("scope"), 0o600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	resolved, err := resolveScopePath(path, func(value string) (string, error) {
		calls++
		if calls == 1 {
			return "", &os.PathError{Op: "lstat", Path: value, Err: os.ErrNotExist}
		}
		return filepath.EvalSymlinks(value)
	}, os.Lstat)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || resolved != want {
		t.Fatalf("transition resolve = %q in %d calls, want %q in 2", resolved, calls, want)
	}
}

func TestScopeIdentityPropagatesSymlinkLoop(t *testing.T) {
	base := t.TempDir()
	left := filepath.Join(base, "left")
	right := filepath.Join(base, "right")
	if err := os.Symlink("right", left); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink("left", right); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := ScopeIdentity(filepath.Join(left, "leaf")); err == nil || !strings.Contains(err.Error(), "resolve filesystem scope") {
		t.Fatalf("symlink loop was treated as a missing scope leaf: %v", err)
	}
}

func TestAcquireRejectsLockDirectoryReplacementWithoutSplitBrain(t *testing.T) {
	cacheHome := t.TempDir()
	t.Setenv("HOME", cacheHome)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(cacheHome, "cache"))
	t.Setenv("LOCALAPPDATA", filepath.Join(cacheHome, "local-cache"))
	scope := filepath.Join(t.TempDir(), "scope")
	var competing *Lock
	var replacementErr error

	first, err := acquireWithHooks(scope, true, acquireHooks{afterLockOpen: func(lockDir string) error {
		moved := lockDir + "-moved"
		if replacementErr = os.Rename(lockDir, moved); replacementErr != nil {
			return replacementErr
		}
		if replacementErr = os.Mkdir(lockDir, 0o700); replacementErr != nil {
			return replacementErr
		}
		competing, replacementErr = Acquire(scope)
		return replacementErr
	}})
	if replacementErr != nil {
		t.Skipf("platform prevents replacement of an open lock directory: %v", replacementErr)
	}
	if first != nil || err == nil || !strings.Contains(err.Error(), "changed identity during acquisition") {
		t.Fatalf("acquisition through replaced lock directory succeeded: lock=%v err=%v", first, err)
	}
	if competing == nil {
		t.Fatal("competing acquisition in replacement directory did not run")
	}
	if unexpected, err := Acquire(scope); err == nil {
		_ = unexpected.Release()
		t.Fatal("replacement-directory lock was not exclusive")
	}
	if err := competing.Release(); err != nil {
		t.Fatal(err)
	}
	reacquired, err := Acquire(scope)
	if err != nil {
		t.Fatalf("replacement-directory lock was not reacquirable: %v", err)
	}
	if err := reacquired.Release(); err != nil {
		t.Fatal(err)
	}
}
