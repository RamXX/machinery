//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package artifactset

import (
	"fmt"
	"os"
)

func txOpenRoot(path string) (*os.Root, *os.File, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("inspect artifact root before opening: %w", err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, nil, fmt.Errorf("artifact root must remain a real directory while opening")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open artifact directory root: %w", err)
	}
	inside, err := root.Lstat(".")
	if err != nil || !os.SameFile(before, inside) {
		_ = root.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("inspect opened artifact directory root: %w", err)
		}
		return nil, nil, fmt.Errorf("artifact directory changed identity while opening root")
	}
	syncFile, err := root.Open(".")
	if err != nil {
		_ = root.Close()
		return nil, nil, fmt.Errorf("open rooted directory sync handle: %w", err)
	}
	return root, syncFile, nil
}

func txOpenSyncRoot(root *os.Root) (*os.File, error) {
	inside, err := root.Lstat(".")
	if err != nil || !inside.IsDir() {
		return nil, fmt.Errorf("inspect retained artifact root: %w", err)
	}
	syncFile, err := root.Open(".")
	if err != nil {
		return nil, fmt.Errorf("open retained rooted directory sync handle: %w", err)
	}
	opened, statErr := syncFile.Stat()
	if statErr != nil || !os.SameFile(inside, opened) {
		_ = syncFile.Close()
		return nil, fmt.Errorf("retained artifact root changed while opening sync handle")
	}
	return syncFile, nil
}

func txSyncHeld(syncFile *os.File) error {
	if err := syncFile.Sync(); err != nil {
		return fmt.Errorf("sync rooted directory handle: %w", err)
	}
	return nil
}
