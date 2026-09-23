package formal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// These tests run fetchJar in separate test-binary processes, the shape
// `go test ./...` gives packages that cold-start one shared user cache. Each
// helper gets its own internal file-lock test root, exactly as distinct
// package test binaries do, so the only coordination the fetchers can share is
// whatever lives beside the cache itself.
const (
	jarFetchHelperEnv = "MACHINERY_TEST_JAR_FETCH_HELPER"
	jarFetchSyncEnv   = "MACHINERY_TEST_JAR_FETCH_SYNC"
	jarFetchIDEnv     = "MACHINERY_TEST_JAR_FETCH_ID"
	jarFetchPeersEnv  = "MACHINERY_TEST_JAR_FETCH_PEERS"
	jarFetchURLEnv    = "MACHINERY_TEST_JAR_FETCH_URL"
	jarFetchDestEnv   = "MACHINERY_TEST_JAR_FETCH_DEST"
	jarFetchSHAEnv    = "MACHINERY_TEST_JAR_FETCH_SHA"
	// jarFetchHold bounds how long the server holds a download open waiting
	// for every peer to request it as well. Racing fetchers all arrive at
	// once and the barrier opens immediately; with one fetcher per cache only
	// the winner downloads and the hold expires.
	jarFetchHold = 2 * time.Second
)

var jarFetchBody = bytes.Repeat([]byte("pinned formal jar bytes\n"), 4096)

func jarFetchSHA() string {
	sum := sha256.Sum256(jarFetchBody)
	return hex.EncodeToString(sum[:])
}

// jarFetchServer serves the pinned body. /race holds each response until
// peers requests arrived or the hold expires; /crash sends half the body and
// then stalls until the test ends; /ok serves at once. requests counts every
// download of any kind.
type jarFetchServer struct {
	*httptest.Server
	requests atomic.Int32
	holding  chan struct{}
}

func newJarFetchServer(t *testing.T, peers int) *jarFetchServer {
	t.Helper()
	server := &jarFetchServer{holding: make(chan struct{})}
	release := make(chan struct{})
	var holdOnce sync.Once
	server.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := server.requests.Add(1)
		w.Header().Set("Content-Length", strconv.Itoa(len(jarFetchBody)))
		switch r.URL.Path {
		case "/race":
			deadline := time.Now().Add(jarFetchHold)
			for count < int32(peers) && time.Now().Before(deadline) {
				time.Sleep(5 * time.Millisecond)
				count = server.requests.Load()
			}
			_, _ = w.Write(jarFetchBody)
		case "/crash":
			_, _ = w.Write(jarFetchBody[:len(jarFetchBody)/2])
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			holdOnce.Do(func() { close(server.holding) })
			select {
			case <-release:
			case <-r.Context().Done():
			}
		default:
			_, _ = w.Write(jarFetchBody)
		}
	}))
	t.Cleanup(func() {
		close(release)
		server.Close()
	})
	return server
}

func jarFetchMark(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
		t.Fatal(err)
	}
}

func jarFetchCount(dir, prefix string) int {
	matches, _ := filepath.Glob(filepath.Join(dir, prefix+"*"))
	return len(matches)
}

func jarFetchAwait(dir, prefix string, want int, bound time.Duration) bool {
	deadline := time.Now().Add(bound)
	for time.Now().Before(deadline) {
		if jarFetchCount(dir, prefix) >= want {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return jarFetchCount(dir, prefix) >= want
}

// TestJarFetchHelper is the child side of the cross-process fetch tests. It
// does nothing unless one of them launched it.
func TestJarFetchHelper(t *testing.T) {
	mode := os.Getenv(jarFetchHelperEnv)
	if mode == "" {
		return
	}
	sync, id := os.Getenv(jarFetchSyncEnv), os.Getenv(jarFetchIDEnv)
	peers, err := strconv.Atoi(os.Getenv(jarFetchPeersEnv))
	if err != nil || sync == "" || id == "" {
		t.Fatalf("malformed helper environment: sync=%q id=%q peers=%v", sync, id, err)
	}
	dest, url, sha := os.Getenv(jarFetchDestEnv), os.Getenv(jarFetchURLEnv), os.Getenv(jarFetchSHAEnv)
	jarFetchMark(t, sync, "ready-"+id)
	if !jarFetchAwait(sync, "ready-", peers, 30*time.Second) {
		t.Fatal("start barrier never opened")
	}
	got, err := fetchJar(dest, url, "test jar", sha)
	if mode == "crash" {
		t.Fatalf("crash helper returned instead of stalling: %q, %v", got, err)
	}
	if err != nil {
		t.Fatalf("fetcher %s: %v", id, err)
	}
	if err := os.WriteFile(filepath.Join(sync, "result-"+id), []byte(got), 0o600); err != nil {
		t.Fatal(err)
	}
}

func startJarFetchHelper(t *testing.T, mode, sync, id string, peers int, dest, url string) (*exec.Cmd, *bytes.Buffer) {
	t.Helper()
	// Helpers are killed explicitly when a test chooses the moment, so the
	// command context is never canceled.
	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestJarFetchHelper$", "-test.count=1")
	cmd.Env = append(os.Environ(),
		jarFetchHelperEnv+"="+mode,
		jarFetchSyncEnv+"="+sync,
		jarFetchIDEnv+"="+id,
		jarFetchPeersEnv+"="+strconv.Itoa(peers),
		jarFetchDestEnv+"="+dest,
		jarFetchURLEnv+"="+url,
		jarFetchSHAEnv+"="+jarFetchSHA(),
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

// jarFetchCache returns an empty cache ancestor and the jar destination in
// its machinery cache parent, the layout jarPath and alloyJarPath resolve.
func jarFetchCache(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "machinery", "tla2tools-test.jar")
}

// raceJarFetchers starts peers fetchers of one jar into an empty cache behind
// a start barrier and returns what each reported and how many downloaded.
func raceJarFetchers(t *testing.T, dest string, peers int) ([]string, int) {
	t.Helper()
	server := newJarFetchServer(t, peers)
	sync := t.TempDir()
	cmds := make([]*exec.Cmd, peers)
	outputs := make([]*bytes.Buffer, peers)
	for index := range peers {
		cmds[index], outputs[index] = startJarFetchHelper(t, "race", sync, strconv.Itoa(index), peers, dest, server.URL+"/race")
	}
	var failures []string
	for index, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			failures = append(failures, fmt.Sprintf("fetcher %d: %v\n%s", index, err, outputs[index].String()))
		}
	}
	if len(failures) > 0 {
		t.Fatalf("%d of %d concurrent cold-cache fetchers failed:\n%s", len(failures), peers, strings.Join(failures, "\n"))
	}
	paths := make([]string, peers)
	for index := range peers {
		body, err := os.ReadFile(filepath.Join(sync, "result-"+strconv.Itoa(index)))
		if err != nil {
			t.Fatal(err)
		}
		paths[index] = string(body)
	}
	return paths, int(server.requests.Load())
}

// jarFetchSnapshot records every entry of the jar's cache parent with its
// type and permission bits and content digest.
func jarFetchSnapshot(t *testing.T, parent string) []string {
	t.Helper()
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot []string
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			t.Fatal(err)
		}
		line := fmt.Sprintf("%s %s", entry.Name(), info.Mode())
		if info.Mode().IsRegular() {
			body, err := os.ReadFile(filepath.Join(parent, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(body)
			line += " " + hex.EncodeToString(sum[:])
		}
		snapshot = append(snapshot, line)
	}
	sort.Strings(snapshot)
	return snapshot
}

func TestFetchJarConcurrentColdStartsShareOneFetcher(t *testing.T) {
	dest := jarFetchCache(t)
	paths, downloads := raceJarFetchers(t, dest, 2)
	if paths[0] != dest || paths[1] != dest {
		t.Fatalf("concurrent fetchers returned %q; want %q", paths, dest)
	}
	if downloads != 1 {
		t.Fatalf("%d fetchers downloaded into one cold cache; want exactly one", downloads)
	}
}

func TestFetchJarRacedCacheIsByteIdenticalToSingleFetch(t *testing.T) {
	singleDest := jarFetchCache(t)
	if _, downloads := raceJarFetchers(t, singleDest, 1); downloads != 1 {
		t.Fatalf("single fetcher downloaded %d times", downloads)
	}
	racedDest := jarFetchCache(t)
	paths, downloads := raceJarFetchers(t, racedDest, 5)
	for _, path := range paths {
		if path != racedDest {
			t.Fatalf("raced fetchers returned %q; want %q", paths, racedDest)
		}
	}
	if downloads != 1 {
		t.Fatalf("%d of 5 raced fetchers downloaded; want exactly one", downloads)
	}
	single, raced := jarFetchSnapshot(t, filepath.Dir(singleDest)), jarFetchSnapshot(t, filepath.Dir(racedDest))
	if strings.Join(single, "\n") != strings.Join(raced, "\n") {
		t.Fatalf("raced jar cache differs from a single fetch:\nsingle:\n%s\nraced:\n%s", strings.Join(single, "\n"), strings.Join(raced, "\n"))
	}
	if len(single) != 1 || !strings.HasPrefix(single[0], filepath.Base(singleDest)+" ") || !strings.HasSuffix(single[0], " "+jarFetchSHA()) {
		t.Fatalf("jar cache parent must hold exactly the verified jar:\n%s", strings.Join(single, "\n"))
	}
}

// startCrashedJarFetcher launches a helper that takes the jar-cache lock,
// receives half of the download into its private temporary file, and stalls
// until killed. It returns the helper, the server, and the live stage.
func startCrashedJarFetcher(t *testing.T, dest string) (*exec.Cmd, *jarFetchServer, string) {
	t.Helper()
	server := newJarFetchServer(t, 1)
	winner, output := startJarFetchHelper(t, "crash", t.TempDir(), "winner", 1, dest, server.URL+"/crash")
	select {
	case <-server.holding:
	case <-time.After(30 * time.Second):
		_ = winner.Process.Kill()
		_ = winner.Wait()
		t.Fatalf("crash helper never started downloading:\n%s", output.String())
	}
	stagePattern := filepath.Join(filepath.Dir(dest), formalJarStagePrefix(dest)+"*")
	deadline := time.Now().Add(10 * time.Second)
	for {
		stages, _ := filepath.Glob(stagePattern)
		if len(stages) == 1 {
			if info, err := os.Stat(stages[0]); err == nil && info.Size() == int64(len(jarFetchBody)/2) {
				return winner, server, stages[0]
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("winner stage never held the partial download: %v", stages)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

type jarFetchResult struct {
	path string
	err  error
}

func TestFetchJarWaiterRecoversFromCrashedWinner(t *testing.T) {
	dest := jarFetchCache(t)
	winner, server, stage := startCrashedJarFetcher(t, dest)
	before := server.requests.Load()
	done := make(chan jarFetchResult, 1)
	go func() {
		path, err := fetchJar(dest, server.URL+"/ok", "test jar", jarFetchSHA())
		done <- jarFetchResult{path, err}
	}()
	select {
	case got := <-done:
		t.Fatalf("waiter did not wait for the live winner: path=%q err=%v", got.path, got.err)
	case <-time.After(500 * time.Millisecond):
	}
	if info, err := os.Stat(stage); err != nil || info.Size() != int64(len(jarFetchBody)/2) {
		t.Fatalf("waiter disturbed the live winner's stage: %v, %v", info, err)
	}
	if err := winner.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = winner.Wait()
	var got jarFetchResult
	select {
	case got = <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("waiter deadlocked behind a crashed winner")
	}
	if got.err != nil || got.path != dest {
		t.Fatalf("waiter did not recover from the crashed winner: %q, %v", got.path, got.err)
	}
	if n := server.requests.Load() - before; n != 1 {
		t.Fatalf("recovering waiter downloaded %d times; want exactly one", n)
	}
	if _, err := os.Lstat(stage); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("crashed winner's stage survived recovery: %v", err)
	}
	if body, err := os.ReadFile(dest); err != nil || !bytes.Equal(body, jarFetchBody) {
		t.Fatalf("recovered jar = %d bytes, %v", len(body), err)
	}
	if snapshot := jarFetchSnapshot(t, filepath.Dir(dest)); len(snapshot) != 1 {
		t.Fatalf("fetch left residue beside the jar:\n%s", strings.Join(snapshot, "\n"))
	}
}
