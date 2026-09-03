// release-archive creates repository-owned deterministic release tarballs.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha1" //nolint:gosec // Git SHA-1 object-format compatibility, verified as identity rather than security
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/RamXX/machinery/internal/gitcontrol"
	"github.com/RamXX/machinery/internal/portablepath"
	"github.com/RamXX/machinery/internal/processcontrol"
)

const (
	archiveRoot         = "machinery"
	defaultGitTimeout   = 30 * time.Second
	defaultGitListLimit = int64(16 << 20)
	defaultGitBlobLimit = int64(64 << 20)
	defaultArchiveLimit = int64(256 << 20)
	archiveStagePrefix  = ".release-archive-"
	archiveStageSuffix  = ".stage"
	archiveRetireSuffix = ".retire"
)

var (
	gitCommandTimeout   = defaultGitTimeout
	gitListLimit        = defaultGitListLimit
	gitBlobLimit        = defaultGitBlobLimit
	archiveInputLimit   = defaultArchiveLimit
	newGitCommand       = exec.CommandContext
	afterCommitResolve  func(string)
	afterInputInspect   func(string) error
	afterInputRead      func(string) error
	afterArchiveSync    func(string) error
	archivePublishPoint = func(string) error { return nil }
	archiveCleanupPoint = func(string, string) error { return nil }
	closeArchiveInput   = func(file *os.File) error { return file.Close() }
	closeArchiveRoot    = func(root *os.Root) error { return root.Close() }
	replaceOutput       = replaceArchive
	syncOutputDir       = syncArchiveDirectory
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
	if err := recoverArchiveStages(directoryRoot); err != nil {
		return err
	}
	tmp, err := directoryRoot.OpenFile(stageName, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("create release archive stage %s: %w", stageName, err)
	}
	tmpPath := filepath.Join(directory, stageName)
	var stageWitness *archiveOutputState
	defer func() {
		retErr = errors.Join(retErr, cleanupArchiveStageIfPresent(directoryRoot, stageName, stageWitness))
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
	if err := archivePublishPoint("before-output-rename"); err != nil {
		return err
	}
	if err := directoryChain.revalidate(); err != nil {
		return err
	}
	if err := stageState.revalidateAt(directoryRoot, stageName); err != nil {
		return fmt.Errorf("release archive stage changed before publication: %w", err)
	}
	if err := replaceOutput(directoryRoot, stageName, outputName); err != nil {
		return err
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
	return nil
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
	info os.FileInfo
	hash string
	mode os.FileMode
	size int64
}

func captureArchiveOutput(root *os.Root, name string) (archiveOutputState, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return archiveOutputState{}, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return archiveOutputState{}, fmt.Errorf("%s must be a regular non-symlink file", name)
	}
	file, err := root.Open(name)
	if err != nil {
		return archiveOutputState{}, err
	}
	opened, statErr := file.Stat()
	hash := sha256.New()
	_, readErr := io.Copy(hash, file)
	closeErr := file.Close()
	after, pathErr := root.Lstat(name)
	if err := errors.Join(statErr, readErr, closeErr, pathErr); err != nil {
		return archiveOutputState{}, err
	}
	if !os.SameFile(before, opened) || !os.SameFile(opened, after) || !sameArchiveMetadata(before, opened) || !sameArchiveMetadata(opened, after) {
		return archiveOutputState{}, fmt.Errorf("%s changed while being witnessed", name)
	}
	return archiveOutputState{info: after, hash: hex.EncodeToString(hash.Sum(nil)), mode: after.Mode(), size: after.Size()}, nil
}

func (state archiveOutputState) revalidateAt(root *os.Root, name string) error {
	current, err := captureArchiveOutput(root, name)
	if err != nil {
		return err
	}
	if !os.SameFile(state.info, current.info) || !sameArchiveMetadata(state.info, current.info) || state.hash != current.hash || state.mode != current.mode || state.size != current.size {
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

// recoverArchiveStages validates the complete private stage namespace while
// the directory-wide archive lock is held. A regular stage is an uncommitted,
// fully replaceable attempt and is removed durably before recomputation. Any
// symlink, special entry, or lookalike is foreign residue and blocks.
func recoverArchiveStages(root *os.Root) error {
	dir, err := root.Open(".")
	if err != nil {
		return fmt.Errorf("open release output directory inventory: %w", err)
	}
	entries, readErr := dir.ReadDir(-1)
	closeErr := dir.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return fmt.Errorf("read release output directory inventory: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	type recoveryStage struct {
		name    string
		witness archiveOutputState
	}
	stages := make([]recoveryStage, 0)
	for _, item := range entries {
		name := item.Name()
		if !strings.HasPrefix(name, archiveStagePrefix) {
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
	for index := range stages {
		stage := &stages[index]
		if err := cleanupArchiveStageIfPresent(root, stage.name, &stage.witness); err != nil {
			return fmt.Errorf("recover interrupted release archive residue %q: %w", stage.name, err)
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
	retiredName := stageName + archiveRetireSuffix
	if info, err := root.Lstat(retiredName); err == nil {
		return fmt.Errorf("release archive retirement path %s already exists (%s); preserving both", retiredName, info.Mode())
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect release archive retirement path %s: %w", retiredName, err)
	}
	if err := root.Rename(stageName, retiredName); err != nil {
		return fmt.Errorf("atomically isolate release archive stage %s: %w", stageName, err)
	}
	if err := syncOutputDir(root); err != nil {
		return fmt.Errorf("persist isolated release archive stage %s: %w", stageName, err)
	}
	retired, err := captureArchiveOutput(root, retiredName)
	if err != nil || !os.SameFile(state.info, retired.info) || state.hash != retired.hash || state.mode != retired.mode || state.size != retired.size || !state.info.ModTime().Equal(retired.info.ModTime()) {
		cause := errors.Join(fmt.Errorf("isolated release archive stage %s did not match its cleanup witness", stageName), err)
		return restoreArchiveStage(root, stageName, retiredName, cause)
	}
	if err := archiveCleanupPoint("after-stage-isolate", retiredName); err != nil {
		return restoreArchiveStage(root, stageName, retiredName, err)
	}
	if err := archiveCleanupPoint("before-stage-remove", retiredName); err != nil {
		return restoreArchiveStage(root, stageName, retiredName, err)
	}
	if err := retired.revalidateAt(root, retiredName); err != nil {
		return restoreArchiveStage(root, stageName, retiredName, fmt.Errorf("retired release archive stage changed before deletion: %w", err))
	}
	if err := root.Remove(retiredName); err != nil {
		return fmt.Errorf("remove isolated release archive stage %s: %w", retiredName, err)
	}
	if err := syncOutputDir(root); err != nil {
		return fmt.Errorf("persist release archive stage cleanup %s: %w", stageName, err)
	}
	return nil
}

func restoreArchiveStage(root *os.Root, stageName, retiredName string, cause error) error {
	if info, err := root.Lstat(stageName); err == nil {
		return errors.Join(cause, fmt.Errorf("release archive stage path %s was repopulated (%s); preserving it and retirement %s", stageName, info.Mode(), retiredName))
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.Join(cause, fmt.Errorf("inspect release archive stage path before restore: %w", err))
	}
	retiredBefore, err := captureArchiveOutput(root, retiredName)
	if err != nil {
		return errors.Join(cause, fmt.Errorf("witness changed release archive retirement %s before restore: %w", retiredName, err))
	}
	// Link is an atomic no-replace restore: unlike rename, it cannot overwrite
	// a stage path populated after the absence check. Keep the retirement link
	// as conservative evidence on this already-failing path; a later run will
	// reject the ambiguous residue rather than guess that it is safe to delete.
	if err := root.Link(retiredName, stageName); err != nil {
		return errors.Join(cause, fmt.Errorf("restore changed release archive retirement %s without replacement: %w", retiredName, err))
	}
	if err := syncOutputDir(root); err != nil {
		return errors.Join(cause, fmt.Errorf("persist restored release archive stage %s: %w", stageName, err))
	}
	retiredAfter, retiredErr := captureArchiveOutput(root, retiredName)
	stageAfter, stageErr := captureArchiveOutput(root, stageName)
	if err := errors.Join(retiredErr, stageErr); err != nil {
		return errors.Join(cause, fmt.Errorf("verify restored release archive stage %s: %w", stageName, err))
	}
	if !os.SameFile(retiredBefore.info, retiredAfter.info) || !os.SameFile(retiredAfter.info, stageAfter.info) ||
		retiredBefore.hash != retiredAfter.hash || retiredAfter.hash != stageAfter.hash ||
		retiredBefore.mode != retiredAfter.mode || retiredAfter.mode != stageAfter.mode ||
		retiredBefore.size != retiredAfter.size || retiredAfter.size != stageAfter.size ||
		!retiredBefore.info.ModTime().Equal(retiredAfter.info.ModTime()) || !retiredAfter.info.ModTime().Equal(stageAfter.info.ModTime()) {
		return errors.Join(cause, fmt.Errorf("restored release archive stage %s changed while linking; preserving both names", stageName))
	}
	return errors.Join(cause, fmt.Errorf("changed release archive stage %s preserved at both its stage and retirement names", stageName))
}

func validArchiveRetirementName(name string) bool {
	return strings.HasSuffix(name, archiveRetireSuffix) && validArchiveStageName(strings.TrimSuffix(name, archiveRetireSuffix))
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
