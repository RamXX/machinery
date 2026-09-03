//go:build windows

package fsatomic

import (
	"errors"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type fileRenameInformation struct {
	ReplaceIfExists byte
	RootDirectory   windows.Handle
	FileNameLength  uint32
	FileName        [windows.MAX_LONG_PATH]uint16
}

var (
	fsatomicNtCreateFile         = windows.NtCreateFile
	fsatomicNtSetInformationFile = windows.NtSetInformationFile
)

func renameNoReplaceBase(oldRoot *os.Root, oldname string, newRoot *os.Root, newname string) error {
	oldDir, err := oldRoot.Open(".")
	if err != nil {
		return err
	}
	newDir, err := newRoot.Open(".")
	if err != nil {
		return errors.Join(err, oldDir.Close())
	}
	oldRootHandle := windows.Handle(oldDir.Fd())
	newRootHandle := windows.Handle(newDir.Fd())
	oldNT, err := windows.NewNTUnicodeString(strings.ReplaceAll(oldname, "/", `\`))
	if err != nil {
		return errors.Join(err, newDir.Close(), oldDir.Close())
	}
	oa := &windows.OBJECT_ATTRIBUTES{RootDirectory: oldRootHandle, ObjectName: oldNT}
	oa.Length = uint32(unsafe.Sizeof(*oa))
	var source windows.Handle
	var iosb windows.IO_STATUS_BLOCK
	var allocation int64
	err = fsatomicNtCreateFile(
		&source,
		windows.DELETE|windows.FILE_READ_ATTRIBUTES|windows.SYNCHRONIZE,
		oa,
		&iosb,
		&allocation,
		0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		windows.FILE_OPEN,
		windows.FILE_OPEN_REPARSE_POINT,
		0,
		0,
	)
	if err != nil {
		return errors.Join(err, newDir.Close(), oldDir.Close())
	}
	newUTF16, err := windows.UTF16FromString(strings.ReplaceAll(newname, "/", `\`))
	if err != nil {
		return errors.Join(err, windows.CloseHandle(source), newDir.Close(), oldDir.Close())
	}
	if len(newUTF16) > len(fileRenameInformation{}.FileName) {
		return errors.Join(windows.ERROR_FILENAME_EXCED_RANGE, windows.CloseHandle(source), newDir.Close(), oldDir.Close())
	}
	nameBytes := (len(newUTF16) - 1) * 2
	info := &fileRenameInformation{}
	info.ReplaceIfExists = 0
	info.RootDirectory = newRootHandle
	info.FileNameLength = uint32(nameBytes)
	copy(info.FileName[:], newUTF16)
	err = fsatomicNtSetInformationFile(source, &iosb, (*byte)(unsafe.Pointer(info)), uint32(unsafe.Sizeof(*info)), windows.FileRenameInformation)
	return errors.Join(err, windows.CloseHandle(source), newDir.Close(), oldDir.Close())
}

func removePrivateDirectory(root *os.Root, private *os.Root, name string) error {
	expected, err := private.Lstat(".")
	if err != nil {
		return err
	}
	rootDir, err := root.Open(".")
	if err != nil {
		return err
	}
	nameNT, err := windows.NewNTUnicodeString(strings.ReplaceAll(name, "/", `\`))
	if err != nil {
		return errors.Join(err, rootDir.Close())
	}
	oa := &windows.OBJECT_ATTRIBUTES{RootDirectory: windows.Handle(rootDir.Fd()), ObjectName: nameNT}
	oa.Length = uint32(unsafe.Sizeof(*oa))
	var handle windows.Handle
	var iosb windows.IO_STATUS_BLOCK
	var allocation int64
	err = fsatomicNtCreateFile(
		&handle,
		windows.DELETE|windows.FILE_READ_ATTRIBUTES|windows.SYNCHRONIZE,
		oa,
		&iosb,
		&allocation,
		0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		windows.FILE_OPEN,
		windows.FILE_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT,
		0,
		0,
	)
	if err != nil {
		return errors.Join(err, rootDir.Close())
	}
	dir := os.NewFile(uintptr(handle), name)
	opened, statErr := dir.Stat()
	if statErr != nil || !os.SameFile(expected, opened) {
		return errors.Join(statErr, errors.New("atomic quarantine directory changed before handle-bound deletion; preserving replacement"), dir.Close(), rootDir.Close())
	}
	type dispositionInformationEx struct{ Flags uint32 }
	disposition := dispositionInformationEx{Flags: windows.FILE_DISPOSITION_DELETE | windows.FILE_DISPOSITION_POSIX_SEMANTICS | windows.FILE_DISPOSITION_IGNORE_READONLY_ATTRIBUTE}
	dispositionErr := fsatomicNtSetInformationFile(
		windows.Handle(dir.Fd()),
		&iosb,
		(*byte)(unsafe.Pointer(&disposition)),
		uint32(unsafe.Sizeof(disposition)),
		windows.FileDispositionInformationEx,
	)
	return errors.Join(dispositionErr, dir.Close(), rootDir.Close())
}
