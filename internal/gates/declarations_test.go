package gates

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const declHeader = "| name | kind | signature | pre / post | maps to |\n|---|---|---|---|---|\n"

func declRow(name, contract string) string {
	return "| `" + name + "` | action | `(ctx) -> ctx` | " + contract + " | - |\n"
}

func parseMatrix(t *testing.T, rows string) ([]Declaration, []DeclarationError) {
	t.Helper()
	return ParseMatrixDeclarations("Order.matrix.md", []byte("# Order\n\n"+declHeader+rows))
}

func errText(errs []DeclarationError) string {
	var out []string
	for _, e := range errs {
		out = append(out, e.String())
	}
	return strings.Join(out, "\n")
}

func TestParseMatrixDeclarationsGrammar(t *testing.T) {
	cases := []struct {
		name     string
		contract string
		group    string
		members  []string
		pairs    []DeclarationPair
		err      string
	}{
		{name: "writes members", contract: "persists. WRITES{Order.status, Order.total}", group: GroupWrites, members: []string{"Order.status", "Order.total"}},
		{name: "writes empty is read-only", contract: "reads only. WRITES{}", group: GroupWrites, members: []string{}},
		{name: "writes whitespace-only is empty", contract: "WRITES{  }", group: GroupWrites, members: []string{}},
		{name: "writes backticked member", contract: "WRITES{`Order.status`}", group: GroupWrites, members: []string{"Order.status"}},
		{name: "writes duplicate", contract: "WRITES{Order.status, Order.status}", err: `WRITES declaration has duplicate member 'Order.status'`},
		{name: "writes empty member", contract: "WRITES{Order.status,}", err: "WRITES declaration contains an empty member"},
		{name: "writes malformed member", contract: "WRITES{Order-status}", err: `WRITES member 'Order-status' is not a dotted identifier`},
		{name: "writes member with whitespace", contract: "WRITES{Order status}", err: `WRITES member 'Order status' is not a dotted identifier`},
		{name: "uses members", contract: "USES{Order.total, line_item_count}", group: GroupUses, members: []string{"Order.total", "line_item_count"}},
		{name: "uses empty", contract: "USES{}", err: "USES{} is empty; a unit that uses nothing declares no USES group"},
		{name: "uses duplicate", contract: "USES{a_b, a_b}", err: `USES declaration has duplicate member 'a_b'`},
		{name: "uses malformed", contract: "USES{9lives}", err: `USES member '9lives' is not a dotted identifier`},
		{name: "produces members", contract: "consumer arm. PRODUCES{Order.markPaid, Payment.capture}", group: GroupProduces, members: []string{"Order.markPaid", "Payment.capture"}},
		{name: "produces empty", contract: "PRODUCES{}", err: "PRODUCES{} is empty; a unit that produces nothing declares no PRODUCES group"},
		{name: "produces duplicate", contract: "PRODUCES{Order.markPaid, Order.markPaid}", err: `PRODUCES declaration has duplicate member 'Order.markPaid'`},
		{name: "produces bare name", contract: "PRODUCES{markPaid}", err: `PRODUCES member 'markPaid' is not one Entity.action identifier`},
		{name: "produces deep path", contract: "PRODUCES{Order.markPaid.now}", err: `PRODUCES member 'Order.markPaid.now' is not one Entity.action identifier`},
		{name: "carries pairs", contract: "CARRIES{column:Order.status, outbox:order.confirmed, sink:auditLog, signal:paged, action:Order.persistOrder}", group: GroupCarries,
			members: []string{"column:Order.status", "outbox:order.confirmed", "sink:auditLog", "signal:paged", "action:Order.persistOrder"},
			pairs:   []DeclarationPair{{"column", "Order.status"}, {"outbox", "order.confirmed"}, {"sink", "auditLog"}, {"signal", "paged"}, {"action", "Order.persistOrder"}}},
		{name: "carries spaced colon", contract: "CARRIES{column : Order.status}", group: GroupCarries, members: []string{"column:Order.status"}, pairs: []DeclarationPair{{"column", "Order.status"}}},
		{name: "carries unknown kind", contract: "CARRIES{queue:order.confirmed}", err: `CARRIES member 'queue:order.confirmed' has unknown kind 'queue'`},
		{name: "carries missing colon", contract: "CARRIES{Order.status}", err: `CARRIES member 'Order.status' has no kind`},
		{name: "carries duplicate pair", contract: "CARRIES{column:Order.status, column:Order.status}", err: `CARRIES declaration has duplicate member 'column:Order.status'`},
		{name: "carries empty group", contract: "CARRIES{}", err: "CARRIES{} is empty"},
		{name: "carries bad target", contract: "CARRIES{column:Order status}", err: `target 'Order status' is not a dotted identifier`},
		{name: "carries fullwidth colon is no colon", contract: "CARRIES{column\uff1aOrder.status}", err: "has no kind"},
		{name: "supersedes in a matrix", contract: "SUPERSEDES{type:OldOrder}", err: "SUPERSEDES{...} belongs on an Architecture Contract row, not in a matrix"},
		{name: "unterminated", contract: "WRITES{Order.status", err: "WRITES{ is never closed in its cell"},
		{name: "nested", contract: "WRITES{Order{status}}", err: "groups do not nest"},
		{name: "two writes groups", contract: "WRITES{Order.status} and WRITES{Order.total}", err: "row carries more than one WRITES group"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decls, errs := parseMatrix(t, declRow("persistOrder", tc.contract))
			if tc.err != "" {
				if !strings.Contains(errText(errs), tc.err) {
					t.Fatalf("want error containing %q, got decls=%+v errs=%s", tc.err, decls, errText(errs))
				}
				if !strings.HasPrefix(errs[0].String(), "Order.matrix.md:5: row 'persistOrder': ") {
					t.Fatalf("finding is not addressed by file, line and row: %s", errs[0])
				}
				return
			}
			if len(errs) != 0 {
				t.Fatalf("unexpected errors: %s", errText(errs))
			}
			if len(decls) != 1 {
				t.Fatalf("want one declaration, got %+v", decls)
			}
			d := decls[0]
			if d.Group != tc.group || !reflect.DeepEqual(d.Members, tc.members) || !reflect.DeepEqual(d.Pairs, tc.pairs) {
				t.Fatalf("got %+v", d)
			}
			if d.File != "Order.matrix.md" || d.Line != 5 || d.Row != "persistOrder" || d.Column != "pre / post" {
				t.Fatalf("declaration location is wrong: %+v", d)
			}
		})
	}
}

func TestParseMatrixDeclarationsRowShapes(t *testing.T) {
	t.Run("different groups on one row", func(t *testing.T) {
		decls, errs := parseMatrix(t, declRow("persistOrder", "WRITES{Order.status} USES{Order.total} CARRIES{outbox:order.confirmed}"))
		if len(errs) != 0 || len(decls) != 3 {
			t.Fatalf("decls=%+v errs=%s", decls, errText(errs))
		}
	})
	t.Run("payload and WRITES on one row", func(t *testing.T) {
		decls, errs := parseMatrix(t, declRow("emitConfirmed", "payload {Order.id, Order.status} WRITES{Order.status}"))
		if len(errs) != 0 || len(decls) != 1 || decls[0].Group != GroupWrites {
			t.Fatalf("decls=%+v errs=%s", decls, errText(errs))
		}
	})
	t.Run("group split across cells is a clean error", func(t *testing.T) {
		decls, errs := parseMatrix(t, "| `persistOrder` | action | `(ctx) -> ctx` | WRITES{Order.status | Order.total} |\n")
		if len(decls) != 0 || len(errs) != 1 || !strings.Contains(errs[0].Message, "cannot span cells or rows") {
			t.Fatalf("decls=%+v errs=%s", decls, errText(errs))
		}
	})
	t.Run("closing brace on the next row does not swallow it", func(t *testing.T) {
		rows := declRow("persistOrder", "WRITES{Order.status,") +
			"| Order.total} | action | `(ctx) -> ctx` | USES{Order.total} | - |\n"
		decls, errs := parseMatrix(t, rows)
		if len(errs) != 1 || errs[0].Line != 5 || !strings.Contains(errs[0].Message, "never closed") {
			t.Fatalf("errs=%s", errText(errs))
		}
		if len(decls) != 1 || decls[0].Line != 6 || decls[0].Group != GroupUses {
			t.Fatalf("the next row was swallowed or lost: %+v", decls)
		}
	})
	t.Run("backticked group in prose is a declaration", func(t *testing.T) {
		decls, errs := parseMatrix(t, declRow("persistOrder", "declares its writes, for example `WRITES{Order.status}`, and nothing else"))
		if len(errs) != 0 || len(decls) != 1 || decls[0].Members[0] != "Order.status" {
			t.Fatalf("decls=%+v errs=%s", decls, errText(errs))
		}
	})
	t.Run("CRLF line endings", func(t *testing.T) {
		body := strings.ReplaceAll("# Order\n\n"+declHeader+declRow("persistOrder", "WRITES{Order.status}")+declRow("guardX", "USES{Order.total}"), "\n", "\r\n")
		decls, errs := ParseMatrixDeclarations("Order.matrix.md", []byte(body))
		if len(errs) != 0 || len(decls) != 2 || decls[0].Members[0] != "Order.status" || decls[1].Line != 6 {
			t.Fatalf("decls=%+v errs=%s", decls, errText(errs))
		}
	})
	t.Run("fenced example is not a row", func(t *testing.T) {
		body := "# Order\n\n```\n" + declHeader + declRow("x", "WRITES{bad member}") + "```\n"
		decls, errs := ParseMatrixDeclarations("Order.matrix.md", []byte(body))
		if len(errs) != 0 || len(decls) != 0 {
			t.Fatalf("decls=%+v errs=%s", decls, errText(errs))
		}
	})
	t.Run("escaped pipe stays inside the cell", func(t *testing.T) {
		decls, errs := parseMatrix(t, "| `persistOrder` | actor | `(id) -> ok \\| err` | WRITES{Order.status} | - |\n")
		if len(errs) != 0 || len(decls) != 1 || decls[0].Column != "pre / post" {
			t.Fatalf("decls=%+v errs=%s", decls, errText(errs))
		}
	})
}

func TestUnknownDeclarationGroup(t *testing.T) {
	cases := []struct {
		name, cell string
		unknown    string // "" means no finding
	}{
		{"consumer private group", "MACHINE-WRITTEN{Order.status}", "MACHINE-WRITTEN"},
		{"misspelled group", "WRITE{Order.status}", "WRITE"},
		{"known groups", "CLAUSES{a-b} READS{x} VALUES{a, b} ORACLESET{o} WRITES{}", ""},
		{"retired clauses group", "CLAUSES{resolved-task, applied-record} RETIRED{sop-coverage}", ""},
		{"named VALUES", "VALUES reason{a, b}", ""},
		{"camel-case type literal", "`RankedConstituent{rank, ticker}`", ""},
		{"lower-case literal", "`err{ErrUnavailable,ErrConflict}`", ""},
		{"prose braces", "every event {start, complete}", ""},
		{"lower-case payload", "payload {Order.id}", ""},
		{"fullwidth brace lookalike", "WRITES\uff5bOrder.status\uff5d", ""},
		{"ornament brace lookalike", "FOO\u2774x\u2775", ""},
		{"single capital", "T{x}", ""},
		{"embedded in a word", "xFOO{y}", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, errs := parseMatrix(t, declRow("persistOrder", tc.cell))
			got := errText(errs)
			if tc.unknown == "" {
				if strings.Contains(got, "unknown declaration group") {
					t.Fatalf("false positive: %s", got)
				}
				return
			}
			if !strings.Contains(got, "unknown declaration group "+tc.unknown+"{...}") || !strings.Contains(got, `row 'persistOrder'`) {
				t.Fatalf("want unknown group %s naming the row, got %s", tc.unknown, got)
			}
		})
	}
}

// TestUnknownDeclarationGroupAcceptsExamples runs the parser over every
// bundled matrix: the closed vocabulary must not reject a shipped design.
func TestUnknownDeclarationGroupAcceptsExamples(t *testing.T) {
	var paths []string
	err := filepath.WalkDir(filepath.Join(repoRoot(), "examples"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".matrix.md") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) < 20 {
		t.Fatalf("expected the whole example corpus, found %d matrices", len(paths))
	}
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, errs := ParseMatrixDeclarations(filepath.Base(path), body); len(errs) != 0 {
			t.Errorf("%s: %s", path, errText(errs))
		}
	}
}

func TestParseContractDeclarations(t *testing.T) {
	arch := "# A\n\n## 2. Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n  - id: order.service\n    kind: container\n    # SUPERSEDES{type:LegacyOrder}\n```\n\n" +
		"## 7. Events\n\n| event | producer | consumer | payload | delivery |\n|---|---|---|---|---|\n" +
		"| `order.confirmed` | orderSvc | shippingSvc | `Order.id` SUPERSEDES{type:OrderPlacedV1} | at-least-once |\n"
	cases := []struct {
		name, arch string
		want       []Declaration
		err        string
	}{
		{name: "table row and fence line", arch: arch, want: []Declaration{
			{File: "ARCHITECTURE.md", Line: 10, Row: "order.service", Column: "contract", Group: GroupSupersedes, Members: []string{"type:LegacyOrder"}, Pairs: []DeclarationPair{{"type", "LegacyOrder"}}},
			{File: "ARCHITECTURE.md", Line: 17, Row: "order.confirmed", Group: GroupSupersedes, Members: []string{"type:OrderPlacedV1"}, Pairs: []DeclarationPair{{"type", "OrderPlacedV1"}}},
		}},
		{name: "unknown kind", arch: "| type | note |\n|---|---|\n| `Order` | SUPERSEDES{entity:OldOrder} |\n", err: `unknown kind 'entity'; kinds are type`},
		{name: "empty", arch: "| type | note |\n|---|---|\n| `Order` | SUPERSEDES{} |\n", err: "SUPERSEDES{} is empty"},
		{name: "duplicate", arch: "| type | note |\n|---|---|\n| `Order` | SUPERSEDES{type:A, type:A} |\n", err: "duplicate member"},
		{name: "two on a row", arch: "| type | note |\n|---|---|\n| `Order` | SUPERSEDES{type:A} | SUPERSEDES{type:B} |\n", err: "more than one SUPERSEDES group"},
		{name: "unterminated", arch: "| type | note |\n|---|---|\n| `Order` | SUPERSEDES{type:A |\n", err: "never closed"},
		{name: "other groups are not read here", arch: "| type | note |\n|---|---|\n| `Order` | FOO{x} WRITES{bad member} |\n"},
		{name: "CRLF", arch: strings.ReplaceAll("| type | note |\n|---|---|\n| `Order` | SUPERSEDES{type:A} |\n", "\n", "\r\n"), want: []Declaration{
			{File: "ARCHITECTURE.md", Line: 3, Row: "Order", Group: GroupSupersedes, Members: []string{"type:A"}, Pairs: []DeclarationPair{{"type", "A"}}},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decls, errs := ParseContractDeclarations("ARCHITECTURE.md", []byte(tc.arch))
			if tc.err != "" {
				if !strings.Contains(errText(errs), tc.err) {
					t.Fatalf("want %q, got %s", tc.err, errText(errs))
				}
				return
			}
			if len(errs) != 0 {
				t.Fatalf("unexpected: %s", errText(errs))
			}
			if !reflect.DeepEqual(decls, tc.want) {
				t.Fatalf("got  %+v\nwant %+v", decls, tc.want)
			}
		})
	}
}

// TestDeclarationFindingsSurfaceInGx pins where the parse-time findings
// land: Gx-trace, as ERRORs in the file:line shape the VALUES findings use.
func TestDeclarationFindingsSurfaceInGx(t *testing.T) {
	design := valuesDesign(t, "| `persistOrder` | action | WRITES{Order.status, Order.status} MACHINE-WRITTEN{Order.state} | `order-paid-final` |\n")
	g := CheckTraceability(design)
	if !hasErr(g, `Order.matrix.md:3: row 'persistOrder': WRITES declaration has duplicate member 'Order.status'`) {
		t.Fatalf("duplicate WRITES member not surfaced: %v", g.Errs)
	}
	if !hasErr(g, `Order.matrix.md:3: row 'persistOrder': unknown declaration group MACHINE-WRITTEN{...}`) {
		t.Fatalf("unknown group not surfaced: %v", g.Errs)
	}
	clean := CheckTraceability(valuesDesign(t, "| `persistOrder` | action | WRITES{Order.state} | `order-paid-final` |\n"))
	for _, e := range clean.Errs {
		if strings.Contains(e, "WRITES") || strings.Contains(e, "declaration group") {
			t.Fatalf("a well-formed declaration produced a finding: %s", e)
		}
	}
	if clean.Counts["declaration groups parsed"] != 1 {
		t.Fatalf("declaration count: %v", clean.Counts)
	}
}
