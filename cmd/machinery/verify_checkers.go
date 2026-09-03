package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/checker"
	"github.com/RamXX/machinery/internal/designlock"
	"github.com/RamXX/machinery/internal/processcontrol"
	machversion "github.com/RamXX/machinery/internal/version"
)

// newVerifyCheckersCmd wires `machinery verify-checkers`, the external half of
// the checker layer. It is to Gk what verify-formal is to the relational gates:
// the pure phase (Gk) trusts the committed evidence because its input_hash binds
// it to the design; this phase re-runs the engine and confirms the committed
// verdict is actually reproducible over the current design.
func newVerifyCheckersCmd() *cobra.Command {
	var registryPath string
	var checkerID string
	c := &cobra.Command{
		Use:   "verify-checkers <design-dir> [--registry <path>] [--checker <id>]",
		Short: "Re-run external checkers and confirm the committed evidence is reproducible",
		Args:  cobra.ExactArgs(1),
	}
	c.Flags().StringVar(&registryPath, "registry", checker.DefaultRegistryPath, "path to the local (git-ignored) checker registry")
	c.Flags().StringVar(&checkerID, "checker", "", "verify only the checker with this id")
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		if rc := verifyCheckersTo(args[0], registryPath, checkerID, output.stdout, output.stderr); rc != 0 {
			return commandExit(rc)
		}
		return nil
	}
	return c
}

// verifyCheckers loads the registry, then re-verifies each committed checker in
// id order. It never re-runs the Gk gate; it requires the committed projection
// to be present (Gk's job is to have blessed it) and confirms a fresh run
// reproduces the committed evidence. Returns 1 if any checker failed, else 0.
func verifyCheckers(design, registryPath, only string) (result int) {
	return verifyCheckersTo(design, registryPath, only, stdoutW, stderrW)
}

var verifyCheckersAfterRegistrySnapshot = func() {}

func verifyCheckersTo(design, registryPath, only string, stdout, stderr io.Writer) (result int) {
	if err := checkIsDir(design); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	absDesign, err := filepath.Abs(design)
	if err != nil {
		fmt.Fprintf(stderr, "verify-checkers: resolve design path: %s\n", err)
		return 1
	}
	design = absDesign
	snapshot, err := designlock.AcquireReader(design)
	if err != nil {
		fmt.Fprintf(stderr, "verify-checkers: acquire design snapshot: %s\n", err)
		return 1
	}
	var success bytes.Buffer
	var registrySnapshot *designlock.RegularFileSnapshot
	defer func() {
		var registryCleanupErr error
		if registrySnapshot != nil {
			registryCleanupErr = registrySnapshot.Close()
		}
		unchangedErr := snapshot.CheckUnchanged()
		releaseErr := snapshot.Release()
		if err := errors.Join(registryCleanupErr, unchangedErr, releaseErr); err != nil {
			fmt.Fprintf(stderr, "verify-checkers: design snapshot was not stable: %s\n", err)
			result = 1
			return
		}
		if result == 0 {
			_, _ = io.Copy(stdout, &success)
		}
	}()
	sourceDesign := snapshot.SourceRoot()
	registrySnapshot, err = snapshot.MaterializeRegularFile(registryPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stderr, "verify-checkers: no checker registry at %s; create it (git-ignored) and add each checker's run command\n", registryPath)
		} else {
			fmt.Fprintf(stderr, "verify-checkers: snapshot checker registry: %s\n", snapshot.LogicalText(err.Error()))
		}
		return 1
	}
	verifyCheckersAfterRegistrySnapshot()
	reg, err := checker.LoadRegistry(registrySnapshot.Path())
	if err != nil {
		fmt.Fprintf(stderr, "verify-checkers: %s\n", snapshot.LogicalText(err.Error()))
		return 1
	}

	manifestPaths, err := checker.ManifestPaths(sourceDesign)
	if err != nil {
		fmt.Fprintf(stderr, "verify-checkers: %s\n", snapshot.LogicalText(err.Error()))
		return 1
	}
	if len(manifestPaths) == 0 {
		fmt.Fprintf(stderr, "verify-checkers: no checkers/*.checker.yaml in %s\n", design)
		return 1
	}

	// Load every manifest first, so the run order can be by checker id (stable
	// and independent of the on-disk file names).
	var mans []*checker.Manifest
	failures := 0
	for _, mp := range manifestPaths {
		man, err := checker.LoadManifest(mp)
		if err != nil {
			fmt.Fprintf(stderr, "verify-checkers %s: ERROR: %s\n", filepath.Base(mp), snapshot.LogicalText(err.Error()))
			failures++
			continue
		}
		mans = append(mans, man)
	}
	if failures > 0 {
		fmt.Fprintf(stdout, "\n0 checker(s) verified, %d failure(s)\n", failures)
		return 1
	}
	if err := checker.ValidateManifestSet(mans); err != nil {
		fmt.Fprintf(stderr, "verify-checkers: ERROR: %s\n", err)
		return 1
	}
	sort.Slice(mans, func(i, j int) bool { return mans[i].Checker.ID < mans[j].Checker.ID })

	verified := 0
	matched := 0
	for _, man := range mans {
		if only != "" && man.Checker.ID != only {
			continue
		}
		matched++
		if verifyOneChecker(design, sourceDesign, registryPath, snapshot, man, reg, &success, stderr) {
			verified++
		} else {
			failures++
		}
	}

	if only != "" && matched == 0 {
		fmt.Fprintf(stderr, "verify-checkers: no checker with id %q in %s\n", only, design)
		return 1
	}

	if failures > 0 {
		fmt.Fprintf(stdout, "\n%d checker(s) verified, %d failure(s)\n", verified, failures)
		return 1
	}
	fmt.Fprintf(&success, "\n%d checker(s) verified\n", verified)
	return 0
}

// verifyOneChecker re-runs one checker's adapter and confirms the committed
// evidence is reproducible. It prints one result line on success and one ERROR
// on failure, returning whether the checker verified.
func verifyOneChecker(design, sourceDesign, registryPath string, snapshot *designlock.Lock, man *checker.Manifest, reg *checker.Registry, successOut, stderr io.Writer) (verified bool) {
	id := man.Checker.ID
	var work string
	fail := func(format string, a ...any) bool {
		message := fmt.Sprintf(format, a...)
		if work != "" {
			message = redactCheckerWorkPath(message, work)
		}
		message = snapshot.LogicalText(message)
		fmt.Fprintf(stderr, "verify-checkers %s: ERROR: %s\n", id, message)
		return false
	}

	entry, ok := reg.Resolve(id)
	if !ok {
		return fail("no registry entry for checker '%s'; add it to %s", id, registryPath)
	}

	// Independently regenerate the current projection. verify-checkers must not
	// trust that Gk happened earlier: two matching evidence files over the same
	// stale or wrong projection are still invalid.
	modelPaths := checker.ModelPaths(sourceDesign)
	if len(modelPaths) != 1 {
		return fail("expected exactly one *.modelith.yaml in %s, found %d", design, len(modelPaths))
	}
	model, err := checker.LoadModel(modelPaths[0])
	if err != nil {
		return fail("cannot load current domain model: %s", err)
	}
	designID, err := checker.DesignID(modelPaths[0])
	if err != nil {
		return fail("cannot hash current domain model: %s", err)
	}
	current, err := checker.Generate(model, man, designID, machversion.Version)
	if err != nil {
		return fail("cannot regenerate current projection: %s", err)
	}
	expectedHash, err := current.InputHash()
	if err != nil {
		return fail("cannot hash current projection: %s", err)
	}

	// The committed projection is the exact checker input and must itself equal
	// the current generation before an external adapter is invoked.
	projectionBytes, err := checker.ReadConfinedFile(sourceDesign, man.Evidence.ProjectionOut)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fail("projection not committed; run 'machinery project %s' first", design)
		}
		return fail("committed projection is unreadable: %s", err)
	}
	committedProjection, err := checker.ParseProjection(projectionBytes)
	if err != nil {
		return fail("committed projection is invalid: %s", err)
	}
	equal, err := checker.ContentEqual(committedProjection, current)
	if err != nil {
		return fail("cannot compare committed projection with current design: %s", err)
	}
	if !equal {
		return fail("committed projection is stale or was generated for a different manifest/design; run 'machinery project %s'", design)
	}
	if committedProjection.CheckerID != id {
		return fail("committed projection checker_id %q does not match manifest %q", committedProjection.CheckerID, id)
	}
	committed, committedBytes, err := checker.LoadEvidenceConfinedBytes(sourceDesign, man.Evidence.EvidenceIn)
	if err != nil {
		return fail("committed evidence for '%s' is unreadable: %s", id, err)
	}
	if committed.Checker.ID != id {
		return fail("committed evidence checker id %q does not match manifest %q", committed.Checker.ID, id)
	}
	if committed.RuntimeClosure != man.Checker.RuntimeClosure {
		return fail("committed evidence runtime_closure %s does not match manifest checker.runtime_closure %s", committed.RuntimeClosure, man.Checker.RuntimeClosure)
	}
	if committed.InputHash != expectedHash {
		return fail("committed evidence input_hash does not bind to the current regenerated projection")
	}

	// Keep checker scratch in the OS-private temp control plane, never beside the
	// design. A process killed before deferred cleanup may leave scratch behind,
	// but that orphan cannot enter a design, repository, sibling, or parent
	// snapshot. Docker Desktop and remote engines receive only this explicit bind.
	work, err = createCheckerWorkDir(design)
	if err != nil {
		return fail("cannot create work dir: %s", err)
	}
	defer func() {
		if err := os.RemoveAll(work); err != nil {
			fmt.Fprintf(stderr, "verify-checkers %s: ERROR: remove private checker work directory: %s\n", id, redactCheckerWorkPath(err.Error(), work))
			verified = false
		}
	}()
	projectionSnapshot := filepath.Join(work, "projection.json")
	if err := os.WriteFile(projectionSnapshot, projectionBytes, 0o600); err != nil {
		return fail("cannot snapshot committed projection: %s", err)
	}
	committedDir := filepath.Join(work, "committed")
	if err := os.Mkdir(committedDir, 0o700); err != nil {
		return fail("cannot create committed evidence snapshot directory: %s", err)
	}
	evidenceSnapshot := filepath.Join(committedDir, "evidence.json")
	if err := os.WriteFile(evidenceSnapshot, committedBytes, 0o600); err != nil {
		return fail("cannot snapshot committed evidence: %s", err)
	}
	committedTraceBytes, err := readEvidenceTrace(sourceDesign, man.Evidence.EvidenceIn, committed)
	if err != nil {
		return fail("committed trace inventory is invalid: %s", err)
	}
	if committed.TraceRef != "" {
		traceSnapshot := filepath.Join(committedDir, filepath.FromSlash(committed.TraceRef))
		if err := os.MkdirAll(filepath.Dir(traceSnapshot), 0o700); err != nil {
			return fail("cannot create committed trace snapshot directory: %s", err)
		}
		if err := os.WriteFile(traceSnapshot, committedTraceBytes, 0o600); err != nil {
			return fail("cannot snapshot committed trace: %s", err)
		}
	}
	manifestRel, err := filepath.Rel(sourceDesign, man.Path)
	if err != nil {
		return fail("cannot resolve checker manifest path: %s", err)
	}
	manifestBytes, err := checker.ReadConfinedFile(sourceDesign, filepath.ToSlash(manifestRel))
	if err != nil {
		return fail("cannot snapshot checker manifest: %s", err)
	}
	manifestSnapshot := filepath.Join(work, "manifest.checker.yaml")
	if err := os.WriteFile(manifestSnapshot, manifestBytes, 0o600); err != nil {
		return fail("cannot snapshot checker manifest: %s", err)
	}

	// {config}: the manifest's opaque config block, written as JSON so an
	// adapter reads it with no YAML parser.
	configPath := filepath.Join(work, "config.json")
	cfg := man.Config
	if cfg == nil {
		cfg = map[string]any{}
	}
	cfgBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fail("cannot serialize manifest config: %s", err)
	}
	if err := os.WriteFile(configPath, append(cfgBytes, '\n'), 0o644); err != nil {
		return fail("cannot write config: %s", err)
	}

	outPath := filepath.Join(work, "evidence.json")
	tokens := checker.Tokens{
		Projection: "/work/projection.json",
		Config:     "/work/config.json",
		Manifest:   "/work/manifest.checker.yaml",
		Out:        "/work/evidence.json",
	}
	for _, command := range append(append([]string(nil), entry.Run...), entry.Verify...) {
		if strings.Contains(command, "{design}") {
			return fail("OCI checker commands cannot use {design}; the canonical projection, manifest, and config are the complete mounted design inputs")
		}
	}
	runArgs := tokens.Substitute(entry.Run)
	var verifyArgs []string
	if len(entry.Verify) > 0 {
		vTokens := tokens
		vTokens.Out = "/work/committed/evidence.json"
		verifyArgs = vTokens.Substitute(entry.Verify)
	}
	inputDigests, err := snapshotOCIInputs(work, registryPath, entry.Runtime.Inputs, snapshot)
	if err != nil {
		return fail("cannot snapshot declared OCI checker inputs: %s", err)
	}
	runtimeClosure, err := checker.RuntimeClosureDigest(entry.Runtime.Digest, entry.Runtime.Platform, entry.Run, entry.Verify, inputDigests)
	if err != nil {
		return fail("cannot compute checker runtime closure: %s", err)
	}
	if man.Checker.RuntimeClosure != runtimeClosure {
		return fail("local registry runtime closure %s does not match manifest checker.runtime_closure %s; refusing to execute a different image, command, or checker input", runtimeClosure, man.Checker.RuntimeClosure)
	}
	snapshots, err := snapshotCheckerCommands(work, entry.Runtime.Engine)
	if err != nil {
		return fail("OCI engine executable is unsafe or cannot be snapshotted: %s", err)
	}
	engineArgs := snapshots[0]
	if err := verifyLocalOCIImage(engineArgs, entry.Runtime.Image, entry.Runtime.Digest, entry.Runtime.Platform, entry.Timeout, work); err != nil {
		return fail("OCI runtime image is not present with the exact immutable identity: %s", err)
	}
	toolBinding, err := checkerToolClosureHash(work)
	if err != nil {
		return fail("cannot bind private checker tool closure: %s", err)
	}

	out, runErr := runCheckerOCI(engineArgs, entry.Runtime.Image, entry.Runtime.Platform, runArgs, runtimeClosure, entry.Timeout, work)
	if runErr != nil {
		return fail("checker '%s' run failed: %s\n%s", id, runErr, strings.TrimSpace(out))
	}
	if out != "" {
		return fail("checker '%s' run emitted stdout/stderr despite the file-only evidence contract: %q", id, out)
	}
	if err := verifyCheckerToolClosure(work, toolBinding); err != nil {
		return fail("checker '%s' mutated its private tool closure: %s", id, err)
	}

	fresh, err := checker.LoadEvidence(outPath)
	if err != nil {
		return fail("checker '%s' produced no readable evidence at {out}: %s", id, err)
	}
	if fresh.Checker.ID != id {
		return fail("fresh evidence checker id %q does not match manifest %q", fresh.Checker.ID, id)
	}
	if fresh.InputHash != expectedHash {
		return fail("fresh evidence is not reproducible: input_hash does not bind to the current regenerated projection")
	}

	if diff := reproDiff(committed, fresh); diff != "" {
		return fail("committed evidence for '%s' is not reproducible: %s", id, diff)
	}
	freshTraceBytes, err := readEvidenceTrace(work, filepath.Base(outPath), fresh)
	if err != nil {
		return fail("fresh trace inventory is invalid: %s", err)
	}
	if !bytes.Equal(committedTraceBytes, freshTraceBytes) {
		return fail("committed evidence for '%s' is not reproducible: trace content differs (committed sha256:%x, fresh sha256:%x)", id, sha256.Sum256(committedTraceBytes), sha256.Sum256(freshTraceBytes))
	}

	// Optional replay/verify against the checker's own trace. {out} here is the
	// committed evidence path, so the checker replays what was committed.
	if len(entry.Verify) > 0 {
		vout, verr := runCheckerOCI(engineArgs, entry.Runtime.Image, entry.Runtime.Platform, verifyArgs, runtimeClosure, entry.Timeout, work)
		if verr != nil {
			return fail("replay/verify failed for '%s': %s\n%s", id, verr, strings.TrimSpace(vout))
		}
		if vout != "" {
			return fail("replay/verify for '%s' emitted stdout/stderr despite the file-only evidence contract: %q", id, vout)
		}
		if err := verifyCheckerToolClosure(work, toolBinding); err != nil {
			return fail("checker '%s' verify phase mutated its private tool closure: %s", id, err)
		}
	}

	fmt.Fprintf(successOut, "verify-checkers %s: ok (verdict=%s, reproduced)\n", id, fresh.Verdict)
	return true
}

func createCheckerWorkDir(design string) (string, error) {
	designRoot, err := filepath.EvalSymlinks(design)
	if err != nil {
		return "", fmt.Errorf("resolve design root before creating checker scratch: %w", err)
	}
	tempRoot, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		return "", fmt.Errorf("resolve OS temp root before creating checker scratch: %w", err)
	}
	rel, err := filepath.Rel(designRoot, tempRoot)
	if err != nil {
		return "", fmt.Errorf("compare design and OS temp roots: %w", err)
	}
	if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
		return "", fmt.Errorf("OS temp root %s must be outside the design root", tempRoot)
	}
	work, err := os.MkdirTemp(tempRoot, "machinery-verify-checkers-")
	if err != nil {
		return "", err
	}
	return work, nil
}

type checkerToolInventoryEntry struct {
	path string
	info os.FileInfo
}

func sameCheckerToolFile(before, after os.FileInfo) bool {
	return before != nil && after != nil && before.Mode() == after.Mode() && before.Size() == after.Size() && before.ModTime().Equal(after.ModTime()) && os.SameFile(before, after)
}

func validateCheckerToolFileSize(path string, info os.FileInfo) error {
	if info.Size() < 0 || info.Size() > checkerToolMaxFileBytes {
		return fmt.Errorf("checker tool file %s size %d exceeds per-file limit %d", path, info.Size(), checkerToolMaxFileBytes)
	}
	return nil
}

func addCheckerToolBytes(total *int64, path string, size int64) error {
	if size < 0 || *total > checkerToolMaxTotalBytes-size {
		return fmt.Errorf("checker tool inventory exceeds total limit %d at %s", checkerToolMaxTotalBytes, path)
	}
	*total += size
	return nil
}

var checkerToolClosurePoint = func(string, string) error { return nil }

func hashCheckerToolFile(root *os.Root, entry checkerToolInventoryEntry, hash io.Writer) (retErr error) {
	file, err := root.Open(entry.path)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil {
		return err
	}
	if !sameCheckerToolFile(entry.info, opened) {
		return fmt.Errorf("checker tool closure entry %s changed while opening", entry.path)
	}
	if err := checkerToolClosurePoint(entry.path, "after-open"); err != nil {
		return err
	}
	contentHash := sha256.New()
	written, err := io.Copy(io.MultiWriter(hash, contentHash), io.LimitReader(file, entry.info.Size()+1))
	if err != nil {
		return err
	}
	if written != entry.info.Size() {
		return fmt.Errorf("checker tool closure entry %s changed size while hashing", entry.path)
	}
	if err := checkerToolClosurePoint(entry.path, "after-hash"); err != nil {
		return err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	verificationHash := sha256.New()
	verified, err := io.Copy(verificationHash, io.LimitReader(file, entry.info.Size()+1))
	if err != nil {
		return err
	}
	if verified != entry.info.Size() || !bytes.Equal(contentHash.Sum(nil), verificationHash.Sum(nil)) {
		return fmt.Errorf("checker tool closure entry %s changed content while hashing", entry.path)
	}
	heldAfter, heldErr := file.Stat()
	liveAfter, liveErr := root.Lstat(entry.path)
	if err := errors.Join(heldErr, liveErr); err != nil {
		return err
	}
	if !sameCheckerToolFile(entry.info, heldAfter) || !sameCheckerToolFile(heldAfter, liveAfter) {
		return fmt.Errorf("checker tool closure entry %s changed identity or metadata while hashing", entry.path)
	}
	return nil
}

func checkerToolClosureHash(workDir string) (digest string, retErr error) {
	root, err := os.OpenRoot(workDir)
	if err != nil {
		return "", fmt.Errorf("open checker workspace root: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	var paths []string
	for _, subtree := range []string{"tool-assets", "tool-path", "tool-snapshots"} {
		walkRoot := filepath.Join(workDir, subtree)
		if err := filepath.WalkDir(walkRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(workDir, path)
			if err != nil {
				return err
			}
			paths = append(paths, rel)
			return nil
		}); err != nil {
			return "", fmt.Errorf("inventory checker tool closure: %w", err)
		}
	}
	sort.Strings(paths)
	entries := make([]checkerToolInventoryEntry, 0, len(paths))
	var total int64
	for _, rel := range paths {
		info, err := root.Lstat(rel)
		if err != nil {
			return "", fmt.Errorf("inspect checker tool closure entry %s: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return "", fmt.Errorf("checker tool closure entry %s is a symlink or special file", rel)
		}
		if info.Mode().IsRegular() {
			if err := validateCheckerToolFileSize(rel, info); err != nil {
				return "", err
			}
			if err := addCheckerToolBytes(&total, rel, info.Size()); err != nil {
				return "", err
			}
		}
		entries = append(entries, checkerToolInventoryEntry{path: rel, info: info})
	}
	hash := sha256.New()
	for _, entry := range entries {
		rel, info := entry.path, entry.info
		if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return "", fmt.Errorf("checker tool closure entry %s is a symlink or special file", rel)
		}
		_, _ = fmt.Fprintf(hash, "%d:%s:%s\n", len(rel), filepath.ToSlash(rel), info.Mode().String())
		if info.Mode().IsRegular() {
			_, _ = fmt.Fprintf(hash, "%d:", info.Size())
			if err := hashCheckerToolFile(root, entry, hash); err != nil {
				return "", fmt.Errorf("hash checker tool closure entry %s: %w", rel, err)
			}
			_, _ = hash.Write([]byte{'\n'})
		}
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

func verifyCheckerToolClosure(workDir, expected string) error {
	actual, err := checkerToolClosureHash(workDir)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("tool closure hash changed: expected %s, got %s", expected, actual)
	}
	return nil
}

func readEvidenceTrace(root, evidenceRel string, evidence *checker.Evidence) ([]byte, error) {
	evidenceDir := filepath.Dir(filepath.FromSlash(evidenceRel))
	generatedRel := filepath.Clean(filepath.Join(evidenceDir, "generated"))
	generatedRoot := filepath.Join(root, generatedRel)
	if evidence.TraceRef == "" {
		if _, err := os.Lstat(generatedRoot); os.IsNotExist(err) {
			return nil, nil
		} else if err != nil {
			return nil, fmt.Errorf("inspect undeclared generated trace inventory: %w", err)
		}
		return nil, fmt.Errorf("generated trace inventory exists but evidence.trace_ref is absent")
	}

	expectedRel := filepath.Clean(filepath.Join(evidenceDir, filepath.FromSlash(evidence.TraceRef)))
	allowedDirs := map[string]bool{}
	for dir := filepath.Dir(expectedRel); ; dir = filepath.Dir(dir) {
		allowedDirs[dir] = true
		if dir == generatedRel {
			break
		}
		if dir == "." || dir == string(os.PathSeparator) {
			return nil, fmt.Errorf("trace_ref %q escapes generated trace inventory", evidence.TraceRef)
		}
	}
	found := false
	entries := 0
	if err := filepath.WalkDir(generatedRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		entries++
		if entries > checkerTraceInventoryLimit {
			return fmt.Errorf("generated trace inventory exceeds %d-entry limit", checkerTraceInventoryLimit)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("generated trace inventory entry %s must not be a symlink", filepath.ToSlash(rel))
		}
		if entry.IsDir() {
			if !allowedDirs[rel] {
				return fmt.Errorf("generated trace inventory contains undeclared directory %s", filepath.ToSlash(rel))
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("generated trace inventory entry %s must be a regular file", filepath.ToSlash(rel))
		}
		if rel != expectedRel {
			return fmt.Errorf("generated trace inventory contains undeclared artifact %s", filepath.ToSlash(rel))
		}
		found = true
		return nil
	}); err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("trace_ref %q is absent from generated trace inventory", evidence.TraceRef)
	}
	return checker.ReadConfinedFileBounded(root, filepath.ToSlash(expectedRel), checkerTraceLimit)
}

const (
	checkerToolMaxFileBytes  int64 = 128 << 20
	checkerToolMaxTotalBytes int64 = 512 << 20

	checkerOutputLimit            = 256 * 1024
	checkerTraceLimit             = 16 * 1024 * 1024
	checkerTraceInventoryLimit    = 1024
	checkerWaitDelay              = 500 * time.Millisecond
	checkerOCIControlPlaneTimeout = 15 * time.Second
)

// snapshotCheckerCommands resolves and copies every command executable before
// any phase runs. Commands sharing the same executable spelling share one
// snapshot, so PATH or on-disk replacement between run and replay is inert.
type checkerSnapshotSource struct {
	path string
	info os.FileInfo
}

func resolveCheckerExecutable(command string) (checkerSnapshotSource, error) {
	resolved, err := exec.LookPath(command)
	if err != nil {
		return checkerSnapshotSource{}, fmt.Errorf("resolve executable %q: %w", command, err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return checkerSnapshotSource{}, fmt.Errorf("resolve executable %q absolutely: %w", command, err)
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		return checkerSnapshotSource{}, fmt.Errorf("inspect executable %s: %w", resolved, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return checkerSnapshotSource{}, fmt.Errorf("executable %s must be a regular, non-symlink file", resolved)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return checkerSnapshotSource{}, fmt.Errorf("executable %s has no execute bit", resolved)
	}
	if err := validateCheckerToolFileSize(resolved, info); err != nil {
		return checkerSnapshotSource{}, err
	}
	return checkerSnapshotSource{path: resolved, info: info}, nil
}

func preflightCheckerSnapshotSources(commands [][]string) (map[string]checkerSnapshotSource, map[string]checkerSnapshotSource, error) {
	byCommand := map[string]checkerSnapshotSource{}
	byAsset := map[string]checkerSnapshotSource{}
	var total int64
	for _, command := range commands {
		if len(command) == 0 {
			continue
		}
		if _, ok := byCommand[command[0]]; !ok {
			source, err := resolveCheckerExecutable(command[0])
			if err != nil {
				return nil, nil, err
			}
			if err := addCheckerToolBytes(&total, source.path, source.info.Size()); err != nil {
				return nil, nil, err
			}
			byCommand[command[0]] = source
		}
		for _, arg := range command[1:] {
			if !filepath.IsAbs(arg) {
				continue
			}
			absolute := filepath.Clean(arg)
			if _, ok := byAsset[absolute]; ok {
				continue
			}
			info, err := os.Lstat(absolute)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return nil, nil, fmt.Errorf("inspect absolute checker argument %s: %w", arg, err)
			}
			if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				return nil, nil, fmt.Errorf("checker path argument %s must be a regular, non-symlink file; directory capabilities are not supported", arg)
			}
			if err := validateCheckerToolFileSize(absolute, info); err != nil {
				return nil, nil, err
			}
			if err := addCheckerToolBytes(&total, absolute, info.Size()); err != nil {
				return nil, nil, err
			}
			byAsset[absolute] = checkerSnapshotSource{path: absolute, info: info}
		}
	}
	return byCommand, byAsset, nil
}

func snapshotCheckerCommands(workDir string, commands ...[]string) ([][]string, error) {
	commandSources, assetSources, err := preflightCheckerSnapshotSources(commands)
	if err != nil {
		return nil, err
	}
	binDir := filepath.Join(workDir, "tool-snapshots")
	if err := os.Mkdir(binDir, 0o700); err != nil {
		return nil, fmt.Errorf("create checker executable snapshot directory: %w", err)
	}
	assetRoot := filepath.Join(workDir, "tool-assets")
	if err := os.Mkdir(assetRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create checker asset snapshot directory: %w", err)
	}
	pathDir := filepath.Join(workDir, "tool-path")
	if err := os.Mkdir(pathDir, 0o700); err != nil {
		return nil, fmt.Errorf("create checker executable PATH directory: %w", err)
	}
	byCommand := map[string]string{}
	pathAliases := map[string]string{}
	byAsset := map[string]string{}
	assetParents := map[string]string{}
	var assetDirs []string
	out := make([][]string, len(commands))
	for i, command := range commands {
		if len(command) == 0 {
			continue
		}
		copyArgs := append([]string(nil), command...)
		if prior := byCommand[command[0]]; prior != "" {
			copyArgs[0] = prior
		} else {
			snapshot, err := snapshotCheckerExecutable(binDir, len(byCommand), commandSources[command[0]])
			if err != nil {
				return nil, err
			}
			byCommand[command[0]] = snapshot
			copyArgs[0] = snapshot
			alias := filepath.Base(command[0])
			foldedAlias := strings.ToLower(alias)
			if prior := pathAliases[foldedAlias]; prior != "" && prior != snapshot {
				return nil, fmt.Errorf("declared checker executables collide on PATH name %q", alias)
			}
			if prior := pathAliases[foldedAlias]; prior == "" {
				if err := os.Link(snapshot, filepath.Join(pathDir, alias)); err != nil {
					return nil, fmt.Errorf("publish checker executable snapshot %q on private PATH: %w", alias, err)
				}
				pathAliases[foldedAlias] = snapshot
			}
		}
		for argIndex := 1; argIndex < len(copyArgs); argIndex++ {
			arg := copyArgs[argIndex]
			if !filepath.IsAbs(arg) {
				continue
			}
			absolute := filepath.Clean(arg)
			source, ok := assetSources[absolute]
			if !ok {
				continue
			}
			if prior := byAsset[absolute]; prior != "" {
				copyArgs[argIndex] = prior
				continue
			}
			parent := filepath.Dir(absolute)
			logicalDir := assetParents[parent]
			if logicalDir == "" {
				logicalDir = filepath.ToSlash(filepath.Join("tool-assets", fmt.Sprintf("%02d", len(assetParents))))
				assetParents[parent] = logicalDir
				physicalDir := filepath.Join(workDir, filepath.FromSlash(logicalDir))
				if err := os.Mkdir(physicalDir, 0o700); err != nil {
					return nil, fmt.Errorf("create checker asset group: %w", err)
				}
				assetDirs = append(assetDirs, physicalDir)
			}
			logical := filepath.ToSlash(filepath.Join(logicalDir, filepath.Base(absolute)))
			physical := filepath.Join(workDir, filepath.FromSlash(logical))
			if err := snapshotCheckerFile(source, physical, source.info.Mode().Perm()); err != nil {
				return nil, err
			}
			byAsset[absolute] = logical
			copyArgs[argIndex] = logical
		}
		out[i] = copyArgs
	}
	for _, dir := range assetDirs {
		if err := syncCheckerSnapshotDir(dir); err != nil {
			return nil, err
		}
	}
	if err := syncCheckerSnapshotDir(assetRoot); err != nil {
		return nil, err
	}
	if err := syncCheckerSnapshotDir(binDir); err != nil {
		return nil, err
	}
	if err := syncCheckerSnapshotDir(pathDir); err != nil {
		return nil, err
	}
	return out, nil
}

var checkerSnapshotPoint = func(string, string) error { return nil }

func snapshotCheckerFile(expected checkerSnapshotSource, destinationPath string, permissions os.FileMode) (retErr error) {
	sourcePath := expected.path
	before, err := os.Lstat(sourcePath)
	if err != nil {
		return fmt.Errorf("inspect checker file %s: %w", sourcePath, err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return fmt.Errorf("checker file %s must be a regular, non-symlink file", sourcePath)
	}
	if !sameCheckerToolFile(expected.info, before) {
		return fmt.Errorf("checker file %s changed after metadata preflight", sourcePath)
	}
	if err := validateCheckerToolFileSize(sourcePath, before); err != nil {
		return err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open checker file %s: %w", sourcePath, err)
	}
	defer func() { retErr = errors.Join(retErr, source.Close()) }()
	after, err := source.Stat()
	if err != nil {
		return fmt.Errorf("reinspect checker file %s: %w", sourcePath, err)
	}
	if !after.Mode().IsRegular() || !sameCheckerToolFile(before, after) {
		return fmt.Errorf("checker file %s changed while it was being opened", sourcePath)
	}
	if err := checkerSnapshotPoint(sourcePath, "after-open"); err != nil {
		return err
	}
	destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, permissions)
	if err != nil {
		return fmt.Errorf("create checker file snapshot: %w", err)
	}
	copiedHash := sha256.New()
	copied, copyErr := io.Copy(io.MultiWriter(destination, copiedHash), io.LimitReader(source, before.Size()+1))
	_, seekErr := source.Seek(0, io.SeekStart)
	stableHash := sha256.New()
	verified, verifyErr := io.Copy(stableHash, io.LimitReader(source, before.Size()+1))
	pointErr := checkerSnapshotPoint(sourcePath, "after-copy")
	heldAfter, heldErr := source.Stat()
	liveAfter, liveErr := os.Lstat(sourcePath)
	var changedErr error
	if copied != before.Size() || verified != before.Size() {
		changedErr = fmt.Errorf("checker file %s changed size while it was being copied", sourcePath)
	} else if seekErr == nil && verifyErr == nil && !bytes.Equal(copiedHash.Sum(nil), stableHash.Sum(nil)) {
		changedErr = fmt.Errorf("checker file %s changed while it was being copied", sourcePath)
	}
	if heldErr == nil && liveErr == nil && (!sameCheckerToolFile(before, heldAfter) || !sameCheckerToolFile(heldAfter, liveAfter)) {
		changedErr = errors.Join(changedErr, fmt.Errorf("checker file %s changed identity or metadata while it was being copied", sourcePath))
	}
	syncErr := destination.Sync()
	chmodErr := destination.Chmod(permissions)
	closeErr := destination.Close()
	if err := errors.Join(copyErr, seekErr, verifyErr, pointErr, heldErr, liveErr, changedErr, syncErr, chmodErr, closeErr); err != nil {
		return fmt.Errorf("persist checker file snapshot: %w", err)
	}
	return nil
}

func snapshotCheckerExecutable(binDir string, index int, source checkerSnapshotSource) (string, error) {
	name := fmt.Sprintf("%02d-%s", index, filepath.Base(source.path))
	snapshot := filepath.Join(binDir, name)
	if err := snapshotCheckerFile(source, snapshot, source.info.Mode().Perm()); err != nil {
		return "", fmt.Errorf("snapshot executable: %w", err)
	}
	return snapshot, nil
}

type boundedCheckerOutput struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	dropped int64
	limit   int
}

func (w *boundedCheckerOutput) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	limit := w.limit
	if limit <= 0 {
		limit = checkerOutputLimit
	}
	remaining := limit - w.buf.Len()
	if remaining > len(p) {
		remaining = len(p)
	}
	if remaining > 0 {
		_, _ = w.buf.Write(p[:remaining])
	}
	w.dropped += int64(len(p) - remaining)
	return len(p), nil
}

func (w *boundedCheckerOutput) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	output := w.buf.String()
	if w.dropped > 0 {
		output += fmt.Sprintf("\n... checker output truncated; %d byte(s) omitted ...\n", w.dropped)
	}
	return output
}

// reproDiff returns "" when fresh reproduces committed, or a specific
// difference: the verdict must match, the input_hash must match, and the
// multiset of (element, verdict) coverage pairs must match.
func reproDiff(committed, fresh *checker.Evidence) string {
	if committed.EvidenceSchema != fresh.EvidenceSchema {
		return fmt.Sprintf("committed evidence_schema %q but a fresh run reports %q", committed.EvidenceSchema, fresh.EvidenceSchema)
	}
	if committed.Checker.ID != fresh.Checker.ID || committed.Checker.Version != fresh.Checker.Version {
		return fmt.Sprintf("checker identity/version differs: committed %q@%q, fresh %q@%q", committed.Checker.ID, committed.Checker.Version, fresh.Checker.ID, fresh.Checker.Version)
	}
	if committed.Verdict != fresh.Verdict {
		return fmt.Sprintf("committed verdict %q but a fresh run reports %q", committed.Verdict, fresh.Verdict)
	}
	if committed.InputHash != fresh.InputHash {
		return fmt.Sprintf("committed input_hash %s but a fresh run computed %s", committed.InputHash, fresh.InputHash)
	}
	if committed.RuntimeClosure != fresh.RuntimeClosure {
		return fmt.Sprintf("committed runtime_closure %s but a fresh run reports %s", committed.RuntimeClosure, fresh.RuntimeClosure)
	}
	if d := coverageDiff(committed.Coverage, fresh.Coverage); d != "" {
		return "coverage differs: " + d
	}
	if d := findingsDiff(committed.Findings, fresh.Findings); d != "" {
		return "findings differ: " + d
	}
	if d := canonicalRawDiff("attestation", committed.Attestation, fresh.Attestation); d != "" {
		return d
	}
	if committed.TraceRef != fresh.TraceRef {
		return fmt.Sprintf("committed trace_ref %q but a fresh run reports %q", committed.TraceRef, fresh.TraceRef)
	}
	return ""
}

// coverageDiff compares two coverage lists as multisets of every semantic row
// Order is ignored; a differing count for any pair is reported deterministically.
func coverageDiff(committed, fresh []checker.CoverageRow) string {
	cm := coverageMultiset(committed)
	fm := coverageMultiset(fresh)
	keys := map[string]bool{}
	for k := range cm {
		keys[k] = true
	}
	for k := range fm {
		keys[k] = true
	}
	var sorted []string
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, k := range sorted {
		if cm[k] != fm[k] {
			return fmt.Sprintf("%s appears %d time(s) in committed evidence but %d time(s) in the fresh run", k, cm[k], fm[k])
		}
	}
	return ""
}

func coverageMultiset(rows []checker.CoverageRow) map[string]int {
	m := make(map[string]int, len(rows))
	for _, r := range rows {
		b, _ := json.Marshal(r)
		m[string(b)]++
	}
	return m
}

func findingsDiff(a, b []checker.Finding) string {
	am, bm := map[string]int{}, map[string]int{}
	for _, rows := range []struct {
		in  []checker.Finding
		out map[string]int
	}{{a, am}, {b, bm}} {
		for _, row := range rows.in {
			enc, _ := json.Marshal(row)
			rows.out[string(enc)]++
		}
	}
	keys := map[string]bool{}
	for k := range am {
		keys[k] = true
	}
	for k := range bm {
		keys[k] = true
	}
	var sorted []string
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, k := range sorted {
		if am[k] != bm[k] {
			return fmt.Sprintf("%s appears %d time(s) in committed evidence but %d time(s) in the fresh run", k, am[k], bm[k])
		}
	}
	return ""
}

func canonicalRawDiff(field string, a, b json.RawMessage) string {
	canonical := func(raw json.RawMessage) (string, error) {
		if len(raw) == 0 {
			return "", nil
		}
		var v any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&v); err != nil {
			return "", err
		}
		var trailing any
		if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
			if err == nil {
				return "", fmt.Errorf("multiple JSON documents are not allowed")
			}
			return "", fmt.Errorf("trailing JSON data: %w", err)
		}
		enc, err := json.Marshal(v)
		return string(enc), err
	}
	ca, ea := canonical(a)
	cb, eb := canonical(b)
	if ea != nil || eb != nil {
		return fmt.Sprintf("%s is not canonical JSON (committed=%v, fresh=%v)", field, ea, eb)
	}
	if ca != cb {
		return field + " differs"
	}
	return ""
}

func snapshotOCIInputs(workDir, registryPath string, inputs []checker.OCIInput, snapshot *designlock.Lock) (digests map[string]string, retErr error) {
	root := filepath.Join(workDir, "runtime-inputs")
	if err := os.Mkdir(root, 0o700); err != nil {
		return nil, fmt.Errorf("create runtime input directory: %w", err)
	}
	absRegistry, err := filepath.Abs(registryPath)
	if err != nil {
		return nil, fmt.Errorf("resolve registry path: %w", err)
	}
	var inputSnapshots []*designlock.RegularFileSnapshot
	defer func() {
		for i := len(inputSnapshots) - 1; i >= 0; i-- {
			retErr = errors.Join(retErr, inputSnapshots[i].Close())
		}
	}()
	digests = make(map[string]string, len(inputs))
	dirs := map[string]bool{root: true}
	for _, input := range inputs {
		destination := filepath.Join(root, filepath.FromSlash(input.Mount))
		parent := filepath.Dir(destination)
		if err := os.MkdirAll(parent, 0o700); err != nil {
			return nil, fmt.Errorf("create runtime input parent: %w", err)
		}
		for dir := parent; strings.HasPrefix(dir, root) && dir != filepath.Dir(root); dir = filepath.Dir(dir) {
			dirs[dir] = true
			if dir == root {
				break
			}
		}
		liveSource := filepath.Join(filepath.Dir(absRegistry), filepath.FromSlash(input.Source))
		stableSource, err := snapshot.MaterializeRegularFile(liveSource)
		if err != nil {
			return nil, fmt.Errorf("materialize runtime input %s from registry-relative path %s: %w", input.Mount, input.Source, err)
		}
		inputSnapshots = append(inputSnapshots, stableSource)
		sourceRoot, err := os.OpenRoot(filepath.Dir(stableSource.Path()))
		if err != nil {
			return nil, fmt.Errorf("open materialized runtime input %s: %w", input.Mount, err)
		}
		copyErr := snapshotRootedCheckerFile(sourceRoot, filepath.Base(stableSource.Path()), destination, 0o600)
		closeErr := sourceRoot.Close()
		if err := errors.Join(copyErr, closeErr); err != nil {
			return nil, fmt.Errorf("snapshot runtime input %s: %w", input.Mount, err)
		}
		body, err := os.ReadFile(destination)
		if err != nil {
			return nil, fmt.Errorf("hash runtime input %s: %w", input.Mount, err)
		}
		digests[input.Mount] = fmt.Sprintf("sha256:%x", sha256.Sum256(body))
	}
	orderedDirs := make([]string, 0, len(dirs))
	for dir := range dirs {
		orderedDirs = append(orderedDirs, dir)
	}
	sort.Slice(orderedDirs, func(i, j int) bool { return len(orderedDirs[i]) > len(orderedDirs[j]) })
	for _, dir := range orderedDirs {
		if err := syncCheckerSnapshotDir(dir); err != nil {
			return nil, err
		}
	}
	return digests, nil
}

func verifyLocalOCIImage(engineArgs []string, image, digest, platform string, timeout time.Duration, workDir string) error {
	if timeout <= 0 || timeout > checkerOCIControlPlaneTimeout {
		timeout = checkerOCIControlPlaneTimeout
	}
	args := append([]string(nil), engineArgs...)
	args = append(args, "image", "inspect", "--platform", platform, "--format", "{{json .RepoDigests}}\n{{json .Os}}\n{{json .Architecture}}", image)
	out, err := runChecker(args, timeout, workDir)
	if err != nil {
		return fmt.Errorf("inspect %s with the snapshotted engine: %w: %s", image, err, strings.TrimSpace(out))
	}
	var repoDigests []string
	dec := json.NewDecoder(strings.NewReader(out))
	if err := dec.Decode(&repoDigests); err != nil {
		return fmt.Errorf("decode OCI RepoDigests: %w", err)
	}
	var imageOS, architecture string
	if err := dec.Decode(&imageOS); err != nil {
		return fmt.Errorf("decode OCI image OS: %w", err)
	}
	if err := dec.Decode(&architecture); err != nil {
		return fmt.Errorf("decode OCI image architecture: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("OCI RepoDigests response has trailing data")
	}
	actualPlatform := imageOS + "/" + architecture
	if actualPlatform != platform {
		return fmt.Errorf("OCI image platform %s does not match required platform %s", actualPlatform, platform)
	}
	sort.Strings(repoDigests)
	for _, repoDigest := range repoDigests {
		if repoDigest == image && strings.HasSuffix(repoDigest, "@"+digest) {
			return nil
		}
	}
	return fmt.Errorf("RepoDigests %v do not contain exact reference %s", repoDigests, image)
}

// runCheckerOCI executes the checker inside one immutable, network-isolated,
// read-only OCI userspace. Only the private /work bind is writable. The local
// engine is snapshotted separately as machinery's control-plane dependency;
// checker code, interpreters, modules, rules, loaders, and native libraries all
// come from the digest-addressed image.
func runCheckerOCI(engineArgs []string, image, platform string, checkerArgs []string, runtimeDigest string, timeout time.Duration, workDir string) (string, error) {
	if len(engineArgs) == 0 {
		return "", fmt.Errorf("empty OCI engine command")
	}
	mountSource := workDir
	if evaluated, err := filepath.EvalSymlinks(workDir); err == nil {
		mountSource = evaluated
	}
	if strings.Contains(mountSource, ",") {
		return "", fmt.Errorf("checker work path contains a comma and cannot be represented by the closed OCI mount contract")
	}
	mount := "type=bind,src=" + mountSource + ",dst=/work"
	inputsMount := "type=bind,src=" + filepath.Join(mountSource, "runtime-inputs") + ",dst=/checker,readonly"
	args := append([]string(nil), engineArgs...)
	args = append(args,
		"run", "--rm", "--pull=never", "--platform", platform, "--network=none", "--read-only",
		"--cap-drop=ALL", "--security-opt=no-new-privileges", "--workdir=/work",
		"--mount", mount,
		"--mount", inputsMount,
		"--tmpfs", "/tmp:rw,noexec,nosuid,size=67108864",
		"--env", "HOME=/work/home", "--env", "LANG=C", "--env", "LC_ALL=C",
		"--env", "TMPDIR=/tmp", "--env", "TZ=UTC",
		"--env", "PYTHONNOUSERSITE=1", "--env", "PYTHONSAFEPATH=1", "--env", "PYTHONPATH=",
		"--env", "MACHINERY_CHECKER_RUNTIME_CLOSURE="+runtimeDigest,
		image,
	)
	args = append(args, checkerArgs...)
	return runChecker(args, timeout, workDir)
}

// snapshotRootedCheckerFile opens the source through one directory capability
// and never reopens it by ambient absolute path. A concurrent rename or parent
// path swap can therefore be detected or remains confined to the directory
// handle that was opened before the copy began.
func snapshotRootedCheckerFile(root *os.Root, sourcePath, destinationPath string, permissions os.FileMode) (retErr error) {
	parts := strings.Split(filepath.ToSlash(sourcePath), "/")
	for i := range parts {
		componentPath := filepath.FromSlash(strings.Join(parts[:i+1], "/"))
		info, err := root.Lstat(componentPath)
		if err != nil {
			return fmt.Errorf("inspect checker file %s: %w", sourcePath, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("checker file %s must not contain symlink path components", sourcePath)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("checker file %s has a non-directory parent component", sourcePath)
		}
	}
	before, err := root.Lstat(sourcePath)
	if err != nil {
		return fmt.Errorf("inspect checker file %s: %w", sourcePath, err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return fmt.Errorf("checker file %s must be a regular, non-symlink file", sourcePath)
	}
	source, err := root.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open checker file %s: %w", sourcePath, err)
	}
	defer func() { retErr = errors.Join(retErr, source.Close()) }()
	afterOpen, err := source.Stat()
	if err != nil {
		return fmt.Errorf("reinspect checker file %s: %w", sourcePath, err)
	}
	if !afterOpen.Mode().IsRegular() || !os.SameFile(before, afterOpen) {
		return fmt.Errorf("checker file %s changed while it was being opened", sourcePath)
	}
	destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, permissions)
	if err != nil {
		return fmt.Errorf("create checker file snapshot: %w", err)
	}
	copiedHash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(destination, copiedHash), source)
	_, seekErr := source.Seek(0, io.SeekStart)
	stableHash := sha256.New()
	_, verifyErr := io.Copy(stableHash, source)
	current, currentErr := root.Lstat(sourcePath)
	var changedErr error
	if seekErr == nil && verifyErr == nil && !bytes.Equal(copiedHash.Sum(nil), stableHash.Sum(nil)) {
		changedErr = fmt.Errorf("checker file %s changed while it was being copied", sourcePath)
	}
	if currentErr == nil && !os.SameFile(before, current) {
		changedErr = errors.Join(changedErr, fmt.Errorf("checker file %s was replaced while it was being copied", sourcePath))
	}
	syncErr := destination.Sync()
	chmodErr := destination.Chmod(permissions)
	closeErr := destination.Close()
	if err := errors.Join(copyErr, seekErr, verifyErr, currentErr, changedErr, syncErr, chmodErr, closeErr); err != nil {
		return fmt.Errorf("persist checker file snapshot: %w", err)
	}
	return nil
}

// runChecker runs an external checker command under a timeout, capturing
// stdout and stderr in deterministic stream order. A nonzero exit or a timeout
// is an error; the captured output is returned in both cases for the caller to
// surface.
func runChecker(args []string, timeout time.Duration, workDir string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("empty command")
	}
	if timeout <= 0 {
		timeout = checker.DefaultCheckerTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.WaitDelay = checkerWaitDelay
	cmd.Dir = workDir
	env, err := deterministicCheckerEnv(workDir)
	if err != nil {
		return "", err
	}
	cmd.Env = env
	stdout := boundedCheckerOutput{limit: checkerOutputLimit / 2}
	stderr := boundedCheckerOutput{limit: checkerOutputLimit / 2}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = processcontrol.Run(ctx, cmd)
	output := stdout.String()
	if stderrOutput := stderr.String(); stderrOutput != "" {
		if output != "" && !strings.HasSuffix(output, "\n") {
			output += "\n"
		}
		output += stderrOutput
	}
	if ctx.Err() == context.DeadlineExceeded {
		return redactCheckerWorkPath(output, workDir), fmt.Errorf("timed out after %s; checker process tree was terminated: %w", timeout, errors.Join(context.DeadlineExceeded, err))
	}
	if errors.Is(err, exec.ErrWaitDelay) {
		return redactCheckerWorkPath(output, workDir), fmt.Errorf("checker descendant held output pipes open beyond %s: %w", checkerWaitDelay, err)
	}
	output = redactCheckerWorkPath(output, workDir)
	if err != nil {
		return output, &redactedCheckerError{message: redactCheckerWorkPath(err.Error(), workDir), cause: err}
	}
	return output, nil
}

type redactedCheckerError struct {
	message string
	cause   error
}

func (e *redactedCheckerError) Error() string { return e.message }
func (e *redactedCheckerError) Unwrap() error { return e.cause }

func redactCheckerWorkPath(value, workDir string) string {
	paths := []string{filepath.Clean(workDir), filepath.ToSlash(filepath.Clean(workDir))}
	if evaluated, err := filepath.EvalSymlinks(workDir); err == nil {
		paths = append(paths, filepath.Clean(evaluated), filepath.ToSlash(filepath.Clean(evaluated)))
	}
	sort.Slice(paths, func(i, j int) bool { return len(paths[i]) > len(paths[j]) })
	for _, path := range paths {
		if path != "" && path != "." {
			value = strings.ReplaceAll(value, path, ".")
		}
	}
	return value
}

func deterministicCheckerEnv(workDir string) ([]string, error) {
	home := filepath.Join(workDir, "home")
	tmp := filepath.Join(workDir, "tmp")
	for _, dir := range []string{home, tmp} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("prepare deterministic checker environment: %w", err)
		}
	}
	env := []string{
		"HOME=home",
		"LANG=C",
		"LC_ALL=C",
		"TEMP=tmp",
		"TMP=tmp",
		"TMPDIR=tmp",
		"TZ=UTC",
	}
	// Descendant executable capabilities are closed: only run/verify images
	// declared in the registry and snapshotted above are visible by name.
	// Windows process startup separately requires the platform variables below.
	env = append(env, "PATH=tool-path")
	// DOCKER_HOST selects the already-provisioned OCI control-plane endpoint;
	// it is not visible inside the checker container and contributes no checker
	// userspace input. TLS variables are intentionally not inherited implicitly.
	for _, key := range []string{"ComSpec", "DOCKER_HOST", "PATHEXT", "SystemRoot", "WINDIR"} {
		if value, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+value)
		}
	}
	sort.Strings(env)
	return env, nil
}
