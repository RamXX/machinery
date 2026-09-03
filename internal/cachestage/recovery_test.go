package cachestage

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/fsatomic"
)

func writeStageFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("partial provision\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestReadDirBoundedRejectsEntryOverflowBeforeUnboundedAllocation(t *testing.T) {
	base := t.TempDir()
	for _, name := range []string{"a", "b", "c"} {
		if err := os.WriteFile(filepath.Join(base, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := readDirBounded(root, ".", 2); err == nil || !strings.Contains(err.Error(), "2-entry limit") {
		t.Fatalf("directory overflow diagnostic = %v", err)
	}
}

func TestValidateTreeUsesOneAggregateBudget(t *testing.T) {
	for _, tc := range []struct {
		name       string
		maxEntries int
		maxDepth   int
		maxBytes   int64
		setup      func(*testing.T, string)
		want       string
	}{
		{
			name:       "broad entry overflow",
			maxEntries: 4,
			maxDepth:   8,
			maxBytes:   1024,
			setup: func(t *testing.T, stage string) {
				for _, name := range []string{"a", "b", "c", "d"} {
					if err := os.WriteFile(filepath.Join(stage, name), nil, 0o600); err != nil {
						t.Fatal(err)
					}
				}
			},
			want: "entry limit",
		},
		{
			name:       "deep directory overflow",
			maxEntries: 16,
			maxDepth:   2,
			maxBytes:   1024,
			setup: func(t *testing.T, stage string) {
				if err := os.MkdirAll(filepath.Join(stage, "a", "b", "c"), 0o700); err != nil {
					t.Fatal(err)
				}
			},
			want: "depth limit",
		},
		{
			name:       "aggregate byte overflow",
			maxEntries: 16,
			maxDepth:   8,
			maxBytes:   5,
			setup: func(t *testing.T, stage string) {
				for _, name := range []string{"a", "b"} {
					if err := os.WriteFile(filepath.Join(stage, name), []byte("abc"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			},
			want: "aggregate limit",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := t.TempDir()
			stage := filepath.Join(base, "stage")
			if err := os.Mkdir(stage, 0o700); err != nil {
				t.Fatal(err)
			}
			tc.setup(t, stage)
			root, err := os.OpenRoot(base)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if err := validateTreeBounded(root, "stage", tc.maxEntries, tc.maxDepth, tc.maxBytes); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("aggregate budget diagnostic = %v; want %q", err, tc.want)
			}
		})
	}
}

const cacheRecoveryCrashEnv = "MACHINERY_CACHE_RECOVERY_CRASH_ROOT"
const cacheRecoveryCrashPointEnv = "MACHINERY_CACHE_RECOVERY_CRASH_POINT"

func TestRecoverResumesProcessCrashAtDurabilityBoundaries(t *testing.T) {
	for _, point := range []string{"after-quarantine", "before-private-remove"} {
		t.Run(point, func(t *testing.T) {
			base := t.TempDir()
			writeStageFile(t, filepath.Join(base, ".java-stage-123", "partial"))
			command := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestCacheRecoveryCrashHelper$")
			command.Env = append(os.Environ(), cacheRecoveryCrashEnv+"="+base, cacheRecoveryCrashPointEnv+"="+point)
			err := command.Run()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 86 {
				t.Fatalf("crash helper at %s = %v", point, err)
			}
			if err := Recover(base, ".java-stage-"); err != nil {
				t.Fatalf("restart after %s: %v", point, err)
			}
			if entries, err := os.ReadDir(base); err != nil || len(entries) != 0 {
				t.Fatalf("restart left residue after %s: %v, %v", point, entries, err)
			}
		})
	}
}

func TestCacheRecoveryCrashHelper(t *testing.T) {
	base := os.Getenv(cacheRecoveryCrashEnv)
	if base == "" {
		return
	}
	point := os.Getenv(cacheRecoveryCrashPointEnv)
	hooks := recoveryHooks{
		afterRetire: func(*os.Root, string) error {
			if point == "after-quarantine" {
				os.Exit(86)
			}
			return nil
		},
		beforePrivateTreeRemove: func(*os.Root, string) error {
			if point == "before-private-remove" {
				os.Exit(86)
			}
			return nil
		},
	}
	_ = recoverTrees(base, ".java-stage-", hooks)
	os.Exit(87)
}

func TestRecoverRejectsForeignFixedRetirementResidue(t *testing.T) {
	base := t.TempDir()
	foreign := filepath.Join(base, ".java-stage-123.machinery-retire")
	writeStageFile(t, filepath.Join(foreign, "user"))
	if err := Recover(base, ".java-stage-"); err == nil || !strings.Contains(err.Error(), "unsafe name") {
		t.Fatalf("foreign fixed retirement residue was accepted: %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(foreign, "user")); err != nil || string(body) != "partial provision\n" {
		t.Fatalf("foreign fixed retirement residue changed: %q, %v", body, err)
	}
}

func TestRecoverResumesLegacyPrivateRetirementQuarantine(t *testing.T) {
	base := t.TempDir()
	legacy := ".java-stage-123.machinery-retire"
	writeStageFile(t, filepath.Join(base, legacy, "partial"))
	root, err := os.OpenRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	quarantined, err := fsatomic.Quarantine(root, legacy, ".java-stage-delete-")
	if err != nil {
		_ = root.Close()
		t.Fatal(err)
	}
	if err := errors.Join(quarantined.Close(), root.Close()); err != nil {
		t.Fatal(err)
	}
	if err := Recover(base, ".java-stage-"); err != nil {
		t.Fatalf("resume legacy private retirement quarantine: %v", err)
	}
	if entries, err := os.ReadDir(base); err != nil || len(entries) != 0 {
		t.Fatalf("legacy private retirement quarantine remains: %v, %v", entries, err)
	}
}

func TestRecoverResumesEmptyQuarantineCrashStates(t *testing.T) {
	for _, state := range []string{"before-object-move", "after-object-delete"} {
		t.Run(state, func(t *testing.T) {
			base := t.TempDir()
			stage := ".java-stage-123"
			writeStageFile(t, filepath.Join(base, stage, "partial"))
			root, err := os.OpenRoot(base)
			if err != nil {
				t.Fatal(err)
			}
			quarantined, err := fsatomic.Quarantine(root, stage, ".java-stage-delete-")
			if err != nil {
				_ = root.Close()
				t.Fatal(err)
			}
			if state == "before-object-move" {
				err = fsatomic.RenameNoReplaceBetween(quarantined.Root(), quarantined.Name(), root, stage)
			} else {
				err = quarantined.Root().RemoveAll(quarantined.Name())
			}
			if err != nil {
				_ = quarantined.Close()
				_ = root.Close()
				t.Fatal(err)
			}
			if err := errors.Join(quarantined.Close(), root.Close()); err != nil {
				t.Fatal(err)
			}
			if err := Recover(base, ".java-stage-"); err != nil {
				t.Fatalf("recover %s: %v", state, err)
			}
			if entries, err := os.ReadDir(base); err != nil || len(entries) != 0 {
				t.Fatalf("%s recovery residue: %v, %v", state, entries, err)
			}
		})
	}
}

func TestRecoverFilesValidatesCompleteReservedInventoryBeforeRemoval(t *testing.T) {
	base := t.TempDir()
	valid := filepath.Join(base, ".machinery-jar-123456")
	if err := os.WriteFile(valid, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(base, ".machinery-jar-654321")
	if err := os.Symlink(outside, bad); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := RecoverFiles(base, ".machinery-jar-"); err == nil {
		t.Fatal("unsafe reserved file stage was accepted")
	}
	if _, err := os.Lstat(valid); err != nil {
		t.Fatalf("valid residue was partially removed before full validation: %v", err)
	}
	if err := os.Remove(bad); err != nil {
		t.Fatal(err)
	}
	if err := RecoverFiles(base, ".machinery-jar-"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(valid); !os.IsNotExist(err) {
		t.Fatalf("valid interrupted file stage remains: %v", err)
	}
	if got, err := os.ReadFile(outside); err != nil || string(got) != "sentinel" {
		t.Fatalf("outside sentinel changed: %q, %v", got, err)
	}
}

func TestRecoverRemovesEveryOwnedStageAndPreservesNeighbors(t *testing.T) {
	base := t.TempDir()
	for _, stage := range []string{".java-stage-9", ".java-stage-4294967295"} {
		writeStageFile(t, filepath.Join(base, stage, "extracted", "lib", "large"))
	}
	neighbor := filepath.Join(base, "21.0.12.1_1", "runtime")
	writeStageFile(t, neighbor)
	if err := Recover(base, ".java-stage-"); err != nil {
		t.Fatal(err)
	}
	for _, stage := range []string{".java-stage-9", ".java-stage-4294967295"} {
		if _, err := os.Lstat(filepath.Join(base, stage)); !os.IsNotExist(err) {
			t.Errorf("interrupted stage %s survived durable recovery: %v", stage, err)
		}
	}
	if body, err := os.ReadFile(neighbor); err != nil || string(body) != "partial provision\n" {
		t.Fatalf("non-stage cache entry changed: %q, %v", body, err)
	}
}

func TestRecoverPreflightsReservedInventoryDeterministically(t *testing.T) {
	base := t.TempDir()
	valid := filepath.Join(base, ".java-stage-2")
	malformed := filepath.Join(base, ".java-stage-not-random")
	writeStageFile(t, filepath.Join(valid, "partial"))
	writeStageFile(t, filepath.Join(malformed, "partial"))
	var first string
	for i := 0; i < 100; i++ {
		err := Recover(base, ".java-stage-")
		if err == nil || !strings.Contains(err.Error(), `.java-stage-not-random`) || !strings.Contains(err.Error(), "unsafe name") {
			t.Fatalf("run %d malformed reserved stage error = %v", i, err)
		}
		if i == 0 {
			first = err.Error()
		} else if err.Error() != first {
			t.Fatalf("reserved-stage diagnostic changed: first=%q run-%d=%q", first, i, err)
		}
		if _, err := os.Lstat(valid); err != nil {
			t.Fatalf("valid stage was partially removed before unsafe inventory entry was rejected: %v", err)
		}
	}
}

func TestRecoverRejectsSymlinkAndNonDirectoryResidue(t *testing.T) {
	t.Run("nested symlink", func(t *testing.T) {
		base := t.TempDir()
		stage := filepath.Join(base, ".structurizr-stage-123")
		if err := os.Mkdir(stage, 0o700); err != nil {
			t.Fatal(err)
		}
		outside := filepath.Join(t.TempDir(), "sentinel")
		if err := os.WriteFile(outside, []byte("outside\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(stage, "escape")); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if err := Recover(base, ".structurizr-stage-"); err == nil || !strings.Contains(err.Error(), "is a symlink") {
			t.Fatalf("symlinked stage residue accepted: %v", err)
		}
		if body, err := os.ReadFile(outside); err != nil || string(body) != "outside\n" {
			t.Fatalf("outside sentinel changed: %q, %v", body, err)
		}
	})
	t.Run("reserved regular file", func(t *testing.T) {
		base := t.TempDir()
		if err := os.WriteFile(filepath.Join(base, ".java-stage-123"), []byte("not a directory"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := Recover(base, ".java-stage-"); err == nil || !strings.Contains(err.Error(), "private real directory") {
			t.Fatalf("non-directory stage residue accepted: %v", err)
		}
	})
}

func TestRecoverTreePreservesLateMutationAfterValidation(t *testing.T) {
	for _, point := range []string{"before quarantine", "after quarantine", "deletion boundary", "same-content root ABA"} {
		t.Run(point, func(t *testing.T) {
			base := t.TempDir()
			stage := ".java-stage-123"
			stagePath := filepath.Join(base, stage)
			partial := filepath.Join(stagePath, "extracted", "partial")
			writeStageFile(t, partial)
			mutated := false
			var privateRoot *os.Root
			hooks := recoveryHooks{
				beforeRetire: func(name string) error {
					if name != stage || mutated || (point != "before quarantine" && point != "same-content root ABA") {
						return nil
					}
					mutated = true
					if point == "same-content root ABA" {
						if err := os.Rename(stagePath, filepath.Join(base, "user-preserved-original")); err != nil {
							return err
						}
						writeStageFile(t, partial)
						return nil
					}
					return os.WriteFile(partial, []byte("late user bytes\n"), 0o600)
				},
				afterRetire: func(root *os.Root, name string) error {
					privateRoot = root
					if point != "after quarantine" || mutated {
						return nil
					}
					mutated = true
					return root.WriteFile(filepath.Join(name, "late-user-file"), []byte("late user bytes\n"), 0o600)
				},
				beforeRemove: func(name string) error {
					if point != "deletion boundary" || filepath.ToSlash(name) != "object/extracted/partial" || mutated {
						return nil
					}
					mutated = true
					return privateRoot.WriteFile(name, []byte("late user bytes\n"), 0o600)
				},
			}
			err := recoverTrees(base, ".java-stage-", hooks)
			if err == nil || !strings.Contains(err.Error(), "preserv") {
				t.Fatalf("late tree mutation at %s was deleted: %v", point, err)
			}
			entries, err := os.ReadDir(base)
			if err != nil {
				t.Fatal(err)
			}
			var quarantineName string
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".java-stage-delete-") {
					quarantineName = entry.Name()
				}
			}
			if quarantineName == "" {
				t.Fatalf("private quarantine was not preserved at %s: %v", point, entries)
			}
			root, err := os.OpenRoot(base)
			if err != nil {
				t.Fatal(err)
			}
			quarantined, err := fsatomic.ResumeQuarantine(root, quarantineName, stage)
			if err != nil {
				_ = root.Close()
				t.Fatal(err)
			}
			switch point {
			case "after quarantine":
				if body, err := quarantined.Root().ReadFile(filepath.Join(quarantined.Name(), "late-user-file")); err != nil || string(body) != "late user bytes\n" {
					t.Fatalf("late quarantine population changed: %q, %v", body, err)
				}
			case "same-content root ABA":
				if _, err := os.Lstat(filepath.Join(base, "user-preserved-original")); err != nil {
					t.Fatalf("original tree was not preserved: %v", err)
				}
			default:
				if body, err := quarantined.Root().ReadFile(filepath.Join(quarantined.Name(), "extracted", "partial")); err != nil || string(body) != "late user bytes\n" {
					t.Fatalf("late tree bytes changed: %q, %v", body, err)
				}
			}
			if err := errors.Join(quarantined.Close(), root.Close()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRecoverFilesPreservesLateContentAndPathABA(t *testing.T) {
	for _, mutation := range []string{"content", "same-content path ABA"} {
		t.Run(mutation, func(t *testing.T) {
			base := t.TempDir()
			name := ".machinery-jar-123456"
			stage := filepath.Join(base, name)
			if err := os.WriteFile(stage, []byte("partial"), 0o600); err != nil {
				t.Fatal(err)
			}
			mutated := false
			err := recoverFiles(base, ".machinery-jar-", recoveryHooks{beforeRemove: func(path string) error {
				if path != name || mutated {
					return nil
				}
				mutated = true
				if mutation == "content" {
					return os.WriteFile(stage, []byte("user content"), 0o600)
				}
				if err := os.Rename(stage, filepath.Join(base, "user-preserved-original")); err != nil {
					return err
				}
				return os.WriteFile(stage, []byte("partial"), 0o600)
			}})
			if err == nil || !strings.Contains(err.Error(), "preserving") {
				t.Fatalf("late file %s was deleted: %v", mutation, err)
			}
			want := "user content"
			if mutation == "same-content path ABA" {
				want = "partial"
				if _, err := os.Lstat(filepath.Join(base, "user-preserved-original")); err != nil {
					t.Fatalf("original file was not preserved: %v", err)
				}
			}
			if body, err := os.ReadFile(stage); err != nil || string(body) != want {
				t.Fatalf("late file %s bytes changed: %q, %v", mutation, body, err)
			}
		})
	}
}

func TestRecoverResumesCrashAfterPrivateQuarantine(t *testing.T) {
	base := t.TempDir()
	writeStageFile(t, filepath.Join(base, ".java-stage-123", "partial"))
	crash := errors.New("injected quarantine crash")
	err := recoverTrees(base, ".java-stage-", recoveryHooks{beforePrivateRemove: func(*os.Root, string) error { return crash }})
	if !errors.Is(err, crash) {
		t.Fatalf("quarantine crash = %v", err)
	}
	if err := Recover(base, ".java-stage-"); err != nil {
		t.Fatalf("resume quarantined recovery: %v", err)
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("recovery left quarantine residue: %v", entries)
	}
}

func TestRecoverPreservesPostCheckPrivateReplacement(t *testing.T) {
	base := t.TempDir()
	writeStageFile(t, filepath.Join(base, ".java-stage-123", "partial"))
	err := recoverTrees(base, ".java-stage-", recoveryHooks{beforePrivateRemove: func(root *os.Root, name string) error {
		if err := root.Rename(name, name+"-original"); err != nil {
			return err
		}
		if err := root.Mkdir(name, 0o700); err != nil {
			return err
		}
		return root.WriteFile(filepath.Join(name, "user"), []byte("preserve"), 0o600)
	}})
	if err == nil || (!strings.Contains(err.Error(), "unexpected inventory") && !strings.Contains(err.Error(), "entry limit")) {
		t.Fatalf("private replacement diagnostic = %v", err)
	}
	entries, readErr := os.ReadDir(base)
	if readErr != nil || len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), ".java-stage-delete-") {
		t.Fatalf("private replacement authority was not preserved: %v, %v", entries, readErr)
	}
}

func TestRecoverPreservesPrivateTreeReplacementAtRemovalBoundary(t *testing.T) {
	base := t.TempDir()
	writeStageFile(t, filepath.Join(base, ".java-stage-123", "partial"))
	err := recoverTrees(base, ".java-stage-", recoveryHooks{beforePrivateTreeRemove: func(root *os.Root, name string) error {
		if err := root.Rename(name, name+"-original"); err != nil {
			return err
		}
		if err := root.Mkdir(name, 0o700); err != nil {
			return err
		}
		return root.WriteFile(filepath.Join(name, "user"), []byte("preserve"), 0o600)
	}})
	if err == nil || !strings.Contains(err.Error(), "removal boundary") {
		t.Fatalf("private removal-boundary replacement diagnostic = %v", err)
	}
	entries, readErr := os.ReadDir(base)
	if readErr != nil || len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), ".java-stage-delete-") {
		t.Fatalf("private removal-boundary evidence was not preserved: %v, %v", entries, readErr)
	}
}

func TestRecoverQuarantineBoundaryPreservesSourceABA(t *testing.T) {
	base := t.TempDir()
	stage := ".java-stage-123"
	writeStageFile(t, filepath.Join(base, stage, "partial"))
	parked := "parked-original"
	err := recoverTrees(base, ".java-stage-", recoveryHooks{quarantine: func(root *os.Root, source, prefix string) (*fsatomic.Quarantined, error) {
		if err := root.Rename(source, parked); err != nil {
			return nil, err
		}
		if err := root.Mkdir(source, 0o700); err != nil {
			return nil, err
		}
		if err := root.WriteFile(filepath.Join(source, "user"), []byte("preserve"), 0o600); err != nil {
			return nil, err
		}
		return fsatomic.Quarantine(root, source, prefix)
	}})
	if err == nil {
		t.Fatal("cache recovery accepted a source ABA at the quarantine boundary")
	}
	if body, err := os.ReadFile(filepath.Join(base, parked, "partial")); err != nil || string(body) != "partial provision\n" {
		t.Fatalf("original stage changed: %q, %v", body, err)
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	preservedReplacement := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".java-stage-delete-") {
			preservedReplacement = true
		}
	}
	if !preservedReplacement {
		t.Fatal("replacement quarantine was not preserved")
	}
}

func TestRecoverFilesResumesPrivateQuarantineCrash(t *testing.T) {
	base := t.TempDir()
	stage := filepath.Join(base, ".machinery-jar-123")
	if err := os.WriteFile(stage, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	crash := errors.New("file quarantine crash")
	err := recoverFiles(base, ".machinery-jar-", recoveryHooks{beforePrivateRemove: func(*os.Root, string) error { return crash }})
	if !errors.Is(err, crash) {
		t.Fatalf("file quarantine crash = %v", err)
	}
	if err := RecoverFiles(base, ".machinery-jar-"); err != nil {
		t.Fatalf("resume file quarantine: %v", err)
	}
	entries, err := os.ReadDir(base)
	if err != nil || len(entries) != 0 {
		t.Fatalf("file quarantine residue: %v, %v", entries, err)
	}
}
