package modelithtx

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPublishRecoversEveryCrashBoundary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		hook     func(*hooks)
		wantBody string
	}{
		{name: "journal", hook: func(h *hooks) { h.afterJournal = crash }, wantBody: "old\n"},
		{name: "park", hook: func(h *hooks) { h.afterPark = crash }, wantBody: "old\n"},
		{name: "install", hook: func(h *hooks) { h.afterInstall = crash }, wantBody: "new\n"},
		{name: "retirement", hook: func(h *hooks) { h.beforeRetireVerify = crash }, wantBody: "new\n"},
		{name: "backup-removal", hook: func(h *hooks) { h.afterBackup = crash }, wantBody: "new\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repo := newRepository(t, "old\n", "new\n")
			expected, err := Fingerprint(filepath.Join(repo, targetName))
			if err != nil {
				t.Fatal(err)
			}
			var injected hooks
			test.hook(&injected)
			if err := publish(repo, expected, injected); !errors.Is(err, errCrash) {
				t.Fatalf("publish error = %v, want injected crash", err)
			}
			if err := Recover(repo); err != nil {
				t.Fatalf("recover: %v", err)
			}
			body, err := os.ReadFile(filepath.Join(repo, targetName, "domain.modelith.md"))
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != test.wantBody {
				t.Fatalf("recovered body = %q, want %q", body, test.wantBody)
			}
			for _, reserved := range []string{stageName, backupName, retireName, journalName, journalNextName, journalAuthorityName, journalRetireName} {
				if _, err := os.Lstat(filepath.Join(repo, reserved)); !os.IsNotExist(err) {
					t.Fatalf("reserved residue %s remains: %v", reserved, err)
				}
			}

			// A retry must be able to stage and publish another exact corpus.
			stageCorpusFile(t, repo, "retry\n")
			retryExpected, err := Fingerprint(filepath.Join(repo, targetName))
			if err != nil {
				t.Fatal(err)
			}
			if err := Publish(repo, retryExpected); err != nil {
				t.Fatalf("retry publish: %v", err)
			}
		})
	}
}

func TestRecoveryRefusesChangedOrReplacedParkedBackup(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{
			name: "changed content",
			mutate: func(t *testing.T, repo string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(repo, backupName, "domain.modelith.md"), []byte("external\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "same-content path replacement",
			mutate: func(t *testing.T, repo string) {
				t.Helper()
				if err := os.Rename(filepath.Join(repo, backupName), filepath.Join(repo, "retained-original")); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(repo, backupName), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(repo, backupName, "domain.modelith.md"), []byte("old\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := newRepository(t, "old\n", "new\n")
			expected, err := Fingerprint(filepath.Join(repo, targetName))
			if err != nil {
				t.Fatal(err)
			}
			if err := publish(repo, expected, hooks{afterPark: crash}); !errors.Is(err, errCrash) {
				t.Fatalf("publish error = %v, want simulated parked crash", err)
			}
			test.mutate(t, repo)
			recoverErr := Recover(repo)
			if recoverErr == nil || !strings.Contains(recoverErr.Error(), "preserving backup and recovery journal") {
				t.Fatalf("recovery accepted changed parked backup: %v", recoverErr)
			}
			if _, err := os.Lstat(filepath.Join(repo, targetName)); !os.IsNotExist(err) {
				t.Fatalf("changed parked backup became public: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(repo, backupName)); err != nil {
				t.Fatalf("changed parked backup was not preserved: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(repo, journalName)); err != nil {
				t.Fatalf("recovery journal was not preserved: %v", err)
			}
		})
	}
}

func TestRecoveryRejectsChangedJournalAuthorityAfterIsolation(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(string, []byte) error
	}{
		{
			name: "content mutation",
			mutate: func(path string, _ []byte) error {
				return os.WriteFile(path, []byte("external journal\n"), 0o600)
			},
		},
		{
			name: "same-file content ABA",
			mutate: func(path string, body []byte) error {
				changed := append([]byte(nil), body...)
				changed[len(changed)-2] ^= 1
				if err := os.WriteFile(path, changed, 0o600); err != nil {
					return err
				}
				return os.WriteFile(path, body, 0o600)
			},
		},
		{
			name: "path replacement",
			mutate: func(path string, _ []byte) error {
				if err := os.Rename(path, path+".original"); err != nil {
					return err
				}
				return os.WriteFile(path, []byte("external journal\n"), 0o600)
			},
		},
		{
			name: "same-byte path ABA",
			mutate: func(path string, body []byte) error {
				if err := os.Rename(path, path+".original"); err != nil {
					return err
				}
				return os.WriteFile(path, body, 0o600)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := newRepository(t, "old\n", "new\n")
			expected, err := Fingerprint(filepath.Join(repo, targetName))
			if err != nil {
				t.Fatal(err)
			}
			if err := publish(repo, expected, hooks{afterPark: crash}); !errors.Is(err, errCrash) {
				t.Fatalf("publish error = %v, want simulated parked crash", err)
			}
			original, err := os.ReadFile(filepath.Join(repo, journalName))
			if err != nil {
				t.Fatal(err)
			}
			recoverErr := recoverTransactionWithHooks(repo, recoveryHooks{afterJournalIsolation: func() error {
				return test.mutate(filepath.Join(repo, journalAuthorityName), original)
			}})
			if recoverErr == nil || !strings.Contains(recoverErr.Error(), "journal changed after isolation") {
				t.Fatalf("recovery accepted changed journal authority: %v", recoverErr)
			}
			if _, err := os.Lstat(filepath.Join(repo, journalName)); err != nil {
				t.Fatalf("changed authority was not preserved at the discoverable journal path: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(repo, journalAuthorityName)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("isolated authority residue remained after failed recovery: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(repo, targetName)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("stale journal authority drove target restoration: %v", err)
			}
			assertFileBody(t, filepath.Join(repo, backupName, "domain.modelith.md"), "old\n")
		})
	}
}

func mutateModelithJournalAuthority(repo, mode string) error {
	path := filepath.Join(repo, journalAuthorityName)
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	switch mode {
	case "content":
		return os.WriteFile(path, []byte("external journal\n"), 0o600)
	case "content-aba":
		changed := append([]byte(nil), body...)
		changed[len(changed)-2] ^= 1
		if err := os.WriteFile(path, changed, 0o600); err != nil {
			return err
		}
		return os.WriteFile(path, body, 0o600)
	case "path":
		if err := os.Rename(path, path+".external-original"); err != nil {
			return err
		}
		return os.WriteFile(path, []byte("external journal\n"), 0o600)
	case "same-byte-path-aba":
		if err := os.Rename(path, path+".external-original"); err != nil {
			return err
		}
		return os.WriteFile(path, body, 0o600)
	default:
		return errors.New("unknown journal mutation")
	}
}

func TestPublishRejectsChangedJournalAuthorityAfterInstallation(t *testing.T) {
	for _, mode := range []string{"content", "content-aba", "path", "same-byte-path-aba"} {
		t.Run(mode, func(t *testing.T) {
			repo := newRepository(t, "old\n", "new\n")
			expected, err := Fingerprint(filepath.Join(repo, targetName))
			if err != nil {
				t.Fatal(err)
			}
			err = publish(repo, expected, hooks{afterJournal: func() error {
				return mutateModelithJournalAuthority(repo, mode)
			}})
			if err == nil || !strings.Contains(err.Error(), "journal authority changed after installation") {
				t.Fatalf("publish accepted changed journal authority: %v", err)
			}
			assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "old\n")
			if _, err := os.Lstat(filepath.Join(repo, backupName)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("changed authority drove publication: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(repo, journalName)); err != nil {
				t.Fatalf("changed authority was not preserved at canonical path: %v", err)
			}
		})
	}
}

func TestPublishAbortCleanupPreservesChangedJournalAuthority(t *testing.T) {
	for _, mode := range []string{"content", "content-aba", "path", "same-byte-path-aba"} {
		t.Run(mode, func(t *testing.T) {
			repo := newRepository(t, "old\n", "new\n")
			expected, err := Fingerprint(filepath.Join(repo, targetName))
			if err != nil {
				t.Fatal(err)
			}
			err = publish(repo, expected, hooks{
				afterJournal: func() error {
					return os.WriteFile(filepath.Join(repo, targetName, "external.txt"), []byte("preserve\n"), 0o644)
				},
				beforeAbortCleanup: func() error {
					return mutateModelithJournalAuthority(repo, mode)
				},
			})
			if err == nil || !strings.Contains(err.Error(), "journal changed at retirement boundary") {
				t.Fatalf("abort cleanup accepted changed journal authority: %v", err)
			}
			assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "old\n")
			assertFileBody(t, filepath.Join(repo, targetName, "external.txt"), "preserve\n")
			if _, err := os.Lstat(filepath.Join(repo, journalName)); err != nil {
				t.Fatalf("abort cleanup did not preserve changed authority: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(repo, stageName)); err != nil {
				t.Fatalf("abort cleanup removed stage after losing authority: %v", err)
			}
		})
	}
}

func TestRecoveryRevalidatesParkedBackupAtRestoreBoundary(t *testing.T) {
	repo := newRepository(t, "old\n", "new\n")
	expected, err := Fingerprint(filepath.Join(repo, targetName))
	if err != nil {
		t.Fatal(err)
	}
	if err := publish(repo, expected, hooks{afterPark: crash}); !errors.Is(err, errCrash) {
		t.Fatalf("publish error = %v, want simulated parked crash", err)
	}
	replacement := filepath.Join(repo, "replacement")
	if err := os.Mkdir(replacement, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(replacement, "domain.modelith.md"), []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	recoverErr := recoverTransactionWithHooks(repo, recoveryHooks{beforeBackupRestoreVerify: func() error {
		if err := os.Rename(filepath.Join(repo, backupName), filepath.Join(repo, "retained-original")); err != nil {
			return err
		}
		return os.Rename(replacement, filepath.Join(repo, backupName))
	}})
	if recoverErr == nil || !strings.Contains(recoverErr.Error(), "changed at the atomic restoration boundary") {
		t.Fatalf("recovery accepted same-content backup ABA: %v", recoverErr)
	}
	if _, err := os.Lstat(filepath.Join(repo, targetName)); !os.IsNotExist(err) {
		t.Fatalf("ABA replacement became public: %v", err)
	}
	assertFileBody(t, filepath.Join(repo, backupName, "domain.modelith.md"), "old\n")
	if _, err := os.Lstat(filepath.Join(repo, journalName)); err != nil {
		t.Fatalf("recovery journal was not preserved: %v", err)
	}
}

func TestRecoveryRevalidatesRestoredBackupBeforeFinishing(t *testing.T) {
	repo := newRepository(t, "old\n", "new\n")
	expected, err := Fingerprint(filepath.Join(repo, targetName))
	if err != nil {
		t.Fatal(err)
	}
	if err := publish(repo, expected, hooks{afterPark: crash}); !errors.Is(err, errCrash) {
		t.Fatalf("publish error = %v, want simulated parked crash", err)
	}
	recoverErr := recoverTransactionWithHooks(repo, recoveryHooks{afterBackupRestore: func() error {
		return os.WriteFile(filepath.Join(repo, targetName, "domain.modelith.md"), []byte("external\n"), 0o644)
	}})
	if recoverErr == nil || !strings.Contains(recoverErr.Error(), "changed before restoration completed") {
		t.Fatalf("recovery accepted mutation after restore rename: %v", recoverErr)
	}
	assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "external\n")
	if _, err := os.Lstat(filepath.Join(repo, journalName)); err != nil {
		t.Fatalf("recovery journal was not preserved: %v", err)
	}
	secondErr := Recover(repo)
	if secondErr == nil || !strings.Contains(secondErr.Error(), "does not match the journaled backup witness") {
		t.Fatalf("second recovery misclassified failed restoration as a pre-park abort: %v", secondErr)
	}
	assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "external\n")
	if _, err := os.Lstat(filepath.Join(repo, journalName)); err != nil {
		t.Fatalf("second recovery removed journal after failed restoration: %v", err)
	}
}

func TestRecoveryRestoreStateSurvivesCrashAndPartialNextJournal(t *testing.T) {
	repo := newRepository(t, "old\n", "new\n")
	expected, err := Fingerprint(filepath.Join(repo, targetName))
	if err != nil {
		t.Fatal(err)
	}
	if err := publish(repo, expected, hooks{afterPark: crash}); !errors.Is(err, errCrash) {
		t.Fatalf("publish error = %v, want simulated parked crash", err)
	}
	if err := recoverTransactionWithHooks(repo, recoveryHooks{beforeBackupRestoreVerify: crash}); !errors.Is(err, errCrash) {
		t.Fatalf("recovery error = %v, want simulated restoration crash", err)
	}
	if err := os.WriteFile(filepath.Join(repo, journalNextName), []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Recover(repo); err != nil {
		t.Fatalf("retry recovery after restoration-state crash: %v", err)
	}
	assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "old\n")
	assertNoTransactionResidue(t, repo)
}

func TestRecoverRejectsUnsafeReservedResidue(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, targetName), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "sentinel")
	if err := os.WriteFile(outside, []byte("outside\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repo, stageName)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		err := Recover(repo)
		if err == nil || err.Error() != "reserved Modelith transaction entry .machinery-modelith-stage must not be a symlink" {
			t.Fatalf("iteration %d: unexpected error %v", i, err)
		}
	}
	body, err := os.ReadFile(outside)
	if err != nil || string(body) != "outside\n" {
		t.Fatalf("outside sentinel changed: body=%q err=%v", body, err)
	}
}

func TestPublishFinalLiveCorpusRevalidationPreservesConcurrentWork(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, repo string)
		check  func(t *testing.T, repo string)
	}{
		{
			name: "add",
			mutate: func(t *testing.T, repo string) {
				if err := os.WriteFile(filepath.Join(repo, targetName, "concurrent.txt"), []byte("external\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			check: func(t *testing.T, repo string) {
				assertFileBody(t, filepath.Join(repo, targetName, "concurrent.txt"), "external\n")
			},
		},
		{
			name: "change",
			mutate: func(t *testing.T, repo string) {
				if err := os.WriteFile(filepath.Join(repo, targetName, "domain.modelith.md"), []byte("ext\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			check: func(t *testing.T, repo string) {
				assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "ext\n")
			},
		},
		{
			name: "remove",
			mutate: func(t *testing.T, repo string) {
				if err := os.Remove(filepath.Join(repo, targetName, "domain.modelith.md")); err != nil {
					t.Fatal(err)
				}
			},
			check: func(t *testing.T, repo string) {
				if _, err := os.Lstat(filepath.Join(repo, targetName, "domain.modelith.md")); !os.IsNotExist(err) {
					t.Fatalf("concurrent removal was overwritten or recreated: %v", err)
				}
			},
		},
		{
			name: "content ABA",
			mutate: func(t *testing.T, repo string) {
				path := filepath.Join(repo, targetName, "domain.modelith.md")
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("tmp\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				changed := info.ModTime().Add(2 * time.Second)
				if err := os.Chtimes(path, changed, changed); err != nil {
					t.Fatal(err)
				}
			},
			check: func(t *testing.T, repo string) {
				assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "old\n")
			},
		},
		{
			name: "directory inventory ABA",
			mutate: func(t *testing.T, repo string) {
				dir := filepath.Join(repo, targetName)
				info, err := os.Stat(dir)
				if err != nil {
					t.Fatal(err)
				}
				temporary := filepath.Join(dir, "temporary")
				if err := os.WriteFile(temporary, []byte("external\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(temporary); err != nil {
					t.Fatal(err)
				}
				changed := info.ModTime().Add(2 * time.Second)
				if err := os.Chtimes(dir, changed, changed); err != nil {
					t.Fatal(err)
				}
			},
			check: func(t *testing.T, repo string) {
				assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "old\n")
			},
		},
		{
			name: "directory identity",
			mutate: func(t *testing.T, repo string) {
				live := filepath.Join(repo, targetName)
				parked := filepath.Join(repo, "external-original")
				if err := os.Rename(live, parked); err != nil {
					t.Fatal(err)
				}
				writeCorpusFile(t, live, "replacement\n")
			},
			check: func(t *testing.T, repo string) {
				assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "replacement\n")
				assertFileBody(t, filepath.Join(repo, "external-original", "domain.modelith.md"), "old\n")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := newRepository(t, "old\n", "new\n")
			expected, err := Fingerprint(filepath.Join(repo, targetName))
			if err != nil {
				t.Fatal(err)
			}
			err = publish(repo, expected, hooks{afterJournal: func() error {
				test.mutate(t, repo)
				return nil
			}})
			if err == nil || !strings.Contains(err.Error(), "live corpus changed before publication") {
				t.Fatalf("concurrent live mutation was accepted: %v", err)
			}
			test.check(t, repo)
			if _, err := os.Lstat(filepath.Join(repo, backupName)); !os.IsNotExist(err) {
				t.Fatalf("publication parked or replaced live work before rejecting it: %v", err)
			}
			assertNoTransactionResidue(t, repo)
		})
	}
}

func TestPublishRevalidatesAtomicallyParkedCorpusBeforeInstall(t *testing.T) {
	for _, test := range []struct {
		name string
		hook func(repo string) hooks
	}{
		{
			name: "change after final live traversal before park",
			hook: func(repo string) hooks {
				return hooks{afterLiveRevalidate: func() error {
					if err := os.WriteFile(filepath.Join(repo, targetName, "domain.modelith.md"), []byte("late\n"), 0o644); err != nil {
						t.Fatal(err)
					}
					return nil
				}}
			},
		},
		{
			name: "change parked inode before install",
			hook: func(repo string) hooks {
				return hooks{afterPark: func() error {
					if err := os.WriteFile(filepath.Join(repo, backupName, "domain.modelith.md"), []byte("late\n"), 0o644); err != nil {
						t.Fatal(err)
					}
					return nil
				}}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := newRepository(t, "old\n", "new\n")
			expected, err := Fingerprint(filepath.Join(repo, targetName))
			if err != nil {
				t.Fatal(err)
			}
			err = publish(repo, expected, test.hook(repo))
			if err == nil || !strings.Contains(err.Error(), "parked Modelith corpus changed before staged publication") {
				t.Fatalf("post-traversal mutation was accepted: %v", err)
			}
			assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "late\n")
			if _, err := os.Lstat(filepath.Join(repo, backupName)); !os.IsNotExist(err) {
				t.Fatalf("parked concurrent work was not restored: %v", err)
			}
			assertNoTransactionResidue(t, repo)
		})
	}
}

func TestPublishConcurrencyAbortCanRetryCleanly(t *testing.T) {
	repo := newRepository(t, "old\n", "new\n")
	expected, err := Fingerprint(filepath.Join(repo, targetName))
	if err != nil {
		t.Fatal(err)
	}
	err = publish(repo, expected, hooks{afterJournal: func() error {
		return os.WriteFile(filepath.Join(repo, targetName, "external.txt"), []byte("preserve\n"), 0o644)
	}})
	if err == nil {
		t.Fatal("concurrent addition was accepted")
	}
	assertNoTransactionResidue(t, repo)
	assertFileBody(t, filepath.Join(repo, targetName, "external.txt"), "preserve\n")
	stageCorpusFile(t, repo, "retry\n")
	fresh, err := Fingerprint(filepath.Join(repo, targetName))
	if err != nil {
		t.Fatal(err)
	}
	if err := Publish(repo, fresh); err != nil {
		t.Fatalf("clean retry after concurrency abort: %v", err)
	}
	assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "retry\n")
	assertNoTransactionResidue(t, repo)
}

func TestRecoverPreservesLiveEditWhenCrashPreemptsConcurrencyAbort(t *testing.T) {
	repo := newRepository(t, "old\n", "new\n")
	expected, err := Fingerprint(filepath.Join(repo, targetName))
	if err != nil {
		t.Fatal(err)
	}
	err = publish(repo, expected, hooks{afterJournal: func() error {
		if err := os.WriteFile(filepath.Join(repo, targetName, "domain.modelith.md"), []byte("external\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return errCrash
	}})
	if !errors.Is(err, errCrash) {
		t.Fatalf("publish error = %v, want simulated crash", err)
	}
	if err := Recover(repo); err != nil {
		t.Fatalf("recover pre-park concurrency crash: %v", err)
	}
	assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "external\n")
	assertNoTransactionResidue(t, repo)
}

func TestInstalledConcurrencyNeverDeletesEditedTargetOrBackup(t *testing.T) {
	for _, test := range []struct {
		name      string
		hooks     func(repo string) hooks
		wantCrash bool
	}{
		{
			name: "edit before installed verification",
			hooks: func(repo string) hooks {
				return hooks{beforeInstalledVerify: func() error {
					return os.WriteFile(filepath.Join(repo, targetName, "domain.modelith.md"), []byte("external\n"), 0o644)
				}}
			},
		},
		{
			name: "edit after installed verification",
			hooks: func(repo string) hooks {
				return hooks{afterInstall: func() error {
					return os.WriteFile(filepath.Join(repo, targetName, "domain.modelith.md"), []byte("external\n"), 0o644)
				}}
			},
		},
		{
			name:      "edit and crash after installed verification",
			wantCrash: true,
			hooks: func(repo string) hooks {
				return hooks{afterInstall: func() error {
					if err := os.WriteFile(filepath.Join(repo, targetName, "domain.modelith.md"), []byte("external\n"), 0o644); err != nil {
						return err
					}
					return errCrash
				}}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := newRepository(t, "old\n", "new\n")
			expected, err := Fingerprint(filepath.Join(repo, targetName))
			if err != nil {
				t.Fatal(err)
			}
			err = publish(repo, expected, test.hooks(repo))
			if test.wantCrash {
				if !errors.Is(err, errCrash) {
					t.Fatalf("publish error = %v, want simulated crash", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "preserving installed target and backup") {
				t.Fatalf("installed concurrent edit was accepted: %v", err)
			}
			assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "external\n")
			assertFileBody(t, filepath.Join(repo, backupName, "domain.modelith.md"), "old\n")
			for i := 0; i < 2; i++ {
				recoverErr := Recover(repo)
				if recoverErr == nil || !strings.Contains(recoverErr.Error(), "preserving installed target and backup") {
					t.Fatalf("recovery %d did not fail closed on edited installed target: %v", i, recoverErr)
				}
				assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "external\n")
				assertFileBody(t, filepath.Join(repo, backupName, "domain.modelith.md"), "old\n")
			}
			if _, err := os.Lstat(filepath.Join(repo, journalName)); err != nil {
				t.Fatalf("authoritative recovery journal was not retained: %v", err)
			}
		})
	}
}

func TestBackupRetirementRejectsLateOpenFileMutation(t *testing.T) {
	repo := newRepository(t, "old\n", "new\n")
	expected, err := Fingerprint(filepath.Join(repo, targetName))
	if err != nil {
		t.Fatal(err)
	}
	var held *os.File
	err = publish(repo, expected, hooks{
		afterInstall: func() error {
			var err error
			held, err = os.OpenFile(filepath.Join(repo, backupName, "domain.modelith.md"), os.O_WRONLY, 0)
			return err
		},
		beforeRetireVerify: func() error {
			if held == nil {
				return errors.New("parked file descriptor was not retained")
			}
			if _, err := held.WriteAt([]byte("ext\n"), 0); err != nil {
				return err
			}
			return errors.Join(held.Sync(), held.Close())
		},
	})
	if held != nil {
		_ = held.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "changed at retirement boundary") {
		t.Fatalf("late write through parked open descriptor was accepted: %v", err)
	}
	assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "new\n")
	assertFileBody(t, filepath.Join(repo, backupName, "domain.modelith.md"), "ext\n")
	if _, err := os.Lstat(filepath.Join(repo, retireName)); !os.IsNotExist(err) {
		t.Fatalf("changed retirement tree was not restored to backup: %v", err)
	}
	recoverErr := Recover(repo)
	if recoverErr == nil || !strings.Contains(recoverErr.Error(), "preserving installed target and backup") {
		t.Fatalf("recovery retired changed backup: %v", recoverErr)
	}
	assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "new\n")
	assertFileBody(t, filepath.Join(repo, backupName, "domain.modelith.md"), "ext\n")
}

func TestRecoveryRetirementRejectsLateOpenFileMutation(t *testing.T) {
	repo := newRepository(t, "old\n", "new\n")
	expected, err := Fingerprint(filepath.Join(repo, targetName))
	if err != nil {
		t.Fatal(err)
	}
	if err := publish(repo, expected, hooks{afterInstall: crash}); !errors.Is(err, errCrash) {
		t.Fatalf("publish error = %v, want simulated installed crash", err)
	}
	held, err := os.OpenFile(filepath.Join(repo, backupName, "domain.modelith.md"), os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	recoverErr := recoverTransaction(repo, func() error {
		if _, err := held.WriteAt([]byte("ext\n"), 0); err != nil {
			return err
		}
		return errors.Join(held.Sync(), held.Close())
	})
	_ = held.Close()
	if recoverErr == nil || !strings.Contains(recoverErr.Error(), "changed at retirement boundary") {
		t.Fatalf("recovery deleted late-mutated backup: %v", recoverErr)
	}
	assertFileBody(t, filepath.Join(repo, targetName, "domain.modelith.md"), "new\n")
	assertFileBody(t, filepath.Join(repo, backupName, "domain.modelith.md"), "ext\n")
	if _, err := os.Lstat(filepath.Join(repo, retireName)); !os.IsNotExist(err) {
		t.Fatalf("recovery did not restore changed retirement tree: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(repo, journalName)); err != nil {
		t.Fatalf("recovery journal was not preserved: %v", err)
	}
}

func TestPathExistsPropagatesNonENOENT(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "not-a-directory"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(repo)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	exists, err := pathExists(root, filepath.Join("not-a-directory", "child"))
	if err == nil || exists || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("non-ENOENT lookup failure collapsed into absence: exists=%t err=%v", exists, err)
	}
}

func assertFileBody(t *testing.T, path, want string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != want {
		t.Fatalf("%s body = %q, want %q", path, body, want)
	}
}

var errCrash = errors.New("simulated power loss")

func crash() error { return errCrash }

func newRepository(t *testing.T, oldBody, newBody string) string {
	t.Helper()
	repo := t.TempDir()
	writeCorpusFile(t, filepath.Join(repo, targetName), oldBody)
	stageCorpusFile(t, repo, newBody)
	return repo
}

func stageCorpusFile(t *testing.T, repo, body string) {
	t.Helper()
	stage := filepath.Join(repo, stageName)
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	writeCorpusFile(t, filepath.Join(stage, targetName), body)
}

func writeCorpusFile(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "domain.modelith.yaml"), []byte("entities: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "domain.modelith.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
