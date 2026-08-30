package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// targetSurfaceModelYAML is the Phase 1 target model the ledger resolves
// against: two human acts (one of them on a second persona), one System act
// that carries no obligation, and one action with no actor at all.
const targetSurfaceModelYAML = `
kind: DomainModel
version: v1
title: Target
entities:
  Deal:
    definition: a deal under negotiation
    attributes:
      - {name: id, type: string}
    actions:
      - {name: create, actor: Seller}
      - {name: approve, actor: TenantAdmin}
      - {name: expire, actor: System}
      - {name: archive}
  Tenant:
    definition: a customer tenant
    attributes:
      - {name: id, type: string}
    actions:
      - {name: suspend, actor: TenantAdmin}
`

const targetSurfacesLedger = `
surface_version: 1
sources:
  - domain.modelith.yaml action list, walked persona by persona
acts:
  - {act: Deal.create, actor: Seller, surface: "Deals > New deal screen"}
  - {act: Deal.approve, actor: TenantAdmin, surface: "Admin console > Approvals queue", milestone: M2}
  - {act: Tenant.suspend, actor: TenantAdmin, surface: "admin CLI: tenant suspend <id>"}
  - {act: "knob:billing.grace_period_days", actor: TenantAdmin, surface: "tenant settings release, Billing tab"}
`

func writeTargetSurfaceFixture(t *testing.T, ledger string) string {
	t.Helper()
	design := t.TempDir()
	files := map[string]string{
		"domain.modelith.yaml": targetSurfaceModelYAML,
		TargetSurfacesName:     ledger,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(design, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return design
}

func checkedLine(g *Gate) string { return strings.Join(g.checkedExtra, ", ") }

func TestCheckTargetSurfacesClean(t *testing.T) {
	design := writeTargetSurfaceFixture(t, targetSurfacesLedger)
	g := CheckTargetSurfaces(design)
	if len(g.Errs) != 0 || len(g.Drift) != 0 {
		t.Fatalf("Gu not clean: errs=%v drift=%v", g.Errs, g.Drift)
	}
	want := "3 obligated actions, 3 covered, 0 deferred acts, 0 deferred personas, 1 knob rows, 1 actorless actions"
	if got := checkedLine(g); got != want {
		t.Errorf("checked line = %q, want %q", got, want)
	}
	sel, err := Select(design, "", "")
	if err != nil || !sel.Run["gu"] {
		t.Fatalf("default gate selection omitted gu: sel=%+v err=%v", sel, err)
	}
	found := false
	for _, gate := range RunSelected(design, "", sel, RunOptions{}) {
		found = found || strings.Contains(gate.Title, "Gu-surfaces")
	}
	if !found {
		t.Error("RunSelected skipped an authored target surface ledger")
	}
}

// The deferral lanes: one act deferred by name, one persona deferred wholesale.
// Both discharge the completeness obligation, and both are counted separately
// from coverage so the reviewer sees what was mapped and what was punted.
func TestCheckTargetSurfacesDeferrals(t *testing.T) {
	cases := []struct {
		name   string
		ledger string
		want   string
	}{
		{
			name: "act level deferral",
			ledger: `
surface_version: 1
acts:
  - {act: Deal.create, actor: Seller, surface: "Deals > New deal screen"}
  - {act: Deal.approve, actor: TenantAdmin, surface: "Admin console > Approvals queue"}
deferrals:
  - {act: Tenant.suspend, reason: tenant lifecycle admin lands in M4}
`,
			want: "3 obligated actions, 2 covered, 1 deferred acts, 0 deferred personas, 0 knob rows, 1 actorless actions",
		},
		{
			name: "actor level deferral covers the persona",
			ledger: `
surface_version: 1
acts:
  - {act: Deal.create, actor: Seller, surface: "Deals > New deal screen"}
deferrals:
  - {act: "actor:TenantAdmin", reason: the whole admin console is scoped to M4}
`,
			want: "3 obligated actions, 1 covered, 0 deferred acts, 1 deferred personas, 0 knob rows, 1 actorless actions",
		},
		{
			name: "knob deferral counts as a deferred act",
			ledger: `
surface_version: 1
acts:
  - {act: Deal.create, actor: Seller, surface: "Deals > New deal screen"}
  - {act: Deal.approve, actor: TenantAdmin, surface: "Admin console > Approvals queue"}
  - {act: Tenant.suspend, actor: TenantAdmin, surface: "admin CLI: tenant suspend <id>"}
deferrals:
  - {act: "knob:billing.grace_period_days", reason: billing knobs are configured by support until M5}
`,
			want: "3 obligated actions, 3 covered, 1 deferred acts, 0 deferred personas, 0 knob rows, 1 actorless actions",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := CheckTargetSurfaces(writeTargetSurfaceFixture(t, tc.ledger))
			if len(g.Errs) != 0 {
				t.Fatalf("ledger must be clean: %v", g.Errs)
			}
			if got := checkedLine(g); got != tc.want {
				t.Errorf("checked line = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCheckTargetSurfacesMutations(t *testing.T) {
	cases := []struct {
		name   string
		ledger string
		mutate func(t *testing.T, design string)
		want   string
	}{
		{"unknown root key", targetSurfacesLedger + "bogus: true\n", nil, `unsupported key "bogus"`},
		{"bad version", strings.Replace(targetSurfacesLedger, "surface_version: 1", "surface_version: 2", 1), nil, "surface_version must be the integer 1"},
		{"acts missing", "surface_version: 1\n", nil, "acts is required"},
		{"acts not a list", "surface_version: 1\nacts: {}\n", nil, "acts must be a list"},
		{"deferrals not a list", targetSurfacesLedger + "deferrals: {}\n", nil, "deferrals must be a list"},
		{"sources not strings", strings.Replace(targetSurfacesLedger, "  - domain.modelith.yaml action list, walked persona by persona", "  - 42", 1), nil, "sources[0] must be a non-empty string"},
		{
			"unknown act key",
			strings.Replace(targetSurfacesLedger, "{act: Deal.create, actor: Seller, surface: \"Deals > New deal screen\"}", "{act: Deal.create, actor: Seller, surface: x, screen: y}", 1),
			nil, `unsupported key "screen" in acts[0]`,
		},
		{
			"unknown deferral key",
			targetSurfacesLedger + "deferrals:\n  - {act: \"actor:TenantAdmin\", reason: later, owner: nobody}\n",
			nil, `unsupported key "owner" in deferrals[0]`,
		},
		{
			"missing obligated action",
			strings.Replace(targetSurfacesLedger, "  - {act: Tenant.suspend, actor: TenantAdmin, surface: \"admin CLI: tenant suspend <id>\"}\n", "", 1),
			nil, "Tenant.suspend (actor TenantAdmin) is named by no acts row and no deferral",
		},
		{
			"dangling entity",
			strings.Replace(targetSurfacesLedger, "act: Deal.create", "act: Widget.create", 1),
			nil, `names unknown entity "Widget"`,
		},
		{
			"dangling action",
			strings.Replace(targetSurfacesLedger, "act: Deal.create", "act: Deal.destroy", 1),
			nil, `names unknown action "destroy" on entity Deal`,
		},
		{
			"actor mismatch",
			strings.Replace(targetSurfacesLedger, "{act: Deal.approve, actor: TenantAdmin,", "{act: Deal.approve, actor: Seller,", 1),
			nil, `names actor "Seller" but the target model declares "TenantAdmin"`,
		},
		{
			"actor named for an actorless action",
			strings.Replace(targetSurfacesLedger, "act: Deal.create, actor: Seller", "act: Deal.archive, actor: Seller", 1),
			nil, "the target model declares no actor for that action",
		},
		{
			"row without a surface",
			strings.Replace(targetSurfacesLedger, `{act: Deal.create, actor: Seller, surface: "Deals > New deal screen"}`, "{act: Deal.create, actor: Seller}", 1),
			nil, "names no surface",
		},
		{
			"row without an actor",
			strings.Replace(targetSurfacesLedger, `{act: Deal.create, actor: Seller, surface: "Deals > New deal screen"}`, "{act: Deal.create, surface: x}", 1),
			nil, "names no actor",
		},
		{
			"empty milestone",
			strings.Replace(targetSurfacesLedger, "milestone: M2", `milestone: ""`, 1),
			nil, "has an empty milestone",
		},
		{
			"duplicate act row",
			targetSurfacesLedger + "  - {act: Deal.create, actor: Seller, surface: another screen}\n",
			nil, `names act "Deal.create", which the ledger already names`,
		},
		{
			"act both mapped and deferred",
			targetSurfacesLedger + "deferrals:\n  - {act: Deal.approve, reason: later}\n",
			nil, `names act "Deal.approve", which the ledger already names`,
		},
		{
			"duplicate knob row",
			targetSurfacesLedger + `  - {act: "knob:billing.grace_period_days", actor: TenantAdmin, surface: elsewhere}` + "\n",
			nil, `names act "knob:billing.grace_period_days", which the ledger already names`,
		},
		{
			"knob with no key",
			strings.Replace(targetSurfacesLedger, `act: "knob:billing.grace_period_days"`, `act: "knob:"`, 1),
			nil, "names a knob with no key",
		},
		{
			"act is neither shape",
			strings.Replace(targetSurfacesLedger, "act: Deal.create", "act: createADeal", 1),
			nil, `act "createADeal" is neither an Entity.action nor a knob:<key>`,
		},
		{
			"deferral without a reason",
			targetSurfacesLedger + "deferrals:\n  - {act: \"actor:Seller\"}\n",
			nil, "is deferred without a reason",
		},
		{
			"deferral of an unknown persona",
			targetSurfacesLedger + "deferrals:\n  - {act: \"actor:Auditor\", reason: later}\n",
			nil, `defers actor "Auditor", which no action in the target model declares`,
		},
		{
			"deferral act is neither shape",
			targetSurfacesLedger + "deferrals:\n  - {act: whatever, reason: later}\n",
			nil, `act "whatever" is not an Entity.action, a knob:<key>, or an actor:<Name>`,
		},
		{"acts row is not a mapping", "surface_version: 1\nacts:\n  - just a string\n", nil, "acts[0] is not a mapping"},
		{"deferral row is not a mapping", targetSurfacesLedger + "deferrals:\n  - just a string\n", nil, "deferrals[0] is not a mapping"},
		{"missing target model", targetSurfacesLedger, func(t *testing.T, design string) {
			if err := os.Remove(filepath.Join(design, "domain.modelith.yaml")); err != nil {
				t.Fatal(err)
			}
		}, "no *.modelith.yaml in the design directory"},
		{"target model is not a modelith model", targetSurfacesLedger, func(t *testing.T, design string) {
			if err := os.WriteFile(filepath.Join(design, "domain.modelith.yaml"), []byte("kind: Other\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "must be a Modelith DomainModel v1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			design := writeTargetSurfaceFixture(t, tc.ledger)
			if tc.mutate != nil {
				tc.mutate(t, design)
			}
			g := CheckTargetSurfaces(design)
			if !strings.Contains(strings.Join(g.Errs, "\n"), tc.want) {
				t.Fatalf("want error containing %q, got %v", tc.want, g.Errs)
			}
		})
	}
}

// A System act and an actorless act carry no obligation: the ledger that names
// neither is still clean, and the actorless count keeps partial actor adoption
// visible instead of reading as full coverage.
func TestCheckTargetSurfacesActorlessCarryNoObligation(t *testing.T) {
	design := writeTargetSurfaceFixture(t, targetSurfacesLedger)
	g := CheckTargetSurfaces(design)
	joined := strings.Join(g.Errs, "\n")
	for _, act := range []string{"Deal.expire", "Deal.archive"} {
		if strings.Contains(joined, act) {
			t.Errorf("%s carries no surface obligation but was reported: %v", act, g.Errs)
		}
	}
	if !strings.Contains(checkedLine(g), "1 actorless actions") {
		t.Errorf("the actorless count must stay visible: %q", checkedLine(g))
	}
}

// A model in which nobody named an actor yet arms nothing. That must be a
// stated non-check, never a silent green.
func TestCheckTargetSurfacesNoHumanActs(t *testing.T) {
	design := t.TempDir()
	model := "kind: DomainModel\nversion: v1\nentities:\n  Deal:\n    definition: d\n    actions:\n      - {name: create}\n"
	write := map[string]string{"domain.modelith.yaml": model, TargetSurfacesName: "surface_version: 1\nacts: []\n"}
	for name, content := range write {
		if err := os.WriteFile(filepath.Join(design, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	g := CheckTargetSurfaces(design)
	if len(g.Errs) != 0 {
		t.Fatalf("an empty obligated set is not an error: %v", g.Errs)
	}
	if !strings.Contains(strings.Join(g.Notes, "\n"), "no target-model action names a human actor") {
		t.Errorf("an unarmed gate must say so: %v", g.Notes)
	}
	if got := checkedLine(g); got != "0 obligated actions, 0 covered, 0 deferred acts, 0 deferred personas, 0 knob rows, 1 actorless actions" {
		t.Errorf("zeros must stay visible on the checked line: %q", got)
	}
}

// Activation mirrors Gs and Gp: presence of the artifact arms the gate in a
// default run, and an explicit --gate gu with no artifact fails loudly rather
// than skipping.
func TestTargetSurfacesActivation(t *testing.T) {
	t.Run("absent file skips the gate in a default run", func(t *testing.T) {
		design := t.TempDir()
		if err := os.WriteFile(filepath.Join(design, "domain.modelith.yaml"), []byte(targetSurfaceModelYAML), 0o644); err != nil {
			t.Fatal(err)
		}
		if HasTargetSurfaces(design) {
			t.Fatal("HasTargetSurfaces must be false without the artifact")
		}
		sel, err := Select(design, "", "")
		if err != nil {
			t.Fatal(err)
		}
		for _, gate := range RunSelected(design, "", sel, RunOptions{}) {
			if strings.Contains(gate.Title, "Gu-surfaces") {
				t.Fatal("Gu ran without an authored ledger")
			}
		}
	})
	t.Run("explicit gu with no file errors", func(t *testing.T) {
		design := t.TempDir()
		sel, err := Select(design, "gu", "")
		if err != nil {
			t.Fatal(err)
		}
		gates := RunSelected(design, "", sel, RunOptions{})
		if len(gates) != 1 || !strings.Contains(strings.Join(gates[0].Errs, "\n"), "no "+TargetSurfacesName) {
			t.Fatalf("explicit gu on a design without a ledger must fail loudly: %+v", gates)
		}
	})
	t.Run("ledger that is not a mapping", func(t *testing.T) {
		design := writeTargetSurfaceFixture(t, "- just a list\n")
		g := CheckTargetSurfaces(design)
		if !strings.Contains(strings.Join(g.Errs, "\n"), "is not a yaml mapping") {
			t.Fatalf("want a mapping error, got %v", g.Errs)
		}
	})
}

// Every *.modelith.yaml at the design ROOT contributes obligations (sharded
// models are one model); the legacy model under legacy/ never does.
func TestCheckTargetSurfacesShardedModel(t *testing.T) {
	design := writeTargetSurfaceFixture(t, targetSurfacesLedger)
	shard := "kind: DomainModel\nversion: v1\nentities:\n  Invoice:\n    definition: i\n    actions:\n      - {name: void, actor: Billing}\n"
	if err := os.WriteFile(filepath.Join(design, "billing.modelith.yaml"), []byte(shard), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(design, "legacy"), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := "kind: DomainModel\nversion: v1\nentities:\n  Ghost:\n    definition: g\n    actions:\n      - {name: haunt, actor: Ancestor}\n"
	if err := os.WriteFile(filepath.Join(design, "legacy", "domain.modelith.yaml"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	g := CheckTargetSurfaces(design)
	joined := strings.Join(g.Errs, "\n")
	if !strings.Contains(joined, "Invoice.void (actor Billing)") {
		t.Errorf("a second root shard must contribute obligations: %v", g.Errs)
	}
	if strings.Contains(joined, "Ghost.haunt") {
		t.Errorf("the legacy model must not contribute obligations: %v", g.Errs)
	}
}
