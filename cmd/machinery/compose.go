package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramirosalas/machinery/internal/compose"
)

func newComposeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "compose <composition.yaml> <coordinator.machine.json> [out-dir]",
		Short: "Generate the cross-aggregate composition spec validated against the coordinator",
		Args:  cobra.RangeArgs(2, 3),
	}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		outdir := ""
		if len(args) > 2 {
			outdir = args[2]
		}
		if err := compose.Run(args[0], args[1], outdir); err != nil {
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
		}
		return nil
	}
	return c
}
