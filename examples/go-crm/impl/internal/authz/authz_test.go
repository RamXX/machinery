package authz_test

// Authorizer contract tests (BUILD.md 7.2 C-AUTHZ-01..14). The Authorizer is
// pure: these tests perform no I/O. Against the scaffolding stub (which returns
// the zero Decision) every "Allowed" expectation and every "Reason set" / "Reason
// empty" expectation is RED.

import (
	"testing"

	"crm/internal/authz"
	"crm/internal/model"
)

var (
	azAdmin  = model.User{ID: "a", Role: model.RoleAdmin, Status: model.StatusActive}
	azMgrT1  = model.User{ID: "m", Role: model.RoleManager, Status: model.StatusActive, TeamID: "t1"}
	azRepT1  = model.User{ID: "r1", Role: model.RoleRep, Status: model.StatusActive, TeamID: "t1"}
	azReadO1 = model.User{ID: "o", Role: model.RoleReadOnly, Status: model.StatusActive, TeamID: "t1"}
)

type authzCase struct {
	id          string
	actor       model.User
	verb        model.Verb
	entity      model.EntityType
	ownerID     string
	teamID      string
	wantAllowed bool
}

func authzCases() []authzCase {
	D := model.EntityDeal
	return []authzCase{
		// C-AUTHZ-01: ReadOnly + create -> Denied.
		{"C-AUTHZ-01", azReadO1, model.VerbCreate, D, "x", "t1", false},
		// C-AUTHZ-02: Rep/Manager/Admin + create -> Allowed.
		{"C-AUTHZ-02_rep", azRepT1, model.VerbCreate, D, "x", "t1", true},
		{"C-AUTHZ-02_mgr", azMgrT1, model.VerbCreate, D, "x", "t1", true},
		{"C-AUTHZ-02_admin", azAdmin, model.VerbCreate, D, "x", "t1", true},
		// C-AUTHZ-03: ReadOnly read in scope -> Allowed; update/delete -> Denied.
		{"C-AUTHZ-03_readOwn", azReadO1, model.VerbRead, D, azReadO1.ID, "t1", true},
		{"C-AUTHZ-03_update", azReadO1, model.VerbUpdate, D, azReadO1.ID, "t1", false},
		{"C-AUTHZ-03_delete", azReadO1, model.VerbDelete, D, azReadO1.ID, "t1", false},
		// C-AUTHZ-04: Admin + read any record -> Allowed.
		{"C-AUTHZ-04", azAdmin, model.VerbRead, D, "someone", "tZ", true},
		// C-AUTHZ-05: Rep + read own record -> Allowed.
		{"C-AUTHZ-05", azRepT1, model.VerbRead, D, azRepT1.ID, "t1", true},
		// C-AUTHZ-06: Rep + read same-team record -> Allowed.
		{"C-AUTHZ-06", azRepT1, model.VerbRead, D, "r9", "t1", true},
		// C-AUTHZ-07: Rep + read other-team record -> Denied.
		{"C-AUTHZ-07", azRepT1, model.VerbRead, D, "r9", "t2", false},
		// C-AUTHZ-08: Manager + update/delete team member's record -> Allowed.
		{"C-AUTHZ-08_update", azMgrT1, model.VerbUpdate, D, "r9", "t1", true},
		{"C-AUTHZ-08_delete", azMgrT1, model.VerbDelete, D, "r9", "t1", true},
		// C-AUTHZ-09: Rep + update/delete a not-owned record -> Denied.
		{"C-AUTHZ-09_update", azRepT1, model.VerbUpdate, D, "r9", "t1", false},
		{"C-AUTHZ-09_delete", azRepT1, model.VerbDelete, D, "r9", "t1", false},
		// C-AUTHZ-10: Manager + reassign within own team -> Allowed.
		{"C-AUTHZ-10", azMgrT1, model.VerbReassign, D, "r9", "t1", true},
		// C-AUTHZ-11: Rep/ReadOnly + reassign -> Denied.
		{"C-AUTHZ-11_rep", azRepT1, model.VerbReassign, D, azRepT1.ID, "t1", false},
		{"C-AUTHZ-11_readonly", azReadO1, model.VerbReassign, D, azReadO1.ID, "t1", false},
		// C-AUTHZ-12: Admin + reassign any record -> Allowed.
		{"C-AUTHZ-12", azAdmin, model.VerbReassign, D, "someone", "tZ", true},
	}
}

func TestAuthzContract(t *testing.T) {
	a := authz.New()
	for _, tc := range authzCases() {
		t.Run(tc.id, func(t *testing.T) {
			dec := a.Authorize(tc.actor, tc.verb, tc.entity, tc.ownerID, tc.teamID)
			if dec.Allowed != tc.wantAllowed {
				t.Errorf("%s: Allowed = %v, want %v", tc.id, dec.Allowed, tc.wantAllowed)
			}
			// C-AUTHZ-13 embedded: Reason set iff !Allowed.
			if !tc.wantAllowed && dec.Reason == "" {
				t.Errorf("%s: denied decision must carry a non-empty Reason", tc.id)
			}
			if tc.wantAllowed && dec.Reason != "" {
				t.Errorf("%s: allowed decision must have empty Reason, got %q", tc.id, dec.Reason)
			}
		})
	}
}

// C-AUTHZ-13: a denied decision has a non-empty Reason; an allowed decision has
// an empty Reason.
func TestAuthzReasonContract(t *testing.T) {
	a := authz.New()
	denied := a.Authorize(azReadO1, model.VerbCreate, model.EntityDeal, "x", "t1")
	if denied.Allowed {
		t.Fatalf("C-AUTHZ-13: ReadOnly+create must be denied")
	}
	if denied.Reason == "" {
		t.Errorf("C-AUTHZ-13: denied Decision.Reason must be non-empty")
	}
	allowed := a.Authorize(azAdmin, model.VerbCreate, model.EntityDeal, "x", "t1")
	if !allowed.Allowed {
		t.Errorf("C-AUTHZ-13: Admin+create must be allowed")
	}
	if allowed.Reason != "" {
		t.Errorf("C-AUTHZ-13: allowed Decision.Reason must be empty, got %q", allowed.Reason)
	}
}

// C-AUTHZ-14: same inputs twice -> identical Decision (purity/determinism). The
// Allowed assertion also pins that the pure function actually decides.
func TestAuthzPurity(t *testing.T) {
	a := authz.New()
	d1 := a.Authorize(azAdmin, model.VerbCreate, model.EntityDeal, "x", "t1")
	d2 := a.Authorize(azAdmin, model.VerbCreate, model.EntityDeal, "x", "t1")
	if d1 != d2 {
		t.Errorf("C-AUTHZ-14: Authorize is not deterministic: %+v vs %+v", d1, d2)
	}
	if !d1.Allowed {
		t.Errorf("C-AUTHZ-14: Admin+create must be Allowed")
	}
}
