package runtimeclosure

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/cachestage"
	"github.com/RamXX/machinery/internal/filelock"
)

const (
	PinnedJavaRuntimeVersion = "21.0.12.1+1"
	PinnedJavaProbeVersion   = "21.0.12.1+1-LTS"
	pinnedJavaReleaseTag     = "jdk-21.0.12.1%2B1"
	javaArchiveMaxBytes      = int64(400 << 20)
	javaExtractMaxBytes      = int64(2 << 30)
	javaExtractMaxFiles      = 30_000
)

var javaArchiveLimits = javaTreeLimits{maxDepth: javaTreeMaxDepth, maxEntries: javaExtractMaxFiles, maxBytes: javaExtractMaxBytes}

type javaArchiveBudget struct {
	tree        javaTreeBudget
	directories map[string]struct{}
}

func newJavaArchiveBudget(limits javaTreeLimits) *javaArchiveBudget {
	return &javaArchiveBudget{
		tree:        javaTreeBudget{label: "Java runtime archive", limits: limits},
		directories: make(map[string]struct{}),
	}
}

// addEntry charges both the archive member and every implicit parent directory
// that extraction may materialize through MkdirAll. Explicit directory headers
// mark their path so later children do not reset or double-spend parent state.
func (budget *javaArchiveBudget) addEntry(name string, size int64, directory bool) error {
	slashName := filepath.ToSlash(name)
	parts := strings.Split(slashName, "/")
	for index := 1; index < len(parts); index++ {
		parent := strings.Join(parts[:index], "/")
		if _, exists := budget.directories[parent]; exists {
			continue
		}
		if err := budget.tree.addEntry(filepath.FromSlash(parent), 0); err != nil {
			return err
		}
		budget.directories[parent] = struct{}{}
	}
	if err := budget.tree.addEntry(name, size); err != nil {
		return err
	}
	if directory {
		budget.directories[slashName] = struct{}{}
	}
	return nil
}

type javaArchivePin struct {
	asset string
	sha   string
	zip   bool
}

var javaArchivePins = map[string]javaArchivePin{
	"linux/arm64":   {"OpenJDK21U-jdk_aarch64_linux_hotspot_21.0.12.1_1.tar.gz", "23e37e026f12f3e706f18938ff611db3032d075b09d0879a25d06718c773e223", false},
	"darwin/arm64":  {"OpenJDK21U-jdk_aarch64_mac_hotspot_21.0.12.1_1.tar.gz", "3623232f33a9c3baadf304480b2535f9a3cba8a58d42ecbb438ba267315d9998", false},
	"windows/arm64": {"OpenJDK21U-jdk_aarch64_windows_hotspot_21.0.12.1_1.zip", "ccf2e51f527d542a70ba5794a600d3aac04b4e967950e227834c7566cb1bec7b", true},
	"linux/amd64":   {"OpenJDK21U-jdk_x64_linux_hotspot_21.0.12.1_1.tar.gz", "ce79869e1307ed8ee1e2baa86a412b1eb5b75d10a01006d788a6f968bcfaee94", false},
	"darwin/amd64":  {"OpenJDK21U-jdk_x64_mac_hotspot_21.0.12.1_1.tar.gz", "44db0f08196daf19a47f90d13388b0c943b67663cb537f998fe29e836fa842ce", false},
	"windows/amd64": {"OpenJDK21U-jdk_x64_windows_hotspot_21.0.12.1_1.zip", "f9d6e191ab098c0d416e7d588a24420a8621cd2f4720dab2459b8b7b2d2d8b4e", true},
}

func provisionedJavaPath() (path string, retErr error) {
	pin, ok := javaArchivePins[runtime.GOOS+"/"+runtime.GOARCH]
	if !ok {
		return "", fmt.Errorf("no pinned Java runtime archive for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve Java runtime cache: %w", err)
	}
	base := filepath.Join(cache, "machinery", "java")
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", err
	}
	lock, err := filelock.AcquireWait(base)
	if err != nil {
		return "", err
	}
	defer func() { retErr = errors.Join(retErr, lock.Release()) }()
	if err := cachestage.Recover(base, ".java-stage-"); err != nil {
		return "", fmt.Errorf("recover interrupted Java runtime provision: %w", err)
	}
	target := filepath.Join(base, strings.ReplaceAll(PinnedJavaRuntimeVersion, "+", "_"), runtime.GOOS+"-"+runtime.GOARCH)
	if home, err := javaHomeInInstall(target); err == nil {
		if err := validateProvisionedJavaHome(home, pin); err != nil {
			return "", fmt.Errorf("pinned Java runtime cache failed receipt validation: %w; remove %s and retry", err, target)
		}
		return javaLauncher(home), nil
	} else if _, statErr := os.Lstat(target); statErr == nil {
		return "", fmt.Errorf("pinned Java runtime cache is malformed: %w; remove %s and retry", err, target)
	}
	stage, err := os.MkdirTemp(base, ".java-stage-")
	if err != nil {
		return "", err
	}
	defer func() {
		if err := cachestage.Recover(base, ".java-stage-"); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("clean Java runtime provision stage: %w", err))
		}
	}()
	archive := filepath.Join(stage, "runtime.archive")
	url := "https://github.com/adoptium/temurin21-binaries/releases/download/" + pinnedJavaReleaseTag + "/" + pin.asset
	if err := downloadJavaRuntimeArchive(url, archive, pin.sha); err != nil {
		return "", err
	}
	extracted := filepath.Join(stage, "extracted")
	if err := os.Mkdir(extracted, 0o700); err != nil {
		return "", err
	}
	if pin.zip {
		err = extractJavaZip(archive, extracted)
	} else {
		err = extractJavaTarGzip(archive, extracted)
	}
	if err != nil {
		return "", err
	}
	home, err := javaHomeInInstall(extracted)
	if err != nil {
		return "", err
	}
	root, err := os.OpenRoot(home)
	if err != nil {
		return "", err
	}
	closure, fingerprintErr := fingerprintJavaRoot(root)
	closeErr := root.Close()
	if err := errors.Join(fingerprintErr, closeErr); err != nil {
		return "", err
	}
	receipt := []byte("archive_sha256=" + pin.sha + "\nclosure_sha256=" + hex.EncodeToString(closure[:]) + "\nversion=" + PinnedJavaRuntimeVersion + "\n")
	if err := os.WriteFile(filepath.Join(home, ".machinery-java-receipt"), receipt, 0o600); err != nil {
		return "", err
	}
	sourceRel, err := filepath.Rel(base, extracted)
	if err != nil {
		return "", err
	}
	targetRel, err := filepath.Rel(base, target)
	if err != nil {
		return "", err
	}
	if err := publishJavaRuntimeTree(base, sourceRel, targetRel); err != nil {
		return "", fmt.Errorf("publish pinned Java runtime: %w", err)
	}
	home, err = javaHomeInInstall(target)
	if err != nil {
		return "", err
	}
	if err := validateProvisionedJavaHome(home, pin); err != nil {
		return "", err
	}
	return javaLauncher(home), nil
}

var (
	downloadJavaRuntimeArchive = downloadPinnedArchive
	publishJavaRuntimeTree     = cachestage.PublishTree
)

func validateProvisionedJavaHome(home string, pin javaArchivePin) error {
	body, err := readProvisionedRegularFile(filepath.Join(home, ".machinery-java-receipt"), 1024)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	if len(lines) != 3 || lines[0] != "archive_sha256="+pin.sha || lines[2] != "version="+PinnedJavaRuntimeVersion || !strings.HasPrefix(lines[1], "closure_sha256=") {
		return fmt.Errorf("Java runtime receipt is malformed or does not match the committed archive pin")
	}
	want, err := hex.DecodeString(strings.TrimPrefix(lines[1], "closure_sha256="))
	if err != nil || len(want) != sha256.Size {
		return fmt.Errorf("Java runtime receipt closure hash is malformed")
	}
	root, err := os.OpenRoot(home)
	if err != nil {
		return err
	}
	got, hashErr := fingerprintJavaRoot(root)
	closeErr := root.Close()
	if err := errors.Join(hashErr, closeErr); err != nil {
		return err
	}
	if !strings.EqualFold(hex.EncodeToString(got[:]), hex.EncodeToString(want)) {
		return fmt.Errorf("Java runtime cache closure does not match its verified archive receipt")
	}
	return nil
}

func downloadPinnedArchive(url, destination, want string) (retErr error) {
	defer func() {
		if retErr != nil {
			removeErr := os.Remove(destination)
			if os.IsNotExist(removeErr) {
				removeErr = nil
			}
			retErr = errors.Join(retErr, removeErr)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("download pinned Java runtime: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download pinned Java runtime: HTTP %s", response.Status)
	}
	if response.ContentLength > javaArchiveMaxBytes {
		return fmt.Errorf("pinned Java runtime archive exceeds %d bytes", javaArchiveMaxBytes)
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, javaArchiveMaxBytes+1))
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(copyErr, syncErr, closeErr); err != nil {
		return err
	}
	if written > javaArchiveMaxBytes {
		return fmt.Errorf("pinned Java runtime archive exceeds %d bytes", javaArchiveMaxBytes)
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if got != want {
		return fmt.Errorf("pinned Java runtime archive checksum mismatch: got %s, want %s", got, want)
	}
	return nil
}

func safeArchiveName(name string) (string, error) {
	if name == "" || strings.Contains(name, `\`) {
		return "", fmt.Errorf("Java runtime archive contains unsafe path %q", name)
	}
	cleanSlash := path.Clean(name)
	first, _, _ := strings.Cut(cleanSlash, "/")
	if strings.HasPrefix(name, "/") || cleanSlash == "." || cleanSlash == ".." || strings.HasPrefix(cleanSlash, "../") || strings.Contains(first, ":") {
		return "", fmt.Errorf("Java runtime archive contains unsafe path %q", name)
	}
	return filepath.FromSlash(cleanSlash), nil
}

func extractJavaTarGzip(archive, destination string) error {
	return extractJavaTarGzipWithLimits(archive, destination, javaArchiveLimits)
}

func extractJavaTarGzipWithLimits(archive, destination string, limits javaTreeLimits) error {
	if err := validateJavaTreeLimits("Java runtime archive", limits); err != nil {
		return err
	}
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	budget := newJavaArchiveBudget(limits)
	type pendingLink struct{ name, target string }
	var links []pendingLink
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		name, err := safeArchiveName(header.Name)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := budget.addEntry(name, 0, true); err != nil {
				return err
			}
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := budget.addEntry(name, header.Size, false); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return err
			}
			mode := os.FileMode(0o600)
			if header.FileInfo().Mode()&0o111 != 0 {
				mode = 0o700
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.CopyN(out, reader, header.Size)
			if err := errors.Join(copyErr, out.Close()); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := budget.addEntry(name, 0, false); err != nil {
				return err
			}
			if strings.Contains(header.Linkname, `\`) || path.IsAbs(header.Linkname) {
				return fmt.Errorf("Java runtime archive symlink %q has absolute target %q", header.Name, header.Linkname)
			}
			targetSlash := path.Clean(path.Join(path.Dir(filepath.ToSlash(name)), header.Linkname))
			if targetSlash == ".." || strings.HasPrefix(targetSlash, "../") {
				return fmt.Errorf("Java runtime archive symlink %q escapes through target %q", header.Name, header.Linkname)
			}
			targetName, err := safeArchiveName(targetSlash)
			if err != nil {
				return fmt.Errorf("Java runtime archive symlink %q has unsafe target %q: %w", header.Name, header.Linkname, err)
			}
			links = append(links, pendingLink{name: name, target: targetName})
		default:
			return fmt.Errorf("Java runtime archive contains link or special entry %q", header.Name)
		}
	}
	for _, link := range links {
		source := filepath.Join(destination, link.target)
		info, err := os.Lstat(source)
		if err != nil || !info.Mode().IsRegular() {
			return errors.Join(err, fmt.Errorf("Java runtime archive symlink %q must resolve to an extracted regular file", link.name))
		}
		if err := budget.tree.addBytes(link.name, info.Size()); err != nil {
			return err
		}
		body, err := readProvisionedRegularFile(source, info.Size())
		if err != nil {
			return err
		}
		target := filepath.Join(destination, link.name)
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		mode := os.FileMode(0o600)
		if info.Mode()&0o111 != 0 {
			mode = 0o700
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			return err
		}
		written, writeErr := out.Write(body)
		if writeErr == nil && written != len(body) {
			writeErr = io.ErrShortWrite
		}
		if err := errors.Join(writeErr, out.Close()); err != nil {
			return err
		}
	}
	return nil
}

func extractJavaZip(archive, destination string) error {
	return extractJavaZipWithLimits(archive, destination, javaArchiveLimits)
}

func extractJavaZipWithLimits(archive, destination string, limits javaTreeLimits) error {
	if err := validateJavaTreeLimits("Java runtime archive", limits); err != nil {
		return err
	}
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()
	budget := newJavaArchiveBudget(limits)
	for _, entry := range reader.File {
		name, err := safeArchiveName(entry.Name)
		if err != nil {
			return err
		}
		mode := entry.Mode()
		if mode&os.ModeSymlink != 0 || !mode.IsRegular() && !mode.IsDir() {
			return fmt.Errorf("Java runtime archive contains link or special entry %q", entry.Name)
		}
		target := filepath.Join(destination, name)
		if mode.IsDir() {
			if err := budget.addEntry(name, 0, true); err != nil {
				return err
			}
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			continue
		}
		if entry.UncompressedSize64 > uint64(limits.maxBytes) {
			return fmt.Errorf("Java runtime archive exceeds %d bytes at %s", limits.maxBytes, filepath.ToSlash(name))
		}
		if err := budget.addEntry(name, int64(entry.UncompressedSize64), false); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		in, err := entry.Open()
		if err != nil {
			return err
		}
		permissions := os.FileMode(0o600)
		if mode&0o111 != 0 {
			permissions = 0o700
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, permissions)
		if err != nil {
			_ = in.Close()
			return err
		}
		copied, copyErr := io.Copy(out, io.LimitReader(in, int64(entry.UncompressedSize64)+1))
		if copyErr == nil && copied != int64(entry.UncompressedSize64) {
			copyErr = fmt.Errorf("Java runtime archive entry %q changed size while extracting", entry.Name)
		}
		if err := errors.Join(copyErr, in.Close(), out.Close()); err != nil {
			return err
		}
	}
	return nil
}

func javaHomeInInstall(install string) (string, error) {
	before, err := os.Lstat(install)
	if err != nil {
		return "", err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return "", fmt.Errorf("pinned Java archive root must be a real directory")
	}
	dir, err := os.Open(install)
	if err != nil {
		return "", err
	}
	opened, statErr := dir.Stat()
	entries, readErr := readJavaDirEntries(dir, 1)
	closeErr := dir.Close()
	after, pathErr := os.Lstat(install)
	if err := errors.Join(statErr, readErr, closeErr, pathErr); err != nil {
		return "", err
	}
	if !sameJavaFileSnapshot(before, opened) || !sameJavaFileSnapshot(before, after) {
		return "", fmt.Errorf("pinned Java archive root changed while enumerating")
	}
	if len(entries) != 1 || !entries[0].IsDir() || entries[0].Type()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("pinned Java archive must contain exactly one real root directory")
	}
	home := filepath.Join(install, entries[0].Name())
	if runtime.GOOS == "darwin" {
		home = filepath.Join(home, "Contents", "Home")
	}
	for _, rel := range []string{"bin", "conf", "lib", filepath.Join("lib", "modules")} {
		info, err := os.Lstat(filepath.Join(home, rel))
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.Join(err, fmt.Errorf("pinned Java runtime lacks real %s", rel))
		}
	}
	return home, nil
}

func readProvisionedRegularFile(path string, limit int64) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Size() < 0 || before.Size() > limit {
		return nil, fmt.Errorf("provisioned Java file %s must be a bounded regular non-symlink file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, statErr := file.Stat()
	body, readErr := io.ReadAll(io.LimitReader(file, limit+1))
	heldAfter, heldErr := file.Stat()
	closeErr := file.Close()
	liveAfter, liveErr := os.Lstat(path)
	if err := errors.Join(statErr, readErr, heldErr, closeErr, liveErr); err != nil {
		return nil, err
	}
	if int64(len(body)) != before.Size() || !sameJavaFileSnapshot(before, opened) || !sameJavaFileSnapshot(before, heldAfter) || !sameJavaFileSnapshot(before, liveAfter) {
		return nil, fmt.Errorf("provisioned Java file %s changed while reading", path)
	}
	return body, nil
}

func javaLauncher(home string) string {
	name := "java"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(home, "bin", name)
}
