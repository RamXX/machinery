// verify-c4: the C4 engine phase. G2 parses workspace.dsl for identifiers and
// tags; whether the DSL actually COMPILES under the Structurizr grammar was an
// attested checklist line ("run structurizr-cli export; fix syntax errors")
// that is literally a shell command. This subcommand is that command, made a
// first-class engine phase like verify-formal (which needs Java) and
// verify-checkers (which needs the registry): pure gates stay dependency-free,
// engine phases shell out and fail loudly when the engine is absent.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// structurizrEnv overrides the binary lookup (a pinned path, a wrapper
// script); PATH lookup of "structurizr-cli" is the default.
const structurizrEnv = "MACHINERY_STRUCTURIZR_CLI"

func newVerifyC4Cmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "verify-c4 <design-dir>",
		Short: "Compile workspace.dsl under structurizr-cli export (the C4 engine phase)",
		Args:  cobra.ExactArgs(1),
	}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		design := args[0]
		dsl := filepath.Join(design, "workspace.dsl")
		if fi, err := os.Stat(dsl); err != nil || fi.IsDir() {
			fmt.Fprintf(stderrW, "verify-c4: %s does not exist; nothing to compile\n", dsl)
			exitFunc(1)
			return nil
		}
		bin := os.Getenv(structurizrEnv)
		if bin == "" {
			path, err := exec.LookPath("structurizr-cli")
			if err != nil {
				fmt.Fprintf(stderrW, "verify-c4: structurizr-cli not found on PATH (needs Java 17+); install it (brew install structurizr-cli, or https://github.com/structurizr/cli/releases) or set %s to the binary\n", structurizrEnv)
				exitFunc(1)
				return nil
			}
			bin = path
		}
		out, err := os.MkdirTemp("", "machinery-verify-c4")
		if err != nil {
			fmt.Fprintf(stderrW, "verify-c4: %v\n", err)
			exitFunc(1)
			return nil
		}
		defer os.RemoveAll(out)
		run := exec.CommandContext(cmd.Context(), bin, "export", "-workspace", dsl, "-format", "mermaid", "-output", out)
		combined, err := run.CombinedOutput()
		if err != nil {
			// the exporter's own message is the diagnosis; pass it through
			fmt.Fprintf(stdoutW, "%s", combined)
			fmt.Fprintf(stderrW, "verify-c4: FAIL %s does not compile under structurizr-cli export\n", dsl)
			exitFunc(1)
			return nil
		}
		exported, _ := filepath.Glob(filepath.Join(out, "*"))
		fmt.Fprintf(stdoutW, "verify-c4: ok, %s compiles (structurizr-cli export, %d view file(s))\n", dsl, len(exported))
		return nil
	}
	return c
}
