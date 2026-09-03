package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/oracle"
	"github.com/RamXX/machinery/internal/version"
)

// repoRoot resolves to the repo root (the test runs from internal/gates).
func repoRoot() string { return "../.." }

func TestCheckC4CleanOnExamples(t *testing.T) {
	for _, ex := range []string{"go-crm", "fulfillment", "portfolio-engine"} {
		t.Run(ex, func(t *testing.T) {
			design := filepath.Join(repoRoot(), "examples", ex, "design")
			g := CheckC4(design)
			if len(g.Errs) != 0 {
				t.Errorf("%s: G2 errors: %v", ex, g.Errs)
			}
			if g.Counts["boundaries"] == 0 {
				t.Errorf("%s: no boundaries parsed", ex)
			}
		})
	}
}

func TestCheckMachinesCleanOnExamples(t *testing.T) {
	for _, ex := range []string{"go-crm", "fulfillment", "portfolio-engine"} {
		t.Run(ex, func(t *testing.T) {
			design := filepath.Join(repoRoot(), "examples", ex, "design")
			g := CheckMachines(design)
			if len(g.Errs) != 0 {
				t.Errorf("%s: G3 errors: %v", ex, g.Errs)
			}
			if len(g.Drift) != 0 {
				t.Errorf("%s: G3 drift: %v", ex, g.Drift)
			}
			if g.Counts["machines"] == 0 {
				t.Errorf("%s: no machines parsed", ex)
			}
		})
	}
}

func TestCheckTraceabilityCleanOnExamples(t *testing.T) {
	for _, ex := range []string{"go-crm", "fulfillment", "portfolio-engine"} {
		t.Run(ex, func(t *testing.T) {
			design := filepath.Join(repoRoot(), "examples", ex, "design")
			g := CheckTraceability(design)
			if len(g.Errs) != 0 {
				t.Errorf("%s: Gx errors: %v", ex, g.Errs)
			}
		})
	}
}

func TestCheckImportsCleanOnGoCRM(t *testing.T) {
	design := filepath.Join(repoRoot(), "examples", "go-crm", "design")
	impl := filepath.Join(repoRoot(), "examples", "go-crm", "impl")
	g := CheckImports(design, impl)
	if len(g.Errs) != 0 {
		t.Errorf("G4 errors: %v", g.Errs)
	}
	if g.Counts["go files checked"] == 0 {
		t.Error("no go files checked")
	}
}

// G2 concatenates EVERY event-contract table (header with producer, consumer,
// delivery): the first-match locator silently ignored later tables (PACK-1).
func TestG2EventContractTablesConcatenate(t *testing.T) {
	design := t.TempDir()
	dsl := "workspace \"W\" \"sys\" {\n  model {\n    sys = softwareSystem \"S\" \"sys\" {\n" +
		"      app = component \"App\" \"logic\" \"Go\"\n" +
		"      bus = container \"Bus\" \"events\" \"NATS\" \"Queue\"\n    }\n  }\n}\n"
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: app\n    code: [\"app/**\"]\n```\n\n" +
		"## Events (orders)\n\nSource: fixture emit sites.\n\n| producer | consumer | payload | delivery |\n|---|---|---|---|\n" +
		"| app | bus | OrderPlaced | at-least-once |\n| app | bus | OrderPaid | at-least-once |\n\n" +
		"## Events (payments)\n\nSource: fixture emit sites.\n\n| producer | consumer | payload | delivery |\n|---|---|---|---|\n" +
		"| bus | app | PaymentSettled | at-least-once |\n" + nfrStub
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	g := CheckC4(design)
	if g.Counts["event contracts"] != 3 {
		t.Fatalf("event contracts = %d, want 3 (rows of BOTH tables): %+v", g.Counts["event contracts"], g.Counts)
	}
	if hasErr(g, "no event-contract table") {
		t.Fatalf("the tables exist and must be found: %v", g.Errs)
	}
}

// nfrStub is the minimal NFR record a coherent G2 fixture carries, so
// fixtures built to exercise other checks report only their own findings.
const nfrStub = "\n## NFR record\n\n- security: out of scope (fixture)\n- capacity: out of scope (fixture)\n- observability: out of scope (fixture)\n"

// coveringInterfaceTable renders one interface-contract row per concrete allow
// edge in a fixture's rules block, so a fixture built to exercise the
// dependency GRAPH is a coherent design and reports graph findings only.
func coveringInterfaceTable(rules string) string {
	var rows []string
	key := ""
	for _, line := range strings.Split(rules, "\n") {
		if t := strings.TrimRight(line, " "); strings.HasPrefix(t, "  ") && strings.HasSuffix(t, ":") &&
			!strings.HasPrefix(strings.TrimSpace(t), "-") {
			key = strings.TrimSuffix(strings.TrimSpace(t), ":")
			continue
		}
		t := strings.TrimSpace(line)
		if key != "allow" || !strings.HasPrefix(t, "- ") {
			continue
		}
		edge := strings.Trim(strings.TrimPrefix(t, "- "), `"`)
		if strings.Contains(edge, "*") {
			continue // a wildcard rule names no concrete pair, so it needs no row
		}
		rows = append(rows, "| `"+edge+"` | fixture shape | none | n/a |")
	}
	if len(rows) == 0 {
		return ""
	}
	return "\n## Interface contracts\n\n| edge | shape | errors | idempotency |\n|---|---|---|---|\n" +
		strings.Join(rows, "\n") + "\n"
}

// c4GraphFixture builds a minimal design whose contract declares the given
// boundary ids and dependency_rules block, and runs G2 over it.
func c4GraphFixture(t *testing.T, ids []string, rules string) *Gate {
	t.Helper()
	design := t.TempDir()
	var comps, bounds strings.Builder
	for _, id := range ids {
		el := id[strings.LastIndex(id, ".")+1:]
		comps.WriteString("      " + el + " = component \"" + el + "\" \"logic\" \"Go\"\n")
		bounds.WriteString("  - id: " + id + "\n    code: [\"" + el + "/**\"]\n")
	}
	dsl := "workspace \"W\" \"sys\" {\n  model {\n    sys = softwareSystem \"S\" \"sys\" {\n" +
		comps.String() + "    }\n  }\n}\n"
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		bounds.String() + rules + "```\n" + coveringInterfaceTable(rules) + nfrStub
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckC4(design)
}

// G2 must reject a cyclic allow graph: a declared cycle destroys modularity
// and cannot be an accident. Cycles closing only through baseline edges are
// the ratchet's tolerated debt: a warn, never a gate failure.
func TestG2AllowGraphAcyclicity(t *testing.T) {
	cases := []struct {
		name  string
		ids   []string
		rules string
		errs  []string // required substrings, one per expected cycle finding
		warns []string // required substrings, one per expected warn
	}{
		{
			name: "clean DAG has no findings",
			ids:  []string{"p.a", "p.b", "p.c"},
			rules: "dependency_rules:\n  allow:\n" +
				"    - p.a -> p.b\n    - p.b -> p.c\n    - p.a -> p.c\n",
		},
		{
			name: "two-node cycle is one error naming both members",
			ids:  []string{"p.a", "p.b"},
			rules: "dependency_rules:\n  allow:\n" +
				"    - p.a -> p.b\n    - p.b -> p.a\n",
			errs: []string{"dependency cycle among p.a, p.b in allow rules (p.a -> p.b -> p.a)"},
		},
		{
			name:  "self-loop is an error",
			ids:   []string{"p.a"},
			rules: "dependency_rules:\n  allow:\n    - p.a -> p.a\n",
			errs:  []string{"allow rule p.a -> p.a is a self-cycle"},
		},
		{
			name: "three-node cycle is one SCC, not three findings",
			ids:  []string{"p.a", "p.b", "p.c"},
			rules: "dependency_rules:\n  allow:\n" +
				"    - p.a -> p.b\n    - p.b -> p.c\n    - p.c -> p.a\n",
			errs: []string{"dependency cycle among p.a, p.b, p.c in allow rules (p.a -> p.b -> p.c -> p.a)"},
		},
		{
			name: "two independent cycles are two findings, deterministically ordered",
			ids:  []string{"p.a", "p.b", "p.c", "p.d"},
			rules: "dependency_rules:\n  allow:\n" +
				"    - p.b -> p.a\n    - p.a -> p.b\n    - p.d -> p.c\n    - p.c -> p.d\n",
			errs: []string{
				"dependency cycle among p.a, p.b in allow rules (p.a -> p.b -> p.a)",
				"dependency cycle among p.c, p.d in allow rules (p.c -> p.d -> p.c)",
			},
		},
		{
			name: "cycle closing only via baseline is a warn, not an error",
			ids:  []string{"p.a", "p.b"},
			rules: "dependency_rules:\n  allow:\n    - p.a -> p.b\n" +
				"  deny:\n    - \"p.b -> p.a\"\n  baseline:\n    - p.b -> p.a\n",
			warns: []string{"dependency cycle among p.a, p.b closes only through baseline edges (p.a -> p.b -> p.a)"},
		},
		{
			name: "wildcard edges fabricate no cycle",
			ids:  []string{"platform.a", "platform.b"},
			rules: "dependency_rules:\n  allow:\n" +
				"    - platform.a -> platform.b\n    - \"platform.* -> platform.*\"\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := c4GraphFixture(t, tc.ids, tc.rules)
			if len(g.Errs) != len(tc.errs) {
				t.Fatalf("errs = %v, want %d finding(s) %v", g.Errs, len(tc.errs), tc.errs)
			}
			for i, want := range tc.errs {
				if !strings.Contains(g.Errs[i], want) {
					t.Errorf("errs[%d] = %q, want substring %q", i, g.Errs[i], want)
				}
			}
			if len(g.Warns) != len(tc.warns) {
				t.Fatalf("warns = %v, want %d warn(s) %v", g.Warns, len(tc.warns), tc.warns)
			}
			for i, want := range tc.warns {
				if !strings.Contains(g.Warns[i], want) {
					t.Errorf("warns[%d] = %q, want substring %q", i, g.Warns[i], want)
				}
			}
		})
	}
}

// The transitive-pairs count is the allow graph's reachable ordered pairs:
// a reported fact next to the declared-edge count, never a finding.
func TestG2TransitivePairsCount(t *testing.T) {
	g := c4GraphFixture(t, []string{"p.a", "p.b", "p.c", "p.d"},
		"dependency_rules:\n  allow:\n"+
			"    - p.a -> p.b\n    - p.b -> p.c\n    - p.a -> p.d\n")
	if len(g.Errs) != 0 {
		t.Fatalf("unexpected errors: %v", g.Errs)
	}
	// a->b, a->c (via b), a->d, b->c
	if got := g.Counts["transitive pairs"]; got != 4 {
		t.Errorf("transitive pairs = %d, want 4: %+v", got, g.Counts)
	}
	if g.Counts["allow rules"] != 3 {
		t.Errorf("allow rules = %d, want 3", g.Counts["allow rules"])
	}
}

// The Gx placement waiver must sit in the component or machine-placement
// cell and carry a non-empty reason: '(no machine:)' with an empty reason, or
// the token buried elsewhere in the row, waives nothing (GATE-5).
func TestGxPlacementWaiverColumnAndReason(t *testing.T) {
	writeGxFixture := func(t *testing.T, placementRow string) *Gate {
		t.Helper()
		design := t.TempDir()
		mustWrite(t, filepath.Join(design, "domain.modelith.yaml"),
			"kind: modelith\nversion: 1\nentities:\n  Widget:\n    actions:\n      - name: publish\n    invariants:\n      - id: widget-owned\n")
		mustWrite(t, filepath.Join(design, "machines", "Ops.machine.json"),
			`{"id":"ops","_role":"operational","initial":"A","states":{"A":{}}}`)
		mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"),
			"# A\n\n## Placement\n\n| component | machine placement | persistence | concurrency |\n|---|---|---|---|\n"+
				placementRow+"\n")
		return CheckTraceability(design)
	}
	t.Run("empty reason is not a waiver", func(t *testing.T) {
		g := writeGxFixture(t, "| `Ghost` | wherever (no machine:) | none | n/a |")
		if !hasErr(g, "has no machine and no '(no machine: <reason>)' waiver") {
			t.Fatalf("an empty waiver reason must not waive: %v", g.Errs)
		}
	})
	t.Run("token outside the machine columns is not a waiver", func(t *testing.T) {
		g := writeGxFixture(t, "| `Ghost` | somewhere | none | racy (no machine: honest note in the wrong column) |")
		if !hasErr(g, "has no machine and no '(no machine: <reason>)' waiver") {
			t.Fatalf("the token buried in another cell must not waive: %v", g.Errs)
		}
	})
	t.Run("placement-cell waiver with a reason waives", func(t *testing.T) {
		g := writeGxFixture(t, "| `Ghost` | none (no machine: pure transform) | none | n/a |")
		if hasErr(g, "has no machine and no") {
			t.Fatalf("a reasoned placement-cell waiver must waive: %v", g.Errs)
		}
		if g.Counts["placement rows waived"] != 1 {
			t.Errorf("placement rows waived = %d, want 1: %+v", g.Counts["placement rows waived"], g.Counts)
		}
	})
	t.Run("component-cell waiver with a reason waives", func(t *testing.T) {
		g := writeGxFixture(t, "| `Ghost` (no machine: reference data, upserted) | none | none | n/a |")
		if hasErr(g, "has no machine and no") {
			t.Fatalf("a reasoned component-cell waiver must waive (the portfolio-engine shape): %v", g.Errs)
		}
		if g.Counts["placement rows waived"] != 1 {
			t.Errorf("placement rows waived = %d, want 1: %+v", g.Counts["placement rows waived"], g.Counts)
		}
	})
}

// --- G2 interface-contract completeness (the allow list is closed) ---

// ifaceFixture builds a design whose contract declares the given boundary ids
// and rules, and whose ARCHITECTURE.md carries the given interface-contract
// table rows (no table at all when rows is "").
func ifaceFixture(t *testing.T, ids []string, rules, rows string) *Gate {
	t.Helper()
	design := t.TempDir()
	var comps, bounds strings.Builder
	for _, id := range ids {
		el := id[strings.LastIndex(id, ".")+1:]
		comps.WriteString("      " + el + " = component \"" + el + "\" \"logic\" \"Go\"\n")
		bounds.WriteString("  - id: " + id + "\n    code: [\"" + el + "/**\"]\n")
	}
	dsl := "workspace \"W\" \"sys\" {\n  model {\n    sys = softwareSystem \"S\" \"sys\" {\n" +
		comps.String() + "    }\n  }\n}\n"
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		bounds.String() + rules + "```\n"
	if rows != "" {
		arch += "\n## Interface contracts\n\n| edge | shape | errors | idempotency |\n|---|---|---|---|\n" + rows + "\n"
	}
	arch += nfrStub
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckC4(design)
}

const twoEdgeRules = "dependency_rules:\n  allow:\n    - p.a -> p.b\n    - p.a -> p.c\n"

func TestG2InterfaceContractCompleteness(t *testing.T) {
	t.Run("a row per allow edge passes", func(t *testing.T) {
		g := ifaceFixture(t, []string{"p.a", "p.b", "p.c"}, twoEdgeRules,
			"| `p.a -> p.b` | Store interface | ErrNotFound | reads safe to retry |\n"+
				"| `p.a -> p.c` | Clock.Now() | none | pure |")
		if len(g.Errs) != 0 {
			t.Fatalf("a complete interface table must pass: %v", g.Errs)
		}
		if g.Counts["edges contracted"] != 2 {
			t.Errorf("edges contracted = %d, want 2: %+v", g.Counts["edges contracted"], g.Counts)
		}
	})
	t.Run("an allow edge with no row is an error naming the edge", func(t *testing.T) {
		g := ifaceFixture(t, []string{"p.a", "p.b", "p.c"}, twoEdgeRules,
			"| `p.a -> p.b` | Store interface | ErrNotFound | reads safe to retry |")
		if !hasErr(g, "allow edge `p.a -> p.c` has no interface-contract row") {
			t.Fatalf("an uncontracted allow edge must fail, named: %v", g.Errs)
		}
		if hasErr(g, "allow edge `p.a -> p.b`") {
			t.Fatalf("the contracted edge must be credited: %v", g.Errs)
		}
	})
	t.Run("one row may cover several edges", func(t *testing.T) {
		g := ifaceFixture(t, []string{"p.a", "p.b", "p.c"}, twoEdgeRules,
			"| `p.a -> p.b`, `p.a -> p.c` | the same Store interface | ErrNotFound | reads safe to retry |")
		if len(g.Errs) != 0 {
			t.Fatalf("a multi-edge row must cover every pair it names: %v", g.Errs)
		}
		if g.Counts["edges contracted"] != 2 {
			t.Errorf("edges contracted = %d, want 2: %+v", g.Counts["edges contracted"], g.Counts)
		}
	})
	t.Run("a reasoned '(no contract:)' waiver passes", func(t *testing.T) {
		g := ifaceFixture(t, []string{"p.a", "p.b", "p.c"}, twoEdgeRules,
			"| `p.a -> p.b` | Store interface | ErrNotFound | reads safe to retry |\n"+
				"| `p.a -> p.c` (no contract: authored in the child design) | n/a | n/a | n/a |")
		if len(g.Errs) != 0 {
			t.Fatalf("a reasoned waiver must waive: %v", g.Errs)
		}
		if g.Counts["edges waived"] != 1 || g.Counts["edges contracted"] != 1 {
			t.Errorf("counts = %+v, want 1 contracted and 1 waived", g.Counts)
		}
	})
	t.Run("a waiver with no reason is an error", func(t *testing.T) {
		g := ifaceFixture(t, []string{"p.a", "p.b", "p.c"}, twoEdgeRules,
			"| `p.a -> p.b` | Store interface | ErrNotFound | reads safe to retry |\n"+
				"| `p.a -> p.c` (no contract:) | n/a | n/a | n/a |")
		if !hasErr(g, "interface-contract waiver for edge `p.a -> p.c` names no reason") {
			t.Fatalf("an empty waiver reason must not waive: %v", g.Errs)
		}
	})
	t.Run("an empty column is an error", func(t *testing.T) {
		g := ifaceFixture(t, []string{"p.a", "p.b", "p.c"}, twoEdgeRules,
			"| `p.a -> p.b` | Store interface |  | reads safe to retry |\n"+
				"| `p.a -> p.c` | Clock.Now() | none | pure |")
		if !hasErr(g, "leaves errors empty") {
			t.Fatalf("an unanswered column must fail: %v", g.Errs)
		}
	})
	t.Run("no interface table at all is one error, not silence", func(t *testing.T) {
		g := ifaceFixture(t, []string{"p.a", "p.b", "p.c"}, twoEdgeRules, "")
		if !hasErr(g, "has no interface-contract table") {
			t.Fatalf("a missing table must fail: %v", g.Errs)
		}
	})
}

// A table headed "crossing" is an interface-contract table too. Designs
// authored against the pre-table wording call the column that, and telling
// such a design it "has no interface-contract table" while it visibly has one
// is a false diagnosis: the rows are then read and their cells reported.
func TestG2InterfaceContractCrossingHeaderSynonym(t *testing.T) {
	design := t.TempDir()
	dsl := "workspace \"W\" \"sys\" {\n  model {\n    sys = softwareSystem \"S\" \"sys\" {\n" +
		"      a = component \"a\" \"logic\" \"Go\"\n      b = component \"b\" \"logic\" \"Go\"\n    }\n  }\n}\n"
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: p.a\n    code: [\"a/**\"]\n  - id: p.b\n    code: [\"b/**\"]\n" +
		"dependency_rules:\n  allow:\n    - p.a -> p.b\n```\n\n" +
		"## Interface contracts\n\n| crossing | shape | errors | idempotency |\n|---|---|---|---|\n" +
		"| `p.a -> p.b` | Store interface | ErrNotFound | reads safe to retry |\n" + nfrStub
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	g := CheckC4(design)
	if hasErr(g, "has no interface-contract table") {
		t.Fatalf("a table headed 'crossing' is an interface-contract table: %v", g.Errs)
	}
	if g.Counts["edges contracted"] != 1 {
		t.Errorf("edges contracted = %d, want 1: %+v", g.Counts["edges contracted"], g.Counts)
	}
}

// Drift: the table may only describe edges the contract allows. A row for a
// denied, undeclared, or merely baselined edge documents an interface the
// architecture does not have.
func TestG2InterfaceContractDrift(t *testing.T) {
	t.Run("a row for a non-allowed declared edge is drift", func(t *testing.T) {
		g := ifaceFixture(t, []string{"p.a", "p.b", "p.c"}, twoEdgeRules,
			"| `p.a -> p.b` | Store | ErrNotFound | retry |\n"+
				"| `p.a -> p.c` | Clock | none | pure |\n"+
				"| `p.b -> p.c` | a shape for an edge nobody allowed | none | pure |")
		if !hasErr(g, "interface-contract row names edge `p.b -> p.c`, which no allow rule declares") {
			t.Fatalf("a contract for an unallowed edge must be drift: %v", g.Errs)
		}
	})
	t.Run("a row naming an undeclared boundary says so", func(t *testing.T) {
		g := ifaceFixture(t, []string{"p.a", "p.b", "p.c"}, twoEdgeRules,
			"| `p.a -> p.b` | Store | ErrNotFound | retry |\n"+
				"| `p.a -> p.c` | Clock | none | pure |\n"+
				"| `p.a -> p.ghost` | a shape for a boundary that does not exist | none | pure |")
		if !hasErr(g, "(`p.ghost` is not a declared boundary or external)") {
			t.Fatalf("an undeclared side must be named: %v", g.Errs)
		}
	})
	t.Run("a baselined edge is not an allow edge", func(t *testing.T) {
		g := ifaceFixture(t, []string{"p.a", "p.b", "p.c"},
			"dependency_rules:\n  allow:\n    - p.a -> p.b\n    - p.a -> p.c\n"+
				"  deny:\n    - p.b -> p.c\n  baseline:\n    - p.b -> p.c\n",
			"| `p.a -> p.b` | Store | ErrNotFound | retry |\n"+
				"| `p.a -> p.c` | Clock | none | pure |\n"+
				"| `p.b -> p.c` | documenting tolerated debt as if it were designed | none | pure |")
		if !hasErr(g, "interface-contract row names edge `p.b -> p.c`, which no allow rule declares") {
			t.Fatalf("a baselined edge is tolerated debt, not a designed interface: %v", g.Errs)
		}
	})
}

// Whole-edge matching: an edge is credited only by a row naming that exact
// pair. A longer id must never satisfy a shorter one, in either position.
func TestG2InterfaceContractWholeEdge(t *testing.T) {
	g := ifaceFixture(t, []string{"p.a", "p.ab", "p.b", "p.bb"},
		"dependency_rules:\n  allow:\n    - p.a -> p.b\n    - p.ab -> p.bb\n",
		"| `p.ab -> p.bb` | the longer pair only | none | pure |")
	if !hasErr(g, "allow edge `p.a -> p.b` has no interface-contract row") {
		t.Fatalf("`p.a -> p.b` must not be credited by the `p.ab -> p.bb` row: %v", g.Errs)
	}
	if g.Counts["edges contracted"] != 1 {
		t.Errorf("edges contracted = %d, want 1: %+v", g.Counts["edges contracted"], g.Counts)
	}
}

// The edge cell is read whole or not at all: a chain or a stray word is a
// loud format error, never a half-understood row.
func TestG2InterfaceContractEdgeCellGrammar(t *testing.T) {
	t.Run("a chain is malformed", func(t *testing.T) {
		g := ifaceFixture(t, []string{"p.a", "p.b", "p.c"}, twoEdgeRules,
			"| `p.a -> p.b -> p.c` | a chain, not a pair | none | pure |")
		if !hasErr(g, "is not a 'from -> to' pair") {
			t.Fatalf("a chain must be reported: %v", g.Errs)
		}
	})
	t.Run("a wildcard is refused", func(t *testing.T) {
		g := ifaceFixture(t, []string{"p.a", "p.b", "p.c"}, twoEdgeRules,
			"| `p.a -> p.*` | one shape for a whole class | none | pure |")
		if !hasErr(g, "uses a wildcard; a contract is a concrete claim") {
			t.Fatalf("a wildcard row must be refused: %v", g.Errs)
		}
	})
	t.Run("a row with no edge at all is reported", func(t *testing.T) {
		g := ifaceFixture(t, []string{"p.a", "p.b", "p.c"}, twoEdgeRules,
			"| the store boundary | prose in the edge column | none | pure |")
		if !hasErr(g, "is not a 'from -> to' pair") {
			t.Fatalf("a row naming no pair must be reported: %v", g.Errs)
		}
	})
	t.Run("a wildcard allow rule carries no obligation", func(t *testing.T) {
		g := ifaceFixture(t, []string{"p.a", "p.b", "p.c"},
			"dependency_rules:\n  allow:\n    - p.a -> p.b\n    - \"p.a -> p.*\"\n",
			"| `p.a -> p.b` | Store | ErrNotFound | retry |")
		if len(g.Errs) != 0 {
			t.Fatalf("a wildcard allow rule names no concrete pair, so it demands no row: %v", g.Errs)
		}
	})
}

// --- Gx placement completeness (the entity list is closed, so it is checked) ---

// gxCompletenessModel declares two entities whose names overlap (Order is a
// substring of OrderLine), one enum with two values, and one scenario: the
// completeness check must demand rows for the two ENTITIES only, and must not
// credit `Order` from a row that names `OrderLine`.
const gxCompletenessModel = `kind: modelith
version: 1
enums:
  OrderKind:
    values:
      - name: Retail
      - name: Wholesale
entities:
  Order:
    attributes:
      - name: kind
        type: OrderKind
    actions:
      - name: place
    invariants:
      - id: order-owned
  OrderLine:
    actions:
      - name: add
scenarios:
  - name: Retail flow
`

// writeGxCompletenessFixture builds a design whose ARCHITECTURE.md carries the
// given placement rows (or no table at all when rows is "").
func writeGxCompletenessFixture(t *testing.T, rows string) *Gate {
	t.Helper()
	design := t.TempDir()
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), gxCompletenessModel)
	mustWrite(t, filepath.Join(design, "machines", "Ops.machine.json"),
		`{"id":"ops","_role":"operational","initial":"A","states":{"A":{}}}`)
	arch := "# A\n"
	if rows != "" {
		arch += "\n## Placement\n\n| component | machine placement | persistence | concurrency |\n|---|---|---|---|\n" + rows + "\n"
	}
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckTraceability(design)
}

// Gx's BUILD.md scans run fence-masked, matching Gb: a fenced example Mode
// line or heading is documentation and must not satisfy a template check.
func TestGxBuildScansAreFenceMasked(t *testing.T) {
	design := t.TempDir()
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), gxCompletenessModel)
	mustWrite(t, filepath.Join(design, "machines", "Ops.machine.json"),
		`{"id":"ops","_role":"operational","initial":"A","states":{"A":{}}}`)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), "# A\n")
	mustWrite(t, filepath.Join(design, "BUILD.md"),
		"# B\n```\nMode: full\n## Toolchain\n```\n")
	g := CheckTraceability(design)
	if !hasErr(g, "declares no mode") {
		t.Fatalf("a fenced Mode line must not count as a declaration: %v", g.Errs)
	}
	if !hasErr(g, "no Toolchain heading") {
		t.Fatalf("a fenced Toolchain heading must not count: %v", g.Errs)
	}
}

// Block-form `tags "..."` statements inside an element body are a legal
// Structurizr spelling; reading only the inline argument once let a
// block-tagged Database element dodge the infra obligations G2 keys off tags.
func TestDSLElementsBlockTags(t *testing.T) {
	els := dslElementsOf(`api = container "API" "svc" "Go" {
    tags "Database"
    orders = component "Orders" "c" "Go" {
        tags "Queue" "Internal"
    }
}
web = container "Web (uses {braces} in prose)" "ui" "TS"
`)
	if !els["api"].Tags["Database"] {
		t.Fatalf("block-form tag missing on api: %+v", els["api"])
	}
	if !els["orders"].Tags["Queue"] || !els["orders"].Tags["Internal"] {
		t.Fatalf("nested block-form tags missing on orders: %+v", els["orders"])
	}
	if els["web"].Tags["Database"] || els["web"].Tags["Queue"] {
		t.Fatalf("tags leaked past the declaring block: %+v", els["web"])
	}
}

func TestGxPlacementCompleteness(t *testing.T) {
	t.Run("every entity in a row passes", func(t *testing.T) {
		g := writeGxCompletenessFixture(t,
			"| `Order` (no machine: CRUD record) | none | db row | single writer |\n"+
				"| `OrderLine` (no machine: rows owned by their order) | none | db rows | with the order |")
		if hasErr(g, "appears in no persistence-and-placement row") {
			t.Fatalf("a complete table must pass: %v", g.Errs)
		}
		if g.Counts["entities placed"] != 2 {
			t.Errorf("entities placed = %d, want 2: %+v", g.Counts["entities placed"], g.Counts)
		}
	})
	t.Run("an entity with no row is an error naming it", func(t *testing.T) {
		g := writeGxCompletenessFixture(t,
			"| `OrderLine` (no machine: rows owned by their order) | none | db rows | with the order |")
		if !hasErr(g, "entity `Order` appears in no persistence-and-placement row") {
			t.Fatalf("a model entity outside the table must fail, named: %v", g.Errs)
		}
		if hasErr(g, "entity `OrderLine` appears in no") {
			t.Fatalf("the row's own entity must be credited: %v", g.Errs)
		}
		// whole-token: `Order` must not be credited by the `OrderLine` row
		if g.Counts["entities placed"] != 1 {
			t.Errorf("entities placed = %d, want 1 (OrderLine only): %+v", g.Counts["entities placed"], g.Counts)
		}
	})
	t.Run("a reasoned '(not placed:)' waiver passes and demands no machine", func(t *testing.T) {
		g := writeGxCompletenessFixture(t,
			"| `OrderLine` (no machine: rows owned by their order) | none | db rows | with the order |\n"+
				"| `Order` (not placed: a value object folded into the line row) | n/a | n/a | n/a |")
		if hasErr(g, "appears in no persistence-and-placement row") {
			t.Fatalf("a reasoned placement waiver must waive: %v", g.Errs)
		}
		if hasErr(g, "has no machine") {
			t.Fatalf("a not-placed entity has no placement and so no machine to demand: %v", g.Errs)
		}
		if g.Counts["entities placement-waived"] != 1 {
			t.Errorf("entities placement-waived = %d, want 1: %+v", g.Counts["entities placement-waived"], g.Counts)
		}
	})
	t.Run("a waiver with no reason is an error", func(t *testing.T) {
		g := writeGxCompletenessFixture(t,
			"| `OrderLine` (no machine: rows owned by their order) | none | db rows | with the order |\n"+
				"| `Order` (not placed:) | n/a | n/a | n/a |")
		if !hasErr(g, "placement waiver for `Order` names no reason") {
			t.Fatalf("an empty waiver reason must not waive: %v", g.Errs)
		}
	})
	t.Run("enum and scenario names are never demanded", func(t *testing.T) {
		g := writeGxCompletenessFixture(t,
			"| `Order` (no machine: CRUD record) | none | db row | single writer |\n"+
				"| `OrderLine` (no machine: rows owned by their order) | none | db rows | with the order |")
		for _, name := range []string{"OrderKind", "Retail", "Wholesale", "Retail flow"} {
			if hasErr(g, name) {
				t.Errorf("%s is not an entity; no placement row may be demanded for it: %v", name, g.Errs)
			}
		}
	})
	t.Run("no placement table at all is an error", func(t *testing.T) {
		g := writeGxCompletenessFixture(t, "")
		if !hasErr(g, "has no persistence-and-placement table") {
			t.Fatalf("a missing table must fail: deleting it would otherwise waive every entity: %v", g.Errs)
		}
	})
}

// A name mentioned only inside a row's parenthetical annotation is prose about
// another component, never a placement of its own.
func TestGxPlacementSubjectsIgnoreAnnotations(t *testing.T) {
	got := placementSubjects("`Holding` (no machine: rows written with their `Portfolio`)")
	if len(got) != 1 || got[0] != "Holding" {
		t.Errorf("placementSubjects = %v, want [Holding]", got)
	}
}

func TestTokenInWholeToken(t *testing.T) {
	if !tokenIn("inv-1", "foo inv-1 bar") {
		t.Error("inv-1 should match standalone")
	}
	if tokenIn("inv-1", "foo inv-12 bar") {
		t.Error("inv-1 must not match inside inv-12")
	}
}

func TestGateEmitFormatting(t *testing.T) {
	g := NewGate("Test")
	g.Count("widgets")
	g.Count("widgets")
	g.Count("gadgets", 3)
	if g.Counts["widgets"] != 2 || g.Counts["gadgets"] != 3 {
		t.Errorf("counts wrong: %+v", g.Counts)
	}
	// order must be insertion order
	if g.countOrder[0] != "widgets" || g.countOrder[1] != "gadgets" {
		t.Errorf("count order wrong: %v", g.countOrder)
	}
}

// --- P-F10: generator version stamps and the freshness diff ---

const stampWidgetMachine = `{"id":"widget","initial":"Draft",
  "_delays":{"persistTimeout":"3000 ms - test bound"},
  "states":{
  "Draft":{"on":{"publish":[{"target":"persisting","guard":"guardCanPublish","actions":"setPending"},{"actions":"recordDenied"}]}},
  "Published":{"type":"final"},
  "persisting":{"invoke":{"src":"saveWidget","onDone":{"target":"Published","actions":"commit"},"onError":{"target":"Draft","actions":"recordError"}},"after":{"persistTimeout":{"target":"Draft","actions":"recordTimeout"}}}}}`

// writeStampFixture builds a design with one machine and a committed oracle
// whose text the caller can transform first (identity when nil).
func writeStampFixture(t *testing.T, mutate func(string) string) string {
	t.Helper()
	design := t.TempDir()
	mdir := filepath.Join(design, "machines")
	if err := os.MkdirAll(mdir, 0o755); err != nil {
		t.Fatal(err)
	}
	mp := filepath.Join(mdir, "Widget.machine.json")
	if err := os.WriteFile(mp, []byte(stampWidgetMachine), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := ir.LoadMachineJSON(mp)
	if err != nil {
		t.Fatal(err)
	}
	text := oracle.Render(m, mp)
	if mutate != nil {
		text = mutate(text)
	}
	if err := os.WriteFile(filepath.Join(mdir, "Widget.oracle.md"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	matrix := "| name | kind | contract |\n|---|---|---|\n" +
		"| `guardCanPublish` | guard | decides publish |\n" +
		"| `setPending` | action | records pending |\n" +
		"| `recordDenied` | action | records refusal |\n" +
		"| `commit` | action | commits |\n" +
		"| `recordError` | action | records error |\n" +
		"| `recordTimeout` | action | records timeout |\n" +
		"| `saveWidget` | actor | persists widget |\n"
	if err := os.WriteFile(filepath.Join(mdir, "Widget.matrix.md"), []byte(matrix), 0o644); err != nil {
		t.Fatal(err)
	}
	return design
}

// A committed oracle stamped by a DIFFERENT machinery version, content
// otherwise identical, is fresh: version-only skew is never DRIFT.
func TestOracleVersionOnlySkewIsNotDrift(t *testing.T) {
	design := writeStampFixture(t, func(text string) string {
		return strings.Replace(text, version.MarkdownStamp(), "<!-- machinery-version: v0.0.1 -->", 1)
	})
	g := CheckMachines(design)
	if len(g.Drift) != 0 || len(g.Errs) != 0 {
		t.Fatalf("version-only skew reported as drift: errs=%v drift=%v", g.Errs, g.Drift)
	}
	if g.Counts["oracles fresh"] != 1 {
		t.Errorf("oracle not counted fresh: %v", g.Counts)
	}
	note := VersionSkewNote(design, []*Gate{g})
	if !strings.Contains(note, "v0.0.1") || !strings.Contains(note, version.Version) {
		t.Errorf("skew note = %q, want both versions named", note)
	}
	if !strings.HasPrefix(note, "note: artifacts generated by machinery ") || !strings.Contains(note, "; regenerate on upgrade") {
		t.Errorf("skew note format drifted: %q", note)
	}
	// the note names the command for the family this design actually carries,
	// and only that one: no alloy, formal, pack, or baseline line applies
	if !strings.Contains(note, "machinery oracle "+design+"/machines") {
		t.Errorf("skew note omits the oracle regeneration command: %q", note)
	}
	for _, unwanted := range []string{"machinery alloy", "machinery verify-formal", "machinery pack generate", "machinery baseline"} {
		if strings.Contains(note, unwanted) {
			t.Errorf("skew note names %q for a design with no such artifacts: %q", unwanted, note)
		}
	}
	if !strings.HasSuffix(note, "the regeneration lands as its own dedicated stamp-only commit, never mixed with a design change") {
		t.Errorf("skew note drops the stamp-only-commit reminder: %q", note)
	}
}

// The regeneration commands follow the artifact families the design carries:
// every family present contributes exactly one command, absent ones none.
func TestVersionSkewNoteNamesEveryApplicableCommand(t *testing.T) {
	design := writeStampFixture(t, func(text string) string {
		return strings.Replace(text, version.MarkdownStamp(), "<!-- machinery-version: v0.0.1 -->", 1)
	})
	formal := filepath.Join(design, "formal")
	if err := os.MkdirAll(formal, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(formal, "domain.relational.yaml"): "entities: []\n",
		filepath.Join(formal, "Widget.semantics.yaml"):  "machine: Widget\n",
		filepath.Join(design, "decomposition.yaml"):     "packs: []\n",
		filepath.Join(design, RatchetFile):              "{}\n",
	}
	for path, body := range files {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	g := CheckMachines(design)
	note := VersionSkewNote(design, []*Gate{g})
	for _, want := range []string{
		"machinery oracle " + design + "/machines",
		"machinery alloy " + design,
		"machinery verify-formal --gen-only " + design,
		"machinery pack generate " + design,
		"machinery baseline " + design + " --impl <dir>",
	} {
		if !strings.Contains(note, want) {
			t.Errorf("skew note omits %q: %q", want, note)
		}
	}
}

// A generated (stamped) formal/*.tla is itself a regeneration obligation,
// even with no semantics or composition source beside it; a hand-written one
// is not.
func TestVersionSkewNoteFormalCommandFollowsGeneratedTLA(t *testing.T) {
	design := writeStampFixture(t, func(text string) string {
		return strings.Replace(text, version.MarkdownStamp(), "<!-- machinery-version: v0.0.1 -->", 1)
	})
	formal := filepath.Join(design, "formal")
	if err := os.MkdirAll(formal, 0o755); err != nil {
		t.Fatal(err)
	}
	hand := filepath.Join(formal, "Hand.tla")
	if err := os.WriteFile(hand, []byte("---- MODULE Hand ----\n====\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := CheckMachines(design)
	if note := VersionSkewNote(design, []*Gate{g}); strings.Contains(note, "verify-formal") {
		t.Errorf("hand-written .tla must not imply a regeneration: %q", note)
	}
	if err := os.WriteFile(hand, []byte(version.StampTLAModule("---- MODULE Hand ----\n====\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	g = CheckMachines(design)
	if note := VersionSkewNote(design, []*Gate{g}); !strings.Contains(note, "machinery verify-formal --gen-only "+design) {
		t.Errorf("generated .tla did not produce the formal regeneration command: %q", note)
	}
}

// Content drift is still DRIFT, stamp or no stamp.
func TestOracleContentDriftStillDrift(t *testing.T) {
	design := writeStampFixture(t, func(text string) string {
		return strings.Replace(text, "| Draft |", "| Drafted |", 1)
	})
	g := CheckMachines(design)
	if len(g.Drift) == 0 {
		t.Fatal("content drift not reported")
	}
}

// A committed oracle with NO stamp (pre-stamp artifact) is fresh when the
// content matches, and produces no skew note.
func TestOracleMissingStampIsFreshAndSilent(t *testing.T) {
	design := writeStampFixture(t, func(text string) string {
		return strings.Replace(text, version.MarkdownStamp()+"\n", "", 1)
	})
	g := CheckMachines(design)
	if len(g.Drift) != 0 || len(g.Errs) != 0 {
		t.Fatalf("pre-stamp oracle reported stale: errs=%v drift=%v", g.Errs, g.Drift)
	}
	if note := VersionSkewNote(design, []*Gate{g}); note != "" {
		t.Errorf("missing stamp must not be skew: %q", note)
	}
}

// A committed oracle stamped with the RUNNING version produces no note.
func TestOracleCurrentStampIsSilent(t *testing.T) {
	design := writeStampFixture(t, nil)
	g := CheckMachines(design)
	if len(g.Drift) != 0 || len(g.Errs) != 0 {
		t.Fatalf("identical oracle reported stale: errs=%v drift=%v", g.Errs, g.Drift)
	}
	if note := VersionSkewNote(design, []*Gate{g}); note != "" {
		t.Errorf("current stamp must not be skew: %q", note)
	}
}

func TestG3RefusesTLAUngenerable(t *testing.T) {
	// S11: a machine the TLA generator refuses must fail G3 with the
	// generator's own message, not surface later at verify-formal.
	d := t.TempDir()
	md := filepath.Join(d, "machines")
	if err := os.MkdirAll(md, 0755); err != nil {
		t.Fatal(err)
	}
	two := `{"id":"m","initial":"Working","_delays":{"A":"1s","B":"2s"},"states":{
		"Working":{"on":{"go":{"target":"workRetry"}}},
		"workRetry":{"always":[{"target":"Failed","guard":"guardRetriesExhausted"}],
			"after":{"A":{"target":"Working"},"B":{"target":"Working"}}},
		"Failed":{"type":"final"}}}`
	if err := os.WriteFile(filepath.Join(md, "Two.machine.json"), []byte(two), 0644); err != nil {
		t.Fatal(err)
	}
	g := CheckMachines(d)
	found := false
	for _, e := range g.Errs {
		if strings.Contains(e, "retry state") && strings.Contains(e, "after entries") {
			found = true
		}
	}
	if !found {
		t.Fatalf("G3 did not surface the generator refusal; errs: %v", g.Errs)
	}
}

func TestG3CountsTLAGenerable(t *testing.T) {
	d := t.TempDir()
	md := filepath.Join(d, "machines")
	if err := os.MkdirAll(md, 0755); err != nil {
		t.Fatal(err)
	}
	ok := `{"id":"m","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: refused when canAdvance is false"}},
		"Won":{"type":"final"}}}`
	if err := os.WriteFile(filepath.Join(md, "Ok.machine.json"), []byte(ok), 0644); err != nil {
		t.Fatal(err)
	}
	g := CheckMachines(d)
	for _, e := range g.Errs {
		if strings.Contains(e, "tla") {
			t.Fatalf("valid machine refused: %v", g.Errs)
		}
	}
	if g.Counts["tla-generable"] != 1 {
		t.Fatalf("tla-generable count: %v", g.Counts)
	}
}
