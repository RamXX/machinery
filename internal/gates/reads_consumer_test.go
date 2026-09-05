package gates

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// The explicit consumer column binds a matrix declaration to the exact
// architectural participant in the event contract, not to the matrix's stem.
// Multiple machines can belong to a consumer. Legacy matrices without this
// column remain usable only when their event has one distinct consumer.
const readsConsumerMatrixHeader = "| name | kind | signature | contract | consumer | maps to |\n|---|---|---|---|---|---|\n"

func readsConsumerDeclaration(consumer, reads string) string {
	return "| `applyPayment` | action | (ctx,evt) -> ctx | reacts to `markPaid` " + reads + " | " + consumer + " | `order-paid-final` |\n"
}

func readsConsumerRow(consumer, payload string) string {
	return "| markPaid | orders | " + consumer + " | " + payload + " | at-least-once | none | Order.id |\n"
}

const readsConsumerAllFields = "Order.id, Order.paidAt"

func readsConsumerFanout() string {
	return readsConsumerRow("payments", readsConsumerAllFields) + readsConsumerRow("audit", readsConsumerAllFields)
}

func readsConsumerDeclarations() map[string]string {
	return map[string]string{
		"PaymentHandler": readsConsumerMatrixHeader + readsConsumerDeclaration("payments", "READS{Order.id}"),
		"AuditHandler":   readsConsumerMatrixHeader + readsConsumerDeclaration("audit", "READS{Order.paidAt}"),
	}
}

// This is a real G2/Gx-ready design, including the lifecycle semantics source,
// declared participants, placement, invariant enforcement and build shape.
// Negative cases vary only the event contract or its consumer declarations.
func readsConsumerDesign(t *testing.T, rows string, matrices map[string]string) string {
	t.Helper()
	design := t.TempDir()
	model := strings.Replace(wiringModel, "    attributes:\n", "    attributes:\n      - name: id\n        type: string\n      - name: paidAt\n        type: string\n", 1)
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), model)
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), wiringMachine)
	mustWrite(t, filepath.Join(design, "formal", "Order.semantics.yaml"), "machine: Order\npattern: control-flow-only\nreason: this fixture exercises event-contract reconciliation, not data refinement\n")
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"), "| name | kind | signature | contract | maps to |\n|---|---|---|---|---|\n| `saveOrder` | actor | (ctx) -> Order | persists `markPaid` | `order-paid-final` |\n")
	placement := "| `Order` | in-memory |\n"
	for _, name := range []string{"PaymentHandler", "AuditHandler", "PaymentJournal"} {
		mustWrite(t, filepath.Join(design, "machines", name+".machine.json"), fmt.Sprintf(`{"id":%q,"_role":"operational","initial":"ready","states":{"ready":{"on":{"markPaid":{"target":"ready"}}}}}`, name))
		placement += "| `" + name + "` | in-memory |\n"
	}
	for name, body := range matrices {
		mustWrite(t, filepath.Join(design, "machines", name+".matrix.md"), body)
	}
	dsl := "workspace \"W\" \"reads fixture\" {\n  model {\n    sys = softwareSystem \"S\" \"system\" {\n"
	contract := "contract_version: 2\nboundaries:\n"
	for _, name := range []string{"orders", "payments", "audit", "archive"} {
		dsl += "      " + name + " = component \"" + name + "\" \"logic\" \"Go\"\n"
		contract += "  - id: " + name + "\n    code: [\"" + name + "/**\"]\n"
	}
	dsl += "    }\n  }\n}\n"
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	arch := "# Consumer reads fixture\n\n## Architecture Contract\n\n```yaml\n" + contract + "```\n\n" +
		"## Placement\n\n| component (placement) | persistence |\n|---|---|\n" + placement + "\n" +
		readsArmed + "Source: exhaustive markPaid fixture consumers.\n\n" + wiringHeader + rows +
		"\n## Traceability\n\n| invariant | where |\n|---|---|\n| order-paid-final | `Order` guard |\n" + nfrStub
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	mustWrite(t, filepath.Join(design, "BUILD.md"), "Mode: full\n\n## Toolchain\nGo 1.27.1.\n")
	return design
}

func readsConsumerCheck(t *testing.T, design string) *Gate {
	t.Helper()
	arch, err := os.ReadFile(filepath.Join(design, "ARCHITECTURE.md"))
	if err != nil {
		t.Fatal(err)
	}
	g := NewGate("consumer READS")
	checkReadsComplete(g, design, string(arch))
	return g
}

func readsConsumerRequireBlocked(t *testing.T, g *Gate, context string) {
	t.Helper()
	if len(g.Errs) == 0 {
		t.Fatalf("unsafe consumer READS contract accepted (%s); counts=%v", context, g.Counts)
	}
	if !strings.Contains(strings.ToLower(strings.Join(g.Errs, "\n")), "read") {
		t.Fatalf("expected actionable READS finding for %s, got unrelated errors: %v", context, g.Errs)
	}
}

func TestReadsConsumerLegacySingleControl(t *testing.T) {
	design := readsConsumerDesign(t, readsConsumerRow("payments", readsConsumerAllFields), map[string]string{
		"PaymentHandler": "| name | kind | signature | contract | maps to |\n|---|---|---|---|---|\n" + readsDeclRow,
	})
	for _, g := range []*Gate{readsConsumerCheck(t, design), CheckC4(design), CheckTraceability(design)} {
		if len(g.Errs) != 0 || len(g.Drift) != 0 {
			t.Fatalf("valid uniquely owned legacy control must pass: errors=%v drift=%v", g.Errs, g.Drift)
		}
	}
}

func TestReadsConsumerRepeatedLegacyStillHasOneOwner(t *testing.T) {
	rows := readsConsumerRow("payments", readsConsumerAllFields)
	design := readsConsumerDesign(t, rows+rows, map[string]string{
		"PaymentHandler": "| name | kind | signature | contract | maps to |\n|---|---|---|---|---|\n" + readsDeclRow,
	})
	if g := readsConsumerCheck(t, design); len(g.Errs) != 0 {
		t.Fatalf("repeated rows do not make a uniquely owned legacy event ambiguous: %v", g.Errs)
	}
}

func TestReadsConsumerExactEdgesPass(t *testing.T) {
	cases := []struct {
		name string
		rows string
		edit func(map[string]string)
	}{
		{"distinct_sets", readsConsumerRow("payments", "Order.id") + readsConsumerRow("audit", "Order.paidAt"), nil},
		{"strict_superset_is_local", readsConsumerRow("payments", "Order.id") + readsConsumerRow("audit", readsConsumerAllFields), func(m map[string]string) {
			m["AuditHandler"] = readsConsumerMatrixHeader + readsConsumerDeclaration("audit", "READS{Order.id, Order.paidAt}")
		}},
		{"repeated_identical_contract_edge", readsConsumerFanout() + readsConsumerRow("payments", readsConsumerAllFields), nil},
		{"two_machines_same_owner_agree", readsConsumerFanout(), func(m map[string]string) {
			m["PaymentJournal"] = readsConsumerMatrixHeader + readsConsumerDeclaration("payments", "READS{Order.id}")
		}},
		{"annotation_does_not_change_participant", readsConsumerRow("`payments` (projection worker)", "Order.id") + readsConsumerRow("audit", "Order.paidAt"), nil},
		{"reasoned_waiver_is_local", readsConsumerRow("payments (no reads: wake-up signal only)", "Order.id") + readsConsumerRow("audit", "Order.paidAt"), func(m map[string]string) {
			delete(m, "PaymentHandler")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := readsConsumerDeclarations()
			if tc.edit != nil {
				tc.edit(m)
			}
			design := readsConsumerDesign(t, tc.rows, m)
			g := readsConsumerCheck(t, design)
			if len(g.Errs) != 0 {
				t.Fatalf("valid edge-local READS contract rejected: %v", g.Errs)
			}
			if g.Counts["event-contract consumer reads declared"] == 0 {
				t.Fatalf("passing contract must actually check declarations: %v", g.Counts)
			}
		})
	}
}

func TestReadsConsumerRejectsOwnershipMutations(t *testing.T) {
	cases := []struct {
		name string
		rows string
		edit func(map[string]string)
	}{
		{"missing_payments", readsConsumerFanout(), func(m map[string]string) { delete(m, "PaymentHandler") }},
		{"missing_audit", readsConsumerFanout(), func(m map[string]string) { delete(m, "AuditHandler") }},
		{"reassigned_owner", readsConsumerFanout(), func(m map[string]string) {
			m["PaymentHandler"] = readsConsumerMatrixHeader + readsConsumerDeclaration("audit", "READS{Order.id}")
		}},
		{"renamed_consumer", readsConsumerRow("archive", readsConsumerAllFields) + readsConsumerRow("audit", readsConsumerAllFields), nil},
		{"near_match_owner", readsConsumerFanout(), func(m map[string]string) {
			m["PaymentHandler"] = readsConsumerMatrixHeader + readsConsumerDeclaration("paymentsBackup", "READS{Order.id}")
		}},
		{"two_machines_same_owner_disagree", readsConsumerFanout(), func(m map[string]string) {
			m["PaymentJournal"] = readsConsumerMatrixHeader + readsConsumerDeclaration("payments", "READS{Order.paidAt}")
		}},
		{"conflicting_duplicate_row", readsConsumerFanout(), func(m map[string]string) {
			m["PaymentHandler"] += readsConsumerDeclaration("payments", "READS{Order.paidAt}")
		}},
		{"conflicting_declarations_one_cell", readsConsumerFanout(), func(m map[string]string) {
			m["PaymentHandler"] = readsConsumerMatrixHeader + readsConsumerDeclaration("payments", "READS{Order.id} READS{Order.paidAt}")
		}},
		{"waiver_does_not_cover_sibling", readsConsumerRow("payments (no reads: wake-up signal only)", readsConsumerAllFields) + readsConsumerRow("audit", readsConsumerAllFields), func(m map[string]string) { delete(m, "AuditHandler") }},
		{"empty_waiver_reason", readsConsumerRow("payments (no reads: )", readsConsumerAllFields) + readsConsumerRow("audit", readsConsumerAllFields), nil},
		{"payload_narrowed_only_for_audit", readsConsumerRow("payments", readsConsumerAllFields) + readsConsumerRow("audit", "Order.id"), nil},
		{"payload_narrowed_only_for_payments", readsConsumerRow("payments", "Order.paidAt") + readsConsumerRow("audit", readsConsumerAllFields), nil},
		{"repeated_edge_short_payload", readsConsumerFanout() + readsConsumerRow("audit", "Order.id"), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := readsConsumerDeclarations()
			if tc.edit != nil {
				tc.edit(m)
			}
			design := readsConsumerDesign(t, tc.rows, m)
			g := readsConsumerCheck(t, design)
			readsConsumerRequireBlocked(t, g, tc.name)
			missingOwner := map[string]string{
				"missing_payments": "payments", "missing_audit": "audit", "renamed_consumer": "archive",
				"reassigned_owner": "payments", "waiver_does_not_cover_sibling": "audit",
			}[tc.name]
			if missingOwner != "" && !strings.Contains(strings.Join(g.Errs, "\n"), missingOwner) {
				t.Fatalf("finding must identify the unsatisfied consumer %q: %v", missingOwner, g.Errs)
			}
			if second := readsConsumerCheck(t, design); !reflect.DeepEqual(g.Errs, second.Errs) {
				t.Fatalf("same input produced different diagnostics: %v / %v", g.Errs, second.Errs)
			}
		})
	}
}

func TestReadsConsumerAmbiguousLegacyNeedsMigration(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		t.Run(fmt.Sprintf("mixed_explicit_%t", mixed), func(t *testing.T) {
			m := map[string]string{"PaymentHandler": "| name | kind | signature | contract | maps to |\n|---|---|---|---|---|\n" + readsDeclRow}
			if mixed {
				m["AuditHandler"] = readsConsumerMatrixHeader + readsConsumerDeclaration("audit", "READS{Order.id}")
			}
			g := readsConsumerCheck(t, readsConsumerDesign(t, readsConsumerFanout(), m))
			readsConsumerRequireBlocked(t, g, "legacy event-wide declaration has multiple possible owners")
			message := strings.ToLower(strings.Join(g.Errs, "\n"))
			for _, want := range []string{"markpaid", "consumer"} {
				if !strings.Contains(message, want) {
					t.Fatalf("migration finding must identify %q: %v", want, g.Errs)
				}
			}
			if !strings.Contains(message, "column") && !strings.Contains(message, "explicit") && !strings.Contains(message, "owner") {
				t.Fatalf("ambiguous legacy input needs ownership migration guidance: %v", g.Errs)
			}
		})
	}
}

func TestReadsConsumerMalformedDeclarationsFailClosed(t *testing.T) {
	cases := []struct{ name, consumer, declaration string }{
		{"empty_set", "payments", "READS{}"},
		{"empty_members", "payments", "READS{,}"},
		{"trailing_empty_member", "payments", "READS{Order.id,}"},
		{"unclosed_set", "payments", "READS{Order.id"},
		{"duplicate_field", "payments", "READS{Order.id, Order.id}"},
		{"blank_explicit_owner", "", "READS{Order.id}"},
		{"multiple_owners", "payments, audit", "READS{Order.id}"},
		{"unknown_payload_field", "payments", "READS{Order.nonexistent}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := readsConsumerDeclarations()
			m["PaymentHandler"] = readsConsumerMatrixHeader + readsConsumerDeclaration(tc.consumer, tc.declaration)
			readsConsumerRequireBlocked(t, readsConsumerCheck(t, readsConsumerDesign(t, readsConsumerFanout(), m)), tc.name)
		})
	}
}

func TestReadsConsumerExplicitBlankOwnerCannotUseLegacyFallback(t *testing.T) {
	m := map[string]string{"PaymentHandler": readsConsumerMatrixHeader + readsConsumerDeclaration("", "READS{Order.id}")}
	g := readsConsumerCheck(t, readsConsumerDesign(t, readsConsumerRow("payments", readsConsumerAllFields), m))
	readsConsumerRequireBlocked(t, g, "an explicitly present but empty consumer cell is not a legacy declaration")
}

// CLI integration compiles this worktree into an isolated temporary candidate.
// It never invokes or replaces the user's installed Machinery executable.
func TestReadsConsumerCLI(t *testing.T) {
	root, err := filepath.Abs(repoRoot())
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "machinery")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, "./cmd/machinery")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("CLI prerequisite build failed (not RED evidence): %v\n%s", err, out)
	}
	cases := []struct {
		name string
		rows string
		gate string
		bad  bool
		edit func(map[string]string)
	}{
		{"legacy_control", readsConsumerRow("payments", readsConsumerAllFields), "g2,gx", false, func(m map[string]string) {
			delete(m, "AuditHandler")
			m["PaymentHandler"] = "| name | kind | signature | contract | maps to |\n|---|---|---|---|---|\n" + readsDeclRow
		}},
		{"distinct_sets", readsConsumerRow("payments", "Order.id") + readsConsumerRow("audit", "Order.paidAt"), "g2,gx", false, nil},
		{"missing_audit", readsConsumerFanout(), "g2,gx", true, func(m map[string]string) { delete(m, "AuditHandler") }},
		{"missing_payments", readsConsumerFanout(), "g2,gx", true, func(m map[string]string) { delete(m, "PaymentHandler") }},
		{"audit_payload_narrowed", readsConsumerRow("payments", readsConsumerAllFields) + readsConsumerRow("audit", "Order.id"), "g2,gx", true, nil},
		{"payments_payload_narrowed", readsConsumerRow("payments", "Order.paidAt") + readsConsumerRow("audit", readsConsumerAllFields), "g2,gx", true, nil},
		{"renamed_known_consumer", readsConsumerRow("archive", readsConsumerAllFields) + readsConsumerRow("audit", readsConsumerAllFields), "g2,gx", true, nil},
		{"conflicting_duplicate", readsConsumerFanout(), "g2,gx", true, func(m map[string]string) {
			m["PaymentHandler"] += readsConsumerDeclaration("payments", "READS{Order.paidAt}")
		}},
		{"unknown_participant_G2", readsConsumerRow("missingComponent", readsConsumerAllFields), "g2", true, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := readsConsumerDeclarations()
			if tc.edit != nil {
				tc.edit(m)
			}
			design := readsConsumerDesign(t, tc.rows, m)
			runCtx, runCancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer runCancel()
			cmd := exec.CommandContext(runCtx, bin, "check", design, "--gate", tc.gate)
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			if runCtx.Err() != nil {
				t.Fatalf("CLI timed out (not RED evidence): %v\n%s", runCtx.Err(), out)
			}
			if tc.bad {
				if err == nil {
					t.Fatalf("unsafe %s contract returned CLI success:\n%s", tc.name, out)
				}
				exitErr, ok := err.(*exec.ExitError)
				if !ok || exitErr.ExitCode() != 1 {
					t.Fatalf("expected gate rejection exit 1, not infrastructure failure: %v\n%s", err, out)
				}
				want := "read"
				if tc.gate == "g2" {
					want = "missingcomponent"
				}
				intendedFinding := false
				for _, line := range strings.Split(string(out), "\n") {
					if strings.Contains(line, "ERROR") && strings.Contains(strings.ToLower(line), want) {
						intendedFinding = true
					}
				}
				if !intendedFinding {
					t.Fatalf("CLI rejection did not diagnose the intended %s obligation:\n%s", want, out)
				}
			} else if err != nil || !strings.Contains(string(out), "0 blocking") {
				t.Fatalf("valid %s contract must pass both gates: %v\n%s", tc.name, err, out)
			}
		})
	}
}
