// Package runtimeclosure binds external engine invocations to a verified Java
// runtime closure and a small deterministic environment.
package runtimeclosure

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const (
	JavaEnv                    = "MACHINERY_JAVA"
	JavaClosureSHAEnv          = "MACHINERY_JAVA_CLOSURE_SHA256"
	RequiredJavaMajor          = 21
	RequiredJavaFeature        = 0
	RequiredJavaSecurity       = 12
	RequiredJavaRelease        = PinnedJavaProbeVersion
	CIJavaVendor               = "Eclipse Adoptium"
	javaClosureMaxFiles        = 10_000
	javaClosureMaxBytes  int64 = 1 << 30
	javaLauncherMaxBytes int64 = 16 << 20
)

type Java struct {
	source      string
	path        string
	rootPath    string
	rootInfo    os.FileInfo
	info        os.FileInfo
	file        *os.File
	root        *os.Root
	body        []byte
	closureHash [sha256.Size]byte
	version     string
}

func OpenJava() (*Java, error) {
	source := os.Getenv(JavaEnv)
	explicit := source != ""
	if source == "" {
		var err error
		source, err = provisionedJavaPath()
		if err != nil {
			return nil, fmt.Errorf("provision pinned Java runtime: %w", err)
		}
	}
	abs, err := filepath.Abs(source)
	if err != nil {
		return nil, err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve java launcher: %w", err)
	}
	if filepath.Base(filepath.Dir(real)) != "bin" {
		return nil, fmt.Errorf("java launcher %s must reside in a JDK bin directory", source)
	}
	rootPath := filepath.Dir(filepath.Dir(real))
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, fmt.Errorf("open Java runtime root: %w", err)
	}
	rootInfo, err := os.Lstat(rootPath)
	if err != nil || !rootInfo.IsDir() {
		return nil, errors.Join(err, fmt.Errorf("Java runtime root must be a directory"), root.Close())
	}
	openedRoot, err := root.Lstat(".")
	if err != nil || !os.SameFile(rootInfo, openedRoot) {
		return nil, errors.Join(err, fmt.Errorf("Java runtime root changed identity while opening"), root.Close())
	}
	launcherName := filepath.Join("bin", filepath.Base(real))
	file, info, body, err := openJavaLauncher(root, launcherName, javaLauncherMaxBytes, nil)
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	closureHash, fingerprintErr := fingerprintJavaRoot(root)
	if fingerprintErr != nil {
		return nil, errors.Join(fingerprintErr, file.Close(), root.Close())
	}
	if explicit {
		want := os.Getenv(JavaClosureSHAEnv)
		got := fmt.Sprintf("%x", closureHash)
		if len(want) != sha256.Size*2 || want != strings.ToLower(want) {
			return nil, errors.Join(fmt.Errorf("%s requires paired exact lowercase sha256 in %s", JavaEnv, JavaClosureSHAEnv), file.Close(), root.Close())
		}
		if want != got {
			return nil, errors.Join(fmt.Errorf("explicit Java runtime closure sha256:%s does not match paired trust root sha256:%s", got, want), file.Close(), root.Close())
		}
	}
	return &Java{source: abs, path: real, rootPath: rootPath, rootInfo: rootInfo, info: info, file: file, root: root, body: body, closureHash: closureHash}, nil
}

// JavaClosureDigest returns the exact closure digest required alongside an
// explicit MACHINERY_JAVA override. Production configuration must record this
// value in source-controlled toolchain policy; it is not a semantic version.
func JavaClosureDigest(javaPath string) (string, error) {
	real, err := filepath.EvalSymlinks(javaPath)
	if err != nil {
		return "", err
	}
	if filepath.Base(filepath.Dir(real)) != "bin" {
		return "", fmt.Errorf("java launcher must reside in a JDK bin directory")
	}
	root, err := os.OpenRoot(filepath.Dir(filepath.Dir(real)))
	if err != nil {
		return "", err
	}
	digest, hashErr := fingerprintJavaRoot(root)
	if err := errors.Join(hashErr, root.Close()); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", digest), nil
}

func (j *Java) Path() string { return j.path }

func (j *Java) Identity() string {
	version := j.version
	if version == "" {
		version = RequiredJavaRelease
	}
	return fmt.Sprintf("Java %s closure sha256:%x", version, j.closureHash)
}

func (j *Java) BindIdentity(output string) error {
	properties, err := validateJavaProperties(output)
	if err != nil {
		return err
	}
	home, err := filepath.EvalSymlinks(properties["java.home"])
	if err != nil {
		return fmt.Errorf("resolve probed java.home: %w", err)
	}
	if home != j.rootPath {
		return fmt.Errorf("java probe reports java.home %q, want verified runtime root %q", home, j.rootPath)
	}
	j.version = properties["java.runtime.version"]
	return nil
}

func (j *Java) Validate() error {
	ambientRoot, ambientErr := os.Lstat(j.rootPath)
	retainedRoot, retainedErr := j.root.Lstat(".")
	if ambientErr != nil || retainedErr != nil || !ambientRoot.IsDir() || !retainedRoot.IsDir() || !os.SameFile(j.rootInfo, ambientRoot) || !os.SameFile(j.rootInfo, retainedRoot) {
		return errors.Join(ambientErr, retainedErr, fmt.Errorf("java runtime root changed identity after verification"))
	}
	resolved, err := filepath.EvalSymlinks(j.source)
	if err != nil || resolved != j.path {
		return errors.Join(err, fmt.Errorf("java launcher symlink chain changed after verification"))
	}
	launcherName := filepath.Join("bin", filepath.Base(j.path))
	ambientInfo, ambientErr := os.Lstat(j.path)
	pathInfo, pathErr := j.root.Lstat(launcherName)
	opened, openedErr := j.file.Stat()
	if ambientErr != nil || pathErr != nil || openedErr != nil || !sameJavaFileSnapshot(j.info, ambientInfo) || !sameJavaFileSnapshot(j.info, pathInfo) || !sameJavaFileSnapshot(j.info, opened) {
		return errors.Join(ambientErr, pathErr, openedErr, fmt.Errorf("java launcher changed identity after verification"))
	}
	if j.info.Size() < 0 || j.info.Size() > javaLauncherMaxBytes {
		return fmt.Errorf("java launcher changed identity after verification")
	}
	if _, err := j.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	again, readErr := io.ReadAll(io.LimitReader(j.file, j.info.Size()+1))
	afterOpen, statErr := j.file.Stat()
	afterPath, pathErr := j.root.Lstat(launcherName)
	if err := errors.Join(readErr, statErr, pathErr); err != nil {
		return err
	}
	if int64(len(again)) != j.info.Size() || !sameJavaFileSnapshot(j.info, afterOpen) || !sameJavaFileSnapshot(j.info, afterPath) {
		return fmt.Errorf("java launcher changed identity or size while revalidating")
	}
	if !bytes.Equal(j.body, again) {
		return fmt.Errorf("java launcher changed content after verification")
	}
	closureHash, err := fingerprintJavaRoot(j.root)
	if err != nil {
		return fmt.Errorf("revalidate Java runtime closure: %w", err)
	}
	if closureHash != j.closureHash {
		return fmt.Errorf("java runtime closure changed after verification (was sha256:%x, now sha256:%x)", j.closureHash, closureHash)
	}
	return nil
}

func (j *Java) Close() error { return errors.Join(j.file.Close(), j.root.Close()) }

func ValidateJavaIdentity(output string) error {
	_, err := validateJavaProperties(output)
	return err
}

func validateJavaProperties(output string) (map[string]string, error) {
	wanted := map[string]bool{"java.home": true, "java.runtime.version": true, "java.vendor": true, "java.version": true, "java.vm.name": true}
	properties := map[string]string{}
	continuationKey := ""
	for lineNumber, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || line == "Property settings:" {
			continuationKey = ""
			continue
		}
		key, value, ok := strings.Cut(line, " = ")
		if !ok && strings.HasSuffix(line, " =") {
			// java -XshowSettings:properties omits the trailing space for an
			// empty value (for example, the pinned runtime's empty
			// java.class.path). Preserve the same closed key vocabulary while
			// accepting that canonical empty-value spelling.
			key, value, ok = strings.TrimSuffix(line, " ="), "", true
		}
		if !ok {
			if continuationKey == "java.library.path" && strings.HasPrefix(raw, "        ") {
				continue
			}
			if canonicalJavaVersionLine(line) {
				continuationKey = ""
				continue
			}
			return nil, fmt.Errorf("java probe emitted unexpected diagnostic on line %d: %q", lineNumber+1, line)
		}
		if !allowedJavaPropertyKey(key) {
			return nil, fmt.Errorf("java probe emitted unexpected property on line %d: %q", lineNumber+1, line)
		}
		continuationKey = key
		if !wanted[key] {
			continue
		}
		if _, duplicate := properties[key]; duplicate {
			return nil, fmt.Errorf("java probe reports %s more than once", key)
		}
		properties[key] = strings.TrimSpace(value)
	}
	for _, key := range []string{"java.home", "java.runtime.version", "java.vendor", "java.version", "java.vm.name"} {
		if properties[key] == "" {
			return nil, fmt.Errorf("java probe did not report %s", key)
		}
	}
	version := properties["java.version"]
	if version != "21.0.12.1" {
		return nil, unsupportedJavaVersion(version)
	}
	parts := strings.FieldsFunc(version, func(r rune) bool { return r < '0' || r > '9' })
	if len(parts) < 3 {
		return nil, unsupportedJavaVersion(version)
	}
	want := []int{RequiredJavaMajor, RequiredJavaFeature, RequiredJavaSecurity, 1}
	for i, expected := range want {
		value, err := strconv.Atoi(parts[i])
		if err != nil || value != expected {
			return nil, unsupportedJavaVersion(version)
		}
	}
	if properties["java.runtime.version"] != PinnedJavaProbeVersion {
		return nil, fmt.Errorf("java runtime version %q is unsupported; require exact pinned build %s", properties["java.runtime.version"], PinnedJavaProbeVersion)
	}
	if properties["java.vendor"] != CIJavaVendor {
		return nil, fmt.Errorf("java vendor %q is unsupported; require exact pinned vendor %s", properties["java.vendor"], CIJavaVendor)
	}
	if properties["java.vm.name"] != "OpenJDK 64-Bit Server VM" {
		return nil, fmt.Errorf("java VM %q is unsupported; require exact pinned OpenJDK 64-Bit Server VM", properties["java.vm.name"])
	}
	return properties, nil
}

func allowedJavaPropertyKey(key string) bool {
	for _, prefix := range []string{
		"file.", "ftp.", "http.", "https.", "java.", "jdk.", "line.", "native.",
		"os.", "path.", "socks", "stderr.", "stdout.", "sun.", "user.",
	} {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func canonicalJavaVersionLine(line string) bool {
	const (
		versionLine = `openjdk version "21.0.12.1" 2026-08-18 LTS`
		runtimeLine = `OpenJDK Runtime Environment Temurin-21.0.12.1+1 (build 21.0.12.1+1-LTS)`
		vmLine      = `OpenJDK 64-Bit Server VM Temurin-21.0.12.1+1 (build 21.0.12.1+1-LTS, mixed mode, sharing)`
	)
	return line == versionLine || line == runtimeLine || line == vmLine
}

func unsupportedJavaVersion(version string) error {
	return fmt.Errorf("java version %q is unsupported; require exact pinned Temurin build %s", version, RequiredJavaRelease)
}

func openJavaLauncher(root *os.Root, name string, maxBytes int64, afterOpen func() error) (*os.File, os.FileInfo, []byte, error) {
	if maxBytes <= 0 {
		return nil, nil, nil, fmt.Errorf("java launcher byte limit must be positive")
	}
	before, err := root.Lstat(name)
	if err != nil {
		return nil, nil, nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, nil, nil, fmt.Errorf("java launcher must resolve to a regular non-symlink file")
	}
	if before.Size() < 0 || before.Size() > maxBytes {
		return nil, nil, nil, fmt.Errorf("java launcher exceeds %d-byte limit", maxBytes)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, nil, nil, err
	}
	opened, statErr := file.Stat()
	if statErr != nil || !sameJavaFileSnapshot(before, opened) {
		return nil, nil, nil, errors.Join(statErr, fmt.Errorf("java launcher changed identity or metadata while opening"), file.Close())
	}
	if afterOpen != nil {
		if err := afterOpen(); err != nil {
			return nil, nil, nil, errors.Join(err, file.Close())
		}
	}
	body, readErr := io.ReadAll(io.LimitReader(file, before.Size()+1))
	afterInfo, statErr := file.Stat()
	afterPath, pathErr := root.Lstat(name)
	if err := errors.Join(readErr, statErr, pathErr); err != nil {
		return nil, nil, nil, errors.Join(err, file.Close())
	}
	if int64(len(body)) != before.Size() || !sameJavaFileSnapshot(before, afterInfo) || !sameJavaFileSnapshot(before, afterPath) {
		return nil, nil, nil, errors.Join(fmt.Errorf("java launcher changed identity, metadata, or size while reading"), file.Close())
	}
	return file, afterPath, body, nil
}

func sameJavaFileSnapshot(before, after os.FileInfo) bool {
	return before != nil && after != nil && os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size() && before.ModTime().Equal(after.ModTime()) && javaFileChangeID(before) == javaFileChangeID(after)
}

func javaFileChangeID(info os.FileInfo) string {
	if info == nil || info.Sys() == nil {
		return ""
	}
	value := reflect.Indirect(reflect.ValueOf(info.Sys()))
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return ""
	}
	for _, name := range []string{"Ctim", "Ctimespec"} {
		field := value.FieldByName(name)
		if !field.IsValid() || field.Kind() != reflect.Struct {
			continue
		}
		sec, nsec := field.FieldByName("Sec"), field.FieldByName("Nsec")
		if sec.IsValid() && nsec.IsValid() && sec.CanInt() && nsec.CanInt() {
			return fmt.Sprintf("ctime:%d:%d", sec.Int(), nsec.Int())
		}
	}
	sec, nsec := value.FieldByName("Ctime"), value.FieldByName("Ctimensec")
	if sec.IsValid() && nsec.IsValid() && sec.CanInt() && nsec.CanInt() {
		return fmt.Sprintf("ctime:%d:%d", sec.Int(), nsec.Int())
	}
	return ""
}

func fingerprintJavaRoot(root *os.Root) ([sha256.Size]byte, error) {
	return fingerprintJavaRootWithHook(root, nil)
}

func fingerprintJavaRootWithHook(root *os.Root, afterOpen func(string) error) ([sha256.Size]byte, error) {
	var zero [sha256.Size]byte
	var names []string
	for _, subtree := range []string{"bin", "conf", "lib"} {
		info, err := root.Lstat(subtree)
		if err != nil {
			return zero, fmt.Errorf("inventory Java runtime %s: %w", subtree, err)
		}
		if !info.IsDir() {
			return zero, fmt.Errorf("Java runtime %s must be a directory", subtree)
		}
		if err := fs.WalkDir(root.FS(), subtree, func(name string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if name != subtree {
				names = append(names, name)
			}
			return nil
		}); err != nil {
			return zero, fmt.Errorf("inventory Java runtime %s: %w", subtree, err)
		}
	}
	sort.Strings(names)
	if len(names) > javaClosureMaxFiles {
		return zero, fmt.Errorf("Java runtime closure has %d entries; limit is %d", len(names), javaClosureMaxFiles)
	}
	infos := make(map[string]os.FileInfo, len(names))
	var censusTotal int64
	for _, name := range names {
		info, err := root.Lstat(name)
		if err != nil {
			return zero, fmt.Errorf("stat Java runtime closure entry %s: %w", name, err)
		}
		infos[name] = info
		if info.IsDir() {
			continue
		}
		slashName := filepath.ToSlash(name)
		if !info.Mode().IsRegular() {
			return zero, fmt.Errorf("Java runtime closure entry %s must be a regular file or directory", slashName)
		}
		if info.Size() < 0 || info.Size() > javaClosureMaxBytes-censusTotal {
			return zero, fmt.Errorf("Java runtime closure exceeds %d bytes", javaClosureMaxBytes)
		}
		censusTotal += info.Size()
	}
	hash := sha256.New()
	var hashedTotal int64
	for _, name := range names {
		info := infos[name]
		slashName := filepath.ToSlash(name)
		if info.IsDir() {
			_, _ = fmt.Fprintf(hash, "d\x00%s\x00", slashName)
			continue
		}
		_, _ = fmt.Fprintf(hash, "f\x00%s\x00%d\x00", slashName, info.Size())
		size, err := hashJavaClosureFile(root, name, info, hash, info.Size(), afterOpen)
		if err != nil {
			return zero, fmt.Errorf("hash Java runtime closure entry %s: %w", slashName, err)
		}
		hashedTotal += size
	}
	if hashedTotal != censusTotal {
		return zero, fmt.Errorf("Java runtime closure hashed %d bytes after census recorded %d", hashedTotal, censusTotal)
	}
	if info, err := root.Lstat(filepath.Join("lib", "modules")); err != nil || !info.Mode().IsRegular() {
		return zero, errors.Join(err, fmt.Errorf("Java runtime closure requires regular lib/modules"))
	}
	var result [sha256.Size]byte
	copy(result[:], hash.Sum(nil))
	return result, nil
}

func hashJavaClosureFile(root *os.Root, name string, before os.FileInfo, hash io.Writer, maxBytes int64, afterOpen func(string) error) (int64, error) {
	if before.Size() < 0 || before.Size() > maxBytes {
		return 0, fmt.Errorf("file exceeds remaining %d-byte closure limit", maxBytes)
	}
	file, err := root.Open(name)
	if err != nil {
		return 0, err
	}
	opened, statErr := file.Stat()
	if statErr != nil || !sameJavaFileSnapshot(before, opened) {
		return 0, errors.Join(statErr, fmt.Errorf("changed identity or metadata while opening"), file.Close())
	}
	if afterOpen != nil {
		if err := afterOpen(name); err != nil {
			return 0, errors.Join(err, file.Close())
		}
	}
	written, copyErr := io.Copy(hash, io.LimitReader(file, before.Size()+1))
	afterInfo, statErr := file.Stat()
	closeErr := file.Close()
	afterPath, pathErr := root.Lstat(name)
	if err := errors.Join(copyErr, statErr, closeErr, pathErr); err != nil {
		return 0, err
	}
	if written != before.Size() || !sameJavaFileSnapshot(before, afterInfo) || !sameJavaFileSnapshot(before, afterPath) {
		return 0, fmt.Errorf("changed identity, metadata, or size while hashing")
	}
	return written, nil
}

func Environment(home, temp, javaPath string) []string {
	path := filepath.Dir(javaPath)
	if runtime.GOOS == "windows" {
		path += string(os.PathListSeparator) + filepath.Join(os.Getenv("SystemRoot"), "System32")
	} else {
		path += string(os.PathListSeparator) + "/usr/bin" + string(os.PathListSeparator) + "/bin"
	}
	env := []string{
		"HOME=" + home,
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
		"PATH=" + path,
		"TEMP=" + temp,
		"TMP=" + temp,
		"TMPDIR=" + temp,
		"TZ=UTC",
	}
	if runtime.GOOS == "windows" {
		for _, key := range []string{"SystemRoot", "WINDIR"} {
			if value := os.Getenv(key); value != "" {
				env = append(env, key+"="+value)
			}
		}
	}
	return env
}
