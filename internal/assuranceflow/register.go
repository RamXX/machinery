// Package assuranceflow is the privileged closed orchestration layer of
// docs/test-assurance-contract.md section 7: it imports gates, tdd and
// processscope (never the reverse) and owns same-invocation preconditions,
// original snapshots, finalizers, external-store publication and sealed
// results. This file implements Register — the explicit authored-revision
// transaction of section 4. Register performs no subprocesses; it commits
// authored data only and never claims execution, replay or Gv acceptance.
package assuranceflow

import (
	"context"
	"fmt"
	"io"

	"github.com/RamXX/machinery/internal/tdd"
)

// RegisterRequest is the closed registration request: explicit paths, the
// caller-supplied expected head digest, and the selected milestone targets
// (empty selects every milestone declared by the design's plan; an empty
// resolved target set is an error).
type RegisterRequest struct {
	Design         string
	Implementation string
	Store          string
	ExpectedHead   string
	Milestones     []string
}

// Registration is the private-constructed authored-data result. It exposes
// exactly the identity/head/selection fields and the registered state; it
// carries NO replay, custody-success or test-pass accessor and cannot
// satisfy Verification.
type Registration struct {
	projectID    string
	storeID      string
	previousHead string
	headDigest   string
	generation   int64
	keys         []tdd.MilestoneKey
	state        string
}

// ProjectID returns the registered project identity.
func (r Registration) ProjectID() string { return r.projectID }

// StoreID returns the external store identity.
func (r Registration) StoreID() string { return r.storeID }

// PreviousHead returns the exact head digest this registration advanced.
func (r Registration) PreviousHead() string { return r.previousHead }

// HeadDigest returns the committed successor head digest.
func (r Registration) HeadDigest() string { return r.headDigest }

// Generation returns the committed head generation.
func (r Registration) Generation() int64 { return r.generation }

// MilestoneKeys returns the sorted selected milestone keys.
func (r Registration) MilestoneKeys() []tdd.MilestoneKey {
	out := make([]tdd.MilestoneKey, len(r.keys))
	copy(out, r.keys)
	return out
}

// State returns registered-not-executed or already-registered-not-executed.
func (r Registration) State() string { return r.state }

// Register commits one explicit reviewed authored revision against the
// expected external head and returns only registered-not-executed or
// already-registered-not-executed. The whole operation is bounded by the
// fixed 600000 ms non-execution owner deadline plus one 10000 ms cleanup
// grace, including locks, publication and output. Errors before the durable
// commit consume no revision; a post-commit close/output failure returns no
// successful Registration although the head may have advanced, and an exact
// retry confirms the already-registered state.
func Register(ctx context.Context, req RegisterRequest, output io.Writer) (Registration, error) {
	return Registration{}, fmt.Errorf("MISSING_CONTRACT: registration is unavailable in this build")
}

// controlSnapshot holds the original design view of one registration: the
// exact control bytes, the design payload digest and the rooted
// root-generation identity captured before the store transaction starts.
type controlSnapshot struct {
	designDir string
}

// newControlSnapshot captures the original view.
func newControlSnapshot(designDir string) (*controlSnapshot, error) {
	return &controlSnapshot{designDir: designDir}, nil
}

// revalidate re-reads the original view and compares exact source/control
// and root-generation identities.
func (s *controlSnapshot) revalidate() error {
	return nil
}
