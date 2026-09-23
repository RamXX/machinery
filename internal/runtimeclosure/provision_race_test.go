package runtimeclosure

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// The tests in this file run the pinned-Java provisioner in separate
// test-binary processes, the shape `go test ./...` gives several packages that
// cold-start one shared user cache. Each helper gets its own internal
// file-lock test root, exactly as distinct package test binaries do, so the
// only coordination the provisioners can share is whatever lives beside the
// cache itself.
const (
	javaProvisionHelperEnv = "MACHINERY_TEST_JAVA_PROVISION_HELPER"
	javaProvisionSyncEnv   = "MACHINERY_TEST_JAVA_PROVISION_SYNC"
	javaProvisionIDEnv     = "MACHINERY_TEST_JAVA_PROVISION_ID"
	javaProvisionPeersEnv  = "MACHINERY_TEST_JAVA_PROVISION_PEERS"
	// javaProvisionHold bounds how long a downloading provisioner waits for
	// its peers to reach the download as well. On a racing implementation
	// every peer arrives at once and the barrier opens immediately; with one
	// provisioner per cache target only the winner downloads, the hold
	// expires, and the winner proceeds alone.
	javaProvisionHold = 2 * time.Second
)

// javaProvisionFakeArchive returns a deterministic runtime archive with the
// layout javaHomeInInstall requires on the running platform.
func javaProvisionFakeArchive(t *testing.T) []byte {
	t.Helper()
	home := "jdk/"
	if runtime.GOOS == "darwin" {
		home = "jdk/Contents/Home/"
	}
	launcher := home + "bin/java"
	if runtime.GOOS == "windows" {
		launcher += ".exe"
	}
	config, modules := home+"conf/security.properties", home+"lib/modules"
	headers := []tar.Header{
		{Name: launcher, Typeflag: tar.TypeReg, Mode: 0o755},
		{Name: config, Typeflag: tar.TypeReg, Mode: 0o644},
		{Name: modules, Typeflag: tar.TypeReg, Mode: 0o644},
	}
	bodies := map[string]string{launcher: "fake launcher", config: "fake config", modules: "fake modules"}
	body, err := os.ReadFile(writeTarGzip(t, headers, bodies))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// installJavaProvisionFake pins archive by its real digest and routes the
// download through the caller's behavior for the rest of the test.
func installJavaProvisionFake(t *testing.T, archive []byte, download func(destination string) error) {
	t.Helper()
	key := runtime.GOOS + "/" + runtime.GOARCH
	originalDownload := downloadJavaRuntimeArchive
	originalPin := javaArchivePins[key]
	t.Cleanup(func() {
		downloadJavaRuntimeArchive = originalDownload
		javaArchivePins[key] = originalPin
	})
	sum := sha256.Sum256(archive)
	javaArchivePins[key] = javaArchivePin{asset: "test.tar.gz", sha: hex.EncodeToString(sum[:])}
	downloadJavaRuntimeArchive = func(_, destination, _ string) error { return download(destination) }
}

func javaProvisionMark(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
		t.Fatal(err)
	}
}

func javaProvisionCount(dir, prefix string) int {
	matches, _ := filepath.Glob(filepath.Join(dir, prefix+"*"))
	return len(matches)
}

// javaProvisionAwait waits until dir holds want markers carrying prefix or the
// bound elapses, and reports whether the barrier opened.
func javaProvisionAwait(dir, prefix string, want int, bound time.Duration) bool {
	deadline := time.Now().Add(bound)
	for time.Now().Before(deadline) {
		if javaProvisionCount(dir, prefix) >= want {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return javaProvisionCount(dir, prefix) >= want
}

// TestJavaProvisionHelper is the child side of the cross-process provisioning
// tests. It does nothing unless one of them launched it.
func TestJavaProvisionHelper(t *testing.T) {
	mode := os.Getenv(javaProvisionHelperEnv)
	if mode == "" {
		return
	}
	sync, id := os.Getenv(javaProvisionSyncEnv), os.Getenv(javaProvisionIDEnv)
	peers, err := strconv.Atoi(os.Getenv(javaProvisionPeersEnv))
	if err != nil || sync == "" || id == "" {
		t.Fatalf("malformed helper environment: sync=%q id=%q peers=%v", sync, id, err)
	}
	archive := javaProvisionFakeArchive(t)
	switch mode {
	case "race":
		installJavaProvisionFake(t, archive, func(destination string) error {
			javaProvisionMark(t, sync, "download-"+id)
			javaProvisionAwait(sync, "download-", peers, javaProvisionHold)
			return os.WriteFile(destination, archive, 0o600)
		})
		javaProvisionMark(t, sync, "ready-"+id)
		if !javaProvisionAwait(sync, "ready-", peers, 30*time.Second) {
			t.Fatal("start barrier never opened")
		}
		got, err := provisionedJavaPath()
		if err != nil {
			t.Fatalf("provisioner %s: %v", id, err)
		}
		if err := os.WriteFile(filepath.Join(sync, "result-"+id), []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
	case "crash":
		// The winner dies holding the provisioning lock with a partial
		// archive in its stage: the SIGKILL residue a waiter must recover
		// from without deadlocking.
		installJavaProvisionFake(t, archive, func(destination string) error {
			if err := os.WriteFile(destination, archive[:len(archive)/2], 0o600); err != nil {
				return err
			}
			javaProvisionMark(t, sync, "holding-"+id)
			select {}
		})
		_, err := provisionedJavaPath()
		t.Fatalf("crash helper returned instead of blocking: %v", err)
	default:
		t.Fatalf("unknown helper mode %q", mode)
	}
}

// startJavaProvisionHelper launches one helper provisioner against cacheRoot
// with its own isolated file-lock test root.
func startJavaProvisionHelper(t *testing.T, cacheRoot, sync, mode, id string, peers int) (*exec.Cmd, *bytes.Buffer) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestJavaProvisionHelper$", "-test.count=1")
	cmd.Env = append(os.Environ(),
		javaProvisionHelperEnv+"="+mode,
		javaProvisionSyncEnv+"="+sync,
		javaProvisionIDEnv+"="+id,
		javaProvisionPeersEnv+"="+strconv.Itoa(peers),
		"MACHINERY_INTERNAL_TEST_LOCK_ROOT="+t.TempDir(),
		"HOME="+cacheRoot,
		"XDG_CACHE_HOME="+cacheRoot,
		"LOCALAPPDATA="+cacheRoot,
		JavaEnv+"=",
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

// raceJavaProvisioners starts peers helper provisioners on one empty cache
// behind a start barrier, and returns the launcher each reported and how many
// of them downloaded.
func raceJavaProvisioners(t *testing.T, cacheRoot string, peers int) ([]string, int) {
	t.Helper()
	sync := t.TempDir()
	cmds := make([]*exec.Cmd, peers)
	outputs := make([]*bytes.Buffer, peers)
	for index := range peers {
		cmds[index], outputs[index] = startJavaProvisionHelper(t, cacheRoot, sync, "race", strconv.Itoa(index), peers)
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
	return paths, javaProvisionCount(sync, "download-")
}

// javaProvisionCacheRoot returns an empty cache root and the pinned runtime
// target os.UserCacheDir resolves beneath it on this platform.
func javaProvisionCacheRoot(t *testing.T) (string, string) {
	t.Helper()
	cacheRoot := t.TempDir()
	base := filepath.Join(cacheRoot, "machinery", "java")
	switch runtime.GOOS {
	case "darwin":
		base = filepath.Join(cacheRoot, "Library", "Caches", "machinery", "java")
	case "windows", "linux", "freebsd", "openbsd", "netbsd", "dragonfly", "illumos", "solaris", "aix":
	default:
		t.Skipf("user cache layout unknown on %s", runtime.GOOS)
	}
	return cacheRoot, filepath.Join(base, strings.ReplaceAll(PinnedJavaRuntimeVersion, "+", "_"), runtime.GOOS+"-"+runtime.GOARCH)
}

// javaProvisionSnapshot records every entry under target with its relative
// path, type and permission bits, and content digest.
func javaProvisionSnapshot(t *testing.T, target string) []string {
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

func TestProvisionJavaConcurrentColdStartsShareOneProvisioner(t *testing.T) {
	if _, supported := javaArchivePins[runtime.GOOS+"/"+runtime.GOARCH]; !supported {
		t.Skip("no Java archive pin for this test platform")
	}
	cacheRoot, _ := javaProvisionCacheRoot(t)
	paths, downloads := raceJavaProvisioners(t, cacheRoot, 2)
	if paths[0] != paths[1] {
		t.Fatalf("concurrent provisioners returned different launchers: %q vs %q", paths[0], paths[1])
	}
	if downloads != 1 {
		t.Fatalf("%d provisioners downloaded into one cold cache target; want exactly one", downloads)
	}
}

func TestProvisionJavaRacedCacheIsByteIdenticalToSingleProvision(t *testing.T) {
	if _, supported := javaArchivePins[runtime.GOOS+"/"+runtime.GOARCH]; !supported {
		t.Skip("no Java archive pin for this test platform")
	}
	singleRoot, singleTarget := javaProvisionCacheRoot(t)
	if _, downloads := raceJavaProvisioners(t, singleRoot, 1); downloads != 1 {
		t.Fatalf("single provisioner downloaded %d times", downloads)
	}
	racedRoot, racedTarget := javaProvisionCacheRoot(t)
	paths, downloads := raceJavaProvisioners(t, racedRoot, 5)
	for _, path := range paths[1:] {
		if path != paths[0] {
			t.Fatalf("raced provisioners returned different launchers: %q", paths)
		}
	}
	if downloads != 1 {
		t.Fatalf("%d of 5 raced provisioners downloaded; want exactly one", downloads)
	}
	single, raced := javaProvisionSnapshot(t, singleTarget), javaProvisionSnapshot(t, racedTarget)
	if strings.Join(single, "\n") != strings.Join(raced, "\n") {
		t.Fatalf("raced runtime differs from a single provision:\nsingle:\n%s\nraced:\n%s", strings.Join(single, "\n"), strings.Join(raced, "\n"))
	}
	var receipt bool
	for _, line := range single {
		if strings.Contains(line, ".machinery-java-receipt ") {
			receipt = true
		}
	}
	if !receipt {
		t.Fatalf("snapshot omits the recorded receipt:\n%s", strings.Join(single, "\n"))
	}
}

// startCrashedJavaWinner launches a helper that takes the provisioning lock,
// leaves a partial archive in its stage, and blocks until killed. It returns
// the helper and its one live stage.
func startCrashedJavaWinner(t *testing.T, base string) (*exec.Cmd, string) {
	t.Helper()
	sync := t.TempDir()
	winner, output := startJavaProvisionHelper(t, os.Getenv("HOME"), sync, "crash", "winner", 1)
	if !javaProvisionAwait(sync, "holding-", 1, 30*time.Second) {
		_ = winner.Process.Kill()
		_ = winner.Wait()
		t.Fatalf("crash helper never started provisioning:\n%s", output.String())
	}
	stages, err := filepath.Glob(filepath.Join(base, ".java-stage-*"))
	if err != nil || len(stages) != 1 {
		t.Fatalf("winner stage = %v, %v; want exactly one live stage", stages, err)
	}
	return winner, stages[0]
}

func TestProvisionJavaWaiterHonorsCancellationWhileWinnerProvisions(t *testing.T) {
	if _, supported := javaArchivePins[runtime.GOOS+"/"+runtime.GOARCH]; !supported {
		t.Skip("no Java archive pin for this test platform")
	}
	base := javaProvisionTestBase(t)
	winner, stage := startCrashedJavaWinner(t, base)
	defer func() {
		_ = winner.Process.Kill()
		_ = winner.Wait()
	}()
	archive := javaProvisionFakeArchive(t)
	var downloads atomic.Int32
	installJavaProvisionFake(t, archive, func(destination string) error {
		downloads.Add(1)
		return os.WriteFile(destination, archive, 0o600)
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan javaProvisionResult, 1)
	go func() {
		path, err := provisionedJavaPathContext(ctx)
		done <- javaProvisionResult{path, err}
	}()
	select {
	case got := <-done:
		t.Fatalf("waiter returned while the winner held the lock: path=%q err=%v", got.path, got.err)
	case <-time.After(200 * time.Millisecond):
	}
	cancel()
	var got javaProvisionResult
	select {
	case got = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("canceled waiter did not return")
	}
	if got.path != "" || !errors.Is(got.err, context.Canceled) {
		t.Fatalf("canceled waiter = path %q, err %v; want context.Canceled", got.path, got.err)
	}
	if n := downloads.Load(); n != 0 {
		t.Fatalf("canceled waiter downloaded %d times", n)
	}
	if body, err := os.ReadFile(filepath.Join(stage, "runtime.archive")); err != nil || len(body) != len(archive)/2 {
		t.Fatalf("canceled waiter disturbed the winner's stage: %d bytes, %v", len(body), err)
	}
	if winner.ProcessState != nil {
		t.Fatalf("winner exited while the waiter was canceled: %v", winner.ProcessState)
	}
}

type javaProvisionResult struct {
	path string
	err  error
}

func TestProvisionJavaWaiterRecoversFromCrashedWinner(t *testing.T) {
	if _, supported := javaArchivePins[runtime.GOOS+"/"+runtime.GOARCH]; !supported {
		t.Skip("no Java archive pin for this test platform")
	}
	base := javaProvisionTestBase(t)
	winner, stage := startCrashedJavaWinner(t, base)
	archive := javaProvisionFakeArchive(t)
	var downloads atomic.Int32
	installJavaProvisionFake(t, archive, func(destination string) error {
		downloads.Add(1)
		return os.WriteFile(destination, archive, 0o600)
	})
	done := make(chan javaProvisionResult, 1)
	go func() {
		path, err := provisionedJavaPath()
		done <- javaProvisionResult{path, err}
	}()
	select {
	case got := <-done:
		t.Fatalf("waiter did not wait for the live winner: path=%q err=%v", got.path, got.err)
	case <-time.After(500 * time.Millisecond):
	}
	if body, err := os.ReadFile(filepath.Join(stage, "runtime.archive")); err != nil || len(body) != len(archive)/2 {
		t.Fatalf("waiter disturbed the live winner's stage: %d bytes, %v", len(body), err)
	}
	if err := winner.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = winner.Wait()
	var got javaProvisionResult
	select {
	case got = <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("waiter deadlocked behind a crashed winner")
	}
	if got.err != nil {
		t.Fatalf("waiter did not recover from the crashed winner: %v", got.err)
	}
	if n := downloads.Load(); n != 1 {
		t.Fatalf("recovering waiter downloaded %d times; want exactly one", n)
	}
	if _, err := os.Lstat(stage); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("crashed winner's stage survived recovery: %v", err)
	}
	if left, _ := filepath.Glob(filepath.Join(base, ".java-stage-*")); len(left) != 0 {
		t.Fatalf("provision left stage residue: %v", left)
	}
}
