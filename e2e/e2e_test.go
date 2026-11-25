// Package e2e is the whole test suite of machinery: testscript files that run
// the real built binary against real files, real git and the real external tools.
package e2e

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

var (
	binDir   string
	repoRoot string
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	repoRoot = filepath.Dir(wd)
	tmp, err := os.MkdirTemp("", "machinery-e2e-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	binDir = tmp
	build := exec.Command("go", "build", "-o", filepath.Join(tmp, "machinery"), "./cmd/machinery")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building machinery: %v\n%s", err, out)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

func toolsDir() string {
	if d := os.Getenv("MACHINERY_TOOLS"); d != "" {
		return d
	}
	return filepath.Join(repoRoot, ".bin")
}

func TestScripts(t *testing.T) {
	tools := toolsDir()
	testscript.Run(t, testscript.Params{
		Dir:                 "testdata/script",
		RequireExplicitExec: true,
		Setup: func(env *testscript.Env) error {
			home := filepath.Join(env.WorkDir, ".home")
			if err := os.MkdirAll(home, 0o755); err != nil {
				return err
			}
			gitcfg := filepath.Join(home, "gitconfig")
			if err := os.WriteFile(gitcfg, nil, 0o644); err != nil {
				return err
			}
			path := strings.Join([]string{binDir, filepath.Join(repoRoot, "e2e", "testdata", "bin"), tools, os.Getenv("PATH")}, string(os.PathListSeparator))
			env.Setenv("PATH", path)
			env.Setenv("HOME", home)
			env.Setenv("REPO", repoRoot)
			env.Setenv("MACHINERY_TOOLS", tools)
			env.Setenv("GIT_CONFIG_GLOBAL", gitcfg)
			env.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			env.Setenv("GIT_AUTHOR_NAME", "Test Author")
			env.Setenv("GIT_AUTHOR_EMAIL", "author@example.test")
			env.Setenv("GIT_COMMITTER_NAME", "Test Author")
			env.Setenv("GIT_COMMITTER_EMAIL", "author@example.test")
			env.Setenv("BEADS_ACTOR", "builder")
			env.Setenv("BD_NON_INTERACTIVE", "1")
			for _, k := range []string{"GOCACHE", "GOMODCACHE", "GOPATH", "GOFLAGS", "GOTOOLCHAIN", "DOCKER_HOST"} {
				if v := goEnv(k); v != "" {
					env.Setenv(k, v)
				}
			}
			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"fixture":   cmdFixture,
			"gitinit":   cmdGitInit,
			"gitcommit": cmdGitCommit,
		},
	})
}

func goEnv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	switch key {
	case "GOCACHE", "GOMODCACHE", "GOPATH":
		out, err := exec.Command("go", "env", key).Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return ""
}

// fixture <name> [<dest>] copies e2e/testdata/fixtures/<name>/ into the work
// directory (or into <dest> below it).
func cmdFixture(ts *testscript.TestScript, neg bool, args []string) {
	if neg || len(args) < 1 || len(args) > 2 {
		ts.Fatalf("usage: fixture <name> [<dest>]")
	}
	src := filepath.Join(repoRoot, "e2e", "testdata", "fixtures", args[0])
	dst := ts.MkAbs(".")
	if len(args) == 2 {
		dst = ts.MkAbs(args[1])
	}
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Name() == ".keep" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	ts.Check(err)
}

func git(ts *testscript.TestScript, args ...string) {
	ts.Check(ts.Exec("git", args...))
}

// gitinit makes the work directory a git repository and commits everything.
func cmdGitInit(ts *testscript.TestScript, neg bool, args []string) {
	if neg || len(args) != 0 {
		ts.Fatalf("usage: gitinit")
	}
	ignore := ts.MkAbs(".gitignore")
	if _, err := os.Stat(ignore); err != nil {
		ts.Check(os.WriteFile(ignore, []byte(".home/\n.machinery/\n.beads/\n"), 0o644))
	}
	git(ts, "init", "-q", "-b", "main")
	git(ts, "add", "-A")
	git(ts, "commit", "-q", "-m", "fixture")
}

// gitcommit <message> commits everything.
func cmdGitCommit(ts *testscript.TestScript, neg bool, args []string) {
	if neg || len(args) != 1 {
		ts.Fatalf("usage: gitcommit <message>")
	}
	git(ts, "add", "-A")
	git(ts, "commit", "-q", "-m", args[0])
}
