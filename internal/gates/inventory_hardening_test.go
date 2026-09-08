package gates

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/dirscan"
)

func TestSortedGlobExtPropagatesDirectoryReadFailure(t *testing.T) {
	want := errors.New("injected inventory failure")
	prior := readGateDirectory
	t.Cleanup(func() { readGateDirectory = prior })
	readGateDirectory = func(string, int) ([]os.DirEntry, error) { return nil, want }

	if _, err := sortedGlobExt(t.TempDir(), ".md"); !errors.Is(err, want) {
		t.Fatalf("sortedGlobExt error = %v, want injected failure", err)
	}
}

func TestFinalHandoffReportsDirectoryReadFailure(t *testing.T) {
	design := t.TempDir()
	want := errors.New("injected inventory failure")
	prior := readGateDirectory
	t.Cleanup(func() { readGateDirectory = prior })
	readGateDirectory = func(string, int) ([]os.DirEntry, error) { return nil, want }

	g := CheckFinalHandoff(design)
	if !containsInventoryFinding(g.Errs, "injected inventory failure") {
		t.Fatalf("CheckFinalHandoff errors = %v, want inventory failure", g.Errs)
	}
}

func TestReadRootDirectoryRejectsCreateDeleteABA(t *testing.T) {
	testReadRootDirectoryABA(t, func(t *testing.T, dir string) {
		transient := filepath.Join(dir, "transient")
		if err := os.WriteFile(transient, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(transient); err != nil {
			t.Fatal(err)
		}
	})
}

func TestReadRootDirectoryRejectsRenameABA(t *testing.T) {
	testReadRootDirectoryABA(t, func(t *testing.T, dir string) {
		stable := filepath.Join(dir, "stable")
		transient := filepath.Join(dir, "transient")
		if err := os.Rename(stable, transient); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(transient, stable); err != nil {
			t.Fatal(err)
		}
	})
}

func testReadRootDirectoryABA(t *testing.T, mutate func(*testing.T, string)) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stable"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	initial, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	prior := rootInventoryAfterFirst
	t.Cleanup(func() { rootInventoryAfterFirst = prior })
	rootInventoryAfterFirst = func(rel string) {
		if rel != "." {
			return
		}
		rootInventoryAfterFirst = func(string) {}
		mutate(t, dir)
		if err := os.Chtimes(dir, initial.ModTime(), initial.ModTime()); err != nil {
			t.Fatal(err)
		}
	}

	channel, err := newInventoryMutationChannel()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = channel.Close() })

	if _, _, err := readRootDirectory(root, ".", 10, channel); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("readRootDirectory accepted same-directory ABA: %v", err)
	}
}

// TestReadRootDirectoryRejectsABAWithABlindChangeStamp is the configuration-
// independent form of the two tests above. It pins the native change stamp to
// a constant, which is what a Linux kernel without multigrain timestamps
// effectively does for any mutation that completes inside one coarse-clock
// tick, and requires the enumeration to be rejected anyway. It reproduces the
// blindness on every host, including those whose real inode clock happens to
// be fine enough to hide it.
func TestReadRootDirectoryRejectsABAWithABlindChangeStamp(t *testing.T) {
	prior := inventoryChangeID
	t.Cleanup(func() { inventoryChangeID = prior })
	inventoryChangeID = func(*os.File, os.FileInfo) (string, error) { return "blind-coarse-stamp", nil }

	testReadRootDirectoryABA(t, func(t *testing.T, dir string) {
		transient := filepath.Join(dir, "transient")
		if err := os.WriteFile(transient, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(transient); err != nil {
			t.Fatal(err)
		}
	})
}

// TestReadRootDirectoryRefusesWithoutMutationWitness pins the fail-closed
// direction: when the kernel mutation-event channel cannot be armed there is
// no granularity-independent witness left, and the enumeration is refused
// with a diagnostic rather than accepted.
func TestReadRootDirectoryRefusesWithoutMutationWitness(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	channel, err := newInventoryMutationChannel()
	if err != nil {
		t.Fatal(err)
	}
	if err := channel.Close(); err != nil {
		t.Fatal(err)
	}

	_, _, err = readRootDirectory(root, ".", 10, channel)
	if err == nil || !strings.Contains(err.Error(), "no reliable change witness") {
		t.Fatalf("readRootDirectory error = %v, want a refusal naming the missing witness", err)
	}
}

// TestDesignInventoryRefusesWithoutMutationChannel pins the same refusal at
// the traversal entry point the gates call.
func TestDesignInventoryRefusesWithoutMutationChannel(t *testing.T) {
	want := errors.New("no mutation-event channel on this host")
	prior := newInventoryMutationChannel
	t.Cleanup(func() { newInventoryMutationChannel = prior })
	newInventoryMutationChannel = func() (*dirscan.MutationChannel, error) { return nil, want }

	err := validateDesignInventory(t.TempDir())
	if !errors.Is(err, want) || !strings.Contains(err.Error(), "no reliable change witness") {
		t.Fatalf("validateDesignInventory error = %v, want a refusal naming the missing witness", err)
	}
}

func containsInventoryFinding(findings []string, needle string) bool {
	for _, finding := range findings {
		if strings.Contains(finding, needle) {
			return true
		}
	}
	return false
}
