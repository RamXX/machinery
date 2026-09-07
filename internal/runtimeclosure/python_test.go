//go:build unix

package runtimeclosure

// RED contract for MAC-imtz: the pinned CPython 3.14.7 runtime closure.
// Every subject needs the real GREEN handle; the RED stub fails each with
// ErrPythonClosureNotImplemented instead of silently passing. No subject
// launches an unowned subprocess.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

)

// pySupportedPlatform pins the supported native pair anything below may
// execute on.
func pySupportedPlatform() string {
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if platform == "darwin/arm64" || platform == "linux/amd64" {
		return platform
	}
	return ""
}

// discoverPythonForTest resolves the host python3 runtime through its real
// symlink chain exactly like the production handle.
func discoverPythonForTest() (string, error) {
	discovered, err := exec.LookPath("python3")
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(discovered)
}

func pyOpenCtx(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 2*time.Minute)
}

// TestPythonClosureOpensPinnedCatalog freezes the open contract: the
// complete closure (interpreter executable, stdlib library tree, static
// patchlevel identity) opens process-free with the exact catalog identity.
func TestPythonClosureOpensPinnedCatalog(t *testing.T) {
	if pySupportedPlatform() == "" {
		t.Skip("UNSUPPORTED_PLATFORM: not a pinned native assurance platform")
	}
	ctx, cancel := pyOpenCtx(t)
	defer cancel()
	handle, err := OpenPython(ctx, PythonRequest{})
	if err != nil {
		t.Fatalf("pinned CPython closure did not open: %v", err)
	}
	defer func() { _ = handle.Close() }()
	identity := handle.Identity()
	if identity.Profile != PythonProfile || identity.Version != RequiredPythonVersion || identity.Platform != pySupportedPlatform() {
		t.Fatalf("closure identity %+v is not the exact pinned catalog", identity)
	}
	if len(identity.Closure) != 64 || strings.ToLower(identity.Closure) != identity.Closure {
		t.Fatalf("closure digest %q is not a lowercase sha256", identity.Closure)
	}
	for _, path := range []string{handle.Binary(), handle.StdlibRoot()} {
		if !filepath.IsAbs(path) {
			t.Fatalf("closure path %q is not absolute", path)
		}
	}
	info, err := os.Lstat(handle.Binary())
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		t.Fatalf("closure interpreter %q is not a regular executable", handle.Binary())
	}
	rootInfo, err := os.Lstat(handle.StdlibRoot())
	if err != nil || !rootInfo.IsDir() {
		t.Fatalf("closure stdlib root %q is not a directory", handle.StdlibRoot())
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("pure closure close failed: %v", err)
	}
}

// TestPythonClosureExpectedDigestEnforced freezes the exact-closure trust
// rule: an expected digest equal to the opened closure is accepted and any
// other value fails closed before work.
func TestPythonClosureExpectedDigestEnforced(t *testing.T) {
	if pySupportedPlatform() == "" {
		t.Skip("UNSUPPORTED_PLATFORM: not a pinned native assurance platform")
	}
	ctx, cancel := pyOpenCtx(t)
	defer cancel()
	handle, err := OpenPython(ctx, PythonRequest{})
	if err != nil {
		t.Fatalf("pinned CPython closure did not open: %v", err)
	}
	closure := handle.Identity().Closure
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
	ctx2, cancel2 := pyOpenCtx(t)
	defer cancel2()
	ok, err := OpenPython(ctx2, PythonRequest{ExpectedClosure: closure})
	if err != nil {
		t.Fatalf("matching expected closure rejected: %v", err)
	}
	_ = ok.Close()
	ctx3, cancel3 := pyOpenCtx(t)
	defer cancel3()
	if _, err := OpenPython(ctx3, PythonRequest{ExpectedClosure: strings.Repeat("0", 64)}); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("mismatched expected closure must fail closed: %v", err)
	}
}

// TestPythonClosureValidateProbesUnderCustody freezes the identity probe:
// Validate launches exactly the runtime's own version invocation as real
// scoped subprocesses and rejects a forged runtime identity; a closed handle
// never validates.
func TestPythonClosureValidateProbesUnderCustody(t *testing.T) {
	if pySupportedPlatform() == "" {
		t.Skip("UNSUPPORTED_PLATFORM: not a pinned native assurance platform")
	}
	underlying := openRuntimeCustodyScope(t)
	scope := &recordingScope{Scope: underlying, runs: new(int)}
	defer func() { _ = closeRuntimeCustodyScope(t, scope.Scope, 45*time.Second) }()
	ctx, cancel := pyOpenCtx(t)
	defer cancel()
	handle, err := OpenPython(ctx, PythonRequest{})
	if err != nil {
		t.Fatalf("pinned CPython closure did not open: %v", err)
	}
	if err := handle.Validate(ctx, scope); err != nil {
		t.Fatalf("pinned closure validation failed: %v", err)
	}
	if *scope.runs != 1 {
		t.Fatalf("identity probes launched %d scoped subprocesses, want exactly the python --version probe", *scope.runs)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("closure close failed: %v", err)
	}
	if err := handle.Validate(ctx, scope); err == nil {
		t.Fatal("validating a closed handle must fail")
	}
	shim := t.TempDir()
	python := filepath.Join(shim, "python3")
	if err := os.WriteFile(python, []byte("#!/bin/sh\necho Python 3.13.0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx2, cancel2 := pyOpenCtx(t)
	defer cancel2()
	forged, err := OpenPython(ctx2, PythonRequest{PythonPath: python})
	if err != nil {
		t.Fatalf("process-free open binds bytes without launching: %v", err)
	}
	defer func() { _ = forged.Close() }()
	if err := forged.Validate(ctx2, scope); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("forged python identity must fail the probe: %v", err)
	}
}

// TestPythonClosureCloseIsPureRevalidation freezes the Close boundary:
// Close never launches a process, detects late mutation of the retained
// runtime bytes, and stays a single final revalidation. The private layout
// reproduces the interpreter's own prefix topology (bin + lib/python3.x
// carrying the stdlib unittest package) over copied bytes.
func TestPythonClosureCloseIsPureRevalidation(t *testing.T) {
	if pySupportedPlatform() == "" {
		t.Skip("UNSUPPORTED_PLATFORM: not a pinned native assurance platform")
	}
	discovered, err := discoverPythonForTest()
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(discovered)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	bin := filepath.Join(root, "python3")
	if err := os.WriteFile(bin, body, 0o755); err != nil {
		t.Fatal(err)
	}
	stdlib := filepath.Join(root, "lib", "python3.14", "unittest")
	if err := os.MkdirAll(stdlib, 0o755); err != nil {
		t.Fatal(err)
	}
	realStdlib := filepath.Join(filepath.Dir(filepath.Dir(discovered)), "lib", "python3.14", "unittest")
	if entries, readErr := os.ReadDir(realStdlib); readErr == nil {
		for _, entry := range entries {
			if entry.Type().IsRegular() {
				data, readErr := os.ReadFile(filepath.Join(realStdlib, entry.Name()))
				if readErr != nil {
					t.Fatal(readErr)
				}
				if err := os.WriteFile(filepath.Join(stdlib, entry.Name()), data, 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
	} else {
		t.Fatalf("cannot mirror the stdlib unittest package: %v", readErr)
	}
	ctx, cancel := pyOpenCtx(t)
	defer cancel()
	handle, err := OpenPython(ctx, PythonRequest{PythonPath: bin})
	if err != nil {
		t.Fatalf("pinned closure did not open over the private interpreter copy: %v", err)
	}
	underlying := openRuntimeCustodyScope(t)
	scope := &recordingScope{Scope: underlying, runs: new(int)}
	defer func() { _ = closeRuntimeCustodyScope(t, scope.Scope, 45*time.Second) }()
	if err := handle.Validate(ctx, scope); err != nil {
		t.Fatalf("private copy validation failed: %v", err)
	}
	if err := os.WriteFile(bin, append(append([]byte{}, body...), 0x00), 0o755); err != nil {
		t.Fatal(err)
	}
	err = handle.Close()
	if err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") && !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Fatalf("late mutation of retained runtime bytes must fail the pure revalidation: %v", err)
	}
}
