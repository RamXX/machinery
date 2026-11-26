package evidence

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/RamXX/machinery/internal/project"
)

// goTestFile is one _test.go file under trace.tests, read statically
// (section 10.3). It is parsed, never compiled.
type goTestFile struct {
	Path   string            // project-relative
	Pkg    string            // import path by its go.mod; "" without one inside the project
	Funcs  []string          // the test functions, sorted
	doc    map[string][2]int // test function -> first and last line of its doc comment
	Broken bool              // not Go source that parses
}

// defines reports whether the file has the test function.
func (f *goTestFile) defines(fn string) bool {
	i := sort.SearchStrings(f.Funcs, fn)
	return i < len(f.Funcs) && f.Funcs[i] == fn
}

// owner returns the test function whose doc comment holds the line, or ""
// when the line is anywhere else in the file.
func (f *goTestFile) owner(line int) string {
	for fn, span := range f.doc {
		if line >= span[0] && line <= span[1] {
			return fn
		}
	}
	return ""
}

// readGoTests parses the Go test files among files.
func readGoTests(p *project.Project, files []string) (map[string]*goTestFile, error) {
	res := map[string]*goTestFile{}
	modules := map[string]string{}
	for _, rel := range files {
		if !strings.HasSuffix(rel, "_test.go") {
			continue
		}
		data, err := p.Read(rel)
		if err != nil {
			return nil, err
		}
		f := &goTestFile{Path: rel, Pkg: importPath(p, path.Dir(rel), modules), doc: map[string][2]int{}}
		res[rel] = f
		fset := token.NewFileSet()
		tree, err := parser.ParseFile(fset, rel, data, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			f.Broken = true
			continue
		}
		for _, decl := range tree.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !isTestFunc(fn) {
				continue
			}
			f.Funcs = append(f.Funcs, fn.Name.Name)
			if fn.Doc != nil {
				f.doc[fn.Name.Name] = [2]int{fset.Position(fn.Doc.Pos()).Line, fset.Position(fn.Doc.End()).Line}
			}
		}
		sort.Strings(f.Funcs)
	}
	return res, nil
}

// isTestFunc reports a top-level test function: no receiver, no type
// parameters, a name go test runs, one parameter of type *testing.T.
func isTestFunc(fn *ast.FuncDecl) bool {
	if fn.Recv != nil || fn.Type.TypeParams != nil || !isTestName(fn.Name.Name) {
		return false
	}
	params := fn.Type.Params.List
	if len(params) != 1 || len(params[0].Names) > 1 {
		return false
	}
	star, ok := params[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "T"
}

func isTestName(name string) bool {
	rest, ok := strings.CutPrefix(name, "Test")
	if !ok {
		return false
	}
	r, _ := utf8.DecodeRuneInString(rest)
	return rest == "" || !unicode.IsLower(r)
}

var moduleRe = regexp.MustCompile(`(?m)^module\s+"?([^\s"]+)"?`)

// importPath returns the import path of a project directory: the module of
// the nearest go.mod at or above it inside the project, plus the directory
// below that go.mod. "" when there is none.
func importPath(p *project.Project, dir string, modules map[string]string) string {
	for at := dir; ; at = path.Dir(at) {
		mod, seen := modules[at]
		if !seen {
			if data, err := os.ReadFile(p.Abs(path.Join(at, "go.mod"))); err == nil {
				if m := moduleRe.FindSubmatch(data); m != nil {
					mod = string(m[1])
				}
			}
			modules[at] = mod
		}
		if mod != "" {
			if at == dir {
				return mod
			}
			if at == "." {
				return mod + "/" + dir
			}
			return mod + "/" + strings.TrimPrefix(dir, at+"/")
		}
		if at == "." || at == "/" {
			return ""
		}
	}
}
