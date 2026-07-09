package authz

// The authorization oracle test: every reachable row of the generated
// decision table (design/formal/Policy.oracle.md, written by `machinery
// alloy` from the domain model + policy annotation) is asserted against the
// pure Authorizer. This is what holds the CODE to the policy: a design
// revision regenerates the oracle, flips expectations under stable ids, and
// this test names exactly which cases the implementation no longer satisfies.
//
// Each abstract owner case expands to every concrete input variant that is
// policy-equivalent under the scope algebra (the oracle header defines the
// vocabulary), and every row runs against all five resource entity types,
// because the policy treats resources alike.

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"crm/internal/model"
)

const oraclePath = "../../../design/formal/Policy.oracle.md"

type oracleRow struct {
	testID, stableID, verb, role, ownerCase, target, expectation string
}

func loadOracle(t *testing.T) []oracleRow {
	t.Helper()
	f, err := os.Open(filepath.FromSlash(oraclePath))
	if err != nil {
		t.Fatalf("authorization oracle not found (run 'machinery alloy design/'): %v", err)
	}
	defer f.Close()
	var rows []oracleRow
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "| O-AUTHZ-") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) < 8 {
			t.Fatalf("malformed oracle row: %s", line)
		}
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		rows = append(rows, oracleRow{
			testID: cells[0], stableID: cells[1], verb: cells[2], role: cells[3],
			ownerCase: cells[4], target: cells[5], expectation: cells[6],
		})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("oracle carries no decision rows; an empty oracle is a failure, not a pass")
	}
	return rows
}

// vector is one concrete instantiation of an abstract owner case.
type vector struct {
	label   string
	actor   model.User
	ownerID string
	teamID  string // the record owner's team
}

// vectorsFor expands an owner case into every policy-equivalent concrete
// variant (the oracle header's "concrete variants" list).
func vectorsFor(ownerCase string, role model.UserRole) []vector {
	mk := func(label, actorTeam, ownerID, teamID string) vector {
		return vector{label: label, actor: model.User{ID: "u-actor", Role: role, TeamID: actorTeam}, ownerID: ownerID, teamID: teamID}
	}
	switch ownerCase {
	case "own-teamed":
		return []vector{mk("actor owns, teamed", "t1", "u-actor", "t1")}
	case "own-teamless":
		return []vector{mk("actor owns, teamless", "", "u-actor", "")}
	case "teammate":
		return []vector{mk("teammate-owned", "t1", "u-owner", "t1")}
	case "outsider":
		return []vector{
			mk("owner in another team", "t1", "u-owner", "t2"),
			mk("owner teamless", "t1", "u-owner", ""),
			mk("actor teamless", "", "u-owner", "t2"),
		}
	case "-": // create rows: no object
		return []vector{mk("no object", "t1", "", "")}
	}
	return nil
}

// targetsFor expands a target case into concrete new owners. The actor's
// team is "t1" in every teamed vector, so a teammate target shares it.
func targetsFor(target string) []model.User {
	switch target {
	case "target-teammate":
		return []model.User{{ID: "u-target", TeamID: "t1"}}
	case "target-outsider":
		return []model.User{
			{ID: "u-target", TeamID: "t9"},
			{ID: "u-target", TeamID: ""},
		}
	default: // "any" or "-": the verdict must hold for every target
		return []model.User{
			{ID: "u-target", TeamID: "t1"},
			{ID: "u-target", TeamID: "t9"},
			{ID: "u-target", TeamID: ""},
		}
	}
}

var oracleVerbs = map[string]model.Verb{
	"create": model.VerbCreate, "read": model.VerbRead,
	"update": model.VerbUpdate, "delete": model.VerbDelete,
}

var resourceEntities = []model.EntityType{
	model.EntityAccount, model.EntityContact, model.EntityDeal,
	model.EntityTask, model.EntityActivity,
}

// TestOracleConformance is P-authz-oracle: the pure Authorizer agrees with
// every reachable row of the generated decision table.
func TestOracleConformance(t *testing.T) {
	a := New()
	skipped := 0
	for _, row := range loadOracle(t) {
		if row.expectation == "unreachable" {
			// forbidden by the domain invariants; construction is refused by
			// the write discipline and authz behavior is unspecified
			skipped++
			continue
		}
		want := row.expectation == "allow"
		role := model.UserRole(row.role)
		t.Run(row.stableID+"_"+row.verb+"_"+row.role+"_"+row.ownerCase+"_"+row.target, func(t *testing.T) {
			for _, v := range vectorsFor(row.ownerCase, role) {
				if v.actor.TeamID == "" && row.target == "target-teammate" {
					continue // a teamless actor has no teammates to target
				}
				for _, entity := range resourceEntities {
					if row.verb == "reassign" {
						for _, target := range targetsFor(row.target) {
							got := a.AuthorizeReassign(v.actor, entity, v.ownerID, v.teamID, target)
							if got.Allowed != want {
								t.Errorf("%s [%s, target team %q, %s]: got allowed=%v, want %v (reason %q)",
									row.testID, v.label, target.TeamID, entity, got.Allowed, want, got.Reason)
							}
							if !got.Allowed && got.Reason == "" {
								t.Errorf("%s: denial without a reason", row.testID)
							}
						}
						continue
					}
					got := a.Authorize(v.actor, oracleVerbs[row.verb], entity, v.ownerID, v.teamID)
					if got.Allowed != want {
						t.Errorf("%s [%s, %s]: got allowed=%v, want %v (reason %q)",
							row.testID, v.label, entity, got.Allowed, want, got.Reason)
					}
					if !got.Allowed && got.Reason == "" {
						t.Errorf("%s: denial without a reason", row.testID)
					}
				}
			}
		})
	}
	if skipped == 0 {
		t.Log("no unreachable rows; every case asserted")
	} else {
		t.Logf("%d unreachable row(s) skipped (forbidden by domain invariants)", skipped)
	}
}
