// Package arch enforces the constraints block: layers and size limits
// (docs/formats.md section 9).
package arch

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/baseline"
	"github.com/RamXX/machinery/internal/project"
)

// Violation is one finding of the arch stage.
type Violation = baseline.Violation

// Counts is what the stage looked at.
type Counts struct {
	Files   int // files read: source files and size-limited files inside scope
	Imports int // imports of layered files that resolve to a project path
	Funcs   int // Go functions in size-limited files
}

// Applies reports whether the project has a constraints block.
func Applies(p *project.Project) bool { return p.Cfg.Constraints != nil }

// BaselinePath is the configured baseline file, or "".
func BaselinePath(p *project.Project) string {
	if p.Cfg.Constraints == nil {
		return ""
	}
	return p.Cfg.Constraints.Baseline
}

// HasRules reports whether the constraints block states any rule.
func HasRules(p *project.Project) bool {
	c := p.Cfg.Constraints
	return c != nil && (len(c.Layers) > 0 || c.Size != nil)
}

// WriteBaseline records the violations in the configured baseline file.
func WriteBaseline(p *project.Project, vs []Violation) error {
	rel := BaselinePath(p)
	if rel == "" {
		return project.Errf("constraints.baseline is not set: --write-baseline needs a baseline file in machinery.yaml")
	}
	return baseline.Write(p, rel, vs)
}

// checker holds what one run needs.
type checker struct {
	p      *project.Project
	all    map[string]bool     // every project file
	byDir  map[string][]string // files by directory
	layers []project.Layer
	mods   map[string]*goMod // go.mod lookup by directory
	vs     []Violation
	c      Counts
}

// Violations reads the files inside scope and returns the violations sorted
// by rule, then subject.
func Violations(p *project.Project) ([]Violation, Counts, error) {
	cons := p.Cfg.Constraints
	if cons == nil {
		return nil, Counts{}, nil
	}
	files, err := p.Files()
	if err != nil {
		return nil, Counts{}, err
	}
	k := &checker{p: p, all: map[string]bool{}, byDir: map[string][]string{}, layers: cons.Layers, mods: map[string]*goMod{}}
	for _, f := range files {
		k.all[f] = true
		k.byDir[path.Dir(f)] = append(k.byDir[path.Dir(f)], f)
	}
	var sizeGlobs []string
	if cons.Size != nil {
		sizeGlobs = cons.Size.Paths
		if len(sizeGlobs) == 0 {
			sizeGlobs = p.Cfg.Scope
		}
	}
	for _, f := range files {
		if !p.InScope(f) {
			continue
		}
		lang := langOf(f)
		sized := cons.Size != nil && project.Match(sizeGlobs, f)
		layer := k.layerOfFile(f)
		if lang == "" && !sized {
			continue
		}
		k.c.Files++
		data, err := p.Read(f)
		if err != nil {
			return nil, k.c, err
		}
		src := string(data)
		if sized {
			k.checkFileSize(f, src)
		}
		if lang == "go" && (sized || layer >= 0) {
			k.checkGo(f, src, sized, layer)
		} else if lang != "" && layer >= 0 {
			k.checkLayers(f, layer, importsOf(lang, f, src, k.all))
		}
	}
	sort.Slice(k.vs, func(i, j int) bool {
		if k.vs[i].Rule != k.vs[j].Rule {
			return k.vs[i].Rule < k.vs[j].Rule
		}
		return k.vs[i].Subject < k.vs[j].Subject
	})
	return k.vs, k.c, nil
}

func (k *checker) checkFileSize(f, src string) {
	max := k.p.Cfg.Constraints.Size.MaxFileLines
	if n := lineCount(src); max > 0 && n > max {
		k.vs = append(k.vs, Violation{Rule: "file-size", Subject: f,
			Message: fmt.Sprintf("file-size: %s has %d lines (max %d)", f, n, max)})
	}
}

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

// layerOfFile is the index of the first layer whose globs match, or -1.
func (k *checker) layerOfFile(rel string) int {
	for i, l := range k.layers {
		if project.Match(l.Paths, rel) {
			return i
		}
	}
	return -1
}

// layerOfDir is the lowest layer that holds a file of the directory; a
// directory with no files is matched by name.
func (k *checker) layerOfDir(dir string) int {
	best := -1
	for _, f := range k.byDir[dir] {
		if i := k.layerOfFile(f); i >= 0 && (best < 0 || i < best) {
			best = i
		}
	}
	if best >= 0 {
		return best
	}
	for i, l := range k.layers {
		if project.Match(l.Paths, dir) || project.Match(l.Paths, dir+"/_") {
			return i
		}
	}
	return -1
}

// target is an import resolved to a project path; dir is true for a Go package.
type target struct {
	path string
	dir  bool
}

func (k *checker) checkLayers(f string, layer int, ts []target) {
	seen := map[string]bool{}
	for _, t := range ts {
		if t.path == f || seen[t.path] {
			continue
		}
		seen[t.path] = true
		k.c.Imports++
		var to int
		if t.dir {
			to = k.layerOfDir(t.path)
		} else {
			to = k.layerOfFile(t.path)
		}
		if to > layer {
			k.vs = append(k.vs, Violation{Rule: "layer", Subject: f + " -> " + t.path,
				Message: fmt.Sprintf("layer: %s (%s) imports %s (%s)", f, k.layers[layer].Name, t.path, k.layers[to].Name)})
		}
	}
}
