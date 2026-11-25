package api

import (
	"context"

	"github.com/RamXX/machinery/examples/crm/internal/fsm"
)

// dealUnits implements every unit the deal table names. recordWin is
// simulated: it succeeds and does nothing, under the closeTimeout deadline
// the engine gives it.
func dealUnits() fsm.Units {
	return fsm.Units{
		Guards: map[string]func(fsm.Ctx) bool{
			"canWriteDeal": canWriteDeal,
			"canWinDeal":   canWinDeal,
			"isAdmin":      isAdmin,
		},
		Actions: map[string]fsm.Action{
			"auditQualify":      noop,
			"auditLose":         noop,
			"auditReopen":       noop,
			"auditDealDenied":   noop,
			"stampWinAttempt":   noop,
			"auditWin":          noop,
			"auditCloseFailure": noop,
			"auditCloseTimeout": noop,
		},
		Actors: map[string]fsm.Actor{
			"recordWin": {Run: func(context.Context, fsm.Ctx) error { return nil }, Simulated: true},
		},
	}
}

// canWriteDeal: the caller is the owner or an Admin.
func canWriteDeal(c fsm.Ctx) bool { return callerMayWriteDeal(c) }

// canWinDeal holds its two declared clauses.
func canWinDeal(c fsm.Ctx) bool { return callerMayWriteDeal(c) && amountIsPositive(c) }

func callerMayWriteDeal(c fsm.Ctx) bool {
	u := unit(c)
	return u.caller.Role == roleAdmin || u.caller.ID == u.deal.OwnerID
}

func amountIsPositive(c fsm.Ctx) bool { return unit(c).deal.AmountCents > 0 }

func isAdmin(c fsm.Ctx) bool { return unit(c).caller.Role == roleAdmin }
