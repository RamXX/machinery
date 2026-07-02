package ir

import (
	"bytes"
	"encoding/json"
	"sort"
)

// IRDumpState is one state in the canonical IR dump.
type IRDumpState struct {
	Path  string   `json:"path"`
	Name  string   `json:"name"`
	Type  *string  `json:"type"`
	Entry []string `json:"entry"`
	Exit  []string `json:"exit"`
}

// IRDumpTransition is one transition in the canonical IR dump.
// Target/Guard are *string so an absent field (Python None) serializes as JSON
// null, matching ir_dump.py exactly; a present empty string stays "".
type IRDumpTransition struct {
	State   string   `json:"state"`
	Kind    string   `json:"kind"`
	Event   string   `json:"event"`
	Target  *string  `json:"target"`
	Guard   *string  `json:"guard"`
	Actions []string `json:"actions"`
}

// IRDump is the canonical serialization proving parse+traversal parity.
type IRDump struct {
	Machine     string             `json:"machine"`
	Initial     string             `json:"initial"`
	States      []IRDumpState      `json:"states"`
	Transitions []IRDumpTransition `json:"transitions"`
}

// Dump builds the canonical IR dump of a parsed machine root *Value.
func Dump(root *Value) IRDump {
	var machineID, initial string
	if ro := root.AsObject(); ro != nil {
		machineID = ro.GetString("id")
		initial = ro.GetString("initial")
	}
	states := WalkStates(root.AsObject().Get2("states"), "")

	var ds []IRDumpState
	for _, s := range states {
		if s.Node == nil || s.Node.Kind != KindObject {
			continue
		}
		o := s.Node.AsObject()
		entry := sortedSet(ActionNames(o.Get2("entry"), nil, ""))
		exit := sortedSet(ActionNames(o.Get2("exit"), nil, ""))
		ds = append(ds, IRDumpState{
			Path:  s.Path,
			Name:  s.Name,
			Type:  typePtr(o),
			Entry: entry,
			Exit:  exit,
		})
	}
	var dt []IRDumpTransition
	for _, s := range states {
		for _, tr := range TransitionsOf(s.Node, nil, s.Path) {
			acts := append([]string{}, tr.Actions...)
			var tgtPtr, guardPtr *string
			if tr.HasTgt {
				tgtPtr = &tr.Target
			}
			if tr.HasGuard {
				guardPtr = &tr.Guard
			}
			dt = append(dt, IRDumpTransition{
				State:   s.Path,
				Kind:    tr.Kind,
				Event:   tr.Event,
				Target:  tgtPtr,
				Guard:   guardPtr,
				Actions: acts,
			})
		}
	}
	return IRDump{
		Machine:     machineID,
		Initial:     initial,
		States:      ds,
		Transitions: dt,
	}
}

func sortedSet(xs []string) []string {
	m := map[string]struct{}{}
	out := []string{}
	for _, x := range xs {
		if _, ok := m[x]; !ok {
			m[x] = struct{}{}
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}

// typePtr returns a *string matching Python node.get("type"): nil when absent
// (serializes as JSON null like Python None), else the value pointer.
func typePtr(o *Object) *string {
	v, ok := o.Get("type")
	if !ok || v == nil {
		return nil
	}
	s := v.AsString()
	return &s
}

// MarshalJSON ensures absent type serializes as JSON null (matching Python None),
// and empty entry/exit serialize as [] (matching Python sorted(set(...)) which is []).
func (s IRDumpState) MarshalJSON() ([]byte, error) {
	type alias IRDumpState
	a := alias(s)
	if a.Entry == nil {
		a.Entry = []string{}
	}
	if a.Exit == nil {
		a.Exit = []string{}
	}
	return json.Marshal(a)
}

// MarshalJSON for IRDumpTransition ensures actions serializes as [] not null.
func (t IRDumpTransition) MarshalJSON() ([]byte, error) {
	type alias IRDumpTransition
	a := alias(t)
	if a.Actions == nil {
		a.Actions = []string{}
	}
	return json.Marshal(a)
}

// MarshalJSON for IRDump ensures empty states/transitions serialize as [].
func (d IRDump) MarshalJSON() ([]byte, error) {
	type alias IRDump
	a := alias(d)
	if a.States == nil {
		a.States = []IRDumpState{}
	}
	if a.Transitions == nil {
		a.Transitions = []IRDumpTransition{}
	}
	return json.Marshal(a)
}

// DumpJSON renders the IR dump as indent=2 JSON, matching ir_dump.py output
// (ensure_ascii=false; HTML escaping disabled so <,>,& stay literal like Python).
func DumpJSON(root *Value) (string, error) {
	d := Dump(root)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(d); err != nil {
		return "", err
	}
	// json.Encoder appends a trailing newline; ir_dump.py uses print() which
	// also adds exactly one trailing newline. Match exactly.
	return buf.String(), nil
}
