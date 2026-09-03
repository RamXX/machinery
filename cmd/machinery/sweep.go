package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/gates"
)

// machinery sweep: the propagation sweep, productized (S18 of the dogfood systemic
// findings). Decision propagation across the attested layer was the top
// residual failure mode of the 2026-08 remediation cycle, and the working
// defence was hand-written grep-the-fact-everywhere invocations. Given a
// unit, guard, event, or knob name, sweep lists every HAND-WRITTEN design
// file that mentions it as a whole token, with context, so a conductor can
// see every home a ruling must reach. Tooling for the attested layer, not a
// gate: it renders mentions, it judges nothing.

func newSweepCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "sweep <name> <design-dir>",
		Short: "List every hand-written design file mentioning a name, for propagation review",
		Args:  cobra.ExactArgs(2),
	}
	var contextN int
	c.Flags().IntVar(&contextN, "context", 0, "print N trimmed lines around each mention")
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		return sweepRunTo(args[0], args[1], contextN, output.stdout, output.stderr)
	}
	return c
}

// sweepSkipsPath reports whether a design-relative path is a generated
// artifact (never hand-written, so never a propagation home): the same
// families the edit hook protects.
func sweepSkipsPath(rel string) bool {
	rel = filepath.ToSlash(rel)
	base := filepath.Base(rel)
	switch {
	case strings.HasSuffix(base, ".oracle.md"):
		return true
	case strings.HasPrefix(rel, "formal/") && (strings.HasSuffix(base, ".tla") || strings.HasSuffix(base, ".cfg") || strings.HasSuffix(base, ".als")):
		return true
	case strings.HasPrefix(rel, "packs/") || strings.HasPrefix(rel, "pack/"):
		return true
	case base == "ratchet.json":
		return true
	}
	return false
}

// sweepTextFile reports whether a file is worth scanning: the hand-written
// design surface is markdown, JSON, YAML, and DSL text.
func sweepTextFile(base string) bool {
	switch strings.ToLower(filepath.Ext(base)) {
	case ".md", ".json", ".yaml", ".yml", ".dsl", ".txt":
		return true
	}
	return false
}

func sweepRun(name, design string, contextN int) error {
	return sweepRunTo(name, design, contextN, stdoutW, stderrW)
}

func sweepRunTo(name, design string, contextN int, stdoutW, stderrW io.Writer) error {
	return withDesignSnapshot(design, func(snapshot string) error {
		return sweepSnapshotRunTo(name, snapshot, design, contextN, stdoutW, stderrW)
	})
}

func sweepSnapshotRunTo(name, design, displayDesign string, contextN int, stdoutW, stderrW io.Writer) error {
	if err := checkIsDir(design); err != nil {
		fmt.Fprintln(stderrW, remapSnapshotText(err.Error(), design, displayDesign))
		return commandExitBecause(1, err)
	}
	// Whole-token match, backtick-tolerant: `guardFoo` and guardFoo both hit;
	// guardFooBar does not. Word characters plus the name characters
	// themselves bound the token, so dotted event names (audit.append) and
	// snake_case knob keys match as whole names too.
	re, err := regexp.Compile(`(^|[^A-Za-z0-9_.])` + regexp.QuoteMeta(name) + `($|[^A-Za-z0-9_.])`)
	if err != nil {
		fmt.Fprintln(stderrW, "machinery_sweep: "+err.Error())
		return commandExitBecause(1, err)
	}
	type hit struct {
		line int
		text string
	}
	byFile := map[string][]hit{}
	linesByFile := map[string][]string{}
	walkErr := filepath.WalkDir(design, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(design, path)
		if rerr != nil {
			rel = path
		}
		// the design's own .machineryignore: a sweep must cover exactly what
		// the gates cover, or "every mention" means two different things
		if gates.DesignIgnores(design, rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if sweepSkipsPath(rel) || !sweepTextFile(d.Name()) {
			return nil
		}
		body, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		lines := strings.Split(string(body), "\n")
		for i, l := range lines {
			if re.MatchString(l) {
				byFile[rel] = append(byFile[rel], hit{line: i + 1, text: strings.TrimSpace(l)})
			}
		}
		if len(byFile[rel]) > 0 {
			linesByFile[rel] = lines
		}
		return nil
	})
	if walkErr != nil {
		fmt.Fprintln(stderrW, "machinery_sweep: "+walkErr.Error())
		return commandExitBecause(1, walkErr)
	}
	if len(byFile) == 0 {
		fmt.Fprintf(stdoutW, "no mentions of %s under %s (hand-written files)\n", quote(name), displayDesign)
		return nil
	}
	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)
	total := 0
	for _, f := range files {
		hits := byFile[f]
		fmt.Fprintf(stdoutW, "%s  (%d)\n", f, len(hits))
		for _, h := range hits {
			if contextN > 0 {
				lines := linesByFile[f]
				lo := max(0, h.line-1-contextN)
				hi := min(len(lines), h.line+contextN)
				for i := lo; i < hi; i++ {
					marker := " "
					if i == h.line-1 {
						marker = ">"
					}
					fmt.Fprintf(stdoutW, "  %s %d: %s\n", marker, i+1, strings.TrimSpace(lines[i]))
				}
			} else {
				fmt.Fprintf(stdoutW, "  %d: %s\n", h.line, h.text)
			}
		}
		total += len(hits)
	}
	fmt.Fprintf(stdoutW, "\n%d mention(s) of %s across %d file(s)\n", total, quote(name), len(files))
	return nil
}
