//go:build windows

package checker

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// This test intentionally runs only as a native Windows executable. It covers
// both the ordinary durable commit and recovery after a simulated process
// death, including the Windows directory FlushFileBuffers implementation.
func TestProjectionTransactionNativeWindowsCommitAndRecovery(t *testing.T) {
	design := t.TempDir()
	target := filepath.Join(design, "checkers", "windows", "projection.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := commitProjectionPlansWithHooks(design, []plannedProjection{{dest: target, rendered: []byte("committed")}}, projectionTransactionHooks{}); err != nil {
		t.Fatal(err)
	}
	assertProjectionContent(t, target, "committed")

	err := commitProjectionPlansWithHooks(design, []plannedProjection{{dest: target, rendered: []byte("uncommitted")}}, projectionTransactionHooks{
		fault: func(point string) error {
			if point == "installed:0" {
				return errSimulatedProjectionCrash
			}
			return nil
		},
	})
	if !errors.Is(err, errSimulatedProjectionCrash) {
		t.Fatalf("simulated crash returned %v", err)
	}
	if err := recoverProjectionTransaction(design); err != nil {
		t.Fatal(err)
	}
	assertProjectionContent(t, target, "committed")
}
