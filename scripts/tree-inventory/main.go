// tree-inventory produces a bounded, deterministic, no-follow filesystem
// inventory for repository shell gates without relying on GNU find extensions.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const readDirPage = 256

type stringList []string

func (values *stringList) String() string { return strings.Join(*values, ",") }
func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}

type inventoryOptions struct {
	maxEntries       int
	maxDepth         int
	maxBytes         int64
	fileSuffix       string
	fileName         string
	regularFilesOnly bool
	prune            map[string]bool
}

type inventoryState struct {
	options     inventoryOptions
	entries     int
	bytes       int64
	paths       []string
	records     []inventoryRecord
	directories map[string]string
}

var inventoryAfterEnumeration = func(string) {}
var inventoryBetweenPasses = func() {}
var snapshotFilePoint = func(string, string) error { return nil }

type inventoryRecord struct {
	path string
	info os.FileInfo
}

type inventoryResult struct {
	paths       []string
	records     []inventoryRecord
	directories map[string]string
}

type snapshotOptions struct {
	inventory     inventoryOptions
	maxFileBytes  int64
	maxTotalBytes int64
	excludes      map[string]bool
}

type snapshotFileRecord struct {
	path    string
	digest  [sha256.Size]byte
	witness string
}

type snapshotPassResult struct {
	lines []string
	files []snapshotFileRecord
}

func main() {
	os.Exit(run())
}

func run() int {
	var roots stringList
	var literals stringList
	var prunes stringList
	var maxEntries int
	var maxDepth int
	var maxBytes int64
	var timeout time.Duration
	var suffix string
	var fileName string
	var regularFilesOnly bool
	var snapshotMode bool
	var excludeFile string
	var maxFileBytes int64
	var maxTotalBytes int64
	flag.Var(&roots, "root", "real directory root to traverse (repeatable)")
	flag.Var(&literals, "literal", "literal regular file to include (repeatable)")
	flag.Var(&prunes, "prune", "root-relative directory subtree to omit (repeatable)")
	flag.IntVar(&maxEntries, "max-entries", 100_000, "aggregate visited-entry ceiling")
	flag.IntVar(&maxDepth, "max-depth", 64, "maximum descendant depth")
	flag.Int64Var(&maxBytes, "max-bytes", 32<<20, "aggregate visited-path byte ceiling")
	flag.DurationVar(&timeout, "timeout", 15*time.Second, "hard traversal deadline")
	flag.StringVar(&suffix, "file-suffix", "", "emit only entries with this suffix")
	flag.StringVar(&fileName, "file-name", "", "emit only entries with this exact base name")
	flag.BoolVar(&regularFilesOnly, "regular-files-only", false, "emit only regular non-symlink files")
	flag.BoolVar(&snapshotMode, "snapshot", false, "emit a typed, content-hashed tree snapshot")
	flag.StringVar(&excludeFile, "exclude-file", "", "newline-delimited root-relative snapshot exclusions")
	flag.Int64Var(&maxFileBytes, "max-file-bytes", 32<<20, "snapshot per-file byte ceiling")
	flag.Int64Var(&maxTotalBytes, "max-total-bytes", 128<<20, "snapshot aggregate regular-file byte ceiling")
	flag.Parse()
	if flag.NArg() != 0 || len(roots)+len(literals) == 0 || maxEntries <= 0 || maxDepth < 0 || maxBytes <= 0 || timeout <= 0 || maxFileBytes <= 0 || maxTotalBytes <= 0 || strings.ContainsAny(fileName, `/\\`) || (snapshotMode && (len(roots) != 1 || len(literals) != 0 || suffix != "" || fileName != "" || regularFilesOnly)) {
		fmt.Fprintln(os.Stderr, "usage: tree-inventory -root PATH... [-literal FILE...] -max-entries N -max-depth N -max-bytes N -timeout DURATION")
		return 2
	}
	pruneSet := make(map[string]bool, len(prunes))
	for _, prune := range prunes {
		clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(prune)))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(prune) {
			fmt.Fprintf(os.Stderr, "tree-inventory: invalid prune path %q\n", prune)
			return 2
		}
		pruneSet[clean] = true
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	watchdog := time.AfterFunc(timeout, func() {
		fmt.Fprintf(os.Stderr, "tree-inventory: traversal exceeded %s timeout\n", timeout)
		os.Exit(124)
	})
	defer watchdog.Stop()
	inventoryLimits := inventoryOptions{
		maxEntries:       maxEntries,
		maxDepth:         maxDepth,
		maxBytes:         maxBytes,
		fileSuffix:       suffix,
		fileName:         fileName,
		regularFilesOnly: regularFilesOnly,
		prune:            pruneSet,
	}
	var paths []string
	var err error
	if snapshotMode {
		excludes := map[string]bool{}
		if excludeFile != "" {
			excludes, err = loadSnapshotExcludes(excludeFile, maxBytes)
		}
		if err == nil {
			paths, err = snapshotTree(ctx, roots[0], snapshotOptions{
				inventory:     inventoryLimits,
				maxFileBytes:  maxFileBytes,
				maxTotalBytes: maxTotalBytes,
				excludes:      excludes,
			})
		}
	} else {
		paths, err = inventory(ctx, roots, literals, inventoryLimits)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "tree-inventory: %v\n", err)
		return 1
	}
	for _, path := range paths {
		if _, err := fmt.Fprintln(os.Stdout, path); err != nil {
			fmt.Fprintf(os.Stderr, "tree-inventory: write inventory: %v\n", err)
			return 1
		}
	}
	return 0
}

func inventory(ctx context.Context, roots, literals []string, options inventoryOptions) ([]string, error) {
	result, err := exactInventory(ctx, roots, literals, options)
	return result.paths, err
}

func exactInventory(ctx context.Context, roots, literals []string, options inventoryOptions) (inventoryResult, error) {
	first, err := inventoryPass(ctx, roots, literals, options)
	if err != nil {
		return inventoryResult{}, err
	}
	inventoryBetweenPasses()
	second, err := inventoryPass(ctx, roots, literals, options)
	if err != nil {
		return inventoryResult{}, err
	}
	if err := compareInventoryResults(first, second); err != nil {
		return inventoryResult{}, err
	}
	return first, nil
}

func inventoryPass(ctx context.Context, roots, literals []string, options inventoryOptions) (inventoryResult, error) {
	if options.maxEntries <= 0 || options.maxDepth < 0 || options.maxBytes <= 0 {
		return inventoryResult{}, fmt.Errorf("inventory limits must be positive (depth may be zero)")
	}
	state := &inventoryState{
		options:     options,
		paths:       make([]string, 0, min(options.maxEntries, readDirPage)),
		records:     make([]inventoryRecord, 0, min(options.maxEntries, readDirPage)),
		directories: make(map[string]string),
	}
	for _, literal := range literals {
		if err := state.checkContext(ctx); err != nil {
			return inventoryResult{}, err
		}
		info, err := os.Lstat(literal)
		if err != nil {
			return inventoryResult{}, fmt.Errorf("inspect literal %s: %w", literal, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return inventoryResult{}, fmt.Errorf("literal %s must be a regular non-symlink file", literal)
		}
		if err := state.visit(filepath.Clean(literal), true, info); err != nil {
			return inventoryResult{}, err
		}
	}
	for _, name := range roots {
		if err := state.walkRoot(ctx, name); err != nil {
			return inventoryResult{}, err
		}
	}
	sort.Strings(state.paths)
	sort.Slice(state.records, func(i, j int) bool { return state.records[i].path < state.records[j].path })
	for i := 1; i < len(state.paths); i++ {
		if state.paths[i] == state.paths[i-1] {
			return inventoryResult{}, fmt.Errorf("overlapping inventory roots produced duplicate path %s", state.paths[i])
		}
	}
	for i := 1; i < len(state.records); i++ {
		if state.records[i].path == state.records[i-1].path {
			return inventoryResult{}, fmt.Errorf("overlapping inventory roots produced duplicate path %s", state.records[i].path)
		}
	}
	return inventoryResult{paths: state.paths, records: state.records, directories: state.directories}, nil
}

func (state *inventoryState) walkRoot(ctx context.Context, name string) (retErr error) {
	if err := state.checkContext(ctx); err != nil {
		return err
	}
	display := filepath.Clean(name)
	before, err := os.Lstat(display)
	if err != nil {
		return err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return fmt.Errorf("inventory root %s must be a real directory", display)
	}
	root, err := os.OpenRoot(display)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	opened, err := root.Lstat(".")
	if err != nil || !stableInfo(before, opened) {
		return errors.Join(err, fmt.Errorf("inventory root %s changed while opening", display))
	}
	if err := state.visit(display, state.matches(display, opened), opened); err != nil {
		return err
	}
	return state.walkDirectory(ctx, root, ".", display, 0)
}

func (state *inventoryState) walkDirectory(ctx context.Context, root *os.Root, rel, display string, depth int) (retErr error) {
	if err := state.checkContext(ctx); err != nil {
		return err
	}
	before, err := root.Lstat(rel)
	if err != nil || before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return errors.Join(err, fmt.Errorf("inventory directory %s must remain real", display))
	}
	dir, err := root.Open(rel)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, dir.Close()) }()
	opened, err := dir.Stat()
	if err != nil || !stableInfo(before, opened) {
		return errors.Join(err, fmt.Errorf("inventory directory %s changed while opening", display))
	}
	initialNative, err := nativeWitness(dir, opened)
	if err != nil {
		return fmt.Errorf("witness inventory directory %s: %w", display, err)
	}
	if prior, exists := state.directories[display]; exists && prior != initialNative {
		return fmt.Errorf("inventory directory %s changed between discovery and traversal", display)
	}
	state.directories[display] = initialNative
	children, err := readDirectory(dir, state.options.maxEntries-state.entries, display)
	if err != nil {
		return err
	}
	inventoryAfterEnumeration(display)
	heldAfter, heldErr := dir.Stat()
	liveAfter, liveErr := root.Lstat(rel)
	var heldNative string
	var nativeErr error
	if heldErr == nil {
		heldNative, nativeErr = nativeWitness(dir, heldAfter)
	}
	if err := errors.Join(heldErr, liveErr, nativeErr); err != nil || !stableInfo(before, heldAfter) || !stableInfo(heldAfter, liveAfter) || heldNative != initialNative {
		return errors.Join(err, fmt.Errorf("inventory directory %s changed while enumerating", display))
	}
	for _, child := range children {
		if err := state.checkContext(ctx); err != nil {
			return err
		}
		if depth >= state.options.maxDepth {
			return fmt.Errorf("inventory exceeds %d-level depth limit at %s", state.options.maxDepth, filepath.Join(display, child.Name()))
		}
		childRel := filepath.Join(rel, child.Name())
		childDisplay := filepath.Join(display, child.Name())
		info, err := root.Lstat(childRel)
		if err != nil {
			return err
		}
		pruned := state.options.prune[filepath.ToSlash(childRel)]
		if err := state.visit(childDisplay, !pruned && state.matches(childDisplay, info), info); err != nil {
			return err
		}
		if info.IsDir() && info.Mode()&os.ModeSymlink == 0 && !pruned {
			if err := state.walkDirectory(ctx, root, childRel, childDisplay, depth+1); err != nil {
				return err
			}
		}
	}
	heldFinal, heldErr := dir.Stat()
	liveFinal, liveErr := root.Lstat(rel)
	var finalNative string
	nativeErr = nil
	if heldErr == nil {
		finalNative, nativeErr = nativeWitness(dir, heldFinal)
	}
	if err := errors.Join(heldErr, liveErr, nativeErr); err != nil || !stableInfo(before, heldFinal) || !stableInfo(heldFinal, liveFinal) || finalNative != initialNative {
		return errors.Join(err, fmt.Errorf("inventory directory %s changed while walking", display))
	}
	return nil
}

func compareInventoryResults(before, after inventoryResult) error {
	if len(before.records) != len(after.records) || len(before.paths) != len(after.paths) || len(before.directories) != len(after.directories) {
		return fmt.Errorf("inventory changed between bounded passes")
	}
	for index, record := range before.records {
		other := after.records[index]
		if record.path != other.path || !stableInfo(record.info, other.info) {
			return fmt.Errorf("inventory changed between bounded passes at %s", record.path)
		}
	}
	for index, path := range before.paths {
		if after.paths[index] != path {
			return fmt.Errorf("ordered inventory changed between bounded passes at %s", path)
		}
	}
	for path, witness := range before.directories {
		if after.directories[path] != witness {
			return fmt.Errorf("directory native witness changed between bounded passes at %s", path)
		}
	}
	return nil
}

func readDirectory(dir interface {
	ReadDir(int) ([]fs.DirEntry, error)
}, remaining int, display string) ([]fs.DirEntry, error) {
	if remaining < 0 {
		return nil, fmt.Errorf("inventory exceeds aggregate entry limit at %s", display)
	}
	entries := make([]fs.DirEntry, 0, min(remaining, readDirPage))
	for {
		request := min(readDirPage, remaining-len(entries))
		if request == 0 {
			request = 1
		}
		page, err := dir.ReadDir(request)
		if len(page) == 0 && err == nil {
			return nil, fmt.Errorf("inventory directory %s returned an empty page without EOF", display)
		}
		entries = append(entries, page...)
		if len(entries) > remaining {
			return nil, fmt.Errorf("inventory exceeds aggregate entry limit at %s", display)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func (state *inventoryState) visit(path string, emit bool, info os.FileInfo) error {
	if strings.ContainsAny(path, "\r\n") {
		return fmt.Errorf("inventory path cannot be represented safely: %q", path)
	}
	if state.entries >= state.options.maxEntries {
		return fmt.Errorf("inventory exceeds %d-entry limit at %s", state.options.maxEntries, path)
	}
	pathBytes := int64(len(path) + 1)
	if pathBytes > state.options.maxBytes-state.bytes {
		return fmt.Errorf("inventory exceeds %d-byte path limit at %s", state.options.maxBytes, path)
	}
	state.entries++
	state.bytes += pathBytes
	state.records = append(state.records, inventoryRecord{path: path, info: info})
	if emit {
		state.paths = append(state.paths, path)
	}
	return nil
}

func (state *inventoryState) matches(path string, info os.FileInfo) bool {
	if state.options.fileSuffix != "" && !strings.HasSuffix(path, state.options.fileSuffix) {
		return false
	}
	if state.options.fileName != "" && filepath.Base(path) != state.options.fileName {
		return false
	}
	return !state.options.regularFilesOnly || info.Mode().IsRegular()
}

func (state *inventoryState) checkContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("inventory traversal timed out: %w", err)
	}
	return nil
}

func loadSnapshotExcludes(path string, limit int64) (map[string]bool, error) {
	body, err := readAmbientFileExact(path, limit)
	if err != nil {
		return nil, fmt.Errorf("read snapshot exclusions: %w", err)
	}
	excludes := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSuffix(string(body), "\n"), "\n") {
		if line == "" {
			continue
		}
		portable := filepath.ToSlash(line)
		if filepath.IsAbs(line) || portable == "." || !fs.ValidPath(portable) || strings.ContainsAny(line, "\r\t") {
			return nil, fmt.Errorf("snapshot exclusion %q must be a clean root-relative path", line)
		}
		if excludes[portable] {
			return nil, fmt.Errorf("duplicate snapshot exclusion %q", line)
		}
		excludes[portable] = true
	}
	return excludes, nil
}

func readAmbientFileExact(path string, limit int64) (ret []byte, retErr error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Size() < 0 || before.Size() > limit {
		return nil, fmt.Errorf("%s must be a bounded regular non-symlink file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil || !stableInfo(before, opened) {
		return nil, errors.Join(err, fmt.Errorf("%s changed while opening", path))
	}
	initialNative, err := nativeWitness(file, opened)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(io.LimitReader(file, before.Size()+1))
	if err != nil {
		return nil, err
	}
	after, statErr := file.Stat()
	live, liveErr := os.Lstat(path)
	var finalNative string
	if statErr == nil {
		finalNative, err = nativeWitness(file, after)
	}
	if joined := errors.Join(statErr, liveErr, err); joined != nil || int64(len(body)) != before.Size() || !stableInfo(before, after) || !stableInfo(after, live) || finalNative != initialNative {
		return nil, errors.Join(joined, fmt.Errorf("%s changed while reading", path))
	}
	return body, nil
}

func snapshotTree(ctx context.Context, rootName string, options snapshotOptions) (ret []string, retErr error) {
	initial, err := exactInventory(ctx, []string{rootName}, nil, options.inventory)
	if err != nil {
		return nil, err
	}
	displayRoot := filepath.Clean(rootName)
	root, err := os.OpenRoot(displayRoot)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	matchedExcludes := map[string]bool{}
	var total int64
	for _, record := range initial.records {
		rel, err := filepath.Rel(displayRoot, record.path)
		if err != nil {
			return nil, err
		}
		if rel == "." {
			continue
		}
		portable := filepath.ToSlash(rel)
		if options.excludes[portable] {
			matchedExcludes[portable] = true
			continue
		}
		if record.info.Mode().IsRegular() {
			size := record.info.Size()
			if size < 0 || size > options.maxFileBytes {
				return nil, fmt.Errorf("snapshot file %s size %d exceeds %d-byte per-file limit", record.path, size, options.maxFileBytes)
			}
			if size > options.maxTotalBytes-total {
				return nil, fmt.Errorf("snapshot files exceed %d-byte aggregate limit at %s", options.maxTotalBytes, record.path)
			}
			total += size
		}
	}
	for excluded := range options.excludes {
		if !matchedExcludes[excluded] {
			return nil, fmt.Errorf("snapshot exclusion %s does not identify an inventoried entry", excluded)
		}
	}
	first, err := snapshotContentPass(ctx, root, displayRoot, initial.records, options)
	if err != nil {
		return nil, err
	}
	second, err := snapshotContentPass(ctx, root, displayRoot, initial.records, options)
	if err != nil {
		return nil, err
	}
	if err := compareSnapshotPasses(first, second); err != nil {
		return nil, err
	}
	final, err := exactInventory(ctx, []string{rootName}, nil, options.inventory)
	if err != nil {
		return nil, err
	}
	if err := compareInventoryResults(initial, final); err != nil {
		return nil, fmt.Errorf("snapshot inventory changed while hashing: %w", err)
	}
	return first.lines, nil
}

func snapshotContentPass(ctx context.Context, root *os.Root, displayRoot string, records []inventoryRecord, options snapshotOptions) (snapshotPassResult, error) {
	result := snapshotPassResult{
		lines: make([]string, 0, len(records)),
		files: make([]snapshotFileRecord, 0, len(records)),
	}
	var outputBytes int64
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return snapshotPassResult{}, fmt.Errorf("snapshot timed out: %w", err)
		}
		rel, err := filepath.Rel(displayRoot, record.path)
		if err != nil {
			return snapshotPassResult{}, err
		}
		if rel == "." || options.excludes[filepath.ToSlash(rel)] {
			continue
		}
		portable := filepath.ToSlash(rel)
		if strings.ContainsAny(portable, "\r\n\t") {
			return snapshotPassResult{}, fmt.Errorf("snapshot path cannot be represented safely: %q", portable)
		}
		var line string
		switch {
		case record.info.Mode().IsRegular():
			digest, witness, err := hashSnapshotFile(ctx, root, rel, record.info)
			if err != nil {
				return snapshotPassResult{}, err
			}
			line = fmt.Sprintf("F\t%s\t%d\tsha256:%x", portable, record.info.Size(), digest)
			result.files = append(result.files, snapshotFileRecord{path: portable, digest: digest, witness: witness})
		case record.info.IsDir():
			line = "D\t" + portable
		case record.info.Mode()&os.ModeSymlink != 0:
			target, err := root.Readlink(rel)
			if err != nil {
				return snapshotPassResult{}, err
			}
			if strings.ContainsAny(target, "\r\n\t") {
				return snapshotPassResult{}, fmt.Errorf("snapshot symlink %s has an unrepresentable target", record.path)
			}
			current, err := root.Lstat(rel)
			if err != nil || !stableInfo(record.info, current) {
				return snapshotPassResult{}, errors.Join(err, fmt.Errorf("snapshot symlink %s changed while reading", record.path))
			}
			line = fmt.Sprintf("L\t%s\t%s", portable, target)
		default:
			line = "S\t" + portable
		}
		lineBytes := int64(len(line) + 1)
		if lineBytes > options.inventory.maxBytes-outputBytes {
			return snapshotPassResult{}, fmt.Errorf("snapshot output exceeds %d-byte limit at %s", options.inventory.maxBytes, portable)
		}
		outputBytes += lineBytes
		result.lines = append(result.lines, line)
	}
	sort.Strings(result.lines)
	return result, nil
}

func compareSnapshotPasses(first, second snapshotPassResult) error {
	if len(first.lines) != len(second.lines) || len(first.files) != len(second.files) {
		return fmt.Errorf("snapshot content changed between bounded passes")
	}
	for index, file := range first.files {
		other := second.files[index]
		if file.path != other.path || file.digest != other.digest || file.witness != other.witness {
			return fmt.Errorf("snapshot file %s changed between bounded content passes", file.path)
		}
	}
	for index, line := range first.lines {
		if second.lines[index] != line {
			return fmt.Errorf("snapshot content changed between bounded passes at record %d", index)
		}
	}
	return nil
}

func hashSnapshotFile(ctx context.Context, root *os.Root, rel string, expected os.FileInfo) (digest [sha256.Size]byte, witness string, retErr error) {
	before, err := root.Lstat(rel)
	if err != nil || !stableInfo(expected, before) || !before.Mode().IsRegular() {
		return digest, "", errors.Join(err, fmt.Errorf("snapshot file %s changed before hashing", rel))
	}
	file, err := root.Open(rel)
	if err != nil {
		return digest, "", err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil || !stableInfo(before, opened) {
		return digest, "", errors.Join(err, fmt.Errorf("snapshot file %s changed while opening", rel))
	}
	initialNative, err := nativeWitness(file, opened)
	if err != nil {
		return digest, "", err
	}
	if err := snapshotFilePoint(rel, "after-open"); err != nil {
		return digest, "", err
	}
	first := sha256.New()
	if err := copySnapshotFileExact(ctx, file, first, opened.Size(), rel); err != nil {
		return digest, "", err
	}
	if err := snapshotFilePoint(rel, "after-first-hash"); err != nil {
		return digest, "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return digest, "", err
	}
	second := sha256.New()
	if err := copySnapshotFileExact(ctx, file, second, opened.Size(), rel); err != nil {
		return digest, "", err
	}
	after, statErr := file.Stat()
	live, liveErr := root.Lstat(rel)
	var finalNative string
	if statErr == nil {
		finalNative, err = nativeWitness(file, after)
	}
	if joined := errors.Join(statErr, liveErr, err); joined != nil || !stableInfo(opened, after) || !stableInfo(after, live) || finalNative != initialNative || !bytes.Equal(first.Sum(nil), second.Sum(nil)) {
		return digest, "", errors.Join(joined, fmt.Errorf("snapshot file %s changed while hashing", rel))
	}
	copy(digest[:], first.Sum(nil))
	return digest, initialNative, nil
}

func copySnapshotFileExact(ctx context.Context, file *os.File, destination io.Writer, size int64, rel string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("snapshot timed out: %w", err)
	}
	written, err := io.CopyN(destination, file, size)
	if err != nil || written != size {
		return errors.Join(err, fmt.Errorf("snapshot file %s changed size while hashing", rel))
	}
	var extra [1]byte
	count, err := file.Read(extra[:])
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if count != 0 {
		return fmt.Errorf("snapshot file %s grew while hashing", rel)
	}
	return nil
}

func stableInfo(before, after os.FileInfo) bool {
	return before != nil && after != nil && os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}
