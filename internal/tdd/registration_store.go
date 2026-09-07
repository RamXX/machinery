// Authored-revision registration surface of docs/test-assurance-contract.md
// section 4: pure lineage validation and deterministic next-head
// construction plus the durable store registration transaction. tdd owns
// the derivation; internal/assuranceflow owns commit authority through
// CommitRegistration. The transaction runs under the fixed 600000 ms
// non-execution owner deadline with one 10000 ms cleanup grace, launches no
// subprocesses, never retries automatically and NEVER advances the head for
// any execution/status operation — only Register commits authored revisions.
package tdd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/tdd/protocol"
)

// Registration state literals of the authored state machine. Registration
// proves authored state only; it is never execution evidence.
const (
	StateRegisteredNotExecuted        = "registered-not-executed"
	StateAlreadyRegisteredNotExecuted = "already-registered-not-executed"
)

// RegisteredManifest binds one milestone entry of an expected head with the
// exact bytes that entry names (archived control bytes for cross-design
// resolution, current loaded bytes for untargeted same-design entries).
type RegisteredManifest struct {
	Key      MilestoneKey
	Revision int64
	Digest   string
	Manifest Manifest
}

// HeadPlanEntry is one plan entry of a decoded head.
type HeadPlanEntry struct {
	Design     string
	PlanDigest string
}

// HeadMilestoneEntry is one milestone entry of a decoded head.
type HeadMilestoneEntry struct {
	Key            MilestoneKey
	Revision       int64
	ManifestDigest string
	Predecessor    string // "" encodes null
}

// ExpectedHead is the decoded closed view of one archived head document.
type ExpectedHead struct {
	ProjectID  string
	Generation int64
	Digest     string
	Previous   string
	Plans      []HeadPlanEntry
	Milestones []HeadMilestoneEntry
}

// DecodeArchivedManifest decodes exact archived control bytes for one head
// milestone entry. Review binding was verified when that revision was
// registered; archived bytes are immutable, so the projection is not
// re-bound against a payload this checkout may not hold. The repository-
// root design "." resolves through its own checkout (LoadManifest), never
// through archived decoding without a payload.
func DecodeArchivedManifest(design, milestone string, raw []byte) (Manifest, error) {
	if !validMilestoneID(milestone) {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: archived milestone %q is not canonical", milestone)
	}
	if design == protocol.RepositoryRoot {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: the repository-root design resolves through its own checkout, not archived decoding without a payload")
	}
	if err := validateRootPath(design); err != nil {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: archived design root: %v", err)
	}
	msPath := filepath.Join(protocol.ControlDirName, protocol.MilestonesDirName, milestone+".json")
	_, doc, err := readControlJSONFromBytes(msPath, raw)
	if err != nil {
		return Manifest{}, err
	}
	m, err := decodeManifest(doc, msPath, design, "", "", raw)
	if err != nil {
		return Manifest{}, err
	}
	if m.ID != milestone {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: archived manifest for milestone %q declares milestone %q", milestone, m.ID)
	}
	return m, nil
}

// DecodeExpectedHead decodes closed archived head bytes with full chain-rule
// validation; the bytes must be the canonical encoding.
func DecodeExpectedHead(raw []byte) (ExpectedHead, error) {
	h, err := decodeHeadClosed(raw)
	if err != nil {
		return ExpectedHead{}, err
	}
	eh := ExpectedHead{ProjectID: h.ProjectID, Generation: h.Generation, Digest: digestOfBytes(raw), Previous: h.Previous}
	for _, p := range h.Plans {
		eh.Plans = append(eh.Plans, HeadPlanEntry{Design: p.Design, PlanDigest: p.PlanDigest})
	}
	for _, m := range h.Milestones {
		eh.Milestones = append(eh.Milestones, HeadMilestoneEntry{
			Key:      MilestoneKey{Design: m.Design, Milestone: m.Milestone},
			Revision: m.Revision, ManifestDigest: m.ManifestDi, Predecessor: m.PredecessorDi,
		})
	}
	return eh, nil
}

// ValidateRegistrationGraph validates the registration transaction shape:
// selected targets are finalized manifests of current inventory milestones,
// obligation keys are authoritative (never invented), and every referenced
// test resolves to a selected target or an exactly registered manifest.
// Strict completeness obligations belong to complete/strict, not register.
func ValidateRegistrationGraph(plan Plan, targets []Manifest, registered []RegisteredManifest, inv Inventory) error {
	return fmt.Errorf("MISSING_CONTRACT: registration graph validation is unavailable in this build")
}

// RegistrationInputs carries the deterministic successor inputs.
type RegistrationInputs struct {
	ProjectID  string
	Plan       Plan
	PlanBytes  []byte
	Targets    []Manifest // selected finalized milestones of Plan.Design
	Untargeted []RegisteredManifest
}

// RegistrationControl is one exact control byte set to archive.
type RegistrationControl struct {
	Digest string
	Bytes  []byte
}

// RegistrationSuccessor is the deterministic desired successor of one
// registration transaction against its expected head.
type RegistrationSuccessor struct {
	HeadBytes      []byte
	HeadDigest     string
	PreviousDigest string
	Generation     int64
	Controls       []RegistrationControl
	BundleRefs     []string
	TargetKeys     []MilestoneKey
}

// BuildRegistrationSuccessor validates lineage (revision 1/null for absent
// keys, prior revision+1 and the exact same-key predecessor otherwise, plan
// change advances every registered milestone of the design, registered
// deletion and untargeted registered control mismatch block) and derives the
// canonical next head plus the exact controls to archive. It performs no I/O.
func BuildRegistrationSuccessor(expectedDigest string, expectedRaw []byte, in RegistrationInputs) (RegistrationSuccessor, error) {
	return RegistrationSuccessor{}, fmt.Errorf("INVALID_SCHEMA: successor construction is unavailable in this build")
}

// ReadArchivedHead opens the store read-only under the plan's project
// identity and returns the exact archived bytes of the expected head.
// A missing or non-matching archive is HISTORY_UNAVAILABLE.
func ReadArchivedHead(ctx context.Context, store, projectID, expectedHead string) ([]byte, error) {
	ctx, cancel := nonExecContext(ctx)
	defer cancel()
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	if !validDigest(expectedHead) {
		return nil, fmt.Errorf("INVALID_SCHEMA: expected head %q is not a digest; there is no wildcard comparison", expectedHead)
	}
	v, err := openStore(ctx, store, projectID)
	if err != nil {
		return nil, err
	}
	defer v.close()
	raw, rerr := os.ReadFile(filepath.Join(store, storeLedger, storeHeads, strings.TrimPrefix(expectedHead, "sha256:")+".json"))
	if rerr != nil || digestOfBytes(raw) != expectedHead {
		return nil, fmt.Errorf("HISTORY_UNAVAILABLE: the archived expected head %s is not available byte-exact in store %s", expectedHead, store)
	}
	if _, derr := decodeHeadClosed(raw); derr != nil {
		return nil, fmt.Errorf("HISTORY_UNAVAILABLE: the archived expected head %s does not decode: %v", expectedHead, derr)
	}
	return raw, nil
}

// ReadArchivedControl returns the exact archived control bytes for one
// digest, verified byte-identical against the address. A committed head
// entry whose control bytes are missing is a broken chain, not fresh state.
func ReadArchivedControl(ctx context.Context, store, projectID, digest string) ([]byte, error) {
	ctx, cancel := nonExecContext(ctx)
	defer cancel()
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	if !validDigest(digest) {
		return nil, fmt.Errorf("INVALID_SCHEMA: control digest %q is not a digest", digest)
	}
	v, err := openStore(ctx, store, projectID)
	if err != nil {
		return nil, err
	}
	defer v.close()
	raw, rerr := os.ReadFile(filepath.Join(store, "controls", strings.TrimPrefix(digest, "sha256:")+".json"))
	if rerr != nil || digestOfBytes(raw) != digest {
		return nil, fmt.Errorf("CONTROL_ROLLBACK: archived control %s is not available byte-exact in store %s; a committed head cannot name missing controls", digest, store)
	}
	return raw, nil
}

// StoreRegistrationRequest is one durable registration transaction.
type StoreRegistrationRequest struct {
	Store        string
	ProjectID    string
	ExpectedHead string
	Successor    RegistrationSuccessor
	Revalidate   func() error // final input revalidation before publication
}

// StoreRegistrationResult reports the committed or idempotent outcome.
// Idempotent marks the exact already-registered exception.
type StoreRegistrationResult struct {
	StoreID      string
	ProjectID    string
	PreviousHead string
	HeadDigest   string
	Generation   int64
	Idempotent   bool
}

// CommitRegistration performs the section-4 registration transaction:
// release the read reservation, acquire the external-store writer, compare
// the exact expected head (the idempotent exception is checked here),
// archive and stage the derived successor, revalidate the original inputs,
// then durably compare-and-advance head.json exactly one generation. The
// CAS loser receives HEAD_CONFLICT; nothing rebases or retries.
func CommitRegistration(ctx context.Context, req StoreRegistrationRequest) (StoreRegistrationResult, error) {
	return StoreRegistrationResult{}, fmt.Errorf("MISSING_CONTRACT: registration commit is unavailable in this build")
}
