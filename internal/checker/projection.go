package checker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// includeOrder is the canonical order of projection layers. The manifest may
// list them in any order; the projection always emits this order so the bytes
// (and the hash) are stable.
var includeOrder = []string{"model", "invariants", "actions", "relationships", "scenarios", "machines", "c4", "oracles"}

// v1 does not yet project these layers. Requesting one fails loudly rather than
// silently omitting it, so a checker never runs believing it received a layer.
var unsupportedLayers = map[string]bool{"actions": true, "scenarios": true, "machines": true, "c4": true, "oracles": true}

// Projection is the canonical design slice. Field order is fixed and every slice
// is sorted by stable_id, so encodeJSON is byte-reproducible. MachineryVersion
// and Generated are excluded from the binding hash.
type Projection struct {
	ProjectionSchema string         `json:"projection_schema"`
	MachineryVersion string         `json:"machinery_version,omitempty"`
	DesignID         string         `json:"design_id"`
	CheckerID        string         `json:"checker_id"`
	Include          []string       `json:"include"`
	Model            *ProjModel     `json:"model,omitempty"`
	Generated        map[string]any `json:"generated,omitempty"`
}

// ProjModel is the domain layer. Entities is always present when model is
// included; invariants and relationships appear only when also requested.
type ProjModel struct {
	Entities      []ProjEntity       `json:"entities"`
	Invariants    []ProjInvariant    `json:"invariants,omitempty"`
	Relationships []ProjRelationship `json:"relationships,omitempty"`
}

type ProjEntity struct {
	StableID   string     `json:"stable_id"`
	Name       string     `json:"name"`
	StatusEnum []string   `json:"status_enum,omitempty"`
	Attributes []ProjAttr `json:"attributes,omitempty"`
}

type ProjAttr struct {
	StableID string `json:"stable_id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
}

type ProjInvariant struct {
	StableID string    `json:"stable_id"`
	ID       string    `json:"id"`
	Text     string    `json:"text"`
	Owner    ProjOwner `json:"owner"`
}

type ProjOwner struct {
	Kind   string `json:"kind"`
	Entity string `json:"entity,omitempty"`
}

type ProjRelationship struct {
	StableID    string `json:"stable_id"`
	From        string `json:"from"`
	To          string `json:"to"`
	Cardinality string `json:"cardinality"`
}

func entityID(name string) string { return "entity:" + name }

// Generate builds the projection the manifest asks for from the model. It fails
// on any include layer v1 does not support rather than omitting it.
func Generate(m *Model, man *Manifest, designID, machineryVersion string) (*Projection, error) {
	need := setOf(man.Projection.Include)
	for layer := range need {
		if unsupportedLayers[layer] {
			return nil, fmt.Errorf("projection include layer %q is not yet supported (v1 supports model, invariants, relationships)", layer)
		}
	}

	p := &Projection{
		ProjectionSchema: SchemaVersion,
		MachineryVersion: machineryVersion,
		DesignID:         designID,
		CheckerID:        man.Checker.ID,
		Include:          canonicalInclude(need),
	}

	if need["model"] || need["invariants"] || need["relationships"] {
		pm := &ProjModel{}
		if need["model"] {
			for _, e := range m.Entities {
				pe := ProjEntity{StableID: entityID(e.Name), Name: e.Name, StatusEnum: e.StatusEnum}
				for _, a := range e.Attributes {
					pe.Attributes = append(pe.Attributes, ProjAttr{
						StableID: "attr:" + e.Name + "." + a.Name,
						Name:     a.Name,
						Type:     a.Type,
					})
				}
				sort.Slice(pe.Attributes, func(i, j int) bool { return pe.Attributes[i].StableID < pe.Attributes[j].StableID })
				pm.Entities = append(pm.Entities, pe)
			}
			sort.Slice(pm.Entities, func(i, j int) bool { return pm.Entities[i].StableID < pm.Entities[j].StableID })
		}
		if need["invariants"] {
			for _, iv := range m.Invariants {
				owner := ProjOwner{Kind: iv.OwnerKind}
				if iv.OwnerKind == "entity" {
					owner.Entity = entityID(iv.Owner)
				}
				pm.Invariants = append(pm.Invariants, ProjInvariant{
					StableID: "inv:" + iv.ID,
					ID:       iv.ID,
					Text:     iv.Statement,
					Owner:    owner,
				})
			}
			sort.Slice(pm.Invariants, func(i, j int) bool { return pm.Invariants[i].StableID < pm.Invariants[j].StableID })
		}
		if need["relationships"] {
			for _, r := range m.Relationships {
				pm.Relationships = append(pm.Relationships, ProjRelationship{
					StableID:    fmt.Sprintf("rel:%s->%s:%s", r.From, r.To, r.Cardinality),
					From:        entityID(r.From),
					To:          entityID(r.To),
					Cardinality: r.Cardinality,
				})
			}
			sort.Slice(pm.Relationships, func(i, j int) bool { return pm.Relationships[i].StableID < pm.Relationships[j].StableID })
		}
		p.Model = pm
	}
	return p, nil
}

func canonicalInclude(need map[string]bool) []string {
	var out []string
	for _, layer := range includeOrder {
		if need[layer] {
			out = append(out, layer)
		}
	}
	return out
}

// bindingBytes is the canonical byte string the input hash covers: version and
// generated nulled, compact, HTML-escaping off. Two projections with the same
// domain content produce identical bindingBytes.
func (p *Projection) bindingBytes() ([]byte, error) {
	c := *p
	c.MachineryVersion = ""
	c.Generated = nil
	return encodeJSON(&c, false)
}

// InputHash is the sha256 over bindingBytes. Evidence must echo it to bind its
// verdict to this exact design.
func (p *Projection) InputHash() (string, error) {
	b, err := p.bindingBytes()
	if err != nil {
		return "", err
	}
	return sha256Prefixed(b), nil
}

// Render returns the committed on-disk bytes: pretty, trailing newline, with the
// input_hash mirrored under generated as a convenience for adapters. The mirror
// never participates in a gate check; the gate always recomputes InputHash.
func (p *Projection) Render() ([]byte, error) {
	h, err := p.InputHash()
	if err != nil {
		return nil, err
	}
	c := *p
	c.Generated = map[string]any{"input_hash": h}
	b, err := encodeJSON(&c, true)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// ParseProjection reads a committed projection file back into a Projection.
func ParseProjection(b []byte) (*Projection, error) {
	var p Projection
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ContentEqual reports whether two projections carry identical domain content,
// ignoring machinery_version and generated. The freshness (DRIFT) check.
func ContentEqual(a, b *Projection) (bool, error) {
	ab, err := a.bindingBytes()
	if err != nil {
		return false, err
	}
	bb, err := b.bindingBytes()
	if err != nil {
		return false, err
	}
	return bytes.Equal(ab, bb), nil
}
