// Package main is the machinery binary entrypoint: a cobra root that delegates
// to the internal packages (ir, lint, oracle, tla, refine, compose, gates, formal, diag).
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/install"
	"github.com/RamXX/machinery/internal/processscope"
	machversion "github.com/RamXX/machinery/internal/version"
)

// version is set at link time via -ldflags "-X main.version=<release tag>" (the
// Makefile and the release workflow both inject it). Versions are numeric
// only: the plain default below is what a bare `go build` without ldflags
// reports, identical to the released binary of the same version.
var version = "v0.9.0"

const activationReexecGuardEnv = "MACHINERY_INTERNAL_ACTIVATION_REEXEC_GUARD"

var (
	ensureMachineryActivation  = install.EnsureActivationConsistency
	validateActivationRecovery = install.ValidateActivationRecovery
	reexecMachineryProcess     = func(recovery error) (int, error) {
		return install.ReexecActivationRecovery(recovery, os.Args, os.Environ(), os.Stdin, os.Stdout, os.Stderr)
	}
)

func main() {
	if exitCode, handled := serveInternalActivation(os.Args[1:]); handled {
		os.Exit(exitCode)
	}
	if exitCode, err := enforceConsistentActivation(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	} else if exitCode >= 0 {
		os.Exit(exitCode)
	}
	// propagate the (possibly ldflags-injected) binary version to the
	// generators and gates, which stamp and compare `machinery-version:`
	// lines in committed artifacts (P-F10)
	machversion.Version = version

	root := newRootCmd()
	if err := root.Execute(); err != nil {
		code, remaining := commandResult(err)
		if remaining != nil {
			fmt.Fprintln(os.Stderr, remaining)
		}
		if code == 0 {
			code = 1
		}
		os.Exit(code)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "machinery",
		Short:         "machinery deterministic design tooling",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}
	root.SetVersionTemplate("machinery version {{.Version}}\n")

	root.AddCommand(newLintCmd())
	root.AddCommand(newOracleCmd())
	root.AddCommand(newTokensEqualCmd())
	root.AddCommand(newTLACmd())
	root.AddCommand(newAlloyCmd())
	root.AddCommand(newRefineCmd())
	root.AddCommand(newComposeCmd())
	root.AddCommand(newCheckCmd())
	root.AddCommand(newAttestCmd())
	root.AddCommand(newProjectCmd())
	root.AddCommand(newPacketCmd())
	root.AddCommand(newVerifyCheckersCmd())
	root.AddCommand(newBaselineCmd())
	root.AddCommand(newVerifyFormalCmd())
	root.AddCommand(newVerifyC4Cmd())
	root.AddCommand(newPackCmd())
	root.AddCommand(newEmbedCmd())
	root.AddCommand(newScaleCmd())
	root.AddCommand(newSweepCmd())
	root.AddCommand(newRecoverCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newPreflightCmd())
	root.AddCommand(newInstallCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newUninstallCmd())
	root.AddCommand(newIRDumpCmd()) // hidden: the Phase-2 parity probe
	root.AddCommand(newHookCmd())   // hidden: agent-host adapter plumbing

	// top-level --version
	root.AddCommand(newVersionCmd())

	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the machinery version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (retErr error) {
			output := trackOutput(cmd.OutOrStdout(), cmd.ErrOrStderr())
			defer func() { retErr = output.join(retErr) }()
			fmt.Fprintln(output.stdout, "machinery version "+version)
			return nil
		},
	}
}

// serveInternalActivation handles the authenticated internal custody
// activation before any ordinary product execution: it acquires the inherited
// internal channel error-first, rejects malformed or untrusted claims without
// running normal CLI behavior, and serves verified broker/guardian requests
// with the same candidate binary. It reports handled=false only for the
// verified absence of any internal claim.
func serveInternalActivation(args []string) (exitCode int, handled bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	io, present, err := processscope.InheritedInternalIO(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "machinery: refusing untrusted internal activation:", err)
		return 1, true
	}
	if !present {
		return 0, false
	}
	defer io.Close()
	wasHandled, code := processscope.ServeInternal(args, io)
	if !wasHandled {
		fmt.Fprintln(os.Stderr, "machinery: internal channel claim without the internal marker; refusing normal execution")
		return 1, true
	}
	return code, true
}

func enforceConsistentActivation() (int, error) {
	recoveryErr := ensureMachineryActivation()
	if recoveryErr == nil {
		_ = os.Unsetenv(activationReexecGuardEnv)
		return -1, nil
	}
	var recovery *install.ActivationRecoveryError
	if !errors.As(recoveryErr, &recovery) {
		return -1, recoveryErr
	}
	if validationErr := validateActivationRecovery(recoveryErr); validationErr != nil {
		return -1, errors.Join(fmt.Errorf("validate restored executable before re-exec: %w", validationErr), install.CloseActivationRecovery(recoveryErr))
	}
	if guard := os.Getenv(activationReexecGuardEnv); guard != "" {
		return -1, errors.Join(fmt.Errorf("refuse repeated activation re-exec (guard %q, restored identity %q)", guard, recovery.Identity), install.CloseActivationRecovery(recoveryErr))
	}
	if err := os.Setenv(activationReexecGuardEnv, recovery.Identity); err != nil {
		return -1, errors.Join(fmt.Errorf("set activation re-exec guard: %w", err), install.CloseActivationRecovery(recoveryErr))
	}
	return reexecMachineryProcess(recoveryErr)
}
