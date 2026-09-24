package checker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WriteFactsDir writes the design's facts to dir as FactsFiles renders them.
// The files are staged in a sibling temporary directory and renamed into
// place, so a reader sees the previous directory or the complete new one,
// never a half-written set. An existing dir is replaced only when it holds
// nothing but *.facts files and relations.txt; anything else in it is a
// refusal, so a mistyped path can never delete unrelated work.
func WriteFactsDir(dir string, facts *DesignFacts) (retErr error) {
	files, err := facts.FactsFiles()
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	parent, base := filepath.Dir(abs), filepath.Base(abs)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	existing, err := factsDirReplaceable(abs)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, "."+base+".tmp-")
	if err != nil {
		return err
	}
	defer func() {
		if retErr != nil {
			_ = os.RemoveAll(tmp)
		}
	}()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := writeSyncedFile(filepath.Join(tmp, name), files[name]); err != nil {
			return err
		}
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		return err
	}
	if err := syncDir(tmp); err != nil {
		return err
	}
	if !existing {
		if err := os.Rename(tmp, abs); err != nil {
			return err
		}
		return syncDir(parent)
	}
	old := tmp + ".old"
	if err := os.Rename(abs, old); err != nil {
		return err
	}
	if err := os.Rename(tmp, abs); err != nil {
		return errors.Join(err, os.Rename(old, abs))
	}
	if err := syncDir(parent); err != nil {
		return err
	}
	return os.RemoveAll(old)
}

// factsDirReplaceable reports whether dir exists, and fails when it exists
// but is not a directory of facts output.
func factsDirReplaceable(dir string) (bool, error) {
	info, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return false, fmt.Errorf("refusing to replace %s: it is not a directory", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if !e.Type().IsRegular() || (e.Name() != RelationsIndexFile && !strings.HasSuffix(e.Name(), ".facts")) {
			return false, fmt.Errorf("refusing to replace %s: it holds %s, which is not facts output", dir, e.Name())
		}
	}
	return true, nil
}

func writeSyncedFile(path string, data []byte) (retErr error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, f.Close()) }()
	if _, err := f.Write(data); err != nil {
		return err
	}
	return f.Sync()
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	// Directory fsync is unsupported on some platforms (Windows); the rename
	// itself is still the commit point there.
	_ = d.Sync()
	return d.Close()
}
