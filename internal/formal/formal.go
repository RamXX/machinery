// Package formal is the Go port of verify_formal.sh + tlc.sh: regenerates the
// formal suite from source and runs the TLC model checker, shelling out to java.
package formal

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/alloy"
	"github.com/RamXX/machinery/internal/artifactset"
	"github.com/RamXX/machinery/internal/compose"
	"github.com/RamXX/machinery/internal/designlock"
	"github.com/RamXX/machinery/internal/filelock"
	"github.com/RamXX/machinery/internal/fsatomic"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/pack"
	"github.com/RamXX/machinery/internal/portablepath"
	"github.com/RamXX/machinery/internal/refine"
	"github.com/RamXX/machinery/internal/runtimeclosure"
	"github.com/RamXX/machinery/internal/tla"
	"github.com/RamXX/machinery/internal/version"
)

const (
	tlaVersion             = "v1.7.4"
	tlaSHA256              = "936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88"
	formalJarDownloadLimit = int64(128 << 20)
	formalArtifactMaxBytes = int64(16 << 20)
	formalDirPageSize      = 128
	formalDirEntryMax      = 4096
)

func copyFormalExact(dst io.Writer, src *os.File, size int64, label string) (int64, error) {
	written, err := io.CopyN(dst, src, size)
	if err != nil {
		return written, fmt.Errorf("read exact %s: copied %d of %d bytes: %w", label, written, size, err)
	}
	var extra [1]byte
	n, probeErr := src.Read(extra[:])
	if n != 0 || probeErr != io.EOF {
		return written, errors.Join(probeErr, fmt.Errorf("%s grew while being read", label))
	}
	return written, nil
}

func readFormalFileExact(path, label string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > formalArtifactMaxBytes {
		return nil, fmt.Errorf("%s must be a regular non-symlink file no larger than %d bytes", label, formalArtifactMaxBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, statErr := file.Stat()
	if statErr != nil || !os.SameFile(info, opened) || opened.Mode() != info.Mode() || opened.Size() != info.Size() {
		return nil, errors.Join(statErr, file.Close(), fmt.Errorf("%s changed identity while opening", label))
	}
	var body bytes.Buffer
	body.Grow(int(info.Size()))
	written, readErr := copyFormalExact(&body, file, info.Size(), label)
	after, pathErr := os.Lstat(path)
	closeErr := file.Close()
	if err := errors.Join(readErr, pathErr, closeErr); err != nil {
		return nil, err
	}
	if written != info.Size() || !os.SameFile(info, after) || info.Mode() != after.Mode() || info.Size() != after.Size() || !info.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("%s changed while reading", label)
	}
	return body.Bytes(), nil
}

var formalUserCacheDir = os.UserCacheDir

// jarPath resolves the pinned tla2tools.jar location (env override honored).
func jarPath() (string, error) {
	if j := os.Getenv("TLA_TOOLS_JAR"); j != "" {
		return j, nil
	}
	cache, err := formalUserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache directory for pinned TLA+ tool: %w", err)
	}
	if cache == "" {
		return "", fmt.Errorf("resolve user cache directory for pinned TLA+ tool: empty path")
	}
	return filepath.Join(cache, "machinery", "tla2tools-"+tlaVersion+".jar"), nil
}

// ensureJar fetches+checksum-verifies the pinned jar on first use (like tlc.sh).
func ensureJar() (string, error) {
	want, err := overrideSHA("TLA_TOOLS_JAR", "TLA_TOOLS_JAR_SHA256", tlaSHA256)
	if err != nil {
		return "", err
	}
	path, err := jarPath()
	if err != nil {
		return "", err
	}
	return fetchJar(path,
		"https://github.com/tlaplus/tlaplus/releases/download/"+tlaVersion+"/tla2tools.jar",
		"tla2tools.jar "+tlaVersion, want)
}

var sha256Re = regexp.MustCompile(`^[0-9a-f]{64}$`)

// overrideSHA keeps a path override pinned by default. A deliberately custom
// jar must carry its explicit checksum in the companion environment variable;
// an invalid checksum never weakens verification.
func overrideSHA(pathEnv, shaEnv, pinned string) (string, error) {
	want := pinned
	if os.Getenv(pathEnv) != "" && os.Getenv(shaEnv) != "" {
		want = strings.ToLower(strings.TrimSpace(os.Getenv(shaEnv)))
	}
	if !sha256Re.MatchString(want) {
		return "", fmt.Errorf("%s must be exactly 64 lowercase hexadecimal characters", shaEnv)
	}
	return want, nil
}

func fileSHA256(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s must be a regular non-symlink jar", path)
	}
	if info.Size() <= 0 || info.Size() > formalJarDownloadLimit {
		return "", fmt.Errorf("%s has invalid jar size %d (limit %d)", path, info.Size(), formalJarDownloadLimit)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) || opened.Mode() != info.Mode() || opened.Size() != info.Size() {
		if err != nil {
			return "", errors.Join(err, f.Close())
		}
		return "", errors.Join(fmt.Errorf("%s changed while opening", path), f.Close())
	}
	h := sha256.New()
	written, copyErr := copyFormalExact(h, f, info.Size(), path)
	pathInfo, statErr := os.Lstat(path)
	closeErr := f.Close()
	if err := errors.Join(copyErr, statErr, closeErr); err != nil {
		return "", err
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(info, pathInfo) || pathInfo.Mode() != info.Mode() || pathInfo.Size() != info.Size() || !pathInfo.ModTime().Equal(info.ModTime()) || written != info.Size() {
		return "", fmt.Errorf("%s changed identity while hashing", path)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func snapshotVerifiedJar(path, wantSHA, label, dir string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s must be a regular non-symlink jar", path)
	}
	if info.Size() <= 0 || info.Size() > formalJarDownloadLimit {
		return "", fmt.Errorf("%s has invalid jar size %d (limit %d)", path, info.Size(), formalJarDownloadLimit)
	}
	in, err := os.Open(path)
	if err != nil {
		return "", err
	}
	opened, err := in.Stat()
	if err != nil || !os.SameFile(info, opened) || opened.Mode() != info.Mode() || opened.Size() != info.Size() {
		if err != nil {
			return "", errors.Join(err, in.Close())
		}
		return "", errors.Join(fmt.Errorf("%s changed while opening", path), in.Close())
	}
	dest := filepath.Join(dir, "verified-tool.jar")
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", errors.Join(err, in.Close())
	}
	h := sha256.New()
	written, copyErr := copyFormalExact(io.MultiWriter(out, h), in, info.Size(), label+" jar")
	syncErr := out.Sync()
	closeOutErr := out.Close()
	pathInfo, statErr := os.Lstat(path)
	closeInErr := in.Close()
	if err := errors.Join(copyErr, syncErr, closeOutErr, statErr, closeInErr); err != nil {
		return "", err
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(info, pathInfo) || pathInfo.Mode() != info.Mode() || pathInfo.Size() != info.Size() || !pathInfo.ModTime().Equal(info.ModTime()) || written != info.Size() {
		return "", fmt.Errorf("%s changed identity while snapshotting", path)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != wantSHA {
		return "", fmt.Errorf("checksum mismatch for %s snapshot: got %s, want %s", label, got, wantSHA)
	}
	return dest, nil
}

var formalAfterJarCacheLock func(string)

func formalJarStagePrefix(dest string) string {
	sum := sha256.Sum256([]byte(filepath.Base(dest)))
	return ".machinery-jar-" + hex.EncodeToString(sum[:8]) + "-"
}

func formalJarLockScope(parent string) string {
	// Scope the advisory lock to the existing cache ancestor, not to any path
	// beneath the replaceable cache parent. A predictable missing sibling could
	// itself be created as a symlink between acquisitions and split the derived
	// file-lock identity. Serializing sibling formal caches is conservative but
	// keeps every acquisition bound to the same stable namespace authority.
	return filepath.Dir(parent)
}

func sameFormalJarCacheIdentity(a, b os.FileInfo) bool {
	return a != nil && b != nil && a.IsDir() && b.IsDir() && a.Mode()&os.ModeSymlink == 0 && b.Mode()&os.ModeSymlink == 0 && a.Mode() == b.Mode() && os.SameFile(a, b)
}

func openFormalJarCache(parent string) (*os.Root, os.FileInfo, error) {
	before, err := os.Lstat(parent)
	if err != nil {
		return nil, nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, nil, fmt.Errorf("formal jar cache parent %s must be a real directory", parent)
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return nil, nil, err
	}
	inside, err := root.Lstat(".")
	if err != nil || !sameFormalJarCacheIdentity(before, inside) {
		return nil, nil, errors.Join(err, root.Close(), fmt.Errorf("formal jar cache parent %s changed while opening", parent))
	}
	return root, inside, nil
}

func validateFormalJarCachePath(root *os.Root, parent string, expected os.FileInfo) error {
	inside, insideErr := root.Lstat(".")
	outside, outsideErr := os.Lstat(parent)
	if err := errors.Join(insideErr, outsideErr); err != nil {
		return errors.Join(err, fmt.Errorf("formal jar cache parent %s changed identity", parent))
	}
	if !sameFormalJarCacheIdentity(expected, inside) || !sameFormalJarCacheIdentity(inside, outside) {
		return fmt.Errorf("formal jar cache parent %s changed identity", parent)
	}
	return nil
}

func createFormalJarTemp(root *os.Root, prefix string) (*os.File, string, error) {
	for range 16 {
		var nonce [16]byte
		if n, err := rand.Read(nonce[:]); err != nil || n != len(nonce) {
			return nil, "", errors.Join(err, fmt.Errorf("generate formal jar temporary name: random source returned %d of %d bytes", n, len(nonce)))
		}
		name := prefix + hex.EncodeToString(nonce[:])
		file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			return file, name, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, "", err
		}
	}
	return nil, "", fmt.Errorf("allocate formal jar temporary name: exhausted 16 attempts")
}

func validFormalJarStage(name, prefix string) bool {
	suffix := strings.TrimPrefix(name, prefix)
	if !strings.HasPrefix(name, prefix) || len(suffix) != 32 || suffix != strings.ToLower(suffix) {
		return false
	}
	_, err := hex.DecodeString(suffix)
	return err == nil
}

func fileSHA256Root(root *os.Root, name, label string) (_ string, retErr error) {
	info, err := root.Lstat(name)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > formalJarDownloadLimit {
		return "", fmt.Errorf("%s must be a regular non-symlink jar with size in 1..%d bytes", label, formalJarDownloadLimit)
	}
	file, err := root.Open(name)
	if err != nil {
		return "", err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil || !sameFormalInventoryMetadata(info, opened) {
		return "", errors.Join(err, fmt.Errorf("%s changed while opening", label))
	}
	hash := sha256.New()
	written, err := copyFormalExact(hash, file, info.Size(), label)
	if err != nil {
		return "", err
	}
	after, err := root.Lstat(name)
	if err != nil || !sameFormalInventoryMetadata(opened, after) || written != info.Size() {
		return "", errors.Join(err, fmt.Errorf("%s changed while hashing", label))
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func recoverFormalJarStages(root *os.Root, prefix string) error {
	entries, err := readFormalRootDirectory(root, "formal jar cache")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix+"delete-") {
			q, err := fsatomic.ResumeQuarantine(root, name, "")
			if err != nil {
				return err
			}
			if !validFormalJarStage(q.Source(), prefix) {
				return errors.Join(fmt.Errorf("formal jar deletion quarantine %s has invalid source %q", name, q.Source()), q.Close())
			}
			info, statErr := q.Root().Lstat(q.Name())
			if errors.Is(statErr, os.ErrNotExist) {
				if err := q.FinishEmpty(); err != nil {
					return err
				}
				continue
			}
			if statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				return errors.Join(statErr, fmt.Errorf("formal jar deletion quarantine %s has invalid inventory", name), q.Close())
			}
			if err := q.Remove(); err != nil {
				return err
			}
			continue
		}
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if !validFormalJarStage(name, prefix) {
			return fmt.Errorf("formal jar cache reserved entry %q has an invalid name", name)
		}
		info, err := root.Lstat(name)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return errors.Join(err, fmt.Errorf("formal jar cache stage %s must be a regular non-symlink file", name))
		}
		q, err := fsatomic.Quarantine(root, name, prefix+"delete-")
		if err != nil {
			return err
		}
		isolated, err := q.Root().Lstat(q.Name())
		if err != nil || !sameFormalInventoryMetadata(info, isolated) {
			return errors.Join(err, fmt.Errorf("formal jar cache stage %s changed before cleanup; preserving it", name), q.Close())
		}
		if err := q.Remove(); err != nil {
			return err
		}
	}
	return syncFormalDirectory(root)
}

// fetchJar verifies every use, including an already-cached jar. The stable
// cache-ancestor lock covers recovery, download, and installation for every
// formal engine. After acquiring it, fetchJar retains one handle-relative root
// for all cache-parent operations and rejects a replaced public parent path on
// return. The destination-derived stage prefix ensures crash recovery can
// never claim another tool's or a foreign generic temporary file.
func fetchJar(dest, url, label, wantSHA string) (_ string, retErr error) {
	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	lock, err := filelock.AcquireWait(formalJarLockScope(parent))
	if err != nil {
		return "", err
	}
	defer func() { retErr = errors.Join(retErr, lock.Release()) }()
	parentRoot, parentInfo, err := openFormalJarCache(parent)
	if err != nil {
		return "", err
	}
	defer func() {
		retErr = errors.Join(retErr, validateFormalJarCachePath(parentRoot, parent, parentInfo), parentRoot.Close())
	}()
	if formalAfterJarCacheLock != nil {
		formalAfterJarCacheLock(dest)
	}
	stagePrefix := formalJarStagePrefix(dest)
	if err := recoverFormalJarStages(parentRoot, stagePrefix); err != nil {
		return "", fmt.Errorf("recover interrupted %s download: %w", label, err)
	}
	destBase := filepath.Base(dest)
	if _, err := parentRoot.Lstat(destBase); err == nil {
		got, herr := fileSHA256Root(parentRoot, destBase, dest)
		if herr != nil {
			return "", herr
		}
		if got != wantSHA {
			return "", fmt.Errorf("checksum mismatch for cached %s: got %s, want %s; remove %s before retrying", label, got, wantSHA, dest)
		}
		return dest, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	tmpFile, tmpBase, err := createFormalJarTemp(parentRoot, stagePrefix)
	if err != nil {
		return "", err
	}
	tmpInfo, err := tmpFile.Stat()
	if err != nil {
		return "", errors.Join(err, tmpFile.Close())
	}
	tmpOpen := true
	tmpOwned := true
	defer func() {
		if tmpOpen {
			retErr = errors.Join(retErr, tmpFile.Close())
		}
		if tmpOwned {
			q, quarantineErr := fsatomic.Quarantine(parentRoot, tmpBase, stagePrefix+"delete-")
			if quarantineErr != nil {
				retErr = errors.Join(retErr, quarantineErr)
				return
			}
			isolated, statErr := q.Root().Lstat(q.Name())
			if statErr != nil || !os.SameFile(tmpInfo, isolated) {
				retErr = errors.Join(retErr, statErr, fmt.Errorf("temporary %s download changed identity before cleanup; preserving it", label), q.Close())
				return
			}
			retErr = errors.Join(retErr, q.Remove())
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { retErr = errors.Join(retErr, resp.Body.Close()) }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetching %s: HTTP %s", label, resp.Status)
	}
	if resp.ContentLength <= 0 {
		return "", fmt.Errorf("fetching %s: response Content-Length must be present and positive", label)
	}
	if resp.ContentLength > formalJarDownloadLimit {
		return "", fmt.Errorf("fetching %s: response Content-Length %d exceeds %d-byte limit", label, resp.ContentLength, formalJarDownloadLimit)
	}
	written, err := io.Copy(tmpFile, io.LimitReader(resp.Body, formalJarDownloadLimit+1))
	if err != nil {
		return "", err
	}
	if written > formalJarDownloadLimit {
		return "", fmt.Errorf("fetching %s: response body exceeds %d-byte limit", label, formalJarDownloadLimit)
	}
	if written != resp.ContentLength {
		return "", fmt.Errorf("fetching %s: response body length %d does not match Content-Length %d", label, written, resp.ContentLength)
	}
	if err := tmpFile.Sync(); err != nil {
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		tmpOpen = false
		return "", err
	}
	tmpOpen = false
	got, err := fileSHA256Root(parentRoot, tmpBase, "temporary "+label)
	if err != nil {
		return "", err
	}
	if got != wantSHA {
		return "", fmt.Errorf("checksum mismatch for %s: got %s, want %s", label, got, wantSHA)
	}
	installErr := fsatomic.RenameNoReplace(parentRoot, tmpBase, destBase)
	if installErr != nil {
		// Windows does not replace an existing destination. Another verifier
		// may have won the cold-cache race after our initial Stat; accept that
		// outcome only after rehashing the winner.
		if destSHA, hashErr := fileSHA256Root(parentRoot, destBase, dest); hashErr == nil && destSHA == wantSHA {
			return dest, nil
		}
		return "", installErr
	}
	tmpOwned = false
	if err := syncFormalCacheDirectory(parentRoot, parent); err != nil {
		return "", fmt.Errorf("sync %s cache directory after install: %w", label, err)
	}
	return dest, nil
}

var syncFormalCacheDirectory = func(root *os.Root, _ string) error {
	return syncFormalDirectory(root)
}

var formalAfterJarResolved = func(string) {}
var formalProcessTimeout = 30 * time.Minute

// newTLCMetaDir creates the private scratch directory for one TLC
// invocation. TLC names its metadata directory from the wall clock at second
// resolution (<metadir>/<yy-mm-dd-hh-mm-ss>/) and refuses to start when that
// name already exists, so two runs sharing a metadir root and a start second
// collide, and one run's cleanup deletes the other's state files. TLC also
// extracts its standard modules (Naturals.tla and friends) into the JVM's
// java.io.tmpdir, which every concurrent JVM shares, so one run's write can
// race another's parse. A fresh MkdirTemp root per invocation, used for both,
// makes every name unique whatever the clock says and keeps every byte of
// TLC scratch out of the design tree. The caller removes the root when TLC
// exits.
func newTLCMetaDir() (string, error) {
	return os.MkdirTemp("", "machinery-tlc-")
}

// tlcArgs builds the java argument list for one .tla/.cfg pair, with TLC's
// metadata routed to metaDir instead of its default <spec-dir>/states/ and
// the JVM's temp dir pointed at the same private root.
func tlcArgs(jar, metaDir, cfgPath, tlaPath string) []string {
	return []string{"-XX:+UseParallelGC", "-Djava.io.tmpdir=" + metaDir, "-cp", jar, "tlc2.TLC", "-cleanup",
		"-metadir", metaDir,
		"-config", filepath.Base(cfgPath), filepath.Base(tlaPath)}
}

// runTLC mirrors tlc.sh: java -cp jar tlc2.TLC on a .tla/.cfg pair.
func runTLC(tlaPath, cfgPath string) (output string, retErr error) {
	jar, err := ensureJar()
	if err != nil {
		return "", err
	}
	formalAfterJarResolved(jar)
	metaDir, err := newTLCMetaDir()
	if err != nil {
		return "", err
	}
	defer func() {
		retErr = errors.Join(retErr, os.RemoveAll(metaDir))
		output = redactPrivatePath(output, metaDir, "<tlc-workdir>")
		retErr = redactPrivateError(retErr, metaDir, "<tlc-workdir>")
	}()
	want, err := overrideSHA("TLA_TOOLS_JAR", "TLA_TOOLS_JAR_SHA256", tlaSHA256)
	if err != nil {
		return "", err
	}
	jar, err = snapshotVerifiedJar(jar, want, "tla2tools.jar", metaDir)
	if err != nil {
		return "", err
	}
	java, err := openFormalJava(metaDir)
	if err != nil {
		return "", err
	}
	defer func() {
		if validateErr := java.Validate(); validateErr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("revalidate %s after TLC: %w", java.Identity(), validateErr))
		}
		retErr = errors.Join(retErr, java.Close())
	}()
	dir := filepath.Dir(tlaPath)
	// TLC is exhaustive model-checking; give it a generous but bounded budget.
	ctx, cancel := context.WithTimeout(context.Background(), formalProcessTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, java.Path(), tlcArgs(jar, metaDir, cfgPath, tlaPath)...)
	cmd.Dir = dir
	cmd.Env = runtimeclosure.Environment(metaDir, metaDir, java.Path())
	output, runErr := runBoundedProcess(ctx, cmd, formalProcessTimeout)
	if runErr == nil {
		runErr = validateTLCSuccessOutput(output)
	}
	return output, runErr
}

func validateTLCSuccessOutput(output string) error {
	successLines := 0
	for lineNo, line := range strings.Split(strings.ReplaceAll(output, "\r", ""), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		switch line {
		case "No error has been found", "No error has been found.", "Model checking completed. No error has been found.":
			successLines++
			continue
		}
		lower := strings.ToLower(line)
		for _, marker := range []string{"warning", "warn:", "deprecated", "exception", "fatal", "severe", "error", "failed", "failure", "assertion", "violation"} {
			if strings.Contains(lower, marker) {
				return fmt.Errorf("TLC emitted an unexpected %s diagnostic on line %d despite exiting successfully", marker, lineNo+1)
			}
		}
	}
	if successLines != 1 {
		return fmt.Errorf("TLC success diagnostics did not contain exactly one canonical success line")
	}
	return nil
}

type generatedArtifact struct {
	body  []byte
	owner string
}

// artifactCollector stages each source in isolation, making source->artifact
// ownership injective before any committed artifact is replaced.
type artifactCollector struct {
	root   string
	files  map[string]generatedArtifact
	folded map[string]string
}

func newArtifactCollector() (*artifactCollector, error) {
	root, err := os.MkdirTemp("", "machinery-formal-stage-")
	if err != nil {
		return nil, err
	}
	return &artifactCollector{root: root, files: map[string]generatedArtifact{}, folded: map[string]string{}}, nil
}

var removeArtifactCollectorRoot = os.RemoveAll

func (c *artifactCollector) add(owner, name string, body []byte) error {
	if filepath.Base(name) != name || name == "." {
		return fmt.Errorf("formal generator %s produced unsafe artifact name %q", owner, name)
	}
	if err := portablepath.ValidateBase(name); err != nil {
		return fmt.Errorf("formal generator %s produced non-portable artifact name: %w", owner, err)
	}
	if prev, ok := c.files[name]; ok {
		return fmt.Errorf("formal artifact ownership collision: %s and %s both produce %s", prev.owner, owner, name)
	}
	fold := strings.ToLower(name)
	if prior, ok := c.folded[fold]; ok {
		prev := c.files[prior]
		return fmt.Errorf("formal artifact ownership collision: %s produces %s and %s produces %s; the names alias on case-insensitive filesystems", prev.owner, prior, owner, name)
	}
	c.files[name] = generatedArtifact{body: append([]byte(nil), body...), owner: owner}
	c.folded[fold] = name
	return nil
}

func (c *artifactCollector) run(owner string, fn func(string) ([]string, error)) error {
	dir, err := os.MkdirTemp(c.root, "source-*")
	if err != nil {
		return err
	}
	names, err := fn(dir)
	if err != nil {
		return err
	}
	sort.Strings(names)
	seen := map[string]bool{}
	for _, name := range names {
		if seen[name] {
			return fmt.Errorf("formal generator %s reported artifact %s twice", owner, name)
		}
		seen[name] = true
		body, rerr := readFormalFileExact(filepath.Join(dir, name), "generated formal artifact "+name)
		if rerr != nil {
			return fmt.Errorf("formal generator %s reported %s but it is unreadable: %w", owner, name, rerr)
		}
		if err := c.add(owner, name, body); err != nil {
			return err
		}
	}
	return nil
}

func commitGeneratedArtifacts(dir string, files map[string]generatedArtifact) (retErr error) {
	snapshot, err := designlock.Acquire(filepath.Dir(dir))
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, snapshot.Release()) }()
	if err := snapshot.CheckUnchanged(); err != nil {
		return err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	return commitGeneratedArtifactsRoot(root, files, nil, renameFormalRoot)
}

func commitGeneratedArtifactsWithRename(dir string, files map[string]generatedArtifact, rename formalRootRename) (retErr error) {
	snapshot, err := designlock.Acquire(filepath.Dir(dir))
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, snapshot.Release()) }()
	if err := snapshot.CheckUnchanged(); err != nil {
		return err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	return commitGeneratedArtifactsRoot(root, files, nil, rename)
}

func commitGeneratedArtifactsRoot(root *os.Root, files map[string]generatedArtifact, removeNames []string, rename formalRootRename) error {
	if err := recoverFormalTransaction(root, rename); err != nil {
		return fmt.Errorf("recover interrupted formal transaction: %w", err)
	}
	var names []string
	for name := range files {
		names = append(names, name)
	}
	removeSet := make(map[string]bool, len(removeNames))
	for _, name := range removeNames {
		if _, owned := files[name]; owned {
			return fmt.Errorf("generated target %s cannot also be removed", name)
		}
		if removeSet[name] {
			return fmt.Errorf("generated removal target %s appears twice", name)
		}
		removeSet[name] = true
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil
	}
	entries := make([]formalJournalEntry, 0, len(names))
	foldedTargets := make(map[string]string, len(names))
	for _, name := range names {
		foldedTargets[strings.ToLower(name)] = name
		oldIdentity, existed, oldInfo, err := formalRegularSnapshot(root, name, "generated target "+name)
		if err != nil {
			return fmt.Errorf("inspect generated target %s: %w", name, err)
		}
		oldWitness := formalAbsentIdentity
		oldMode := uint32(0)
		if existed {
			oldMode = uint32(oldInfo.Mode())
			oldWitness, err = formalNativeWitness(root, name, "generated target "+name, oldInfo)
			if err != nil {
				return fmt.Errorf("inspect native identity of generated target %s: %w", name, err)
			}
		}
		stageKind := "stage"
		newIdentity := formalAbsentIdentity
		newWitness := formalAbsentIdentity
		newMode := uint32(0)
		if removeSet[name] {
			if !existed {
				return fmt.Errorf("generated removal target %s does not exist", name)
			}
			stageKind = "stage-delete"
		} else {
			newIdentity = formalBodyIdentity(files[name].body)
		}
		entry := formalJournalEntry{
			Target: name, Stage: formalScratchName(stageKind, name),
			Backup: formalScratchName("backup", name), Existed: existed,
			OldIdentity: oldIdentity, NewIdentity: newIdentity,
			OldWitness: oldWitness, NewWitness: newWitness,
			OldMode: oldMode, NewMode: newMode,
		}
		for _, scratch := range []string{entry.Stage, entry.Backup} {
			if _, err := root.Lstat(scratch); err == nil {
				return fmt.Errorf("orphan formal transaction scratch %s exists without a journal", scratch)
			} else if !os.IsNotExist(err) {
				return err
			}
		}
		entries = append(entries, entry)
	}
	dirEntries, err := readFormalRootDirectory(root, ".")
	if err != nil {
		return err
	}
	for _, entry := range dirEntries {
		if want, ok := foldedTargets[strings.ToLower(entry.Name())]; ok && entry.Name() != want {
			return fmt.Errorf("existing formal artifact %q aliases generated target %q on case-insensitive filesystems", entry.Name(), want)
		}
	}
	cleanupUnjournaledStages := func(primary error) error {
		var cleanupErrs []error
		for i := len(entries) - 1; i >= 0; i-- {
			entry := entries[i]
			if formalEntryDeletes(entry) || !validFormalNativeWitness(entry.NewWitness) {
				continue
			}
			identity, exists, info, err := formalRegularSnapshot(root, entry.Stage, "unjournaled stage "+entry.Stage)
			if err != nil {
				cleanupErrs = append(cleanupErrs, err)
				continue
			}
			if !exists {
				continue
			}
			witness, err := formalNativeWitness(root, entry.Stage, "unjournaled stage "+entry.Stage, info)
			if err != nil || identity != entry.NewIdentity || witness != entry.NewWitness {
				cleanupErrs = append(cleanupErrs, errors.Join(err, fmt.Errorf("preserve changed unjournaled formal stage %s", entry.Stage)))
				continue
			}
			if err := requireFormalSnapshot(root, entry.Stage, identity, witness, true, info, "unjournaled stage "+entry.Stage); err != nil {
				cleanupErrs = append(cleanupErrs, err)
				continue
			}
			if err := removeFormalSnapshotExact(root, entry.Stage, identity, witness, info, "unjournaled stage "+entry.Stage); err != nil {
				cleanupErrs = append(cleanupErrs, err)
				continue
			}
			cleanupErrs = append(cleanupErrs, syncFormalDir(root))
		}
		return errors.Join(append([]error{primary}, cleanupErrs...)...)
	}
	for i := range entries {
		entry := entries[i]
		if formalEntryDeletes(entry) {
			continue
		}
		f, err := root.OpenFile(entry.Stage, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return cleanupUnjournaledStages(fmt.Errorf("create stage for %s before journal creation: %w", entry.Target, err))
		}
		_, writeErr := f.Write(files[entry.Target].body)
		syncErr := f.Sync()
		closeErr := f.Close()
		if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
			return cleanupUnjournaledStages(fmt.Errorf("write stage for %s before journal creation: %w", entry.Target, err))
		}
		if err := syncFormalDir(root); err != nil {
			return cleanupUnjournaledStages(fmt.Errorf("sync stage for %s before journal creation: %w", entry.Target, err))
		}
		if err := requireFormalIdentity(root, entry.Stage, entry.NewIdentity, "staged new image "+entry.Stage); err != nil {
			return cleanupUnjournaledStages(fmt.Errorf("verify stage for %s before journal creation: %w", entry.Target, err))
		}
		_, _, stageInfo, err := formalRegularSnapshot(root, entry.Stage, "staged new image "+entry.Stage)
		if err != nil {
			return cleanupUnjournaledStages(fmt.Errorf("snapshot stage for %s before journal creation: %w", entry.Target, err))
		}
		entries[i].NewWitness, err = formalNativeWitness(root, entry.Stage, "staged new image "+entry.Stage, stageInfo)
		if err != nil {
			return cleanupUnjournaledStages(fmt.Errorf("inspect native identity of stage for %s before journal creation: %w", entry.Target, err))
		}
		entries[i].NewMode = uint32(stageInfo.Mode())
	}
	if err := createFormalJournal(root, entries); err != nil {
		return cleanupUnjournaledStages(fmt.Errorf("create formal transaction journal: %w", err))
	}
	fail := func(context string, cause error) error {
		primary := fmt.Errorf("%s: %w", context, cause)
		if rbErr := recoverFormalTransaction(root, rename); rbErr != nil {
			return errors.Join(primary, fmt.Errorf("rollback also failed: %w", rbErr))
		}
		return primary
	}

	if err := appendFormalPhase(root, "parking"); err != nil {
		return fail("record formal parking phase", err)
	}
	for i := range entries {
		entry := &entries[i]
		if !entry.Existed {
			continue
		}
		if err := requireFormalDurableWitness(root, entry.Target, entry.OldIdentity, entry.OldWitness, entry.OldMode, "pre-transaction target "+entry.Target); err != nil {
			return fail("verify old "+entry.Target, err)
		}
		if err := rename(root, entry.Target, entry.Backup); err != nil {
			return fail("park old "+entry.Target, err)
		}
		if err := syncFormalDir(root); err != nil {
			return fail("sync parked "+entry.Target, err)
		}
		if err := requireFormalIdentity(root, entry.Target, formalAbsentIdentity, "parked source "+entry.Target); err != nil {
			return fail("verify parked source "+entry.Target, err)
		}
		identity, exists, info, err := formalRegularSnapshot(root, entry.Backup, "parked old image "+entry.Backup)
		if err != nil || !exists || identity != entry.OldIdentity || uint32(info.Mode()) != entry.OldMode {
			return fail("verify parked "+entry.Target, errors.Join(err, fmt.Errorf("parked old image does not match its durable content identity")))
		}
		witness, err := formalNativeWitness(root, entry.Backup, "parked old image "+entry.Backup, info)
		if err != nil {
			return fail("verify parked "+entry.Target, err)
		}
		if err := appendFormalWitness(root, entry.Target, "old", witness); err != nil {
			return fail("record parked witness for "+entry.Target, err)
		}
		entry.OldWitness = witness
		if err := requireFormalDurableWitness(root, entry.Backup, entry.OldIdentity, entry.OldWitness, entry.OldMode, "parked old image "+entry.Backup); err != nil {
			return fail("verify parked "+entry.Target, err)
		}
	}

	if err := appendFormalPhase(root, "installing"); err != nil {
		return fail("record formal installing phase", err)
	}
	for i := range entries {
		entry := &entries[i]
		if formalEntryDeletes(*entry) {
			continue
		}
		if err := requireFormalIdentity(root, entry.Target, formalAbsentIdentity, "install destination "+entry.Target); err != nil {
			return fail("verify install destination "+entry.Target, err)
		}
		if err := requireFormalDurableWitness(root, entry.Stage, entry.NewIdentity, entry.NewWitness, entry.NewMode, "staged install image "+entry.Stage); err != nil {
			return fail("verify install source "+entry.Target, err)
		}
		if err := rename(root, entry.Stage, entry.Target); err != nil {
			return fail("install "+entry.Target, err)
		}
		if err := syncFormalDir(root); err != nil {
			return fail("sync installed "+entry.Target, err)
		}
		if err := requireFormalIdentity(root, entry.Stage, formalAbsentIdentity, "installed stage "+entry.Stage); err != nil {
			return fail("verify consumed stage "+entry.Target, err)
		}
		identity, exists, info, err := formalRegularSnapshot(root, entry.Target, "installed new image "+entry.Target)
		if err != nil || !exists || identity != entry.NewIdentity || uint32(info.Mode()) != entry.NewMode {
			return fail("verify installed "+entry.Target, errors.Join(err, fmt.Errorf("installed new image does not match its durable content identity")))
		}
		witness, err := formalNativeWitness(root, entry.Target, "installed new image "+entry.Target, info)
		if err != nil {
			return fail("verify installed "+entry.Target, err)
		}
		if err := appendFormalWitness(root, entry.Target, "new", witness); err != nil {
			return fail("record installed witness for "+entry.Target, err)
		}
		entry.NewWitness = witness
		if err := requireFormalDurableWitness(root, entry.Target, entry.NewIdentity, entry.NewWitness, entry.NewMode, "installed new image "+entry.Target); err != nil {
			return fail("verify installed "+entry.Target, err)
		}
	}
	if err := appendFormalPhase(root, "committed"); err != nil {
		return fail("record formal committed phase", err)
	}
	if err := recoverFormalTransaction(root, rename); err != nil {
		return fmt.Errorf("finalize committed formal transaction: %w", err)
	}
	return nil
}

const manualTLAMarker = `\* machinery:manual`

func isDeclaredManualTLA(path string) (bool, error) {
	b, err := readFormalFileExact(path, "manual TLA source")
	if err != nil {
		return false, err
	}
	first, _, _ := strings.Cut(string(b), "\n")
	return strings.TrimSuffix(first, "\r") == manualTLAMarker, nil
}

func canonicalFormalGeneratedArtifact(name string, body []byte) bool {
	lines := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	switch filepath.Ext(name) {
	case ".tla":
		module := strings.TrimSuffix(name, ".tla")
		if len(lines) < 3 || lines[0] != "---- MODULE "+module+" ----" || !strings.HasPrefix(lines[1], `\* machinery-version: `) {
			return false
		}
		if strings.HasPrefix(lines[2], `\* GENERATED`) || strings.HasPrefix(lines[2], `\* Generated`) {
			return true
		}
		return len(lines) >= 5 && lines[2] == "EXTENDS Naturals" && lines[3] == "" && strings.HasPrefix(lines[4], `\* Generated from `) && strings.HasSuffix(lines[4], ` by machinery tla. Control-flow model.`)
	case ".cfg":
		if len(lines) < 2 || !strings.HasPrefix(lines[0], `\* machinery-version: `) {
			return false
		}
		specifications := 0
		for _, line := range lines[1:] {
			switch {
			case strings.HasPrefix(line, "CONSTANT ") && strings.TrimSpace(line) == line:
			case strings.HasPrefix(line, "SPECIFICATION ") && strings.TrimSpace(line) == line:
				specifications++
			case strings.HasPrefix(line, "INVARIANT ") && strings.TrimSpace(line) == line:
			case strings.HasPrefix(line, "PROPERTY ") && strings.TrimSpace(line) == line:
			default:
				return false
			}
		}
		return specifications == 1
	case ".als":
		return len(lines) >= 2 && strings.HasPrefix(lines[0], "// Code generated from ") && strings.HasSuffix(lines[0], " by machinery alloy. DO NOT EDIT.") && strings.HasPrefix(lines[1], "// machinery-version: ")
	case ".md":
		return len(lines) >= 4 && strings.HasPrefix(lines[0], "# Generated ") && lines[1] == "" && strings.Contains(lines[2], " by `machinery alloy`. DO NOT EDIT BY HAND.") && strings.HasPrefix(lines[3], "<!-- machinery-version: ") && strings.HasSuffix(lines[3], " -->")
	default:
		return false
	}
}

type designLock struct {
	lock    *filelock.Lock
	root    *os.Root
	release func() error
}

func acquireDesignLock(formalDir string) (*designLock, error) {
	root, err := os.OpenRoot(formalDir)
	if err != nil {
		return nil, err
	}
	l, err := filelock.Acquire(formalDir)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("formal verification is already active for %s: %w", formalDir, err), root.Close())
	}
	if err := recoverFormalTransaction(root, renameFormalRoot); err != nil {
		return nil, errors.Join(fmt.Errorf("recover interrupted formal transaction: %w", err), l.Release(), root.Close())
	}
	return &designLock{lock: l, root: root}, nil
}

func (l *designLock) releaseAll() error {
	if l.release != nil {
		return l.release()
	}
	return errors.Join(l.root.Close(), l.lock.Release())
}

func applyFormalRelease(exitCode int, release func() error) (int, error) {
	if err := release(); err != nil {
		return 1, err
	}
	return exitCode, nil
}

// VerifyFormal regenerates the whole formal suite for a design from source
// and, unless genOnly is set, TLC-checks every .tla/.cfg pair. Mirrors
// verify_formal.sh line-for-line in its full-mode output. genOnly exists so
// Java-free environments (the nightly regen gate, adopter CI) can assert
// freshness through the same code path that the checked run uses, instead of
// re-implementing the generator orchestration in shell.
var formalAfterDesignSourceSnapshot = func() {}

func VerifyFormal(design string, genOnly bool) (exitCode int) {
	return VerifyFormalTo(design, genOnly, os.Stdout, os.Stderr)
}

// VerifyFormalTo is VerifyFormal with explicit deterministic output sinks.
func VerifyFormalTo(design string, genOnly bool, stdoutW, stderrW io.Writer) (exitCode int) {
	fdir := filepath.Join(design, "formal")
	snapshot, err := designlock.Acquire(design)
	if err != nil {
		fmt.Fprintln(stderrW, err)
		return 1
	}
	defer func() {
		if err := snapshot.Release(); err != nil {
			fmt.Fprintln(stderrW, "formal design snapshot lock release:", snapshot.LogicalError(err))
			exitCode = 1
		}
	}()
	if err := snapshot.ResumeExpected("verify-formal", "rerun `machinery verify-formal` with the same arguments"); err != nil {
		fmt.Fprintln(stderrW, err)
		return 1
	}
	if err := requireRealDir(fdir, true, false); err != nil {
		fmt.Fprintln(stderrW, err)
		return 1
	}
	if err := snapshot.Refresh(); err != nil {
		fmt.Fprintln(stderrW, err)
		return 1
	}
	sourceDesign := snapshot.SourceRoot()
	sourceMdir := filepath.Join(sourceDesign, "machines")
	sourceFdir := filepath.Join(sourceDesign, "formal")
	formalAfterDesignSourceSnapshot()
	if err := requireRealDir(sourceMdir, false, true); err != nil {
		fmt.Fprintln(stderrW, snapshot.LogicalError(err))
		return 1
	}
	if err := rejectSymlinkedFormalInputs(sourceMdir, sourceFdir); err != nil {
		fmt.Fprintln(stderrW, snapshot.LogicalError(err))
		return 1
	}
	collector, err := newArtifactCollector()
	if err != nil {
		fmt.Fprintln(stderrW, err)
		return 1
	}
	defer func() {
		if err := removeArtifactCollectorRoot(collector.root); err != nil {
			err = redactPrivateError(err, collector.root, "<formal-stage>")
			fmt.Fprintln(stderrW, "verify-formal: remove generator staging directory:", err)
			exitCode = 1
		}
	}()

	// regenerate; a generator that cannot produce its spec is a verification
	// failure, never a silent skip (a stale committed spec must not pass as fresh)
	genFail := 0
	genErr := func(err error) {
		fmt.Fprintln(stderrW, snapshot.LogicalError(err))
		genFail++
	}
	machineSrcs, machineDiscoveryErr := globExt(sourceMdir, ".machine.json")
	semSrcs, semanticsDiscoveryErr := globExt(sourceFdir, ".semantics.yaml")
	compSrcs, compositionDiscoveryErr := globExt(sourceFdir, ".composition.yaml")
	for _, discoveryErr := range []error{machineDiscoveryErr, semanticsDiscoveryErr, compositionDiscoveryErr} {
		if discoveryErr != nil {
			genErr(fmt.Errorf("discover formal source inventory: %w", discoveryErr))
		}
	}
	controlFlowOnly := 0
	for _, mj := range machineSrcs {
		root, lerr := ir.LoadMachineJSON(mj)
		if lerr != nil {
			genErr(lerr)
			continue
		}
		if _, lerr := ir.TLAModuleName(root); lerr != nil {
			genErr(fmt.Errorf("%s: %w", filepath.Base(mj), lerr))
			continue
		}
		owner := "machine " + filepath.Base(mj)
		mid, tlaBody, cfgBody, gerr := tla.Generate(mj)
		if gerr != nil {
			genErr(gerr)
			continue
		}
		if err := collector.add(owner, mid+".tla", []byte(version.StampTLAModule(tlaBody))); err != nil {
			genErr(err)
		}
		if err := collector.add(owner, mid+".cfg", []byte(version.StampCfg(cfgBody))); err != nil {
			genErr(err)
		}
	}
	for _, sem := range semSrcs {
		m := strings.TrimSuffix(filepath.Base(sem), ".semantics.yaml")
		semData, serr := readFormalFileExact(sem, "formal semantics source")
		if serr != nil {
			genErr(serr)
			continue
		}
		semV, serr := ir.LoadYAML(semData)
		if serr != nil || semV.Kind != ir.KindObject {
			genErr(fmt.Errorf("%s: semantics file is not a mapping", filepath.Base(sem)))
			continue
		}
		if semV.AsObject().GetString("pattern") == "control-flow-only" {
			if err := refine.ValidateControlFlowOnly(filepath.Join(sourceMdir, m+".machine.json"), sem); err != nil {
				genErr(err)
			} else {
				controlFlowOnly++
			}
			continue
		}
		owner := "semantics " + filepath.Base(sem)
		if err := collector.run(owner, func(out string) ([]string, error) {
			return refine.RunWrittenInSnapshot(snapshot, filepath.Join(sourceMdir, m+".machine.json"), sem, out)
		}); err != nil {
			genErr(err)
		}
	}
	for _, comp := range compSrcs {
		data, err := readFormalFileExact(comp, "formal composition source")
		if err != nil {
			genErr(fmt.Errorf("compose_gen: %w", err))
			continue
		}
		compV, err := ir.LoadYAML(data)
		if err != nil || compV.Kind != ir.KindObject {
			genErr(fmt.Errorf("compose_gen: %s is not a composition mapping", comp))
			continue
		}
		coord := compV.AsObject().GetString("coordinator")
		if coord == "" {
			genErr(fmt.Errorf("compose_gen: %s declares no coordinator", comp))
			continue
		}
		owner := "composition " + filepath.Base(comp)
		if err := collector.run(owner, func(out string) ([]string, error) {
			return compose.RunWrittenInSnapshot(snapshot, comp, filepath.Join(sourceMdir, coord+".machine.json"), out)
		}); err != nil {
			genErr(err)
		}
	}
	packmapPresent, packmapErr := optionalPathExists(filepath.Join(sourceDesign, "packmap.yaml"))
	if packmapErr != nil {
		genErr(fmt.Errorf("discover packmap.yaml: %w", packmapErr))
	} else if packmapPresent {
		files, perr := pack.GenerateRefinement(sourceDesign)
		if perr != nil {
			genErr(perr)
		} else {
			var names []string
			for name := range files {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				if err := collector.add("packmap.yaml", name, []byte(files[name])); err != nil {
					genErr(err)
				}
			}
		}
	}

	// static relational policy layer (opt-in: present only when the design
	// carries a policy annotation)
	policyAnn := filepath.Join(sourceFdir, alloy.AnnotationName)
	havePolicy := false
	var policyCommands []alloy.Command
	policyPresent, policyErr := optionalPathExists(policyAnn)
	if policyErr != nil {
		genErr(fmt.Errorf("discover policy annotation: %w", policyErr))
	} else if policyPresent {
		havePolicy = true
		domainPath, _, perr := alloy.Paths(sourceDesign)
		if perr != nil {
			genErr(perr)
			havePolicy = false
		} else if als, oracleMD, stats, aerr := alloy.GenerateAll(domainPath, policyAnn); aerr != nil {
			genErr(aerr)
			havePolicy = false
		} else if werr := collector.add("policy annotation", alloy.OutputName, []byte(als)); werr != nil {
			genErr(werr)
			havePolicy = false
		} else if werr := collector.add("policy annotation", alloy.OracleName, []byte(oracleMD)); werr != nil {
			genErr(werr)
			havePolicy = false
		} else {
			policyCommands = stats.Commands
		}
	}

	// static relational integrity model (opt-in: present only when the design
	// carries an integrity annotation)
	integrityAnn := filepath.Join(sourceFdir, alloy.IntegrityAnnotationName)
	haveIntegrity := false
	var integrityCommands []alloy.Command
	integrityPresent, integrityErr := optionalPathExists(integrityAnn)
	if integrityErr != nil {
		genErr(fmt.Errorf("discover integrity annotation: %w", integrityErr))
	} else if integrityPresent {
		haveIntegrity = true
		domainPath, _, perr := alloy.Paths(sourceDesign)
		if perr != nil {
			genErr(perr)
			haveIntegrity = false
		} else if als, stats, aerr := alloy.GenerateIntegrity(domainPath, integrityAnn); aerr != nil {
			genErr(aerr)
			haveIntegrity = false
		} else if werr := collector.add("integrity annotation", alloy.IntegrityOutputName, []byte(als)); werr != nil {
			genErr(werr)
			haveIntegrity = false
		} else {
			integrityCommands = stats.Commands
		}
	}

	// static relational isolation model (opt-in: present only when the design
	// carries an isolation annotation)
	isolationAnn := filepath.Join(sourceFdir, alloy.IsolationAnnotationName)
	haveIsolation := false
	var isolationCommands []alloy.Command
	isolationPresent, isolationErr := optionalPathExists(isolationAnn)
	if isolationErr != nil {
		genErr(fmt.Errorf("discover isolation annotation: %w", isolationErr))
	} else if isolationPresent {
		haveIsolation = true
		domainPath, _, perr := alloy.Paths(sourceDesign)
		if perr != nil {
			genErr(perr)
			haveIsolation = false
		} else if als, oracleMD, stats, aerr := alloy.GenerateIsolation(domainPath, isolationAnn); aerr != nil {
			genErr(aerr)
			haveIsolation = false
		} else if werr := collector.add("isolation annotation", alloy.IsolationOutputName, []byte(als)); werr != nil {
			genErr(werr)
			haveIsolation = false
		} else if werr := collector.add("isolation annotation", alloy.IsolationOracleName, []byte(oracleMD)); werr != nil {
			genErr(werr)
			haveIsolation = false
		} else {
			isolationCommands = stats.Commands
		}
	}

	// Reconcile every committed TLA/CFG half against this run's ownership map
	// BEFORE replacing anything. A stale orphan must never turn a failed verify
	// into a mutating command.
	// A pair no source regenerated is accepted only through the exact manual
	// source marker on the TLA module. Unmarked pairs and all unowned halves are
	// blocking errors; only the resulting allow-list is ever handed to TLC.
	bases := map[string]bool{}
	tlaPaths, tlaDiscoveryErr := globExt(sourceFdir, ".tla")
	if tlaDiscoveryErr != nil {
		genErr(fmt.Errorf("discover TLA modules: %w", tlaDiscoveryErr))
	}
	for _, path := range tlaPaths {
		bases[strings.TrimSuffix(filepath.Base(path), ".tla")] = true
	}
	cfgPaths, cfgDiscoveryErr := globExt(sourceFdir, ".cfg")
	if cfgDiscoveryErr != nil {
		genErr(fmt.Errorf("discover TLC configurations: %w", cfgDiscoveryErr))
	}
	for _, path := range cfgPaths {
		bases[strings.TrimSuffix(filepath.Base(path), ".cfg")] = true
	}
	for name := range collector.files {
		switch {
		case strings.HasSuffix(name, ".tla"):
			bases[strings.TrimSuffix(name, ".tla")] = true
		case strings.HasSuffix(name, ".cfg"):
			bases[strings.TrimSuffix(name, ".cfg")] = true
		}
	}
	var baseNames []string
	for base := range bases {
		baseNames = append(baseNames, base)
	}
	sort.Strings(baseNames)
	generatedPairs, manualPairs := 0, 0
	var manualPairNames []string
	var runPairs []string
	var staleNames []string
	for _, base := range baseNames {
		tlaName, cfgName := base+".tla", base+".cfg"
		_, genTLA := collector.files[tlaName]
		_, genCFG := collector.files[cfgName]
		diskTLA, diskTLAErr := optionalPathExists(filepath.Join(sourceFdir, tlaName))
		diskCFG, diskCFGErr := optionalPathExists(filepath.Join(sourceFdir, cfgName))
		if diskTLAErr != nil || diskCFGErr != nil {
			genErr(errors.Join(diskTLAErr, diskCFGErr))
			continue
		}
		if genTLA || genCFG {
			if genCFG && !genTLA {
				genErr(fmt.Errorf("verify-formal: generator owns %s without %s", cfgName, tlaName))
				continue
			}
			if genTLA && genCFG {
				generatedPairs++
				runPairs = append(runPairs, base)
			}
			// A fresh generated TLA may intentionally be an imported auxiliary
			// module with no cfg (for example a pack contract module).
			continue
		}
		if !diskTLA || !diskCFG {
			if diskTLA {
				body, readErr := readFormalFileExact(filepath.Join(sourceFdir, tlaName), "formal TLA artifact "+tlaName)
				if readErr != nil {
					genErr(readErr)
				} else if canonicalFormalGeneratedArtifact(tlaName, body) {
					staleNames = append(staleNames, tlaName)
				} else {
					genErr(fmt.Errorf("verify-formal: unpaired artifact %s is not canonically Machinery-generated; refusing deletion", tlaName))
				}
			}
			if diskCFG {
				body, readErr := readFormalFileExact(filepath.Join(sourceFdir, cfgName), "formal cfg artifact "+cfgName)
				if readErr != nil {
					genErr(readErr)
				} else if canonicalFormalGeneratedArtifact(cfgName, body) {
					staleNames = append(staleNames, cfgName)
				} else {
					genErr(fmt.Errorf("verify-formal: unpaired artifact %s is not canonically Machinery-generated; refusing deletion", cfgName))
				}
			}
			continue
		}
		manual, merr := isDeclaredManualTLA(filepath.Join(sourceFdir, tlaName))
		if merr != nil {
			genErr(merr)
			continue
		}
		if !manual {
			tlaBody, tlaErr := readFormalFileExact(filepath.Join(sourceFdir, tlaName), "formal TLA artifact "+tlaName)
			cfgBody, cfgErr := readFormalFileExact(filepath.Join(sourceFdir, cfgName), "formal cfg artifact "+cfgName)
			if err := errors.Join(tlaErr, cfgErr); err != nil {
				genErr(err)
			} else if canonicalFormalGeneratedArtifact(tlaName, tlaBody) && canonicalFormalGeneratedArtifact(cfgName, cfgBody) {
				staleNames = append(staleNames, tlaName, cfgName)
			} else {
				genErr(fmt.Errorf("verify-formal: unowned pair %s/%s is neither canonically Machinery-generated nor declared manual; refusing deletion", tlaName, cfgName))
			}
			continue
		}
		cfgBody, cerr := readFormalFileExact(filepath.Join(sourceFdir, cfgName), "manual formal cfg artifact "+cfgName)
		if cerr != nil {
			genErr(cerr)
			continue
		}
		cfgFirst, _, _ := strings.Cut(string(cfgBody), "\n")
		if strings.TrimSuffix(cfgFirst, "\r") == manualTLAMarker {
			genErr(fmt.Errorf("verify-formal: %s declares %q, but the manual-source marker belongs only on the first line of %s", cfgName, manualTLAMarker, tlaName))
			continue
		}
		manualPairs++
		manualPairNames = append(manualPairNames, base)
		runPairs = append(runPairs, base)
	}
	for _, name := range []string{alloy.OutputName, alloy.OracleName, alloy.IntegrityOutputName, alloy.IsolationOutputName, alloy.IsolationOracleName} {
		if _, owned := collector.files[name]; owned {
			continue
		}
		if _, err := os.Lstat(filepath.Join(sourceFdir, name)); err == nil {
			body, readErr := readFormalFileExact(filepath.Join(sourceFdir, name), "relational formal artifact "+name)
			if readErr != nil {
				genErr(readErr)
			} else if canonicalFormalGeneratedArtifact(name, body) {
				staleNames = append(staleNames, name)
			} else {
				genErr(fmt.Errorf("verify-formal: unowned relational artifact %s is not canonically Machinery-generated; refusing deletion", name))
			}
		} else if !os.IsNotExist(err) {
			genErr(fmt.Errorf("verify-formal: inspect relational artifact %s: %w", name, err))
		}
	}
	if len(baseNames) == 0 && !havePolicy && !haveIntegrity && !haveIsolation && genFail == 0 {
		genErr(fmt.Errorf("verify-formal: nothing to verify: no generated sources, declared manual TLA pairs, or relational annotations under %s", fdir))
	}

	// Only a fully generated, closed inventory may reach the design. Stage and
	// sync every file first; on any replacement failure the prior set is restored.
	if genFail == 0 {
		staleConditions := make([]artifactset.RemovalPrecondition, 0, len(staleNames))
		sort.Strings(staleNames)
		for _, name := range staleNames {
			sourceBody, readErr := readFormalFileExact(filepath.Join(sourceFdir, name), "stale formal artifact "+name)
			if readErr != nil {
				genErr(fmt.Errorf("verify-formal: read snapshotted stale artifact %s: %w", name, readErr))
				continue
			}
			liveBody, condition, inspectErr := artifactset.InspectRemovalCandidate(fdir, name)
			if inspectErr != nil {
				genErr(fmt.Errorf("verify-formal: inspect live stale artifact %s: %w", name, inspectErr))
				continue
			}
			if !bytes.Equal(sourceBody, liveBody) {
				genErr(fmt.Errorf("verify-formal: stale artifact %s changed after the immutable snapshot; refusing deletion", name))
				continue
			}
			staleConditions = append(staleConditions, condition)
		}
		if genFail != 0 {
			goto formalPublicationDone
		}
		expected := make([]designlock.OutputExpectation, 0, len(collector.files)+len(staleNames))
		committed := make(map[string][]byte, len(collector.files))
		for name, artifact := range collector.files {
			expected = append(expected, designlock.ExpectFile(filepath.Join(fdir, name), artifact.body, 0o644))
			committed[name] = artifact.body
		}
		for _, name := range staleNames {
			expected = append(expected, designlock.ExpectAbsent(filepath.Join(fdir, name)))
		}
		if err := snapshot.PublishExpectedRooted("verify-formal", "rerun `machinery verify-formal` with the same arguments", expected, func(outputs *designlock.OutputScope) error {
			return outputs.WithRoot(fdir, func(root *os.Root) error {
				return artifactset.ReconcilePlannedRooted(fdir, root, committed, staleConditions)
			})
		}); err != nil {
			genErr(fmt.Errorf("formal artifact transaction: %w", err))
		}
	}

formalPublicationDone:

	if genOnly {
		fmt.Fprintf(stdoutW, "%d spec pair(s) regenerated from source, %d declared manual pair(s), %d control-flow-only declaration(s); TLC skipped (--gen-only)\n", generatedPairs, manualPairs, controlFlowOnly)
		if havePolicy {
			fmt.Fprintf(stdoutW, "relational policy model + authz oracle regenerated (%s, %d commands; %s); Alloy skipped (--gen-only)\n", alloy.OutputName, len(policyCommands), alloy.OracleName)
		}
		if haveIntegrity {
			fmt.Fprintf(stdoutW, "relational integrity model regenerated (%s, %d commands); Alloy skipped (--gen-only)\n", alloy.IntegrityOutputName, len(integrityCommands))
		}
		if haveIsolation {
			fmt.Fprintf(stdoutW, "relational isolation model + tenant oracle regenerated (%s, %d commands; %s); Alloy skipped (--gen-only)\n", alloy.IsolationOutputName, len(isolationCommands), alloy.IsolationOracleName)
		}
		if genFail > 0 {
			fmt.Fprintf(stderrW, "verify-formal: %d generator failure(s); the committed specs above were NOT regenerated from source\n", genFail)
			return 1
		}
		if len(runPairs) == 0 && !havePolicy && !haveIntegrity && !haveIsolation {
			if len(staleNames) > 0 {
				fmt.Fprintf(stdoutW, "verify-formal: removed %d stale generated artifact(s)\n", len(staleNames))
				return 0
			}
			fmt.Fprintln(stderrW, "verify-formal: no .tla/.cfg pairs under "+fdir+": nothing to generate is a failure, not a pass")
			return 1
		}
		return 0
	}
	verificationDir, err := materializeFormalVerificationSuite(sourceFdir, collector.files, manualPairNames)
	if err != nil {
		fmt.Fprintln(stderrW, "verify-formal: materialize immutable verification suite:", err)
		return 1
	}
	defer func() {
		if err := os.RemoveAll(verificationDir); err != nil {
			fmt.Fprintln(stderrW, "verify-formal: remove immutable verification suite:", err)
			exitCode = 1
		}
	}()

	pass, fail := 0, 0
	if controlFlowOnly > 0 {
		fmt.Fprintf(stdoutW, "  NOTE  %d control-flow-only semantics declaration(s) validated; no data-refinement pair claimed\n", controlFlowOnly)
	}
	for _, name := range runPairs {
		tlaF := filepath.Join(verificationDir, name+".tla")
		cfgF := filepath.Join(verificationDir, name+".cfg")
		out, err := runTLC(tlaF, cfgF)
		if err == nil && strings.Contains(out, "No error has been found") {
			fmt.Fprintf(stdoutW, "  PASS  %s\n", name)
			pass++
		} else {
			fmt.Fprintf(stdoutW, "  FAIL  %s\n", name)
			// an infrastructure failure (missing jar, missing java, timeout)
			// produces little or no TLC output; the error object is the only
			// diagnostic and must never be discarded
			if err != nil {
				fmt.Fprintf(stdoutW, "        error: %v\n", err)
			}
			for _, summary := range tlcSemanticFailureSummary(out) {
				fmt.Fprintf(stdoutW, "        %s\n", summary)
			}
			fail++
		}
	}
	runLayer := func(present bool, alsName, prefix string, commands []alloy.Command) {
		if !present {
			return
		}
		vs, aerr := runAlloy(filepath.Join(verificationDir, alsName), commands)
		if aerr != nil {
			fmt.Fprintln(stderrW, aerr)
			fail++
			return
		}
		for _, v := range vs {
			name := prefix + "/" + v.Command.Name
			if v.Pass {
				fmt.Fprintf(stdoutW, "  PASS  %s\n", name)
				pass++
			} else {
				fmt.Fprintf(stdoutW, "  FAIL  %s\n", name)
				if v.Detail != "" {
					fmt.Fprintf(stdoutW, "        %s\n", v.Detail)
				}
				fail++
			}
		}
	}
	runLayer(havePolicy, alloy.OutputName, "Policy", policyCommands)
	runLayer(haveIntegrity, alloy.IntegrityOutputName, "Integrity", integrityCommands)
	runLayer(haveIsolation, alloy.IsolationOutputName, "Isolation", isolationCommands)
	fmt.Fprintln(stdoutW, "")
	fmt.Fprintf(stdoutW, "%d passed, %d failed\n", pass, fail)
	if genFail > 0 {
		fmt.Fprintf(stderrW, "verify-formal: %d generator failure(s); the committed specs above were NOT regenerated from source\n", genFail)
		return 1
	}
	if pass+fail == 0 {
		if len(staleNames) > 0 {
			fmt.Fprintf(stdoutW, "verify-formal: removed %d stale generated artifact(s)\n", len(staleNames))
			return 0
		}
		fmt.Fprintln(stderrW, "verify-formal: no .tla/.cfg pairs under "+fdir+": nothing to check is a failure, not a pass")
		return 1
	}
	if fail > 0 {
		return 1
	}
	return 0
}

func tlcSemanticFailureSummary(output string) []string {
	seen := map[string]bool{}
	var summaries []string
	add := func(summary string) {
		if !seen[summary] {
			seen[summary] = true
			summaries = append(summaries, summary)
		}
	}
	for _, raw := range strings.Split(strings.ReplaceAll(output, "\r", ""), "\n") {
		line := strings.TrimSpace(raw)
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "deadlock reached"):
			add("semantic result: deadlock reached")
		case strings.Contains(lower, "temporal properties were violated"):
			add("semantic result: temporal property violated")
		case strings.Contains(lower, "the behavior up to this point is"):
			add("counterexample trace produced; dynamic engine trace suppressed")
		case strings.HasPrefix(line, "Error: Invariant ") && strings.HasSuffix(line, " is violated."):
			name := strings.TrimSuffix(strings.TrimPrefix(line, "Error: Invariant "), " is violated.")
			if portableTLCIdentifier(name) {
				add("semantic result: invariant " + name + " violated")
			}
		}
	}
	return summaries
}

func portableTLCIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' || (i > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func materializeFormalVerificationSuite(sourceDir string, generated map[string]generatedArtifact, manualPairs []string) (string, error) {
	dir, err := os.MkdirTemp("", "machinery-formal-verify-")
	if err != nil {
		return "", err
	}
	fail := func(err error) (string, error) {
		return "", errors.Join(err, os.RemoveAll(dir))
	}
	var names []string
	for name := range generated {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), generated[name].body, 0o600); err != nil {
			return fail(err)
		}
	}
	manual := append([]string(nil), manualPairs...)
	sort.Strings(manual)
	for _, base := range manual {
		for _, ext := range []string{".tla", ".cfg"} {
			name := base + ext
			body, err := readFormalFileExact(filepath.Join(sourceDir, name), "manual formal artifact "+name)
			if err != nil {
				return fail(err)
			}
			if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
				return fail(err)
			}
		}
	}
	return dir, nil
}

func globExt(dir, ext string) ([]string, error) {
	entries, err := readFormalDirectory(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ext) {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out, nil
}

func optionalPathExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func requireRealDir(path string, create, allowMissing bool) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) && allowMissing {
		return nil
	}
	if os.IsNotExist(err) && create {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return fmt.Errorf("verify-formal: required directory %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("verify-formal: %s must be a real directory, not a symlink or non-directory", path)
	}
	return nil
}

func rejectSymlinkedFormalInputs(machineDir, formalDir string) error {
	for _, scan := range []struct {
		dir      string
		suffixes []string
	}{
		{machineDir, []string{".machine.json"}},
		{formalDir, []string{".semantics.yaml", ".composition.yaml", ".relational.yaml", ".tla", ".cfg", ".als", ".oracle.md"}},
	} {
		entries, err := readFormalDirectory(scan.dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("verify-formal: read %s: %w", scan.dir, err)
		}
		for _, entry := range entries {
			matched := false
			for _, suffix := range scan.suffixes {
				if strings.HasSuffix(entry.Name(), suffix) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				return fmt.Errorf("verify-formal: inspect %s: %w", filepath.Join(scan.dir, entry.Name()), err)
			}
			if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				return fmt.Errorf("verify-formal: source or artifact %s is a symlink or non-regular file; formal inputs and outputs must be regular files inside the design", filepath.Join(scan.dir, entry.Name()))
			}
		}
	}
	return nil
}
