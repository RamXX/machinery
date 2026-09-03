package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/testgit"
)

func TestRunIgnoresTwoRepositoryAndConfigInjection(t *testing.T) {
	repoA := newRepository(t)
	repoB := newRepository(t)
	environ := append(os.Environ(),
		"GIT_DIR="+filepath.Join(repoB, ".git"),
		"GIT_WORK_TREE="+repoB,
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=core.repositoryformatversion",
		"GIT_CONFIG_VALUE_0=999",
		"GIT_INDEX_FILE="+filepath.Join(repoB, ".git", "index"),
		"GIT_OBJECT_DIRECTORY="+filepath.Join(repoB, ".git", "objects"),
		"GIT_ALTERNATE_OBJECT_DIRECTORIES="+filepath.Join(repoB, ".git", "objects"),
		"GIT_REPLACE_REF_BASE=refs/replace-hostile/",
		"GIT_TRACE=1",
		"GIT_ASKPASS=/definitely/not/allowed",
	)
	var stdout, stderr bytes.Buffer
	if status := run([]string{"-root", repoA, "--", "rev-parse", "--show-toplevel"}, environ, &stdout, &stderr); status != 0 {
		t.Fatalf("sanitized Git command failed with %d: %s", status, stderr.String())
	}
	want, err := filepath.EvalSymlinks(repoA)
	if err != nil {
		t.Fatal(err)
	}
	got, err := filepath.EvalSymlinks(strings.TrimSpace(stdout.String()))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Git resolved injected repository %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("successful sanitized Git command emitted stderr: %s", stderr.String())
	}
}

func TestRunTimesOutAndKillsGitProcessTree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX process-tree fixture")
	}
	bin := t.TempDir()
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	repo := newRepository(t)
	fakeGit := filepath.Join(bin, "git")
	script := "#!/bin/sh\nsleep 30 &\nprintf '%s\\n' \"$!\" >\"$MACHINERY_TEST_PID_FILE\"\nwait\n"
	if err := os.WriteFile(fakeGit, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	environ := append(os.Environ(), "MACHINERY_TEST_PID_FILE="+pidFile)
	var stdout, stderr bytes.Buffer
	started := time.Now()
	// Leave enough startup budget for a saturated full-suite runner to schedule
	// the shell and publish its descendant PID before the deadline. The command
	// remains tightly bounded, and the assertion below still proves that the
	// descendant is gone when the timeout returns.
	status := run([]string{"-root", repo, "-timeout", "3s", "--", "status"}, environ, &stdout, &stderr)
	if status != 124 || !strings.Contains(stderr.String(), "timed out after 3s") {
		t.Fatalf("timeout status=%d stderr=%q", status, stderr.String())
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("bounded Git command took %s", elapsed)
	}
	rawPID, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("read descendant pid: %v; stderr=%q", err, stderr.String())
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(rawPID)))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		probeContext, cancel := context.WithTimeout(context.Background(), time.Second)
		probe := exec.CommandContext(probeContext, "kill", "-0", strconv.Itoa(pid))
		err = probe.Run()
		cancel()
		if err != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Git descendant %d survived timeout", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRunRejectsSuccessStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable fixture")
	}
	bin := t.TempDir()
	fakeGit := filepath.Join(bin, "git")
	if err := os.WriteFile(fakeGit, []byte("#!/bin/sh\nprintf 'warning\\n' >&2\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stdout, stderr bytes.Buffer
	status := run([]string{"-root", t.TempDir(), "--", "status"}, os.Environ(), &stdout, &stderr)
	if status == 0 || !strings.Contains(stderr.String(), "successful Git command emitted stderr") {
		t.Fatalf("success stderr was accepted: status=%d stderr=%q", status, stderr.String())
	}
}

func TestRunRejectsSuccessfulStdoutOverflow(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable fixture")
	}
	bin := t.TempDir()
	fakeGit := filepath.Join(bin, "git")
	if err := os.WriteFile(fakeGit, []byte("#!/bin/sh\nprintf '0123456789'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	originalLimit := outputLimit
	outputLimit = 4
	t.Cleanup(func() { outputLimit = originalLimit })
	var stdout, stderr bytes.Buffer
	status := run([]string{"-root", t.TempDir(), "--", "status"}, os.Environ(), &stdout, &stderr)
	if status == 0 || !strings.Contains(stderr.String(), "stdout exceeded 4-byte capture limit") {
		t.Fatalf("overflow status=%d stderr=%q", status, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[output truncated at 4 bytes]") {
		t.Fatalf("bounded stdout = %q", stdout.String())
	}
}

func newRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		if output, err := testgit.Run(t.Context(), root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	git("init", "-q")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(root, "fixture"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "fixture")
	git("commit", "-qm", "fixture")
	return root
}
