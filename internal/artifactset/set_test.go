package artifactset

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestTransactionHashIsBoundedAgainstContinuousAppender(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "growing")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, syncFile, err := txOpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()     //nolint:errcheck // test cleanup
	defer syncFile.Close() //nolint:errcheck // test cleanup
	stop := make(chan struct{})
	appenderDone := make(chan error, 1)
	ops := txDefaultOps(nil)
	ops.root, ops.syncFile = root, syncFile
	ops.hashAfterOpen = func(string) error {
		appender, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			return err
		}
		chunk := bytes.Repeat([]byte("a"), 64<<10)
		if _, err := appender.Write(chunk); err != nil {
			_ = appender.Close()
			return err
		}
		go func() {
			var appendErr error
			defer func() { appenderDone <- errors.Join(appendErr, appender.Close()) }()
			for {
				select {
				case <-stop:
					return
				default:
					if _, err := appender.Write(chunk); err != nil {
						appendErr = err
						return
					}
				}
			}
		}()
		return nil
	}
	result := make(chan error, 1)
	go func() {
		_, _, err := txHashPath("growing", "growing artifact", ops)
		result <- err
	}()
	select {
	case err := <-result:
		close(stop)
		if appendErr := <-appenderDone; appendErr != nil {
			t.Fatal(appendErr)
		}
		if err == nil || !strings.Contains(err.Error(), "changed") {
			t.Fatalf("continuous growth was accepted: %v", err)
		}
	case <-time.After(2 * time.Second):
		close(stop)
		<-appenderDone
		t.Fatal("hash followed a continuous appender instead of stopping at the witnessed size")
	}
}

func TestTransactionDirectoryEnumerationStopsAtFixedCeiling(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a", "b", "c"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	root, syncFile, err := txOpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()     //nolint:errcheck // test cleanup
	defer syncFile.Close() //nolint:errcheck // test cleanup
	_, err = txReadDirBounded(txOps{root: root}, 2)
	if err == nil || !strings.Contains(err.Error(), "2-entry limit") {
		t.Fatalf("over-limit directory inventory was accepted: %v", err)
	}
}

func TestArtifactTransactionRejectsItemInventoryBeforeAllocation(t *testing.T) {
	remove := make([]string, txMaxItemCount+1)
	_, _, err := txValidateReconcileTargets(txOps{}, nil, remove)
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("%d-item limit", txMaxItemCount)) {
		t.Fatalf("over-limit transaction item inventory was accepted: %v", err)
	}
}

func TestCommitRejectsPortableAlias(t *testing.T) {
	err := Commit(t.TempDir(), map[string][]byte{"Foo.tla": []byte("a"), "foo.tla": []byte("b")})
	if err == nil || !strings.Contains(err.Error(), "portable") {
		t.Fatalf("case-fold alias accepted: %v", err)
	}
}

func TestInspectRemovalCandidateRejectsSparseOversize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stale.txt")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(artifactRemovalCandidateMaxBytes + 1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	_, _, err = InspectRemovalCandidate(dir, "stale.txt")
	want := fmt.Sprintf("exceeds %d-byte limit", artifactRemovalCandidateMaxBytes)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("oversized sparse candidate was accepted: %v", err)
	}
}

func TestInspectRemovalCandidateRejectsGrowthBeyondStreamLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stale.txt")
	writeBytes(t, path, bytes.Repeat([]byte("a"), 16))
	const testLimit int64 = 32
	grew := false

	_, _, err := inspectRemovalCandidate(dir, "stale.txt", testLimit, func() error {
		grew = true
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			return err
		}
		_, writeErr := file.Write(bytes.Repeat([]byte("b"), 32))
		return errors.Join(writeErr, file.Close())
	})
	if !grew {
		t.Fatal("growth injection did not run")
	}
	want := fmt.Sprintf("exceeds %d-byte limit", testLimit)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("candidate growth beyond bounded stream was accepted: %v", err)
	}
}

func TestReconcileGuardedRootedRejectsReplacementBeforeAnyMutation(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{"A": "owned-old", "B": "other-old"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	_, condition, err := InspectRemovalCandidate(dir, "A")
	if err != nil {
		t.Fatal(err)
	}
	replacement := filepath.Join(dir, "replacement")
	if err := os.WriteFile(replacement, []byte("foreign"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, filepath.Join(dir, "A")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	err = ReconcileGuardedRooted(dir, root, map[string][]byte{"A": []byte("new-a"), "B": []byte("new-b")}, nil, []RemovalPrecondition{condition})
	if err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("guarded replacement was accepted: %v", err)
	}
	for name, want := range map[string]string{"A": "foreign", "B": "other-old"} {
		got, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil || string(got) != want {
			t.Fatalf("%s mutated: %q, %v", name, got, readErr)
		}
	}
}

func TestCommitRollsBackEveryFile(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{"A": "old-a", "B": "old-b"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	installs := 0
	rename := func(old, new string) error {
		if strings.HasPrefix(filepath.Base(old), ".machinery-artifact-new-") {
			installs++
			if installs == 2 {
				return fmt.Errorf("injected failure")
			}
		}
		return nil
	}
	err := commitWithRename(dir, map[string][]byte{"A": []byte("new-a"), "B": []byte("new-b")}, rename)
	if err == nil {
		t.Fatal("injected failure succeeded")
	}
	for name, want := range map[string]string{"A": "old-a", "B": "old-b"} {
		got, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil || string(got) != want {
			t.Fatalf("%s not restored: %q, %v", name, got, readErr)
		}
	}
}

func TestCommitRejectsSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("sentinel"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "A")); err != nil {
		t.Fatal(err)
	}
	if err := Commit(dir, map[string][]byte{"A": []byte("new")}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink target accepted: %v", err)
	}
}

func TestCommitRecoversEveryDurableBoundary(t *testing.T) {
	points := []string{
		"prepared-journal",
		"park:A",
		"park:B",
		"install:A",
		"install:B",
		"journal-update-isolated",
		"journal-update-installed",
		"committed-journal",
		"journal-isolated",
		"commit-cleanup:A",
		"commit-cleanup:B",
		"journal-retired",
		"journal-quarantined",
		"journal-quarantine-delete",
		"journal-cleanup",
	}
	for _, point := range points {
		t.Run(point, func(t *testing.T) {
			dir := t.TempDir()
			outside := filepath.Join(t.TempDir(), "outside")
			writeFiles(t, dir, map[string]string{"A": "old-a", "B": "old-b"})
			writeFiles(t, filepath.Dir(outside), map[string]string{filepath.Base(outside): "sentinel"})

			injected := errors.New("power loss")
			ops := txDefaultOps(nil)
			ops.faultAfter = func(got string) error {
				if got == point {
					return injected
				}
				return nil
			}
			err := txCommit(dir, map[string][]byte{"A": []byte("new-a"), "B": []byte("new-b")}, ops)
			var crash *txCrash
			if !errors.As(err, &crash) || !errors.Is(err, injected) {
				t.Fatalf("fault point %s did not simulate a crash: %v", point, err)
			}
			if err := Commit(dir, map[string][]byte{}); err != nil {
				t.Fatalf("recover after %s: %v", point, err)
			}

			wantA, wantB := "old-a", "old-b"
			if point == "journal-update-installed" || point == "committed-journal" || point == "journal-isolated" || strings.HasPrefix(point, "commit-cleanup:") || point == "journal-retired" || point == "journal-quarantined" || point == "journal-quarantine-delete" || point == "journal-cleanup" {
				wantA, wantB = "new-a", "new-b"
			}
			assertFile(t, filepath.Join(dir, "A"), wantA)
			assertFile(t, filepath.Join(dir, "B"), wantB)
			assertFile(t, outside, "sentinel")
			assertNoTransactionFiles(t, dir)
		})
	}
}

func TestReconcileDeletionRecoversEveryDurableBoundary(t *testing.T) {
	points := []string{
		"prepared-journal",
		"park:A",
		"park:Z",
		"install:A",
		"install:Z",
		"journal-update-isolated",
		"journal-update-installed",
		"committed-journal",
		"journal-isolated",
		"commit-cleanup:A",
		"commit-cleanup:Z",
		"journal-retired",
		"journal-quarantined",
		"journal-quarantine-delete",
		"journal-cleanup",
	}
	for _, point := range points {
		t.Run(point, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"A": "old-a", "Z": "obsolete"})
			ops := txDefaultOps(nil)
			ops.faultAfter = func(got string) error {
				if got == point {
					return errors.New("power loss")
				}
				return nil
			}
			if err := txReconcile(dir, map[string][]byte{"A": []byte("new-a")}, []string{"Z"}, ops); err == nil {
				t.Fatal("injected crash succeeded")
			}
			if err := Commit(dir, map[string][]byte{}); err != nil {
				t.Fatalf("recover after %s: %v", point, err)
			}
			committed := point == "journal-update-installed" || point == "committed-journal" || point == "journal-isolated" || strings.HasPrefix(point, "commit-cleanup:") || point == "journal-retired" || point == "journal-quarantined" || point == "journal-quarantine-delete" || point == "journal-cleanup"
			if committed {
				assertFile(t, filepath.Join(dir, "A"), "new-a")
				if _, err := os.Lstat(filepath.Join(dir, "Z")); !os.IsNotExist(err) {
					t.Fatalf("obsolete target restored after committed recovery: %v", err)
				}
			} else {
				assertFile(t, filepath.Join(dir, "A"), "old-a")
				assertFile(t, filepath.Join(dir, "Z"), "obsolete")
			}
			assertNoTransactionFiles(t, dir)
		})
	}
}

func TestReconcileDeletionIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"A": "old", "Z": "obsolete"})
	for i := 0; i < 2; i++ {
		if err := Reconcile(dir, map[string][]byte{"A": []byte("new")}, []string{"Z"}); err != nil {
			t.Fatalf("reconcile %d: %v", i, err)
		}
	}
	assertFile(t, filepath.Join(dir, "A"), "new")
	if _, err := os.Lstat(filepath.Join(dir, "Z")); !os.IsNotExist(err) {
		t.Fatalf("obsolete target remains: %v", err)
	}
	assertNoTransactionFiles(t, dir)
}

func TestReconcilePlannedRejectsAtomicReplacementWithoutFinalMutation(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"A": "old-a", "Z": "owned-stale"})
	body, condition, err := InspectRemovalCandidate(dir, "Z")
	if err != nil || string(body) != "owned-stale" {
		t.Fatalf("inspect removal candidate: body=%q err=%v", body, err)
	}
	replacement := filepath.Join(dir, "replacement")
	writeBytes(t, replacement, []byte("owned-stale"))
	if err := os.Rename(replacement, filepath.Join(dir, "Z")); err != nil {
		t.Fatal(err)
	}
	err = ReconcilePlanned(dir, map[string][]byte{"A": []byte("new-a")}, []RemovalPrecondition{condition})
	if err == nil || !strings.Contains(err.Error(), "no longer matches its ownership plan") {
		t.Fatalf("atomic replacement was accepted: %v", err)
	}
	assertFile(t, filepath.Join(dir, "A"), "old-a")
	assertFile(t, filepath.Join(dir, "Z"), "owned-stale")
	assertNoTransactionFiles(t, dir)
}

func TestReconcilePlannedRejectsManufacturedConditionsWithoutMutation(t *testing.T) {
	for _, condition := range []RemovalPrecondition{{}, {Name: "Z"}} {
		dir := t.TempDir()
		writeFiles(t, dir, map[string]string{"A": "old-a", "Z": "owned-stale"})
		err := ReconcilePlanned(dir, map[string][]byte{"A": []byte("new-a")}, []RemovalPrecondition{condition})
		if err == nil || !strings.Contains(err.Error(), "lacks an inspected identity") {
			t.Fatalf("manufactured condition %+v was accepted: %v", condition, err)
		}
		assertFile(t, filepath.Join(dir, "A"), "old-a")
		assertFile(t, filepath.Join(dir, "Z"), "owned-stale")
		assertNoTransactionFiles(t, dir)
	}
}

func TestCommitRootedCannotBeRedirectedByParentSwap(t *testing.T) {
	outer := t.TempDir()
	dir := filepath.Join(outer, "output")
	parked := filepath.Join(outer, "parked")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := os.Rename(dir, parked); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeBytes(t, filepath.Join(dir, "outside-sentinel"), []byte("outside"))
	if err := CommitRooted(dir, root, map[string][]byte{"generated.txt": []byte("generated")}); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dir, "outside-sentinel"), "outside")
	if _, err := os.Lstat(filepath.Join(dir, "generated.txt")); !os.IsNotExist(err) {
		t.Fatalf("rooted commit escaped into replacement directory: %v", err)
	}
	assertFile(t, filepath.Join(parked, "generated.txt"), "generated")
}

func TestRollbackRecoveryIsRestartableAtEveryMutation(t *testing.T) {
	tests := []struct {
		name, forwardPoint, recoveryPoint string
	}{
		{"remove installed target", "install:B", "rollback-remove-target:B"},
		{"restore backup", "park:B", "rollback-restore:B"},
		{"remove staged temp", "park:B", "rollback-temp:B"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"A": "old-a", "B": "old-b"})
			crashCommitAt(t, dir, tc.forwardPoint)

			ops := txDefaultOps(nil)
			ops.faultAfter = func(point string) error {
				if point == tc.recoveryPoint {
					return errors.New("recovery power loss")
				}
				return nil
			}
			if err := txCommit(dir, map[string][]byte{}, ops); err == nil {
				t.Fatalf("recovery fault %s did not fire", tc.recoveryPoint)
			}
			if err := Commit(dir, map[string][]byte{}); err != nil {
				t.Fatalf("restart recovery after %s: %v", tc.recoveryPoint, err)
			}
			assertFile(t, filepath.Join(dir, "A"), "old-a")
			assertFile(t, filepath.Join(dir, "B"), "old-b")
			assertNoTransactionFiles(t, dir)
		})
	}
}

func TestPreparedRecoveryIsRestartableAfterJournalIsolation(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"A": "old-a", "B": "old-b"})
	crashCommitAt(t, dir, "install:B")

	ops := txDefaultOps(nil)
	ops.faultAfter = func(point string) error {
		if point == "journal-isolated" {
			return errors.New("recovery power loss")
		}
		return nil
	}
	if err := txCommit(dir, map[string][]byte{}, ops); err == nil {
		t.Fatal("journal-isolated recovery fault did not fire")
	}
	if _, err := os.Lstat(filepath.Join(dir, txRecoveryName)); err != nil {
		t.Fatalf("isolated journal was not retained: %v", err)
	}
	if err := Commit(dir, map[string][]byte{}); err != nil {
		t.Fatalf("restart recovery after journal isolation: %v", err)
	}
	assertFile(t, filepath.Join(dir, "A"), "old-a")
	assertFile(t, filepath.Join(dir, "B"), "old-b")
	assertNoTransactionFiles(t, dir)
}

func TestRollbackRejectsBehindCursorContentEditWithoutDeletingIt(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"A": "old-a", "B": "old-b"})
	crashCommitAt(t, dir, "install:B")

	mutated := false
	ops := txDefaultOps(nil)
	ops.faultAfter = func(point string) error {
		if point != "rollback-restore:B" || mutated {
			return nil
		}
		mutated = true
		return os.WriteFile(filepath.Join(dir, "A"), []byte("foreign-a"), 0o644)
	}
	err := txCommit(dir, map[string][]byte{}, ops)
	if err == nil || !strings.Contains(err.Error(), "rollback target A changed content since preflight") {
		t.Fatalf("behind-cursor content edit was not rejected: %v", err)
	}
	if !mutated {
		t.Fatal("behind-cursor mutation hook did not run")
	}
	assertFile(t, filepath.Join(dir, "A"), "foreign-a")
	assertFile(t, filepath.Join(dir, "B"), "old-b")
	if !recoveryJournalExists(t, dir) {
		t.Fatalf("failed-closed rollback discarded its recovery journal: %v", err)
	}
}

func TestRollbackRejectsBehindCursorABAWithoutDeletingIt(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"A": "old-a", "B": "old-b"})
	crashCommitAt(t, dir, "install:B")

	var replacementInfo os.FileInfo
	mutated := false
	ops := txDefaultOps(nil)
	ops.faultAfter = func(point string) error {
		if point != "rollback-restore:B" || mutated {
			return nil
		}
		mutated = true
		replacement := filepath.Join(dir, "replacement")
		if err := os.WriteFile(replacement, []byte("new-a"), 0o644); err != nil {
			return err
		}
		var err error
		replacementInfo, err = os.Lstat(replacement)
		if err != nil {
			return err
		}
		if err := os.Remove(filepath.Join(dir, "A")); err != nil {
			return err
		}
		return os.Rename(replacement, filepath.Join(dir, "A"))
	}
	err := txCommit(dir, map[string][]byte{}, ops)
	if err == nil || !strings.Contains(err.Error(), "rollback target A changed identity or mode since preflight") {
		t.Fatalf("behind-cursor ABA replacement was not rejected: %v", err)
	}
	if !mutated || replacementInfo == nil {
		t.Fatal("behind-cursor ABA mutation hook did not run")
	}
	assertFile(t, filepath.Join(dir, "A"), "new-a")
	currentInfo, err := os.Lstat(filepath.Join(dir, "A"))
	if err != nil || !os.SameFile(replacementInfo, currentInfo) {
		t.Fatalf("foreign ABA replacement identity was not preserved: current=%v err=%v", currentInfo, err)
	}
	assertFile(t, filepath.Join(dir, "B"), "old-b")
	if !recoveryJournalExists(t, dir) {
		t.Fatalf("failed-closed rollback discarded its recovery journal: %v", err)
	}
}

func TestCommittedFinalizeRejectsBehindCursorBackupABA(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"A": "old-a", "B": "old-b"})
	crashCommitAt(t, dir, "committed-journal")
	item := readJournalItem(t, dir, "B")

	var replacementInfo os.FileInfo
	mutated := false
	ops := txDefaultOps(nil)
	ops.faultAfter = func(point string) error {
		if point != "commit-cleanup:A" || mutated {
			return nil
		}
		mutated = true
		var err error
		replacementInfo, err = replaceSameContent(t, filepath.Join(dir, item.Backup), []byte("old-b"))
		return err
	}
	err := txCommit(dir, map[string][]byte{}, ops)
	if err == nil || !strings.Contains(err.Error(), "committed backup") || !strings.Contains(err.Error(), "changed identity") {
		t.Fatalf("behind-cursor committed backup ABA was not rejected: %v", err)
	}
	if !mutated || replacementInfo == nil {
		t.Fatal("behind-cursor finalization mutation did not run")
	}
	assertFile(t, filepath.Join(dir, item.Backup), "old-b")
	current, statErr := os.Lstat(filepath.Join(dir, item.Backup))
	if statErr != nil || !os.SameFile(replacementInfo, current) {
		t.Fatalf("foreign backup replacement was not preserved: current=%v err=%v", current, statErr)
	}
	if !recoveryJournalExists(t, dir) {
		t.Fatalf("failed-closed finalization discarded recovery journal: %v", statErr)
	}
}

func TestRecoveryRejectsSameContentABAForEveryJournaledRole(t *testing.T) {
	tests := []struct {
		name  string
		point string
		path  func(txItem) string
		body  []byte
	}{
		{name: "target", point: "prepared-journal", path: func(item txItem) string { return item.Target }, body: []byte("old-a")},
		{name: "temp", point: "prepared-journal", path: func(item txItem) string { return item.Temp }, body: []byte("new-a")},
		{name: "backup", point: "park:A", path: func(item txItem) string { return item.Backup }, body: []byte("old-a")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"A": "old-a", "B": "old-b"})
			crashCommitAt(t, dir, tc.point)
			item := readJournalItem(t, dir, "A")
			path := filepath.Join(dir, tc.path(item))
			replacementInfo, err := replaceSameContent(t, path, tc.body)
			if err != nil {
				t.Fatal(err)
			}
			err = Commit(dir, map[string][]byte{})
			if err == nil || !strings.Contains(err.Error(), "native identity mismatch") {
				t.Fatalf("same-content %s ABA was accepted: %v", tc.name, err)
			}
			current, statErr := os.Lstat(path)
			if statErr != nil || !os.SameFile(replacementInfo, current) {
				t.Fatalf("foreign %s replacement was not preserved: current=%v err=%v", tc.name, current, statErr)
			}
			if !recoveryJournalExists(t, dir) {
				t.Fatalf("failed-closed %s recovery discarded journal: %v", tc.name, statErr)
			}
		})
	}
}

func TestCommitFsyncsEveryRenameAndRemoveBoundary(t *testing.T) {
	dir := t.TempDir()
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	writeFiles(t, dir, map[string]string{"A": "old-a", "B": "old-b"})
	ops := txDefaultOps(nil)
	syncs := 0
	ops.syncObserve = func(path string) {
		if path != resolvedDir {
			t.Fatalf("synced outside transaction directory: %s", path)
		}
		syncs++
	}
	if err := txCommit(dir, map[string][]byte{"A": []byte("new-a"), "B": []byte("new-b")}, ops); err != nil {
		t.Fatal(err)
	}
	// prepared journal, two parks, two installs, committed journal, journal
	// isolation, two backup quarantine/removal pairs, journal retirement, and
	// its quarantine/removal pair.
	if syncs != 14 {
		t.Fatalf("directory sync count = %d, want 14", syncs)
	}
}

func TestJournalRetirementNeverOverwritesDestinationCollision(t *testing.T) {
	for _, tc := range []struct {
		name, oldName, newName string
	}{
		{"isolation", txJournalName, txRecoveryName},
		{"retirement", txRecoveryName, txRetiredName},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"A": "old"})
			collided := false
			ops := txDefaultOps(func(oldName, newName string) error {
				if collided || oldName != tc.oldName || newName != tc.newName {
					return nil
				}
				collided = true
				return os.WriteFile(filepath.Join(dir, newName), []byte("foreign destination\n"), 0o600)
			})
			err := txCommit(dir, map[string][]byte{"A": []byte("new")}, ops)
			if err == nil || !collided {
				t.Fatalf("destination collision was not rejected: collided=%v err=%v", collided, err)
			}
			assertFile(t, filepath.Join(dir, tc.newName), "foreign destination\n")
			if _, statErr := os.Lstat(filepath.Join(dir, tc.oldName)); statErr != nil {
				t.Fatalf("source authority was lost after destination collision: %v", statErr)
			}
		})
	}
}

func TestJournalInstallationNeverOverwritesBoundaryCollision(t *testing.T) {
	for _, collideAt := range []int{1, 2} {
		t.Run(fmt.Sprintf("journal-install-%d", collideAt), func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"A": "old"})
			installs := 0
			var foreign = fmt.Sprintf("foreign journal %d\n", collideAt)
			ops := txDefaultOps(func(oldName, newName string) error {
				if !strings.HasPrefix(oldName, txStagePrefix) || newName != txJournalName {
					return nil
				}
				installs++
				if installs != collideAt {
					return nil
				}
				return os.WriteFile(filepath.Join(dir, txJournalName), []byte(foreign), 0o600)
			})
			err := txCommit(dir, map[string][]byte{"A": []byte("new")}, ops)
			if err == nil || installs != collideAt {
				t.Fatalf("journal destination collision was not rejected: installs=%d err=%v", installs, err)
			}
			assertFile(t, filepath.Join(dir, txJournalName), foreign)
			if collideAt == 1 {
				assertFile(t, filepath.Join(dir, "A"), "old")
			}
			if collideAt == 2 {
				entries, readErr := os.ReadDir(dir)
				if readErr != nil {
					t.Fatal(readErr)
				}
				foundPreparedAuthority := false
				for _, entry := range entries {
					if txValidQuarantineName(entry.Name(), txJournalUpdateQuarantinePrefix) {
						foundPreparedAuthority = true
					}
				}
				if !foundPreparedAuthority {
					t.Fatal("committed-journal collision lost the isolated prepared authority")
				}
			}
		})
	}
}

func TestArtifactMoveNeverOverwritesBoundaryCollision(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"A": "old"})
	var backup string
	ops := txDefaultOps(func(oldName, newName string) error {
		if oldName != "A" || !strings.HasPrefix(newName, txOldPrefix) {
			return nil
		}
		backup = newName
		return os.WriteFile(filepath.Join(dir, newName), []byte("foreign backup"), 0o600)
	})
	err := txCommit(dir, map[string][]byte{"A": []byte("new")}, ops)
	if err == nil || backup == "" {
		t.Fatalf("artifact move collision was not rejected: backup=%q err=%v", backup, err)
	}
	assertFile(t, filepath.Join(dir, "A"), "old")
	assertFile(t, filepath.Join(dir, backup), "foreign backup")
}

func TestQuarantinedJournalDeletionCannotDeletePublicReplacement(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"A": "old"})
	held := filepath.Join(dir, "held-quarantine")
	var public string
	ops := txDefaultOps(nil)
	ops.faultAfter = func(point string) error {
		if point != "journal-quarantine-delete" || public != "" {
			return nil
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if txValidQuarantineName(entry.Name(), txJournalQuarantinePrefix) {
				public = filepath.Join(dir, entry.Name())
				break
			}
		}
		if public == "" {
			return fmt.Errorf("quarantine not found")
		}
		if err := os.Rename(public, held); err != nil {
			return err
		}
		if err := os.Mkdir(public, 0o700); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(public, "object"), []byte("foreign replacement\n"), 0o600)
	}
	err := txCommit(dir, map[string][]byte{"A": []byte("new")}, ops)
	if err == nil || public == "" {
		t.Fatalf("public quarantine replacement was not exercised: %v", err)
	}
	assertFile(t, filepath.Join(public, "object"), "foreign replacement\n")
	if _, statErr := os.Lstat(filepath.Join(held, "object")); !os.IsNotExist(statErr) {
		t.Fatalf("held authority was not removed through its retained namespace: %v", statErr)
	}
}

func TestRecoveryResumesObjectQuarantineAfterCrash(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"A": "old"})
	injected := errors.New("power loss after object quarantine")
	crashed := false
	ops := txDefaultOps(nil)
	ops.faultAfter = func(point string) error {
		if !crashed && strings.HasPrefix(point, "object-quarantine-delete:") {
			crashed = true
			return injected
		}
		return nil
	}
	err := txCommit(dir, map[string][]byte{"A": []byte("new")}, ops)
	var crash *txCrash
	if !crashed || !errors.As(err, &crash) || !errors.Is(err, injected) {
		t.Fatalf("object quarantine crash was not retained: crashed=%v err=%v", crashed, err)
	}
	if err := Commit(dir, map[string][]byte{}); err != nil {
		t.Fatalf("resume object quarantine: %v", err)
	}
	assertFile(t, filepath.Join(dir, "A"), "new")
	assertNoTransactionFiles(t, dir)
}

func TestRecoveryRejectsJournalMutationAfterParseBeforeIsolation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T, path string, body []byte, before os.FileInfo)
	}{
		{
			name: "path replacement",
			mutate: func(t *testing.T, path string, body []byte, _ os.FileInfo) {
				if err := os.Rename(path, path+".original"); err != nil {
					t.Fatal(err)
				}
				writeBytes(t, path, body)
			},
		},
		{
			name: "content replacement",
			mutate: func(t *testing.T, path string, body []byte, _ os.FileInfo) {
				changed := append([]byte(nil), body...)
				changed[0] ^= 1
				writeBytes(t, path, changed)
			},
		},
		{
			name: "same-byte metadata ABA",
			mutate: func(t *testing.T, path string, body []byte, before os.FileInfo) {
				if txChangeID(before) == "" {
					t.Skip("platform does not expose a journal change identity")
				}
				writeBytes(t, path, body)
				if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"A": "old-a", "B": "old-b"})
			crashCommitAt(t, dir, "prepared-journal")
			journalPath := filepath.Join(dir, txJournalName)
			body, err := os.ReadFile(journalPath)
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.Lstat(journalPath)
			if err != nil {
				t.Fatal(err)
			}
			mutated := false
			ops := txDefaultOps(func(old, new string) error {
				if mutated || old != txJournalName || new != txRecoveryName {
					return nil
				}
				mutated = true
				tc.mutate(t, journalPath, body, before)
				return nil
			})
			err = txCommit(dir, map[string][]byte{}, ops)
			if err == nil || !strings.Contains(err.Error(), "isolate parsed transaction journal") || !strings.Contains(err.Error(), "changed") {
				t.Fatalf("post-parse journal mutation error = %v", err)
			}
			if !mutated {
				t.Fatal("post-parse journal mutation hook did not run")
			}
			assertFile(t, filepath.Join(dir, "A"), "old-a")
			assertFile(t, filepath.Join(dir, "B"), "old-b")
			if _, statErr := os.Lstat(journalPath); statErr != nil {
				t.Fatalf("foreign journal path was removed: %v", statErr)
			}
			if _, statErr := os.Lstat(filepath.Join(dir, txRecoveryName)); !os.IsNotExist(statErr) {
				t.Fatalf("failed isolation published recovery authority: %v", statErr)
			}
		})
	}
}

func TestRecoveryRejectsIsolatedJournalMutationBetweenDestructiveSteps(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T, path string, body []byte, before os.FileInfo)
	}{
		{
			name: "path replacement",
			mutate: func(t *testing.T, path string, body []byte, _ os.FileInfo) {
				if err := os.Rename(path, path+".original"); err != nil {
					t.Fatal(err)
				}
				writeBytes(t, path, body)
			},
		},
		{
			name: "content replacement",
			mutate: func(t *testing.T, path string, body []byte, _ os.FileInfo) {
				changed := append([]byte(nil), body...)
				changed[0] ^= 1
				writeBytes(t, path, changed)
			},
		},
		{
			name: "same-byte metadata ABA",
			mutate: func(t *testing.T, path string, body []byte, before os.FileInfo) {
				if txChangeID(before) == "" {
					t.Skip("platform does not expose a journal change identity")
				}
				writeBytes(t, path, body)
				if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"A": "old-a", "B": "old-b"})
			crashCommitAt(t, dir, "install:B")
			mutated := false
			ops := txDefaultOps(nil)
			ops.faultAfter = func(point string) error {
				if mutated || point != "rollback-restore:B" {
					return nil
				}
				mutated = true
				path := filepath.Join(dir, txRecoveryName)
				body, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				before, err := os.Lstat(path)
				if err != nil {
					return err
				}
				tc.mutate(t, path, body, before)
				return nil
			}
			err := txCommit(dir, map[string][]byte{}, ops)
			if err == nil || !strings.Contains(err.Error(), "transaction journal authority changed") {
				t.Fatalf("isolated journal mutation was accepted: %v", err)
			}
			if !mutated {
				t.Fatal("isolated journal mutation hook did not run")
			}
			assertFile(t, filepath.Join(dir, "A"), "new-a")
			assertFile(t, filepath.Join(dir, "B"), "old-b")
			if _, statErr := os.Lstat(filepath.Join(dir, txRecoveryName)); statErr != nil {
				t.Fatalf("foreign recovery authority was not preserved: %v", statErr)
			}
		})
	}
}

func TestCommitRejectsReservedTransactionNames(t *testing.T) {
	for _, name := range []string{txJournalName, txRecoveryName, txRetiredName, strings.ToUpper(txJournalName), txNewPrefix + "claim", txOldPrefix + "claim", txStagePrefix + "claim"} {
		t.Run(name, func(t *testing.T) {
			err := Commit(t.TempDir(), map[string][]byte{name: []byte("data")})
			if err == nil || !strings.Contains(err.Error(), "reserved") {
				t.Fatalf("reserved transaction name accepted: %v", err)
			}
		})
	}
}

func TestRecoveryFailsClosedOnUnsafeJournal(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, string, string)
	}{
		{"malformed", func(t *testing.T, dir, _ string) {
			writeBytes(t, filepath.Join(dir, txJournalName), []byte("{not-json\n"))
		}},
		{"truncated", func(t *testing.T, dir, _ string) {
			writeBytes(t, filepath.Join(dir, txJournalName), []byte("{\"version\":1"))
		}},
		{"unknown field", func(t *testing.T, dir, _ string) {
			writeBytes(t, filepath.Join(dir, txJournalName), []byte("{\"version\":1,\"phase\":\"prepared\",\"items\":[],\"checksum\":\"\",\"unknown\":true}\n"))
		}},
		{"noncanonical duplicate field", func(t *testing.T, dir, _ string) {
			writeBytes(t, filepath.Join(dir, txJournalName), []byte("{\"version\":1,\"version\":1}\n"))
		}},
		{"checksum corruption", func(t *testing.T, dir, _ string) {
			journal := txJournal{Version: txVersion, Phase: txPrepared, Items: []txItem{{
				Target: "A", Temp: txNewPrefix + "x", Backup: txOldPrefix + "x", HadOld: true,
				OldHash: txHash([]byte("old-a")), OldIdentity: "unix:1:1", NewHash: txHash([]byte("new-a")), NewIdentity: "unix:2:2",
			}}}
			body, err := txEncode(journal)
			if err != nil {
				t.Fatal(err)
			}
			checksumStart := bytes.Index(body, []byte(`"checksum":"sha256:`))
			if checksumStart < 0 {
				t.Fatal("encoded journal lacks checksum")
			}
			index := checksumStart + len(`"checksum":"sha256:`)
			if body[index] == '0' {
				body[index] = '1'
			} else {
				body[index] = '0'
			}
			writeBytes(t, filepath.Join(dir, txJournalName), body)
		}},
		{"symlink journal", func(t *testing.T, dir, outside string) {
			if err := os.Symlink(outside, filepath.Join(dir, txJournalName)); err != nil {
				t.Fatal(err)
			}
		}},
		{"nonregular journal", func(t *testing.T, dir, _ string) {
			if err := os.Mkdir(filepath.Join(dir, txJournalName), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"traversal target", func(t *testing.T, dir, _ string) {
			journal := txJournal{Version: txVersion, Phase: txPrepared, Items: []txItem{{
				Target: "../outside", Temp: txNewPrefix + "x", Backup: txOldPrefix + "x", NewHash: txHash([]byte("new")),
			}}}
			body, err := txEncode(journal)
			if err != nil {
				t.Fatal(err)
			}
			writeBytes(t, filepath.Join(dir, txJournalName), body)
		}},
		{"symlink orphan temp", func(t *testing.T, dir, outside string) {
			if err := os.Symlink(outside, filepath.Join(dir, txNewPrefix+"orphan")); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			outside := filepath.Join(t.TempDir(), "outside")
			writeBytes(t, outside, []byte("sentinel"))
			writeFiles(t, dir, map[string]string{"A": "old-a"})
			tc.setup(t, dir, outside)
			if err := Commit(dir, map[string][]byte{}); err == nil {
				t.Fatal("unsafe journal state was accepted")
			}
			assertFile(t, filepath.Join(dir, "A"), "old-a")
			assertFile(t, outside, "sentinel")
		})
	}
}

func TestJournalAliasDecodeDiagnosticIsDeterministic(t *testing.T) {
	journal := txJournal{Version: txVersion, Phase: txPrepared, Items: []txItem{
		{Target: "A", Temp: txNewPrefix + "one", Backup: txOldPrefix + "one", NewHash: txHash([]byte("a")), NewIdentity: "unix:1:1"},
		{Target: "B", Temp: txNewPrefix + "one", Backup: txOldPrefix + "one", NewHash: txHash([]byte("b")), NewIdentity: "unix:2:2"},
	}}
	body, err := txEncode(journal)
	if err != nil {
		t.Fatal(err)
	}
	const want = `journal temp ".machinery-artifact-new-one" aliases temp .machinery-artifact-new-one`
	for i := 0; i < 1_000; i++ {
		_, err := txDecode(body)
		if err == nil || err.Error() != want {
			t.Fatalf("decode %d diagnostic = %v, want %q", i, err, want)
		}
	}
}

func TestArtifactNameDiagnosticIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	root, syncFile, err := txOpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syncFile.Close(); _ = root.Close() })
	ops := txDefaultOps(nil)
	ops.root, ops.syncFile = root, syncFile
	files := map[string][]byte{
		"../bad":                 []byte("traversal"),
		txJournalName:            []byte("reserved"),
		"CON":                    []byte("nonportable"),
		"safe/also-traversal.md": []byte("traversal"),
	}
	const want = `unsafe artifact name "../bad"`
	for i := 0; i < 1_000; i++ {
		_, err := txValidateTargets(ops, files)
		if err == nil || err.Error() != want {
			t.Fatalf("validation %d diagnostic = %v, want %q", i, err, want)
		}
	}
}

func TestReservedEntryDiagnosticIsDeterministicWithMultipleBadEntries(t *testing.T) {
	dir := t.TempDir()
	// Create in reverse canonical order so this proof does not accidentally
	// inherit insertion order from a particular filesystem implementation.
	for _, name := range []string{
		".MACHINERY-ARTIFACT-z-invalid",
		".MACHINERY-ARTIFACT-a-invalid",
	} {
		writeBytes(t, filepath.Join(dir, name), []byte("reserved\n"))
	}
	root, syncFile, err := txOpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syncFile.Close(); _ = root.Close() })
	ops := txDefaultOps(nil)
	ops.root, ops.syncFile = root, syncFile
	const want = `reserved transaction path ".MACHINERY-ARTIFACT-a-invalid" uses noncanonical case`
	for i := 0; i < 1_000; i++ {
		_, err := txInspect(ops)
		if err == nil || err.Error() != want {
			t.Fatalf("inspection %d diagnostic = %v, want %q", i, err, want)
		}
	}
}

func TestRecoveryRejectsCorruptedTransactionBeforeMutation(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	writeBytes(t, outside, []byte("sentinel"))
	writeFiles(t, dir, map[string]string{"A": "old-a", "B": "old-b"})
	crashCommitAt(t, dir, "install:A")

	root, syncFile, err := txOpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	ops := txDefaultOps(nil)
	ops.root, ops.syncFile = root, syncFile
	inv, err := txInspect(ops)
	_ = syncFile.Close()
	_ = root.Close()
	if err != nil || len(inv.backups) == 0 {
		t.Fatalf("inspect interrupted transaction: %v, %#v", err, inv)
	}
	writeBytes(t, filepath.Join(dir, inv.backups[0]), []byte("corrupted"))
	before := directoryBytes(t, dir)
	if err := Commit(dir, map[string][]byte{}); err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("corrupted transaction accepted: %v", err)
	}
	after := directoryBytes(t, dir)
	wantAfter := make([]string, len(before))
	for i, entry := range before {
		wantAfter[i] = strings.Replace(entry, txJournalName+"=", txRecoveryName+"=", 1)
	}
	sort.Strings(wantAfter)
	if fmt.Sprint(wantAfter) != fmt.Sprint(after) {
		t.Fatalf("failed-closed recovery changed more than atomic journal isolation:\nbefore=%v\nafter=%v", before, after)
	}
	assertFile(t, outside, "sentinel")
}

func TestRecoveryCleansOrphanTempsButRejectsOrphanBackups(t *testing.T) {
	t.Run("temp", func(t *testing.T) {
		dir := t.TempDir()
		writeBytes(t, filepath.Join(dir, txNewPrefix+"orphan"), []byte("staged"))
		if err := Commit(dir, map[string][]byte{}); err != nil {
			t.Fatal(err)
		}
		assertNoTransactionFiles(t, dir)
	})
	t.Run("backup", func(t *testing.T) {
		dir := t.TempDir()
		backup := filepath.Join(dir, txOldPrefix+"orphan")
		writeBytes(t, backup, []byte("possibly-old"))
		if err := Commit(dir, map[string][]byte{}); err == nil {
			t.Fatal("orphan backup was discarded")
		}
		assertFile(t, backup, "possibly-old")
	})
}

func TestCommitConfinedWhenOutputDirectoryIsSwapped(t *testing.T) {
	parent := t.TempDir()
	output := filepath.Join(parent, "output")
	moved := filepath.Join(parent, "moved-output")
	if err := os.Mkdir(output, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, output, map[string]string{"A": "old-a", "B": "old-b"})

	ops := txDefaultOps(nil)
	swapped := false
	ops.faultAfter = func(point string) error {
		if point != "root-open" || swapped {
			return nil
		}
		swapped = true
		if err := os.Rename(output, moved); err != nil {
			return err
		}
		if err := os.Mkdir(output, 0o755); err != nil {
			return err
		}
		writeFiles(t, output, map[string]string{"A": "outside-a", "B": "outside-b"})
		return nil
	}
	if err := txCommit(output, map[string][]byte{"A": []byte("new-a"), "B": []byte("new-b")}, ops); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(moved, "A"), "new-a")
	assertFile(t, filepath.Join(moved, "B"), "new-b")
	assertFile(t, filepath.Join(output, "A"), "outside-a")
	assertFile(t, filepath.Join(output, "B"), "outside-b")
	assertNoTransactionFiles(t, moved)
	assertNoTransactionFiles(t, output)
}

func TestCommitConfinedWhenParentDirectoryIsSwapped(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "parent")
	output := filepath.Join(parent, "output")
	movedParent := filepath.Join(base, "moved-parent")
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, output, map[string]string{"A": "old-a"})

	ops := txDefaultOps(nil)
	ops.faultAfter = func(point string) error {
		if point != "root-open" {
			return nil
		}
		if err := os.Rename(parent, movedParent); err != nil {
			return err
		}
		if err := os.MkdirAll(output, 0o755); err != nil {
			return err
		}
		writeFiles(t, output, map[string]string{"A": "outside-a"})
		return nil
	}
	if err := txCommit(output, map[string][]byte{"A": []byte("new-a")}, ops); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(movedParent, "output", "A"), "new-a")
	assertFile(t, filepath.Join(output, "A"), "outside-a")
	assertNoTransactionFiles(t, filepath.Join(movedParent, "output"))
	assertNoTransactionFiles(t, output)
}

func TestRecoveryConfinedWhenOutputDirectoryIsSwapped(t *testing.T) {
	parent := t.TempDir()
	output := filepath.Join(parent, "output")
	moved := filepath.Join(parent, "moved-output")
	if err := os.Mkdir(output, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, output, map[string]string{"A": "old-a", "B": "old-b"})
	crashCommitAt(t, output, "install:A")

	ops := txDefaultOps(nil)
	ops.faultAfter = func(point string) error {
		if point != "root-open" {
			return nil
		}
		if err := os.Rename(output, moved); err != nil {
			return err
		}
		if err := os.Mkdir(output, 0o755); err != nil {
			return err
		}
		writeFiles(t, output, map[string]string{"A": "outside-a", "B": "outside-b"})
		return nil
	}
	if err := txCommit(output, map[string][]byte{}, ops); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(moved, "A"), "old-a")
	assertFile(t, filepath.Join(moved, "B"), "old-b")
	assertFile(t, filepath.Join(output, "A"), "outside-a")
	assertFile(t, filepath.Join(output, "B"), "outside-b")
	assertNoTransactionFiles(t, moved)
	assertNoTransactionFiles(t, output)
}

type testLocker struct {
	releaseErr error
}

func (l testLocker) Release() error { return l.releaseErr }

func TestCommitPropagatesLockReleaseFailure(t *testing.T) {
	releaseErr := errors.New("release failed")
	ops := txDefaultOps(nil)
	ops.acquire = func(string) (txLocker, error) { return testLocker{releaseErr: releaseErr}, nil }
	err := txCommit(t.TempDir(), map[string][]byte{}, ops)
	if !errors.Is(err, releaseErr) || !strings.Contains(err.Error(), "release artifact transaction lock") {
		t.Fatalf("lock release failure not propagated: %v", err)
	}
}

func TestCommitJoinsOperationAndLockReleaseFailures(t *testing.T) {
	dir := t.TempDir()
	writeBytes(t, filepath.Join(dir, txJournalName), []byte("malformed"))
	releaseErr := errors.New("release failed")
	ops := txDefaultOps(nil)
	ops.acquire = func(string) (txLocker, error) { return testLocker{releaseErr: releaseErr}, nil }
	err := txCommit(dir, map[string][]byte{}, ops)
	if !errors.Is(err, releaseErr) || !strings.Contains(err.Error(), "recover artifact transaction") {
		t.Fatalf("operation and release failures were not joined: %v", err)
	}
}

func crashCommitAt(t *testing.T, dir, point string) {
	t.Helper()
	ops := txDefaultOps(nil)
	ops.faultAfter = func(got string) error {
		if got == point {
			return errors.New("power loss")
		}
		return nil
	}
	if err := txCommit(dir, map[string][]byte{"A": []byte("new-a"), "B": []byte("new-b")}, ops); err == nil {
		t.Fatalf("fault point %s did not fire", point)
	}
}

func readJournalItem(t *testing.T, dir, target string) txItem {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(dir, txJournalName))
	if err != nil {
		t.Fatal(err)
	}
	journal, err := txDecode(body)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range journal.Items {
		if item.Target == target {
			return item
		}
	}
	t.Fatalf("journal item %q missing", target)
	return txItem{}
}

func replaceSameContent(t *testing.T, path string, body []byte) (os.FileInfo, error) {
	t.Helper()
	replacement := path + ".foreign-replacement"
	if err := os.WriteFile(replacement, body, 0o644); err != nil {
		return nil, err
	}
	info, err := os.Lstat(replacement)
	if err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		return nil, err
	}
	if err := os.Rename(replacement, path); err != nil {
		return nil, err
	}
	return info, nil
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		writeBytes(t, filepath.Join(dir, name), []byte(body))
	}
}

func writeBytes(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil || string(body) != want {
		t.Fatalf("%s = %q, %v; want %q", path, body, err, want)
	}
}

func assertNoTransactionFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(strings.ToLower(entry.Name()), txPrefix) {
			t.Fatalf("transaction residue remains: %s", entry.Name())
		}
	}
}

func recoveryJournalExists(t *testing.T, dir string) bool {
	t.Helper()
	found := 0
	for _, name := range []string{txJournalName, txRecoveryName, txRetiredName} {
		if _, err := os.Lstat(filepath.Join(dir, name)); err == nil {
			found++
		} else if !os.IsNotExist(err) {
			t.Fatalf("inspect recovery journal %s: %v", name, err)
		}
	}
	if found > 1 {
		t.Fatalf("multiple recovery journals remain")
	}
	return found == 1
}

func directoryBytes(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			result = append(result, entry.Name()+"=<dir>")
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			result = append(result, entry.Name()+"=<unreadable>")
			continue
		}
		result = append(result, entry.Name()+"="+string(body))
	}
	sort.Strings(result)
	return result
}
