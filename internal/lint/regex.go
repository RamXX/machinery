package lint

import "regexp"

// Mirrors machine_lint.IDENT-based regexes used by namedunit_names.
//
//	re.findall(r"`(IDENT)`", cell)  -> backtick-wrapped identifiers
var (
	regexpBacktickGroup = regexp.MustCompile("`([A-Za-z_][A-Za-z0-9_]*)`")
	// A whole named-unit name (guard/action/actor) must be a single identifier:
	// the same IDENT the contract-table extraction above recognizes, anchored.
	// A hyphenated name passes structural lint but can never be extracted from
	// the contract table, so G3 can never reconcile it; reject it at lint.
	regexpIdent = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	// State and event names flow into oracle stable-id hash inputs (joined
	// with '|') and TLA+ identifiers; every shipped example machine satisfies
	// this pattern. A '|' would collide stable ids, and spaces or hyphens
	// produce invalid TLA+ identifiers.
	regexpName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	// _oracle_tag overrides the derived 4-rune oracle tag so two machines in
	// one design (Deal vs DealAggregate) can disambiguate without renaming.
	regexpOracleTag = regexp.MustCompile(`^[A-Z0-9]{2,8}$`)
	// A raw-millisecond after key ("5000") bypasses the named-delay contract.
	regexpAllDigits = regexp.MustCompile(`^[0-9]+$`)
)
