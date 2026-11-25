package shipping

import (
	"encoding/json"
	"errors"
	"os"
)

// Store keeps shipments in one JSON file.
type Store struct{ path string }

// OpenStore returns a store over the file. The file need not exist yet.
func OpenStore(path string) *Store { return &Store{path: path} }

type document struct {
	Shipments map[string]*Shipment `json:"shipments"`
}

func (st *Store) load() (*document, error) {
	doc := &document{Shipments: map[string]*Shipment{}}
	data, err := os.ReadFile(st.path)
	if errors.Is(err, os.ErrNotExist) {
		return doc, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, doc); err != nil {
		return nil, err
	}
	if doc.Shipments == nil {
		doc.Shipments = map[string]*Shipment{}
	}
	return doc, nil
}

// Get returns the shipment with the id.
func (st *Store) Get(id string) (*Shipment, error) {
	doc, err := st.load()
	if err != nil {
		return nil, err
	}
	s, ok := doc.Shipments[id]
	if !ok {
		return nil, refuse("SHIPMENT_NOT_FOUND", "no shipment %s", id)
	}
	return s, nil
}

// Add stores a new shipment and refuses a duplicate id.
func (st *Store) Add(s *Shipment) error {
	doc, err := st.load()
	if err != nil {
		return err
	}
	if _, ok := doc.Shipments[s.ID]; ok {
		return refuse("SHIPMENT_EXISTS", "shipment %s already exists", s.ID)
	}
	doc.Shipments[s.ID] = s
	return st.save(doc)
}

// Update replaces a stored shipment.
func (st *Store) Update(s *Shipment) error {
	doc, err := st.load()
	if err != nil {
		return err
	}
	doc.Shipments[s.ID] = s
	return st.save(doc)
}

func (st *Store) save(doc *document) error {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(st.path, append(data, '\n'), 0o644)
}
