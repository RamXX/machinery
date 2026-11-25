package formal

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode"

	"github.com/RamXX/machinery/internal/design"
)

func lowerCamel(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func singleWord(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if !(unicode.IsLetter(r) || (i > 0 && unicode.IsDigit(r))) {
			return false
		}
	}
	return true
}

// multiplicity maps one side of a Modelith cardinality to an Alloy keyword and
// an optional bound (lo, hi; hi < 0 means unbounded) for counts and ranges.
func multiplicity(side string) (kw string, lo, hi int, bounded bool) {
	switch side {
	case "1":
		return "one", 1, 1, false
	case "0..1":
		return "lone", 0, 1, false
	case "n", "0..n":
		return "set", 0, -1, false
	case "1..n":
		return "some", 1, -1, false
	}
	var a, b int
	if strings.Contains(side, "..") {
		parts := strings.SplitN(side, "..", 2)
		fmt.Sscanf(parts[0], "%d", &a)
		if parts[1] == "n" {
			b = -1
		} else {
			fmt.Sscanf(parts[1], "%d", &b)
		}
	} else {
		fmt.Sscanf(side, "%d", &a)
		b = a
	}
	return "set", a, b, true
}

func boundFact(expr string, lo, hi int) string {
	switch {
	case hi < 0:
		return fmt.Sprintf("#%s >= %d", expr, lo)
	case lo == hi:
		return fmt.Sprintf("#%s = %d", expr, lo)
	}
	return fmt.Sprintf("#%s >= %d and #%s =< %d", expr, lo, expr, hi)
}

// AlloyModule is the module name derived from a model path.
func AlloyModule(modelPath string) string {
	base := path.Base(modelPath)
	for _, suf := range []string{".modelith.yaml", ".modelith.yml", ".yaml", ".yml"} {
		if strings.HasSuffix(base, suf) {
			return strings.TrimSuffix(base, suf)
		}
	}
	return base
}

// AlloySkeleton derives the Alloy module from a Modelith model (section 16.2).
func AlloySkeleton(m *design.Model) []byte {
	entity := map[string]bool{}
	for _, e := range m.Entities {
		entity[e.Name] = true
	}
	enum := map[string]bool{}
	valueCount := map[string]int{}
	for _, e := range m.Enums {
		enum[e.Name] = true
		for _, v := range e.Values {
			valueCount[v]++
		}
	}
	valueName := func(en, v string) string {
		if valueCount[v] > 1 || entity[v] || enum[v] {
			return en + v
		}
		return v
	}
	var skipped []string
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	var body strings.Builder
	bw := func(format string, a ...any) { fmt.Fprintf(&body, format+"\n", a...) }

	for _, e := range m.Enums {
		bw("abstract sig %s {}", e.Name)
		if len(e.Values) > 0 {
			names := make([]string, len(e.Values))
			for i, v := range e.Values {
				names[i] = valueName(e.Name, v)
			}
			bw("one sig %s extends %s {}", strings.Join(names, ", "), e.Name)
		}
		bw("")
	}
	type field struct {
		owner, name, target, left string
		owned                     bool
	}
	var fields []field
	var facts []string
	for _, e := range m.Entities {
		if e.Derived {
			skipped = append(skipped, "derived entity "+e.Name)
			continue
		}
		used := map[string]int{}
		var lines []string
		for _, a := range e.Attributes {
			switch {
			case strings.Contains(a.Type, "."):
				skipped = append(skipped, fmt.Sprintf("%s.%s: scoped type %s", e.Name, a.Name, a.Type))
			case enum[a.Type] || entity[a.Type]:
				used[a.Name]++
				lines = append(lines, fmt.Sprintf("  %s: one %s", a.Name, a.Type))
			}
		}
		for _, r := range e.Relationships {
			if strings.Contains(r.Entity, ".") {
				skipped = append(skipped, fmt.Sprintf("%s: relationship to scoped entity %s", e.Name, r.Entity))
				continue
			}
			if !entity[r.Entity] {
				continue
			}
			sides := strings.SplitN(r.Cardinality, ":", 2)
			if len(sides) != 2 {
				continue
			}
			name := lowerCamel(r.Entity)
			if singleWord(r.Role) {
				name = lowerCamel(r.Role)
			}
			used[name]++
			if used[name] > 1 {
				name = fmt.Sprintf("%s%d", name, used[name])
			}
			kw, lo, hi, bounded := multiplicity(sides[1])
			lines = append(lines, fmt.Sprintf("  %s: %s %s", name, kw, r.Entity))
			if bounded {
				facts = append(facts, fmt.Sprintf("fact { all x: %s | %s }", e.Name, boundFact("x."+name, lo, hi)))
			}
			lkw, llo, lhi, lbounded := multiplicity(sides[0])
			switch {
			case lbounded:
				facts = append(facts, fmt.Sprintf("fact { all y: %s | %s }", r.Entity, boundFact(name+".y", llo, lhi)))
			case lkw != "set":
				facts = append(facts, fmt.Sprintf("fact { all y: %s | %s %s.y }", r.Entity, lkw, name))
			}
			if r.Ownership == "owned" && lkw != "one" && lkw != "some" {
				facts = append(facts, fmt.Sprintf("fact { all y: %s | some %s.y }", r.Entity, name))
			}
			fields = append(fields, field{e.Name, name, r.Entity, sides[0], r.Ownership == "owned"})
		}
		head := "sig " + e.Name
		if e.SubtypeOf != "" && entity[e.SubtypeOf] {
			head += " extends " + e.SubtypeOf
		}
		if len(lines) == 0 {
			bw("%s {}", head)
		} else {
			bw("%s {", head)
			bw("%s", strings.Join(lines, ",\n"))
			bw("}")
		}
		bw("")
	}
	// mirrored relationships: exactly one field each way between two entities
	count := map[string]int{}
	for _, f := range fields {
		count[f.owner+">"+f.target]++
	}
	done := map[string]bool{}
	for _, f := range fields {
		if f.owner == f.target || count[f.owner+">"+f.target] != 1 || count[f.target+">"+f.owner] != 1 {
			continue
		}
		key := f.owner + "|" + f.target
		if f.target < f.owner {
			key = f.target + "|" + f.owner
		}
		if done[key] {
			continue
		}
		done[key] = true
		var other field
		for _, g := range fields {
			if g.owner == f.target && g.target == f.owner {
				other = g
			}
		}
		facts = append(facts, fmt.Sprintf("fact { all a: %s, b: %s | b in a.%s iff a in b.%s }", f.owner, f.target, f.name, other.name))
	}
	sort.Strings(skipped)
	w("module %s", AlloyModule(m.Path))
	w("-- Generated by machinery formal gen from %s. Do not edit.", m.Path)
	w("-- Facts, predicates and assertions live in a separate file that opens this module.")
	w("-- Attributes of primitive types are not structural and are left out.")
	for _, s := range skipped {
		w("-- skipped: %s", s)
	}
	w("")
	b.WriteString(body.String())
	for _, f := range facts {
		w("%s", f)
	}
	return []byte(b.String())
}

// AlloyPropsTemplate is the property file machinery formal new writes.
func AlloyPropsTemplate(module string) []byte {
	return []byte(fmt.Sprintf(`-- Structural properties of %s. Hand-written; machinery never overwrites this file.
-- Facts, predicates and assertions only: machinery writes the commands from the runs.
open %s

-- Replace this assertion with the structural rules the model must keep. The
-- formal stage asks for a mutation that each assertion must catch.
assert ReplaceMe { all x: univ | x = x }

-- Reach predicate: the model admits a non-empty instance, so a pass is not vacuous.
pred NonEmpty { some univ }
`, module, module))
}
