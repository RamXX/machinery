package datalog

import (
	"errors"
	"strings"
	"testing"
)

const decls = ".decl r(a:symbol, b:symbol)\n.input r\n.decl s(a:symbol)\n.input s\n.decl n(a:symbol, c:number)\n"

// TestParseRejects covers every construct outside the subset and every
// compile-time check. Each case names the position and a message fragment.
func TestParseRejects(t *testing.T) {
	cases := []struct {
		name, src, pos, msg string
	}{
		{"type", ".type T <: symbol\n", "1:1", "directive .type is not supported"},
		{"plan", decls + "s(X) :- r(X, _).\n.plan 0:(1)\n", "7:1", "directive .plan is not supported"},
		{"printsize", decls + ".printsize s\n", "6:1", "directive .printsize is not supported"},
		{"functor directive", ".functor f(a:symbol):symbol\n", "1:1", "directive .functor is not supported"},
		{"component", ".comp C {}\n", "1:1", "directive .comp is not supported"},
		{"init", ".init c = C\n", "1:1", "directive .init is not supported"},
		{"pragma", ".pragma \"x\" \"y\"\n", "1:1", "directive .pragma is not supported"},
		{"unknown directive", ".frobnicate\n", "1:1", "unknown directive .frobnicate"},
		{"qualifier brie", ".decl q(a:symbol) brie\n", "1:19", `relation qualifier "brie" is not supported`},
		{"choice-domain", ".decl q(a:symbol) choice-domain a\n", "1:19", `relation qualifier "choice" is not supported`},
		{"inline", ".decl q(a:symbol) inline\n", "1:19", `relation qualifier "inline" is not supported`},
		{"magic", ".decl q(a:symbol) magic\n", "1:19", `relation qualifier "magic" is not supported`},
		{"float type", ".decl q(a:float)\n", "1:11", `type "float" is not supported`},
		{"user type", ".decl q(a:Name)\n", "1:11", `type "Name" is not supported`},
		{"nullary decl", ".decl q()\n", "1:9", "nullary relations are not supported"},
		{"duplicate attr", ".decl q(a:symbol, a:symbol)\n", "1:19", `attribute "a" declared twice`},
		{"record", decls + "s(X) :- r(X, [Y, Z]).\n", "6:14", "records are not supported"},
		{"adt", decls + "s(X) :- r(X, $A(Y)).\n", "6:14", "algebraic data types are not supported"},
		{"arithmetic", decls + "s(X) :- r(X, Y), X = Y + Y.\n", "6:24", "arithmetic is not supported"},
		{"number literal", decls + "s(X) :- n(X, 3).\n", "6:14", "number literals are not supported"},
		{"functor call", decls + "s(X) :- r(X, Y), X = cat(Y, Y).\n", "6:22", "functors are not supported"},
		{"user functor", decls + "s(X) :- r(X, @f(X)).\n", "6:14", "user-defined functors are not supported"},
		{"less than", decls + "s(X) :- r(X, Y), X < Y.\n", "6:20", "only = and != comparisons are supported"},
		{"subsumption", decls + "s(X) <= s(Y) :- r(X, Y).\n", "6:6", "only = and != comparisons are supported"},
		{"disjunction", decls + "s(X) :- r(X, _) ; r(_, X).\n", "6:17", "disjunction"},
		{"multiple heads", decls + "s(X), s(Y) :- r(X, Y).\n", "6:5", "multiple heads"},
		{"program fact", decls + "s(\"a\").\n", "6:1", "facts in the program are not supported"},
		{"max", decls + "n(X, M) :- s(X), M = max Y : { n(X, Y) }.\n", "6:22", `aggregate "max" is not supported`},
		{"sum", decls + "n(X, M) :- s(X), M = sum Y : { n(X, Y) }.\n", "6:22", `aggregate "sum" is not supported`},
		{"mean", decls + "n(X, M) :- s(X), M = mean Y : { n(X, Y) }.\n", "6:22", `aggregate "mean" is not supported`},
		{"count without braces", decls + "n(X, C) :- s(X), C = count : r(X, _).\n", "6:30", "the braced form is required"},
		{"nested aggregate", decls + "n(X, C) :- s(X), C = count : { s(X), D = count : { r(X, _) } }.\n", "6:42", "nested aggregates"},
		{"aggregate with !=", decls + "n(X, C) :- s(X), C != count : { r(X, _) }.\n", "6:20", "must be bound with '='"},
		{"aggregate into constant", decls + "n(X, C) :- s(X), \"c\" = count : { r(X, _) }.\n", "6:18", "must be bound to a variable"},
		{"min target not variable", decls + "n(X, C) :- s(X), C = min \"a\" : { n(X, _) }.\n", "6:26", "min target must be a variable"},
		{"preprocessor", "#include \"x.dl\"\n", "1:1", "C preprocessor"},
		{"io parameters", ".decl q(a:symbol)\n.input q(IO=file, filename=\"x\")\n", "2:9", ".input parameters are not supported"},
		{"two outputs", ".decl q(a:symbol)\n.decl p(a:symbol)\n.output q, p\n", "3:10", "one relation per .output"},
		{"nullary atom", decls + "s(X) :- r(X, _), s().\n", "6:20", "nullary atoms"},
		{"escape", decls + "s(X) :- r(X, \"a\\tb\").\n", "6:16", "escape sequences"},
		{"tab in string", decls + "s(X) :- r(X, \"a\tb\").\n", "6:16", "tab, newline or carriage return"},
		{"newline in string", decls + "s(X) :- r(X, \"a\nb\").\n", "6:16", "tab, newline or carriage return"},
		{"unterminated string", decls + "s(X) :- r(X, \"ab", "6:14", "unterminated string"},
		{"unterminated comment", "/* open\n.decl q(a:symbol)\n", "1:1", "unterminated block comment"},
		{"reserved relation", ".decl count(a:symbol)\n", "1:7", `"count" is a reserved Soufflé word`},
		{"reserved variable", decls + "s(min) :- s(min).\n", "6:3", `"min" is a reserved Soufflé word`},
		{"wildcard relation", ".decl _(a:symbol)\n", "1:7", "wildcard _ cannot be used"},
		{"question mark", ".decl q?(a:symbol)\n", "1:8", "'?'"},
		{"single quote", decls + "s(X) :- r(X, 'a').\n", "6:14", "single-quoted"},
		{"control byte", "\x01", "1:1", "unexpected byte 0x01"},
		{"stray token", ") oops\n", "1:1", "expected a directive or a rule"},
		{"missing if", decls + "s(X) r(X, _).\n", "6:6", "expected ':-'"},
		{"bad body separator", decls + "s(X) :- r(X, _) s(X).\n", "6:17", "expected ',' or '.'"},
		{"bad arg separator", decls + "s(X) :- r(X _).\n", "6:13", "expected ',' or ')'"},
		{"bad attr separator", ".decl q(a:symbol b:symbol)\n", "1:18", "expected ',' or ')'"},
		{"missing colon", ".decl q(a symbol)\n", "1:11", "expected ':'"},
		{"bad term", decls + "s(X) :- r(X, :-).\n", "6:14", "expected a variable"},
		{"bad comparison op", decls + "s(X) :- X.\n", "6:10", "expected '=' or '!='"},
		{"eof in rule", decls + "s(X) :- r(X, _)", "6:16", "expected ',' or '.'"},

		// Compile-time checks.
		{"declared twice", ".decl q(a:symbol)\n.decl q(b:symbol)\n", "2:7", `relation "q" is declared twice (first at 1:7)`},
		{"undeclared in rule", decls + "s(X) :- t(X).\n", "6:9", `relation "t" is not declared`},
		{"undeclared head", decls + "t(X) :- s(X).\n", "6:1", `relation "t" is not declared`},
		{"undeclared input", ".input q\n", "1:1", `.input names undeclared relation "q"`},
		{"undeclared output", ".output q\n", "1:1", `.output names undeclared relation "q"`},
		{"output twice", ".decl q(a:symbol)\n.output q\n.output q\n", "3:1", `marked .output twice`},
		{"input twice", ".decl q(a:symbol)\n.input q\n.input q\n", "3:1", `marked .input twice`},
		{"number input", ".decl q(a:number)\n.input q\n", "2:1", `input relation "q" has number attribute "a"`},
		{"arity", decls + "s(X) :- r(X).\n", "6:9", `relation "r" has arity 2, used here with 1 arguments`},
		{"head wildcard", decls + "s(_) :- s(_).\n", "6:3", "wildcard _ cannot appear in a rule head"},
		{"unbound head", decls + "s(Y) :- s(X).\n", "6:3", "variable Y in the head is not bound"},
		{"unbound negation", decls + "s(X) :- s(X), !r(X, Y).\n", "6:21", "variable Y in a negated atom is not bound"},
		{"unbound comparison", decls + "s(X) :- s(X), X != Y.\n", "6:20", "variable Y in a comparison is not bound"},
		{"unbound equality", decls + "s(X) :- s(X), Y = Z.\n", "6:15", "variable Y in a comparison is not bound"},
		{"wildcard comparison", decls + "s(X) :- s(X), X != _.\n", "6:15", "wildcard _ cannot be compared"},
		{"head constant in number", decls + "n(X, \"1\") :- s(X).\n", "6:6", `string constant in number attribute "c"`},
		{"body constant in number", decls + ".decl m(a:symbol)\nm(X) :- n(X, \"1\").\n", "7:14", `string constant in number attribute "c"`},
		{"negated constant in number", decls + "s(X) :- s(X), !n(X, \"1\").\n", "6:21", "string constant where a number is expected"},
		{"number into symbol", decls + ".decl m(a:symbol)\nm(C) :- n(_, C).\n", "7:3", `variable C is number but attribute "a" of "m" is symbol`},
		{"symbol into number", decls + "n(X, X) :- s(X).\n", "6:6", `variable X is symbol but attribute "c" of "n" is number`},
		{"mixed join", decls + "s(X) :- n(X, C), r(X, C).\n", "6:23", `variable C is number but attribute "b" of "r" is symbol`},
		{"mixed negation", decls + "s(X) :- n(X, C), !s(C).\n", "6:21", "variable C is number where symbol is expected"},
		{"mixed comparison", decls + "s(X) :- n(X, C), X != C.\n", "6:18", "comparison between symbol and number"},
		{"group unbound", decls + "n(X, C) :- C = count : { r(X, _) }.\n", "6:12", "variable X is shared with the aggregate but not bound by a positive atom outside it"},
		{"result inside", decls + "n(X, C) :- s(X), C = count : { n(X, C) }.\n", "6:18", "aggregate result C also occurs inside the aggregate"},
		{"min over symbol", decls + ".decl m(a:symbol, b:symbol)\nm(X, M) :- s(X), M = min Y : { r(X, Y) }.\n", "7:26", "min over symbol variable Y is not supported"},
		{"min target outside", decls + "n(X, M) :- n(X, Y), M = min Y : { n(X, Y) }.\n", "6:29", "the min target Y must occur only inside the aggregate"},
		{"min target unbound", decls + "n(X, M) :- s(X), M = min Y : { n(X, _) }.\n", "6:26", "the min target Y is not bound inside the aggregate"},
		{"aggregate without atom", decls + "n(X, C) :- s(X), C = count : { !s(X) }.\n", "6:18", "at least one positive atom"},
		{"aggregate inner unbound", decls + "n(X, C) :- s(X), C = count : { r(X, _), !r(Z, X) }.\n", "6:44", "variable Z in a negated atom is not bound"},
		{"aggregate result symbol", decls + "s(X) :- r(X, C), C = count : { s(X) }.\n", "6:18", "aggregate result C is already bound to a symbol"},
		{"self negation", decls + ".decl p(a:symbol)\np(X) :- s(X), !p(X).\n", "7:15", "cycle through negation p -> p"},
		{"aggregate on self", decls + "n(X, C) :- s(X), C = count : { n(X, _) }.\n", "6:32", "cycle through aggregation n -> n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.src, "t.dl")
			if err == nil {
				t.Fatalf("accepted:\n%s", tc.src)
			}
			var pe *Error
			if !errors.As(err, &pe) {
				t.Fatalf("error %v is not a *Error", err)
			}
			if pe.Pos.String() != tc.pos || !strings.Contains(pe.Msg, tc.msg) {
				t.Fatalf("got %v\nwant position %s and message containing %q", err, tc.pos, tc.msg)
			}
			if !strings.HasPrefix(err.Error(), "t.dl:"+tc.pos+": ") {
				t.Fatalf("error text %q lacks the name:line:col prefix", err)
			}
		})
	}
}

func TestParseAcceptsCommentsAndLayout(t *testing.T) {
	src := `// line comment
/* block
   comment */ .decl r(a:symbol, b:symbol) // trailing
.input r
.decl s(a:symbol)
.output s
s(X) :- /* inline */ r(X, _),
        r(_, X).
`
	p, err := Parse(src, "c.dl")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(p.InputRelations(), ","); got != "r" {
		t.Fatalf("inputs %q", got)
	}
	if got := strings.Join(p.OutputRelations(), ","); got != "s" {
		t.Fatalf("outputs %q", got)
	}
}

func TestParseEmptyProgram(t *testing.T) {
	p, err := Parse("  // nothing\n", "e.dl")
	if err != nil {
		t.Fatal(err)
	}
	res, err := p.Run(nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.OutputRelations()) != 0 {
		t.Fatal("unexpected outputs")
	}
}

func TestParseTooManyAttributes(t *testing.T) {
	var attrs []string
	for i := 0; i <= maxArity; i++ {
		attrs = append(attrs, "a"+strings.Repeat("x", i)+":symbol")
	}
	_, err := Parse(".decl wide("+strings.Join(attrs, ", ")+")\n", "w.dl")
	if err == nil || !strings.Contains(err.Error(), "at most 64 are supported") {
		t.Fatalf("got %v", err)
	}
}

func TestTypeString(t *testing.T) {
	if Symbol.String() != "symbol" || Number.String() != "number" {
		t.Fatal("type names")
	}
}
