// Gv-attest: the attestation-evidence gate. Every machinery gate splits its
// domain into a deterministic half the tool checks and an attested half the
// LLM (or the human reviewer) judges. Ga-accept already gave ONE attested
// half a committed record; every other one lived in conversation, so the
// standing answer to "judged by whom, and is the judgment still current?" was
// somebody's memory of a summary that scrolled away.
//
// This gate generalizes Ga's pattern to all of them. One committed file,
// design/attestations.yaml, carries one row per attested claim: which claim
// (from a closed vocabulary this file owns), who attested it, on what date,
// over which artifacts, and the content hash of each of those artifacts at
// attestation time. The gate proves the record exists, names a real claim,
// names an attestor, points at artifacts that exist, and is still CURRENT:
// an artifact whose bytes moved since the attestation invalidates it, exactly
// as Gk's input_hash invalidates a checker verdict computed over a different
// design and Ga's commit binding invalidates a review run on a different tree.
//
// What the gate never checks is whether a judgment is TRUE. That is the whole
// point of the split: the content stays attested, and the bookkeeping around
// it becomes deterministic.

package gates

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/pack"
)

// AttestationsFileName is the committed attestation-evidence file under the
// design. Gv auto-activates from either this file or phase artifacts that make
// a vocabulary claim owed, so deleting the evidence cannot delete the gate.
//
// One file per DESIGN, not one per gate. The attested halves are halves of
// the design's own gates, keyed by nothing else (unlike Ga's milestone number
// or Gj's machine name, which each key a natural per-file partition), so a
// per-gate split would buy nothing but five files to keep in sync and five
// activation checks instead of one stat. It follows the single-file taste of
// migration.yaml, surfaces.yaml, and decomposition.yaml.
const AttestationsFileName = "attestations.yaml"

// attestHashPrefix is the only digest the schema accepts. Pinning one
// algorithm keeps the record comparable across a repository's history; a
// field that accepts several is a field where two rows can disagree about
// what "the hash" means.
const attestHashPrefix = "sha256:"

var (
	attestRootKeys  = stringSet("attestation_version", "attestations", "_comment")
	attestRowKeys   = stringSet("claim", "attestor", "date", "covers", "note", "_comment")
	attestCoverKeys = stringSet("path", "hash", "_comment")
	// attestHashRe pins the digest shape before the byte comparison, so a
	// truncated or upper-cased hash reads as a malformed record rather than
	// as a mismatch.
	attestHashRe = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

// attestClaim is one member of the closed attested-claim vocabulary: the
// claim id, the judgment it stands for, and the artifact whose presence makes
// the claim OWED. The vocabulary is closed on purpose. An open one would let
// a design invent a claim id, attest it, and pass a gate that never asked for
// it, which records diligence instead of holding it.
type attestClaim struct {
	id   string
	what string
	// owed reports whether this design has reached the phase that owes the
	// claim; "" from owedBy means the claim is never owed on its own.
	owed func(design string) bool
	// owedBy names the artifact in the coverage finding.
	owedBy string
}

// attestVocabulary is the closed set, in canonical gate order. It is
// enumerated from the four LLM-attested blocks in skills/machinery/SKILL.md
// (Gate 2, Gate 3, Gate 4 including the isolated-child list, and milestone
// acceptance) plus the attestation list in agents/machinery-fsm-author.md.
// Adding an attested half to SKILL.md means adding its id here; that coupling
// is the point.
var attestVocabulary = []attestClaim{
	{
		id:     "g2.action-ownership",
		what:   "every Modelith action maps to an owning component (checked instead when the design authors the action-ownership table)",
		owed:   hasArchitectureDoc,
		owedBy: "ARCHITECTURE.md",
	},
	{
		id:     "g2.interface-contract-rightness",
		what:   "each interface contract is the RIGHT one: the shape matches what the code will exchange, the error list is exhaustive, the idempotency claim survives a retry",
		owed:   hasArchitectureDoc,
		owedBy: "ARCHITECTURE.md",
	},
	{
		id:     "g2.placement-rightness",
		what:   "each persistence-and-placement decision is the RIGHT one",
		owed:   hasArchitectureDoc,
		owedBy: "ARCHITECTURE.md",
	},
	{
		id:     "g2.adoption-closure-discovery",
		what:   "the adoption closure is fully DISCOVERED: a member nobody declared is invisible to the gate",
		owed:   hasArchitectureDoc,
		owedBy: "ARCHITECTURE.md",
	},
	{
		id:     "g2.event-contract-completeness",
		what:   "the event-contract table covers every cross-component event and the dependency declaration itself is complete",
		owed:   hasArchitectureDoc,
		owedBy: "ARCHITECTURE.md",
	},
	{
		id:     "g2.nfr-content",
		what:   "the NFR record's CONTENT is true (presence and topic coverage are checked; the posture is judgment)",
		owed:   hasArchitectureDoc,
		owedBy: "ARCHITECTURE.md",
	},
	{
		id:     "g3.guard-semantics",
		what:   "each guard's semantics actually enforce the invariant it names",
		owed:   HasMachines,
		owedBy: "machines/*.machine.json",
	},
	{
		id:     "g3.invariant-enforcement",
		what:   "every Modelith invariant is guarded or structurally impossible; any that is neither is listed",
		owed:   HasMachines,
		owedBy: "machines/*.machine.json",
	},
	{
		id:     "g3.residual-transitions",
		what:   "every C4 dependency failure has its residual transition, reclassified by its mitigation rather than deleted",
		owed:   HasMachines,
		owedBy: "machines/*.machine.json",
	},
	{
		id:     "g3.event-redelivery",
		what:   "every consumed external event has its event-contract row and a redelivery story (deterministic slices exist: `_external_events` arms the row-existence sweep per declared event, and G2 refuses a bare dedupe cell under at-least-once; what stays judged is the story's ADEQUACY and the completeness of the declarations)",
		owed:   HasMachines,
		owedBy: "machines/*.machine.json",
	},
	{
		id:     "gt.conformance-test-shape",
		what:   "a wholesale-conformance test parses the committed oracle table and asserts, per row, the next state AND the expected actions (Gt verifies the citation and the ids, never the assertions)",
		owed:   HasBuildDoc,
		owedBy: "BUILD.md",
	},
	{
		id:     "g4.zero-context",
		what:   "a coding agent with no prior context could build the system from BUILD.md alone (per shard, when sharded)",
		owed:   HasBuildDoc,
		owedBy: "BUILD.md",
	},
	{
		id:     "g4.standin-coverage",
		what:   "isolated child only: the neighbor stand-in section exists, every neighboring boundary has a stand-in held to its oracle, and the environment recipe is self-contained",
		owed:   declaresNeighborStandIns,
		owedBy: "the BUILD.md 'Neighbor stand-ins' section",
	},
	{
		id:     "g4.pack-event-discipline",
		what:   "pack child only: the implementation carries no emitter or handler for an event absent from its pack",
		owed:   pack.HasPack,
		owedBy: "pack/",
	},
	{
		id:     "ga.review-quality",
		what:   "the milestone reviewer judged WELL: the DoD was really met, the acceptance file's attestations are true, and its findings list is complete",
		owed:   HasAcceptanceDir,
		owedBy: AcceptanceDirName + "/",
	},
}

// attestClaimByID indexes the vocabulary for the resolution check.
var attestClaimByID = func() map[string]attestClaim {
	m := make(map[string]attestClaim, len(attestVocabulary))
	for _, c := range attestVocabulary {
		m[c.id] = c
	}
	return m
}()

// AttestationClaimIDs returns the closed claim vocabulary in canonical order.
// The CLI and the docs read it from here rather than repeating it, so the set
// cannot drift between the gate and what the gate tells people to write.
func AttestationClaimIDs() []string {
	out := make([]string, 0, len(attestVocabulary))
	for _, c := range attestVocabulary {
		out = append(out, c.id)
	}
	return out
}

// AttestationPath is the evidence file's path within a design.
func AttestationPath(design string) string {
	return filepath.Join(design, AttestationsFileName)
}

// AttestationActive reports whether the design committed attestation
// evidence. Suite activation additionally uses AttestationOwed.
func AttestationActive(design string) bool {
	has, err := probeRegularFile(design, AttestationsFileName)
	return has || err != nil
}

// AttestationOwed reports whether phase artifacts have made at least one
// closed-vocabulary judgment due. Gv activates from the obligation, not only
// from the evidence file: otherwise omitting attestations.yaml omits the gate.
func AttestationOwed(design string) bool {
	for _, claim := range attestVocabulary {
		if claim.owed != nil && claim.owed(design) {
			return true
		}
	}
	return false
}

// ContentHash returns the digest the attestation schema records for path, in
// the schema's own "sha256:<hex>" spelling. It is the one place the digest is
// computed: the gate compares against it and `machinery attest` prints it, so
// an attestor never hand-rolls the value the gate will demand.
func ContentHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return attestHashPrefix + hex.EncodeToString(sum[:]), nil
}

// hasArchitectureDoc reports whether Phase 2 produced an architecture
// document (the artifact the Gate 2 attested claims cover).
func hasArchitectureDoc(design string) bool {
	fi, err := os.Stat(filepath.Join(design, "ARCHITECTURE.md"))
	return err == nil && !fi.IsDir()
}

// declaresNeighborStandIns reports whether the build document declares the
// isolated delivery posture, which is what makes the stand-in claim owed. A
// full-environment child owes nothing here, so keying the obligation on the
// section is keying it on the posture itself.
func declaresNeighborStandIns(design string) bool {
	paths := []string{filepath.Join(design, "BUILD.md")}
	paths = append(paths, sortedGlobExt(filepath.Join(design, "BUILD"), ".md")...)
	for _, p := range paths {
		if strings.Contains(strings.ToLower(readDesignOrEmpty(design, p)), "neighbor stand-ins") {
			return true
		}
	}
	return false
}

// attestRow is one parsed attestation.
type attestRow struct {
	claim    string
	attestor string
	date     string
	covers   []attestCover
}

// attestCover is one covered artifact plus the hash it carried when the claim
// was attested.
type attestCover struct {
	path string
	hash string
}

// CheckAttestations implements Gv-attest.
func CheckAttestations(design string) *Gate {
	g := NewGate("Gv-attest  attestation evidence")
	g.startOrder()
	path := AttestationPath(design)
	has, probeErr := probeRegularFile(design, AttestationsFileName)
	if probeErr != nil {
		g.Errs = append(g.Errs, probeErr.Error())
		return g
	}
	if !has {
		g.Errs = append(g.Errs, "no "+AttestationsFileName+" in the design; the attestation gate was requested but no attested judgment was committed (write "+AttestationsFileName+", or drop gv from the gate list)")
		return g
	}
	rows := parseAttestations(g, design, path)
	if rows == nil {
		return g
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if seen[row.claim] {
			g.Errs = append(g.Errs, fmt.Sprintf("%s attests %s twice; one row per claim id (git history is the record of prior attestations)", AttestationsFileName, row.claim))
			continue
		}
		seen[row.claim] = true
		checkAttestationFreshness(g, design, row)
		checkAttestationSubjects(g, design, row)
		g.Count("attested claims")
	}
	checkAttestationCoverage(g, design, seen)
	g.RequireNonzero("attested claims", "no attestation row was checked")
	return g
}

func buildArtifactPaths(g *Gate, design string) []string {
	var out []string
	if ok, err := probeRegularFile(design, "BUILD.md"); err != nil {
		g.Errs = append(g.Errs, "BUILD.md inventory failed: "+err.Error())
	} else if ok {
		out = append(out, "BUILD.md")
	}
	paths, _ := strictSortedGlob(g, filepath.Join(design, "BUILD"), "*.md", "build shard")
	for _, path := range paths {
		rel, _ := filepath.Rel(design, path)
		out = append(out, filepath.ToSlash(rel))
	}
	return out
}

func attestationRequiredPaths(g *Gate, design, claim string) []string {
	var out []string
	switch {
	case strings.HasPrefix(claim, "g2."):
		out = append(out, "ARCHITECTURE.md")
	case strings.HasPrefix(claim, "g3."):
		for _, ext := range []string{".machine.json", ".matrix.md"} {
			paths, _ := strictSortedGlob(g, filepath.Join(design, "machines"), "*"+ext, "attestation subject")
			for _, path := range paths {
				rel, _ := filepath.Rel(design, path)
				out = append(out, filepath.ToSlash(rel))
			}
		}
	case claim == "gt.conformance-test-shape" || claim == "g4.zero-context" || claim == "g4.standin-coverage":
		out = append(out, buildArtifactPaths(g, design)...)
	case claim == "ga.review-quality":
		paths, _ := strictSortedGlob(g, filepath.Join(design, AcceptanceDirName), "*.yaml", "acceptance evidence")
		for _, path := range paths {
			rel, _ := filepath.Rel(design, path)
			out = append(out, filepath.ToSlash(rel))
		}
	case claim == "g4.pack-event-discipline":
		err := filepath.Walk(filepath.Join(design, "pack"), func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi == nil || fi.IsDir() {
				return nil
			}
			if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
				return fmt.Errorf("%s must be a regular attestation subject", path)
			}
			rel, _ := filepath.Rel(design, path)
			out = append(out, filepath.ToSlash(rel))
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			g.Errs = append(g.Errs, "pack attestation subject inventory failed: "+err.Error())
		}
	}
	sort.Strings(out)
	return out
}

// checkAttestationSubjects binds each judgment to the artifacts that define
// its subject. Freshness alone only proves that *some* file did not change.
func checkAttestationSubjects(g *Gate, design string, row attestRow) {
	covered := map[string]bool{}
	for _, cover := range row.covers {
		covered[filepath.ToSlash(filepath.Clean(filepath.FromSlash(cover.path)))] = true
	}
	for _, required := range attestationRequiredPaths(g, design, row.claim) {
		if !covered[required] {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: %s does not cover required subject %s; this claim cannot be discharged by an unrelated current file", AttestationsFileName, row.claim, ir.Repr(required)))
		}
	}
}

// parseAttestations reads and validates the evidence file's shape. nil means
// the file could not be trusted; every reason was recorded as an ERROR first.
func parseAttestations(g *Gate, design, path string) []attestRow {
	raw, err := readDesignFile(design, path)
	if err != nil {
		g.Errs = append(g.Errs, AttestationsFileName+" is unreadable: "+err.Error())
		return nil
	}
	value, err := ir.LoadYAML(raw)
	if err != nil {
		g.Errs = append(g.Errs, AttestationsFileName+": invalid YAML: "+err.Error())
		return nil
	}
	root := value.AsObject()
	if root == nil {
		g.Errs = append(g.Errs, AttestationsFileName+" is not a yaml mapping (empty file?)")
		return nil
	}
	for _, key := range root.Keys() {
		if !attestRootKeys[key] {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: unsupported key %s (the fields are attestation_version, attestations)", AttestationsFileName, ir.Repr(key)))
		}
	}
	if ver := root.Get2("attestation_version"); ver == nil || ver.Kind != ir.KindNumber || string(ver.AsNumber()) != "1" {
		g.Errs = append(g.Errs, AttestationsFileName+": attestation_version must be the integer 1")
		return nil
	}
	list := root.Get2("attestations")
	if list == nil || list.Kind != ir.KindArray || len(list.AsArray()) == 0 {
		g.Errs = append(g.Errs, AttestationsFileName+": attestations must be a non-empty list of rows; an empty record is a failure, not a pass")
		return nil
	}
	var out []attestRow
	for i, item := range list.AsArray() {
		if row, ok := parseAttestationRow(g, fmt.Sprintf("%s attestations[%d]", AttestationsFileName, i), item); ok {
			out = append(out, row)
		}
	}
	return out
}

// parseAttestationRow validates one row. The row is dropped (and every reason
// recorded) rather than half-checked: a row missing its claim id cannot be
// held to anything downstream.
func parseAttestationRow(g *Gate, where string, item *ir.Value) (attestRow, bool) {
	obj := item.AsObject()
	if obj == nil {
		g.Errs = append(g.Errs, where+" is not a mapping")
		return attestRow{}, false
	}
	for _, key := range obj.Keys() {
		if !attestRowKeys[key] {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: unsupported key %s (a row is claim, attestor, date, covers, and an optional note)", where, ir.Repr(key)))
		}
	}
	row := attestRow{
		claim:    strings.TrimSpace(obj.GetString("claim")),
		attestor: strings.TrimSpace(obj.GetString("attestor")),
		date:     strings.TrimSpace(obj.GetString("date")),
	}
	ok := true
	if row.claim == "" {
		g.Errs = append(g.Errs, where+".claim is required: the attested claim id this row records")
		ok = false
	} else if _, known := attestClaimByID[row.claim]; !known {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: claim %s is not in the attested-claim vocabulary (the ids are %s)", where, ir.Repr(row.claim), strings.Join(AttestationClaimIDs(), ", ")))
		ok = false
	}
	if row.attestor == "" {
		g.Errs = append(g.Errs, where+".attestor is required: who or what made the judgment; an attestation without an attestor attributes nothing")
		ok = false
	}
	if _, derr := time.Parse("2006-01-02", row.date); derr != nil {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: date %s is not a real YYYY-MM-DD date", where, ir.Repr(row.date)))
		ok = false
	}
	covers := obj.Get2("covers")
	if covers == nil || covers.Kind != ir.KindArray || len(covers.AsArray()) == 0 {
		g.Errs = append(g.Errs, where+".covers must be a non-empty list of {path, hash}; a judgment over nothing cannot go stale, and cannot be reviewed either")
		return attestRow{}, false
	}
	for i, c := range covers.AsArray() {
		cover, cok := parseAttestationCover(g, fmt.Sprintf("%s covers[%d]", where, i), c)
		if !cok {
			ok = false
			continue
		}
		row.covers = append(row.covers, cover)
	}
	if !ok || len(row.covers) == 0 {
		return attestRow{}, false
	}
	return row, true
}

// parseAttestationCover validates one covered-artifact entry.
func parseAttestationCover(g *Gate, where string, item *ir.Value) (attestCover, bool) {
	obj := item.AsObject()
	if obj == nil {
		g.Errs = append(g.Errs, where+" is not a mapping; each covered artifact is {path, hash}")
		return attestCover{}, false
	}
	for _, key := range obj.Keys() {
		if !attestCoverKeys[key] {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: unsupported key %s (a covered artifact is path and hash)", where, ir.Repr(key)))
		}
	}
	cover := attestCover{
		path: strings.TrimSpace(obj.GetString("path")),
		hash: strings.TrimSpace(obj.GetString("hash")),
	}
	ok := true
	switch {
	case cover.path == "":
		g.Errs = append(g.Errs, where+".path is required: the covered artifact, relative to the design directory")
		ok = false
	case filepath.IsAbs(cover.path) || strings.HasPrefix(cover.path, "/"):
		g.Errs = append(g.Errs, fmt.Sprintf("%s: path %s is absolute; covered artifacts are design-relative, so the record travels with the design", where, ir.Repr(cover.path)))
		ok = false
	case escapesDesign(cover.path):
		g.Errs = append(g.Errs, fmt.Sprintf("%s: path %s escapes the design directory; an attestation covers artifacts the design owns", where, ir.Repr(cover.path)))
		ok = false
	}
	if !attestHashRe.MatchString(cover.hash) {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: hash %s is not sha256:<64 lower-case hex> (run 'machinery attest <path>' and paste what it prints)", where, ir.Repr(cover.hash)))
		ok = false
	}
	return cover, ok
}

// escapesDesign reports whether a design-relative path climbs out of the
// design directory.
func escapesDesign(rel string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	return clean == ".." || strings.HasPrefix(clean, "../")
}

// checkAttestationFreshness holds one row to its referents: every covered
// artifact still exists, and its bytes still hash to what the attestor saw.
// A moved artifact makes the row STALE, an ERROR rather than a DRIFT: DRIFT
// means a GENERATED artifact fell behind its source and is fixed by
// regenerating, while a stale attestation can only be fixed by a person
// judging again. Recording that as regenerable would misdescribe the remedy.
func checkAttestationFreshness(g *Gate, design string, row attestRow) {
	for _, cover := range row.covers {
		full := filepath.Join(design, filepath.FromSlash(cover.path))
		data, err := readDesignFile(design, full)
		if err != nil {
			switch {
			case os.IsNotExist(err):
				g.Errs = append(g.Errs, fmt.Sprintf("%s: %s covers %s, which the design does not carry; an attestation over an absent artifact covers nothing", AttestationsFileName, row.claim, ir.Repr(cover.path)))
			case strings.Contains(err.Error(), "is a directory"):
				g.Errs = append(g.Errs, fmt.Sprintf("%s: %s covers %s, which is a directory; a content hash binds to one file's bytes (list the files)", AttestationsFileName, row.claim, ir.Repr(cover.path)))
			case strings.Contains(err.Error(), "symlink"):
				g.Errs = append(g.Errs, fmt.Sprintf("%s: %s is reached through a symlink; attestations bind files physically owned by the design tree", AttestationsFileName, ir.Repr(cover.path)))
			default:
				g.Errs = append(g.Errs, fmt.Sprintf("%s: %s is unreadable: %s", AttestationsFileName, ir.Repr(cover.path), err.Error()))
			}
			continue
		}
		sum := sha256.Sum256(data)
		current := attestHashPrefix + hex.EncodeToString(sum[:])
		if current != cover.hash {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: %s is STALE: %s changed since it was attested (recorded %s, current %s); re-read the artifact, judge it again, and update the row with 'machinery attest %s'",
				AttestationsFileName, row.claim, cover.path, cover.hash, current, filepath.ToSlash(full)))
			continue
		}
		g.Count("covered artifacts current")
	}
}

// checkAttestationCoverage reports every vocabulary claim this design has
// reached the phase for and left unattested.
//
// A warning, not an error, and deliberately so. The evidence file may be
// adopted incrementally; making its first populated commit fail the gate for
// every claim not yet re-judged would make adopting the record more expensive
// than not adopting it, which is the one outcome that guarantees the attested
// halves stay in conversation. What blocks is a record that is WRONG (an
// unknown claim, an unattributed row, a dangling referent, a stale hash): a
// misleading record is worse than an incomplete one. File-level absence still
// blocks once any phase artifact makes a claim owed, and an empty file is an
// ERROR because an empty check is a failure, not a pass. --complete promotes
// the remaining coverage warnings to errors at final handoff.
func checkAttestationCoverage(g *Gate, design string, attested map[string]bool) {
	for _, claim := range attestVocabulary {
		if claim.owed == nil || !claim.owed(design) {
			continue
		}
		g.Count("claims owed")
		if attested[claim.id] {
			continue
		}
		g.Warns = append(g.Warns, fmt.Sprintf("%s has no row for %s, which %s makes owed: %s", AttestationsFileName, claim.id, claim.owedBy, claim.what))
	}
}
