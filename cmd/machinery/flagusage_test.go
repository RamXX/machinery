package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Cobra reads BACKQUOTES in a flag's usage string as the name of the flag's
// argument placeholder: "read the baseline at this ref (`git show <ref>`)"
// renders as "--against git show <ref>" in the help, with the type gone and
// the sentence mangled. It is silent, so it survives review; this catches it.
func TestFlagUsageStringsCarryNoBackquotes(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		check := func(f *pflag.Flag) {
			if strings.Contains(f.Usage, "`") {
				t.Errorf("%s --%s: usage string contains a backquote, which cobra reads as the argument placeholder name; write the term without backquotes", c.CommandPath(), f.Name)
			}
		}
		c.Flags().VisitAll(check)
		c.PersistentFlags().VisitAll(check)
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	for _, c := range allCommands() {
		walk(c)
	}
}

// allCommands builds every subcommand the binary registers, so the guard
// covers the whole surface rather than the one command a test remembered.
func allCommands() []*cobra.Command {
	return []*cobra.Command{
		newLintCmd(), newOracleCmd(), newTokensEqualCmd(), newTLACmd(), newAlloyCmd(),
		newRefineCmd(), newComposeCmd(), newCheckCmd(), newAttestCmd(), newProjectCmd(),
		newVerifyCheckersCmd(), newBaselineCmd(), newVerifyFormalCmd(), newVerifyC4Cmd(),
		newPackCmd(), newEmbedCmd(), newScaleCmd(), newSweepCmd(), newDoctorCmd(),
		newPreflightCmd(), newInstallCmd(), newUpdateCmd(), newUninstallCmd(),
		newIRDumpCmd(), newHookCmd(),
	}
}
