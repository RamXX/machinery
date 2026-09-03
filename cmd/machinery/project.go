package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/checker"
	machversion "github.com/RamXX/machinery/internal/version"
)

func newProjectCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "project <design-dir>",
		Short: "Generate the committed projection for every external-checker manifest",
		Args:  cobra.ExactArgs(1),
	}
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		design := args[0]
		has, err := checker.HasCheckers(design)
		if err != nil {
			fmt.Fprintf(output.stderr, "machinery_project: %s\n", err)
			return commandExitBecause(1, err)
		}
		if !has {
			fmt.Fprintf(output.stderr, "machinery_project: no checkers/*.checker.yaml in %s\n", design)
			return commandExit(1)
		}
		results, err := checker.ProjectAll(design, machversion.Version)
		if err != nil {
			fmt.Fprintf(output.stderr, "machinery_project: %s\n", err)
			return commandExitBecause(1, err)
		}
		for _, r := range results {
			fmt.Fprintf(output.stdout, "wrote %s (checker %s)\n", r.Path, r.CheckerID)
		}
		return nil
	}
	return c
}
