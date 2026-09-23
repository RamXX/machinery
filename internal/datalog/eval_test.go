package datalog

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

const tcProgram = `.decl edge(a:symbol, b:symbol)
.input edge
.decl path(a:symbol, b:symbol)
.output path
path(X, Y) :- edge(X, Y).
path(X, Z) :- path(X, Y), edge(Y, Z).
`

func mustParse(t testing.TB, src string) *Program {
	t.Helper()
	p, err := Parse(src, "t.dl")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func chain(n int) [][]string {
	var rows [][]string
	for i := 0; i < n; i++ {
		rows = append(rows, []string{fmt.Sprint("n", i), fmt.Sprint("n", i+1)})
	}
	return rows
}

func TestRunInputErrors(t *testing.T) {
	p := mustParse(t, tcProgram)
	cases := []struct {
		name string
		in   Inputs
		msg  string
	}{
		{"missing input", Inputs{}, `no input supplied for .input relation "edge"`},
		{"wrong arity", Inputs{"edge": {{"a", "b"}, {"a", "b", "c"}}}, `input "edge" row 2 has 3 columns, relation arity is 2`},
		{"tab in symbol", Inputs{"edge": {{"a\tb", "c"}}}, "may not contain a tab"},
		{"newline in symbol", Inputs{"edge": {{"a", "b\n"}}}, "may not contain a tab"},
		{"carriage return in symbol", Inputs{"edge": {{"a\r", "b"}}}, "may not contain a tab"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := p.Run(tc.in, Options{})
			if err == nil || !strings.Contains(err.Error(), tc.msg) {
				t.Fatalf("got %v, want %q", err, tc.msg)
			}
			if res != nil {
				t.Fatal("partial result returned with an error")
			}
		})
	}
}

func TestRunIgnoresUndeclaredInputs(t *testing.T) {
	p := mustParse(t, tcProgram)
	res, err := p.Run(Inputs{"edge": {{"a", "b"}}, "unrelated": {{"x", "y", "z"}}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output("path"); !reflect.DeepEqual(got, [][]string{{"a", "b"}}) {
		t.Fatalf("got %v", got)
	}
}

func TestFactFileWrongArity(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "edge.facts"), []byte("a\tb\tc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	in, err := ReadFacts(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = mustParse(t, tcProgram).Run(in, Options{})
	if err == nil || !strings.Contains(err.Error(), `input "edge" row 1 has 3 columns, relation arity is 2`) {
		t.Fatalf("got %v", err)
	}
}

func TestParseFacts(t *testing.T) {
	cases := []struct {
		name, data string
		want       [][]string
		err        string
	}{
		{"empty", "", nil, ""},
		{"one empty symbol", "\n", [][]string{{""}}, ""},
		{"no final newline", "a\tb\nc\td", [][]string{{"a", "b"}, {"c", "d"}}, ""},
		{"crlf", "a\tb\r\nc\td\r\n", [][]string{{"a", "b"}, {"c", "d"}}, ""},
		{"empty fields", "\tb\na\t\n", [][]string{{"", "b"}, {"a", ""}}, ""},
		{"spaces kept", " a \t b \n", [][]string{{" a ", " b "}}, ""},
		{"ragged", "a\tb\nc\n", nil, "x.facts:2: 1 columns, earlier lines have 2"},
		{"inner carriage return", "a\rb\n", nil, "x.facts:1: a carriage return inside a line is not supported"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseFacts(tc.data, "x.facts")
			if tc.err != "" {
				if err == nil || err.Error() != tc.err {
					t.Fatalf("got %v, want %q", err, tc.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestReadFactsErrors(t *testing.T) {
	if _, err := ReadFacts(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing directory accepted")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "r.facts"), []byte("a\tb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFacts(dir); err == nil || !strings.Contains(err.Error(), "r.facts:2") {
		t.Fatalf("got %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "r.facts")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub.facts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	in, err := ReadFacts(dir)
	if err != nil || len(in) != 0 {
		t.Fatalf("directories and non-fact files must be skipped: %v %v", in, err)
	}
	if err := os.Symlink(filepath.Join(dir, "gone"), filepath.Join(dir, "dangling.facts")); err != nil {
		t.Skip("symlinks unavailable")
	}
	if _, err := ReadFacts(dir); err == nil {
		t.Fatal("unreadable fact file accepted")
	}
}

func TestLongSymbols(t *testing.T) {
	long := strings.Repeat("x", 1<<20) + " tail"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "edge.facts"), []byte(long+"\t"+long+"2\n"+long+"2\tend\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	in, err := ReadFacts(dir)
	if err != nil {
		t.Fatal(err)
	}
	res, err := mustParse(t, tcProgram).Run(in, Options{Explain: true})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Output("path")
	if len(got) != 3 || got[0][0] != long || got[0][1] != "end" {
		t.Fatalf("long symbols mangled: %d rows", len(got))
	}
	d, err := res.Explain("path", []string{long, "end"})
	if err != nil || len(d.Children) != 2 {
		t.Fatalf("explain: %v", err)
	}
	out := t.TempDir()
	if err := res.WriteCSV(out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "path.csv"))
	if err != nil || !strings.Contains(string(b), long+"\tend\n") {
		t.Fatalf("csv: %v", err)
	}
}

func TestTupleLimitMidStratum(t *testing.T) {
	p := mustParse(t, tcProgram)
	// 20 edges derive 210 path tuples; 100 is hit inside the recursive
	// stratum, after the first round.
	res, err := p.Run(Inputs{"edge": chain(20)}, Options{MaxTuples: 100})
	if !errors.Is(err, ErrLimit) || !strings.Contains(err.Error(), `more than 100 tuples (while filling "path")`) {
		t.Fatalf("got %v", err)
	}
	if res != nil {
		t.Fatal("partial result returned")
	}
	if _, err := p.Run(Inputs{"edge": chain(20)}, Options{MaxTuples: 10}); !errors.Is(err, ErrLimit) || !strings.Contains(err.Error(), `"edge"`) {
		t.Fatalf("input loading must count against the limit: %v", err)
	}
	if res, err := p.Run(Inputs{"edge": chain(20)}, Options{MaxTuples: 230}); err != nil || len(res.Output("path")) != 210 {
		t.Fatalf("exact budget: %v", err)
	}
}

func TestIterationLimit(t *testing.T) {
	p := mustParse(t, tcProgram)
	res, err := p.Run(Inputs{"edge": chain(30)}, Options{MaxIterations: 5})
	if !errors.Is(err, ErrLimit) || !strings.Contains(err.Error(), "more than 5 iterations") || res != nil {
		t.Fatalf("got %v %v", res, err)
	}
	if _, err := p.Run(Inputs{"edge": chain(30)}, Options{MaxIterations: 31}); err != nil {
		t.Fatalf("30 edges need 31 rounds: %v", err)
	}
}

func TestExplain(t *testing.T) {
	p := mustParse(t, tcProgram)
	res, err := p.Run(Inputs{"edge": {{"b", "c"}, {"a", "b"}, {"c", "d"}}}, Options{Explain: true})
	if err != nil {
		t.Fatal(err)
	}
	d, err := res.Explain("path", []string{"a", "d"})
	if err != nil {
		t.Fatal(err)
	}
	want := `path("a", "d")  [rule 2 at 6:1]
  path("a", "c")  [rule 2 at 6:1]
    path("a", "b")  [rule 1 at 5:1]
      edge("a", "b")  [input]
    edge("b", "c")  [input]
  edge("c", "d")  [input]
`
	if got := d.String(); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	leaf, err := res.Explain("edge", []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if leaf.Rule != 0 || len(leaf.Children) != 0 || leaf.Pos != (Pos{}) || leaf.String() != "edge(\"a\", \"b\")  [input]\n" {
		t.Fatalf("an input fact must be a one-node tree: %+v", leaf)
	}
	if _, err := res.Explain("path", []string{"d", "a"}); err == nil || !strings.Contains(err.Error(), `path("d", "a") is not in the result`) {
		t.Fatalf("got %v", err)
	}
	if _, err := res.Explain("nope", []string{"a"}); err == nil || !strings.Contains(err.Error(), `relation "nope" is not declared`) {
		t.Fatalf("got %v", err)
	}
	plain, err := p.Run(Inputs{"edge": {{"a", "b"}}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plain.Explain("path", []string{"a", "b"}); err == nil || !strings.Contains(err.Error(), "Options.Explain") {
		t.Fatalf("got %v", err)
	}
}

func TestExplainNegationAndAggregate(t *testing.T) {
	src := `.decl k(a:symbol)
.input k
.decl r(a:symbol, b:symbol)
.input r
.decl blocked(a:symbol)
.input blocked
.decl c(a:symbol, n:number)
.output c
c(K, N) :- k(K), !blocked(K), N = count : { r(K, _) }.
`
	res, err := mustParse(t, src).Run(Inputs{"k": {{"a"}, {"b"}}, "r": {{"a", "1"}, {"a", "2"}}, "blocked": {{"b"}}}, Options{Explain: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output("c"); !reflect.DeepEqual(got, [][]string{{"a", "2"}}) {
		t.Fatalf("got %v", got)
	}
	d, err := res.Explain("c", []string{"a", "2"})
	if err != nil {
		t.Fatal(err)
	}
	// Only positive atoms outside aggregates are children.
	if d.Rule != 1 || len(d.Children) != 1 || d.Children[0].Relation != "k" {
		t.Fatalf("got %s", d)
	}
}

// TestDeterminism shuffles inputs and checks that outputs and explanations
// are identical: inputs are sorted before loading and rules run in order.
func TestDeterminism(t *testing.T) {
	p := mustParse(t, tcProgram)
	edges := [][]string{{"a", "b"}, {"b", "c"}, {"a", "c"}, {"c", "d"}, {"b", "d"}, {"d", "a"}}
	var firstOut [][]string
	var firstExp string
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 20; i++ {
		shuffled := append([][]string(nil), edges...)
		rng.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		shuffled = append(shuffled, shuffled[0]) // and a duplicate
		res, err := p.Run(Inputs{"edge": shuffled}, Options{Explain: true})
		if err != nil {
			t.Fatal(err)
		}
		d, err := res.Explain("path", []string{"a", "a"})
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			firstOut, firstExp = res.Output("path"), d.String()
			continue
		}
		if !reflect.DeepEqual(res.Output("path"), firstOut) || d.String() != firstExp {
			t.Fatalf("run %d differs", i)
		}
	}
}

func TestConcurrentRuns(t *testing.T) {
	p := mustParse(t, tcProgram)
	want, err := p.Run(Inputs{"edge": chain(40)}, Options{Explain: true})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := p.Run(Inputs{"edge": chain(40)}, Options{Explain: true})
			if err != nil {
				errs <- err
				return
			}
			if !reflect.DeepEqual(res.Output("path"), want.Output("path")) {
				errs <- errors.New("output differs")
				return
			}
			if _, err := res.Explain("path", []string{"n0", "n40"}); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

func TestOutputAccessors(t *testing.T) {
	src := `.decl r(a:symbol, n:number)
.decl k(a:symbol)
.input k
.decl v(a:symbol, b:symbol)
.input v
.output k
.output r
r(K, N) :- k(K), N = count : { v(K, _) }.
`
	p := mustParse(t, src)
	var rows [][]string
	for i := 0; i < 12; i++ {
		rows = append(rows, []string{"many", fmt.Sprint(i)})
	}
	rows = append(rows, []string{"two", "a"}, []string{"two", "b"})
	res, err := p.Run(Inputs{"k": {{"many"}, {"two"}, {"none"}}, "v": rows}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(res.OutputRelations(), ","); got != "r,k" {
		t.Fatalf("outputs in declaration order: %q", got)
	}
	if res.Output("missing") != nil {
		t.Fatal("undeclared relation must give nil")
	}
	// Number columns sort numerically: 0 < 2 < 12.
	if got := res.FormatCSV("r"); got != "many\t12\nnone\t0\ntwo\t2\n" {
		t.Fatalf("got %q", got)
	}
	byNumber := mustParse(t, `.decl k(a:symbol)
.input k
.decl v(a:symbol, b:symbol)
.input v
.decl c(n:number, a:symbol)
.output c
c(N, K) :- k(K), N = count : { v(K, _) }.
`)
	res2, err := byNumber.Run(Inputs{"k": {{"many"}, {"two"}, {"none"}}, "v": rows}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := res2.FormatCSV("c"); got != "0\tnone\n2\ttwo\n12\tmany\n" {
		t.Fatalf("got %q", got)
	}
	out := t.TempDir()
	if err := res.WriteCSV(out); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"r.csv", "k.csv"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Fatal(err)
		}
	}
	if err := res.WriteCSV(filepath.Join(out, "missing", "dir")); err == nil {
		t.Fatal("writing into a missing directory must fail")
	}
}

func TestAggregateResultAlreadyBound(t *testing.T) {
	src := `.decl k(a:symbol)
.input k
.decl v(a:symbol, b:symbol)
.input v
.decl c(a:symbol, n:number)
c(K, N) :- k(K), N = count : { v(K, _) }.
.decl same(a:symbol, b:symbol)
.output same
same(A, B) :- c(A, N), k(B), A != B, N = count : { v(B, _) }.
`
	res, err := mustParse(t, src).Run(Inputs{
		"k": {{"a"}, {"b"}, {"c"}},
		"v": {{"a", "1"}, {"b", "1"}, {"c", "1"}, {"c", "2"}},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.FormatCSV("same"); got != "a\tb\nb\ta\n" {
		t.Fatalf("got %q", got)
	}
}

func TestConstantOnlyComparison(t *testing.T) {
	src := `.decl k(a:symbol)
.input k
.decl yes(a:symbol)
.output yes
yes(X) :- k(X), "a" != "b".
.decl no(a:symbol)
.output no
no(X) :- k(X), "a" = "b".
`
	res, err := mustParse(t, src).Run(Inputs{"k": {{"x"}}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Output("yes")) != 1 || len(res.Output("no")) != 0 {
		t.Fatalf("yes=%v no=%v", res.Output("yes"), res.Output("no"))
	}
}

func TestErrorFormat(t *testing.T) {
	e := &Error{Name: "f.dl", Pos: Pos{3, 7}, Msg: "boom"}
	if e.Error() != "f.dl:3:7: boom" {
		t.Fatal(e.Error())
	}
}
