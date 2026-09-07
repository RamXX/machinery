package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/processcontrol"
)

const checkerFixtureHelperChildEnv = "MACHINERY_CHECKER_FIXTURE_HELPER_CHILD"

func TestCheckerFixtureHelperAmbientCoverageProtocol(t *testing.T) {
	if mode := os.Getenv(checkerFixtureHelperChildEnv); mode != "" {
		if testing.CoverMode() == "" {
			t.Fatal("child must remain coverage-instrumented")
		}
		if mode == "ambient-goflags" && !strings.Contains(os.Getenv("GOFLAGS"), "-coverpkg=") {
			t.Fatalf("ambient GOFLAGS were not passed to child: %q", os.Getenv("GOFLAGS"))
		}
		if mode == "persisted-goenv" && os.Getenv("GOENV") == "" {
			t.Fatal("persisted GOENV fixture was not passed to child")
		}
		return
	}
	for _, tc := range []struct {
		name      string
		persisted bool
	}{
		{"ambient-goflags", false},
		{"persisted-goenv", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owned := t.TempDir()
			sentinel := filepath.Join(owned, "must-not-exist.out")
			coverageFlags := "-cover -coverpkg=github.com/RamXX/machinery/cmd/machinery -covermode=atomic -coverprofile=" + sentinel
			goenv, flags := "off", coverageFlags
			if tc.persisted {
				goenv, flags = filepath.Join(owned, "goenv"), ""
				if err := os.WriteFile(goenv, []byte("GOFLAGS="+coverageFlags+"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			binary := checkerFixtureHelperBuildCoveredBinary(t, owned)
			profile := filepath.Join(owned, "parent.out")
			stdout, stderr, err := checkerFixtureHelperRun(t, t.Context(), binary, environmentWithOverrides(os.Environ(), []string{
				checkerFixtureHelperChildEnv + "=" + tc.name, "GOENV=" + goenv, "GOFLAGS=" + flags,
			}), "-test.v", "-test.count=1", "-test.timeout=60s", "-test.coverprofile="+profile,
				"-test.run=^(TestCheckerFixtureHelperAmbientCoverageProtocol|TestCheckerFixtureProtocolControls)$")
			if err != nil {
				t.Fatalf("covered helper child failed: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
			}
			if strings.Contains(stdout+stderr, "GOCOVERDIR not set") {
				t.Fatalf("covered parent leaked coverage diagnostic: stdout=%q stderr=%q", stdout, stderr)
			}
			if _, err := os.Stat(sentinel); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("private helper honored ambient profile destination: %v", err)
			}
			checkerCoverageProfile(t, context.Background(), profile)
		})
	}
}

func TestCheckerFixtureHelperCachedCompilerFailure(t *testing.T) {
	if os.Getenv(checkerFixtureHelperChildEnv) == "cached-compiler-failure" {
		originalPath := os.Getenv("PATH")
		t.Setenv("PATH", t.TempDir())
		t.Run("first-build-fails", func(t *testing.T) { _ = checkerFixtureEngineArgs(t, "oci-engine") })
		t.Setenv("PATH", originalPath)
		t.Run("cached-after-path-repair", func(t *testing.T) { _ = checkerFixtureEngineArgs(t, "oci-engine") })
		return
	}
	owned := t.TempDir()
	binary := checkerFixtureHelperBuildCoveredBinary(t, owned)
	stdout, stderr, err := checkerFixtureHelperRun(t, t.Context(), binary,
		environmentWithOverrides(os.Environ(), []string{checkerFixtureHelperChildEnv + "=cached-compiler-failure"}),
		"-test.v", "-test.count=1", "-test.timeout=60s", "-test.coverprofile="+filepath.Join(owned, "parent.out"),
		"-test.run=^TestCheckerFixtureHelperCachedCompilerFailure$")
	if !checkerFixtureHelperOnlyExitError(err, 1) {
		t.Fatalf("child terminal error = %v, want only native exit 1", err)
	}
	for _, leaf := range []string{"first-build-fails", "cached-after-path-repair"} {
		if !strings.Contains(stdout, "=== RUN   TestCheckerFixtureHelperCachedCompilerFailure/"+leaf+"\n") || !strings.Contains(stdout, "--- FAIL: TestCheckerFixtureHelperCachedCompilerFailure/"+leaf) {
			t.Fatalf("missing expected failed child %q: stdout=%q stderr=%q", leaf, stdout, stderr)
		}
	}
	if strings.Count(stdout, "executable file not found") != 2 || strings.Contains(stdout+stderr, "context deadline exceeded") || strings.Contains(stdout+stderr, "panic:") || strings.Contains(stdout+stderr, "--- SKIP:") || strings.Contains(stdout, "=== RUN   TestCheckerFixtureHelperCachedCompilerFailure/") && strings.Count(stdout, "=== RUN   TestCheckerFixtureHelperCachedCompilerFailure/") != 2 {
		t.Fatalf("cached compiler failure had an unexpected terminal shape: stdout=%q stderr=%q", stdout, stderr)
	}
	if strings.Count(stdout, "compile checker fixture executable") != 2 {
		t.Fatalf("compiler failure was not cached across PATH repair: stdout=%q stderr=%q", stdout, stderr)
	}
}

func checkerFixtureHelperBuildCoveredBinary(t *testing.T, owned string) string {
	t.Helper()
	binary := filepath.Join(owned, "checker-helper.test")
	stdout, stderr, err := checkerFixtureHelperRun(t, t.Context(), "go", environmentWithOverrides(os.Environ(), []string{"GOENV=off", "GOFLAGS=-cover=false"}), "test", "-c", "-cover", "-o", binary, ".")
	if err != nil || stdout != "" || stderr != "" {
		t.Fatalf("build covered helper-test child: err=%v stdout=%q stderr=%q", err, stdout, stderr)
	}
	return binary
}

func checkerFixtureHelperRun(t *testing.T, parent context.Context, name string, environment []string, args ...string) (string, string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env, cmd.WaitDelay = environment, time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := processcontrol.Run(ctx, cmd)
	return stdout.String(), stderr.String(), err
}

func checkerFixtureHelperOnlyExitError(err error, want int) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode() == want
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		errors := joined.Unwrap()
		return len(errors) != 0 && allCheckerFixtureHelperExitErrors(errors, want)
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return checkerFixtureHelperOnlyExitError(wrapped.Unwrap(), want)
	}
	return false
}

func allCheckerFixtureHelperExitErrors(errors []error, want int) bool {
	for _, err := range errors {
		if !checkerFixtureHelperOnlyExitError(err, want) {
			return false
		}
	}
	return true
}
