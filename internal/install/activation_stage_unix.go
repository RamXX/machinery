//go:build !windows

package install

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const installActivationDir = ".machinery-install-activation"

func activationStagingPath() (string, error) {
	journal, _, err := installJournalPaths()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(journal), installActivationDir), nil
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
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return syncDir(filepath.Dir(dir))
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
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return fail(err)
	}
	image := filepath.Join(dir, "machinery")
	out, err := os.OpenFile(image, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return fail(err)
	}
	if _, err := io.Copy(out, source); err != nil {
		return fail(errors.Join(err, closeInstallFile(out)))
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
