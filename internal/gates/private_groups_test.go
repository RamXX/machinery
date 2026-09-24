package gates

// A design's own group notation (a group name its own tooling reads) is
// declared once, in the Architecture Contract's private_groups: list. A
// declared private group is skipped by the declaration parser: it is not an
// error, it is not projected, and its members never declare anything. An
// undeclared unknown name stays an error, so a misspelled public group is
// still caught.

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

func privateContract(list string) string {
	return "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\n" + list +
		"boundaries:\n  - id: order.service\n    code: [\"src/order/**\"]\n```\n\n" +
		"## Placement\n\n| component (placement) | persistence |\n|---|---|\n| `Order` | in-memory |\n"
}

func TestParsePrivateGroupsGrammar(t *testing.T) {
	cases := []struct {
		name, yaml string
		want       []string // valid entries honored
		errs       []string // substrings, one per expected error
	}{
		{name: "two names", yaml: "private_groups: [OWNED-BY, AUDIT_TRAIL]", want: []string{"AUDIT_TRAIL", "OWNED-BY"}},
		{name: "block list", yaml: "private_groups:\n  - OWNED-BY\n  - HELD2", want: []string{"HELD2", "OWNED-BY"}},
		{name: "a public group", yaml: "private_groups: [WRITES, OWNED-BY]", want: []string{"OWNED-BY"},
			errs: []string{"private_groups[0] 'WRITES' is a public machinery group"}},
		{name: "retired is public", yaml: "private_groups: [RETIRED]",
			errs: []string{"private_groups[0] 'RETIRED' is a public machinery group"}},
		{name: "lower case", yaml: "private_groups: [owned-by]",
			errs: []string{"private_groups[0] 'owned-by' is not a group name"}},
		{name: "single letter", yaml: "private_groups: [T]",
			errs: []string{"private_groups[0] 'T' is not a group name"}},
		{name: "trailing hyphen", yaml: "private_groups: [OWNED-]",
			errs: []string{"private_groups[0] 'OWNED-' is not a group name"}},
		{name: "braces included", yaml: "private_groups: [\"OWNED-BY{}\"]",
			errs: []string{"private_groups[0] 'OWNED-BY{}' is not a group name"}},
		{name: "duplicate", yaml: "private_groups: [OWNED-BY, OWNED-BY]", want: []string{"OWNED-BY"},
			errs: []string{"private_groups[1] 'OWNED-BY' is listed twice"}},
		{name: "not a list", yaml: "private_groups: OWNED-BY",
			errs: []string{"private_groups must be a non-empty list of group names"}},
		{name: "empty list", yaml: "private_groups: []",
			errs: []string{"private_groups must be a non-empty list of group names"}},
		{name: "non-string entry", yaml: "private_groups: [{name: OWNED-BY}]",
			errs: []string{"private_groups[0] is not a string"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := ir.LoadYAML([]byte(tc.yaml))
			if err != nil {
				t.Fatal(err)
			}
			got, errs := parsePrivateGroups(v.AsObject().Get2("private_groups"))
			var names []string
			for n := range got {
				names = append(names, n)
			}
			sort.Strings(names)
			if !reflect.DeepEqual(names, tc.want) {
				t.Fatalf("honored %v, want %v", names, tc.want)
			}
			if len(errs) != len(tc.errs) {
				t.Fatalf("errors %q, want %q", errs, tc.errs)
			}
			for i, want := range tc.errs {
				if !strings.Contains(errs[i], want) {
					t.Fatalf("error %q does not contain %q", errs[i], want)
				}
			}
		})
	}
}

func TestPrivateGroupsSkippedByTheMatrixParser(t *testing.T) {
	private := map[string]bool{"OWNED-BY": true}
	parse := func(cell string, private map[string]bool) ([]Declaration, []DeclarationError) {
		return ParseMatrixDeclarationsWith("Order.matrix.md", []byte("# Order\n\n"+declHeader+declRow("persistOrder", cell)), private)
	}
	t.Run("a declared private group is skipped whole", func(t *testing.T) {
		decls, errs := parse("persists. OWNED-BY{Order.status, WRITES{x}} WRITES{Order.total}", private)
		if len(errs) != 0 {
			t.Fatalf("a declared private group must not be an error: %s", errText(errs))
		}
		if len(decls) != 1 || decls[0].Group != GroupWrites || !reflect.DeepEqual(decls[0].Members, []string{"Order.total"}) {
			t.Fatalf("the private group's body must declare nothing: %+v", decls)
		}
	})
	t.Run("an unclosed private group is skipped too", func(t *testing.T) {
		if _, errs := parse("OWNED-BY{Order.status", private); len(errs) != 0 {
			t.Fatalf("a private group is opaque; its shape is its owner's: %s", errText(errs))
		}
	})
	t.Run("a misspelled public group stays an error", func(t *testing.T) {
		_, errs := parse("WRITE{Order.status}", private)
		if !strings.Contains(errText(errs), "unknown declaration group WRITE{...}") {
			t.Fatalf("WRITE must stay an error beside a declared private group: %s", errText(errs))
		}
	})
	t.Run("the same group undeclared is an error", func(t *testing.T) {
		_, errs := parse("OWNED-BY{Order.status}", nil)
		if !strings.Contains(errText(errs), "unknown declaration group OWNED-BY{...}") {
			t.Fatalf("an undeclared private group must fail: %s", errText(errs))
		}
	})
	t.Run("a private list never shadows a public group", func(t *testing.T) {
		decls, errs := parse("WRITES{Order.status, Order.status}", map[string]bool{GroupWrites: true})
		if !strings.Contains(errText(errs), "duplicate member") || len(decls) != 0 {
			t.Fatalf("WRITES must stay parsed even if a caller lists it: decls=%+v errs=%s", decls, errText(errs))
		}
	})
}

func TestUnknownGroupMessageNamesEveryGroupAndTheEscape(t *testing.T) {
	_, errs := parseMatrix(t, declRow("persistOrder", "WRITE{Order.status}"))
	got := errText(errs)
	for _, group := range []string{"CARRIES", "CLAUSES", "ORACLESET", "PRODUCES", "READS", "RETIRED", "USES", "VALUES", "WRITES", "RESERVED", "SUPERSEDES", "payload {...}"} {
		if !strings.Contains(got, group) {
			t.Fatalf("the message omits %s: %s", group, got)
		}
	}
	if !strings.Contains(got, "private_groups:") || !strings.Contains(got, "Architecture Contract") {
		t.Fatalf("the message must name the private_groups escape: %s", got)
	}
}

func TestMaskPrivateGroups(t *testing.T) {
	private := map[string]bool{"OWNED-BY": true}
	cases := []struct{ cell, want string }{
		{"a OWNED-BY{derived: x (y)} b", "a                          b"},
		{"OWNED-BY{payload {f}} VALUES{a}", "                    } VALUES{a}"}, // the group closes at its first brace
		{"OWNED-BY{open to the end", "                        "},
		{"NOT-MINE{x} WRITES{a}", "NOT-MINE{x} WRITES{a}"},
	}
	for _, tc := range cases {
		if got := MaskPrivateGroups(tc.cell, private); got != tc.want {
			t.Fatalf("MaskPrivateGroups(%q) = %q, want %q", tc.cell, got, tc.want)
		}
		if len(MaskPrivateGroups(tc.cell, private)) != len(tc.cell) {
			t.Fatalf("masking must preserve byte offsets")
		}
	}
	if got := MaskPrivateGroups("OWNED-BY{x}", nil); got != "OWNED-BY{x}" {
		t.Fatalf("no private list masks nothing: %q", got)
	}
}

func TestPrivateGroupsInGxAndG2(t *testing.T) {
	rows := "| `persistOrder` | action | persists. OWNED-BY{Order.state} derived: x (inside) | `order-paid-final` |\n"
	t.Run("declared: Gx skips it and counts it", func(t *testing.T) {
		design := shapeDesign(t, "| name | kind | contract (pre / post) | maps to |\n|---|---|---|---|\n"+
			"| `persistOrder` | action | persists. OWNED-BY{Order.state, derived: x} | `order-paid-final` |\n",
			map[string]string{"ARCHITECTURE.md": privateContract("private_groups: [OWNED-BY]\n")})
		g := CheckTraceability(design)
		for _, e := range g.Errs {
			if strings.Contains(e, "OWNED-BY") || strings.Contains(e, "derived") {
				t.Fatalf("a declared private group produced a finding: %s", e)
			}
		}
		if g.Counts["private declaration groups skipped"] != 1 {
			t.Fatalf("the skip must be visible in the checked line: %v", g.Counts)
		}
		if c4 := CheckC4(design); hasFinding(c4.Errs, "private_groups") {
			t.Fatalf("a valid private_groups list failed G2: %v", c4.Errs)
		}
	})
	t.Run("undeclared: Gx fails with the escape named", func(t *testing.T) {
		g := CheckTraceability(valuesDesign(t, rows))
		if !hasErr(g, "unknown declaration group OWNED-BY{...}") || !hasErr(g, "private_groups:") {
			t.Fatalf("an undeclared private group must fail and name the escape: %v", g.Errs)
		}
	})
	t.Run("a public name declared private fails G2", func(t *testing.T) {
		design := shapeDesign(t, "| name | kind | contract (pre / post) | maps to |\n|---|---|---|---|\n",
			map[string]string{"ARCHITECTURE.md": privateContract("private_groups: [WRITES]\n")})
		if c4 := CheckC4(design); !hasFinding(c4.Errs, "Architecture Contract: private_groups[0] 'WRITES' is a public machinery group") {
			t.Fatalf("declaring WRITES private must fail: %v", c4.Errs)
		}
	})
	t.Run("the unknown-key rule still holds", func(t *testing.T) {
		design := shapeDesign(t, "| name | kind | contract (pre / post) | maps to |\n|---|---|---|---|\n",
			map[string]string{"ARCHITECTURE.md": privateContract("private_group: [OWNED-BY]\n")})
		if c4 := CheckC4(design); !hasFinding(c4.Errs, "unsupported key 'private_group'") {
			t.Fatalf("a misspelled key must fail: %v", c4.Errs)
		}
	})
}

func TestPrivateGroupsExported(t *testing.T) {
	design := shapeDesign(t, "| name | kind | contract (pre / post) | maps to |\n|---|---|---|---|\n",
		map[string]string{"ARCHITECTURE.md": privateContract("private_groups: [OWNED-BY, GUARDED]\n")})
	got, err := PrivateGroups(design)
	if err != nil || !reflect.DeepEqual(got, map[string]bool{"OWNED-BY": true, "GUARDED": true}) {
		t.Fatalf("got %v, %v", got, err)
	}
	bad := shapeDesign(t, "| name | kind | contract (pre / post) | maps to |\n|---|---|---|---|\n",
		map[string]string{"ARCHITECTURE.md": privateContract("private_groups: [OWNED-BY, USES]\n")})
	got, err = PrivateGroups(bad)
	if err == nil || !strings.Contains(err.Error(), "'USES' is a public machinery group") || !reflect.DeepEqual(got, map[string]bool{"OWNED-BY": true}) {
		t.Fatalf("an invalid entry must be an error and never honored: %v, %v", got, err)
	}
	none, err := PrivateGroups(valuesDesign(t, ""))
	if err != nil || len(none) != 0 {
		t.Fatalf("a contract without the key declares nothing: %v, %v", none, err)
	}
}
