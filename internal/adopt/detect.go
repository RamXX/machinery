package adopt

import (
	"path"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/testpolicy"
)

// Sources are the design files found in the project.
type Sources struct {
	Models    []string
	Machines  []string
	Contracts []string
}

// discover finds the design sources among the project files, sorted. Files
// under the gen directory are generated, never sources.
func discover(files []string, genDir string) Sources {
	var s Sources
	gen := strings.TrimSuffix(genDir, "/") + "/"
	for _, f := range files {
		if strings.HasPrefix(f, gen) {
			continue
		}
		base := path.Base(f)
		switch {
		case strings.HasSuffix(base, ".modelith.yaml"):
			s.Models = append(s.Models, f)
		case strings.HasSuffix(base, ".machine.json"):
			s.Machines = append(s.Machines, f)
		case base == "openapi.yaml" || base == "openapi.yml" || base == "openapi.json":
			s.Contracts = append(s.Contracts, f)
		}
	}
	return s
}

// machinesDir is the directory design.machines names: the one holding the most
// machine files, the first in path order on a tie.
func (s Sources) machinesDir() string {
	n := map[string]int{}
	best := ""
	for _, f := range s.Machines {
		d := path.Dir(f)
		n[d]++
		if best == "" || n[d] > n[best] {
			best = d
		}
	}
	return best
}

// langByExt names the language of a source file by extension, or "".
func langByExt(f string) string {
	switch strings.ToLower(path.Ext(f)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".ts", ".tsx", ".mts", ".cts":
		return "typescript"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	case ".ex", ".exs":
		return "elixir"
	case ".rs":
		return "rust"
	}
	return ""
}

var langByMarker = map[string]string{
	"go.mod": "go", "package.json": "javascript", "pyproject.toml": "python", "mix.exs": "elixir", "Cargo.toml": "rust",
}

// languages lists the languages found in scope: by extension, and by marker
// files inside the scope or in the parents of its paths.
func languages(p *project.Project, files []string) []string {
	set := map[string]bool{}
	exists := map[string]bool{}
	for _, f := range files {
		exists[f] = true
	}
	for _, f := range files {
		if !p.InScope(f) {
			continue
		}
		if l := langByExt(f); l != "" {
			set[l] = true
		}
		if l, ok := langByMarker[path.Base(f)]; ok {
			set[l] = true
		}
	}
	for _, g := range p.Cfg.Scope {
		dir := strings.TrimSuffix(project.GlobPrefix(g), "/")
		for {
			for marker, l := range langByMarker {
				m := path.Join(dir, marker)
				if dir == "" {
					m = marker
				}
				if exists[m] {
					set[l] = true
				}
			}
			if dir == "" || dir == "." {
				break
			}
			dir = path.Dir(dir)
			if dir == "." {
				dir = ""
			}
		}
	}
	if set["typescript"] {
		delete(set, "javascript") // package.json of a typescript project
	}
	var res []string
	for l := range set {
		res = append(res, l)
	}
	sort.Strings(res)
	return res
}

// counts is what the report says about code and tests in scope.
type counts struct {
	Code, E2E, Unit, Doubles int
}

// count reads the code and test files the policy would look at.
func count(p *project.Project, files []string) (counts, error) {
	patterns, markers := testpolicy.Policy(p)
	roots := p.TestRoots()
	var c counts
	for _, f := range files {
		underRoot := project.Match(roots, f)
		if project.Match(patterns, f) && (underRoot || p.InScope(f)) {
			if underRoot {
				c.E2E++
			} else {
				c.Unit++
			}
			data, err := p.Read(f)
			if err != nil {
				return c, err
			}
			for _, m := range markers {
				if strings.Contains(string(data), m) {
					c.Doubles++
					break
				}
			}
			continue
		}
		if p.InScope(f) && langByExt(f) != "" {
			c.Code++
		}
	}
	return c, nil
}
