//go:build !windows

package install

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func privateFilePermissionsOK(info os.FileInfo) bool { return info.Mode().Perm()&0o077 == 0 }
func installDirectoryOwnedByCurrentUser(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid())
}
func syncDirectoryPath(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), closeInstallFile(f))
}
func syncRootDirectoryFile(f *os.File, _ string) error { return f.Sync() }
func stableInstallDirIdentity(_ string, info os.FileInfo) (string, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", fmt.Errorf("directory has no native stat identity")
	}
	return fmt.Sprintf("%d:%d", stat.Dev, stat.Ino), nil
}
