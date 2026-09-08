package gates

import (
	"strings"
	"testing"
)

// These extra cases exercise authoring boundaries without changing frozen RED.
func TestReadsConsumerSupplementalPass(t *testing.T) {
	cases := []struct{ name, matrix string }{
		{"reordered_annotated_owner", "| consumer (participant) | contract |\n|---|---|\n| `payments` (worker) | `markPaid` READS{Order.id} |\n"},
		{"escaped_pipe_before_owner", "| contract | consumer |\n|---|---|\n| x \\| `markPaid` READS{Order.id} | payments |\n"},
		{"table_local_column_positions", "| contract | consumer |\n|---|---|\n| `markPaid` READS{Order.id} | payments |\n\n| consumer | contract |\n|---|---|\n| audit | `markPaid` READS{Order.paidAt} |\n"},
		{"legacy_prose_unique_owner", "Consumes `markPaid` READS{Order.id}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows := readsConsumerRow("payments", "Order.id")
			if tc.name == "table_local_column_positions" {
				rows += readsConsumerRow("audit", "Order.paidAt")
			}
			g := readsConsumerCheck(t, readsConsumerDesign(t, rows, map[string]string{"PaymentHandler": tc.matrix}))
			if len(g.Errs) != 0 || g.Counts["event-contract consumer reads declared"] == 0 {
				t.Fatalf("valid declaration must execute and pass: errors=%v counts=%v", g.Errs, g.Counts)
			}
		})
	}
}

func TestReadsConsumerSupplementalExactSetOrder(t *testing.T) {
	m := readsConsumerDeclarations()
	m["PaymentHandler"] = readsConsumerMatrixHeader + readsConsumerDeclaration("payments", "READS{Order.id, Order.paidAt}")
	m["PaymentJournal"] = readsConsumerMatrixHeader + readsConsumerDeclaration("payments", "READS{`Order.paidAt`, `Order.id`}")
	g := readsConsumerCheck(t, readsConsumerDesign(t, readsConsumerFanout(), m))
	if len(g.Errs) != 0 {
		t.Fatalf("the same exact set in a different order must agree: %v", g.Errs)
	}
}

func TestReadsConsumerSupplementalFailClosed(t *testing.T) {
	cases := []struct{ name, matrix, consumer string }{
		{"duplicate_owner_columns", "| contract | consumer | consumer |\n|---|---|---|\n| `markPaid` READS{Order.id} | payments | payments |\n", "payments"},
		{"short_explicit_owner_row", "| contract | consumer |\n|---|---|\n| `markPaid` READS{Order.id} |\n", "payments"},
		{"extra_closing_brace", readsConsumerMatrixHeader + readsConsumerDeclaration("payments", "READS{Order.id}}"), "payments"},
		{"unclosed_waiver_with_valid_declaration", readsConsumerMatrixHeader + readsConsumerDeclaration("payments", "READS{Order.id}"), "payments (no reads: wake-up"},
		{"duplicate_waivers", readsConsumerMatrixHeader + readsConsumerDeclaration("payments", "READS{Order.id}"), "payments (no reads: wake-up) (no reads: )"},
		{"unknown_owner_cannot_hide_behind_valid_row", readsConsumerMatrixHeader + readsConsumerDeclaration("payments", "READS{Order.id}") + readsConsumerDeclaration("archive", "READS{Order.id}"), "payments"},
		{"nonconsumer_header_is_legacy", "| contract | nonconsumer |\n|---|---|\n| `markPaid` READS{Order.id} | payments |\n", "payments"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows := readsConsumerRow(tc.consumer, readsConsumerAllFields)
			if tc.name == "nonconsumer_header_is_legacy" {
				rows += readsConsumerRow("audit", readsConsumerAllFields)
			}
			g := readsConsumerCheck(t, readsConsumerDesign(t, rows, map[string]string{"PaymentHandler": tc.matrix}))
			readsConsumerRequireBlocked(t, g, tc.name)
			if tc.name == "duplicate_owner_columns" && !strings.Contains(strings.Join(g.Errs, "\n"), "PaymentHandler.matrix.md:3") {
				t.Fatalf("finding must address the physical declaring row: %v", g.Errs)
			}
		})
	}
}

// READS members name domain fields in the ubiquitous language, which is what
// v0.6.11 accepted: any trimmed, non-empty member. Multi-word names are not
// malformed, and no non-empty member is ever reported as empty.
func TestReadsConsumerAcceptsMultiWordFieldNames(t *testing.T) {
	const payload = "Order.id, occurrence time, PAIR KEY, unsupported-element accounting"
	const declared = "READS{Order.id, occurrence time, PAIR KEY, unsupported-element accounting}"
	m := map[string]string{"PaymentHandler": readsConsumerMatrixHeader + readsConsumerDeclaration("payments", declared)}
	g := readsConsumerCheck(t, readsConsumerDesign(t, readsConsumerRow("payments", payload), m))
	if len(g.Errs) != 0 || len(g.Drift) != 0 {
		t.Fatalf("multi-word READS members are the v0.6.11 grammar: errors=%v drift=%v", g.Errs, g.Drift)
	}
	if g.Counts["declared read fields carried"] != 4 {
		t.Fatalf("every declared member must be reconciled against the payload: counts=%v", g.Counts)
	}
}

// A member that is present but not an identifier was reported as empty. The
// remaining rejections keep their own accurate wording.
func TestReadsConsumerEmptyMemberDiagnosticIsAccurate(t *testing.T) {
	m := map[string]string{"PaymentHandler": readsConsumerMatrixHeader + readsConsumerDeclaration("payments", "READS{Order.id, }")}
	g := readsConsumerCheck(t, readsConsumerDesign(t, readsConsumerRow("payments", readsConsumerAllFields), m))
	readsConsumerRequireBlocked(t, g, "an empty member is still rejected")
	joined := strings.Join(g.Errs, "\n")
	if !strings.Contains(joined, "empty READS member") {
		t.Fatalf("an empty member must be named as one: %v", g.Errs)
	}
}

// A contract cell may use the verb. v0.6.11 collected a declaration only from
// a full READS{...} group, so a row narrating what a machine reads was never a
// declaration and armed no obligation.
func TestReadsConsumerBareWordProseIsNotADeclaration(t *testing.T) {
	const prose = "| `recomputeConsumed` | action | (ctx,evt) -> ctx | RESIDUAL, owned by the metering layer: " +
		"the `markPaid` append is the producing layer's; this machine only READS those rows and never appends one | payments | `order-paid-final` |\n"
	m := map[string]string{
		"PaymentHandler": readsConsumerMatrixHeader + readsConsumerDeclaration("payments", "READS{Order.id}") + prose,
	}
	g := readsConsumerCheck(t, readsConsumerDesign(t, readsConsumerRow("payments", "Order.id"), m))
	if len(g.Errs) != 0 || len(g.Drift) != 0 {
		t.Fatalf("prose using the verb READS is not a declaration: errors=%v drift=%v", g.Errs, g.Drift)
	}
}

// A row that does carry a group is judged exactly as before: a second bare
// READS beside it leaves the row ambiguous.
func TestReadsConsumerGroupBesideBareWordStillFails(t *testing.T) {
	m := map[string]string{
		"PaymentHandler": readsConsumerMatrixHeader + readsConsumerDeclaration("payments", "READS{Order.id} and separately READS the ledger"),
	}
	g := readsConsumerCheck(t, readsConsumerDesign(t, readsConsumerRow("payments", readsConsumerAllFields), m))
	readsConsumerRequireBlocked(t, g, "a declaration beside a second bare READS is ambiguous")
}
