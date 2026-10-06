package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/gitcontrol"
	"github.com/RamXX/machinery/internal/processcontrol"
)

// Git selects the mutation boundary without walking local ignored trees. Each
// bounded pass re-reads the selection so additions and removals still fail closed.
func gitInventoryPass(ctx context.Context, rootName string, options inventoryOptions) (ret inventoryResult, retErr error) {
	if options.maxEntries <= 0 || options.maxDepth < 0 || options.maxBytes <= 0 {
		return inventoryResult{}, fmt.Errorf("inventory limits must be positive (depth may be zero)")
	}
	if err := ctx.Err(); err != nil {
		return inventoryResult{}, fmt.Errorf("inventory traversal timed out: %w", err)
	}
	display := filepath.Clean(rootName)
	info, err := os.Lstat(display)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return inventoryResult{}, errors.Join(err, fmt.Errorf("inventory root %s must be a real directory", display))
	}
	root, err := os.OpenRoot(display)
	if err != nil {
		return inventoryResult{}, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	cmd := exec.CommandContext(ctx, "git", "-C", display, "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--")
	cmd.Env = gitcontrol.Environment(os.Environ())
	stdout, stderr, err := processcontrol.RunCapturedStreamLimits(ctx, cmd, int(min(options.maxBytes, 64<<20)), 4096)
	if err != nil || stderr != "" {
		return inventoryResult{}, errors.Join(err, fmt.Errorf("select Git snapshot paths: %s", stderr))
	}
	if stdout != "" && !strings.HasSuffix(stdout, "\x00") {
		return inventoryResult{}, fmt.Errorf("git snapshot path list is not NUL-terminated")
	}
	state := &inventoryState{options: options}
	seen := map[string]bool{}
	entries := 0
	for remaining := stdout; remaining != ""; {
		path, rest, _ := strings.Cut(remaining, "\x00")
		remaining = rest
		entries++
		if entries > options.maxEntries {
			return inventoryResult{}, fmt.Errorf("inventory exceeds %d-entry limit", options.maxEntries)
		}
		if path == "" {
			continue
		}
		// Git emits an untracked nested repository as a directory ending in '/'.
		path = strings.TrimSuffix(path, "/")
		if !fs.ValidPath(path) || strings.ContainsAny(path, "\r\n\t") {
			return inventoryResult{}, fmt.Errorf("snapshot path cannot be represented safely: %q", path)
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		if err := state.checkContext(ctx); err != nil {
			return inventoryResult{}, err
		}
		omitted, err := omitGitPath(root, path, options)
		if err != nil {
			return inventoryResult{}, err
		}
		if omitted {
			continue
		}
		if strings.Count(path, "/") >= options.maxDepth {
			return inventoryResult{}, fmt.Errorf("inventory exceeds %d-level depth limit at %s", options.maxDepth, path)
		}
		info, err := root.Lstat(filepath.FromSlash(path))
		if err != nil {
			return inventoryResult{}, err
		}
		if err := state.visit(filepath.Join(display, filepath.FromSlash(path)), true, info); err != nil {
			return inventoryResult{}, err
		}
	}
	sort.Strings(state.paths)
	sort.Slice(state.records, func(i, j int) bool { return state.records[i].path < state.records[j].path })
	return inventoryResult{paths: state.paths, records: state.records}, nil
}

func omitGitPath(root *os.Root, path string, options inventoryOptions) (bool, error) {
	parts := strings.Split(path, "/")
	for index := range parts {
		prefix := strings.Join(parts[:index+1], "/")
		if options.prune[prefix] {
			return true, nil
		}
		info, err := root.Lstat(filepath.FromSlash(prefix))
		if err != nil {
			return false, err
		}
		if info.IsDir() {
			marker, err := root.Lstat(filepath.Join(filepath.FromSlash(prefix), ".git"))
			if err == nil && (marker.Mode().IsRegular() || marker.IsDir()) {
				return true, nil
			}
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return false, err
			}
		} else if index < len(parts)-1 {
			return false, fmt.Errorf("git snapshot parent %s must be a real directory", prefix)
		}
	}
	return false, nil
}
