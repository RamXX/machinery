package hook

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/gates"
)

const hookReviewLimits = "Hashes bind the observed files and scope; they do not prove tests ran, reviewer identity, or judgment correctness. Top-level Git administration and the exact Machinery attestation record are excluded; applications that use them as runtime inputs are outside this review boundary."
const hookReviewSource = "package example\n"

type hookReviewScenario struct {
	Event, Fault                        string
	Strict, Wave, Empty, Plan, Semantic bool
}
type hookReviewWriter struct {
	bytes.Buffer
	t                         *testing.T
	root, temporary, sentinel string
	scenario                  hookReviewScenario
	fired                     int
	run                       []*gates.Gate
}

// This is the exact PM-approved optional writer method. It compiles before
// production recognizes the interface; absence must fail the fired assertion.
func (w *hookReviewWriter) beforeAttestationFinalization(snapshot *gates.Snapshot, run []*gates.Gate) {
	w.fired++
	if w.fired != 1 {
		w.t.Fatal("finalization callback repeated")
	}
	if w.Len() != 0 {
		w.t.Fatalf("hook wrote output before finalization: %q", w.String())
	}
	w.run = run
	if w.scenario.Empty {
		if len(run) != 0 {
			w.t.Fatalf("empty selection supplied gates: %+v", run)
		}
	} else if !w.scenario.Plan && !w.scenario.Semantic {
		found := false
		for _, g := range run {
			if strings.HasPrefix(g.Title, "Gv-attest") {
				found = true
				if len(g.Errs) != 0 || !strings.Contains(strings.Join(g.Notes, "\n"), "current implementation review pending final snapshot release") {
					w.t.Fatalf("C valid provisional v2 control unavailable; mutation NOT YET EXERCISED: errors=%v notes=%v", g.Errs, g.Notes)
				}
				hookReviewNoCurrent(w.t, g)
			}
		}
		if !found {
			w.t.Fatal("current fixture did not reach Gv")
		}
	}
	switch w.scenario.Fault {
	case "original":
		writeFile(w.t, filepath.Join(w.root, "src", "handler.go"), "package changed\n")
	case "cleanup":
		copyRoot := snapshot.DesignPath()
		rel, err := filepath.Rel(w.temporary, copyRoot)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || copyRoot == filepath.Join(w.root, "design") {
			w.t.Fatalf("unsafe private-copy fixture path %q relative %q: %v", copyRoot, rel, err)
		}
		path := filepath.Join(copyRoot, "BUILD.md")
		if w.scenario.Empty {
			path = filepath.Join(copyRoot, "receipt-cleanup.txt")
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			w.t.Fatalf("expected real owned private regular entry: %v %v", info, err)
		}
		if err := os.Remove(path); err != nil {
			w.t.Fatal(err)
		}
		if err := os.Symlink(w.sentinel, path); err != nil {
			w.t.Fatal(err)
		}
		w.t.Log("C actual owned private regular entry replaced with sentinel symlink")
	}
}

func hookReviewHash(body string) string { return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(body))) }
func hookReviewNoCurrent(t *testing.T, g *gates.Gate) {
	t.Helper()
	for key, n := range g.Counts {
		if n > 0 && (strings.Contains(key, "implementation") || strings.Contains(key, "scope")) {
			t.Errorf("pending/failed hook published %s=%d", key, n)
		}
	}
}
func hookReviewFixture(t *testing.T, scenario hookReviewScenario) string {
	t.Helper()
	root := t.TempDir()
	enabled := true
	cfg := Config{Design: "design", Impl: "src", Gates: "gv", Hooks: &enabled, Strict: scenario.Strict}
	if scenario.Empty {
		cfg.Gates = "g4,gt"
		cfg.Impl = ""
	}
	config, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, ConfigName), string(config))
	if _, ok, warning := Load(root); !ok || warning != "" {
		t.Fatalf("hook fixture configuration invalid: ok=%t warning=%s", ok, warning)
	}
	writeFile(t, filepath.Join(root, "src", "handler.go"), hookReviewSource)
	writeFile(t, filepath.Join(root, "design", "receipt-cleanup.txt"), "owned cleanup witness\n")
	if scenario.Wave {
		writeFile(t, filepath.Join(root, "design", waveSentinelName), "open\n")
	}
	if !scenario.Empty {
		writeFile(t, filepath.Join(root, "design", "BUILD.md"), "# Build\n")
		row := map[string]any{"claim": "gt.conformance-test-shape", "kind": "current", "attestor": "R", "date": "2026-09-05", "covers": []map[string]string{{"path": "BUILD.md", "hash": hookReviewHash("# Build\n")}}}
		mode := func(path string) string {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			return fmt.Sprintf("%04o", info.Mode().Perm())
		}
		rootMode, fileMode := mode(filepath.Join(root, "src")), mode(filepath.Join(root, "src", "handler.go"))
		fileHash := hookReviewHash(hookReviewSource)
		grammar := fmt.Sprintf("machinery-attestation-scope-v1\nroot\t../src\npolicy\tfull-root-v1\nexclude\tvcs-root:.git\nexclude\tevidence:none\ndirectory\t.\t%s\nfile\thandler.go\t%s\t%d\t%s\n", rootMode, fileMode, len(hookReviewSource), fileHash)
		row["implementation"] = map[string]any{"root": "../src", "policy": "full-root-v1", "hash": hookReviewHash(grammar), "entries": []map[string]any{{"path": ".", "type": "directory", "mode": rootMode}, {"path": "handler.go", "type": "file", "mode": fileMode, "size": len(hookReviewSource), "hash": fileHash}}}
		plan := map[string]any{"claim": "g4.zero-context", "kind": "plan", "attestor": "R", "date": "2026-09-05", "covers": row["covers"]}
		version := 2
		rows := []map[string]any{row, plan}
		if scenario.Plan {
			version = 1
			delete(plan, "kind")
			rows = []map[string]any{plan}
		}
		if scenario.Semantic {
			version = 1
			delete(plan, "kind")
			plan["claim"] = "invented-claim"
			rows = []map[string]any{plan}
		}
		body, err := json.Marshal(map[string]any{"attestation_version": version, "attestations": rows})
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(root, "design", gates.AttestationsFileName), string(body))
	}
	if err := appendState(root, "attestation-review", "design"); err != nil {
		t.Fatal(err)
	}
	return root
}

func hookReviewExercise(t *testing.T, scenario hookReviewScenario, temporary string) {
	t.Helper()
	root := hookReviewFixture(t, scenario)
	sentinel := filepath.Join(t.TempDir(), "external-sentinel")
	writeFile(t, sentinel, "sentinel must survive\n")
	w := &hookReviewWriter{t: t, root: root, temporary: temporary, sentinel: sentinel, scenario: scenario}
	raw, err := json.Marshal(Input{HookEventName: scenario.Event, SessionID: "attestation-review", Cwd: root})
	if err != nil {
		t.Fatal(err)
	}
	runErr := Run(bytes.NewReader(raw), w, root)
	if w.fired != 1 {
		t.Fatalf("B INTERFACE_ABSENT: finalization callback fired %d times; no custody mutation credited; output=%q error=%v", w.fired, w.String(), runErr)
	}
	if got, err := os.ReadFile(sentinel); err != nil || string(got) != "sentinel must survive\n" {
		t.Fatalf("outside sentinel changed: %q %v", got, err)
	}
	var messages []stopOut
	dec := json.NewDecoder(strings.NewReader(w.String()))
	for {
		var msg stopOut
		err := dec.Decode(&msg)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("not pure stop JSON: %q: %v", w.String(), err)
		}
		messages = append(messages, msg)
	}
	state, err := readStateRecord(root, "attestation-review")
	if err != nil {
		t.Fatal(err)
	}
	if scenario.Fault != "none" {
		if len(messages) != 1 || messages[0].Decision != "block" || !state.design {
			t.Fatalf("custody must emit exactly one block and retain ledger even with relaxed/wave policy: messages=%+v state=%+v err=%v", messages, state, runErr)
		}
		text := messages[0].Reason + fmt.Sprint(runErr)
		if scenario.Fault == "cleanup" {
			if !strings.Contains(text, "private snapshot cleanup") || !strings.Contains(text, "symlink") {
				t.Fatalf("actual cleanup cause not exposed: %s", text)
			}
		} else if !strings.Contains(text, "changed") {
			t.Fatalf("actual original mutation cause missing: %s", text)
		}
		for _, g := range w.run {
			hookReviewNoCurrent(t, g)
			if strings.Contains(strings.Join(g.Notes, "\n"), "current implementation review pending") {
				t.Fatalf("failed result stayed pending: %v", g.Notes)
			}
		}
		for _, prefix := range []string{"machinery-design-source-", "machinery-impl-snapshot-", "machinery-attestation-"} {
			if strings.Contains(w.String(), prefix) {
				t.Fatalf("private path leaked in stop JSON: %q", w.String())
			}
		}
	} else {
		if runErr != nil {
			t.Fatalf("no-fault finalization: %v", runErr)
		}
		for _, m := range messages {
			if m.Decision == "block" {
				t.Fatalf("no-fault configured outcome blocked: %+v", m)
			}
		}
		retain := scenario.Semantic && scenario.Strict && scenario.Wave
		if state.design != retain {
			t.Fatalf("no-fault ledger state=%+v want retained=%t", state, retain)
		}
		if !scenario.Plan && !scenario.Empty && !scenario.Semantic {
			found := false
			for _, g := range w.run {
				if strings.HasPrefix(g.Title, "Gv-attest") {
					found = true
					if g.Counts["current implementation reviews"] != 1 || !strings.Contains(strings.Join(g.Notes, "\n"), hookReviewLimits) {
						t.Fatalf("hook returned before finalized current finding/limits: %+v", g)
					}
				}
			}
			if !found {
				t.Fatal("current Gv result absent")
			}
		}
		if scenario.Semantic {
			if len(messages) != 1 || messages[0].SystemMessage == "" {
				t.Fatalf("semantic warning/wave control lost its existing report: %+v", messages)
			}
		}
	}
	t.Logf("OBSERVED callback=%d fault=%s event=%s strict=%t wave=%t output-messages=%d ledger-retained=%t", w.fired, scenario.Fault, scenario.Event, scenario.Strict, scenario.Wave, len(messages), state.design)
}

// Parent always runs every native case. The '--' protocol selects an explicit
// child invocation, not a skip-if-runtime/feature-missing escape. Destructive
// copy cases run with private TMPDIR before acquisition; process exit closes
// held handles and the parent owns cleanup of this exact temporary subtree.
func TestAttestationHookProcess(t *testing.T) {
	for i, arg := range os.Args {
		if arg == "attestation-hook-child" {
			if len(os.Args) != i+3 {
				t.Fatal("invalid child arguments")
			}
			var scenario hookReviewScenario
			if err := json.Unmarshal([]byte(os.Args[i+1]), &scenario); err != nil {
				t.Fatal(err)
			}
			temporary := os.Args[i+2]
			if os.TempDir() != temporary {
				t.Fatalf("TMPDIR isolation not active: %s != %s", os.TempDir(), temporary)
			}
			isolateHookState(t)
			control := scenario
			control.Fault = "none"
			hookReviewExercise(t, control, temporary)
			if scenario.Fault != "none" {
				t.Log("C fired unchanged control passed; actual fault leg begins")
				hookReviewExercise(t, scenario, temporary)
			}
			return
		}
	}
	cases := []struct {
		name     string
		scenario hookReviewScenario
	}{
		{"B-callback-on-legacy-plan", hookReviewScenario{Event: "Stop", Fault: "none", Plan: true}},
		{"B-empty-selection", hookReviewScenario{Event: "Stop", Fault: "none", Empty: true}},
		{"C-empty-cleanup", hookReviewScenario{Event: "Stop", Fault: "cleanup", Empty: true}},
		{"B-semantic-warning-with-finalization", hookReviewScenario{Event: "Stop", Fault: "none", Semantic: true}},
		{"B-semantic-wave-with-finalization", hookReviewScenario{Event: "Stop", Fault: "none", Semantic: true, Strict: true, Wave: true}},
	}
	for _, event := range []string{"Stop", "SubagentStop"} {
		for _, policy := range []struct {
			name         string
			strict, wave bool
		}{{"relaxed", false, false}, {"strict", true, false}, {"wave", true, true}} {
			for _, fault := range []string{"none", "original", "cleanup"} {
				cases = append(cases, struct {
					name     string
					scenario hookReviewScenario
				}{"C-" + event + "/" + policy.name + "/" + fault, hookReviewScenario{Event: event, Fault: fault, Strict: policy.strict, Wave: policy.wave}})
			}
		}
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			temporary := t.TempDir()
			body, err := json.Marshal(tc.scenario)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAttestationHookProcess$", "-test.v", "--", "attestation-hook-child", string(body), temporary)
			for _, e := range os.Environ() {
				if !strings.HasPrefix(e, "TMPDIR=") {
					cmd.Env = append(cmd.Env, e)
				}
			}
			cmd.Env = append(cmd.Env, "TMPDIR="+temporary)
			out, err := cmd.CombinedOutput()
			t.Logf("helper output:\n%s", out)
			if ctx.Err() != nil {
				t.Fatalf("native helper timeout is not RED: %v", ctx.Err())
			}
			if err != nil {
				t.Fatalf("attestation hook assertions: %v", err)
			}
		})
	}
}

func TestAttestationDHookCompatibility(t *testing.T) {
	for _, s := range []hookReviewScenario{{Event: "Stop", Fault: "none", Plan: true}, {Event: "SubagentStop", Fault: "none", Plan: true}, {Event: "Stop", Fault: "none", Semantic: true}, {Event: "Stop", Fault: "none", Semantic: true, Strict: true, Wave: true}} {
		t.Run(fmt.Sprintf("%s/semantic=%t/wave=%t", s.Event, s.Semantic, s.Wave), func(t *testing.T) {
			isolateHookState(t)
			root := hookReviewFixture(t, s)
			out := runEvent(t, root, Input{HookEventName: s.Event, SessionID: "attestation-review", Cwd: root})
			var message stopOut
			if out != "" {
				if err := json.Unmarshal([]byte(out), &message); err != nil {
					t.Fatal(err)
				}
			}
			if message.Decision != "" {
				t.Fatalf("baseline semantic policy changed: %s", out)
			}
			state, err := readStateRecord(root, "attestation-review")
			if err != nil {
				t.Fatal(err)
			}
			if state.design != (s.Semantic && s.Strict && s.Wave) {
				t.Fatalf("baseline ledger policy changed: %+v", state)
			}
		})
	}
}
