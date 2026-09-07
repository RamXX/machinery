//go:build unix

package runtimeclosure

// Frozen RED suite for MAC-wi2u: the pinned Go 1.27.1 runtime closure that
// go-testing/v1 executes under. Positive subjects bind the real host
// toolchain through native custody; negative subjects build small synthetic
// roots so mutation is observable without ever touching the user's real
// GOROOT. On the RED stub every subject fails at OpenGo/Validate; the
// catalog-pinning controls pass.

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
)

func validGoPlatform(p string) bool { return p == "darwin/arm64" || p == "linux/amd64" }

// TestGoClosurePinsExactCatalogIdentity is a passing control: the closure
// constants are the exact first-release catalog values of
// docs/test-assurance-contract.md section 6 and the assurance lane pins.
func TestGoClosurePinsExactCatalogIdentity(t *testing.T) {
	if GoProfile != "go" {
		t.Fatalf("go runtime profile identity is %q", GoProfile)
	}
	if RequiredGoVersion != "1.27.1" {
		t.Fatalf("pinned go version is %q, want 1.27.1", RequiredGoVersion)
	}
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if !validGoPlatform(platform) {
		t.Fatalf("current platform %s is not a pinned native assurance platform", platform)
	}
}

// TestOpenGoBindsHostGoToolchainClosure proves the real host toolchain opens
// into a full closure identity: exact version, pinned platform, sha256 tree
// digest and resolved binary, without any subprocess at open time.
func TestOpenGoBindsHostGoToolchainClosure(t *testing.T) {
	handle, err := OpenGo(context.Background(), GoRequest{})
	if err != nil {
		t.Fatalf("open host go closure: %v", err)
	}
	defer func() {
		if err := handle.Close(); err != nil {
			t.Fatalf("close host go closure: %v", err)
		}
	}()
	identity := handle.Identity()
	if identity.Profile != GoProfile || identity.Version != RequiredGoVersion {
		t.Fatalf("closure identity is %+v", identity)
	}
	if identity.Platform != runtime.GOOS+"/"+runtime.GOARCH || !validGoPlatform(identity.Platform) {
		t.Fatalf("closure platform is %q", identity.Platform)
	}
	if !regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(identity.Closure) {
		t.Fatalf("closure digest is not a sha256 identity: %q", identity.Closure)
	}
	bin := handle.Binary()
	if !filepath.IsAbs(bin) || filepath.Base(bin) != "go" {
		t.Fatalf("closure binary is %q", bin)
	}
	resolved, err := filepath.EvalSymlinks(bin)
	if err != nil || resolved != bin {
		t.Fatalf("closure binary is not the resolved real file: %q -> %q (%v)", bin, resolved, err)
	}
	if fi, err := os.Stat(bin); err != nil || fi.IsDir() {
		t.Fatalf("closure binary is not a regular file: %v", err)
	}
}

// TestOpenGoRejectsMissingRuntimeRoot proves an absent runtime fails closed
// before any work happens on a suite.
func TestOpenGoRejectsMissingRuntimeRoot(t *testing.T) {
	if _, err := OpenGo(context.Background(), GoRequest{RuntimeRoot: filepath.Join(t.TempDir(), "absent")}); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("absent runtime root must fail UNSUPPORTED_VERSION, got %v", err)
	}
}

// TestOpenGoRequiresExactPinnedVersionFile proves the pure-file pre-bind
// reads the toolchain's own VERSION identity and rejects any other version.
func TestOpenGoRequiresExactPinnedVersionFile(t *testing.T) {
	root := syntheticGoRoot(t, "go1.26.0", "go version go1.26.0 "+runtime.GOOS+"/"+runtime.GOARCH+"\n", "")
	if _, err := OpenGo(context.Background(), GoRequest{RuntimeRoot: root}); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("wrong toolchain version must fail UNSUPPORTED_VERSION, got %v", err)
	}
}

// TestOpenGoRejectsExpectedClosureMismatch proves the explicit trust root is
// compared against actually opened bytes, never accepted as a receipt.
func TestOpenGoRejectsExpectedClosureMismatch(t *testing.T) {
	if _, err := OpenGo(context.Background(), GoRequest{ExpectedClosure: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("mismatched expected closure must fail on the digest comparison, got %v", err)
	}
}

// TestGoValidateBindsNativeIdentityUnderScope proves Validate's identity
// probes are real native executions under the supplied live custody scope
// and that a passing validation leaves a closeable handle.
func TestGoValidateBindsNativeIdentityUnderScope(t *testing.T) {
	scope := openRuntimeCustodyScope(t)
	defer func() {
		report := closeRuntimeCustodyScope(t, scope, 45*time.Second)
		if report.Status != processscope.StatusCleaned {
			t.Fatalf("custody close did not verify: %+v", report)
		}
	}()
	handle, err := OpenGo(context.Background(), GoRequest{})
	if err != nil {
		t.Fatalf("open host go closure: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := handle.Validate(ctx, scope); err != nil {
		t.Fatalf("validate under custody: %v", err)
	}
	if handle.Identity().Version != RequiredGoVersion {
		t.Fatalf("validated identity lost the pinned version: %+v", handle.Identity())
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("close after validation: %v", err)
	}
}

// TestGoValidateRejectsInconsistentVersionProbe proves a synthetic root that
// is version-consistent at open time but reports a different native version
// under its own probe fails closed.
func TestGoValidateRejectsInconsistentVersionProbe(t *testing.T) {
	scope := openRuntimeCustodyScope(t)
	defer func() {
		report := closeRuntimeCustodyScope(t, scope, 45*time.Second)
		if report.Status != processscope.StatusCleaned {
			t.Fatalf("custody close did not verify: %+v", report)
		}
	}()
	root := syntheticGoRoot(t, RequiredGoVersionPrefix(), "go version go1.26.9 "+runtime.GOOS+"/"+runtime.GOARCH+"\n", "")
	handle, err := OpenGo(context.Background(), GoRequest{RuntimeRoot: root})
	if err != nil {
		t.Fatalf("synthetic consistent root must open: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := handle.Validate(ctx, scope); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("version-drifted probe must fail UNSUPPORTED_VERSION, got %v", err)
	}
}

// TestGoValidateRejectsForeignRootProbe proves the runtime must agree its
// own GOROOT is the verified root; a probe reporting elsewhere fails.
func TestGoValidateRejectsForeignRootProbe(t *testing.T) {
	scope := openRuntimeCustodyScope(t)
	defer func() {
		report := closeRuntimeCustodyScope(t, scope, 45*time.Second)
		if report.Status != processscope.StatusCleaned {
			t.Fatalf("custody close did not verify: %+v", report)
		}
	}()
	root := syntheticGoRoot(t, RequiredGoVersionPrefix(), "go version go"+RequiredGoVersion+" "+runtime.GOOS+"/"+runtime.GOARCH+"\n", "/elsewhere/goroot")
	handle, err := OpenGo(context.Background(), GoRequest{RuntimeRoot: root})
	if err != nil {
		t.Fatalf("synthetic consistent root must open: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := handle.Validate(ctx, scope); err == nil || !strings.Contains(err.Error(), "GOROOT") {
		t.Fatalf("foreign root probe must fail on the GOROOT comparison, got %v", err)
	}
}

// TestGoCloseDetectsPostVerificationMutation proves Close's final
// revalidation catches a closure file mutated after open, without launching
// anything.
func TestGoCloseDetectsPostVerificationMutation(t *testing.T) {
	root := syntheticGoRoot(t, RequiredGoVersionPrefix(), "go version go"+RequiredGoVersion+" "+runtime.GOOS+"/"+runtime.GOARCH+"\n", "")
	handle, err := OpenGo(context.Background(), GoRequest{RuntimeRoot: root})
	if err != nil {
		t.Fatalf("synthetic consistent root must open: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte("go1.27.1-mutated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := handle.Close(); err == nil || !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Fatalf("mutated closure must fail STALE_INPUT at Close, got %v", err)
	}
}

// TestGoCloseIsProcessFreeFinalRevalidation proves the handle cannot be used
// twice: a second Close and any post-close validation fail closed.
func TestGoCloseIsProcessFreeFinalRevalidation(t *testing.T) {
	handle, err := OpenGo(context.Background(), GoRequest{})
	if err != nil {
		t.Fatalf("open host go closure: %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := handle.Close(); err == nil {
		t.Fatal("second close must fail closed")
	}
	scope := openRuntimeCustodyScope(t)
	defer func() { _ = closeRuntimeCustodyScope(t, scope, 45*time.Second) }()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := handle.Validate(ctx, scope); err == nil {
		t.Fatal("validate after close must fail closed")
	}
}

// RequiredGoVersionPrefix returns the VERSION-file first line of the pinned
// toolchain (the file continues with a tab-indented toolchain block).
func RequiredGoVersionPrefix() string { return "go" + RequiredGoVersion }

// syntheticGoRoot builds a small GOROOT-shaped tree whose bin/go is an
// executable script reporting the supplied probe outputs. OpenGo's pure-file
// checks see a consistent pinned tree; Validate's real probes expose the
// supplied inconsistencies.
func syntheticGoRoot(t *testing.T, versionLine, versionOutput, gorootOutput string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte(versionLine+"\n\ttime now\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\ncase \"$1\" in\nversion) printf '%s' '" + versionOutput + "' ;;\nenv) if [ \"$2\" = \"GOROOT\" ]; then printf '%s' '" + gorootOutput + "' ; fi ;;\nesac\n"
	if gorootOutput == "" {
		script = "#!/bin/sh\ncase \"$1\" in\nversion) printf '%s' '" + versionOutput + "' ;;\nenv) if [ \"$2\" = \"GOROOT\" ]; then printf '%s' \"$(cd \"$(dirname \"$0\")/..\" && pwd -P)\" ; fi ;;\nesac\n"
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "go"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}
