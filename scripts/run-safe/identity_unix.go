//go:build !windows

package main

import (
	"fmt"
	"os"
	"reflect"
	"syscall"
)

func nativeExecutableWitness(_ *os.File, info os.FileInfo) (identity, change string, err error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return "", "", fmt.Errorf("executable lacks native Unix identity")
	}
	change = unixChangeWitness(stat)
	if change == "" {
		return "", "", fmt.Errorf("executable lacks native Unix change identity")
	}
	return fmt.Sprintf("unix:%x:%x", stat.Dev, stat.Ino), change, nil
}

func nativeSymlinkWitness(_ string, info os.FileInfo) (identity, change string, err error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return "", "", fmt.Errorf("executable symlink lacks native Unix identity")
	}
	change = unixChangeWitness(stat)
	if change == "" {
		return "", "", fmt.Errorf("executable symlink lacks native Unix change identity")
	}
	return fmt.Sprintf("unix:%x:%x", stat.Dev, stat.Ino), change, nil
}

func unixChangeWitness(stat any) string {
	value := reflect.Indirect(reflect.ValueOf(stat))
	for _, name := range []string{"Ctim", "Ctimespec"} {
		field := value.FieldByName(name)
		if field.IsValid() {
			sec, secOK := reflectedInteger(field.FieldByName("Sec"))
			nsec, nsecOK := reflectedInteger(field.FieldByName("Nsec"))
			if secOK && nsecOK {
				return fmt.Sprintf("ctime:%d:%d", sec, nsec)
			}
		}
	}
	sec, secOK := reflectedInteger(value.FieldByName("Ctime"))
	nsec, nsecOK := reflectedInteger(value.FieldByName("Ctimensec"))
	if secOK && nsecOK {
		return fmt.Sprintf("ctime:%d:%d", sec, nsec)
	}
	return ""
}

func reflectedInteger(value reflect.Value) (int64, bool) {
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
