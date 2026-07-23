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
	c.RunE = func(cmd *cobra.Command, args []string) error {
		design := args[0]
		if !checker.HasCheckers(design) {
			fmt.Fprintf(stderrW, "machinery_project: no checkers/*.checker.yaml in %s\n", design)
			exitFunc(1)
			return nil
		}
		results, err := checker.ProjectAll(design, machversion.Version)
		if err != nil {
			fmt.Fprintf(stderrW, "machinery_project: %s\n", err)
			exitFunc(1)
			return nil
		}
		for _, r := range results {
			fmt.Fprintf(stdoutW, "wrote %s (checker %s)\n", r.Path, r.CheckerID)
		}
		return nil
	}
	return c
}
