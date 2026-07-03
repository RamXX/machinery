package main

import (
	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/formal"
)

// Phase-1 stubs. Each subcommand exits 2 with "not implemented" until its phase
// replaces the body. The differential harness depends on these existing.

func newVerifyFormalCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "verify-formal <design-dir>",
		Short: "Regenerate + TLC-check the formal suite for a design",
		Args:  cobra.ExactArgs(1),
	}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		rc := formal.VerifyFormal(args[0])
		exitFunc(rc)
		return nil
	}
	return c
}

func newDoctorCmd() *cobra.Command {
	c := &cobra.Command{Use: "doctor", Short: "Check prerequisites and install status"}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		doctorRun()
		return nil
	}
	return c
}

func newPreflightCmd() *cobra.Command {
	c := &cobra.Command{Use: "preflight", Short: "Check runtime prerequisites (warns; installs nothing)"}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		preflightRun()
		return nil
	}
	return c
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
