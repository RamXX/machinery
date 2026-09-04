package formal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunBoundedProcessTruncationIsAlwaysAnError(t *testing.T) {
	for _, stream := range []string{"oversize-stdout", "oversize-stderr", "oversize-combined"} {
		t.Run(stream, func(t *testing.T) {
			cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestFormalProcessHelper", "--", stream)
			cmd.Env = append(os.Environ(), "MACHINERY_FORMAL_PROCESS_HELPER=1")
			out, err := runBoundedProcess(context.Background(), cmd, time.Minute)
			wantErr := fmt.Sprintf("process combined output exceeded %d-byte limit", formalOutputLimit)
			if err == nil || err.Error() != wantErr {
				t.Fatalf("bounded process overflow error = %v, want %q", err, wantErr)
			}
			wantSuffix := fmt.Sprintf("\n[output truncated at %d bytes]\n", formalOutputLimit)
			if !strings.HasSuffix(out, wantSuffix) || len(out) != formalOutputLimit+len(wantSuffix) {
				t.Fatalf("bounded output length/suffix = %d/%q", len(out), out[len(out)-min(len(out), 80):])
			}
		})
	}
}

func TestRunBoundedProcessTimesOut(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestFormalProcessHelper", "--", "sleep")
	cmd.Env = append(os.Environ(), "MACHINERY_FORMAL_PROCESS_HELPER=1")
	_, err := runBoundedProcess(ctx, cmd, 50*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "process timed out after 50ms") || !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("timeout was not explicit: %v", err)
	}
}

func TestRunTLCRejectsCanonicalSuccessFollowedByOutputOverflow(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "tla2tools.jar")
	if err := os.WriteFile(jar, []byte("verified"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha, err := fileSHA256(jar)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TLA_TOOLS_JAR", jar)
	t.Setenv("TLA_TOOLS_JAR_SHA256", sha)
	javaPath := filepath.Join(dir, "runtime", "bin", "java")
	engine := "echo 'No error has been found'\nprintf '%s' '" + strings.Repeat("x", formalOutputLimit+4096) + "'\nexit 0\n"
	writeJavaRuntime(t, javaPath, supportedJavaScript(engine))
	t.Setenv("MACHINERY_JAVA", javaPath)

	out, err := runTLC(filepath.Join(dir, "Toy.tla"), filepath.Join(dir, "Toy.cfg"))
	want := fmt.Sprintf("process combined output exceeded %d-byte limit", formalOutputLimit)
	if err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(out, "[output truncated") {
		t.Fatalf("TLC success prefix hid overflow: output tail=%q err=%v", out[len(out)-min(len(out), 100):], err)
	}
}

func TestOpenFormalJavaRejectsProbeOutputOverflow(t *testing.T) {
	dir := t.TempDir()
	javaPath := filepath.Join(dir, "runtime", "bin", "java")
	probe := "#!/bin/sh\njava_home=$(CDPATH= cd -- \"$(dirname -- \"$0\")/..\" && pwd -P)\n" +
		"echo \"    java.home = $java_home\" >&2\n" +
		"echo '    java.runtime.version = 21.0.12.1+1-LTS' >&2\n" +
		"echo '    java.vendor = Eclipse Adoptium' >&2\n" +
		"echo '    java.version = 21.0.12.1' >&2\n" +
		"echo '    java.vm.name = OpenJDK 64-Bit Server VM' >&2\n" +
		"printf '%s' '" + strings.Repeat("x", formalOutputLimit+4096) + "' >&2\nexit 0\n"
	writeJavaRuntime(t, javaPath, probe)
	t.Setenv("MACHINERY_JAVA", javaPath)

	java, err := openFormalJava(dir)
	if java != nil {
		_ = java.Close()
	}
	want := fmt.Sprintf("process combined output exceeded %d-byte limit", formalOutputLimit)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Java probe overflow was accepted: %v", err)
	}
}

func TestRunAlloyRejectsOutputOverflow(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "alloy.jar")
	if err := os.WriteFile(jar, []byte("verified"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha, err := fileSHA256(jar)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("ALLOY_TOOLS_JAR", jar)
	t.Setenv("ALLOY_TOOLS_JAR_SHA256", sha)
	javaPath := filepath.Join(dir, "runtime", "bin", "java")
	receipt := `{"commands":{},"coreMinimization":null,"inferPartialInstance":null,"repeat":null,"sigs":null,"solver":null,"symmetry":null,"timestamp":null,"unrolls":null}`
	engine := "out=''\nwhile [ $# -gt 0 ]; do if [ \"$1\" = '-o' ]; then out=$2; shift 2; else shift; fi; done\nprintf '%s' '" + receipt + "' > \"$out/receipt.json\"\nprintf '%s' '" + strings.Repeat("x", formalOutputLimit+4096) + "'\nexit 0\n"
	writeJavaRuntime(t, javaPath, supportedJavaScript(engine))
	t.Setenv("MACHINERY_JAVA", javaPath)
	als := filepath.Join(dir, "Policy.als")
	if err := os.WriteFile(als, []byte("check x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err = runAlloy(als, nil)
	want := fmt.Sprintf("process combined output exceeded %d-byte limit", formalOutputLimit)
	if err == nil || !strings.Contains(err.Error(), "alloy exec failed") || !strings.Contains(err.Error(), want) {
		t.Fatalf("Alloy output overflow was accepted: %v", err)
	}
}

func TestFormalProcessHelper(t *testing.T) {
	if os.Getenv("MACHINERY_FORMAL_PROCESS_HELPER") != "1" {
		return
	}
	switch os.Args[len(os.Args)-1] {
	case "oversize-stdout":
		_, _ = os.Stdout.Write([]byte(strings.Repeat("x", formalOutputLimit+4096)))
	case "oversize-stderr":
		_, _ = os.Stderr.Write([]byte(strings.Repeat("x", formalOutputLimit+4096)))
	case "oversize-combined":
		_, _ = os.Stdout.Write([]byte(strings.Repeat("x", formalOutputLimit/2+4096)))
		_, _ = os.Stderr.Write([]byte(strings.Repeat("y", formalOutputLimit/2+4096)))
	case "sleep":
		time.Sleep(30 * time.Second)
	}
	os.Exit(0)
}
