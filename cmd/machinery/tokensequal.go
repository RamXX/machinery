// tokens-equal: the token-identity proof as an artifact. The hard-TDD
// protocol allows an owner-sanctioned formatting-only amendment to a locked
// file only when it carries a token-identity proof; until now no tool could
// produce or verify one, so the proof was a claim. Two files are
// token-identical exactly when their whitespace-delimited token streams are
// equal: reflow, indentation, and blank lines are formatting; any other
// change is content and refuses the proof.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newTokensEqualCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "tokens-equal <old-file> <new-file>",
		Short: "Prove two files are formatting-only variants (equal whitespace-delimited token streams)",
		Args:  cobra.ExactArgs(2),
	}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		return tokensEqualRun(args[0], args[1])
	}
	return c
}

func tokensEqualRun(oldPath, newPath string) error {
	oldBody, err := os.ReadFile(oldPath)
	if err != nil {
		fmt.Fprintln(stderrW, "tokens-equal:", err)
		exitFunc(1)
		return err
	}
	newBody, err := os.ReadFile(newPath)
	if err != nil {
		fmt.Fprintln(stderrW, "tokens-equal:", err)
		exitFunc(1)
		return err
	}
	oldToks := strings.Fields(string(oldBody))
	newToks := strings.Fields(string(newBody))
	limit := min(len(oldToks), len(newToks))
	for i := range limit {
		if oldToks[i] != newToks[i] {
			fmt.Fprintf(stdoutW, "NOT token-identical: token %d differs: %q vs %q (context: %s | %s)\n",
				i+1, oldToks[i], newToks[i], tokenContext(oldToks, i), tokenContext(newToks, i))
			exitFunc(1)
			return fmt.Errorf("token %d differs", i+1)
		}
	}
	if len(oldToks) != len(newToks) {
		longer, at := newToks, len(oldToks)
		which := newPath
		if len(oldToks) > len(newToks) {
			longer, which = oldToks, oldPath
		}
		fmt.Fprintf(stdoutW, "NOT token-identical: %s carries %d extra token(s) from token %d (%s)\n",
			which, len(longer)-limit, at+1, tokenContext(longer, at))
		exitFunc(1)
		return fmt.Errorf("token counts differ: %d vs %d", len(oldToks), len(newToks))
	}
	fmt.Fprintf(stdoutW, "token-identical: %d tokens; the change is formatting-only\n", len(oldToks))
	return nil
}

// tokenContext renders a few tokens around index i for the divergence report.
func tokenContext(toks []string, i int) string {
	lo := max(i-2, 0)
	hi := min(i+3, len(toks))
	return strings.Join(toks[lo:hi], " ")
}
