package arch

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strconv"
	"strings"
)

// goMod is the nearest go.mod of a directory: where it lives and its module path.
type goMod struct {
	dir  string // project-relative, "" for the root
	path string
}

func (k *checker) modOf(dir string) *goMod {
	if m, ok := k.mods[dir]; ok {
		return m
	}
	var m *goMod
	rel := path.Join(dir, "go.mod")
	if dir == "." {
		rel = "go.mod"
	}
	if k.all[rel] {
		if data, err := k.p.Read(rel); err == nil {
			if mp := modulePath(string(data)); mp != "" {
				d := dir
				if d == "." {
					d = ""
				}
				m = &goMod{dir: d, path: mp}
			}
		}
	} else if dir != "." && dir != "" {
		m = k.modOf(path.Dir(dir))
	}
	k.mods[dir] = m
	return m
}

func modulePath(gomod string) string {
	for _, line := range strings.Split(gomod, "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "module"); ok && (rest == "" || rest[0] == ' ' || rest[0] == '\t') {
			rest, _, _ = strings.Cut(rest, "//")
			rest = strings.TrimSpace(rest)
			if u, err := strconv.Unquote(rest); err == nil {
				return u
			}
			return rest
		}
	}
	return ""
}

// checkGo parses a Go file: imports for the layer rule, functions for the
// size rule.
func (k *checker) checkGo(f, src string, sized bool, layer int) {
	mode := parser.ImportsOnly
	if sized {
		mode = parser.SkipObjectResolution
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, f, src, mode)
	if err != nil {
		k.vs = append(k.vs, Violation{Rule: "parse", Subject: f, Message: fmt.Sprintf("parse: %s: %v", f, err)})
		if file == nil {
			return
		}
	}
	if layer >= 0 {
		k.checkGoImports(f, layer, file)
	}
	if sized {
		k.checkFuncs(f, fset, file)
	}
}

func (k *checker) checkGoImports(f string, layer int, file *ast.File) {
	mod := k.modOf(path.Dir(f))
	if mod == nil {
		if len(file.Imports) > 0 {
			k.vs = append(k.vs, Violation{Rule: "go-mod", Subject: f,
				Message: fmt.Sprintf("go-mod: %s: no go.mod found, so its imports cannot be mapped to project paths", f)})
		}
		return
	}
	var ts []target
	for _, imp := range file.Imports {
		ip, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		rel, ok := "", false
		if ip == mod.path {
			rel, ok = "", true
		} else if r, found := strings.CutPrefix(ip, mod.path+"/"); found {
			rel, ok = r, true
		}
		if !ok {
			continue
		}
		dir := path.Join(mod.dir, rel)
		if dir == "" {
			dir = "."
		}
		ts = append(ts, target{path: dir, dir: true})
	}
	k.checkLayers(f, layer, ts)
}

func (k *checker) checkFuncs(f string, fset *token.FileSet, file *ast.File) {
	max := k.p.Cfg.Constraints.Size.MaxFuncLines
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		k.c.Funcs++
		n := fset.Position(fn.End()).Line - fset.Position(fn.Pos()).Line + 1
		if max > 0 && n > max {
			subject := f + ":" + funcName(fn)
			k.vs = append(k.vs, Violation{Rule: "func-size", Subject: subject,
				Message: fmt.Sprintf("func-size: %s has %d lines (max %d)", subject, n, max)})
		}
	}
}

func funcName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	t := fn.Recv.List[0].Type
	for {
		switch x := t.(type) {
		case *ast.StarExpr:
			t = x.X
			continue
		case *ast.IndexExpr:
			t = x.X
			continue
		case *ast.IndexListExpr:
			t = x.X
			continue
		case *ast.ParenExpr:
			t = x.X
			continue
		case *ast.Ident:
			return x.Name + "." + fn.Name.Name
		}
		return fn.Name.Name
	}
}
