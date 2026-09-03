package main

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/install"
)

func TestEnforceConsistentActivationReexecutesRestoredBinary(t *testing.T) {
	previousEnsure, previousValidate, previousReexec := ensureMachineryActivation, validateActivationRecovery, reexecMachineryProcess
	t.Cleanup(func() {
		ensureMachineryActivation, validateActivationRecovery, reexecMachineryProcess = previousEnsure, previousValidate, previousReexec
	})
	t.Setenv(activationReexecGuardEnv, "")
	ensureMachineryActivation = func() error {
		return &install.ActivationRecoveryError{Executable: "/restored/machinery", Identity: "sha256:restored"}
	}
	validateActivationRecovery = func(error) error { return nil }
	called := ""
	reexecMachineryProcess = func(recovery error) (int, error) {
		called, _ = install.ActivationRecoveryExecutable(recovery)
		return 23, nil
	}
	exitCode, err := enforceConsistentActivation()
	if err != nil || exitCode != 23 || called != "/restored/machinery" {
		t.Fatalf("activation = code %d, called %q, err %v", exitCode, called, err)
	}
	if got := os.Getenv(activationReexecGuardEnv); got != "sha256:restored" {
		t.Fatalf("activation guard = %q", got)
	}
}

func TestEnforceConsistentActivationFailsClosedWithoutRecoverySignal(t *testing.T) {
	previousEnsure, previousValidate, previousReexec := ensureMachineryActivation, validateActivationRecovery, reexecMachineryProcess
	t.Cleanup(func() {
		ensureMachineryActivation, validateActivationRecovery, reexecMachineryProcess = previousEnsure, previousValidate, previousReexec
	})
	want := errors.New("untrusted journal")
	ensureMachineryActivation = func() error { return want }
	reexecMachineryProcess = func(error) (int, error) {
		t.Fatal("reexec called without a recovery signal")
		return 0, nil
	}
	if _, err := enforceConsistentActivation(); !errors.Is(err, want) {
		t.Fatalf("activation error = %v, want %v", err, want)
	}
}

func TestEnforceConsistentActivationRejectsReexecLoop(t *testing.T) {
	previousEnsure, previousValidate, previousReexec := ensureMachineryActivation, validateActivationRecovery, reexecMachineryProcess
	t.Cleanup(func() {
		ensureMachineryActivation, validateActivationRecovery, reexecMachineryProcess = previousEnsure, previousValidate, previousReexec
	})
	t.Setenv(activationReexecGuardEnv, "sha256:same")
	ensureMachineryActivation = func() error {
		return &install.ActivationRecoveryError{Executable: "/restored/machinery", Identity: "sha256:same"}
	}
	validateActivationRecovery = func(error) error { return nil }
	reexecMachineryProcess = func(error) (int, error) {
		t.Fatal("looping activation reexec was attempted")
		return 0, nil
	}
	if _, err := enforceConsistentActivation(); err == nil || !strings.Contains(err.Error(), "refuse repeated") {
		t.Fatalf("loop guard error = %v", err)
	}
}
