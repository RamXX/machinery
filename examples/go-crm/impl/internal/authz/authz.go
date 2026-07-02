// Package authz is the pure authorization decision function (BUILD.md 4.6, 5.6).
// It has no I/O and imports only the model kernel. It is the single home of the
// four rbac-* invariants. It is NOT a state machine: it gets a contract spec and
// contract tests (C-AUTHZ-*).
package authz

import "crm/internal/model"

// Decision is the result of an authorization query. Reason is set iff !Allowed
// (BUILD.md 4.6, C-AUTHZ-13).
type Decision struct {
	Allowed bool
	Reason  string
}

// Authorizer decides whether an actor may perform a verb on an entity of a
// given type owned by ownerID within teamID (BUILD.md 4.6).
type Authorizer interface {
	Authorize(actor model.User, verb model.Verb, entity model.EntityType, ownerID, teamID string) Decision
}

// authorizer is the concrete pure implementation.
type authorizer struct{}

// New returns the pure Authorizer.
func New() Authorizer { return authorizer{} }

// Authorize is the pure decision over the four rbac-* invariants
// (rbac-crud-verbs, rbac-read-visibility, rbac-write-scope,
// rbac-reassign-authority). No I/O; deterministic in its inputs (C-AUTHZ-14).
func (authorizer) Authorize(actor model.User, verb model.Verb, entity model.EntityType, ownerID, teamID string) Decision {
	switch verb {
	case model.VerbCreate:
		// rbac-crud-verbs: ReadOnly may not create; Admin/Manager/Rep may.
		if actor.Role == model.RoleReadOnly {
			return deny("rbac-crud-verbs: ReadOnly may not create")
		}
		return allow()

	case model.VerbRead:
		// rbac-read-visibility: Admin reads all; others read own or same-team.
		if actor.Role == model.RoleAdmin {
			return allow()
		}
		if inReadScope(actor, ownerID, teamID) {
			return allow()
		}
		return deny("rbac-read-visibility: record outside the actor's visibility scope")

	case model.VerbUpdate, model.VerbDelete:
		// rbac-crud-verbs + rbac-write-scope.
		switch actor.Role {
		case model.RoleAdmin:
			return allow()
		case model.RoleManager:
			if ownsRecord(actor, ownerID) || inTeam(actor, teamID) {
				return allow()
			}
			return deny("rbac-write-scope: Manager may write only a team member's record")
		case model.RoleRep:
			if ownsRecord(actor, ownerID) {
				return allow()
			}
			return deny("rbac-write-scope: Rep may write only its own record")
		default: // ReadOnly
			return deny("rbac-crud-verbs: ReadOnly may not write")
		}

	case model.VerbReassign:
		// rbac-reassign-authority: only Admin, or Manager within the manager's Team.
		switch actor.Role {
		case model.RoleAdmin:
			return allow()
		case model.RoleManager:
			if inTeam(actor, teamID) {
				return allow()
			}
			return deny("rbac-reassign-authority: Manager may reassign only within its own Team")
		default: // Rep, ReadOnly
			return deny("rbac-reassign-authority: only Admin or an in-team Manager may reassign")
		}

	default:
		return deny("rbac-crud-verbs: unknown verb")
	}
}

// allow is the granted decision (empty Reason, per C-AUTHZ-13).
func allow() Decision { return Decision{Allowed: true} }

// deny is the refused decision carrying a non-empty Reason (C-AUTHZ-13).
func deny(reason string) Decision { return Decision{Allowed: false, Reason: reason} }

// ownsRecord reports whether the actor owns the record.
func ownsRecord(actor model.User, ownerID string) bool {
	return ownerID != "" && ownerID == actor.ID
}

// inTeam reports whether the record's owning team is the actor's team.
func inTeam(actor model.User, teamID string) bool {
	return teamID != "" && teamID == actor.TeamID
}

// inReadScope reports whether the record is inside a non-Admin actor's read
// VisibilityScope: an own record or a same-team record.
func inReadScope(actor model.User, ownerID, teamID string) bool {
	return ownsRecord(actor, ownerID) || inTeam(actor, teamID)
}
