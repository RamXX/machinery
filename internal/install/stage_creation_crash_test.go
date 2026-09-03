package install

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const installStageCrashExitCode = 91

func TestInstallStageCreationCrashHelper(t *testing.T) {
	if os.Getenv("MACHINERY_INSTALL_STAGE_CRASH_HELPER") != "1" {
		return
	}
	point := os.Getenv("MACHINERY_INSTALL_STAGE_CRASH_POINT")
	target := os.Getenv("MACHINERY_INSTALL_STAGE_CRASH_TARGET")
	source := os.Getenv("MACHINERY_INSTALL_STAGE_CRASH_SOURCE")
	afterInstallStageCreationBoundary = func(got, path string) {
		if got == point && sameInstallPath(path, target) {
			os.Exit(installStageCrashExitCode)
		}
	}
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release() //nolint:errcheck // helper normally exits at the injected boundary
	if _, err := beginArtifactTransaction([]string{target}); err != nil {
		t.Fatal(err)
	}
	if err := stageInstallEntryNoReplace(source, target, installStagePublish); err != nil {
		t.Fatal(err)
	}
	t.Fatalf("install stage creation did not reach crash point %q", point)
}

func TestInstallStageCreationRecoversEveryPersistenceBoundary(t *testing.T) {
	for _, point := range []string{
		"before-mkdir",
		"after-mkdir",
		"after-parent-sync",
		"before-stageid-persist",
		"after-stageid-persist",
	} {
		t.Run(point, func(t *testing.T) {
			config := privateConfigDir(t)
			t.Setenv("MACHINERY_CONFIG_DIR", config)
			target := filepath.Join(t.TempDir(), "target")
			source := filepath.Join(t.TempDir(), "source")
			write(t, filepath.Join(target, "original"), "old")
			write(t, filepath.Join(source, "new"), "new")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestInstallStageCreationCrashHelper$")
			command.Env = append(os.Environ(),
				"MACHINERY_CONFIG_DIR="+config,
				"MACHINERY_INSTALL_STAGE_CRASH_HELPER=1",
				"MACHINERY_INSTALL_STAGE_CRASH_POINT="+point,
				"MACHINERY_INSTALL_STAGE_CRASH_TARGET="+target,
				"MACHINERY_INSTALL_STAGE_CRASH_SOURCE="+source,
			)
			err := command.Run()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != installStageCrashExitCode {
				t.Fatalf("stage creation helper at %s = %v", point, err)
			}
			lock, err := acquireInstallOperationLock()
			if err != nil {
				t.Fatalf("recover stage creation crash at %s: %v", point, err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
			if got, err := os.ReadFile(filepath.Join(target, "original")); err != nil || string(got) != "old" {
				t.Fatalf("original after %s recovery = %q, %v", point, got, err)
			}
			if _, err := os.Lstat(filepath.Join(target, "new")); !os.IsNotExist(err) {
				t.Fatalf("transaction image survived %s recovery: %v", point, err)
			}
			entries, err := os.ReadDir(filepath.Dir(target))
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), installRestoreStagePrefix) || strings.HasPrefix(entry.Name(), installPrepareDeletePrefix) {
					t.Fatalf("stage creation residue after %s recovery: %s", point, entry.Name())
				}
			}
			assertNoInstallJournal(t)
		})
	}
}

func TestInstallStageCreationRejectsForeignFinalNameCollision(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	target := filepath.Join(t.TempDir(), "target")
	source := filepath.Join(t.TempDir(), "source")
	write(t, filepath.Join(target, "original"), "old")
	write(t, filepath.Join(source, "new"), "new")
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release() //nolint:errcheck // test cleanup
	tx, err := beginArtifactTransaction([]string{target})
	if err != nil {
		t.Fatal(err)
	}
	afterInstallStageCreationBoundary = func(point, path string) {
		if point != "before-mkdir" || !sameInstallPath(path, target) {
			return
		}
		afterInstallStageCreationBoundary = nil
		stageName := tx.journal.Items[0].StageName
		if err := os.Mkdir(filepath.Join(filepath.Dir(target), stageName), 0o700); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(filepath.Dir(target), stageName, "foreign"), "preserve")
	}
	t.Cleanup(func() { afterInstallStageCreationBoundary = nil })
	err = stageInstallEntryNoReplace(source, target, installStagePublish)
	if err == nil || !strings.Contains(err.Error(), "without replacement") {
		t.Fatalf("foreign final-name collision was accepted: %v", err)
	}
	if err := tx.rollback(); err == nil || !strings.Contains(err.Error(), "creation witness") {
		t.Fatalf("rollback accepted foreign stage collision: %v", err)
	}
	foreign := filepath.Join(filepath.Dir(target), tx.journal.Items[0].StageName, "foreign")
	if got, err := os.ReadFile(foreign); err != nil || string(got) != "preserve" {
		t.Fatalf("foreign stage collision changed: %q, %v", got, err)
	}
}

func TestInstallStageCreationRejectsWitnessCopyABA(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	target := filepath.Join(t.TempDir(), "target")
	source := filepath.Join(t.TempDir(), "source")
	write(t, filepath.Join(target, "original"), "old")
	write(t, filepath.Join(source, "new"), "new")
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release() //nolint:errcheck // test cleanup
	tx, err := beginArtifactTransaction([]string{target})
	if err != nil {
		t.Fatal(err)
	}
	afterInstallStageCreationBoundary = func(point, path string) {
		if point != "before-stageid-persist" || !sameInstallPath(path, target) {
			return
		}
		afterInstallStageCreationBoundary = nil
		stage := filepath.Join(filepath.Dir(target), tx.journal.Items[0].StageName)
		parked := stage + ".parked"
		if err := os.Rename(stage, parked); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(stage, 0o700); err != nil {
			t.Fatal(err)
		}
		witness, err := os.ReadFile(filepath.Join(parked, installRestoreIntentFile))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stage, installRestoreIntentFile), witness, 0o600); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(stage, "foreign"), "preserve")
	}
	t.Cleanup(func() { afterInstallStageCreationBoundary = nil })
	err = stageInstallEntryNoReplace(source, target, installStagePublish)
	if err == nil || !strings.Contains(err.Error(), "changed identity") {
		t.Fatalf("witness-copy ABA was accepted: %v", err)
	}
	if err := tx.rollback(); err == nil || !strings.Contains(err.Error(), "changed identity") {
		t.Fatalf("rollback accepted witness-copy ABA: %v", err)
	}
	foreign := filepath.Join(filepath.Dir(target), tx.journal.Items[0].StageName, "foreign")
	if got, err := os.ReadFile(foreign); err != nil || string(got) != "preserve" {
		t.Fatalf("witness-copy ABA replacement changed: %q, %v", got, err)
	}
}
