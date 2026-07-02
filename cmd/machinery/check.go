package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ramirosalas/machinery/internal/gates"
)

func newCheckCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "check <design-dir> [--impl d] [--gate g2,g3,gx,g4]",
		Short: "Run the deterministic verification gates on a design",
		Args:  cobra.ExactArgs(1),
	}
	var implDir string
	var gateList string
	c.Flags().StringVar(&implDir, "impl", "", "implementation directory for G4-import")
	c.Flags().StringVar(&gateList, "gate", "", "comma list of gates to run: g2,g3,gx,g4")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		design := args[0]
		if fi, err := os.Stat(design); err != nil || !fi.IsDir() {
			fmt.Fprintf(stderrW, "machinery_check: design directory %s does not exist\n", quote(design))
			exitFunc(1)
			return nil
		}
		gatesToRun := map[string]bool{}
		gs := "g2,g3,gx,g4"
		if gateList != "" {
			gs = gateList
		}
		for _, g := range strings.Split(strings.ToLower(gs), ",") {
			gatesToRun[strings.TrimSpace(g)] = true
		}
		var unknown []string
		for g := range gatesToRun {
			if g != "g2" && g != "g3" && g != "gx" && g != "g4" {
				unknown = append(unknown, g)
			}
		}
		if len(unknown) > 0 {
			sort.Strings(unknown)
			fmt.Fprintf(stderrW, "machinery_check: unknown gate(s): %s\n", strings.Join(unknown, ", "))
			exitFunc(1)
			return nil
		}
		if gatesToRun["g4"] && strings.Contains(gateList, "g4") && implDir == "" {
			fmt.Fprintln(stderrW, "machinery_check: --gate g4 requires --impl")
			exitFunc(1)
			return nil
		}

		fail := 0
		if gatesToRun["g2"] {
			fail += gates.CheckC4(design).Emit(os.Stdout)
		}
		if gatesToRun["g3"] {
			fail += gates.CheckMachines(design).Emit(os.Stdout)
		}
		if gatesToRun["gx"] {
			fail += gates.CheckTraceability(design).Emit(os.Stdout)
		}
		if gatesToRun["g4"] && implDir != "" {
			fail += gates.CheckImports(design, implDir).Emit(os.Stdout)
		}
		fmt.Fprintf(os.Stdout, "\n%d blocking (ERROR/DRIFT) finding(s)\n", fail)
		if fail > 0 {
			exitFunc(1)
		}
		return nil
	}
	return c
}

func quote(s string) string { return "'" + s + "'" }
