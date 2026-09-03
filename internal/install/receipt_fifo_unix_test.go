//go:build !windows

package install

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestLoadReceiptRejectsFIFOWithoutOpening(t *testing.T) {
	config := privateConfigDir(t)
	t.Setenv("MACHINERY_CONFIG_DIR", config)
	path := filepath.Join(config, "install.json")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, exists, err := loadReceipt(); !exists || err == nil || !strings.Contains(err.Error(), "private regular file") {
		t.Fatalf("FIFO receipt: exists=%v err=%v", exists, err)
	}
	if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("FIFO fixture changed: %v %v", info, err)
	}
}

func TestLoadReceiptRejectsNonPrivateLeafConfigDirectory(t *testing.T) {
	config := t.TempDir()
	if err := os.Chmod(config, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MACHINERY_CONFIG_DIR", config)
	if err := os.WriteFile(filepath.Join(config, "install.json"), []byte(`{"schema_version":2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadReceipt(); err == nil || !strings.Contains(err.Error(), "not confined") {
		t.Fatalf("non-private leaf config error = %v", err)
	}
}
