package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/testgit"
)

const archiveCrashEnv = "MACHINERY_RELEASE_ARCHIVE_CRASH_OUTPUT"

func TestMain(m *testing.M) {
	// Darwin exposes its default temporary tree through /var, a system
	// symlink. Tests of strict output ancestry need a canonical real path so
	// incidental platform aliasing does not weaken the production contract.
	if runtime.GOOS != "windows" {
		if canonical, err := filepath.EvalSymlinks(os.TempDir()); err == nil {
			_ = os.Setenv("TMPDIR", canonical)
		}
	}
	os.Exit(m.Run())
}

func TestWriteArchiveIsByteDeterministic(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "machinery")
	if err := os.WriteFile(input, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	entries := []entry{{name: "machinery", mode: 0o755, kind: entryRegular, data: []byte("binary")}}
	a, b := filepath.Join(dir, "a.tar.gz"), filepath.Join(dir, "b.tar.gz")
	stamp := time.Unix(1_700_000_000, 0).UTC()
	if err := writeArchive(a, stamp, entries); err != nil {
		t.Fatal(err)
	}
	if err := writeArchive(b, stamp, entries); err != nil {
		t.Fatal(err)
	}
	ab, _ := os.ReadFile(a)
	bb, _ := os.ReadFile(b)
	if !bytes.Equal(ab, bb) {
		t.Fatal("identical inputs produced different archive bytes")
	}
}

func TestSingleInputRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires developer mode on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := singleInputEntry(link, "machinery"); err == nil || !strings.Contains(err.Error(), "not a symlink") {
		t.Fatalf("symlink input error = %v", err)
	}
}

func TestSingleInputRejectsSymlinkSwapAfterInspection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires developer mode on Windows")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "machinery")
	parked := filepath.Join(dir, "parked")
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(input, []byte("trusted"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("outside"), 0o755); err != nil {
		t.Fatal(err)
	}
	afterInputInspect = func(string) error {
		if err := os.Rename(input, parked); err != nil {
			return err
		}
		return os.Symlink(outside, input)
	}
	t.Cleanup(func() {
		afterInputInspect = nil
		_ = os.Remove(input)
		_ = os.Rename(parked, input)
	})
	if _, err := singleInputEntry(input, "machinery"); err == nil {
		t.Fatal("archive followed a symlink substituted after inspection")
	}
	if got, err := os.ReadFile(outside); err != nil || string(got) != "outside" {
		t.Fatalf("outside input changed: %q, %v", got, err)
	}
}

func TestSingleInputRejectsConcurrentContentChange(t *testing.T) {
	input := filepath.Join(t.TempDir(), "machinery")
	if err := os.WriteFile(input, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(input)
	if err != nil {
		t.Fatal(err)
	}
	afterInputRead = func(path string) error {
		if err := os.WriteFile(path, []byte("change"), 0o755); err != nil {
			return err
		}
		return os.Chtimes(path, info.ModTime(), info.ModTime())
	}
	t.Cleanup(func() { afterInputRead = nil })
	if _, err := singleInputEntry(input, "machinery"); err == nil || !strings.Contains(err.Error(), "changed while being read") {
		t.Fatalf("concurrent input error = %v", err)
	}
}

func TestSingleInputRejectsPathReplacementABA(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "machinery")
	body := []byte("binary")
	if err := os.WriteFile(input, body, 0o755); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(input)
	if err != nil {
		t.Fatal(err)
	}
	afterInputRead = func(path string) error {
		replacement := filepath.Join(dir, "replacement")
		if err := os.WriteFile(replacement, body, 0o755); err != nil {
			return err
		}
		if err := os.Chtimes(replacement, info.ModTime(), info.ModTime()); err != nil {
			return err
		}
		return os.Rename(replacement, path)
	}
	t.Cleanup(func() { afterInputRead = nil })
	if _, err := singleInputEntry(input, "machinery"); err == nil || !strings.Contains(err.Error(), "changed while being read") {
		t.Fatalf("replacement input error = %v", err)
	}
}

func TestSingleInputRejectsContentABAWithRestoredMtime(t *testing.T) {
	input := filepath.Join(t.TempDir(), "machinery")
	body := []byte("binary")
	if err := os.WriteFile(input, body, 0o755); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(input)
	if err != nil {
		t.Fatal(err)
	}
	if archiveFileChangeID(info) == "" {
		t.Skip("platform does not expose a metadata change identity")
	}
	time.Sleep(time.Millisecond)
	afterInputRead = func(path string) error {
		if err := os.WriteFile(path, []byte("change"), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, body, 0o755); err != nil {
			return err
		}
		return os.Chtimes(path, info.ModTime(), info.ModTime())
	}
	t.Cleanup(func() { afterInputRead = nil })
	if _, err := singleInputEntry(input, "machinery"); err == nil || !strings.Contains(err.Error(), "changed while being read") {
		t.Fatalf("content ABA error = %v", err)
	}
}

func TestSingleInputRejectsParentReplacement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses rename-over-open-directory semantics")
	}
	base := t.TempDir()
	parent := filepath.Join(base, "input")
	parked := filepath.Join(base, "input-parked")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(parent, "machinery")
	if err := os.WriteFile(input, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	afterInputRead = func(string) error {
		if err := os.Rename(parent, parked); err != nil {
			return err
		}
		return os.Mkdir(parent, 0o700)
	}
	t.Cleanup(func() {
		afterInputRead = nil
		_ = os.Remove(parent)
		_ = os.Rename(parked, parent)
	})
	if _, err := singleInputEntry(input, "machinery"); err == nil || !strings.Contains(err.Error(), "changed while being read") {
		t.Fatalf("parent replacement error = %v", err)
	}
}

func TestSingleInputPropagatesCloseFailures(t *testing.T) {
	input := filepath.Join(t.TempDir(), "machinery")
	if err := os.WriteFile(input, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	originalClose := closeArchiveInput
	sentinel := errors.New("injected input close failure")
	closeArchiveInput = func(file *os.File) error {
		return errors.Join(file.Close(), sentinel)
	}
	t.Cleanup(func() { closeArchiveInput = originalClose })
	if _, err := singleInputEntry(input, "machinery"); !errors.Is(err, sentinel) {
		t.Fatalf("input close failure was lost: %v", err)
	}
}

func TestSingleInputPropagatesRootCloseFailures(t *testing.T) {
	input := filepath.Join(t.TempDir(), "machinery")
	if err := os.WriteFile(input, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	originalClose := closeArchiveRoot
	sentinel := errors.New("injected root close failure")
	closeArchiveRoot = func(root *os.Root) error {
		return errors.Join(root.Close(), sentinel)
	}
	t.Cleanup(func() { closeArchiveRoot = originalClose })
	if _, err := singleInputEntry(input, "machinery"); !errors.Is(err, sentinel) {
		t.Fatalf("root close failure was lost: %v", err)
	}
}

func TestResolvedCommitProducesByteIdenticalArchivesAcrossDirtyRuns(t *testing.T) {
	repo := newGitRepo(t)
	writeRepoFile(t, repo, "file.txt", "committed\n", 0o644)
	git(t, repo, "add", "file.txt")
	git(t, repo, "commit", "-qm", "commit")
	stamp := time.Unix(1_700_000_000, 0).UTC()
	first := filepath.Join(t.TempDir(), "first.tar.gz")
	if err := writeArchive(first, stamp, mustCommittedEntries(t, repo)); err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, repo, "file.txt", "dirty bytes with a different mtime\n", 0o755)
	second := filepath.Join(t.TempDir(), "second.tar.gz")
	if err := writeArchive(second, stamp, mustCommittedEntries(t, repo)); err != nil {
		t.Fatal(err)
	}
	firstBytes, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatal("same resolved commit produced different bytes across dirty runs")
	}
}

func TestCommittedEntriesIgnoreWorktreeIndexAndSymlinkSubstitution(t *testing.T) {
	repo := newGitRepo(t)
	writeRepoFile(t, repo, "plain.txt", "committed\n", 0o644)
	writeRepoFile(t, repo, "tool.sh", "#!/bin/sh\nexit 0\n", 0o755)
	git(t, repo, "add", "plain.txt", "tool.sh")
	git(t, repo, "commit", "-qm", "release tree")
	originalBlob := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD:plain.txt"))
	replacementBlob := strings.TrimSpace(gitInput(t, repo, "replace-ref-bytes\n", "hash-object", "-w", "--stdin"))
	git(t, repo, "replace", originalBlob, replacementBlob)

	// Neither staged index bytes nor the later worktree symlink may influence
	// the resolved commit tree.
	writeRepoFile(t, repo, "plain.txt", "index-only\n", 0o644)
	git(t, repo, "add", "plain.txt")
	outside := filepath.Join(t.TempDir(), "outside-secret")
	if err := os.WriteFile(outside, []byte("malicious\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(repo, "plain.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repo, "plain.txt")); err != nil {
		t.Fatal(err)
	}

	entries, err := committedEntries(repo)
	if err != nil {
		t.Fatal(err)
	}
	plain := findEntry(t, entries, "machinery/plain.txt")
	if string(plain.data) != "committed\n" || plain.kind != entryRegular || plain.mode != 0o644 {
		t.Fatalf("plain committed entry = %#v", plain)
	}
	tool := findEntry(t, entries, "machinery/tool.sh")
	if tool.mode != 0o755 || string(tool.data) != "#!/bin/sh\nexit 0\n" {
		t.Fatalf("tool committed entry = %#v", tool)
	}
}

func TestCommittedEntriesBindResolvedCommitAcrossRefMovement(t *testing.T) {
	repo := newGitRepo(t)
	writeRepoFile(t, repo, "version.txt", "one\n", 0o644)
	git(t, repo, "add", "version.txt")
	git(t, repo, "commit", "-qm", "one")
	afterCommitResolve = func(string) {
		writeRepoFile(t, repo, "version.txt", "two\n", 0o644)
		git(t, repo, "add", "version.txt")
		git(t, repo, "commit", "-qm", "two")
	}
	t.Cleanup(func() { afterCommitResolve = nil })
	entries, err := committedEntries(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(findEntry(t, entries, "machinery/version.txt").data); got != "one\n" {
		t.Fatalf("archive followed moved HEAD: %q", got)
	}
}

func TestCommittedSymlinkUsesBlobTargetNotWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires developer mode on Windows")
	}
	repo := newGitRepo(t)
	writeRepoFile(t, repo, "target.txt", "target\n", 0o644)
	if err := os.Symlink("target.txt", filepath.Join(repo, "link")); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "target.txt", "link")
	git(t, repo, "commit", "-qm", "symlink")
	if err := os.Remove(filepath.Join(repo, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("elsewhere.txt", filepath.Join(repo, "link")); err != nil {
		t.Fatal(err)
	}
	item := findEntry(t, mustCommittedEntries(t, repo), "machinery/link")
	if item.kind != entrySymlink || string(item.data) != "target.txt" || item.mode != 0o777 {
		t.Fatalf("committed symlink = %#v", item)
	}
}

func TestCommittedEntriesRejectPortableAliasesAndUnsupportedTopology(t *testing.T) {
	seen, prefixes := map[string]string{}, map[string]string{}
	for _, path := range []string{"Dir/a.txt", "dir/b.txt"} {
		err := validateTreePath(path, seen, prefixes)
		if path == "Dir/a.txt" && err != nil {
			t.Fatal(err)
		}
		if path == "dir/b.txt" && (err == nil || !strings.Contains(err.Error(), "aliases portable prefix")) {
			t.Fatalf("prefix alias error = %v", err)
		}
	}
	for _, path := range []string{"CON", "aux.txt", "name.", "name ", "café", "cafe\u0301", `dir\file`} {
		if err := validateTreePath(path, map[string]string{}, map[string]string{}); err == nil {
			t.Errorf("accepted ambiguous path %q", path)
		}
	}

	repo := newGitRepo(t)
	writeRepoFile(t, repo, "base", "base", 0o644)
	git(t, repo, "add", "base")
	git(t, repo, "commit", "-qm", "base")
	commit := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	git(t, repo, "update-index", "--add", "--cacheinfo", "160000,"+commit+",vendor")
	tree := strings.TrimSpace(git(t, repo, "write-tree"))
	submoduleCommit := strings.TrimSpace(gitInput(t, repo, "submodule\n", "commit-tree", tree, "-p", commit))
	git(t, repo, "update-ref", "HEAD", submoduleCommit)
	if _, err := committedEntries(repo); err == nil || !strings.Contains(err.Error(), "submodule") {
		t.Fatalf("submodule error = %v", err)
	}
}

func TestRunGitHasDeterministicTimeoutAndOutputBounds(t *testing.T) {
	originalTimeout := gitCommandTimeout
	t.Cleanup(func() {
		gitCommandTimeout = originalTimeout
	})
	repo := newGitRepo(t)
	writeRepoFile(t, repo, "file", "data", 0o644)
	git(t, repo, "add", "file")
	git(t, repo, "commit", "-qm", "commit")
	got, err := runGit(repo, 8, "rev-parse", "HEAD")
	if err == nil || err.Error() != "git rev-parse HEAD exceeded 8-byte output bound" {
		t.Fatalf("output-bound result = %q, error = %v", got, err)
	}
	gitCommandTimeout = time.Nanosecond
	if _, err := runGit(repo, 8, "status"); err == nil || !strings.Contains(err.Error(), "timed out after 1ns") {
		t.Fatalf("timeout error = %v", err)
	}
}

func TestRunGitKillsBackgroundDescendantsOnTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake git executable uses a POSIX shell")
	}
	bin := t.TempDir()
	sentinel := filepath.Join(t.TempDir(), "survived")
	script := `#!/bin/sh
(
  /bin/sleep 1
  printf survived > "$MACHINERY_RELEASE_GIT_SENTINEL"
) &
/bin/sleep 10
`
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("MACHINERY_RELEASE_GIT_SENTINEL", sentinel)
	old := gitCommandTimeout
	gitCommandTimeout = 50 * time.Millisecond
	t.Cleanup(func() { gitCommandTimeout = old })
	started := time.Now()
	if _, err := runGit(t.TempDir(), 1024, "status"); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("timed-out git process tree was not reported: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("background child retained git output pipes for %s", elapsed)
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err := os.Lstat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("background git descendant survived timeout cleanup: %v", err)
	}
}

func TestRunGitIgnoresAmbientRepositoryRedirectionAndRejectsSuccessStderr(t *testing.T) {
	repoA := newGitRepo(t)
	writeRepoFile(t, repoA, "a", "a", 0o644)
	git(t, repoA, "add", "a")
	git(t, repoA, "commit", "-qm", "a")
	headA := strings.TrimSpace(git(t, repoA, "rev-parse", "HEAD"))
	repoB := newGitRepo(t)
	writeRepoFile(t, repoB, "b", "b", 0o644)
	git(t, repoB, "add", "b")
	git(t, repoB, "commit", "-qm", "b")
	t.Setenv("GIT_DIR", filepath.Join(repoB, ".git"))
	t.Setenv("GIT_WORK_TREE", repoB)
	t.Setenv("GIT_TRACE", "1")
	got, err := runGit(repoA, 1024, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(string(got)) != headA {
		t.Fatalf("ambient Git redirection changed release repository: got=%q want=%q err=%v", got, headA, err)
	}

	if runtime.GOOS == "windows" {
		return
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte("#!/bin/sh\nprintf ok\necho 'warning: injected config' >&2\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if _, err := runGit(t.TempDir(), 1024, "status"); err == nil || !strings.Contains(err.Error(), "emitted stderr on success") || !strings.Contains(err.Error(), "injected config") {
		t.Fatalf("successful Git warning was discarded: %v", err)
	}
}

func TestWriteArchivePublishesAfterSyncAndSurvivesPreRenameCrash(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "release.tar.gz")
	if err := os.WriteFile(output, []byte("previous-complete-archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestArchiveCrashHelper$")
	command.Env = append(os.Environ(), archiveCrashEnv+"="+output)
	err := command.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 86 {
		t.Fatalf("crash helper = %v", err)
	}
	if got, err := os.ReadFile(output); err != nil || string(got) != "previous-complete-archive" {
		t.Fatalf("pre-rename crash changed published archive: %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "crash-reached")); err != nil {
		t.Fatalf("crash hook did not run after file sync: %v", err)
	}
	stageName, err := archiveStageName(output)
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(filepath.Join(dir, stageName)); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("pre-rename crash did not retain one recoverable regular stage: info=%v err=%v", info, err)
	}

	originalSync := syncOutputDir
	var synced bool
	syncOutputDir = func(root *os.Root) error {
		synced = true
		return originalSync(root)
	}
	t.Cleanup(func() { syncOutputDir = originalSync })
	entries := []entry{{name: "machinery/file", mode: 0o644, kind: entryRegular, data: []byte("new")}}
	if err := writeArchive(output, time.Unix(1_700_000_000, 0).UTC(), entries); err != nil {
		t.Fatal(err)
	}
	if !synced {
		t.Fatal("successful rename did not fsync the output directory")
	}
	for _, item := range mustReadDir(t, dir) {
		if strings.HasPrefix(item.Name(), archiveStagePrefix) {
			t.Fatalf("successful retry stranded reserved release archive residue %q", item.Name())
		}
	}
}

func TestWriteArchiveRejectsIntermediateOutputDirectorySymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires developer mode on Windows")
	}
	base := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(base, "redirect")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(link, "nested", "release.tar.gz")
	entries := []entry{{name: "machinery/file", mode: 0o644, kind: entryRegular, data: []byte("new")}}
	err := writeArchive(output, time.Unix(1_700_000_000, 0).UTC(), entries)
	if err == nil || !strings.Contains(err.Error(), "must be a real directory") {
		t.Fatalf("intermediate output symlink was accepted: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(outside, "nested")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("output creation escaped through intermediate symlink: %v", err)
	}
}

func TestWriteArchiveStageCleanupPreservesLateMutations(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native change-time ABA witness is platform-specific")
	}
	for _, tc := range []struct {
		name   string
		point  string
		mutate func(*testing.T, string, string)
	}{
		{
			name:  "content after isolation",
			point: "after-stage-isolate",
			mutate: func(t *testing.T, path, _ string) {
				if err := os.WriteFile(path, []byte("late-content"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:  "path replacement before removal",
			point: "before-stage-remove",
			mutate: func(t *testing.T, path, parked string) {
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(path, parked); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, body, 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:  "same-content ABA before removal",
			point: "before-stage-remove",
			mutate: func(t *testing.T, path, _ string) {
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				info, err := os.Lstat(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("different-content"), info.Mode().Perm()); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, body, info.Mode().Perm()); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			output := filepath.Join(dir, "release.tar.gz")
			stageName, err := archiveStageName(output)
			if err != nil {
				t.Fatal(err)
			}
			stagePath := filepath.Join(dir, stageName)
			original := []byte("interrupted-stage")
			if err := os.WriteFile(stagePath, original, 0o600); err != nil {
				t.Fatal(err)
			}
			parked := filepath.Join(dir, "parked-original-stage")
			prior := archiveCleanupPoint
			archiveCleanupPoint = func(point, name string) error {
				if point == tc.point {
					tc.mutate(t, filepath.Join(dir, name), parked)
				}
				return nil
			}
			entries := []entry{{name: "machinery/file", mode: 0o644, kind: entryRegular, data: []byte("new")}}
			err = writeArchive(output, time.Unix(1_700_000_000, 0).UTC(), entries)
			archiveCleanupPoint = prior
			t.Cleanup(func() { archiveCleanupPoint = prior })
			if err == nil || !strings.Contains(err.Error(), "preserv") && !strings.Contains(err.Error(), "changed") {
				t.Fatalf("late stage mutation was accepted: %v", err)
			}
			if info, statErr := os.Lstat(stagePath); statErr != nil || !info.Mode().IsRegular() {
				t.Fatalf("late stage mutation was not preserved at its stage path: info=%v err=%v", info, statErr)
			}
			if _, statErr := os.Lstat(output); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("publication continued after unsafe recovery: %v", statErr)
			}
			if tc.name == "path replacement before removal" {
				if body, readErr := os.ReadFile(parked); readErr != nil || !bytes.Equal(body, original) {
					t.Fatalf("isolated original was not preserved: body=%q err=%v", body, readErr)
				}
			}
		})
	}
}

func TestWriteArchiveDeferredCleanupPreservesLateStageReplacement(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "release.tar.gz")
	stageName, err := archiveStageName(output)
	if err != nil {
		t.Fatal(err)
	}
	stagePath := filepath.Join(dir, stageName)
	parked := filepath.Join(dir, "parked-owned-stage")
	priorSync, priorCleanup := afterArchiveSync, archiveCleanupPoint
	afterArchiveSync = func(string) error { return errors.New("injected post-sync failure") }
	archiveCleanupPoint = func(point, _ string) error {
		if point != "before-stage-isolate" {
			return nil
		}
		body, err := os.ReadFile(stagePath)
		if err != nil {
			return err
		}
		if err := os.Rename(stagePath, parked); err != nil {
			return err
		}
		return os.WriteFile(stagePath, body, 0o600)
	}
	entries := []entry{{name: "machinery/file", mode: 0o644, kind: entryRegular, data: []byte("new")}}
	err = writeArchive(output, time.Unix(1_700_000_000, 0).UTC(), entries)
	afterArchiveSync, archiveCleanupPoint = priorSync, priorCleanup
	t.Cleanup(func() {
		afterArchiveSync, archiveCleanupPoint = priorSync, priorCleanup
	})
	if err == nil || !strings.Contains(err.Error(), "changed before cleanup") {
		t.Fatalf("deferred cleanup accepted a late same-content path replacement: %v", err)
	}
	for _, path := range []string{stagePath, parked} {
		if info, statErr := os.Lstat(path); statErr != nil || !info.Mode().IsRegular() {
			t.Fatalf("deferred cleanup deleted %s: info=%v err=%v", path, info, statErr)
		}
	}
}

func TestWriteArchiveRetainsOutputParentAcrossPublicationReplacement(t *testing.T) {
	for _, point := range []string{"before-output-rename", "after-output-rename", "before-output-sync"} {
		t.Run(point, func(t *testing.T) {
			base := t.TempDir()
			directory := filepath.Join(base, "release")
			held := filepath.Join(base, "held-release")
			if err := os.Mkdir(directory, 0o755); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(directory, "machinery.tar.gz")
			prior := archivePublishPoint
			archivePublishPoint = func(got string) error {
				if got != point {
					return nil
				}
				if err := os.Rename(directory, held); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(directory, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(directory, "outside-sentinel"), []byte("outside"), 0o600); err != nil {
					t.Fatal(err)
				}
				return nil
			}
			entries := []entry{{name: "machinery/file", mode: 0o644, kind: entryRegular, data: []byte("new")}}
			err := writeArchive(output, time.Unix(1_700_000_000, 0).UTC(), entries)
			archivePublishPoint = prior
			t.Cleanup(func() { archivePublishPoint = prior })
			if err == nil || !strings.Contains(err.Error(), "release output directory changed identity") {
				t.Fatalf("parent replacement at %s returned success or an unrelated error: %v", point, err)
			}
			if got, readErr := os.ReadFile(filepath.Join(directory, "outside-sentinel")); readErr != nil || string(got) != "outside" {
				t.Fatalf("rooted publication touched replacement directory: got %q, err %v", got, readErr)
			}
			if _, statErr := os.Lstat(output); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("logical output appeared in replacement directory: %v", statErr)
			}
			retainedOutput := filepath.Join(held, filepath.Base(output))
			_, retainedErr := os.Lstat(retainedOutput)
			if point == "before-output-rename" && !errors.Is(retainedErr, os.ErrNotExist) {
				t.Fatalf("pre-rename replacement still published output: %v", retainedErr)
			}
			if point != "before-output-rename" && retainedErr != nil {
				t.Fatalf("rooted rename did not stay on retained parent at %s: %v", point, retainedErr)
			}
		})
	}
}

func TestWriteArchiveRetainsAndRevalidatesEveryOutputAncestor(t *testing.T) {
	base := t.TempDir()
	ancestor := filepath.Join(base, "ancestor")
	directory := filepath.Join(ancestor, "nested", "release")
	held := filepath.Join(base, "held-ancestor")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(directory, "machinery.tar.gz")
	prior := archivePublishPoint
	archivePublishPoint = func(point string) error {
		if point != "before-output-rename" {
			return nil
		}
		if err := os.Rename(ancestor, held); err != nil {
			return err
		}
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(directory, "outside-sentinel"), []byte("outside"), 0o600)
	}
	entries := []entry{{name: "machinery/file", mode: 0o644, kind: entryRegular, data: []byte("new")}}
	err := writeArchive(output, time.Unix(1_700_000_000, 0).UTC(), entries)
	archivePublishPoint = prior
	t.Cleanup(func() { archivePublishPoint = prior })
	if err == nil || !strings.Contains(err.Error(), "release output directory changed identity") {
		t.Fatalf("intermediate ancestor replacement was accepted: %v", err)
	}
	if body, readErr := os.ReadFile(filepath.Join(directory, "outside-sentinel")); readErr != nil || string(body) != "outside" {
		t.Fatalf("rooted archive operation touched replacement ancestor: body=%q err=%v", body, readErr)
	}
	if _, statErr := os.Lstat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("archive appeared beneath replacement ancestor: %v", statErr)
	}
}

func TestWriteArchiveRejectsPublishedOutputABA(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "machinery.tar.gz")
	prior := archivePublishPoint
	archivePublishPoint = func(point string) error {
		if point != "before-output-final-validation" {
			return nil
		}
		body, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		replacement := filepath.Join(directory, "replacement.tmp")
		if err := os.WriteFile(replacement, body, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(replacement, output); err != nil {
			t.Fatal(err)
		}
		return nil
	}
	entries := []entry{{name: "machinery/file", mode: 0o644, kind: entryRegular, data: []byte("new")}}
	err := writeArchive(output, time.Unix(1_700_000_000, 0).UTC(), entries)
	archivePublishPoint = prior
	t.Cleanup(func() { archivePublishPoint = prior })
	if err == nil || !strings.Contains(err.Error(), "changed before success") {
		t.Fatalf("published output same-byte ABA was accepted: %v", err)
	}
	if info, statErr := os.Lstat(output); statErr != nil || !info.Mode().IsRegular() {
		t.Fatalf("replacement output was not preserved: info=%v err=%v", info, statErr)
	}
}

func TestWriteArchiveRejectsForeignOrUnsafeReservedResidue(t *testing.T) {
	entries := []entry{{name: "machinery/file", mode: 0o644, kind: entryRegular, data: []byte("new")}}
	for _, tc := range []struct {
		name  string
		entry string
		setup func(*testing.T, string)
	}{
		{"lookalike", archiveStagePrefix + "not-a-valid-stage", func(t *testing.T, path string) {
			writeRepoFile(t, filepath.Dir(path), filepath.Base(path), "foreign", 0o600)
		}},
		{"directory", archiveStagePrefix + strings.Repeat("0", sha256.Size*2) + archiveStageSuffix, func(t *testing.T, path string) {
			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{"symlink", archiveStagePrefix + strings.Repeat("1", sha256.Size*2) + archiveStageSuffix, func(t *testing.T, path string) {
			if err := os.Symlink(filepath.Join(t.TempDir(), "outside"), path); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			residue := filepath.Join(dir, tc.entry)
			tc.setup(t, residue)
			output := filepath.Join(dir, "release.tar.gz")
			err := writeArchive(output, time.Unix(1_700_000_000, 0).UTC(), entries)
			if err == nil || !strings.Contains(err.Error(), "release archive residue") && !strings.Contains(err.Error(), "reserved release archive residue") {
				t.Fatalf("unsafe residue did not block publication: %v", err)
			}
			if _, err := os.Lstat(residue); err != nil {
				t.Fatalf("foreign residue was deleted: %v", err)
			}
			if _, err := os.Lstat(output); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("publication started before residue validation: %v", err)
			}
		})
	}
}

func TestWriteArchiveValidatesCompleteResidueInventoryBeforeCleanup(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, archiveStagePrefix+strings.Repeat("0", sha256.Size*2)+archiveStageSuffix)
	unsafe := filepath.Join(dir, archiveStagePrefix+"z-unsafe-residue")
	if err := os.WriteFile(valid, []byte("owned-looking-stage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unsafe, []byte("foreign"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "release.tar.gz")
	entries := []entry{{name: "machinery/file", mode: 0o644, kind: entryRegular, data: []byte("new")}}
	err := writeArchive(output, time.Unix(1_700_000_000, 0).UTC(), entries)
	if err == nil || !strings.Contains(err.Error(), "unexpected reserved release archive residue") {
		t.Fatalf("unsafe later residue did not fail the complete inventory: %v", err)
	}
	for _, path := range []string{valid, unsafe} {
		if info, statErr := os.Lstat(path); statErr != nil || !info.Mode().IsRegular() {
			t.Fatalf("preflight mutated residue %s: info=%v err=%v", path, info, statErr)
		}
	}
}

func TestWriteArchiveRecoversLegacyRegularCrashResidue(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, archiveStagePrefix+"123456789")
	if err := os.WriteFile(legacy, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "release.tar.gz")
	entries := []entry{{name: "machinery/file", mode: 0o644, kind: entryRegular, data: []byte("new")}}
	if err := writeArchive(output, time.Unix(1_700_000_000, 0).UTC(), entries); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(legacy); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy crash residue survived retry: %v", err)
	}
}

func mustReadDir(t *testing.T, path string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestArchiveCrashHelper(t *testing.T) {
	output := os.Getenv(archiveCrashEnv)
	if output == "" {
		return
	}
	afterArchiveSync = func(string) error {
		_ = os.WriteFile(filepath.Join(filepath.Dir(output), "crash-reached"), []byte("yes"), 0o600)
		os.Exit(86)
		return nil
	}
	entries := []entry{{name: "machinery/file", mode: 0o644, kind: entryRegular, data: []byte("replacement")}}
	_ = writeArchive(output, time.Unix(1_700_000_000, 0).UTC(), entries)
	os.Exit(87)
}

func TestArchiveHeaderAndOrderAreCanonical(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "archive.tar.gz")
	stamp := time.Unix(1_700_000_000, 0).UTC()
	entries := []entry{
		{name: "machinery/z", mode: 0o755, kind: entryRegular, data: []byte("z")},
		{name: "machinery/a", mode: 0o644, kind: entryRegular, data: []byte("a")},
		{name: "machinery/link", mode: 0o777, kind: entrySymlink, data: []byte("a")},
	}
	if err := writeArchive(output, stamp, entries); err != nil {
		t.Fatal(err)
	}
	headers := readArchive(t, output)
	if got := []string{headers[0].Name, headers[1].Name, headers[2].Name}; strings.Join(got, ",") != "machinery/a,machinery/link,machinery/z" {
		t.Fatalf("archive order = %v", got)
	}
	for _, header := range headers {
		if header.Uid != 0 || header.Gid != 0 || !header.ModTime.Equal(stamp) || header.Uname != "" || header.Gname != "" {
			t.Fatalf("noncanonical header: %#v", header)
		}
	}
}

func newGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	git(t, repo, "init", "-q")
	git(t, repo, "config", "user.name", "Release Test")
	git(t, repo, "config", "user.email", "release@example.invalid")
	return repo
}

func writeRepoFile(t *testing.T, repo, rel, content string, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, repo string, args ...string) string {
	t.Helper()
	return gitInput(t, repo, "", args...)
}

func gitInput(t *testing.T, repo, input string, args ...string) string {
	t.Helper()
	output, err := testgit.RunInput(t.Context(), repo, strings.NewReader(input), args...)
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func mustCommittedEntries(t *testing.T, repo string) []entry {
	t.Helper()
	entries, err := committedEntries(repo)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func findEntry(t *testing.T, entries []entry, name string) entry {
	t.Helper()
	for _, item := range entries {
		if item.name == name {
			return item
		}
	}
	t.Fatalf("entry %q missing from %#v", name, entries)
	return entry{}
}

func readArchive(t *testing.T, path string) []*tar.Header {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	var headers []*tar.Header
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return headers
		}
		if err != nil {
			t.Fatal(err)
		}
		copy := *header
		headers = append(headers, &copy)
	}
}
