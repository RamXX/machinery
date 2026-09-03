package cachestage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func seedPublicationTree(t *testing.T, base string) (source, target string) {
	t.Helper()
	source = filepath.Join(".java-stage-123", "extracted")
	target = filepath.Join("21.0.12.1_1", "test-platform")
	writeStageFile(t, filepath.Join(base, source, "runtime", "lib", "modules"))
	writeStageFile(t, filepath.Join(base, source, "runtime", ".machinery-java-receipt"))
	return source, target
}

func TestPublishCrashBeforeRenameLeavesRecoverableStage(t *testing.T) {
	base := t.TempDir()
	source, target := seedPublicationTree(t, base)
	crash := errors.New("injected crash after durable tree sync")
	err := publish(base, source, target, publishHooks{afterTreeSync: func() error { return crash }})
	if !errors.Is(err, crash) {
		t.Fatalf("pre-rename crash was hidden: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(base, source)); err != nil {
		t.Fatalf("pre-rename crash lost the recoverable stage: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(base, target)); !os.IsNotExist(err) {
		t.Fatalf("pre-rename crash exposed a target: %v", err)
	}
	if err := Recover(base, ".java-stage-"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(base, ".java-stage-123")); !os.IsNotExist(err) {
		t.Fatalf("retry did not recover the pre-rename stage: %v", err)
	}
}

func TestPublishCrashAfterRenameExposesOnlyCompleteSyncedTree(t *testing.T) {
	base := t.TempDir()
	source, target := seedPublicationTree(t, base)
	crash := errors.New("injected crash after rename")
	err := publish(base, source, target, publishHooks{afterRename: func() error { return crash }})
	if !errors.Is(err, crash) {
		t.Fatalf("post-rename crash was hidden: %v", err)
	}
	for _, rel := range []string{
		filepath.Join("runtime", "lib", "modules"),
		filepath.Join("runtime", ".machinery-java-receipt"),
	} {
		body, err := os.ReadFile(filepath.Join(base, target, rel))
		if err != nil || string(body) != "partial provision\n" {
			t.Fatalf("post-rename target %s is partial: %q, %v", rel, body, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(base, source)); !os.IsNotExist(err) {
		t.Fatalf("post-rename source still exists: %v", err)
	}
	if err := Recover(base, ".java-stage-"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(base, target)); err != nil {
		t.Fatalf("retry recovery removed the complete published target: %v", err)
	}
}

func TestPublishNeverReplacesPartialOrExistingTarget(t *testing.T) {
	base := t.TempDir()
	source, target := seedPublicationTree(t, base)
	partial := filepath.Join(base, target)
	if err := os.MkdirAll(partial, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(partial, "partial"), []byte("untrusted"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := PublishTree(base, source, target); err == nil {
		t.Fatal("publisher replaced a pre-existing partial target")
	}
	if body, err := os.ReadFile(filepath.Join(partial, "partial")); err != nil || string(body) != "untrusted" {
		t.Fatalf("partial target changed: %q, %v", body, err)
	}
	if _, err := os.Lstat(filepath.Join(base, source)); err != nil {
		t.Fatalf("refused publication mutated its complete source: %v", err)
	}
}

func TestPublishDurablyInstallsTree(t *testing.T) {
	base := t.TempDir()
	source, target := seedPublicationTree(t, base)
	if err := PublishTree(base, source, target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(base, source)); !os.IsNotExist(err) {
		t.Fatalf("successful publication retained source: %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(base, target, "runtime", "lib", "modules")); err != nil || string(body) != "partial provision\n" {
		t.Fatalf("published file = %q, %v", body, err)
	}
}

func TestPublishRejectsAfterTreeSyncMutationAndRecovers(t *testing.T) {
	for _, mutation := range []string{"add", "change", "remove", "aba"} {
		t.Run(mutation, func(t *testing.T) {
			base := t.TempDir()
			source, target := seedPublicationTree(t, base)
			modules := filepath.Join(base, source, "runtime", "lib", "modules")
			receipt := filepath.Join(base, source, "runtime", ".machinery-java-receipt")
			added := filepath.Join(base, source, "runtime", "lib", "added")
			var replacementInfo os.FileInfo

			err := publish(base, source, target, publishHooks{afterTreeSync: func() error {
				switch mutation {
				case "add":
					writeStageFile(t, added)
				case "change":
					if err := os.WriteFile(modules, []byte("foreign mutation\n"), 0o600); err != nil {
						return err
					}
				case "remove":
					if err := os.Remove(receipt); err != nil {
						return err
					}
				case "aba":
					replacement := filepath.Join(base, source, "runtime", "lib", "replacement")
					if err := os.WriteFile(replacement, []byte("partial provision\n"), 0o600); err != nil {
						return err
					}
					var err error
					replacementInfo, err = os.Lstat(replacement)
					if err != nil {
						return err
					}
					if err := os.Remove(modules); err != nil {
						return err
					}
					if err := os.Rename(replacement, modules); err != nil {
						return err
					}
				}
				return nil
			}})
			if err == nil || !strings.Contains(err.Error(), "changed after durable sync") {
				t.Fatalf("after-sync %s mutation was published: %v", mutation, err)
			}
			if _, err := os.Lstat(filepath.Join(base, source)); err != nil {
				t.Fatalf("rejected %s mutation lost its recoverable source: %v", mutation, err)
			}
			if _, err := os.Lstat(filepath.Join(base, target)); !os.IsNotExist(err) {
				t.Fatalf("rejected %s mutation exposed a cache target: %v", mutation, err)
			}
			switch mutation {
			case "add":
				if _, err := os.Lstat(added); err != nil {
					t.Fatalf("added source entry was not preserved: %v", err)
				}
			case "change":
				if body, err := os.ReadFile(modules); err != nil || string(body) != "foreign mutation\n" {
					t.Fatalf("changed source content was not preserved: %q, %v", body, err)
				}
			case "remove":
				if _, err := os.Lstat(receipt); !os.IsNotExist(err) {
					t.Fatalf("removed source entry was recreated: %v", err)
				}
			case "aba":
				current, err := os.Lstat(modules)
				if err != nil || replacementInfo == nil || !os.SameFile(replacementInfo, current) {
					t.Fatalf("same-content replacement identity was not preserved: current=%v err=%v", current, err)
				}
			}

			if err := Recover(base, ".java-stage-"); err != nil {
				t.Fatalf("recover rejected %s mutation: %v", mutation, err)
			}
			if _, err := os.Lstat(filepath.Join(base, ".java-stage-123")); !os.IsNotExist(err) {
				t.Fatalf("recover retained rejected %s stage: %v", mutation, err)
			}
			retrySource, retryTarget := seedPublicationTree(t, base)
			if retrySource != source || retryTarget != target {
				t.Fatalf("retry paths changed: (%s, %s), want (%s, %s)", retrySource, retryTarget, source, target)
			}
			if err := PublishTree(base, retrySource, retryTarget); err != nil {
				t.Fatalf("automatic retry after rejected %s mutation failed: %v", mutation, err)
			}
			if body, err := os.ReadFile(filepath.Join(base, target, "runtime", "lib", "modules")); err != nil || string(body) != "partial provision\n" {
				t.Fatalf("retry after rejected %s mutation published %q, %v", mutation, body, err)
			}
		})
	}
}

func TestPublishRejectsFinalSourceMutationAfterRevalidation(t *testing.T) {
	for _, mutation := range []string{"change", "same-content root ABA"} {
		t.Run(mutation, func(t *testing.T) {
			base := t.TempDir()
			source, target := seedPublicationTree(t, base)
			sourcePath := filepath.Join(base, source)
			modules := filepath.Join(sourcePath, "runtime", "lib", "modules")
			parked := filepath.Join(base, "user-preserved-source")
			err := publish(base, source, target, publishHooks{beforeRename: func() error {
				switch mutation {
				case "change":
					return os.WriteFile(modules, []byte("late foreign mutation\n"), 0o600)
				case "same-content root ABA":
					if err := os.Rename(sourcePath, parked); err != nil {
						return err
					}
					writeStageFile(t, filepath.Join(sourcePath, "runtime", "lib", "modules"))
					writeStageFile(t, filepath.Join(sourcePath, "runtime", ".machinery-java-receipt"))
				}
				return nil
			}})
			if err == nil || !strings.Contains(err.Error(), "does not match the exact synced source") {
				t.Fatalf("late source %s was accepted: %v", mutation, err)
			}
			if _, err := os.Lstat(filepath.Join(base, source)); !os.IsNotExist(err) {
				t.Fatalf("late source %s unexpectedly remained at the stage name: %v", mutation, err)
			}
			publishedModules := filepath.Join(base, target, "runtime", "lib", "modules")
			if mutation == "change" {
				if body, err := os.ReadFile(publishedModules); err != nil || string(body) != "late foreign mutation\n" {
					t.Fatalf("late foreign bytes were not preserved: %q, %v", body, err)
				}
			} else {
				if body, err := os.ReadFile(publishedModules); err != nil || string(body) != "partial provision\n" {
					t.Fatalf("same-content replacement was not preserved: %q, %v", body, err)
				}
				if _, err := os.Lstat(parked); err != nil {
					t.Fatalf("original witnessed source was not preserved: %v", err)
				}
			}
		})
	}
}

func TestPublishRejectsPostRenamePopulationAndPreservesIt(t *testing.T) {
	base := t.TempDir()
	source, target := seedPublicationTree(t, base)
	added := filepath.Join(base, target, "runtime", "late-user-file")
	err := publish(base, source, target, publishHooks{afterRename: func() error {
		return os.WriteFile(added, []byte("preserve me\n"), 0o600)
	}})
	if err == nil || !strings.Contains(err.Error(), "does not match the exact synced source") {
		t.Fatalf("post-rename population was accepted: %v", err)
	}
	if body, err := os.ReadFile(added); err != nil || string(body) != "preserve me\n" {
		t.Fatalf("post-rename population was not preserved: %q, %v", body, err)
	}
}

func TestPublishRechecksTargetAbsenceAtRenameBoundary(t *testing.T) {
	base := t.TempDir()
	source, target := seedPublicationTree(t, base)
	sentinel := filepath.Join(base, target, "user-sentinel")
	err := publish(base, source, target, publishHooks{beforeRename: func() error {
		if err := os.MkdirAll(filepath.Dir(sentinel), 0o700); err != nil {
			return err
		}
		return os.WriteFile(sentinel, []byte("preserve me\n"), 0o600)
	}})
	if err == nil || !strings.Contains(err.Error(), "appeared at the publish boundary") {
		t.Fatalf("late target was replaced: %v", err)
	}
	if body, err := os.ReadFile(sentinel); err != nil || string(body) != "preserve me\n" {
		t.Fatalf("late target bytes changed: %q, %v", body, err)
	}
	if _, err := os.Lstat(filepath.Join(base, source)); err != nil {
		t.Fatalf("source was moved despite late target: %v", err)
	}
}
