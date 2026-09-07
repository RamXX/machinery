// Replay-input retention core (docs/test-assurance-contract.md section 4):
// the exact machinery.tdd.tree/v1 typed topology encoding shared by source
// bundles and the control/judgment inventories, the closed bundle.json
// document, Capture over the held immutable InputView and verified
// materialization. Capture launches no subprocesses, claims no execution
// and is never RED evidence.

package tdd

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/tdd/protocol"
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

// treeEntry is one typed topology entry of a tree digest.
type treeEntry struct {
	rel  string
	dir  bool
	perm uint32
	size int64
	dig  [sha256.Size]byte
	role string
}

// encodeTreeDigest implements the exact typed encoding: UTF-8 domain
// machinery.tdd.tree/v1, entries sorted by UTF-8 path, each encoded as one
// kind byte (0 directory, 1 file), U64-big-endian path length and bytes,
// U32 permission bits, U64 logical size, 32 raw digest bytes (zero for
// directories), U64 role length and ASCII role. The digest excludes
// itself. The same encoder reproduces digests across platforms.
func encodeTreeDigest(entries []treeEntry) string {
	sorted := make([]treeEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].rel < sorted[j].rel })
	h := sha256.New()
	h.Write([]byte(protocol.TreeDigestDomain))
	var b8 [8]byte
	var b4 [4]byte
	for _, e := range sorted {
		if e.dir {
			h.Write([]byte{0})
		} else {
			h.Write([]byte{1})
		}
		binary.BigEndian.PutUint64(b8[:], uint64(len(e.rel)))
		h.Write(b8[:])
		h.Write([]byte(e.rel))
		binary.BigEndian.PutUint32(b4[:], e.perm)
		h.Write(b4[:])
		binary.BigEndian.PutUint64(b8[:], uint64(e.size))
		h.Write(b8[:])
		h.Write(e.dig[:])
		binary.BigEndian.PutUint64(b8[:], uint64(len(e.role)))
		h.Write(b8[:])
		h.Write([]byte(e.role))
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// treeDigestOfEntries is the entries-only convenience used by status.
func treeDigestOfEntries(entries []bundleEntryData) string {
	out := make([]treeEntry, 0, len(entries))
	for _, e := range entries {
		te := treeEntry{rel: e.Path, dir: e.Kind == "directory", perm: e.Mode, size: e.Size, role: e.Role}
		if e.Digest != "" {
			raw, _ := hex.DecodeString(strings.TrimPrefix(e.Digest, "sha256:"))
			copy(te.dig[:], raw)
		}
		out = append(out, te)
	}
	return encodeTreeDigest(out)
}

// bundleEntryData is the decoded closed bundle entry.
type bundleEntryData struct {
	Path   string
	Kind   string
	Mode   uint32
	Size   int64
	Digest string // "" for directories (JSON null)
	Role   string
}

// encodeBundleCanonical renders the closed bundle.json in encoding C with
// entries sorted canonically by path.
func encodeBundleCanonical(entries []bundleEntryData, treeDigest string) []byte {
	sorted := make([]bundleEntryData, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	members := make([]string, 0, len(sorted))
	for _, e := range sorted {
		members = append(members, canonObj(map[string]string{
			"digest": canonOptRef(e.Digest),
			"kind":   canonString(e.Kind),
			"mode":   canonInt(int64(e.Mode)),
			"path":   canonString(e.Path),
			"role":   canonString(e.Role),
			"size":   canonInt(e.Size),
		}))
	}
	return []byte(canonObj(map[string]string{
		"entries":     canonArr(members),
		"schema":      canonString(SchemaBundle),
		"tree_digest": canonString(treeDigest),
	}))
}

// decodeBundleClosed decodes bundle.json under the closed rules: exact
// keys, kinds, portable modes, digests, roles, sorted unique paths, parent
// presence and the recomputed tree digest.
func decodeBundleClosed(raw []byte) ([]bundleEntryData, string, error) {
	_, doc, err := readControlJSONFromBytes("bundle.json", raw)
	if err != nil {
		return nil, "", err
	}
	root, err := asObject(doc, "bundle.json")
	if err != nil {
		return nil, "", err
	}
	if err := exactKeys(root, "bundle.json", "schema", "entries", "tree_digest"); err != nil {
		return nil, "", err
	}
	schema, _ := reqStr(root, "schema", "bundle.json.schema")
	if schema != SchemaBundle {
		return nil, "", fmt.Errorf("UNSUPPORTED_VERSION: bundle schema %q is not %s", schema, SchemaBundle)
	}
	treeDigest, err := reqStr(root, "tree_digest", "bundle.json.tree_digest")
	if err != nil {
		return nil, "", err
	}
	if !validDigest(treeDigest) {
		return nil, "", fmt.Errorf("INVALID_SCHEMA: bundle tree_digest %q is not a digest", treeDigest)
	}
	eObjs, err := reqObjArray(root, "entries", "bundle.json.entries")
	if err != nil {
		return nil, "", err
	}
	var entries []bundleEntryData
	seen := map[string]bool{}
	for i, eo := range eObjs {
		where := fmt.Sprintf("bundle.json.entries[%d]", i)
		if err := exactKeys(eo, where, "path", "kind", "mode", "size", "digest", "role"); err != nil {
			return nil, "", err
		}
		var e bundleEntryData
		e.Path, err = reqStr(eo, "path", where+".path")
		if err != nil {
			return nil, "", err
		}
		if err := validateRootPath(e.Path); err != nil {
			return nil, "", fmt.Errorf("INVALID_SCHEMA: %s.path: %w", where, err)
		}
		e.Kind, err = reqStr(eo, "kind", where+".kind")
		if err != nil {
			return nil, "", err
		}
		if e.Kind != "file" && e.Kind != "directory" {
			return nil, "", fmt.Errorf("INVALID_SCHEMA: %s.kind %q must be file or directory", where, e.Kind)
		}
		mode, err := reqInt(eo, "mode", where+".mode")
		if err != nil {
			return nil, "", err
		}
		if mode < 0 || mode > 0o777 {
			return nil, "", fmt.Errorf("INVALID_SCHEMA: %s.mode %o is outside the portable rwx bits", where, mode)
		}
		e.Mode = uint32(mode)
		e.Size, err = reqInt(eo, "size", where+".size")
		if err != nil || e.Size < 0 {
			return nil, "", fmt.Errorf("INVALID_SCHEMA: %s.size must be a nonnegative integer", where)
		}
		e.Digest, err = optStr(eo, "digest", where+".digest")
		if err != nil {
			return nil, "", err
		}
		if e.Kind == "directory" {
			if e.Digest != "" || e.Size != 0 {
				return nil, "", fmt.Errorf("INVALID_SCHEMA: %s: directories have size 0 and null digest", where)
			}
		} else if !validDigest(e.Digest) {
			return nil, "", fmt.Errorf("INVALID_SCHEMA: %s: files carry a content digest", where)
		}
		e.Role, err = reqStr(eo, "role", where+".role")
		if err != nil {
			return nil, "", err
		}
		switch e.Role {
		case RoleFrozen, RoleSubject, RoleDesign, RoleDependency:
		default:
			return nil, "", fmt.Errorf("INVALID_SCHEMA: %s.role %q is not one of frozen|subject|design|dependency", where, e.Role)
		}
		if e.Path == protocol.RepositoryRoot && e.Kind != "directory" {
			return nil, "", fmt.Errorf("INVALID_SCHEMA: %s: the root path \".\" is only valid with kind directory", where)
		}
		if seen[e.Path] {
			return nil, "", fmt.Errorf("INVALID_SCHEMA: %s: duplicate path %q", where, e.Path)
		}
		seen[e.Path] = true
		entries = append(entries, e)
	}
	if len(entries) == 0 {
		return nil, "", fmt.Errorf("INVALID_SCHEMA: a bundle retains at least its root directory entry")
	}
	if _, ok := seen[protocol.RepositoryRoot]; !ok {
		return nil, "", fmt.Errorf("INVALID_SCHEMA: the captured repository root directory entry \".\" is missing")
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Path >= entries[i].Path {
			return nil, "", fmt.Errorf("INVALID_SCHEMA: bundle entries are not canonically sorted")
		}
	}
	byPath := map[string]bundleEntryData{}
	for _, e := range entries {
		byPath[e.Path] = e
	}
	for _, e := range entries {
		if e.Path == protocol.RepositoryRoot {
			continue
		}
		parent := parentSlashPath(e.Path)
		p, ok := byPath[parent]
		if !ok {
			return nil, "", fmt.Errorf("INVALID_SCHEMA: entry %q has no parent directory entry", e.Path)
		}
		if p.Kind != "directory" {
			return nil, "", fmt.Errorf("INVALID_SCHEMA: parent %q of %q collides with a file entry", parent, e.Path)
		}
	}
	recomputed := treeDigestOfEntries(entries)
	if recomputed != treeDigest {
		return nil, "", fmt.Errorf("INVALID_SCHEMA: bundle tree digest %s does not match the recomputed inventory %s", treeDigest, recomputed)
	}
	return entries, treeDigest, nil
}

func parentSlashPath(p string) string {
	i := strings.LastIndexByte(p, '/')
	if i < 0 {
		return protocol.RepositoryRoot
	}
	return p[:i]
}

// Capture captures one validated content-addressed bundle of the held
// immutable InputView into the explicit external store, archives the exact
// control bytes and binds the judgment-control identity. Non-subject files
// are frozen by default; the control namespace and judgment control are
// captured separately and never contaminate the source bundle.
func Capture(ctx context.Context, req CaptureRequest) (BundleRef, error) {
	ctx, cancel := nonExecContext(ctx)
	defer cancel()
	if err := checkCtx(ctx); err != nil {
		return BundleRef{}, err
	}
	if err := validateCaptureRequest(req); err != nil {
		return BundleRef{}, err
	}
	if err := ValidateStorePlacement(req.Store, req.Inputs.SourceRoot, req.Inputs.ControlRoot); err != nil {
		return BundleRef{}, err
	}
	v, err := openStore(ctx, req.Store, "")
	if err != nil {
		return BundleRef{}, err
	}
	defer v.close()
	engine, err := newCaptureEngine(ctx, req, v)
	if err != nil {
		return BundleRef{}, err
	}
	if err := engine.run(); err != nil {
		return BundleRef{}, err
	}
	if err := v.close(); err != nil {
		return BundleRef{}, err
	}
	return BundleRef{ref: engine.treeDigest, root: filepath.Join(req.Store, storeObjects, strings.TrimPrefix(engine.treeDigest, "sha256:")), valid: true}, nil
}

// MaterializeBundle materializes one captured bundle into a NEW destination
// with exact bytes, modes and empty-directory topology; every blob is
// read-verified and the final tree is re-verified before success.
func MaterializeBundle(ctx context.Context, storePath, projectID, ref, dest string) error {
	ctx, cancel := nonExecContext(ctx)
	defer cancel()
	if err := checkCtx(ctx); err != nil {
		return err
	}
	if !validDigest(ref) {
		return fmt.Errorf("INVALID_SCHEMA: bundle reference %q is not a digest", ref)
	}
	if fi, err := os.Lstat(dest); err == nil {
		_ = fi
		return fmt.Errorf("STORE_ROOT_MISMATCH: materialization destination %s already exists", dest)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("STORE_ROOT_MISMATCH: cannot inspect destination %s: %w", dest, err)
	}
	v, err := openStore(ctx, storePath, projectID)
	if err != nil {
		return err
	}
	defer v.close()
	bundlePath := filepath.Join(storePath, storeObjects, strings.TrimPrefix(ref, "sha256:"), "bundle.json")
	raw, _, err := readControlJSON(bundlePath)
	if err != nil {
		return fmt.Errorf("MISSING_CONTRACT: bundle %s is not retained in the store: %w", ref, err)
	}
	entries, treeDigest, err := decodeBundleClosed(raw)
	if err != nil {
		return err
	}
	if treeDigest != ref {
		return fmt.Errorf("INVALID_SCHEMA: bundle at %s carries tree digest %s", ref, treeDigest)
	}
	if string(encodeBundleCanonical(entries, treeDigest)) != string(raw) {
		return fmt.Errorf("INVALID_SCHEMA: bundle.json bytes are not the canonical encoding")
	}
	// read-verify every referenced blob before touching the destination
	content := map[string][]byte{}
	for _, e := range entries {
		if e.Kind != "file" {
			continue
		}
		data, err := readVerifiedBlob(storePath, e.Digest)
		if err != nil {
			return err
		}
		if int64(len(data)) != e.Size {
			return fmt.Errorf("INVALID_SCHEMA: blob %s length %d does not match the recorded size %d", e.Digest, len(data), e.Size)
		}
		content[e.Path] = data
	}
	if err := checkCtx(ctx); err != nil {
		return err
	}
	// materialize: ancestors first, exact files, then directory modes
	// deepest-first, then full topology re-verification
	if err := os.Mkdir(dest, 0o700); err != nil {
		return fmt.Errorf("CUSTODY_ERROR: creating materialization root: %w", err)
	}
	fail := func(err error) error {
		cctx, ccancel := cleanupContext()
		defer ccancel()
		_ = cctx
		if rmErr := os.RemoveAll(dest); rmErr != nil {
			return fmt.Errorf("%w; cleanup also failed: %w", err, rmErr)
		}
		return err
	}
	dirs := make([]bundleEntryData, 0, len(entries))
	for _, e := range entries {
		if e.Kind != "directory" || e.Path == protocol.RepositoryRoot {
			continue
		}
		dirs = append(dirs, e)
		if err := os.MkdirAll(filepath.Join(dest, filepath.FromSlash(e.Path)), 0o700); err != nil {
			return fail(fmt.Errorf("CUSTODY_ERROR: materializing %s: %w", e.Path, err))
		}
	}
	for _, e := range entries {
		if e.Kind != "file" {
			continue
		}
		p := filepath.Join(dest, filepath.FromSlash(e.Path))
		if err := durableWriteBytes(p, os.FileMode(e.Mode), content[e.Path]); err != nil {
			return fail(fmt.Errorf("CUSTODY_ERROR: materializing %s: %w", e.Path, err))
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Path > dirs[j].Path }) // deepest first
	for _, e := range dirs {
		if err := os.Chmod(filepath.Join(dest, filepath.FromSlash(e.Path)), os.FileMode(e.Mode)); err != nil {
			return fail(fmt.Errorf("CUSTODY_ERROR: applying directory mode of %s: %w", e.Path, err))
		}
	}
	rootMode := os.FileMode(0o755)
	for _, e := range entries {
		if e.Path == protocol.RepositoryRoot {
			rootMode = os.FileMode(e.Mode)
		}
	}
	if err := os.Chmod(dest, rootMode); err != nil {
		return fail(fmt.Errorf("CUSTODY_ERROR: applying root mode: %w", err))
	}
	// complete topology verification over the materialized tree
	verify := make([]treeEntry, 0, len(entries))
	for _, e := range entries {
		te := treeEntry{rel: e.Path, dir: e.Kind == "directory", perm: e.Mode, size: e.Size, role: e.Role}
		if e.Kind == "file" {
			data, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(e.Path)))
			if err != nil {
				return fail(fmt.Errorf("CUSTODY_ERROR: verifying %s: %w", e.Path, err))
			}
			sum := sha256.Sum256(data)
			te.dig = sum
		}
		fi, err := os.Lstat(filepath.Join(dest, filepath.FromSlash(e.Path)))
		if err != nil || fi.Mode().Perm() != os.FileMode(e.Mode) {
			return fail(fmt.Errorf("CUSTODY_ERROR: verified topology differs at %s", e.Path))
		}
		verify = append(verify, te)
	}
	if encodeTreeDigest(verify) != ref {
		return fail(fmt.Errorf("STALE_INPUT: materialized topology does not reproduce the bundle identity"))
	}
	return v.close()
}

// readVerifiedBlob reads one content-addressed blob and verifies its bytes
// hash to its address on every read.
func readVerifiedBlob(storePath, digest string) ([]byte, error) {
	blobPath := filepath.Join(storePath, storeBlobs, strings.TrimPrefix(digest, "sha256:"))
	fi, err := os.Lstat(blobPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("INVALID_SCHEMA: blob %s is missing from the store", digest)
		}
		return nil, err
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("INVALID_SCHEMA: blob %s is not a regular non-symlink file", digest)
	}
	data, err := os.ReadFile(blobPath)
	if err != nil {
		return nil, err
	}
	if digestOfBytes(data) != digest {
		return nil, fmt.Errorf("INVALID_SCHEMA: blob %s does not hash to its address; the store is corrupt", digest)
	}
	return data, nil
}
