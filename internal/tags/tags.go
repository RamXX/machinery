// Package tags reads the machinery:serves and machinery:verifies comment tags
// (docs/formats.md section 8).
package tags

import (
	"regexp"
	"sort"
	"strings"

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
}

var tokenRe = regexp.MustCompile(`^[A-Za-z0-9_.:*-]+$`)

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
	var res []Tag
	for _, f := range files {
		code := project.Match(p.Cfg.Trace.Code, f) && p.InScope(f)
		test := project.Match(p.Cfg.Trace.Tests, f)
		if !code && !test {
			continue
		}
		data, err := p.Read(f)
		if err != nil {
			return nil, err
		}
		for i, line := range strings.Split(string(data), "\n") {
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
