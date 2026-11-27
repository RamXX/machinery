package evidence

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/fs"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/project"
)

// goTestFile is one _test.go file under trace.tests, or a staged source an
// entry of evidence.adapter.sources names there, read statically (section
// 10.3). It is parsed, never compiled.
type goTestFile struct {
	Path   string    // project-relative
	Funcs  []string  // the test functions, sorted
	docs   []funcDoc // the doc comment of each top-level function that has one
	Broken bool      // not Go source that parses, or not readable
	// Err is why a broken file cannot be read as Go: the first error of the
	// parser, as <line>:<column>: <message>, or the error of the read
	Err string
}

// funcDoc is the doc comment of a top-level function: its first and last line.
type funcDoc struct {
	fn          string
	test        bool // a test function (isTestFunc)
	first, last int
}

// defines reports whether the file has the test function.
func (f *goTestFile) defines(fn string) bool {
	i := sort.SearchStrings(f.Funcs, fn)
	return i < len(f.Funcs) && f.Funcs[i] == fn
}

// owner returns the top-level function whose doc comment holds the line and
// whether it is a test function, or "" when the line is anywhere else in the
// file.
func (f *goTestFile) owner(line int) (fn string, test bool) {
	for _, d := range f.docs {
		if line >= d.first && line <= d.last {
			return d.fn, d.test
		}
	}
	return "", false
}

// readGoTests parses the Go test files among files, and those staged says
// are test sources whatever their name. A file it cannot read is broken, with
// the error of the read.
func readGoTests(p *project.Project, files []string, staged func(string) bool) (map[string]*goTestFile, error) {
	res := map[string]*goTestFile{}
	for _, rel := range files {
		if !strings.HasSuffix(rel, "_test.go") && !staged(rel) {
			continue
		}
		data, err := p.Read(rel)
		if err != nil {
			// the line scan of its markers stops the command on the same
			// read (section 10.3); the file is never read as passed
			res[rel] = &goTestFile{Path: rel, Broken: true, Err: readError(err)}
			continue
		}
		res[rel] = parseGoTest(rel, data)
	}
	return res, nil
}

// readError is the error of a read without the path, which the message
// that carries it names already.
func readError(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return err.Error()
}

// parseError is the first error the Go parser reports, as
// <line>:<column>: <message>.
func parseError(err error) string {
	var list scanner.ErrorList
	if errors.As(err, &list) && len(list) > 0 {
		return fmt.Sprintf("%d:%d: %s", list[0].Pos.Line, list[0].Pos.Column, list[0].Msg)
	}
	return err.Error()
}

// parseGoTest reads the test functions of one file and the comment above
// each top-level function.
func parseGoTest(name string, data []byte) *goTestFile {
	f := &goTestFile{Path: name}
	fset := token.NewFileSet()
	tree, err := parser.ParseFile(fset, name, data, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		f.Broken, f.Err = true, parseError(err)
		return f
	}
	for _, decl := range tree.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		test := isTestFunc(fn)
		if test {
			f.Funcs = append(f.Funcs, fn.Name.Name)
		}
		if fn.Doc != nil {
			f.docs = append(f.docs, funcDoc{fn: fn.Name.Name, test: test,
				first: fset.Position(fn.Doc.Pos()).Line, last: fset.Position(fn.Doc.End()).Line})
		}
	}
	sort.Strings(f.Funcs)
	return f
}

// isTestFunc reports a test function for the adapter (section 10.3): a
// top-level function without a receiver whose name begins with Test and that
// has exactly one parameter, whatever its type is written as. That is more
// than go test runs, on purpose. A TestMain whose parameter is written as a
// pointer to a type named M is the test main.
func isTestFunc(fn *ast.FuncDecl) bool {
	if fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") {
		return false
	}
	params := fn.Type.Params.List
	if len(params) != 1 || len(params[0].Names) > 1 {
		return false
	}
	if fn.Name.Name == "TestMain" {
		if star, ok := params[0].Type.(*ast.StarExpr); ok {
			switch t := star.X.(type) {
			case *ast.Ident:
				return t.Name != "M"
			case *ast.SelectorExpr:
				return t.Sel.Name != "M"
			}
		}
	}
	return true
}
