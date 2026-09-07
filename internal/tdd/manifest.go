// Package tdd declaration producer — RED safe-default skeleton. LoadPlan,
// LoadManifest, Validate and the projection/digest helpers are compile-only
// stubs: every entry point refuses with a deterministic not-implemented
// error so the frozen RED suite fails semantically (stubs refuse; no
// fabricated success, no partial validation).
package tdd

import "fmt"

// LoadPlan loads and validates design/assurance/plan.json over the held
// design snapshot directory.
func LoadPlan(design string) (Plan, error) {
	return Plan{}, fmt.Errorf("INVALID_SCHEMA: RED STUB: LoadPlan not implemented")
}

// LoadManifest loads and validates one milestone manifest file.
func LoadManifest(path string) (Manifest, error) {
	return Manifest{}, fmt.Errorf("INVALID_SCHEMA: RED STUB: LoadManifest not implemented")
}

// Validate reconciles a loaded plan, its manifests and the authoritative
// inventory as finalized declarations.
func Validate(plan Plan, manifests []Manifest, inventory Inventory) error {
	return fmt.Errorf("MISSING_CONTRACT: RED STUB: Validate not implemented")
}

// ReviewProjectionPlan returns canonical projection C of the plan.
func ReviewProjectionPlan(p Plan) []byte { return nil }

// ReviewProjectionManifest returns canonical projection C of the manifest.
func ReviewProjectionManifest(m Manifest) []byte { return nil }

// ReviewSubjectDigest derives review.subject_digest over the projection and
// the authoritative design payload digest.
func ReviewSubjectDigest(projection []byte, payloadDigest string) (string, error) {
	return "", fmt.Errorf("INVALID_SCHEMA: RED STUB: ReviewSubjectDigest not implemented")
}

// DesignPayloadDigest computes the typed design payload tree digest of the
// held design directory.
func DesignPayloadDigest(designDir string) (string, error) {
	return "", fmt.Errorf("INVALID_SCHEMA: RED STUB: DesignPayloadDigest not implemented")
}
