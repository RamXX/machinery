package api

import "github.com/RamXX/machinery/examples/crm/internal/fsm"

// noop is an effect of the skeleton: it does nothing and says so.
var noop = fsm.Action{Simulated: true}

// taskUnits implements every unit the task table names.
func taskUnits() fsm.Units {
	return fsm.Units{
		Guards: map[string]func(fsm.Ctx) bool{
			"canWriteTask": canWriteTask,
		},
		Actions: map[string]fsm.Action{
			"auditStart":      noop,
			"auditComplete":   noop,
			"auditCancel":     noop,
			"auditTaskDenied": noop,
		},
	}
}

// canWriteTask: the caller is the assignee or an Admin.
func canWriteTask(c fsm.Ctx) bool {
	u := unit(c)
	return u.caller.Role == roleAdmin || u.caller.ID == u.task.AssigneeID
}
