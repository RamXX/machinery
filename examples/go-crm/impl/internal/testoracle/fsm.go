// Package testoracle contains Go CRM test support only.
package testoracle

import "errors"

type Identity struct{ ID, Source, Trigger, Guard, Target string }
type Row struct {
	Identity
	Actions []string
}
type State struct {
	Name, Kind  string
	Entry, Exit []string
}
type Suite struct {
	Name   string
	Rows   []Row
	States []State
}
type Witness struct {
	Name, RowID, Source, Trigger string
	Guards                       map[string]bool
}
type Expectation struct {
	Machine string
	Row     Row
	Next    string
	Actions []string
}
type Registration struct {
	Machine, Native string
	Identity        Identity
}
type Observation struct {
	Registration Registration
	Next         string
	Actions      []string
}

// ErrScaffold is deliberately nonqualifying RED. The independent initial
// subprocess RED predates this compile-only surface. GREEN implements this API.
var ErrScaffold = errors.New("oracle-scaffold: parser and reconciliation are not implemented")

func Parse(oracle, machine []byte) (*Suite, error)              { return nil, ErrScaffold }
func Load(machineDir, machineName string) (*Suite, error)       { return nil, ErrScaffold }
func (s *Suite) Bind(w Witness) (Expectation, error)            { return Expectation{}, ErrScaffold }
func (s *Suite) Finish() error                                  { return ErrScaffold }
func (e Expectation) Check(next string, actions []string) error { return ErrScaffold }
