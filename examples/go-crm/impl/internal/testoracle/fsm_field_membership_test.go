package testoracle

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// TestFSMParseExactFieldMembership distinguishes structural field names from
// arbitrary data keys in the already declared metadata and context positions.
func TestFSMParseExactFieldMembership(t *testing.T) {
	md, err := os.ReadFile("../../../design/machines/Session.oracle.md")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../../design/machines/Session.machine.json")
	if err != nil {
		t.Fatal(err)
	}
	control, err := Parse(md, raw)
	if err != nil || control == nil {
		t.Fatalf("committed Session control: suite=%v error=%v", control, err)
	}
	if control.Name != "session" || len(control.Rows) != 60 {
		t.Fatalf("committed Session identity/count: name=%q rows=%d", control.Name, len(control.Rows))
	}
	checkUnchanged := func(t *testing.T, machine []byte) {
		t.Helper()
		got, err := Parse(md, machine)
		if err != nil || got == nil {
			t.Fatalf("valid data rejected: suite=%v error=%v", got, err)
		}
		if got.Name != control.Name || !reflect.DeepEqual(got.Rows, control.Rows) || !reflect.DeepEqual(got.States, control.States) {
			t.Fatalf("data changed parsed name, full row identities/ordered actions, or state declarations: got=%+v want=%+v", got, control)
		}
	}
	t.Run("committed_control", func(t *testing.T) { checkUnchanged(t, raw) })
	t.Run("remarshaled_control", func(t *testing.T) {
		checkUnchanged(t, fieldMembershipJSON(t, fieldMembershipObject(t, raw)))
	})

	type fieldCase struct {
		scope string
		path  []string
		key   string
	}
	statePath := []string{"states", "Anonymous"}
	transitionPath := []string{"states", "Anonymous", "on", "login"}
	invokePath := []string{"states", "Authenticating", "invoke"}
	cases := []fieldCase{
		{"root", nil, "id initial"},
		{"root", nil, "_comment _role"},
		{"root", nil, "context states"},
		{"state", statePath, "on after"},
		{"state", statePath, "type entry"},
		{"state", statePath, "invoke _refusal"},
		{"transition", transitionPath, "target guard"},
		{"transition", transitionPath, "guard actions"},
		{"invoke", invokePath, "src input"},
		{"invoke", invokePath, "onDone onError"},
		{"root", nil, "notAllowed"},
		{"state", statePath, "notAllowed"},
		{"transition", transitionPath, "notAllowed"},
		{"invoke", invokePath, "notAllowed"},
	}
	for _, tc := range cases {
		t.Run("unknown/"+tc.scope+"/"+tc.key, func(t *testing.T) {
			// A fresh committed copy and exactly one insertion isolate every case.
			doc := fieldMembershipObject(t, raw)
			target := fieldMembershipTarget(t, doc, tc.path)
			if _, exists := target[tc.key]; exists {
				t.Fatalf("unknown key %q already exists at %v", tc.key, tc.path)
			}
			value := map[string]any{"unexpectedSemanticData": true}
			target[tc.key] = value
			mutated := fieldMembershipJSON(t, doc)
			inserted := fieldMembershipTarget(t, fieldMembershipObject(t, mutated), tc.path)
			if !reflect.DeepEqual(inserted[tc.key], value) {
				t.Fatalf("unknown field %q was not inserted at %v", tc.key, tc.path)
			}
			got, err := Parse(md, mutated)
			if err == nil {
				rows := 0
				if got != nil {
					rows = len(got.Rows)
				}
				t.Fatalf("unsafe unknown field accepted: scope=%s key=%q rows=%d error=nil", tc.scope, tc.key, rows)
			}
			if got != nil {
				t.Errorf("unknown field %q returned nonnil suite", tc.key)
			}
			if want := "oracle-parse: unknown field " + tc.key; err.Error() != want {
				t.Errorf("unknown field diagnostic: got=%q want=%q", err.Error(), want)
			}
		})
	}

	// The same names are legitimate data keys here. Expression-like strings
	// deliberately refer to missing data: parsing must not evaluate them.
	data := map[string]any{}
	metadata := map[string]any{}
	for _, tc := range cases {
		data[tc.key] = []any{nil, true, float64(7), "data", map[string]any{"states": nil, "target guard": []any{}}}
		metadata[tc.key] = "context.missing.value + event.notPresent()"
	}
	data["states"] = map[string]any{"invoke": map[string]any{"src input": nil}}
	valid := []struct {
		name  string
		path  []string
		key   string
		value any
	}{
		{"comment", nil, "_comment", "context.missing.value + event.notPresent()"},
		{"role", nil, "_role", "arbitrary data, not a structural role identifier"},
		{"delays", nil, "_delays", metadata},
		{"counters", nil, "_counters", metadata},
		{"refusal", statePath, "_refusal", metadata},
		{"invoke_input", invokePath, "input", metadata},
		{"context", nil, "context", data},
	}
	for _, tc := range valid {
		t.Run("valid_data/"+tc.name, func(t *testing.T) {
			doc := fieldMembershipObject(t, raw)
			fieldMembershipTarget(t, doc, tc.path)[tc.key] = tc.value
			checkUnchanged(t, fieldMembershipJSON(t, doc))
		})
	}
	t.Run("valid_data/combined", func(t *testing.T) {
		doc := fieldMembershipObject(t, raw)
		for _, tc := range valid {
			fieldMembershipTarget(t, doc, tc.path)[tc.key] = tc.value
		}
		checkUnchanged(t, fieldMembershipJSON(t, doc))
	})
}

func fieldMembershipObject(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil || doc == nil {
		t.Fatalf("fixture must be a JSON object: %v", err)
	}
	return doc
}

func fieldMembershipTarget(t *testing.T, doc map[string]any, path []string) map[string]any {
	t.Helper()
	for _, key := range path {
		value, exists := doc[key]
		if !exists {
			t.Fatalf("fixture target %v missing %q", path, key)
		}
		next, ok := value.(map[string]any)
		if !ok || next == nil {
			t.Fatalf("fixture target %v at %q is %T, want object", path, key, value)
		}
		doc = next
	}
	return doc
}

func fieldMembershipJSON(t *testing.T, doc map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
