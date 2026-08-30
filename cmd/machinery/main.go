// Package main is the machinery binary entrypoint: a cobra root that delegates
// to the internal packages (ir, lint, oracle, tla, refine, compose, gates, formal, diag).
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	machversion "github.com/RamXX/machinery/internal/version"
)

// version is set at link time via -ldflags "-X main.version=<release tag>" (the
// Makefile and the release workflow both inject it). Versions are numeric
// only: the plain default below is what a bare `go build` without ldflags
// reports, identical to the released binary of the same version.
var version = "v0.4.1"

func main() {
	// propagate the (possibly ldflags-injected) binary version to the
	// generators and gates, which stamp and compare `machinery-version:`
	// lines in committed artifacts (P-F10)
	machversion.Version = version

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
	root.AddCommand(newTLACmd())
	root.AddCommand(newAlloyCmd())
	root.AddCommand(newRefineCmd())
	root.AddCommand(newComposeCmd())
	root.AddCommand(newCheckCmd())
	root.AddCommand(newAttestCmd())
	root.AddCommand(newProjectCmd())
	root.AddCommand(newVerifyCheckersCmd())
	root.AddCommand(newBaselineCmd())
	root.AddCommand(newVerifyFormalCmd())
	root.AddCommand(newVerifyC4Cmd())
	root.AddCommand(newPackCmd())
	root.AddCommand(newScaleCmd())
	root.AddCommand(newSweepCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newPreflightCmd())
	root.AddCommand(newInstallCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newUninstallCmd())
	root.AddCommand(newIRDumpCmd()) // hidden: the Phase-2 parity probe
	root.AddCommand(newHookCmd())   // hidden: agent-host adapter plumbing

	// top-level --version
	ver := &cobra.Command{
		Use:   "version",
		Short: "Print the machinery version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("machinery version " + version)
		},
	}
	root.AddCommand(ver)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
