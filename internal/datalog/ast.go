package datalog

// Type is an attribute type: symbol, or number for aggregate results.
type Type int

// The two attribute types the subset admits.
const (
	Symbol Type = iota
	Number
)

func (t Type) String() string {
	if t == Number {
		return "number"
	}
	return "symbol"
}

// Attr is one declared attribute of a relation.
type Attr struct {
	Name string
	Type Type
}

// Decl is a relation declaration.
type Decl struct {
	Name   string
	Attrs  []Attr
	Input  bool
	Output bool
	Pos    Pos
}

type termKind int

const (
	termVar termKind = iota
	termConst
	termWild
)

type term struct {
	kind termKind
	text string // variable name or constant value
	pos  Pos
}

type atom struct {
	rel  string
	args []term
	pos  Pos
}

type litKind int

const (
	litAtom litKind = iota
	litNeg
	litCmp
	litAgg
)

type aggKind int

const (
	aggCount aggKind = iota
	aggMin
)

type literal struct {
	kind litKind
	pos  Pos
	atom atom // litAtom, litNeg

	// litCmp
	neq         bool
	left, right term

	// litAgg: result = agg [target] : { body }
	result string
	agg    aggKind
	target term
	body   []literal
}

type rule struct {
	index int // 1-based, source order
	head  atom
	body  []literal
	pos   Pos
}

type directive struct {
	output bool
	rel    string
	pos    Pos
}
