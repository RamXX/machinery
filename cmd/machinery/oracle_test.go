package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/testgit"
)

func writeMachine(t *testing.T, dir, name, src string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func oracleRunFilesForTest(files []string) error {
	return executeDirectCaptured(oracleRunFilesTo(files, false, "", stdoutW, stderrW))
}

func TestOracleRefusesLintFailingMachine(t *testing.T) {
	// IR-F14: `machinery oracle` generated from machines that fail lint
	// (silently narrowing array targets to their first element, among others).
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Bad.machine.json", `{"id":"m14","initial":"Idle","states":{
		"Idle":{"on":{"GO":{"target":["Done","Other"]}}},
		"Done":{"type":"final"},
		"Other":{"type":"final"}}}`)
	_ = capturedOracleRun(d, false, "")
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "fails lint") {
		t.Fatalf("stderr %q", errB.String())
	}
	if leftovers, _ := filepath.Glob(filepath.Join(d, "*.oracle.md")); len(leftovers) != 0 {
		t.Fatalf("oracle written for a lint-failing machine: %v", leftovers)
	}
}

func TestOracleNonCheckCommandRejectsNonportableDesignInventory(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	design := t.TempDir()
	machines := filepath.Join(design, "machines")
	if err := os.MkdirAll(machines, 0o755); err != nil {
		t.Fatal(err)
	}
	writeMachine(t, machines, "CON.machine.json", validMachineA)
	if err := capturedOracleRun(machines, false, ""); err == nil {
		t.Fatal("oracle accepted a nonportable inventory before source discovery")
	}
	if len(*codes) == 0 || (*codes)[0] != 1 || !strings.Contains(errB.String(), "non-portable design path") {
		t.Fatalf("codes=%v stderr=%q", *codes, errB.String())
	}
}

func TestOracleTagCollisionIsError(t *testing.T) {
	// IR-F12: Deal and DealAggregate both derive the tag DEAL; identical
	// stable ids across machines in one design must be a hard error.
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Deal.machine.json", `{"id":"deal","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Won":{"type":"final"}}}`)
	writeMachine(t, d, "DealAggregate.machine.json", `{"id":"dealAggregate","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Closed","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Closed":{"type":"final"}}}`)
	_ = capturedOracleRun(d, false, "")
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "_oracle_tag") {
		t.Fatalf("stderr should point at _oracle_tag, got %q", errB.String())
	}
}

func TestOracleMultipleTagCollisionDiagnosticsAreDeterministic(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Deal.machine.json", `{"id":"deal","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Won":{"type":"final"}}}`)
	writeMachine(t, d, "DealAggregate.machine.json", `{"id":"dealAggregate","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Closed","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Closed":{"type":"final"}}}`)
	writeMachine(t, d, "Order.machine.json", `{"id":"order","initial":"Open","states":{
		"Open":{"on":{"close":{"target":"Closed","guard":"canClose"}},"_refusal":{"close":"fixture: the command boundary refuses when canClose is false"}},
		"Closed":{"type":"final"}}}`)
	writeMachine(t, d, "OrderAggregate.machine.json", `{"id":"orderAggregate","initial":"Open","states":{
		"Open":{"on":{"close":{"target":"Done","guard":"canClose"}},"_refusal":{"close":"fixture: the command boundary refuses when canClose is false"}},
		"Done":{"type":"final"}}}`)

	var first string
	for i := 0; i < 100; i++ {
		errB.Reset()
		*codes = nil
		_ = capturedOracleRun(d, false, "")
		if len(*codes) != 1 || (*codes)[0] != 1 {
			t.Fatalf("run %d exit codes %v, want [1]", i, *codes)
		}
		got := errB.String()
		if i == 0 {
			first = got
		} else if got != first {
			t.Fatalf("run %d diagnostic changed:\nfirst: %q\n got: %q", i, first, got)
		}
	}
	deal := strings.Index(first, "stable-id tag DEAL")
	order := strings.Index(first, "stable-id tag ORDE")
	if deal < 0 || order < 0 || deal > order {
		t.Fatalf("contested tags are not reported in sorted order: %q", first)
	}
}

func TestOracleTagOverrideDisambiguates(t *testing.T) {
	out, _, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Deal.machine.json", `{"id":"deal","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Won":{"type":"final"}}}`)
	writeMachine(t, d, "DealAggregate.machine.json", `{"id":"dealAggregate","_oracle_tag":"DEALAGG","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Closed","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Closed":{"type":"final"}}}`)
	if err := capturedOracleRun(d, false, ""); err != nil {
		t.Fatalf("oracleRun: %v (stdout %q)", err, out.String())
	}
	if len(*codes) != 0 {
		t.Fatalf("exit codes %v, want none", *codes)
	}
	body, err := os.ReadFile(filepath.Join(d, "DealAggregate.oracle.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "DEALAGG-") {
		t.Fatalf("override tag missing from generated oracle:\n%s", body)
	}
}

const validMachineA = `{"id":"alpha","initial":"Lead","states":{
	"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
	"Won":{"type":"final"}}}`

const validMachineB = `{"id":"beta","initial":"Open","states":{
	"Open":{"on":{"close":{"target":"Closed","guard":"canClose"}},"_refusal":{"close":"fixture: the command boundary refuses when canClose is false"}},
	"Closed":{"type":"final"}}}`

func TestOracleLateInvalidMachineLeavesCompleteArtifactSetUnchanged(t *testing.T) {
	// Validation continues through the complete inventory, but one late
	// invalid machine aborts every planned oracle replacement.
	out, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Alpha.machine.json", validMachineA)
	writeMachine(t, d, "Beta.machine.json", validMachineB)
	writeMachine(t, d, "ZBroken.machine.json", `{"id":"broken","initial":`)
	alphaOracle := filepath.Join(d, "Alpha.oracle.md")
	if err := os.WriteFile(alphaOracle, []byte("old-alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = capturedOracleRun(d, false, "")
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	got, err := os.ReadFile(alphaOracle)
	if err != nil || string(got) != "old-alpha\n" {
		t.Fatalf("earlier oracle changed after late validation failure: %q, %v (stdout %q, stderr %q)", got, err, out.String(), errB.String())
	}
	for _, absent := range []string{"Beta.oracle.md", "ZBroken.oracle.md"} {
		if _, err := os.Stat(filepath.Join(d, absent)); !os.IsNotExist(err) {
			t.Fatalf("%s written by failed transaction: %v", absent, err)
		}
	}
	if !strings.Contains(errB.String(), "Broken") && !strings.Contains(errB.String(), "broken") {
		t.Fatalf("broken machine not named on stderr: %q", errB.String())
	}
	if !strings.Contains(errB.String(), "no oracles were regenerated") {
		t.Fatalf("failure summary missing: %q", errB.String())
	}
}

func TestOracleSingleFileMode(t *testing.T) {
	// S12: a named file regenerates exactly that file, leaving siblings alone.
	out, _, codes := withCapturedIO(t)
	d := t.TempDir()
	fa := writeMachine(t, d, "Alpha.machine.json", validMachineA)
	writeMachine(t, d, "Beta.machine.json", validMachineB)
	if err := oracleRunFilesForTest([]string{fa}); err != nil {
		t.Fatalf("oracleRunFiles: %v (stdout %q)", err, out.String())
	}
	if len(*codes) != 0 {
		t.Fatalf("exit codes %v, want none", *codes)
	}
	if _, err := os.Stat(filepath.Join(d, "Alpha.oracle.md")); err != nil {
		t.Fatal("named file's oracle not generated")
	}
	if _, err := os.Stat(filepath.Join(d, "Beta.oracle.md")); err == nil {
		t.Fatal("unnamed sibling's oracle generated by a per-file run")
	}
}

func TestOracleSingleFileReservesSiblingTags(t *testing.T) {
	// S12: a per-file run must not silently mint a stable-id tag a directory
	// sibling already owns; both ids share the derived tag here.
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	fa := writeMachine(t, d, "Deal.machine.json", `{"id":"deal","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Won":{"type":"final"}}}`)
	writeMachine(t, d, "DealAggregate.machine.json", `{"id":"dealAggregate","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Closed","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Closed":{"type":"final"}}}`)
	_ = oracleRunFilesForTest([]string{fa})
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "_oracle_tag") {
		t.Fatalf("stderr should point at _oracle_tag, got %q", errB.String())
	}
	if _, err := os.Stat(filepath.Join(d, "Deal.oracle.md")); err == nil {
		t.Fatal("oracle generated for a machine whose tag a sibling contests")
	}
}

func TestOracleLintFailureAbortsCompleteArtifactSet(t *testing.T) {
	// A lint failure is reported without leaving a valid sibling freshly
	// generated beside a stale or absent invalid sibling.
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Alpha.machine.json", validMachineA)
	writeMachine(t, d, "Bad.machine.json", `{"id":"m14","initial":"Idle","states":{
		"Idle":{"on":{"GO":{"target":["Done","Other"]}}},
		"Done":{"type":"final"},
		"Other":{"type":"final"}}}`)
	_ = capturedOracleRun(d, false, "")
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if _, err := os.Stat(filepath.Join(d, "Alpha.oracle.md")); !os.IsNotExist(err) {
		t.Fatalf("valid sibling generated by failed transaction: %v", err)
	}
	if _, err := os.Stat(filepath.Join(d, "Bad.oracle.md")); err == nil {
		t.Fatal("oracle written for a lint-failing machine")
	}
	if !strings.Contains(errB.String(), "fails lint") {
		t.Fatalf("stderr %q", errB.String())
	}
}

func TestOracleNonMachineFileArgIsError(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	p := writeMachine(t, d, "notes.txt", "hello")
	_ = oracleRunFilesForTest([]string{p})
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "not a *.machine.json") {
		t.Fatalf("stderr %q", errB.String())
	}
}

func TestOracleFailsClosedOnMalformedUnselectedSibling(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	selected := writeMachine(t, d, "Alpha.machine.json", validMachineA)
	writeMachine(t, d, "Broken.machine.json", `{not json`)
	_ = oracleRunFilesForTest([]string{selected})
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "sibling machine Broken.machine.json is unreadable or malformed") {
		t.Fatalf("malformed sibling was omitted from closed inventory: %q", errB.String())
	}
	if _, err := os.Lstat(filepath.Join(d, "Alpha.oracle.md")); !os.IsNotExist(err) {
		t.Fatalf("malformed sibling allowed selected oracle publication: %v", err)
	}
}

func TestOracleFailsClosedOnSymlinkedUnselectedSibling(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	selected := writeMachine(t, d, "Alpha.machine.json", validMachineA)
	target := writeMachine(t, t.TempDir(), "Beta.machine.json", validMachineB)
	if err := os.Symlink(target, filepath.Join(d, "Beta.machine.json")); err != nil {
		t.Fatal(err)
	}
	_ = oracleRunFilesForTest([]string{selected})
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "symlink") {
		t.Fatalf("symlinked sibling was omitted from closed inventory: %q", errB.String())
	}
	if _, err := os.Lstat(filepath.Join(d, "Alpha.oracle.md")); !os.IsNotExist(err) {
		t.Fatalf("symlinked sibling allowed selected oracle publication: %v", err)
	}
}

func TestOracleFailsClosedOnSiblingModuleIdentityMismatch(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	selected := writeMachine(t, d, "Alpha.machine.json", validMachineA)
	writeMachine(t, d, "Wrong.machine.json", validMachineB)
	_ = oracleRunFilesForTest([]string{selected})
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), `filename stem "Wrong" does not match canonical machine id/module identity "Beta"`) {
		t.Fatalf("sibling module alias was omitted from closed inventory: %q", errB.String())
	}
	if _, err := os.Lstat(filepath.Join(d, "Alpha.oracle.md")); !os.IsNotExist(err) {
		t.Fatalf("aliased sibling allowed selected oracle publication: %v", err)
	}
}

func TestOracleFailsClosedOnDuplicateSiblingStableIDNamespace(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	selected := writeMachine(t, d, "Alpha.machine.json", validMachineA)
	writeMachine(t, d, "Beta.machine.json", `{"id":"beta","_oracle_tag":"ALPH","initial":"Open","states":{
		"Open":{"on":{"close":{"target":"Closed","guard":"canClose"}},"_refusal":{"close":"fixture: the command boundary refuses when canClose is false"}},
		"Closed":{"type":"final"}}}`)
	_ = oracleRunFilesForTest([]string{selected})
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "stable-id tag ALPH") || !strings.Contains(errB.String(), "Alpha.machine.json and Beta.machine.json") {
		t.Fatalf("duplicate sibling stable-id namespace was not diagnosed deterministically: %q", errB.String())
	}
	if _, err := os.Lstat(filepath.Join(d, "Alpha.oracle.md")); !os.IsNotExist(err) {
		t.Fatalf("duplicate sibling stable-id namespace allowed selected oracle publication: %v", err)
	}
}

func TestOracleEmptySelectionIsError(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	_ = oracleRunFilesForTest(nil)
	if len(*codes) == 0 || (*codes)[0] != 1 || !strings.Contains(errB.String(), "no machine files selected") {
		t.Fatalf("empty selection not rejected: codes=%v stderr=%q", *codes, errB.String())
	}
}

func TestOracleRejectsEveryNonPortableMachineNameBeforeMutation(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	files := []string{
		filepath.Join(d, "CON.machine.json"),
		filepath.Join(d, "naïve.machine.json"),
		filepath.Join(d, "Trailing..machine.json"),
	}
	_ = oracleRunFilesForTest(files)
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	for _, want := range []string{"CON.machine.json", "naïve.machine.json", "Trailing."} {
		if !strings.Contains(errB.String(), want) {
			t.Fatalf("portable-name diagnostics omit %q: %q", want, errB.String())
		}
	}
	if matches, _ := filepath.Glob(filepath.Join(d, "*.oracle.md")); len(matches) != 0 {
		t.Fatalf("portable-name failure wrote artifacts: %v", matches)
	}
}

func TestOracleRejectsFilenameMachineIdentityMismatch(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Wrong.machine.json", validMachineA)
	oracle := filepath.Join(d, "Wrong.oracle.md")
	if err := os.WriteFile(oracle, []byte("identity-sentinel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = capturedOracleRun(d, false, "")
	if len(*codes) == 0 || (*codes)[0] != 1 || !strings.Contains(errB.String(), "canonical machine id/module identity") {
		t.Fatalf("identity mismatch not rejected: codes=%v stderr=%q", *codes, errB.String())
	}
	got, err := os.ReadFile(oracle)
	if err != nil || string(got) != "identity-sentinel\n" {
		t.Fatalf("identity mismatch mutated oracle: %q, %v", got, err)
	}
}

func TestOracleArtifactSetRejectsSymlinkTargetWithoutMutation(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Alpha.machine.json", validMachineA)
	outside := filepath.Join(t.TempDir(), "outside.oracle.md")
	if err := os.WriteFile(outside, []byte("sentinel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(d, "Alpha.oracle.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_ = capturedOracleRun(d, false, "")
	if len(*codes) == 0 || (*codes)[0] != 1 || !strings.Contains(errB.String(), "symlink") {
		t.Fatalf("symlink target not rejected: codes=%v stderr=%q", *codes, errB.String())
	}
	got, err := os.ReadFile(outside)
	if err != nil || string(got) != "sentinel\n" {
		t.Fatalf("symlink referent mutated: %q, %v", got, err)
	}
}

func TestOracleRejectsSymlinkMachineSourceWithoutMutation(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	realMachine := writeMachine(t, t.TempDir(), "Alpha.machine.json", validMachineA)
	linkedMachine := filepath.Join(d, "Alpha.machine.json")
	if err := os.Symlink(realMachine, linkedMachine); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	oracle := filepath.Join(d, "Alpha.oracle.md")
	if err := os.WriteFile(oracle, []byte("source-sentinel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = oracleRunFilesForTest([]string{linkedMachine})
	if len(*codes) == 0 || (*codes)[0] != 1 || !strings.Contains(errB.String(), "regular machine file") {
		t.Fatalf("symlink source not rejected: codes=%v stderr=%q", *codes, errB.String())
	}
	got, err := os.ReadFile(oracle)
	if err != nil || string(got) != "source-sentinel\n" {
		t.Fatalf("symlink source failure mutated oracle: %q, %v", got, err)
	}
}

func TestOracleDiffRejectsInterruptedPublicationBeforeMachineReads(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	design := t.TempDir()
	machines := filepath.Join(design, "machines")
	if err := os.MkdirAll(machines, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(machines, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(machines, ".machinery-artifact-set.journal"), []byte("seeded\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = capturedOracleRun(machines, true, "")
	if len(*codes) == 0 || (*codes)[0] != 1 || !strings.Contains(errB.String(), "interrupted Machinery publication") {
		t.Fatalf("oracle --diff read through interrupted publication: codes=%v stderr=%q", *codes, errB.String())
	}
}

func TestOracleRemovesStaleOwnedOutputAndPreservesManualOracle(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	design := t.TempDir()
	machines := filepath.Join(design, "machines")
	if err := os.MkdirAll(machines, 0o755); err != nil {
		t.Fatal(err)
	}
	writeMachine(t, machines, "Alpha.machine.json", validMachineA)
	writeMachine(t, machines, "Beta.machine.json", validMachineB)
	if err := os.WriteFile(filepath.Join(machines, "Manual.oracle.md"), []byte("# manually curated oracle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := capturedOracleRun(machines, false, ""); err != nil {
		t.Fatalf("initial generation: %v (%s)", err, errB.String())
	}
	if err := os.Remove(filepath.Join(machines, "Beta.machine.json")); err != nil {
		t.Fatal(err)
	}
	if err := capturedOracleRun(machines, false, ""); err != nil {
		t.Fatalf("convergent generation: %v (%s)", err, errB.String())
	}
	if _, err := os.Lstat(filepath.Join(machines, "Beta.oracle.md")); !os.IsNotExist(err) {
		t.Fatalf("stale owned oracle retained: %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(machines, "Manual.oracle.md")); err != nil || string(body) != "# manually curated oracle\n" {
		t.Fatalf("manual oracle changed: %q %v", body, err)
	}
	if len(*codes) != 0 {
		t.Fatalf("successful convergence emitted exit codes %v", *codes)
	}
	if err := capturedOracleRun(machines, false, ""); err != nil {
		t.Fatalf("idempotent rerun: %v", err)
	}
}

func TestOracleStaleOwnershipRequiresCanonicalLeadingIdentity(t *testing.T) {
	d := t.TempDir()
	keep := map[string][]byte{}

	handwritten := filepath.Join(d, "Manual.oracle.md")
	handwrittenBody := "# Handwritten notes\n\nQuoted example only:\nGenerated from `Manual.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.\n"
	if err := os.WriteFile(handwritten, []byte(handwrittenBody), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, err := staleOwnedOracles(d, keep)
	if err != nil {
		t.Fatalf("handwritten body marker must not authorize deletion: %v", err)
	}
	if len(stale) != 0 {
		t.Fatalf("handwritten body marker planned deletion: %#v", stale)
	}

	foreign := filepath.Join(d, "Beta.oracle.md")
	foreignBody := "# Generated transition oracle: `Wrong`\n\nGenerated from `Beta.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.\n<!-- machinery-version: test -->\n"
	if err := os.WriteFile(foreign, []byte(foreignBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := staleOwnedOracles(d, keep); err == nil || !strings.Contains(err.Error(), "title identity") {
		t.Fatalf("inconsistent title identity must fail closed, got %v", err)
	}
	got, err := os.ReadFile(foreign)
	if err != nil || string(got) != foreignBody {
		t.Fatalf("foreign inconsistent-title oracle changed: %q, %v", got, err)
	}
}

func TestOracleStaleReplacementFailsBeforeAnyMutation(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Alpha.machine.json", validMachineA)
	writeMachine(t, d, "Beta.machine.json", validMachineB)
	if err := capturedOracleRun(d, false, ""); err != nil {
		t.Fatal(err)
	}
	alphaPath := filepath.Join(d, "Alpha.oracle.md")
	alphaBefore, err := os.ReadFile(alphaPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(d, "Beta.machine.json")); err != nil {
		t.Fatal(err)
	}
	betaPath := filepath.Join(d, "Beta.oracle.md")
	prior := oracleAfterStalePlan
	oracleAfterStalePlan = func() {
		replacement := filepath.Join(d, ".replacement")
		if err := os.WriteFile(replacement, []byte("foreign replacement\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(replacement, betaPath); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { oracleAfterStalePlan = prior }()
	if err := capturedOracleRun(d, false, ""); err == nil {
		t.Fatal("post-inspection replacement must fail")
	}
	if len(*codes) == 0 || (*codes)[0] != 1 || (!strings.Contains(errB.String(), "changed after it was inspected") && !strings.Contains(errB.String(), "changed outside the snapshot lock")) {
		t.Fatalf("replacement failure not surfaced deterministically: codes=%v stderr=%q", *codes, errB.String())
	}
	if got, err := os.ReadFile(betaPath); err != nil || string(got) != "foreign replacement\n" {
		t.Fatalf("foreign replacement changed: %q, %v", got, err)
	}
	if got, err := os.ReadFile(alphaPath); err != nil || !bytes.Equal(got, alphaBefore) {
		t.Fatalf("other oracle partially mutated: %q, %v", got, err)
	}
}

func TestOracleArtifactSetRejectsPortableAliasWithoutMutation(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Alpha.machine.json", validMachineA)
	alias := filepath.Join(d, "alpha.oracle.md")
	if err := os.WriteFile(alias, []byte("portable-sentinel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = capturedOracleRun(d, false, "")
	if len(*codes) == 0 || (*codes)[0] != 1 || !strings.Contains(errB.String(), "portable") {
		t.Fatalf("portable alias not rejected: codes=%v stderr=%q", *codes, errB.String())
	}
	got, err := os.ReadFile(alias)
	if err != nil || string(got) != "portable-sentinel\n" {
		t.Fatalf("portable alias mutated: %q, %v", got, err)
	}
}

func TestOracleWritableSelectionMustBeOneArtifactTransaction(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	a, b := t.TempDir(), t.TempDir()
	fa := writeMachine(t, a, "Alpha.machine.json", validMachineA)
	fb := writeMachine(t, b, "Beta.machine.json", validMachineB)
	alphaOracle := filepath.Join(a, "Alpha.oracle.md")
	if err := os.WriteFile(alphaOracle, []byte("cross-dir-sentinel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = oracleRunFilesForTest([]string{fa, fb})
	if len(*codes) == 0 || (*codes)[0] != 1 || !strings.Contains(errB.String(), "all-or-rollback artifact transaction") {
		t.Fatalf("cross-directory write set not rejected: codes=%v stderr=%q", *codes, errB.String())
	}
	got, err := os.ReadFile(alphaOracle)
	if err != nil || string(got) != "cross-dir-sentinel\n" {
		t.Fatalf("cross-directory rejection mutated first oracle: %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(b, "Beta.oracle.md")); !os.IsNotExist(err) {
		t.Fatalf("cross-directory rejection wrote second oracle: %v", err)
	}
}

func TestOracleDiffClassifiesChurn(t *testing.T) {
	outB, _, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Thing.machine.json", `{"id":"thing","initial":"A","states":{
		"A":{"on":{"go":{"target":"B"},"stop":{"target":"C"}}},
		"B":{"type":"final"},
		"C":{"type":"final"}}}`)
	_ = capturedOracleRun(d, false, "")
	if len(*codes) != 0 {
		t.Fatalf("generation failed: %v", *codes)
	}
	outB.Reset()
	_ = capturedOracleRun(d, true, "")
	if !strings.Contains(outB.String(), "no churn") {
		t.Fatalf("a fresh oracle must diff clean, got %q", outB.String())
	}
	committed, err := os.ReadFile(filepath.Join(d, "Thing.oracle.md"))
	if err != nil {
		t.Fatal(err)
	}
	// drop one transition, add another: the diff owes a deleted and a new id
	writeMachine(t, d, "Thing.machine.json", `{"id":"thing","initial":"A","states":{
		"A":{"on":{"go":{"target":"B"},"pause":{"target":"C"}}},
		"B":{"type":"final"},
		"C":{"type":"final"}}}`)
	outB.Reset()
	_ = capturedOracleRun(d, true, "")
	out := outB.String()
	if !strings.Contains(out, "new") || !strings.Contains(out, "deleted") {
		t.Fatalf("churn must classify as new+deleted, got %q", out)
	}
	after, err := os.ReadFile(filepath.Join(d, "Thing.oracle.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(committed) {
		t.Fatal("--diff must not rewrite the committed oracle")
	}
}

func TestOracleDiffBaselineIsSnapshotBoundAndFailsClosed(t *testing.T) {
	t.Run("immutable baseline replacement", func(t *testing.T) {
		outB, errB, codes := withCapturedIO(t)
		d := t.TempDir()
		writeMachine(t, d, "Alpha.machine.json", validMachineA)
		if err := capturedOracleRun(d, false, ""); err != nil {
			t.Fatal(err)
		}
		outB.Reset()
		prior := oracleAfterDiffBaselineInspect
		oracleAfterDiffBaselineInspect = func(source string) {
			replacement := filepath.Join(filepath.Dir(source), "replacement")
			if err := os.WriteFile(replacement, []byte("foreign\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(replacement, source); err != nil {
				t.Fatal(err)
			}
		}
		defer func() { oracleAfterDiffBaselineInspect = prior }()
		if err := capturedOracleRun(d, true, ""); err == nil {
			t.Fatal("replaced immutable baseline was accepted")
		}
		if outB.Len() != 0 || len(*codes) == 0 || !strings.Contains(errB.String(), "changed identity") {
			t.Fatalf("replacement did not fail before diff output: stdout=%q stderr=%q codes=%v", outB.String(), errB.String(), *codes)
		}
	})

	t.Run("live mutation after materialization", func(t *testing.T) {
		outB, errB, codes := withCapturedIO(t)
		d := t.TempDir()
		writeMachine(t, d, "Alpha.machine.json", validMachineA)
		if err := capturedOracleRun(d, false, ""); err != nil {
			t.Fatal(err)
		}
		outB.Reset()
		prior := oracleAfterDiffBaselineInspect
		oracleAfterDiffBaselineInspect = func(string) {
			if err := os.WriteFile(filepath.Join(d, "Alpha.oracle.md"), []byte("live mutation\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		defer func() { oracleAfterDiffBaselineInspect = prior }()
		if err := capturedOracleRun(d, true, ""); err == nil {
			t.Fatal("live baseline mutation was accepted")
		}
		if outB.Len() != 0 || len(*codes) == 0 || !strings.Contains(errB.String(), "design changed during --diff") {
			t.Fatalf("live mutation did not suppress stale diff: stdout=%q stderr=%q codes=%v", outB.String(), errB.String(), *codes)
		}
	})

	t.Run("unreadable baseline", func(t *testing.T) {
		outB, _, _ := withCapturedIO(t)
		d := t.TempDir()
		writeMachine(t, d, "Alpha.machine.json", validMachineA)
		if err := capturedOracleRun(d, false, ""); err != nil {
			t.Fatal(err)
		}
		baseline := filepath.Join(d, "Alpha.oracle.md")
		if err := os.Chmod(baseline, 0); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(baseline, 0o644) //nolint:errcheck // best-effort test cleanup
		outB.Reset()
		if err := capturedOracleRun(d, true, ""); err == nil {
			t.Fatal("unreadable baseline was collapsed to absent")
		}
		if strings.Contains(outB.String(), "new oracle") {
			t.Fatalf("unreadable baseline was reported as absent: %q", outB.String())
		}
	})
}

func TestOracleDiffRenameShapedChurn(t *testing.T) {
	outB, _, codes := withCapturedIO(t)
	d := t.TempDir()
	base := `{"id":"thing",%s"initial":"A","states":{
		"A":{"on":{"go":{"target":"B"}}},
		"B":{"type":"final"}}}`
	writeMachine(t, d, "Thing.machine.json", strings.Replace(base, "%s", "", 1))
	_ = capturedOracleRun(d, false, "")
	if len(*codes) != 0 {
		t.Fatalf("generation failed: %v", *codes)
	}
	// an _oracle_tag change churns every stable id with identical row content:
	// exactly the rename-shaped class the revision protocol maps, never
	// processes as delete-all-plus-new
	writeMachine(t, d, "Thing.machine.json", strings.Replace(base, "%s", `"_oracle_tag":"WIDG",`, 1))
	outB.Reset()
	_ = capturedOracleRun(d, true, "")
	if !strings.Contains(outB.String(), "rename-shaped") {
		t.Fatalf("identical-content id churn must classify as rename-shaped, got %q", outB.String())
	}
}

func TestTokensEqual(t *testing.T) {
	outB, _, codes := withCapturedIO(t)
	d := t.TempDir()
	a := filepath.Join(d, "a.md")
	b := filepath.Join(d, "b.md")
	c := filepath.Join(d, "c.md")
	if err := os.WriteFile(a, []byte("one two\nthree   four\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("  one\ntwo three\nfour"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c, []byte("one two\nthree five\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := capturedTokensEqualRun(a, b); err != nil {
		t.Fatalf("reflow is formatting-only: %v", err)
	}
	if !strings.Contains(outB.String(), "token-identical: 4 tokens") {
		t.Fatalf("got %q", outB.String())
	}
	outB.Reset()
	_ = capturedTokensEqualRun(a, c)
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("a wording change must exit 1: %v", *codes)
	}
	if !strings.Contains(outB.String(), "NOT token-identical") {
		t.Fatalf("got %q", outB.String())
	}
}

func TestTokensEqualRejectsMutationDuringComparison(t *testing.T) {
	_, _, codes := withCapturedIO(t)
	d := t.TempDir()
	a := filepath.Join(d, "a.md")
	b := filepath.Join(d, "b.md")
	for _, path := range []string{a, b} {
		if err := os.WriteFile(path, []byte("one two"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	prior := stableRegularAfterInitialRead
	stableRegularAfterInitialRead = func(path string) {
		if path == a {
			if err := os.WriteFile(a, []byte("one six"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	defer func() { stableRegularAfterInitialRead = prior }()
	if err := capturedTokensEqualRun(a, b); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("tokens-equal hid concurrent mutation: %v", err)
	}
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("tokens-equal mutation did not exit 1: %v", *codes)
	}
}

// gitInit makes dir a repository with one commit of everything in it. It
// skips the test when git is unavailable, since --against is a git feature.
func gitInit(t *testing.T, dir string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@example.invalid"},
		{"config", "user.name", "t"},
		{"add", "-A"},
		{"-c", "commit.gpgsign=false", "commit", "-q", "-m", "baseline"},
	} {
		if out, err := testgit.Run(t.Context(), dir, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func TestOracleDiffAgainstRef(t *testing.T) {
	outB, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Thing.machine.json", `{"id":"thing","initial":"A","states":{
		"A":{"on":{"go":{"target":"B"},"stop":{"target":"C"}}},
		"B":{"type":"final"},
		"C":{"type":"final"}}}`)
	if err := capturedOracleRun(d, false, ""); err != nil {
		t.Fatal(err)
	}
	gitInit(t, d)

	// the churn: one transition renamed, and the regeneration ALREADY written,
	// which is exactly the state where plain --diff reports nothing.
	writeMachine(t, d, "Thing.machine.json", `{"id":"thing","initial":"A","states":{
		"A":{"on":{"go":{"target":"B"},"pause":{"target":"C"}}},
		"B":{"type":"final"},
		"C":{"type":"final"}}}`)
	outB.Reset()
	if err := capturedOracleRun(d, false, ""); err != nil {
		t.Fatal(err)
	}
	outB.Reset()
	if err := capturedOracleRun(d, true, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(outB.String(), "no churn") {
		t.Fatalf("after a regeneration, plain --diff sees nothing; got %q", outB.String())
	}
	// --against HEAD recovers the affected-test list
	outB.Reset()
	if err := capturedOracleRun(d, true, "HEAD"); err != nil {
		t.Fatalf("oracleRun --against: %v (stderr %q)", err, errB.String())
	}
	out := outB.String()
	if !strings.Contains(out, "new ") || !strings.Contains(out, "deleted ") {
		t.Fatalf("--against HEAD must classify new+deleted, got %q", out)
	}
	if len(*codes) != 0 {
		t.Fatalf("a clean --against run must not exit non-zero: %v", *codes)
	}
}

func TestOracleDiffAgainstPinsMovingBranchAcrossAllMachines(t *testing.T) {
	outB, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	for _, name := range []string{"Alpha", "Beta"} {
		writeMachine(t, d, name+".machine.json", fmt.Sprintf(`{"id":%q,"initial":"A","states":{
			"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`, strings.ToLower(name)))
	}
	if err := capturedOracleRun(d, false, ""); err != nil {
		t.Fatal(err)
	}
	gitInit(t, d)
	git := func(args ...string) string {
		t.Helper()
		out, err := testgit.Run(t.Context(), d, args...)
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	baseline := git("rev-parse", "HEAD")
	for _, name := range []string{"Alpha", "Beta"} {
		writeMachine(t, d, name+".machine.json", fmt.Sprintf(`{"id":%q,"initial":"A","states":{
			"A":{"on":{"pause":{"target":"B"}}},"B":{"type":"final"}}}`, strings.ToLower(name)))
	}
	if err := capturedOracleRun(d, false, ""); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("-c", "commit.gpgsign=false", "commit", "-q", "-m", "changed")
	changed := git("rev-parse", "HEAD")
	git("reset", "--mixed", baseline)
	git("branch", "-f", "moving", baseline)

	priorHook := gitOracleReadHook
	reads := 0
	gitOracleReadHook = func() {
		reads++
		if reads == 1 {
			git("branch", "-f", "moving", changed)
		}
	}
	defer func() { gitOracleReadHook = priorHook }()
	outB.Reset()
	if err := capturedOracleRun(d, true, "moving"); err != nil {
		t.Fatalf("oracleRun moving branch: %v (stderr %q)", err, errB.String())
	}
	out := outB.String()
	if !strings.Contains(out, "baseline: "+baseline+" (resolved from moving)") {
		t.Fatalf("report did not display pinned baseline %s: %q", baseline, out)
	}
	for _, name := range []string{"Alpha.oracle.md", "Beta.oracle.md"} {
		if !strings.Contains(out, "== "+name) {
			t.Fatalf("moving branch produced a hybrid baseline; missing %s in %q", name, out)
		}
	}
	if reads != 2 {
		t.Fatalf("baseline blob reads = %d, want 2", reads)
	}
	if len(*codes) != 0 {
		t.Fatalf("pinned moving-ref run exited nonzero: %v", *codes)
	}
}

func TestOracleDiffAgainstRefFailsLoudly(t *testing.T) {
	cases := []struct {
		name    string
		ref     string
		commit  bool
		wantSub string
	}{
		{name: "unknown ref", ref: "no-such-ref", commit: true, wantSub: "does not resolve"},
		{name: "path absent at the ref", ref: "HEAD", commit: false, wantSub: "path did not exist there"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, errB, codes := withCapturedIO(t)
			d := t.TempDir()
			writeMachine(t, d, "Thing.machine.json", `{"id":"thing","initial":"A","states":{
				"A":{"on":{"go":{"target":"B"}}},
				"B":{"type":"final"}}}`)
			if tc.commit {
				if err := capturedOracleRun(d, false, ""); err != nil {
					t.Fatal(err)
				}
			}
			gitInit(t, d)
			if !tc.commit {
				// the oracle exists in the working tree but not at the ref
				if err := capturedOracleRun(d, false, ""); err != nil {
					t.Fatal(err)
				}
			}
			if err := capturedOracleRun(d, true, tc.ref); err == nil {
				t.Fatal("a baseline that cannot be read must fail loudly")
			}
			if len(*codes) == 0 || (*codes)[0] != 1 {
				t.Fatalf("exit codes %v, want [1]", *codes)
			}
			if !strings.Contains(errB.String(), tc.wantSub) {
				t.Fatalf("stderr %q does not name %q", errB.String(), tc.wantSub)
			}
		})
	}
}

func TestOracleAgainstWithoutDiffIsRefused(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	c := newOracleCmd()
	c.SilenceUsage, c.SilenceErrors = true, true
	c.SetArgs([]string{t.TempDir(), "--against", "HEAD"})
	if err := executeCapturedCommand(c); err != nil {
		t.Fatal(err)
	}
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "pass --diff too") {
		t.Fatalf("stderr %q", errB.String())
	}
}

func TestGitHelpersDegradeCleanly(t *testing.T) {
	t.Run("git environment forces deterministic locale", func(t *testing.T) {
		env := oracleGitEnvironment()
		last := map[string]string{}
		for _, entry := range env {
			if key, value, ok := strings.Cut(entry, "="); ok {
				last[key] = value
			}
		}
		if last["LC_ALL"] != "C" || last["LANG"] != "C" {
			t.Fatalf("git locale = LC_ALL=%q LANG=%q, want C/C", last["LC_ALL"], last["LANG"])
		}
		for _, blocked := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_OBJECT_DIRECTORY", "GIT_INDEX_FILE", "GIT_REPLACE_REF_BASE", "GIT_TRACE"} {
			if _, ok := last[blocked]; ok {
				t.Fatalf("closed git environment retained %s", blocked)
			}
		}
	})
	t.Run("realPath falls back to the path it was given", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "no-such-dir")
		if got := realPath(missing); got != missing {
			t.Fatalf("realPath(%q) = %q, want the input back", missing, got)
		}
	})
	t.Run("gitMessage renders a non-exec error", func(t *testing.T) {
		if got := gitMessage(errors.New("  boom  ")); got != "boom" {
			t.Fatalf("gitMessage = %q, want %q", got, "boom")
		}
	})
	t.Run("gitShowOracle refuses a path outside the repository", func(t *testing.T) {
		root := t.TempDir()
		outside := filepath.Join(t.TempDir(), "Thing.oracle.md")
		if _, err := gitShowOracle(root, "HEAD", outside); err == nil {
			t.Fatal("a path outside the repository must fail loudly")
		}
	})
	t.Run("gitRootOf refuses a directory outside a repository", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git not available")
		}
		// a temp dir is not inside the machinery checkout
		if _, err := gitRootOf(t.TempDir()); err == nil {
			t.Skip("the temp directory happens to sit inside a repository")
		}
	})
}

func TestOracleGitIgnoresInjectedRepositoryEnvironment(t *testing.T) {
	repoA := t.TempDir()
	repoB := t.TempDir()
	writeMachine(t, repoA, "Alpha.machine.json", validMachineA)
	writeMachine(t, repoB, "Beta.machine.json", validMachineB)
	gitInit(t, repoA)
	gitInit(t, repoB)
	commit := func(root string) string {
		t.Helper()
		body, err := testgit.Run(t.Context(), root, "rev-parse", "HEAD")
		if err != nil {
			t.Fatalf("resolve %s HEAD: %v: %s", root, err, body)
		}
		return strings.TrimSpace(string(body))
	}
	wantA := commit(repoA)
	wantB := commit(repoB)
	if wantA == wantB {
		t.Fatal("two repository fixture commits unexpectedly match")
	}
	t.Setenv("GIT_DIR", filepath.Join(repoB, ".git"))
	t.Setenv("GIT_WORK_TREE", repoB)
	t.Setenv("GIT_COMMON_DIR", filepath.Join(repoB, ".git"))
	t.Setenv("GIT_REPLACE_REF_BASE", "refs/hostile-replacements/")
	got, err := gitResolveCommit(repoA, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if got != wantA {
		t.Fatalf("injected repository environment redirected repo A to %s; want %s", got, wantA)
	}
}
