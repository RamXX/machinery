// tokens-equal: the token-identity proof as an artifact. The hard-TDD
// protocol allows an owner-sanctioned formatting-only amendment to a locked
// file only when it carries a token-identity proof; until now no tool could
// produce or verify one, so the proof was a claim. Two files are
// token-identical exactly when their whitespace-delimited token streams are
// equal: reflow, indentation, and blank lines are formatting; any other
// change is content and refuses the proof.
package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

func newTokensEqualCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "tokens-equal <old-file> <new-file>",
		Short: "Prove two files are formatting-only variants (equal whitespace-delimited token streams)",
		Args:  cobra.ExactArgs(2),
	}
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		return tokensEqualRunTo(args[0], args[1], output.stdout, output.stderr)
	}
	return c
}

func tokensEqualRun(oldPath, newPath string) error {
	return tokensEqualRunTo(oldPath, newPath, stdoutW, stderrW)
}

func tokensEqualRunTo(oldPath, newPath string, stdoutW, stderrW io.Writer) error {
	oldFile, err := openStableRegular(oldPath)
	if err != nil {
		fmt.Fprintln(stderrW, "tokens-equal:", err)
		return commandExitBecause(1, err)
	}
	newFile, err := openStableRegular(newPath)
	if err != nil {
		err = errors.Join(err, oldFile.close())
		fmt.Fprintln(stderrW, "tokens-equal:", err)
		return commandExitBecause(1, err)
	}
	oldBody, oldReadErr := oldFile.read()
	newBody, newReadErr := newFile.read()
	stableRegularAfterInitialRead(oldPath)
	stableRegularAfterInitialRead(newPath)
	oldValidateErr := oldFile.revalidate(oldBody)
	newValidateErr := newFile.revalidate(newBody)
	closeErr := errors.Join(oldFile.close(), newFile.close())
	if err := errors.Join(oldReadErr, newReadErr, oldValidateErr, newValidateErr, closeErr); err != nil {
		fmt.Fprintln(stderrW, "tokens-equal:", err)
		return commandExitBecause(1, err)
	}
	oldToks := strings.Fields(string(oldBody))
	newToks := strings.Fields(string(newBody))
	limit := min(len(oldToks), len(newToks))
	for i := range limit {
		if oldToks[i] != newToks[i] {
			fmt.Fprintf(stdoutW, "NOT token-identical: token %d differs: %q vs %q (context: %s | %s)\n",
				i+1, oldToks[i], newToks[i], tokenContext(oldToks, i), tokenContext(newToks, i))
			return commandExitBecause(1, fmt.Errorf("token %d differs", i+1))
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
		return commandExitBecause(1, fmt.Errorf("token counts differ: %d vs %d", len(oldToks), len(newToks)))
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
