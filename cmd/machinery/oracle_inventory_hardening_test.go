package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/dirscan"
)

func TestReadOracleRootDirRejectsCreateDeleteABA(t *testing.T) {
	testReadOracleRootDirABA(t, func(t *testing.T, dir string) {
		transient := filepath.Join(dir, "transient")
		if err := os.WriteFile(transient, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(transient); err != nil {
			t.Fatal(err)
		}
	})
}

func TestReadOracleRootDirRejectsRenameABA(t *testing.T) {
	testReadOracleRootDirABA(t, func(t *testing.T, dir string) {
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

func testReadOracleRootDirABA(t *testing.T, mutate func(*testing.T, string)) {
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
	prior := oracleInventoryAfterFirst
	t.Cleanup(func() { oracleInventoryAfterFirst = prior })
	oracleInventoryAfterFirst = func(rel string) {
		if rel != "." {
			return
		}
		oracleInventoryAfterFirst = func(string) {}
		mutate(t, dir)
		if err := os.Chtimes(dir, initial.ModTime(), initial.ModTime()); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := readOracleRootDir(root, ".", 10); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("readOracleRootDir accepted same-directory ABA: %v", err)
	}
}

// TestReadOracleRootDirRejectsABAWithABlindChangeStamp is the configuration-
// independent form of the two tests above. It pins the native change stamp to
// a constant, which is what a Linux kernel without multigrain timestamps
// (mainline 6.13) effectively does for any mutation that completes inside one
// coarse-clock tick, and requires the enumeration to be rejected anyway.
func TestReadOracleRootDirRejectsABAWithABlindChangeStamp(t *testing.T) {
	prior := oracleChangeID
	t.Cleanup(func() { oracleChangeID = prior })
	oracleChangeID = func(*os.File, os.FileInfo) (string, error) { return "blind-coarse-stamp", nil }

	testReadOracleRootDirABA(t, func(t *testing.T, dir string) {
		transient := filepath.Join(dir, "transient")
		if err := os.WriteFile(transient, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(transient); err != nil {
			t.Fatal(err)
		}
	})
}

// TestReadOracleRootDirRefusesWithoutMutationWitness pins the fail-closed
// direction: without the kernel event channel there is no
// granularity-independent witness left, so the enumeration is refused with a
// diagnostic rather than accepted.
func TestReadOracleRootDirRefusesWithoutMutationWitness(t *testing.T) {
	want := errors.New("no mutation-event channel on this host")
	prior := newOracleMutationChannel
	t.Cleanup(func() { newOracleMutationChannel = prior })
	newOracleMutationChannel = func() (*dirscan.MutationChannel, error) { return nil, want }
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })

	_, err = readOracleRootDir(root, ".", 10)
	if !errors.Is(err, want) || !strings.Contains(err.Error(), "no reliable change witness") {
		t.Fatalf("readOracleRootDir error = %v, want a refusal naming the missing witness", err)
	}
}
