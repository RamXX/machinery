// Package threat reads the design's threat ledger (design/threats.yaml) and
// detects the domain-model subjects that look security-relevant. It is the
// shared vocabulary behind Gz-threat (the ledger itself), Gb-plan (paired
// criteria and the RED threat table) and Ga-accept (the threat-first review),
// so the three gates can never disagree on what an adversary is or which
// subjects owe a classification. See docs/threat-driven-verification.md.
package threat

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/RamXX/machinery/internal/ir"
)

// FileName is the ledger's name directly under the design directory.
const FileName = "threats.yaml"

// Mode says whether missing classifications block (enforce) or are listed
// as notes (audit). An absent ledger is audit.
type Mode string

const (
	ModeAudit   Mode = "audit"
	ModeEnforce Mode = "enforce"
)

// Adversaries is the closed standard adversary set, in canonical order.
// Every classified subject lists each exactly once.
var Adversaries = []string{
	"outsider_without_key",
	"trusted_key_holder",
	"unsigned_document_controller",
	"valid_signature_wrong_meaning",
	"optional_input_omission",
	"duplicate_replay_reorder",
	"signed_field_injection",
	"cross_bundle_substitution",
}

// Kinds is the closed vocabulary of what makes a subject security-relevant.
var Kinds = []string{"verifies", "authenticates", "signs", "hashes", "approves", "admits_input", "gates_decision"}

// AcceptedRisk is an owner-signed acceptance that the component's own output
// makes visible to its caller.
type AcceptedRisk struct {
	Owner, Date, Reason, VisibleIn string
}

// Row is one adversary's disposition on one subject: a threat invariant with
// its locked negative test, an accepted risk, or not applicable.
type Row struct {
	Adversary     string
	Invariant     string
	NegativeTest  string
	NotApplicable string
	Risk          *AcceptedRisk
}

// Covered reports whether the row is closed by a negative test (the only
// disposition a RED threat table must bind to a test).
func (r Row) Covered() bool { return r.Invariant != "" && r.NegativeTest != "" }

// Subject is one classified entity or Entity.action.
type Subject struct {
	Name  string
	Kinds []string
	Rows  []Row
}

// Exemption declares a detected candidate not security-relevant.
type Exemption struct{ Subject, Reason string }

// Waiver is a brownfield, owner-signed deferral of classification.
type Waiver struct{ Subject, Owner, Date, Reason string }

// Verifier names a C4 element that makes security decisions.
type Verifier struct{ Component, Decides string }

// Ledger is the parsed threats.yaml. Present is false for an absent file.
type Ledger struct {
	Present       bool
	Mode          Mode
	EnforcedSince string
	Subjects      []Subject
	NotRelevant   []Exemption
	Waivers       []Waiver
	Verifiers     []Verifier
}

// Enforcing reports whether missing classifications are errors.
func (l *Ledger) Enforcing() bool { return l != nil && l.Mode == ModeEnforce }

var (
	topKeys      = keys("mode", "enforced_since", "subjects", "not_security_relevant", "waivers", "verifiers", "_comment")
	subjectKeys  = keys("subject", "kinds", "adversaries", "_comment")
	rowKeys      = keys("adversary", "invariant", "negative_test", "accepted_risk", "not_applicable", "_comment")
	riskKeys     = keys("owner", "date", "reason", "visible_in")
	exemptKeys   = keys("subject", "reason", "_comment")
	waiverKeys   = keys("subject", "owner", "date", "reason", "_comment")
	verifierKeys = keys("component", "decides", "_comment")
	dateRE       = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

func keys(ks ...string) map[string]bool {
	m := map[string]bool{}
	for _, k := range ks {
		m[k] = true
	}
	return m
}

// Parse validates a ledger's own structure. Model resolution (do subjects
// and invariants exist?) is Resolve's job. The returned ledger is never nil
// unless the bytes are not a YAML mapping; problems are complete sentences
// prefixed with the file name.
func Parse(data []byte) (*Ledger, []string) {
	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, FileName+": "+fmt.Sprintf(format, args...))
	}
	v, err := ir.LoadYAML(data)
	if err != nil {
		add("invalid YAML: %s", err)
		return nil, problems
	}
	root := v.AsObject()
	if root == nil {
		add("not a yaml mapping (empty file?)")
		return nil, problems
	}
	l := &Ledger{Present: true, Mode: ModeAudit}
	unknownKeys(root, topKeys, "top level", add)
	switch m := root.GetString("mode"); m {
	case "", string(ModeAudit):
	case string(ModeEnforce):
		l.Mode = ModeEnforce
	default:
		add("mode must be audit or enforce, got %s", ir.Repr(m))
	}
	l.EnforcedSince = root.GetString("enforced_since")
	if l.EnforcedSince != "" && !dateRE.MatchString(l.EnforcedSince) {
		add("enforced_since must be YYYY-MM-DD, got %s", ir.Repr(l.EnforcedSince))
	}
	if l.Mode == ModeEnforce && l.EnforcedSince == "" {
		add("mode enforce needs enforced_since (YYYY-MM-DD): the date the threat-first review becomes mandatory for milestone acceptances")
	}

	listed := map[string]int{}
	note := func(subject string) {
		if subject != "" {
			listed[subject]++
		}
	}
	for i, sv := range items(root.Get2("subjects")) {
		so := sv.AsObject()
		if so == nil {
			add("subjects[%d] is not a mapping", i)
			continue
		}
		unknownKeys(so, subjectKeys, fmt.Sprintf("subjects[%d]", i), add)
		s := Subject{Name: so.GetString("subject")}
		if s.Name == "" {
			add("subjects[%d] needs a subject (an entity or Entity.action)", i)
			continue
		}
		note(s.Name)
		s.Kinds = strs(so.Get2("kinds"))
		if len(s.Kinds) == 0 {
			add("%s: kinds is empty; name at least one of %s", s.Name, strings.Join(Kinds, ", "))
		}
		for _, k := range s.Kinds {
			if !contains(Kinds, k) {
				add("%s: unknown kind %s (expected one of %s)", s.Name, ir.Repr(k), strings.Join(Kinds, ", "))
			}
		}
		seen := map[string]bool{}
		for j, rv := range items(so.Get2("adversaries")) {
			ro := rv.AsObject()
			if ro == nil {
				add("%s: adversaries[%d] is not a mapping", s.Name, j)
				continue
			}
			unknownKeys(ro, rowKeys, fmt.Sprintf("%s adversaries[%d]", s.Name, j), add)
			r := parseRow(s.Name, ro, add)
			if r.Adversary == "" {
				add("%s: adversaries[%d] needs an adversary", s.Name, j)
				continue
			}
			if !contains(Adversaries, r.Adversary) {
				add("%s: unknown adversary %s (expected one of %s)", s.Name, r.Adversary, strings.Join(Adversaries, ", "))
				continue
			}
			if seen[r.Adversary] {
				add("%s: adversary %s is classified twice", s.Name, r.Adversary)
				continue
			}
			seen[r.Adversary] = true
			s.Rows = append(s.Rows, r)
		}
		for _, a := range Adversaries {
			if !seen[a] {
				add("%s: adversary %s is not classified; every security-relevant subject classifies the whole standard set", s.Name, a)
			}
		}
		l.Subjects = append(l.Subjects, s)
	}
	for i, ev := range items(root.Get2("not_security_relevant")) {
		eo := ev.AsObject()
		if eo == nil {
			add("not_security_relevant[%d] is not a mapping", i)
			continue
		}
		unknownKeys(eo, exemptKeys, fmt.Sprintf("not_security_relevant[%d]", i), add)
		e := Exemption{Subject: eo.GetString("subject"), Reason: eo.GetString("reason")}
		if e.Subject == "" || strings.TrimSpace(e.Reason) == "" {
			add("not_security_relevant[%d] needs subject and reason", i)
			continue
		}
		note(e.Subject)
		l.NotRelevant = append(l.NotRelevant, e)
	}
	for i, wv := range items(root.Get2("waivers")) {
		wo := wv.AsObject()
		if wo == nil {
			add("waivers[%d] is not a mapping", i)
			continue
		}
		unknownKeys(wo, waiverKeys, fmt.Sprintf("waivers[%d]", i), add)
		w := Waiver{Subject: wo.GetString("subject"), Owner: wo.GetString("owner"), Date: wo.GetString("date"), Reason: wo.GetString("reason")}
		if w.Subject == "" {
			add("waivers[%d] needs a subject", i)
			continue
		}
		note(w.Subject)
		if w.Owner == "" || w.Date == "" || strings.TrimSpace(w.Reason) == "" {
			add("waiver for %s needs owner, date, and reason; an unsigned waiver is a hole", w.Subject)
			continue
		}
		if !dateRE.MatchString(w.Date) {
			add("waiver for %s: date must be YYYY-MM-DD, got %s", w.Subject, ir.Repr(w.Date))
		}
		l.Waivers = append(l.Waivers, w)
	}
	for i, vv := range items(root.Get2("verifiers")) {
		vo := vv.AsObject()
		if vo == nil {
			add("verifiers[%d] is not a mapping", i)
			continue
		}
		unknownKeys(vo, verifierKeys, fmt.Sprintf("verifiers[%d]", i), add)
		ver := Verifier{Component: vo.GetString("component"), Decides: vo.GetString("decides")}
		if ver.Component == "" || strings.TrimSpace(ver.Decides) == "" {
			add("verifiers[%d] needs component and decides", i)
			continue
		}
		l.Verifiers = append(l.Verifiers, ver)
	}
	var multi []string
	for s, n := range listed {
		if n > 1 {
			multi = append(multi, s)
		}
	}
	sort.Strings(multi)
	for _, s := range multi {
		add("%s is listed more than once across subjects, not_security_relevant, and waivers; it has exactly one classification", s)
	}
	return l, problems
}

func parseRow(subject string, ro *ir.Object, add func(string, ...any)) Row {
	r := Row{
		Adversary:     ro.GetString("adversary"),
		Invariant:     ro.GetString("invariant"),
		NegativeTest:  ro.GetString("negative_test"),
		NotApplicable: ro.GetString("not_applicable"),
	}
	if rk := ro.GetObject("accepted_risk"); rk != nil {
		unknownKeys(rk, riskKeys, subject+" "+r.Adversary+" accepted_risk", add)
		r.Risk = &AcceptedRisk{Owner: rk.GetString("owner"), Date: rk.GetString("date"), Reason: rk.GetString("reason"), VisibleIn: rk.GetString("visible_in")}
	} else if ro.Has("accepted_risk") {
		add("%s %s: accepted_risk must be a mapping of owner, date, reason, visible_in", subject, r.Adversary)
		r.Risk = &AcceptedRisk{}
	}
	where := subject + " " + r.Adversary
	n := 0
	if r.Invariant != "" || r.NegativeTest != "" {
		n++
	}
	if r.Risk != nil {
		n++
	}
	if ro.Has("not_applicable") {
		n++
	}
	switch {
	case n != 1:
		add("%s: a row takes exactly one disposition: invariant plus negative_test, accepted_risk, or not_applicable", where)
	case r.Invariant != "" || r.NegativeTest != "":
		if r.Invariant == "" || r.NegativeTest == "" {
			add("%s: a tested row needs both invariant and negative_test", where)
		}
		if Documented(r.NegativeTest) {
			add("%s: negative_test \"documented\" is not a closure; name the locked negative test", where)
		} else if AgreementOnly(r.NegativeTest) {
			add("%s: negative_test %s only claims agreement; agreement with a reference is not correctness, name an independent negative test", where, ir.Repr(r.NegativeTest))
		}
	case r.Risk != nil:
		k := r.Risk
		switch {
		case k.Owner == "":
			add("%s: accepted_risk needs owner", where)
		case k.Date == "" || !dateRE.MatchString(k.Date):
			add("%s: accepted_risk needs date as YYYY-MM-DD", where)
		case strings.TrimSpace(k.Reason) == "":
			add("%s: accepted_risk needs reason", where)
		case strings.TrimSpace(k.VisibleIn) == "":
			add("%s: accepted_risk needs visible_in: where the component's own output shows the risk to its caller; a risk no caller can see is hidden, not accepted", where)
		}
		if Documented(k.Reason) || Documented(k.VisibleIn) {
			add("%s: \"documented\" is not a closure; an accepted risk is visible in the component's output", where)
		}
	default:
		if strings.TrimSpace(r.NotApplicable) == "" {
			add("%s: not_applicable needs a reason", where)
		} else if Documented(r.NotApplicable) {
			add("%s: \"documented\" is not a closure; fix the gap or record an owner-signed accepted_risk", where)
		}
	}
	return r
}

// Documented reports a disposition that closes a gap by pointing at prose.
func Documented(s string) bool {
	t := strings.ToLower(strings.TrimSpace(s))
	return t == "documented" || strings.HasPrefix(t, "documented ") || strings.HasPrefix(t, "documented:") || strings.HasPrefix(t, "see doc")
}

var agreementPhrases = []string{
	"matches the reference", "matches reference", "match the reference", "agrees with", "agreement with",
	"same as the reference", "same output as", "reference implementation", "parity with", "differential against",
}

// AgreementOnly reports text that claims correctness only by agreement with
// another implementation. A test id or path never contains a space, so a
// bare identifier is never agreement-only.
func AgreementOnly(s string) bool {
	t := strings.ToLower(strings.TrimSpace(s))
	if !strings.ContainsAny(t, " \t") {
		return false
	}
	for _, p := range agreementPhrases {
		if strings.Contains(t, p) {
			return true
		}
	}
	return false
}

// Covers reports whether subject (an entity or Entity.action) has a
// classification: listed itself, or its entity listed, in any section.
func (l *Ledger) Covers(subject string) bool {
	if l == nil {
		return false
	}
	entity, _, _ := strings.Cut(subject, ".")
	for _, name := range l.listedNames() {
		if name == subject || name == entity {
			return true
		}
	}
	return false
}

func (l *Ledger) listedNames() []string {
	var out []string
	for _, s := range l.Subjects {
		out = append(out, s.Name)
	}
	for _, e := range l.NotRelevant {
		out = append(out, e.Subject)
	}
	for _, w := range l.Waivers {
		out = append(out, w.Subject)
	}
	return out
}

// ThreatInvariants is the set of invariant ids the ledger's rows produce.
func (l *Ledger) ThreatInvariants() map[string]bool {
	ids := map[string]bool{}
	if l == nil {
		return ids
	}
	for _, s := range l.Subjects {
		for _, r := range s.Rows {
			if r.Invariant != "" {
				ids[r.Invariant] = true
			}
		}
	}
	return ids
}

// Resolve binds the ledger to the domain model: every listed subject is a
// declared entity or Entity.action, and every row's invariant is declared.
func (l *Ledger) Resolve(model *ir.Object) []string {
	if l == nil || model == nil {
		return nil
	}
	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, FileName+": "+fmt.Sprintf(format, args...))
	}
	subjects := ModelSubjects(model)
	for _, name := range l.listedNames() {
		if !subjects[name] {
			add("subject %s is not an entity or Entity.action the model declares", name)
		}
	}
	invariants := ModelInvariants(model)
	for _, s := range l.Subjects {
		for _, r := range s.Rows {
			if r.Invariant != "" && !invariants[r.Invariant] {
				add("%s: adversary %s names invariant %s, which the model does not declare; a threat invariant is a model invariant so Gc can hold its carrier", s.Name, r.Adversary, r.Invariant)
			}
		}
	}
	return problems
}

// ModelSubjects is the set of entity names and Entity.action ids.
func ModelSubjects(model *ir.Object) map[string]bool {
	out := map[string]bool{}
	entities := model.GetObject("entities")
	if entities == nil {
		return out
	}
	for _, name := range entities.Keys() {
		out[name] = true
		e := entities.Get2(name).AsObject()
		if e == nil {
			continue
		}
		for _, a := range items(e.Get2("actions")) {
			if an := actionName(a); an != "" {
				out[name+"."+an] = true
			}
		}
	}
	return out
}

// ModelInvariants is the set of declared invariant ids, top-level and
// per-entity (one global id space, as Gc holds it).
func ModelInvariants(model *ir.Object) map[string]bool {
	out := map[string]bool{}
	for _, iv := range items(model.Get2("invariants")) {
		if o := iv.AsObject(); o != nil && o.GetString("id") != "" {
			out[o.GetString("id")] = true
		}
	}
	if entities := model.GetObject("entities"); entities != nil {
		for _, name := range entities.Keys() {
			e := entities.Get2(name).AsObject()
			if e == nil {
				continue
			}
			for _, iv := range items(e.Get2("invariants")) {
				if o := iv.AsObject(); o != nil && o.GetString("id") != "" {
					out[o.GetString("id")] = true
				}
			}
		}
	}
	return out
}

// Candidate is a model subject whose own wording marks it security-relevant.
type Candidate struct {
	Subject  string // Entity or Entity.action
	Entity   string
	Evidence string // the word and where it was found
	Hash     string // EntityHash of the owning entity
}

// Detect lists the entities and actions whose names, definitions, attribute
// names, or action names and descriptions use security vocabulary, sorted by
// subject. An entity that matches covers its actions, so only the entity is
// listed; otherwise each matching action is listed on its own.
func Detect(model *ir.Object) []Candidate {
	var out []Candidate
	entities := model.GetObject("entities")
	if entities == nil {
		return nil
	}
	for _, name := range entities.Keys() {
		e := entities.Get2(name).AsObject()
		if e == nil {
			continue
		}
		hash := EntityHash(model, name)
		evidence := firstWord("entity name", name)
		if evidence == "" {
			evidence = firstWord("definition", e.GetString("definition"))
		}
		if evidence == "" {
			for _, a := range items(e.Get2("attributes")) {
				if ao := a.AsObject(); ao != nil {
					if evidence = firstWord("attribute "+ao.GetString("name"), ao.GetString("name")); evidence != "" {
						break
					}
				}
			}
		}
		if evidence != "" {
			out = append(out, Candidate{Subject: name, Entity: name, Evidence: evidence, Hash: hash})
			continue
		}
		for _, a := range items(e.Get2("actions")) {
			an := actionName(a)
			if an == "" {
				continue
			}
			ev := firstWord("action name", an)
			if ev == "" {
				if ao := a.AsObject(); ao != nil {
					ev = firstWord("action description", ao.GetString("description"))
				}
			}
			if ev != "" {
				out = append(out, Candidate{Subject: name + "." + an, Entity: name, Evidence: ev, Hash: hash})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Subject < out[j].Subject })
	return out
}

func firstWord(where, text string) string {
	for _, w := range Words(text) {
		if SecurityWord(w) {
			return fmt.Sprintf("%q in %s", w, where)
		}
	}
	return ""
}

// EntityHash is a content hash of one entity's model definition, so a
// baselined candidate re-arms when its definition changes.
func EntityHash(model *ir.Object, entity string) string {
	entities := model.GetObject("entities")
	if entities == nil || !entities.Has(entity) {
		return ""
	}
	sum := sha256.Sum256([]byte(entity + "\x00" + ir.Repr(entities.Get2(entity))))
	return fmt.Sprintf("sha256:%x", sum[:8])
}

func actionName(a *ir.Value) string {
	if a == nil {
		return ""
	}
	if a.Kind == ir.KindString {
		return a.AsString()
	}
	if o := a.AsObject(); o != nil {
		return o.GetString("name")
	}
	return ""
}

// Words splits identifiers and prose into lower-case words: on every
// non-alphanumeric rune, at lower-to-upper camel boundaries, and before the
// last capital of an acronym run (APIKey is api, key).
func Words(s string) []string {
	var out []string
	rs := []rune(s)
	start := -1
	flush := func(end int) {
		if start >= 0 && end > start {
			out = append(out, strings.ToLower(string(rs[start:end])))
		}
		start = -1
	}
	for i, r := range rs {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush(i)
			continue
		}
		if start < 0 {
			start = i
			continue
		}
		prev := rs[i-1]
		switch {
		case unicode.IsUpper(r) && unicode.IsLower(prev):
			flush(i)
			start = i
		case unicode.IsUpper(r) && unicode.IsUpper(prev) && i+1 < len(rs) && unicode.IsLower(rs[i+1]):
			flush(i)
			start = i
		}
	}
	flush(len(rs))
	return out
}

var (
	securityExact = keys("sign", "signs", "signed", "signing", "signer", "signers", "signoff", "login", "logon", "signin",
		"hash", "hashes", "hashed", "hashing", "admit", "admits", "admitted", "token", "tokens", "secret", "secrets",
		"permission", "permissions", "nonce", "hmac", "mfa")
	securityStems = []string{"verif", "authentic", "credential", "countersign", "attest", "authoriz", "authoris",
		"certificat", "approv", "admissi", "passw", "passphrase", "signatur", "digest", "notariz", "provenance"}
	verifierStems = []string{"verif", "checker", "attest", "signer", "signing", "gatekeep", "validator", "authoriz", "authoris", "authentic"}
	verifierExact = keys("gate", "gates", "authz", "authn", "auth")
)

// SecurityWord reports whether one lower-case word is security vocabulary.
func SecurityWord(w string) bool {
	if securityExact[w] {
		return true
	}
	for _, s := range securityStems {
		if strings.HasPrefix(w, s) {
			return true
		}
	}
	return false
}

// VerifierWord reports whether one lower-case word marks an architecture
// element that makes security decisions.
func VerifierWord(w string) bool {
	if verifierExact[w] {
		return true
	}
	for _, s := range verifierStems {
		if strings.HasPrefix(w, s) {
			return true
		}
	}
	return false
}

// VerifierEvidence returns the first verifier word in text, or "".
func VerifierEvidence(text string) string {
	for _, w := range Words(text) {
		if VerifierWord(w) {
			return w
		}
	}
	return ""
}

func unknownKeys(o *ir.Object, allowed map[string]bool, where string, add func(string, ...any)) {
	for _, k := range o.Keys() {
		if !allowed[k] {
			add("%s: unknown key %s (a typo here silently classifies nothing)", where, ir.Repr(k))
		}
	}
}

func items(v *ir.Value) []*ir.Value {
	if v == nil || v.Kind != ir.KindArray {
		return nil
	}
	return v.AsArray()
}

func strs(v *ir.Value) []string {
	if v == nil {
		return nil
	}
	if v.Kind == ir.KindString {
		return []string{v.AsString()}
	}
	var out []string
	for _, it := range items(v) {
		if it != nil && it.Kind == ir.KindString {
			out = append(out, it.AsString())
		}
	}
	return out
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if x == y {
			return true
		}
	}
	return false
}
