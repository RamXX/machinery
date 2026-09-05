package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
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
			var wf struct {
				Jobs map[string]struct {
					RunsOn   any  `yaml:"runs-on"`
					Continue bool `yaml:"continue-on-error"`
					Steps    []struct {
						Run  string         `yaml:"run"`
						Uses string         `yaml:"uses"`
						If   string         `yaml:"if"`
						With map[string]any `yaml:"with"`
					} `yaml:"steps"`
				} `yaml:"jobs"`
			}
			if e := yaml.Unmarshal(b, &wf); e != nil {
				t.Fatal(e)
			}
			found := false
			for id, j := range wf.Jobs {
				for _, s := range j.Steps {
					if strings.Contains(s.Run, "./scripts/integration-lane") && strings.Contains(s.Run, "--lane required") {
						found = true
						if j.Continue || s.If != "" || !strings.Contains(fmt.Sprint(j.RunsOn), "ubuntu") {
							t.Errorf("%s can omit required Linux lane", id)
						}
					}
					if strings.Contains(fmt.Sprint(j.RunsOn), "macos") && (strings.Contains(s.Run, "machinery_integration") || strings.Contains(s.Run, "--lane required")) {
						t.Errorf("native macOS portability job %s requires services", id)
					}
					if strings.Contains(s.Uses, "actions/checkout") && (strings.Contains(id, "golden") || strings.Contains(id, "integration")) && fmt.Sprint(s.With["fetch-depth"]) != "0" {
						t.Errorf("%s lacks ancestry required by acceptance checks", id)
					}
				}
			}
			if !found {
				t.Error("workflow has no unconditional required integration lane")
			}
		})
	}
	b, e := os.ReadFile(filepath.Join(repo, "scripts/preflight.sh"))
	if e != nil {
		t.Fatal(e)
	}
	s := string(b)
	lane := strings.Index(s, "./scripts/integration-lane")
	race := strings.Index(s, "go test -race -count=1 ./...")
	formal := strings.Index(s, "make verify-formal")
	if lane < 0 || race < 0 || lane < race || formal < lane {
		t.Errorf("preflight must run service-free race, then self-provisioning lane, then consuming formal gates; offsets race=%d lane=%d formal=%d", race, lane, formal)
	}
	if strings.Contains(s, "skipping local gate suite") {
		t.Error("required integration coverage remains bypassable via SKIP_PREFLIGHT")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "make", "-n", "test-integration")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil || !bytes.Contains(out, []byte("./scripts/integration-lane --lane required")) {
		t.Errorf("Make entrypoint differs from shared lane: %v %s", err, out)
	}
}
