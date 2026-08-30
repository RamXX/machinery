package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const surfaceTargetModelYAML = `
kind: DomainModel
version: v1
title: Target
enums:
  ThingStatus:
    values:
      - {name: Open, definition: open}
      - {name: Closed, definition: closed}
entities:
  Thing:
    definition: target thing
    attributes:
      - {name: id, type: string}
      - {name: status, type: ThingStatus}
    actions:
      - {name: create, actor: User}
      - {name: close, actor: User}
  Audit:
    definition: audit record
    attributes:
      - {name: id, type: string}
`

const surfaceWorkspaceDSL = `
workspace {
  model {
    sys = softwareSystem "System" {
      api   = container "API Service" "Serves things." "Go"
      store = container "Store" "Persists things." "Postgres" "Database"
    }
  }
}
`

const surfaceLedger = `
surface_version: 1
system: legacy thing service, a REST API over Postgres
classes:
  routes:
    source: legacy router.go route table
    items:
      - {name: "POST /things", disposition: covered, via: action, target: Thing.create}
      - {name: "GET /things", disposition: covered, via: entity, target: Thing}
      - {name: "GET /admin/metrics", disposition: dropped, rationale: superseded by the target observability stack}
  commands:
    none: the legacy system is a service; it has no CLI surface
  tables:
    source: legacy schema catalog
    items:
      - {name: things, disposition: covered, via: component, target: store}
      - {name: audit_log, disposition: covered, via: machine, target: Thing}
      - {name: sessions, disposition: deferred, rationale: session storage is designed in the next iteration}
  jobs:
    none: no scheduled or background work in the legacy service
  events:
    none: no queues or topics in a single-process service
  integrations:
    none: no outbound calls to external services
`

func writeSurfaceFixture(t *testing.T, ledger string) string {
	t.Helper()
	design := t.TempDir()
	for _, dir := range []string{"legacy", "machines"} {
		if err := os.MkdirAll(filepath.Join(design, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"domain.modelith.yaml":        surfaceTargetModelYAML,
		"workspace.dsl":               surfaceWorkspaceDSL,
		"machines/Thing.machine.json": `{"id": "Thing", "initial": "Open", "states": {"Open": {}, "Closed": {}}}`,
		SurfaceLedgerName:             ledger,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(design, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return design
}

func TestCheckSurfaceClean(t *testing.T) {
	design := writeSurfaceFixture(t, surfaceLedger)
	g := CheckSurface(design)
	if len(g.Errs) != 0 || len(g.Drift) != 0 {
		t.Fatalf("Gs not clean: errs=%v drift=%v", g.Errs, g.Drift)
	}
	want := map[string]int{
		"routes": 3, "tables": 3, "covered": 4, "dropped": 1, "deferred": 1,
		"waived classes": 4, "surface items": 6,
	}
	for count, n := range want {
		if g.Counts[count] != n {
			t.Errorf("Gs counted %s=%d, want %d: %+v", count, g.Counts[count], n, g.Counts)
		}
	}
	sel, err := Select(design, "", "")
	if err != nil || !sel.Run["gs"] {
		t.Fatalf("default gate selection omitted gs: sel=%+v err=%v", sel, err)
	}
	found := false
	for _, gate := range RunSelected(design, "", sel, RunOptions{}) {
		found = found || strings.Contains(gate.Title, "Gs-surface")
	}
	if !found {
		t.Error("RunSelected skipped an authored surface ledger")
	}
}

// as_of is the optional revision/date anchor of the ledger (P-F3): accepted
// by the strict-key schema, surfaced on the checked line, and validated as a
// non-empty string when present.
func TestCheckSurfaceAsOfAnchor(t *testing.T) {
	t.Run("accepted and surfaced", func(t *testing.T) {
		design := writeSurfaceFixture(t, "as_of: legacy@a1b2c3\n"+surfaceLedger)
		g := CheckSurface(design)
		if len(g.Errs) != 0 || len(g.Warns) != 0 {
			t.Fatalf("as_of is a legal root key: errs=%v warns=%v", g.Errs, g.Warns)
		}
		if !strings.Contains(strings.Join(g.checkedExtra, ", "), "as_of legacy@a1b2c3") {
			t.Errorf("as_of must appear on the checked line: %v", g.checkedExtra)
		}
	})
	t.Run("absent stays silent", func(t *testing.T) {
		g := CheckSurface(writeSurfaceFixture(t, surfaceLedger))
		if strings.Contains(strings.Join(g.checkedExtra, ", "), "as_of") {
			t.Errorf("no as_of, no checked-line segment: %v", g.checkedExtra)
		}
	})
	t.Run("non-string errors", func(t *testing.T) {
		g := CheckSurface(writeSurfaceFixture(t, "as_of: 20260630\n"+surfaceLedger))
		if !strings.Contains(strings.Join(g.Errs, "\n"), "as_of must be a non-empty string") {
			t.Fatalf("a non-string as_of must error: %v", g.Errs)
		}
	})
}

// The anchor's SHAPE. docs/surface-ledger.md documents as_of as "the legacy
// commit or date the surface was enumerated against; non-empty string": the
// meaning is pinned, the format is not, so a shape the docs never demanded is
// a WARN and never blocks. What it catches is the anchor that quietly stopped
// being one, which is the whole reason the field is printed on every run.
func TestCheckSurfaceAsOfShape(t *testing.T) {
	accepted := []string{
		"2026-07-22", // ISO date
		"a1b2c3d",    // the shortest binding revision
		"9f3c1a2b7d4e5f60718293a4b5c6d7e8f9012345", // a full sha
		"v1.4.2",          // a release tag
		"release/2026-06", // a branch
		"legacy@a1b2c3",   // system@revision
		"svn:41207",       // a non-git revision
	}
	for _, asOf := range accepted {
		t.Run("accepted "+asOf, func(t *testing.T) {
			g := CheckSurface(writeSurfaceFixture(t, "as_of: "+asOf+"\n"+surfaceLedger))
			if len(g.Errs) != 0 || len(g.Warns) != 0 {
				t.Fatalf("%q is an anchor a reviewer can act on: errs=%v warns=%v", asOf, g.Errs, g.Warns)
			}
		})
	}

	// prose where an anchor belongs: the value the go-crm example carried,
	// which reads as documentation and compares against nothing
	t.Run("prose warns and stays surfaced", func(t *testing.T) {
		const prose = "2026-07-22, enumerated from the committed legacy model and the migration.yaml inventory"
		g := CheckSurface(writeSurfaceFixture(t, "as_of: "+prose+"\n"+surfaceLedger))
		if len(g.Errs) != 0 {
			t.Fatalf("the shape rule must never block: %v", g.Errs)
		}
		joined := strings.Join(g.Warns, "\n")
		if !strings.Contains(joined, "is none of the shapes an anchor takes") {
			t.Fatalf("prose in as_of must warn: %v", g.Warns)
		}
		for _, want := range []string{"an ISO date (YYYY-MM-DD)", "a VCS revision (7 to 40 hex characters)", "a tag-like token"} {
			if !strings.Contains(joined, want) {
				t.Errorf("the warning must name the expected shape %q: %v", want, g.Warns)
			}
		}
		if !strings.Contains(strings.Join(g.checkedExtra, ", "), "as_of "+prose) {
			t.Errorf("a warned anchor is still surfaced verbatim: %v", g.checkedExtra)
		}
	})

	t.Run("date shape that is not a day", func(t *testing.T) {
		g := CheckSurface(writeSurfaceFixture(t, "as_of: 2026-02-31\n"+surfaceLedger))
		if !strings.Contains(strings.Join(g.Warns, "\n"), "names no real calendar day") {
			t.Fatalf("an impossible date must warn: %v", g.Warns)
		}
		if len(g.Errs) != 0 {
			t.Errorf("the shape rule must never block: %v", g.Errs)
		}
	})

	t.Run("a hex string too short to name a commit", func(t *testing.T) {
		// 'a1b2c' is under git's own 7-character abbreviation floor, but it is
		// still one token, so it lands in the tag lane rather than warning:
		// the gate does not invent a rule the documentation never stated
		g := CheckSurface(writeSurfaceFixture(t, "as_of: a1b2c\n"+surfaceLedger))
		if len(g.Warns) != 0 {
			t.Errorf("a single token is an anchor shape: %v", g.Warns)
		}
	})
}

func TestCheckSurfaceMutations(t *testing.T) {
	allWaived := `
surface_version: 1
system: legacy thing service
classes:
  routes: {none: nothing}
  commands: {none: nothing}
  tables: {none: nothing}
  jobs: {none: nothing}
  events: {none: nothing}
  integrations: {none: nothing}
`
	cases := []struct {
		name   string
		ledger string
		mutate func(t *testing.T, design string)
		want   string
	}{
		{"unknown root key", surfaceLedger + "bogus: true\n", nil, "unsupported key"},
		{"bad version", strings.Replace(surfaceLedger, "surface_version: 1", "surface_version: 2", 1), nil, "surface_version must be the integer 1"},
		{"missing system", strings.Replace(surfaceLedger, "system: legacy thing service, a REST API over Postgres", "system: \"\"", 1), nil, "system is required"},
		{"missing class", strings.Replace(surfaceLedger, "  commands:\n    none: the legacy system is a service; it has no CLI surface\n", "", 1), nil, "classes.commands is missing"},
		{"unknown class", surfaceLedger + "  webhooks:\n    none: not a real class\n", nil, "not a surface class"},
		{"waiver mixed with inventory", strings.Replace(surfaceLedger, "  jobs:\n    none:", "  jobs:\n    source: cron -l\n    none:", 1), nil, "mixes a waiver with an inventory"},
		{"empty waiver reason", strings.Replace(surfaceLedger, "none: no queues or topics in a single-process service", "none: \"\"", 1), nil, "needs a reason"},
		{"missing source", strings.Replace(surfaceLedger, "    source: legacy schema catalog\n", "", 1), nil, "classes.tables.source is required"},
		{"duplicate item", strings.Replace(surfaceLedger, "{name: \"GET /things\"", "{name: \"POST /things\"", 1), nil, "lists \"POST /things\" twice"},
		{"covered without target", strings.Replace(surfaceLedger, "disposition: covered, via: entity, target: Thing}", "disposition: covered}", 1), nil, "names no via/target design element"},
		{"bad via", strings.Replace(surfaceLedger, "via: entity", "via: table", 1), nil, "via must be entity, action, component, or machine"},
		{"unknown entity", strings.Replace(surfaceLedger, "via: entity, target: Thing}", "via: entity, target: Widget}", 1), nil, "unknown target entity"},
		{"unknown action", strings.Replace(surfaceLedger, "target: Thing.create", "target: Thing.destroy", 1), nil, "unknown action"},
		{"unknown component", strings.Replace(surfaceLedger, "via: component, target: store", "via: component, target: warehouse", 1), nil, "not a workspace.dsl element"},
		{"missing machine", strings.Replace(surfaceLedger, "via: machine, target: Thing", "via: machine, target: Audit", 1), nil, "machines/Audit.machine.json, which does not exist"},
		{"dropped without rationale", strings.Replace(surfaceLedger, "disposition: dropped, rationale: superseded by the target observability stack", "disposition: dropped", 1), nil, "without a rationale"},
		{"deferred with target", strings.Replace(surfaceLedger, "disposition: deferred, rationale: session storage is designed in the next iteration", "disposition: deferred, rationale: later, via: entity, target: Thing", 1), nil, "a capability with a target is covered"},
		{"bad disposition", strings.Replace(surfaceLedger, "disposition: dropped,", "disposition: ignored,", 1), nil, "disposition must be covered, dropped, or deferred"},
		{"only waivers", allWaived, nil, "nothing checked"},
		{"missing target model", surfaceLedger, func(t *testing.T, design string) {
			if err := os.Remove(filepath.Join(design, "domain.modelith.yaml")); err != nil {
				t.Fatal(err)
			}
		}, "covered bindings resolve against the Phase 1 target model"},
		{"component before phase 2", surfaceLedger, func(t *testing.T, design string) {
			if err := os.Remove(filepath.Join(design, "workspace.dsl")); err != nil {
				t.Fatal(err)
			}
		}, "workspace.dsl does not exist yet"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			design := writeSurfaceFixture(t, tc.ledger)
			if tc.mutate != nil {
				tc.mutate(t, design)
			}
			g := CheckSurface(design)
			if !strings.Contains(strings.Join(g.Errs, "\n"), tc.want) {
				t.Fatalf("want error containing %q, got %v", tc.want, g.Errs)
			}
		})
	}
}

func TestExplicitSurfaceGateRequiresLedger(t *testing.T) {
	design := t.TempDir()
	sel, err := Select(design, "gs", "")
	if err != nil {
		t.Fatal(err)
	}
	gates := RunSelected(design, "", sel, RunOptions{})
	if len(gates) != 1 || !strings.Contains(strings.Join(gates[0].Errs, "\n"), "no "+SurfaceLedgerName) {
		t.Fatalf("explicit gs on a design without a ledger must fail loudly: %+v", gates)
	}
}
