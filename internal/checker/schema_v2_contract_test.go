package checker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The published 2.0 schema is generated from the relation catalog; this holds
// the two to each other so neither can drift alone.
func TestPublishedProjectionV2SchemaMatchesTheRelationCatalog(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "schemas", "projection-v2.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	properties := schema["properties"].(map[string]any)
	if properties["projection_schema"].(map[string]any)["const"] != ProjectionSchemaV2 {
		t.Fatal("the v2 schema must pin projection_schema 2.0")
	}
	var supported []string
	for _, layer := range includeOrder {
		if !unsupportedLayers[layer] {
			supported = append(supported, layer)
		}
	}
	include := properties["include"].(map[string]any)["items"].(map[string]any)
	assertStringSet(t, stringsFromAny(include["enum"].([]any)), supported)
	layers := properties["layers"].(map[string]any)["properties"].(map[string]any)
	assertStringSet(t, objectKeys(layers), supported)
	description := schema["description"].(string)
	const opener = "defining relation ("
	at := strings.Index(description, opener)
	if at < 0 {
		t.Fatal("the schema description must list the defining relations")
	}
	list := description[at+len(opener):]
	list = list[:strings.Index(list, ")")]
	defining := map[string]bool{}
	for _, name := range strings.Split(list, ", ") {
		defining[name] = true
	}
	for _, spec := range RelationCatalog() {
		layer := layers[spec.Layer].(map[string]any)
		relation, ok := layer["properties"].(map[string]any)[spec.Name].(map[string]any)
		if !ok {
			t.Fatalf("schema layer %s lacks relation %s", spec.Layer, spec.Name)
		}
		items := relation["items"].(map[string]any)
		want := append([]string{"stable_id", "source"}, spec.Columns...)
		got := stringsFromAny(items["required"].([]any))
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("schema %s.%s required = %v, want %v", spec.Layer, spec.Name, got, want)
		}
		assertStringSet(t, objectKeys(items["properties"].(map[string]any)), want)
		if spec.Defines != defining[spec.Name] {
			t.Fatalf("schema description must list %s as defining=%v", spec.Name, spec.Defines)
		}
	}
	for layer, body := range layers {
		if n := len(body.(map[string]any)["properties"].(map[string]any)); n != len(LayerRelations(layer)) {
			t.Fatalf("schema layer %s carries %d relations, the catalog %d", layer, n, len(LayerRelations(layer)))
		}
	}
}
