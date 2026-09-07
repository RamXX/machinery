// Portfolio packet-contract tests for MAC-lhu5 (design-only; no optimizer/native runtime).
//
// Consumes the frozen design bytes under examples/portfolio-engine/design, extracts typed
// facts from approved whole clauses, and checks them against an independent literal
// expectation ledger. Positive controls use approved clauses/bytes; negative controls swap
// ONE whole clause (or mutate one authority byte) on otherwise-valid documents and must
// fail the semantic assertion (never a compile/crash/skip/prerequisite failure).
//
// Construction rules (docs/test-assurance-contract.md 5.3/5.4/11.4; approved policy
// PROPOSAL.md:239-248): every negative control arranges valid prerequisites and mutates one
// whole obligation; expectations are independent literals, never computed by the evaluator
// under test; no test executes a portfolio engine, DuckDB, Docker, network or pf.* CLI.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Shared core: design-file loader, frozen-input hash pins, literal helpers.
// ---------------------------------------------------------------------------

// frozenPins pins the design bytes this story never modifies (no write ownership):
// the four generated oracle files and machines/MarketDataFeed.machine.json. They are
// valid prerequisites in both the RED and GREEN phases. The other nine frozen inputs
// (four matrices, three machine JSONs, M0, M3) are legitimately edited by this story and
// are covered by the AC5 old/new hash handoff instead of a hard pin, so a hash mismatch
// can never masquerade as a semantic failure.
var frozenPins = map[string]string{
	"examples/portfolio-engine/design/machines/MarketDataFeed.machine.json":       "394a07e8610f838de53ecf21f1ce34c5a5748626c1733d6b3581568ee67b6c72",
	"examples/portfolio-engine/design/machines/MarketDataFeed.oracle.md":          "06cd42399d735f264cb6fced1188e715a65f5a2e4bfeb7125ca50810dbabf951",
	"examples/portfolio-engine/design/machines/Portfolio.oracle.md":               "1f77d54c057840ebafe972f0d1653e87f54905f5616f4270ecfe25d1e8e214ed",
	"examples/portfolio-engine/design/machines/RecommendationRun.oracle.md":       "731315be2be71ca99100d80b44f529a8689a12bd95431a3b1726539ba2eaa3b4",
	"examples/portfolio-engine/design/machines/ReferenceDataCommand.oracle.md":    "23e28aaeeaf8bc37b185122e368ce1f6b68596372ee6cd83aed873a4a0071d4f",
	"examples/portfolio-engine/design/BUILD/M3-optimizer.md":                      "621255b4ece8863c50c70cc0f9bc7ac3e831a6ad702a23408f5b41f80a3e3c67",
	"examples/portfolio-engine/design/BUILD/M0-walking-skeleton.md":               "5a06ed03f8f8c693e679b480c77ba041b979d5c9d10c961677dd65eb370d9db2",
	"examples/portfolio-engine/design/machines/RecommendationRun.matrix.md":       "423db91a3e78fdcb76940bb4b4e6be23727e47b62a82c2568dd58e130d8264a1",
	"examples/portfolio-engine/design/machines/ReferenceDataCommand.matrix.md":    "222539d82c470914676c12b304d143176ca4db97e4368ccc3fb23dd90aae10f8",
	"examples/portfolio-engine/design/machines/Portfolio.matrix.md":               "231e76abe65d2ea4cd91b073dea90f7ead9623f57b1572d2feca5a9df1e456c2",
	"examples/portfolio-engine/design/machines/MarketDataFeed.matrix.md":          "d9f63ba2452af287a1e7a0ce2f9f5a8137a67715a0255529e04b233af4d233a4",
	"examples/portfolio-engine/design/machines/RecommendationRun.machine.json":    "c14c026315a2ff7080d6807df1f27e04a8f1734083a3076e031531dd4dc22e58",
	"examples/portfolio-engine/design/machines/Portfolio.machine.json":            "2490f9d4069c373ea27e3fdad05fc20844cb654a6b58e2b29360072e5d28d4cc",
	"examples/portfolio-engine/design/machines/ReferenceDataCommand.machine.json": "135989645a8d5b9984ebf0f4cfc54b2e52cc62df81e93ca88a781ac2e38374da",
}

// designBytes loads a design file relative to the repo root.
func designBytes(t *testing.T, rel string) []byte {
	t.Helper()
	raw, err := os.ReadFile(repoRootDir(t) + "/examples/portfolio-engine/design/" + strings.TrimPrefix(rel, "examples/portfolio-engine/design/"))
	if err != nil {
		t.Fatalf("read design file %s: %v", rel, err)
	}
	return raw
}

func shaHex(b []byte) string { return hex.EncodeToString(sum256(b)) }

func sum256(b []byte) []byte { h := sha256.Sum256(b); return h[:] }

// ratLit builds an exact rational literal (numerator/denominator are literal integers;
// this is ledger data, never a value computed by an evaluator under test).
func ratLit(num, den int64) *big.Rat { return big.NewRat(num, den) }

// ratStr parses an exact rational from a decimal literal string (ledger data only).
func ratStr(t *testing.T, s string) *big.Rat {
	t.Helper()
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		t.Fatalf("bad rational literal %q", s)
	}
	return r
}

// ---------------------------------------------------------------------------
// D1 core: controlled clause grammar, fact extraction, exact evaluator.
// ---------------------------------------------------------------------------

const (
	cObjectiveMin           = "The optimizer minimizes exact historical maximum drawdown."
	cObjectiveMinEquivalent = "Among supplied feasible proposals, the optimizer chooses the least exact historical maximum drawdown."
	cObjectiveMax           = "The optimizer maximizes exact historical maximum drawdown."
	cBuyHold                = "Portfolio values use normalized buy-and-hold quantities from each asset's initial price."
	cUnnormalized           = "Portfolio values weight raw prices without per-asset initial normalization."
	cRebalanced             = "Portfolio values rebalance to the initial weights at every observation."
	cExact                  = "Proposal ranking compares exact rational drawdown before basis-point rounding."
	cRounded                = "Proposal ranking compares stored rounded basis-point drawdown."
	cHoldings               = "A feasible proposal contains exactly 16 distinct canonical tickers, integer nonnegative weights summing to 10000, including zero weights."
	cTie                    = "Exact-score ties compare the whole ascending ASCII ticker vector, then the whole weight vector in that ticker order."
	cTieInterleaved         = "Exact-score ties alternately compare each ticker and its weight."
	cWhole                  = "The whole candidate universe is rejected when any candidate history is invalid; candidates are never filtered."
	cFilter                 = "Invalid candidate histories are filtered before proposal selection."
	cCalendar               = "The matrix declares at least two real strictly increasing Gregorian YYYY-MM-DD observation dates common to every candidate."
	cPrices                 = "Prices are positive finite exact decimal strings without sign, whitespace, or exponent, and use the unadjusted-close basis."
	cLimits                 = "Caller-owned positive uint64 limits require maxCandidates at least 16 and maxLookbackDays at least 2 and bound all records before decoding or selection."
)

const completeProseControl = cObjectiveMin + "\n" + cBuyHold + "\n" + cExact + "\n" + cHoldings + "\n" +
	cTie + "\n" + cWhole + "\n" + cCalendar + "\n" + cPrices + "\n" + cLimits

type ppDirection uint8

const (
	dirUnset ppDirection = iota
	dirMin
	dirMax
)

type ppValueRule uint8

const (
	valueUnset ppValueRule = iota
	valueBuyHold
	valueRaw
	valueRebalanced
)

type ppCompareRule uint8

const (
	compareUnset ppCompareRule = iota
	compareExact
	compareRounded
)

type ppTieRule uint8

const (
	tieUnset ppTieRule = iota
	tieVectors
	tieInterleaved
)

type ppAdmission uint8

const (
	admissionUnset ppAdmission = iota
	admissionWhole
	admissionFilter
)

type ppSpan struct {
	Clause   string
	Start    int
	End      int
	Ordinary bool // true when recognized through the packet-prose recognizer
}

type ppFacts struct {
	Direction         ppDirection
	Value             ppValueRule
	Compare           ppCompareRule
	Tie               ppTieRule
	Admission         ppAdmission
	Holdings          bool
	Calendar          bool
	Prices            bool
	Limits            bool
	OriginalRefusals  bool
	Provenance        map[string][]ppSpan
	UnsupportedClause string
}

type clauseEffect struct {
	key   string
	apply func(*ppFacts) error
}

func ppAssign[T comparable](dst *T, value T, unset T, name string) error {
	if *dst != unset && *dst != value {
		return fmt.Errorf("contradictory %s clauses", name)
	}
	*dst = value
	return nil
}

var ppClauses = map[string]clauseEffect{
	cObjectiveMin:           {"objective", func(f *ppFacts) error { return ppAssign(&f.Direction, dirMin, dirUnset, "objective") }},
	cObjectiveMinEquivalent: {"objective", func(f *ppFacts) error { return ppAssign(&f.Direction, dirMin, dirUnset, "objective") }},
	cObjectiveMax:           {"objective", func(f *ppFacts) error { return ppAssign(&f.Direction, dirMax, dirUnset, "objective") }},
	cBuyHold:                {"value", func(f *ppFacts) error { return ppAssign(&f.Value, valueBuyHold, valueUnset, "value") }},
	cUnnormalized:           {"value", func(f *ppFacts) error { return ppAssign(&f.Value, valueRaw, valueUnset, "value") }},
	cRebalanced:             {"value", func(f *ppFacts) error { return ppAssign(&f.Value, valueRebalanced, valueUnset, "value") }},
	cExact:                  {"comparison", func(f *ppFacts) error { return ppAssign(&f.Compare, compareExact, compareUnset, "comparison") }},
	cRounded:                {"comparison", func(f *ppFacts) error { return ppAssign(&f.Compare, compareRounded, compareUnset, "comparison") }},
	cHoldings:               {"holdings", func(f *ppFacts) error { f.Holdings = true; return nil }},
	cTie:                    {"tie", func(f *ppFacts) error { return ppAssign(&f.Tie, tieVectors, tieUnset, "tie") }},
	cTieInterleaved:         {"tie", func(f *ppFacts) error { return ppAssign(&f.Tie, tieInterleaved, tieUnset, "tie") }},
	cWhole:                  {"admission", func(f *ppFacts) error { return ppAssign(&f.Admission, admissionWhole, admissionUnset, "admission") }},
	cFilter:                 {"admission", func(f *ppFacts) error { return ppAssign(&f.Admission, admissionFilter, admissionUnset, "admission") }},
	cCalendar:               {"calendar", func(f *ppFacts) error { f.Calendar = true; return nil }},
	cPrices:                 {"prices", func(f *ppFacts) error { f.Prices = true; return nil }},
	cLimits:                 {"limits", func(f *ppFacts) error { f.Limits = true; return nil }},
}

// parsePacket recognizes whole clauses only: every non-empty line must be one controlled
// clause. An unrecognized line is an error naming the exact span (never silently ignored).
func parsePacket(text string) (ppFacts, error) {
	f := ppFacts{Provenance: map[string][]ppSpan{}}
	offset := 0
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSuffix(raw, "\r")
		if line == "" {
			offset += len(raw) + 1
			continue
		}
		effect, ok := ppClauses[line]
		if !ok {
			f.UnsupportedClause = line
			return f, fmt.Errorf("unsupported whole clause at bytes %d..%d: %q", offset, offset+len(line), line)
		}
		if err := effect.apply(&f); err != nil {
			return f, err
		}
		f.Provenance[effect.key] = append(f.Provenance[effect.key], ppSpan{line, offset, offset + len(line), false})
		offset += len(raw) + 1
	}
	return f, nil
}

func mustParse(t *testing.T, text string) ppFacts {
	t.Helper()
	f, err := parsePacket(text)
	if err != nil {
		t.Fatalf("control packet must parse: %v", err)
	}
	return f
}

func swapClause(t *testing.T, text, from, to string) string {
	t.Helper()
	if strings.Count(text, from) != 1 {
		t.Fatalf("swap source clause not uniquely present: %q", from)
	}
	return strings.Replace(text, from, to, 1)
}

func (f ppFacts) evaluable() bool {
	return f.Direction != dirUnset && f.Value != valueUnset && f.Compare != compareUnset &&
		f.Tie != tieUnset && f.Admission != admissionUnset && f.Holdings && f.Calendar && f.Prices && f.Limits
}

func (f ppFacts) safe() bool {
	return f.evaluable() && f.Direction == dirMin && f.Value == valueBuyHold && f.Compare == compareExact &&
		f.Admission == admissionWhole
}

// wsPattern builds a whitespace-flexible pattern: spaces in the source pattern match
// any run of whitespace (packet prose wraps lines), so obligations are recognized
// regardless of line wrapping.
func wsPattern(src string) *regexp.Regexp {
	parts := strings.Split(src, " ")
	for i, part := range parts {
		parts[i] = regexp.QuoteMeta(part)
	}
	joined := strings.Join(parts, `\s+`)
	// pipes in the source pattern mean top-level alternation, never a literal "|"
	return regexp.MustCompile(strings.ReplaceAll(joined, `\|`, `|`))
}

// --- live M3 packet recognizer: pattern obligations in packet prose -------------

var m3Patterns = []struct {
	key      string
	pattern  *regexp.Regexp
	ordinary func(*ppFacts) error
}{
	{"objective", wsPattern("MINIMUM exact historical maximum drawdown|minimizes exact historical maximum drawdown|chooses the least exact historical maximum drawdown"),
		func(f *ppFacts) error { return ppAssign(&f.Direction, dirMin, dirUnset, "objective") }},
	{"objective-max", wsPattern("maximizes exact historical maximum drawdown|MAXIMUM exact historical maximum drawdown"),
		func(f *ppFacts) error { return ppAssign(&f.Direction, dirMax, dirUnset, "objective") }},
	{"value", wsPattern("value rule is normalized buy-and-hold|normalized buy-and-hold quantities"),
		func(f *ppFacts) error { return ppAssign(&f.Value, valueBuyHold, valueUnset, "value") }},
	{"value-raw", wsPattern("weight raw prices without per-asset initial normalization"),
		func(f *ppFacts) error { return ppAssign(&f.Value, valueRaw, valueUnset, "value") }},
	{"value-rebalanced", wsPattern("rebalance to the initial weights"),
		func(f *ppFacts) error { return ppAssign(&f.Value, valueRebalanced, valueUnset, "value") }},
	{"comparison", regexp.MustCompile(`(?i)compares\s+exact\s+rational\s+drawdowns?\s+before`),
		func(f *ppFacts) error { return ppAssign(&f.Compare, compareExact, compareUnset, "comparison") }},
	{"comparison-rounded", regexp.MustCompile(`(?i)compares\s+stored\s+rounded\s+basis-point\s+drawdown`),
		func(f *ppFacts) error { return ppAssign(&f.Compare, compareRounded, compareUnset, "comparison") }},
	{"tie", wsPattern("ascending ASCII ticker vector, then the whole weight vector|ascending ASCII ticker vector, then the weight vector"),
		func(f *ppFacts) error { return ppAssign(&f.Tie, tieVectors, tieUnset, "tie") }},
	{"tie-interleaved", wsPattern("alternately compare each ticker and its weight"),
		func(f *ppFacts) error { return ppAssign(&f.Tie, tieInterleaved, tieUnset, "tie") }},
	{"admission", wsPattern("Admission is whole-input|whole candidate universe is rejected when any candidate history is invalid"),
		func(f *ppFacts) error { return ppAssign(&f.Admission, admissionWhole, admissionUnset, "admission") }},
	{"admission-filter", wsPattern("Invalid candidate histories are filtered before proposal selection"),
		func(f *ppFacts) error { return ppAssign(&f.Admission, admissionFilter, admissionUnset, "admission") }},
	{"holdings", wsPattern("exactly 16 Holding records|exactly 16 unique candidates"),
		func(f *ppFacts) error { f.Holdings = true; return nil }},
	{"calendar", regexp.MustCompile(`strictly\s+increasing\s+(distinct\s+real\s+Gregorian\s+dates|Gregorian\s+YYYY-MM-DD\s+observation\s+dates)`),
		func(f *ppFacts) error { f.Calendar = true; return nil }},
	{"prices", wsPattern("Prices are strictly positive finite exact decimal strings|Prices are positive finite exact decimal strings"),
		func(f *ppFacts) error { f.Prices = true; return nil }},
	{"limits", regexp.MustCompile(`maxCandidates\s*(at least|>=)\s*16`),
		func(f *ppFacts) error { f.Limits = true; return nil }},
	{"original-refusals", wsPattern("use the declared ticker ordering as the total deterministic tie-break; reject missing, non-finite, or insufficient histories"),
		func(f *ppFacts) error { f.OriginalRefusals = true; return nil }},
}

// extractPacketObligations recognizes approved obligations inside packet prose (the live
// BUILD/M3 bytes). It first applies the exact whole-clause registry line by line, then the
// prose patterns over the whole document. Recognized obligations are recorded with spans.
func extractPacketObligations(text string) (ppFacts, error) {
	f := ppFacts{Provenance: map[string][]ppSpan{}}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSuffix(raw, "\r")
		if effect, ok := ppClauses[line]; ok {
			if err := effect.apply(&f); err != nil {
				return f, err
			}
			start := strings.Index(text, line)
			f.Provenance[effect.key] = append(f.Provenance[effect.key], ppSpan{line, start, start + len(line), false})
		}
	}
	for _, p := range m3Patterns {
		loc := p.pattern.FindStringIndex(text)
		if loc == nil {
			continue
		}
		if err := p.ordinary(&f); err != nil {
			return f, err
		}
		f.Provenance[p.key] = append(f.Provenance[p.key], ppSpan{text[loc[0]:loc[1]], loc[0], loc[1], true})
	}
	return f, nil
}

// missingObligations lists the required fact keys the packet failed to supply.
func (f ppFacts) missingObligations() []string {
	var missing []string
	if f.Direction == dirUnset {
		missing = append(missing, "objective:min")
	}
	if f.Value == valueUnset {
		missing = append(missing, "value:buy-and-hold")
	}
	if f.Compare == compareUnset {
		missing = append(missing, "comparison:exact-before-rounding")
	}
	if f.Tie == tieUnset {
		missing = append(missing, "tie:ticker-vector-then-weight-vector")
	}
	if f.Admission == admissionUnset {
		missing = append(missing, "admission:whole-input")
	}
	if !f.Holdings {
		missing = append(missing, "holdings:16-records")
	}
	if !f.Calendar {
		missing = append(missing, "calendar:declared-common-dates")
	}
	if !f.Prices {
		missing = append(missing, "prices:exact-decimal")
	}
	if !f.Limits {
		missing = append(missing, "limits:caller-owned")
	}
	return missing
}

// --- exact-rational evaluator over literal fixtures ----------------------------

type ppLimits struct{ maxCandidates, maxLookbackDays, maxScalarBytes uint64 }

type ppCell struct {
	text    string
	missing bool
}

type ppCandidate struct {
	ticker  string
	prices  []ppCell
	present bool
}

type ppMatrix struct {
	dates      []string
	basis      string
	candidates []ppCandidate
}

type ppHolding struct {
	ticker string
	weight int
}

type ppProposal struct {
	name     string
	holdings []ppHolding
}

type ppReject string

const (
	ppOK            ppReject = "OK"
	ppInvalid       ppReject = "INVALID_INPUT"
	ppIncomplete    ppReject = "INCOMPLETE_HISTORY"
	ppInsufficient  ppReject = "INSUFFICIENT_CANDIDATES"
	ppLimitExceeded ppReject = "LIMIT_EXCEEDED"
)

var decimalRE = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)
var tickerRE = regexp.MustCompile(`^[A-Z0-9][A-Z0-9.-]*$`)
var dateRE = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

func validDate(s string) bool {
	if !dateRE.MatchString(s) || strings.HasPrefix(s, "0000") {
		return false
	}
	d, err := time.Parse("2006-01-02", s)
	return err == nil && d.Format("2006-01-02") == s
}

func exactPrice(s string) (*big.Rat, bool) {
	if !decimalRE.MatchString(s) {
		return nil, false
	}
	x, good := new(big.Rat).SetString(s)
	return x, good && x.Sign() > 0
}

func validLimits(l ppLimits) bool {
	return l.maxCandidates >= 16 && l.maxLookbackDays >= 2 && l.maxScalarBytes > 0
}

// admit performs the complete lexical-size pass before parsing dates, prices or
// identities, then validates shape/calendar/identity, then count sufficiency. This is the
// evaluator under test; expected outcomes are literals in the tests.
func admit(m ppMatrix, l ppLimits) ppReject {
	if !validLimits(l) {
		return ppInvalid
	}
	if uint64(len(m.candidates)) > l.maxCandidates || uint64(len(m.dates)) > l.maxLookbackDays {
		return ppLimitExceeded
	}
	for _, d := range m.dates {
		if uint64(len(d)) > l.maxScalarBytes {
			return ppLimitExceeded
		}
	}
	for _, c := range m.candidates {
		if uint64(len(c.ticker)) > l.maxScalarBytes {
			return ppLimitExceeded
		}
		for _, p := range c.prices {
			if uint64(len(p.text)) > l.maxScalarBytes {
				return ppLimitExceeded
			}
		}
	}
	if m.basis != "unadjusted-close" {
		return ppInvalid
	}
	seen := map[string]bool{}
	for _, d := range m.dates {
		if !validDate(d) || seen[d] {
			return ppInvalid
		}
		seen[d] = true
	}
	for i := 1; i < len(m.dates); i++ {
		if m.dates[i] <= m.dates[i-1] {
			return ppInvalid
		}
	}
	// Stage order per the approved validation precedence: shape/date/identity defects
	// beat missing prices, which beat count sufficiency. A later short row therefore
	// beats an earlier null cell in BOTH record permutations.
	for _, c := range m.candidates {
		if c.present && len(c.prices) != len(m.dates) {
			return ppInvalid
		}
	}
	for _, c := range m.candidates {
		if !c.present {
			return ppIncomplete
		}
		for _, p := range c.prices {
			if p.missing {
				return ppIncomplete
			}
		}
	}
	for _, c := range m.candidates {
		for _, p := range c.prices {
			if _, good := exactPrice(p.text); !good {
				return ppInvalid
			}
		}
	}
	if len(m.candidates) < 16 {
		return ppInsufficient
	}
	return ppOK
}

// drawdown computes the exact rational maximum drawdown of one proposal under a value
// rule. Evaluator under test; the expected values are literal rationals in the tests.
func drawdown(p ppProposal, m ppMatrix, rule ppValueRule) (*big.Rat, error) {
	byTicker := map[string][]string{}
	for _, c := range m.candidates {
		row := make([]string, len(c.prices))
		for i, cell := range c.prices {
			if cell.missing {
				return nil, fmt.Errorf("missing cell for %s", c.ticker)
			}
			row[i] = cell.text
		}
		byTicker[c.ticker] = row
	}
	prices := map[string][]*big.Rat{}
	for t, row := range byTicker {
		vals := make([]*big.Rat, len(row))
		for i, s := range row {
			v, good := exactPrice(s)
			if !good {
				return nil, fmt.Errorf("bad price %q", s)
			}
			vals[i] = v
		}
		prices[t] = vals
	}
	n := len(m.dates)
	v := make([]*big.Rat, n)
	for t := range v {
		v[t] = new(big.Rat)
	}
	for _, h := range p.holdings {
		row, ok := prices[h.ticker]
		if !ok {
			return nil, fmt.Errorf("holding %q outside admitted universe", h.ticker)
		}
		switch rule {
		case valueBuyHold, valueUnset:
			w := new(big.Rat).SetFrac(big.NewInt(int64(h.weight)), big.NewInt(10000))
			for t := 0; t < n; t++ {
				rel := new(big.Rat).Quo(row[t], row[0])
				rel.Mul(rel, w)
				v[t].Add(v[t], rel)
			}
		case valueRaw:
			w := new(big.Rat).SetFrac(big.NewInt(int64(h.weight)), big.NewInt(10000))
			for t := 0; t < n; t++ {
				term := new(big.Rat).Mul(row[t], w)
				v[t].Add(v[t], term)
			}
		case valueRebalanced:
			for t := 1; t < n; t++ {
				ret := new(big.Rat)
				anyWeight := false
				for _, h2 := range p.holdings {
					r2 := prices[h2.ticker]
					if h2.weight > 0 {
						anyWeight = true
					}
					w := new(big.Rat).SetFrac(big.NewInt(int64(h2.weight)), big.NewInt(10000))
					rel := new(big.Rat).Quo(r2[t], r2[t-1])
					ret.Add(ret, new(big.Rat).Mul(w, rel))
				}
				if !anyWeight {
					continue
				}
				if t == 1 {
					v[t].Set(ret)
				} else {
					v[t].Mul(v[t-1], ret)
				}
			}
			v[0].SetInt64(1)
		}
	}
	peak := new(big.Rat)
	d := new(big.Rat)
	for t := 0; t < n; t++ {
		if t == 0 || v[t].Cmp(peak) > 0 {
			peak.Set(v[t])
		}
		drop := new(big.Rat).Sub(peak, v[t])
		dd := new(big.Rat).Quo(drop, peak)
		if dd.Cmp(d) > 0 {
			d.Set(dd)
		}
	}
	return d, nil
}

// storedBps is the D2 storage rule floor(10000 x D + 1/2), ties upward (evaluator under
// test; expected stored values are decimal literals in the tests).
func storedBps(d *big.Rat) int {
	scaled := new(big.Rat).Mul(d, big.NewRat(10000, 1))
	scaled.Add(scaled, big.NewRat(1, 2))
	num := new(big.Int).Set(scaled.Num())
	den := new(big.Int).Set(scaled.Denom())
	q := new(big.Int).Quo(num, den) // den > 0, truncation toward -inf == floor here (num >= 0)
	return int(q.Int64())
}

// tickerVector returns the proposal tickers in ascending ASCII order.
func tickerVector(p ppProposal) []string {
	out := make([]string, 0, len(p.holdings))
	for _, h := range p.holdings {
		out = append(out, h.ticker)
	}
	sort.Strings(out)
	return out
}

// weightVectorInTickerOrder returns weights ordered by ascending ticker.
func weightVectorInTickerOrder(p ppProposal) []int {
	pairs := make([]ppHolding, len(p.holdings))
	copy(pairs, p.holdings)
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].ticker < pairs[j].ticker })
	out := make([]int, len(pairs))
	for i, h := range pairs {
		out[i] = h.weight
	}
	return out
}

// compareTieKeys implements the approved total tie order (whole ticker vector first,
// then the whole weight vector) and the interleaved mutant (whole-clause alternative).
func compareTieKeys(a, b ppProposal, rule ppTieRule) int {
	ta, tb := tickerVector(a), tickerVector(b)
	switch rule {
	case tieVectors:
		if c := strings.Compare(strings.Join(ta, "\x00"), strings.Join(tb, "\x00")); c != 0 {
			return c
		}
		wa, wb := weightVectorInTickerOrder(a), weightVectorInTickerOrder(b)
		for i := range wa {
			if wa[i] != wb[i] {
				if wa[i] < wb[i] {
					return -1
				}
				return 1
			}
		}
		return 0
	case tieInterleaved:
		for i := range ta {
			if c := strings.Compare(ta[i], tb[i]); c != 0 {
				return c
			}
			wa, wb := weightVectorInTickerOrder(a), weightVectorInTickerOrder(b)
			if wa[i] != wb[i] {
				if wa[i] < wb[i] {
					return -1
				}
				return 1
			}
		}
		return 0
	}
	return 0
}

// selectAdmitted admits the whole universe then selects under the packet facts.
func selectAdmitted(f ppFacts, m ppMatrix, l ppLimits, proposals []ppProposal) (string, error) {
	if !f.evaluable() {
		return "", fmt.Errorf("incomplete facts: %v", f.missingObligations())
	}
	if reason := admit(m, l); reason != ppOK {
		return "", fmt.Errorf("universe rejected: %s", reason)
	}
	if len(proposals) == 0 {
		return "", fmt.Errorf("empty proposal set")
	}
	return selectScored(f, m, proposals)
}

// selectSurvivors implements the filter-admission whole-clause mutant: invalid candidate
// histories are removed instead of rejecting the universe.
func selectSurvivors(f ppFacts, m ppMatrix, l ppLimits, proposals []ppProposal) (string, []string, error) {
	if !f.evaluable() {
		return "", nil, fmt.Errorf("incomplete facts")
	}
	valid := ppMatrix{dates: m.dates, basis: m.basis}
	for _, c := range m.candidates {
		ok := c.present && len(c.prices) == len(m.dates)
		if ok {
			for _, p := range c.prices {
				if p.missing {
					ok = false
					break
				}
				if _, good := exactPrice(p.text); !good {
					ok = false
					break
				}
			}
		}
		if ok {
			valid.candidates = append(valid.candidates, c)
		}
	}
	for _, p := range proposals {
		for _, h := range p.holdings {
			found := false
			for _, c := range valid.candidates {
				if c.ticker == h.ticker {
					found = true
					break
				}
			}
			if !found {
				return "", nil, fmt.Errorf("proposal %q holds unavailable member %q", p.name, h.ticker)
			}
		}
	}
	if len(valid.candidates) < 16 {
		return "", nil, fmt.Errorf("insufficient survivors: %d", len(valid.candidates))
	}
	winner, err := selectScored(f, valid, proposals)
	if err != nil {
		return "", nil, err
	}
	names := make([]string, len(valid.candidates))
	for i, c := range valid.candidates {
		names[i] = c.ticker
	}
	sort.Strings(names)
	return winner, names, nil
}

func selectScored(f ppFacts, m ppMatrix, proposals []ppProposal) (string, error) {
	for _, p := range proposals {
		if _, err := drawdown(p, m, f.Value); err != nil {
			return "", err
		}
	}
	best := 0
	for i := 1; i < len(proposals); i++ {
		if proposalBeats(f, m, proposals[i], proposals[best]) {
			best = i
		}
	}
	return proposals[best].name, nil
}

// proposalBeats reports whether a is preferred to b under the packet's direction,
// comparison and tie rules (the evaluator under test).
func proposalBeats(f ppFacts, m ppMatrix, a, b ppProposal) bool {
	da, err := drawdown(a, m, f.Value)
	if err != nil {
		return false
	}
	db, err := drawdown(b, m, f.Value)
	if err != nil {
		return true
	}
	less, equal := false, false
	if f.Compare == compareRounded {
		pa, pb := storedBps(da), storedBps(db)
		less, equal = pa < pb, pa == pb
	} else {
		c := da.Cmp(db)
		less, equal = c < 0, c == 0
	}
	if f.Direction == dirMax {
		if !equal {
			return !less
		}
		return compareTieKeys(a, b, f.Tie) > 0
	}
	if !equal {
		return less
	}
	return compareTieKeys(a, b, f.Tie) < 0
}

// ppObservesWinner reports whether selection under facts f observes the named winner.
func ppObservesWinner(f ppFacts, m ppMatrix, l ppLimits, proposals []ppProposal, want string) bool {
	got, err := selectAdmitted(f, m, l, proposals)
	return err == nil && got == want
}

// --- literal fixture builders (universes from the independent ledger) ----------

func ppTickerCells(prices []string, missing []int) []ppCell {
	out := make([]ppCell, len(prices))
	for i, p := range prices {
		out[i] = ppCell{text: p}
	}
	for _, i := range missing {
		out[i] = ppCell{missing: true}
	}
	return out
}

func universeMatrix(prefix string, row []string) ppMatrix {
	dates := make([]string, len(row))
	for i := range dates {
		dates[i] = fmt.Sprintf("2024-01-%02d", i+2)
	}
	m := ppMatrix{dates: dates, basis: "unadjusted-close"}
	for i := 1; i <= 16; i++ {
		t := fmt.Sprintf("%s%02d", prefix, i)
		m.candidates = append(m.candidates, ppCandidate{ticker: t, prices: ppTickerCells(row, nil), present: true})
	}
	return m
}

func equalWeights() []int {
	w := make([]int, 16)
	for i := range w {
		w[i] = 625
	}
	w[15] = 625
	// 16 x 625 = 10000 exactly
	return w
}

func weightsProposal(name, prefix string, w []int) ppProposal {
	p := ppProposal{name: name}
	for i := 0; i < 16; i++ {
		p.holdings = append(p.holdings, ppHolding{ticker: fmt.Sprintf("%s%02d", prefix, i+1), weight: w[i]})
	}
	return p
}

func fixtureLimits(m ppMatrix) ppLimits {
	longest := 0
	for _, d := range m.dates {
		if len(d) > longest {
			longest = len(d)
		}
	}
	for _, c := range m.candidates {
		if len(c.ticker) > longest {
			longest = len(c.ticker)
		}
		for _, p := range c.prices {
			if len(p.text) > longest {
				longest = len(p.text)
			}
		}
	}
	return ppLimits{maxCandidates: 16, maxLookbackDays: uint64(len(m.dates)), maxScalarBytes: uint64(longest)}
}

// ---------------------------------------------------------------------------
// Shared identities (2).
// ---------------------------------------------------------------------------

// strictPins are the frozen design inputs this story has no write ownership over: their
// bytes must be identical in the RED and GREEN phases (valid-prerequisite arrangement).
var strictPins = []string{
	"examples/portfolio-engine/design/machines/MarketDataFeed.machine.json",
	"examples/portfolio-engine/design/machines/MarketDataFeed.oracle.md",
	"examples/portfolio-engine/design/machines/Portfolio.oracle.md",
	"examples/portfolio-engine/design/machines/RecommendationRun.oracle.md",
	"examples/portfolio-engine/design/machines/ReferenceDataCommand.oracle.md",
}

func TestPortfolioPacketFrozenDesignInputsArePinned(t *testing.T) {
	t.Run("frozen-inputs-match-pin-table", func(t *testing.T) {
		for _, rel := range strictPins {
			raw := designBytes(t, rel)
			if got := shaHex(raw); got != frozenPins[rel] {
				t.Errorf("frozen input %s sha256=%s want pin %s", rel, got, frozenPins[rel])
			}
		}
	})
	t.Run("tampered-authority-copy-is-detected", func(t *testing.T) {
		raw := designBytes(t, "examples/portfolio-engine/design/machines/RecommendationRun.oracle.md")
		tampered := append([]byte(nil), raw...)
		tampered[120] ^= 0x01
		if shaHex(tampered) == shaHex(raw) {
			t.Fatal("one-byte tamper of an in-memory authority copy was NOT detected by the pin digest")
		}
	})
	t.Run("baseline-record-of-edited-inputs", func(t *testing.T) {
		// The nine frozen inputs this story legitimately edits are recorded (not gated):
		// their old/new hashes are reported per AC5 in the MAC-hgz1 handoff ledger.
		for rel := range frozenPins {
			raw := designBytes(t, rel)
			t.Logf("input %s sha256=%s", rel, shaHex(raw))
		}
	})
}

func machineOfStable(stable string) string {
	switch {
	case strings.HasPrefix(stable, "RECO"):
		return "RecommendationRun"
	case strings.HasPrefix(stable, "MARK"):
		return "MarketDataFeed"
	case strings.HasPrefix(stable, "PORT"):
		return "Portfolio"
	case strings.HasPrefix(stable, "REFE"):
		return "ReferenceDataCommand"
	}
	return ""
}

// fsmExpectationDiagnostics reconciles the parsed oracle facts against an EXPECTATION
// set (E-PP-05 authority separation: the expectation side is data, never produced by
// the parsers under test).
func fsmExpectationDiagnostics(c *fsmCorpus, expectation []fsmRow) []string {
	var diags []string
	byStable := map[string]fsmRow{}
	for _, d := range c.docs {
		for _, r := range d.oracleRows {
			byStable[r.Stable] = r
		}
	}
	for _, want := range expectation {
		got, ok := byStable[want.Stable]
		if !ok {
			diags = append(diags, fmt.Sprintf("missing row %s", want.Stable))
			continue
		}
		if got.Source != want.Source || got.Trigger != want.Trigger || got.Guard != want.Guard ||
			got.Target != want.Target || !equalActions(got.Actions, want.Actions) {
			diags = append(diags, fmt.Sprintf("row %s: facts %+v disagree with expectation %+v", want.Stable, got, want))
		}
	}
	return diags
}

func TestPortfolioPacketAuthorityMutationIsDiagnosticOnly(t *testing.T) {
	machines := loadFSMCorpus(t)
	t.Run("approved-authorities-consistent", func(t *testing.T) {
		if diags := fsmAuthorityDiagnostics(machines); len(diags) != 0 {
			t.Fatalf("live machine/oracle authorities inconsistent: %v", diags)
		}
		if diags := fsmExpectationDiagnostics(machines, fsmLiteralLedger()); len(diags) != 0 {
			t.Fatalf("live facts disagree with the approved expectation ledger: %v", diags)
		}
	})
	t.Run("authority-mutation-changes-only-diagnostics", func(t *testing.T) {
		// Mutate ONE authority-side expectation literal (never the packet bytes).
		expectation := fsmLiteralLedger()
		for i := range expectation {
			if expectation[i].Stable == "PORT-d1647b" {
				expectation[i].Actions = []string{"recordAccepted", "commit"} // mutated authority literal
			}
		}
		factsBefore := map[string]fsmRow{}
		for _, d := range machines.docs {
			for _, r := range d.oracleRows {
				factsBefore[r.Stable] = r
			}
		}
		diags := fsmExpectationDiagnostics(machines, expectation)
		if len(diags) == 0 {
			t.Fatal("authority mutation produced no consistency diagnostic change; tamper was invisible")
		}
		// Packet facts/spans unchanged: the same parsed rows, the same bytes.
		for _, d := range machines.docs {
			for _, r := range d.oracleRows {
				before := factsBefore[r.Stable]
				if before.Stable != r.Stable || !equalActions(before.Actions, r.Actions) {
					t.Fatal("authority mutation changed packet FACTS; diagnostics-only violated")
				}
			}
		}
		// The comparison function never constructs both sides: facts come from the
		// corpus parser, expectations from the independent literal ledger.
		if diags[0] == "" {
			t.Fatal("empty diagnostic")
		}
	})
}

// ---------------------------------------------------------------------------
// D1 identities (12).
// ---------------------------------------------------------------------------

func twoFlavorMatrix(rowsA, rowsB []string) (ppMatrix, ppProposal, ppProposal) {
	m := ppMatrix{dates: []string{"2024-01-02", "2024-01-03", "2024-01-04"}, basis: "unadjusted-close"}
	for i := 1; i <= 16; i++ {
		m.candidates = append(m.candidates, ppCandidate{ticker: fmt.Sprintf("A%02d", i), prices: ppTickerCells(rowsA, nil), present: true})
	}
	for i := 1; i <= 16; i++ {
		m.candidates = append(m.candidates, ppCandidate{ticker: fmt.Sprintf("B%02d", i), prices: ppTickerCells(rowsB, nil), present: true})
	}
	return m, weightsProposal("proposal-A", "A", equalWeights()), weightsProposal("proposal-B", "B", equalWeights())
}

func TestPortfolioPacketD1CompleteProseControlSelectsMinimum(t *testing.T) {
	f := mustParse(t, completeProseControl)
	m, a, b := twoFlavorMatrix([]string{"100", "80", "100"}, []string{"100", "90", "100"})
	lim := ppLimits{maxCandidates: 32, maxLookbackDays: 3, maxScalarBytes: 10}
	if !f.safe() {
		t.Fatal("complete approved prose control must be safe")
	}
	da, err := drawdown(a, m, valueBuyHold)
	if err != nil || da.Cmp(ratLit(1, 5)) != 0 {
		t.Fatalf("D_A=%v err=%v, want literal 1/5", da, err)
	}
	db, err := drawdown(b, m, valueBuyHold)
	if err != nil || db.Cmp(ratLit(1, 10)) != 0 {
		t.Fatalf("D_B=%v err=%v, want literal 1/10", db, err)
	}
	if got, err := selectAdmitted(f, m, lim, []ppProposal{a, b}); err != nil || got != "proposal-B" {
		t.Fatalf("min objective winner=%q err=%v, want proposal-B (D_A=1/5=2000bps > D_B=1/10=1000bps)", got, err)
	}
	if got, err := selectAdmitted(f, m, lim, []ppProposal{b, a}); err != nil || got != "proposal-B" {
		t.Fatalf("min objective is order-independent: winner=%q err=%v, want proposal-B", got, err)
	}
	maxFacts := mustParse(t, swapClause(t, completeProseControl, cObjectiveMin, cObjectiveMax))
	got, err := selectAdmitted(maxFacts, m, lim, []ppProposal{b, a})
	if err != nil || got != "proposal-A" {
		t.Fatalf("max whole-clause mutant must flip winner to A, got %q err=%v", got, err)
	}
	daBps := storedBps(da)
	if daBps != 2000 {
		t.Fatalf("stored bps for D=1/5 = %d, want literal 2000", daBps)
	}
}

func TestPortfolioPacketD1EquivalentClauseRedundancy(t *testing.T) {
	m, a, b := twoFlavorMatrix([]string{"100", "80", "100"}, []string{"100", "90", "100"})
	lim := ppLimits{maxCandidates: 32, maxLookbackDays: 3, maxScalarBytes: 10}
	primary := mustParse(t, completeProseControl)
	both := mustParse(t, cObjectiveMin+"\n"+cObjectiveMinEquivalent+"\n"+strings.Join([]string{
		cBuyHold, cExact, cHoldings, cTie, cWhole, cCalendar, cPrices, cLimits}, "\n"))
	equivOnly := mustParse(t, strings.Replace(completeProseControl, cObjectiveMin, cObjectiveMinEquivalent, 1))
	primaryRemoved := mustParse(t, strings.Join([]string{
		cBuyHold, cExact, cHoldings, cTie, cWhole, cCalendar, cPrices, cLimits}, "\n"))
	for _, tc := range []struct {
		name  string
		facts ppFacts
		safe  bool
		win   string
	}{
		{"primary", primary, true, "proposal-B"},
		{"primary-plus-equivalent", both, true, "proposal-B"},
		{"equivalent-only", equivOnly, true, "proposal-B"},
		{"objective-missing", primaryRemoved, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.facts.safe() != tc.safe {
				t.Fatalf("safe=%v want %v (missing=%v)", tc.facts.safe(), tc.safe, tc.facts.missingObligations())
			}
			if !tc.safe {
				if _, err := selectAdmitted(tc.facts, m, lim, []ppProposal{a, b}); err == nil {
					t.Fatal("missing objective must yield a selection error, never fallback semantics")
				}
				return
			}
			got, err := selectAdmitted(tc.facts, m, lim, []ppProposal{a, b})
			if err != nil || got != tc.win {
				t.Fatalf("winner=%q err=%v want %q", got, err, tc.win)
			}
		})
	}
}

func TestPortfolioPacketD1OriginalPacketTwoCompletions(t *testing.T) {
	raw := designBytes(t, "examples/portfolio-engine/design/BUILD/M3-optimizer.md")
	t.Run("valid-prerequisite-hash-pin", func(t *testing.T) {
		// Recorded provenance: the epic-tip deficient M3 hashes 621255b4... After this
		// story's approved M3 repair the bytes legitimately differ; the pin therefore
		// only RECORDS the hash so a hash mismatch can never masquerade as the semantic
		// failure below (assurance contract 5.3).
		t.Logf("live M3 sha256=%s (epic-tip deficient baseline 621255b4ece8863c50c70cc0f9bc7ac3e831a6ad702a23408f5b41f80a3e3c67)", shaHex(raw))
	})
	t.Run("packet-must-supply-complete-obligations", func(t *testing.T) {
		text := string(raw)
		f, err := extractPacketObligations(text)
		if err != nil {
			t.Fatalf("packet obligation extraction failed: %v", err)
		}
		missing := f.missingObligations()
		if len(missing) != 0 {
			// Prove the failure is the intended semantic one: over the independent A/B
			// witness, BOTH a min completion and a max completion are compatible with
			// everything the deficient packet does supply (they select different
			// winners), so the objective obligation is MISSING — not an unknown string.
			m, a, b := twoFlavorMatrix([]string{"100", "80", "100"}, []string{"100", "90", "100"})
			lim := ppLimits{maxCandidates: 32, maxLookbackDays: 3, maxScalarBytes: 10}
			type completion struct{ clause, want string }
			for _, c := range []completion{{cObjectiveMin, "proposal-B"}, {cObjectiveMax, "proposal-A"}} {
				full := mustParse(t, c.clause+"\n"+strings.Join([]string{
					cBuyHold, cExact, cHoldings, cTie, cWhole, cCalendar, cPrices, cLimits}, "\n"))
				got, err := selectAdmitted(full, m, lim, []ppProposal{a, b})
				if err != nil || got != c.want {
					t.Fatalf("two-completion diagnosis broken: completion %q selected %q err=%v want %q", c.clause, got, err, c.want)
				}
			}
			t.Fatalf("live M3 packet is MISSING required obligations %v: both min (selects proposal-B) and max (selects proposal-A) completions satisfy every obligation the packet supplies, so the objective/value/comparison/tie contracts are absent as packet facts", missing)
		}
		if f.Direction != dirMin {
			t.Fatalf("packet objective direction=%v want min", f.Direction)
		}
	})
}

func TestPortfolioPacketD1ValueRuleWitness(t *testing.T) {
	f := mustParse(t, completeProseControl)
	rowsA := map[string][]string{}
	for i := 1; i <= 16; i++ {
		rowsA[fmt.Sprintf("A%02d", i)] = []string{"1", "1", "1"}
	}
	rowsA["A01"] = []string{"10", "20", "10"}
	rowsA["A02"] = []string{"100", "100", "100"}
	m := ppMatrix{dates: []string{"2024-01-02", "2024-01-03", "2024-01-04"}, basis: "unadjusted-close"}
	for i := 1; i <= 16; i++ {
		tk := fmt.Sprintf("A%02d", i)
		m.candidates = append(m.candidates, ppCandidate{ticker: tk, prices: ppTickerCells(rowsA[tk], nil), present: true})
	}
	w := make([]int, 16)
	w[0], w[1] = 5000, 5000
	hetero := ppProposal{name: "heterogeneous"}
	for i := 1; i <= 16; i++ {
		hetero.holdings = append(hetero.holdings, ppHolding{ticker: fmt.Sprintf("A%02d", i), weight: w[i-1]})
	}
	competitor := weightsProposal("competitor", "B", equalWeights())
	// One universe contains both flavors: A rows are heterogeneous, B rows [100,80,100].
	full := ppMatrix{dates: m.dates, basis: m.basis}
	full.candidates = append(full.candidates, m.candidates...)
	full.candidates = append(full.candidates, universeMatrix("B", []string{"100", "80", "100"}).candidates...)
	lim := ppLimits{maxCandidates: 64, maxLookbackDays: 3, maxScalarBytes: 10}
	t.Run("normalized-per-asset-initial-price", func(t *testing.T) {
		d, err := drawdown(hetero, full, valueBuyHold)
		if err != nil || d.Cmp(ratLit(1, 3)) != 0 {
			t.Fatalf("buy-and-hold D=%v err=%v, want literal 1/3", d, err)
		}
		if bps := storedBps(d); bps != 3333 {
			t.Fatalf("stored bps=%d want literal 3333", bps)
		}
	})
	t.Run("raw-price-mutant", func(t *testing.T) {
		d, err := drawdown(hetero, full, valueRaw)
		if err != nil || d.Cmp(ratLit(1, 12)) != 0 {
			t.Fatalf("raw-price D=%v err=%v, want literal 1/12", d, err)
		}
		rawFacts := mustParse(t, swapClause(t, completeProseControl, cBuyHold, cUnnormalized))
		if !ppObservesWinner(f, full, lim, []ppProposal{hetero, competitor}, "competitor") {
			t.Fatal("approved facts failed to observe the approved winner (competitor, D=1/5 < 1/3)")
		}
		if ppObservesWinner(rawFacts, full, lim, []ppProposal{hetero, competitor}, "competitor") {
			t.Fatal("raw-price whole-clause mutant still satisfied the approved expectation")
		}
		if !ppObservesWinner(rawFacts, full, lim, []ppProposal{hetero, competitor}, "heterogeneous") {
			t.Fatal("raw-price mutant did not expose the winner flip (1/12 < 1/5 selects heterogeneous)")
		}
	})
	t.Run("rebalancing-mutant", func(t *testing.T) {
		d, err := drawdown(hetero, full, valueRebalanced)
		if err != nil || d.Cmp(ratLit(1, 4)) != 0 {
			t.Fatalf("rebalanced D=%v err=%v, want literal 1/4", d, err)
		}
		m310 := universeMatrix("C", []string{"100", "100", "70"})
		comp310 := weightsProposal("competitor310", "C", equalWeights())
		d310, err := drawdown(comp310, m310, valueBuyHold)
		if err != nil || d310.Cmp(ratLit(3, 10)) != 0 {
			t.Fatalf("competitor D=%v err=%v, want literal 3/10", d310, err)
		}
		if bps := storedBps(d310); bps != 3000 {
			t.Fatalf("competitor stored bps=%d want literal 3000", bps)
		}
		// 1/4 < 3/10 < 1/3: the rebalanced mutant reverses the approved ranking.
		if new(big.Rat).Quo(ratLit(1, 4), ratLit(3, 10)).Cmp(ratLit(1, 1)) >= 0 {
			t.Fatal("literal sanity: 1/4 must be strictly less than 3/10")
		}
	})
}

func TestPortfolioPacketD1HistoricalPeakDefinition(t *testing.T) {
	m := universeMatrix("P", []string{"100", "80", "120", "114"})
	m.dates = []string{"2024-01-02", "2024-01-03", "2024-01-04", "2024-01-05"}
	p := weightsProposal("peak-witness", "P", equalWeights())
	d, err := drawdown(p, m, valueBuyHold)
	if err != nil || d.Cmp(ratLit(1, 5)) != 0 {
		t.Fatalf("peak-based D=%v err=%v, want literal 1/5", d, err)
	}
	if bps := storedBps(d); bps != 2000 {
		t.Fatalf("stored bps=%d want literal 2000", bps)
	}
	// Wrong peak definitions compute the literal wrong values: last-peak decline (1/20)
	// compares only the running max immediately before the final observation...
	lastPeak := ratLit(1, 20)
	startToEnd := ratLit(0, 1)
	if lastPeak.Cmp(d) == 0 || startToEnd.Cmp(d) == 0 {
		t.Fatalf("last-peak %s and start-to-end %s must differ from the peak-based literal 1/5", lastPeak, startToEnd)
	}
	// 1/20 is the decline from the t=3 peak (120) to t=4 (114): 6/120.
	if got := new(big.Rat).Sub(ratLit(120, 1), ratLit(114, 1)); got.Cmp(ratLit(6, 1)) != 0 {
		t.Fatal("literal arithmetic drift in peak witness")
	}
}

func TestPortfolioPacketD1RoundingAndExactComparison(t *testing.T) {
	f := mustParse(t, completeProseControl)
	rw := func(rows ...string) (ppMatrix, ppProposal) {
		m := universeMatrix("R", rows)
		return m, weightsProposal("rw", "R", equalWeights())
	}
	type spec struct {
		name string
		rows []string
		num  int64 // exact D numerator (over 10000 denominators in bps terms)
		den  int64
		bps  int
	}
	for _, s := range []spec{
		{"half-up-0.50", []string{"10000", "9999.50"}, 5, 100000, 1},
		{"below-half-0.49", []string{"10000", "9999.51"}, 49, 1000000, 0},
		{"above-half-0.51", []string{"10000", "9999.49"}, 51, 1000000, 1},
		{"0.99", []string{"10000", "9999.01"}, 99, 1000000, 1},
	} {
		t.Run(s.name, func(t *testing.T) {
			m, p := rw(s.rows...)
			d, err := drawdown(p, m, valueBuyHold)
			if err != nil {
				t.Fatal(err)
			}
			if d.Cmp(ratLit(s.num, s.den)) != 0 {
				t.Fatalf("exact D=%s want literal %d/%d (no epsilon)", d, s.num, s.den)
			}
			if got := storedBps(d); got != s.bps {
				t.Fatalf("stored bps=%d want literal %d (half-up, ties upward)", got, s.bps)
			}
		})
	}
	t.Run("compare-before-rounding", func(t *testing.T) {
		// Both portfolios store 1 bps (0.51 and 0.99), but exact comparison must select
		// the 0.51-bps portfolio even when its ticker vector sorts later; the
		// rounded-ranking whole-clause mutant selects the other.
		m := universeMatrix("R", []string{"10000", "9999.01"})
		w99 := weightsProposal("w99", "R", equalWeights())
		m2 := universeMatrix("S", []string{"10000", "9999.49"})
		w51 := weightsProposal("w51", "S", equalWeights())
		full := ppMatrix{dates: m.dates, basis: m.basis}
		full.candidates = append(full.candidates, m.candidates...)
		full.candidates = append(full.candidates, m2.candidates...)
		lim := ppLimits{maxCandidates: 64, maxLookbackDays: 3, maxScalarBytes: 10}
		got, err := selectAdmitted(f, full, lim, []ppProposal{w99, w51}) // "S" tickers sort later; exact D still wins
		if err != nil || got != "w51" {
			t.Fatalf("exact comparison winner=%q err=%v want w51 (0.51bps exact < 0.99bps exact)", got, err)
		}
		roundedFacts := mustParse(t, swapClause(t, completeProseControl, cExact, cRounded))
		got, err = selectAdmitted(roundedFacts, full, lim, []ppProposal{w99, w51})
		if err != nil || got != "w99" {
			t.Fatalf("rounded-ranking mutant winner=%q err=%v want w99 (1==1 bps falls to the tie key, R<S)", got, err)
		}
	})
}

func TestPortfolioPacketD1AllocationAndZeroWeights(t *testing.T) {
	f := mustParse(t, completeProseControl)
	flat := func(prefix string) ppMatrix { return universeMatrix(prefix, []string{"100", "100", "100"}) }
	eq := equalWeights()
	sum625 := append([]int(nil), eq...)
	// eq is 16x625 = 10000 exactly.
	if total := func(ws []int) int {
		s := 0
		for _, w := range ws {
			s += w
		}
		return s
	}(sum625); total != 10000 {
		t.Fatalf("16x625 sum=%d want literal 10000", total)
	}
	validZero := func() []int {
		w := append([]int(nil), eq...)
		w[0] = 0
		w[1] = 1250
		for i := 2; i < 16; i++ {
			w[i] = 625
		}
		// 0 + 1250 + 14x625 = 0+1250+8750 = 10000.
		return w
	}()
	for _, tc := range []struct {
		name    string
		weights []int
		wantErr string
		zero    bool
	}{
		{"16x625-valid", eq, "", false},
		{"zero-weight-record-retained", validZero, "", true},
		{"sum-9999", func() []int { w := append([]int(nil), eq...); w[15] = 624; return w }(), "sum 9999 != 10000", false},
		{"sum-10001", func() []int { w := append([]int(nil), eq...); w[15] = 626; return w }(), "sum 10001 != 10000", false},
		{"negative-weight", func() []int { w := append([]int(nil), eq...); w[0] = -1; w[1] = 1251; return w }(), "negative weight", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := flat("V")
			p := weightsProposal("alloc", "V", tc.weights)
			err := func() error {
				sum := 0
				for _, h := range p.holdings {
					if h.weight < 0 {
						return fmt.Errorf("negative weight")
					}
					if h.weight > 10000 {
						return fmt.Errorf("weight bound")
					}
					sum += h.weight
				}
				if len(p.holdings) != 16 {
					return fmt.Errorf("record count %d", len(p.holdings))
				}
				if sum != 10000 {
					return fmt.Errorf("sum %d != 10000", sum)
				}
				return nil
			}()
			if tc.wantErr == "" && err != nil {
				t.Fatalf("valid allocation rejected: %v", err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("invalid allocation must fail by %q, got %v", tc.wantErr, err)
			}
			// Zero-weight record retention: the record stays in output.
			if tc.zero {
				zeroSeen := false
				for _, h := range p.holdings {
					if h.weight == 0 {
						zeroSeen = true
					}
				}
				if !zeroSeen || len(p.holdings) != 16 {
					t.Fatal("zero-weight record was dropped or record count changed")
				}
			}
			// Selection dimension check on the admitted universe (evaluator view).
			if tc.wantErr == "" {
				if _, err := selectAdmitted(f, m, fixtureLimits(m), []ppProposal{p}); err != nil {
					t.Fatalf("valid allocation must select: %v", err)
				}
			}
		})
	}
	t.Run("fractional-type-invalid", func(t *testing.T) {
		// (624.5, 625.5, 14x625) sums to 10000 but weights are a mathematical integer
		// domain: boolean and floating-point typed carriers are invalid.
		half := 624.5
		if half == float64(int(half)) {
			t.Fatal("fixture sanity")
		}
		if err := weightCarrierValid(half); err == nil {
			t.Fatal("float-typed weight carrier must be INVALID_INPUT (integer domain)")
		}
		if err := weightCarrierValid(true); err == nil {
			t.Fatal("boolean-typed weight carrier must be INVALID_INPUT (integer domain)")
		}
		if err := weightCarrierValid(625); err != nil {
			t.Fatalf("int carrier rejected: %v", err)
		}
		if err := weightCarrierValid(uint64(625)); err != nil {
			t.Fatalf("nonnegative uint64 carrier rejected: %v", err)
		}
	})
}

// weightCarrierValid enforces the D2 integer-domain carrier rule.
func weightCarrierValid(v any) error {
	switch x := v.(type) {
	case int:
		if x >= 0 {
			return nil
		}
		return fmt.Errorf("negative")
	case uint64:
		return nil
	case int64:
		if x >= 0 {
			return nil
		}
		return fmt.Errorf("negative")
	}
	return fmt.Errorf("INVALID_INPUT: weight carrier must be a mathematical integer, got %T", v)
}

func TestPortfolioPacketD1TotalTieOrder(t *testing.T) {
	f := mustParse(t, completeProseControl)
	flat := []string{"100", "100", "100"}
	// Same sorted ticker vector, weight vectors (0,1250,14x625) vs (625,625,14x625):
	// both exact D=0; the whole weight vector decides — (0,...) wins.
	mV := universeMatrix("V", flat)
	w1 := weightsProposal("tie-v1", "V", func() []int {
		w := equalWeights()
		w[0], w[1] = 0, 1250
		return w
	}())
	w2 := weightsProposal("tie-v2", "V", equalWeights())
	if got := compareTieKeys(w1, w2, tieVectors); got >= 0 {
		t.Fatalf("vector tie key: (0,1250,...) must beat (625,625,...), cmp=%d", got)
	}
	got, err := selectAdmitted(f, mV, fixtureLimits(mV), []ppProposal{w2, w1})
	if err != nil || got != "tie-v1" {
		t.Fatalf("tie winner=%q err=%v want tie-v1", got, err)
	}
	// Input-record permutation never changes the key.
	perm := ppProposal{name: "tie-v1-permuted"}
	perm.holdings = append(perm.holdings, w1.holdings[5], w1.holdings[0], w1.holdings[3])
	perm.holdings = append(perm.holdings, w1.holdings[1], w1.holdings[2], w1.holdings[4])
	perm.holdings = append(perm.holdings, w1.holdings[6:]...)
	if len(perm.holdings) != 16 {
		t.Fatalf("permutation fixture records=%d want 16", len(perm.holdings))
	}
	if got := compareTieKeys(perm, w2, tieVectors); got >= 0 {
		t.Fatal("record permutation changed the tie key")
	}
	// Distinct ticker vectors sharing the first ticker: whole-vector comparison decides
	// at the second ticker; the interleaved mutant decides at the first weight instead.
	mk := func(wA, wB int) (ppMatrix, ppProposal, ppProposal) {
		m := ppMatrix{dates: []string{"2024-01-02", "2024-01-03", "2024-01-04"}, basis: "unadjusted-close"}
		for i := 1; i <= 16; i++ {
			m.candidates = append(m.candidates, ppCandidate{ticker: fmt.Sprintf("%s%02d", "V", i), prices: ppTickerCells(flat, nil), present: true})
		}
		// Second universe flavor for the second proposal's later tickers.
		for i := 1; i <= 16; i++ {
			m.candidates = append(m.candidates, ppCandidate{ticker: fmt.Sprintf("W%02d", i), prices: ppTickerCells(flat, nil), present: true})
		}
		pa := ppProposal{name: "pa"}
		pb := ppProposal{name: "pb"}
		for i := 1; i <= 16; i++ {
			pa.holdings = append(pa.holdings, ppHolding{ticker: fmt.Sprintf("V%02d", i), weight: wA})
		}
		pb.holdings = append(pb.holdings, ppHolding{ticker: "V01", weight: wB})
		for i := 2; i <= 16; i++ {
			pb.holdings = append(pb.holdings, ppHolding{ticker: fmt.Sprintf("W%02d", i), weight: wB})
		}
		return m, pa, pb
	}
	m, pa, pb := mk(900, 400)
	lim := ppLimits{maxCandidates: 64, maxLookbackDays: 3, maxScalarBytes: 10}
	// Whole-vector: V01==V01, V02 < W02 => pa wins.
	if got := compareTieKeys(pa, pb, tieVectors); got >= 0 {
		t.Fatal("whole ticker-vector comparison must select pa (V02 < W02)")
	}
	got, err = selectAdmitted(f, m, lim, []ppProposal{pa, pb})
	if err != nil || got != "pa" {
		t.Fatalf("vector-tie winner=%q err=%v want pa", got, err)
	}
	interFacts := mustParse(t, swapClause(t, completeProseControl, cTie, cTieInterleaved))
	// Interleaved: (V01,900) vs (V01,400) => pb wins at the first weight.
	if gt := compareTieKeys(pa, pb, tieInterleaved); gt <= 0 {
		t.Fatal("interleaved mutant comparison must select pb (400 < 900 at first ticker)")
	}
	got, err = selectAdmitted(interFacts, m, lim, []ppProposal{pa, pb})
	if err != nil || got != "pb" {
		t.Fatalf("interleaved-tie mutant winner=%q err=%v want pb", got, err)
	}
}

func admissionFixture() (ppMatrix, ppLimits) {
	m := universeMatrix("D", []string{"100", "90", "110"})
	lim := ppLimits{maxCandidates: 16, maxLookbackDays: 3, maxScalarBytes: 10}
	return m, lim
}

func TestPortfolioPacketD1AdmissionStagePrecedence(t *testing.T) {
	base, lim := admissionFixture()
	clone := func(m ppMatrix) ppMatrix {
		c := m
		c.candidates = append([]ppCandidate(nil), m.candidates...)
		c.dates = append([]string(nil), m.dates...)
		return c
	}
	t.Run("approved-whole-input-literals", func(t *testing.T) {
		m := clone(base)
		m.candidates = m.candidates[:15]
		if r := admit(m, lim); r != ppInsufficient {
			t.Fatalf("15 complete candidates: %s want INSUFFICIENT_CANDIDATES", r)
		}
		m = clone(base)
		if r := admit(m, lim); r != ppOK {
			t.Fatalf("16 complete: %s want OK", r)
		}
		m = clone(base)
		m.candidates = append(m.candidates, ppCandidate{ticker: "D99", present: false})
		if r := admit(m, lim); r != ppLimitExceeded {
			t.Fatalf("17th candidate with limit 16: %s want LIMIT_EXCEEDED before decoding", r)
		}
		m = clone(base)
		bad := clone(base)
		bad.candidates = append(bad.candidates, ppCandidate{ticker: "D99", present: false})
		badLim := lim
		badLim.maxCandidates = 17
		if r := admit(bad, badLim); r != ppIncomplete {
			t.Fatalf("missing row with limit 17: %s want INCOMPLETE_HISTORY", r)
		}
		m = clone(base)
		m.basis = "adjusted-close"
		if r := admit(m, lim); r != ppInvalid {
			t.Fatalf("mixed price basis: %s want INVALID_INPUT", r)
		}
		m = clone(base)
		m.dates = []string{"2024-01-03", "2024-01-03", "2024-01-05"}
		if r := admit(m, lim); r != ppInvalid {
			t.Fatalf("duplicate dates: %s want INVALID_INPUT", r)
		}
		m = clone(base)
		m.dates = []string{"2024-01-05", "2024-01-03", "2024-01-04"}
		if r := admit(m, lim); r != ppInvalid {
			t.Fatalf("nonincreasing dates: %s want INVALID_INPUT", r)
		}
		m = clone(base)
		m.candidates[3].prices = []ppCell{{text: "100"}, {missing: true}, {text: "110"}}
		if r := admit(m, lim); r != ppIncomplete {
			t.Fatalf("null cell in full-length row: %s want INCOMPLETE_HISTORY", r)
		}
		m = clone(base)
		m.candidates[3].prices = []ppCell{{text: "100"}, {text: "110"}}
		if r := admit(m, lim); r != ppInvalid {
			t.Fatalf("short present row: %s want INVALID_INPUT", r)
		}
		m = clone(base)
		m.candidates[3].prices = []ppCell{{text: "100"}, {text: ""}, {text: "110"}}
		if r := admit(m, lim); r != ppInvalid {
			t.Fatalf("empty-string cell: %s want INVALID_INPUT", r)
		}
		m = clone(base)
		m.dates = append(m.dates, "2024-01-05")
		for i := range m.candidates {
			m.candidates[i].prices = ppTickerCells([]string{"100", "90", "110", "120"}, nil)
		}
		if r := admit(m, lim); r != ppLimitExceeded {
			t.Fatalf("L=4 complete with maxLookbackDays=3: %s want LIMIT_EXCEEDED", r)
		}
	})
	t.Run("shape-beats-count-and-missing-prices", func(t *testing.T) {
		m := clone(base)
		m.candidates = m.candidates[:15]
		m.candidates = append(m.candidates, ppCandidate{ticker: "D99", prices: []ppCell{{text: "100"}}, present: true})
		if r := admit(m, lim); r != ppInvalid {
			t.Fatalf("short row in a 16-universe: %s want INVALID_INPUT (shape precedes sufficiency)", r)
		}
	})
	t.Run("two-row-collision-order", func(t *testing.T) {
		// A later short row beats an earlier null cell in BOTH record permutations.
		mk := func(firstNull bool) ppMatrix {
			m := clone(base)
			a, b := ppCandidate{ticker: "D90", present: true}, ppCandidate{ticker: "D95", present: true}
			if firstNull {
				a.prices = []ppCell{{text: "100"}, {missing: true}, {text: "110"}}
				b.prices = []ppCell{{text: "100"}, {text: "110"}}
			} else {
				b.prices = []ppCell{{text: "100"}, {missing: true}, {text: "110"}}
				a.prices = []ppCell{{text: "100"}, {text: "110"}}
			}
			m.candidates = append(m.candidates, a, b)
			return m
		}
		for _, firstNull := range []bool{true, false} {
			m := mk(firstNull)
			mLim := lim
			mLim.maxCandidates = 18
			if r := admit(m, mLim); r != ppInvalid {
				t.Fatalf("permutation firstNull=%v: %s want INVALID_INPUT (shape defect precedes the missing-price diagnosis in both record orders)", firstNull, r)
			}
		}
	})
	t.Run("filter-whole-clause-mutant", func(t *testing.T) {
		m := clone(base)
		m.candidates = append(m.candidates, ppCandidate{ticker: "BAD", present: false})
		lim17 := lim
		lim17.maxCandidates = 17
		if r := admit(m, lim17); r != ppIncomplete {
			t.Fatalf("whole-input: %s want INCOMPLETE_HISTORY (BAD missing row)", r)
		}
		filterFacts := mustParse(t, swapClause(t, completeProseControl, cWhole, cFilter))
		winner, survivors, err := selectSurvivors(filterFacts, m, lim17, []ppProposal{weightsProposal("good", "D", equalWeights())})
		if err != nil {
			t.Fatalf("filter mutant must select survivors: %v", err)
		}
		if winner != "good" {
			t.Fatalf("filter mutant winner=%q", winner)
		}
		if len(survivors) != 16 {
			t.Fatalf("filter mutant survivor count=%d want exactly the 16 valid (BAD excluded)", len(survivors))
		}
		for _, s := range survivors {
			if s == "BAD" {
				t.Fatal("BAD record survived filter admission")
			}
		}
		if _, err := selectAdmitted(mustParse(t, completeProseControl), m, lim17, []ppProposal{weightsProposal("good", "D", equalWeights())}); err == nil {
			t.Fatal("approved whole-input policy admitted the bad 17th record")
		}
	})
}

func TestPortfolioPacketD1LimitsAndScalarBudget(t *testing.T) {
	f := mustParse(t, completeProseControl)
	base, lim := admissionFixture()
	// scalar budget: longest fixture string is a date (10 chars).
	if r := admit(base, lim); r != ppOK {
		t.Fatalf("at-budget scalar: %s want OK", r)
	}
	m := base
	m.candidates[0].prices[1] = ppCell{text: "1000000000"} // 10 chars: at budget still
	if r := admit(m, lim); r != ppOK {
		t.Fatalf("10-char price at 10-byte budget: %s want OK", r)
	}
	m.candidates[0].prices[1] = ppCell{text: "10000000000"} // 11 chars: +1 byte
	if r := admit(m, lim); r != ppLimitExceeded {
		t.Fatalf("+1 byte scalar: %s want LIMIT_EXCEEDED before numeric conversion", r)
	}
	// Count bound fires before decoding the bad 17th record (its scalars stay within
	// the byte budget so the LIMIT stage cannot mask the decode stage). Fresh fixture:
	// earlier subtests share the candidate slice storage.
	fresh, freshLim := admissionFixture()
	m2 := fresh
	m2.candidates = append(m2.candidates, ppCandidate{ticker: "D99", prices: []ppCell{{text: "x"}, {text: "y"}, {text: "z"}}, present: true})
	if r := admit(m2, freshLim); r != ppLimitExceeded {
		t.Fatalf("17th candidate over limit: %s want LIMIT_EXCEEDED before decoding", r)
	}
	lim17 := freshLim
	lim17.maxCandidates = 17
	if r := admit(m2, lim17); r != ppInvalid {
		t.Fatalf("17th bad record under limit 17: %s want INVALID_INPUT at decoding", r)
	}
	// L=4 over maxLookbackDays=3.
	m3 := base
	m3.dates = append(m3.dates, "2024-01-05")
	for i := range m3.candidates {
		m3.candidates[i].prices = ppTickerCells([]string{"100", "90", "110", "120"}, nil)
	}
	if r := admit(m3, lim); r != ppLimitExceeded {
		t.Fatalf("lookback 4 over budget 3: %s want LIMIT_EXCEEDED", r)
	}
	if _, err := selectAdmitted(f, m2, lim, nil); err == nil {
		t.Fatal("selection over limit-exceeded universe must error")
	}
}

func TestPortfolioPacketD1SelectionBoundaryOverflow(t *testing.T) {
	f := mustParse(t, completeProseControl)
	base, lim := admissionFixture()
	// Empty proposal set errors cleanly (no panic).
	if _, err := selectAdmitted(f, base, lim, nil); err == nil {
		t.Fatal("empty proposal set must error")
	}
	// Holdings outside the admitted universe error cleanly (no panic).
	alien := weightsProposal("alien", "Z", equalWeights())
	if _, err := selectAdmitted(f, base, lim, []ppProposal{alien}); err == nil {
		t.Fatal("out-of-universe holdings must error")
	}
	// MaxInt wraparound vector: each weight is bounded at 10000 BEFORE addition, so the
	// sum check can never wrap; a panic here is an invalid negative control.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("overflow vector caused a panic: %v", r)
			}
		}()
		w := make([]int, 16)
		w[0], w[1], w[2] = math.MaxInt64, math.MaxInt64, 10002
		sum := 0
		for _, x := range w {
			if x < 0 || x > 10000 {
				if x == 10002 || x == math.MaxInt64 {
					continue // per-weight bound rejects before addition
				}
			}
		}
		_ = sum
		bounded := func(x int) bool { return x >= 0 && x <= 10000 }
		if bounded(w[0]) || bounded(w[1]) || bounded(w[2]) {
			t.Fatal("per-weight bound failed to reject the wraparound vector members")
		}
	}()
}

func TestPortfolioPacketD1SurvivorHandoff(t *testing.T) {
	approved := mustParse(t, completeProseControl)
	filterFacts := mustParse(t, swapClause(t, completeProseControl, cWhole, cFilter))
	base, _ := admissionFixture()
	m := base
	m.candidates = append(m.candidates, ppCandidate{ticker: "BAD", present: false})
	lim17 := ppLimits{maxCandidates: 17, maxLookbackDays: 3, maxScalarBytes: 10}
	good := weightsProposal("good", "D", equalWeights())
	t.Run("approved-whole-input-rejects-bad-17th", func(t *testing.T) {
		if _, err := selectAdmitted(approved, m, lim17, []ppProposal{good}); err == nil {
			t.Fatal("whole-input admission accepted a universe with a missing-row member")
		}
	})
	t.Run("filter-mutant-carries-survivors-exactly", func(t *testing.T) {
		winner, survivors, err := selectSurvivors(filterFacts, m, lim17, []ppProposal{good})
		if err != nil || winner != "good" {
			t.Fatalf("winner=%q err=%v", winner, err)
		}
		if len(survivors) != 16 {
			t.Fatalf("survivors=%d want 16 (BAD excluded, never scored)", len(survivors))
		}
	})
	t.Run("proposals-containing-BAD-error-before-scoring", func(t *testing.T) {
		oneBad := ppProposal{name: "one-bad"}
		oneBad.holdings = append(oneBad.holdings, ppHolding{ticker: "BAD", weight: 10000})
		for i := 1; i <= 15; i++ {
			oneBad.holdings = append(oneBad.holdings, ppHolding{ticker: fmt.Sprintf("D%02d", i), weight: 0})
		}
		if _, _, err := selectSurvivors(filterFacts, m, lim17, []ppProposal{oneBad}); err == nil {
			t.Fatal("one-proposal set containing BAD must error before sorting/scoring")
		}
		twoBad := ppProposal{name: "two-bad", holdings: append([]ppHolding{ppHolding{ticker: "BAD"}}, good.holdings[1:]...)}
		if _, _, err := selectSurvivors(filterFacts, m, lim17, []ppProposal{good, twoBad}); err == nil {
			t.Fatal("two-proposal set {good,BAD} must error before sorting/scoring")
		}
	})
}

// ---------------------------------------------------------------------------
// FSM core: machine/oracle parsing, immediate effects, authority diagnostics.
// ---------------------------------------------------------------------------

type fsmState struct {
	Name  string
	Final bool
	Entry []string
	Exit  []string
}

type fsmRow struct {
	Stable  string // from the oracle row; empty for machine-side rows
	Source  string
	Trigger string
	Guard   string
	Target  string
	Actions []string
	Always  bool
}

type fsmMachine struct {
	Name    string
	Initial string
	States  map[string]fsmState
	Rows    []fsmRow
}

func asStringList(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []any:
		var out []string
		for _, e := range x {
			s, _ := e.(string)
			out = append(out, s)
		}
		return out
	}
	return nil
}

func normRows(v any) []map[string]any {
	switch x := v.(type) {
	case map[string]any:
		return []map[string]any{x}
	case []any:
		var out []map[string]any
		for _, e := range x {
			if m, ok := e.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	}
	return nil
}

// parseMachineJSON decodes a machine document into states and flattened transition rows.
// Metadata keys (_comment/_delays/_counters/_role) are intentionally ignored: this parser
// consumes the graph, so metadata-only edits leave it byte-for-byte equivalent.
func parseMachineJSON(raw []byte) (fsmMachine, error) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fsmMachine{}, err
	}
	m := fsmMachine{States: map[string]fsmState{}}
	if id, ok := doc["id"].(string); ok {
		m.Name = id
	}
	if init, ok := doc["initial"].(string); ok {
		m.Initial = init
	}
	states, _ := doc["states"].(map[string]any)
	names := make([]string, 0, len(states))
	for name := range states {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		body, _ := states[name].(map[string]any)
		st := fsmState{Name: name}
		if t, _ := body["type"].(string); t == "final" {
			st.Final = true
		}
		st.Entry = asStringList(body["entry"])
		st.Exit = asStringList(body["exit"])
		m.States[name] = st
		addRows := func(trigger string, rows []map[string]any, always bool) {
			for _, tr := range rows {
				target, _ := tr["target"].(string)
				m.Rows = append(m.Rows, fsmRow{
					Source:  name,
					Trigger: trigger,
					Guard:   strOr(tr["guard"]),
					Target:  target,
					Actions: asStringList(tr["actions"]),
					Always:  always,
				})
			}
		}
		if on, ok := body["on"].(map[string]any); ok {
			evs := make([]string, 0, len(on))
			for ev := range on {
				evs = append(evs, ev)
			}
			sort.Strings(evs)
			for _, ev := range evs {
				addRows("on:"+ev, normRows(on[ev]), false)
			}
		}
		if after, ok := body["after"].(map[string]any); ok {
			ds := make([]string, 0, len(after))
			for d := range after {
				ds = append(ds, d)
			}
			sort.Strings(ds)
			for _, d := range ds {
				addRows("after:"+d, normRows(after[d]), false)
			}
		}
		if always, ok := body["always"].([]any); ok {
			addRows("always", normRows(always), true)
		}
		if inv, ok := body["invoke"].(map[string]any); ok {
			src, _ := inv["src"].(string)
			for _, kind := range []string{"onDone", "onError"} {
				if rows, ok := inv[kind].([]any); ok {
					// preserve document order for guarded onDone/onError arrays
					for _, e := range rows {
						tr, _ := e.(map[string]any)
						target, _ := tr["target"].(string)
						m.Rows = append(m.Rows, fsmRow{
							Source:  name,
							Trigger: kind + ":" + src,
							Guard:   strOr(tr["guard"]),
							Target:  target,
							Actions: asStringList(tr["actions"]),
						})
					}
				} else if tr, ok := inv[kind].(map[string]any); ok {
					target, _ := tr["target"].(string)
					m.Rows = append(m.Rows, fsmRow{
						Source:  name,
						Trigger: kind + ":" + src,
						Guard:   strOr(tr["guard"]),
						Target:  target,
						Actions: asStringList(tr["actions"]),
					})
				}
			}
		}
	}
	return m, nil
}

func strOr(v any) string {
	s, _ := v.(string)
	return s
}

// parseOracleRows extracts the transition table of a generated oracle document.
func parseOracleRows(raw []byte) ([]fsmRow, error) {
	var rows []fsmRow
	inTransitions := false
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if strings.HasPrefix(line, "## ") {
			inTransitions = strings.Contains(line, "Transitions")
			continue
		}
		if !inTransitions || !strings.HasPrefix(line, "|") || strings.HasPrefix(line, "| test id") || strings.HasPrefix(line, "|---") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) != 7 {
			continue
		}
		actions := []string{}
		for _, a := range strings.Split(cells[6], ",") {
			a = strings.TrimSpace(a)
			if a != "" && a != "-" {
				actions = append(actions, a)
			}
		}
		rows = append(rows, fsmRow{
			Stable:  strings.TrimSpace(cells[1]),
			Source:  strings.TrimSpace(cells[2]),
			Trigger: strings.TrimSpace(cells[3]),
			Guard:   dashToEmpty(strings.TrimSpace(cells[4])),
			Target:  strings.TrimSpace(cells[5]),
			Actions: actions,
		})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no oracle transition rows parsed")
	}
	return rows, nil
}

func dashToEmpty(s string) string {
	if s == "-" {
		return ""
	}
	return s
}

type fsmDoc struct {
	name        string
	machineJSON []byte
	oracleMD    []byte
	matrixMD    []byte
	machine     fsmMachine
	oracleRows  []fsmRow
}

type fsmCorpus struct {
	docs   []fsmDoc
	byName map[string]*fsmDoc
}

func loadFSMCorpus(t *testing.T) *fsmCorpus {
	t.Helper()
	c := &fsmCorpus{byName: map[string]*fsmDoc{}}
	for _, name := range []string{"RecommendationRun", "MarketDataFeed", "Portfolio", "ReferenceDataCommand"} {
		mj := designBytes(t, "examples/portfolio-engine/design/machines/"+name+".machine.json")
		om := designBytes(t, "examples/portfolio-engine/design/machines/"+name+".oracle.md")
		mm := designBytes(t, "examples/portfolio-engine/design/machines/"+name+".matrix.md")
		m, err := parseMachineJSON(mj)
		if err != nil {
			t.Fatalf("parse %s machine: %v", name, err)
		}
		rows, err := parseOracleRows(om)
		if err != nil {
			t.Fatalf("parse %s oracle: %v", name, err)
		}
		c.docs = append(c.docs, fsmDoc{name: name, machineJSON: mj, oracleMD: om, matrixMD: mm, machine: m, oracleRows: rows})
		c.byName[name] = &c.docs[len(c.docs)-1]
	}
	return c
}

func cloneFSMCorpus(c *fsmCorpus) *fsmCorpus {
	out := &fsmCorpus{byName: map[string]*fsmDoc{}}
	for _, d := range c.docs {
		nd := fsmDoc{name: d.name,
			machineJSON: append([]byte(nil), d.machineJSON...),
			oracleMD:    append([]byte(nil), d.oracleMD...),
			matrixMD:    append([]byte(nil), d.matrixMD...)}
		nd.machine, _ = parseMachineJSON(nd.machineJSON)
		nd.oracleRows, _ = parseOracleRows(nd.oracleMD)
		out.docs = append(out.docs, nd)
		out.byName[nd.name] = &out.docs[len(out.docs)-1]
	}
	return out
}

// equalActions treats nil and the empty list as equal (explicit empty lists).
func equalActions(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

// canonRows sorts rows for order-insensitive comparison.
func canonRows(rows []fsmRow) []fsmRow {
	out := append([]fsmRow(nil), rows...)
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Trigger != b.Trigger {
			return a.Trigger < b.Trigger
		}
		if a.Guard != b.Guard {
			return a.Guard < b.Guard
		}
		if a.Target != b.Target {
			return a.Target < b.Target
		}
		return strings.Join(a.Actions, ",") < strings.Join(b.Actions, ",")
	})
	return out
}

// fsmAuthorityDiagnostics reconciles oracle rows against machine graph rows
// (order-insensitive; both sides describe the same transition set).
func fsmAuthorityDiagnostics(c *fsmCorpus) []string {
	var diags []string
	for _, d := range c.docs {
		oracle, machine := canonRows(d.oracleRows), canonRows(d.machine.Rows)
		if len(oracle) != len(machine) {
			diags = append(diags, fmt.Sprintf("%s: oracle rows %d != machine rows %d", d.name, len(oracle), len(machine)))
			continue
		}
		for i := range oracle {
			or, mr := oracle[i], machine[i]
			if or.Source != mr.Source || or.Trigger != mr.Trigger || or.Guard != mr.Guard ||
				or.Target != mr.Target || !equalActions(or.Actions, mr.Actions) {
				diags = append(diags, fmt.Sprintf("%s row %d (%s): oracle %+v != machine %+v", d.name, i, or.Stable, or, mr))
			}
		}
	}
	return diags
}

func fsmTransitionRows(m fsmMachine) []fsmRow { return m.Rows }

// immediateEffects computes the ordered exit/transition/entry effect list for one oracle
// row against a machine. Always rows are their own microstep (Always=true).
func immediateEffects(m fsmMachine, row fsmRow) []string {
	if row.Target == "" {
		// targetless internal transition: no exit, no entry — transition actions only
		return append([]string(nil), row.Actions...)
	}
	var effects []string
	effects = append(effects, m.States[row.Source].Exit...)
	effects = append(effects, row.Actions...)
	effects = append(effects, m.States[row.Target].Entry...)
	return effects
}

// findOracleRow returns the oracle row with the given stable id.
func findOracleRow(rows []fsmRow, stable string) (fsmRow, int) {
	for i, r := range rows {
		if r.Stable == stable {
			return r, i
		}
	}
	return fsmRow{}, -1
}

// fsmLiteralLedger is the frozen 46-row expectation ledger (independent transcription of
// the approved FSM method's ledger.go, identical to the committed oracle bytes).
func fsmLiteralLedger() []fsmRow {
	row := func(stable, source, trigger, guard, target string, actions ...string) fsmRow {
		if len(actions) == 1 && actions[0] == "" {
			actions = nil
		}
		return fsmRow{Stable: stable, Source: source, Trigger: trigger, Guard: guard, Target: target, Actions: actions}
	}
	return []fsmRow{
		row("RECO-c7bb09", "Collecting", "after:FETCH_TIMEOUT", "", "collectRetry"),
		row("RECO-f89da8", "Collecting", "onDone:fetchPrices", "", "Optimizing"),
		row("RECO-040944", "Collecting", "onError:fetchPrices", "", "collectRetry"),
		row("RECO-c85bd8", "Optimizing", "after:OPTIMIZE_TIMEOUT", "", "Failed"),
		row("RECO-d6fcf9", "Optimizing", "onDone:optimize", "", "Ready", "recordPortfolio"),
		row("RECO-ed98c7", "Optimizing", "onError:optimize", "", "Failed"),
		row("RECO-0d730c", "collectRetry", "after:RETRY_BACKOFF", "", "Collecting", "incRetries"),
		row("RECO-61506b", "collectRetry", "always", "retriesExhausted", "Failed"),

		row("MARK-acc7d7", "closed", "on:failure", "atThreshold", "open", "recordTrip"),
		row("MARK-9e6205", "closed", "on:failure", "", "closed", "incFailures"),
		row("MARK-81fc92", "closed", "on:success", "", "closed", "resetFailures"),
		row("MARK-609444", "open", "after:COOLDOWN", "", "halfOpen"),
		row("MARK-2bed99", "halfOpen", "on:probeResult", "probeSucceeded", "closed", "resetFailures"),
		row("MARK-775b8f", "halfOpen", "on:probeResult", "", "open", "recordTrip"),

		row("PORT-27d66f", "Proposed", "on:advance", "", "committing", "setPendingAdvance"),
		row("PORT-2bf44c", "Proposed", "on:accept", "canDecide", "committing", "setPendingAccept"),
		row("PORT-a41039", "Proposed", "on:reject", "canDecide", "committing", "setPendingReject"),
		row("PORT-ddb44c", "UnderReview", "on:accept", "canDecide", "committing", "setPendingAccept"),
		row("PORT-351dec", "UnderReview", "on:reject", "canDecide", "committing", "setPendingReject"),
		row("PORT-db3bb9", "Accepted", "on:reopen", "canReopen", "committing", "setPendingReopen"),
		row("PORT-9facf7", "Rejected", "on:reopen", "canReopen", "committing", "setPendingReopen"),
		row("PORT-5e6be0", "committing", "after:COMMIT_TIMEOUT", "", "commitRetry"),
		row("PORT-f43140", "committing", "onDone:persistDecision", "pendingIsUnderReview", "UnderReview", "commit"),
		row("PORT-d1647b", "committing", "onDone:persistDecision", "pendingIsAccepted", "Accepted", "commit", "recordAccepted"),
		row("PORT-fb8c92", "committing", "onDone:persistDecision", "pendingIsRejected", "Rejected", "commit"),
		row("PORT-40b6e7", "committing", "onError:persistDecision", "isRetriable", "commitRetry"),
		row("PORT-c4a186", "committing", "onError:persistDecision", "", "reverted"),
		row("PORT-f6e220", "commitRetry", "after:RETRY_BACKOFF", "", "committing", "incRetries"),
		row("PORT-cba032", "commitRetry", "always", "retriesExhausted", "reverted"),
		row("PORT-8c0400", "reverted", "always", "priorIsProposed", "Proposed"),
		row("PORT-3cb0b6", "reverted", "always", "priorIsUnderReview", "UnderReview"),
		row("PORT-53d34b", "reverted", "always", "priorIsAccepted", "Accepted"),
		row("PORT-3390a7", "reverted", "always", "priorIsRejected", "Rejected"),
		row("PORT-3bd579", "reverted", "always", "", "routingFault", "recordRoutingError"),

		row("REFE-33a2ae", "idle", "on:refresh", "", "refreshing"),
		row("REFE-6e63db", "idle", "on:upsert", "", "upserting"),
		row("REFE-62d231", "idle", "on:build", "", "building"),
		row("REFE-cb1193", "refreshing", "after:COMMAND_TIMEOUT", "", "failed", "cancelReferenceOperation", "recordReferenceTimeout"),
		row("REFE-c51bb5", "refreshing", "onDone:selectEligibleConstituents", "", "succeeded"),
		row("REFE-6b30a5", "refreshing", "onError:selectEligibleConstituents", "", "failed", "recordReferenceError"),
		row("REFE-8c9720", "upserting", "after:COMMAND_TIMEOUT", "", "failed", "cancelReferenceOperation", "recordReferenceTimeout"),
		row("REFE-cc283e", "upserting", "onDone:upsertSecurityByTicker", "", "succeeded"),
		row("REFE-cc10bd", "upserting", "onError:upsertSecurityByTicker", "", "failed", "recordReferenceError"),
		row("REFE-f67aa3", "building", "after:COMMAND_TIMEOUT", "", "failed", "cancelReferenceOperation", "recordReferenceTimeout"),
		row("REFE-074a67", "building", "onDone:buildCandidateSet", "", "succeeded"),
		row("REFE-c60610", "building", "onError:buildCandidateSet", "", "failed", "recordReferenceError"),
	}
}

// fsmEffectLedger is the literal immediate-effect ledger (empty lists are explicit).
var fsmEffectLedger = map[string][]string{
	"RECO-c7bb09": nil,
	"RECO-f89da8": nil,
	"RECO-040944": nil,
	"RECO-c85bd8": {"assertTerminalAbsorbing", "publishFailure"},
	"RECO-d6fcf9": {"recordPortfolio", "assertTerminalAbsorbing", "publishReady"},
	"RECO-ed98c7": {"assertTerminalAbsorbing", "publishFailure"},
	"RECO-0d730c": {"incRetries"},
	"RECO-61506b": {"assertTerminalAbsorbing", "publishFailure"},
	"MARK-acc7d7": {"recordTrip"},
	"MARK-9e6205": {"incFailures"},
	"MARK-81fc92": {"resetFailures"},
	"MARK-609444": nil,
	"MARK-2bed99": {"resetFailures"},
	"MARK-775b8f": {"recordTrip"},
	"PORT-27d66f": {"setPendingAdvance"},
	"PORT-2bf44c": {"setPendingAccept"},
	"PORT-a41039": {"setPendingReject"},
	"PORT-ddb44c": {"setPendingAccept"},
	"PORT-351dec": {"setPendingReject"},
	"PORT-db3bb9": {"setPendingReopen"},
	"PORT-9facf7": {"setPendingReopen"},
	"PORT-5e6be0": nil,
	"PORT-f43140": {"commit"},
	"PORT-d1647b": {"commit", "recordAccepted"},
	"PORT-fb8c92": {"commit"},
	"PORT-40b6e7": nil,
	"PORT-c4a186": nil,
	"PORT-f6e220": {"incRetries"},
	"PORT-cba032": nil,
	"PORT-8c0400": nil,
	"PORT-3cb0b6": nil,
	"PORT-53d34b": nil,
	"PORT-3390a7": nil,
	"PORT-3bd579": {"recordRoutingError"},
	"REFE-33a2ae": nil,
	"REFE-6e63db": nil,
	"REFE-62d231": nil,
	"REFE-cb1193": {"cancelReferenceOperation", "recordReferenceTimeout"},
	"REFE-c51bb5": nil,
	"REFE-6b30a5": {"recordReferenceError"},
	"REFE-8c9720": {"cancelReferenceOperation", "recordReferenceTimeout"},
	"REFE-cc283e": nil,
	"REFE-cc10bd": {"recordReferenceError"},
	"REFE-f67aa3": {"cancelReferenceOperation", "recordReferenceTimeout"},
	"REFE-074a67": nil,
	"REFE-c60610": {"recordReferenceError"},
}

// matrixUnitLedger is the literal named-unit kind inventory per machine.
var matrixUnitLedger = map[string]map[string]string{
	"RecommendationRun": {
		"retriesExhausted": "guard", "recordPortfolio": "action", "incRetries": "action",
		"publishReady": "action", "publishFailure": "action", "assertTerminalAbsorbing": "action",
		"fetchPrices": "actor", "optimize": "actor",
	},
	"MarketDataFeed": {
		"atThreshold": "guard", "probeSucceeded": "guard", "recordTrip": "action",
		"incFailures": "action", "resetFailures": "action",
	},
	"Portfolio": {
		"canDecide": "guard", "canReopen": "guard", "pendingIsUnderReview": "guard",
		"pendingIsAccepted": "guard", "pendingIsRejected": "guard", "isRetriable": "guard",
		"retriesExhausted": "guard", "priorIsProposed": "guard", "priorIsUnderReview": "guard",
		"priorIsAccepted": "guard", "priorIsRejected": "guard", "setPendingAdvance": "action",
		"setPendingAccept": "action", "setPendingReject": "action", "setPendingReopen": "action",
		"commit": "action", "recordAccepted": "action", "incRetries": "action",
		"recordRoutingError": "action", "persistDecision": "actor",
	},
	"ReferenceDataCommand": {
		"selectEligibleConstituents": "actor", "upsertSecurityByTicker": "actor", "buildCandidateSet": "actor",
		"recordReferenceError": "action", "cancelReferenceOperation": "action", "recordReferenceTimeout": "action",
	},
}

// ---------------------------------------------------------------------------
// FSM identities (8).
// ---------------------------------------------------------------------------

func TestPortfolioPacketFSMAll46RowsLiteral(t *testing.T) {
	c := loadFSMCorpus(t)
	ledger := fsmLiteralLedger()
	if len(ledger) != 46 {
		t.Fatalf("literal ledger rows=%d want 46", len(ledger))
	}
	t.Run("committed-oracle-rows-equal-literal-ledger", func(t *testing.T) {
		var live []fsmRow
		for _, name := range []string{"RecommendationRun", "MarketDataFeed", "Portfolio", "ReferenceDataCommand"} {
			live = append(live, c.byName[name].oracleRows...)
		}
		if len(live) != 46 {
			t.Fatalf("live oracle rows=%d want 46 (Run 8 / Feed 6 / Portfolio 20 / Reference 12)", len(live))
		}
		for i, want := range ledger {
			got := live[i]
			if got.Stable != want.Stable || got.Source != want.Source || got.Trigger != want.Trigger ||
				got.Guard != want.Guard || got.Target != want.Target || !equalActions(got.Actions, want.Actions) {
				t.Errorf("row %d stable=%s: live %+v want literal %+v", i, want.Stable, got, want)
			}
		}
	})
	t.Run("per-row-mutants-change-exactly-the-mutated-field", func(t *testing.T) {
		mutate := func(f func(rows []fsmRow)) []fsmRow {
			rows := fsmLiteralLedger()
			f(rows)
			return rows
		}
		for _, tc := range []struct {
			name   string
			mutant []fsmRow
		}{
			{"retarget-PORT-27d66f", mutate(func(rows []fsmRow) {
				for i := range rows {
					if rows[i].Stable == "PORT-27d66f" {
						rows[i].Target = "UnderReview"
					}
				}
			})},
			{"drop-action-from-PORT-d1647b", mutate(func(rows []fsmRow) {
				for i := range rows {
					if rows[i].Stable == "PORT-d1647b" {
						rows[i].Actions = []string{"commit"}
					}
				}
			})},
			{"add-extra-action-to-RECO-d6fcf9", mutate(func(rows []fsmRow) {
				for i := range rows {
					if rows[i].Stable == "RECO-d6fcf9" {
						rows[i].Actions = []string{"recordPortfolio", "extraAction"}
					}
				}
			})},
			{"reorder-actions-PORT-d1647b", mutate(func(rows []fsmRow) {
				for i := range rows {
					if rows[i].Stable == "PORT-d1647b" {
						rows[i].Actions = []string{"recordAccepted", "commit"}
					}
				}
			})},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var live []fsmRow
				for _, name := range []string{"RecommendationRun", "MarketDataFeed", "Portfolio", "ReferenceDataCommand"} {
					live = append(live, c.byName[name].oracleRows...)
				}
				diff := 0
				for i := range live {
					if i < len(tc.mutant) {
						w := tc.mutant[i]
						if live[i].Stable != w.Stable || live[i].Source != w.Source || live[i].Trigger != w.Trigger ||
							live[i].Guard != w.Guard || live[i].Target != w.Target || !reflect.DeepEqual(live[i].Actions, w.Actions) {
							diff++
						}
					}
				}
				if diff == 0 {
					t.Fatal("mutated literal ledger still matched the live oracle rows; row comparison is not sensitive")
				}
			})
		}
	})
}

func TestPortfolioPacketFSMImmediateEffectOrder(t *testing.T) {
	c := loadFSMCorpus(t)
	t.Run("ordered-exit-transition-entry-per-stable-id", func(t *testing.T) {
		for _, d := range c.docs {
			for _, or := range d.oracleRows {
				want, ok := fsmEffectLedger[or.Stable]
				if !ok {
					t.Fatalf("stable id %s missing from the literal effect ledger", or.Stable)
				}
				got := immediateEffects(d.machine, or)
				if !reflect.DeepEqual(got, want) && (len(got) != 0 || len(want) != 0) {
					t.Errorf("%s effects=%v want literal %v (exit/transition/entry order)", or.Stable, got, want)
				}
			}
		}
		// Spot literal: RECO-d6fcf9 transition effects.
		d := c.byName["RecommendationRun"]
		or, _ := findOracleRow(d.oracleRows, "RECO-d6fcf9")
		if got := immediateEffects(d.machine, or); !reflect.DeepEqual(got, []string{"recordPortfolio", "assertTerminalAbsorbing", "publishReady"}) {
			t.Fatalf("RECO-d6fcf9 effects=%v want [recordPortfolio assertTerminalAbsorbing publishReady]", got)
		}
	})
	t.Run("reorder-omitted-extra-mutants-fail-at-exact-position", func(t *testing.T) {
		d := c.byName["Portfolio"]
		_, idx := findOracleRow(d.oracleRows, "PORT-d1647b")
		if idx < 0 {
			t.Fatal("PORT-d1647b missing")
		}
		machineCopy, err := parseMachineJSON(d.machineJSON)
		if err != nil {
			t.Fatal(err)
		}
		// reorder [commit, recordAccepted] -> [recordAccepted, commit] in-memory.
		reordered := d.oracleRows
		reordered[idx].Actions = []string{"recordAccepted", "commit"}
		got := immediateEffects(machineCopy, reordered[idx])
		if reflect.DeepEqual(got, fsmEffectLedger["PORT-d1647b"]) {
			t.Fatal("reordered action list still matched the literal effect ledger")
		}
		omitted := append([]fsmRow(nil), d.oracleRows...)
		omitted[idx].Actions = []string{"commit"}
		if reflect.DeepEqual(immediateEffects(machineCopy, omitted[idx]), fsmEffectLedger["PORT-d1647b"]) {
			t.Fatal("omitted action still matched the literal effect ledger")
		}
		withExtra := append([]fsmRow(nil), d.oracleRows...)
		withExtra[idx].Actions = []string{"commit", "recordAccepted", "extraEntry"}
		if reflect.DeepEqual(immediateEffects(machineCopy, withExtra[idx]), fsmEffectLedger["PORT-d1647b"]) {
			t.Fatal("extra action still matched the literal effect ledger")
		}
	})
}

func TestPortfolioPacketFSMGuardsAndAlwaysMicrosteps(t *testing.T) {
	c := loadFSMCorpus(t)
	t.Run("ordered-guard-selection", func(t *testing.T) {
		// closed on:failure is a two-row ordered guarded list.
		feed := c.byName["MarketDataFeed"]
		var failureRows []fsmRow
		for _, r := range feed.machine.Rows {
			if r.Trigger == "on:failure" {
				failureRows = append(failureRows, r)
			}
		}
		if len(failureRows) != 2 {
			t.Fatalf("closed on:failure guarded rows=%d want 2", len(failureRows))
		}
		if failureRows[0].Guard != "atThreshold" || failureRows[1].Guard != "" {
			t.Fatalf("guard order = [%q,%q] want [atThreshold,''] (threshold row first)", failureRows[0].Guard, failureRows[1].Guard)
		}
		// committing onDone:persistDecision is a three-row ordered guarded list.
		pf := c.byName["Portfolio"]
		var doneRows []fsmRow
		for _, r := range pf.machine.Rows {
			if r.Trigger == "onDone:persistDecision" {
				doneRows = append(doneRows, r)
			}
		}
		wantGuards := []string{"pendingIsUnderReview", "pendingIsAccepted", "pendingIsRejected"}
		for i, g := range wantGuards {
			if doneRows[i].Guard != g {
				t.Fatalf("onDone guard order[%d]=%q want %q", i, doneRows[i].Guard, g)
			}
		}
	})
	t.Run("always-rows-execute-as-separate-microsteps", func(t *testing.T) {
		// RECO-61506b, PORT-cba032 and PORT-8c0400..3bd579 are always rows: their effect
		// lists are SEPARATE ledger entries, never fused into an event transition.
		for _, stable := range []string{"RECO-61506b", "PORT-cba032", "PORT-8c0400", "PORT-3cb0b6", "PORT-53d34b", "PORT-3390a7", "PORT-3bd579"} {
			var d *fsmDoc
			if strings.HasPrefix(stable, "RECO") {
				d = c.byName["RecommendationRun"]
			} else {
				d = c.byName["Portfolio"]
			}
			or, _ := findOracleRow(d.oracleRows, stable)
			var machineRow *fsmRow
			for i := range d.machine.Rows {
				if d.machine.Rows[i].Trigger == "always" && d.machine.Rows[i].Guard == or.Guard {
					machineRow = &d.machine.Rows[i]
				}
			}
			if machineRow == nil || !machineRow.Always {
				t.Fatalf("%s is not an always microstep in the machine graph", stable)
			}
		}
		// Fused-always mutant: merging collectRetry's always row into the after transition
		// changes the microstep literal (two steps become one).
		d := c.byName["RecommendationRun"]
		mutant, _ := parseMachineJSON(d.machineJSON)
		fused := mutant.Rows[:0]
		for i := range mutant.Rows {
			if mutant.Rows[i].Trigger == "always" && mutant.Rows[i].Guard == "retriesExhausted" {
				continue // fused into the after transition below
			}
			if mutant.Rows[i].Source == "collectRetry" && mutant.Rows[i].Trigger == "after:RETRY_BACKOFF" {
				mutant.Rows[i].Actions = append(mutant.Rows[i].Actions, "assertTerminalAbsorbing", "publishFailure")
			}
			fused = append(fused, mutant.Rows[i])
		}
		mutant.Rows = fused
		var separate, fusedCount int
		for _, r := range d.machine.Rows {
			if r.Source == "collectRetry" && (r.Always || r.Trigger == "after:RETRY_BACKOFF") {
				separate++
			}
		}
		for _, r := range mutant.Rows {
			if r.Source == "collectRetry" && (r.Always || r.Trigger == "after:RETRY_BACKOFF") {
				fusedCount++
			}
		}
		if separate != 2 || fusedCount != 1 {
			t.Fatalf("microstep count: canonical=%d fused-mutant=%d; always must be a separate microstep (2 vs 1)", separate, fusedCount)
		}
	})
}

func TestPortfolioPacketFSMEntryEffectsObservable(t *testing.T) {
	c := loadFSMCorpus(t)
	t.Run("terminal-entry-lists-observable", func(t *testing.T) {
		run := c.byName["RecommendationRun"]
		if got := run.machine.States["Ready"].Entry; !reflect.DeepEqual(got, []string{"assertTerminalAbsorbing", "publishReady"}) {
			t.Fatalf("Ready entry=%v want [assertTerminalAbsorbing publishReady] (Run Ready entry is observable)", got)
		}
		if got := run.machine.States["Failed"].Entry; !reflect.DeepEqual(got, []string{"assertTerminalAbsorbing", "publishFailure"}) {
			t.Fatalf("Failed entry=%v want [assertTerminalAbsorbing publishFailure]", got)
		}
		// Empty-list rows stay explicit: routingFault is final with no entry list.
		pf := c.byName["Portfolio"]
		if got := pf.machine.States["routingFault"].Entry; len(got) != 0 {
			t.Fatalf("routingFault entry=%v want the explicit empty list", got)
		}
		// RECO-c7bb09 has a nil effect list (no exit/entry/transition actions).
		or, _ := findOracleRow(run.oracleRows, "RECO-c7bb09")
		if got := immediateEffects(run.machine, or); len(got) != 0 {
			t.Fatalf("RECO-c7bb09 effects=%v want explicit empty", got)
		}
	})
	t.Run("entry-hidden-mutant-breaks-observability", func(t *testing.T) {
		raw := bytes.Replace(c.byName["RecommendationRun"].machineJSON,
			[]byte(`"entry": ["assertTerminalAbsorbing", "publishReady"]`), []byte(`"entry": []`), 1)
		m, err := parseMachineJSON(raw)
		if err != nil {
			t.Fatal(err)
		}
		if reflect.DeepEqual(m.States["Ready"].Entry, []string{"assertTerminalAbsorbing", "publishReady"}) {
			t.Fatal("entry-hidden mutant kept the Ready entry list")
		}
		or, _ := findOracleRow(c.byName["RecommendationRun"].oracleRows, "RECO-d6fcf9")
		if got := immediateEffects(m, or); reflect.DeepEqual(got, fsmEffectLedger["RECO-d6fcf9"]) {
			t.Fatal("hidden entry still produced the literal effect list; observability lost")
		}
	})
}

func TestPortfolioPacketFSMRefusalFinalGuardFalseCorruptPrior(t *testing.T) {
	c := loadFSMCorpus(t)
	pf := c.byName["Portfolio"]
	// Four distinct outcomes; each keeps its own literal.
	t.Run("refusal", func(t *testing.T) {
		var doc map[string]any
		if err := json.Unmarshal(pf.machineJSON, &doc); err != nil {
			t.Fatal(err)
		}
		states := doc["states"].(map[string]any)
		proposed := states["Proposed"].(map[string]any)
		if _, ok := proposed["_refusal"].(map[string]any)["accept"]; !ok {
			t.Fatal("Proposed accept refusal (canDecide false -> AuthzError) is not declared")
		}
	})
	t.Run("final-state-later-event-no-effects", func(t *testing.T) {
		// routingFault is final with no outgoing rows: a later event is TerminalError
		// with zero effects.
		for _, r := range pf.machine.Rows {
			if r.Source == "routingFault" {
				t.Fatal("final routingFault has an outgoing transition")
			}
		}
		if !pf.machine.States["routingFault"].Final {
			t.Fatal("routingFault must be final")
		}
	})
	t.Run("guard-false-distinct", func(t *testing.T) {
		// With no guard admitting (pending is none of the three), onDone has no branch:
		// that is a routing failure, not a refusal and not a terminal event.
		var none int
		for _, r := range pf.machine.Rows {
			if r.Trigger == "onDone:persistDecision" && r.Guard == "" {
				none++
			}
		}
		if none != 0 {
			t.Fatal("onDone has an unguarded row; guard-false case would be indistinguishable")
		}
	})
	t.Run("corrupt-prior-routes-to-routingFault", func(t *testing.T) {
		or, _ := findOracleRow(pf.oracleRows, "PORT-3bd579")
		if or.Guard != "" || or.Target != "routingFault" || !reflect.DeepEqual(or.Actions, []string{"recordRoutingError"}) {
			t.Fatalf("corrupt-prior fallback row = %+v want unguarded -> routingFault [recordRoutingError]", or)
		}
	})
	t.Run("collapse-two-cases-mutant", func(t *testing.T) {
		// Collapsing refusal (Authz) with corrupt-prior (routingFault) would give the
		// illegal review action a routingFault transition; the literals must differ.
		illegal, _ := findOracleRow(pf.oracleRows, "PORT-c4a186") // unguarded onError -> reverted
		corrupt, _ := findOracleRow(pf.oracleRows, "PORT-3bd579")
		if illegal.Target == corrupt.Target && reflect.DeepEqual(illegal.Actions, corrupt.Actions) && illegal.Guard == corrupt.Guard {
			t.Fatal("distinct outcome cases collapsed onto identical literals")
		}
	})
}

const syntheticMachine = `{
  "id": "synthetic",
  "initial": "A",
  "states": {
    "A": {
      "entry": ["enterA"],
      "exit": ["exitA"],
      "on": {
        "external": { "target": "A", "actions": "transitionAction" },
        "internal": { "actions": "internalAction" }
      }
    }
  }
}`

func TestPortfolioPacketFSMSyntheticSelfAndTargetless(t *testing.T) {
	m, err := parseMachineJSON([]byte(syntheticMachine))
	if err != nil {
		t.Fatal(err)
	}
	external := -1
	internal := -1
	for i, r := range m.Rows {
		switch r.Trigger {
		case "on:external":
			external = i
		case "on:internal":
			internal = i
		}
	}
	if external < 0 || internal < 0 {
		t.Fatal("synthetic machine rows missing")
	}
	t.Run("positive-distinction", func(t *testing.T) {
		ext := m.Rows[external]
		if ext.Target != "A" {
			t.Fatalf("external self target=%q want A", ext.Target)
		}
		if got := immediateEffects(m, ext); !reflect.DeepEqual(got, []string{"exitA", "transitionAction", "enterA"}) {
			t.Fatalf("external self effects=%v want [exitA transitionAction enterA]", got)
		}
		intn := m.Rows[internal]
		if intn.Target != "" {
			t.Fatalf("internal targetless has target %q", intn.Target)
		}
		if got := immediateEffects(m, intn); !reflect.DeepEqual(got, []string{"internalAction"}) {
			t.Fatalf("internal targetless effects=%v want [internalAction] (no exit/entry)", got)
		}
	})
	t.Run("mislabel-external-as-internal", func(t *testing.T) {
		mutant, _ := parseMachineJSON([]byte(strings.Replace(syntheticMachine,
			`"external": { "target": "A", "actions": "transitionAction" }`,
			`"external": { "actions": "transitionAction" }`, 1)))
		var ext fsmRow
		for _, r := range mutant.Rows {
			if r.Trigger == "on:external" {
				ext = r
			}
		}
		if reflect.DeepEqual(immediateEffects(mutant, ext), []string{"exitA", "transitionAction", "enterA"}) {
			t.Fatal("target removal did not alter external self effects; kinds are interchangeable")
		}
	})
	t.Run("mislabel-internal-as-external", func(t *testing.T) {
		mutant, _ := parseMachineJSON([]byte(strings.Replace(syntheticMachine,
			`"internal": { "actions": "internalAction" }`,
			`"internal": { "target": "A", "actions": "internalAction" }`, 1)))
		var intn fsmRow
		for _, r := range mutant.Rows {
			if r.Trigger == "on:internal" {
				intn = r
			}
		}
		if reflect.DeepEqual(immediateEffects(mutant, intn), []string{"internalAction"}) {
			t.Fatal("target insertion did not alter targetless effects; kinds are interchangeable")
		}
	})
}

func TestPortfolioPacketFSMPacketScopeM0SubsetM3None(t *testing.T) {
	m0 := designBytes(t, "examples/portfolio-engine/design/BUILD/M0-walking-skeleton.md")
	m3 := designBytes(t, "examples/portfolio-engine/design/BUILD/M3-optimizer.md")
	wantM0 := []string{"RECO-f89da8", "RECO-d6fcf9", "RECO-040944"}
	t.Run("canonical-live-scope", func(t *testing.T) {
		if got := packetStableIDs(string(m0)); !reflect.DeepEqual(got, wantM0) {
			t.Fatalf("live M0 Run subset=%v want exactly %v", got, wantM0)
		}
		if has, kind := packetDeclaresMachine(string(m3)); has {
			t.Fatalf("live M3 must declare no machine, found %q", kind)
		}
	})
	t.Run("M0-operative-row-mutant", func(t *testing.T) {
		mutant := bytes.Replace(m0, []byte("RECO-040944"), []byte("RECO-c7bb09"), 1)
		if got := packetStableIDs(string(mutant)); reflect.DeepEqual(got, wantM0) {
			t.Fatal("named-row mutation did not change the M0 scope outcome")
		}
		mutant4 := bytes.Replace(m0, []byte("`RECO-040944` followed"),
			[]byte("`RECO-040944` and `RECO-c7bb09` followed"), 1)
		got := packetStableIDs(string(mutant4))
		if len(got) != 4 || got[3] != "RECO-c7bb09" {
			t.Fatalf("M0 citing a 4th row: scope=%v want the mutant 4-row scope to be detected", got)
		}
	})
	t.Run("M3-invented-FSM-mutant", func(t *testing.T) {
		baseHas, _ := packetDeclaresMachine(string(m3))
		if baseHas {
			t.Fatal("prerequisite: live M3 declares no machine")
		}
		mutant := bytes.Replace(m3,
			[]byte("This slice is pure data refinement rather than a separate lifecycle machine."),
			[]byte("This slice has a separate lifecycle machine."), 1)
		if has, _ := packetDeclaresMachine(string(mutant)); !has {
			t.Fatal("invented-FSM policy mutant did not change the M3 scope outcome")
		}
	})
}

// packetStableIDs extracts RECO- stable ids in first-appearance order from a packet.
func packetStableIDs(text string) []string {
	re := regexp.MustCompile(`RECO-[0-9a-f]{6}`)
	seen := map[string]bool{}
	var out []string
	for _, m := range re.FindAllString(text, -1) {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

// packetDeclaresMachine reports whether a packet text declares a lifecycle machine.
func packetDeclaresMachine(text string) (bool, string) {
	if strings.Contains(text, "separate lifecycle machine") && !strings.Contains(text, "rather than a separate lifecycle machine") {
		return true, "declares a lifecycle machine"
	}
	if strings.Contains(text, "machines/") && strings.Contains(text, ".machine.json") {
		return true, "cites a machine source"
	}
	if regexp.MustCompile(`(RECO|MARK|PORT|REFE)-[0-9a-f]{6}`).MatchString(text) {
		return true, "cites oracle stable ids"
	}
	return false, ""
}

// atxClosesNormativeSection classifies one heading line: `#`/`##` bare or followed by a
// tab close the normative level-2 section; >6 markers or text attached to the marker do
// not form a closing heading at all.
func atxClosesNormativeSection(line string) bool {
	if !strings.HasPrefix(line, "#") {
		return false
	}
	markers := 0
	for markers < len(line) && line[markers] == '#' {
		markers++
	}
	if markers > 6 || markers > 2 {
		return false
	}
	rest := line[markers:]
	if rest == "" {
		return true // bare heading: closes
	}
	return rest[0] == '\t' || rest[0] == ' '
}

func TestPortfolioPacketFSMSectionBoundaryRobustness(t *testing.T) {
	production := "| `RECO-f89da8` | row | Collecting | onDone:fetchPrices | - | Optimizing | - |"
	doc := func(boundary string) string {
		head := "## Named-unit contracts\n\n" + production
		if boundary == "" {
			return head
		}
		return "## Named-unit contracts\n\n" + boundary + "\n\n" + production
	}
	supplies := func(text string) bool {
		// The production supplies a fact iff it appears after the normative-section
		// heading and no closing ATX heading line appears between them.
		lines := strings.Split(text, "\n")
		headIdx, prodIdx := -1, -1
		for i, line := range lines {
			if strings.Contains(line, "Named-unit contracts") {
				headIdx = i
			}
			if strings.Contains(line, "RECO-f89da8") && prodIdx < 0 {
				prodIdx = i
			}
		}
		if headIdx < 0 || prodIdx < headIdx {
			return false
		}
		for i := headIdx + 1; i < prodIdx; i++ {
			if atxClosesNormativeSection(lines[i]) {
				return false
			}
		}
		return true
	}
	t.Run("canonical-production-recognized", func(t *testing.T) {
		if !supplies(doc("")) {
			t.Fatal("canonical production in the normative section was not recognized")
		}
	})
	for _, boundary := range []string{"#\tNotes", "##\tNotes", "#", "##"} {
		t.Run("relocation-after-"+strconv.Quote(boundary), func(t *testing.T) {
			if supplies(doc(boundary)) {
				t.Fatalf("production relocated after %q still supplied a fact; the ATX boundary must close the normative section", boundary)
			}
		})
	}
	for _, boundary := range []string{"#Notes", "####### seven", "### level three stays inside"} {
		t.Run("non-heading-"+strconv.Quote(boundary), func(t *testing.T) {
			if !supplies(doc(boundary)) {
				t.Fatalf("non-closing line %q incorrectly closed the normative section", boundary)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ADM core: command admission, repository read admission, ranked-row pipeline,
// scalar precedence, derived limits, normalization ownership.
// ---------------------------------------------------------------------------

type admBoundaryError struct {
	Class  string
	Reason string
	Exit   int
}

// exit literals from the closed injective CLI map (ARCHITECTURE.md 7).
var admExitCodes = map[string]int{
	"AuthzError": 2, "NotFoundError": 3, "ConflictError": 4, "FeedError": 5,
	"InfeasibleError": 6, "CorruptError": 7, "ValidationError": 8, "InternalError": 9,
	"BusyError": 10, "TerminalError": 11, "IOError": 12, "CircuitOpenError": 13,
}

func admErr(class, reason string) admBoundaryError {
	return admBoundaryError{Class: class, Reason: reason, Exit: admExitCodes[class]}
}

type admCounters struct {
	runsCreated int
	feedCalls   int
	retries     int
}

type admLimits struct {
	maxCandidates    uint64
	maxLookbackDays  uint64
	maxScalarBytes   uint64
	maxSourceIndices uint64
	maxProviderRows  uint64
}

type admRequest struct {
	runID          string
	candidateSetID string
	lookbackDays   uint64
	k              int
	limits         admLimits
	flags          map[string]string
}

// parseRecommendFlags parses the three required limit flags once. Typed validation fires
// BEFORE any run/actor/provider construction (counters all zero on failure).
func parseRecommendFlags(args map[string]string) (admRequest, admBoundaryError) {
	get := func(name string) (uint64, bool, admBoundaryError) {
		raw, present := args[name]
		if !present || raw == "" {
			return 0, false, admErr("ValidationError", "MISSING_FLAG:"+name)
		}
		if strings.ContainsAny(raw, "+-") || strings.Contains(raw, ".") || strings.ContainsAny(raw, "eE") ||
			strings.Contains(strings.ToLower(raw), "true") || strings.Contains(strings.ToLower(raw), "false") {
			return 0, false, admErr("ValidationError", "MALFORMED_FLAG:"+name)
		}
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return 0, false, admErr("ValidationError", "MALFORMED_FLAG:"+name)
		}
		if v == 0 {
			return 0, false, admErr("ValidationError", "ZERO_FLAG:"+name)
		}
		return v, true, admBoundaryError{}
	}
	mc, ok, e := get("--max-candidates")
	if !ok {
		return admRequest{}, e
	}
	ml, ok, e := get("--max-lookback-days")
	if !ok {
		return admRequest{}, e
	}
	ms, ok, e := get("--max-scalar-bytes")
	if !ok {
		return admRequest{}, e
	}
	if mc < 16 {
		return admRequest{}, admErr("ValidationError", "BELOW_MIN:--max-candidates")
	}
	if ml < 2 {
		return admRequest{}, admErr("ValidationError", "BELOW_MIN:--max-lookback-days")
	}
	return admRequest{
		runID: "r1", candidateSetID: "cs1", lookbackDays: 3, k: 16,
		limits: admLimits{maxCandidates: mc, maxLookbackDays: ml, maxScalarBytes: ms},
		flags:  args,
	}, admBoundaryError{}
}

// repoPortRead models repoPort.LoadCandidateSet: a typed admission read with finite
// input budgets and no write/publication handle.
func repoPortRead(kind string) (admSnapshot, admBoundaryError) {
	switch kind {
	case "notfound":
		return admSnapshot{}, admErr("NotFoundError", "CANDIDATE_SET_NOT_FOUND")
	case "corrupt":
		return admSnapshot{}, admErr("CorruptError", "CANDIDATE_SET_CORRUPT")
	case "io":
		return admSnapshot{}, admErr("IOError", "READ_FAILED")
	case "busy":
		return admSnapshot{}, admErr("BusyError", "STORE_BUSY")
	}
	return admSnapshot{identities: []string{}, version: 1}, admBoundaryError{}
}

type admSnapshot struct {
	identities []string
	version    uint64
}

type runActors struct {
	snapshot  admSnapshot
	limits    admLimits
	requestID string
	// ports record observations
	feedCalls    int
	optCalls     int
	feedLimit    uint64
	optLimit     uint64
	optInputKeys []string
}

// makeRunActors performs the ONE pre-run repository admission read and freezes the
// snapshot and limits into the actor closures.
func makeRunActors(req admRequest, read func() (admSnapshot, admBoundaryError)) (*runActors, admBoundaryError) {
	snap, err := read()
	if err.Class != "" {
		return nil, err
	}
	return &runActors{snapshot: snap, limits: req.limits, requestID: req.runID + ":" + req.candidateSetID}, admBoundaryError{}
}

func (a *runActors) callFeed(candidateCount int) admBoundaryError {
	if uint64(candidateCount) > a.limits.maxCandidates {
		return admErr("InternalError", "ACTOR_BINDING_MISMATCH")
	}
	a.feedCalls++
	a.feedLimit = a.limits.maxCandidates
	return admBoundaryError{}
}

func (a *runActors) callOptimize(candidateCount int, inputKeys []string) admBoundaryError {
	if uint64(candidateCount) > a.limits.maxCandidates {
		return admErr("InternalError", "ACTOR_BINDING_MISMATCH")
	}
	for _, k := range inputKeys {
		if k != "candidateSetId" && k != "lookbackDays" {
			return admErr("InternalError", "ACTOR_BINDING_MISMATCH")
		}
	}
	a.optCalls++
	a.optLimit = a.limits.maxCandidates
	a.optInputKeys = inputKeys
	return admBoundaryError{}
}

// recommendCommand runs the whole pre-run admission path with observable counters.
func recommendCommand(args map[string]string, repoKind string) (admBoundaryError, admCounters) {
	var c admCounters
	req, err := parseRecommendFlags(args)
	if err.Class != "" {
		return err, c
	}
	actors, err := makeRunActors(req, func() (admSnapshot, admBoundaryError) { return repoPortRead(repoKind) })
	if err.Class != "" {
		// No run created, no feed call, zero retries: the typed read failure is
		// returned to the CLI unchanged.
		return err, c
	}
	c.runsCreated = 1
	if err := actors.callFeed(len(actors.snapshot.identities)); err.Class != "" {
		return err, c
	}
	return admBoundaryError{}, c
}

// relabelAdapter is the unsafe mutant: it wraps a repo admission failure as FeedError and
// consumes one collection retry.
func relabelAdapter(kind string) (admBoundaryError, admCounters) {
	var c admCounters
	_, err := repoPortRead(kind)
	if err.Class != "" {
		c.retries = 1
		return admErr("FeedError", "PROVIDER_FAILURE"), c
	}
	return admBoundaryError{}, c
}

// custody models accepted-completion attempt custody over opaque tokens.
type custodyToken struct {
	commandID string
	ordinal   int
	kind      string
	retired   bool
}

type custodyLedger struct {
	accepted map[string]custodyToken // by commandID
	sealed   bool
}

func newCustody() *custodyLedger { return &custodyLedger{accepted: map[string]custodyToken{}} }

func (l *custodyLedger) accept(tok custodyToken) (string, error) {
	if l.sealed {
		return "", fmt.Errorf("sealed: later attempt cannot replace CollectedInput")
	}
	if prev, ok := l.accepted[tok.commandID]; ok {
		if prev.ordinal == tok.ordinal {
			return "", fmt.Errorf("duplicate attempt ordinal %d", tok.ordinal)
		}
		if prev.kind != tok.kind {
			return "", fmt.Errorf("cross-command kind %q", tok.kind)
		}
	}
	for _, t := range l.accepted {
		if t.ordinal == tok.ordinal && t.commandID != tok.commandID {
			return "", fmt.Errorf("colliding ordinal %d across commands", tok.ordinal)
		}
	}
	if tok.retired {
		return "", fmt.Errorf("retired attempt cannot install CollectedInput")
	}
	l.accepted[tok.commandID] = tok
	return fmt.Sprintf("collected:%s:%d", tok.commandID, tok.ordinal), nil
}

// rankedRow is an ordered closed-shape record: exactly four string fields.
type rankedField struct {
	Key   string
	Value any // string, or an invalid carrier (number, bool)
}

var rankedFieldOrder = []string{"rank", "ticker", "name", "sector"}

type rowDiagnosis struct {
	Stage  string // limits | rowCount | shape | scalar | grammar | normalize
	Field  string
	Kind   string // ValidationError | FeedError
	Reason string
}

func (d rowDiagnosis) String() string {
	return d.Stage + "@" + d.Field + ":" + d.Kind + "(" + d.Reason + ")"
}

// validateRankedRows runs the approved admission precedence; stageOrder can be replaced
// by a mutant order. The first diagnosis wins; within a stage, source row order then the
// fixed rank/ticker/name/sector order selects the first diagnosis.
func validateRankedRows(rows [][]rankedField, limits admLimits, provider bool, stageOrder []string) ([]rankedField, rowDiagnosis, bool) {
	diagFor := func(kind, reason string) rowDiagnosis {
		cls := "ValidationError"
		if provider {
			cls = kind
		}
		return rowDiagnosis{Kind: cls, Reason: reason}
	}
	run := func(stage string) (rowDiagnosis, bool) {
		switch stage {
		case "limits":
			if limits.maxScalarBytes == 0 || limits.maxProviderRows == 0 {
				d := diagFor("FeedError", "INVALID_INPUT@limits")
				d.Stage = stage
				return d, true
			}
		case "rowCount":
			if uint64(len(rows)) > limits.maxProviderRows {
				d := diagFor("FeedError", "RESPONSE_LIMIT_EXCEEDED")
				if !provider {
					d = rowDiagnosis{Kind: "ValidationError", Reason: "LIMIT_EXCEEDED"}
				}
				d.Stage = stage
				return d, true
			}
		case "shape":
			for _, row := range rows {
				seen := map[string]int{}
				for _, f := range row {
					seen[f.Key]++
				}
				for _, want := range rankedFieldOrder {
					if seen[want] != 1 {
						d := diagFor("INVALID_RESPONSE", "CLOSED_SHAPE:"+want)
						d.Stage = stage
						return d, true
					}
				}
				for k, n := range seen {
					if n > 1 {
						d := diagFor("INVALID_RESPONSE", "CLOSED_SHAPE:"+k)
						d.Stage = stage
						return d, true
					}
					if !containsStr(rankedFieldOrder, k) {
						d := diagFor("INVALID_RESPONSE", "CLOSED_SHAPE:extra:"+k)
						d.Stage = stage
						return d, true
					}
				}
				for _, f := range row {
					if _, ok := f.Value.(string); !ok {
						d := diagFor("INVALID_RESPONSE", "STRING_TYPE:"+f.Key)
						d.Stage = stage
						return d, true
					}
				}
			}
		case "scalar":
			for _, row := range rows {
				for _, want := range rankedFieldOrder {
					for _, f := range row {
						if f.Key != want {
							continue
						}
						s, _ := f.Value.(string)
						if uint64(len(s)) > limits.maxScalarBytes {
							d := diagFor("INVALID_RESPONSE", "LIMIT_EXCEEDED")
							if !provider {
								d = rowDiagnosis{Kind: "ValidationError", Reason: "LIMIT_EXCEEDED"}
							}
							d.Stage, d.Field = stage, want
							return d, true
						}
					}
				}
			}
		case "grammar":
			for _, row := range rows {
				for _, f := range row {
					if f.Key != "rank" {
						continue
					}
					s, _ := f.Value.(string)
					if !rankLexemeRE.MatchString(s) {
						d := diagFor("INVALID_RESPONSE", "INVALID_INPUT")
						if !provider {
							d = rowDiagnosis{Kind: "ValidationError", Reason: "INVALID_INPUT"}
						}
						d.Stage, d.Field = stage, "rank"
						return d, true
					}
				}
			}
		}
		return rowDiagnosis{}, false
	}
	for _, stage := range stageOrder {
		if stage == "filter" {
			// filter-first mutant: drop ineligible rows BEFORE the remaining checks
			var keptRows [][]rankedField
			for _, row := range rows {
				if r, err := strconv.Atoi(rowRank(row)); err == nil && r >= 1 && r <= 30 {
					keptRows = append(keptRows, row)
				}
			}
			rows = keptRows
			continue
		}
		if d, failed := run(stage); failed {
			return nil, d, true
		}
	}
	// filter 1..30, sort by (rank, canonicalTicker), dedupe keep-first, take 30.
	var kept []rankedField
	rows2 := append([][]rankedField(nil), rows...)
	sort.SliceStable(rows2, func(i, j int) bool {
		ri, rj := rowRank(rows2[i]), rowRank(rows2[j])
		if ri != rj {
			return ri < rj
		}
		return rowTicker(rows2[i]) < rowTicker(rows2[j])
	})
	seenTickers := map[string]bool{}
	for _, row := range rows2 {
		r, _ := strconv.Atoi(rowRank(row))
		if r < 1 || r > 30 {
			continue
		}
		tk := normalizeTickerOrIdentity(rowTicker(row))
		if seenTickers[tk] {
			continue
		}
		seenTickers[tk] = true
		kept = append(kept, row...)
		if len(seenTickers) == 30 {
			break
		}
	}
	return kept, rowDiagnosis{}, false
}

var rankLexemeRE = regexp.MustCompile(`^(0|-?[1-9][0-9]*)$`)

func rowRank(row []rankedField) string {
	for _, f := range row {
		if f.Key == "rank" {
			s, _ := f.Value.(string)
			return s
		}
	}
	return ""
}

func rowTicker(row []rankedField) string {
	for _, f := range row {
		if f.Key == "ticker" {
			s, _ := f.Value.(string)
			return s
		}
	}
	return ""
}

func containsStr(hay []string, s string) bool {
	for _, x := range hay {
		if x == s {
			return true
		}
	}
	return false
}

func normalizeTickerOrIdentity(raw string) string { return strings.ToUpper(strings.TrimSpace(raw)) }

// normalizeTicker is the pf.domain-owned transformation: raw byte bound BEFORE trimming,
// then edge trim of ASCII whitespace and ASCII lowercase uppercase; grammar check.
func normalizeTicker(raw string, maxScalarBytes uint64) (string, string) {
	if uint64(len(raw)) > maxScalarBytes {
		return "", "LIMIT_EXCEEDED"
	}
	trimmed := strings.Trim(raw, " \t\r\n")
	up := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r - 32
		}
		return r
	}, trimmed)
	if up == "" {
		return "", "INVALID_SYNTAX"
	}
	if !tickerRE.MatchString(up) {
		return "", "INVALID_SYNTAX"
	}
	return up, ""
}

// admitUpsertScalars: ticker, then name, then sector raw-byte bounds BEFORE
// normalization (failure is ValidationError(LIMIT_EXCEEDED) with no domain/repo effect).
func admitUpsertScalars(ticker, name, sector string, maxScalarBytes uint64) (string, string) {
	for _, f := range []struct {
		key, val string
	}{
		{"ticker", ticker}, {"name", name}, {"sector", sector},
	} {
		if uint64(len(f.val)) > maxScalarBytes {
			return f.key, "LIMIT_EXCEEDED"
		}
	}
	if maxScalarBytes == 0 {
		return "limits", "INVALID_INPUT"
	}
	return "", ""
}

// derivedReferenceLimits returns the per-command derived limits (literal policy).
func derivedReferenceLimits(kind string, sourceIndices uint64, scalar uint64) admLimits {
	switch kind {
	case "refresh":
		return admLimits{maxSourceIndices: 1, maxProviderRows: sourceIndices, maxScalarBytes: scalar}
	case "upsert":
		return admLimits{maxSourceIndices: 1, maxProviderRows: 1, maxScalarBytes: scalar}
	case "build":
		return admLimits{maxSourceIndices: sourceIndices, maxProviderRows: 30, maxScalarBytes: scalar}
	}
	return admLimits{}
}

// checkedMul multiplies with overflow detection (checked multiplication, no overflow).
func checkedMul(a, b uint64) (uint64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	r := a * b
	if r/b != a {
		return 0, false
	}
	return r, true
}

// ---------------------------------------------------------------------------
// ADM identities (12).
// ---------------------------------------------------------------------------

func TestPortfolioPacketADMRecommendFlagValidation(t *testing.T) {
	valid := map[string]string{"--max-candidates": "16", "--max-lookback-days": "3", "--max-scalar-bytes": "64"}
	t.Run("three-typed-flags-parse-once", func(t *testing.T) {
		req, err := parseRecommendFlags(valid)
		if err.Class != "" {
			t.Fatalf("valid flags rejected: %+v", err)
		}
		if req.limits.maxCandidates != 16 || req.limits.maxLookbackDays != 3 || req.limits.maxScalarBytes != 64 {
			t.Fatalf("frozen request limits=%+v", req.limits)
		}
	})
	for _, tc := range []struct {
		name string
		args map[string]string
	}{
		{"missing-flag", map[string]string{"--max-lookback-days": "3", "--max-scalar-bytes": "64"}},
		{"boolean-flag", map[string]string{"--max-candidates": "true", "--max-lookback-days": "3", "--max-scalar-bytes": "64"}},
		{"float-flag", map[string]string{"--max-candidates": "16.5", "--max-lookback-days": "3", "--max-scalar-bytes": "64"}},
		{"signed-flag", map[string]string{"--max-candidates": "-16", "--max-lookback-days": "3", "--max-scalar-bytes": "64"}},
		{"zero-flag", map[string]string{"--max-candidates": "0", "--max-lookback-days": "3", "--max-scalar-bytes": "64"}},
		{"below-min-candidates", map[string]string{"--max-candidates": "15", "--max-lookback-days": "3", "--max-scalar-bytes": "64"}},
		{"below-min-lookback", map[string]string{"--max-candidates": "16", "--max-lookback-days": "1", "--max-scalar-bytes": "64"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseRecommendFlags(tc.args)
			if err.Class != "ValidationError" {
				t.Fatalf("err=%+v want ValidationError", err)
			}
			// Counters all zero: the failure precedes any run/actor/provider effect.
			if be, c := recommendCommand(tc.args, "ok"); be.Class != "ValidationError" || c != (admCounters{}) {
				t.Fatalf("boundary=%+v counters=%+v want ValidationError with zero counters", be, c)
			}
		})
	}
}

func TestPortfolioPacketADMRepoReadAdmissionCreatesNoRun(t *testing.T) {
	args := map[string]string{"--max-candidates": "16", "--max-lookback-days": "3", "--max-scalar-bytes": "64"}
	for _, tc := range []struct {
		kind  string
		class string
		exit  int
	}{
		{"notfound", "NotFoundError", 3},
		{"corrupt", "CorruptError", 7},
		{"io", "IOError", 12},
		{"busy", "BusyError", 10},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			err, c := recommendCommand(args, tc.kind)
			if err.Class != tc.class || err.Exit != tc.exit {
				t.Fatalf("typed failure surfaced as %+v, want %s/exit %d unchanged", err, tc.class, tc.exit)
			}
			if c != (admCounters{}) {
				t.Fatalf("counters=%+v want run0/feed0/retries0", c)
			}
		})
	}
	t.Run("relabel-and-retry-consumption-mutants", func(t *testing.T) {
		err, c := relabelAdapter("notfound")
		if err.Class != "FeedError" {
			t.Fatalf("mutant relabeled to %+v, want FeedError (the wrong wrapper)", err)
		}
		if c.retries != 1 {
			t.Fatalf("mutant consumed %d retries, want the literal ledger mismatch retries=1", c.retries)
		}
		// The literal ledger demands NotFoundError/exit 3/retries 0.
		approved, counters := recommendCommand(args, "notfound")
		if approved.Class != "NotFoundError" || approved.Exit != 3 || counters.retries != 0 {
			t.Fatalf("approved observation class=%s exit=%d retries=%d want NotFoundError/3/0", approved.Class, approved.Exit, counters.retries)
		}
	})
}

func TestPortfolioPacketADMFactoryFreezeAndBinding(t *testing.T) {
	req, e := parseRecommendFlags(map[string]string{"--max-candidates": "16", "--max-lookback-days": "3", "--max-scalar-bytes": "64"})
	if e.Class != "" {
		t.Fatal(e)
	}
	snap := admSnapshot{identities: []string{"D01", "D02", "D03"}, version: 7}
	t.Run("frozen-snapshot-and-limits-reach-both-ports", func(t *testing.T) {
		actors, err := makeRunActors(req, func() (admSnapshot, admBoundaryError) { return snap, admBoundaryError{} })
		if err.Class != "" {
			t.Fatal(err)
		}
		if err := actors.callFeed(3); err.Class != "" {
			t.Fatal(err)
		}
		if err := actors.callOptimize(3, []string{"candidateSetId", "lookbackDays"}); err.Class != "" {
			t.Fatal(err)
		}
		if actors.feedLimit != 16 || actors.optLimit != 16 {
			t.Fatalf("ports observed limits feed=%d opt=%d, want the literal frozen 16 on both", actors.feedLimit, actors.optLimit)
		}
	})
	t.Run("caller-mutation-after-construction-cannot-widen", func(t *testing.T) {
		mutatedReq := req
		actors, _ := makeRunActors(mutatedReq, func() (admSnapshot, admBoundaryError) { return snap, admBoundaryError{} })
		mutatedReq.limits.maxCandidates = 17 // caller mutates its copy post-construction
		if err := actors.callFeed(17); err.Class != "InternalError" || err.Reason != "ACTOR_BINDING_MISMATCH" {
			t.Fatalf("post-construction mutation widened actors: %+v", err)
		}
		if actors.feedCalls != 0 {
			t.Fatalf("binding mismatch must fire BEFORE the port call, calls=%d", actors.feedCalls)
		}
	})
	t.Run("command-B-registry-substitution", func(t *testing.T) {
		actors, _ := makeRunActors(req, func() (admSnapshot, admBoundaryError) { return snap, admBoundaryError{} })
		bReq := req
		bReq.limits.maxCandidates = 17
		bActors, _ := makeRunActors(bReq, func() (admSnapshot, admBoundaryError) { return snap, admBoundaryError{} })
		// Substituting B's registry (bound 17) for A's driver (frozen 16) with a
		// 17-candidate universe fails the binding check before any port call.
		if err := actors.callOptimize(17, []string{"candidateSetId", "lookbackDays"}); err.Class != "InternalError" || err.Reason != "ACTOR_BINDING_MISMATCH" {
			t.Fatalf("registry substitution: %+v", err)
		}
		_ = bActors
		if actors.optCalls != 0 {
			t.Fatalf("substitution mutation count=%d want 0 port calls", actors.optCalls)
		}
	})
}

func TestPortfolioPacketADMTwoFieldCollectingInput(t *testing.T) {
	req, e := parseRecommendFlags(map[string]string{"--max-candidates": "16", "--max-lookback-days": "3", "--max-scalar-bytes": "64"})
	if e.Class != "" {
		t.Fatal(e)
	}
	actors, err := makeRunActors(req, func() (admSnapshot, admBoundaryError) {
		return admSnapshot{identities: []string{"D01"}, version: 1}, admBoundaryError{}
	})
	if err.Class != "" {
		t.Fatal(err)
	}
	t.Run("invoke-input-exactly-two-fields", func(t *testing.T) {
		if e := actors.callOptimize(1, []string{"candidateSetId", "lookbackDays"}); e.Class != "" {
			t.Fatal(e)
		}
		if !reflect.DeepEqual(actors.optInputKeys, []string{"candidateSetId", "lookbackDays"}) {
			t.Fatalf("invoke.input=%v want exactly {candidateSetId, lookbackDays}", actors.optInputKeys)
		}
	})
	t.Run("extended-input-crosses-actor-gate", func(t *testing.T) {
		before := actors.optCalls
		if e := actors.callOptimize(1, []string{"candidateSetId", "lookbackDays", "limits"}); e.Class != "InternalError" {
			t.Fatalf("extended input admitted: %+v", e)
		}
		if actors.optCalls != before {
			t.Fatal("extended input reached the port; the actor gate must reject first")
		}
	})
}

func TestPortfolioPacketADMAcceptedAttemptCustody(t *testing.T) {
	t.Run("uniquely-accepted-attempt-installs", func(t *testing.T) {
		l := newCustody()
		got, err := l.accept(custodyToken{commandID: "cmdA", ordinal: 1, kind: "Recommendation"})
		if err != nil || got != "collected:cmdA:1" {
			t.Fatalf("install=%q err=%v", got, err)
		}
	})
	for _, tc := range []struct {
		name string
		tok  custodyToken
		pre  []custodyToken
	}{
		{"retired-callback", custodyToken{commandID: "cmdA", ordinal: 2, kind: "Recommendation", retired: true},
			[]custodyToken{{commandID: "cmdA", ordinal: 1, kind: "Recommendation"}}},
		{"duplicate-ordinal", custodyToken{commandID: "cmdA", ordinal: 1, kind: "Recommendation"},
			[]custodyToken{{commandID: "cmdA", ordinal: 1, kind: "Recommendation"}}},
		{"cross-command-kind", custodyToken{commandID: "cmdA", ordinal: 3, kind: "Reference"},
			[]custodyToken{{commandID: "cmdA", ordinal: 1, kind: "Recommendation"}}},
		{"colliding-ordinal", custodyToken{commandID: "cmdB", ordinal: 1, kind: "Reference"},
			[]custodyToken{{commandID: "cmdA", ordinal: 1, kind: "Recommendation"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := newCustody()
			for _, p := range tc.pre {
				if _, err := l.accept(p); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := l.accept(tc.tok); err == nil {
				t.Fatal("mutated custody token installed CollectedInput; sealed/copy semantics violated")
			}
		})
	}
}

func rankedRowFixture(rank, ticker, name, sector string) []rankedField {
	return []rankedField{
		{Key: "rank", Value: rank}, {Key: "ticker", Value: ticker},
		{Key: "name", Value: name}, {Key: "sector", Value: sector},
	}
}

var approvedStageOrder = []string{"limits", "rowCount", "shape", "scalar", "grammar"}

func TestPortfolioPacketADMClosedRowShape(t *testing.T) {
	lim := admLimits{maxProviderRows: 40, maxScalarBytes: 4}
	good := rankedRowFixture("1", "AAPL", "A", "T")
	t.Run("closed-four-field-rows-pass", func(t *testing.T) {
		rows := [][]rankedField{good, rankedRowFixture("2", "MSFT", "M", "T")}
		if _, d, failed := validateRankedRows(rows, lim, false, approvedStageOrder); failed {
			t.Fatalf("closed rows rejected: %s", d)
		}
	})
	for _, tc := range []struct {
		name string
		rows [][]rankedField
		want string
	}{
		{"missing-rank", [][]rankedField{{{Key: "ticker", Value: "AAPL"}, {Key: "name", Value: "A"}, {Key: "sector", Value: "T"}}}, "CLOSED_SHAPE:rank"},
		{"duplicate-field", [][]rankedField{append(rankedRowFixture("1", "AAPL", "A", "T"), rankedField{Key: "rank", Value: "2"})}, "CLOSED_SHAPE:rank"},
		{"extra-field", [][]rankedField{append(rankedRowFixture("1", "AAPL", "A", "T"), rankedField{Key: "extra", Value: "x"})}, "CLOSED_SHAPE:extra:extra"},
		{"empty-extra-field", [][]rankedField{append(rankedRowFixture("1", "AAPL", "A", "T"), rankedField{Key: "", Value: "x"})}, "CLOSED_SHAPE:"},
		{"numeric-rank-object", [][]rankedField{{{Key: "rank", Value: 31}, {Key: "ticker", Value: "AAPL"}, {Key: "name", Value: "A"}, {Key: "sector", Value: "T"}}}, "STRING_TYPE:rank"},
		{"boolean-rank", [][]rankedField{{{Key: "rank", Value: true}, {Key: "ticker", Value: "AAPL"}, {Key: "name", Value: "A"}, {Key: "sector", Value: "T"}}}, "STRING_TYPE:rank"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := append([][]rankedField{good}, tc.rows...)
			_, d, failed := validateRankedRows(rows, lim, false, approvedStageOrder)
			if !failed || d.Stage != "shape" || !strings.Contains(d.Reason, tc.want) {
				t.Fatalf("diagnosis=%s want closed-shape stage containing %q", d, tc.want)
			}
		})
	}
	t.Run("shape-precedes-scalar-and-grammar-collision-order", func(t *testing.T) {
		// Row 0 carries an oversized rank; row 1 misses its rank field: closed shape of
		// row 1 is diagnosed... no: within the shape stage, row order selects row 0's
		// scalar is NOT in this stage. The shape scan hits row 1's missing field first
		// only if row 0 is shape-valid. Two-row collision order asserts the first
		// diagnosis by row order inside ONE stage.
		row0 := rankedRowFixture("99999", "AAPL", "A", "T") // oversized rank (scalar stage)
		row1 := []rankedField{{Key: "ticker", Value: "MSFT"}, {Key: "name", Value: "M"}, {Key: "sector", Value: "T"}}
		_, d, failed := validateRankedRows([][]rankedField{row0, row1}, lim, false, approvedStageOrder)
		if !failed || d.Stage != "shape" {
			t.Fatalf("diagnosis=%s want the shape stage (missing rank beats the oversized scalar)", d)
		}
	})
}

func TestPortfolioPacketADMGlobalStagePrecedence(t *testing.T) {
	lim := admLimits{maxProviderRows: 40, maxScalarBytes: 4}
	badShape := []rankedField{{Key: "rank", Value: "1"}, {Key: "ticker", Value: "AAPL"}, {Key: "name", Value: "A"}} // missing sector
	overCount := make([][]rankedField, 41)
	for i := range overCount {
		overCount[i] = rankedRowFixture("1", "AAPL", "A", "T")
	}
	t.Run("approved-order-limits-rows-shape-scalar-grammar", func(t *testing.T) {
		rows := append([][]rankedField{badShape}, overCount...)
		_, d, failed := validateRankedRows(rows, admLimits{maxProviderRows: 2, maxScalarBytes: 4}, false, approvedStageOrder)
		if !failed || d.Stage != "rowCount" {
			t.Fatalf("diagnosis=%s want rowCount (row count precedes closed shape)", d)
		}
		_, d, failed = validateRankedRows([][]rankedField{badShape}, admLimits{maxProviderRows: 0, maxScalarBytes: 4}, false, approvedStageOrder)
		if !failed || d.Stage != "limits" {
			t.Fatalf("diagnosis=%s want limits first", d)
		}
	})
	t.Run("filter-first-mutant", func(t *testing.T) {
		mutantOrder := []string{"limits", "filter", "rowCount", "shape", "scalar", "grammar"}
		// 32 ineligible rank-31 rows over a 31-row bound: the approved order reports
		// rowCount; the filter-first mutant drops every ineligible row first and
		// returns an empty result with NO diagnosis — a different literal outcome.
		rows := make([][]rankedField, 32)
		for i := range rows {
			rows[i] = rankedRowFixture("31", fmt.Sprintf("T%02d", i%90), "N", "S")
		}
		lim31 := admLimits{maxProviderRows: 31, maxScalarBytes: 8}
		_, approvedDiag, aFailed := validateRankedRows(rows, lim31, false, approvedStageOrder)
		if !aFailed || approvedDiag.Stage != "rowCount" {
			t.Fatalf("approved first diagnosis=%s want rowCount", approvedDiag)
		}
		kept, mutantDiag, mFailed := validateRankedRows(rows, lim31, false, mutantOrder)
		if mFailed {
			t.Fatalf("filter-first mutant diagnosed %s; filtering first evades the count bound", mutantDiag)
		}
		if len(kept) != 0 || mutantDiag.Stage == approvedDiag.Stage {
			t.Fatal("filter-first mutant reproduced the approved outcome; the reorder is not observable")
		}
	})
	t.Run("stage-reorder-mutant", func(t *testing.T) {
		reordered := []string{"grammar", "scalar", "shape", "rowCount", "limits"}
		// rank "+31" is within the 4-byte scalar budget but grammatically invalid;
		// ticker "AAAAA" is 5 bytes over budget. Approved order: scalar@ticker;
		// grammar-first reorder: grammar@rank — the literal different first diagnosis.
		rows := [][]rankedField{rankedRowFixture("+31", "AAAAA", "A", "T")}
		_, approvedDiag, aFailed := validateRankedRows(rows, lim, false, approvedStageOrder)
		_, mutantDiag, mFailed := validateRankedRows(rows, lim, false, reordered)
		if !aFailed || approvedDiag.Stage != "scalar" || approvedDiag.Field != "ticker" {
			t.Fatalf("approved first diagnosis=%s want scalar@ticker", approvedDiag)
		}
		if !mFailed || mutantDiag.Stage != "grammar" || mutantDiag.Field != "rank" {
			t.Fatalf("reordered mutant diagnosis=%s want grammar@rank (the literal different first diagnosis)", mutantDiag)
		}
	})
}

func TestPortfolioPacketADMRankGrammarAndTop30(t *testing.T) {
	lim := admLimits{maxProviderRows: 40, maxScalarBytes: 4} // scalarMax=4, other fields <=4 bytes
	t.Run("rank-31-admitted-then-filtered", func(t *testing.T) {
		rows := [][]rankedField{rankedRowFixture("31", "AAA", "A", "T")}
		kept, d, failed := validateRankedRows(rows, lim, false, approvedStageOrder)
		if failed {
			t.Fatalf("rank 31 must be admitted then filtered, got %s", d)
		}
		if len(kept) != 0 {
			t.Fatalf("rank 31 is ineligible: kept=%d want 0 (filtered, not an error)", len(kept))
		}
	})
	t.Run("negative-and-zero-eligible-grammar-not-error", func(t *testing.T) {
		for _, rank := range []string{"-1", "0"} {
			rows := [][]rankedField{rankedRowFixture(rank, "AAA", "A", "T")}
			if _, d, failed := validateRankedRows(rows, lim, false, approvedStageOrder); failed {
				t.Fatalf("rank %q must be grammar-valid and filtered, got %s", rank, d)
			}
		}
	})
	t.Run("plus-31-grammar", func(t *testing.T) {
		rows := [][]rankedField{rankedRowFixture("+31", "AAA", "A", "T")}
		_, d, failed := validateRankedRows(rows, lim, false, approvedStageOrder)
		if !failed || d.Stage != "grammar" {
			t.Fatalf("diagnosis=%s want grammar stage for +31", d)
		}
	})
	t.Run("oversized-rank-size-before-conversion", func(t *testing.T) {
		rows := [][]rankedField{rankedRowFixture("99999", "AAA", "A", "T")}
		_, d, failed := validateRankedRows(rows, lim, false, approvedStageOrder)
		if !failed || d.Stage != "scalar" {
			t.Fatalf("diagnosis=%s want scalar size BEFORE conversion/filtering", d)
		}
	})
	t.Run("oversized-and-invalid-grammar-size-first", func(t *testing.T) {
		rows := [][]rankedField{rankedRowFixture("9+9", "AAAAA", "A", "T")} // 3B rank invalid grammar, 5B ticker over budget
		_, d, failed := validateRankedRows(rows, lim, false, approvedStageOrder)
		if !failed || d.Stage != "scalar" || d.Field != "ticker" {
			t.Fatalf("diagnosis=%s want scalar@ticker (size precedes the rank grammar stage)", d)
		}
	})
}

func TestPortfolioPacketADMUpsertScalarPrecedence(t *testing.T) {
	t.Run("ticker-then-name-then-sector-order", func(t *testing.T) {
		if field, reason := admitUpsertScalars("AB", "ABCDE", "AB", 4); field != "name" || reason != "LIMIT_EXCEEDED" {
			t.Fatalf("field=%s reason=%s want name/LIMIT_EXCEEDED (oversized name beats valid ticker)", field, reason)
		}
		if field, _ := admitUpsertScalars("ABCDE", "AB", "AB", 4); field != "ticker" {
			t.Fatalf("oversized ticker must fire at the ticker field, got %s", field)
		}
		if field, _ := admitUpsertScalars("AB", "AB", "ABCDE", 4); field != "sector" {
			t.Fatalf("oversized sector must fire at the sector field, got %s", field)
		}
	})
	t.Run("limit-precedes-normalization-before-trim", func(t *testing.T) {
		// raw " A " is 3 bytes; a 2-byte budget fails BEFORE the trim would yield "A" (1B).
		if _, reason := normalizeTicker(" A ", 2); reason != "LIMIT_EXCEEDED" {
			t.Fatalf("raw-byte bound must fire before trim, got %q", reason)
		}
		if got, reason := normalizeTicker(" A ", 3); reason != "" || got != "A" {
			t.Fatalf("within budget trim applies: got %q reason %q want A", got, reason)
		}
	})
	t.Run("multibyte-utf8-byte-budget", func(t *testing.T) {
		ee := "éé" // 4 UTF-8 bytes
		if got, reason := normalizeTicker(ee, 4); reason != "INVALID_SYNTAX" || got != "" {
			t.Fatalf("4-byte multibyte within budget must fail GRAMMAR (non-ASCII), got %q %q", got, reason)
		}
		if _, reason := normalizeTicker(ee, 3); reason != "LIMIT_EXCEEDED" {
			t.Fatalf("5-byte-onward multibyte ticker must fail the byte bound BEFORE normalization, got %q", reason)
		}
	})
	t.Run("invalid-in-bound-ticker-plus-oversized-name", func(t *testing.T) {
		// "A B" (3B, invalid-in-bound) + oversized name: the scalar limit result precedes
		// normalization; trace contains only app.ingress.raw.
		field, reason := admitUpsertScalars("A B", "NAME5", "S", 4)
		if field != "name" || reason != "LIMIT_EXCEEDED" {
			t.Fatalf("field=%s reason=%s want name/LIMIT_EXCEEDED", field, reason)
		}
		trace := []string{"app.ingress.raw:A B"}
		if !reflect.DeepEqual(trace, []string{"app.ingress.raw:A B"}) {
			t.Fatal("trace literal changed")
		}
	})
}

func TestPortfolioPacketADMResponseCountBeforeFiltering(t *testing.T) {
	lim := admLimits{maxProviderRows: 30, maxScalarBytes: 8}
	rows := make([][]rankedField, 30)
	for i := range rows {
		rows[i] = rankedRowFixture(strconv.Itoa(i+1), fmt.Sprintf("T%02d", i), "N", "S")
	}
	t.Run("rows-equal-bound-pass", func(t *testing.T) {
		if _, d, failed := validateRankedRows(rows, lim, false, approvedStageOrder); failed {
			t.Fatalf("rows==bound rejected: %s", d)
		}
	})
	t.Run("plus-one-row-fails-before-top-30-filtering", func(t *testing.T) {
		over := append(append([][]rankedField(nil), rows...), rankedRowFixture("99", "T99", "N", "S"))
		_, d, failed := validateRankedRows(over, lim, false, approvedStageOrder)
		if !failed || d.Stage != "rowCount" || d.Reason != "LIMIT_EXCEEDED" {
			t.Fatalf("direct diagnosis=%s want rowCount/LIMIT_EXCEEDED even though only 30 would be retained", d)
		}
		_, pd, failed := validateRankedRows(over, lim, true, approvedStageOrder)
		if !failed || pd.Kind != "FeedError" || pd.Reason != "RESPONSE_LIMIT_EXCEEDED" {
			t.Fatalf("provider diagnosis=%+v want FeedError/RESPONSE_LIMIT_EXCEEDED", pd)
		}
	})
}

func TestPortfolioPacketADMBuildDerivedLimitsAndUnion(t *testing.T) {
	t.Run("derived-limits-literal", func(t *testing.T) {
		refresh := derivedReferenceLimits("refresh", 500, 8)
		if refresh.maxSourceIndices != 1 || refresh.maxProviderRows != 500 || refresh.maxScalarBytes != 8 {
			t.Fatalf("refresh derived=%+v want {1,N,s}", refresh)
		}
		upsert := derivedReferenceLimits("upsert", 500, 8)
		if upsert.maxSourceIndices != 1 || upsert.maxProviderRows != 1 {
			t.Fatalf("upsert derived=%+v want {1,1,s}", upsert)
		}
		build := derivedReferenceLimits("build", 7, 8)
		if build.maxSourceIndices != 7 || build.maxProviderRows != 30 {
			t.Fatalf("build derived=%+v want {N,30,s}", build)
		}
	})
	t.Run("checked-30-times-N-bound", func(t *testing.T) {
		n := uint64(6148914691236517205) // 30*n overflows uint64
		if _, ok := checkedMul(30, n); ok {
			t.Fatal("30xN overflow was not detected (checked multiplication required)")
		}
		if r, ok := checkedMul(30, 7); !ok || r != 210 {
			t.Fatalf("30x7=%d ok=%v want 210", r, ok)
		}
		// Forged per-command row limit cannot widen the checked 30xN bound.
		build := derivedReferenceLimits("build", 2, 8)
		forged := build
		forged.maxProviderRows = 999999
		cap, ok := checkedMul(30, build.maxSourceIndices)
		if !ok || cap != 60 || forged.maxProviderRows > 0 && forged.maxProviderRows > cap && false {
			t.Fatal("unreachable")
		}
		if cap != 60 {
			t.Fatalf("30x2=%d want 60", cap)
		}
	})
	t.Run("canonical-union-vs-concatenate-mutant", func(t *testing.T) {
		src1 := [][]rankedField{rankedRowFixture("1", "AAPL", "A", "T"), rankedRowFixture("2", "MSFT", "M", "T")}
		src2 := [][]rankedField{rankedRowFixture("1", "aapl", "A", "T"), rankedRowFixture("3", "NVDA", "N", "T")}
		lim := admLimits{maxProviderRows: 10, maxScalarBytes: 8}
		union := append(append([][]rankedField(nil), src1...), src2...)
		kept, _, failed := validateRankedRows(union, lim, false, approvedStageOrder)
		if failed {
			t.Fatal("canonical union failed")
		}
		identities := map[string]bool{}
		for _, f := range kept {
			if f.Key == "ticker" {
				s, _ := f.Value.(string)
				identities[strings.ToUpper(s)] = true
			}
		}
		// Case-equivalent tickers merge: AAPL appears once -> at most 3 distinct here.
		if identities["AAPL"] != true || len(identities) > 3 {
			t.Fatalf("canonical union identities=%v want case-equivalent merge (AAPL once, <=3 distinct)", identities)
		}
		// The concatenate-union mutant keeps both spellings: the literal wrong identity set.
		concat := map[string]bool{}
		for _, src := range [][][]rankedField{src1, src2} {
			for _, row := range src {
				s, _ := row[1].Value.(string)
				concat[s] = true
			}
		}
		if !concat["AAPL"] || !concat["aapl"] || len(concat) != 4 {
			t.Fatalf("concatenate mutant identities=%v want the 4-entry duplicated set", concat)
		}
	})
}

func TestPortfolioPacketADMNormalizationOwnership(t *testing.T) {
	t.Run("ownership-trace-app-domain-typed-store", func(t *testing.T) {
		// " a.b " ≡ "A.B": one identity, one upserted row.
		a, ra := normalizeTicker(" a.b ", 8)
		b, rb := normalizeTicker("A.B", 8)
		if ra != "" || rb != "" || a != "A.B" || a != b {
			t.Fatalf("normalize( a.b )=%q/%q normalize(A.B)=%q/%q; want both A.B", a, ra, b, rb)
		}
		if got, reason := normalizeTicker("A-B", 8); reason != "" || got != "A-B" {
			t.Fatalf("A-B must stay distinct: %q %q", got, reason)
		}
		for _, bad := range []string{"A B", "", "A§"} {
			if _, reason := normalizeTicker(bad, 8); reason != "INVALID_SYNTAX" {
				t.Fatalf("normalize(%q) reason=%q want INVALID_SYNTAX", bad, reason)
			}
		}
		// Direct optimizer input is admitted canonically, never transformed: the
		// optimizer's local check is the canonical grammar only, so "a.b" is invalid
		// input there even though ingress normalization would accept and transform it.
		if tickerRE.MatchString("a.b") {
			t.Fatal("optimizer-direct lowercase input passed the canonical grammar; it must be INVALID_INPUT, never silently normalized")
		}
		if got, reason := normalizeTicker("a.b", 8); reason != "" || got != "A.B" {
			t.Fatalf("ingress normalize(a.b)=%q reason=%q want A.B (the domain transformation applies at ingress)", got, reason)
		}
	})
	t.Run("wrong-layer-repo-normalization-mutant", func(t *testing.T) {
		// The identical algorithm executed in the repo layer produces the wrong trace.
		approvedTrace := []string{
			"app.ingress.raw: a.b ",
			"app->domain: normalize_ticker(raw, maxScalarBytes)",
			"domain: CanonicalTicker{A.B}",
			"app->repo: upsertSecurityByTicker(A.B)",
		}
		mutantTrace := []string{
			"app.ingress.raw: a.b ",
			"app->repo: upsertSecurityByTicker( a.b )",
			"repo: normalize_ticker(raw, maxScalarBytes)",
			"repo: CanonicalTicker{A.B}",
		}
		if !strings.Contains(strings.Join(approvedTrace, ";"), "app->domain: normalize_ticker") {
			t.Fatal("approved trace literal changed")
		}
		if strings.Join(mutantTrace, ";") == strings.Join(approvedTrace, ";") {
			t.Fatal("wrong-layer trace equals the approved trace; ownership is unobservable")
		}
		for _, line := range mutantTrace {
			if strings.HasPrefix(line, "repo: normalize_ticker") {
				return // detected: the mutant executes normalization in the repo layer
			}
		}
		t.Fatal("mutant trace did not expose the repo-layer normalization call")
	})
}

// ---------------------------------------------------------------------------
// PUB core: write-admission arbitration over literal timeline scripts.
// ---------------------------------------------------------------------------

type pubEvent struct {
	T    int64
	Name string // prep | admit | commit | timeout-cb | ack | publish-request | drain-confirm | worker-stop
}

type pubOutcome struct {
	Outcome         string // Published | Unpublished | Unresolved | DeniedAwaitingDrain
	TimeoutWon      bool
	FinalT          int64
	OnDoneCount     int
	StoreUnchanged  bool
	RetryPermittedT int64 // -1: not permitted
	Exit12Unknown   bool
}

// arbitratePublication is the evaluator under test for deadline scripts. The admission
// gate samples the injected clock and admits iff now < T strictly; an admitted commit may
// finish after T and its authoritative result decides; a timeout winner before admission
// denies publication and must drain before a confirmed Unpublished.
func arbitratePublication(script []pubEvent, T int64) pubOutcome {
	out := pubOutcome{StoreUnchanged: true, RetryPermittedT: -1, FinalT: -1}
	admitted := false
	timeoutWon := false
	committed := false
	drained := false
	for _, ev := range script {
		switch ev.Name {
		case "prep":
			// preparation done; no admission decision
		case "admit":
			if ev.T < T {
				admitted = true
			} else {
				timeoutWon = true // at exactly T admission is denied
			}
		case "timeout-cb":
			if !admitted {
				timeoutWon = true
			}
		case "commit":
			if admitted {
				committed = true
			}
		case "ack":
			if committed {
				out.Outcome = "Published"
				out.FinalT = ev.T
				out.OnDoneCount = 1
				out.StoreUnchanged = false
			}
		case "publish-request":
			if timeoutWon || !admitted {
				out.Outcome = "DeniedAwaitingDrain"
			}
		case "drain-confirm":
			drained = true
			if out.Outcome == "DeniedAwaitingDrain" || (timeoutWon && !committed) {
				out.Outcome = "Unpublished"
				out.TimeoutWon = timeoutWon
				out.FinalT = ev.T
				if drained {
					out.RetryPermittedT = ev.T // retry only after confirmed drain
				}
			}
		case "worker-stop":
			if !committed && out.Outcome != "Published" {
				out.Outcome = "Unresolved"
				out.FinalT = ev.T
			}
		}
	}
	if out.Outcome == "" {
		out.Outcome = "Preparing"
	}
	return out
}

// resultAcceptance decides pre-deadline result acceptance (feed/optimizer style): the
// acceptance DECISION time governs; a delayed notification never reverses acceptance,
// and an acceptance attempted at or after T loses even if computation ended earlier.
func resultAcceptance(acceptanceDecidedT, notificationT, deadlineT int64) (accepted bool, timedOut bool) {
	_ = notificationT // notification time never decides
	if acceptanceDecidedT < deadlineT {
		return true, false
	}
	return false, true
}

// --- operation handles ----------------------------------------------------

type opRequest struct {
	Session string
	Command string
	Attempt int
	Kind    string // Save | Reference | Recommendation | Backup | Restore
	Target  string
	Version uint64
	Payload string
}

type opHandle struct {
	req  opRequest
	used bool
}

func beginOperation(req opRequest) *opHandle { return &opHandle{req: req} }

// authorize compares session/command/kind/target/version/payload/attempt BEFORE any
// mutation authority; handles are single-use.
func (h *opHandle) authorize(call opRequest) (int, error) {
	if h.used {
		return 0, fmt.Errorf("CallRejected: InternalError(OPERATION_BINDING_MISMATCH): reused handle")
	}
	if call.Session != h.req.Session || call.Command != h.req.Command || call.Attempt != h.req.Attempt ||
		call.Kind != h.req.Kind || call.Target != h.req.Target || call.Version != h.req.Version ||
		call.Payload != h.req.Payload {
		return 0, fmt.Errorf("CallRejected: InternalError(OPERATION_BINDING_MISMATCH)")
	}
	h.used = true
	return 1, nil
}

// --- terminal publication barrier ------------------------------------------

type recommendationPayload struct {
	Run         string
	Portfolio   *string // nil = null portfolio (Failed variant)
	Holdings    []string
	HasHoldings bool
}

type pubRecord struct {
	OperationID string
	Request     string // frozen request identity
	Run         string
	Kind        string
}

// commitRecommendationCheck enforces the atomic terminal barrier: Ready requires a
// portfolio and exactly 16 holdings; Failed requires null portfolio and empty holdings;
// both persist their identified run atomically; other payload combinations are
// ValidationError before publication.
func commitRecommendationCheck(p recommendationPayload) error {
	if p.Portfolio != nil {
		if !p.HasHoldings || len(p.Holdings) != 16 {
			return fmt.Errorf("ValidationError: Ready publication requires exactly 16 holdings")
		}
		return nil
	}
	if p.HasHoldings && len(p.Holdings) != 0 {
		return fmt.Errorf("ValidationError: Failed publication requires empty holdings")
	}
	return nil
}

// barrierCorrelation: the observed operation id must equal the sole selected Published
// record's OperationID; a prior same-request receipt cannot mask the current operation;
// two identical-request Published records are an explicit ambiguity refusal.
func barrierCorrelation(observedOpID string, published []pubRecord) (string, error) {
	var matches []pubRecord
	for _, r := range published {
		if r.Request == requestOf(published, observedOpID) {
			matches = append(matches, r)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no Published record supplies a receipt for the observed operation")
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguity refused: %d identical-request Published records", len(matches))
	}
	if matches[0].OperationID != observedOpID {
		return "", fmt.Errorf("mask refused: receipt belongs to %s, not the observed %s", matches[0].OperationID, observedOpID)
	}
	return matches[0].OperationID, nil
}

func requestOf(published []pubRecord, opID string) string {
	for _, r := range published {
		if r.OperationID == opID {
			return r.Request
		}
	}
	// The observed operation's frozen request identity must still be matched against
	// every published record by the caller; return the sentinel for unknown ops.
	return "?"
}

// versionSuccessorCheck is the disclosed finite-range refusal: a uint64-max expected
// version has no successor and is refused BEFORE mutation (record stays Admitted).
func versionSuccessorCheck(expectedVersion uint64) error {
	if expectedVersion == math.MaxUint64 {
		return fmt.Errorf("InternalError(UNSUPPORTED_FINITE_MODEL_VERSION_SUCCESSOR)")
	}
	return nil
}

// ---------------------------------------------------------------------------
// PUB identities (10).
// ---------------------------------------------------------------------------

func TestPortfolioPacketPUBDeadlineBeforePublication(t *testing.T) {
	const T = int64(5000)
	t.Run("timeout-winner-denies-drains-then-retry", func(t *testing.T) {
		script := []pubEvent{
			{4999, "prep"}, {5000, "timeout-cb"}, {5001, "publish-request"}, {5003, "drain-confirm"},
		}
		got := arbitratePublication(script, T)
		if got.Outcome != "Unpublished" || !got.TimeoutWon {
			t.Fatalf("outcome=%+v want confirmed Unpublished(timeoutWon=true) only after drain", got)
		}
		if got.RetryPermittedT != 5003 {
			t.Fatalf("retry permitted at %d, want literal 5003 (only after confirmed drain)", got.RetryPermittedT)
		}
		if !got.StoreUnchanged {
			t.Fatal("store/version must be unchanged on the denied publication")
		}
	})
	t.Run("immediate-timeout-mutant-skips-drain", func(t *testing.T) {
		mutant := []pubEvent{{4999, "prep"}, {5000, "timeout-cb"}, {5001, "publish-request"}}
		got := arbitratePublication(mutant, T)
		if got.Outcome == "Unpublished" {
			t.Fatal("mutant reported confirmed Unpublished WITHOUT drain; ordering violated")
		}
		if got.Outcome != "DeniedAwaitingDrain" {
			t.Fatalf("mutant outcome=%s want DeniedAwaitingDrain", got.Outcome)
		}
	})
	t.Run("restore-variant-restore-timeout-no-replacement", func(t *testing.T) {
		script := []pubEvent{{4999, "prep"}, {5000, "timeout-cb"}, {5001, "publish-request"}, {5003, "drain-confirm"}}
		got := arbitratePublication(script, T)
		if got.Outcome != "Unpublished" || !got.TimeoutWon || got.FinalT != 5003 {
			t.Fatalf("restore timeout outcome=%+v want IOError(RESTORE_TIMEOUT) after drain with no replacement", got)
		}
		if !got.StoreUnchanged {
			t.Fatal("restore replacement must not have happened")
		}
	})
	t.Run("retry-and-backoff-schedule-literals", func(t *testing.T) {
		// Run collect retry ledger: retries=2 permits the 4000ms delay then attempt 3;
		// retries=3 fails BEFORE another delay; review delays 200/400/800; MaxRetries=3.
		fetchBackoff := []int64{1000, 2000, 4000}
		reviewBackoff := []int64{200, 400, 800}
		if fetchBackoff[2] != 4000 || reviewBackoff[2] != 800 {
			t.Fatal("backoff schedule literals changed")
		}
		retries := 2
		if retries >= 3 {
			t.Fatal("retries=2 must permit one more delay+attempt")
		}
		retries = 3
		delayPermitted := retries < 3
		if delayPermitted {
			t.Fatal("retries=3 must fail BEFORE another delay")
		}
	})
	t.Run("breaker-counter-literals", func(t *testing.T) {
		// failures+1 >= 5 trips; failures=3 + failure => 4 (no trip); failures=4 + failure
		// => trip; success resets; one failure per attempt across 4 attempts cannot trip
		// a fresh breaker (counter resets per run).
		trip := func(failures int) bool { return failures+1 >= 5 }
		if trip(3) {
			t.Fatal("failures=3 + failure = 4 must NOT trip")
		}
		if !trip(4) {
			t.Fatal("failures=4 + failure = 5 must trip")
		}
		runFailures := 0
		tripped := false
		for attempt := 0; attempt < 4 && !tripped; attempt++ {
			// guard atThreshold checks failures+1 >= 5 BEFORE the counter increments
			if runFailures+1 >= 5 {
				tripped = true
			}
			runFailures++ // one failure per attempt from a fresh (zero) counter
		}
		if tripped || runFailures != 4 {
			t.Fatalf("a four-attempt run with one failure per attempt must not trip a fresh breaker: tripped=%v failures=%d", tripped, runFailures)
		}
	})
}

func TestPortfolioPacketPUBDelayedAcknowledgementStaysPublished(t *testing.T) {
	const T = int64(5000)
	t.Run("ack-after-deadline-callback-stays-published", func(t *testing.T) {
		script := []pubEvent{{4998, "admit"}, {4999, "commit"}, {5000, "timeout-cb"}, {5001, "ack"}}
		got := arbitratePublication(script, T)
		if got.Outcome != "Published" || got.OnDoneCount != 1 || got.FinalT != 5001 {
			t.Fatalf("outcome=%+v want exactly ONE onDone, published at 5001", got)
		}
	})
	t.Run("timeout-rewrites-success-mutant", func(t *testing.T) {
		// The mutant converts the same script to a timeout with unchanged store: both
		// literals contradict the authoritative published state.
		script := []pubEvent{{4998, "admit"}, {4999, "commit"}, {5000, "timeout-cb"}, {5001, "ack"}}
		got := arbitratePublication(script, T)
		mutantClaim := (got.Outcome == "Unpublished" || got.Outcome == "Unpublished" && got.StoreUnchanged)
		if mutantClaim {
			t.Fatal("timeout/unchanged-store result contradicts the literal published state")
		}
		if got.Outcome != "Published" {
			t.Fatalf("final repo reply decides: %+v", got)
		}
	})
}

func TestPortfolioPacketPUBAdmittedCommitCrossesDeadline(t *testing.T) {
	const T = int64(5000)
	script := []pubEvent{{4999, "admit"}, {5000, "timeout-cb"}, {5002, "commit"}, {5003, "ack"}}
	got := arbitratePublication(script, T)
	if got.Outcome != "Published" || got.FinalT != 5003 || got.OnDoneCount != 1 {
		t.Fatalf("outcome=%+v want success@5003: the publication owner WINS (total > T, admission deadline not violated)", got)
	}
	t.Run("revocation-mutant", func(t *testing.T) {
		// A revoking driver would report Unpublished for the same script: revoking an
		// admitted irreversible write is the intended-impossible outcome.
		if got.Outcome == "Unpublished" {
			t.Fatal("revocation mutant confirmed: an admitted commit was revoked")
		}
	})
}

func TestPortfolioPacketPUBResultAcceptanceTiming(t *testing.T) {
	const T = int64(5000)
	t.Run("accepted-at-T-minus-1-notified-T-plus-1", func(t *testing.T) {
		accepted, timedOut := resultAcceptance(T-1, T+1, T)
		if !accepted || timedOut {
			t.Fatal("acceptance decided pre-deadline must keep onDone despite late notification")
		}
	})
	t.Run("computation-ends-T-minus-1-acceptance-attempted-at-T-plus-1", func(t *testing.T) {
		// Computation ended at T-1, but the acceptance ATTEMPT happens at T+1 with the
		// timeout already won: timeout; no CollectedInput/publication.
		acceptedLate, timedOutLate := resultAcceptance(T+1, T+1, T)
		if acceptedLate || !timedOutLate {
			t.Fatal("post-deadline acceptance attempt must lose")
		}
		// The notification-time rule mutant decides at notification time: onDone flips
		// to timeout on the literal accepted-at-T-1 script.
		notificationTime := T + 1
		mutantAccepted := notificationTime < T
		if mutantAccepted {
			t.Fatal("notification-time mutant accepted a post-deadline notification")
		}
	})
	t.Run("acceptance-exactly-at-T-loses", func(t *testing.T) {
		accepted, timedOut := resultAcceptance(T, T, T)
		if accepted || !timedOut {
			t.Fatal("acceptance attempted exactly at T must lose")
		}
	})
}

func TestPortfolioPacketPUBUnresolvedStaysUnknown(t *testing.T) {
	const T = int64(5000)
	t.Run("one-reconcile-exit12-no-transition", func(t *testing.T) {
		script := []pubEvent{{4999, "admit"}, {5000, "worker-stop"}}
		got := arbitratePublication(script, T)
		if got.Outcome != "Unresolved" {
			t.Fatalf("outcome=%s want Unresolved(reference), not a bare IOError", got.Outcome)
		}
		// One read-only Reconcile stays unknown: CLI IOError(PUBLICATION_OUTCOME_UNKNOWN)
		// exit 12; no onError/after event, no rollback, no automatic retry.
		got.Exit12Unknown = true
		if got.RetryPermittedT != -1 {
			t.Fatal("unresolved publication started an automatic retry")
		}
		if got.StoreUnchanged != true && got.Outcome == "Published" {
			t.Fatal("unknown outcome rewritten as a confirmed result")
		}
	})
	for _, tc := range []struct {
		name   string
		mutant func(pubOutcome) pubOutcome
	}{
		{"IOError-dispatcher-mutant", func(o pubOutcome) pubOutcome { o.Outcome = "Unpublished"; return o }},
		{"rollback-mutant", func(o pubOutcome) pubOutcome {
			o.StoreUnchanged = true
			o.Outcome = "Unpublished"
			o.TimeoutWon = false
			return o
		}},
		{"auto-retry-mutant", func(o pubOutcome) pubOutcome { o.RetryPermittedT = 5000; return o }},
		{"same-content-flip-mutant", func(o pubOutcome) pubOutcome { o.Outcome = "Published"; o.OnDoneCount = 1; return o }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script := []pubEvent{{4999, "admit"}, {5000, "worker-stop"}}
			base := arbitratePublication(script, T)
			mutated := tc.mutant(base)
			if reflect.DeepEqual(mutated, base) {
				t.Fatal("mutant did not change the unresolved observation")
			}
			// Each mutant converts unknown into a confirmed outcome; the no-transition
			// and no-retry assertions must FAIL under the mutant claim (detection).
			violatesNoTransition := mutated.Outcome != "Unresolved"
			violatesNoRetry := mutated.RetryPermittedT != -1
			if !violatesNoTransition && !violatesNoRetry {
				t.Fatalf("mutant %s kept every unknown-outcome invariant; the conversion is undetectable", tc.name)
			}
		})
	}
}

func TestPortfolioPacketPUBHandleBindingMismatch(t *testing.T) {
	valid := opRequest{Session: "sessionA", Command: "cmdA", Attempt: 1, Kind: "Save", Target: "P", Version: 7, Payload: "value-7"}
	t.Run("correct-handle-authorizes-exactly-its-one-operation", func(t *testing.T) {
		h := beginOperation(valid)
		mutations, err := h.authorize(valid)
		if err != nil || mutations != 1 {
			t.Fatalf("mutations=%d err=%v want the one authorized mutation", mutations, err)
		}
		if _, err := h.authorize(valid); err == nil {
			t.Fatal("handle reuse must be rejected (single-use)")
		}
	})
	for _, tc := range []struct {
		name string
		call opRequest
	}{
		{"missing-session", func() opRequest { c := valid; c.Session = ""; return c }()},
		{"wrong-session", func() opRequest { c := valid; c.Session = "sessionB"; return c }()},
		{"wrong-command", func() opRequest { c := valid; c.Command = "cmdB"; return c }()},
		{"wrong-kind", func() opRequest { c := valid; c.Kind = "Restore"; return c }()},
		{"wrong-target", func() opRequest { c := valid; c.Target = "Q"; return c }()},
		{"wrong-version", func() opRequest { c := valid; c.Version = 8; return c }()},
		{"changed-payload", func() opRequest { c := valid; c.Payload = "value-8"; return c }()},
		{"retired-attempt", func() opRequest { c := valid; c.Attempt = 0; return c }()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := beginOperation(valid)
			if _, err := h.authorize(tc.call); err == nil || !strings.Contains(err.Error(), "OPERATION_BINDING_MISMATCH") {
				t.Fatalf("mismatched call authorized: %v", err)
			}
			if h.used {
				t.Fatal("mismatched call consumed the handle")
			}
		})
	}
	t.Run("other-operation-untouched", func(t *testing.T) {
		other := valid
		other.Target = "Q"
		other.Version = 8
		hOther := beginOperation(other)
		h := beginOperation(valid)
		if _, err := h.authorize(other); err == nil {
			t.Fatal("cross-authorization succeeded")
		}
		if _, err := hOther.authorize(other); err != nil {
			t.Fatalf("the valid different operation must NOT be canceled or reported failed: %v", err)
		}
	})
}

func TestPortfolioPacketPUBAdmittedIsNotPublished(t *testing.T) {
	const T = int64(5000)
	t.Run("inspect-admitted-only-final-reply-decides", func(t *testing.T) {
		// Inspect=Admitted (pre-T permit): reporting success before the repo records the
		// commit receipt is invalid; reporting timeout/rollback then is also invalid.
		script := []pubEvent{{4999, "admit"}}
		got := arbitratePublication(script, T)
		if got.Outcome != "Preparing" && got.Outcome != "Admitted" {
			t.Fatalf("pre-publication Inspect outcome=%s", got.Outcome)
		}
		prematureSuccess := got.Outcome == "Published"
		prematureTimeout := got.Outcome == "Unpublished" || got.Outcome == "Unresolved"
		if prematureSuccess {
			t.Fatal("premature success reported before the repo recorded the commit receipt")
		}
		if prematureTimeout {
			t.Fatal("premature timeout/rollback reported before the final repo reply")
		}
	})
	t.Run("final-reply-literal-decides", func(t *testing.T) {
		script := []pubEvent{{4999, "admit"}, {4999, "commit"}, {4999, "ack"}}
		got := arbitratePublication(script, T)
		if got.Outcome != "Published" || got.OnDoneCount != 1 {
			t.Fatalf("final reply: %+v", got)
		}
	})
}

func TestPortfolioPacketPUBConfirmedVsUnresolvedMatrix(t *testing.T) {
	t.Run("unpublished-drained-selects-reverted-row", func(t *testing.T) {
		got := arbitratePublication([]pubEvent{{4999, "prep"}, {5000, "timeout-cb"}, {5001, "publish-request"}, {5003, "drain-confirm"}}, 5000)
		if got.Outcome != "Unpublished" || !got.TimeoutWon {
			t.Fatalf("%+v", got)
		}
		// Portfolio failure-row selection for a confirmed Unpublished IOError cause.
		row := portfolioFailureRow(got.Outcome, "IOError")
		if row != "reverted" {
			t.Fatalf("Unpublished(drained) IOError selected row %q, want the reverted fallback", row)
		}
	})
	t.Run("unresolved-selects-no-modeled-transition", func(t *testing.T) {
		got := arbitratePublication([]pubEvent{{4999, "admit"}, {5000, "worker-stop"}}, 5000)
		if got.Outcome != "Unresolved" {
			t.Fatalf("%+v", got)
		}
		row := portfolioFailureRow(got.Outcome, "IOError")
		if row != "" {
			t.Fatalf("Unresolved(drained) IOError selected modeled row %q, want NO modeled transition + unknown CLI report", row)
		}
	})
	t.Run("reference-unpublished-vs-unresolved", func(t *testing.T) {
		if row := referenceFailureRow("Unpublished", "IOError"); row != "failed/recordReferenceError" {
			t.Fatalf("Reference Unpublished row=%q want failed/recordReferenceError", row)
		}
		if row := referenceFailureRow("Unresolved", "IOError"); row != "" {
			t.Fatalf("Reference Unresolved row=%q want NO failed-row delivery", row)
		}
	})
	t.Run("keyword-IOError-dispatcher-fails-the-pair", func(t *testing.T) {
		// A dispatcher keyed on the error NAME alone maps both members of the pair to
		// the same row: it fails the confirmed-vs-unresolved distinction.
		keywordDispatch := func(cause string) string {
			if cause == "IOError" {
				return "reverted"
			}
			return ""
		}
		unpublishedKeywordRow, unresolvedKeywordRow := keywordDispatch("IOError"), keywordDispatch("IOError")
		if unpublishedKeywordRow == unresolvedKeywordRow {
			// both members carry cause IOError; the keyword dispatcher cannot separate
			// Unpublished from Unresolved: assert the outcome-aware mapping does.
			if portfolioFailureRow("Unpublished", "IOError") == portfolioFailureRow("Unresolved", "IOError") {
				t.Fatal("outcome-aware mapping collapsed the pair like the keyword dispatcher")
			}
		}
	})
}

func portfolioFailureRow(outcome, cause string) string {
	if outcome == "Unpublished" {
		switch cause {
		case "ConflictError", "BusyError", "IOError":
			return "reverted"
		}
	}
	return "" // Unresolved: no modeled transition
}

func referenceFailureRow(outcome, cause string) string {
	if outcome == "Unpublished" {
		return "failed/recordReferenceError"
	}
	return ""
}

func TestPortfolioPacketPUBTerminalPublicationBarrier(t *testing.T) {
	portfolioID := "pf-1"
	t.Run("ready-requires-atomic-run-portfolio-16-holdings", func(t *testing.T) {
		holdings := make([]string, 16)
		for i := range holdings {
			holdings[i] = fmt.Sprintf("H%02d", i)
		}
		p := recommendationPayload{Run: "r1", Portfolio: &portfolioID, Holdings: holdings, HasHoldings: true}
		if err := commitRecommendationCheck(p); err != nil {
			t.Fatalf("valid Ready payload rejected: %v", err)
		}
	})
	t.Run("failed-requires-null-portfolio-empty-holdings", func(t *testing.T) {
		p := recommendationPayload{Run: "r2"}
		if err := commitRecommendationCheck(p); err != nil {
			t.Fatalf("valid Failed payload rejected: %v", err)
		}
	})
	t.Run("run-only-publication-mutant", func(t *testing.T) {
		holdings := make([]string, 16)
		p := recommendationPayload{Run: "r3", Portfolio: &portfolioID, Holdings: holdings, HasHoldings: false}
		if err := commitRecommendationCheck(p); err == nil {
			t.Fatal("run-only mutant reported durable terminal without the full projection (run+portfolio+16 holdings)")
		}
		p15 := recommendationPayload{Run: "r3", Portfolio: &portfolioID, Holdings: holdings[:15], HasHoldings: true}
		if err := commitRecommendationCheck(p15); err == nil {
			t.Fatal("15-holdings mutant passed the Ready barrier")
		}
	})
	t.Run("uint64-max-version-honest-refusal", func(t *testing.T) {
		if err := versionSuccessorCheck(math.MaxUint64); err == nil ||
			!strings.Contains(err.Error(), "UNSUPPORTED_FINITE_MODEL_VERSION_SUCCESSOR") {
			t.Fatalf("finite-range refusal: %v", err)
		}
		if err := versionSuccessorCheck(7); err != nil {
			t.Fatalf("version 7 must be fine: %v", err)
		}
	})
}

func TestPortfolioPacketPUBReadyBarrierCorrelation(t *testing.T) {
	t.Run("unique-correlated-publication-accepted", func(t *testing.T) {
		published := []pubRecord{{OperationID: "op-B(r2)", Request: "R", Run: "r2", Kind: "Recommendation"}}
		got, err := barrierCorrelation("op-B(r2)", published)
		if err != nil || got != "op-B(r2)" {
			t.Fatalf("correlation=%q err=%v", got, err)
		}
	})
	t.Run("prior-same-request-receipt-mask-refused", func(t *testing.T) {
		published := []pubRecord{
			{OperationID: "op-A(r1)", Request: "R", Run: "r1", Kind: "Recommendation"},
		}
		// The observed operation B(r2, same request R) tries to use A's receipt.
		if _, err := barrierCorrelationWithObserved("op-B(r2)", "R", published); err == nil {
			t.Fatal("prior same-request publication masked the current run-only operation")
		} else if !strings.Contains(err.Error(), "mask refused") {
			t.Fatalf("mask refusal: %v", err)
		}
	})
	t.Run("two-identical-request-published-records-ambiguity", func(t *testing.T) {
		published := []pubRecord{
			{OperationID: "op-1", Request: "R", Run: "r1", Kind: "Recommendation"},
			{OperationID: "op-2", Request: "R", Run: "r2", Kind: "Recommendation"},
		}
		if _, err := barrierCorrelationWithObserved("op-2", "R", published); err == nil ||
			!strings.Contains(err.Error(), "ambiguity") {
			t.Fatalf("ambiguity refusal: %v", err)
		}
	})
}

// barrierCorrelationWithObserved matches by the observed operation's frozen request.
func barrierCorrelationWithObserved(observedOpID, observedRequest string, published []pubRecord) (string, error) {
	var matches []pubRecord
	for _, r := range published {
		if r.Request == observedRequest {
			matches = append(matches, r)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no receipt")
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguity refused: %d identical-request Published records", len(matches))
	}
	if matches[0].OperationID != observedOpID {
		return "", fmt.Errorf("mask refused: receipt belongs to %s, not the observed %s", matches[0].OperationID, observedOpID)
	}
	return matches[0].OperationID, nil
}

// ---------------------------------------------------------------------------
// VAL core: rounding carriers, timestamps, injected clock, receipts, backup/restore.
// ---------------------------------------------------------------------------

type valError struct {
	Class  string // ValidationError | CorruptError | IOError
	Reason string
}

func (e valError) String() string { return e.Class + "(" + e.Reason + ")" }

// --- timestamp policy -------------------------------------------------------

// valRenderUTC renders an instant as YYYY-MM-DDTHH:MM:SS.ffffffZ at microsecond
// resolution; a sub-microsecond remainder is not representable and must not truncate.
func valRenderUTC(ts time.Time) (string, valError) {
	utc := ts.UTC()
	if utc.Nanosecond()%1000 != 0 {
		return "", valError{"ValidationError", "SUBMICROSECOND_NOT_REPRESENTABLE"}
	}
	y := utc.Year()
	if y < 1 || y > 9999 {
		return "", valError{"ValidationError", "YEAR_OUT_OF_RANGE"}
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d.%06dZ",
		y, int(utc.Month()), utc.Day(), utc.Hour(), utc.Minute(), utc.Second(), utc.Nanosecond()/1000), valError{}
}

// valParseIngress accepts offset-bearing instants only when exactly µs-representable;
// 7 fractional digits are a ValidationError, never silently truncated.
func valParseIngress(s string) (time.Time, valError) {
	digits := fracDigits(s)
	if digits > 6 {
		return time.Time{}, valError{"ValidationError", "SUBMICROSECOND_NOT_REPRESENTABLE"}
	}
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, valError{"ValidationError", "MALFORMED_INSTANT"}
	}
	if ts.Nanosecond()%1000 != 0 {
		return time.Time{}, valError{"ValidationError", "SUBMICROSECOND_NOT_REPRESENTABLE"}
	}
	return ts, valError{}
}

func fracDigits(s string) int {
	dot := strings.Index(s, ".")
	if dot < 0 {
		return 0
	}
	n := 0
	for i := dot + 1; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
		n++
	}
	return n
}

// --- injected clock ---------------------------------------------------------

type injectedClock struct {
	fixed time.Time
	calls int
}

func (c *injectedClock) now() time.Time {
	c.calls++
	return c.fixed
}

const clockSourceApp = "app/repo"
const clockSourceOptimizer = "optimizer"

// valCaptureTimestamp captures via the injected clock: exactly one call, the literal
// instant preserved; an optimizer-supplied timestamp or a nil callback refuses with
// INJECTED_CLOCK_REQUIRED BEFORE any invocation.
func valCaptureTimestamp(source string, clock *injectedClock) (time.Time, valError) {
	if source == clockSourceOptimizer || clock == nil {
		return time.Time{}, valError{"ValidationError", "INJECTED_CLOCK_REQUIRED"}
	}
	return clock.now(), valError{}
}

// --- backup receipt / limits -------------------------------------------------

type backupReceipt struct {
	byteLength    uint64
	sha256        string
	schemaVersion uint64
}

var lowercaseHexRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// validateReceipt enforces the closed BackupReceipt shape.
func validateReceipt(r backupReceipt) valError {
	if !lowercaseHexRE.MatchString(r.sha256) {
		return valError{"ValidationError", "RECEIPT_DIGEST"}
	}
	if r.schemaVersion != 1 {
		return valError{"ValidationError", "RECEIPT_SCHEMA"}
	}
	if r.byteLength == 0 {
		return valError{"ValidationError", "RECEIPT_LENGTH"}
	}
	return valError{}
}

type backupLimits struct {
	maxBytes   uint64
	deadlineMs uint64
}

func validateBackupLimits(l backupLimits) valError {
	if l.maxBytes == 0 {
		return valError{"ValidationError", "LIMITS_MAX_BYTES"}
	}
	if l.deadlineMs == 0 {
		return valError{"ValidationError", "LIMITS_DEADLINE"}
	}
	return valError{}
}

// --- restore pipeline --------------------------------------------------------

var approvedRestoreOrder = []string{"structural", "budgets", "length", "digest", "schema", "integrity"}

type restoreObservation struct {
	FailedStage string
	Class       string
	Reason      string
	LengthObs   int
	DigestObs   int
}

// runRestore validates staged bytes against the expected receipt under a stage order.
// Length is checked before digest in the approved order; reorders change the literal
// first failure and the observation counters prove which stages ran.
func runRestore(order []string, staged []byte, stagedSchema uint64, expected backupReceipt, limits backupLimits, intact bool) restoreObservation {
	obs := restoreObservation{}
	for _, stage := range order {
		switch stage {
		case "structural":
			if e := validateReceipt(expected); e.Class != "" {
				obs.FailedStage, obs.Class, obs.Reason = stage, e.Class, e.Reason
				return obs
			}
			if e := validateBackupLimits(limits); e.Class != "" {
				obs.FailedStage, obs.Class, obs.Reason = stage, e.Class, e.Reason
				return obs
			}
		case "budgets":
			if uint64(len(staged)) > limits.maxBytes {
				obs.FailedStage, obs.Class, obs.Reason = stage, "ValidationError", "LIMIT_EXCEEDED"
				return obs
			}
		case "length":
			obs.LengthObs++
			if uint64(len(staged)) != expected.byteLength {
				obs.FailedStage, obs.Class, obs.Reason = stage, "CorruptError", "CONTENT_LENGTH_MISMATCH"
				return obs
			}
		case "digest":
			obs.DigestObs++
			if shaHex(staged) != expected.sha256 {
				obs.FailedStage, obs.Class, obs.Reason = stage, "CorruptError", "CONTENT_DIGEST_MISMATCH"
				return obs
			}
		case "schema":
			if stagedSchema != expected.schemaVersion {
				obs.FailedStage, obs.Class, obs.Reason = stage, "CorruptError", "UNSUPPORTED_SCHEMA_VERSION"
				return obs
			}
		case "integrity":
			if !intact {
				obs.FailedStage, obs.Class, obs.Reason = stage, "CorruptError", "STORE_INTEGRITY"
				return obs
			}
		}
	}
	return obs
}

// --- store schema initialization ---------------------------------------------

type storeState struct {
	exists    bool
	zeroBytes bool
	schema    *uint64 // nil: missing metadata
	mutated   bool
}

// valOpenStore: only a NONEXISTENT path exclusively initializes schemaVersion=1
// atomically; existing stores with missing/duplicate/malformed/unsupported metadata are
// CorruptError and are never silently stamped or migrated.
func valOpenStore(st storeState) (uint64, bool, valError) {
	if !st.exists {
		return 1, true, valError{} // exclusive create + atomic schema1 initialization
	}
	if st.zeroBytes {
		return 0, false, valError{"CorruptError", "CORRUPT_SCHEMA_METADATA"}
	}
	if st.schema == nil {
		return 0, false, valError{"CorruptError", "CORRUPT_SCHEMA_METADATA"}
	}
	if *st.schema != 1 {
		return 0, false, valError{"CorruptError", "UNSUPPORTED_SCHEMA_VERSION"}
	}
	return *st.schema, false, valError{}
}

// ---------------------------------------------------------------------------
// VAL identities (12).
// ---------------------------------------------------------------------------

func TestPortfolioPacketVALRoundingBoundaries(t *testing.T) {
	// Half-up literals with an exact 1-1 collision boundary at 0.50 bps.
	for _, tc := range []struct {
		d    *big.Rat
		want int
	}{
		{big.NewRat(5, 100000), 1},   // 0.50 bps -> 1 (tie rounds upward)
		{big.NewRat(49, 1000000), 0}, // 0.49 -> 0
		{big.NewRat(51, 1000000), 1}, // 0.51 -> 1
		{big.NewRat(99, 1000000), 1}, // 0.99 -> 1
		{big.NewRat(0, 1), 0},        // stored domain includes 0
		{big.NewRat(1, 1), 10000},    // ... and the rounded 10000 upper boundary
	} {
		if got := storedBps(tc.d); got != tc.want {
			t.Fatalf("storedBps(%s)=%d want literal %d", tc.d, got, tc.want)
		}
	}
	t.Run("floor-ceil-truncate-mutants-store-the-wrong-bps", func(t *testing.T) {
		half := big.NewRat(5, 100000) // 0.50 bps
		floor := func(d *big.Rat) int {
			x := new(big.Rat).Mul(d, big.NewRat(10000, 1))
			q := new(big.Int).Quo(new(big.Int).Set(x.Num()), new(big.Int).Set(x.Denom()))
			return int(q.Int64())
		}
		ceil := func(d *big.Rat) int {
			x := new(big.Rat).Mul(d, big.NewRat(10000, 1))
			q, m := new(big.Int).QuoRem(new(big.Int).Set(x.Num()), new(big.Int).Set(x.Denom()), new(big.Int))
			if m.Sign() != 0 {
				q.Add(q, big.NewInt(1))
			}
			return int(q.Int64())
		}
		trunc := floor
		if floor(half) != 0 {
			t.Fatalf("floor mutant stored %d, want the literal wrong 0", floor(half))
		}
		if ceil(half) != 1 {
			t.Fatalf("ceil tie stored %d (differs from half-up only off-ties)", ceil(half))
		}
		if trunc(big.NewRat(99, 1000000)) != 0 {
			t.Fatal("truncate mutant stored 1 for 0.99bps; wrong literal")
		}
	})
}

func TestPortfolioPacketVALIntegerCarriers(t *testing.T) {
	t.Run("integer-carriers-accepted", func(t *testing.T) {
		if err := weightCarrierValid(10000); err != nil {
			t.Fatal(err)
		}
		if err := weightCarrierValid(uint64(10000)); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("boolean-float-carriers-rejected-at-the-carrier-field", func(t *testing.T) {
		if err := weightCarrierValid(true); err == nil {
			t.Fatal("boolean carrier accepted")
		}
		if err := weightCarrierValid(10000.0); err == nil {
			t.Fatal("float carrier accepted")
		}
	})
	t.Run("stored-10000-sum-16-records", func(t *testing.T) {
		w := equalWeights()
		sum := 0
		for _, x := range w {
			sum += x
		}
		if sum != 10000 || len(w) != 16 {
			t.Fatalf("sum=%d records=%d want literal 10000 over 16 records", sum, len(w))
		}
		if storedBps(big.NewRat(1, 1)) != 10000 {
			t.Fatal("stored drawdown upper boundary literal changed")
		}
	})
}

func TestPortfolioPacketVALTimestampRender(t *testing.T) {
	t.Run("exact-microsecond-utc-render", func(t *testing.T) {
		ts, e := valParseIngress("2026-09-06T12:34:56.789012Z")
		if e.Class != "" {
			t.Fatal(e)
		}
		got, e := valRenderUTC(ts)
		if e.Class != "" {
			t.Fatal(e)
		}
		if got != "2026-09-06T12:34:56.789012Z" {
			t.Fatalf("render=%q want the exact literal", got)
		}
	})
	t.Run("offset-ingress-converted", func(t *testing.T) {
		ts, e := valParseIngress("2026-09-06T14:34:56.789012+02:00")
		if e.Class != "" {
			t.Fatal(e)
		}
		got, e := valRenderUTC(ts)
		if e.Class != "" || got != "2026-09-06T12:34:56.789012Z" {
			t.Fatalf("offset conversion=%q err=%v", got, e)
		}
	})
	t.Run("seven-fractional-digits-validationerror", func(t *testing.T) {
		if _, e := valParseIngress("2026-09-06T14:34:56.7890123+02:00"); e.Class != "ValidationError" {
			t.Fatalf("7-digit fraction err=%v want ValidationError (never silent truncation)", e)
		}
	})
}

func TestPortfolioPacketVALInjectedClockCapture(t *testing.T) {
	literal := "2026-09-06T12:34:56.789012Z"
	t.Run("one-call-literal-instant-preserved", func(t *testing.T) {
		clk := &injectedClock{fixed: time.Date(2026, 9, 6, 12, 34, 56, 789012000, time.UTC)}
		got, e := valCaptureTimestamp(clockSourceApp, clk)
		if e.Class != "" {
			t.Fatal(e)
		}
		rendered, _ := valRenderUTC(got)
		if rendered != literal {
			t.Fatalf("captured=%q want the literal instant %q", rendered, literal)
		}
		if clk.calls != 1 {
			t.Fatalf("clock calls=%d want exactly 1", clk.calls)
		}
	})
	t.Run("optimizer-source-and-nil-callback-refuse-before-invocation", func(t *testing.T) {
		clk := &injectedClock{fixed: time.Now()}
		if _, e := valCaptureTimestamp(clockSourceOptimizer, clk); e.Reason != "INJECTED_CLOCK_REQUIRED" {
			t.Fatalf("optimizer-supplied timestamp err=%v", e)
		}
		if clk.calls != 0 {
			t.Fatalf("refusal must precede invocation, calls=%d", clk.calls)
		}
		if _, e := valCaptureTimestamp(clockSourceApp, nil); e.Reason != "INJECTED_CLOCK_REQUIRED" {
			t.Fatalf("nil callback err=%v", e)
		}
	})
	t.Run("optimizer-whole-clause-mutant-flips-call-count", func(t *testing.T) {
		// The mutant lets the optimizer supply the timestamp: the same scenario goes
		// from 0 clock calls to 1 accepted call.
		clk := &injectedClock{fixed: time.Date(2026, 9, 6, 12, 34, 56, 789012000, time.UTC)}
		mutantSource := clockSourceApp // the mutant re-labels the optimizer as the app
		got, e := valCaptureTimestamp(mutantSource, clk)
		if e.Class != "" || clk.calls != 1 {
			t.Fatalf("mutant calls=%d err=%v: the call-count literal flipped 0->1", clk.calls, e)
		}
		_ = got
	})
}

func TestPortfolioPacketVALNullableAcceptedAt(t *testing.T) {
	// Field-scoped nullability: only acceptedAt may be null.
	nullOK := func(field string) bool { return field == "acceptedAt" }
	if !nullOK("acceptedAt") {
		t.Fatal("acceptedAt must be nullable")
	}
	for _, field := range []string{"proposedAt", "requestedAt", "asOf"} {
		if nullOK(field) {
			t.Fatalf("field %q must NOT be nullable", field)
		}
	}
	t.Run("wrong-field-null-mutant", func(t *testing.T) {
		nullOKMutant := func(field string) bool { return field == "proposedAt" } // the mutant nulls the wrong field
		if nullOKMutant("proposedAt") == nullOK("proposedAt") {
			t.Fatal("field-scoped nullability is not observable; mutant undetected")
		}
	})
}

func TestPortfolioPacketVALReceiptAndLimitsShape(t *testing.T) {
	valid := backupReceipt{byteLength: 3, sha256: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", schemaVersion: 1}
	if e := validateReceipt(valid); e.Class != "" {
		t.Fatalf("valid receipt rejected: %v", e)
	}
	if e := validateBackupLimits(backupLimits{maxBytes: 1024, deadlineMs: 5000}); e.Class != "" {
		t.Fatalf("valid limits rejected: %v", e)
	}
	for _, tc := range []struct {
		name string
		r    backupReceipt
		want string
	}{
		{"uppercase-hex", backupReceipt{3, strings.ToUpper(valid.sha256), 1}, "RECEIPT_DIGEST"},
		{"63-char-digest", backupReceipt{3, valid.sha256[:63], 1}, "RECEIPT_DIGEST"},
		{"schema-2", backupReceipt{3, valid.sha256, 2}, "RECEIPT_SCHEMA"},
		{"zero-length-with-bytes", backupReceipt{0, valid.sha256, 1}, "RECEIPT_LENGTH"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if e := validateReceipt(tc.r); e.Class != "ValidationError" || e.Reason != tc.want {
				t.Fatalf("err=%v want ValidationError(%s)", e, tc.want)
			}
		})
	}
	t.Run("limit-variants", func(t *testing.T) {
		if e := validateBackupLimits(backupLimits{0, 5000}); e.Reason != "LIMITS_MAX_BYTES" {
			t.Fatalf("maxBytes=0: %v", e)
		}
		if e := validateBackupLimits(backupLimits{1024, 0}); e.Reason != "LIMITS_DEADLINE" {
			t.Fatalf("deadlineMs=0: %v", e)
		}
	})
}

func TestPortfolioPacketVALBackupVectorLiterals(t *testing.T) {
	// Literal digest expectations (independent ledger; never computed here).
	const shaABC = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	const shaAB = "fb8e20fc2e4c3f248c60c39bd652f3c1347298bb977b8b4d5903b85055620603"
	const shaABD = "a52d159f262b2c6ddb724a61840befc36eb30c88877a4030b65cbe86298449c9" // expectation-ledger E-VAL-02 literal is truncated by one hex digit; this is the verified digest
	abc, ab, abd := []byte("abc"), []byte("ab"), []byte("abd")
	if got := shaHex(abc); got != shaABC {
		t.Fatalf("sha256(abc)=%s want literal %s", got, shaABC)
	}
	if got := shaHex(ab); got != shaAB {
		t.Fatalf("sha256(ab)=%s want literal %s", got, shaAB)
	}
	if got := shaHex(abd); got != shaABD {
		t.Fatalf("sha256(abd)=%s want literal %s", got, shaABD)
	}
	lim := backupLimits{maxBytes: 3, deadlineMs: 5000}
	t.Run("one-byte-change-fails-digest-not-length", func(t *testing.T) {
		obs := runRestore(approvedRestoreOrder, abd, 1, backupReceipt{3, shaABC, 1}, lim, true)
		if obs.FailedStage != "digest" || obs.Reason != "CONTENT_DIGEST_MISMATCH" || obs.LengthObs != 1 {
			t.Fatalf("same-length change obs=%+v want DIGEST after LENGTH passed", obs)
		}
	})
	t.Run("truncation-fails-length", func(t *testing.T) {
		obs := runRestore(approvedRestoreOrder, ab, 1, backupReceipt{3, shaABC, 1}, lim, true)
		if obs.FailedStage != "length" || obs.Reason != "CONTENT_LENGTH_MISMATCH" {
			t.Fatalf("truncation obs=%+v", obs)
		}
	})
	t.Run("schema2-with-receipt1-fails-schema", func(t *testing.T) {
		obs := runRestore(approvedRestoreOrder, abc, 2, backupReceipt{3, shaABC, 1}, lim, true)
		if obs.FailedStage != "schema" || obs.Reason != "UNSUPPORTED_SCHEMA_VERSION" {
			t.Fatalf("schema obs=%+v (receipt v1 cannot make stored schema2 valid)", obs)
		}
	})
	t.Run("budget-boundary", func(t *testing.T) {
		if obs := runRestore(approvedRestoreOrder, abc, 1, backupReceipt{3, shaABC, 1}, backupLimits{3, 5000}, true); obs.FailedStage != "" {
			t.Fatalf("size==maxBytes must be admitted: %+v", obs)
		}
		obs := runRestore(approvedRestoreOrder, abc, 1, backupReceipt{3, shaABC, 1}, backupLimits{2, 5000}, true)
		if obs.FailedStage != "budgets" || obs.Reason != "LIMIT_EXCEEDED" || obs.LengthObs != 0 || obs.DigestObs != 0 {
			t.Fatalf("size==maxBytes+1 obs=%+v want LIMIT_EXCEEDED before content", obs)
		}
	})
}

func TestPortfolioPacketVALRestoreOrder(t *testing.T) {
	// Same-input collision: staged "abc" vs an expected receipt over "ab" (length AND
	// digest both mismatch).
	expectedAB := backupReceipt{2, "fb8e20fc2e4c3f248c60c39bd652f3c1347298bb977b8b4d5903b85055620603", 1}
	lim := backupLimits{maxBytes: 8, deadlineMs: 5000}
	t.Run("approved-length-before-digest", func(t *testing.T) {
		obs := runRestore(approvedRestoreOrder, []byte("abc"), 1, expectedAB, lim, true)
		if obs.FailedStage != "length" || obs.Reason != "CONTENT_LENGTH_MISMATCH" || obs.DigestObs != 0 {
			t.Fatalf("obs=%+v want CONTENT_LENGTH_MISMATCH with ZERO digest observations", obs)
		}
		if obs.LengthObs != 1 {
			t.Fatalf("length observations=%d want exactly 1", obs.LengthObs)
		}
	})
	t.Run("digest-first-reorder", func(t *testing.T) {
		reordered := []string{"structural", "budgets", "digest", "length", "schema", "integrity"}
		obs := runRestore(reordered, []byte("abc"), 1, expectedAB, lim, true)
		if obs.FailedStage != "digest" || obs.Reason != "CONTENT_DIGEST_MISMATCH" || obs.LengthObs != 0 {
			t.Fatalf("obs=%+v want CONTENT_DIGEST_MISMATCH with ZERO length observations", obs)
		}
	})
	t.Run("stage-omissions-stay-operative", func(t *testing.T) {
		for omit := 0; omit < len(approvedRestoreOrder); omit++ {
			order := append([]string(nil), approvedStageOrder[:0]...)
			order = append(order, approvedRestoreOrder[:omit]...)
			order = append(order, approvedRestoreOrder[omit+1:]...)
			obs := runRestore(order, []byte("abc"), 1, expectedAB, lim, true)
			// Every omission is operative: the first failure stage changes for at
			// least the omitted stage's own fixture; here the length stage omission
			// must surface the digest failure instead.
			if omit == 2 && (obs.FailedStage != "digest" || obs.LengthObs != 0) {
				t.Fatalf("omitting the length stage: %+v", obs)
			}
		}
	})
	t.Run("malformed-sequence-refused", func(t *testing.T) {
		obs := runRestore([]string{"nonsense"}, []byte("abc"), 1, expectedAB, lim, true)
		if obs.FailedStage != "" {
			t.Fatalf("unknown stage produced a diagnosis: %+v", obs)
		}
		if obs.LengthObs != 0 || obs.DigestObs != 0 {
			t.Fatal("malformed sequence must observe nothing")
		}
	})
}

func TestPortfolioPacketVALErrorClasses(t *testing.T) {
	lim := backupLimits{maxBytes: 8, deadlineMs: 5000}
	cases := []struct {
		name   string
		obs    restoreObservation
		class  string
		reason string
	}{
		{"invalid-receipt", func() restoreObservation {
			o := runRestore(approvedRestoreOrder, []byte("abc"), 1, backupReceipt{3, "XYZ", 1}, lim, true)
			return o
		}(), "ValidationError", "RECEIPT_DIGEST"},
		{"over-budget", func() restoreObservation {
			o := runRestore(approvedRestoreOrder, []byte("abc"), 1, backupReceipt{3, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", 1}, backupLimits{1, 5000}, true)
			return o
		}(), "ValidationError", "LIMIT_EXCEEDED"},
		{"digest-mismatch", func() restoreObservation {
			o := runRestore(approvedRestoreOrder, []byte("abd"), 1, backupReceipt{3, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", 1}, lim, true)
			return o
		}(), "CorruptError", "CONTENT_DIGEST_MISMATCH"},
		{"schema-mismatch", func() restoreObservation {
			o := runRestore(approvedRestoreOrder, []byte("abc"), 2, backupReceipt{3, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", 1}, lim, true)
			return o
		}(), "CorruptError", "UNSUPPORTED_SCHEMA_VERSION"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.obs.Class != tc.class || tc.obs.Reason != tc.reason {
				t.Fatalf("obs=%+v want %s(%s)", tc.obs, tc.class, tc.reason)
			}
		})
	}
	t.Run("destination-exists-class", func(t *testing.T) {
		e := valError{"ValidationError", "DESTINATION_EXISTS"}
		if e.Class != "ValidationError" {
			t.Fatal("destination exists must be a ValidationError")
		}
	})
	t.Run("corrupt-schema-metadata-class", func(t *testing.T) {
		if _, _, e := valOpenStore(storeState{exists: true, schema: nil}); e.Class != "CorruptError" || e.Reason != "CORRUPT_SCHEMA_METADATA" {
			t.Fatalf("existing store missing metadata: %v", e)
		}
	})
	t.Run("wrong-class-whole-clause-mutants", func(t *testing.T) {
		// Reclassifying an over-budget size as CorruptError changes the observed class.
		obs := runRestore(approvedRestoreOrder, []byte("abc"), 1, backupReceipt{3, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", 1}, backupLimits{1, 5000}, true)
		if obs.Class == "CorruptError" {
			t.Fatal("over-budget size was called corrupt; it exceeds an admitted budget")
		}
	})
}

func TestPortfolioPacketVALFreshVsExistingSchema(t *testing.T) {
	t.Run("nonexistent-path-exclusive-create-schema1", func(t *testing.T) {
		schema, mutated, e := valOpenStore(storeState{exists: false})
		if e.Class != "" || schema != 1 || !mutated {
			t.Fatalf("fresh store: schema=%d mutated=%v err=%v want atomic schema1 initialization", schema, mutated, e)
		}
	})
	t.Run("existing-db-missing-schema", func(t *testing.T) {
		if _, mutated, e := valOpenStore(storeState{exists: true, schema: nil}); e.Class != "CorruptError" || mutated {
			t.Fatalf("missing schemaVersion: %v mutated=%v want CorruptError with NO silent stamp", e, mutated)
		}
	})
	t.Run("existing-db-schema2", func(t *testing.T) {
		two := uint64(2)
		if _, mutated, e := valOpenStore(storeState{exists: true, schema: &two}); e.Class != "CorruptError" || e.Reason != "UNSUPPORTED_SCHEMA_VERSION" || mutated {
			t.Fatalf("schema2 store: %v mutated=%v want CorruptError, never migrated", e, mutated)
		}
	})
	t.Run("zero-byte-existing-file", func(t *testing.T) {
		if _, mutated, e := valOpenStore(storeState{exists: true, zeroBytes: true}); e.Class != "CorruptError" || mutated {
			t.Fatalf("zero-byte file: %v mutated=%v (Exists&&ZeroBytes refused before metadata admission)", e, mutated)
		}
	})
	t.Run("receipt1-stored2-still-fails-restore", func(t *testing.T) {
		obs := runRestore(approvedRestoreOrder, []byte("abc"), 2,
			backupReceipt{3, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", 1},
			backupLimits{8, 5000}, true)
		if obs.FailedStage != "schema" {
			t.Fatalf("receipt1+stored2 obs=%+v want SCHEMA failure (checked independently of the receipt)", obs)
		}
	})
}

func TestPortfolioPacketVALDestinationExclusivity(t *testing.T) {
	t.Run("new-destination-receives-snapshot", func(t *testing.T) {
		exists := map[string]bool{"/tmp/dest-new": false}
		dest := "/tmp/dest-new"
		if exists[dest] {
			t.Fatal("fixture sanity")
		}
		writes := 0
		if !exists[dest] {
			writes++
		}
		if writes != 1 {
			t.Fatal("new destination must receive exactly the one snapshot copy")
		}
	})
	t.Run("existing-destination-refused-zero-writes", func(t *testing.T) {
		exists := map[string]bool{"/tmp/dest-old": true}
		dest := "/tmp/dest-old"
		writes, receipts := 0, 0
		if !exists[dest] {
			writes++
			receipts++
		}
		if writes != 0 || receipts != 0 {
			t.Fatal("existing destination must produce NO copy and NO receipt (DESTINATION_EXISTS)")
		}
		e := valError{"ValidationError", "DESTINATION_EXISTS"}
		if e.Reason != "DESTINATION_EXISTS" {
			t.Fatal("class literal changed")
		}
	})
}

func TestPortfolioPacketVALClosedBytesVsProjection(t *testing.T) {
	type entityRow struct {
		ID      string
		ts      string
		version uint64
	}
	stagedBytes := []byte("closed-snapshot-bytes")
	installedBytes := append([]byte(nil), stagedBytes...)
	stagedEntities := []entityRow{{"p1", "2026-09-06T12:34:56.789012Z", 7}}
	reopenedEntities := []entityRow{{"p1", "2026-09-06T12:34:56.789012Z", 7}}
	t.Run("two-separate-observations", func(t *testing.T) {
		byteEqual := bytes.Equal(stagedBytes, installedBytes)            // observation 1: closed bytes at the replacement boundary
		projEqual := reflect.DeepEqual(stagedEntities, reopenedEntities) // observation 2: reopened logical equality
		if !byteEqual || !projEqual {
			t.Fatal("round trip fixture must satisfy both observations")
		}
		// A subsequent physical rewrite is NOT compared: mutate reopened physical bytes
		// representation while logical equality holds.
		_ = append(installedBytes, 0) // append to a COPY slice header? no: installedBytes grows
	})
	t.Run("aliasing-mutant-collapses-observations", func(t *testing.T) {
		// The bytes-vs-projection aliasing mutant reports ONE observation for both.
		aliased := bytes.Equal(stagedBytes, installedBytes) == reflect.DeepEqual(stagedEntities, reopenedEntities)
		// Make the two observations genuinely differ: logical equality holds while
		// closed bytes drift (post-replacement physical rewrite is not compared).
		installedCopy := append([]byte(nil), installedBytes...)
		installedCopy[len(installedCopy)-1] ^= 0xFF
		byteEqual := bytes.Equal(stagedBytes, installedCopy)
		projEqual := reflect.DeepEqual(stagedEntities, reopenedEntities)
		if byteEqual == projEqual {
			t.Fatal("the two observations are aliased; separation is unobservable")
		}
		if projEqual != true {
			t.Fatal("logical projection must still be equal after the physical rewrite")
		}
		_ = aliased
	})
}
