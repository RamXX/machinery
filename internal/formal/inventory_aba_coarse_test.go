package formal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// anchoredWitnessCoarsener returns a witness-time coarsener that collapses
// every timestamp into quantum-sized buckets anchored at the first observed
// witness time, simulating a kernel whose inode timestamps tick only once per
// quantum. Sub-quantum before/after differences compare equal by
// construction, on any host clock: determinism does not depend on where the
// real clock's tick boundaries fall.
func anchoredWitnessCoarsener(quantum time.Duration) func(int64, int64) (int64, int64) {
	var anchorSec, anchorNsec int64
	anchored := false
	return func(sec, nsec int64) (int64, int64) {
		if !anchored {
			anchorSec, anchorNsec, anchored = sec, nsec, true
		}
		delta := (sec-anchorSec)*int64(time.Second) + (nsec - anchorNsec)
		if delta < 0 {
			delta = 0
		}
		return anchorSec, delta / int64(quantum)
	}
}

func requireFormalMutationSentinel(t *testing.T, dir string) {
	t.Helper()
	probe, err := newFormalDirectoryMutationSentinel(dir)
	if err != nil || probe == nil {
		t.Skip("platform lacks a kernel mutation-event witness channel")
	}
	_ = probe.Close() //nolint:errcheck // capability probe
}

// [tdd-red] MAC-o82q: the same-directory ABA rejection must not depend on
// timestamp tick luck. This reproduces the Linux coarse-clock blindness
// deterministically on any host: the witness clock is coarsened by a full
// hour bucket, so the sub-millisecond create/remove/restore mutation inside
// the inventory seam compares timestamp-equal on every platform, and only a
// granularity-independent signal (kernel mutation events) can reject it.
func TestFormalDirectoryInventoryRejectsSameDirectoryABAUnderCoarseClock(t *testing.T) {
	dir := t.TempDir()
	requireFormalMutationSentinel(t, dir)
	before, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	originalCoarsener := formalWitnessTimeCoarsener
	t.Cleanup(func() { formalWitnessTimeCoarsener = originalCoarsener })
	formalWitnessTimeCoarsener = anchoredWitnessCoarsener(time.Hour)
	originalHook := formalAfterDirectoryInventoryPass
	t.Cleanup(func() { formalAfterDirectoryInventoryPass = originalHook })
	formalAfterDirectoryInventoryPass = func(string) {
		formalAfterDirectoryInventoryPass = nil
		transient := filepath.Join(dir, "transient")
		if err := os.WriteFile(transient, []byte("transient"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(transient); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(dir, before.ModTime(), before.ModTime()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := readFormalDirectory(dir); err == nil || !strings.Contains(err.Error(), "changed between inventory passes") {
		t.Fatalf("coarse-clock same-directory ABA error = %v", err)
	}
}

// [tdd-red] MAC-o82q control: under the same coarsened clock, a benign
// double-pass enumeration with no mutation must still succeed. This proves
// the mutation-event channel stays silent for the inventory's own reads
// (atime noise excluded) and that coarse-clock rejection is not a false
// positive on quiet directories.
func TestFormalDirectoryInventoryToleratesBenignEnumerationUnderCoarseClock(t *testing.T) {
	dir := t.TempDir()
	requireFormalMutationSentinel(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "A.tla"), []byte("module A end"), 0o600); err != nil {
		t.Fatal(err)
	}
	originalCoarsener := formalWitnessTimeCoarsener
	t.Cleanup(func() { formalWitnessTimeCoarsener = originalCoarsener })
	formalWitnessTimeCoarsener = anchoredWitnessCoarsener(time.Hour)
	entries, err := readFormalDirectory(dir)
	if err != nil {
		t.Fatalf("benign coarse-clock enumeration rejected: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "A.tla" {
		t.Fatalf("benign coarse-clock inventory = %v", entries)
	}
}
