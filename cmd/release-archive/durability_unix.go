//go:build !windows

package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
)

func syncArchiveDirectory(root *os.Root) error {
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}

func archiveNativeFileIdentity(_ *os.File, info os.FileInfo) (string, error) {
	if info == nil || info.Sys() == nil {
		return "", fmt.Errorf("release archive file lacks native identity")
	}
	value := reflect.ValueOf(info.Sys())
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return "", fmt.Errorf("release archive file lacks native identity")
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return "", fmt.Errorf("release archive file lacks native identity")
	}
	device, inode := value.FieldByName("Dev"), value.FieldByName("Ino")
	deviceValue, deviceOK := archiveIdentityInteger(device)
	inodeValue, inodeOK := archiveIdentityInteger(inode)
	if !deviceOK || !inodeOK {
		return "", fmt.Errorf("release archive file lacks native device/inode identity")
	}
	return fmt.Sprintf("unix:%x:%x", deviceValue, inodeValue), nil
}

func archiveIdentityInteger(value reflect.Value) (uint64, bool) {
	if !value.IsValid() {
		return 0, false
	}
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		integer := value.Int()
		return uint64(integer), integer >= 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint(), true
	default:
		return 0, false
	}
}
