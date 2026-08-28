// Gd-idcite: design-side stable-id citations (S6 of the dogfood systemic
// findings, with the N3 suffix convention). Gt holds oracle ids against the
// implementation suite; nothing held the ids cited in BUILD shards, matrices,
// and DoD lines against the committed oracles, and every review round found
// dangling or wrong citations that only ad-hoc conductor sweeps caught. This
// gate makes a dangling citation a deterministic ERROR: every token in a
// hand-written design file that looks like a stable id under a KNOWN tag must
// resolve to a committed oracle row.
package gates

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// stableIDToken matches a stable-id citation in prose: a known-looking tag,
// a hyphen, exactly six hex chars, then optionally the falsifying-clause
// suffix convention: one lowercase letter, or a letter range `a..c`. The
// surrounding groups keep the match whole-token (backtick-tolerant).
var stableIDToken = regexp.MustCompile(`(^|[^A-Za-z0-9_-])([A-Z][A-Z0-9]*)-([0-9a-f]{6})([a-z](?:\.\.[a-z])?)?($|[^A-Za-z0-9_])`)

// positionalIDToken matches a positional oracle row citation (T-TAG-NN).
// Rows renumber when transitions are inserted, so citing one in hand-written
// prose is fragile; the stable id is the durable name.
var positionalIDToken = regexp.MustCompile(`(^|[^A-Za-z0-9_-])T-([A-Z][A-Z0-9]*)-(\d+)($|[^A-Za-z0-9_-])`)

// idciteSkips reports whether a design-relative path is outside the
// hand-written surface this gate audits: generated artifacts (which cite
// nothing by hand) and the removed-ids allowance file itself.
func idciteSkips(rel string) bool {
	rel = filepath.ToSlash(rel)
	base := filepath.Base(rel)
	switch {
	case strings.HasSuffix(base, ".oracle.md"):
		return true
	case strings.HasPrefix(rel, "formal/") && (strings.HasSuffix(base, ".tla") || strings.HasSuffix(base, ".cfg") || strings.HasSuffix(base, ".als")):
		return true
	case strings.HasPrefix(rel, "packs/") || strings.HasPrefix(rel, "pack/"):
		return true
	case base == "ratchet.json" || base == "removed-ids.txt":
		return true
	}
	return false
}

func idciteScannable(base string) bool {
	switch strings.ToLower(filepath.Ext(base)) {
	case ".md", ".json", ".yaml", ".yml", ".dsl", ".txt":
		return true
	}
	return false
}

// loadCommittedIDs walks the design for every *.oracle.md (machine oracles
// under machines/, alloy oracles under formal/) and returns the committed
// stable-id set plus the tag vocabulary. The tag set is the false-positive
// wall: an id-shaped token under a tag no committed oracle mints is NOT a
// citation and is never flagged.
func loadCommittedIDs(design string) (ids map[string]bool, tags map[string]bool) {
	ids = map[string]bool{}
	tags = map[string]bool{}
	_ = filepath.Walk(design, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || !strings.HasSuffix(fi.Name(), ".oracle.md") {
			return nil
		}
		body, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		for _, m := range stableIDToken.FindAllStringSubmatch(string(body), -1) {
			if m[4] != "" {
				continue // a suffixed form in an oracle would be noise, not a minted id
			}
			ids[m[2]+"-"+m[3]] = true
			tags[m[2]] = true
		}
		return nil
	})
	return ids, tags
}

// loadRemovedIDs reads the dated removed-ids allowance: design/removed-ids.txt,
// one id per line, '#' comments and blank lines ignored. An id listed there
// may be cited historically without erroring; the gate counts the uses.
func loadRemovedIDs(design string) map[string]bool {
	out := map[string]bool{}
	body, err := os.ReadFile(filepath.Join(design, "removed-ids.txt"))
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out[line] = true
	}
	return out
}

// CheckIDCitations implements Gd-idcite.
func CheckIDCitations(design string) *Gate {
	g := NewGate("Gd-idcite  design-side stable-id citations")
	committed, tags := loadCommittedIDs(design)
	if len(committed) == 0 {
		g.Warns = append(g.Warns, "no committed oracle rows found; nothing to resolve citations against")
		return g
	}
	removed := loadRemovedIDs(design)
	filesScanned := 0
	walkErr := filepath.Walk(design, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(design, path)
		if rerr != nil {
			rel = path
		}
		if idciteSkips(rel) || !idciteScannable(fi.Name()) {
			return nil
		}
		// DECISIONS.md and STATE.md are LEDGERS: their entries cite ids as
		// they stood at writing time, and a superseded id there is history,
		// not drift. Judging them would grow the removed-ids allowance
		// without bound; their citations are counted, never resolved.
		ledger := rel == "DECISIONS.md" || rel == "STATE.md"
		body, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		filesScanned++
		for lineNo, line := range strings.Split(string(body), "\n") {
			for _, m := range stableIDToken.FindAllStringSubmatch(line, -1) {
				tag, hexPart, suffix := m[2], m[3], m[4]
				if !tags[tag] {
					continue // unknown tag: id-shaped noise, not a citation
				}
				base := tag + "-" + hexPart
				if ledger {
					g.Count("ledger citations (historical, unjudged)")
					continue
				}
				switch {
				case suffix != "":
					// N3, normative: a suffixed id is a falsifying-clause test
					// derived from the base. A resolving base makes it neither
					// dangling nor a citation of the base; a dangling base is
					// an error on the suffixed form.
					g.Count("suffixed-form citations")
					if !committed[base] && !removed[base] {
						g.Errs = append(g.Errs, rel+":"+strconv.Itoa(lineNo+1)+": "+base+suffix+" cites a falsifying-clause derivative of "+base+", which no committed oracle mints")
					}
				case committed[base]:
					g.Count("citations resolved")
				case removed[base]:
					g.Count("allowed removed-id citations")
				default:
					g.Errs = append(g.Errs, rel+":"+strconv.Itoa(lineNo+1)+": "+base+" resolves to no committed oracle row; regenerate the oracle, fix the citation, or record the id in removed-ids.txt with its removal date")
				}
			}
			for _, m := range positionalIDToken.FindAllStringSubmatch(line, -1) {
				if !tags[m[2]] || ledger {
					continue
				}
				g.Warns = append(g.Warns, rel+":"+strconv.Itoa(lineNo+1)+": T-"+m[2]+"-"+m[3]+" is a positional row citation; rows renumber, so cite the stable id instead")
			}
		}
		return nil
	})
	if walkErr != nil {
		g.Errs = append(g.Errs, walkErr.Error())
	}
	g.Count("files scanned", filesScanned)
	g.Count("committed oracle ids", len(committed))
	// S4: count claims beside tables, checked against the table itself.
	checkProseCounts(g, design)
	// S17: clause-declared guards hold every line naming them to their
	// clause vocabulary.
	checkClauseDrift(g, design)
	return g
}
