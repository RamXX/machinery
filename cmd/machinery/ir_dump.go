package main

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/RamXX/machinery/internal/ir"
)

// irDumpRun implements `machinery ir-dump <machine.json>`: canonical IR
// serialization for the Phase-2 differential parity probe.
func irDumpRun(path string) error {
	return irDumpRunTo(path, stdoutW, stderrW)
}

func irDumpRunTo(path string, stdoutW, stderrW io.Writer) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	design := filepath.Dir(abs)
	if filepath.Base(design) == "machines" {
		design = filepath.Dir(design)
	}
	rel, err := filepath.Rel(design, abs)
	if err != nil {
		return err
	}
	return withDesignSnapshot(design, func(snapshot string) error {
		root, err := ir.LoadMachineJSON(filepath.Join(snapshot, rel))
		if err != nil {
			fmt.Fprintf(stderrW, "ir_dump: %s\n", remapSnapshotText(err.Error(), snapshot, design))
			return commandExitBecause(1, err)
		}
		out, err := ir.DumpJSON(root)
		if err != nil {
			fmt.Fprintf(stderrW, "ir_dump: %s\n", err)
			return commandExitBecause(1, err)
		}
		fmt.Fprint(stdoutW, out)
		return nil
	})
}
