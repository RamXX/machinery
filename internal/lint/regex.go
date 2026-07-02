package lint

import "regexp"

// Mirrors machine_lint.IDENT-based regexes used by namedunit_names.
//
//	re.findall(r"`(IDENT)`", cell)  -> backtick-wrapped identifiers
//	re.findall(r"\b(IDENT)\b", cell) -> word-boundary identifiers
var (
	regexpBacktickGroup = regexp.MustCompile("`([A-Za-z_][A-Za-z0-9_]*)`")
	regexpWord          = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\b`)
)
