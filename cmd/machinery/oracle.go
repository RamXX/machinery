package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/ramirosalas/machinery/internal/oracle"
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
	for _, f := range files {
		out := replaceExt(f, ".machine.json", ".oracle.md")
		body, err := oracle.Generate(f)
		if err != nil {
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
			return err
		}
		if err := os.WriteFile(out, []byte(body), 0644); err != nil {
			fmt.Fprintf(stderrW, "oracle_gen: %s\n", err)
			exitFunc(1)
			return err
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
