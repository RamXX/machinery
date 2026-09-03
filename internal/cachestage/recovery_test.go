package cachestage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	for _, point := range []string{"before retirement", "after retirement", "deletion boundary", "same-content root ABA"} {
		t.Run(point, func(t *testing.T) {
			base := t.TempDir()
			stage := ".java-stage-123"
			stagePath := filepath.Join(base, stage)
			partial := filepath.Join(stagePath, "extracted", "partial")
			writeStageFile(t, partial)
			retirement := stage + ".machinery-retire"
			retiredPartial := filepath.Join(retirement, "extracted", "partial")
			mutated := false
			hooks := recoveryHooks{
				beforeRetire: func(name string) error {
					if name != stage || mutated || (point != "before retirement" && point != "same-content root ABA") {
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
				afterRetire: func(name string) error {
					if point != "after retirement" || name != retirement || mutated {
						return nil
					}
					mutated = true
					return os.WriteFile(filepath.Join(base, name, "late-user-file"), []byte("late user bytes\n"), 0o600)
				},
				beforeRemove: func(name string) error {
					if point != "deletion boundary" || name != retiredPartial || mutated {
						return nil
					}
					mutated = true
					return os.WriteFile(filepath.Join(base, name), []byte("late user bytes\n"), 0o600)
				},
			}
			err := recoverTrees(base, ".java-stage-", hooks)
			if err == nil || !strings.Contains(err.Error(), "preserv") {
				t.Fatalf("late tree mutation at %s was deleted: %v", point, err)
			}
			if _, err := os.Lstat(filepath.Join(base, retirement)); err != nil {
				t.Fatalf("retirement tree was not preserved at %s: %v", point, err)
			}
			switch point {
			case "after retirement":
				if body, err := os.ReadFile(filepath.Join(base, retirement, "late-user-file")); err != nil || string(body) != "late user bytes\n" {
					t.Fatalf("late retirement population changed: %q, %v", body, err)
				}
			case "same-content root ABA":
				if _, err := os.Lstat(filepath.Join(base, "user-preserved-original")); err != nil {
					t.Fatalf("original tree was not preserved: %v", err)
				}
			default:
				if body, err := os.ReadFile(filepath.Join(base, retiredPartial)); err != nil || string(body) != "late user bytes\n" {
					t.Fatalf("late tree bytes changed: %q, %v", body, err)
				}
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
