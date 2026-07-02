// Package model is the dependency-free kernel: the nine record entities, the
// enums, the typed errors, and the pure data helpers shared by every boundary.
// It has no behavior and imports nothing, so importing it is not a cross-boundary
// dependency in the sense of the section 4.5 Architecture Contract.
package model

// UserRole is the enum from BUILD.md 3.2 (in order).
type UserRole string

const (
	RoleAdmin    UserRole = "Admin"
	RoleManager  UserRole = "Manager"
	RoleRep      UserRole = "Rep"
	RoleReadOnly UserRole = "ReadOnly"
)

// UserStatus is the enum from BUILD.md 3.2 and the User aggregate state.
type UserStatus string

const (
	StatusActive   UserStatus = "Active"
	StatusDisabled UserStatus = "Disabled"
)

// DealStage is the enum from BUILD.md 3.2 and the Deal aggregate resting state.
type DealStage string

const (
	StageLead        DealStage = "Lead"
	StageQualified   DealStage = "Qualified"
	StageProposal    DealStage = "Proposal"
	StageNegotiation DealStage = "Negotiation"
	StageWon         DealStage = "Won"
	StageLost        DealStage = "Lost"
)

// TaskStatus is the enum from BUILD.md 3.2 and the Task aggregate state.
type TaskStatus string

const (
	TaskOpen       TaskStatus = "Open"
	TaskInProgress TaskStatus = "InProgress"
	TaskDone       TaskStatus = "Done"
	TaskCancelled  TaskStatus = "Cancelled"
)

// ActivityType is the enum from BUILD.md 3.2.
type ActivityType string

const (
	ActivityCall    ActivityType = "Call"
	ActivityMeeting ActivityType = "Meeting"
	ActivityEmail   ActivityType = "Email"
	ActivityNote    ActivityType = "Note"
)

// Verb is the CRUD/reassign verb passed to the Authorizer (BUILD.md 4.6).
type Verb string

const (
	VerbCreate   Verb = "create"
	VerbRead     Verb = "read"
	VerbUpdate   Verb = "update"
	VerbDelete   Verb = "delete"
	VerbReassign Verb = "reassign"
)

// EntityType is one of the nine record types (BUILD.md 4.6).
type EntityType string

const (
	EntityUser     EntityType = "User"
	EntityTeam     EntityType = "Team"
	EntityAccount  EntityType = "Account"
	EntityContact  EntityType = "Contact"
	EntityDeal     EntityType = "Deal"
	EntityPipeline EntityType = "Pipeline"
	EntityActivity EntityType = "Activity"
	EntityTask     EntityType = "Task"
	EntityTag      EntityType = "Tag"
)

// stageOrder is the forward order of DealStage (BUILD.md 3.2). Won/Lost are
// terminal outcomes and are not part of the forward chain.
var stageOrder = []DealStage{StageLead, StageQualified, StageProposal, StageNegotiation}

// StageIndex returns the position of a stage in the forward chain, or -1 for a
// terminal stage (Won/Lost). Pure data helper; embeds BUILD.md 3.2.
func StageIndex(s DealStage) int {
	for i, v := range stageOrder {
		if v == s {
			return i
		}
	}
	return -1
}

// NextStage returns the next forward stage and true, or ("", false) when there
// is no forward stage (Negotiation, Won, Lost). Pure data helper (BUILD.md 3.2:
// Lead->Qualified, Qualified->Proposal, Proposal->Negotiation).
func NextStage(s DealStage) (DealStage, bool) {
	i := StageIndex(s)
	if i < 0 || i+1 >= len(stageOrder) {
		return "", false
	}
	return stageOrder[i+1], true
}

// IsTerminalStage reports whether a Deal stage is terminal (Won/Lost).
func IsTerminalStage(s DealStage) bool {
	return s == StageWon || s == StageLost
}

// IsTerminalStatus reports whether a Task status is terminal (Done/Cancelled).
func IsTerminalStatus(s TaskStatus) bool {
	return s == TaskDone || s == TaskCancelled
}
