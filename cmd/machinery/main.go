// Command machinery is the machinery v2 command-line tool.
package main

import (
	"os"

	"github.com/RamXX/machinery/v2/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
