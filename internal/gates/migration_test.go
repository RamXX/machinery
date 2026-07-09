package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const migrationLegacyModel = `
kind: DomainModel
version: v1
title: Legacy
enums:
  LegacyStatus:
    values:
      - {name: Old, definition: old}
      - {name: Live, definition: live}
entities:
  LegacyThing:
    definition: legacy thing
    attributes:
      - {name: id, type: string}
      - {name: status, type: LegacyStatus}
`

const migrationTargetModel = `
kind: DomainModel
version: v1
title: Target
enums:
  TargetStatus:
    values:
      - {name: Ready, definition: ready}
      - {name: Done, definition: done}
entities:
  Thing:
    definition: target thing
    attributes:
      - {name: id, type: string}
      - {name: status, type: TargetStatus}
      - {name: createdAt, type: timestamp}
  Audit:
    definition: new audit record
    attributes:
      - {name: id, type: string}
`

const migrationContract = `
contract_version: 1
mode: rebuild
legacy: {model: legacy/domain.modelith.yaml}
target: {model: domain.modelith.yaml}
dispositions:
  - {legacy: LegacyThing, target: Thing, strategy: replace, rationale: replace the prototype store}
new_entities: [Audit]
assets:
  - {name: legacy characterization suite, kind: test, strategy: reuse, target: target compatibility suite, rationale: preserves known behavior, verification: run against legacy and target adapters}
data_mappings:
  - {source: LegacyThing.id, target: Thing.id, transform: identity, validation: exact id equality, rollback: restore snapshot}
  - {source: LegacyThing.status, target: Thing.status, transform: translate with state mapping, validation: every legacy value translated, rollback: restore snapshot}
  - {source: "-", target: Thing.createdAt, transform: derive from import timestamp, validation: non-null timestamp, rollback: discard target row}
state_mappings:
  - {source: LegacyThing.Old, target: Thing.Ready, reason: old records enter the ready state}
  - {source: LegacyThing.Live, target: Thing.Done, reason: live legacy records are complete}
phases:
  - id: baseline
    source_of_truth: legacy
    read_path: legacy
    write_path: legacy
    backfill: no backfill before shadow infrastructure is ready
    entry_criteria: legacy characterization suite is green
    exit_criteria: target import can run repeatedly
    rollback: remain on legacy
    observability: [legacy error rate]
  - id: cutover
    source_of_truth: target
    read_path: target
    write_path: target
    backfill: final incremental import before traffic switch
    entry_criteria: parity and reconciliation are green
    exit_criteria: rollback window expires without regression
    rollback: return traffic to baseline and restore snapshot
    observability: [target error rate, reconciliation drift]
cutover:
  phase: cutover
  rollback_phase: baseline
  decision_criteria: zero reconciliation drift and target SLOs green
  rollback_window: 24h
  max_data_loss: zero acknowledged writes
risks:
  - dependency: import pipeline
    detection: import lag and row-count mismatch
    mitigation: stop cutover and replay from snapshot
    residual: malformed legacy records require manual quarantine
    owner: migration steward
`

func writeMigrationFixture(t *testing.T, contract string) string {
	t.Helper()
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "legacy"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"domain.modelith.yaml":        migrationTargetModel,
		"legacy/domain.modelith.yaml": migrationLegacyModel,
		MigrationContractName:         contract,
		"ARCHITECTURE.md":             "# Architecture\n\n## Transition architecture\n\nLegacy and target coexist behind a switch.\n",
		"BUILD.md":                    "# Build\n\n## Migration implementation plan\n\nImplement the checked transition contract.\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(design, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return design
}

func TestCheckMigrationClean(t *testing.T) {
	design := writeMigrationFixture(t, migrationContract)
	g := CheckMigration(design)
	if len(g.Errs) != 0 || len(g.Drift) != 0 {
		t.Fatalf("Gm not clean: errs=%v drift=%v", g.Errs, g.Drift)
	}
	for _, count := range []string{"legacy entities", "target entities", "dispositions", "salvage decisions", "data mappings", "state mappings", "transition phases", "cutover contracts", "transition risks"} {
		if g.Counts[count] == 0 {
			t.Errorf("Gm did not count %s: %+v", count, g.Counts)
		}
	}
	sel, err := Select(design, "")
	if err != nil || !sel.Run["gm"] {
		t.Fatalf("default gate selection omitted gm: sel=%+v err=%v", sel, err)
	}
	found := false
	for _, gate := range RunSelected(design, "", sel) {
		found = found || strings.Contains(gate.Title, "Gm-transition")
	}
	if !found {
		t.Error("RunSelected skipped an authored migration contract")
	}
}

func TestCheckMigrationMutations(t *testing.T) {
	authorityRegression := strings.Replace(migrationContract, "  - id: cutover\n    source_of_truth: target", "  - id: cutover\n    source_of_truth: legacy", 1)
	authorityRegression = strings.Replace(authorityRegression, "  - id: baseline\n    source_of_truth: legacy", "  - id: baseline\n    source_of_truth: target", 1)
	cases := []struct {
		name     string
		contract string
		mutate   func(t *testing.T, design string)
		want     string
	}{
		{"unknown key", migrationContract + "bogus: true\n", nil, "unsupported key"},
		{"missing disposition", strings.Replace(migrationContract, "  - {legacy: LegacyThing, target: Thing, strategy: replace, rationale: replace the prototype store}\n", "", 1), nil, "has no disposition"},
		{"incomplete data mapping", strings.Replace(migrationContract, "  - {source: \"-\", target: Thing.createdAt, transform: derive from import timestamp, validation: non-null timestamp, rollback: discard target row}\n", "", 1), nil, "does not map or derive target attribute Thing.createdAt"},
		{"incomplete state mapping", strings.Replace(migrationContract, "  - {source: LegacyThing.Old, target: Thing.Ready, reason: old records enter the ready state}\n", "", 1), nil, "does not map or drain legacy lifecycle value LegacyThing.Old"},
		{"dual write obligations", strings.Replace(migrationContract, "write_path: legacy", "write_path: dual", 1), nil, "idempotency is required"},
		{"shadow parity", strings.Replace(migrationContract, "read_path: legacy", "read_path: shadow", 1), nil, "parity is required"},
		{"bad cutover", strings.Replace(migrationContract, "source_of_truth: target", "source_of_truth: legacy", 1), nil, "must use target"},
		{"authority regression", authorityRegression, nil, "cannot return source_of_truth to legacy"},
		{"path escape", strings.Replace(migrationContract, "legacy/domain.modelith.yaml", "../outside.modelith.yaml", 1), nil, "escapes the design directory"},
		{"same model", strings.Replace(migrationContract, "legacy/domain.modelith.yaml", "domain.modelith.yaml", 1), nil, "must be different files"},
		{"invalid legacy model", migrationContract, func(t *testing.T, design string) {
			path := filepath.Join(design, "legacy", "domain.modelith.yaml")
			if err := os.WriteFile(path, []byte(strings.Replace(migrationLegacyModel, "kind: DomainModel", "kind: Other", 1)), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "must be a Modelith DomainModel v1"},
		{"missing architecture bridge", migrationContract, func(t *testing.T, design string) {
			if err := os.WriteFile(filepath.Join(design, "ARCHITECTURE.md"), []byte("# Architecture\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "Transition architecture"},
		{"missing build bridge", migrationContract, func(t *testing.T, design string) {
			if err := os.WriteFile(filepath.Join(design, "BUILD.md"), []byte("# Build\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "Migration implementation plan"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			design := writeMigrationFixture(t, tc.contract)
			if tc.mutate != nil {
				tc.mutate(t, design)
			}
			g := CheckMigration(design)
			if !strings.Contains(strings.Join(g.Errs, "\n"), tc.want) {
				t.Fatalf("want error containing %q, got %v", tc.want, g.Errs)
			}
		})
	}
}

func TestExplicitMigrationGateRequiresContract(t *testing.T) {
	design := t.TempDir()
	sel, err := Select(design, "gm")
	if err != nil {
		t.Fatal(err)
	}
	gates := RunSelected(design, "", sel)
	if len(gates) != 1 || len(gates[0].Errs) == 0 || !strings.Contains(gates[0].Errs[0], "no migration.yaml") {
		t.Fatalf("explicit gm did not fail on absence: %+v", gates)
	}
}
