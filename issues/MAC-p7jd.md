---
id: MAC-p7jd
title: "Invalidate reviews when implementation subjects change"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd, red-approved, delivered]
parent: MAC-ui8a
created_at: 2026-09-05T19:33:46Z
created_by: ramirosalas
updated_at: 2026-09-06T07:38:06Z
content_hash: "sha256:8c3c827bf6ce2c4cc035335edf8e4f1b3ea3d200f809917828f6fe8d7e0ed367"
blocks: [MAC-vx24, MAC-gcrr, MAC-ou97, MAC-hgz1]
follows: [MAC-p8ce, MAC-2u36, MAC-a89e, MAC-olrx]
assignee: dev-MAC-p7jd
---

## Description
## USER INTENT
Strengthen Machinery mission-critical assurance with observable fail-closed behavior and precise limits.

## Context (Embedded)
Assessment F3: gt.conformance-test-shape covers BUILD artifacts, while g4.pack-event-discipline covers pack. Neither binds tested/reviewed implementation. Historic Ga ancestor acceptance is valid history, not current-tree assurance. Preserve judgment-vs-mechanical distinction.

## Ownership
The original13 required files plus the already used optional fourteenth GREEN supplemental file remain the implementation boundary. Two additional existing test files are CONDITIONALLY scoped below solely for the exact independently reviewed fixture amendment; this bookkeeping is not TEST-EDIT AUTHORIZED. The following original13 required files are the reviewed ownership boundary; implementation and test editing remain subject to their independent phase authorizations. You are not alone: preserve other edits, especially accepted MAC-p8ce/MAC-olrx behavior; coordinate shared paths. No generic helper factoring is authorized. Optional internal/gates/attest_green_test.go is the reported fourteenth file only for independent supplemental GREEN proof.

## Boundary Map
PRODUCES:
- internal/gates/attest.go -> AttestationReview; CheckAttestationsWithImplementation(design, impl string) *Gate; RenderAttestation(design, impl string, review AttestationReview) ([]byte, error); private checker/render/pending-result contract below
- internal/gates/attest_implementation_test.go -> NEW frozen existing-interface staged A/B/C/D and real filesystem/Git regression proof
- cmd/machinery/attest.go -> newAttestCmd() *cobra.Command generation flags/output; preserve stableAttestationHashes(paths []string) ([]string, error)
- docs/attestation-evidence.md -> closed v2 schema, migration, scope/digest grammar and honest limits
- internal/gates/suite.go -> Snapshot capture/RunSelected/CheckUnchanged/Release lifecycle and explicit-release wrappers, only attestation integration
- cmd/machinery/check.go -> Gv-facing help/messages only
- internal/gates/attest_test.go -> only independently PM-authorized exact wrong-version amendment and v2 cases
- cmd/machinery/attest_test.go -> only independently PM-authorized exact mandatory alias-proof amendment
- cmd/machinery/attest_implementation_test.go -> NEW frozen real isolated CLI/local-Git A/B/C/D proof
- internal/designlock/attestation_snapshot.go -> NEW AttestationTreeSnapshot, AttestationTreeEntry and (*Lock).MaterializeAttestationTree(path string) (*AttestationTreeSnapshot, error); exact APIs below
- internal/designlock/attestation_snapshot_test.go -> NEW real held-root/filesystem/custody proof; tests referencing new Go symbols are GREEN supplemental, not baseline RED
- internal/hook/hook.go -> stop snapshot finalization before decision/state-clear ordering only
- internal/hook/attestation_snapshot_test.go -> NEW real Stop/SubagentStop configured ledger/custody proof
- internal/gates/attest_green_test.go -> OPTIONAL new-symbol GREEN supplemental tests only, report purpose and cost; never edit frozen RED
- internal/hook/hook_test.go -> CONDITIONAL only func copyTree(t *testing.T, src, dst string), exact36-line additive temporary-fixture v1-to-v2 plan/history construction proposed against51454e0, patch ae6e9241ea5d7b0c4556b2ee02c6e2910b2663c2452761e7984f1c6663252971; editable only after separate explicit independent PM exact-text authorization. Preserve all original helper/caller/assertion bytes outside that exact hunk.
- internal/gates/obligation_ownership_test.go -> CONDITIONAL only func obligationParentFixture(t *testing.T) (string,string), exact29-line additive parent temporary-fixture construction in the same patch, under the same separate independent PM authorization. No assertion/Ga/selector/current-credit relaxation.
CONSUMES:
- Existing Machinery implementation at epic a82277a; preserve compatibility and generic snapshot semantics.
  spec: CheckAttestations(design string) *Gate; attestationRequiredPaths(g *Gate, design, claim string) []string; stableAttestationHashes(paths []string) ([]string, error); (*Snapshot).RunSelected(impl string, sel Selection, opt RunOptions) []*Gate; SelectRunAndNote(design, impl, gateList string, opt RunOptions) (Selection, []*Gate, string, error); (*designlock.Lock).MaterializeExternalTree(path string) (*ExternalTreeSnapshot, error) remains generic-only, not strict-subject authority.
- Existing internal/designlock private snapshot helpers/state, consumed without production edits to their files.
  source: snapshotBudget/readSnapshotDir/copySnapshotFile/sameFingerprintFile/validateInventoryPath/newPrivateSnapshot; exact approved new-capability use and held-root protocol embedded below.

### Story Acceptance Criteria
1. Implementation/test behavior claims bind a complete explicit implementation/test scope under rooted inventory and content hashes, not only BUILD or pack. Any code/test/config addition, removal, rename, content change or scope narrowing affecting the claim invalidates freshness.
2. Distinguish plan-only claims, current implementation review and historical milestone acceptance in closed schema, gate diagnostics and docs. Historical ancestor records remain historical; they cannot alone imply current implementation approval.
3. Provide explicit compatibility migration for existing attestations. Legacy design-only covers never quietly grandfather implementation assertions as fresh; users receive actionable missing-subject diagnostics.
4. Negative tests remove assertions after review, alter event handlers, add excluded files, change scope, alias paths/symlinks and replay stale attestations. Positive unchanged reviewed scope and harmless evidence-only commit remain usable.
5. Real CLI attest/check path exercises changed implementation and historical/current distinction. Hashing proves binding, not reviewer honesty or that tests executed; output never claims otherwise.

## Testing Requirements
- Explicit hard TDD: separate RED test author, expected behavioral assertion failure plus passing control on unmodified production, independent PM replay, frozen RED test/fixture bytes retained through GREEN. Compilation, import, timeout or unavailable infrastructure are not valid RED. Existing tests encoding the unsafe contract may be corrected during reviewed RED; GREEN cannot silently weaken them.
- Integration tests: MANDATORY (no mocks). Exercise real process/filesystem/runtime boundaries; no stubs, no skip-if-missing. Fixture policy inputs may test parser logic but are not live remote proof.
- go test ./internal/gates -run Attest; go test ./cmd/machinery -run Attest; CLI integration with real temp git repository and design+implementation roots.
- Full scripts/preflight.sh ONLY at end of epic. No pushes, sync, remote mutation, installed binary/plugin/agent/skill replacement, or dev-link. Another agent uses installed Machinery in NIL; builds/tests use isolated output and homes.
- Shipped product enforcement is standalone Machinery: never requires pvg, nd, Paivot labels/metadata or commit conventions. Paivot is local development coordination only.

## OUT OF SCOPE
- Other assessment areas have sibling stories; final preflight/local main merge/isolated candidate binary are final gate responsibilities. Do not omit small directly related safety fixes.

## DIFF BUDGET
- Measured paused51454e069ebe4039f02d6d9108acf9354c7ad6c8 remains14 paths,3331 additions+66 deletions=3397 changedLOC. Exact UNAPPLIED fixture proposal ae6e9241ea5d7b0c4556b2ee02c6e2910b2663c2452761e7984f1c6663252971 adds65 lines/0deletions across2existingtest helpers (copyTree36,obligationParentFixture29). IF separately independently PM-authorized and applied unchanged, aggregate becomes16 paths,3396 additions+66 deletions=3462 changedLOC. Conditional scope/cost is not application, exact-test authorization, same-SHA replay or delivery proof. Existing 2710-3310 and older estimates remain historical cost forecasts; no proof trimming or broader helper/test/framework scope.

## Conditional fixture consumer handoff
Exact source proposal /tmp/machinery-p7jd-fixture-proposal.geNVdA/PROPOSAL-INDEX.md SHA2561049ea7adf1eaa142948f2f6cd3d4d8327ae20a9b3326222f57041f7f19c9169 documents only the two named helpers and full13-caller external replay. Its shared application remains HELD until the independent PM issues exact TEST-EDIT AUTHORIZED disposition. The five AC and no-grandfathering remain unchanged. No existing acceptance record/date/attestor/cover hash is refreshed; GoCRM becomes12plan+1historical and parent8plan only in copied test inputs, with explicit missing-current warning/current0. Actual original silence/ledger, Ga ancestry, default-versus-explicit Gt and intended positive/negative assertions stay exact.
Once this story is accepted, MAC-hgz1 consumes the exact accepted helper bytes/SHA, accepted substantive review boundary, full input/acceptance inventory and all13-caller same-SHA blast evidence. Its later source-example migration WILL invalidate these exact v1 guards; hgz1 now owns only the later independently exact-before-edit reviewed adaptation inside these same two helpers. That later v2 construction is not selected or authorized here and cannot silently downgrade a shipped current review into apparent new evidence. p7 never depends on hgz1/uzxr to establish this temporary-fixture repair; no reverse dependency, new story or criterion is introduced.

## Approved architecture authority and executable clarifications
Independent contract review APPROVED revision 2. Authority: proposal SHA256 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179, read in full and embedded below; rejected revision 1 remains historical only. Architecture revision 2 is independently approved, and the subsequent PRE-RED PM EXACT TEST-EDIT / SEAM AUTHORIZATION has authorized bounded RED authoring for its exact two existing-test amendments and four seam uses. This is not approve-red, GREEN dispatch authorization, delivery or executed AC proof. Earlier architecture-pending/authoring-pending statements remain history, not the current phase disposition. Dispatcher may resume the healthy retained RED author within that existing authorization; independent RED review is still required.

Three reviewed executable clarifications apply to the embedded contract:
- Repeated Snapshot.Release returns the latched final disposition without revalidation, re-closing, re-finalization or resurrecting invalid current results. Capture and Snapshot.RunSelected after release fail closed; repeated RunSelected calls before release are supported and covered, retaining all captures/pending results until that single finalization.
- Every late Release/finalization cause and Gate error is remapped to logical caller-facing paths before exposure; no private snapshot path leaks. Add matched success/error tests through convenience wrappers and direct snapshot use.
- Every DISPLAYED current-review report includes the exact scope-boundary limits stated below, including displayed CLI/hook reports. Silent successful hooks may remain silent; this is not a new reporting feature.

No execution authentication, reviewer-honesty claim, MAC-l7m0 dependency, Docker requirement, or new user choice is introduced. Generic designlock.go, external_snapshot.go, source_snapshot.go, snapshot_inventory.go, scale.go, portablepath, cmd/machinery/hook.go, generic snapshot callers, accept.go and accept_test.go remain read-only. New API declarations below are approved PRODUCES, not preexisting callable test seams.

## Current independent PM authoring authorization
Independent PM has explicitly authorized only: (1) internal/gates/attest_test.go TestAttestationMutations wrong-version integer 2 -> 3 and exact diagnostic "attestation_version must be the integer 1 or 2", with separate new accepted-v2 plan and malformed-v2 cases using real existing fixtures/CheckAttestations; (2) cmd/machinery/attest_test.go TestAttestRejectsIdentityAliases os.Link setup failure t.Skipf("hard links unavailable: %v", err) -> t.Fatalf("create required hard-link alias fixture: %v", err). The existing mandatory alias case was chosen, not an alternative new case. Retain its real os.Link/CLI, exit 1, empty stdout and identity-alias assertions. All other existing cases/helpers/tests remain unchanged absent a new named review. The complete PRE-RED PM EXACT TEST-EDIT / SEAM AUTHORIZATION preserved in this story governs exact restrictions; this clarification issues no additional test authorization.
New frozen RED tests are confined to the named attestation implementation files, new hook/designlock test files where they can compile using existing APIs/seams, and the exact authorized amendments. New helper API unit tests that cannot compile before production belong to explicitly reported GREEN supplemental proof; they cannot replace frozen existing CLI/suite/hook acceptance tests. A/B/C/D classification below is mandatory.
There is NO blanket permission to add fault/callback/budget seams. Independent PM has approved exactly four bounded uses, with all original restrictions retained in this story: existing Snapshot APIs for frozen lifecycle tests; hook writer interface beforeAttestationFinalization(*gates.Snapshot, []*gates.Gate); suite-local attestationBeforeFinalRelease func(*Snapshot) only for first-Release GREEN supplemental real mutation/owned-copy cleanup tests; and new designlock capability tests using existing testAfterSnapshotCopyReadChunk plus the exact newAttestationSnapshotBudget factory with fixed production defaults and reviewed lowered test budgets. Preserve actual-operation, fired/no-fault control, no direct verdict mutation, helper-process TMPDIR isolation and cleanup restrictions. Any other seam/use needs its own exact independent PM review. Stage A existing-interface behavioral failure plus Stage D passing controls is the required baseline RED; B interface absence and C not-yet-reached mutations must be recorded honestly. Do not call unavailable Go symbols from frozen baseline tests.
Focused verification after scope/test freeze: go test ./internal/gates -run Attest; go test ./cmd/machinery -run Attest; go test ./internal/designlock -run Attestation; go test ./internal/hook -run Attestation. Record all exact discovered leaf names, baseline/final outcomes and reached/not-reached mutations; choose bounded timeouts and record runtime before any broader replay. No full preflight until epic final gate.

## Approved schema and classification

Write attestation_version: 2 for new rows. Continue to parse closed v1 for explicit migration semantics below. Reject every other version, including numeric spellings other than integer 1 or 2. Root allowed keys remain attestation_version, attestations, optional string _comment. attestations is a nonempty array. YAML duplicate keys at every depth, unknown keys, wrong node types, nulls, aliases/merge keys, and duplicate rows are errors. All optional note/_comment fields are strings, not arbitrary YAML payloads.

V2 row allowed keys: claim, kind, attestor, date, covers, implementation, note, _comment. claim/kind/attestor/date are nonempty strings; date is an actual YYYY-MM-DD date. covers is a nonempty array of closed {path, hash, optional string _comment}. cover path is a canonical portable design-relative regular-file path, never absolute, '.', '..', backslash, traversal, or a normalized alias; use portablepath.ValidateRelative. No duplicate/casefold-equivalent cover paths. Hash is exactly sha256: followed by 64 lowercase hex digits. covers may not name attestations.yaml (self-reference). Existing required design-subject coverage remains mandatory, including all BUILD packets, pack files and acceptance files relevant to the claim.

kind is exactly plan, current, or historical. There remains at most one row per claim ID, including across kinds: upgrading a plan to current replaces that claim's row; git history retains previous reviews.

Claim classification is an explicit table in code, not a prefix-derived runtime guess:

* Design-plan judgments: the six g2 IDs; the four g3 IDs; g4.zero-context. Allowed kind: plan only. Their words and diagnostics describe the design/model/BUILD plan, not observed implementation behavior. They keep their existing required design subjects and owed predicates.
* Implementation/test judgments: gt.conformance-test-shape, g4.pack-event-discipline, g4.standin-coverage. Allowed kind: current, or plan to record intent only. current requires the complete implementation scope below AND existing required design covers. plan never counts as current implementation review. Stand-in coverage is classified here because its judgment includes actual stand-ins held to an oracle and an environment recipe; keeping only the BUILD paragraph would repeat the same defect.
* ga.review-quality: historical only. It judges the milestone acceptance record(s), with covers over the required acceptance/*.yaml files. It has no implementation field. Ga's existing acceptance schema is unchanged; its commit fields remain the history anchors.

implementation is REQUIRED exactly when kind=current, and FORBIDDEN otherwise. Closed object keys: root, policy, entries, hash. root is the normalized slash-separated lexical relative path from the logical design root to the supplied implementation root ('.' or leading '..' components are permitted only in this locator, since sibling/parent implementation roots are normal). It must exactly equal the locator independently derived from the actual invocation; no absolute paths, backslashes, redundant components, or nonportable non-dot components. This locator grants no path-opening authority. All reads come from the invocation's held root capability. policy must equal full-root-v1. There are no user include, exclude, extension, gitignore or subtree selectors. entries is a nonempty sorted inventory (including root directory '.'); hash binds the canonical serialization below.

Every inventory entry is a closed mapping. Directory: {path, type: directory, mode}; regular file: {path, type: file, mode, size, hash}. mode is a four-character octal permission string 0000–0777; size is a nonnegative integer within limits; hash has the same fixed SHA256 spelling. File-only keys on directories are errors. '.' is allowed only for the single root directory; all other paths use portablepath.ValidateRelative. Sort by raw ASCII path bytes; reject unsorted rows, duplicates, casefold collisions, missing parents, file/descendant conflicts and wrong root shape. Do not silently normalize a submitted manifest.

Concrete schema/digest example: ARCHITECTURE.md bytes are "# Architecture\n", BUILD.md bytes are "# Build\n", both implementation files are distinct regular files with bytes "package example\n" (16 bytes), implementation root mode 0755 and file modes 0644. acceptance/M1.yaml bytes here are "milestone: 1\n" solely to illustrate Gv's historical cover hashing: that fragment is NOT valid Ga acceptance and MUST NOT be presented as a valid milestone acceptance fixture. The current scope digest below was calculated from the exact canonical grammar; this example is not a --complete fixture and does not prove a test contains assertions. Use the existing complete Ga record shape for the real Git history tests.

```yaml
attestation_version: 2
attestations:
  - claim: g2.nfr-content
    kind: plan
    attestor: R
    date: '2026-09-05'
    covers:
      - {path: ARCHITECTURE.md, hash: 'sha256:2e7cb38229f7877e569cc05dd9c8fb71b30954ad2c9978328d69f5d377f81505'}
  - claim: gt.conformance-test-shape
    kind: current
    attestor: R
    date: '2026-09-05'
    covers:
      - {path: BUILD.md, hash: 'sha256:bbc290c9f84e532bd47737480381f0db3afae637d696806856d95d0a186bb619'}
    implementation:
      root: ../src
      policy: full-root-v1
      entries:
        - {path: '.', type: directory, mode: '0755'}
        - {path: handler.go, type: file, mode: '0644', size: 16, hash: 'sha256:e0e0431b63a883552b05817d33ba13f79262019cdefb4e5ae060c299c1de1eb8'}
        - {path: handler_test.go, type: file, mode: '0644', size: 16, hash: 'sha256:e0e0431b63a883552b05817d33ba13f79262019cdefb4e5ae060c299c1de1eb8'}
      hash: 'sha256:4e0ca743a34c585082fdcfb7f57ca6a47a1e3448510c3033f5dd7281a351ac2a'
  - claim: ga.review-quality
    kind: historical
    attestor: R
    date: '2026-09-05'
    covers:
      - {path: acceptance/M1.yaml, hash: 'sha256:a44677835fe6e178bd0d0d51faea9ccfc327ec4442bf96a4a96031392c7da249'}
```

Invalid examples/categories: version: '2' (wrong type); version 3 (unsupported); kind: accepted (unknown); current without implementation (GV_MISSING_IMPLEMENTATION_SUBJECT); kind: historical on gt.conformance-test-shape (GV_KIND); implementation.exclude: ['generated/**'] (GV_SCHEMA); root ../src changed to ../src/tests (GV_SCOPE_ROOT or GV_SCOPE_HASH); duplicated handler.go or ./handler.go (GV_SCOPE_PATH); omitted handler_test.go while retaining the old scope hash (GV_SCOPE_HASH), or after recomputing the manifest hash (GV_SCOPE_INVENTORY against independently enumerated root). A syntactically sound current row with changed bytes is GV_STALE_CONTENT, not schema failure or DRIFT.

## Complete scope and deterministic freshness

The authority is the entire actual --impl root. Enumerate every directory and regular file recursively, including ignored, untracked, hidden, generated, extensionless, binary, vendor, build-output, test, and configuration files. G4 contract ignores, .gitignore, .machinery configuration and programming-language extension lists NEVER determine this inventory. They are themselves ordinary included files. Empty directories and permission changes are bound conservatively. This can be larger than the language scanner's scope; that is intentional. Dependencies/configuration outside the supplied root are outside this explicitly reported review, and Machinery must not call this whole-program or environment completeness. Users select a root containing the code/test/config they want reviewed; switching root changes the bound locator.

Exactly two reserved exclusions exist, determined by policy and invocation, never by the row:

1. The supplied implementation root's top-level .git metadata entry, if present. It must be a real directory or regular worktree gitfile, never a symlink/special file. Its content/presence is excluded, allowing ordinary Git commits. All nested .git entries cause GV_SCOPE_UNSUPPORTED_METADATA, instead of being silently pruned. This is a deliberately conservative policy: nested-repository users must provide a source export without nested metadata. The existing generic snapshot skips .git at every depth and is insufficient for this particular rule. Top-level VCS administration, hooks and repository-local Git configuration are NOT implementation/test execution inputs covered by this review; using them as runtime source is outside this reported scope. There is no language-independent proof that arbitrary code does not read VCS metadata. Documentation and output must state this exact boundary, not claim excluded metadata can never affect behavior.
2. The exact logical <design>/attestations.yaml path, only when it lies within the implementation root. This is Machinery's reserved review record, not arbitrary *.yaml, an evidence directory, or a glob. Omit that entry's presence/content/mode from the inventory, with a constant exclusion descriptor included in the digest whether the file exists yet or not. Gv separately parses the present file under the closed schema. All sibling files, nested files named attestations.yaml elsewhere, acceptance YAML, logs, docs, generated tests and configuration remain included. Changing a note/attestor/date in this one record and committing it is the matched harmless evidence-only positive. This proposal does NOT promise freshness survives edits to arbitrary prose/docs or adding acceptance records; they remain conservatively bound when inside --impl.

Exclusions are reserved semantic boundaries, not authenticated assurances about how the application executes. If the user intends application code to consume Machinery evidence or Git metadata as runtime inputs, full-root-v1 does not cover that runtime and must not be described as doing so. This precise limit is preferable to claiming that hashing can discover all transitive execution dependencies. No further exclusion expansion is implied.

Before hashing compare the submitted inventory to a fresh authoritative inventory in BOTH directions. Any added, removed or renamed path is GV_SCOPE_INVENTORY; changed type/mode/size/hash is GV_STALE_CONTENT (type aliases/symlinks instead use custody categories below). Report the first differing portable path deterministically plus summary counts; do not print private temp paths or raw file content. Every current row is checked against the same fresh inventory for the invocation. Required design covers are independently checked too, so unrelated unchanged files cannot discharge a claim.

Canonical digest input is UTF-8/ASCII, exactly this header and tab/newline grammar, no YAML formatting and no timestamps/commit/inode numbers:

```
machinery-attestation-scope-v1\n
root\t<logical-root-locator>\n
policy\tfull-root-v1\n
exclude\tvcs-root:.git\n
exclude\tevidence:<implementation-relative-design-attestations-path-or-none>\n
directory\t<path>\t<mode>\n
file\t<path>\t<mode>\t<decimal-size>\tsha256:<hex>\n
...
```

The last two line forms repeat for the globally path-sorted inventory, including root '.'. Hash the exact byte concatenation with SHA256, print sha256:<lowercase hex>. Portable paths cannot contain tabs/newlines, so this encoding is unambiguous. Counts derive from entries and need no second mutable field. Digest includes both inventory and scope-policy/root/exclusion identity. The row's hash must first match its own submitted inventory, then the fresh authoritative inventory hash. Neither rewriting exclusions (unknown schema) nor removing entries (set comparison) can preserve a prior review. Changing the root requires a different digest and must also match the invocation's root. A dishonest author can calculate a new receipt; that is a NEW self-authored assertion, not proof an independent reviewer re-reviewed it.

Limits: preserve snapshot bounds, 100,000 entries, depth 64, 1 GiB per regular file, 8 GiB aggregate; stream file bytes with existing bounded readers/copy routines. Apply one budget to the full logical inventory including design overlay; do not reset budgets per directory, row or overlay. V2 YAML and each required design cover obey the existing 16 MiB designArtifactMaxBytes reader limit in internal/gates/confinement.go (exact epic source verified after graph discovery). Generation must enforce the 16 MiB serialized document bound too, and warn that merging multiple independently generated rows must fit that same document bound. If inventory YAML cannot fit, fail GV_EVIDENCE_LIMIT with the size and limit rather than truncate. No flags to relax limits in this story. Existing explicit-file hash mode retains its separate 16 MiB/file bound. Scope inventory limits alone do not guarantee a maximum-size manifest can be stored.

Reject symlink root leaves, directory/file symlinks anywhere in included scope, special files, invalid portable names, duplicate/casefold names, and replaced roots/entries during capture. The held rooted reader prevents escaping through descendants; it must never resolve a row path against the ambient original root. A no-follow check of the root leaf is required; do not invent a ban on platform ancestor aliases such as macOS /tmp.

Alias policy distinction: duplicate logical receipt paths and path-normalization aliases are invalid independently of filesystem identity. Hardlinks at two different logical paths are not inherently a byte-freshness exploit: complete immutable snapshots retain both paths and bytes, and additions/removals still differ. Nevertheless this proposal conservatively rejects same-inode regular-file pairs during attestation capture, matching the existing explicit-file CLI policy and making alias fixtures deterministic. This is the approved policy, not a claim that copies lose byte safety. Cross-root hardlinks outside the enumerated root cannot be exhaustively discovered; held descriptor/identity/content revalidation handles observed mutations. No inode identity persists across invocations, because checkout portability must remain possible.

## Root custody and approved exact API contract

Existing signatures remain source-compatible:

```go
func CheckAttestations(design string) *Gate
func AttestationClaimIDs() []string
func ContentHash(path string) (string, error)
func stableAttestationHashes(paths []string) ([]string, error)
func (s *Snapshot) RunSelected(impl string, sel Selection, opt RunOptions) []*Gate
func SelectRunAndNote(design, impl, gateList string, opt RunOptions) (Selection, []*Gate, string, error)
```

CheckAttestations(design) remains a design-only compatibility wrapper calling checkAttestationsInSnapshot(design, nil), with the existing rooted readDesignFile behavior and diagnostic wording preserved. It does not acquire an implementation root or newly claim whole-operation snapshot guarantees for standalone legacy callers. Plan/history remain usable, but any current row produces GV_IMPL_REQUIRED and legacy behavioral rows produce GV_MISSING_IMPLEMENTATION_SUBJECT. It does not infer impl from cwd, Git root, BUILD text or evidence. Do not change it variadically: a distinct API keeps absent-root semantics visible. Production suite callers always pass the held design snapshot to the internal checker, and the new WithImplementation convenience API acquires the full snapshot itself.

New gates APIs/types (owned by attest.go, private subject fields protect capture construction):

```go
type AttestationReview struct { Claim, Kind, Attestor, Date, Note string }
func CheckAttestationsWithImplementation(design, impl string) *Gate
func RenderAttestation(design, impl string, review AttestationReview) ([]byte, error)
type attestationSubject struct { /* private captured manifest, root locator, policy */ }
func checkAttestationsInSnapshot(design string, subject *attestationSubject) *Gate
func (s *Snapshot) captureAttestationSubject(impl string) (*attestationSubject, *designlock.AttestationTreeSnapshot, error)
func renderAttestationInSnapshot(design string, subject *attestationSubject, review AttestationReview) ([]byte, error)
```

CheckAttestationsWithImplementation acquires one Snapshot, captures through the new capability, checks, and explicitly releases before returning its Gate. It folds custody/close/release failures into Gate.Errs and returns no published current-review count on failure. RenderAttestation does the same but returns buffered YAML only after capture/revalidation/close/release all succeed. It generates hashes and exact manifest; it does not execute tests or attest reviewer identity. Direct convenience caller failures use the same Gv categories, never optimistic success on absent impl.

Suite wiring: when Gv is actually applicable and impl is nonempty, use captureAttestationSubject for the existing implementation preparation; pass the resulting subject to checkAttestationsInSnapshot. Other gates receive an immutable implementation Path with the SAME inclusion/topology as the existing MaterializeExternalTree result. Otherwise keep existing preparation. Add one private RunOptions field `attestationSubject *attestationSubject`; no exported user-settable snapshot paths. At Gv call use checkAttestationsInSnapshot(design, opt.attestationSubject). Keep current selection predicates, MAC-p8ce changes, cargo authority handling and remapping. Add a private suite-local `implementationSnapshot` interface with Path() string, Logical() string and Close() error so existing generic and new attestation captures can share the local stable variable; it is not a new public API. Generic impl and cargo cleanup stays in RunSelected. Strict attestation captures are instead registered on Snapshot and retained until Release; error exits either close an unregistered capture immediately or let the registered owner close it, never both. A current row without impl is diagnosed by Gv after parsing, not by a global ban on --gate gv without --impl.

Approved narrowly scoped shared capability (new internal/designlock/attestation_snapshot.go; preserve generic MaterializeExternalTree callers):

```go
type AttestationTreeEntry struct {
    Path string
    Directory bool
    Mode uint32
    Size int64
    SHA256 [32]byte
}
type AttestationTreeSnapshot struct { /* private tree, full inventory, held source roots */ }
func (l *Lock) MaterializeAttestationTree(path string) (*AttestationTreeSnapshot, error)
func (s *AttestationTreeSnapshot) Path() string
func (s *AttestationTreeSnapshot) Logical() string
func (s *AttestationTreeSnapshot) Entries() []AttestationTreeEntry
func (s *AttestationTreeSnapshot) CheckUnchanged() error
func (s *AttestationTreeSnapshot) Close() error
```

Entries returns a copy, never mutable internal authority. The capability captures a complete logical inventory from held no-follow ORIGINAL source root(s), applies the top-level/nested .git policy, detects hardlink pairs before copies erase alias identities, and proves inventory bytes equal the returned stable tree plus retained design overlay. Evidence filtering/digest is in gates; designlock supplies the unfiltered observed entries (apart from VCS metadata) so unrelated consumers do not learn the attestation schema. It retains the original root handles until final CheckUnchanged and Close. The existing design SourceRoot is a private copy, NOT an original held descriptor; revision 1 did not make that distinction sufficiently explicit.

MaterializeAttestationTree acquisition owns this exact protocol: resolve logical topology lexically; open original authority root(s) below; compare pre-open Lstat, held-root Stat and l.rootInfo for the design; enumerate design THROUGH ITS HELD ORIGINAL ROOT using the existing design fingerprint policy and compare the exact result to l.snapshot AND retained SourceRoot bytes (root identity/mode checked separately). This bridges the earlier private-copy acquisition to the newly retained original authority, failing if they differ; it does not claim the new original handle existed at AcquireReader time. Capture the strict full implementation inventory and copy through held root operations, with witnessed pre/open/read/post identity/content checks and a second complete held-root pass. Prove the copy plus retained design overlay equals the full inventory before returning. Reject nested .git discovered in the strict implementation scope even when the generic design copy omitted it. Required design covers are read only from the already verified private design copy.

| Logical topology | Original handles retained by capability | Stable implementation Path and final authoritative checks |
|---|---|---|
| Disjoint design and impl | Independently open/retain original design root and original impl root once; compare design handle to l.rootInfo | Private implementation copy; held design fingerprint must equal l.snapshot/retained generation; held strict impl inventory must equal captured inventory |
| Impl is ancestor of design | Open original impl root; obtain design root by held impl.OpenRoot(design-relative-path), validate all intermediate real directories and equality with l.rootInfo; retain both handles | Generic-compatible impl copy omits design subtree; full Entries includes that subtree from original inventory, proven equal to retained design copy; final held impl pass checks ALL scope including design; held design pass also preserves original design generation |
| Impl equals design | One original root handle, shared ownership with exactly one close | Path is retained SourceRoot; strict full inventory and generic-policy design snapshot comparison both run through original handle; reject nested .git in strict scope |
| Impl is inside design | Open original design root; obtain impl root by held design.OpenRoot(impl-relative-path), validate intervening real directories; retain both | Path is retained SourceRoot/subpath; final strict impl inventory uses retained impl handle and design-generation check uses retained design handle; root-name/ancestor identity witnesses catch replacement of subpath |

Logical relative strings decide topology but do not grant read authority. No EvalSymlinks-based following inside these roots. Merge full inventory/path/case/identity checks and budgets before evidence filtering. A design subtree containing code/test/config cannot disappear. Original handles capture identity only for the observation interval; inode identities are not persisted across commands. In-design/equal impl with a private-source path supplied by an internal caller must resolve to the known logical root through the owning Lock rather than treat a private temporary directory as a new logical implementation root; reject an unrecognized private path.

AttestationTreeSnapshot.CheckUnchanged owns the final primary attestation proof: enumerate/rehash through its retained original design/impl handles under the above policies; compare with captured generation, full inventory, bytes and root/descendant identity witnesses; check that logical root names still identify the held roots using Lstat (metadata checks, no byte-opening authority); fail on any difference. Closing the capability aggregates root-handle and private-copy close errors. Even the reserved evidence file must not change DURING a single observation, because the held design-generation check binds the parsed row; editing it BETWEEN successful invocations remains permitted by the scope digest exclusion.

Approved precise shared integration choice: implement the new held-root traversal/copy and private state in internal/designlock/attestation_snapshot.go, in the SAME package, using existing snapshotBudget/readSnapshotDir/copySnapshotFile/sameFingerprintFile/validateInventoryPath/newPrivateSnapshot helpers directly. Do NOT call MaterializeExternalTree or TrackExternalTree to establish the strict subject, because their path reopen/exclusion policy is different. A new private helper `captureAttestationRoot(root *os.Root, logical string, policy attestationInventoryPolicy, copyTo string) ([]AttestationTreeEntry, error)` performs the bounded held-root work; private policy distinguishes strict full implementation versus the exact generic design comparison. copyTo is empty for revalidation. Preserve each helper's current limits and join all close errors. Prove equality using the existing fingerprint entry spelling when comparing to l.snapshot; remove the synthetic '.' row only for that existing map comparison, checking root identity/mode separately.

This choice requires ZERO production edits to designlock.go, external_snapshot.go or source_snapshot.go. They remain read-only consumers/providers of existing private helpers/state. No vague helper factoring is authorized: if implementation cannot use those helpers without changing them, return the exact necessary signature/hunks for a further scope review. The new capability has its own held-root walk precisely so generic capture semantics stay unchanged. Its additional duplication/LOC is included in the revised estimate.

Correction to the no-reopen assurance: Gv's manifest/content comparisons and final primary freshness proof use retained original capabilities and verified private copies, never a fresh ambient path as subject authority. Existing Lock.CheckUnchanged (designlock.go2101 onward) and checkExternalUnchanged DO reopen ambient paths via fingerprint/fingerprintRoot; they remain supplementary fail-closed suite guards. A supplementary failure invalidates the result, and a supplementary pass can NEVER substitute for, override or repair a failed held-capability check. Existing workspace/Cargo readers may also keep their own established custody behavior. This proposal does NOT claim every operation in the entire suite avoids ambient reopens. Filesystem name/metadata checks are allowed to detect replacement and cannot supply replacement subject bytes. These checks bind a stable observation, not authenticated execution, continuous isolation after the final read, or an absolute guarantee against every undetectable ABA race on every platform.

## Finalization and publishable current-review findings

Add private Snapshot fields `attestationCaptures []*designlock.AttestationTreeSnapshot`, `attestationPending []*pendingAttestationResult`, `attestationCustodyErr error`, `attestationFinalized bool`. Define the private pending result in attest.go as its Gate pointer, successful current-row count, scope/file count(s) and pending-note identity. captureAttestationSubject sets a private subject field `pendingResults *[]*pendingAttestationResult` to `&s.attestationPending`; the internal checker appends through that pointer, so suite and convenience callers use the same owner with no post-call registration gap. A current subject without its owner/pending sink is an internal custody error, never an immediate publish path. This sidecar avoids modifying Gate/gates.go. The internal checker records semantic errors immediately, but stores successful implementation/current counters and success notes in the sidecar, NOT public Counts or Notes. Public Gate includes only `current implementation review pending final snapshot release` until finalized. Design-plan/history/generic row bookkeeping keeps existing semantics and must not be labeled current implementation.

Exact lifecycle:

1. Snapshot.RunSelected captures/registers the strict capability, runs gates, remaps logical paths, calls its strict CheckUnchanged AND existing Lock.CheckUnchanged, and performs existing generic/Cargo cleanup. Any of these custody/cleanup errors is accumulated in attestationCustodyErr, appended as G0/Gv failure, and marks every pending current result invalid. Strict capabilities remain held until Snapshot.Release. RunSelected returns provisional Gv findings with ZERO public current-review/file/scope success counters even when checks have passed so far.
2. Snapshot.Release performs one final strict CheckUnchanged for each retained capability and, when attestation captures exist, one final supplementary Lock.CheckUnchanged before any design-copy cleanup. It then closes each capability, closes workspace and releases Lock, joining ALL errors with accumulated attestationCustodyErr. Do not short-circuit cleanup after the first error. Snapshot.CheckUnchanged itself must accumulate any returned supplementary error in attestationCustodyErr when an attestation capture exists, so a caller cannot observe a failure and have a subsequent restoration erase it. Finalize pending results only after all operations return. Success removes the pending note and publishes stored current counters via Gate.Count plus its scope-boundary note. Any error removes the pending note, publishes NO current counters/positive current notes, and appends `GV_SCOPE_CUSTODY: current review not established; final snapshot validation or release failed` with the cause. Store the joined final error; idempotent Release returns that same disposition, finalizes once and never later resurrects invalid findings.
3. SelectRunAndNote changes its defer-only successful path to derive VersionSkewNote while the snapshot is held, explicitly call Release, then return finalized run/note/error. Early selection errors still release safely; no success result escapes on release failure. The package RunSelected convenience wrapper retains its signature and returns only after Release, which has already finalized/invalidated its gate pointers; it may retain the additional G0 failure for consistency. New WithImplementation and RenderAttestation wrappers use the same finalization sequence, with Render returning nil bytes on failure. Existing Snapshot.RunSelected direct callers must Release before reporting positive current assurance.

Verified hook consequence: internal/hook/hook.go stop acquires the snapshot around 1273 and defers Release at 1277; it emits Gv into an internal text buffer at 1343, but its eventual non-block stopOut JSON is written to the actual writer and clearCheckedState can run BEFORE deferred Release. The green default can also clear state then return nil before release fails. cmd/machinery/hook.go newHookCmd passes output.stdout directly to hook.Run under the installation lock; there is no outer JSON buffer that reliably retracts an already emitted non-block result. A later returned Go error does not prove every host cancels that result. This is a source-supported ordering risk, not a demonstrated host exploit. Suppressing counters alone is insufficient.

The approved scope includes internal/hook/hook.go, only stop's snapshot-finalization/decision order, and new internal/hook/attestation_snapshot_test.go. Keep the returned []*Gate instead of immediately emitting it; collect armed/ratchet and wave-sentinel facts while sourceDesignDir exists; complete snapshot.CheckUnchanged and explicitly snapshot.Release BEFORE rendering/tallying finalized gates, deciding shouldBlock/wave deferral, calling clearCheckedState or emitting any non-blocking stop result. Release is idempotent so the existing deferred cleanup can remain for early exits; early errors may emit an explicit block before cleanup because they authorize nothing. On strict capability, supplementary check, cleanup or Release failure, immediately emit stopOut{Decision: "block", Reason: <custody cause>} and retain checked-state ledger, regardless of cfg.Strict, ratchet presence or wave deferral. The empty-selected-gate branch likewise must release successfully before clearing state or returning a non-blocking result. Existing semantic error/warning, import-arming and wave-defer policy applies only AFTER successful custody finalization; do not change that product policy in this story.

Hook proof: run actual Stop and SubagentStop events against a real local configured design/impl/ledger; stable finalized current review permits the existing configured outcome and clears state; mutate a subject between gate evaluation and finalization, or fail owned snapshot cleanup, and assert exactly one block JSON, no earlier non-block output, retained touched-state ledger and no public current counters. Exercise strict=false and wave-open so those policies cannot waive custody failure. Also retain existing semantic-warning/wave controls after successful finalization. The new tests may use a specifically PM-reviewed callback around the hook's finalization boundary to mutate actual files; they must not mock the gate verdict or accept a bare process error as proof of blocking. cmd/machinery/hook.go and the outer install lock remain read-only; this proposal promises finalized DESIGN/IMPLEMENTATION snapshot custody before a stop authorization, not that unrelated outer host/installation operations are newly transactional.

Required finalization tests: expose no count before release; publish it after successful release; mutate a real original file after RunSelected but before Release and require suppression; replace/rename an original root and require suppression through the retained/name witness; trigger final supplementary ambient failure with held data unchanged and require suppression; observe Snapshot.CheckUnchanged failure, restore bytes, then Release and require the failure remains latched; trigger actual owned private-copy/workspace cleanup or root/filelock close failure using only explicitly reviewed local error-injection seams, require suppression and error propagation. Pair every case with successful cleanup/control. Test SelectRunAndNote, package RunSelected and WithImplementation return paths; generation must return nil/empty stdout on late failure. Do not count an earlier Gate counter plus later G0 error as satisfying suppression. OS close-error forcing may need a narrow deterministic test seam in the NEW capability or suite owned code; PM must approve its exact use, and it must inject the failing lifecycle operation rather than fake a successful filesystem snapshot. No further shared-helper edits beyond the explicitly named suite/hook boundary changes are proposed.

## CLI, migration and current/history outcomes

New generation syntax, alongside unchanged existing commands:

```
machinery attest --design design --claim gt.conformance-test-shape --kind current --impl . --attestor R --date 2026-09-05 [--note text]
machinery attest --design design --claim g2.nfr-content --kind plan --attestor R --date 2026-09-05
machinery attest --design design --claim ga.review-quality --kind historical --attestor R --date 2026-09-05
machinery attest design/ARCHITECTURE.md design/BUILD.md
machinery attest --claims
```

Generation requires all named fields (including explicit date; no wall-clock nondeterminism), exactly one claim, no positional files. --impl is required for current and forbidden for plan/historical. --claims alone preserves byte-for-byte ID/order output. --claims with existing positional files retains its existing precedence; reject combination with any new generation flag. File-hash mode retains its exact output and custody tests. Generation emits one complete v2 YAML document containing the generated row and all required design covers. It does NOT overwrite or merge design/attestations.yaml; docs say review subjects, then merge the generated row into that file. If --claim has no actual required design subject, generation fails rather than manufacture a cover. No --exclude or --include flags exist.

All discovery, validation, hashing, final unchanged checks and close/release errors must finish before first generation stdout byte. On such failure stdout is empty, stderr has the category/remedy, exit 1. Successful generation writes one buffered document; an output write failure can physically truncate stdout and must return nonzero—do not promise that an arbitrary pipe provides atomic writes. Display these exact limits in generation stderr/help and finalized Gv scope notes (therefore CLI and hook output), as well as docs: 'Hashes bind the observed files and scope; they do not prove tests ran, reviewer identity, or judgment correctness. Top-level Git administration and the exact Machinery attestation record are excluded; applications that use them as runtime inputs are outside this review boundary.' The generated row's caller-supplied attestor is attribution text, not authentication.

V1 migration: keep parsing only the original v1 keys. Six g2/four g3/g4.zero-context rows retain their existing design-cover semantics as implicit plan judgments (emit an informational migration note, not a new coverage warning). ga.review-quality becomes explicitly reported legacy historical judgment over acceptance evidence; it grants no current approval. V1 gt.conformance-test-shape/g4.pack-event-discipline/g4.standin-coverage ALWAYS produce GV_MISSING_IMPLEMENTATION_SUBJECT, even with matching BUILD/pack hashes and even without --impl. Remedy names the claim and generation command, tells the user to review the complete implementation/test scope and write v2 kind=current, or explicitly recast the statement as v2 kind=plan. Never auto-add hashes or silently infer a current review from an old row.

Ordinary Gv preserves incremental adoption: owed claims with no row warn; absent evidence after owed artifact activation is still an error. A v2 behavioral plan row reports 'plan only; current implementation review missing' as a coverage warning and cannot discharge the corresponding implementation obligation. All malformed/stale/current-without-impl/legacy-behavior rows error in ordinary and complete modes. --warnings-as-errors and existing --complete warning promotion make missing current reviews blocking; --complete already requires --impl globally. Update check help/error text to name Gv alongside G4/Gt where relevant. Counts distinguish plan judgments, current implementation reviews and historical review records; retain the existing generic attested-claims count for compatibility. Never count a failed or plan/history row as current implementation review.

Ga alone retains its exact existing repository ancestry acceptance semantics, including exported identity and missing-history behavior. Gv historical rows validate the recorded acceptance-file covers and label their judgment historical; Gv alone does not pretend to have executed Ga's ancestry validation. With --gate ga,gv, a historical accepted ancestor and stale current implementation can coexist: Ga passes its history check; Gv fails current freshness. Changing --commit to the old reviewed ancestor cannot repair Gv, which always observes supplied current root. Absence of current review cannot be rescued by ga.review-quality. --complete needs its separate actual current implementation rows as well as valid history. A later Git commit updating only the reserved attestation record keeps the same implementation digest and Ga's ancestor check valid.

## Five-AC RED/GREEN test contract

All new tests use actual files, directories and local Git repositories; CLI process tests build an isolated binary under t.TempDir, never replace installed machinery or run installation hooks. No Docker/Java/Node product/runtime dependency, network, mocks, timing lotteries or skip-if-missing. Fixture setup failure is not RED. Frozen test/fixture bytes and exact existing-test amendments require PM authorization before any author edits. The old production has no v2 interface, so a v2 passing freshness control CANNOT be required on a82277a. Tests must be partitioned and reported by the following stage contract, not collapsed into one universal baseline-positive rule.

| Frozen test category | Exact a82277a outcome | Exact final implementation outcome | Evidence it supplies |
|---|---|---|---|
| A — existing-interface behavioral RED: three legacy behavioral claim fixtures, using existing CheckAttestations/RunSelected/CLI; independently evaluate unchanged and assertion/handler-mutated trees; final assertions require GV_MISSING_IMPLEMENTATION_SUBJECT in BOTH cases | Compiles, setup works, old implementation accepts each legacy design-only row instead of issuing required missing-subject error; assertion fails with that exact observed error absence. Both unchanged and mutated legacy cases are negative cases under the new contract | Both unchanged and mutated legacy cases are rejected with GV_MISSING_IMPLEMENTATION_SUBJECT and never counted current | Actual unsafe legacy acceptance reproduced through existing interfaces. This is the primary required behavioral RED, not v2 sensitivity. The old acceptance is a recorded bug observation, NEVER a frozen passing control |
| B — future-interface acceptance: complete v2 document parsing, generation flags/output, kind rules, new categories, finalize-visible counts; invoke only existing CLI/process or existing public suite symbols so baseline compilation succeeds | Fails specifically because version 2/flags/expected interface behavior is absent; report unsupported-version/unknown-flag output verbatim as INTERFACE_ABSENT, not successful fail-closed freshness or qualifying behavioral RED | Valid generation/schema/current result succeeds, invalid categories produce their exact specified errors, fully released wrapper publishes counters | Frozen desired interface contract. No claim that a baseline unknown flag proves staleness detection |
| C — paired sensitivity after interface exists: for each mutation in matrix below, generate or load frozen independently calculated v2 receipt, verify unchanged actual scope first, then apply exactly one mutation and assert its category/path and zero published current count | First unchanged v2 control fails with known missing-interface behavior; mutation stage is NOT credited as executed freshness evidence. Freeze the entire test without skip/feature probe/conditional success; test remains failing on base | Unchanged control passes with finalized current count, mutation returns the specified error and no current success; evidence-only commit control passes on both invocations | Actual addition/removal/content/scope/alias/replay sensitivity, credited ONLY after both legs execute against implemented interface |
| D — baseline compatibility controls (separate mandatory frozen controls): existing six-g2 v1 plan fixture; valid old filehash/--claims; ordinary partial design-plan coverage; existing local Ga ancestor control | Passes, with old exact outputs/counts where preserved | Still passes with specified compatibility semantics; no added warning on legacy plan migration | Meaningful passing baseline controls accompanying A. These are explicitly compatibility controls, not fabricated valid current implementation reviews |

A tests assert the final contract from the beginning; they do not first assert legacy success and later flip it in GREEN. Separate unchanged/mutated legacy subtests make both negative outcomes observable even if one fails. B/C tests must compile against base and use existing callable seams: a frozen Go test referencing a nonexistent new exported API is forbidden, because compilation failure is not RED. New API unit tests requiring new symbols may be supplemental GREEN tests with explicit provenance; they cannot replace frozen existing-interface/process acceptance tests. Any B/C mutation not reached because its unchanged leg failed is reported NOT YET EXERCISED, never passed. Do not add skip-if-feature-missing, accept-any-error, or baseline-vs-GREEN branches to make this staging look green.

The independent PM base replay must observe A's exact legacy false-acceptance assertion failure plus D's passing compatibility controls, and classify B/C failures honestly. The final PM replay must observe A rejection passing, B interface acceptance passing, C BOTH legs passing, and D compatibility still passing, with frozen bytes unchanged. This satisfies hard TDD without pretending the old product could generate a valid v2 scope or requiring a now-unsafe legacy current assertion to remain valid.

The following matrix specifies FINAL behavior for B/C/A as indicated above. Its passing-current controls are required when the interface exists; they are not asserted to pass on a82277a.

| AC | Real passing control | Single challenge and exact expected observation |
|---|---|---|
| 1 | Full source/test/config tree, valid generated current receipt; Gv current-review count is 1 | Remove assertion from test while keeping BUILD/oracle citation: GV_STALE_CONTENT for test path, current count 0 |
| 1 | Same generated tree | Alter event handler while pack unchanged: GV_STALE_CONTENT for handler path |
| 1 | Same generated tree | Add/delete/rename source, test, config, extensionless and generated file, one case each: GV_SCOPE_INVENTORY with added/removed path |
| 1 | Ignored directory/file exists and is explicitly inventoried | Add ignored/untracked handler; change .gitignore/contract ignore; both inventory/config changes invalidate. No matching source extension is required |
| 1,4 | Complete submitted entries and root locator | Remove one entry with old hash -> GV_SCOPE_HASH; recompute submitted hash -> GV_SCOPE_INVENTORY; add exclude field -> GV_SCHEMA; narrow --impl or row.root -> GV_SCOPE_ROOT/GV_SCOPE_HASH; unchanged subset cannot satisfy old full-root manifest |
| 1,4 | impl contains design, with design/helpers/test.go captured in logical overlay | Change/add design subtree code -> GV_STALE_CONTENT/GV_SCOPE_INVENTORY, proving generic snapshot omission cannot hide it |
| 2 | v2 plan row on design claim, v2 current on behavior, v2 historical on ga | Wrong kind/implementation-field combination -> GV_KIND/GV_SCHEMA; behavioral plan produces missing-current warning and never current count |
| 2,5 | Real Git commit A with valid M1 acceptance and later evidence commit B; actual CLI --gate ga,gv --impl root is green | Change source at C, keep M1 ancestor A and replay receipt B: Ga history remains checked; Gv GV_STALE_CONTENT; explicit --commit A still cannot make Gv current |
| 3 | Legacy six-g2 fixture remains accepted as plan with same existing counts and no added warnings | Legacy gt/pack/stand-in design-only row -> GV_MISSING_IMPLEMENTATION_SUBJECT naming root review/migration remedy, with/without --impl |
| 3 | Explicitly migrated v2 current row from actual scope | Legacy row relabeled kind without v2 version fails schema; v2 current with empty/missing subject errors; v2 plan conversion is explicit and incomplete |
| 4 | Unchanged scope checked repeatedly, including new local commit modifying only design/attestations.yaml note/date | Remains fresh with same scope hash, no HEAD coupling; modify adjacent evidence/handler.go or a different attestations.yaml -> inventory/content failure |
| 4 | Distinct portable file paths, actual regular files | Duplicate receipt path, ./ alias, casefold alias -> GV_SCOPE_PATH; leaf/directory/root symlink -> GV_SCOPE_CUSTODY; real hardlink pair -> GV_SCOPE_ALIAS under the approved conservative policy |
| 4 | Top-level .git is ordinary repository metadata, with commits changing it | nested src/.git/handler.go -> GV_SCOPE_UNSUPPORTED_METADATA; arbitrary .ignored/handler.go is included and addition fails freshness; generated/tests assertions remain included |
| 4 | Stable real held root/copy using existing approved fingerprint/copy callback seams | File/root replacement, file mutation between reads, design-overlay mutation -> GV_SCOPE_CUSTODY or underlying G0-snapshot custody error; stdout empty on generation, no current success counter |
| 4 | Bounded small regular tree | FIFO/special entry -> custody failure before open; limit+1 entries/depth/size/aggregate -> named limit failure, no partial inventory; use existing reviewed lowered-budget helper seams where actual 8 GiB creation is impractical, not mocked files |
| 5 | Spawn isolated real binary; --attest generation -> write row -> check --gate gv --impl root returns 0 | Change test/handler and rerun returns 1 with Gv stale category; without --impl current receipt gives GV_IMPL_REQUIRED; --claims and old filehash output remain exact |
| 5 | Complete-mode fixture valid under all other gates | Missing behavioral current row is the sole warning promoted to blocking; ordinary corresponding plan case warns only; no unrelated missing phase artifact accepted as RED |
| 5 | Actual built CLI generation emits the complete valid v2 document only after renderer success; approved callback fires without mutation in paired real RenderAttestation control, which returns valid bytes after successful finalization | Actual built CLI ordinary input and alias failures: exit exactly 1, zero stdout bytes, specific diagnostic. Actual RenderAttestation original-subject mutation at first final Release and approved private-copy cleanup fault: callback demonstrably fires, actual operation fails with its particular cause, returned bytes are nil. Independently verify delivered CLI/error/output wiring to compose those renderer observations into the CLI before-output guarantee. A separately injected late-release fault in the standalone CLI executable is UNOBSERVED, not claimed as an executed end-to-end case. Output-sink failure retains its separately stated nonzero/possible-partial-output contract. |

Parser tests separately assert duplicate keys, unknown fields, wrong types, enum, version, path grammar, hash format, conditional fields, ordering and exact inventory self-hash. They are category B, not substitutes for category C stale-tree filesystem/CLI proof. Categories A and D provide the independently replayed base behavioral failure and compatibility control. Record each command, expected failure class, reached/not-reached mutation stage, and final result separately. A single aggregate nonzero go test or CLI exit is insufficient evidence.

## Approved AC5 compositional proof and limits
Original architect and independent challenger APPROVED the narrow clarification, SHA256 5fc3193106a803b7760020b646ed4db7b3d417f5e196b2266d10cf8227733937. It resolves proof composition only; all five product AC, approved R2 SHA256 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179, ownership, two exact test amendments/four authorized seams and A/B/C/D staging remain unchanged. No new seam, runtime, user choice, dependency or implementation authorization is introduced.

The product requirement remains: in attestation GENERATION mode, discovery, hashing, final validation and owned snapshot cleanup/release finish before the first stdout byte. A failure from those operations produces CLI exit 1, an actionable error on stderr, and empty stdout. RenderAttestation returns nil bytes and the actual error on such failure. An output-sink error after successful finalization remains the separately documented case where physical partial stdout is possible and the command must fail. The empty-stdout rule does not prohibit `machinery check` from printing its blocking diagnostics.

The final evidence bar is conjunctive; all three parts are required on the same delivered GREEN revision:

1. **Observed renderer lifecycle proof.** Use only PM-authorized seam 3 in the optional internal/gates/attest_green_test.go, with its existing safety/child-process restrictions. The fired no-fault control must call real RenderAttestation and return a complete valid document and nil error. Separate actual original-file mutation and approved owned-copy regular-entry-to-symlink cleanup fault must each reach the first finalization callback, return a non-nil error and `body == nil`, and preserve logical-path diagnostics. The cleanup case must expose the concrete real private-snapshot cleanup cause, not merely an earlier generic validation error; retain sentinel/cleanup-safety evidence. These are GREEN supplemental observations, not compile-failing RED or independently observed CLI late faults. No fake renderer return or directly injected verdict is acceptable.

2. **Independently reviewed delivered control-flow closure.** Review the exact final source and call sites, with SHA/path/line evidence, from newAttestCmd's generation branch through real gates.RenderAttestation, its final Snapshot.Release/error join, and command error-to-exit handling. Establish that renderer success is the only path to the generation stdout write; the real renderer itself has no stdout emission; all non-nil renderer errors take the common failure branch before any write; that branch reports stderr and maps to exit 1; and no fallback, logging, deferred emission, retry or alternate generation path prints a partial document on an error. Verify final validation/cleanup is completed inside the renderer before it returns bytes, rather than deferred by a CLI-owned snapshot until after printing. Include command-output tracking and root command/error handling to the extent they can affect this path. Actual CLI ordinary-error tests must exercise this same renderer-error branch, not only Cobra unknown-flag/arity rejection. If the delivered implementation does not have this closure, compositional evidence is incomplete and cannot pass PM review.

3. **Observed standalone process proof.** Build the ordinary isolated CLI from that exact GREEN revision, recording build command/binary path and source SHA (plus binary digest if used in the delivery evidence). Run actual generation success and ordinary missing/unreadable-input and alias failures using real fixture files; require exact exit 1, zero stdout bytes and the intended error category for failures. Retain the already-required real CLI changed-implementation, historical/current distinction, missing implementation, migration and compatibility cases. Do not replace them with renderer-only tests, package mocks or source inspection. Existing A/B/C/D staging remains unchanged: unsupported new flags/version on base is INTERFACE_ABSENT, not freshness or late-failure proof.

Final report wording must distinguish the evidence tiers: `OBSERVED: real renderer late validation/cleanup failure -> nil bytes; OBSERVED: built CLI success and ordinary renderer-input/alias failure -> expected output/exit; REVIEWED: exact delivered renderer-to-CLI error/output closure; COMPOSED: CLI late renderer failure -> exit 1/empty stdout; UNOBSERVED: independently injected standalone CLI late-release failure; UNFORCED: individual OS root-handle/filelock Close primitive errors.` The approved actual private-copy cleanup failure supplies the lifecycle-fault alternative; every OS Close primitive need not be forced. None of these observations establishes authenticated execution or reviewer honesty.

Option B is not necessary merely because the standalone late-fault injection is unobserved. The new generation adapter has a reviewable error/output boundary, and real renderer faults plus actual CLI traversal of that same error branch can establish the bounded guarantee compositionally. B would require a further concrete review only if final delivered wiring prevents establishing that closure or an explicit requirement is added for independently injected standalone CLI late-fault observation. Do not infer a production fault environment variable, exported callback, CLI refactor or wider ownership from this clarification.

Source status: exact a82277a cmd/machinery/attest.go currently only calls stableAttestationHashes and then prints buffered old filehash output; it DOES NOT call the future RenderAttestation API. The inspected current io.go/commandResult/main.go path maps command errors to nonzero exit and tracks write errors, but this does not establish the future generation path. Exact diff a82277a..6cb2d974 on these relevant attestation/error-output/suite paths was empty. Delivered GREEN wiring/source/error/defer closure is therefore a mandatory future independent review obligation, not an already verified implementation or executed proof.

## Approved ownership, exact authorized test amendments and cost

Retain declared ownership: internal/gates/attest.go; internal/gates/attest_implementation_test.go (new); cmd/machinery/attest.go; docs/attestation-evidence.md. Approved directly related additions: internal/gates/suite.go only Gv capture/private option/wiring; cmd/machinery/check.go only Gv-facing help/messages; internal/gates/attest_test.go; cmd/machinery/attest_test.go; cmd/machinery/attest_implementation_test.go (new real process/local-Git tests). Optional internal/gates/attest_green_test.go only if independent GREEN supplemental proof needs a separate file, not to change frozen RED. Historical/Ga tests can live in the attestation implementation tests without editing accept.go or accept_test.go.

Approved concrete scope expansion: internal/designlock/attestation_snapshot.go (new capability) and internal/designlock/attestation_snapshot_test.go (new real FS/custody tests); internal/hook/hook.go only stop's finalization-before-decision/state-clear ordering; internal/hook/attestation_snapshot_test.go new real-event/ledger tests. The suite.go ownership includes capture lifecycle, pending-result finalization in Release, and explicit-release convenience wrappers described above. designlock.go/external_snapshot.go/scale.go/source_snapshot.go/snapshot_inventory.go/portablepath/cmd/machinery/hook.go and generic snapshot callers remain read-only. No unspecified helper factoring is authorized. No new ownership is implied for MAC-olrx/MAC-p8ce files or execution/container policy.

Exact legacy amendments already authorized by independent PM (no broader changes):

1. internal/gates/attest_test.go TestAttestationMutations, case "wrong version" around line 176: change input version 2 to 3 and expected supported-version diagnostic to 1-or-2; add separate accepted-v2 and malformed-v2 cases. Do not merely delete the assertion.
2. cmd/machinery/attest_test.go TestAttestRejectsIdentityAliases: independent PM selected the existing mandatory alias-case repair, replacing the os.Link failure t.Skipf with t.Fatalf("create required hard-link alias fixture: %v", err); preserve the real alias, CLI invocation and existing failure/empty-stdout/diagnostic assertions. No alternative case is authorized and required alias proof may not skip.
3. Keep existing six-g2 clean/count/coverage-warning tests and explicit-file custody tests unchanged unless the reviewed implementation demonstrates another exact necessary amendment. Existing Gv coverage tests retain partial design-plan adoption semantics. Exact epic text search of attestation_version/attestEvidence/attestRowFor in internal/gates and cmd/machinery, plus the three behavioral claim IDs, found no additional legacy behavioral current-success fixture in that bounded scope. Other hits are cmd/machinery/repository_contract_test.go's role-document vocabulary assertion and internal/gates/failclosed_io_test.go TestAttestationPackTraversalErrorIsBlocking; neither changes. Preserve internal/gates/determinism_hardening_test.go TestAttestationRejectsSymlinkReferent and TestAttestationClaimMustCoverItsSubject unchanged, including their legacy design-only rooted error behavior. If RED discovery finds another exact affected test elsewhere, request its named amendment; do not weaken it in GREEN.

Revised realistic forecast (PM POST-FREEZE COST REVIEW, SAME-STORY FORECAST ADJUSTMENT JUSTIFIED): approximately2710–3310 total, retaining13 required/optional14 paths. Components:~1485 RED after exact setup repairs;175–375 GREEN supplements;900–1250 production;150–200 docs/help. Actual47ba449 RED is1487 changed lines across five tests, versus1478 at7ec5d609. Distinct approved gates, built-CLI/Git/current-history/complete-output and configured hook/state/custody boundaries explain the increase; current tests already reuse testgit.Run, copyDirInto and package helpers. A new cross-package fixture API would add ownership/couple proof, while relying on future renderer output would destroy independent baseline fixtures. This is not proof that no lines can be saved; it is why LOC reduction cannot remove required controls or change frozen bytes. No story split, new API/file/generic factoring or changed seam is authorized. R2, AC5 composition, all five AC, staged proof, custody/finalization/hook safety and read-only boundaries stay exact. Report actual final files/LOC/runtime and investigate further material overrun; no cap-based omission or automatic acceptance. No preflight now; final gate remains responsible.


## MANDATORY SKILLS
- developer; codebase-memory; pm_acceptor.

## Delivery Requirements
Append exact RED/GREEN SHAs, commands/output, ownership inventory and per-AC proof. Use pvg story deliver; PM accepts separately. Never invoke pushing pvg story merge.

## nd_contract
status: new

### evidence
- Created 2026-09-05 from assessment and source-verified interfaces.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified

## Acceptance Criteria


## Design


## Notes
ANCHOR ROUND-1 RUNTIME CLASSIFICATION: This story's required current cases are service-free Go/native filesystem/local-process tests and real local CLI paths; no Docker/Java/Node dependency is implicit in ordinary native suites. Preserve actual non-mocked path tests. If implementation introduces any external runtime-backed case, it MUST add a dedicated closed fragment under testdata/integration-lanes via the shared required lane and declare ownership/dependency before delivery; no skip-if-missing, env-gated omission or reliance on later incidental execution. Missing service cannot silently convert required coverage to success.
DISPATCHER CONTRACT HOLD: RED author verified clean story/MAC-p7jd at a82277a and stopped before source/test edits, tests, commits or delivery. Existing closed v1 attestation format and design-only CheckAttestations interface cannot express the implementation scope promised by AC1/2/5. Sr PM is reviewing the smallest source-verified contract/ownership repair; schema/CLI choices and suite.go wiring need review before RED. Healthy retained worktree and claim preserved; no recovery cleanup. User-pending strict execution containment policy is separate and must not be silently imposed on freshness binding.
SR PM SAME-STORY PRE-RED ARCHITECTURE TASK (MAC-p7jd; scope proposal, not implementation or TEST-EDIT authorization)

Disposition: retain this P0 repair under MAC-ui8a, with all five canonical AC unchanged. The paused RED author has no authorized schema/API to test yet. Route a bounded read-only technical contract author within this story, followed by independent contract review and explicit PM RED/test-edit authorization. Do not create a prerequisite or couple MAC-l7m0: review freshness is distinct from authenticated execution, containment, or reviewer honesty. Existing user requirements suffice; no new product question is identified. No status/label/assignee/worktree change is requested.

Verified source boundary:
- internal/gates/attest.go: CheckAttestations(design string) *Gate has no implementation-root input. attestationRequiredPaths(g *Gate, design, claim string) []string binds gt.conformance-test-shape to BUILD artifacts and g4.pack-event-discipline to pack. checkAttestationFreshness(g *Gate, design string, row attestRow) hashes only design-relative covers. The closed v1 root/row/cover schema has no implementation scope or plan/current/history field; cover keys are path/hash, with sha256: plus 64 lowercase hexadecimal digits.
- cmd/machinery/attest.go: stableAttestationHashes(paths []string) ([]string, error) hashes explicit files only; newAttestCmd offers positional paths and --claims, not scope generation. It opens all files, rejects identity/path aliases, reads and revalidates all before emitting output. Its helper is cmd/machinery/scale.go: openStableRegular(path string) (*stableRegularFile, error), read and revalidate. It rejects nonregular/symlink leaves and oversize files, compares opened identity, and rechecks identity/metadata/content. These are constraints to preserve, not proof of complete rooted directory custody.
- internal/gates/suite.go already accepts impl in (*Snapshot).RunSelected(impl string, sel Selection, opt RunOptions) []*Gate and materializes it through (*designlock.Lock).MaterializeExternalTree(path string) (*ExternalTreeSnapshot, error), then passes the stable path into runSelectedInSnapshot. Gv discards that input by calling CheckAttestations(design). cmd/machinery/check.go already sends --impl through SelectRunAndNote; its help currently describes G4/Gt. Scope/API review must use the held snapshot rather than reopen an ambient implementation root.
- internal/designlock/external_snapshot.go materialization uses held no-follow rooted copy plus tracked pre/post fingerprints for external trees and retained design source paths for in-design trees. Its existing inclusion/exclusion rules and limits need explicit comparison with the new attested-scope contract; mere reuse does not establish that they cover every AC1 subject.
- Exact epic a82277a suite.go was read. Accepted MAC-p8ce parent-owned relational Gt selection changes are present and must be preserved. A bounded Gv wiring change must not revert them.
- internal/gates/attest_test.go currently requires rejection of version 2. If the reviewed contract chooses a version bump, identify that exact legacy assertion for independent PM amendment; do not treat compilation failure or unreviewed test weakening as RED.

Required contract-author deliverable (proposal in task handoff/shared story note; no source, tests, or docs edits in this read-only phase):
1. Exact closed schema/version decision and representative valid/invalid examples for plan-only judgment, current implementation review, and historical milestone acceptance; claim classification, duplicate/unknown/wrong-type rejection, legacy migration, missing-subject diagnostics, and current-vs-history gate outcomes.
2. Exact public/internal API signatures and root authority flow from real CLI/check and shared suite callers through the held design/implementation snapshot. Specify absent --impl, ordinary versus --complete behavior, logical portable paths versus private snapshot paths, and compatibility for existing direct callers.
3. Complete explicit inventory authority and deterministic digest format: which code/test/config paths and types participate; additions/removals/renames/content changes; ignored/untracked files; narrowly justified evidence-only/generated exclusions; scope-policy binding and anti-narrowing; path ordering/normalization and alias/symlink/race/limit failures. Exclusions cannot create a loophole where changed behavior remains current, and whole-HEAD binding cannot invalidate harmless evidence-only commits. Explain avoidance of self-referential attestation hashes.
4. Exact CLI generation syntax, output schema and failure/partial-output semantics while preserving existing explicit-file hash and --claims behavior where compatible. State accurately that hashes bind observed subjects, not reviewer honesty or test execution.
5. Precise Ga-history/Gv-current interaction and compatibility migration, with historical ancestor acceptance retained as history but insufficient for present approval.
6. A concrete per-AC real filesystem/git/CLI test matrix with matched valid controls and intended rejection categories: unchanged scope and evidence-only commit pass; removed assertion, changed handler, added code/test/config (including excluded-file challenge), deleted/renamed file, scope narrowing, stale replay and path alias/symlink changes fail. Separate parser/schema proof from freshness proof; specify deterministic custody tests using existing permitted seams only after PM review. No mocks, skip-if-missing, Docker/Java/Node requirement, or authenticated-execution claim is introduced.

Proposed implementation ownership for subsequent review ONLY: retain current attest.go, attest_implementation_test.go, CLI attest.go, attestation-evidence.md and exact associated tests; add internal/gates/suite.go solely for reviewed Gv root/API wiring and cmd/machinery/check.go only for required attestation-facing CLI diagnostics/help/wiring, with named associated tests if needed. scale.go and internal/designlock remain consumed/read-only unless the author demonstrates a specific necessary gap and receives a separate scope review. Final API/schema, exact test-edit list and ownership must be canonicalized and independently reviewed before RED resumes. Original ~4-7 files/<1000 changed LOC is an estimate, not permission to trim proof: contract author must give a realistic revised file/LOC estimate if bounded wiring/schema tests exceed it; report actual cost later.

Evidence/custody: initial instruction/context checks, shared pvg issues show, graph-first symbol/call discovery plus exact source reads; graph coverage generation 2026-09-05T23:08:53Z reported metadata_match/no_recorded_issue for nine cited paths, best-effort only. Exact git show a82277a:internal/gates/suite.go verified epic wiring and preserved predecessor behavior. Earlier nonexistent guessed helper path was a reported lookup error, corrected by graph discovery to scale.go; it is not a product defect. No new experimental parser/freshness bypass, tests, implementation or delivery is claimed.
SR PM CANONICAL CONTRACT REPAIR — architecture revision 2 independently APPROVED; RED authorization pending.
Canonicalized exact approved schema/API/CLI/full-root inventory/custody/finalization/test staging under MAC-p7jd. Proposal SHA256 verified as 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179. All five user AC preserved byte-for-byte. Three review clarifications incorporated: latched/idempotent Release with use-after-release rejection and repeated pre-release runs; late error logical-path remapping; scope limits on DISPLAYED current-review reports without forcing silent hooks to emit.
Reviewed 13-file ownership and 1,900–2,900 LOC forecast supersede original estimate transparently; optional fourteenth file is reported GREEN-supplemental only. No generic snapshot helper edits or execution-policy dependency. Existing claim/status/labels/dependencies preserved. This canonicalization is neither TEST-EDIT AUTHORIZED nor approve-red. Independent PM must name two exact existing-test amendments/alias choice and any concrete new callback/fault/budget seam before RED authoring; new-symbol helper tests are supplemental GREEN, not compile-failing RED.
Supported single-executable EDITOR guarded the exact old canonical Description. Shared pvg readback matched the new canonical text exactly and retained all prior non-Description content. No source/test/docs/worktree or installed assets changed.

Prior canonical Description retained verbatim for append-only decision history:
<previous_canonical_description>
SR PM terminal canonical-scope checks: pvg lint --backlog --epic MAC-ui8a PASSED (33 issues; 0 errors, 0 review findings); pvg nd dep cycles reported no cycles; pvg rtm check --epic MAC-ui8a PASSED (18 stories, 2 closed; 0 tagged requirements extracted/0 uncovered, so this is structural coverage only, not product AC proof). Canonical readback matched approved embedded contract and all five original AC byte-for-byte; original Description retained in append-only notes. Pending independent PM test-edit/seam authorization; no approve-red or queue advance performed.
PRE-RED PM EXACT TEST-EDIT / SEAM AUTHORIZATION — MAC-p7jd — 2026-09-05

Verdict: AUTHORIZED ONLY FOR THE BOUNDED USES BELOW. This is independent pre-RED authoring authorization, not approve-red, GREEN dispatch authorization, delivery, acceptance, architecture replacement, or a waiver of proof. Keep in_progress, hard-tdd, assignee dev-MAC-p7jd and the healthy retained worktree/claim. Architecture revision 2 SHA256 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179 and its five AC remain authoritative.

TEST-EDIT AUTHORIZED: internal/gates/attest_test.go — only TestAttestationMutations case "wrong version": change the evidence version from integer 2 to integer 3, and replace expected diagnostic "attestation_version must be the integer 1" with "attestation_version must be the integer 1 or 2". This preserves rejection of unsupported versions while permitting the approved closed v2 schema. Add separate new accepted-v2 plan and malformed-v2 schema tests in this owned file, scoped to the canonical closed v2 contract (real existing attestFixture/design files and CheckAttestations; no new API references). The valid control must actually accept a well-formed v2 plan row; malformed fixtures must assert their particular schema/kind failure and no current success, not any-error. Record/freeze their exact discovered leaf names. All other cases, helper definitions, existing assertions and existing tests in this file remain unchanged.

TEST-EDIT AUTHORIZED: cmd/machinery/attest_test.go — only TestAttestRejectsIdentityAliases: replace the os.Link setup failure t.Skipf("hard links unavailable: %v", err) with t.Fatalf("create required hard-link alias fixture: %v", err). Keep actual os.Link, real CLI invocation, exit 1, empty stdout and "alias the same file identity" assertions unchanged. This selects the mandatory existing-case amendment, not an alternative new alias case. Setup failure is an infrastructure/setup failure, never passing, skipped, or behavioral RED.

Carry [test-edit-authorized] in each commit subject containing either existing-test amendment; the frozen RED commit also needs the normal tdd-red marker. This authorizes no later GREEN edits to frozen files. Other existing tests, including six-g2/count/coverage controls, explicit-file custody, determinism_hardening tests, failclosed_io tests, accept.go/accept_test.go and predecessor tests remain unchanged absent a further named review.

SEAM 1 — AUTHORIZED EXISTING-API FROZEN TESTS:
In new internal/gates/attest_implementation_test.go, use AcquireSnapshot, RunSelected, TrackExternal, CheckUnchanged, DesignPath and Release directly. No new seam is required for these tests. Pair valid unchanged v2 current scope with each actual owned filesystem operation after RunSelected and before Release: original subject content mutation; original root rename/replacement; separately tracked external-file mutation while held strict design/impl inputs remain unchanged; observed CheckUnchanged failure followed by exact byte restoration. Assert zero public current-review/file/scope success counts while provisional, publish only after successful Release, suppression after each failure, latched error despite restoration, logical paths, same final disposition on repeated Release, no resurrection, repeated pre-release runs retaining all pending results, and fail-closed use after release. No mutation may be silently credited if the v2 unchanged control fails first. These are B/C on base as applicable, not additional A proof. Real original/root mutations must remain confined to test-owned fixtures with matched no-mutation controls.

SEAM 2 — AUTHORIZED EXACT HOOK WRITER INTERFACE:
In owned internal/hook/hook.go, an unexported optional interface on the existing io.Writer may have exactly:
    interface { beforeAttestationFinalization(*gates.Snapshot, []*gates.Gate) }
Invoke it exactly once on each successful acquisition/selection path reaching stop finalization, after collecting gate/ratchet/wave facts, immediately before the final CheckUnchanged plus explicit Release, before rendering/tallying/policy/wave deferral/clearCheckedState/non-blocking output. On the empty-selection path pass nil gates and invoke before that branch's finalization and state clear. Early already-blocking error exits need not invoke it. Ordinary writers do nothing; no public config, flag, environment fault switch or production fault verdict is added.

New internal/hook/attestation_snapshot_test.go may implement this method in a recording writer in package hook, so frozen tests compile before the interface exists. The method may record firing, inspect provisional gate values and mutate a precisely identified actual fixture file or owned private-copy file; it cannot alter Gate fields, return an error/verdict, call Release recursively, skip normal checks, replace the snapshot, or manufacture success/failure. Require a B mandatory-fired test (base absent callback = INTERFACE_ABSENT) and C tests that first establish a valid unchanged finalized v2 control, then challenge the same setup. Original-file mutation and private-copy regular entry replaced with a symlink are approved operations. Private-copy paths must be obtained from the actual snapshot DesignPath and validated as below, not discovered by broad temp globs. Assert exactly one block JSON, no preceding allow/non-block JSON or output, retained touched ledger, zero published current counters/positive current notes and logical-path diagnostics. Cover actual Stop and SubagentStop, strict=false, strict=true and open-wave policy, plus empty-selected-gate finalization. Preserve semantic warning/wave no-fault outcomes. For a successful no-fault fixture the existing configured behavior and ledger-clear expectation must be demonstrated; a wave-deferred semantic-error control follows the existing ledger retention policy instead of inventing a clear requirement.

SEAM 3 — AUTHORIZED GREEN-SUPPLEMENTAL FIRST RELEASE CALLBACK:
Only internal/gates/suite.go may add:
    var attestationBeforeFinalRelease = func(*Snapshot) {}
Invoke only on the first Release before its final validation/cleanup; repeated Release must return its already-latched disposition without callback, revalidation or refinalization. This callback cannot return or directly inject an error/verdict, mutate Gate/pending/custody fields, replace handles or call Release recursively. Tests may inspect state and perform actual original-file mutation or the reviewed private-copy symlink cleanup fault. Install, count firing and restore the prior callback with cleanup; no parallel tests or goroutines may race the package global.
Tests referencing this new variable belong only in optional new internal/gates/attest_green_test.go, reported as supplemental GREEN proof and its file/LOC cost. Exercise SelectRunAndNote, package RunSelected, CheckAttestationsWithImplementation and RenderAttestation with matched fired no-fault controls. Require correct joined error propagation, suppression/finalized counts, logical paths and RenderAttestation nil bytes. For the cleanup case require the concrete private-snapshot cleanup cause from the actual operation, not merely an earlier generic validation error. New-variable compilation failure is never baseline RED.

SEAM 4 — AUTHORIZED NEW CAPABILITY COPY/BOUND TESTS:
Reuse, without editing designlock.go or its existing tests, existing testAfterSnapshotCopyReadChunk func(string) from new internal/designlock/attestation_snapshot_test.go. Arm only after unrelated AcquireReader preparation; match the exact owned target label/path passed into copySnapshotFile; count the first real chunk and mutate actual original file bytes or rename/replace the original fixture root. Use a nonempty file large enough to witness the intended read, explicit fired assertions, deterministic operations and a same-callback no-fault control. No timing lottery, fake reader/FileInfo/verdict, or success conditional on hook absence. Tests referencing new MaterializeAttestationTree/types are GREEN supplemental. Existing-symbol frozen cases may be authored only when they prove the stated old interface behavior and compile on base; they cannot be relabeled strict new-capability proof.

Only NEW internal/designlock/attestation_snapshot.go may additionally declare:
    var newAttestationSnapshotBudget = func() snapshotBudget {
        return snapshotBudget{maxEntries: snapshotInventoryMaxEntries, maxBytes: snapshotAggregateMaxBytes, maxDepth: snapshotInventoryMaxDepth}
    }
The exact production defaults remain 100000 entries, depth 64, 1 GiB regular-file cap and 8 GiB aggregate. Tests may temporarily lower only returned maxEntries/maxBytes/maxDepth, restoring the factory and using no parallelism; production exposes no user override. One budget spans each complete logical inventory pass including overlays, never reset per directory/row/overlay. Real small trees must exercise entry, aggregate and depth at-limit and limit-plus-one, including a combined overlay case that would incorrectly pass if budgets were reset. Use actual filesystem FileInfo and byte reads. The fixed per-file cap remains tested with an actual oversized sparse file; never fake size metadata. Boundary tests are supplemental GREEN and report the actual limits exercised: lower-budget boundary tests do not mean an actual 8 GiB tree was hashed. Generic designlock.go, external_snapshot.go, source_snapshot.go, snapshot_inventory.go and helpers remain read-only.

PRIVATE-COPY FAULT SAFETY FOR SEAMS 2/3:
Run destructive owned-copy cases in a helper test process with TMPDIR set to one unique parent-owned test directory before any acquisition. This is test-process temp isolation, not a production fault env switch. Validate the candidate copy belongs strictly beneath that directory, differs from original fixtures, and is the actual returned copy; modify only a known regular entry into a symlink targeting another owned sentinel fixture. Do not follow the symlink or touch arbitrary temp trees. Capture/verify the real cleanup error, preserved external sentinel and output/count/ledger outcome. Child exit closes retained handles; parent cleans only its unique owned tree. Matched helper-process no-fault control is mandatory. No tests run against the retained dev worktree, installed assets, shared vault or user files.

EVIDENCE STAGING AND LIMITS:
A remains mandatory existing-interface behavioral assertion failure: each of the three legacy behavioral claims falsely accepted by base, with independent unchanged/mutated cases requiring the final GV_MISSING_IMPLEMENTATION_SUBJECT contract. D remains mandatory passing base compatibility controls. B unsupported version/flags/missing callback and C mutation stages not reached after failing valid-v2 controls must be reported as INTERFACE_ABSENT / NOT YET EXERCISED, not staleness/custody proof. Final replay must reach both C legs and keep all frozen bytes unchanged. No skip-if-missing or accept-any-error.

These seams do not force an OS root-handle/filelock Close error; the canonical allows actual owned cleanup failure as the alternative lifecycle fault. Report the unforced OS close boundary honestly; do not invent a need to force every close primitive.
They also do not independently inject a late-release fault into the standalone CLI executable. Actual CLI generation/check changed-implementation/history/alias/ordinary-error and empty-stdout tests remain mandatory. RenderAttestation nil-bytes fault proof, CLI adapter source inspection and actual CLI other-error proof are compositional evidence, not a directly observed CLI late-release exit.
Before approve-red, resolve the exact canonical AC5 matrix wording "final check/close failure -> exit 1 and empty stdout": if it intends composed shared-renderer fault plus CLI wiring/process proof, record that bounded clarification; if it intends independent CLI late-release injection, the author must propose the exact additional seam for review. This authorization neither silently waives that wording nor invents a production environment switch or wider ownership. It does not block the now-authorized bounded RED authoring.

Evidence reviewed (read-only):
- pm_acceptor and codebase-memory skills read fully. pvg nd show MAC-p7jd --json read all 485 Body lines in pages; shared tracker status in_progress, hard-tdd, assignee dev-MAC-p7jd, parent MAC-ui8a.
- shasum -a 256 /tmp/machinery-attestation-contract.jODZpc/PROPOSAL.md matched approved SHA above.
- Graph list_projects/index_status/schema/search_graph (74 results, has_more false), trace_path AcquireSnapshot both directions depth 1, exact get_code_snippet, and check_index_coverage. Generation 2026-09-05T23:58:53Z, best-effort metadata_match/no_recorded_issue for eight main paths; guessed internal/designlock/scale.go reported missing and exact git tree confirmed it does not exist. Relevant bounds are in snapshot_inventory.go/designlock.go; no nonexistent source relied upon.
- Exact git show epic/MAC-ui8a for suite.go, hook.go, attest tests, CLI attest.go, private_snapshot.go, snapshot_inventory.go, source_snapshot.go, external_snapshot.go, designlock.go and snapshot_bounds_test.go. git rev-parse HEAD epic/MAC-ui8a: main 497419ab4512fcff765cd5feb27aed4c67b5608d, epic 6cb2d974ea8aea211a5974f453cef2b5802bb11e. git diff --name-only a82277a..epic/MAC-ui8a showed only subsequent unrelated baseline/docs/gates/regeneration files; exact proposed seam/test files unchanged from RED base. Main-vs-epic suite differences are accepted parent-owned Gt selection, preserved.
- Source confirms existing callback is after first actual chunk write; actual private cleanup rejects symlinks; current hook renders/checks and clears/emits before deferred Release; cargo callback is before gate evaluation and ledger callbacks are other boundaries.
- pvg notes search machinery attestation returned no additional context. No test/build run, source/test/docs edit, retained worktree change, installation, network mutation or workflow transition performed. This review authorizes future testing; it supplies no executed behavioral AC proof.
SR PM AC5 COMPOSITION CLARIFICATION CANONICALIZED
Scope: only the independently approved clarification and existing authorization-status correction. Original architect/challenger approval and exact SHA above; no new product behavior, seam, ownership, runtime, user decision, dependency or test authorization. Replaced the single ambiguous final AC5 matrix row with the approved exact row and embedded the full three-part conjunctive evidence bar. All five product AC and A/B/C/D staging are unchanged.
Both real renderer original-file mutation and approved private-copy cleanup fault require fired controls, nil bytes and their actual causes. Ordinary built CLI failure MUST reach the same real renderer-error branch, not merely Cobra argument rejection. Exact final source review includes root error handling, output tracking and deferred paths. Every part belongs to the same delivered GREEN revision; independently injected CLI late failure is explicitly unobserved. No current CLI-to-future-renderer implementation is claimed.
The prior canonical below is retained only as historical text; its architecture/authorization-pending language is superseded by the full terminal in_progress contract.

> ## USER INTENT
> Strengthen Machinery mission-critical assurance with observable fail-closed behavior and precise limits.
> 
> ## Context (Embedded)
> Assessment F3: gt.conformance-test-shape covers BUILD artifacts, while g4.pack-event-discipline covers pack. Neither binds tested/reviewed implementation. Historic Ga ancestor acceptance is valid history, not current-tree assurance. Preserve judgment-vs-mechanical distinction.
> 
> ## Ownership
> The following 13 required files are the reviewed ownership boundary; implementation and test editing remain subject to their independent phase authorizations. You are not alone: preserve other edits, especially accepted MAC-p8ce/MAC-olrx behavior; coordinate shared paths. No generic helper factoring is authorized. Optional internal/gates/attest_green_test.go is the reported fourteenth file only for independent supplemental GREEN proof.
> 
> ## Boundary Map
> PRODUCES:
> - internal/gates/attest.go -> AttestationReview; CheckAttestationsWithImplementation(design, impl string) *Gate; RenderAttestation(design, impl string, review AttestationReview) ([]byte, error); private checker/render/pending-result contract below
> - internal/gates/attest_implementation_test.go -> NEW frozen existing-interface staged A/B/C/D and real filesystem/Git regression proof
> - cmd/machinery/attest.go -> newAttestCmd() *cobra.Command generation flags/output; preserve stableAttestationHashes(paths []string) ([]string, error)
> - docs/attestation-evidence.md -> closed v2 schema, migration, scope/digest grammar and honest limits
> - internal/gates/suite.go -> Snapshot capture/RunSelected/CheckUnchanged/Release lifecycle and explicit-release wrappers, only attestation integration
> - cmd/machinery/check.go -> Gv-facing help/messages only
> - internal/gates/attest_test.go -> only independently PM-authorized exact wrong-version amendment and v2 cases
> - cmd/machinery/attest_test.go -> only independently PM-authorized exact mandatory alias-proof amendment
> - cmd/machinery/attest_implementation_test.go -> NEW frozen real isolated CLI/local-Git A/B/C/D proof
> - internal/designlock/attestation_snapshot.go -> NEW AttestationTreeSnapshot, AttestationTreeEntry and (*Lock).MaterializeAttestationTree(path string) (*AttestationTreeSnapshot, error); exact APIs below
> - internal/designlock/attestation_snapshot_test.go -> NEW real held-root/filesystem/custody proof; tests referencing new Go symbols are GREEN supplemental, not baseline RED
> - internal/hook/hook.go -> stop snapshot finalization before decision/state-clear ordering only
> - internal/hook/attestation_snapshot_test.go -> NEW real Stop/SubagentStop configured ledger/custody proof
> - internal/gates/attest_green_test.go -> OPTIONAL new-symbol GREEN supplemental tests only, report purpose and cost; never edit frozen RED
> CONSUMES:
> - Existing Machinery implementation at epic a82277a; preserve compatibility and generic snapshot semantics.
>   spec: CheckAttestations(design string) *Gate; attestationRequiredPaths(g *Gate, design, claim string) []string; stableAttestationHashes(paths []string) ([]string, error); (*Snapshot).RunSelected(impl string, sel Selection, opt RunOptions) []*Gate; SelectRunAndNote(design, impl, gateList string, opt RunOptions) (Selection, []*Gate, string, error); (*designlock.Lock).MaterializeExternalTree(path string) (*ExternalTreeSnapshot, error) remains generic-only, not strict-subject authority.
> - Existing internal/designlock private snapshot helpers/state, consumed without production edits to their files.
>   source: snapshotBudget/readSnapshotDir/copySnapshotFile/sameFingerprintFile/validateInventoryPath/newPrivateSnapshot; exact approved new-capability use and held-root protocol embedded below.
> 
> ### Story Acceptance Criteria
> 1. Implementation/test behavior claims bind a complete explicit implementation/test scope under rooted inventory and content hashes, not only BUILD or pack. Any code/test/config addition, removal, rename, content change or scope narrowing affecting the claim invalidates freshness.
> 2. Distinguish plan-only claims, current implementation review and historical milestone acceptance in closed schema, gate diagnostics and docs. Historical ancestor records remain historical; they cannot alone imply current implementation approval.
> 3. Provide explicit compatibility migration for existing attestations. Legacy design-only covers never quietly grandfather implementation assertions as fresh; users receive actionable missing-subject diagnostics.
> 4. Negative tests remove assertions after review, alter event handlers, add excluded files, change scope, alias paths/symlinks and replay stale attestations. Positive unchanged reviewed scope and harmless evidence-only commit remain usable.
> 5. Real CLI attest/check path exercises changed implementation and historical/current distinction. Hashing proves binding, not reviewer honesty or that tests executed; output never claims otherwise.
> 
> ## Testing Requirements
> - Explicit hard TDD: separate RED test author, expected behavioral assertion failure plus passing control on unmodified production, independent PM replay, frozen RED test/fixture bytes retained through GREEN. Compilation, import, timeout or unavailable infrastructure are not valid RED. Existing tests encoding the unsafe contract may be corrected during reviewed RED; GREEN cannot silently weaken them.
> - Integration tests: MANDATORY (no mocks). Exercise real process/filesystem/runtime boundaries; no stubs, no skip-if-missing. Fixture policy inputs may test parser logic but are not live remote proof.
> - go test ./internal/gates -run Attest; go test ./cmd/machinery -run Attest; CLI integration with real temp git repository and design+implementation roots.
> - Full scripts/preflight.sh ONLY at end of epic. No pushes, sync, remote mutation, installed binary/plugin/agent/skill replacement, or dev-link. Another agent uses installed Machinery in NIL; builds/tests use isolated output and homes.
> - Shipped product enforcement is standalone Machinery: never requires pvg, nd, Paivot labels/metadata or commit conventions. Paivot is local development coordination only.
> 
> ## OUT OF SCOPE
> - Other assessment areas have sibling stories; final preflight/local main merge/isolated candidate binary are final gate responsibilities. Do not omit small directly related safety fixes.
> 
> ## DIFF BUDGET
> - Supersedes original ~4-7 files/<1000 changed LOC: 13 required files, approximately 1,900–2,900 changed LOC (900–1,350 production, 850–1,350 tests, 150–200 docs/help). Optional fourteenth GREEN-supplemental test file must have its purpose and cost reported. Report actual files/LOC; overruns trigger PM investigation, not automatic rejection or weaker proof.
> 
> ## Approved architecture authority and executable clarifications
> Independent contract review APPROVED revision 2. Authority: proposal SHA256 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179, read in full and embedded below; rejected revision 1 remains historical only. This is architecture/canonical scope approval, NOT TEST-EDIT AUTHORIZED, approve-red, GREEN authorization, delivery or AC proof. Earlier proposal/hold language in append-only history is superseded as to architecture selection only. RED remains paused until independent PM names exact test-edit and seam uses.
> 
> Three reviewed executable clarifications apply to the embedded contract:
> - Repeated Snapshot.Release returns the latched final disposition without revalidation, re-closing, re-finalization or resurrecting invalid current results. Capture and Snapshot.RunSelected after release fail closed; repeated RunSelected calls before release are supported and covered, retaining all captures/pending results until that single finalization.
> - Every late Release/finalization cause and Gate error is remapped to logical caller-facing paths before exposure; no private snapshot path leaks. Add matched success/error tests through convenience wrappers and direct snapshot use.
> - Every DISPLAYED current-review report includes the exact scope-boundary limits stated below, including displayed CLI/hook reports. Silent successful hooks may remain silent; this is not a new reporting feature.
> 
> No execution authentication, reviewer-honesty claim, MAC-l7m0 dependency, Docker requirement, or new user choice is introduced. Generic designlock.go, external_snapshot.go, source_snapshot.go, snapshot_inventory.go, scale.go, portablepath, cmd/machinery/hook.go, generic snapshot callers, accept.go and accept_test.go remain read-only. New API declarations below are approved PRODUCES, not preexisting callable test seams.
> 
> ## Independent PM test-edit and RED-start handoff
> Before RED resumes, PM must explicitly authorize: (1) internal/gates/attest_test.go TestAttestationMutations wrong-version fixture 2 -> 3 plus 1-or-2 supported diagnostic and separate valid/malformed v2 cases; (2) cmd/machinery/attest_test.go TestAttestRejectsIdentityAliases os.Link setup failure t.Skipf -> t.Fatal, OR a specifically named mandatory new alias case if PM chooses to leave the existing case intact. All other existing tests remain unchanged absent a new named review.
> New frozen RED tests are confined to the named attestation implementation files, new hook/designlock test files where they can compile using existing APIs/seams, and the exact authorized amendments. New helper API unit tests that cannot compile before production belong to explicitly reported GREEN supplemental proof; they cannot replace frozen existing CLI/suite/hook acceptance tests. A/B/C/D classification below is mandatory.
> There is NO blanket permission to add new fault/callback/budget seams. Before authoring any such seam/test, provide its exact owned path, signature/callback boundary, real operation and mutation/failure, no-fault control, cleanup and A/B/C/D or supplemental classification for independent PM approval. Stage A existing-interface behavioral failure plus Stage D passing controls is the required baseline RED; B interface absence and C not-yet-reached mutations must be recorded honestly. Do not call unavailable Go symbols from frozen baseline tests.
> Focused verification after scope/test freeze: go test ./internal/gates -run Attest; go test ./cmd/machinery -run Attest; go test ./internal/designlock -run Attestation; go test ./internal/hook -run Attestation. Record all exact discovered leaf names, baseline/final outcomes and reached/not-reached mutations; choose bounded timeouts and record runtime before any broader replay. No full preflight until epic final gate.
> 
> ## Approved schema and classification
> 
> Write attestation_version: 2 for new rows. Continue to parse closed v1 for explicit migration semantics below. Reject every other version, including numeric spellings other than integer 1 or 2. Root allowed keys remain attestation_version, attestations, optional string _comment. attestations is a nonempty array. YAML duplicate keys at every depth, unknown keys, wrong node types, nulls, aliases/merge keys, and duplicate rows are errors. All optional note/_comment fields are strings, not arbitrary YAML payloads.
> 
> V2 row allowed keys: claim, kind, attestor, date, covers, implementation, note, _comment. claim/kind/attestor/date are nonempty strings; date is an actual YYYY-MM-DD date. covers is a nonempty array of closed {path, hash, optional string _comment}. cover path is a canonical portable design-relative regular-file path, never absolute, '.', '..', backslash, traversal, or a normalized alias; use portablepath.ValidateRelative. No duplicate/casefold-equivalent cover paths. Hash is exactly sha256: followed by 64 lowercase hex digits. covers may not name attestations.yaml (self-reference). Existing required design-subject coverage remains mandatory, including all BUILD packets, pack files and acceptance files relevant to the claim.
> 
> kind is exactly plan, current, or historical. There remains at most one row per claim ID, including across kinds: upgrading a plan to current replaces that claim's row; git history retains previous reviews.
> 
> Claim classification is an explicit table in code, not a prefix-derived runtime guess:
> 
> * Design-plan judgments: the six g2 IDs; the four g3 IDs; g4.zero-context. Allowed kind: plan only. Their words and diagnostics describe the design/model/BUILD plan, not observed implementation behavior. They keep their existing required design subjects and owed predicates.
> * Implementation/test judgments: gt.conformance-test-shape, g4.pack-event-discipline, g4.standin-coverage. Allowed kind: current, or plan to record intent only. current requires the complete implementation scope below AND existing required design covers. plan never counts as current implementation review. Stand-in coverage is classified here because its judgment includes actual stand-ins held to an oracle and an environment recipe; keeping only the BUILD paragraph would repeat the same defect.
> * ga.review-quality: historical only. It judges the milestone acceptance record(s), with covers over the required acceptance/*.yaml files. It has no implementation field. Ga's existing acceptance schema is unchanged; its commit fields remain the history anchors.
> 
> implementation is REQUIRED exactly when kind=current, and FORBIDDEN otherwise. Closed object keys: root, policy, entries, hash. root is the normalized slash-separated lexical relative path from the logical design root to the supplied implementation root ('.' or leading '..' components are permitted only in this locator, since sibling/parent implementation roots are normal). It must exactly equal the locator independently derived from the actual invocation; no absolute paths, backslashes, redundant components, or nonportable non-dot components. This locator grants no path-opening authority. All reads come from the invocation's held root capability. policy must equal full-root-v1. There are no user include, exclude, extension, gitignore or subtree selectors. entries is a nonempty sorted inventory (including root directory '.'); hash binds the canonical serialization below.
> 
> Every inventory entry is a closed mapping. Directory: {path, type: directory, mode}; regular file: {path, type: file, mode, size, hash}. mode is a four-character octal permission string 0000–0777; size is a nonnegative integer within limits; hash has the same fixed SHA256 spelling. File-only keys on directories are errors. '.' is allowed only for the single root directory; all other paths use portablepath.ValidateRelative. Sort by raw ASCII path bytes; reject unsorted rows, duplicates, casefold collisions, missing parents, file/descendant conflicts and wrong root shape. Do not silently normalize a submitted manifest.
> 
> Concrete schema/digest example: ARCHITECTURE.md bytes are "# Architecture\n", BUILD.md bytes are "# Build\n", both implementation files are distinct regular files with bytes "package example\n" (16 bytes), implementation root mode 0755 and file modes 0644. acceptance/M1.yaml bytes here are "milestone: 1\n" solely to illustrate Gv's historical cover hashing: that fragment is NOT valid Ga acceptance and MUST NOT be presented as a valid milestone acceptance fixture. The current scope digest below was calculated from the exact canonical grammar; this example is not a --complete fixture and does not prove a test contains assertions. Use the existing complete Ga record shape for the real Git history tests.
> 
> ```yaml
> attestation_version: 2
> attestations:
>   - claim: g2.nfr-content
>     kind: plan
>     attestor: R
>     date: '2026-09-05'
>     covers:
>       - {path: ARCHITECTURE.md, hash: 'sha256:2e7cb38229f7877e569cc05dd9c8fb71b30954ad2c9978328d69f5d377f81505'}
>   - claim: gt.conformance-test-shape
>     kind: current
>     attestor: R
>     date: '2026-09-05'
>     covers:
>       - {path: BUILD.md, hash: 'sha256:bbc290c9f84e532bd47737480381f0db3afae637d696806856d95d0a186bb619'}
>     implementation:
>       root: ../src
>       policy: full-root-v1
>       entries:
>         - {path: '.', type: directory, mode: '0755'}
>         - {path: handler.go, type: file, mode: '0644', size: 16, hash: 'sha256:e0e0431b63a883552b05817d33ba13f79262019cdefb4e5ae060c299c1de1eb8'}
>         - {path: handler_test.go, type: file, mode: '0644', size: 16, hash: 'sha256:e0e0431b63a883552b05817d33ba13f79262019cdefb4e5ae060c299c1de1eb8'}
>       hash: 'sha256:4e0ca743a34c585082fdcfb7f57ca6a47a1e3448510c3033f5dd7281a351ac2a'
>   - claim: ga.review-quality
>     kind: historical
>     attestor: R
>     date: '2026-09-05'
>     covers:
>       - {path: acceptance/M1.yaml, hash: 'sha256:a44677835fe6e178bd0d0d51faea9ccfc327ec4442bf96a4a96031392c7da249'}
> ```
> 
> Invalid examples/categories: version: '2' (wrong type); version 3 (unsupported); kind: accepted (unknown); current without implementation (GV_MISSING_IMPLEMENTATION_SUBJECT); kind: historical on gt.conformance-test-shape (GV_KIND); implementation.exclude: ['generated/**'] (GV_SCHEMA); root ../src changed to ../src/tests (GV_SCOPE_ROOT or GV_SCOPE_HASH); duplicated handler.go or ./handler.go (GV_SCOPE_PATH); omitted handler_test.go while retaining the old scope hash (GV_SCOPE_HASH), or after recomputing the manifest hash (GV_SCOPE_INVENTORY against independently enumerated root). A syntactically sound current row with changed bytes is GV_STALE_CONTENT, not schema failure or DRIFT.
> 
> ## Complete scope and deterministic freshness
> 
> The authority is the entire actual --impl root. Enumerate every directory and regular file recursively, including ignored, untracked, hidden, generated, extensionless, binary, vendor, build-output, test, and configuration files. G4 contract ignores, .gitignore, .machinery configuration and programming-language extension lists NEVER determine this inventory. They are themselves ordinary included files. Empty directories and permission changes are bound conservatively. This can be larger than the language scanner's scope; that is intentional. Dependencies/configuration outside the supplied root are outside this explicitly reported review, and Machinery must not call this whole-program or environment completeness. Users select a root containing the code/test/config they want reviewed; switching root changes the bound locator.
> 
> Exactly two reserved exclusions exist, determined by policy and invocation, never by the row:
> 
> 1. The supplied implementation root's top-level .git metadata entry, if present. It must be a real directory or regular worktree gitfile, never a symlink/special file. Its content/presence is excluded, allowing ordinary Git commits. All nested .git entries cause GV_SCOPE_UNSUPPORTED_METADATA, instead of being silently pruned. This is a deliberately conservative policy: nested-repository users must provide a source export without nested metadata. The existing generic snapshot skips .git at every depth and is insufficient for this particular rule. Top-level VCS administration, hooks and repository-local Git configuration are NOT implementation/test execution inputs covered by this review; using them as runtime source is outside this reported scope. There is no language-independent proof that arbitrary code does not read VCS metadata. Documentation and output must state this exact boundary, not claim excluded metadata can never affect behavior.
> 2. The exact logical <design>/attestations.yaml path, only when it lies within the implementation root. This is Machinery's reserved review record, not arbitrary *.yaml, an evidence directory, or a glob. Omit that entry's presence/content/mode from the inventory, with a constant exclusion descriptor included in the digest whether the file exists yet or not. Gv separately parses the present file under the closed schema. All sibling files, nested files named attestations.yaml elsewhere, acceptance YAML, logs, docs, generated tests and configuration remain included. Changing a note/attestor/date in this one record and committing it is the matched harmless evidence-only positive. This proposal does NOT promise freshness survives edits to arbitrary prose/docs or adding acceptance records; they remain conservatively bound when inside --impl.
> 
> Exclusions are reserved semantic boundaries, not authenticated assurances about how the application executes. If the user intends application code to consume Machinery evidence or Git metadata as runtime inputs, full-root-v1 does not cover that runtime and must not be described as doing so. This precise limit is preferable to claiming that hashing can discover all transitive execution dependencies. No further exclusion expansion is implied.
> 
> Before hashing compare the submitted inventory to a fresh authoritative inventory in BOTH directions. Any added, removed or renamed path is GV_SCOPE_INVENTORY; changed type/mode/size/hash is GV_STALE_CONTENT (type aliases/symlinks instead use custody categories below). Report the first differing portable path deterministically plus summary counts; do not print private temp paths or raw file content. Every current row is checked against the same fresh inventory for the invocation. Required design covers are independently checked too, so unrelated unchanged files cannot discharge a claim.
> 
> Canonical digest input is UTF-8/ASCII, exactly this header and tab/newline grammar, no YAML formatting and no timestamps/commit/inode numbers:
> 
> ```
> machinery-attestation-scope-v1\n
> root\t<logical-root-locator>\n
> policy\tfull-root-v1\n
> exclude\tvcs-root:.git\n
> exclude\tevidence:<implementation-relative-design-attestations-path-or-none>\n
> directory\t<path>\t<mode>\n
> file\t<path>\t<mode>\t<decimal-size>\tsha256:<hex>\n
> ...
> ```
> 
> The last two line forms repeat for the globally path-sorted inventory, including root '.'. Hash the exact byte concatenation with SHA256, print sha256:<lowercase hex>. Portable paths cannot contain tabs/newlines, so this encoding is unambiguous. Counts derive from entries and need no second mutable field. Digest includes both inventory and scope-policy/root/exclusion identity. The row's hash must first match its own submitted inventory, then the fresh authoritative inventory hash. Neither rewriting exclusions (unknown schema) nor removing entries (set comparison) can preserve a prior review. Changing the root requires a different digest and must also match the invocation's root. A dishonest author can calculate a new receipt; that is a NEW self-authored assertion, not proof an independent reviewer re-reviewed it.
> 
> Limits: preserve snapshot bounds, 100,000 entries, depth 64, 1 GiB per regular file, 8 GiB aggregate; stream file bytes with existing bounded readers/copy routines. Apply one budget to the full logical inventory including design overlay; do not reset budgets per directory, row or overlay. V2 YAML and each required design cover obey the existing 16 MiB designArtifactMaxBytes reader limit in internal/gates/confinement.go (exact epic source verified after graph discovery). Generation must enforce the 16 MiB serialized document bound too, and warn that merging multiple independently generated rows must fit that same document bound. If inventory YAML cannot fit, fail GV_EVIDENCE_LIMIT with the size and limit rather than truncate. No flags to relax limits in this story. Existing explicit-file hash mode retains its separate 16 MiB/file bound. Scope inventory limits alone do not guarantee a maximum-size manifest can be stored.
> 
> Reject symlink root leaves, directory/file symlinks anywhere in included scope, special files, invalid portable names, duplicate/casefold names, and replaced roots/entries during capture. The held rooted reader prevents escaping through descendants; it must never resolve a row path against the ambient original root. A no-follow check of the root leaf is required; do not invent a ban on platform ancestor aliases such as macOS /tmp.
> 
> Alias policy distinction: duplicate logical receipt paths and path-normalization aliases are invalid independently of filesystem identity. Hardlinks at two different logical paths are not inherently a byte-freshness exploit: complete immutable snapshots retain both paths and bytes, and additions/removals still differ. Nevertheless this proposal conservatively rejects same-inode regular-file pairs during attestation capture, matching the existing explicit-file CLI policy and making alias fixtures deterministic. This is the approved policy, not a claim that copies lose byte safety. Cross-root hardlinks outside the enumerated root cannot be exhaustively discovered; held descriptor/identity/content revalidation handles observed mutations. No inode identity persists across invocations, because checkout portability must remain possible.
> 
> ## Root custody and approved exact API contract
> 
> Existing signatures remain source-compatible:
> 
> ```go
> func CheckAttestations(design string) *Gate
> func AttestationClaimIDs() []string
> func ContentHash(path string) (string, error)
> func stableAttestationHashes(paths []string) ([]string, error)
> func (s *Snapshot) RunSelected(impl string, sel Selection, opt RunOptions) []*Gate
> func SelectRunAndNote(design, impl, gateList string, opt RunOptions) (Selection, []*Gate, string, error)
> ```
> 
> CheckAttestations(design) remains a design-only compatibility wrapper calling checkAttestationsInSnapshot(design, nil), with the existing rooted readDesignFile behavior and diagnostic wording preserved. It does not acquire an implementation root or newly claim whole-operation snapshot guarantees for standalone legacy callers. Plan/history remain usable, but any current row produces GV_IMPL_REQUIRED and legacy behavioral rows produce GV_MISSING_IMPLEMENTATION_SUBJECT. It does not infer impl from cwd, Git root, BUILD text or evidence. Do not change it variadically: a distinct API keeps absent-root semantics visible. Production suite callers always pass the held design snapshot to the internal checker, and the new WithImplementation convenience API acquires the full snapshot itself.
> 
> New gates APIs/types (owned by attest.go, private subject fields protect capture construction):
> 
> ```go
> type AttestationReview struct { Claim, Kind, Attestor, Date, Note string }
> func CheckAttestationsWithImplementation(design, impl string) *Gate
> func RenderAttestation(design, impl string, review AttestationReview) ([]byte, error)
> type attestationSubject struct { /* private captured manifest, root locator, policy */ }
> func checkAttestationsInSnapshot(design string, subject *attestationSubject) *Gate
> func (s *Snapshot) captureAttestationSubject(impl string) (*attestationSubject, *designlock.AttestationTreeSnapshot, error)
> func renderAttestationInSnapshot(design string, subject *attestationSubject, review AttestationReview) ([]byte, error)
> ```
> 
> CheckAttestationsWithImplementation acquires one Snapshot, captures through the new capability, checks, and explicitly releases before returning its Gate. It folds custody/close/release failures into Gate.Errs and returns no published current-review count on failure. RenderAttestation does the same but returns buffered YAML only after capture/revalidation/close/release all succeed. It generates hashes and exact manifest; it does not execute tests or attest reviewer identity. Direct convenience caller failures use the same Gv categories, never optimistic success on absent impl.
> 
> Suite wiring: when Gv is actually applicable and impl is nonempty, use captureAttestationSubject for the existing implementation preparation; pass the resulting subject to checkAttestationsInSnapshot. Other gates receive an immutable implementation Path with the SAME inclusion/topology as the existing MaterializeExternalTree result. Otherwise keep existing preparation. Add one private RunOptions field `attestationSubject *attestationSubject`; no exported user-settable snapshot paths. At Gv call use checkAttestationsInSnapshot(design, opt.attestationSubject). Keep current selection predicates, MAC-p8ce changes, cargo authority handling and remapping. Add a private suite-local `implementationSnapshot` interface with Path() string, Logical() string and Close() error so existing generic and new attestation captures can share the local stable variable; it is not a new public API. Generic impl and cargo cleanup stays in RunSelected. Strict attestation captures are instead registered on Snapshot and retained until Release; error exits either close an unregistered capture immediately or let the registered owner close it, never both. A current row without impl is diagnosed by Gv after parsing, not by a global ban on --gate gv without --impl.
> 
> Approved narrowly scoped shared capability (new internal/designlock/attestation_snapshot.go; preserve generic MaterializeExternalTree callers):
> 
> ```go
> type AttestationTreeEntry struct {
>     Path string
>     Directory bool
>     Mode uint32
>     Size int64
>     SHA256 [32]byte
> }
> type AttestationTreeSnapshot struct { /* private tree, full inventory, held source roots */ }
> func (l *Lock) MaterializeAttestationTree(path string) (*AttestationTreeSnapshot, error)
> func (s *AttestationTreeSnapshot) Path() string
> func (s *AttestationTreeSnapshot) Logical() string
> func (s *AttestationTreeSnapshot) Entries() []AttestationTreeEntry
> func (s *AttestationTreeSnapshot) CheckUnchanged() error
> func (s *AttestationTreeSnapshot) Close() error
> ```
> 
> Entries returns a copy, never mutable internal authority. The capability captures a complete logical inventory from held no-follow ORIGINAL source root(s), applies the top-level/nested .git policy, detects hardlink pairs before copies erase alias identities, and proves inventory bytes equal the returned stable tree plus retained design overlay. Evidence filtering/digest is in gates; designlock supplies the unfiltered observed entries (apart from VCS metadata) so unrelated consumers do not learn the attestation schema. It retains the original root handles until final CheckUnchanged and Close. The existing design SourceRoot is a private copy, NOT an original held descriptor; revision 1 did not make that distinction sufficiently explicit.
> 
> MaterializeAttestationTree acquisition owns this exact protocol: resolve logical topology lexically; open original authority root(s) below; compare pre-open Lstat, held-root Stat and l.rootInfo for the design; enumerate design THROUGH ITS HELD ORIGINAL ROOT using the existing design fingerprint policy and compare the exact result to l.snapshot AND retained SourceRoot bytes (root identity/mode checked separately). This bridges the earlier private-copy acquisition to the newly retained original authority, failing if they differ; it does not claim the new original handle existed at AcquireReader time. Capture the strict full implementation inventory and copy through held root operations, with witnessed pre/open/read/post identity/content checks and a second complete held-root pass. Prove the copy plus retained design overlay equals the full inventory before returning. Reject nested .git discovered in the strict implementation scope even when the generic design copy omitted it. Required design covers are read only from the already verified private design copy.
> 
> | Logical topology | Original handles retained by capability | Stable implementation Path and final authoritative checks |
> |---|---|---|
> | Disjoint design and impl | Independently open/retain original design root and original impl root once; compare design handle to l.rootInfo | Private implementation copy; held design fingerprint must equal l.snapshot/retained generation; held strict impl inventory must equal captured inventory |
> | Impl is ancestor of design | Open original impl root; obtain design root by held impl.OpenRoot(design-relative-path), validate all intermediate real directories and equality with l.rootInfo; retain both handles | Generic-compatible impl copy omits design subtree; full Entries includes that subtree from original inventory, proven equal to retained design copy; final held impl pass checks ALL scope including design; held design pass also preserves original design generation |
> | Impl equals design | One original root handle, shared ownership with exactly one close | Path is retained SourceRoot; strict full inventory and generic-policy design snapshot comparison both run through original handle; reject nested .git in strict scope |
> | Impl is inside design | Open original design root; obtain impl root by held design.OpenRoot(impl-relative-path), validate intervening real directories; retain both | Path is retained SourceRoot/subpath; final strict impl inventory uses retained impl handle and design-generation check uses retained design handle; root-name/ancestor identity witnesses catch replacement of subpath |
> 
> Logical relative strings decide topology but do not grant read authority. No EvalSymlinks-based following inside these roots. Merge full inventory/path/case/identity checks and budgets before evidence filtering. A design subtree containing code/test/config cannot disappear. Original handles capture identity only for the observation interval; inode identities are not persisted across commands. In-design/equal impl with a private-source path supplied by an internal caller must resolve to the known logical root through the owning Lock rather than treat a private temporary directory as a new logical implementation root; reject an unrecognized private path.
> 
> AttestationTreeSnapshot.CheckUnchanged owns the final primary attestation proof: enumerate/rehash through its retained original design/impl handles under the above policies; compare with captured generation, full inventory, bytes and root/descendant identity witnesses; check that logical root names still identify the held roots using Lstat (metadata checks, no byte-opening authority); fail on any difference. Closing the capability aggregates root-handle and private-copy close errors. Even the reserved evidence file must not change DURING a single observation, because the held design-generation check binds the parsed row; editing it BETWEEN successful invocations remains permitted by the scope digest exclusion.
> 
> Approved precise shared integration choice: implement the new held-root traversal/copy and private state in internal/designlock/attestation_snapshot.go, in the SAME package, using existing snapshotBudget/readSnapshotDir/copySnapshotFile/sameFingerprintFile/validateInventoryPath/newPrivateSnapshot helpers directly. Do NOT call MaterializeExternalTree or TrackExternalTree to establish the strict subject, because their path reopen/exclusion policy is different. A new private helper `captureAttestationRoot(root *os.Root, logical string, policy attestationInventoryPolicy, copyTo string) ([]AttestationTreeEntry, error)` performs the bounded held-root work; private policy distinguishes strict full implementation versus the exact generic design comparison. copyTo is empty for revalidation. Preserve each helper's current limits and join all close errors. Prove equality using the existing fingerprint entry spelling when comparing to l.snapshot; remove the synthetic '.' row only for that existing map comparison, checking root identity/mode separately.
> 
> This choice requires ZERO production edits to designlock.go, external_snapshot.go or source_snapshot.go. They remain read-only consumers/providers of existing private helpers/state. No vague helper factoring is authorized: if implementation cannot use those helpers without changing them, return the exact necessary signature/hunks for a further scope review. The new capability has its own held-root walk precisely so generic capture semantics stay unchanged. Its additional duplication/LOC is included in the revised estimate.
> 
> Correction to the no-reopen assurance: Gv's manifest/content comparisons and final primary freshness proof use retained original capabilities and verified private copies, never a fresh ambient path as subject authority. Existing Lock.CheckUnchanged (designlock.go2101 onward) and checkExternalUnchanged DO reopen ambient paths via fingerprint/fingerprintRoot; they remain supplementary fail-closed suite guards. A supplementary failure invalidates the result, and a supplementary pass can NEVER substitute for, override or repair a failed held-capability check. Existing workspace/Cargo readers may also keep their own established custody behavior. This proposal does NOT claim every operation in the entire suite avoids ambient reopens. Filesystem name/metadata checks are allowed to detect replacement and cannot supply replacement subject bytes. These checks bind a stable observation, not authenticated execution, continuous isolation after the final read, or an absolute guarantee against every undetectable ABA race on every platform.
> 
> ## Finalization and publishable current-review findings
> 
> Add private Snapshot fields `attestationCaptures []*designlock.AttestationTreeSnapshot`, `attestationPending []*pendingAttestationResult`, `attestationCustodyErr error`, `attestationFinalized bool`. Define the private pending result in attest.go as its Gate pointer, successful current-row count, scope/file count(s) and pending-note identity. captureAttestationSubject sets a private subject field `pendingResults *[]*pendingAttestationResult` to `&s.attestationPending`; the internal checker appends through that pointer, so suite and convenience callers use the same owner with no post-call registration gap. A current subject without its owner/pending sink is an internal custody error, never an immediate publish path. This sidecar avoids modifying Gate/gates.go. The internal checker records semantic errors immediately, but stores successful implementation/current counters and success notes in the sidecar, NOT public Counts or Notes. Public Gate includes only `current implementation review pending final snapshot release` until finalized. Design-plan/history/generic row bookkeeping keeps existing semantics and must not be labeled current implementation.
> 
> Exact lifecycle:
> 
> 1. Snapshot.RunSelected captures/registers the strict capability, runs gates, remaps logical paths, calls its strict CheckUnchanged AND existing Lock.CheckUnchanged, and performs existing generic/Cargo cleanup. Any of these custody/cleanup errors is accumulated in attestationCustodyErr, appended as G0/Gv failure, and marks every pending current result invalid. Strict capabilities remain held until Snapshot.Release. RunSelected returns provisional Gv findings with ZERO public current-review/file/scope success counters even when checks have passed so far.
> 2. Snapshot.Release performs one final strict CheckUnchanged for each retained capability and, when attestation captures exist, one final supplementary Lock.CheckUnchanged before any design-copy cleanup. It then closes each capability, closes workspace and releases Lock, joining ALL errors with accumulated attestationCustodyErr. Do not short-circuit cleanup after the first error. Snapshot.CheckUnchanged itself must accumulate any returned supplementary error in attestationCustodyErr when an attestation capture exists, so a caller cannot observe a failure and have a subsequent restoration erase it. Finalize pending results only after all operations return. Success removes the pending note and publishes stored current counters via Gate.Count plus its scope-boundary note. Any error removes the pending note, publishes NO current counters/positive current notes, and appends `GV_SCOPE_CUSTODY: current review not established; final snapshot validation or release failed` with the cause. Store the joined final error; idempotent Release returns that same disposition, finalizes once and never later resurrects invalid findings.
> 3. SelectRunAndNote changes its defer-only successful path to derive VersionSkewNote while the snapshot is held, explicitly call Release, then return finalized run/note/error. Early selection errors still release safely; no success result escapes on release failure. The package RunSelected convenience wrapper retains its signature and returns only after Release, which has already finalized/invalidated its gate pointers; it may retain the additional G0 failure for consistency. New WithImplementation and RenderAttestation wrappers use the same finalization sequence, with Render returning nil bytes on failure. Existing Snapshot.RunSelected direct callers must Release before reporting positive current assurance.
> 
> Verified hook consequence: internal/hook/hook.go stop acquires the snapshot around 1273 and defers Release at 1277; it emits Gv into an internal text buffer at 1343, but its eventual non-block stopOut JSON is written to the actual writer and clearCheckedState can run BEFORE deferred Release. The green default can also clear state then return nil before release fails. cmd/machinery/hook.go newHookCmd passes output.stdout directly to hook.Run under the installation lock; there is no outer JSON buffer that reliably retracts an already emitted non-block result. A later returned Go error does not prove every host cancels that result. This is a source-supported ordering risk, not a demonstrated host exploit. Suppressing counters alone is insufficient.
> 
> The approved scope includes internal/hook/hook.go, only stop's snapshot-finalization/decision order, and new internal/hook/attestation_snapshot_test.go. Keep the returned []*Gate instead of immediately emitting it; collect armed/ratchet and wave-sentinel facts while sourceDesignDir exists; complete snapshot.CheckUnchanged and explicitly snapshot.Release BEFORE rendering/tallying finalized gates, deciding shouldBlock/wave deferral, calling clearCheckedState or emitting any non-blocking stop result. Release is idempotent so the existing deferred cleanup can remain for early exits; early errors may emit an explicit block before cleanup because they authorize nothing. On strict capability, supplementary check, cleanup or Release failure, immediately emit stopOut{Decision: "block", Reason: <custody cause>} and retain checked-state ledger, regardless of cfg.Strict, ratchet presence or wave deferral. The empty-selected-gate branch likewise must release successfully before clearing state or returning a non-blocking result. Existing semantic error/warning, import-arming and wave-defer policy applies only AFTER successful custody finalization; do not change that product policy in this story.
> 
> Hook proof: run actual Stop and SubagentStop events against a real local configured design/impl/ledger; stable finalized current review permits the existing configured outcome and clears state; mutate a subject between gate evaluation and finalization, or fail owned snapshot cleanup, and assert exactly one block JSON, no earlier non-block output, retained touched-state ledger and no public current counters. Exercise strict=false and wave-open so those policies cannot waive custody failure. Also retain existing semantic-warning/wave controls after successful finalization. The new tests may use a specifically PM-reviewed callback around the hook's finalization boundary to mutate actual files; they must not mock the gate verdict or accept a bare process error as proof of blocking. cmd/machinery/hook.go and the outer install lock remain read-only; this proposal promises finalized DESIGN/IMPLEMENTATION snapshot custody before a stop authorization, not that unrelated outer host/installation operations are newly transactional.
> 
> Required finalization tests: expose no count before release; publish it after successful release; mutate a real original file after RunSelected but before Release and require suppression; replace/rename an original root and require suppression through the retained/name witness; trigger final supplementary ambient failure with held data unchanged and require suppression; observe Snapshot.CheckUnchanged failure, restore bytes, then Release and require the failure remains latched; trigger actual owned private-copy/workspace cleanup or root/filelock close failure using only explicitly reviewed local error-injection seams, require suppression and error propagation. Pair every case with successful cleanup/control. Test SelectRunAndNote, package RunSelected and WithImplementation return paths; generation must return nil/empty stdout on late failure. Do not count an earlier Gate counter plus later G0 error as satisfying suppression. OS close-error forcing may need a narrow deterministic test seam in the NEW capability or suite owned code; PM must approve its exact use, and it must inject the failing lifecycle operation rather than fake a successful filesystem snapshot. No further shared-helper edits beyond the explicitly named suite/hook boundary changes are proposed.
> 
> ## CLI, migration and current/history outcomes
> 
> New generation syntax, alongside unchanged existing commands:
> 
> ```
> machinery attest --design design --claim gt.conformance-test-shape --kind current --impl . --attestor R --date 2026-09-05 [--note text]
> machinery attest --design design --claim g2.nfr-content --kind plan --attestor R --date 2026-09-05
> machinery attest --design design --claim ga.review-quality --kind historical --attestor R --date 2026-09-05
> machinery attest design/ARCHITECTURE.md design/BUILD.md
> machinery attest --claims
> ```
> 
> Generation requires all named fields (including explicit date; no wall-clock nondeterminism), exactly one claim, no positional files. --impl is required for current and forbidden for plan/historical. --claims alone preserves byte-for-byte ID/order output. --claims with existing positional files retains its existing precedence; reject combination with any new generation flag. File-hash mode retains its exact output and custody tests. Generation emits one complete v2 YAML document containing the generated row and all required design covers. It does NOT overwrite or merge design/attestations.yaml; docs say review subjects, then merge the generated row into that file. If --claim has no actual required design subject, generation fails rather than manufacture a cover. No --exclude or --include flags exist.
> 
> All discovery, validation, hashing, final unchanged checks and close/release errors must finish before first generation stdout byte. On such failure stdout is empty, stderr has the category/remedy, exit 1. Successful generation writes one buffered document; an output write failure can physically truncate stdout and must return nonzero—do not promise that an arbitrary pipe provides atomic writes. Display these exact limits in generation stderr/help and finalized Gv scope notes (therefore CLI and hook output), as well as docs: 'Hashes bind the observed files and scope; they do not prove tests ran, reviewer identity, or judgment correctness. Top-level Git administration and the exact Machinery attestation record are excluded; applications that use them as runtime inputs are outside this review boundary.' The generated row's caller-supplied attestor is attribution text, not authentication.
> 
> V1 migration: keep parsing only the original v1 keys. Six g2/four g3/g4.zero-context rows retain their existing design-cover semantics as implicit plan judgments (emit an informational migration note, not a new coverage warning). ga.review-quality becomes explicitly reported legacy historical judgment over acceptance evidence; it grants no current approval. V1 gt.conformance-test-shape/g4.pack-event-discipline/g4.standin-coverage ALWAYS produce GV_MISSING_IMPLEMENTATION_SUBJECT, even with matching BUILD/pack hashes and even without --impl. Remedy names the claim and generation command, tells the user to review the complete implementation/test scope and write v2 kind=current, or explicitly recast the statement as v2 kind=plan. Never auto-add hashes or silently infer a current review from an old row.
> 
> Ordinary Gv preserves incremental adoption: owed claims with no row warn; absent evidence after owed artifact activation is still an error. A v2 behavioral plan row reports 'plan only; current implementation review missing' as a coverage warning and cannot discharge the corresponding implementation obligation. All malformed/stale/current-without-impl/legacy-behavior rows error in ordinary and complete modes. --warnings-as-errors and existing --complete warning promotion make missing current reviews blocking; --complete already requires --impl globally. Update check help/error text to name Gv alongside G4/Gt where relevant. Counts distinguish plan judgments, current implementation reviews and historical review records; retain the existing generic attested-claims count for compatibility. Never count a failed or plan/history row as current implementation review.
> 
> Ga alone retains its exact existing repository ancestry acceptance semantics, including exported identity and missing-history behavior. Gv historical rows validate the recorded acceptance-file covers and label their judgment historical; Gv alone does not pretend to have executed Ga's ancestry validation. With --gate ga,gv, a historical accepted ancestor and stale current implementation can coexist: Ga passes its history check; Gv fails current freshness. Changing --commit to the old reviewed ancestor cannot repair Gv, which always observes supplied current root. Absence of current review cannot be rescued by ga.review-quality. --complete needs its separate actual current implementation rows as well as valid history. A later Git commit updating only the reserved attestation record keeps the same implementation digest and Ga's ancestor check valid.
> 
> ## Five-AC RED/GREEN test contract
> 
> All new tests use actual files, directories and local Git repositories; CLI process tests build an isolated binary under t.TempDir, never replace installed machinery or run installation hooks. No Docker/Java/Node product/runtime dependency, network, mocks, timing lotteries or skip-if-missing. Fixture setup failure is not RED. Frozen test/fixture bytes and exact existing-test amendments require PM authorization before any author edits. The old production has no v2 interface, so a v2 passing freshness control CANNOT be required on a82277a. Tests must be partitioned and reported by the following stage contract, not collapsed into one universal baseline-positive rule.
> 
> | Frozen test category | Exact a82277a outcome | Exact final implementation outcome | Evidence it supplies |
> |---|---|---|---|
> | A — existing-interface behavioral RED: three legacy behavioral claim fixtures, using existing CheckAttestations/RunSelected/CLI; independently evaluate unchanged and assertion/handler-mutated trees; final assertions require GV_MISSING_IMPLEMENTATION_SUBJECT in BOTH cases | Compiles, setup works, old implementation accepts each legacy design-only row instead of issuing required missing-subject error; assertion fails with that exact observed error absence. Both unchanged and mutated legacy cases are negative cases under the new contract | Both unchanged and mutated legacy cases are rejected with GV_MISSING_IMPLEMENTATION_SUBJECT and never counted current | Actual unsafe legacy acceptance reproduced through existing interfaces. This is the primary required behavioral RED, not v2 sensitivity. The old acceptance is a recorded bug observation, NEVER a frozen passing control |
> | B — future-interface acceptance: complete v2 document parsing, generation flags/output, kind rules, new categories, finalize-visible counts; invoke only existing CLI/process or existing public suite symbols so baseline compilation succeeds | Fails specifically because version 2/flags/expected interface behavior is absent; report unsupported-version/unknown-flag output verbatim as INTERFACE_ABSENT, not successful fail-closed freshness or qualifying behavioral RED | Valid generation/schema/current result succeeds, invalid categories produce their exact specified errors, fully released wrapper publishes counters | Frozen desired interface contract. No claim that a baseline unknown flag proves staleness detection |
> | C — paired sensitivity after interface exists: for each mutation in matrix below, generate or load frozen independently calculated v2 receipt, verify unchanged actual scope first, then apply exactly one mutation and assert its category/path and zero published current count | First unchanged v2 control fails with known missing-interface behavior; mutation stage is NOT credited as executed freshness evidence. Freeze the entire test without skip/feature probe/conditional success; test remains failing on base | Unchanged control passes with finalized current count, mutation returns the specified error and no current success; evidence-only commit control passes on both invocations | Actual addition/removal/content/scope/alias/replay sensitivity, credited ONLY after both legs execute against implemented interface |
> | D — baseline compatibility controls (separate mandatory frozen controls): existing six-g2 v1 plan fixture; valid old filehash/--claims; ordinary partial design-plan coverage; existing local Ga ancestor control | Passes, with old exact outputs/counts where preserved | Still passes with specified compatibility semantics; no added warning on legacy plan migration | Meaningful passing baseline controls accompanying A. These are explicitly compatibility controls, not fabricated valid current implementation reviews |
> 
> A tests assert the final contract from the beginning; they do not first assert legacy success and later flip it in GREEN. Separate unchanged/mutated legacy subtests make both negative outcomes observable even if one fails. B/C tests must compile against base and use existing callable seams: a frozen Go test referencing a nonexistent new exported API is forbidden, because compilation failure is not RED. New API unit tests requiring new symbols may be supplemental GREEN tests with explicit provenance; they cannot replace frozen existing-interface/process acceptance tests. Any B/C mutation not reached because its unchanged leg failed is reported NOT YET EXERCISED, never passed. Do not add skip-if-feature-missing, accept-any-error, or baseline-vs-GREEN branches to make this staging look green.
> 
> The independent PM base replay must observe A's exact legacy false-acceptance assertion failure plus D's passing compatibility controls, and classify B/C failures honestly. The final PM replay must observe A rejection passing, B interface acceptance passing, C BOTH legs passing, and D compatibility still passing, with frozen bytes unchanged. This satisfies hard TDD without pretending the old product could generate a valid v2 scope or requiring a now-unsafe legacy current assertion to remain valid.
> 
> The following matrix specifies FINAL behavior for B/C/A as indicated above. Its passing-current controls are required when the interface exists; they are not asserted to pass on a82277a.
> 
> | AC | Real passing control | Single challenge and exact expected observation |
> |---|---|---|
> | 1 | Full source/test/config tree, valid generated current receipt; Gv current-review count is 1 | Remove assertion from test while keeping BUILD/oracle citation: GV_STALE_CONTENT for test path, current count 0 |
> | 1 | Same generated tree | Alter event handler while pack unchanged: GV_STALE_CONTENT for handler path |
> | 1 | Same generated tree | Add/delete/rename source, test, config, extensionless and generated file, one case each: GV_SCOPE_INVENTORY with added/removed path |
> | 1 | Ignored directory/file exists and is explicitly inventoried | Add ignored/untracked handler; change .gitignore/contract ignore; both inventory/config changes invalidate. No matching source extension is required |
> | 1,4 | Complete submitted entries and root locator | Remove one entry with old hash -> GV_SCOPE_HASH; recompute submitted hash -> GV_SCOPE_INVENTORY; add exclude field -> GV_SCHEMA; narrow --impl or row.root -> GV_SCOPE_ROOT/GV_SCOPE_HASH; unchanged subset cannot satisfy old full-root manifest |
> | 1,4 | impl contains design, with design/helpers/test.go captured in logical overlay | Change/add design subtree code -> GV_STALE_CONTENT/GV_SCOPE_INVENTORY, proving generic snapshot omission cannot hide it |
> | 2 | v2 plan row on design claim, v2 current on behavior, v2 historical on ga | Wrong kind/implementation-field combination -> GV_KIND/GV_SCHEMA; behavioral plan produces missing-current warning and never current count |
> | 2,5 | Real Git commit A with valid M1 acceptance and later evidence commit B; actual CLI --gate ga,gv --impl root is green | Change source at C, keep M1 ancestor A and replay receipt B: Ga history remains checked; Gv GV_STALE_CONTENT; explicit --commit A still cannot make Gv current |
> | 3 | Legacy six-g2 fixture remains accepted as plan with same existing counts and no added warnings | Legacy gt/pack/stand-in design-only row -> GV_MISSING_IMPLEMENTATION_SUBJECT naming root review/migration remedy, with/without --impl |
> | 3 | Explicitly migrated v2 current row from actual scope | Legacy row relabeled kind without v2 version fails schema; v2 current with empty/missing subject errors; v2 plan conversion is explicit and incomplete |
> | 4 | Unchanged scope checked repeatedly, including new local commit modifying only design/attestations.yaml note/date | Remains fresh with same scope hash, no HEAD coupling; modify adjacent evidence/handler.go or a different attestations.yaml -> inventory/content failure |
> | 4 | Distinct portable file paths, actual regular files | Duplicate receipt path, ./ alias, casefold alias -> GV_SCOPE_PATH; leaf/directory/root symlink -> GV_SCOPE_CUSTODY; real hardlink pair -> GV_SCOPE_ALIAS under the approved conservative policy |
> | 4 | Top-level .git is ordinary repository metadata, with commits changing it | nested src/.git/handler.go -> GV_SCOPE_UNSUPPORTED_METADATA; arbitrary .ignored/handler.go is included and addition fails freshness; generated/tests assertions remain included |
> | 4 | Stable real held root/copy using existing approved fingerprint/copy callback seams | File/root replacement, file mutation between reads, design-overlay mutation -> GV_SCOPE_CUSTODY or underlying G0-snapshot custody error; stdout empty on generation, no current success counter |
> | 4 | Bounded small regular tree | FIFO/special entry -> custody failure before open; limit+1 entries/depth/size/aggregate -> named limit failure, no partial inventory; use existing reviewed lowered-budget helper seams where actual 8 GiB creation is impractical, not mocked files |
> | 5 | Spawn isolated real binary; --attest generation -> write row -> check --gate gv --impl root returns 0 | Change test/handler and rerun returns 1 with Gv stale category; without --impl current receipt gives GV_IMPL_REQUIRED; --claims and old filehash output remain exact |
> | 5 | Complete-mode fixture valid under all other gates | Missing behavioral current row is the sole warning promoted to blocking; ordinary corresponding plan case warns only; no unrelated missing phase artifact accepted as RED |
> | 5 | Valid generation, all files stable; buffered stdout full v2 document | Later missing/unreadable input, alias/custody failure, final check/close failure -> exit 1 and empty stdout; output sink failure -> nonzero with possible physical partial write explicitly permitted |
> 
> Parser tests separately assert duplicate keys, unknown fields, wrong types, enum, version, path grammar, hash format, conditional fields, ordering and exact inventory self-hash. They are category B, not substitutes for category C stale-tree filesystem/CLI proof. Categories A and D provide the independently replayed base behavioral failure and compatibility control. Record each command, expected failure class, reached/not-reached mutation stage, and final result separately. A single aggregate nonzero go test or CLI exit is insufficient evidence.
> 
> ## Approved ownership, pending exact test amendments and cost
> 
> Retain declared ownership: internal/gates/attest.go; internal/gates/attest_implementation_test.go (new); cmd/machinery/attest.go; docs/attestation-evidence.md. Approved directly related additions: internal/gates/suite.go only Gv capture/private option/wiring; cmd/machinery/check.go only Gv-facing help/messages; internal/gates/attest_test.go; cmd/machinery/attest_test.go; cmd/machinery/attest_implementation_test.go (new real process/local-Git tests). Optional internal/gates/attest_green_test.go only if independent GREEN supplemental proof needs a separate file, not to change frozen RED. Historical/Ga tests can live in the attestation implementation tests without editing accept.go or accept_test.go.
> 
> Approved concrete scope expansion: internal/designlock/attestation_snapshot.go (new capability) and internal/designlock/attestation_snapshot_test.go (new real FS/custody tests); internal/hook/hook.go only stop's finalization-before-decision/state-clear ordering; internal/hook/attestation_snapshot_test.go new real-event/ledger tests. The suite.go ownership includes capture lifecycle, pending-result finalization in Release, and explicit-release convenience wrappers described above. designlock.go/external_snapshot.go/scale.go/source_snapshot.go/snapshot_inventory.go/portablepath/cmd/machinery/hook.go and generic snapshot callers remain read-only. No unspecified helper factoring is authorized. No new ownership is implied for MAC-olrx/MAC-p8ce files or execution/container policy.
> 
> Exact known legacy amendments requiring PM authorization:
> 
> 1. internal/gates/attest_test.go TestAttestationMutations, case "wrong version" around line 176: change input version 2 to 3 and expected supported-version diagnostic to 1-or-2; add separate accepted-v2 and malformed-v2 cases. Do not merely delete the assertion.
> 2. cmd/machinery/attest_test.go TestAttestRejectsIdentityAliases: existing os.Link failure currently t.Skipf; change required local native test setup failure to t.Fatal, or add a separately named mandatory alias fixture and retain compatibility test only if PM explicitly prefers it. New required alias proof may not skip.
> 3. Keep existing six-g2 clean/count/coverage-warning tests and explicit-file custody tests unchanged unless the reviewed implementation demonstrates another exact necessary amendment. Existing Gv coverage tests retain partial design-plan adoption semantics. Exact epic text search of attestation_version/attestEvidence/attestRowFor in internal/gates and cmd/machinery, plus the three behavioral claim IDs, found no additional legacy behavioral current-success fixture in that bounded scope. Other hits are cmd/machinery/repository_contract_test.go's role-document vocabulary assertion and internal/gates/failclosed_io_test.go TestAttestationPackTraversalErrorIsBlocking; neither changes. Preserve internal/gates/determinism_hardening_test.go TestAttestationRejectsSymlinkReferent and TestAttestationClaimMustCoverItsSubject unchanged, including their legacy design-only rooted error behavior. If RED discovery finds another exact affected test elsewhere, request its named amendment; do not weaken it in GREEN.
> 
> Revised realistic forecast: 13 required files (the original 9 gates/CLI/docs/test paths, 2 new designlock paths, 2 hook paths), approximately 1,900–2,900 changed LOC (900–1,350 production including explicit held-root traversal/lifecycle, 850–1,350 tests, 150–200 docs/help). An optional supplemental GREEN test file would be a fourteenth, with its evidence purpose reported. A substantial full CLI complete-mode fixture could add cost; re-use existing real valid fixtures without rewriting their proof. No designlock.go/external_snapshot.go factoring is included. The independently reviewed 13-file/1,900–2,900 LOC forecast supersedes the original 4–7 files/<1000 estimate; do not omit custody finalization, hook decision safety or real CLI controls to meet the old estimate. Final report must give actual files/LOC and ownership. No preflight now; target gates/CLI plus new designlock/hook native tests only; full preflight remains epic-final responsibility.
> 
> 
> ## MANDATORY SKILLS
> - developer; codebase-memory; pm_acceptor.
> 
> ## Delivery Requirements
> Append exact RED/GREEN SHAs, commands/output, ownership inventory and per-AC proof. Use pvg story deliver; PM accepts separately. Never invoke pushing pvg story merge.
> 
> ## nd_contract
> status: new
> 
> ### evidence
> - Created 2026-09-05 from assessment and source-verified interfaces.
> 
> ### proof
> - [ ] AC #1: independently verified
> - [ ] AC #2: independently verified
> - [ ] AC #3: independently verified
> - [ ] AC #4: independently verified
> - [ ] AC #5: independently verified
> 
POST-FREEZE PM TEST-EDIT AUTHORIZATION AND COST REVIEW — MAC-p7jd — 2026-09-06

Disposition: the two reported defects are authored fixture SETUP errors. Exact repairs below are authorized BEFORE any edit to frozen RED commit 7ec5d609acc1597ee2c0ddbf5401b92f834d5ab3. This is neither RED approval nor GREEN acceptance, delivery or rejection. Keep in_progress / hard-tdd / assignee dev-MAC-p7jd and retain the healthy worktree/claim. All five canonical AC, R2 architecture, prior exact seam restrictions and approved AC5 conjunctive composition remain unchanged.

TEST-EDIT AUTHORIZED: internal/hook/attestation_snapshot_test.go — ONLY hookReviewFixture setup:
- Immediately before the existing Config literal, add: enabled := true
- Change that literal from Config{Design: "design", Impl: "src", Gates: "gv", Strict: scenario.Strict} to Config{Design: "design", Impl: "src", Gates: "gv", Hooks: &enabled, Strict: scenario.Strict}.
- Immediately after the existing writeFile of ConfigName, add:
    if _, ok, warning := Load(root); !ok || warning != "" {
        t.Fatalf("hook fixture configuration invalid: ok=%t warning=%s", ok, warning)
    }
This check must occur before appendState arms the obligation. It calls the real existing Load and must fail on invalid/disabled setup; it must not discard warnings, change closed parsing, accept hooks:null, or bypass Run. The new enabled pointer merely serializes the intended hooks:true value. Preserve every hook event/scenario, callback signature/fired check, mutation, count/output assertion, ledger assertion, child process/TMPDIR guard, sentinel, timeout and other byte outside these exact setup hunks and their necessary gofmt formatting.
Frozen pre-edit file SHA256: c3f595d56538b97f47e9f8faa66e731fb371cba0d8256f0ddfb38f9ac421f021.

Source cause: Config.Hooks is *bool with json:"hooks" and no omitempty; the original nil pointer serializes as null. decodeConfig routes hooks through decodeConfigBool and rejects null. Load returns the configuration warning before Run can acquire the intended snapshot. Original hook.jsonl contains "config key \"hooks\" must be a boolean"; all 27 new failing leaves are SETUP, including those whose existing assertion text says B INTERFACE_ABSENT. That printed label does not override the actual earlier config failure. No old hook failure may be credited as interface/custody RED. The one passing existing leaf is TestSelectGatesActivatesGvOnAttestationEvidence.

TEST-EDIT AUTHORIZED: cmd/machinery/attest_implementation_test.go — ONLY TestAttestImplementationCLI / "C-complete-sole-current-warning" setup. Immediately after its existing cliReviewFixture literal and before both existing copyDirInto calls, add:
    for _, path := range []string{f.design, f.impl} {
        if err := os.MkdirAll(path, 0o755); err != nil {
            t.Fatal(err)
        }
    }
Targets are the two explicit descendants of that subtest's unique t.TempDir root. Keep shared copyDirInto, all copied example inputs, Git history setup, generated claim/kind selection, current control, warning mutation, exit/count/sole-warning assertions and every other frozen byte unchanged except necessary gofmt for this insertion.
Frozen pre-edit file SHA256: 62511c2a3ab4f04cf8b4bfa9fbe7bd0e0ec2e02c850e2e4726d0d5f198ccd58a.

Source cause: existing readonly cmd/machinery/golden_test.go copyDirInto enumerates the source and writes ordinary files under dst, creating nested destination directories only when it encounters source directories. It does not create dst itself. The original failure opening destination design/ARCHITECTURE.md occurred before any complete-mode CLI behavior. This one leaf is SETUP, not C sensitivity or missing-interface proof. Repairing its root setup does not establish that the subsequent complete fixture is valid under every other gate; its unchanged valid control and sole-warning challenge remain mandatory.

Freeze/replay rules:
- Keep original 7ec5d609 and original logs intact; no amend/squash/rewrite or replacement of the original failed evidence.
- Make a separate exact repair commit with BOTH tdd-red and [test-edit-authorized] in its subject. Record original and repaired commit SHAs, exact two-file diff and before/after SHA256 for all five frozen RED files. Verify the other three frozen files unchanged and both repaired files differ only by these authorized setup hunks.
- Re-run the complete previously scoped attestation test selections on unchanged production: gates -run Attest, cmd/machinery -run Attest, hook -run Attestation, designlock -run Attestation, with bounded recorded timeouts, leaf names/counts, elapsed times, and new raw logs. No full preflight.
- The repaired fixtures must pass setup checks. Require actual A legacy-false-acceptance assertion failures plus passing D controls, and classify B unsupported version/flags/callback absence and C unexecuted mutation legs honestly. A new setup failure must be reported as SETUP and its exact additional repair reviewed; no further frozen edit is implicitly allowed.
- Independently replay/review the repaired RED and freeze its final bytes before approve-red. These authorizations do not authorize GREEN dispatch or waive any unresolved acceptance evidence.

Original evidence retained:
- /tmp/machinery-p7jd-red-proof.RvOHwO/hook.jsonl SHA256 ff109b68b5b8399e54a5e57765d9d205ae831d7e646de527f859e232248cdbe0: independently counted 28 terminal leaves, 27 failures and 1 pass; failures inspected as the hooks:null setup problem.
- /tmp/machinery-p7jd-red-proof.RvOHwO/cli.jsonl SHA256 a0fccbdfac2294d5292fc1cd879ca1ad0dbbb13128b16943401ed775373e35a6: independently counted 28 terminal leaves, 17 failures and 11 passes; exact complete fixture missing-destination error inspected. Author classification of remaining failures is 6 A legacy false acceptance and 10 B/C interface absent/unreached; complete independent behavioral replay remains pending.
- Other original gates.jsonl/designlock.jsonl/verify-initial.txt remain alongside those logs. Reported gates 6.501s and CLI 2.866s runs are original author timings, not fresh PM replay.

PM COST/SCOPE INVESTIGATION — SAME-STORY FORECAST ADJUSTMENT JUSTIFIED:
git show --stat 7ec5d609 independently confirms five test files, 1475 insertions and 3 deletions. New files total 1426 LOC (612 gates, 457 CLI, 357 hook); existing authorized amendments are 49 net lines. The prior 850–1350 test forecast is already exceeded before mandatory GREEN supplements. A realistic current forecast is approximately 2710–3310 total: ~1485 RED after the setup repairs, 175–375 supplemental GREEN tests, 900–1250 production and 150–200 docs/help. Combined test estimate is ~1660–1860. These are estimates, not a cap, proof of completion or permission to broaden ownership.

The increase is justified by the approved matrices and distinct observed boundaries: gates independently calculate receipt/inventory and exercise direct lifecycle; CLI builds an isolated executable, handles real Git/current/history/complete-mode and output/exit cases; hooks exercise configured state, Stop/SubagentStop, strict/wave policies and owned destructive-copy child processes. Reviewed source already reuses testgit.Run, copyDirInto and existing package helpers. Sharing a new cross-package fixture API would add ownership and couple proof setup; depending on the future production renderer cannot create independently frozen baseline fixtures. This does not prove no line can be saved; it explains why reducing line count is not a reason to trim the required tests or change frozen fixtures.

No new file/production API/generic factoring is approved by the cost finding. Retain the canonical 13 required paths plus the already authorized optional fourteenth supplemental file and all read-only boundaries. No story split is required by this bounded overrun; splitting the same finalization/schema/CLI custody obligation would add coordination without removing required proof. SrPM should canonicalize the revised forecast and rationale under MAC-p7jd, preserving all AC/R2/AC5 proof requirements; dispatcher has this follow-up. Estimate canonicalization does not block these exact repairs/re-RED, and further actual overrun remains subject to PM investigation.

Review evidence/limits:
- Current shared canonical story, current R2 authorizations and the approved AC5 composition section were read; original pm_acceptor/developer frozen-test rules apply. Current story state is in_progress, hard-tdd, assignee dev-MAC-p7jd, parent MAC-ui8a.
- Graph index_status/search_graph identified Config/Load and copyDirInto; coverage generation 2026-09-06T02:42:16Z reports best-effort metadata_match/no_recorded_issue for internal/hook/hook.go and cmd/machinery/golden_test.go. No claim of exhaustive graph proof.
- Exact git show 7ec5d609 of both frozen test files and their readonly Config/decodeConfig/Load/copyDirInto dependencies; exact file hashing and raw JSONL hashing/leaf extraction above. Source inspection of the tests supplies cost evidence, not acceptance.
- Main remained clean at 497419ab4512fcff765cd5feb27aed4c67b5608d; observed epic 70652b948bf090008b1965c85daf36ea374daea4. PM used committed refs, not the developer worktree. No PM source/test/docs edits, execution replay, installed asset change, remote mutation, full preflight or status transition.
SAME-STORY COST CANONICALIZATION ONLY
GREEN PAUSED — RED-DISPUTE checkpoint; not delivery or acceptance.

RED-DISPUTE 1: cmd/machinery/attest_implementation_test.go TestAttestImplementationCLI/C-complete-sole-current-warning lines412–424 converts acceptance date 2026-09-03 to time.Time via yaml.Unmarshal(map[string]any), then yaml.Marshal rewrites it to 2026-09-03T00:00:00Z. Actual built complete CLI now reaches all13 generation calls and finalized Gv current review, but Ga correctly rejects all6 M0..M5 dates; 6 blocking findings, missing-current challenge UNREACHED. Ga/examples/helpers/frozen files unchanged. Exact repair requires independent PM authorization.

RED-DISPUTE 2: internal/hook/attestation_snapshot_test.go hookReviewFixture scenario.Empty sets Gates="", Impl=""; unchanged progressive selectGatesCheckedInSnapshot unconditionally selects Gl. B-empty-selection and C-empty-cleanup reach callback with one gate and fail len(run)==0 before the intended empty branch/cleanup mutation. Independent PM must choose exact fixture correction; no selector/policy expansion authorized.

PROOF: source390d4dc38be1ad5a16ef3cec171e08d9bf6c0036, branch story/MAC-p7jd, clean retained worktree. Initial47ba449 replay matches189 leaves/47pass/142fail/0skip (24A genuine failures,8D passes,26B missing-interface failures,92C unreached). First WIP e55d534 compilation failure int/int64 preserved; fixed999b9ab. Exact second targeted runs count1/timeout5m/json: gates -run Attest at999b9ab 133leaves/126pass/7fail/0skip,8.984s; CLI -run Attest at390d4dc 28/26/2/0,7.226s; hook -run Attestation at390d4dc 28/26/2/0,4.363s. These are separate revisions, not final combined proof. Native names and all stdout/stderr/source hashes/report are in /tmp/machinery-p7jd-green-proof.eHzGMx/. PAUSED-RED-DISPUTE.md records all failures, reached stages and cost. Coverage NOT MEASURED; no profiles/broader tests/preflight. Baseline designlock0tests is NO capability proof.

Implementation failures retained: gates6 custody cases return duplicate G0 findings and1 addition diagnostic omits descendant; production fixes committed390d4dc but gates replay owed. CLI missing-input reaches real renderer exit1/empty stdout but omits BUILD.md from diagnostic; fix still owed. All26 other CLI leaves pass; all26 other hook leaves pass, including actual original/owned-copy cleanup faults across Stop/SubagentStop strict/relaxed/wave, with callback/sentinel/ledger assertions. Mandatory NEW designlock/wrapper/renderer supplements and docs remain unauthored, all final AC proof owed.

WIP cost:6 production files1125add/43delete=1168lines; combined frozen RED11files2609add/46delete=2655lines. No generic designlock/external/source/inventory/portablepath/scale, Ga, cmd hook, example, golden, installed binary/asset, main/epic, dependency, service/container or remote edits. All5 approved frozen hashes unchanged. No uncommitted source or active test/build/helper process at pause. No delivery/release/acceptance/merge transition. Root acknowledged pause and will route independent review.

LEARNINGS: generic YAML-map round trips can alter exact date lexemes; empty progressive hook config still activates Gl; duplicate lifecycle errors need a consumable finalized wrapper shape. Passing partial cases do not establish the complete contract.
GREEN-PHASE RED-DISPUTE PM ADJUDICATION — MAC-p7jd — 2026-09-06

Disposition: BOTH exact fixture disputes are justified; only the two repair hunks below are TEST-EDIT AUTHORIZED before editing. This is a bounded healthy dispute decision, not delivery, acceptance, rejection, approve-red or a claim of completed GREEN proof. Source checkpoint is 390d4dc38be1ad5a16ef3cec171e08d9bf6c0036; frozen repaired RED remains 47ba44906a09bc2fa010092a86b133d0d749c52c, with original 7ec5d609 retained. Preserve current in_progress / hard-tdd / red-approved / assignee dev-MAC-p7jd / parent MAC-ui8a. The existing red-approved label records earlier review; this note does not approve future amended bytes without their required evidence/audit.

Authority reviewed: current shared canonical Description and final in_progress contract; compared the Description to the previously reviewed R2/AC5 canonical text and read its changed cost forecast. R2 SHA256 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179, approved AC5 composition 5fc3193106a803b7760020b646ed4db7b3d417f5e196b2266d10cf8227733937, all five AC, four exact seam uses/read-only boundaries and current 2710–3310 forecast remain unchanged. Prior hooks:true/Load and complete-fixture MkdirAll authorizations remain valid history.

TEST-EDIT AUTHORIZED 1: cmd/machinery/attest_implementation_test.go, TestAttestImplementationCLI / C-complete-sole-current-warning.
Replace ONLY the body of the existing for _, path := range files acceptance-copy loop (currently var row map[string]any -> yaml.Unmarshal -> row["commit"]=anchor -> yaml.Marshal -> cliReviewWrite) with exactly:
    lines := strings.Split(string(cliReviewRead(t, path)), "\n")
    commits := 0
    for i, line := range lines {
        if strings.HasPrefix(line, "commit: ") {
            lines[i] = fmt.Sprintf("commit: %q", anchor)
            commits++
        }
    }
    if commits != 1 {
        t.Fatalf("acceptance fixture %s: want exactly one root commit line, got %d", path, commits)
    }
    cliReviewWrite(t, path, strings.Join(lines, "\n"))

This deliberately edits only the one known root commit line in each copied fixture; every other byte, including all date lexemes, array items, findings, reviewer, milestones and attestation text is preserved. The quoted anchor remains a YAML string, including if a generated Git object ID were numeric-looking. The guard must require exactly one root commit line and fail otherwise; no fallback reserialization or hardcoded date correction is authorized. The six committed source fixtures each have exactly one unindented "commit: " line and unchanged plain "date: 2026-09-03". No new import is needed (strings/fmt already exist). Preserve the enclosing file loop and all subsequent generation, valid-current, complete-mode, plan conversion, exact exit/count and sole-warning assertions; all other file bytes unchanged except necessary local gofmt.

Frozen pre-repair/current SHA256: e02ad18fe2745d23f05e69ce1e01dd2b1be358f94d99cb3a215b14f6e2cee3b8.

Reason/evidence: the untyped yaml.v3 map decode resolves plain dates to time.Time; its encoder formats time.Time as RFC3339Nano. The frozen loop therefore rewrites each date to 2026-09-03T00:00:00Z. Unchanged internal/gates/accept.go parseAcceptance requires YYYY-MM-DD and correctly rejects those six altered values. Raw 390d4dc CLI output shows all 13 v2 generation calls succeeding, finalized Gv current count 1, and exactly six Ga date errors at M0–M5. The intended missing-current mutation is UNREACHED. This is a fixture transformation defect; Ga, example/golden acceptance YAML, shared helpers and product date semantics must remain unchanged. Earlier base unknown-flag/generation failures retain their interface classification, not retrospectively claimed complete-fixture validity.

TEST-EDIT AUTHORIZED 2: internal/hook/attestation_snapshot_test.go, hookReviewFixture, only the scenario.Empty branch:
    cfg.Gates = ""
becomes:
    cfg.Gates = "g4,gt"
Keep the existing cfg.Impl = "" and every other byte/assertion unchanged. This selects the existing explicit configuration path that drops both implementation-facing gates when no impl is configured, yielding empty sel.Run with its existing visible warning. It does not alter selection, config parsing, event routing, hook finalization or product policy. The earlier Hooks:true and real Load validation stay intact.

Frozen pre-repair/current SHA256: a3c088c095e2dba4379f548903a058eedbd5338ef872e507ab12edaa1ca06e49.

Reason/evidence: selectGatesCheckedInSnapshot first validates inventory; with a nonempty explicit gates string it selects those gates, drops g4 and gt if cfg.Impl is empty, and returns their explanatory warning. Its progressive empty-string path always sets run["gl"]=true. That selector is unchanged in 390d4dc. Original B-empty-selection and C-empty-cleanup therefore reach the callback with one gate and fail "empty selection supplied gates" before the cleanup mutation. These are wrong fixture selection/setup history, not observed empty-branch proof.
The existing no-fault assertions already permit nonblocking warning JSON and require ledger clearing. Keep the len(run)==0 check, fired-once check, no-before-finalization output, helper-process/private-TMPDIR custody, actual regular-entry-to-symlink operation, sentinel preservation, exactly-one-block/retained-ledger checks, concrete private-snapshot cleanup/symlink cause and no-private-path assertions unchanged. Post-repair no-fault success and the actual empty-branch cleanup failure must be observed independently; no credit is conferred by this authorization. In particular, any post-repair failure from empty-branch cause/path formatting remains implementation work, never a reason to weaken frozen assertions.

Frozen boundaries and repair protocol:
- Only these two hunks are authorized; no other existing test edit, helper factoring, new seam, generated/example edit or product-policy change is granted.
- Use a separate repair commit whose subject carries the literal [test-edit-authorized]. Preserve 7ec5d609, 47ba449, all implementation commits and original raw logs; no amend/squash/rebase or replacement of failed evidence.
- Record exact diff, repair/source SHA, before/after hashes for all five frozen files, and prove the other three remain byte-identical. The two edited files may differ only by the exact hunks above and necessary local formatting.
- Run all four scoped selectors against one recorded post-repair revision: go test -count=1 -timeout=5m -json ./internal/gates -run Attest; corresponding ./cmd/machinery -run Attest; ./internal/hook -run Attestation; ./internal/designlock -run Attestation. Retain full output, stderr, native leaf inventory/pass/fail/skip counts and durations. No full preflight.
- Demonstrate the complete CLI unchanged control passes all gates, then the existing missing-current mutation is reached and yields its sole promoted warning. Demonstrate B-empty-selection fires with zero gates and its no-fault outcome; C-empty-cleanup must pass its matched control, execute the actual mutation and satisfy every unchanged custody/output/path assertion.
- Run the normal TDD audit against the approved RED/base references; carry the authorization marker into evidence and submit the exact exception diff/new hashes for independent reviewer verification. Corrected fixture bytes remain frozen after repair; no blanket follow-on edits. Re-RED/independent verification requirements of the existing frozen-test protocol are not waived.
- Final GREEN still requires mandatory new designlock and real renderer/wrapper supplemental tests, docs, all AC proof and same-revision AC5 OBSERVED renderer/process plus REVIEWED closure conjunction. No tests-to-run in designlock is no capability proof, and these repairs do not substitute for new mandatory tests.

Separate implementation failures — NO TEST EDIT AUTHORIZED:
The seven gates failures were recorded at 999b9ab: six duplicate G0 capture/release diagnostic results and one missing added descendant. Production fixes are in 390d4dc but gates replay is owed. The 390d4dc CLI missing-input leaf passes unchanged control, removes BUILD.md and gets exit1/zero stdout, but its message omits BUILD.md; fix production diagnostics while retaining its assertion. Neither problem is an approved fixture dispute. The current empty hook branch directly formats the final error while the nonempty branch explicitly calls LogicalError; the unchanged no-private-path test must adjudicate any exposed difference after the setup repair.

Observed evidence and limits:
- Full /tmp/machinery-p7jd-green-proof.eHzGMx/PAUSED-RED-DISPUTE.md read; independently verified SHA256 16872fe3a24b8d133b235487b37e28039b2fa5a6b7d721fb4fe1bb9f15791d94.
- Original raw round2-cli.jsonl SHA256 4af74aea7cf6da8a90bb1bc4a371b033227d478f277607ed54ea1fff48623f24 and round2-hook.jsonl SHA256 d2fe1b0d27b482c180eb6c084770a93368eeebea56008a671f34afd77f85a021 verified. Relevant full raw output inspected, including all thirteen successful generation calls, all six Ga errors, missing BUILD diagnostic and both empty callback failures. Independent terminal-leaf extraction confirms each package 28 leaves/26pass/2fail. These are recorded 390d4dc CLI/hook observations, not a fresh PM execution.
- Gates126/133pass belongs to 999b9ab and is not aggregated with 390d4dc CLI/hook into completed same-revision proof. No PM test execution was necessary to resolve these source/log-supported fixture doubts.
- Exact git show 390d4dc of frozen blocks, selector, Ga date validation and example M0; git grep confirms identical plain dates and one root commit line in M0–M5. git diff 47ba449..390d4dc confirms no edits to the five frozen files, Ga or acceptance examples; hook production diff is confined to finalization, selector untouched. Actual two file hashes match the report.
- Read installed module source for pinned gopkg.in/yaml.v3 v3.0.1: decode.go scalar interface assigns the resolved value, encode.go timev uses RFC3339Nano. No package install, library modification or invented timestamp behavior.
- Codebase-memory index/search/selector trace and coverage generation 2026-09-06T02:42:16Z are best-effort metadata_match/no recorded issue for hook.go, accept.go and six source acceptance files. Exact committed source is branch authority.
- Measured production6files1125add/43del =1168 changed lines; combined11files2609add/46del =2655. Remaining tests/docs and actual final cost still owed; the canonical forecast is not completion proof.
- Main497419ab4512fcff765cd5feb27aed4c67b5608d clean and epic70652b948bf090008b1965c85daf36ea374daea4 unchanged. PM inspected committed refs and proof files only, not developer worktree internals. No source/test/docs, installed binary/assets, remote, services, Docker or preflight changes; private pvg tracker writes only. Machinery standalone constraint unchanged.
SR PM LEGACY FIXTURE OWNERSHIP / PROPOSAL-PREPARATION HOLD. Read complete PAUSED-BLAST-DISPUTE.md SHA256d2837c4fbd4205ba151b856e96a7747d3cc32b58b1a69b4c49fe1d6fedb6bc1f and terminal report.229scoped+45race PASS belong to4b246ba; paused51454e069ebe4039f02d6d9108acf9354c7ad6c8 is docs-only later and still owes finalsameSHA proof. Four old-fixture leaves remain held: hook_test.go TestStopGreenDesignClearsStateSilently and obligation_ownership_test.go TestObligationParentRealCLI obligation-free/Policy/Isolation default-gate legs. Legacy v1 gt correctly fails GV_MISSING_IMPLEMENTATION_SUBJECT; silence/ledger, default-vs-explicit Gt, real controls/negatives, Ga ancestry/selection and no-grandfathering are fixed. hgz1 now owns all8bundled evidence migrations AFTER p7/uzxr/lhu5; no p7 reverse dependency or bundled writes to unblock this story. Root may resume the healthy retained GREEN author for an UNAPPLIED EXTERNAL exact test-local fixture proposal against51454e0 limited to those2existingtest paths, with full old/new hunks, original-assertion byte equality, helper callers, real input/evidence inventory, warning/silence semantics and per-file cost. This is proposal preparation only, not oldtest editing or TEST-EDIT AUTHORIZED. Separate independent PM must approve exacttext before any subsequent sanctioned amendment. Verified existing APIs: gates.AttestationReview{Claim,Kind,Attestor,Date,Note string}; RenderAttestation(design,impl string,review AttestationReview)([]byte,error); CheckAttestationsWithImplementation(design,impl string)*Gate. No missing core schema/API identified. Exact valid fixture construction remains unresolved: plan warnings cannot be assumed silent; real/synthetic impl hashes do not establish substantive current conformance, especially before uzxr repair. If no fixture preserves all constraints, report exact technical conflict for independent specialist review; do not weaken assertions, invent review or change product semantics. Existing filesystem skip remains uncredited. Scoped structural verification after exact canonical repair: pvg lint --backlog --epic MAC-ui8a scanned37 issues,0 errors/0 review findings; pvg rtm check37 stories/19closed/0 extracted requirements; pvg nd dep cycles found none. These are structure checks, not AC proof. Claims/status/labels/dependencies retained; no source/test/example/toolchain/remote/preflight mutation or Paivot product dependency.


## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-06.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## Implementation Evidence (GREEN literal-none repair)

### Summary
- Corrected only `internal/gates/attest.go`: digest descriptor `none` no longer authorizes omission of a genuine literal `none` entry. Actual in-root evidence remains the only entry exclusion; digest grammar/schema/full-root-v1 unchanged.
- External raw proof: `/tmp/machinery-p7jd-none-green.eMIDLD/GREEN-REPORT.md`; raw JSON/test identities/control and negative logs are hash-bound by evidence-sha256 `30867e546b29a2b691215409f41ed97b196bbdaa42d4213f74cc883b0b6d516a`.

### Commit SHA
- `39a498164fc6e09169ea5b33d470890d315f6a50` on `story/MAC-p7jd`; production delta 3 additions/1 deletion.

### CI Results
- New frozen literal-none CLI matrix: 120 PASS / 0 FAIL / 0 SKIP leaves (four topologies x file/empty-directory, controls and reached negatives).
- Existing targeted scopes: gates Attest 151 PASS; original CLI Attest 28 PASS (plus new matrix); hook 28 PASS; designlock 22 PASS.
- Race: gates 23 PASS; designlock 22 PASS. Exact six hook callers 6 PASS; parent callers 7 PASS. Broader replay 149 PASS / 0 FAIL / 1 existing Darwin casefold SKIP.
- `pvg verify internal/gates/attest.go --format text`: PASS (1 file, 0 issues). `git diff --check`, gofmt: clean. Whole changed-set scanner retains 11 existing hook return-empty heuristic findings, explicitly not clean.

### Commands run
- `GOWORK=off GOPROXY=off GOTOOLCHAIN=local go test -count=1 -timeout=5m -json ./cmd/machinery -run ^TestAttestationScopeNone$`
- Same bounded native JSON command for required gates/CLI/hook/designlock Attest selectors, both required race selectors, six hook + seven parent callers, and four broader selectors; complete exact commands/results are in the external report.
- `pvg story verify-tdd --range a82277af5650b487cea1260c24ffcc1c86d69d8d..HEAD --json`: 13 commits, 0 merge skips, 0 violations.

### Frozen hashes
- `cmd/machinery/attest_scope_none_test.go`: `3d9d8250f759807795251685e1cb924ba8eceb4c819b3fa0376bd15b254f9fc0`.
- Approved helpers unchanged: hook `2b185970a3f69a140338cee97e14db0da82ef0aaf01250364357b104c1458b33`; obligation parent `cc5f5e12a0d41a6eeda60895fece31713c5f068817bb2d3f08b7f1c6ff4d4b04`.

### AC proof
- [x] AC1: Complete full-root inventory retains real top-level literal `none` when the evidence descriptor is absent; later content/mode/add/remove/rename changes now invalidate.
- [x] AC2: Existing closed current/plan/history schema and diagnostics remain covered by the unchanged targeted suites.
- [x] AC3: Existing migration/current-review proof remains passing and unchanged.
- [x] AC4: New real built-CLI controls and independently reached negatives cover all four topologies and both literal entry kinds; actual evidence/.git exclusions remain.
- [x] AC5: Existing bounded CLI/custody/race/caller proof replayed; approved observation/composition limits retained.

LEARNINGS:
- A sentinel digest descriptor is metadata, not filesystem exclusion authority; retain a separate boolean for an actual resolved exclusion.
- The Darwin casefold skip is still a Linux final-gate obligation, not a passing result.

## nd_contract
status: delivered

### evidence
- Commit 39a498164fc6e09169ea5b33d470890d315f6a50; raw same-SHA proof at /tmp/machinery-p7jd-none-green.eMIDLD.
- Targeted/race/caller/broader native replays completed with the stated counts; frozen TDD range has zero violations.

### proof
- [x] AC #1: literal `none` is retained absent a real in-root evidence exclusion.
- [x] AC #2: existing closed classification behavior remains passing.
- [x] AC #3: existing migration behavior remains passing.
- [x] AC #4: four-topology literal file/empty-directory controls and negatives pass.
- [x] AC #5: required bounded CLI/custody/caller evidence replays; disclosed limits retained.

## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-06.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


# MAC-p7jd supplemental literal-none RED candidate

## Implementation Evidence (DELIVERED — supplemental RED only)

PROOF:

Candidate commit `c71faeab997d176224ed6f38437de4604cda0b98`, branch `story/MAC-p7jd`, adds ONLY `cmd/machinery/attest_scope_none_test.go` (364 lines). File SHA256 `3d9d8250f759807795251685e1cb924ba8eceb4c819b3fa0376bd15b254f9fc0`. The commit subject carries `tdd-red`; it precedes both final replays. Production is byte-identical to rejected `412d01b57ae4770a6cf4fb4f48442fb7e632cdd7`. No existing test, fixture helper, production, document, example, schema, dependency, API or seam changed. This delivers a reviewable regression candidate, not GREEN completion or independent RED approval.

### Authority and source review

Read developer, nd plus CLI reference, pvg, vault-knowledge and codebase-memory skills. PM acceptance belongs to the separate independent reviewer, not this author. Read live canonical AC1–5, R2 and current authoring boundaries, exact 2026-09-06T06:44:43Z AUTHORITATIVE SAME-STORY SCOPE AMENDMENT and rejection report. R2 `/tmp/machinery-attestation-contract.jODZpc/PROPOSAL.md` SHA256 `8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179`; full AC5 clarification SHA256 `5fc3193106a803b7760020b646ed4db7b3d417f5e196b2266d10cf8227733937`; full PM rejection `/tmp/machinery-p7jd-pm-final.pT2Q0q/PM-FINAL-REVIEW.md` SHA256 `c3950515759f8cf148b4e2469102bbe1a7b7dabfc4c9b22bdb14297ec07071d6`.

Graph project `Users-ramirosalas-workspace-machinery`, generation `2026-09-06T02:42:16Z`, list pagination complete. Parent supplied goldenBin discovery; own coverage checked all five used source paths. Story helper/new test files are missing from the main index; exact worktree source at rejected412d01b supplies authority. This is best-effort graph coverage, not completeness. Read actual goldenBin/cliReview helpers and hermetic testgit implementation. Vault search/read reinforced independent inventories and reached validation layers. Epic log shows no landed p7 implementation.

### CI/Test Results

All shell commands used explicit worktree cd. Every test used `GOWORK=off GOPROXY=off GOTOOLCHAIN=local`; Go1.27.1 darwin/arm64, offline existing cache, actual ordinary CLI built by goldenBin and real bounded Git/CLI subprocesses. TestMain's existing isolated control directory and cleanup remained unchanged. No mocks, skip gates, verdict injection, product dependency on Paivot, background jobs, services/containers, installation, remote operations or preflight.

Commands run:

- `env GOWORK=off GOPROXY=off GOTOOLCHAIN=local go test -count=1 -timeout=5m -json ./cmd/machinery -run '^TestAttestationScopeNone$'` — first precommit author inventory: 80 PASS / 40 FAIL / 0 SKIP, package27.383s; raw `author-first.jsonl` SHA256 `2c2a7f25e33a3adc2acaf65ff896beca9e7ad6bd2eed0128e18efe0b0716a3ff`.
- `env GOWORK=off GOPROXY=off GOTOOLCHAIN=local go test -count=1 -timeout=2m -json ./cmd/machinery -run '^TestAttestationScopeNone$/././^evidence-only$'` — precommit actual evidence-commit check after adding diagnostic logging and stricter exact-count assertions: 4 PASS / 4 inventory FAIL / 0 SKIP, package3.641s; raw `author-evidence-precommit.jsonl` SHA256 `0c7994025b028d61ad0712dda07959d74ee9b05abfc9f3df6711344f4428f68a`.
- `env GOWORK=off GOPROXY=off GOTOOLCHAIN=local go test -count=1 -timeout=5m -coverprofile=/tmp/machinery-p7jd-none-red.kVs2en/final-new.cover -json ./cmd/machinery -run '^TestAttestationScopeNone$'` — committed complete new matrix: 80 PASS / 40 FAIL / 0 SKIP; package37.147s, suite36.72s; raw `final-new.jsonl` SHA256 `7b7e9f791d9e47a7b21910e855d31c6ec042ddfa18ad61b175e7ad84650c8943`.
- `env GOWORK=off GOPROXY=off GOTOOLCHAIN=local go test -count=1 -timeout=5m -json ./cmd/machinery -run '^TestAttest'` — committed containing CLI selector: 108 PASS / 40 FAIL / 0 SKIP, package45.028s; raw `final-cmd-attest.jsonl` SHA256 `a062950581b1e3f884ac6ab66950c63fef58356290afd66b07543f5832612738`. This overlaps all120 new leaves; the additional28 existing CLI leaves all PASS. Do not sum overlapping selectors as unique coverage.
- `pvg verify cmd/machinery/attest_scope_none_test.go --format text` — `VERIFY: PASSED (1 files scanned, 0 issues)`.
- `pvg story verify-tdd --range a82277af5650b487cea1260c24ffcc1c86d69d8d..HEAD --json` —12 commits checked,0 merges skipped,0 violations.
- `shasum -a 256 -c /tmp/machinery-p7jd-final-proof.O4NxfV/frozen-test-sha256.txt` — all seven OK.
- `git diff --check a82277a HEAD` — empty; `git status --short` — empty after commit.
- `node /tmp/machinery-p7jd-none-red.kVs2en/audit.cjs` — emits `audit.json` with complete leaf names/outcomes/timings/reachability, every changed source hash, all seven expected/actual frozen hashes, source/branch/protected refs and exact17-path cost inventory. Script SHA256 `c5811b50666a98f1cef5920d344647e0a2c5f3d81229e766dac10f7a8f5aac32`; audit SHA256 `d5ed8b75aac5187a15ddcb9b96b713144007f23f6b0fcaf1f9a3914f3dcb197d`.

Summary: new RED80PASS/40 expected behavioral FAIL/0SKIP; existing CLI28PASS/0FAIL/0SKIP. Zero compile/import/setup/timeout/missing-input failures and zero unclassified new failures. All four raw stderr files are empty. The four commands exited1 because they contain intentional RED assertions. Coverage: instrumented parent command package0.0%; actual spawned ordinary CLI is uninstrumented, so this is not a product/whole-project coverage percentage or a threshold assertion. No missing-execution claim is disguised as coverage. Full original four-package/custody/hook/13-caller/broader/race proof and final-epic preflight remain owed on the eventual GREEN candidate, not waived or re-claimed from this RED-only run.

### Reached matrix and precise failure attribution

The120 independent leaves are the Cartesian product of disjoint / inside-design / equal / ancestor, file / empty-directory, and15 named cases. Eight independent absent-evidence inventory leaves compare every generated entry/key/type/mode/file-size/hash plus root/policy/exact grammar digest with a standard-filesystem oracle. Another112 leaves first reach actual unchanged generated-receipt CLI exit0/current1. Separate inventory failure never prevents another leaf's mutation. Every literal input is Lstat-verified; files are regular with recorded byte hashes; directories remain empty and mode0755 changes to0700. Reports retain actual CLI argv, elapsed time, output, generated rows and submitted forgery rows.

All72 intended negative challenges execute:8 each of content-or-mode, addition, removal, rename, omission-old-digest, omission-rehashed, sibling-evidence, nested-evidence and nested-git. Another8 independent-full-manifest checks and24 evidence/Git controls also reach their operations, giving104 post-control challenges total. No mutation is unreached. Exact negative diagnostics are joined category-and-message assertions, preventing the word `none` in a temporary test directory from satisfying the named-path requirement.

Equal and ancestor cells: all60 leaves PASS. Disjoint and inside-design cells:20PASS/40FAIL. The40 failures comprise:

-16 independent manifest mismatches: absent evidence plus after evidence-only / Git-directory / regular-gitfile controls, each in the four affected layout/type cells. Actual manifests omit only literal `none`; the filesystem oracle includes it. Those12 post-control exclusion failures occur AFTER actual unchanged/evidence/Git freshness succeeds and digest stability is checked, not before those operations.
-4 rejected independently constructed complete manifests: actual valid complete receipt includes `none`, but the checker rejects it against its narrowed fresh projection. These controls have their own leaves and cannot prevent omission challenges.
-16 unsafe exit0/current1 results: actual content or mode change, addition, removal and independently rehashed omission, each in four affected cells. Example: `content-or-mode` wants `GV_STALE_CONTENT: scope path none changed`, but actual output reports `1 current implementation reviews` and `0 blocking`.
-4 rename diagnostic/set failures: actual rename DOES fail closed with exit1 and no current success, but reports only `none-renamed` as added. Required raw-ASCII-first path `none` was removed too and should be reported first. These are incomplete inventory diagnostics, NOT falsely current rename results.

All8 old-digest omission tests PASS with `GV_SCOPE_HASH: submitted inventory does not match its recorded hash`. Both forged manifests originate from an independently enumerated complete inventory, removing exactly one `none` entry; rehashed variants calculate the exact R2 digest independently. The full independently generated manifest's separate control fails on the four affected cells as disclosed; generated-receipt baseline controls do pass before every forge is submitted. No generator omission is adopted as an expected oracle.

All16 unrelated sibling/nested `attestations.yaml` content changes reject with exact `GV_STALE_CONTENT` paths. All8 nested `.git` additions reject with `GV_SCOPE_UNSUPPORTED_METADATA` naming the actual logical nested path. Top-level real Git directories and `--separate-git-dir` regular gitfiles are checked through real local commits; ordinary evidence-only changes include presence/content/mode and a real commit. Fixed evidence descriptors are independently checked both before the logical receipt exists and after it exists. No valid `none` name is reserved/rejected, no digest/schema change is introduced, and the original frozen alias/custody/root-narrowing proof is unchanged.

Minimal BUILD fixtures have the same ordinary nonblocking missing-g4.zero-context warning before and after. This is not a complete/zero-warning fixture; warnings cannot satisfy exact exit1/category/path/no-current assertions. Existing reported quality heuristics and case-sensitive Linux final-epic obligation are retained history, not waived by this clean single-new-file scan.

### Ownership, cost and immutable preservation

Increment:1 new file/364 additions/0 deletions; eventual production repair0 lines authored. Aggregate againsta82277a:17 files,3760 additions+66 deletions=3826 changed lines, within the current3770–4000 forecast. Four verification runs cost113.199 cumulative package-seconds; final two overlap in wall time. Authoring involved one complete initial matrix, a bounded eight-leaf evidence-commit refinement check, then the two committed replays. No requirements were trimmed for cost.

All seven frozen hashes match the prior reviewed manifest. Both helper files are byte-identical to412d01b, retaining the exact authorized36+29=65 additions and zero deletions; no additional fixture authority is assumed. Main stays497419ab4512fcff765cd5feb27aed4c67b5608d; epic stays70652b948bf090008b1965c85daf36ea374daea4. Worktree is healthy/clean and retained. No running verification subprocess remains.

### AC Verification

| AC | Supplemental test evidence | Disposition |
|---|---|---|
|1|All eight actual none input cells; independent complete inventory/digest; real content/mode/add/remove/rename and independent omission forgery|RED bar authored/reached; production remains rejected, GREEN pending|
|2|No schema/kind/history changes;28 existing CLI leaves pass including preserved historical/current controls|Prior positive proof retained, not whole-story reacceptance|
|3|No grandfathering or fixture renewal; existing migration/compatibility CLI proof passes unchanged|Prior positive proof retained|
|4|All72 negatives reached; all topologies, exact evidence and VCS exclusions and actual evidence-only commits|RED regression proof complete for independent review;40 failures remain|
|5|Ordinary built CLI generation/check and real filesystem/Git; exact current counts, categories and honest limits|Supplemental actual process proof; prior AC5 OBSERVED/REVIEWED/COMPOSED/UNOBSERVED limits retained|

LEARNINGS:

- A complete-manifest assertion and a mutation challenge need independent leaves: shared generator/checker omissions can pass unchanged controls while excluding real inputs.
- Independently constructed forged inventories test self-hash and actual-root comparison separately; correct old-digest rejection does not establish complete observed scope.
- A rename can reject for its new name while silently missing the removed original; report that path-set defect separately from unsafe current acceptance.
- The literal absence descriptor is grammar, never an additional reserved filename. Real evidence-only commits and same-basename sibling inputs expose this boundary precisely.

DISCOVERED_BUG:
  title: Existing MAC-p7jd literal-none inventory projection rejection reproduced
  context: Production412d01b drops legitimate none entries in disjoint and inside-design scopes; actual changed bytes/mode/add/remove/rehashed narrowed scope falsely retain current approval. Rename rejects only its added name and complete independent manifests are falsely rejected. This is the already canonical same-story blocker, not a new bug request.
  affected_files: internal/gates/attest.go captureAttestationSubject
  discovered_during: MAC-p7jd supplemental RED against412d01b

## nd_contract
status: delivered

### evidence
- Supplemental RED candidatec71faeab997d176224ed6f38437de4604cda0b98, new-file hash3d9d8250f759807795251685e1cb924ba8eceb4c819b3fa0376bd15b254f9fc0; complete raw native matrix and report above. Existing production unchanged/rejected.
- Independent supplemental RED exact-text replay/review is still pending. No approve-red, GREEN repair, acceptance, close or merge occurred or is authorized by this delivery. Prior red-approved label refers to the earlier test set, not this supplemental candidate.

### proof
- [x] Supplemental RED AC1/4 matrix authored and all120 leaves executed;80PASS/40behavioralFAIL/0SKIP, all72 negative challenges reached with separate controls.
- [x] Existing seven frozen tests and exact two fixture helper constructions preserved;28 additional existing CLI leaves PASS.
- [ ] AC1/4 product correctness: same-story failure remains; separately routed GREEN and independent final review required.
- [ ] Independent supplemental RED approval: pending exact-text replay and review.


## nd_contract
status: rejected

### evidence
- PM rejection applied via pvg story reject on 2026-09-05.

### proof
- [ ] Story requires another developer delivery before it can be accepted.


# MAC-p7jd independent whole-story GREEN review

REJECTED [2026-09-06]. One established blocking implementation defect; no acceptance or merge authority.

Candidate: `412d01b57ae4770a6cf4fb4f48442fb7e632cdd7`; production base: `a82277af5650b487cea1260c24ffcc1c86d69d8d`. PM reviewed an external exact Git archive in `/tmp/machinery-p7jd-pm-final.pT2Q0q`, not developer-worktree internals. Every changed exported source/test/doc byte matches the candidate Git object. All finite processes completed. Machinery remains standalone; private pvg is coordination only.

## Decision: complete scope silently excludes a real entry named none

EXPECTED — AC1: “Implementation/test behavior claims bind a complete explicit implementation/test scope under rooted inventory and content hashes, not only BUILD or pack. Any code/test/config addition, removal, rename, content change or scope narrowing affecting the claim invalidates freshness.” AC4 requires meaningful changed-implementation/scope negatives and matched unchanged positives. Approved R2 allows only top-level `.git` and the exact logical `<design>/attestations.yaml` record as scope exclusions. There are no arbitrary path selectors.

DELIVERED — `internal/gates/attest.go:874` initializes `attestationSubject.exclusion` to the digest sentinel `"none"`. Lines 878–881 replace it with an actual evidence-relative path only when that path lies inside implementation. Lines 882–885 then compare every real entry's path with this field and skip equality. A real top-level implementation file or empty directory named `none` is therefore removed from the manifest in disjoint and implementation-inside-design topologies. Both generation and checking make the same omission.

GAP — ordinary CLI generation produces a valid current receipt missing this real entry; subsequent real file-content change or empty-directory mode change remains current with exit 0 and byte-identical output. This is an unauthorized third exclusion, not a reviewer-honesty limitation, allowed evidence-only change, unsupported filesystem, missing fixture setup, or forced late-failure boundary. The lower retained-root snapshot inventories the entry; projection into the attestation manifest drops it. Same-invocation custody protection does not fix freshness across separate invocations, each of which captures the now-changed bytes afresh.

FIX — same-story hard-TDD repair, new positive and negative regression FIRST against this exact candidate, independently reviewed before production repair. Cover real `none` regular files and empty directories, including disjoint and implementation-inside-design; assert the exact inventory includes them, unchanged current review succeeds, and meaningful content/mode or scope change invalidates it. Preserve the intended actual evidence-record and top-level `.git` exclusions, all four topology semantics, and full-root inventory/custody behavior. Distinguish the digest's absence descriptor from actual path-filter authority; do not reserve or reject an otherwise valid name as an improvised workaround. Root/Sr PM must route exact test ownership and any cost/path adjustment before authoring; this review does not invent a filename, authorize editing seven frozen files, or grant any new seam. No production-first one-line fix or blanket test amendment is authorized. After repair, freeze new proof and replay scoped, custody, CLI, hook, caller, relevant broader/race and TDD proof on one final revision for independent re-review.

No additional blocker was established in the complete changed-source sweep. Search of other `none`, `exclusion` and `copySkip` comparisons found the above collision only; private-copy skipping has an explicit nonempty-path guard. Equal-root and implementation-ancestor controls include `none` and correctly reject its changed bytes. This bounds the finding without claiming arbitrary inputs have been exhaustively proved.

## Real CLI reproductions and immutable evidence

Built from the exact archive with `go build -o /tmp/machinery-p7jd-pm-final.pT2Q0q/machinery-pm ./cmd/machinery`. Binary SHA256 `534f53900fc9457d0950bcc6ed1d72aeac1d397d030a8017532cc7911e53f0ad`.

For each fixture below, from its own directory, ran:

```text
<binary> attest --design <design> --impl <impl> --claim gt.conformance-test-shape --kind current --attestor 'PM diagnostic fixture only' --date 2026-09-06
<binary> check <design> --impl <impl> --gate gv
```

The generated document was saved unchanged as that design's `attestations.yaml`, followed by a check before and after mutation. All six generation and unchanged controls returned 0. Attribution is explicitly diagnostic, not an assertion that a real reviewer or test execution was authenticated. These intentionally minimal fixtures have one unchanged ordinary missing-`g4.zero-context` warning; that warning is nonblocking and does not explain acceptance of changed `none`. This is not a `--complete` or zero-warning claim.

| Fixture directory | Design / implementation | Mutation | Check after mutation |
| --- | --- | --- | --- |
| `none-proof` | `design` / `impl` | real `none` file bytes | BUG: exit 0, current 1 |
| `none-inside` | `design` / `design/impl` | real `none` file bytes | BUG: exit 0, current 1 |
| `none-dir-disjoint` | `design` / `impl` | real empty `none` directory mode 0755 to 0700 | BUG: exit 0, current 1 |
| `none-dir-inside` | `design` / `design/impl` | real empty `none` directory mode 0755 to 0700 | BUG: exit 0, current 1 |
| `none-equal` | `.` / `.` | same real file-byte mutation | expected exit 1, `GV_STALE_CONTENT: scope path none changed` |
| `none-ancestor` | `design` / `.` | same real file-byte mutation | expected exit 1, `GV_STALE_CONTENT: scope path none changed` |

All paths are beneath the PM directory above (`/tmp` resolves to `/private/tmp` on this host). Files changed from `assert actual behavior\n` to `return unchecked behavior\n`; other implementation file `handler.go` stayed `package example\n`. Original input copies `none-input-before.txt` and `none-input-after.txt` have SHAs `02f5fd2d7d6e18dc8c09b2062ce84cfe680107baaeed2d3ff9e0c1a505815c0c` and `0738305e11fd26f62bf4b8adea38aa2921ca8836089708c277b2c77f338408b8`. Empty-directory children remain `[]`, with actual 0700 mode recorded.

Raw documents and before/after stdout/stderr use prefixes `none`, `none-inside`, `none-dir-disjoint`, `none-dir-inside`, `none-equal`, `none-ancestor`: suffixes `-generated.yaml`, `-generate.stderr`, `-control.stdout`, `-control.stderr`, `-mutated.stdout`, `-mutated.stderr`. Bad-case receipts contain only `.` and `handler.go`; no `none` entry. All four bad-case unchanged and changed stdout hashes are identical: `dcf9098147973c7d631978167fc25f20c90e9c7071770dd775b69164bc75c86d`. Check stderr is empty. Receipt SHAs:

- `none-generated.yaml`: `ec6f0b05ee7b4d58381e060e6181d5464e0070ab9f582a6dc53007aa0ff237d7`.
- `none-inside-generated.yaml` and `none-dir-inside-generated.yaml`: `14178ac6b5966b8fc4bcb3c3c490a65f6739915b41553bc264e30d417fa0a04e`.
- `none-dir-disjoint-generated.yaml`: `4021d38cd6ab39388e6c5162f28642fb3ed89fda18bdba164728ed5633ed3f06`.
- Equal/ancestor: `603c620a9074167233c9ce833c14654b2635531b4da37ee0e121045367d93849` / `516906b467e493171972e3414b3c434c439b63484d29f477975b32fff6ed9e67`.

Final finite repeat command `node pm-none-replay.cjs > pm-none-replay.json` independently rechecks all six preserved mutants, records executable/actual-receipt hashes, exact cwd/argv/status/stdout/stderr and target type/mode/content/children. Script SHA `04a4bff364fb659383eea5f1b54884288dc522f458a7f0f44dbe83c8854f7c35`; JSON SHA `0fd1686fc37b983c35d569d3ece449056d7f37ea3b25d589b4036a5b87d1d78a`. It expects the observed defective outcomes, so its exit 0 is diagnostic reproducibility, NOT a passing product regression.

## Independent same-revision tests and provenance

All native commands use `go test -count=1 -timeout=5m -json`; race runs additionally `-race`. Output files below live in the PM archive; corresponding `.stderr` files are empty. Exact names, package durations and outcomes are retained in `pm-review-audit.json`. Scopes overlap and are not summed into unique coverage.

| Raw JSONL | Packages / selector | PASS / FAIL / SKIP |
| --- | --- | --- |
| `pm-scoped.jsonl` | `./internal/gates ./cmd/machinery ./internal/hook ./internal/designlock`, `Attest|Attestation` | 229 / 0 / 0 |
| `pm-race-designlock.jsonl` | designlock, `Attestation`, race | 22 / 0 / 0 |
| `pm-race-gates.jsonl` | gates, `TestAttestGreen|TestAttestationCRelease`, race | 23 / 0 / 0 |
| `pm-callers-hook.jsonl` | hook, six exact helper callers listed below | 6 / 0 / 0 |
| `pm-callers-parent.jsonl` | gates, `TestObligationParent` | 7 / 0 / 0 |
| `pm-blast-gates.jsonl` | gates, `TestSelect|TestRunSelected|TestCargoWorkspacePointerMutation|TestGateSnapshot` | 33 / 0 / 1 |
| `pm-blast-hook.jsonl` | hook, `TestStop|TestSelectGates|TestGreenStop` | 24 / 0 / 0 |
| `pm-blast-accepted.jsonl` | gates, `TestObligation|TestReadsConsumer` | 79 / 0 / 0 |
| `pm-blast-designlock.jsonl` | designlock, `TestMaterializeDesignWorkspace|TestExternalTreeSnapshotCleanup|TestRegularFileSnapshotCleanup|TestExternalSnapshotRejects|TestUniversalSnapshotBoundary` | 13 / 0 / 0 |

Hook caller selector: `TestStopGreenDesignClearsStateSilently|TestPreEditObligationSurvivesLostPostAndReplacementSession|TestWaveDeferralSurvivesCrashAndOutOfBandClose|TestStopDriftBlocks|TestStopWarnsWhenStagedImplGatesLackImpl|TestReapedStopStillGatesTheTree`.

Audit script `pm-review-audit.cjs` SHA `505ca6beae19b27505790254decaa4efa58a569b623043a9476d19e29e9abf14`; result `pm-review-audit.json` SHA `d5ef7a3f84b733c7293d35f68761cd4e90c3d49894ca727b92f8affd4799d8a2`. It includes SHA256 for every raw run, all leaf identities, changed-source hashes, original 189 frozen leaf lineage, fixture diagnostics and quality provenance. Author's full evidence manifest independently matched actual files. Original 47 PASS / 142 FAIL frozen RED and earlier explicitly classified SETUP failures remain valid retained history; current omission is a coverage gap, not retroactive falsification.

`pvg story verify-tdd --range a82277a..412d01b --json`: 11 commits, zero merges/violations. `git diff --check a82277a 412d01b`: empty. Final diff: 16 paths, 3396 additions + 66 deletions = 3462 changed lines. All seven frozen test hashes match their exact authorities; only the separate approved 65-addition fixture patch affects the additional two old helper files. Read-only generic designlock/source/external/inventory/portablepath, shared Ga, command hook and module/dependency boundaries are unchanged. hgz1's explicit future two-helper ownership and fresh actual-baseline PM amendment hold remain untouched; no future current-to-plan downgrade is authorized.

## Whole-contract source review and AC5 closure

Read full live canonical AC/R2/AC5/current authority and delivery notes, full 277-line R2 proposal SHA `8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179`, full AC5 clarification SHA `5fc3193106a803b7760020b646ed4db7b3d417f5e196b2266d10cf8227733937`, and full author GREEN report SHA `815e8dd3c79f0b28dba51b8f7354b9cb9fe29f862327839de14d3163645bf033`. Graph-first main index coverage supplied existing wrapper/caller relationships; new story paths absent from the main index were reviewed from exact committed Git objects, not assumed indexed.

Reviewed complete new attestation capability, attestation checker/renderer, changed suite/hook/CLI source and docs; complete five new RED/GREEN test files, exact existing-test patches and relevant old helper callers. Inspected unchanged YAML parser, logical error mapping, generic file-copy/inventory providers, command output/error/root-exit machinery and old skipped test where they affect the proof. This does not claim every unrelated line of the repository was reviewed.

Source sweep covered closed v2 kinds/required covers/legacy errors; inventory grammar/digest/self-check/both-direction set comparison; alias, hardlink, bounds and all four topologies; original held-root generation and private-copy ownership; before/after reads and final revalidation; full Release joining/latching/repeat behavior and provisional public counters; all wrappers, strict-false/wave/empty Stop paths and ledger/output ordering; CLI flags/dates/missing BUILD diagnostics; complete output/Ga history; exact 16 MiB and per-pass overlay budget documentation. No additional source defect is established beyond the sentinel collision. Positive inspected properties do not establish the contradicted full-inventory AC.

AC5 evidence tiers at the same candidate:

- OBSERVED: actual Render late original mutation and real owned-copy cleanup fault return nil bytes and actual causes, paired with fired no-fault controls; owned sentinel/logical-path protection passes.
- OBSERVED: built ordinary CLI success and ordinary renderer-input/alias failures exercise the real branch, with required failure exit 1 and empty stdout; real current/history/changed-code cases pass their covered fixtures.
- REVIEWED: `cmd/machinery/attest.go:120` invokes Render; `internal/gates/attest.go:949–965` joins check and final Release before returning bytes; `internal/gates/suite.go:82–106` completes retained-capability/workspace/lock finalization. Any renderer error reaches CLI error branch before its first generation stdout write at line 128. `cmd/machinery/io.go` and main error/exit handling preserve failure and do not defer, retry or provide an alternate stdout emission. Render itself emits no stdout.
- COMPOSED: a late renderer failure traverses this same CLI closure to exit 1/empty stdout. This bounded guarantee does not cure the manifest's missing real input.
- UNOBSERVED: independently injected standalone CLI late-release fault. UNFORCED: individual OS root-handle/filelock Close primitive errors. Actual owned-copy cleanup supplies the approved lifecycle-fault alternative. Sink partial writes are separate; no execution or reviewer-identity authentication is claimed.

The full-story acceptance proof remains false for AC1 and incomplete for AC4; AC2/3 and the bounded AC5 closure have positive independent evidence, not blanket story acceptance.

## Honest remaining limits and disposition

Author targeted four-package coverage 16.1% is not whole-project coverage or a threshold claim. Scoped quality scan is NOT clean: 11 existing return-empty heuristics. Full containing functions `logicalSnapshotError`, `generatedReason`, `hookFileChangeID`, `relToRoot` were independently byte-compared to base and are unchanged; audit contains hashes. No source was altered to silence them.

One unchanged broader skip, `TestSelectRejectsNonportableAndAliasedDesignPaths/case_folded_collision`, skips before its assertion on this case-insensitive filesystem. It is neither PASS nor waived; all required attestation and helper scopes have zero skips. Exact case-sensitive Linux/final-epic leaf remains owed: `go test -count=1 -timeout=5m -json ./internal/gates -run '^TestSelectRejectsNonportableAndAliasedDesignPaths$/^case_folded_collision$'`. No full preflight, Linux/container setup, service or installed artifact operation was performed or authorized. This disclosed final-gate obligation is separate from the concrete blocking defect.

Root/Sr PM handoff: same-story AC1/4 regression RED and repair, preserve all prior proof and frozen bytes, route exact new test scope before edits. No broader product compatibility amnesty, fixture renewal, test filename authority or new seam follows from this rejection. Main was clean at final read-only check. All external PM evidence remains available; nothing material was removed.


## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


# MAC-p7jd GREEN delivery evidence

PROOF:

## Revision, authority, and result

Candidate **412d01b57ae4770a6cf4fb4f48442fb7e632cdd7**, branch story/MAC-p7jd,
worktree /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-p7jd.
Production base a82277af5650b487cea1260c24ffcc1c86d69d8d. Clean worktree; all
finite processes completed. Delivery is for independent PM review, NOT acceptance.

Fresh same-candidate results:229 scoped leaves PASS,45 targeted race leaves PASS,
all13 affected fixture-caller leaves PASS, no failures/skips in those sets.
Broader regressions:149 PASS,0 FAIL,1 pre-existing filesystem-dependent SKIP.
Sets overlap; they are not summed as unique total test coverage.

R2 authority8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179
and AC5 clarification5fc3193106a803b7760020b646ed4db7b3d417f5e196b2266d10cf8227733937
remain unchanged. Exact fixture authority was read completely from
/tmp/machinery-p7jd-pm-fixture.Mm3FM4/PM-DISPOSITION.md,
SHA25691b9e8773a2174e0cdd30a8b8c162b442436accdbe965341b08c96ecfed8ddea,
and the canonical live Body matched19a64c1fc64129b941dd1af9148caedff7f6295e03c839685f7a1ff40f01683b.

412d01b applies ONLY the exact65-addition/0-deletion fixture patch
ae6e9241ea5d7b0c4556b2ee02c6e2910b2663c2452761e7984f1c6663252971 against51454e0.
Its commit contains literal [test-edit-authorized]. No production/example/golden
or other test change was made during this final authorized resume.

## Exact executed commands and native inventory

All shell commands used the absolute worktree cwd. Every native test command
below is `go test -count=1 -timeout=5m -json <package> -run '<selector>'`, plus
`-race` only in the race rows. Raw stdout JSON and native stderr are the named
`.jsonl`/`.stderr` files in this directory; no output was replaced by a summary.

| Run | Package | Selector | PASS/FAIL/SKIP leaves | Native seconds |
|---|---|---|---|---:|
| gates | ./internal/gates | Attest |151/0/0|17.876|
| cli | ./cmd/machinery | Attest |28/0/0|15.601|
| hook | ./internal/hook | Attestation |28/0/0|8.673|
| designlock | ./internal/designlock | Attestation |22/0/0|2.745|
| race-gates | ./internal/gates | TestAttestGreen\|TestAttestationCRelease |23/0/0|29.950|
| race-designlock | ./internal/designlock | Attestation |22/0/0|2.604|
| callers-hook | ./internal/hook | full six-name selector below |6/0/0|4.551|
| callers-parent | ./internal/gates | TestObligationParent |7/0/0|8.637|
| blast-gates | ./internal/gates | TestSelect\|TestRunSelected\|TestCargoWorkspacePointerMutation\|TestGateSnapshot |33/0/1|2.853|
| blast-hook | ./internal/hook | TestStop\|TestSelectGates\|TestGreenStop |24/0/0|5.943|
| blast-accepted | ./internal/gates | TestObligation\|TestReadsConsumer |79/0/0|19.544|
| blast-designlock | ./internal/designlock | full selector below |13/0/0|0.787|

The literal callers-hook selector is:
`TestStopGreenDesignClearsStateSilently|TestPreEditObligationSurvivesLostPostAndReplacementSession|TestWaveDeferralSurvivesCrashAndOutOfBandClose|TestStopDriftBlocks|TestStopWarnsWhenStagedImplGatesLackImpl|TestReapedStopStillGatesTheTree`.
The literal blast-designlock selector is:
`TestMaterializeDesignWorkspace|TestExternalTreeSnapshotCleanup|TestRegularFileSnapshotCleanup|TestExternalSnapshotRejects|TestUniversalSnapshotBoundary`.

`native-summary.json` lists every producing package, elapsed time and raw hash.
Each run's `*-native-leaves.json` contains the complete actual native leaf names,
outcomes and times, not a sampled list. `frozen-189-lineage.json` independently
matches all189 original baseline identities to PASS on this candidate.
No original leaf is missing. Audit script `audit.cjs` is external proof tooling,
not a product/build/test dependency.

### Coverage and build

Same-candidate coverage command:
`go test -count=1 -timeout=5m -coverprofile=/tmp/machinery-p7jd-final-proof.O4NxfV/attestation.cover -json ./internal/gates ./cmd/machinery ./internal/hook ./internal/designlock -run 'Attest|Attestation'`.
229PASS0FAIL0SKIP again; native designlock3.215s,hook8.133s,CLI16.051s,gates19.052s.
`go tool cover -func=.../attestation.cover` reports **16.1%** statement coverage
across the four complete packages under this targeted selector, NOT full-suite
coverage. Instrumented subprocess percentage printouts are preserved, not added
together. The stored profile's statement-weighted per-file counts are:
new capability224/269=83.3%; gates/attest.go458/529=86.6%; CLI/attest.go85/101=84.2%.
`coverage-by-file.json`, `coverage-functions.txt`, raw coverage output and profile
are retained. No coverage threshold or unobserved branch is implied satisfied.

`go build -o /tmp/machinery-p7jd-final-proof.O4NxfV/machinery ./cmd/machinery`
succeeded. This isolated candidate's `attest --help` was captured, including all
generation flags, fixed limits, merge bound and exact non-authentication text.
Installed binaries, plugins, skills, NIL runtime and services were not changed.
go.mod/go.sum diff is empty; shipped Machinery has no pvg/nd/Paivot dependency.

## Reached A/B/C/D and supplemental proof

Original RED7ec5d609acc1597ee2c0ddbf5401b92f834d5ab3, independently repaired
frozen RED47ba44906a09bc2fa010092a86b133d0d749c52c, baseline47PASS142FAIL0SKIP,
and every intermediate build/setup/diagnostic failure remain in
/tmp/machinery-p7jd-green-proof.eHzGMx. A genuine baseline failures, B interface
absence, C initially unreachable controls/challenges, and D compatibility are
not retroactively reclassified as valid RED successes. Final exact native names
retain these stage prefixes. All C unchanged controls and their mutation legs
now execute successfully; previously unreached schema challenges are reached.

Required production observations include:

- Complete implementation root inventories bind hidden/ignored/generated/vendor/
  config/test files and empty directories across all4 logical topologies.
  Exact evidence-only edits/commits and top-level Git administration remain
  compatible; unrelated attestations.yaml and nested Git metadata do not escape.
- Real changed handlers, removed assertions, added/removed/renamed files, narrowed
  manifests/roots, content/mode/size changes, aliases/symlinks/special files and
  stale replay fail through the unchanged frozen assertions. The independent
  digest vector and both-direction path-set checks pass.
- Legacy behavioral v1 fails even on unchanged design covers; v1 plan/history
  remains explicitly limited. V2 wrong kind/missing root/closed schema rejects.
  Real Git ancestor Ga evidence remains historical; it does not refresh current
  implementation approval. Complete-mode unchanged success runs first, then
  explicit plan conversion produces the sole missing-current warning and failure.
- Pending current counters stay absent until release. Multiple RunSelected
  results finalize together; original mutation/root replacement/supplementary
  failure/observed-then-restored failure suppress current counters. Repeated
  Release returns the same disposition; post-release access fails closed.
- Actual Stop/SubagentStop control/fault matrices cover relaxed/strict/wave and
  empty selection. Cleanup is completed before tally/output/ledger clearing;
  real failure blocks, retains ledger and preserves external sentinels.

New GREEN proof (first added/frozen in4b246ba, unchanged thereafter) adds22
capability and18 gates leaves. It uses only the four approved seam boundaries:
existing Snapshot APIs; approved hook writer callback; suite first-final-Release
callback; new-capability tests using the existing first-read callback and exact
fixed-default/lowered-budget factory. No callback mutates a verdict or directly
closes/replaces a retained lifecycle handle.

- Capability: actual held original roots, private copies, original-generation
  gap, same-byte/restored-mtime design identity replacement, root rename,
  private-copy mutation, defensive Entries copy and release checks.
- Copy: real2*64KiB reads with fired unchanged control and actual mid-read
  file/root mutations. No fabricated reader, FileInfo or result.
- Limits: defaults100000 entries/depth64/1GiB per file/8GiB aggregate;
  actual lowered at/plus-one inventories for entries/depth/aggregate and one
  combined design-overlay budget. Real sparse1GiB+1 rejection before reading
  that file. No actual8GiB allocation claim. Actual16MiB-at/plus-one document,
  generated document and required-cover cases return bounded errors/nil bytes.
- Four wrappers Render/WithImplementation/package RunSelected/SelectRunAndNote:
  native subprocess fired no-fault controls precede real original mutation or
  verified-owned-copy symlink cleanup faults; errors/counter suppression,
  repeated-release disposition, sentinel survival and logical-path diagnostics
  all remain asserted. TMPDIR isolation precedes acquisition; no test-only
  production environment switch or exported injection API exists.

## AC5 observation versus composition

OBSERVED on412d01b: actual Render late-original mutation and owned-copy cleanup
faults return nil bytes plus real causes. Built standalone CLI success, ordinary
renderer-input/alias failures, exit1/empty stdout, historical/current distinction,
and complete-mode promotion all execute. These are conjunctive same-SHA evidence,
not an injected standalone CLI late-failure experiment.

Source closure for independent PM review:

1. internal/gates/attest.go:949 RenderAttestation buffers a document; line965
   joins render error, CheckUnchanged and Release; any error returns nil bytes.
2. internal/gates/suite.go:82 Release runs primary capability checks and
   supplementary lock guard, closes all captures/workspace/lock, joins failures,
   then latches/finalizes pending results exactly once. Cleanup does not short-circuit.
3. cmd/machinery/attest.go:120 invokes Render; its error branch reports and
   returns exit1 BEFORE the first document-output operation atline128.

Thus the late CLI guarantee is **COMPOSED**, subject to independent final source
review. Separately injected standalone CLI late failure is **UNOBSERVED**;
individual OS Close primitive failures are **UNFORCED**. Sink partialwrites are
separate and may physically truncate output while returning a nonzero result.
Hashes do not prove execution, reviewer identity or judgment correctness. The
exact approved scope-boundary text is in finalized Gv notes, generation help/
stderr and docs. Silent successful hooks remain silent.

## Authorized fixture amendment and future consumer

Two exact before/after hashes:

- hook_test.go: ab6986f14b95c942f4f627579b6f5ac0e34b475e4addcc64e293d9a66c292a92
  ->2b185970a3f69a140338cee97e14db0da82ef0aaf01250364357b104c1458b33.
- obligation_ownership_test.go:326e63f9bb4bbe12ecb5979cd3ad13fa6d83bbdd3924e835d96b929809d66e97
  ->cc5f5e12a0d41a6eeda60895fece31713c5f068817bb2d3f08b7f1c6ff4d4b04.

All original assertions and outside-helper bytes are preserved. The full13
caller boundary, actual complete62-file GoCRM/73-file parent/75-file formal
variant inventories, unchanged6 acceptance files, and exact evidence YAML
construction are independently audited in the proposal and PM disposition.
This final candidate replays those same13 original leaves successfully.
Added setup assertions require the actual sole plan-warning and current count0.
No current subject, new review, refreshed date/attestor/cover hash or fake
acceptance anchor is introduced. GoCRM runtime is not copied/executed; parent
Go tests prove their intended decision-ID coverage, not FSM conformance.

After acceptance, hgz1 consumes these exact helper bytes and full13-caller
boundary. Its source v2/current migration WILL invalidate the current v1 guards.
SrPM has explicitly assigned hgz1 only those two later helper adaptations,
requiring independent exact BEFORE-EDIT PM authorization at its actual baseline.
No future downgrade/manifest stripping/representation is preauthorized here.
No reverse p7 dependency or source-example mutation was introduced.

## Quality, immutable scope and known limits

`pvg story verify-tdd --base a82277af5650b487cea1260c24ffcc1c86d69d8d --json`:
11 commits,0 merges,0 violations. `frozen-test-sha256.txt` is byte-identical to
the approved7-file manifest; original raw failures are retained, not rewritten.
`git diff --check` and `gofmt -l <15 exact changed Go files>` produce empty output.

`pvg verify <all16 exact changed paths> --format text --include-tests` returns
FAILED:15 source files scanned,11 heuristic return-empty stub findings, all in
pre-existing hook.go branches at993,1069,1103,2646,2651,2656,2671,3965,3972,3976,3986.
This is NOT reported as a clean scan. `quality-provenance.json` proves every
entire containing function is byte-identical to a822 production base: nil-error
diagnostic handling, non-generated path classification, unavailable platform
change-ID fallback, and outside/unresolvable relative-path handling. These are
intentional existing return conventions outside the authorized stop hunks, not
new unimplemented work. No unrelated source was changed to silence the scanner.

One broader test remains SKIPPED, not passed:
TestSelectRejectsNonportableAndAliasedDesignPaths/case_folded_collision,
internal/gates/failclosed_io_test.go:149..162. It writes BUILD.md and build.md,
observes fewer than2 entries on this darwin/arm64 case-insensitive filesystem,
and takes its existing t.Skip atline158 before the collision assertion. The
source is unchanged. Final case-sensitive Linux CI/epic gate owes that native
leaf (e.g. `go test -count=1 -timeout=5m -json ./internal/gates -run
'^TestSelectRejectsNonportableAndAliasedDesignPaths$/^case_folded_collision$'`).
No filesystem/service setup or skip bypass is authorized/performed here. All
required attestation and caller scopes have0 skips. User-directed targeted scope
excludes full scripts/preflight.sh and unrelated integrations until epic final.
Bundled example/golden v1 migration belongs to hgz1, not production grandfathering.

Final actual scope:16 paths3396 additions+66 deletions=3462 changed lines.
`source-numstat.txt` gives every path's additions/deletions;
`source-sha256.txt` binds all16 files. Cost decomposition: production1184,
original frozen tests1489, supplemental tests566, docs158, last fixture repairs65.
Growth beyond older forecast is explicit held-root/copy/finalization proof plus
paired real-operation tests and exact legacy compatibility repairs, not factoring
or new APIs/seams. No proof was trimmed to fit a numerical forecast.

## Acceptance-criteria verification

| AC | Requirement | Implementation / proof | Developer result |
|---|---|---|---|
|1|Complete implementation/test scope, not only BUILD/pack; additions/removals/changes/narrowing stale|designlock/attestation_snapshot.go; gates/attest.go; frozen C scope/topology/narrowing plus new capability/limits tests|Observed PASS|
|2|Closed plan/current/history distinction in schema, diagnostics and docs|gates/attest.go; docs/attestation-evidence.md; B kind/schema, D plan/history, real CLI history and complete-warning tests|Observed PASS|
|3|Actionable compatibility migration, no behavioral grandfathering|A legacy unchanged/mutated tests, D compatibility, all13 authorized fixture callers; exact current0/plan warning retained|Observed PASS|
|4|Real negative/positive code/test/scope/alias/stale/evidence-only cases|frozen C gate/CLI/hook matrices plus real held-root/copy/limit and wrapper supplements|Observed PASS; unrelated Linux leaf remains owed|
|5|Real CLI attest/check, history/current separation, honest binding limits|same-SHA built CLI tests; actual Render late faults; source closure above; exact displayed limits|OBSERVED + COMPOSED; independent PM final review required|

LEARNINGS:

- Keep a current result provisional until every retained-root check and owned
  cleanup returns; an optimistic gate counter plus later custody error is unsafe.
- Snapshot copies do not replace original-root authority. Identity witnesses,
  full inventory and one overlay-inclusive budget are separate obligations.
- Fixture YAML roundtrips can silently change calendar scalars; exact guarded
  edits preserve provenance. A missing selector is not evidence of empty selection.
- A legacy fixture can be explicitly prospective with current0 and its warning
  intact; ordinary check/Stop success is not current review or complete-mode approval.
- Freeze new supplemental assertions only after their full planned matrix settles;
  preserve failed compile/setup history, and route every later existing-test edit.

## Evidence index and handoff

All files referenced without an absolute prefix are in
/tmp/machinery-p7jd-final-proof.O4NxfV. `evidence-sha256.txt` binds raw native runs,
stderr, source/profile/audit manifests, coverage profile and isolated binary.
This report is hashed separately. Prior stages remain in the original RED/PM
directories, /tmp/machinery-p7jd-green-proof.eHzGMx, and the separate fixture
proposal/PM directories; none is substituted for this final same-SHA proof.

Standalone product/dependency constraint, no installed replacements, no remotes,
main/epic/rebase/merge changes, no services/assets and no full preflight observed.
All work is committed to the story branch. PM must independently review the full
R2 contract and AC5 composition, known platform/quality findings and delivery
evidence before acceptance. Developer does not accept, close or merge.

## PM EXACT LEGACY-FIXTURE AMENDMENT ADJUDICATION — 2026-09-06 — 51454e0

TEST-EDIT AUTHORIZED: only the exact unapplied 65-addition/0-deletion patch below, SHA256 ae6e9241ea5d7b0c4556b2ee02c6e2910b2663c2452761e7984f1c6663252971, against committed 51454e069ebe4039f02d6d9108acf9354c7ad6c8. This is bounded GREEN-phase fixture-dispute adjudication, NOT delivery, final acceptance, rejection, RED reapproval, a status transition, or permission for other edits. Root must verify this durable disposition before resuming application. The earlier effective-application hold for unowned future incompatibility is resolved by the independently read-back SrPM canonical scope below; no implementation prerequisite or reverse dependency is invented.

WHY THESE EXACT REPAIRS:
Approved R2 (SHA256 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179) and AC5 clarification (SHA256 5fc3193106a803b7760020b646ed4db7b3d417f5e196b2266d10cf8227733937) remain unchanged, as do all five AC. Legacy v1 implementation claims MUST receive GV_MISSING_IMPLEMENTATION_SUBJECT. The original four broader failures are truthful old-fixture incompatibility, not a reason for production grandfathering: silent Stop's ledger assertion and obligation-free parent's explicit Gt control were initially unreached; the Policy/Isolation default positives failed solely on Gv while explicit positives and later negatives executed. Preserve all original logs at /tmp/machinery-p7jd-green-proof.eHzGMx, including PAUSED-BLAST-DISPUTE.md SHA256 d2837c4fbd4205ba151b856e96a7747d3cc32b58b1a69b4c49fe1d6fedb6bc1f and PAUSED-BLAST-TERMINAL.md. Earlier 2e897a4 date/empty-selection repairs and their setup history are separate authority, not permission inferred here.

EXACT FILE/BYTE BOUNDARIES:
1. internal/hook/hook_test.go, only the 36 added lines inside existing copyTree after unchanged copyDesignTree: before SHA256 ab6986f14b95c942f4f627579b6f5ac0e34b475e4addcc64e293d9a66c292a92; after SHA256 2b185970a3f69a140338cee97e14db0da82ef0aaf01250364357b104c1458b33. The original helper lines and all bytes outside this helper remain identical (outside-helper SHA256 af77e952bcaaf42801810c3e2c3be7a2b4a6a0970752118fac4b0eff666e64fc).
2. internal/gates/obligation_ownership_test.go, only the 29 added lines inside obligationParentFixture after unchanged CheckPack and before return: before SHA256 326e63f9bb4bbe12ecb5979cd3ad13fa6d83bbdd3924e835d96b929809d66e97; after SHA256 cc5f5e12a0d41a6eeda60895fece31713c5f068817bb2d3f08b7f1c6ff4d4b04. Original helper lines and outside bytes remain identical (outside-helper SHA256 6a3b9fe71dcb40c85c8c1ba4bd47a6ca6ef6ca3bdff161105552d87804cbf145).
No imports, assertions, skips, sibling tests, generic helper factoring, selectors, production/Ga/CLI policy, source examples, generated artifacts, acceptance records or other shared files may change under this authority. All seven frozen attestation test hashes remain unchanged; PM independently matched them to final-test-sha256.txt in its archive. New external observer/audit files are diagnostic-only and must NOT enter the story.

SUBSTANTIVE EVIDENCE HONESTY:
The helpers copy all original inputs first. Their exact count-one v1/header/claim/note guards fail closed on an unexpected source. Go CRM's copied evidence becomes 12 design/prospective-plan rows plus one ga.review-quality historical row; parent becomes eight plan rows. No current row or implementation subject is invented. The Go CRM gt note explicitly disclaims current implementation review and test execution; parent gt's existing prospective delegated-conformance note remains unchanged. Retained old dates/attestors/notes are copied fixture provenance, not a new review or refreshed endorsement. Explicit kind/note construction is justified for these temporary test meanings only; it does not validate the source example's old conformance claim or authenticate reviewer honesty. No cover hash/membership, date, attestor, acceptance anchor, source input or unrelated note is renewed. Ga's historical covers remain fully valid against the same six exact acceptance files.
Both helpers now require no Gv errors/drift, exactly one gt.conformance-test-shape plan-only/current-missing warning, and zero current-review count; Go CRM also requires historical count one. These checks strengthen fixture setup while preserving all original semantics. Ordinary default CLI nonblocking-warning success/platform-green and silent Stop are not current implementation approval or --complete success. Existing CLI warning promotion remains unchanged. Go CRM implementation is NOT copied or executed; MAC-uzxr's conformance defect is not hidden by manufactured current credit. Parent runtime tests still execute real Go tests for intended decision-ID coverage, not a substantive FSM review.

FULL CALLER BOUNDARY:
copyTree currently has six callers, each with crmDesign: TestStopGreenDesignClearsStateSilently; TestPreEditObligationSurvivesLostPostAndReplacementSession; TestWaveDeferralSurvivesCrashAndOutOfBandClose; TestStopDriftBlocks; TestStopWarnsWhenStagedImplGatesLackImpl; TestReapedStopStillGatesTheTree. The seven parent leaves are TestObligationParentSelectionRetainsRelationalCoverage/{Policy.oracle.md,Isolation.oracle.md}; TestObligationParentRealCLI/{obligation-free-parent-control,Policy.oracle.md,Isolation.oracle.md}; and unchanged GREEN consumer TestObligationParentDeletedRelationalOracleRemainsRequired/{gp,gn}. The latter existing green test file stays read-only. Review/replay must cover all 13 leaves, not just four old failing ones. The generic-looking name copyTree creates a real future coupling, addressed explicitly below; it is not broad authorization for additional fixture shapes.

INDEPENDENT PM PROOF:
Graph-first Verify review used ready main project Users-ramirosalas-workspace-machinery; previously observed generation 2026-09-06T02:42:16Z, main coverage missing both obligation story paths. Exact git-object source/caller search and complete input enumeration supplied missing/stale story coverage; graph absence was not treated as exhaustive proof.
PM exported exact git archive 51454e069ebe4039f02d6d9108acf9354c7ad6c8 into unique external /tmp/machinery-p7jd-pm-fixture.Mm3FM4; applied the reviewed patch ONLY there with apply_patch. No retained developer-worktree internals were inspected or written.
Commands from that archive:
- go test -count=1 -timeout=5m -json ./internal/hook -run 'TestStopGreenDesignClearsStateSilently|TestPreEditObligationSurvivesLostPostAndReplacementSession|TestWaveDeferralSurvivesCrashAndOutOfBandClose|TestStopDriftBlocks|TestStopWarnsWhenStagedImplGatesLackImpl|TestReapedStopStillGatesTheTree' -> 6 native leaves PASS, 0 FAIL/SKIP, 4.611s; pm-hook.jsonl SHA256 05ab2fce54a6b1c56fb480929d1a4bbfe631ba394d38f8486a027310b9d8e4c3.
- go test -count=1 -timeout=5m -json ./internal/gates -run TestObligationParent -> 7 native leaves PASS (10 test/pass events including parents), 0 FAIL/SKIP, 9.228s; pm-parent.jsonl SHA256 399b0c04b43d3f4901cbcf45b7473c65456df0e5c456d22d48493184d397cdaf.
- go build -o /tmp/machinery-p7jd-pm-fixture.Mm3FM4/machinery-diagnostic ./cmd/machinery -> exit0; isolated binary SHA256 953bcb3656ce2fae546bc129cea7e019bca70096b3ee9a2c309e535f24d15281.
- Separate external-only observer tests copied the fully inspected proposal observers with ONLY the output/binary-directory constant changed to this PM archive. go test -count=1 -timeout=5m -json ./internal/hook -run '^TestFixtureProposalHookInventory$' -> 1 PASS,0 FAIL/SKIP,0.962s; pm-observer-hook.jsonl SHA256 dfc4b26db854c95e932578ee739e4593d5530673f5a4987c38d8dbcbd1e7dc1b. Same command ./internal/gates -run '^TestFixtureProposalParentInventory$' -> 3 leaves PASS,0 FAIL/SKIP,3.850s; pm-observer-parent.jsonl SHA256 9cc39b942fe05ea35671665b2f4004b4b38e757d7a5390a3294d7ae9ce2a30f1. All four native stderr files empty.
Actual raw Gv reports current0, GoCRM historical1/parent historical0, and the exact sole warning. Actual Stop stdout is empty and both ledger flags false. All default/explicit parent controls exit0; missing Policy AUTHZ-21866c and Isolation TENANT-97e9c6 both exit1 with real Gt/oracle/ID diagnostics and no platform-green. Positive parent fixture Go execution also remains in the unchanged native tests.
- node /tmp/machinery-p7jd-pm-fixture.Mm3FM4/pm-audit.cjs -> exit0; independent report pm-audit.json SHA256 43773cab089ffd3ec3cc48411c32c5b39c0a3f277632f6e09e377555f2c091f0; script SHA256 c4ede818c39c558185ccc3528d31b758ef825db5a52c85c901069c13245c2db6. The first diagnostic audit used an incorrect expected path parent/impl/orders/coverage_test.go; actual unchanged helper writes parent/impl/coverage_test.go. That PM audit-only path typo was corrected and rerun, not a product/test failure.
The six independently generated input inventories reproduce the proposal inventories exactly. GoCRM68 entries/62 regular files, no implementation, no added/missing files; only attestations.yaml differs. Parent none93 entries/73 files, four original local module/code additions; Policy/Isolation96 entries/75 files, six original module/code/oracle/test additions. No baseline input is missing; only parent/design/attestations.yaml differs from copied baseline. Control-to-missing-ID input delta is solely parent/impl/coverage_test.go.
Reverse ONLY v2 header/kind insertions and the exact GoCRM gt note substitution and both generated attestation documents match the original git bytes completely. GoCRM YAML before503ede9141ba48ca7a1309a9054ec6e322357e8adcf260f67847c812f23ee90a/after0f75b667ce7107b39dabbbff138c496336a33839cb2e9877b97e5c8732aa3bf4; parent beforec30843d1fed9bab3691cb311acecfd452e8808f36fa8756aac0e1dfc90aa14e8/after5eaada36a4b494ef6ac414935504d52be8d93c3f82db42761941229cc8352c51.
All six Ga acceptance files are byte-identical to exact514 git: M0=1cce160669cd04f77bff60f1ae74e6760bdf358c08adeeddd6f03260ecc01339; M1=b0d518e198e5197de8e4ed132b3e58431a353a591670af867bc9590af6b11f63; M2=f4490e4d8d34a565985f98138a81b165e861bb75e02e2f4852cc868f030bdd2a; M3=11f6d1ee99d95e51f09d14457604f0b1c56ec91484d05ae6c72e5e7d649147df; M4=66593a6aca928f948dd22b9f46b249adadf8155dfa457984d31c07f03039704d; M5=64c90b5ccbe557ae0c04ed1cb619f5bc2c44e7258bb4a4c358107fbd1bf481c9.

SCOPE / KNOWN FUTURE INCOMPATIBILITY DECISION:
Independent source verification agrees with SrPM FINAL.md SHA256 4282fb85ed5ad07ebf1537cc7bdc455a5401459783a3aea86f802a349c709407: hgz1 will migrate BOTH copied sources to v2 and GoCRM gt to a substantively reviewed current record, invalidating these v1 guards. Narrow fail-closed guards are acceptable NOW only because the missing future ownership was explicitly repaired before this authorization. Independently read back p7 PRODUCES exactly16 conditional/current paths with measured conditional3462changedLOC and hgz1 exactly41possible paths, including only these same two helper bodies, accepted p7 construction/hash/full13-caller inventory CONSUMES and mandatory new exact BEFORE-EDIT PM authorization at the actual hgz1 baseline. SrPM scope-only terminal contracts remain independently distinguishable. There is no authorization today for a future v2/current downgrade, automatic stripping of an implementation manifest, refreshed attestor/date/hash, new fixture framework, or selected future construction. hgz1 must resolve any provenance/representation conflict explicitly and preserve all13 tests and current0/plan-warning semantics; accepted p7+uzxr+lhu5 precede hgz1, then lnu6. p7 has no reverse dependency or source-example obligation.
Cost independently verified by git diff --numstat a82277af5650b487cea1260c24ffcc1c86d69d8d 51454e0:14paths3331add66delete=3397changed. Exact patch adds2paths65add0delete ->16paths3396add66delete=3462changed. Bounded increase is direct no-grandfathering fixture compatibility plus mandatory paired proof, not permission to trim proof or expand more files. Updated scope is not final implementation acceptance.

APPLICATION AND NEXT PROOF:
Author must apply ONLY the exact patch in a separate commit whose subject contains literal [test-edit-authorized], verify the two before/after hashes, preserve all seven frozen hashes and original logs, and record the resulting full source/test hash inventory plus exact commit SHA. No unstated follow-up repairs. If guards unexpectedly fail, source differs, any new semantic failure occurs, or desired hunk differs: pause for a new exact review.
Mandatory fresh same-revision scoped tests, targeted race proof, all13 caller replays and relevant broader regression commands are owed after repair. Repeat the recorded Attest/Attestation four-package scopes and broader selectors in PAUSED-BLAST-DISPUTE, with raw exact native names, pass/fail/skip/cause counts and bounded runtimes; rerun normal TDD range audit with all marked repairs. Reconcile every skip/failure explicitly. Existing macOS case-fold skip is not waived or passing proof here. Prior229scoped/45race PASS at4b246ba, docs-only51454e0, and this external patched diagnostic are distinct revisions/artifacts: do not combine them as a delivered same-SHA run. No full preflight now. AC5 composed renderer/real CLI/source closure still requires independent final review; injected standalone late CLI remains UNOBSERVED and OS close failures UNFORCED, not silently proven. No acceptance of product behavior, docs, current review, or whole-story completion is issued.

EXACT AUTHORIZED PATCH:
```diff
diff --git a/internal/hook/hook_test.go b/internal/hook/hook_test.go
index faf0172..cb4cf1b 100644
--- a/internal/hook/hook_test.go
+++ b/internal/hook/hook_test.go
@@ -167,6 +167,42 @@ func copyTree(t *testing.T, src, dst string) {
 	if err := copyDesignTree(src, dst); err != nil {
 		t.Fatal(err)
 	}
+	// These Stop fixtures review design/ledger behavior, not the CRM runtime.
+	// Preserve cover bytes and historical acceptance anchors; explicitly recast
+	// the legacy implementation claim as unfulfilled test-plan intent.
+	path := filepath.Join(dst, gates.AttestationsFileName)
+	raw, err := os.ReadFile(path)
+	if err != nil {
+		t.Fatal(err)
+	}
+	text := string(raw)
+	replace := func(old, next string) {
+		if strings.Count(text, old) != 1 {
+			t.Fatalf("fixture migration needs exactly one %q", old)
+		}
+		text = strings.Replace(text, old, next, 1)
+	}
+	replace("attestation_version: 1\n", "attestation_version: 2\n")
+	for _, claim := range []string{
+		"g2.action-ownership", "g2.interface-contract-rightness", "g2.placement-rightness",
+		"g2.adoption-closure-discovery", "g2.event-contract-completeness", "g2.nfr-content",
+		"g3.guard-semantics", "g3.invariant-enforcement", "g3.residual-transitions", "g3.event-redelivery",
+		"gt.conformance-test-shape", "g4.zero-context", "ga.review-quality",
+	} {
+		kind := "plan"
+		if claim == "ga.review-quality" {
+			kind = "historical"
+		}
+		line := "  - claim: " + claim + "\n"
+		replace(line, line+"    kind: "+kind+"\n")
+	}
+	replace("    note: The Go tests key executable table cases on every stable oracle id and assert next state plus ordered actions.\n",
+		"    note: Fixture plan only; conformance tests are intended to cover every committed oracle row and assert next state plus ordered actions. No current implementation review or test execution is claimed.\n")
+	writeFile(t, path, text)
+	g := gates.CheckAttestations(dst)
+	if len(g.Errs) != 0 || len(g.Drift) != 0 || len(g.Warns) != 1 || !strings.Contains(g.Warns[0], "gt.conformance-test-shape: plan only; current implementation review missing") || g.Counts["current implementation reviews"] != 0 || g.Counts["historical review records"] != 1 {
+		t.Fatalf("fixture must retain missing-current warning and historical evidence: %+v", g)
+	}
 }
 
 // copyDesignTree takes a governed reader snapshot before copying a shared
diff --git a/internal/gates/obligation_ownership_test.go b/internal/gates/obligation_ownership_test.go
index f4ca7f7..b5c4eed 100644
--- a/internal/gates/obligation_ownership_test.go
+++ b/internal/gates/obligation_ownership_test.go
@@ -350,6 +350,35 @@ func obligationParentFixture(t *testing.T) (string, string) {
 		t.Fatalf("complete decomposition fixture invalid: %v", err)
 	}
 	requireObligationClean(t, CheckPack(design))
+	// The parent manifest delegates runtime conformance to its children. These
+	// local tests prove decision-ID ownership, not substantive FSM conformance.
+	// Keep that claim plan-only, without changing any required design covers.
+	path := filepath.Join(design, AttestationsFileName)
+	raw, err := os.ReadFile(path)
+	if err != nil {
+		t.Fatal(err)
+	}
+	text := string(raw)
+	replace := func(old, next string) {
+		if strings.Count(text, old) != 1 {
+			t.Fatalf("fixture migration needs exactly one %q", old)
+		}
+		text = strings.Replace(text, old, next, 1)
+	}
+	replace("attestation_version: 1\n", "attestation_version: 2\n")
+	for _, claim := range []string{
+		"g2.action-ownership", "g2.interface-contract-rightness", "g2.placement-rightness",
+		"g2.adoption-closure-discovery", "g2.event-contract-completeness", "g2.nfr-content",
+		"gt.conformance-test-shape", "g4.zero-context",
+	} {
+		line := "  - claim: " + claim + "\n"
+		replace(line, line+"    kind: plan\n")
+	}
+	writeSuiteFile(t, path, text)
+	g := CheckAttestations(design)
+	if len(g.Errs) != 0 || len(g.Drift) != 0 || len(g.Warns) != 1 || !strings.Contains(g.Warns[0], "gt.conformance-test-shape: plan only; current implementation review missing") || g.Counts["current implementation reviews"] != 0 {
+		t.Fatalf("parent fixture must remain an unfulfilled conformance plan: %+v", g)
+	}
 	return design, impl
 }
```


## nd_contract
status: in_progress

### evidence
- PROPOSAL ONLY, UNAPPLIED, NOT TEST-EDIT AUTHORIZATION: exact test-local fixture amendment against clean51454e069ebe4039f02d6d9108acf9354c7ad6c8. Shared source/tests/examples unchanged; no delivery or RED approval. Independent PM review required before any [test-edit-authorized] commit.
- Index /tmp/machinery-p7jd-fixture-proposal.geNVdA/PROPOSAL-INDEX.md SHA2561049ea7adf1eaa142948f2f6cd3d4d8327ae20a9b3326222f57041f7f19c9169. Exact fixture-amendment.patch SHA256ae6e9241ea5d7b0c4556b2ee02c6e2910b2663c2452761e7984f1c6663252971,65 additions0 deletions in only internal/hook/hook_test.go copyTree and internal/gates/obligation_ownership_test.go obligationParentFixture. Proposed total16 paths3462 changed lines, not approved scope.
- External archive diagnostic only: all13 original affected-helper leaves PASS0FAIL0SKIP (6hook4.704s,7parent9.320s). Exact original assertions/outside-helper bytes unchanged; zero deleted original lines. Raw Gv explicitly retains missing-current warning and current_reviews0. Actual Stop stdout empty/ledger cleared. Actual parent default+explicit positive controls exit0 and removed-ID challenges exit1 with original assertions. These are NONQUALIFYING external observations, not same-SHA shared story proof.
- Full before/after evidence YAML, all6 acceptance-byte hashes, full62-file GoCRM design inventory (implementation absent),73-file complete parent control and75-file Policy/Isolation control/missing-ID inventories, raw CLI/Stop outputs, exact native names, audit scripts and provenance in that directory. Only copied evidence changes; cover hashes/attestor/date unchanged; ga historical and all acceptance anchors unchanged. No GoCRM runtime/parser or parent decision-ID synthetic code is claimed as current substantive conformance.
- Precise forward consequence: hgz1 planned source v2 migration WILL fail both exactv1 fixture guards. Its later GoCRM current row is not this fixture's current evidence. PM/Sr PM must explicitly route future exact amendment/decoupling authority for these2 constructors before hgz1 regression completion; its present scope does not own these files. No automatic broadening or reverse dependency proposed. This65-line proposal is intentionally unchanged pending judgment.
- Earlier4 broader failures,1 filesystem skip,229 scoped+45race passes at4b246ba remain preserved and unresolved shared-state evidence. Clean shared51454e0; all finite diagnostics complete; no background process, install, asset/service operation, remote, preflight, merge or production dependency added.

### proof
- [ ] AC #1: prior observations retained; final independent review and same-SHA replay pending.
- [ ] AC #2: exact proposal preserves honest plan/current/history distinction; no shared repair authorized yet.
- [ ] AC #3: four existing fixture conflicts have an externally demonstrated construction; PM must adjudicate exact patch and forward hgz1 boundary.
- [ ] AC #4: prior held-root/hook/race proof retained; existing unrelated platform skip remains unresolved.
- [ ] AC #5: prior composed renderer/CLI observations retained; final same-SHA verification and independent source closure remain mandatory.

## nd_contract
status: in_progress

### evidence
- Clean paused head51454e069ebe4039f02d6d9108acf9354c7ad6c8; last commit docs-only corrects `machinery check --design design --impl src --gate gv` to `machinery check design --impl src --gate gv`. Docs SHA256d7ca4cc79a2af87e6946dba3bba445b21cee9639da13309ec2384033d0a8ed41. Full source/test implementation remains4b246baf7f9384f373d4935f01bbe07c7fcf7076;229 scoped and45 race leaves passed with0 failures/0 skips on4b246ba, not claimed as a51454e0 replay. Fresh same-SHA proof is owed on eventual delivery.
- Full pause report /tmp/machinery-p7jd-green-proof.eHzGMx/PAUSED-BLAST-DISPUTE.md SHA256d2837c4fbd4205ba151b856e96a7747d3cc32b58b1a69b4c49fe1d6fedb6bc1f preserved and appended in Notes. Raw/native inventories/hash files there bind4 existing fixture conflicts,1 existing filesystem skip, all passing observations and failed command histories. Final scope14 paths3331 additions66 deletions=3397 unchanged.
- RED-DISPUTE remains: internal/hook/hook_test.go TestStopGreenDesignClearsStateSilently uses examples/go-crm/design; internal/gates/obligation_ownership_test.go TestObligationParentRealCLI/{obligation-free-parent-control,Policy.oracle.md,Isolation.oracle.md} uses examples/checkout-split/parent/design. Their unchanged v1 gt.conformance-test-shape rows correctly fail GV_MISSING_IMPLEMENTATION_SUBJECT. No grandfathering/fixture/helper/example/Ga/selector change authorized or made. Root routes migration ownership independently.
- Broad native replay: gates33PASS0FAIL1SKIP (existing case_folded_collision on case-insensitive filesystem); hook23PASS1FAIL0SKIP; accepted-stories76PASS3FAIL0SKIP; designlock13PASS0FAIL0SKIP. No skipped case credited. No active/background process, no delivery/acceptance/merge/install/remote/preflight operation.

### proof
- [ ] AC #1: scoped v2/current/plan/history observations pass; independent overall review pending.
- [ ] AC #2: full inventory/digest/kinds/docs observations supplied; final same-SHA proof pending.
- [ ] AC #3: required scope migration passes;4 broader legacy fixture conflicts need exact ownership/repair authority.
- [ ] AC #4: real capability/limits/hook and45 race observations pass; broader fixture/platform-skip resolution pending.
- [ ] AC #5: observed Render late faults return nil bytes; built CLI ordinary success/failure observed; composed source closure recorded; independent review and final same-SHA replay pending. Standalone CLI late injection UNOBSERVED, individual OS Close errors UNFORCED, sink partialwrites separate.

# MAC-p7jd GREEN paused: legacy fixture migration outside ownership

Candidate: 4b246baf7f9384f373d4935f01bbe07c7fcf7076, story/MAC-p7jd.
Production base: a82277af5650b487cea1260c24ffcc1c86d69d8d.
Worktree: /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-p7jd.
Clean worktree; all bounded native processes completed. No delivery, acceptance,
closure, merge, rebase, remote operation, install, preflight or example mutation.

## Exact dispute

Four existing, read-only regression leaves assume that unchanged example v1
implementation-behavior attestations remain green. The approved MAC-p7jd contract
instead requires GV_MISSING_IMPLEMENTATION_SUBJECT for every such v1 row, with
no grandfathering. Their semantic fixtures now conflict with that requirement:

1. internal/hook/hook_test.go:833 TestStopGreenDesignClearsStateSilently copies
   examples/go-crm/design through copyTree, records a design obligation, and
   expects silent Stop before checking ledger clearing. Its exact failure at
   line840 is one ordinary mid-phase systemMessage reporting one gate ERROR.
   The ledger assertion after Fatalf is NOT reached in this leaf.
2. internal/gates/obligation_ownership_test.go:395 TestObligationParentRealCLI:
   obligation-free-parent-control fails line423 on its default-gate positive
   control; the subsequent explicit zero-obligation Gt assertions are NOT reached.
3. The same test's Policy.oracle.md leaf reports line445 default-gate failure.
4. The same test's Isolation.oracle.md leaf reports the same default-gate failure.
   For 3/4 the actual parent Go tests execute, explicit Gt positive controls run
   successfully, and default gates fail solely on legacy Gv. Existing later
   negative assertions are retained and were allowed to execute (Errorf, not Fatalf).

obligationParentFixture copies the complete examples/checkout-split tree, then
creates local module/dependency/code inputs. Its parent/design/attestations.yaml
is unchanged v1 with gt.conformance-test-shape. Default parent selection remains
the accepted source behavior; no selector expansion or production amnesty is justified.
All reads-consumer leaves in this replay pass.

Direct actual candidate command `go run ./cmd/machinery check
examples/go-crm/design --gate gv` shows exactly one blocking error:
GV_MISSING_IMPLEMENTATION_SUBJECT: gt.conformance-test-shape has legacy design-only
covers; review the complete implementation/test scope and run machinery attest
with explicit current kind/root/reviewer/date, or explicitly recast as v2 plan.
All plan/history migration notes remain informational.

No repair is proposed as approved. Review must select exact test-local fixture
migration hunks or a separately scoped example migration. All existing assertions,
helpers, accepted semantics and examples remain untouched pending exact authority.
The two newly added supplemental test files are frozen at their first addition
commit 4b246ba; this dispute does not license changing them.

## Same-revision observations (all at 4b246ba)

Each scoped command uses `go test -count=1 -timeout=5m -json`:

| Package / selector | Leaves | Pass | Fail | Skip | Native seconds |
|---|---:|---:|---:|---:|---:|
| internal/gates / Attest | 151 | 151 | 0 | 0 | 19.303 |
| cmd/machinery / Attest | 28 | 28 | 0 | 0 | 16.104 |
| internal/hook / Attestation | 28 | 28 | 0 | 0 | 7.931 |
| internal/designlock / Attestation | 22 | 22 | 0 | 0 | 2.004 |

229/229 scoped leaves pass, including all 189 frozen leaves and 40 new leaves.
Original baseline47PASS142FAIL0SKIP and all compile/setup failures remain preserved.
The original A/B/C/D native names remain in final-*-native-leaves.json. Every C
control/challenge now executes in GREEN; the16 schema mutation legs that only
appeared green through unreachable baseline setup now actually execute. The prior
complete/date and empty/Gl fixture disputes are resolved by the exact separately
authorized commit2e897a4; their earlier failures remain setup history.

Targeted race replay uses the same command plus `-race`:

- internal/designlock / Attestation: 22PASS0FAIL0SKIP, 2.744s.
- internal/gates / TestAttestGreen|TestAttestationCRelease: 23PASS0FAIL0SKIP, 31.065s.

Broader read-only replay is NOT clean proof:

- gates / TestSelect|TestRunSelected|TestCargoWorkspacePointerMutation|TestGateSnapshot:
  33PASS0FAIL1SKIP, 3.282s. The existing
  TestSelectRejectsNonportableAndAliasedDesignPaths/case_folded_collision skips
  on this case-insensitive filesystem. No skip was added or credited as proof.
- hook / TestStop|TestSelectGates|TestGreenStop: 23PASS1FAIL0SKIP, 4.958s.
- gates / TestObligation|TestReadsConsumer: 76PASS3FAIL0SKIP, 11.689s.
- designlock / TestMaterializeDesignWorkspace|TestExternalTreeSnapshotCleanup|
  TestRegularFileSnapshotCleanup|TestExternalSnapshotRejects|TestUniversalSnapshotBoundary:
  13PASS0FAIL0SKIP, 0.693s.

## New proof reached

Capability tests use actual original held roots across4 topologies, full entries,
design overlay hash, private copy hash, defensive Entries copy, repeated Close,
released CheckUnchanged/Materialize rejection. Real original-file mutation,
same-byte restored-mtime design file identity replacement, original/design root
renames, private-copy mutation and post-reader/pre-capture design generation gap
all fail custody after matched unchanged observations.

Existing read-chunk callback runs on real2*64KiB file copy; unchanged, actual file
mutation and root rename/replacement cases execute (no fake FileInfo/readers).
Factory defaults assert100000 entries/depth64/1GiB/8GiB. Lowered actual-tree
at/plus-one entry, depth and aggregate controls execute, including one10-byte
aggregate over5-byte implementation+5-byte design overlay. Actual sparse1GiB+1
file is rejected before that file's read callback. No real8GiB-tree claim.

Four real wrappers Render, WithImplementation, package RunSelected, and
SelectRunAndNote each execute native subprocess no-fault/original/owned-copy
cleanup cases. Every fault invocation first executes its fired matched no-fault
control. Only the approved before-final-Release callback mutates real filesystem
objects, never results or lifecycle handles. Cleanup swaps a verified owned
BUILD.md for a sentinel symlink under child-private TMPDIR. Callbacks fire once,
current counters are still absent before release, final faults suppress them,
repeated Release is latched, outside sentinel survives, actual cleanup/symlink
cause is exposed without private paths, and released Snapshot APIs fail closed.

Actual Render input/output document and design-cover16MiB-at/plus-one cases
execute, with nil bytes and GV_EVIDENCE_LIMIT at plus-one. Exact v2 claim/date/
cover-hash whitespace mutations reject after current success controls.

AC5 OBSERVED: Render late original mutation and owned-copy cleanup return nil
bytes plus actual errors; built CLI success and ordinary renderer-input/alias
failures execute, returning exit1/empty stdout on failures. COMPOSED source
closure: internal/gates/attest.go:949 Render buffers then line965 errors.Join of
render/check/Release errors; nonnil returns nil bytes. suite.go:82 Release checks
strict originals, supplementary lock, closes capabilities/workspace/lock, joins
errors and finalizes once. cmd/machinery/attest.go:120 receives Render; line123
error branch reports and returns1 before line128 output. Independent PM source
review remains required. Standalone CLI injected late failure UNOBSERVED;
individual OS Close primitive failures UNFORCED. Sink partialwrites are separate.

## Remaining owned work / diagnostics

The newly written docs line in the generation example mistakenly spells
`machinery check --design design --impl src --gate gv`; check's design path is
positional. It must become `machinery check design --impl src --gate gv` in a
docs-only correction after pause direction, followed by same-revision proof.
An initial diagnostic repeated this invalid flag; its stderr is preserved as
blast-hook-example-gv.*, then the corrected successful diagnostic execution is
preserved separately as blast-hook-example-gv-corrected.*. This is not product
failure and not authorization to change check's CLI grammar.

Quality scan `pvg verify <14 exact owned paths> --format text --include-tests`
scans13 source files (docs ignored), reports11 return-empty stub heuristics in
existing hook.go lines993,1069,1103,2646,2651,2656,2671,3965,3972,3976,3986.
All lie outside this story's changed stop hunks1318..1386; no new stub finding.
This is not reported as a clean scan or repaired by unrelated source edits.
TDD range audit checks9 commits,0 merges,0 violations. An initial invocation
erroneously included story positional ID and was rejected; preserved stderr,
then documented range-only and base/json invocations succeeded.

## Measured scope and immutability

14 exact owned paths,3331 additions+66 deletions=3397 changed lines;87 above the
upper forecast3310. Decomposition: production1184, frozen tests1489 (including
two exact fixture repairs), new tests566, docs158. New tests grew191 beyond the
375-line upper estimate to keep real paired subprocess operations, full root/
limit matrix, explicit cleanup/identity assertions, and document boundaries.
No generic helper factoring, new public API/seam, or ownership expansion.

| Path | Add | Delete |
|---|---:|---:|
| cmd/machinery/attest.go | 43 | 1 |
| cmd/machinery/attest_implementation_test.go | 464 | 0 |
| cmd/machinery/attest_test.go | 1 | 1 |
| cmd/machinery/check.go | 2 | 2 |
| docs/attestation-evidence.md | 140 | 18 |
| internal/designlock/attestation_snapshot.go | 464 | 0 |
| internal/designlock/attestation_snapshot_test.go | 310 | 0 |
| internal/gates/attest.go | 514 | 13 |
| internal/gates/attest_green_test.go | 256 | 0 |
| internal/gates/attest_implementation_test.go | 612 | 0 |
| internal/gates/attest_test.go | 48 | 2 |
| internal/gates/suite.go | 98 | 22 |
| internal/hook/attestation_snapshot_test.go | 361 | 0 |
| internal/hook/hook.go | 18 | 7 |

All5 approved frozen hashes and both new-file hashes are final-test-sha256.txt.
All14 source hashes are paused-final-source-sha256.txt. Raw stdout/stderr,
exact native leaf inventories and elapsed times use final-*, race-* and blast-*
files in this same evidence directory; paused-blast-evidence-sha256.txt binds
raw JSON and the4 disputed input source files. paused-build-profile.txt records
Go/platform/CGO settings. Prior RED and failed intermediate outputs are retained.

## nd_contract
status: in_progress

### evidence
- Candidate4b246ba;229 scoped leaves and45 race leaves pass, no skips in either.
- Broader replay exposes4 out-of-scope legacy fixture conflicts and1 existing
  filesystem-conditional skip. No repair or delivery asserted.
- Exact authorization needed before any existing fixture/helper/example change.

### proof
- [ ] AC #1: v2 grammar/current/plan/history observations pass; pending overall review.
- [ ] AC #2: full scope inventory/hash observations pass; pending overall review.
- [ ] AC #3: suite/current/history/complete behavior passes; legacy regression fixture authority unresolved.
- [ ] AC #4: held-root/copy/bounds and hooks observed; broader fixture/skip unresolved.
- [ ] AC #5: generation and composed finalization observations recorded; docs command correction and independent source closure remain.

## nd_contract
status: in_progress

### evidence
- Clean WIP390d4dc; two exact frozen-fixture disputes above; full report/raw leaf history retained. No test edits or repair authority assumed.
- Required next step independent PM exact fixture repair decision; mandatory remaining implementation, docs and new supplemental proof remain pending.

### proof
- [ ] AC #1: final held-root/copy/limits/same-revision proof pending
- [ ] AC #2: complete fixture and final docs pending
- [ ] AC #3: reached migration cases require final replay/review
- [ ] AC #4: disputed empty fixture and mandatory supplements pending
- [ ] AC #5: complete-mode and same-revision renderer/CLI/source-closure conjunction pending

## nd_contract
status: red-approved

### evidence
- RED tests approved via pvg story approve-red on 2026-09-05.

### proof
- [ ] GREEN developer must implement against the approved RED tests without modifying them.


# Independent PM RED review — MAC-p7jd

Decision: APPROVED for RED transition only. The frozen suite together with the explicit mandatory GREEN supplements and same-revision source/proof bars can prove the approved story. This is not product acceptance, implemented custody proof, or permission to modify frozen tests.

Reviewed candidate 47ba44906a09bc2fa010092a86b133d0d749c52c against unchanged production base a82277af5650b487cea1260c24ffcc1c86d69d8d in the independent detached checkout /tmp/machinery-p7jd-pm-red.fnQSrl/checkout. Original RED 7ec5d609acc1597ee2c0ddbf5401b92f834d5ab3 remains intact. Both commits carry tdd-red and [test-edit-authorized]. Canonical pvg issues show MAC-p7jd --json current body, exact PRE-RED and POST-FREEZE authorizations, approved AC5 clarification, complete author RED-REPORT.md, latest terminal delivered contract and SrPM 03:21:32Z cost-only canonicalization reviewed. Historical contracts were not treated as current authority.

## Scope, repair and evidence integrity

Full changed-source review: internal/gates/attest_implementation_test.go (612 lines); cmd/machinery/attest_implementation_test.go (462); internal/hook/attestation_snapshot_test.go (361); exact full diffs of both existing attest_test.go files. Total five tests, 1484 additions/3 deletions. No production, example, golden, dependency, documentation or generic helper edits. All original existing-test edits match prior authorization: wrong version 2 to 3 plus integer-1-or-2 diagnostic and separate v2 cases; os.Link skip to mandatory Fatal only. The subsequent two-file repair is exactly the checked two destination MkdirAll loop and hook Hooks:true plus real Load(root) no-warning check before ledger arming. Other three frozen hashes remain identical. Exact repair patch SHA256 2292e7da2123bdbc3890564468d52853c53306aa44e2efd0c9f60ef184e2a5e4.

Author report SHA256 independently matched 17843fb0be042d7a0b661933611de53cd3cfc7c6666aad0a7c9012b32ec15c70. Approved R2 proposal hash independently matched 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179; AC5 clarification hash matched 5fc3193106a803b7760020b646ed4db7b3d417f5e196b2266d10cf8227733937. Author raw repaired logs/leaf inventories match their recorded hashes. Normalizing their package/name/action TSV against independent action/name TSV produces identical names and terminal outcomes for every selection; elapsed columns intentionally differ.

Graph-first codebase-memory review refreshed list_projects/index_status: ready, main 497419ab4512fcff765cd5feb27aed4c67b5608d, generation 2026-09-06T02:42:16Z. search_graph located Config, Load and stop in internal/hook/hook.go and copyDirInto in cmd/machinery/golden_test.go; all seven results read, has_more false. check_index_coverage covered the five changed tests, both attest.go, suite.go, hook.go, golden_test.go and designlock scope. Existing cited files metadata_match/no recorded issue; three new test files missing from main graph. This is best-effort, not completeness proof. Exact candidate source read from detached committed checkout supplied ground truth, including Config/Load/copyDirInto and stop ordering. Source confirms original hooks:null parsing and absent destination setup defects, and current hook deferred Release after possible emission/ledger clear. New optional writer method does not manufacture verdicts or activate itself.

## Independent synchronous replay

Every command ran from /tmp/machinery-p7jd-pm-red.fnQSrl/checkout on Go go1.27.1 darwin/arm64, with shell time and exact exit capture. No service or remote was used. All four invocations terminated synchronously.

1. go test -count=1 -timeout=5m -json ./internal/gates -run Attest
2. go test -count=1 -timeout=5m -json ./cmd/machinery -run Attest
3. go test -count=1 -timeout=5m -json ./internal/hook -run Attestation
4. go test -count=1 -timeout=5m -json ./internal/designlock -run Attestation

| Selection | Native leaves | Pass | Fail | A genuine fail | B interface/diagnostic fail | C family control unavailable | D pass | Starts/terminals | Exit | Shell wall seconds |
|---|---:|---:|---:|---:|---:|---:|---:|---|---:|---:|
| gates Attest | 133 | 31 | 102 | 18 | 21 | 63 | 2 | 142/142 | 1 | 7.929 |
| CLI Attest | 28 | 11 | 17 | 6 | 1 | 10 | 2 | 29/29 | 1 | 4.390 |
| hook Attestation | 28 | 5 | 23 | 0 | 4 | 19 | 4 | 30/30 | 1 | 4.380 |
| designlock Attestation | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0/0 | 0 | 0.667 |
| Total | 189 | 47 | 142 | 24 | 26 | 92 | 8 | 201/201 | — | — |

Zero skips, no compilation/import/setup/runtime/timeout failure. designlock prints `testing: warning: no tests to run`: zero capability coverage, not a passing capability test. No -cover run; line/branch coverage NOT MEASURED. Shell wall time was returned by time to the tool transcript; stderr artifact files are not claimed to contain shell timing. Native package JSON event windows: gates 2026-09-05T20:23:42.039071-07:00 to 20:23:48.584112-07:00; CLI 20:23:50.056936 to 20:23:53.017055; hook 20:23:53.519814 to 20:23:57.429602; designlock 20:23:57.865657 to 20:23:58.150333.

The real CLI was built privately by the frozen test with `go build -o <t.TempDir>/machinery .`; independent binary digest 08151cca48bfd3d66886f1139a77f5d196552a87d99930d7ff038ac122759e1d is recorded in cli.jsonl. Binary/build paths can affect its digest versus the author's build. Tests use bounded real process/Git calls and isolated existing test harness state; no mocks or substituted runtime.

## Actual causes and test strength

A: 18 gates leaves execute three legacy behavioral claims in unchanged/mutated variants through direct, suite-with-impl and suite-without-impl routes. All report errors=[] and one attested claim; the required GV_MISSING_IMPLEMENTATION_SUBJECT assertion fails. Six real CLI leaves return code 0 instead of required 1 for the same three claims/variants. Handler/assertion mutations occur on real fixtures before checks. These 24 are qualifying existing-interface behavioral RED. The compatibility control is separate, never the unsafe legacy success.

D: two gates full/partial six-g2 controls pass preserving claims and 0/5 warnings; two CLI controls preserve exact filehash/claims precedence/legacy plan and actual local Git ancestor acceptance; four valid configured hook semantic/wave/Stop/SubagentStop controls pass. Additional passes are 28 existing gates leaves, the independent digest fixture vector, nine existing CLI leaves and one existing hook selection control. The digest-vector pass is fixture grammar verification, not product capability.

B/C: baseline v2 rejection is `attestation_version must be the integer 1`; generation is `unknown flag: --design`; hook callback firing is zero. All 63 gates C cases abort at valid-v2 control, ten CLI C cases abort at first generation, and 19 hook C-family cases abort in matched no-fault callback control. Their actual mutations/cleanup are not credited. Sixteen B malformed-schema challenge legs also abort at the valid plan control. B/C are frozen desired behavior, not extra observed staleness or custody failures. The repaired complete CLI fixture now copies real go-crm data, initializes/commits local Git, rewrites acceptance anchors and reads old claims; its first g2.action-ownership generation fails on --design. It has not established an unchanged --complete pass or isolated final warning, and approval does not promise that later fixture prerequisites are valid.

Reviewed assertions require independent complete inventory/digest including hidden/ignored/vendor/build/config inputs; additions/removals/renames, assertion removal, handler/config/mode changes; all four design/impl topologies and exact evidence exclusion; forged narrowing with and without recomputed digest; symlink/hardlink/FIFO/nested metadata; schema/kind/missing-root; provisional counter suppression, two pending runs, final release publication/failure, latching after restoration, repeated release and post-release refusal. C cases require the valid control before challenge. No skip-if-missing, accept-any-error or baseline-vs-GREEN pass branch exists. Existing helper returned nil after Fatal is unreachable, not a production stub.

Real CLI coverage binds generation to independent filesystem inventory/digest checks, stale assertion/handler/config/addition, missing root, ordinary real renderer missing-input and hardlink alias errors with exact exit 1/empty stdout, real Ga/current stale replay including old --commit, reserved evidence-only commit, ordinary/warnings-as-errors and sole-warning --complete controls. Hook cases require the exact callback, no prior output, real operation, matched no-fault child-process control, one block, retained ledger, suppressed counters, concrete cleanup/symlink cause and intact outside sentinel across Stop/SubagentStop and strict/relaxed/wave policies; empty selection and existing semantic-warning/wave behavior are included. Faults remain unobserved until GREEN.

## Frozen SHA256 acceptance bar

- internal/gates/attest_implementation_test.go: fcae6e3a9dc66d8fb9abb1d151129604f59a3e38d9ac221d4f16ce91a8086260
- internal/gates/attest_test.go: f0eef53c28fdf940654700832887f2c326a478c5df1d741ae9f538a3d24db253
- cmd/machinery/attest_test.go: beb2d58af4d3144316f3789c3b7cbb726a4761af57ea85f0017297b9c24cd6cd
- cmd/machinery/attest_implementation_test.go: e02ad18fe2745d23f05e69ce1e01dd2b1be358f94d99cb3a215b14f6e2cee3b8
- internal/hook/attestation_snapshot_test.go: a3c088c095e2dba4379f548903a058eedbd5338ef872e507ab12edaa1ca06e49

Independent raw JSONL SHA256 under /tmp/machinery-p7jd-pm-red.fnQSrl:

- gates.jsonl: 8741817e07aec80322ef798fb9b255562e8a009da4c90d6fc51fcfe1ac5ed347
- cli.jsonl: 77971a1fc1d3e46b2f9242d5b62a0ee1fe87b2b6342c98c5a5cde7d9ab96f6a0
- hook.jsonl: a88930673c853b49e0a2d8df49668c7da044cfca757cb1ae27f6fbbf2b6407fd
- designlock.jsonl: 2c204e18e49d4e1e28b2d4a7d4e283f20e28bff73dda95a9a3e9a52f1226403c

Exact native leaf names, terminal actions and per-leaf elapsed values are in gates-leaves.tsv (4460a2be7ca1922de70e50062863712abb83a63480d11f7f7fac72d1a3ab1b36), cli-leaves.tsv (1e3a93ffc10db02d452da8dc44aae310ee8703ae6c5ae024d292d0db81e80c59), hook-leaves.tsv (235ec2339e3bc561e5db42b17ecc8099ec365b69fee72eb636bd48bd384362d6), designlock-leaves.tsv (empty e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855). Inventory extraction used unique native run names and terminal pass/fail/skip names with no descendant prefix; parent totals are not duplicate leaves. Hook child output is embedded in its 23 parent scenario leaves.

## Mandatory GREEN bar, unchanged

All five frozen files must remain unchanged and every frozen test pass. Independently verify-tdd again and verify every C/schema fault reaches its passing control and actual operation. A pass of these five files alone is insufficient: new internal/designlock/attestation_snapshot_test.go and the approved supplemental internal/gates/attest_green_test.go must supply the already-authorized new API/held-root and wrapper/renderer evidence. No additional ownership, seam or test edit is granted.

Required supplements preserve exact R2 capabilities, four topology held-root acquisition/name/identity/content/copy/overlay proof, defensive Entries copy, actual root/file replacement and read-chunk mutation with fired/no-fault controls, entries/depth/aggregate at-limit and limit+1 under approved lowered budgets with one combined overlay budget, actual oversized sparse file per-file limit, and evidence/document bounds. Lower-budget proof must not claim actual 8 GiB hashing. Full closed-schema grammar/category/limit correctness remains a final implementation/source review obligation; frozen parser samples do not license an open schema.

Suite-owned first-Release callback supplements must exercise real SelectRunAndNote, package RunSelected, WithImplementation and RenderAttestation, matched fired no-fault/original-mutation/actual-owned-cleanup cases, error joining and nil renderer bytes, suppression/finalization, logical paths and idempotency. Child TMPDIR isolation, original/copy distinction and sentinel safety remain exact. Generic helper files and outer CLI hook are read-only. Hook finalization must precede tally/output/ledger clear and remain blocking on custody failure despite relaxed/wave/empty selection. Any later genuine frozen-fixture defect requires a new exact dispute/review; no example/golden or test repair is preauthorized.

AC5 is conjunctive on the SAME final GREEN revision: OBSERVED real renderer late original mutation and actual private cleanup failure -> nil bytes with concrete causes and fired/no-fault controls; OBSERVED ordinary built CLI complete generation success and renderer-input/alias failure -> exact exit 1/empty stdout; REVIEWED actual delivered generation CLI -> RenderAttestation -> Release/error join -> command output/defer/error exit closure, including no alternate/fallback/error emission. COMPOSED late renderer failure -> CLI exit 1/empty stdout. UNOBSERVED independently injected standalone CLI late failure. UNFORCED individual OS Close primitive errors. Output-sink partial writes remain a separate documented limit. Hashing does not authenticate execution, reviewer identity or judgment correctness. Docs/help/all displayed current reports must retain the exact scope limits.

Cost-only forecast 2710–3310 and 13 required/optional14 paths is reasonable for reviewed separate gates/process/Git/hook/custody proof; actual RED 1487 changed lines is independently measured. Forecast is neither a cap nor acceptance or permission to trim proof/factor helpers. Final actual cost and material overrun remain reviewable.

## Verification and custody

Independent `pvg story verify-tdd --range a82277af5650b487cea1260c24ffcc1c86d69d8d..47ba44906a09bc2fa010092a86b133d0d749c52c` PASS: two commits, zero skipped merges, no unauthorized test edits. `pvg story verify-delivery MAC-p7jd` 9/9 shape checks, explicitly not source/behavioral approval. `git diff --check a82277a HEAD` clean; detached checkout clean. No full preflight, remote, install/update/settings, service, Dagger/container, installed binary or developer worktree action. Installed machinery SHA256 checked as 5205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849. All review processes completed; independent logs/report retained. Transition must be approve-red only, then status/labels read back.

## nd_contract
status: in_progress

### evidence
- Independent RED review APPROVED candidate47ba449 against unchanged a82277a after exact authorized repair audit and complete synchronous scoped replay:189 leaves,47 pass,142 fail,0 skip;24 genuine A failures,8 D passing controls,26 B interface/diagnostic failures,92 C family unreached cases.
- Frozen hashes and complete source/assertion/cause/AC/mandatory-supplement analysis above; raw logs and exact leaves in /tmp/machinery-p7jd-pm-red.fnQSrl. This is RED approval only, not GREEN acceptance.
- Independent verify-tdd PASS two commits; verify-delivery9/9 shape only; no production/test/docs edits by PM. Proper approve-red transition and readback follow.

### proof
- [x] RED AC #1: independent complete-scope/digest and topology/forgery/mutation assertions frozen; actual v2 sensitivity and held-root/bounds supplements mandatory GREEN.
- [x] RED AC #2: schema/kind/current-history/complete-warning contracts reviewed; real Ga/plan compatibility observed; future v2/history/complete control and docs remain mandatory.
- [x] RED AC #3: independently observed24 real legacy false-acceptance failures with8 D controls; explicit migration diagnostic assertions frozen.
- [x] RED AC #4: actual filesystem/lifecycle/hook fault assertions with control gates and exact seam/safety constraints reviewed; mutation legs presently unreached and not credited.
- [x] RED AC #5: real CLI A/D baseline observed; generation/current/ordinary-error and same-revision mandatory renderer plus CLI plus source-closure conjunctive proof retained with honest COMPOSED/UNOBSERVED/UNFORCED limits.

## nd_contract
status: delivered

### evidence
- Sr PM canonicalized only the final independent POST-FREEZE PM COST REVIEW: same-story forecast approximately2710–3310 total,13 required/optional14 paths unchanged. Components ~1485 RED +175–375 GREEN tests +900–1250 production +150–200 docs/help; no cap, new ownership/API/seam or proof trimming.
- Independently read exact PM authority and source stats: 7ec5d609 five files1475 insertions/3 deletions; repaired47ba449 five files1484 insertions/3 deletions (=1487 changed lines). Rounded forecast is honest estimate, not final measured total. Distinct gates/CLI/Git/hook custody matrices and existing helper reuse justify preserving proof rather than forced factoring.
- Only two canonical forecast paragraphs changed; all five AC, R2, AC5 compositional bar, frozen tests, exact existing amendments/seams and read-only boundaries unchanged. Existing RED-only delivered contract/state retained; no approve-red/GREEN acceptance by Sr PM.
- RED-only tests committed at 47ba44906a09bc2fa010092a86b133d0d749c52c, preserving original 7ec5d609 and unchanged a82277a production. Exact independent PM repair authorization followed.
- Full scoped replay: 189 native leaves, 47 passes, 142 expected/absent-interface failures, zero skips; 24 genuine A failures, eight D passes, 26 B failures, 92 unreached C family cases. No setup failures remain in this replay; no current custody guarantee claimed.
- pvg verify PASS five files/zero issues; verify-tdd PASS two commits/no unauthorized edits. Independent PM review/approve-red pending; this is not GREEN acceptance.
- Complete report, exact leaf inventories, raw logs, timings, hashes and repair diff: /tmp/machinery-p7jd-red-proof.RvOHwO/.

### proof
- [x] RED AC #1: frozen full-scope/inventory cases compile and valid-control failures are classified; final GREEN implementation proof pending.
- [x] RED AC #2: plan/current/history tests and real legacy controls supplied; new semantics remain unexercised until GREEN.
- [x] RED AC #3: 24 genuine legacy false-acceptance failures with passing compatibility controls supplied.
- [x] RED AC #4: negative/positive scope and custody contracts frozen; C fault execution and approved new API supplements remain mandatory in GREEN.
- [x] RED AC #5: actual baseline CLI integration and explicit same-revision AC5 composition limits recorded; renderer late faults, new CLI behavior and final wiring review remain mandatory in GREEN.


Prior canonical Description (historical previous forecast; all authority retained):
> ## USER INTENT
> Strengthen Machinery mission-critical assurance with observable fail-closed behavior and precise limits.
> 
> ## Context (Embedded)
> Assessment F3: gt.conformance-test-shape covers BUILD artifacts, while g4.pack-event-discipline covers pack. Neither binds tested/reviewed implementation. Historic Ga ancestor acceptance is valid history, not current-tree assurance. Preserve judgment-vs-mechanical distinction.
> 
> ## Ownership
> The following 13 required files are the reviewed ownership boundary; implementation and test editing remain subject to their independent phase authorizations. You are not alone: preserve other edits, especially accepted MAC-p8ce/MAC-olrx behavior; coordinate shared paths. No generic helper factoring is authorized. Optional internal/gates/attest_green_test.go is the reported fourteenth file only for independent supplemental GREEN proof.
> 
> ## Boundary Map
> PRODUCES:
> - internal/gates/attest.go -> AttestationReview; CheckAttestationsWithImplementation(design, impl string) *Gate; RenderAttestation(design, impl string, review AttestationReview) ([]byte, error); private checker/render/pending-result contract below
> - internal/gates/attest_implementation_test.go -> NEW frozen existing-interface staged A/B/C/D and real filesystem/Git regression proof
> - cmd/machinery/attest.go -> newAttestCmd() *cobra.Command generation flags/output; preserve stableAttestationHashes(paths []string) ([]string, error)
> - docs/attestation-evidence.md -> closed v2 schema, migration, scope/digest grammar and honest limits
> - internal/gates/suite.go -> Snapshot capture/RunSelected/CheckUnchanged/Release lifecycle and explicit-release wrappers, only attestation integration
> - cmd/machinery/check.go -> Gv-facing help/messages only
> - internal/gates/attest_test.go -> only independently PM-authorized exact wrong-version amendment and v2 cases
> - cmd/machinery/attest_test.go -> only independently PM-authorized exact mandatory alias-proof amendment
> - cmd/machinery/attest_implementation_test.go -> NEW frozen real isolated CLI/local-Git A/B/C/D proof
> - internal/designlock/attestation_snapshot.go -> NEW AttestationTreeSnapshot, AttestationTreeEntry and (*Lock).MaterializeAttestationTree(path string) (*AttestationTreeSnapshot, error); exact APIs below
> - internal/designlock/attestation_snapshot_test.go -> NEW real held-root/filesystem/custody proof; tests referencing new Go symbols are GREEN supplemental, not baseline RED
> - internal/hook/hook.go -> stop snapshot finalization before decision/state-clear ordering only
> - internal/hook/attestation_snapshot_test.go -> NEW real Stop/SubagentStop configured ledger/custody proof
> - internal/gates/attest_green_test.go -> OPTIONAL new-symbol GREEN supplemental tests only, report purpose and cost; never edit frozen RED
> CONSUMES:
> - Existing Machinery implementation at epic a82277a; preserve compatibility and generic snapshot semantics.
>   spec: CheckAttestations(design string) *Gate; attestationRequiredPaths(g *Gate, design, claim string) []string; stableAttestationHashes(paths []string) ([]string, error); (*Snapshot).RunSelected(impl string, sel Selection, opt RunOptions) []*Gate; SelectRunAndNote(design, impl, gateList string, opt RunOptions) (Selection, []*Gate, string, error); (*designlock.Lock).MaterializeExternalTree(path string) (*ExternalTreeSnapshot, error) remains generic-only, not strict-subject authority.
> - Existing internal/designlock private snapshot helpers/state, consumed without production edits to their files.
>   source: snapshotBudget/readSnapshotDir/copySnapshotFile/sameFingerprintFile/validateInventoryPath/newPrivateSnapshot; exact approved new-capability use and held-root protocol embedded below.
> 
> ### Story Acceptance Criteria
> 1. Implementation/test behavior claims bind a complete explicit implementation/test scope under rooted inventory and content hashes, not only BUILD or pack. Any code/test/config addition, removal, rename, content change or scope narrowing affecting the claim invalidates freshness.
> 2. Distinguish plan-only claims, current implementation review and historical milestone acceptance in closed schema, gate diagnostics and docs. Historical ancestor records remain historical; they cannot alone imply current implementation approval.
> 3. Provide explicit compatibility migration for existing attestations. Legacy design-only covers never quietly grandfather implementation assertions as fresh; users receive actionable missing-subject diagnostics.
> 4. Negative tests remove assertions after review, alter event handlers, add excluded files, change scope, alias paths/symlinks and replay stale attestations. Positive unchanged reviewed scope and harmless evidence-only commit remain usable.
> 5. Real CLI attest/check path exercises changed implementation and historical/current distinction. Hashing proves binding, not reviewer honesty or that tests executed; output never claims otherwise.
> 
> ## Testing Requirements
> - Explicit hard TDD: separate RED test author, expected behavioral assertion failure plus passing control on unmodified production, independent PM replay, frozen RED test/fixture bytes retained through GREEN. Compilation, import, timeout or unavailable infrastructure are not valid RED. Existing tests encoding the unsafe contract may be corrected during reviewed RED; GREEN cannot silently weaken them.
> - Integration tests: MANDATORY (no mocks). Exercise real process/filesystem/runtime boundaries; no stubs, no skip-if-missing. Fixture policy inputs may test parser logic but are not live remote proof.
> - go test ./internal/gates -run Attest; go test ./cmd/machinery -run Attest; CLI integration with real temp git repository and design+implementation roots.
> - Full scripts/preflight.sh ONLY at end of epic. No pushes, sync, remote mutation, installed binary/plugin/agent/skill replacement, or dev-link. Another agent uses installed Machinery in NIL; builds/tests use isolated output and homes.
> - Shipped product enforcement is standalone Machinery: never requires pvg, nd, Paivot labels/metadata or commit conventions. Paivot is local development coordination only.
> 
> ## OUT OF SCOPE
> - Other assessment areas have sibling stories; final preflight/local main merge/isolated candidate binary are final gate responsibilities. Do not omit small directly related safety fixes.
> 
> ## DIFF BUDGET
> - Supersedes original ~4-7 files/<1000 changed LOC: 13 required files, approximately 1,900–2,900 changed LOC (900–1,350 production, 850–1,350 tests, 150–200 docs/help). Optional fourteenth GREEN-supplemental test file must have its purpose and cost reported. Report actual files/LOC; overruns trigger PM investigation, not automatic rejection or weaker proof.
> 
> ## Approved architecture authority and executable clarifications
> Independent contract review APPROVED revision 2. Authority: proposal SHA256 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179, read in full and embedded below; rejected revision 1 remains historical only. Architecture revision 2 is independently approved, and the subsequent PRE-RED PM EXACT TEST-EDIT / SEAM AUTHORIZATION has authorized bounded RED authoring for its exact two existing-test amendments and four seam uses. This is not approve-red, GREEN dispatch authorization, delivery or executed AC proof. Earlier architecture-pending/authoring-pending statements remain history, not the current phase disposition. Dispatcher may resume the healthy retained RED author within that existing authorization; independent RED review is still required.
> 
> Three reviewed executable clarifications apply to the embedded contract:
> - Repeated Snapshot.Release returns the latched final disposition without revalidation, re-closing, re-finalization or resurrecting invalid current results. Capture and Snapshot.RunSelected after release fail closed; repeated RunSelected calls before release are supported and covered, retaining all captures/pending results until that single finalization.
> - Every late Release/finalization cause and Gate error is remapped to logical caller-facing paths before exposure; no private snapshot path leaks. Add matched success/error tests through convenience wrappers and direct snapshot use.
> - Every DISPLAYED current-review report includes the exact scope-boundary limits stated below, including displayed CLI/hook reports. Silent successful hooks may remain silent; this is not a new reporting feature.
> 
> No execution authentication, reviewer-honesty claim, MAC-l7m0 dependency, Docker requirement, or new user choice is introduced. Generic designlock.go, external_snapshot.go, source_snapshot.go, snapshot_inventory.go, scale.go, portablepath, cmd/machinery/hook.go, generic snapshot callers, accept.go and accept_test.go remain read-only. New API declarations below are approved PRODUCES, not preexisting callable test seams.
> 
> ## Current independent PM authoring authorization
> Independent PM has explicitly authorized only: (1) internal/gates/attest_test.go TestAttestationMutations wrong-version integer 2 -> 3 and exact diagnostic "attestation_version must be the integer 1 or 2", with separate new accepted-v2 plan and malformed-v2 cases using real existing fixtures/CheckAttestations; (2) cmd/machinery/attest_test.go TestAttestRejectsIdentityAliases os.Link setup failure t.Skipf("hard links unavailable: %v", err) -> t.Fatalf("create required hard-link alias fixture: %v", err). The existing mandatory alias case was chosen, not an alternative new case. Retain its real os.Link/CLI, exit 1, empty stdout and identity-alias assertions. All other existing cases/helpers/tests remain unchanged absent a new named review. The complete PRE-RED PM EXACT TEST-EDIT / SEAM AUTHORIZATION preserved in this story governs exact restrictions; this clarification issues no additional test authorization.
> New frozen RED tests are confined to the named attestation implementation files, new hook/designlock test files where they can compile using existing APIs/seams, and the exact authorized amendments. New helper API unit tests that cannot compile before production belong to explicitly reported GREEN supplemental proof; they cannot replace frozen existing CLI/suite/hook acceptance tests. A/B/C/D classification below is mandatory.
> There is NO blanket permission to add fault/callback/budget seams. Independent PM has approved exactly four bounded uses, with all original restrictions retained in this story: existing Snapshot APIs for frozen lifecycle tests; hook writer interface beforeAttestationFinalization(*gates.Snapshot, []*gates.Gate); suite-local attestationBeforeFinalRelease func(*Snapshot) only for first-Release GREEN supplemental real mutation/owned-copy cleanup tests; and new designlock capability tests using existing testAfterSnapshotCopyReadChunk plus the exact newAttestationSnapshotBudget factory with fixed production defaults and reviewed lowered test budgets. Preserve actual-operation, fired/no-fault control, no direct verdict mutation, helper-process TMPDIR isolation and cleanup restrictions. Any other seam/use needs its own exact independent PM review. Stage A existing-interface behavioral failure plus Stage D passing controls is the required baseline RED; B interface absence and C not-yet-reached mutations must be recorded honestly. Do not call unavailable Go symbols from frozen baseline tests.
> Focused verification after scope/test freeze: go test ./internal/gates -run Attest; go test ./cmd/machinery -run Attest; go test ./internal/designlock -run Attestation; go test ./internal/hook -run Attestation. Record all exact discovered leaf names, baseline/final outcomes and reached/not-reached mutations; choose bounded timeouts and record runtime before any broader replay. No full preflight until epic final gate.
> 
> ## Approved schema and classification
> 
> Write attestation_version: 2 for new rows. Continue to parse closed v1 for explicit migration semantics below. Reject every other version, including numeric spellings other than integer 1 or 2. Root allowed keys remain attestation_version, attestations, optional string _comment. attestations is a nonempty array. YAML duplicate keys at every depth, unknown keys, wrong node types, nulls, aliases/merge keys, and duplicate rows are errors. All optional note/_comment fields are strings, not arbitrary YAML payloads.
> 
> V2 row allowed keys: claim, kind, attestor, date, covers, implementation, note, _comment. claim/kind/attestor/date are nonempty strings; date is an actual YYYY-MM-DD date. covers is a nonempty array of closed {path, hash, optional string _comment}. cover path is a canonical portable design-relative regular-file path, never absolute, '.', '..', backslash, traversal, or a normalized alias; use portablepath.ValidateRelative. No duplicate/casefold-equivalent cover paths. Hash is exactly sha256: followed by 64 lowercase hex digits. covers may not name attestations.yaml (self-reference). Existing required design-subject coverage remains mandatory, including all BUILD packets, pack files and acceptance files relevant to the claim.
> 
> kind is exactly plan, current, or historical. There remains at most one row per claim ID, including across kinds: upgrading a plan to current replaces that claim's row; git history retains previous reviews.
> 
> Claim classification is an explicit table in code, not a prefix-derived runtime guess:
> 
> * Design-plan judgments: the six g2 IDs; the four g3 IDs; g4.zero-context. Allowed kind: plan only. Their words and diagnostics describe the design/model/BUILD plan, not observed implementation behavior. They keep their existing required design subjects and owed predicates.
> * Implementation/test judgments: gt.conformance-test-shape, g4.pack-event-discipline, g4.standin-coverage. Allowed kind: current, or plan to record intent only. current requires the complete implementation scope below AND existing required design covers. plan never counts as current implementation review. Stand-in coverage is classified here because its judgment includes actual stand-ins held to an oracle and an environment recipe; keeping only the BUILD paragraph would repeat the same defect.
> * ga.review-quality: historical only. It judges the milestone acceptance record(s), with covers over the required acceptance/*.yaml files. It has no implementation field. Ga's existing acceptance schema is unchanged; its commit fields remain the history anchors.
> 
> implementation is REQUIRED exactly when kind=current, and FORBIDDEN otherwise. Closed object keys: root, policy, entries, hash. root is the normalized slash-separated lexical relative path from the logical design root to the supplied implementation root ('.' or leading '..' components are permitted only in this locator, since sibling/parent implementation roots are normal). It must exactly equal the locator independently derived from the actual invocation; no absolute paths, backslashes, redundant components, or nonportable non-dot components. This locator grants no path-opening authority. All reads come from the invocation's held root capability. policy must equal full-root-v1. There are no user include, exclude, extension, gitignore or subtree selectors. entries is a nonempty sorted inventory (including root directory '.'); hash binds the canonical serialization below.
> 
> Every inventory entry is a closed mapping. Directory: {path, type: directory, mode}; regular file: {path, type: file, mode, size, hash}. mode is a four-character octal permission string 0000–0777; size is a nonnegative integer within limits; hash has the same fixed SHA256 spelling. File-only keys on directories are errors. '.' is allowed only for the single root directory; all other paths use portablepath.ValidateRelative. Sort by raw ASCII path bytes; reject unsorted rows, duplicates, casefold collisions, missing parents, file/descendant conflicts and wrong root shape. Do not silently normalize a submitted manifest.
> 
> Concrete schema/digest example: ARCHITECTURE.md bytes are "# Architecture\n", BUILD.md bytes are "# Build\n", both implementation files are distinct regular files with bytes "package example\n" (16 bytes), implementation root mode 0755 and file modes 0644. acceptance/M1.yaml bytes here are "milestone: 1\n" solely to illustrate Gv's historical cover hashing: that fragment is NOT valid Ga acceptance and MUST NOT be presented as a valid milestone acceptance fixture. The current scope digest below was calculated from the exact canonical grammar; this example is not a --complete fixture and does not prove a test contains assertions. Use the existing complete Ga record shape for the real Git history tests.
> 
> ```yaml
> attestation_version: 2
> attestations:
>   - claim: g2.nfr-content
>     kind: plan
>     attestor: R
>     date: '2026-09-05'
>     covers:
>       - {path: ARCHITECTURE.md, hash: 'sha256:2e7cb38229f7877e569cc05dd9c8fb71b30954ad2c9978328d69f5d377f81505'}
>   - claim: gt.conformance-test-shape
>     kind: current
>     attestor: R
>     date: '2026-09-05'
>     covers:
>       - {path: BUILD.md, hash: 'sha256:bbc290c9f84e532bd47737480381f0db3afae637d696806856d95d0a186bb619'}
>     implementation:
>       root: ../src
>       policy: full-root-v1
>       entries:
>         - {path: '.', type: directory, mode: '0755'}
>         - {path: handler.go, type: file, mode: '0644', size: 16, hash: 'sha256:e0e0431b63a883552b05817d33ba13f79262019cdefb4e5ae060c299c1de1eb8'}
>         - {path: handler_test.go, type: file, mode: '0644', size: 16, hash: 'sha256:e0e0431b63a883552b05817d33ba13f79262019cdefb4e5ae060c299c1de1eb8'}
>       hash: 'sha256:4e0ca743a34c585082fdcfb7f57ca6a47a1e3448510c3033f5dd7281a351ac2a'
>   - claim: ga.review-quality
>     kind: historical
>     attestor: R
>     date: '2026-09-05'
>     covers:
>       - {path: acceptance/M1.yaml, hash: 'sha256:a44677835fe6e178bd0d0d51faea9ccfc327ec4442bf96a4a96031392c7da249'}
> ```
> 
> Invalid examples/categories: version: '2' (wrong type); version 3 (unsupported); kind: accepted (unknown); current without implementation (GV_MISSING_IMPLEMENTATION_SUBJECT); kind: historical on gt.conformance-test-shape (GV_KIND); implementation.exclude: ['generated/**'] (GV_SCHEMA); root ../src changed to ../src/tests (GV_SCOPE_ROOT or GV_SCOPE_HASH); duplicated handler.go or ./handler.go (GV_SCOPE_PATH); omitted handler_test.go while retaining the old scope hash (GV_SCOPE_HASH), or after recomputing the manifest hash (GV_SCOPE_INVENTORY against independently enumerated root). A syntactically sound current row with changed bytes is GV_STALE_CONTENT, not schema failure or DRIFT.
> 
> ## Complete scope and deterministic freshness
> 
> The authority is the entire actual --impl root. Enumerate every directory and regular file recursively, including ignored, untracked, hidden, generated, extensionless, binary, vendor, build-output, test, and configuration files. G4 contract ignores, .gitignore, .machinery configuration and programming-language extension lists NEVER determine this inventory. They are themselves ordinary included files. Empty directories and permission changes are bound conservatively. This can be larger than the language scanner's scope; that is intentional. Dependencies/configuration outside the supplied root are outside this explicitly reported review, and Machinery must not call this whole-program or environment completeness. Users select a root containing the code/test/config they want reviewed; switching root changes the bound locator.
> 
> Exactly two reserved exclusions exist, determined by policy and invocation, never by the row:
> 
> 1. The supplied implementation root's top-level .git metadata entry, if present. It must be a real directory or regular worktree gitfile, never a symlink/special file. Its content/presence is excluded, allowing ordinary Git commits. All nested .git entries cause GV_SCOPE_UNSUPPORTED_METADATA, instead of being silently pruned. This is a deliberately conservative policy: nested-repository users must provide a source export without nested metadata. The existing generic snapshot skips .git at every depth and is insufficient for this particular rule. Top-level VCS administration, hooks and repository-local Git configuration are NOT implementation/test execution inputs covered by this review; using them as runtime source is outside this reported scope. There is no language-independent proof that arbitrary code does not read VCS metadata. Documentation and output must state this exact boundary, not claim excluded metadata can never affect behavior.
> 2. The exact logical <design>/attestations.yaml path, only when it lies within the implementation root. This is Machinery's reserved review record, not arbitrary *.yaml, an evidence directory, or a glob. Omit that entry's presence/content/mode from the inventory, with a constant exclusion descriptor included in the digest whether the file exists yet or not. Gv separately parses the present file under the closed schema. All sibling files, nested files named attestations.yaml elsewhere, acceptance YAML, logs, docs, generated tests and configuration remain included. Changing a note/attestor/date in this one record and committing it is the matched harmless evidence-only positive. This proposal does NOT promise freshness survives edits to arbitrary prose/docs or adding acceptance records; they remain conservatively bound when inside --impl.
> 
> Exclusions are reserved semantic boundaries, not authenticated assurances about how the application executes. If the user intends application code to consume Machinery evidence or Git metadata as runtime inputs, full-root-v1 does not cover that runtime and must not be described as doing so. This precise limit is preferable to claiming that hashing can discover all transitive execution dependencies. No further exclusion expansion is implied.
> 
> Before hashing compare the submitted inventory to a fresh authoritative inventory in BOTH directions. Any added, removed or renamed path is GV_SCOPE_INVENTORY; changed type/mode/size/hash is GV_STALE_CONTENT (type aliases/symlinks instead use custody categories below). Report the first differing portable path deterministically plus summary counts; do not print private temp paths or raw file content. Every current row is checked against the same fresh inventory for the invocation. Required design covers are independently checked too, so unrelated unchanged files cannot discharge a claim.
> 
> Canonical digest input is UTF-8/ASCII, exactly this header and tab/newline grammar, no YAML formatting and no timestamps/commit/inode numbers:
> 
> ```
> machinery-attestation-scope-v1\n
> root\t<logical-root-locator>\n
> policy\tfull-root-v1\n
> exclude\tvcs-root:.git\n
> exclude\tevidence:<implementation-relative-design-attestations-path-or-none>\n
> directory\t<path>\t<mode>\n
> file\t<path>\t<mode>\t<decimal-size>\tsha256:<hex>\n
> ...
> ```
> 
> The last two line forms repeat for the globally path-sorted inventory, including root '.'. Hash the exact byte concatenation with SHA256, print sha256:<lowercase hex>. Portable paths cannot contain tabs/newlines, so this encoding is unambiguous. Counts derive from entries and need no second mutable field. Digest includes both inventory and scope-policy/root/exclusion identity. The row's hash must first match its own submitted inventory, then the fresh authoritative inventory hash. Neither rewriting exclusions (unknown schema) nor removing entries (set comparison) can preserve a prior review. Changing the root requires a different digest and must also match the invocation's root. A dishonest author can calculate a new receipt; that is a NEW self-authored assertion, not proof an independent reviewer re-reviewed it.
> 
> Limits: preserve snapshot bounds, 100,000 entries, depth 64, 1 GiB per regular file, 8 GiB aggregate; stream file bytes with existing bounded readers/copy routines. Apply one budget to the full logical inventory including design overlay; do not reset budgets per directory, row or overlay. V2 YAML and each required design cover obey the existing 16 MiB designArtifactMaxBytes reader limit in internal/gates/confinement.go (exact epic source verified after graph discovery). Generation must enforce the 16 MiB serialized document bound too, and warn that merging multiple independently generated rows must fit that same document bound. If inventory YAML cannot fit, fail GV_EVIDENCE_LIMIT with the size and limit rather than truncate. No flags to relax limits in this story. Existing explicit-file hash mode retains its separate 16 MiB/file bound. Scope inventory limits alone do not guarantee a maximum-size manifest can be stored.
> 
> Reject symlink root leaves, directory/file symlinks anywhere in included scope, special files, invalid portable names, duplicate/casefold names, and replaced roots/entries during capture. The held rooted reader prevents escaping through descendants; it must never resolve a row path against the ambient original root. A no-follow check of the root leaf is required; do not invent a ban on platform ancestor aliases such as macOS /tmp.
> 
> Alias policy distinction: duplicate logical receipt paths and path-normalization aliases are invalid independently of filesystem identity. Hardlinks at two different logical paths are not inherently a byte-freshness exploit: complete immutable snapshots retain both paths and bytes, and additions/removals still differ. Nevertheless this proposal conservatively rejects same-inode regular-file pairs during attestation capture, matching the existing explicit-file CLI policy and making alias fixtures deterministic. This is the approved policy, not a claim that copies lose byte safety. Cross-root hardlinks outside the enumerated root cannot be exhaustively discovered; held descriptor/identity/content revalidation handles observed mutations. No inode identity persists across invocations, because checkout portability must remain possible.
> 
> ## Root custody and approved exact API contract
> 
> Existing signatures remain source-compatible:
> 
> ```go
> func CheckAttestations(design string) *Gate
> func AttestationClaimIDs() []string
> func ContentHash(path string) (string, error)
> func stableAttestationHashes(paths []string) ([]string, error)
> func (s *Snapshot) RunSelected(impl string, sel Selection, opt RunOptions) []*Gate
> func SelectRunAndNote(design, impl, gateList string, opt RunOptions) (Selection, []*Gate, string, error)
> ```
> 
> CheckAttestations(design) remains a design-only compatibility wrapper calling checkAttestationsInSnapshot(design, nil), with the existing rooted readDesignFile behavior and diagnostic wording preserved. It does not acquire an implementation root or newly claim whole-operation snapshot guarantees for standalone legacy callers. Plan/history remain usable, but any current row produces GV_IMPL_REQUIRED and legacy behavioral rows produce GV_MISSING_IMPLEMENTATION_SUBJECT. It does not infer impl from cwd, Git root, BUILD text or evidence. Do not change it variadically: a distinct API keeps absent-root semantics visible. Production suite callers always pass the held design snapshot to the internal checker, and the new WithImplementation convenience API acquires the full snapshot itself.
> 
> New gates APIs/types (owned by attest.go, private subject fields protect capture construction):
> 
> ```go
> type AttestationReview struct { Claim, Kind, Attestor, Date, Note string }
> func CheckAttestationsWithImplementation(design, impl string) *Gate
> func RenderAttestation(design, impl string, review AttestationReview) ([]byte, error)
> type attestationSubject struct { /* private captured manifest, root locator, policy */ }
> func checkAttestationsInSnapshot(design string, subject *attestationSubject) *Gate
> func (s *Snapshot) captureAttestationSubject(impl string) (*attestationSubject, *designlock.AttestationTreeSnapshot, error)
> func renderAttestationInSnapshot(design string, subject *attestationSubject, review AttestationReview) ([]byte, error)
> ```
> 
> CheckAttestationsWithImplementation acquires one Snapshot, captures through the new capability, checks, and explicitly releases before returning its Gate. It folds custody/close/release failures into Gate.Errs and returns no published current-review count on failure. RenderAttestation does the same but returns buffered YAML only after capture/revalidation/close/release all succeed. It generates hashes and exact manifest; it does not execute tests or attest reviewer identity. Direct convenience caller failures use the same Gv categories, never optimistic success on absent impl.
> 
> Suite wiring: when Gv is actually applicable and impl is nonempty, use captureAttestationSubject for the existing implementation preparation; pass the resulting subject to checkAttestationsInSnapshot. Other gates receive an immutable implementation Path with the SAME inclusion/topology as the existing MaterializeExternalTree result. Otherwise keep existing preparation. Add one private RunOptions field `attestationSubject *attestationSubject`; no exported user-settable snapshot paths. At Gv call use checkAttestationsInSnapshot(design, opt.attestationSubject). Keep current selection predicates, MAC-p8ce changes, cargo authority handling and remapping. Add a private suite-local `implementationSnapshot` interface with Path() string, Logical() string and Close() error so existing generic and new attestation captures can share the local stable variable; it is not a new public API. Generic impl and cargo cleanup stays in RunSelected. Strict attestation captures are instead registered on Snapshot and retained until Release; error exits either close an unregistered capture immediately or let the registered owner close it, never both. A current row without impl is diagnosed by Gv after parsing, not by a global ban on --gate gv without --impl.
> 
> Approved narrowly scoped shared capability (new internal/designlock/attestation_snapshot.go; preserve generic MaterializeExternalTree callers):
> 
> ```go
> type AttestationTreeEntry struct {
>     Path string
>     Directory bool
>     Mode uint32
>     Size int64
>     SHA256 [32]byte
> }
> type AttestationTreeSnapshot struct { /* private tree, full inventory, held source roots */ }
> func (l *Lock) MaterializeAttestationTree(path string) (*AttestationTreeSnapshot, error)
> func (s *AttestationTreeSnapshot) Path() string
> func (s *AttestationTreeSnapshot) Logical() string
> func (s *AttestationTreeSnapshot) Entries() []AttestationTreeEntry
> func (s *AttestationTreeSnapshot) CheckUnchanged() error
> func (s *AttestationTreeSnapshot) Close() error
> ```
> 
> Entries returns a copy, never mutable internal authority. The capability captures a complete logical inventory from held no-follow ORIGINAL source root(s), applies the top-level/nested .git policy, detects hardlink pairs before copies erase alias identities, and proves inventory bytes equal the returned stable tree plus retained design overlay. Evidence filtering/digest is in gates; designlock supplies the unfiltered observed entries (apart from VCS metadata) so unrelated consumers do not learn the attestation schema. It retains the original root handles until final CheckUnchanged and Close. The existing design SourceRoot is a private copy, NOT an original held descriptor; revision 1 did not make that distinction sufficiently explicit.
> 
> MaterializeAttestationTree acquisition owns this exact protocol: resolve logical topology lexically; open original authority root(s) below; compare pre-open Lstat, held-root Stat and l.rootInfo for the design; enumerate design THROUGH ITS HELD ORIGINAL ROOT using the existing design fingerprint policy and compare the exact result to l.snapshot AND retained SourceRoot bytes (root identity/mode checked separately). This bridges the earlier private-copy acquisition to the newly retained original authority, failing if they differ; it does not claim the new original handle existed at AcquireReader time. Capture the strict full implementation inventory and copy through held root operations, with witnessed pre/open/read/post identity/content checks and a second complete held-root pass. Prove the copy plus retained design overlay equals the full inventory before returning. Reject nested .git discovered in the strict implementation scope even when the generic design copy omitted it. Required design covers are read only from the already verified private design copy.
> 
> | Logical topology | Original handles retained by capability | Stable implementation Path and final authoritative checks |
> |---|---|---|
> | Disjoint design and impl | Independently open/retain original design root and original impl root once; compare design handle to l.rootInfo | Private implementation copy; held design fingerprint must equal l.snapshot/retained generation; held strict impl inventory must equal captured inventory |
> | Impl is ancestor of design | Open original impl root; obtain design root by held impl.OpenRoot(design-relative-path), validate all intermediate real directories and equality with l.rootInfo; retain both handles | Generic-compatible impl copy omits design subtree; full Entries includes that subtree from original inventory, proven equal to retained design copy; final held impl pass checks ALL scope including design; held design pass also preserves original design generation |
> | Impl equals design | One original root handle, shared ownership with exactly one close | Path is retained SourceRoot; strict full inventory and generic-policy design snapshot comparison both run through original handle; reject nested .git in strict scope |
> | Impl is inside design | Open original design root; obtain impl root by held design.OpenRoot(impl-relative-path), validate intervening real directories; retain both | Path is retained SourceRoot/subpath; final strict impl inventory uses retained impl handle and design-generation check uses retained design handle; root-name/ancestor identity witnesses catch replacement of subpath |
> 
> Logical relative strings decide topology but do not grant read authority. No EvalSymlinks-based following inside these roots. Merge full inventory/path/case/identity checks and budgets before evidence filtering. A design subtree containing code/test/config cannot disappear. Original handles capture identity only for the observation interval; inode identities are not persisted across commands. In-design/equal impl with a private-source path supplied by an internal caller must resolve to the known logical root through the owning Lock rather than treat a private temporary directory as a new logical implementation root; reject an unrecognized private path.
> 
> AttestationTreeSnapshot.CheckUnchanged owns the final primary attestation proof: enumerate/rehash through its retained original design/impl handles under the above policies; compare with captured generation, full inventory, bytes and root/descendant identity witnesses; check that logical root names still identify the held roots using Lstat (metadata checks, no byte-opening authority); fail on any difference. Closing the capability aggregates root-handle and private-copy close errors. Even the reserved evidence file must not change DURING a single observation, because the held design-generation check binds the parsed row; editing it BETWEEN successful invocations remains permitted by the scope digest exclusion.
> 
> Approved precise shared integration choice: implement the new held-root traversal/copy and private state in internal/designlock/attestation_snapshot.go, in the SAME package, using existing snapshotBudget/readSnapshotDir/copySnapshotFile/sameFingerprintFile/validateInventoryPath/newPrivateSnapshot helpers directly. Do NOT call MaterializeExternalTree or TrackExternalTree to establish the strict subject, because their path reopen/exclusion policy is different. A new private helper `captureAttestationRoot(root *os.Root, logical string, policy attestationInventoryPolicy, copyTo string) ([]AttestationTreeEntry, error)` performs the bounded held-root work; private policy distinguishes strict full implementation versus the exact generic design comparison. copyTo is empty for revalidation. Preserve each helper's current limits and join all close errors. Prove equality using the existing fingerprint entry spelling when comparing to l.snapshot; remove the synthetic '.' row only for that existing map comparison, checking root identity/mode separately.
> 
> This choice requires ZERO production edits to designlock.go, external_snapshot.go or source_snapshot.go. They remain read-only consumers/providers of existing private helpers/state. No vague helper factoring is authorized: if implementation cannot use those helpers without changing them, return the exact necessary signature/hunks for a further scope review. The new capability has its own held-root walk precisely so generic capture semantics stay unchanged. Its additional duplication/LOC is included in the revised estimate.
> 
> Correction to the no-reopen assurance: Gv's manifest/content comparisons and final primary freshness proof use retained original capabilities and verified private copies, never a fresh ambient path as subject authority. Existing Lock.CheckUnchanged (designlock.go2101 onward) and checkExternalUnchanged DO reopen ambient paths via fingerprint/fingerprintRoot; they remain supplementary fail-closed suite guards. A supplementary failure invalidates the result, and a supplementary pass can NEVER substitute for, override or repair a failed held-capability check. Existing workspace/Cargo readers may also keep their own established custody behavior. This proposal does NOT claim every operation in the entire suite avoids ambient reopens. Filesystem name/metadata checks are allowed to detect replacement and cannot supply replacement subject bytes. These checks bind a stable observation, not authenticated execution, continuous isolation after the final read, or an absolute guarantee against every undetectable ABA race on every platform.
> 
> ## Finalization and publishable current-review findings
> 
> Add private Snapshot fields `attestationCaptures []*designlock.AttestationTreeSnapshot`, `attestationPending []*pendingAttestationResult`, `attestationCustodyErr error`, `attestationFinalized bool`. Define the private pending result in attest.go as its Gate pointer, successful current-row count, scope/file count(s) and pending-note identity. captureAttestationSubject sets a private subject field `pendingResults *[]*pendingAttestationResult` to `&s.attestationPending`; the internal checker appends through that pointer, so suite and convenience callers use the same owner with no post-call registration gap. A current subject without its owner/pending sink is an internal custody error, never an immediate publish path. This sidecar avoids modifying Gate/gates.go. The internal checker records semantic errors immediately, but stores successful implementation/current counters and success notes in the sidecar, NOT public Counts or Notes. Public Gate includes only `current implementation review pending final snapshot release` until finalized. Design-plan/history/generic row bookkeeping keeps existing semantics and must not be labeled current implementation.
> 
> Exact lifecycle:
> 
> 1. Snapshot.RunSelected captures/registers the strict capability, runs gates, remaps logical paths, calls its strict CheckUnchanged AND existing Lock.CheckUnchanged, and performs existing generic/Cargo cleanup. Any of these custody/cleanup errors is accumulated in attestationCustodyErr, appended as G0/Gv failure, and marks every pending current result invalid. Strict capabilities remain held until Snapshot.Release. RunSelected returns provisional Gv findings with ZERO public current-review/file/scope success counters even when checks have passed so far.
> 2. Snapshot.Release performs one final strict CheckUnchanged for each retained capability and, when attestation captures exist, one final supplementary Lock.CheckUnchanged before any design-copy cleanup. It then closes each capability, closes workspace and releases Lock, joining ALL errors with accumulated attestationCustodyErr. Do not short-circuit cleanup after the first error. Snapshot.CheckUnchanged itself must accumulate any returned supplementary error in attestationCustodyErr when an attestation capture exists, so a caller cannot observe a failure and have a subsequent restoration erase it. Finalize pending results only after all operations return. Success removes the pending note and publishes stored current counters via Gate.Count plus its scope-boundary note. Any error removes the pending note, publishes NO current counters/positive current notes, and appends `GV_SCOPE_CUSTODY: current review not established; final snapshot validation or release failed` with the cause. Store the joined final error; idempotent Release returns that same disposition, finalizes once and never later resurrects invalid findings.
> 3. SelectRunAndNote changes its defer-only successful path to derive VersionSkewNote while the snapshot is held, explicitly call Release, then return finalized run/note/error. Early selection errors still release safely; no success result escapes on release failure. The package RunSelected convenience wrapper retains its signature and returns only after Release, which has already finalized/invalidated its gate pointers; it may retain the additional G0 failure for consistency. New WithImplementation and RenderAttestation wrappers use the same finalization sequence, with Render returning nil bytes on failure. Existing Snapshot.RunSelected direct callers must Release before reporting positive current assurance.
> 
> Verified hook consequence: internal/hook/hook.go stop acquires the snapshot around 1273 and defers Release at 1277; it emits Gv into an internal text buffer at 1343, but its eventual non-block stopOut JSON is written to the actual writer and clearCheckedState can run BEFORE deferred Release. The green default can also clear state then return nil before release fails. cmd/machinery/hook.go newHookCmd passes output.stdout directly to hook.Run under the installation lock; there is no outer JSON buffer that reliably retracts an already emitted non-block result. A later returned Go error does not prove every host cancels that result. This is a source-supported ordering risk, not a demonstrated host exploit. Suppressing counters alone is insufficient.
> 
> The approved scope includes internal/hook/hook.go, only stop's snapshot-finalization/decision order, and new internal/hook/attestation_snapshot_test.go. Keep the returned []*Gate instead of immediately emitting it; collect armed/ratchet and wave-sentinel facts while sourceDesignDir exists; complete snapshot.CheckUnchanged and explicitly snapshot.Release BEFORE rendering/tallying finalized gates, deciding shouldBlock/wave deferral, calling clearCheckedState or emitting any non-blocking stop result. Release is idempotent so the existing deferred cleanup can remain for early exits; early errors may emit an explicit block before cleanup because they authorize nothing. On strict capability, supplementary check, cleanup or Release failure, immediately emit stopOut{Decision: "block", Reason: <custody cause>} and retain checked-state ledger, regardless of cfg.Strict, ratchet presence or wave deferral. The empty-selected-gate branch likewise must release successfully before clearing state or returning a non-blocking result. Existing semantic error/warning, import-arming and wave-defer policy applies only AFTER successful custody finalization; do not change that product policy in this story.
> 
> Hook proof: run actual Stop and SubagentStop events against a real local configured design/impl/ledger; stable finalized current review permits the existing configured outcome and clears state; mutate a subject between gate evaluation and finalization, or fail owned snapshot cleanup, and assert exactly one block JSON, no earlier non-block output, retained touched-state ledger and no public current counters. Exercise strict=false and wave-open so those policies cannot waive custody failure. Also retain existing semantic-warning/wave controls after successful finalization. The new tests may use a specifically PM-reviewed callback around the hook's finalization boundary to mutate actual files; they must not mock the gate verdict or accept a bare process error as proof of blocking. cmd/machinery/hook.go and the outer install lock remain read-only; this proposal promises finalized DESIGN/IMPLEMENTATION snapshot custody before a stop authorization, not that unrelated outer host/installation operations are newly transactional.
> 
> Required finalization tests: expose no count before release; publish it after successful release; mutate a real original file after RunSelected but before Release and require suppression; replace/rename an original root and require suppression through the retained/name witness; trigger final supplementary ambient failure with held data unchanged and require suppression; observe Snapshot.CheckUnchanged failure, restore bytes, then Release and require the failure remains latched; trigger actual owned private-copy/workspace cleanup or root/filelock close failure using only explicitly reviewed local error-injection seams, require suppression and error propagation. Pair every case with successful cleanup/control. Test SelectRunAndNote, package RunSelected and WithImplementation return paths; generation must return nil/empty stdout on late failure. Do not count an earlier Gate counter plus later G0 error as satisfying suppression. OS close-error forcing may need a narrow deterministic test seam in the NEW capability or suite owned code; PM must approve its exact use, and it must inject the failing lifecycle operation rather than fake a successful filesystem snapshot. No further shared-helper edits beyond the explicitly named suite/hook boundary changes are proposed.
> 
> ## CLI, migration and current/history outcomes
> 
> New generation syntax, alongside unchanged existing commands:
> 
> ```
> machinery attest --design design --claim gt.conformance-test-shape --kind current --impl . --attestor R --date 2026-09-05 [--note text]
> machinery attest --design design --claim g2.nfr-content --kind plan --attestor R --date 2026-09-05
> machinery attest --design design --claim ga.review-quality --kind historical --attestor R --date 2026-09-05
> machinery attest design/ARCHITECTURE.md design/BUILD.md
> machinery attest --claims
> ```
> 
> Generation requires all named fields (including explicit date; no wall-clock nondeterminism), exactly one claim, no positional files. --impl is required for current and forbidden for plan/historical. --claims alone preserves byte-for-byte ID/order output. --claims with existing positional files retains its existing precedence; reject combination with any new generation flag. File-hash mode retains its exact output and custody tests. Generation emits one complete v2 YAML document containing the generated row and all required design covers. It does NOT overwrite or merge design/attestations.yaml; docs say review subjects, then merge the generated row into that file. If --claim has no actual required design subject, generation fails rather than manufacture a cover. No --exclude or --include flags exist.
> 
> All discovery, validation, hashing, final unchanged checks and close/release errors must finish before first generation stdout byte. On such failure stdout is empty, stderr has the category/remedy, exit 1. Successful generation writes one buffered document; an output write failure can physically truncate stdout and must return nonzero—do not promise that an arbitrary pipe provides atomic writes. Display these exact limits in generation stderr/help and finalized Gv scope notes (therefore CLI and hook output), as well as docs: 'Hashes bind the observed files and scope; they do not prove tests ran, reviewer identity, or judgment correctness. Top-level Git administration and the exact Machinery attestation record are excluded; applications that use them as runtime inputs are outside this review boundary.' The generated row's caller-supplied attestor is attribution text, not authentication.
> 
> V1 migration: keep parsing only the original v1 keys. Six g2/four g3/g4.zero-context rows retain their existing design-cover semantics as implicit plan judgments (emit an informational migration note, not a new coverage warning). ga.review-quality becomes explicitly reported legacy historical judgment over acceptance evidence; it grants no current approval. V1 gt.conformance-test-shape/g4.pack-event-discipline/g4.standin-coverage ALWAYS produce GV_MISSING_IMPLEMENTATION_SUBJECT, even with matching BUILD/pack hashes and even without --impl. Remedy names the claim and generation command, tells the user to review the complete implementation/test scope and write v2 kind=current, or explicitly recast the statement as v2 kind=plan. Never auto-add hashes or silently infer a current review from an old row.
> 
> Ordinary Gv preserves incremental adoption: owed claims with no row warn; absent evidence after owed artifact activation is still an error. A v2 behavioral plan row reports 'plan only; current implementation review missing' as a coverage warning and cannot discharge the corresponding implementation obligation. All malformed/stale/current-without-impl/legacy-behavior rows error in ordinary and complete modes. --warnings-as-errors and existing --complete warning promotion make missing current reviews blocking; --complete already requires --impl globally. Update check help/error text to name Gv alongside G4/Gt where relevant. Counts distinguish plan judgments, current implementation reviews and historical review records; retain the existing generic attested-claims count for compatibility. Never count a failed or plan/history row as current implementation review.
> 
> Ga alone retains its exact existing repository ancestry acceptance semantics, including exported identity and missing-history behavior. Gv historical rows validate the recorded acceptance-file covers and label their judgment historical; Gv alone does not pretend to have executed Ga's ancestry validation. With --gate ga,gv, a historical accepted ancestor and stale current implementation can coexist: Ga passes its history check; Gv fails current freshness. Changing --commit to the old reviewed ancestor cannot repair Gv, which always observes supplied current root. Absence of current review cannot be rescued by ga.review-quality. --complete needs its separate actual current implementation rows as well as valid history. A later Git commit updating only the reserved attestation record keeps the same implementation digest and Ga's ancestor check valid.
> 
> ## Five-AC RED/GREEN test contract
> 
> All new tests use actual files, directories and local Git repositories; CLI process tests build an isolated binary under t.TempDir, never replace installed machinery or run installation hooks. No Docker/Java/Node product/runtime dependency, network, mocks, timing lotteries or skip-if-missing. Fixture setup failure is not RED. Frozen test/fixture bytes and exact existing-test amendments require PM authorization before any author edits. The old production has no v2 interface, so a v2 passing freshness control CANNOT be required on a82277a. Tests must be partitioned and reported by the following stage contract, not collapsed into one universal baseline-positive rule.
> 
> | Frozen test category | Exact a82277a outcome | Exact final implementation outcome | Evidence it supplies |
> |---|---|---|---|
> | A — existing-interface behavioral RED: three legacy behavioral claim fixtures, using existing CheckAttestations/RunSelected/CLI; independently evaluate unchanged and assertion/handler-mutated trees; final assertions require GV_MISSING_IMPLEMENTATION_SUBJECT in BOTH cases | Compiles, setup works, old implementation accepts each legacy design-only row instead of issuing required missing-subject error; assertion fails with that exact observed error absence. Both unchanged and mutated legacy cases are negative cases under the new contract | Both unchanged and mutated legacy cases are rejected with GV_MISSING_IMPLEMENTATION_SUBJECT and never counted current | Actual unsafe legacy acceptance reproduced through existing interfaces. This is the primary required behavioral RED, not v2 sensitivity. The old acceptance is a recorded bug observation, NEVER a frozen passing control |
> | B — future-interface acceptance: complete v2 document parsing, generation flags/output, kind rules, new categories, finalize-visible counts; invoke only existing CLI/process or existing public suite symbols so baseline compilation succeeds | Fails specifically because version 2/flags/expected interface behavior is absent; report unsupported-version/unknown-flag output verbatim as INTERFACE_ABSENT, not successful fail-closed freshness or qualifying behavioral RED | Valid generation/schema/current result succeeds, invalid categories produce their exact specified errors, fully released wrapper publishes counters | Frozen desired interface contract. No claim that a baseline unknown flag proves staleness detection |
> | C — paired sensitivity after interface exists: for each mutation in matrix below, generate or load frozen independently calculated v2 receipt, verify unchanged actual scope first, then apply exactly one mutation and assert its category/path and zero published current count | First unchanged v2 control fails with known missing-interface behavior; mutation stage is NOT credited as executed freshness evidence. Freeze the entire test without skip/feature probe/conditional success; test remains failing on base | Unchanged control passes with finalized current count, mutation returns the specified error and no current success; evidence-only commit control passes on both invocations | Actual addition/removal/content/scope/alias/replay sensitivity, credited ONLY after both legs execute against implemented interface |
> | D — baseline compatibility controls (separate mandatory frozen controls): existing six-g2 v1 plan fixture; valid old filehash/--claims; ordinary partial design-plan coverage; existing local Ga ancestor control | Passes, with old exact outputs/counts where preserved | Still passes with specified compatibility semantics; no added warning on legacy plan migration | Meaningful passing baseline controls accompanying A. These are explicitly compatibility controls, not fabricated valid current implementation reviews |
> 
> A tests assert the final contract from the beginning; they do not first assert legacy success and later flip it in GREEN. Separate unchanged/mutated legacy subtests make both negative outcomes observable even if one fails. B/C tests must compile against base and use existing callable seams: a frozen Go test referencing a nonexistent new exported API is forbidden, because compilation failure is not RED. New API unit tests requiring new symbols may be supplemental GREEN tests with explicit provenance; they cannot replace frozen existing-interface/process acceptance tests. Any B/C mutation not reached because its unchanged leg failed is reported NOT YET EXERCISED, never passed. Do not add skip-if-feature-missing, accept-any-error, or baseline-vs-GREEN branches to make this staging look green.
> 
> The independent PM base replay must observe A's exact legacy false-acceptance assertion failure plus D's passing compatibility controls, and classify B/C failures honestly. The final PM replay must observe A rejection passing, B interface acceptance passing, C BOTH legs passing, and D compatibility still passing, with frozen bytes unchanged. This satisfies hard TDD without pretending the old product could generate a valid v2 scope or requiring a now-unsafe legacy current assertion to remain valid.
> 
> The following matrix specifies FINAL behavior for B/C/A as indicated above. Its passing-current controls are required when the interface exists; they are not asserted to pass on a82277a.
> 
> | AC | Real passing control | Single challenge and exact expected observation |
> |---|---|---|
> | 1 | Full source/test/config tree, valid generated current receipt; Gv current-review count is 1 | Remove assertion from test while keeping BUILD/oracle citation: GV_STALE_CONTENT for test path, current count 0 |
> | 1 | Same generated tree | Alter event handler while pack unchanged: GV_STALE_CONTENT for handler path |
> | 1 | Same generated tree | Add/delete/rename source, test, config, extensionless and generated file, one case each: GV_SCOPE_INVENTORY with added/removed path |
> | 1 | Ignored directory/file exists and is explicitly inventoried | Add ignored/untracked handler; change .gitignore/contract ignore; both inventory/config changes invalidate. No matching source extension is required |
> | 1,4 | Complete submitted entries and root locator | Remove one entry with old hash -> GV_SCOPE_HASH; recompute submitted hash -> GV_SCOPE_INVENTORY; add exclude field -> GV_SCHEMA; narrow --impl or row.root -> GV_SCOPE_ROOT/GV_SCOPE_HASH; unchanged subset cannot satisfy old full-root manifest |
> | 1,4 | impl contains design, with design/helpers/test.go captured in logical overlay | Change/add design subtree code -> GV_STALE_CONTENT/GV_SCOPE_INVENTORY, proving generic snapshot omission cannot hide it |
> | 2 | v2 plan row on design claim, v2 current on behavior, v2 historical on ga | Wrong kind/implementation-field combination -> GV_KIND/GV_SCHEMA; behavioral plan produces missing-current warning and never current count |
> | 2,5 | Real Git commit A with valid M1 acceptance and later evidence commit B; actual CLI --gate ga,gv --impl root is green | Change source at C, keep M1 ancestor A and replay receipt B: Ga history remains checked; Gv GV_STALE_CONTENT; explicit --commit A still cannot make Gv current |
> | 3 | Legacy six-g2 fixture remains accepted as plan with same existing counts and no added warnings | Legacy gt/pack/stand-in design-only row -> GV_MISSING_IMPLEMENTATION_SUBJECT naming root review/migration remedy, with/without --impl |
> | 3 | Explicitly migrated v2 current row from actual scope | Legacy row relabeled kind without v2 version fails schema; v2 current with empty/missing subject errors; v2 plan conversion is explicit and incomplete |
> | 4 | Unchanged scope checked repeatedly, including new local commit modifying only design/attestations.yaml note/date | Remains fresh with same scope hash, no HEAD coupling; modify adjacent evidence/handler.go or a different attestations.yaml -> inventory/content failure |
> | 4 | Distinct portable file paths, actual regular files | Duplicate receipt path, ./ alias, casefold alias -> GV_SCOPE_PATH; leaf/directory/root symlink -> GV_SCOPE_CUSTODY; real hardlink pair -> GV_SCOPE_ALIAS under the approved conservative policy |
> | 4 | Top-level .git is ordinary repository metadata, with commits changing it | nested src/.git/handler.go -> GV_SCOPE_UNSUPPORTED_METADATA; arbitrary .ignored/handler.go is included and addition fails freshness; generated/tests assertions remain included |
> | 4 | Stable real held root/copy using existing approved fingerprint/copy callback seams | File/root replacement, file mutation between reads, design-overlay mutation -> GV_SCOPE_CUSTODY or underlying G0-snapshot custody error; stdout empty on generation, no current success counter |
> | 4 | Bounded small regular tree | FIFO/special entry -> custody failure before open; limit+1 entries/depth/size/aggregate -> named limit failure, no partial inventory; use existing reviewed lowered-budget helper seams where actual 8 GiB creation is impractical, not mocked files |
> | 5 | Spawn isolated real binary; --attest generation -> write row -> check --gate gv --impl root returns 0 | Change test/handler and rerun returns 1 with Gv stale category; without --impl current receipt gives GV_IMPL_REQUIRED; --claims and old filehash output remain exact |
> | 5 | Complete-mode fixture valid under all other gates | Missing behavioral current row is the sole warning promoted to blocking; ordinary corresponding plan case warns only; no unrelated missing phase artifact accepted as RED |
> | 5 | Actual built CLI generation emits the complete valid v2 document only after renderer success; approved callback fires without mutation in paired real RenderAttestation control, which returns valid bytes after successful finalization | Actual built CLI ordinary input and alias failures: exit exactly 1, zero stdout bytes, specific diagnostic. Actual RenderAttestation original-subject mutation at first final Release and approved private-copy cleanup fault: callback demonstrably fires, actual operation fails with its particular cause, returned bytes are nil. Independently verify delivered CLI/error/output wiring to compose those renderer observations into the CLI before-output guarantee. A separately injected late-release fault in the standalone CLI executable is UNOBSERVED, not claimed as an executed end-to-end case. Output-sink failure retains its separately stated nonzero/possible-partial-output contract. |
> 
> Parser tests separately assert duplicate keys, unknown fields, wrong types, enum, version, path grammar, hash format, conditional fields, ordering and exact inventory self-hash. They are category B, not substitutes for category C stale-tree filesystem/CLI proof. Categories A and D provide the independently replayed base behavioral failure and compatibility control. Record each command, expected failure class, reached/not-reached mutation stage, and final result separately. A single aggregate nonzero go test or CLI exit is insufficient evidence.
> 
> ## Approved AC5 compositional proof and limits
> Original architect and independent challenger APPROVED the narrow clarification, SHA256 5fc3193106a803b7760020b646ed4db7b3d417f5e196b2266d10cf8227733937. It resolves proof composition only; all five product AC, approved R2 SHA256 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179, ownership, two exact test amendments/four authorized seams and A/B/C/D staging remain unchanged. No new seam, runtime, user choice, dependency or implementation authorization is introduced.
> 
> The product requirement remains: in attestation GENERATION mode, discovery, hashing, final validation and owned snapshot cleanup/release finish before the first stdout byte. A failure from those operations produces CLI exit 1, an actionable error on stderr, and empty stdout. RenderAttestation returns nil bytes and the actual error on such failure. An output-sink error after successful finalization remains the separately documented case where physical partial stdout is possible and the command must fail. The empty-stdout rule does not prohibit `machinery check` from printing its blocking diagnostics.
> 
> The final evidence bar is conjunctive; all three parts are required on the same delivered GREEN revision:
> 
> 1. **Observed renderer lifecycle proof.** Use only PM-authorized seam 3 in the optional internal/gates/attest_green_test.go, with its existing safety/child-process restrictions. The fired no-fault control must call real RenderAttestation and return a complete valid document and nil error. Separate actual original-file mutation and approved owned-copy regular-entry-to-symlink cleanup fault must each reach the first finalization callback, return a non-nil error and `body == nil`, and preserve logical-path diagnostics. The cleanup case must expose the concrete real private-snapshot cleanup cause, not merely an earlier generic validation error; retain sentinel/cleanup-safety evidence. These are GREEN supplemental observations, not compile-failing RED or independently observed CLI late faults. No fake renderer return or directly injected verdict is acceptable.
> 
> 2. **Independently reviewed delivered control-flow closure.** Review the exact final source and call sites, with SHA/path/line evidence, from newAttestCmd's generation branch through real gates.RenderAttestation, its final Snapshot.Release/error join, and command error-to-exit handling. Establish that renderer success is the only path to the generation stdout write; the real renderer itself has no stdout emission; all non-nil renderer errors take the common failure branch before any write; that branch reports stderr and maps to exit 1; and no fallback, logging, deferred emission, retry or alternate generation path prints a partial document on an error. Verify final validation/cleanup is completed inside the renderer before it returns bytes, rather than deferred by a CLI-owned snapshot until after printing. Include command-output tracking and root command/error handling to the extent they can affect this path. Actual CLI ordinary-error tests must exercise this same renderer-error branch, not only Cobra unknown-flag/arity rejection. If the delivered implementation does not have this closure, compositional evidence is incomplete and cannot pass PM review.
> 
> 3. **Observed standalone process proof.** Build the ordinary isolated CLI from that exact GREEN revision, recording build command/binary path and source SHA (plus binary digest if used in the delivery evidence). Run actual generation success and ordinary missing/unreadable-input and alias failures using real fixture files; require exact exit 1, zero stdout bytes and the intended error category for failures. Retain the already-required real CLI changed-implementation, historical/current distinction, missing implementation, migration and compatibility cases. Do not replace them with renderer-only tests, package mocks or source inspection. Existing A/B/C/D staging remains unchanged: unsupported new flags/version on base is INTERFACE_ABSENT, not freshness or late-failure proof.
> 
> Final report wording must distinguish the evidence tiers: `OBSERVED: real renderer late validation/cleanup failure -> nil bytes; OBSERVED: built CLI success and ordinary renderer-input/alias failure -> expected output/exit; REVIEWED: exact delivered renderer-to-CLI error/output closure; COMPOSED: CLI late renderer failure -> exit 1/empty stdout; UNOBSERVED: independently injected standalone CLI late-release failure; UNFORCED: individual OS root-handle/filelock Close primitive errors.` The approved actual private-copy cleanup failure supplies the lifecycle-fault alternative; every OS Close primitive need not be forced. None of these observations establishes authenticated execution or reviewer honesty.
> 
> Option B is not necessary merely because the standalone late-fault injection is unobserved. The new generation adapter has a reviewable error/output boundary, and real renderer faults plus actual CLI traversal of that same error branch can establish the bounded guarantee compositionally. B would require a further concrete review only if final delivered wiring prevents establishing that closure or an explicit requirement is added for independently injected standalone CLI late-fault observation. Do not infer a production fault environment variable, exported callback, CLI refactor or wider ownership from this clarification.
> 
> Source status: exact a82277a cmd/machinery/attest.go currently only calls stableAttestationHashes and then prints buffered old filehash output; it DOES NOT call the future RenderAttestation API. The inspected current io.go/commandResult/main.go path maps command errors to nonzero exit and tracks write errors, but this does not establish the future generation path. Exact diff a82277a..6cb2d974 on these relevant attestation/error-output/suite paths was empty. Delivered GREEN wiring/source/error/defer closure is therefore a mandatory future independent review obligation, not an already verified implementation or executed proof.
> 
> ## Approved ownership, exact authorized test amendments and cost
> 
> Retain declared ownership: internal/gates/attest.go; internal/gates/attest_implementation_test.go (new); cmd/machinery/attest.go; docs/attestation-evidence.md. Approved directly related additions: internal/gates/suite.go only Gv capture/private option/wiring; cmd/machinery/check.go only Gv-facing help/messages; internal/gates/attest_test.go; cmd/machinery/attest_test.go; cmd/machinery/attest_implementation_test.go (new real process/local-Git tests). Optional internal/gates/attest_green_test.go only if independent GREEN supplemental proof needs a separate file, not to change frozen RED. Historical/Ga tests can live in the attestation implementation tests without editing accept.go or accept_test.go.
> 
> Approved concrete scope expansion: internal/designlock/attestation_snapshot.go (new capability) and internal/designlock/attestation_snapshot_test.go (new real FS/custody tests); internal/hook/hook.go only stop's finalization-before-decision/state-clear ordering; internal/hook/attestation_snapshot_test.go new real-event/ledger tests. The suite.go ownership includes capture lifecycle, pending-result finalization in Release, and explicit-release convenience wrappers described above. designlock.go/external_snapshot.go/scale.go/source_snapshot.go/snapshot_inventory.go/portablepath/cmd/machinery/hook.go and generic snapshot callers remain read-only. No unspecified helper factoring is authorized. No new ownership is implied for MAC-olrx/MAC-p8ce files or execution/container policy.
> 
> Exact legacy amendments already authorized by independent PM (no broader changes):
> 
> 1. internal/gates/attest_test.go TestAttestationMutations, case "wrong version" around line 176: change input version 2 to 3 and expected supported-version diagnostic to 1-or-2; add separate accepted-v2 and malformed-v2 cases. Do not merely delete the assertion.
> 2. cmd/machinery/attest_test.go TestAttestRejectsIdentityAliases: independent PM selected the existing mandatory alias-case repair, replacing the os.Link failure t.Skipf with t.Fatalf("create required hard-link alias fixture: %v", err); preserve the real alias, CLI invocation and existing failure/empty-stdout/diagnostic assertions. No alternative case is authorized and required alias proof may not skip.
> 3. Keep existing six-g2 clean/count/coverage-warning tests and explicit-file custody tests unchanged unless the reviewed implementation demonstrates another exact necessary amendment. Existing Gv coverage tests retain partial design-plan adoption semantics. Exact epic text search of attestation_version/attestEvidence/attestRowFor in internal/gates and cmd/machinery, plus the three behavioral claim IDs, found no additional legacy behavioral current-success fixture in that bounded scope. Other hits are cmd/machinery/repository_contract_test.go's role-document vocabulary assertion and internal/gates/failclosed_io_test.go TestAttestationPackTraversalErrorIsBlocking; neither changes. Preserve internal/gates/determinism_hardening_test.go TestAttestationRejectsSymlinkReferent and TestAttestationClaimMustCoverItsSubject unchanged, including their legacy design-only rooted error behavior. If RED discovery finds another exact affected test elsewhere, request its named amendment; do not weaken it in GREEN.
> 
> Revised realistic forecast: 13 required files (the original 9 gates/CLI/docs/test paths, 2 new designlock paths, 2 hook paths), approximately 1,900–2,900 changed LOC (900–1,350 production including explicit held-root traversal/lifecycle, 850–1,350 tests, 150–200 docs/help). An optional supplemental GREEN test file would be a fourteenth, with its evidence purpose reported. A substantial full CLI complete-mode fixture could add cost; re-use existing real valid fixtures without rewriting their proof. No designlock.go/external_snapshot.go factoring is included. The independently reviewed 13-file/1,900–2,900 LOC forecast supersedes the original 4–7 files/<1000 estimate; do not omit custody finalization, hook decision safety or real CLI controls to meet the old estimate. Final report must give actual files/LOC and ownership. No preflight now; target gates/CLI plus new designlock/hook native tests only; full preflight remains epic-final responsibility.
> 
> 
> ## MANDATORY SKILLS
> - developer; codebase-memory; pm_acceptor.
> 
> ## Delivery Requirements
> Append exact RED/GREEN SHAs, commands/output, ownership inventory and per-AC proof. Use pvg story deliver; PM accepts separately. Never invoke pushing pvg story merge.
> 
> ## nd_contract
> status: new
> 
> ### evidence
> - Created 2026-09-05 from assessment and source-verified interfaces.
> 
> ### proof
> - [ ] AC #1: independently verified
> - [ ] AC #2: independently verified
> - [ ] AC #3: independently verified
> - [ ] AC #4: independently verified
> - [ ] AC #5: independently verified
> 

## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## Implementation Evidence

PROOF: RED-only delivery for MAC-p7jd. Product implementation and AC acceptance are NOT complete. Independent PM replay/approval and all GREEN proof remain pending.

### Revision and ownership

- Approved unchanged production base: a82277af5650b487cea1260c24ffcc1c86d69d8d.
- Original frozen RED: 7ec5d609acc1597ee2c0ddbf5401b92f834d5ab3.
- Exact PM-authorized setup repair: 47ba44906a09bc2fa010092a86b133d0d749c52c, subject `test(MAC-p7jd): tdd-red [test-edit-authorized] repair two fixture setup prerequisites`.
- Branch story/MAC-p7jd; retained worktree /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-p7jd. Clean after repair. No production/docs edits. Five changed test files: 1484 insertions, 3 deletions against base. Three new files total 1435 LOC (612 gates, 462 CLI, 361 hook).
- Exact repair diff: /tmp/machinery-p7jd-red-proof.RvOHwO/repair.patch, SHA256 2292e7da2123bdbc3890564468d52853c53306aa44e2efd0c9f60ef184e2a5e4. It contains only checked MkdirAll of the complete CLI fixture's two destinations, and hooks:true serialization plus immediate real Load(root) validation. All other bytes preserved. No amend/squash/rewrite.
- Remaining canonical scope: unchanged 13 required/optional 14 paths; PM-supported forecast 2710–3310 total, not a cap or authority to trim proof or expand ownership.

### CI/Test Results

Commands run:

Every command ran from `cd /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-p7jd`; no installed binary, network service, Docker, or Paivot product dependency. Go reports `go version go1.27.1 darwin/arm64`. Each selection uses native Go, real temporary filesystem; CLI tests build a private executable and use real bounded Git/process calls. CLI build digest logged: sha256:04e5c6fcd1f94574f547391ee2c3f39507db5b7df2b1ec0f09deda0db054dbbf.

1. `go test -count=1 -timeout=5m -json ./internal/gates -run Attest`
2. `go test -count=1 -timeout=5m -json ./cmd/machinery -run Attest`
3. `go test -count=1 -timeout=5m -json ./internal/hook -run Attestation`
4. `go test -count=1 -timeout=5m -json ./internal/designlock -run Attestation`
5. `pvg verify internal/gates/attest_implementation_test.go internal/gates/attest_test.go cmd/machinery/attest_implementation_test.go cmd/machinery/attest_test.go internal/hook/attestation_snapshot_test.go --format text`
6. `pvg story verify-tdd --range a82277af5650b487cea1260c24ffcc1c86d69d8d..47ba44906a09bc2fa010092a86b133d0d749c52c`
7. `git diff --check`; `git diff 7ec5d609 HEAD`; `git diff a82277a HEAD --stat`; SHA256 of all five files and original committed versions; raw JSONL/timing/leaf inventory hashes.

Tests were wrapped with shell `time`, with stdout in `<package>-repair.jsonl` and stderr/time in `<package>-repair.timing` under this report's directory. Exit status explicitly captured after each command. All processes terminated; no timeout, skip, build failure or missing runtime. No full preflight and no -cover, per dispatcher constraint pending separately owned MAC-yig6 fixture repair. Coverage percentage: NOT MEASURED; package selections are not line/branch coverage.

Summary:

| Package selection | Native leaves | Pass | Fail | D pass | Genuine A fail | B absent/diagnostic fail | C control unavailable, challenge unreached | Starts/terminals | Exit | Wall seconds |
|---|---:|---:|---:|---:|---:|---:|---:|---|---:|---:|
| gates Attest | 133 | 31 | 102 | 2 | 18 | 21 | 63 | 142/142 | 1 | 6.993 |
| cmd/machinery Attest | 28 | 11 | 17 | 2 | 6 | 1 | 10 | 29/29 | 1 | 4.005 |
| hook Attestation | 28 | 5 | 23 | 4 | 0 | 4 | 19 | 30/30 | 1 | 4.601 |
| designlock Attestation | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0/0 | 0 | 0.506 |
| Total | 189 | 47 | 142 | 8 | 24 | 26 | 92 | 201/201 | — | — |

No skips. Non-D passes: 28 existing gates controls plus one independent digest fixture vector, nine existing CLI controls, one existing hook selection control. The B digest-vector pass validates independent test grammar, not product capability. Gates B failures include the one authorized wrong-version diagnostic expectation (integer 1 or 2), 17 v2-plan/closed-schema leaves and three kind/root leaves. Sixteen schema challenge legs fail their valid v2 control first; no schema sensitivity is credited. Hook C's 19 failures are also callback INTERFACE_ABSENT mechanically; categorized C to retain intended fault/control family and prevent claiming mutation execution. Count categories are mutually exclusive reporting buckets, not 26+92 distinct observed negative semantics.

The designlock command reports `testing: warning: no tests to run`; zero tests is explicitly NO capability proof. New designlock APIs and their tests are GREEN-only obligations, not baseline-compatible RED.

Native leaf inventories, including exact names, terminal status and per-leaf elapsed values:

- gates-repair-leaves.tsv SHA256 20ce84c46812bc7653788bb6e2c56a800a085b58bb607549868fad6aa177573b.
- cli-repair-leaves.tsv SHA256 c0472619543abcf439c6d48c3dc2b6c58da87c63560bed28606049d15d8b6f3b.
- hook-repair-leaves.tsv SHA256 73f7e4cb7d0924743fcf36e90e649b544d6419658d948113984147432504e324.
- designlock-repair-leaves.tsv is empty SHA256 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855.

Inventories derive from full parsed JSONL: collect unique run Test names, retain pass/fail/skip terminal events whose names have no child name prefix. Starts/terminals include parent tests; leaves do not double count them. Raw hook subprocess output is embedded in parent logs; 23 child scenario executions are represented by the 23 parent scenario leaves, not inflated into additional independent passes.

Raw replay log SHA256:

- gates-repair.jsonl 367b42cff448d0840cdfe203a0791c0bdeaaa3c6b693d94978bfdb67e9b5bad4; timing 0dff428ebfff1778c7244969c0f87014f41a023a12b0b6aecd49a9877116fda4. JSON event range 2026-09-05T20:15:01.640304-07:00 through 20:15:07.963944-07:00.
- cli-repair.jsonl ca105ed15d24a1175dad98838990e3f0bff658e03bbd5fa9018b43635dd48820; timing 79bc63b5a34e496c74b420b139f64860c51258233f8b518eee11a56a3bbc197a. Event range 20:15:16.671789-07:00 through 20:15:19.597079-07:00.
- hook-repair.jsonl 88f19ff77121699d9912a72cedb3c349f2de9cbc79b900260e5fd8d3fabe9e7c; timing 13d70c223fbf448579366ede2016968fc0eb5ea0c7ce211d886170eb758c08a1. Event range 20:15:26.911037-07:00 through 20:15:31.019569-07:00.
- designlock-repair.jsonl d731200774545f86037c6783d0b64c95e4cb128dbdf7379d2461926da376e109; timing da3c1b29f5d0b71939c14248ef68286204d65b22ad3c504d8e9d790987d201e8. Event range 20:15:38.259971-07:00 through 20:15:38.563773-07:00.
- verify-repair.txt 58f6786769382040ae1cc2005d92b28a359605afa74eac5115ac10dd7266e57d: `VERIFY: PASSED (5 files scanned, 0 issues)`.
- hard-TDD guard: checked two commits, skipped zero merges, PASS no unauthorized test edits. This is structural, not independent behavioral approval.

### Failure causes and positive controls

A is reached genuine baseline false acceptance: three legacy behavioral claims (gt.conformance-test-shape, g4.standin-coverage, g4.pack-event-discipline), each unchanged or after actual handler/assertion mutation. Gates exercise direct CheckAttestations, suite with implementation and suite without it: 18 cases accept design covers without GV_MISSING_IMPLEMENTATION_SUBJECT. Six actual CLI cases return exit 0 instead of required exit 1/missing-subject diagnostic. These are existing-interface behavioral assertion failures, not parse/permission/runtime failures.

D passes: gates full/partial six-claim g2 legacy plan controls preserve counts and 0/5 warnings; built CLI preserves filehash, --claims precedence, legacy plan and real Git ancestor Ga acceptance; four hook Stop/SubagentStop/semantic-warning/wave policies pass on actual existing Run with valid configuration and plain writer. New Hooks:true fixtures now pass immediate Load(root) with no warning.

B/C limits: current production rejects v2 with `attestation_version must be the integer 1`, rejects generation flags with `unknown flag: --design`, and never invokes the optional hook writer callback. Callback fired count is explicitly zero with no fault credited. C tests require unchanged current/valid-v2 control before performing mutation/fault and final assertions; these legs are NOT YET EXERCISED. Frozen optional interface compiles against existing types but does not imply production activation.

Complete-mode precise stage now reached: checked design/src destinations were created; full real go-crm fixture copied, Git initialized/committed, acceptance anchor fields rewritten, old claims read. The first real generation call for g2.action-ownership kind plan exits 1, stdout empty, `unknown flag: --design`. No full-v2 document, unchanged --complete control or sole-current-warning mutation has yet executed. The repair proves that setup advanced to the intended missing interface, NOT that all later complete-mode gates will pass.

### Frozen file SHA256 before/after repair

| File | Original 7ec5d609 | Repaired 47ba449 |
|---|---|---|
| internal/gates/attest_implementation_test.go | fcae6e3a9dc66d8fb9abb1d151129604f59a3e38d9ac221d4f16ce91a8086260 | identical |
| internal/gates/attest_test.go | f0eef53c28fdf940654700832887f2c326a478c5df1d741ae9f538a3d24db253 | identical |
| cmd/machinery/attest_test.go | beb2d58af4d3144316f3789c3b7cbb726a4761af57ea85f0017297b9c24cd6cd | identical |
| cmd/machinery/attest_implementation_test.go | 62511c2a3ab4f04cf8b4bfa9fbe7bd0e0ec2e02c850e2e4726d0d5f198ccd58a | e02ad18fe2745d23f05e69ce1e01dd2b1be358f94d99cb3a215b14f6e2cee3b8 |
| internal/hook/attestation_snapshot_test.go | c3f595d56538b97f47e9f8faa66e731fb371cba0d8256f0ddfb38f9ac421f021 | a3c088c095e2dba4379f548903a058eedbd5338ef872e507ab12edaa1ca06e49 |

Original evidence preserved unchanged in this directory: gates.jsonl SHA256 4a62a3ee993dd8b082b6a4299342333fa735ad65bc0ece6007a336febf725749 (133 leaves,31 pass,102 fail,6.501s); cli.jsonl a0fccbdfac2294d5292fc1cd879ca1ad0dbbb13128b16943401ed775373e35a6 (28,11 pass,17 fail,2.866s); hook.jsonl ff109b68b5b8399e54a5e57765d9d205ae831d7e646de527f859e232248cdbe0 (28,1 pass,27 SETUP fail,1.345s); designlock.jsonl 1fdbaf715ac9fe57f94a40d00e9ab27fe55433926478adb7d6aaaaeeea2d92f5 (zero tests,0.288s). Original CLI includes one SETUP failure (missing destination), six A, ten B/C. Original hook's hooks:null errors are SETUP regardless of their printed B assertion label. Independent PM authorized exact repairs before edits and retained all original history. No further fixture repairs were made.

### AC Verification

| AC | Frozen test mapping | RED evidence / remaining final proof |
|---|---|---|
| 1 complete implementation inventory and content | gates CScopeMutations, CTopologyAndEvidenceBoundary, CReceiptCannotNarrowScope; CLI B-generation-inventory/C-real-process | Independent digest fixture control passes; complete entries/content/topology mutation legs wait for valid v2. Whole root including hidden/untracked/ignored/vendor/build/config and exact exclusions asserted, not observed implemented. |
| 2 plan/current/historical separation | gates BKindAndMissingRoot/BV2PlanAndMalformedSchema; CLI C-historical-current-replay/C-plan-warning-promotion/C-complete-sole-current-warning | D real Ga ancestor and legacy plan pass. New schema/kind/current/history/sole-warning semantics interface-absent or unreached. Docs/help/consumer diagnostics await GREEN. |
| 3 explicit legacy migration | gates ARejectsLegacyBehavior and DBaselinePlanControls; CLI A-legacy and D-filehash-claims-plan | 24 genuine false-acceptance A failures and eight D compatibility controls across selected packages; missing-subject behavior requires GREEN. |
| 4 real negative and positive scope/custody cases | gates C mutation/topology/receipt/lifecycle families; CLI C real-process/history; hook process matrix | Frozen assertions cover real files, scope alias/symlink, assertion removal, handler change, stale replay, evidence-only commit, retained capability finalization. Current controls/fault stages unreached; callback absent. All must reach unchanged/fired controls on GREEN. |
| 5 actual CLI and truthful guarantees | built CLI suite plus hook matrix; optional GREEN renderer file and new designlock tests owed | OBSERVED existing CLI A/D integration only. Future generation/current output/custody not observed. Same delivered GREEN revision must meet the conjunctive AC5 composition below. |

### Mandatory GREEN supplements and observation boundary

Frozen files remain unchanged throughout GREEN. New API tests go only in authorized new internal/designlock/attestation_snapshot_test.go and optional internal/gates/attest_green_test.go. Required: real held-root capability and inventory bounds (including overlay aggregate); approved lower fixed budget factory and exact existing read-chunk callback uses; actual late original mutation and real owned private-copy cleanup failure; mandatory callback-fired counts and matched no-fault controls. Do not fake Gate values/errors, recursively Release from callback, or add fault env flags. Every v2 negative must pass its valid control before the challenge. Hook finalization before output/ledger clear must fail closed even strict:false/wave/empty selection; direct lifecycle must publish all-or-none only after release and preserve latching/idempotency.

AC5 is conjunctive on the SAME final GREEN revision: OBSERVED actual RenderAttestation late original mutation and real cleanup failure return nil bytes, with reached/no-fault callback controls; OBSERVED built CLI full document success and ordinary real renderer input/alias failures exit 1 with empty stdout; REVIEWED final CLI -> RenderAttestation -> Release/error joining -> output/defer/fallback closure. Late CLI guarantee is COMPOSED, not independently injected standalone late-process proof. Individual OS Close errors are UNFORCED. Output-sink partial writes remain a separate limit. Current a82277a production is still old stableAttestationHashes, not future renderer integration. Hashes bind observed scope/files, not execution, reviewer identity or judgment correctness.

### LEARNINGS

- Immediate real config validation prevents a hooks:null setup error from being mislabeled callback absence. Original misleading labels were corrected in evidence, never counted as RED behavior.
- Existing copyDirInto requires the destination root; preserve shared helpers and repair only explicitly authorized fixture prerequisites.
- Interface-absent controls must halt mutation credit. A callback-shaped test method is not evidence that production finalization reached it.
- Full independent scope and real CLI/hook lifecycle proof costs more than the original test estimate; PM investigated the overrun rather than trimming proof or introducing shared unapproved APIs.
- A packaging-only hash loop briefly used zsh's special `path` variable, hiding commands in that one shell. It changed no files or tests; rerun with task_file produced all hashes and exact diff. Test logs/runs are unaffected.

## nd_contract
status: delivered

### evidence
- RED-only tests committed at 47ba44906a09bc2fa010092a86b133d0d749c52c, preserving original 7ec5d609 and unchanged a82277a production. Exact independent PM repair authorization followed.
- Full scoped replay: 189 native leaves, 47 passes, 142 expected/absent-interface failures, zero skips; 24 genuine A failures, eight D passes, 26 B failures, 92 unreached C family cases. No setup failures remain in this replay; no current custody guarantee claimed.
- pvg verify PASS five files/zero issues; verify-tdd PASS two commits/no unauthorized edits. Independent PM review/approve-red pending; this is not GREEN acceptance.
- Complete report, exact leaf inventories, raw logs, timings, hashes and repair diff: /tmp/machinery-p7jd-red-proof.RvOHwO/.

### proof
- [x] RED AC #1: frozen full-scope/inventory cases compile and valid-control failures are classified; final GREEN implementation proof pending.
- [x] RED AC #2: plan/current/history tests and real legacy controls supplied; new semantics remain unexercised until GREEN.
- [x] RED AC #3: 24 genuine legacy false-acceptance failures with passing compatibility controls supplied.
- [x] RED AC #4: negative/positive scope and custody contracts frozen; C fault execution and approved new API supplements remain mandatory in GREEN.
- [x] RED AC #5: actual baseline CLI integration and explicit same-revision AC5 composition limits recorded; renderer late faults, new CLI behavior and final wiring review remain mandatory in GREEN.

## nd_contract
status: in_progress

### evidence
- Architecture revision 2 independently APPROVED; unchanged authority SHA256 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179.
- Exact two existing-test amendments and four bounded seam uses already authorized by independent PRE-RED PM review, preserved in full; this Sr PM clarification grants no additional seam or test scope.
- Original architect and challenger independently APPROVED AC5 composition clarification SHA256 5fc3193106a803b7760020b646ed4db7b3d417f5e196b2266d10cf8227733937. Canonical exact row, conjunctive same-GREEN-revision evidence bar and observation limits now embedded.
- Five product AC byte-identical; previous canonical and all existing history preserved. Actual command strings and unique active section structure semantically reread after literal-safe supported edit; no reliance on constructed-text equality alone.
- Current a82277a CLI contains old stableAttestationHashes path, NOT future RenderAttestation generation integration. Delivered source/output/error/defer closure remains an unexecuted mandatory final review obligation.
- Bounded RED authoring may resume under the existing PM authorization and dispatcher scheduling. Independent RED replay/approve-red, GREEN implementation, tests and final acceptance remain pending. No observed behavioral proof is claimed by this contract repair.
- Status in_progress, hard-tdd, assignee dev-MAC-p7jd, dependencies and healthy retained worktree/claim unchanged; no source/test/docs, installed assets, remote or preflight changes.

### proof
- [ ] AC #1: complete scope/inventory binding requires frozen RED and GREEN proof
- [ ] AC #2: closed plan/current/history semantics require behavioral replay
- [ ] AC #3: legacy migration requires A failures on base and D passing compatibility controls
- [ ] AC #4: authorized real mutation/custody/lifecycle tests require reached final proof
- [ ] AC #5: same delivered GREEN revision must supply OBSERVED real renderer late validation/cleanup failure with nil bytes; OBSERVED built CLI success and ordinary real renderer-error/alias failures with exact output/exit; REVIEWED final renderer-to-CLI output/error/defer closure. Late CLI guarantee is COMPOSED, not independently observed injection; standalone late injection remains UNOBSERVED and individual OS Close errors UNFORCED. Actual cleanup cause, existing real CLI freshness/history/migration cases and output-sink partial-write limit remain mandatory.


## nd_contract
status: in_progress

### evidence
- Independent PRE-RED PM exact two-test amendments and four bounded seam/use decisions recorded above after canonical and exact source review.
- Claim, hard-tdd, parent and status intentionally retained; A/B/C/D execution, frozen RED review and all GREEN implementation proof remain pending.
- Standalone CLI late-release proof wording requires explicit interpretation before approve-red; OS close unforced boundary disclosed, actual cleanup alternative authorized.

### proof
- [ ] AC #1: complete scope/inventory binding requires frozen RED and GREEN proof
- [ ] AC #2: closed plan/current/history semantics require behavioral replay
- [ ] AC #3: legacy migration requires A failures on base and D compatibility controls
- [ ] AC #4: approved real mutation/custody/lifecycle cases require reached final proof
- [ ] AC #5: actual CLI/history/hook behavior and accurate limits require final replay


## USER INTENT
Strengthen Machinery mission-critical assurance with observable fail-closed behavior and precise limits.

## Context (Embedded)
Assessment F3: gt.conformance-test-shape covers BUILD artifacts, while g4.pack-event-discipline covers pack. Neither binds tested/reviewed implementation. Historic Ga ancestor acceptance is valid history, not current-tree assurance. Preserve judgment-vs-mechanical distinction.

## Ownership
Own only internal/gates/attest.go, internal/gates/attest_implementation_test.go, cmd/machinery/attest.go, docs/attestation-evidence.md and directly associated tests. You are not alone: preserve other edits and coordinate shared paths.

## Boundary Map
PRODUCES:
- internal/gates/attest.go -> hardened contract and regression evidence
- internal/gates/attest_implementation_test.go -> hardened contract and regression evidence
- cmd/machinery/attest.go -> hardened contract and regression evidence
- docs/attestation-evidence.md -> hardened contract and regression evidence
CONSUMES:
- Existing Machinery implementation.
  spec: attestationRequiredPaths(g *Gate, design, claim string) []string; stableAttestationHashes(paths []string) ([]string, error)

### Story Acceptance Criteria
1. Implementation/test behavior claims bind a complete explicit implementation/test scope under rooted inventory and content hashes, not only BUILD or pack. Any code/test/config addition, removal, rename, content change or scope narrowing affecting the claim invalidates freshness.
2. Distinguish plan-only claims, current implementation review and historical milestone acceptance in closed schema, gate diagnostics and docs. Historical ancestor records remain historical; they cannot alone imply current implementation approval.
3. Provide explicit compatibility migration for existing attestations. Legacy design-only covers never quietly grandfather implementation assertions as fresh; users receive actionable missing-subject diagnostics.
4. Negative tests remove assertions after review, alter event handlers, add excluded files, change scope, alias paths/symlinks and replay stale attestations. Positive unchanged reviewed scope and harmless evidence-only commit remain usable.
5. Real CLI attest/check path exercises changed implementation and historical/current distinction. Hashing proves binding, not reviewer honesty or that tests executed; output never claims otherwise.

## Testing Requirements
- Explicit hard TDD: separate RED test author, expected behavioral assertion failure plus passing control on unmodified production, independent PM replay, frozen RED test/fixture bytes retained through GREEN. Compilation, import, timeout or unavailable infrastructure are not valid RED. Existing tests encoding the unsafe contract may be corrected during reviewed RED; GREEN cannot silently weaken them.
- Integration tests: MANDATORY (no mocks). Exercise real process/filesystem/runtime boundaries; no stubs, no skip-if-missing. Fixture policy inputs may test parser logic but are not live remote proof.
- go test ./internal/gates -run Attest; go test ./cmd/machinery -run Attest; CLI integration with real temp git repository and design+implementation roots.
- Full scripts/preflight.sh ONLY at end of epic. No pushes, sync, remote mutation, installed binary/plugin/agent/skill replacement, or dev-link. Another agent uses installed Machinery in NIL; builds/tests use isolated output and homes.
- Shipped product enforcement is standalone Machinery: never requires pvg, nd, Paivot labels/metadata or commit conventions. Paivot is local development coordination only.

## OUT OF SCOPE
- Other assessment areas have sibling stories; final preflight/local main merge/isolated candidate binary are final gate responsibilities. Do not omit small directly related safety fixes.

## DIFF BUDGET
- ~4-7 files, under 1000 changed LOC; overrun triggers PM investigation rather than weaker proof.

## MANDATORY SKILLS
- developer; codebase-memory; pm_acceptor.

## Delivery Requirements
Append exact RED/GREEN SHAs, commands/output, ownership inventory and per-AC proof. Use pvg story deliver; PM accepts separately. Never invoke pushing pvg story merge.

## nd_contract
status: new

### evidence
- Created 2026-09-05 from assessment and source-verified interfaces.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified

</previous_canonical_description>

## nd_contract
status: in_progress

### evidence
- Full approved revision 2 read and SHA256 verified; canonical description guarded-edit/readback verified; all five user AC and prior history retained.
- Architecture contract ready for independent PM exact test-edit/seam authorization; no RED run, delivery or implementation proof claimed.

### proof
- [ ] AC #1: approved complete scope contract; RED/GREEN implementation proof pending
- [ ] AC #2: approved plan/current/history schema; behavioral proof pending
- [ ] AC #3: approved explicit migration; behavioral proof pending
- [ ] AC #4: approved real mutation/custody/lifecycle matrix; phase authorization and proof pending
- [ ] AC #5: approved real CLI/hooks/history scope and honest limits; proof pending

## nd_contract
status: in_progress

### evidence
- Bounded same-story contract triage completed; RED remains intentionally paused before edits at its healthy retained a82277a worktree.
- Scope/API/schema proposal awaits read-only contract author and independent review. No status, labels, claim, dependency or worktree changes made by this triage.

### proof
- [ ] AC #1: implementation scope/inventory contract and RED/GREEN proof pending
- [ ] AC #2: plan/current/history schema and diagnostics contract pending
- [ ] AC #3: explicit migration and missing-subject proof pending
- [ ] AC #4: matched positive/negative real custody tests pending
- [ ] AC #5: real CLI freshness/history proof and accurate limits pending

## History
- 2026-09-05T19:35:07Z dep_added: blocks MAC-vx24
- 2026-09-05T19:35:08Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:16Z dep_added: blocks MAC-ou97
- 2026-09-05T23:09:26Z status: open -> in_progress
- 2026-09-05T23:09:26Z auto-follows: linked to predecessor MAC-p8ce
- 2026-09-05T23:09:26Z claimed by dev-MAC-p7jd
- 2026-09-06T03:17:09Z dep_added: blocks MAC-hgz1
- 2026-09-06T03:19:43Z status: in_progress -> in_progress
- 2026-09-06T03:19:43Z auto-follows: linked to predecessor MAC-2u36
- 2026-09-06T03:27:48Z status: in_progress -> open
- 2026-09-06T04:10:43Z status: open -> in_progress
- 2026-09-06T04:10:43Z auto-follows: linked to predecessor MAC-a89e
- 2026-09-06T04:10:43Z claimed by dev-MAC-p7jd
- 2026-09-06T06:03:26Z status: in_progress -> in_progress
- 2026-09-06T06:03:26Z auto-follows: linked to predecessor MAC-olrx
- 2026-09-06T06:30:50Z status: in_progress -> open
- 2026-09-06T06:30:50Z released by ramirosalas
- 2026-09-06T06:45:46Z status: open -> in_progress
- 2026-09-06T06:45:46Z claimed by dev-MAC-p7jd
- 2026-09-06T07:00:52Z status: in_progress -> in_progress
- 2026-09-06T07:11:34Z status: in_progress -> open
- 2026-09-06T07:12:20Z released by ramirosalas
- 2026-09-06T07:29:40Z status: open -> in_progress
- 2026-09-06T07:29:40Z claimed by dev-MAC-p7jd
- 2026-09-06T07:38:05Z status: in_progress -> in_progress

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-gcrr]], [[MAC-ou97]], [[MAC-hgz1]]
- Follows: [[MAC-p8ce]], [[MAC-2u36]], [[MAC-a89e]], [[MAC-olrx]]

## Comments

### 2026-09-06T00:40:28Z ramirosalas
## nd_contract
status: in_progress

### evidence
- Architecture revision 2 independently APPROVED; unchanged authority SHA256 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179.
- Exact two existing-test amendments and four bounded seam uses already authorized by independent PRE-RED PM review, preserved in full; this Sr PM clarification grants no additional seam or test scope.
- Original architect and challenger independently APPROVED AC5 composition clarification SHA256 5fc3193106a803b7760020b646ed4db7b3d417f5e196b2266d10cf8227733937. Canonical exact row, conjunctive same-GREEN-revision evidence bar and observation limits now embedded.
- Five product AC byte-identical; previous canonical and all existing history preserved. Actual command strings and unique active section structure semantically reread after literal-safe supported edit; no reliance on constructed-text equality alone.
- Current a82277a CLI contains old stableAttestationHashes path, NOT future RenderAttestation generation integration. Delivered source/output/error/defer closure remains an unexecuted mandatory final review obligation.
- Bounded RED authoring may resume under the existing PM authorization and dispatcher scheduling. Independent RED replay/approve-red, GREEN implementation, tests and final acceptance remain pending. No observed behavioral proof is claimed by this contract repair.
- Status in_progress, hard-tdd, assignee dev-MAC-p7jd, dependencies and healthy retained worktree/claim unchanged; no source/test/docs, installed assets, remote or preflight changes.

### proof
- [ ] AC #1: complete scope/inventory binding requires frozen RED and GREEN proof
- [ ] AC #2: closed plan/current/history semantics require behavioral replay
- [ ] AC #3: legacy migration requires A failures on base and D passing compatibility controls
- [ ] AC #4: authorized real mutation/custody/lifecycle tests require reached final proof
- [ ] AC #5: same delivered GREEN revision must supply OBSERVED real renderer late validation/cleanup failure with nil bytes; OBSERVED built CLI success and ordinary real renderer-error/alias failures with exact output/exit; REVIEWED final renderer-to-CLI output/error/defer closure. Late CLI guarantee is COMPOSED, not independently observed injection; standalone late injection remains UNOBSERVED and individual OS Close errors UNFORCED. Actual cleanup cause, existing real CLI freshness/history/migration cases and output-sink partial-write limit remain mandatory.

Structural checks: scoped backlog lint PASSED (33 issues, 0 errors, 0 review findings); no dependency cycles; scoped RTM PASSED (18 stories/3 closed; 0 tagged requirements extracted/0 uncovered). These checks are structural, not product AC proof. This terminal full contract supersedes older trailing architecture-pending blocks without deleting history.

### 2026-09-06T03:12:26Z ramirosalas
## nd_contract
status: in_progress

### evidence
- Independent PM POST-FREEZE exact setup-repair authorization is recorded in Notes for internal/hook/attestation_snapshot_test.go hookReviewFixture and cmd/machinery/attest_implementation_test.go C-complete-sole-current-warning, against frozen RED 7ec5d609acc1597ee2c0ddbf5401b92f834d5ab3. Only those setup hunks are authorized; repair commit requires tdd-red and [test-edit-authorized].
- Original logs and hashes retained. Original 27 new hook failures and the one CLI complete fixture failure are SETUP, not behavioral RED. Complete repaired replay and independent RED review/freeze remain mandatory; no red-approved, delivery, rejection, GREEN dispatch or acceptance granted.
- PM investigated the test overrun and supports same-story ~2710–3310 forecast within unchanged 13 required/optional14 paths; SrPM canonical estimate update pending, with no proof trimming or scope expansion.
- R2 authority 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179 and approved AC5 composition 5fc3193106a803b7760020b646ed4db7b3d417f5e196b2266d10cf8227733937 remain authoritative. Exact prior seam restrictions and five product AC unchanged.
- Status in_progress, hard-tdd, assignee dev-MAC-p7jd and healthy retained claim/worktree preserved. PM reviewed source/logs only and made tracker notes only.

### proof
- [ ] AC #1: complete scope/inventory binding requires frozen corrected RED and final GREEN proof
- [ ] AC #2: closed plan/current/history semantics require behavioral replay
- [ ] AC #3: A legacy false acceptance and D passing compatibility controls require complete independent repaired replay
- [ ] AC #4: real mutation/custody/lifecycle tests must reach intended boundaries; SETUP is not proof
- [ ] AC #5: unchanged conjunctive same-GREEN-revision bar: OBSERVED renderer real late validation/cleanup failure -> nil bytes, OBSERVED built CLI success/ordinary renderer-error/alias output/exit, REVIEWED delivered renderer-to-CLI error/output closure; late CLI guarantee COMPOSED, independent standalone late injection UNOBSERVED, individual OS Close errors UNFORCED. Real cleanup cause, freshness/history/migration and output-sink limit remain required.


### 2026-09-06T03:19:56Z ramirosalas
## Implementation Evidence

PROOF: RED-only delivery for MAC-p7jd. Product implementation and AC acceptance are NOT complete. Independent PM replay/approval and all GREEN proof remain pending.

### Commit

- Branch: story/MAC-p7jd
- SHA: 47ba44906a09bc2fa010092a86b133d0d749c52c

### Revision and ownership

- Approved unchanged production base: a82277af5650b487cea1260c24ffcc1c86d69d8d.
- Original frozen RED: 7ec5d609acc1597ee2c0ddbf5401b92f834d5ab3.
- Exact PM-authorized setup repair: 47ba44906a09bc2fa010092a86b133d0d749c52c, subject `test(MAC-p7jd): tdd-red [test-edit-authorized] repair two fixture setup prerequisites`.
- Branch story/MAC-p7jd; retained worktree /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-p7jd. Clean after repair. No production/docs edits. Five changed test files: 1484 insertions, 3 deletions against base. Three new files total 1435 LOC (612 gates, 462 CLI, 361 hook).
- Exact repair diff: /tmp/machinery-p7jd-red-proof.RvOHwO/repair.patch, SHA256 2292e7da2123bdbc3890564468d52853c53306aa44e2efd0c9f60ef184e2a5e4. It contains only checked MkdirAll of the complete CLI fixture's two destinations, and hooks:true serialization plus immediate real Load(root) validation. All other bytes preserved. No amend/squash/rewrite.
- Remaining canonical scope: unchanged 13 required/optional 14 paths; PM-supported forecast 2710–3310 total, not a cap or authority to trim proof or expand ownership.

### CI/Test Results

Commands run:

Every command ran from `cd /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-p7jd`; no installed binary, network service, Docker, or Paivot product dependency. Go reports `go version go1.27.1 darwin/arm64`. Each selection uses native Go, real temporary filesystem; CLI tests build a private executable and use real bounded Git/process calls. CLI build digest logged: sha256:04e5c6fcd1f94574f547391ee2c3f39507db5b7df2b1ec0f09deda0db054dbbf.

1. `go test -count=1 -timeout=5m -json ./internal/gates -run Attest`
2. `go test -count=1 -timeout=5m -json ./cmd/machinery -run Attest`
3. `go test -count=1 -timeout=5m -json ./internal/hook -run Attestation`
4. `go test -count=1 -timeout=5m -json ./internal/designlock -run Attestation`
5. `pvg verify internal/gates/attest_implementation_test.go internal/gates/attest_test.go cmd/machinery/attest_implementation_test.go cmd/machinery/attest_test.go internal/hook/attestation_snapshot_test.go --format text`
6. `pvg story verify-tdd --range a82277af5650b487cea1260c24ffcc1c86d69d8d..47ba44906a09bc2fa010092a86b133d0d749c52c`
7. `git diff --check`; `git diff 7ec5d609 HEAD`; `git diff a82277a HEAD --stat`; SHA256 of all five files and original committed versions; raw JSONL/timing/leaf inventory hashes.

Tests were wrapped with shell `time`, with stdout in `<package>-repair.jsonl` and stderr/time in `<package>-repair.timing` under this report's directory. Exit status explicitly captured after each command. All processes terminated; no timeout, skip, build failure or missing runtime. No full preflight and no -cover, per dispatcher constraint pending separately owned MAC-yig6 fixture repair. Coverage percentage: NOT MEASURED; package selections are not line/branch coverage.

Summary:

| Package selection | Native leaves | Pass | Fail | D pass | Genuine A fail | B absent/diagnostic fail | C control unavailable, challenge unreached | Starts/terminals | Exit | Wall seconds |
|---|---:|---:|---:|---:|---:|---:|---:|---|---:|---:|
| gates Attest | 133 | 31 | 102 | 2 | 18 | 21 | 63 | 142/142 | 1 | 6.993 |
| cmd/machinery Attest | 28 | 11 | 17 | 2 | 6 | 1 | 10 | 29/29 | 1 | 4.005 |
| hook Attestation | 28 | 5 | 23 | 4 | 0 | 4 | 19 | 30/30 | 1 | 4.601 |
| designlock Attestation | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0/0 | 0 | 0.506 |
| Total | 189 | 47 | 142 | 8 | 24 | 26 | 92 | 201/201 | — | — |

No skips. Non-D passes: 28 existing gates controls plus one independent digest fixture vector, nine existing CLI controls, one existing hook selection control. The B digest-vector pass validates independent test grammar, not product capability. Gates B failures include the one authorized wrong-version diagnostic expectation (integer 1 or 2), 17 v2-plan/closed-schema leaves and three kind/root leaves. Sixteen schema challenge legs fail their valid v2 control first; no schema sensitivity is credited. Hook C's 19 failures are also callback INTERFACE_ABSENT mechanically; categorized C to retain intended fault/control family and prevent claiming mutation execution. Count categories are mutually exclusive reporting buckets, not 26+92 distinct observed negative semantics.

The designlock command reports `testing: warning: no tests to run`; zero tests is explicitly NO capability proof. New designlock APIs and their tests are GREEN-only obligations, not baseline-compatible RED.

Native leaf inventories, including exact names, terminal status and per-leaf elapsed values:

- gates-repair-leaves.tsv SHA256 20ce84c46812bc7653788bb6e2c56a800a085b58bb607549868fad6aa177573b.
- cli-repair-leaves.tsv SHA256 c0472619543abcf439c6d48c3dc2b6c58da87c63560bed28606049d15d8b6f3b.
- hook-repair-leaves.tsv SHA256 73f7e4cb7d0924743fcf36e90e649b544d6419658d948113984147432504e324.
- designlock-repair-leaves.tsv is empty SHA256 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855.

Inventories derive from full parsed JSONL: collect unique run Test names, retain pass/fail/skip terminal events whose names have no child name prefix. Starts/terminals include parent tests; leaves do not double count them. Raw hook subprocess output is embedded in parent logs; 23 child scenario executions are represented by the 23 parent scenario leaves, not inflated into additional independent passes.

Raw replay log SHA256:

- gates-repair.jsonl 367b42cff448d0840cdfe203a0791c0bdeaaa3c6b693d94978bfdb67e9b5bad4; timing 0dff428ebfff1778c7244969c0f87014f41a023a12b0b6aecd49a9877116fda4. JSON event range 2026-09-05T20:15:01.640304-07:00 through 20:15:07.963944-07:00.
- cli-repair.jsonl ca105ed15d24a1175dad98838990e3f0bff658e03bbd5fa9018b43635dd48820; timing 79bc63b5a34e496c74b420b139f64860c51258233f8b518eee11a56a3bbc197a. Event range 20:15:16.671789-07:00 through 20:15:19.597079-07:00.
- hook-repair.jsonl 88f19ff77121699d9912a72cedb3c349f2de9cbc79b900260e5fd8d3fabe9e7c; timing 13d70c223fbf448579366ede2016968fc0eb5ea0c7ce211d886170eb758c08a1. Event range 20:15:26.911037-07:00 through 20:15:31.019569-07:00.
- designlock-repair.jsonl d731200774545f86037c6783d0b64c95e4cb128dbdf7379d2461926da376e109; timing da3c1b29f5d0b71939c14248ef68286204d65b22ad3c504d8e9d790987d201e8. Event range 20:15:38.259971-07:00 through 20:15:38.563773-07:00.
- verify-repair.txt 58f6786769382040ae1cc2005d92b28a359605afa74eac5115ac10dd7266e57d: `VERIFY: PASSED (5 files scanned, 0 issues)`.
- hard-TDD guard: checked two commits, skipped zero merges, PASS no unauthorized test edits. This is structural, not independent behavioral approval.

### Failure causes and positive controls

A is reached genuine baseline false acceptance: three legacy behavioral claims (gt.conformance-test-shape, g4.standin-coverage, g4.pack-event-discipline), each unchanged or after actual handler/assertion mutation. Gates exercise direct CheckAttestations, suite with implementation and suite without it: 18 cases accept design covers without GV_MISSING_IMPLEMENTATION_SUBJECT. Six actual CLI cases return exit 0 instead of required exit 1/missing-subject diagnostic. These are existing-interface behavioral assertion failures, not parse/permission/runtime failures.

D passes: gates full/partial six-claim g2 legacy plan controls preserve counts and 0/5 warnings; built CLI preserves filehash, --claims precedence, legacy plan and real Git ancestor Ga acceptance; four hook Stop/SubagentStop/semantic-warning/wave policies pass on actual existing Run with valid configuration and plain writer. New Hooks:true fixtures now pass immediate Load(root) with no warning.

B/C limits: current production rejects v2 with `attestation_version must be the integer 1`, rejects generation flags with `unknown flag: --design`, and never invokes the optional hook writer callback. Callback fired count is explicitly zero with no fault credited. C tests require unchanged current/valid-v2 control before performing mutation/fault and final assertions; these legs are NOT YET EXERCISED. Frozen optional interface compiles against existing types but does not imply production activation.

Complete-mode precise stage now reached: checked design/src destinations were created; full real go-crm fixture copied, Git initialized/committed, acceptance anchor fields rewritten, old claims read. The first real generation call for g2.action-ownership kind plan exits 1, stdout empty, `unknown flag: --design`. No full-v2 document, unchanged --complete control or sole-current-warning mutation has yet executed. The repair proves that setup advanced to the intended missing interface, NOT that all later complete-mode gates will pass.

### Frozen file SHA256 before/after repair

| File | Original 7ec5d609 | Repaired 47ba449 |
|---|---|---|
| internal/gates/attest_implementation_test.go | fcae6e3a9dc66d8fb9abb1d151129604f59a3e38d9ac221d4f16ce91a8086260 | identical |
| internal/gates/attest_test.go | f0eef53c28fdf940654700832887f2c326a478c5df1d741ae9f538a3d24db253 | identical |
| cmd/machinery/attest_test.go | beb2d58af4d3144316f3789c3b7cbb726a4761af57ea85f0017297b9c24cd6cd | identical |
| cmd/machinery/attest_implementation_test.go | 62511c2a3ab4f04cf8b4bfa9fbe7bd0e0ec2e02c850e2e4726d0d5f198ccd58a | e02ad18fe2745d23f05e69ce1e01dd2b1be358f94d99cb3a215b14f6e2cee3b8 |
| internal/hook/attestation_snapshot_test.go | c3f595d56538b97f47e9f8faa66e731fb371cba0d8256f0ddfb38f9ac421f021 | a3c088c095e2dba4379f548903a058eedbd5338ef872e507ab12edaa1ca06e49 |

Original evidence preserved unchanged in this directory: gates.jsonl SHA256 4a62a3ee993dd8b082b6a4299342333fa735ad65bc0ece6007a336febf725749 (133 leaves,31 pass,102 fail,6.501s); cli.jsonl a0fccbdfac2294d5292fc1cd879ca1ad0dbbb13128b16943401ed775373e35a6 (28,11 pass,17 fail,2.866s); hook.jsonl ff109b68b5b8399e54a5e57765d9d205ae831d7e646de527f859e232248cdbe0 (28,1 pass,27 SETUP fail,1.345s); designlock.jsonl 1fdbaf715ac9fe57f94a40d00e9ab27fe55433926478adb7d6aaaaeeea2d92f5 (zero tests,0.288s). Original CLI includes one SETUP failure (missing destination), six A, ten B/C. Original hook's hooks:null errors are SETUP regardless of their printed B assertion label. Independent PM authorized exact repairs before edits and retained all original history. No further fixture repairs were made.

### AC Verification

| AC | Frozen test mapping | RED evidence / remaining final proof |
|---|---|---|
| 1 complete implementation inventory and content | gates CScopeMutations, CTopologyAndEvidenceBoundary, CReceiptCannotNarrowScope; CLI B-generation-inventory/C-real-process | Independent digest fixture control passes; complete entries/content/topology mutation legs wait for valid v2. Whole root including hidden/untracked/ignored/vendor/build/config and exact exclusions asserted, not observed implemented. |
| 2 plan/current/historical separation | gates BKindAndMissingRoot/BV2PlanAndMalformedSchema; CLI C-historical-current-replay/C-plan-warning-promotion/C-complete-sole-current-warning | D real Ga ancestor and legacy plan pass. New schema/kind/current/history/sole-warning semantics interface-absent or unreached. Docs/help/consumer diagnostics await GREEN. |
| 3 explicit legacy migration | gates ARejectsLegacyBehavior and DBaselinePlanControls; CLI A-legacy and D-filehash-claims-plan | 24 genuine false-acceptance A failures and eight D compatibility controls across selected packages; missing-subject behavior requires GREEN. |
| 4 real negative and positive scope/custody cases | gates C mutation/topology/receipt/lifecycle families; CLI C real-process/history; hook process matrix | Frozen assertions cover real files, scope alias/symlink, assertion removal, handler change, stale replay, evidence-only commit, retained capability finalization. Current controls/fault stages unreached; callback absent. All must reach unchanged/fired controls on GREEN. |
| 5 actual CLI and truthful guarantees | built CLI suite plus hook matrix; optional GREEN renderer file and new designlock tests owed | OBSERVED existing CLI A/D integration only. Future generation/current output/custody not observed. Same delivered GREEN revision must meet the conjunctive AC5 composition below. |

### Mandatory GREEN supplements and observation boundary

Frozen files remain unchanged throughout GREEN. New API tests go only in authorized new internal/designlock/attestation_snapshot_test.go and optional internal/gates/attest_green_test.go. Required: real held-root capability and inventory bounds (including overlay aggregate); approved lower fixed budget factory and exact existing read-chunk callback uses; actual late original mutation and real owned private-copy cleanup failure; mandatory callback-fired counts and matched no-fault controls. Do not fake Gate values/errors, recursively Release from callback, or add fault env flags. Every v2 negative must pass its valid control before the challenge. Hook finalization before output/ledger clear must fail closed even strict:false/wave/empty selection; direct lifecycle must publish all-or-none only after release and preserve latching/idempotency.

AC5 is conjunctive on the SAME final GREEN revision: OBSERVED actual RenderAttestation late original mutation and real cleanup failure return nil bytes, with reached/no-fault callback controls; OBSERVED built CLI full document success and ordinary real renderer input/alias failures exit 1 with empty stdout; REVIEWED final CLI -> RenderAttestation -> Release/error joining -> output/defer/fallback closure. Late CLI guarantee is COMPOSED, not independently injected standalone late-process proof. Individual OS Close errors are UNFORCED. Output-sink partial writes remain a separate limit. Current a82277a production is still old stableAttestationHashes, not future renderer integration. Hashes bind observed scope/files, not execution, reviewer identity or judgment correctness.

### LEARNINGS

- Immediate real config validation prevents a hooks:null setup error from being mislabeled callback absence. Original misleading labels were corrected in evidence, never counted as RED behavior.
- Existing copyDirInto requires the destination root; preserve shared helpers and repair only explicitly authorized fixture prerequisites.
- Interface-absent controls must halt mutation credit. A callback-shaped test method is not evidence that production finalization reached it.
- Full independent scope and real CLI/hook lifecycle proof costs more than the original test estimate; PM investigated the overrun rather than trimming proof or introducing shared unapproved APIs.
- A packaging-only hash loop briefly used zsh's special `path` variable, hiding commands in that one shell. It changed no files or tests; rerun with task_file produced all hashes and exact diff. Test logs/runs are unaffected.

## nd_contract
status: delivered

### evidence
- RED-only tests committed at 47ba44906a09bc2fa010092a86b133d0d749c52c, preserving original 7ec5d609 and unchanged a82277a production. Exact independent PM repair authorization followed.
- Full scoped replay: 189 native leaves, 47 passes, 142 expected/absent-interface failures, zero skips; 24 genuine A failures, eight D passes, 26 B failures, 92 unreached C family cases. No setup failures remain in this replay; no current custody guarantee claimed.
- pvg verify PASS five files/zero issues; verify-tdd PASS two commits/no unauthorized edits. Independent PM review/approve-red pending; this is not GREEN acceptance.
- Complete report, exact leaf inventories, raw logs, timings, hashes and repair diff: /tmp/machinery-p7jd-red-proof.RvOHwO/.

### proof
- [x] RED AC #1: frozen full-scope/inventory cases compile and valid-control failures are classified; final GREEN implementation proof pending.
- [x] RED AC #2: plan/current/history tests and real legacy controls supplied; new semantics remain unexercised until GREEN.
- [x] RED AC #3: 24 genuine legacy false-acceptance failures with passing compatibility controls supplied.
- [x] RED AC #4: negative/positive scope and custody contracts frozen; C fault execution and approved new API supplements remain mandatory in GREEN.
- [x] RED AC #5: actual baseline CLI integration and explicit same-revision AC5 composition limits recorded; renderer late faults, new CLI behavior and final wiring review remain mandatory in GREEN.

### 2026-09-06T03:21:32Z ramirosalas
## nd_contract
status: delivered

### evidence
- Sr PM canonicalized only the final independent POST-FREEZE PM COST REVIEW: same-story forecast approximately2710–3310 total,13 required/optional14 paths unchanged. Components ~1485 RED +175–375 GREEN tests +900–1250 production +150–200 docs/help; no cap, new ownership/API/seam or proof trimming.
- Independently read exact PM authority and source stats: 7ec5d609 five files1475 insertions/3 deletions; repaired47ba449 five files1484 insertions/3 deletions (=1487 changed lines). Rounded forecast is honest estimate, not final measured total. Distinct gates/CLI/Git/hook custody matrices and existing helper reuse justify preserving proof rather than forced factoring.
- Only two canonical forecast paragraphs changed; all five AC, R2, AC5 compositional bar, frozen tests, exact existing amendments/seams and read-only boundaries unchanged. Existing RED-only delivered contract/state retained; no approve-red/GREEN acceptance by Sr PM.
- RED-only tests committed at 47ba44906a09bc2fa010092a86b133d0d749c52c, preserving original 7ec5d609 and unchanged a82277a production. Exact independent PM repair authorization followed.
- Full scoped replay: 189 native leaves, 47 passes, 142 expected/absent-interface failures, zero skips; 24 genuine A failures, eight D passes, 26 B failures, 92 unreached C family cases. No setup failures remain in this replay; no current custody guarantee claimed.
- pvg verify PASS five files/zero issues; verify-tdd PASS two commits/no unauthorized edits. Independent PM review/approve-red pending; this is not GREEN acceptance.
- Complete report, exact leaf inventories, raw logs, timings, hashes and repair diff: /tmp/machinery-p7jd-red-proof.RvOHwO/.

### proof
- [x] RED AC #1: frozen full-scope/inventory cases compile and valid-control failures are classified; final GREEN implementation proof pending.
- [x] RED AC #2: plan/current/history tests and real legacy controls supplied; new semantics remain unexercised until GREEN.
- [x] RED AC #3: 24 genuine legacy false-acceptance failures with passing compatibility controls supplied.
- [x] RED AC #4: negative/positive scope and custody contracts frozen; C fault execution and approved new API supplements remain mandatory in GREEN.
- [x] RED AC #5: actual baseline CLI integration and explicit same-revision AC5 composition limits recorded; renderer late faults, new CLI behavior and final wiring review remain mandatory in GREEN.


### 2026-09-06T03:28:54Z ramirosalas
## nd_contract
status: new

### evidence
- Independent PM RED APPROVED at 2026-09-06T03:27Z. Proper `pvg story approve-red MAC-p7jd` succeeded; readback is open with hard-tdd, red-approved, no delivered/accepted label; assignee dev-MAC-p7jd retained. This returns the story for GREEN, never accepts/closes it.
- Full independent PM report appended to Notes and retained at /tmp/machinery-p7jd-pm-red.fnQSrl/PM-RED-REPORT.md SHA25629d563f703679534f25cdd68cbb122e02918d826deb0f4ecfdaaec515d5a7590. It contains exact source review, author/independent hashes, all AC mapping, native leaf inventory paths, cause matrix, actual timings and mandatory supplements. Candidate47ba44906a09bc2fa010092a86b133d0d749c52c is tests-only against a82277af5650b487cea1260c24ffcc1c86d69d8d; exact repair-only authorization audited; original7ec5d609 retained.
- Independently replayed complete scoped RED:189 leaves,47 pass,142 fail,0 skips; starts/terminals201/201. Gates133/31pass/102fail(7.929s); CLI28/11/17(4.390s); hook28/5/23(4.380s); designlock0(0.667s, no-tests warning, NO capability proof). Exactly24 A real false-acceptance failures,8 D compatibility controls,26 B interface/diagnostic failures,92 C-family control-unavailable cases. No v2/late fault execution credited;16 B schema mutations also unreached. No setup/compiler/import/runtime/timeout failure. Coverage percentage NOT MEASURED.
- Independent verify-tdd PASS2commits/0skipped merges, no unauthorized edits. verify-delivery9/9shape only; full source/assertion review done independently. All native names/outcomes match author inventory. Five frozen SHA256: internal/gates/attest_implementation_test.go=fcae6e3a9dc66d8fb9abb1d151129604f59a3e38d9ac221d4f16ce91a8086260; internal/gates/attest_test.go=f0eef53c28fdf940654700832887f2c326a478c5df1d741ae9f538a3d24db253; cmd/machinery/attest_test.go=beb2d58af4d3144316f3789c3b7cbb726a4761af57ea85f0017297b9c24cd6cd; cmd/machinery/attest_implementation_test.go=e02ad18fe2745d23f05e69ce1e01dd2b1be358f94d99cb3a215b14f6e2cee3b8; internal/hook/attestation_snapshot_test.go=a3c088c095e2dba4379f548903a058eedbd5338ef872e507ab12edaa1ca06e49.
- Required GREEN: all frozen bytes unchanged/all tests pass; actual C valid controls and challenge reachability; new designlock held-root/topology/identity/copy/budget/overlay/per-file proof and suite wrapper/renderer supplements via only previously authorized seams/new files. Complete fixture has reached first unknown --design only: later full --complete success/sole-warning isolation must be demonstrated, not assumed. No further frozen or example/golden repair authorization.
- Same final GREEN revision must supply OBSERVED RenderAttestation actual late original mutation/real owned-cleanup failures returning nil bytes with fired/no-fault controls and concrete causes; OBSERVED built CLI full generation success and actual ordinary renderer-input/alias error exit1/empty stdout; REVIEWED exact delivered CLI->renderer->Release/errorjoin->output/defer/exit closure. Late CLI result COMPOSED; independent standalone late injection UNOBSERVED; individual OS Close errors UNFORCED; output-sink partial-write limit separate. All five AC/R2/AC5 and13required/optional14path ownership preserved; forecast2710–3310 is not permission to trim proof.
- Own detached checkout is clean at47ba449; all review test/child/build processes terminated. Main clean and installed binary remains SHA2565205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849. No production/test/docs edits by PM; no developer worktree action, remote, preflight, service/container or installed asset change.

### proof
- [x] RED AC #1: complete-scope independent digest/topology/forgery/mutation bar frozen; actual implementation and new capability supplements owed in GREEN.
- [x] RED AC #2: closed plan/current/history and warning/complete proof contracts reviewed; future v2/history/complete controls and docs owed.
- [x] RED AC #3:24 real existing-interface false-acceptance failures independently observed with8 passing D controls and explicit migration assertions.
- [x] RED AC #4: real lifecycle/filesystem/hook challenge assertions reviewed with exact control/firing/safety requirements; unexecuted mutations explicitly not credited.
- [x] RED AC #5: real baseline CLI A/D observed; full same-GREEN renderer/process/source-closure conjunctive bar preserved. Product acceptance remains pending.

### 2026-09-06T04:31:42Z ramirosalas
GREEN PAUSED — RED-DISPUTE checkpoint; not delivery or acceptance.

RED-DISPUTE 1: cmd/machinery/attest_implementation_test.go TestAttestImplementationCLI/C-complete-sole-current-warning lines412–424 converts acceptance date 2026-09-03 to time.Time via yaml.Unmarshal(map[string]any), then yaml.Marshal rewrites it to 2026-09-03T00:00:00Z. Actual built complete CLI now reaches all13 generation calls and finalized Gv current review, but Ga correctly rejects all6 M0..M5 dates; 6 blocking findings, missing-current challenge UNREACHED. Ga/examples/helpers/frozen files unchanged. Exact repair requires independent PM authorization.

RED-DISPUTE 2: internal/hook/attestation_snapshot_test.go hookReviewFixture scenario.Empty sets Gates="", Impl=""; unchanged progressive selectGatesCheckedInSnapshot unconditionally selects Gl. B-empty-selection and C-empty-cleanup reach callback with one gate and fail len(run)==0 before the intended empty branch/cleanup mutation. Independent PM must choose exact fixture correction; no selector/policy expansion authorized.

PROOF: source390d4dc38be1ad5a16ef3cec171e08d9bf6c0036, branch story/MAC-p7jd, clean retained worktree. Initial47ba449 replay matches189 leaves/47pass/142fail/0skip (24A genuine failures,8D passes,26B missing-interface failures,92C unreached). First WIP e55d534 compilation failure int/int64 preserved; fixed999b9ab. Exact second targeted runs count1/timeout5m/json: gates -run Attest at999b9ab 133leaves/126pass/7fail/0skip,8.984s; CLI -run Attest at390d4dc 28/26/2/0,7.226s; hook -run Attestation at390d4dc 28/26/2/0,4.363s. These are separate revisions, not final combined proof. Native names and all stdout/stderr/source hashes/report are in /tmp/machinery-p7jd-green-proof.eHzGMx/. PAUSED-RED-DISPUTE.md records all failures, reached stages and cost. Coverage NOT MEASURED; no profiles/broader tests/preflight. Baseline designlock0tests is NO capability proof.

Implementation failures retained: gates6 custody cases return duplicate G0 findings and1 addition diagnostic omits descendant; production fixes committed390d4dc but gates replay owed. CLI missing-input reaches real renderer exit1/empty stdout but omits BUILD.md from diagnostic; fix still owed. All26 other CLI leaves pass; all26 other hook leaves pass, including actual original/owned-copy cleanup faults across Stop/SubagentStop strict/relaxed/wave, with callback/sentinel/ledger assertions. Mandatory NEW designlock/wrapper/renderer supplements and docs remain unauthored, all final AC proof owed.

WIP cost:6 production files1125add/43delete=1168lines; combined frozen RED11files2609add/46delete=2655lines. No generic designlock/external/source/inventory/portablepath/scale, Ga, cmd hook, example, golden, installed binary/asset, main/epic, dependency, service/container or remote edits. All5 approved frozen hashes unchanged. No uncommitted source or active test/build/helper process at pause. No delivery/release/acceptance/merge transition. Root acknowledged pause and will route independent review.

LEARNINGS: generic YAML-map round trips can alter exact date lexemes; empty progressive hook config still activates Gl; duplicate lifecycle errors need a consumable finalized wrapper shape. Passing partial cases do not establish the complete contract.

## nd_contract
status: in_progress

### evidence
- Clean WIP390d4dc; two exact frozen-fixture disputes above; full report/raw leaf history retained. No test edits or repair authority assumed.
- Required next step independent PM exact fixture repair decision; mandatory remaining implementation, docs and new supplemental proof remain pending.

### proof
- [ ] AC #1: final held-root/copy/limits/same-revision proof pending
- [ ] AC #2: complete fixture and final docs pending
- [ ] AC #3: reached migration cases require final replay/review
- [ ] AC #4: disputed empty fixture and mandatory supplements pending
- [ ] AC #5: complete-mode and same-revision renderer/CLI/source-closure conjunction pending

### 2026-09-06T04:38:19Z ramirosalas
## nd_contract
status: in_progress

### evidence
- Independent PM GREEN-phase RED-DISPUTE adjudication recorded in Notes against clean 390d4dc and frozen47ba449: exact guarded commit-line preservation repair in C-complete-sole-current-warning, and exact Empty cfg.Gates="g4,gt" setup repair. Separate [test-edit-authorized] commit required; no wider frozen edit or new seam permitted.
- Original Ga date and progressive Gl failures remain fixture setup history; complete missing-current challenge and empty cleanup behavior remain unreached until repaired replay. All semantic/path/cause/output/ledger assertions preserved.
- Existing hard-tdd/red-approved labels, in_progress status and dev-MAC-p7jd claim retained; no new RED approval, delivery, rejection or GREEN acceptance performed. Exact amended hashes/diff, TDD audit and same-revision four-selector replay remain mandatory.
- Separate gates diagnostic fixes require replay; missing BUILD.md production diagnostic fix remains owed. Mandatory new designlock/renderer supplements and docs are not yet supplied.
- Five AC, R2 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179, AC5 composition5fc3193106a803b7760020b646ed4db7b3d417f5e196b2266d10cf8227733937, prior seam restrictions/ownership and current cost forecast unchanged.

### proof
- [ ] AC #1: final held-root/copy/inventory/bounds and same-revision verification pending
- [ ] AC #2: repaired complete-mode control, correct kinds and final docs pending
- [ ] AC #3: reached migration observations require final same-revision replay/review
- [ ] AC #4: repaired empty fixture must reach real cleanup; mandatory supplemental proof and all preserved assertions remain required
- [ ] AC #5: complete-mode sole-current-warning and same-delivered-revision OBSERVED renderer faults/nil bytes plus OBSERVED built CLI outcomes and REVIEWED renderer-to-CLI closure remain mandatory; CLI late guarantee COMPOSED, independent standalone late injection UNOBSERVED, individual OS Close errors UNFORCED


### 2026-09-06T05:05:18Z ramirosalas
RED-DISPUTE: GREEN paused at clean4b246baf7f9384f373d4935f01bbe07c7fcf7076. Full PAUSED-BLAST-DISPUTE.md appended in Notes:229 scoped leaves+45 targeted race leaves PASS/no skips; broader replay4 legacy v1 fixture conflicts and1 existing filesystem skip unresolved. No fixture/helper/example changes, no delivery. Latest nd_contract status: in_progress; proof pending exact repair authority, docs-only positional check command correction and final review. No active/background process.

### 2026-09-06T05:06:12Z ramirosalas
## nd_contract
status: in_progress

### evidence
- Clean paused head51454e069ebe4039f02d6d9108acf9354c7ad6c8; last commit docs-only corrects `machinery check --design design --impl src --gate gv` to `machinery check design --impl src --gate gv`. Docs SHA256d7ca4cc79a2af87e6946dba3bba445b21cee9639da13309ec2384033d0a8ed41. Full source/test implementation remains4b246baf7f9384f373d4935f01bbe07c7fcf7076;229 scoped and45 race leaves passed with0 failures/0 skips on4b246ba, not claimed as a51454e0 replay. Fresh same-SHA proof is owed on eventual delivery.
- Full pause report /tmp/machinery-p7jd-green-proof.eHzGMx/PAUSED-BLAST-DISPUTE.md SHA256d2837c4fbd4205ba151b856e96a7747d3cc32b58b1a69b4c49fe1d6fedb6bc1f preserved and appended in Notes. Raw/native inventories/hash files there bind4 existing fixture conflicts,1 existing filesystem skip, all passing observations and failed command histories. Final scope14 paths3331 additions66 deletions=3397 unchanged.
- RED-DISPUTE remains: internal/hook/hook_test.go TestStopGreenDesignClearsStateSilently uses examples/go-crm/design; internal/gates/obligation_ownership_test.go TestObligationParentRealCLI/{obligation-free-parent-control,Policy.oracle.md,Isolation.oracle.md} uses examples/checkout-split/parent/design. Their unchanged v1 gt.conformance-test-shape rows correctly fail GV_MISSING_IMPLEMENTATION_SUBJECT. No grandfathering/fixture/helper/example/Ga/selector change authorized or made. Root routes migration ownership independently.
- Broad native replay: gates33PASS0FAIL1SKIP (existing case_folded_collision on case-insensitive filesystem); hook23PASS1FAIL0SKIP; accepted-stories76PASS3FAIL0SKIP; designlock13PASS0FAIL0SKIP. No skipped case credited. No active/background process, no delivery/acceptance/merge/install/remote/preflight operation.

### proof
- [ ] AC #1: scoped v2/current/plan/history observations pass; independent overall review pending.
- [ ] AC #2: full inventory/digest/kinds/docs observations supplied; final same-SHA proof pending.
- [ ] AC #3: required scope migration passes;4 broader legacy fixture conflicts need exact ownership/repair authority.
- [ ] AC #4: real capability/limits/hook and45 race observations pass; broader fixture/platform-skip resolution pending.
- [ ] AC #5: observed Render late faults return nil bytes; built CLI ordinary success/failure observed; composed source closure recorded; independent review and final same-SHA replay pending. Standalone CLI late injection UNOBSERVED, individual OS Close errors UNFORCED, sink partialwrites separate.

### 2026-09-06T05:26:12Z ramirosalas
## nd_contract
status: in_progress

### evidence
- SR PM LEGACY FIXTURE OWNERSHIP / PROPOSAL-PREPARATION HOLD. Read complete PAUSED-BLAST-DISPUTE.md SHA256d2837c4fbd4205ba151b856e96a7747d3cc32b58b1a69b4c49fe1d6fedb6bc1f and terminal report.229scoped+45race PASS belong to4b246ba; paused51454e069ebe4039f02d6d9108acf9354c7ad6c8 is docs-only later and still owes finalsameSHA proof. Four old-fixture leaves remain held: hook_test.go TestStopGreenDesignClearsStateSilently and obligation_ownership_test.go TestObligationParentRealCLI obligation-free/Policy/Isolation default-gate legs. Legacy v1 gt correctly fails GV_MISSING_IMPLEMENTATION_SUBJECT; silence/ledger, default-vs-explicit Gt, real controls/negatives, Ga ancestry/selection and no-grandfathering are fixed. hgz1 now owns all8bundled evidence migrations AFTER p7/uzxr/lhu5; no p7 reverse dependency or bundled writes to unblock this story. Root may resume the healthy retained GREEN author for an UNAPPLIED EXTERNAL exact test-local fixture proposal against51454e0 limited to those2existingtest paths, with full old/new hunks, original-assertion byte equality, helper callers, real input/evidence inventory, warning/silence semantics and per-file cost. This is proposal preparation only, not oldtest editing or TEST-EDIT AUTHORIZED. Separate independent PM must approve exacttext before any subsequent sanctioned amendment. Verified existing APIs: gates.AttestationReview{Claim,Kind,Attestor,Date,Note string}; RenderAttestation(design,impl string,review AttestationReview)([]byte,error); CheckAttestationsWithImplementation(design,impl string)*Gate. No missing core schema/API identified. Exact valid fixture construction remains unresolved: plan warnings cannot be assumed silent; real/synthetic impl hashes do not establish substantive current conformance, especially before uzxr repair. If no fixture preserves all constraints, report exact technical conflict for independent specialist review; do not weaken assertions, invent review or change product semantics. Existing filesystem skip remains uncredited. Scoped structural verification after exact canonical repair: pvg lint --backlog --epic MAC-ui8a scanned37 issues,0 errors/0 review findings; pvg rtm check37 stories/19closed/0 extracted requirements; pvg nd dep cycles found none. These are structure checks, not AC proof. Claims/status/labels/dependencies retained; no source/test/example/toolchain/remote/preflight mutation or Paivot product dependency.

### proof
- [ ] AC #1: Scoped v2/current/plan/history observations supplied; independent final review pending.
- [ ] AC #2: Full-root inventory/digest/docs observations supplied; final sameSHA proof pending.
- [ ] AC #3: No-grandfathering retained; four broader fixture leaves require exact independently reviewed repair.
- [ ] AC #4: Capability/race observations supplied; broader fixture/platform-skip resolution pending.
- [ ] AC #5: Render/CLI observations and composed source closure preserved; independent review/final replay pending.

### 2026-09-06T05:37:21Z ramirosalas
## nd_contract
status: in_progress

### evidence
- PROPOSAL ONLY, UNAPPLIED, NOT TEST-EDIT AUTHORIZATION: exact test-local fixture amendment against clean51454e069ebe4039f02d6d9108acf9354c7ad6c8. Shared source/tests/examples unchanged; no delivery or RED approval. Independent PM review required before any [test-edit-authorized] commit.
- Index /tmp/machinery-p7jd-fixture-proposal.geNVdA/PROPOSAL-INDEX.md SHA2561049ea7adf1eaa142948f2f6cd3d4d8327ae20a9b3326222f57041f7f19c9169. Exact fixture-amendment.patch SHA256ae6e9241ea5d7b0c4556b2ee02c6e2910b2663c2452761e7984f1c6663252971,65 additions0 deletions in only internal/hook/hook_test.go copyTree and internal/gates/obligation_ownership_test.go obligationParentFixture. Proposed total16 paths3462 changed lines, not approved scope.
- External archive diagnostic only: all13 original affected-helper leaves PASS0FAIL0SKIP (6hook4.704s,7parent9.320s). Exact original assertions/outside-helper bytes unchanged; zero deleted original lines. Raw Gv explicitly retains missing-current warning and current_reviews0. Actual Stop stdout empty/ledger cleared. Actual parent default+explicit positive controls exit0 and removed-ID challenges exit1 with original assertions. These are NONQUALIFYING external observations, not same-SHA shared story proof.
- Full before/after evidence YAML, all6 acceptance-byte hashes, full62-file GoCRM design inventory (implementation absent),73-file complete parent control and75-file Policy/Isolation control/missing-ID inventories, raw CLI/Stop outputs, exact native names, audit scripts and provenance in that directory. Only copied evidence changes; cover hashes/attestor/date unchanged; ga historical and all acceptance anchors unchanged. No GoCRM runtime/parser or parent decision-ID synthetic code is claimed as current substantive conformance.
- Precise forward consequence: hgz1 planned source v2 migration WILL fail both exactv1 fixture guards. Its later GoCRM current row is not this fixture's current evidence. PM/Sr PM must explicitly route future exact amendment/decoupling authority for these2 constructors before hgz1 regression completion; its present scope does not own these files. No automatic broadening or reverse dependency proposed. This65-line proposal is intentionally unchanged pending judgment.
- Earlier4 broader failures,1 filesystem skip,229 scoped+45race passes at4b246ba remain preserved and unresolved shared-state evidence. Clean shared51454e0; all finite diagnostics complete; no background process, install, asset/service operation, remote, preflight, merge or production dependency added.

### proof
- [ ] AC #1: prior observations retained; final independent review and same-SHA replay pending.
- [ ] AC #2: exact proposal preserves honest plan/current/history distinction; no shared repair authorized yet.
- [ ] AC #3: four existing fixture conflicts have an externally demonstrated construction; PM must adjudicate exact patch and forward hgz1 boundary.
- [ ] AC #4: prior held-root/hook/race proof retained; existing unrelated platform skip remains unresolved.
- [ ] AC #5: prior composed renderer/CLI observations retained; final same-SHA verification and independent source closure remain mandatory.

### 2026-09-06T05:46:06Z ramirosalas
## nd_contract
status: in_progress

### evidence
- Sr PM source-supported CONDITIONAL scope/cost update, coordinated with sole independent PM who confirmed technical replay but held exact application until serialization. Current51454e0 remains14paths/3397changed. Exact UNAPPLIED patch ae6e9241ea5d7b0c4556b2ee02c6e2910b2663c2452761e7984f1c6663252971 adds65/0 solely in copyTree36 and obligationParentFixture29; conditional total16paths/3462changed. Two exact conditional PRODUCES now match later hgz1 CONSUMES. Five AC unchanged. Separate explicit PM TEST-EDIT AUTHORIZED disposition is still required; no authoring/application/delivery/approval by SrPM. Accepted initial construction/full13-caller blast/input/acceptance evidence passes forward to hgz1, never a reverse dependency.
- Proven guarded pvg nd edit used exclusive-lock editor and apply_patch only, fullraw expected header/body + external exacttext trial + exactpostreadback; all prior Notes/History/Links/Comments and metadata retained except normal hash/time. Original story AC compared byte-exact. Source/proposal read complete; no product/test/example mutation or Paivot runtime/build/test dependency.
- Scoped lint37issues/0errors/0review; cyclesnone; globalRTM37stories19closed0extracted requirements. Structural results only, not epic completion or AC proof. Claims/status/labels/dependencies unchanged. External preservation manifests /tmp/machinery-private-fixture-serialization.clhWCS.

### proof
- [ ] AC #1: Scoped v2/current/plan/history observations supplied; final review/replay pending.
- [ ] AC #2: Full scope/hash inventory observations supplied; final same-SHA proof pending.
- [ ] AC #3: No-grandfathering remains fixed; exact conditional two-helper amendment awaits separate PM disposition.
- [ ] AC #4: Real capability/race history retained;13-caller proof and all broader obligations require final candidate review.
- [ ] AC #5: CLI/Render/history distinction observations retained; no delivery or acceptance granted.

### 2026-09-06T05:50:12Z ramirosalas
## nd_contract
status: in_progress

### evidence
- Independent PM EXACT LEGACY-FIXTURE AMENDMENT ADJUDICATION authorizes only patch ae6e9241ea5d7b0c4556b2ee02c6e2910b2663c2452761e7984f1c6663252971 against51454e069ebe4039f02d6d9108acf9354c7ad6c8:36 additions in copyTree and29 in obligationParentFixture, no deletions/other-file authority. Separate [test-edit-authorized] commit and all frozen hashes required. Original assertions/Ga/source examples/no-grandfathering retained.
- Independent external archive replay13 caller leaves PASS0FAIL0SKIP; four external observer leaves reproduce exact real Gv/Stop/CLI and full inventories. Audit /tmp/machinery-p7jd-pm-fixture.Mm3FM4/pm-audit.json SHA25643773cab089ffd3ec3cc48411c32c5b39c0a3f277632f6e09e377555f2c091f0. External evidence is diagnostic only, not delivered same-SHA proof.
- Independently read-back SrPM p7 conditional16paths/3462changed and hgz1 future41paths with exact two-helper accepted-baseline CONSUMES resolves earlier missing-owner hold. Future v2/current fixture handling still requires separate exact BEFORE-EDIT PM approval; no automatic downgrade, provenance renewal or reverse dependency. Root verifies durable authorization before application.
- Status in_progress, hard-tdd/red-approved labels and dev-MAC-p7jd claim preserved. No delivery/accept/reject/approve-red transition, product source/test write, developer worktree access, install/remote/main/epic/preflight mutation. All original failed/setup/skip history retained.

### proof
- [ ] AC #1: Complete implementation scope binding still requires final same-revision proof and independent acceptance review.
- [ ] AC #2: Plan/current/history distinction retained by exact fixture amendment; whole-story review remains pending.
- [ ] AC #3: Legacy no-grandfathering remains enforced; exact two-helper repair authorized but shared application/replay not certified here.
- [ ] AC #4: Frozen tests and all13 affected callers preserved; fresh scoped/race/broader regression and TDD audit owed.
- [ ] AC #5: Real CLI/history and composed renderer closure require final same-SHA review; no execution/reviewer-honesty or separately injected CLI/OS-close proof claimed.


### 2026-09-06T06:04:04Z ramirosalas
## nd_contract
status: delivered

### evidence
- GREEN developer handoff for independent PM review, not acceptance. Clean committed candidate 412d01b57ae4770a6cf4fb4f48442fb7e632cdd7 on story/MAC-p7jd; production base a82277af5650b487cea1260c24ffcc1c86d69d8d. All finite verification processes completed.
- PROOF and LEARNINGS: /tmp/machinery-p7jd-final-proof.O4NxfV/GREEN-REPORT.md, SHA256 815e8dd3c79f0b28dba51b8f7354b9cb9fe29f862327839de14d3163645bf033. Raw evidence manifest /tmp/machinery-p7jd-final-proof.O4NxfV/evidence-sha256.txt, SHA256 8c6749f50855d9ec76d7a75a0f6975b615933783d8cd29020a481bbb137599e1. Full report appended to canonical Notes.
- Fresh same-SHA native proof: four scoped package selectors 229 PASS / 0 FAIL / 0 SKIP; targeted race 45 PASS / 0 FAIL / 0 SKIP; all 13 amended-helper callers PASS / 0 FAIL / 0 SKIP. Broader selectors 149 PASS / 0 FAIL / 1 existing filesystem SKIP. Overlapping scopes are not summed as unique coverage. Complete names, commands, timings, raw stdout/stderr and hashes are in the report/index.
- Exact PM-authorized fixture patch ae6e9241ea5d7b0c4556b2ee02c6e2910b2663c2452761e7984f1c6663252971 applied in separate [test-edit-authorized] commit: 65 additions / 0 deletions across only internal/hook/hook_test.go and internal/gates/obligation_ownership_test.go. Both before/after hashes recorded; all seven frozen test hashes preserved. Final canonical scope 16 paths, 3396 additions + 66 deletions = 3462 changed lines.
- pvg story verify-tdd --base a82277af5650b487cea1260c24ffcc1c86d69d8d --json: 11 commits, 0 merges, 0 violations. git diff --check and gofmt on exact changed Go paths: empty. Isolated standalone go build succeeded; no dependency or installed-tool changes.
- Scoped pvg verify --format text --include-tests is NOT clean: 11 pre-existing hook.go heuristic findings. Full containing functions mechanically byte-identical to production base, with signatures/hashes in quality-provenance.json. No unrelated source changed to suppress findings.
- Existing TestSelectRejectsNonportableAndAliasedDesignPaths/case_folded_collision skip is uncredited; case-sensitive Linux CI/final epic gate remains owed. No skip bypass or full preflight performed. Bundled v1 example migration and the two explicitly assigned future helper adaptations belong to hgz1 under fresh exact PM authority; no future downgrade is preauthorized.
- AC5 actual renderer late mutation/cleanup and actual standalone CLI ordinary input/error/success outcomes are OBSERVED on the same SHA; renderer-to-Release/error-join-to-CLI-output closure is COMPOSED for independent PM source review. Standalone injected late CLI failure is UNOBSERVED, individual OS Close primitives UNFORCED, sink partial writes separate. No synthetic production fault API.
- pvg story deliver MAC-p7jd succeeded using the shared live nd vault. Developer has not accepted, closed, merged, or changed main/epic. Independent PM must review the full R2 contract, AC5 composition, and disclosed limits before acceptance.

### proof
- [x] AC #1: Complete implementation-root manifest and retained-root custody bind added, removed, renamed, changed and narrowed implementation/test scope; frozen topology/digest/scope cases and new capability/limit controls pass on candidate 412d01b.
- [x] AC #2: Closed v2 plan/current/historical distinctions are implemented in schema, diagnostics, CLI and documentation; actual kind/schema, historical and complete-warning controls pass.
- [x] AC #3: Legacy behavioral evidence is never grandfathered as current; migration diagnostics and all 13 exact amended fixture callers preserve plan warnings/current zero and historical Ga semantics.
- [x] AC #4: Required real positive/negative filesystem, code/test mutation, alias, stale replay and evidence-only cases pass without skips; the distinct pre-existing case-insensitive-filesystem skip remains uncredited and owed on Linux.
- [x] AC #5: Same-SHA actual CLI and renderer observations plus explicit source composition support the approved guarantee and honest binding limits; independent PM final source review remains required, with UNOBSERVED/UNFORCED boundaries retained.

### 2026-09-06T06:30:50Z ramirosalas
## PM Decision
REJECTED [2026-09-06]: whole-story GREEN candidate 412d01b57ae4770a6cf4fb4f48442fb7e632cdd7.
EXPECTED: AC1 requires a complete implementation/test scope and invalidation after code/test/config change or scope narrowing; AC4 requires meaningful negative and matched positive proof. R2 permits only the exact attestation record and top-level .git exclusions.
DELIVERED: internal/gates/attest.go:874 initializes exclusion to the digest sentinel "none"; lines 882-885 use it as a real entry filter. Independently built ordinary CLI omits actual top-level none file/empty directory in disjoint and implementation-inside-design roots. Four real file-content/directory-mode mutants remain current 1 with exit 0 and stdout identical to the unchanged controls. Equal-root and implementation-ancestor file controls correctly stale and exit 1.
GAP: unauthorized third scope exclusion. Same-generation custody and existing 229 scoped / 45 race / 13 caller PASS do not establish freshness for an input omitted from both generation and checking. Existing test history is preserved, not invalidated.
FIX: root/Sr PM route exact same-story new regression ownership first. Add independently reviewed positive+negative RED against 412d01b for real none file/empty directory, both affected topologies, exact inventory and change invalidation; retain actual evidence/.git exclusions and other topology controls. Only then repair production, freeze and rerun same-final-SHA scoped/custody/CLI/hook/caller/relevant broader/race and TDD proof. No production-first repair, new seam, invented test filename authority or existing frozen test amendment is granted.
Full review appended to Notes: /tmp/machinery-p7jd-pm-final.pT2Q0q/PM-FINAL-REVIEW.md SHA256 c3950515759f8cf148b4e2469102bbe1a7b7dabfc4c9b22bdb14297ec07071d6.

## nd_contract
status: rejected

### evidence
- Independent exact Git archive review of 412d01b57ae4770a6cf4fb4f48442fb7e632cdd7 against a82277af5650b487cea1260c24ffcc1c86d69d8d; full R2/AC5 and changed source/tests/docs plus relevant provider/caller closure reviewed. No production/test/developer-worktree or installed artifact mutation.
- Same-SHA independent native proof: 229 scoped PASS, 45 race PASS, all 13 helper callers PASS; broader selectors 149 PASS and 1 unchanged casefold SKIP, all zero FAIL. Overlaps not summed. Seven frozen hashes and exact authorized 65-addition patch retained. TDD 11 commits / 0 violations. Final 16 paths / 3462 changed lines.
- Own audit /tmp/machinery-p7jd-pm-final.pT2Q0q/pm-review-audit.json SHA256 d5ef7a3f84b733c7293d35f68761cd4e90c3d49894ca727b92f8affd4799d8a2 preserves exact raw names/hashes/counts/source/lineage/quality. Original frozen RED and SETUP histories remain unchanged.
- Actual defect repeat /tmp/machinery-p7jd-pm-final.pT2Q0q/pm-none-replay.json SHA256 0fd1686fc37b983c35d569d3ece449056d7f37ea3b25d589b4036a5b87d1d78a records six exact cwd/argv/status/input/mode/receipt/stdout cases. Built binary SHA256 534f53900fc9457d0950bcc6ed1d72aeac1d397d030a8017532cc7911e53f0ad.
- AC5 OBSERVED real renderer late original/owned-cleanup faults -> nil bytes/actual causes and real CLI ordinary error -> exit1/empty output; REVIEWED exact Render -> final Release/error join -> first output closure; COMPOSED late CLI guarantee. Standalone injected late CLI UNOBSERVED, individual OS Close UNFORCED, sink partial writes separate; no execution/reviewer authentication claim.
- Quality scan remains FAILED with 11 existing heuristic findings in four byte-identical base hook functions. Case-sensitive Linux case_folded_collision leaf remains owed at final epic gate, not passed/waived; required attestation/caller scopes have zero skips. No full preflight/runtime/install/service/remote operation.
- Root must route same-story RED-first repair and exact scope authority before authoring. No blanket test amendment or new path/seam authorized. hgz1 future two-helper fresh actual-baseline amendment hold remains exact. All finite PM processes complete; no acceptance or merge.

### proof
- [ ] AC #1: BLOCKED: actual implementation path none is omitted in two supported topologies and its change is falsely current.
- [x] AC #2: Closed kind/schema/current/history distinctions independently source-reviewed and exercised on covered fixtures.
- [x] AC #3: Legacy no-grandfathering/migration and exact 13 authorized helper callers independently verified; future hgz1 migration remains separately owned.
- [ ] AC #4: BLOCKED: existing negative/positive proof misses the demonstrated real-file/empty-directory sentinel collision; new exact regression-first proof and repair required.
- [x] AC #5: Real current/history/changed-code CLI observations and approved bounded error/output composition independently verified, with limitations above; this is not whole-story acceptance or cure for AC1.


### 2026-09-06T06:44:43Z ramirosalas
## AUTHORITATIVE SAME-STORY SCOPE AMENDMENT — rejected412d01b literal-none regression

This append-only live amendment supersedes ONLY earlier16-path ownership and cost forecasts for the confirmed same-story AC1/4 rejection repair. It adds exact seventeenth path cmd/machinery/attest_scope_none_test.go to PRODUCES for TestAttestationScopeNone. All five original AC, full R2, seven frozen tests, exact two authorized fixture-helper constructions, status/labels/claim/dependencies and historical evidence remain unchanged. Prior description text is retained history; this is the current scope declaration, not RED approval or production/test application authorization. Root must route fresh RED after this scope checkpoint, then independent exact RED review before separate GREEN. No new bug/epic or production seam/API.

## DIFF BUDGET
- Rejected412d01b57ae4770a6cf4fb4f48442fb7e632cdd7 is measured16 paths,3396 additions+66 deletions=3462 aggregate changedLOC againsta82277af5650b487cea1260c24ffcc1c86d69d8d. The exact independently authorized65-addition fixture patch is included, not still unapplied. The bounded literal-none rejection repair adds ONE new regression file, making17 paths; existing production repair stays within internal/gates/attest.go.
- Conditional forecast: approximately300–500 added regression LOC for the four-topology/two-entry-type independent CLI matrix and5–20 changed production LOC for the already-owned projection correction: approximately3770–4000 aggregate changedLOC. This is a reasoned forecast, not a cap or permission to trim proof; actual diff, cumulative rework cost and leaf runtimes must be reported separately. If the complete required matrix materially exceeds this estimate, explain the measured reason and escalate any new path/API/seam before editing it. Earlier2710–3310 and13/14/conditional16-path budgets below are historical only.

## Same-story rejection repair: literal none scope entry
Authority and exact cause: independent PM REJECTED candidate412d01b57ae4770a6cf4fb4f48442fb7e632cdd7 for AC1/4; PM-FINAL-REVIEW SHA256 c3950515759f8cf148b4e2469102bbe1a7b7dabfc4c9b22bdb14297ec07071d6. captureAttestationSubject initializes the digest exclusion descriptor to "none" and also compares every actual entry path against that descriptor unconditionally. When the logical design evidence is outside the implementation root, a legitimate top-level file or empty directory named none disappears from the projected manifest. The real snapshot includes it. Generation and later checking share the omission, so separate-invocation changes can still report current. PM observed file-content and empty-directory-mode mutations in disjoint and implementation-inside-design layouts falsely returning exit0/current1; equal-root and implementation-ancestor file controls correctly returned GV_STALE_CONTENT. This is one same-story blocker, not a new bug/story or change to the five AC/R2 digest grammar.

New exact ownership is ONLY cmd/machinery/attest_scope_none_test.go, with TestAttestationScopeNone and locally named subtests/helpers confined to that file. Existing built-CLI helpers are read-only dependencies verified at exact412d01b: cmd/machinery/golden_test.go supplies goldenBin(t *testing.T) string; cmd/machinery/attest_implementation_test.go supplies cliReviewExec(t *testing.T, binary string, f *cliReviewFixture, args ...string) cliReviewResult, cliReviewWrite(t *testing.T, path, body string), cliReviewRead(t *testing.T, path string) []byte, and cliReviewCheck(t *testing.T, binary string, f *cliReviewFixture, extra ...string) cliReviewResult. cliReviewFixture has root/design/impl strings; cliReviewResult has out/stderr strings and code int. Reuse those real-process/temp-filesystem conventions without modifying their implementations; use isolated fixture/configuration and ordinary locally built binary, never installed assets or mocked gate/renderer verdicts. The new file is absent at412d01b and may be authored only through the root-routed fresh RED task, not by this scope repair.

Required regression matrix (all independently observable, no skip-if-feature-missing):
1. Four real layouts: disjoint design/implementation roots; implementation inside design; equal roots; implementation ancestor of design. In EACH layout test separately a valid top-level regular file named none and an empty directory named none. Independently enumerate the expected full rooted receipt inventory, including none and its exact path/type/mode and file size/content hash where applicable; do not take the generator or checker output as the expected-inventory oracle. Verify generated receipt contents and the exact unchanged CLI current-review count1/exit0 control. Ordinary fixture warnings must not masquerade as the intended failure.
2. In fresh isolated cases for all eight layout/type cells, establish the unchanged receipt/control before one real mutation: file content change or empty-directory mode0755-to0700 change must return exit1/GV_STALE_CONTENT naming none, with no current success. Separately challenge real addition, removal and rename of the same literal-none file/directory: require exit1/GV_SCOPE_INVENTORY and the particular added/removed logical path, not an arbitrary error. Empty-directory cases must remain empty; file cases must be actual regular files. Keep receipt-inspection assertions in independent cases so an omitted inventory entry cannot prevent the unchanged/mutated CLI legs from executing and being recorded.
3. Exercise submitted-scope omission of none with unchanged original digest (GV_SCOPE_HASH) and with independently recomputed valid narrowed digest (GV_SCOPE_INVENTORY); retain the existing root-narrowing controls and no partial-root success. These are separate receipt edits, not renewed reviews. Preserve the exact R2 encoding: the descriptor string none still means no logical evidence exclusion when the design evidence is outside the implementation root, never a reserved real basename. The repair cannot reject that valid filename/directory, change digest/schema grammar, broaden exclusions, grandfather old behavior or silently narrow the claim.
4. Across the topology fixtures preserve exactly the existing exclusion contract: only the logical design/attestations.yaml path, when within implementation, is excluded even if absent; unrelated sibling/nested attestations.yaml remains inventoried and changes invalidate. Preserve top-level .git ordinary-directory/regular-gitfile metadata exclusion, nested .git rejection and all other rooted inventory/custody restrictions. Retain explicit evidence-only freshness controls and existing frozen exclusion/alias/custody proof; no blanket none, evidence-name, hidden-path or metadata filtering is authorized.

Serialization and freeze: root routes a fresh RED author against exact rejected412d01b, confined to the new file. Run the complete new leaf inventory using the existing native built-CLI harness; record passing unchanged/equal/ancestor controls and exact behavioral assertion failures for omitted entries and falsely current changed subjects, not compilation/import/timeout failures. Record every reached mutation independently; never claim that an early inventory assertion exercised a later challenge. Independent PM must replay/review the exact new test text and candidate before any production correction. Freeze the new regression bytes/hash after that review, retaining the seven existing frozen test hashes and the two exact65-line authorized fixture-helper constructions unchanged. This scope entry does NOT approve RED, authorize existing-test edits, authorize production application, change labels/claims or grant any new seam/API. Only a separately routed GREEN implementer may then correct the already-owned internal/gates/attest.go projection under that reviewed freeze; no prescribed code patch is implied.

Completion evidence remains conjunctive on one final candidate: new full regression matrix, original scoped attestation/designlock/hook tests, all13 fixture-helper callers, broader caller blast coverage, targeted race proof, frozen hash comparison and RED-before-GREEN lineage, followed by independent PM re-review. Preserve the original Ga ancestry, default-versus-explicit Gt, silence/state/ledger assertions and approved AC5 OBSERVED/REVIEWED/COMPOSED/UNOBSERVED limits. Existing case-sensitive Linux final-epic obligation and known quality findings are not waived. MAC-hgz1 still consumes the accepted p7 helper construction/full blast proof and owns its separately exact-before-edit reviewed future two-helper adaptation; no future v2 construction/current downgrade or reverse dependency is introduced. No hgz1/lnu6 scope or criteria change is necessary here.

## nd_contract
status: rejected

### evidence
- SrPM bounded source-confirmed same-story scope amendment; PM rejection report /tmp/machinery-p7jd-pm-final.pT2Q0q/PM-FINAL-REVIEW.md SHA256 c3950515759f8cf148b4e2469102bbe1a7b7dabfc4c9b22bdb14297ec07071d6 read in full; exact412d01b source/APIs and canonical R2 reviewed. Complete scope and authorization hold above supersede only prior path/cost declarations.
- No source/test/example implementation, existing-test permission, phase/claim/status/label/dependency change, acceptance, remote operation or installed-assets mutation. Paivot/nd remain private coordination only, never Machinery build/runtime/test dependency. hgz1 future41-path two-helper hold and lnu6 six-example scope unchanged.
- External /tmp/machinery-p7jd-none-scope.Rmw8XS retains three stopped local preparations, original expected raw snapshot and exact proposal. No nd edit was invoked: apply_patch external update lost one final LF even with prefix-only patch. Root authorized this supported append-only comments route instead; old body must remain an exact prefix and metadata unchanged except normal hash/time. Failed trials are nonproof, not normalized history. Structural checks remain separately owed after readback and are not completion proof.

### proof
- [ ] AC #1: Pending literal-none inventory/freshness correction and same-candidate independent review; rejected412d01b remains blocked.
- [x] AC #2: Prior independent positive closed-kind/schema/current/history review retained; no new execution claimed.
- [x] AC #3: Prior independent no-grandfathering/migration and exact13-helper-caller proof retained; future hgz1 hold unchanged.
- [ ] AC #4: Pending newly owned real four-topology RED-first regression, independent review, frozen GREEN replay and full original proof.
- [x] AC #5: Prior bounded observed/reviewed/composed CLI/error/output evidence and explicit unobserved limitations retained; not whole-story acceptance or cure for AC1.

### 2026-09-06T07:01:43Z ramirosalas
## Supplemental RED delivery held for independent review

The one complete Implementation Evidence/PROOF/LEARNINGS report is appended in Notes and retained at /tmp/machinery-p7jd-none-red.kVs2en/RED-REPORT.md. No historical report is duplicated here. Supported pvg story deliver succeeded, retaining the existing historical red-approved label and adding delivered. Its generated contract was inserted before old comments: first verify-delivery was 8/9 because the last old contract still read rejected. This supported EOF comment records the actual supplemental RED terminal state; no manual label or phase changes are made.

## nd_contract
status: delivered

### evidence
- Supplemental RED ONLY, commit c71faeab997d176224ed6f38437de4604cda0b98 on story/MAC-p7jd; new cmd/machinery/attest_scope_none_test.go SHA256 3d9d8250f759807795251685e1cb924ba8eceb4c819b3fa0376bd15b254f9fc0. Production is unchanged from rejected 412d01b57ae4770a6cf4fb4f48442fb7e632cdd7. One new file, 364 additions; aggregate 17 paths / 3826 changed lines.
- Final exact-source native new matrix 80 PASS / 40 expected behavioral FAIL / 0 SKIP, 37.147 seconds; all 120 leaves and all 72 negative challenges reached. Broader CLI selector 108 PASS / 40 expected FAIL / 0 SKIP, including the same 120 leaves plus 28 existing passing CLI leaves. No compile/setup/timeout/missing-input failure. Complete raw names/results, actual CLI output, generated/submitted rows, original input metadata/hashes, commands and source inventory are in /tmp/machinery-p7jd-none-red.kVs2en/audit.json, SHA256 d5ed8b75aac5187a15ddcb9b96b713144007f23f6b0fcaf1f9a3914f3dcb197d.
- Failure attribution: 16 inventory mismatches, 4 independently complete manifests rejected, 16 unsafe exit0/current1 mutation or narrowing results, 4 rename diagnostic/path-set failures. Rename did reject; it omitted the removed none path and reported only none-renamed. All passing exclusion/freshness legs are recorded even where a subsequent independent inventory assertion fails.
- Seven existing frozen hashes and both exact authorized fixture helper constructions unchanged. pvg verify new file: 1 file / 0 issues. TDD lineage: 12 commits / 0 violations. No production fix, existing-test edit, approval, acceptance, close, merge, remote operation, installed-asset replacement, service/container mutation or preflight. Main and epic remain untouched; retained story worktree is clean and healthy.
- RED-HELD: independent supplemental exact-test-text review/replay is required before separately routed GREEN. Existing red-approved label is historical for the earlier suite and does NOT approve this new file. The product AC1/4 rejection remains unresolved; delivered here means only a committed, reviewable RED candidate. Prior R2, AC5 bounded composition and final case-sensitive Linux/preflight obligations remain unchanged.

### proof
- [x] RED AC #1: Four topologies x regular none file/empty none directory; independent exact inventory/digest and separate actual mutation/narrowing challenges authored and reached. Product repair pending.
- [x] RED AC #2: Existing kind/history boundaries and previous positive proof retained; no schema or classification changes. Additional existing CLI controls pass.
- [x] RED AC #3: Existing migration/no-grandfathering and exact fixture helper bytes retained; no evidence renewal or weakened assertion.
- [x] RED AC #4: All 72 intended negatives reached, matched actual unchanged controls, real evidence-only commits, exact evidence/.git exclusions and same-basename sibling inputs. Forty intentional RED failures remain.
- [x] RED AC #5: Real ordinary locally built CLI and filesystem/Git, precise exit/count/category/path and limits; previous OBSERVED/REVIEWED/COMPOSED/UNOBSERVED limits preserved.
- [ ] Supplemental RED approval: independent PM exact-text replay/review pending.
- [ ] GREEN completion and whole-story acceptance: not delivered; separately routed production correction and all required final proof pending.

### 2026-09-06T07:12:52Z ramirosalas
# MAC-p7jd independent supplemental RED review

SUPPLEMENTAL RED APPROVED [2026-09-06]. Exact test-text approval only. The product remains rejected for AC1/4 until separately routed GREEN and independent final review; this is never whole-story acceptance or closure.

Candidate c71faeab997d176224ed6f38437de4604cda0b98 adds only cmd/machinery/attest_scope_none_test.go, 364 lines, SHA256 3d9d8250f759807795251685e1cb924ba8eceb4c819b3fa0376bd15b254f9fc0. These exact new regression bytes are now frozen alongside the seven previously frozen files and both exact authorized fixture helpers. No existing-test amendment, production code text, new seam/API, schema/digest change or extra exclusion is approved here. Root owns the separately routed GREEN task within the current seventeen-path authority.

## Authority and review boundary

Read canonical MAC-p7jd through pvg nd show --json: status in_progress, delivered, final contract explicitly supplemental RED held; incoming Body SHA256 637552bee833225207fc94724b006550575eb686cb817d5ce52d8346f31ffee1. Read all five original AC and the complete authoritative 2026-09-06T06:44:43Z scope amendment. Read in full and independently hash-verified: R2 PROPOSAL.md 8e20b8d4e1707a2b381f9e8e4f359ca641222f0deec896407f2cdbefe5c0f179; AC5-COMPOSITIONAL-CLARIFICATION.md 5fc3193106a803b7760020b646ed4db7b3d417f5e196b2266d10cf8227733937; prior whole-story PM-FINAL-REVIEW.md c3950515759f8cf148b4e2469102bbe1a7b7dabfc4c9b22bdb14297ec07071d6; source-scope FINAL.md 70569f479256af07d65d3a48c29211d895d26a49b9e0ee0692fda7bde95e5ae8; author RED-REPORT.md b75488049a3dff0f681a8e2e815c2e01a5c5f97f831f21c2a4a509e0f7bbc996 and FINAL.md d1a49c6de6016035af471bf87f768ecd3faca1bf01371871d6b6d51908db58eb.

PM used an external git archive of the exact candidate in /tmp/machinery-p7jd-none-pm.bUihT1. No developer-worktree internals were inspected. All seventeen changed archived paths were independently compared byte-for-byte to exact Git blobs. The only difference from rejected412d01b57ae4770a6cf4fb4f48442fb7e632cdd7 is the one new test file. Production, docs, all earlier tests and helper files are unchanged.

Graph first: project Users-ramirosalas-workspace-machinery, ready generation2026-09-06T02:42:16Z; function search returned goldenBin lines37-66 without additional pages. Coverage checked once for nine task paths. New story paths are missing in this main index; metadata_match of old source is not exact-candidate coverage. Two provisional coverage names main_test.go/testgit/git.go were corrected by exact archive search to testmain_test.go/testgit/testgit.go. All material source claims use exact candidate fallback, never index completeness. Vault search returned no matching new context.

Read the entire364-line test, actual readonly cliReview helpers, goldenBin, TestMain isolation/cleanup and hermetic bounded testgit implementation; complete464-line attestation capability and the relevant manifest/checker/render/suite/CLI closure. The strict held-root traversal includes the real none entry. captureAttestationSubject lines874-885 projects it away by comparing the path against a digest absence descriptor. Both real generation and check use that projection. compareAttestationManifest checks submitted self-hash, both path sets, then content; strict snapshots and Release retain their independent custody checks. This explains the observed separate-invocation freshness defect without inventing a different cause. Prior wider source/custody/AC5 review at412d01b remains retained evidence, not newly replayed here.

## Test quality decision

If all new tests pass unchanged, together with the frozen previous bar they prove the literal-none AC1/4 gap is closed while preserving the intended exclusions for these four topologies and two input types. This is bounded regression proof, not exhaustive whole-program assurance.

The oracle walks original files using filepath.WalkDir, DirEntry.Info, os.ReadFile and SHA256; it does not invoke production inventory/projection/digest helpers. It independently builds every directory/file key and value, exact root locator, full-root-v1 policy and R2 digest. JSON comparison normalizes YAML numeric widths without discarding keys or values. Only actual top-level .git and exact logical design/attestations.yaml are filtered by independently resolved paths. The absent-evidence descriptor remains grammar, never an extra filename filter.

Each of120 leaves owns a real filesystem/Git fixture: four topologies x file/empty-directory x fifteen cases. Actual types/modes are verified with Lstat; files have recorded input hashes, directories remain empty; meaningful file byte change or0755-to0700 mode change is performed. Add/remove/rename are real filesystem operations, not receipt-only approximations. All generation/check commands run the ordinary freshly built CLI; native setup, Git and subprocess errors are fatal, with no mock, feature probe, skip gate or injected verdict.

Independent inventory assertions, full-manifest validity and each mutation are separate leaves. Thus an omitted-entry assertion never prevents another leaf's challenge. Both omission variants start from a complete independently enumerated inventory, remove exactly one none entry, and either retain its original full digest or independently hash the narrowed inventory. All eight old-digest cases exercise GV_SCOPE_HASH; the rehashed cases separately exercise actual-root comparison. The independent complete receipt is accepted in equal/ancestor and rejected in the four affected cells, explicitly observed in separate leaves rather than hidden as an unexecuted control.

Positive evidence controls change record presence/content and, for evidence-only, actual mode plus a real commit. Both real .git directories and regular --separate-git-dir gitfiles undergo local commits. Their digest stability/freshness legs run before the post-control independent inventory comparison. Unrelated sibling and nested attestations.yaml content changes require GV_STALE_CONTENT for their exact logical paths; nested .git must produce GV_SCOPE_UNSUPPORTED_METADATA at its actual path. Existing frozen root-narrowing/alias/custody restrictions remain unchanged.

Joined category-plus-message assertions prevent a temporary path containing none from satisfying the expected failing path. The rename expectation is justified: a complete sorted inventory contains removed none and added none-renamed; raw-ASCII first difference is none. R2 requires deterministic first differing portable path, and the unchanged comparator sorts the union before comparing. This is not arbitrary selection between unrelated paths. The four failing rename leaves correctly reject already; they expose the missing removed-path diagnostic and are not falsely classified as unsafe current.

## Independent execution and exact outcomes

One complete containing selector independently replayed all120 new leaves and28 existing CLI leaves on this exact candidate, avoiding redundant overlap:

`env GOWORK=off GOPROXY=off GOTOOLCHAIN=local go test -count=1 -timeout=5m -json ./cmd/machinery -run '^TestAttest'`

Go1.27.1 darwin/arm64, offline existing cache. goldenBin builds an ordinary isolated CLI with go build -o <temporary-binary> . from exact archived command source; actual binary path/digest are in raw JSONL. Existing TestMain isolates user control/configuration and cleans it after completion. cliReviewExec retains30s timeout and hermetic testgit retains10s bounds. No installed binary is used. No timeout expansion or new seam.

- New matrix:80 PASS /40 expected behavioral FAIL /0 SKIP, exactly120 terminal leaves.
- Existing compatibility:28 PASS /0 FAIL /0 SKIP, exactly28 terminal leaves.
- Total unique selector:108 PASS /40 FAIL /0 SKIP. Go package37.324s; synchronous runner wall39.406s. Go exits1 for expected RED assertions; runner exits0 after checking all required outcomes/reachability.
- All112 generated unchanged controls and104 post-control challenges reached, including all72 intended negative mutations. Zero missing leaf terminals, compile/import/setup/timeout/infrastructure errors, zero unreached challenges, zero unclassified failures. Raw command stderr is empty.
- Equal/ancestor:60 PASS. Each disjoint/inside-design type cell:5 PASS /10 FAIL.
-40 failures:16 independent inventory mismatches;4 complete independent manifests rejected;16 unsafe exit0/current1 results from content-or-mode/addition/removal/rehashed omission;4 rename removed-path diagnostic failures with actual exit1 and no current success.
- All8 old-digest forgeries,16 unrelated sibling/nested evidence mutations and8 nested-Git rejection cases pass. The12 affected post-control inventory failures follow successful actual evidence/Git freshness and stable digest legs.

Raw JSONL pm-attest.jsonl SHA2567fdf3a78ee5a3ce6803d9b7629981a8e4c7eaab55b67c30166489d68af036f5a retains full leaf identities, outcomes, exact argv, generated/submitted receipts, actual input metadata/hash and CLI output. pm-replay.json SHA25692f1a4636ee67e8fb8e2a7cf3d7f5e9dfac81bb05b14bb468284241ddfeb1e5d retains every leaf and full per-leaf output. pm-source-audit.json SHA256319038127f5c3ce73090731cac1e396646383341d313bbc10f7dbfaf443228da retains source/frozen/helper hashes, lineage, failure attribution and exact topology counts. Reproduction scripts: pm-replay.cjs SHA25602873c25598da71facf85fb21c723fd8b22300194a92defd902ed0155e70203b; pm-audit.cjs SHA256ae5c9d16830c7fa67d817fd249c5985864b2321476b24645b5dc6c8bdb9f8ad7. All are under the external PM directory above.

Minimal BUILD fixtures retain the ordinary nonblocking missing-g4.zero-context warning, equally present in passing controls and failing challenges. This is expected fixture semantics and previously documented, not a --complete or zero-warning claim. Exact failure category/path/exit/count assertions cannot be discharged by that warning. No coverage percentage was measured in this independent CLI replay; ordinary spawned CLI execution is real but uninstrumented. Prior broader quality findings and case-sensitive Linux final-gate leaf remain owed.

## Freeze, cost and completion hold

Seven frozen hashes independently match the prior manifest. Both full helper files are exact412d01b bytes: internal/hook/hook_test.go SHA2562b185970a3f69a140338cee97e14db0da82ef0aaf01250364357b104c1458b33 and internal/gates/obligation_ownership_test.go SHA256cc5f5e12a0d41a6eeda60895fece31713c5f068817bb2d3f08b7f1c6ff4d4b04. Exact function texts retain hashes copyTree58e8d45f3abe36516fa2a4436a4e8895b1aacada86dd40db5b30bb2697b3b31c and obligationParentFixture677c09b9b2f39072c3d30a9f2ff911f5a4b216b850a5b1fe7972ffea6dd4c846. The previously authorized36+29 additive fixture lines remain unchanged; no future hgz1 amendment is inferred.

pvg verify cmd/machinery/attest_scope_none_test.go --format text:1 file,0 issues. pvg story verify-tdd --range a82277a..c71faeab997d176224ed6f38437de4604cda0b98 --json:12 commits,0 merge skips,0 violations; candidate subject carries tdd-red before this independent replay. git diff --check is empty. Aggregate17 paths,3760 additions+66 deletions=3826 changedLOC; this increment364 added test lines, no production repair. Author113.199 cumulative package-seconds plus this PM37.324 =150.523 package-seconds; this arithmetic is not unique test coverage. Forecast was not treated as a proof cap.

Final GREEN must pass this new120-leaf frozen matrix plus original scoped attestation/designlock/hook proof, all13 helper callers, relevant broader and race proof, exact frozen/hash/TDD audit and independent whole-story review on one candidate. Original AC2/3/5 positive evidence and the approved OBSERVED/REVIEWED/COMPOSED/UNOBSERVED AC5 limits remain intact. Case-sensitive Linux and full preflight remain final-epic obligations. Product AC1/4 is still unresolved now. No production patch is prescribed or applied.

Only supported phase/claim operations and the append-only exact-review record are in scope. pvg story approve-red exited1 after moving status to open and removing delivered, because the historical red-approved label already existed. Readback confirmed open, old assignee retained, hard-tdd/red-approved, and the prior delivered contract still at EOF. This partial transition is not reported as command success. Supported pvg story release then cleared the old claim, preserving red-approved and open. This independently approved exact-test report is appended at actual EOF so the historical label no longer stands in for new review authority. No manual label choreography or loop recovery was used. design.machinery is off; no design gate waiver or installed Machinery invocation is needed. No accept/close/merge, source/test edit, remote operation, package/tool installation, agent/skill replacement, preflight, service/Docker/engine mutation occurred. Main497419ab4512fcff765cd5feb27aed4c67b5608d and epic70652b948bf090008b1965c85daf36ea374daea4 are unchanged; main working tree is clean. Every finite verification process completed. External evidence is retained.

## nd_contract
status: new

### evidence
- SUPPLEMENTAL RED APPROVED exact candidatec71faeab997d176224ed6f38437de4604cda0b98 and new-test SHA2563d9d8250f759807795251685e1cb924ba8eceb4c819b3fa0376bd15b254f9fc0. This new independent disposition supersedes the supplemental RED hold only; historical whole-story rejection remains valid until GREEN repair and final independent review.
- Independent actual replay: new80PASS/40behavioralFAIL/0SKIP; prior CLI28PASS/0FAIL/0SKIP; all120new leaves,112controls,104postcontrol challenges and72negatives reached. Full raw/source artifacts listed above.
- Seven old frozen files, both exact authorized helpers and new364-line test are immutable. The partially completed approve-red plus supported release returned the story to open/unassigned with hard-tdd/red-approved; this exact independent approval supplies new supplemental authority at EOF. Root owns the separate GREEN route. No acceptance, closure, production-text authority or product completion is claimed.

### proof
- [x] Supplemental RED AC1/4: exact-text independent review and real CLI replay establish a sound bounded regression bar, complete independent inventory/digest, meaningful reached negatives and preserved exclusions.
- [x] Supplemental RED AC2/3/5: prior semantics and immutable evidence retained;28 actual CLI compatibility leaves pass. Prior bounded AC5 classification preserved without claiming new late-fault execution.
- [ ] Product AC1/4: production correction and complete same-candidate GREEN proof remain required.
- [ ] Whole-story acceptance: independent final GREEN review pending; do not accept, close or merge from this RED approval.
