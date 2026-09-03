package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

const snapshotDiagnosticHelperEnv = "MACHINERY_SNAPSHOT_DIAGNOSTIC_HELPER"

type snapshotDiagnosticInvocation struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// TestSnapshotDiagnosticCommandHelper is a subprocess boundary for commands
// that correctly write to the real process streams (not only the CLI's test
// writers). It lets the contract compare the complete user-visible diagnostic
// without redirecting process-global file descriptors inside the test runner.
func TestSnapshotDiagnosticCommandHelper(t *testing.T) {
	payload := os.Getenv(snapshotDiagnosticHelperEnv)
	if payload == "" {
		return
	}
	var invocation snapshotDiagnosticInvocation
	if err := json.Unmarshal([]byte(payload), &invocation); err != nil {
		fmt.Fprintln(os.Stderr, "snapshot diagnostic helper:", err)
		return
	}
	constructors := map[string]func() *cobra.Command{
		"alloy":           newAlloyCmd,
		"attest":          newAttestCmd,
		"baseline":        newBaselineCmd,
		"check":           newCheckCmd,
		"compose":         newComposeCmd,
		"embed":           newEmbedCmd,
		"formal":          newVerifyFormalCmd,
		"ir-dump":         newIRDumpCmd,
		"lint":            newLintCmd,
		"oracle":          newOracleCmd,
		"pack":            newPackCmd,
		"refine":          newRefineCmd,
		"scale":           newScaleCmd,
		"sweep":           newSweepCmd,
		"tla":             newTLACmd,
		"tokens-equal":    newTokensEqualCmd,
		"verify-c4":       newVerifyC4Cmd,
		"verify-checkers": newVerifyCheckersCmd,
	}
	constructor := constructors[invocation.Command]
	if constructor == nil {
		fmt.Fprintln(os.Stderr, "snapshot diagnostic helper: unknown command", invocation.Command)
		return
	}
	cmd := constructor()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.SetArgs(invocation.Args)
	if err := executeCapturedCommand(cmd); err != nil {
		fmt.Fprintln(os.Stderr, "command error:", err)
	}
}

func runSnapshotDiagnosticCommand(t *testing.T, invocation snapshotDiagnosticInvocation) string {
	t.Helper()
	payload, err := json.Marshal(invocation)
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestSnapshotDiagnosticCommandHelper$", "-test.count=1")
	cmd.Env = append(os.Environ(), snapshotDiagnosticHelperEnv+"="+string(payload))
	body, runErr := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("%s diagnostic command timed out: %v", invocation.Command, ctx.Err())
	}
	var exitErr *exec.ExitError
	status := 0
	if errors.As(runErr, &exitErr) {
		status = exitErr.ExitCode()
	} else if runErr != nil {
		t.Fatalf("%s diagnostic helper failed to start: %v", invocation.Command, runErr)
	}
	return fmt.Sprintf("exit=%d\n%s", status, body)
}

func seedSnapshotDiagnosticDesign(t *testing.T) (design, impl, registry string) {
	t.Helper()
	design = seedDesignAccessContractDesign(t)
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(design, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("formal/Contract.semantics.yaml", "machine: Contract\n")
	write("formal/system.composition.yaml", "composition: [\n")
	write("decomposition.yaml", "revision: 1\nsubsystems: []\n")
	write("packmap.yaml", "revision: 1\nmapping: {}\n")
	outside := filepath.Join(t.TempDir(), "outside-sentinel")
	if err := os.WriteFile(outside, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(design, "invalid-input-link")); err != nil {
		t.Skipf("platform cannot create the symlink needed for the strict inventory contract: %v", err)
	}
	impl = t.TempDir()
	if err := os.WriteFile(filepath.Join(impl, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	registry = filepath.Join(t.TempDir(), "checkers.local.yaml")
	registryBody := "checkers:\n  test:\n    runtime:\n      kind: oci\n      engine: [machinery-contract-engine]\n      image: example.invalid/test@sha256:1111111111111111111111111111111111111111111111111111111111111111\n      platform: linux/amd64\n    run: [checker, '{out}']\n"
	if err := os.WriteFile(registry, []byte(registryBody), 0o600); err != nil {
		t.Fatal(err)
	}
	return design, impl, registry
}

func snapshotDirectInputDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(cmdTestNeutralTempRoot, "machinery-diagnostic-user-input-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("remove neutral diagnostic input root: %v", err)
		}
	})
	if pathWithin(dir, cmdTestControlRoot) {
		t.Fatalf("direct-input fixture %s remained inside private control root %s", dir, cmdTestControlRoot)
	}
	return dir
}

// Every migrated entry point must render an invalid-input failure from its
// logical source path. Repeating each whole CLI invocation catches random
// immutable-snapshot and engine-scratch names that can otherwise leak from a
// private implementation detail into a supposedly deterministic diagnostic.
func TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree(t *testing.T) {
	directInputRoot := snapshotDirectInputDir(t)
	// Every fixture below is an explicit user input to at least one command.
	// Keep it separate from TestMain's private control state, then enforce that
	// its nonce appears only for a command that was actually given a descendant.
	for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(key, directInputRoot)
	}
	design, impl, registry := seedSnapshotDiagnosticDesign(t)
	machine := filepath.Join(design, "machines", "Contract.machine.json")
	semantics := filepath.Join(design, "formal", "Contract.semantics.yaml")
	composition := filepath.Join(design, "formal", "system.composition.yaml")
	missingA := filepath.Join(directInputRoot, "missing-a.txt")
	missingB := filepath.Join(directInputRoot, "missing-b.txt")
	cases := []snapshotDiagnosticInvocation{
		{"tla", []string{machine}},
		{"alloy", []string{design}},
		{"oracle", []string{filepath.Join(design, "machines")}},
		{"oracle", []string{"--diff", filepath.Join(design, "machines")}},
		{"embed", []string{"refresh", design}},
		{"embed", []string{"refresh", "--dry-run", design}},
		{"compose", []string{composition, machine}},
		{"refine", []string{machine, semantics}},
		{"formal", []string{"--gen-only", design}},
		{"pack", []string{"generate", design}},
		{"pack", []string{"refine", design}},
		{"baseline", []string{"--impl", impl, "--date", "2026-01-01", design}},
		{"lint", []string{filepath.Join(design, "machines")}},
		{"check", []string{design}},
		{"verify-c4", []string{design}},
		{"verify-checkers", []string{"--registry", registry, design}},
		{"scale", []string{design}},
		{"sweep", []string{designAccessMarker, design}},
		{"ir-dump", []string{machine}},
		{"attest", []string{missingA}},
		{"tokens-equal", []string{missingA, missingB}},
	}
	privatePrefixes := []string{
		"machinery-cmd-test-control-",
		"machinery-design-source-",
		"machinery-design-workspace-",
		"machinery-external-tree-",
		"machinery-external-file-",
		"machinery-formal-",
		"machinery-pack-machine-",
		"machinery-tlc-",
		"machinery-alloy-",
		"machinery-c4-runtime-",
		"machinery-verify-c4-",
		"machinery-structurizr-tool-",
	}
	for _, tc := range cases {
		name := tc.Command + "/" + strings.Join(tc.Args, "_")
		t.Run(name, func(t *testing.T) {
			first := runSnapshotDiagnosticCommand(t, tc)
			second := runSnapshotDiagnosticCommand(t, tc)
			if first != second {
				t.Fatalf("diagnostic changed across identical invocations\nfirst:\n%s\nsecond:\n%s", first, second)
			}
			for _, prefix := range privatePrefixes {
				if strings.Contains(first, prefix) {
					t.Fatalf("diagnostic exposed private path prefix %q:\n%s", prefix, first)
				}
			}
			if strings.Contains(first, filepath.Base(directInputRoot)) {
				allowed := false
				for _, argument := range tc.Args {
					if pathWithin(argument, directInputRoot) {
						allowed = true
						break
					}
				}
				if !allowed {
					t.Fatalf("diagnostic exposed neutral fixture nonce without a matching explicit user argument:\n%s", first)
				}
			}
		})
	}
}
