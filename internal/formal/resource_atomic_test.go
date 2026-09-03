package formal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/fsatomic"
)

func TestCopyFormalExactRejectsContinuousAppender(t *testing.T) {
	path := filepath.Join(t.TempDir(), "growing.tla")
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
	_, err = copyFormalExact(io.Discard, file, int64(len(initial)), "growing formal artifact")
	close(stop)
	<-done
	if err == nil || !strings.Contains(err.Error(), "grew while being read") {
		t.Fatalf("continuous append error = %v", err)
	}
}

func TestFormalDirectoryInventoryCeiling(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i <= formalDirEntryMax; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("entry-%06d", i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := readFormalDirectory(dir); err == nil || !strings.Contains(err.Error(), "entry limit") {
		t.Fatalf("inventory limit error = %v", err)
	}
}

func TestFormalDirectoryInventoryIsExactlyOrdered(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"z.tla", "a.cfg", "m.als"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := readFormalDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(entries))
	for i := range entries {
		got[i] = entries[i].Name()
	}
	if strings.Join(got, ",") != "a.cfg,m.als,z.tla" {
		t.Fatalf("ordered inventory = %v", got)
	}
}

func TestFormalDirectoryInventoryRejectsEntryABAWithSameMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "A.tla")
	if err := os.WriteFile(path, []byte("same"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	originalHook := formalAfterDirectoryInventoryPass
	t.Cleanup(func() { formalAfterDirectoryInventoryPass = originalHook })
	held := filepath.Join(t.TempDir(), "held")
	formalAfterDirectoryInventoryPass = func(string) {
		formalAfterDirectoryInventoryPass = nil
		if err := os.Rename(path, held); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("same"), before.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := readFormalDirectory(dir); err == nil || !strings.Contains(err.Error(), "changed between inventory passes") {
		t.Fatalf("same-metadata entry ABA error = %v", err)
	}
}

func TestFormalDirectoryInventoryRejectsSameDirectoryABA(t *testing.T) {
	dir := t.TempDir()
	before, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
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
		t.Fatalf("same-directory ABA error = %v", err)
	}
}

func TestFormalExactReaderRejectsOversizedSparseFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized.tla")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(formalArtifactMaxBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := readFormalFileExact(path, "oversized formal input"); err == nil || !strings.Contains(err.Error(), "no larger than") {
		t.Fatalf("oversized formal input error = %v", err)
	}
}

func TestFormalExactDeletionPreservesPostCheckReplacement(t *testing.T) {
	dir := t.TempDir()
	name := "A.tla"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	identity, exists, info, err := formalRegularSnapshot(root, name, name)
	if err != nil || !exists {
		t.Fatal(err)
	}
	witness, err := formalNativeWitness(root, name, name, info)
	if err != nil {
		t.Fatal(err)
	}
	if err := fsatomic.RenameNoReplace(root, name, "kept-original"); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile(name, []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = removeFormalSnapshotExact(root, name, identity, witness, info, name)
	if err == nil || !strings.Contains(err.Error(), "differs from its exact authority") {
		t.Fatalf("replacement deletion error = %v", err)
	}
	if body, err := root.ReadFile("kept-original"); err != nil || string(body) != "owned" {
		t.Fatalf("original = %q, %v", body, err)
	}
	entries, err := readFormalRootDirectory(root, dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), formalQuarantinePrefix) {
			continue
		}
		q, err := fsatomic.ResumeQuarantine(root, entry.Name(), name)
		if err != nil {
			t.Fatal(err)
		}
		defer q.Close()
		body, err := q.Root().ReadFile(q.Name())
		if err != nil || string(body) != "replacement" {
			t.Fatalf("quarantined replacement = %q, %v", body, err)
		}
		found = true
	}
	if !found {
		t.Fatal("replacement quarantine not preserved")
	}
}

func TestFormalParkingDestinationCollisionIsNeverClobbered(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "A.tla"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	backup := formalScratchName("backup", "A.tla")
	injected := false
	rename := func(root *os.Root, oldname, newname string) error {
		if !injected && oldname == "A.tla" && newname == backup {
			injected = true
			if err := root.WriteFile(backup, []byte("collision"), 0o600); err != nil {
				return err
			}
		}
		return renameFormalRoot(root, oldname, newname)
	}
	err := commitGeneratedArtifactsWithRename(dir, map[string]generatedArtifact{
		"A.tla": {body: []byte("new"), owner: "test"},
	}, rename)
	if err == nil {
		t.Fatal("destination collision was accepted")
	}
	if body, err := os.ReadFile(filepath.Join(dir, "A.tla")); err != nil || string(body) != "old" {
		t.Fatalf("live target = %q, %v", body, err)
	}
	if body, err := os.ReadFile(filepath.Join(dir, backup)); err != nil || string(body) != "collision" {
		t.Fatalf("collision = %q, %v", body, err)
	}
}

func TestFormalParkingPreservesPostCheckSourceReplacement(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "A.tla"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	backup := formalScratchName("backup", "A.tla")
	injected := false
	rename := func(root *os.Root, oldname, newname string) error {
		if !injected && oldname == "A.tla" && newname == backup {
			injected = true
			if err := root.Rename("A.tla", "kept-original"); err != nil {
				return err
			}
			if err := root.WriteFile("A.tla", []byte("replacement"), 0o600); err != nil {
				return err
			}
		}
		return renameFormalRoot(root, oldname, newname)
	}
	err := commitGeneratedArtifactsWithRename(dir, map[string]generatedArtifact{
		"A.tla": {body: []byte("new"), owner: "test"},
	}, rename)
	if err == nil {
		t.Fatal("post-check source replacement was accepted")
	}
	if body, err := os.ReadFile(filepath.Join(dir, "kept-original")); err != nil || string(body) != "old" {
		t.Fatalf("kept original = %q, %v", body, err)
	}
	if body, err := os.ReadFile(filepath.Join(dir, backup)); err != nil || string(body) != "replacement" {
		t.Fatalf("preserved source replacement = %q, %v", body, err)
	}
}

func TestFormalRecoveryFinishesUnjournaledStageQuarantine(t *testing.T) {
	dir := t.TempDir()
	stage := formalScratchName("stage", "A.tla")
	if err := os.WriteFile(filepath.Join(dir, stage), []byte("staged"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	q, err := fsatomic.Quarantine(root, stage, formalQuarantinePrefix)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
	if err := recoverFormalTransaction(root, renameFormalRoot); err != nil {
		t.Fatal(err)
	}
	entries, err := readFormalDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("formal quarantine recovery left residue: %v", entries)
	}
}
