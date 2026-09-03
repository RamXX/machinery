// Package artifactset commits generated files as a confined, portable,
// crash-recoverable set. It is the common write boundary for generators that
// must never leave a mixture of old and new artifacts.
package artifactset

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/designlock"
)

// artifactRemovalCandidateMaxBytes is the single ownership-classification
// bound for stale generated artifacts. Candidate names are discovered from
// ambient output inventories, so their contents are untrusted until this
// bounded read proves ownership.
const artifactRemovalCandidateMaxBytes int64 = 64 << 20

func ensureRealDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve artifact output directory: %w", err)
	}
	info, statErr := os.Lstat(abs)
	if os.IsNotExist(statErr) {
		parent, err := ensureRealDir(filepath.Dir(abs))
		if err != nil {
			return "", err
		}
		abs = filepath.Join(parent, filepath.Base(abs))
		if err := os.Mkdir(abs, 0o755); err != nil && !os.IsExist(err) {
			return "", fmt.Errorf("create artifact output directory: %w", err)
		}
		info, statErr = os.Lstat(abs)
	}
	if statErr != nil {
		return "", fmt.Errorf("inspect artifact output directory: %w", statErr)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("artifact output %s must be a real directory", abs)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve artifact output directory symlinks: %w", err)
	}
	return resolved, nil
}

// RemoveGenerated removes an already-validated sorted set of generated
// basenames from a rooted output directory and durably syncs the directory.
// Callers hold the design publication sentinel, so a crash between removals
// remains unreadable and an exact rerun converges the inventory.
func RemoveGenerated(dir string, names []string) (retErr error) {
	resolved, err := ensureRealDir(dir)
	if err != nil {
		return err
	}
	ordered := append([]string(nil), names...)
	sort.Strings(ordered)
	for _, name := range ordered {
		if filepath.Base(name) != name || name == "" || name == "." {
			return fmt.Errorf("unsafe generated removal target %q", name)
		}
	}
	return txReconcile(resolved, nil, ordered, txDefaultOps(nil))
}

// Commit replaces every named artifact as one durable transaction. Recovery
// runs after lock acquisition before a new transaction may begin.
func Commit(dir string, files map[string][]byte) error {
	return txCommit(dir, files, txDefaultOps(nil))
}

// Reconcile atomically installs files and removes the named obsolete owned
// artifacts in one durable transaction. Missing removal targets are harmless,
// which makes an exact rerun byte-idempotent after successful convergence.
func Reconcile(dir string, files map[string][]byte, remove []string) error {
	return txReconcile(dir, files, remove, txDefaultOps(nil))
}

// RemovalPrecondition binds a conditionally-owned stale artifact to the exact
// regular file inspected by InspectRemovalCandidate. Its unexported identity
// fields prevent callers from manufacturing a name-only deletion plan.
type RemovalPrecondition struct {
	Name   string
	digest string
	mode   fs.FileMode
	info   os.FileInfo
}

// InspectRemovalCandidate reads one possible stale artifact through a rooted,
// no-follow handle and returns both its bytes for ownership classification and
// the exact identity ReconcilePlanned must still observe before deleting it.
func InspectRemovalCandidate(dir, name string) (body []byte, condition RemovalPrecondition, retErr error) {
	return inspectRemovalCandidate(dir, name, artifactRemovalCandidateMaxBytes, nil)
}

func inspectRemovalCandidate(dir, name string, maxBytes int64, afterOpen func() error) (body []byte, condition RemovalPrecondition, retErr error) {
	if maxBytes <= 0 {
		return nil, condition, fmt.Errorf("artifact removal candidate byte limit must be positive")
	}
	if err := txValidateTarget(name); err != nil {
		return nil, condition, err
	}
	resolved, err := ensureRealDir(dir)
	if err != nil {
		return nil, condition, err
	}
	root, syncFile, err := txOpenRoot(resolved)
	if err != nil {
		return nil, condition, err
	}
	defer func() { retErr = errors.Join(retErr, syncFile.Close(), root.Close()) }()
	info, err := root.Lstat(name)
	if err != nil {
		return nil, condition, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, condition, fmt.Errorf("artifact removal candidate %s must be a regular file", name)
	}
	if info.Size() < 0 || info.Size() > maxBytes {
		return nil, condition, fmt.Errorf("artifact removal candidate %s exceeds %d-byte limit", name, maxBytes)
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, condition, err
	}
	opened, statErr := f.Stat()
	if statErr != nil || !sameRemovalCandidateSnapshot(info, opened) {
		_ = f.Close()
		return nil, condition, fmt.Errorf("artifact removal candidate %s changed identity while opening", name)
	}
	if afterOpen != nil {
		if hookErr := afterOpen(); hookErr != nil {
			return nil, condition, errors.Join(hookErr, f.Close())
		}
	}
	body, readErr := io.ReadAll(io.LimitReader(f, maxBytes+1))
	afterRead, statErr := f.Stat()
	closeErr := f.Close()
	if err := errors.Join(readErr, statErr, closeErr); err != nil {
		return nil, condition, err
	}
	afterPath, pathErr := root.Lstat(name)
	if int64(len(body)) > maxBytes {
		return nil, condition, fmt.Errorf("artifact removal candidate %s exceeds %d-byte limit", name, maxBytes)
	}
	if pathErr != nil || !sameRemovalCandidateSnapshot(info, afterRead) || !sameRemovalCandidateSnapshot(info, afterPath) || int64(len(body)) != info.Size() {
		return nil, condition, fmt.Errorf("artifact removal candidate %s changed identity while reading", name)
	}
	sum := sha256.Sum256(body)
	return body, RemovalPrecondition{Name: name, digest: fmt.Sprintf("sha256:%x", sum), mode: info.Mode(), info: info}, nil
}

func sameRemovalCandidateSnapshot(before, after os.FileInfo) bool {
	return before != nil && after != nil &&
		os.SameFile(before, after) &&
		before.Mode() == after.Mode() &&
		before.Size() == after.Size() &&
		before.ModTime().Equal(after.ModTime()) &&
		txChangeID(before) == txChangeID(after)
}

// ReconcilePlanned atomically installs files and deletes only candidates that
// still match the exact identity observed by InspectRemovalCandidate.
func ReconcilePlanned(dir string, files map[string][]byte, remove []RemovalPrecondition) error {
	return txReconcilePlanned(dir, files, remove, txDefaultOps(nil))
}

// CommitRooted commits through a retained output-directory capability.
func CommitRooted(scope string, root *os.Root, files map[string][]byte) error {
	return txReconcilePlannedRooted(scope, root, files, nil, txDefaultOps(nil))
}

// ReconcilePlannedRooted is ReconcilePlanned through a retained output root;
// it never resolves or reopens the output directory by ambient pathname.
func ReconcilePlannedRooted(scope string, root *os.Root, files map[string][]byte, remove []RemovalPrecondition) error {
	return txReconcilePlannedRooted(scope, root, files, remove, txDefaultOps(nil))
}

// ReconcileGuardedRooted additionally binds every existing install target to
// an identity returned by InspectRemovalCandidate. Generators use this after
// proving the current bytes belong to the same writer/source, preventing a
// foreign replacement between ownership inspection and transaction parking.
func ReconcileGuardedRooted(scope string, root *os.Root, files map[string][]byte, remove, replace []RemovalPrecondition) error {
	return txReconcileGuardedRooted(scope, root, files, remove, replace, txDefaultOps(nil))
}

// ReconcileRooted reconciles an unconditional owned removal set through a
// retained output-directory capability.
func ReconcileRooted(scope string, root *os.Root, files map[string][]byte, remove []string) error {
	conditions := make([]RemovalPrecondition, len(remove))
	for i, name := range remove {
		conditions[i] = RemovalPrecondition{Name: name}
	}
	syncFile, err := txOpenSyncRoot(root)
	if err != nil {
		return err
	}
	return txReconcilePlannedOpened(scope, root, syncFile, false, files, conditions, nil, false, txDefaultOps(nil))
}

// CommitScoped commits files under designRoot while holding the canonical
// design snapshot lock. Outputs outside designRoot (for example formal's
// private generation collector) are ordinary temporary artifact sets and do
// not participate in the committed design snapshot.
func CommitScoped(designRoot, dir string, files map[string][]byte) error {
	inside, err := withinDesignRoot(designRoot, dir)
	if err != nil {
		return err
	}
	if !inside {
		return Commit(dir, files)
	}
	lock, err := designlock.Acquire(designRoot)
	if err != nil {
		return err
	}
	if err := lock.ResumeExpected("artifactset", "rerun the exact Machinery writer command that was interrupted"); err != nil {
		return errors.Join(err, lock.Release())
	}
	expected := make([]designlock.OutputExpectation, 0, len(files))
	for name, body := range files {
		expected = append(expected, designlock.ExpectFile(filepath.Join(dir, name), body, 0o644))
	}
	commitErr := lock.PublishExpectedRooted("artifactset", "rerun the exact Machinery writer command that was interrupted", expected, func(outputs *designlock.OutputScope) error {
		return outputs.WithRoot(dir, func(root *os.Root) error {
			return CommitRooted(dir, root, files)
		})
	})
	return errors.Join(commitErr, lock.Release())
}

func withinDesignRoot(designRoot, dir string) (bool, error) {
	rootAbs, err := filepath.Abs(designRoot)
	if err != nil {
		return false, fmt.Errorf("resolve design root: %w", err)
	}
	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		return false, fmt.Errorf("resolve artifact output: %w", err)
	}
	rel, err := filepath.Rel(rootAbs, dirAbs)
	if err != nil {
		return false, fmt.Errorf("relate artifact output to design root: %w", err)
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))), nil
}

func commitWithRename(dir string, files map[string][]byte, rename func(string, string) error) error {
	return txCommit(dir, files, txDefaultOps(rename))
}
