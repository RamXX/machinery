//go:build !windows

package install

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const (
	extractUmaskHelperEnv  = "MACHINERY_TEST_EXTRACT_UMASK_HELPER"
	extractUmaskArchiveEnv = "MACHINERY_TEST_EXTRACT_UMASK_ARCHIVE"
	extractUmaskDestEnv    = "MACHINERY_TEST_EXTRACT_UMASK_DEST"
)

func TestCopyEntryFromRootReportsLexicalFirstInvalidEntry(t *testing.T) {
	for iteration := 0; iteration < 50; iteration++ {
		parent := t.TempDir()
		source := filepath.Join(parent, "source")
		if err := os.Mkdir(source, 0o755); err != nil {
			t.Fatal(err)
		}
		// Create in reverse lexical order so the assertion is independent of
		// filesystems that happen to enumerate by insertion order.
		if err := syscall.Mkfifo(filepath.Join(source, "z-invalid"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(filepath.Join(source, "a-invalid"), 0o600); err != nil {
			t.Fatal(err)
		}
		root, err := os.OpenRoot(parent)
		if err != nil {
			t.Fatal(err)
		}
		err = copyEntryFromRoot(root, "source", filepath.Join(t.TempDir(), "snapshot"))
		if closeErr := root.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
		if err == nil || !strings.Contains(err.Error(), "a-invalid") {
			t.Fatalf("iteration %d first failure = %v, want a-invalid", iteration, err)
		}
	}
}

func TestInstallRollbackRestoresModeAndMtimeUnderRestrictiveUmask(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	target := filepath.Join(t.TempDir(), "machinery-hook")
	write(t, target, "original")
	if err := os.Chmod(target, 0o751); err != nil {
		t.Fatal(err)
	}
	wantTime := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(target, wantTime, wantTime); err != nil {
		t.Fatal(err)
	}
	oldMask := syscall.Umask(0o077)
	defer syscall.Umask(oldMask)

	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := beginArtifactTransaction([]string{target})
	if err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(t.TempDir(), "changed")
	write(t, staged, "changed")
	if err := os.Chmod(staged, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(staged, time.Unix(1_800_000_000, 0), time.Unix(1_800_000_000, 0)); err != nil {
		t.Fatal(err)
	}
	if err := durableRemoveAll(target); err != nil {
		t.Fatal(err)
	}
	if err := copyEntryNoFollow(staged, target); err != nil {
		t.Fatal(err)
	}
	if err := tx.rollback(); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o751 {
		t.Fatalf("restored mode = %o, want 751", got)
	}
	if !info.ModTime().Equal(wantTime) {
		t.Fatalf("restored mtime = %s, want %s", info.ModTime(), wantTime)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "original" {
		t.Fatalf("restored content = %q, %v", got, err)
	}
}

func TestExtractTarGzIsIndependentOfProcessUmask(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "source.tar.gz")
	if err := os.WriteFile(archive, sourceTarball(t, "machinery"), 0o600); err != nil {
		t.Fatal(err)
	}
	var outputs []string
	for _, mask := range []string{"022", "077"} {
		destination := filepath.Join(t.TempDir(), "extracted")
		command := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestExtractTarGzUmaskHelper$")
		command.Env = append(os.Environ(),
			extractUmaskHelperEnv+"="+mask,
			extractUmaskArchiveEnv+"="+archive,
			extractUmaskDestEnv+"="+destination,
		)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("umask %s helper: %v\n%s", mask, err, output)
		}
		outputs = append(outputs, strings.TrimSpace(string(output)))
	}
	if outputs[0] != outputs[1] {
		t.Fatalf("extraction differs by umask:\n022 %s\n077 %s", outputs[0], outputs[1])
	}
}

func TestExtractTarGzUmaskHelper(t *testing.T) {
	mask := os.Getenv(extractUmaskHelperEnv)
	if mask == "" {
		return
	}
	parsed := 0
	for _, digit := range mask {
		parsed = parsed*8 + int(digit-'0')
	}
	syscall.Umask(parsed)
	destination := os.Getenv(extractUmaskDestEnv)
	if err := extractTarGz(os.Getenv(extractUmaskArchiveEnv), destination); err != nil {
		t.Fatal(err)
	}
	digest, err := artifactTreeDigest(destination)
	if err != nil {
		t.Fatal(err)
	}
	var modes bytes.Buffer
	if err := filepath.Walk(destination, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		_, _ = fmt.Fprintf(&modes, "%s:%o\n", filepath.ToSlash(strings.TrimPrefix(path, destination)), info.Mode().Perm())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s\n%s", digest, modes.String())
}
