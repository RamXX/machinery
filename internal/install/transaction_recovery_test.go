package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRootedDirectorySyncPropagatesCloseFailure(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	want := errors.New("injected close failure")
	previous := closeInstallFile
	closeInstallFile = func(f *os.File) error { return errors.Join(f.Close(), want) }
	t.Cleanup(func() { closeInstallFile = previous })
	if err := syncRootRelativeDir(root, "."); !errors.Is(err, want) {
		t.Fatalf("rooted directory sync error = %v, want injected close failure", err)
	}
}

func TestScratchCleanupAndCloseFailuresPropagateThroughCallers(t *testing.T) {
	t.Run("directory removal", func(t *testing.T) {
		want := errors.New("injected scratch directory cleanup failure")
		previous := removeInstallScratchDir
		removeInstallScratchDir = func(path string, durable bool) error {
			return errors.Join(previous(path, durable), want)
		}
		t.Cleanup(func() { removeInstallScratchDir = previous })
		source := fakeSource(t)
		err := stageAndCommit(filepath.Join(t.TempDir(), "home"), func(stage string) error {
			return buildRealStage(stage, source)
		})
		if !errors.Is(err, want) {
			t.Fatalf("stage cleanup error = %v, want injected directory cleanup failure", err)
		}
	})

	t.Run("file removal", func(t *testing.T) {
		want := errors.New("injected scratch file cleanup failure")
		previous := removeInstallScratchFile
		removeInstallScratchFile = func(path string, durable bool) error {
			return errors.Join(previous(path, durable), want)
		}
		t.Cleanup(func() { removeInstallScratchFile = previous })
		err := writeRendered(filepath.Join(t.TempDir(), "rendered.md"), "rendered\n")
		if !errors.Is(err, want) {
			t.Fatalf("render cleanup error = %v, want injected file cleanup failure", err)
		}
	})

	t.Run("file close", func(t *testing.T) {
		want := errors.New("injected scratch file close failure")
		previous := closeInstallFile
		closeInstallFile = func(file *os.File) error {
			return errors.Join(file.Close(), want)
		}
		t.Cleanup(func() { closeInstallFile = previous })
		err := writeRendered(filepath.Join(t.TempDir(), "rendered.md"), "rendered\n")
		if !errors.Is(err, want) {
			t.Fatalf("render close error = %v, want injected file close failure", err)
		}
	})
}

func TestInstallTransactionRecoveryAtPreparedAndMidMutation(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	write(t, first, "old-first")

	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := beginArtifactTransaction([]string{first, second}); err != nil {
		t.Fatal(err)
	}
	transactionReplaceForTest(t, first, "new-first")
	transactionReplaceForTest(t, second, "new-second")
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}

	recovered, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.Release()
	if got, err := os.ReadFile(first); err != nil || string(got) != "old-first" {
		t.Fatalf("first after recovery = %q, %v", got, err)
	}
	if _, err := os.Lstat(second); !os.IsNotExist(err) {
		t.Fatalf("new second target survived rollback: %v", err)
	}
	assertNoInstallJournal(t)
}

func TestPreparedRecoveryPreservesConcurrentPostImageChanges(t *testing.T) {
	tests := []struct {
		name        string
		initial     func(*testing.T, string)
		transaction func(*testing.T, string)
		concurrent  func(*testing.T, string)
		assertLive  func(*testing.T, string)
	}{
		{
			name:    "existing file overwritten after transaction write",
			initial: func(t *testing.T, target string) { write(t, target, "old") },
			transaction: func(t *testing.T, target string) {
				transactionReplaceForTest(t, target, "transaction")
			},
			concurrent: func(t *testing.T, target string) { write(t, target, "concurrent") },
			assertLive: func(t *testing.T, target string) {
				if got, err := os.ReadFile(target); err != nil || string(got) != "concurrent" {
					t.Fatalf("concurrent file was not preserved: %q, %v", got, err)
				}
			},
		},
		{
			name:    "new file overwritten after transaction write",
			initial: func(*testing.T, string) {},
			transaction: func(t *testing.T, target string) {
				transactionReplaceForTest(t, target, "transaction")
			},
			concurrent: func(t *testing.T, target string) { write(t, target, "concurrent-new") },
			assertLive: func(t *testing.T, target string) {
				if got, err := os.ReadFile(target); err != nil || string(got) != "concurrent-new" {
					t.Fatalf("concurrent new file was not preserved: %q, %v", got, err)
				}
			},
		},
		{
			name:    "file metadata changed after transaction write",
			initial: func(t *testing.T, target string) { write(t, target, "old") },
			transaction: func(t *testing.T, target string) {
				transactionReplaceForTest(t, target, "transaction")
			},
			concurrent: func(t *testing.T, target string) {
				changed := time.Unix(1_900_000_000, 0)
				if err := os.Chtimes(target, changed, changed); err != nil {
					t.Fatal(err)
				}
			},
			assertLive: func(t *testing.T, target string) {
				info, err := os.Stat(target)
				if err != nil || !info.ModTime().Equal(time.Unix(1_900_000_000, 0)) {
					t.Fatalf("concurrent metadata edit was not preserved: %v, %v", info, err)
				}
			},
		},
		{
			name:    "recreated file after transaction removal",
			initial: func(t *testing.T, target string) { write(t, target, "old") },
			transaction: func(t *testing.T, target string) {
				if err := durableRemoveAll(target); err != nil {
					t.Fatal(err)
				}
			},
			concurrent: func(t *testing.T, target string) { write(t, target, "host-recreated") },
			assertLive: func(t *testing.T, target string) {
				if got, err := os.ReadFile(target); err != nil || string(got) != "host-recreated" {
					t.Fatalf("host recreation was not preserved: %q, %v", got, err)
				}
			},
		},
		{
			name:    "nested directory edit after transaction write",
			initial: func(t *testing.T, target string) { write(t, filepath.Join(target, "child"), "old") },
			transaction: func(t *testing.T, target string) {
				staged := filepath.Join(t.TempDir(), "tree")
				write(t, filepath.Join(staged, "child"), "transaction")
				if err := durableRemoveAll(target); err != nil {
					t.Fatal(err)
				}
				if err := copyEntryNoFollow(staged, target); err != nil {
					t.Fatal(err)
				}
			},
			concurrent: func(t *testing.T, target string) { write(t, filepath.Join(target, "child"), "concurrent-tree") },
			assertLive: func(t *testing.T, target string) {
				if got, err := os.ReadFile(filepath.Join(target, "child")); err != nil || string(got) != "concurrent-tree" {
					t.Fatalf("concurrent tree edit was not preserved: %q, %v", got, err)
				}
			},
		},
		{
			name: "symlink retargeted after transaction write",
			initial: func(t *testing.T, target string) {
				if runtime.GOOS == "windows" {
					t.Skip("symlink fixture requires unprivileged symlink creation")
				}
				write(t, filepath.Join(filepath.Dir(target), "old-link-target"), "old")
				if err := os.Symlink("old-link-target", target); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			},
			transaction: func(t *testing.T, target string) {
				staged := filepath.Join(t.TempDir(), "staged-link")
				if err := os.Symlink("transaction-link-target", staged); err != nil {
					t.Fatal(err)
				}
				if err := durableRemoveAll(target); err != nil {
					t.Fatal(err)
				}
				if err := copyEntryNoFollow(staged, target); err != nil {
					t.Fatal(err)
				}
			},
			concurrent: func(t *testing.T, target string) {
				if err := os.Remove(target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("concurrent-link-target", target); err != nil {
					t.Fatal(err)
				}
			},
			assertLive: func(t *testing.T, target string) {
				if got, err := os.Readlink(target); err != nil || got != "concurrent-link-target" {
					t.Fatalf("concurrent symlink target was not preserved: %q, %v", got, err)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
			target := filepath.Join(t.TempDir(), "target")
			tc.initial(t, target)
			lock, err := acquireInstallOperationLock()
			if err != nil {
				t.Fatal(err)
			}
			tx, err := beginArtifactTransaction([]string{target})
			if err != nil {
				t.Fatal(err)
			}
			tc.transaction(t, target)
			tc.concurrent(t, target)
			if err := tx.closeAnchors(); err != nil {
				t.Fatal(err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
			if _, err := acquireInstallOperationLock(); err == nil || !strings.Contains(err.Error(), "preserve concurrently changed") {
				t.Fatalf("concurrent post-image recovery error = %v", err)
			}
			tc.assertLive(t, target)
			journalRoot, _, err := installJournalPaths()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(journalRoot); err != nil {
				t.Fatalf("recoverable journal was removed: %v", err)
			}
		})
	}
}

func TestPreparedRecoveryPreservesCrashBoundaryWithoutMatchingPostImage(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, string) error
	}{
		{
			name: "declared replacement before namespace mutation",
			prepare: func(t *testing.T, target string) error {
				staged := filepath.Join(t.TempDir(), "staged")
				write(t, staged, "intended")
				return declareInstallPresentPostImage(target, staged)
			},
		},
		{
			name: "unknown partial mutation",
			prepare: func(t *testing.T, target string) error {
				if err := declareInstallPostImage(target, installPostUnknown, ""); err != nil {
					return err
				}
				write(t, target, "partial")
				return nil
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
			target := filepath.Join(t.TempDir(), "target")
			write(t, target, "old")
			lock, err := acquireInstallOperationLock()
			if err != nil {
				t.Fatal(err)
			}
			tx, err := beginArtifactTransaction([]string{target})
			if err != nil {
				t.Fatal(err)
			}
			if err := tc.prepare(t, target); err != nil {
				t.Fatal(err)
			}
			if err := tx.closeAnchors(); err != nil {
				t.Fatal(err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
			if _, err := acquireInstallOperationLock(); err == nil || !strings.Contains(err.Error(), "post-image") {
				t.Fatalf("unmatched crash-boundary recovery error = %v", err)
			}
			if got, err := os.ReadFile(target); err != nil || (string(got) != "old" && string(got) != "partial") {
				t.Fatalf("unmatched crash-boundary state was not preserved: %q, %v", got, err)
			}
		})
	}
}

func TestRollbackRevalidatesPostImageImmediatelyBeforeRemoval(t *testing.T) {
	for _, recovery := range []bool{false, true} {
		name := "in-process rollback"
		if recovery {
			name = "prepared recovery"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
			target := filepath.Join(t.TempDir(), "target")
			write(t, target, "old")
			lock, err := acquireInstallOperationLock()
			if err != nil {
				t.Fatal(err)
			}
			tx, err := beginArtifactTransaction([]string{target})
			if err != nil {
				t.Fatal(err)
			}
			transactionReplaceForTest(t, target, "transaction")
			oldHook := afterInstallPostImageValidation
			t.Cleanup(func() { afterInstallPostImageValidation = oldHook })
			afterInstallPostImageValidation = func(path string) {
				if !sameInstallPath(path, target) {
					return
				}
				afterInstallPostImageValidation = nil
				write(t, target, "concurrent-after-validation")
			}
			if recovery {
				if err := tx.closeAnchors(); err != nil {
					t.Fatal(err)
				}
				if err := lock.Release(); err != nil {
					t.Fatal(err)
				}
				if _, err := acquireInstallOperationLock(); err == nil || !strings.Contains(err.Error(), "changed after post-image validation") {
					t.Fatalf("prepared recovery boundary error = %v", err)
				}
			} else {
				if err := tx.rollback(); err == nil || !strings.Contains(err.Error(), "changed after post-image validation") {
					t.Fatalf("rollback boundary error = %v", err)
				}
				if err := lock.Release(); err != nil {
					t.Fatal(err)
				}
			}
			if got, err := os.ReadFile(target); err != nil || string(got) != "concurrent-after-validation" {
				t.Fatalf("post-validation concurrent edit was not preserved: %q, %v", got, err)
			}
		})
	}
}

func TestCommitRejectsAndPreservesConcurrentPostImageChange(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	target := filepath.Join(t.TempDir(), "target")
	write(t, target, "old")
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	tx, err := beginArtifactTransaction([]string{target})
	if err != nil {
		t.Fatal(err)
	}
	transactionReplaceForTest(t, target, "transaction")
	write(t, target, "concurrent-before-commit")
	if err := tx.commit(); err == nil || !strings.Contains(err.Error(), "changed before commit") || !strings.Contains(err.Error(), "preserve concurrently changed") {
		t.Fatalf("commit concurrent post-image error = %v", err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "concurrent-before-commit" {
		t.Fatalf("commit path did not preserve concurrent edit: %q, %v", got, err)
	}
	journalRoot, _, err := installJournalPaths()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(journalRoot); err != nil {
		t.Fatalf("commit failure removed recoverable journal: %v", err)
	}
}

func transactionReplaceForTest(t *testing.T, target, content string) {
	t.Helper()
	staged := filepath.Join(t.TempDir(), "transaction-post-image")
	write(t, staged, content)
	if err := durableRemoveAll(target); err != nil {
		t.Fatal(err)
	}
	if err := copyEntryNoFollow(staged, target); err != nil {
		t.Fatal(err)
	}
}

func TestInstallTransactionRecoveryFinalizesCommittedStateAndScratch(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	target := filepath.Join(t.TempDir(), "target")
	write(t, target, "old")
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := beginArtifactTransaction([]string{target})
	if err != nil {
		t.Fatal(err)
	}
	scratch, _, err := installScratchFile(t.TempDir(), "seeded")
	if err != nil {
		t.Fatal(err)
	}
	scratchPath := scratch.Name()
	if _, err := scratch.WriteString("journal-owned scratch"); err != nil {
		t.Fatal(err)
	}
	if err := scratch.Close(); err != nil {
		t.Fatal(err)
	}
	write(t, target, "committed")
	if err := createJournalMarker(tx.root, installJournalCommitted); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}

	recovered, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.Release()
	if got, err := os.ReadFile(target); err != nil || string(got) != "committed" {
		t.Fatalf("committed target after recovery = %q, %v", got, err)
	}
	if _, err := os.Lstat(scratchPath); !os.IsNotExist(err) {
		t.Fatalf("journal scratch survived committed cleanup: %v", err)
	}
	assertNoInstallJournal(t)
}

func TestInstallTransactionSnapshotPrefixIsSafeToDiscard(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	outside := t.TempDir()
	first := filepath.Join(outside, "first")
	second := filepath.Join(outside, "second")
	write(t, first, "first-live")
	write(t, second, "second-live")
	root, _, err := installJournalPaths()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	j := installJournal{Schema: installJournalSchema, Phase: "snapshotting", Items: []installJournalItem{
		seededInstallJournalItem(t, first, "snapshots/000000", true),
		seededInstallJournalItem(t, second, "snapshots/000001", true),
	}}
	if err := writeJournalMetadata(root, j); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "snapshots"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := copyEntryNoFollow(first, filepath.Join(root, "snapshots", "000000")); err != nil {
		t.Fatal(err)
	}

	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	for path, want := range map[string]string{first: "first-live", second: "second-live"} {
		if got, err := os.ReadFile(path); err != nil || string(got) != want {
			t.Fatalf("snapshotting recovery changed %s: %q, %v", path, got, err)
		}
	}
	assertNoInstallJournal(t)
}

func TestInstallTransactionRejectsEscapesAndSymlinkJournal(t *testing.T) {
	t.Run("symlink root", func(t *testing.T) {
		t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
		outside := t.TempDir()
		sentinel := filepath.Join(outside, "sentinel")
		write(t, sentinel, "outside")
		root, _, err := installJournalPaths()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(root), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, root); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, err := acquireInstallOperationLock(); err == nil || !strings.Contains(err.Error(), "reject") {
			t.Fatalf("symlink journal error = %v", err)
		}
		if got, err := os.ReadFile(sentinel); err != nil || string(got) != "outside" {
			t.Fatalf("outside sentinel changed: %q, %v", got, err)
		}
	})

	t.Run("backup traversal", func(t *testing.T) {
		t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
		outside := t.TempDir()
		target := filepath.Join(outside, "target")
		sentinel := filepath.Join(outside, "sentinel")
		write(t, target, "live")
		write(t, sentinel, "outside")
		root, _, err := installJournalPaths()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(root, 0o700); err != nil {
			t.Fatal(err)
		}
		j := installJournal{Schema: installJournalSchema, Phase: "snapshotting", Items: []installJournalItem{seededInstallJournalItem(t, target, "../../sentinel", true)}}
		if err := writeJournalMetadata(root, j); err != nil {
			t.Fatal(err)
		}
		if err := createJournalMarker(root, installJournalPrepared); err != nil {
			t.Fatal(err)
		}
		if _, err := acquireInstallOperationLock(); err == nil || !strings.Contains(err.Error(), "escapes") {
			t.Fatalf("traversal journal error = %v", err)
		}
		if got, err := os.ReadFile(sentinel); err != nil || string(got) != "outside" {
			t.Fatalf("outside sentinel changed: %q, %v", got, err)
		}
	})
}

func TestInstallTransactionRejectsLegacySchemaWithoutMutation(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	target := filepath.Join(t.TempDir(), "target")
	write(t, target, "unchanged")
	root, _, err := installJournalPaths()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := installJournal{Schema: 1, Phase: "snapshotting", Items: []installJournalItem{{Target: target, Backup: "snapshots/000000", Existed: true}}}
	if err := writeJournalMetadata(root, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := acquireInstallOperationLock(); err == nil || !strings.Contains(err.Error(), "schema 1") || !strings.Contains(err.Error(), "no artifact was mutated") {
		t.Fatalf("legacy journal error = %v", err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "unchanged" {
		t.Fatalf("legacy recovery mutated target: %q, %v", got, err)
	}
}

func TestInstallTransactionRejectsSchemaTwoWithoutPostImages(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	target := filepath.Join(t.TempDir(), "target")
	write(t, target, "unchanged")
	root, _, err := installJournalPaths()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := installJournal{Schema: 2, Phase: "snapshotting", Items: []installJournalItem{{Target: target, Backup: "snapshots/000000", Existed: true}}}
	if err := writeJournalMetadata(root, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := acquireInstallOperationLock(); err == nil || !strings.Contains(err.Error(), "schema 2") || !strings.Contains(err.Error(), "no artifact was mutated") {
		t.Fatalf("schema 2 journal error = %v", err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "unchanged" {
		t.Fatalf("schema 2 recovery mutated target: %q, %v", got, err)
	}
}

func TestInstallTransactionRejectsTamperedSnapshotsBeforeRollback(t *testing.T) {
	tests := []struct {
		name   string
		dir    bool
		tamper func(t *testing.T, backup string)
	}{
		{name: "bytes", tamper: func(t *testing.T, backup string) { write(t, backup, "tampered") }},
		{name: "mode", tamper: func(t *testing.T, backup string) {
			if runtime.GOOS == "windows" {
				t.Skip("Windows synthesizes POSIX mode bits")
			}
			if err := os.Chmod(backup, 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "tree", dir: true, tamper: func(t *testing.T, backup string) { write(t, filepath.Join(backup, "rogue"), "tampered") }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
			target := filepath.Join(t.TempDir(), "target")
			live := target
			if tc.dir {
				live = filepath.Join(target, "child")
			}
			write(t, live, "original")
			if err := os.Chmod(live, 0o644); err != nil {
				t.Fatal(err)
			}
			lock, err := acquireInstallOperationLock()
			if err != nil {
				t.Fatal(err)
			}
			tx, err := beginArtifactTransaction([]string{target})
			if err != nil {
				t.Fatal(err)
			}
			backup := filepath.Join(tx.root, filepath.FromSlash(tx.journal.Items[0].Backup))
			write(t, live, "changed")
			if err := tx.closeAnchors(); err != nil {
				t.Fatal(err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
			tc.tamper(t, backup)
			if _, err := acquireInstallOperationLock(); err == nil || !strings.Contains(err.Error(), "integrity") {
				t.Fatalf("tampered snapshot recovery error = %v", err)
			}
			if got, err := os.ReadFile(live); err != nil || string(got) != "changed" {
				t.Fatalf("recovery mutated live target before integrity failure: %q, %v", got, err)
			}
		})
	}
}

func seededInstallJournalItem(t *testing.T, target, backup string, existed bool) installJournalItem {
	t.Helper()
	anchor, anchorID, err := installArtifactAnchor(filepath.Dir(target))
	if err != nil {
		t.Fatal(err)
	}
	return installJournalItem{Target: target, Backup: backup, Existed: existed, Anchor: anchor, AnchorID: anchorID}
}

func TestInstallTransactionRejectsParentSwapBeforeMutation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fixture uses POSIX directory replacement; Windows runs the same mutation validator")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	base := t.TempDir()
	selected := filepath.Join(base, "selected")
	target := filepath.Join(selected, "artifact")
	write(t, target, "inside")
	outside := t.TempDir()
	outsideTarget := filepath.Join(outside, "artifact")
	write(t, outsideTarget, "outside")
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := beginArtifactTransaction([]string{target})
	if err != nil {
		t.Fatal(err)
	}
	parked := filepath.Join(base, "selected-parked")
	afterInstallMutationValidation = func(string) {
		afterInstallMutationValidation = nil
		if err := os.Rename(selected, parked); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, selected); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { afterInstallMutationValidation = nil })
	if err := durableRemoveAll(target); err != nil {
		t.Fatalf("root-relative mutation failed after parent swap: %v", err)
	}
	if got, err := os.ReadFile(outsideTarget); err != nil || string(got) != "outside" {
		t.Fatalf("outside target changed: %q, %v", got, err)
	}
	if err := os.Remove(selected); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(parked, selected); err != nil {
		t.Fatal(err)
	}
	if err := tx.rollback(); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestRenameReplaceUsesRetainedParentBeforeFastPathMutation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX directory replacement; Windows exercises the same os.Root path")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	base := t.TempDir()
	selected := filepath.Join(base, "selected")
	target := filepath.Join(selected, "artifact")
	write(t, target, "old")
	outside := t.TempDir()
	outsideTarget := filepath.Join(outside, "artifact")
	write(t, outsideTarget, "outside")
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := beginArtifactTransaction([]string{target})
	if err != nil {
		t.Fatal(err)
	}
	staged, cleanup, err := installScratchFile(base, "replacement")
	if err != nil {
		t.Fatal(err)
	}
	stagedPath := staged.Name()
	defer func() {
		if err := cleanup(); err != nil {
			t.Error(err)
		}
	}()
	if _, err := staged.WriteString("new"); err != nil {
		t.Fatal(err)
	}
	if err := staged.Close(); err != nil {
		t.Fatal(err)
	}
	parked := filepath.Join(base, "selected-parked")
	afterInstallMutationValidation = func(string) {
		afterInstallMutationValidation = nil
		if err := os.Rename(selected, parked); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, selected); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { afterInstallMutationValidation = nil })
	if err := renameReplace(stagedPath, target); err != nil {
		t.Fatalf("rooted replacement failed after parent swap: %v", err)
	}
	if got, err := os.ReadFile(outsideTarget); err != nil || string(got) != "outside" {
		t.Fatalf("outside target changed: %q, %v", got, err)
	}
	if err := os.Remove(selected); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(parked, selected); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "new" {
		t.Fatalf("rooted target = %q, %v", got, err)
	}
	if err := tx.rollback(); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestInstallTransactionCleansTombstoneAndPropagatesFinalizeFailure(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	root, tombstone, err := installJournalPaths()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tombstone, "stale"), 0o700); err != nil {
		t.Fatal(err)
	}
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(tombstone); !os.IsNotExist(err) {
		t.Fatalf("tombstone survived lock recovery: %v", err)
	}
	target := filepath.Join(t.TempDir(), "target")
	write(t, target, "old")
	tx, err := beginArtifactTransaction([]string{target})
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, tombstone); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := tx.commit(); err == nil || !strings.Contains(err.Error(), "tombstone") {
		t.Fatalf("commit did not propagate finalization failure: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	_ = root
}

func TestValidateJournalItemsReportsFirstPriorOverlapDeterministically(t *testing.T) {
	anchor := privateConfigDir(t)
	info, err := os.Lstat(anchor)
	if err != nil {
		t.Fatal(err)
	}
	anchorID, err := stableInstallDirIdentity(anchor, info)
	if err != nil {
		t.Fatal(err)
	}
	items := []installJournalItem{
		{Target: filepath.Join(anchor, "z-prior", "leaf"), Backup: "snapshots/000000", Anchor: anchor, AnchorID: anchorID},
		{Target: filepath.Join(anchor, "a-prior", "leaf"), Backup: "snapshots/000001", Anchor: anchor, AnchorID: anchorID},
		{Target: anchor, Backup: "snapshots/000002", Anchor: anchor, AnchorID: anchorID},
	}
	want := fmt.Sprintf("journal targets overlap or repeat: %s and %s", items[0].Target, anchor)
	for iteration := 0; iteration < 100; iteration++ {
		err := validateJournalItems(filepath.Join(anchor, "journal"), items, false)
		if err == nil || err.Error() != want {
			t.Fatalf("iteration %d overlap diagnostic = %v, want %q", iteration, err, want)
		}
	}
}

func TestRetainedActiveCapabilityChoosesCanonicalOverlappingAnchor(t *testing.T) {
	base := t.TempDir()
	outerPath := filepath.Join(base, "a-anchor")
	innerPath := filepath.Join(outerPath, "nested")
	if err := os.MkdirAll(innerPath, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(innerPath, "artifact")
	outerRoot, err := os.OpenRoot(outerPath)
	if err != nil {
		t.Fatal(err)
	}
	innerRoot, err := os.OpenRoot(innerPath)
	if err != nil {
		outerRoot.Close()
		t.Fatal(err)
	}
	outer := &installAnchorCapability{path: outerPath, id: "outer", root: outerRoot, items: []installJournalItem{{Target: target}}}
	inner := &installAnchorCapability{path: innerPath, id: "inner", root: innerRoot, items: []installJournalItem{{Target: target}}}
	activeInstallAnchors.Lock()
	previous := activeInstallAnchors.byPath
	// Insert the lexically later/nested candidate first. Map iteration must not
	// decide which retained root is selected.
	activeInstallAnchors.byPath = map[string][]*installAnchorCapability{innerPath: {inner}, outerPath: {outer}}
	activeInstallAnchors.Unlock()
	t.Cleanup(func() {
		activeInstallAnchors.Lock()
		activeInstallAnchors.byPath = previous
		activeInstallAnchors.Unlock()
		_ = outerRoot.Close()
		_ = innerRoot.Close()
	})
	for iteration := 0; iteration < 100; iteration++ {
		got, rel, ok := retainedActiveCapability(target)
		if !ok || got != outerRoot || rel != filepath.Join("nested", "artifact") {
			t.Fatalf("iteration %d capability = %p, %q, %v; want outer %p", iteration, got, rel, ok, outerRoot)
		}
	}
}

func assertNoInstallJournal(t *testing.T) {
	t.Helper()
	root, tombstone, err := installJournalPaths()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{root, tombstone} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("install transaction artifact remains at %s: %v", path, err)
		}
	}
}
