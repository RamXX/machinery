package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/refine"
)

func newRefineCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "refine <machine.json> <semantics.yaml> [out-dir]",
		Short: "Generate the data-refined model + refinement from a semantics annotation",
		Args:  cobra.RangeArgs(2, 3),
	}
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		outdir := ""
		if len(args) > 2 {
			outdir = args[2]
		}
		if err := refine.RunTo(args[0], args[1], outdir, output.stdout); err != nil {
			fmt.Fprintln(output.stderr, err)
			return commandExitBecause(1, err)
		}
		return nil
	}
	return c
}
