// RED safe-default skeleton for the MAC-p9z1 external store surface:
// initialization, root-identity validation, head chain custody, placement
// rules and bounded transport. Every entry point refuses with a
// deterministic not-implemented error; no store path is created, adopted or
// mutated by a stub.
package tdd

import (
	"context"
	"fmt"
)

// Closed store-surface schema identities of the external store.
const (
	SchemaStore  = "machinery.tdd.store/v1"
	SchemaHead   = "machinery.tdd.head/v1"
	SchemaExport = "machinery.tdd.export/v1"
)

// StoreInitResult is the durable generation-zero initialization outcome.
type StoreInitResult struct {
	StoreID    string
	ProjectID  string
	HeadDigest string
	Generation int64
}

// StoreExportResult reports one bounded closed archive export.
type StoreExportResult struct {
	StoreID    string
	ProjectID  string
	HeadDigest string
	Generation int64
	Entries    int64
	TotalBytes int64
	Archive    string
}

// StoreImportResult reports one validated atomic import publication. State
// is always recorded-only; import never certifies replay.
type StoreImportResult struct {
	StoreID    string
	ProjectID  string
	HeadDigest string
	Generation int64
	Blobs      int
	Objects    int
	Controls   int
	Heads      int
	Runs       int
	State      string
}

// InitStore atomically creates a new 0700 external store with the closed
// store.json identity and the durable generation-zero head.
func InitStore(ctx context.Context, path, projectID string) (StoreInitResult, error) {
	return StoreInitResult{}, fmt.Errorf("MISSING_STORE: RED STUB: InitStore not implemented")
}

// ExportStore writes a new bounded closed archive of the complete store
// history with checksums; no live locks or transaction state.
func ExportStore(ctx context.Context, storePath, outPath string) (StoreExportResult, error) {
	return StoreExportResult{}, fmt.Errorf("MISSING_STORE: RED STUB: ExportStore not implemented")
}

// ImportStore validates and atomically publishes an archived store into a
// NEW destination under the caller-supplied expected head digest.
func ImportStore(ctx context.Context, destPath, archivePath, projectID, expectedHead string) (StoreImportResult, error) {
	return StoreImportResult{}, fmt.Errorf("INVALID_SCHEMA: RED STUB: ImportStore not implemented")
}

// ValidateStorePlacement rejects a store that overlaps any governed root by
// canonical alias or parent/child relationship, not lexical prefixes.
func ValidateStorePlacement(storePath string, governedRoots ...string) error {
	return fmt.Errorf("STORE_ROOT_MISMATCH: RED STUB: ValidateStorePlacement not implemented")
}
