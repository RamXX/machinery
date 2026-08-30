package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/gates"
)

// newAttestCmd is the attestor's side of Gv-attest: it prints the content
// hash the gate will demand, so nobody hand-rolls a digest the gate then
// rejects as malformed or, worse, gets right by accident with a different
// algorithm. `--claims` prints the closed claim vocabulary from the gate
// itself, which is the other value that must never be copied by hand.
func newAttestCmd() *cobra.Command {
	var listClaims bool
	c := &cobra.Command{
		Use:   "attest [<path> ...]",
		Short: "Print attestation content hashes (and the claim vocabulary) for " + gates.AttestationsFileName,
		Long: "Print the sha256 content hash Gv-attest records for each path, in the schema's own\n" +
			"spelling, one 'sha256:<hex>  <path>' line per file. Paste the hash into the covered\n" +
			"artifact's row in design/" + gates.AttestationsFileName + ". The shell equivalent is\n" +
			"'shasum -a 256 <path>' (or 'sha256sum <path>') with the 'sha256:' prefix added.\n\n" +
			"With --claims, print the closed attested-claim vocabulary instead.",
		Args: cobra.ArbitraryArgs,
	}
	c.Flags().BoolVar(&listClaims, "claims", false, "print the closed attested-claim vocabulary and exit")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		if listClaims {
			for _, id := range gates.AttestationClaimIDs() {
				fmt.Fprintln(stdoutW, id)
			}
			return nil
		}
		if len(args) == 0 {
			fmt.Fprintln(stderrW, "machinery attest: name at least one path to hash, or pass --claims")
			exitFunc(1)
			return nil
		}
		failed := false
		for _, path := range args {
			hash, err := gates.ContentHash(path)
			if err != nil {
				fmt.Fprintf(stderrW, "machinery attest: %s\n", err)
				failed = true
				continue
			}
			fmt.Fprintf(stdoutW, "%s  %s\n", hash, path)
		}
		if failed {
			exitFunc(1)
		}
		return nil
	}
	return c
}
