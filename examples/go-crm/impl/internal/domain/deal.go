package domain

import (
	"time"

	"crm/internal/authz"
	"crm/internal/model"
)

// DealState is the Deal machine state: the six resting DealStage values plus the
// three persist-overlay states (BUILD.md 5.1, Deal.machine.json).
type DealState string

const (
	DSLead         DealState = "Lead"
	DSQualified    DealState = "Qualified"
	DSProposal     DealState = "Proposal"
	DSNegotiation  DealState = "Negotiation"
	DSWon          DealState = "Won"
	DSLost         DealState = "Lost"
	DSPersisting   DealState = "persisting"
	DSPersistRetry DealState = "persistRetry"
	DSRolledBack   DealState = "rolledBack"
)

// DealEventKind is the trigger for a Deal transition: a domain event, an invoke
// result, an after delay, or an always condition (Deal.machine.json).
type DealEventKind string

const (
	DEvAdvanceStage   DealEventKind = "advanceStage"
	DEvWin            DealEventKind = "win"
	DEvLose           DealEventKind = "lose"
	DEvReopen         DealEventKind = "reopen"
	DEvSaveDone       DealEventKind = "saveDone"       // invoke onDone
	DEvSaveError      DealEventKind = "saveError"      // invoke onError
	DEvPersistTimeout DealEventKind = "persistTimeout" // after persistTimeout
	DEvAlways         DealEventKind = "always"         // persistRetry / rolledBack always
	DEvRetryBackoff   DealEventKind = "retryBackoff"   // after persistRetryBackoff
)

// DealEvent is a single trigger. CloseDate is supplied on win; Err carries the
// classified repo error on saveError.
type DealEvent struct {
	Kind      DealEventKind
	CloseDate *time.Time
	Err       error
}

// Deal is the Deal aggregate (BUILD.md 5.1, 9). State is the explicit machine
// state; the remaining fields are the machine context.
type Deal struct {
	DealID           string
	Title            string
	AmountCents      int64
	State            DealState
	CloseDate        *time.Time
	OwnerID          string
	TeamID           string // owner team, for the rbac-write-scope re-check
	Actor            model.User
	PendingStage     DealState
	PriorStage       DealState
	PendingCloseDate *time.Time
	Retries          int
	LastError        error
	Rejection        string // reason set by record*Denied / record*Rejected

	Authz authz.Authorizer // domain -> authz re-check (allowed edge)
}

// Fire applies an event to the Deal, mirroring the Deal.machine.json transition
// oracle (BUILD.md 7.1 T-DEAL-01..57). It mutates State and context and returns
// the fired actions. The in-memory stage advances to the pending value only
// after a successful save (commitStage on saveDone); any error keeps priorStage.
func (d *Deal) Fire(evt DealEvent) Effect {
	switch d.State {
	case DSLead, DSQualified, DSProposal:
		return d.fireForward(evt)
	case DSNegotiation:
		return d.fireNegotiation(evt)
	case DSWon, DSLost:
		return d.fireTerminal(evt)
	case DSPersisting:
		return d.firePersisting(evt)
	case DSPersistRetry:
		return d.firePersistRetry(evt)
	case DSRolledBack:
		return d.fireRolledBack(evt)
	}
	return Effect{}
}

// fireForward handles the non-terminal forward stages (Lead/Qualified/Proposal).
func (d *Deal) fireForward(evt DealEvent) Effect {
	switch evt.Kind {
	case DEvAdvanceStage:
		if d.guardCanAdvance(evt) {
			d.setPendingAdvance(evt)
			d.State = DSPersisting
			return effect("setPendingAdvance")
		}
		d.recordAdvanceDenied(evt)
		return effect("recordAdvanceDenied")
	case DEvWin:
		if d.guardCanWin(evt) {
			d.setPendingWin(evt)
			d.State = DSPersisting
			return effect("setPendingWin")
		}
		d.recordWinDenied(evt)
		return effect("recordWinDenied")
	case DEvLose:
		if d.guardCanLose(evt) {
			d.setPendingLose(evt)
			d.State = DSPersisting
			return effect("setPendingLose")
		}
		d.recordLoseDenied(evt)
		return effect("recordLoseDenied")
	case DEvReopen:
		d.recordReopenNotTerminal()
		return effect("recordReopenNotTerminal")
	}
	return Effect{}
}

// fireNegotiation handles Negotiation: no forward stage (advance rejected
// structurally by deal-stage-forward), win/lose allowed, reopen not-terminal.
func (d *Deal) fireNegotiation(evt DealEvent) Effect {
	switch evt.Kind {
	case DEvAdvanceStage:
		d.recordAdvanceDenied(evt)
		return effect("recordAdvanceDenied")
	case DEvWin:
		if d.guardCanWin(evt) {
			d.setPendingWin(evt)
			d.State = DSPersisting
			return effect("setPendingWin")
		}
		d.recordWinDenied(evt)
		return effect("recordWinDenied")
	case DEvLose:
		if d.guardCanLose(evt) {
			d.setPendingLose(evt)
			d.State = DSPersisting
			return effect("setPendingLose")
		}
		d.recordLoseDenied(evt)
		return effect("recordLoseDenied")
	case DEvReopen:
		d.recordReopenNotTerminal()
		return effect("recordReopenNotTerminal")
	}
	return Effect{}
}

// fireTerminal handles Won/Lost: only an authorized reopen advances; every other
// event is structurally rejected (deal-terminal).
func (d *Deal) fireTerminal(evt DealEvent) Effect {
	switch evt.Kind {
	case DEvReopen:
		if d.guardCanReopen(evt) {
			d.setPendingReopen(evt)
			d.State = DSPersisting
			return effect("setPendingReopen")
		}
		d.recordReopenDenied(evt)
		return effect("recordReopenDenied")
	case DEvAdvanceStage, DEvWin, DEvLose:
		d.recordTerminalRejected(evt)
		return effect("recordTerminalRejected")
	}
	return Effect{}
}

// firePersisting routes the save outcome: onDone by pendingStage; onError by the
// classified typed error; persistTimeout rolls back.
func (d *Deal) firePersisting(evt DealEvent) Effect {
	switch evt.Kind {
	case DEvSaveDone:
		switch {
		case d.pendingIsQualified(), d.pendingIsProposal(), d.pendingIsNegotiation(), d.pendingIsLost():
			d.commitStage()
			return effect("commitStage")
		case d.pendingIsWon():
			d.commitStage()
			d.commitCloseDate()
			return effect("commitStage", "commitCloseDate")
		default:
			d.recordRoutingError()
			d.State = DSRolledBack
			return effect("recordRoutingError")
		}
	case DEvSaveError:
		switch {
		case d.isErrLocked(evt):
			d.recordError(evt)
			d.State = DSPersistRetry
			return effect("recordError")
		case d.isErrConstraint(evt):
			d.recordConstraint(evt)
			d.State = DSRolledBack
			return effect("recordConstraint")
		case d.isErrDiskFull(evt):
			d.recordDiskFull(evt)
			d.State = DSRolledBack
			return effect("recordDiskFull")
		case d.isErrTimeout(evt):
			d.recordTimeout(evt)
			d.State = DSRolledBack
			return effect("recordTimeout")
		default:
			d.recordUnknownError(evt)
			d.State = DSRolledBack
			return effect("recordUnknownError")
		}
	case DEvPersistTimeout:
		d.recordTimeout(evt)
		d.State = DSRolledBack
		return effect("recordTimeout")
	}
	return Effect{}
}

// firePersistRetry backs off then retries, or gives up when the bound is hit.
func (d *Deal) firePersistRetry(evt DealEvent) Effect {
	switch evt.Kind {
	case DEvAlways:
		if d.retriesExhausted() {
			d.recordRetriesExhausted()
			d.State = DSRolledBack
			return effect("recordRetriesExhausted")
		}
	case DEvRetryBackoff:
		d.incrementRetries()
		d.State = DSPersisting
		return effect("incrementRetries")
	}
	return Effect{}
}

// fireRolledBack atomically returns the aggregate to its pre-transition stage.
func (d *Deal) fireRolledBack(evt DealEvent) Effect {
	if evt.Kind != DEvAlways {
		return Effect{}
	}
	switch {
	case d.priorIsLead():
		d.State = DSLead
	case d.priorIsQualified():
		d.State = DSQualified
	case d.priorIsProposal():
		d.State = DSProposal
	case d.priorIsNegotiation():
		d.State = DSNegotiation
	case d.priorIsWon():
		d.State = DSWon
	case d.priorIsLost():
		d.State = DSLost
	}
	return Effect{}
}

// effect builds an ordered Effect from the fired action names.
func effect(actions ...string) Effect { return Effect{Actions: actions} }

// canWrite is the rbac-write-scope re-check the write guards share.
func (d *Deal) canWrite() bool {
	return d.Authz.Authorize(d.Actor, model.VerbUpdate, model.EntityDeal, d.OwnerID, d.TeamID).Allowed
}

// --- Guards (BUILD.md 5.1 named-unit contract table). ---

func (d *Deal) guardCanAdvance(evt DealEvent) bool {
	_, hasNext := model.NextStage(model.DealStage(d.State))
	return hasNext && d.canWrite() && d.AmountCents >= 0
}
func (d *Deal) guardCanWin(evt DealEvent) bool {
	return evt.CloseDate != nil && d.canWrite() && d.AmountCents >= 0
}
func (d *Deal) guardCanLose(evt DealEvent) bool { return d.canWrite() && d.AmountCents >= 0 }
func (d *Deal) guardCanReopen(evt DealEvent) bool {
	return d.Authz.Authorize(d.Actor, model.VerbReassign, model.EntityDeal, d.OwnerID, d.TeamID).Allowed
}

func (d *Deal) pendingIsQualified() bool   { return d.PendingStage == DSQualified }
func (d *Deal) pendingIsProposal() bool    { return d.PendingStage == DSProposal }
func (d *Deal) pendingIsNegotiation() bool { return d.PendingStage == DSNegotiation }
func (d *Deal) pendingIsWon() bool         { return d.PendingStage == DSWon }
func (d *Deal) pendingIsLost() bool        { return d.PendingStage == DSLost }

func (d *Deal) priorIsLead() bool        { return d.PriorStage == DSLead }
func (d *Deal) priorIsQualified() bool   { return d.PriorStage == DSQualified }
func (d *Deal) priorIsProposal() bool    { return d.PriorStage == DSProposal }
func (d *Deal) priorIsNegotiation() bool { return d.PriorStage == DSNegotiation }
func (d *Deal) priorIsWon() bool         { return d.PriorStage == DSWon }
func (d *Deal) priorIsLost() bool        { return d.PriorStage == DSLost }

func (d *Deal) isErrLocked(evt DealEvent) bool     { return errIs(evt.Err, model.ErrLocked) }
func (d *Deal) isErrConstraint(evt DealEvent) bool { return errIs(evt.Err, model.ErrConstraint) }
func (d *Deal) isErrDiskFull(evt DealEvent) bool   { return errIs(evt.Err, model.ErrDiskFull) }
func (d *Deal) isErrTimeout(evt DealEvent) bool    { return errIs(evt.Err, model.ErrTimeout) }
func (d *Deal) retriesExhausted() bool             { return d.Retries >= maxRetries }

// --- Actions (BUILD.md 5.1). ---

func (d *Deal) setPendingAdvance(evt DealEvent) {
	d.PriorStage = d.State
	next, _ := model.NextStage(model.DealStage(d.State))
	d.PendingStage = DealState(next)
}
func (d *Deal) setPendingWin(evt DealEvent) {
	d.PriorStage = d.State
	d.PendingStage = DSWon
	d.PendingCloseDate = evt.CloseDate
}
func (d *Deal) setPendingLose(evt DealEvent) {
	d.PriorStage = d.State
	d.PendingStage = DSLost
}
func (d *Deal) setPendingReopen(evt DealEvent) {
	d.PriorStage = d.State
	d.PendingStage = DSNegotiation
}
func (d *Deal) commitStage()      { d.State = d.PendingStage }
func (d *Deal) commitCloseDate()  { d.CloseDate = d.PendingCloseDate }
func (d *Deal) incrementRetries() { d.Retries++ }

func (d *Deal) recordError(evt DealEvent)        { d.LastError = evt.Err }
func (d *Deal) recordConstraint(evt DealEvent)   { d.LastError = evt.Err }
func (d *Deal) recordDiskFull(evt DealEvent)     { d.LastError = evt.Err }
func (d *Deal) recordTimeout(evt DealEvent)      { d.LastError = evt.Err }
func (d *Deal) recordUnknownError(evt DealEvent) { d.LastError = evt.Err }
func (d *Deal) recordRetriesExhausted()          { d.LastError = model.ErrLocked }
func (d *Deal) recordRoutingError()              { d.Rejection = "deal: unroutable pending stage" }
func (d *Deal) recordAdvanceDenied(evt DealEvent) {
	d.Rejection = "deal-stage-forward/rbac-write-scope/deal-amount-nonneg"
}
func (d *Deal) recordWinDenied(evt DealEvent) {
	d.Rejection = "deal-won-has-closedate/rbac-write-scope/deal-amount-nonneg"
}
func (d *Deal) recordLoseDenied(evt DealEvent)   { d.Rejection = "rbac-write-scope/deal-amount-nonneg" }
func (d *Deal) recordReopenDenied(evt DealEvent) { d.Rejection = "rbac-reassign-authority" }
func (d *Deal) recordReopenNotTerminal()         { d.Rejection = "deal-terminal: reopen only on Won/Lost" }
func (d *Deal) recordTerminalRejected(evt DealEvent) {
	d.Rejection = "deal-terminal: terminal deal accepts only reopen"
}
