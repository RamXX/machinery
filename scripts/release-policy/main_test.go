// Package main tests for the release-policy verifier.
//
// Reject-state enumeration (every code is asserted by the scenario table in
// TestRejectStateTable, and TestRejectStateEnumerationComplete proves the
// table covers exactly this set):
//
//	R01 tag-version          tag does not match the policy tag_pattern
//	R02 tag-unresolved       tag does not resolve to a commit in -root
//	R03 expect-sha           -expect-sha disagrees with the tag commit (stale SHA)
//	R04 main-ancestry        tag commit is not reachable from the main ref
//	R05 policy-invalid       policy missing, unparseable, or schema-invalid
//	R06 run-missing          no required-workflow run for the exact commit
//	R07 run-incomplete       candidate run pending / queued / in_progress
//	R08 run-conclusion       completed but conclusion is not success
//	R09 workflow-identity    run name/path identity does not match the policy
//	R10 job-missing          required job absent from the candidate run
//	R11 job-conclusion       required job skipped / cancelled / failed / in progress
//	R12 event-disallowed     run identity matches but event is not permitted
//	R13 api-unavailable      transport error, non-2xx, or timeout (fail closed)
//	R14 api-malformed        unparseable or inconsistent API response
//	R15 manifest-unreadable  -artifact-manifest file missing or unreadable
//	R16 manifest-header      manifest commit header is not the release commit
//	R17 manifest-malformed   manifest structure invalid (header/line/sort/dup/hex)
//	R18 artifact-identity    manifest name outside inventory or digest mismatch
//	R19 main-ref-unresolved  the main ref itself does not resolve (fail closed)
//	R20 git-unavailable      a Git query failed operationally (fail closed)
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/testgit"
	"gopkg.in/yaml.v3"
)

const denyPrefix = "deny ["

type policyRepo struct {
	root                    string
	base, middle, tip, side string
}

func newPolicyRepo(t *testing.T) policyRepo {
	t.Helper()
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		if out, err := testgit.Run(t.Context(), root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	rev := func(revspec string) string {
		t.Helper()
		out, err := testgit.Run(t.Context(), root, "rev-parse", revspec)
		if err != nil {
			t.Fatalf("git rev-parse %s: %v\n%s", revspec, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "-b", "main")
	git("config", "user.name", "Fixture")
	git("config", "user.email", "fixture@example.invalid")
	for _, n := range []string{"base", "middle", "tip"} {
		if err := os.WriteFile(filepath.Join(root, n+".txt"), []byte(n), 0o644); err != nil {
			t.Fatal(err)
		}
		git("add", n+".txt")
		git("commit", "-qm", "commit "+n)
	}
	git("tag", "v0.1.0")
	git("tag", "v0.0.1", "main~2")
	git("branch", "side", "main~1")
	git("checkout", "-q", "side")
	if err := os.WriteFile(filepath.Join(root, "side.txt"), []byte("side"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "side.txt")
	git("commit", "-qm", "commit side")
	git("tag", "v0.2.0")
	git("checkout", "-q", "main")
	return policyRepo{root: root, base: rev("main~2"), middle: rev("main~1"), tip: rev("main"), side: rev("side")}
}

func packageDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source file")
	}
	return filepath.Dir(file)
}

func repoRootDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join(packageDir(t), "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(abs, ".git")); err != nil {
		t.Fatalf("test repository root not found at %s", abs)
	}
	return abs
}

func fixtureBytes(t *testing.T, rel ...string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(append([]string{packageDir(t), "testdata"}, rel...)...))
	if err != nil {
		t.Fatalf("read fixture %s: %v", filepath.Join(rel...), err)
	}
	return raw
}

func runsFixture(t *testing.T, name string) []byte {
	t.Helper()
	return fixtureBytes(t, "api", name)
}

func mutateRuns(t *testing.T, base string, f func(runs []map[string]any)) []byte {
	t.Helper()
	var doc struct {
		TotalCount   int              `json:"total_count"`
		WorkflowRuns []map[string]any `json:"workflow_runs"`
	}
	if err := json.Unmarshal(runsFixture(t, base), &doc); err != nil {
		t.Fatal(err)
	}
	f(doc.WorkflowRuns)
	out, err := json.Marshal(&doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func setRun(t *testing.T, runs []map[string]any, name string, fields map[string]any) {
	t.Helper()
	for _, run := range runs {
		if run["name"] == name {
			for k, v := range fields {
				run[k] = v
			}
			return
		}
	}
	t.Fatalf("fixture run %q not found", name)
}

func mutateJobs(t *testing.T, id string, f func(jobs []map[string]any)) []byte {
	t.Helper()
	var doc struct {
		TotalCount int              `json:"total_count"`
		Jobs       []map[string]any `json:"jobs"`
	}
	if err := json.Unmarshal(fixtureBytes(t, "api", "jobs-"+id+".json"), &doc); err != nil {
		t.Fatal(err)
	}
	f(doc.Jobs)
	out, err := json.Marshal(&doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func setJob(t *testing.T, jobs []map[string]any, name string, fields map[string]any) {
	t.Helper()
	for _, job := range jobs {
		if job["name"] == name {
			for k, v := range fields {
				job[k] = v
			}
			return
		}
	}
	t.Fatalf("fixture job %q not found", name)
}

func serveAPI(t *testing.T, runs func() []byte, jobs func(id string) []byte) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/example/machinery/actions/runs":
			body := bytes.ReplaceAll(runs(), []byte("__COMMIT__"), []byte(r.URL.Query().Get("head_sha")))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/repos/example/machinery/actions/runs/") && strings.HasSuffix(r.URL.Path, "/jobs"):
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/repos/example/machinery/actions/runs/"), "/jobs")
			var body []byte
			if jobs != nil {
				body = jobs(id)
			}
			if body == nil {
				if raw, err := os.ReadFile(filepath.Join(packageDir(t), "testdata", "api", "jobs-"+id+".json")); err == nil {
					body = raw
				}
			}
			if body == nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func statusServer(t *testing.T, code int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func slowServer(t *testing.T, delay time.Duration) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func closedServerURL(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	srv.Close()
	return srv.URL
}

func testPolicyPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(packageDir(t), "testdata", "policy.json")
}

func mutatedPolicy(t *testing.T, f func(p map[string]any)) string {
	t.Helper()
	var p map[string]any
	if err := json.Unmarshal(fixtureBytes(t, "policy.json"), &p); err != nil {
		t.Fatal(err)
	}
	f(p)
	out, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func inventoryNames() []string {
	return []string{
		"machinery-darwin-amd64", "machinery-darwin-arm64", "machinery-linux-amd64", "machinery-linux-arm64",
		"machinery_0.1.0_darwin_amd64.tar.gz", "machinery_0.1.0_darwin_arm64.tar.gz",
		"machinery_0.1.0_linux_amd64.tar.gz", "machinery_0.1.0_linux_arm64.tar.gz",
	}
}

func writeDist(t *testing.T, dir string, skip string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range inventoryNames() {
		if name == skip {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func nameDigest(name string) string {
	sum := sha256.Sum256([]byte(name))
	return hex.EncodeToString(sum[:])
}

func writeManifest(t *testing.T, commit string, names []string, digests map[string]string) string {
	t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, "commit %s\n", commit)
	for _, name := range names {
		digest := nameDigest(name)
		if override, ok := digests[name]; ok {
			digest = override
		}
		fmt.Fprintf(&b, "%s  %s\n", digest, name)
	}
	path := filepath.Join(t.TempDir(), "dist-manifest.txt")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func runVerifier(t *testing.T, args, environ []string) (int, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	status := run(args, environ, &stdout, &stderr)
	return status, stdout.String() + stderr.String()
}

func TestAllowExactCommitGreenControl(t *testing.T) {
	repo := newPolicyRepo(t)
	base := serveAPI(t, func() []byte { return runsFixture(t, "runs-green.json") }, nil)
	status, combined := runVerifier(t, []string{
		"-root", repo.root, "-tag", "v0.1.0", "-policy", testPolicyPath(t),
		"-api-base", base, "-repository", "example/machinery", "-expect-sha", repo.tip,
	}, os.Environ())
	if status != 0 || !strings.Contains(combined, "verdict ALLOW") {
		t.Fatalf("green exact-commit verification denied: status=%d\n%s", status, combined)
	}
	for _, want := range []string{"ci.yml", "formal.yml", "security.yml", "integration-required"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("allow summary omitted %s:\n%s", want, combined)
		}
	}
}

func TestAllowTagBehindMainTipControl(t *testing.T) {
	repo := newPolicyRepo(t)
	base := serveAPI(t, func() []byte { return runsFixture(t, "runs-green.json") }, nil)
	status, combined := runVerifier(t, []string{
		"-root", repo.root, "-tag", "v0.0.1", "-policy", testPolicyPath(t),
		"-api-base", base, "-repository", "example/machinery",
	}, os.Environ())
	if status != 0 || !strings.Contains(combined, "verdict ALLOW") {
		t.Fatalf("tag behind main tip must be permitted ancestry: status=%d\n%s", status, combined)
	}
}

type rejectScenario struct {
	name       string
	code       string
	detail     string
	tag        string
	serverMode string
	runs       func(t *testing.T) []byte
	jobs       func(id string) []byte
	args       func(t *testing.T, repo policyRepo, base, policy string) []string
	extra      func(t *testing.T, repo policyRepo) []string
	environ    func() []string
}

func rejectScenarios(t *testing.T) []rejectScenario {
	t.Helper()
	return []rejectScenario{
		{name: "tag omits leading v", code: "R01", detail: "tag", tag: "0.1.0"},
		{name: "tag carries prerelease suffix", code: "R01", tag: "v0.1.0-rc1"},
		{name: "tag absent from repository", code: "R02", detail: "unresolved", tag: "v3.2.1"},
		{name: "expect-sha is stale", code: "R03", detail: "expect-sha", extra: func(t *testing.T, repo policyRepo) []string {
			return []string{"-expect-sha", repo.middle}
		}},
		{name: "tag off unmerged side branch", code: "R04", detail: "ancestor", tag: "v0.2.0"},
		{name: "policy file missing", code: "R05", detail: "policy", args: func(t *testing.T, repo policyRepo, base, _ string) []string {
			return []string{"-root", repo.root, "-tag", "v0.1.0", "-policy", filepath.Join(t.TempDir(), "absent-policy.json"), "-api-base", base, "-repository", "example/machinery"}
		}},
		{name: "policy malformed json", code: "R05", args: func(t *testing.T, repo policyRepo, base, _ string) []string {
			path := filepath.Join(t.TempDir(), "policy.json")
			if err := os.WriteFile(path, []byte(`{"policy_version":`), 0o644); err != nil {
				t.Fatal(err)
			}
			return []string{"-root", repo.root, "-tag", "v0.1.0", "-policy", path, "-api-base", base, "-repository", "example/machinery"}
		}},
		{name: "policy tag pattern not a regexp", code: "R05", args: func(t *testing.T, repo policyRepo, base, _ string) []string {
			return []string{"-root", repo.root, "-tag", "v0.1.0", "-policy", mutatedPolicy(t, func(p map[string]any) { p["tag_pattern"] = "^v(" }), "-api-base", base, "-repository", "example/machinery"}
		}},
		{name: "policy required jobs empty", code: "R05", args: func(t *testing.T, repo policyRepo, base, _ string) []string {
			return []string{"-root", repo.root, "-tag", "v0.1.0", "-policy", mutatedPolicy(t, func(p map[string]any) {
				p["required_workflows"].([]any)[0].(map[string]any)["required_jobs"] = []string{}
			}), "-api-base", base, "-repository", "example/machinery"}
		}},
		{name: "policy contexts empty", code: "R05", args: func(t *testing.T, repo policyRepo, base, _ string) []string {
			return []string{"-root", repo.root, "-tag", "v0.1.0", "-policy", mutatedPolicy(t, func(p map[string]any) {
				p["github_required_status_checks"].(map[string]any)["payload"].(map[string]any)["contexts"] = []string{}
			}), "-api-base", base, "-repository", "example/machinery"}
		}},
		{name: "formal run missing for exact commit", code: "R06", detail: "formal.yml", runs: func(t *testing.T) []byte {
			return runsFixture(t, "runs-missing-formal.json")
		}},
		{name: "runs only for a stale sha", code: "R06", detail: "exact commit", runs: func(t *testing.T) []byte {
			return runsFixture(t, "runs-stale-sha.json")
		}},
		{name: "run pending", code: "R07", detail: "in_progress", runs: func(t *testing.T) []byte {
			return runsFixture(t, "runs-pending.json")
		}},
		{name: "run queued", code: "R07", detail: "queued", runs: func(t *testing.T) []byte {
			return runsFixture(t, "runs-queued.json")
		}},
		{name: "run conclusion failure", code: "R08", detail: "failure", runs: func(t *testing.T) []byte {
			return mutateRuns(t, "runs-green.json", func(runs []map[string]any) {
				setRun(t, runs, "ci", map[string]any{"conclusion": "failure"})
			})
		}},
		{name: "run conclusion cancelled", code: "R08", detail: "cancelled", runs: func(t *testing.T) []byte {
			return mutateRuns(t, "runs-green.json", func(runs []map[string]any) {
				setRun(t, runs, "ci", map[string]any{"conclusion": "cancelled"})
			})
		}},
		{name: "run conclusion timed_out", code: "R08", detail: "timed_out", runs: func(t *testing.T) []byte {
			return mutateRuns(t, "runs-green.json", func(runs []map[string]any) {
				setRun(t, runs, "ci", map[string]any{"conclusion": "timed_out"})
			})
		}},
		{name: "run conclusion startup_failure", code: "R08", runs: func(t *testing.T) []byte {
			return mutateRuns(t, "runs-green.json", func(runs []map[string]any) {
				setRun(t, runs, "ci", map[string]any{"conclusion": "startup_failure"})
			})
		}},
		{name: "run conclusion action_required", code: "R08", runs: func(t *testing.T) []byte {
			return mutateRuns(t, "runs-green.json", func(runs []map[string]any) {
				setRun(t, runs, "ci", map[string]any{"conclusion": "action_required"})
			})
		}},
		{name: "run conclusion neutral", code: "R08", detail: "neutral", runs: func(t *testing.T) []byte {
			return mutateRuns(t, "runs-green.json", func(runs []map[string]any) {
				setRun(t, runs, "ci", map[string]any{"conclusion": "neutral"})
			})
		}},
		{name: "workflow path identity mismatch", code: "R09", detail: "identity", runs: func(t *testing.T) []byte {
			return runsFixture(t, "runs-wrong-path.json")
		}},
		{name: "workflow name identity mismatch", code: "R09", detail: "identity", runs: func(t *testing.T) []byte {
			return runsFixture(t, "runs-wrong-name.json")
		}},
		{name: "required integration lane job missing", code: "R10", detail: "integration-required", jobs: func(id string) []byte {
			if id != "101" {
				return nil
			}
			return fixtureBytes(t, "api", "jobs-101-missing-lane.json")
		}},
		{name: "required job skipped", code: "R11", detail: "skipped", jobs: func(id string) []byte {
			if id != "101" {
				return nil
			}
			return fixtureBytes(t, "api", "jobs-101-skipped-lane.json")
		}},
		{name: "required job cancelled", code: "R11", detail: "cancelled", jobs: func(id string) []byte {
			if id != "101" {
				return nil
			}
			return mutateJobs(t, "101", func(jobs []map[string]any) {
				setJob(t, jobs, "integration-required", map[string]any{"conclusion": "cancelled"})
			})
		}},
		{name: "required job failed", code: "R11", detail: "failure", jobs: func(id string) []byte {
			if id != "101" {
				return nil
			}
			return mutateJobs(t, "101", func(jobs []map[string]any) {
				setJob(t, jobs, "integration-required", map[string]any{"conclusion": "failure"})
			})
		}},
		{name: "required job still running", code: "R11", detail: "in_progress", jobs: func(id string) []byte {
			if id != "101" {
				return nil
			}
			return mutateJobs(t, "101", func(jobs []map[string]any) {
				setJob(t, jobs, "integration-required", map[string]any{"status": "in_progress", "conclusion": nil})
			})
		}},
		{name: "disallowed event", code: "R12", detail: "event", runs: func(t *testing.T) []byte {
			return runsFixture(t, "runs-wrong-event.json")
		}},
		{name: "api http 500 fails closed", code: "R13", detail: "fail closed", serverMode: "500"},
		{name: "api connection refused fails closed", code: "R13", detail: "fail closed", serverMode: "closed"},
		{name: "api timeout fails closed", code: "R13", detail: "fail closed", serverMode: "slow", extra: func(*testing.T, policyRepo) []string {
			return []string{"-timeout", "200ms"}
		}},
		{name: "api response not json", code: "R14", detail: "malformed", runs: func(t *testing.T) []byte {
			return runsFixture(t, "runs-malformed.json")
		}},
		{name: "api run missing name field", code: "R14", detail: "malformed", runs: func(t *testing.T) []byte {
			return runsFixture(t, "runs-empty-field.json")
		}},
		{name: "api run head_sha malformed", code: "R14", detail: "malformed", runs: func(t *testing.T) []byte {
			return runsFixture(t, "runs-bad-sha.json")
		}},
		{name: "api page truncated", code: "R14", detail: "page", runs: func(t *testing.T) []byte {
			return runsFixture(t, "runs-truncated.json")
		}},
		{name: "artifact manifest missing", code: "R15", detail: "manifest", args: func(t *testing.T, repo policyRepo, _, policy string) []string {
			return []string{"-root", repo.root, "-tag", "v0.1.0", "-policy", policy, "-artifact-manifest", filepath.Join(t.TempDir(), "absent-manifest.txt")}
		}},
		{name: "artifact manifest header names another commit", code: "R16", detail: "header", args: func(t *testing.T, repo policyRepo, _, policy string) []string {
			return []string{"-root", repo.root, "-tag", "v0.1.0", "-policy", policy, "-artifact-manifest", writeManifest(t, repo.middle, inventoryNames(), nil)}
		}},
		{name: "artifact manifest unsorted", code: "R17", args: func(t *testing.T, repo policyRepo, _, policy string) []string {
			names := inventoryNames()
			names[0], names[1] = names[1], names[0]
			return []string{"-root", repo.root, "-tag", "v0.1.0", "-policy", policy, "-artifact-manifest", writeManifest(t, repo.tip, names, nil)}
		}},
		{name: "artifact manifest duplicate name", code: "R17", args: func(t *testing.T, repo policyRepo, _, policy string) []string {
			names := append(append([]string{}, inventoryNames()...), inventoryNames()[0])
			sort.Strings(names)
			return []string{"-root", repo.root, "-tag", "v0.1.0", "-policy", policy, "-artifact-manifest", writeManifest(t, repo.tip, names, nil)}
		}},
		{name: "artifact manifest digest not hex", code: "R17", args: func(t *testing.T, repo policyRepo, _, policy string) []string {
			return []string{"-root", repo.root, "-tag", "v0.1.0", "-policy", policy, "-artifact-manifest", writeManifest(t, repo.tip, inventoryNames(), map[string]string{"machinery-linux-amd64": "zz" + strings.Repeat("0", 62)})}
		}},
		{name: "artifact manifest bad header line", code: "R17", args: func(t *testing.T, repo policyRepo, _, policy string) []string {
			path := filepath.Join(t.TempDir(), "dist-manifest.txt")
			if err := os.WriteFile(path, []byte("commit not-a-sha\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			return []string{"-root", repo.root, "-tag", "v0.1.0", "-policy", policy, "-artifact-manifest", path}
		}},
		{name: "artifact name outside inventory", code: "R18", detail: "artifact", args: func(t *testing.T, repo policyRepo, _, policy string) []string {
			names := inventoryNames()
			names[0] = "machinery-rogue-binary"
			sort.Strings(names)
			return []string{"-root", repo.root, "-tag", "v0.1.0", "-policy", policy, "-artifact-manifest", writeManifest(t, repo.tip, names, nil)}
		}},
		{name: "artifact content digest mismatch", code: "R18", args: func(t *testing.T, repo policyRepo, _, policy string) []string {
			dir := t.TempDir()
			writeDist(t, dir, "")
			return []string{"-root", repo.root, "-tag", "v0.1.0", "-policy", policy, "-artifact-manifest", writeManifest(t, repo.tip, inventoryNames(), map[string]string{"machinery-linux-amd64": strings.Repeat("a", 64)}), "-artifacts-dir", dir}
		}},
		{name: "artifact file missing", code: "R18", args: func(t *testing.T, repo policyRepo, _, policy string) []string {
			dir := t.TempDir()
			writeDist(t, dir, "machinery-linux-amd64")
			return []string{"-root", repo.root, "-tag", "v0.1.0", "-policy", policy, "-artifact-manifest", writeManifest(t, repo.tip, inventoryNames(), nil), "-artifacts-dir", dir}
		}},
		{name: "main ref unresolved", code: "R19", detail: "main ref", extra: func(*testing.T, policyRepo) []string {
			return []string{"-main-ref", "refs/remotes/origin/absent"}
		}},
		{name: "git unavailable", code: "R20", environ: func() []string {
			return []string{"PATH=/machinery-release-policy-test-missing"}
		}},
	}
}

func TestRejectStateTable(t *testing.T) {
	scenarios := rejectScenarios(t)
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			repo := newPolicyRepo(t)
			policy := testPolicyPath(t)
			var base string
			switch sc.serverMode {
			case "500":
				base = statusServer(t, 500)
			case "slow":
				base = slowServer(t, 3*time.Second)
			case "closed":
				base = closedServerURL(t)
			default:
				runs := sc.runs
				if runs == nil {
					runs = func(t *testing.T) []byte { return runsFixture(t, "runs-green.json") }
				}
				base = serveAPI(t, func() []byte { return runs(t) }, sc.jobs)
			}
			tag := sc.tag
			if tag == "" {
				tag = "v0.1.0"
			}
			args := []string{"-root", repo.root, "-tag", tag, "-policy", policy, "-api-base", base, "-repository", "example/machinery"}
			if sc.extra != nil {
				args = append(args, sc.extra(t, repo)...)
			}
			if sc.args != nil {
				args = sc.args(t, repo, base, policy)
			}
			environ := os.Environ()
			if sc.environ != nil {
				environ = sc.environ()
			}
			status, combined := runVerifier(t, args, environ)
			if status != 1 {
				t.Fatalf("scenario must deny with exit 1, got %d\n%s", status, combined)
			}
			if !strings.Contains(combined, denyPrefix+sc.code+"]") {
				t.Fatalf("scenario must emit %s%s]\n%s", denyPrefix, sc.code, combined)
			}
			if sc.detail != "" && !strings.Contains(combined, sc.detail) {
				t.Fatalf("deny detail %q missing\n%s", sc.detail, combined)
			}
			if !strings.Contains(combined, "verdict DENY") {
				t.Fatalf("verdict line missing\n%s", combined)
			}
		})
	}
}

func TestRejectStateEnumerationComplete(t *testing.T) {
	got := map[string]bool{}
	for _, sc := range rejectScenarios(t) {
		got[sc.code] = true
	}
	for i := 1; i <= 20; i++ {
		code := fmt.Sprintf("R%02d", i)
		if !got[code] {
			t.Fatalf("reject-state enumeration incomplete: %s has no scenario", code)
		}
	}
	for code := range got {
		if len(code) != 3 || code[0] != 'R' {
			t.Fatalf("unexpected reject code %q", code)
		}
		n := int(code[1]-'0')*10 + int(code[2]-'0')
		if n < 1 || n > 20 {
			t.Fatalf("reject code %q outside the enumerated set", code)
		}
	}
}

func TestShallowCloneAncestryDeniedThenRepaired(t *testing.T) {
	origin := newPolicyRepo(t)
	clone := filepath.Join(t.TempDir(), "clone")
	if out, err := testgit.Run(t.Context(), "", "clone", "-q", "--depth", "1", "-b", "main", "file://"+origin.root, clone); err != nil {
		t.Fatalf("shallow clone: %v\n%s", err, out)
	}
	if out, err := testgit.Run(t.Context(), clone, "fetch", "-q", "--depth", "1", "origin", "refs/tags/v0.0.1:refs/tags/v0.0.1"); err != nil {
		t.Fatalf("fetch tag into shallow clone: %v\n%s", err, out)
	}
	base := serveAPI(t, func() []byte { return runsFixture(t, "runs-green.json") }, nil)
	args := []string{"-root", clone, "-tag", "v0.0.1", "-policy", testPolicyPath(t), "-main-ref", "refs/remotes/origin/main", "-api-base", base, "-repository", "example/machinery"}
	status, combined := runVerifier(t, args, os.Environ())
	if status != 1 || !strings.Contains(combined, denyPrefix+"R04]") {
		t.Fatalf("shallow clone must fail closed on ancestry: status=%d\n%s", status, combined)
	}
	if out, err := testgit.Run(t.Context(), clone, "fetch", "-q", "--unshallow", "origin"); err != nil {
		t.Fatalf("unshallow: %v\n%s", err, out)
	}
	status, combined = runVerifier(t, args, os.Environ())
	if status != 0 || !strings.Contains(combined, "verdict ALLOW") {
		t.Fatalf("repaired full-history clone must verify: status=%d\n%s", status, combined)
	}
}

func TestProcessInvocation(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "release-policy")
	buildCmd := exec.CommandContext(t.Context(), "go", "build", "-o", bin, ".")
	buildCmd.Dir = packageDir(t)
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build verifier binary: %v\n%s", err, out)
	}
	repo := newPolicyRepo(t)
	green := serveAPI(t, func() []byte { return runsFixture(t, "runs-green.json") }, nil)
	failed := serveAPI(t, func() []byte {
		return mutateRuns(t, "runs-green.json", func(runs []map[string]any) {
			setRun(t, runs, "ci", map[string]any{"conclusion": "failure"})
		})
	}, nil)
	invoke := func(args ...string) (int, string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), bin, args...)
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		status := 0
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			status = exitErr.ExitCode()
		} else if err != nil {
			t.Fatalf("invoke verifier: %v\n%s", err, out)
		}
		return status, string(out)
	}
	status, combined := invoke("-root", repo.root, "-tag", "v0.1.0", "-policy", testPolicyPath(t), "-api-base", green, "-repository", "example/machinery")
	if status != 0 || !strings.Contains(combined, "verdict ALLOW") {
		t.Fatalf("process invocation must allow green: status=%d\n%s", status, combined)
	}
	status, combined = invoke("-root", repo.root, "-tag", "v0.1.0", "-policy", testPolicyPath(t), "-api-base", failed, "-repository", "example/machinery")
	if status != 1 || !strings.Contains(combined, denyPrefix+"R08]") {
		t.Fatalf("process invocation must deny failed conclusion: status=%d\n%s", status, combined)
	}
	dir := t.TempDir()
	writeDist(t, dir, "")
	status, combined = invoke("-root", repo.root, "-tag", "v0.1.0", "-policy", testPolicyPath(t), "-artifact-manifest", writeManifest(t, repo.tip, inventoryNames(), nil), "-artifacts-dir", dir)
	if status != 0 || !strings.Contains(combined, "verdict ALLOW") {
		t.Fatalf("process invocation must verify local artifact identity: status=%d\n%s", status, combined)
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"-tag", "v0.1.0"},
		{"-root", t.TempDir()},
		{"-frob"},
	} {
		if status, combined := runVerifier(t, args, os.Environ()); status != 2 {
			t.Fatalf("usage error must exit 2, got %d for %v\n%s", status, args, combined)
		}
	}
}

type workflowStep struct {
	Name string         `yaml:"name"`
	Uses string         `yaml:"uses"`
	Run  string         `yaml:"run"`
	With map[string]any `yaml:"with"`
}

type workflowJob struct {
	Needs          any            `yaml:"needs"`
	Permissions    any            `yaml:"permissions"`
	TimeoutMinutes any            `yaml:"timeout-minutes"`
	Steps          []workflowStep `yaml:"steps"`
}

type workflowFile struct {
	Permissions any `yaml:"permissions"`
	Concurrency *struct {
		Group string `yaml:"group"`
	} `yaml:"concurrency"`
	Jobs map[string]workflowJob `yaml:"jobs"`
}

func loadWorkflow(t *testing.T, rel string) workflowFile {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRootDir(t), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read workflow %s: %v", rel, err)
	}
	var wf workflowFile
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("parse workflow %s: %v", rel, err)
	}
	return wf
}

func permMap(t *testing.T, v any) map[string]string {
	t.Helper()
	out := map[string]string{}
	perms, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("permissions is not a mapping: %#v", v)
	}
	for k, val := range perms {
		out[k] = fmt.Sprint(val)
	}
	return out
}

func needsList(t *testing.T, v any) []string {
	t.Helper()
	switch needs := v.(type) {
	case string:
		return []string{needs}
	case []any:
		out := make([]string, 0, len(needs))
		for _, item := range needs {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		t.Fatalf("needs is not a string or list: %#v", v)
		return nil
	}
}

func checkoutFetchDepth(t *testing.T, job workflowJob) string {
	t.Helper()
	for _, step := range job.Steps {
		if strings.HasPrefix(step.Uses, "actions/checkout@") {
			if step.With == nil {
				return ""
			}
			return fmt.Sprint(step.With["fetch-depth"])
		}
	}
	return ""
}

func TestWorkflowHardeningContracts(t *testing.T) {
	release := loadWorkflow(t, ".github/workflows/release.yml")
	if got := permMap(t, release.Permissions); got["contents"] != "read" || len(got) != 1 {
		t.Fatalf("release.yml must default to contents: read only, got %v", got)
	}
	publish := release.Jobs["publish"]
	if publish.Needs == nil {
		t.Fatal("release.yml publish job must declare needs")
	}
	needs := needsList(t, publish.Needs)
	for _, want := range []string{"build", "gate", "manifest"} {
		found := false
		for _, n := range needs {
			if n == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("release.yml publish needs must include %s, got %v", want, needs)
		}
	}
	wantPublish := map[string]string{"contents": "write", "id-token": "write", "attestations": "write"}
	if got := permMap(t, publish.Permissions); fmt.Sprint(got) != fmt.Sprint(wantPublish) {
		t.Fatalf("publish permissions must be exactly %v, got %v", wantPublish, got)
	}
	gate := release.Jobs["gate"]
	if got := permMap(t, gate.Permissions); got["contents"] != "read" || got["actions"] != "read" || len(got) != 2 {
		t.Fatalf("gate permissions must be contents+actions read, got %v", got)
	}
	if gate.TimeoutMinutes == nil {
		t.Fatal("gate job must declare timeout-minutes")
	}
	if depth := checkoutFetchDepth(t, gate); depth != "0" {
		t.Fatalf("gate checkout must fetch full history for ancestry, got fetch-depth %q", depth)
	}
	if depth := checkoutFetchDepth(t, publish); depth != "0" {
		t.Fatalf("publish checkout must fetch full history for ancestry, got fetch-depth %q", depth)
	}
	gateRunsPolicy, publishRunsPolicy := false, false
	for _, step := range gate.Steps {
		if strings.Contains(step.Run, "scripts/release-policy") && strings.Contains(step.Run, "-api-base") {
			gateRunsPolicy = true
		}
	}
	for _, step := range publish.Steps {
		if strings.Contains(step.Run, "scripts/release-policy") && strings.Contains(step.Run, "-artifact-manifest") {
			publishRunsPolicy = true
		}
	}
	if !gateRunsPolicy {
		t.Fatal("gate job must run scripts/release-policy with -api-base before publication")
	}
	if !publishRunsPolicy {
		t.Fatal("publish job must run scripts/release-policy with -artifact-manifest")
	}
	manifestJob, ok := release.Jobs["manifest"]
	if !ok {
		t.Fatal("release.yml must carry a manifest job binding artifact identity")
	}
	uploaded, bound := false, false
	for _, step := range manifestJob.Steps {
		if strings.HasPrefix(step.Uses, "actions/upload-artifact@") && step.With != nil && fmt.Sprint(step.With["name"]) == "build-manifest" {
			uploaded = true
		}
		if strings.Contains(step.Run, "dist-manifest.txt") {
			bound = true
		}
	}
	if !uploaded || !bound {
		t.Fatal("manifest job must write and upload the build-manifest artifact")
	}
	for jobName, job := range release.Jobs {
		for _, step := range job.Steps {
			if strings.Contains(step.Run, "${{") {
				t.Fatalf("release.yml %s interpolates expressions into a run script; untrusted event strings must reach scripts via env indirection only", jobName)
			}
		}
	}
	if release.Concurrency == nil || release.Concurrency.Group == "" {
		t.Fatal("release.yml must retain its publication concurrency group")
	}

	nightly := loadWorkflow(t, ".github/workflows/nightly.yml")
	for _, jobName := range []string{"golden-nightly", "integration-required"} {
		job, ok := nightly.Jobs[jobName]
		if !ok || job.Steps == nil {
			t.Fatalf("nightly.yml must retain the %s job", jobName)
		}
		if depth := checkoutFetchDepth(t, job); depth != "0" {
			t.Fatalf("nightly %s performs acceptance ancestry checks and must fetch full history, got fetch-depth %q", jobName, depth)
		}
	}

	ci := loadWorkflow(t, ".github/workflows/ci.yml")
	if got := permMap(t, ci.Permissions); got["contents"] != "read" || len(got) != 1 {
		t.Fatalf("ci.yml must declare least privileges contents: read, got %v", got)
	}
	if _, ok := ci.Jobs["integration-required"]; !ok {
		t.Fatal("ci.yml must retain the required integration-lane job")
	}
	formal := loadWorkflow(t, ".github/workflows/formal.yml")
	if got := permMap(t, formal.Permissions); got["contents"] != "read" || len(got) != 1 {
		t.Fatalf("formal.yml must declare least privileges contents: read, got %v", got)
	}
	for _, job := range []string{"integration-required", "verify-formal"} {
		if _, ok := formal.Jobs[job]; !ok {
			t.Fatalf("formal.yml must retain the %s job", job)
		}
	}
	security := loadWorkflow(t, ".github/workflows/security.yml")
	if got := permMap(t, security.Permissions); got["contents"] != "read" || len(got) != 1 {
		t.Fatalf("security.yml must declare least privileges contents: read, got %v", got)
	}
	for _, job := range []string{"govulncheck", "gitleaks"} {
		if _, ok := security.Jobs[job]; !ok {
			t.Fatalf("security.yml must retain the %s job", job)
		}
	}
}

func TestCommittedPolicyPayloadMatchesWorkflows(t *testing.T) {
	root := repoRootDir(t)
	path := filepath.Join(root, "docs", "release-policy.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("committed policy payload missing: %v", err)
	}
	policy, err := loadPolicy(path)
	if err != nil {
		t.Fatalf("committed policy payload must load: %v", err)
	}
	if policy.Target.Repository != "RamXX/machinery" || policy.Target.Branch != "main" {
		t.Fatalf("policy target must be RamXX/machinery main, got %s %s", policy.Target.Repository, policy.Target.Branch)
	}
	requiredJobs := map[string]bool{}
	for _, wf := range policy.RequiredWorkflows {
		body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(wf.Path)))
		if err != nil {
			t.Fatalf("policy requires unknown workflow %s: %v", wf.Path, err)
		}
		var parsed workflowFile
		if err := yaml.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("parse %s: %v", wf.Path, err)
		}
		for _, job := range wf.RequiredJobs {
			if _, ok := parsed.Jobs[job]; !ok {
				t.Fatalf("policy requires job %q absent from %s", job, wf.Path)
			}
			requiredJobs[job] = true
		}
	}
	for _, wf := range policy.RequiredWorkflows {
		if wf.Path != ".github/workflows/ci.yml" && wf.Path != ".github/workflows/formal.yml" {
			continue
		}
		found := false
		for _, job := range wf.RequiredJobs {
			if job == "integration-required" {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s must require the integration-lane job by its exact identity", wf.Path)
		}
	}
	releaseBytes, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	release := string(releaseBytes)
	if !strings.Contains(release, policy.TagPattern) {
		t.Fatalf("release.yml must enforce the same numeric tag pattern as the policy: %s", policy.TagPattern)
	}
	// The build matrix is the producer: every (goos, goarch) leg emits exactly
	// the bare machinery-<goos>-<goarch> binary and the versioned
	// machinery_{version}_<goos>_<goarch>.tar.gz tarball, and the manifest job
	// hashes precisely those outputs (windows/amd64 included since MAC-r0hy).
	// R18 enforces set equality between this inventory and dist-manifest.txt at
	// publish time, so the committed policy must carry exactly the closed set
	// of matrix outputs — nothing more, nothing less.
	legRe := regexp.MustCompile(`(?m)^ +- goos: (\S+)\n +goarch: (\S+)$`)
	legs := legRe.FindAllStringSubmatch(release, -1)
	if len(legs) == 0 {
		t.Fatal("release.yml publishes no build matrix legs")
	}
	expectedInventory := map[string]bool{}
	for _, leg := range legs {
		expectedInventory["machinery-"+leg[1]+"-"+leg[2]] = true
		expectedInventory[fmt.Sprintf("machinery_{version}_%s_%s.tar.gz", leg[1], leg[2])] = true
	}
	if len(policy.ArtifactInventory.Names) != len(expectedInventory) {
		t.Fatalf("artifact inventory must cover exactly the %d build matrix outputs (%d binary/tarball pairs), got %d names",
			len(expectedInventory), len(legs), len(policy.ArtifactInventory.Names))
	}
	versioned := 0
	for _, name := range policy.ArtifactInventory.Names {
		if !expectedInventory[name] {
			t.Fatalf("policy artifact %q is not a release build matrix output", name)
		}
		if strings.Contains(name, "{version}") {
			versioned++
		}
	}
	if versioned != len(legs) {
		t.Fatalf("artifact inventory must carry one versioned tarball per matrix leg, got %d of %d legs", versioned, len(legs))
	}
	if !strings.Contains(release, fmt.Sprintf("if [ \"$count\" -ne %d ]", len(legs)*2)) {
		t.Fatalf("release.yml manifest job must expect exactly %d build artifacts", len(legs)*2)
	}
	// The publish-time dist/ compare must expect the same closed matrix set
	// plus exactly the two publish-stage additions: the downloaded
	// dist-manifest.txt and the reproducible machinery-source.tar.gz. Every
	// bare matrix binary — darwin included — is a published release asset
	// (install.sh, checksums-sha256.txt, SLSA subject), so an undercount
	// fails a real tag run and an overcount would mask substitution.
	inventoryStart := strings.Index(release, "Require the exact release artifact inventory")
	if inventoryStart < 0 {
		t.Fatal("release.yml has no publish inventory step")
	}
	inventoryEnd := strings.Index(release[inventoryStart:], "Generate checksums")
	if inventoryEnd < 0 {
		t.Fatal("release.yml publish inventory step is not followed by checksum generation")
	}
	inventoryStep := release[inventoryStart : inventoryStart+inventoryEnd]
	nameRe := regexp.MustCompile(`machinery(?:-[A-Za-z0-9.-]+|_\$\{plain\}_[A-Za-z0-9._]+)|dist-manifest\.txt`)
	listed := map[string]bool{}
	for _, name := range nameRe.FindAllString(inventoryStep, -1) {
		if listed[name] {
			t.Fatalf("publish inventory lists %q twice", name)
		}
		listed[name] = true
	}
	expectedDist := map[string]bool{"dist-manifest.txt": true, "machinery-source.tar.gz": true}
	for _, leg := range legs {
		expectedDist["machinery-"+leg[1]+"-"+leg[2]] = true
		expectedDist[fmt.Sprintf("machinery_${plain}_%s_%s.tar.gz", leg[1], leg[2])] = true
	}
	if len(listed) != len(expectedDist) {
		t.Fatalf("publish inventory must expect exactly %d dist files, found %d: %v", len(expectedDist), len(listed), listed)
	}
	for name := range expectedDist {
		if !listed[name] {
			t.Fatalf("publish inventory is missing %q", name)
		}
	}
	for name := range listed {
		if !expectedDist[name] {
			t.Fatalf("publish inventory carries unexpected name %q", name)
		}
	}
	github := policy.GitHubRequiredStatusChecks
	if !github.Payload.Strict || len(github.Payload.Contexts) == 0 {
		t.Fatal("branch-protection payload must be strict with non-empty contexts")
	}
	for job := range requiredJobs {
		found := false
		for _, ctx := range github.Payload.Contexts {
			if ctx == job {
				found = true
			}
		}
		if !found {
			t.Fatalf("branch-protection contexts omit required job %s", job)
		}
	}
	for _, command := range []string{github.ApplyCommand, github.VerifyCommand, github.PreApplyNameConfirmation} {
		if strings.TrimSpace(command) == "" {
			t.Fatal("policy must document apply, verify, and pre-apply name-confirmation commands")
		}
	}
	for _, command := range []string{github.ApplyCommand, github.VerifyCommand} {
		if !strings.Contains(command, "RamXX/machinery") || !strings.Contains(command, "required_status_checks") {
			t.Fatalf("documented commands must target the required_status_checks endpoint of RamXX/machinery: %s", command)
		}
	}
	if !strings.Contains(strings.ToLower(policy.ApplicationStatus), "not applied") {
		t.Fatalf("policy must state plainly that it is not applied: %s", policy.ApplicationStatus)
	}
}
