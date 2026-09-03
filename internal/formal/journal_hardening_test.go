package formal

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormalJournalRecoversEveryPartialWitnessByte(t *testing.T) {
	for _, image := range []string{"old", "new"} {
		t.Run(image, func(t *testing.T) {
			probeDir := t.TempDir()
			_, probeRoot, probeRecord := seedPartialFormalWitness(t, probeDir, image)
			if err := probeRoot.Close(); err != nil {
				t.Fatal(err)
			}
			for cut := 1; cut < len(probeRecord); cut++ {
				t.Run(fmt.Sprintf("byte-%03d", cut), func(t *testing.T) {
					dir := t.TempDir()
					wantWitness, root, record := seedPartialFormalWitness(t, dir, image)
					defer root.Close() //nolint:errcheck // test cleanup
					appendRawFormalJournalBytes(t, dir, record[:cut])
					header, phase, err := readFormalJournal(root)
					if err != nil {
						t.Fatalf("read %s witness cut %d: %v", image, cut, err)
					}
					if phase != map[string]string{"old": "parking", "new": "installing"}[image] {
						t.Fatalf("phase = %q", phase)
					}
					gotWitness := header.Entries[0].OldWitness
					if image == "new" {
						gotWitness = header.Entries[0].NewWitness
					}
					if gotWitness != wantWitness {
						t.Fatalf("derived witness = %q, want %q", gotWitness, wantWitness)
					}
					if err := recoverFormalTransaction(root, renameFormalRoot); err != nil {
						t.Fatalf("recover %s witness cut %d: %v", image, cut, err)
					}
					body, err := os.ReadFile(filepath.Join(dir, "A.tla"))
					if err != nil || string(body) != "old-a" {
						t.Fatalf("recovered %s witness cut %d restored %q, %v", image, cut, body, err)
					}
					entries, err := os.ReadDir(dir)
					if err != nil || len(entries) != 1 || entries[0].Name() != "A.tla" {
						t.Fatalf("recovered %s witness cut %d residue: %v, %v", image, cut, entries, err)
					}
				})
			}
		})
	}
}

func TestFormalJournalPartialWitnessRejectsSameContentABA(t *testing.T) {
	dir := t.TempDir()
	_, root, record := seedPartialFormalWitness(t, dir, "old")
	defer root.Close() //nolint:errcheck // test cleanup
	backup := formalScratchName("backup", "A.tla")
	replacement := backup + ".replacement"
	if err := os.WriteFile(filepath.Join(dir, replacement), []byte("old-a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, replacement), filepath.Join(dir, backup)); err != nil {
		t.Fatal(err)
	}
	appendRawFormalJournalBytes(t, dir, record[:1])
	if _, _, err := readFormalJournal(root); err == nil || !strings.Contains(err.Error(), "durable native object") {
		t.Fatalf("same-content foreign backup satisfied partial witness: %v", err)
	}
}

func TestFormalJournalRecoversZeroByteWitnessCrash(t *testing.T) {
	for _, image := range []string{"old", "new"} {
		t.Run(image, func(t *testing.T) {
			dir := t.TempDir()
			_, root, _ := seedPartialFormalWitness(t, dir, image)
			defer root.Close() //nolint:errcheck // test cleanup
			if err := recoverFormalTransaction(root, renameFormalRoot); err != nil {
				t.Fatalf("recover zero-byte %s witness crash: %v", image, err)
			}
			body, err := os.ReadFile(filepath.Join(dir, "A.tla"))
			if err != nil || string(body) != "old-a" {
				t.Fatalf("zero-byte %s witness recovery restored %q, %v", image, body, err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 || entries[0].Name() != "A.tla" {
				t.Fatalf("zero-byte %s witness recovery residue: %v, %v", image, entries, err)
			}
		})
	}
}

func TestAppendFormalJournalRejectsPathABA(t *testing.T) {
	for _, point := range []string{"after-open", "after-sync"} {
		t.Run(point, func(t *testing.T) {
			dir := t.TempDir()
			root, before := createTestFormalJournal(t, dir)
			defer root.Close() //nolint:errcheck // test cleanup
			body, err := encodeFormalRecord(formalPhaseRecord{Phase: "parking"})
			if err != nil {
				t.Fatal(err)
			}
			ops := defaultFormalJournalOps()
			replace := func(root *os.Root, file *os.File) error {
				current, err := snapshotOpenedFormalJournal(file, false)
				if err != nil {
					return err
				}
				return replaceFormalJournalPath(root, point+"-parked", current.body)
			}
			if point == "after-open" {
				ops.afterOpen = replace
			} else {
				ops.afterSync = replace
			}
			err = appendFormalJournalRecordWithOps(root, body, ops)
			if err == nil || !strings.Contains(err.Error(), "changed") {
				t.Fatalf("journal %s ABA accepted: %v", point, err)
			}
			live, readErr := os.ReadFile(filepath.Join(dir, formalJournalName))
			if readErr != nil {
				t.Fatal(readErr)
			}
			wantLive := before
			if point == "after-sync" {
				wantLive = append(append([]byte(nil), before...), body...)
			}
			if !bytes.Equal(live, wantLive) {
				t.Fatalf("foreign same-content journal changed: got %q want %q", live, wantLive)
			}
		})
	}
}

func TestCreateFormalJournalFailureCleanupPreservesReplacement(t *testing.T) {
	for _, point := range []string{"write", "sync", "close"} {
		t.Run(point, func(t *testing.T) {
			dir := t.TempDir()
			root, err := os.OpenRoot(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close() //nolint:errcheck // test cleanup
			ops := defaultFormalJournalOps()
			injected := errors.New("injected " + point + " failure")
			replace := func() error {
				return replaceFormalJournalPath(root, point+"-created", []byte("foreign\n"))
			}
			switch point {
			case "write":
				ops.write = func(_ *os.File, _ []byte) (int, error) {
					return 0, errors.Join(replace(), injected)
				}
			case "sync":
				ops.sync = func(file *os.File) error {
					return errors.Join(file.Sync(), replace(), injected)
				}
			case "close":
				ops.close = func(file *os.File) error {
					return errors.Join(replace(), file.Close(), injected)
				}
			}
			err = createFormalJournalWithOps(root, []formalJournalEntry{formalRecoveryEntry("A.tla", true, "old", "new")}, ops)
			if !errors.Is(err, injected) || !strings.Contains(err.Error(), "preserving") {
				t.Fatalf("create failure did not join ownership-safe cleanup error: %v", err)
			}
			live, readErr := os.ReadFile(filepath.Join(dir, formalJournalName))
			if readErr != nil || string(live) != "foreign\n" {
				t.Fatalf("replacement journal was not preserved: %q, %v", live, readErr)
			}
		})
	}
}

func TestCreateFormalJournalFailureRemovesOwnedFile(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	ops := defaultFormalJournalOps()
	injected := errors.New("injected write failure")
	ops.write = func(_ *os.File, _ []byte) (int, error) { return 0, injected }
	err = createFormalJournalWithOps(root, []formalJournalEntry{formalRecoveryEntry("A.tla", true, "old", "new")}, ops)
	if !errors.Is(err, injected) {
		t.Fatalf("create failure missing cause: %v", err)
	}
	if _, err := root.Lstat(formalJournalName); !os.IsNotExist(err) {
		t.Fatalf("owned failed journal was not removed: %v", err)
	}
}

func TestFormalJournalReservesCompleteTransactionBeforeCreate(t *testing.T) {
	entries := make([]formalJournalEntry, 5000)
	for i := range entries {
		entries[i] = formalRecoveryEntry(fmt.Sprintf("A%06d.tla", i), true, "old", "new")
	}
	fits := func(count int) bool {
		header, err := encodeFormalRecord(formalJournalHeader{Version: 2, Phase: "prepared", Entries: entries[:count]})
		if err != nil {
			t.Fatal(err)
		}
		required, err := formalJournalRequiredSize(header, entries[:count])
		if err != nil {
			t.Fatal(err)
		}
		return required <= formalJournalMax
	}
	low, high := 1, len(entries)
	for low < high {
		mid := low + (high-low+1)/2
		if fits(mid) {
			low = mid
		} else {
			high = mid - 1
		}
	}
	if low == len(entries) || !fits(low) || fits(low+1) {
		t.Fatalf("failed to locate journal capacity boundary at %d", low)
	}

	oversizeDir := t.TempDir()
	oversizeRoot, err := os.OpenRoot(oversizeDir)
	if err != nil {
		t.Fatal(err)
	}
	err = createFormalJournal(oversizeRoot, entries[:low+1])
	if err == nil || !strings.Contains(err.Error(), "exceeding") {
		t.Fatalf("oversized complete transaction reservation was accepted: %v", err)
	}
	if _, statErr := oversizeRoot.Lstat(formalJournalName); !os.IsNotExist(statErr) {
		t.Fatalf("oversized reservation mutated journal path: %v", statErr)
	}
	if err := oversizeRoot.Close(); err != nil {
		t.Fatal(err)
	}

	fitDir := t.TempDir()
	fitRoot, err := os.OpenRoot(fitDir)
	if err != nil {
		t.Fatal(err)
	}
	defer fitRoot.Close() //nolint:errcheck // test cleanup
	if err := createFormalJournal(fitRoot, entries[:low]); err != nil {
		t.Fatalf("largest fitting complete transaction was rejected: %v", err)
	}
}

func TestAppendFormalJournalChecksLimitBeforeMutation(t *testing.T) {
	dir := t.TempDir()
	root, _ := createTestFormalJournal(t, dir)
	defer root.Close() //nolint:errcheck // test cleanup
	path := filepath.Join(dir, formalJournalName)
	body, err := encodeFormalRecord(formalPhaseRecord{Phase: "parking"})
	if err != nil {
		t.Fatal(err)
	}
	wantSize := int64(formalJournalMax - len(body) + 1)
	if err := os.Truncate(path, wantSize); err != nil {
		t.Fatal(err)
	}
	if err := appendFormalJournalRecord(root, body); err == nil || !strings.Contains(err.Error(), "would exceed") {
		t.Fatalf("over-limit append was accepted: %v", err)
	}
	info, err := os.Lstat(path)
	if err != nil || info.Size() != wantSize {
		t.Fatalf("rejected append mutated journal: size=%v err=%v", info, err)
	}
}

func TestFormalRecoveryRejectsJournalPathReplacementBeforeMutation(t *testing.T) {
	dir := t.TempDir()
	_, root, _ := seedPartialFormalWitness(t, dir, "old")
	defer root.Close() //nolint:errcheck // test cleanup
	journalBody, err := os.ReadFile(filepath.Join(dir, formalJournalName))
	if err != nil {
		t.Fatal(err)
	}
	injected := errors.New("injected recovery journal replacement")
	replaced := false
	rename := func(root *os.Root, old, new string) error {
		if !replaced && old == formalScratchName("backup", "A.tla") && new == "A.tla" {
			replaced = true
			return errors.Join(replaceNamedFormalJournalPath(root, formalRetiredJournalName, "retained-original-journal", journalBody), injected)
		}
		return root.Rename(old, new)
	}
	err = recoverFormalTransaction(root, rename)
	if !replaced || !errors.Is(err, injected) || !strings.Contains(err.Error(), "changed identity") {
		t.Fatalf("recovery journal replacement was not rejected through retained authority: %v", err)
	}
	live, readErr := os.ReadFile(filepath.Join(dir, formalRetiredJournalName))
	if readErr != nil || !bytes.Equal(live, journalBody) {
		t.Fatalf("replacement journal was not preserved: %q, %v", live, readErr)
	}
	retained, readErr := os.ReadFile(filepath.Join(dir, "retained-original-journal"))
	if readErr != nil || !bytes.Equal(retained, journalBody) {
		t.Fatalf("retained original journal authority was lost: %q, %v", retained, readErr)
	}
	if _, statErr := os.Lstat(filepath.Join(dir, "A.tla")); !os.IsNotExist(statErr) {
		t.Fatalf("recovery mutated target after journal replacement: %v", statErr)
	}
	assertFormalFile(t, filepath.Join(dir, formalScratchName("backup", "A.tla")), "old-a")
}

func TestFormalRecoveryIsolationRestoresReplacementOnBoundaryABA(t *testing.T) {
	dir := t.TempDir()
	_, root, _ := seedPartialFormalWitness(t, dir, "old")
	defer root.Close() //nolint:errcheck // test cleanup
	journalBody, err := os.ReadFile(filepath.Join(dir, formalJournalName))
	if err != nil {
		t.Fatal(err)
	}
	replaced := false
	rename := func(root *os.Root, old, new string) error {
		if !replaced && old == formalJournalName && new == formalRetiredJournalName {
			replaced = true
			if err := root.Rename(old, new); err != nil {
				return err
			}
			return replaceNamedFormalJournalPath(root, new, "retained-boundary-journal", journalBody)
		}
		return root.Rename(old, new)
	}
	err = recoverFormalTransaction(root, rename)
	if !replaced || err == nil || !strings.Contains(err.Error(), "isolated formal recovery journal") {
		t.Fatalf("recovery isolation boundary replacement was accepted: %v", err)
	}
	for _, name := range []string{formalJournalName, "retained-boundary-journal"} {
		body, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil || !bytes.Equal(body, journalBody) {
			t.Fatalf("journal %s was not preserved after isolation ABA: %q, %v", name, body, readErr)
		}
	}
	if _, statErr := os.Lstat(filepath.Join(dir, formalRetiredJournalName)); !os.IsNotExist(statErr) {
		t.Fatalf("mismatched tombstone was not restored to canonical path: %v", statErr)
	}
	if _, statErr := os.Lstat(filepath.Join(dir, "A.tla")); !os.IsNotExist(statErr) {
		t.Fatalf("isolation ABA mutated recovery target: %v", statErr)
	}
	assertFormalFile(t, filepath.Join(dir, formalScratchName("backup", "A.tla")), "old-a")
}

func TestFormalRecoveryRestartsAfterDurableIsolationRename(t *testing.T) {
	dir := t.TempDir()
	_, root, _ := seedPartialFormalWitness(t, dir, "old")
	defer root.Close() //nolint:errcheck // test cleanup
	injected := errors.New("crash after journal isolation rename")
	crashed := false
	rename := func(root *os.Root, old, new string) error {
		if !crashed && old == formalJournalName && new == formalRetiredJournalName {
			crashed = true
			return errors.Join(root.Rename(old, new), syncFormalDir(root), injected)
		}
		return root.Rename(old, new)
	}
	if err := recoverFormalTransaction(root, rename); !crashed || !errors.Is(err, injected) {
		t.Fatalf("durable isolation crash was not surfaced: %v", err)
	}
	if _, err := root.Lstat(formalRetiredJournalName); err != nil {
		t.Fatalf("durable recovery tombstone was not retained: %v", err)
	}
	if err := recoverFormalTransaction(root, renameFormalRoot); err != nil {
		t.Fatalf("restart from isolated recovery journal: %v", err)
	}
	assertFormalFile(t, filepath.Join(dir, "A.tla"), "old-a")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "A.tla" {
		t.Fatalf("restarted isolated recovery left residue: %v, %v", entries, err)
	}
}

func TestFormalRecoveryRejectsJournalContentABAThroughRetainedHandle(t *testing.T) {
	dir := t.TempDir()
	_, root, _ := seedPartialFormalWitness(t, dir, "old")
	defer root.Close() //nolint:errcheck // test cleanup
	journalPath := filepath.Join(dir, formalJournalName)
	journalBody, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	if formalJournalChangeID(before) == "" {
		t.Skip("platform does not expose journal change metadata")
	}
	injected := errors.New("injected recovery journal content ABA")
	mutated := false
	rename := func(root *os.Root, old, new string) error {
		if !mutated && old == formalScratchName("backup", "A.tla") && new == "A.tla" {
			mutated = true
			journalPath = filepath.Join(dir, formalRetiredJournalName)
			before, statErr := os.Lstat(journalPath)
			if statErr != nil {
				return statErr
			}
			file, openErr := root.OpenFile(formalRetiredJournalName, os.O_WRONLY|os.O_TRUNC, 0)
			if openErr != nil {
				return openErr
			}
			_, writeErr := file.Write(journalBody)
			closeErr := file.Close()
			timeErr := os.Chtimes(journalPath, before.ModTime(), before.ModTime())
			return errors.Join(writeErr, closeErr, timeErr, injected)
		}
		return root.Rename(old, new)
	}
	err = recoverFormalTransaction(root, rename)
	if !mutated || !errors.Is(err, injected) || !strings.Contains(err.Error(), "authority changed") {
		t.Fatalf("recovery journal same-content ABA was not rejected: %v", err)
	}
	live, readErr := os.ReadFile(journalPath)
	if readErr != nil || !bytes.Equal(live, journalBody) {
		t.Fatalf("same-content journal evidence was not preserved: %q, %v", live, readErr)
	}
	if _, statErr := os.Lstat(filepath.Join(dir, "A.tla")); !os.IsNotExist(statErr) {
		t.Fatalf("recovery mutated target after journal content ABA: %v", statErr)
	}
	assertFormalFile(t, filepath.Join(dir, formalScratchName("backup", "A.tla")), "old-a")
}

func seedPartialFormalWitness(t *testing.T, dir, image string) (string, *os.Root, []byte) {
	t.Helper()
	entry := formalRecoveryEntry("A.tla", true, "old-a", "new-a")
	writeFormalRecoveryFile(t, dir, entry.Target, "old-a")
	writeFormalRecoveryFile(t, dir, entry.Stage, "new-a")
	entries := []formalJournalEntry{entry}
	hydrateFormalRecoveryWitnesses(t, dir, entries)
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := createFormalJournal(root, entries); err != nil {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatal(err)
	}
	if err := appendFormalPhase(root, "parking"); err != nil {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatal(err)
	}
	if err := root.Rename(entry.Target, entry.Backup); err != nil {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatal(err)
	}
	if err := syncFormalDir(root); err != nil {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatal(err)
	}
	_, exists, oldInfo, err := formalRegularSnapshot(root, entry.Backup, "test parked image")
	if err != nil || !exists {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatalf("snapshot parked image: exists=%t err=%v", exists, err)
	}
	oldWitness, err := formalNativeWitness(root, entry.Backup, "test parked image", oldInfo)
	if err != nil {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatal(err)
	}
	if image == "old" {
		record, err := encodeFormalRecord(formalWitnessRecord{Target: entry.Target, Image: "old", Witness: oldWitness})
		if err != nil {
			root.Close() //nolint:errcheck // test failure cleanup
			t.Fatal(err)
		}
		return oldWitness, root, record
	}
	if err := appendFormalWitness(root, entry.Target, "old", oldWitness); err != nil {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatal(err)
	}
	if err := appendFormalPhase(root, "installing"); err != nil {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatal(err)
	}
	if err := root.Rename(entry.Stage, entry.Target); err != nil {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatal(err)
	}
	if err := syncFormalDir(root); err != nil {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatal(err)
	}
	_, exists, newInfo, err := formalRegularSnapshot(root, entry.Target, "test installed image")
	if err != nil || !exists {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatalf("snapshot installed image: exists=%t err=%v", exists, err)
	}
	newWitness, err := formalNativeWitness(root, entry.Target, "test installed image", newInfo)
	if err != nil {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatal(err)
	}
	record, err := encodeFormalRecord(formalWitnessRecord{Target: entry.Target, Image: "new", Witness: newWitness})
	if err != nil {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatal(err)
	}
	return newWitness, root, record
}

func appendRawFormalJournalBytes(t *testing.T, dir string, body []byte) {
	t.Helper()
	file, err := os.OpenFile(filepath.Join(dir, formalJournalName), os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.Write(body)
	if err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		t.Fatal(err)
	}
}

func createTestFormalJournal(t *testing.T, dir string) (*os.Root, []byte) {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := createFormalJournal(root, []formalJournalEntry{formalRecoveryEntry("A.tla", true, "old", "new")}); err != nil {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, formalJournalName))
	if err != nil {
		root.Close() //nolint:errcheck // test failure cleanup
		t.Fatal(err)
	}
	return root, body
}

func replaceFormalJournalPath(root *os.Root, parked string, replacement []byte) error {
	return replaceNamedFormalJournalPath(root, formalJournalName, parked, replacement)
}

func replaceNamedFormalJournalPath(root *os.Root, name, parked string, replacement []byte) error {
	if err := root.Rename(name, parked); err != nil {
		return err
	}
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	written, writeErr := io.Copy(file, bytes.NewReader(replacement))
	if writeErr == nil && written != int64(len(replacement)) {
		writeErr = io.ErrShortWrite
	}
	return errors.Join(writeErr, file.Sync(), file.Close(), syncFormalDir(root))
}

func assertFormalFile(t *testing.T, path, want string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil || string(body) != want {
		t.Fatalf("%s = %q, %v; want %q", path, body, err, want)
	}
}
