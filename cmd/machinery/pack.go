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
	gen.RunE = func(cmd *cobra.Command, args []string) error {
		ids, err := pack.WritePacks(args[0])
		if err != nil {
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
			return nil
		}
		packs, err := pack.GeneratePacks(args[0])
		if err != nil {
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
			return nil
		}
		for _, id := range ids {
			fmt.Fprintf(stdoutW, "generated packs/%s.pack (%d files, hash %.12s)\n",
				id, len(packs[id]), pack.ContentHash(packs[id]))
		}
		return nil
	}

	ref := &cobra.Command{
		Use:   "refine <child-design>",
		Short: "Generate the contract-refinement proof artifacts from packmap.yaml (reconciled)",
		Args:  cobra.ExactArgs(1),
	}
	ref.RunE = func(cmd *cobra.Command, args []string) error {
		names, err := pack.WriteRefinement(args[0])
		if err != nil {
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
			return nil
		}
		for _, n := range names {
			fmt.Fprintf(stdoutW, "generated formal/%s\n", n)
		}
		return nil
	}

	c.AddCommand(gen, ref)
	return c
}
