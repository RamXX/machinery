package runtimeclosure

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/cachestage"
)

func javaProvisionTestBase(t *testing.T) string {
	t.Helper()
	cacheRoot := t.TempDir()
	t.Setenv("HOME", cacheRoot)
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	t.Setenv("LOCALAPPDATA", cacheRoot)
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(cache, "machinery", "java")
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatal(err)
	}
	return base
}

func writeTarGzip(t *testing.T, entries []tar.Header, bodies map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "runtime.tar.gz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	for i := range entries {
		header := entries[i]
		body := []byte(bodies[header.Name])
		header.Size = int64(len(body))
		if err := tw.WriteHeader(&header); err != nil {
			t.Fatal(err)
		}
		if len(body) > 0 {
			if _, err := tw.Write(body); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

type javaZipTestEntry struct {
	name string
	body string
	mode os.FileMode
}

func writeJavaZip(t *testing.T, entries []javaZipTestEntry) string {
	t.Helper()
	archive := filepath.Join(t.TempDir(), "runtime.zip")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Store}
		header.SetMode(entry.mode)
		out, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := out.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := errors.Join(writer.Close(), file.Close()); err != nil {
		t.Fatal(err)
	}
	return archive
}

func TestProvisionOfficialJavaArchiveAndReuseCache(t *testing.T) {
	if os.Getenv("MACHINERY_TEST_OFFICIAL_JAVA") != "1" {
		t.Skip("set MACHINERY_TEST_OFFICIAL_JAVA=1 for the official archive/cache contract")
	}
	t.Setenv(JavaEnv, "")
	first, err := OpenJava()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	cmd := exec.CommandContext(ctx, first.Path(), "-XshowSettings:properties", "-version")
	cmd.Env = Environment(t.TempDir(), t.TempDir(), first.Path())
	out, probeErr := cmd.CombinedOutput()
	cancel()
	if probeErr != nil {
		t.Fatalf("probe provisioned Java: %v", probeErr)
	}
	if err := first.BindIdentity(string(out)); err != nil {
		t.Fatal(err)
	}
	firstPath, firstIdentity := first.Path(), first.Identity()
	if err := errors.Join(first.Validate(), first.Close()); err != nil {
		t.Fatal(err)
	}
	second, err := OpenJava()
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if second.Path() != firstPath || second.Identity() != firstIdentity {
		t.Fatalf("cache reuse changed runtime identity:\nfirst %s %s\nsecond %s %s", firstPath, firstIdentity, second.Path(), second.Identity())
	}
}

func TestProvisionJavaRecoversCrashStageBeforeCacheUse(t *testing.T) {
	if _, supported := javaArchivePins[runtime.GOOS+"/"+runtime.GOARCH]; !supported {
		t.Skip("no Java archive pin for this test platform")
	}
	base := javaProvisionTestBase(t)
	stage := filepath.Join(base, ".java-stage-123456")
	if err := os.MkdirAll(filepath.Join(stage, "extracted", "lib"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "runtime.archive"), []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, strings.ReplaceAll(PinnedJavaRuntimeVersion, "+", "_"), runtime.GOOS+"-"+runtime.GOARCH)
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	_, firstErr := provisionedJavaPath()
	if firstErr == nil || !strings.Contains(firstErr.Error(), "cache is malformed") {
		t.Fatalf("retry did not reach deterministic cache validation after recovery: %v", firstErr)
	}
	if _, err := os.Lstat(stage); !os.IsNotExist(err) {
		t.Fatalf("SIGKILL-equivalent Java stage survived locked retry: %v", err)
	}
	_, secondErr := provisionedJavaPath()
	if secondErr == nil || secondErr.Error() != firstErr.Error() {
		t.Fatalf("post-recovery retry diagnostic changed:\nfirst:  %v\nsecond: %v", firstErr, secondErr)
	}
}

func TestProvisionJavaFailsClosedOnUnsafeCrashStage(t *testing.T) {
	if _, supported := javaArchivePins[runtime.GOOS+"/"+runtime.GOARCH]; !supported {
		t.Skip("no Java archive pin for this test platform")
	}
	base := javaProvisionTestBase(t)
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(base, ".java-stage-123")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := provisionedJavaPath(); err == nil || !strings.Contains(err.Error(), "private real directory") {
		t.Fatalf("unsafe Java stage did not fail closed: %v", err)
	}
	if body, err := os.ReadFile(outside); err != nil || string(body) != "sentinel" {
		t.Fatalf("unsafe-stage recovery touched outside sentinel: %q, %v", body, err)
	}
}

func TestProvisionJavaRetryConvergesAfterCompleteTargetRename(t *testing.T) {
	pin, supported := javaArchivePins[runtime.GOOS+"/"+runtime.GOARCH]
	if !supported {
		t.Skip("no Java archive pin for this test platform")
	}
	base := javaProvisionTestBase(t)
	target := filepath.Join(base, strings.ReplaceAll(PinnedJavaRuntimeVersion, "+", "_"), runtime.GOOS+"-"+runtime.GOARCH)
	home := filepath.Join(target, "jdk")
	if runtime.GOOS == "darwin" {
		home = filepath.Join(home, "Contents", "Home")
	}
	for _, dir := range []string{"bin", "conf", "lib"} {
		if err := os.MkdirAll(filepath.Join(home, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(javaLauncher(home), []byte("complete launcher"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "conf", "security.properties"), []byte("complete config"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "lib", "modules"), []byte("complete modules"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(home)
	if err != nil {
		t.Fatal(err)
	}
	closure, hashErr := fingerprintJavaRoot(root)
	closeErr := root.Close()
	if err := errors.Join(hashErr, closeErr); err != nil {
		t.Fatal(err)
	}
	receipt := fmt.Sprintf("archive_sha256=%s\nclosure_sha256=%x\nversion=%s\n", pin.sha, closure, PinnedJavaRuntimeVersion)
	if err := os.WriteFile(filepath.Join(home, ".machinery-java-receipt"), []byte(receipt), 0o600); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(base, ".java-stage-77")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "runtime.archive"), []byte("post-rename residue"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := provisionedJavaPath()
	if err != nil || got != javaLauncher(home) {
		t.Fatalf("retry did not converge to the complete renamed target: path=%s err=%v", got, err)
	}
	if _, err := os.Lstat(stage); !os.IsNotExist(err) {
		t.Fatalf("retry retained post-rename Java stage residue: %v", err)
	}
}

func TestExtractJavaTarMaterializesOnlyConfinedRelativeSymlinks(t *testing.T) {
	archive := writeTarGzip(t, []tar.Header{
		{Name: "jdk/lib/real", Typeflag: tar.TypeReg, Mode: 0o644},
		{Name: "jdk/lib/link", Typeflag: tar.TypeSymlink, Linkname: "real", Mode: 0o777},
	}, map[string]string{"jdk/lib/real": "verified"})
	destination := t.TempDir()
	if err := extractJavaTarGzip(archive, destination); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(destination, "jdk", "lib", "link"))
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("safe link was not materialized as a regular file: %v, %v", info, err)
	}
	body, err := os.ReadFile(filepath.Join(destination, "jdk", "lib", "link"))
	if err != nil || string(body) != "verified" {
		t.Fatalf("materialized link = %q, %v", body, err)
	}
}

func TestExtractJavaTarRejectsEscapingAndHardLinks(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header tar.Header
		want   string
	}{
		{"escaping symlink", tar.Header{Name: "jdk/lib/link", Typeflag: tar.TypeSymlink, Linkname: "../../../outside"}, "escapes"},
		{"absolute symlink", tar.Header{Name: "jdk/lib/link", Typeflag: tar.TypeSymlink, Linkname: "/outside"}, "absolute"},
		{"hard link", tar.Header{Name: "jdk/lib/link", Typeflag: tar.TypeLink, Linkname: "jdk/lib/real"}, "link or special"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			archive := writeTarGzip(t, []tar.Header{tc.header}, nil)
			err := extractJavaTarGzip(archive, t.TempDir())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unsafe archive accepted or wrong error: %v", err)
			}
		})
	}
}

func TestJavaArchiveDepthBoundaryIsPortableAndExact(t *testing.T) {
	atLimit := javaDeepRelativePath(javaTreeMaxDepth-1) + "/payload"
	overLimit := javaDeepRelativePath(javaTreeMaxDepth) + "/payload"
	tests := []struct {
		name    string
		archive func(string) string
		extract func(string, string) error
	}{
		{
			name: "tar-gzip",
			archive: func(name string) string {
				return writeTarGzip(t, []tar.Header{{Name: name, Typeflag: tar.TypeReg, Mode: 0o644}}, map[string]string{name: "x"})
			},
			extract: extractJavaTarGzip,
		},
		{
			name: "zip",
			archive: func(name string) string {
				return writeJavaZip(t, []javaZipTestEntry{{name: name, body: "x", mode: 0o644}})
			},
			extract: extractJavaZip,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name+"/at-limit", func(t *testing.T) {
			destination := t.TempDir()
			if err := tc.extract(tc.archive(atLimit), destination); err != nil {
				t.Fatalf("archive at %d-component depth rejected: %v", javaTreeMaxDepth, err)
			}
			if body, err := os.ReadFile(filepath.Join(destination, filepath.FromSlash(atLimit))); err != nil || string(body) != "x" {
				t.Fatalf("at-limit archive payload = %q, %v", body, err)
			}
		})
		t.Run(tc.name+"/over-limit", func(t *testing.T) {
			destination := t.TempDir()
			err := tc.extract(tc.archive(overLimit), destination)
			if err == nil || !strings.Contains(err.Error(), "depth limit") {
				t.Fatalf("archive beyond %d-component depth accepted: %v", javaTreeMaxDepth, err)
			}
			if _, statErr := os.Lstat(filepath.Join(destination, filepath.FromSlash(overLimit))); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("over-depth archive entry was materialized: %v", statErr)
			}
		})
	}
}

func TestJavaArchiveUsesOneAggregateEntryAndByteBudget(t *testing.T) {
	tests := []struct {
		name    string
		archive func() string
		extract func(string, string, javaTreeLimits) error
	}{
		{
			name: "tar-gzip",
			archive: func() string {
				return writeTarGzip(t, []tar.Header{
					{Name: "a/one", Typeflag: tar.TypeReg, Mode: 0o644},
					{Name: "b/two", Typeflag: tar.TypeReg, Mode: 0o644},
				}, map[string]string{"a/one": "x", "b/two": "y"})
			},
			extract: extractJavaTarGzipWithLimits,
		},
		{
			name: "zip",
			archive: func() string {
				return writeJavaZip(t, []javaZipTestEntry{
					{name: "a/one", body: "x", mode: 0o644},
					{name: "b/two", body: "y", mode: 0o644},
				})
			},
			extract: extractJavaZipWithLimits,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name+"/at-limit", func(t *testing.T) {
			if err := tc.extract(tc.archive(), t.TempDir(), javaTreeLimits{maxDepth: 2, maxEntries: 4, maxBytes: 2}); err != nil {
				t.Fatalf("aggregate archive boundary rejected: %v", err)
			}
		})
		t.Run(tc.name+"/entry-over-limit", func(t *testing.T) {
			err := tc.extract(tc.archive(), t.TempDir(), javaTreeLimits{maxDepth: 2, maxEntries: 3, maxBytes: 2})
			if err == nil || !strings.Contains(err.Error(), "entry limit") {
				t.Fatalf("archive entry budget reset by directory: %v", err)
			}
		})
		t.Run(tc.name+"/byte-over-limit", func(t *testing.T) {
			err := tc.extract(tc.archive(), t.TempDir(), javaTreeLimits{maxDepth: 2, maxEntries: 4, maxBytes: 1})
			if err == nil || !strings.Contains(err.Error(), "bytes") {
				t.Fatalf("archive byte budget reset by directory: %v", err)
			}
		})
	}
}

func TestTarSymlinkMaterializationConsumesAggregateByteBudget(t *testing.T) {
	archive := writeTarGzip(t, []tar.Header{
		{Name: "jdk/real", Typeflag: tar.TypeReg, Mode: 0o644},
		{Name: "jdk/link", Typeflag: tar.TypeSymlink, Linkname: "real", Mode: 0o777},
	}, map[string]string{"jdk/real": "x"})
	destination := t.TempDir()
	err := extractJavaTarGzipWithLimits(archive, destination, javaTreeLimits{maxDepth: 2, maxEntries: 3, maxBytes: 1})
	if err == nil || !strings.Contains(err.Error(), "bytes") {
		t.Fatalf("materialized symlink bypassed aggregate archive byte budget: %v", err)
	}
	if _, statErr := os.Lstat(filepath.Join(destination, "jdk", "link")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("over-budget symlink was materialized: %v", statErr)
	}
}

func TestProvisionJavaRejectsOverDepthArchiveBeforePublication(t *testing.T) {
	if _, supported := javaArchivePins[runtime.GOOS+"/"+runtime.GOARCH]; !supported {
		t.Skip("no Java archive pin for this test platform")
	}
	base := javaProvisionTestBase(t)
	overLimit := javaDeepRelativePath(javaTreeMaxDepth) + "/payload"
	archive := writeTarGzip(t, []tar.Header{{Name: overLimit, Typeflag: tar.TypeReg, Mode: 0o644}}, map[string]string{overLimit: "x"})
	originalDownload := downloadJavaRuntimeArchive
	originalPublish := publishJavaRuntimeTree
	originalPin := javaArchivePins[runtime.GOOS+"/"+runtime.GOARCH]
	t.Cleanup(func() {
		downloadJavaRuntimeArchive = originalDownload
		publishJavaRuntimeTree = originalPublish
		javaArchivePins[runtime.GOOS+"/"+runtime.GOARCH] = originalPin
	})
	javaArchivePins[runtime.GOOS+"/"+runtime.GOARCH] = javaArchivePin{asset: "test.tar.gz", sha: strings.Repeat("0", 64)}
	downloadJavaRuntimeArchive = func(_, destination, _ string) error {
		body, err := os.ReadFile(archive)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, body, 0o600)
	}
	published := false
	publishJavaRuntimeTree = func(base, source, target string) error {
		published = true
		return cachestage.PublishTree(base, source, target)
	}
	if _, err := provisionedJavaPath(); err == nil || !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("over-depth Java archive reached publication: %v", err)
	}
	if published {
		t.Fatal("over-depth Java archive invoked cache publication")
	}
	target := filepath.Join(base, strings.ReplaceAll(PinnedJavaRuntimeVersion, "+", "_"), runtime.GOOS+"-"+runtime.GOARCH)
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("over-depth Java archive created public install target: %v", err)
	}
}

func TestDownloadPinnedArchiveChecksumFailureLeavesNoSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write(bytes.Repeat([]byte("x"), 1024))
	}))
	defer server.Close()
	destination := filepath.Join(t.TempDir(), "archive")
	err := downloadPinnedArchive(server.URL, destination, strings.Repeat("0", 64))
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("checksum failure accepted: %v", err)
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		t.Fatalf("checksum failure left a partial archive: %v", err)
	}
}
