package main

// MAC-hgz1: migrate the eight shipped example attestation documents to the
// accepted v2 plan/current/historical classification with truthful BUILD
// conformance obligations. This test holds the migrated state to the closed
// contract using the real CLI, the real filesystem, and the real example
// trees -- no mocks of the CLI, parser, or process output.
//
// RED (unmigrated state) fails for intended semantic reasons:
//   - every example document is v1 without explicit kinds (GV forces
//     GV_MISSING_IMPLEMENTATION_SUBJECT on all twelve behavioral rows);
//   - the BUILD conformance/context obligations are wrong (orders/payments/
//     surreal lack the wholesale committed-FSM parsing obligation; go-crm
//     asserts whole-design greenfield, pins a stale x/crypto version, and
//     does not describe the accepted parser-backed suite);
//   - the go-crm current review row and its implementation subject do not
//     exist.
// The generated plan/current controls at the end are separately valid and
// pass in RED, proving the harness itself is not the failure.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

// migratedExample is one expected migrated document: the exact row sequence
// with per-row kind, preserved or renewed provenance, and the preserved
// covers membership (paths in their committed order; hash freshness is
// proven by the real gate runs below, never asserted by literal here).
type migratedRow struct {
	claim    string
	kind     string
	attestor string
	date     string
	note     string // "" = whatever the row carries; otherwise a prefix check
	covers   []string
}

const (
	origReview    = "Machinery production-readiness review"
	origCodexProd = "Codex production-readiness review"
	origCodexCRM  = "Codex CRM design review"
	origAccept    = "Codex release acceptance review"
	coordinator   = "coordinator"
	reviewDate    = "2026-09-06"
	// reReviewPrefix marks rows whose covered bytes changed (BUILD.md edits
	// here, or the accepted MAC-lhu5 portfolio repair) and were therefore
	// re-reviewed by the coordinator for this story.
	reReviewPrefix = "Re-reviewed 2026-09-06 over the corrected BUILD"
	// crmBumpDate/crmBumpPrefix pin the two go-crm rows covering BUILD.md.
	// They moved off reviewDate when the dependabot bump of
	// golang.org/x/crypto (0.55.0 to 0.56.0, then 0.56.0 to 0.57.0) staled the
	// pin text BUILD.md quotes from the authoritative impl/go.mod, forcing a
	// re-review of the corrected document and of the bumped implementation root.
	crmBumpDate        = "2026-09-23"
	crmBumpPrefix      = "Re-reviewed 2026-09-23 over the corrected BUILD"
	portfolioRenewal   = "Renews the 2026-09-02 Codex deterministic design review"
	nextGateAttestor   = "Codex lane/next-gates"
	nextGateDate       = "2026-09-22"
	nextGateG2Prefix   = "Re-reviewed 2026-09-22 over AUTHORIZATION.md"
	nextGateG3Prefix   = "Re-reviewed 2026-09-22 over the MarketDataFeed derived fact waiver"
	legacyCRMReview    = "Re-reviewed 2026-09-22 over current ARCHITECTURE.md"
	fulfillmentReview  = "Re-reviewed 2026-09-22 over current BUILD.md"
	planOnlyWarnPrefix = "plan only; current implementation review missing"
)

var (
	g2Claims = []string{
		"g2.action-ownership", "g2.interface-contract-rightness", "g2.placement-rightness",
		"g2.adoption-closure-discovery", "g2.event-contract-completeness", "g2.nfr-content",
	}
	g3Claims = []string{
		"g3.guard-semantics", "g3.invariant-enforcement", "g3.residual-transitions", "g3.event-redelivery",
	}
)

func machineCovers(names ...string) []string {
	var out []string
	for _, n := range names {
		out = append(out, "machines/"+n+".machine.json", "machines/"+n+".matrix.md")
	}
	return out
}

func planRows(claims []string, attestor, date string, covers []string) []migratedRow {
	out := make([]migratedRow, 0, len(claims))
	for _, c := range claims {
		out = append(out, migratedRow{claim: c, kind: "plan", attestor: attestor, date: date, covers: covers})
	}
	return out
}

// expectedMigration is the reviewed target state for all eight documents.
func expectedMigration() map[string][]migratedRow {
	ordersPack := []string{
		"pack/OrdersContract.cfg", "pack/OrdersContract.machine.json", "pack/OrdersContract.tla",
		"pack/domain.modelith.yaml", "pack/events.md", "pack/pack.yaml",
	}
	paymentsPack := []string{
		"pack/PaymentsContract.cfg", "pack/PaymentsContract.machine.json", "pack/PaymentsContract.tla",
		"pack/domain.modelith.yaml", "pack/events.md", "pack/pack.yaml",
	}
	portfolioBuild := []string{
		"BUILD.md", "BUILD/M0-walking-skeleton.md", "BUILD/M1-run-pipeline.md", "BUILD/M2-feed-breaker.md",
		"BUILD/M3-optimizer.md", "BUILD/M4-portfolio-review.md", "BUILD/M5-reference-operations.md",
	}
	fulfillmentMachines := machineCovers("FulfillmentSaga", "Order", "OutboxMessage", "Payment", "Reservation", "Shipment")
	crmMachines := machineCovers("CommandExecution", "Deal", "Session", "Task", "User")
	portfolioMachines := machineCovers("MarketDataFeed", "Portfolio", "RecommendationRun", "ReferenceDataCommand")
	acceptance := []string{
		"acceptance/M0.yaml", "acceptance/M1.yaml", "acceptance/M2.yaml",
		"acceptance/M3.yaml", "acceptance/M4.yaml", "acceptance/M5.yaml",
	}

	// checkout-split/orders: 14 rows. Design rows and the pack row keep their
	// original provenance and covers; the three BUILD-covering behavioral
	// rows are re-reviewed plans over the corrected BUILD.md.
	orders := planRows(g2Claims, nextGateAttestor, nextGateDate, []string{"ARCHITECTURE.md", "AUTHORIZATION.md"})
	for i := range orders {
		orders[i].note = nextGateG2Prefix
	}
	orders = append(orders, planRows(g3Claims, origReview, "2026-09-02",
		machineCovers("Order"))...)
	orders = append(orders,
		migratedRow{claim: "gt.conformance-test-shape", kind: "plan", attestor: coordinator, date: reviewDate, note: reReviewPrefix, covers: []string{"BUILD.md"}},
		migratedRow{claim: "g4.zero-context", kind: "plan", attestor: coordinator, date: reviewDate, note: reReviewPrefix, covers: []string{"BUILD.md"}},
		migratedRow{claim: "g4.standin-coverage", kind: "plan", attestor: coordinator, date: reviewDate, note: reReviewPrefix, covers: []string{"BUILD.md"}},
		migratedRow{claim: "g4.pack-event-discipline", kind: "plan", attestor: origReview, date: "2026-09-02", covers: ordersPack},
	)

	// checkout-split/payments: mirror of orders.
	payments := planRows(g2Claims, nextGateAttestor, nextGateDate, []string{"ARCHITECTURE.md", "AUTHORIZATION.md"})
	for i := range payments {
		payments[i].note = nextGateG2Prefix
	}
	payments = append(payments, planRows(g3Claims, origReview, "2026-09-02",
		machineCovers("Payment"))...)
	payments = append(payments,
		migratedRow{claim: "gt.conformance-test-shape", kind: "plan", attestor: coordinator, date: reviewDate, note: reReviewPrefix, covers: []string{"BUILD.md"}},
		migratedRow{claim: "g4.zero-context", kind: "plan", attestor: coordinator, date: reviewDate, note: reReviewPrefix, covers: []string{"BUILD.md"}},
		migratedRow{claim: "g4.standin-coverage", kind: "plan", attestor: coordinator, date: reviewDate, note: reReviewPrefix, covers: []string{"BUILD.md"}},
		migratedRow{claim: "g4.pack-event-discipline", kind: "plan", attestor: origReview, date: "2026-09-02", covers: paymentsPack},
	)

	// checkout-split/parent: BUILD.md is read-only; every row keeps original
	// provenance, the gt row stays an explicitly reviewed delegated-conformance
	// plan over unchanged bytes.
	parent := planRows(g2Claims, origReview, "2026-09-02", []string{"ARCHITECTURE.md"})
	parent = append(parent,
		migratedRow{claim: "gt.conformance-test-shape", kind: "plan", attestor: origReview, date: "2026-09-02", covers: []string{"BUILD.md"}},
		migratedRow{claim: "g4.zero-context", kind: "plan", attestor: origReview, date: "2026-09-02", covers: []string{"BUILD.md"}},
	)

	// fulfillment: the two BUILD-covering rows were re-reviewed against current
	// bytes after the frozen-test amendment; they remain plan judgments.
	fulfillment := planRows(g2Claims, nextGateAttestor, nextGateDate, []string{"ARCHITECTURE.md", "AUTHORIZATION.md"})
	for i := range fulfillment {
		fulfillment[i].note = nextGateG2Prefix
	}
	fulfillment = append(fulfillment, planRows(g3Claims, origCodexProd, "2026-09-02", fulfillmentMachines)...)
	fulfillment = append(fulfillment,
		migratedRow{claim: "gt.conformance-test-shape", kind: "plan", attestor: nextGateAttestor, date: nextGateDate, note: fulfillmentReview, covers: []string{"BUILD.md"}},
		migratedRow{claim: "g4.zero-context", kind: "plan", attestor: nextGateAttestor, date: nextGateDate, note: fulfillmentReview, covers: []string{"BUILD.md"}},
	)

	// go-crm: six Architecture Contract rows were re-reviewed over the current
	// bytes after the testoracle ignore amendment. Its gt row is a
	// real current review over examples/go-crm/impl; the zero-context row is a
	// re-reviewed plan over the corrected BUILD.md; historical acceptance
	// stays historical with original provenance.
	goCrm := planRows(g2Claims, nextGateAttestor, nextGateDate, []string{"ARCHITECTURE.md"})
	for i := range goCrm {
		goCrm[i].note = legacyCRMReview
	}
	goCrm = append(goCrm, planRows(g3Claims, origCodexCRM, "2026-09-02", crmMachines)...)
	goCrm = append(goCrm,
		migratedRow{claim: "gt.conformance-test-shape", kind: "current", attestor: coordinator, date: crmBumpDate, covers: []string{"BUILD.md"}},
		migratedRow{claim: "g4.zero-context", kind: "plan", attestor: coordinator, date: crmBumpDate, note: crmBumpPrefix, covers: []string{"BUILD.md"}},
		migratedRow{claim: "ga.review-quality", kind: "historical", attestor: origAccept, date: "2026-09-03", covers: acceptance},
	)

	// pii-flow: BUILD.md is read-only; all rows keep original provenance and
	// the behavioral gt stays prospective plan over unchanged bytes.
	piiFlow := planRows(g2Claims, origReview, "2026-09-02", []string{"ARCHITECTURE.md"})
	piiFlow = append(piiFlow, planRows(g3Claims, origReview, "2026-09-02", machineCovers("DataSubject"))...)
	piiFlow = append(piiFlow,
		migratedRow{claim: "gt.conformance-test-shape", kind: "plan", attestor: origReview, date: "2026-09-02", covers: []string{"BUILD.md"}},
		migratedRow{claim: "g4.zero-context", kind: "plan", attestor: origReview, date: "2026-09-02", covers: []string{"BUILD.md"}},
	)

	// portfolio-engine: no implementation exists; all twelve rows are plans
	// renewed by review over the accepted MAC-lhu5-revised subjects.
	portfolio := planRows(g2Claims, nextGateAttestor, nextGateDate, []string{"ARCHITECTURE.md", "AUTHORIZATION.md"})
	for i := range portfolio {
		portfolio[i].note = nextGateG2Prefix
	}
	portfolio = append(portfolio, planRows(g3Claims, nextGateAttestor, nextGateDate, portfolioMachines)...)
	for i := len(portfolio) - 4; i < len(portfolio); i++ {
		portfolio[i].note = nextGateG3Prefix
	}
	portfolio = append(portfolio,
		migratedRow{claim: "gt.conformance-test-shape", kind: "plan", attestor: coordinator, date: reviewDate, note: portfolioRenewal, covers: portfolioBuild},
		migratedRow{claim: "g4.zero-context", kind: "plan", attestor: coordinator, date: reviewDate, note: portfolioRenewal, covers: portfolioBuild},
	)

	// surreal-crm: design-only rebuild; gt/zero re-reviewed plans over the
	// corrected BUILD.md, design rows keep original provenance.
	surreal := planRows(g2Claims, origCodexCRM, "2026-09-02", []string{"ARCHITECTURE.md"})
	surreal = append(surreal, planRows(g3Claims, origCodexCRM, "2026-09-02", crmMachines)...)
	surreal = append(surreal,
		migratedRow{claim: "gt.conformance-test-shape", kind: "plan", attestor: coordinator, date: reviewDate, note: reReviewPrefix, covers: []string{"BUILD.md"}},
		migratedRow{claim: "g4.zero-context", kind: "plan", attestor: coordinator, date: reviewDate, note: reReviewPrefix, covers: []string{"BUILD.md"}},
	)

	return map[string][]migratedRow{
		"checkout-split/orders":   orders,
		"checkout-split/parent":   parent,
		"checkout-split/payments": payments,
		"fulfillment":             fulfillment,
		"go-crm":                  goCrm,
		"pii-flow":                piiFlow,
		"portfolio-engine":        portfolio,
		"surreal-crm":             surreal,
	}
}

func TestExampleAttestationMigration(t *testing.T) {
	root := repoRootDir(t)
	expected := expectedMigration()
	total := 0
	for _, rows := range expected {
		total += len(rows)
	}
	if total != 97 {
		t.Fatalf("expected-migration table must cover the full 97-row inventory, got %d", total)
	}

	t.Run("SchemaAndProvenance", func(t *testing.T) {
		for ex, want := range expected {
			t.Run(ex, func(t *testing.T) {
				path := filepath.Join(root, "examples", ex, "design", "attestations.yaml")
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				value, err := ir.LoadYAML(raw)
				if err != nil {
					t.Fatalf("invalid YAML: %v", err)
				}
				obj := value.AsObject()
				if obj == nil {
					t.Fatal("attestations.yaml is not a mapping")
				}
				if got := obj.Get2("attestation_version"); got == nil || got.Kind != ir.KindNumber || string(got.AsNumber()) != "2" {
					t.Errorf("attestation_version must be the integer 2 (v1 documents must be migrated deliberately, not grandfathered)")
				}
				list := obj.Get2("attestations")
				if list == nil || list.Kind != ir.KindArray {
					t.Fatal("attestations must be a list")
				}
				rows := list.AsArray()
				if len(rows) != len(want) {
					t.Fatalf("row count drift: want %d rows, got %d", len(want), len(rows))
				}
				for i, item := range rows {
					row := item.AsObject()
					if row == nil {
						t.Fatalf("row %d is not a mapping", i)
					}
					w := want[i]
					if got := row.GetString("claim"); got != w.claim {
						t.Errorf("row %d claim order drift: want %s, got %s", i, w.claim, got)
						continue
					}
					if got := row.GetString("kind"); got != w.kind {
						t.Errorf("%s: kind must be %q, got %q", w.claim, w.kind, got)
					}
					if got := row.GetString("attestor"); got != w.attestor {
						t.Errorf("%s: attestor must be %q, got %q (renewed hashes are not renewed reviews and unchanged rows keep their original attribution)", w.claim, w.attestor, got)
					}
					if got := row.GetString("date"); got != w.date {
						t.Errorf("%s: date must be %q, got %q", w.claim, w.date, got)
					}
					if w.note != "" {
						if got := row.GetString("note"); !strings.HasPrefix(got, w.note) {
							t.Errorf("%s: renewed row must carry the re-review note, got %q", w.claim, got)
						}
					}
					covers := row.Get2("covers")
					if covers == nil || covers.Kind != ir.KindArray {
						t.Errorf("%s: covers must be a list", w.claim)
						continue
					}
					gotCovers := []string{}
					for _, c := range covers.AsArray() {
						co := c.AsObject()
						if co == nil {
							t.Fatalf("%s: cover is not a mapping", w.claim)
						}
						gotCovers = append(gotCovers, co.GetString("path"))
					}
					if strings.Join(gotCovers, "|") != strings.Join(w.covers, "|") {
						t.Errorf("%s: covers membership/order drift:\n want %v\n got  %v", w.claim, w.covers, gotCovers)
					}
				}
				// The go-crm current row must bind the real implementation
				// root under the closed manifest schema.
				if ex == "go-crm" {
					for _, item := range rows {
						row := item.AsObject()
						if row.GetString("claim") != "gt.conformance-test-shape" {
							continue
						}
						impl := row.Get2("implementation")
						if impl == nil || impl.AsObject() == nil {
							t.Errorf("go-crm gt current row requires a complete implementation subject")
							continue
						}
						m := impl.AsObject()
						if got := m.GetString("root"); got != "../impl" {
							t.Errorf("implementation root must be the reviewed ../impl locator, got %q", got)
						}
						if got := m.GetString("policy"); got != "full-root-v1" {
							t.Errorf("implementation policy must be full-root-v1, got %q", got)
						}
						if !regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(m.GetString("hash")) {
							t.Errorf("implementation hash must be sha256:<64 lowercase hex>, got %q", m.GetString("hash"))
						}
						entries := m.Get2("entries")
						if entries == nil || entries.Kind != ir.KindArray || len(entries.AsArray()) < 2 {
							t.Errorf("implementation entries must carry the complete full-root inventory")
						}
					}
				}
			})
		}
	})

	t.Run("BuildObligations", func(t *testing.T) {
		read := func(parts ...string) string {
			t.Helper()
			body, err := os.ReadFile(filepath.Join(append([]string{root, "examples"}, parts...)...))
			if err != nil {
				t.Fatal(err)
			}
			// Prose wraps across lines; match on normalized whitespace so the
			// obligation is checked semantically, not by line breaks.
			return strings.Join(strings.Fields(string(body)), " ")
		}
		mustContain := func(file, body string, wants ...string) {
			t.Helper()
			for _, w := range wants {
				if !strings.Contains(body, w) {
					t.Errorf("%s missing required obligation %q", file, w)
				}
			}
		}
		// AC1: orders/payments/surreal explicitly require wholesale
		// committed-FSM parsing, row/guard-input reconciliation, next state
		// and expected actions including entry/exit semantics.
		for _, c := range []struct{ file, machine string }{
			{"checkout-split/orders/design/BUILD.md", "Order"},
			{"checkout-split/payments/design/BUILD.md", "Payment"},
		} {
			body := read(c.file)
			mustContain(c.file, body,
				"Wholesale conformance obligation",
				"parses the committed",
				"reconciles the row against the machine's declared guard inputs",
				"next state and the complete ordered expected-actions list",
				"entry and exit semantics",
				"not evidence that any suite currently runs",
			)
		}
		surreal := read("surreal-crm/design/BUILD.md")
		mustContain("surreal-crm/design/BUILD.md", surreal,
			"Wholesale conformance obligation",
			"One wholesale conformance test per machine parses the committed",
			"reconciles the row against the machine's",
			"next state and the complete ordered expected-actions list",
			"entry and exit semantics",
			"not evidence that any suite currently runs",
		)
		// AC1: fulfillment's sound prospective obligation stays intact.
		fulfillment := read("fulfillment/design/BUILD.md")
		mustContain("fulfillment/design/BUILD.md", fulfillment,
			"wholesale conformance test that parses the committed oracle table",
			"asserts both the target state and the complete ordered expected-actions list",
		)
		// AC1: portfolio keeps packet-alone handoff and does not invent an M3
		// FSM (non-regression over the accepted repaired root).
		portfolio := read("portfolio-engine/design/BUILD.md")
		mustContain("portfolio-engine/design/BUILD.md", portfolio,
			"The zero-context claim applies independently to each packet.",
		)
		if m3 := portfolio[strings.Index(portfolio, "M3 - Optimizer slice"):]; !strings.Contains(m3, "optimizer invariants") || strings.Contains(m3, "M3 machine") {
			t.Errorf("portfolio M3 must remain the pure optimizer slice, not an invented FSM")
		}
		// AC1/AC2: go-crm describes the actual parser-backed suite with
		// bounded evidence and resolves the migration/toolchain drift.
		goCrm := read("go-crm/design/BUILD.md")
		mustContain("go-crm/design/BUILD.md", goCrm,
			"Wholesale conformance suite (accepted implementation)",
			"impl/internal/testoracle",
			"197 committed transition rows",
			"does not authenticate test execution",
			"declared rebuild",
			"no production migration has run",
		)
		if strings.Contains(goCrm, "greenfield design") {
			t.Errorf("go-crm BUILD.md must not assert whole-design greenfield; the design declares a rebuild/prototype migration with disposable nonproduction seed data")
		}
		if strings.Contains(goCrm, "v0.53.0") {
			t.Errorf("go-crm BUILD.md pins the stale x/crypto v0.53.0; the authoritative pin is whatever impl/go.mod carries")
		}
		modBody, err := os.ReadFile(filepath.Join(root, "examples", "go-crm", "impl", "go.mod"))
		if err != nil {
			t.Fatal(err)
		}
		pin := regexp.MustCompile(`(?m)^[\t ]*golang\.org/x/crypto (v[0-9.]+)`).FindSubmatch(modBody)
		if pin == nil {
			t.Fatal("impl/go.mod does not pin golang.org/x/crypto")
		}
		if !strings.Contains(goCrm, string(pin[1])) {
			t.Errorf("go-crm BUILD.md must align the x/crypto pin with authoritative impl/go.mod %s", pin[1])
		}
	})

	t.Run("DesignOnlyPlanWarnings", func(t *testing.T) {
		for _, ex := range []string{
			"checkout-split/orders", "checkout-split/parent", "checkout-split/payments",
			"fulfillment", "pii-flow", "portfolio-engine", "surreal-crm",
		} {
			t.Run(ex, func(t *testing.T) {
				design := filepath.Join(root, "examples", ex, "design")
				out, _, code := runBin(t, "check", design, "--gate", "gv")
				if code != 0 {
					t.Fatalf("migrated design-only plan document must pass gv (warnings allowed); exit %d:\n%s", code, out)
				}
				if !strings.Contains(out, planOnlyWarnPrefix) {
					t.Fatalf("behavioral plan rows must carry the missing-current warning")
				}
				if strings.Contains(out, "GV_MISSING_IMPLEMENTATION_SUBJECT") || strings.Contains(out, "Legacy v1") {
					t.Fatalf("v1 legacy diagnostics must be gone after migration:\n%s", out)
				}
				out2, _, code2 := runBin(t, "check", design, "--gate", "gv", "--warnings-as-errors")
				if code2 != 1 {
					t.Fatalf("--warnings-as-errors must block the design-only plan for exactly the missing-current warning; exit %d:\n%s", code2, out2)
				}
				if !strings.Contains(out2, planOnlyWarnPrefix) || !strings.Contains(out2, "warnings are errors") {
					t.Fatalf("warning promotion must block on the intended missing-current reason:\n%s", out2)
				}
			})
		}
	})

	t.Run("GoCrmCurrentReview", func(t *testing.T) {
		design := filepath.Join(root, "examples", "go-crm", "design")
		impl := filepath.Join(root, "examples", "go-crm", "impl")
		out, _, code := runBin(t, "check", design, "--gate", "gv", "--impl", impl)
		if code != 0 {
			t.Fatalf("reviewed unchanged current scope must pass gv with --impl; exit %d:\n%s", code, out)
		}
		for _, want := range []string{"1 current implementation reviews", "1 historical review records", "11 plan judgments"} {
			if !strings.Contains(out, want) {
				t.Fatalf("gv counts missing %q:\n%s", want, out)
			}
		}
		if strings.Contains(out, planOnlyWarnPrefix) {
			t.Fatalf("go-crm has no design-only behavioral plan; missing-current warning must be absent:\n%s", out)
		}
		out, _, code = runBin(t, "check", design, "--gate", "gv")
		if code != 1 || !strings.Contains(out, "GV_IMPL_REQUIRED") {
			t.Fatalf("checking the current row without --impl must fail GV_IMPL_REQUIRED; exit %d:\n%s", code, out)
		}
	})

	t.Run("GoCrmCurrentInvalidation", func(t *testing.T) {
		base := t.TempDir()
		copyDirInto(t, filepath.Join(root, "examples", "go-crm"), base)
		design := filepath.Join(base, "design")
		impl := filepath.Join(base, "impl")
		gv := func() (string, int) {
			out, _, code := runBin(t, "check", design, "--gate", "gv", "--impl", impl)
			return out, code
		}
		expectBlocking := func(category string) {
			t.Helper()
			out, code := gv()
			if code != 1 || !strings.Contains(out, category) {
				t.Fatalf("expected blocking category %s (exit %d):\n%s", category, code, out)
			}
		}
		// Safe control: the untouched copied scope passes.
		if out, code := gv(); code != 0 {
			t.Fatalf("safe control must pass; exit %d:\n%s", code, out)
		}
		// Real test mutation invalidates the current scope content.
		testFile := filepath.Join(impl, "internal", "testoracle", "fsm_field_membership_test.go")
		body, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(testFile, append(body, []byte("\n// mutation\n")...), 0o644); err != nil {
			t.Fatal(err)
		}
		expectBlocking("GV_STALE_CONTENT")
		if err := os.WriteFile(testFile, body, 0o644); err != nil {
			t.Fatal(err)
		}
		// Handler mutation invalidates the current scope content.
		handler := filepath.Join(impl, "internal", "repo", "repo.go")
		hbody, err := os.ReadFile(handler)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(handler, append(hbody, []byte("\n// mutation\n")...), 0o644); err != nil {
			t.Fatal(err)
		}
		expectBlocking("GV_STALE_CONTENT")
		if err := os.WriteFile(handler, hbody, 0o644); err != nil {
			t.Fatal(err)
		}
		// Config mutation invalidates the current scope content.
		modFile := filepath.Join(impl, "go.mod")
		mbody, err := os.ReadFile(modFile)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(modFile, append(mbody, []byte("\n// mutation\n")...), 0o644); err != nil {
			t.Fatal(err)
		}
		expectBlocking("GV_STALE_CONTENT")
		if err := os.WriteFile(modFile, mbody, 0o644); err != nil {
			t.Fatal(err)
		}
		// Added, removed, and renamed paths are stale scope inventory.
		added := filepath.Join(impl, "internal", "model", "mutation.go")
		if err := os.WriteFile(added, []byte("package model\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		expectBlocking("GV_SCOPE_INVENTORY")
		if err := os.Remove(added); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(filepath.Join(impl, "internal", "model", "entities.go"), filepath.Join(impl, "internal", "model", "entities_renamed.go")); err != nil {
			t.Fatal(err)
		}
		expectBlocking("GV_SCOPE_INVENTORY")
		if err := os.Rename(filepath.Join(impl, "internal", "model", "entities_renamed.go"), filepath.Join(impl, "internal", "model", "entities.go")); err != nil {
			t.Fatal(err)
		}
		removed := filepath.Join(impl, "internal", "model", "errors.go")
		orig, err := os.ReadFile(removed)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(removed); err != nil {
			t.Fatal(err)
		}
		expectBlocking("GV_SCOPE_INVENTORY")
		if err := os.WriteFile(removed, orig, 0o644); err != nil {
			t.Fatal(err)
		}
		// Safe control again after restoration.
		if out, code := gv(); code != 0 {
			t.Fatalf("restored scope must pass again; exit %d:\n%s", code, out)
		}
	})

	t.Run("StaleDesignCovers", func(t *testing.T) {
		base := t.TempDir()
		copyDirInto(t, filepath.Join(root, "examples", "checkout-split", "orders", "design"), base)
		build := filepath.Join(base, "BUILD.md")
		body, err := os.ReadFile(build)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(build, append(body, []byte("\nmutated after attestation\n")...), 0o644); err != nil {
			t.Fatal(err)
		}
		out, _, code := runBin(t, "check", base, "--gate", "gv")
		if code != 1 {
			t.Fatalf("stale covers must block; exit %d:\n%s", code, out)
		}
		for _, want := range []string{
			"gt.conformance-test-shape is STALE: BUILD.md changed",
			"g4.zero-context is STALE: BUILD.md changed",
			"g4.standin-coverage is STALE: BUILD.md changed",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("stale output missing %q:\n%s", want, out)
			}
		}
		if strings.Contains(out, "GV_MISSING_IMPLEMENTATION_SUBJECT") {
			t.Fatalf("migrated plan rows must not raise legacy missing-subject errors:\n%s", out)
		}
	})

	t.Run("WrongKindNegatives", func(t *testing.T) {
		// A plan-class claim cannot be current.
		base := t.TempDir()
		copyDirInto(t, filepath.Join(root, "examples", "pii-flow", "design"), base)
		path := filepath.Join(base, "attestations.yaml")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		mutated := strings.Replace(string(body),
			"  - claim: g2.nfr-content\n    kind: plan\n",
			"  - claim: g2.nfr-content\n    kind: current\n", 1)
		if mutated == string(body) {
			t.Fatalf("kind flip fixture not applied; document not migrated yet")
		}
		if err := os.WriteFile(path, []byte(mutated), 0o644); err != nil {
			t.Fatal(err)
		}
		out, _, code := runBin(t, "check", base, "--gate", "gv")
		if code != 1 || !strings.Contains(out, "GV_KIND") {
			t.Fatalf("wrong kind must block as GV_KIND; exit %d:\n%s", code, out)
		}
		// A behavioral claim marked current without a subject is incomplete.
		base2 := t.TempDir()
		copyDirInto(t, filepath.Join(root, "examples", "pii-flow", "design"), base2)
		path2 := filepath.Join(base2, "attestations.yaml")
		body2, err := os.ReadFile(path2)
		if err != nil {
			t.Fatal(err)
		}
		mutated2 := strings.Replace(string(body2),
			"  - claim: gt.conformance-test-shape\n    kind: plan\n",
			"  - claim: gt.conformance-test-shape\n    kind: current\n", 1)
		if mutated2 == string(body2) {
			t.Fatalf("gt kind flip fixture not applied; document not migrated yet")
		}
		if err := os.WriteFile(path2, []byte(mutated2), 0o644); err != nil {
			t.Fatal(err)
		}
		out2, _, code2 := runBin(t, "check", base2, "--gate", "gv")
		if code2 != 1 || !strings.Contains(out2, "GV_MISSING_IMPLEMENTATION_SUBJECT: gt.conformance-test-shape requires a complete implementation subject") {
			t.Fatalf("current without subject must block as GV_MISSING_IMPLEMENTATION_SUBJECT; exit %d:\n%s", code2, out2)
		}
	})

	t.Run("GeneratedControls", func(t *testing.T) {
		// Plan generation control over a real, unmutated design copy.
		planCopy := t.TempDir()
		copyDirInto(t, filepath.Join(root, "examples", "pii-flow", "design"), planCopy)
		out, _, code := runBin(t, "attest", "--design", planCopy,
			"--claim", "g2.nfr-content", "--kind", "plan",
			"--attestor", "control-reviewer", "--date", "2026-09-06")
		if code != 0 {
			t.Fatalf("plan generation control must succeed; exit %d", code)
		}
		value, err := ir.LoadYAML([]byte(out))
		if err != nil {
			t.Fatalf("generated plan document invalid: %v", err)
		}
		rows := value.AsObject().Get2("attestations").AsArray()
		if len(rows) != 1 || rows[0].AsObject().GetString("kind") != "plan" {
			t.Fatalf("generated plan control must be one plan row:\n%s", out)
		}
		// Current generation control over the real implementation copy.
		curBase := t.TempDir()
		copyDirInto(t, filepath.Join(root, "examples", "go-crm"), curBase)
		out2, _, code2 := runBin(t, "attest",
			"--design", filepath.Join(curBase, "design"),
			"--claim", "gt.conformance-test-shape", "--kind", "current",
			"--impl", filepath.Join(curBase, "impl"),
			"--attestor", "control-reviewer", "--date", "2026-09-06")
		if code2 != 0 {
			t.Fatalf("current generation control must succeed; exit %d", code2)
		}
		value2, err := ir.LoadYAML([]byte(out2))
		if err != nil {
			t.Fatalf("generated current document invalid: %v", err)
		}
		row := value2.AsObject().Get2("attestations").AsArray()[0].AsObject()
		if row.GetString("kind") != "current" {
			t.Fatalf("generated current control must be a current row:\n%s", out2)
		}
		manifest := row.Get2("implementation").AsObject()
		if manifest.GetString("root") != "../impl" || manifest.GetString("policy") != "full-root-v1" {
			t.Fatalf("generated manifest must bind ../impl full-root-v1:\n%s", out2)
		}
		if entries := manifest.Get2("entries").AsArray(); len(entries) < 2 {
			t.Fatalf("generated manifest must carry the full inventory:\n%s", out2)
		}
	})
}
