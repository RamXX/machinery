package alloy

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func genAll(t *testing.T, domain, annotation string) (string, string, Stats) {
	t.Helper()
	dir := t.TempDir()
	dm := write(t, dir, "domain.modelith.yaml", domain)
	an := write(t, dir, "policy.relational.yaml", annotation)
	als, oracle, stats, err := GenerateAll(dm, an)
	if err != nil {
		t.Fatal(err)
	}
	return als, oracle, stats
}

func TestOracleGoCRM(t *testing.T) {
	_, oracle, stats, err := GenerateAll(
		filepath.Join(repoRoot(), "examples/go-crm/design/domain.modelith.yaml"),
		filepath.Join(repoRoot(), "examples/go-crm/design/formal/policy.relational.yaml"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if stats.OracleRows != 70 {
		t.Errorf("oracle rows = %d, want 70", stats.OracleRows)
	}
	for _, want := range []string{
		// the create grants row for the read-only role
		"| create | ReadOnly | - | - | deny | rbac-crud-verbs |",
		// the case the experiment found: forbidden by the amended invariants
		"| update | Manager | own-teamless | - | unreachable | single-team, manager-has-team |",
		// the amended reassign target rule, both directions
		"| reassign | Manager | teammate | target-teammate | allow | rbac-reassign-authority, task-assignee-visible |",
		"| reassign | Manager | teammate | target-outsider | deny | rbac-reassign-authority, task-assignee-visible |",
		// admin reassign is unconstrained
		"| reassign | Admin | outsider | any | allow | rbac-reassign-authority, task-assignee-visible |",
		"DO NOT EDIT",
	} {
		if !strings.Contains(oracle, want) {
			t.Errorf("oracle missing %q", want)
		}
	}
	// unreachable rows never carry a verdict a test could assert
	for _, line := range strings.Split(oracle, "\n") {
		if !strings.HasPrefix(line, "| O-AUTHZ-") {
			continue
		}
		if strings.Contains(line, "unreachable") && !strings.Contains(line, "single-team, manager-has-team") {
			t.Errorf("unreachable row without its forbidding invariants: %s", line)
		}
	}
}

// stableRow extracts "stable id" -> full row for content assertions.
func stableRows(oracle string) map[string]string {
	re := regexp.MustCompile(`\| (O-AUTHZ-\d+) \| (AUTHZ-[0-9a-f]+) \|(.*)\|`)
	out := map[string]string{}
	for _, line := range strings.Split(oracle, "\n") {
		if m := re.FindStringSubmatch(line); m != nil {
			out[m[2]] = m[3]
		}
	}
	return out
}

// The stable id hashes the case (verb, role, owner case, target), never the
// verdict: flipping a rule must flip the expectation UNDER THE SAME ID, so a
// revision diff names exactly which cases changed behavior.
func TestOracleStableIDsSurviveRuleChange(t *testing.T) {
	ann := strings.Replace(miniAnnotation, ", invariant: u-team", "", 1)
	loose := ann + `  - invariant: u-team
    reassign:
      scope: {A: all, B: team}
      target: {A: any, B: any}
`
	tight := ann + `  - invariant: u-team
    reassign:
      scope: {A: all, B: team}
      target: {A: any, B: team}
`
	_, o1, _ := genAll(t, miniDomain, loose)
	_, o2, _ := genAll(t, miniDomain, tight)
	r1, r2 := stableRows(o1), stableRows(o2)

	// find B's teammate-owner reassign rows in the tight version
	var flipped int
	for id, row2 := range r2 {
		if !strings.Contains(row2, "reassign | B |") {
			continue
		}
		row1, ok := r1[id]
		if !ok {
			continue // target expansion differs between any and team shapes
		}
		if row1 != row2 {
			flipped++
			if !strings.Contains(row1, "allow") && !strings.Contains(row2, "deny") {
				t.Errorf("id %s changed in an unexpected direction:\n  loose: %s\n  tight: %s", id, row1, row2)
			}
		}
	}
	// A's rows (target any in both) must be byte-identical under identical ids
	for id, row2 := range r2 {
		if strings.Contains(row2, "reassign | A |") {
			if r1[id] != row2 {
				t.Errorf("unchanged rule produced a changed row for %s: %q vs %q", id, r1[id], row2)
			}
		}
	}
}

func TestOracleTeamlessDesign(t *testing.T) {
	domain := `
enums:
  R:
    values:
      - {name: A, definition: a}
      - {name: B, definition: b}
entities:
  U:
    attributes:
      - {name: role, type: R}
  Rec:
    relationships:
      - {entity: U, cardinality: "n:1"}
invariants:
  - {id: top-read, statement: s}
`
	annotation := `
subjects:
  entity: U
  role_attr: role
resources: [Rec]
rules:
  - invariant: top-read
    verbs: [read]
    scope: {A: all, B: own}
`
	_, oracle, stats := genAll(t, domain, annotation)
	if strings.Contains(oracle, "own-teamed") || strings.Contains(oracle, "teammate") {
		t.Error("teamless design must use the own/other vocabulary")
	}
	if !strings.Contains(oracle, "| read | B | own | - | allow |") ||
		!strings.Contains(oracle, "| read | B | other | - | deny |") {
		t.Errorf("teamless rows wrong:\n%s", oracle)
	}
	if stats.OracleRows != 4 { // 2 roles x 2 cases, no grants rule so no create rows
		t.Errorf("rows = %d, want 4", stats.OracleRows)
	}
}

func TestOracleMembershipOneUnreachable(t *testing.T) {
	annotation := strings.Replace(miniAnnotation, "membership: lone", "membership: one", 1)
	_, oracle, _ := genAll(t, miniDomain, annotation)
	// with membership one, every own-teamless row is unreachable for every role
	for _, line := range strings.Split(oracle, "\n") {
		if !strings.HasPrefix(line, "| O-AUTHZ-") {
			continue
		}
		if strings.Contains(line, "own-teamless") && !strings.Contains(line, "unreachable") {
			t.Errorf("own-teamless must be unreachable under membership one: %s", line)
		}
	}
}

func TestOracleDeterminism(t *testing.T) {
	_, o1, _ := genAll(t, miniDomain, miniAnnotation)
	_, o2, _ := genAll(t, miniDomain, miniAnnotation)
	if o1 != o2 {
		t.Error("oracle generation is not deterministic")
	}
}
