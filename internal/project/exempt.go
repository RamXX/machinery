package project

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/v2/internal/gitx"
	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

// ExcusedName is the file that may hold entries of evidence.adapter.excused
// apart from machinery.yaml (section 2.2). The tool reads it; a change to it
// leaves the runs of earlier commits valid (section 10.1).
const ExcusedName = "design/excused.yaml"

// ExcusedFile mirrors design/excused.yaml.
type ExcusedFile struct {
	Version int              `yaml:"version"`
	Excused []AdapterExcused `yaml:"excused"`
}

// ReadExcused reads the entries of design/excused.yaml of the project at
// root, nil when there is no such file.
func ReadExcused(root string) ([]AdapterExcused, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ExcusedName)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, Errf("%s: %v", ExcusedName, err)
	}
	return ParseExcused(data)
}

// ParseExcused decodes design/excused.yaml: one document, version 1, the key
// excused and nothing else, its entries held to the rules of
// evidence.adapter.excused (section 2).
func ParseExcused(data []byte) ([]AdapterExcused, error) {
	var f ExcusedFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil && !errors.Is(err, io.EOF) {
		return nil, Errf("%s: %s", ExcusedName, cleanYAMLError(err))
	}
	if f.Version != 1 {
		return nil, Errf("%s: version must be 1", ExcusedName)
	}
	var more yaml.Node
	if err := dec.Decode(&more); !errors.Is(err, io.EOF) {
		return nil, Errf("%s: the file holds more than one YAML document", ExcusedName)
	}
	key := func(n int) string { return fmt.Sprintf("%s: entry %d: tests", ExcusedName, n) }
	if err := validateExcused(f.Excused, ExcusedName, key); err != nil {
		return nil, err
	}
	return f.Excused, nil
}

// ToolOwned reports a path that is bookkeeping of the tool and not part of
// the system under test: the owed file, the adoption file, the excused
// file, a receipts file (a .jsonl file directly in the receipts directory)
// and a property record of a vital area (a .json file directly in
// <gen>/vital). A change to one leaves a run of an earlier commit valid
// (section 10.1).
func (c *Config) ToolOwned(rel string) bool {
	switch {
	case rel == c.Design.Owed, rel == AdoptionName, rel == ExcusedName:
		return true
	case strings.HasSuffix(rel, ".jsonl") && path.Dir(rel) == strings.TrimSuffix(c.Receipts.Dir, "/"):
		return true
	}
	return strings.HasSuffix(rel, ".json") && path.Dir(rel) == strings.TrimSuffix(c.Design.Gen, "/")+"/vital"
}

// Protected says what a path is when a change to it can change what a test
// checks or what a verdict is: a file under trace.tests or trace.code, a
// design source (what the contract hash covers, section 15.4), the
// configuration or a build manifest. No glob of evidence.exempt may exempt
// it. "" for any other path.
func (c *Config) Protected(rel string) string {
	switch {
	case c.Trace != nil && Match(c.Trace.Tests, rel):
		return "a file under trace.tests"
	case c.Trace != nil && Match(c.Trace.Code, rel):
		return "a file under trace.code"
	case c.DesignSource(rel):
		return "a design source"
	case rel == ConfigName:
		return "the configuration"
	case BuildManifest(rel):
		return "a build manifest"
	}
	return ""
}

// DesignSource reports a path of a design source (section 15.4): a file the
// configuration names as one, a file of the machines directory, or a file
// the globs of the imported machines match.
func (c *Config) DesignSource(rel string) bool { return Match(c.designSourceGlobs(), rel) }

// VerdictFile reports a file whose change can change a verdict that owed
// sync writes (section 11.2): machinery.yaml, the adoption file, the excused
// file, a design source (the status file among them) and a file under
// trace.tests.
func (c *Config) VerdictFile(rel string) bool {
	switch {
	case rel == ConfigName, rel == AdoptionName, rel == ExcusedName, rel == c.Design.Status:
		return true
	case c.DesignSource(rel):
		return true
	}
	return c.Trace != nil && Match(c.Trace.Tests, rel)
}

// designSourceGlobs are the paths of the design sources: the files the
// configuration names, every file of the machines directory and the globs
// of the imported machines.
func (c *Config) designSourceGlobs() []string {
	globs := append([]string(nil), c.Design.Models...)
	if c.Design.Machines != "" {
		globs = append(globs, strings.TrimSuffix(c.Design.Machines, "/")+"/**")
	}
	if im := c.ImportedMachines(); im != nil {
		globs = append(globs, im.Files...)
		if f := im.RegisterFile(); f != "" {
			globs = append(globs, f)
		}
	}
	if fm := c.ImportedFailureModes(); fm != nil {
		globs = append(globs, fm.File)
		if fm.Links != "" {
			globs = append(globs, fm.Links)
		}
	}
	for _, f := range []string{c.Design.FailureModes, c.Design.Risk, c.Design.Status, c.Design.Facts} {
		if f != "" {
			globs = append(globs, f)
		}
	}
	for _, ct := range c.Contracts {
		globs = append(globs, ct.Path)
	}
	return globs
}

// BuildManifest reports a file that says how the project is built, by its
// name: go.mod, go.sum, go.work and go.work.sum, package.json, a lock file
// (a name that ends in .lock, .lockb, -lock.json or -lock.yaml, and
// npm-shrinkwrap.json), a Makefile and a *.mk file.
func BuildManifest(rel string) bool {
	base := path.Base(rel)
	switch base {
	case "go.mod", "go.sum", "go.work", "go.work.sum", "package.json", "npm-shrinkwrap.json", "Makefile", "makefile", "GNUmakefile":
		return true
	}
	for _, suffix := range []string{".lock", ".lockb", "-lock.json", "-lock.yaml", ".mk"} {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	return false
}

// SubmoduleVoids is what a change of a submodule pointer does to the runs of
// earlier commits (section 10.1).
func SubmoduleVoids(rel string) string {
	return rel + " is a submodule pointer: a change of it voids earlier runs"
}

// checkExempt refuses a glob of evidence.exempt that matches a file of the
// project that is protected, the first such file, sorted, named; one that
// matches a submodule pointer; and one that can match a path below a
// submodule, whose files machinery never sees. The files are those git
// tracks and those it does not ignore, or every file below the root outside
// git.
func (p *Project) checkExempt() error {
	globs := p.Cfg.EvidenceExempt()
	if len(globs) == 0 {
		return nil
	}
	files, err := p.listed()
	if err != nil {
		return err
	}
	links, subs, err := p.submodules()
	if err != nil {
		return err
	}
	for _, g := range globs {
		for _, f := range files {
			switch {
			case !Match([]string{g}, f):
			case links[f]:
				return Errf("evidence.exempt: %q would exempt %s: %s; narrow the glob", g, f, SubmoduleVoids(f))
			case p.Cfg.ToolOwned(f):
			case p.Cfg.Protected(f) != "":
				return Errf("evidence.exempt: %q would exempt %s, %s; a run is never valid across a change to it: narrow the glob", g, f, p.Cfg.Protected(f))
			}
		}
		for _, s := range subs {
			if globReachesBelow(g, s) {
				return Errf("evidence.exempt: %q matches paths below %s, a submodule whose files machinery never sees: %s; narrow the glob", g, s, SubmoduleVoids(s))
			}
		}
	}
	return nil
}

// submodules returns the submodule pointers the index records and, sorted,
// the paths of every submodule: those pointers and the paths .gitmodules
// records. Nothing outside git.
func (p *Project) submodules() (links map[string]bool, subs []string, err error) {
	links = map[string]bool{}
	repo, err := gitx.IsRepo(p.Root)
	if err != nil {
		return nil, nil, err
	}
	if !repo {
		return links, nil, nil
	}
	gl, err := gitx.Gitlinks(p.Root)
	if err != nil {
		return nil, nil, err
	}
	recorded, err := gitx.Submodules(p.Root)
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]bool{}
	for _, s := range append(gl, recorded...) {
		if !seen[s] {
			seen[s] = true
			subs = append(subs, s)
		}
	}
	for _, l := range gl {
		links[l] = true
	}
	sort.Strings(subs)
	return links, subs, nil
}

// globReachesBelow reports whether a glob can match a path below a
// directory. Read segment by segment, with its braces expanded, the glob
// reaches below when each segment of the directory is matched by the segment
// of the glob in its place and the glob goes on after them, or when a "**"
// comes before their end. It may say yes of a glob that matches nothing
// there.
func globReachesBelow(g, dir string) bool {
	ds := strings.Split(dir, "/")
	for _, alt := range expandBraces(g) {
		gs := strings.Split(alt, "/")
		matched := true
		for i, d := range ds {
			if i >= len(gs) {
				matched = false
				break
			}
			if gs[i] == "**" {
				return true
			}
			if ok, _ := doublestar.Match(gs[i], d); !ok {
				matched = false
				break
			}
		}
		if matched && len(gs) > len(ds) {
			return true
		}
	}
	return false
}

// expandBraces returns the globs a pattern with braces stands for, one for
// each alternative ({a,b} gives a and b), nested braces included; a pattern
// without braces stands for itself.
func expandBraces(g string) []string {
	depth, start := 0, 0
	for i := 0; i < len(g); i++ {
		switch g[i] {
		case '\\':
			i++
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth == 0 {
				continue
			}
			if depth--; depth == 0 {
				var res []string
				for _, alt := range splitAlternatives(g[start+1 : i]) {
					res = append(res, expandBraces(g[:start]+alt+g[i+1:])...)
				}
				return res
			}
		}
	}
	return []string{g}
}

// splitAlternatives splits the inside of a brace at its commas, those of
// nested braces left alone.
func splitAlternatives(s string) []string {
	var res []string
	depth, from := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '{':
			depth++
		case '}':
			depth--
		case ',':
			if depth == 0 {
				res = append(res, s[from:i])
				from = i + 1
			}
		}
	}
	return append(res, s[from:])
}

// listed returns the files of the project git knows of and does not ignore,
// sorted, or every file below the root when it is no git work tree.
func (p *Project) listed() ([]string, error) {
	repo, err := gitx.IsRepo(p.Root)
	if err != nil {
		return nil, err
	}
	if !repo {
		return p.Files()
	}
	files, err := gitx.Listed(p.Root)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}
