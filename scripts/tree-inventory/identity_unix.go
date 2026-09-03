//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package main

import (
	"fmt"
	"os"
	"reflect"
	"syscall"
)

func nativeWitness(_ *os.File, info os.FileInfo) (string, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return "", fmt.Errorf("filesystem entry lacks native Unix identity")
	}
	value := reflect.Indirect(reflect.ValueOf(stat))
	dev, devOK := nativeInteger(value.FieldByName("Dev"))
	ino, inoOK := nativeInteger(value.FieldByName("Ino"))
	if !devOK || !inoOK {
		return "", fmt.Errorf("filesystem entry lacks native Unix device/inode identity")
	}
	change, changeOK := nativeTimespec(value, "Ctim", "Ctimespec")
	if !changeOK {
		sec, secOK := nativeInteger(value.FieldByName("Ctime"))
		nsec, nsecOK := nativeInteger(value.FieldByName("Ctimensec"))
		if secOK {
			if !nsecOK {
				nsec = 0
			}
			change, changeOK = fmt.Sprintf("%x:%x", sec, nsec), true
		}
	}
	if !changeOK {
		return "", fmt.Errorf("filesystem entry lacks native Unix change identity")
	}
	generation := "none"
	if generationValue, ok := nativeInteger(value.FieldByName("Gen")); ok {
		generation = fmt.Sprintf("gen:%x", generationValue)
	} else if birth, ok := nativeTimespec(value, "Birthtimespec"); ok {
		generation = "birth:" + birth
	}
	return fmt.Sprintf("unix:%x:%x:%s:change:%s", dev, ino, generation, change), nil
}

func nativeTimespec(value reflect.Value, names ...string) (string, bool) {
	for _, name := range names {
		field := value.FieldByName(name)
		if !field.IsValid() {
			continue
		}
		sec, secOK := nativeInteger(field.FieldByName("Sec"))
		nsec, nsecOK := nativeInteger(field.FieldByName("Nsec"))
		if secOK && nsecOK {
			return fmt.Sprintf("%x:%x", sec, nsec), true
		}
	}
	return "", false
}

func nativeInteger(value reflect.Value) (uint64, bool) {
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
