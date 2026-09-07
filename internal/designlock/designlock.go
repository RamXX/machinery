// Package designlock serializes every read snapshot and generated-artifact
// mutation for one Machinery design root.
package designlock

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/filelock"
	"github.com/RamXX/machinery/internal/fsatomic"
	"github.com/RamXX/machinery/internal/portablepath"
)

const scopeName = ".machinery-design-snapshot"
const publishSentinel = ".machinery-design-publish.json"
const publishSentinelStage = ".machinery-design-publish.new"
const publishSentinelRetired = ".machinery-design-publish.json.retired"
const publishSentinelStageRetired = ".machinery-design-publish.new.retired"
const publishSentinelQuarantinePrefix = ".machinery-design-publish-quarantine-"
const publishSentinelStageQuarantinePrefix = ".machinery-design-publish-stage-quarantine-"
const publishRecordMaxBytes = 1 << 20
const fingerprintBufferBytes = 64 << 10
const snapshotInventoryPageEntries = 256
const snapshotInventoryMaxEntries = 100_000
const snapshotInventoryMaxDepth = 64
const snapshotRegularFileMaxBytes int64 = 1 << 30
const snapshotAggregateMaxBytes int64 = 8 << 30

// Lock is an exclusive, process-safe design snapshot lock.
type Lock struct {
	lock                  *filelock.Lock
	root                  string
	rootInfo              os.FileInfo
	sourceRoot            string
	sourceCleanup         *privateSnapshotCleanup
	retiredSourceCleanups []*privateSnapshotCleanup
	sourceAliases         []string
	inputAliases          []pathAlias
	snapshot              map[string]string
	external              map[string]externalFileState
	externalTrees         map[string]externalTreeState
}

type pathAlias struct{ from, to string }

type externalFileState struct {
	value string
	info  os.FileInfo
}

type externalTreeState struct {
	digest string
	root   os.FileInfo
}

var (
	testAfterExternalInputRead     func(string)
	testAfterFingerprintFileOpen   func(string)
	testAfterFingerprintReadChunk  func(string)
	testAfterFingerprintFileRead   func(string)
	testAfterSnapshotCopyReadChunk func(string)
	publishRenameNoReplace         = fsatomic.RenameNoReplace
)

func streamFingerprint(path string, reader io.Reader, expected int64) ([sha256.Size]byte, error) {
	if expected < 0 {
		return [sha256.Size]byte{}, fmt.Errorf("fingerprint input %s has negative size %d", path, expected)
	}
	if expected > snapshotRegularFileMaxBytes {
		return [sha256.Size]byte{}, fmt.Errorf("fingerprint input %s exceeds %d-byte snapshot limit", path, snapshotRegularFileMaxBytes)
	}
	hash := sha256.New()
	buffer := make([]byte, fingerprintBufferBytes)
	first := true
	remaining := expected
	for remaining > 0 {
		chunk := int64(len(buffer))
		if remaining < chunk {
			chunk = remaining
		}
		read, err := reader.Read(buffer[:chunk])
		if read > 0 {
			if _, writeErr := hash.Write(buffer[:read]); writeErr != nil {
				return [sha256.Size]byte{}, writeErr
			}
			if first && testAfterFingerprintReadChunk != nil {
				testAfterFingerprintReadChunk(path)
			}
			first = false
			remaining -= int64(read)
		}
		if errors.Is(err, io.EOF) {
			if remaining != 0 {
				return [sha256.Size]byte{}, fmt.Errorf("fingerprint input %s ended early with %d bytes remaining", path, remaining)
			}
			break
		}
		if err != nil {
			return [sha256.Size]byte{}, err
		}
		if read == 0 {
			return [sha256.Size]byte{}, fmt.Errorf("fingerprint input %s made no read progress", path)
		}
	}
	var overflow [1]byte
	read, err := reader.Read(overflow[:])
	if read != 0 {
		return [sha256.Size]byte{}, fmt.Errorf("fingerprint input %s exceeds witnessed size %d", path, expected)
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return [sha256.Size]byte{}, err
	}
	if err == nil {
		return [sha256.Size]byte{}, fmt.Errorf("fingerprint input %s made no progress during overflow probe", path)
	}
	var digest [sha256.Size]byte
	copy(digest[:], hash.Sum(nil))
	return digest, nil
}

func copySnapshotFile(path string, reader io.Reader, writer io.Writer, expected int64) ([sha256.Size]byte, error) {
	if expected < 0 || expected > snapshotRegularFileMaxBytes {
		return [sha256.Size]byte{}, fmt.Errorf("snapshot input %s has invalid size %d (maximum %d)", path, expected, snapshotRegularFileMaxBytes)
	}
	hash := sha256.New()
	output := io.MultiWriter(writer, hash)
	buffer := make([]byte, fingerprintBufferBytes)
	remaining := expected
	first := true
	for remaining > 0 {
		chunk := int64(len(buffer))
		if remaining < chunk {
			chunk = remaining
		}
		read, err := reader.Read(buffer[:chunk])
		if read > 0 {
			if _, writeErr := output.Write(buffer[:read]); writeErr != nil {
				return [sha256.Size]byte{}, writeErr
			}
			remaining -= int64(read)
			if first && testAfterSnapshotCopyReadChunk != nil {
				testAfterSnapshotCopyReadChunk(path)
			}
			first = false
		}
		if errors.Is(err, io.EOF) {
			return [sha256.Size]byte{}, fmt.Errorf("snapshot input %s ended early with %d bytes remaining", path, remaining)
		}
		if err != nil {
			return [sha256.Size]byte{}, err
		}
		if read == 0 {
			return [sha256.Size]byte{}, fmt.Errorf("snapshot input %s made no read progress", path)
		}
	}
	var overflow [1]byte
	read, err := reader.Read(overflow[:])
	if read != 0 {
		return [sha256.Size]byte{}, fmt.Errorf("snapshot input %s exceeds witnessed size %d", path, expected)
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return [sha256.Size]byte{}, err
	}
	if err == nil {
		return [sha256.Size]byte{}, fmt.Errorf("snapshot input %s made no progress during overflow probe", path)
	}
	var digest [sha256.Size]byte
	copy(digest[:], hash.Sum(nil))
	return digest, nil
}

// TrackExternalTree binds an implementation/source tree outside the design
// root to the same reader/writer snapshot.
func (l *Lock) TrackExternalTree(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rootInfo, err := os.Lstat(abs)
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return fmt.Errorf("tracked external tree %s must be a real directory", abs)
	}
	values, err := l.fingerprintExternalTree(abs)
	if err != nil {
		return err
	}
	if l.externalTrees == nil {
		l.externalTrees = map[string]externalTreeState{}
	}
	l.externalTrees[abs] = externalTreeState{digest: fingerprintDigest(values), root: rootInfo}
	return nil
}

func (l *Lock) fingerprintExternalTree(root string) (map[string]string, error) {
	values, err := fingerprint(root)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(root, l.root)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		prefix := filepath.ToSlash(rel)
		for key := range values {
			if key == prefix || key == prefix+"/" || strings.HasPrefix(key, prefix+"/") {
				delete(values, key)
			}
		}
	}
	return values, nil
}

// TrackExternal adds a regular file outside the design root (for example the
// project routing config) to subsequent CheckUnchanged calls.
func (l *Lock) TrackExternal(paths ...string) error {
	if l.external == nil {
		l.external = map[string]externalFileState{}
	}
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		info, err := os.Lstat(abs)
		if err != nil {
			return err
		}
		value, err := fingerprintExternal(abs)
		if err != nil {
			return err
		}
		l.external[abs] = externalFileState{value: value, info: info}
	}
	return nil
}

func fingerprintExternal(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("tracked external input %s must be a regular file, not a symlink or special entry", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	openedInfo, statErr := f.Stat()
	if statErr != nil || !sameFingerprintFile(info, openedInfo) {
		_ = f.Close()
		return "", fmt.Errorf("tracked external input %s changed identity while opening", path)
	}
	if testAfterFingerprintFileOpen != nil {
		testAfterFingerprintFileOpen(path)
	}
	digest, readErr := streamFingerprint(path, f, openedInfo.Size())
	openedAfter, retainedStatErr := f.Stat()
	closeErr := f.Close()
	if err := errors.Join(readErr, retainedStatErr, closeErr); err != nil {
		return "", err
	}
	if testAfterExternalInputRead != nil {
		testAfterExternalInputRead(path)
	}
	if testAfterFingerprintFileRead != nil {
		testAfterFingerprintFileRead(path)
	}
	after, err := os.Lstat(path)
	if err != nil || !sameFingerprintFile(info, openedAfter) || !sameFingerprintFile(info, after) {
		return "", fmt.Errorf("tracked external input %s changed identity while reading", path)
	}
	return fmt.Sprintf("file:%o:%x", info.Mode().Perm(), digest), nil
}

func sameFingerprintFile(before, after os.FileInfo) bool {
	if before == nil || after == nil || !before.Mode().IsRegular() || !after.Mode().IsRegular() ||
		!os.SameFile(before, after) || before.Mode() != after.Mode() || before.Size() != after.Size() ||
		!before.ModTime().Equal(after.ModTime()) {
		return false
	}
	beforeChange, afterChange := fingerprintFileChangeID(before), fingerprintFileChangeID(after)
	return beforeChange == "" || afterChange == "" || beforeChange == afterChange
}

func fingerprintFileChangeID(info os.FileInfo) string {
	if info == nil || info.Sys() == nil {
		return ""
	}
	value := reflect.ValueOf(info.Sys())
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return ""
	}
	for _, name := range []string{"Ctim", "Ctimespec"} {
		field := value.FieldByName(name)
		if field.IsValid() && field.Kind() == reflect.Struct {
			sec, nsec := field.FieldByName("Sec"), field.FieldByName("Nsec")
			if sec.IsValid() && nsec.IsValid() && sec.CanInt() && nsec.CanInt() {
				return fmt.Sprintf("%d:%d", sec.Int(), nsec.Int())
			}
		}
	}
	ctime, ctimeNsec := value.FieldByName("Ctime"), value.FieldByName("Ctimensec")
	if ctime.IsValid() && ctimeNsec.IsValid() && ctime.CanInt() && ctimeNsec.CanInt() {
		return fmt.Sprintf("%d:%d", ctime.Int(), ctimeNsec.Int())
	}
	return ""
}

type publishRecord struct {
	Version           int                     `json:"version"`
	Operation         string                  `json:"operation"`
	Recovery          string                  `json:"recovery"`
	InputFingerprint  string                  `json:"input_fingerprint"`
	OutputFingerprint string                  `json:"output_fingerprint,omitempty"`
	Outputs           []string                `json:"outputs"`
	Expected          []publishExpectedRecord `json:"expected,omitempty"`
}

type publishExpectedRecord struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}

// OutputExpectation binds one publication target before mutation. Body is
// copied into a digest immediately; callers may reuse their render buffer.
type OutputExpectation struct {
	path       string
	digest     string
	mode       fs.FileMode
	absent     bool
	tree       map[string]string
	treeDigest string
}

// ExpectFile declares the exact regular-file bytes and permission mode a
// successful publication must leave at path.
func ExpectFile(path string, body []byte, mode fs.FileMode) OutputExpectation {
	sum := sha256.Sum256(body)
	return OutputExpectation{path: path, digest: fmt.Sprintf("sha256:%x", sum), mode: mode.Perm()}
}

// ExpectAbsent declares that path must not exist after successful publication.
func ExpectAbsent(path string) OutputExpectation { return OutputExpectation{path: path, absent: true} }

// ExpectTree declares the complete regular-file/directory inventory below
// path. The container path itself is not mode-bound; every generated member
// directory and file is. Relative names must remain within the tree.
func ExpectTree(path string, files map[string][]byte, dirMode, fileMode fs.FileMode) (OutputExpectation, error) {
	values := map[string]string{}
	folded := map[string]string{}
	for name, body := range files {
		name = filepath.ToSlash(filepath.Clean(filepath.FromSlash(name)))
		if name == "." || strings.HasPrefix(name, "../") || filepath.IsAbs(filepath.FromSlash(name)) {
			return OutputExpectation{}, fmt.Errorf("unsafe expected tree member %q", name)
		}
		fold := strings.ToLower(name)
		if prior, ok := folded[fold]; ok {
			return OutputExpectation{}, fmt.Errorf("expected tree members %s and %s alias on case-insensitive filesystems", prior, name)
		}
		folded[fold] = name
		for parent := filepath.ToSlash(filepath.Dir(filepath.FromSlash(name))); parent != "."; parent = filepath.ToSlash(filepath.Dir(filepath.FromSlash(parent))) {
			values[parent+"/"] = fmt.Sprintf("dir:%o", dirMode.Perm())
		}
		sum := sha256.Sum256(body)
		values[name] = fmt.Sprintf("file:%o:%x", fileMode.Perm(), sum)
	}
	return OutputExpectation{path: path, tree: values}, nil
}

// Acquire waits for exclusive ownership of designRoot's canonical snapshot
// scope. A separate scope suffix prevents collision with legacy subsystem
// locks which may already use the design directory itself as their key.
func Acquire(designRoot string) (*Lock, error) {
	return acquire(designRoot, false)
}

// AcquireReader refuses to expose a design while any supported transaction
// family has crash residue. Writers use Acquire so they can run recovery;
// gates use AcquireReader and fail before reading a parked/installing set.
func AcquireReader(designRoot string) (*Lock, error) {
	return acquire(designRoot, true)
}

func acquire(designRoot string, reader bool) (*Lock, error) {
	root, err := filepath.Abs(designRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve design root: %w", err)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect design root: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return nil, fmt.Errorf("design root must be a real directory, not a symlink or special entry")
	}
	lock, err := filelock.AcquireWait(filepath.Join(root, scopeName))
	if err != nil {
		return nil, fmt.Errorf("acquire design snapshot lock for %s: %w", root, err)
	}
	if !reader {
		if err := reconcilePublishStage(root); err != nil {
			return nil, errors.Join(err, lock.Release())
		}
	}
	if reader {
		if journal, err := findInterruptedJournal(root); err != nil {
			return nil, errors.Join(err, lock.Release())
		} else if journal != "" {
			if journal == publishSentinel {
				record, readErr := readPublishRecord(root)
				if readErr != nil {
					return nil, errors.Join(fmt.Errorf("invalid interrupted Machinery publication sentinel: %w; no gate reads were performed", readErr), lock.Release())
				}
				return nil, errors.Join(fmt.Errorf("interrupted Machinery publication %q prevents a consistent design snapshot; no gate reads were performed; %s, or inspect it read-only with `machinery recover %s`, then retry `machinery check %s`", record.Operation, record.Recovery, root, root), lock.Release())
			}
			return nil, errors.Join(fmt.Errorf("interrupted Machinery publication journal %s prevents a consistent design snapshot; no gate reads were performed; rerun the exact writer command that was interrupted to recover, or inspect it read-only with `machinery recover %s`, then retry `machinery check %s`", journal, root, root), lock.Release())
		}
	}
	snapshot, err := fingerprint(root)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("snapshot design root %s: %w", root, err), lock.Release())
	}
	result := &Lock{lock: lock, root: root, rootInfo: rootInfo, snapshot: snapshot}
	if err := result.materializeDesignSource(); err != nil {
		return nil, errors.Join(err, lock.Release())
	}
	return result, nil
}

func findInterruptedJournal(root string) (string, error) {
	found, err := collectInterruptedJournalPaths(root)
	if err != nil {
		return "", err
	}
	if len(found) == 0 {
		return "", nil
	}
	return found[0], nil
}

// collectInterruptedJournalPaths walks one design root through a retained
// no-follow capability and reports every interrupted-publication journal
// location, in deterministic sorted order. It never mutates anything.
func collectInterruptedJournalPaths(root string) ([]string, error) {
	before, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	capability, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer capability.Close()
	inside, err := capability.Lstat(".")
	if err != nil || !os.SameFile(before, inside) {
		return nil, fmt.Errorf("design root changed identity while inspecting interrupted publications")
	}
	budget := snapshotBudget{maxEntries: snapshotInventoryMaxEntries, maxBytes: snapshotAggregateMaxBytes, maxDepth: snapshotInventoryMaxDepth}
	var found []string
	var walk func(string, int) error
	walk = func(dir string, depth int) error {
		if err := budget.enterDirectory("interrupted-publication directory "+filepath.ToSlash(dir), depth); err != nil {
			return err
		}
		handle, err := capability.Open(dir)
		if err != nil {
			return err
		}
		entries, readErr := readSnapshotDir(handle, "interrupted-publication directory "+filepath.ToSlash(dir), &budget)
		closeErr := handle.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			if entry.Name() == ".git" {
				continue
			}
			rel := entry.Name()
			if dir != "." {
				rel = filepath.Join(dir, entry.Name())
			}
			info, err := capability.Lstat(rel)
			if err != nil {
				return err
			}
			if err := budget.addFile(filepath.ToSlash(rel), info); err != nil {
				return err
			}
			if interruptedJournalRel(filepath.ToSlash(rel)) {
				found = append(found, filepath.ToSlash(rel))
			}
			if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
				if err := walk(rel, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(".", 0); err != nil {
		return nil, fmt.Errorf("inspect interrupted design publications: %w", err)
	}
	sort.Strings(found)
	return found, nil
}

func interruptedJournalRel(rel string) bool {
	switch rel {
	case publishSentinel,
		publishSentinelStage,
		publishSentinelRetired,
		publishSentinelStageRetired,
		".machinery-artifact-set.journal",
		"machines/.machinery-artifact-set.journal",
		"formal/.machinery-artifact-set.journal",
		"formal/.machinery-formal-transaction.jsonl",
		"packs/.machinery-pack-transaction.jsonl",
		".machinery/checker-project-transaction.json",
		".machinery/checker-project-transaction.committed.json",
		".machinery-embed-refresh-transaction.json",
		".machinery-embed-refresh-transaction.stage":
		return true
	default:
		base := filepath.Base(filepath.FromSlash(strings.TrimSuffix(filepath.ToSlash(rel), "/")))
		return publishQuarantineName(base) || embedScratchResidueRel(rel)
	}
}

// Publish durably marks the design unreadable, runs one already-planned
// publication, verifies every non-output source byte stayed unchanged during
// publication, and only then removes the marker. Any crash or detected edit
// leaves the marker so readers fail closed until the same writer is rerun.
func (l *Lock) Publish(writer, recovery string, outputs []string, fn func() error) error {
	return l.publish(writer, recovery, outputs, nil, false, func(*OutputScope) error { return fn() })
}

// PublishExpected is Publish with an exact predeclared output contract. The
// contract is persisted in the crash sentinel and verified twice after the
// callback, including immediately before the sentinel is cleared.
func (l *Lock) PublishExpected(writer, recovery string, expected []OutputExpectation, fn func() error) error {
	return l.PublishExpectedRooted(writer, recovery, expected, func(*OutputScope) error { return fn() })
}

// PublishExpectedRooted additionally gives the callback retained rooted
// capabilities for every declared output directory. Writers must use
// OutputScope.WithRoot for mutations so a parent/output path swap cannot
// redirect the transaction after publication planning.
func (l *Lock) PublishExpectedRooted(writer, recovery string, expected []OutputExpectation, fn func(*OutputScope) error) error {
	outputs := make([]string, 0, len(expected))
	for _, output := range expected {
		path := output.path
		if output.tree != nil || output.treeDigest != "" {
			path += "/**"
		}
		outputs = append(outputs, path)
	}
	return l.publish(writer, recovery, outputs, expected, false, fn)
}

// ResumeExpected completes a prior output-bound publication whose inner
// transaction already finalized and left no recovery residue. It is safe to
// call at writer entry: no sentinel is a no-op, while live inner residue is
// left for the writer's normal recovery callback.
func (l *Lock) ResumeExpected(writer, recovery string) error {
	prior, err := readPublishRecord(l.root)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if prior.Operation != writer || prior.Recovery != recovery {
		return fmt.Errorf("publication sentinel belongs to operation %q; %s", prior.Operation, prior.Recovery)
	}
	recoveryDirs := recoveryOutputDirs(prior.Outputs)
	now, err := fingerprint(l.root)
	if err != nil {
		return err
	}
	for rel := range now {
		if transactionResidueRel(rel) || artifactsetResidueInOutputDir(rel, recoveryDirs) {
			return nil
		}
	}
	priorExpected := make([]OutputExpectation, 0, len(prior.Expected))
	for _, item := range prior.Expected {
		output, err := l.expectationFromRecord(item)
		if err != nil {
			return err
		}
		priorExpected = append(priorExpected, output)
	}
	capabilities, err := l.openExternalOutputRoots(priorExpected)
	if err != nil {
		return err
	}
	validationErr := l.validateExpectedOutputs(priorExpected, capabilities)
	var closeErr error
	for _, capability := range capabilities {
		closeErr = errors.Join(closeErr, capability.root.Close())
	}
	if closeErr != nil {
		return closeErr
	}
	switch classifyResumeValidation(validationErr) {
	case resumeNeedsCallback:
		// The sentinel was installed but the callback did not reach its exact
		// output state. The writer's normal plan/callback must run.
		return nil
	case resumeOutputsComplete:
		return l.publish(writer, recovery, nil, nil, true, func(*OutputScope) error { return nil })
	default:
		return fmt.Errorf("invalid expected-publication resume decision")
	}
}

type resumeValidationDecision uint8

const (
	resumeOutputsComplete resumeValidationDecision = iota
	resumeNeedsCallback
)

func classifyResumeValidation(validationErr error) resumeValidationDecision {
	if validationErr != nil {
		return resumeNeedsCallback
	}
	return resumeOutputsComplete
}

var testAfterPublishCallback func()
var testBeforePublishCallback func()
var testBeforeFinalOutputValidation func() error
var testAfterPublishStageSync func() error
var testAfterPublishStageDirSync func() error
var testAfterPublishStageRename func() error
var testPublishCleanupPoint func(string, string) error
var testAfterExpectedOutputRead func(string)
var testBetweenExpectedTreeFingerprints func(string)
var testBetweenFingerprintPasses func(string)
var publishQuarantineRemove = func(quarantine *fsatomic.Quarantined) error { return quarantine.Remove() }

type externalOutputRoot struct {
	root      *os.Root
	info      os.FileInfo
	path      string
	relParent string
}

// OutputScope is valid only during a PublishExpectedRooted callback.
type OutputScope struct {
	lock          *Lock
	externalRoots map[string]*externalOutputRoot
}

// OpenRoot returns a retained rooted capability for an existing directory
// inside the held design. Callers use it only for pre-publication recovery
// discovery; mutations still go through OutputScope.WithRoot.
func (l *Lock) OpenRoot(dir string) (*os.Root, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(l.root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("rooted design path %s escapes held design", dir)
	}
	ancestor, err := os.OpenRoot(l.root)
	if err != nil {
		return nil, err
	}
	inside, statErr := ancestor.Lstat(".")
	if statErr != nil || !os.SameFile(l.rootInfo, inside) {
		_ = ancestor.Close()
		return nil, fmt.Errorf("design root changed identity before rooted access")
	}
	if err := ensureRootedDirectory(ancestor, rel); err != nil {
		_ = ancestor.Close()
		return nil, err
	}
	root, err := ancestor.OpenRoot(rel)
	closeErr := ancestor.Close()
	if err != nil {
		return nil, errors.Join(err, closeErr)
	}
	if closeErr != nil {
		_ = root.Close()
		return nil, closeErr
	}
	return root, nil
}

// WithRoot opens the exact declared output directory beneath its retained
// design/external ancestor, creates a missing tail without following symlinks,
// and closes the rooted capability after fn returns.
func (s *OutputScope) WithRoot(dir string, fn func(*os.Root) error) (retErr error) {
	if s == nil || s.lock == nil {
		return fmt.Errorf("output scope is not active")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(s.lock.root, abs)
	if err != nil {
		return err
	}
	var ancestor *os.Root
	closeAncestor := false
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		capability := s.externalRoots[abs]
		if capability == nil {
			return fmt.Errorf("external output directory %s has no retained capability", dir)
		}
		ancestor = capability.root
		rel = capability.relParent
	} else {
		ancestor, err = os.OpenRoot(s.lock.root)
		if err != nil {
			return err
		}
		closeAncestor = true
		inside, statErr := ancestor.Lstat(".")
		if statErr != nil || !os.SameFile(s.lock.rootInfo, inside) {
			_ = ancestor.Close()
			return fmt.Errorf("design output root changed identity before mutation")
		}
	}
	if closeAncestor {
		defer func() { retErr = errors.Join(retErr, ancestor.Close()) }()
	}
	if err := ensureRootedDirectory(ancestor, rel); err != nil {
		return fmt.Errorf("prepare rooted output directory %s: %w", dir, err)
	}
	outputRoot, err := ancestor.OpenRoot(rel)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, outputRoot.Close()) }()
	return fn(outputRoot)
}

func ensureRootedDirectory(root *os.Root, rel string) error {
	if rel == "." || rel == "" {
		return nil
	}
	current := ""
	for _, component := range strings.Split(filepath.Clean(rel), string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := root.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) {
			if err := root.Mkdir(current, 0o755); err != nil {
				return err
			}
			info, err = root.Lstat(current)
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("component %s must be a real directory", filepath.ToSlash(current))
		}
	}
	return nil
}

func (l *Lock) publish(writer, recovery string, outputs []string, expected []OutputExpectation, resume bool, fn func(*OutputScope) error) (retErr error) {
	if err := l.CheckUnchanged(); err != nil {
		return err
	}
	prior, priorErr := readPublishRecord(l.root)
	if priorErr != nil && !errors.Is(priorErr, fs.ErrNotExist) {
		return priorErr
	}
	if priorErr == nil && prior.Operation == writer && prior.Recovery == recovery && (len(expected) > 0 || resume) {
		current, _, err := l.expectedOutputRecords(expected)
		if err != nil {
			return err
		}
		present := map[string]bool{}
		for _, item := range current {
			present[item.Path] = true
		}
		for _, item := range prior.Expected {
			if present[item.Path] || (!resume && item.Value != "absent") {
				continue
			}
			output, err := l.expectationFromRecord(item)
			if err != nil {
				return err
			}
			expected = append(expected, output)
			path := output.path
			if output.tree != nil || output.treeDigest != "" {
				path += "/**"
			}
			outputs = append(outputs, path)
		}
	}
	externalRoots, err := l.openExternalOutputRoots(expected)
	if err != nil {
		return err
	}
	defer func() {
		for _, capability := range externalRoots {
			retErr = errors.Join(retErr, capability.root.Close())
		}
	}()
	identityOutputs, err := l.canonicalOutputs(outputs)
	if err != nil {
		return err
	}
	recoveryOutputs := identityOutputs
	if priorErr == nil {
		if prior.Operation == writer && prior.Recovery == recovery {
			recoveryOutputs = append(append([]string(nil), identityOutputs...), prior.Outputs...)
		}
	}
	excluded, err := l.excludedPaths(outputs)
	if err != nil {
		return err
	}
	recoveryDirs := recoveryOutputDirs(recoveryOutputs)
	before := filterFingerprint(l.snapshot, excluded, recoveryDirs)
	expectedRecords, expectedFingerprint, err := l.expectedOutputRecords(expected)
	if err != nil {
		return err
	}
	record := publishRecord{Version: 1, Operation: writer, Recovery: recovery,
		InputFingerprint: l.publicationInputFingerprint(before), OutputFingerprint: expectedFingerprint, Outputs: identityOutputs, Expected: expectedRecords}
	authority, err := l.beginPublish(record)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, authority.close()) }()
	if err := authority.revalidate("before publication callback"); err != nil {
		return err
	}
	if testBeforePublishCallback != nil {
		testBeforePublishCallback()
	}
	if err := authority.revalidate("immediately before publication callback"); err != nil {
		return err
	}
	callbackErr := fn(&OutputScope{lock: l, externalRoots: externalRoots})
	authorityErr := authority.revalidate("after publication callback")
	if err := errors.Join(callbackErr, authorityErr); err != nil {
		return err
	}
	if testAfterPublishCallback != nil {
		testAfterPublishCallback()
	}
	if err := authority.revalidate("after publication callback observation"); err != nil {
		return err
	}
	if err := l.validateExpectedOutputs(expected, externalRoots); err != nil {
		return fmt.Errorf("published output does not match its predeclared identity: %w; publication sentinel retained; %s", err, recovery)
	}
	if err := authority.revalidate("after initial output validation"); err != nil {
		return err
	}
	if err := l.checkExternalUnchanged(); err != nil {
		return err
	}
	if err := authority.revalidate("after external input validation"); err != nil {
		return err
	}
	afterAll, err := fingerprint(l.root)
	if err != nil {
		return err
	}
	var afterKeys []string
	for key := range afterAll {
		afterKeys = append(afterKeys, key)
	}
	sort.Strings(afterKeys)
	for _, key := range afterKeys {
		if transactionResidueRel(key) || artifactsetResidueInOutputDir(key, recoveryDirs) {
			return fmt.Errorf("publication callback reported success with transaction residue %s; publication sentinel retained; %s", key, recovery)
		}
	}
	after := filterFingerprint(afterAll, excluded, recoveryDirs)
	if changed := firstFingerprintChange(before, after); changed != "" {
		return fmt.Errorf("design source changed outside the snapshot lock during publication: %s; publication sentinel retained; %s", changed, recovery)
	}
	if err := authority.revalidate("after design input validation"); err != nil {
		return err
	}
	if testBeforeFinalOutputValidation != nil {
		if err := testBeforeFinalOutputValidation(); err != nil {
			return err
		}
	}
	if err := l.validateExpectedOutputs(expected, externalRoots); err != nil {
		return fmt.Errorf("published output changed before publication completion: %w; publication sentinel retained; %s", err, recovery)
	}
	if err := authority.revalidate("after final output validation"); err != nil {
		return err
	}
	if err := authority.clear(); err != nil {
		return err
	}
	l.snapshot = afterAll
	delete(l.snapshot, publishSentinel)
	return nil
}

func (l *Lock) expectedOutputRecords(expected []OutputExpectation) ([]publishExpectedRecord, string, error) {
	if len(expected) == 0 {
		return nil, "", nil
	}
	values := map[string]string{}
	folded := map[string]string{}
	for _, output := range expected {
		abs, err := filepath.Abs(output.path)
		if err != nil {
			return nil, "", err
		}
		rel, err := filepath.Rel(l.root, abs)
		if err != nil {
			return nil, "", err
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			if output.tree != nil || output.treeDigest != "" {
				return nil, "", fmt.Errorf("external expected publication output %s cannot be a subtree", output.path)
			}
			rel = "external:" + filepath.ToSlash(abs)
		}
		rel = filepath.ToSlash(rel)
		if rel == "." || strings.HasSuffix(rel, "/") {
			return nil, "", fmt.Errorf("expected publication output %s must name a file", output.path)
		}
		fold := strings.ToLower(rel)
		if prior, ok := folded[fold]; ok {
			return nil, "", fmt.Errorf("expected publication outputs %s and %s alias on case-insensitive filesystems", prior, rel)
		}
		folded[fold] = rel
		value := "absent"
		if output.tree != nil || output.treeDigest != "" {
			digest := output.treeDigest
			if digest == "" {
				digest = fingerprintDigest(output.tree)
			}
			value = "tree:" + digest
		} else if !output.absent {
			if output.digest == "" {
				return nil, "", fmt.Errorf("expected publication output %s has no content digest", rel)
			}
			value = fmt.Sprintf("file:%o:%s", output.mode.Perm(), output.digest)
		}
		values[rel] = value
	}
	records := make([]publishExpectedRecord, 0, len(values))
	for path, value := range values {
		records = append(records, publishExpectedRecord{Path: path, Value: value})
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Path < records[j].Path })
	return records, fingerprintDigest(values), nil
}

func (l *Lock) pathFromExpectedRecord(recordPath string) (string, error) {
	return expectedRecordPath(l.root, recordPath)
}

func expectedRecordPath(root, recordPath string) (string, error) {
	if strings.HasPrefix(recordPath, "external:") {
		path := strings.TrimPrefix(recordPath, "external:")
		if !filepath.IsAbs(filepath.FromSlash(path)) {
			return "", fmt.Errorf("publication sentinel has non-absolute external output %q", recordPath)
		}
		return filepath.FromSlash(path), nil
	}
	path := filepath.Clean(filepath.FromSlash(recordPath))
	if path == "." || path == ".." || filepath.IsAbs(path) || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("publication sentinel has unsafe expected output %q", recordPath)
	}
	return filepath.Join(root, path), nil
}

func (l *Lock) expectationFromRecord(record publishExpectedRecord) (OutputExpectation, error) {
	return expectationFromRecord(l.root, record)
}

func expectationFromRecord(root string, record publishExpectedRecord) (OutputExpectation, error) {
	path, err := expectedRecordPath(root, record.Path)
	if err != nil {
		return OutputExpectation{}, err
	}
	if record.Value == "absent" {
		return ExpectAbsent(path), nil
	}
	if strings.HasPrefix(record.Value, "tree:sha256:") {
		return OutputExpectation{path: path, treeDigest: strings.TrimPrefix(record.Value, "tree:")}, nil
	}
	parts := strings.Split(record.Value, ":")
	if len(parts) != 4 || parts[0] != "file" || parts[2] != "sha256" {
		return OutputExpectation{}, fmt.Errorf("publication sentinel has invalid expected output identity for %s", record.Path)
	}
	mode, err := strconv.ParseUint(parts[1], 8, 32)
	if err != nil {
		return OutputExpectation{}, fmt.Errorf("publication sentinel has invalid expected output mode for %s", record.Path)
	}
	return OutputExpectation{path: path, digest: "sha256:" + parts[3], mode: fs.FileMode(mode)}, nil
}

func (l *Lock) openExternalOutputRoots(expected []OutputExpectation) (map[string]*externalOutputRoot, error) {
	capabilities := map[string]*externalOutputRoot{}
	fail := func(err error) (map[string]*externalOutputRoot, error) {
		for _, capability := range capabilities {
			err = errors.Join(err, capability.root.Close())
		}
		return nil, err
	}
	for _, output := range expected {
		abs, err := filepath.Abs(output.path)
		if err != nil {
			return fail(err)
		}
		if inside, _ := l.insideDesign(abs); inside {
			continue
		}
		if output.tree != nil || output.treeDigest != "" {
			return fail(fmt.Errorf("external expected publication output %s cannot be a subtree", output.path))
		}
		parent := filepath.Dir(abs)
		if _, exists := capabilities[parent]; exists {
			continue
		}
		ancestor := parent
		for {
			info, statErr := os.Lstat(ancestor)
			if statErr == nil {
				if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
					return fail(fmt.Errorf("external output ancestor %s must be a real directory", ancestor))
				}
				break
			}
			if !errors.Is(statErr, fs.ErrNotExist) {
				return fail(fmt.Errorf("inspect external output ancestor %s: %w", ancestor, statErr))
			}
			next := filepath.Dir(ancestor)
			if next == ancestor {
				return fail(fmt.Errorf("external output parent %s has no existing directory ancestor", parent))
			}
			ancestor = next
		}
		resolved, err := filepath.EvalSymlinks(ancestor)
		if err != nil {
			return fail(fmt.Errorf("resolve external output ancestor %s: %w", ancestor, err))
		}
		info, err := os.Lstat(resolved)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fail(fmt.Errorf("external output ancestor %s must resolve to a real directory", ancestor))
		}
		root, err := os.OpenRoot(resolved)
		if err != nil {
			return fail(err)
		}
		opened, statErr := root.Stat(".")
		if statErr != nil || !os.SameFile(info, opened) {
			_ = root.Close()
			return fail(fmt.Errorf("external output ancestor %s changed identity while opening", ancestor))
		}
		relParent, err := filepath.Rel(ancestor, parent)
		if err != nil || relParent == ".." || strings.HasPrefix(relParent, ".."+string(filepath.Separator)) {
			_ = root.Close()
			return fail(fmt.Errorf("confine external output parent %s under %s", parent, ancestor))
		}
		if err := validateExternalOutputParent(root, relParent, false); err != nil {
			_ = root.Close()
			return fail(fmt.Errorf("validate external output parent %s: %w", parent, err))
		}
		capabilities[parent] = &externalOutputRoot{root: root, info: info, path: resolved, relParent: relParent}
	}
	return capabilities, nil
}

func validateExternalOutputParent(root *os.Root, rel string, requireComplete bool) error {
	if rel == "." || rel == "" {
		return nil
	}
	current := ""
	for _, component := range strings.Split(filepath.Clean(rel), string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := root.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) && !requireComplete {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("component %s must be a real directory", filepath.ToSlash(current))
		}
	}
	return nil
}

func (l *Lock) validateExpectedOutputs(expected []OutputExpectation, externalRoots map[string]*externalOutputRoot) (retErr error) {
	if len(expected) == 0 {
		return nil
	}
	root, err := os.OpenRoot(l.root)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	inside, err := root.Stat(".")
	if err != nil || !os.SameFile(l.rootInfo, inside) {
		return fmt.Errorf("design root changed identity while validating outputs")
	}
	ordered := append([]OutputExpectation(nil), expected...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].path < ordered[j].path })
	for _, output := range ordered {
		abs, err := filepath.Abs(output.path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(l.root, abs)
		if err != nil {
			return err
		}
		outputRoot := root
		outputRootInfo := l.rootInfo
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			capability := externalRoots[filepath.Dir(abs)]
			if capability == nil {
				return fmt.Errorf("external output %s has no retained parent capability", output.path)
			}
			parentInfo, statErr := os.Lstat(capability.path)
			if statErr != nil || !os.SameFile(capability.info, parentInfo) {
				return fmt.Errorf("external output parent %s changed identity", filepath.Dir(abs))
			}
			outputRoot = capability.root
			outputRootInfo = capability.info
			if err := validateExternalOutputParent(outputRoot, capability.relParent, true); err != nil {
				return fmt.Errorf("external output parent %s changed: %w", filepath.Dir(abs), err)
			}
			rel = filepath.Join(capability.relParent, filepath.Base(abs))
		}
		openedRoot, rootStatErr := outputRoot.Stat(".")
		if rootStatErr != nil || !os.SameFile(outputRootInfo, openedRoot) {
			return fmt.Errorf("output parent changed identity while validating %s", output.path)
		}
		info, err := outputRoot.Lstat(rel)
		if output.tree != nil || output.treeDigest != "" {
			if err != nil {
				return fmt.Errorf("inspect output tree %s: %w", filepath.ToSlash(rel), err)
			}
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return fmt.Errorf("output tree %s must be a real directory", filepath.ToSlash(rel))
			}
			got, fingerprintErr := fingerprintRoot(filepath.Join(l.root, rel), false)
			if fingerprintErr != nil {
				return fmt.Errorf("fingerprint output tree %s: %w", filepath.ToSlash(rel), fingerprintErr)
			}
			if testBetweenExpectedTreeFingerprints != nil {
				testBetweenExpectedTreeFingerprints(output.path)
			}
			confirmed, fingerprintErr := fingerprintRoot(filepath.Join(l.root, rel), false)
			if fingerprintErr != nil || firstFingerprintChange(got, confirmed) != "" {
				return fmt.Errorf("output tree %s changed while validating", filepath.ToSlash(rel))
			}
			if output.tree != nil {
				if changed := firstFingerprintChange(output.tree, got); changed != "" {
					return fmt.Errorf("output tree %s differs at %s", filepath.ToSlash(rel), changed)
				}
			} else if fingerprintDigest(got) != output.treeDigest {
				return fmt.Errorf("output tree %s has digest %s, want %s", filepath.ToSlash(rel), fingerprintDigest(got), output.treeDigest)
			}
			continue
		}
		if output.absent {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return fmt.Errorf("inspect deleted output %s: %w", filepath.ToSlash(rel), err)
			}
			return fmt.Errorf("output %s exists but was declared deleted", filepath.ToSlash(rel))
		}
		if err != nil {
			return fmt.Errorf("inspect output %s: %w", filepath.ToSlash(rel), err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("output %s must be a regular non-symlink file", filepath.ToSlash(rel))
		}
		if runtime.GOOS != "windows" && info.Mode().Perm() != output.mode.Perm() {
			return fmt.Errorf("output %s has mode %04o, want %04o", filepath.ToSlash(rel), info.Mode().Perm(), output.mode.Perm())
		}
		f, err := outputRoot.Open(rel)
		if err != nil {
			return err
		}
		opened, statErr := f.Stat()
		if statErr != nil || !sameFingerprintFile(info, opened) {
			_ = f.Close()
			return fmt.Errorf("output %s changed identity while opening", filepath.ToSlash(rel))
		}
		digest, readErr := streamFingerprint(filepath.ToSlash(rel), f, opened.Size())
		openedAfter, retainedStatErr := f.Stat()
		closeErr := f.Close()
		if err := errors.Join(readErr, retainedStatErr, closeErr); err != nil {
			return err
		}
		if testAfterExpectedOutputRead != nil {
			testAfterExpectedOutputRead(output.path)
		}
		after, statErr := outputRoot.Lstat(rel)
		if statErr != nil || !sameFingerprintFile(info, openedAfter) || !sameFingerprintFile(info, after) {
			return fmt.Errorf("output %s changed identity while reading", filepath.ToSlash(rel))
		}
		got := fmt.Sprintf("sha256:%x", digest)
		if got != output.digest {
			return fmt.Errorf("output %s has digest %s, want %s", filepath.ToSlash(rel), got, output.digest)
		}
	}
	return nil
}

func (l *Lock) publicationInputFingerprint(design map[string]string) string {
	bound := make(map[string]string, len(design)+len(l.external)+len(l.externalTrees))
	for path, value := range design {
		bound["design:"+path] = value
	}
	for path, state := range l.external {
		bound["external-file:"+filepath.ToSlash(path)] = state.value
	}
	for path, state := range l.externalTrees {
		bound["external-tree:"+filepath.ToSlash(path)] = state.digest
	}
	return fingerprintDigest(bound)
}

type publishFileState struct {
	info   os.FileInfo
	body   []byte
	mode   os.FileMode
	size   int64
	mtime  time.Time
	change string
}

type publishAuthority struct {
	root   *os.Root
	state  publishFileState
	record publishRecord
	path   string
}

func (authority *publishAuthority) close() error {
	if authority == nil || authority.root == nil {
		return nil
	}
	err := authority.root.Close()
	authority.root = nil
	return err
}

func (authority *publishAuthority) revalidate(boundary string) error {
	if authority == nil || authority.root == nil {
		return fmt.Errorf("design publication sentinel authority is not retained at %s", boundary)
	}
	if err := authority.state.revalidateAt(authority.root, authority.path); err != nil {
		return fmt.Errorf("design publication sentinel changed %s; publication obligation retained: %w", boundary, err)
	}
	return nil
}

func (authority *publishAuthority) clear() error {
	if err := removePublishFileExact(authority.root, authority.path, publishSentinelRetired, authority.state, "completed design publication sentinel", nil); err != nil {
		return fmt.Errorf("clear design publication sentinel: %w", err)
	}
	return nil
}

func capturePublishFile(root *os.Root, name string, maxBytes int64) (publishFileState, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return publishFileState{}, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return publishFileState{}, fmt.Errorf("%s must be a private regular file", name)
	}
	if runtime.GOOS != "windows" && before.Mode().Perm()&0o077 != 0 {
		return publishFileState{}, fmt.Errorf("%s must have mode 0600", name)
	}
	if before.Size() < 0 || before.Size() > maxBytes {
		return publishFileState{}, fmt.Errorf("%s exceeds %d bytes", name, maxBytes)
	}
	file, err := root.Open(name)
	if err != nil {
		return publishFileState{}, err
	}
	opened, statErr := file.Stat()
	body, readErr := io.ReadAll(io.LimitReader(file, maxBytes+1))
	openedAfter, afterStatErr := file.Stat()
	closeErr := file.Close()
	after, pathErr := root.Lstat(name)
	if err := errors.Join(statErr, readErr, afterStatErr, closeErr, pathErr); err != nil {
		return publishFileState{}, err
	}
	if int64(len(body)) > maxBytes {
		return publishFileState{}, fmt.Errorf("%s exceeds %d bytes", name, maxBytes)
	}
	if !samePublishFileInfo(before, opened) || !samePublishFileInfo(opened, openedAfter) || !samePublishFileInfo(openedAfter, after) {
		return publishFileState{}, fmt.Errorf("%s changed identity or metadata while being witnessed", name)
	}
	return publishFileState{
		info:   after,
		body:   bytes.Clone(body),
		mode:   after.Mode(),
		size:   after.Size(),
		mtime:  after.ModTime(),
		change: fingerprintFileChangeID(after),
	}, nil
}

func samePublishFileInfo(before, after os.FileInfo) bool {
	return before != nil && after != nil && before.Mode().IsRegular() && after.Mode().IsRegular() &&
		os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size() &&
		before.ModTime().Equal(after.ModTime()) && fingerprintFileChangeID(before) == fingerprintFileChangeID(after)
}

func (state publishFileState) revalidateAt(root *os.Root, name string) error {
	current, err := capturePublishFile(root, name, publishRecordMaxBytes+1)
	if err != nil {
		return err
	}
	if !os.SameFile(state.info, current.info) || state.mode != current.mode || state.size != current.size ||
		!state.mtime.Equal(current.mtime) || state.change != current.change || !bytes.Equal(state.body, current.body) {
		return fmt.Errorf("%s changed identity, body, mode, size, mtime, or native change witness", name)
	}
	return nil
}

func samePublishFileAcrossRename(before, after publishFileState) bool {
	return os.SameFile(before.info, after.info) && before.mode == after.mode && before.size == after.size &&
		before.mtime.Equal(after.mtime) && bytes.Equal(before.body, after.body)
}

func removePublishFileExact(root *os.Root, name, retiredName string, state publishFileState, label string, guard func() error) error {
	if guard != nil {
		if err := guard(); err != nil {
			return fmt.Errorf("%s cleanup authority changed; preserving it: %w", label, err)
		}
	}
	if testPublishCleanupPoint != nil {
		if err := testPublishCleanupPoint("before-isolate", name); err != nil {
			return err
		}
	}
	if err := state.revalidateAt(root, name); err != nil {
		return fmt.Errorf("%s changed before isolation; preserving it: %w", label, err)
	}
	if info, err := root.Lstat(retiredName); err == nil {
		return fmt.Errorf("%s retirement %s already exists (%s); preserving both", label, retiredName, info.Mode())
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if testPublishCleanupPoint != nil {
		if err := testPublishCleanupPoint("before-retire", retiredName); err != nil {
			return err
		}
	}
	if info, err := root.Lstat(retiredName); err == nil {
		return fmt.Errorf("%s retirement %s appeared concurrently (%s); preserving it and the source", label, retiredName, info.Mode())
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if testPublishCleanupPoint != nil {
		if err := testPublishCleanupPoint("before-quarantine", name); err != nil {
			return err
		}
	}
	prefix := publishSentinelQuarantinePrefix
	if name == publishSentinelStage {
		prefix = publishSentinelStageQuarantinePrefix
	}
	quarantine, err := fsatomic.Quarantine(root, name, prefix)
	if err != nil {
		return fmt.Errorf("atomically quarantine %s: %w", label, err)
	}
	defer quarantine.Close()
	isolated, err := capturePublishFile(quarantine.Root(), quarantine.Name(), publishRecordMaxBytes+1)
	if err != nil || !samePublishFileAcrossRename(state, isolated) {
		restoreErr := quarantine.Restore()
		return errors.Join(fmt.Errorf("quarantined %s no longer matches its exact authority; restoring without replacement", label), err, restoreErr)
	}
	if testPublishCleanupPoint != nil {
		if err := testPublishCleanupPoint("after-isolate", name); err != nil {
			return err
		}
		if err := testPublishCleanupPoint("quarantine-durable", name); err != nil {
			return err
		}
		if err := testPublishCleanupPoint("before-remove", name); err != nil {
			return err
		}
	}
	if guard != nil {
		if err := guard(); err != nil {
			return fmt.Errorf("%s cleanup authority changed before deletion; preserving quarantine: %w", label, err)
		}
	}
	if err := isolated.revalidateAt(quarantine.Root(), quarantine.Name()); err != nil {
		return fmt.Errorf("privately quarantined %s changed before deletion; preserving it: %w", label, err)
	}
	if testPublishCleanupPoint != nil {
		if err := testPublishCleanupPoint("after-quarantine", name); err != nil {
			return err
		}
	}
	if info, err := root.Lstat(name); err == nil {
		return fmt.Errorf("%s public name %s was repopulated (%s); preserving it and the private quarantine", label, name, info.Mode())
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := publishQuarantineRemove(quarantine); err != nil {
		return fmt.Errorf("remove privately quarantined %s: %w", label, err)
	}
	if testPublishCleanupPoint != nil {
		if err := testPublishCleanupPoint("quarantine-removed", name); err != nil {
			return err
		}
	}
	return nil
}

func (l *Lock) beginPublish(record publishRecord) (_ *publishAuthority, retErr error) {
	body, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	body = append(body, '\n')
	root, err := os.OpenRoot(l.root)
	if err != nil {
		return nil, err
	}
	keepRoot := false
	defer func() {
		if !keepRoot {
			retErr = errors.Join(retErr, root.Close())
		}
	}()
	rooted, err := root.Lstat(".")
	if err != nil || !os.SameFile(l.rootInfo, rooted) {
		return nil, errors.Join(fmt.Errorf("design root changed identity before publication sentinel installation"), err)
	}
	for _, retired := range []string{publishSentinelRetired, publishSentinelStageRetired} {
		if info, err := root.Lstat(retired); err == nil {
			return nil, fmt.Errorf("ambiguous design publication retirement %s exists (%s)", retired, info.Mode())
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	if _, err := root.Lstat(publishSentinel); err == nil {
		state, prior, err := capturePublishRecord(root, publishSentinel)
		if err != nil {
			return nil, err
		}
		if !publishRecordsEqual(prior, record) {
			return nil, fmt.Errorf("publication sentinel belongs to operation %q with a different input/output identity; %s", prior.Operation, prior.Recovery)
		}
		keepRoot = true
		return &publishAuthority{root: root, state: state, record: prior, path: publishSentinel}, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if info, err := root.Lstat(publishSentinelStage); err == nil {
		stage, captureErr := capturePublishFile(root, publishSentinelStage, publishRecordMaxBytes+1)
		if captureErr != nil {
			return nil, errors.Join(fmt.Errorf("witness unexpected publication sentinel stage (%s)", info.Mode()), captureErr)
		}
		guardAbsent := func() error { return requirePublishFileAbsent(root, publishSentinel) }
		if err := removePublishFileExact(root, publishSentinelStage, publishSentinelStageRetired, stage, "unexpected publication sentinel stage", guardAbsent); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	f, err := root.OpenFile(publishSentinelStage, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create design publication sentinel: %w", err)
	}
	_, writeErr := f.Write(body)
	syncErr := f.Sync()
	closeErr := f.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return nil, fmt.Errorf("persist design publication sentinel: %w", err)
	}
	stageState, err := capturePublishFile(root, publishSentinelStage, publishRecordMaxBytes+1)
	if err != nil || !bytes.Equal(stageState.body, body) || !publishSentinelPermissionsSafe(stageState.mode) {
		return nil, errors.Join(fmt.Errorf("staged design publication sentinel does not match its exact record"), err)
	}
	if testAfterPublishStageSync != nil {
		if err := testAfterPublishStageSync(); err != nil {
			return nil, err
		}
	}
	if err := stageState.revalidateAt(root, publishSentinelStage); err != nil {
		return nil, fmt.Errorf("staged design publication sentinel changed after file sync: %w", err)
	}
	if err := syncDirectory(l.root); err != nil {
		return nil, fmt.Errorf("sync staged design publication sentinel directory: %w", err)
	}
	if testAfterPublishStageDirSync != nil {
		if err := testAfterPublishStageDirSync(); err != nil {
			return nil, err
		}
	}
	if err := stageState.revalidateAt(root, publishSentinelStage); err != nil {
		return nil, fmt.Errorf("staged design publication sentinel changed after directory sync: %w", err)
	}
	if info, err := root.Lstat(publishSentinel); err == nil {
		return nil, fmt.Errorf("design publication sentinel appeared before installation (%s)", info.Mode())
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err := publishRenameNoReplace(root, publishSentinelStage, publishSentinel); err != nil {
		return nil, fmt.Errorf("install design publication sentinel: %w", err)
	}
	installedState, installedRecord, err := capturePublishRecord(root, publishSentinel)
	if err != nil || !samePublishFileAcrossRename(stageState, installedState) || !publishRecordsEqual(record, installedRecord) {
		return nil, errors.Join(fmt.Errorf("installed design publication sentinel differs from its staged authority"), err)
	}
	if testAfterPublishStageRename != nil {
		if err := testAfterPublishStageRename(); err != nil {
			return nil, err
		}
	}
	if err := installedState.revalidateAt(root, publishSentinel); err != nil {
		return nil, fmt.Errorf("installed design publication sentinel changed after rename: %w", err)
	}
	if err := syncDirectory(l.root); err != nil {
		return nil, err
	}
	finalState, finalRecord, err := capturePublishRecord(root, publishSentinel)
	if err != nil || !publishRecordsEqual(record, finalRecord) || !os.SameFile(installedState.info, finalState.info) ||
		installedState.mode != finalState.mode || installedState.size != finalState.size || !installedState.mtime.Equal(finalState.mtime) ||
		installedState.change != finalState.change || !bytes.Equal(installedState.body, finalState.body) {
		return nil, errors.Join(fmt.Errorf("design publication sentinel changed before authority handoff"), err)
	}
	keepRoot = true
	return &publishAuthority{root: root, state: finalState, record: finalRecord, path: publishSentinel}, nil
}

func readPublishRecord(rootPath string) (publishRecord, error) {
	return readPublishRecordName(rootPath, publishSentinel)
}

func readPublishRecordName(rootPath, name string) (record publishRecord, retErr error) {
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return record, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	_, record, err = capturePublishRecord(root, name)
	return record, err
}

func capturePublishRecord(root *os.Root, name string) (publishFileState, publishRecord, error) {
	state, err := capturePublishFile(root, name, publishRecordMaxBytes)
	if err != nil {
		return publishFileState{}, publishRecord{}, err
	}
	record, err := decodePublishRecord(state.body)
	return state, record, err
}

func decodePublishRecord(body []byte) (record publishRecord, err error) {
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&record); err != nil {
		return record, fmt.Errorf("decode design publication sentinel: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return record, fmt.Errorf("design publication sentinel has trailing JSON")
	}
	canonical, _ := json.Marshal(record)
	canonical = append(canonical, '\n')
	if !bytes.Equal(canonical, body) || record.Version != 1 || record.Operation == "" || record.Recovery == "" {
		return record, fmt.Errorf("design publication sentinel is not canonical")
	}
	values := map[string]string{}
	outputSet := map[string]bool{}
	for _, output := range record.Outputs {
		outputSet[strings.TrimSuffix(output, "/")] = true
	}
	for i, expected := range record.Expected {
		if expected.Path == "" || expected.Value == "" || (i > 0 && record.Expected[i-1].Path >= expected.Path) {
			return record, fmt.Errorf("design publication sentinel expected outputs are not canonical")
		}
		if !outputSet[expected.Path] {
			return record, fmt.Errorf("design publication sentinel expected output %s is not declared", expected.Path)
		}
		if expected.Value != "absent" && !validExpectedFileValue(expected.Value) && !validExpectedTreeValue(expected.Value) {
			return record, fmt.Errorf("design publication sentinel expected output %s has invalid identity", expected.Path)
		}
		values[expected.Path] = expected.Value
	}
	if len(record.Expected) > 0 && fingerprintDigest(values) != record.OutputFingerprint {
		return record, fmt.Errorf("design publication sentinel output fingerprint does not match its expected inventory")
	}
	return record, nil
}

func reconcilePublishStage(rootPath string) error {
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return err
	}
	defer root.Close()
	quarantines, err := findPublishQuarantines(root)
	if err != nil {
		return err
	}
	if len(quarantines) > 1 {
		return fmt.Errorf("ambiguous design publication quarantines exist: %s", strings.Join(quarantines, ", "))
	}
	if len(quarantines) == 1 {
		if err := resumePublishQuarantine(root, quarantines[0]); err != nil {
			return err
		}
	}
	for _, retired := range []string{publishSentinelRetired, publishSentinelStageRetired} {
		if info, err := root.Lstat(retired); err == nil {
			return fmt.Errorf("ambiguous design publication retirement %s exists (%s)", retired, info.Mode())
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	stageInfo, err := root.Lstat(publishSentinelStage)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	stageState, err := capturePublishFile(root, publishSentinelStage, publishRecordMaxBytes+1)
	if err != nil {
		return fmt.Errorf("witness staged design publication sentinel (%s): %w", stageInfo.Mode(), err)
	}
	installedState, installed, installedErr := capturePublishRecord(root, publishSentinel)
	if errors.Is(installedErr, fs.ErrNotExist) {
		// A lone stage is strictly pre-rename, therefore the callback never ran.
		// Its contents may be partial because the crash can interrupt Write or
		// Sync. Discard the confined private regular file durably and let the
		// writer execute the normal publication.
		guardAbsent := func() error { return requirePublishFileAbsent(root, publishSentinel) }
		return removePublishFileExact(root, publishSentinelStage, publishSentinelStageRetired, stageState, "uninstalled staged design publication sentinel", guardAbsent)
	}
	if installedErr != nil {
		return installedErr
	}
	stage, err := decodePublishRecord(stageState.body)
	if err != nil {
		return fmt.Errorf("invalid staged design publication sentinel: %w", err)
	}
	if !publishRecordsEqual(installed, stage) {
		return fmt.Errorf("staged and installed design publication sentinels have different identities")
	}
	guardInstalled := func() error { return installedState.revalidateAt(root, publishSentinel) }
	return removePublishFileExact(root, publishSentinelStage, publishSentinelStageRetired, stageState, "redundant staged design publication sentinel", guardInstalled)
}

func findPublishQuarantine(root *os.Root) (string, error) {
	quarantines, err := findPublishQuarantines(root)
	if err != nil || len(quarantines) == 0 {
		return "", err
	}
	return quarantines[0], nil
}

func findPublishQuarantines(root *os.Root) ([]string, error) {
	dir, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	budget := snapshotBudget{maxEntries: snapshotInventoryMaxEntries, maxBytes: snapshotAggregateMaxBytes}
	entries, readErr := readSnapshotDir(dir, "design publication recovery inventory", &budget)
	closeErr := dir.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, err
	}
	var quarantines []string
	for _, entry := range entries {
		name := entry.Name()
		if publishQuarantineName(name) {
			quarantines = append(quarantines, name)
		}
	}
	sort.Strings(quarantines)
	return quarantines, nil
}

func resumePublishQuarantine(root *os.Root, directory string) (retErr error) {
	quarantine, err := fsatomic.ResumeQuarantine(root, directory, "")
	if err != nil {
		return fmt.Errorf("resume design publication quarantine %s: %w", directory, err)
	}
	defer func() { retErr = errors.Join(retErr, quarantine.Close()) }()
	source := quarantine.Source()
	wantPrefix := ""
	switch source {
	case publishSentinel:
		wantPrefix = publishSentinelQuarantinePrefix
	case publishSentinelStage:
		wantPrefix = publishSentinelStageQuarantinePrefix
	default:
		return fmt.Errorf("design publication quarantine %s claims foreign source %s; preserving it", directory, source)
	}
	if !strings.HasPrefix(directory, wantPrefix) {
		return fmt.Errorf("design publication quarantine %s has the wrong source prefix; preserving it", directory)
	}
	if testPublishCleanupPoint != nil {
		if err := testPublishCleanupPoint("reconcile-quarantine-open", source); err != nil {
			return err
		}
	}
	if info, err := root.Lstat(source); err == nil {
		return fmt.Errorf("design publication source %s was repopulated (%s); preserving it and quarantine %s", source, info.Mode(), directory)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	isolated, err := capturePublishFile(quarantine.Root(), quarantine.Name(), publishRecordMaxBytes+1)
	if errors.Is(err, fs.ErrNotExist) {
		if err := quarantine.FinishEmpty(); err != nil {
			return fmt.Errorf("finish empty design publication quarantine %s: %w", directory, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("witness design publication quarantine %s: %w", directory, err)
	}
	if source == publishSentinel {
		if _, err := decodePublishRecord(isolated.body); err != nil {
			return fmt.Errorf("quarantined installed publication sentinel is invalid; preserving %s: %w", directory, err)
		}
	} else if installedState, installed, installedErr := capturePublishRecord(root, publishSentinel); installedErr == nil {
		stage, err := decodePublishRecord(isolated.body)
		if err != nil || !publishRecordsEqual(installed, stage) {
			return errors.Join(fmt.Errorf("quarantined publication stage does not match installed authority; preserving %s", directory), err)
		}
		if err := installedState.revalidateAt(root, publishSentinel); err != nil {
			return fmt.Errorf("installed publication authority changed while reconciling quarantine %s: %w", directory, err)
		}
	} else if !errors.Is(installedErr, fs.ErrNotExist) {
		return installedErr
	}
	if testPublishCleanupPoint != nil {
		if err := testPublishCleanupPoint("reconcile-before-quarantine-remove", source); err != nil {
			return err
		}
	}
	if err := publishQuarantineRemove(quarantine); err != nil {
		return fmt.Errorf("finish design publication quarantine %s: %w", directory, err)
	}
	if testPublishCleanupPoint != nil {
		if err := testPublishCleanupPoint("reconcile-quarantine-removed", source); err != nil {
			return err
		}
	}
	return nil
}

func publishQuarantineName(name string) bool {
	return strings.HasPrefix(name, publishSentinelQuarantinePrefix) || strings.HasPrefix(name, publishSentinelStageQuarantinePrefix)
}

func requirePublishFileAbsent(root *os.Root, name string) error {
	if info, err := root.Lstat(name); err == nil {
		return fmt.Errorf("%s appeared as %s", name, info.Mode())
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func validExpectedFileValue(value string) bool {
	parts := strings.Split(value, ":")
	if len(parts) != 4 || parts[0] != "file" || parts[2] != "sha256" || len(parts[3]) != 64 || parts[1] == "" {
		return false
	}
	for _, ch := range parts[1] {
		if ch < '0' || ch > '7' {
			return false
		}
	}
	return isLowerHex(parts[3])
}

func validExpectedTreeValue(value string) bool {
	const prefix = "tree:sha256:"
	return strings.HasPrefix(value, prefix) && len(value) == len(prefix)+64 && isLowerHex(strings.TrimPrefix(value, prefix))
}

func isLowerHex(value string) bool {
	for _, ch := range value {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

func publishRecordsEqual(a, b publishRecord) bool {
	if a.Version != b.Version || a.Operation != b.Operation || a.Recovery != b.Recovery || a.InputFingerprint != b.InputFingerprint || a.OutputFingerprint != b.OutputFingerprint {
		return false
	}
	if len(a.Expected) != len(b.Expected) {
		return false
	}
	for i := range a.Expected {
		if a.Expected[i] != b.Expected[i] {
			return false
		}
	}
	if strings.Join(a.Outputs, "\x00") == strings.Join(b.Outputs, "\x00") {
		return true
	}
	// An exact retry may have completed deletion of one previously recorded
	// stale output before crashing. Its now-smaller inventory may finish the
	// same operation, but it may never introduce a different target.
	prior := map[string]bool{}
	for _, output := range a.Outputs {
		prior[output] = true
	}
	for _, output := range b.Outputs {
		if !prior[output] {
			return false
		}
	}
	return true
}

func fingerprintDigest(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, key := range keys {
		fmt.Fprintf(h, "%s\x00%s\x00", key, values[key])
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}

type pathExclusions map[string]bool // true=subtree, false=exact entry

func (l *Lock) excludedPaths(outputs []string) (pathExclusions, error) {
	return excludedPathsFor(l.root, outputs)
}

func excludedPathsFor(root string, outputs []string) (pathExclusions, error) {
	out := pathExclusions{publishSentinel: false}
	for _, path := range outputs {
		prefix := strings.HasSuffix(filepath.ToSlash(path), "/**")
		if prefix {
			path = strings.TrimSuffix(filepath.ToSlash(path), "/**")
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			return nil, err
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			// External output is an explicit capability: it is bound by absolute
			// identity here and independently protected by artifactset's rooted
			// transaction lock. It is not part of the design fingerprint.
			out["external:"+filepath.ToSlash(abs)] = false
			continue
		}
		rel = filepath.ToSlash(rel)
		if prefix {
			rel += "/"
		}
		out[rel] = prefix
		if !prefix {
			parent := filepath.ToSlash(filepath.Dir(rel))
			for parent != "." && parent != "/" {
				out[strings.TrimSuffix(parent, "/")+"/"] = false
				parent = filepath.ToSlash(filepath.Dir(parent))
			}
		}
	}
	return out, nil
}

func (l *Lock) canonicalOutputs(outputs []string) ([]string, error) {
	set := map[string]bool{}
	for _, path := range outputs {
		prefix := strings.HasSuffix(filepath.ToSlash(path), "/**")
		if prefix {
			path = strings.TrimSuffix(filepath.ToSlash(path), "/**")
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(l.root, abs)
		if err != nil {
			return nil, err
		}
		identity := filepath.ToSlash(rel)
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			identity = "external:" + filepath.ToSlash(abs)
		}
		if prefix {
			identity += "/"
		}
		set[identity] = true
	}
	out := make([]string, 0, len(set))
	for path := range set {
		out = append(out, path)
	}
	sort.Strings(out)
	return out, nil
}

func filterFingerprint(in map[string]string, excluded pathExclusions, recoveryDirs map[string]bool) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		if transactionSemanticRel(key) || artifactsetResidueInOutputDir(key, recoveryDirs) {
			continue
		}
		_, skip := excluded[key]
		if !skip {
			for path, subtree := range excluded {
				if subtree && strings.HasSuffix(path, "/") && strings.HasPrefix(key, path) {
					skip = true
					break
				}
			}
		}
		if !skip {
			out[key] = value
		}
	}
	return out
}

func recoveryOutputDirs(outputs []string) map[string]bool {
	dirs := map[string]bool{}
	for _, output := range outputs {
		if strings.HasPrefix(output, "external:") {
			continue
		}
		subtree := strings.HasSuffix(output, "/")
		output = strings.TrimSuffix(output, "/")
		dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(output)))
		if subtree {
			dir = output
		}
		if dir == "." {
			dir = ""
		}
		dirs[dir] = true
	}
	return dirs
}

func artifactsetResidueInOutputDir(rel string, dirs map[string]bool) bool {
	clean := strings.TrimSuffix(filepath.ToSlash(rel), "/")
	dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(clean)))
	if dir == "." {
		dir = ""
	}
	if !dirs[dir] {
		return false
	}
	base := filepath.Base(filepath.FromSlash(clean))
	if base == ".machinery-artifact-set.journal" {
		return true
	}
	for _, prefix := range []string{".machinery-artifact-new-", ".machinery-artifact-old-", ".machinery-artifact-journal-stage-"} {
		if hasHexSuffix(base, prefix, 32) {
			return true
		}
	}
	return false
}

func transactionSemanticRel(rel string) bool {
	if rel == ".machinery/" || rel == publishSentinelStage {
		return true
	}
	return transactionResidueRel(rel)
}

func transactionResidueRel(rel string) bool {
	slash := filepath.ToSlash(rel)
	switch slash {
	case "formal/.machinery-formal-transaction.jsonl",
		"packs/.machinery-pack-transaction.jsonl",
		".machinery/checker-project-transaction.json",
		".machinery/checker-project-transaction.committed.json",
		".machinery-embed-refresh-transaction.json",
		".machinery-embed-refresh-transaction.stage":
		return true
	}
	if embedScratchResidueRel(slash) {
		return true
	}
	clean := strings.TrimSuffix(slash, "/")
	base := filepath.Base(filepath.FromSlash(clean))
	parent := filepath.ToSlash(filepath.Dir(filepath.FromSlash(clean)))
	if publishQuarantineName(base) {
		return true
	}
	if parent == "formal" {
		for _, prefix := range []string{".machinery-formal-stage-", ".machinery-formal-stage-delete-", ".machinery-formal-backup-"} {
			if hasHexSuffix(base, prefix, 64) {
				return true
			}
		}
	}
	if parent == "packs" {
		for _, prefix := range []string{".machinery-pack-stage-", ".machinery-pack-backup-"} {
			if hasHexSuffix(base, prefix, 64) {
				return true
			}
		}
	}
	for _, prefix := range []string{".machinery-project-stage-", ".machinery-project-backup-"} {
		if hasHexSuffix(base, prefix, 16) {
			return true
		}
	}
	return false
}

func embedScratchResidueRel(rel string) bool {
	clean := strings.TrimSuffix(filepath.ToSlash(rel), "/")
	base := filepath.Base(filepath.FromSlash(clean))
	if hasDecimalSuffix(base, ".machinery-embed-new-", 6) || hasDecimalSuffix(base, ".machinery-embed-old-", 6) {
		return true
	}
	for _, component := range strings.Split(clean, "/") {
		if strings.HasPrefix(component, ".machinery-embed-delete-") {
			return true
		}
	}
	return false
}

func hasDecimalSuffix(value, prefix string, size int) bool {
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+size {
		return false
	}
	for _, ch := range value[len(prefix):] {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func hasHexSuffix(value, prefix string, size int) bool {
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+size {
		return false
	}
	for _, ch := range value[len(prefix):] {
		if ch < '0' || ch > '9' {
			if ch < 'a' || ch > 'f' {
				return false
			}
		}
	}
	return true
}

func firstFingerprintChange(before, after map[string]string) string {
	keys := map[string]bool{}
	for key := range before {
		keys[key] = true
	}
	for key := range after {
		keys[key] = true
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	for _, key := range ordered {
		if before[key] != after[key] {
			return key
		}
	}
	return ""
}

// CheckUnchanged fails when a process which does not cooperate with the
// advisory lock changed the design after acquisition. Writers call it after
// rendering and immediately before their first committed mutation.
func (l *Lock) CheckUnchanged() error {
	rootInfo, err := os.Lstat(l.root)
	if err != nil || !os.SameFile(l.rootInfo, rootInfo) {
		return fmt.Errorf("design root changed identity outside the snapshot lock")
	}
	now, err := fingerprint(l.root)
	if err != nil {
		return fmt.Errorf("recheck design snapshot: %w", err)
	}
	keys := make(map[string]bool, len(l.snapshot)+len(now))
	for key := range l.snapshot {
		keys[key] = true
	}
	for key := range now {
		keys[key] = true
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	for _, key := range ordered {
		if l.snapshot[key] != now[key] {
			return fmt.Errorf("design changed outside the snapshot lock while generating: %s", key)
		}
	}
	if err := l.checkExternalUnchanged(); err != nil {
		return err
	}
	return nil
}

func (l *Lock) checkExternalUnchanged() error {
	extKeys := make([]string, 0, len(l.external))
	for path := range l.external {
		extKeys = append(extKeys, path)
	}
	sort.Strings(extKeys)
	for _, path := range extKeys {
		state := l.external[path]
		info, err := os.Lstat(path)
		if err != nil || !os.SameFile(state.info, info) {
			return fmt.Errorf("tracked external input changed identity outside the snapshot lock: %s", path)
		}
		value, err := fingerprintExternal(path)
		if err != nil {
			return fmt.Errorf("recheck tracked external input: %w", err)
		}
		if value != state.value {
			return fmt.Errorf("tracked external input changed outside the snapshot lock: %s", path)
		}
	}
	treeKeys := make([]string, 0, len(l.externalTrees))
	for path := range l.externalTrees {
		treeKeys = append(treeKeys, path)
	}
	sort.Strings(treeKeys)
	for _, path := range treeKeys {
		state := l.externalTrees[path]
		rootInfo, err := os.Lstat(path)
		if err != nil || !os.SameFile(state.root, rootInfo) {
			return fmt.Errorf("tracked external tree changed root identity outside the snapshot lock: %s", path)
		}
		values, err := l.fingerprintExternalTree(path)
		if err != nil {
			return fmt.Errorf("recheck tracked external tree: %w", err)
		}
		if fingerprintDigest(values) != state.digest {
			return fmt.Errorf("tracked external tree changed outside the snapshot lock: %s", path)
		}
	}
	return nil
}

// Refresh records a new baseline after the lock holder performs deterministic
// one-time directory setup and before it reads any governed source.
func (l *Lock) Refresh() error {
	rootInfo, err := os.Lstat(l.root)
	if err != nil || !os.SameFile(l.rootInfo, rootInfo) {
		return fmt.Errorf("design root changed identity before snapshot refresh")
	}
	now, err := fingerprint(l.root)
	if err != nil {
		return fmt.Errorf("refresh design snapshot: %w", err)
	}
	l.snapshot = now
	if l.sourceRoot != "" {
		prior := l.sourceRoot
		priorCleanup := l.sourceCleanup
		l.sourceRoot = ""
		l.sourceCleanup = nil
		if err := l.materializeDesignSource(); err != nil {
			l.sourceRoot = prior
			l.sourceCleanup = priorCleanup
			return err
		}
		if priorCleanup == nil {
			return fmt.Errorf("prior immutable design source %s has no retained cleanup authority", prior)
		}
		if err := priorCleanup.Close(); err != nil {
			l.retiredSourceCleanups = append(l.retiredSourceCleanups, priorCleanup)
			return fmt.Errorf("remove prior immutable design source: %w", err)
		}
		return nil
	}
	return nil
}

func fingerprint(rootPath string) (out map[string]string, retErr error) {
	first, err := fingerprintRoot(rootPath, true)
	if err != nil {
		return nil, err
	}
	if testBetweenFingerprintPasses != nil {
		testBetweenFingerprintPasses(rootPath)
	}
	second, err := fingerprintRoot(rootPath, true)
	if err != nil {
		return nil, err
	}
	if changed := firstFingerprintChange(first, second); changed != "" {
		return nil, fmt.Errorf("design inventory changed while fingerprinting at %s", changed)
	}
	return second, nil
}

func fingerprintRoot(rootPath string, skipGit bool) (out map[string]string, retErr error) {
	before, err := os.Lstat(rootPath)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	inside, err := root.Lstat(".")
	if err != nil || !os.SameFile(before, inside) {
		return nil, fmt.Errorf("design root changed identity while opening rooted snapshot")
	}
	out = map[string]string{}
	caseFolded := map[string]string{}
	budget := snapshotBudget{maxEntries: snapshotInventoryMaxEntries, maxBytes: snapshotAggregateMaxBytes, maxDepth: snapshotInventoryMaxDepth}
	var walk func(string, int) error
	walk = func(dir string, depth int) error {
		if err := budget.enterDirectory("design inventory directory "+filepath.ToSlash(dir), depth); err != nil {
			return err
		}
		dirInfo, err := root.Lstat(dir)
		if err != nil || !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("design inventory directory %s changed identity before reading", filepath.ToSlash(dir))
		}
		f, err := root.Open(dir)
		if err != nil {
			return err
		}
		entries, readErr := readSnapshotDir(f, "design inventory directory "+filepath.ToSlash(dir), &budget)
		closeErr := f.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			name := entry.Name()
			if skipGit && name == ".git" {
				continue
			}
			rel := name
			if dir != "." {
				rel = filepath.Join(dir, name)
			}
			display := filepath.ToSlash(rel)
			if err := validateInventoryPath(display, caseFolded); err != nil {
				return err
			}
			info, err := root.Lstat(rel)
			if err != nil {
				return err
			}
			if err := budget.addFile(display, info); err != nil {
				return err
			}
			switch {
			case info.IsDir():
				out[display+"/"] = fmt.Sprintf("dir:%o", info.Mode().Perm())
				if err := walk(rel, depth+1); err != nil {
					return err
				}
			case info.Mode().IsRegular():
				opened, err := root.Open(rel)
				if err != nil {
					return err
				}
				openedInfo, statErr := opened.Stat()
				if statErr != nil || !sameFingerprintFile(info, openedInfo) {
					_ = opened.Close()
					return fmt.Errorf("design inventory entry %s changed identity while opening", display)
				}
				fullPath := filepath.Join(rootPath, rel)
				if testAfterFingerprintFileOpen != nil {
					testAfterFingerprintFileOpen(fullPath)
				}
				digest, readErr := streamFingerprint(fullPath, opened, openedInfo.Size())
				openedAfter, retainedStatErr := opened.Stat()
				closeErr := opened.Close()
				if err := errors.Join(readErr, retainedStatErr, closeErr); err != nil {
					return err
				}
				if testAfterFingerprintFileRead != nil {
					testAfterFingerprintFileRead(fullPath)
				}
				after, statErr := root.Lstat(rel)
				if statErr != nil || !sameFingerprintFile(info, openedAfter) || !sameFingerprintFile(info, after) {
					return fmt.Errorf("design inventory entry %s changed identity while reading", display)
				}
				out[display] = fmt.Sprintf("file:%o:%x", info.Mode().Perm(), digest)
			case info.Mode()&os.ModeSymlink != 0:
				return fmt.Errorf("design inventory entry %s is a symlink; a governed directory must be a real directory and a governed file must be regular under the held design root", display)
			default:
				return fmt.Errorf("design inventory entry %s is a special file (%s); only real directories and regular files are permitted", display, info.Mode())
			}
		}
		after, err := root.Lstat(dir)
		if err != nil {
			return err
		}
		if !os.SameFile(dirInfo, after) || dirInfo.Mode() != after.Mode() {
			return fmt.Errorf("design inventory directory %s changed identity while reading", filepath.ToSlash(dir))
		}
		return nil
	}
	if err := walk(".", 0); err != nil {
		return nil, err
	}
	return out, nil
}

func validateInventoryPath(display string, caseFolded map[string]string) error {
	if err := portablepath.ValidateRelative(display); err != nil {
		return fmt.Errorf("non-portable design path %s in design inventory: %w", display, err)
	}
	folded := strings.ToLower(display)
	if prior, exists := caseFolded[folded]; exists && prior != display {
		return fmt.Errorf("portable design-path collision: design inventory entries %s and %s alias on case-insensitive filesystems", prior, display)
	}
	caseFolded[folded] = display
	return nil
}

// Release relinquishes the design snapshot lock.
func (l *Lock) Release() error {
	if l == nil {
		return nil
	}
	var cleanupErrs []error
	if l.sourceRoot != "" {
		if l.sourceCleanup == nil {
			cleanupErrs = append(cleanupErrs, fmt.Errorf("immutable design source %s has no retained cleanup authority", l.sourceRoot))
		} else {
			if err := l.sourceCleanup.Close(); err != nil {
				cleanupErrs = append(cleanupErrs, err)
			} else {
				l.sourceRoot = ""
				l.sourceCleanup = nil
			}
		}
	}
	retained := l.retiredSourceCleanups[:0]
	for _, cleanup := range l.retiredSourceCleanups {
		if err := cleanup.Close(); err != nil {
			cleanupErrs = append(cleanupErrs, err)
			retained = append(retained, cleanup)
		}
	}
	l.retiredSourceCleanups = retained
	if l.lock != nil {
		cleanupErrs = append(cleanupErrs, l.lock.Release())
		l.lock = nil
	}
	return errors.Join(cleanupErrs...)
}

// With runs fn while holding one design snapshot.
func With(designRoot string, fn func() error) (retErr error) {
	lock, err := Acquire(designRoot)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, lock.Release()) }()
	return fn()
}

// RecoveryAction is the safe, actionable decision for one design root after
// read-only inspection of interrupted publication evidence.
type RecoveryAction string

const (
	// RecoveryActionNone reports no interrupted publication evidence.
	RecoveryActionNone RecoveryAction = "none"
	// RecoveryActionFinalize reports a content-and-mode complete publication
	// that a fresh process identity may safely finalize after revalidation.
	RecoveryActionFinalize RecoveryAction = "finalize"
	// RecoveryActionRerun reports an interruption whose only safe completion
	// is rerunning the exact writer command that was interrupted.
	RecoveryActionRerun RecoveryAction = "rerun-writer"
	// RecoveryActionConflict reports tampered or ambiguous evidence that must
	// be preserved untouched for manual review.
	RecoveryActionConflict RecoveryAction = "conflict"
)

// RecoveryOutput is the read-only status of one declared expected output.
type RecoveryOutput struct {
	Path     string
	Declared string
	Status   string
	Detail   string
}

// RecoveryJournal is one interrupted-publication journal found in the design.
type RecoveryJournal struct {
	Path   string
	Role   string
	Status string
	Detail string
}

// RecoveryReport is the complete read-only inspection result for one design
// root. Producing it never mutates, deletes, or rewrites any artifact.
type RecoveryReport struct {
	Root              string
	Action            RecoveryAction
	Writer            string
	RecoveryCommand   string
	InputFingerprint  string
	InputInventory    string
	DesignInputDigest string
	OutputFingerprint string
	Outputs           []RecoveryOutput
	Journals          []RecoveryJournal
	LiveWriter        bool
	Reasons           []string
}

// InspectRecovery performs a strictly read-only inspection of one design
// root: it identifies the interrupted publication's writer, recovery
// instruction, expected output inventory with per-output status, every
// journal and recovery residue location, whether a live process holds the
// design snapshot lock, and the actionable safe recovery decision with the
// exact reasons for every refusal. It never deletes evidence or changes
// artifacts.
func InspectRecovery(designRoot string) (*RecoveryReport, error) {
	root, err := filepath.Abs(designRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve design root: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect design root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("design root must be a real directory, not a symlink or special entry")
	}
	return inspectRecoveryAt(root, true)
}

// RecoverInterrupted completes one interrupted publication from a fresh
// process identity. It finalizes only a transaction whose exact expected
// input and output inventories revalidate, whose journal is canonical and
// rooted, and which no live writer owns; every other state is refused with
// the exact conflict explained and all evidence preserved.
func RecoverInterrupted(designRoot string) (*RecoveryReport, error) {
	root, err := filepath.Abs(designRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve design root: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect design root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("design root must be a real directory, not a symlink or special entry")
	}
	report, err := InspectRecovery(designRoot)
	if err != nil {
		return report, err
	}
	if report.Action == RecoveryActionNone {
		return report, nil
	}
	if report.Action != RecoveryActionFinalize {
		return report, recoveryRefusalError(report)
	}
	lock, err := Acquire(designRoot)
	if err != nil {
		return report, err
	}
	// Revalidate under the held lock: the state may have changed since the
	// unlocked inspection, and the completion below must operate on exactly
	// the state that was inspected.
	current, err := inspectRecoveryAt(lock.root, false)
	if err != nil {
		return report, errors.Join(err, lock.Release())
	}
	if current.Action == RecoveryActionNone {
		return current, lock.Release()
	}
	if current.Action != RecoveryActionFinalize {
		return current, errors.Join(recoveryRefusalError(current), lock.Release())
	}
	prior, err := readPublishRecord(lock.root)
	if err != nil {
		return current, errors.Join(err, lock.Release())
	}
	// Complete through the publication machinery itself: it re-derives the
	// sentinel equality (which re-proves the recorded input inventory),
	// revalidates every output twice through native file identities, refuses
	// any residue, and clears the sentinel through its own hardened retire /
	// quarantine / remove sequence.
	completeErr := lock.publish(prior.Operation, prior.Recovery, nil, nil, true, func(*OutputScope) error { return nil })
	if completeErr != nil {
		return current, errors.Join(fmt.Errorf("complete interrupted publication (nothing was deleted; the journal is retained): %w", completeErr), lock.Release())
	}
	if _, err := os.Lstat(filepath.Join(lock.root, publishSentinel)); !errors.Is(err, fs.ErrNotExist) {
		return current, errors.Join(fmt.Errorf("publication sentinel still present after recovery completion: %v", err), lock.Release())
	}
	releaseErr := lock.Release()
	final, err := InspectRecovery(designRoot)
	if err != nil {
		return current, errors.Join(err, releaseErr)
	}
	return final, releaseErr
}

func recoveryRefusalError(report *RecoveryReport) error {
	return fmt.Errorf("refusing to complete the interrupted publication:\n  - %s\nnothing was modified; all evidence is preserved", strings.Join(report.Reasons, "\n  - "))
}

func inspectRecoveryAt(root string, probe bool) (*RecoveryReport, error) {
	report := &RecoveryReport{Root: root, Action: RecoveryActionNone}
	if probe {
		held, err := filelock.Acquire(filepath.Join(root, scopeName))
		switch {
		case err == nil:
			if err := held.Release(); err != nil {
				return nil, fmt.Errorf("release design snapshot lock probe for %s: %w", root, err)
			}
		case filelock.IsContended(err):
			report.LiveWriter = true
		default:
			return nil, fmt.Errorf("probe design snapshot lock for %s: %w", root, err)
		}
	}
	journals, err := collectInterruptedJournals(root)
	if err != nil {
		return nil, err
	}
	report.Journals = journals
	sentinel := RecoveryJournal{}
	haveSentinel := false
	var residue []string
	for _, journal := range journals {
		if journal.Path == publishSentinel {
			sentinel, haveSentinel = journal, true
			continue
		}
		residue = append(residue, journal.Path)
	}
	if !haveSentinel {
		if len(residue) > 0 {
			report.Action = RecoveryActionRerun
			report.Reasons = append(report.Reasons, fmt.Sprintf("interrupted publication journals present (%s) without a publication sentinel; rerun the exact writer command that was interrupted", strings.Join(residue, ", ")))
		} else {
			report.Reasons = append(report.Reasons, "no interrupted publication found")
		}
		return report, nil
	}
	if sentinel.Status != "present" {
		report.Action = RecoveryActionConflict
		report.Reasons = append(report.Reasons, fmt.Sprintf("publication sentinel is invalid or not private (possibly tampered): %s", sentinel.Detail))
		return report, nil
	}
	record, err := readPublishRecord(root)
	if err != nil {
		return nil, err
	}
	report.Writer = record.Operation
	report.RecoveryCommand = record.Recovery
	report.InputFingerprint = record.InputFingerprint
	report.OutputFingerprint = record.OutputFingerprint
	report.Outputs = inspectExpectedOutputs(root, record)
	digest, digestErr := recoveryDesignInputDigest(root, record)
	switch {
	case digestErr != nil:
		report.InputInventory = "unwitnessable"
		report.Reasons = append(report.Reasons, fmt.Sprintf("current design cannot be witnessed for input comparison: %v", digestErr))
	case digest == record.InputFingerprint:
		report.InputInventory = "verified"
		report.DesignInputDigest = digest
	default:
		report.InputInventory = "mismatch"
		report.DesignInputDigest = digest
		report.Reasons = append(report.Reasons, "input inventory does not match the recorded publication inputs: design inputs changed after the publication began, or the writer bound tracked external inputs that a fresh process cannot re-witness; rerun the exact writer command")
	}
	outputFailures := 0
	for _, output := range report.Outputs {
		if output.Status != "match" {
			outputFailures++
			report.Reasons = append(report.Reasons, output.Detail)
		}
	}
	switch {
	case report.LiveWriter:
		report.Action = RecoveryActionRerun
		report.Reasons = append(report.Reasons, "another process holds the design snapshot lock; let it finish or rerun the writer after it exits")
	case len(residue) > 0:
		report.Action = RecoveryActionRerun
		report.Reasons = append(report.Reasons, fmt.Sprintf("publication recovery residue present (%s); rerun the exact writer command that was interrupted (its own recovery reconciles owned residue)", strings.Join(residue, ", ")))
	case outputFailures > 0:
		report.Action = RecoveryActionRerun
	case report.InputInventory != "verified":
		report.Action = RecoveryActionRerun
	default:
		report.Action = RecoveryActionFinalize
	}
	return report, nil
}

func recoveryJournalRole(rel string) string {
	switch rel {
	case publishSentinel:
		return "publish-sentinel"
	case publishSentinelStage:
		return "publish-stage"
	case publishSentinelRetired:
		return "publish-sentinel-retired"
	case publishSentinelStageRetired:
		return "publish-stage-retired"
	}
	if publishQuarantineName(filepath.Base(filepath.FromSlash(rel))) {
		return "publish-quarantine"
	}
	if embedScratchResidueRel(rel) {
		return "embed-refresh-residue"
	}
	return "interrupted-transaction-journal"
}

func collectInterruptedJournals(root string) ([]RecoveryJournal, error) {
	paths, err := collectInterruptedJournalPaths(root)
	if err != nil {
		return nil, err
	}
	out := make([]RecoveryJournal, 0, len(paths))
	for _, rel := range paths {
		journal := RecoveryJournal{Path: rel, Role: recoveryJournalRole(rel), Status: "present"}
		if rel == publishSentinel {
			if _, readErr := readPublishRecord(root); readErr != nil {
				journal.Status = "invalid"
				journal.Detail = readErr.Error()
			}
		}
		out = append(out, journal)
	}
	return out, nil
}

// inspectExpectedOutputs reports the read-only status of every declared
// expected output, mirroring the publication's own validation semantics:
// rooted no-follow inspection, native file identity before and after every
// read, exact content digest, exact permission mode, and exact tree digest.
func inspectExpectedOutputs(rootPath string, record publishRecord) []RecoveryOutput {
	out := make([]RecoveryOutput, 0, len(record.Expected))
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		for _, item := range record.Expected {
			out = append(out, RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "inspect-error", Detail: fmt.Sprintf("open design root: %v", err)})
		}
		return out
	}
	defer root.Close()
	for _, item := range record.Expected {
		expectation, err := expectationFromRecord(rootPath, item)
		if err != nil {
			out = append(out, RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "inspect-error", Detail: err.Error()})
			continue
		}
		out = append(out, recoveryProbeOutput(root, rootPath, expectation, item))
	}
	return out
}

func recoveryInspectError(item publishExpectedRecord, format string, args ...any) RecoveryOutput {
	return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "inspect-error", Detail: fmt.Sprintf(format, args...)}
}

func recoveryProbeOutput(root *os.Root, rootPath string, expectation OutputExpectation, item publishExpectedRecord) RecoveryOutput {
	abs, err := filepath.Abs(expectation.path)
	if err != nil {
		return recoveryInspectError(item, "resolve output path: %v", err)
	}
	rel, err := filepath.Rel(rootPath, abs)
	if err != nil {
		return recoveryInspectError(item, "relate output path: %v", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return recoveryProbeExternalOutput(expectation, item)
	}
	return recoveryProbeDesignOutput(root, rootPath, expectation, item, rel)
}

func recoveryProbeDesignOutput(root *os.Root, rootPath string, expectation OutputExpectation, item publishExpectedRecord, rel string) RecoveryOutput {
	display := filepath.ToSlash(rel)
	info, err := root.Lstat(rel)
	if expectation.tree != nil || expectation.treeDigest != "" {
		if errors.Is(err, fs.ErrNotExist) {
			return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "missing", Detail: fmt.Sprintf("output tree %s is missing", display)}
		}
		if err != nil {
			return recoveryInspectError(item, "inspect output tree %s: %v", display, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "tree-mismatch", Detail: fmt.Sprintf("output tree %s must be a real directory, not a symlink or file", display)}
		}
		got, ferr := fingerprintRoot(filepath.Join(rootPath, rel), false)
		if ferr != nil {
			return recoveryInspectError(item, "fingerprint output tree %s: %v", display, ferr)
		}
		confirmed, cerr := fingerprintRoot(filepath.Join(rootPath, rel), false)
		if cerr != nil {
			return recoveryInspectError(item, "fingerprint output tree %s: %v", display, cerr)
		}
		if changed := firstFingerprintChange(got, confirmed); changed != "" {
			return recoveryInspectError(item, "output tree %s changed while validating at %s", display, changed)
		}
		want := expectation.treeDigest
		if expectation.tree != nil {
			want = fingerprintDigest(expectation.tree)
		}
		if digest := fingerprintDigest(got); digest != want {
			return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "tree-mismatch", Detail: fmt.Sprintf("output tree %s contains unexpected, missing, or modified members (digest %s, want %s)", display, digest, want)}
		}
		return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "match"}
	}
	if expectation.absent {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "match"}
		case err != nil:
			return recoveryInspectError(item, "inspect deleted output %s: %v", display, err)
		default:
			return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "unexpected", Detail: fmt.Sprintf("output %s exists but was declared deleted", display)}
		}
	}
	if errors.Is(err, fs.ErrNotExist) {
		return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "missing", Detail: fmt.Sprintf("output %s is missing", display)}
	}
	if err != nil {
		return recoveryInspectError(item, "inspect output %s: %v", display, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "symlink", Detail: fmt.Sprintf("output %s must be a regular non-symlink file, not a symbolic link", display)}
	}
	if !info.Mode().IsRegular() {
		return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "special", Detail: fmt.Sprintf("output %s must be a regular non-symlink file, not a special entry", display)}
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != expectation.mode.Perm() {
		return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "mode-mismatch", Detail: fmt.Sprintf("output %s has mode %04o, want %04o", display, info.Mode().Perm(), expectation.mode.Perm())}
	}
	file, err := root.Open(rel)
	if err != nil {
		return recoveryInspectError(item, "open output %s: %v", display, err)
	}
	opened, statErr := file.Stat()
	if statErr != nil || !sameFingerprintFile(info, opened) {
		_ = file.Close()
		return recoveryInspectError(item, "output %s changed identity while opening", display)
	}
	digest, readErr := streamFingerprint(display, file, opened.Size())
	openedAfter, retainedStatErr := file.Stat()
	closeErr := file.Close()
	if err := errors.Join(readErr, retainedStatErr, closeErr); err != nil {
		return recoveryInspectError(item, "read output %s: %v", display, err)
	}
	after, statErr := root.Lstat(rel)
	if statErr != nil || !sameFingerprintFile(info, openedAfter) || !sameFingerprintFile(info, after) {
		return recoveryInspectError(item, "output %s changed identity while reading", display)
	}
	if got := fmt.Sprintf("sha256:%x", digest); got != expectation.digest {
		return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "content-mismatch", Detail: fmt.Sprintf("output %s has digest %s, want %s", display, got, expectation.digest)}
	}
	return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "match"}
}

// recoveryProbeExternalOutput applies the same read-only identity discipline
// to outputs recorded outside the design root. The hardened apply path
// independently revalidates external outputs through retained parent
// capabilities, so a report is never sufficient for mutation.
func recoveryProbeExternalOutput(expectation OutputExpectation, item publishExpectedRecord) RecoveryOutput {
	abs := expectation.path
	info, err := os.Lstat(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "missing", Detail: fmt.Sprintf("external output %s is missing", abs)}
	}
	if err != nil {
		return recoveryInspectError(item, "inspect external output %s: %v", abs, err)
	}
	for walk := abs; ; {
		parent := filepath.Dir(walk)
		if parent == walk {
			break
		}
		parentInfo, parentErr := os.Lstat(parent)
		if parentErr != nil {
			return recoveryInspectError(item, "inspect external output ancestor %s: %v", parent, parentErr)
		}
		if parentInfo.Mode()&os.ModeSymlink != 0 || !parentInfo.IsDir() {
			return recoveryInspectError(item, "external output ancestor %s must be a real directory, not a symlink or file", parent)
		}
		walk = parent
	}
	if expectation.absent {
		return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "unexpected", Detail: fmt.Sprintf("external output %s exists but was declared deleted", abs)}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "symlink", Detail: fmt.Sprintf("external output %s must be a regular non-symlink file, not a symbolic link", abs)}
	}
	if !info.Mode().IsRegular() {
		return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "special", Detail: fmt.Sprintf("external output %s must be a regular non-symlink file, not a special entry", abs)}
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != expectation.mode.Perm() {
		return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "mode-mismatch", Detail: fmt.Sprintf("external output %s has mode %04o, want %04o", abs, info.Mode().Perm(), expectation.mode.Perm())}
	}
	file, err := os.Open(abs)
	if err != nil {
		return recoveryInspectError(item, "open external output %s: %v", abs, err)
	}
	opened, statErr := file.Stat()
	if statErr != nil || !sameFingerprintFile(info, opened) {
		_ = file.Close()
		return recoveryInspectError(item, "external output %s changed identity while opening", abs)
	}
	digest, readErr := streamFingerprint(abs, file, opened.Size())
	openedAfter, retainedStatErr := file.Stat()
	closeErr := file.Close()
	if err := errors.Join(readErr, retainedStatErr, closeErr); err != nil {
		return recoveryInspectError(item, "read external output %s: %v", abs, err)
	}
	after, statErr := os.Lstat(abs)
	if statErr != nil || !sameFingerprintFile(info, openedAfter) || !sameFingerprintFile(info, after) {
		return recoveryInspectError(item, "external output %s changed identity while reading", abs)
	}
	if got := fmt.Sprintf("sha256:%x", digest); got != expectation.digest {
		return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "content-mismatch", Detail: fmt.Sprintf("external output %s has digest %s, want %s", abs, got, expectation.digest)}
	}
	return RecoveryOutput{Path: item.Path, Declared: item.Value, Status: "match"}
}

// recoveryDesignInputDigest recomputes the design-root portion of the
// publication's recorded input fingerprint: the full snapshot filtered by
// the exact declared output exclusions and recovery directories. It matches
// the recorded value only when the writer tracked no external inputs and no
// design input changed since the publication began.
func recoveryDesignInputDigest(rootPath string, record publishRecord) (string, error) {
	outputs := make([]string, 0, len(record.Outputs))
	for _, identity := range record.Outputs {
		if external := strings.TrimPrefix(identity, "external:"); external != identity {
			outputs = append(outputs, filepath.FromSlash(external))
			continue
		}
		subtree := strings.HasSuffix(identity, "/")
		clean := strings.TrimSuffix(identity, "/")
		abs := filepath.Join(rootPath, filepath.FromSlash(clean))
		if subtree {
			abs += "/**"
		}
		outputs = append(outputs, abs)
	}
	excluded, err := excludedPathsFor(rootPath, outputs)
	if err != nil {
		return "", err
	}
	snapshot, err := fingerprint(rootPath)
	if err != nil {
		return "", err
	}
	filtered := filterFingerprint(snapshot, excluded, recoveryOutputDirs(record.Outputs))
	values := make(map[string]string, len(filtered))
	for key, value := range filtered {
		values["design:"+key] = value
	}
	return fingerprintDigest(values), nil
}
