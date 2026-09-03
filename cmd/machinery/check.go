package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/gates"
)

func newCheckCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "check <design-dir> [--impl d] [--commit sha] [--gate gm,gs,gu,gp,gi,gn,gc,g2,g3,gd,gl,gx,gk,gb,ge,ga,gj,gv,g4,gt,g5]",
		Short: "Run the deterministic verification gates on a design",
		Args:  cobra.ExactArgs(1),
	}
	var implDir string
	var gateList string
	var commit string
	var warningsAsErrors bool
	var complete bool
	c.Flags().StringVar(&implDir, "impl", "", "implementation directory for G4-import and Gt-tests")
	c.Flags().StringVar(&gateList, "gate", "", "comma list of gates to run: gm,gs,gu,gp,gi,gn,gc,g2,g3,gd,gl,gx,gk,gb,ge,ga,gj,gv,g4,gt,g5")
	c.Flags().StringVar(&commit, "commit", "", "commit under review, bound by Ga-accept's evidence (env MACHINERY_COMMIT; the flag wins)")
	c.Flags().BoolVar(&warningsAsErrors, "warnings-as-errors", false, "treat every gate warning as a blocking finding")
	c.Flags().BoolVar(&complete, "complete", false, "final-handoff mode: require all phase artifacts, --impl, closed milestones, and zero warnings")
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		stdout, stderr := output.stdout, output.stderr
		design := args[0]
		if err := checkIsDir(design); err != nil {
			fmt.Fprintln(stderr, err)
			return commandExitBecause(1, err)
		}
		if complete && gateList != "" {
			fmt.Fprintln(stderr, "machinery_check: --complete cannot be combined with --gate; final handoff always runs the full applicable suite")
			return commandExit(1)
		}
		if complete && implDir == "" {
			fmt.Fprintln(stderr, "machinery_check: --complete requires --impl so final handoff includes G4-import and Gt-tests")
			return commandExit(1)
		}
		warningsAsErrors = warningsAsErrors || complete
		// the flag wins over the environment, so a CI job's exported commit
		// never silently overrides what the operator typed
		if commit == "" {
			commit = os.Getenv("MACHINERY_COMMIT")
		}
		sel, run, skewNote, err := gates.SelectRunAndNote(design, implDir, gateList, gates.RunOptions{Commit: commit, Complete: complete})
		if sel.Note != "" {
			fmt.Fprintln(stdout, sel.Note)
		}
		if err != nil {
			fmt.Fprintf(stderr, "machinery_check: %s\n", err)
			return commandExitBecause(1, err)
		}
		if sel.Explicit && implDir == "" {
			for _, gname := range []string{"g4", "gt"} {
				if sel.Run[gname] {
					fmt.Fprintf(stderr, "machinery_check: --gate %s requires --impl\n", gname)
					return commandExit(1)
				}
			}
		}

		fail := 0
		for _, g := range run {
			fail += g.Emit(stdout)
			if warningsAsErrors {
				fail += len(g.Warns)
			}
		}
		// P-F10: committed artifacts stamped by another machinery version are
		// worth one non-blocking INFO line; a missing stamp says nothing.
		if skewNote != "" {
			fmt.Fprintln(stdout, skewNote)
		}
		if warningsAsErrors {
			fmt.Fprintf(stdout, "\n%d blocking (ERROR/DRIFT/warning) finding(s); warnings are errors\n", fail)
		} else {
			fmt.Fprintf(stdout, "\n%d blocking (ERROR/DRIFT) finding(s)\n", fail)
		}
		// the platform-green summary (brownfield guide section 5): design
		// gates, G4-import, and Gt-tests together are platform-green; less is
		// component-green and must not be reported as the platform passing.
		// Only a default (non-explicit) run earns the claim: an explicit
		// --gate subset verified only what it named.
		if fail == 0 && !sel.Explicit {
			if implDir != "" {
				fmt.Fprintln(stdout, "platform-green: design gates, G4-import, and Gt-tests all green")
			} else {
				fmt.Fprintln(stdout, "design-green: all applicable design gates green (add --impl for G4-import and Gt-tests toward platform-green)")
			}
		}
		if fail > 0 {
			return commandExit(1)
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
