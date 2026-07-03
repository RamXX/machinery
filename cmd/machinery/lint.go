package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/lint"
)

func newLintCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "lint <machines-dir>",
		Short: "Structural lint + matrix reconciliation for machinery machines",
		Args:  cobra.MaximumNArgs(1),
	}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		mdir := "."
		if len(args) > 0 {
			mdir = args[0]
		}
		rc := lint.Run(mdir, os.Stdout, os.Stderr)
		exitFunc(rc)
		return nil
	}
	_ = fmt.Sprintf
	_ = filepath.Base
	return c
}
