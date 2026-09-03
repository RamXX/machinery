//go:build windows

package fsatomic

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestWindowsRenameOpenIsRootHandleRelative(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "parent"), 0o700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	want := errors.New("injected NtCreateFile stop")
	original := fsatomicNtCreateFile
	t.Cleanup(func() { fsatomicNtCreateFile = original })
	fsatomicNtCreateFile = func(_ *windows.Handle, _ uint32, oa *windows.OBJECT_ATTRIBUTES, _ *windows.IO_STATUS_BLOCK, _ *int64, _ uint32, _ uint32, _ uint32, _ uint32, _ uintptr, _ uint32) error {
		if oa == nil || oa.RootDirectory == 0 || oa.ObjectName == nil {
			t.Fatalf("NtCreateFile did not receive root-relative object attributes: %#v", oa)
		}
		if got := oa.ObjectName.String(); got != "source" {
			t.Fatalf("NtCreateFile source name = %q, want basename source", got)
		}
		return want
	}
	if err := RenameNoReplace(root, filepath.Join("parent", "source"), "destination"); !errors.Is(err, want) {
		t.Fatalf("RenameNoReplace error = %v, want injected error", err)
	}
}

func TestWindowsRenameDestinationIsRootHandleRelative(t *testing.T) {
	sourceDir := t.TempDir()
	destinationDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(sourceDir, "parent"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(destinationDir, "parent"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "parent", "source"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	sourceRoot, err := os.OpenRoot(sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	defer sourceRoot.Close()
	destinationRoot, err := os.OpenRoot(destinationDir)
	if err != nil {
		t.Fatal(err)
	}
	defer destinationRoot.Close()
	want := errors.New("injected NtSetInformationFile stop")
	original := fsatomicNtSetInformationFile
	t.Cleanup(func() { fsatomicNtSetInformationFile = original })
	fsatomicNtSetInformationFile = func(_ windows.Handle, _ *windows.IO_STATUS_BLOCK, buffer *byte, _ uint32, class uint32) error {
		if class != windows.FileRenameInformation {
			t.Fatalf("information class = %d, want FileRenameInformation", class)
		}
		info := (*fileRenameInformation)(unsafe.Pointer(buffer))
		if info.RootDirectory == 0 || info.ReplaceIfExists != 0 {
			t.Fatalf("destination rename authority = %#v", info)
		}
		nameLength := int(info.FileNameLength / 2)
		got := windows.UTF16ToString((*[windows.MAX_LONG_PATH]uint16)(unsafe.Pointer(&info.FileName[0]))[:nameLength:nameLength])
		if got != "destination" {
			t.Fatalf("NtSetInformationFile destination name = %q, want basename destination", got)
		}
		return want
	}
	if err := RenameNoReplaceBetween(sourceRoot, filepath.Join("parent", "source"), destinationRoot, filepath.Join("parent", "destination")); !errors.Is(err, want) {
		t.Fatalf("RenameNoReplaceBetween error = %v, want injected error", err)
	}
}

func TestWindowsPrivateDeletionUsesHandleBoundDisposition(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "quarantine"), 0o700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	private, err := root.OpenRoot("quarantine")
	if err != nil {
		t.Fatal(err)
	}
	defer private.Close()
	originalCreate := fsatomicNtCreateFile
	originalSet := fsatomicNtSetInformationFile
	t.Cleanup(func() {
		fsatomicNtCreateFile = originalCreate
		fsatomicNtSetInformationFile = originalSet
	})
	fsatomicNtCreateFile = func(handle *windows.Handle, access uint32, oa *windows.OBJECT_ATTRIBUTES, iosb *windows.IO_STATUS_BLOCK, allocation *int64, attributes uint32, sharing uint32, disposition uint32, options uint32, ea uintptr, eaLength uint32) error {
		if oa == nil || oa.RootDirectory == 0 || oa.ObjectName == nil || oa.ObjectName.String() != "quarantine" {
			t.Fatalf("private deletion object attributes = %#v", oa)
		}
		return originalCreate(handle, access, oa, iosb, allocation, attributes, sharing, disposition, options, ea, eaLength)
	}
	want := errors.New("injected disposition stop")
	fsatomicNtSetInformationFile = func(handle windows.Handle, _ *windows.IO_STATUS_BLOCK, _ *byte, _ uint32, class uint32) error {
		if handle == 0 || class != windows.FileDispositionInformationEx {
			t.Fatalf("private deletion handle/class = %v/%d", handle, class)
		}
		return want
	}
	if err := removePrivateDirectory(root, private, "quarantine"); !errors.Is(err, want) {
		t.Fatalf("removePrivateDirectory error = %v, want injected disposition error", err)
	}
	if _, err := root.Lstat("quarantine"); err != nil {
		t.Fatalf("injected deletion removed quarantine: %v", err)
	}
}

func TestWindowsPrivateDeletionSurvivesRootPathReplacement(t *testing.T) {
	parent := t.TempDir()
	rootPath := filepath.Join(parent, "root")
	keptPath := filepath.Join(parent, "kept")
	if err := os.MkdirAll(filepath.Join(rootPath, "quarantine"), 0o700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	private, err := root.OpenRoot("quarantine")
	if err != nil {
		t.Fatal(err)
	}
	defer private.Close()
	if err := os.Rename(rootPath, keptPath); err != nil {
		t.Skipf("filesystem does not permit held-root rename: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(rootPath, "quarantine"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootPath, "quarantine", "replacement"), []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := removePrivateDirectory(root, private, "quarantine"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(keptPath, "quarantine")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("held-root quarantine remains: %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(rootPath, "quarantine", "replacement")); err != nil || string(body) != "replacement" {
		t.Fatalf("replacement-root quarantine = %q, %v", body, err)
	}
}

func TestWindowsRenameSurvivesRootPathReplacement(t *testing.T) {
	parent := t.TempDir()
	rootPath := filepath.Join(parent, "root")
	keptPath := filepath.Join(parent, "kept")
	if err := os.Mkdir(rootPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootPath, "source"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := os.Rename(rootPath, keptPath); err != nil {
		t.Skipf("filesystem does not permit held-root rename: %v", err)
	}
	if err := os.Mkdir(rootPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootPath, "source"), []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RenameNoReplace(root, "source", "destination"); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(filepath.Join(keptPath, "destination")); err != nil || string(body) != "owned" {
		t.Fatalf("held-root destination = %q, %v", body, err)
	}
	if body, err := os.ReadFile(filepath.Join(rootPath, "source")); err != nil || string(body) != "replacement" {
		t.Fatalf("replacement-root source = %q, %v", body, err)
	}
}
