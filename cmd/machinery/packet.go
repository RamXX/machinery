package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/gates"
)

// newPacketCmd projects bounded per-slice executor packets from the authored
// slice map (design/slices.yaml). It reads the design under the same snapshot
// custody every other reader uses, runs Gw-packet over the whole map, and
// writes nothing when the gate fails: an over-budget packet or a projection
// that dropped an obligation is never handed to an executor.
func newPacketCmd() *cobra.Command {
	var milestone, slice, out string
	c := &cobra.Command{
		Use:   "packet <design-dir> --milestone <id> --out <dir> [--slice <id>]",
		Short: "Project bounded per-slice executor packets from the authored slice map",
		Long: "Projects one packet per slice of a milestone from design/slices.yaml: the cited shard\n" +
			"sections, oracle rows, matrices, Architecture Contract rows and invariants, each excerpt\n" +
			"verbatim under its stable id and source path:line. The projection is byte-reproducible\n" +
			"for the same design bytes. Gw-packet runs first over the whole map (citations resolve,\n" +
			"packets fit their byte budgets, every obligation the milestone owes is claimed exactly\n" +
			"once or waived); when it fails nothing is written. One size line per packet is printed,\n" +
			"in bytes and in token-equivalents at the documented divisor.",
		Args: cobra.ExactArgs(1),
	}
	c.Flags().StringVar(&milestone, "milestone", "", "milestone id to project, of the form M<n> (required)")
	c.Flags().StringVar(&slice, "slice", "", "one slice id of the milestone to project (default: every slice of the milestone)")
	c.Flags().StringVar(&out, "out", "", "directory to write <slice-id>.packet.md files into (required)")
	_ = c.MarkFlagRequired("milestone")
	_ = c.MarkFlagRequired("out")
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		design := args[0]
		snapshot, err := gates.AcquireSnapshot(design)
		if err != nil {
			fmt.Fprintf(output.stderr, "machinery_packet: %s\n", err)
			return commandExitBecause(1, err)
		}
		packets, gate := snapshot.ProjectPackets(milestone, slice)
		if releaseErr := snapshot.Release(); releaseErr != nil {
			fmt.Fprintf(output.stderr, "machinery_packet: release design snapshot lock: %s\n", releaseErr)
			return commandExitBecause(1, releaseErr)
		}
		fail := gate.Emit(output.stdout)
		if fail > 0 || len(gate.Errs) > 0 {
			fmt.Fprintln(output.stderr, "machinery_packet: Gw-packet failed; nothing written")
			return commandExit(1)
		}
		if err := os.MkdirAll(out, 0o755); err != nil {
			fmt.Fprintf(output.stderr, "machinery_packet: %s\n", err)
			return commandExitBecause(1, err)
		}
		var written []string
		for _, p := range packets {
			dest := filepath.Join(out, p.Slice+".packet.md")
			if err := os.WriteFile(dest, p.Body, 0o644); err != nil {
				fmt.Fprintf(output.stderr, "machinery_packet: %s\n", err)
				return commandExitBecause(1, errors.Join(err, errors.New("packet output incomplete")))
			}
			written = append(written, p.SizeLine(filepath.ToSlash(dest)))
		}
		for _, line := range written {
			fmt.Fprintln(output.stdout, line)
		}
		return nil
	}
	return c
}
