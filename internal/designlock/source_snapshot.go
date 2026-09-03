package designlock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var designSourceBeforeOpen = func(string) {}
var designSourceAfterRead = func(string) {}

func (l *Lock) materializeDesignSource() error {
	current, err := os.Lstat(l.root)
	if err != nil || !os.SameFile(l.rootInfo, current) {
		return fmt.Errorf("design root changed identity before immutable source materialization")
	}
	cleanup, err := newPrivateSnapshot("machinery-design-source-")
	if err != nil {
		return fmt.Errorf("create immutable design source: %w", err)
	}
	temp := cleanup.Path()
	values, err := l.copyExternalTree(l.root, temp, designSourceBeforeOpen, designSourceAfterRead)
	if err != nil {
		return errors.Join(fmt.Errorf("materialize immutable design source; refusing a potentially ABA-derived generation: %w", err), cleanup.Close())
	}
	if got, want := fingerprintDigest(values), fingerprintDigest(l.snapshot); got != want {
		return errors.Join(fmt.Errorf("design changed while materializing immutable source; refusing a potentially ABA-derived generation"), cleanup.Close())
	}
	l.sourceRoot = temp
	l.sourceCleanup = cleanup
	l.sourceAliases = append(l.sourceAliases, temp)
	return nil
}

// SourceRoot returns the immutable design tree an operation must use for all
// discovery, reads, and rendering. Publication targets remain under the real
// design path passed to Publish.
func (l *Lock) SourceRoot() string {
	if l == nil {
		return ""
	}
	if l.sourceRoot == "" {
		return l.root
	}
	return l.sourceRoot
}

// SourcePath maps a path under the held design root into the immutable source
// tree. External paths are returned unchanged and require an explicit stable
// capability such as MaterializeRegularFile/MaterializeExternalTree.
func (l *Lock) SourcePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(l.root, abs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path, nil
	}
	return filepath.Join(l.SourceRoot(), rel), nil
}

// LogicalText removes the private randomized source-root name from diagnostics
// while preserving the held design's stable canonical path.
func (l *Lock) LogicalText(value string) string {
	if l == nil {
		return value
	}
	for _, source := range l.sourceAliases {
		value = strings.ReplaceAll(value, source, l.root)
	}
	for _, alias := range l.inputAliases {
		value = strings.ReplaceAll(value, alias.from, alias.to)
	}
	return value
}

// LogicalError preserves error unwrapping while making diagnostics independent
// of the private materialization directory chosen for this process.
func (l *Lock) LogicalError(err error) error {
	if err == nil {
		return nil
	}
	return logicalError{lock: l, err: err}
}

type logicalError struct {
	lock *Lock
	err  error
}

func (e logicalError) Error() string { return e.lock.LogicalText(e.err.Error()) }
func (e logicalError) Unwrap() error { return e.err }

func (l *Lock) insideSource(abs string) bool {
	if l == nil || l.sourceRoot == "" {
		return false
	}
	rel, err := filepath.Rel(l.sourceRoot, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}
