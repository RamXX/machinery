package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/checker"
)

// newVerifyCheckersCmd wires `machinery verify-checkers`, the external half of
// the checker layer. It is to Gk what verify-formal is to the relational gates:
// the pure phase (Gk) trusts the committed evidence because its input_hash binds
// it to the design; this phase re-runs the engine and confirms the committed
// verdict is actually reproducible over the current design.
func newVerifyCheckersCmd() *cobra.Command {
	var registryPath string
	var checkerID string
	c := &cobra.Command{
		Use:   "verify-checkers <design-dir> [--registry <path>] [--checker <id>]",
		Short: "Re-run external checkers and confirm the committed evidence is reproducible",
		Args:  cobra.ExactArgs(1),
	}
	c.Flags().StringVar(&registryPath, "registry", checker.DefaultRegistryPath, "path to the local (git-ignored) checker registry")
	c.Flags().StringVar(&checkerID, "checker", "", "verify only the checker with this id")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		if rc := verifyCheckers(args[0], registryPath, checkerID); rc != 0 {
			exitFunc(rc)
		}
		return nil
	}
	return c
}

// verifyCheckers loads the registry, then re-verifies each committed checker in
// id order. It never re-runs the Gk gate; it requires the committed projection
// to be present (Gk's job is to have blessed it) and confirms a fresh run
// reproduces the committed evidence. Returns 1 if any checker failed, else 0.
func verifyCheckers(design, registryPath, only string) int {
	if err := checkIsDir(design); err != nil {
		fmt.Fprintln(stderrW, err)
		return 1
	}

	reg, err := checker.LoadRegistry(registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(stderrW, "verify-checkers: no checker registry at %s; create it (git-ignored) and add each checker's run command\n", registryPath)
		} else {
			fmt.Fprintf(stderrW, "verify-checkers: %s\n", err)
		}
		return 1
	}

	manifestPaths := checker.ManifestPaths(design)
	if len(manifestPaths) == 0 {
		fmt.Fprintf(stderrW, "verify-checkers: no checkers/*.checker.yaml in %s\n", design)
		return 1
	}

	// Load every manifest first, so the run order can be by checker id (stable
	// and independent of the on-disk file names).
	var mans []*checker.Manifest
	failures := 0
	for _, mp := range manifestPaths {
		man, err := checker.LoadManifest(mp)
		if err != nil {
			fmt.Fprintf(stderrW, "verify-checkers %s: ERROR: %s\n", filepath.Base(mp), err)
			failures++
			continue
		}
		mans = append(mans, man)
	}
	sort.Slice(mans, func(i, j int) bool { return mans[i].Checker.ID < mans[j].Checker.ID })

	verified := 0
	matched := 0
	for _, man := range mans {
		if only != "" && man.Checker.ID != only {
			continue
		}
		matched++
		if verifyOneChecker(design, man, reg) {
			verified++
		} else {
			failures++
		}
	}

	if only != "" && matched == 0 {
		fmt.Fprintf(stderrW, "verify-checkers: no checker with id %q in %s\n", only, design)
		return 1
	}

	if failures > 0 {
		fmt.Fprintf(stdoutW, "\n%d checker(s) verified, %d failure(s)\n", verified, failures)
		return 1
	}
	fmt.Fprintf(stdoutW, "\n%d checker(s) verified\n", verified)
	return 0
}

// verifyOneChecker re-runs one checker's adapter and confirms the committed
// evidence is reproducible. It prints one result line on success and one ERROR
// on failure, returning whether the checker verified.
func verifyOneChecker(design string, man *checker.Manifest, reg *checker.Registry) bool {
	id := man.Checker.ID
	fail := func(format string, a ...any) bool {
		fmt.Fprintf(stderrW, "verify-checkers "+id+": ERROR: "+format+"\n", a...)
		return false
	}

	entry, ok := reg.Resolve(id)
	if !ok {
		return fail("no registry entry for checker '%s'; add it to %s", id, reg.Path)
	}

	// The committed projection is the checker's input. Gk requires it fresh;
	// here we only require it present (running Gk is a separate phase).
	projPath := filepath.Join(design, man.Evidence.ProjectionOut)
	if !fileExists(projPath) {
		return fail("projection not committed; run 'machinery project %s' first", design)
	}

	evPath := filepath.Join(design, man.Evidence.EvidenceIn)
	committed, err := checker.LoadEvidence(evPath)
	if err != nil {
		return fail("committed evidence for '%s' is unreadable: %s", id, err)
	}

	work, err := os.MkdirTemp("", "machinery-verify-checkers")
	if err != nil {
		return fail("cannot create work dir: %s", err)
	}
	defer os.RemoveAll(work)

	// {config}: the manifest's opaque config block, written as JSON so an
	// adapter reads it with no YAML parser.
	configPath := filepath.Join(work, "config.json")
	cfg := man.Config
	if cfg == nil {
		cfg = map[string]any{}
	}
	cfgBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fail("cannot serialize manifest config: %s", err)
	}
	if err := os.WriteFile(configPath, append(cfgBytes, '\n'), 0o644); err != nil {
		return fail("cannot write config: %s", err)
	}

	outPath := filepath.Join(work, "evidence.json")
	tokens := checker.Tokens{
		Projection: projPath,
		Config:     configPath,
		Manifest:   man.Path,
		Out:        outPath,
		Design:     design,
	}

	out, runErr := runChecker(tokens.Substitute(entry.Run), entry.Timeout)
	if runErr != nil {
		return fail("checker '%s' run failed: %s\n%s", id, runErr, strings.TrimSpace(out))
	}

	fresh, err := checker.LoadEvidence(outPath)
	if err != nil {
		return fail("checker '%s' produced no readable evidence at {out}: %s", id, err)
	}

	if diff := reproDiff(committed, fresh); diff != "" {
		return fail("committed evidence for '%s' is not reproducible: %s", id, diff)
	}

	// Optional replay/verify against the checker's own trace. {out} here is the
	// committed evidence path, so the checker replays what was committed.
	if len(entry.Verify) > 0 {
		vTokens := tokens
		vTokens.Out = evPath
		vout, verr := runChecker(vTokens.Substitute(entry.Verify), entry.Timeout)
		if verr != nil {
			return fail("replay/verify failed for '%s': %s\n%s", id, verr, strings.TrimSpace(vout))
		}
	}

	fmt.Fprintf(stdoutW, "verify-checkers %s: ok (verdict=%s, reproduced)\n", id, fresh.Verdict)
	return true
}

// reproDiff returns "" when fresh reproduces committed, or a specific
// difference: the verdict must match, the input_hash must match, and the
// multiset of (element, verdict) coverage pairs must match.
func reproDiff(committed, fresh *checker.Evidence) string {
	if committed.Verdict != fresh.Verdict {
		return fmt.Sprintf("committed verdict %q but a fresh run reports %q", committed.Verdict, fresh.Verdict)
	}
	if committed.InputHash != fresh.InputHash {
		return fmt.Sprintf("committed input_hash %s but a fresh run computed %s", committed.InputHash, fresh.InputHash)
	}
	if d := coverageDiff(committed.Coverage, fresh.Coverage); d != "" {
		return "coverage differs: " + d
	}
	return ""
}

// coverageDiff compares two coverage lists as multisets of (element, verdict).
// Order is ignored; a differing count for any pair is reported deterministically.
func coverageDiff(committed, fresh []checker.CoverageRow) string {
	cm := coverageMultiset(committed)
	fm := coverageMultiset(fresh)
	keys := map[string]bool{}
	for k := range cm {
		keys[k] = true
	}
	for k := range fm {
		keys[k] = true
	}
	var sorted []string
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, k := range sorted {
		if cm[k] != fm[k] {
			return fmt.Sprintf("%s appears %d time(s) in committed evidence but %d time(s) in the fresh run", k, cm[k], fm[k])
		}
	}
	return ""
}

func coverageMultiset(rows []checker.CoverageRow) map[string]int {
	m := make(map[string]int, len(rows))
	for _, r := range rows {
		m["("+r.Element+", "+r.Verdict+")"]++
	}
	return m
}

// runChecker runs an external checker command under a timeout, capturing
// combined stdout+stderr. A nonzero exit or a timeout is an error; the captured
// output is returned in both cases for the caller to surface.
func runChecker(args []string, timeout time.Duration) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("empty command")
	}
	if timeout <= 0 {
		timeout = checker.DefaultCheckerTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	var buf strings.Builder
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return buf.String(), fmt.Errorf("timed out after %s", timeout)
	}
	return buf.String(), err
}

// fileExists reports whether path is an existing regular file.
func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}
