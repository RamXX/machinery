//go:build !windows

package install

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func activationStagingPath() (string, error) {
	journal, _, err := installJournalPaths()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(journal), installActivationDir), nil
}

func activationRecoveryPending() (bool, error) {
	path, err := activationStagingPath()
	if err != nil {
		return false, fmt.Errorf("resolve executable activation path: %w", err)
	}
	if _, err := os.Lstat(path); err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("inspect executable activation path %s: %w", path, err)
	}
	return false, nil
}

func cleanupActivationExecutable() error {
	dir, err := activationStagingPath()
	if err != nil {
		return err
	}
	info, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("activation staging path %s is not a real directory", dir)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	if err := durableRemoveAll(dir); err != nil {
		return err
	}
	return nil
}

func stageActivationExecutable(_ string, source *os.File, identity string) (string, error) {
	if err := cleanupActivationExecutable(); err != nil {
		return "", err
	}
	dir, err := activationStagingPath()
	if err != nil {
		return "", err
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		return "", err
	}
	fail := func(primary error) (string, error) {
		return "", errors.Join(primary, cleanupActivationExecutable())
	}
	if err := syncDir(filepath.Dir(dir)); err != nil {
		return fail(err)
	}
	info, err := source.Stat()
	if err != nil {
		return fail(err)
	}
	if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > installArtifactMaxFileBytes {
		return fail(fmt.Errorf("activation executable exceeds fixed file bounds"))
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return fail(err)
	}
	image := filepath.Join(dir, "machinery")
	out, err := os.OpenFile(image, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return fail(err)
	}
	written, copyErr := io.Copy(out, io.LimitReader(source, info.Size()+1))
	after, statErr := source.Stat()
	if err := errors.Join(copyErr, statErr); err != nil {
		return fail(errors.Join(err, closeInstallFile(out)))
	}
	if written != info.Size() || !sameInstallArtifactInfo(info, after) {
		return fail(errors.Join(fmt.Errorf("activation executable changed while staging"), closeInstallFile(out)))
	}
	if err := out.Chmod(info.Mode().Perm()); err != nil {
		return fail(errors.Join(err, closeInstallFile(out)))
	}
	if err := out.Sync(); err != nil {
		return fail(errors.Join(err, closeInstallFile(out)))
	}
	if err := closeInstallFile(out); err != nil {
		return fail(err)
	}
	if err := syncDir(dir); err != nil {
		return fail(err)
	}
	stagedIdentity, err := activationExecutableIdentity(image)
	if err != nil || stagedIdentity != identity {
		return fail(errors.Join(err, fmt.Errorf("staged activation image identity does not match restored executable")))
	}
	// Once complete, make accidental mutation impossible for ordinary writes.
	// The next startup owns cleanup under the operation lock and restores 0700.
	if err := os.Chmod(dir, 0o500); err != nil {
		return fail(err)
	}
	if err := syncDir(dir); err != nil {
		return fail(err)
	}
	return image, nil
}

func validateActivationExecutablePath(recovery *ActivationRecoveryError) error {
	if recovery.activationPath == "" {
		return fmt.Errorf("activation image path is missing")
	}
	identity, err := activationExecutableIdentity(recovery.activationPath)
	if err != nil {
		return err
	}
	if identity != recovery.Identity {
		return fmt.Errorf("activation image identity changed before re-exec")
	}
	return nil
}
