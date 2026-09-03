package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

var errOutputSink = errors.New("injected output failure")

type failedOutputWriter struct{}

func (failedOutputWriter) Write([]byte) (int, error) { return 0, errOutputSink }

type shortOutputWriter struct{}

func (shortOutputWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return len(p) - 1, nil
}

func failCommandStdout(t *testing.T) (*bytes.Buffer, *[]int) {
	t.Helper()
	var stderr bytes.Buffer
	var codes []int
	priorCodes := capturedExitCodes
	capturedExitCodes = &codes
	stdoutW, stderrW = failedOutputWriter{}, &stderr
	t.Cleanup(func() {
		stdoutW, stderrW = os.Stdout, os.Stderr
		capturedExitCodes = priorCodes
	})
	return &stderr, &codes
}

func requireOutputFailure(t *testing.T, err error) {
	t.Helper()
	if err == nil || !errors.Is(err, errOutputSink) || !strings.Contains(err.Error(), "write machinery stdout") {
		t.Fatalf("command did not return the stdout failure: %v", err)
	}
}

func TestTrackedOutputRejectsShortWrite(t *testing.T) {
	output := trackOutput(shortOutputWriter{}, io.Discard)
	_, _ = output.stdout.Write([]byte("result"))
	if err := output.join(nil); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short output write was not retained: %v", err)
	}
}

func TestVersionReturnsOutputFailure(t *testing.T) {
	cmd := newVersionCmd()
	cmd.SetOut(failedOutputWriter{})
	requireOutputFailure(t, executeCapturedCommand(cmd))
}

func TestAttestReturnsOutputFailure(t *testing.T) {
	_, codes := failCommandStdout(t)
	cmd := newAttestCmd()
	cmd.SetArgs([]string{"--claims"})
	requireOutputFailure(t, executeCapturedCommand(cmd))
	if len(*codes) != 0 {
		t.Fatalf("successful attest attempted an early exit before output validation: %v", *codes)
	}
}

func TestLintSuccessReturnsOutputFailureInsteadOfExitingZero(t *testing.T) {
	_, codes := failCommandStdout(t)
	design := t.TempDir()
	machines := filepath.Join(design, "machines")
	if err := os.Mkdir(machines, 0o755); err != nil {
		t.Fatal(err)
	}
	machine := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	if err := os.WriteFile(filepath.Join(machines, "Toy.machine.json"), []byte(machine), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newLintCmd()
	cmd.SetArgs([]string{machines})
	requireOutputFailure(t, executeCapturedCommand(cmd))
	if len(*codes) != 0 {
		t.Fatalf("successful lint attempted an early exit before output validation: %v", *codes)
	}
}

func TestVerifyFormalSuccessReturnsOutputFailureInsteadOfExitingZero(t *testing.T) {
	_, codes := failCommandStdout(t)
	design := t.TempDir()
	machines := filepath.Join(design, "machines")
	if err := os.Mkdir(machines, 0o755); err != nil {
		t.Fatal(err)
	}
	machine := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	if err := os.WriteFile(filepath.Join(machines, "Toy.machine.json"), []byte(machine), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newVerifyFormalCmd()
	cmd.SetArgs([]string{"--gen-only", design})
	requireOutputFailure(t, executeCapturedCommand(cmd))
	if len(*codes) != 0 {
		t.Fatalf("successful verify-formal attempted an early exit before output validation: %v", *codes)
	}
	if _, err := os.Stat(filepath.Join(design, "formal", "Toy.tla")); err != nil {
		t.Fatalf("verify-formal did not reach its successful publication before the broken pipe: %v", err)
	}
}

func TestVerifyCheckersBufferedSuccessReturnsOutputFailure(t *testing.T) {
	design := setupVerifyDesign(t)
	stub := writeScript(t, "/bin/cp \""+design.evPath+"\" \"$1\"\n")
	registry := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{out}\"]\n")
	syncTestRuntimeClosure(t, design.dir, registry)
	_, codes := failCommandStdout(t)
	cmd := newVerifyCheckersCmd()
	cmd.SetArgs([]string{design.dir, "--registry", registry})
	requireOutputFailure(t, executeCapturedCommand(cmd))
	if len(*codes) != 0 {
		t.Fatalf("successful verify-checkers attempted an early exit before output validation: %v", *codes)
	}
}

func TestCheckGateOutputFailureIsBlocking(t *testing.T) {
	_, _ = failCommandStdout(t)
	cmd := newCheckCmd()
	cmd.SetArgs([]string{t.TempDir(), "--gate", "gl"})
	requireOutputFailure(t, executeCapturedCommand(cmd))
}

func TestMachineReadableIRDumpReturnsOutputFailure(t *testing.T) {
	_, _ = failCommandStdout(t)
	machine := filepath.Join(t.TempDir(), "Toy.machine.json")
	body := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	if err := os.WriteFile(machine, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newIRDumpCmd()
	cmd.SetArgs([]string{machine})
	requireOutputFailure(t, executeCapturedCommand(cmd))
}

func TestInstallAndUninstallReturnOutputFailureAfterMutation(t *testing.T) {
	home := t.TempDir()
	installCmd := newInstallCmd()
	installCmd.SetOut(failedOutputWriter{})
	installCmd.SetErr(io.Discard)
	installCmd.SetArgs([]string{"--from", "../..", "--home", home, "--copy"})
	requireOutputFailure(t, executeCapturedCommand(installCmd))
	installed := filepath.Join(home, "skills", "machinery", "SKILL.md")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("install did not commit before reporting the broken pipe: %v", err)
	}

	uninstallCmd := newUninstallCmd()
	uninstallCmd.SetOut(failedOutputWriter{})
	uninstallCmd.SetErr(io.Discard)
	uninstallCmd.SetArgs([]string{"--home", home})
	requireOutputFailure(t, executeCapturedCommand(uninstallCmd))
	if _, err := os.Lstat(installed); !os.IsNotExist(err) {
		t.Fatalf("uninstall did not commit before reporting the broken pipe: %v", err)
	}
}

func TestPreflightReturnsOutputFailure(t *testing.T) {
	_, _ = failCommandStdout(t)
	cmd := newPreflightCmd()
	err := executeCapturedCommand(cmd)
	if !errors.Is(err, errOutputSink) {
		t.Fatalf("preflight did not include the output failure: %v", err)
	}
}

func TestGeneratorCommandsReturnOutputFailureAfterPublishing(t *testing.T) {
	goCRM := t.TempDir()
	copyDirInto(t, "../../examples/go-crm/design", goCRM)
	fulfillment := t.TempDir()
	copyDirInto(t, "../../examples/fulfillment/design", fulfillment)
	tests := []struct {
		name      string
		command   func() *cobra.Command
		args      func(string) []string
		published string
	}{
		{
			name:    "tla",
			command: newTLACmd,
			args: func(out string) []string {
				return []string{filepath.Join(goCRM, "machines/Deal.machine.json"), out}
			},
			published: "Deal.tla",
		},
		{
			name:    "refine",
			command: newRefineCmd,
			args: func(out string) []string {
				return []string{filepath.Join(goCRM, "machines/Deal.machine.json"), filepath.Join(goCRM, "formal/Deal.semantics.yaml"), out}
			},
			published: "DealRefinement.tla",
		},
		{
			name:    "compose",
			command: newComposeCmd,
			args: func(out string) []string {
				return []string{filepath.Join(fulfillment, "formal/checkout.composition.yaml"), filepath.Join(fulfillment, "machines/FulfillmentSaga.machine.json"), out}
			},
			published: "Checkout.tla",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, codes := failCommandStdout(t)
			out := t.TempDir()
			cmd := test.command()
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(test.args(out))
			requireOutputFailure(t, executeCapturedCommand(cmd))
			if len(*codes) != 1 || (*codes)[0] != 1 {
				t.Fatalf("generator output failure did not produce a failing status: %v", *codes)
			}
			if _, err := os.Stat(filepath.Join(out, test.published)); err != nil {
				t.Fatalf("generator did not publish %s before the broken pipe: %v", test.published, err)
			}
		})
	}
}
