// Package tags reads the machinery:serves and machinery:verifies comment tags
// and a project's own trace markers (docs/formats.md sections 8 and 8.1).
package tags

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/RamXX/machinery/internal/project"
)

// Tag kinds.
const (
	Serves   = "serves"
	Verifies = "verifies"
)

// Tag is one tag line: the ids exactly as written (a token may be a prefix
// pattern ending in *).
type Tag struct {
	File string
	Line int
	Kind string
	IDs  []string
	// Marker is true for a project's own marker (section 8.1): a bare token
	// is resolved against the design.
	Marker bool
	// Scenario is true for a scenarios marker: IDs holds one scenario name.
	Scenario bool
}

var tokenRe = regexp.MustCompile(`^[A-Za-z0-9_.:*-]+$`)

// commentChars may come before a marker on its line (section 8.1).
const commentChars = " \t/#*-;%!<"

// CodeFiles lists the files that may carry serves tags: matched by
// trace.code and inside scope. Empty when the project has no trace block.
func CodeFiles(p *project.Project) ([]string, error) {
	if p.Cfg.Trace == nil {
		return nil, nil
	}
	all, err := p.Glob(p.Cfg.Trace.Code)
	if err != nil {
		return nil, err
	}
	var res []string
	for _, f := range all {
		if p.InScope(f) {
			res = append(res, f)
		}
	}
	return res, nil
}

// TestFiles lists the files that may carry verifies tags: matched by
// trace.tests, even when outside scope.
func TestFiles(p *project.Project) ([]string, error) {
	if p.Cfg.Trace == nil {
		return nil, nil
	}
	return p.Glob(p.Cfg.Trace.Tests)
}

// Scan reads the tags of the given files. A file is read for serves tags when
// trace.code matches it inside scope, and for verifies tags when trace.tests
// matches it. The result is sorted by file, then line.
func Scan(p *project.Project, files []string) ([]Tag, error) {
	if p.Cfg.Trace == nil {
		return nil, nil
	}
	mk := p.Cfg.Trace.Markers
	if mk == nil {
		mk = &project.Markers{}
	}
	var res []Tag
	seen := map[string]bool{}
	for _, f := range files {
		if seen[f] {
			continue // matched by both trace.code and trace.tests: read once
		}
		seen[f] = true
		code := project.Match(p.Cfg.Trace.Code, f) && p.InScope(f)
		test := project.Match(p.Cfg.Trace.Tests, f)
		if !code && !test {
			continue
		}
		data, err := p.Read(f)
		if err != nil {
			return nil, err
		}
		lists := map[string]bool{} // the marker lists that go on to the next line, by kind
		all := append(append(append([]string{}, mk.Serves...), mk.Verifies...), mk.Scenarios...)
		for i, line := range strings.Split(string(data), "\n") {
			_, marked := markerRest(line, all) // a line with a marker starts its own list
			if code {
				if ids := parse(line, "machinery:serves"); len(ids) > 0 {
					res = append(res, Tag{File: f, Line: i + 1, Kind: Serves, IDs: ids})
				}
			}
			if test {
				if ids := parse(line, "machinery:verifies"); len(ids) > 0 {
					res = append(res, Tag{File: f, Line: i + 1, Kind: Verifies, IDs: ids})
				}
			}
			for _, m := range []struct {
				kind    string
				on      bool
				markers []string
			}{{Serves, code, mk.Serves}, {Verifies, test, mk.Verifies}} {
				if !m.on {
					continue
				}
				rest, ok := markerRest(line, m.markers)
				if !ok && (!lists[m.kind] || marked) {
					lists[m.kind] = false
					continue
				}
				if !ok {
					rest = strings.TrimLeft(line, commentChars)
				}
				ids, more := parseList(rest)
				lists[m.kind] = more
				if len(ids) > 0 {
					res = append(res, Tag{File: f, Line: i + 1, Kind: m.kind, IDs: ids, Marker: true})
				}
			}
			if test {
				if rest, ok := markerRest(line, mk.Scenarios); ok {
					if name := strings.TrimSpace(rest); name != "" {
						res = append(res, Tag{File: f, Line: i + 1, Kind: Verifies, IDs: []string{name}, Marker: true, Scenario: true})
					}
				}
			}
		}
	}
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].File != res[j].File {
			return res[i].File < res[j].File
		}
		return res[i].Line < res[j].Line
	})
	return res, nil
}

// parse returns the id tokens after the keyword, stopping at the first token
// that is not an id.
func parse(line, keyword string) []string {
	i := strings.Index(line, keyword)
	if i < 0 {
		return nil
	}
	rest := line[i+len(keyword):]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return nil
	}
	var ids []string
	for _, tok := range strings.Fields(rest) {
		if !tokenRe.MatchString(tok) {
			break
		}
		ids = append(ids, tok)
	}
	return ids
}

// markerRest returns the text after the longest marker that follows only
// whitespace and comment characters on the line.
func markerRest(line string, markers []string) (rest string, ok bool) {
	best := ""
	for _, m := range markers {
		i := strings.Index(line, m)
		if i < 0 || strings.Trim(line[:i], commentChars) != "" || len(m) <= len(best) {
			continue
		}
		best, rest, ok = m, line[i+len(m):], true
	}
	return rest, ok
}

// parseList reads the ids of a marker list: separated by commas and spaces,
// stopping at the first token that is not an id, a period that ends the list
// dropped. more is true when every token was an id and the text ends with a
// comma: the list goes on with the next line.
func parseList(text string) (ids []string, more bool) {
	s := strings.TrimSpace(text)
	stopped := false
	for _, tok := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || unicode.IsSpace(r) }) {
		if !tokenRe.MatchString(tok) {
			stopped = true
			break
		}
		ids = append(ids, tok)
	}
	if n := len(ids); n > 0 {
		if last := strings.TrimSuffix(ids[n-1], "."); last == "" {
			ids = ids[:n-1]
		} else {
			ids[n-1] = last
		}
	}
	return ids, !stopped && len(ids) > 0 && strings.HasSuffix(s, ",")
}

// Expand returns the ids a tag token names: the id itself when it is in ids,
// or every id with the prefix when the pattern ends in *. Nil when nothing
// matches.
func Expand(pattern string, ids []string) []string {
	var res []string
	if prefix, ok := strings.CutSuffix(pattern, "*"); ok {
		for _, id := range ids {
			if strings.HasPrefix(id, prefix) {
				res = append(res, id)
			}
		}
		return res
	}
	for _, id := range ids {
		if id == pattern {
			return []string{id}
		}
	}
	return nil
}

// kinds are the id kinds of section 4.
var kinds = map[string]bool{"act": true, "inv": true, "sc": true, "tc": true, "fm": true, "fv": true, "cx": true, "ck": true}

// Resolve returns the ids a token of the tag names, or nil (section 8.1). A
// token of a machinery tag, or one with a kind prefix, is taken as written.
// A bare token of a marker is tried as inv:, act:, fm: (lower-cased) and sc:,
// in that order. scenarios maps a scenario name to its sc id.
func Resolve(t Tag, tok string, ids []string, scenarios map[string]string) []string {
	if t.Scenario {
		if id, ok := scenarios[tok]; ok {
			return Expand(id, ids)
		}
		return nil
	}
	if kind, _, ok := strings.Cut(tok, ":"); !t.Marker || (ok && kinds[kind]) {
		return Expand(tok, ids)
	}
	for _, c := range []string{"inv:" + tok, "act:" + tok, "fm:" + strings.ToLower(tok), "sc:" + tok} {
		if hits := Expand(c, ids); len(hits) > 0 {
			return hits
		}
	}
	return nil
}
