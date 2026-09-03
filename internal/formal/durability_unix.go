//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package formal

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"syscall"
)

func syncFormalDirectory(root *os.Root) error {
	f, err := root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}

func formalNativeFileWitness(_ *os.File, info os.FileInfo) (string, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return "", fmt.Errorf("formal file lacks native Unix identity")
	}
	sec, nsec, ok := formalUnixStableGeneration(stat)
	if !ok {
		return "", fmt.Errorf("formal file lacks native Unix generation identity")
	}
	return fmt.Sprintf("unix:%x:%x:%x:%x", stat.Dev, stat.Ino, sec, nsec), nil
}

func formalUnixStableGeneration(stat any) (int64, int64, bool) {
	value := reflect.Indirect(reflect.ValueOf(stat))
	for _, name := range []string{"Birthtimespec"} {
		field := value.FieldByName(name)
		if field.IsValid() {
			sec, secOK := formalUnixInteger(field.FieldByName("Sec"))
			nsec, nsecOK := formalUnixInteger(field.FieldByName("Nsec"))
			return sec, nsec, secOK && nsecOK
		}
	}
	if generation, ok := formalUnixInteger(value.FieldByName("Gen")); ok {
		return generation, 0, true
	}
	for _, name := range []string{"Ctim", "Ctimespec"} {
		field := value.FieldByName(name)
		if field.IsValid() {
			sec, secOK := formalUnixInteger(field.FieldByName("Sec"))
			nsec, nsecOK := formalUnixInteger(field.FieldByName("Nsec"))
			return sec, nsec, secOK && nsecOK
		}
	}
	sec, secOK := formalUnixInteger(value.FieldByName("Ctime"))
	nsec, nsecOK := formalUnixInteger(value.FieldByName("Ctimensec"))
	return sec, nsec, secOK && nsecOK
}

func formalUnixInteger(value reflect.Value) (int64, bool) {
	if !value.IsValid() {
		return 0, false
	}
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(value.Uint()), true
	default:
		return 0, false
	}
}
