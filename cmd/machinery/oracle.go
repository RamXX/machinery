package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/artifactset"
	"github.com/RamXX/machinery/internal/designlock"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/lint"
	"github.com/RamXX/machinery/internal/oracle"
	"github.com/RamXX/machinery/internal/portablepath"
	"github.com/RamXX/machinery/internal/processcontrol"
)

func newOracleCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "oracle [<machines-dir> | <file.machine.json>...]",
		Short: "Regenerate transition oracles from machine JSON (a directory, or named machine files)",
		Args:  cobra.ArbitraryArgs,
	}
	diff := c.Flags().Bool("diff", false,
		"classify the churn against the committed oracles instead of writing: new, deleted, and modified stable ids, with rename-shaped pairs called out; the output IS the affected-test list of the revision protocol")
	against := c.Flags().String("against", "",
		"with --diff: read the baseline oracles at this git ref (git show <ref>:<path>) instead of from the working tree, so the affected-test list survives a regeneration that has already been written")
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		stdoutW, stderrW := output.stdout, output.stderr
		if *against != "" && !*diff {
			err := fmt.Errorf("oracle_gen: --against names a baseline for --diff; pass --diff too (without it there is nothing to compare)")
			fmt.Fprintln(stderrW, err)
			return commandExitBecause(1, err)
		}
		if len(args) == 0 {
			return oracleRunTo(".", *diff, *against, stdoutW, stderrW)
		}
		// Classify lexically so source discovery itself happens only after the
		// design snapshot lock is held. Machine files have one closed suffix;
		// every other single argument is the machines directory.
		if len(args) == 1 && !strings.HasSuffix(filepath.Base(args[0]), ".machine.json") {
			return oracleRunTo(args[0], *diff, *against, stdoutW, stderrW)
		}
		return oracleRunFilesTo(args, *diff, *against, stdoutW, stderrW)
	}
	return c
}

func oracleRun(mdir string, diff bool, against string) error {
	return oracleRunTo(mdir, diff, against, stdoutW, stderrW)
}

func oracleRunTo(mdir string, diff bool, against string, stdoutW, stderrW io.Writer) error {
	acquire := designlock.Acquire
	if diff {
		acquire = designlock.AcquireReader
	}
	snapshot, err := acquire(filepath.Dir(mdir))
	if err != nil {
		reportOracleAcquireErrorTo(err, stderrW)
		return commandExitBecause(1, err)
	}
	retErr := oracleRunInSnapshot(snapshot, mdir, diff, against, stdoutW, stderrW)
	return snapshot.LogicalError(errors.Join(retErr, snapshot.Release()))
}

func oracleRunInSnapshot(snapshot *designlock.Lock, mdir string, diff bool, against string, stdoutW, stderrW io.Writer) error {
	sourceDir, err := snapshot.SourcePath(mdir)
	if err != nil {
		return fmt.Errorf("oracle_gen: resolve immutable machine source: %w", err)
	}
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		fmt.Fprintf(stdoutW, "no *.machine.json under %s\n", mdir)
		return commandExitBecause(1, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" && strings.HasSuffix(e.Name(), ".machine.json") {
			files = append(files, filepath.Join(sourceDir, e.Name()))
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		fmt.Fprintf(stdoutW, "no *.machine.json under %s\n", mdir)
		return commandExitBecause(1, fmt.Errorf("no *.machine.json under %s", mdir))
	}
	artifactDir, err := filepath.Abs(mdir)
	if err != nil {
		return err
	}
	return oracleRunFilesInSnapshot(snapshot, files, artifactDir, diff, against, stdoutW, stderrW)
}

func oracleRunFilesTo(files []string, diff bool, against string, stdoutW, stderrW io.Writer) error {
	if len(files) == 0 {
		return oracleRunFilesInSnapshot(nil, files, "", diff, against, stdoutW, stderrW)
	}
	artifactDir, err := filepath.Abs(filepath.Dir(files[0]))
	if err != nil {
		return err
	}
	for _, file := range files[1:] {
		dir, err := filepath.Abs(filepath.Dir(file))
		if err != nil {
			return err
		}
		if dir != artifactDir {
			err := fmt.Errorf("oracle_gen: writable machine files must share one output directory for an all-or-rollback artifact transaction (%s and %s differ)", artifactDir, dir)
			fmt.Fprintln(stderrW, err)
			return commandExitBecause(1, err)
		}
	}
	acquire := designlock.Acquire
	if diff {
		acquire = designlock.AcquireReader
	}
	snapshot, err := acquire(filepath.Dir(artifactDir))
	if err != nil {
		reportOracleAcquireErrorTo(err, stderrW)
		return commandExitBecause(1, err)
	}
	sourceFiles := make([]string, 0, len(files))
	for _, file := range files {
		sourceFile, mapErr := snapshot.SourcePath(file)
		if mapErr != nil {
			return errors.Join(fmt.Errorf("oracle_gen: resolve immutable machine source: %w", mapErr), snapshot.Release())
		}
		sourceFiles = append(sourceFiles, sourceFile)
	}
	retErr := oracleRunFilesInSnapshot(snapshot, sourceFiles, artifactDir, diff, against, stdoutW, stderrW)
	return snapshot.LogicalError(errors.Join(retErr, snapshot.Release()))
}

func reportOracleAcquireErrorTo(err error, stderrW io.Writer) {
	if strings.Contains(err.Error(), "symlink") {
		fmt.Fprintln(stderrW, "oracle_gen: every selected source must be a regular machine file and every generated target must be regular, not a symlink:", err)
		return
	}
	fmt.Fprintln(stderrW, "oracle_gen:", err)
}

func oracleRunFilesInSnapshot(snapshot *designlock.Lock, files []string, artifactDir string, diff bool, against string, stdoutW, stderrW io.Writer) error {
	if snapshot != nil && !diff {
		if err := snapshot.ResumeExpected("oracle", "rerun `machinery oracle` with the same arguments"); err != nil {
			return err
		}
	}
	type mach struct {
		path string
		m    *ir.Value
		tag  string
	}
	var failures []string
	var diffReport strings.Builder
	diffOut := stdoutW
	if diff {
		diffOut = &diffReport
	}
	report := func(msg string) {
		if snapshot != nil {
			msg = snapshot.LogicalText(msg)
		}
		failures = append(failures, msg)
		fmt.Fprintln(stderrW, msg)
	}
	if len(files) == 0 {
		err := fmt.Errorf("oracle_gen: no machine files selected; no oracles were regenerated")
		fmt.Fprintln(stderrW, err)
		return commandExitBecause(1, err)
	}
	// pass 1: load + lint every named machine. A machine that fails lint must
	// not generate (the oracle would encode the defects, e.g. an array target
	// silently narrowed to its first element).
	var machines []mach
	for _, f := range files {
		base := filepath.Base(f)
		if !strings.HasSuffix(base, ".machine.json") {
			report(fmt.Sprintf("oracle_gen: %s is not a *.machine.json file", f))
			continue
		}
		stem := strings.TrimSuffix(base, ".machine.json")
		outName := stem + ".oracle.md"
		nameOK := true
		for _, candidate := range []struct {
			kind, name string
		}{
			{"machine filename", base},
			{"machine filename stem", stem},
			{"derived oracle filename", outName},
			{"derived oracle filename stem", strings.TrimSuffix(outName, ".oracle.md")},
		} {
			if err := portablepath.ValidateBase(candidate.name); err != nil {
				report(fmt.Sprintf("oracle_gen: %s %q is not portable: %v", candidate.kind, candidate.name, err))
				nameOK = false
			}
		}
		if !nameOK {
			continue
		}
		info, statErr := os.Lstat(f)
		if statErr != nil {
			report(fmt.Sprintf("oracle_gen: %s: %v", f, statErr))
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			report(fmt.Sprintf("oracle_gen: %s must be a regular machine file, not a symlink or special file", f))
			continue
		}
		m, err := ir.LoadMachineJSON(f)
		if err != nil {
			report(fmt.Sprintf("oracle_gen: %v", err))
			continue
		}
		machineOK := true
		module, moduleErr := ir.TLAModuleName(m)
		if moduleErr != nil {
			report(fmt.Sprintf("oracle_gen: %s has no canonical portable machine/module identity: %v", base, moduleErr))
			machineOK = false
		} else if !strings.EqualFold(stem, module) {
			report(fmt.Sprintf("oracle_gen: machine filename stem %q does not match canonical machine id/module identity %q", stem, module))
			machineOK = false
		}
		lintErrs, _, _, _ := lint.LintMachine(m, base)
		if len(lintErrs) > 0 {
			for _, e := range lintErrs {
				fmt.Fprintf(stderrW, "  ERROR  %s\n", e)
			}
			report(fmt.Sprintf("oracle_gen: %s fails lint (%d error(s) above); fix the machine before generating its oracle", base, len(lintErrs)))
			machineOK = false
		}
		if !machineOK {
			continue
		}
		machines = append(machines, mach{path: f, m: m, tag: oracle.Tag(m, f)})
	}
	// pass 2: stable-id tag census. Two machines with the same tag would mint
	// identical stable ids for different transitions, so every claimant of a
	// contested tag is refused, none generates. A per-file run also reserves
	// the tags of parseable directory SIBLINGS not named on the command line,
	// so regenerating one machine cannot silently take a tag a sibling owns;
	// an unparseable sibling reserves nothing and is caught on its own run.
	tagOwners := map[string][]string{}
	for _, mc := range machines {
		tagOwners[mc.tag] = append(tagOwners[mc.tag], filepath.Base(mc.path))
	}
	siblings, siblingInventoryErrs := siblingMachineFiles(files)
	for _, inventoryErr := range siblingInventoryErrs {
		report("oracle_gen: " + inventoryErr)
	}
	for _, f := range siblings {
		base := filepath.Base(f)
		stem := strings.TrimSuffix(base, ".machine.json")
		if err := portablepath.ValidateBase(base); err != nil {
			report(fmt.Sprintf("oracle_gen: sibling machine filename %q is not portable: %v", base, err))
			continue
		}
		info, statErr := os.Lstat(f)
		if statErr != nil {
			report(fmt.Sprintf("oracle_gen: inspect sibling machine %s: %v", base, statErr))
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			report(fmt.Sprintf("oracle_gen: sibling machine %s must be a regular file, not a symlink or special file", base))
			continue
		}
		m, loadErr := ir.LoadMachineJSON(f)
		if loadErr != nil {
			report(fmt.Sprintf("oracle_gen: sibling machine %s is unreadable or malformed: %v", base, loadErr))
			continue
		}
		module, moduleErr := ir.TLAModuleName(m)
		if moduleErr != nil {
			report(fmt.Sprintf("oracle_gen: sibling machine %s has no canonical portable machine/module identity: %v", base, moduleErr))
			continue
		}
		if !strings.EqualFold(stem, module) {
			report(fmt.Sprintf("oracle_gen: sibling machine filename stem %q does not match canonical machine id/module identity %q", stem, module))
			continue
		}
		t := oracle.Tag(m, f)
		tagOwners[t] = append(tagOwners[t], base)
	}
	contested := map[string]bool{}
	var contestedTags []string
	for tag, owners := range tagOwners {
		if len(owners) > 1 {
			contestedTags = append(contestedTags, tag)
		}
	}
	sort.Strings(contestedTags)
	for _, tag := range contestedTags {
		contested[tag] = true
		owners := tagOwners[tag]
		sort.Strings(owners)
		report(fmt.Sprintf("oracle_gen: stable-id tag %s is derived for %s; set _oracle_tag on all but one to disambiguate", tag, strings.Join(owners, " and ")))
	}
	// under --diff --against the baseline is the oracle set AT A GIT REF, not
	// the working tree. Resolve the repository and the ref ONCE, before any
	// classification: a bad ref must fail the whole run loudly, never produce
	// a per-file "new oracle" line that reads like real churn.
	var baseRoot, baseCommit string
	if diff && against != "" {
		root, rerr := gitRootOf(artifactDir)
		if rerr != nil {
			fmt.Fprintln(stderrW, "oracle_gen: "+rerr.Error())
			return commandExitBecause(1, rerr)
		}
		commit, verr := gitResolveCommit(root, against)
		if verr != nil {
			fmt.Fprintln(stderrW, "oracle_gen: "+verr.Error())
			return commandExitBecause(1, verr)
		}
		baseRoot, baseCommit = root, commit
		fmt.Fprintf(diffOut, "baseline: %s (resolved from %s)\n", baseCommit, against)
	}
	// pass 3: render every machine that came through clean. Under
	// --diff nothing is written: the fresh render is classified against the
	// committed oracle instead, mechanizing step 3 of the revision protocol
	// (the conductor once read a git diff and classified the churn by hand).
	churned := 0
	artifacts := map[string][]byte{}
	type generated struct {
		name string
		rows int
	}
	var generatedFiles []generated
	for _, mc := range machines {
		if contested[mc.tag] {
			continue
		}
		out := filepath.Join(artifactDir, strings.TrimSuffix(filepath.Base(mc.path), ".machine.json")+".oracle.md")
		body := oracle.Render(mc.m, mc.path)
		if diff {
			if against != "" {
				base, berr := gitShowOracle(baseRoot, baseCommit, out)
				if berr != nil {
					fmt.Fprintln(stderrW, "oracle_gen: "+berr.Error())
					return commandExitBecause(1, berr)
				}
				churned += diffOneOracleBodyTo(diffOut, filepath.Base(out), base, true, body)
				continue
			}
			lines, baselineErr := diffOneOracleTo(diffOut, snapshot, out, body)
			if baselineErr != nil {
				report("oracle_gen: read committed oracle baseline: " + baselineErr.Error())
				continue
			}
			churned += lines
			continue
		}
		name := filepath.Base(out)
		if _, duplicate := artifacts[name]; duplicate {
			report(fmt.Sprintf("oracle_gen: more than one selected machine maps to output %s", name))
			continue
		}
		artifacts[name] = []byte(body)
		generatedFiles = append(generatedFiles, generated{name: name, rows: countSubstr(body, "| T-")})
	}
	if diff && churned == 0 && len(failures) == 0 {
		if against != "" {
			fmt.Fprintf(diffOut, "no churn: every machine renders exactly what %s carries (resolved from %s)\n", baseCommit, against)
		} else {
			fmt.Fprintf(diffOut, "no churn: every committed oracle matches its machine\n")
		}
	}
	if len(failures) == 0 && !diff {
		if snapshot == nil {
			report("oracle_gen: internal error: no design snapshot for artifact publication")
		} else {
			stale, staleErr := staleOwnedOracles(artifactDir, artifacts)
			if staleErr != nil {
				report("oracle_gen: inventory stale generated oracles: " + staleErr.Error())
			}
			if staleErr == nil && len(stale) > 0 {
				oracleAfterStalePlan()
			}
			expected := make([]designlock.OutputExpectation, 0, len(artifacts)+len(stale))
			for name, body := range artifacts {
				expected = append(expected, designlock.ExpectFile(filepath.Join(artifactDir, name), body, 0o644))
			}
			for _, name := range stale {
				expected = append(expected, designlock.ExpectAbsent(filepath.Join(artifactDir, name.Name)))
			}
			err := snapshot.PublishExpectedRooted("oracle", "rerun `machinery oracle` with the same arguments", expected, func(outputs *designlock.OutputScope) error {
				return outputs.WithRoot(artifactDir, func(root *os.Root) error {
					return artifactset.ReconcilePlannedRooted(artifactDir, root, artifacts, stale)
				})
			})
			if err != nil {
				report(fmt.Sprintf("oracle_gen: commit oracle artifact set: %v", err))
			} else {
				for _, generated := range generatedFiles {
					fmt.Fprintf(stdoutW, "generated %s  (%d transition rows)\n", generated.name, generated.rows)
				}
			}
		}
	}
	if len(failures) > 0 {
		err := fmt.Errorf("oracle_gen: %d finding(s); no oracles were regenerated", len(failures))
		fmt.Fprintln(stderrW, err)
		return commandExitBecause(1, err)
	}
	if diff && snapshot != nil {
		if err := snapshot.CheckUnchanged(); err != nil {
			report("oracle_gen: " + err.Error())
			err := fmt.Errorf("oracle_gen: design changed during --diff; discard the reported churn and retry")
			fmt.Fprintln(stderrW, err)
			return commandExitBecause(1, err)
		}
		if _, err := io.WriteString(stdoutW, diffReport.String()); err != nil {
			return fmt.Errorf("oracle_gen: write diff report: %w", err)
		}
	}
	return nil
}

func staleOwnedOracles(dir string, keep map[string][]byte) (stale []artifactset.RemovalPrecondition, retErr error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	f, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	entries, readErr := f.ReadDir(-1)
	closeErr := f.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".oracle.md") {
			continue
		}
		if _, exists := keep[name]; exists {
			continue
		}
		body, condition, err := artifactset.InspectRemovalCandidate(dir, name)
		if err != nil {
			return nil, err
		}
		owned, err := canonicalOracleOwner(name, body)
		if err != nil {
			return nil, err
		}
		if owned {
			stale = append(stale, condition)
		}
	}
	return stale, nil
}

func canonicalOracleOwner(name string, body []byte) (bool, error) {
	lines := bytes.SplitN(body, []byte("\n"), 6)
	if len(lines) < 4 || !bytes.HasPrefix(lines[0], []byte("# Generated transition oracle: `")) || !bytes.HasSuffix(lines[0], []byte("`")) || len(lines[1]) != 0 {
		return false, nil
	}
	title := strings.TrimSuffix(strings.TrimPrefix(string(lines[0]), "# Generated transition oracle: `"), "`")
	const sourcePrefix = "Generated from `"
	const sourceSuffix = "` by `machinery oracle`. DO NOT EDIT BY HAND."
	sourceLine := string(lines[2])
	if !strings.HasPrefix(sourceLine, sourcePrefix) || !strings.HasSuffix(sourceLine, sourceSuffix) || !bytes.HasPrefix(lines[3], []byte("<!-- machinery-version: ")) || !bytes.HasSuffix(lines[3], []byte(" -->")) {
		return false, nil
	}
	source := strings.TrimSuffix(strings.TrimPrefix(sourceLine, sourcePrefix), sourceSuffix)
	if filepath.Base(source) != source || !strings.HasSuffix(source, ".machine.json") {
		return true, fmt.Errorf("stale oracle %s has invalid generated source owner", name)
	}
	if err := portablepath.ValidateBase(source); err != nil {
		return true, fmt.Errorf("stale oracle %s has non-portable source owner: %w", name, err)
	}
	want := strings.TrimSuffix(source, ".machine.json") + ".oracle.md"
	if name != want {
		return true, fmt.Errorf("stale oracle %s does not match generated source owner %s", name, source)
	}
	if !oracleMachineID(title) || ir.Title(title) != strings.TrimSuffix(source, ".machine.json") {
		return true, fmt.Errorf("stale oracle %s title identity does not match source owner %s", name, source)
	}
	return true, nil
}

func oracleMachineID(value string) bool {
	if value == "" || !oracleIDLetter(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		ch := value[i]
		if !oracleIDLetter(ch) && (ch < '0' || ch > '9') && ch != '_' {
			return false
		}
	}
	return true
}

func oracleIDLetter(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

// oracleDiffRow is one transition-oracle row for churn classification: the
// sequential test id, and the row's remaining cells joined (the sequential id
// is EXCLUDED from content comparison, because renumbering is not an
// expectation change).
type oracleDiffRow struct {
	testID  string
	content string
}

// oracleDiffRows parses an oracle body's transition tables into rows keyed by
// stable id.
func oracleDiffRows(body string) map[string]oracleDiffRow {
	out := map[string]oracleDiffRow{}
	for _, tbl := range ir.ParseMdTables(body) {
		si := ir.FindCol(tbl.Header, "stable id")
		ti := ir.FindCol(tbl.Header, "test id")
		if si < 0 {
			continue
		}
		for _, r := range tbl.Rows {
			if si >= len(r) {
				continue
			}
			id := strings.TrimSpace(r[si])
			if id == "" || id == "-" {
				continue
			}
			var rest []string
			testID := ""
			for i, c := range r {
				switch i {
				case si:
				case ti:
					testID = strings.TrimSpace(c)
				default:
					rest = append(rest, strings.Join(strings.Fields(c), " "))
				}
			}
			out[id] = oracleDiffRow{testID: testID, content: strings.Join(rest, "|")}
		}
	}
	return out
}

// diffOneOracle classifies the churn between the committed oracle at path and
// the fresh render, printing the affected-test list: new ids, deleted ids,
// modified rows, and rename-shaped pairs (a deleted id whose row content
// matches a new id's exactly, the id churn a rename produces with no
// behavioral change). Returns the number of churn lines printed.
func diffOneOracleTo(out io.Writer, snapshot *designlock.Lock, path, fresh string) (int, error) {
	committedBody, haveBaseline, err := readOracleDiffBaseline(snapshot, path)
	if err != nil {
		return 0, err
	}
	return diffOneOracleBodyTo(out, filepath.Base(path), string(committedBody), haveBaseline, fresh), nil
}

func readOracleDiffBaseline(snapshot *designlock.Lock, path string) (_ []byte, present bool, retErr error) {
	if snapshot == nil {
		return nil, false, fmt.Errorf("no held design snapshot")
	}
	source, err := snapshot.SourcePath(path)
	if err != nil {
		return nil, false, err
	}
	dirPath := filepath.Dir(source)
	dirBefore, err := os.Lstat(dirPath)
	if err != nil {
		return nil, false, err
	}
	if dirBefore.Mode()&os.ModeSymlink != 0 || !dirBefore.IsDir() {
		return nil, false, fmt.Errorf("oracle baseline directory must be a real directory")
	}
	root, err := os.OpenRoot(dirPath)
	if err != nil {
		return nil, false, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	dirOpened, err := root.Stat(".")
	if err != nil || !os.SameFile(dirBefore, dirOpened) {
		return nil, false, errors.Join(err, fmt.Errorf("oracle baseline directory changed identity while opening"))
	}
	name := filepath.Base(source)
	before, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, false, fmt.Errorf("oracle baseline %s must be a regular non-symlink file", name)
	}
	if before.Mode().Perm()&0o444 == 0 {
		return nil, false, fmt.Errorf("oracle baseline %s is not readable by policy", name)
	}
	oracleAfterDiffBaselineInspect(source)
	file, err := root.Open(name)
	if err != nil {
		return nil, false, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) || before.Mode() != opened.Mode() {
		return nil, false, errors.Join(err, fmt.Errorf("oracle baseline %s changed identity or mode while opening", name))
	}
	body, err := io.ReadAll(io.LimitReader(file, oracleDiffBaselineLimit+1))
	if err != nil {
		return nil, false, err
	}
	if len(body) > oracleDiffBaselineLimit {
		return nil, false, fmt.Errorf("oracle baseline %s exceeds %d bytes", name, oracleDiffBaselineLimit)
	}
	after, err := root.Lstat(name)
	if err != nil || !os.SameFile(opened, after) || opened.Mode() != after.Mode() || opened.Size() != after.Size() {
		return nil, false, errors.Join(err, fmt.Errorf("oracle baseline %s changed while reading", name))
	}
	dirAfter, err := os.Lstat(dirPath)
	if err != nil || !os.SameFile(dirOpened, dirAfter) {
		return nil, false, errors.Join(err, fmt.Errorf("oracle baseline directory changed while reading"))
	}
	return body, true, nil
}

// diffOneOracleBody classifies the churn between a baseline oracle body and
// the fresh render. The baseline is the working-tree file under plain --diff
// and the file AT A GIT REF under --against; the classification is identical,
// which is the point: the affected-test list must not depend on whether the
// author has already written the regeneration.
func diffOneOracleBodyTo(out io.Writer, name, committedBody string, haveBaseline bool, fresh string) int {
	if !haveBaseline {
		fmt.Fprintf(out, "== %s\n  new oracle: not committed yet; every row is a new test\n", name)
		return 1
	}
	committed := oracleDiffRows(committedBody)
	freshRows := oracleDiffRows(fresh)
	var added, deleted, modified []string
	for id := range freshRows {
		if _, ok := committed[id]; !ok {
			added = append(added, id)
		}
	}
	for id, oldRow := range committed {
		newRow, ok := freshRows[id]
		if !ok {
			deleted = append(deleted, id)
			continue
		}
		if newRow.content != oldRow.content {
			modified = append(modified, id)
		}
	}
	sort.Strings(added)
	sort.Strings(deleted)
	sort.Strings(modified)
	// rename-shaped churn: identical content under a new id. Ambiguous
	// matches (several deleted rows with the same content) stay listed as
	// plain delete+new; a guessed pairing would be worse than none.
	renameOf := map[string]string{}
	contentOfAdded := map[string][]string{}
	for _, id := range added {
		contentOfAdded[freshRows[id].content] = append(contentOfAdded[freshRows[id].content], id)
	}
	for _, id := range deleted {
		if candidates := contentOfAdded[committed[id].content]; len(candidates) == 1 {
			if _, taken := renameOf[candidates[0]]; !taken {
				renameOf[candidates[0]] = id
			}
		}
	}
	lines := 0
	emit := func(format string, a ...any) {
		if lines == 0 {
			fmt.Fprintf(out, "== %s\n", name)
		}
		lines++
		fmt.Fprintf(out, format, a...)
	}
	for _, id := range added {
		if old, ok := renameOf[id]; ok {
			emit("  rename-shaped  %s -> %s (identical row content; record the old-id to new-id mapping, never delete-all-plus-new)\n", old, id)
			continue
		}
		emit("  new       %s (%s): a new test\n", id, freshRows[id].testID)
	}
	for _, id := range deleted {
		renamed := false
		for _, old := range renameOf {
			if old == id {
				renamed = true
				break
			}
		}
		if !renamed {
			emit("  deleted   %s: its test is retired (add the id to removed-ids.txt if it stays cited)\n", id)
		}
	}
	for _, id := range modified {
		emit("  modified  %s (%s): the expectation changed; regenerate its test\n", id, freshRows[id].testID)
	}
	return lines
}

// siblingMachineFiles returns the *.machine.json files that share a directory
// with any named file but are not themselves named: the tag-reservation set
// for a per-file run. A whole-directory run names every file, so this returns
// nothing there.
func siblingMachineFiles(named []string) (files, problems []string) {
	inNamed := map[string]bool{}
	dirs := map[string]bool{}
	for _, f := range named {
		abs, err := filepath.Abs(f)
		if err != nil {
			abs = f
		}
		inNamed[abs] = true
		dirs[filepath.Dir(abs)] = true
	}
	var orderedDirs []string
	for d := range dirs {
		orderedDirs = append(orderedDirs, d)
	}
	sort.Strings(orderedDirs)
	for _, d := range orderedDirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			problems = append(problems, fmt.Sprintf("read sibling machine inventory %s: %v", d, err))
			continue
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".machine.json") {
				continue
			}
			p := filepath.Join(d, e.Name())
			if !inNamed[p] {
				files = append(files, p)
			}
		}
	}
	sort.Strings(files)
	sort.Strings(problems)
	return files, problems
}

func countSubstr(s, sub string) int {
	n := 0
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			n++
		}
	}
	return n
}

// --- the --against baseline: oracle bodies read at a git ref ---

// gitBaselineTimeout bounds every git invocation the baseline reader makes.
// A hung git must fail the run, not stall a CI job forever.
const gitBaselineTimeout = 20 * time.Second

// gitBaselineOutputLimit bounds both stdout and stderr independently. Oracle
// baselines are generated text, but a corrupted repository or hostile git
// wrapper must not make the CLI allocate without bound.
const gitBaselineOutputLimit = 16 << 20

const oracleDiffBaselineLimit = 8 << 20

var oracleAfterDiffBaselineInspect = func(string) {}

type gitBoundedBuffer struct {
	buf      bytes.Buffer
	exceeded bool
}

func (b *gitBoundedBuffer) Write(p []byte) (int, error) {
	remaining := gitBaselineOutputLimit - b.buf.Len()
	if remaining > 0 {
		if remaining > len(p) {
			remaining = len(p)
		}
		_, _ = b.buf.Write(p[:remaining])
	}
	if remaining < len(p) {
		b.exceeded = true
	}
	// Report the whole write consumed while discarding bytes past the cap. This
	// lets the child terminate normally; the caller then fails closed on the
	// exceeded bit without retaining the discarded output.
	return len(p), nil
}

func (b *gitBoundedBuffer) String() string { return b.buf.String() }

func gitRun(root string, args ...string) (stdout, stderr string, retErr error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitBaselineTimeout)
	defer cancel()
	cmdArgs := append([]string{"-C", root}, args...)
	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	cmd.Env = oracleGitEnvironment()
	var out, errOut gitBoundedBuffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := processcontrol.Run(ctx, cmd)
	if ctx.Err() != nil {
		return "", errOut.String(), fmt.Errorf("git command timed out after %s: %w", gitBaselineTimeout, ctx.Err())
	}
	if out.exceeded || errOut.exceeded {
		return "", errOut.String(), fmt.Errorf("git command output exceeded %d bytes", gitBaselineOutputLimit)
	}
	if err == nil && strings.TrimSpace(errOut.String()) != "" {
		return "", errOut.String(), fmt.Errorf("git command emitted unexpected stderr despite exiting successfully")
	}
	return out.String(), errOut.String(), err
}

func oracleGitEnvironment() []string {
	// Git has a large environment control surface: repository/worktree/object
	// redirects, replacement refs, config injection, tracing, and prompts all
	// override -C. Start from a closed process environment and retain only the
	// OS variables needed to locate and execute git itself.
	allowed := map[string]bool{
		"PATH": true, "PATHEXT": true, "SYSTEMROOT": true, "SYSTEMDRIVE": true,
		"COMSPEC": true, "WINDIR": true, "TMPDIR": true, "TMP": true, "TEMP": true,
	}
	env := make([]string, 0, len(allowed)+12)
	for _, item := range os.Environ() {
		key, _, ok := strings.Cut(item, "=")
		if ok && allowed[strings.ToUpper(key)] {
			env = append(env, item)
		}
	}
	sort.Strings(env)
	return append(env,
		"GIT_CONFIG_COUNT=0",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_NO_LAZY_FETCH=1",
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_PAGER=cat",
		"GIT_TERMINAL_PROMPT=0",
		"LANG=C",
		"LC_ALL=C",
	)
}

// gitRootOf resolves the repository that owns dir, failing loudly when there
// is none: --against reads history, and a design outside a repository has
// none to read.
func gitRootOf(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("cannot resolve %s: %w", dir, err)
	}
	out, stderr, err := gitRun(abs, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("--against needs a git repository, and %s is not inside one (git: %s)", abs, gitFailureMessage(err, stderr))
	}
	return realPath(strings.TrimSpace(out)), nil
}

// realPath resolves symlinks so a path and the repository root git reports are
// comparable. On macOS a temporary directory is /var/... to the process and
// /private/var/... to git, and a repository-relative path computed across that
// difference is a "../../.." escape git refuses.
func realPath(p string) string {
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return p
}

// gitResolveCommit pins a possibly moving ref to one immutable commit object.
func gitResolveCommit(root, ref string) (string, error) {
	out, stderr, err := gitRun(root, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	if err != nil {
		what := "git ref " + ir.Repr(ref) + " does not resolve in " + root
		if msg := gitFailureMessage(err, stderr); msg != "" {
			return "", fmt.Errorf("%s; git says: %s", what, msg)
		}
		return "", errors.New(what)
	}
	commit := strings.TrimSpace(out)
	if (len(commit) != 40 && len(commit) != 64) || strings.IndexFunc(commit, func(r rune) bool {
		return (r < '0' || r > '9') && (r < 'a' || r > 'f')
	}) >= 0 {
		return "", fmt.Errorf("git ref %s resolved to a noncanonical commit id %q", ir.Repr(ref), commit)
	}
	return commit, nil
}

// gitShowText runs `git show <commit>:<rel>`. Callers must resolve a moving
// user ref exactly once with gitResolveCommit before reading any paths.
func gitShowText(root, ref, rel string) (string, error) {
	if rel == "" {
		return gitResolveCommit(root, ref)
	}
	what := "no " + rel + " at " + ref + " (the path did not exist there)"
	out, stderr, err := gitRun(root, "show", ref+":"+rel)
	if err != nil {
		if msg := gitFailureMessage(err, stderr); msg != "" {
			return "", fmt.Errorf("%s; git says: %s", what, msg)
		}
		return "", errors.New(what)
	}
	return out, nil
}

var gitOracleReadHook = func() {}

var oracleAfterStalePlan = func() {}

// gitShowOracle reads one oracle file as it stood at ref.
func gitShowOracle(root, ref, path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve %s: %w", path, err)
	}
	rel, err := filepath.Rel(root, realPath(filepath.Dir(abs)))
	if err == nil {
		rel = filepath.Join(rel, filepath.Base(abs))
	}
	if err != nil {
		return "", fmt.Errorf("%s is not inside the repository at %s", path, root)
	}
	body, err := gitShowText(root, ref, filepath.ToSlash(rel))
	if err == nil {
		gitOracleReadHook()
	}
	return body, err
}

// gitMessage renders git's own stderr when it wrote any, so a finding says
// what git said instead of only "exit status 128". An empty message means git
// failed silently (`rev-parse --quiet` does), and the caller's own sentence is
// then the whole finding.
func gitMessage(err error) string {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return strings.TrimSpace(string(ee.Stderr))
	}
	return strings.TrimSpace(err.Error())
}

func gitFailureMessage(err error, stderr string) string {
	if msg := strings.TrimSpace(stderr); msg != "" {
		return msg
	}
	return gitMessage(err)
}
