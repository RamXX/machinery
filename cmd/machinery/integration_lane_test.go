//go:build machinery_integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/formal"
	"github.com/RamXX/machinery/internal/processcontrol"
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
	for _, scenario := range []string{"success", "cold-cache", "missing-docker", "wrong-pin", "wrong-digest", "wrong-platform", "wrong-java-pin", "wrong-tlc-pin", "offline-formal", "leaked-resource", "failed-assertion", "timeout", "cancellation", "node-success", "node-missing", "node-zero", "node-skipped", "node-partial", "node-aggregate-only"} {
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
			t.Cleanup(func() { integrationEmergencyCleanup(t, cidMarker) })
			suite := map[string]any{"id": "fullpath", "lane": "required", "adapter": "go-json", "package": "./sample", "source_files": []string{"sample/pilot_integration_test.go"}, "tests": []string{"TestPilot"}, "runtimes": []string{"go", "docker"}, "timeout": "30s", "stdout_limit": 1048576, "stderr_limit": 1048576}
			body := integrationFullPathGo(marker, cidMarker, scenario)
			if scenario == "offline-formal" || scenario == "wrong-java-pin" || scenario == "wrong-tlc-pin" {
				suite["runtimes"] = []string{"go", "java", "tlc"}
				body = "//go:build machinery_integration\n\npackage sample\nimport \"testing\"\nfunc TestPilot(t *testing.T) { t.Fatal(\"invalid formal prerequisites must block this test\") }\n"
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
				}
				integrationWrite(t, filepath.Join(root, "sample/pilot.integration.test.mjs"), js)
			} else {
				integrationWrite(t, filepath.Join(root, "sample/pilot_integration_test.go"), body)
			}
			if scenario == "timeout" {
				suite["timeout"] = "5s"
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
			if scenario == "cancellation" {
				deadline := time.Now().Add(25 * time.Second)
				for time.Now().Before(deadline) {
					if _, err := os.Stat(cidMarker); err == nil {
						break
					}
					time.Sleep(20 * time.Millisecond)
				}
				if _, err := os.Stat(cidMarker); err != nil {
					_ = cmd.Process.Signal(os.Interrupt)
					_ = cmd.Wait()
					t.Fatalf("cancellation control never created real owned container: %v output=%s", err, output.String())
				}
				if err := cmd.Process.Signal(os.Interrupt); err != nil {
					_ = cmd.Wait()
					t.Fatal(err)
				}
			}
			err = cmd.Wait()
			wantPass := scenario == "success" || scenario == "cold-cache" || scenario == "node-success"
			if wantPass && err != nil {
				t.Fatalf("provisioned real runtime execution rejected: %v %s", err, output.String())
			}
			if !wantPass && err == nil {
				t.Fatalf("unsafe execution accepted: %s", output.String())
			}
			if !wantPass {
				diagnostic := map[string]string{"missing-docker": "docker", "wrong-pin": "pin", "wrong-digest": "pin", "wrong-platform": "platform", "wrong-java-pin": "pin", "wrong-tlc-pin": "pin", "offline-formal": "provision", "leaked-resource": "leak", "failed-assertion": "failed", "timeout": "timeout", "cancellation": "cancel", "node-missing": "node", "node-zero": "execution", "node-skipped": "skip", "node-partial": "execution", "node-aggregate-only": "execution"}[scenario]
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
			var receipt struct {
				Status   string                                  `json:"status"`
				Suites   []struct{ Started, Passed int }         `json:"suites"`
				Runtimes []struct{ ID, Identity, Status string } `json:"runtimes"`
				Cleanup  struct{ Status string }                 `json:"cleanup"`
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
				if len(receipt.Runtimes) == 0 {
					t.Fatal("runtime evidence absent")
				}
				for _, r := range receipt.Runtimes {
					if r.ID == "" || r.Identity == "" || r.Status != "passed" {
						t.Fatalf("runtime not identified/provisioned: %+v", r)
					}
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
			if entries, e := os.ReadDir(work); e == nil && len(entries) != 0 {
				t.Fatalf("owned temporary work leaked: %v", entries)
			} else if e != nil && !os.IsNotExist(e) {
				t.Fatal(e)
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
