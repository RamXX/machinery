package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/gates"
)

func newEmbedCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "embed",
		Short: "Declared table copies: the write half of the Ge-embed gate",
	}

	refresh := &cobra.Command{
		Use:   "refresh <design>",
		Short: "Re-copy every machinery:embed table from its source table, row by row",
		Long: "Re-copy every `machinery:embed` table under <design> from the source table it names.\n\n" +
			"Rows are matched to their source by key, not by position. A row carrying\n" +
			"'(shard-local: <reason>)' is left exactly as it is. A row with no source row is\n" +
			"REPORTED and kept, never deleted: a rename or a retirement is a judgment. A\n" +
			"marker claiming `complete` under a `where=` selector also APPENDS every selected\n" +
			"source row the copy lacks. The run is deterministic and idempotent: running it\n" +
			"twice changes nothing the second time.",
		Args: cobra.ExactArgs(1),
	}
	dryRun := refresh.Flags().Bool("dry-run", false,
		"report what would be re-copied and appended without writing any file")
	refresh.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		return embedRefreshRunTo(args[0], *dryRun, output.stdout, output.stderr)
	}

	c.AddCommand(refresh)
	return c
}

func embedRefreshRun(design string, dryRun bool) error {
	return embedRefreshRunTo(design, dryRun, stdoutW, stderrW)
}

func embedRefreshRunTo(design string, dryRun bool, stdoutW, stderrW io.Writer) error {
	reports, changed, err := gates.RefreshEmbeds(design, dryRun)
	if err != nil {
		fmt.Fprintln(stderrW, "embed refresh: "+err.Error())
		return commandExitBecause(1, err)
	}
	if len(reports) == 0 {
		fmt.Fprintf(stdoutW, "no machinery:embed markers under %s\n", design)
		return commandExitBecause(1, fmt.Errorf("no machinery:embed markers under %s", design))
	}
	gates.SortReports(reports)
	verb := "re-copied"
	if dryRun {
		verb = "would be re-copied"
	}
	recopied, appended, problems := 0, 0, 0
	for _, r := range reports {
		where := r.File + ": embed of " + r.From + " [" + r.Table + "]"
		if r.Problem != "" {
			problems++
			fmt.Fprintf(stdoutW, "%s: skipped: %s\n", where, r.Problem)
			continue
		}
		recopied += r.Recopied
		appended += r.Appended
		if r.Recopied > 0 || r.Appended > 0 {
			fmt.Fprintf(stdoutW, "%s: %d rows %s, %d appended\n", where, r.Recopied, verb, r.Appended)
		}
		if len(r.Ambiguous) > 0 {
			fmt.Fprintf(stdoutW, "%s: %d row(s) key ambiguously: %s (several source rows share the key and none is a byte copy; re-copy those by hand)\n",
				where, len(r.Ambiguous), strings.Join(orphanNames(r.Ambiguous), ", "))
		}
		if len(r.Orphans) > 0 {
			fmt.Fprintf(stdoutW, "%s: no source row for %d row(s): %s (a rename or a retired row; kept, never deleted)\n",
				where, len(r.Orphans), strings.Join(orphanNames(r.Orphans), ", "))
		}
	}
	head := fmt.Sprintf("%d markers, %d rows %s, %d rows appended, %d files changed",
		len(reports), recopied, verb, appended, len(changed))
	if dryRun {
		head = fmt.Sprintf("%d markers, %d rows would be re-copied, %d rows would be appended, %d files would change",
			len(reports), recopied, appended, len(changed))
	}
	fmt.Fprintln(stdoutW, head)
	if problems > 0 {
		fmt.Fprintf(stderrW, "embed refresh: %d marker(s) skipped; run `machinery check %s --gate ge` for the reasons\n", problems, design)
		return commandExit(1)
	}
	return nil
}

// orphanNames renders row keys for a report line: the key's first field is
// the row's leading token, which is what an author recognizes.
func orphanNames(keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		parts := strings.Split(k, "\x00")
		if len(parts) > 2 {
			out = append(out, parts[0]+" ("+parts[1]+" -> "+parts[2]+")")
			continue
		}
		out = append(out, parts[0])
	}
	return out
}
