//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package designlock

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestAcquireRejectsFIFOAndSocket(t *testing.T) {
	t.Run("fifo", func(t *testing.T) {
		design := t.TempDir()
		if err := syscall.Mkfifo(filepath.Join(design, "input.fifo"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Acquire(design); err == nil || !strings.Contains(err.Error(), "special file") {
			t.Fatalf("Acquire error = %v", err)
		}
	})
	t.Run("socket", func(t *testing.T) {
		design, err := os.MkdirTemp("/tmp", "machinery-designlock-")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(design) })
		var listenConfig net.ListenConfig
		listener, err := listenConfig.Listen(context.Background(), "unix", filepath.Join(design, "input.sock"))
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		if _, err := Acquire(design); err == nil || !strings.Contains(err.Error(), "special file") {
			t.Fatalf("Acquire error = %v", err)
		}
	})
}

func TestMaterializeExternalTreeRejectsFIFOWithoutOpeningIt(t *testing.T) {
	design := t.TempDir()
	impl := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(impl, "blocked.fifo"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if _, err := lock.MaterializeExternalTree(impl); err == nil || !strings.Contains(err.Error(), "special file") {
		t.Fatalf("FIFO implementation input accepted: %v", err)
	}
}
