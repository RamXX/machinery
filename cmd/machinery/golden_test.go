package main

// Golden-corpus regression harness. The corpus under testdata/golden was
// captured from the frozen Python toolchain at the pre-go-migration tag and
// byte-matched the Go binary at parity; from then on it is the Go binary's own
// regression net. After an INTENDED output change, re-capture with:
//
//	go test ./cmd/machinery -run TestGolden -update
//
// and review the golden diff like any other code change.

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

var updateGolden = flag.Bool("update", false, "re-capture the golden corpus from the current binary")

var (
	buildOnce sync.Once
	builtBin  string
	buildErr  error
)

func goldenBin(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "machinery-golden-bin")
		if err != nil {
			buildErr = err
			return
		}
		builtBin = filepath.Join(dir, "machinery")
		out, err := exec.CommandContext(t.Context(), "go", "build", "-o", builtBin, ".").CombinedOutput()
		if err != nil {
			buildErr = fmt.Errorf("go build: %w\n%s", err, out)
		}
	})
	if buildErr != nil {
		t.Fatal(buildErr)
	}
	return builtBin
}

func repoRootDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func goldenDir(t *testing.T, caseName string) string {
	return filepath.Join(repoRootDir(t), "testdata", "golden", caseName)
}

// runBin executes the built binary, returning stdout, stderr, exit code.
func runBin(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), goldenBin(t), args...)
	var out, errB bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errB
	err := cmd.Run()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return out.String(), errB.String(), code
}

func compareOrUpdate(t *testing.T, path, got string) {
	t.Helper()
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden %s (run with -update to capture): %v", path, err)
	}
	if string(want) != got {
		t.Errorf("golden mismatch %s:\n--- want\n%s\n--- got\n%s", path, clip(string(want)), clip(got))
	}
}

func clip(s string) string {
	if len(s) > 4000 {
		return s[:4000] + "\n[...clipped...]"
	}
	return s
}

var goldenExamples = []struct {
	ex        string // examples/<ex>
	checkCase string // golden dir name for check (portfolio-engine is check-portfolio)
	withImpl  bool
}{
	{"go-crm", "check-go-crm", true},
	{"fulfillment", "check-fulfillment", false},
	{"portfolio-engine", "check-portfolio", false},
}

func TestGoldenLint(t *testing.T) {
	root := repoRootDir(t)
	for _, c := range goldenExamples {
		t.Run(c.ex, func(t *testing.T) {
			out, errS, code := runBin(t, "lint", filepath.Join(root, "examples", c.ex, "design", "machines"))
			g := goldenDir(t, "lint-"+c.ex)
			compareOrUpdate(t, filepath.Join(g, "stdout.txt"), out)
			compareOrUpdate(t, filepath.Join(g, "stderr.txt"), errS)
			compareOrUpdate(t, filepath.Join(g, "exitcode.txt"), fmt.Sprintf("%d\n", code))
		})
	}
}

func TestGoldenOracle(t *testing.T) {
	root := repoRootDir(t)
	for _, c := range goldenExamples {
		t.Run(c.ex, func(t *testing.T) {
			scratch := t.TempDir()
			src := filepath.Join(root, "examples", c.ex, "design", "machines")
			entries, err := os.ReadDir(src)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".machine.json") {
					data, err := os.ReadFile(filepath.Join(src, e.Name()))
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(scratch, e.Name()), data, 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			// capture-golden ran oracle with stderr folded into stdout
			cmd := exec.CommandContext(t.Context(), goldenBin(t), "oracle", scratch)
			var combined bytes.Buffer
			cmd.Stdout, cmd.Stderr = &combined, &combined
			_ = cmd.Run()
			g := goldenDir(t, "oracle-"+c.ex)
			compareOrUpdate(t, filepath.Join(g, "stdout.txt"), combined.String())
			produced, err := filepath.Glob(filepath.Join(scratch, "*.oracle.md"))
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range produced {
				data, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				compareOrUpdate(t, filepath.Join(g, filepath.Base(p)), string(data))
			}
			assertSameFileSet(t, g, produced, ".oracle.md")
		})
	}
}

func TestGoldenCheck(t *testing.T) {
	root := repoRootDir(t)
	for _, c := range goldenExamples {
		t.Run(c.ex, func(t *testing.T) {
			args := []string{"check", filepath.Join(root, "examples", c.ex, "design")}
			if c.withImpl {
				args = append(args, "--impl", filepath.Join(root, "examples", c.ex, "impl"))
			}
			out, errS, code := runBin(t, args...)
			g := goldenDir(t, c.checkCase)
			compareOrUpdate(t, filepath.Join(g, "stdout.txt"), out)
			compareOrUpdate(t, filepath.Join(g, "stderr.txt"), errS)
			compareOrUpdate(t, filepath.Join(g, "exitcode.txt"), fmt.Sprintf("%d\n", code))
		})
	}
}

func TestGoldenGen(t *testing.T) {
	root := repoRootDir(t)
	for _, c := range goldenExamples {
		t.Run(c.ex, func(t *testing.T) {
			d := filepath.Join(root, "examples", c.ex, "design")
			scratch := t.TempDir()
			machines, _ := filepath.Glob(filepath.Join(d, "machines", "*.machine.json"))
			for _, mj := range machines {
				_, _, _ = runBin(t, "tla", mj, scratch)
			}
			sems, _ := filepath.Glob(filepath.Join(d, "formal", "*.semantics.yaml"))
			for _, sem := range sems {
				m := strings.TrimSuffix(filepath.Base(sem), ".semantics.yaml")
				_, _, _ = runBin(t, "refine", filepath.Join(d, "machines", m+".machine.json"), sem, scratch)
			}
			comps, _ := filepath.Glob(filepath.Join(d, "formal", "*.composition.yaml"))
			for _, comp := range comps {
				data, err := os.ReadFile(comp)
				if err != nil {
					t.Fatal(err)
				}
				compV, err := ir.LoadYAML(data)
				if err != nil || compV.Kind != ir.KindObject {
					t.Fatalf("bad composition %s: %v", comp, err)
				}
				coord := compV.AsObject().GetString("coordinator")
				_, _, _ = runBin(t, "compose", comp, filepath.Join(d, "machines", coord+".machine.json"), scratch)
			}
			g := goldenDir(t, "gen-"+c.ex)
			var produced []string
			for _, pat := range []string{"*.tla", "*.cfg"} {
				m, _ := filepath.Glob(filepath.Join(scratch, pat))
				produced = append(produced, m...)
			}
			for _, p := range produced {
				data, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				compareOrUpdate(t, filepath.Join(g, filepath.Base(p)), string(data))
			}
			assertSameFileSet(t, g, produced, ".tla", ".cfg")
		})
	}
}

// copyDirInto copies a directory tree (regular files only) into dst.
func copyDirInto(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		sp := filepath.Join(src, e.Name())
		dp := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := os.MkdirAll(dp, 0o755); err != nil {
				t.Fatal(err)
			}
			copyDirInto(t, sp, dp)
			continue
		}
		data, err := os.ReadFile(sp)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dp, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGoldenPackGenerate(t *testing.T) {
	root := repoRootDir(t)
	scratch := t.TempDir()
	copyDirInto(t, filepath.Join(root, "examples", "checkout-split", "parent", "design"), scratch)
	// drop the committed packs so the golden captures a from-scratch generation
	if err := os.RemoveAll(filepath.Join(scratch, "packs")); err != nil {
		t.Fatal(err)
	}
	out, errS, code := runBin(t, "pack", "generate", scratch)
	g := goldenDir(t, "pack-generate-checkout-split")
	compareOrUpdate(t, filepath.Join(g, "stdout.txt"), out)
	compareOrUpdate(t, filepath.Join(g, "stderr.txt"), errS)
	compareOrUpdate(t, filepath.Join(g, "exitcode.txt"), fmt.Sprintf("%d\n", code))
	packDirs, err := filepath.Glob(filepath.Join(scratch, "packs", "*.pack"))
	if err != nil {
		t.Fatal(err)
	}
	if len(packDirs) == 0 {
		t.Fatal("pack generate produced no packs")
	}
	for _, pd := range packDirs {
		produced, err := filepath.Glob(filepath.Join(pd, "*"))
		if err != nil {
			t.Fatal(err)
		}
		gp := filepath.Join(g, filepath.Base(pd))
		for _, p := range produced {
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			compareOrUpdate(t, filepath.Join(gp, filepath.Base(p)), string(data))
		}
		assertSameFileSet(t, gp, produced, ".yaml", ".md", ".json", ".tla", ".cfg")
	}
}

func TestGoldenScale(t *testing.T) {
	root := repoRootDir(t)
	cases := []struct {
		caseName string
		design   string
	}{
		{"scale-go-crm", filepath.Join(root, "examples", "go-crm", "design")},
		{"scale-checkout-split-parent", filepath.Join(root, "examples", "checkout-split", "parent", "design")},
	}
	for _, c := range cases {
		t.Run(c.caseName, func(t *testing.T) {
			out, errS, code := runBin(t, "scale", c.design)
			g := goldenDir(t, c.caseName)
			compareOrUpdate(t, filepath.Join(g, "stdout.txt"), out)
			compareOrUpdate(t, filepath.Join(g, "stderr.txt"), errS)
			compareOrUpdate(t, filepath.Join(g, "exitcode.txt"), fmt.Sprintf("%d\n", code))
		})
	}
}

// assertSameFileSet fails when the golden dir holds files (with the given
// extensions) the run no longer produces: a generator silently dropping an
// artifact must not pass.
func assertSameFileSet(t *testing.T, goldenDirPath string, produced []string, exts ...string) {
	t.Helper()
	if *updateGolden {
		// on update, remove stale goldens for these extensions
		entries, _ := os.ReadDir(goldenDirPath)
		producedSet := map[string]bool{}
		for _, p := range produced {
			producedSet[filepath.Base(p)] = true
		}
		for _, e := range entries {
			for _, ext := range exts {
				if strings.HasSuffix(e.Name(), ext) && !producedSet[e.Name()] {
					_ = os.Remove(filepath.Join(goldenDirPath, e.Name()))
				}
			}
		}
		return
	}
	producedSet := map[string]bool{}
	for _, p := range produced {
		producedSet[filepath.Base(p)] = true
	}
	entries, err := os.ReadDir(goldenDirPath)
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for _, e := range entries {
		for _, ext := range exts {
			if strings.HasSuffix(e.Name(), ext) && !producedSet[e.Name()] {
				missing = append(missing, e.Name())
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("golden files no longer produced: %v", missing)
	}
}
