//go:build unix

package runtimeclosure

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const layoutTestPlatformPkg = "typescript-darwin-arm64"

// tsLayoutRoot returns a symlink-free temporary root so expected paths
// compare equal to the resolver's canonical paths (macOS TMPDIR lives
// behind /var -> /private/var).
func tsLayoutRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func tsLayoutWrite(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

func tsLayoutLink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

// tsLayoutKeg lays down a Homebrew-shaped keg: <prefix>/bin/tsc links to
// Cellar/typescript/<kegName>/bin/tsc, which links to the keg's platform
// compiler, exactly the chain Homebrew installs. It returns the prefix bin
// entry and the keg's node_modules directory.
func tsLayoutKeg(t *testing.T, root, kegName, pkgVersion string) (string, string) {
	t.Helper()
	keg := filepath.Join(root, "Cellar", "typescript", kegName)
	modules := filepath.Join(keg, "libexec", "lib", "node_modules")
	tsLayoutWrite(t, filepath.Join(modules, "typescript", "package.json"), `{"name":"typescript","version":"`+pkgVersion+`"}`, 0o644)
	tsLayoutWrite(t, filepath.Join(modules, "typescript", "bin", "tsc"), "#!/usr/bin/env node\n", 0o755)
	platform := filepath.Join(modules, "@typescript", layoutTestPlatformPkg)
	tsLayoutWrite(t, filepath.Join(platform, "package.json"), `{"name":"@typescript/`+layoutTestPlatformPkg+`","version":"`+pkgVersion+`"}`, 0o644)
	tsLayoutWrite(t, filepath.Join(platform, "lib", "tsc"), "native compiler bytes\n", 0o755)
	tsLayoutWrite(t, filepath.Join(platform, "lib", "lib.d.ts"), "declare var x: number;\n", 0o644)
	tsLayoutLink(t, "../libexec/lib/node_modules/@typescript/"+layoutTestPlatformPkg+"/lib/tsc", filepath.Join(keg, "bin", "tsc"))
	entry := filepath.Join(root, "bin", "tsc")
	tsLayoutLink(t, "../Cellar/typescript/"+kegName+"/bin/tsc", entry)
	return entry, modules
}

func TestTypeScriptLayoutNPMPackageBin(t *testing.T) {
	root := tsLayoutRoot(t)
	pkg := filepath.Join(root, "lib", "node_modules", "typescript")
	tsLayoutWrite(t, filepath.Join(pkg, "bin", "tsc"), "#!/usr/bin/env node\n", 0o755)
	entry := filepath.Join(root, "bin", "tsc")
	tsLayoutLink(t, "../lib/node_modules/typescript/bin/tsc", entry)
	layout, err := resolveTypeScriptLayout(entry, layoutTestPlatformPkg)
	if err != nil {
		t.Fatalf("npm global layout rejected: %v", err)
	}
	want := typeScriptLayout{
		pkgRoot:      pkg,
		compilerPath: filepath.Join(pkg, "node_modules", "@typescript", layoutTestPlatformPkg, "lib", "tsc"),
		closureRoot:  pkg,
	}
	if layout != want {
		t.Fatalf("npm layout = %+v, want %+v", layout, want)
	}
}

func TestTypeScriptLayoutHomebrewSymlinkedEntry(t *testing.T) {
	root := tsLayoutRoot(t)
	entry, modules := tsLayoutKeg(t, root, "7.0.2", "7.0.2")
	layout, err := resolveTypeScriptLayout(entry, layoutTestPlatformPkg)
	if err != nil {
		t.Fatalf("Homebrew symlinked entry rejected: %v", err)
	}
	want := typeScriptLayout{
		pkgRoot:      filepath.Join(modules, "typescript"),
		compilerPath: filepath.Join(modules, "@typescript", layoutTestPlatformPkg, "lib", "tsc"),
		closureRoot:  modules,
		kegVersion:   "7.0.2",
	}
	if layout != want {
		t.Fatalf("Homebrew layout = %+v, want %+v", layout, want)
	}
}

func TestTypeScriptLayoutHomebrewRevisionSuffixIsSameVersion(t *testing.T) {
	for _, keg := range []string{"7.0.2_1", "7.0.2_12"} {
		t.Run(keg, func(t *testing.T) {
			root := tsLayoutRoot(t)
			entry, _ := tsLayoutKeg(t, root, keg, "7.0.2")
			layout, err := resolveTypeScriptLayout(entry, layoutTestPlatformPkg)
			if err != nil {
				t.Fatalf("revision keg %s rejected: %v", keg, err)
			}
			if layout.kegVersion != "7.0.2" {
				t.Fatalf("revision keg %s carries version %q, want 7.0.2", keg, layout.kegVersion)
			}
		})
	}
}

func TestTypeScriptLayoutRejectsEntryOutsideItsPackage(t *testing.T) {
	cases := map[string]func(t *testing.T, root string) string{
		// A plain link to a loose file that is in no package bin directory.
		"loose file": func(t *testing.T, root string) string {
			tsLayoutWrite(t, filepath.Join(root, "tools", "tsc"), "#!/bin/sh\n", 0o755)
			entry := filepath.Join(root, "bin", "tsc")
			tsLayoutLink(t, "../tools/tsc", entry)
			return entry
		},
		// A keg-shaped link whose real target leaves the keg.
		"keg link escapes keg": func(t *testing.T, root string) string {
			entry, _ := tsLayoutKeg(t, root, "7.0.2_1", "7.0.2")
			kegBin := filepath.Join(root, "Cellar", "typescript", "7.0.2_1", "bin", "tsc")
			if err := os.Remove(kegBin); err != nil {
				t.Fatal(err)
			}
			tsLayoutWrite(t, filepath.Join(root, "elsewhere", "lib", "tsc"), "foreign bytes\n", 0o755)
			tsLayoutLink(t, filepath.Join(root, "elsewhere", "lib", "tsc"), kegBin)
			return entry
		},
		// A keg directory whose name is not a version (plus revision).
		"keg name is not a version": func(t *testing.T, root string) string {
			entry, _ := tsLayoutKeg(t, root, "7.0.2-beta", "7.0.2")
			return entry
		},
		// A version-shaped keg that is not the typescript formula.
		"keg is another formula": func(t *testing.T, root string) string {
			entry, _ := tsLayoutKeg(t, root, "7.0.2_1", "7.0.2")
			if err := os.Rename(filepath.Join(root, "Cellar", "typescript"), filepath.Join(root, "Cellar", "notypescript")); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(entry); err != nil {
				t.Fatal(err)
			}
			tsLayoutLink(t, "../Cellar/notypescript/7.0.2_1/bin/tsc", entry)
			return entry
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			entry := build(t, tsLayoutRoot(t))
			layout, err := resolveTypeScriptLayout(entry, layoutTestPlatformPkg)
			if err == nil {
				t.Fatalf("entry outside its package accepted as %+v", layout)
			}
			if !strings.Contains(err.Error(), "unsupported installation layout") {
				t.Fatalf("entry outside its package failed for the wrong reason: %v", err)
			}
		})
	}
}

func TestTypeScriptLayoutRejectsSymlinkLoop(t *testing.T) {
	root := tsLayoutRoot(t)
	entry := filepath.Join(root, "bin", "tsc")
	tsLayoutLink(t, "tsc2", entry)
	tsLayoutLink(t, "tsc", filepath.Join(root, "bin", "tsc2"))
	if _, err := resolveTypeScriptLayout(entry, layoutTestPlatformPkg); err == nil {
		t.Fatal("symlink loop accepted")
	}
}

// openTypeScriptFixture opens the closure from a fixture keg with a fixture
// node, so the whole open path (identity checks, fingerprint) runs on the
// Homebrew layout without the host toolchain.
func openTypeScriptFixture(t *testing.T, kegName, pkgVersion string) (*TypeScript, string, error) {
	t.Helper()
	platformPkg, err := typeScriptPlatformPackage()
	if err != nil {
		t.Skip(err.Error())
	}
	root := tsLayoutRoot(t)
	keg := filepath.Join(root, "Cellar", "typescript", kegName)
	modules := filepath.Join(keg, "libexec", "lib", "node_modules")
	tsLayoutWrite(t, filepath.Join(modules, "typescript", "package.json"), `{"name":"typescript","version":"`+pkgVersion+`"}`, 0o644)
	tsLayoutWrite(t, filepath.Join(modules, "typescript", "bin", "tsc"), "#!/usr/bin/env node\n", 0o755)
	platform := filepath.Join(modules, "@typescript", platformPkg)
	tsLayoutWrite(t, filepath.Join(platform, "package.json"), `{"name":"@typescript/`+platformPkg+`","version":"`+pkgVersion+`"}`, 0o644)
	tsLayoutWrite(t, filepath.Join(platform, "lib", "tsc"), "native compiler bytes\n", 0o755)
	tsLayoutWrite(t, filepath.Join(platform, "lib", "lib.d.ts"), "declare var x: number;\n", 0o644)
	tsLayoutLink(t, "../libexec/lib/node_modules/@typescript/"+platformPkg+"/lib/tsc", filepath.Join(keg, "bin", "tsc"))
	tsLayoutLink(t, "../Cellar/typescript/"+kegName+"/bin/tsc", filepath.Join(root, "bin", "tsc"))
	node := filepath.Join(root, "node", "bin", "node")
	tsLayoutWrite(t, node, "fixture node bytes\n", 0o755)
	t.Setenv("PATH", filepath.Join(root, "bin"))
	handle, err := OpenTypeScript(context.Background(), TypeScriptRequest{NodePath: node})
	return handle, modules, err
}

func TestTypeScriptOpenHomebrewKegBindsBothPackages(t *testing.T) {
	handle, modules, err := openTypeScriptFixture(t, "7.0.2_1", RequiredTypeScriptVersion)
	if err != nil {
		t.Fatalf("Homebrew keg closure did not open: %v", err)
	}
	defer func() { _ = handle.Close() }()
	platformPkg, _ := typeScriptPlatformPackage()
	if want := filepath.Join(modules, "@typescript", platformPkg, "lib", "tsc"); handle.CompilerPath() != want {
		t.Fatalf("compiler path %s, want %s", handle.CompilerPath(), want)
	}
	if handle.pkgRoot != modules {
		t.Fatalf("fingerprinted root %s, want the keg node_modules %s holding both packages", handle.pkgRoot, modules)
	}
	if handle.Identity().Version != TypeScriptIdentityVersion {
		t.Fatalf("identity %q, want %q", handle.Identity().Version, TypeScriptIdentityVersion)
	}
	// The platform package's declaration library is part of the closure: a
	// change to it after open must fail the revalidation in Close.
	if err := os.WriteFile(filepath.Join(modules, "@typescript", platformPkg, "lib", "lib.d.ts"), []byte("tampered\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := handle.Close(); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("tampered platform declaration library was not detected: %v", err)
	}
}

func TestTypeScriptOpenRejectsKegVersionMismatch(t *testing.T) {
	_, _, err := openTypeScriptFixture(t, "7.0.3_1", RequiredTypeScriptVersion)
	if err == nil || !strings.Contains(err.Error(), "keg version") {
		t.Fatalf("keg 7.0.3_1 installing %s was not rejected: %v", RequiredTypeScriptVersion, err)
	}
}

func TestTypeScriptOpenRejectsWrongPackageVersionInKeg(t *testing.T) {
	_, _, err := openTypeScriptFixture(t, "7.0.1_1", "7.0.1")
	if err == nil || !strings.Contains(err.Error(), "is not the exact pinned compiler") {
		t.Fatalf("keg carrying TypeScript 7.0.1 was not rejected: %v", err)
	}
}
