package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/gates"
	"github.com/RamXX/machinery/internal/testgit"
	"gopkg.in/yaml.v3"
)

const cliReviewLimits = "Hashes bind the observed files and scope; they do not prove tests ran, reviewer identity, or judgment correctness. Top-level Git administration and the exact Machinery attestation record are excluded; applications that use them as runtime inputs are outside this review boundary."
const cliReviewHandler = "package fixture\nfunc Handle() string { return \"accepted\" }\n"
const cliReviewTest = "package fixture\nimport \"testing\"\n// ORACLESET{machines/Thing.oracle.md}\nfunc TestAction(t *testing.T) { if Handle() != \"accepted\" { t.Fatal(\"wrong action\") } }\n"

type cliReviewFixture struct{ root, design, impl string }
type cliReviewResult struct {
	out, stderr string
	code        int
}

func cliReviewWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
func cliReviewRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func cliReviewHash(b []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(b)) }
func cliReviewGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	args = append([]string{"-c", "user.name=Receipt test", "-c", "user.email=receipt@example.invalid"}, args...)
	b, err := testgit.Run(t.Context(), root, args...)
	if err != nil {
		t.Fatalf("real git %v: %v\n%s", args, err, b)
	}
	return strings.TrimSpace(string(b))
}
func newCLIReviewFixture(t *testing.T) *cliReviewFixture {
	t.Helper()
	root := t.TempDir()
	f := &cliReviewFixture{root, filepath.Join(root, "design"), filepath.Join(root, "src")}
	for path, body := range map[string]string{"design/BUILD.md": "# Build\n", "src/handler.go": cliReviewHandler, "src/handler_test.go": cliReviewTest, "src/.gitignore": ".ignored/\n", "src/.ignored/config": "enabled\n", "src/config.yaml": "enabled: true\n"} {
		cliReviewWrite(t, filepath.Join(root, path), body)
	}
	cliReviewGit(t, root, "init", "-q")
	cliReviewGit(t, root, "add", ".")
	cliReviewGit(t, root, "commit", "-q", "-m", "reviewed source")
	return f
}
func cliReviewExec(t *testing.T, binary string, f *cliReviewFixture, args ...string) cliReviewResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = f.root
	for _, e := range os.Environ() {
		key, _, _ := strings.Cut(e, "=")
		if key != "MACHINERY_COMMIT" && !strings.HasPrefix(key, "GIT_") {
			cmd.Env = append(cmd.Env, e)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	start := time.Now()
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("CLI timeout, not behavioral RED: %v", ctx.Err())
	}
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("CLI setup: %v", err)
		}
		code = exit.ExitCode()
	}
	t.Logf("CLI %v exit=%d duration=%s stdout-bytes=%d stderr-bytes=%d", args, code, time.Since(start), out.Len(), stderr.Len())
	return cliReviewResult{out.String(), stderr.String(), code}
}
func cliReviewGenerate(t *testing.T, binary string, f *cliReviewFixture, claim, kind string) map[string]any {
	t.Helper()
	args := []string{"attest", "--design", f.design, "--claim", claim, "--kind", kind, "--attestor", "R", "--date", "2026-09-05"}
	if kind == "current" {
		args = append(args, "--impl", f.impl)
	}
	r := cliReviewExec(t, binary, f, args...)
	if r.code != 0 {
		t.Fatalf("B generation interface unavailable; dependent C mutation NOT YET EXERCISED: code=%d stdout=%q stderr=%q", r.code, r.out, r.stderr)
	}
	if !strings.Contains(r.stderr, cliReviewLimits) {
		t.Fatalf("generation must state exact limits on stderr: %q", r.stderr)
	}
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(r.out), &doc); err != nil {
		t.Fatalf("generation is not a complete YAML document: %v\n%s", err, r.out)
	}
	rows, ok := doc["attestations"].([]any)
	if doc["attestation_version"] != 2 || !ok || len(rows) != 1 {
		t.Fatalf("generation document shape=%v", doc)
	}
	row, ok := rows[0].(map[string]any)
	if !ok || row["claim"] != claim || row["kind"] != kind {
		t.Fatalf("generation row=%v", rows[0])
	}
	return row
}
func cliReviewSave(t *testing.T, f *cliReviewFixture, version int, rows ...map[string]any) {
	t.Helper()
	body, err := yaml.Marshal(map[string]any{"attestation_version": version, "attestations": rows})
	if err != nil {
		t.Fatal(err)
	}
	cliReviewWrite(t, filepath.Join(f.design, gates.AttestationsFileName), string(body))
}
func cliReviewBoundRow(t *testing.T, f *cliReviewFixture, claim, kind string, paths ...string) map[string]any {
	t.Helper()
	covers := []map[string]string{}
	for _, p := range paths {
		covers = append(covers, map[string]string{"path": p, "hash": cliReviewHash(cliReviewRead(t, filepath.Join(f.design, p)))})
	}
	row := map[string]any{"claim": claim, "attestor": "R", "date": "2026-09-05", "covers": covers}
	if kind != "" {
		row["kind"] = kind
	}
	return row
}
func cliReviewCurrent(t *testing.T, r cliReviewResult) {
	t.Helper()
	if r.code != 0 || !strings.Contains(r.out, "1 current implementation reviews") || !strings.Contains(r.out, cliReviewLimits) {
		t.Fatalf("C unchanged current control failed; mutation NOT YET EXERCISED: code=%d stdout=%s stderr=%s", r.code, r.out, r.stderr)
	}
}
func cliReviewRejected(t *testing.T, r cliReviewResult, category, path string) {
	t.Helper()
	text := r.out + r.stderr
	if r.code != 1 || !strings.Contains(text, category) || !strings.Contains(text, path) || strings.Contains(r.out, "1 current implementation reviews") {
		t.Fatalf("want exit 1, %s naming %q, no current success: %+v", category, path, r)
	}
}
func cliReviewCheck(t *testing.T, binary string, f *cliReviewFixture, extra ...string) cliReviewResult {
	t.Helper()
	args := []string{"check", f.design, "--impl", f.impl, "--gate", "gv"}
	return cliReviewExec(t, binary, f, append(args, extra...)...)
}
func cliReviewHistory(t *testing.T, f *cliReviewFixture) string {
	t.Helper()
	cliReviewWrite(t, filepath.Join(f.design, "BUILD.md"), "# BUILD\n\n## Build plan\n\n**M0 - Walking skeleton.** DoD: T-CMD-01 and CMD-abc123 green.\nStatus: closed\n")
	cliReviewWrite(t, filepath.Join(f.design, "machines", "Thing.oracle.md"), "# Generated transition oracle: Thing\n\n| test id | stable id | source | trigger | guard | target | actions |\n|---|---|---|---|---|---|---|\n| T-CMD-01 | CMD-abc123 | A | on:go | - | B | send |\n")
	cliReviewGit(t, f.root, "add", ".")
	cliReviewGit(t, f.root, "commit", "-q", "-m", "reviewed milestone")
	anchor := cliReviewGit(t, f.root, "rev-parse", "HEAD")
	cliReviewWrite(t, filepath.Join(f.design, "acceptance", "M0.yaml"), "milestone: 0\ncommit: "+anchor+"\nverdict: ACCEPTED\ndod_ids: [T-CMD-01, CMD-abc123]\nattestations: [The fixture reviewer recorded its judgment.]\nfindings: []\nreviewer: R\ndate: 2026-09-05\n")
	cliReviewGit(t, f.root, "add", ".")
	cliReviewGit(t, f.root, "commit", "-q", "-m", "historical acceptance")
	return anchor
}

func TestAttestImplementationCLI(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "machinery")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build real CLI (not RED proof): %v\n%s", err, out)
	}
	t.Logf("built real CLI %s digest=%s", binary, cliReviewHash(cliReviewRead(t, binary)))
	t.Run("D-filehash-claims-plan", func(t *testing.T) {
		f := newCLIReviewFixture(t)
		path := filepath.Join(f.design, "ARCHITECTURE.md")
		cliReviewWrite(t, path, "# Architecture\n")
		r := cliReviewExec(t, binary, f, "attest", path)
		if r.code != 0 || r.out != cliReviewHash(cliReviewRead(t, path))+"  "+path+"\n" {
			t.Fatalf("D filehash compatibility: %+v", r)
		}
		r = cliReviewExec(t, binary, f, "attest", "--claims", "absent-but-claims-takes-precedence")
		if r.code != 0 || r.out != strings.Join(gates.AttestationClaimIDs(), "\n")+"\n" {
			t.Fatalf("D claims compatibility: %+v", r)
		}
		var rows []map[string]any
		for _, id := range gates.AttestationClaimIDs() {
			if strings.HasPrefix(id, "g2.") {
				rows = append(rows, cliReviewBoundRow(t, f, id, "", "ARCHITECTURE.md"))
			}
		}
		cliReviewSave(t, f, 1, rows...)
		r = cliReviewExec(t, binary, f, "check", f.design, "--gate", "gv")
		if r.code != 0 || !strings.Contains(r.out, "6 attested claims") {
			t.Fatalf("D legacy plan: %+v", r)
		}
	})
	t.Run("D-real-Ga-ancestor", func(t *testing.T) {
		f := newCLIReviewFixture(t)
		anchor := cliReviewHistory(t, f)
		r := cliReviewExec(t, binary, f, "check", f.design, "--gate", "ga")
		if r.code != 0 || !strings.Contains(r.out, "commit bindings verified") {
			t.Fatalf("D Ga ancestor %s: %+v", anchor, r)
		}
	})
	for _, claim := range []string{"gt.conformance-test-shape", "g4.standin-coverage", "g4.pack-event-discipline"} {
		for _, changed := range []bool{false, true} {
			t.Run(fmt.Sprintf("A-legacy/%s/changed=%t", claim, changed), func(t *testing.T) {
				f := newCLIReviewFixture(t)
				paths := []string{"BUILD.md"}
				if claim == "g4.pack-event-discipline" {
					cliReviewWrite(t, filepath.Join(f.design, "pack", "pack.yaml"), "events: []\n")
					paths = []string{"pack/pack.yaml"}
				}
				cliReviewSave(t, f, 1, cliReviewBoundRow(t, f, claim, "", paths...))
				if changed {
					cliReviewWrite(t, filepath.Join(f.impl, "handler_test.go"), "package fixture\n// ORACLESET{machines/Thing.oracle.md}\n")
					cliReviewWrite(t, filepath.Join(f.impl, "handler.go"), "package fixture\nfunc Handle() string { return \"forbidden\" }\n")
				}
				r := cliReviewCheck(t, binary, f)
				t.Logf("A actual legacy result: %+v", r)
				cliReviewRejected(t, r, "GV_MISSING_IMPLEMENTATION_SUBJECT", claim)
			})
		}
	}
	t.Run("B-generation-inventory", func(t *testing.T) {
		f := newCLIReviewFixture(t)
		row := cliReviewGenerate(t, binary, f, "gt.conformance-test-shape", "current")
		scope, ok := row["implementation"].(map[string]any)
		if !ok || scope["root"] != "../src" || scope["policy"] != "full-root-v1" {
			t.Fatalf("scope identity=%v", row["implementation"])
		}
		entries, ok := scope["entries"].([]any)
		if !ok {
			t.Fatal("inventory absent")
		}
		var actual, want []string
		var digest strings.Builder
		digest.WriteString("machinery-attestation-scope-v1\nroot\t../src\npolicy\tfull-root-v1\nexclude\tvcs-root:.git\nexclude\tevidence:none\n")
		for _, value := range entries {
			e := value.(map[string]any)
			p := e["path"].(string)
			actual = append(actual, p)
			info, err := os.Stat(filepath.Join(f.impl, p))
			if err != nil {
				t.Fatal(err)
			}
			if e["mode"] != fmt.Sprintf("%04o", info.Mode().Perm()) {
				t.Fatalf("mode for %s=%v", p, e)
			}
			if info.IsDir() {
				if e["type"] != "directory" {
					t.Fatal(e)
				}
				fmt.Fprintf(&digest, "directory\t%s\t%s\n", p, e["mode"])
			} else {
				h := cliReviewHash(cliReviewRead(t, filepath.Join(f.impl, p)))
				if e["type"] != "file" || e["hash"] != h || fmt.Sprint(e["size"]) != fmt.Sprint(info.Size()) {
					t.Fatalf("file metadata %s=%v", p, e)
				}
				fmt.Fprintf(&digest, "file\t%s\t%s\t%d\t%s\n", p, e["mode"], info.Size(), h)
			}
		}
		if err := filepath.WalkDir(f.impl, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(f.impl, path)
			want = append(want, filepath.ToSlash(rel))
			return err
		}); err != nil {
			t.Fatal(err)
		}
		sort.Strings(want)
		if strings.Join(actual, "\n") != strings.Join(want, "\n") || scope["hash"] != cliReviewHash([]byte(digest.String())) {
			t.Fatalf("inventory/digest incomplete: actual=%v want=%v scope=%v", actual, want, scope)
		}
		cliReviewSave(t, f, 2, row)
		cliReviewCurrent(t, cliReviewCheck(t, binary, f))
	})
	for _, mutation := range []string{"test-assertion", "event-handler", "ignored-addition", "config", "missing-root", "alias", "missing-input"} {
		t.Run("C-real-process/"+mutation, func(t *testing.T) {
			f := newCLIReviewFixture(t)
			row := cliReviewGenerate(t, binary, f, "gt.conformance-test-shape", "current")
			cliReviewSave(t, f, 2, row)
			cliReviewCurrent(t, cliReviewCheck(t, binary, f))
			t.Log("C unchanged real CLI control passed; applying challenge")
			switch mutation {
			case "test-assertion":
				cliReviewWrite(t, filepath.Join(f.impl, "handler_test.go"), "package fixture\n// ORACLESET{machines/Thing.oracle.md}\n")
				cliReviewRejected(t, cliReviewCheck(t, binary, f), "GV_STALE_CONTENT", "handler_test.go")
			case "event-handler":
				cliReviewWrite(t, filepath.Join(f.impl, "handler.go"), strings.ReplaceAll(cliReviewHandler, "accepted", "rejected"))
				cliReviewRejected(t, cliReviewCheck(t, binary, f), "GV_STALE_CONTENT", "handler.go")
			case "ignored-addition":
				cliReviewWrite(t, filepath.Join(f.impl, ".ignored", "handler"), "behavior\n")
				cliReviewRejected(t, cliReviewCheck(t, binary, f), "GV_SCOPE_INVENTORY", ".ignored/handler")
			case "config":
				cliReviewWrite(t, filepath.Join(f.impl, "config.yaml"), "enabled: false\n")
				cliReviewRejected(t, cliReviewCheck(t, binary, f), "GV_STALE_CONTENT", "config.yaml")
			case "missing-root":
				cliReviewRejected(t, cliReviewExec(t, binary, f, "check", f.design, "--gate", "gv"), "GV_IMPL_REQUIRED", "")
			case "alias", "missing-input":
				category := "GV_SCOPE_ALIAS"
				path := "handler"
				if mutation == "alias" {
					if err := os.Link(filepath.Join(f.impl, "handler.go"), filepath.Join(f.impl, "handler-alias.go")); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.Remove(filepath.Join(f.design, "BUILD.md")); err != nil {
						t.Fatal(err)
					}
					category = "BUILD.md"
					path = ""
				}
				r := cliReviewExec(t, binary, f, "attest", "--design", f.design, "--claim", "gt.conformance-test-shape", "--kind", "current", "--impl", f.impl, "--attestor", "R", "--date", "2026-09-05")
				cliReviewRejected(t, r, category, path)
				if r.out != "" {
					t.Fatalf("ordinary real renderer error emitted partial generation stdout: %q", r.out)
				}
			}
		})
	}
	t.Run("C-historical-current-replay", func(t *testing.T) {
		f := newCLIReviewFixture(t)
		anchor := cliReviewHistory(t, f)
		f.impl = f.root
		current := cliReviewGenerate(t, binary, f, "gt.conformance-test-shape", "current")
		historical := cliReviewGenerate(t, binary, f, "ga.review-quality", "historical")
		plan := cliReviewGenerate(t, binary, f, "g4.zero-context", "plan")
		cliReviewSave(t, f, 2, current, historical, plan)
		check := func(extra ...string) cliReviewResult {
			return cliReviewExec(t, binary, f, append([]string{"check", f.design, "--impl", f.impl, "--gate", "ga,gv"}, extra...)...)
		}
		cliReviewCurrent(t, check())
		receipt := cliReviewRead(t, filepath.Join(f.design, gates.AttestationsFileName))
		current["note"] = "attribution note corrected"
		cliReviewSave(t, f, 2, current, historical, plan)
		cliReviewGit(t, f.root, "add", "design/attestations.yaml")
		cliReviewGit(t, f.root, "commit", "-q", "-m", "evidence-only update")
		r := check()
		cliReviewCurrent(t, r)
		if !strings.Contains(r.out, "historical review records") {
			t.Fatalf("history not explicitly distinguished: %s", r.out)
		}
		cliReviewWrite(t, filepath.Join(f.root, "src", "handler.go"), strings.ReplaceAll(cliReviewHandler, "accepted", "rejected"))
		cliReviewGit(t, f.root, "add", "src/handler.go")
		cliReviewGit(t, f.root, "commit", "-q", "-m", "handler changed")
		cliReviewWrite(t, filepath.Join(f.design, gates.AttestationsFileName), string(receipt))
		for _, extra := range [][]string{nil, {"--commit", anchor}} {
			r := check(extra...)
			cliReviewRejected(t, r, "GV_STALE_CONTENT", "src/handler.go")
			if !strings.Contains(r.out, "commit bindings verified") {
				t.Fatalf("Ga ancestor did not remain independently checked: %s", r.out)
			}
		}
	})
	t.Run("C-plan-warning-promotion", func(t *testing.T) {
		f := newCLIReviewFixture(t)
		current := cliReviewGenerate(t, binary, f, "gt.conformance-test-shape", "current")
		plan := cliReviewGenerate(t, binary, f, "g4.zero-context", "plan")
		cliReviewSave(t, f, 2, current, plan)
		cliReviewCurrent(t, cliReviewCheck(t, binary, f, "--warnings-as-errors"))
		current["kind"] = "plan"
		delete(current, "implementation")
		cliReviewSave(t, f, 2, current, plan)
		r := cliReviewCheck(t, binary, f)
		if r.code != 0 || !strings.Contains(r.out, "plan only; current implementation review missing") {
			t.Fatalf("ordinary plan must warn: %+v", r)
		}
		r = cliReviewCheck(t, binary, f, "--warnings-as-errors")
		cliReviewRejected(t, r, "plan only; current implementation review missing", "")
		if !strings.Contains(r.out, "1 blocking (ERROR/DRIFT/warning)") {
			t.Fatalf("the sole warning was not isolated: %s", r.out)
		}
	})
	t.Run("C-complete-sole-current-warning", func(t *testing.T) {
		root := t.TempDir()
		f := &cliReviewFixture{root, filepath.Join(root, "design"), filepath.Join(root, "src")}
		for _, path := range []string{f.design, f.impl} {
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		copyDirInto(t, "../../examples/go-crm/design", f.design)
		copyDirInto(t, "../../examples/go-crm/impl", f.impl)
		cliReviewGit(t, f.root, "init", "-q")
		cliReviewGit(t, f.root, "add", ".")
		cliReviewGit(t, f.root, "commit", "-q", "-m", "complete design fixture")
		anchor := cliReviewGit(t, f.root, "rev-parse", "HEAD")
		files, err := filepath.Glob(filepath.Join(f.design, "acceptance", "*.yaml"))
		if err != nil || len(files) == 0 {
			t.Fatalf("acceptance fixture: %v %v", files, err)
		}
		for _, path := range files {
			lines := strings.Split(string(cliReviewRead(t, path)), "\n")
			commits := 0
			for i, line := range lines {
				if strings.HasPrefix(line, "commit: ") {
					lines[i] = fmt.Sprintf("commit: %q", anchor)
					commits++
				}
			}
			if commits != 1 {
				t.Fatalf("acceptance fixture %s: want exactly one root commit line, got %d", path, commits)
			}
			cliReviewWrite(t, path, strings.Join(lines, "\n"))
		}
		var old map[string]any
		if err := yaml.Unmarshal(cliReviewRead(t, filepath.Join(f.design, gates.AttestationsFileName)), &old); err != nil {
			t.Fatal(err)
		}
		var rows []map[string]any
		for _, value := range old["attestations"].([]any) {
			id := value.(map[string]any)["claim"].(string)
			kind := "plan"
			if id == "gt.conformance-test-shape" {
				kind = "current"
			}
			if id == "ga.review-quality" {
				kind = "historical"
			}
			rows = append(rows, cliReviewGenerate(t, binary, f, id, kind))
		}
		cliReviewSave(t, f, 2, rows...)
		check := func() cliReviewResult {
			return cliReviewExec(t, binary, f, "check", f.design, "--impl", f.impl, "--complete")
		}
		r := check()
		cliReviewCurrent(t, r)
		if !strings.Contains(r.out, "0 blocking (ERROR/DRIFT/warning)") {
			t.Fatalf("complete fixture has unrelated findings: %s", r.out)
		}
		for _, row := range rows {
			if row["claim"] == "gt.conformance-test-shape" {
				row["kind"] = "plan"
				delete(row, "implementation")
			}
		}
		cliReviewSave(t, f, 2, rows...)
		r = check()
		cliReviewRejected(t, r, "plan only; current implementation review missing", "")
		if !strings.Contains(r.out, "1 blocking (ERROR/DRIFT/warning)") {
			t.Fatalf("complete must fail solely on missing current review: %s", r.out)
		}
	})
}
