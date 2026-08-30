package main

import (
	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/formal"
)

// Once the Phase-1 stub file; every body below has long been implemented
// (verify-formal, doctor, preflight, and the hidden ir-dump). The file name
// survives for history.

func newVerifyFormalCmd() *cobra.Command {
	var genOnly bool
	c := &cobra.Command{
		Use:   "verify-formal <design-dir>",
		Short: "Regenerate + TLC-check the formal suite for a design",
		Args:  cobra.ExactArgs(1),
	}
	c.Flags().BoolVar(&genOnly, "gen-only", false,
		"regenerate the formal suite from source without running TLC (no Java needed)")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		rc := formal.VerifyFormal(args[0], genOnly)
		exitFunc(rc)
		return nil
	}
	return c
}

func newDoctorCmd() *cobra.Command {
	var targets []string
	c := &cobra.Command{Use: "doctor", Short: "Check prerequisites and install status", Args: cobra.NoArgs}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		return doctorRun(targets)
	}
	c.Flags().StringArrayVar(&targets, "target", nil, "host adapter to inspect: claude, codex, opencode, or all (repeatable)")
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
