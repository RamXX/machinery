package main

import (
	"errors"
	"fmt"
	"io"
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
		Use:   "baseline <design-dir> --impl <dir> [--gate g4,gy,gl] [--grow]",
		Short: "Record the adoption debt snapshot in design/ratchet.json (G4 boundary edges; Gy-rules and Gl-ledger findings on request)",
		Long: `Scan the implementation exactly as G4-import does, print the baseline: rules
that would tolerate today's violating edges (paste them into the Architecture
Contract's dependency_rules after review), and write design/ratchet.json, the
set-based snapshot of every tolerated edge's offender files. From then on G4
fails when a baselined edge gains a new offender file, and machinery host
adapters with blocking stop hooks reject import findings at turn end (the
snapshot is what arms that blocking). Rerunning baseline rewrites ratchet.json
and may accept newly added offender files, even when no new dependency rules
are proposed. Review ratchet changes before adopting them as accepted debt.
Rerun after burning down debt to tighten the ratchet.

--gate selects what is recorded: g4 (the default, as above), gy (the Gy-rules
findings) and gl (the Gl-ledger undeclared-fact warnings), comma separated.
--impl is required only with g4; for gy it adds the implementation's oracle
bindings, exactly as machinery check --impl does. A recorded Gy/Gl finding
reports as a baselined NOTE and never blocks; a new one reports as before; a
recorded one that disappears is a resolved NOTE. The first gy (or gl) run
records every current finding; later runs keep only recorded findings still
observed, so that part of the ratchet only shrinks, unless --grow accepts the
new ones as debt. Gy refuses to record while the design has projection
errors: a row the rules cannot read is a broken design, not debt. A run
rewrites only the sections it records and keeps the others.`,
		Args: cobra.ExactArgs(1),
	}
	var implDir, date, gateList string
	var grow bool
	c.Flags().StringVar(&implDir, "impl", "", "implementation directory to scan (required with g4)")
	c.Flags().StringVar(&gateList, "gate", "g4", "comma list of the debt to record: g4 (boundary edges), gy (Gy-rules findings), gl (Gl-ledger undeclared-fact warnings)")
	c.Flags().BoolVar(&grow, "grow", false, "gy/gl: also record findings that are new since the last baseline (otherwise the recorded set only shrinks)")
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
		sel, err := parseBaselineGates(gateList)
		if err != nil {
			fmt.Fprintf(stderrW, "machinery_baseline: %s\n", err)
			return commandExitBecause(1, err)
		}
		if grow && !sel.gy && !sel.gl {
			fmt.Fprintln(stderrW, "machinery_baseline: --grow applies to --gate gy and gl; a g4 rerun always re-snapshots the observed edges")
			return commandExit(1)
		}
		if sel.g4 && implDir == "" {
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
		stableImplPath := ""
		if implDir != "" {
			stableImpl, err := snapshot.MaterializeExternalTree(implDir)
			if err != nil {
				return fmt.Errorf("machinery_baseline: snapshot implementation: %w", err)
			}
			defer func() { retErr = errors.Join(retErr, stableImpl.Close()) }()
			stableImplPath = stableImpl.Path()
		}
		if err := snapshot.ResumeExpected("baseline", "rerun `machinery baseline` with the same arguments"); err != nil {
			return err
		}
		// the recorded sections this run does not rewrite are kept; a run
		// that records gy or gl must be able to read them
		prior, priorErr := gates.LoadRatchet(sourceDesign)
		if priorErr != nil && (sel.gy || sel.gl) {
			return fmt.Errorf("machinery_baseline: %w; fix or remove it before recording consistency debt", snapshot.LogicalError(priorErr))
		}
		var ratchet *gates.Ratchet
		if sel.g4 {
			rep, err := gates.BuildBaseline(sourceDesign, stableImplPath, date)
			if err != nil {
				return fmt.Errorf("machinery_baseline: %w", snapshot.LogicalError(err))
			}
			printEdgeBaseline(stdoutW, rep, date)
			ratchet = rep.Ratchet
			if prior != nil {
				ratchet.Rules, ratchet.Undeclared = prior.Rules, prior.Undeclared
			}
		} else if prior != nil {
			// the date is the G4 snapshot's; a gy/gl-only run keeps it
			ratchet = prior
		} else {
			ratchet = &gates.Ratchet{Date: date}
		}
		var recs []gates.DebtRecord
		if sel.gy || sel.gl {
			recs, err = gates.RecordConsistencyDebt(sourceDesign, stableImplPath, ratchet, sel.gy, sel.gl, grow)
			if err != nil {
				return fmt.Errorf("machinery_baseline: %w", snapshot.LogicalError(err))
			}
			if sel.g4 {
				fmt.Fprintln(stdoutW)
			}
			printConsistencyBaseline(stdoutW, recs)
		}

		ratchetBody, err := gates.RenderRatchet(ratchet)
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
		if sel.g4 {
			total := 0
			for _, files := range ratchet.Edges {
				total += len(files)
			}
			fmt.Fprintf(stdoutW, "\nwrote %s/%s: %d edge(s), %d offender file(s)\n", design, gates.RatchetFile, len(ratchet.Edges), total)
			fmt.Fprintln(stdoutW, "rerunning baseline rewrites ratchet.json and may accept newly added offender files, even when no new dependency rules are proposed. Review ratchet changes before adopting them as accepted debt.")
			fmt.Fprintln(stdoutW, "armed: G4 now fails when a baselined edge gains a new offender file, and the machinery plugin blocks import findings at turn end")
		}
		if len(recs) > 0 {
			occurrences := 0
			for _, u := range ratchet.Undeclared {
				occurrences += u.Count
			}
			fmt.Fprintf(stdoutW, "\nwrote %s/%s: %d Gy-rules finding(s), %d Gl-ledger undeclared-fact warning(s) baselined\n", design, gates.RatchetFile, len(ratchet.Rules), occurrences)
			fmt.Fprintln(stdoutW, "armed: a baselined finding reports as a note and never blocks; a new one reports as before; rerun this command after fixing findings to shrink the ratchet (it grows only with --grow)")
		}
		return nil
	}
	return c
}

// baselineSelection is what one baseline run records.
type baselineSelection struct{ g4, gy, gl bool }

func parseBaselineGates(list string) (baselineSelection, error) {
	var sel baselineSelection
	for _, tok := range strings.Split(strings.ToLower(list), ",") {
		switch strings.TrimSpace(tok) {
		case "g4":
			sel.g4 = true
		case "gy":
			sel.gy = true
		case "gl":
			sel.gl = true
		case "":
			return sel, fmt.Errorf("--gate %q contains an empty gate name", list)
		default:
			return sel, fmt.Errorf("--gate %q: baseline records g4, gy and gl only, not %q", list, strings.TrimSpace(tok))
		}
	}
	return sel, nil
}

// printConsistencyBaseline reports what a gy/gl run recorded, per gate.
func printConsistencyBaseline(w io.Writer, recs []gates.DebtRecord) {
	fmt.Fprintln(w, "== baseline  consistency debt snapshot ==")
	for _, r := range recs {
		what := "finding(s)"
		if r.Gate == "Gl-ledger" {
			what = "undeclared-fact warning(s)"
		}
		line := fmt.Sprintf("  %s: %d %s observed; %d recorded", r.Gate, r.Observed, what, r.Recorded)
		if r.First {
			line += " (first recording)"
		}
		if r.Dropped > 0 {
			line += fmt.Sprintf("; %d resolved and dropped", r.Dropped)
		}
		if r.NotRecorded > 0 {
			line += fmt.Sprintf("; %d not recorded (new since the last baseline, still blocking; --grow accepts them as debt)", r.NotRecorded)
		}
		fmt.Fprintln(w, line)
	}
}

// printEdgeBaseline prints the G4 part of the report: the proposed rules, the
// ignore suggestions, and the imports that map to no boundary.
func printEdgeBaseline(stdoutW io.Writer, rep *gates.BaselineReport, date string) {
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
		fmt.Fprintln(stdoutW, "\nno new baseline dependency rules proposed")
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
