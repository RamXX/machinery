package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/lint"
	"github.com/RamXX/machinery/internal/oracle"
)

func newOracleCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "oracle <machines-dir>",
		Short: "Regenerate transition oracles from machine JSON",
		Args:  cobra.MaximumNArgs(1),
	}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		mdir := "."
		if len(args) > 0 {
			mdir = args[0]
		}
		return oracleRun(mdir)
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
	fail := func(err error) error {
		fmt.Fprintln(stderrW, err)
		exitFunc(1)
		return err
	}
	// pass 1: load + lint every machine, and collect stable-id tags. A machine
	// that fails lint must not generate (the oracle would encode the defects,
	// e.g. an array target silently narrowed to its first element), and two
	// machines with the same tag would mint identical stable ids for
	// different transitions.
	type mach struct {
		path string
		m    *ir.Value
		tag  string
	}
	var machines []mach
	tagOwner := map[string]string{}
	for _, f := range files {
		m, err := ir.LoadMachineJSON(f)
		if err != nil {
			return fail(fmt.Errorf("oracle_gen: %w", err))
		}
		base := filepath.Base(f)
		lintErrs, _, _, _ := lint.LintMachine(m, base)
		if len(lintErrs) > 0 {
			for _, e := range lintErrs {
				fmt.Fprintf(stderrW, "  ERROR  %s\n", e)
			}
			return fail(fmt.Errorf("oracle_gen: %s fails lint (%d error(s) above); fix the machine before generating its oracle", base, len(lintErrs)))
		}
		tag := oracle.Tag(m, f)
		if prev, dup := tagOwner[tag]; dup {
			return fail(fmt.Errorf("oracle_gen: stable-id tag %s is derived for both %s and %s; set _oracle_tag on one of them to disambiguate", tag, prev, base))
		}
		tagOwner[tag] = base
		machines = append(machines, mach{path: f, m: m, tag: tag})
	}
	// pass 2: everything is clean; generate
	for _, mc := range machines {
		out := replaceExt(mc.path, ".machine.json", ".oracle.md")
		body := oracle.Render(mc.m, mc.path)
		if err := os.WriteFile(out, []byte(body), 0644); err != nil {
			return fail(fmt.Errorf("oracle_gen: %w", err))
		}
		// count transition rows: body.count('| T-')
		cnt := countSubstr(body, "| T-")
		fmt.Fprintf(stdoutW, "generated %s  (%d transition rows)\n", filepath.Base(out), cnt)
	}
	return nil
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
