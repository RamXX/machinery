//go:build !windows

package hook

import (
	"fmt"
	"os"
	"reflect"
	"syscall"
)

func hookNativeDirectoryWitness(_ *os.File, info os.FileInfo) (string, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return "", fmt.Errorf("hook state directory lacks native Unix identity")
	}
	value := reflect.Indirect(reflect.ValueOf(stat))
	dev, devOK := hookNativeInteger(value.FieldByName("Dev"))
	ino, inoOK := hookNativeInteger(value.FieldByName("Ino"))
	if !devOK || !inoOK {
		return "", fmt.Errorf("hook state directory lacks native Unix device/inode identity")
	}
	// A native filesystem generation or birth time strengthens the stable
	// device/inode identity where the operating system exposes it. The random,
	// store-local generation bound into the independent marker covers Unix
	// platforms whose stat ABI does not expose either value.
	if generation, ok := hookNativeInteger(value.FieldByName("Gen")); ok {
		return fmt.Sprintf("unix:%x:%x:gen:%x", dev, ino, generation), nil
	}
	if birth := value.FieldByName("Birthtimespec"); birth.IsValid() {
		sec, secOK := hookNativeInteger(birth.FieldByName("Sec"))
		nsec, nsecOK := hookNativeInteger(birth.FieldByName("Nsec"))
		if secOK && nsecOK {
			return fmt.Sprintf("unix:%x:%x:birth:%x:%x", dev, ino, sec, nsec), nil
		}
	}
	return fmt.Sprintf("unix:%x:%x", dev, ino), nil
}

func hookNativeInteger(value reflect.Value) (uint64, bool) {
	if !value.IsValid() {
		return 0, false
	}
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint64(value.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Uint(), true
	default:
		return 0, false
	}
}
