package checker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// includeOrder is the canonical order of projection layers. The manifest may
// list them in any order; the projection always emits this order so the bytes
// (and the hash) are stable.
var includeOrder = []string{"model", "invariants", "actions", "relationships", "scenarios", "machines", "c4", "oracles"}

var knownLayers = func() map[string]bool {
	m := make(map[string]bool, len(includeOrder))
	for _, layer := range includeOrder {
		m[layer] = true
	}
	return m
}()

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
	ManifestHash     string         `json:"manifest_hash"`
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
	Name        string `json:"name,omitempty"`
}

func entityID(name string) string { return "entity:" + name }

// Generate builds the projection the manifest asks for from the model. It fails
// on any include layer v1 does not support rather than omitting it.
func Generate(m *Model, man *Manifest, designID, machineryVersion string) (*Projection, error) {
	need := setOf(man.Projection.Include)
	if len(need) == 0 {
		return nil, fmt.Errorf("projection include must name at least one layer")
	}
	var unknown []string
	for layer := range need {
		if !knownLayers[layer] {
			unknown = append(unknown, layer)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("projection include layer %q is unknown", unknown[0])
	}
	for _, layer := range includeOrder {
		if need[layer] && unsupportedLayers[layer] {
			return nil, fmt.Errorf("projection include layer %q is not yet supported (v1 supports model, invariants, relationships)", layer)
		}
	}

	manifestHash, err := man.SemanticHash()
	if err != nil {
		return nil, err
	}
	p := &Projection{
		ProjectionSchema: SchemaVersion,
		MachineryVersion: machineryVersion,
		DesignID:         designID,
		CheckerID:        man.Checker.ID,
		ManifestHash:     manifestHash,
		Include:          canonicalInclude(need),
	}

	if need["model"] || need["invariants"] || need["relationships"] {
		// entities is required by the projection schema whenever model exists.
		// Keep it an empty JSON array for invariant-only/relationship-only
		// projections; a nil slice would serialize as null and violate the
		// contract consumed by external checkers.
		pm := &ProjModel{Entities: []ProjEntity{}}
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
			seenRelationshipIDs := map[string]bool{}
			for _, r := range m.Relationships {
				stableID := fmt.Sprintf("rel:%s->%s:%s", r.From, r.To, r.Cardinality)
				if r.Name != "" {
					stableID += ":" + r.Name
				}
				if seenRelationshipIDs[stableID] {
					return nil, fmt.Errorf("relationship stable identity collision %q; parallel relationships need distinct role/name values", stableID)
				}
				seenRelationshipIDs[stableID] = true
				pm.Relationships = append(pm.Relationships, ProjRelationship{
					StableID:    stableID,
					From:        entityID(r.From),
					To:          entityID(r.To),
					Cardinality: r.Cardinality,
					Name:        r.Name,
				})
			}
			sort.Slice(pm.Relationships, func(i, j int) bool { return pm.Relationships[i].StableID < pm.Relationships[j].StableID })
		}
		p.Model = pm
	}
	if err := p.validate(); err != nil {
		return nil, fmt.Errorf("generated projection violates its schema: %w", err)
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
	if err := p.validate(); err != nil {
		return nil, fmt.Errorf("projection violates its schema: %w", err)
	}
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
	if err := decodeStrictJSON(b, &p); err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		return nil, err
	}
	if generated, present := fields["generated"]; present {
		trimmed := bytes.TrimSpace(generated)
		if len(trimmed) == 0 || trimmed[0] != '{' {
			return nil, fmt.Errorf("generated must be a JSON object")
		}
	}
	if modelRaw, present := fields["model"]; present {
		var modelFields map[string]json.RawMessage
		if err := json.Unmarshal(modelRaw, &modelFields); err != nil {
			return nil, fmt.Errorf("model must be a JSON object: %w", err)
		}
		for _, arrayField := range []string{"entities", "invariants", "relationships"} {
			if raw, present := modelFields[arrayField]; present {
				trimmed := bytes.TrimSpace(raw)
				if len(trimmed) == 0 || trimmed[0] != '[' {
					return nil, fmt.Errorf("model.%s must be a JSON array", arrayField)
				}
			}
		}
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

func (p *Projection) validate() error {
	if p.ProjectionSchema != SchemaVersion {
		return fmt.Errorf("projection_schema must be %q, got %q", SchemaVersion, p.ProjectionSchema)
	}
	if strings.TrimSpace(p.MachineryVersion) == "" {
		return fmt.Errorf("machinery_version must be non-empty")
	}
	if !validSHA256(p.DesignID) {
		return fmt.Errorf("design_id must be a lowercase sha256:<64 hex> digest")
	}
	if strings.TrimSpace(p.CheckerID) == "" {
		return fmt.Errorf("checker_id must be non-empty")
	}
	if !validSHA256(p.ManifestHash) {
		return fmt.Errorf("manifest_hash must be a lowercase sha256:<64 hex> digest")
	}
	if len(p.Include) == 0 {
		return fmt.Errorf("include must name at least one layer")
	}
	need := map[string]bool{}
	for _, layer := range p.Include {
		if !knownLayers[layer] {
			return fmt.Errorf("include names unknown layer %q", layer)
		}
		if unsupportedLayers[layer] {
			return fmt.Errorf("include names unsupported layer %q", layer)
		}
		if need[layer] {
			return fmt.Errorf("include names layer %q more than once", layer)
		}
		need[layer] = true
	}
	canonical := canonicalInclude(need)
	for i := range canonical {
		if canonical[i] != p.Include[i] {
			return fmt.Errorf("include is not in canonical layer order")
		}
	}
	if p.Model == nil {
		return fmt.Errorf("model is required for the supported projection layers")
	}
	if p.Model.Entities == nil {
		return fmt.Errorf("model.entities must be an array, not null")
	}
	if !need["model"] && len(p.Model.Entities) != 0 {
		return fmt.Errorf("model.entities must be empty when model is not included")
	}
	if !need["invariants"] && len(p.Model.Invariants) != 0 {
		return fmt.Errorf("model.invariants is populated but invariants is not included")
	}
	if !need["relationships"] && len(p.Model.Relationships) != 0 {
		return fmt.Errorf("model.relationships is populated but relationships is not included")
	}
	ids := map[string]string{}
	addID := func(id, kind string) error {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("%s stable_id must be non-empty", kind)
		}
		if prior, exists := ids[id]; exists {
			return fmt.Errorf("duplicate stable_id %q used by %s and %s", id, prior, kind)
		}
		ids[id] = kind
		return nil
	}
	for i, entity := range p.Model.Entities {
		kind := fmt.Sprintf("model.entities[%d]", i)
		if strings.TrimSpace(entity.Name) == "" {
			return fmt.Errorf("%s.name must be non-empty", kind)
		}
		if err := addID(entity.StableID, kind); err != nil {
			return err
		}
		seenStatus := map[string]bool{}
		for j, status := range entity.StatusEnum {
			if strings.TrimSpace(status) == "" {
				return fmt.Errorf("%s.status_enum[%d] must be non-empty", kind, j)
			}
			if seenStatus[status] {
				return fmt.Errorf("%s.status_enum repeats %q", kind, status)
			}
			seenStatus[status] = true
		}
		for j, attr := range entity.Attributes {
			attrKind := fmt.Sprintf("%s.attributes[%d]", kind, j)
			if strings.TrimSpace(attr.Name) == "" || strings.TrimSpace(attr.Type) == "" {
				return fmt.Errorf("%s name and type must be non-empty", attrKind)
			}
			if err := addID(attr.StableID, attrKind); err != nil {
				return err
			}
		}
	}
	for i, invariant := range p.Model.Invariants {
		kind := fmt.Sprintf("model.invariants[%d]", i)
		if strings.TrimSpace(invariant.ID) == "" || strings.TrimSpace(invariant.Text) == "" {
			return fmt.Errorf("%s id and text must be non-empty", kind)
		}
		if invariant.Owner.Kind != "entity" && invariant.Owner.Kind != "top" {
			return fmt.Errorf("%s.owner.kind must be entity or top", kind)
		}
		if invariant.Owner.Kind == "entity" && strings.TrimSpace(invariant.Owner.Entity) == "" {
			return fmt.Errorf("%s.owner.entity is required for entity ownership", kind)
		}
		if invariant.Owner.Kind == "top" && invariant.Owner.Entity != "" {
			return fmt.Errorf("%s.owner.entity must be absent for top ownership", kind)
		}
		if err := addID(invariant.StableID, kind); err != nil {
			return err
		}
	}
	for i, relationship := range p.Model.Relationships {
		kind := fmt.Sprintf("model.relationships[%d]", i)
		if strings.TrimSpace(relationship.From) == "" || strings.TrimSpace(relationship.To) == "" || strings.TrimSpace(relationship.Cardinality) == "" {
			return fmt.Errorf("%s from, to, and cardinality must be non-empty", kind)
		}
		if err := addID(relationship.StableID, kind); err != nil {
			return err
		}
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, c := range value[len("sha256:"):] {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f':
			continue
		default:
			return false
		}
	}
	return true
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
