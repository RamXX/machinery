// Command integration-lane is the compile-safe RED contract bootstrap.
// The RED-phase review explicitly permits this fail-closed entry point only.
// It performs no validation, provisioning, execution or cleanup; the separate
// GREEN phase must supply those behaviors without changing the frozen tests.
package main

import (
	"fmt"
	"io"
	"os"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(_ []string, _ io.Writer, stderr io.Writer) int {
	fmt.Fprintln(stderr, "required integration lane is not implemented")
	return 1
}
