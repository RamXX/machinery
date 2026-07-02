package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ramirosalas/machinery/internal/tla"
)

func newTLACmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "tla <machine.json> [out-dir]",
		Short: "Generate the TLA+ control-flow model from a machine",
		Args:  cobra.RangeArgs(1, 2),
	}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		outdir := ""
		if len(args) > 1 {
			outdir = args[1]
		}
		if err := tla.Run(args[0], outdir); err != nil {
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
		}
		return nil
	}
	_ = os.Stdout
	return c
}
