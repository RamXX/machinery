package designlock

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMaterializeRegularFileBoundsContinuousAppender(t *testing.T) {
	design := t.TempDir()
	externalDir := t.TempDir()
	input := filepath.Join(externalDir, "input.bin")
	if err := os.WriteFile(input, bytes.Repeat([]byte("a"), 2*fingerprintBufferBytes), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := AcquireReader(design)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Release() }()

	stop, stopped := make(chan struct{}), make(chan struct{})
	appended := make(chan struct{})
	launched := make(chan struct{})
	var once sync.Once
	prior := testAfterSnapshotCopyReadChunk
	testAfterSnapshotCopyReadChunk = func(path string) {
		if path != input {
			return
		}
		once.Do(func() {
			close(launched)
			go func() {
				defer close(stopped)
				file, openErr := os.OpenFile(input, os.O_APPEND|os.O_WRONLY, 0)
				if openErr != nil {
					close(appended)
					return
				}
				defer file.Close()
				first := true
				for {
					if _, writeErr := file.Write([]byte("growth")); writeErr != nil {
						if first {
							close(appended)
						}
						return
					}
					if first {
						close(appended)
						first = false
					}
					select {
					case <-stop:
						return
					default:
					}
				}
			}()
			<-appended
		})
	}
	started := time.Now()
	snapshot, err := lock.MaterializeRegularFile(input)
	testAfterSnapshotCopyReadChunk = prior
	t.Cleanup(func() { testAfterSnapshotCopyReadChunk = prior })
	select {
	case <-launched:
		close(stop)
		<-stopped
	default:
	}
	if snapshot != nil {
		_ = snapshot.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "witnessed size") {
		t.Fatalf("continuously growing input was accepted: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("bounded materialization took %s", elapsed)
	}
}

func TestExpectedOutputValidationBoundsContinuousAppender(t *testing.T) {
	design := t.TempDir()
	output := filepath.Join(design, "generated.bin")
	body := bytes.Repeat([]byte("b"), 2*fingerprintBufferBytes)
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Release() }()

	stop, stopped := make(chan struct{}), make(chan struct{})
	appended := make(chan struct{})
	launched := make(chan struct{})
	var once sync.Once
	prior := testAfterFingerprintReadChunk
	err = lock.PublishExpected("continuous-output", "rerun continuous-output writer", []OutputExpectation{ExpectFile(output, body, 0o644)}, func() error {
		if err := os.WriteFile(output, body, 0o644); err != nil {
			return err
		}
		testAfterFingerprintReadChunk = func(path string) {
			if path != filepath.Base(output) {
				return
			}
			once.Do(func() {
				close(launched)
				go func() {
					defer close(stopped)
					file, openErr := os.OpenFile(output, os.O_APPEND|os.O_WRONLY, 0)
					if openErr != nil {
						close(appended)
						return
					}
					defer file.Close()
					first := true
					for {
						if _, writeErr := file.Write([]byte("growth")); writeErr != nil {
							if first {
								close(appended)
							}
							return
						}
						if first {
							close(appended)
							first = false
						}
						select {
						case <-stop:
							return
						default:
						}
					}
				}()
				<-appended
			})
		}
		return nil
	})
	testAfterFingerprintReadChunk = prior
	t.Cleanup(func() { testAfterFingerprintReadChunk = prior })
	select {
	case <-launched:
		close(stop)
		<-stopped
	default:
	}
	if err == nil || !strings.Contains(err.Error(), "witnessed size") {
		t.Fatalf("continuously growing output was accepted: %v", err)
	}
}

func TestSnapshotInventoryRejectsEntryLimitBeforeAccumulation(t *testing.T) {
	dir := t.TempDir()
	for index := 0; index < 9; index++ {
		name := filepath.Join(dir, string(rune('a'+index)))
		if err := os.WriteFile(name, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	handle, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	budget := snapshotBudget{maxEntries: 8, maxBytes: snapshotAggregateMaxBytes}
	entries, err := readSnapshotDir(handle, "test inventory", &budget)
	closeErr := handle.Close()
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	if err == nil || !strings.Contains(err.Error(), "8-entry snapshot inventory limit") {
		t.Fatalf("over-limit inventory was accepted: entries=%d err=%v", len(entries), err)
	}
	if entries != nil {
		t.Fatalf("over-limit page was accumulated: %d entries", len(entries))
	}
}

func TestChildPackInventoryRejectsFixedEntryLimit(t *testing.T) {
	pack := t.TempDir()
	for index := 0; index <= packCapabilityMaxEntries; index++ {
		name := filepath.Join(pack, fmt.Sprintf("entry-%03d", index))
		if err := os.WriteFile(name, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files, err := readStrictPackFiles(pack)
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("%d-entry snapshot inventory limit", packCapabilityMaxEntries)) {
		t.Fatalf("over-limit child pack was accepted: files=%d err=%v", len(files), err)
	}
}

func TestCopyExternalTreePreservesDestinationCollision(t *testing.T) {
	design := t.TempDir()
	source := t.TempDir()
	dest := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "result.txt"), []byte("source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	collision := filepath.Join(dest, "result.txt")
	if err := os.WriteFile(collision, []byte("destination\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := AcquireReader(design)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Release() }()
	if _, err := lock.copyExternalTree(source, dest, nil, nil); err == nil {
		t.Fatal("copy overwrote an existing destination member")
	}
	got, err := os.ReadFile(collision)
	if err != nil || string(got) != "destination\n" {
		t.Fatalf("destination collision changed: %q, %v", got, err)
	}
}
