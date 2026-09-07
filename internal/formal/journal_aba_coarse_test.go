package formal

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// requireFormalJournalMutationSentinel skips the test when the platform
// provides no kernel mutation-event channel for an already-open journal file.
func requireFormalJournalMutationSentinel(t *testing.T) {
	t.Helper()
	probe, err := os.CreateTemp("", "formal-journal-sentinel-probe")
	if err != nil {
		t.Fatal(err)
	}
	name := probe.Name()
	sentinel, sentinelErr := newFormalJournalMutationSentinel(probe)
	closeErr := probe.Close()
	removeErr := os.Remove(name)
	if sentinelErr != nil || sentinel == nil {
		t.Skip("platform lacks a kernel journal mutation-event witness channel")
	}
	if err := errors.Join(closeErr, removeErr, sentinel.Close()); err != nil {
		t.Fatal(err)
	}
}

// [tdd-red] MAC-3hzt: the retained-handle journal content ABA rejection must
// not depend on timestamp tick luck. This reproduces the Linux coarse-clock
// blindness deterministically on any host: the witness clock is coarsened by
// a full hour bucket (seed and recovery included, so recorded and live
// witnesses stay comparable), the same-bytes truncate/rewrite mutation
// through the retired journal name compares timestamp-equal on every
// platform, and only a granularity-independent signal (kernel mutation
// events) can reject it.
func TestFormalRecoveryRejectsJournalContentABAThroughRetainedHandleUnderCoarseClock(t *testing.T) {
	requireFormalJournalMutationSentinel(t)
	dir := t.TempDir()
	originalCoarsener := formalWitnessTimeCoarsener
	t.Cleanup(func() { formalWitnessTimeCoarsener = originalCoarsener })
	formalWitnessTimeCoarsener = anchoredWitnessCoarsener(time.Hour)
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
	injected := errors.New("injected coarse-clock recovery journal content ABA")
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
		t.Fatalf("coarse-clock recovery journal content ABA was not rejected: %v", err)
	}
	live, readErr := os.ReadFile(journalPath)
	if readErr != nil || !bytes.Equal(live, journalBody) {
		t.Fatalf("coarse-clock same-content journal evidence was not preserved: %q, %v", live, readErr)
	}
	if _, statErr := os.Lstat(filepath.Join(dir, "A.tla")); !os.IsNotExist(statErr) {
		t.Fatalf("coarse-clock recovery mutated target after journal content ABA: %v", statErr)
	}
	assertFormalFile(t, filepath.Join(dir, formalScratchName("backup", "A.tla")), "old-a")
}

// [tdd-red] MAC-3hzt control: under the same coarsened clock, a benign
// recovery with no journal tampering must still succeed, including the
// protocol's own journal isolation rename, quarantine, and unlink through the
// retained authority. This proves the mutation-event channel stays silent
// for the recovery's own reads and protocol moves (atime, rename, and unlink
// noise excluded) and that coarse-clock rejection is not a false positive on
// quiet journals.
func TestFormalRecoveryToleratesBenignRetainedJournalUnderCoarseClock(t *testing.T) {
	requireFormalJournalMutationSentinel(t)
	dir := t.TempDir()
	originalCoarsener := formalWitnessTimeCoarsener
	t.Cleanup(func() { formalWitnessTimeCoarsener = originalCoarsener })
	formalWitnessTimeCoarsener = anchoredWitnessCoarsener(time.Hour)
	_, root, _ := seedPartialFormalWitness(t, dir, "old")
	defer root.Close() //nolint:errcheck // test cleanup
	if err := recoverFormalTransaction(root, renameFormalRoot); err != nil {
		t.Fatalf("benign coarse-clock recovery rejected: %v", err)
	}
	assertFormalFile(t, filepath.Join(dir, "A.tla"), "old-a")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "A.tla" {
		t.Fatalf("benign coarse-clock recovery left residue: %v, %v", entries, err)
	}
}
