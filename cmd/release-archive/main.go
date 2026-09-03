// release-archive creates repository-owned deterministic release tarballs.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // Git SHA-1 object-format compatibility, verified as identity rather than security
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/filelock"
	"github.com/RamXX/machinery/internal/fsatomic"
	"github.com/RamXX/machinery/internal/gitcontrol"
	"github.com/RamXX/machinery/internal/portablepath"
	"github.com/RamXX/machinery/internal/processcontrol"
)

const (
	archiveRoot                 = "machinery"
	defaultGitTimeout           = 30 * time.Second
	defaultGitListLimit         = int64(16 << 20)
	defaultGitBlobLimit         = int64(64 << 20)
	defaultArchiveLimit         = int64(256 << 20)
	archiveOutputLimit          = int64(512 << 20)
	archiveStagePrefix          = ".release-archive-"
	archiveStageSuffix          = ".stage"
	archiveRetireSuffix         = ".retire"
	archivePublishDeletePrefix  = ".release-output-delete-"
	archivePublishWitnessSuffix = ".publish-witness"
	archiveWitnessStagePrefix   = ".release-archive-witness-"
	archivePublishWitnessLimit  = int64(4096)
	archiveMaxEntries           = 65_536
	archiveDirBatch             = 256
)

var (
	errArchivePublicationAmbiguous  = errors.New("ambiguous release archive publication")
	gitCommandTimeout               = defaultGitTimeout
	gitListLimit                    = defaultGitListLimit
	gitBlobLimit                    = defaultGitBlobLimit
	archiveInputLimit               = defaultArchiveLimit
	newGitCommand                   = exec.CommandContext
	afterCommitResolve              func(string)
	afterInputInspect               func(string) error
	afterInputRead                  func(string) error
	afterArchiveSync                func(string) error
	archivePublishPoint             = func(string) error { return nil }
	archivePublicationRecoveryPoint = func(*os.Root, string) error { return nil }
	archiveWitnessPoint             = func(string) error { return nil }
	archiveWitnessRandomRead        = rand.Read
	archiveCleanupPoint             = func(string, string) error { return nil }
	archivePrivateCleanupPoint      = func(*os.Root, string) error { return nil }
	archivePrivateRemovePoint       = func(*os.Root, string) error { return nil }
	quarantineArchiveCleanup        = fsatomic.Quarantine
	publishArchiveNoReplace         = fsatomic.RenameNoReplace
	closeArchiveInput               = func(file *os.File) error { return file.Close() }
	closeArchiveRoot                = func(root *os.Root) error { return root.Close() }
	syncOutputDir                   = syncArchiveDirectory
)

type entryKind uint8

const (
	entryRegular entryKind = iota
	entrySymlink
)

type entry struct {
	name string
	mode os.FileMode
	kind entryKind
	data []byte
}

type archivePublicationRetirement struct {
	handle  *fsatomic.Quarantined
	witness archiveOutputState
}

type archivePublicationWitness struct {
	Version     int    `json:"version"`
	Stage       string `json:"stage"`
	Output      string `json:"output"`
	Authority   string `json:"authority"`
	Identity    string `json:"identity"`
	Hash        string `json:"hash"`
	Mode        uint32 `json:"mode"`
	Size        int64  `json:"size"`
	ModTimeNano int64  `json:"mod_time_unix_nano"`
}

type archivePublicationWitnessAuthority struct {
	name       string
	state      archiveOutputState
	finalState archiveOutputState
	published  bool
}

func main() {
	var input, output, archiveName, root string
	var epoch int64
	flag.StringVar(&input, "input", "", "single input file")
	flag.StringVar(&output, "output", "", "output .tar.gz")
	flag.StringVar(&archiveName, "name", "", "single input name inside archive")
	flag.StringVar(&root, "git-root", "", "archive one resolved Git HEAD tree under machinery/")
	flag.Int64Var(&epoch, "epoch", 0, "SOURCE_DATE_EPOCH")
	flag.Parse()
	if output == "" || epoch <= 0 || (input == "") == (root == "") {
		fmt.Fprintln(os.Stderr, "usage: release-archive -output out.tar.gz -epoch N (-input file -name name | -git-root dir)")
		os.Exit(2)
	}
	var entries []entry
	var err error
	if input != "" {
		entries, err = singleInputEntry(input, archiveName)
	} else {
		entries, err = committedEntries(root)
	}
	if err == nil {
		err = writeArchive(output, time.Unix(epoch, 0).UTC(), entries)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func singleInputEntry(input, name string) ([]entry, error) {
	if err := portablepath.ValidateBase(name); err != nil {
		return nil, fmt.Errorf("archive input name: %w", err)
	}
	abs, err := filepath.Abs(input)
	if err != nil {
		return nil, fmt.Errorf("resolve archive input: %w", err)
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return nil, fmt.Errorf("resolve archive input parent: %w", err)
	}
	parentBefore, err := os.Lstat(parent)
	if err != nil || !parentBefore.IsDir() || parentBefore.Mode()&os.ModeSymlink != 0 {
		return nil, errors.Join(fmt.Errorf("archive input parent must be a real directory"), err)
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return nil, fmt.Errorf("open archive input parent: %w", err)
	}
	openedParent, parentStatErr := root.Lstat(".")
	if parentStatErr != nil || !os.SameFile(parentBefore, openedParent) || parentBefore.Mode() != openedParent.Mode() {
		return nil, errors.Join(fmt.Errorf("archive input parent changed while opening"), parentStatErr, closeArchiveRoot(root))
	}
	base := filepath.Base(abs)
	before, err := root.Lstat(base)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("inspect archive input: %w", err), closeArchiveRoot(root))
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, errors.Join(fmt.Errorf("input must be a regular file, not a symlink or special file"), closeArchiveRoot(root))
	}
	if afterInputInspect != nil {
		if err := afterInputInspect(abs); err != nil {
			return nil, errors.Join(err, closeArchiveRoot(root))
		}
	}
	// Root.Open performs the platform's no-follow, beneath-root walk. A final
	// symlink that remains within the root is still rejected by the SameFile
	// proof below; a link outside the bound parent cannot be followed at all.
	f, err := root.Open(base)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("open archive input: %w", err), closeArchiveRoot(root))
	}
	opened, statErr := f.Stat()
	if statErr != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) || !sameArchiveMetadata(before, opened) {
		return nil, errors.Join(fmt.Errorf("input must be the inspected regular file"), statErr, closeArchiveInput(f), closeArchiveRoot(root))
	}
	data, readErr := readBounded(f, archiveInputLimit, "archive input")
	if readErr != nil {
		return nil, errors.Join(readErr, closeArchiveInput(f), closeArchiveRoot(root))
	}
	if afterInputRead != nil {
		if err := afterInputRead(abs); err != nil {
			return nil, errors.Join(err, closeArchiveInput(f), closeArchiveRoot(root))
		}
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Join(fmt.Errorf("rewind archive input: %w", err), closeArchiveInput(f), closeArchiveRoot(root))
	}
	recheck, readErr := readBounded(f, archiveInputLimit, "archive input revalidation")
	openedAfter, statErr := f.Stat()
	pathAfter, pathErr := root.Lstat(base)
	parentAfter, parentErr := os.Lstat(parent)
	closeErr := errors.Join(closeArchiveInput(f), closeArchiveRoot(root))
	if err := errors.Join(readErr, statErr, pathErr, parentErr, closeErr); err != nil {
		return nil, err
	}
	if !bytes.Equal(data, recheck) || !os.SameFile(parentBefore, parentAfter) || parentBefore.Mode() != parentAfter.Mode() ||
		!os.SameFile(before, openedAfter) || !os.SameFile(before, pathAfter) ||
		!sameArchiveMetadata(before, openedAfter) || !sameArchiveMetadata(before, pathAfter) {
		return nil, fmt.Errorf("archive input changed while being read")
	}
	return []entry{{name: name, mode: 0o755, kind: entryRegular, data: data}}, nil
}

func sameArchiveMetadata(left, right os.FileInfo) bool {
	if left == nil || right == nil || left.Mode() != right.Mode() || left.Size() != right.Size() || !left.ModTime().Equal(right.ModTime()) {
		return false
	}
	leftChange, rightChange := archiveFileChangeID(left), archiveFileChangeID(right)
	return leftChange == rightChange
}

// archiveFileChangeID extracts the strongest native metadata-change witness
// exposed through os.FileInfo. It catches content ABA even when an attacker
// restores the bytes, size, mode, and mtime before the second read.
func archiveFileChangeID(info os.FileInfo) string {
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
	sec, nsec := value.FieldByName("Ctime"), value.FieldByName("Ctimensec")
	if sec.IsValid() && nsec.IsValid() && sec.CanInt() && nsec.CanInt() {
		return fmt.Sprintf("%d:%d", sec.Int(), nsec.Int())
	}
	return ""
}

func committedEntries(root string) ([]entry, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve Git root: %w", err)
	}
	resolved, err := runGit(root, gitListLimit, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return nil, err
	}
	commit := strings.TrimSpace(string(resolved))
	if !validObjectID(commit) {
		return nil, fmt.Errorf("git rev-parse returned malformed commit object %q", commit)
	}
	if afterCommitResolve != nil {
		afterCommitResolve(commit)
	}
	tree, err := runGit(root, gitListLimit, "ls-tree", "-r", "-z", "--full-tree", commit)
	if err != nil {
		return nil, err
	}
	type treeEntry struct {
		path string
		oid  string
		mode os.FileMode
		kind entryKind
	}
	var planned []treeEntry
	seenPath := make(map[string]string)
	seenPrefix := make(map[string]string)
	for _, record := range bytes.Split(tree, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		if len(planned) >= archiveMaxEntries {
			return nil, fmt.Errorf("committed source tree exceeds %d-entry limit", archiveMaxEntries)
		}
		tab := bytes.IndexByte(record, '\t')
		if tab <= 0 || tab == len(record)-1 {
			return nil, fmt.Errorf("git ls-tree returned malformed record")
		}
		meta := strings.Fields(string(record[:tab]))
		if len(meta) != 3 || !validObjectID(meta[2]) {
			return nil, fmt.Errorf("git ls-tree returned malformed entry metadata")
		}
		path := string(record[tab+1:])
		if err := validateTreePath(path, seenPath, seenPrefix); err != nil {
			return nil, err
		}
		var mode os.FileMode
		var kind entryKind
		switch {
		case meta[0] == "100644" && meta[1] == "blob":
			mode, kind = 0o644, entryRegular
		case meta[0] == "100755" && meta[1] == "blob":
			mode, kind = 0o755, entryRegular
		case meta[0] == "120000" && meta[1] == "blob":
			mode, kind = 0o777, entrySymlink
		case meta[0] == "160000" || meta[1] == "commit":
			return nil, fmt.Errorf("source tree entry %q is a submodule, which release archives do not support", path)
		default:
			return nil, fmt.Errorf("source tree entry %q has unsupported Git mode/type %s %s", path, meta[0], meta[1])
		}
		planned = append(planned, treeEntry{path: path, oid: meta[2], mode: mode, kind: kind})
	}
	sort.Slice(planned, func(i, j int) bool { return planned[i].path < planned[j].path })
	entries := make([]entry, 0, len(planned))
	var total int64
	for _, item := range planned {
		data, err := runGit(root, gitBlobLimit, "cat-file", "blob", item.oid)
		if err != nil {
			return nil, fmt.Errorf("read committed blob for %q: %w", item.path, err)
		}
		total += int64(len(data))
		if total > archiveInputLimit {
			return nil, fmt.Errorf("committed source tree exceeds %d-byte archive input bound", archiveInputLimit)
		}
		if err := verifyBlobObjectID(item.oid, data); err != nil {
			return nil, fmt.Errorf("verify committed blob for %q: %w", item.path, err)
		}
		if item.kind == entrySymlink {
			target := string(data)
			if err := portablepath.ValidateRelative(target); err != nil {
				return nil, fmt.Errorf("committed symlink %q has unsafe or non-portable blob target: %w", item.path, err)
			}
		}
		entries = append(entries, entry{name: archiveRoot + "/" + item.path, mode: item.mode, kind: item.kind, data: data})
	}
	return entries, nil
}

func verifyBlobObjectID(objectID string, data []byte) error {
	header := []byte(fmt.Sprintf("blob %d\x00", len(data)))
	var got string
	switch len(objectID) {
	case 40:
		hash := sha1.New() //nolint:gosec // required by repositories using Git's SHA-1 object format
		_, _ = hash.Write(header)
		_, _ = hash.Write(data)
		got = hex.EncodeToString(hash.Sum(nil))
	case 64:
		hash := sha256.New()
		_, _ = hash.Write(header)
		_, _ = hash.Write(data)
		got = hex.EncodeToString(hash.Sum(nil))
	default:
		return fmt.Errorf("unsupported Git object ID length %d", len(objectID))
	}
	if !strings.EqualFold(got, objectID) {
		return fmt.Errorf("blob object identity mismatch: expected %s, got %s", objectID, got)
	}
	return nil
}

func validateTreePath(path string, seenPath, seenPrefix map[string]string) error {
	if err := portablepath.ValidateRelative(path); err != nil {
		return fmt.Errorf("source tree path %q is not portable: %w", path, err)
	}
	folded := strings.ToLower(path)
	if prior, ok := seenPath[folded]; ok {
		return fmt.Errorf("source tree path %q aliases portable path %q", path, prior)
	}
	parts := strings.Split(path, "/")
	for i := range parts {
		prefix := strings.Join(parts[:i+1], "/")
		key := strings.ToLower(prefix)
		if prior, ok := seenPrefix[key]; ok && prior != prefix {
			return fmt.Errorf("source tree path prefix %q aliases portable prefix %q", prefix, prior)
		}
		seenPrefix[key] = prefix
	}
	seenPath[folded] = path
	return nil
}

func validObjectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func runGit(root string, limit int64, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()
	command := newGitCommand(ctx, "git", append([]string{"-C", root}, args...)...)
	command.Env = cleanGitEnvironment(os.Environ())
	stdout := newBoundedBuffer(limit)
	stderr := newBoundedBuffer(64 << 10)
	command.Stdout = stdout
	command.Stderr = stderr
	err := processcontrol.Run(ctx, command)
	operation := "git " + strings.Join(args, " ")
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, fmt.Errorf("%s timed out after %s", operation, gitCommandTimeout)
	}
	if stdout.exceeded {
		return nil, fmt.Errorf("%s exceeded %d-byte output bound", operation, limit)
	}
	if stderr.exceeded {
		return nil, fmt.Errorf("%s exceeded stderr output bound", operation)
	}
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return nil, fmt.Errorf("%s: %w", operation, err)
		}
		return nil, fmt.Errorf("%s: %w: %s", operation, err, message)
	}
	if message := strings.TrimSpace(stderr.String()); message != "" {
		return nil, fmt.Errorf("%s emitted stderr on success: %s", operation, message)
	}
	return bytes.Clone(stdout.Bytes()), nil
}

func cleanGitEnvironment(environment []string) []string {
	return gitcontrol.Environment(environment)
}

type boundedBuffer struct {
	buffer   bytes.Buffer
	limit    int64
	exceeded bool
}

func newBoundedBuffer(limit int64) *boundedBuffer { return &boundedBuffer{limit: limit} }

func (b *boundedBuffer) Bytes() []byte  { return b.buffer.Bytes() }
func (b *boundedBuffer) Len() int       { return b.buffer.Len() }
func (b *boundedBuffer) String() string { return b.buffer.String() }

func (b *boundedBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := b.limit - int64(b.Len())
	if remaining <= 0 {
		b.exceeded = b.exceeded || original > 0
		return original, nil
	}
	if int64(len(data)) > remaining {
		data = data[:remaining]
		b.exceeded = true
	}
	_, _ = b.buffer.Write(data)
	return original, nil
}

func readBounded(reader io.Reader, limit int64, label string) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", label, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s exceeds %d-byte bound", label, limit)
	}
	return data, nil
}

func writeArchive(output string, stamp time.Time, entries []entry) (retErr error) {
	if len(entries) > archiveMaxEntries {
		return fmt.Errorf("archive exceeds %d-entry limit", archiveMaxEntries)
	}
	seenPath := make(map[string]string)
	seenPrefix := make(map[string]string)
	var total int64
	for _, item := range entries {
		if err := validateTreePath(item.name, seenPath, seenPrefix); err != nil {
			return fmt.Errorf("archive entry: %w", err)
		}
		total += int64(len(item.data))
		if total > archiveInputLimit {
			return fmt.Errorf("archive entries exceed %d-byte input bound", archiveInputLimit)
		}
		if item.kind == entrySymlink {
			if err := portablepath.ValidateRelative(string(item.data)); err != nil {
				return fmt.Errorf("archive symlink %q target: %w", item.name, err)
			}
		}
	}
	output, err := filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve release output: %w", err)
	}
	directory := filepath.Dir(output)
	outputName := filepath.Base(output)
	if outputName == "." || outputName == string(filepath.Separator) {
		return fmt.Errorf("release output must name a file")
	}
	directoryChain, err := openArchiveDirectoryChain(directory)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, directoryChain.close()) }()
	lock, err := filelock.AcquireWait(filepath.Join(directory, ".release-archive-directory"))
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, lock.Release())
	}()
	if err := directoryChain.revalidate(); err != nil {
		return err
	}
	directoryRoot := directoryChain.root()
	stageName, err := archiveStageName(output)
	if err != nil {
		return err
	}
	if err := recoverArchivePublication(directoryRoot, outputName, stageName); err != nil {
		return err
	}
	if err := recoverArchiveStages(directoryRoot); err != nil {
		return err
	}
	witnessName := archivePublicationWitnessName(stageName)
	var publicationWitnessAuthority *archivePublicationWitnessAuthority
	preservePublicationWitness := false
	defer func() {
		if !preservePublicationWitness {
			retErr = errors.Join(retErr, cleanupArchivePublicationWitness(directoryRoot, witnessName, publicationWitnessAuthority))
		}
	}()
	var outputBefore *archiveOutputState
	if _, err := directoryRoot.Lstat(outputName); err == nil {
		state, err := captureArchiveOutput(directoryRoot, outputName)
		if err != nil {
			return fmt.Errorf("witness existing release archive output: %w", err)
		}
		outputBefore = &state
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	tmp, err := directoryRoot.OpenFile(stageName, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("create release archive stage %s: %w", stageName, err)
	}
	tmpPath := filepath.Join(directory, stageName)
	var stageWitness *archiveOutputState
	preserveStage := false
	defer func() {
		if !preserveStage {
			retErr = errors.Join(retErr, cleanupArchiveStageIfPresent(directoryRoot, stageName, stageWitness))
		}
	}()
	gz, err := gzip.NewWriterLevel(tmp, gzip.BestCompression)
	if err != nil {
		return errors.Join(err, tmp.Close())
	}
	gz.ModTime = stamp
	gz.OS = 255
	tw := tar.NewWriter(gz)
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	for _, item := range entries {
		header := &tar.Header{Name: item.name, Mode: int64(item.mode.Perm()), ModTime: stamp, Uid: 0, Gid: 0, Format: tar.FormatPAX}
		switch item.kind {
		case entryRegular:
			header.Typeflag = tar.TypeReg
			header.Size = int64(len(item.data))
		case entrySymlink:
			header.Typeflag = tar.TypeSymlink
			header.Linkname = string(item.data)
		default:
			return errors.Join(fmt.Errorf("archive entry %q has unsupported internal type %d", item.name, item.kind), tw.Close(), gz.Close(), tmp.Close())
		}
		if err := tw.WriteHeader(header); err != nil {
			return errors.Join(err, tw.Close(), gz.Close(), tmp.Close())
		}
		if item.kind == entryRegular {
			if _, err := tw.Write(item.data); err != nil {
				return errors.Join(err, tw.Close(), gz.Close(), tmp.Close())
			}
		}
	}
	if err := tw.Close(); err != nil {
		return errors.Join(err, gz.Close(), tmp.Close())
	}
	if err := gz.Close(); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Sync(); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if afterArchiveSync != nil {
		if err := afterArchiveSync(tmpPath); err != nil {
			return err
		}
	}
	stageState, err := captureArchiveOutput(directoryRoot, stageName)
	if err != nil {
		return fmt.Errorf("witness synced release archive stage: %w", err)
	}
	stageWitness = &stageState
	publicationWitness := newArchivePublicationWitness(stageName, outputName, stageState)
	witnessAuthority, err := writeArchivePublicationWitness(directoryRoot, witnessName, publicationWitness)
	if err != nil {
		return err
	}
	publicationWitnessAuthority = &witnessAuthority
	if err := archivePublishPoint("before-output-rename"); err != nil {
		return err
	}
	if err := directoryChain.revalidate(); err != nil {
		return err
	}
	if err := stageState.revalidateAt(directoryRoot, stageName); err != nil {
		return fmt.Errorf("release archive stage changed before publication: %w", err)
	}
	replacedOutput, err := publishArchiveOutput(directoryRoot, stageName, outputName, stageState, outputBefore)
	if err != nil {
		preserveStage = errors.Is(err, errArchivePublicationAmbiguous)
		preservePublicationWitness = preserveStage
		return err
	}
	preservePublicationWitness = true
	if replacedOutput != nil {
		defer func() { retErr = errors.Join(retErr, replacedOutput.handle.Close()) }()
	}
	if err := archivePublishPoint("after-output-rename"); err != nil {
		return err
	}
	published, err := captureArchiveOutput(directoryRoot, outputName)
	if err != nil {
		return fmt.Errorf("witness published release archive: %w", err)
	}
	if !os.SameFile(stageState.info, published.info) || stageState.hash != published.hash || stageState.mode != published.mode || stageState.size != published.size {
		return fmt.Errorf("published release archive does not match its synced stage")
	}
	if err := publicationWitness.matches(published); err != nil {
		return err
	}
	if err := archivePublishPoint("before-output-sync"); err != nil {
		return err
	}
	if err := published.revalidateAt(directoryRoot, outputName); err != nil {
		return fmt.Errorf("published release archive changed before directory sync: %w", err)
	}
	if err := syncOutputDir(directoryRoot); err != nil {
		return err
	}
	if err := archivePublishPoint("before-output-final-validation"); err != nil {
		return err
	}
	if err := published.revalidateAt(directoryRoot, outputName); err != nil {
		return fmt.Errorf("published release archive changed before success: %w", err)
	}
	if err := directoryChain.revalidate(); err != nil {
		return err
	}
	logicalOutput, err := os.Lstat(output)
	if err != nil {
		return fmt.Errorf("published release archive is absent from its requested path: %w", err)
	}
	if logicalOutput.Mode()&os.ModeSymlink != 0 || !logicalOutput.Mode().IsRegular() || !os.SameFile(published.info, logicalOutput) || !sameArchiveMetadata(published.info, logicalOutput) {
		return fmt.Errorf("published release archive at its requested path changed identity or metadata before success")
	}
	if replacedOutput != nil {
		if err := validateArchiveQuarantineInventory(replacedOutput.handle.Root(), true); err != nil {
			return preserveArchiveQuarantine(replacedOutput.handle, err)
		}
		if err := replacedOutput.witness.revalidateAt(replacedOutput.handle.Root(), replacedOutput.handle.Name()); err != nil {
			return preserveArchiveQuarantine(replacedOutput.handle, fmt.Errorf("prior release archive output changed before retirement: %w", err))
		}
		if err := replacedOutput.handle.Remove(); err != nil {
			return err
		}
	}
	if err := cleanupArchivePublicationWitness(directoryRoot, witnessName, publicationWitnessAuthority); err != nil {
		return fmt.Errorf("retire release archive publication witness: %w", err)
	}
	preservePublicationWitness = false
	publicationWitnessAuthority = nil
	return nil
}

func publishArchiveOutput(root *os.Root, stageName, outputName string, stage archiveOutputState, previous *archiveOutputState) (*archivePublicationRetirement, error) {
	if err := stage.revalidateAt(root, stageName); err != nil {
		return nil, err
	}
	var quarantined *fsatomic.Quarantined
	if previous != nil {
		if err := previous.revalidateAt(root, outputName); err != nil {
			return nil, fmt.Errorf("release archive output changed before replacement; preserving it: %w", err)
		}
		var err error
		quarantined, err = fsatomic.Quarantine(root, outputName, archivePublishDeletePrefix)
		if err != nil {
			return nil, fmt.Errorf("quarantine previous release archive output: %w", err)
		}
		if err := validateArchiveQuarantineInventory(quarantined.Root(), true); err != nil {
			return nil, preserveArchiveQuarantine(quarantined, err)
		}
		retired, err := captureArchiveOutput(quarantined.Root(), quarantined.Name())
		if err != nil || !os.SameFile(previous.info, retired.info) || previous.hash != retired.hash || previous.mode != retired.mode || previous.size != retired.size || !previous.info.ModTime().Equal(retired.info.ModTime()) {
			return nil, preserveArchiveQuarantine(quarantined, errors.Join(err, fmt.Errorf("previous release archive changed while entering private replacement authority")))
		}
	} else if _, err := root.Lstat(outputName); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			err = fmt.Errorf("release archive output appeared at publication boundary")
		}
		return nil, errors.Join(errArchivePublicationAmbiguous, err)
	}
	if err := stage.revalidateAt(root, stageName); err != nil {
		if quarantined != nil {
			return nil, errors.Join(err, quarantined.Restore(), quarantined.Close())
		}
		return nil, err
	}
	if err := publishArchiveNoReplace(root, stageName, outputName); err != nil {
		if quarantined != nil {
			return nil, errors.Join(errArchivePublicationAmbiguous, err, quarantined.Restore(), quarantined.Close())
		}
		return nil, errors.Join(errArchivePublicationAmbiguous, err)
	}
	if quarantined == nil {
		return nil, nil
	}
	retired, err := captureArchiveOutput(quarantined.Root(), quarantined.Name())
	if err != nil {
		return nil, preserveArchiveQuarantine(quarantined, err)
	}
	return &archivePublicationRetirement{handle: quarantined, witness: retired}, nil
}

func recoverArchivePublication(root *os.Root, outputName, stageName string) (retErr error) {
	entries, err := readArchiveDirectoryBounded(root, archiveMaxEntries)
	if err != nil {
		return err
	}
	witnessName := archivePublicationWitnessName(stageName)
	var authorityName string
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), archiveWitnessStagePrefix) {
			continue
		}
		if !validArchivePublicationWitnessStageName(entry.Name()) {
			return fmt.Errorf("foreign release archive publication witness authority %q", entry.Name())
		}
		if authorityName != "" {
			return fmt.Errorf("multiple release archive publication witness authorities exist")
		}
		authorityName = entry.Name()
	}
	var publicationWitness archivePublicationWitness
	var witnessState archiveOutputState
	witnessPresent := false
	var witnessErr error
	if _, err := root.Lstat(witnessName); err == nil {
		witnessPresent = true
		publicationWitness, witnessState, witnessErr = readArchivePublicationWitness(root, witnessName, stageName, outputName)
	} else if !errors.Is(err, os.ErrNotExist) {
		witnessErr = err
	}
	var authorityWitness archivePublicationWitness
	var authorityState archiveOutputState
	var authorityErr error
	if authorityName != "" {
		authorityWitness, authorityState, authorityErr = readArchivePublicationWitness(root, authorityName, stageName, outputName)
		if authorityErr == nil && authorityWitness.Authority != strings.TrimPrefix(authorityName, archiveWitnessStagePrefix) {
			authorityErr = fmt.Errorf("release archive witness authority token does not match its private name")
		}
	}
	var quarantined *fsatomic.Quarantined
	var retired archiveOutputState
	defer func() {
		if quarantined != nil {
			retErr = errors.Join(retErr, quarantined.Close())
		}
	}()
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), archivePublishDeletePrefix) {
			continue
		}
		candidate, err := fsatomic.ResumeQuarantine(root, entry.Name(), outputName)
		if err != nil {
			return err
		}
		if quarantined != nil {
			return errors.Join(fmt.Errorf("multiple release archive publication authorities exist"), candidate.Close())
		}
		quarantined = candidate
	}
	if quarantined != nil {
		if _, err := quarantined.Root().Lstat(quarantined.Name()); errors.Is(err, os.ErrNotExist) {
			if err := quarantined.FinishEmpty(); err != nil {
				return err
			}
			quarantined = nil
		} else if err != nil {
			return err
		} else if err := validateArchiveQuarantineInventory(quarantined.Root(), true); err != nil {
			return err
		} else {
			retired, err = captureArchiveOutput(quarantined.Root(), quarantined.Name())
			if err != nil {
				return fmt.Errorf("witness prior release archive output during recovery: %w", err)
			}
		}
	}
	if authorityName == "" {
		if witnessPresent || witnessErr != nil {
			return errors.Join(witnessErr, fmt.Errorf("release archive publication witness lacks its private authority; preserving it as foreign"))
		}
		if quarantined != nil {
			return fmt.Errorf("release archive publication witness authority is absent; preserving prior-output authority")
		}
	} else if authorityErr != nil {
		if witnessPresent || witnessErr != nil || quarantined != nil {
			return fmt.Errorf("release archive publication witness authority is corrupt; preserving all publication evidence: %w", errors.Join(authorityErr, witnessErr))
		}
		if err := cleanupArchiveStageIfPresent(root, authorityName, nil); err != nil {
			return fmt.Errorf("retire incomplete private release archive publication witness authority: %w", err)
		}
		authorityName = ""
	} else {
		if witnessErr != nil {
			return fmt.Errorf("release archive publication witness is corrupt; preserving publication authority: %w", witnessErr)
		}
		if !witnessPresent {
			if err := authorityState.revalidateAt(root, authorityName); err != nil {
				return fmt.Errorf("private release archive publication witness authority changed before publication: %w", err)
			}
			if err := root.Link(authorityName, witnessName); err != nil {
				return fmt.Errorf("recover release archive publication witness without replacement: %w", err)
			}
			if err := syncOutputDir(root); err != nil {
				return fmt.Errorf("persist recovered release archive publication witness: %w", err)
			}
			authorityWitness, authorityState, authorityErr = readArchivePublicationWitness(root, authorityName, stageName, outputName)
			if authorityErr != nil {
				return fmt.Errorf("revalidate private release archive publication witness after recovery link: %w", authorityErr)
			}
			publicationWitness, witnessState, witnessErr = readArchivePublicationWitness(root, witnessName, stageName, outputName)
			if witnessErr != nil {
				return fmt.Errorf("verify recovered release archive publication witness: %w", witnessErr)
			}
			witnessPresent = true
		}
		if publicationWitness != authorityWitness || !sameArchiveOutputState(authorityState, witnessState) {
			return fmt.Errorf("release archive publication witness does not match its exact private authority; preserving both")
		}
	}
	stage, stagePresent, err := captureArchiveOutputIfPresent(root, stageName)
	if err != nil {
		return err
	}
	output, outputPresent, err := captureArchiveOutputIfPresent(root, outputName)
	if err != nil {
		return err
	}
	if !witnessPresent {
		return nil
	}
	authority := &archivePublicationWitnessAuthority{name: authorityName, state: authorityState, finalState: witnessState, published: true}
	if stagePresent {
		if err := publicationWitness.matches(stage); err != nil {
			return fmt.Errorf("staged release archive does not match its durable publication witness; preserving all evidence: %w", err)
		}
		if quarantined != nil && outputPresent {
			return fmt.Errorf("ambiguous release archive publication retains output, stage, witness, and prior-output authority")
		}
		if quarantined != nil {
			if err := retired.revalidateAt(quarantined.Root(), quarantined.Name()); err != nil {
				return fmt.Errorf("prior release archive output changed before restoration; preserving it: %w", err)
			}
			if err := quarantined.Restore(); err != nil {
				return fmt.Errorf("restore prior release archive output before publication recovery: %w", err)
			}
			quarantined = nil
		}
		return cleanupArchivePublicationWitness(root, witnessName, authority)
	}
	if !outputPresent {
		return fmt.Errorf("release archive publication lost both stage and output; preserving witness and prior-output authority")
	}
	if err := publicationWitness.matches(output); err != nil {
		return fmt.Errorf("published release archive does not match its durable staged-output witness; preserving output, witness, and prior-output authority: %w", err)
	}
	if quarantined != nil {
		if err := archivePublicationRecoveryPoint(root, outputName); err != nil {
			return fmt.Errorf("release archive publication recovery boundary: %w", err)
		}
		if err := output.revalidateAt(root, outputName); err != nil {
			return fmt.Errorf("published release archive changed at prior-output retirement boundary; preserving all authority: %w", err)
		}
		if err := retired.revalidateAt(quarantined.Root(), quarantined.Name()); err != nil {
			return fmt.Errorf("prior release archive output changed during crash recovery; preserving it: %w", err)
		}
		if err := output.revalidateAt(root, outputName); err != nil {
			return fmt.Errorf("published release archive changed immediately before prior-output retirement; preserving all authority: %w", err)
		}
		if err := quarantined.Remove(); err != nil {
			return err
		}
		quarantined = nil
	}
	return cleanupArchivePublicationWitness(root, witnessName, authority)
}

func captureArchiveOutputIfPresent(root *os.Root, name string) (archiveOutputState, bool, error) {
	if _, err := root.Lstat(name); errors.Is(err, os.ErrNotExist) {
		return archiveOutputState{}, false, nil
	} else if err != nil {
		return archiveOutputState{}, false, err
	}
	state, err := captureArchiveOutput(root, name)
	return state, err == nil, err
}

func readArchiveDirectoryBounded(root *os.Root, maxEntries int) ([]os.DirEntry, error) {
	before, err := root.Lstat(".")
	if err != nil {
		return nil, err
	}
	dir, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	opened, statErr := dir.Stat()
	entries := make([]os.DirEntry, 0, min(archiveDirBatch, maxEntries))
	var readErr error
	for {
		batch, err := dir.ReadDir(archiveDirBatch)
		if len(batch) > maxEntries-len(entries) {
			readErr = fmt.Errorf("release output directory exceeds %d-entry limit", maxEntries)
			break
		}
		entries = append(entries, batch...)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			readErr = err
			break
		}
	}
	after, afterErr := dir.Stat()
	pathAfter, pathErr := root.Lstat(".")
	closeErr := dir.Close()
	if err := errors.Join(statErr, readErr, afterErr, pathErr, closeErr); err != nil {
		return nil, err
	}
	if !before.IsDir() || !os.SameFile(before, opened) || !os.SameFile(before, after) || !os.SameFile(before, pathAfter) ||
		before.Mode() != opened.Mode() || before.Mode() != after.Mode() || before.Mode() != pathAfter.Mode() ||
		!before.ModTime().Equal(opened.ModTime()) || !before.ModTime().Equal(after.ModTime()) || !before.ModTime().Equal(pathAfter.ModTime()) {
		return nil, fmt.Errorf("release output directory changed while being inventoried")
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

// archiveDirectoryChain retains every ancestor of the requested output
// directory. Construction and validation use one-component rooted operations,
// so no intermediate symlink can redirect creation or publication.
type archiveDirectoryChain struct {
	roots []*os.Root
	paths []string
	names []string
	infos []os.FileInfo
}

func openArchiveDirectoryChain(directory string) (*archiveDirectoryChain, error) {
	directory = filepath.Clean(directory)
	volume := filepath.VolumeName(directory)
	anchor := volume + string(filepath.Separator)
	relative, err := filepath.Rel(anchor, directory)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, errors.Join(fmt.Errorf("release output directory %s is not beneath its volume root", directory), err)
	}
	anchorInfo, err := os.Lstat(anchor)
	if err != nil {
		return nil, fmt.Errorf("inspect release output volume root %s: %w", anchor, err)
	}
	if anchorInfo.Mode()&os.ModeSymlink != 0 || !anchorInfo.IsDir() {
		return nil, fmt.Errorf("release output directory ancestor %s must be a real directory", anchor)
	}
	anchorRoot, err := os.OpenRoot(anchor)
	if err != nil {
		return nil, fmt.Errorf("retain release output volume root %s: %w", anchor, err)
	}
	chain := &archiveDirectoryChain{
		roots: []*os.Root{anchorRoot},
		paths: []string{anchor},
		names: []string{"."},
		infos: []os.FileInfo{anchorInfo},
	}
	openedAnchor, err := anchorRoot.Lstat(".")
	if err != nil || !openedAnchor.IsDir() || !os.SameFile(anchorInfo, openedAnchor) || anchorInfo.Mode() != openedAnchor.Mode() {
		return nil, errors.Join(fmt.Errorf("release output volume root changed while opening"), err, chain.close())
	}
	if relative == "." {
		return chain, nil
	}
	currentPath := anchor
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." || component == ".." {
			return nil, errors.Join(fmt.Errorf("invalid release output directory component %q", component), chain.close())
		}
		parent := chain.root()
		info, inspectErr := parent.Lstat(component)
		if errors.Is(inspectErr, os.ErrNotExist) {
			if err := parent.Mkdir(component, 0o755); err != nil && !os.IsExist(err) {
				return nil, errors.Join(fmt.Errorf("create release output directory component %s: %w", component, err), chain.close())
			}
			if err := syncOutputDir(parent); err != nil {
				return nil, errors.Join(fmt.Errorf("persist release output directory component %s: %w", component, err), chain.close())
			}
			info, inspectErr = parent.Lstat(component)
		}
		if inspectErr != nil {
			return nil, errors.Join(fmt.Errorf("inspect release output directory component %s: %w", component, inspectErr), chain.close())
		}
		currentPath = filepath.Join(currentPath, component)
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, errors.Join(fmt.Errorf("release output directory ancestor %s must be a real directory", currentPath), chain.close())
		}
		child, err := parent.OpenRoot(component)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("retain release output directory component %s: %w", currentPath, err), chain.close())
		}
		opened, statErr := child.Lstat(".")
		if statErr != nil || !opened.IsDir() || !os.SameFile(info, opened) || info.Mode() != opened.Mode() {
			return nil, errors.Join(fmt.Errorf("release output directory component %s changed while opening", currentPath), statErr, child.Close(), chain.close())
		}
		chain.roots = append(chain.roots, child)
		chain.paths = append(chain.paths, currentPath)
		chain.names = append(chain.names, component)
		chain.infos = append(chain.infos, info)
	}
	return chain, nil
}

func (chain *archiveDirectoryChain) root() *os.Root {
	return chain.roots[len(chain.roots)-1]
}

func (chain *archiveDirectoryChain) revalidate() error {
	for index, root := range chain.roots {
		rooted, rootErr := root.Lstat(".")
		logical, logicalErr := os.Lstat(chain.paths[index])
		if err := errors.Join(rootErr, logicalErr); err != nil {
			return fmt.Errorf("release output directory changed identity at %s: %w", chain.paths[index], err)
		}
		expected := chain.infos[index]
		if !rooted.IsDir() || rooted.Mode()&os.ModeSymlink != 0 || !logical.IsDir() || logical.Mode()&os.ModeSymlink != 0 ||
			!os.SameFile(expected, rooted) || !os.SameFile(expected, logical) || expected.Mode() != rooted.Mode() || expected.Mode() != logical.Mode() {
			return fmt.Errorf("release output directory changed identity at %s", chain.paths[index])
		}
		if index == 0 {
			continue
		}
		fromParent, err := chain.roots[index-1].Lstat(chain.names[index])
		if err != nil || !fromParent.IsDir() || fromParent.Mode()&os.ModeSymlink != 0 || !os.SameFile(expected, fromParent) || expected.Mode() != fromParent.Mode() {
			return errors.Join(fmt.Errorf("release output directory changed identity at %s", chain.paths[index]), err)
		}
	}
	return nil
}

func (chain *archiveDirectoryChain) close() error {
	var errs []error
	for index := len(chain.roots) - 1; index >= 0; index-- {
		if err := chain.roots[index].Close(); err != nil {
			errs = append(errs, fmt.Errorf("close retained release output directory %s: %w", chain.paths[index], err))
		}
	}
	return errors.Join(errs...)
}

type archiveOutputState struct {
	info     os.FileInfo
	identity string
	hash     string
	mode     os.FileMode
	size     int64
	change   string
}

func captureArchiveOutput(root *os.Root, name string) (archiveOutputState, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return archiveOutputState{}, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return archiveOutputState{}, fmt.Errorf("%s must be a regular non-symlink file", name)
	}
	if before.Size() < 0 || before.Size() > archiveOutputLimit {
		return archiveOutputState{}, fmt.Errorf("%s exceeds %d-byte release archive limit", name, archiveOutputLimit)
	}
	file, err := root.Open(name)
	if err != nil {
		return archiveOutputState{}, err
	}
	opened, statErr := file.Stat()
	identity, identityErr := archiveNativeFileIdentity(file, opened)
	digest, readErr := hashArchiveFile(file, before.Size())
	closeErr := file.Close()
	after, pathErr := root.Lstat(name)
	if err := errors.Join(statErr, identityErr, readErr, closeErr, pathErr); err != nil {
		return archiveOutputState{}, err
	}
	if !os.SameFile(before, opened) || !os.SameFile(opened, after) || !sameArchiveMetadata(before, opened) || !sameArchiveMetadata(opened, after) {
		return archiveOutputState{}, fmt.Errorf("%s changed while being witnessed", name)
	}
	return archiveOutputState{info: after, identity: identity, hash: digest, mode: after.Mode(), size: after.Size(), change: archiveFileChangeID(after)}, nil
}

func hashArchiveFile(file *os.File, exactSize int64) (string, error) {
	if exactSize < 0 || exactSize > archiveOutputLimit {
		return "", fmt.Errorf("archive file size %d exceeds %d-byte limit", exactSize, archiveOutputLimit)
	}
	hash := sha256.New()
	written, err := io.CopyN(hash, file, exactSize)
	if err != nil {
		return "", err
	}
	var extra [1]byte
	extraCount, err := file.Read(extra[:])
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if written != exactSize || extraCount != 0 {
		return "", fmt.Errorf("archive file changed beyond its exact %d-byte snapshot", exactSize)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (state archiveOutputState) revalidateAt(root *os.Root, name string) error {
	current, err := captureArchiveOutput(root, name)
	if err != nil {
		return err
	}
	if !os.SameFile(state.info, current.info) || !sameArchiveMetadata(state.info, current.info) || state.identity != current.identity || state.hash != current.hash || state.mode != current.mode || state.size != current.size || state.change != current.change {
		return fmt.Errorf("%s changed content, identity, mode, or metadata", name)
	}
	return nil
}

func archiveStageName(output string) (string, error) {
	identity, err := filelock.ScopeIdentity(output)
	if err != nil {
		return "", fmt.Errorf("resolve release output identity: %w", err)
	}
	sum := sha256.Sum256([]byte(identity))
	return archiveStagePrefix + hex.EncodeToString(sum[:]) + archiveStageSuffix, nil
}

func archivePublicationWitnessName(stageName string) string {
	return stageName + archivePublishWitnessSuffix
}

func validArchivePublicationWitnessName(name string) bool {
	return strings.HasSuffix(name, archivePublishWitnessSuffix) && validArchiveStageName(strings.TrimSuffix(name, archivePublishWitnessSuffix))
}

func newArchivePublicationWitness(stageName, outputName string, state archiveOutputState) archivePublicationWitness {
	return archivePublicationWitness{
		Version:     2,
		Stage:       stageName,
		Output:      outputName,
		Identity:    state.identity,
		Hash:        state.hash,
		Mode:        uint32(state.mode),
		Size:        state.size,
		ModTimeNano: state.info.ModTime().UnixNano(),
	}
}

func (witness archivePublicationWitness) validate(stageName, outputName string) error {
	if witness.Version != 2 || witness.Stage != stageName || witness.Output != outputName ||
		len(witness.Authority) != 32 || !allASCII(witness.Authority, "0123456789abcdef") {
		return fmt.Errorf("release archive publication witness has foreign protocol binding")
	}
	if witness.Identity == "" || len(witness.Hash) != sha256.Size*2 {
		return fmt.Errorf("release archive publication witness has incomplete identity or digest")
	}
	decoded, err := hex.DecodeString(witness.Hash)
	if err != nil || hex.EncodeToString(decoded) != witness.Hash {
		return errors.Join(err, fmt.Errorf("release archive publication witness has non-canonical digest"))
	}
	mode := os.FileMode(witness.Mode)
	if mode&os.ModeSymlink != 0 || !mode.IsRegular() || witness.Size < 0 || witness.Size > archiveOutputLimit {
		return fmt.Errorf("release archive publication witness has invalid file metadata")
	}
	return nil
}

func (witness archivePublicationWitness) matches(state archiveOutputState) error {
	if witness.Identity != state.identity || witness.Hash != state.hash || os.FileMode(witness.Mode) != state.mode || witness.Size != state.size || witness.ModTimeNano != state.info.ModTime().UnixNano() {
		return fmt.Errorf("live release archive does not match durable staged-output identity, content, mode, size, or metadata")
	}
	return nil
}

func marshalArchivePublicationWitness(witness archivePublicationWitness) ([]byte, error) {
	data, err := json.Marshal(witness)
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if int64(len(data)) > archivePublishWitnessLimit {
		return nil, fmt.Errorf("release archive publication witness exceeds %d-byte limit", archivePublishWitnessLimit)
	}
	return data, nil
}

func validArchivePublicationWitnessStageName(name string) bool {
	prefix := archiveWitnessStagePrefix
	return strings.HasPrefix(name, prefix) && len(name) == len(prefix)+32 && allASCII(strings.TrimPrefix(name, prefix), "0123456789abcdef")
}

func newArchivePublicationWitnessStageName() (string, string, error) {
	var random [16]byte
	n, err := archiveWitnessRandomRead(random[:])
	if err != nil || n != len(random) {
		return "", "", errors.Join(err, fmt.Errorf("generate exact release archive witness authority: read %d of %d random bytes", n, len(random)))
	}
	authority := hex.EncodeToString(random[:])
	return archiveWitnessStagePrefix + authority, authority, nil
}

func writeArchivePublicationWitness(root *os.Root, name string, witness archivePublicationWitness) (authority archivePublicationWitnessAuthority, retErr error) {
	stageName, token, err := newArchivePublicationWitnessStageName()
	if err != nil {
		return authority, err
	}
	witness.Authority = token
	data, err := marshalArchivePublicationWitness(witness)
	if err != nil {
		return authority, err
	}
	file, err := root.OpenFile(stageName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return authority, fmt.Errorf("create private release archive publication witness authority: %w", err)
	}
	authority.name = stageName
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, cleanupArchivePublicationWitness(root, name, &authority))
		}
	}()
	if err := archiveWitnessPoint("create"); err != nil {
		return authority, errors.Join(err, file.Close())
	}
	first := len(data) / 2
	written, writeErr := file.Write(data[:first])
	if writeErr == nil && written != first {
		writeErr = io.ErrShortWrite
	}
	if writeErr != nil {
		return authority, errors.Join(fmt.Errorf("write private release archive witness authority prefix: %w", writeErr), file.Close())
	}
	if err := archiveWitnessPoint("partial-write"); err != nil {
		return authority, errors.Join(err, file.Close())
	}
	written, writeErr = file.Write(data[first:])
	if writeErr == nil && written != len(data)-first {
		writeErr = io.ErrShortWrite
	}
	if writeErr != nil {
		return authority, errors.Join(fmt.Errorf("write private release archive witness authority: %w", writeErr), file.Close())
	}
	if err := archiveWitnessPoint("write"); err != nil {
		return authority, errors.Join(err, file.Close())
	}
	if err := file.Sync(); err != nil {
		return authority, errors.Join(fmt.Errorf("sync private release archive witness authority: %w", err), file.Close())
	}
	if err := archiveWitnessPoint("file-sync"); err != nil {
		return authority, errors.Join(err, file.Close())
	}
	if err := file.Close(); err != nil {
		return authority, fmt.Errorf("close private release archive witness authority: %w", err)
	}
	if err := archiveWitnessPoint("close"); err != nil {
		return authority, err
	}
	if err := syncOutputDir(root); err != nil {
		return authority, fmt.Errorf("persist private release archive publication witness authority directory entry: %w", err)
	}
	if err := archiveWitnessPoint("directory-sync"); err != nil {
		return authority, err
	}
	got, state, err := readArchivePublicationWitness(root, stageName, witness.Stage, witness.Output)
	if err != nil || got != witness {
		return authority, errors.Join(err, fmt.Errorf("private release archive publication witness authority did not round-trip exactly"))
	}
	authority.state = state
	if err := archiveWitnessPoint("before-publication"); err != nil {
		return authority, err
	}
	if err := root.Link(stageName, name); err != nil {
		return authority, fmt.Errorf("publish release archive witness without replacement: %w", err)
	}
	authority.published = true
	if err := archiveWitnessPoint("after-publication"); err != nil {
		return authority, err
	}
	if err := syncOutputDir(root); err != nil {
		return authority, fmt.Errorf("persist release archive publication witness directory entry: %w", err)
	}
	if err := archiveWitnessPoint("publication-directory-sync"); err != nil {
		return authority, err
	}
	gotAuthority, linkedAuthorityState, authorityErr := readArchivePublicationWitness(root, stageName, witness.Stage, witness.Output)
	got, finalState, err := readArchivePublicationWitness(root, name, witness.Stage, witness.Output)
	if joined := errors.Join(authorityErr, err); joined != nil || gotAuthority != witness || got != witness || !sameArchiveOutputState(linkedAuthorityState, finalState) {
		return authority, errors.Join(joined, fmt.Errorf("durable release archive publication witness did not retain exact private authority"))
	}
	authority.state = linkedAuthorityState
	authority.finalState = finalState
	return authority, nil
}

func readArchivePublicationWitness(root *os.Root, name, stageName, outputName string) (archivePublicationWitness, archiveOutputState, error) {
	state, err := captureArchiveOutput(root, name)
	if err != nil {
		return archivePublicationWitness{}, archiveOutputState{}, err
	}
	if state.size > archivePublishWitnessLimit {
		return archivePublicationWitness{}, archiveOutputState{}, fmt.Errorf("release archive publication witness exceeds %d-byte limit", archivePublishWitnessLimit)
	}
	file, err := root.Open(name)
	if err != nil {
		return archivePublicationWitness{}, archiveOutputState{}, err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, archivePublishWitnessLimit+1))
	closeErr := file.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return archivePublicationWitness{}, archiveOutputState{}, err
	}
	if int64(len(data)) != state.size || int64(len(data)) > archivePublishWitnessLimit {
		return archivePublicationWitness{}, archiveOutputState{}, fmt.Errorf("release archive publication witness changed size while being read")
	}
	if err := state.revalidateAt(root, name); err != nil {
		return archivePublicationWitness{}, archiveOutputState{}, fmt.Errorf("release archive publication witness changed while being read: %w", err)
	}
	var witness archivePublicationWitness
	if err := json.Unmarshal(data, &witness); err != nil {
		return archivePublicationWitness{}, archiveOutputState{}, fmt.Errorf("decode release archive publication witness: %w", err)
	}
	if err := witness.validate(stageName, outputName); err != nil {
		return archivePublicationWitness{}, archiveOutputState{}, err
	}
	canonical, err := marshalArchivePublicationWitness(witness)
	if err != nil || !bytes.Equal(data, canonical) {
		return archivePublicationWitness{}, archiveOutputState{}, errors.Join(err, fmt.Errorf("release archive publication witness is not canonical"))
	}
	return witness, state, nil
}

func sameArchiveOutputState(left, right archiveOutputState) bool {
	return left.info != nil && right.info != nil && os.SameFile(left.info, right.info) &&
		sameArchiveMetadata(left.info, right.info) && left.identity == right.identity && left.hash == right.hash &&
		left.mode == right.mode && left.size == right.size && left.change == right.change
}

func cleanupArchivePublicationWitness(root *os.Root, witnessName string, authority *archivePublicationWitnessAuthority) error {
	if authority == nil {
		return nil
	}
	if authority.published {
		currentAuthority, authorityErr := captureArchiveOutput(root, authority.name)
		currentWitness, witnessErr := captureArchiveOutput(root, witnessName)
		if err := errors.Join(authorityErr, witnessErr); err != nil {
			return fmt.Errorf("revalidate linked release archive publication witness authority: %w", err)
		}
		if !sameArchiveOutputState(currentAuthority, currentWitness) ||
			authority.state.info != nil && (authority.state.identity != currentAuthority.identity || authority.state.hash != currentAuthority.hash || authority.state.mode != currentAuthority.mode || authority.state.size != currentAuthority.size || !authority.state.info.ModTime().Equal(currentAuthority.info.ModTime())) {
			return fmt.Errorf("release archive publication witness changed before cleanup; preserving it")
		}
		authority.state = currentAuthority
		authority.finalState = currentWitness
		if err := cleanupArchiveStageIfPresent(root, witnessName, &authority.finalState); err != nil {
			return err
		}
		afterUnlink, err := captureArchiveOutput(root, authority.name)
		if err != nil {
			return fmt.Errorf("revalidate private release archive witness authority after unlinking public name: %w", err)
		}
		if authority.state.identity != afterUnlink.identity || authority.state.hash != afterUnlink.hash || authority.state.mode != afterUnlink.mode || authority.state.size != afterUnlink.size || !authority.state.info.ModTime().Equal(afterUnlink.info.ModTime()) {
			return fmt.Errorf("private release archive publication witness authority changed during public-name retirement; preserving it")
		}
		authority.state = afterUnlink
	}
	if authority.name == "" {
		return nil
	}
	expected := &authority.state
	if authority.state.info == nil {
		expected = nil
	}
	return cleanupArchiveStageIfPresent(root, authority.name, expected)
}

// recoverArchiveStages validates the complete private stage namespace while
// the directory-wide archive lock is held. A regular stage is an uncommitted,
// fully replaceable attempt and is removed durably before recomputation. Any
// symlink, special entry, or lookalike is foreign residue and blocks.
func recoverArchiveStages(root *os.Root) error {
	return recoverArchiveStagesBounded(root, archiveMaxEntries)
}

func recoverArchiveStagesBounded(root *os.Root, maxEntries int) (retErr error) {
	if maxEntries <= 0 {
		return fmt.Errorf("release output directory entry limit must be positive")
	}
	before, err := root.Lstat(".")
	if err != nil {
		return err
	}
	dir, err := root.Open(".")
	if err != nil {
		return fmt.Errorf("open release output directory inventory: %w", err)
	}
	opened, err := dir.Stat()
	if err != nil || !opened.IsDir() || !os.SameFile(before, opened) || before.Mode() != opened.Mode() {
		return errors.Join(err, fmt.Errorf("release output directory changed while opening inventory"), dir.Close())
	}
	entries := make([]os.DirEntry, 0, min(archiveDirBatch, maxEntries))
	var readErr error
	for {
		batch, err := dir.ReadDir(archiveDirBatch)
		if len(batch) > maxEntries-len(entries) {
			readErr = fmt.Errorf("release output directory exceeds %d-entry limit", maxEntries)
			break
		}
		entries = append(entries, batch...)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			readErr = err
			break
		}
	}
	after, statErr := dir.Stat()
	pathAfter, pathErr := root.Lstat(".")
	closeErr := dir.Close()
	if err := errors.Join(readErr, statErr, pathErr, closeErr); err != nil {
		return fmt.Errorf("read release output directory inventory: %w", err)
	}
	if !pathAfter.IsDir() || pathAfter.Mode()&os.ModeSymlink != 0 || !os.SameFile(opened, after) || !os.SameFile(opened, pathAfter) ||
		opened.Mode() != after.Mode() || opened.Mode() != pathAfter.Mode() || !opened.ModTime().Equal(after.ModTime()) || !opened.ModTime().Equal(pathAfter.ModTime()) {
		return fmt.Errorf("release output directory changed while being inventoried")
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	type recoveryStage struct {
		name    string
		witness archiveOutputState
	}
	stages := make([]recoveryStage, 0)
	type recoveryQuarantine struct {
		handle  *fsatomic.Quarantined
		witness archiveOutputState
		empty   bool
	}
	quarantines := make([]recoveryQuarantine, 0)
	defer func() {
		for _, quarantine := range quarantines {
			retErr = errors.Join(retErr, quarantine.handle.Close())
		}
	}()
	for _, item := range entries {
		name := item.Name()
		if !strings.HasPrefix(name, archiveStagePrefix) {
			continue
		}
		if strings.HasPrefix(name, ".release-archive-delete-") {
			handle, err := fsatomic.ResumeQuarantine(root, name, "")
			if err != nil || !validArchiveStageName(handle.Source()) && !validArchiveRetirementName(handle.Source()) && !validArchivePublicationWitnessName(handle.Source()) && !validArchivePublicationWitnessAuthorityName(handle.Source()) {
				if handle != nil {
					_ = handle.Close()
				}
				return errors.Join(err, fmt.Errorf("release archive quarantine %q has invalid source authority", name))
			}
			if _, err := handle.Root().Lstat(handle.Name()); errors.Is(err, os.ErrNotExist) {
				if err := validateArchiveQuarantineInventory(handle.Root(), false); err != nil {
					_ = handle.Close()
					return fmt.Errorf("release archive quarantine %q: %w", name, err)
				}
				quarantines = append(quarantines, recoveryQuarantine{handle: handle, empty: true})
				continue
			} else if err != nil {
				_ = handle.Close()
				return err
			}
			if err := validateArchiveQuarantineInventory(handle.Root(), true); err != nil {
				_ = handle.Close()
				return fmt.Errorf("release archive quarantine %q: %w", name, err)
			}
			witness, err := captureArchiveOutput(handle.Root(), handle.Name())
			if err != nil {
				_ = handle.Close()
				return fmt.Errorf("witness release archive quarantine %q: %w", name, err)
			}
			quarantines = append(quarantines, recoveryQuarantine{handle: handle, witness: witness})
			continue
		}
		if validArchiveRetirementName(name) {
			return fmt.Errorf("ambiguous retired release archive residue %q must be preserved", name)
		}
		if !validArchiveStageName(name) {
			return fmt.Errorf("unexpected reserved release archive residue %q", name)
		}
		witness, err := captureArchiveOutput(root, name)
		if err != nil {
			return fmt.Errorf("witness release archive residue %q: %w", name, err)
		}
		stages = append(stages, recoveryStage{name: name, witness: witness})
	}
	for index := range quarantines {
		quarantine := &quarantines[index]
		if quarantine.empty {
			if err := quarantine.handle.FinishEmpty(); err != nil {
				return err
			}
			continue
		}
		if err := quarantine.witness.revalidateAt(quarantine.handle.Root(), quarantine.handle.Name()); err != nil {
			return fmt.Errorf("release archive quarantine changed during recovery; preserving it: %w", err)
		}
		if err := quarantine.handle.Remove(); err != nil {
			return err
		}
	}
	for index := range stages {
		stage := &stages[index]
		if err := cleanupArchiveStageIfPresent(root, stage.name, &stage.witness); err != nil {
			return fmt.Errorf("recover interrupted release archive residue %q: %w", stage.name, err)
		}
	}
	return nil
}

func validateArchiveQuarantineInventory(root *os.Root, objectExpected bool) error {
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := dir.ReadDir(2)
	closeErr := dir.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return errors.Join(readErr, closeErr)
	}
	if closeErr != nil {
		return closeErr
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var want []string
	if objectExpected {
		want = []string{"object"}
	}
	if len(entries) != len(want) {
		return fmt.Errorf("private deletion authority has unexpected inventory")
	}
	for index := range want {
		if entries[index].Name() != want[index] {
			return fmt.Errorf("private deletion authority has unexpected inventory")
		}
	}
	return nil
}

func cleanupArchiveStageIfPresent(root *os.Root, stageName string, expected *archiveOutputState) error {
	if _, err := root.Lstat(stageName); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect release archive stage %s before cleanup: %w", stageName, err)
	}
	state := archiveOutputState{}
	if expected == nil {
		captured, err := captureArchiveOutput(root, stageName)
		if err != nil {
			return fmt.Errorf("witness release archive stage %s before cleanup: %w; preserving it", stageName, err)
		}
		state = captured
	} else {
		state = *expected
	}
	if err := archiveCleanupPoint("before-stage-isolate", stageName); err != nil {
		return err
	}
	if err := state.revalidateAt(root, stageName); err != nil {
		return fmt.Errorf("release archive stage %s changed before cleanup; preserving it: %w", stageName, err)
	}
	quarantined, err := quarantineArchiveCleanup(root, stageName, ".release-archive-delete-")
	if err != nil {
		return fmt.Errorf("quarantine release archive deletion authority: %w", err)
	}
	if err := archiveCleanupPoint("after-stage-isolate", stageName); err != nil {
		return preserveArchiveQuarantine(quarantined, err)
	}
	if err := archiveCleanupPoint("before-stage-remove", stageName); err != nil {
		return preserveArchiveQuarantine(quarantined, err)
	}
	if err := archivePrivateCleanupPoint(quarantined.Root(), quarantined.Name()); err != nil {
		return preserveArchiveQuarantine(quarantined, err)
	}
	if err := validateArchiveQuarantineInventory(quarantined.Root(), true); err != nil {
		return preserveArchiveQuarantine(quarantined, err)
	}
	deleting, err := captureArchiveOutput(quarantined.Root(), quarantined.Name())
	if err != nil || !os.SameFile(state.info, deleting.info) || state.hash != deleting.hash || state.mode != deleting.mode || state.size != deleting.size || !state.info.ModTime().Equal(deleting.info.ModTime()) {
		return preserveArchiveQuarantine(quarantined, errors.Join(err, fmt.Errorf("release archive deletion authority changed; preserving it")))
	}
	if err := deleting.revalidateAt(quarantined.Root(), quarantined.Name()); err != nil {
		return preserveArchiveQuarantine(quarantined, fmt.Errorf("release archive deletion authority changed before deletion: %w", err))
	}
	if err := archivePrivateRemovePoint(quarantined.Root(), quarantined.Name()); err != nil {
		return preserveArchiveQuarantine(quarantined, err)
	}
	if err := quarantined.Remove(); err != nil {
		return fmt.Errorf("remove isolated release archive stage %s: %w", stageName, err)
	}
	return nil
}

func preserveArchiveQuarantine(quarantined *fsatomic.Quarantined, cause error) error {
	return errors.Join(cause, fmt.Errorf("preserving private release archive deletion authority"), quarantined.Close())
}

func validArchiveRetirementName(name string) bool {
	return strings.HasSuffix(name, archiveRetireSuffix) && validArchiveStageName(strings.TrimSuffix(name, archiveRetireSuffix))
}

func validArchivePublicationWitnessAuthorityName(name string) bool {
	return validArchivePublicationWitnessStageName(name)
}

func validArchiveStageName(name string) bool {
	suffix := strings.TrimPrefix(name, archiveStagePrefix)
	// Compatibility with stages produced by the former os.CreateTemp-based
	// writer. Go's random suffix is one to ten ASCII decimal digits.
	if len(suffix) >= 1 && len(suffix) <= 10 && allASCII(suffix, "0123456789") {
		return true
	}
	if !strings.HasSuffix(suffix, archiveStageSuffix) {
		return false
	}
	digest := strings.TrimSuffix(suffix, archiveStageSuffix)
	return len(digest) == sha256.Size*2 && allASCII(digest, "0123456789abcdef")
}

func allASCII(value, allowed string) bool {
	for i := range len(value) {
		if !strings.ContainsRune(allowed, rune(value[i])) {
			return false
		}
	}
	return true
}
