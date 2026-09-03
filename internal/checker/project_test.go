package checker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/filelock"
)

func TestProjectAll(t *testing.T) {
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "d.modelith.yaml"), []byte(sampleModel), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(design, "checkers"), 0o755); err != nil {
		t.Fatal(err)
	}
	man := "checker: {id: test, runtime_closure: sha256:1111111111111111111111111111111111111111111111111111111111111111}\n" +
		"projection: {include: [model, invariants, relationships]}\n" +
		"evidence: {projection_out: checkers/test/projection.json, evidence_in: checkers/test/evidence.json}\n"
	if err := os.WriteFile(filepath.Join(design, "checkers", "test.checker.yaml"), []byte(man), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := ProjectAll(design, "v0")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].CheckerID != "test" || results[0].Path != "checkers/test/projection.json" {
		t.Fatalf("unexpected results: %+v", results)
	}

	// ProjectAll created the nested output directory and wrote a valid projection
	b, err := os.ReadFile(filepath.Join(design, "checkers", "test", "projection.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseProjection(b)
	if err != nil {
		t.Fatal(err)
	}
	h, err := p.InputHash()
	if err != nil {
		t.Fatal(err)
	}
	// the generated block mirrors the binding hash for adapters
	var raw struct {
		Generated map[string]string `json:"generated"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if raw.Generated["input_hash"] != h {
		t.Fatalf("committed projection mirror %q != fresh hash %q", raw.Generated["input_hash"], h)
	}
}

func TestProjectAllRejectsOrphanedOutputsBeforeMutation(t *testing.T) {
	design := t.TempDir()
	modelPath := filepath.Join(design, "d.modelith.yaml")
	if err := os.WriteFile(modelPath, []byte(sampleModel), 0o644); err != nil {
		t.Fatal(err)
	}
	checkers := filepath.Join(design, "checkers")
	if err := os.Mkdir(checkers, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := func(id string) string {
		return "checker: {id: " + id + ", runtime_closure: sha256:1111111111111111111111111111111111111111111111111111111111111111}\n" +
			"projection: {include: [model]}\n" +
			"evidence: {projection_out: checkers/" + id + "/projection.json, evidence_in: checkers/" + id + "/evidence.json}\n"
	}
	manifestPaths := map[string]string{}
	for _, id := range []string{"current", "removed"} {
		path := filepath.Join(checkers, id+".checker.yaml")
		if err := os.WriteFile(path, []byte(manifest(id)), 0o644); err != nil {
			t.Fatal(err)
		}
		manifestPaths[id] = path
	}
	if _, err := ProjectAll(design, "v-test"); err != nil {
		t.Fatal(err)
	}
	currentProjection := filepath.Join(checkers, "current", "projection.json")
	before, err := os.ReadFile(currentProjection)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(manifestPaths["removed"]); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, append([]byte(sampleModel), []byte("\n# would change the surviving projection\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	_, firstErr := ProjectAll(design, "v-test")
	_, secondErr := ProjectAll(design, "v-test")
	if firstErr == nil || secondErr == nil || firstErr.Error() != secondErr.Error() || !strings.Contains(firstErr.Error(), "orphan checker directory checkers/removed") {
		t.Fatalf("deleted manifest's stale outputs were not rejected deterministically:\nfirst: %v\nsecond: %v", firstErr, secondErr)
	}
	after, err := os.ReadFile(currentProjection)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("ProjectAll mutated a surviving projection before rejecting orphaned outputs")
	}

	if err := os.Remove(manifestPaths["current"]); err != nil {
		t.Fatal(err)
	}
	if _, err := ProjectAll(design, "v-test"); err == nil || !strings.Contains(err.Error(), "checker inventory is not exact") || !strings.Contains(err.Error(), "checkers/current") || !strings.Contains(err.Error(), "checkers/removed") {
		t.Fatalf("zero-manifest orphan inventory was not rejected as an exact-set failure: %v", err)
	}
}

func TestCommitProjectionPlansRollsBackAllTargets(t *testing.T) {
	dir := t.TempDir()
	one := filepath.Join(dir, "checkers", "one", "projection.json")
	two := filepath.Join(dir, "checkers", "two", "projection.json")
	if err := os.MkdirAll(filepath.Dir(one), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(two), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(one, []byte("old-one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(two, []byte("old-two"), 0o644); err != nil {
		t.Fatal(err)
	}
	plans := []plannedProjection{
		{dest: one, rendered: []byte("new-one")},
		{dest: two, rendered: []byte("new-two")},
	}
	failed := false
	rename := func(from, to string) error {
		if !failed && to == two && strings.Contains(filepath.Base(from), ".machinery-project-") && !strings.Contains(filepath.Base(from), "backup") {
			failed = true
			return errors.New("injected second install failure")
		}
		return nil
	}
	if err := commitProjectionPlansWithRename(dir, plans, rename); err == nil {
		t.Fatal("injected commit failure was ignored")
	}
	for path, want := range map[string]string{one: "old-one", two: "old-two"} {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Fatalf("%s was not rolled back: got %q want %q", path, got, want)
		}
	}
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(info.Name(), ".machinery-project-") || info.Name() == projectionControlDirName {
			t.Errorf("transaction left temporary/control artifact: %s", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestProjectionTransactionRecoversEveryDurablePhaseAcrossDirectories(t *testing.T) {
	tests := []struct {
		point     string
		committed bool
	}{
		{point: "prepared"},
		{point: "parked:0"},
		{point: "installed:0"},
		{point: "committed", committed: true},
	}
	for _, tt := range tests {
		t.Run(tt.point, func(t *testing.T) {
			design := t.TempDir()
			one := filepath.Join(design, "checkers", "one", "projection.json")
			two := filepath.Join(design, "checkers", "two", "projection.json")
			if err := os.MkdirAll(filepath.Dir(one), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(one, []byte("old-one"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(two), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(two, []byte("old-two"), 0o644); err != nil {
				t.Fatal(err)
			}
			plans := []plannedProjection{
				{dest: two, rendered: []byte("new-two")},
				{dest: one, rendered: []byte("new-one")},
			}
			err := commitProjectionPlansWithHooks(design, plans, projectionTransactionHooks{
				fault: func(point string) error {
					if point == tt.point {
						return errSimulatedProjectionCrash
					}
					return nil
				},
			})
			if !errors.Is(err, errSimulatedProjectionCrash) {
				t.Fatalf("fault %s returned %v", tt.point, err)
			}
			if err := recoverProjectionTransaction(design); err != nil {
				t.Fatal(err)
			}
			if tt.committed {
				assertProjectionContent(t, one, "new-one")
				assertProjectionContent(t, two, "new-two")
			} else {
				assertProjectionContent(t, one, "old-one")
				assertProjectionContent(t, two, "old-two")
			}
			if _, err := os.Lstat(filepath.Join(design, projectionControlDirName)); !os.IsNotExist(err) {
				t.Fatalf("recovery left transaction control state: %v", err)
			}
		})
	}
}

func TestProjectionRecoveryPreservesConcurrentEditAfterInstalledCrash(t *testing.T) {
	design := t.TempDir()
	target := filepath.Join(design, "checkers", "one", "projection.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := commitProjectionPlansWithHooks(design, []plannedProjection{{dest: target, rendered: []byte("transaction-new")}}, projectionTransactionHooks{
		fault: func(point string) error {
			if point == "installed:0" {
				return errSimulatedProjectionCrash
			}
			return nil
		},
	})
	if !errors.Is(err, errSimulatedProjectionCrash) {
		t.Fatalf("simulated crash returned %v", err)
	}
	if err := os.WriteFile(target, []byte("user-edit"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := recoverProjectionTransaction(design); err == nil || !strings.Contains(err.Error(), "preserving") {
		t.Fatalf("recovery did not fail closed on a concurrent edit: %v", err)
	}
	assertProjectionContent(t, target, "user-edit")
	if _, err := os.Lstat(filepath.Join(design, projectionControlDirName, projectionPreparedName)); err != nil {
		t.Fatalf("recovery discarded its durable retry evidence: %v", err)
	}
}

func TestProjectionRecoveryRejectsSameContentPathABA(t *testing.T) {
	design := t.TempDir()
	target := filepath.Join(design, "checkers", "one", "projection.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := commitProjectionPlansWithHooks(design, []plannedProjection{{dest: target, rendered: []byte("transaction-new")}}, projectionTransactionHooks{
		fault: func(point string) error {
			if point == "installed:0" {
				return errSimulatedProjectionCrash
			}
			return nil
		},
	})
	if !errors.Is(err, errSimulatedProjectionCrash) {
		t.Fatalf("simulated crash returned %v", err)
	}
	parkedPostImage := target + ".user-parked"
	if err := os.Rename(target, parkedPostImage); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("transaction-new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := recoverProjectionTransaction(design); err == nil || !strings.Contains(err.Error(), "preserving") {
		t.Fatalf("recovery accepted a same-content path ABA: %v", err)
	}
	assertProjectionContent(t, target, "transaction-new")
	assertProjectionContent(t, parkedPostImage, "transaction-new")
}

func TestProjectionRecoveryRevalidatesAtRemovalBoundary(t *testing.T) {
	design := t.TempDir()
	target := filepath.Join(design, "checkers", "one", "projection.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := commitProjectionPlansWithHooks(design, []plannedProjection{{dest: target, rendered: []byte("transaction-new")}}, projectionTransactionHooks{
		fault: func(point string) error {
			if point == "installed:0" {
				return errSimulatedProjectionCrash
			}
			return nil
		},
	})
	if !errors.Is(err, errSimulatedProjectionCrash) {
		t.Fatalf("simulated crash returned %v", err)
	}
	root, err := openDesignRoot(design)
	if err != nil {
		t.Fatal(err)
	}
	defer root.close()
	mutated := false
	err = recoverProjectionTransactionRootWithHooks(root, projectionTransactionHooks{
		beforeRemove: func(path string) error {
			if !mutated && path == target {
				mutated = true
				return os.WriteFile(target, []byte("boundary-edit"), 0o644)
			}
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "removal boundary") {
		t.Fatalf("recovery did not reject the removal-boundary mutation: %v", err)
	}
	assertProjectionContent(t, target, "boundary-edit")
}

func TestProjectionRecoveryRejectsControlRecordMutationAfterParse(t *testing.T) {
	mutations := []struct {
		name   string
		mutate func(*testing.T, string, []byte, os.FileInfo)
	}{
		{
			name: "path replacement",
			mutate: func(t *testing.T, path string, body []byte, _ os.FileInfo) {
				t.Helper()
				if err := os.Rename(path, path+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, body, 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "content replacement",
			mutate: func(t *testing.T, path string, body []byte, _ os.FileInfo) {
				t.Helper()
				changed := append([]byte(nil), body...)
				changed[0] ^= 1
				if err := os.WriteFile(path, changed, 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "same-byte metadata ABA",
			mutate: func(t *testing.T, path string, body []byte, before os.FileInfo) {
				t.Helper()
				if projectionControlChangeID(before) == "" {
					t.Skip("platform does not expose a control-record change identity")
				}
				if err := os.WriteFile(path, body, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, role := range []struct {
		name, record, point string
	}{
		{name: "prepared", record: projectionPreparedName, point: "prepared"},
		{name: "bound", record: projectionBoundName, point: "prepared"},
		{name: "committed", record: projectionCommittedName, point: "committed"},
	} {
		for _, mutation := range mutations {
			t.Run(role.name+"/"+mutation.name, func(t *testing.T) {
				design := t.TempDir()
				target := filepath.Join(design, "checkers", "one", "projection.json")
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
					t.Fatal(err)
				}
				err := commitProjectionPlansWithHooks(design, []plannedProjection{{dest: target, rendered: []byte("new")}}, projectionTransactionHooks{
					fault: func(point string) error {
						if point == role.point {
							return errSimulatedProjectionCrash
						}
						return nil
					},
				})
				if !errors.Is(err, errSimulatedProjectionCrash) {
					t.Fatalf("create %s recovery state: %v", role.name, err)
				}
				recordPath := filepath.Join(design, projectionControlDirName, role.record)
				body, err := os.ReadFile(recordPath)
				if err != nil {
					t.Fatal(err)
				}
				before, err := os.Lstat(recordPath)
				if err != nil {
					t.Fatal(err)
				}
				root, err := openDesignRoot(design)
				if err != nil {
					t.Fatal(err)
				}
				defer root.close()
				mutated := false
				err = recoverProjectionTransactionRootWithHooks(root, projectionTransactionHooks{
					beforeRemove: func(path string) error {
						if mutated || path == recordPath {
							return nil
						}
						mutated = true
						mutation.mutate(t, recordPath, body, before)
						return nil
					},
				})
				if err == nil || !strings.Contains(err.Error(), "authority changed") {
					t.Fatalf("post-parse %s mutation was accepted: %v", role.name, err)
				}
				if !mutated {
					t.Fatal("control-record mutation hook did not run")
				}
				if _, statErr := os.Lstat(recordPath); statErr != nil {
					t.Fatalf("foreign control record was not preserved: %v", statErr)
				}
			})
		}
	}
}

func TestProjectionRecoveryPreservesReplacedIncompleteControlRecord(t *testing.T) {
	for _, mutation := range []struct {
		name   string
		mutate func(*testing.T, string, []byte, os.FileInfo)
	}{
		{
			name: "path replacement",
			mutate: func(t *testing.T, path string, body []byte, _ os.FileInfo) {
				if err := os.Rename(path, path+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, body, 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "content replacement",
			mutate: func(t *testing.T, path string, body []byte, _ os.FileInfo) {
				if err := os.WriteFile(path, append(body, 'x'), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "same-byte metadata ABA",
			mutate: func(t *testing.T, path string, body []byte, before os.FileInfo) {
				if projectionControlChangeID(before) == "" {
					t.Skip("platform does not expose a control-record change identity")
				}
				if err := os.WriteFile(path, body, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			design := t.TempDir()
			control := filepath.Join(design, projectionControlDirName)
			if err := os.Mkdir(control, 0o700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(control, projectionPreparedName+".new")
			body := []byte("partial journal")
			if err := os.WriteFile(path, body, 0o600); err != nil {
				t.Fatal(err)
			}
			before, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			root, err := openDesignRoot(design)
			if err != nil {
				t.Fatal(err)
			}
			defer root.close()
			mutated := false
			err = recoverProjectionTransactionRootWithHooks(root, projectionTransactionHooks{
				beforeRemove: func(got string) error {
					if mutated || got != path {
						return nil
					}
					mutated = true
					mutation.mutate(t, path, body, before)
					return nil
				},
			})
			if err == nil || !strings.Contains(err.Error(), "changed at the removal boundary") {
				t.Fatalf("incomplete-record replacement was accepted: %v", err)
			}
			if !mutated {
				t.Fatal("incomplete-record mutation hook did not run")
			}
			if _, statErr := os.Lstat(path); statErr != nil {
				t.Fatalf("replacement incomplete record was not preserved: %v", statErr)
			}
		})
	}
}

func TestProjectionRecoveryRejectsDualAndExtraControlRecords(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files []string
		want  string
	}{
		{
			name:  "dual live and retired",
			files: []string{projectionPreparedName, projectionPreparedName + ".retired"},
			want:  "dual checker projection transaction controls",
		},
		{
			name:  "extra transaction control",
			files: []string{"checker-project-transaction.extra.json"},
			want:  "unknown checker projection transaction control path",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			control := filepath.Join(design, projectionControlDirName)
			if err := os.Mkdir(control, 0o700); err != nil {
				t.Fatal(err)
			}
			for _, name := range tc.files {
				if err := os.WriteFile(filepath.Join(control, name), []byte("foreign\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			err := recoverProjectionTransaction(design)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid control inventory was accepted: %v", err)
			}
			for _, name := range tc.files {
				if _, statErr := os.Lstat(filepath.Join(control, name)); statErr != nil {
					t.Fatalf("invalid control %s was not preserved: %v", name, statErr)
				}
			}
		})
	}
}

func TestProjectionRecoveryResumesPartialControlRetirement(t *testing.T) {
	for _, tc := range []struct {
		name   string
		retire []string
		remove []string
	}{
		{name: "one retired", retire: []string{projectionPreparedName}},
		{
			name:   "deletion in progress",
			retire: []string{projectionPreparedName, projectionBoundName, projectionCommittedName},
			remove: []string{projectionPreparedName, projectionBoundName},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			target := filepath.Join(design, "checkers", "one", "projection.json")
			err := commitProjectionPlansWithHooks(design, []plannedProjection{{dest: target, rendered: []byte("new")}}, projectionTransactionHooks{
				fault: func(point string) error {
					if point == "committed" {
						return errSimulatedProjectionCrash
					}
					return nil
				},
			})
			if !errors.Is(err, errSimulatedProjectionCrash) {
				t.Fatalf("create committed recovery state: %v", err)
			}
			control := filepath.Join(design, projectionControlDirName)
			for _, name := range tc.retire {
				if err := os.Rename(filepath.Join(control, name), filepath.Join(control, name+".retired")); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range tc.remove {
				if err := os.Remove(filepath.Join(control, name+".retired")); err != nil {
					t.Fatal(err)
				}
			}
			if err := recoverProjectionTransaction(design); err != nil {
				t.Fatalf("resume control retirement: %v", err)
			}
			assertProjectionContent(t, target, "new")
			if _, err := os.Lstat(control); !os.IsNotExist(err) {
				t.Fatalf("retirement recovery left control residue: %v", err)
			}
		})
	}
}

func TestProjectionRollbackPreservesConcurrentInstalledTargetOnLaterFailure(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func(*testing.T, string)
		want   string
	}{
		{
			name: "content edit",
			mutate: func(t *testing.T, target string) {
				t.Helper()
				if err := os.WriteFile(target, []byte("user-edit"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			want: "user-edit",
		},
		{
			name: "same-content path ABA",
			mutate: func(t *testing.T, target string) {
				t.Helper()
				if err := os.Rename(target, target+".user-parked"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, []byte("new-one"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			want: "new-one",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			design := t.TempDir()
			one := filepath.Join(design, "checkers", "one", "projection.json")
			two := filepath.Join(design, "checkers", "two", "projection.json")
			for path, body := range map[string]string{one: "old-one", two: "old-two"} {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			injected := false
			err := commitProjectionPlansWithHooks(design, []plannedProjection{
				{dest: one, rendered: []byte("new-one")},
				{dest: two, rendered: []byte("new-two")},
			}, projectionTransactionHooks{
				beforeRename: func(from, to string) error {
					if !injected && to == two && strings.Contains(filepath.Base(from), ".machinery-project-stage-") {
						injected = true
						tt.mutate(t, one)
						return errors.New("injected later projection failure")
					}
					return nil
				},
			})
			if err == nil || !strings.Contains(err.Error(), "preserving") {
				t.Fatalf("ordinary rollback did not fail closed: %v", err)
			}
			assertProjectionContent(t, one, tt.want)
			assertProjectionContent(t, two, "old-two")
			if _, err := os.Lstat(filepath.Join(design, projectionControlDirName, projectionBoundName)); err != nil {
				t.Fatalf("rollback discarded the exact post-image journal: %v", err)
			}
		})
	}
}

func TestProjectionTransactionCommittedRecordRecoversWithoutPreparedRecord(t *testing.T) {
	design := t.TempDir()
	target := filepath.Join(design, "checkers", "one", "projection.json")
	err := commitProjectionPlansWithHooks(design, []plannedProjection{{dest: target, rendered: []byte("new")}}, projectionTransactionHooks{
		fault: func(point string) error {
			if point == "committed" {
				return errSimulatedProjectionCrash
			}
			return nil
		},
	})
	if !errors.Is(err, errSimulatedProjectionCrash) {
		t.Fatalf("committed fault returned %v", err)
	}
	control := filepath.Join(design, projectionControlDirName)
	if err := os.Remove(filepath.Join(control, projectionPreparedName)); err != nil {
		t.Fatal(err)
	}
	if err := recoverProjectionTransaction(design); err != nil {
		t.Fatal(err)
	}
	assertProjectionContent(t, target, "new")
}

func TestProjectAllRecoversUncommittedTransactionImmediatelyAfterLock(t *testing.T) {
	design := t.TempDir()
	one := filepath.Join(design, "checkers", "one", "projection.json")
	two := filepath.Join(design, "checkers", "two", "projection.json")
	if err := os.MkdirAll(filepath.Dir(one), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(one, []byte("old-one"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := commitProjectionPlansWithHooks(design, []plannedProjection{
		{dest: one, rendered: []byte("partial-new-one")},
		{dest: two, rendered: []byte("partial-new-two")},
	}, projectionTransactionHooks{
		fault: func(point string) error {
			if point == "installed:0" {
				return errSimulatedProjectionCrash
			}
			return nil
		},
	})
	if !errors.Is(err, errSimulatedProjectionCrash) {
		t.Fatalf("fault returned %v", err)
	}
	if err := os.WriteFile(filepath.Join(design, "broken.modelith.yaml"), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ProjectAll(design, "v-test"); err == nil {
		t.Fatal("malformed model unexpectedly projected")
	}
	assertProjectionContent(t, one, "old-one")
	if _, err := os.Lstat(two); !os.IsNotExist(err) {
		t.Fatalf("ProjectAll did not recover absent target before model validation: %v", err)
	}
}

func TestProjectionTransactionRejectsEscapingSymlinkWithoutTouchingSentinel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require elevated Windows privileges")
	}
	design := t.TempDir()
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("outside-safe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(design, "escape")); err != nil {
		t.Fatal(err)
	}
	record := projectionTransactionRecord{
		Version: projectionTransactionVersion,
		Phase:   "prepared",
		Entries: []projectionTransactionEntry{{
			Target: "escape/sentinel", Stage: "escape/.machinery-project-stage-deadbeef", Backup: "escape/.machinery-project-backup-deadbeef", Existed: true,
		}},
	}
	writeRawProjectionJournal(t, design, record)
	if err := recoverProjectionTransaction(design); err == nil || !strings.Contains(err.Error(), "symlink component") {
		t.Fatalf("escaping journal was accepted: %v", err)
	}
	assertProjectionContent(t, sentinel, "outside-safe")
}

func TestRootedReadRejectsParentSwapWithoutReadingOutside(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require elevated Windows privileges")
	}
	design := t.TempDir()
	parent := filepath.Join(design, "checkers", "privacy")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "projection.json"), []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "projection.json")
	if err := os.WriteFile(sentinel, []byte("outside-safe"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := openDesignRoot(design)
	if err != nil {
		t.Fatal(err)
	}
	defer root.close()
	if err := os.Rename(parent, parent+"-parked"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, parent); err != nil {
		t.Fatal(err)
	}
	if _, err := root.readRegular("checkers/privacy/projection.json", "projection", false); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("rooted read followed swapped parent: %v", err)
	}
	assertProjectionContent(t, sentinel, "outside-safe")
}

func TestProjectionTransactionRejectsParentSwapWithoutWritingOutside(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require elevated Windows privileges")
	}
	design := t.TempDir()
	parent := filepath.Join(design, "checkers", "privacy")
	target := filepath.Join(parent, "projection.json")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("inside-old"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "projection.json")
	if err := os.WriteFile(sentinel, []byte("outside-safe"), 0o644); err != nil {
		t.Fatal(err)
	}
	swapped := false
	err := commitProjectionPlansWithHooks(design, []plannedProjection{{dest: target, rendered: []byte("inside-new")}}, projectionTransactionHooks{
		beforeRename: func(_, _ string) error {
			if swapped {
				return nil
			}
			swapped = true
			if err := os.Rename(parent, parent+"-parked"); err != nil {
				return err
			}
			return os.Symlink(outside, parent)
		},
	})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("transaction followed swapped parent: %v", err)
	}
	assertProjectionContent(t, sentinel, "outside-safe")
}

func TestProjectionRecoveryRejectsParentSwapWithoutWritingOutside(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require elevated Windows privileges")
	}
	design := t.TempDir()
	parent := filepath.Join(design, "checkers", "privacy")
	target := filepath.Join(parent, "projection.json")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("inside-old"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := commitProjectionPlansWithHooks(design, []plannedProjection{{dest: target, rendered: []byte("inside-new")}}, projectionTransactionHooks{
		fault: func(point string) error {
			if point == "installed:0" {
				return errSimulatedProjectionCrash
			}
			return nil
		},
	})
	if !errors.Is(err, errSimulatedProjectionCrash) {
		t.Fatalf("simulated crash returned %v", err)
	}
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "projection.json")
	if err := os.WriteFile(sentinel, []byte("outside-safe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(parent, parent+"-parked"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, parent); err != nil {
		t.Fatal(err)
	}
	if err := recoverProjectionTransaction(design); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("recovery followed swapped parent: %v", err)
	}
	assertProjectionContent(t, sentinel, "outside-safe")
}

func TestProjectionControlCleanupSyncFailureIsReportedAndRetryable(t *testing.T) {
	design := t.TempDir()
	root, err := openDesignRoot(design)
	if err != nil {
		t.Fatal(err)
	}
	defer root.close()
	if err := root.root.Mkdir(projectionControlDirName, 0o700); err != nil {
		t.Fatal(err)
	}
	syncErr := errors.New("injected design directory sync failure")
	root.syncDirHook = func(rel string) error {
		if rel == "." {
			return syncErr
		}
		return nil
	}
	if err := removeProjectionControlDirIfEmpty(root, projectionControlDirName); !errors.Is(err, syncErr) {
		t.Fatalf("cleanup discarded sync failure: %v", err)
	}
	if info, err := root.root.Lstat(projectionControlDirName); err != nil || !info.IsDir() {
		t.Fatalf("cleanup did not retain a retry point: info=%v err=%v", info, err)
	}
	root.syncDirHook = nil
	if err := removeProjectionControlDirIfEmpty(root, projectionControlDirName); err != nil {
		t.Fatalf("retry cleanup failed: %v", err)
	}
	if _, err := root.root.Lstat(projectionControlDirName); !os.IsNotExist(err) {
		t.Fatalf("retry left control directory: %v", err)
	}
}

func TestRootedDirectorySyncRejectsParentSwap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require elevated Windows privileges")
	}
	design := t.TempDir()
	parent := filepath.Join(design, "checkers", "privacy")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("outside-safe"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := openDesignRoot(design)
	if err != nil {
		t.Fatal(err)
	}
	defer root.close()
	root.syncDirHook = func(rel string) error {
		root.syncDirHook = nil
		if rel != "checkers/privacy" {
			return fmt.Errorf("unexpected sync path %s", rel)
		}
		if err := os.Rename(parent, parent+"-parked"); err != nil {
			return err
		}
		return os.Symlink(outside, parent)
	}
	if err := root.syncDir("checkers/privacy"); err == nil {
		t.Fatal("directory sync reopened a swapped ambient path")
	}
	assertProjectionContent(t, sentinel, "outside-safe")
}

func TestProjectionTransactionRejectsMalformedSymlinkAndNonRegularJournal(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, string, string)
	}{
		{
			name: "malformed",
			setup: func(t *testing.T, _ string, journal string) {
				if err := os.WriteFile(journal, []byte("{not-json"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "symlink",
			setup: func(t *testing.T, design, journal string) {
				if runtime.GOOS == "windows" {
					t.Skip("symlink creation may require elevated Windows privileges")
				}
				outside := filepath.Join(t.TempDir(), "journal")
				if err := os.WriteFile(outside, []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, journal); err != nil {
					t.Fatal(err)
				}
				_ = design
			},
		},
		{
			name: "non-regular",
			setup: func(t *testing.T, _ string, journal string) {
				if err := os.Mkdir(journal, 0o700); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			design := t.TempDir()
			control := filepath.Join(design, projectionControlDirName)
			if err := os.Mkdir(control, 0o700); err != nil {
				t.Fatal(err)
			}
			journal := filepath.Join(control, projectionPreparedName)
			tt.setup(t, design, journal)
			if err := recoverProjectionTransaction(design); err == nil {
				t.Fatal("unsafe transaction journal was accepted")
			}
		})
	}
}

func assertProjectionContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

func writeRawProjectionJournal(t *testing.T, design string, record projectionTransactionRecord) {
	t.Helper()
	control := filepath.Join(design, projectionControlDirName)
	if err := os.Mkdir(control, 0o700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(control, projectionPreparedName), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestProjectAllRejectsPortableManifestAliasesBeforeWriting(t *testing.T) {
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "d.modelith.yaml"), []byte(sampleModel), 0o644); err != nil {
		t.Fatal(err)
	}
	checkers := filepath.Join(design, "checkers")
	if err := os.MkdirAll(filepath.Join(checkers, "Privacy"), 0o755); err != nil {
		t.Fatal(err)
	}
	firstOut := filepath.Join(checkers, "Privacy", "projection.json")
	if err := os.WriteFile(firstOut, []byte("sentinel"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifests := map[string]string{
		"a.checker.yaml": "checker: {id: Privacy, runtime_closure: sha256:1111111111111111111111111111111111111111111111111111111111111111}\nprojection: {include: [model]}\nevidence: {projection_out: checkers/Privacy/projection.json, evidence_in: checkers/Privacy/evidence.json}\n",
		"b.checker.yaml": "checker: {id: privacy, runtime_closure: sha256:1111111111111111111111111111111111111111111111111111111111111111}\nprojection: {include: [model]}\nevidence: {projection_out: checkers/privacy/projection.json, evidence_in: checkers/privacy/evidence.json}\n",
	}
	for name, body := range manifests {
		if err := os.WriteFile(filepath.Join(checkers, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ProjectAll(design, "v-test"); err == nil || !strings.Contains(err.Error(), "case-insensitive") {
		t.Fatalf("portable checker id collision was accepted: %v", err)
	}
	got, err := os.ReadFile(firstOut)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "sentinel" {
		t.Fatalf("project wrote before global validation: %q", got)
	}
}

func TestProjectAllHonorsDesignScopedAdvisoryLock(t *testing.T) {
	design := t.TempDir()
	lock, err := filelock.Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if _, err := ProjectAll(design, "v-test"); err == nil || !strings.Contains(err.Error(), "holds the lock") {
		t.Fatalf("concurrent projection transaction was not rejected: %v", err)
	}
}

func TestProjectAllNoModelErrors(t *testing.T) {
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "checkers"), 0o755); err != nil {
		t.Fatal(err)
	}
	// a manifest but no *.modelith.yaml to project from
	man := "checker: {id: test, runtime_closure: sha256:1111111111111111111111111111111111111111111111111111111111111111}\nprojection: {include: [model]}\nevidence: {projection_out: checkers/test/projection.json, evidence_in: checkers/test/evidence.json}\n"
	if err := os.WriteFile(filepath.Join(design, "checkers", "test.checker.yaml"), []byte(man), 0o644); err != nil {
		t.Fatal(err)
	}
	_, firstErr := ProjectAll(design, "v0")
	_, secondErr := ProjectAll(design, "v0")
	if firstErr == nil {
		t.Fatal("expected an error projecting with no domain model")
	}
	if secondErr == nil || secondErr.Error() != firstErr.Error() || strings.Contains(firstErr.Error(), "machinery-design-source-") {
		t.Fatalf("invalid-input diagnostic is unstable or leaks private source root:\nfirst: %v\nsecond: %v", firstErr, secondErr)
	}
}

func TestProjectAllRejectsMultipleModels(t *testing.T) {
	design := t.TempDir()
	for _, name := range []string{"a.modelith.yaml", "b.modelith.yaml"} {
		if err := os.WriteFile(filepath.Join(design, name), []byte(sampleModel), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(design, "checkers"), 0o755); err != nil {
		t.Fatal(err)
	}
	man := "checker: {id: test, runtime_closure: sha256:1111111111111111111111111111111111111111111111111111111111111111}\nprojection: {include: [model]}\nevidence: {projection_out: checkers/test/projection.json, evidence_in: checkers/test/evidence.json}\n"
	if err := os.WriteFile(filepath.Join(design, "checkers", "test.checker.yaml"), []byte(man), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ProjectAll(design, "v0"); err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("multiple models were not rejected exactly: %v", err)
	}
}

func TestProjectAllPropagatesUnsupportedLayer(t *testing.T) {
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "d.modelith.yaml"), []byte(sampleModel), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(design, "checkers"), 0o755); err != nil {
		t.Fatal(err)
	}
	man := "checker: {id: test, runtime_closure: sha256:1111111111111111111111111111111111111111111111111111111111111111}\nprojection: {include: [model, machines]}\nevidence: {projection_out: checkers/test/projection.json, evidence_in: checkers/test/evidence.json}\n"
	if err := os.WriteFile(filepath.Join(design, "checkers", "test.checker.yaml"), []byte(man), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ProjectAll(design, "v0"); err == nil {
		t.Fatal("expected an error from an unsupported include layer")
	}
}
