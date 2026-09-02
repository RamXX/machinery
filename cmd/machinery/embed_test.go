package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// embedDesign writes a design with one source table and one embedded copy
// that has drifted from it.
func embedDesign(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	src := "| name | note |\n|---|---|\n| `alpha` | current |\n| `beta` | current |\n"
	shard := "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n" +
		"| name | note |\n|---|---|\n| `alpha` | STALE |\n| `gamma` | renamed away |\n"
	if err := os.WriteFile(filepath.Join(d, "ARCHITECTURE.md"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "SHARD.md"), []byte(shard), 0644); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestEmbedRefreshRewritesAndReports(t *testing.T) {
	outB, _, codes := withCapturedIO(t)
	d := embedDesign(t)
	if err := embedRefreshRun(d, false); err != nil {
		t.Fatal(err)
	}
	if len(*codes) != 0 {
		t.Fatalf("exit codes %v, want none", *codes)
	}
	out := outB.String()
	for _, want := range []string{"1 rows re-copied", "no source row for 1 row(s): gamma", "1 markers"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout %q does not carry %q", out, want)
		}
	}
	body, err := os.ReadFile(filepath.Join(d, "SHARD.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "STALE") {
		t.Fatalf("the drifted row survived:\n%s", body)
	}
	if !strings.Contains(string(body), "| `gamma` | renamed away |") {
		t.Fatalf("the unmatched row was deleted:\n%s", body)
	}
}

func TestEmbedRefreshDryRunLeavesTheFile(t *testing.T) {
	outB, _, _ := withCapturedIO(t)
	d := embedDesign(t)
	before, err := os.ReadFile(filepath.Join(d, "SHARD.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := embedRefreshRun(d, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(outB.String(), "would be re-copied") {
		t.Fatalf("stdout %q does not read as a dry run", outB.String())
	}
	after, err := os.ReadFile(filepath.Join(d, "SHARD.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("the dry run wrote the file")
	}
}

func TestEmbedRefreshFailsLoudlyWithNoMarkers(t *testing.T) {
	outB, _, codes := withCapturedIO(t)
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "ARCHITECTURE.md"), []byte("# nothing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := embedRefreshRun(d, false); err == nil {
		t.Fatal("a design with no markers must fail loudly")
	}
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(outB.String(), "no machinery:embed markers") {
		t.Fatalf("stdout %q", outB.String())
	}
}
