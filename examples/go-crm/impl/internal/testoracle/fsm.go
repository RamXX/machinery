// Package testoracle contains Go CRM test support only.
package testoracle

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

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
	Name      string
	Rows      []Row
	States    []State
	witnesses map[string]bool
	used      map[string]bool
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

var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
var stableID = regexp.MustCompile(`^[A-Z]{4}-[0-9a-f]{6}$`)
var testID = regexp.MustCompile(`^T-[A-Z]+-[0-9]+$`)
var heading = regexp.MustCompile("^# Generated transition oracle: `([A-Za-z_][A-Za-z0-9_-]*)`$")
var separator = regexp.MustCompile(`^:?-{3,}:?$`)

func parseError(message string) error { return fmt.Errorf("oracle-parse: %s", message) }

// Parse joins the closed, flat Go CRM Markdown and JSON dialects. It never
// executes context, metadata, guards, actions, or automatic transitions.
func Parse(oracle, machine []byte) (*Suite, error) {
	s, err := parseMarkdown(oracle)
	if err != nil {
		return nil, err
	}
	j, err := parseMachine(machine)
	if err != nil {
		return nil, err
	}
	mismatch := func() (*Suite, error) {
		return nil, fmt.Errorf("oracle-identity: %s Markdown and JSON disagree", s.Name)
	}
	if s.Name != j.Name || len(s.States) != len(j.States) || len(s.Rows) != len(j.Rows) {
		return mismatch()
	}
	states := map[string]State{}
	for _, state := range j.States {
		states[state.Name] = state
	}
	for _, state := range s.States {
		other, ok := states[state.Name]
		if !ok || state.Kind != other.Kind || !slices.Equal(state.Entry, other.Entry) || !slices.Equal(state.Exit, other.Exit) {
			return mismatch()
		}
	}
	groups := map[string][]Row{}
	for _, row := range j.Rows {
		key := row.Source + "|" + row.Trigger
		groups[key] = append(groups[key], row)
	}
	tag := strings.ToUpper(s.Name)
	if len(tag) < 4 {
		return mismatch()
	}
	tag = tag[:4]
	fallback := map[string]bool{}
	for _, row := range s.Rows {
		sum := sha256.Sum256([]byte(tag + "|" + row.Source + "|" + row.Trigger + "|" + row.Guard))
		if row.ID != fmt.Sprintf("%s-%x", tag, sum[:3]) {
			return mismatch()
		}
		key := row.Source + "|" + row.Trigger
		group := groups[key]
		if len(group) == 0 {
			return mismatch()
		}
		other := group[0]
		if row.Source != other.Source || row.Trigger != other.Trigger || row.Guard != other.Guard || row.Target != other.Target || !slices.Equal(row.Actions, other.Actions) {
			return mismatch()
		}
		groups[key] = group[1:]
	}
	// Reconciliation precedes reachability so a reordered counterpart reports
	// an identity mismatch, even if its new order also shadows an alternative.
	for _, row := range s.Rows {
		key := row.Source + "|" + row.Trigger
		if fallback[key] {
			return nil, parseError("alternative after fallback")
		}
		fallback[key] = row.Guard == ""
	}
	s.witnesses, s.used = map[string]bool{}, map[string]bool{}
	return s, nil
}

func parseMarkdown(raw []byte) (*Suite, error) {
	s := &Suite{}
	section, stage, total := "", 0, -1
	seenSections, names, ids, tests, identities := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, rawLine := range strings.Split(string(raw), "\n") {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "# Generated transition oracle:") {
			m := heading.FindStringSubmatch(line)
			if len(m) != 2 || s.Name != "" {
				return nil, parseError("machine heading")
			}
			s.Name = m[1]
			continue
		}
		if line == "## State entry / exit actions" || line == "## Transitions" {
			if seenSections[line] || (section != "" && stage != 2) {
				return nil, parseError("duplicate or incomplete section")
			}
			seenSections[line] = true
			section, stage = line, 0
			continue
		}
		if strings.HasPrefix(line, "Total transitions (test cases):") {
			n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Total transitions (test cases):")))
			if err != nil || n < 0 || total != -1 || section != "## Transitions" || stage != 2 {
				return nil, parseError("transition total")
			}
			total = n
			continue
		}
		if !strings.HasPrefix(line, "|") {
			continue
		}
		if section == "" || total != -1 || !strings.HasSuffix(line, "|") {
			return nil, parseError("table row outside section or missing delimiter")
		}
		cells := strings.Split(line[1:len(line)-1], "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		columns := []string{"state", "kind", "entry", "exit"}
		if section == "## Transitions" {
			columns = []string{"test id", "stable id", "source", "trigger", "guard", "target", "actions"}
		}
		if len(cells) != len(columns) {
			return nil, parseError("table width")
		}
		if stage == 0 {
			if !slices.Equal(cells, columns) {
				return nil, parseError("table header")
			}
			stage++
			continue
		}
		if stage == 1 {
			for _, cell := range cells {
				if !separator.MatchString(cell) {
					return nil, parseError("table separator")
				}
			}
			stage++
			continue
		}
		if section == "## State entry / exit actions" {
			entry, err := cellActions(cells[2])
			if err != nil {
				return nil, err
			}
			exit, err := cellActions(cells[3])
			if err != nil {
				return nil, err
			}
			if !identifier.MatchString(cells[0]) || names[cells[0]] || (cells[1] != "atomic" && cells[1] != "final") {
				return nil, parseError("state identity or kind")
			}
			names[cells[0]] = true
			s.States = append(s.States, State{cells[0], cells[1], entry, exit})
			continue
		}
		actions, err := cellActions(cells[6])
		if err != nil {
			return nil, err
		}
		guard := cells[4]
		if guard == "-" {
			guard = ""
		} else if !identifier.MatchString(guard) {
			return nil, parseError("guard")
		}
		identity := cells[2] + "|" + cells[3] + "|" + guard
		if !testID.MatchString(cells[0]) || !stableID.MatchString(cells[1]) || !identifier.MatchString(cells[2]) || !validTrigger(cells[3]) || (cells[5] != "(internal)" && !identifier.MatchString(cells[5])) || tests[cells[0]] || ids[cells[1]] || identities[identity] {
			return nil, parseError("transition identity")
		}
		tests[cells[0]], ids[cells[1]], identities[identity] = true, true, true
		s.Rows = append(s.Rows, Row{Identity{cells[1], cells[2], cells[3], guard, cells[5]}, actions})
	}
	if s.Name == "" || len(seenSections) != 2 || stage != 2 || len(s.States) == 0 || total != len(s.Rows) {
		return nil, parseError("incomplete oracle or count mismatch")
	}
	if err := validateReferences(s); err != nil {
		return nil, err
	}
	return s, nil
}

func cellActions(cell string) ([]string, error) {
	if cell == "-" {
		return nil, nil
	}
	parts := strings.Split(cell, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if !identifier.MatchString(parts[i]) {
			return nil, parseError("action cell")
		}
	}
	return parts, nil
}

func validTrigger(trigger string) bool {
	if trigger == "always" {
		return true
	}
	kind, key, ok := strings.Cut(trigger, ":")
	return ok && identifier.MatchString(key) && (kind == "on" || kind == "after" || kind == "onDone" || kind == "onError")
}

// Read tokens rather than unmarshalling into maps: duplicate keys must fail
// even inside otherwise inert context and metadata values.
func jsonValue(d *json.Decoder) (any, error) {
	t, err := d.Token()
	if err != nil {
		return nil, err
	}
	if delimiter, ok := t.(json.Delim); ok {
		switch delimiter {
		case '{':
			object := map[string]any{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return nil, err
				}
				name, ok := key.(string)
				if !ok {
					return nil, parseError("object key")
				}
				if _, exists := object[name]; exists {
					return nil, parseError("duplicate JSON key " + name)
				}
				value, err := jsonValue(d)
				if err != nil {
					return nil, err
				}
				object[name] = value
			}
			_, err := d.Token()
			return object, err
		case '[':
			var array []any
			for d.More() {
				value, err := jsonValue(d)
				if err != nil {
					return nil, err
				}
				array = append(array, value)
			}
			_, err := d.Token()
			return array, err
		default:
			return nil, parseError("unexpected JSON delimiter")
		}
	}
	return t, nil
}

func objectFields(value any, allowed string) (map[string]any, error) {
	obj, ok := value.(map[string]any)
	if !ok {
		return nil, parseError("expected object")
	}
	for key := range obj {
		if !strings.Contains(" "+allowed+" ", " "+key+" ") {
			return nil, parseError("unknown field " + key)
		}
	}
	return obj, nil
}

func jsonIdentifier(value any) (string, error) {
	s, ok := value.(string)
	if !ok || !identifier.MatchString(s) {
		return "", parseError("expected identifier")
	}
	return s, nil
}

func jsonActions(obj map[string]any, key string) ([]string, error) {
	v, exists := obj[key]
	if !exists {
		return nil, nil
	}
	values, ok := v.([]any)
	if !ok {
		values = []any{v}
	}
	if len(values) == 0 {
		return nil, parseError("empty actions")
	}
	var actions []string
	for _, value := range values {
		action, err := jsonIdentifier(value)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func stringMap(value any) error {
	obj, ok := value.(map[string]any)
	if !ok {
		return parseError("expected string metadata map")
	}
	for _, v := range obj {
		if _, ok := v.(string); !ok {
			return parseError("metadata value must be string")
		}
	}
	return nil
}

func parseMachine(raw []byte) (*Suite, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	value, err := jsonValue(d)
	if err != nil {
		return nil, parseError(err.Error())
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, parseError("trailing JSON")
	}
	root, err := objectFields(value, "id initial context states _comment _role _delays _counters")
	if err != nil {
		return nil, err
	}
	name, err := jsonIdentifier(root["id"])
	if err != nil {
		return nil, err
	}
	initial, err := jsonIdentifier(root["initial"])
	if err != nil {
		return nil, err
	}
	for _, key := range []string{"_comment", "_role"} {
		if v, ok := root[key]; ok {
			if _, ok := v.(string); !ok {
				return nil, parseError("metadata string")
			}
		}
	}
	for _, key := range []string{"_delays", "_counters"} {
		if v, ok := root[key]; ok {
			if err := stringMap(v); err != nil {
				return nil, err
			}
		}
	}
	if v, ok := root["context"]; ok {
		if _, ok := v.(map[string]any); !ok {
			return nil, parseError("context object")
		}
	}
	states, ok := root["states"].(map[string]any)
	if !ok || len(states) == 0 {
		return nil, parseError("states object")
	}
	if _, ok := states[initial]; !ok {
		return nil, parseError("unknown initial state")
	}
	s := &Suite{Name: name}
	for stateName, value := range states {
		if !identifier.MatchString(stateName) {
			return nil, parseError("state name")
		}
		obj, err := objectFields(value, "type entry exit on after always invoke _refusal")
		if err != nil {
			return nil, err
		}
		kind := "atomic"
		if v, ok := obj["type"]; ok {
			kind, ok = v.(string)
			if !ok || (kind != "atomic" && kind != "final") {
				return nil, parseError("state kind")
			}
		}
		entry, err := jsonActions(obj, "entry")
		if err != nil {
			return nil, err
		}
		exit, err := jsonActions(obj, "exit")
		if err != nil {
			return nil, err
		}
		if v, ok := obj["_refusal"]; ok {
			if err := stringMap(v); err != nil {
				return nil, err
			}
		}
		s.States = append(s.States, State{stateName, kind, entry, exit})
		for _, kind := range []string{"on", "after"} {
			if value, exists := obj[kind]; exists {
				events, ok := value.(map[string]any)
				if !ok {
					return nil, parseError("event map")
				}
				for event, transitions := range events {
					if !identifier.MatchString(event) {
						return nil, parseError("event identity")
					}
					if err := addAlternatives(s, stateName, kind+":"+event, transitions); err != nil {
						return nil, err
					}
				}
			}
		}
		if value, exists := obj["always"]; exists {
			if err := addAlternatives(s, stateName, "always", value); err != nil {
				return nil, err
			}
		}
		if value, exists := obj["invoke"]; exists {
			inv, err := objectFields(value, "src input onDone onError")
			if err != nil {
				return nil, err
			}
			src, err := jsonIdentifier(inv["src"])
			if err != nil {
				return nil, err
			}
			if input, exists := inv["input"]; exists {
				if err := stringMap(input); err != nil {
					return nil, err
				}
			}
			for _, event := range []string{"onDone", "onError"} {
				if transitions, exists := inv[event]; exists {
					if err := addAlternatives(s, stateName, event+":"+src, transitions); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	if err := validateReferences(s); err != nil {
		return nil, err
	}
	return s, nil
}

func addAlternatives(s *Suite, source, trigger string, value any) error {
	values, ok := value.([]any)
	if !ok {
		values = []any{value}
	}
	if len(values) == 0 {
		return parseError("empty alternatives")
	}
	guards := map[string]bool{}
	for _, value := range values {
		obj, err := objectFields(value, "target guard actions")
		if err != nil {
			return err
		}
		target, guard := "(internal)", ""
		if v, exists := obj["target"]; exists {
			target, err = jsonIdentifier(v)
			if err != nil {
				return err
			}
		}
		if v, exists := obj["guard"]; exists {
			guard, err = jsonIdentifier(v)
			if err != nil {
				return err
			}
		}
		if guards[guard] {
			return parseError("duplicate transition identity")
		}
		guards[guard] = true
		actions, err := jsonActions(obj, "actions")
		if err != nil {
			return err
		}
		s.Rows = append(s.Rows, Row{Identity{Source: source, Trigger: trigger, Guard: guard, Target: target}, actions})
	}
	return nil
}

func validateReferences(s *Suite) error {
	states := map[string]State{}
	for _, state := range s.States {
		states[state.Name] = state
	}
	for _, row := range s.Rows {
		source, exists := states[row.Source]
		if !exists || source.Kind == "final" {
			return parseError("unknown source or final outgoing transition")
		}
		if _, exists := states[row.Target]; row.Target != "(internal)" && !exists {
			return parseError("unknown target")
		}
	}
	return nil
}

// Load rereads both current sources on every call.
func Load(machineDir, machineName string) (*Suite, error) {
	var inputs [][]byte
	for _, suffix := range []string{".oracle.md", ".machine.json"} {
		path := filepath.Join(machineDir, machineName+suffix)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("oracle-load %s: %w", path, err)
		}
		inputs = append(inputs, raw)
	}
	return Parse(inputs[0], inputs[1])
}

// Bind selects the first enabled alternative from the actual input facts and
// records a distinct witness only after its closed identity has been checked.
func (s *Suite) Bind(w Witness) (Expectation, error) {
	fail := func() (Expectation, error) {
		return Expectation{}, fmt.Errorf("oracle-witness %s %s: invalid identity, guard facts, priority or duplicate name %q", s.Name, w.RowID, w.Name)
	}
	if w.Name == "" || s.witnesses == nil || s.witnesses[w.Name] {
		return fail()
	}
	var alternatives []Row
	keys := map[string]bool{}
	for _, row := range s.Rows {
		if row.Source == w.Source && row.Trigger == w.Trigger {
			alternatives = append(alternatives, row)
			if row.Guard != "" {
				keys[row.Guard] = true
			}
		}
	}
	if len(alternatives) == 0 || len(keys) != len(w.Guards) {
		return fail()
	}
	for key := range keys {
		if _, ok := w.Guards[key]; !ok {
			return fail()
		}
	}
	for _, row := range alternatives {
		if row.Guard != "" && !w.Guards[row.Guard] {
			continue
		}
		if row.ID != w.RowID {
			return fail()
		}
		exp := Expectation{Machine: s.Name, Row: row, Next: row.Target}
		if row.Target == "(internal)" {
			exp.Next = row.Source
		} else {
			for _, state := range s.States {
				if state.Name == row.Source {
					exp.Actions = append(exp.Actions, state.Exit...)
				}
			}
		}
		exp.Actions = append(exp.Actions, row.Actions...)
		if row.Target != "(internal)" {
			for _, state := range s.States {
				if state.Name == row.Target {
					exp.Actions = append(exp.Actions, state.Entry...)
				}
			}
		}
		s.witnesses[w.Name], s.used[row.ID] = true, true
		return exp, nil
	}
	return fail()
}

// Finish requires every parsed row to have a successfully bound witness.
// Native registration and successful execution are verified separately by tests.
func (s *Suite) Finish() error {
	if len(s.Rows) == 0 {
		return fmt.Errorf("oracle-unused-row %s: empty suite", s.Name)
	}
	for _, row := range s.Rows {
		if !s.used[row.ID] {
			return fmt.Errorf("oracle-unused-row %s %s", s.Name, row.ID)
		}
	}
	return nil
}

// Check compares the full one-Fire state and ordered action list, including
// multiplicity; nil and empty lists both mean no actions.
func (e Expectation) Check(next string, actions []string) error {
	if next != e.Next {
		return fmt.Errorf("oracle-next-state %s %s: got=%q want=%q", e.Machine, e.Row.ID, next, e.Next)
	}
	if !slices.Equal(actions, e.Actions) {
		return fmt.Errorf("oracle-actions %s %s: got=%q want=%q", e.Machine, e.Row.ID, actions, e.Actions)
	}
	return nil
}
