package ir

import (
	"os"
	"strings"
	"testing"
)

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func mustJSON(t *testing.T, src string) *Value {
	t.Helper()
	v, err := LoadMachineJSONStr("test", src)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return v
}

func TestObjectOrderPreserved(t *testing.T) {
	o := NewObject()
	for _, k := range []string{"zebra", "apple", "mango", "Banana"} {
		o.Set(k, StringValue(k))
	}
	// Python dict preserves insertion order; our Object must too.
	got := strings.Join(o.Keys(), ",")
	want := "zebra,apple,mango,Banana"
	if got != want {
		t.Fatalf("order not preserved: got %q want %q", got, want)
	}
}

func TestWalkStatesDepthFirst(t *testing.T) {
	m := mustJSON(t, `{"id":"m","initial":"A","states":{
		"A":{"on":{"x":"B"},"states":{"A1":{"type":"final"},"A2":{"on":{"y":"A1"}}}},
		"B":{"type":"final"}}}`)
	states := WalkStates(m.AsObject().Get2("states"), "")
	// Expect: A, A.A1, A.A2, B
	paths := []string{}
	for _, s := range states {
		paths = append(paths, s.Path)
	}
	want := "A A.A1 A.A2 B"
	if got := strings.Join(paths, " "); got != want {
		t.Fatalf("walk order: got %q want %q", got, want)
	}
}

func TestTransitionsOfKindsAndOrder(t *testing.T) {
	m := mustJSON(t, `{"id":"m","initial":"A","states":{
		"A":{"on":{"go":"B"},"after":{"t":"C"},"always":[{"target":"D"}],
		      "invoke":{"src":"act","onDone":{"target":"E"},"onError":{"target":"F"}}},
		"B":{"type":"final"},"C":{"type":"final"},"D":{"type":"final"},
		"E":{"type":"final"},"F":{"type":"final"}}}`)
	node := m.AsObject().Get2("states").AsObject().Get2("A")
	trs := TransitionsOf(node, nil, "A")
	// Order must be: on, after, always, then invoke.onDone, invoke.onError
	var kinds []string
	for _, tr := range trs {
		kinds = append(kinds, tr.Kind+":"+tr.Event)
	}
	got := strings.Join(kinds, " ")
	want := "on:go after:t always: onDone:act onError:act"
	if got != want {
		t.Fatalf("transitions: got %q want %q", got, want)
	}
}

func TestActionNamesPolymorphism(t *testing.T) {
	// string, {type}, list of those
	v, _ := LoadMachineJSONStr("t", `{"x":"a","y":{"type":"b"},"z":["c",{"type":"d"}]}`)
	o := v.AsObject()
	got := ActionNames(o.Get2("x"), nil, "")
	got = append(got, ActionNames(o.Get2("y"), nil, "")...)
	got = append(got, ActionNames(o.Get2("z"), nil, "")...)
	want := "a b c d"
	if strings.Join(got, " ") != want {
		t.Fatalf("action_names: got %v want %s", got, want)
	}
}

func TestActionNamesBogusValueIsProblem(t *testing.T) {
	v, _ := LoadMachineJSONStr("t", `[42]`)
	var probs []string
	ActionNames(v, &probs, "entry")
	if len(probs) == 0 || !strings.Contains(probs[0], "unsupported action value") {
		t.Fatalf("expected problem, got %v", probs)
	}
}

func TestNormGuardAbsentIsNotString(t *testing.T) {
	// A transition dict without guard: HasGuard must be false (Python None).
	trs := TransitionsOf(mustJSON(t, `{"states":{"S":{"on":{"e":`+
		`[{"target":"X","actions":"a"}]`+
		`}}}}`).AsObject().Get2("states").AsObject().Get2("S"), nil, "S")
	if len(trs) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(trs))
	}
	if trs[0].HasGuard {
		t.Fatalf("absent guard must be HasGuard=false (Python None)")
	}
	if trs[0].Guard != "" {
		t.Fatalf("absent guard must be empty string")
	}
	if !trs[0].HasTgt || trs[0].Target != "X" {
		t.Fatalf("target resolution: %+v", trs[0])
	}
}

func TestParseMdTablesSeparator(t *testing.T) {
	text := "| a | b |\n|---|---|\n| 1 | 2 |\n"
	tbls := ParseMdTables(text)
	if len(tbls) != 1 || len(tbls[0].Rows) != 1 {
		t.Fatalf("got %v", tbls)
	}
	if tbls[0].Header[0] != "a" || tbls[0].Rows[0][1] != "2" {
		t.Fatalf("cells: %+v", tbls[0])
	}
}

func TestParseMdTablesNoSeparator(t *testing.T) {
	// A table without a separator row: all rows are data (matches Python).
	text := "| a | b |\n| 1 | 2 |\n"
	tbls := ParseMdTables(text)
	if len(tbls) != 1 || len(tbls[0].Rows) != 1 {
		t.Fatalf("got %v", tbls)
	}
}

func TestCleanCell(t *testing.T) {
	cases := map[string]string{
		"`foo`":            "foo",
		"`foo` (internal)": "foo",
		"`a` (final)":      "a",
		"-":                "-",
	}
	for in, want := range cases {
		if got := CleanCell(in); got != want {
			t.Errorf("CleanCell(%q)=%q want %q", in, got, want)
		}
	}
}

func TestReprPythonStyle(t *testing.T) {
	type c struct {
		in   interface{}
		want string
	}
	cases := []c{
		{"foo", "'foo'"},
		{"", "''"},
		{"a'b", "\"a'b\""},
		{nil, "None"},
		{true, "True"},
		{false, "False"},
		{42, "42"},
		{[]string{"a", "b"}, "['a', 'b']"},
	}
	for _, k := range cases {
		if got := Repr(k.in); got != k.want {
			t.Errorf("Repr(%v)=%q want %q", k.in, got, k.want)
		}
	}
}

func TestActionObjectUnknownKeysAreProblems(t *testing.T) {
	// IR-F07: {"type": "notify", "params": {...}} silently accepted the extra
	// keys; the docs promise unknown keys are hard errors.
	v, _ := LoadMachineJSONStr("t", `[{"type":"notify","params":{"channel":"email"},"tyop":1}]`)
	var probs []string
	names := ActionNames(v, &probs, "Idle on:GO")
	if len(names) != 1 || names[0] != "notify" {
		t.Fatalf("names: %v", names)
	}
	joined := strings.Join(probs, "\n")
	if !strings.Contains(joined, "'params'") || !strings.Contains(joined, "'tyop'") {
		t.Fatalf("expected problems naming 'params' and 'tyop', got %v", probs)
	}
}

func TestParseMdTablesHonorsEscapedPipes(t *testing.T) {
	// IR-F10: a GFM \| escape inside a cell must not split the cell.
	text := "| source | description | event |\n|---|---|---|\n| Idle | fires `A\\|B` payloads | GO |\n"
	tbls := ParseMdTables(text)
	if len(tbls) != 1 || len(tbls[0].Rows) != 1 {
		t.Fatalf("got %+v", tbls)
	}
	row := tbls[0].Rows[0]
	if len(row) != 3 {
		t.Fatalf("cells shifted: %q", row)
	}
	if row[1] != "fires `A|B` payloads" {
		t.Errorf("escaped pipe not unescaped: %q", row[1])
	}
	if row[2] != "GO" {
		t.Errorf("event cell shifted: %q", row[2])
	}
}

func TestIsUpperFirstUnicode(t *testing.T) {
	// IR-F11: non-ASCII uppercase must count as uppercase.
	cases := map[string]bool{
		"A": true, "a": false, "Éveil": true, "éveil": false,
		"1x": false, "": false, "_x": false,
	}
	for in, want := range cases {
		if got := IsUpperFirst(in); got != want {
			t.Errorf("IsUpperFirst(%q)=%v want %v", in, got, want)
		}
	}
}

func TestFindColMatchesLabelsNotSubstrings(t *testing.T) {
	// IR-F26: "resource" is not a "source" column and "retarget" is not a
	// "target" column; a label matches whole-word at the cell start only.
	header := []string{"resource", "event log", "guard rails", "retarget", "actions needed"}
	if i := FindCol(header, "source"); i != -1 {
		t.Errorf("FindCol(source)=%d want -1", i)
	}
	if i := FindCol(header, "target"); i != -1 {
		t.Errorf("FindCol(target)=%d want -1", i)
	}
	if i := FindCol(header, "actions"); i != 4 {
		t.Errorf("FindCol(actions)=%d want 4", i)
	}
	real := []string{"#", "source", "event / after / always", "guard", "target", "actions", "derived-from"}
	for name, want := range map[string]int{"source": 1, "event": 2, "guard": 3, "target": 4, "actions": 5} {
		if i := FindCol(real, name); i != want {
			t.Errorf("FindCol(%s)=%d want %d", name, i, want)
		}
	}
	// annotated cells still match after parenthetical stripping
	if i := FindCol([]string{"event", "payload (Modelith attributes)"}, "payload"); i != 1 {
		t.Errorf("FindCol(payload)=%d want 1", i)
	}
	if i := FindCol([]string{"invariant", "test id(s)"}, "test id"); i != 1 {
		t.Errorf("FindCol(test id)=%d want 1", i)
	}
}

func TestDuplicateJSONKeysAreError(t *testing.T) {
	// IR-F27: duplicate keys last-wins silently dropped the first definition.
	_, err := LoadMachineJSONStr("t", `{"states":{"Idle":{"on":{"GO":{"target":"Done","guard":"guardA"}}},"Idle":{"on":{"GO":{"target":"Done"}}},"Done":{"type":"final"}}}`)
	if err == nil || !strings.Contains(err.Error(), "duplicate key 'Idle'") {
		t.Fatalf("expected duplicate-key error, got %v", err)
	}
}

func TestDuplicateJSONKeysErrorViaFile(t *testing.T) {
	d := t.TempDir()
	p := d + "/Dup.machine.json"
	if err := writeFile(p, `{"id":"m","id":"n","states":{"A":{"type":"final"}}}`); err != nil {
		t.Fatal(err)
	}
	_, err := LoadMachineJSON(p)
	if err == nil || !strings.Contains(err.Error(), "invalid JSON") || !strings.Contains(err.Error(), "duplicate key 'id'") {
		t.Fatalf("expected wrapped duplicate-key error, got %v", err)
	}
}

func TestDumpNullVsEmpty(t *testing.T) {
	m := mustJSON(t, `{"id":"m","initial":"A","states":{
		"A":{"on":{"x":[{"target":"B","guard":"g"},{"actions":"y"}]}},
		"B":{"type":"final"}}}`)
	out, _ := DumpJSON(m)
	// second branch has no target -> must serialize target: null
	if !strings.Contains(out, `"target": null`) {
		t.Fatalf("absent target must be null, got:\n%s", out)
	}
	if !strings.Contains(out, `"guard": null`) {
		t.Fatalf("absent guard must be null, got:\n%s", out)
	}
}
