//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Exercises retained-root replacement followed by a write-capable rooted
// directory FlushFileBuffers on native Windows runners.
func TestWindowsDurableArchiveReplacement(t *testing.T) {
	output := filepath.Join(t.TempDir(), "release.tar.gz")
	stamp := time.Unix(1_700_000_000, 0).UTC()
	for _, content := range []string{"first", "second"} {
		entries := []entry{{name: "machinery/file", mode: 0o644, kind: entryRegular, data: []byte(content)}}
		if err := writeArchive(output, stamp, entries); err != nil {
			t.Fatal(err)
		}
	}
	if info, err := os.Stat(output); err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		t.Fatalf("durable replacement output = %#v, %v", info, err)
	}
}
