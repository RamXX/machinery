package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunRejectsSuccessfulWarningStderr(t *testing.T) {
	fake := fakeCommand(t, "docker", "printf 'warning: host drift\\n' >&2\n")
	var stdout, stderr bytes.Buffer
	status := run([]string{"-timeout", "5s", "--", fake, "pull", "--quiet"}, &stdout, &stderr)
	if status == 0 || !strings.Contains(stderr.String(), "successful command emitted stderr") {
		t.Fatalf("warning stderr accepted: status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestRunAcceptsOnlyExactConfiguredReceipt(t *testing.T) {
	fake := fakeCommand(t, "modelith", "printf 'wrote examples/demo.modelith.md\\n' >&2\n")
	receipt := filepath.Join(t.TempDir(), "receipt")
	if err := os.WriteFile(receipt, []byte("wrote examples/demo.modelith.md\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	status := run([]string{"-expect-stderr-file", receipt, "--", fake, "render", "examples/demo.modelith.yaml"}, &stdout, &stderr)
	if status != 0 || stdout.Len() != 0 || stderr.String() != "wrote examples/demo.modelith.md\n" {
		t.Fatalf("exact receipt status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	if err := os.WriteFile(receipt, []byte("different\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	status = run([]string{"-expect-stderr-file", receipt, "--", fake, "render"}, &stdout, &stderr)
	if status == 0 || !strings.Contains(stderr.String(), "did not exactly match") {
		t.Fatalf("noncanonical receipt accepted: status=%d stderr=%q", status, stderr.String())
	}
}

func TestRunBoundsStdoutAndStderrIndependently(t *testing.T) {
	fake := fakeCommand(t, "docker", "printf '0123456789'\nprintf 'abcdefghij' >&2\n")
	var stdout, stderr bytes.Buffer
	status := run([]string{"-stdout-limit", "4", "-stderr-limit", "6", "--", fake, "inspect"}, &stdout, &stderr)
	if status == 0 || !strings.Contains(stdout.String(), "truncated at 4 bytes") ||
		!strings.Contains(stderr.String(), "truncated at 6 bytes") ||
		!strings.Contains(stderr.String(), "stdout exceeded 4-byte") ||
		!strings.Contains(stderr.String(), "stderr exceeded 6-byte") {
		t.Fatalf("overflow status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestExecutableSnapshotAcceptsBoundHomebrewStyleSymlinkChain(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable fixture")
	}
	root := t.TempDir()
	target := filepath.Join(root, "Cellar", "modelith", "0.4.0", "bin", "modelith")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("#!/bin/sh\nprintf 'modelith version 0.4.0\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(bin, "modelith")
	if err := os.Symlink(filepath.Join("..", "Cellar", "modelith", "0.4.0", "bin", "modelith"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	work := t.TempDir()
	var stderr bytes.Buffer
	snapshot := filepath.Join(work, "snapshot")
	receipt := filepath.Join(work, "receipt")
	status := run([]string{"snapshot-executable", "-source", link, "-destination", snapshot, "-receipt", receipt}, &bytes.Buffer{}, &stderr)
	if status != 0 {
		t.Fatalf("Homebrew-style symlink chain rejected: status=%d stderr=%q", status, stderr.String())
	}
	var stdout bytes.Buffer
	if status := run([]string{"-executable-receipt", receipt, "--", snapshot, "--version"}, &stdout, &stderr); status != 0 || stdout.String() != "modelith version 0.4.0\n" {
		t.Fatalf("snapshotted symlink target status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	if status := run([]string{"verify-executable", "-receipt", receipt}, &bytes.Buffer{}, &stderr); status != 0 {
		t.Fatalf("unchanged symlink chain did not revalidate: status=%d stderr=%q", status, stderr.String())
	}
}

func TestExecutableSnapshotIsImmutableAndSourceContentABAIsRejected(t *testing.T) {
	source := fakeCommand(t, "modelith", "printf 'old-engine\\n'\n")
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	snapshot := filepath.Join(work, "modelith")
	receipt := filepath.Join(work, "receipt")
	var stderr bytes.Buffer
	if status := run([]string{"snapshot-executable", "-source", source, "-destination", snapshot, "-receipt", receipt}, &bytes.Buffer{}, &stderr); status != 0 {
		t.Fatalf("snapshot status=%d stderr=%q", status, stderr.String())
	}
	var stdout bytes.Buffer
	stderr.Reset()
	if status := run([]string{"-executable-receipt", receipt, "--", snapshot, "--version"}, &stdout, &stderr); status != 0 || stdout.String() != "old-engine\n" {
		t.Fatalf("version snapshot status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	if err := os.WriteFile(source, []byte("#!/bin/sh\nprintf 'new-engine\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if status := run([]string{"-executable-receipt", receipt, "--", snapshot, "render"}, &stdout, &stderr); status != 0 || stdout.String() != "old-engine\n" {
		t.Fatalf("render snapshot status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	if err := os.WriteFile(source, original, 0o755); err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	if status := run([]string{"verify-executable", "-receipt", receipt}, &bytes.Buffer{}, &stderr); status == 0 || !strings.Contains(stderr.String(), "source executable changed") {
		t.Fatalf("source content ABA accepted: status=%d stderr=%q", status, stderr.String())
	}
}

func TestExecutableSnapshotRejectsSameContentPathABA(t *testing.T) {
	source := fakeCommand(t, "modelith", "printf 'engine\\n'\n")
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	snapshot := filepath.Join(work, "modelith")
	receipt := filepath.Join(work, "receipt")
	var stderr bytes.Buffer
	if status := run([]string{"snapshot-executable", "-source", source, "-destination", snapshot, "-receipt", receipt}, &bytes.Buffer{}, &stderr); status != 0 {
		t.Fatalf("snapshot status=%d stderr=%q", status, stderr.String())
	}
	if status := run([]string{"-executable-receipt", receipt, "--", snapshot, "--version"}, &bytes.Buffer{}, &stderr); status != 0 {
		t.Fatalf("version status=%d stderr=%q", status, stderr.String())
	}
	if err := os.Rename(source, source+".retained"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, body, 0o755); err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	if status := run([]string{"-executable-receipt", receipt, "--", snapshot, "render"}, &bytes.Buffer{}, &stderr); status != 0 {
		t.Fatalf("render through immutable snapshot status=%d stderr=%q", status, stderr.String())
	}
	stderr.Reset()
	if status := run([]string{"verify-executable", "-receipt", receipt}, &bytes.Buffer{}, &stderr); status == 0 || !strings.Contains(stderr.String(), "source executable changed") {
		t.Fatalf("same-content path ABA accepted: status=%d stderr=%q", status, stderr.String())
	}
}

func TestExecutableSnapshotVersionToRenderSymlinkABAIsRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX symlink executable fixture")
	}
	source := fakeCommand(t, "modelith", "printf 'engine\\n'\n")
	work := t.TempDir()
	snapshot := filepath.Join(work, "modelith")
	receipt := filepath.Join(work, "receipt")
	var stderr bytes.Buffer
	if status := run([]string{"snapshot-executable", "-source", source, "-destination", snapshot, "-receipt", receipt}, &bytes.Buffer{}, &stderr); status != 0 {
		t.Fatalf("snapshot status=%d stderr=%q", status, stderr.String())
	}
	if status := run([]string{"-executable-receipt", receipt, "--", snapshot, "--version"}, &bytes.Buffer{}, &stderr); status != 0 {
		t.Fatalf("version status=%d stderr=%q", status, stderr.String())
	}
	retained := source + ".retained"
	if err := os.Rename(source, retained); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(retained, source); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	stderr.Reset()
	if status := run([]string{"-executable-receipt", receipt, "--", snapshot, "render"}, &bytes.Buffer{}, &stderr); status != 0 {
		t.Fatalf("render through immutable snapshot status=%d stderr=%q", status, stderr.String())
	}
	stderr.Reset()
	if status := run([]string{"verify-executable", "-receipt", receipt}, &bytes.Buffer{}, &stderr); status == 0 || !strings.Contains(stderr.String(), "symlink chain changed") {
		t.Fatalf("version-to-render symlink ABA accepted: status=%d stderr=%q", status, stderr.String())
	}
}

func TestExecutableSnapshotRejectsSymlinkChainAndTargetABA(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX symlink executable fixture")
	}
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string, string, []byte)
		want   string
	}{
		{
			name: "recreated chain link",
			mutate: func(t *testing.T, source, _ string, _ []byte) {
				t.Helper()
				text, err := os.Readlink(source)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(source); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(text, source); err != nil {
					t.Fatal(err)
				}
			},
			want: "symlink chain changed",
		},
		{
			name: "same-content terminal target",
			mutate: func(t *testing.T, _ string, target string, body []byte) {
				t.Helper()
				if err := os.Rename(target, target+".retained"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, body, 0o755); err != nil {
					t.Fatal(err)
				}
			},
			want: "source executable changed",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			target := filepath.Join(root, "Cellar", "modelith", "0.4.0", "modelith")
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				t.Fatal(err)
			}
			body := []byte("#!/bin/sh\nprintf 'engine\\n'\n")
			if err := os.WriteFile(target, body, 0o755); err != nil {
				t.Fatal(err)
			}
			source := filepath.Join(root, "modelith")
			if err := os.Symlink(filepath.Join("Cellar", "modelith", "0.4.0", "modelith"), source); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			work := t.TempDir()
			snapshot := filepath.Join(work, "modelith")
			receipt := filepath.Join(work, "receipt")
			var stderr bytes.Buffer
			if status := run([]string{"snapshot-executable", "-source", source, "-destination", snapshot, "-receipt", receipt}, &bytes.Buffer{}, &stderr); status != 0 {
				t.Fatalf("snapshot status=%d stderr=%q", status, stderr.String())
			}
			if status := run([]string{"-executable-receipt", receipt, "--", snapshot, "--version"}, &bytes.Buffer{}, &stderr); status != 0 {
				t.Fatalf("version status=%d stderr=%q", status, stderr.String())
			}
			test.mutate(t, source, target, body)
			stderr.Reset()
			if status := run([]string{"-executable-receipt", receipt, "--", snapshot, "render"}, &bytes.Buffer{}, &stderr); status != 0 {
				t.Fatalf("render through immutable snapshot status=%d stderr=%q", status, stderr.String())
			}
			stderr.Reset()
			if status := run([]string{"verify-executable", "-receipt", receipt}, &bytes.Buffer{}, &stderr); status == 0 || !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("chain/target ABA accepted: status=%d stderr=%q", status, stderr.String())
			}
		})
	}
}

func TestExecutableSnapshotRejectsCycleAndSpecialTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX symlink executable fixture")
	}
	for _, test := range []struct {
		name string
		make func(*testing.T, string) string
		want string
	}{
		{
			name: "cycle",
			make: func(t *testing.T, root string) string {
				t.Helper()
				first, second := filepath.Join(root, "first"), filepath.Join(root, "second")
				if err := os.Symlink("second", first); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				if err := os.Symlink("first", second); err != nil {
					t.Fatal(err)
				}
				return first
			},
			want: "contains a cycle",
		},
		{
			name: "directory target",
			make: func(t *testing.T, root string) string {
				t.Helper()
				target := filepath.Join(root, "directory")
				if err := os.Mkdir(target, 0o755); err != nil {
					t.Fatal(err)
				}
				source := filepath.Join(root, "modelith")
				if err := os.Symlink("directory", source); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				return source
			},
			want: "is special",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			source := test.make(t, root)
			work := t.TempDir()
			var stderr bytes.Buffer
			status := run([]string{"snapshot-executable", "-source", source, "-destination", filepath.Join(work, "snapshot"), "-receipt", filepath.Join(work, "receipt")}, &bytes.Buffer{}, &stderr)
			if status == 0 || !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("unsafe chain accepted: status=%d stderr=%q", status, stderr.String())
			}
		})
	}
}

func TestRunRejectsChangedExecutableSnapshot(t *testing.T) {
	source := fakeCommand(t, "modelith", "printf 'engine\\n'\n")
	work := t.TempDir()
	snapshot := filepath.Join(work, "modelith")
	receipt := filepath.Join(work, "receipt")
	var stderr bytes.Buffer
	if status := run([]string{"snapshot-executable", "-source", source, "-destination", snapshot, "-receipt", receipt}, &bytes.Buffer{}, &stderr); status != 0 {
		t.Fatalf("snapshot status=%d stderr=%q", status, stderr.String())
	}
	if err := os.WriteFile(snapshot, []byte("#!/bin/sh\nprintf 'foreign\\n'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	if status := run([]string{"-executable-receipt", receipt, "--", snapshot}, &bytes.Buffer{}, &stderr); status == 0 || !strings.Contains(stderr.String(), "snapshot preflight failed") {
		t.Fatalf("changed snapshot executed: status=%d stderr=%q", status, stderr.String())
	}
}

func fakeCommand(t *testing.T, name, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake external-command fixture")
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
