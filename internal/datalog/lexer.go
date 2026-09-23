package datalog

import (
	"fmt"
	"strings"
)

// Pos is a 1-based source position. Col counts bytes.
type Pos struct {
	Line int
	Col  int
}

func (p Pos) String() string { return fmt.Sprintf("%d:%d", p.Line, p.Col) }

// Error is a parse or compile error with the source name and position.
type Error struct {
	Name string
	Pos  Pos
	Msg  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Name, e.Pos.Line, e.Pos.Col, e.Msg)
}

type tokKind int

const (
	tEOF       tokKind = iota
	tIdent             // identifier, including the wildcard _
	tString            // "..." (text holds the unquoted value)
	tDirective         // .decl, .input, ... (text holds the name without the dot)
	tLParen
	tRParen
	tLBrace
	tRBrace
	tComma
	tColon
	tIf  // :-
	tDot // clause terminator
	tBang
	tNeq // !=
	tEq  // =
)

var tokNames = map[tokKind]string{
	tEOF: "end of input", tIdent: "identifier", tString: "string",
	tDirective: "directive", tLParen: "'('", tRParen: "')'", tLBrace: "'{'",
	tRBrace: "'}'", tComma: "','", tColon: "':'", tIf: "':-'", tDot: "'.'",
	tBang: "'!'", tNeq: "'!='", tEq: "'='",
}

type token struct {
	kind tokKind
	text string
	pos  Pos
}

func (t token) describe() string {
	switch t.kind {
	case tIdent:
		return fmt.Sprintf("identifier %q", t.text)
	case tString:
		return fmt.Sprintf("string %q", t.text)
	case tDirective:
		return "directive ." + t.text
	}
	return tokNames[t.kind]
}

// unsupportedChars names the Soufflé feature a stray character most likely
// belongs to, so the rejection says what was refused rather than just where.
var unsupportedChars = map[byte]string{
	';':  "disjunction (';') is not supported",
	'+':  "arithmetic is not supported",
	'-':  "arithmetic is not supported",
	'*':  "arithmetic is not supported",
	'/':  "arithmetic is not supported",
	'%':  "arithmetic is not supported",
	'^':  "arithmetic is not supported",
	'&':  "arithmetic is not supported",
	'|':  "arithmetic is not supported",
	'~':  "arithmetic is not supported",
	'<':  "only = and != comparisons are supported",
	'>':  "only = and != comparisons are supported",
	'[':  "records are not supported",
	']':  "records are not supported",
	'$':  "algebraic data types are not supported",
	'@':  "user-defined functors are not supported",
	'#':  "the C preprocessor is not supported",
	'\\': "a backslash outside a string is not supported",
	'?':  "identifiers containing '?' are not supported",
	'\'': "single-quoted strings are not supported",
}

type lexer struct {
	name string
	src  string
	i    int
	line int
	col  int
}

func (lx *lexer) errAt(p Pos, format string, args ...any) *Error {
	return &Error{Name: lx.name, Pos: p, Msg: fmt.Sprintf(format, args...)}
}

func (lx *lexer) pos() Pos { return Pos{lx.line, lx.col} }

func (lx *lexer) advance() {
	if lx.src[lx.i] == '\n' {
		lx.line++
		lx.col = 1
	} else {
		lx.col++
	}
	lx.i++
}

func (lx *lexer) peekAt(k int) byte {
	if lx.i+k < len(lx.src) {
		return lx.src[lx.i+k]
	}
	return 0
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentChar(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

// skipSpace consumes whitespace and comments.
func (lx *lexer) skipSpace() error {
	for lx.i < len(lx.src) {
		c := lx.src[lx.i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			lx.advance()
		case c == '/' && lx.peekAt(1) == '/':
			for lx.i < len(lx.src) && lx.src[lx.i] != '\n' {
				lx.advance()
			}
		case c == '/' && lx.peekAt(1) == '*':
			start := lx.pos()
			lx.advance()
			lx.advance()
			for {
				if lx.i >= len(lx.src) {
					return lx.errAt(start, "unterminated block comment")
				}
				if lx.src[lx.i] == '*' && lx.peekAt(1) == '/' {
					lx.advance()
					lx.advance()
					break
				}
				lx.advance()
			}
		default:
			return nil
		}
	}
	return nil
}

func (lx *lexer) next() (token, error) {
	if err := lx.skipSpace(); err != nil {
		return token{}, err
	}
	p := lx.pos()
	if lx.i >= len(lx.src) {
		return token{kind: tEOF, pos: p}, nil
	}
	c := lx.src[lx.i]
	single := func(k tokKind, n int) (token, error) {
		for j := 0; j < n; j++ {
			lx.advance()
		}
		return token{kind: k, pos: p}, nil
	}
	switch {
	case isIdentStart(c):
		start := lx.i
		for lx.i < len(lx.src) && isIdentChar(lx.src[lx.i]) {
			lx.advance()
		}
		if lx.peekAt(0) == '?' {
			return token{}, lx.errAt(lx.pos(), "%s", unsupportedChars['?'])
		}
		return token{kind: tIdent, text: lx.src[start:lx.i], pos: p}, nil
	case c >= '0' && c <= '9':
		return token{}, lx.errAt(p, "number literals are not supported")
	case c == '"':
		lx.advance()
		var b strings.Builder
		for {
			if lx.i >= len(lx.src) {
				return token{}, lx.errAt(p, "unterminated string")
			}
			ch := lx.src[lx.i]
			switch ch {
			case '"':
				lx.advance()
				return token{kind: tString, text: b.String(), pos: p}, nil
			case '\\':
				return token{}, lx.errAt(lx.pos(), "escape sequences in strings are not supported")
			case '\n', '\r', '\t':
				return token{}, lx.errAt(lx.pos(), "a string may not contain a tab, newline or carriage return")
			}
			b.WriteByte(ch)
			lx.advance()
		}
	case c == '.':
		if isIdentStart(lx.peekAt(1)) {
			lx.advance()
			start := lx.i
			for lx.i < len(lx.src) && isIdentChar(lx.src[lx.i]) {
				lx.advance()
			}
			return token{kind: tDirective, text: lx.src[start:lx.i], pos: p}, nil
		}
		return single(tDot, 1)
	case c == '(':
		return single(tLParen, 1)
	case c == ')':
		return single(tRParen, 1)
	case c == '{':
		return single(tLBrace, 1)
	case c == '}':
		return single(tRBrace, 1)
	case c == ',':
		return single(tComma, 1)
	case c == ':':
		if lx.peekAt(1) == '-' {
			return single(tIf, 2)
		}
		return single(tColon, 1)
	case c == '!':
		if lx.peekAt(1) == '=' {
			return single(tNeq, 2)
		}
		return single(tBang, 1)
	case c == '=':
		return single(tEq, 1)
	}
	if msg, ok := unsupportedChars[c]; ok {
		return token{}, lx.errAt(p, "%s", msg)
	}
	if c < 0x20 || c >= 0x7f {
		return token{}, lx.errAt(p, "unexpected byte 0x%02x", c)
	}
	return token{}, lx.errAt(p, "unexpected character %q", string(c))
}
