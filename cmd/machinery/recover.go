package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/designlock"
)

// newRecoverCmd is the safe user surface for interrupted design
// publications: default invocation is a strictly read-only report of the
// writer, expected outputs, journals, live-writer state and the actionable
// recovery decision; --apply finalizes only a fully revalidated
// content-and-mode complete transaction and refuses everything else while
// preserving all evidence.
func newRecoverCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "recover <design-dir>",
		Short: "Inspect an interrupted design publication; with --apply, complete a fully validated one",
		Long: `Read-only by default: reports the interrupted publication's writer, its
recovery instruction, every declared expected output with content/mode
status, every journal and recovery residue location, whether a live writer
holds the design lock, and the actionable safe recovery decision with the
exact reasons for every refusal. Nothing is ever deleted or rewritten during
inspection.

With --apply, completes a transaction whose exact expected input and output
inventories revalidate on this fresh process identity, whose journal is
canonical and rooted, and which no concurrent writer owns. Partial
publications, tampered journals, swapped or symlinked outputs and changed
inputs are refused with the exact conflict; all evidence is preserved. The
remedy for refused states is rerunning the exact writer command that was
interrupted. Never manually delete publication sentinels or journals.`,
		Args: cobra.ExactArgs(1),
	}
	apply := c.Flags().Bool("apply", false, "complete the interrupted publication after every validation passes (default: read-only report)")
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		stdoutW, stderrW := output.stdout, output.stderr
		design := args[0]
		if info, err := os.Stat(design); err != nil || !info.IsDir() {
			failure := fmt.Errorf("machinery_recover: design %s must be an existing directory", quote(design))
			fmt.Fprintln(stderrW, failure)
			return commandExitBecause(1, failure)
		}
		if !*apply {
			report, err := designlock.InspectRecovery(design)
			if err != nil {
				failure := fmt.Errorf("machinery_recover: %w", err)
				fmt.Fprintln(stderrW, failure)
				return commandExitBecause(1, failure)
			}
			renderRecoveryReport(stdoutW, report)
			return nil
		}
		report, err := designlock.RecoverInterrupted(design)
		if err != nil {
			if report != nil {
				renderRecoveryReport(stdoutW, report)
			}
			failure := fmt.Errorf("machinery_recover: %w", err)
			fmt.Fprintln(stderrW, failure)
			return commandExitBecause(1, failure)
		}
		renderRecoveryReport(stdoutW, report)
		if report.Finalized {
			fmt.Fprintln(stdoutW, "completed: interrupted publication finalized; the design is consistent")
		} else {
			fmt.Fprintln(stdoutW, "nothing to recover: no interrupted publication found")
		}
		return nil
	}
	return c
}

func renderRecoveryReport(w io.Writer, report *designlock.RecoveryReport) {
	fmt.Fprintf(w, "design root: %s\n", report.Root)
	stage := "clean"
	if len(report.Journals) > 0 {
		stage = "interrupted"
	}
	fmt.Fprintf(w, "stage: %s\n", stage)
	if report.Writer != "" {
		fmt.Fprintf(w, "writer: %s\n", report.Writer)
	}
	if report.RecoveryCommand != "" {
		fmt.Fprintf(w, "recovery: %s\n", report.RecoveryCommand)
	}
	live := "no"
	if report.LiveWriter {
		live = "yes"
	}
	fmt.Fprintf(w, "live writer: %s\n", live)
	switch report.InputInventory {
	case "verified":
		fmt.Fprintf(w, "input inventory: verified (design digest %s)\n", report.DesignInputDigest)
	case "mismatch":
		fmt.Fprintf(w, "input inventory: mismatch (design digest %s, recorded %s)\n", report.DesignInputDigest, report.InputFingerprint)
	case "unwitnessable":
		fmt.Fprintln(w, "input inventory: unwitnessable")
	}
	if len(report.Outputs) > 0 {
		fmt.Fprintln(w, "outputs:")
		for _, item := range report.Outputs {
			line := "  " + item.Path + "  " + item.Declared + "  " + item.Status
			if item.Detail != "" {
				line += " (" + item.Detail + ")"
			}
			fmt.Fprintln(w, line)
		}
	}
	if len(report.Journals) > 0 {
		fmt.Fprintln(w, "journals:")
		for _, journal := range report.Journals {
			line := "  " + journal.Path + "  " + journal.Role + "  " + journal.Status
			if journal.Detail != "" {
				line += " (" + journal.Detail + ")"
			}
			fmt.Fprintln(w, line)
		}
	}
	fmt.Fprintf(w, "action: %s\n", report.Action)
	if len(report.Reasons) > 0 {
		fmt.Fprintln(w, "reasons:")
		for _, reason := range report.Reasons {
			fmt.Fprintln(w, "  - "+reason)
		}
	}
}
