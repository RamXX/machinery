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
	sel, err := Select(design, "")
	if err != nil || !sel.Run["gs"] {
		t.Fatalf("default gate selection omitted gs: sel=%+v err=%v", sel, err)
	}
	found := false
	for _, gate := range RunSelected(design, "", sel) {
		found = found || strings.Contains(gate.Title, "Gs-surface")
	}
	if !found {
		t.Error("RunSelected skipped an authored surface ledger")
	}
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
	sel, err := Select(design, "gs")
	if err != nil {
		t.Fatal(err)
	}
	gates := RunSelected(design, "", sel)
	if len(gates) != 1 || !strings.Contains(strings.Join(gates[0].Errs, "\n"), "no "+SurfaceLedgerName) {
		t.Fatalf("explicit gs on a design without a ledger must fail loudly: %+v", gates)
	}
}
