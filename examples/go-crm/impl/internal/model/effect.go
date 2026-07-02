package model

// Effect is what a machine Fire call reports: the ordered list of named actions
// that fired on the transition (the BUILD.md matrix "actions" column). It is a
// behavior-free value type shared by the Deal/Task/User (domain), Session, and
// CommandExecution machines, so it lives in the model kernel. Tests assert on
// Effect.Actions and on the machine's resulting state field.
type Effect struct {
	Actions []string
}

// Has reports whether action a is among the fired actions (test convenience).
func (e Effect) Has(a string) bool {
	for _, x := range e.Actions {
		if x == a {
			return true
		}
	}
	return false
}
