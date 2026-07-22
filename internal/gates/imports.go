package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/ir"
)

// --- imports (G4-import) ---

// walkSourceFiles collects source files under root, FOLLOWING directory
// symlinks (the Python glob("**") did; monorepo/pnpm/bazel layouts satisfy
// boundary code globs via symlinks, and skipping them makes the code
// invisible to the gate). Cycles are broken by tracking resolved paths.
// Dangling symlinks are skipped.
func walkSourceFiles(root string) ([]string, error) {
	var files []string
	visited := map[string]bool{}
	var walk func(dir string) error
	walk = func(dir string) error {
		real, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return err
		}
		if visited[real] {
			return nil // symlink cycle
		}
		visited[real] = true
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			p := filepath.Join(dir, e.Name())
			if e.Type()&os.ModeSymlink != 0 {
				fi, statErr := os.Stat(p) // follow the link
				if statErr != nil {
					continue // dangling symlink
				}
				if fi.IsDir() {
					if err := walk(p); err != nil {
						return err
					}
					continue
				}
			} else if e.IsDir() {
				if err := walk(p); err != nil {
					return err
				}
				continue
			}
			if _, ok := langExts[filepath.Ext(p)]; ok {
				files = append(files, p)
			} else if isTestFile(e.Name()) {
				// *.test.mjs / *.test.cjs: test files in extensions langExts
				// never maps for import parsing; Gt still needs them walked
				files = append(files, p)
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return files, err
	}
	return files, nil
}

// testFilePatterns, isTestFile, and rustSplitTests are the ONE test-file
// classifier, shared by G4 (which SKIPS test files, per its documented
// semantics) and Gt (which scans exactly the files G4 skips): the two gates
// can never disagree about what a test file is.
var testFilePatterns = []string{"*_test.go", "*_test.py", "test_*.py", "*.test.ts", "*.test.tsx",
	"*.test.js", "*.test.jsx", "*.test.mjs", "*.test.cjs", "*_test.exs", "*_spec.rb"}

func isTestFile(rel string) bool {
	base := filepath.Base(rel)
	for _, p := range testFilePatterns {
		if ok, _ := filepath.Match(p, base); ok {
			return true
		}
	}
	// a .rs file is test-only iff it lives under a tests/ or benches/
	// directory (Rust integration tests and benchmarks), at any depth;
	// every other .rs file is production code and rustSplitTests carves its
	// #[cfg(test)] spans out for Gt
	if strings.HasSuffix(base, ".rs") {
		for _, seg := range strings.Split(filepath.ToSlash(filepath.Dir(rel)), "/") {
			if seg == "tests" || seg == "benches" {
				return true
			}
		}
	}
	return false
}

// cfgTestAttr marks the Rust unit-test idiom: a #[cfg(test)] item (usually
// `mod tests { ... }`) inside a production .rs file.
const cfgTestAttr = "#[cfg(test)]"

// rustSplitTests carves a .rs file into its production text and its
// #[cfg(test)] item spans. A .rs file is test-only iff it lives under tests/
// or benches/ (isTestFile); every OTHER .rs file is production code: G4 scans
// the production portion (a wholesale skip made production imports invisible,
// NG-1) and Gt's corpus receives ONLY the cfg(test) spans, where Rust unit
// tests actually live (production text once wholesale-covered oracles, NG-7).
// The production text keeps its line structure: span bytes become spaces.
func rustSplitTests(text string) (string, []string) {
	var spans [][2]int
	for i := 0; i < len(text); {
		j := strings.Index(text[i:], cfgTestAttr)
		if j < 0 {
			break
		}
		start := i + j
		end := rustItemEnd(text, start+len(cfgTestAttr))
		spans = append(spans, [2]int{start, end})
		i = end
	}
	if len(spans) == 0 {
		return text, nil
	}
	prod := []byte(text)
	var tests []string
	for _, sp := range spans {
		tests = append(tests, text[sp[0]:sp[1]])
		for k := sp[0]; k < sp[1]; k++ {
			if prod[k] != '\n' {
				prod[k] = ' '
			}
		}
	}
	return string(prod), tests
}

// rustItemEnd returns the end offset of the item that follows a #[cfg(test)]
// attribute: the matching close brace of its first block, the terminating
// semicolon of a braceless item, or end of text. Brace counting is
// lint-grade (a brace inside a string literal would miscount), matching the
// regex-grade parsing of every other language here.
func rustItemEnd(text string, from int) int {
	depth := 0
	for i := from; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth <= 0 {
				return i + 1
			}
		case ';':
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(text)
}

func matchGlob(rel, pattern string) bool {
	pattern = strings.TrimSuffix(pattern, "/")
	if ok, _ := filepath.Match(pattern, rel); ok {
		return true
	}
	static := strings.ReplaceAll(pattern, "/**", "")
	static = strings.ReplaceAll(static, "/*", "")
	static = strings.TrimSuffix(static, "/")
	return rel == static || strings.HasPrefix(rel, static+"/")
}

func boundaryOf(rel string, pkgmap [][2]string) string {
	best := -1
	var bestBid string
	for _, pm := range pkgmap {
		pattern, bid := pm[0], pm[1]
		if matchGlob(rel, pattern) {
			static := strings.ReplaceAll(pattern, "/**", "")
			static = strings.ReplaceAll(static, "/*", "")
			if len(static) > best {
				best = len(static)
				bestBid = bid
			}
		}
	}
	return bestBid
}

func goModuleName(impl string) string {
	data, err := os.ReadFile(filepath.Join(impl, "go.mod"))
	if err != nil {
		return ""
	}
	m := regexp.MustCompile(`(?m)^module\s+(\S+)`).FindSubmatch(data)
	if m != nil {
		return string(m[1])
	}
	return ""
}

var (
	goBlockImportRe = regexp.MustCompile(`(?ms)^import\s*\((.*?)\)`)
	goLineImportRe  = regexp.MustCompile(`(?m)^import\s+(?:[\w.]+\s+)?"([^"]+)"`)
	goBlockLineRe   = regexp.MustCompile(`(?:^|\s)(?:[\w.]+\s+)?"([^"]+)"`)
	pyImportRe      = regexp.MustCompile(`(?m)^\s*import\s+([\w.]+(?:\s+as\s+\w+)?(?:\s*,\s*[\w.]+(?:\s+as\s+\w+)?)*)`)
	pyFromRe        = regexp.MustCompile(`(?m)^\s*from\s+([\w.]+)\s+import\b`)
	// from/import/dynamic import()/require() forms
	tsImportRe    = regexp.MustCompile(`(?:from|import\s*\(|import|require\()\s*['"]([^'"]+)['"]`)
	exModRe       = regexp.MustCompile(`(?m)^\s*(?:alias|import|use|require)\s+([A-Z][\w.]*)`)
	rustUseRe     = regexp.MustCompile(`(?m)^\s*use\s+([\w:]+)`)
	exDefmoduleRe = regexp.MustCompile(`(?m)^\s*defmodule\s+([A-Z][\w.]*)`)
)

func goImports(text string) []string {
	var out []string
	for _, blk := range goBlockImportRe.FindAllStringSubmatch(text, -1) {
		for _, m := range goBlockLineRe.FindAllStringSubmatch(blk[1], -1) {
			out = append(out, m[1])
		}
	}
	for _, m := range goLineImportRe.FindAllStringSubmatch(text, -1) {
		out = append(out, m[1])
	}
	return out
}

func pyImports(text, rel string) []string {
	var out []string
	for _, m := range pyImportRe.FindAllStringSubmatch(text, -1) {
		// "import a as x, b.c" imports every listed module, not just the first
		for _, part := range strings.Split(m[1], ",") {
			mod := strings.Fields(strings.TrimSpace(part))[0]
			out = append(out, strings.ReplaceAll(mod, ".", "/"))
		}
	}
	for _, m := range pyFromRe.FindAllStringSubmatch(text, -1) {
		mod := m[1]
		if strings.HasPrefix(mod, ".") {
			base := filepath.Dir(rel)
			for i := 1; i < len(mod)-len(strings.TrimLeft(mod, ".")); i++ {
				base = filepath.Dir(base)
			}
			mod = filepath.Join(base, strings.ReplaceAll(strings.TrimLeft(mod, "."), ".", "/"))
			mod = strings.Trim(mod, "/")
			out = append(out, mod)
		} else {
			out = append(out, strings.ReplaceAll(mod, ".", "/"))
		}
	}
	return out
}

func tsImports(text, rel string) []string {
	var out []string
	for _, m := range tsImportRe.FindAllStringSubmatch(text, -1) {
		spec := m[1]
		if strings.HasPrefix(spec, ".") {
			joined, _ := filepath.Abs(filepath.Join(filepath.Dir(rel), spec))
			_ = joined
			out = append(out, filepath.Clean(filepath.Join(filepath.Dir(rel), spec)))
		} else {
			out = append(out, spec)
		}
	}
	return out
}

func exModules(text string) []string {
	var out []string
	for _, m := range exModRe.FindAllStringSubmatch(text, -1) {
		out = append(out, m[1])
	}
	return out
}

func rustImports(text string) []string {
	var out []string
	for _, m := range rustUseRe.FindAllStringSubmatch(text, -1) {
		path := m[1]
		if strings.HasPrefix(path, "crate::") {
			out = append(out, "src/"+strings.ReplaceAll(path[len("crate::"):], "::", "/"))
		} else {
			out = append(out, strings.SplitN(path, "::", 2)[0])
		}
	}
	return out
}

var langExts = map[string]string{
	".go": "go", ".py": "python", ".ts": "ts", ".tsx": "ts", ".js": "ts",
	".jsx": "ts", ".ex": "elixir", ".exs": "elixir", ".rs": "rust",
}

// scanEdge is one observed cross-boundary edge with its offender files, as
// judged by the contract: "allowed", "denied", "undeclared", or "baselined".
type scanEdge struct {
	Src, Dst string
	Witness  string
	Files    []string // sorted
	Status   string
}

// importScan collects what `machinery baseline` needs from a G4 pass, so the
// generator and the gate share one discovery implementation and can never
// disagree about what the code contains.
type importScan struct {
	Edges         []scanEdge
	UnmappedFiles []string            // source files outside every boundary (rel)
	OrphanRefs    map[string][]string // module-internal import -> referencing files
	Complete      bool                // the walk and judgment actually ran
}

// CheckImports implements G4-import.
func CheckImports(design, impl string) *Gate {
	return checkImports(design, impl, nil)
}

// checkImports is CheckImports with an optional scan collector; scan may be
// nil (the plain gate) and collecting must never change the gate's findings.
func checkImports(design, impl string, scan *importScan) *Gate {
	g := NewGate("G4-import  code respects the contract")
	g.startOrder()
	if fi, err := os.Stat(impl); err != nil || !fi.IsDir() {
		g.Errs = append(g.Errs, fmt.Sprintf("--impl %s is not a directory", ir.Repr(impl)))
		return g
	}
	cg := NewGate("_")
	c := loadContract(filepath.Join(design, "ARCHITECTURE.md"), cg)
	if c == nil {
		g.Errs = append(g.Errs, cg.Errs...)
		if len(cg.Errs) == 0 {
			g.Errs = append(g.Errs, "no contract to check against")
		}
		return g
	}
	co := c.AsObject()
	var boundaries []*ir.Value
	for _, b := range objSlice(co.Get2("boundaries")) {
		if bo := b.AsObject(); bo != nil && bo.GetString("id") != "" {
			boundaries = append(boundaries, b)
		}
	}
	var externals []*ir.Value
	for _, x := range objSlice(co.Get2("externals")) {
		if xo := x.AsObject(); xo != nil && xo.GetString("id") != "" {
			externals = append(externals, x)
		}
	}
	var ignore []string
	for _, ig := range objSlice(co.Get2("ignore")) {
		ignore = append(ignore, ig.AsString())
	}
	var pkgmap [][2]string
	exposes := map[string][]string{}
	for _, b := range boundaries {
		bo := b.AsObject()
		for _, code := range objSlice(bo.Get2("code")) {
			pkgmap = append(pkgmap, [2]string{code.AsString(), bo.GetString("id")})
		}
		if exp := bo.Get2("exposes"); exp != nil {
			var es []string
			for _, e := range exp.AsArray() {
				es = append(es, e.AsString())
			}
			exposes[bo.GetString("id")] = es
		}
	}
	var extByPrefix, extModules, boundModules [][2]string
	for _, x := range externals {
		xo := x.AsObject()
		for _, p := range objSlice(xo.Get2("imports")) {
			extByPrefix = append(extByPrefix, [2]string{p.AsString(), xo.GetString("id")})
		}
		for _, mp := range objSlice(xo.Get2("modules")) {
			extModules = append(extModules, [2]string{mp.AsString(), xo.GetString("id")})
		}
	}
	for _, b := range boundaries {
		bo := b.AsObject()
		for _, mp := range objSlice(bo.Get2("modules")) {
			boundModules = append(boundModules, [2]string{mp.AsString(), bo.GetString("id")})
		}
	}
	rules := co.GetObject("dependency_rules")
	if rules == nil {
		rules = ir.NewObject()
	}
	allow := contractEdges(rules, "allow", nil)
	deny := contractEdges(rules, "deny", nil)
	baselineRules := contractEdges(rules, "baseline", nil)
	// a wildcard baseline rule would amnesty the whole edge space, so it
	// never matches here; G2 owns the hard ERROR on the rule itself (GATE-7),
	// and the edges it would have covered stay undeclared/denied below
	baselineRules = dropWildcardEdges(baselineRules)
	ratchet, ratchetErr := LoadRatchet(design)
	if ratchetErr != nil {
		g.Errs = append(g.Errs, ratchetErr.Error())
	}
	if ratchet != nil && ratchet.Date != "" {
		g.Notes = append(g.Notes, ratchetAgeNote(ratchet.Date, time.Now()))
	}

	matchRule := func(edges [][2]string, src, dst string) bool {
		for _, e := range edges {
			ok1, _ := filepath.Match(e[0], src)
			ok2, _ := filepath.Match(e[1], dst)
			if ok1 && ok2 {
				return true
			}
		}
		return false
	}

	goModule := goModuleName(impl)

	internalTarget := func(ref string) (string, string) {
		if goModule != "" && (ref == goModule || strings.HasPrefix(ref, goModule+"/")) {
			rel := strings.TrimLeft(ref[len(goModule):], "/")
			return boundaryOf(rel, pkgmap), rel
		}
		for _, bm := range boundModules {
			if ref == bm[0] || strings.HasPrefix(ref, bm[0]+".") {
				return bm[1], ref
			}
		}
		if b := boundaryOf(ref, pkgmap); b != "" {
			return b, ref
		}
		for _, ext := range []string{"", ".py", ".ts", ".tsx", ".js", ".rs"} {
			if b := boundaryOf(ref+ext, pkgmap); b != "" {
				return b, ref + ext
			}
		}
		return "", ""
	}

	externalTarget := func(ref string) string {
		for _, ep := range extByPrefix {
			prefix := strings.TrimSuffix(ep[0], "/")
			if ref == ep[0] || strings.HasPrefix(ref, prefix+"/") {
				return ep[1]
			}
		}
		for _, em := range extModules {
			if ref == em[0] || strings.HasPrefix(ref, em[0]+".") {
				return em[1]
			}
		}
		return ""
	}

	// Each distinct cross-boundary edge is judged once, but every witness file
	// is counted so a violation's error names the real amount of work.
	type edgeRec struct {
		witness string
		files   map[string]bool
	}
	edgeHits := map[[2]string]*edgeRec{}
	var edgeOrder [][2]string
	files, walkErr := walkSourceFiles(impl)
	if walkErr != nil {
		g.Errs = append(g.Errs, "walking "+impl+": "+walkErr.Error())
	}
	sort.Strings(files)

	for _, path := range files {
		rel, _ := filepath.Rel(impl, path)
		ignored := false
		for _, ig := range ignore {
			if matchGlob(rel, ig) {
				ignored = true
				break
			}
		}
		if ignored {
			g.Count("files ignored by contract")
			continue
		}
		if isTestFile(rel) {
			g.Count("test files skipped")
			continue
		}
		lang := langExts[filepath.Ext(path)]
		srcB := boundaryOf(rel, pkgmap)
		text := readFileOrErr(path, g)
		if lang == "rust" {
			// judge only the production portion: imports living inside a
			// #[cfg(test)] module are test wiring (Gt's corpus), and a file
			// carrying such a module is NOT thereby a test file (NG-1)
			text, _ = rustSplitTests(text)
		}
		if srcB == "" && lang == "elixir" {
			for _, mod := range exDefmoduleRe.FindAllStringSubmatch(text, -1) {
				for _, bm := range boundModules {
					if mod[1] == bm[0] || strings.HasPrefix(mod[1], bm[0]+".") {
						srcB = bm[1]
						break
					}
				}
			}
		}
		if srcB == "" {
			g.Errs = append(g.Errs, "source file "+rel+" maps to no contract boundary; add it to a boundary's code globs or to the contract ignore list")
			if scan != nil {
				scan.UnmappedFiles = append(scan.UnmappedFiles, rel)
			}
			continue
		}
		g.Count(lang + " files checked")

		var refs []string
		switch lang {
		case "go":
			refs = goImports(text)
		case "python":
			refs = pyImports(text, rel)
		case "ts":
			refs = tsImports(text, rel)
		case "elixir":
			refs = exModules(text)
		case "rust":
			refs = rustImports(text)
		}
		for _, ref := range refs {
			dstB, norm := internalTarget(ref)
			if dstB == "" {
				dstB = externalTarget(ref)
				norm = ref
				if dstB == "" {
					if goModule != "" && strings.HasPrefix(ref, goModule+"/") {
						g.Errs = append(g.Errs, rel+": imports "+ref+", which maps to no contract boundary (code outside the contract)")
						if scan != nil {
							if scan.OrphanRefs == nil {
								scan.OrphanRefs = map[string][]string{}
							}
							scan.OrphanRefs[ref] = append(scan.OrphanRefs[ref], rel)
						}
					}
					continue
				}
			}
			g.Count("imports resolved")
			if dstB == srcB {
				continue
			}
			exp := exposes[dstB]
			if exp != nil && norm != "" {
				exposedDirs := map[string]bool{}
				for _, e := range exp {
					if !strings.Contains(e, "*") {
						exposedDirs[filepath.Dir(e)] = true
					}
				}
				ok := exposedDirs[norm]
				if !ok {
					for _, e := range exp {
						for _, cand := range []string{norm, norm + ".py", norm + ".ts", norm + ".js", norm + ".rs"} {
							if m, _ := filepath.Match(e, cand); m {
								ok = true
								break
							}
						}
						if ok {
							break
						}
					}
				}
				if !ok {
					g.Errs = append(g.Errs, rel+": imports "+ref+", which is not in the exposes list of "+dstB)
				}
			}
			edge := [2]string{srcB, dstB}
			if rec, hit := edgeHits[edge]; hit {
				rec.files[rel] = true
				continue
			}
			edgeHits[edge] = &edgeRec{witness: rel, files: map[string]bool{rel: true}}
			edgeOrder = append(edgeOrder, edge)
		}
	}
	missingRatchetReported := false
	for _, edge := range edgeOrder {
		srcB, dstB := edge[0], edge[1]
		rec := edgeHits[edge]
		seen := "seen in " + rec.witness
		if extra := len(rec.files) - 1; extra == 1 {
			seen += " and 1 more file"
		} else if extra > 1 {
			seen += fmt.Sprintf(" and %d more files", extra)
		}
		denied := matchRule(deny, srcB, dstB)
		allowed := matchRule(allow, srcB, dstB)
		baselined := matchRule(baselineRules, srcB, dstB)
		status := ""
		switch {
		case baselined && allowed:
			// G2 reports the allow+baseline contradiction; judge as allowed
			// here so the finding is not duplicated per edge
			status = "allowed"
			g.Count("edges verified")
		case baselined:
			// baseline tolerates the edge (even a denied one: intent stays
			// written as deny while the debt is being burned down) but the
			// ratchet holds it to its snapshot: no new offender files
			status = "baselined"
			g.Count("baselined edges")
			key := srcB + " -> " + dstB
			if ratchet == nil {
				if !missingRatchetReported {
					g.Errs = append(g.Errs, "contract has baseline: rules but design has no "+RatchetFile+"; run 'machinery baseline <design> --impl <dir>' to record the snapshot")
					missingRatchetReported = true
				}
			} else if snap, ok := ratchet.Edges[key]; !ok {
				g.Errs = append(g.Errs, "baselined edge "+key+" has no "+RatchetFile+" entry ("+seen+"); rerun 'machinery baseline' to record it")
			} else {
				snapSet := setOf(snap)
				var grew []string
				for f := range rec.files {
					if !snapSet[f] {
						grew = append(grew, f)
					}
				}
				sort.Strings(grew)
				if len(grew) > 0 {
					show := strings.Join(grew, ", ")
					if len(grew) > 3 {
						show = strings.Join(grew[:3], ", ") + fmt.Sprintf(" and %d more", len(grew)-3)
					}
					g.Errs = append(g.Errs, fmt.Sprintf("ratchet: baselined edge %s grew by %d new offender file(s) (%s); fix the new code, or rerun 'machinery baseline' only if the growth is a deliberate decision", key, len(grew), show))
				} else {
					shrunk := 0
					for _, f := range snap {
						if !rec.files[f] {
							shrunk++
						}
					}
					if shrunk > 0 {
						g.Notes = append(g.Notes, fmt.Sprintf("ratchet can tighten: %s dropped %d offender file(s); rerun 'machinery baseline'", key, shrunk))
					}
					g.Count("ratcheted edges")
				}
			}
		case denied && !allowed:
			status = "denied"
			g.Errs = append(g.Errs, srcB+" -> "+dstB+" is denied by the contract ("+seen+"); either the code violates the boundary or the contract needs an explicit allow")
		case !allowed && !denied:
			status = "undeclared"
			g.Errs = append(g.Errs, "undeclared cross-boundary edge "+srcB+" -> "+dstB+" ("+seen+"); add an explicit allow or deny to the contract")
		default:
			status = "allowed"
			g.Count("edges verified")
		}
		if scan != nil {
			var files []string
			for f := range rec.files {
				files = append(files, f)
			}
			sort.Strings(files)
			scan.Edges = append(scan.Edges, scanEdge{Src: srcB, Dst: dstB, Witness: rec.witness, Files: files, Status: status})
		}
	}
	if ratchet != nil {
		observed := map[string]bool{}
		for _, e := range edgeOrder {
			observed[e[0]+" -> "+e[1]] = true
		}
		var stale []string
		for k := range ratchet.Edges {
			if !observed[k] {
				stale = append(stale, k)
			}
		}
		sort.Strings(stale)
		for _, k := range stale {
			g.Notes = append(g.Notes, "ratchet edge "+k+" is no longer observed; rerun 'machinery baseline' to retire it")
		}
	}
	if scan != nil {
		scan.Complete = true
	}

	anyChecked := false
	for k, v := range g.Counts {
		if strings.HasSuffix(k, "files checked") && v > 0 {
			anyChecked = true
			break
		}
	}
	if !anyChecked {
		g.Errs = append(g.Errs, "no source files under "+impl+" mapped to any contract boundary; the gate checked nothing")
	}
	g.RequireNonzero("imports resolved", "no imports were resolved against the contract")
	return g
}

// dropWildcardEdges removes rules with a wildcard on either side; used only
// for baseline: rules, which are an enumerated-edges ratchet.
func dropWildcardEdges(edges [][2]string) [][2]string {
	var out [][2]string
	for _, e := range edges {
		if strings.Contains(e[0], "*") || strings.Contains(e[1], "*") {
			continue
		}
		out = append(out, e)
	}
	return out
}

// ratchetAgeNote makes tolerated debt visible: the snapshot date and its age
// in days (GATE-8). Non-blocking by design: the ratchet has no expiry, but a
// silent date hid year-old amnesties.
func ratchetAgeNote(date string, now time.Time) string {
	for _, layout := range []string{"2006-01-02", "2006-01"} {
		if t, err := time.Parse(layout, date); err == nil {
			days := int(now.Sub(t).Hours() / 24)
			if days < 0 {
				days = 0
			}
			return fmt.Sprintf("ratchet snapshot %s, %d day(s) old", date, days)
		}
	}
	return fmt.Sprintf("ratchet snapshot dated %s (not a YYYY-MM or YYYY-MM-DD date)", ir.Repr(date))
}
