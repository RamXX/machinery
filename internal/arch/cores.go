package arch

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/project"
)

// Core is the core of one vital area as stage arch holds it (section 9): the
// globs of its files, its entry points, and the serves claims of the act ids
// its properties are over, which must go through it.
type Core struct {
	Area   string
	Paths  []string
	Entry  []string // <file>#<symbol>
	Serves []Serve  // serves claims of the act ids its properties are over
	Acts   bool     // a property of the area is over an act id
	Traced bool     // trace is configured, so serves tags were read
}

// Serve is one serves claim: a file that serves an act id.
type Serve struct {
	File string
	ID   string
}

// CoreCount is what the stage reports per core.
type CoreCount struct {
	Area    string
	Files   int
	Lines   int
	Entries int
}

// checkCores holds every core to the limits of constraints.cores.
func (k *checker) checkCores(cores []Core, files []string) {
	lim := k.p.Cfg.Constraints.Cores
	if lim == nil {
		return
	}
	for _, c := range cores {
		k.c.Cores = append(k.c.Cores, k.checkCore(c, files, lim))
	}
}

func (k *checker) undemonstrated(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	for _, have := range k.c.Undemonstrated {
		if have == msg {
			return
		}
	}
	k.c.Undemonstrated = append(k.c.Undemonstrated, msg)
}

func (k *checker) add(rule, subject, format string, a ...any) {
	for _, v := range k.vs {
		if v.Rule == rule && v.Subject == subject {
			return
		}
	}
	k.vs = append(k.vs, Violation{Rule: rule, Subject: subject, Message: fmt.Sprintf(format, a...)})
}

func (k *checker) checkCore(c Core, files []string, lim *project.Cores) CoreCount {
	cc := CoreCount{Area: c.Area, Entries: len(c.Entry)}
	for _, g := range c.Paths {
		hit := false
		for _, f := range files {
			if project.Match([]string{g}, f) {
				hit = true
				break
			}
		}
		if !hit {
			k.add("core-missing", c.Area+": "+g, "core-missing: %s: %s matches no file", c.Area, g)
		}
	}
	inCore := map[string]bool{}
	coreDirs := map[string]bool{}
	for _, f := range files {
		if !project.Match(c.Paths, f) {
			continue
		}
		inCore[f] = true
		coreDirs[path.Dir(f)] = true
		if !k.p.InScope(f) {
			k.add("core-scope", f, "core-scope: %s: %s is outside scope; widen the scope or move the core", c.Area, f)
			continue
		}
		data, err := k.p.Read(f)
		if err != nil {
			k.undemonstrated("core %s: %s: cannot read: %v", c.Area, f, err)
			continue
		}
		src := string(data)
		n := lineCount(src)
		cc.Files++
		cc.Lines += n
		if n > lim.MaxFileLines {
			k.add("core-file-size", f, "core-file-size: %s has %d lines (max %d for a core)", f, n, lim.MaxFileLines)
		}
		if langOf(f) != "go" {
			if len(lim.DenyImports) > 0 {
				k.undemonstrated("core %s: %s: deny_imports is checked in Go files only", c.Area, f)
			}
			k.undemonstrated("core %s: %s: max_func_lines is checked in Go files only", c.Area, f)
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, f, src, parser.SkipObjectResolution)
		if file == nil || err != nil {
			k.add("parse", f, "parse: %s: %v", f, err)
			continue
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				lines := fset.Position(fn.End()).Line - fset.Position(fn.Pos()).Line + 1
				if lines > lim.MaxFuncLines {
					subject := f + ":" + funcName(fn)
					k.add("core-func-size", subject, "core-func-size: %s has %d lines (max %d for a core)", subject, lines, lim.MaxFuncLines)
				}
			}
		}
		for _, imp := range file.Imports {
			ip, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			for _, deny := range lim.DenyImports {
				if ip == deny || strings.HasPrefix(ip, deny+"/") {
					k.add("core-impure", f+" -> "+ip, "core-impure: %s imports %s, which constraints.cores.deny_imports denies", f, ip)
					break
				}
			}
		}
	}
	for _, e := range c.Entry {
		k.checkEntry(c, e, inCore)
	}
	k.checkBypass(c, inCore, coreDirs)
	return cc
}

// checkEntry finds an entry point's symbol in its core file.
func (k *checker) checkEntry(c Core, e string, inCore map[string]bool) {
	i := strings.LastIndex(e, "#")
	if i < 0 {
		return // stage risk reports the shape
	}
	f, sym := e[:i], e[i+1:]
	switch {
	case !k.all[f]:
		k.add("core-entry", e, "core-entry: %s: %s: no such file", c.Area, e)
		return
	case !inCore[f]:
		k.add("core-entry", e, "core-entry: %s: %s: %s is not matched by core.paths", c.Area, e, f)
		return
	case !k.p.InScope(f):
		return // core-scope says it
	case langOf(f) != "go":
		k.undemonstrated("core %s: entry %s: symbols are found in Go files only", c.Area, e)
		return
	}
	data, err := k.p.Read(f)
	if err != nil {
		return
	}
	file, err := parser.ParseFile(token.NewFileSet(), f, data, parser.SkipObjectResolution)
	if file == nil || err != nil {
		return // the parse finding says it
	}
	if !hasSymbol(file, sym) {
		k.add("core-entry", e, "core-entry: %s: %s: no function, method or type %s in %s", c.Area, e, sym, f)
	}
}

// hasSymbol reports whether a Go file declares a function, a method
// (Type.Method) or a type of the name.
func hasSymbol(file *ast.File, sym string) bool {
	for _, decl := range file.Decls {
		switch x := decl.(type) {
		case *ast.FuncDecl:
			if funcName(x) == sym {
				return true
			}
		case *ast.GenDecl:
			for _, spec := range x.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == sym {
					return true
				}
			}
		}
	}
	return false
}

// checkBypass reports a file outside the core that serves an act id a
// property is over and imports no core file.
func (k *checker) checkBypass(c Core, inCore, coreDirs map[string]bool) {
	if c.Acts && !c.Traced {
		k.undemonstrated("core %s: core-bypass reads serves tags and trace is not configured", c.Area)
		return
	}
	serves := append([]Serve(nil), c.Serves...)
	sort.Slice(serves, func(i, j int) bool {
		if serves[i].File != serves[j].File {
			return serves[i].File < serves[j].File
		}
		return serves[i].ID < serves[j].ID
	})
	for _, s := range serves {
		if inCore[s.File] {
			continue
		}
		imports, ok := k.projectImports(s.File)
		if !ok {
			k.undemonstrated("core %s: %s: imports are not read for this language", c.Area, s.File)
			continue
		}
		through := false
		for _, t := range imports {
			if (t.dir && coreDirs[t.path]) || (!t.dir && inCore[t.path]) {
				through = true
				break
			}
		}
		if !through {
			k.add("core-bypass", s.File+" -> "+s.ID, "core-bypass: %s serves %s outside the core of %s and does not import it: the path that must not fail has a second implementation", s.File, s.ID, c.Area)
		}
	}
}

// projectImports lists what a file imports inside the project; ok is false
// when its imports cannot be read.
func (k *checker) projectImports(f string) (ts []target, ok bool) {
	lang := langOf(f)
	if lang == "" {
		return nil, false
	}
	data, err := k.p.Read(f)
	if err != nil {
		return nil, false
	}
	if lang != "go" {
		return importsOf(lang, f, string(data), k.all), true
	}
	file, err := parser.ParseFile(token.NewFileSet(), f, data, parser.ImportsOnly)
	if file == nil || err != nil {
		return nil, false
	}
	mod := k.modOf(path.Dir(f))
	if mod == nil {
		return nil, len(file.Imports) == 0
	}
	for _, imp := range file.Imports {
		ip, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		rel, found := "", ip == mod.path
		if r, under := strings.CutPrefix(ip, mod.path+"/"); under {
			rel, found = r, true
		}
		if !found {
			continue
		}
		dir := path.Join(mod.dir, rel)
		if dir == "" {
			dir = "."
		}
		ts = append(ts, target{path: dir, dir: true})
	}
	return ts, true
}
