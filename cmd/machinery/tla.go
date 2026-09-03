package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/tla"
)

func newTLACmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "tla <machine.json> [out-dir]",
		Short: "Generate the TLA+ control-flow model from a machine",
		Args:  cobra.RangeArgs(1, 2),
	}
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		outdir := ""
		if len(args) > 1 {
			outdir = args[1]
		}
		if err := tla.RunTo(args[0], outdir, output.stdout); err != nil {
			fmt.Fprintln(output.stderr, err)
			return commandExitBecause(1, err)
		}
		return nil
	}
	return c
}
