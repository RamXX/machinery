package authz_test

// Property tests for the four rbac-* invariants (BUILD.md 7.3). Each asserts the
// positive (Allowed) cases that the pure decision must grant, so each is RED
// against the zero-Decision stub.

import (
	"testing"

	"crm/internal/authz"
	"crm/internal/model"
)

// P-rbac-crud-verbs: ReadOnly is Allowed only for read; Admin/Manager/Rep are
// Allowed for create/read/update/delete (subject to scope).
func TestPropRbacCrudVerbs(t *testing.T) {
	a := authz.New()
	writeVerbs := []model.Verb{model.VerbCreate, model.VerbUpdate, model.VerbDelete}

	// ReadOnly: read (in own scope) allowed; every write verb denied.
	if d := a.Authorize(azReadO1, model.VerbRead, model.EntityDeal, azReadO1.ID, "t1"); !d.Allowed {
		t.Errorf("P-rbac-crud-verbs: ReadOnly read in scope must be Allowed")
	}
	for _, v := range append([]model.Verb{}, append(writeVerbs, model.VerbReassign)...) {
		if d := a.Authorize(azReadO1, v, model.EntityDeal, azReadO1.ID, "t1"); d.Allowed {
			t.Errorf("P-rbac-crud-verbs: ReadOnly %s must be Denied", v)
		}
	}
	// Admin: every CRUD verb allowed (scope-independent).
	for _, v := range append([]model.Verb{model.VerbRead}, writeVerbs...) {
		if d := a.Authorize(azAdmin, v, model.EntityDeal, "any", "tZ"); !d.Allowed {
			t.Errorf("P-rbac-crud-verbs: Admin %s must be Allowed", v)
		}
	}
}

// P-rbac-read-visibility: Admin reads all; others read only own or same-team.
func TestPropRbacReadVisibility(t *testing.T) {
	a := authz.New()
	if d := a.Authorize(azAdmin, model.VerbRead, model.EntityDeal, "anyone", "tZ"); !d.Allowed {
		t.Errorf("P-rbac-read-visibility: Admin must read any record")
	}
	if d := a.Authorize(azRepT1, model.VerbRead, model.EntityDeal, azRepT1.ID, "t1"); !d.Allowed {
		t.Errorf("P-rbac-read-visibility: Rep must read own record")
	}
	if d := a.Authorize(azRepT1, model.VerbRead, model.EntityDeal, "r9", "t1"); !d.Allowed {
		t.Errorf("P-rbac-read-visibility: Rep must read same-team record")
	}
	if d := a.Authorize(azRepT1, model.VerbRead, model.EntityDeal, "r9", "t2"); d.Allowed {
		t.Errorf("P-rbac-read-visibility: Rep must NOT read other-team record")
	}
}

// P-rbac-write-scope: Admin any; Manager team members; Rep own; ReadOnly none.
func TestPropRbacWriteScope(t *testing.T) {
	a := authz.New()
	if d := a.Authorize(azAdmin, model.VerbUpdate, model.EntityDeal, "anyone", "tZ"); !d.Allowed {
		t.Errorf("P-rbac-write-scope: Admin may write any record")
	}
	if d := a.Authorize(azMgrT1, model.VerbUpdate, model.EntityDeal, "r9", "t1"); !d.Allowed {
		t.Errorf("P-rbac-write-scope: Manager may write a team member's record")
	}
	if d := a.Authorize(azRepT1, model.VerbUpdate, model.EntityDeal, azRepT1.ID, "t1"); !d.Allowed {
		t.Errorf("P-rbac-write-scope: Rep may write its own record")
	}
	if d := a.Authorize(azRepT1, model.VerbUpdate, model.EntityDeal, "r9", "t1"); d.Allowed {
		t.Errorf("P-rbac-write-scope: Rep may NOT write a not-owned record")
	}
	if d := a.Authorize(azReadO1, model.VerbUpdate, model.EntityDeal, azReadO1.ID, "t1"); d.Allowed {
		t.Errorf("P-rbac-write-scope: ReadOnly may write nothing")
	}
}

// P-rbac-reassign-authority: Allowed only for Admin, or Manager within own team.
func TestPropRbacReassignAuthority(t *testing.T) {
	a := authz.New()
	if d := a.Authorize(azAdmin, model.VerbReassign, model.EntityDeal, "anyone", "tZ"); !d.Allowed {
		t.Errorf("P-rbac-reassign-authority: Admin may reassign any record")
	}
	if d := a.Authorize(azMgrT1, model.VerbReassign, model.EntityDeal, "r9", "t1"); !d.Allowed {
		t.Errorf("P-rbac-reassign-authority: Manager may reassign within own team")
	}
	if d := a.Authorize(azMgrT1, model.VerbReassign, model.EntityDeal, "r9", "t2"); d.Allowed {
		t.Errorf("P-rbac-reassign-authority: Manager may NOT reassign outside own team")
	}
	if d := a.Authorize(azRepT1, model.VerbReassign, model.EntityDeal, azRepT1.ID, "t1"); d.Allowed {
		t.Errorf("P-rbac-reassign-authority: Rep may not reassign")
	}
}
