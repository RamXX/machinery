package formal

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/alloy"
)

// The fixture receipt was produced by `alloy exec` on the go-crm Policy.als
// generated from the FAITHFUL annotation (the invariants as originally
// written), so it carries both failure modes: the teamless-Manager
// counterexample on CapableWritesOwn and the outsider handoff on
// ReassignRetainsAuthority.
func loadReceipt(t *testing.T) alloyReceipt {
	t.Helper()
	raw, err := os.ReadFile("testdata/policy-receipt.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := decodeAlloyReceipt(raw)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestVerdicts(t *testing.T) {
	r := loadReceipt(t)
	commands := []alloy.Command{
		{Kind: "run", Name: "SomeWorld"},
		{Kind: "check", Name: "WriteImpliesRead"},
		{Kind: "check", Name: "CapableWritesOwn"},
		{Kind: "check", Name: "ReassignRetainsAuthority"},
		{Kind: "run", Name: "Possible_Rep_update"},
	}
	selected := alloyReceipt{Commands: map[string]alloyCommandResult{}}
	for _, command := range commands {
		selected.Commands[command.Name] = r.Commands[command.Name]
	}
	vs, err := verdicts(selected, commands, func(name string) string { return "counterexample: rendered-" + name })
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"SomeWorld":                true,  // run, SAT: world exists
		"WriteImpliesRead":         true,  // check, UNSAT: holds
		"CapableWritesOwn":         false, // check, SAT: the teamless Manager
		"ReassignRetainsAuthority": false, // check, SAT: the outsider handoff
		"Possible_Rep_update":      true,  // run, SAT: grant exercisable
	}
	for _, v := range vs {
		if v.Pass != want[v.Command.Name] {
			t.Errorf("%s: pass = %v, want %v", v.Command.Name, v.Pass, want[v.Command.Name])
		}
	}
	// failing checks carry the rendered counterexample
	for _, v := range vs {
		if v.Command.Name == "CapableWritesOwn" && v.Detail != "counterexample: rendered-CapableWritesOwn" {
			t.Errorf("CapableWritesOwn detail = %q", v.Detail)
		}
	}
}

func TestVerdictsMissingCommand(t *testing.T) {
	r := alloyReceipt{Commands: map[string]alloyCommandResult{}}
	_, err := verdicts(r, []alloy.Command{{Kind: "check", Name: "NotThere"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "no result for command") {
		t.Errorf("want missing-command error, got %v", err)
	}
}

func TestVerdictFailedRunDetail(t *testing.T) {
	// a run with no solution is a vacuity failure, with an explanatory detail
	r := alloyReceipt{Commands: map[string]alloyCommandResult{
		"Possible_X_read": {Name: "Possible_X_read", Type: "run"},
	}}
	vs, err := verdicts(r, []alloy.Command{{Kind: "run", Name: "Possible_X_read"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if vs[0].Pass || !strings.Contains(vs[0].Detail, "no instance within scope") {
		t.Errorf("verdict = %+v; want failed run with vacuity detail", vs[0])
	}
}

func TestAlloyReceiptAuthoritativeLoader(t *testing.T) {
	commands := []alloy.Command{{Kind: "check", Name: "Safe"}}
	valid := `{"commands":{"Safe":{"bitwidth":4,"expects":-1,"maxprefix":-1,"maxseq":-1,"minprefix":-1,"name":"Safe","overall":6,"source":"check Safe","type":"check"}},"coreMinimization":2,"inferPartialInstance":true,"repeat":1,"sigs":{},"solver":"sat4j","symmetry":20,"timestamp":1,"unrolls":-1}`
	tests := []struct {
		name, raw, want string
	}{
		{"typo key", strings.Replace(valid, `"commands"`, `"commmands"`, 1), "unknown field"},
		{"case-aliased key", strings.Replace(valid, `"commands"`, `"Commands"`, 1), "unknown field"},
		{"duplicate key", strings.Replace(valid, `"commands":`, `"commands":{},"commands":`, 1), "duplicate"},
		{"malformed extra command", strings.Replace(valid, `"commands":{`, `"commands":{"Extra":{"name":"Extra","type":"check"},`, 1), "missing required field"},
		{"missing command", strings.Replace(valid, `"Safe":{"bitwidth":4,"expects":-1,"maxprefix":-1,"maxseq":-1,"minprefix":-1,"name":"Safe","overall":6,"source":"check Safe","type":"check"}`, ``, 1), "no result"},
		{"wrong identity", strings.Replace(valid, `"name":"Safe"`, `"name":"Typo"`, 1), "contains identity"},
		{"wrong type", strings.Replace(valid, `"type":"check"`, `"type":"run"`, 1), "has type"},
		{"unknown command field", strings.Replace(valid, `"overall":6`, `"overal":6`, 1), "unknown field"},
		{"null solution", strings.Replace(valid, `"source":`, `"solution":null,"source":`, 1), "not null"},
		{"empty solution", strings.Replace(valid, `"source":`, `"solution":[],"source":`, 1), "want exactly 1"},
		{"null required field", strings.Replace(valid, `"overall":6`, `"overall":null`, 1), "must not be null"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := loadAlloyReceipt([]byte(tc.raw), commands); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want error containing %q", err, tc.want)
			}
		})
	}
	if _, err := loadAlloyReceipt([]byte(valid), commands); err != nil {
		t.Fatalf("valid authoritative receipt rejected: %v", err)
	}
}

func TestAlloyReceiptInventoryDiagnosticsAreDeterministic(t *testing.T) {
	tests := []struct {
		name     string
		receipt  alloyReceipt
		commands []alloy.Command
		want     string
	}{
		{
			name: "multiple unexpected",
			receipt: alloyReceipt{Commands: map[string]alloyCommandResult{
				"ZExtra": {Name: "ZExtra", Type: "check"},
				"AExtra": {Name: "AExtra", Type: "check"},
			}},
			commands: []alloy.Command{{Name: "Safe", Kind: "check"}},
			want:     "unexpected command AExtra",
		},
		{
			name:     "multiple missing",
			receipt:  alloyReceipt{Commands: map[string]alloyCommandResult{}},
			commands: []alloy.Command{{Name: "ZSafe", Kind: "check"}, {Name: "ASafe", Kind: "run"}},
			want:     "no result for command ASafe",
		},
		{
			name: "multiple wrong",
			receipt: alloyReceipt{Commands: map[string]alloyCommandResult{
				"ZSafe": {Name: "WrongZ", Type: "run"},
				"ASafe": {Name: "WrongA", Type: "check"},
			}},
			commands: []alloy.Command{{Name: "ZSafe", Kind: "check"}, {Name: "ASafe", Kind: "run"}},
			want:     "command key ASafe contains identity WrongA",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < 100; i++ {
				err := validateAlloyReceiptInventory(tc.receipt, tc.commands)
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("iteration %d: got %v, want %q", i, err, tc.want)
				}
			}
		})
	}
}

// The text fixture is the real solver output for the go-crm faithful policy's
// CapableWritesOwn counterexample: six users, a teamless Manager (User$5),
// and Record$0 owned by that Manager.
func TestRenderSolutionText(t *testing.T) {
	raw, err := os.ReadFile("testdata/policy-solution.txt")
	if err != nil {
		t.Fatal(err)
	}
	got := renderSolutionText(string(raw))
	for _, want := range []string{
		"counterexample: ",
		"User$5{role=Manager$0, team=(none)}",
		"Record$0{owner=User$5}",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered %q missing %q", got, want)
		}
	}
}

func TestRenderSolutionTextEmpty(t *testing.T) {
	if got := renderSolutionText("no relations here"); got != "" {
		t.Errorf("want empty render, got %q", got)
	}
}

func TestRenderSolutionTextCanonicalizesSolverIterationOrder(t *testing.T) {
	first := strings.Join([]string{
		"$X={User$1}",
		"this/User={User$1, User$0}",
		"this/User<:role={User$1->Manager$0, User$0->Member$0}",
		"this/User<:team={User$0->Team$0}",
	}, "\n")
	shuffled := strings.Join([]string{
		"this/User<:team={User$0->Team$0}",
		"this/User<:role={User$0->Member$0, User$1->Manager$0}",
		"this/User={User$0, User$1}",
		"$X={User$1}",
		"$X={User$1}", // identical solver detail is de-duplicated
	}, "\n")
	want := renderSolutionText(first)
	if got := renderSolutionText(shuffled); got != want {
		t.Fatalf("equivalent solver solutions rendered differently:\nwant %q\n got %q", want, got)
	}
}

func TestAlloyJarPathOverride(t *testing.T) {
	t.Setenv("ALLOY_TOOLS_JAR", "/tmp/custom.jar")
	if got, err := alloyJarPath(); err != nil || got != "/tmp/custom.jar" {
		t.Error("env override ignored")
	}
	t.Setenv("ALLOY_TOOLS_JAR", "")
	got, err := alloyJarPath()
	if err != nil || !strings.Contains(got, "alloy-dist-"+alloyVersion) {
		t.Errorf("default path = %q, %v", got, err)
	}
}

func TestRunAlloyRedactsPrivateWorkdirFromEngineFailure(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "alloy.jar")
	if err := os.WriteFile(jar, []byte("verified"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha, _ := fileSHA256(jar)
	t.Setenv("ALLOY_TOOLS_JAR", jar)
	t.Setenv("ALLOY_TOOLS_JAR_SHA256", sha)
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJavaRuntime(t, filepath.Join(bin, "java"), supportedJavaScript("printf '%s\\n' \"$@\"\nexit 9\n"))
	t.Setenv("PATH", bin)
	als := filepath.Join(dir, "Policy.als")
	if err := os.WriteFile(als, []byte("module Policy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var want string
	for i := 0; i < 10; i++ {
		_, err := runAlloy(als, []alloy.Command{{Name: "x", Kind: "check"}})
		if err == nil || strings.Contains(err.Error(), "machinery-alloy") || !strings.Contains(err.Error(), "<alloy-workdir>") {
			t.Fatalf("private workdir leaked or stable placeholder absent: %v", err)
		}
		if i == 0 {
			want = err.Error()
		} else if err.Error() != want {
			t.Fatalf("run %d diagnostic changed:\nwant %s\n got %s", i, want, err)
		}
	}
}

func TestAlloySuccessDiagnosticsAreClosedAndRejectWarnings(t *testing.T) {
	commands := []alloy.Command{{Name: "SomeWorld", Kind: "run"}, {Name: "Safe", Kind: "check"}}
	canonical := "00. run SomeWorld 0\b\b\b\b 1/1 SAT\n01. check Safe 0 UNSAT\n"
	if err := validateAlloySuccessOutput(canonical, commands); err != nil {
		t.Fatalf("canonical command diagnostics rejected: %v", err)
	}
	if err := validateAlloySuccessOutput(canonical+"WARNING solver fallback\n", commands); err == nil {
		t.Fatal("exit-zero Alloy warning was discarded")
	}
	if err := validateAlloySuccessOutput("00. run SomeWorld 0 WARNING solver-fallback 1/1 SAT\n01. check Safe 0 UNSAT\n", commands); err == nil {
		t.Fatal("same-line Alloy warning was accepted as canonical progress")
	}
}

func TestRunAlloyRejectsExitZeroWarningWithValidReceipt(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "alloy.jar")
	if err := os.WriteFile(jar, []byte("verified"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha, _ := fileSHA256(jar)
	t.Setenv("ALLOY_TOOLS_JAR", jar)
	t.Setenv("ALLOY_TOOLS_JAR_SHA256", sha)
	javaPath := filepath.Join(dir, "runtime", "bin", "java")
	receipt := `{"commands":{"x":{"bitwidth":4,"expects":0,"maxprefix":0,"maxseq":0,"minprefix":0,"name":"x","overall":0,"solution":[{"duration":0,"incremental":false,"instances":[],"localtime":"","timezone":"","utctime":0}],"source":"check x","type":"check"}},"coreMinimization":null,"inferPartialInstance":null,"repeat":null,"sigs":null,"solver":null,"symmetry":null,"timestamp":null,"unrolls":null}`
	engine := "out=''\nwhile [ $# -gt 0 ]; do if [ \"$1\" = '-o' ]; then out=$2; shift 2; else shift; fi; done\nprintf '%s' '" + receipt + "' > \"$out/receipt.json\"\necho '00. check x 0 UNSAT'\necho 'WARNING solver fallback'\nexit 0\n"
	writeJavaRuntime(t, javaPath, supportedJavaScript(engine))
	t.Setenv("MACHINERY_JAVA", javaPath)
	als := filepath.Join(dir, "Policy.als")
	if err := os.WriteFile(als, []byte("check x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runAlloy(als, []alloy.Command{{Name: "x", Kind: "check"}}); err == nil || !strings.Contains(err.Error(), "unexpected success diagnostics") {
		t.Fatalf("exit-zero Alloy warning was accepted: %v", err)
	}
}

func TestRunAlloyRejectsCanonicalSATWithoutSolutionArtifact(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "alloy.jar")
	if err := os.WriteFile(jar, []byte("verified"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha, _ := fileSHA256(jar)
	t.Setenv("ALLOY_TOOLS_JAR", jar)
	t.Setenv("ALLOY_TOOLS_JAR_SHA256", sha)
	javaPath := filepath.Join(dir, "runtime", "bin", "java")
	receipt := `{"commands":{"x":{"bitwidth":4,"expects":0,"maxprefix":0,"maxseq":0,"minprefix":0,"name":"x","overall":0,"solution":[{"duration":0,"incremental":false,"instances":[{"values":{}}],"localtime":"","timezone":"","utctime":0}],"source":"check x","type":"check"}},"coreMinimization":null,"inferPartialInstance":null,"repeat":null,"sigs":null,"solver":null,"symmetry":null,"timestamp":null,"unrolls":null}`
	engine := "out=''\nwhile [ $# -gt 0 ]; do if [ \"$1\" = '-o' ]; then out=$2; shift 2; else shift; fi; done\nprintf '%s' '" + receipt + "' > \"$out/receipt.json\"\necho '00. check x 0 1/1 SAT'\nexit 0\n"
	writeJavaRuntime(t, javaPath, supportedJavaScript(engine))
	t.Setenv("MACHINERY_JAVA", javaPath)
	als := filepath.Join(dir, "Policy.als")
	if err := os.WriteFile(als, []byte("check x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runAlloy(als, []alloy.Command{{Name: "x", Kind: "check"}}); err == nil || !strings.Contains(err.Error(), "solution inventory") {
		t.Fatalf("canonical SAT without a solution artifact was accepted: %v", err)
	}
}

func TestRunAlloyRejectsStdoutReceiptOutcomeContradictions(t *testing.T) {
	const rootTail = `,"coreMinimization":null,"inferPartialInstance":null,"repeat":null,"sigs":null,"solver":null,"symmetry":null,"timestamp":null,"unrolls":null}`
	base := `{"commands":{"x":{"bitwidth":4,"expects":0,"maxprefix":0,"maxseq":0,"minprefix":0,"name":"x","overall":0,`
	cases := []struct {
		name, outcome, receiptTail string
	}{
		{"stdout SAT receipt has no solution", "1/1 SAT", `"source":"check x","type":"check"}}` + rootTail},
		{"stdout UNSAT receipt has instances", "UNSAT", `"solution":[{"duration":0,"incremental":false,"instances":[{"values":{}}],"localtime":"","timezone":"","utctime":0}],"source":"check x","type":"check"}}` + rootTail},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			jar := filepath.Join(dir, "alloy.jar")
			if err := os.WriteFile(jar, []byte("verified"), 0o644); err != nil {
				t.Fatal(err)
			}
			sha, _ := fileSHA256(jar)
			t.Setenv("ALLOY_TOOLS_JAR", jar)
			t.Setenv("ALLOY_TOOLS_JAR_SHA256", sha)
			javaPath := filepath.Join(dir, "runtime", "bin", "java")
			receipt := base + tc.receiptTail
			engine := "out=''\nwhile [ $# -gt 0 ]; do if [ \"$1\" = '-o' ]; then out=$2; shift 2; else shift; fi; done\nprintf '%s' '" + receipt + "' > \"$out/receipt.json\"\necho '00. check x 0 " + tc.outcome + "'\nexit 0\n"
			writeJavaRuntime(t, javaPath, supportedJavaScript(engine))
			t.Setenv("MACHINERY_JAVA", javaPath)
			als := filepath.Join(dir, "Policy.als")
			if err := os.WriteFile(als, []byte("check x\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := runAlloy(als, []alloy.Command{{Name: "x", Kind: "check"}}); err == nil || !strings.Contains(err.Error(), "contradicts engine results") {
				t.Fatalf("stdout/receipt contradiction was accepted: %v", err)
			}
		})
	}
}

func TestAlloySolutionArtifactsAreRegularStableAndExact(t *testing.T) {
	sat := alloyCommandResult{
		Name: "x", Type: "check",
		Solution: []alloySolution{{Instances: []alloyInstance{{}}}},
	}
	receipt := alloyReceipt{Commands: map[string]alloyCommandResult{"x": sat}}
	commands := []alloy.Command{{Name: "x", Kind: "check"}}
	name := "x-solution-0.txt"
	body := []byte("---Trace---\n------State 0 (loop)-------\nthis/User={User$0}\nthis/User<:role={User$0->Member$0}\n\n")

	t.Run("symlink", func(t *testing.T) {
		dir := t.TempDir()
		outside := filepath.Join(t.TempDir(), "outside.txt")
		if err := os.WriteFile(outside, body, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(dir, name)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, err := loadAlloySolutionDetails(dir, receipt, commands); err == nil || !strings.Contains(err.Error(), "regular non-symlink") {
			t.Fatalf("symlinked solution was accepted: %v", err)
		}
	})

	t.Run("replacement after inspection", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
		prior := formalAfterAlloySolutionInspect
		formalAfterAlloySolutionInspect = func(inspected string) {
			if inspected != name {
				return
			}
			replacement := filepath.Join(dir, "replacement")
			if err := os.WriteFile(replacement, []byte("foreign\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(replacement, path); err != nil {
				t.Fatal(err)
			}
		}
		defer func() { formalAfterAlloySolutionInspect = prior }()
		if _, err := loadAlloySolutionDetails(dir, receipt, commands); err == nil || !strings.Contains(err.Error(), "changed identity") {
			t.Fatalf("replaced solution was accepted: %v", err)
		}
	})

	t.Run("unexpected inventory", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "foreign-solution-0.txt"), body, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadAlloySolutionDetails(dir, receipt, commands); err == nil || !strings.Contains(err.Error(), "want exactly") {
			t.Fatalf("unexpected solution inventory was accepted: %v", err)
		}
	})

	t.Run("unrelated extra artifact", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "engine.log"), []byte("noise\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadAlloySolutionDetails(dir, receipt, commands); err == nil || !strings.Contains(err.Error(), "unexpected Alloy output artifact") {
			t.Fatalf("unrelated output artifact was accepted: %v", err)
		}
	})

	t.Run("empty or noncanonical body", func(t *testing.T) {
		for _, body := range [][]byte{nil, []byte("arbitrary output\n"), []byte("this/User={} ")} {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := loadAlloySolutionDetails(dir, receipt, commands); err == nil || !strings.Contains(err.Error(), "not canonical") {
				t.Fatalf("noncanonical solution body passed: %q err=%v", body, err)
			}
		}
	})

	t.Run("same-size content mutation", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
		prior := formalAfterAlloySolutionRead
		formalAfterAlloySolutionRead = func(read string) {
			if read == name {
				mutated := bytes.Replace(body, []byte("User$0"), []byte("User$1"), 1)
				if err := os.WriteFile(path, mutated, 0o600); err != nil {
					t.Fatal(err)
				}
			}
		}
		defer func() { formalAfterAlloySolutionRead = prior }()
		if _, err := loadAlloySolutionDetails(dir, receipt, commands); err == nil || !strings.Contains(err.Error(), "changed content") {
			t.Fatalf("same-size solution mutation passed: %v", err)
		}
	})
}

func TestRunAlloySeparatesToolSnapshotFromEngineOutput(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "alloy.jar")
	if err := os.WriteFile(jar, []byte("verified"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha, _ := fileSHA256(jar)
	t.Setenv("ALLOY_TOOLS_JAR", jar)
	t.Setenv("ALLOY_TOOLS_JAR_SHA256", sha)
	javaPath := filepath.Join(dir, "runtime", "bin", "java")
	receipt := `{"commands":{},"coreMinimization":null,"inferPartialInstance":null,"repeat":null,"sigs":null,"solver":null,"symmetry":null,"timestamp":null,"unrolls":null}`
	engine := "jar=$2\nout=''\nwhile [ $# -gt 0 ]; do if [ \"$1\" = '-o' ]; then out=$2; shift 2; else shift; fi; done\nif [ \"$(dirname \"$jar\")\" = \"$out\" ]; then echo 'tool/output alias'; exit 23; fi\nprintf '%s' '" + receipt + "' > \"$out/receipt.json\"\n"
	writeJavaRuntime(t, javaPath, supportedJavaScript(engine))
	t.Setenv("MACHINERY_JAVA", javaPath)
	als := filepath.Join(dir, "Policy.als")
	if err := os.WriteFile(als, []byte("check x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runAlloy(als, nil); err != nil {
		t.Fatalf("separate tool/output roots rejected: %v", err)
	}
}
