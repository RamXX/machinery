package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/gates"
)

func runAttest(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	out, errB, codes := withCapturedIO(t)
	cmd := newAttestCmd()
	cmd.SetOut(out)
	cmd.SetErr(errB)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	code := 0
	if len(*codes) > 0 {
		code = (*codes)[0]
	}
	return out.String(), errB.String(), code
}

// The helper prints exactly the value the gate demands, for exactly the file
// named. An attestor who pastes this line's hash cannot produce a malformed
// or wrong-algorithm digest, which is the whole reason the helper exists.
func TestAttestPrintsTheHashTheGateDemands(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ARCHITECTURE.md")
	if err := os.WriteFile(path, []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	want, err := gates.ContentHash(path)
	if err != nil {
		t.Fatal(err)
	}
	out, _, code := runAttest(t, path)
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if out != want+"  "+path+"\n" {
		t.Fatalf("attest printed %q, want %q", out, want+"  "+path+"\n")
	}
}

// Several paths in one run, so re-attesting a multi-artifact claim is one
// command.
func TestAttestHashesEveryPath(t *testing.T) {
	dir := t.TempDir()
	var paths []string
	for _, name := range []string{"a.md", "b.md"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	out, _, code := runAttest(t, paths...)
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if lines := strings.Count(out, "\n"); lines != 2 {
		t.Fatalf("want one line per path, got %d:\n%s", lines, out)
	}
}

// --claims prints the gate's own vocabulary, so the docs and the attestor
// read the closed set from the binary instead of a transcription.
func TestAttestClaimsPrintsTheVocabulary(t *testing.T) {
	out, _, code := runAttest(t, "--claims")
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	got := strings.Split(strings.TrimSpace(out), "\n")
	want := gates.AttestationClaimIDs()
	if len(got) != len(want) {
		t.Fatalf("printed %d claim ids, want %d:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("claim %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// An unreadable path fails loudly with a non-zero exit; silently printing
// nothing would let a re-attestation land with a stale hash still in place.
func TestAttestFailsOnAnAbsentPath(t *testing.T) {
	_, errB, code := runAttest(t, filepath.Join(t.TempDir(), "absent.md"))
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errB, "machinery attest:") {
		t.Fatalf("stderr %q, want a named failure", errB)
	}
}

// No paths and no --claims is a usage error, not a silent success.
func TestAttestWithoutArgumentsFails(t *testing.T) {
	_, errB, code := runAttest(t)
	if code != 1 || !strings.Contains(errB, "name at least one path") {
		t.Fatalf("exit %d, stderr %q", code, errB)
	}
}

// `--gate gv` runs the gate end to end through the CLI: green on a design
// whose attestation binds, blocking once the covered artifact moves.
func TestCheckGateGvRunsAndCatchesStaleness(t *testing.T) {
	design := t.TempDir()
	arch := filepath.Join(design, "ARCHITECTURE.md")
	if err := os.WriteFile(arch, []byte("# A\n\n## Architecture Contract\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, err := gates.ContentHash(arch)
	if err != nil {
		t.Fatal(err)
	}
	evidence := "attestation_version: 1\nattestations:\n"
	for _, id := range gates.AttestationClaimIDs() {
		if !strings.HasPrefix(id, "g2.") {
			continue
		}
		evidence += "  - claim: " + id + "\n    attestor: conductor + owner sign-off\n" +
			"    date: 2026-08-30\n    covers:\n      - {path: ARCHITECTURE.md, hash: " + hash + "}\n"
	}
	evPath := filepath.Join(design, gates.AttestationsFileName)
	if err := os.WriteFile(evPath, []byte(evidence), 0o644); err != nil {
		t.Fatal(err)
	}

	run := func() (string, int) {
		t.Helper()
		out, _, codes := withCapturedIO(t)
		cmd := newCheckCmd()
		cmd.SetArgs([]string{design, "--gate", "gv"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		code := 0
		if len(*codes) > 0 {
			code = (*codes)[0]
		}
		return out.String(), code
	}

	out, code := run()
	if code != 0 {
		t.Fatalf("a bound attestation must pass, exit %d:\n%s", code, out)
	}
	if !strings.Contains(out, "Gv-attest") || !strings.Contains(out, "6 attested claims") {
		t.Fatalf("output must name the gate and what it checked:\n%s", out)
	}

	if err := os.WriteFile(arch, []byte("# A\n\n## Architecture Contract\n\nA new sentence.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code = run()
	if code == 0 {
		t.Fatalf("an edited covered artifact must block:\n%s", out)
	}
	if !strings.Contains(out, "is STALE") {
		t.Fatalf("the finding must name the staleness:\n%s", out)
	}
}
