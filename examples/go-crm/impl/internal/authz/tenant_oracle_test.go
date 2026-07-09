package authz

// The tenant-scoping oracle test: every row of the generated tenant decision
// table (design/formal/Isolation.oracle.md, written by `machinery alloy` from
// the domain model + isolation annotation) is asserted against the pure
// AuthorizeLink function. This is what holds the CODE to the isolation
// invariants: a design revision regenerates the oracle, flips expectations
// under stable ids, and this test names exactly which cross-entity references
// the implementation would now let cross a tenant boundary.
//
// The isolation algebra decides every case from whether the source owner and
// the target owner share a tenant, so each abstract tenant case expands to the
// concrete owner-team pairs that are isolation-equivalent, and every row runs
// against that expansion.

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"crm/internal/model"
)

const tenantOraclePath = "../../../design/formal/Isolation.oracle.md"

type tenantRow struct {
	testID, stableID, reference, tenantCase, expectation string
}

func loadTenantOracle(t *testing.T) []tenantRow {
	t.Helper()
	f, err := os.Open(filepath.FromSlash(tenantOraclePath))
	if err != nil {
		t.Fatalf("tenant oracle not found (run 'machinery alloy design/'): %v", err)
	}
	defer f.Close()
	var rows []tenantRow
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "| O-TENANT-") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) < 6 {
			t.Fatalf("malformed tenant oracle row: %s", line)
		}
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		rows = append(rows, tenantRow{
			testID: cells[0], stableID: cells[1], reference: cells[2],
			tenantCase: cells[3], expectation: cells[4],
		})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("tenant oracle carries no decision rows; an empty oracle is a failure, not a pass")
	}
	return rows
}

// ownerPair is one concrete (source owner, target owner) instantiation of an
// abstract tenant case.
type ownerPair struct {
	label                string
	sourceTeam, destTeam string
}

// pairsFor expands a tenant case into every isolation-equivalent concrete pair
// of owner teams.
func pairsFor(tenantCase string) []ownerPair {
	switch tenantCase {
	case "same-tenant":
		return []ownerPair{{"both in t1", "t1", "t1"}}
	case "cross-tenant":
		return []ownerPair{{"t1 vs t2", "t1", "t2"}}
	case "source-teamless":
		return []ownerPair{
			{"source teamless, dest t1", "", "t1"},
			{"both teamless", "", ""},
		}
	case "target-teamless":
		return []ownerPair{{"source t1, dest teamless", "t1", ""}}
	}
	return nil
}

// TestTenantOracleConformance: the pure AuthorizeLink agrees with every row of
// the generated tenant-scoping decision table.
func TestTenantOracleConformance(t *testing.T) {
	a := New()
	rows := loadTenantOracle(t)
	for _, row := range rows {
		want := row.expectation == "allow"
		t.Run(row.stableID+"_"+strings.ReplaceAll(row.reference, " ", "")+"_"+row.tenantCase, func(t *testing.T) {
			pairs := pairsFor(row.tenantCase)
			if len(pairs) == 0 {
				t.Fatalf("unknown tenant case %q", row.tenantCase)
			}
			for _, p := range pairs {
				src := model.User{ID: "u-src", TeamID: p.sourceTeam}
				dst := model.User{ID: "u-dst", TeamID: p.destTeam}
				got := a.AuthorizeLink(src, dst)
				if got.Allowed != want {
					t.Errorf("%s [%s]: got allowed=%v, want %v (reason %q)",
						row.testID, p.label, got.Allowed, want, got.Reason)
				}
				if !got.Allowed && got.Reason == "" {
					t.Errorf("%s: denial without a reason", row.testID)
				}
			}
		})
	}
}
