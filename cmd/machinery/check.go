package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/gates"
)

func newCheckCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "check <design-dir> [--impl d] [--commit sha] [--gate gm,gs,gu,gp,gi,gn,gc,g2,g3,gd,gl,gx,gk,gb,ge,ga,gj,g4,gt,g5]",
		Short: "Run the deterministic verification gates on a design",
		Args:  cobra.ExactArgs(1),
	}
	var implDir string
	var gateList string
	var commit string
	c.Flags().StringVar(&implDir, "impl", "", "implementation directory for G4-import and Gt-tests")
	c.Flags().StringVar(&gateList, "gate", "", "comma list of gates to run: gm,gs,gu,gp,gi,gn,gc,g2,g3,gd,gl,gx,gk,gb,ge,ga,gj,g4,gt,g5")
	c.Flags().StringVar(&commit, "commit", "", "commit under review, bound by Ga-accept's evidence (env MACHINERY_COMMIT; the flag wins)")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		design := args[0]
		if err := checkIsDir(design); err != nil {
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
			return nil
		}
		sel, err := gates.Select(design, gateList, implDir)
		if sel.Note != "" {
			fmt.Fprintln(stdoutW, sel.Note)
		}
		if err != nil {
			fmt.Fprintf(stderrW, "machinery_check: %s\n", err)
			exitFunc(1)
			return nil
		}
		if sel.Explicit && implDir == "" {
			for _, gname := range []string{"g4", "gt"} {
				if sel.Run[gname] {
					fmt.Fprintf(stderrW, "machinery_check: --gate %s requires --impl\n", gname)
					exitFunc(1)
					return nil
				}
			}
		}

		// the flag wins over the environment, so a CI job's exported commit
		// never silently overrides what the operator typed
		if commit == "" {
			commit = os.Getenv("MACHINERY_COMMIT")
		}

		fail := 0
		run := gates.RunSelected(design, implDir, sel, gates.RunOptions{Commit: commit})
		for _, g := range run {
			fail += g.Emit(stdoutW)
		}
		// P-F10: committed artifacts stamped by another machinery version are
		// worth one non-blocking INFO line; a missing stamp says nothing.
		if note := gates.VersionSkewNote(run); note != "" {
			fmt.Fprintln(stdoutW, note)
		}
		fmt.Fprintf(stdoutW, "\n%d blocking (ERROR/DRIFT) finding(s)\n", fail)
		// the platform-green summary (brownfield guide section 5): design
		// gates, G4-import, and Gt-tests together are platform-green; less is
		// component-green and must not be reported as the platform passing.
		// Only a default (non-explicit) run earns the claim: an explicit
		// --gate subset verified only what it named.
		if fail == 0 && !sel.Explicit {
			if implDir != "" {
				fmt.Fprintln(stdoutW, "platform-green: design gates, G4-import, and Gt-tests all green")
			} else {
				fmt.Fprintln(stdoutW, "design-green: all applicable design gates green (add --impl for G4-import and Gt-tests toward platform-green)")
			}
		}
		if fail > 0 {
			exitFunc(1)
		}
		return nil
	}
	return c
}

func quote(s string) string { return "'" + s + "'" }

// checkIsDir mirrors the Python "design directory does not exist" error, and
// tells a present-but-not-a-directory path apart from a missing one.
func checkIsDir(design string) error {
	fi, err := os.Stat(design)
	if err != nil {
		return fmt.Errorf("machinery_check: design directory %s does not exist", quote(design))
	}
	if !fi.IsDir() {
		return fmt.Errorf("machinery_check: design path %s is not a directory", quote(design))
	}
	return nil
}
