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
	c.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return oracleRun(".")
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
			return oracleRun(args[0])
		case dirs == 0:
			return oracleRunFiles(args)
		default:
			err := fmt.Errorf("oracle_gen: pass one machines directory or one-or-more *.machine.json files, not a mix")
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
			return err
		}
	}
	return c
}

func oracleRun(mdir string) error {
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
	return oracleRunFiles(files)
}

// oracleRunFiles regenerates the oracles for the named machine files. A
// machine that fails to load or fails lint is reported and SKIPPED while the
// valid machines still regenerate: one half-written file must not block a
// whole directory's regeneration (a five-agent wave was blocked by exactly
// that shape). The run exits 1 when anything was skipped, so CI still fails
// loudly; the difference is that every machine's own oracle is as fresh as
// its own source allows.
func oracleRunFiles(files []string) error {
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
	// pass 3: generate for every machine that came through clean
	for _, mc := range machines {
		if contested[mc.tag] {
			continue
		}
		out := replaceExt(mc.path, ".machine.json", ".oracle.md")
		body := oracle.Render(mc.m, mc.path)
		if err := os.WriteFile(out, []byte(body), 0644); err != nil {
			report(fmt.Sprintf("oracle_gen: %v", err))
			continue
		}
		// count transition rows: body.count('| T-')
		cnt := countSubstr(body, "| T-")
		fmt.Fprintf(stdoutW, "generated %s  (%d transition rows)\n", filepath.Base(out), cnt)
	}
	if len(failures) > 0 {
		err := fmt.Errorf("oracle_gen: %d machine(s) failed; the valid ones were regenerated", len(failures))
		fmt.Fprintln(stderrW, err)
		exitFunc(1)
		return err
	}
	return nil
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
