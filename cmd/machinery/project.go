package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/checker"
	"github.com/RamXX/machinery/internal/gates"
	machversion "github.com/RamXX/machinery/internal/version"
)

func newProjectCmd() *cobra.Command {
	var factsDir string
	c := &cobra.Command{
		Use:   "project <design-dir> [--facts <dir>]",
		Short: "Generate the committed projection for every external-checker manifest",
		Long: "Without flags, writes the committed projection.json of every checker manifest under\n" +
			"design/checkers/. A manifest that names only model, invariants and relationships gets the\n" +
			"1.0 projection; one that names any other layer gets the 2.0 projection.\n\n" +
			"With --facts <dir>, writes the design's facts instead: one tab-separated <relation>.facts\n" +
			"file per relation of every layer the design has, plus relations.txt (name, arity, layer,\n" +
			"columns), in the input format Souffle and internal/datalog read. The directory is staged\n" +
			"and renamed into place; no checker projection is touched and no manifest is needed.",
		Args: cobra.ExactArgs(1),
	}
	c.Flags().StringVar(&factsDir, "facts", "", "write the design's fact relations to this directory instead of the checker projections")
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		design := args[0]
		if cmd.Flags().Changed("facts") {
			return writeDesignFacts(output, design, factsDir)
		}
		has, err := checker.HasCheckers(design)
		if err != nil {
			fmt.Fprintf(output.stderr, "machinery_project: %s\n", err)
			return commandExitBecause(1, err)
		}
		if !has {
			fmt.Fprintf(output.stderr, "machinery_project: no checkers/*.checker.yaml in %s\n", design)
			return commandExit(1)
		}
		results, err := checker.ProjectAllWithFacts(design, machversion.Version, gates.LoadDesignFacts)
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

// writeDesignFacts is `machinery project --facts`: read every fact layer
// inside a reader snapshot, release it, then publish the directory.
func writeDesignFacts(output *commandOutput, design, dir string) error {
	if dir == "" {
		err := errors.New("--facts needs a directory")
		fmt.Fprintf(output.stderr, "machinery_project: %s\n", err)
		return commandExitBecause(1, err)
	}
	snapshot, err := gates.AcquireSnapshot(design)
	if err != nil {
		fmt.Fprintf(output.stderr, "machinery_project: %s\n", err)
		return commandExitBecause(1, err)
	}
	facts, err := snapshot.LoadDesignFacts()
	if releaseErr := snapshot.Release(); releaseErr != nil {
		err = errors.Join(err, releaseErr)
	}
	if err != nil {
		fmt.Fprintf(output.stderr, "machinery_project: %s\n", err)
		return commandExitBecause(1, err)
	}
	if err := checker.WriteFactsDir(dir, facts); err != nil {
		fmt.Fprintf(output.stderr, "machinery_project: %s\n", err)
		return commandExitBecause(1, err)
	}
	relations := 0
	for _, layer := range facts.Layers() {
		relations += len(checker.LayerRelations(layer))
	}
	fmt.Fprintf(output.stdout, "wrote %s (%d relations across %d layers)\n", dir, relations, len(facts.Layers()))
	return nil
}
