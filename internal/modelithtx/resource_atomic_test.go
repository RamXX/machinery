package modelithtx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/fsatomic"
)

func writeResourceTestJournal(t *testing.T, root *os.Root, name, phase string) journal {
	t.Helper()
	record := journal{
		Version:          journalVersion,
		Phase:            phase,
		ExpectedDigest:   "sha256:" + strings.Repeat("1", 64),
		ExpectedIdentity: "sha256:" + strings.Repeat("2", 64),
		StagedDigest:     "sha256:" + strings.Repeat("3", 64),
	}
	body, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	if err := root.WriteFile(name, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return record
}

func TestCopyModelithFileExactRejectsContinuousAppender(t *testing.T) {
	path := filepath.Join(t.TempDir(), "growing")
	initial := strings.Repeat("x", 1<<20)
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	started := make(chan struct{})
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		appender, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			close(started)
			return
		}
		defer appender.Close()
		_, _ = appender.Write([]byte("y"))
		close(started)
		for {
			select {
			case <-stop:
				return
			default:
				_, _ = appender.Write([]byte("y"))
			}
		}
	}()
	<-started
	err = copyModelithFileExact(io.Discard, file, int64(len(initial)), "growing")
	close(stop)
	<-done
	if err == nil || !strings.Contains(err.Error(), "grew while being read") {
		t.Fatalf("continuous append error = %v", err)
	}
}

func TestModelithDirectoryInventoryCeiling(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i <= modelithDirEntryMax; i++ {
		name := filepath.Join(dir, fmt.Sprintf("entry-%06d", i))
		if err := os.WriteFile(name, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := readDir(root, "."); err == nil || !strings.Contains(err.Error(), "entry limit") {
		t.Fatalf("inventory limit error = %v", err)
	}
}

func TestModelithFingerprintRejectsOversizedSparseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "oversized")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(modelithFileSizeMax + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Fingerprint(dir); err == nil || !strings.Contains(err.Error(), "exceeding") {
		t.Fatalf("oversized Modelith input error = %v", err)
	}
}

func TestModelithExactDeletionPreservesPostCheckReplacement(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, retireName), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, retireName, "owned"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(repo)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	state, err := captureCorpusState(root, retireName)
	if err != nil {
		t.Fatal(err)
	}
	if err := fsatomic.RenameNoReplace(root, retireName, "kept-original"); err != nil {
		t.Fatal(err)
	}
	if err := root.Mkdir(retireName, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile(filepath.Join(retireName, "replacement"), []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = removeCorpusStateExact(root, retireName, state)
	if err == nil || !strings.Contains(err.Error(), "changed at deletion isolation") {
		t.Fatalf("replacement deletion error = %v", err)
	}
	if body, err := root.ReadFile(filepath.Join("kept-original", "owned")); err != nil || string(body) != "owned" {
		t.Fatalf("original = %q, %v", body, err)
	}
	entries, err := readDir(root, ".")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), modelithQuarantinePrefix(retireName)) {
			continue
		}
		q, err := fsatomic.ResumeQuarantine(root, entry.Name(), retireName)
		if err != nil {
			t.Fatal(err)
		}
		defer q.Close()
		body, err := q.Root().ReadFile(filepath.Join(q.Name(), "replacement"))
		if err != nil || string(body) != "replacement" {
			t.Fatalf("quarantined replacement = %q, %v", body, err)
		}
		found = true
	}
	if !found {
		t.Fatal("replacement quarantine not preserved")
	}
}

func TestRecoverResumesUnjournaledStageQuarantine(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, stageName), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, stageName, "scratch"), []byte("scratch"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(repo)
	if err != nil {
		t.Fatal(err)
	}
	q, err := fsatomic.Quarantine(root, stageName, modelithQuarantinePrefix(stageName))
	if err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(q.Close(), root.Close()); err != nil {
		t.Fatal(err)
	}
	if err := Recover(repo); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == stageName || strings.HasPrefix(entry.Name(), modelithQuarantinePrefix(stageName)) {
			t.Fatalf("recovery residue remains: %s", entry.Name())
		}
	}
}

func TestPublishParkingDestinationCollisionIsNeverClobbered(t *testing.T) {
	repo := newRepository(t, "old", "new")
	expected, err := Fingerprint(filepath.Join(repo, targetName))
	if err != nil {
		t.Fatal(err)
	}
	err = publish(repo, expected, hooks{afterLiveRevalidate: func() error {
		if err := os.Mkdir(filepath.Join(repo, backupName), 0o700); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(repo, backupName, "collision"), []byte("collision"), 0o600)
	}})
	if err == nil {
		t.Fatal("parking destination collision was accepted")
	}
	if body, err := os.ReadFile(filepath.Join(repo, backupName, "collision")); err != nil || string(body) != "collision" {
		t.Fatalf("collision = %q, %v", body, err)
	}
	if _, err := os.Lstat(filepath.Join(repo, targetName)); err != nil {
		t.Fatalf("live corpus was moved despite collision: %v", err)
	}
}

func TestPublishRetirementPreservesPostCheckReplacement(t *testing.T) {
	repo := newRepository(t, "old\n", "new\n")
	expected, err := Fingerprint(filepath.Join(repo, targetName))
	if err != nil {
		t.Fatal(err)
	}
	err = publish(repo, expected, hooks{beforeRetireMove: func() error {
		if err := os.Rename(filepath.Join(repo, retireName), filepath.Join(repo, "kept-retirement")); err != nil {
			return err
		}
		if err := os.Mkdir(filepath.Join(repo, retireName), 0o700); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(repo, retireName, "replacement"), []byte("replacement"), 0o600)
	}})
	if err == nil || !strings.Contains(err.Error(), "changed at deletion isolation") {
		t.Fatalf("post-check retirement replacement error = %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(repo, "kept-retirement", "domain.modelith.md")); err != nil || string(body) != "old\n" {
		t.Fatalf("kept retirement = %q, %v", body, err)
	}
	root, err := os.OpenRoot(repo)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	entries, err := readDir(root, ".")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), modelithQuarantinePrefix(retireName)) {
			continue
		}
		q, err := fsatomic.ResumeQuarantine(root, entry.Name(), retireName)
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := q.Root().ReadFile(filepath.Join(q.Name(), "replacement"))
		closeErr := q.Close()
		if readErr != nil || closeErr != nil || string(body) != "replacement" {
			t.Fatalf("quarantined post-check replacement = %q, %v, close=%v", body, readErr, closeErr)
		}
		found = true
	}
	if !found {
		t.Fatal("post-check replacement quarantine was not preserved")
	}
}

func TestRecoverJournalReplacementCrashStates(t *testing.T) {
	for _, test := range []struct {
		name      string
		authority bool
		next      bool
		wantPhase string
	}{
		{name: "before successor installation", next: true, wantPhase: journalRestoring},
		{name: "after successor installation", authority: true, wantPhase: journalRestoring},
		{name: "successor lost restores previous", wantPhase: journalPrepared},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			writeResourceTestJournal(t, root, journalPreviousName, journalPrepared)
			if test.authority {
				writeResourceTestJournal(t, root, journalAuthorityName, journalRestoring)
			}
			if test.next {
				writeResourceTestJournal(t, root, journalNextName, journalRestoring)
			}
			if err := recoverJournalReplacement(root); err != nil {
				t.Fatal(err)
			}
			state, exists, err := readRegularState(root, journalAuthorityName)
			if err != nil || !exists {
				t.Fatal(err)
			}
			record, err := decodeJournal(state.body)
			if err != nil || record.Phase != test.wantPhase {
				t.Fatalf("recovered phase = %q, %v; want %q", record.Phase, err, test.wantPhase)
			}
			for _, residue := range []string{journalPreviousName, journalNextName} {
				if _, err := root.Lstat(residue); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("phase-transition residue %s remains: %v", residue, err)
				}
			}
		})
	}
}

func TestJournalPhaseMovePreservesPostCheckReplacement(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	previousRecord := writeResourceTestJournal(t, root, journalAuthorityName, journalPrepared)
	authority, exists, err := readRegularState(root, journalAuthorityName)
	if err != nil || !exists {
		t.Fatal(err)
	}
	nextRecord := previousRecord
	nextRecord.Phase = journalRestoring
	originalHook := modelithBeforeJournalPhaseMove
	t.Cleanup(func() { modelithBeforeJournalPhaseMove = originalHook })
	modelithBeforeJournalPhaseMove = func() error {
		modelithBeforeJournalPhaseMove = nil
		if err := fsatomic.RenameNoReplace(root, journalAuthorityName, "kept-authority"); err != nil {
			return err
		}
		return root.WriteFile(journalAuthorityName, []byte("replacement"), 0o600)
	}
	if _, err := replaceJournalAuthority(root, authority, nextRecord); err == nil || !strings.Contains(err.Error(), "changed at phase-transition isolation") {
		t.Fatalf("post-check journal replacement error = %v", err)
	}
	kept, keptExists, err := readRegularState(root, "kept-authority")
	if err != nil || !keptExists || !sameRegularFileAfterRename(authority, kept) {
		t.Fatalf("original journal authority was not preserved: %v", err)
	}
	if body, err := root.ReadFile(journalPreviousName); err != nil || string(body) != "replacement" {
		t.Fatalf("post-check journal replacement = %q, %v", body, err)
	}
	if state, exists, err := readRegularState(root, journalNextName); err != nil || !exists {
		t.Fatalf("next journal was not preserved: %v", err)
	} else if record, err := decodeJournal(state.body); err != nil || record.Phase != journalRestoring {
		t.Fatalf("next journal phase = %q, %v", record.Phase, err)
	}
}
