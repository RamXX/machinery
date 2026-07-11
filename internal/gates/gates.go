// Package gates is the Go port of machinery_check.py: the deterministic gate
// suite (G2-c4, G3-machine, Gx-trace, G4-import). Pure static analysis over
// the design artifacts and (with --impl) the code.
package gates

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/alloy"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/lint"
	"github.com/RamXX/machinery/internal/oracle"
)

// --- Gate accumulator (mirrors machinery_check.Gate) ---

// Gate holds findings plus an explicit record of what was verified.
type Gate struct {
	Title        string
	Errs         []string
	Drift        []string
	Warns        []string
	Notes        []string
	Counts       map[string]int
	countOrder   []string // insertion order of count keys (matches Python dict)
	checkedExtra []string // preformatted checked: segments (zeros stay visible)
}

// NewGate builds an empty gate.
func NewGate(title string) *Gate {
	return &Gate{Title: title, Counts: map[string]int{}}
}

// startOrder initializes the count-order tracking (no-op; kept for clarity).
func (g *Gate) startOrder() {}

// Count increments label by n (default 1).
func (g *Gate) Count(label string, n ...int) {
	if _, ok := g.Counts[label]; !ok {
		g.countOrder = append(g.countOrder, label)
	}
	inc := 1
	if len(n) > 0 {
		inc = n[0]
	}
	g.Counts[label] += inc
}

// RequireNonzero adds an ERROR if label was never counted.
func (g *Gate) RequireNonzero(label, what string) {
	if g.Counts[label] == 0 {
		g.Errs = append(g.Errs, "nothing checked: "+what+"; an empty check is a failure, not a pass")
	}
}

// CheckedExtra appends a preformatted segment to the checked: line. Unlike
// Count, the segment is emitted verbatim even when the numbers inside are
// zero: a zero that must stay visible in every run (per-pack boundary-event
// counts) would otherwise vanish with the zero-count suppression.
func (g *Gate) CheckedExtra(segment string) {
	g.checkedExtra = append(g.checkedExtra, segment)
}

// Emit prints the gate like Python (ERRS, DRIFT, warns, notes, checked:, ok).
// Returns the number of blocking findings (errs + drift).
func (g *Gate) Emit(out io.Writer) int {
	fmt.Fprintf(out, "== %s ==\n", g.Title)
	for _, e := range g.Errs {
		fmt.Fprintf(out, "  ERROR  %s\n", e)
	}
	for _, d := range g.Drift {
		fmt.Fprintf(out, "  DRIFT  %s\n", d)
	}
	for _, w := range g.Warns {
		fmt.Fprintf(out, "  warn   %s\n", w)
	}
	for _, a := range g.Notes {
		fmt.Fprintf(out, "  note   %s\n", a)
	}
	var parts []string
	for _, k := range g.countOrder {
		v := g.Counts[k]
		if v != 0 {
			parts = append(parts, fmt.Sprintf("%d %s", v, k))
		}
	}
	parts = append(parts, g.checkedExtra...)
	checked := strings.Join(parts, ", ")
	if checked == "" {
		fmt.Fprintln(out, "  checked: nothing")
	} else {
		fmt.Fprintf(out, "  checked: %s\n", checked)
	}
	if len(g.Errs) == 0 && len(g.Drift) == 0 && len(g.Warns) == 0 {
		fmt.Fprintln(out, "  ok")
	}
	return len(g.Errs) + len(g.Drift)
}

// --- helpers ---

func readOrEmpty(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// modeRe matches the BUILD.md template's mandatory mode declaration line.
var modeRe = regexp.MustCompile(`(?m)^Mode:\s*(full|manifest)\b`)

// headingContains reports whether any markdown heading line contains the
// (lowercase) needle.
func headingContains(text, needle string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "#") && strings.Contains(strings.ToLower(line), needle) {
			return true
		}
	}
	return false
}

// isPersisted reports whether a placement-table persistence cell means the
// machine's state survives the process (and so needs a migration protocol).
func isPersisted(cell string) bool {
	c := strings.ToLower(strings.TrimSpace(cell))
	switch {
	case c == "" || c == "-" || c == "none" || c == "n/a":
		return false
	case strings.Contains(c, "in-memory") || strings.Contains(c, "in memory") || strings.Contains(c, "ephemeral"):
		return false
	}
	return true
}

// sortedGlobExt lists <dir>/*<ext> sorted; empty when dir does not exist.
func sortedGlobExt(dir, ext string) []string {
	entries, _ := os.ReadDir(dir)
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ext) {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	return out
}

// tokenIn mirrors _token_in: whole-token containment (inv-1 must not match inv-12).
func tokenIn(token, text string) bool {
	// re.search(rf"(?<![A-Za-z0-9_-]){token}(?![A-Za-z0-9_-])", text)
	// Implement as a manual scan over token boundaries.
	idx := 0
	for {
		i := strings.Index(text[idx:], token)
		if i < 0 {
			return false
		}
		pos := idx + i
		beforeOK := pos == 0 || !isTokenChar(text[pos-1])
		afterOK := pos+len(token) == len(text) || !isTokenChar(text[pos+len(token)])
		if beforeOK && afterOK {
			return true
		}
		idx = pos + 1
	}
}

func isTokenChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '-' || b == '_'
}

// --- contract (G2-c4) ---

var (
	edgeRuleRe = regexp.MustCompile(`^"?\s*([^\s"]+)\s*->\s*([^\s"#]+)`)
	dslDeclRe  = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(person|softwareSystem|container|component)\b(.*)$`)
	// mitigation tokens allow hyphens inside segments: external ids like
	// external.rest-of-monolith must be able to satisfy their own row
	mitTokRe = regexp.MustCompile("`([A-Za-z_][A-Za-z0-9_-]*(?:\\.[A-Za-z_][A-Za-z0-9_-]*)*)`")
	kebabRe  = regexp.MustCompile("`([a-z][a-z0-9]*(?:-[a-z0-9]+)+)`")
)

func loadContract(archPath string, g *Gate) *ir.Value {
	if _, err := os.Stat(archPath); err != nil {
		g.Errs = append(g.Errs, filepath.Base(archPath)+" does not exist")
		return nil
	}
	data, err := os.ReadFile(archPath)
	if err != nil {
		g.Errs = append(g.Errs, filepath.Base(archPath)+" is unreadable: "+err.Error())
		return nil
	}
	fence, ok := ir.ContractFence(string(data))
	if !ok {
		g.Errs = append(g.Errs, "no Architecture Contract found (need a ```yaml fence under a heading containing 'Architecture Contract', starting with contract_version)")
		return nil
	}
	c, err := ir.LoadYAML([]byte(fence))
	if err != nil {
		g.Errs = append(g.Errs, "Architecture Contract is not valid YAML: "+err.Error())
		return nil
	}
	co := c.AsObject()
	if co == nil {
		g.Errs = append(g.Errs, "Architecture Contract yaml is not a mapping (got a list or scalar)")
		return nil
	}
	if _, ok := co.Get("contract_version"); !ok {
		g.Errs = append(g.Errs, "Architecture Contract has no contract_version")
		return nil
	}
	// the version's VALUE is checked, not just its presence: v1 contracts
	// mislabeled as current used to pass silently
	if v := co.Get2("contract_version"); v == nil || v.Kind != ir.KindNumber || string(v.AsNumber()) != "2" {
		got := "?"
		if v != nil {
			got = goReprValue(v)
		}
		g.Errs = append(g.Errs, "contract_version "+got+" is not supported; this toolchain checks contract v2 (element bindings, externals, ignore)")
		return nil
	}
	if co.Get2("boundaries") == nil {
		g.Errs = append(g.Errs, "Architecture Contract declares no boundaries")
		return nil
	}
	return c
}

// contractEdges parses src->dst rules. g may be nil (a G4-only run parses the
// rules a second time and must not duplicate G2's findings): unparseable rules
// are then skipped silently here because G2 reports them.
func contractEdges(rules *ir.Object, key string, g *Gate) [][2]string {
	var out [][2]string
	v := rules.Get2(key)
	if v == nil {
		return out
	}
	for _, e := range v.AsArray() {
		s := e.AsString()
		m := edgeRuleRe.FindStringSubmatch(s)
		if m != nil {
			out = append(out, [2]string{m[1], m[2]})
		} else if g != nil {
			g.Errs = append(g.Errs, fmt.Sprintf("unparseable %s rule: %s (expected 'src -> dst')", key, ir.Repr(s)))
		}
	}
	return out
}

func dslElements(dslPath string) map[string]dslEl {
	els := map[string]dslEl{}
	text := readOrEmpty(dslPath)
	for _, line := range strings.Split(text, "\n") {
		m := dslDeclRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name, kind, rest := m[1], m[2], m[3]
		args := quoteStrings(rest)
		tagIdx := 3
		if kind == "person" || kind == "softwareSystem" {
			tagIdx = 2
		}
		tags := map[string]bool{}
		if len(args) > tagIdx {
			for _, t := range strings.Split(args[tagIdx], ",") {
				tags[strings.TrimSpace(t)] = true
			}
		}
		display := name
		if len(args) > 0 {
			display = args[0]
		}
		els[name] = dslEl{Kind: kind, Tags: tags, Display: display}
	}
	return els
}

type dslEl struct {
	Kind    string
	Tags    map[string]bool
	Display string
}

func quoteStrings(s string) []string {
	var out []string
	inQuote := false
	var cur strings.Builder
	for _, r := range s {
		if r == '"' {
			if inQuote {
				out = append(out, cur.String())
				cur.Reset()
			}
			inQuote = !inQuote
		} else if inQuote {
			cur.WriteRune(r)
		}
	}
	return out
}

// CheckC4 implements G2-c4.
func CheckC4(design string) *Gate {
	g := NewGate("G2-c4  Architecture Contract")
	g.startOrder()
	arch := filepath.Join(design, "ARCHITECTURE.md")
	c := loadContract(arch, g)
	if c == nil {
		return g
	}
	co := c.AsObject()
	boundaries := objSlice(co.Get2("boundaries"))
	externals := objSlice(co.Get2("externals"))

	var ids []string
	for _, b := range boundaries {
		bo := b.AsObject()
		if bo == nil || bo.GetString("id") == "" {
			g.Errs = append(g.Errs, "boundary without an id: "+goReprValue(b))
			continue
		}
		ids = append(ids, bo.GetString("id"))
		g.Count("boundaries")
		if bo.Get2("code") == nil {
			g.Errs = append(g.Errs, fmt.Sprintf("boundary %s declares no code globs; G4 cannot map it", ir.Repr(bo.GetString("id"))))
		}
	}
	for _, bid := range uniqueDuplicates(ids) {
		if countStr(ids, bid) > 1 {
			g.Errs = append(g.Errs, fmt.Sprintf("duplicate boundary id %s", ir.Repr(bid)))
		}
	}
	var extIDs []string
	for _, x := range externals {
		xo := x.AsObject()
		if xo == nil || xo.GetString("id") == "" {
			g.Errs = append(g.Errs, "externals entry without an id: "+goReprValue(x))
			continue
		}
		extIDs = append(extIDs, xo.GetString("id"))
		g.Count("externals")
	}
	declared := setOf(ids)
	for _, x := range extIDs {
		declared[x] = true
	}

	rules := co.GetObject("dependency_rules")
	if rules == nil {
		rules = ir.NewObject()
	}
	allow := contractEdges(rules, "allow", g)
	deny := contractEdges(rules, "deny", g)
	baseline := contractEdges(rules, "baseline", g)
	g.Count("allow rules", len(allow))
	g.Count("deny rules", len(deny))
	g.Count("baseline rules", len(baseline))
	all := append(append(append([][2]string{}, allow...), deny...), baseline...)
	for _, e := range all {
		for _, side := range e {
			if strings.Contains(side, "*") {
				continue
			}
			if !declared[side] {
				hint := ""
				if strings.HasPrefix(side, "external") {
					hint = " (declare it under externals:)"
				}
				g.Errs = append(g.Errs, fmt.Sprintf("rule references undeclared boundary %s%s", ir.Repr(side), hint))
			}
		}
	}
	denySet := edgeSet(deny)
	for _, e := range allow { // slice order, not map order: deterministic output
		if denySet[e] {
			g.Errs = append(g.Errs, fmt.Sprintf("edge %s -> %s is both allowed and denied", e[0], e[1]))
			denySet[e] = false // report each edge once even if listed twice
		}
	}
	// allow+baseline contradicts (an allowed edge is not a violation, so there
	// is nothing to amnesty); deny+baseline is legitimate and recommended: the
	// deny records the intent, the baseline records the tolerated debt
	baseSet := edgeSet(baseline)
	for _, e := range allow {
		if baseSet[e] {
			g.Errs = append(g.Errs, fmt.Sprintf("edge %s -> %s is both allowed and baselined; baseline marks tolerated violations, an allowed edge needs no amnesty", e[0], e[1]))
			baseSet[e] = false
		}
	}

	dslPath := filepath.Join(design, "workspace.dsl")
	if _, err := os.Stat(dslPath); err != nil {
		g.Errs = append(g.Errs, "workspace.dsl does not exist; the contract has no model to bind to")
		return g
	}
	els := dslElements(dslPath)
	g.Count("dsl elements", len(els))
	if len(els) == 0 {
		g.Errs = append(g.Errs, "workspace.dsl parsed but no elements found")
	}

	for _, b := range boundaries {
		bo := b.AsObject()
		if bo == nil || bo.GetString("id") == "" {
			continue
		}
		el := bo.GetString("element")
		if el == "" {
			el = lastSegment(bo.GetString("id"))
		}
		if _, ok := els[el]; !ok {
			g.Errs = append(g.Errs, fmt.Sprintf("boundary %s maps to no workspace.dsl element (looked for %s; set element: explicitly if the id differs)", ir.Repr(bo.GetString("id")), ir.Repr(el)))
		} else {
			g.Count("boundaries bound to dsl")
		}
	}
	for _, x := range externals {
		xo := x.AsObject()
		if xo == nil || xo.GetString("id") == "" {
			continue
		}
		el := xo.GetString("element")
		if el != "" {
			if _, ok := els[el]; !ok {
				g.Errs = append(g.Errs, fmt.Sprintf("external %s maps to element %s not in workspace.dsl", ir.Repr(xo.GetString("id")), ir.Repr(el)))
			}
		}
	}

	// mitigation coverage
	required := map[string]string{}
	for _, x := range externals {
		xo := x.AsObject()
		if xo != nil && xo.GetString("id") != "" {
			required[xo.GetString("id")] = xo.GetString("element")
		}
	}
	infraTags := map[string]bool{"Database": true, "Queue": true, "External": true}
	infra := map[string]bool{}
	for name, e := range els {
		for t := range e.Tags {
			if infraTags[t] {
				infra[name] = true
			}
		}
	}
	text := readOrEmpty(arch)
	var mitRows [][]string
	for _, tbl := range ir.ParseMdTables(text) {
		hl := strings.ToLower(strings.Join(tbl.Header, " "))
		if strings.Contains(hl, "failure") && strings.Contains(hl, "mitigation") {
			mitRows = tbl.Rows
			break
		}
	}
	covered := map[string]bool{}
	for _, r := range mitRows {
		if len(r) == 0 {
			continue
		}
		g.Count("mitigation rows")
		for _, m := range mitTokRe.FindAllStringSubmatch(r[0], -1) {
			tok := m[1]
			if _, ok := els[tok]; ok {
				covered[tok] = true
			} else if _, ok := required[tok]; ok {
				covered[tok] = true
			} else {
				g.Errs = append(g.Errs, "mitigation row names `"+tok+"`, which is neither a workspace.dsl element nor a declared external")
			}
		}
	}
	need := map[string]bool{}
	for k := range required {
		need[k] = true
	}
	for k := range infra {
		need[k] = true
	}
	if len(need) > 0 && len(mitRows) == 0 {
		g.Errs = append(g.Errs, "no mitigation table found (header needs 'failure' and 'mitigation' columns) although the design declares infrastructure dependencies")
	}
	var needSorted []string
	for k := range need {
		needSorted = append(needSorted, k)
	}
	sort.Strings(needSorted)
	for _, dep := range needSorted {
		alt := required[dep]
		if covered[dep] || (alt != "" && covered[alt]) {
			g.Count("dependencies with mitigation rows")
		} else {
			g.Errs = append(g.Errs, "infrastructure dependency `"+dep+"` has no mitigation row (name it in the first column, backticked)")
		}
	}

	// bus/queue coupling is invisible to G4-import; the event-contract table
	// is the governing artifact, so a queue-coupled design must have one
	queueCoupled := false
	for name, e := range els {
		if e.Tags["Queue"] {
			queueCoupled = true
			_ = name
		}
	}
	if queueCoupled {
		found := false
		for _, tbl := range ir.ParseMdTables(text) {
			hl := strings.ToLower(strings.Join(tbl.Header, " "))
			if strings.Contains(hl, "producer") && strings.Contains(hl, "consumer") && strings.Contains(hl, "delivery") {
				found = true
				g.Count("event contracts", len(tbl.Rows))
				break
			}
		}
		if !found {
			g.Errs = append(g.Errs, "the model has a Queue-tagged element but ARCHITECTURE.md has no event-contract table (columns: producer, consumer, payload, delivery, ordering, dedupe); bus coupling is invisible to G4-import, so this table is its governing artifact")
		} else {
			// a header with zero rows governs nothing; the queue coupling is
			// then entirely unchecked
			g.RequireNonzero("event contracts", "the event-contract table has no rows although the model has a Queue-tagged element")
		}
	}

	g.RequireNonzero("boundaries", "no boundaries parsed")
	g.RequireNonzero("dsl elements", "no workspace.dsl elements parsed")
	return g
}

func objSlice(v *ir.Value) []*ir.Value {
	if v == nil || v.Kind != ir.KindArray {
		return nil
	}
	return v.AsArray()
}

func goReprValue(v *ir.Value) string { return ir.Repr(v) }

func edgeSet(edges [][2]string) map[[2]string]bool {
	m := map[[2]string]bool{}
	for _, e := range edges {
		m[e] = true
	}
	return m
}

// uniqueDuplicates returns the unique values in first-occurrence order (the
// caller filters to count>1); deterministic so error order never varies.
func uniqueDuplicates(xs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func countStr(xs []string, x string) int {
	n := 0
	for _, e := range xs {
		if e == x {
			n++
		}
	}
	return n
}

func setOf(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func lastSegment(s string) string {
	if i := strings.LastIndex(s, "."); i >= 0 {
		return s[i+1:]
	}
	return s
}

// --- machines (G3-machine) ---

// CheckMachines implements G3-machine.
func CheckMachines(design string) *Gate {
	g := NewGate("G3-machine  machines + oracle")
	g.startOrder()
	mdir := filepath.Join(design, "machines")
	files := sortedGlob(mdir, "*.machine.json")
	if len(files) == 0 {
		g.Errs = append(g.Errs, "no *.machine.json under "+mdir)
		return g
	}
	for _, path := range files {
		base := filepath.Base(path)
		m, err := ir.LoadMachineJSON(path)
		if err != nil {
			g.Errs = append(g.Errs, err.Error())
			continue
		}
		g.Count("machines")
		errs, warns, notes, counts := lint.LintMachine(m, base)
		g.Errs = append(g.Errs, errs...)
		g.Warns = append(g.Warns, warns...)
		g.Notes = append(g.Notes, notes...)
		g.Count("transitions", counts.Transitions)

		var exhaustive []string
		for _, s := range ir.WalkStates(m.AsObject().Get2("states"), "") {
			if s.Node != nil && s.Node.Kind == ir.KindObject {
				note := strings.TrimSpace(s.Node.AsObject().GetString("_exhaustive"))
				if note != "" {
					exhaustive = append(exhaustive, s.Name)
				}
			}
		}
		if len(exhaustive) > 0 {
			g.Count("states relying on unproven _exhaustive liveness", len(exhaustive))
			sort.Strings(exhaustive)
			g.Warns = append(g.Warns, base+": liveness for "+strings.Join(exhaustive, ", ")+
				" rests on an UNPROVEN _exhaustive claim (guards are erased in the TLA+ model, so TLC cannot check it); verify the guard set is provably total, or add an unguarded fallback branch so the liveness proof becomes sound")
		}

		opath := machineSibling(path, ".oracle.md")
		fresh := oracle.Render(m, path)
		if _, err := os.Stat(opath); err != nil {
			g.Errs = append(g.Errs, base+": no committed oracle ("+filepath.Base(opath)+"); run machinery oracle")
		} else if committed, err := os.ReadFile(opath); err != nil {
			g.Errs = append(g.Errs, base+": committed oracle is unreadable: "+err.Error())
		} else if string(committed) != fresh {
			g.Drift = append(g.Drift, base+": committed oracle is stale (differs from a fresh generation); rerun machinery oracle")
		} else {
			g.Count("oracles fresh")
		}

		mpath := machineSibling(path, ".matrix.md")
		if _, err := os.Stat(mpath); err != nil {
			g.Warns = append(g.Warns, base+": no matrix file; named-unit contracts are unchecked (transitions are covered by the generated oracle)")
			continue
		}
		mtext := readOrEmpty(mpath)
		merrs, drift, nrows := lint.ReconcileMatrix(m, mtext, base)
		g.Errs = append(g.Errs, merrs...)
		g.Drift = append(g.Drift, drift...)
		g.Count("matrix rows reconciled", nrows)
		declared := lint.NamedUnitNames(mtext)
		guards, actions, actors := lint.MachineUnitNames(m)
		for _, pair := range []struct {
			kind  string
			names map[string]bool
		}{{"guard", guards}, {"action", actions}, {"actor", actors}} {
			var sorted []string
			for n := range pair.names {
				sorted = append(sorted, n)
			}
			sort.Strings(sorted)
			for _, name := range sorted {
				if declared[name] {
					g.Count("named units covered")
				} else {
					g.Drift = append(g.Drift, base+": "+pair.kind+" "+ir.Repr(name)+" has no named-unit contract row in "+filepath.Base(mpath))
				}
			}
		}
	}
	g.RequireNonzero("machines", "no machines parsed")
	g.RequireNonzero("transitions", "no transitions parsed")
	return g
}

func sortedGlob(dir, pattern string) []string {
	entries, _ := os.ReadDir(dir)
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			matched, _ := filepath.Match(pattern, e.Name())
			if matched {
				out = append(out, filepath.Join(dir, e.Name()))
			}
		}
	}
	sort.Strings(out)
	return out
}

// machineSibling swaps a *.machine.json path's suffix for a sibling
// artifact's (".oracle.md", ".matrix.md").
func machineSibling(path, newExt string) string {
	return path[:len(path)-len(".machine.json")] + newExt
}

// --- traceability (Gx-trace) ---

func loadModelith(design string, g *Gate) *ir.Value {
	entries, _ := os.ReadDir(design)
	var paths []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".modelith.yaml") {
			paths = append(paths, filepath.Join(design, e.Name()))
		}
	}
	if len(paths) == 0 {
		g.Errs = append(g.Errs, "no *.modelith.yaml in the design directory")
		return nil
	}
	if len(paths) > 1 {
		var names []string
		for _, p := range paths {
			names = append(names, filepath.Base(p))
		}
		sort.Strings(names)
		g.Errs = append(g.Errs, "multiple modelith models: "+strings.Join(names, ", "))
		return nil
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		g.Errs = append(g.Errs, filepath.Base(paths[0])+": invalid YAML: "+err.Error())
		return nil
	}
	v, err := ir.LoadYAML(data)
	if err != nil {
		g.Errs = append(g.Errs, filepath.Base(paths[0])+": invalid YAML: "+err.Error())
		return nil
	}
	if v.AsObject() == nil {
		g.Errs = append(g.Errs, filepath.Base(paths[0])+": not a yaml mapping (empty file?)")
		return nil
	}
	return v
}

// CheckTraceability implements Gx-trace.
func CheckTraceability(design string) *Gate {
	g := NewGate("Gx-trace  cross-layer traceability")
	g.startOrder()
	dm := loadModelith(design, g)
	if dm == nil {
		return g
	}
	dmo := dm.AsObject()
	enums := map[string][]string{}
	if ev := dmo.Get2("enums"); ev != nil {
		for _, k := range ev.AsObject().Keys() {
			var vals []string
			for _, vv := range ev.AsObject().Get2(k).AsObject().Get2("values").AsArray() {
				vals = append(vals, vv.AsObject().GetString("name"))
			}
			enums[k] = vals
		}
	}
	entities := dmo.GetObject("entities")
	if entities.Len() == 0 {
		g.Errs = append(g.Errs, "modelith model declares no entities; an empty domain model is a failure, not a pass")
		return g
	}
	g.Count("entities", entities.Len())

	invIDs := map[string]bool{}
	for _, i := range objSlice(dmo.Get2("invariants")) {
		invIDs[i.AsObject().GetString("id")] = true
	}
	actionsByEntity := map[string]map[string]bool{}
	enumByEntity := map[string]string{}
	allActions := map[string]bool{}
	for _, ename := range entities.Keys() {
		e := entities.Get2(ename).AsObject()
		acts := map[string]bool{}
		for _, a := range objSlice(e.Get2("actions")) {
			if a.Kind == ir.KindObject {
				acts[a.AsObject().GetString("name")] = true
			} else {
				acts[a.AsString()] = true
			}
		}
		actionsByEntity[ename] = acts
		for a := range acts {
			allActions[a] = true
		}
		for _, i := range objSlice(e.Get2("invariants")) {
			invIDs[i.AsObject().GetString("id")] = true
		}
		for _, a := range objSlice(e.Get2("attributes")) {
			ao := a.AsObject()
			if _, ok := enums[ao.GetString("type")]; ok {
				name := ao.GetString("name")
				if name == "status" || name == "stage" || name == "state" {
					enumByEntity[ename] = ao.GetString("type")
				}
			}
		}
	}

	mdir := filepath.Join(design, "machines")
	machineFiles := sortedGlob(mdir, "*.machine.json")
	if len(machineFiles) == 0 {
		g.Errs = append(g.Errs, "no machines under "+mdir+" to trace")
		return g
	}

	machineNames := map[string]bool{}
	claimed := map[string]bool{}
	for _, path := range machineFiles {
		base := filepath.Base(path)
		name := strings.TrimSuffix(base, ".machine.json")
		machineNames[name] = true
		m, err := ir.LoadMachineJSON(path)
		if err != nil {
			g.Errs = append(g.Errs, err.Error())
			continue
		}
		mo := m.AsObject()
		entity := mo.GetString("_lifecycle_of")
		if entity == "" {
			if _, ok := entities.Get(name); ok {
				entity = name
			}
		}
		role := mo.GetString("_role")
		if role == "operational" {
			entity = ""
		}
		if entity == "" && role != "operational" {
			g.Errs = append(g.Errs, base+": maps to no Modelith entity and is not declared operational (set _lifecycle_of: <Entity> or _role: operational)")
			continue
		}
		if role == "operational" {
			g.Count("operational machines")
			continue
		}
		if _, ok := entities.Get(entity); !ok {
			g.Errs = append(g.Errs, base+": _lifecycle_of "+ir.Repr(entity)+" is not a Modelith entity")
			continue
		}
		claimed[entity] = true
		enumName := enumByEntity[entity]
		if enumName == "" {
			g.Errs = append(g.Errs, base+": entity "+ir.Repr(entity)+" has no enum-typed lifecycle attribute")
			continue
		}
		vals := setOf(enums[enumName])
		var top []string
		for _, s := range ir.WalkStates(mo.Get2("states"), "") {
			if !strings.Contains(s.Path, ".") {
				top = append(top, s.Name)
			}
		}
		domainStates := map[string]bool{}
		var overlay []string
		for _, n := range top {
			if ir.IsUpperFirst(n) {
				domainStates[n] = true
			} else {
				overlay = append(overlay, n)
			}
		}
		var diff []string
		for s := range domainStates {
			if !vals[s] {
				diff = append(diff, s)
			}
		}
		sort.Strings(diff)
		for _, s := range diff {
			g.Errs = append(g.Errs, base+": domain state "+ir.Repr(s)+" is not a value of enum "+enumName+" (overlay states are lowerCamel by convention)")
		}
		var diff2 []string
		for v := range vals {
			if !domainStates[v] {
				diff2 = append(diff2, v)
			}
		}
		sort.Strings(diff2)
		for _, v := range diff2 {
			g.Errs = append(g.Errs, base+": enum "+enumName+" value "+ir.Repr(v)+" has no machine state; the lifecycle is incomplete")
		}
		if len(overlay) > 0 {
			g.Notes = append(g.Notes, name+": "+fmt.Sprintf("%d", len(overlay))+" operational-overlay states ("+strings.Join(overlay, ", ")+")")
		}
		events := map[string]bool{}
		for _, s := range ir.WalkStates(mo.Get2("states"), "") {
			if on := s.Node.AsObject().Get2("on"); on != nil {
				for _, ev := range on.AsObject().Keys() {
					events[ev] = true
				}
			}
		}
		var evSorted []string
		for ev := range events {
			evSorted = append(evSorted, ev)
		}
		sort.Strings(evSorted)
		for _, ev := range evSorted {
			if !actionsByEntity[entity][ev] {
				g.Errs = append(g.Errs, base+": event "+ir.Repr(ev)+" is not a Modelith action of "+entity)
			}
		}
		g.Count("lifecycle machines traced")
	}

	var enumEntities []string
	for ename := range enumByEntity {
		enumEntities = append(enumEntities, ename)
	}
	sort.Strings(enumEntities)
	for _, ename := range enumEntities {
		enumName := enumByEntity[ename]
		if !claimed[ename] {
			g.Errs = append(g.Errs, "entity "+ename+" has lifecycle enum "+enumName+" but no machine ("+ename+".machine.json); model the lifecycle or drop the enum")
		} else {
			g.Count("lifecycle entities with machines")
		}
	}

	// placement table
	persistedPlacements := 0
	arch := filepath.Join(design, "ARCHITECTURE.md")
	if _, err := os.Stat(arch); err == nil {
		text := readOrEmpty(arch)
		for _, tbl := range ir.ParseMdTables(text) {
			hl := strings.ToLower(strings.Join(tbl.Header, " "))
			if strings.Contains(hl, "placement") && strings.Contains(hl, "persistence") {
				pi := ir.FindCol(tbl.Header, "persistence")
				for _, r := range tbl.Rows {
					if len(r) == 0 {
						continue
					}
					if pi >= 0 && pi < len(r) && isPersisted(r[pi]) {
						persistedPlacements++
					}
					// only a backticked token names a component; a bare word is
					// prose, and accepting it lets de-backticked rows pass silently
					var named []string
					if bt := backtickTokens(r[0]); len(bt) > 0 {
						named = []string{bt[0]}
					}
					if len(named) == 0 {
						g.Errs = append(g.Errs, "placement row names no component in backticks: "+ir.Repr(r[0]))
					}
					for _, comp := range named {
						if machineNames[comp] {
							g.Count("placement rows with machines")
						} else if strings.Contains(strings.Join(r, " "), "(no machine:") {
							g.Count("placement rows waived")
						} else {
							g.Errs = append(g.Errs, "placement row component `"+comp+"` has no machine and no '(no machine: <reason>)' waiver")
						}
					}
				}
				break
			}
		}
	}

	// invariants enforcement
	var cells, unitCells []string
	matrixFiles := globExt(mdir, ".matrix.md")
	for _, f := range matrixFiles {
		for _, tbl := range ir.ParseMdTables(readOrEmpty(f)) {
			mi := ir.FindCol(tbl.Header, "maps to")
			for _, r := range tbl.Rows {
				cells = append(cells, r...)
				if mi >= 0 && mi < len(r) {
					unitCells = append(unitCells, r[mi])
				}
			}
		}
	}
	build := filepath.Join(design, "BUILD.md")
	if _, err := os.Stat(build); err == nil {
		buildText := readOrEmpty(build)
		for _, tbl := range ir.ParseMdTables(buildText) {
			for _, r := range tbl.Rows {
				cells = append(cells, r...)
			}
		}
		// manifest mode shards contribute to the traceability corpus too
		for _, shard := range sortedGlobExt(filepath.Join(design, "BUILD"), ".md") {
			for _, tbl := range ir.ParseMdTables(readOrEmpty(shard)) {
				for _, r := range tbl.Rows {
					cells = append(cells, r...)
				}
			}
			g.Count("build shards scanned")
		}
		// template conformance: deterministic structural requirements
		if modeRe.FindString(buildText) == "" {
			g.Errs = append(g.Errs, "BUILD.md declares no mode; the template requires a top 'Mode: full ...' or 'Mode: manifest ...' line")
		} else {
			g.Count("build mode declared")
		}
		if !headingContains(buildText, "toolchain") {
			g.Errs = append(g.Errs, "BUILD.md has no Toolchain heading; two implementing agents must not diverge on environment (pin language, libraries, test framework)")
		} else {
			g.Count("toolchain section present")
		}
		if persistedPlacements > 0 {
			if !headingContains(buildText, "state migration") {
				g.Errs = append(g.Errs, fmt.Sprintf("%d placement row(s) persist machine state but BUILD.md has no State migration heading (mapping table or drain rule for future state changes, or 'no persisted instances yet')", persistedPlacements))
			} else {
				g.Count("state-migration section present")
			}
		}
	} else {
		g.Warns = append(g.Warns, "BUILD.md not present; invariant enforcement is checked against the matrices only (fine before Phase 4)")
	}
	corpus := strings.Join(cells, "\n")
	unitCorpus := strings.Join(unitCells, "\n")
	if len(invIDs) == 0 {
		g.Errs = append(g.Errs, "the domain model declares no invariants; nothing constrains the design")
	}
	// the relational policy model is an enforcement artifact too: an
	// invariant compiled into Policy.als is solver-checked, which is
	// stronger than a prose reference (Gp-policy holds the annotation to
	// the domain model; here it only credits coverage)
	policyIDs := alloy.CarriedIDs(filepath.Join(design, "formal", alloy.AnnotationName))
	integrityIDs := alloy.CarriedIntegrityIDs(filepath.Join(design, "formal", alloy.IntegrityAnnotationName))
	isolationIDs := alloy.CarriedIsolationIDs(filepath.Join(design, "formal", alloy.IsolationAnnotationName))
	var invSorted []string
	for iid := range invIDs {
		invSorted = append(invSorted, iid)
	}
	sort.Strings(invSorted)
	for _, iid := range invSorted {
		if tokenIn(iid, corpus) || policyIDs[iid] || integrityIDs[iid] || isolationIDs[iid] {
			g.Count("invariants enforced")
			switch {
			case tokenIn(iid, unitCorpus):
				g.Count("invariants unit-backed (guard/action/actor)")
			case policyIDs[iid]:
				g.Count("invariants policy-checked (relational model)")
			case integrityIDs[iid]:
				g.Count("invariants integrity-checked (relational model)")
			case isolationIDs[iid]:
				g.Count("invariants isolation-checked (relational model)")
			default:
				g.Count("invariants attested (structural/prose)")
			}
		} else {
			g.Errs = append(g.Errs, "invariant "+ir.Repr(iid)+" is referenced by no matrix or BUILD.md table; it is enforced by nothing")
		}
	}

	// orphan references
	known := map[string]bool{}
	for iid := range invIDs {
		known[iid] = true
	}
	for a := range allActions {
		known[a] = true
	}
	for _, ename := range entities.Keys() {
		known[ename] = true
	}
	for _, vs := range enums {
		for _, v := range vs {
			known[v] = true
		}
	}
	for _, f := range matrixFiles {
		base := filepath.Base(f)
		for _, tbl := range ir.ParseMdTables(readOrEmpty(f)) {
			mi := ir.FindCol(tbl.Header, "maps to")
			if mi < 0 {
				continue
			}
			for _, r := range tbl.Rows {
				if mi >= len(r) {
					continue
				}
				for _, m := range kebabRe.FindAllStringSubmatch(r[mi], -1) {
					if !known[m[1]] {
						g.Drift = append(g.Drift, base+": maps-to references `"+m[1]+"`, which is not a declared invariant (typo or a stale reference)")
					}
				}
			}
		}
	}
	g.RequireNonzero("invariants enforced", "no invariant was traced to an enforcement artifact")
	return g
}

func backtickTokens(s string) []string {
	re := regexp.MustCompile("`([A-Za-z_][A-Za-z0-9_]*)`")
	var out []string
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		out = append(out, m[1])
	}
	return out
}

func globExt(dir, ext string) []string {
	entries, _ := os.ReadDir(dir)
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ext) {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	return out
}
