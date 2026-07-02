package main

import (
	"io"
	"os"
)

// indirections so tests/the real main can observe exit + stderr.
var (
	stdoutW  io.Writer = os.Stdout
	stderrW  io.Writer = os.Stderr
	exitFunc           = os.Exit
)
