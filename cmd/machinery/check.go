package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/gates"
)

func newCheckCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "check <design-dir> [--impl d] [--gate gm,gs,gp,gi,gn,g2,g3,gx,gb,g4,gt,g5]",
		Short: "Run the deterministic verification gates on a design",
		Args:  cobra.ExactArgs(1),
	}
	var implDir string
	var gateList string
	c.Flags().StringVar(&implDir, "impl", "", "implementation directory for G4-import and Gt-tests")
	c.Flags().StringVar(&gateList, "gate", "", "comma list of gates to run: gm,gs,gp,gi,gn,g2,g3,gx,gb,g4,gt,g5")
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

		fail := 0
		run := gates.RunSelected(design, implDir, sel)
		for _, g := range run {
			fail += g.Emit(stdoutW)
		}
		// P-F10: committed artifacts stamped by another machinery version are
		// worth one non-blocking INFO line; a missing stamp says nothing.
		if note := gates.VersionSkewNote(run); note != "" {
			fmt.Fprintln(stdoutW, note)
		}
		fmt.Fprintf(stdoutW, "\n%d blocking (ERROR/DRIFT) finding(s)\n", fail)
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
