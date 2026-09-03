package designlock

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/fsatomic"
)

func TestCheckUnchangedRejectsNonCooperativeMutation(t *testing.T) {
	design := t.TempDir()
	path := filepath.Join(design, "source.yaml")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if err := os.WriteFile(path, []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := lock.CheckUnchanged(); err == nil || !strings.Contains(err.Error(), "source.yaml") {
		t.Fatalf("CheckUnchanged error = %v, want changed source path", err)
	}
}

func TestUniversalSnapshotBoundaryRejectsNonportableInventory(t *testing.T) {
	for _, acquire := range []struct {
		name string
		fn   func(string) (*Lock, error)
	}{{"writer", Acquire}, {"reader", AcquireReader}} {
		t.Run(acquire.name, func(t *testing.T) {
			for _, name := range []string{"CON.yaml", "naïve.yaml", "trailing."} {
				t.Run(name, func(t *testing.T) {
					design := t.TempDir()
					if err := os.WriteFile(filepath.Join(design, name), []byte("source\n"), 0o644); err != nil {
						t.Skipf("filesystem cannot create adversarial name %q: %v", name, err)
					}
					if lock, err := acquire.fn(design); err == nil {
						_ = lock.Release()
						t.Fatalf("accepted nonportable design entry %q", name)
					} else if !strings.Contains(err.Error(), "non-portable design path") {
						t.Fatalf("error = %v, want portable inventory diagnostic", err)
					}
				})
			}
		})
	}
}

func TestUniversalSnapshotBoundaryRejectsCaseFoldAliasesDeterministically(t *testing.T) {
	for i := 0; i < 20; i++ {
		seen := map[string]string{}
		if err := validateInventoryPath("Machines/Thing.machine.json", seen); err != nil {
			t.Fatal(err)
		}
		err := validateInventoryPath("machines/thing.machine.json", seen)
		if err == nil || err.Error() != "portable design-path collision: design inventory entries Machines/Thing.machine.json and machines/thing.machine.json alias on case-insensitive filesystems" {
			t.Fatalf("iteration %d: error = %v", i, err)
		}
	}
}

func TestAcquireWriterRejectsABADuringImmutableMaterialization(t *testing.T) {
	design := t.TempDir()
	path := filepath.Join(design, "source.yaml")
	if err := os.WriteFile(path, []byte("A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	priorBefore, priorAfter := designSourceBeforeOpen, designSourceAfterRead
	designSourceBeforeOpen = func(rel string) {
		if rel == "source.yaml" {
			if err := os.WriteFile(path, []byte("B\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	designSourceAfterRead = func(rel string) {
		if rel == "source.yaml" {
			if err := os.WriteFile(path, []byte("A\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	defer func() {
		designSourceBeforeOpen, designSourceAfterRead = priorBefore, priorAfter
	}()
	if lock, err := Acquire(design); err == nil {
		_ = lock.Release()
		t.Fatal("A→B→A materialization was accepted")
	} else if !strings.Contains(err.Error(), "ABA-derived") {
		t.Fatalf("Acquire error = %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != "A\n" {
		t.Fatalf("test did not restore A: %q, %v", body, err)
	}
}

func TestImmutableSourceNeverExposesPostAcquireABA(t *testing.T) {
	design := t.TempDir()
	path := filepath.Join(design, "source.yaml")
	if err := os.WriteFile(path, []byte("A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	sourcePath, err := lock.SourcePath(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("B\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(sourcePath)
	if err != nil || string(body) != "A\n" {
		t.Fatalf("immutable source exposed transient B: %q, %v", body, err)
	}
	if err := os.WriteFile(path, []byte("A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := lock.CheckUnchanged(); err != nil {
		t.Fatalf("restored live tree should match acquisition fingerprint: %v", err)
	}
	body, err = os.ReadFile(sourcePath)
	if err != nil || string(body) != "A\n" {
		t.Fatalf("immutable source changed after live A→B→A: %q, %v", body, err)
	}
}

func TestRefreshAtomicallyRematerializesDeterministicSetup(t *testing.T) {
	design := t.TempDir()
	write := filepath.Join(design, "machines", "Thing.machine.json")
	if err := os.MkdirAll(filepath.Dir(write), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(write, []byte("machine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	prior := lock.SourceRoot()
	created := filepath.Join(design, "formal", "setup.txt")
	if err := os.MkdirAll(filepath.Dir(created), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(created, []byte("created before governed reads\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := lock.Refresh(); err != nil {
		t.Fatal(err)
	}
	if lock.SourceRoot() == prior {
		t.Fatal("Refresh retained the stale immutable source tree")
	}
	if _, err := os.Stat(prior); !os.IsNotExist(err) {
		t.Fatalf("prior immutable source still exists: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(lock.SourceRoot(), "formal", "setup.txt"))
	if err != nil || string(body) != "created before governed reads\n" {
		t.Fatalf("refreshed immutable source omitted setup: %q, %v", body, err)
	}
}

func TestPublishExactRetryIgnoresOnlyOwnedRecoveryResidue(t *testing.T) {
	for _, tc := range []struct {
		name, residue, output string
	}{
		{"artifact journal", ".machinery-artifact-set.journal", "generated.txt"},
		{"artifact stage", ".machinery-artifact-new-0123456789abcdef0123456789abcdef", "generated.txt"},
		{"nested artifact journal", "generated/deep/.machinery-artifact-set.journal", "generated/deep/result.txt"},
		{"nested artifact stage", "generated/deep/.machinery-artifact-new-0123456789abcdef0123456789abcdef", "generated/deep/result.txt"},
		{"formal journal", "formal/.machinery-formal-transaction.jsonl", "generated.txt"},
		{"formal stage", "formal/.machinery-formal-stage-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "generated.txt"},
		{"checker journal", ".machinery/checker-project-transaction.json", "generated.txt"},
		{"checker stage", "checkers/test/.machinery-project-stage-0123456789abcdef", "generated.txt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			source := filepath.Join(design, "source.yaml")
			if err := os.WriteFile(source, []byte("A\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			residue := filepath.Join(design, filepath.FromSlash(tc.residue))
			if err := os.MkdirAll(filepath.Dir(residue), 0o755); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(design, filepath.FromSlash(tc.output))
			if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
				t.Fatal(err)
			}
			lock, err := Acquire(design)
			if err != nil {
				t.Fatal(err)
			}
			err = lock.Publish("test-writer", "rerun test writer", []string{output}, func() error {
				if err := os.WriteFile(residue, []byte("owned crash residue\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				return os.ErrInvalid
			})
			if err == nil {
				t.Fatal("simulated crash succeeded")
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}

			lock, err = Acquire(design)
			if err != nil {
				t.Fatal(err)
			}
			called := false
			if err := lock.Publish("different-writer", "rerun different", []string{output}, func() error {
				called = true
				return nil
			}); err == nil || called {
				t.Fatalf("different operation crossed prior sentinel: err=%v called=%v", err, called)
			}
			if err := lock.Publish("test-writer", "rerun test writer", []string{output}, func() error {
				if err := os.Remove(residue); err != nil {
					return err
				}
				return os.WriteFile(output, []byte("recovered\n"), 0o644)
			}); err != nil {
				t.Fatalf("exact retry could not enter recovery: %v", err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPublishDoesNotNormalizeLookalikeRecoveryNames(t *testing.T) {
	design := t.TempDir()
	source := filepath.Join(design, "source.yaml")
	if err := os.WriteFile(source, []byte("A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(design, "generated.txt")
	if err := lock.Publish("test-writer", "rerun test writer", []string{output}, func() error {
		return os.ErrInvalid
	}); err == nil {
		t.Fatal("simulated failed publication succeeded")
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	lookalike := filepath.Join(design, ".machinery-artifact-new-not-owned")
	if err := os.WriteFile(lookalike, []byte("foreign\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err = Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	called := false
	err = lock.Publish("test-writer", "rerun test writer", []string{output}, func() error {
		called = true
		return nil
	})
	if err == nil || called {
		t.Fatalf("foreign lookalike was normalized out of input identity: err=%v called=%v", err, called)
	}
}

func TestPublishDoesNotNormalizeValidArtifactResidueOutsideOutputNamespace(t *testing.T) {
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "source.yaml"), []byte("A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(design, "generated", "result.txt")
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.Publish("test-writer", "rerun test writer", []string{output}, func() error { return os.ErrInvalid }); err == nil {
		t.Fatal("simulated failed publication succeeded")
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	foreignDir := filepath.Join(design, "manual")
	if err := os.Mkdir(foreignDir, 0o755); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(foreignDir, ".machinery-artifact-new-0123456789abcdef0123456789abcdef")
	if err := os.WriteFile(foreign, []byte("foreign\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err = Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	called := false
	err = lock.Publish("test-writer", "rerun test writer", []string{output}, func() error {
		called = true
		return nil
	})
	if err == nil || called {
		t.Fatalf("valid-looking residue outside the declared output namespace was normalized: err=%v called=%v", err, called)
	}
}

func TestPublishDoesNotHideUnjournaledFormalStageLookalike(t *testing.T) {
	design := t.TempDir()
	lookalike := filepath.Join(design, ".machinery-formal-stage-orphan")
	if err := os.Mkdir(lookalike, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lookalike, "transient.tla"), []byte("B\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if _, err := os.Stat(filepath.Join(lock.SourceRoot(), ".machinery-formal-stage-orphan", "transient.tla")); err != nil {
		t.Fatalf("unjournaled formal scratch was hidden from the governed snapshot: %v", err)
	}
	if err := os.WriteFile(filepath.Join(lookalike, "transient.tla"), []byte("A\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := lock.CheckUnchanged(); err == nil || !strings.Contains(err.Error(), ".machinery-formal-stage-orphan") {
		t.Fatalf("unjournaled formal scratch mutation was hidden: %v", err)
	}
}

func TestPublishExpectedRejectsPostCallbackOutputTamperingAndExactRetryRecovers(t *testing.T) {
	for _, tc := range []struct {
		name     string
		initial  []byte
		expect   func(string) OutputExpectation
		publish  func(string) error
		tamper   func(string) error
		recover  func(string) error
		modeOnly bool
	}{
		{
			name: "content swap", expect: func(path string) OutputExpectation { return ExpectFile(path, []byte("new\n"), 0o644) },
			publish: func(path string) error { return os.WriteFile(path, []byte("new\n"), 0o644) },
			tamper:  func(path string) error { return os.WriteFile(path, []byte("evil\n"), 0o644) },
			recover: func(path string) error { return os.WriteFile(path, []byte("new\n"), 0o644) },
		},
		{
			name: "unexpected deletion", expect: func(path string) OutputExpectation { return ExpectFile(path, []byte("new\n"), 0o644) },
			publish: func(path string) error { return os.WriteFile(path, []byte("new\n"), 0o644) },
			tamper:  os.Remove,
			recover: func(path string) error { return os.WriteFile(path, []byte("new\n"), 0o644) },
		},
		{
			name: "unexpected addition", initial: []byte("old\n"), expect: ExpectAbsent,
			publish: os.Remove,
			tamper:  func(path string) error { return os.WriteFile(path, []byte("evil\n"), 0o644) },
			recover: func(path string) error {
				if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
					return err
				}
				return nil
			},
		},
		{
			name: "mode swap", expect: func(path string) OutputExpectation { return ExpectFile(path, []byte("new\n"), 0o644) },
			publish: func(path string) error { return os.WriteFile(path, []byte("new\n"), 0o644) },
			tamper:  func(path string) error { return os.Chmod(path, 0o600) },
			recover: func(path string) error {
				return errors.Join(os.WriteFile(path, []byte("new\n"), 0o644), os.Chmod(path, 0o644))
			},
			modeOnly: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.modeOnly && runtime.GOOS == "windows" {
				t.Skip("permission modes are not stable output identity on Windows")
			}
			design := t.TempDir()
			output := filepath.Join(design, "generated.txt")
			if tc.initial != nil {
				if err := os.WriteFile(output, tc.initial, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			lock, err := Acquire(design)
			if err != nil {
				t.Fatal(err)
			}
			expected := []OutputExpectation{tc.expect(output)}
			testAfterPublishCallback = func() {
				if err := tc.tamper(output); err != nil {
					t.Fatal(err)
				}
			}
			err = lock.PublishExpected("expected-writer", "rerun expected writer", expected, func() error { return tc.publish(output) })
			testAfterPublishCallback = nil
			if err == nil || !strings.Contains(err.Error(), "predeclared identity") {
				t.Fatalf("post-callback tampering was accepted: %v", err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
			lock, err = Acquire(design)
			if err != nil {
				t.Fatal(err)
			}
			if err := lock.PublishExpected("expected-writer", "rerun expected writer", expected, func() error { return tc.recover(output) }); err != nil {
				t.Fatalf("exact output-bound retry failed: %v", err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPublishExpectedTreeRejectsUnexpectedInventory(t *testing.T) {
	design := t.TempDir()
	tree := filepath.Join(design, "packs")
	files := map[string][]byte{"orders.pack/pack.yaml": []byte("id: orders\n")}
	expected, err := ExpectTree(tree, files, 0o755, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	writeExpected := func() error {
		if err := os.RemoveAll(tree); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(tree, "orders.pack"), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(tree, "orders.pack", "pack.yaml"), files["orders.pack/pack.yaml"], 0o644)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	testAfterPublishCallback = func() {
		if err := os.WriteFile(filepath.Join(tree, "orders.pack", "foreign.txt"), []byte("foreign\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	err = lock.PublishExpected("tree-writer", "rerun tree writer", []OutputExpectation{expected}, writeExpected)
	testAfterPublishCallback = nil
	if err == nil || !strings.Contains(err.Error(), "foreign.txt") {
		t.Fatalf("unexpected output-tree addition was accepted: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	lock, err = Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if err := lock.PublishExpected("tree-writer", "rerun tree writer", []OutputExpectation{expected}, writeExpected); err != nil {
		t.Fatalf("exact tree retry failed: %v", err)
	}
}

func TestPublishExpectedRejectsAtomicPathReplacementAfterHash(t *testing.T) {
	design := t.TempDir()
	output := filepath.Join(design, "generated.txt")
	expected := []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	replaced := false
	testAfterExpectedOutputRead = func(path string) {
		if replaced || path != output {
			return
		}
		replaced = true
		temp := filepath.Join(design, "replacement.tmp")
		if err := os.WriteFile(temp, []byte("new\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(temp, output); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { testAfterExpectedOutputRead = nil }()
	err = lock.PublishExpected("atomic-replace", "rerun atomic replacement test", expected, func() error {
		return os.WriteFile(output, []byte("new\n"), 0o644)
	})
	if err == nil || !strings.Contains(err.Error(), "changed identity while reading") {
		t.Fatalf("atomic output replacement was accepted: %v", err)
	}
}

func TestPublishExpectedTreeRejectsMembershipMutationBetweenInventories(t *testing.T) {
	for _, remove := range []bool{false, true} {
		name := "addition"
		if remove {
			name = "removal"
		}
		t.Run(name, func(t *testing.T) {
			design := t.TempDir()
			tree := filepath.Join(design, "packs")
			files := map[string][]byte{"pack.yaml": []byte("id: orders\n")}
			expected, err := ExpectTree(tree, files, 0o755, 0o644)
			if err != nil {
				t.Fatal(err)
			}
			lock, err := Acquire(design)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Release()
			mutated := false
			testBetweenExpectedTreeFingerprints = func(path string) {
				if mutated || path != tree {
					return
				}
				mutated = true
				if remove {
					if err := os.Remove(filepath.Join(tree, "pack.yaml")); err != nil {
						t.Fatal(err)
					}
					return
				}
				if err := os.WriteFile(filepath.Join(tree, "foreign.txt"), []byte("foreign\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			defer func() { testBetweenExpectedTreeFingerprints = nil }()
			err = lock.PublishExpected("tree-membership", "rerun tree membership test", []OutputExpectation{expected}, func() error {
				if err := os.MkdirAll(tree, 0o755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(tree, "pack.yaml"), files["pack.yaml"], 0o644)
			})
			if err == nil || !strings.Contains(err.Error(), "changed while validating") {
				t.Fatalf("tree membership %s was accepted: %v", name, err)
			}
		})
	}
}

func TestFingerprintExternalRejectsAtomicPathReplacementAfterRead(t *testing.T) {
	design := t.TempDir()
	externalDir := t.TempDir()
	external := filepath.Join(externalDir, "routing.yaml")
	if err := os.WriteFile(external, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	replaced := false
	testAfterExternalInputRead = func(path string) {
		if replaced || path != external {
			return
		}
		replaced = true
		temp := filepath.Join(externalDir, "routing.tmp")
		if err := os.WriteFile(temp, []byte("version: 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(temp, external); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { testAfterExternalInputRead = nil }()
	if err := lock.TrackExternal(external); err == nil || !strings.Contains(err.Error(), "changed identity while reading") {
		t.Fatalf("atomic external-input replacement was accepted: %v", err)
	}
}

func TestFingerprintStreamingHandlesLargeSparseGovernedFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "large-archaeology.bin")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(64 << 20); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := fingerprintExternal(path); err != nil {
		t.Fatalf("stream large external fingerprint: %v", err)
	}
	values, err := fingerprintRoot(root, true)
	if err != nil {
		t.Fatalf("stream large rooted fingerprint: %v", err)
	}
	if !strings.HasPrefix(values["large-archaeology.bin"], "file:600:") {
		t.Fatalf("large rooted fingerprint = %q", values["large-archaeology.bin"])
	}
}

func TestStreamFingerprintEnforcesWitnessedSize(t *testing.T) {
	if _, err := streamFingerprint("short", strings.NewReader("abc"), 4); err == nil || !strings.Contains(err.Error(), "ended early") {
		t.Fatalf("short fingerprint error = %v", err)
	}
	if _, err := streamFingerprint("extra", strings.NewReader("abc"), 2); err == nil || !strings.Contains(err.Error(), "exceeds witnessed size") {
		t.Fatalf("extra fingerprint error = %v", err)
	}
	if _, err := streamFingerprint("exact", strings.NewReader("abc"), 3); err != nil {
		t.Fatalf("exact fingerprint rejected: %v", err)
	}
}

func TestFingerprintsRejectGrowthDuringBoundedStream(t *testing.T) {
	for _, tc := range []struct {
		name        string
		fingerprint func(string, string) error
	}{
		{
			name: "external",
			fingerprint: func(_ string, path string) error {
				_, err := fingerprintExternal(path)
				return err
			},
		},
		{
			name: "rooted",
			fingerprint: func(root, _ string) error {
				_, err := fingerprintRoot(root, true)
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "input.bin")
			if err := os.WriteFile(path, bytes.Repeat([]byte("a"), fingerprintBufferBytes*2), 0o600); err != nil {
				t.Fatal(err)
			}
			mutated := false
			testAfterFingerprintReadChunk = func(got string) {
				if mutated || got != path {
					return
				}
				mutated = true
				file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := file.Write([]byte("growth")); err != nil {
					_ = file.Close()
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}
			t.Cleanup(func() { testAfterFingerprintReadChunk = nil })
			if err := tc.fingerprint(root, path); err == nil ||
				(!strings.Contains(err.Error(), "changed identity while reading") && !strings.Contains(err.Error(), "exceeds witnessed size")) {
				t.Fatalf("concurrent growth fingerprint error = %v", err)
			}
		})
	}
}

func TestFingerprintContinuousAppenderCannotPreventBoundedTermination(t *testing.T) {
	for _, tc := range []struct {
		name        string
		fingerprint func(string, string) error
	}{
		{
			name: "external",
			fingerprint: func(_ string, path string) error {
				_, err := fingerprintExternal(path)
				return err
			},
		},
		{
			name: "rooted",
			fingerprint: func(root, _ string) error {
				_, err := fingerprintRoot(root, true)
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "growing.bin")
			if err := os.WriteFile(path, bytes.Repeat([]byte("a"), fingerprintBufferBytes*2), 0o600); err != nil {
				t.Fatal(err)
			}
			stop := make(chan struct{})
			appended := make(chan struct{})
			appenderDone := make(chan error, 1)
			started := false
			testAfterFingerprintReadChunk = func(got string) {
				if started || got != path {
					return
				}
				started = true
				go func() {
					file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
					if err != nil {
						appenderDone <- err
						return
					}
					first := true
					for {
						select {
						case <-stop:
							appenderDone <- file.Close()
							return
						default:
						}
						if _, err := file.Write([]byte("growth")); err != nil {
							_ = file.Close()
							appenderDone <- err
							return
						}
						if first {
							close(appended)
							first = false
						}
					}
				}()
				<-appended
			}
			t.Cleanup(func() { testAfterFingerprintReadChunk = nil })

			result := make(chan error, 1)
			go func() { result <- tc.fingerprint(root, path) }()
			var fingerprintErr error
			select {
			case fingerprintErr = <-result:
			case <-time.After(2 * time.Second):
				close(stop)
				<-appenderDone
				t.Fatal("continuous appender prevented fingerprint termination")
			}
			close(stop)
			if err := <-appenderDone; err != nil {
				t.Fatal(err)
			}
			if fingerprintErr == nil || !strings.Contains(fingerprintErr.Error(), "exceeds witnessed size") {
				t.Fatalf("continuous append fingerprint error = %v", fingerprintErr)
			}
		})
	}
}

func TestRootedFingerprintRejectsAtomicReplacementAfterRead(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "input.txt")
	if err := os.WriteFile(path, []byte("stable\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mutated := false
	testAfterFingerprintFileRead = func(got string) {
		if mutated || got != path {
			return
		}
		mutated = true
		replacement := filepath.Join(root, "replacement")
		if err := os.WriteFile(replacement, []byte("stable\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(replacement, path); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { testAfterFingerprintFileRead = nil })
	if _, err := fingerprintRoot(root, true); err == nil || !strings.Contains(err.Error(), "changed identity while reading") {
		t.Fatalf("rooted atomic replacement fingerprint error = %v", err)
	}
}

func TestFingerprintsRejectContentABAWithRestoredMtime(t *testing.T) {
	for _, tc := range []struct {
		name        string
		fingerprint func(string, string) error
	}{
		{
			name: "external",
			fingerprint: func(_ string, path string) error {
				_, err := fingerprintExternal(path)
				return err
			},
		},
		{
			name: "rooted",
			fingerprint: func(root, _ string) error {
				_, err := fingerprintRoot(root, true)
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "input.txt")
			original := []byte("stable\n")
			if err := os.WriteFile(path, original, 0o600); err != nil {
				t.Fatal(err)
			}
			before, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			if fingerprintFileChangeID(before) == "" {
				t.Skip("platform does not expose a stable file change identifier")
			}
			mutated := false
			testAfterFingerprintFileRead = func(got string) {
				if mutated || got != path {
					return
				}
				mutated = true
				if err := os.WriteFile(path, []byte("changed"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, original, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
					t.Fatal(err)
				}
			}
			t.Cleanup(func() { testAfterFingerprintFileRead = nil })
			if err := tc.fingerprint(root, path); err == nil || !strings.Contains(err.Error(), "changed identity while reading") {
				t.Fatalf("content ABA fingerprint error = %v", err)
			}
		})
	}
}

func TestFingerprintRejectsDirectoryMembershipMutationBetweenPasses(t *testing.T) {
	for _, remove := range []bool{false, true} {
		name := "addition"
		if remove {
			name = "removal"
		}
		t.Run(name, func(t *testing.T) {
			design := t.TempDir()
			member := filepath.Join(design, "member.txt")
			if remove {
				if err := os.WriteFile(member, []byte("member\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			lock, err := Acquire(design)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Release()
			mutated := false
			testBetweenFingerprintPasses = func(root string) {
				if mutated || root != design {
					return
				}
				mutated = true
				if remove {
					if err := os.Remove(member); err != nil {
						t.Fatal(err)
					}
					return
				}
				if err := os.WriteFile(member, []byte("member\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			defer func() { testBetweenFingerprintPasses = nil }()
			if err := lock.CheckUnchanged(); err == nil || !strings.Contains(err.Error(), "changed while fingerprinting") {
				t.Fatalf("directory membership %s was accepted: %v", name, err)
			}
		})
	}
}

func TestPublishExpectedExternalOutputUsesRetainedParentCapability(t *testing.T) {
	for _, tc := range []struct {
		name   string
		tamper func(string) error
	}{
		{"content", func(path string) error { return os.WriteFile(path, []byte("evil\n"), 0o644) }},
		{"deletion", os.Remove},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			external := t.TempDir()
			output := filepath.Join(external, "generated.txt")
			expected := []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}
			lock, err := Acquire(design)
			if err != nil {
				t.Fatal(err)
			}
			testAfterPublishCallback = func() {
				if err := tc.tamper(output); err != nil {
					t.Fatal(err)
				}
			}
			err = lock.PublishExpected("external-writer", "rerun external writer", expected, func() error {
				return os.WriteFile(output, []byte("new\n"), 0o644)
			})
			testAfterPublishCallback = nil
			if err == nil || !strings.Contains(err.Error(), "predeclared identity") {
				t.Fatalf("external output tampering was accepted: %v", err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
			lock, err = Acquire(design)
			if err != nil {
				t.Fatal(err)
			}
			if err := lock.PublishExpected("external-writer", "rerun external writer", expected, func() error {
				return os.WriteFile(output, []byte("new\n"), 0o644)
			}); err != nil {
				t.Fatalf("exact external retry failed: %v", err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPublishExpectedRootedConfinesMissingExternalParentSwap(t *testing.T) {
	design := t.TempDir()
	outer := t.TempDir()
	base := filepath.Join(outer, "base")
	parked := filepath.Join(outer, "parked")
	if err := os.Mkdir(base, 0o755); err != nil {
		t.Fatal(err)
	}
	outdir := filepath.Join(base, "missing", "generated")
	output := filepath.Join(outdir, "result.txt")
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	testBeforePublishCallback = func() {
		if err := os.Rename(base, parked); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(base, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, "outside-sentinel"), []byte("outside\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { testBeforePublishCallback = nil }()
	expected := []OutputExpectation{ExpectFile(output, []byte("generated\n"), 0o644)}
	err = lock.PublishExpectedRooted("rooted-swap", "rerun rooted swap", expected, func(outputs *OutputScope) error {
		return outputs.WithRoot(outdir, func(root *os.Root) error {
			return root.WriteFile("result.txt", []byte("generated\n"), 0o644)
		})
	})
	if err == nil || !strings.Contains(err.Error(), "changed identity") {
		t.Fatalf("external parent swap was accepted: %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(base, "outside-sentinel")); err != nil || string(body) != "outside\n" {
		t.Fatalf("replacement-tree sentinel changed: %q %v", body, err)
	}
	if _, err := os.Lstat(filepath.Join(base, "missing", "generated", "result.txt")); !os.IsNotExist(err) {
		t.Fatalf("mutation escaped into replacement tree: %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(parked, "missing", "generated", "result.txt")); err != nil || string(body) != "generated\n" {
		t.Fatalf("retained original root did not receive output: %q %v", body, err)
	}
}

func TestPublishExpectedExternalParentSwapFailsClosed(t *testing.T) {
	design := t.TempDir()
	container := t.TempDir()
	external := filepath.Join(container, "out")
	parked := filepath.Join(container, "parked")
	if err := os.Mkdir(external, 0o755); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(external, "generated.txt")
	expected := []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	testAfterPublishCallback = func() {
		if err := os.Rename(external, parked); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(external, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(external, "outside-sentinel"), []byte("untouched\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	err = lock.PublishExpected("external-writer", "rerun external writer", expected, func() error {
		return os.WriteFile(output, []byte("new\n"), 0o644)
	})
	testAfterPublishCallback = nil
	if err == nil || !strings.Contains(err.Error(), "changed identity") {
		t.Fatalf("external parent replacement was accepted: %v", err)
	}
	if body, readErr := os.ReadFile(filepath.Join(external, "outside-sentinel")); readErr != nil || string(body) != "untouched\n" {
		t.Fatalf("replacement-parent sentinel changed: %q, %v", body, readErr)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestPublishExpectedCreatesMissingExternalOutputDirectoryUnderRetainedAncestor(t *testing.T) {
	design := t.TempDir()
	container := t.TempDir()
	parent := filepath.Join(container, "new", "nested")
	output := filepath.Join(parent, "generated.txt")
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	expected := []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}
	if err := lock.PublishExpected("external-writer", "rerun external writer", expected, func() error {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return err
		}
		return os.WriteFile(output, []byte("new\n"), 0o644)
	}); err != nil {
		t.Fatalf("missing external output directory was not supported: %v", err)
	}
}

func TestPublishExpectedRetryRetainsPreviouslyDeclaredAbsentOutput(t *testing.T) {
	design := t.TempDir()
	current := filepath.Join(design, "current.txt")
	stale := filepath.Join(design, "stale.txt")
	if err := os.WriteFile(stale, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	first := []OutputExpectation{
		ExpectFile(current, []byte("new\n"), 0o644),
		ExpectAbsent(stale),
	}
	testBeforeFinalOutputValidation = func() error { return errors.New("simulated crash before sentinel clear") }
	err = lock.PublishExpected("reconcile-writer", "rerun reconcile writer", first, func() error {
		if err := os.WriteFile(current, []byte("new\n"), 0o644); err != nil {
			return err
		}
		return os.Remove(stale)
	})
	testBeforeFinalOutputValidation = nil
	if err == nil || !strings.Contains(err.Error(), "simulated crash") {
		t.Fatalf("simulated pre-clear crash succeeded: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	lock, err = Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	// Discovery no longer sees stale.txt, so the retry declares only the file
	// it still owns. PublishExpected must restore the prior absent expectation
	// from the exact same operation sentinel before identity comparison.
	retry := []OutputExpectation{ExpectFile(current, []byte("new\n"), 0o644)}
	if err := lock.PublishExpected("reconcile-writer", "rerun reconcile writer", retry, func() error {
		return os.WriteFile(current, []byte("new\n"), 0o644)
	}); err != nil {
		t.Fatalf("exact retry was stranded after stale deletion: %v", err)
	}
	if _, err := os.Lstat(stale); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("retry resurrected stale output: %v", err)
	}
}

func TestResumeExpectedClearsFinalizedPublicationWithoutRediscovery(t *testing.T) {
	design := t.TempDir()
	current := filepath.Join(design, "current.txt")
	stale := filepath.Join(design, "stale.txt")
	if err := os.WriteFile(stale, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	expected := []OutputExpectation{ExpectFile(current, []byte("new\n"), 0o644), ExpectAbsent(stale)}
	testBeforeFinalOutputValidation = func() error { return errors.New("simulated crash after inner finalize") }
	err = lock.PublishExpected("cleanup-writer", "rerun cleanup writer", expected, func() error {
		if err := os.WriteFile(current, []byte("new\n"), 0o644); err != nil {
			return err
		}
		return os.Remove(stale)
	})
	testBeforeFinalOutputValidation = nil
	if err == nil {
		t.Fatal("simulated finalized crash succeeded")
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	lock, err = Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.ResumeExpected("cleanup-writer", "rerun cleanup writer"); err != nil {
		t.Fatalf("finalized publication could not resume from persisted expectations: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	reader, err := AcquireReader(design)
	if err != nil {
		t.Fatalf("reader remained blocked after finalized publication resume: %v", err)
	}
	if err := reader.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestPublishStageCrashBoundariesRecoverBeforeAnyReaderSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(func() error)
	}{
		{"file sync", func(h func() error) { testAfterPublishStageSync = h }},
		{"directory sync", func(h func() error) { testAfterPublishStageDirSync = h }},
		{"rename", func(h func() error) { testAfterPublishStageRename = h }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				testAfterPublishStageSync = nil
				testAfterPublishStageDirSync = nil
				testAfterPublishStageRename = nil
			})
			design := t.TempDir()
			output := filepath.Join(design, "generated.txt")
			expected := []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}
			lock, err := Acquire(design)
			if err != nil {
				t.Fatal(err)
			}
			tc.set(func() error { return errors.New("simulated publication-stage crash") })
			err = lock.PublishExpected("stage-writer", "rerun stage writer", expected, func() error {
				return os.WriteFile(output, []byte("new\n"), 0o644)
			})
			tc.set(nil)
			if err == nil {
				t.Fatal("simulated stage crash succeeded")
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
			if reader, err := AcquireReader(design); err == nil {
				_ = reader.Release()
				t.Fatal("reader acquired across staged/interrupted publication")
			}
			lock, err = Acquire(design)
			if err != nil {
				t.Fatalf("writer could not reconcile staged publication: %v", err)
			}
			if err := lock.PublishExpected("stage-writer", "rerun stage writer", expected, func() error {
				return os.WriteFile(output, []byte("new\n"), 0o644)
			}); err != nil {
				t.Fatalf("exact retry failed after staged publication recovery: %v", err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPublishRetainedSentinelAuthorityRejectsCallbackMutation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*testing.T, string, string) bool
	}{
		{
			name: "mode corruption",
			mutate: func(t *testing.T, sentinel, _ string) bool {
				if runtime.GOOS == "windows" {
					return false
				}
				if err := os.Chmod(sentinel, 0o644); err != nil {
					t.Fatal(err)
				}
				return true
			},
		},
		{
			name: "content replacement",
			mutate: func(t *testing.T, sentinel, _ string) bool {
				body, err := os.ReadFile(sentinel)
				if err != nil {
					t.Fatal(err)
				}
				body[len(body)/2] ^= 1
				if err := os.WriteFile(sentinel, body, 0o600); err != nil {
					t.Fatal(err)
				}
				return true
			},
		},
		{
			name: "same-body path replacement",
			mutate: func(t *testing.T, sentinel, parked string) bool {
				body, err := os.ReadFile(sentinel)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(sentinel, parked); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(sentinel, body, 0o600); err != nil {
					t.Fatal(err)
				}
				return true
			},
		},
		{
			name: "same-byte ABA",
			mutate: func(t *testing.T, sentinel, _ string) bool {
				info, err := os.Lstat(sentinel)
				if err != nil {
					t.Fatal(err)
				}
				if fingerprintFileChangeID(info) == "" {
					return false
				}
				body, err := os.ReadFile(sentinel)
				if err != nil {
					t.Fatal(err)
				}
				changed := bytes.Clone(body)
				changed[len(changed)/2] ^= 1
				if err := os.WriteFile(sentinel, changed, info.Mode().Perm()); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(sentinel, body, info.Mode().Perm()); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(sentinel, info.ModTime(), info.ModTime()); err != nil {
					t.Fatal(err)
				}
				return true
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			output := filepath.Join(design, "generated.txt")
			sentinel := filepath.Join(design, publishSentinel)
			parked := filepath.Join(design, "parked-original-sentinel")
			lock, err := Acquire(design)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Release()
			mutated := false
			prior := testAfterPublishCallback
			testAfterPublishCallback = func() { mutated = tc.mutate(t, sentinel, parked) }
			expected := []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}
			err = lock.PublishExpected("authority-writer", "rerun authority writer", expected, func() error {
				return os.WriteFile(output, []byte("new\n"), 0o644)
			})
			testAfterPublishCallback = prior
			t.Cleanup(func() { testAfterPublishCallback = prior })
			if !mutated {
				t.Skip("native mutation witness unavailable on this platform")
			}
			if err == nil || !strings.Contains(err.Error(), "publication sentinel changed") {
				t.Fatalf("callback-time sentinel mutation was accepted: %v", err)
			}
			if info, statErr := os.Lstat(sentinel); statErr != nil || !info.Mode().IsRegular() {
				t.Fatalf("callback-time replacement was not preserved: info=%v err=%v", info, statErr)
			}
			if tc.name == "same-body path replacement" {
				if info, statErr := os.Lstat(parked); statErr != nil || !info.Mode().IsRegular() {
					t.Fatalf("original sentinel was not preserved: info=%v err=%v", info, statErr)
				}
			}
		})
	}
}

func TestPublishSentinelClearPreservesLatePathReplacement(t *testing.T) {
	design := t.TempDir()
	output := filepath.Join(design, "generated.txt")
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	prior := testPublishCleanupPoint
	testPublishCleanupPoint = func(point, name string) error {
		if point != "before-remove" || name != publishSentinel {
			return nil
		}
		return os.WriteFile(filepath.Join(design, name), []byte("concurrent replacement\n"), 0o600)
	}
	expected := []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}
	err = lock.PublishExpected("clear-writer", "rerun clear writer", expected, func() error {
		return os.WriteFile(output, []byte("new\n"), 0o644)
	})
	testPublishCleanupPoint = prior
	t.Cleanup(func() { testPublishCleanupPoint = prior })
	if err == nil || !strings.Contains(err.Error(), "was repopulated") {
		t.Fatalf("late sentinel replacement was deleted or accepted: %v", err)
	}
	if body, readErr := os.ReadFile(filepath.Join(design, publishSentinel)); readErr != nil || string(body) != "concurrent replacement\n" {
		t.Fatalf("sentinel replacement = %q, %v", body, readErr)
	}
	root, openErr := os.OpenRoot(design)
	if openErr != nil {
		t.Fatal(openErr)
	}
	quarantine, findErr := findPublishQuarantine(root)
	closeErr := root.Close()
	if findErr != nil || closeErr != nil || quarantine == "" {
		t.Fatalf("original sentinel quarantine = %q, find %v, close %v", quarantine, findErr, closeErr)
	}
}

func TestPublishSentinelRetirementDoesNotReplaceDestinationCollision(t *testing.T) {
	design := t.TempDir()
	output := filepath.Join(design, "generated.txt")
	retired := filepath.Join(design, publishSentinelRetired)
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	prior := testPublishCleanupPoint
	testPublishCleanupPoint = func(point, name string) error {
		if point == "before-retire" && name == publishSentinelRetired {
			return os.WriteFile(retired, []byte("concurrent retirement\n"), 0o600)
		}
		return nil
	}
	err = lock.PublishExpected("collision-writer", "rerun collision writer", []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}, func() error {
		return os.WriteFile(output, []byte("new\n"), 0o644)
	})
	testPublishCleanupPoint = prior
	t.Cleanup(func() { testPublishCleanupPoint = prior })
	if err == nil || !strings.Contains(err.Error(), "appeared concurrently") {
		t.Fatalf("retirement destination collision was overwritten or accepted: %v", err)
	}
	for path, want := range map[string]string{
		filepath.Join(design, publishSentinel): "\"operation\":\"collision-writer\"",
		retired:                                "concurrent retirement\n",
	} {
		body, readErr := os.ReadFile(path)
		if readErr != nil || !strings.Contains(string(body), want) {
			t.Fatalf("preserved %s = %q, %v; want content %q", path, body, readErr, want)
		}
	}
}

func TestPublishSentinelInstallPreservesAtomicDestinationCollision(t *testing.T) {
	design := t.TempDir()
	output := filepath.Join(design, "generated.txt")
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	concurrent := []byte("concurrent publication authority\n")
	prior := publishRenameNoReplace
	publishRenameNoReplace = func(root *os.Root, from, to string) error {
		if from == publishSentinelStage && to == publishSentinel {
			file, openErr := root.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
			if openErr != nil {
				return openErr
			}
			_, writeErr := file.Write(concurrent)
			if err := errors.Join(writeErr, file.Close()); err != nil {
				return err
			}
		}
		return prior(root, from, to)
	}
	err = lock.PublishExpected("install-collision", "rerun install collision", []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}, func() error {
		return os.WriteFile(output, []byte("new\n"), 0o644)
	})
	publishRenameNoReplace = prior
	t.Cleanup(func() { publishRenameNoReplace = prior })
	if err == nil || !strings.Contains(err.Error(), "install design publication sentinel") {
		t.Fatalf("destination collision was overwritten or accepted: %v", err)
	}
	body, readErr := os.ReadFile(filepath.Join(design, publishSentinel))
	if readErr != nil || !bytes.Equal(body, concurrent) {
		t.Fatalf("concurrent publication authority = %q, %v", body, readErr)
	}
	if _, statErr := os.Lstat(filepath.Join(design, publishSentinelStage)); statErr != nil {
		t.Fatalf("staged publication authority was not preserved: %v", statErr)
	}
}

func TestPublishSentinelPrivateDeletionPreservesPublicReplacement(t *testing.T) {
	design := t.TempDir()
	output := filepath.Join(design, "generated.txt")
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	prior := testPublishCleanupPoint
	testPublishCleanupPoint = func(point, name string) error {
		if point == "after-quarantine" && name == publishSentinel {
			return os.WriteFile(filepath.Join(design, publishSentinel), []byte("post-check replacement\n"), 0o600)
		}
		return nil
	}
	err = lock.PublishExpected("replacement-writer", "rerun replacement writer", []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}, func() error {
		return os.WriteFile(output, []byte("new\n"), 0o644)
	})
	testPublishCleanupPoint = prior
	t.Cleanup(func() { testPublishCleanupPoint = prior })
	if err == nil || !strings.Contains(err.Error(), "repopulated") {
		t.Fatalf("post-check retirement replacement was deleted or accepted: %v", err)
	}
	body, readErr := os.ReadFile(filepath.Join(design, publishSentinel))
	if readErr != nil || string(body) != "post-check replacement\n" {
		t.Fatalf("public replacement = %q, %v", body, readErr)
	}
	root, openErr := os.OpenRoot(design)
	if openErr != nil {
		t.Fatal(openErr)
	}
	quarantine, findErr := findPublishQuarantine(root)
	closeErr := root.Close()
	if findErr != nil || closeErr != nil || quarantine == "" {
		t.Fatalf("original authority was not preserved in private quarantine: name=%q find=%v close=%v", quarantine, findErr, closeErr)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	if reader, acquireErr := AcquireReader(design); acquireErr == nil {
		_ = reader.Release()
		t.Fatal("reader accepted design with preserved publication quarantine")
	}
}

func TestPublishSentinelCleanupCrashBoundariesRestartCleanly(t *testing.T) {
	for _, point := range []string{
		"before-isolate",
		"before-retire",
		"before-quarantine",
		"after-isolate",
		"quarantine-durable",
		"before-remove",
		"after-quarantine",
		"quarantine-removed",
	} {
		t.Run(point, func(t *testing.T) {
			design := t.TempDir()
			output := filepath.Join(design, "generated.txt")
			lock, err := Acquire(design)
			if err != nil {
				t.Fatal(err)
			}
			prior := testPublishCleanupPoint
			testPublishCleanupPoint = func(got, _ string) error {
				if got == point {
					panic("power loss at " + point)
				}
				return nil
			}
			func() {
				defer func() {
					if recovered := recover(); recovered == nil {
						t.Fatalf("cleanup crash point %s did not fire", point)
					}
				}()
				_ = lock.PublishExpected("cleanup-crash", "rerun cleanup crash", []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}, func() error {
					return os.WriteFile(output, []byte("new\n"), 0o644)
				})
			}()
			testPublishCleanupPoint = prior
			t.Cleanup(func() { testPublishCleanupPoint = prior })
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
			lock, err = Acquire(design)
			if err != nil {
				t.Fatalf("restart after %s: %v", point, err)
			}
			if err := lock.PublishExpected("cleanup-crash", "rerun cleanup crash", []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}, func() error {
				return os.WriteFile(output, []byte("new\n"), 0o644)
			}); err != nil {
				t.Fatalf("exact retry after %s: %v", point, err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(design)
			if err != nil {
				t.Fatal(err)
			}
			quarantine, findErr := findPublishQuarantine(root)
			closeErr := root.Close()
			if findErr != nil || closeErr != nil || quarantine != "" {
				t.Fatalf("residue after %s retry: name=%q find=%v close=%v", point, quarantine, findErr, closeErr)
			}
		})
	}
}

func TestPublishSentinelCleanupResumesCrashAfterPrivateObjectRemoval(t *testing.T) {
	design := t.TempDir()
	output := filepath.Join(design, "generated.txt")
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	prior := publishQuarantineRemove
	publishQuarantineRemove = func(quarantine *fsatomic.Quarantined) error {
		if err := quarantine.Root().Remove(quarantine.Name()); err != nil {
			return err
		}
		panic("power loss after private object removal")
	}
	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("private-object removal crash did not fire")
			}
		}()
		_ = lock.PublishExpected("remove-crash", "rerun remove crash", []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}, func() error {
			return os.WriteFile(output, []byte("new\n"), 0o644)
		})
	}()
	publishQuarantineRemove = prior
	t.Cleanup(func() { publishQuarantineRemove = prior })
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	lock, err = Acquire(design)
	if err != nil {
		t.Fatalf("restart did not finish empty quarantine: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestPublishQuarantineRecoveryCrashBoundariesRestartCleanly(t *testing.T) {
	for _, point := range []string{"reconcile-quarantine-open", "reconcile-before-quarantine-remove", "reconcile-quarantine-removed"} {
		t.Run(point, func(t *testing.T) {
			design := t.TempDir()
			output := filepath.Join(design, "generated.txt")
			lock, err := Acquire(design)
			if err != nil {
				t.Fatal(err)
			}
			prior := testPublishCleanupPoint
			testPublishCleanupPoint = func(got, _ string) error {
				if got == "after-isolate" {
					return errors.New("leave durable quarantine")
				}
				return nil
			}
			err = lock.PublishExpected("recovery-crash", "rerun recovery crash", []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}, func() error {
				return os.WriteFile(output, []byte("new\n"), 0o644)
			})
			testPublishCleanupPoint = prior
			if err == nil {
				t.Fatal("setup did not retain quarantine")
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable, "-test.run=^TestPublishQuarantineRecoveryCrashHelper$")
			cmd.Env = append(os.Environ(), "MACHINERY_PUBLISH_QUARANTINE_CRASH_DESIGN="+design, "MACHINERY_PUBLISH_QUARANTINE_CRASH_POINT="+point)
			if err := cmd.Run(); err == nil {
				t.Fatalf("recovery helper did not crash at %s", point)
			}
			lock, err = Acquire(design)
			if err != nil {
				t.Fatalf("second restart after %s: %v", point, err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPublishQuarantineRecoveryCrashHelper(t *testing.T) {
	design := os.Getenv("MACHINERY_PUBLISH_QUARANTINE_CRASH_DESIGN")
	point := os.Getenv("MACHINERY_PUBLISH_QUARANTINE_CRASH_POINT")
	if design == "" || point == "" {
		return
	}
	testPublishCleanupPoint = func(got, _ string) error {
		if got == point {
			panic("simulated process death during publication quarantine recovery")
		}
		return nil
	}
	_, _ = Acquire(design)
	t.Fatal("publication quarantine recovery crash point did not fire")
}

func TestPublishQuarantineRecoveryPreservesForeignResidue(t *testing.T) {
	design := t.TempDir()
	foreign := filepath.Join(design, "foreign")
	if err := os.WriteFile(foreign, []byte("foreign\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(design)
	if err != nil {
		t.Fatal(err)
	}
	quarantine, err := fsatomic.Quarantine(root, "foreign", publishSentinelQuarantinePrefix)
	if err != nil {
		t.Fatal(err)
	}
	if err := quarantine.Close(); err != nil {
		t.Fatal(err)
	}
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	if lock, err := Acquire(design); err == nil {
		_ = lock.Release()
		t.Fatal("foreign quarantine was accepted")
	}
	root, err = os.OpenRoot(design)
	if err != nil {
		t.Fatal(err)
	}
	name, findErr := findPublishQuarantine(root)
	closeErr := root.Close()
	if findErr != nil || closeErr != nil || name == "" {
		t.Fatalf("foreign quarantine was not preserved: name=%q find=%v close=%v", name, findErr, closeErr)
	}
}

func TestReconcilePublishStagePreservesLateMutation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		point  string
		mutate func(*testing.T, string, string) bool
	}{
		{
			name:  "mode corruption",
			point: "before-isolate",
			mutate: func(t *testing.T, path, _ string) bool {
				if runtime.GOOS == "windows" {
					return false
				}
				if err := os.Chmod(path, 0o644); err != nil {
					t.Fatal(err)
				}
				return true
			},
		},
		{
			name:  "content after isolation",
			point: "after-isolate",
			mutate: func(t *testing.T, path, _ string) bool {
				root, err := os.OpenRoot(filepath.Dir(path))
				if err != nil {
					t.Fatal(err)
				}
				quarantine, findErr := findPublishQuarantine(root)
				if closeErr := root.Close(); findErr != nil || closeErr != nil || quarantine == "" {
					t.Fatalf("find isolated stage: name=%q find=%v close=%v", quarantine, findErr, closeErr)
				}
				if err := os.WriteFile(filepath.Join(filepath.Dir(path), quarantine, "object"), []byte("late replacement\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				return true
			},
		},
		{
			name:  "same-body path replacement",
			point: "before-quarantine",
			mutate: func(t *testing.T, path, parked string) bool {
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(path, parked); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, body, 0o600); err != nil {
					t.Fatal(err)
				}
				return true
			},
		},
		{
			name:  "same-byte ABA",
			point: "before-isolate",
			mutate: func(t *testing.T, path, _ string) bool {
				info, err := os.Lstat(path)
				if err != nil {
					t.Fatal(err)
				}
				if fingerprintFileChangeID(info) == "" {
					return false
				}
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				changed := bytes.Clone(body)
				changed[0] ^= 1
				if err := os.WriteFile(path, changed, info.Mode().Perm()); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, body, info.Mode().Perm()); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
					t.Fatal(err)
				}
				return true
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			stage := filepath.Join(design, publishSentinelStage)
			parked := filepath.Join(design, "parked-original-stage")
			if err := os.WriteFile(stage, []byte("partial stage\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			mutated := false
			prior := testPublishCleanupPoint
			testPublishCleanupPoint = func(point, name string) error {
				if point == tc.point {
					mutated = tc.mutate(t, filepath.Join(design, name), parked)
				}
				return nil
			}
			lock, err := Acquire(design)
			testPublishCleanupPoint = prior
			t.Cleanup(func() { testPublishCleanupPoint = prior })
			if lock != nil {
				_ = lock.Release()
			}
			if !mutated {
				t.Skip("native mutation witness unavailable on this platform")
			}
			if err == nil || !strings.Contains(err.Error(), "preserv") && !strings.Contains(err.Error(), "changed") && !strings.Contains(err.Error(), "restoring") {
				t.Fatalf("reconcile-time stage mutation was accepted: %v", err)
			}
			if tc.name == "content after isolation" {
				root, openErr := os.OpenRoot(design)
				if openErr != nil {
					t.Fatal(openErr)
				}
				quarantine, findErr := findPublishQuarantine(root)
				closeErr := root.Close()
				if findErr != nil || closeErr != nil || quarantine == "" {
					t.Fatalf("mutated stage quarantine was not preserved: name=%q find=%v close=%v", quarantine, findErr, closeErr)
				}
			} else if info, statErr := os.Lstat(stage); statErr != nil || !info.Mode().IsRegular() {
				t.Fatalf("reconcile-time stage replacement was not preserved: info=%v err=%v", info, statErr)
			}
			if tc.name == "same-body path replacement" {
				if info, statErr := os.Lstat(parked); statErr != nil || !info.Mode().IsRegular() {
					t.Fatalf("isolated original stage was not preserved: info=%v err=%v", info, statErr)
				}
			}
		})
	}
}

func TestReconcilePublishStageRetainsStageWhenInstalledAuthorityChanges(t *testing.T) {
	design := t.TempDir()
	output := filepath.Join(design, "generated.txt")
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	expected := []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}
	err = lock.PublishExpected("guard-writer", "rerun guard writer", expected, func() error {
		if err := os.WriteFile(output, []byte("new\n"), 0o644); err != nil {
			return err
		}
		return errors.New("retain installed sentinel")
	})
	if err == nil {
		t.Fatal("failed publication unexpectedly cleared its sentinel")
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(design, publishSentinel)
	stage := filepath.Join(design, publishSentinelStage)
	body, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stage, body, 0o600); err != nil {
		t.Fatal(err)
	}
	prior := testPublishCleanupPoint
	testPublishCleanupPoint = func(point, _ string) error {
		if point == "before-remove" {
			return os.WriteFile(sentinel, []byte("corrupt installed authority\n"), 0o600)
		}
		return nil
	}
	lock, err = Acquire(design)
	testPublishCleanupPoint = prior
	t.Cleanup(func() { testPublishCleanupPoint = prior })
	if lock != nil {
		_ = lock.Release()
	}
	if err == nil || !strings.Contains(err.Error(), "cleanup authority changed") {
		t.Fatalf("reconciliation deleted its matching stage after installed-authority mutation: %v", err)
	}
	if body, readErr := os.ReadFile(sentinel); readErr != nil || string(body) != "corrupt installed authority\n" {
		t.Fatalf("installed replacement = %q, %v", body, readErr)
	}
	root, openErr := os.OpenRoot(design)
	if openErr != nil {
		t.Fatal(openErr)
	}
	quarantine, findErr := findPublishQuarantine(root)
	closeErr := root.Close()
	if findErr != nil || closeErr != nil || quarantine == "" {
		t.Fatalf("matching stage was not preserved in quarantine: name=%q find=%v close=%v", quarantine, findErr, closeErr)
	}
}

func TestWriterDurablyDiscardsEveryPartialLonePublishStage(t *testing.T) {
	for _, tc := range []struct {
		name string
		body []byte
	}{
		{name: "zero-byte", body: nil},
		{name: "truncated", body: []byte(`{"version":1`)},
		{name: "oversize", body: bytes.Repeat([]byte("x"), publishRecordMaxBytes+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			stage := filepath.Join(design, publishSentinelStage)
			if err := os.WriteFile(stage, tc.body, 0o600); err != nil {
				t.Fatal(err)
			}
			lock, err := Acquire(design)
			if err != nil {
				t.Fatalf("writer did not recover lone partial stage: %v", err)
			}
			output := filepath.Join(design, "generated.txt")
			expected := []OutputExpectation{ExpectFile(output, []byte("new\n"), 0o644)}
			if err := lock.PublishExpected("partial-stage", "rerun partial-stage writer", expected, func() error {
				return os.WriteFile(output, []byte("new\n"), 0o644)
			}); err != nil {
				t.Fatalf("normal callback did not proceed after lone-stage recovery: %v", err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(stage); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("lone stage remains after recovery: %v", err)
			}
		})
	}
}

func TestWriterRefusesUnsafeLonePublishStageTypes(t *testing.T) {
	for _, kind := range []string{"symlink", "directory"} {
		t.Run(kind, func(t *testing.T) {
			design := t.TempDir()
			stage := filepath.Join(design, publishSentinelStage)
			switch kind {
			case "symlink":
				target := filepath.Join(t.TempDir(), "outside")
				if err := os.WriteFile(target, []byte("outside sentinel\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, stage); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(stage, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Acquire(design); err == nil || !strings.Contains(err.Error(), "private regular file") {
				t.Fatalf("unsafe %s stage was accepted: %v", kind, err)
			}
		})
	}
}

func TestAcquireReaderBoundsOversizedPublishSentinel(t *testing.T) {
	design := t.TempDir()
	body := bytes.Repeat([]byte("x"), publishRecordMaxBytes+1)
	if err := os.WriteFile(filepath.Join(design, publishSentinel), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if reader, err := AcquireReader(design); err == nil || !strings.Contains(err.Error(), "exceeds") {
		if reader != nil {
			_ = reader.Release()
		}
		t.Fatalf("oversized publication sentinel was not bounded: %v", err)
	}
}

func TestCheckUnchangedRejectsEmptyDirectoryCreateDeleteAndModeChange(t *testing.T) {
	for _, tc := range []struct {
		name   string
		setup  func(string) error
		mutate func(string) error
	}{
		{"create", func(string) error { return nil }, func(root string) error { return os.Mkdir(filepath.Join(root, "empty"), 0o755) }},
		{"delete", func(root string) error { return os.Mkdir(filepath.Join(root, "empty"), 0o755) }, func(root string) error { return os.Remove(filepath.Join(root, "empty")) }},
		{"mode", func(root string) error { return os.Mkdir(filepath.Join(root, "empty"), 0o755) }, func(root string) error { return os.Chmod(filepath.Join(root, "empty"), 0o700) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			if err := tc.setup(design); err != nil {
				t.Fatal(err)
			}
			lock, err := Acquire(design)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Release()
			if err := tc.mutate(design); err != nil {
				t.Fatal(err)
			}
			if err := lock.CheckUnchanged(); err == nil || !strings.Contains(err.Error(), "empty/") {
				t.Fatalf("CheckUnchanged error = %v", err)
			}
		})
	}
}

func TestCheckUnchangedRejectsTrackedImplementationMutation(t *testing.T) {
	design := t.TempDir()
	impl := t.TempDir()
	path := filepath.Join(impl, "service.go")
	if err := os.WriteFile(path, []byte("package service\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if err := lock.TrackExternalTree(impl); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := lock.CheckUnchanged(); err == nil || !strings.Contains(err.Error(), "external tree changed") {
		t.Fatalf("CheckUnchanged error = %v", err)
	}
}

func TestExternalSnapshotRejectsClassifyOpenSymlinkSwap(t *testing.T) {
	design := t.TempDir()
	impl := t.TempDir()
	dest := t.TempDir()
	source := filepath.Join(impl, "service.go")
	outside := filepath.Join(t.TempDir(), "outside.go")
	if err := os.WriteFile(source, []byte("inside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("outside sentinel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	swapped := false
	_, err = lock.copyExternalTree(impl, dest, func(rel string) {
		if rel != "service.go" || swapped {
			return
		}
		swapped = true
		if removeErr := os.Remove(source); removeErr != nil {
			t.Fatal(removeErr)
		}
		if symlinkErr := os.Symlink(outside, source); symlinkErr != nil {
			t.Fatal(symlinkErr)
		}
	}, nil)
	if err == nil {
		t.Fatal("regular-to-symlink classify/open swap was accepted")
	}
	body, readErr := os.ReadFile(outside)
	if readErr != nil || string(body) != "outside sentinel\n" {
		t.Fatalf("outside sentinel changed: %q, %v", body, readErr)
	}
}

func TestExternalSnapshotRejectsRootSwap(t *testing.T) {
	design := t.TempDir()
	parent := t.TempDir()
	impl := filepath.Join(parent, "impl")
	parked := filepath.Join(parent, "parked")
	if err := os.Mkdir(impl, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(impl, "service.go"), []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	_, err = lock.copyExternalTree(impl, t.TempDir(), func(rel string) {
		if rel != "." {
			return
		}
		if renameErr := os.Rename(impl, parked); renameErr != nil {
			t.Fatal(renameErr)
		}
		if mkdirErr := os.Mkdir(impl, 0o755); mkdirErr != nil {
			t.Fatal(mkdirErr)
		}
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "root changed identity") {
		t.Fatalf("root swap error = %v", err)
	}
}

func TestAcquireReaderRejectsEveryInterruptedJournalAtOwnedLocation(t *testing.T) {
	paths := []string{
		".machinery-artifact-set.journal",
		"machines/.machinery-artifact-set.journal",
		"formal/.machinery-artifact-set.journal",
		"formal/.machinery-formal-transaction.jsonl",
		"packs/.machinery-pack-transaction.jsonl",
		".machinery/checker-project-transaction.json",
		".machinery/checker-project-transaction.committed.json",
	}
	for _, rel := range paths {
		t.Run(rel, func(t *testing.T) {
			design := t.TempDir()
			path := filepath.Join(design, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("seeded\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := AcquireReader(design); err == nil || !strings.Contains(err.Error(), rel) {
				t.Fatalf("AcquireReader error = %v, want journal %s", err, rel)
			}
		})
	}
}

func TestAcquireReaderDoesNotMatchJournalBasenameOutsideOwnedLocation(t *testing.T) {
	design := t.TempDir()
	path := filepath.Join(design, "docs", ".machinery-formal-transaction.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("documentation fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := AcquireReader(design)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestPublishDetectsMutationAfterPrecheckAndLeavesSentinel(t *testing.T) {
	design := t.TempDir()
	source := filepath.Join(design, "source.yaml")
	output := filepath.Join(design, "generated.txt")
	if err := os.WriteFile(source, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("old output\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	err = lock.Publish("test", "rerun test writer", []string{output}, func() error {
		if err := os.WriteFile(output, []byte("new output\n"), 0o644); err != nil {
			return err
		}
		return os.WriteFile(source, []byte("edited during publish\n"), 0o644)
	})
	if err == nil || !strings.Contains(err.Error(), "source.yaml") {
		t.Fatalf("Publish error = %v", err)
	}
	if releaseErr := lock.Release(); releaseErr != nil {
		t.Fatal(releaseErr)
	}
	if _, err := AcquireReader(design); err == nil || !strings.Contains(err.Error(), "interrupted Machinery publication") {
		t.Fatalf("reader accepted retained publication sentinel: %v", err)
	}
}

func TestFailedOperationCannotBeClearedBySameWriterDifferentOutput(t *testing.T) {
	design := t.TempDir()
	a := filepath.Join(design, "A.tla")
	b := filepath.Join(design, "B.tla")
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.Publish("tla", "rerun tla A", []string{a}, func() error { return os.ErrInvalid }); err == nil {
		t.Fatal("failed publish unexpectedly succeeded")
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	lock, err = Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if err := lock.Publish("tla", "rerun tla B", []string{b}, func() error { return nil }); err == nil || !strings.Contains(err.Error(), "different input/output identity") {
		t.Fatalf("different TLA operation error = %v", err)
	}
}

func TestExactRetrySurvivesFirstRunParentCreation(t *testing.T) {
	design := t.TempDir()
	output := filepath.Join(design, "checkers", "custom", "projection.json")
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.Publish("checker-project", "rerun project", []string{output}, func() error {
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return err
		}
		return os.ErrInvalid
	}); err == nil {
		t.Fatal("seeded first-run crash unexpectedly succeeded")
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}

	lock, err = Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if err := lock.Publish("checker-project", "rerun project", []string{output}, func() error {
		return os.WriteFile(output, []byte("complete\n"), 0o644)
	}); err != nil {
		t.Fatalf("exact retry rejected after parent creation: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(design, publishSentinel)); !os.IsNotExist(err) {
		t.Fatalf("sentinel retained after exact retry: %v", err)
	}
}

func TestAcquireRejectsSymlinkedGovernedInput(t *testing.T) {
	design := t.TempDir()
	outside := filepath.Join(t.TempDir(), "machine.json")
	if err := os.WriteFile(outside, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(design, "machine.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(design); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("Acquire error = %v, want symlink rejection", err)
	}
}

func TestAcquireReaderRejectsMalformedAndSymlinkSentinel(t *testing.T) {
	t.Run("malformed", func(t *testing.T) {
		design := t.TempDir()
		if err := os.WriteFile(filepath.Join(design, publishSentinel), []byte(`{"version":1,"version":1}`+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := AcquireReader(design); err == nil || !strings.Contains(err.Error(), "invalid interrupted") {
			t.Fatalf("AcquireReader error = %v", err)
		}
	})
	t.Run("symlink", func(t *testing.T) {
		design := t.TempDir()
		outside := filepath.Join(t.TempDir(), "sentinel")
		if err := os.WriteFile(outside, []byte("outside\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(design, publishSentinel)); err != nil {
			t.Fatal(err)
		}
		if _, err := AcquireReader(design); err == nil || !strings.Contains(err.Error(), "private regular file") {
			t.Fatalf("AcquireReader error = %v", err)
		}
	})
}
