package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/checker"
)

// piiFlowOCIRequiredEnv makes the real-engine pii-flow tests mandatory: the
// design-engines CI job sets it after provisioning both pinned images, so a
// missing engine or image fails there instead of skipping.
const piiFlowOCIRequiredEnv = "MACHINERY_REQUIRE_OCI_GOLDEN"

// piiFlowNoSouffleImage is the plain pinned CPython image the design-engines
// job already provisions. It carries no souffle, which is exactly what the
// missing-engine case needs.
const piiFlowNoSouffleImage = "python@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc"

type piiFlowOCI struct {
	engine  []string
	image   string
	closure string
	root    string
}

// requirePiiFlowOCI resolves Docker and the example registry's pinned Souffle
// image. Without the required-lane variable, an absent engine or image is a
// skip that says how to provision it; with it, either is a failure.
func requirePiiFlowOCI(t *testing.T) piiFlowOCI {
	t.Helper()
	required := os.Getenv(piiFlowOCIRequiredEnv) == "1"
	unavailable := func(format string, args ...any) {
		t.Helper()
		if required {
			t.Fatalf(format, args...)
		}
		t.Skipf(format+" (skipping; provision with scripts/pii-flow-image.sh, or set "+piiFlowOCIRequiredEnv+"=1 to require it)", args...)
	}
	docker, err := exec.LookPath("docker")
	if err != nil {
		unavailable("pii-flow Souffle tests need a Docker engine: %v", err)
	}
	if docker, err = filepath.EvalSymlinks(docker); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("DOCKER_HOST") == "" {
		endpoint, inspectErr := dockerContextEndpoint(docker)
		if inspectErr != nil {
			unavailable("pii-flow Souffle tests cannot resolve the Docker endpoint: %v", inspectErr)
		}
		t.Setenv("DOCKER_HOST", endpoint)
	}
	root := repoRootDir(t)
	reg, err := checker.LoadRegistry(filepath.Join(root, "examples", "pii-flow", "checkers.local.example.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := reg.Resolve("pii-flow")
	if !ok {
		t.Fatal("pii-flow registry entry is missing")
	}
	if err := verifyLocalOCIImage([]string{docker}, entry.Runtime.Image, entry.Runtime.Digest, entry.Runtime.Platform, checkerOCIControlPlaneTimeout, root); err != nil {
		unavailable("pinned pii-flow Souffle image is not provisioned: %v", err)
	}
	manifest, err := checker.LoadManifest(filepath.Join(root, "examples", "pii-flow", "design", "checkers", "pii-flow.checker.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return piiFlowOCI{engine: []string{docker}, image: entry.Runtime.Image, closure: manifest.Checker.RuntimeClosure, root: root}
}

// piiFlowRun is one adapter invocation's inputs, staged into a daemon-visible
// scratch directory with the same /work and read-only /checker topology
// verify-checkers uses.
type piiFlowRun struct {
	image      string
	mode       string
	projection []byte
	config     []byte
	rules      []byte
	evidence   []byte // committed evidence for verify mode
	trace      []byte // committed trace for verify mode
}

type piiFlowResult struct {
	output   string
	err      error
	evidence []byte
	trace    []byte
}

func (o piiFlowOCI) run(t *testing.T, r piiFlowRun) piiFlowResult {
	t.Helper()
	// Scratch lives inside the repository, like the verify-checkers ambient
	// probe, so a containerized CI daemon resolves the same bind paths.
	work, err := os.MkdirTemp(o.root, ".oci-pii-flow-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(work); err != nil {
			t.Error(err)
		}
	})
	design := filepath.Join(o.root, "examples", "pii-flow", "design", "checkers", "pii-flow")
	adapter, err := os.ReadFile(filepath.Join(design, "adapter.py"))
	if err != nil {
		t.Fatal(err)
	}
	inputs := filepath.Join(work, "runtime-inputs")
	write := func(path string, body []byte) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(inputs, "adapter.py"), adapter)
	write(filepath.Join(inputs, "rules.dl"), r.rules)
	write(filepath.Join(work, "projection.json"), r.projection)
	write(filepath.Join(work, "config.json"), r.config)
	evidencePath := filepath.Join(work, "evidence.json")
	tracePath := filepath.Join(work, "generated", "souffle-outputs.json")
	if r.mode == "verify" {
		write(evidencePath, r.evidence)
		write(tracePath, r.trace)
	}
	args := []string{"python3", "/checker/adapter.py", r.mode, "/work/projection.json", "/work/config.json", "/work/evidence.json", "/checker/rules.dl"}
	image := r.image
	if image == "" {
		image = o.image
	}
	out, runErr := runCheckerOCI(o.engine, image, "linux/amd64", args, o.closure, 120*time.Second, work)
	res := piiFlowResult{output: out, err: runErr}
	if body, err := os.ReadFile(evidencePath); err == nil {
		res.evidence = body
	}
	if body, err := os.ReadFile(tracePath); err == nil {
		res.trace = body
	}
	return res
}

func piiFlowShipped(t *testing.T, root string) (projection, config, rules, evidence, trace []byte) {
	t.Helper()
	dir := filepath.Join(root, "examples", "pii-flow", "design", "checkers")
	read := func(rel string) []byte {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	manifest, err := checker.LoadManifest(filepath.Join(dir, "pii-flow.checker.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	config, err = json.Marshal(manifest.Config)
	if err != nil {
		t.Fatal(err)
	}
	return read("pii-flow/projection.json"), config, read("pii-flow/rules.dl"), read("pii-flow/evidence.json"), read("pii-flow/generated/souffle-outputs.json")
}

// mutateJSON decodes body, applies edit, and re-encodes it.
func mutateJSON(t *testing.T, body []byte, edit func(map[string]any)) []byte {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatal(err)
	}
	edit(doc)
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func decodeEvidence(t *testing.T, body []byte) checker.Evidence {
	t.Helper()
	var ev checker.Evidence
	if err := json.Unmarshal(body, &ev); err != nil {
		t.Fatalf("evidence is not JSON: %v\n%s", err, body)
	}
	return ev
}

// TestPiiFlowSouffleVerdicts proves the reference checker's verdict comes from
// Souffle in the pinned image: the shipped design passes byte-for-byte with
// the committed evidence, a design whose sensitive data reaches a sink
// unredacted fails naming exactly that sink, and every fail-closed path
// leaves no evidence behind.
func TestPiiFlowSouffleVerdicts(t *testing.T) {
	oci := requirePiiFlowOCI(t)
	projection, config, rules, committedEvidence, committedTrace := piiFlowShipped(t, oci.root)

	t.Run("shipped design passes and reproduces the committed evidence", func(t *testing.T) {
		res := oci.run(t, piiFlowRun{mode: "run", projection: projection, config: config, rules: rules})
		if res.err != nil || res.output != "" {
			t.Fatalf("adapter failed: %v\n%s", res.err, res.output)
		}
		if !bytes.Equal(res.evidence, committedEvidence) {
			t.Fatalf("fresh evidence differs from the committed copy:\n%s", res.evidence)
		}
		if !bytes.Equal(res.trace, committedTrace) {
			t.Fatalf("fresh trace differs from the committed copy:\n%s", res.trace)
		}
		ev := decodeEvidence(t, res.evidence)
		// leak is a declared output relation that Souffle wrote empty: that
		// is an explicit pass row, not a coverage gap.
		if ev.Verdict != "pass" || len(ev.Findings) != 0 || len(ev.Coverage) != 1 ||
			ev.Coverage[0].Element != "inv:priv-no-unredacted-export" || ev.Coverage[0].Verdict != "pass" {
			t.Fatalf("shipped design verdict = %+v", ev)
		}
		if !strings.Contains(string(res.trace), `"leak": []`) {
			t.Fatalf("trace does not record the empty leak relation:\n%s", res.trace)
		}
	})

	leakCases := []struct {
		name      string
		edit      func(projection, config map[string]any)
		wantSinks []string
	}{
		{
			name: "a flow that bypasses the redactor leaks at the export sink",
			edit: func(p, _ map[string]any) {
				model := p["model"].(map[string]any)
				model["relationships"] = append(model["relationships"].([]any), map[string]any{
					"stable_id": "rel:ProcessingActivity->AnalyticsExport:1:1", "from": "entity:ProcessingActivity",
					"to": "entity:AnalyticsExport", "cardinality": "1:1",
				})
			},
			wantSinks: []string{"entity:AnalyticsExport"},
		},
		{
			name: "only the sink upstream of the redactor leaks",
			edit: func(_, c map[string]any) {
				c["sinks"] = []any{"entity:AnalyticsExport", "entity:ProcessingActivity"}
			},
			wantSinks: []string{"entity:ProcessingActivity"},
		},
	}
	for _, tc := range leakCases {
		t.Run(tc.name, func(t *testing.T) {
			var p, c map[string]any
			if err := json.Unmarshal(projection, &p); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(config, &c); err != nil {
				t.Fatal(err)
			}
			tc.edit(p, c)
			pBody, _ := json.Marshal(p)
			cBody, _ := json.Marshal(c)
			res := oci.run(t, piiFlowRun{mode: "run", projection: pBody, config: cBody, rules: rules})
			if res.err != nil || res.output != "" {
				t.Fatalf("adapter failed: %v\n%s", res.err, res.output)
			}
			ev := decodeEvidence(t, res.evidence)
			if ev.Verdict != "fail" || len(ev.Coverage) != 1 || ev.Coverage[0].Verdict != "fail" {
				t.Fatalf("leaking design verdict = %+v", ev)
			}
			var sinks []string
			for _, f := range ev.Findings {
				if f.Severity != "blocking" || f.Code != "leak" {
					t.Fatalf("unexpected finding %+v", f)
				}
				sinks = append(sinks, f.Element)
			}
			if !reflect.DeepEqual(sinks, tc.wantSinks) {
				t.Fatalf("leaking sinks = %v, want %v", sinks, tc.wantSinks)
			}
		})
	}

	failClosed := []struct {
		name  string
		run   piiFlowRun
		wants string
	}{
		{
			name:  "souffle missing from the image",
			run:   piiFlowRun{image: piiFlowNoSouffleImage, mode: "run", projection: projection, config: config, rules: rules},
			wants: "souffle executable not found on PATH",
		},
		{
			name:  "rules with a syntax error",
			run:   piiFlowRun{mode: "run", projection: projection, config: config, rules: append(append([]byte(nil), rules...), []byte("\nleak(E) :- tainted(E) sink(E).\n")...)},
			wants: "souffle exited",
		},
		{
			name:  "a declared output relation souffle never writes",
			run:   piiFlowRun{mode: "run", projection: projection, config: config, rules: bytes.Replace(rules, []byte(".output leak\n"), nil, 1)},
			wants: "souffle wrote no output for declared relation(s): leak.csv",
		},
		{
			name: "a fact value holding a tab",
			run: piiFlowRun{mode: "run", config: config, rules: rules, projection: mutateJSON(t, projection, func(p map[string]any) {
				entity := p["model"].(map[string]any)["entities"].([]any)[0].(map[string]any)
				entity["stable_id"] = "entity:Analytics\tExport"
			})},
			wants: "holds a tab, carriage return, or newline",
		},
		{
			name: "a projection layer the adapter does not read",
			run: piiFlowRun{mode: "run", config: config, rules: rules, projection: mutateJSON(t, projection, func(p map[string]any) {
				p["layers"] = map[string]any{"machines": map[string]any{"state": []any{}}}
			})},
			wants: "projection carries layers or keys this adapter does not read: layers",
		},
		{
			name: "a 2.0 projection",
			run: piiFlowRun{mode: "run", config: config, rules: rules, projection: mutateJSON(t, projection, func(p map[string]any) {
				p["projection_schema"] = "2.0"
			})},
			wants: "this adapter reads only the 1.0 contract",
		},
		{
			name: "replay of committed evidence that souffle does not reproduce",
			run: piiFlowRun{mode: "verify", projection: projection, config: config, rules: rules, trace: committedTrace,
				evidence: bytes.Replace(committedEvidence, []byte(`"verdict": "pass"`), []byte(`"verdict": "fail"`), 1)},
			wants: "committed evidence differs from what souffle derives now",
		},
	}
	for _, tc := range failClosed {
		t.Run(tc.name, func(t *testing.T) {
			res := oci.run(t, tc.run)
			if res.err == nil {
				t.Fatalf("adapter accepted the case; output:\n%s", res.output)
			}
			if !strings.Contains(res.output, tc.wants) {
				t.Fatalf("adapter diagnostic does not name the cause %q:\n%s", tc.wants, res.output)
			}
			if tc.run.mode == "run" && (res.evidence != nil || res.trace != nil) {
				t.Fatalf("a failed run left evidence or trace behind:\n%s", res.evidence)
			}
		})
	}

	t.Run("replay of the committed evidence succeeds silently", func(t *testing.T) {
		res := oci.run(t, piiFlowRun{mode: "verify", projection: projection, config: config, rules: rules, evidence: committedEvidence, trace: committedTrace})
		if res.err != nil || res.output != "" {
			t.Fatalf("replay failed: %v\n%s", res.err, res.output)
		}
	})
}
