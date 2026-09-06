---
id: MAC-p7jd
title: "Invalidate reviews when implementation subjects change"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd, red-approved]
parent: MAC-ui8a
created_at: 2026-09-05T19:33:46Z
created_by: ramirosalas
updated_at: 2026-09-06T05:06:11Z
content_hash: "sha256:949d7bcd5fde8dfe89563e9bed4aec45b00a06ff02ba46b36b9a0976b5713bff"
blocks: [MAC-vx24, MAC-gcrr, MAC-ou97, MAC-hgz1]
assignee: dev-MAC-p7jd
follows: [MAC-p8ce, MAC-2u36, MAC-a89e]
---

## Description
## USER INTENT
Strengthen Machinery mission-critical assurance with observable fail-closed behavior and precise limits.

## Context (Embedded)
Assessment F3: gt.conformance-test-shape covers BUILD artifacts, while g4.pack-event-discipline covers pack. Neither binds tested/reviewed implementation. Historic Ga ancestor acceptance is valid history, not current-tree assurance. Preserve judgment-vs-mechanical distinction.

## Ownership
The following 13 required files are the reviewed ownership boundary; implementation and test editing remain subject to their independent phase authorizations. You are not alone: preserve other edits, especially accepted MAC-p8ce/MAC-olrx behavior; coordinate shared paths. No generic helper factoring is authorized. Optional internal/gates/attest_green_test.go is the reported fourteenth file only for independent supplemental GREEN proof.

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
- Current independently reviewed forecast: 13 required paths plus the already-authorized optional fourteenth GREEN-supplemental file, approximately 2710–3310 total changed LOC (~1485 RED,175–375 supplemental GREEN tests,900–1250 production,150–200 docs/help; combined tests~1660–1860). Supersedes the former1900–2900 and original4–7/<1000 estimates as cost history only. Measured repaired RED47ba44906a09bc2fa010092a86b133d0d749c52c is five test files,1484 insertions/3 deletions (1487 changed lines); original7ec5d609 had1475 insertions/3 deletions. Rounded forecasts are not exact final totals, caps, completion proof or permission to expand ownership/trim proof.

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

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-gcrr]], [[MAC-ou97]], [[MAC-hgz1]]
- Follows: [[MAC-p8ce]], [[MAC-2u36]], [[MAC-a89e]]

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
