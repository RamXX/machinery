package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/gates"
)

// checkpointFindings runs the checkpoint (non-landing) selection the CLI
// default suite would run and returns the blocking findings of the named
// gate prefix. It proves a fixture is red at the checkpoint, so a green stop
// below is the landing selection at work, not a vacuous fixture.
func checkpointFindings(t *testing.T, design, titlePrefix string) []string {
	t.Helper()
	_, run, _, err := gates.SelectRunAndNote(design, "", "", gates.RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, g := range run {
		if strings.HasPrefix(g.Title, titlePrefix) {
			out = append(out, g.Errs...)
			out = append(out, g.Drift...)
		}
	}
	return out
}

func strictCRMRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyTree(t, crmDesign, filepath.Join(root, "design"))
	writeFile(t, filepath.Join(root, ConfigName), `{"strict": true}`)
	return root
}

func decodeStop(t *testing.T, out string) stopOut {
	t.Helper()
	var got stopOut
	if out == "" {
		return got
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	return got
}

// B5: the stop hook runs the landing selection. Between checkpoints the main
// line is expected to be stale for attestation and acceptance records, so a
// checkpoint-only finding (Gv, Ga) never blocks a turn end, in strict mode
// too. DRIFT still blocks.
func TestStrictStopIgnoresCheckpointOnlyFindings(t *testing.T) {
	cases := []struct {
		name, gate string
		mutate     func(t *testing.T, design string)
	}{
		{"rejected acceptance for a closed milestone", "Ga", func(t *testing.T, design string) {
			p := filepath.Join(design, "acceptance", "M0.yaml")
			raw, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, p, strings.Replace(string(raw), "verdict: ACCEPTED", "verdict: REJECTED", 1))
		}},
		{"stale attestation row", "Gv", func(t *testing.T, design string) {
			p := filepath.Join(design, "ARCHITECTURE.md")
			raw, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, p, string(raw)+"\n")
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := strictCRMRoot(t)
			design := filepath.Join(root, "design")
			c.mutate(t, design)
			if len(checkpointFindings(t, design, c.gate)) == 0 {
				t.Fatalf("fixture is not red at the checkpoint for %s", c.gate)
			}
			sid := "s-landing-" + c.gate
			t.Cleanup(func() { clearState(root, sid) })
			if err := appendState(root, sid, "design"); err != nil {
				t.Fatal(err)
			}
			got := decodeStop(t, runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"}))
			if got.Decision == "block" {
				t.Fatalf("a checkpoint-only %s finding must not block a strict stop: %s", c.gate, got.Reason)
			}
			if d, i := readState(root, sid); d || i {
				t.Fatal("a landing-green stop must clear the obligation")
			}

			// DRIFT still blocks in the same tree
			oracle := filepath.Join(design, "machines", "Deal.oracle.md")
			raw, err := os.ReadFile(oracle)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, oracle, string(raw)+"\ntampered\n")
			if err := appendState(root, sid, "design"); err != nil {
				t.Fatal(err)
			}
			got = decodeStop(t, runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"}))
			if got.Decision != "block" || !strings.Contains(got.Reason, "DRIFT") {
				t.Fatalf("DRIFT must still block the landing stop: %+v", got)
			}
			if strings.Contains(got.Reason, "== Ga-") || strings.Contains(got.Reason, "== Gv-") {
				t.Fatalf("the landing stop must not run the checkpoint-only gates: %s", got.Reason)
			}
		})
	}
}
