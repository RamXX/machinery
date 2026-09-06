package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Prospective only: these require reviewed support seams and are not RED.
func TestCheckerFixtureHelperBuildProspectivePreconditions(t *testing.T) {
	source, err := checkerFixtureExecutableSource()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("valid-control", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
		defer cancel()
		path, err := checkerFixtureExecutableBuild(ctx, source)
		if err != nil || !pathWithin(path, cmdTestControlRoot) {
			t.Fatalf("valid build path=%q err=%v", path, err)
		}
	})
	t.Run("already-canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		path, err := checkerFixtureExecutableBuild(ctx, source)
		if path != "" || !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled build path=%q err=%v", path, err)
		}
	})
	for _, source := range []string{"verify_checkers_test.go", filepath.Join(t.TempDir(), "missing.go"), t.TempDir()} {
		t.Run("unavailable-"+filepath.Base(source), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
			defer cancel()
			path, err := checkerFixtureExecutableBuild(ctx, source)
			if path != "" || err == nil || !strings.Contains(err.Error(), "checker fixture test source") {
				t.Fatalf("source path=%q err=%v", path, err)
			}
		})
	}
}
