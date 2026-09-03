package checker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestPublishedProjectionSchemaMatchesImplementedV1Surface(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "schemas", "projection.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	properties := schema["properties"].(map[string]any)
	assertStringSet(t, objectKeys(properties), []string{"checker_id", "design_id", "generated", "include", "machinery_version", "manifest_hash", "model", "projection_schema"})
	include := properties["include"].(map[string]any)
	items := include["items"].(map[string]any)
	assertStringSet(t, stringsFromAny(items["enum"].([]any)), []string{"invariants", "model", "relationships"})
	model := properties["model"].(map[string]any)
	assertStringSet(t, objectKeys(model["properties"].(map[string]any)), []string{"entities", "invariants", "relationships"})

	defs := schema["$defs"].(map[string]any)
	assertStringSet(t, objectKeys(defs), []string{"attribute", "entity", "invariant", "relationship", "stableId"})
}

func TestPublishedEvidenceSchemaRejectsReservedUnimplementedSignature(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "schemas", "evidence.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	properties := schema["properties"].(map[string]any)
	if _, exists := properties["input_signature"]; exists {
		t.Fatal("published evidence schema advertises unsupported input_signature")
	}
}

func objectKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func stringsFromAny(values []any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.(string))
	}
	return out
}

func assertStringSet(t *testing.T, got, want []string) {
	t.Helper()
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("schema surface mismatch: got %v want %v", got, want)
	}
}
