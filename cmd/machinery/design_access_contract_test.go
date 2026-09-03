package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Every top-level command is deliberately classified. This inventory makes a
// newly registered command choose snapshot semantics in review instead of
// silently reading or writing design state without the canonical lock.
func TestEveryTopLevelCommandClassifiesDesignAccess(t *testing.T) {
	classification := map[string]struct {
		access      string
		readerModes []string
	}{
		"newLintCmd":           {"reader", []string{"machines"}},
		"newOracleCmd":         {"mixed", []string{"diff", "diff-against"}},
		"newTokensEqualCmd":    {access: "stable-file-reader"},
		"newTLACmd":            {access: "writer"},
		"newAlloyCmd":          {access: "writer"},
		"newRefineCmd":         {access: "writer"},
		"newComposeCmd":        {access: "writer"},
		"newCheckCmd":          {"reader", []string{"default", "selected-gate", "complete"}},
		"newAttestCmd":         {access: "stable-file-reader"},
		"newProjectCmd":        {access: "writer"},
		"newVerifyCheckersCmd": {"reader", []string{"all", "selected"}},
		"newBaselineCmd":       {access: "writer"},
		"newVerifyFormalCmd":   {access: "writer"},
		"newVerifyC4Cmd":       {"reader", []string{"default"}},
		"newPackCmd":           {access: "writer"},
		"newEmbedCmd":          {"mixed", []string{"refresh-dry-run"}},
		"newScaleCmd":          {"reader", []string{"default"}},
		"newSweepCmd":          {"reader", []string{"default"}},
		"newDoctorCmd":         {access: "none"},
		"newPreflightCmd":      {access: "none"},
		"newInstallCmd":        {access: "none"},
		"newUpdateCmd":         {access: "none"},
		"newUninstallCmd":      {access: "none"},
		"newVersionCmd":        {access: "none"},
		"newIRDumpCmd":         {"reader", []string{"default"}},
		"newHookCmd":           {access: "event-mixed"},
	}
	body, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`root\.AddCommand\((new[A-Za-z0-9]+Cmd)\(\)\)`)
	matches := re.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		t.Fatal("no top-level command registrations found")
	}
	seen := map[string]bool{}
	for _, match := range matches {
		constructor := match[1]
		seen[constructor] = true
		if classification[constructor].access == "" {
			t.Errorf("%s has no design-access classification", constructor)
		}
	}
	for constructor := range classification {
		if !seen[constructor] {
			t.Errorf("stale design-access classification for unregistered %s", constructor)
		}
	}
	covered := map[string][]string{}
	for _, tc := range designReaderContractCases() {
		covered[tc.constructor] = append(covered[tc.constructor], tc.mode)
	}
	for constructor, class := range classification {
		sort.Strings(class.readerModes)
		got := covered[constructor]
		sort.Strings(got)
		if strings.Join(got, "\x00") != strings.Join(class.readerModes, "\x00") {
			t.Errorf("%s behavioral reader modes = %v, want %v", constructor, got, class.readerModes)
		}
	}
}

const designAccessMarker = "DESIGN_DERIVED_SHOULD_NOT_APPEAR"

func seedDesignAccessContractDesign(t *testing.T) string {
	t.Helper()
	design := t.TempDir()
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
	write("domain.modelith.yaml", "kind: DomainModel\nversion: v1\ntitle: Contract Test\nentities: {}\n")
	write("machines/Contract.machine.json", `{"id":"Contract","initial":"Done","states":{"Done":{"type":"final"}}}`+"\n")
	write("formal/.keep", "keeps the formal directory present\n")
	write("workspace.dsl", "workspace { model { } views { } }\n")
	write("ARCHITECTURE.md", "# Architecture\n\n"+designAccessMarker+"\n")
	return design
}

func seedDesignAccessResidue(t *testing.T, design, rel string) {
	t.Helper()
	path := filepath.Join(design, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "seeded interrupted transaction\n"
	if rel == ".machinery-design-publish.json" {
		body = `{"version":1,"operation":"contract-test","recovery":"rerun the interrupted writer","input_fingerprint":"","outputs":[]}` + "\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func designAccessTreeState(t *testing.T, root string) string {
	t.Helper()
	var rows []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			rows = append(rows, filepath.ToSlash(rel)+"/ "+info.Mode().String())
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rows = append(rows, fmt.Sprintf("%s %s %x", filepath.ToSlash(rel), info.Mode(), body))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(rows)
	return strings.Join(rows, "\n")
}

type designReaderContractCase struct {
	constructor string
	mode        string
	new         func() *cobra.Command
	args        func(design, impl, registry string) []string
}

func designReaderContractCases() []designReaderContractCase {
	return []designReaderContractCase{
		{"newLintCmd", "machines", newLintCmd, func(design, _, _ string) []string { return []string{filepath.Join(design, "machines")} }},
		{"newOracleCmd", "diff", newOracleCmd, func(design, _, _ string) []string { return []string{"--diff", filepath.Join(design, "machines")} }},
		{"newOracleCmd", "diff-against", newOracleCmd, func(design, _, _ string) []string {
			return []string{"--diff", "--against", "HEAD", filepath.Join(design, "machines")}
		}},
		{"newCheckCmd", "default", newCheckCmd, func(design, _, _ string) []string { return []string{design} }},
		{"newCheckCmd", "selected-gate", newCheckCmd, func(design, _, _ string) []string { return []string{"--gate", "g2", design} }},
		{"newCheckCmd", "complete", newCheckCmd, func(design, impl, _ string) []string { return []string{"--complete", "--impl", impl, design} }},
		{"newVerifyCheckersCmd", "all", newVerifyCheckersCmd, func(design, _, registry string) []string { return []string{"--registry", registry, design} }},
		{"newVerifyCheckersCmd", "selected", newVerifyCheckersCmd, func(design, _, registry string) []string {
			return []string{"--registry", registry, "--checker", "test", design}
		}},
		{"newVerifyC4Cmd", "default", newVerifyC4Cmd, func(design, _, _ string) []string { return []string{design} }},
		{"newEmbedCmd", "refresh-dry-run", newEmbedCmd, func(design, _, _ string) []string { return []string{"refresh", "--dry-run", design} }},
		{"newScaleCmd", "default", newScaleCmd, func(design, _, _ string) []string { return []string{design} }},
		{"newSweepCmd", "default", newSweepCmd, func(design, _, _ string) []string { return []string{designAccessMarker, design} }},
		{"newIRDumpCmd", "default", newIRDumpCmd, func(design, _, _ string) []string {
			return []string{filepath.Join(design, "machines", "Contract.machine.json")}
		}},
	}
}

// Readers must fail before they expose even one design-derived byte whenever
// an owned publication family has crash residue. The command inventory is
// behavioral (not merely a list of constructors), and deliberately names the
// read-only variants of mixed commands: oracle --diff and embed --dry-run.
func TestEveryDesignReaderBlocksOnInterruptedPublication(t *testing.T) {
	residues := []string{
		".machinery-design-publish.json",
		"formal/.machinery-formal-transaction.jsonl",
	}
	for _, residue := range residues {
		for _, tc := range designReaderContractCases() {
			t.Run(residue+"/"+tc.constructor+"/"+tc.mode, func(t *testing.T) {
				design := seedDesignAccessContractDesign(t)
				seedDesignAccessResidue(t, design, residue)
				impl := t.TempDir()
				registry := filepath.Join(t.TempDir(), "checkers.local.yaml")
				registryBody := "checkers:\n  test:\n    runtime:\n      kind: oci\n      engine: [machinery-contract-engine]\n      image: example.invalid/test@sha256:1111111111111111111111111111111111111111111111111111111111111111\n      platform: linux/amd64\n    run: [checker, '{out}']\n"
				if err := os.WriteFile(registry, []byte(registryBody), 0o600); err != nil {
					t.Fatal(err)
				}
				before := designAccessTreeState(t, design)
				out, errOut, codes := withCapturedIO(t)
				cmd := tc.new()
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
				var cobraOut bytes.Buffer
				cmd.SetOut(&cobraOut)
				cmd.SetErr(&cobraOut)
				cmd.SetArgs(tc.args(design, impl, registry))
				runErr := executeCapturedCommand(cmd)
				diagnostic := out.String() + errOut.String() + cobraOut.String()
				if runErr != nil {
					diagnostic += runErr.Error()
				}
				if !strings.Contains(diagnostic, "interrupted Machinery publication") {
					t.Fatalf("command did not fail on recognized residue; exit codes=%v error=%v output=%q", *codes, runErr, diagnostic)
				}
				if strings.Contains(diagnostic, designAccessMarker) {
					t.Fatalf("command emitted design-derived content before refusing the interrupted publication: %q", diagnostic)
				}
				if after := designAccessTreeState(t, design); after != before {
					t.Fatalf("read-only command mutated the design while refusing interrupted publication\nbefore:\n%s\nafter:\n%s", before, after)
				}
			})
		}
	}
}
