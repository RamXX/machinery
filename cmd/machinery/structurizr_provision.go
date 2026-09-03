package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/cachestage"
	"github.com/RamXX/machinery/internal/filelock"
	"github.com/RamXX/machinery/internal/safefile"
	machversion "github.com/RamXX/machinery/internal/version"
)

const structurizrArchiveMaxBytes = int64(200 << 20)
const structurizrReceiptMaxBytes int64 = 4 << 10

var structurizrHTTPDo = http.DefaultClient.Do

var closeStructurizrZip = func(reader *zip.ReadCloser) error { return reader.Close() }

func provisionStructurizr() (path string, retErr error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	base := filepath.Join(cache, "machinery", "structurizr")
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", err
	}
	lock, err := filelock.AcquireWait(base)
	if err != nil {
		return "", err
	}
	defer func() { retErr = errors.Join(retErr, lock.Release()) }()
	if err := cachestage.Recover(base, ".structurizr-stage-"); err != nil {
		return "", fmt.Errorf("recover interrupted Structurizr provision: %w", err)
	}
	target := filepath.Join(base, machversion.StructurizrVersion)
	if _, err := os.Lstat(target); err == nil {
		if err := validateStructurizrCache(target); err != nil {
			return "", fmt.Errorf("pinned Structurizr cache is invalid: %w; remove %s and retry", err, target)
		}
		return structurizrLauncher(target), nil
	}
	stage, err := os.MkdirTemp(base, ".structurizr-stage-")
	if err != nil {
		return "", err
	}
	defer func() {
		if err := cachestage.Recover(base, ".structurizr-stage-"); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("clean Structurizr provision stage: %w", err))
		}
	}()
	archive := filepath.Join(stage, "structurizr-cli.zip")
	url := "https://github.com/structurizr/cli/releases/download/v" + machversion.StructurizrVersion + "/structurizr-cli.zip"
	if err := downloadStructurizrArchive(url, archive); err != nil {
		return "", err
	}
	extracted := filepath.Join(stage, "extracted")
	if err := os.Mkdir(extracted, 0o700); err != nil {
		return "", err
	}
	if err := extractStructurizrZip(archive, extracted); err != nil {
		return "", err
	}
	launcher := structurizrLauncher(extracted)
	if info, err := os.Lstat(launcher); err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.Join(err, fmt.Errorf("pinned Structurizr archive lacks a regular launcher"))
	}
	digest, err := fingerprintStructurizrTree(extracted)
	if err != nil {
		return "", err
	}
	receipt := fmt.Sprintf("archive_sha256=%s\nclosure_sha256=%x\nversion=%s\n", machversion.StructurizrLinuxZipSHA256, digest, machversion.StructurizrVersion)
	if err := os.WriteFile(filepath.Join(extracted, ".machinery-structurizr-receipt"), []byte(receipt), 0o600); err != nil {
		return "", err
	}
	sourceRel, err := filepath.Rel(base, extracted)
	if err != nil {
		return "", err
	}
	targetRel, err := filepath.Rel(base, target)
	if err != nil {
		return "", err
	}
	if err := cachestage.PublishTree(base, sourceRel, targetRel); err != nil {
		return "", fmt.Errorf("publish pinned Structurizr: %w", err)
	}
	if err := validateStructurizrCache(target); err != nil {
		return "", err
	}
	return structurizrLauncher(target), nil
}

func structurizrLauncher(root string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(root, "structurizr.bat")
	}
	return filepath.Join(root, "structurizr.sh")
}

func validateStructurizrCache(root string) error {
	body, err := safefile.Read(filepath.Join(root, ".machinery-structurizr-receipt"), "structurizr receipt", structurizrReceiptMaxBytes)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	if len(lines) != 3 || lines[0] != "archive_sha256="+machversion.StructurizrLinuxZipSHA256 || lines[2] != "version="+machversion.StructurizrVersion || !strings.HasPrefix(lines[1], "closure_sha256=") {
		return fmt.Errorf("structurizr receipt does not match the embedded release pin")
	}
	want := strings.TrimPrefix(lines[1], "closure_sha256=")
	if _, err := hex.DecodeString(want); err != nil || len(want) != sha256.Size*2 {
		return fmt.Errorf("structurizr receipt closure hash is malformed")
	}
	got, err := fingerprintStructurizrTree(root)
	if err != nil {
		return err
	}
	if fmt.Sprintf("%x", got) != want {
		return fmt.Errorf("structurizr cache closure differs from the checksum-verified archive receipt")
	}
	return nil
}

func downloadStructurizrArchive(url, destination string) (retErr error) {
	defer func() {
		if retErr != nil {
			removeErr := os.Remove(destination)
			if os.IsNotExist(removeErr) {
				removeErr = nil
			}
			retErr = errors.Join(retErr, removeErr)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := structurizrHTTPDo(request)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, response.Body.Close()) }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download pinned Structurizr: HTTP %s", response.Status)
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, structurizrArchiveMaxBytes+1))
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(copyErr, syncErr, closeErr); err != nil {
		return err
	}
	if written > structurizrArchiveMaxBytes {
		return fmt.Errorf("pinned Structurizr archive exceeds %d bytes", structurizrArchiveMaxBytes)
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if got != machversion.StructurizrLinuxZipSHA256 {
		return fmt.Errorf("pinned Structurizr archive checksum mismatch: got %s, want %s", got, machversion.StructurizrLinuxZipSHA256)
	}
	return nil
}

func extractStructurizrZip(archive, destination string) (retErr error) {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, closeStructurizrZip(reader)) }()
	files := 0
	var total int64
	for _, entry := range reader.File {
		name, err := safeStructurizrArchiveName(entry.Name)
		if err != nil {
			return err
		}
		mode := entry.Mode()
		if mode&os.ModeSymlink != 0 || !mode.IsRegular() && !mode.IsDir() {
			return fmt.Errorf("structurizr archive contains link or special entry %q", entry.Name)
		}
		target := filepath.Join(destination, name)
		if mode.IsDir() {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			continue
		}
		files++
		total += int64(entry.UncompressedSize64)
		if files > structurizrTreeMaxFiles || total > structurizrTreeMaxBytes {
			return fmt.Errorf("structurizr archive exceeds extraction bounds")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		in, err := entry.Open()
		if err != nil {
			return err
		}
		permissions := os.FileMode(0o600)
		if mode&0o111 != 0 || strings.HasSuffix(entry.Name, ".sh") || strings.HasSuffix(entry.Name, ".bat") {
			permissions = 0o700
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, permissions)
		if err != nil {
			return errors.Join(err, in.Close())
		}
		written, copyErr := io.Copy(out, io.LimitReader(in, int64(entry.UncompressedSize64)+1))
		if written != int64(entry.UncompressedSize64) {
			copyErr = errors.Join(copyErr, fmt.Errorf("structurizr archive entry %q size changed", entry.Name))
		}
		if err := errors.Join(copyErr, in.Close(), out.Close()); err != nil {
			return err
		}
	}
	return nil
}

func safeStructurizrArchiveName(name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	if name == "" || filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("structurizr archive contains unsafe path %q", name)
	}
	return clean, nil
}
