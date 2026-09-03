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

const installAuthorityCrashExitCode = 92

func TestInstallAuthorityRecordCrashHelper(t *testing.T) {
	if os.Getenv("MACHINERY_INSTALL_AUTHORITY_CRASH_HELPER") != "1" {
		return
	}
	family := os.Getenv("MACHINERY_INSTALL_AUTHORITY_FAMILY")
	point := os.Getenv("MACHINERY_INSTALL_AUTHORITY_POINT")
	target := os.Getenv("MACHINERY_INSTALL_AUTHORITY_TARGET")
	source := os.Getenv("MACHINERY_INSTALL_AUTHORITY_SOURCE")
	afterInstallAuthorityRecordBoundary = func(gotFamily, gotPoint string) {
		if gotFamily == family && gotPoint == point {
			os.Exit(installAuthorityCrashExitCode)
		}
	}
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release() //nolint:errcheck // helper normally exits at the injected boundary
	switch family {
	case "metadata", "marker:" + installJournalPrepared:
		if _, err := beginArtifactTransaction([]string{target}); err != nil {
			t.Fatal(err)
		}
	case "metadata-next":
		if _, err := beginArtifactTransaction([]string{target}); err != nil {
			t.Fatal(err)
		}
		if err := declareInstallPresentPostImage(target, target); err != nil {
			t.Fatal(err)
		}
	case "marker:" + installJournalCommitted:
		tx, err := beginArtifactTransaction([]string{target})
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.commit(); err != nil {
			t.Fatal(err)
		}
	case "stage-witness":
		if _, err := beginArtifactTransaction([]string{target}); err != nil {
			t.Fatal(err)
		}
		if err := stageInstallEntryNoReplace(source, target, installStagePublish); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unknown authority family %q", family)
	}
	t.Fatalf("authority record did not reach %s/%s", family, point)
}

func TestInstallAuthorityRecordsRecoverEveryPublicationBoundary(t *testing.T) {
	for _, family := range []string{
		"metadata",
		"metadata-next",
		"marker:" + installJournalPrepared,
		"marker:" + installJournalCommitted,
		"stage-witness",
	} {
		for _, point := range []string{
			"after-create",
			"partial-write",
			"after-write",
			"fsync",
			"close",
			"pre-rename",
			"post-rename",
		} {
			t.Run(family+"/"+point, func(t *testing.T) {
				config := privateConfigDir(t)
				t.Setenv("MACHINERY_CONFIG_DIR", config)
				target := filepath.Join(t.TempDir(), "target")
				source := filepath.Join(t.TempDir(), "source")
				write(t, filepath.Join(target, "original"), "old")
				write(t, filepath.Join(source, "new"), "new")
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestInstallAuthorityRecordCrashHelper$")
				command.Env = append(os.Environ(),
					"MACHINERY_CONFIG_DIR="+config,
					"MACHINERY_INSTALL_AUTHORITY_CRASH_HELPER=1",
					"MACHINERY_INSTALL_AUTHORITY_FAMILY="+family,
					"MACHINERY_INSTALL_AUTHORITY_POINT="+point,
					"MACHINERY_INSTALL_AUTHORITY_TARGET="+target,
					"MACHINERY_INSTALL_AUTHORITY_SOURCE="+source,
				)
				err := command.Run()
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) || exitErr.ExitCode() != installAuthorityCrashExitCode {
					t.Fatalf("authority helper at %s/%s = %v", family, point, err)
				}
				lock, err := acquireInstallOperationLock()
				if err != nil {
					t.Fatalf("recover authority crash at %s/%s: %v", family, point, err)
				}
				if err := lock.Release(); err != nil {
					t.Fatal(err)
				}
				if got, err := os.ReadFile(filepath.Join(target, "original")); err != nil || string(got) != "old" {
					t.Fatalf("original after %s/%s recovery = %q, %v", family, point, got, err)
				}
				assertNoInstallJournal(t)
				entries, err := os.ReadDir(filepath.Dir(target))
				if err != nil {
					t.Fatal(err)
				}
				for _, entry := range entries {
					if strings.HasPrefix(entry.Name(), installRestoreStagePrefix) || strings.HasPrefix(entry.Name(), installRecordDeletePrefix) {
						t.Fatalf("authority residue after %s/%s recovery: %s", family, point, entry.Name())
					}
				}
			})
		}
	}
}

func TestInstallAuthorityRecordPreservesForeignFixedCollision(t *testing.T) {
	rootPath := t.TempDir()
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	name := "AUTHORITY"
	afterInstallAuthorityRecordBoundary = func(family, point string) {
		if family == "test" && point == "pre-rename" {
			afterInstallAuthorityRecordBoundary = nil
			if err := os.WriteFile(filepath.Join(rootPath, name), []byte("foreign\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Cleanup(func() { afterInstallAuthorityRecordBoundary = nil })
	err = publishInstallAuthorityRecord(root, name, ".authority-temp-", "test", []byte("owned\n"), 64)
	if err == nil || !strings.Contains(err.Error(), "without replacement") {
		t.Fatalf("foreign fixed authority collision was accepted: %v", err)
	}
	if err := publishInstallAuthorityRecord(root, name, ".authority-temp-", "test", []byte("owned\n"), 64); err == nil || !strings.Contains(err.Error(), "foreign data") {
		t.Fatalf("foreign fixed authority was accepted on retry: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(rootPath, name)); err != nil || string(got) != "foreign\n" {
		t.Fatalf("foreign fixed authority changed: %q, %v", got, err)
	}
}
