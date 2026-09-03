package modelithtx

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/testgit"
)

func TestRenderScriptStagesCompleteSetBeforePublishing(t *testing.T) {
	t.Parallel()
	repo := scriptFixture(t)
	beforeA := readFixture(t, repo, "examples/a/domain.modelith.md")
	beforeB := readFixture(t, repo, "examples/b/domain.modelith.md")
	output, err := runRenderFixture(t, repo, "later-failure")
	if err == nil || !strings.Contains(output, "modelith render failed for examples/b/domain.modelith.yaml") {
		t.Fatalf("output=%q err=%v", output, err)
	}
	if got := readFixture(t, repo, "examples/a/domain.modelith.md"); got != beforeA {
		t.Fatalf("first live render changed after later failure: %q", got)
	}
	if got := readFixture(t, repo, "examples/b/domain.modelith.md"); got != beforeB {
		t.Fatalf("second live render changed after later failure: %q", got)
	}
	assertNoTransactionResidue(t, repo)
}

func TestRenderScriptRejectsWarningsAndUnexpectedOutput(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		mode string
		want string
	}{
		{mode: "version-warning", want: "run-safe: successful command emitted stderr"},
		{mode: "render-warning", want: "run-safe: successful command stderr did not exactly match its permitted receipt"},
		{mode: "render-stdout", want: "run-safe: successful command stdout did not exactly match its permitted receipt"},
		{mode: "extra-output", want: "modelith render changed staged corpus paths outside the exact render inventory"},
	} {
		t.Run(test.mode, func(t *testing.T) {
			repo := scriptFixture(t)
			before := readFixture(t, repo, "examples/a/domain.modelith.md")
			output, err := runRenderFixture(t, repo, test.mode)
			if err == nil || !strings.Contains(output, test.want) {
				t.Fatalf("output=%q err=%v", output, err)
			}
			if got := readFixture(t, repo, "examples/a/domain.modelith.md"); got != before {
				t.Fatalf("live render changed on rejected output: %q", got)
			}
			assertNoTransactionResidue(t, repo)
		})
	}
}

func TestRenderScriptPublishesExactSetAndIsByteIdempotent(t *testing.T) {
	t.Parallel()
	repo := scriptFixture(t)
	if output, err := runRenderFixture(t, repo, "exact"); err != nil {
		t.Fatalf("first render: %v\n%s", err, output)
	}
	firstA := readFixture(t, repo, "examples/a/domain.modelith.md")
	firstB := readFixture(t, repo, "examples/b/domain.modelith.md")
	if output, err := runRenderFixture(t, repo, "exact"); err != nil {
		t.Fatalf("second render: %v\n%s", err, output)
	}
	if got := readFixture(t, repo, "examples/a/domain.modelith.md"); got != firstA {
		t.Fatalf("first render is not byte-idempotent: %q != %q", got, firstA)
	}
	if got := readFixture(t, repo, "examples/b/domain.modelith.md"); got != firstB {
		t.Fatalf("second render is not byte-idempotent: %q != %q", got, firstB)
	}
	assertNoTransactionResidue(t, repo)
}

func scriptFixture(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	sourceRepo := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	repo := t.TempDir()
	copyFixtureFile(t, filepath.Join(sourceRepo, "go.mod"), filepath.Join(repo, "go.mod"))
	for _, rel := range []string{
		"scripts/git-safe.sh",
		"scripts/git-safe/main.go",
		"scripts/modelith-render.sh",
		"scripts/modelith-inventory.sh",
		"scripts/modelith-tx.go",
	} {
		copyFixtureFile(t, filepath.Join(sourceRepo, rel), filepath.Join(repo, rel))
	}
	copyFixtureTree(t, filepath.Join(sourceRepo, "scripts", "run-safe"), filepath.Join(repo, "scripts", "run-safe"), "_test.go")
	copyFixtureTree(t, filepath.Join(sourceRepo, "internal", "modelithtx"), filepath.Join(repo, "internal", "modelithtx"), "_test.go")
	copyFixtureTree(t, filepath.Join(sourceRepo, "internal", "filelock"), filepath.Join(repo, "internal", "filelock"), "_test.go")
	copyFixtureTree(t, filepath.Join(sourceRepo, "internal", "gitcontrol"), filepath.Join(repo, "internal", "gitcontrol"), "_test.go")
	copyFixtureTree(t, filepath.Join(sourceRepo, "internal", "processcontrol"), filepath.Join(repo, "internal", "processcontrol"), "_test.go")
	for _, component := range []string{"a", "b"} {
		dir := filepath.Join(repo, "examples", component)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "domain.modelith.yaml"), []byte("entities: []\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "domain.modelith.md"), []byte("old "+component+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	bin := filepath.Join(repo, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	fake := `#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  --version)
    printf 'modelith version 0.4.0\n'
    if [[ "${MODELITH_FAKE_MODE:-}" == version-warning ]]; then printf 'WARNING: hostile environment\n' >&2; fi
    ;;
  render)
    source=${2:?}
    if [[ "${MODELITH_FAKE_MODE:-}" == later-failure && "$source" == examples/b/* ]]; then exit 9; fi
    rendered=${source%.yaml}.md
    printf 'rendered %s\n\n' "$source" >"$rendered"
	printf 'wrote %s\n' "$rendered" >&2
    if [[ "${MODELITH_FAKE_MODE:-}" == render-warning ]]; then printf 'deprecated option\n' >&2; fi
    if [[ "${MODELITH_FAKE_MODE:-}" == render-stdout ]]; then printf 'unexpected success text\n'; fi
    if [[ "${MODELITH_FAKE_MODE:-}" == extra-output ]]; then printf 'unexpected\n' > examples/unexpected.txt; fi
    ;;
  *) exit 2 ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "modelith"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		if output, err := testgit.Run(t.Context(), repo, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	git("init", "-q")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.invalid")
	git("add", ".")
	git("commit", "-qm", "fixture")
	return repo
}

func runRenderFixture(t *testing.T, repo, mode string) (string, error) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "bash", "scripts/modelith-render.sh", "render", "examples", "v0.4.0")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "PATH="+filepath.Join(repo, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"), "MODELITH_FAKE_MODE="+mode)
	body, err := cmd.CombinedOutput()
	return string(body), err
}

func copyFixtureTree(t *testing.T, source, target, excludeSuffix string) {
	t.Helper()
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), excludeSuffix) {
			continue
		}
		copyFixtureFile(t, filepath.Join(source, entry.Name()), filepath.Join(target, entry.Name()))
	}
}

func copyFixtureFile(t *testing.T, source, target string) {
	t.Helper()
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, body, info.Mode().Perm()); err != nil {
		t.Fatal(err)
	}
}

func readFixture(t *testing.T, repo, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(repo, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func assertNoTransactionResidue(t *testing.T, repo string) {
	t.Helper()
	for _, name := range []string{stageName, backupName, retireName, journalName, journalNextName, journalAuthorityName, journalRetireName} {
		if _, err := os.Lstat(filepath.Join(repo, name)); !os.IsNotExist(err) {
			t.Fatalf("transaction residue %s remains: %v", name, err)
		}
	}
}
