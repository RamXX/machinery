package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/lint"
	"github.com/RamXX/machinery/internal/oracle"
)

func newOracleCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "oracle [<machines-dir> | <file.machine.json>...]",
		Short: "Regenerate transition oracles from machine JSON (a directory, or named machine files)",
		Args:  cobra.ArbitraryArgs,
	}
	diff := c.Flags().Bool("diff", false,
		"classify the churn against the committed oracles instead of writing: new, deleted, and modified stable ids, with rename-shaped pairs called out; the output IS the affected-test list of the revision protocol")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return oracleRun(".", *diff)
		}
		// One directory, or one-or-more machine files; never a mix, so a typo
		// in a file name cannot silently widen into a directory sweep.
		dirs := 0
		for _, a := range args {
			if fi, err := os.Stat(a); err == nil && fi.IsDir() {
				dirs++
			}
		}
		switch {
		case dirs == 1 && len(args) == 1:
			return oracleRun(args[0], *diff)
		case dirs == 0:
			return oracleRunFiles(args, *diff)
		default:
			err := fmt.Errorf("oracle_gen: pass one machines directory or one-or-more *.machine.json files, not a mix")
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
			return err
		}
	}
	return c
}

func oracleRun(mdir string, diff bool) error {
	entries, err := os.ReadDir(mdir)
	if err != nil {
		fmt.Fprintf(stdoutW, "no *.machine.json under %s\n", mdir)
		exitFunc(1)
		return err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" && hasSuffix(e.Name(), ".machine.json") {
			files = append(files, filepath.Join(mdir, e.Name()))
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		fmt.Fprintf(stdoutW, "no *.machine.json under %s\n", mdir)
		exitFunc(1)
		return fmt.Errorf("no *.machine.json under %s", mdir)
	}
	return oracleRunFiles(files, diff)
}

// oracleRunFiles regenerates the oracles for the named machine files. A
// machine that fails to load or fails lint is reported and SKIPPED while the
// valid machines still regenerate: one half-written file must not block a
// whole directory's regeneration (a five-agent wave was blocked by exactly
// that shape). The run exits 1 when anything was skipped, so CI still fails
// loudly; the difference is that every machine's own oracle is as fresh as
// its own source allows.
func oracleRunFiles(files []string, diff bool) error {
	type mach struct {
		path string
		m    *ir.Value
		tag  string
	}
	var failures []string
	report := func(msg string) {
		failures = append(failures, msg)
		fmt.Fprintln(stderrW, msg)
	}
	// pass 1: load + lint every named machine. A machine that fails lint must
	// not generate (the oracle would encode the defects, e.g. an array target
	// silently narrowed to its first element).
	var machines []mach
	for _, f := range files {
		if !hasSuffix(filepath.Base(f), ".machine.json") {
			report(fmt.Sprintf("oracle_gen: %s is not a *.machine.json file", f))
			continue
		}
		m, err := ir.LoadMachineJSON(f)
		if err != nil {
			report(fmt.Sprintf("oracle_gen: %v", err))
			continue
		}
		base := filepath.Base(f)
		lintErrs, _, _, _ := lint.LintMachine(m, base)
		if len(lintErrs) > 0 {
			for _, e := range lintErrs {
				fmt.Fprintf(stderrW, "  ERROR  %s\n", e)
			}
			report(fmt.Sprintf("oracle_gen: %s fails lint (%d error(s) above); fix the machine before generating its oracle", base, len(lintErrs)))
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
	for _, f := range siblingMachineFiles(files) {
		if m, err := ir.LoadMachineJSON(f); err == nil {
			t := oracle.Tag(m, f)
			tagOwners[t] = append(tagOwners[t], filepath.Base(f))
		}
	}
	contested := map[string]bool{}
	for tag, owners := range tagOwners {
		if len(owners) > 1 {
			contested[tag] = true
			sort.Strings(owners)
			report(fmt.Sprintf("oracle_gen: stable-id tag %s is derived for %s; set _oracle_tag on all but one to disambiguate", tag, strings.Join(owners, " and ")))
		}
	}
	// pass 3: generate for every machine that came through clean. Under
	// --diff nothing is written: the fresh render is classified against the
	// committed oracle instead, mechanizing step 3 of the revision protocol
	// (the conductor once read a git diff and classified the churn by hand).
	churned := 0
	for _, mc := range machines {
		if contested[mc.tag] {
			continue
		}
		out := replaceExt(mc.path, ".machine.json", ".oracle.md")
		body := oracle.Render(mc.m, mc.path)
		if diff {
			churned += diffOneOracle(out, body)
			continue
		}
		if err := os.WriteFile(out, []byte(body), 0644); err != nil {
			report(fmt.Sprintf("oracle_gen: %v", err))
			continue
		}
		// count transition rows: body.count('| T-')
		cnt := countSubstr(body, "| T-")
		fmt.Fprintf(stdoutW, "generated %s  (%d transition rows)\n", filepath.Base(out), cnt)
	}
	if diff && churned == 0 && len(failures) == 0 {
		fmt.Fprintf(stdoutW, "no churn: every committed oracle matches its machine\n")
	}
	if len(failures) > 0 {
		err := fmt.Errorf("oracle_gen: %d machine(s) failed; the valid ones were regenerated", len(failures))
		fmt.Fprintln(stderrW, err)
		exitFunc(1)
		return err
	}
	return nil
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
func diffOneOracle(path, fresh string) int {
	name := filepath.Base(path)
	committedBody, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stdoutW, "== %s\n  new oracle: not committed yet; every row is a new test\n", name)
		return 1
	}
	committed := oracleDiffRows(string(committedBody))
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
			fmt.Fprintf(stdoutW, "== %s\n", name)
		}
		lines++
		fmt.Fprintf(stdoutW, format, a...)
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
func siblingMachineFiles(named []string) []string {
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
	var out []string
	for d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !hasSuffix(e.Name(), ".machine.json") {
				continue
			}
			p := filepath.Join(d, e.Name())
			if !inNamed[p] {
				out = append(out, p)
			}
		}
	}
	sort.Strings(out)
	return out
}

func hasSuffix(s, suf string) bool { return len(s) >= len(suf) && s[len(s)-len(suf):] == suf }

func replaceExt(path, oldExt, newExt string) string {
	return path[:len(path)-len(oldExt)] + newExt
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
