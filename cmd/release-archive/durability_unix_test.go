//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/fsatomic"
)

func TestArchivePublicationSyscallBoundaryCollisionPreservesOutput(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "release.tar.gz")
	if err := os.WriteFile(output, []byte("previous"), 0o600); err != nil {
		t.Fatal(err)
	}
	previous := publishArchiveNoReplace
	publishArchiveNoReplace = func(root *os.Root, oldName, newName string) error {
		file, err := root.OpenFile(newName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		if _, err := file.WriteString("boundary-collision"); err != nil {
			_ = file.Close()
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
		return fsatomic.RenameNoReplace(root, oldName, newName)
	}
	t.Cleanup(func() { publishArchiveNoReplace = previous })
	err := writeArchive(output, time.Unix(1_700_000_000, 0).UTC(), []entry{{name: "machinery/file", mode: 0o644, kind: entryRegular, data: []byte("new")}})
	if err == nil {
		t.Fatal("archive publication clobbered a syscall-boundary collision")
	}
	if body, err := os.ReadFile(output); err != nil || string(body) != "boundary-collision" {
		t.Fatalf("boundary output changed: %q, %v", body, err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	preservedPrior := false
	for _, item := range entries {
		if strings.HasPrefix(item.Name(), archivePublishDeletePrefix) {
			preservedPrior = true
		}
	}
	if !preservedPrior {
		t.Fatal("prior output authority was not preserved after collision")
	}
	stageName, err := archiveStageName(output)
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(filepath.Join(directory, stageName)); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("publication stage was not preserved for fail-closed recovery: %#v, %v", info, err)
	}
	if err := writeArchive(output, time.Unix(1_700_000_000, 0).UTC(), []entry{{name: "machinery/file", mode: 0o644, kind: entryRegular, data: []byte("retry")}}); err == nil || !strings.Contains(err.Error(), "ambiguous release archive publication") {
		t.Fatalf("retry did not fail closed on retained collision evidence: %v", err)
	}
	if body, err := os.ReadFile(output); err != nil || string(body) != "boundary-collision" {
		t.Fatalf("retry changed boundary output: %q, %v", body, err)
	}
}
