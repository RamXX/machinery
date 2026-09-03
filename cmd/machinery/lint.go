package main

import (
	"bytes"
	"errors"
	"io"
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
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		mdir := "."
		if len(args) > 0 {
			mdir = args[0]
		}
		abs, err := filepath.Abs(mdir)
		if err != nil {
			return err
		}
		design, rel := abs, "."
		if filepath.Base(abs) == "machines" {
			design, rel = filepath.Dir(abs), "machines"
		}
		return withDesignSnapshot(design, func(snapshot string) error {
			var out, errOut bytes.Buffer
			rc := lint.Run(filepath.Join(snapshot, rel), &out, &errOut)
			_, outErr := io.WriteString(output.stdout, remapSnapshotText(out.String(), snapshot, design))
			_, stderrErr := io.WriteString(output.stderr, remapSnapshotText(errOut.String(), snapshot, design))
			if rc != 0 {
				return errors.Join(commandExit(rc), outErr, stderrErr)
			}
			return errors.Join(outErr, stderrErr)
		})
	}
	return c
}
