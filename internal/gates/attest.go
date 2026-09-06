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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/designlock"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/pack"
	"github.com/RamXX/machinery/internal/portablepath"
	"gopkg.in/yaml.v3"
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
	owed        func(design string) bool
	owedChecked func(design string) (bool, error)
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
		what:   "a coding agent with no prior context could execute each milestone from its BUILD packet alone (or the single BUILD.md in full mode)",
		owed:   HasBuildDoc,
		owedBy: "BUILD.md",
	},
	{
		id:          "g4.standin-coverage",
		what:        "isolated child only: the neighbor stand-in section exists, every neighboring boundary has a stand-in held to its oracle, and the environment recipe is self-contained",
		owed:        declaresNeighborStandInsConservative,
		owedChecked: declaresNeighborStandIns,
		owedBy:      "the BUILD.md 'Neighbor stand-ins' section",
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
		if claim.owedChecked != nil {
			owed, err := claim.owedChecked(design)
			if err != nil || owed {
				return true
			}
		}
	}
	return false
}

// ContentHash returns the digest the attestation schema records for path, in
// the schema's own "sha256:<hex>" spelling. It is the one place the digest is
// computed: the gate compares against it and `machinery attest` prints it, so
// an attestor never hand-rolls the value the gate will demand.
func ContentHash(path string) (string, error) {
	data, err := readRegularFile(path)
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
func declaresNeighborStandIns(design string) (bool, error) {
	paths := []string{filepath.Join(design, "BUILD.md")}
	packets, err := sortedGlobExt(filepath.Join(design, "BUILD"), ".md")
	if err != nil {
		return false, err
	}
	paths = append(paths, packets...)
	for _, p := range paths {
		if strings.Contains(strings.ToLower(readDesignOrEmpty(design, p)), "neighbor stand-ins") {
			return true, nil
		}
	}
	return false, nil
}

func declaresNeighborStandInsConservative(design string) bool {
	declared, err := declaresNeighborStandIns(design)
	return err != nil || declared
}

// attestRow is one parsed attestation.
type attestRow struct {
	claim          string
	attestor       string
	date           string
	covers         []attestCover
	kind           string
	version        string
	implementation *attestationManifest
}

// attestCover is one covered artifact plus the hash it carried when the claim
// was attested.
type attestCover struct {
	path string
	hash string
}

// CheckAttestations implements Gv-attest.
func CheckAttestations(design string) *Gate {
	return checkAttestationsInSnapshot(design, nil)
}

func checkAttestationsInSnapshot(design string, subject *attestationSubject) *Gate {
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
	pending := &pendingAttestationResult{gate: g}
	for _, row := range rows {
		if seen[row.claim] {
			g.Errs = append(g.Errs, fmt.Sprintf("%s attests %s twice; one row per claim id (git history is the record of prior attestations)", AttestationsFileName, row.claim))
			continue
		}
		seen[row.claim] = true
		before := len(g.Errs)
		checkAttestationFreshness(g, design, row)
		checkAttestationSubjects(g, design, row)
		g.Count("attested claims")
		class := attestationClaimKinds[row.claim]
		if row.version == "1" && class == "current" {
			g.Errs = append(g.Errs, fmt.Sprintf("GV_MISSING_IMPLEMENTATION_SUBJECT: %s has legacy design-only covers; review the complete implementation/test scope and run machinery attest --design <design> --claim %s --kind current --impl <root> --attestor <reviewer> --date YYYY-MM-DD, or explicitly recast as v2 kind=plan", row.claim, row.claim))
			continue
		}
		if row.kind == "current" {
			if subject == nil {
				g.Errs = append(g.Errs, "GV_IMPL_REQUIRED: "+row.claim+" requires --impl <reviewed-root>; the receipt locator grants no read authority")
			} else if subject.pendingResults == nil {
				g.Errs = append(g.Errs, "GV_SCOPE_CUSTODY: current subject has no finalization owner")
			} else if err := compareAttestationManifest(row.implementation, subject); err != nil {
				g.Errs = append(g.Errs, row.claim+": "+err.Error())
			}
			if len(g.Errs) == before {
				pending.current++
				pending.entries = len(subject.manifest.Entries)
				for _, e := range subject.manifest.Entries {
					if e.Type == "file" {
						pending.files++
					}
				}
			}
		} else if len(g.Errs) == before {
			if row.kind == "historical" {
				g.Count("historical review records")
				g.Notes = append(g.Notes, "Historical acceptance-file judgment only; Gv does not execute Ga ancestry validation or establish current implementation approval.")
			} else {
				g.Count("plan judgments")
			}
			if row.version == "1" {
				g.Notes = append(g.Notes, "Legacy v1 "+row.claim+" is an implicit "+row.kind+" judgment; migrate to explicit v2 kind="+row.kind+".")
			}
			if class == "current" {
				g.Warns = append(g.Warns, row.claim+": plan only; current implementation review missing")
			}
		}
	}
	checkAttestationCoverage(g, design, seen)
	g.RequireNonzero("attested claims", "no attestation row was checked")
	if pending.current > 0 {
		g.Notes = append(g.Notes, attestationPendingNote)
		*subject.pendingResults = append(*subject.pendingResults, pending)
	}
	return g
}

func buildArtifactPaths(g *Gate, design string) []string {
	var out []string
	if ok, err := probeRegularFile(design, "BUILD.md"); err != nil {
		g.Errs = append(g.Errs, "BUILD.md inventory failed: "+err.Error())
	} else if ok {
		out = append(out, "BUILD.md")
	}
	paths, _ := strictSortedGlob(g, filepath.Join(design, "BUILD"), "*.md", "build packet")
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
		err := walkTreeBounded(filepath.Join(design, "pack"), func(path string, fi os.FileInfo, err error) error {
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
	start := len(g.Errs)
	defer func() {
		for i := start; i < len(g.Errs); i++ {
			if !strings.Contains(g.Errs[i], "GV_") {
				g.Errs[i] = "GV_SCHEMA: " + g.Errs[i]
			}
		}
	}()
	raw, err := readDesignFile(design, path)
	if err != nil {
		g.Errs = append(g.Errs, AttestationsFileName+" is unreadable: "+err.Error())
		return nil
	}
	value, err := ir.LoadYAML(raw)
	if err != nil {
		g.Errs = append(g.Errs, "GV_SCHEMA: "+AttestationsFileName+": invalid YAML: "+err.Error())
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
	ver := root.Get2("attestation_version")
	if ver == nil || ver.Kind != ir.KindNumber || (string(ver.AsNumber()) != "1" && string(ver.AsNumber()) != "2") {
		g.Errs = append(g.Errs, "GV_SCHEMA: "+AttestationsFileName+": attestation_version must be the integer 1 or 2")
		return nil
	}
	validateAttestationStrings(g, AttestationsFileName, root, "_comment")
	list := root.Get2("attestations")
	if list == nil || list.Kind != ir.KindArray || len(list.AsArray()) == 0 {
		g.Errs = append(g.Errs, AttestationsFileName+": attestations must be a non-empty list of rows; an empty record is a failure, not a pass")
		return nil
	}
	var out []attestRow
	for i, item := range list.AsArray() {
		if row, ok := parseAttestationRow(g, fmt.Sprintf("%s attestations[%d]", AttestationsFileName, i), item, string(ver.AsNumber())); ok {
			out = append(out, row)
		}
	}
	return out
}

// parseAttestationRow validates one row. The row is dropped (and every reason
// recorded) rather than half-checked: a row missing its claim id cannot be
// held to anything downstream.
func parseAttestationRow(g *Gate, where string, item *ir.Value, version string) (attestRow, bool) {
	obj := item.AsObject()
	if obj == nil {
		g.Errs = append(g.Errs, where+" is not a mapping")
		return attestRow{}, false
	}
	for _, key := range obj.Keys() {
		if !attestRowKeys[key] && !(version == "2" && (key == "kind" || key == "implementation")) {
			g.Errs = append(g.Errs, fmt.Sprintf("GV_SCHEMA: %s: unsupported key %s (a row is claim, attestor, date, covers, and an optional note)", where, ir.Repr(key)))
		}
	}
	row := attestRow{
		claim:    strings.TrimSpace(obj.GetString("claim")),
		attestor: strings.TrimSpace(obj.GetString("attestor")),
		date:     strings.TrimSpace(obj.GetString("date")),
		version:  version,
		kind:     obj.GetString("kind"),
	}
	validateAttestationStrings(g, where, obj, "claim", "attestor", "date", "note", "_comment")
	if version == "1" {
		row.kind = attestationClaimKinds[row.claim]
		if row.kind == "current" {
			row.kind = "plan"
		}
	} else {
		validateAttestationStrings(g, where, obj, "kind")
		class := attestationClaimKinds[row.claim]
		if row.kind != "plan" && row.kind != "current" && row.kind != "historical" || class != "" && row.kind != class && !(class == "current" && row.kind == "plan") {
			g.Errs = append(g.Errs, "GV_KIND: "+where+" kind is incompatible with "+row.claim)
		}
		impl := obj.Get2("implementation")
		if row.kind == "current" {
			if impl == nil {
				g.Errs = append(g.Errs, "GV_MISSING_IMPLEMENTATION_SUBJECT: "+row.claim+" requires a complete implementation subject")
			} else {
				row.implementation = parseAttestationManifest(g, where, impl)
			}
		} else if impl != nil {
			g.Errs = append(g.Errs, "GV_SCHEMA: "+where+" implementation is forbidden unless kind=current")
		}
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
	coverPaths := map[string]bool{}
	for _, cover := range row.covers {
		folded := strings.ToLower(cover.path)
		if coverPaths[folded] {
			g.Errs = append(g.Errs, "GV_SCHEMA: duplicate/casefold cover path "+cover.path)
		}
		coverPaths[folded] = true
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
	validateAttestationStrings(g, where, obj, "path", "hash", "_comment")
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
	if err := portablepath.ValidateRelative(obj.GetString("path")); err != nil {
		g.Errs = append(g.Errs, "GV_SCOPE_PATH: "+where+" "+err.Error())
		ok = false
	}
	if cover.path == AttestationsFileName {
		g.Errs = append(g.Errs, "GV_SCHEMA: covers may not self-reference "+AttestationsFileName)
		ok = false
	}
	return cover, ok
}

// AttestationReview is attribution supplied by the caller, not authentication.
type AttestationReview struct{ Claim, Kind, Attestor, Date, Note string }

const attestationScopeLimits = "Hashes bind the observed files and scope; they do not prove tests ran, reviewer identity, or judgment correctness. Top-level Git administration and the exact Machinery attestation record are excluded; applications that use them as runtime inputs are outside this review boundary."
const attestationPendingNote = "current implementation review pending final snapshot release"

// Classification is deliberately explicit; vocabulary prefixes confer no kind.
var attestationClaimKinds = map[string]string{
	"g2.action-ownership": "plan", "g2.interface-contract-rightness": "plan", "g2.placement-rightness": "plan", "g2.adoption-closure-discovery": "plan", "g2.event-contract-completeness": "plan", "g2.nfr-content": "plan",
	"g3.guard-semantics": "plan", "g3.invariant-enforcement": "plan", "g3.residual-transitions": "plan", "g3.event-redelivery": "plan", "g4.zero-context": "plan",
	"gt.conformance-test-shape": "current", "g4.pack-event-discipline": "current", "g4.standin-coverage": "current", "ga.review-quality": "historical",
}

type attestationEntry struct {
	Path string `yaml:"path"`
	Type string `yaml:"type"`
	Mode string `yaml:"mode"`
	Size *int64 `yaml:"size,omitempty"`
	Hash string `yaml:"hash,omitempty"`
}
type attestationManifest struct {
	Root    string             `yaml:"root"`
	Policy  string             `yaml:"policy"`
	Entries []attestationEntry `yaml:"entries"`
	Hash    string             `yaml:"hash"`
}
type attestationSubject struct {
	manifest       *attestationManifest
	exclusion      string
	pendingResults *[]*pendingAttestationResult
}
type pendingAttestationResult struct {
	gate                    *Gate
	current, files, entries int
}

func validateAttestationStrings(g *Gate, where string, obj *ir.Object, keys ...string) {
	for _, key := range keys {
		if value := obj.Get2(key); value != nil && value.Kind != ir.KindString {
			g.Errs = append(g.Errs, "GV_SCHEMA: "+where+"."+key+" must be a string")
		}
	}
}

func attestationClosed(g *Gate, where string, value *ir.Value, keys ...string) *ir.Object {
	obj := value.AsObject()
	if obj == nil {
		g.Errs = append(g.Errs, "GV_SCHEMA: "+where+" must be a mapping")
		return nil
	}
	allowed := stringSet(keys...)
	for _, key := range obj.Keys() {
		if !allowed[key] {
			g.Errs = append(g.Errs, "GV_SCHEMA: "+where+" unsupported key "+key)
		}
	}
	return obj
}

func validAttestationLocator(locator string) bool {
	if locator == "" || filepath.ToSlash(filepath.Clean(locator)) != locator || strings.Contains(locator, "\\") || filepath.IsAbs(locator) {
		return false
	}
	if locator == "." {
		return true
	}
	for strings.HasPrefix(locator, "../") {
		locator = strings.TrimPrefix(locator, "../")
	}
	return locator == ".." || portablepath.ValidateRelative(locator) == nil
}

func parseAttestationManifest(g *Gate, where string, value *ir.Value) *attestationManifest {
	obj := attestationClosed(g, where+" implementation", value, "root", "policy", "entries", "hash")
	if obj == nil {
		return nil
	}
	validateAttestationStrings(g, where, obj, "root", "policy", "hash")
	m := &attestationManifest{Root: obj.GetString("root"), Policy: obj.GetString("policy"), Hash: obj.GetString("hash")}
	if !validAttestationLocator(m.Root) {
		g.Errs = append(g.Errs, "GV_SCOPE_ROOT: invalid implementation root "+m.Root)
	}
	if m.Policy != "full-root-v1" || !attestHashRe.MatchString(m.Hash) {
		g.Errs = append(g.Errs, "GV_SCHEMA: implementation requires policy full-root-v1 and a lowercase sha256 hash")
	}
	list := obj.Get2("entries")
	if list == nil || list.Kind != ir.KindArray || len(list.AsArray()) == 0 || len(list.AsArray()) > 100000 {
		g.Errs = append(g.Errs, "GV_SCHEMA: implementation entries must contain 1..100000 rows")
		return nil
	}
	paths := map[string]string{}
	parents := map[string]bool{}
	previous := ""
	var aggregate int64
	for i, item := range list.AsArray() {
		eo := attestationClosed(g, where+" entry", item, "path", "type", "mode", "size", "hash")
		if eo == nil {
			continue
		}
		validateAttestationStrings(g, where, eo, "path", "type", "mode", "hash")
		e := attestationEntry{Path: eo.GetString("path"), Type: eo.GetString("type"), Mode: eo.GetString("mode"), Hash: eo.GetString("hash")}
		if e.Path != "." && portablepath.ValidateRelative(e.Path) != nil || i == 0 && (e.Path != "." || e.Type != "directory") || i > 0 && e.Path <= previous || paths[strings.ToLower(e.Path)] != "" || e.Path != "." && !parents[filepath.ToSlash(filepath.Dir(e.Path))] {
			g.Errs = append(g.Errs, "GV_SCOPE_PATH: invalid, unsorted, aliased, or parentless entry "+e.Path)
		}
		previous = e.Path
		paths[strings.ToLower(e.Path)] = e.Path
		mode, err := strconv.ParseUint(e.Mode, 8, 32)
		if err != nil || len(e.Mode) != 4 || e.Mode[0] != '0' || mode > 0777 {
			g.Errs = append(g.Errs, "GV_SCHEMA: entry "+e.Path+" mode must be a four-character octal permission string 0000..0777")
		}
		if strings.Count(e.Path, "/") > 64 {
			g.Errs = append(g.Errs, "GV_EVIDENCE_LIMIT: scope path exceeds depth 64: "+e.Path)
		}
		size := eo.Get2("size")
		switch e.Type {
		case "directory":
			parents[e.Path] = true
			if size != nil || eo.Get2("hash") != nil {
				g.Errs = append(g.Errs, "GV_SCHEMA: directory "+e.Path+" forbids size/hash")
			}
		case "file":
			if size == nil || size.Kind != ir.KindNumber {
				g.Errs = append(g.Errs, "GV_SCHEMA: file "+e.Path+" size must be a nonnegative integer")
			} else {
				n, err := strconv.ParseInt(string(size.AsNumber()), 10, 64)
				if err != nil || n < 0 || n > 1<<30 || strconv.FormatInt(n, 10) != string(size.AsNumber()) {
					g.Errs = append(g.Errs, "GV_EVIDENCE_LIMIT: file "+e.Path+" size must be an integer within 1073741824 bytes")
				} else {
					e.Size = &n
					aggregate += n
				}
			}
			if !attestHashRe.MatchString(e.Hash) {
				g.Errs = append(g.Errs, "GV_SCHEMA: file "+e.Path+" requires sha256:<64 lowercase hex>")
			}
		default:
			g.Errs = append(g.Errs, "GV_SCHEMA: entry "+e.Path+" type must be directory or file")
		}
		m.Entries = append(m.Entries, e)
	}
	if aggregate > 8<<30 {
		g.Errs = append(g.Errs, "GV_EVIDENCE_LIMIT: scope exceeds 8589934592-byte aggregate limit")
	}
	return m
}

func attestationDigest(m *attestationManifest, exclusion string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "machinery-attestation-scope-v1\nroot\t%s\npolicy\t%s\nexclude\tvcs-root:.git\nexclude\tevidence:%s\n", m.Root, m.Policy, exclusion)
	for _, e := range m.Entries {
		if e.Type == "directory" {
			fmt.Fprintf(&b, "directory\t%s\t%s\n", e.Path, e.Mode)
		} else {
			size := int64(0)
			if e.Size != nil {
				size = *e.Size
			}
			fmt.Fprintf(&b, "file\t%s\t%s\t%d\t%s\n", e.Path, e.Mode, size, e.Hash)
		}
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(b.String())))
}

func compareAttestationManifest(recorded *attestationManifest, subject *attestationSubject) error {
	if recorded == nil {
		return fmt.Errorf("GV_MISSING_IMPLEMENTATION_SUBJECT: current review requires a complete subject")
	}
	fresh := subject.manifest
	if recorded.Root != fresh.Root {
		return fmt.Errorf("GV_SCOPE_ROOT: recorded root %s differs from supplied root %s", recorded.Root, fresh.Root)
	}
	if attestationDigest(recorded, subject.exclusion) != recorded.Hash {
		return fmt.Errorf("GV_SCOPE_HASH: submitted inventory does not match its recorded hash")
	}
	want := map[string]attestationEntry{}
	got := map[string]attestationEntry{}
	all := map[string]bool{}
	for _, e := range recorded.Entries {
		want[e.Path] = e
		all[e.Path] = true
	}
	for _, e := range fresh.Entries {
		got[e.Path] = e
		all[e.Path] = true
	}
	var names []string
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)
	// Compare both path sets before content so additions cannot hide behind a
	// simultaneous parent-mode change or an earlier altered file.
	var differing []string
	for _, name := range names {
		_, w := want[name]
		_, g := got[name]
		if w != g {
			differing = append(differing, name)
		}
	}
	if len(differing) > 0 {
		shown := differing
		if len(shown) > 8 {
			shown = shown[:8]
		}
		return fmt.Errorf("GV_SCOPE_INVENTORY: scope path %s added or removed (recorded %d entries, observed %d; %d changed paths: %s)", differing[0], len(want), len(got), len(differing), strings.Join(shown, ", "))
	}
	for _, name := range names {
		w, g := want[name], got[name]
		sizeEqual := w.Size == nil && g.Size == nil || w.Size != nil && g.Size != nil && *w.Size == *g.Size
		if w.Type != g.Type || w.Mode != g.Mode || w.Hash != g.Hash || !sizeEqual {
			return fmt.Errorf("GV_STALE_CONTENT: scope path %s changed (recorded %d entries, observed %d); review the changed implementation/test subject again", name, len(want), len(got))
		}
	}
	if recorded.Hash != fresh.Hash {
		return fmt.Errorf("GV_SCOPE_HASH: scope digest differs")
	}
	return nil
}

func (s *Snapshot) captureAttestationSubject(impl string) (*attestationSubject, *designlock.AttestationTreeSnapshot, error) {
	if s.attestationFinalized {
		return nil, nil, fmt.Errorf("GV_SCOPE_CUSTODY: snapshot released")
	}
	stable, err := s.lock.MaterializeAttestationTree(impl)
	if err != nil {
		s.attestationCustodyErr = errors.Join(s.attestationCustodyErr, err)
		return nil, nil, err
	}
	s.attestationCaptures = append(s.attestationCaptures, stable)
	design, err := filepath.Abs(s.logicalDesign)
	if err != nil {
		return nil, stable, err
	}
	locator, err := filepath.Rel(design, stable.Logical())
	if err != nil {
		return nil, stable, err
	}
	subject := &attestationSubject{manifest: &attestationManifest{Root: filepath.ToSlash(locator), Policy: "full-root-v1"}, exclusion: "none", pendingResults: &s.attestationPending}
	if !validAttestationLocator(subject.manifest.Root) {
		return nil, stable, fmt.Errorf("GV_SCOPE_ROOT: nonportable implementation locator %s", subject.manifest.Root)
	}
	excluded, err := filepath.Rel(stable.Logical(), filepath.Join(design, AttestationsFileName))
	if err == nil && !escapesDesign(excluded) {
		subject.exclusion = filepath.ToSlash(excluded)
	}
	for _, e := range stable.Entries() {
		if e.Path == subject.exclusion {
			continue
		}
		entry := attestationEntry{Path: e.Path, Type: "directory", Mode: fmt.Sprintf("%04o", e.Mode)}
		if !e.Directory {
			entry.Type = "file"
			size := e.Size
			entry.Size = &size
			entry.Hash = fmt.Sprintf("sha256:%x", e.SHA256)
		}
		subject.manifest.Entries = append(subject.manifest.Entries, entry)
	}
	subject.manifest.Hash = attestationDigest(subject.manifest, subject.exclusion)
	return subject, stable, nil
}

func finalizeAttestationResults(pending []*pendingAttestationResult, err error) {
	for _, p := range pending {
		g := p.gate
		notes := g.Notes[:0]
		for _, note := range g.Notes {
			if note != attestationPendingNote {
				notes = append(notes, note)
			}
		}
		g.Notes = notes
		if err != nil {
			g.Errs = append(g.Errs, "GV_SCOPE_CUSTODY: current review not established; final snapshot validation or release failed: "+err.Error())
			continue
		}
		if len(g.Errs) > 0 {
			continue
		}
		g.Count("current implementation reviews", p.current)
		g.Count("implementation files bound", p.files)
		g.Count("scope entries bound", p.entries)
		g.Notes = append(g.Notes, attestationScopeLimits)
	}
}

// CheckAttestationsWithImplementation returns only finalized review findings.
func CheckAttestationsWithImplementation(design, impl string) *Gate {
	s, err := AcquireSnapshot(design)
	if err != nil {
		return &Gate{Title: "Gv-attest  attestation evidence", Errs: []string{"GV_SCOPE_CUSTODY: " + err.Error()}}
	}
	var subject *attestationSubject
	if impl != "" {
		subject, _, err = s.captureAttestationSubject(impl)
	}
	var g *Gate
	if err == nil {
		g = checkAttestationsInSnapshot(s.design, subject)
	} else {
		g = &Gate{Title: "Gv-attest  attestation evidence", Errs: []string{err.Error()}}
	}
	finalErr := s.Release()
	if finalErr != nil && len(s.attestationPending) == 0 {
		g.Errs = append(g.Errs, "GV_SCOPE_CUSTODY: "+finalErr.Error())
	}
	remapGatePaths([]*Gate{g}, s.design, s.logicalDesign)
	return g
}

// RenderAttestation buffers a complete v2 document. No bytes escape until all
// validation and owned cleanup finish. It neither runs tests nor signs a review.
func RenderAttestation(design, impl string, review AttestationReview) ([]byte, error) {
	s, err := AcquireSnapshot(design)
	if err != nil {
		return nil, fmt.Errorf("GV_SCOPE_CUSTODY: %w", err)
	}
	var subject *attestationSubject
	if impl != "" && review.Kind == "current" {
		subject, _, err = s.captureAttestationSubject(impl)
	} else if impl != "" {
		err = fmt.Errorf("GV_KIND: --impl is forbidden for plan/historical generation")
	}
	var body []byte
	if err == nil {
		body, err = renderAttestationInSnapshot(s.design, subject, review)
	}
	// Even plan/history generation validates held design custody before release.
	err = errors.Join(err, s.CheckUnchanged(), s.Release())
	if err != nil {
		return nil, s.LogicalError(err)
	}
	return body, nil
}

func renderAttestationInSnapshot(design string, subject *attestationSubject, review AttestationReview) ([]byte, error) {
	class, known := attestationClaimKinds[review.Claim]
	if !known || review.Kind != class && !(class == "current" && review.Kind == "plan") {
		return nil, fmt.Errorf("GV_KIND: unknown claim or incompatible kind for %s", review.Claim)
	}
	if review.Kind == "current" && subject == nil {
		return nil, fmt.Errorf("GV_IMPL_REQUIRED: current generation requires --impl <reviewed-root>")
	}
	if strings.TrimSpace(review.Attestor) == "" {
		return nil, fmt.Errorf("GV_SCHEMA: attestor is required")
	}
	if _, err := time.Parse("2006-01-02", review.Date); err != nil {
		return nil, fmt.Errorf("GV_SCHEMA: date must be an explicit real YYYY-MM-DD date")
	}
	g := NewGate("Gv-attest")
	paths := attestationRequiredPaths(g, design, review.Claim)
	if len(paths) == 0 || len(g.Errs) > 0 {
		return nil, fmt.Errorf("GV_MISSING_IMPLEMENTATION_SUBJECT: %s has no available required design subject: %s", review.Claim, strings.Join(g.Errs, "; "))
	}
	covers := make([]map[string]string, 0, len(paths))
	for _, path := range paths {
		if err := portablepath.ValidateRelative(path); err != nil {
			return nil, fmt.Errorf("GV_SCOPE_PATH: %w", err)
		}
		body, err := readDesignFile(design, filepath.Join(design, filepath.FromSlash(path)))
		if err != nil {
			return nil, fmt.Errorf("GV_SCOPE_CUSTODY: read required subject %s: %w", path, err)
		}
		covers = append(covers, map[string]string{"path": path, "hash": fmt.Sprintf("sha256:%x", sha256.Sum256(body))})
	}
	row := map[string]any{"claim": review.Claim, "kind": review.Kind, "attestor": review.Attestor, "date": review.Date, "covers": covers}
	if review.Note != "" {
		row["note"] = review.Note
	}
	if review.Kind == "current" {
		row["implementation"] = subject.manifest
	}
	body, err := yaml.Marshal(map[string]any{"attestation_version": 2, "attestations": []any{row}})
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > designArtifactMaxBytes {
		return nil, fmt.Errorf("GV_EVIDENCE_LIMIT: generated document has %d bytes, maximum %d; merged rows must also fit this document limit", len(body), designArtifactMaxBytes)
	}
	return body, nil
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
		owed := claim.owed != nil && claim.owed(design)
		if claim.owedChecked != nil {
			checkedOwed, err := claim.owedChecked(design)
			if err != nil {
				g.Errs = append(g.Errs, "cannot determine whether "+claim.id+" is owed: "+err.Error())
				continue
			}
			owed = checkedOwed
		}
		if !owed {
			continue
		}
		g.Count("claims owed")
		if attested[claim.id] {
			continue
		}
		g.Warns = append(g.Warns, fmt.Sprintf("%s has no row for %s, which %s makes owed: %s", AttestationsFileName, claim.id, claim.owedBy, claim.what))
	}
}
