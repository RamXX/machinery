// Package gates implements deterministic design checks. This file owns the
// Gx closed-vocabulary declaration and enum reconciliation rule.
package gates

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/RamXX/machinery/internal/ir"
)

var (
	valuesGroup     = regexp.MustCompile(`\bVALUES(?:\s+([A-Za-z][A-Za-z0-9_-]*))?\s*\{([^}]*)\}`)
	valuesOpening   = regexp.MustCompile(`\bVALUES(?:\s+[A-Za-z][A-Za-z0-9_-]*)?\s*\{`)
	closedVocabWord = regexp.MustCompile(`(?i)(\bclosed\b.*\b(vocabulary|enum)\b|\breason\s+class\b)`)
	valueMember     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
)

func vocabularyKey(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

func modelEnumValues(dm *ir.Value) map[string][]string {
	out := map[string][]string{}
	enums := dm.AsObject().GetObject("enums")
	if enums == nil {
		return out
	}
	for _, name := range enums.Keys() {
		var values []string
		for _, value := range objSlice(enums.Get2(name).AsObject().Get2("values")) {
			if member := value.AsObject().GetString("name"); member != "" {
				values = append(values, member)
			}
		}
		sort.Strings(values)
		out[vocabularyKey(name)] = values
	}
	return out
}

func parseValues(body string) ([]string, string) {
	if strings.TrimSpace(body) == "" {
		return nil, "VALUES declaration has no members"
	}
	seen := map[string]bool{}
	var out []string
	for _, raw := range strings.Split(body, ",") {
		member := strings.TrimSpace(raw)
		if member == "" {
			return nil, "VALUES declaration contains an empty member"
		}
		if !valueMember.MatchString(member) {
			return nil, "VALUES member " + ir.Repr(member) + " is not an identifier"
		}
		if seen[member] {
			return nil, "VALUES declaration has duplicate member " + ir.Repr(member)
		}
		seen[member] = true
		out = append(out, member)
	}
	sort.Strings(out)
	return out, ""
}

func checkClosedVocabularies(g *Gate, design string, dm *ir.Value) {
	enums := modelEnumValues(dm)
	type declaration struct {
		where  string
		values []string
		named  bool
	}
	declared := map[string]declaration{}
	paths, _ := strictSortedGlob(g, filepath.Join(design, "machines"), "*.matrix.md", "vocabulary matrix")
	for _, path := range paths {
		body, err := readDesignFile(design, path)
		if err != nil {
			g.Errs = append(g.Errs, filepath.Base(path)+": unreadable vocabulary matrix: "+err.Error())
			continue
		}
		nameCol, contractCol := -1, -1
		for lineNo, line := range strings.Split(string(body), "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "|") {
				nameCol, contractCol = -1, -1
				continue
			}
			cells := splitTableRow(line)
			if ni, _, ci, header := namedUnitCols(cells); header {
				nameCol, contractCol = ni, ci
				continue
			}
			if nameCol < 0 || contractCol < 0 || isMarkdownSeparator(cells) {
				continue
			}
			name := strings.Trim(strings.TrimSpace(cellAt(cells, nameCol)), "`")
			contract := cellAt(cells, contractCol)
			where := filepath.Base(path) + ":" + strconv.Itoa(lineNo+1)
			groups := valuesGroup.FindAllStringSubmatch(contract, -1)
			if len(groups) == 0 {
				if valuesOpening.MatchString(contract) {
					g.Errs = append(g.Errs, where+": malformed VALUES declaration; write VALUES{a, b, c}")
				} else if closedVocabWord.MatchString(contract) {
					g.Errs = append(g.Errs, where+": "+ir.Repr(name)+" closed vocabulary has no VALUES{...} declaration; prose may quote the values but cannot define them")
				}
				continue
			}
			if len(groups) != 1 {
				g.Errs = append(g.Errs, where+": vocabulary row must carry exactly one VALUES{a, b, c} declaration")
				continue
			}
			values, why := parseValues(groups[0][2])
			if why != "" {
				g.Errs = append(g.Errs, where+": "+why)
				continue
			}
			vocabulary := name
			if groups[0][1] != "" {
				vocabulary = groups[0][1]
			}
			key := vocabularyKey(vocabulary)
			if key == "" {
				g.Errs = append(g.Errs, where+": VALUES declaration belongs to an empty unit name")
				continue
			}
			if prior, ok := declared[key]; ok {
				if !prior.named || groups[0][1] == "" {
					g.Errs = append(g.Errs, where+": vocabulary "+ir.Repr(vocabulary)+" already has its one declaration at "+prior.where)
					continue
				}
				if strings.Join(prior.values, "\x00") != strings.Join(values, "\x00") {
					g.Errs = append(g.Errs, where+": conflicting VALUES for "+ir.Repr(vocabulary)+": "+prior.where+" spells "+fmt.Sprint(prior.values)+", this row spells "+fmt.Sprint(values))
					continue
				}
				g.Count("closed vocabularies reconciled")
				continue
			}
			declared[key] = declaration{where: where, values: values, named: groups[0][1] != ""}
			if expected, ok := enums[key]; ok && strings.Join(values, "\x00") != strings.Join(expected, "\x00") {
				g.Errs = append(g.Errs, where+": VALUES mismatch for "+ir.Repr(vocabulary)+": matrix spells "+fmt.Sprint(values)+", Modelith enum spells "+fmt.Sprint(expected))
				continue
			}
			g.Count("closed vocabularies declared")
			if _, ok := enums[key]; ok {
				g.Count("closed vocabularies enum-matched")
			}
		}
	}
}
