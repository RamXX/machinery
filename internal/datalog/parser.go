package datalog

import "fmt"

// reserved are Soufflé keywords, functor names, types and qualifiers. They
// cannot name a relation, an attribute or a variable, because Soufflé would
// read them as something else.
var reserved = map[string]bool{
	"as": true, "autoinc": true, "band": true, "bnot": true, "bor": true,
	"bshl": true, "bshr": true, "bshru": true, "bxor": true, "cat": true,
	"contains": true, "count": true, "false": true, "land": true, "lnot": true,
	"lor": true, "lxor": true, "match": true, "max": true, "mean": true,
	"min": true, "nil": true, "ord": true, "range": true, "strlen": true,
	"substr": true, "sum": true, "to_float": true, "to_number": true,
	"to_string": true, "to_unsigned": true, "true": true, "itou": true,
	"itof": true, "utoi": true, "utof": true, "ftoi": true, "ftou": true,
	"symbol": true, "number": true, "float": true, "unsigned": true,
	"input": true, "output": true, "printsize": true, "brie": true,
	"btree": true, "btree_delete": true, "eqrel": true, "inline": true,
	"no_inline": true, "magic": true, "no_magic": true, "overridable": true,
	"choice": true, "override": true, "subsumes": true, "mod": true,
	"lambda": true, "global": true, "stateful": true, "exp": true,
}

// qualifiers are Soufflé relation qualifiers; each is a named rejection.
var qualifiers = map[string]bool{
	"brie": true, "btree": true, "btree_delete": true, "eqrel": true,
	"inline": true, "no_inline": true, "magic": true, "no_magic": true,
	"overridable": true, "choice": true, "input": true, "output": true,
	"printsize": true,
}

var unsupportedDirectives = map[string]bool{
	"type": true, "plan": true, "printsize": true, "functor": true,
	"comp": true, "init": true, "pragma": true, "override": true,
	"limitsize": true, "include": true, "once": true, "lattice": true,
}

type parser struct {
	lx  *lexer
	buf []token
}

func (p *parser) errAt(pos Pos, format string, args ...any) *Error {
	return p.lx.errAt(pos, format, args...)
}

// peek returns the k-th token ahead, lexing lazily so errors surface in
// source order.
func (p *parser) peek(k int) (token, error) {
	for len(p.buf) <= k {
		if n := len(p.buf); n > 0 && p.buf[n-1].kind == tEOF {
			return p.buf[n-1], nil
		}
		t, err := p.lx.next()
		if err != nil {
			return token{}, err
		}
		p.buf = append(p.buf, t)
	}
	return p.buf[k], nil
}

func (p *parser) take() (token, error) {
	t, err := p.peek(0)
	if err != nil {
		return t, err
	}
	if t.kind != tEOF {
		p.buf = p.buf[1:]
	}
	return t, nil
}

func (p *parser) expect(k tokKind, context string) (token, error) {
	t, err := p.take()
	if err != nil {
		return t, err
	}
	if t.kind != k {
		return t, p.errAt(t.pos, "expected %s %s, found %s", tokNames[k], context, t.describe())
	}
	return t, nil
}

func (p *parser) name(context string) (token, error) {
	t, err := p.expect(tIdent, context)
	if err != nil {
		return t, err
	}
	if t.text == "_" {
		return t, p.errAt(t.pos, "the wildcard _ cannot be used %s", context)
	}
	if reserved[t.text] {
		return t, p.errAt(t.pos, "%q is a reserved Soufflé word and cannot be used %s", t.text, context)
	}
	return t, nil
}

type parsed struct {
	decls      []*Decl
	directives []directive
	rules      []*rule
}

func parse(name, src string) (*parsed, error) {
	p := &parser{lx: &lexer{name: name, src: src, line: 1, col: 1}}
	out := &parsed{}
	for {
		t, err := p.peek(0)
		if err != nil {
			return nil, err
		}
		switch t.kind {
		case tEOF:
			return out, nil
		case tDirective:
			if err := p.directive(out); err != nil {
				return nil, err
			}
		case tIdent:
			r, err := p.rule()
			if err != nil {
				return nil, err
			}
			r.index = len(out.rules) + 1
			out.rules = append(out.rules, r)
		default:
			return nil, p.errAt(t.pos, "expected a directive or a rule, found %s", t.describe())
		}
	}
}

func (p *parser) directive(out *parsed) error {
	t, _ := p.take()
	switch t.text {
	case "decl":
		d, err := p.decl()
		if err != nil {
			return err
		}
		out.decls = append(out.decls, d)
		return nil
	case "input", "output":
		n, err := p.name("as a relation name")
		if err != nil {
			return err
		}
		nx, err := p.peek(0)
		if err != nil {
			return err
		}
		switch nx.kind {
		case tLParen:
			return p.errAt(nx.pos, ".%s parameters are not supported; facts are read from <relation>.facts and written to <relation>.csv", t.text)
		case tComma:
			return p.errAt(nx.pos, "one relation per .%s directive", t.text)
		}
		out.directives = append(out.directives, directive{output: t.text == "output", rel: n.text, pos: t.pos})
		return nil
	}
	if unsupportedDirectives[t.text] {
		return p.errAt(t.pos, "directive .%s is not supported", t.text)
	}
	return p.errAt(t.pos, "unknown directive .%s", t.text)
}

func (p *parser) decl() (*Decl, error) {
	n, err := p.name("as a relation name")
	if err != nil {
		return nil, err
	}
	d := &Decl{Name: n.text, Pos: n.pos}
	if _, err := p.expect(tLParen, "after the relation name"); err != nil {
		return nil, err
	}
	if t, err := p.peek(0); err != nil {
		return nil, err
	} else if t.kind == tRParen {
		return nil, p.errAt(t.pos, "nullary relations are not supported")
	}
	seen := map[string]bool{}
	for {
		an, err := p.name("as an attribute name")
		if err != nil {
			return nil, err
		}
		if seen[an.text] {
			return nil, p.errAt(an.pos, "attribute %q declared twice in relation %q", an.text, d.Name)
		}
		seen[an.text] = true
		if _, err := p.expect(tColon, "after the attribute name"); err != nil {
			return nil, err
		}
		ty, err := p.expect(tIdent, "as the attribute type")
		if err != nil {
			return nil, err
		}
		var typ Type
		switch ty.text {
		case "symbol":
			typ = Symbol
		case "number":
			typ = Number
		default:
			return nil, p.errAt(ty.pos, "type %q is not supported (only symbol, and number for aggregate results)", ty.text)
		}
		d.Attrs = append(d.Attrs, Attr{Name: an.text, Type: typ})
		t, err := p.take()
		if err != nil {
			return nil, err
		}
		if t.kind == tRParen {
			break
		}
		if t.kind != tComma {
			return nil, p.errAt(t.pos, "expected ',' or ')' in the attribute list, found %s", t.describe())
		}
	}
	nx, err := p.peek(0)
	if err != nil {
		return nil, err
	}
	if nx.kind == tIdent && qualifiers[nx.text] {
		return nil, p.errAt(nx.pos, "relation qualifier %q is not supported", nx.text)
	}
	return d, nil
}

func (p *parser) term(context string) (term, error) {
	t, err := p.take()
	if err != nil {
		return term{}, err
	}
	switch t.kind {
	case tString:
		return term{kind: termConst, text: t.text, pos: t.pos}, nil
	case tIdent:
		if t.text == "_" {
			return term{kind: termWild, pos: t.pos}, nil
		}
		nx, err := p.peek(0)
		if err != nil {
			return term{}, err
		}
		if nx.kind == tLParen {
			return term{}, p.errAt(t.pos, "functors are not supported (%s(...))", t.text)
		}
		if reserved[t.text] {
			return term{}, p.errAt(t.pos, "%q is a reserved Soufflé word and cannot be used as a variable", t.text)
		}
		return term{kind: termVar, text: t.text, pos: t.pos}, nil
	}
	return term{}, p.errAt(t.pos, "expected a variable, _ or a string %s, found %s", context, t.describe())
}

func (p *parser) atom() (atom, error) {
	n, err := p.name("as a relation name")
	if err != nil {
		return atom{}, err
	}
	a := atom{rel: n.text, pos: n.pos}
	if _, err := p.expect(tLParen, fmt.Sprintf("after %q", n.text)); err != nil {
		return atom{}, err
	}
	if t, err := p.peek(0); err != nil {
		return atom{}, err
	} else if t.kind == tRParen {
		return atom{}, p.errAt(t.pos, "nullary atoms are not supported")
	}
	for {
		tm, err := p.term("as an argument")
		if err != nil {
			return atom{}, err
		}
		a.args = append(a.args, tm)
		t, err := p.take()
		if err != nil {
			return atom{}, err
		}
		if t.kind == tRParen {
			return a, nil
		}
		if t.kind != tComma {
			return atom{}, p.errAt(t.pos, "expected ',' or ')' in the argument list, found %s", t.describe())
		}
	}
}

func (p *parser) rule() (*rule, error) {
	head, err := p.atom()
	if err != nil {
		return nil, err
	}
	r := &rule{head: head, pos: head.pos}
	t, err := p.take()
	if err != nil {
		return nil, err
	}
	switch t.kind {
	case tIf:
	case tDot:
		return nil, p.errAt(head.pos, "facts in the program are not supported; supply them through .input")
	case tComma:
		return nil, p.errAt(t.pos, "rules with multiple heads are not supported")
	default:
		return nil, p.errAt(t.pos, "expected ':-' after the rule head, found %s", t.describe())
	}
	body, err := p.body(tDot, false)
	if err != nil {
		return nil, err
	}
	r.body = body
	return r, nil
}

// body parses literals separated by commas up to (and consuming) end.
func (p *parser) body(end tokKind, inAgg bool) ([]literal, error) {
	var lits []literal
	for {
		l, err := p.literal(inAgg)
		if err != nil {
			return nil, err
		}
		lits = append(lits, l)
		t, err := p.take()
		if err != nil {
			return nil, err
		}
		if t.kind == end {
			return lits, nil
		}
		if t.kind != tComma {
			return nil, p.errAt(t.pos, "expected ',' or %s after a body literal, found %s", tokNames[end], t.describe())
		}
	}
}

func (p *parser) literal(inAgg bool) (literal, error) {
	t, err := p.peek(0)
	if err != nil {
		return literal{}, err
	}
	if t.kind == tBang {
		_, _ = p.take() // the peeked token; peek already surfaced any error
		a, err := p.atom()
		if err != nil {
			return literal{}, err
		}
		return literal{kind: litNeg, pos: t.pos, atom: a}, nil
	}
	if t.kind == tIdent && t.text != "_" {
		nx, err := p.peek(1)
		if err != nil {
			return literal{}, err
		}
		if nx.kind == tLParen {
			a, err := p.atom()
			if err != nil {
				return literal{}, err
			}
			return literal{kind: litAtom, pos: a.pos, atom: a}, nil
		}
	}
	left, err := p.term("in a body literal")
	if err != nil {
		return literal{}, err
	}
	op, err := p.take()
	if err != nil {
		return literal{}, err
	}
	if op.kind != tEq && op.kind != tNeq {
		return literal{}, p.errAt(op.pos, "expected '=' or '!=' after %q, found %s", left.text, op.describe())
	}
	rt, err := p.peek(0)
	if err != nil {
		return literal{}, err
	}
	if rt.kind == tIdent {
		switch rt.text {
		case "count", "min":
			if op.kind != tEq {
				return literal{}, p.errAt(op.pos, "an aggregate must be bound with '='")
			}
			if inAgg {
				return literal{}, p.errAt(rt.pos, "nested aggregates are not supported")
			}
			if left.kind != termVar {
				return literal{}, p.errAt(left.pos, "an aggregate result must be bound to a variable")
			}
			return p.aggregate(left)
		case "max", "sum", "mean", "range":
			return literal{}, p.errAt(rt.pos, "aggregate %q is not supported (only count and min)", rt.text)
		}
	}
	right, err := p.term("in a comparison")
	if err != nil {
		return literal{}, err
	}
	return literal{kind: litCmp, pos: left.pos, neq: op.kind == tNeq, left: left, right: right}, nil
}

func (p *parser) aggregate(result term) (literal, error) {
	kw, _ := p.take()
	l := literal{kind: litAgg, pos: result.pos, result: result.text}
	if kw.text == "min" {
		l.agg = aggMin
		tg, err := p.term("as the min target")
		if err != nil {
			return literal{}, err
		}
		if tg.kind != termVar {
			return literal{}, p.errAt(tg.pos, "the min target must be a variable")
		}
		l.target = tg
	}
	if _, err := p.expect(tColon, "after "+kw.text); err != nil {
		return literal{}, err
	}
	if _, err := p.expect(tLBrace, "to open the aggregate body (the braced form is required)"); err != nil {
		return literal{}, err
	}
	body, err := p.body(tRBrace, true)
	if err != nil {
		return literal{}, err
	}
	l.body = body
	return l, nil
}
