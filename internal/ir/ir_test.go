package ir

import (
	"strings"
	"testing"
)

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
		"`foo`":           "foo",
		"`foo` (internal)": "foo",
		"`a` (final)":     "a",
		"-":               "-",
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
