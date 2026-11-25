package api

import (
	"context"

	"github.com/RamXX/machinery/examples/crm/internal/fsm"
)

// dealUnits implements every unit the deal table names. Every action writes
// its audit row; stampWinAttempt writes win-attempt. The recordWin actor
// writes the deal_wins row under the closeTimeout deadline the engine gives
// it; a failed or late write is undone and selects onError or after.
func dealUnits() fsm.Units {
	actions := audits(map[string]string{
		"auditQualify":      applied,
		"auditLose":         applied,
		"auditReopen":       applied,
		"auditWin":          applied,
		"auditDealDenied":   refused,
		"auditCloseFailure": failed,
		"auditCloseTimeout": failed,
	})
	actions["stampWinAttempt"] = auditAction("win-attempt", applied)
	return fsm.Units{
		Guards: map[string]func(fsm.Ctx) bool{
			"canWriteDeal": canWriteDeal,
			"canWinDeal":   canWinDeal,
			"isAdmin":      isAdmin,
		},
		Actions: actions,
		Actors: map[string]fsm.Actor{
			"recordWin": {Run: recordWin},
		},
	}
}

func recordWin(ctx context.Context, c fsm.Ctx) error {
	u := unit(c)
	return u.tx.RecordWin(ctx, u.deal.ID, u.deal.AmountCents)
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
