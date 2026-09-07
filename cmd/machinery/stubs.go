package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/formal"
	"github.com/RamXX/machinery/internal/processcontrol"
	"github.com/RamXX/machinery/internal/processscope"
)

// Once the Phase-1 stub file; every body below has long been implemented
// (verify-formal, doctor, preflight, and the hidden ir-dump). The file name
// survives for history.

func newVerifyFormalCmd() *cobra.Command {
	var genOnly bool
	c := &cobra.Command{
		Use:   "verify-formal <design-dir>",
		Short: "Regenerate + TLC-check the formal suite for a design",
		Args:  cobra.ExactArgs(1),
	}
	c.Flags().BoolVar(&genOnly, "gen-only", false,
		"regenerate the formal suite from source without running TLC (no Java needed)")
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		rc := verifyFormalScoped(cmd.Context(), args[0], genOnly, output.stdout, output.stderr)
		if rc != 0 {
			return commandExit(rc)
		}
		return nil
	}
	return c
}

// verifyFormalScoped is the CLI custody handoff: a full verification opens the
// candidate binary's own native custody scope and passes it through the
// execution context, so every engine subprocess runs as an owned guarded job
// and the run fails closed when custody is unavailable or cannot verify
// cleanup. Generation-only mode launches no engine subprocess and opens no
// scope.
func verifyFormalScoped(ctx context.Context, design string, genOnly bool, stdoutW, stderrW io.Writer) int {
	if genOnly {
		return formal.VerifyFormalInScope(ctx, design, true, stdoutW, stderrW)
	}
	scope, err := openCandidateProcessScope(ctx)
	if err != nil {
		fmt.Fprintln(stderrW, "verify-formal: native process custody is required for engine execution:", err)
		return 1
	}
	rc := formal.VerifyFormalInScope(processcontrol.WithScope(ctx, scope), design, false, stdoutW, stderrW)
	closeCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	rep, closeErr := scope.Close(closeCtx)
	cancel()
	if closeErr != nil || rep.Status != processscope.StatusCleaned {
		fmt.Fprintf(stderrW, "verify-formal: native custody cleanup did not verify: err=%v report=%+v\n", closeErr, rep)
		if rc == 0 {
			rc = 1
		}
	}
	return rc
}

// openCandidateProcessScope starts the custody broker from the exact running
// candidate executable, with the wall budget capped at the shipped maximum
// and the shared cleanup grace at its maximum.
func openCandidateProcessScope(ctx context.Context) (processscope.Scope, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve candidate executable: %w", err)
	}
	body, err := os.ReadFile(exe)
	if err != nil {
		return nil, fmt.Errorf("read candidate executable: %w", err)
	}
	digest := sha256.Sum256(body)
	openCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	scratch := filepath.Join(os.TempDir(), fmt.Sprintf("machinery-processscope-%d", os.Getuid()))
	return processscope.Open(openCtx, processscope.Options{
		HelperExecutable: exe,
		HelperDigest:     hex.EncodeToString(digest[:]),
		ScratchRoot:      scratch,
		Limits:           processscope.Limits{Jobs: 4, WallMS: 3600000, CleanupMS: 30000},
	})
}

func newDoctorCmd() *cobra.Command {
	var targets []string
	c := &cobra.Command{Use: "doctor", Short: "Check prerequisites and install status", Args: cobra.NoArgs}
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		return doctorRunTo(targets, output.stdout)
	}
	c.Flags().StringArrayVar(&targets, "target", nil, "host adapter to inspect: claude, codex, opencode, or all (repeatable)")
	return c
}

func newPreflightCmd() *cobra.Command {
	c := &cobra.Command{Use: "preflight", Short: "Check runtime prerequisites (installs nothing)", Args: cobra.NoArgs}
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		return preflightRunTo(output.stdout)
	}
	return c
}

func newIRDumpCmd() *cobra.Command {
	c := &cobra.Command{
		Use:    "ir-dump <machine.json>",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
	}
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		return irDumpRunTo(args[0], output.stdout, output.stderr)
	}
	return c
}
