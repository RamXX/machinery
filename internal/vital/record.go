package vital

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"

	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/gen"
)

func init() { gen.Extra = append(gen.Extra, Records) }

type recordDoc struct {
	Version    int           `json:"version"`
	Area       string        `json:"area"`
	Why        string        `json:"why"`
	Source     string        `json:"source"`
	Properties []recProperty `json:"properties"`
	Core       recCore       `json:"core"`
	Assumes    []recAssume   `json:"assumes"`
	Over       []recOver     `json:"over"`
}

type recProperty struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Statement string   `json:"statement"`
	Over      []string `json:"over"`
	Status    string   `json:"status"`
}

type recCore struct {
	Paths []string  `json:"paths"`
	Entry []string  `json:"entry"`
	Files []recFile `json:"files"`
}

type recFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type recAssume struct {
	Name      string   `json:"name"`
	Statement string   `json:"statement"`
	Evidence  []string `json:"evidence"`
	Status    string   `json:"status"`
}

type recOver struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9-]+$`)

// RecordPath is the path of the property record of a vital area (section 23.4).
func RecordPath(d *design.Design, area string) string {
	return gen.Dir(d) + "/vital/" + area + ".json"
}

// Records renders the property record of every vital area (section 23.4).
// Statuses come from committed files only: the owed list decides.
func Records(d *design.Design) ([]gen.File, error) {
	var files []gen.File
	for _, a := range d.VitalAreas() {
		if !nameRe.MatchString(a.Name) {
			continue // stage risk reports the name; it is no file name
		}
		doc, err := record(d, a)
		if err != nil {
			return nil, err
		}
		data, err := gen.JSON(doc)
		if err != nil {
			return nil, err
		}
		files = append(files, gen.File{Path: RecordPath(d, a.Name), Data: data})
	}
	return files, nil
}

func record(d *design.Design, a design.Area) (recordDoc, error) {
	doc := recordDoc{Version: 1, Area: a.Name, Why: a.Why, Source: d.P.Cfg.Design.Risk,
		Properties: []recProperty{}, Assumes: []recAssume{}, Over: []recOver{},
		Core: recCore{Paths: []string{}, Entry: []string{}, Files: []recFile{}}}
	for _, pr := range a.Property {
		doc.Properties = append(doc.Properties, recProperty{ID: design.ProofID(a.Name, pr.Name), Name: pr.Name,
			Kind: pr.Kind, Statement: pr.Statement, Over: nonNil(pr.Over), Status: design.AwaitingProof})
	}
	if c := a.Core; c != nil {
		doc.Core.Paths, doc.Core.Entry = nonNil(c.Paths), nonNil(c.Entry)
		matched, err := d.P.Glob(c.Paths)
		if err != nil {
			return doc, err
		}
		for _, f := range matched {
			data, err := d.P.Read(f)
			if err != nil {
				return doc, err
			}
			sum := sha256.Sum256(data)
			doc.Core.Files = append(doc.Core.Files, recFile{Path: f, SHA256: hex.EncodeToString(sum[:])})
		}
	}
	owed := d.OwedSet()
	for _, as := range a.Assumes {
		st := design.AssumedEvidenced
		for _, id := range as.Evidence {
			if _, ok := owed[id]; ok {
				st = design.AssumedUnproven
			}
		}
		doc.Assumes = append(doc.Assumes, recAssume{Name: as.Name, Statement: as.Statement, Evidence: nonNil(as.Evidence), Status: st})
	}
	for _, id := range design.OverIDs([]design.Area{a}) {
		st := "owed"
		if d.CommittedStatus(id) != "owed" {
			st = "proven " + design.ProofAwaited
		}
		doc.Over = append(doc.Over, recOver{ID: id, Status: st})
	}
	return doc, nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
