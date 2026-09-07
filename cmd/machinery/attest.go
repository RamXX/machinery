package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/gates"
)

func stableAttestationHashes(paths []string) ([]string, error) {
	files := make([]*stableRegularFile, 0, len(paths))
	closeFiles := func() error {
		var errs []error
		for _, file := range files {
			errs = append(errs, file.close())
		}
		return errors.Join(errs...)
	}
	for i, path := range paths {
		file, err := openStableRegular(path)
		if err != nil {
			return nil, errors.Join(err, closeFiles())
		}
		for prior := range files {
			if os.SameFile(files[prior].info, file.info) || strings.EqualFold(files[prior].path, file.path) {
				return nil, errors.Join(
					fmt.Errorf("attestation paths %s and %s alias the same file identity", paths[prior], paths[i]),
					file.close(),
					closeFiles(),
				)
			}
		}
		files = append(files, file)
	}
	bodies := make([][]byte, len(files))
	var readErrs []error
	for i, file := range files {
		body, err := file.read()
		bodies[i] = body
		if err != nil {
			readErrs = append(readErrs, fmt.Errorf("read %s: %w", paths[i], err))
		}
		stableRegularAfterInitialRead(paths[i])
	}
	for i, file := range files {
		if err := file.revalidate(bodies[i]); err != nil {
			readErrs = append(readErrs, err)
		}
	}
	if err := errors.Join(errors.Join(readErrs...), closeFiles()); err != nil {
		return nil, err
	}
	hashes := make([]string, len(bodies))
	for i, body := range bodies {
		sum := sha256.Sum256(body)
		hashes[i] = "sha256:" + hex.EncodeToString(sum[:])
	}
	return hashes, nil
}

// newAttestCmd is the attestor's side of Gv-attest: it prints the content
// hash the gate will demand, so nobody hand-rolls a digest the gate then
// rejects as malformed or, worse, gets right by accident with a different
// algorithm. `--claims` prints the closed claim vocabulary from the gate
// itself, which is the other value that must never be copied by hand.
func newAttestCmd() *cobra.Command {
	var listClaims bool
	var design, impl string
	var review gates.AttestationReview
	const limits = "Hashes bind the observed files and scope; they do not prove tests ran, reviewer identity, or judgment correctness. Top-level Git administration and the exact Machinery attestation record are excluded; applications that use them as runtime inputs are outside this review boundary."
	c := &cobra.Command{
		Use:   "attest [<path> ...]",
		Short: "Print attestation content hashes (and the claim vocabulary) for " + gates.AttestationsFileName,
		Long: "Print the sha256 content hash Gv-attest records for each path, in the schema's own\n" +
			"spelling, one 'sha256:<hex>  <path>' line per file. Paste the hash into the covered\n" +
			"artifact's row in design/" + gates.AttestationsFileName + ". The shell equivalent is\n" +
			"'shasum -a 256 <path>' (or 'sha256sum <path>') with the 'sha256:' prefix added.\n\n" +
			"With --claims, print the closed attested-claim vocabulary instead.\n\n" +
			"Generate one complete v2 row with --design, --claim, --kind plan|current|historical,\n" +
			"--attestor and --date YYYY-MM-DD. Current requires --impl; plan/historical forbid it.\n" +
			"Review subjects, then merge the row into attestations.yaml. Generation never writes that file.\n" +
			"Full-root-v1 has fixed bounds: 100000 entries, depth 64, 1 GiB/file, 8 GiB aggregate;\n" +
			"the generated and merged evidence document must fit 16 MiB.\n" + limits,
		Args: cobra.ArbitraryArgs,
	}
	c.Flags().BoolVar(&listClaims, "claims", false, "print the closed attested-claim vocabulary and exit")
	c.Flags().StringVar(&design, "design", "", "logical design root for v2 generation")
	c.Flags().StringVar(&impl, "impl", "", "complete implementation/test root for kind=current")
	c.Flags().StringVar(&review.Claim, "claim", "", "one closed-vocabulary claim")
	c.Flags().StringVar(&review.Kind, "kind", "", "plan, current, or historical")
	c.Flags().StringVar(&review.Attestor, "attestor", "", "reviewer attribution (not authentication)")
	c.Flags().StringVar(&review.Date, "date", "", "explicit review date YYYY-MM-DD")
	c.Flags().StringVar(&review.Note, "note", "", "optional review note")
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		stdout, stderr := output.stdout, output.stderr
		generation := false
		for _, flag := range []string{"design", "impl", "claim", "kind", "attestor", "date", "note"} {
			generation = generation || cmd.Flags().Changed(flag)
		}
		if generation {
			var err error
			if listClaims || len(args) > 0 {
				err = fmt.Errorf("GV_SCHEMA: generation cannot combine --claims or positional files")
			}
			for _, flag := range []string{"design", "claim", "kind", "attestor", "date"} {
				if !cmd.Flags().Changed(flag) {
					err = errors.Join(err, fmt.Errorf("GV_SCHEMA: generation requires --%s", flag))
				}
			}
			var body []byte
			if err == nil {
				body, err = gates.RenderAttestation(design, impl, review)
			}
			if err != nil {
				fmt.Fprintf(stderr, "machinery attest: %s\n", err)
				return commandExitBecause(1, err)
			}
			fmt.Fprintln(stderr, limits)
			fmt.Fprintln(stderr, "Generated and merged evidence documents must fit the fixed 16 MiB document limit.")
			fmt.Fprint(stdout, string(body))
			return nil
		}
		if listClaims {
			for _, id := range gates.AttestationClaimIDs() {
				fmt.Fprintln(stdout, id)
			}
			return nil
		}
		if len(args) == 0 {
			fmt.Fprintln(stderr, "machinery attest: name at least one path to hash, or pass --claims")
			return commandExit(1)
		}
		hashes, err := stableAttestationHashes(args)
		if err != nil {
			fmt.Fprintf(stderr, "machinery attest: %s\n", err)
			return commandExitBecause(1, err)
		}
		var rendered strings.Builder
		for i, path := range args {
			fmt.Fprintf(&rendered, "%s  %s\n", hashes[i], path)
		}
		fmt.Fprint(stdout, rendered.String())
		return nil
	}
	return c
}
