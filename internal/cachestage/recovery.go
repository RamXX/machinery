// Package cachestage recovers private cache-local extraction stages left by
// an interrupted provisioner. Callers must hold their cache's process lock.
package cachestage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/fsatomic"
)

// Recover removes every valid MkdirTemp stage carrying prefix. It validates
// the entire reserved inventory before removing anything, so a malformed,
// symlinked, special, or non-private residue fails closed deterministically.
func Recover(base, prefix string) error {
	return recoverTrees(base, prefix, recoveryHooks{})
}

type recoveryHooks struct {
	beforeRetire            func(string) error
	afterRetire             func(*os.Root, string) error
	beforeRemove            func(string) error
	beforePrivateRemove     func(*os.Root, string) error
	beforePrivateTreeRemove func(*os.Root, string) error
	quarantine              func(*os.Root, string, string) (*fsatomic.Quarantined, error)
}

type recoveryTree struct {
	name    string
	witness treeWitness
}

type recoveryQuarantine struct {
	handle  *fsatomic.Quarantined
	witness treeWitness
	empty   bool
}

type recoveryFileQuarantine struct {
	handle  *fsatomic.Quarantined
	witness treeWitnessEntry
}

func recoverTrees(base, prefix string, hooks recoveryHooks) (retErr error) {
	if prefix == "" || strings.ContainsAny(prefix, `/\\`) {
		return fmt.Errorf("cache stage prefix %q must be one path-segment prefix", prefix)
	}
	before, err := os.Lstat(base)
	if err != nil {
		return err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return fmt.Errorf("cache stage root %s must be a real directory", base)
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	inside, err := root.Lstat(".")
	if err != nil || !os.SameFile(before, inside) {
		return errors.Join(err, fmt.Errorf("cache stage root changed identity while opening"))
	}
	entries, err := readDir(root, ".")
	if err != nil {
		return err
	}
	var stages []recoveryTree
	var quarantines []recoveryQuarantine
	defer func() {
		for _, quarantine := range quarantines {
			retErr = errors.Join(retErr, quarantine.handle.Close())
		}
	}()
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if strings.HasPrefix(name, prefix+"delete-") {
			info, err := root.Lstat(name)
			if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !privateStageMode(info.Mode()) {
				return errors.Join(err, fmt.Errorf("reserved cache quarantine %s must be a private real directory", name))
			}
			handle, err := fsatomic.ResumeQuarantine(root, name, "")
			if err != nil {
				return fmt.Errorf("resume cache quarantine %s: %w", name, err)
			}
			source := handle.Source()
			if legacySource := strings.TrimSuffix(source, ".machinery-retire"); legacySource != source {
				source = legacySource
			}
			if !strings.HasPrefix(source, prefix) || !validRandomSuffix(strings.TrimPrefix(source, prefix)) {
				_ = handle.Close()
				return fmt.Errorf("cache quarantine %s records unsafe source %q", name, handle.Source())
			}
			object, err := handle.Root().Lstat(handle.Name())
			if errors.Is(err, os.ErrNotExist) {
				if err := validateCacheQuarantineInventory(handle.Root(), false); err != nil {
					_ = handle.Close()
					return fmt.Errorf("cache quarantine %s: %w", name, err)
				}
				quarantines = append(quarantines, recoveryQuarantine{handle: handle, empty: true})
				continue
			}
			if err != nil || !object.IsDir() || object.Mode()&os.ModeSymlink != 0 || !privateStageMode(object.Mode()) {
				_ = handle.Close()
				return errors.Join(err, fmt.Errorf("cache quarantine %s object must be a private real directory", name))
			}
			if err := validateCacheQuarantineInventory(handle.Root(), true); err != nil {
				_ = handle.Close()
				return fmt.Errorf("cache quarantine %s: %w", name, err)
			}
			witness, err := syncTreeWitness(handle.Root(), handle.Name())
			if err != nil {
				_ = handle.Close()
				return fmt.Errorf("witness cache quarantine %s: %w", name, err)
			}
			quarantines = append(quarantines, recoveryQuarantine{handle: handle, witness: witness})
			continue
		}
		if !validRandomSuffix(strings.TrimPrefix(name, prefix)) {
			return fmt.Errorf("reserved cache stage entry %q has an unsafe name", name)
		}
		info, err := root.Lstat(name)
		if err != nil {
			return fmt.Errorf("inspect reserved cache stage %s: %w", name, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || !privateStageMode(info.Mode()) {
			return fmt.Errorf("reserved cache stage %s must be a private real directory", name)
		}
		witness, err := syncTreeWitness(root, name)
		if err != nil {
			return fmt.Errorf("witness reserved cache stage %s: %w", name, err)
		}
		stages = append(stages, recoveryTree{name: name, witness: witness})
	}
	for index := range quarantines {
		quarantine := &quarantines[index]
		if quarantine.empty {
			if err := quarantine.handle.FinishEmpty(); err != nil {
				return err
			}
			continue
		}
		current, err := syncTreeWitness(quarantine.handle.Root(), quarantine.handle.Name())
		if err != nil || compareTreeWitness(quarantine.witness, current) != nil {
			return errors.Join(err, fmt.Errorf("cache quarantine changed during recovery; preserving it"))
		}
		if hooks.beforePrivateTreeRemove != nil {
			if err := hooks.beforePrivateTreeRemove(quarantine.handle.Root(), quarantine.handle.Name()); err != nil {
				return err
			}
		}
		current, err = syncTreeWitness(quarantine.handle.Root(), quarantine.handle.Name())
		if err != nil || compareTreeWitness(quarantine.witness, current) != nil {
			return errors.Join(err, fmt.Errorf("cache quarantine changed at private deletion boundary; preserving it"))
		}
		if err := quarantine.handle.RemoveAll(); err != nil {
			return err
		}
	}
	// readDir is sorted, but keep the mutation plan's order explicit.
	sort.Slice(stages, func(i, j int) bool { return stages[i].name < stages[j].name })
	for _, stage := range stages {
		if hooks.beforeRetire != nil {
			if err := hooks.beforeRetire(stage.name); err != nil {
				return err
			}
		}
		quarantineTree := fsatomic.Quarantine
		if hooks.quarantine != nil {
			quarantineTree = hooks.quarantine
		}
		quarantined, err := quarantineTree(root, stage.name, prefix+"delete-")
		if err != nil {
			return fmt.Errorf("atomically quarantine interrupted cache stage %s: %w", stage.name, err)
		}
		if hooks.afterRetire != nil {
			if err := hooks.afterRetire(quarantined.Root(), quarantined.Name()); err != nil {
				return preserveCacheQuarantine(quarantined, err)
			}
		}
		if err := validateCacheQuarantineInventory(quarantined.Root(), true); err != nil {
			return preserveCacheQuarantine(quarantined, err)
		}
		rebased := rebaseTreeWitness(stage.witness, stage.name, quarantined.Name())
		current, err := syncTreeWitness(quarantined.Root(), quarantined.Name())
		if err != nil {
			return preserveCacheQuarantine(quarantined, fmt.Errorf("verify quarantined cache stage %s: %w", stage.name, err))
		}
		if err := compareTreeWitness(rebased, current); err != nil {
			return preserveCacheQuarantine(quarantined, fmt.Errorf("quarantined cache stage %s changed after validation: %w", stage.name, err))
		}
		if hooks.beforeRemove != nil {
			for index := len(rebased.entries) - 1; index >= 0; index-- {
				if err := hooks.beforeRemove(rebased.entries[index].path); err != nil {
					return preserveCacheQuarantine(quarantined, err)
				}
			}
		}
		current, err = syncTreeWitness(quarantined.Root(), quarantined.Name())
		if err != nil {
			return preserveCacheQuarantine(quarantined, fmt.Errorf("revalidate quarantined cache stage %s at deletion boundary: %w", stage.name, err))
		}
		if err := compareTreeWitness(rebased, current); err != nil {
			return preserveCacheQuarantine(quarantined, fmt.Errorf("quarantined cache stage %s changed at deletion boundary: %w", stage.name, err))
		}
		if hooks.beforePrivateRemove != nil {
			if err := hooks.beforePrivateRemove(quarantined.Root(), quarantined.Name()); err != nil {
				return preserveCacheQuarantine(quarantined, err)
			}
		}
		if err := validateCacheQuarantineInventory(quarantined.Root(), true); err != nil {
			return preserveCacheQuarantine(quarantined, err)
		}
		privateCurrent, err := syncTreeWitness(quarantined.Root(), quarantined.Name())
		if err != nil {
			return preserveCacheQuarantine(quarantined, fmt.Errorf("verify private cache deletion authority: %w", err))
		}
		if err := compareTreeWitness(rebased, privateCurrent); err != nil {
			return preserveCacheQuarantine(quarantined, fmt.Errorf("private cache deletion authority changed: %w", err))
		}
		if hooks.beforePrivateTreeRemove != nil {
			if err := hooks.beforePrivateTreeRemove(quarantined.Root(), quarantined.Name()); err != nil {
				return preserveCacheQuarantine(quarantined, err)
			}
		}
		privateCurrent, err = syncTreeWitness(quarantined.Root(), quarantined.Name())
		if err != nil || compareTreeWitness(rebased, privateCurrent) != nil {
			return preserveCacheQuarantine(quarantined, errors.Join(err, fmt.Errorf("private cache deletion authority changed at removal boundary")))
		}
		if err := quarantined.RemoveAll(); err != nil {
			return fmt.Errorf("conditionally remove interrupted cache stage %s: %w", stage.name, err)
		}
	}
	if len(stages) > 0 {
		if err := syncStageDirectory(root, "."); err != nil {
			return fmt.Errorf("make interrupted cache-stage cleanup durable: %w", err)
		}
	}
	return nil
}

func validateCacheQuarantineInventory(root *os.Root, objectExpected bool) error {
	entries, err := readDirBounded(root, ".", 1)
	if err != nil {
		return err
	}
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

func preserveCacheQuarantine(quarantined *fsatomic.Quarantined, cause error) error {
	return errors.Join(cause, fmt.Errorf("preserving private cache deletion authority"), quarantined.Close())
}

// RecoverFiles removes valid private CreateTemp files carrying prefix. It is
// the file counterpart to Recover and is intended for cache downloads whose
// advisory lock is held by the caller. Every reserved entry is validated
// before any removal, so a symlink, special file, directory, or malformed
// suffix fails closed without partially cleaning the inventory.
func RecoverFiles(base, prefix string) error {
	return recoverFiles(base, prefix, recoveryHooks{})
}

func recoverFiles(base, prefix string, hooks recoveryHooks) (retErr error) {
	if prefix == "" || strings.ContainsAny(prefix, `/\\`) {
		return fmt.Errorf("cache file-stage prefix %q must be one path-segment prefix", prefix)
	}
	before, err := os.Lstat(base)
	if err != nil {
		return err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return fmt.Errorf("cache file-stage root %s must be a real directory", base)
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	inside, err := root.Lstat(".")
	if err != nil || !os.SameFile(before, inside) {
		return errors.Join(err, fmt.Errorf("cache file-stage root changed identity while opening"))
	}
	entries, err := readDir(root, ".")
	if err != nil {
		return err
	}
	var stages []treeWitnessEntry
	var quarantines []recoveryFileQuarantine
	defer func() {
		for _, quarantine := range quarantines {
			retErr = errors.Join(retErr, quarantine.handle.Close())
		}
	}()
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if strings.HasPrefix(name, prefix+"delete-") {
			handle, err := fsatomic.ResumeQuarantine(root, name, "")
			if err != nil {
				return err
			}
			if !strings.HasPrefix(handle.Source(), prefix) || !validRandomSuffix(strings.TrimPrefix(handle.Source(), prefix)) {
				_ = handle.Close()
				return fmt.Errorf("cache file quarantine %s records unsafe source %q", name, handle.Source())
			}
			if _, err := handle.Root().Lstat(handle.Name()); errors.Is(err, os.ErrNotExist) {
				if err := validateCacheQuarantineInventory(handle.Root(), false); err != nil {
					_ = handle.Close()
					return err
				}
				if err := handle.FinishEmpty(); err != nil {
					return err
				}
				continue
			} else if err != nil {
				_ = handle.Close()
				return err
			}
			if err := validateCacheQuarantineInventory(handle.Root(), true); err != nil {
				_ = handle.Close()
				return err
			}
			info, err := handle.Root().Lstat(handle.Name())
			if err != nil {
				_ = handle.Close()
				return err
			}
			witness, err := syncFileWitness(handle.Root(), handle.Name(), info)
			if err != nil {
				_ = handle.Close()
				return err
			}
			if err := revalidateFileWitness(handle.Root(), witness); err != nil {
				_ = handle.Close()
				return err
			}
			quarantines = append(quarantines, recoveryFileQuarantine{handle: handle, witness: witness})
			continue
		}
		if !validRandomSuffix(strings.TrimPrefix(name, prefix)) {
			return fmt.Errorf("reserved cache file-stage entry %q has an unsafe name", name)
		}
		info, err := root.Lstat(name)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || !privateStageMode(info.Mode()) {
			return fmt.Errorf("reserved cache file-stage %s must be a private regular non-symlink file", name)
		}
		witness, err := syncFileWitness(root, name, info)
		if err != nil {
			return fmt.Errorf("witness reserved cache file-stage %s: %w", name, err)
		}
		stages = append(stages, witness)
	}
	for _, quarantine := range quarantines {
		if err := validateCacheQuarantineInventory(quarantine.handle.Root(), true); err != nil {
			return err
		}
		if err := revalidateFileWitness(quarantine.handle.Root(), quarantine.witness); err != nil {
			return fmt.Errorf("cache file quarantine changed at deletion boundary; preserving it: %w", err)
		}
		if err := quarantine.handle.Remove(); err != nil {
			return err
		}
	}
	for _, stage := range stages {
		if hooks.beforeRemove != nil {
			if err := hooks.beforeRemove(stage.path); err != nil {
				return err
			}
		}
		if err := revalidateFileWitness(root, stage); err != nil {
			return fmt.Errorf("cache stage file changed at deletion boundary: %w", err)
		}
		quarantined, err := fsatomic.Quarantine(root, stage.path, prefix+"delete-")
		if err != nil {
			return err
		}
		if hooks.beforePrivateRemove != nil {
			if err := hooks.beforePrivateRemove(quarantined.Root(), quarantined.Name()); err != nil {
				return preserveCacheQuarantine(quarantined, err)
			}
		}
		if err := validateCacheQuarantineInventory(quarantined.Root(), true); err != nil {
			return preserveCacheQuarantine(quarantined, err)
		}
		info, err := quarantined.Root().Lstat(quarantined.Name())
		if err != nil {
			return preserveCacheQuarantine(quarantined, err)
		}
		current, err := syncFileWitness(quarantined.Root(), quarantined.Name(), info)
		if err != nil || stage.hash != current.hash || stage.mode != current.mode || stage.info == nil || current.info == nil ||
			!os.SameFile(stage.info, current.info) || stage.info.Size() != current.info.Size() || !stage.info.ModTime().Equal(current.info.ModTime()) {
			return preserveCacheQuarantine(quarantined, errors.Join(err, fmt.Errorf("private cache file deletion authority changed; preserving it")))
		}
		if err := quarantined.Remove(); err != nil {
			return fmt.Errorf("conditionally remove cache file-stage %s: %w", stage.path, err)
		}
	}
	if len(stages) > 0 {
		if err := syncStageDirectory(root, "."); err != nil {
			return err
		}
	}
	return nil
}

func revalidateFileWitness(root *os.Root, want treeWitnessEntry) error {
	info, err := root.Lstat(want.path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || !privateStageMode(info.Mode()) {
		return fmt.Errorf("%s is no longer a private regular file", filepath.ToSlash(want.path))
	}
	got, err := syncFileWitness(root, want.path, info)
	if err != nil {
		return err
	}
	if want.info == nil || !os.SameFile(want.info, got.info) || want.mode != got.mode || want.info.Size() != got.info.Size() ||
		!want.info.ModTime().Equal(got.info.ModTime()) || want.hash != got.hash {
		return fmt.Errorf("%s no longer matches its exact validated identity and content; preserving it", filepath.ToSlash(want.path))
	}
	return nil
}

func validRandomSuffix(suffix string) bool {
	if suffix == "" || len(suffix) > 10 {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func readDir(root *os.Root, rel string) ([]os.DirEntry, error) {
	return readDirBounded(root, rel, cacheStageMaxEntries)
}

func readDirBounded(root *os.Root, rel string, maxEntries int) ([]os.DirEntry, error) {
	if maxEntries <= 0 {
		return nil, fmt.Errorf("cache directory entry limit must be positive")
	}
	before, err := root.Lstat(rel)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, fmt.Errorf("cache directory %s must be a real directory", filepath.ToSlash(rel))
	}
	dir, err := root.Open(rel)
	if err != nil {
		return nil, err
	}
	opened, err := dir.Stat()
	if err != nil || !opened.IsDir() || !os.SameFile(before, opened) || before.Mode() != opened.Mode() {
		return nil, errors.Join(err, fmt.Errorf("cache directory %s changed while opening", filepath.ToSlash(rel)), dir.Close())
	}
	capacity := min(cacheStageDirectoryBatch, maxEntries)
	entries := make([]os.DirEntry, 0, capacity)
	var readErr error
	for {
		batch, err := dir.ReadDir(cacheStageDirectoryBatch)
		if len(batch) > maxEntries-len(entries) {
			readErr = fmt.Errorf("cache directory %s exceeds %d-entry limit", filepath.ToSlash(rel), maxEntries)
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
	pathAfter, pathErr := root.Lstat(rel)
	closeErr := dir.Close()
	if err := errors.Join(readErr, statErr, pathErr, closeErr); err != nil {
		return nil, err
	}
	if pathAfter.Mode()&os.ModeSymlink != 0 || !pathAfter.IsDir() ||
		!os.SameFile(opened, after) || !os.SameFile(opened, pathAfter) || opened.Mode() != after.Mode() || opened.Mode() != pathAfter.Mode() ||
		!opened.ModTime().Equal(after.ModTime()) || !opened.ModTime().Equal(pathAfter.ModTime()) {
		return nil, fmt.Errorf("cache directory %s changed while being inventoried", filepath.ToSlash(rel))
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

type cacheTreeBudget struct {
	entries    int
	bytes      int64
	maxEntries int
	maxBytes   int64
}

func (budget *cacheTreeBudget) add(info os.FileInfo, display string) error {
	budget.entries++
	if budget.entries > budget.maxEntries {
		return fmt.Errorf("cache tree exceeds %d-entry limit at %s", budget.maxEntries, display)
	}
	if info.Mode().IsRegular() {
		if info.Size() < 0 || info.Size() > cacheStageMaxFileBytes {
			return fmt.Errorf("cache tree file %s has size %d, exceeding %d-byte limit", display, info.Size(), cacheStageMaxFileBytes)
		}
		if info.Size() > budget.maxBytes-budget.bytes {
			return fmt.Errorf("cache tree exceeds %d-byte aggregate limit at %s", budget.maxBytes, display)
		}
		budget.bytes += info.Size()
	}
	return nil
}

func validateTree(root *os.Root, dir string) error {
	return validateTreeBounded(root, dir, cacheStageMaxEntries, cacheStageMaxDepth, cacheStageMaxTotalBytes)
}

func validateTreeBounded(root *os.Root, dir string, maxEntries, maxDepth int, maxBytes int64) error {
	if maxEntries <= 0 || maxDepth < 0 || maxBytes < 0 {
		return fmt.Errorf("cache tree limits must be non-negative with a positive entry limit")
	}
	budget := cacheTreeBudget{maxEntries: maxEntries, maxBytes: maxBytes}
	return validateTreeWithBudget(root, dir, &budget, 0, maxDepth)
}

func validateTreeWithBudget(root *os.Root, dir string, budget *cacheTreeBudget, depth, maxDepth int) error {
	if depth > maxDepth {
		return fmt.Errorf("cache tree exceeds %d-directory depth limit at %s", maxDepth, filepath.ToSlash(dir))
	}
	dirInfo, err := root.Lstat(dir)
	if err != nil || dirInfo.Mode()&os.ModeSymlink != 0 || !dirInfo.IsDir() || !privateStageMode(dirInfo.Mode()) {
		return errors.Join(err, fmt.Errorf("%s must be a private real directory", filepath.ToSlash(dir)))
	}
	if err := budget.add(dirInfo, filepath.ToSlash(dir)); err != nil {
		return err
	}
	remaining := budget.maxEntries - budget.entries
	readLimit := max(remaining, 1)
	entries, err := readDirBounded(root, dir, readLimit)
	if err != nil {
		return err
	}
	if len(entries) > remaining {
		return fmt.Errorf("cache tree exceeds %d-entry limit at %s", budget.maxEntries, filepath.ToSlash(dir))
	}
	for _, entry := range entries {
		child := filepath.Join(dir, entry.Name())
		info, err := root.Lstat(child)
		if err != nil {
			return err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("%s is a symlink", filepath.ToSlash(child))
		case info.IsDir():
			if err := validateTreeWithBudget(root, child, budget, depth+1, maxDepth); err != nil {
				return err
			}
		case info.Mode().IsRegular():
			if !privateStageMode(info.Mode()) {
				return fmt.Errorf("%s is not private", filepath.ToSlash(child))
			}
			if err := budget.add(info, filepath.ToSlash(child)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s is a special entry (%s)", filepath.ToSlash(child), info.Mode())
		}
	}
	return nil
}
