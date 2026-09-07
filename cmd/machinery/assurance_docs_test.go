package main

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// MAC-gcrr documentation contract. These tests pin the truthful consumer
// assurance guidance: what machinery's checks actually establish, what green
// means, and what it does not. They fail when a shipped document claims more
// than the accepted implementation delivers (defaults-only bootstrap, warning-
// only plugin failures, universal-correctness claims), and when a required
// truthful statement is missing. They are string/section contracts against the
// shipped files only; the real registry/bind-path flow lives in the tagged
// integration suite registered in testdata/integration-lanes/consumer-docs.json.

func assuranceDoc(t *testing.T, rel string) string {
	t.Helper()
	return mustRepositoryFile(t, filepath.Join(repoRootDir(t), rel))
}

// assuranceNormalize collapses markdown line wrapping so a phrase contract
// binds to the sentence, not to one wrapping of it.
func assuranceNormalize(doc string) string {
	return strings.Join(strings.Fields(doc), " ")
}

func requirePhrases(t *testing.T, doc, name string, phrases []string) {
	t.Helper()
	normalized := assuranceNormalize(doc)
	for _, phrase := range phrases {
		if !strings.Contains(normalized, assuranceNormalize(phrase)) {
			t.Errorf("%s is missing the truthful guidance %q", name, phrase)
		}
	}
}

func forbidPhrases(t *testing.T, doc, name string, phrases []string) {
	t.Helper()
	normalized := assuranceNormalize(doc)
	for _, phrase := range phrases {
		if strings.Contains(normalized, assuranceNormalize(phrase)) {
			t.Errorf("%s still carries the false claim %q", name, phrase)
		}
	}
}

// TestAssuranceDocsInstallerBootstrapReceiptParity pins the corrected
// installer/bootstrap contract from the receipt-aware convergence work: a
// valid supported receipt plans the complete recorded update, not the default
// homes; a no-receipt bootstrap keeps plugin-aware defaults; explicit selectors
// never combine with bootstrap. Both shipped files must reject the old
// defaults-only / not-from-receipt / native-only-via-update claims.
func TestAssuranceDocsInstallerBootstrapReceiptParity(t *testing.T) {
	readme := assuranceDoc(t, "README.md")
	portability := assuranceDoc(t, "docs/agent-portability.md")

	forbidPhrases(t, readme, "README.md", []string{
		"updates the binary and the default home group",
	})
	forbidPhrases(t, portability, "docs/agent-portability.md", []string{
		"updates the binary and the default home group only",
		"plans from the default homes, not from the receipt",
	})

	requirePhrases(t, readme, "README.md", []string{
		"uses the complete recorded home, native-target, and host-plugin plan",
		"plugin-aware default homes",
	})
	requirePhrases(t, portability, "docs/agent-portability.md", []string{
		"uses the complete recorded home, native-target, and host-plugin plan",
		"plugin-aware default homes",
		"cannot be combined with `--bootstrap-defaults`",
		"preserving each recorded group's copy/symlink mode",
	})
}

// TestAssuranceDocsSafeRepairAndFailClosedReceipts pins the repair and
// fail-closed boundary: safely edited/missing owned regular content with
// unchanged safe parents/topology is repairable; corrupt/unsupported/unsafe
// receipts, unsafe substitutions, and uncertain plugin ownership fail closed
// with diagnostics and no silent defaults fallback — including inspection
// under --skip-plugins. Docs must not imply arbitrary missing-root recreation.
func TestAssuranceDocsSafeRepairAndFailClosedReceipts(t *testing.T) {
	portability := assuranceDoc(t, "docs/agent-portability.md")

	requirePhrases(t, portability, "docs/agent-portability.md", []string{
		"safely edited or missing owned regular content",
		"repaired to exact current-release content",
		"unchanged safe parents",
		"fail closed with a diagnostic naming the problem",
		"no silent fallback to the defaults",
		"receipt and ownership inspection still run under `--skip-plugins`",
	})
	forbidPhrases(t, portability, "docs/agent-portability.md", []string{
		"recreate arbitrary missing roots",
		"recreates arbitrary missing roots",
	})
}

// TestAssuranceDocsPluginRefreshFailureIsReportedNotWarned pins the corrected
// post-direct-commit host plugin failure semantics: host CLI absence, failed
// inventory, and failed refreshes are returned failures, not warnings over a
// successful update; direct changes may already be committed; host caches are
// never rolled back by machinery; the retry obligation is recorded. The docs
// must also distinguish this from pre-plan discovery failure and from the
// explicit --skip-plugins opt-out.
func TestAssuranceDocsPluginRefreshFailureIsReportedNotWarned(t *testing.T) {
	readme := assuranceDoc(t, "README.md")
	portability := assuranceDoc(t, "docs/agent-portability.md")

	forbidPhrases(t, readme, "README.md", []string{
		"is reported as a warning while direct skills and adapters still update",
		"managed-scope refusal is reported as a warning",
	})
	forbidPhrases(t, portability, "docs/agent-portability.md", []string{
		"is reported as a warning while direct skills and adapters still update",
	})

	requirePhrases(t, readme, "README.md", []string{
		"is a returned failure, not a warning over a successful update",
		"cannot roll a host plugin cache back",
		"distinct from plugin discovery",
	})
	requirePhrases(t, portability, "docs/agent-portability.md", []string{
		"a returned failure whose error names the exact retry",
		"never rolled back by machinery",
	})
}

// TestAssuranceDocsUpdateTransactionLimits pins the precise transactional
// restoration scope: binary/direct/receipt roots restore to exact pre-run
// state before the direct commit (absence restored as absence), concurrent
// foreign changes are refused, delegated children publish no intermediate
// receipts, and the retained journal recovers an interrupted run.
func TestAssuranceDocsUpdateTransactionLimits(t *testing.T) {
	portability := assuranceDoc(t, "docs/agent-portability.md")

	requirePhrases(t, portability, "docs/agent-portability.md", []string{
		"restores every root it owns to its complete pre-run state",
		"restoring a previously absent artifact to absence",
		"refused rather than overwritten",
		"publish no intermediate receipts",
		"interrupted journal recovery",
	})
}

// TestAssuranceDocsRecoverCommandDocumented pins the honest description of
// machinery recover for interrupted design publication: a read-only report by
// default, with --apply completing only a fully revalidated publication.
func TestAssuranceDocsRecoverCommandDocumented(t *testing.T) {
	readme := assuranceDoc(t, "README.md")
	portability := assuranceDoc(t, "docs/agent-portability.md")

	if !strings.Contains(readme, "`recover`") {
		t.Error("README.md command inventory must list the recover command")
	}
	requirePhrases(t, portability, "docs/agent-portability.md", []string{
		"`machinery recover <design-dir>`",
		"completes a recovery only after full revalidation",
	})
}

// TestAssuranceDocsOpenCodeAdapterBounds pins the OpenCode governance plugin
// bounds: every governance subprocess is deadline- and output-bounded, and
// hook responses are validated against the documented protocols with
// unrecognized, truncated, or combined-protocol output blocking.
func TestAssuranceDocsOpenCodeAdapterBounds(t *testing.T) {
	portability := assuranceDoc(t, "docs/agent-portability.md")

	requirePhrases(t, portability, "docs/agent-portability.md", []string{
		"a deadline plus a capped output capture",
		"unrecognized, truncated, or combined-protocol responses block",
	})
}

// TestAssuranceDocsExternalCheckerRegistryRelativeInputs pins the accurate
// registry-input resolution story: sources resolve relative to the registry
// file with `..` rejected, and the documented repo-root workaround (registry
// beside the committed inputs plus --registry) removes the copy without
// changing the closure. The broader extension stays explicitly unimplemented.
func TestAssuranceDocsExternalCheckerRegistryRelativeInputs(t *testing.T) {
	docs := assuranceDoc(t, "docs/external-checkers.md")

	requirePhrases(t, docs, "docs/external-checkers.md", []string{
		"resolves against the directory containing the registry file",
		"`..` segments are rejected",
		"place the registry beside them",
		"that extension is not implemented today",
	})
}

// TestAssuranceDocsContainerizedCIDaemonVisibility pins the daemon-visible
// bind-path guidance: the requirement is daemon-visible matching paths, dind
// is one option rather than the only topology, the concrete daemon error is
// named, and ambient engine availability is never assumed.
func TestAssuranceDocsContainerizedCIDaemonVisibility(t *testing.T) {
	docs := assuranceDoc(t, "docs/external-checkers.md")

	requirePhrases(t, docs, "docs/external-checkers.md", []string{
		"daemon-visible matching paths",
		"one option, not the only one",
		"invalid mount config",
	})
	forbidPhrases(t, docs, "docs/external-checkers.md", []string{
		"a host socket cannot work",
	})
}

// TestAssuranceDocsPlatformEmulationTruth pins the same-platform emulation
// statement: the declared platform is always used, emulated reproduction of
// the pinned userspace is supported, and it is never presented as native-host
// test evidence.
func TestAssuranceDocsPlatformEmulationTruth(t *testing.T) {
	docs := assuranceDoc(t, "docs/external-checkers.md")

	requirePhrases(t, docs, "docs/external-checkers.md", []string{
		"emulated reproduction of the checker run",
		"not native-host test evidence",
	})
}

// TestAssuranceDocsCheckerContainerBudgets pins the closed checker-container
// budgets and lifetime ownership: finite resource budgets, per-stream output
// bounds whose breach aborts and cleans up, registered container identity,
// and force-removal on every exit path.
func TestAssuranceDocsCheckerContainerBudgets(t *testing.T) {
	docs := assuranceDoc(t, "docs/external-checkers.md")

	requirePhrases(t, docs, "docs/external-checkers.md", []string{
		"128 MiB memory",
		"32-process",
		"128 KiB per stream",
		"force-removed",
		"leaves no owned container behind",
	})
}

// TestAssuranceDocsSeparatedAssuranceClaims pins the four-claims separation
// (artifact consistency, proof execution, test execution, current review), the
// discovery-only Gt oracle-coverage label, the attestation kinds, and the
// named residual runtime obligations that remain owned tests rather than
// proven properties. Text naming an obligation must not claim enforcement.
func TestAssuranceDocsSeparatedAssuranceClaims(t *testing.T) {
	readme := assuranceDoc(t, "README.md")

	requirePhrases(t, readme, "README.md", []string{
		"artifact consistency",
		"proof execution",
		"test execution",
		"current review",
		"static discovery",
		"tests were not executed",
		"`plan`, `current`, or `historical`",
		"replay",
		"races",
		"migration",
		"restore",
		"load",
		"enforces nothing by itself",
	})
}

// TestAssuranceDocsReleaseNoteDiscipline pins the release-note discipline
// document: the historic v0.6.3 generator omission, explicit callouts for
// generated-output changes, proof-scope changes never silently called
// equivalent, compatibility/migration sections, and the exact-commit release
// publication enforcement described truthfully as not yet applied.
func TestAssuranceDocsReleaseNoteDiscipline(t *testing.T) {
	docs := assuranceDoc(t, "docs/release-notes.md")

	requirePhrases(t, docs, "docs/release-notes.md", []string{
		"v0.6.3",
		"Live_OverlayResolves",
		"never silently called equivalent",
		"Compatibility and migration",
		"exact-commit",
		"not claimed as remotely enforced",
	})
}

// TestAssuranceDocsChangelogCoversAcceptedHardening pins the changelog's
// Unreleased section to the accepted hardening work, in user-facing terms.
func TestAssuranceDocsChangelogCoversAcceptedHardening(t *testing.T) {
	changelog := assuranceDoc(t, "CHANGELOG.md")

	requirePhrases(t, changelog, "CHANGELOG.md", []string{
		"[Unreleased]",
		"converge on the recorded installation",
		"`machinery recover`",
		"static discovery",
		"hostile host",
		"Required integration lane",
		"128 MiB",
	})
}

// TestAssuranceDocsStandaloneMachineryOnly keeps the consumer-facing
// documentation standalone: no tracker, delivery-process, or story identity
// may leak into the shipped guidance.
func TestAssuranceDocsStandaloneMachineryOnly(t *testing.T) {
	pattern := regexp.MustCompile(`Paivot|pvg\b|MAC-[0-9a-z]{4}`)
	for _, rel := range []string{
		"README.md",
		"docs/agent-portability.md",
		"docs/external-checkers.md",
		"docs/release-notes.md",
		"CHANGELOG.md",
	} {
		doc := assuranceDoc(t, rel)
		if loc := pattern.FindString(doc); loc != "" {
			t.Errorf("%s mentions delivery-process identity %q; product docs must require Machinery only", rel, loc)
		}
	}
}

// TestAssuranceDocsPreservedTruths guards statements that were true before
// this contract and must survive it (passing controls on unmodified text).
func TestAssuranceDocsPreservedTruths(t *testing.T) {
	readme := assuranceDoc(t, "README.md")
	portability := assuranceDoc(t, "docs/agent-portability.md")
	docs := assuranceDoc(t, "docs/external-checkers.md")

	requirePhrases(t, readme, "README.md", []string{
		"It does not skip work when the requested version matches the installed version",
	})
	requirePhrases(t, portability, "docs/agent-portability.md", []string{
		"Missing subagent support falls back to executing the same role inline",
	})
	requirePhrases(t, docs, "docs/external-checkers.md", []string{
		"Tags are rejected",
	})
}
