package checker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ProjectionSchemaV2 is the projection contract a manifest receives as soon as
// it includes any layer beyond the v1 set (model, invariants, relationships).
// A manifest naming only v1 layers keeps receiving SchemaVersion ("1.0")
// byte for byte; see Generate.
const ProjectionSchemaV2 = "2.0"

// v1Layers are the layers the 1.0 projection carries. A manifest whose include
// is a subset of these is rendered exactly as before projection v2 existed.
var v1Layers = map[string]bool{"model": true, "invariants": true, "relationships": true}

// RelationSpec declares one fact relation: the layer that owns it, its column
// names in tuple order, and whether its rows define the design element their
// stable_id names. The catalog is data: the JSON layers, the .facts files, the
// relations.txt index and the published schema are all derived from it.
type RelationSpec struct {
	Name    string
	Layer   string
	Columns []string
	// Defines marks a relation whose rows introduce an element. Two rows of a
	// defining layer that carry the same stable_id from different source
	// locations are a duplicate definition, reported with both locations.
	Defines bool
}

// relationCatalog is the v2 relation vocabulary in canonical order. Every
// column holds an identifier or an enumerated value, never free prose: a
// Modelith description, an invariant statement, a derived-fact reason or an
// authorization waiver reason is never projected as a fact.
//
// action_writes is omitted because Modelith states no structured
// post-condition for an action; it can be added the day the model does.
var relationCatalog = []RelationSpec{
	{Name: "entity", Layer: "model", Columns: []string{"id"}, Defines: true},
	{Name: "attr", Layer: "model", Columns: []string{"id", "entity", "datatype"}, Defines: true},
	{Name: "enum_member", Layer: "model", Columns: []string{"id", "enum", "value"}, Defines: true},

	{Name: "invariant", Layer: "invariants", Columns: []string{"id"}, Defines: true},
	{Name: "invariant_owner", Layer: "invariants", Columns: []string{"id", "entity"}},

	{Name: "action", Layer: "actions", Columns: []string{"id", "entity", "name", "actor"}, Defines: true},

	{Name: "relationship", Layer: "relationships", Columns: []string{"id", "src", "dst", "cardinality"}, Defines: true},

	{Name: "machine", Layer: "machines", Columns: []string{"id"}, Defines: true},
	{Name: "state", Layer: "machines", Columns: []string{"id", "machine", "kind"}, Defines: true},
	{Name: "transition", Layer: "machines", Columns: []string{"id", "src", "event", "dst"}, Defines: true},
	{Name: "guard_on", Layer: "machines", Columns: []string{"transition", "guard"}},
	{Name: "action_on", Layer: "machines", Columns: []string{"transition", "action"}},
	{Name: "invoke", Layer: "machines", Columns: []string{"state", "service"}},
	{Name: "context_key", Layer: "machines", Columns: []string{"machine", "key"}},

	// matrix names every machines/<X>.matrix.md by its stem, whether or not a
	// machine of the same stem exists (a contract-only record has none).
	{Name: "matrix", Layer: "matrices", Columns: []string{"id"}, Defines: true},
	{Name: "unit", Layer: "matrices", Columns: []string{"id", "machine", "name", "kind"}, Defines: true},
	{Name: "unit_clauses", Layer: "matrices", Columns: []string{"unit", "clause", "status"}},
	{Name: "unit_reads", Layer: "matrices", Columns: []string{"unit", "event", "field"}},
	{Name: "unit_values", Layer: "matrices", Columns: []string{"unit", "group", "member"}},
	{Name: "unit_derived", Layer: "matrices", Columns: []string{"unit", "fact"}},
	{Name: "unit_payload", Layer: "matrices", Columns: []string{"unit", "event", "field"}},
	{Name: "unit_writes", Layer: "matrices", Columns: []string{"unit", "fact"}},
	{Name: "unit_uses", Layer: "matrices", Columns: []string{"unit", "fact"}},
	{Name: "unit_carries", Layer: "matrices", Columns: []string{"unit", "kind", "target"}},
	// unit_produces names each Modelith action a matrix row's cascade or
	// consumer arm performs (PRODUCES{Entity.action}); a produced action owes
	// an authorization admission like a System action.
	{Name: "unit_produces", Layer: "matrices", Columns: []string{"unit", "action"}},
	// unit_declares marks each declaration group present on a unit row
	// (WRITES, USES, PRODUCES, CARRIES, VALUES, CLAUSES, READS, payload), so a rule
	// can tell WRITES{} (declared read-only) from no WRITES at all: an empty
	// group contributes no member row, only this marker.
	{Name: "unit_declares", Layer: "matrices", Columns: []string{"unit", "group"}},

	{Name: "event", Layer: "events", Columns: []string{"id", "producer"}, Defines: true},
	{Name: "event_participant", Layer: "events", Columns: []string{"id", "participant"}},
	{Name: "event_consumer", Layer: "events", Columns: []string{"id", "consumer"}},
	{Name: "event_payload_field", Layer: "events", Columns: []string{"id", "field"}},

	{Name: "c4_element", Layer: "c4", Columns: []string{"id", "kind", "parent"}, Defines: true},
	{Name: "c4_relationship", Layer: "c4", Columns: []string{"src", "dst"}},
	{Name: "boundary", Layer: "c4", Columns: []string{"id", "element", "role"}, Defines: true},
	{Name: "allowed_edge", Layer: "c4", Columns: []string{"src", "dst"}},
	{Name: "reads_row", Layer: "c4", Columns: []string{"artifact", "reader"}},
	// no_machine_waiver names each component whose persistence-and-placement
	// row carries '(no machine: <reason>)' with a reason: a contract-only
	// record. The reason is prose and is not projected; a waiver naming no
	// reason is no waiver and projects nothing.
	{Name: "no_machine_waiver", Layer: "c4", Columns: []string{"component"}},

	{Name: "admission", Layer: "authorization", Columns: []string{"subject", "capability"}, Defines: true},
	{Name: "no_authorization", Layer: "authorization", Columns: []string{"subject"}, Defines: true},

	{Name: "oracle_row", Layer: "oracles", Columns: []string{"id", "machine", "transition"}, Defines: true},

	{Name: "milestone", Layer: "milestones", Columns: []string{"id", "status"}, Defines: true},
	{Name: "dod_id", Layer: "milestones", Columns: []string{"milestone", "oracle"}},
	{Name: "slice_claim", Layer: "milestones", Columns: []string{"slice", "oracle"}},
	// packet_cites names the row key of each row:<path>#<section>#<key>
	// citation a slice makes in slices.yaml, the table row its packet carries.
	{Name: "packet_cites", Layer: "milestones", Columns: []string{"slice", "row"}},
	// milestone_says_bound and milestone_says_unbound are BUILD.md's oracle
	// binding table (a table with an oracle column and a bound-at column): a
	// row keyed by oracle id whose bound-at cell names test file paths, or
	// the literal unbound.
	{Name: "milestone_says_bound", Layer: "milestones", Columns: []string{"oracle", "path"}},
	{Name: "milestone_says_unbound", Layer: "milestones", Columns: []string{"oracle"}},
	// bound_at and test_file come from the implementation, only when a check
	// is given --impl: test_file is every test file Gt's corpus scans, and
	// bound_at(oracle, path) a committed oracle row that file binds under Gt's
	// credit rules. Paths are relative to --impl. The project command reads no
	// implementation, so it always emits both empty.
	{Name: "bound_at", Layer: "milestones", Columns: []string{"oracle", "path"}},
	{Name: "test_file", Layer: "milestones", Columns: []string{"path"}},

	{Name: "type_owner", Layer: "supersession", Columns: []string{"type", "owner"}},
	{Name: "supersedes", Layer: "supersession", Columns: []string{"new", "old"}},
	// reserved names each type a RESERVED{type:Name} contract row declares as
	// not yet defined, with the reserving row's subject.
	{Name: "reserved", Layer: "supersession", Columns: []string{"type", "row"}},
}

var relationByName = func() map[string]RelationSpec {
	m := make(map[string]RelationSpec, len(relationCatalog))
	for _, spec := range relationCatalog {
		m[spec.Name] = spec
	}
	return m
}()

// RelationCatalog returns a copy of the relation catalog in canonical order.
func RelationCatalog() []RelationSpec {
	out := make([]RelationSpec, len(relationCatalog))
	for i, spec := range relationCatalog {
		spec.Columns = append([]string(nil), spec.Columns...)
		out[i] = spec
	}
	return out
}

// LayerRelations returns the relation names a layer owns, in catalog order.
func LayerRelations(layer string) []string {
	var out []string
	for _, spec := range relationCatalog {
		if spec.Layer == layer {
			out = append(out, spec.Name)
		}
	}
	return out
}

// Source locates the design text an element was read from: a design-relative
// path with forward slashes and the 1-based line the element starts on.
type Source struct {
	Path string `json:"path"`
	Line int    `json:"line"`
}

func (s Source) String() string { return s.Path + ":" + strconv.Itoa(s.Line) }

// PortableSourcePath renders a design-relative path the one way a projection
// carries it: forward slashes only, whatever the host separator was.
func PortableSourcePath(rel string) string {
	return strings.ReplaceAll(rel, `\`, "/")
}

func (s Source) validate() error {
	if strings.TrimSpace(s.Path) == "" {
		return fmt.Errorf("source.path must be non-empty")
	}
	if strings.Contains(s.Path, `\`) {
		return fmt.Errorf("source.path %q must use forward slashes", s.Path)
	}
	if strings.HasPrefix(s.Path, "/") {
		return fmt.Errorf("source.path %q must be design-relative", s.Path)
	}
	if s.Line < 1 {
		return fmt.Errorf("source.line must be a 1-based line number, got %d", s.Line)
	}
	return nil
}

// Row is one tuple of a relation, with the element identity and location it
// was read from. Values are in the relation's column order.
type Row struct {
	StableID string
	Source   Source
	Values   []string
}

func tupleKey(values []string) string { return strings.Join(values, "\x00") }

// validSymbol reports whether a value can be a fact symbol: the tab-separated
// fact format (internal/datalog, Soufflé) has no escape, so a tab, carriage
// return or newline inside a value would change the tuple it encodes.
func validSymbol(v string) bool { return !strings.ContainsAny(v, "\t\r\n") }

// DesignFacts is every fact relation read from one design, grouped by the
// layers the design actually has. A layer is present only when its source
// artifacts exist (a design with no machines/ has no machines layer at all,
// rather than an empty one), and a present layer claims every one of its
// relations, empty or not.
type DesignFacts struct {
	layers map[string]bool
	rows   map[string][]Row
	tuples map[string]map[string]bool
	defs   map[string]Source
}

// NewDesignFacts returns an empty fact set with no layers.
func NewDesignFacts() *DesignFacts {
	return &DesignFacts{layers: map[string]bool{}, rows: map[string][]Row{}, tuples: map[string]map[string]bool{}, defs: map[string]Source{}}
}

// MarkLayer records that the design has a layer, even when it yields no rows.
func (f *DesignFacts) MarkLayer(layer string) error {
	if !knownLayers[layer] || unsupportedLayers[layer] {
		return fmt.Errorf("layer %q is not a projectable layer", layer)
	}
	f.layers[layer] = true
	return nil
}

// HasLayer reports whether the design has a layer.
func (f *DesignFacts) HasLayer(layer string) bool { return f != nil && f.layers[layer] }

// Layers returns the present layers in canonical order.
func (f *DesignFacts) Layers() []string {
	var out []string
	for _, layer := range includeOrder {
		if f.layers[layer] {
			out = append(out, layer)
		}
	}
	return out
}

// Add records one tuple. The relation's layer is marked present. An identical
// tuple is recorded once (the first source wins), because a relation is a set.
// A defining relation rejects a stable_id already defined in the same layer
// from a different source, naming both locations.
func (f *DesignFacts) Add(relation, stableID string, src Source, values ...string) error {
	spec, ok := relationByName[relation]
	if !ok {
		return fmt.Errorf("unknown relation %q", relation)
	}
	if len(values) != len(spec.Columns) {
		return fmt.Errorf("%s: relation %s has %d columns, got %d values", src, relation, len(spec.Columns), len(values))
	}
	if strings.TrimSpace(stableID) == "" {
		return fmt.Errorf("%s: relation %s row has an empty stable_id", src, relation)
	}
	if err := src.validate(); err != nil {
		return fmt.Errorf("relation %s row %s: %w", relation, stableID, err)
	}
	for i, v := range append([]string{stableID}, values...) {
		if !validSymbol(v) {
			column := "stable_id"
			if i > 0 {
				column = spec.Columns[i-1]
			}
			return fmt.Errorf("%s: relation %s column %s value %q contains a tab, carriage return or newline, which a fact symbol cannot hold", src, relation, column, v)
		}
	}
	f.layers[spec.Layer] = true
	if spec.Defines {
		key := spec.Layer + "\x00" + stableID
		if prior, exists := f.defs[key]; exists && prior != src {
			return fmt.Errorf("duplicate stable id %q: defined at %s and at %s", stableID, prior, src)
		}
		if _, exists := f.defs[key]; !exists {
			f.defs[key] = src
		}
	}
	key := tupleKey(values)
	if f.tuples[relation] == nil {
		f.tuples[relation] = map[string]bool{}
	}
	if f.tuples[relation][key] {
		return nil
	}
	f.tuples[relation][key] = true
	f.rows[relation] = append(f.rows[relation], Row{StableID: stableID, Source: src, Values: append([]string(nil), values...)})
	return nil
}

// Rows returns a relation's rows sorted column by column (bytewise), a copy.
func (f *DesignFacts) Rows(relation string) []Row {
	rows := append([]Row(nil), f.rows[relation]...)
	sortRows(rows)
	return rows
}

func sortRows(rows []Row) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i].Values, rows[j].Values
		for k := 0; k < len(a) && k < len(b); k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return len(a) < len(b)
	})
}

// RelationsIndexFile is the index written beside the .facts files.
const RelationsIndexFile = "relations.txt"

// FactsFiles renders the --facts directory contents: one <relation>.facts per
// relation of every present layer (tab-separated, one tuple per line, lines
// sorted bytewise, an empty file for an empty relation) plus relations.txt,
// which lists each emitted relation as name, arity, layer and comma-joined
// column names, tab-separated, in catalog order. The encoding is exactly the
// one internal/datalog.ParseFacts and Soufflé read.
func (f *DesignFacts) FactsFiles() (map[string][]byte, error) {
	out := map[string][]byte{}
	var index bytes.Buffer
	for _, spec := range relationCatalog {
		if !f.layers[spec.Layer] {
			continue
		}
		lines := make([]string, 0, len(f.rows[spec.Name]))
		for _, row := range f.rows[spec.Name] {
			for _, v := range row.Values {
				if !validSymbol(v) {
					return nil, fmt.Errorf("relation %s holds a value with a tab, carriage return or newline", spec.Name)
				}
			}
			lines = append(lines, strings.Join(row.Values, "\t"))
		}
		sort.Strings(lines)
		var body bytes.Buffer
		for _, line := range lines {
			body.WriteString(line)
			body.WriteByte('\n')
		}
		out[spec.Name+".facts"] = body.Bytes()
		fmt.Fprintf(&index, "%s\t%d\t%s\t%s\n", spec.Name, len(spec.Columns), spec.Layer, strings.Join(spec.Columns, ","))
	}
	out[RelationsIndexFile] = index.Bytes()
	return out, nil
}

// ProjLayers is the v2 relational body of a projection: for each included
// layer, every relation it owns with its rows. It marshals in canonical layer
// and catalog order, and each row as {"stable_id", "source", <columns>...}.
type ProjLayers struct {
	order     []string
	relations map[string]map[string][]Row
}

func newProjLayers() *ProjLayers {
	return &ProjLayers{relations: map[string]map[string][]Row{}}
}

func (l *ProjLayers) add(layer string, facts *DesignFacts) {
	l.order = append(l.order, layer)
	rels := map[string][]Row{}
	for _, name := range LayerRelations(layer) {
		rows := facts.Rows(name)
		if rows == nil {
			rows = []Row{}
		}
		rels[name] = rows
	}
	l.relations[layer] = rels
}

// Layers returns the layer names carried, in canonical order.
func (l *ProjLayers) Layers() []string { return append([]string(nil), l.order...) }

// Rows returns the rows of one relation of one layer.
func (l *ProjLayers) Rows(layer, relation string) []Row {
	return append([]Row(nil), l.relations[layer][relation]...)
}

// MarshalJSON writes the layers in canonical order with rows in column order.
func (l *ProjLayers) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, layer := range l.order {
		if i > 0 {
			b.WriteByte(',')
		}
		writeJSONString(&b, layer)
		b.WriteString(":{")
		for j, name := range LayerRelations(layer) {
			if j > 0 {
				b.WriteByte(',')
			}
			writeJSONString(&b, name)
			b.WriteString(":[")
			spec := relationByName[name]
			for k, row := range l.relations[layer][name] {
				if k > 0 {
					b.WriteByte(',')
				}
				b.WriteString(`{"stable_id":`)
				writeJSONString(&b, row.StableID)
				b.WriteString(`,"source":{"path":`)
				writeJSONString(&b, row.Source.Path)
				b.WriteString(`,"line":`)
				b.WriteString(strconv.Itoa(row.Source.Line))
				b.WriteByte('}')
				for c, column := range spec.Columns {
					b.WriteByte(',')
					writeJSONString(&b, column)
					b.WriteByte(':')
					writeJSONString(&b, row.Values[c])
				}
				b.WriteByte('}')
			}
			b.WriteByte(']')
		}
		b.WriteByte('}')
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

func writeJSONString(b *bytes.Buffer, s string) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	b.Write(bytes.TrimRight(buf.Bytes(), "\n"))
}

// UnmarshalJSON reads layers back strictly: known layers only, every relation
// of a layer present as an array, every row carrying exactly stable_id,
// source and the relation's columns as strings.
func (l *ProjLayers) UnmarshalJSON(data []byte) error {
	var raw map[string]map[string][]map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("layers: %w", err)
	}
	if raw == nil {
		return fmt.Errorf("layers must be a JSON object")
	}
	out := newProjLayers()
	for _, layer := range includeOrder {
		rels, present := raw[layer]
		if !present {
			continue
		}
		if rels == nil {
			return fmt.Errorf("layers.%s must be an object", layer)
		}
		out.order = append(out.order, layer)
		out.relations[layer] = map[string][]Row{}
		names := LayerRelations(layer)
		for name := range rels {
			if spec, ok := relationByName[name]; !ok || spec.Layer != layer {
				return fmt.Errorf("layers.%s names unknown relation %q", layer, name)
			}
		}
		for _, name := range names {
			rawRows, ok := rels[name]
			if !ok || rawRows == nil {
				return fmt.Errorf("layers.%s.%s must be present as an array", layer, name)
			}
			spec := relationByName[name]
			rows := make([]Row, 0, len(rawRows))
			for i, fields := range rawRows {
				where := fmt.Sprintf("layers.%s.%s[%d]", layer, name, i)
				if len(fields) != len(spec.Columns)+2 {
					return fmt.Errorf("%s must carry exactly stable_id, source and %s", where, strings.Join(spec.Columns, ", "))
				}
				var row Row
				if err := json.Unmarshal(fields["stable_id"], &row.StableID); err != nil {
					return fmt.Errorf("%s.stable_id: %w", where, err)
				}
				if err := decodeStrictJSON(fields["source"], &row.Source); err != nil {
					return fmt.Errorf("%s.source: %w", where, err)
				}
				for _, column := range spec.Columns {
					rawValue, ok := fields[column]
					if !ok {
						return fmt.Errorf("%s is missing column %q", where, column)
					}
					var v string
					if err := json.Unmarshal(rawValue, &v); err != nil {
						return fmt.Errorf("%s.%s must be a string: %w", where, column, err)
					}
					row.Values = append(row.Values, v)
				}
				rows = append(rows, row)
			}
			out.relations[layer][name] = rows
		}
	}
	for layer := range raw {
		if !knownLayers[layer] {
			return fmt.Errorf("layers names unknown layer %q", layer)
		}
	}
	*l = *out
	return nil
}

// validate holds the v2 layers to their contract against the include list.
func (l *ProjLayers) validate(include []string) error {
	if strings.Join(l.order, ",") != strings.Join(include, ",") {
		return fmt.Errorf("layers must carry exactly the included layers in canonical order (%s), got (%s)", strings.Join(include, ", "), strings.Join(l.order, ", "))
	}
	defs := map[string]Source{}
	for _, layer := range l.order {
		for _, name := range LayerRelations(layer) {
			spec := relationByName[name]
			rows, ok := l.relations[layer][name]
			if !ok || rows == nil {
				return fmt.Errorf("layers.%s.%s must be an array", layer, name)
			}
			seen := map[string]bool{}
			for i, row := range rows {
				where := fmt.Sprintf("layers.%s.%s[%d]", layer, name, i)
				if strings.TrimSpace(row.StableID) == "" {
					return fmt.Errorf("%s.stable_id must be non-empty", where)
				}
				if err := row.Source.validate(); err != nil {
					return fmt.Errorf("%s: %w", where, err)
				}
				if len(row.Values) != len(spec.Columns) {
					return fmt.Errorf("%s has %d values for %d columns", where, len(row.Values), len(spec.Columns))
				}
				for _, v := range append([]string{row.StableID}, row.Values...) {
					if !validSymbol(v) {
						return fmt.Errorf("%s holds a value with a tab, carriage return or newline", where)
					}
				}
				key := tupleKey(row.Values)
				if seen[key] {
					return fmt.Errorf("%s repeats a tuple; a relation is a set", where)
				}
				seen[key] = true
				if spec.Defines {
					dkey := layer + "\x00" + row.StableID
					if prior, exists := defs[dkey]; exists && prior != row.Source {
						return fmt.Errorf("duplicate stable id %q: defined at %s and at %s", row.StableID, prior, row.Source)
					}
					defs[dkey] = row.Source
				}
			}
		}
	}
	return nil
}
