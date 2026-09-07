//go:build unix

package runtimeclosure

// RED contract for MAC-8yai: the pinned Elixir 1.20.4 / OTP 29.0.6 (ERTS
// 17.0.6) runtime closure. Every subject needs the real GREEN handle; the
// RED stub fails each with ErrElixirClosureNotImplemented instead of
// silently passing. No subject launches an unowned subprocess.

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

// exSupportedPlatform pins the supported native pair anything below may
// execute on.
func exSupportedPlatform() string {
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if platform == "darwin/arm64" || platform == "linux/amd64" {
		return platform
	}
	return ""
}

// discoverElixirForTest resolves the host elixir runtime through its real
// symlink chain exactly like the production handle.
func discoverElixirForTest() (string, error) {
	discovered, err := exec.LookPath("elixir")
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(discovered)
}

func exOpenCtx(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 2*time.Minute)
}

// TestElixirClosureOpensPinnedCatalog freezes the open contract: the
// complete closure (the Elixir launcher and its lib tree, the mix launcher,
// the Erlang/OTP installation with its ERTS) opens process-free with the
// exact catalog identity.
func TestElixirClosureOpensPinnedCatalog(t *testing.T) {
	if exSupportedPlatform() == "" {
		t.Skip("UNSUPPORTED_PLATFORM: not a pinned native assurance platform")
	}
	ctx, cancel := exOpenCtx(t)
	defer cancel()
	handle, err := OpenElixir(ctx, ElixirRequest{})
	if err != nil {
		t.Fatalf("pinned Elixir closure did not open: %v", err)
	}
	defer func() { _ = handle.Close() }()
	identity := handle.Identity()
	if identity.Profile != ElixirProfile || identity.Version != ElixirIdentityVersion || identity.Platform != exSupportedPlatform() {
		t.Fatalf("closure identity %+v is not the exact pinned catalog", identity)
	}
	if len(identity.Closure) != 64 || strings.ToLower(identity.Closure) != identity.Closure {
		t.Fatalf("closure digest %q is not a lowercase sha256", identity.Closure)
	}
	for _, path := range []string{handle.ElixirPath(), handle.MixPath(), handle.ErlangPath()} {
		if !filepath.IsAbs(path) {
			t.Fatalf("closure path %q is not absolute", path)
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			t.Fatalf("closure path %q is not a regular executable", path)
		}
	}
	if handle.MixPath() == handle.ElixirPath() || handle.ElixirPath() == handle.ErlangPath() {
		t.Fatal("the elixir, mix and erl closure members must be distinct")
	}
	if filepath.Dir(handle.MixPath()) != filepath.Dir(handle.ElixirPath()) {
		t.Fatal("mix launcher does not live beside the elixir launcher")
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("pure closure close failed: %v", err)
	}
}

// TestElixirClosureExpectedDigestEnforced freezes the exact-closure trust
// rule: an expected digest equal to the opened closure is accepted and any
// other value fails closed before work.
func TestElixirClosureExpectedDigestEnforced(t *testing.T) {
	if exSupportedPlatform() == "" {
		t.Skip("UNSUPPORTED_PLATFORM: not a pinned native assurance platform")
	}
	ctx, cancel := exOpenCtx(t)
	defer cancel()
	handle, err := OpenElixir(ctx, ElixirRequest{})
	if err != nil {
		t.Fatalf("pinned Elixir closure did not open: %v", err)
	}
	closure := handle.Identity().Closure
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
	ctx2, cancel2 := exOpenCtx(t)
	defer cancel2()
	ok, err := OpenElixir(ctx2, ElixirRequest{ExpectedClosure: closure})
	if err != nil {
		t.Fatalf("matching expected closure rejected: %v", err)
	}
	_ = ok.Close()
	ctx3, cancel3 := exOpenCtx(t)
	defer cancel3()
	if _, err := OpenElixir(ctx3, ElixirRequest{ExpectedClosure: strings.Repeat("0", 64)}); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("mismatched expected closure must fail closed: %v", err)
	}
}

// TestElixirClosureValidateProbesUnderCustody freezes the identity probes:
// Validate launches exactly the runtime's own version invocations (elixir
// --version, mix --version) as real scoped subprocesses, and a forged or
// absent runtime layout is rejected process-free before any probe.
func TestElixirClosureValidateProbesUnderCustody(t *testing.T) {
	if exSupportedPlatform() == "" {
		t.Skip("UNSUPPORTED_PLATFORM: not a pinned native assurance platform")
	}
	underlying := openRuntimeCustodyScope(t)
	scope := &recordingScope{Scope: underlying, runs: new(int)}
	defer func() { _ = closeRuntimeCustodyScope(t, scope.Scope, 45*time.Second) }()
	ctx, cancel := exOpenCtx(t)
	defer cancel()
	handle, err := OpenElixir(ctx, ElixirRequest{})
	if err != nil {
		t.Fatalf("pinned Elixir closure did not open: %v", err)
	}
	if err := handle.Validate(ctx, scope); err != nil {
		t.Fatalf("pinned closure validation failed: %v", err)
	}
	if *scope.runs != 2 {
		t.Fatalf("identity probes launched %d scoped subprocesses, want exactly the elixir and mix probes", *scope.runs)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("closure close failed: %v", err)
	}
	if err := handle.Validate(ctx, scope); err == nil {
		t.Fatal("validating a closed handle must fail")
	}
	shim := t.TempDir()
	forged := filepath.Join(shim, "elixir")
	if err := os.WriteFile(forged, []byte("#!/bin/sh\necho \"Elixir 1.19.0\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx2, cancel2 := exOpenCtx(t)
	defer cancel2()
	if _, err := OpenElixir(ctx2, ElixirRequest{ElixirPath: forged}); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("forged runtime layout must be rejected process-free: %v", err)
	}
}

// TestElixirClosureCloseIsPureRevalidation freezes the Close boundary:
// Close never launches a process, detects late mutation of the retained
// runtime tree, and stays a single final revalidation. The private copy
// preserves the runtime's own layout so the real probes keep working over
// copied bytes.
func TestElixirClosureCloseIsPureRevalidation(t *testing.T) {
	if exSupportedPlatform() == "" {
		t.Skip("UNSUPPORTED_PLATFORM: not a pinned native assurance platform")
	}
	discovered, err := discoverElixirForTest()
	if err != nil {
		t.Fatal(err)
	}
	elixirRoot := filepath.Dir(filepath.Dir(discovered))
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, "lib", "elixir", "bin")), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := exCopyTree(elixirRoot, filepath.Join(root, "lib", "elixir")); err != nil {
		t.Fatalf("copy elixir tree: %v", err)
	}
	ctx, cancel := exOpenCtx(t)
	defer cancel()
	handle, err := OpenElixir(ctx, ElixirRequest{ElixirPath: filepath.Join(root, "lib", "elixir", "bin", "elixir")})
	if err != nil {
		t.Fatalf("pinned closure did not open over the private elixir copy: %v", err)
	}
	underlying := openRuntimeCustodyScope(t)
	scope := &recordingScope{Scope: underlying, runs: new(int)}
	defer func() { _ = closeRuntimeCustodyScope(t, scope.Scope, 45*time.Second) }()
	if err := handle.Validate(ctx, scope); err != nil {
		t.Fatalf("private copy validation failed: %v", err)
	}
	victim := filepath.Join(root, "lib", "elixir", "bin", "mix")
	body, err := os.ReadFile(victim)
	if err != nil {
		t.Fatal(err)
	}
	// The copied launcher keeps its read-only install mode; make the
	// private copy writable before appending the mutation byte.
	if err := os.Chmod(victim, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(victim, append(append([]byte{}, body...), 0x00), 0o755); err != nil {
		t.Fatal(err)
	}
	err = handle.Close()
	if err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("late mutation of retained runtime bytes must fail the pure revalidation: %v", err)
	}
}

// exCopyTree copies a regular-file/directory tree (symlinks are resolved
// into their real bytes) for the private-copy closure tests.
func exCopyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			resolvedInfo, err := os.Stat(resolved)
			if err != nil {
				return err
			}
			if resolvedInfo.IsDir() {
				return exCopyTree(resolved, target)
			}
			body, err := os.ReadFile(resolved)
			if err != nil {
				return err
			}
			return os.WriteFile(target, body, resolvedInfo.Mode().Perm())
		case info.IsDir():
			return os.MkdirAll(target, info.Mode().Perm())
		case info.Mode().IsRegular():
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, body, info.Mode().Perm())
		default:
			return nil
		}
	})
}
