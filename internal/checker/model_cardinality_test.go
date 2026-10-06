package checker

import (
	"strings"
	"testing"
)

func TestModelManyToManyV1Compatibility(t *testing.T) {
	model, err := LoadModel(writeTemp(t, "d.modelith.yaml", strings.ReplaceAll(sampleModel, "cardinality: 1:n", "cardinality: n:n")))
	if err != nil {
		t.Fatal(err)
	}
	proj, err := Generate(model, manifestWith([]string{"model", "relationships"}, nil), validTestDesignID, "test")
	if err != nil {
		t.Fatal(err)
	}
	if proj.ProjectionSchema != "1.0" || len(proj.Model.Relationships) != 1 {
		t.Fatalf("unexpected projection: %+v", proj)
	}
	rel := proj.Model.Relationships[0]
	if rel.Cardinality != "n:m" || rel.StableID != "rel:DataSubject->Export:n:m" {
		t.Fatalf("many-to-many compatibility representation: %+v", rel)
	}
	body, err := proj.Render()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseProjection(body); err != nil {
		t.Fatalf("v1 projection round trip: %v", err)
	}
}

func TestModelUnsupportedCardinalityNamesV2Projection(t *testing.T) {
	for _, card := range []string{"0:1", "0:n", "1:5", "2:10", "many"} {
		t.Run(card, func(t *testing.T) {
			_, err := LoadModel(writeTemp(t, "d.modelith.yaml", strings.ReplaceAll(sampleModel, "cardinality: 1:n", "cardinality: '"+card+"'")))
			if err == nil || !strings.Contains(err.Error(), "unsupported cardinality \""+card+"\"") || !strings.Contains(err.Error(), "v2 projection") {
				t.Fatalf("expected cardinality and v2 projection diagnostic, got %v", err)
			}
		})
	}
}
