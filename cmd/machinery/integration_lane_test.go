//go:build machinery_integration

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
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/formal"
	"github.com/RamXX/machinery/internal/processcontrol"
	"github.com/RamXX/machinery/internal/runtimeclosure"
)

const integrationPilotImage = "python@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc"
const integrationPilotPlatform = "linux/amd64"

// This is a real-runtime lane: absence is a hard failure, never a skip. All
// container IDs below come from Docker's cidfile, are paired to a unique run
// label, and are reclaimed individually. No broad Docker cleanup is permitted.
func integrationExec(t *testing.T, timeout time.Duration, argv ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	out, errout, err := processcontrol.RunCapturedStreamLimits(ctx, cmd, 1<<20, 1<<20)
	if err != nil {
		t.Fatalf("real process %s failed: %v stdout=%s stderr=%s", argv[0], err, out, errout)
	}
	return out
}

func integrationWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func integrationOwnContainer(t *testing.T, work, runID, program string) (string, string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(work, "containers"), 0o700); err != nil {
		t.Fatal(err)
	}
	cidfile := filepath.Join(work, "containers", "pilot.cid")
	cleanup := func() {
		cid, readErr := os.ReadFile(cidfile)
		if os.IsNotExist(readErr) {
			return
		}
		if readErr != nil {
			t.Errorf("read owned cid: %v", readErr)
			return
		}
		id := strings.TrimSpace(string(cid))
		if len(id) != 64 {
			t.Errorf("refuse cleanup for malformed cid %q", id)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", `{{index .Config.Labels "dev.machinery.integration-run"}}`, id)
		b, err := cmd.CombinedOutput()
		if err != nil && strings.Contains(strings.ToLower(string(b)), "no such object") {
			return
		}
		if err != nil || strings.TrimSpace(string(b)) != runID {
			t.Errorf("refuse cleanup: ownership of %s changed: %v %s", id, err, b)
			return
		}
		cmd = exec.CommandContext(ctx, "docker", "rm", "-f", id)
		b, err = cmd.CombinedOutput()
		if err != nil {
			t.Errorf("owned container cleanup %s failed: %v %s", id, err, b)
		}
	}
	t.Cleanup(cleanup)
	out := integrationExec(t, 30*time.Second, "docker", "run", "-d", "--pull=never", "--platform", integrationPilotPlatform, "--network=none", "--read-only", "--memory=128m", "--cpus=0.5", "--pids-limit=32", "--label", "dev.machinery.integration-run="+runID, "--cidfile", cidfile, integrationPilotImage, "python3", "-c", program)
	b, err := os.ReadFile(cidfile)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.TrimSpace(string(b))
	if len(id) != 64 || strings.TrimSpace(out) != id {
		t.Fatalf("noncanonical owned cid: %q %q", id, out)
	}
	return id, cidfile
}

func TestIntegrationLanePilotOCI(t *testing.T) {
	// Pulling this exact immutable pin is harmless with a warm daemon cache and
	// provisions a cold cache. No existing image/container is removed to fake cold.
	out := integrationExec(t, 10*time.Minute, "docker", "pull", "--quiet", "--platform", integrationPilotPlatform, integrationPilotImage)
	if strings.TrimSpace(out) != integrationPilotImage && strings.TrimSpace(out) != "docker.io/library/"+integrationPilotImage {
		t.Fatalf("unexpected pull receipt: %q", out)
	}
	out = integrationExec(t, 30*time.Second, "docker", "image", "inspect", "--format", "{{json .RepoDigests}} {{.Os}}/{{.Architecture}}", integrationPilotImage)
	if !strings.Contains(out, `"`+integrationPilotImage+`"`) || !strings.HasSuffix(strings.TrimSpace(out), integrationPilotPlatform) {
		t.Fatalf("wrong pinned image/platform: %s", out)
	}
	for _, tc := range []struct {
		name, program string
		terminate     bool
	}{
		{"success", `print(6*7)`, false},
		{"termination", `import time; time.sleep(300)`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			work := t.TempDir()
			runID := fmt.Sprintf("machinery-pilot-%d-%d", os.Getpid(), time.Now().UnixNano())
			id, _ := integrationOwnContainer(t, work, runID, tc.program)
			if tc.terminate {
				integrationExec(t, 15*time.Second, "docker", "kill", id)
			} else {
				if strings.TrimSpace(integrationExec(t, 30*time.Second, "docker", "wait", id)) != "0" {
					t.Fatal("OCI arithmetic process failed")
				}
				if strings.TrimSpace(integrationExec(t, 15*time.Second, "docker", "logs", id)) != "42" {
					t.Fatal("OCI did not compute expected result")
				}
			}
			integrationExec(t, 15*time.Second, "docker", "rm", "-f", id)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			b, err := exec.CommandContext(ctx, "docker", "inspect", id).CombinedOutput()
			if err == nil || !strings.Contains(strings.ToLower(string(b)), "no such object") {
				t.Fatalf("owned container survived cleanup: %v %s", err, b)
			}
		})
	}
}

func TestIntegrationLanePilotFormal(t *testing.T) {
	cache := os.Getenv("MACHINERY_INTEGRATION_CACHE")
	if cache == "" {
		cache = t.TempDir()
	}
	// A private user cache is required on both supported OS families. This only
	// changes the isolated test process, never the maintainer's installed runtime.
	t.Setenv("HOME", cache)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(cache, "cache"))
	for _, key := range []string{"MACHINERY_JAVA", "MACHINERY_JAVA_CLOSURE_SHA256", "TLA_TOOLS_JAR", "TLA_TOOLS_JAR_SHA256", "JAVA_TOOL_OPTIONS", "JDK_JAVA_OPTIONS", "CLASSPATH"} {
		t.Setenv(key, "")
	}
	design := t.TempDir()
	integrationWrite(t, filepath.Join(design, "formal/Pilot.tla"), "\\* machinery:manual\n---- MODULE Pilot ----\nEXTENDS Integers\nVARIABLE x\nInit == x = 0\nNext == x' = 1 - x\nSpec == Init /\\ [][Next]_x\nSafe == x \\in {0, 1}\n====\n")
	integrationWrite(t, filepath.Join(design, "formal/Pilot.cfg"), "SPECIFICATION Spec\nINVARIANT Safe\n")
	var out, errout bytes.Buffer
	rc := formal.VerifyFormalTo(design, false, &out, &errout)
	if rc != 0 || !strings.Contains(out.String(), "PASS  Pilot") || !strings.Contains(out.String(), "1 passed, 0 failed") {
		t.Fatalf("actual pinned Java/TLC did not prove finite model: status=%d stdout=%s stderr=%s", rc, out.String(), errout.String())
	}
	// A real unsafe mutation must yield a counterexample, not a success banner.
	integrationWrite(t, filepath.Join(design, "formal/Pilot.cfg"), "SPECIFICATION Spec\nINVARIANT Broken\n")
	integrationWrite(t, filepath.Join(design, "formal/Pilot.tla"), "\\* machinery:manual\n---- MODULE Pilot ----\nEXTENDS Integers\nVARIABLE x\nInit == x = 0\nNext == x' = 1 - x\nSpec == Init /\\ [][Next]_x\nBroken == x = 0\n====\n")
	out.Reset()
	errout.Reset()
	rc = formal.VerifyFormalTo(design, false, &out, &errout)
	if rc == 0 || !strings.Contains(out.String()+errout.String(), "Broken") {
		t.Fatalf("actual pinned TLC missed unsafe transition: %d %s %s", rc, out.String(), errout.String())
	}
	// Validate the exact native test source used by the candidate CLI positive.
	// This control cannot satisfy that separate CLI assertion; it only prevents
	// a fixture compilation/runtime defect being mistaken for behavioral RED.
	java, e := runtimeclosure.OpenJava()
	if e != nil {
		t.Fatal(e)
	}
	javaPath := java.Path()
	if e = java.Close(); e != nil {
		t.Fatal(e)
	}
	userCache, e := os.UserCacheDir()
	if e != nil {
		t.Fatal(e)
	}
	jar := filepath.Join(userCache, "machinery", "tla2tools-v1.7.4.jar")
	fixture := t.TempDir()
	integrationWrite(t, filepath.Join(fixture, "go.mod"), "module lane.example/formalcontrol\n\ngo 1.27.0\n")
	marker := filepath.Join(fixture, "executed")
	closure := filepath.Join(fixture, "closure.json")
	integrationWrite(t, filepath.Join(fixture, "pilot_integration_test.go"), integrationFormalGo(marker, closure))
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-json", "-count=1", "-tags", "machinery_integration", ".")
	cmd.Dir = fixture
	cmd.Env = append(os.Environ(), "MACHINERY_INTEGRATION_JAVA="+javaPath, "MACHINERY_INTEGRATION_TLC_JAR="+jar)
	b, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("formal native fixture control failed: %v %s", e, b)
	}
	if b, e := os.ReadFile(marker); e != nil || string(b) != "executed" {
		t.Fatalf("formal fixture did not execute actual closure: %v %s", e, b)
	}
}

func TestIntegrationLaneFullPath(t *testing.T) {
	integrationExec(t, 10*time.Minute, "docker", "pull", "--quiet", "--platform", integrationPilotPlatform, integrationPilotImage)
	unrelatedID, _ := integrationOwnContainer(t, t.TempDir(), fmt.Sprintf("unrelated-pilot-%d-%d", os.Getpid(), time.Now().UnixNano()), "import time; time.sleep(900)")
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "integration-lane")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binary, "./scripts/integration-lane")
	cmd.Dir = repo
	b, err := cmd.CombinedOutput()
	cancel()
	if err != nil {
		t.Fatalf("bootstrap/contract build failure is not RED evidence: %v %s", err, b)
	}
	for _, scenario := range []string{"success", "cold-cache", "missing-docker", "wrong-pin", "wrong-digest", "wrong-platform", "wrong-java-pin", "wrong-tlc-pin", "offline-formal", "leaked-resource", "failed-assertion", "timeout", "cancellation", "node-success", "node-missing", "node-zero", "node-skipped", "node-partial", "node-aggregate-only", "node-duplicate", "node-missing-terminal", "process-success", "process-failure", "process-timeout", "process-cancellation"} {
		t.Run(scenario, func(t *testing.T) {
			t.Cleanup(func() {
				state := integrationExec(t, 10*time.Second, "docker", "inspect", "--format", "{{.State.Running}}", unrelatedID)
				if strings.TrimSpace(state) != "true" {
					t.Error("lane touched an unrelated container")
				}
			})
			root := t.TempDir()
			cache := filepath.Join(t.TempDir(), "fresh-cache")
			work := filepath.Join(t.TempDir(), "owned")
			integrationWrite(t, filepath.Join(work, "user-sentinel"), "caller-owned bytes must survive\n")
			workInfo, e := os.Stat(work)
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() {
				info, e := os.Stat(work)
				if e != nil || !os.SameFile(info, workInfo) {
					t.Errorf("caller-owned work root removed/replaced: %v", e)
					return
				}
				b, e := os.ReadFile(filepath.Join(work, "user-sentinel"))
				if e != nil || string(b) != "caller-owned bytes must survive\n" {
					t.Errorf("caller-owned sentinel changed: %v %q", e, b)
				}
				entries, e := os.ReadDir(work)
				if e != nil || len(entries) != 1 || entries[0].Name() != "user-sentinel" {
					t.Errorf("owned work residue or caller data loss: %v %v", e, entries)
				}
			})
			report := filepath.Join(t.TempDir(), "report.json")
			for _, name := range []string{"schema.json", "runtime-pins.json"} {
				b, e := os.ReadFile(filepath.Join(repo, "testdata/integration-lanes", name))
				if e != nil {
					t.Fatal(e)
				}
				integrationWrite(t, filepath.Join(root, "testdata/integration-lanes", name), string(b))
			}
			javaPin, e := os.ReadFile(filepath.Join(repo, ".java-runtime-pin"))
			if e != nil {
				t.Fatal(e)
			}
			integrationWrite(t, filepath.Join(root, ".java-runtime-pin"), string(javaPin))
			integrationWrite(t, filepath.Join(root, "go.mod"), "module lane.example/fullpath\n\ngo 1.27.0\n")
			integrationWrite(t, filepath.Join(root, "sample/native.go"), "package sample\n")
			marker := filepath.Join(root, "executed")
			cidMarker := filepath.Join(root, "created-cid")
			pidMarker := filepath.Join(root, "created-pid")
			t.Cleanup(func() { integrationEmergencyCleanup(t, cidMarker) })
			t.Cleanup(func() { integrationHostCleanup(t, pidMarker) })
			suite := map[string]any{"id": "fullpath", "lane": "required", "adapter": "go-json", "package": "./sample", "source_files": []string{"sample/pilot_integration_test.go"}, "tests": []string{"TestPilot"}, "runtimes": []string{"go", "docker"}, "timeout": "30s", "stdout_limit": 1048576, "stderr_limit": 1048576}
			body := integrationFullPathGo(marker, cidMarker, scenario)
			if scenario == "cold-cache" || scenario == "offline-formal" || scenario == "wrong-java-pin" || scenario == "wrong-tlc-pin" {
				suite["runtimes"] = []string{"go", "java", "tlc"}
				body = integrationFormalGo(marker, filepath.Join(root, "executed-closure.json"))
				if entries, e := os.ReadDir(cache); !os.IsNotExist(e) && (e != nil || len(entries) != 0) {
					t.Fatalf("formal positive/negative cache was not empty: %v %v", e, entries)
				}
			}
			if strings.HasPrefix(scenario, "process-") {
				suite["runtimes"] = []string{"go"}
				body = integrationProcessGo(marker, pidMarker, scenario)
			}
			if strings.HasPrefix(scenario, "node-") {
				suite["adapter"] = "node-tap"
				suite["package"] = "."
				suite["source_files"] = []string{"sample/pilot.integration.test.mjs"}
				suite["tests"] = []string{"native pilot"}
				suite["runtimes"] = []string{"node"}
				js := fmt.Sprintf("import test from 'node:test'; import assert from 'node:assert/strict'; import fs from 'node:fs'; test('native pilot',()=>{fs.writeFileSync(%q,'executed');assert.equal(6*7,42)});\n", marker)
				switch scenario {
				case "node-zero":
					js = "import test from 'node:test'; process.exit(0); test('native pilot',()=>{});\n"
				case "node-skipped":
					js = "import test from 'node:test'; test.skip('native pilot',()=>{});\n"
				case "node-partial":
					js = "import test from 'node:test'; test('native pilot',()=>{process.stdout.write('TAP version 13\\nok 1');process.exit(0)});\n"
				case "node-aggregate-only":
					js = "import test from 'node:test'; console.log('TAP version 13\\n1..1\\n# pass 1\\n# fail 0'); process.exit(0); test('native pilot',()=>{});\n"
				case "node-duplicate":
					js = "import test from 'node:test'; test('native pilot',()=>{}); test('native pilot',()=>{});\n"
				case "node-missing-terminal":
					js = fmt.Sprintf("import test from 'node:test'; import fs from 'node:fs'; test('native pilot',async()=>{fs.writeFileSync(%q,'executed');setInterval(()=>{},1000);await new Promise(()=>{})});\n", marker)
				}
				integrationWrite(t, filepath.Join(root, "sample/pilot.integration.test.mjs"), js)
			} else {
				integrationWrite(t, filepath.Join(root, "sample/pilot_integration_test.go"), body)
			}
			if scenario == "timeout" || scenario == "process-timeout" || scenario == "node-missing-terminal" {
				suite["timeout"] = "10s"
			}
			b, err := json.Marshal(map[string]any{"version": 1, "suites": []any{suite}})
			if err != nil {
				t.Fatal(err)
			}
			integrationWrite(t, filepath.Join(root, "testdata/integration-lanes/pilot.json"), string(b))
			if scenario == "wrong-pin" || scenario == "wrong-digest" || scenario == "wrong-platform" {
				pinpath := filepath.Join(root, "testdata/integration-lanes/runtime-pins.json")
				p, e := os.ReadFile(pinpath)
				if e != nil {
					t.Fatal(e)
				}
				if scenario == "wrong-pin" {
					p = bytes.ReplaceAll(p, []byte(integrationPilotImage), []byte("python:latest"))
				} else if scenario == "wrong-digest" {
					p = bytes.ReplaceAll(p, []byte(integrationPilotImage), []byte("python@sha256:"+strings.Repeat("a", 64)))
				} else {
					p = bytes.ReplaceAll(p, []byte(integrationPilotPlatform), []byte("linux/not-a-real-architecture"))
				}
				integrationWrite(t, pinpath, string(p))
			}
			if scenario == "wrong-java-pin" || scenario == "wrong-tlc-pin" {
				pinpath := filepath.Join(root, "testdata/integration-lanes/runtime-pins.json")
				p, e := os.ReadFile(pinpath)
				if e != nil {
					t.Fatal(e)
				}
				if scenario == "wrong-java-pin" {
					p = bytes.ReplaceAll(p, []byte("21.0.12.1+1"), []byte("99.0.0+1"))
				} else {
					p = bytes.ReplaceAll(p, []byte("936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88"), []byte(strings.Repeat("0", 64)))
				}
				integrationWrite(t, pinpath, string(p))
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, "--root", root, "--lane", "required", "--report", report, "--cache-dir", cache, "--work-dir", work)
			cmd.Env = os.Environ()
			if scenario == "missing-docker" {
				cmd.Env = append(cmd.Env, "DOCKER_HOST=unix:///unavailable/machinery.sock")
			}
			if scenario == "node-missing" {
				cmd.Env = append(cmd.Env, "PATH="+t.TempDir())
			}
			if scenario == "offline-formal" {
				cmd.Env = append(cmd.Env, "HTTPS_PROXY=http://127.0.0.1:1", "HTTP_PROXY=http://127.0.0.1:1", "NO_PROXY=")
			}
			var output bytes.Buffer
			cmd.Stdout = &output
			cmd.Stderr = &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			if scenario == "cancellation" || scenario == "process-cancellation" {
				creation := cidMarker
				if scenario == "process-cancellation" {
					creation = pidMarker
				}
				deadline := time.Now().Add(25 * time.Second)
				for time.Now().Before(deadline) {
					if _, err := os.Stat(creation); err == nil {
						break
					}
					time.Sleep(20 * time.Millisecond)
				}
				if _, err := os.Stat(creation); err != nil {
					_ = cmd.Process.Signal(os.Interrupt)
					_ = cmd.Wait()
					t.Fatalf("cancellation control never created real owned resource: %v output=%s", err, output.String())
				}
				if err := cmd.Process.Signal(os.Interrupt); err != nil {
					_ = cmd.Wait()
					t.Fatal(err)
				}
			}
			err = cmd.Wait()
			wantPass := scenario == "success" || scenario == "cold-cache" || scenario == "node-success" || scenario == "process-success"
			if wantPass && err != nil {
				t.Fatalf("provisioned real runtime execution rejected: %v %s", err, output.String())
			}
			if !wantPass && err == nil {
				t.Fatalf("unsafe execution accepted: %s", output.String())
			}
			if !wantPass {
				diagnostic := map[string]string{"missing-docker": "docker", "wrong-pin": "pin", "wrong-digest": "pin", "wrong-platform": "platform", "wrong-java-pin": "pin", "wrong-tlc-pin": "pin", "offline-formal": "provision", "leaked-resource": "leak", "failed-assertion": "failed", "timeout": "timeout", "cancellation": "cancel", "node-missing": "node", "node-zero": "execution", "node-skipped": "skip", "node-partial": "execution", "node-aggregate-only": "execution", "node-duplicate": "duplicate", "node-missing-terminal": "timeout", "process-failure": "failed", "process-timeout": "timeout", "process-cancellation": "cancel"}[scenario]
				if !strings.Contains(strings.ToLower(output.String()), diagnostic) {
					t.Fatalf("failure was not the intended %s outcome: %s", diagnostic, output.String())
				}
			}
			if scenario == "leaked-resource" || scenario == "failed-assertion" || scenario == "timeout" || scenario == "cancellation" {
				if _, e := os.Stat(cidMarker); e != nil {
					t.Fatalf("resource-lifecycle challenge never created its real container: %v %s", e, output.String())
				}
			}
			if wantPass {
				if b, e := os.ReadFile(marker); e != nil || string(b) != "executed" {
					t.Fatalf("success without actual test body: %q %v", b, e)
				}
			}
			if strings.HasPrefix(scenario, "process-") {
				pid, script := integrationReadPID(t, pidMarker)
				if integrationHostAlive(t, pid, script) {
					t.Fatal("owned host descendant survived lane return")
				}
			}
			if scenario == "node-missing-terminal" {
				if b, e := os.ReadFile(marker); e != nil || string(b) != "executed" {
					t.Fatalf("Node timeout challenge never entered required body: %v %q", e, b)
				}
			}
			var receipt struct {
				Status   string                      `json:"status"`
				Suites   []integrationSuiteReceipt   `json:"suites"`
				Runtimes []integrationRuntimeReceipt `json:"runtimes"`
				Cleanup  struct{ Status string }     `json:"cleanup"`
			}
			b, e = os.ReadFile(report)
			if e != nil {
				t.Fatalf("missing machine-readable failed/success report: %v %s", e, output.String())
			}
			if e = json.Unmarshal(b, &receipt); e != nil {
				t.Fatal(e)
			}
			if wantPass && (receipt.Status != "passed" || len(receipt.Suites) != 1 || receipt.Suites[0].Started != 1 || receipt.Suites[0].Passed != 1 || receipt.Cleanup.Status != "passed") {
				t.Fatalf("report substitutes for execution: %+v", receipt)
			}
			if !wantPass && receipt.Status != "failed" {
				t.Fatalf("failed execution report not fail-closed: %+v", receipt)
			}
			if wantPass {
				integrationAssertNativeReceipt(t, receipt.Suites[0], suite)
				integrationAssertRuntimes(t, receipt.Runtimes, suite["runtimes"].([]string), cache, javaPin)
				if scenario == "cold-cache" {
					integrationAssertExecutedClosure(t, root, receipt.Runtimes)
				}
			}
			if cid, e := os.ReadFile(cidMarker); e == nil {
				id, _ := integrationReadOwnedCID(t, cid)
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cleanupCancel()
				inspect, e := exec.CommandContext(cleanupCtx, "docker", "inspect", id).CombinedOutput()
				if e == nil {
					t.Fatalf("owned resource leaked beyond lane return: %s (verified emergency cleanup registered)", inspect)
				}
				if !bytes.Contains(bytes.ToLower(inspect), []byte("no such object")) {
					t.Fatalf("cannot verify owned cleanup: %v %s", e, inspect)
				}
			}
		})
	}
}

func integrationFullPathGo(marker, cidMarker, scenario string) string {
	return fmt.Sprintf(`//go:build machinery_integration

package sample
import("context";"os";"os/exec";"path/filepath";"strings";"testing";"time")
func TestPilot(t *testing.T) {
  work:=os.Getenv("MACHINERY_INTEGRATION_WORK");runID:=os.Getenv("MACHINERY_INTEGRATION_RUN_ID")
  if work==""||runID=="" {t.Fatal("runner did not allocate owned resource scope")}
  if e:=os.MkdirAll(filepath.Join(work,"containers"),0700);e!=nil{t.Fatal(e)}
  ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second);defer cancel()
  cidFile:=filepath.Join(work,"containers","child.cid")
  out,e:=exec.CommandContext(ctx,"docker","run","-d","--pull=never","--platform",%q,"--network=none","--read-only","--memory=128m","--cpus=0.5","--pids-limit=32","--label","dev.machinery.integration-run="+runID,"--cidfile",cidFile,%q,"python3","-c","import time; time.sleep(90)").CombinedOutput()
  if e!=nil{t.Fatalf("real pinned OCI could not start: %%v %%s",e,out)}
  id:=strings.TrimSpace(string(out));if len(id)!=64{t.Fatal("invalid cid")}
  if e:=os.WriteFile(%q,[]byte(id+"\n"+runID),0600);e!=nil{t.Fatal(e)}
  if e:=os.WriteFile(%q,[]byte("executed"),0600);e!=nil{t.Fatal(e)}
  switch %q {
  case "leaked-resource": return
  case "failed-assertion": t.Fatal("intentional safety assertion")
  case "timeout","cancellation": time.Sleep(time.Minute)
  }
  if out,e=exec.CommandContext(ctx,"docker","rm","-f",id).CombinedOutput();e!=nil{t.Fatalf("owned cleanup: %%v %%s",e,out)}
}
`, integrationPilotPlatform, integrationPilotImage, cidMarker, marker, scenario)
}

// All identities below are checked independently against bytes/native output,
// never by accepting an arbitrary nonempty record from the candidate report.
type integrationSuiteReceipt struct {
	ID, Adapter                                string
	Selected, Started, Passed, Failed, Skipped int
	Tests                                      []struct{ Name, Source, Status string }
	Events                                     string `json:"events_file"`
	EventsSHA                                  string `json:"events_sha256"`
}
type integrationRuntimeReceipt struct {
	ID, Identity, Status, Path string
	SHA256                     string `json:"sha256"`
	ClosureSHA                 string `json:"closure_sha256"`
	ArchiveSHA                 string `json:"archive_sha256"`
}

func integrationFileSHA(t *testing.T, path string) string {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	return fmt.Sprintf("%x", sha256.Sum256(b))
}

func integrationAssertNativeReceipt(t *testing.T, r integrationSuiteReceipt, s map[string]any) {
	t.Helper()
	name := s["tests"].([]string)[0]
	source := s["source_files"].([]string)[0]
	if r.ID != s["id"] || r.Adapter != s["adapter"] || r.Selected != 1 || r.Started != 1 || r.Passed != 1 || r.Failed != 0 || r.Skipped != 0 || len(r.Tests) != 1 || r.Tests[0].Name != name || r.Tests[0].Source != source || r.Tests[0].Status != "passed" {
		t.Fatalf("inexact native receipt: %+v", r)
	}
	b, e := os.ReadFile(r.Events)
	if e != nil {
		t.Fatalf("missing retained native events: %v", e)
	}
	if len(b) == 0 || r.EventsSHA != fmt.Sprintf("%x", sha256.Sum256(b)) {
		t.Fatal("native event bytes/hash not bound")
	}
	starts, passes := 0, 0
	if r.Adapter == "go-json" {
		for _, line := range bytes.Split(bytes.TrimSpace(b), []byte("\n")) {
			var event struct{ Action, Test, Package string }
			if e := json.Unmarshal(line, &event); e != nil {
				t.Fatal(e)
			}
			if event.Action == "skip" || event.Action == "fail" {
				t.Fatalf("unexpected native terminal: %s", line)
			}
			if event.Test != "" && event.Test != name {
				t.Fatalf("unexpected native identity: %s", line)
			}
			if event.Test == name {
				if event.Package != "lane.example/fullpath/sample" {
					t.Fatalf("wrong package identity: %s", line)
				}
				if event.Action == "run" {
					starts++
				}
				if event.Action == "pass" {
					passes++
				}
			}
		}
	} else {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "# Subtest: ") {
				if line != "# Subtest: "+name {
					t.Fatalf("wrong Node native identity: %s", line)
				}
				starts++
			}
			if strings.HasPrefix(line, "ok ") {
				if line != "ok 1 - "+name {
					t.Fatalf("wrong Node terminal identity: %s", line)
				}
				passes++
			}
			if strings.HasPrefix(line, "not ok ") || strings.Contains(line, "# SKIP") || strings.Contains(line, "# TODO") {
				t.Fatalf("unexpected Node terminal: %s", line)
			}
		}
		if !strings.Contains(string(b), "\n1..1\n") || !strings.Contains(string(b), "\n# tests 1\n") || !strings.Contains(string(b), "\n# pass 1\n") || !strings.Contains(string(b), "\n# fail 0\n") {
			t.Fatalf("incomplete Node native stream: %s", b)
		}
	}
	if starts != 1 || passes != 1 {
		t.Fatalf("native terminal cardinality starts=%d passes=%d", starts, passes)
	}
}

func integrationAssertRuntimes(t *testing.T, rs []integrationRuntimeReceipt, want []string, cache string, javaPin []byte) {
	t.Helper()
	got := []string{}
	for _, r := range rs {
		got = append(got, r.ID)
		if r.Status != "passed" {
			t.Fatalf("runtime not verified: %+v", r)
		}
		switch r.ID {
		case "docker":
			if r.Identity != integrationPilotImage+" "+integrationPilotPlatform || r.SHA256 != strings.Split(integrationPilotImage, "sha256:")[1] {
				t.Fatalf("wrong OCI closure: %+v", r)
			}
		case "go", "node":
			tool, e := exec.LookPath(r.ID)
			if e != nil {
				t.Fatal(e)
			}
			arg := "version"
			if r.ID == "node" {
				arg = "--version"
			}
			wantVersion := strings.TrimSpace(integrationExec(t, 10*time.Second, tool, arg))
			if r.Identity != wantVersion || r.SHA256 != integrationFileSHA(t, tool) || r.SHA256 != integrationFileSHA(t, r.Path) {
				t.Fatalf("wrong native executable identity: %+v", r)
			}
		case "java":
			key := "JAVA_RUNTIME_" + strings.ToUpper(runtime.GOOS) + "_" + strings.ToUpper(runtime.GOARCH) + "_SHA256="
			archive := ""
			for _, line := range strings.Split(string(javaPin), "\n") {
				if strings.HasPrefix(line, key) {
					archive = strings.TrimPrefix(line, key)
				}
			}
			closure, e := runtimeclosure.JavaClosureDigest(r.Path)
			if e != nil {
				t.Fatal(e)
			}
			if archive == "" || r.Identity != runtimeclosure.RequiredJavaRelease || r.ArchiveSHA != archive || r.ClosureSHA != closure || r.SHA256 != integrationFileSHA(t, r.Path) {
				t.Fatalf("wrong provisioned Java closure: %+v", r)
			}
		case "tlc":
			if r.Identity != "v1.7.4" || r.SHA256 != "936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88" || r.SHA256 != integrationFileSHA(t, r.Path) {
				t.Fatalf("wrong pinned TLC jar: %+v", r)
			}
		default:
			t.Fatalf("unexpected runtime %q", r.ID)
		}
		if r.ID == "java" || r.ID == "tlc" {
			relative, e := filepath.Rel(cache, r.Path)
			if e != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				t.Fatalf("formal closure did not provision into fresh private cache: %+v", r)
			}
		}
	}
	sort.Strings(got)
	want = append([]string(nil), want...)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("runtime union got=%v want=%v", got, want)
	}
}

func integrationAssertExecutedClosure(t *testing.T, root string, rs []integrationRuntimeReceipt) {
	t.Helper()
	b, e := os.ReadFile(filepath.Join(root, "executed-closure.json"))
	if e != nil {
		t.Fatalf("candidate formal test never executed closure: %v", e)
	}
	var actual map[string]string
	if e = json.Unmarshal(b, &actual); e != nil {
		t.Fatal(e)
	}
	for _, r := range rs {
		if r.ID == "java" || r.ID == "tlc" {
			if actual[r.ID] != r.Path || actual[r.ID+"_sha256"] != r.SHA256 {
				t.Fatalf("reported closure differs from actual native invocation: %+v actual=%v", r, actual)
			}
		}
	}
}

func integrationFormalGo(marker, closureMarker string) string {
	return fmt.Sprintf(`//go:build machinery_integration

package sample
import("context";"crypto/sha256";"encoding/json";"fmt";"os";"os/exec";"path/filepath";"strings";"testing";"time")
func TestPilot(t *testing.T) {
 java,jar:=os.Getenv("MACHINERY_INTEGRATION_JAVA"),os.Getenv("MACHINERY_INTEGRATION_TLC_JAR")
 if java==""||jar==""{t.Fatal("candidate did not provision formal closure")}
 actual:=map[string]string{"java":java,"tlc":jar};for _,id:=range []string{"java","tlc"}{b,e:=os.ReadFile(actual[id]);if e!=nil{t.Fatal(e)};actual[id+"_sha256"]=fmt.Sprintf("%%x",sha256.Sum256(b))}
 work:=t.TempDir();model:="---- MODULE Pilot ----\nEXTENDS Integers\nVARIABLE x\nInit == x = 0\nNext == x' = 1 - x\nSpec == Init /\\ [][Next]_x\nSafe == x \\in {0,1}\nBroken == x = 0\n====\n"
 if e:=os.WriteFile(filepath.Join(work,"Pilot.tla"),[]byte(model),0600);e!=nil{t.Fatal(e)}
 for _,invariant:=range []string{"Safe","Broken"}{
  if e:=os.WriteFile(filepath.Join(work,"Pilot.cfg"),[]byte("SPECIFICATION Spec\nINVARIANT "+invariant+"\n"),0600);e!=nil{t.Fatal(e)}
  ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second)
  cmd:=exec.CommandContext(ctx,java,"-XX:+UseParallelGC","-cp",jar,"tlc2.TLC","-cleanup","-config","Pilot.cfg","Pilot.tla");cmd.Dir=work
  out,e:=cmd.CombinedOutput();cancel()
  if invariant=="Safe"&&(e!=nil||!strings.Contains(string(out),"No error has been found")){t.Fatalf("safe actual TLC %%v %%s",e,out)}
  if invariant=="Broken"&&(e==nil||!strings.Contains(string(out),"Invariant Broken is violated")){t.Fatalf("unsafe actual TLC %%v %%s",e,out)}
 }
 b,e:=json.Marshal(actual);if e!=nil{t.Fatal(e)};if e=os.WriteFile(%q,b,0600);e!=nil{t.Fatal(e)}
 if e=os.WriteFile(%q,[]byte("executed"),0600);e!=nil{t.Fatal(e)}
}
`, closureMarker, marker)
}

func integrationProcessGo(marker, pidMarker, scenario string) string {
	return fmt.Sprintf(`//go:build machinery_integration

package sample
import("fmt";"os";"os/exec";"path/filepath";"testing";"time")
func TestPilot(t *testing.T) {
 script:=filepath.Join(filepath.Dir(%q),"owned-descendant.sh")
 if e:=os.WriteFile(script,[]byte("#!/bin/sh\nwhile :; do /bin/sleep 1; done\n"),0700);e!=nil{t.Fatal(e)}
 child:=exec.Command("/bin/sh",script);if e:=child.Start();e!=nil{t.Fatal(e)}
 if e:=os.WriteFile(%q,[]byte(fmt.Sprintf("%%d\n%%s",child.Process.Pid,script)),0600);e!=nil{t.Fatal(e)}
 if e:=os.WriteFile(%q,[]byte("executed"),0600);e!=nil{t.Fatal(e)}
 switch %q {case "process-failure":t.Fatal("intentional assertion after host descendant start");case "process-timeout","process-cancellation":time.Sleep(time.Minute)}
 // The lane, not a deferred fixture cleanup, must terminate this descendant.
}
`, pidMarker, pidMarker, marker, scenario)
}

func integrationReadPID(t *testing.T, path string) (int, string) {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatalf("host descendant never started: %v", e)
	}
	parts := strings.SplitN(string(b), "\n", 2)
	if len(parts) != 2 {
		t.Fatal("malformed owned PID receipt")
	}
	pid, e := strconv.Atoi(parts[0])
	if e != nil || pid < 2 || !filepath.IsAbs(parts[1]) {
		t.Fatal("invalid owned PID receipt")
	}
	return pid, parts[1]
}
func integrationHostAlive(t *testing.T, pid int, script string) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	b, e := exec.CommandContext(ctx, "ps", "-p", strconv.Itoa(pid), "-o", "stat=", "-o", "command=").CombinedOutput()
	if e != nil && len(bytes.TrimSpace(b)) == 0 {
		return false
	}
	if e != nil {
		t.Fatalf("cannot inspect exact owned host PID: %v %s", e, b)
	}
	state := strings.Fields(string(b))
	if len(state) == 0 || strings.HasPrefix(state[0], "Z") {
		return false
	}
	if !strings.Contains(string(b), script) {
		t.Fatalf("refuse PID reuse/foreign process: %s", b)
	}
	return true
}
func integrationHostCleanup(t *testing.T, path string) {
	t.Helper()
	if _, e := os.Stat(path); os.IsNotExist(e) {
		return
	}
	pid, script := integrationReadPID(t, path)
	if !integrationHostAlive(t, pid, script) {
		return
	}
	p, e := os.FindProcess(pid)
	if e != nil {
		t.Error(e)
		return
	}
	if e = p.Kill(); e != nil {
		t.Errorf("owned emergency host cleanup failed: %v", e)
	}
}

func TestIntegrationLaneHostDescendantControl(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "real-owned-child.sh")
	integrationWrite(t, script, "#!/bin/sh\nwhile :; do /bin/sleep 1; done\n")
	child := exec.Command("/bin/sh", script)
	if e := child.Start(); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })
	if !integrationHostAlive(t, child.Process.Pid, script) {
		t.Fatal("real host-descendant sensitivity control was not live")
	}
	if e := child.Process.Kill(); e != nil {
		t.Fatal(e)
	}
	_ = child.Wait()
	if integrationHostAlive(t, child.Process.Pid, script) {
		t.Fatal("exact killed host descendant remained alive")
	}
	fixture := t.TempDir()
	pidMarker := filepath.Join(fixture, "created-pid")
	t.Cleanup(func() { integrationHostCleanup(t, pidMarker) })
	integrationWrite(t, filepath.Join(fixture, "go.mod"), "module lane.example/hostcontrol\n\ngo 1.27.0\n")
	integrationWrite(t, filepath.Join(fixture, "pilot_integration_test.go"), integrationProcessGo(filepath.Join(fixture, "executed"), pidMarker, "process-success"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-count=1", "-tags", "machinery_integration", ".")
	cmd.Dir = fixture
	b, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("actual descendant fixture failed: %v %s", e, b)
	}
	pid, ownedScript := integrationReadPID(t, pidMarker)
	if !integrationHostAlive(t, pid, ownedScript) {
		t.Fatal("sensitivity control did not leave observable real host descendant after native parent exit")
	}
	integrationHostCleanup(t, pidMarker)
}

func TestIntegrationLaneNodeEventControls(t *testing.T) {
	for _, tc := range []struct {
		name, body     string
		starts, passes int
		timeout        bool
	}{
		{"duplicate", "test('native pilot',()=>{});test('native pilot',()=>{});", 2, 2, false},
		{"missing-terminal", "test('native pilot',async()=>{setInterval(()=>{},1000);await new Promise(()=>{})});", 0, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "actual.test.mjs")
			integrationWrite(t, path, "import test from 'node:test';"+tc.body)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "node", "--test", "--test-reporter=tap", path)
			out, _, e := processcontrol.RunCapturedStreamLimits(ctx, cmd, 1<<20, 1<<20)
			if tc.timeout {
				if ctx.Err() == nil || e == nil {
					t.Fatalf("missing-terminal challenge terminated normally: %v %s", e, out)
				}
			} else if e != nil {
				t.Fatalf("duplicate native control failed: %v %s", e, out)
			}
			if strings.Count(out, "# Subtest: native pilot\n") != tc.starts || strings.Count(out, " - native pilot\n") != tc.passes {
				t.Fatalf("native control did not produce intended event shape: %s", out)
			}
		})
	}
}
func integrationReadOwnedCID(t *testing.T, body []byte) (string, string) {
	t.Helper()
	parts := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(parts) != 2 || len(parts[0]) != 64 || parts[1] == "" {
		t.Fatalf("invalid owned-resource receipt %q", body)
	}
	for _, c := range parts[0] {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Fatal("non-hex container ID")
		}
	}
	return parts[0], parts[1]
}

func integrationEmergencyCleanup(t *testing.T, marker string) {
	t.Helper()
	body, err := os.ReadFile(marker)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Errorf("read owned cleanup receipt: %v", err)
		return
	}
	id, runID := integrationReadOwnedCID(t, body)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	b, err := exec.CommandContext(ctx, "docker", "inspect", "--format", `{{index .Config.Labels "dev.machinery.integration-run"}}`, id).CombinedOutput()
	if err != nil && bytes.Contains(bytes.ToLower(b), []byte("no such object")) {
		return
	}
	if err != nil || strings.TrimSpace(string(b)) != runID {
		t.Errorf("refuse emergency cleanup without exact ownership: %s %v %s", id, err, b)
		return
	}
	b, err = exec.CommandContext(ctx, "docker", "rm", "-f", id).CombinedOutput()
	if err != nil {
		t.Errorf("emergency cleanup failed for owned %s: %v %s", id, err, b)
	}
}
