// Package cli owns the process lifecycle: it parses argv, opens the database,
// owns the single write transaction, authorizes (through crm.domain, never by
// importing crm.authz), runs the domain mutation, and renders (BUILD.md 4.2,
// 5.5, 9). It imports crm.session, crm.domain, crm.repo, and the model kernel;
// it does NOT import crm.authz (BUILD.md 4.5 deny, enforced by C-ARCH-01).
package cli

import (
	"errors"

	"crm/internal/model"
)

// CmdState is the CommandExecution machine state (BUILD.md 5.5,
// CommandExecution.machine.json).
type CmdState string

const (
	CParsing          CmdState = "Parsing"
	COpening          CmdState = "Opening"
	CDBLocked         CmdState = "DBLocked"
	CResolvingSession CmdState = "ResolvingSession"
	CAuthorizing      CmdState = "Authorizing"
	CExecuting        CmdState = "Executing"
	CRendering        CmdState = "Rendering"
	CDone             CmdState = "Done"
	CDenied           CmdState = "Denied"
	CValidationFailed CmdState = "ValidationFailed"
	CDBError          CmdState = "DBError"
	CCorrupt          CmdState = "Corrupt"
)

// ExitClass is the terminal-state exit classification (BUILD.md 5.5: "Five
// terminal states set the process exit code"). The spec assigns no numeric
// codes, so tests assert the record*Exit action and this class, not a number.
type ExitClass string

const (
	ExitUnset      ExitClass = ""
	ExitSuccess    ExitClass = "success"
	ExitDenied     ExitClass = "denied"
	ExitValidation ExitClass = "validation"
	ExitDBError    ExitClass = "dberror"
	ExitCorrupt    ExitClass = "corrupt"
)

// CmdEventKind is the trigger: always, an invoke result, or an after delay
// (CommandExecution.machine.json). This machine takes no external user events.
type CmdEventKind string

const (
	CEvAlways                CmdEventKind = "always"
	CEvInvokeDone            CmdEventKind = "invokeDone"
	CEvInvokeError           CmdEventKind = "invokeError"
	CEvOpenTimeout           CmdEventKind = "openTimeout"
	CEvDbRetryBackoff        CmdEventKind = "dbRetryBackoff"
	CEvSessionResolveTimeout CmdEventKind = "sessionResolveTimeout"
	CEvQueryTimeout          CmdEventKind = "queryTimeout"
)

// CmdEvent is a single trigger; Err carries the classified error on invokeError.
type CmdEvent struct {
	Kind CmdEventKind
	Err  error
}

// CommandExecution is the explicit operational-envelope state machine that the
// T-CMD oracle (BUILD.md 7.1 T-CMD-01..33) exercises. It mirrors
// CommandExecution.machine.json.
type CommandExecution struct {
	State         CmdState
	Args          []string // raw argv, parsed by guardParseOk / captureArgs
	Verb          model.Verb
	EntityType    model.EntityType
	Actor         model.User
	TargetOwnerID string
	TargetTeamID  string
	Phase         string // "open" | "execute"
	Retries       int
	LastError     error
	Exit          ExitClass

	// Authorize routes to crm.domain (which owns the crm.authz call) so the
	// command layer never imports crm.authz. Wired by the implementer.
	Authorize func(actor model.User, verb model.Verb, entity model.EntityType, ownerID, teamID string) (bool, string)
}

// Fire applies a trigger, mirroring CommandExecution.machine.json (BUILD.md 7.1
// T-CMD-01..33). Entering a state runs its entry action(s) after the transition
// action(s): Opening/Executing set the retry phase, Rendering renders, and the
// five terminal states set the exit classification.
func (c *CommandExecution) Fire(evt CmdEvent) model.Effect {
	switch c.State {
	case CParsing:
		return c.fireParsing(evt)
	case COpening:
		return c.fireOpening(evt)
	case CDBLocked:
		return c.fireDBLocked(evt)
	case CResolvingSession:
		return c.fireResolvingSession(evt)
	case CAuthorizing:
		return c.fireAuthorizing(evt)
	case CExecuting:
		return c.fireExecuting(evt)
	case CRendering:
		return c.fireRendering(evt)
	}
	return model.Effect{}
}

func (c *CommandExecution) fireParsing(evt CmdEvent) model.Effect {
	if evt.Kind != CEvAlways {
		return model.Effect{}
	}
	if c.guardParseOk(evt) {
		c.captureArgs(evt)
		return c.enter(COpening, "captureArgs")
	}
	c.recordParseError(evt)
	return c.enter(CValidationFailed, "recordParseError")
}

func (c *CommandExecution) fireOpening(evt CmdEvent) model.Effect {
	switch evt.Kind {
	case CEvInvokeDone:
		c.captureTx(evt)
		return c.enter(CResolvingSession, "captureTx")
	case CEvInvokeError:
		switch {
		case c.isErrLocked(evt):
			c.recordError(evt)
			return c.enter(CDBLocked, "recordError")
		case c.isErrCorrupt(evt):
			c.recordCorrupt(evt)
			return c.enter(CCorrupt, "recordCorrupt")
		case c.isErrUnavailable(evt):
			c.recordUnavailable(evt)
			return c.enter(CDBError, "recordUnavailable")
		default:
			c.recordOpenError(evt)
			return c.enter(CDBError, "recordOpenError")
		}
	case CEvOpenTimeout:
		c.recordTimeout(evt)
		return c.enter(CDBError, "recordTimeout")
	}
	return model.Effect{}
}

func (c *CommandExecution) fireDBLocked(evt CmdEvent) model.Effect {
	switch evt.Kind {
	case CEvAlways:
		if c.retriesExhausted() {
			c.recordLockExhausted(evt)
			return c.enter(CDBError, "recordLockExhausted")
		}
	case CEvDbRetryBackoff:
		switch {
		case c.phaseIsOpen():
			c.incrementRetries()
			return c.enter(COpening, "incrementRetries")
		case c.phaseIsExecute():
			c.incrementRetries()
			return c.enter(CExecuting, "incrementRetries")
		}
	}
	return model.Effect{}
}

func (c *CommandExecution) fireResolvingSession(evt CmdEvent) model.Effect {
	switch evt.Kind {
	case CEvInvokeDone:
		c.captureActor(evt)
		return c.enter(CAuthorizing, "captureActor")
	case CEvInvokeError:
		switch {
		case c.isErrNoSession(evt), c.isErrExpired(evt):
			c.recordNeedLogin(evt)
			return c.enter(CDenied, "recordNeedLogin")
		case c.isErrLocked(evt):
			c.Phase = "open" // a lock during resolve retries the open
			c.recordError(evt)
			return c.enter(CDBLocked, "recordError")
		default:
			c.recordSessionError(evt)
			return c.enter(CDBError, "recordSessionError")
		}
	case CEvSessionResolveTimeout:
		c.recordTimeout(evt)
		return c.enter(CDBError, "recordTimeout")
	}
	return model.Effect{}
}

func (c *CommandExecution) fireAuthorizing(evt CmdEvent) model.Effect {
	if evt.Kind != CEvAlways {
		return model.Effect{}
	}
	if c.guardAuthorized(evt) {
		c.recordAllowed(evt)
		return c.enter(CExecuting, "recordAllowed")
	}
	c.recordDenyReason(evt)
	return c.enter(CDenied, "recordDenyReason")
}

func (c *CommandExecution) fireExecuting(evt CmdEvent) model.Effect {
	switch evt.Kind {
	case CEvInvokeDone:
		c.captureResult(evt)
		return c.enter(CRendering, "captureResult")
	case CEvInvokeError:
		switch {
		case c.isErrConstraint(evt):
			c.ensureRolledBack()
			c.recordConstraint(evt)
			return c.enter(CValidationFailed, "ensureRolledBack", "recordConstraint")
		case c.isErrLocked(evt):
			c.Phase = "execute" // retry the whole write Tx
			c.ensureRolledBack()
			c.recordError(evt)
			return c.enter(CDBLocked, "ensureRolledBack", "recordError")
		case c.isErrConflict(evt):
			c.Phase = "execute"
			c.ensureRolledBack()
			c.recordConflict(evt)
			return c.enter(CDBLocked, "ensureRolledBack", "recordConflict")
		case c.isErrDiskFull(evt):
			c.ensureRolledBack()
			c.recordDiskFull(evt)
			return c.enter(CDBError, "ensureRolledBack", "recordDiskFull")
		case c.isErrTimeout(evt):
			c.ensureRolledBack()
			c.recordTimeout(evt)
			return c.enter(CDBError, "ensureRolledBack", "recordTimeout")
		default:
			c.ensureRolledBack()
			c.recordExecuteError(evt)
			return c.enter(CDBError, "ensureRolledBack", "recordExecuteError")
		}
	case CEvQueryTimeout:
		c.ensureRolledBack()
		c.recordTimeout(evt)
		return c.enter(CDBError, "ensureRolledBack", "recordTimeout")
	}
	return model.Effect{}
}

func (c *CommandExecution) fireRendering(evt CmdEvent) model.Effect {
	if evt.Kind != CEvAlways {
		return model.Effect{}
	}
	return c.enter(CDone)
}

// enter transitions to state s, appending s's entry action(s) after the given
// transition action(s) (BUILD.md 5.5: entry actions fire on state entry).
func (c *CommandExecution) enter(s CmdState, transitionActions ...string) model.Effect {
	actions := append([]string{}, transitionActions...)
	c.State = s
	switch s {
	case COpening:
		c.setPhaseOpen()
		actions = append(actions, "setPhaseOpen")
	case CExecuting:
		c.setPhaseExecute()
		actions = append(actions, "setPhaseExecute")
	case CRendering:
		c.renderOutput()
		actions = append(actions, "renderOutput")
	case CDone:
		c.recordSuccessExit()
		actions = append(actions, "recordSuccessExit")
	case CDenied:
		c.recordDeniedExit()
		actions = append(actions, "recordDeniedExit")
	case CValidationFailed:
		c.recordValidationExit()
		actions = append(actions, "recordValidationExit")
	case CDBError:
		c.recordDBErrorExit()
		actions = append(actions, "recordDBErrorExit")
	case CCorrupt:
		c.recordCorruptExit()
		actions = append(actions, "recordCorruptExit")
	}
	return model.Effect{Actions: actions}
}

// --- Guards (BUILD.md 5.5). ---

// cmdNouns and cmdVerbs are the recognized argv tokens for guardParseOk /
// captureArgs (BUILD.md 9 cobra command tree).
var cmdNouns = map[string]model.EntityType{
	"user": model.EntityUser, "team": model.EntityTeam, "account": model.EntityAccount,
	"contact": model.EntityContact, "deal": model.EntityDeal, "pipeline": model.EntityPipeline,
	"activity": model.EntityActivity, "task": model.EntityTask, "tag": model.EntityTag,
}

var cmdVerbs = map[string]model.Verb{
	"create": model.VerbCreate, "read": model.VerbRead, "list": model.VerbRead,
	"show": model.VerbRead, "update": model.VerbUpdate, "rename": model.VerbUpdate,
	"delete": model.VerbDelete, "remove": model.VerbDelete, "reassign": model.VerbReassign,
	"advance": model.VerbUpdate, "win": model.VerbUpdate, "lose": model.VerbUpdate,
	"reopen": model.VerbUpdate, "start": model.VerbUpdate, "complete": model.VerbUpdate,
	"cancel": model.VerbUpdate, "disable": model.VerbUpdate, "enable": model.VerbUpdate,
	"log": model.VerbCreate, "apply": model.VerbUpdate, "set-default": model.VerbUpdate,
	"assign-role": model.VerbUpdate, "change-password": model.VerbUpdate,
}

// guardParseOk reports whether argv parses to a valid (noun, verb, flags).
func (c *CommandExecution) guardParseOk(evt CmdEvent) bool {
	if len(c.Args) < 2 {
		return false
	}
	_, nounOK := cmdNouns[c.Args[0]]
	_, verbOK := cmdVerbs[c.Args[1]]
	return nounOK && verbOK
}

// guardAuthorized is the single call site of the pure authz decision, routed
// through the injected domain callback so crm.cli never imports crm.authz.
func (c *CommandExecution) guardAuthorized(evt CmdEvent) bool {
	if c.Authorize == nil {
		return false
	}
	allowed, reason := c.Authorize(c.Actor, c.Verb, c.EntityType, c.TargetOwnerID, c.TargetTeamID)
	if !allowed && c.LastError == nil {
		c.LastError = errors.New(reason)
	}
	return allowed
}

func (c *CommandExecution) phaseIsOpen() bool    { return c.Phase == "open" }
func (c *CommandExecution) phaseIsExecute() bool { return c.Phase == "execute" }

func (c *CommandExecution) isErrLocked(evt CmdEvent) bool { return cmdErrIs(evt.Err, model.ErrLocked) }
func (c *CommandExecution) isErrCorrupt(evt CmdEvent) bool {
	return cmdErrIs(evt.Err, model.ErrCorrupt)
}
func (c *CommandExecution) isErrUnavailable(evt CmdEvent) bool {
	return cmdErrIs(evt.Err, model.ErrUnavailable)
}
func (c *CommandExecution) isErrNoSession(evt CmdEvent) bool {
	return cmdErrIs(evt.Err, model.ErrNoSession)
}
func (c *CommandExecution) isErrExpired(evt CmdEvent) bool {
	return cmdErrIs(evt.Err, model.ErrExpired)
}
func (c *CommandExecution) isErrConstraint(evt CmdEvent) bool {
	return cmdErrIs(evt.Err, model.ErrConstraint)
}
func (c *CommandExecution) isErrConflict(evt CmdEvent) bool {
	return cmdErrIs(evt.Err, model.ErrConflict)
}
func (c *CommandExecution) isErrDiskFull(evt CmdEvent) bool {
	return cmdErrIs(evt.Err, model.ErrDiskFull)
}
func (c *CommandExecution) isErrTimeout(evt CmdEvent) bool {
	return cmdErrIs(evt.Err, model.ErrTimeout)
}
func (c *CommandExecution) retriesExhausted() bool { return c.Retries >= cmdMaxRetries }

// cmdMaxRetries is the DB open/write retry bound (BUILD.md 9).
const cmdMaxRetries = 3

// cmdErrIs matches err against a sentinel, tolerating a nil err.
func cmdErrIs(err, target error) bool { return err != nil && errors.Is(err, target) }

// --- Actions (BUILD.md 5.5). ---

func (c *CommandExecution) captureArgs(evt CmdEvent) {
	if len(c.Args) >= 1 {
		c.EntityType = cmdNouns[c.Args[0]]
	}
	if len(c.Args) >= 2 {
		c.Verb = cmdVerbs[c.Args[1]]
	}
}
func (c *CommandExecution) recordParseError(evt CmdEvent) {
	c.LastError = errors.New("cli: unrecognized command")
}
func (c *CommandExecution) setPhaseOpen()              { c.Phase = "open" }
func (c *CommandExecution) setPhaseExecute()           { c.Phase = "execute" }
func (c *CommandExecution) captureTx(evt CmdEvent)     {}
func (c *CommandExecution) captureActor(evt CmdEvent)  {}
func (c *CommandExecution) captureResult(evt CmdEvent) {}
func (c *CommandExecution) incrementRetries()          { c.Retries++ }
func (c *CommandExecution) ensureRolledBack()          {}
func (c *CommandExecution) renderOutput()              {}
func (c *CommandExecution) recordAllowed(evt CmdEvent) { c.LastError = nil }
func (c *CommandExecution) recordDenyReason(evt CmdEvent) {
	if c.LastError == nil {
		c.LastError = errors.New("authorization denied")
	}
}
func (c *CommandExecution) recordError(evt CmdEvent)         { c.LastError = evt.Err }
func (c *CommandExecution) recordCorrupt(evt CmdEvent)       { c.LastError = evt.Err }
func (c *CommandExecution) recordUnavailable(evt CmdEvent)   { c.LastError = evt.Err }
func (c *CommandExecution) recordOpenError(evt CmdEvent)     { c.LastError = evt.Err }
func (c *CommandExecution) recordNeedLogin(evt CmdEvent)     { c.LastError = evt.Err }
func (c *CommandExecution) recordSessionError(evt CmdEvent)  { c.LastError = evt.Err }
func (c *CommandExecution) recordConstraint(evt CmdEvent)    { c.LastError = evt.Err }
func (c *CommandExecution) recordConflict(evt CmdEvent)      { c.LastError = evt.Err }
func (c *CommandExecution) recordDiskFull(evt CmdEvent)      { c.LastError = evt.Err }
func (c *CommandExecution) recordTimeout(evt CmdEvent)       { c.LastError = model.ErrTimeout }
func (c *CommandExecution) recordExecuteError(evt CmdEvent)  { c.LastError = evt.Err }
func (c *CommandExecution) recordLockExhausted(evt CmdEvent) { c.LastError = model.ErrLocked }
func (c *CommandExecution) recordSuccessExit()               { c.Exit = ExitSuccess }
func (c *CommandExecution) recordDeniedExit()                { c.Exit = ExitDenied }
func (c *CommandExecution) recordValidationExit()            { c.Exit = ExitValidation }
func (c *CommandExecution) recordDBErrorExit()               { c.Exit = ExitDBError }
func (c *CommandExecution) recordCorruptExit()               { c.Exit = ExitCorrupt }
