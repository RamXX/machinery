package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/artifactset"
	"github.com/RamXX/machinery/internal/designlock"
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
fails when a baselined edge gains a new offender file, and machinery host
adapters with blocking stop hooks reject import findings at turn end (the
snapshot is what arms that blocking). Rerun after burning down debt to tighten
the ratchet.`,
		Args: cobra.ExactArgs(1),
	}
	var implDir, date string
	c.Flags().StringVar(&implDir, "impl", "", "implementation directory to scan (required)")
	c.Flags().StringVar(&date, "date", "", "stamp for the snapshot and rule comments (YYYY-MM-DD; otherwise SOURCE_DATE_EPOCH or an existing ratchet date is required)")
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		stdoutW, stderrW := output.stdout, output.stderr
		design := args[0]
		if err := checkIsDir(design); err != nil {
			fmt.Fprintln(stderrW, err)
			return commandExitBecause(1, err)
		}
		if implDir == "" {
			fmt.Fprintln(stderrW, "machinery_baseline: --impl is required")
			return commandExit(1)
		}
		snapshot, err := designlock.Acquire(design)
		if err != nil {
			return err
		}
		defer func() { retErr = snapshot.LogicalError(errors.Join(retErr, snapshot.Release())) }()
		sourceDesign := snapshot.SourceRoot()
		date, err = resolveBaselineDate(sourceDesign, date, os.Getenv("SOURCE_DATE_EPOCH"))
		if err != nil {
			return fmt.Errorf("machinery_baseline: %w", err)
		}
		stableImpl, err := snapshot.MaterializeExternalTree(implDir)
		if err != nil {
			return fmt.Errorf("machinery_baseline: snapshot implementation: %w", err)
		}
		defer func() { retErr = errors.Join(retErr, stableImpl.Close()) }()
		if err := snapshot.ResumeExpected("baseline", "rerun `machinery baseline` with the same arguments"); err != nil {
			return err
		}
		rep, err := gates.BuildBaseline(sourceDesign, stableImpl.Path(), date)
		if err != nil {
			return fmt.Errorf("machinery_baseline: %w", snapshot.LogicalError(err))
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

		ratchetBody, err := gates.RenderRatchet(rep.Ratchet)
		if err != nil {
			return fmt.Errorf("machinery_baseline: render %s: %w", gates.RatchetFile, err)
		}
		expected := []designlock.OutputExpectation{designlock.ExpectFile(filepath.Join(design, gates.RatchetFile), ratchetBody, 0o644)}
		if err := snapshot.PublishExpectedRooted("baseline", "rerun `machinery baseline` with the same arguments", expected, func(outputs *designlock.OutputScope) error {
			return outputs.WithRoot(design, func(root *os.Root) error {
				return artifactset.CommitRooted(design, root, map[string][]byte{gates.RatchetFile: ratchetBody})
			})
		}); err != nil {
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

func resolveBaselineDate(design, explicit, sourceDateEpoch string) (string, error) {
	if explicit = strings.TrimSpace(explicit); explicit != "" {
		parsed, err := time.Parse("2006-01-02", explicit)
		if err != nil || parsed.Format("2006-01-02") != explicit {
			return "", fmt.Errorf("--date must be a real canonical YYYY-MM-DD date")
		}
		return explicit, nil
	}
	if sourceDateEpoch = strings.TrimSpace(sourceDateEpoch); sourceDateEpoch != "" {
		seconds, err := strconv.ParseInt(sourceDateEpoch, 10, 64)
		if err != nil {
			return "", fmt.Errorf("SOURCE_DATE_EPOCH must be integer Unix seconds: %w", err)
		}
		return time.Unix(seconds, 0).UTC().Format("2006-01-02"), nil
	}
	existing, err := gates.LoadRatchet(design)
	if err != nil {
		return "", fmt.Errorf("cannot reuse existing ratchet date: %w", err)
	}
	if existing != nil && strings.TrimSpace(existing.Date) != "" {
		for _, layout := range []string{"2006-01-02", "2006-01"} {
			if parsed, parseErr := time.Parse(layout, existing.Date); parseErr == nil && parsed.Format(layout) == existing.Date {
				return existing.Date, nil
			}
		}
		return "", fmt.Errorf("existing ratchet date %q is not canonical YYYY-MM or YYYY-MM-DD", existing.Date)
	}
	return "", fmt.Errorf("a deterministic snapshot date is required: pass --date or SOURCE_DATE_EPOCH (an existing ratchet date is reused automatically)")
}
