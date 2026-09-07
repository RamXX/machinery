//go:build !darwin && !linux

package install

import (
	"fmt"
	"os"
	"runtime"
)

func bootstrapReceiptWritable(file *os.File) (bool, error) {
	return false, fmt.Errorf("receipt descriptor access-mode query unsupported on %s", runtime.GOOS)
}
