// Package main is the machinery binary entrypoint: a cobra root that delegates
// to the internal packages (ir, lint, oracle, tla, refine, compose, gates, formal, diag).
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at link time via -ldflags "-X main.version=v0.1.5" (the
// Makefile and the release workflow both inject it). The -dev default below is
// what a bare `go build` without ldflags reports, so an ad-hoc build is never
// mistaken for a released binary.
var version = "v0.1.5-dev"

func main() {
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
	root.AddCommand(newRefineCmd())
	root.AddCommand(newComposeCmd())
	root.AddCommand(newCheckCmd())
	root.AddCommand(newBaselineCmd())
	root.AddCommand(newVerifyFormalCmd())
	root.AddCommand(newPackCmd())
	root.AddCommand(newScaleCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newPreflightCmd())
	root.AddCommand(newInstallCmd())
	root.AddCommand(newUninstallCmd())
	root.AddCommand(newIRDumpCmd()) // hidden: the Phase-2 parity probe
	root.AddCommand(newHookCmd())   // hidden: Claude Code plugin plumbing

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
