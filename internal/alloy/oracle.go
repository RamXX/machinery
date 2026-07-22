// The authorization oracle: the policy annotation compiled a second way, as
// an enumerated role x verb x ownership-case decision table that
// implementation tests consume, exactly as machine tests consume the
// transition oracles. The Alloy model proves the invariant set coherent; this
// table is the same semantics as concrete test cases, which is what holds the
// CODE to the policy. Content-derived stable ids identify the case (inputs),
// never the verdict, so a design revision flips expectations under stable
// keys and the test diff names exactly which cases changed behavior.

package alloy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/RamXX/machinery/internal/version"
)

// OwnerCase is one algebra-distinguishable relationship between the acting
// subject and the record owner. The v1 scope algebra (all|own|team|none)
// evaluates every expression from exactly two booleans, so these cases are
// the complete semantics boundary: any two concrete configurations that agree
// on (Own, SameTeam) are policy-equivalent.
type OwnerCase struct {
	Key      string
	Own      bool // the actor owns the record
	SameTeam bool // sameTeam(actor, owner): both hold the same team
	// TeamlessActor marks the case as requiring the actor to hold no team,
	// which team-multiplicity invariants can forbid for some or all roles.
	TeamlessActor bool
}

func (p *Policy) ownerCases() []OwnerCase {
	if p.TeamEntity == "" {
		return []OwnerCase{
			{Key: "own", Own: true},
			{Key: "other"},
		}
	}
	return []OwnerCase{
		{Key: "own-teamed", Own: true, SameTeam: true},
		{Key: "own-teamless", Own: true, TeamlessActor: true},
		{Key: "teammate", SameTeam: true},
		{Key: "outsider"},
	}
}

// unreachableBy returns the invariant ids that forbid this (role, case)
// configuration, or nil when it is constructible.
func (p *Policy) unreachableBy(role string, c OwnerCase) []string {
	if !c.TeamlessActor {
		return nil
	}
	if p.Membership == "one" {
		return p.TeamInvs
	}
	for _, r := range p.RequiredFor {
		if r == role {
			return p.TeamInvs
		}
	}
	return nil
}

// evalScope evaluates a scope expression on a case.
func evalScope(e ScopeExpr, c OwnerCase) bool {
	if e.All {
		return true
	}
	if e.None {
		return false
	}
	return (e.Own && c.Own) || (e.Team && c.SameTeam)
}

// scopeFor returns the scope expression a rule gives to role ("" entries were
// expanded at load time, so every role either appears or has no grant).
func scopeFor(entries []ScopeEntry, role string) (ScopeExpr, bool) {
	for _, e := range entries {
		for _, r := range e.Roles {
			if r == role {
				return e.Expr, true
			}
		}
	}
	return ScopeExpr{}, false
}

// oracleRow is one decision case before rendering.
type oracleRow struct {
	verb, role, ownerCase, target string
	expectation                   string // allow | deny | unreachable
	invariants                    []string
}

// key is the stable-id input: the case, never the verdict.
func (r oracleRow) key() string {
	return fmt.Sprintf("%s|%s|%s|%s", r.verb, r.role, r.ownerCase, r.target)
}

// rows enumerates the complete decision table in deterministic order:
// verbs in vocabulary order, roles in enum order, cases in fixed order.
func (p *Policy) oracleRows() []oracleRow {
	var rows []oracleRow
	cases := p.ownerCases()

	// create: grants only, no object
	if p.grantsRule != nil {
		for _, role := range p.Roles {
			r := oracleRow{verb: "create", role: role, ownerCase: "-", target: "-"}
			if holds(p.rolesWith("create"), role) {
				r.expectation = "allow"
			} else {
				r.expectation = "deny"
			}
			r.invariants = p.grantsRule.Invariants
			rows = append(rows, r)
		}
	}

	// scoped verbs
	for _, verb := range []string{"read", "update", "delete"} {
		rule := p.scopeByVerb[verb]
		if rule == nil {
			continue
		}
		granted := p.rolesWith(verb)
		for _, role := range p.Roles {
			for _, c := range cases {
				r := oracleRow{verb: verb, role: role, ownerCase: c.Key, target: "-"}
				if forbidden := p.unreachableBy(role, c); forbidden != nil {
					r.expectation = "unreachable"
					r.invariants = forbidden
					rows = append(rows, r)
					continue
				}
				if p.grantsRule != nil && !holds(granted, role) {
					r.expectation = "deny"
					r.invariants = p.grantsRule.Invariants
					rows = append(rows, r)
					continue
				}
				expr, ok := scopeFor(rule.Scope, role)
				if ok && evalScope(expr, c) {
					r.expectation = "allow"
				} else {
					r.expectation = "deny"
				}
				r.invariants = rule.Invariants
				rows = append(rows, r)
			}
		}
	}

	// reassign: authority over the record AND the target rule
	if p.reassignRule != nil {
		rr := p.reassignRule
		targets := []struct {
			key      string
			sameTeam bool
		}{
			{"target-teammate", true},
			{"target-outsider", false},
		}
		for _, role := range p.Roles {
			expr, hasScope := scopeFor(rr.Reassign, role)
			for _, c := range cases {
				if forbidden := p.unreachableBy(role, c); forbidden != nil {
					rows = append(rows, oracleRow{verb: "reassign", role: role, ownerCase: c.Key, target: "-",
						expectation: "unreachable", invariants: forbidden})
					continue
				}
				authority := hasScope && evalScope(expr, c)
				if !authority {
					// no authority over the record: the target cannot matter
					rows = append(rows, oracleRow{verb: "reassign", role: role, ownerCase: c.Key, target: "-",
						expectation: "deny", invariants: rr.Invariants})
					continue
				}
				if p.TeamEntity == "" || rr.ReassignAny[role] {
					rows = append(rows, oracleRow{verb: "reassign", role: role, ownerCase: c.Key, target: "any",
						expectation: "allow", invariants: rr.Invariants})
					continue
				}
				for _, t := range targets {
					r := oracleRow{verb: "reassign", role: role, ownerCase: c.Key, target: t.key, invariants: rr.Invariants}
					if t.sameTeam {
						r.expectation = "allow"
					} else {
						r.expectation = "deny"
					}
					// a teammate target needs the actor teamed; a case that
					// leaves the actor teamless cannot produce one
					if t.sameTeam && c.TeamlessActor {
						continue
					}
					rows = append(rows, r)
				}
			}
		}
	}
	return rows
}

func holds(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// stableAuthzID hashes the case inputs; the 6-hex prefix is widened per
// colliding group exactly as the transition oracle does.
func stableAuthzID(key string) string {
	h := sha256.Sum256([]byte("AUTHZ|" + key))
	return hex.EncodeToString(h[:])
}

// GenerateOracle renders the authorization oracle markdown for a reconciled
// policy. Deterministic; returns the body and the row count.
func (p *Policy) GenerateOracle() (string, int) {
	rows := p.oracleRows()

	hashes := make([]string, len(rows))
	byPrefix := map[string]map[string]bool{}
	for i, r := range rows {
		hashes[i] = stableAuthzID(r.key())
		pfx := hashes[i][:6]
		if byPrefix[pfx] == nil {
			byPrefix[pfx] = map[string]bool{}
		}
		byPrefix[pfx][hashes[i]] = true
	}
	stable := func(i int) string {
		h := hashes[i]
		n := 6
		group := byPrefix[h[:6]]
		for n < len(h) {
			clash := false
			for other := range group {
				if other != h && other[:n] == h[:n] {
					clash = true
					break
				}
			}
			if !clash {
				break
			}
			n += 2
		}
		return "AUTHZ-" + h[:n]
	}

	var b strings.Builder
	w := func(format string, args ...interface{}) { fmt.Fprintf(&b, format+"\n", args...) }
	w("# Generated authorization oracle: policy")
	w("")
	w("Generated from `%s` + `%s` by `machinery alloy`. DO NOT EDIT BY HAND.", p.DomainFile, p.AnnotationFile)
	w("%s", version.MarkdownStamp())
	w("Single source of truth for the authorization tests: one row is one decision case")
	w("for the pure authz function. Key tests on the STABLE id, not the row number; row")
	w("numbers renumber when the design changes, stable ids do not. A design revision")
	w("flips an expectation under an unchanged stable id, so the test diff names exactly")
	w("which cases changed behavior.")
	w("")
	w("The scope algebra decides every case from two booleans (does the actor own the")
	w("record; do the actor and the owner share a team), so this table is the COMPLETE")
	w("semantics of the policy: two configurations that agree on the case columns are")
	w("policy-equivalent, and a correct implementation agrees with every reachable row.")
	w("")
	w("## Case vocabulary")
	w("")
	if p.TeamEntity != "" {
		w("Owner cases (relationship between the actor and the record owner):")
		w("")
		w("- `own-teamed`: the actor owns the record and holds a team.")
		w("- `own-teamless`: the actor owns the record and holds no team.")
		w("- `teammate`: the record is owned by another member of the actor's team.")
		w("- `outsider`: the owner is neither the actor nor a teammate. Concrete variants")
		w("  (all policy-equivalent, test every one): owner in a different team; owner")
		w("  teamless; actor teamless with any other owner.")
		w("")
		w("Target cases (reassign only; where the record would go):")
		w("")
		w("- `target-teammate`: the new owner is a member of the actor's team (the actor")
		w("  itself, when teamed, is its own teammate).")
		w("- `target-outsider`: the new owner is not a member of the actor's team.")
		w("- `any`: the rule places no constraint on the target for this role.")
	} else {
		w("Owner cases (this design has no team scoping):")
		w("")
		w("- `own`: the actor owns the record.")
		w("- `other`: the record is owned by someone else.")
	}
	w("")
	w("Rows marked `unreachable` are configurations the named invariants forbid; the")
	w("write discipline refuses to construct them, and authorization behavior on them")
	w("is unspecified. Tests skip them (or assert the construction is refused).")
	w("")
	w("## Decisions")
	w("")
	w("| test id | stable id | verb | role | owner case | target | expectation | invariants |")
	w("|---|---|---|---|---|---|---|---|")
	for i, r := range rows {
		w("| O-AUTHZ-%02d | %s | %s | %s | %s | %s | %s | %s |",
			i+1, stable(i), r.verb, r.role, r.ownerCase, r.target, r.expectation, strings.Join(r.invariants, ", "))
	}
	return b.String(), len(rows)
}
