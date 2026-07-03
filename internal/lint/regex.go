package lint

import "regexp"

// Mirrors machine_lint.IDENT-based regexes used by namedunit_names.
//
//	re.findall(r"`(IDENT)`", cell)  -> backtick-wrapped identifiers
//	re.findall(r"\b(IDENT)\b", cell) -> word-boundary identifiers
var (
	regexpBacktickGroup = regexp.MustCompile("`([A-Za-z_][A-Za-z0-9_]*)`")
	regexpWord          = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\b`)
	// A whole named-unit name (guard/action/actor) must be a single identifier:
	// the same IDENT the contract-table extraction above recognizes, anchored.
	// A hyphenated name passes structural lint but can never be extracted from
	// the contract table, so G3 can never reconcile it; reject it at lint.
	regexpIdent = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)
