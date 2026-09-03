//go:build !windows

package dirscan

import (
	"fmt"
	"os"
	"reflect"
)

// directoryChangeID returns the inode-change timestamp. Directory entry
// creation, deletion, and rename update ctime even when a filesystem's mtime
// granularity is too coarse to distinguish an ABA mutation.
func directoryChangeID(_ *os.File, info os.FileInfo) (string, error) {
	if info == nil || info.Sys() == nil {
		return "", fmt.Errorf("directory has no native stat data")
	}
	value := reflect.ValueOf(info.Sys())
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return "", fmt.Errorf("directory has nil native stat data")
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return "", fmt.Errorf("directory native stat data has unexpected type")
	}
	for _, name := range []string{"Ctim", "Ctimespec"} {
		field := value.FieldByName(name)
		if !field.IsValid() || field.Kind() != reflect.Struct {
			continue
		}
		sec, nsec := field.FieldByName("Sec"), field.FieldByName("Nsec")
		if sec.IsValid() && nsec.IsValid() && sec.CanInt() && nsec.CanInt() {
			return fmt.Sprintf("%d:%d", sec.Int(), nsec.Int()), nil
		}
	}
	ctime, nsec := value.FieldByName("Ctime"), value.FieldByName("Ctimensec")
	if ctime.IsValid() && nsec.IsValid() && ctime.CanInt() && nsec.CanInt() {
		return fmt.Sprintf("%d:%d", ctime.Int(), nsec.Int()), nil
	}
	return "", fmt.Errorf("directory native stat data has no change timestamp")
}
