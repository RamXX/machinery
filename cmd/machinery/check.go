package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/gates"
	"github.com/RamXX/machinery/internal/pack"
)

func newCheckCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "check <design-dir> [--impl d] [--gate g2,g3,gx,g4,g5]",
		Short: "Run the deterministic verification gates on a design",
		Args:  cobra.ExactArgs(1),
	}
	var implDir string
	var gateList string
	c.Flags().StringVar(&implDir, "impl", "", "implementation directory for G4-import")
	c.Flags().StringVar(&gateList, "gate", "", "comma list of gates to run: g2,g3,gx,g4,g5")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		design := args[0]
		if err := checkIsDir(design); err != nil {
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
			return nil
		}
		gatesToRun := map[string]bool{}
		explicit := gateList != ""
		gs := "g2,g3,gx,g4,g5"
		if !explicit && pack.HasDecomposition(design) {
			if fi, err := os.Stat(design + "/machines"); err != nil || !fi.IsDir() {
				// a pure decomposed parent authors no machines: its behavior
				// layer is the children's, held by the packs; G3/Gx run there
				gs = "g2,g5"
				fmt.Fprintln(stdoutW, "note: decomposed parent with no machines/; running g2,g5 (G3/Gx/G4 run on the child designs)")
			}
		}
		if explicit {
			gs = gateList
		}
		for _, g := range strings.Split(strings.ToLower(gs), ",") {
			gatesToRun[strings.TrimSpace(g)] = true
		}
		var unknown []string
		for g := range gatesToRun {
			if g != "g2" && g != "g3" && g != "gx" && g != "g4" && g != "g5" {
				unknown = append(unknown, g)
			}
		}
		if len(unknown) > 0 {
			sort.Strings(unknown)
			fmt.Fprintf(stderrW, "machinery_check: unknown gate(s): %s\n", strings.Join(unknown, ", "))
			exitFunc(1)
			return nil
		}
		if gatesToRun["g4"] && explicit && implDir == "" {
			fmt.Fprintln(stderrW, "machinery_check: --gate g4 requires --impl")
			exitFunc(1)
			return nil
		}

		fail := 0
		if gatesToRun["g2"] {
			fail += gates.CheckC4(design).Emit(stdoutW)
		}
		if gatesToRun["g3"] {
			fail += gates.CheckMachines(design).Emit(stdoutW)
		}
		if gatesToRun["gx"] {
			fail += gates.CheckTraceability(design).Emit(stdoutW)
		}
		if gatesToRun["g4"] && implDir != "" {
			fail += gates.CheckImports(design, implDir).Emit(stdoutW)
		}
		// G5 runs by default only on decomposed designs; asking for it
		// explicitly on a plain design is an error, never a silent pass
		if gatesToRun["g5"] && (explicit || pack.HasDecomposition(design) || pack.HasPack(design)) {
			fail += gates.CheckPack(design).Emit(stdoutW)
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
