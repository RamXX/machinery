package gates

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/version"
)

// These are advice-selection fixtures; executable generation is covered below.
func TestVersionSkewRegenerationOnlySafeGenerators(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files []string
		want  []string
	}{
		{"ratchet-only", []string{RatchetFile}, nil},
		{"oracle", []string{"machines/Widget.machine.json"}, []string{"oracle DESIGN/machines"}},
		{"alloy", []string{"formal/domain.relational.yaml"}, []string{"alloy DESIGN"}},
		{"semantics", []string{"formal/Widget.semantics.yaml"}, []string{"verify-formal --gen-only DESIGN"}},
		{"composition", []string{"formal/System.composition.yaml"}, []string{"verify-formal --gen-only DESIGN"}},
		{"pack", []string{"decomposition.yaml"}, []string{"pack generate DESIGN"}},
		{"all-with-ratchet", []string{RatchetFile, "machines/Widget.machine.json", "formal/domain.relational.yaml", "formal/Widget.semantics.yaml", "formal/System.composition.yaml", "decomposition.yaml"}, []string{"oracle DESIGN/machines", "alloy DESIGN", "verify-formal --gen-only DESIGN", "pack generate DESIGN"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			for _, file := range tc.files {
				mustWrite(t, filepath.Join(design, file), "{}\n")
			}
			var want []string
			for _, command := range tc.want {
				want = append(want, "machinery "+strings.ReplaceAll(command, "DESIGN", design))
			}
			g := NewGate("version skew")
			g.recordStamp("<!-- machinery-version: v0.0.1 -->")
			note := VersionSkewNote(design, []*Gate{g})
			if got := regenerationAdvice(note); !reflect.DeepEqual(got, want) {
				t.Errorf("routine regeneration advice = %v; want exactly %v", got, want)
			}
			if again := VersionSkewNote(design, []*Gate{g}); again != note {
				t.Error("unchanged inputs produced nondeterministic advice")
			}
		})
	}
}

func regenerationAdvice(output string) []string {
	var commands []string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "  machinery ") {
			commands = append(commands, strings.TrimSpace(line))
		}
	}
	return commands
}

func TestRegenRatchetRealCLI(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "machinery")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, "./cmd/machinery")
	build.Dir = filepath.Join("..", "..")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build real CLI: %v\n%s", err, out)
	}
	run := func(t *testing.T, args ...string) (string, error) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, args...)
		out, err := cmd.CombinedOutput()
		if ctx.Err() != nil {
			t.Fatalf("CLI deadline: %v\n%s", ctx.Err(), out)
		}
		return string(out), err
	}
	mustRun := func(t *testing.T, args ...string) string {
		t.Helper()
		out, err := run(t, args...)
		if err != nil {
			t.Fatalf("CLI %v failed: %v\n%s", args, err, out)
		}
		return out
	}
	read := func(t *testing.T, path string) []byte {
		t.Helper()
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	fixture := func(t *testing.T, ratcheted bool) (string, string) {
		t.Helper()
		rules := "  allow: [\"alpha -> beta\"]"
		if ratcheted {
			rules = "  deny: [\"alpha -> beta\"]\n  baseline: [\"alpha -> beta\"]"
		}
		design, impl := writeFixture(t, fixtureOpts{rules: rules})
		// The oracle fixture is valid and fresh except for its version stamp.
		oracleDesign := writeStampFixture(t, nil)
		copyGateTree(t, filepath.Join(oracleDesign, "machines"), filepath.Join(design, "machines"))
		mustRun(t, "oracle", filepath.Join(design, "machines"))
		path := filepath.Join(design, "machines", "Widget.oracle.md")
		body := string(read(t, path))
		if !strings.Contains(body, version.MarkdownStamp()) {
			t.Fatal("CLI did not generate the current oracle stamp")
		}
		mustWrite(t, path, strings.Replace(body, version.MarkdownStamp(), "<!-- machinery-version: v0.0.1 -->", 1))
		if ratcheted {
			mustRun(t, "baseline", design, "--impl", impl, "--date", "2026-01-15")
			ratchet, err := LoadRatchet(design)
			if err != nil || ratchet == nil || !reflect.DeepEqual(ratchet.Edges["alpha -> beta"], []string{"alpha/a.go"}) {
				t.Fatalf("initial CLI baseline did not record the existing offender: %+v, %v", ratchet, err)
			}
		}
		mustRun(t, "check", design, "--impl", impl, "--gate", "g3,g4")
		return design, impl
	}
	follow := func(t *testing.T, output, design, impl string) {
		t.Helper()
		commands := regenerationAdvice(output)
		want := []string{"machinery oracle " + design + "/machines"}
		if !reflect.DeepEqual(commands, want) {
			t.Errorf("expected exactly applicable safe generators %v; got %v", want, commands)
		}
		// Execute EVERY advised command, including the unsafe baseline on RED.
		// Replace its documented placeholder as a real user would; no shell runs.
		for _, command := range commands {
			args := strings.Fields(strings.ReplaceAll(command, "<dir>", impl))
			mustRun(t, args[1:]...)
		}
		body := string(read(t, filepath.Join(design, "machines", "Widget.oracle.md")))
		if !strings.Contains(body, version.MarkdownStamp()) || strings.Contains(body, "v0.0.1") {
			t.Fatal("following advice did not actually regenerate the old oracle stamp")
		}
	}
	newOffender := func(t *testing.T, impl string) {
		mustWrite(t, filepath.Join(impl, "alpha", "b.go"), "package alpha\nimport _ \"example.com/m/beta\"\n")
	}
	requireRatchetFailure := func(t *testing.T, design, impl string) string {
		t.Helper()
		out, err := run(t, "check", design, "--impl", impl, "--gate", "g3,g4")
		if err == nil || !strings.Contains(out, "ratchet: baselined edge alpha -> beta grew by 1 new offender file") || !strings.Contains(out, "alpha/b.go") {
			t.Errorf("G4 must reject the genuine new offender: err=%v\n%s", err, out)
		}
		return out
	}
	t.Run("no-debt-version-skew-control", func(t *testing.T) {
		design, impl := fixture(t, false)
		out := mustRun(t, "check", design, "--impl", impl, "--gate", "g3,g4")
		follow(t, out, design, impl)
		out = mustRun(t, "check", design, "--impl", impl, "--gate", "g3,g4")
		if strings.Contains(out, "regenerate on upgrade") {
			t.Fatal("regeneration left version skew")
		}
		if _, err := os.Stat(filepath.Join(design, RatchetFile)); !os.IsNotExist(err) {
			t.Fatalf("routine regeneration unexpectedly created a ratchet: %v", err)
		}
	})
	t.Run("new-offender-survives-all-advice", func(t *testing.T) {
		design, impl := fixture(t, true)
		before := read(t, filepath.Join(design, RatchetFile))
		newOffender(t, impl)
		out := requireRatchetFailure(t, design, impl)
		follow(t, out, design, impl)
		if after := read(t, filepath.Join(design, RatchetFile)); !bytes.Equal(before, after) {
			t.Errorf("routine regeneration changed accepted debt:\nbefore %s\nafter %s", before, after)
		}
		requireRatchetFailure(t, design, impl)
	})
	t.Run("explicit-baseline-reviews-debt-change", func(t *testing.T) {
		design, impl := fixture(t, true)
		newOffender(t, impl)
		requireRatchetFailure(t, design, impl)
		out := mustRun(t, "baseline", design, "--impl", impl)
		lower := strings.ToLower(out)
		if !strings.Contains(lower, "review") || !strings.Contains(lower, "ratchet") || !(strings.Contains(lower, "debt") || strings.Contains(lower, "offender")) {
			t.Errorf("explicit debt expansion needs ratchet/debt review guidance even for an already-baselined edge:\n%s", out)
		}
		ratchet, err := LoadRatchet(design)
		if err != nil || ratchet == nil || !reflect.DeepEqual(ratchet.Edges["alpha -> beta"], []string{"alpha/a.go", "alpha/b.go"}) {
			t.Fatalf("explicit baseline must remain available: %+v, %v", ratchet, err)
		}
		if strings.Contains(string(read(t, filepath.Join(design, RatchetFile))), "machinery-version") {
			t.Error("a version stamp must not substitute for preventing debt expansion")
		}
		mustRun(t, "check", design, "--impl", impl, "--gate", "g4")
	})
}
