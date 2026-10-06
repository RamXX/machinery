// Gz-threat: adversary classification. The domain model says what the
// product does; nothing in it says who attacks it. This gate holds the
// sidecar that does, design/threats.yaml (see internal/threat and
// docs/threat-driven-verification.md): every model subject whose wording
// looks security-relevant is classified against the standard adversary set,
// declared not security-relevant with a reason, or waived by an owner.
//
// The ledger is optional. Without it, or with mode: audit, every missing
// classification is an "audit:" NOTE and nothing blocks; mode: enforce makes
// them ERRORs. What an adopter wrote is never audit-softened: a ledger that
// does not parse or does not resolve against the model is an ERROR in any
// mode, and so is a threat invariant closed by a formal/waivers.yaml note or
// a declared verifier the Architecture Contract does not govern.
//
// Migration: `machinery baseline --gate gz` records each unclassified
// candidate with the hash of its entity's model definition under
// threat_debt in ratchet.json. A recorded candidate whose definition is
// unchanged is a baselined NOTE, as Gy/Gl debt is; a changed definition or a
// new candidate is a finding again.

package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/threat"
)

const threatGateName = "Gz-threat"

// threatAuditNote tells an audit-mode reader how the notes become blocking.
const threatAuditNote = "audit: set mode: enforce and enforced_since in design/" + threat.FileName + " to make these blocking; see docs/threat-driven-verification.md"

// ThreatDebt is one baselined Gz-threat candidate: the subject (an entity or
// Entity.action) and the hash of its entity's model definition when it was
// recorded. A subject occurs once; a changed hash re-arms the finding.
type ThreatDebt struct {
	Subject string `json:"subject"`
	Hash    string `json:"hash"`
}

func (d ThreatDebt) key() string { return d.Subject + "\x00" + d.Hash }

// threatInputs is what both the gate and the baseline read: the model, the
// ledger (an absent file is an empty audit ledger), the workspace.dsl
// elements, and the ledger's parse and resolve problems.
type threatInputs struct {
	model    *ir.Object
	ledger   *threat.Ledger
	elements map[string]dslEl // nil when workspace.dsl is absent
	// elementExempt holds not_security_relevant entries that name a
	// workspace.dsl element rather than a model subject (A8 exemptions).
	elementExempt map[string]bool
	problems      []string
}

// loadThreatInputs reads the model (its errors land on g) and the ledger.
// It returns nil only when the model does not load.
func loadThreatInputs(design string, g *Gate) *threatInputs {
	dm := loadModelith(design, g)
	if dm == nil {
		return nil
	}
	in := &threatInputs{model: dm.AsObject(), elementExempt: map[string]bool{}}
	data, err := readDesignFile(design, filepath.Join(design, threat.FileName))
	switch {
	case os.IsNotExist(err):
		in.ledger = &threat.Ledger{Mode: threat.ModeAudit}
	case err != nil:
		in.problems = append(in.problems, threat.FileName+" is unreadable: "+err.Error()+" (the ledger must be a regular file inside the design)")
		in.ledger = &threat.Ledger{Mode: threat.ModeAudit}
	default:
		l, problems := threat.Parse(data)
		in.problems = append(in.problems, problems...)
		if l == nil {
			l = &threat.Ledger{Present: true, Mode: threat.ModeAudit}
		}
		in.ledger = l
	}
	dslPath := filepath.Join(design, "workspace.dsl")
	if text, err := readDesignFile(design, dslPath); err == nil {
		in.elements = dslElementsOf(string(text))
	} else if !os.IsNotExist(err) {
		g.Errs = append(g.Errs, "workspace.dsl is unreadable: "+err.Error())
	}
	// a not_security_relevant entry may name a workspace.dsl element (A8)
	// instead of a model subject; only the rest must resolve against the model
	subjects := threat.ModelSubjects(in.model)
	resolvable := *in.ledger
	resolvable.NotRelevant = nil
	for _, e := range in.ledger.NotRelevant {
		if _, isElement := in.elements[e.Subject]; isElement && !subjects[e.Subject] {
			in.elementExempt[e.Subject] = true
			continue
		}
		resolvable.NotRelevant = append(resolvable.NotRelevant, e)
	}
	in.problems = append(in.problems, resolvable.Resolve(in.model)...)
	return in
}

// uncovered lists the detected candidates the ledger does not classify.
func (in *threatInputs) uncovered() (all, open []threat.Candidate) {
	all = threat.Detect(in.model)
	for _, c := range all {
		if !in.ledger.Covers(c.Subject) {
			open = append(open, c)
		}
	}
	return all, open
}

// CheckThreats implements Gz-threat. impl, when set, holds every locked
// negative test the ledger names to the implementation's test files.
func CheckThreats(design, impl string) *Gate {
	g := NewGate(threatGateName + "  adversary classification")
	g.startOrder()
	in := loadThreatInputs(design, g)
	if in == nil {
		return g
	}
	g.Errs = append(g.Errs, in.problems...)
	l := in.ledger
	enforce := l.Enforcing()
	audits := 0
	finding := func(msg string) {
		if enforce {
			g.Errs = append(g.Errs, msg)
			return
		}
		audits++
		g.Notes = append(g.Notes, "audit: "+msg)
	}

	checkThreatClassification(g, design, in, finding)

	g.Count("subjects classified", len(l.Subjects))
	for _, s := range l.Subjects {
		g.Count("adversary rows", len(s.Rows))
	}

	// A7: a threat invariant is closed by a test or an owner-signed risk,
	// never by a waiver note in the carrier annex
	if l.Present {
		annex := annexWaiverIDs(design)
		for _, s := range l.Subjects {
			for _, r := range s.Rows {
				if r.Invariant != "" && annex[r.Invariant] {
					g.Errs = append(g.Errs, fmt.Sprintf("formal/%s waives threat invariant %s (%s, %s): a threat invariant cannot be closed by a waiver note; fix it or record an owner-signed accepted_risk in %s",
						WaiverAnnexName, ir.Repr(r.Invariant), s.Name, r.Adversary, threat.FileName))
				}
			}
		}
	}

	checkThreatVerifiers(g, design, in, finding)

	if impl != "" {
		checkNegativeTests(g, design, impl, l, finding)
	}
	if audits > 0 {
		g.Notes = append(g.Notes, threatAuditNote)
	}
	return g
}

// checkThreatClassification is A1: every detected candidate is classified,
// or recorded in the ratchet with an unchanged model definition.
func checkThreatClassification(g *Gate, design string, in *threatInputs, finding func(string)) {
	all, open := in.uncovered()
	g.Count("candidates detected", len(all))
	g.Count("candidates classified", len(all)-len(open))
	ratchet, err := LoadRatchet(design)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
	}
	recorded := map[string]ThreatDebt{}
	var order []ThreatDebt
	if ratchet != nil {
		order = ratchet.Threats
		for _, d := range ratchet.Threats {
			recorded[d.Subject] = d
		}
	}
	observed := map[string]bool{}
	baselined := 0
	for _, c := range open {
		msg := fmt.Sprintf("%s looks security-relevant (%s) and is not classified in design/%s; list it under subjects with its adversary rows, under not_security_relevant with a reason, or under waivers with an owner, date, and reason",
			c.Subject, c.Evidence, threat.FileName)
		if d, ok := recorded[c.Subject]; ok {
			observed[c.Subject] = true
			if d.Hash == c.Hash {
				baselined++
				g.Notes = append(g.Notes, baselinedPrefix+msg)
				continue
			}
			msg += fmt.Sprintf("; its model definition changed since the baseline (%s recorded %s, now %s), so the classification is owed again", RatchetFile, d.Hash, c.Hash)
		}
		finding(msg)
	}
	resolved := 0
	for _, d := range order {
		if !observed[d.Subject] {
			resolved++
			g.Notes = append(g.Notes, "baselined threat candidate "+d.Subject+resolvedSuffix)
		}
	}
	if baselined > 0 {
		g.Count(BaselinedCount, baselined)
	}
	if resolved > 0 {
		g.Count(resolvedCount, resolved)
	}
}

// checkThreatVerifiers is A8: every declared verifier is a workspace.dsl
// element the Architecture Contract governs, and every element whose wording
// marks it a verifier is declared or exempted.
func checkThreatVerifiers(g *Gate, design string, in *threatInputs, finding func(string)) {
	l := in.ledger
	declared := map[string]bool{}
	var governance func(string) string
	for _, v := range l.Verifiers {
		g.Count("verifiers")
		declared[v.Component] = true
		el, ok := in.elements[v.Component]
		if !ok {
			g.Errs = append(g.Errs, fmt.Sprintf("verifier %s is not an element workspace.dsl declares; declare it as a component, container, or softwareSystem so the Architecture Contract can govern its imports", v.Component))
			continue
		}
		if el.Kind == "person" {
			g.Errs = append(g.Errs, fmt.Sprintf("verifier %s is a person in workspace.dsl; a verifier is a component, container, or softwareSystem whose code the Architecture Contract governs", v.Component))
			continue
		}
		if governance == nil {
			governance = contractGovernance(design)
		}
		if why := governance(v.Component); why != "" {
			g.Errs = append(g.Errs, fmt.Sprintf("verifier %s is not governed by the Architecture Contract: %s", v.Component, why))
		}
	}
	names := make([]string, 0, len(in.elements))
	for name := range in.elements {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		el := in.elements[name]
		if el.Kind == "person" {
			continue
		}
		g.Count("workspace.dsl elements scanned")
		ev := threat.VerifierEvidence(name + " " + el.Display + " " + el.Description)
		if ev == "" || declared[name] || in.elementExempt[name] {
			continue
		}
		finding(fmt.Sprintf("workspace.dsl element %s speaks of %q in its name or description and is neither a declared verifier nor listed under not_security_relevant; declare it under verifiers with what it decides, or list it under not_security_relevant with a reason",
			name, ev))
	}
}

// contractGovernance loads the Architecture Contract once and returns a
// lookup that explains why an element's code is not held to the contract's
// import rules ("" when a boundary with code globs binds it). G2-c4 reports
// the contract's own defects; here a contract that does not load governs
// nothing.
func contractGovernance(design string) func(string) string {
	cg := NewGate("_")
	c := loadContract(design, filepath.Join(design, "ARCHITECTURE.md"), cg)
	if c == nil || len(cg.Errs) > 0 {
		return func(string) string {
			return "ARCHITECTURE.md carries no valid Architecture Contract (G2-c4 reports why), so no import rule holds its code"
		}
	}
	co := c.AsObject()
	boundaries := objSlice(co.Get2("boundaries"))
	externals := objSlice(co.Get2("externals"))
	bindings := elementBindings(boundaries, externals)
	withCode := map[string]bool{}
	for _, b := range boundaries {
		if bo := b.AsObject(); bo != nil && bo.GetString("id") != "" && len(listStrings(bo.Get2("code"))) > 0 {
			withCode[bo.GetString("id")] = true
		}
	}
	isBoundary := map[string]bool{}
	for _, b := range boundaries {
		if bo := b.AsObject(); bo != nil {
			isBoundary[bo.GetString("id")] = true
		}
	}
	return func(element string) string {
		id, ok := bindings[element]
		switch {
		case !ok:
			return fmt.Sprintf("no boundary binds element %s; bind it (element: %s) on a boundary with code globs so G4-import holds its imports", element, element)
		case !isBoundary[id]:
			return fmt.Sprintf("element %s binds only external %s, whose code the contract does not hold to import rules; a verifier is the design's own code, bound on a boundary with code globs", element, id)
		case !withCode[id]:
			return fmt.Sprintf("boundary %s binds element %s but declares no code globs, so G4-import cannot map its files", id, element)
		}
		return ""
	}
}

// checkNegativeTests is the --impl rule: every tested row's negative_test
// token (a stable id or a test file path) appears in the implementation's
// test files, read with Gt-tests' own corpus.
func checkNegativeTests(g *Gate, design, impl string, l *threat.Ledger, finding func(string)) {
	type want struct{ subject, adversary, token string }
	var wants []want
	for _, s := range l.Subjects {
		for _, r := range s.Rows {
			if r.Covered() {
				wants = append(wants, want{s.Name, r.Adversary, r.NegativeTest})
			}
		}
	}
	if len(wants) == 0 {
		return
	}
	corpus := testCorpus(design, impl, g)
	for _, w := range wants {
		if negativeTestFound(w.token, corpus) {
			g.Count("negative tests found")
			continue
		}
		finding(fmt.Sprintf("%s %s: negative_test %s appears in no test file under the implementation; write the locked negative test keyed on that id, or name its test file path",
			w.subject, w.adversary, ir.Repr(w.token)))
	}
}

func negativeTestFound(token string, corpus testCorpusData) bool {
	tok := filepath.ToSlash(strings.TrimPrefix(token, "./"))
	for _, f := range corpus.files {
		rel := filepath.ToSlash(f.rel)
		if rel == tok || strings.HasSuffix(rel, "/"+tok) {
			return true
		}
	}
	return idTokenIn(token, corpus.joinedCode)
}

// ObserveThreatDebt returns the Gz-threat candidates a design leaves
// unclassified today, keyed for the ratchet. It refuses (an error) when the
// model does not load or the ledger does not parse or resolve: a malformed
// ledger is a broken design, not debt.
func ObserveThreatDebt(design string) ([]ThreatDebt, error) {
	g := NewGate("_")
	in := loadThreatInputs(design, g)
	if in == nil {
		return nil, fmt.Errorf("the threat ledger needs the domain model: %s", strings.Join(g.Errs, "; "))
	}
	if len(in.problems) > 0 {
		return nil, fmt.Errorf("the threat ledger has %d problem(s); a ledger that does not parse or resolve is a broken design, not debt, and is never baselined; fix them and rerun:\n  %s",
			len(in.problems), strings.Join(in.problems, "\n  "))
	}
	_, open := in.uncovered()
	out := []ThreatDebt{}
	for _, c := range open {
		out = append(out, ThreatDebt{Subject: c.Subject, Hash: c.Hash})
	}
	return out, nil
}

// RecordThreatDebt rewrites r's threat_debt section from the design's
// current unclassified candidates, exactly as RecordConsistencyDebt does for
// Gy/Gl: an absent section records every candidate; otherwise only recorded
// entries still observed with the same hash are kept (the ratchet shrinks)
// unless grow is set. Nothing in r changes when an error is returned.
func RecordThreatDebt(design string, r *Ratchet, grow bool) (DebtRecord, error) {
	current, err := ObserveThreatDebt(design)
	if err != nil {
		return DebtRecord{}, err
	}
	rec := DebtRecord{Gate: threatGateName, Observed: len(current), First: r.Threats == nil}
	prior := map[string]bool{}
	for _, d := range r.Threats {
		prior[d.key()] = true
	}
	kept := map[string]bool{}
	debt := []ThreatDebt{}
	for _, d := range current {
		if rec.First || grow || prior[d.key()] {
			debt = append(debt, d)
			kept[d.key()] = true
		} else {
			rec.NotRecorded++
		}
	}
	for k := range prior {
		if !kept[k] {
			rec.Dropped++
		}
	}
	rec.Recorded = len(debt)
	r.Threats = debt
	return rec, nil
}

// decodeThreatDebt reads the threat_debt section strictly: an array of
// {subject, hash} objects, both non-empty, no subject twice.
func decodeThreatDebt(raw []byte) ([]ThreatDebt, error) {
	var entries []ThreatDebt
	if err := decodeStrictArray(raw, &entries); err != nil {
		return nil, err
	}
	out := []ThreatDebt{}
	seen := map[string]bool{}
	for i, e := range entries {
		switch {
		case e.Subject == "":
			return nil, fmt.Errorf("entry %d: subject is required", i)
		case e.Hash == "":
			return nil, fmt.Errorf("entry %d: hash is required", i)
		}
		if seen[e.Subject] {
			return nil, fmt.Errorf("entry %d: duplicate subject %s; one entry per subject", i, e.Subject)
		}
		seen[e.Subject] = true
		out = append(out, e)
	}
	return out, nil
}
