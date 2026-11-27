package evidence

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/build/constraint"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

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
	// Constrained: the file carries a build constraint, a //go:build or
	// // +build line, a name go builds only for some systems, or an import
	// of "C" (cgo off leaves it out), so a run may not have built it
	// (section 10.3)
	Constrained bool
}

// funcDoc is the doc comment of a test function, a fuzz test, a benchmark or
// an example: its first and last line. The comment of any other function is
// read as text of its file (section 10.3, "Whose claim").
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

// owner returns the function whose doc comment holds the line and whether it
// is a test function, for a test function, a fuzz test, a benchmark or an
// example; "" when the line is anywhere else in the file, the comment of
// any other function (a helper, a method, TestMain) included.
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
	f.Constrained = constrainedName(path.Base(name)) || constrainedHeader(tree) || importsC(tree)
	for _, decl := range tree.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		test := isTestFunc(fn)
		if test {
			f.Funcs = append(f.Funcs, fn.Name.Name)
		}
		if fn.Doc != nil && (test || isRunnable(fn)) {
			f.docs = append(f.docs, funcDoc{fn: fn.Name.Name, test: test,
				first: fset.Position(fn.Doc.Pos()).Line, last: fset.Position(fn.Doc.End()).Line})
		}
	}
	sort.Strings(f.Funcs)
	return f
}

// constrainedHeader reports a //go:build or // +build line before the
// package clause, where go reads them, whatever the expression says.
func constrainedHeader(tree *ast.File) bool {
	for _, group := range tree.Comments {
		if group.Pos() >= tree.Package {
			break
		}
		for _, c := range group.List {
			if constraint.IsGoBuild(c.Text) || constraint.IsPlusBuild(c.Text) {
				return true
			}
		}
	}
	return false
}

// importsC reports a file that uses cgo: go leaves it out of every build
// with cgo off, whatever its other constraints say.
func importsC(tree *ast.File) bool {
	for _, im := range tree.Imports {
		if im.Path == nil {
			continue
		}
		// "C" or `C`: go reads the path however it is quoted
		if p, err := strconv.Unquote(im.Path.Value); err == nil && p == "C" {
			return true
		}
	}
	return false
}

// constrainedName reports a file name go builds only for some systems: a
// _GOOS, _GOARCH or _GOOS_GOARCH suffix before _test.go or .go, by the lists
// of known systems of go/build, or a name go never builds (one that begins
// with _ or .). go/build keeps those lists to itself, so it is asked whether
// it would build the name for a system that is none of them.
func constrainedName(name string) bool {
	nowhere := build.Context{GOOS: "machinery", GOARCH: "machinery", Compiler: "gc",
		JoinPath: path.Join,
		OpenFile: func(string) (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("package p\n")), nil },
	}
	ok, err := nowhere.MatchFile(".", name)
	return err == nil && !ok
}

// isRunnable reports a fuzz test, a benchmark or an example: a top-level
// function without a receiver named Fuzz, Benchmark or Example, or one of
// those followed by a character that is not a lower-case letter, as go test
// names them. go test runs them as kinds of their own, and the adapter does
// not judge them: a marker above one never falls back to its file.
func isRunnable(fn *ast.FuncDecl) bool {
	if fn.Recv != nil {
		return false
	}
	for _, prefix := range []string{"Fuzz", "Benchmark", "Example"} {
		if rest, ok := strings.CutPrefix(fn.Name.Name, prefix); ok {
			r, _ := utf8.DecodeRuneInString(rest)
			return rest == "" || !unicode.IsLower(r)
		}
	}
	return false
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
