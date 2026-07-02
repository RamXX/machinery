// Package main is the machinery binary entrypoint: a cobra root that delegates
// to the internal packages (ir, lint, oracle, tla, refine, compose, gates, formal, diag).
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at link time via -ldflags "-X main.version=v0.1.0".
// The default below is what `go build` without ldflags reports.
var version = "v0.1.0"

func main() {
	root := &cobra.Command{
		Use:           "machinery",
		Short:         "machinery deterministic design tooling",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().Bool("version", false, "print version and exit")

	root.AddCommand(newLintCmd())
	root.AddCommand(newOracleCmd())
	root.AddCommand(newTLACmd())
	root.AddCommand(newRefineCmd())
	root.AddCommand(newComposeCmd())
	root.AddCommand(newCheckCmd())
	root.AddCommand(newVerifyFormalCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newPreflightCmd())
	root.AddCommand(newIRDumpCmd()) // hidden: the Phase-2 parity probe

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
