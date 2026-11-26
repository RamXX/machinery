// Package delegate reads the owner's delegate policy and judges an answer on
// the owner's behalf against one of its clauses (docs/formats.md section 15.9).
// The policy is the owner's file: this package only reads it.
package delegate

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/RamXX/machinery/internal/project"
)

// The effects a clause may have.
const (
	Allow  = "allow"
	Refuse = "refuse"
	Hold   = "hold"
)

// Reserved are the topics only a clause that names them may answer.
var Reserved = map[string]bool{"publish": true, "payment": true, "destructive": true, "method": true, "delegate-policy": true}

// Policy is the delegate policy file.
type Policy struct {
	Version int      `yaml:"version"`
	Owner   string   `yaml:"owner"`
	Clauses []Clause `yaml:"clauses"`
}

// Clause is what the owner lets an agent answer, and how.
type Clause struct {
	ID      string   `yaml:"id"`
	About   []string `yaml:"about"`
	Effect  string   `yaml:"effect"`
	Limit   *Limit   `yaml:"limit,omitempty"`
	Confirm bool     `yaml:"confirm,omitempty"`
	Note    string   `yaml:"note"`
}

// Limit is the most the option answers under a clause may move together.
type Limit struct {
	Amount string `yaml:"amount"`
	Unit   string `yaml:"unit"`
}

var (
	idRe     = regexp.MustCompile(`^[a-z][a-z0-9-]+$`)
	amountRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)
)

// Load reads the policy that delegate.policy names; nil when none is
// configured. A problem with the file is a config error.
func Load(p *project.Project) (*Policy, error) {
	if p.Cfg.Delegate == nil {
		return nil, nil
	}
	rel := p.Cfg.Delegate.Policy
	path := filepath.FromSlash(rel)
	if !filepath.IsAbs(path) {
		path = filepath.Join(p.Root, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			err = pe.Err
		}
		return nil, project.Errf("delegate.policy %s: cannot read: %v", rel, err)
	}
	pol, msg := Parse(data)
	if msg != "" {
		return nil, project.Errf("delegate.policy %s: %s", rel, msg)
	}
	return pol, nil
}

// Parse decodes and checks policy bytes; msg names the first problem.
func Parse(data []byte) (pol *Policy, msg string) {
	var p Policy
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil {
		return nil, "not a delegate policy: " + strings.ReplaceAll(strings.TrimPrefix(err.Error(), "yaml: "), "\n", " ")
	}
	switch {
	case p.Version != 1:
		return nil, "version must be 1"
	case strings.TrimSpace(p.Owner) == "":
		return nil, "owner is required"
	}
	seen := map[string]bool{}
	for i, c := range p.Clauses {
		switch {
		case !idRe.MatchString(c.ID):
			return nil, fmt.Sprintf("clause %d: id must match ^[a-z][a-z0-9-]+$", i+1)
		case seen[c.ID]:
			return nil, fmt.Sprintf("clause %s: duplicate id", c.ID)
		case len(c.About) == 0:
			return nil, fmt.Sprintf("clause %s: about needs at least one topic", c.ID)
		case c.Effect != Allow && c.Effect != Refuse && c.Effect != Hold:
			return nil, fmt.Sprintf("clause %s: effect must be allow, refuse or hold, got %q", c.ID, c.Effect)
		case c.Limit != nil && (!amountRe.MatchString(c.Limit.Amount) || strings.TrimSpace(c.Limit.Unit) == ""):
			return nil, fmt.Sprintf("clause %s: limit needs an amount (a number) and a unit", c.ID)
		case strings.TrimSpace(c.Note) == "":
			return nil, fmt.Sprintf("clause %s: note is required", c.ID)
		}
		seen[c.ID] = true
	}
	return &p, ""
}

// Clause returns the clause with the id, or nil.
func (p *Policy) Clause(id string) *Clause {
	for i := range p.Clauses {
		if p.Clauses[i].ID == id {
			return &p.Clauses[i]
		}
	}
	return nil
}

// Question is what a clause is judged against.
type Question struct {
	ID     string
	Topics []string
	Amount string // "" when the question moves nothing
	Unit   string
}

// Refusal says why the clause does not permit the answer, kind being "option"
// or "refuse", or "" when it does. spent returns the sum of the earlier
// option answers under the clause in the limit's unit; it is called only when
// the limit decides (section 15.9).
func (c *Clause) Refusal(q Question, kind string, spent func() *big.Rat) string {
	if len(q.Topics) == 0 {
		return fmt.Sprintf("%s names no topic: a clause covers only the topics it names", q.ID)
	}
	names := map[string]bool{}
	for _, t := range c.About {
		names[t] = true
	}
	for _, t := range q.Topics {
		if Reserved[t] && !names[t] {
			return fmt.Sprintf("%s is about %s, a reserved topic, and clause %s does not name it", q.ID, t, c.ID)
		}
	}
	for _, t := range q.Topics {
		if !names[t] && !(names["*"] && !Reserved[t]) {
			return fmt.Sprintf("clause %s does not cover %s", c.ID, t)
		}
	}
	if (kind == "option" && c.Effect != Allow) || (kind == "refuse" && c.Effect != Refuse) {
		return fmt.Sprintf("clause %s has effect %s: it does not permit --%s", c.ID, c.Effect, kind)
	}
	if kind != "option" || q.Amount == "" {
		return ""
	}
	switch {
	case c.Limit == nil:
		return fmt.Sprintf("clause %s sets no limit: an answer that moves %s %s needs one", c.ID, q.Amount, q.Unit)
	case c.Limit.Unit != q.Unit:
		return fmt.Sprintf("clause %s limits %s, the question is in %s", c.ID, c.Limit.Unit, q.Unit)
	}
	amount, _ := ParseAmount(q.Amount)
	limit, _ := ParseAmount(c.Limit.Amount)
	sp := spent()
	if new(big.Rat).Add(amount, sp).Cmp(limit) > 0 {
		return fmt.Sprintf("%s %s with %s %s answered under clause %s exceeds its limit of %s %s",
			q.Amount, q.Unit, FormatAmount(sp), q.Unit, c.ID, c.Limit.Amount, q.Unit)
	}
	return ""
}

// ValidAmount reports whether s is an amount of section 15.9.
func ValidAmount(s string) bool { return amountRe.MatchString(s) }

// ParseAmount reads an amount exactly.
func ParseAmount(s string) (*big.Rat, bool) {
	if !amountRe.MatchString(s) {
		return new(big.Rat), false
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return new(big.Rat), false
	}
	return r, true
}

// FormatAmount prints an amount as a decimal without trailing zeros.
func FormatAmount(r *big.Rat) string {
	s := r.FloatString(30)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}
