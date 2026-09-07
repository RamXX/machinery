// External store custody for the closed v1 executable-assurance contract
// (docs/test-assurance-contract.md section 4): durable generation-zero
// initialization of a new 0700 root, rooted-identity validation, the
// canonical head chain with fail-closed rollback detection, placement rules
// and the bounded immutable transport (export/import). Every operation runs
// under the fixed 600000 ms non-execution owner deadline with one 10000 ms
// cleanup grace, launches no subprocesses and never retries automatically.
// No environment variable is ever consulted for store discovery.
package tdd

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/tdd/protocol"
)

// Closed store-surface schema identities of the external store.
const (
	SchemaStore  = "machinery.tdd.store/v1"
	SchemaHead   = "machinery.tdd.head/v1"
	SchemaExport = "machinery.tdd.export/v1"
)

// StoreIdentity is the decoded closed store.json identity.
type StoreIdentity struct {
	StoreID   string
	ProjectID string
}

// StoreInitResult is the durable generation-zero initialization outcome.
type StoreInitResult struct {
	StoreID    string
	ProjectID  string
	HeadDigest string
	Generation int64
}

// StoreExportResult reports one bounded closed-archive export.
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

// Store namespaces of the closed layout.
const (
	storeObjects = "objects"
	storeBlobs   = "blobs"
	storeRuns    = "runs"
	storeLedger  = "ledger"
	storeHeads   = "heads"
	storeStaging = "staging"
	storeRootMod = 0o700
	storeFileMod = 0o600
	storeBlobMod = 0o400
)

var blobNameRe = regexp.MustCompile(`^[0-9a-f]{64}$`)
var runNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}\.json$`)

// nonExecContext imposes the fixed 600000 ms owner deadline from entry; an
// earlier caller deadline still wins.
func nonExecContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, time.Duration(protocol.LimitWallDefaultMS)*time.Millisecond)
}

// cleanupContext is the single 10000 ms final cleanup grace.
func cleanupContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(protocol.LimitCleanupDefaultMS)*time.Millisecond)
}

func ctxFailure(ctx context.Context) error {
	switch {
	case ctx == nil:
		return nil
	case errors.Is(ctx.Err(), context.Canceled):
		return fmt.Errorf("TIMEOUT: operation cancelled before completion; no result is claimed")
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return fmt.Errorf("TIMEOUT: non-execution owner deadline exceeded; no result is claimed")
	default:
		return nil
	}
}

func checkCtx(ctx context.Context) error {
	if err := ctxFailure(ctx); err != nil {
		return err
	}
	return nil
}

// fsyncDir durably flushes one directory.
func fsyncDir(path string) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := d.Sync(); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}

// durableWriteBytes writes exact bytes to a NEW path (EEXIST is an error):
// create, write, fsync, close; failure removes the partial file.
func durableWriteBytes(path string, perm os.FileMode, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return err
	}
	return nil
}

// publishImmutableFile atomically publishes exact bytes at a FINAL path
// that may already exist: the bytes are written to a private temp file,
// fsynced, then hard-linked into place (atomic no-replace). An existing
// final path is verified byte-identical; divergence is corruption.
func publishImmutableFile(finalPath string, perm os.FileMode, data []byte) error {
	dir := filepath.Dir(finalPath)
	nonce := make([]byte, 8)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	tmp := filepath.Join(dir, fmt.Sprintf(".publish-%s", hex.EncodeToString(nonce)))
	if err := durableWriteBytes(tmp, storeFileMod, data); err != nil {
		return err
	}
	defer os.Remove(tmp)
	if err := os.Chmod(tmp, perm); err != nil {
		return err
	}
	if err := os.Link(tmp, finalPath); err != nil {
		if !errors.Is(err, fs.ErrExist) {
			return err
		}
		existing, rerr := os.ReadFile(finalPath)
		if rerr != nil {
			return rerr
		}
		if string(existing) != string(data) {
			return fmt.Errorf("INVALID_SCHEMA: immutable object %s holds different bytes than the canonical derivation; the store is corrupt", finalPath)
		}
		return nil
	}
	return fsyncDir(dir)
}

// newUUID mints a random v4 UUID.
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32]), nil
}

// ---- canonical head encoding ----

type headPlanEntry struct{ Design, PlanDigest string }

type headMilestoneEntry struct {
	Design        string
	Milestone     string
	Revision      int64
	ManifestDi    string
	PredecessorDi string // "" encodes null
}

type headDoc struct {
	ProjectID  string
	Generation int64
	Previous   string // "" encodes null
	Plans      []headPlanEntry
	Milestones []headMilestoneEntry
}

// encodeHeadCanonical renders the closed head record in the canonical JSON
// encoding C: sorted keys, sorted plans (by design) and sorted milestones
// (by design, milestone), no whitespace, shortest decimals.
func encodeHeadCanonical(h headDoc) []byte {
	plans := make([]string, 0, len(h.Plans))
	seen := map[string]bool{}
	for _, p := range h.Plans {
		if seen[p.Design] {
			continue
		}
		seen[p.Design] = true
		plans = append(plans, canonObj(map[string]string{
			"design": canonString(p.Design), "plan_digest": canonString(p.PlanDigest),
		}))
	}
	sort.Slice(plans, func(i, j int) bool { return plans[i] < plans[j] })
	ms := make([]headMilestoneEntry, len(h.Milestones))
	copy(ms, h.Milestones)
	sort.Slice(ms, func(i, j int) bool {
		if ms[i].Design != ms[j].Design {
			return ms[i].Design < ms[j].Design
		}
		return ms[i].Milestone < ms[j].Milestone
	})
	members := make([]string, 0, len(ms))
	mkeys := map[string]bool{}
	for _, m := range ms {
		k := m.Design + "\x00" + m.Milestone
		if mkeys[k] {
			continue
		}
		mkeys[k] = true
		members = append(members, canonObj(map[string]string{
			"design": canonString(m.Design), "manifest_digest": canonString(m.ManifestDi),
			"milestone": canonString(m.Milestone), "predecessor": canonOptRef(m.PredecessorDi),
			"revision": canonInt(m.Revision),
		}))
	}
	return []byte(canonObj(map[string]string{
		"generation": canonInt(h.Generation),
		"milestones": canonArr(members),
		"plans":      canonArr(plans),
		"previous":   canonOptRef(h.Previous),
		"project_id": canonString(h.ProjectID),
		"schema":     canonString(SchemaHead),
	}))
}

// decodeHeadClosed decodes one head document with the closed rules and
// enforces that the supplied bytes ARE the canonical encoding.
func decodeHeadClosed(raw []byte) (headDoc, error) {
	_, doc, err := readControlJSONFromBytes("ledger/head.json", raw)
	if err != nil {
		return headDoc{}, err
	}
	root, err := asObject(doc, "head.json")
	if err != nil {
		return headDoc{}, err
	}
	if err := exactKeys(root, "head.json", "schema", "project_id", "generation", "previous", "plans", "milestones"); err != nil {
		return headDoc{}, err
	}
	schema, _ := reqStr(root, "schema", "head.json.schema")
	if schema != SchemaHead {
		return headDoc{}, fmt.Errorf("INVALID_SCHEMA: head schema %q is not %s", schema, SchemaHead)
	}
	var h headDoc
	h.ProjectID, err = reqStr(root, "project_id", "head.json.project_id")
	if err != nil {
		return headDoc{}, err
	}
	if !validUUID(h.ProjectID) {
		return headDoc{}, fmt.Errorf("INVALID_SCHEMA: head project_id %q is not a UUID", h.ProjectID)
	}
	h.Generation, err = reqInt(root, "generation", "head.json.generation")
	if err != nil {
		return headDoc{}, err
	}
	if h.Generation < 0 {
		return headDoc{}, fmt.Errorf("INVALID_SCHEMA: head generation must be nonnegative")
	}
	h.Previous, err = optStr(root, "previous", "head.json.previous")
	if err != nil {
		return headDoc{}, err
	}
	if h.Previous != "" && !validDigest(h.Previous) {
		return headDoc{}, fmt.Errorf("INVALID_SCHEMA: head previous %q is not a digest", h.Previous)
	}
	if h.Generation == 0 && h.Previous != "" {
		return headDoc{}, fmt.Errorf("CONTROL_ROLLBACK: the initial generation-zero head must carry a null previous")
	}
	if h.Generation > 0 && h.Previous == "" {
		return headDoc{}, fmt.Errorf("CONTROL_ROLLBACK: a successor head must name its predecessor")
	}
	planObjs, err := reqObjArray(root, "plans", "head.json.plans")
	if err != nil {
		return headDoc{}, err
	}
	seenP := map[string]bool{}
	for i, po := range planObjs {
		where := fmt.Sprintf("head.json.plans[%d]", i)
		if err := exactKeys(po, where, "design", "plan_digest"); err != nil {
			return headDoc{}, err
		}
		d, err := reqStr(po, "design", where+".design")
		if err != nil {
			return headDoc{}, err
		}
		if err := validateRootPath(d); err != nil {
			return headDoc{}, fmt.Errorf("INVALID_SCHEMA: %s.design: %w", where, err)
		}
		pd, err := reqStr(po, "plan_digest", where+".plan_digest")
		if err != nil {
			return headDoc{}, err
		}
		if !validDigest(pd) {
			return headDoc{}, fmt.Errorf("INVALID_SCHEMA: %s.plan_digest %q is not a digest", where, pd)
		}
		if seenP[d] {
			return headDoc{}, fmt.Errorf("INVALID_SCHEMA: %s: duplicate plan design %q; plans are unique", where, d)
		}
		seenP[d] = true
		h.Plans = append(h.Plans, headPlanEntry{Design: d, PlanDigest: pd})
	}
	msObjs, err := reqObjArray(root, "milestones", "head.json.milestones")
	if err != nil {
		return headDoc{}, err
	}
	seenM := map[string]bool{}
	for i, mo := range msObjs {
		where := fmt.Sprintf("head.json.milestones[%d]", i)
		if err := exactKeys(mo, where, "design", "milestone", "revision", "manifest_digest", "predecessor"); err != nil {
			return headDoc{}, err
		}
		d, err := reqStr(mo, "design", where+".design")
		if err != nil {
			return headDoc{}, err
		}
		if err := validateRootPath(d); err != nil {
			return headDoc{}, fmt.Errorf("INVALID_SCHEMA: %s.design: %w", where, err)
		}
		mid, err := reqStr(mo, "milestone", where+".milestone")
		if err != nil {
			return headDoc{}, err
		}
		if !validMilestoneID(mid) {
			return headDoc{}, fmt.Errorf("INVALID_SCHEMA: %s.milestone %q is not canonical", where, mid)
		}
		k := d + "\x00" + mid
		if seenM[k] {
			return headDoc{}, fmt.Errorf("INVALID_SCHEMA: %s: duplicate milestone entry for %s/%s", where, d, mid)
		}
		seenM[k] = true
		rev, err := reqInt(mo, "revision", where+".revision")
		if err != nil {
			return headDoc{}, err
		}
		if rev <= 0 {
			return headDoc{}, fmt.Errorf("INVALID_SCHEMA: %s.revision must be positive", where)
		}
		md, err := reqStr(mo, "manifest_digest", where+".manifest_digest")
		if err != nil {
			return headDoc{}, err
		}
		if !validDigest(md) {
			return headDoc{}, fmt.Errorf("INVALID_SCHEMA: %s.manifest_digest %q is not a digest", where, md)
		}
		pred, err := optStr(mo, "predecessor", where+".predecessor")
		if err != nil {
			return headDoc{}, err
		}
		if pred != "" && !validDigest(pred) {
			return headDoc{}, fmt.Errorf("INVALID_SCHEMA: %s.predecessor %q is not a digest", where, pred)
		}
		h.Milestones = append(h.Milestones, headMilestoneEntry{Design: d, Milestone: mid, Revision: rev, ManifestDi: md, PredecessorDi: pred})
	}
	// sorted-array canonical order (plans by design, milestones by pair)
	for i := 1; i < len(h.Plans); i++ {
		if h.Plans[i-1].Design >= h.Plans[i].Design {
			return headDoc{}, fmt.Errorf("INVALID_SCHEMA: head plans are not canonically sorted")
		}
	}
	for i := 1; i < len(h.Milestones); i++ {
		a, b := h.Milestones[i-1], h.Milestones[i]
		if a.Design > b.Design || (a.Design == b.Design && a.Milestone >= b.Milestone) {
			return headDoc{}, fmt.Errorf("INVALID_SCHEMA: head milestones are not canonically sorted")
		}
	}
	// the bytes must BE the canonical encoding; anything else is tampering
	if string(encodeHeadCanonical(h)) != string(raw) {
		return headDoc{}, fmt.Errorf("CONTROL_ROLLBACK: head.json bytes are not the canonical head encoding")
	}
	if h.Generation == 0 && (len(h.Plans) > 0 || len(h.Milestones) > 0) {
		return headDoc{}, fmt.Errorf("CONTROL_ROLLBACK: the initial generation-zero head must carry empty arrays")
	}
	return h, nil
}

// ---- store root validation and opening ----

type storeView struct {
	root       string
	dir        *os.File
	id         StoreIdentity
	head       headDoc
	headDigest string
}

// ValidateStorePlacement rejects a store that equals, contains or lies
// inside any governed root. Canonical aliases (symlinks) are resolved; the
// comparison is path-boundary based, never a raw lexical prefix.
func ValidateStorePlacement(storePath string, governedRoots ...string) error {
	store, err := resolveForPlacement(storePath)
	if err != nil {
		return err
	}
	for _, root := range governedRoots {
		if root == "" {
			continue
		}
		rel, rerr := resolveForPlacement(root)
		if rerr != nil {
			return rerr
		}
		if rel == store {
			return fmt.Errorf("STORE_ROOT_MISMATCH: store %s overlaps governed root %s (equal)", storePath, root)
		}
		if underPath(store, rel) {
			return fmt.Errorf("STORE_ROOT_MISMATCH: store %s lies inside governed root %s", storePath, root)
		}
		if underPath(rel, store) {
			return fmt.Errorf("STORE_ROOT_MISMATCH: governed root %s lies inside store %s", root, storePath)
		}
	}
	return nil
}

func resolveForPlacement(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("INVALID_SCHEMA: cannot resolve %s: %w", p, err)
	}
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	// the path itself may not exist yet: canonicalize the nearest existing
	// ancestor and rejoin the remainder so aliases cannot hide overlap
	dir, rest := filepath.Split(abs)
	for dir != string(filepath.Separator) && dir != "" {
		if resolved, err := filepath.EvalSymlinks(filepath.Clean(dir)); err == nil {
			return filepath.Join(resolved, rest), nil
		}
		rest = filepath.Join(filepath.Base(filepath.Clean(dir)), rest)
		dir = filepath.Dir(filepath.Clean(dir))
	}
	return abs, nil
}

// underPath reports whether path lies strictly beneath dir with a real path
// boundary (never a lexical prefix like "src2" under "src").
func underPath(path, dir string) bool {
	if path == dir {
		return false
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// validateStoreLayout checks the closed namespace set of the store root.
func validateStoreLayout(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("INVALID_SCHEMA: cannot enumerate store root: %w", err)
	}
	for _, e := range entries {
		switch e.Name() {
		case "store.json":
			if !e.Type().IsRegular() {
				return fmt.Errorf("INVALID_SCHEMA: store.json must be a regular file")
			}
		case storeObjects, storeBlobs, "controls", storeRuns, storeLedger, storeStaging:
			if !e.IsDir() {
				return fmt.Errorf("INVALID_SCHEMA: store entry %s must be a directory", e.Name())
			}
		default:
			return fmt.Errorf("INVALID_SCHEMA: store root holds unknown entry %q; the store contains only store.json, objects/, blobs/, controls/, runs/, ledger/ and transaction staging", e.Name())
		}
	}
	ledgerEntries, err := os.ReadDir(filepath.Join(root, storeLedger))
	if err != nil {
		return fmt.Errorf("INVALID_SCHEMA: cannot enumerate ledger/: %w", err)
	}
	for _, e := range ledgerEntries {
		switch e.Name() {
		case "head.json":
			if !e.Type().IsRegular() {
				return fmt.Errorf("INVALID_SCHEMA: ledger/head.json must be a regular file")
			}
		case storeHeads:
			if !e.IsDir() {
				return fmt.Errorf("INVALID_SCHEMA: ledger/heads must be a directory")
			}
		default:
			return fmt.Errorf("INVALID_SCHEMA: ledger/ holds unknown entry %q", e.Name())
		}
	}
	return nil
}

// validateHeadChain walks the archived chain from the current head back to
// generation zero, verifying archived bytes, digests, generations and the
// initial-empty invariant, and rejecting rollback-below-known-history.
func validateHeadChain(root string, head headDoc, headDigest string) error {
	headsDir := filepath.Join(root, storeLedger, storeHeads)
	archived := func(dig string) ([]byte, bool) {
		raw, err := os.ReadFile(filepath.Join(headsDir, strings.TrimPrefix(dig, "sha256:")+".json"))
		if err != nil {
			return nil, false
		}
		return raw, true
	}
	cur, curDig := head, headDigest
	chain := map[string]bool{headDigest: true}
	for cur.Generation > 0 {
		raw, ok := archived(curDig)
		if !ok || digestOfBytes(raw) != curDig {
			return fmt.Errorf("CONTROL_ROLLBACK: head %s (generation %d) has no matching archived exact-byte copy", curDig, cur.Generation)
		}
		prevRaw, ok := archived(cur.Previous)
		if !ok {
			return fmt.Errorf("CONTROL_ROLLBACK: predecessor head %s of generation %d is not archived; the chain is broken", cur.Previous, cur.Generation)
		}
		prev, err := decodeHeadClosed(prevRaw)
		if err != nil {
			return fmt.Errorf("CONTROL_ROLLBACK: predecessor head %s does not decode: %w", cur.Previous, err)
		}
		if prev.Generation != cur.Generation-1 {
			return fmt.Errorf("CONTROL_ROLLBACK: chain discontinuity: generation %d is preceded by generation %d", cur.Generation, prev.Generation)
		}
		cur, curDig = prev, cur.Previous
		chain[curDig] = true
	}
	raw, ok := archived(curDig)
	if !ok || digestOfBytes(raw) != curDig {
		return fmt.Errorf("CONTROL_ROLLBACK: generation-zero head %s has no matching archived copy", curDig)
	}
	if cur.Previous != "" || len(cur.Plans) > 0 || len(cur.Milestones) > 0 {
		return fmt.Errorf("CONTROL_ROLLBACK: the generation-zero head must be the initial empty head")
	}
	// orphan rule: at most ONE staged successor beyond the current head is
	// tolerated (crash between archive and advance) and it must chain from
	// the current head; anything further ahead or unchained is rollback
	archives, err := os.ReadDir(headsDir)
	if err != nil {
		return fmt.Errorf("INVALID_SCHEMA: cannot enumerate ledger/heads/: %w", err)
	}
	for _, a := range archives {
		name := a.Name()
		if !strings.HasSuffix(name, ".json") || !blobNameRe.MatchString(strings.TrimSuffix(name, ".json")) {
			return fmt.Errorf("INVALID_SCHEMA: ledger/heads/ holds unknown entry %q", name)
		}
		rawArc, rerr := os.ReadFile(filepath.Join(headsDir, name))
		if rerr != nil {
			return rerr
		}
		dig := digestOfBytes(rawArc)
		if dig == headDigest || chain[dig] {
			continue
		}
		arc, derr := decodeHeadClosed(rawArc)
		if derr != nil {
			return fmt.Errorf("CONTROL_ROLLBACK: archived head %s does not decode: %w", dig, derr)
		}
		if arc.Generation == head.Generation+1 {
			if arc.Previous == headDigest {
				continue // recoverable staged successor
			}
			return fmt.Errorf("CONTROL_ROLLBACK: staged successor %s does not chain from the current head", dig)
		}
		if arc.Generation > head.Generation+1 {
			return fmt.Errorf("CONTROL_ROLLBACK: archived head %s is generations ahead of the current head; committed history was rolled back", dig)
		}
		// archives at or below the current generation that are not on the
		// chain are stray history (impossible under the single writer)
		return fmt.Errorf("CONTROL_ROLLBACK: archived head %s (generation %d) is not on the current chain", dig, arc.Generation)
	}
	return nil
}

func digestOfBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// openStore opens and fully validates the store for reading. wantProject ""
// skips the project comparison (export); otherwise a mismatch is blocking.
func openStore(ctx context.Context, path, wantProject string) (*storeView, error) {
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("MISSING_STORE: store %s does not exist; there is no environment fallback", path)
		}
		return nil, fmt.Errorf("MISSING_STORE: store %s: %w", path, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("STORE_ROOT_MISMATCH: store root %s is a symlink; a substituted root is blocking", path)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("STORE_ROOT_MISMATCH: store root %s is not a directory", path)
	}
	if fi.Mode().Perm() != storeRootMod {
		return nil, fmt.Errorf("STORE_ROOT_MISMATCH: store root %s mode is %o, want 0700", path, fi.Mode().Perm())
	}
	dir, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("MISSING_STORE: %w", err)
	}
	st, err := dir.Stat()
	if err != nil || !st.IsDir() || !os.SameFile(fi, st) {
		dir.Close()
		return nil, fmt.Errorf("STORE_ROOT_MISMATCH: store root identity changed while opening %s", path)
	}
	v := &storeView{root: path, dir: dir}
	if err := v.loadIdentity(wantProject); err != nil {
		dir.Close()
		return nil, err
	}
	if err := validateStoreLayout(path); err != nil {
		dir.Close()
		return nil, err
	}
	headRaw, _, err := readControlJSON(filepath.Join(path, storeLedger, "head.json"))
	if err != nil {
		dir.Close()
		return nil, fmt.Errorf("HISTORY_UNAVAILABLE: %w", err)
	}
	head, err := decodeHeadClosed(headRaw)
	if err != nil {
		dir.Close()
		return nil, err
	}
	if head.ProjectID != v.id.ProjectID {
		dir.Close()
		return nil, fmt.Errorf("STORE_ROOT_MISMATCH: head project %s does not match store identity %s", head.ProjectID, v.id.ProjectID)
	}
	v.head = head
	v.headDigest = digestOfBytes(headRaw)
	if err := validateHeadChain(path, head, v.headDigest); err != nil {
		dir.Close()
		return nil, err
	}
	return v, nil
}

func (v *storeView) loadIdentity(wantProject string) error {
	raw, _, err := readControlJSON(filepath.Join(v.root, "store.json"))
	if err != nil {
		return err
	}
	_, doc, err := readControlJSONFromBytes("store.json", raw)
	if err != nil {
		return err
	}
	root, err := asObject(doc, "store.json")
	if err != nil {
		return err
	}
	if err := exactKeys(root, "store.json", "schema", "store_id", "project_id"); err != nil {
		return err
	}
	schema, _ := reqStr(root, "schema", "store.json.schema")
	if schema != SchemaStore {
		return fmt.Errorf("UNSUPPORTED_VERSION: store schema %q is not %s", schema, SchemaStore)
	}
	sid, err := reqStr(root, "store_id", "store.json.store_id")
	if err != nil {
		return err
	}
	pid, err := reqStr(root, "project_id", "store.json.project_id")
	if err != nil {
		return err
	}
	if !validUUID(sid) || !validUUID(pid) {
		return fmt.Errorf("INVALID_SCHEMA: store identity fields must be UUIDs")
	}
	if string(encodeStoreIdentity(sid, pid)) != string(raw) {
		return fmt.Errorf("INVALID_SCHEMA: store.json bytes are not the canonical encoding")
	}
	if wantProject != "" && wantProject != pid {
		return fmt.Errorf("STORE_ROOT_MISMATCH: store project %s does not match the required project %s", pid, wantProject)
	}
	v.id = StoreIdentity{StoreID: sid, ProjectID: pid}
	return nil
}

func encodeStoreIdentity(storeID, projectID string) []byte {
	return []byte(canonObj(map[string]string{
		"project_id": canonString(projectID), "schema": canonString(SchemaStore),
		"store_id": canonString(storeID),
	}))
}

// close re-verifies the retained root identity before releasing it.
func (v *storeView) close() error {
	if v.dir == nil {
		return nil
	}
	st, err := v.dir.Stat()
	if err != nil || !st.IsDir() || st.Mode().Perm() != storeRootMod {
		v.dir.Close()
		v.dir = nil
		return fmt.Errorf("CUSTODY_ERROR: store root identity drifted before close")
	}
	fi, err := os.Lstat(v.root)
	if err != nil || !os.SameFile(fi, st) {
		v.dir.Close()
		v.dir = nil
		return fmt.Errorf("CUSTODY_ERROR: store root was substituted before close")
	}
	err = v.dir.Close()
	v.dir = nil
	return err
}

// readControlJSONFromBytes decodes closed JSON metadata from bytes already
// read (16 MiB cap enforced by the caller of readControlJSON).
func readControlJSONFromBytes(name string, data []byte) ([]byte, *ir.Value, error) {
	v, err := ir.LoadMachineJSONBytes(name, data)
	if err != nil {
		return nil, nil, fmt.Errorf("INVALID_SCHEMA: %s: %w", name, err)
	}
	return data, v, nil
}

// ---- initialization ----

// InitStore atomically creates a new external 0700 store: closed identity,
// durable generation-zero head, and only its owned root. An existing path
// is never adopted; failure leaves no initialized-success result.
func InitStore(ctx context.Context, path, projectID string) (StoreInitResult, error) {
	ctx, cancel := nonExecContext(ctx)
	defer cancel()
	if err := checkCtx(ctx); err != nil {
		return StoreInitResult{}, err
	}
	if !validUUID(projectID) {
		return StoreInitResult{}, fmt.Errorf("INVALID_SCHEMA: project %q is not a UUID", projectID)
	}
	if fi, err := os.Lstat(path); err == nil {
		_ = fi
		return StoreInitResult{}, fmt.Errorf("STORE_ROOT_MISMATCH: store path %s already exists; init requires a new path and never adopts an existing, partial or empty destination", path)
	} else if !os.IsNotExist(err) {
		return StoreInitResult{}, fmt.Errorf("STORE_ROOT_MISMATCH: cannot inspect store path %s: %w", path, err)
	}
	storeID, err := newUUID()
	if err != nil {
		return StoreInitResult{}, fmt.Errorf("CUSTODY_ERROR: minting store id: %w", err)
	}
	fail := func(err error) (StoreInitResult, error) {
		cctx, ccancel := cleanupContext()
		defer ccancel()
		if rmErr := os.RemoveAll(path); rmErr != nil {
			return StoreInitResult{}, fmt.Errorf("%w; cleanup also failed: %w", err, rmErr)
		}
		_ = cctx
		return StoreInitResult{}, err
	}
	if err := os.Mkdir(path, storeRootMod); err != nil {
		return StoreInitResult{}, fmt.Errorf("CUSTODY_ERROR: creating store root: %w", err)
	}
	for _, ns := range []string{storeObjects, storeBlobs, "controls", storeRuns, storeLedger, filepath.Join(storeLedger, storeHeads)} {
		if err := os.Mkdir(filepath.Join(path, filepath.FromSlash(ns)), storeRootMod); err != nil {
			return fail(fmt.Errorf("CUSTODY_ERROR: creating %s: %w", ns, err))
		}
	}
	if err := durableWriteBytes(filepath.Join(path, "store.json"), storeFileMod, encodeStoreIdentity(storeID, projectID)); err != nil {
		return fail(fmt.Errorf("CUSTODY_ERROR: writing store.json: %w", err))
	}
	headBytes := encodeHeadCanonical(headDoc{ProjectID: projectID, Generation: 0})
	if err := publishImmutableFile(filepath.Join(path, storeLedger, storeHeads, digestHexBytes(headBytes)+".json"), storeFileMod, headBytes); err != nil {
		return fail(fmt.Errorf("CUSTODY_ERROR: archiving the generation-zero head: %w", err))
	}
	if err := publishImmutableFile(filepath.Join(path, storeLedger, "head.json"), storeFileMod, headBytes); err != nil {
		return fail(fmt.Errorf("CUSTODY_ERROR: advancing head.json: %w", err))
	}
	for _, d := range []string{filepath.Join(path, storeLedger, storeHeads), filepath.Join(path, storeLedger), path} {
		if err := fsyncDir(d); err != nil {
			return fail(fmt.Errorf("CUSTODY_ERROR: fsync %s: %w", d, err))
		}
	}
	if err := checkCtx(ctx); err != nil {
		return fail(err)
	}
	return StoreInitResult{StoreID: storeID, ProjectID: projectID, HeadDigest: digestOfBytes(headBytes), Generation: 0}, nil
}

func digestHexBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ---- transport ----

type exportEntry struct {
	path   string
	kind   string
	digest string
	size   int64
	data   []byte
}

// ExportStore writes a NEW bounded closed archive (magic MTDDARCV, version
// 1, U64-length + payload + sha256 records, canonical index first) holding
// the complete object/control/head/run history with per-record checksums.
func ExportStore(ctx context.Context, storePath, outPath string) (StoreExportResult, error) {
	ctx, cancel := nonExecContext(ctx)
	defer cancel()
	if err := checkCtx(ctx); err != nil {
		return StoreExportResult{}, err
	}
	if fi, err := os.Lstat(outPath); err == nil {
		_ = fi
		return StoreExportResult{}, fmt.Errorf("STORE_ROOT_MISMATCH: archive output %s already exists; export writes only a new path", outPath)
	} else if !os.IsNotExist(err) {
		return StoreExportResult{}, fmt.Errorf("STORE_ROOT_MISMATCH: cannot inspect archive output %s: %w", outPath, err)
	}
	v, err := openStore(ctx, storePath, "")
	if err != nil {
		return StoreExportResult{}, err
	}
	defer v.close()
	// transaction staging must be quiescent for a complete export
	if _, err := os.Lstat(filepath.Join(storePath, storeStaging)); err == nil {
		return StoreExportResult{}, fmt.Errorf("INVALID_SCHEMA: store holds live transaction staging; export requires a quiescent store")
	}
	entries, err := collectStoreEntries(storePath)
	if err != nil {
		return StoreExportResult{}, err
	}
	if err := checkCtx(ctx); err != nil {
		return StoreExportResult{}, err
	}
	index := renderExportIndex(v.id, v.headDigest, v.head.Generation, entries)
	var total int64
	for _, e := range entries {
		total += e.size
	}
	if len(entries) > int(protocol.LimitEntriesMax) || total > protocol.LimitBundleMaxBytes {
		return StoreExportResult{}, fmt.Errorf("OUTPUT_LIMIT: store history exceeds the export bounds (%d entries, %d bytes)", len(entries), total)
	}
	if err := durableWriteBytes(outPath, storeFileMod, renderArchive(index, entries)); err != nil {
		os.Remove(outPath)
		return StoreExportResult{}, fmt.Errorf("CUSTODY_ERROR: writing archive: %w", err)
	}
	return StoreExportResult{
		StoreID: v.id.StoreID, ProjectID: v.id.ProjectID, HeadDigest: v.headDigest,
		Generation: v.head.Generation, Entries: int64(len(entries)), TotalBytes: total, Archive: outPath,
	}, nil
}

// collectStoreEntries walks the six closed namespaces and read-verifies the
// content digests of every archived byte.
func collectStoreEntries(storePath string) ([]exportEntry, error) {
	var entries []exportEntry
	add := func(rel, kind string) error {
		raw, err := os.ReadFile(filepath.Join(storePath, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("INVALID_SCHEMA: cannot read %s: %w", rel, err)
		}
		entries = append(entries, exportEntry{path: rel, kind: kind, digest: digestOfBytes(raw), size: int64(len(raw)), data: raw})
		return nil
	}
	if err := add("store.json", "store"); err != nil {
		return nil, err
	}
	dirs := []struct {
		dir  string
		kind string
	}{
		{storeBlobs, "blob"}, {"controls", "control"}, {storeRuns, "run"},
		{filepath.Join(storeLedger, storeHeads), "head-archive"},
	}
	for _, d := range dirs {
		files, err := os.ReadDir(filepath.Join(storePath, filepath.FromSlash(d.dir)))
		if err != nil {
			return nil, fmt.Errorf("INVALID_SCHEMA: cannot enumerate %s: %w", d.dir, err)
		}
		for _, f := range files {
			if !f.Type().IsRegular() {
				return nil, fmt.Errorf("INVALID_SCHEMA: %s holds non-regular entry %q", d.dir, f.Name())
			}
			switch d.kind {
			case "blob":
				if !blobNameRe.MatchString(f.Name()) {
					return nil, fmt.Errorf("INVALID_SCHEMA: %s holds unknown entry %q", d.dir, f.Name())
				}
			case "control", "head-archive":
				if !blobNameRe.MatchString(strings.TrimSuffix(f.Name(), ".json")) || !strings.HasSuffix(f.Name(), ".json") {
					return nil, fmt.Errorf("INVALID_SCHEMA: %s holds unknown entry %q", d.dir, f.Name())
				}
			case "run":
				if !runNameRe.MatchString(f.Name()) {
					return nil, fmt.Errorf("INVALID_SCHEMA: runs/ holds unknown entry %q", f.Name())
				}
			}
			if err := add(filepath.ToSlash(filepath.Join(d.dir, f.Name())), d.kind); err != nil {
				return nil, err
			}
		}
	}
	objDirs, err := os.ReadDir(filepath.Join(storePath, storeObjects))
	if err != nil {
		return nil, fmt.Errorf("INVALID_SCHEMA: cannot enumerate objects/: %w", err)
	}
	for _, o := range objDirs {
		if !o.IsDir() || !blobNameRe.MatchString(o.Name()) {
			return nil, fmt.Errorf("INVALID_SCHEMA: objects/ holds unknown entry %q", o.Name())
		}
		objFiles, err := os.ReadDir(filepath.Join(storePath, storeObjects, o.Name()))
		if err != nil || len(objFiles) != 1 || objFiles[0].Name() != "bundle.json" || !objFiles[0].Type().IsRegular() {
			return nil, fmt.Errorf("INVALID_SCHEMA: object %s must hold exactly bundle.json", o.Name())
		}
		if err := add(filepath.ToSlash(filepath.Join(storeObjects, o.Name(), "bundle.json")), "object"); err != nil {
			return nil, err
		}
	}
	if err := add(filepath.Join(storeLedger, "head.json"), "head"); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	for i := 1; i < len(entries); i++ {
		if entries[i-1].path == entries[i].path {
			return nil, fmt.Errorf("INVALID_SCHEMA: duplicate store path %q", entries[i].path)
		}
	}
	return entries, nil
}

func renderExportIndex(id StoreIdentity, headDigest string, generation int64, entries []exportEntry) []byte {
	members := make([]string, 0, len(entries))
	for _, e := range entries {
		members = append(members, canonObj(map[string]string{
			"digest": canonString(e.digest), "kind": canonString(e.kind),
			"path": canonString(e.path), "size": canonInt(e.size),
		}))
	}
	return []byte(canonObj(map[string]string{
		"entries":     canonArr(members),
		"generation":  canonInt(generation),
		"head_digest": canonString(headDigest),
		"project_id":  canonString(id.ProjectID),
		"schema":      canonString(SchemaExport),
		"store_id":    canonString(id.StoreID),
		"total_bytes": canonInt(func() int64 {
			var t int64
			for _, e := range entries {
				t += e.size
			}
			return t
		}()),
	}))
}

func renderArchive(index []byte, entries []exportEntry) []byte {
	var out []byte
	out = append(out, archiveExportMagic()...)
	frame := func(payload []byte) {
		var head [8]byte
		binary.BigEndian.PutUint64(head[:], uint64(len(payload)))
		out = append(out, head[:]...)
		out = append(out, payload...)
		sum := sha256.Sum256(payload)
		out = append(out, sum[:]...)
	}
	frame(index)
	for _, e := range entries {
		frame(e.data)
	}
	return out
}

func archiveExportMagic() []byte { return []byte{'M', 'T', 'D', 'D', 'A', 'R', 'C', 'V', 0x01} }

// readArchiveFrame reads one U64-length + payload + checksum record.
func readArchiveFrame(data []byte, off int64) (payload []byte, next int64, err error) {
	if off+9 > int64(len(data)) {
		return nil, 0, fmt.Errorf("INVALID_SCHEMA: truncated record header")
	}
	n := int64(binary.BigEndian.Uint64(data[off : off+8]))
	off += 8
	if n < 0 || off+n+32 > int64(len(data)) {
		return nil, 0, fmt.Errorf("INVALID_SCHEMA: truncated record body")
	}
	payload = data[off : off+n]
	off += n
	sum := sha256.Sum256(payload)
	if string(data[off:off+32]) != string(sum[:]) {
		return nil, 0, fmt.Errorf("INVALID_SCHEMA: record checksum mismatch")
	}
	return payload, off + 32, nil
}

type importIndexEntry struct {
	path   string
	kind   string
	digest string
	size   int64
}

// ImportStore validates a closed archive and atomically publishes it into a
// NEW destination under the caller-supplied expected exported head digest.
// Imported evidence is recorded-only; import never certifies replay.
func ImportStore(ctx context.Context, destPath, archivePath, projectID, expectedHead string) (StoreImportResult, error) {
	ctx, cancel := nonExecContext(ctx)
	defer cancel()
	if err := checkCtx(ctx); err != nil {
		return StoreImportResult{}, err
	}
	if !validUUID(projectID) {
		return StoreImportResult{}, fmt.Errorf("INVALID_SCHEMA: project %q is not a UUID", projectID)
	}
	if !validDigest(expectedHead) {
		return StoreImportResult{}, fmt.Errorf("INVALID_SCHEMA: expected head %q is not a digest; there is no wildcard comparison", expectedHead)
	}
	if fi, err := os.Lstat(destPath); err == nil {
		_ = fi
		return StoreImportResult{}, fmt.Errorf("STORE_ROOT_MISMATCH: destination %s already exists; import requires a new destination", destPath)
	} else if !os.IsNotExist(err) {
		return StoreImportResult{}, fmt.Errorf("STORE_ROOT_MISMATCH: cannot inspect destination %s: %w", destPath, err)
	}
	fi, err := os.Lstat(archivePath)
	if err != nil {
		return StoreImportResult{}, fmt.Errorf("INVALID_SCHEMA: cannot read archive %s: %w", archivePath, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		return StoreImportResult{}, fmt.Errorf("INVALID_SCHEMA: archive %s must be a regular non-symlink file", archivePath)
	}
	if fi.Size() > protocol.LimitBundleMaxBytes {
		return StoreImportResult{}, fmt.Errorf("OUTPUT_LIMIT: archive exceeds the %d-byte bundle bound", int64(protocol.LimitBundleMaxBytes))
	}
	data, err := os.ReadFile(archivePath)
	if err != nil {
		return StoreImportResult{}, fmt.Errorf("INVALID_SCHEMA: %w", err)
	}
	if string(data[:len(archiveExportMagic())]) != string(archiveExportMagic()) {
		return StoreImportResult{}, fmt.Errorf("INVALID_SCHEMA: archive magic is not %s version 1", "MTDDARCV")
	}
	off := int64(len(archiveExportMagic()))
	indexRaw, off, err := readArchiveFrame(data, off)
	if err != nil {
		return StoreImportResult{}, err
	}
	if int64(len(indexRaw)) > protocol.ControlInputMaxBytes {
		return StoreImportResult{}, fmt.Errorf("OUTPUT_LIMIT: archive index exceeds the %d-byte control-input cap", int64(protocol.ControlInputMaxBytes))
	}
	idxEntries, id, headDigest, generation, _, err := decodeExportIndex(indexRaw)
	if err != nil {
		return StoreImportResult{}, err
	}
	if id.ProjectID != projectID {
		return StoreImportResult{}, fmt.Errorf("STORE_ROOT_MISMATCH: archive project %s does not match the requested project %s", id.ProjectID, projectID)
	}
	if headDigest != expectedHead {
		return StoreImportResult{}, fmt.Errorf("CONTROL_ROLLBACK: archive head %s does not match the caller-supplied expected head %s; an unauthenticated local archive cannot prove global newest state", headDigest, expectedHead)
	}
	// staging publication: build the complete validated store beside dest
	nonce := make([]byte, 8)
	rand.Read(nonce)
	staging := fmt.Sprintf("%s.staging-%s", destPath, hex.EncodeToString(nonce))
	rollback := func(err error) (StoreImportResult, error) {
		cctx, ccancel := cleanupContext()
		defer ccancel()
		_ = cctx
		if rmErr := os.RemoveAll(staging); rmErr != nil {
			return StoreImportResult{}, fmt.Errorf("%w; staging cleanup also failed: %w", err, rmErr)
		}
		return StoreImportResult{}, err
	}
	if err := buildStagedStore(staging, id, idxEntries, func(want importIndexEntry) ([]byte, error) {
		if off >= int64(len(data)) {
			return nil, fmt.Errorf("INVALID_SCHEMA: archive ended before entry %s", want.path)
		}
		var payload []byte
		var err error
		payload, off, err = readArchiveFrame(data, off)
		if err != nil {
			return nil, err
		}
		if digestOfBytes(payload) != want.digest || int64(len(payload)) != want.size {
			return nil, fmt.Errorf("INVALID_SCHEMA: record %s does not match its index digest/size", want.path)
		}
		return payload, nil
	}); err != nil {
		return rollback(err)
	}
	if off != int64(len(data)) {
		return rollback(fmt.Errorf("INVALID_SCHEMA: archive carries trailing data after the declared entries"))
	}
	if err := checkCtx(ctx); err != nil {
		return rollback(err)
	}
	// full validation of the staged store before publication
	sv, err := openStore(ctx, staging, projectID)
	if err != nil {
		return rollback(err)
	}
	if sv.headDigest != headDigest || sv.head.Generation != generation {
		sv.close()
		return rollback(fmt.Errorf("CONTROL_ROLLBACK: staged store head %s (generation %d) does not match the archive index", sv.headDigest, sv.head.Generation))
	}
	if err := verifyStagedGraph(staging); err != nil {
		sv.close()
		return rollback(err)
	}
	if err := sv.close(); err != nil {
		return rollback(err)
	}
	if err := os.Rename(staging, destPath); err != nil {
		return rollback(fmt.Errorf("CUSTODY_ERROR: atomic publication: %w", err))
	}
	if err := fsyncDir(filepath.Dir(destPath)); err != nil {
		return StoreImportResult{}, fmt.Errorf("CUSTODY_ERROR: fsync destination parent: %w", err)
	}
	counts := map[string]int{}
	for _, e := range idxEntries {
		counts[e.kind]++
	}
	return StoreImportResult{
		StoreID: id.StoreID, ProjectID: id.ProjectID, HeadDigest: headDigest,
		Generation: generation, Blobs: counts["blob"], Objects: counts["object"],
		Controls: counts["control"], Heads: counts["head"] + counts["head-archive"],
		Runs: counts["run"], State: "imported-recorded-only",
	}, nil
}

// decodeExportIndex decodes the closed index document.
func decodeExportIndex(raw []byte) (entries []importIndexEntry, id StoreIdentity, headDigest string, generation, total int64, err error) {
	_, doc, derr := readControlJSONFromBytes("export-index", raw)
	if derr != nil {
		return nil, id, "", 0, 0, derr
	}
	root, err2 := asObject(doc, "export index")
	if err2 != nil {
		return nil, id, "", 0, 0, err2
	}
	if err := exactKeys(root, "export index", "schema", "store_id", "project_id", "head_digest", "generation", "total_bytes", "entries"); err != nil {
		return nil, id, "", 0, 0, err
	}
	schema, _ := reqStr(root, "schema", "export index.schema")
	if schema != SchemaExport {
		return nil, id, "", 0, 0, fmt.Errorf("UNSUPPORTED_VERSION: archive schema %q is not %s", schema, SchemaExport)
	}
	id.StoreID, err = reqStr(root, "store_id", "export index.store_id")
	if err != nil {
		return nil, id, "", 0, 0, err
	}
	id.ProjectID, err = reqStr(root, "project_id", "export index.project_id")
	if err != nil {
		return nil, id, "", 0, 0, err
	}
	if !validUUID(id.StoreID) || !validUUID(id.ProjectID) {
		return nil, id, "", 0, 0, fmt.Errorf("INVALID_SCHEMA: archive identity fields must be UUIDs")
	}
	headDigest, err = reqStr(root, "head_digest", "export index.head_digest")
	if err != nil {
		return nil, id, "", 0, 0, err
	}
	if !validDigest(headDigest) {
		return nil, id, "", 0, 0, fmt.Errorf("INVALID_SCHEMA: archive head digest %q is invalid", headDigest)
	}
	generation, err = reqInt(root, "generation", "export index.generation")
	if err != nil || generation < 0 {
		return nil, id, "", 0, 0, fmt.Errorf("INVALID_SCHEMA: archive generation must be a nonnegative integer")
	}
	total, err = reqInt(root, "total_bytes", "export index.total_bytes")
	if err != nil || total < 0 {
		return nil, id, "", 0, 0, fmt.Errorf("INVALID_SCHEMA: archive total_bytes must be nonnegative")
	}
	eObjs, err := reqObjArray(root, "entries", "export index.entries")
	if err != nil {
		return nil, id, "", 0, 0, err
	}
	seen := map[string]bool{}
	var running int64
	for i, eo := range eObjs {
		where := fmt.Sprintf("export index.entries[%d]", i)
		if err := exactKeys(eo, where, "path", "kind", "digest", "size"); err != nil {
			return nil, id, "", 0, 0, err
		}
		var e importIndexEntry
		e.path, err = reqStr(eo, "path", where+".path")
		if err != nil {
			return nil, id, "", 0, 0, err
		}
		e.kind, err = reqStr(eo, "kind", where+".kind")
		if err != nil {
			return nil, id, "", 0, 0, err
		}
		e.digest, err = reqStr(eo, "digest", where+".digest")
		if err != nil {
			return nil, id, "", 0, 0, err
		}
		if !validDigest(e.digest) {
			return nil, id, "", 0, 0, fmt.Errorf("INVALID_SCHEMA: %s.digest %q is not a digest", where, e.digest)
		}
		e.size, err = reqInt(eo, "size", where+".size")
		if err != nil || e.size <= 0 {
			return nil, id, "", 0, 0, fmt.Errorf("INVALID_SCHEMA: %s.size must be a positive integer", where)
		}
		if e.size > protocol.LimitBundleMaxBytes {
			return nil, id, "", 0, 0, fmt.Errorf("OUTPUT_LIMIT: %s declares size %d beyond the bundle bound", where, e.size)
		}
		if err := validateArchivePath(e.path, e.kind); err != nil {
			return nil, id, "", 0, 0, fmt.Errorf("INVALID_SCHEMA: %s: %w", where, err)
		}
		if seen[e.path] {
			return nil, id, "", 0, 0, fmt.Errorf("INVALID_SCHEMA: %s: duplicate path %q", where, e.path)
		}
		seen[e.path] = true
		running += e.size
		if running > protocol.LimitBundleMaxBytes || running > total {
			return nil, id, "", 0, 0, fmt.Errorf("OUTPUT_LIMIT: archive entries exceed the declared/bounded total")
		}
		entries = append(entries, e)
	}
	if running != total {
		return nil, id, "", 0, 0, fmt.Errorf("INVALID_SCHEMA: entry sizes sum to %d, index declares %d", running, total)
	}
	for i := 1; i < len(entries); i++ {
		if entries[i-1].path >= entries[i].path {
			return nil, id, "", 0, 0, fmt.Errorf("INVALID_SCHEMA: index entries are not canonically sorted")
		}
	}
	return entries, id, headDigest, generation, total, nil
}

// validateArchivePath enforces the closed per-kind store-relative layout.
func validateArchivePath(p, kind string) error {
	if err := validatePath(p); err != nil {
		return err
	}
	dir := path_Dir(p)
	base := path_Base(p)
	switch kind {
	case "store":
		if p != "store.json" {
			return fmt.Errorf("kind store must be exactly store.json")
		}
	case "blob":
		if dir != storeBlobs || !blobNameRe.MatchString(base) {
			return fmt.Errorf("kind blob must be blobs/<64 hex>")
		}
	case "control":
		if dir != "controls" || !blobNameRe.MatchString(strings.TrimSuffix(base, ".json")) || !strings.HasSuffix(base, ".json") {
			return fmt.Errorf("kind control must be controls/<64 hex>.json")
		}
	case "object":
		objDir := strings.SplitN(dir, "/", 2)
		if len(objDir) != 2 || objDir[0] != storeObjects || !blobNameRe.MatchString(objDir[1]) || base != "bundle.json" {
			return fmt.Errorf("kind object must be objects/<64 hex>/bundle.json")
		}
	case "head":
		if p != "ledger/head.json" {
			return fmt.Errorf("kind head must be exactly ledger/head.json")
		}
	case "head-archive":
		if dir != "ledger/heads" || !blobNameRe.MatchString(strings.TrimSuffix(base, ".json")) || !strings.HasSuffix(base, ".json") {
			return fmt.Errorf("kind head-archive must be ledger/heads/<64 hex>.json")
		}
	case "run":
		if dir != storeRuns || !runNameRe.MatchString(base) {
			return fmt.Errorf("kind run must be runs/<run id>.json")
		}
	default:
		return fmt.Errorf("unknown archive entry kind %q", kind)
	}
	return nil
}

// path_Dir/path_Base are slash-path helpers independent of filepath.
func path_Dir(p string) string {
	i := strings.LastIndexByte(p, '/')
	if i < 0 {
		return ""
	}
	return p[:i]
}

func path_Base(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}

// buildStagedStore materializes the archive into a staging root, pulling
// each record through the supplied reader in index order.
func buildStagedStore(staging string, id StoreIdentity, entries []importIndexEntry, next func(want importIndexEntry) ([]byte, error)) error {
	if err := os.Mkdir(staging, storeRootMod); err != nil {
		return fmt.Errorf("CUSTODY_ERROR: creating staging root: %w", err)
	}
	for _, ns := range []string{storeObjects, storeBlobs, "controls", storeRuns, filepath.Join(storeLedger, storeHeads)} {
		if err := os.MkdirAll(filepath.Join(staging, filepath.FromSlash(ns)), storeRootMod); err != nil {
			return fmt.Errorf("CUSTODY_ERROR: creating staging %s: %w", ns, err)
		}
	}
	for _, e := range entries {
		payload, err := next(e)
		if err != nil {
			return err
		}
		target := filepath.Join(staging, filepath.FromSlash(e.path))
		if parent := filepath.Dir(target); parent != staging {
			if err := os.MkdirAll(parent, storeRootMod); err != nil {
				return fmt.Errorf("CUSTODY_ERROR: creating staging parent of %s: %w", e.path, err)
			}
		}
		perm := os.FileMode(storeFileMod)
		if e.kind == "blob" {
			perm = storeBlobMod
		}
		if err := publishImmutableFile(target, perm, payload); err != nil {
			return err
		}
		if e.kind == "blob" && digestHexBytes(payload) != path_Base(e.path) {
			return fmt.Errorf("INVALID_SCHEMA: blob %s content does not hash to its content-addressed name", e.path)
		}
	}
	return nil
}

// verifyStagedGraph deep-validates every object bundle against the staged
// blobs: closed bundle documents, recomputed tree digests and read-verified
// referenced content.
func verifyStagedGraph(staging string) error {
	objDirs, err := os.ReadDir(filepath.Join(staging, storeObjects))
	if err != nil {
		return err
	}
	for _, o := range objDirs {
		bundlePath := filepath.Join(staging, storeObjects, o.Name(), "bundle.json")
		raw, _, err := readControlJSON(bundlePath)
		if err != nil {
			return err
		}
		entries, treeDigest, err := decodeBundleClosed(raw)
		if err != nil {
			return err
		}
		if treeDigest != "sha256:"+o.Name() {
			return fmt.Errorf("INVALID_SCHEMA: object %s bundle tree digest %s does not match its address", o.Name(), treeDigest)
		}
		for _, e := range entries {
			if e.Kind != "file" {
				continue
			}
			blobPath := filepath.Join(staging, storeBlobs, strings.TrimPrefix(e.Digest, "sha256:"))
			data, err := os.ReadFile(blobPath)
			if err != nil {
				return fmt.Errorf("INVALID_SCHEMA: object %s references blob %s which is not in the archive", o.Name(), e.Digest)
			}
			if digestOfBytes(data) != e.Digest {
				return fmt.Errorf("INVALID_SCHEMA: blob %s does not hash to its address", e.Digest)
			}
		}
	}
	return nil
}
