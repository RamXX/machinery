package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/gates"
)

func newCheckCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "check <design-dir> [--impl d] [--gate gp,g2,g3,gx,g4,g5]",
		Short: "Run the deterministic verification gates on a design",
		Args:  cobra.ExactArgs(1),
	}
	var implDir string
	var gateList string
	c.Flags().StringVar(&implDir, "impl", "", "implementation directory for G4-import")
	c.Flags().StringVar(&gateList, "gate", "", "comma list of gates to run: gp,g2,g3,gx,g4,g5")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		design := args[0]
		if err := checkIsDir(design); err != nil {
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
			return nil
		}
		sel, err := gates.Select(design, gateList)
		if sel.Note != "" {
			fmt.Fprintln(stdoutW, sel.Note)
		}
		if err != nil {
			fmt.Fprintf(stderrW, "machinery_check: %s\n", err)
			exitFunc(1)
			return nil
		}
		if sel.Run["g4"] && sel.Explicit && implDir == "" {
			fmt.Fprintln(stderrW, "machinery_check: --gate g4 requires --impl")
			exitFunc(1)
			return nil
		}

		fail := 0
		for _, g := range gates.RunSelected(design, implDir, sel) {
			fail += g.Emit(stdoutW)
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

// checkIsDir mirrors the Python "design directory does not exist" error.
func checkIsDir(design string) error {
	fi, err := os.Stat(design)
	if err != nil || !fi.IsDir() {
		return fmt.Errorf("machinery_check: design directory %s does not exist", quote(design))
	}
	return nil
}
