package install

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/filelock"
)

func TestConcurrentInstallCannotInterleaveAndLockIsReusable(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	src := fakeSource(t)
	firstHome := filepath.Join(t.TempDir(), "first")
	secondHome := filepath.Join(t.TempDir(), "second")

	entered := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)
	var once sync.Once
	go func() {
		firstDone <- Install(Options{
			Homes: []string{firstHome},
			From:  src,
			Out:   io.Discard,
			beforeCommit: func(string) error {
				once.Do(func() { close(entered) })
				<-release
				return nil
			},
		})
	}()
	<-entered

	err := Install(Options{Homes: []string{secondHome}, From: src, Out: io.Discard})
	if err == nil || !strings.Contains(err.Error(), "another operation holds the lock") {
		t.Fatalf("contending Install error = %v", err)
	}
	if _, statErr := os.Lstat(filepath.Join(secondHome, "skills", "machinery")); !os.IsNotExist(statErr) {
		t.Fatalf("contending install mutated its target: %v", statErr)
	}

	close(release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if err := Install(Options{Homes: []string{secondHome}, From: src, Out: io.Discard}); err != nil {
		t.Fatalf("released install lock was not reusable: %v", err)
	}
}

func TestAcquireInstallOperationLockPreservesReleaseFailures(t *testing.T) {
	releaseErr := errors.New("injected lock release failure")
	originalRelease := installFileLockRelease
	installFileLockRelease = func(lock *filelock.Lock) error {
		return errors.Join(originalRelease(lock), releaseErr)
	}
	t.Cleanup(func() { installFileLockRelease = originalRelease })

	t.Run("activation cleanup", func(t *testing.T) {
		config := privateConfigDir(t)
		t.Setenv("MACHINERY_CONFIG_DIR", config)
		if !createInvalidActivationStageForTest(t) {
			t.Skip("activation staging cleanup is not applicable on this platform")
		}
		_, err := acquireInstallOperationLock()
		if err == nil || !strings.Contains(err.Error(), "not a real directory") || !errors.Is(err, releaseErr) {
			t.Fatalf("activation cleanup acquisition error = %v", err)
		}
	})

	t.Run("transaction recovery", func(t *testing.T) {
		config := privateConfigDir(t)
		t.Setenv("MACHINERY_CONFIG_DIR", config)
		journal, _, err := installJournalPaths()
		if err != nil {
			t.Fatal(err)
		}
		write(t, journal, "not-a-directory")
		_, err = acquireInstallOperationLock()
		if err == nil || !strings.Contains(err.Error(), "recover interrupted install transaction") || !errors.Is(err, releaseErr) {
			t.Fatalf("transaction recovery acquisition error = %v", err)
		}
	})

	t.Run("lost delegated parent", func(t *testing.T) {
		t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
		t.Setenv(installLockCapabilityEnv, "invalid-parent-capability")
		_, err := acquireInstallOperationLock()
		if err == nil || !strings.Contains(err.Error(), "lost its parent lock") || !errors.Is(err, releaseErr) {
			t.Fatalf("delegated acquisition error = %v", err)
		}
	})
}

func TestUpdateContentionFailsBeforeReadingOrMutatingState(t *testing.T) {
	config := privateConfigDir(t)
	t.Setenv("MACHINERY_CONFIG_DIR", config)
	write(t, filepath.Join(config, "install.json"), "{corrupt")
	destination := filepath.Join(t.TempDir(), "machinery")
	write(t, destination, "known-good")

	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	called := false
	_, updateErr := Update(UpdateOptions{
		Executable: destination,
		Out:        io.Discard,
		run: func(string, ...string) (string, error) {
			called = true
			return "", nil
		},
	})
	if releaseErr := lock.Release(); releaseErr != nil {
		t.Fatal(releaseErr)
	}
	if updateErr == nil || !strings.Contains(updateErr.Error(), "another operation holds the lock") {
		t.Fatalf("contending Update error = %v", updateErr)
	}
	if called {
		t.Fatal("contending Update executed external work")
	}
	if got, readErr := os.ReadFile(destination); readErr != nil || string(got) != "known-good" {
		t.Fatalf("contending Update changed destination: got %q, err %v", got, readErr)
	}
}

func TestUninstallContentionFailsBeforeMutation(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := filepath.Join(t.TempDir(), "home")
	seedHomeArtifactInventory(t, home)
	if err := Install(Options{Homes: []string{home}, From: fakeSource(t)}); err != nil {
		t.Fatal(err)
	}
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	uninstallErr := Uninstall([]string{home}, io.Discard)
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	if uninstallErr == nil || !strings.Contains(uninstallErr.Error(), "another operation holds the lock") {
		t.Fatalf("contending Uninstall error = %v", uninstallErr)
	}
	if _, err := os.Stat(filepath.Join(home, "skills", "machinery", "SKILL.md")); err != nil {
		t.Fatalf("contending uninstall mutated target: %v", err)
	}
}

func TestAbandonedLegacyReceiptLockDirectoryDoesNotBlock(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	receiptPath, err := installationReceiptPath()
	if err != nil {
		t.Fatal(err)
	}
	legacyLock := receiptPath + ".lock"
	if err := os.MkdirAll(legacyLock, 0o700); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(t.TempDir(), "home")
	seedHomeArtifactInventory(t, home)
	if err := recordHomeInstall([]string{home}, false); err != nil {
		t.Fatalf("legacy mkdir lock blocked crash-released advisory locking: %v", err)
	}
	receipt, exists, err := loadReceipt()
	if err != nil || !exists || len(receipt.HomeInstalls) != 1 {
		t.Fatalf("receipt = %+v, exists=%v, err=%v", receipt, exists, err)
	}
}

func TestUpdateChildCapabilityUsesParentsHeldLock(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := lock.Release(); err != nil {
			t.Error(err)
		}
	}()
	scope, err := installOperationScope()
	if err != nil {
		t.Fatal(err)
	}
	capability, cleanup, err := createInstallLockCapability(scope)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestInstallLockCapabilityHelper$")
	cmd.Env = append(os.Environ(),
		"MACHINERY_TEST_INSTALL_LOCK_HELPER=1",
		installLockCapabilityEnv+"="+capability.String(),
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("capability child failed: %v\n%s", err, output)
	}
}

func TestInstallInspectionLockRecoversPreparedTransactionBeforeCallback(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	target := filepath.Join(t.TempDir(), "artifact")
	write(t, target, "old")
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := beginArtifactTransaction([]string{target})
	if err != nil {
		t.Fatal(err)
	}
	transactionReplaceForTest(t, target, "partial")
	if err := tx.closeAnchors(); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	called := false
	if err := WithInstallInspectionLock(func() error {
		called = true
		got, err := os.ReadFile(target)
		if err != nil {
			return err
		}
		if string(got) != "old" {
			return fmt.Errorf("inspection observed %q, want recovered old state", got)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("inspection callback was not called")
	}
	assertNoInstallJournal(t)
}

func TestHostHookInvocationRecoversPreparedInstallTransaction(t *testing.T) {
	config := privateConfigDir(t)
	t.Setenv("MACHINERY_CONFIG_DIR", config)
	target := filepath.Join(t.TempDir(), "installed-adapter")
	write(t, target, "old")
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := beginArtifactTransaction([]string{target})
	if err != nil {
		t.Fatal(err)
	}
	transactionReplaceForTest(t, target, "partial")
	if err := tx.closeAnchors(); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}

	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "design"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(project, "design", "domain.modelith.yaml"), "model: {}\n")
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "run", "./cmd/machinery", "hook", "--root", project)
	command.Dir = repo
	command.Env = append(os.Environ(), "MACHINERY_CONFIG_DIR="+config)
	command.Stdin = strings.NewReader(`{"hook_event_name":"Stop","session_id":"recovery-proof"}`)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("host hook invocation failed: %v\n%s", err, output)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "old" {
		t.Fatalf("host hook observed unrecovered installation: %q, %v", got, err)
	}
	assertNoInstallJournal(t)
}

func TestDelegatedChildAbortsAfterParentLockIsLost(t *testing.T) {
	config := privateConfigDir(t)
	t.Setenv("MACHINERY_CONFIG_DIR", config)
	target := filepath.Join(t.TempDir(), "target")
	write(t, target, "old")
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := beginArtifactTransaction([]string{target}); err != nil {
		t.Fatal(err)
	}
	transactionReplaceForTest(t, target, "new")
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestInstallOrphanCapabilityHelper$")
	cmd.Env = append(os.Environ(),
		"MACHINERY_TEST_ORPHAN_CAPABILITY_HELPER=1",
		"MACHINERY_CONFIG_DIR="+config,
		installLockCapabilityEnv+"=orphaned-parent-capability",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("orphan capability helper failed: %v\n%s", err, output)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "old" {
		t.Fatalf("orphan child did not recover parent transaction: %q, %v", got, err)
	}
}

func TestInstallOrphanCapabilityHelper(t *testing.T) {
	if os.Getenv("MACHINERY_TEST_ORPHAN_CAPABILITY_HELPER") != "1" {
		return
	}
	_, err := acquireInstallOperationLock()
	if err == nil || !strings.Contains(err.Error(), "lost its parent lock") {
		t.Fatalf("orphan delegated acquisition error = %v", err)
	}
}

func TestInstallLockCapabilityHelper(t *testing.T) {
	if os.Getenv("MACHINERY_TEST_INSTALL_LOCK_HELPER") != "1" {
		return
	}
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
}
