//go:build !windows

package hook

import (
	"os"
	"reflect"
	"runtime"
	"syscall"
	"testing"
)

type witnessFileInfo struct {
	os.FileInfo
	stat *syscall.Stat_t
}

func (i witnessFileInfo) Sys() any { return i.stat }

func TestDarwinWitnessSurvivesVolatileStatFields(t *testing.T) {
	if runtime.GOOS != "darwin" {
		return
	}
	dir := t.TempDir()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	before := *info.Sys().(*syscall.Stat_t)
	after := before
	value := reflect.ValueOf(&after).Elem()
	for _, name := range []string{"Dev", "Gen"} {
		field := value.FieldByName(name)
		if field.Kind() >= reflect.Int && field.Kind() <= reflect.Int64 {
			field.SetInt(field.Int() + 1)
		} else {
			field.SetUint(field.Uint() + 1)
		}
	}
	first, err := hookNativeDirectoryWitness(nil, witnessFileInfo{info, &before})
	if err != nil {
		t.Fatal(err)
	}
	second, err := hookNativeDirectoryWitness(nil, witnessFileInfo{info, &after})
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("OS-volatile stat fields changed witness: %s -> %s", first, second)
	}
}
