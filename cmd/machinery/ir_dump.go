package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ramirosalas/machinery/internal/ir"
)

// irDumpRun implements `machinery ir-dump <machine.json>`: canonical IR
// serialization for the Phase-2 differential parity probe.
func irDumpRun(path string) error {
	root, err := ir.LoadMachineJSON(path)
	if err != nil {
		fmt.Fprintf(stderrW, "ir_dump: %s\n", err)
		exitFunc(1)
		return err
	}
	out, err := ir.DumpJSON(root)
	if err != nil {
		fmt.Fprintf(stderrW, "ir_dump: %s\n", err)
		exitFunc(1)
		return err
	}
	fmt.Fprint(stdoutW, out)
	_ = filepath.Base(path) // path normalization happens in diff-tool.sh
	_ = os.Stdout
	return nil
}
