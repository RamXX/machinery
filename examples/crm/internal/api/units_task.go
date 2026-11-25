package api

import (
	"strings"

	"github.com/RamXX/machinery/examples/crm/internal/fsm"
)

// Outcomes of an audit entry.
const (
	applied = "applied"
	refused = "refused"
	failed  = "failed"
)

// audits builds the audit actions from their outcomes. Each writes one row
// in the request's transaction, named after the action without its audit
// prefix, in lower case (auditStart writes start).
func audits(outcomes map[string]string) map[string]fsm.Action {
	out := map[string]fsm.Action{}
	for name, outcome := range outcomes {
		out[name] = auditAction(strings.ToLower(strings.TrimPrefix(name, "audit")), outcome)
	}
	return out
}

// auditAction writes one audit row: the caller did action on the record.
func auditAction(action, outcome string) fsm.Action {
	return fsm.Action{Run: func(c fsm.Ctx) error {
		u := unit(c)
		return u.tx.Audit(u.caller.ID, action, u.subject(), outcome)
	}}
}

// subject is how the audit trail names the record of the request.
func (u *unitCtx) subject() string {
	if u.deal != nil {
		return "deal:" + u.deal.ID
	}
	return "task:" + u.task.ID
}

// taskUnits implements every unit the task table names.
func taskUnits() fsm.Units {
	return fsm.Units{
		Guards: map[string]func(fsm.Ctx) bool{
			"canWriteTask": canWriteTask,
		},
		Actions: audits(map[string]string{
			"auditStart":      applied,
			"auditComplete":   applied,
			"auditCancel":     applied,
			"auditTaskDenied": refused,
		}),
	}
}

// canWriteTask: the caller is the assignee or an Admin.
func canWriteTask(c fsm.Ctx) bool {
	u := unit(c)
	return u.caller.Role == roleAdmin || u.caller.ID == u.task.AssigneeID
}
