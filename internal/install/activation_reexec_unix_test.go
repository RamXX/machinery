//go:build !windows

package install

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const (
	activationHelperEnv      = "MACHINERY_TEST_ACTIVATION_HELPER"
	activationCompanionEnv   = "MACHINERY_TEST_ACTIVATION_COMPANION"
	activationObservationEnv = "MACHINERY_TEST_ACTIVATION_OBSERVATION"
)

func TestPreparedExecutableRecoveryReexecsExactRetainedImageBeforeHostObservation(t *testing.T) {
	config := privateConfigDir(t)
	t.Setenv("MACHINERY_CONFIG_DIR", config)
	running, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(running)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	executable := filepath.Join(root, "machinery-recovered-test")
	companion := filepath.Join(root, "adapter")
	observation := filepath.Join(root, "host-observation")
	if err := os.WriteFile(executable, raw, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(executable, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, companion, "old-companion")

	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := beginArtifactTransaction([]string{executable, companion})
	if err != nil {
		t.Fatal(err)
	}
	staged, cleanup, err := installScratchFile(root, "replacement-executable")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			t.Error(err)
		}
	}()
	if err := staged.Chmod(0o755); err != nil {
		t.Fatal(err)
	}
	in, err := os.Open(running)
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(staged, in)
	if err := in.Close(); err != nil {
		t.Fatal(err)
	}
	if copyErr != nil {
		t.Fatal(copyErr)
	}
	if err := staged.Sync(); err != nil {
		t.Fatal(err)
	}
	stagedPath := staged.Name()
	if err := staged.Close(); err != nil {
		t.Fatal(err)
	}
	if err := renameReplace(stagedPath, executable); err != nil {
		t.Fatal(err)
	}
	transactionReplaceForTest(t, companion, "partial-new-companion")
	if err := tx.closeAnchors(); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}

	command := exec.CommandContext(context.Background(), executable, "-test.run=^TestActivationRecoveryReexecHelper$")
	command.Env = append(os.Environ(),
		activationHelperEnv+"=new-image",
		activationCompanionEnv+"="+companion,
		activationObservationEnv+"="+observation,
		"MACHINERY_CONFIG_DIR="+config,
	)
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	startedPID := command.Process.Pid
	if waitErr := command.Wait(); waitErr != nil {
		t.Fatalf("activation helper failed: %v\n%s", waitErr, output.String())
	}
	observed, err := os.ReadFile(observation)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("pid=%d companion=old-companion", startedPID)
	if strings.TrimSpace(string(observed)) != want {
		t.Fatalf("host observation = %q, want %q", observed, want)
	}
	if runtime.GOOS == "linux" {
		if got, err := os.ReadFile(observation + ".stage-swapped"); err != nil || string(got) != "swapped" {
			t.Fatalf("after-final-validation activation swap hook did not run: %q, %v", got, err)
		}
		if _, err := os.Lstat(observation + ".stage-malicious-ran"); !os.IsNotExist(err) {
			t.Fatalf("malicious staged pathname executed instead of retained fd: %v", err)
		}
	}
	if got, err := os.ReadFile(companion); err != nil || string(got) != "old-companion" {
		t.Fatalf("companion after recovery = %q, %v", got, err)
	}
	// The validation hook replaced the restored destination pathname with a
	// malicious script. The unchanged PID and Go-helper observation prove
	// activation did not reopen that mutable destination after validation.
	if got, err := os.ReadFile(executable); err != nil || !strings.Contains(string(got), "MALICIOUS-PATH") {
		t.Fatalf("ABA path fixture was not installed: %q, %v", got, err)
	}
}

func TestActivationRecoveryReexecHelper(t *testing.T) {
	stage := os.Getenv(activationHelperEnv)
	if stage == "" {
		return
	}
	companion := os.Getenv(activationCompanionEnv)
	observation := os.Getenv(activationObservationEnv)
	if stage == "restored-image" {
		if err := EnsureActivationConsistency(); err != nil {
			t.Fatal(err)
		}
		content, err := os.ReadFile(companion)
		if err != nil {
			t.Fatal(err)
		}
		result := "pid=" + strconv.Itoa(os.Getpid()) + " companion=" + string(content)
		if err := os.WriteFile(observation, []byte(result), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}

	recoveryErr := EnsureActivationConsistency()
	if _, ok := ActivationRecoveryExecutable(recoveryErr); !ok {
		t.Fatalf("new image continued without activation recovery: %v", recoveryErr)
	}
	if err := os.Setenv(activationHelperEnv, "restored-image"); err != nil {
		t.Fatal(err)
	}
	afterActivationValidation = func(_ string) {
		afterActivationValidation = nil
		executable, _ := ActivationRecoveryExecutable(recoveryErr)
		parked := executable + ".verified"
		if err := os.Rename(executable, parked); err != nil {
			t.Fatal(err)
		}
		malicious := "#!/bin/sh\necho MALICIOUS-PATH >" + observation + "\n"
		if err := os.WriteFile(executable, []byte(malicious), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(executable, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if runtime.GOOS == "linux" {
		setBeforeActivationExecForTest(func(staged string) {
			setBeforeActivationExecForTest(nil)
			if err := os.Chmod(filepath.Dir(staged), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(staged, staged+".post-validation"); err != nil {
				t.Fatal(err)
			}
			malicious := "#!/bin/sh\necho ran >" + observation + ".stage-malicious-ran\n"
			if err := os.WriteFile(staged, []byte(malicious), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(staged, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(observation+".stage-swapped", []byte("swapped"), 0o600); err != nil {
				t.Fatal(err)
			}
		})
	}
	if _, err := ReexecActivationRecovery(recoveryErr, os.Args, os.Environ(), os.Stdin, os.Stdout, os.Stderr); err != nil {
		t.Fatal(err)
	}
	t.Fatal("restored executable reexec returned unexpectedly")
}

func TestActivationReexecFailsClosedOnPostValidationStagedPathSwap(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	running, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	operation, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	file, identity, err := retainActivationExecutable(running)
	if err != nil {
		operation.Release()
		t.Fatal(err)
	}
	activationPath, err := stageActivationExecutable(running, file, identity)
	if err != nil {
		file.Close()
		operation.Release()
		t.Fatal(err)
	}
	recovery := &ActivationRecoveryError{
		Executable:     running,
		Identity:       identity,
		file:           file,
		lock:           operation.lock,
		activationPath: activationPath,
	}
	operation.lock = nil
	marker := filepath.Join(t.TempDir(), "malicious-ran")
	afterActivationValidation = func(staged string) {
		afterActivationValidation = nil
		if err := os.Chmod(filepath.Dir(staged), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(staged, staged+".verified"); err != nil {
			t.Fatal(err)
		}
		malicious := "#!/bin/sh\necho ran >" + marker + "\n"
		if err := os.WriteFile(staged, []byte(malicious), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { afterActivationValidation = nil })
	_, err = ReexecActivationRecovery(recovery, []string{running}, os.Environ(), os.Stdin, os.Stdout, os.Stderr)
	if err == nil || !strings.Contains(err.Error(), "changed after final validation") {
		t.Fatalf("staged ABA error = %v", err)
	}
	if _, err := os.Lstat(marker); !os.IsNotExist(err) {
		t.Fatalf("malicious staged replacement executed: %v", err)
	}
	// Reexec failure must release the retained operation lock.
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatalf("activation failure leaked operation lock: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
}
