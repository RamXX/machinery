package runtimeclosure

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setExplicitJava(t *testing.T, path string) {
	t.Helper()
	digest, err := JavaClosureDigest(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(JavaEnv, path)
	t.Setenv(JavaClosureSHAEnv, digest)
}

func TestEnvironmentDropsHostileAmbientJavaSettings(t *testing.T) {
	for key, value := range map[string]string{
		"JAVA_TOOL_OPTIONS":  "-javaagent:/tmp/hostile.jar",
		"JDK_JAVA_OPTIONS":   "-Duser.language=hostile",
		"CLASSPATH":          "/tmp/hostile",
		"MACHINERY_SENTINEL": "must-not-pass",
	} {
		t.Setenv(key, value)
	}
	env := Environment("/private/home", "/private/tmp", "/verified/bin/java")
	joined := strings.Join(env, "\n")
	for _, forbidden := range []string{"JAVA_TOOL_OPTIONS", "JDK_JAVA_OPTIONS", "CLASSPATH", "MACHINERY_SENTINEL", "hostile"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("minimal environment retained %s:\n%s", forbidden, joined)
		}
	}
	for _, required := range []string{"HOME=/private/home", "TZ=UTC", "LANG=C.UTF-8", "LC_ALL=C.UTF-8", "TMPDIR=/private/tmp"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("minimal environment lacks %s:\n%s", required, joined)
		}
	}
}

func TestValidateJavaIdentityRequiresExactCIPolicy(t *testing.T) {
	valid := "    java.home = /jdk\n    java.runtime.version = 21.0.12.1+1-LTS\n    java.vendor = Eclipse Adoptium\n    java.version = 21.0.12.1\n    java.vm.name = OpenJDK 64-Bit Server VM\n"
	if err := ValidateJavaIdentity(valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, output, want string
	}{
		{"wrong major", strings.Replace(valid, "java.version = 21.0.12.1", "java.version = 17.0.12.1", 1), "require exact pinned Temurin build"},
		{"wrong security release", strings.Replace(valid, "java.version = 21.0.12.1", "java.version = 21.0.11.1", 1), "require exact pinned Temurin build"},
		{"wrong build", strings.Replace(valid, "21.0.12.1+1-LTS", "21.0.12.1+2-LTS", 1), "exact pinned build"},
		{"missing vendor", "java.home = /jdk\njava.runtime.version = 21.0.12.1+1-LTS\njava.version = 21.0.12.1\njava.vm.name = OpenJDK VM\n", "vendor"},
		{"wrong VM", strings.Replace(valid, "OpenJDK 64-Bit Server VM", "Example VM", 1), "exact pinned OpenJDK"},
		{"duplicate version", valid + "java.version = 21.0.12.1\n", "more than once"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateJavaIdentity(tc.output); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("identity accepted or wrong diagnostic: %v", err)
			}
		})
	}
	if err := ValidateJavaIdentity(strings.Replace(valid, "Eclipse Adoptium", "Homebrew", 1)); err == nil || !strings.Contains(err.Error(), "exact pinned vendor") {
		t.Fatalf("non-pinned Java distributor accepted: %v", err)
	}
}

func TestValidateJavaIdentityRejectsEveryNonCanonicalDiagnostic(t *testing.T) {
	valid := "Property settings:\n    java.class.path =\n    java.home = /jdk\n    java.runtime.version = 21.0.12.1+1-LTS\n    java.vendor = Eclipse Adoptium\n    java.version = 21.0.12.1\n    java.vm.name = OpenJDK 64-Bit Server VM\n\nopenjdk version \"21.0.12.1\" 2026-08-18 LTS\nOpenJDK Runtime Environment Temurin-21.0.12.1+1 (build 21.0.12.1+1-LTS)\nOpenJDK 64-Bit Server VM Temurin-21.0.12.1+1 (build 21.0.12.1+1-LTS, mixed mode, sharing)\n"
	if err := ValidateJavaIdentity(valid); err != nil {
		t.Fatalf("canonical pinned probe rejected: %v", err)
	}
	for _, diagnostic := range []string{
		"WARNING: CDS archive ignored",
		"Error: hostile option",
		"Exception in thread main",
		"fatal: runtime damaged",
		"deprecated option in use",
		"Picked up JAVA_TOOL_OPTIONS: -javaagent:hostile.jar",
		"arbitrary unclassified output",
		"warning = disguised diagnostic",
	} {
		t.Run(diagnostic, func(t *testing.T) {
			output := diagnostic + "\n" + valid
			for i := 0; i < 20; i++ {
				err := ValidateJavaIdentity(output)
				if err == nil || !strings.Contains(err.Error(), "unexpected") || !strings.Contains(err.Error(), diagnostic) {
					t.Fatalf("iteration %d: diagnostic accepted or misreported: %v", i, err)
				}
			}
		})
	}
}

func TestJavaValidateRejectsPathSwap(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"bin", "conf", "lib"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "lib", "modules"), []byte("modules"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "bin", "java")
	if err := os.WriteFile(path, []byte("original"), 0o755); err != nil {
		t.Fatal(err)
	}
	setExplicitJava(t, path)
	java, err := OpenJava()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = java.Close() })
	if err := os.Rename(path, path+".old"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("replacement"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := java.Validate(); err == nil || !strings.Contains(err.Error(), "changed identity") {
		t.Fatalf("java path swap accepted: %v", err)
	}
}

func TestJavaValidateRejectsRuntimeClosureMutation(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"bin", "conf", "lib"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	javaPath := filepath.Join(dir, "bin", "java")
	if err := os.WriteFile(javaPath, []byte("java"), 0o755); err != nil {
		t.Fatal(err)
	}
	modules := filepath.Join(dir, "lib", "modules")
	if err := os.WriteFile(modules, []byte("original runtime"), 0o644); err != nil {
		t.Fatal(err)
	}
	setExplicitJava(t, javaPath)
	java, err := OpenJava()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = java.Close() })
	if err := os.WriteFile(modules, []byte("mutated runtime"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := java.Validate(); err == nil || !strings.Contains(err.Error(), "runtime closure changed") {
		t.Fatalf("runtime closure mutation accepted: %v", err)
	}
}

func TestJavaValidateRejectsRuntimeRootSwapWithSameLauncherIdentity(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "jdk")
	for _, name := range []string{"bin", "conf", "lib"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	javaPath := filepath.Join(root, "bin", "java")
	if err := os.WriteFile(javaPath, []byte("java"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lib", "modules"), []byte("modules"), 0o644); err != nil {
		t.Fatal(err)
	}
	setExplicitJava(t, javaPath)
	java, err := OpenJava()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = java.Close() })
	oldRoot := filepath.Join(parent, "old-jdk")
	if err := os.Rename(root, oldRoot); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"bin", "conf", "lib"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Link(filepath.Join(oldRoot, "bin", "java"), javaPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lib", "modules"), []byte("replacement"), 0o644); err != nil {
		t.Fatal(err)
	}
	outsideSentinel := filepath.Join(parent, "outside-sentinel")
	if err := os.WriteFile(outsideSentinel, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := java.Validate(); err == nil || !strings.Contains(err.Error(), "runtime root changed identity") {
		t.Fatalf("Java runtime root swap accepted: %v", err)
	}
	if got, err := os.ReadFile(outsideSentinel); err != nil || string(got) != "untouched" {
		t.Fatalf("root swap validation touched outside sentinel: %q, %v", got, err)
	}
}

func TestExplicitJavaOverrideRejectsModifiedSameVersionClosure(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"bin", "conf", "lib"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	javaPath := filepath.Join(root, "bin", "java")
	if err := os.WriteFile(javaPath, []byte("same-version launcher"), 0o755); err != nil {
		t.Fatal(err)
	}
	modules := filepath.Join(root, "lib", "modules")
	if err := os.WriteFile(modules, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	setExplicitJava(t, javaPath)
	if err := os.WriteFile(modules, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenJava(); err == nil || !strings.Contains(err.Error(), "does not match paired trust root") {
		t.Fatalf("modified explicit Java closure accepted: %v", err)
	}
}

func TestOpenJavaRejectsSparseOversizedLauncherBeforeClosureHash(t *testing.T) {
	root := makeTestJavaRoot(t, []byte("java"), []byte("modules"))
	javaPath := filepath.Join(root, "bin", "java")
	if err := os.Truncate(javaPath, javaLauncherMaxBytes+1); err != nil {
		t.Fatal(err)
	}
	t.Setenv(JavaEnv, javaPath)
	t.Setenv(JavaClosureSHAEnv, strings.Repeat("0", 64))
	if _, err := OpenJava(); err == nil || !strings.Contains(err.Error(), "exceeds") || !strings.Contains(err.Error(), fmt.Sprintf("%d-byte limit", javaLauncherMaxBytes)) {
		t.Fatalf("sparse oversized Java launcher was accepted: %v", err)
	}
}

func TestOpenJavaLauncherRejectsPostOpenGrowth(t *testing.T) {
	dir := makeTestJavaRoot(t, []byte(strings.Repeat("a", 16)), []byte("modules"))
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	javaPath := filepath.Join(dir, "bin", "java")
	grew := false
	file, _, _, err := openJavaLauncher(root, filepath.Join("bin", "java"), 32, func() error {
		grew = true
		launcher, err := os.OpenFile(javaPath, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			return err
		}
		_, writeErr := launcher.Write([]byte(strings.Repeat("b", 32)))
		return errors.Join(writeErr, launcher.Close())
	})
	if file != nil {
		_ = file.Close()
	}
	if !grew || err == nil || !strings.Contains(err.Error(), "changed identity, metadata, or size while reading") {
		t.Fatalf("post-open launcher growth was accepted: %v", err)
	}
}

func TestOpenJavaLauncherRejectsSameContentPathABA(t *testing.T) {
	dir := makeTestJavaRoot(t, []byte("same launcher"), []byte("modules"))
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	javaPath := filepath.Join(dir, "bin", "java")
	replacement := javaPath + ".replacement"
	file, _, _, err := openJavaLauncher(root, filepath.Join("bin", "java"), javaLauncherMaxBytes, func() error {
		if err := os.WriteFile(replacement, []byte("same launcher"), 0o755); err != nil {
			return err
		}
		if err := os.Rename(javaPath, javaPath+".old"); err != nil {
			return err
		}
		return os.Rename(replacement, javaPath)
	})
	if file != nil {
		_ = file.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "changed identity") {
		t.Fatalf("same-content launcher path ABA was accepted: %v", err)
	}
	body, readErr := os.ReadFile(javaPath)
	if readErr != nil || string(body) != "same launcher" {
		t.Fatalf("foreign launcher replacement was not preserved: %q, %v", body, readErr)
	}
}

func TestOpenJavaLauncherRejectsSameInodeMetadataABA(t *testing.T) {
	dir := makeTestJavaRoot(t, []byte("same launcher"), []byte("modules"))
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	javaPath := filepath.Join(dir, "bin", "java")
	before, err := os.Lstat(javaPath)
	if err != nil {
		t.Fatal(err)
	}
	if javaFileChangeID(before) == "" {
		t.Skip("platform does not expose launcher change metadata")
	}
	file, _, _, err := openJavaLauncher(root, filepath.Join("bin", "java"), javaLauncherMaxBytes, func() error {
		launcher, err := os.OpenFile(javaPath, os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil {
			return err
		}
		_, writeErr := launcher.Write([]byte("same launcher"))
		return errors.Join(writeErr, launcher.Close(), os.Chtimes(javaPath, before.ModTime(), before.ModTime()))
	})
	if file != nil {
		_ = file.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "metadata") {
		t.Fatalf("same-inode launcher metadata ABA was accepted: %v", err)
	}
}

func TestFingerprintJavaRootRejectsSparseAggregateOversizeBeforeHashing(t *testing.T) {
	dir := makeTestJavaRoot(t, []byte("java"), []byte("modules"))
	if err := os.Truncate(filepath.Join(dir, "lib", "modules"), javaClosureMaxBytes+1); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	if _, err := fingerprintJavaRoot(root); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("exceeds %d bytes", javaClosureMaxBytes)) {
		t.Fatalf("sparse aggregate oversize was accepted: %v", err)
	}
}

func TestFingerprintJavaRootRejectsPostCensusGrowth(t *testing.T) {
	dir := makeTestJavaRoot(t, []byte("java"), []byte("modules"))
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	modules := filepath.Join("lib", "modules")
	grew := false
	_, err = fingerprintJavaRootWithHook(root, func(name string) error {
		if name != modules || grew {
			return nil
		}
		grew = true
		file, err := os.OpenFile(filepath.Join(dir, modules), os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			return err
		}
		_, writeErr := file.Write([]byte(" growth"))
		return errors.Join(writeErr, file.Close())
	})
	if !grew || err == nil || !strings.Contains(err.Error(), "changed identity, metadata, or size while hashing") {
		t.Fatalf("post-census closure growth was accepted: %v", err)
	}
}

func TestJavaInventoryRejectsEntryBeyondFixedCeiling(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"c", "a", "b"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	f, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() //nolint:errcheck // test cleanup
	if _, err := readJavaDirEntries(f, 2); err == nil || !strings.Contains(err.Error(), "entry limit") {
		t.Fatalf("high-entry Java inventory was accepted: %v", err)
	}
}

func TestJavaClosureDepthBoundaryIsPortableAndExact(t *testing.T) {
	dir := makeTestJavaRoot(t, []byte("java"), []byte("modules"))
	deep := filepath.Join(dir, "conf", filepath.FromSlash(javaDeepRelativePath(javaTreeMaxDepth-1)))
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	if _, err := fingerprintJavaRoot(root); err != nil {
		t.Fatalf("closure at %d-component depth rejected: %v", javaTreeMaxDepth, err)
	}
	if err := os.Mkdir(filepath.Join(deep, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := fingerprintJavaRoot(root); err == nil || !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("closure beyond %d-component depth accepted: %v", javaTreeMaxDepth, err)
	}
}

func TestJavaClosureUsesOneAggregateEntryAndByteBudget(t *testing.T) {
	dir := makeTestJavaRoot(t, []byte("java"), []byte("modules"))
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	wantBytes := int64(len("java") + len("modules"))
	atLimit := javaTreeLimits{maxDepth: javaTreeMaxDepth, maxEntries: 5, maxBytes: wantBytes}
	names, _, gotBytes, err := inventoryJavaRootWithLimits(root, atLimit)
	if err != nil || len(names) != atLimit.maxEntries || gotBytes != wantBytes {
		t.Fatalf("aggregate closure boundary = %d entries, %d bytes, %v; want %d, %d", len(names), gotBytes, err, atLimit.maxEntries, wantBytes)
	}
	for _, tc := range []struct {
		name   string
		limits javaTreeLimits
		want   string
	}{
		{"entries", javaTreeLimits{maxDepth: javaTreeMaxDepth, maxEntries: atLimit.maxEntries - 1, maxBytes: wantBytes}, "entry limit"},
		{"bytes", javaTreeLimits{maxDepth: javaTreeMaxDepth, maxEntries: atLimit.maxEntries, maxBytes: wantBytes - 1}, "bytes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, err := inventoryJavaRootWithLimits(root, tc.limits); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("aggregate %s budget reset across Java subtrees: %v", tc.name, err)
			}
		})
	}
}

func TestFingerprintJavaRootRejectsContinuousAppender(t *testing.T) {
	dir := makeTestJavaRoot(t, []byte("java"), []byte(strings.Repeat("x", 1<<20)))
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	modules := filepath.Join("lib", "modules")
	done := make(chan struct{})
	stopped := make(chan struct{})
	started := false
	_, err = fingerprintJavaRootWithHook(root, func(name string) error {
		if name != modules || started {
			return nil
		}
		started = true
		first := make(chan struct{})
		go func() {
			defer close(stopped)
			f, openErr := os.OpenFile(filepath.Join(dir, modules), os.O_WRONLY|os.O_APPEND, 0)
			if openErr != nil {
				close(first)
				return
			}
			defer f.Close() //nolint:errcheck // test mutation
			for i := 0; ; i++ {
				_, _ = f.Write([]byte("growth"))
				if i == 0 {
					close(first)
				}
				select {
				case <-done:
					return
				default:
				}
			}
		}()
		<-first
		return nil
	})
	close(done)
	<-stopped
	if err == nil || !strings.Contains(err.Error(), "changed identity, metadata, or size while hashing") {
		t.Fatalf("continuous appender was accepted: %v", err)
	}
}

func makeTestJavaRoot(t *testing.T, launcher, modules []byte) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"bin", "conf", "lib"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "java"), launcher, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lib", "modules"), modules, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func javaDeepRelativePath(depth int) string {
	return strings.TrimSuffix(strings.Repeat("d/", depth), "/")
}
