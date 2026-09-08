package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// These black-box tests define the contributor CLI contract, not its internal
// parser, provisioning or cleanup implementation. Fixture programs are actual
// native tests compiled/executed by Go; none replace a runtime executable.
type laneTestReport struct {
	Version int    `json:"version"`
	Lane    string `json:"lane"`
	Status  string `json:"status"`
	Suites  []struct {
		ID       string `json:"id"`
		Selected int    `json:"selected"`
		Started  int    `json:"started"`
		Passed   int    `json:"passed"`
		Failed   int    `json:"failed"`
		Skipped  int    `json:"skipped"`
		Tests    []struct {
			Name   string `json:"name"`
			Source string `json:"source"`
			Status string `json:"status"`
		} `json:"tests"`
		Events    string `json:"events_file"`
		EventsSHA string `json:"events_sha256"`
	} `json:"suites"`
	Cleanup struct {
		Status string `json:"status"`
	} `json:"cleanup"`
}

func laneWrite(t *testing.T, root, name, body string) {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func laneJSON(t *testing.T, root, name string, value any) {
	t.Helper()
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	laneWrite(t, root, name, string(b)+"\n")
}

func laneSuite(id, file string, tests ...string) map[string]any {
	return map[string]any{"id": id, "lane": "required", "adapter": "go-json", "package": "./sample", "source_files": []string{file}, "tests": tests, "runtimes": []string{"go"}, "timeout": "30s", "stdout_limit": 1 << 20, "stderr_limit": 1 << 20}
}

func laneFixture(t *testing.T, body string) (string, map[string]any) {
	t.Helper()
	root := t.TempDir()
	laneWrite(t, root, "go.mod", "module lane.example/fixture\n\ngo 1.27.0\n")
	laneWrite(t, root, "sample/native.go", "package sample\n")
	laneWrite(t, root, "sample/pilot_integration_test.go", "//go:build machinery_integration\n\npackage sample\nimport (\"testing\"; \"os\"; \"time\"; \"fmt\")\nvar _ = os.Getpid\nvar _ = time.Now\nvar _ = fmt.Println\n"+body)
	repo := laneRepo(t)
	for _, name := range []string{"schema.json", "runtime-pins.json"} {
		b, err := os.ReadFile(filepath.Join(repo, "testdata/integration-lanes", name))
		if err != nil {
			t.Fatal(err)
		}
		laneWrite(t, root, "testdata/integration-lanes/"+name, string(b))
	}
	suite := laneSuite("fixture", "sample/pilot_integration_test.go", "TestPilot")
	laneSave(t, root, suite)
	return root, suite
}

func laneSave(t *testing.T, root string, suites ...map[string]any) {
	t.Helper()
	laneJSON(t, root, "testdata/integration-lanes/pilot.json", map[string]any{"version": 1, "suites": suites})
}

func laneRepo(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func laneInvoke(t *testing.T, root string, extra ...string) (int, string, laneTestReport) {
	t.Helper()
	work := t.TempDir()
	report := filepath.Join(work, "report.json")
	args := []string{"--root", root, "--lane", "required", "--report", report, "--work-dir", filepath.Join(work, "owned"), "--cache-dir", filepath.Join(work, "cache")}
	args = append(args, extra...)
	var out, errout bytes.Buffer
	status := run(args, &out, &errout)
	var got laneTestReport
	if b, err := os.ReadFile(report); err == nil {
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("malformed report: %v", err)
		}
	}
	return status, out.String() + errout.String(), got
}

func laneRequireFailure(t *testing.T, root, diagnostic string) {
	t.Helper()
	rc, out, report := laneInvoke(t, root)
	if rc == 0 || report.Status == "passed" || !strings.Contains(strings.ToLower(out), diagnostic) {
		t.Fatalf("require rejection diagnostic %q, status=%d report=%+v output=%s", diagnostic, rc, report, out)
	}
}

func TestLaneRealGoPositiveControl(t *testing.T) {
	root, _ := laneFixture(t, `func TestPilot(t *testing.T) { if 6*7 != 42 { t.Fatal("wrong result") } }`)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-json", "-count=1", "-tags", "machinery_integration", "./sample")
	cmd.Dir = root
	b, err := cmd.CombinedOutput()
	if err != nil || !bytes.Contains(b, []byte(`"Action":"run"`)) || !bytes.Contains(b, []byte(`"Action":"pass","Package":"lane.example/fixture/sample","Test":"TestPilot"`)) {
		t.Fatalf("actual native positive control: %v %s", err, b)
	}
}

func TestLaneRealGoDuplicateExecutionControl(t *testing.T) {
	root, _ := laneFixture(t, `func TestMain(m *testing.M) { a:=m.Run(); b:=m.Run(); os.Exit(a|b) }; func TestPilot(t *testing.T) {}`)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-json", "-count=1", "-tags", "machinery_integration", "./sample")
	cmd.Dir = root
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("real repeated-native control failed: %v %s", err, b)
	}
	starts, passes := 0, 0
	for _, line := range bytes.Split(bytes.TrimSpace(b), []byte("\n")) {
		var e struct{ Action, Test string }
		if err := json.Unmarshal(line, &e); err != nil {
			t.Fatal(err)
		}
		if e.Test == "TestPilot" {
			if e.Action == "run" {
				starts++
			}
			if e.Action == "pass" {
				passes++
			}
		}
	}
	if starts != 2 || passes != 2 {
		t.Fatalf("duplicate control did not execute exactly twice: starts=%d passes=%d events=%s", starts, passes, b)
	}
}

func TestLaneSuccessRequiresActualNonzeroExecution(t *testing.T) {
	root, _ := laneFixture(t, `func TestPilot(t *testing.T) { if err:=os.WriteFile("executed",[]byte("42"),0600);err!=nil { t.Fatal(err) } }`)
	for attempt := 0; attempt < 2; attempt++ {
		_ = os.Remove(filepath.Join(root, "sample/executed"))
		rc, out, report := laneInvoke(t, root)
		if rc != 0 {
			t.Fatalf("actual passing test must execute, status=%d output=%s", rc, out)
		}
		b, err := os.ReadFile(filepath.Join(root, "sample/executed"))
		if err != nil || string(b) != "42" {
			t.Fatalf("execution marker missing (cached run?): %q %v", b, err)
		}
		if report.Version != 1 || report.Lane != "required" || report.Status != "passed" || len(report.Suites) != 1 || report.Cleanup.Status != "passed" {
			t.Fatalf("incomplete report: %+v", report)
		}
		s := report.Suites[0]
		if s.ID != "fixture" || s.Selected != 1 || s.Started != 1 || s.Passed != 1 || s.Failed != 0 || s.Skipped != 0 || len(s.Tests) != 1 || s.Tests[0].Name != "TestPilot" || s.Tests[0].Source != "sample/pilot_integration_test.go" || s.Tests[0].Status != "passed" || s.Events == "" || len(s.EventsSHA) != 64 {
			t.Fatalf("execution not accounted exactly: %+v", s)
		}
		events, err := os.ReadFile(s.Events)
		if err != nil {
			t.Fatalf("native events were not retained: %v", err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(events)) != s.EventsSHA {
			t.Fatal("report hash does not bind retained native events")
		}
		started, passed := 0, 0
		for _, line := range bytes.Split(bytes.TrimSpace(events), []byte("\n")) {
			var event struct{ Action, Test, Package string }
			if err := json.Unmarshal(line, &event); err != nil {
				t.Fatalf("not real Go JSON events: %v", err)
			}
			if event.Package == "lane.example/fixture/sample" && event.Test == "TestPilot" {
				if event.Action == "run" {
					started++
				}
				if event.Action == "pass" {
					passed++
				}
			}
		}
		if started != 1 || passed != 1 {
			t.Fatalf("native stream does not prove exact execution: starts=%d passes=%d", started, passed)
		}
	}
}

func TestLaneClosedInventoryRejectsUnprovedSelections(t *testing.T) {
	cases := []struct {
		name, diagnostic string
		mutate           func(*testing.T, string, map[string]any)
	}{
		{"empty", "empty", func(t *testing.T, r string, s map[string]any) { laneSave(t, r) }},
		{"unknown-field", "unknown", func(t *testing.T, r string, s map[string]any) { s["skip_if_missing"] = true; laneSave(t, r, s) }},
		{"unknown-version", "version", func(t *testing.T, r string, s map[string]any) {
			laneJSON(t, r, "testdata/integration-lanes/pilot.json", map[string]any{"version": 2, "suites": []any{s}})
		}},
		{"duplicate-suite", "duplicate", func(t *testing.T, r string, s map[string]any) { laneSave(t, r, s, s) }},
		{"duplicate-test", "duplicate", func(t *testing.T, r string, s map[string]any) {
			s["tests"] = []string{"TestPilot", "TestPilot"}
			laneSave(t, r, s)
		}},
		{"future-test", "TestFuture", func(t *testing.T, r string, s map[string]any) { s["tests"] = []string{"TestFuture"}; laneSave(t, r, s) }},
		{"unregistered-source", "unregistered", func(t *testing.T, r string, s map[string]any) {
			laneWrite(t, r, "sample/forgotten_integration_test.go", "//go:build machinery_integration\n\npackage sample\nimport \"testing\"\nfunc TestForgotten(t *testing.T) { t.Fatal(\"must not vanish\") }\n")
		}},
		{"omitted-case", "unregistered", func(t *testing.T, r string, s map[string]any) {
			p := filepath.Join(r, "sample/pilot_integration_test.go")
			b, e := os.ReadFile(p)
			if e != nil {
				t.Fatal(e)
			}
			laneWrite(t, r, "sample/pilot_integration_test.go", string(b)+"\nfunc TestUnregistered(t *testing.T) { t.Fatal(\"must not vanish\") }\n")
		}},
		{"parent-traversal", "path", func(t *testing.T, r string, s map[string]any) {
			s["source_files"] = []string{"../outside_test.go"}
			laneSave(t, r, s)
		}},
		{"absolute-path", "path", func(t *testing.T, r string, s map[string]any) {
			s["source_files"] = []string{filepath.Join(r, "sample/pilot_integration_test.go")}
			laneSave(t, r, s)
		}},
		{"symlink-source", "symlink", func(t *testing.T, r string, s map[string]any) {
			if e := os.Symlink(filepath.Join(r, "sample/pilot_integration_test.go"), filepath.Join(r, "sample/link_integration_test.go")); e != nil {
				t.Fatal(e)
			}
			s["source_files"] = []string{"sample/link_integration_test.go"}
			laneSave(t, r, s)
		}},
		{"shell-adapter", "adapter", func(t *testing.T, r string, s map[string]any) { s["adapter"] = "shell"; laneSave(t, r, s) }},
		{"filtered-package", "package", func(t *testing.T, r string, s map[string]any) {
			s["package"] = "./sample -run TestNothing"
			laneSave(t, r, s)
		}},
		{"zero-bound", "limit", func(t *testing.T, r string, s map[string]any) { s["stdout_limit"] = 0; laneSave(t, r, s) }},
		{"orphan-fragment", "fragment", func(t *testing.T, r string, s map[string]any) {
			laneWrite(t, r, "testdata/integration-lanes/forgotten.yaml", "suites: []\n")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, s := laneFixture(t, `func TestPilot(t *testing.T) {}`)
			tc.mutate(t, r, s)
			laneRequireFailure(t, r, strings.ToLower(tc.diagnostic))
		})
	}
}

func TestLaneDiscoversUnionAndRejectsIdentityRegisteredTwice(t *testing.T) {
	r, s := laneFixture(t, `func TestPilot(t *testing.T) {}`)
	laneWrite(t, r, "sample/second_integration_test.go", "//go:build machinery_integration\n\npackage sample\nimport \"testing\"\nfunc TestSecond(t *testing.T) {}\n")
	s2 := laneSuite("second", "sample/second_integration_test.go", "TestSecond")
	laneJSON(t, r, "testdata/integration-lanes/second.json", map[string]any{"version": 1, "suites": []any{s2}})
	rc, out, report := laneInvoke(t, r)
	if rc != 0 || len(report.Suites) != 2 || report.Suites[0].ID != "fixture" || report.Suites[1].ID != "second" {
		t.Fatalf("deterministically discovered union missing: %d %s %+v", rc, out, report)
	}
	s2["id"] = "duplicate-identity"
	s2["source_files"] = s["source_files"]
	s2["tests"] = s["tests"]
	laneJSON(t, r, "testdata/integration-lanes/second.json", map[string]any{"version": 1, "suites": []any{s2}})
	laneRequireFailure(t, r, "duplicate")
}

func TestLaneNativeFailuresCannotBecomeSuccess(t *testing.T) {
	for _, tc := range []struct{ name, body, diagnostic string }{
		{"assertion", `func TestPilot(t *testing.T) { t.Fatal("safety assertion violated") }`, "failed"},
		{"skip", `func TestPilot(t *testing.T) { t.Skip("required test must not skip") }`, "skip"},
		{"empty-process", `func TestMain(m *testing.M) { os.Exit(0) }; func TestPilot(t *testing.T) { t.Fatal("must execute") }`, "execution"},
		{"aggregate-only", `func TestMain(m *testing.M) { fmt.Println("PASS"); os.Exit(0) }; func TestPilot(t *testing.T) { t.Fatal("must execute") }`, "execution"},
		{"forged-json", `func TestMain(m *testing.M) { fmt.Println("{\"Action\":\"pass\",\"Test\":\"TestPilot\"}"); os.Exit(0) }; func TestPilot(t *testing.T) { t.Fatal("must execute") }`, "execution"},
		{"duplicate-execution", `func TestMain(m *testing.M) { a:=m.Run(); b:=m.Run(); os.Exit(a|b) }; func TestPilot(t *testing.T) {}`, "duplicate"},
		{"premature-exit", `func TestPilot(t *testing.T) { os.Exit(0) }`, "failed"},
		{"unexpected-subtest-failure", `func TestPilot(t *testing.T) { t.Run("hidden",func(t *testing.T){t.Fatal("unexpected failure")}) }`, "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) { r, _ := laneFixture(t, tc.body); laneRequireFailure(t, r, tc.diagnostic) })
	}
}

func TestLaneRuntimeAbsenceFailsBeforeTests(t *testing.T) {
	r, _ := laneFixture(t, `func TestPilot(t *testing.T) { t.Fatal("must not execute without runtime") }`)
	t.Setenv("PATH", t.TempDir())
	laneRequireFailure(t, r, "go")
}

func TestLaneNativeSelectionDoesNotProbeServices(t *testing.T) {
	r, _ := laneFixture(t, `func TestPilot(t *testing.T) { t.Fatal("runtime test entered native lane") }`)
	t.Setenv("DOCKER_HOST", "unix:///deliberately-unavailable/machinery.sock")
	t.Setenv("MACHINERY_JAVA", "/deliberately-unavailable/java")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-count=1", "./...")
	cmd.Dir = r
	b, err := cmd.CombinedOutput()
	if err != nil || !bytes.Contains(b, []byte("[no test files]")) {
		t.Fatalf("native lane probed services: %v %s", err, b)
	}
	cmd = exec.CommandContext(ctx, "go", "list", "-json", "./cmd/machinery")
	cmd.Dir = laneRepo(t)
	b, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("actual repo native selection failed: %v %s", err, b)
	}
	var pkg struct{ TestGoFiles, IgnoredGoFiles []string }
	if err = json.Unmarshal(b, &pkg); err != nil {
		t.Fatal(err)
	}
	for _, file := range pkg.TestGoFiles {
		if file == "integration_lane_test.go" {
			t.Fatal("actual runtime pilot selected by ordinary Go lane")
		}
	}
	found := false
	for _, file := range pkg.IgnoredGoFiles {
		if file == "integration_lane_test.go" {
			found = true
		}
	}
	if !found {
		t.Fatal("actual runtime pilot has no explicit native-lane exclusion")
	}
}

func TestLaneBoundsPartialNativeOutput(t *testing.T) {
	for _, tc := range []struct {
		name, body, key string
		value           any
		diagnostic      string
	}{
		{"stdout", `func TestPilot(t *testing.T) { for { fmt.Println("unbounded output cannot become a valid receipt") } }`, "stdout_limit", 2048, "limit"},
		{"stderr", `func TestPilot(t *testing.T) { for { fmt.Fprintln(os.Stderr,"unbounded stderr cannot become a valid receipt") } }`, "stderr_limit", 2048, "limit"},
		{"timeout", `func TestPilot(t *testing.T) { time.Sleep(time.Minute) }`, "timeout", "3s", "timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, s := laneFixture(t, tc.body)
			s[tc.key] = tc.value
			laneSave(t, r, s)
			start := time.Now()
			laneRequireFailure(t, r, tc.diagnostic)
			if time.Since(start) > 20*time.Second {
				t.Fatal("bounded execution exceeded 20-second outer margin")
			}
		})
	}
}

func TestLaneWiringIsMandatoryAndNativeJobsAreServiceFree(t *testing.T) {
	repo := laneRepo(t)
	for _, file := range []string{"ci.yml", "formal.yml", "nightly.yml"} {
		t.Run(file, func(t *testing.T) {
			b, e := os.ReadFile(filepath.Join(repo, ".github/workflows", file))
			if e != nil {
				t.Fatal(e)
			}
			if e := laneValidateWorkflow(b); e != nil {
				t.Error(e)
			}
		})
	}
	b, e := os.ReadFile(filepath.Join(repo, "scripts/preflight.sh"))
	if e != nil {
		t.Fatal(e)
	}
	if e := laneValidatePreflight(string(b)); e != nil {
		t.Error(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "make", "-n", "test-integration")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "go run ./scripts/integration-lane --lane required" {
		t.Errorf("Make entrypoint differs from shared lane: %v %s", err, out)
	}
}

const laneExactInvocation = "go run ./scripts/integration-lane --lane required"

// These are assertion helpers for the repository's executable wiring. They
// inspect data only, never provision or execute a candidate runtime lane.
func laneValidateWorkflow(body []byte) error {
	var wf struct {
		Jobs map[string]struct {
			RunsOn   any `yaml:"runs-on"`
			If       any `yaml:"if"`
			Continue any `yaml:"continue-on-error"`
			Steps    []struct {
				Run      string         `yaml:"run"`
				Uses     string         `yaml:"uses"`
				If       any            `yaml:"if"`
				Continue any            `yaml:"continue-on-error"`
				With     map[string]any `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if e := yaml.Unmarshal(body, &wf); e != nil {
		return e
	}
	found := 0
	permissive := func(v any) bool { return v != nil && v != false }
	for id, j := range wf.Jobs {
		for _, s := range j.Steps {
			if strings.Contains(s.Run, "./scripts/integration-lane") {
				found++
				if strings.TrimSpace(s.Run) != laneExactInvocation || j.If != nil || s.If != nil || permissive(j.Continue) || permissive(s.Continue) || fmt.Sprint(j.RunsOn) != "ubuntu-latest" {
					return fmt.Errorf("%s does not execute exact mandatory Linux lane", id)
				}
			}
			if strings.Contains(fmt.Sprint(j.RunsOn), "macos") && (strings.Contains(s.Run, "machinery_integration") || strings.Contains(s.Run, "--lane required")) {
				return fmt.Errorf("native job %s probes services", id)
			}
			if strings.Contains(s.Uses, "actions/checkout") && (strings.Contains(id, "golden") || strings.Contains(id, "integration")) && fmt.Sprint(s.With["fetch-depth"]) != "0" {
				return fmt.Errorf("%s lacks full history", id)
			}
		}
	}
	if found != 1 {
		return fmt.Errorf("require exactly one unconditional required lane, got %d", found)
	}
	return nil
}

type laneShellToken struct {
	text   string
	quoted bool
}

// A deliberately conservative shell-token reader: transparent top-level
// dispatch is required. Quoted diagnostics/comments cannot count as execution;
// shell grouping/conditionals, continuations and separators remain visible.
func laneShellLines(body string) ([][]laneShellToken, error) {
	body = strings.ReplaceAll(body, "\\\n", "")
	lines := [][]laneShellToken{{}}
	var word strings.Builder
	quoted := false
	flush := func() {
		if word.Len() > 0 || quoted {
			lines[len(lines)-1] = append(lines[len(lines)-1], laneShellToken{word.String(), quoted})
			word.Reset()
			quoted = false
		}
	}
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch {
		case c == '\'' || c == '"':
			quoted = true
			quote := c
			closed := false
			for i++; i < len(body); i++ {
				if body[i] == quote {
					closed = true
					break
				}
				if body[i] == '\\' && quote == '"' && i+1 < len(body) {
					i++
				}
				word.WriteByte(body[i])
			}
			if !closed {
				return nil, fmt.Errorf("unterminated shell quote")
			}
		case c == '#' && word.Len() == 0:
			for i < len(body) && body[i] != '\n' {
				i++
			}
			flush()
			lines = append(lines, []laneShellToken{})
		case c == '\n':
			flush()
			lines = append(lines, []laneShellToken{})
		case c == ' ' || c == '\t' || c == '\r':
			flush()
		case strings.ContainsRune(";(){}|&", rune(c)):
			flush()
			text := string(c)
			if i+1 < len(body) && body[i+1] == c && (c == '|' || c == '&') {
				text += string(c)
				i++
			}
			lines[len(lines)-1] = append(lines[len(lines)-1], laneShellToken{text, false})
		case c == '\\' && i+1 < len(body):
			i++
			word.WriteByte(body[i])
			quoted = true
		default:
			word.WriteByte(c)
		}
	}
	flush()
	return lines, nil
}

func laneValidatePreflight(body string) error {
	lines, e := laneShellLines(body)
	if e != nil {
		return e
	}
	stack := []string{}
	lane, race, formal, strict := -1, -1, -1, false
	c4, checker := -1, -1
	for line, words := range lines {
		plain := []string{}
		for _, w := range words {
			plain = append(plain, w.text)
		}
		text := strings.Join(plain, " ")
		if len(stack) == 0 && text == "set -euo pipefail" {
			strict = true
		}
		for i, w := range words {
			if strings.Contains(w.text, "SKIP_PREFLIGHT") {
				return fmt.Errorf("executable preflight bypass variable is forbidden")
			}
			if w.quoted {
				continue
			}
			if w.text == "set" && i+1 < len(words) && strings.HasPrefix(words[i+1].text, "+") {
				return fmt.Errorf("preflight disables error propagation")
			}
			if w.text == "exit" && (i+1 >= len(words) || words[i+1].text != "1") {
				return fmt.Errorf("preflight may exit successfully without required lane")
			}
			if w.text == "exec" || w.text == "eval" {
				return fmt.Errorf("opaque control transfer forbidden in preflight")
			}
			if w.text == "alias" || ((w.text == "go" || w.text == "make") && i+1 < len(words) && words[i+1].text == "(") {
				return fmt.Errorf("preflight shadows required executable")
			}
			if w.text == "./scripts/integration-lane" {
				if len(stack) != 0 || text != laneExactInvocation || !strict {
					return fmt.Errorf("lane invocation is conditional, ignored, quoted or inexact")
				}
				if lane >= 0 {
					return fmt.Errorf("duplicate lane invocation")
				}
				lane = line
			}
			if i == 0 && len(stack) == 0 && strings.HasPrefix(text, "go test -race -count=1 ./...") {
				race = line
			}
			if i == 0 && len(stack) == 0 && strings.HasPrefix(text, "make verify-formal") {
				formal = line
			}
			if w.text == "verify-c4" && i > 0 && words[i-1].text == ".bin/machinery" {
				c4 = line
			}
			if w.text == "verify-checkers" && i > 0 && words[i-1].text == ".bin/machinery" {
				checker = line
			}
			switch w.text {
			case "if":
				stack = append(stack, "fi")
			case "case":
				stack = append(stack, "esac")
			case "for", "while", "until", "select":
				stack = append(stack, "done")
			case "{":
				stack = append(stack, "}")
			case "(":
				stack = append(stack, ")")
			case "fi", "esac", "done", "}", ")":
				if len(stack) > 0 && stack[len(stack)-1] == w.text {
					stack = stack[:len(stack)-1]
				}
			}
		}
	}
	if !strict || race < 0 || lane <= race || formal <= lane || c4 <= lane || checker <= lane {
		return fmt.Errorf("missing strict native -> required -> formal/C4/checker execution order")
	}
	return nil
}

func TestLaneWiringGuardsRejectDisabledExecution(t *testing.T) {
	workflow := "jobs:\n  integration-required:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@pinned\n        with: {fetch-depth: 0}\n      - run: " + laneExactInvocation + "\n"
	preflight := "set -euo pipefail\ngo test -race -count=1 ./...\n" + laneExactInvocation + "\nmake verify-formal\n.bin/machinery verify-c4 example\n.bin/machinery verify-checkers example\n"
	if e := laneValidateWorkflow([]byte(workflow)); e != nil {
		t.Fatal(e)
	}
	if e := laneValidatePreflight(preflight); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct{ name, old, new string }{
		{"job-if", "    runs-on:", "    if: false\n    runs-on:"},
		{"job-ignore", "    runs-on:", "    continue-on-error: true\n    runs-on:"},
		{"step-if", "      - run:", "      - if: false\n        run:"},
		{"step-ignore", "      - run:", "      - continue-on-error: true\n        run:"},
		{"step-ignore-expression", "      - run:", "      - continue-on-error: ${{ true }}\n        run:"},
		{"echo-only", laneExactInvocation, "echo " + laneExactInvocation},
		{"comment-only", laneExactInvocation, "'# " + laneExactInvocation + "'"},
		{"ignored-exit", laneExactInvocation, laneExactInvocation + " || true"},
		{"false-prefix", laneExactInvocation, "false && " + laneExactInvocation},
		{"wrong-lane", laneExactInvocation, "go run ./scripts/integration-lane --lane optional"},
	} {
		t.Run("workflow/"+tc.name, func(t *testing.T) {
			if e := laneValidateWorkflow([]byte(strings.Replace(workflow, tc.old, tc.new, 1))); e == nil {
				t.Fatal("disabled workflow was accepted")
			}
		})
	}
	for _, tc := range []struct{ name, body string }{
		{"renamed-bypass-diagnostic", "if [ \"${SKIP_PREFLIGHT:-0}\" = 1 ]; then echo differently-worded; exit 0; fi\n" + preflight},
		{"unconditional-early-success", "exit 0\n" + preflight},
		{"conditional-lane", strings.Replace(preflight, laneExactInvocation, "if false; then\n"+laneExactInvocation+"\nfi", 1)},
		{"function-only", strings.Replace(preflight, laneExactInvocation, "never_called() {\n"+laneExactInvocation+"\n}", 1)},
		{"subshell-only", strings.Replace(preflight, laneExactInvocation, "(\n"+laneExactInvocation+"\n)", 1)},
		{"comment-only", strings.Replace(preflight, laneExactInvocation, "# "+laneExactInvocation, 1)},
		{"quoted-only", strings.Replace(preflight, laneExactInvocation, "echo '"+laneExactInvocation+"'", 1)},
		{"ignored-failure", strings.Replace(preflight, laneExactInvocation, laneExactInvocation+" || true", 1)},
		{"disabled-errexit", strings.Replace(preflight, laneExactInvocation, "set +e\n"+laneExactInvocation, 1)},
		{"wrong-argv", strings.Replace(preflight, laneExactInvocation, laneExactInvocation+" --lane optional", 1)},
		{"wrong-order", strings.Replace(preflight, "make verify-formal", "", 1) + "\n" + laneExactInvocation},
		{"shadowed-go", "go() { return 0; }\n" + preflight},
		{"early-consumer", ".bin/machinery verify-c4 example\n" + strings.Replace(preflight, ".bin/machinery verify-c4 example", "", 1)},
		{"removed-c4", strings.Replace(preflight, ".bin/machinery verify-c4 example", "", 1)},
		{"removed-checker", strings.Replace(preflight, ".bin/machinery verify-checkers example", "", 1)},
	} {
		t.Run("preflight/"+tc.name, func(t *testing.T) {
			if e := laneValidatePreflight(tc.body); e == nil {
				t.Fatal("bypassed preflight was accepted")
			}
		})
	}
}

func TestLanePreflightFailurePropagationControl(t *testing.T) {
	// Small real Bash programs establish sensitivity of the structural guards.
	// They are not preflight itself and do not replace any native test runtime.
	for _, tc := range []struct {
		name, program string
		pass          bool
	}{
		{"strict", "set -euo pipefail\nfalse\nprintf reached", false},
		{"ignored", "set -euo pipefail\nfalse || true\nprintf reached", true},
		{"conditional", "set -euo pipefail\nif false; then false; fi\nprintf reached", true},
		{"renamed-bypass", "set -euo pipefail\nif [ \"${SKIP_PREFLIGHT:-0}\" = 1 ]; then printf reached; exit 0; fi\nfalse", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "bash", "--noprofile", "--norc", "-c", tc.program)
			cmd.Env = append(os.Environ(), "SKIP_PREFLIGHT=1")
			out, e := cmd.CombinedOutput()
			if (e == nil) != tc.pass || (string(out) == "reached") != tc.pass {
				t.Fatalf("real shell bypass control: err=%v out=%q", e, out)
			}
		})
	}
}

// TestLaneStreamFailureNeverFormatsANilError freezes the diagnostic contract
// broken on the first hosted run: a guarded job that returned no error but
// spoke on a stream it must keep silent was reported as "%!w(<nil>)", which
// destroyed the only evidence in the log.
func TestLaneStreamFailureNeverFormatsANilError(t *testing.T) {
	noise := "go: downloading github.com/spf13/cobra v1.10.2"
	silent := laneStreamFailure("native package selection failed", nil, noise)
	if silent == nil {
		t.Fatal("a stream failure with no error must still be an error")
	}
	for _, text := range []string{silent.Error()} {
		if strings.Contains(text, "%!") {
			t.Errorf("nil error was formatted through a verb: %s", text)
		}
		if !strings.Contains(text, noise) || !strings.Contains(text, "native package selection failed") {
			t.Errorf("diagnostic lost its evidence: %s", text)
		}
	}
	wrapped := laneStreamFailure("native package selection failed", os.ErrClosed, noise)
	if !errors.Is(wrapped, os.ErrClosed) {
		t.Error("a real error must stay unwrappable-to")
	}
	if !strings.Contains(wrapped.Error(), noise) {
		t.Errorf("wrapped diagnostic lost its stream: %s", wrapped)
	}
	if bare := laneStreamFailure("native package selection failed", os.ErrClosed); strings.Contains(bare.Error(), "%!") || !errors.Is(bare, os.ErrClosed) {
		t.Errorf("empty stream produced a malformed diagnostic: %v", bare)
	}
}

// TestWarmGoModuleClosureSilencesColdCacheSelection reproduces the first
// hosted-run defect offline: with a cold module cache the selection command
// writes "go: downloading ..." to stderr, and the lane treats any selection
// stderr as failure. The module proxy is the host's own module cache in
// proxy layout, so the reproduction needs no network.
func TestWarmGoModuleClosureSilencesColdCacheSelection(t *testing.T) {
	repo := laneRepo(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	hostCache, err := exec.CommandContext(ctx, "go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Fatalf("resolve host module cache: %v", err)
	}
	proxy := filepath.Join(strings.TrimSpace(string(hostCache)), "cache", "download")
	if info, err := os.Stat(proxy); err != nil || !info.IsDir() {
		t.Skipf("host module cache is not in proxy layout: %v", err)
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	// The Go toolchain writes the module cache read-only, so it cannot live
	// under t.TempDir(): removal is restored explicitly here.
	cold, err := os.MkdirTemp("", "lane-cold-modcache-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = filepath.WalkDir(cold, func(path string, d os.DirEntry, err error) error {
			if err == nil {
				_ = os.Chmod(path, 0o700)
			}
			return nil
		})
		if err := os.RemoveAll(cold); err != nil {
			t.Errorf("cold module cache was not removed: %v", err)
		}
	})
	t.Setenv("GOMODCACHE", cold)
	t.Setenv("GOPROXY", "file://"+filepath.ToSlash(proxy))
	t.Setenv("GONOSUMDB", "*")
	t.Setenv("GONOSUMCHECK", "1")
	t.Setenv("GOFLAGS", "")

	custody, err := newLaneCustody()
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	selection := func(scratch string) (string, string, error) {
		return custody.run(ctx, scratch, repo, environment(nil), 3*time.Minute, 1<<20, 4<<20, goBin, "list", "-json", "-tags", "machinery_integration", "./internal/checker")
	}

	coldScratch := filepath.Join(work, "cold")
	if err := os.MkdirAll(coldScratch, 0o700); err != nil {
		t.Fatal(err)
	}
	_, coldErrout, coldErr := selection(coldScratch)
	if coldErr == nil && coldErrout == "" {
		t.Skip("module cache did not go cold; the reproduction needs an empty GOMODCACHE")
	}
	if !strings.Contains(coldErrout, "go: downloading") {
		t.Fatalf("cold selection did not reproduce the download noise: err=%v stderr=%q", coldErr, coldErrout)
	}
	// This is exactly the shape the lane rejected on the hosted runner.
	if diagnostic := laneStreamFailure("native package selection failed", coldErr, coldErrout); !strings.Contains(diagnostic.Error(), "go: downloading") {
		t.Fatalf("cold-cache diagnostic hid the cause: %v", diagnostic)
	}

	warmScratch := filepath.Join(work, "warm")
	if err := os.MkdirAll(warmScratch, 0o700); err != nil {
		t.Fatal(err)
	}
	receipt, err := warmGoModuleClosure(ctx, custody, repo, warmScratch, goBin)
	if err != nil {
		t.Fatalf("warming the module closure failed: %v", err)
	}
	if receipt.ID != "go-modules" || receipt.Status != "passed" || len(receipt.SHA) != 64 || receipt.Identity != "go.mod+go.sum" {
		t.Fatalf("module closure receipt is not evidence: %+v", receipt)
	}

	afterScratch := filepath.Join(work, "after")
	if err := os.MkdirAll(afterScratch, 0o700); err != nil {
		t.Fatal(err)
	}
	out, errout, err := selection(afterScratch)
	if err != nil || errout != "" {
		t.Fatalf("selection after warming was not silent: err=%v stderr=%q", err, errout)
	}
	if !strings.Contains(out, `"ImportPath"`) {
		t.Fatalf("selection produced no package description: %s", out)
	}
}

// TestWarmGoModuleClosureRejectsUnexpectedStderr proves the warm step checks
// the toolchain's stream vocabulary instead of ignoring stderr wholesale.
func TestWarmGoModuleClosureRejectsUnexpectedStderr(t *testing.T) {
	for _, tc := range []struct {
		name     string
		line     string
		progress bool
	}{
		{"download", "go: downloading github.com/spf13/cobra v1.10.2", true},
		{"extract", "go: extracting github.com/spf13/cobra v1.10.2", true},
		{"finding", "go: finding module for package example.com/x", true},
		{"nothing-to-download", "go: no module dependencies to download", true},
		{"warning", "go: warning: ignoring symlink", false},
		{"error", "go: module lookup disabled by GOPROXY=off", false},
		{"bare-prefix", "go: downloading", false},
	} {
		if got := goDownloadProgress.MatchString(tc.line); got != tc.progress {
			t.Errorf("%s: progress classification = %v, want %v", tc.name, got, tc.progress)
		}
	}
}
