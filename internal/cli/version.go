package cli

import (
	"fmt"

	"github.com/RamXX/machinery/internal/version"
)

func init() {
	Register(&Command{Name: "version", Summary: "Print the version.", Run: func(e *Env, args []string) int {
		fmt.Fprintf(e.Stdout, "machinery %s (%s, %s)\n", version.Version, version.Commit(), version.BuildDate)
		return 0
	}})
}
