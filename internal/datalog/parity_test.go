package datalog

// The parity harness. Every directory under testdata/programs holds
//
//	program.dl       the program
//	facts/*.facts    its input relations (Soufflé's default input format)
//	expected/*.csv   the output relations, or expected/error.txt for a
//	                 program both engines must reject
//
// For each program the test (a) runs this package and compares the result
// byte for byte with expected/, and (b) when souffle is on PATH, runs
// `souffle -F facts -D <tmp> program.dl` and compares every output relation
// with ours. Soufflé does not promise any row order in its output files, so
// (b) sorts the lines of both outputs bytewise before comparing; (a) needs no
// sort because this package writes rows in its own documented order.
//
// How expected/ was produced: `go test ./internal/datalog -run TestParity
// -update` with souffle on PATH. The flag refuses to run without souffle,
// and writes a program's expected/ files from this package's output only
// after the Soufflé comparison for that program passed, so every committed
// expected file is an output both engines agree on. For a rejected program,
// error.txt holds this package's error and Soufflé is only required to fail.

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/programs/*/expected from this package, after Soufflé agrees")

func programDirs(t testing.TB) []string {
	t.Helper()
	dirs, err := filepath.Glob(filepath.Join("testdata", "programs", "*"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(dirs)
	if len(dirs) < 12 {
		t.Fatalf("expected at least 12 parity programs, found %d", len(dirs))
	}
	return dirs
}

func sortedLines(s string) []string {
	lines := strings.SplitAfter(s, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	sort.Strings(lines)
	return lines
}

func TestParity(t *testing.T) {
	souffle, _ := exec.LookPath("souffle")
	if *update && souffle == "" {
		t.Fatal("-update needs souffle on PATH: expected outputs are only written after Soufflé agrees")
	}
	ran := 0
	for _, dir := range programDirs(t) {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			srcBytes, err := os.ReadFile(filepath.Join(dir, "program.dl"))
			if err != nil {
				t.Fatal(err)
			}
			factsDir := filepath.Join(dir, "facts")
			expDir := filepath.Join(dir, "expected")
			prog, perr := Parse(string(srcBytes), "program.dl")

			var res *Result
			if perr == nil {
				in, err := ReadFacts(factsDir)
				if err != nil {
					t.Fatal(err)
				}
				res, err = prog.Run(in, Options{Explain: true})
				if err != nil {
					t.Fatalf("run: %v", err)
				}
			}

			if souffle != "" {
				ours := map[string]string{}
				if res != nil {
					for _, name := range res.OutputRelations() {
						ours[name] = res.FormatCSV(name)
					}
				}
				for _, p := range souffleProblems(t.Context(), souffle, dir, t.TempDir(), perr, ours) {
					t.Error(p)
				}
				ran++
			} else {
				t.Log("souffle not on PATH: the Soufflé half of parity is skipped; install souffle to run it")
			}

			if *update {
				writeExpected(t, expDir, perr, res)
				return
			}
			if perr != nil {
				want, err := os.ReadFile(filepath.Join(expDir, "error.txt"))
				if err != nil {
					t.Fatalf("program rejected (%v) but expected/error.txt is missing: %v", perr, err)
				}
				if got := perr.Error(); got != strings.TrimSuffix(string(want), "\n") {
					t.Fatalf("error mismatch\n got: %s\nwant: %s", got, want)
				}
				return
			}
			entries, err := os.ReadDir(expDir)
			if err != nil {
				t.Fatal(err)
			}
			var files []string
			for _, e := range entries {
				files = append(files, e.Name())
			}
			var want []string
			for _, name := range res.OutputRelations() {
				want = append(want, name+".csv")
			}
			sort.Strings(files)
			sort.Strings(want)
			if strings.Join(files, ",") != strings.Join(want, ",") {
				t.Fatalf("expected/ holds %v, program outputs %v", files, want)
			}
			for _, name := range res.OutputRelations() {
				exp, err := os.ReadFile(filepath.Join(expDir, name+".csv"))
				if err != nil {
					t.Fatal(err)
				}
				if got := res.FormatCSV(name); got != string(exp) {
					t.Errorf("%s: output differs from expected\n got:\n%s\nwant:\n%s", name, got, exp)
				}
			}
		})
	}
	if souffle != "" {
		t.Logf("Soufflé parity ran for %d programs (%s)", ran, souffle)
	}
}

// souffleProblems runs souffle on the program in dir and lists every way its
// outputs differ from ours (both sides sorted). perr is our parse error, if
// any, in which case Soufflé is only required to reject the program too.
func souffleProblems(ctx context.Context, souffle, dir, tmp string, perr error, ours map[string]string) []string {
	out := filepath.Join(tmp, "souffle-out")
	empty := filepath.Join(tmp, "souffle-nofacts")
	for _, d := range []string{out, empty} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return []string{err.Error()}
		}
	}
	factsDir := filepath.Join(dir, "facts")
	if _, err := os.Stat(factsDir); err != nil {
		factsDir = empty
	}
	cmd := exec.CommandContext(ctx, souffle, "-F", factsDir, "-D", out, filepath.Join(dir, "program.dl"))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if perr != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			return []string{fmt.Sprintf("we reject the program (%v) but souffle accepted it (err=%v)\n%s", perr, err, stderr.String())}
		}
		return nil
	}
	if err != nil {
		return []string{fmt.Sprintf("souffle failed: %v\n%s", err, stderr.String())}
	}
	var problems []string
	entries, err := os.ReadDir(out)
	if err != nil {
		return []string{err.Error()}
	}
	if len(entries) != len(ours) {
		problems = append(problems, fmt.Sprintf("souffle wrote %d files, we produce %d outputs", len(entries), len(ours)))
	}
	names := make([]string, 0, len(ours))
	for name := range ours {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		theirs, err := os.ReadFile(filepath.Join(out, name+".csv"))
		if err != nil {
			problems = append(problems, fmt.Sprintf("souffle did not write %s.csv: %v", name, err))
			continue
		}
		a, b := strings.Join(sortedLines(ours[name]), ""), strings.Join(sortedLines(string(theirs)), "")
		if a != b {
			problems = append(problems, fmt.Sprintf("%s: parity failure (both sides sorted)\n ours:\n%s\nsouffle:\n%s", name, a, b))
		}
	}
	return problems
}

func writeExpected(t *testing.T, expDir string, perr error, res *Result) {
	t.Helper()
	if t.Failed() {
		t.Fatal("not updating expected/: the Soufflé comparison failed")
	}
	if err := os.RemoveAll(expDir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(expDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if perr != nil {
		if err := os.WriteFile(filepath.Join(expDir, "error.txt"), []byte(perr.Error()+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := res.WriteCSV(expDir); err != nil {
		t.Fatal(err)
	}
}

// TestParityHarnessDetectsDifferences guards the harness itself: a changed
// row, a missing relation and an accepted-but-should-fail program must all
// be reported, so a green parity run cannot be vacuous.
func TestParityHarnessDetectsDifferences(t *testing.T) {
	souffle, err := exec.LookPath("souffle")
	if err != nil {
		t.Skip("souffle not on PATH: the harness self-test needs it")
	}
	dir := filepath.Join("testdata", "programs", "transitive_closure")
	src, err := os.ReadFile(filepath.Join(dir, "program.dl"))
	if err != nil {
		t.Fatal(err)
	}
	prog, err := Parse(string(src), "program.dl")
	if err != nil {
		t.Fatal(err)
	}
	in, err := ReadFacts(filepath.Join(dir, "facts"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := prog.Run(in, Options{})
	if err != nil {
		t.Fatal(err)
	}
	good := map[string]string{"path": res.FormatCSV("path")}
	if p := souffleProblems(t.Context(), souffle, dir, t.TempDir(), nil, good); len(p) != 0 {
		t.Fatalf("unexpected problems: %v", p)
	}
	tampered := map[string]string{"path": strings.Replace(good["path"], "a\tb\n", "a\tz\n", 1)}
	if p := souffleProblems(t.Context(), souffle, dir, t.TempDir(), nil, tampered); len(p) != 1 || !strings.Contains(p[0], "parity failure") {
		t.Fatalf("a changed row was not reported: %v", p)
	}
	if p := souffleProblems(t.Context(), souffle, dir, t.TempDir(), nil, map[string]string{}); len(p) != 1 {
		t.Fatalf("an extra souffle output was not reported: %v", p)
	}
	if p := souffleProblems(t.Context(), souffle, dir, t.TempDir(), errors.New("rejected"), nil); len(p) != 1 {
		t.Fatalf("souffle accepting a program we reject was not reported: %v", p)
	}
}
