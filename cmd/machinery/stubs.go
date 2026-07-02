package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Phase-1 stubs. Each subcommand exits 2 with "not implemented" until its phase
// replaces the body. The differential harness depends on these existing.

func newLintCmd() *cobra.Command {
	return &cobra.Command{Use: "lint <machines-dir>", RunE: func(cmd *cobra.Command, args []string) error {
		return exit2(fmt.Errorf("lint: not implemented"))
	}}
}

func newOracleCmd() *cobra.Command {
	return &cobra.Command{Use: "oracle <machines-dir>", RunE: func(cmd *cobra.Command, args []string) error {
		return exit2(fmt.Errorf("oracle: not implemented"))
	}}
}

func newTLACmd() *cobra.Command {
	return &cobra.Command{Use: "tla <machine.json> [out-dir]", RunE: func(cmd *cobra.Command, args []string) error {
		return exit2(fmt.Errorf("tla: not implemented"))
	}}
}

func newRefineCmd() *cobra.Command {
	return &cobra.Command{Use: "refine <machine.json> <semantics.yaml> [out-dir]", RunE: func(cmd *cobra.Command, args []string) error {
		return exit2(fmt.Errorf("refine: not implemented"))
	}}
}

func newComposeCmd() *cobra.Command {
	return &cobra.Command{Use: "compose <composition.yaml> <coordinator.machine.json> [out-dir]", RunE: func(cmd *cobra.Command, args []string) error {
		return exit2(fmt.Errorf("compose: not implemented"))
	}}
}

func newCheckCmd() *cobra.Command {
	return &cobra.Command{Use: "check <design-dir> [--impl d] [--gate ...]", RunE: func(cmd *cobra.Command, args []string) error {
		return exit2(fmt.Errorf("check: not implemented"))
	}}
}

func newVerifyFormalCmd() *cobra.Command {
	return &cobra.Command{Use: "verify-formal <design-dir>", RunE: func(cmd *cobra.Command, args []string) error {
		return exit2(fmt.Errorf("verify-formal: not implemented"))
	}}
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{Use: "doctor", RunE: func(cmd *cobra.Command, args []string) error {
		return exit2(fmt.Errorf("doctor: not implemented"))
	}}
}

func newPreflightCmd() *cobra.Command {
	return &cobra.Command{Use: "preflight", RunE: func(cmd *cobra.Command, args []string) error {
		return exit2(fmt.Errorf("preflight: not implemented"))
	}}
}

func newIRDumpCmd() *cobra.Command {
	c := &cobra.Command{
		Use:    "ir-dump <machine.json>",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
	}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		return irDumpRun(args[0])
	}
	return c
}

// exit2 mirrors Python sys.exit(nonzero): print to stderr, exit code 2.
// We return an error that main prints; main exits 1 on RunE error, so to get
// exit code 2 we os.Exit directly here.
func exit2(err error) error {
	fmt.Fprintln(stderrW, err)
	exitFunc(2)
	return err
}
