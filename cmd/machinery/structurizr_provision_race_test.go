package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	machversion "github.com/RamXX/machinery/internal/version"
)

// These tests run the Structurizr provisioner in separate test-binary
// processes, the shape `go test ./...` gives packages that cold-start one
// shared user cache. Each helper gets its own internal file-lock test root,
// exactly as distinct package test binaries do. This package's TestMain
// sandboxes HOME per process, so helpers point themselves at the shared cache
// explicitly.
const (
	structurizrProvisionHelperEnv = "MACHINERY_TEST_STRUCTURIZR_PROVISION_HELPER"
	structurizrProvisionSyncEnv   = "MACHINERY_TEST_STRUCTURIZR_PROVISION_SYNC"
	structurizrProvisionIDEnv     = "MACHINERY_TEST_STRUCTURIZR_PROVISION_ID"
	structurizrProvisionPeersEnv  = "MACHINERY_TEST_STRUCTURIZR_PROVISION_PEERS"
	structurizrProvisionCacheEnv  = "MACHINERY_TEST_STRUCTURIZR_PROVISION_CACHE"
	// structurizrProvisionHold bounds how long a downloading provisioner
	// waits for its peers to reach the download too. Racing provisioners all
	// arrive at once; with one provisioner per cache the hold expires and the
	// winner proceeds alone.
	structurizrProvisionHold = 2 * time.Second
)

// structurizrFakeArchive returns a deterministic archive with the launcher
// layout the provisioner requires.
func structurizrFakeArchive(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range []struct {
		name, body string
		mode       os.FileMode
	}{
		{"structurizr.sh", "#!/bin/sh\necho fake\n", 0o755},
		{"structurizr.bat", "@echo fake\r\n", 0o644},
		{"lib/structurizr-cli.jar", "fake jar", 0o644},
	} {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Store}
		header.SetMode(entry.mode)
		out, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := out.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func installStructurizrFakeDownload(t *testing.T, download func(destination string) error) {
	t.Helper()
	original := downloadStructurizrArchive
	t.Cleanup(func() { downloadStructurizrArchive = original })
	downloadStructurizrArchive = func(_, destination string) error { return download(destination) }
}

func structurizrProvisionMark(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
		t.Fatal(err)
	}
}

func structurizrProvisionCount(dir, prefix string) int {
	matches, _ := filepath.Glob(filepath.Join(dir, prefix+"*"))
	return len(matches)
}

func structurizrProvisionAwait(dir, prefix string, want int, bound time.Duration) bool {
	deadline := time.Now().Add(bound)
	for time.Now().Before(deadline) {
		if structurizrProvisionCount(dir, prefix) >= want {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return structurizrProvisionCount(dir, prefix) >= want
}

func setStructurizrCacheRoot(t *testing.T, cacheRoot string) {
	t.Helper()
	t.Setenv("HOME", cacheRoot)
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	t.Setenv("LOCALAPPDATA", cacheRoot)
}

// TestStructurizrProvisionHelper is the child side of the cross-process
// provisioning tests. It does nothing unless one of them launched it.
func TestStructurizrProvisionHelper(t *testing.T) {
	mode := os.Getenv(structurizrProvisionHelperEnv)
	if mode == "" {
		return
	}
	sync, id := os.Getenv(structurizrProvisionSyncEnv), os.Getenv(structurizrProvisionIDEnv)
	peers, err := strconv.Atoi(os.Getenv(structurizrProvisionPeersEnv))
	if err != nil || sync == "" || id == "" {
		t.Fatalf("malformed helper environment: sync=%q id=%q peers=%v", sync, id, err)
	}
	setStructurizrCacheRoot(t, os.Getenv(structurizrProvisionCacheEnv))
	archive := structurizrFakeArchive(t)
	switch mode {
	case "race":
		installStructurizrFakeDownload(t, func(destination string) error {
			structurizrProvisionMark(t, sync, "download-"+id)
			structurizrProvisionAwait(sync, "download-", peers, structurizrProvisionHold)
			return os.WriteFile(destination, archive, 0o600)
		})
		structurizrProvisionMark(t, sync, "ready-"+id)
		if !structurizrProvisionAwait(sync, "ready-", peers, 30*time.Second) {
			t.Fatal("start barrier never opened")
		}
		got, err := provisionStructurizr()
		if err != nil {
			t.Fatalf("provisioner %s: %v", id, err)
		}
		if err := os.WriteFile(filepath.Join(sync, "result-"+id), []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
	case "crash":
		installStructurizrFakeDownload(t, func(destination string) error {
			if err := os.WriteFile(destination, archive[:len(archive)/2], 0o600); err != nil {
				return err
			}
			structurizrProvisionMark(t, sync, "holding-"+id)
			select {}
		})
		_, err := provisionStructurizr()
		t.Fatalf("crash helper returned instead of blocking: %v", err)
	default:
		t.Fatalf("unknown helper mode %q", mode)
	}
}

func startStructurizrProvisionHelper(t *testing.T, cacheRoot, sync, mode, id string, peers int) (*exec.Cmd, *bytes.Buffer) {
	t.Helper()
	// Helpers are killed explicitly when a test chooses the moment, so the
	// command context is never canceled.
	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestStructurizrProvisionHelper$", "-test.count=1")
	cmd.Env = append(os.Environ(),
		structurizrProvisionHelperEnv+"="+mode,
		structurizrProvisionSyncEnv+"="+sync,
		structurizrProvisionIDEnv+"="+id,
		structurizrProvisionPeersEnv+"="+strconv.Itoa(peers),
		structurizrProvisionCacheEnv+"="+cacheRoot,
		"MACHINERY_INTERNAL_TEST_LOCK_ROOT="+t.TempDir(),
	)
	output := &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = output, output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	return cmd, output
}

// structurizrProvisionCache returns an empty cache root and the pinned
// target os.UserCacheDir resolves beneath it on this platform.
func structurizrProvisionCache(t *testing.T) (string, string) {
	t.Helper()
	cacheRoot := t.TempDir()
	setStructurizrCacheRoot(t, cacheRoot)
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(cache, "machinery", "structurizr")
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatal(err)
	}
	return cacheRoot, filepath.Join(base, machversion.StructurizrVersion)
}

func raceStructurizrProvisioners(t *testing.T, cacheRoot string, peers int) ([]string, int) {
	t.Helper()
	sync := t.TempDir()
	cmds := make([]*exec.Cmd, peers)
	outputs := make([]*bytes.Buffer, peers)
	for index := range peers {
		cmds[index], outputs[index] = startStructurizrProvisionHelper(t, cacheRoot, sync, "race", strconv.Itoa(index), peers)
	}
	var failures []string
	for index, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			failures = append(failures, fmt.Sprintf("provisioner %d: %v\n%s", index, err, outputs[index].String()))
		}
	}
	if len(failures) > 0 {
		t.Fatalf("%d of %d concurrent cold-start provisioners failed:\n%s", len(failures), peers, strings.Join(failures, "\n"))
	}
	paths := make([]string, peers)
	for index := range peers {
		body, err := os.ReadFile(filepath.Join(sync, "result-"+strconv.Itoa(index)))
		if err != nil {
			t.Fatal(err)
		}
		paths[index] = string(body)
	}
	return paths, structurizrProvisionCount(sync, "download-")
}

func structurizrProvisionSnapshot(t *testing.T, target string) []string {
	t.Helper()
	var snapshot []string
	err := filepath.WalkDir(target, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(target, path)
		if err != nil {
			return err
		}
		line := fmt.Sprintf("%s %s", filepath.ToSlash(rel), info.Mode())
		if info.Mode().IsRegular() {
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(body)
			line += " " + hex.EncodeToString(sum[:])
		}
		snapshot = append(snapshot, line)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestProvisionStructurizrConcurrentColdStartsShareOneProvisioner(t *testing.T) {
	cacheRoot, target := structurizrProvisionCache(t)
	paths, downloads := raceStructurizrProvisioners(t, cacheRoot, 2)
	if paths[0] != paths[1] || paths[0] != structurizrLauncher(target) {
		t.Fatalf("concurrent provisioners returned %q; want %q", paths, structurizrLauncher(target))
	}
	if downloads != 1 {
		t.Fatalf("%d provisioners downloaded into one cold cache; want exactly one", downloads)
	}
}

func TestProvisionStructurizrRacedCacheIsByteIdenticalToSingleProvision(t *testing.T) {
	singleRoot, singleTarget := structurizrProvisionCache(t)
	if _, downloads := raceStructurizrProvisioners(t, singleRoot, 1); downloads != 1 {
		t.Fatalf("single provisioner downloaded %d times", downloads)
	}
	racedRoot, racedTarget := structurizrProvisionCache(t)
	paths, downloads := raceStructurizrProvisioners(t, racedRoot, 5)
	for _, path := range paths {
		if path != structurizrLauncher(racedTarget) {
			t.Fatalf("raced provisioners returned %q", paths)
		}
	}
	if downloads != 1 {
		t.Fatalf("%d of 5 raced provisioners downloaded; want exactly one", downloads)
	}
	single, raced := structurizrProvisionSnapshot(t, singleTarget), structurizrProvisionSnapshot(t, racedTarget)
	if strings.Join(single, "\n") != strings.Join(raced, "\n") {
		t.Fatalf("raced runtime differs from a single provision:\nsingle:\n%s\nraced:\n%s", strings.Join(single, "\n"), strings.Join(raced, "\n"))
	}
	if !strings.Contains(strings.Join(single, "\n"), ".machinery-structurizr-receipt ") {
		t.Fatalf("snapshot omits the recorded receipt:\n%s", strings.Join(single, "\n"))
	}
}

func startCrashedStructurizrWinner(t *testing.T, cacheRoot, base string) (*exec.Cmd, string) {
	t.Helper()
	sync := t.TempDir()
	winner, output := startStructurizrProvisionHelper(t, cacheRoot, sync, "crash", "winner", 1)
	if !structurizrProvisionAwait(sync, "holding-", 1, 30*time.Second) {
		_ = winner.Process.Kill()
		_ = winner.Wait()
		t.Fatalf("crash helper never started provisioning:\n%s", output.String())
	}
	stages, err := filepath.Glob(filepath.Join(base, ".structurizr-stage-*"))
	if err != nil || len(stages) != 1 {
		t.Fatalf("winner stage = %v, %v; want exactly one live stage", stages, err)
	}
	return winner, stages[0]
}

type structurizrProvisionResult struct {
	path string
	err  error
}

func TestProvisionStructurizrWaiterRecoversFromCrashedWinner(t *testing.T) {
	cacheRoot, target := structurizrProvisionCache(t)
	base := filepath.Dir(target)
	winner, stage := startCrashedStructurizrWinner(t, cacheRoot, base)
	archive := structurizrFakeArchive(t)
	var downloads atomic.Int32
	installStructurizrFakeDownload(t, func(destination string) error {
		downloads.Add(1)
		return os.WriteFile(destination, archive, 0o600)
	})
	done := make(chan structurizrProvisionResult, 1)
	go func() {
		path, err := provisionStructurizr()
		done <- structurizrProvisionResult{path, err}
	}()
	select {
	case got := <-done:
		t.Fatalf("waiter did not wait for the live winner: path=%q err=%v", got.path, got.err)
	case <-time.After(500 * time.Millisecond):
	}
	if body, err := os.ReadFile(filepath.Join(stage, "structurizr-cli.zip")); err != nil || len(body) != len(archive)/2 {
		t.Fatalf("waiter disturbed the live winner's stage: %d bytes, %v", len(body), err)
	}
	if err := winner.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = winner.Wait()
	var got structurizrProvisionResult
	select {
	case got = <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("waiter deadlocked behind a crashed winner")
	}
	if got.err != nil || got.path != structurizrLauncher(target) {
		t.Fatalf("waiter did not recover from the crashed winner: %q, %v", got.path, got.err)
	}
	if n := downloads.Load(); n != 1 {
		t.Fatalf("recovering waiter downloaded %d times; want exactly one", n)
	}
	if _, err := os.Lstat(stage); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("crashed winner's stage survived recovery: %v", err)
	}
	if left, _ := filepath.Glob(filepath.Join(base, ".structurizr-stage-*")); len(left) != 0 {
		t.Fatalf("provision left stage residue: %v", left)
	}
}
