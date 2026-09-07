//go:build unix

package runtimeclosure

// RED contract for MAC-avfp: the pinned Node 26.8.1 / TypeScript 7.0.2
// runtime closure. Every subject needs the real GREEN handle; the RED stub
// fails each with ErrTypeScriptClosureNotImplemented instead of silently
// passing. No subject launches an unowned subprocess.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
)

// tsSupportedPlatform pins the supported native pair anything below may
// execute on.
func tsSupportedPlatform() string {
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if platform == "darwin/arm64" || platform == "linux/amd64" {
		return platform
	}
	return ""
}

// discoverNodeForTest resolves the host node runtime through its real
// symlink chain exactly like the production handle.
func discoverNodeForTest() (string, error) {
	discovered, err := exec.LookPath("node")
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(discovered)
}

func tsOpenCtx(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 2*time.Minute)
}

// TestTypeScriptClosureOpensPinnedCatalog freezes the open contract: the
// complete closure (node executable, TypeScript package tree, platform
// native compiler) opens process-free with the exact catalog identity.
func TestTypeScriptClosureOpensPinnedCatalog(t *testing.T) {
	if tsSupportedPlatform() == "" {
		t.Skip("UNSUPPORTED_PLATFORM: not a pinned native assurance platform")
	}
	ctx, cancel := tsOpenCtx(t)
	defer cancel()
	handle, err := OpenTypeScript(ctx, TypeScriptRequest{})
	if err != nil {
		t.Fatalf("pinned TypeScript closure did not open: %v", err)
	}
	defer func() { _ = handle.Close() }()
	identity := handle.Identity()
	if identity.Profile != TypeScriptProfile || identity.Version != TypeScriptIdentityVersion || identity.Platform != tsSupportedPlatform() {
		t.Fatalf("closure identity %+v is not the exact pinned catalog", identity)
	}
	if len(identity.Closure) != 64 || strings.ToLower(identity.Closure) != identity.Closure {
		t.Fatalf("closure digest %q is not a lowercase sha256", identity.Closure)
	}
	for _, path := range []string{handle.NodePath(), handle.CompilerPath()} {
		if !filepath.IsAbs(path) {
			t.Fatalf("closure path %q is not absolute", path)
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			t.Fatalf("closure path %q is not a regular executable", path)
		}
	}
	if handle.NodePath() == handle.CompilerPath() {
		t.Fatal("node runtime and native compiler must be distinct closure members")
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("pure closure close failed: %v", err)
	}
}

// TestTypeScriptClosureExpectedDigestEnforced freezes the exact-closure
// trust rule: an expected digest equal to the opened closure is accepted and
// any other value fails closed before work.
func TestTypeScriptClosureExpectedDigestEnforced(t *testing.T) {
	if tsSupportedPlatform() == "" {
		t.Skip("UNSUPPORTED_PLATFORM: not a pinned native assurance platform")
	}
	ctx, cancel := tsOpenCtx(t)
	defer cancel()
	handle, err := OpenTypeScript(ctx, TypeScriptRequest{})
	if err != nil {
		t.Fatalf("pinned TypeScript closure did not open: %v", err)
	}
	closure := handle.Identity().Closure
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
	ctx2, cancel2 := tsOpenCtx(t)
	defer cancel2()
	ok, err := OpenTypeScript(ctx2, TypeScriptRequest{ExpectedClosure: closure})
	if err != nil {
		t.Fatalf("matching expected closure rejected: %v", err)
	}
	_ = ok.Close()
	ctx3, cancel3 := tsOpenCtx(t)
	defer cancel3()
	if _, err := OpenTypeScript(ctx3, TypeScriptRequest{ExpectedClosure: strings.Repeat("0", 64)}); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("mismatched expected closure must fail closed: %v", err)
	}
}

// recordingScope counts scoped launches (including child-scoped probes) to
// prove the probe/Close boundary; the counter is shared across the child
// chain.
type recordingScope struct {
	processscope.Scope
	runs *int
}

func (r *recordingScope) Run(ctx context.Context, cmd processscope.Command, streams processscope.Streams) (processscope.Result, error) {
	*r.runs++
	return r.Scope.Run(ctx, cmd, streams)
}

func (r *recordingScope) Child(ctx context.Context) (processscope.Scope, error) {
	child, err := r.Scope.Child(ctx)
	if err != nil {
		return nil, err
	}
	return &recordingScope{Scope: child, runs: r.runs}, nil
}

// TestTypeScriptClosureValidateProbesUnderCustody freezes the identity
// probes: Validate launches exactly the runtime's own version invocations as
// real scoped subprocesses and rejects a forged runtime identity.
func TestTypeScriptClosureValidateProbesUnderCustody(t *testing.T) {
	if tsSupportedPlatform() == "" {
		t.Skip("UNSUPPORTED_PLATFORM: not a pinned native assurance platform")
	}
	underlying := openRuntimeCustodyScope(t)
	scope := &recordingScope{Scope: underlying, runs: new(int)}
	defer func() { _ = closeRuntimeCustodyScope(t, scope.Scope, 45*time.Second) }()
	ctx, cancel := tsOpenCtx(t)
	defer cancel()
	handle, err := OpenTypeScript(ctx, TypeScriptRequest{})
	if err != nil {
		t.Fatalf("pinned TypeScript closure did not open: %v", err)
	}
	if err := handle.Validate(ctx, scope); err != nil {
		t.Fatalf("pinned closure validation failed: %v", err)
	}
	if *scope.runs != 2 {
		t.Fatalf("identity probes launched %d scoped subprocesses, want exactly the node and compiler probes", scope.runs)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("closure close failed: %v", err)
	}
	if err := handle.Validate(ctx, scope); err == nil {
		t.Fatal("validating a closed handle must fail")
	}
	shim := t.TempDir()
	node := filepath.Join(shim, "node")
	if err := os.WriteFile(node, []byte("#!/bin/sh\necho v25.9.9\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx2, cancel2 := tsOpenCtx(t)
	defer cancel2()
	forged, err := OpenTypeScript(ctx2, TypeScriptRequest{NodePath: node})
	if err != nil {
		t.Fatalf("process-free open binds bytes without launching: %v", err)
	}
	defer func() { _ = forged.Close() }()
	if err := forged.Validate(ctx2, scope); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("forged node identity must fail the probe: %v", err)
	}
}

// TestTypeScriptClosureCloseIsPureRevalidation freezes the Close boundary:
// Close never launches a process, detects late mutation of the retained
// runtime bytes, and stays a single final revalidation. The private copy
// preserves the runtime's own shared-library layout so the real probes keep
// working over copied bytes.
func TestTypeScriptClosureCloseIsPureRevalidation(t *testing.T) {
	if tsSupportedPlatform() == "" {
		t.Skip("UNSUPPORTED_PLATFORM: not a pinned native assurance platform")
	}
	discovered, err := discoverNodeForTest()
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(discovered)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	node := filepath.Join(root, "bin", "node")
	if err := os.WriteFile(node, body, 0o755); err != nil {
		t.Fatal(err)
	}
	if lib := discoverNodeSharedLibraryForTest(t, discovered); lib != "" {
		libBody, err := os.ReadFile(lib)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "lib", filepath.Base(lib)), libBody, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := tsOpenCtx(t)
	defer cancel()
	handle, err := OpenTypeScript(ctx, TypeScriptRequest{NodePath: node})
	if err != nil {
		t.Fatalf("pinned closure did not open over the private node copy: %v", err)
	}
	underlying := openRuntimeCustodyScope(t)
	scope := &recordingScope{Scope: underlying, runs: new(int)}
	defer func() { _ = closeRuntimeCustodyScope(t, scope.Scope, 45*time.Second) }()
	if err := handle.Validate(ctx, scope); err != nil {
		t.Fatalf("private copy validation failed: %v", err)
	}
	if err := os.WriteFile(node, append(append([]byte{}, body...), 0x00), 0o755); err != nil {
		t.Fatal(err)
	}
	err = handle.Close()
	if err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("late mutation of retained runtime bytes must fail the pure revalidation: %v", err)
	}
}

// discoverNodeSharedLibraryForTest mirrors the production discovery of the
// node runtime's own shared library.
func discoverNodeSharedLibraryForTest(t *testing.T, nodePath string) string {
	t.Helper()
	libDir := filepath.Join(filepath.Dir(filepath.Dir(nodePath)), "lib")
	entries, err := os.ReadDir(libDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.Type().IsRegular() && strings.HasPrefix(name, "libnode.") && strings.HasSuffix(name, ".dylib") {
			return filepath.Join(libDir, name)
		}
	}
	return ""
}
