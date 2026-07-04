package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/gates"
)

func newBaselineCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "baseline <design-dir> --impl <dir>",
		Short: "Record the Stage-1 boundary debt snapshot (baseline rules + ratchet)",
		Long: `Scan the implementation exactly as G4-import does, print the baseline: rules
that would tolerate today's violating edges (paste them into the Architecture
Contract's dependency_rules after review), and write design/ratchet.json, the
set-based snapshot of every tolerated edge's offender files. From then on G4
fails when a baselined edge gains a new offender file, and the machinery
Claude Code plugin blocks import findings at turn end (the snapshot is what
arms that blocking). Rerun after burning down debt to tighten the ratchet.`,
		Args: cobra.ExactArgs(1),
	}
	var implDir, date string
	c.Flags().StringVar(&implDir, "impl", "", "implementation directory to scan (required)")
	c.Flags().StringVar(&date, "date", "", "stamp for the snapshot and rule comments (default: current YYYY-MM)")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		design := args[0]
		if err := checkIsDir(design); err != nil {
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
			return nil
		}
		if implDir == "" {
			fmt.Fprintln(stderrW, "machinery_baseline: --impl is required")
			exitFunc(1)
			return nil
		}
		if date == "" {
			date = time.Now().Format("2006-01")
		}
		rep, err := gates.BuildBaseline(design, implDir, date)
		if err != nil {
			return fmt.Errorf("machinery_baseline: %w", err)
		}

		fmt.Fprintln(stdoutW, "== baseline  boundary debt snapshot ==")
		fmt.Fprintf(stdoutW, "  observed: %d cross-boundary edge(s); %d need a baseline rule; %d source file(s) outside every boundary; %d import(s) map to no boundary\n",
			rep.EdgesObserved, len(rep.Proposed), rep.UnmappedFiles, len(rep.Orphans))

		if len(rep.Proposed) > 0 {
			fmt.Fprintln(stdoutW, "\nadd to the Architecture Contract under dependency_rules (review each edge; keep intent explicit: a deny: for the same edge is legitimate and recommended when the edge should eventually die):")
			fmt.Fprintln(stdoutW, "  baseline:")
			for _, p := range rep.Proposed {
				comment := "# " + date + " seen in " + p.Witness
				if p.More == 1 {
					comment += " and 1 more file"
				} else if p.More > 1 {
					comment += fmt.Sprintf(" and %d more files", p.More)
				}
				fmt.Fprintf(stdoutW, "    - %q   %s\n", p.Edge, comment)
			}
		} else {
			fmt.Fprintln(stdoutW, "\nthe contract already covers every observed edge; nothing new to baseline")
		}

		if len(rep.IgnoreGlobs) > 0 {
			fmt.Fprintln(stdoutW, "\nsuggested ignore: globs for the source files outside every boundary (each glob amnesties a whole directory; review before pasting, and remember ignored code that modeled code imports still needs an external with imports: prefixes):")
			for _, gl := range rep.IgnoreGlobs {
				fmt.Fprintf(stdoutW, "    - %q\n", gl)
			}
		}

		if len(rep.Orphans) > 0 {
			fmt.Fprintln(stdoutW, "\nimports that map to no contract boundary (declare an external, e.g. external.rest_of_monolith, and list these under its imports: prefixes):")
			for _, o := range rep.Orphans {
				fmt.Fprintf(stdoutW, "    - %s (%d file(s))\n", o.Ref, o.Files)
			}
		}

		if err := gates.WriteRatchet(design, rep.Ratchet); err != nil {
			return fmt.Errorf("machinery_baseline: writing %s: %w", gates.RatchetFile, err)
		}
		total := 0
		for _, files := range rep.Ratchet.Edges {
			total += len(files)
		}
		fmt.Fprintf(stdoutW, "\nwrote %s/%s: %d edge(s), %d offender file(s)\n", design, gates.RatchetFile, len(rep.Ratchet.Edges), total)
		fmt.Fprintln(stdoutW, "armed: G4 now fails when a baselined edge gains a new offender file, and the machinery plugin blocks import findings at turn end")
		return nil
	}
	return c
}
