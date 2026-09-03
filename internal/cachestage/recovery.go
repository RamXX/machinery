// Package cachestage recovers private cache-local extraction stages left by
// an interrupted provisioner. Callers must hold their cache's process lock.
package cachestage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Recover removes every valid MkdirTemp stage carrying prefix. It validates
// the entire reserved inventory before removing anything, so a malformed,
// symlinked, special, or non-private residue fails closed deterministically.
func Recover(base, prefix string) error {
	return recoverTrees(base, prefix, recoveryHooks{})
}

type recoveryHooks struct {
	beforeRetire func(string) error
	afterRetire  func(string) error
	beforeRemove func(string) error
}

type recoveryTree struct {
	name       string
	retirement string
	witness    treeWitness
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
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
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
		stages = append(stages, recoveryTree{name: name, retirement: name + ".machinery-retire", witness: witness})
	}
	// readDir is sorted, but keep the mutation plan's order explicit.
	sort.Slice(stages, func(i, j int) bool { return stages[i].name < stages[j].name })
	for _, stage := range stages {
		if hooks.beforeRetire != nil {
			if err := hooks.beforeRetire(stage.name); err != nil {
				return err
			}
		}
		if info, err := root.Lstat(stage.retirement); err == nil {
			return fmt.Errorf("cache stage retirement path %s already exists (%s)", stage.retirement, info.Mode())
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := root.Rename(stage.name, stage.retirement); err != nil {
			return fmt.Errorf("atomically retire interrupted cache stage %s: %w", stage.name, err)
		}
		if err := syncStageDirectory(root, "."); err != nil {
			return fmt.Errorf("sync retired cache stage %s: %w", stage.name, err)
		}
		if hooks.afterRetire != nil {
			if err := hooks.afterRetire(stage.retirement); err != nil {
				return err
			}
		}
		rebased := rebaseTreeWitness(stage.witness, stage.name, stage.retirement)
		current, err := syncTreeWitness(root, stage.retirement)
		if err != nil {
			return fmt.Errorf("verify retired cache stage %s: %w; preserving retirement tree", stage.name, err)
		}
		if err := compareTreeWitness(rebased, current); err != nil {
			return fmt.Errorf("retired cache stage %s changed after validation: %w; preserving retirement tree", stage.name, err)
		}
		if err := removeTreeWitness(root, rebased, hooks); err != nil {
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
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
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
	for _, stage := range stages {
		if err := removeFileWitness(root, stage, hooks); err != nil {
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

func removeTreeWitness(root *os.Root, witness treeWitness, hooks recoveryHooks) error {
	for i := len(witness.entries) - 1; i >= 0; i-- {
		entry := witness.entries[i]
		if entry.isDir {
			if err := removeDirectoryWitness(root, entry, hooks); err != nil {
				return err
			}
			continue
		}
		if err := removeFileWitness(root, entry, hooks); err != nil {
			return err
		}
	}
	return nil
}

func removeFileWitness(root *os.Root, want treeWitnessEntry, hooks recoveryHooks) error {
	if err := revalidateFileWitness(root, want); err != nil {
		return err
	}
	if hooks.beforeRemove != nil {
		if err := hooks.beforeRemove(want.path); err != nil {
			return err
		}
	}
	if err := revalidateFileWitness(root, want); err != nil {
		return fmt.Errorf("cache stage file changed at deletion boundary: %w", err)
	}
	if err := root.Remove(want.path); err != nil {
		return err
	}
	return syncStageDirectory(root, filepath.Dir(want.path))
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

func removeDirectoryWitness(root *os.Root, want treeWitnessEntry, hooks recoveryHooks) error {
	if err := revalidateEmptyDirectoryWitness(root, want); err != nil {
		return err
	}
	if hooks.beforeRemove != nil {
		if err := hooks.beforeRemove(want.path); err != nil {
			return err
		}
	}
	if err := revalidateEmptyDirectoryWitness(root, want); err != nil {
		return fmt.Errorf("cache stage directory changed at deletion boundary: %w", err)
	}
	if err := root.Remove(want.path); err != nil {
		return err
	}
	return syncStageDirectory(root, filepath.Dir(want.path))
}

func revalidateEmptyDirectoryWitness(root *os.Root, want treeWitnessEntry) error {
	info, err := root.Lstat(want.path)
	if err != nil {
		return err
	}
	if want.info == nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || !privateStageMode(info.Mode()) ||
		!os.SameFile(want.info, info) || want.mode != info.Mode() {
		return fmt.Errorf("%s no longer matches its validated directory identity; preserving it", filepath.ToSlash(want.path))
	}
	entries, err := readDir(root, want.path)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("%s was populated during cleanup; preserving it", filepath.ToSlash(want.path))
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
	dir, err := root.Open(rel)
	if err != nil {
		return nil, err
	}
	entries, readErr := dir.ReadDir(-1)
	closeErr := dir.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func validateTree(root *os.Root, dir string) error {
	entries, err := readDir(root, dir)
	if err != nil {
		return err
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
			if !privateStageMode(info.Mode()) {
				return fmt.Errorf("%s is not private", filepath.ToSlash(child))
			}
			if err := validateTree(root, child); err != nil {
				return err
			}
		case info.Mode().IsRegular():
			if !privateStageMode(info.Mode()) {
				return fmt.Errorf("%s is not private", filepath.ToSlash(child))
			}
		default:
			return fmt.Errorf("%s is a special entry (%s)", filepath.ToSlash(child), info.Mode())
		}
	}
	return nil
}
