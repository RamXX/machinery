package checker

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeRegistry drops a registry file into a temp dir and returns its path.
func writeRegistry(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "checkers.local.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const validRegistry = `checkers:
  privacy:
    run: ["privacy-checker", "run", "--projection", "{projection}", "--config", "{config}", "--out", "{out}"]
    verify: ["privacy-checker", "verify", "--trace", "{out}"]
    timeout: "45s"
  invariants:
    run: ["inv-checker", "{manifest}", "{design}", "{out}"]
`

func TestLoadRegistryValidAndResolve(t *testing.T) {
	reg, err := LoadRegistry(writeRegistry(t, validRegistry))
	if err != nil {
		t.Fatal(err)
	}

	got := reg.IDs()
	if len(got) != 2 || got[0] != "invariants" || got[1] != "privacy" {
		t.Fatalf("IDs not sorted/complete: %v", got)
	}

	privacy, ok := reg.Resolve("privacy")
	if !ok {
		t.Fatal("privacy should resolve")
	}
	if privacy.Timeout != 45*time.Second {
		t.Fatalf("explicit timeout not parsed: %v", privacy.Timeout)
	}
	if len(privacy.Verify) == 0 {
		t.Fatal("privacy verify command not carried")
	}

	if _, ok := reg.Resolve("nonexistent"); ok {
		t.Fatal("a missing id must not resolve")
	}
}

func TestLoadRegistryDefaultTimeout(t *testing.T) {
	reg, err := LoadRegistry(writeRegistry(t, validRegistry))
	if err != nil {
		t.Fatal(err)
	}
	inv, ok := reg.Resolve("invariants")
	if !ok {
		t.Fatal("invariants should resolve")
	}
	if inv.Timeout != DefaultCheckerTimeout {
		t.Fatalf("absent timeout should default to %v, got %v", DefaultCheckerTimeout, inv.Timeout)
	}
	if len(inv.Verify) != 0 {
		t.Fatalf("absent verify should be empty, got %v", inv.Verify)
	}
}

func TestTokenSubstitutionReplacesAll(t *testing.T) {
	args := []string{"tool", "--projection", "{projection}", "--config", "{config}", "--manifest", "{manifest}", "--out", "{out}", "--design", "{design}"}
	tok := Tokens{
		Projection: "/d/proj.json",
		Config:     "/tmp/cfg.json",
		Manifest:   "/d/checkers/x.checker.yaml",
		Out:        "/tmp/out.json",
		Design:     "/d",
	}
	got := tok.Substitute(args)
	want := []string{"tool", "--projection", "/d/proj.json", "--config", "/tmp/cfg.json", "--manifest", "/d/checkers/x.checker.yaml", "--out", "/tmp/out.json", "--design", "/d"}
	if len(got) != len(want) {
		t.Fatalf("length mismatch: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg %d = %q, want %q", i, got[i], want[i])
		}
	}
	// The source args must be untouched (Substitute returns a fresh slice).
	if args[2] != "{projection}" {
		t.Fatal("Substitute mutated the input args")
	}
}

func TestTokenSubstitutionMultipleTokensPerArg(t *testing.T) {
	tok := Tokens{Projection: "P", Out: "O"}
	got := tok.Substitute([]string{"{projection}:{out}"})
	if got[0] != "P:O" {
		t.Fatalf("multiple tokens in one arg not both replaced: %q", got[0])
	}
}

func TestLoadRegistryMalformedYAMLIsError(t *testing.T) {
	if _, err := LoadRegistry(writeRegistry(t, "checkers: [this is: not, a: map\n")); err == nil {
		t.Fatal("malformed YAML must be an error")
	}
}

func TestLoadRegistryEmptyRunIsError(t *testing.T) {
	body := `checkers:
  broken:
    verify: ["something"]
`
	if _, err := LoadRegistry(writeRegistry(t, body)); err == nil {
		t.Fatal("an entry with an empty run command must be an error")
	}
}

func TestLoadRegistryInvalidTimeoutIsError(t *testing.T) {
	body := `checkers:
  bad:
    run: ["x"]
    timeout: "not-a-duration"
`
	if _, err := LoadRegistry(writeRegistry(t, body)); err == nil {
		t.Fatal("an invalid timeout string must be an error")
	}
}

func TestLoadRegistryMissingFileIsError(t *testing.T) {
	if _, err := LoadRegistry(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("a missing registry file must be an error")
	}
}
