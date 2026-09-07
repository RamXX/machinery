package runtimeclosure

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// anchoredJavaWitnessCoarsener returns a change-ID coarsener that collapses
// every ctime into quantum-sized buckets anchored at the first observed
// timestamp, simulating a kernel whose inode timestamps tick only once per
// quantum. Sub-quantum before/after differences compare equal by
// construction, on any host clock: determinism does not depend on where the
// real clock's tick boundaries fall.
func anchoredJavaWitnessCoarsener(quantum time.Duration) func(int64, int64) (int64, int64) {
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

func requireJavaMutationSentinel(t *testing.T, javaPath string) {
	t.Helper()
	probeFile, err := os.Open(javaPath)
	if err != nil {
		t.Fatal(err)
	}
	defer probeFile.Close() //nolint:errcheck // capability probe
	probe, err := newJavaFileMutationSentinel(probeFile)
	if err != nil || probe == nil {
		t.Skip("platform lacks a kernel mutation-event witness channel")
	}
	_ = probe.Close() //nolint:errcheck // capability probe
}

// [tdd-red] MAC-o82q: the same-inode launcher metadata ABA rejection must
// not depend on timestamp tick luck. This reproduces the Linux coarse-clock
// blindness deterministically on any host: the witness clock is coarsened by
// a full hour bucket, so the sub-millisecond truncate/rewrite/mtime-restore
// mutation inside the open seam compares timestamp-equal on every platform,
// and only a granularity-independent signal (kernel mutation events) can
// reject it.
func TestOpenJavaLauncherRejectsSameInodeMetadataABAUnderCoarseClock(t *testing.T) {
	dir := makeTestJavaRoot(t, []byte("same launcher"), []byte("modules"))
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	javaPath := filepath.Join(dir, "bin", "java")
	before, err := os.Lstat(javaPath)
	if err != nil {
		t.Fatal(err)
	}
	if javaFileChangeID(before) == "" {
		t.Skip("platform does not expose launcher change metadata")
	}
	requireJavaMutationSentinel(t, javaPath)
	originalCoarsener := javaWitnessTimeCoarsener
	t.Cleanup(func() { javaWitnessTimeCoarsener = originalCoarsener })
	javaWitnessTimeCoarsener = anchoredJavaWitnessCoarsener(time.Hour)
	file, _, _, err := openJavaLauncher(root, filepath.Join("bin", "java"), javaLauncherMaxBytes, func() error {
		launcher, err := os.OpenFile(javaPath, os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil {
			return err
		}
		_, writeErr := launcher.Write([]byte("same launcher"))
		return errors.Join(writeErr, launcher.Close(), os.Chtimes(javaPath, before.ModTime(), before.ModTime()))
	})
	if file != nil {
		_ = file.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "metadata") {
		t.Fatalf("coarse-clock same-inode launcher metadata ABA was accepted: %v", err)
	}
}

// [tdd-red] MAC-o82q control: under the same coarsened clock, a benign
// launcher open-and-read with no mutation must still succeed. This proves
// the mutation-event channel stays silent for the broker's own reads (atime
// noise excluded) and that coarse-clock rejection is not a false positive on
// an untouched launcher.
func TestOpenJavaLauncherToleratesBenignReadUnderCoarseClock(t *testing.T) {
	dir := makeTestJavaRoot(t, []byte("same launcher"), []byte("modules"))
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	javaPath := filepath.Join(dir, "bin", "java")
	requireJavaMutationSentinel(t, javaPath)
	originalCoarsener := javaWitnessTimeCoarsener
	t.Cleanup(func() { javaWitnessTimeCoarsener = originalCoarsener })
	javaWitnessTimeCoarsener = anchoredJavaWitnessCoarsener(time.Hour)
	file, info, body, err := openJavaLauncher(root, filepath.Join("bin", "java"), javaLauncherMaxBytes, nil)
	if err != nil {
		t.Fatalf("benign coarse-clock launcher read rejected: %v", err)
	}
	defer file.Close() //nolint:errcheck // test cleanup
	if info.Size() != int64(len("same launcher")) || string(body) != "same launcher" {
		t.Fatalf("benign coarse-clock launcher body = %q", body)
	}
}
