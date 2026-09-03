package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	machversion "github.com/RamXX/machinery/internal/version"
)

func TestValidateStructurizrCacheRejectsOversizedSparseReceipt(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".machinery-structurizr-receipt")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(structurizrReceiptMaxBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := validateStructurizrCache(root); err == nil || !strings.Contains(err.Error(), "byte limit") {
		t.Fatalf("Structurizr cache accepted oversized receipt: %v", err)
	}
}

func structurizrProvisionTestBase(t *testing.T) string {
	t.Helper()
	cacheRoot := t.TempDir()
	t.Setenv("HOME", cacheRoot)
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	t.Setenv("LOCALAPPDATA", cacheRoot)
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(cache, "machinery", "structurizr")
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatal(err)
	}
	return base
}

func TestProvisionStructurizrRecoversCrashStageBeforeCacheUse(t *testing.T) {
	base := structurizrProvisionTestBase(t)
	stage := filepath.Join(base, ".structurizr-stage-987654")
	if err := os.MkdirAll(filepath.Join(stage, "extracted", "lib"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "structurizr-cli.zip"), []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, machversion.StructurizrVersion)
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	_, firstErr := provisionStructurizr()
	if firstErr == nil || !strings.Contains(firstErr.Error(), "cache is invalid") {
		t.Fatalf("retry did not reach deterministic cache validation after recovery: %v", firstErr)
	}
	if _, err := os.Lstat(stage); !os.IsNotExist(err) {
		t.Fatalf("SIGKILL-equivalent Structurizr stage survived locked retry: %v", err)
	}
	_, secondErr := provisionStructurizr()
	if secondErr == nil || secondErr.Error() != firstErr.Error() {
		t.Fatalf("post-recovery retry diagnostic changed:\nfirst:  %v\nsecond: %v", firstErr, secondErr)
	}
}

func TestProvisionStructurizrFailsClosedOnUnsafeCrashStage(t *testing.T) {
	base := structurizrProvisionTestBase(t)
	unsafe := filepath.Join(base, ".structurizr-stage-123")
	if err := os.WriteFile(unsafe, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := provisionStructurizr(); err == nil || !strings.Contains(err.Error(), "private real directory") {
		t.Fatalf("unsafe Structurizr stage did not fail closed: %v", err)
	}
	if body, err := os.ReadFile(unsafe); err != nil || string(body) != "not a directory" {
		t.Fatalf("unsafe reserved residue was mutated: %q, %v", body, err)
	}
}

func TestProvisionStructurizrRetryConvergesAfterCompleteTargetRename(t *testing.T) {
	base := structurizrProvisionTestBase(t)
	target := filepath.Join(base, machversion.StructurizrVersion)
	if err := os.MkdirAll(filepath.Join(target, "lib"), 0o700); err != nil {
		t.Fatal(err)
	}
	launcher := structurizrLauncher(target)
	if err := os.WriteFile(launcher, []byte("complete launcher"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "lib", "structurizr.jar"), []byte("complete jar"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := fingerprintStructurizrTree(target)
	if err != nil {
		t.Fatal(err)
	}
	receipt := fmt.Sprintf("archive_sha256=%s\nclosure_sha256=%x\nversion=%s\n", machversion.StructurizrLinuxZipSHA256, digest, machversion.StructurizrVersion)
	if err := os.WriteFile(filepath.Join(target, ".machinery-structurizr-receipt"), []byte(receipt), 0o600); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(base, ".structurizr-stage-88")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "structurizr-cli.zip"), []byte("post-rename residue"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := provisionStructurizr()
	if err != nil || got != launcher {
		t.Fatalf("retry did not converge to the complete renamed target: path=%s err=%v", got, err)
	}
	if _, err := os.Lstat(stage); !os.IsNotExist(err) {
		t.Fatalf("retry retained post-rename Structurizr stage residue: %v", err)
	}
}
