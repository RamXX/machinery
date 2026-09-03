package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/pack"
)

func newPackCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "pack",
		Short: "Recursive decomposition: generate contract packs (parent) and refinement proofs (child)",
	}

	gen := &cobra.Command{
		Use:   "generate <parent-design>",
		Short: "Generate the frozen per-subsystem contract packs from decomposition.yaml",
		Args:  cobra.ExactArgs(1),
	}
	gen.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		written, err := pack.WritePacksWithMetadata(args[0])
		if err != nil {
			fmt.Fprintln(output.stderr, err)
			return commandExitBecause(1, err)
		}
		for _, result := range written {
			fmt.Fprintf(output.stdout, "generated packs/%s.pack (%d files, hash %.12s)\n",
				result.ID, result.FileCount, result.Hash)
		}
		return nil
	}

	ref := &cobra.Command{
		Use:   "refine <child-design>",
		Short: "Generate the contract-refinement proof artifacts from packmap.yaml (reconciled)",
		Args:  cobra.ExactArgs(1),
	}
	ref.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		names, err := pack.WriteRefinement(args[0])
		if err != nil {
			fmt.Fprintln(output.stderr, err)
			return commandExitBecause(1, err)
		}
		for _, n := range names {
			fmt.Fprintf(output.stdout, "generated formal/%s\n", n)
		}
		return nil
	}

	c.AddCommand(gen, ref)
	return c
}
