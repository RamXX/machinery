//go:build darwin || linux

package install

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// Query the open descriptor, not its pathname or the file permission bits.
func bootstrapReceiptWritable(file *os.File) (bool, error) {
	if file == nil {
		return false, os.ErrInvalid
	}
	conn, err := file.SyscallConn()
	if err != nil {
		return false, err
	}
	var flags int
	var queryErr error
	controlErr := conn.Control(func(fd uintptr) { flags, queryErr = unix.FcntlInt(fd, unix.F_GETFL, 0) })
	if err := errors.Join(controlErr, queryErr); err != nil {
		return false, err
	}
	return flags&unix.O_ACCMODE == unix.O_WRONLY || flags&unix.O_ACCMODE == unix.O_RDWR, nil
}

func TestBootstrapReceiptWriterDescriptors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipt-same-path")
	want := []byte("unchanged receipt bytes\n")
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	var writers []*os.File
	for _, tc := range []struct {
		name     string
		mode     int
		writable bool
	}{
		{"readonly", os.O_RDONLY, false}, {"writeonly", os.O_WRONLY, true},
		{"readwrite", os.O_RDWR, true}, {"distinct_writeonly", os.O_WRONLY, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := os.OpenFile(path, tc.mode, 0)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = file.Close() })
			if _, err := file.Seek(3, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			got, err := bootstrapReceiptWritable(file)
			if err != nil || got != tc.writable {
				t.Fatalf("writable=%v err=%v", got, err)
			}
			if offset, err := file.Seek(0, io.SeekCurrent); err != nil || offset != 3 {
				t.Fatalf("query changed offset: %d %v", offset, err)
			}
			info, err := file.Stat()
			if err != nil || info.Mode() != before.Mode() || !os.SameFile(before, info) {
				t.Fatalf("query changed identity/mode: %v", err)
			}
			if raw, err := os.ReadFile(path); err != nil || !bytes.Equal(raw, want) {
				t.Fatalf("query changed bytes: %q %v", raw, err)
			}
			if tc.writable {
				if n, err := file.Write(nil); err != nil || n != 0 {
					t.Fatalf("writer unusable: %d %v", n, err)
				}
				writers = append(writers, file)
			} else {
				var b [1]byte
				if n, err := file.Read(b[:]); err != nil || n != 1 || b[0] != want[3] {
					t.Fatalf("reader unusable: %d %v", n, err)
				}
			}
			if err := closeInstallFile(file); err != nil {
				t.Fatalf("normal close failed: %v", err)
			}
			conn, err := file.SyscallConn()
			if err != nil {
				t.Fatal(err)
			}
			actualControlErr := conn.Control(func(uintptr) {})
			if actualControlErr == nil {
				t.Fatal("closed handle Control unexpectedly succeeded")
			}
			if writable, err := bootstrapReceiptWritable(file); writable || err == nil || !errors.Is(err, actualControlErr) {
				t.Fatalf("closed handle query = %v, %v; want actual Control error %v", writable, err, actualControlErr)
			}
		})
	}
	if len(writers) != 3 || writers[0] == writers[1] || writers[0] == writers[2] {
		t.Fatal("distinct writable handles were not all observed")
	}
}

func TestBootstrapReceiptWriterInvalidHandle(t *testing.T) {
	if writable, err := bootstrapReceiptWritable(nil); writable || !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("invalid handle = %v, %v", writable, err)
	}
}

func TestBootstrapReceiptWriterSyscallError(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "receipt-invalid-descriptor-")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := file.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	var closeErr error
	if err := conn.Control(func(fd uintptr) { closeErr = unix.Close(int(fd)) }); err != nil || closeErr != nil {
		t.Fatalf("invalidate owned descriptor: %v %v", err, closeErr)
	}
	// No intervening descriptor allocation: query the actual invalid fd, then
	// retire the Go handle immediately so it cannot close a reused descriptor.
	writable, queryErr := bootstrapReceiptWritable(file)
	retireErr := file.Close()
	if writable || !errors.Is(queryErr, unix.EBADF) || !errors.Is(retireErr, unix.EBADF) {
		t.Fatalf("syscall failure not retained: writable=%v query=%v close=%v", writable, queryErr, retireErr)
	}
}
