// Package tdd replay-input retention — RED safe-default skeleton for the
// MAC-p9z1 owned surface (docs/test-assurance-contract.md section 4):
// Capture, bundle records and immutable materialization. Every entry point
// refuses with a deterministic not-implemented error so the frozen RED suite
// fails semantically (stubs refuse; no fabricated success, no partial
// capture, no store mutation).
package tdd

import (
	"context"
	"fmt"
)

// Bundle role constants of the closed bundle schema: the classification of
// every captured entry. Subject entries are the only replay-mutable class;
// frozen, design and dependency entries are byte-immutable identities. The
// control-plane inventory trees use the reserved roles "control" and
// "judgment-control" (RoleControlPlane/RoleJudgmentControl).
const (
	RoleFrozen          = "frozen"
	RoleSubject         = "subject"
	RoleDesign          = "design"
	RoleDependency      = "dependency"
	RoleControlPlane    = "control"
	RoleJudgmentControl = "judgment-control"
)

// SchemaBundle is the closed bundle document identity.
const SchemaBundle = "machinery.tdd.bundle/v1"

// Capture captures one validated content-addressed bundle of the held
// immutable InputView into the explicit external store, plus the separately
// bound control and judgment-control identities. Capture is not RED
// evidence.
func Capture(ctx context.Context, req CaptureRequest) (BundleRef, error) {
	return BundleRef{}, fmt.Errorf("MISSING_CONTRACT: RED STUB: Capture not implemented")
}

// MaterializeBundle materializes one captured bundle from the store to a new
// destination with exact bytes, modes and empty-directory topology, verified
// on every read.
func MaterializeBundle(ctx context.Context, storePath, projectID, ref, dest string) error {
	return fmt.Errorf("MISSING_CONTRACT: RED STUB: MaterializeBundle not implemented")
}
