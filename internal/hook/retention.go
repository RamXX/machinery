package hook

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RamXX/machinery/internal/dirscan"
	"github.com/RamXX/machinery/internal/filelock"
)

// The per-user hook state store is shared by every project this machine
// governs. Without retention it only ever grows: one ledger plus one route
// snapshot per project root that was armed and never discharged, and a heavy
// local sweep arms thousands of ephemeral roots. Past hookStateDirMaxEntries
// every bounded inventory fails, so the hook fails closed on its own state and
// takes every shell and write tool in every repository with it.
//
// Retention therefore has two constant bounds, both enforced on the arming
// write path and both far below the fail-closed ceiling:
//
//   - hookStateRouteRetention bounds one project's route snapshots.
//   - hookStateDirRetentionCeiling bounds the whole store.
//
// Bytes follow from entries: every retained file already has its own byte
// ceiling (hookStateMaxBytes, hookRouteMaxBytes), so entries times per-file
// ceiling bounds the store's size as well as its count.
const (
	// hookStateRouteRetention is how many route snapshots one project root
	// keeps, newest first. A route snapshot is the pre-event routing recovery
	// aid for one host session; the obligation itself lives in the ledger,
	// whose route digests bind the stop-time decision and are never pruned.
	hookStateRouteRetention = 8

	// hookStateDirRetentionCeiling is the store-wide entry count at which the
	// arming write path compacts. An eighth of the fail-closed limit leaves
	// seven eighths of headroom for live projects between two compactions,
	// and still holds hundreds of unfinished project obligations.
	hookStateDirRetentionCeiling = hookStateDirMaxEntries / 8

	// hookStateDirRetentionTarget is how far below the ceiling one compaction
	// pass aims. Compacting to the ceiling itself would compact again on the
	// very next write; compacting to half of it amortizes one pass over the
	// next hundred-odd armed obligations.
	hookStateDirRetentionTarget = hookStateDirRetentionCeiling / 2

	// hookStateDirRepairMaxEntries bounds a repair-mode enumeration: the store
	// is already over hookStateDirMaxEntries, so compaction must be able to
	// see it in order to reclaim it. It stays bounded; an unbounded read is
	// how a runaway directory becomes a runaway process.
	hookStateDirRepairMaxEntries = 1 << 20

	// hookStateReclaimBatch is how many generations one pass removes while
	// holding the store-wide reclamation lock. It bounds how long a governed
	// event in an unrelated repository can wait behind housekeeping.
	hookStateReclaimBatch = 64

	// hookStateRouteReclaimBudget bounds how many surplus route snapshots one
	// write reclaims. Surplus was unbounded before this policy existed, so the
	// first governed event after an upgrade can meet hundreds of them; the
	// budget converges the project over the next few events rather than paying
	// for all of them inside one tool call.
	hookStateRouteReclaimBudget = 16

	// hookStateRouteReclaimBatch is how many of those one store-wide lock hold
	// covers. Route snapshots are removed through the quarantined path, which
	// costs tens of milliseconds each, so the hold stays far inside the
	// hookStateLockWaitLimit an enumeration elsewhere is waiting on.
	hookStateRouteReclaimBatch = 8
)

// hookStateRepairDepth raises the store enumeration ceiling while compaction
// owns the store. Nested inventories taken by compaction itself (interrupted
// deletion recovery, per-project temps) must see the same oversized directory
// compaction is repairing instead of failing closed on it.
var hookStateRepairDepth atomic.Int32

var (
	hookStateCompactionMu sync.Mutex
	// hookStateAutoCompacted keeps one process to a single automatic
	// compaction attempt. A hook process handles one event; retrying the full
	// repair pass for every inventory inside that event would turn a bounded
	// recovery into an unbounded one.
	hookStateAutoCompacted bool
	// hookStateBoundExhausted records that a write-path compaction could not
	// reach the target. The store is then full of obligations for project
	// roots that still exist, and rescanning it on every later write in this
	// process would cost the store's whole size per armed tool call.
	hookStateBoundExhausted bool
)

// errHookStateGenerationRetained marks a generation compaction deliberately
// left alone: its project root still exists, its ledger records no root, or
// another process holds its lock. It is a skip, never a failure.
var errHookStateGenerationRetained = errors.New("hook state generation retained")

var (
	heldHookStateLocksMu sync.Mutex
	heldHookStateLocks   = map[string]int{}
)

// noteHookStateLockHeld records that this process holds one project's state
// lock, so reclamation can recognize it and leave that generation alone.
func noteHookStateLockHeld(scope string) {
	heldHookStateLocksMu.Lock()
	defer heldHookStateLocksMu.Unlock()
	heldHookStateLocks[scope]++
}

func noteHookStateLockReleased(scope string) {
	heldHookStateLocksMu.Lock()
	defer heldHookStateLocksMu.Unlock()
	if heldHookStateLocks[scope] <= 1 {
		delete(heldHookStateLocks, scope)
		return
	}
	heldHookStateLocks[scope]--
}

func hookStateLockHeldHere(scope string) bool {
	heldHookStateLocksMu.Lock()
	defer heldHookStateLocksMu.Unlock()
	return heldHookStateLocks[scope] > 0
}

// readHookStateDir enumerates the store under the fail-closed entry limit and,
// when that limit refuses the read, compacts the store once and retries. A
// store filled by an older version therefore heals itself instead of bricking
// every governed tool until a human prunes it by hand.
func readHookStateDir(dir string) ([]os.DirEntry, error) {
	entries, err := readHookStateDirLocked(dir)
	if err == nil || !errors.Is(err, dirscan.ErrTooManyEntries) {
		return entries, err
	}
	// Compaction already owns the store on this call path, so its own nested
	// inventory must not start a second pass: that pass would block on the
	// store lock this process is holding until the lock-wait budget expires.
	// Only reachable above the repair ceiling, which is a store no filesystem
	// this hook governs will produce, and cheaper to exclude than to reason
	// about.
	if hookStateRepairDepth.Load() > 0 || !beginHookStateAutoCompaction() {
		return nil, hookStateOverLimit(dir, err)
	}
	if _, compactErr := compactHookStateDir(dir, hookStateDirRetentionTarget); compactErr != nil {
		return nil, errors.Join(hookStateOverLimit(dir, err), compactErr)
	}
	entries, err = readHookStateDirLocked(dir)
	if errors.Is(err, dirscan.ErrTooManyEntries) {
		return nil, hookStateOverLimit(dir, err)
	}
	return entries, err
}

// readHookStateDirLocked enumerates the store while excluding bulk
// reclamation. An ordinary arming write publishes one file and a bounded
// enumeration retry absorbs it, but a compaction pass removes many files in a
// row and would otherwise outlast every retry, turning housekeeping into a
// denied tool call in an unrelated repository. Compaction itself already owns
// the store, so it reads without reacquiring rather than blocking on the
// exclusive lock it is holding.
//
// The repair-depth flag is process-wide, which is exact for a hook: one
// process handles one event. A test binary that runs several events as
// goroutines shares it, so one goroutine's compaction suppresses another's
// shared acquisition; cross-process exclusion is therefore proved with a real
// child process, never with goroutines.
func readHookStateDirLocked(dir string) (entries []os.DirEntry, retErr error) {
	if hookStateRepairDepth.Load() > 0 {
		return dirscan.Read(dir, hookStateDirEntryCeiling())
	}
	ctx, cancel := context.WithTimeout(context.Background(), hookStateLockWaitLimit)
	defer cancel()
	lock, err := filelock.AcquireSharedWaitContext(ctx, hookStateStoreScope(dir))
	if err != nil {
		return nil, fmt.Errorf("hook state store %s remained under compaction within %s: %w", dir, hookStateLockWaitLimit, err)
	}
	defer func() { retErr = errors.Join(retErr, lock.Release()) }()
	return dirscan.Read(dir, hookStateDirEntryCeiling())
}

// hookStateStoreScope names the store-wide reclamation lock. No file is
// created at this path; filelock keys its own lock object by the identity.
func hookStateStoreScope(dir string) string {
	return filepath.Join(dir, ".compaction")
}

func hookStateDirEntryCeiling() int {
	if hookStateRepairDepth.Load() > 0 {
		return hookStateDirRepairMaxEntries
	}
	return hookStateDirMaxEntries
}

func hookStateOverLimit(dir string, err error) error {
	return fmt.Errorf("%w; the governance hook state store %s is above its %d-entry limit and could not reclaim enough of it. Run 'machinery doctor --repair', which reclaims obligations whose project root no longer exists and keeps every obligation whose root still does", err, dir, hookStateDirMaxEntries)
}

func beginHookStateAutoCompaction() bool {
	hookStateCompactionMu.Lock()
	defer hookStateCompactionMu.Unlock()
	if hookStateAutoCompacted {
		return false
	}
	hookStateAutoCompacted = true
	return true
}

// boundHookStateStore keeps the store under its retention ceiling from the
// arming write path, so the bound is a property of every write rather than of
// a cleanup somebody has to remember to run. Only the enumeration can fail the
// caller: a single unreclaimable generation, including a foreign corrupt one,
// must never deny the tool call of an unrelated project.
func boundHookStateStore() error {
	dir, err := stateDirPathExact()
	if err != nil {
		return err
	}
	if hookStateBoundIsExhausted() {
		return nil
	}
	entries, err := readHookStateDir(dir)
	if err != nil {
		return err
	}
	if len(entries) <= hookStateDirRetentionCeiling {
		return nil
	}
	compaction, err := compactHookStateDir(dir, hookStateDirRetentionTarget)
	if err != nil {
		return err
	}
	if compaction.Remaining > hookStateDirRetentionTarget {
		exhaustHookStateBound()
	}
	return nil
}

func hookStateBoundIsExhausted() bool {
	hookStateCompactionMu.Lock()
	defer hookStateCompactionMu.Unlock()
	return hookStateBoundExhausted
}

func exhaustHookStateBound() {
	hookStateCompactionMu.Lock()
	defer hookStateCompactionMu.Unlock()
	hookStateBoundExhausted = true
}

// hookStateCompaction is one compaction pass stated in the operator's terms.
type hookStateCompaction struct {
	Dir       string
	Before    int
	Remaining int
	Reclaimed int
	Retained  int
	// FirstRetainedErr is the first hard reason a generation was kept, such as
	// a corrupt foreign ledger. It is reported, never raised: compaction is
	// housekeeping for the store, not adjudication of one project's state.
	FirstRetainedErr error
}

// compactHookStateDir reclaims whole generations until the store holds at most
// target entries. A generation is reclaimable only when its ledger records a
// project root that no longer exists: that obligation can never be discharged
// because there is no tree left to gate. Everything else is retained, so
// compaction can shrink the store but can never discharge a live obligation.
func compactHookStateDir(dir string, target int) (result hookStateCompaction, retErr error) {
	hookStateRepairDepth.Add(1)
	defer hookStateRepairDepth.Add(-1)
	entries, err := dirscan.Read(dir, hookStateDirRepairMaxEntries)
	if err != nil {
		return hookStateCompaction{Dir: dir}, err
	}
	result = hookStateCompaction{Dir: dir, Before: len(entries), Remaining: len(entries)}
	if result.Remaining <= target {
		return result, nil
	}
	// The store binding is validated once for the whole pass and every file is
	// then read and removed through one retained directory authority. Revalidating
	// the binding per file would take the initialization-marker lock thousands of
	// times while the store, and therefore every governed tool, is blocked.
	binding, err := validatedStateDirectoryBinding()
	if err != nil {
		return result, err
	}
	native, err := captureStateDirectoryIdentity(dir)
	if err != nil {
		return result, err
	}
	if native != binding.native {
		return result, fmt.Errorf("hook state store %s does not match its bound directory identity", dir)
	}
	storeRoot, err := os.OpenRoot(dir)
	if err != nil {
		return result, err
	}
	defer func() { retErr = errors.Join(retErr, storeRoot.Close()) }()
	generations, quarantined := groupHookStateGenerations(entries)
	// Reclamation runs in bounded batches, each holding the store-wide lock
	// for a fraction of a second. A single pass over a full store removes
	// thousands of files; holding readers out for all of it would stall every
	// governed event on the machine instead of the few this batch covers.
	pending := make([]*hookStateGeneration, 0, hookStateReclaimBatch)
	flush := func() error {
		if len(pending) == 0 {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), hookStateLockWaitLimit)
		defer cancel()
		lock, err := filelock.AcquireWaitContext(ctx, hookStateStoreScope(dir))
		if err != nil {
			return fmt.Errorf("hook state store %s could not be claimed for reclamation within %s: %w", dir, hookStateLockWaitLimit, err)
		}
		for _, generation := range pending {
			removed, err := reclaimHookStateGeneration(storeRoot, dir, generation, quarantined)
			result.Reclaimed += removed
			result.Remaining -= removed
			if err != nil {
				result.Retained++
				if !errors.Is(err, errHookStateGenerationRetained) && result.FirstRetainedErr == nil {
					result.FirstRetainedErr = err
				}
			}
		}
		pending = pending[:0]
		return lock.Release()
	}
	need := result.Remaining - target
	for _, generation := range generations {
		if need <= 0 {
			break
		}
		if generation.blocked || !generation.ledger {
			result.Retained++
			continue
		}
		// Screen before locking. A store whose obligations all belong to live
		// project roots must cost one cheap read per generation, not one lock
		// acquisition per generation on every governed tool call.
		dead, err := hookStateGenerationLooksDead(storeRoot, dir, generation)
		if err != nil || !dead {
			result.Retained++
			if err != nil && !errors.Is(err, errHookStateGenerationRetained) && result.FirstRetainedErr == nil {
				result.FirstRetainedErr = err
			}
			continue
		}
		pending = append(pending, generation)
		need -= 1 + len(generation.routes)
		if len(pending) < hookStateReclaimBatch {
			continue
		}
		if err := flush(); err != nil {
			return result, err
		}
		need = result.Remaining - target
	}
	if err := flush(); err != nil {
		return result, err
	}
	if result.Reclaimed > 0 {
		// One directory sync per pass, not per generation. A crash before this
		// sync leaves entries that a later pass reclaims again.
		if err := syncStateDirectory(dir); err != nil {
			return result, err
		}
	}
	return result, nil
}

type hookStateEntryKind uint8

const (
	hookStateEntryUnrelated hookStateEntryKind = iota
	hookStateEntryLedger
	hookStateEntryRoute
	// hookStateEntryEvidence is a durable crash temp or an otherwise
	// noncanonical name bound to a project base. It is never reclaimed and it
	// retains its whole generation.
	hookStateEntryEvidence
)

// classifyHookStateEntry maps one store filename onto the project generation
// that owns it. The shapes are exactly the ones statePath, routeStatePath and
// the durable temp writers produce; anything else that still carries a project
// base is crash evidence and freezes that generation.
func classifyHookStateEntry(name string) (string, hookStateEntryKind) {
	trimmed := strings.TrimPrefix(name, ".")
	temp := trimmed != name
	const suffix = ".state"
	if len(trimmed) < 64+len(suffix) || !validHookHexDigest(trimmed[:64]) || !strings.HasPrefix(trimmed[64:], suffix) {
		return "", hookStateEntryUnrelated
	}
	base := trimmed[:64] + suffix
	rest := trimmed[len(base):]
	if !temp && rest == "" {
		return base, hookStateEntryLedger
	}
	if !temp && strings.HasPrefix(rest, ".route-") {
		digest := strings.TrimSuffix(strings.TrimPrefix(rest, ".route-"), ".json")
		if strings.HasSuffix(rest, ".json") && validHookHexDigest(digest) {
			return base, hookStateEntryRoute
		}
	}
	return base, hookStateEntryEvidence
}

// hookStateGeneration is every store entry that belongs to one project root.
type hookStateGeneration struct {
	base    string
	ledger  bool
	routes  []string
	blocked bool
	modTime time.Time
}

// groupHookStateGenerations returns the generations oldest ledger first, so a
// bounded compaction reclaims the least recently touched garbage rather than
// an arbitrary slice of it.
func groupHookStateGenerations(entries []os.DirEntry) ([]*hookStateGeneration, bool) {
	byBase := make(map[string]*hookStateGeneration)
	generation := func(base string) *hookStateGeneration {
		found, ok := byBase[base]
		if !ok {
			found = &hookStateGeneration{base: base}
			byBase[base] = found
		}
		return found
	}
	quarantined := false
	for _, entry := range entries {
		if looksLikeHookQuarantine(entry.Name(), "d") || looksLikeHookQuarantine(entry.Name(), "r") {
			quarantined = true
		}
		base, kind := classifyHookStateEntry(entry.Name())
		switch kind {
		case hookStateEntryLedger:
			owner := generation(base)
			owner.ledger = true
			if info, err := entry.Info(); err == nil {
				owner.modTime = info.ModTime()
			} else {
				owner.blocked = true
			}
		case hookStateEntryRoute:
			owner := generation(base)
			owner.routes = append(owner.routes, entry.Name())
		case hookStateEntryEvidence:
			generation(base).blocked = true
		}
	}
	ordered := make([]*hookStateGeneration, 0, len(byBase))
	for _, owner := range byBase {
		sort.Strings(owner.routes)
		ordered = append(ordered, owner)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if !ordered[i].modTime.Equal(ordered[j].modTime) {
			return ordered[i].modTime.Before(ordered[j].modTime)
		}
		return ordered[i].base < ordered[j].base
	})
	return ordered, quarantined
}

// hookStateGenerationLooksDead reads one ledger without taking its lock and
// reports whether its project root is gone. It is a screen, never the
// decision: the locked reclaim re-reads and re-proves everything.
func hookStateGenerationLooksDead(storeRoot *os.Root, dir string, generation *hookStateGeneration) (bool, error) {
	witness, err := readBoundedHookRootFile(storeRoot, generation.base, filepath.Join(dir, generation.base), "hook state", hookStateMaxBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, errHookStateGenerationRetained
		}
		return false, err
	}
	record, err := parseHookStateRecord(witness.body)
	if err != nil {
		return false, fmt.Errorf("hook state ledger %s is corrupt; refusing to reclaim it: %w", filepath.Join(dir, generation.base), err)
	}
	return hookStateRootVanished(record.root), nil
}

// reclaimHookStateGeneration removes one dead generation under this project's
// own state lock. It proves the project root is gone while holding that lock,
// so no other process can be arming the generation it removes: an absent root
// cannot be canonicalized, and therefore cannot be governed, by anyone.
//
// Removal is witness-checked but deliberately not quarantined. A quarantine
// exists so an interrupted deletion of a live obligation stays restorable;
// this generation has no tree left to gate, so there is nothing to restore,
// and paying a durable quarantine per file would leave every governed tool on
// the machine blocked for minutes while a full store is repaired. Routes go
// first and the ledger last, so an interrupted pass leaves the obligation
// readable and reclaimable rather than orphaned.
func reclaimHookStateGeneration(storeRoot *os.Root, dir string, generation *hookStateGeneration, quarantined bool) (removed int, retErr error) {
	ledgerPath := filepath.Join(dir, generation.base)
	scope := ledgerPath + ".session"
	// A generation this process is itself arming is retained, never reclaimed.
	// The lock layer already refuses the second acquisition on every supported
	// platform, so this guard is not what makes that safe: flock builds
	// (darwin, linux and the other BSDs) scope the lock to the descriptor, so a
	// second open conflicts and reports contention, and the fcntl builds (AIX,
	// Solaris), where a record lock really does belong to the process, hold an
	// in-process reservation that refuses it one layer down. The guard states
	// the invariant where the decision is made, and keeps the accounting
	// truthful: this generation is retained deliberately, not incidentally.
	if hookStateLockHeldHere(scope) {
		return 0, errHookStateGenerationRetained
	}
	lock, err := filelock.Acquire(scope)
	if err != nil {
		if filelock.IsContended(err) {
			return 0, errHookStateGenerationRetained
		}
		return 0, err
	}
	defer func() { retErr = errors.Join(retErr, lock.Release()) }()
	// Interrupted-deletion recovery enumerates the whole store, so it runs
	// only when the store actually holds a quarantine. Compacting a store of
	// thousands of dead generations must not cost one full enumeration each.
	if quarantined {
		if err := recoverHookDeletionQuarantines(ledgerPath); err != nil {
			return 0, err
		}
	}
	witness, err := readBoundedHookRootFile(storeRoot, generation.base, ledgerPath, "hook state", hookStateMaxBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, errHookStateGenerationRetained
		}
		return 0, err
	}
	record, err := parseHookStateRecord(witness.body)
	if err != nil {
		// A corrupt ledger is exactly the state the hook must keep failing
		// closed on. Reclaiming it would launder corruption into a clean
		// project; report it and leave it for the operator.
		return 0, fmt.Errorf("hook state ledger %s is corrupt; refusing to reclaim it: %w", ledgerPath, err)
	}
	if !hookStateRootVanished(record.root) {
		return 0, errHookStateGenerationRetained
	}
	for _, name := range generation.routes {
		if err := removeWitnessedHookStateFile(storeRoot, name, filepath.Join(dir, name), "hook route snapshot", hookRouteMaxBytes); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return removed, err
		}
		removed++
	}
	if err := removeWitnessedHookStateFile(storeRoot, generation.base, ledgerPath, "hook state", hookStateMaxBytes); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return removed, nil
		}
		return removed, err
	}
	return removed + 1, nil
}

// removeWitnessedHookStateFile removes one file of a dead generation after two
// independent reads agree on exactly which file it is, and proves it is gone
// afterwards. A replacement that appears between the reads, or survives the
// removal, is preserved and reported rather than silently discarded.
func removeWitnessedHookStateFile(storeRoot *os.Root, name, display, kind string, limit int64) error {
	first, err := readBoundedHookRootFile(storeRoot, name, display, kind, limit)
	if err != nil {
		return err
	}
	second, err := readBoundedHookRootFile(storeRoot, name, display, kind, limit)
	if err != nil {
		return err
	}
	if !sameHookFileWitness(first, second) {
		return fmt.Errorf("%s %s changed before reclamation; preserving it", kind, display)
	}
	if err := storeRoot.Remove(name); err != nil {
		return err
	}
	if _, err := storeRoot.Lstat(name); err == nil {
		return fmt.Errorf("%s %s was replaced during reclamation; preserving the replacement", kind, display)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// hookStateRootVanished reports whether a recorded project root is provably
// absent. Only os.ErrNotExist counts: an unreadable or unreachable path is
// unknown, and unknown always retains the obligation.
//
// The residual is stated rather than hidden. A project root on detached or
// unmounted storage is indistinguishable from a deleted one, so its obligation
// can be reclaimed while the store is over its ceiling. That is a fail-open,
// not a deferral: the next governed edit in that project re-arms the whole-tree
// obligation, but if the volume returns and the session ends with no edit
// before Stop, that session's gate does not run at all. It stays narrow (the
// root must vanish while a session is live and the store must be over its
// ceiling at that moment) and oldest-ledger-first ordering reaches a freshly
// armed obligation last. The alternative, retaining it, is what fails every
// tool in every other repository closed.
func hookStateRootVanished(root string) bool {
	if root == "" || !filepath.IsAbs(root) {
		return false
	}
	_, err := os.Lstat(root)
	return errors.Is(err, os.ErrNotExist)
}

// routeSnapshotAge orders one project's route snapshots newest first.
type routeSnapshotAge struct {
	path    string
	modTime time.Time
}

// retainProjectRouteSnapshots keeps at most hookStateRouteRetention route
// snapshots for one project, newest first, and never the snapshot the caller
// just wrote. The caller holds this project's state lock, so the inventory it
// prunes cannot change underneath it.
func retainProjectRouteSnapshots(root, keep string) (retErr error) {
	paths, err := routeStatePaths(root)
	if err != nil {
		return err
	}
	if len(paths) <= hookStateRouteRetention {
		return nil
	}
	ordered := make([]routeSnapshotAge, 0, len(paths))
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		ordered = append(ordered, routeSnapshotAge{path: path, modTime: info.ModTime()})
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].path == keep || ordered[j].path == keep {
			return ordered[i].path == keep
		}
		if !ordered[i].modTime.Equal(ordered[j].modTime) {
			return ordered[i].modTime.After(ordered[j].modTime)
		}
		return ordered[i].path < ordered[j].path
	})
	if len(ordered) <= hookStateRouteRetention {
		return nil
	}
	surplus := ordered[hookStateRouteRetention:]
	// Two bounds, both about the store this write shares with every other
	// repository on the machine. A quarantined removal costs tens of
	// milliseconds, so one write reclaims at most a budget's worth (a project
	// that accumulated surplus under an older version converges over the next
	// few governed events instead of stalling one of them), and each store-wide
	// lock hold covers at most a batch (an enumeration in an unrelated
	// repository waits for one batch, never for the whole backlog, and never
	// past its own lock-wait budget).
	if len(surplus) > hookStateRouteReclaimBudget {
		surplus = surplus[:hookStateRouteReclaimBudget]
	}
	dir := filepath.Dir(keep)
	removed := false
	for start := 0; start < len(surplus); start += hookStateRouteReclaimBatch {
		end := min(start+hookStateRouteReclaimBatch, len(surplus))
		batchRemoved, err := removeRouteSnapshotBatch(dir, surplus[start:end])
		removed = removed || batchRemoved
		if err != nil {
			return err
		}
	}
	if !removed {
		return nil
	}
	return syncStateDirectory(dir)
}

// removeRouteSnapshotBatch removes one bounded run of surplus snapshots while
// holding the store-wide lock, and releases it before the caller takes the
// next run.
func removeRouteSnapshotBatch(dir string, batch []routeSnapshotAge) (removed bool, retErr error) {
	ctx, cancel := context.WithTimeout(context.Background(), hookStateLockWaitLimit)
	defer cancel()
	storeLock, err := filelock.AcquireWaitContext(ctx, hookStateStoreScope(dir))
	if err != nil {
		return false, fmt.Errorf("hook state store %s could not be claimed for route retention within %s: %w", dir, hookStateLockWaitLimit, err)
	}
	defer func() { retErr = errors.Join(retErr, storeLock.Release()) }()
	for _, snapshot := range batch {
		witness, err := readBoundedHookStateFile(snapshot.path, "hook route snapshot", hookRouteMaxBytes)
		if err != nil {
			return removed, err
		}
		if witness == nil {
			continue
		}
		if err := removeHookFileWitness(snapshot.path, "hook route snapshot", hookRouteMaxBytes, witness); err != nil {
			return removed, err
		}
		removed = true
	}
	return removed, nil
}

// StateReport writes the governance hook state store's retention status. With
// repair it also compacts the store. Both work when the store is already at or
// above its fail-closed entry limit, which is the only moment either matters:
// that is when every governed tool in every repository is blocked.
//
// Neither path creates the store. A machine that never armed an obligation has
// no store, and materializing one here would fabricate the durable evidence
// that distinguishes a first initialization from a lost one.
func StateReport(w io.Writer, repair bool) (ok bool, retErr error) {
	dir, err := stateDirPathExact()
	if err != nil {
		return false, fmt.Errorf("resolve the governance hook state store: %w", err)
	}
	info, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(w, "  auto     no governance hook state store at %s; nothing has armed a gate obligation on this machine\n", dir)
		return true, nil
	}
	if err != nil {
		fmt.Fprintf(w, "  ERROR    governance hook state store %s cannot be inspected: %v\n", dir, err)
		return false, nil
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		fmt.Fprintf(w, "  ERROR    governance hook state store %s must be a real directory\n", dir)
		return false, nil
	}
	if repair {
		compaction, err := compactHookStateDir(dir, hookStateDirRetentionTarget)
		if err != nil {
			fmt.Fprintf(w, "  ERROR    governance hook state store %s could not be compacted: %v. A missing or drifted store initialization marker fails here, and fails every governed event for the same reason, so read the message above as being about the store's identity rather than its retention\n", dir, err)
			return false, nil
		}
		fmt.Fprintf(w, "  ok       governance hook state store %s compacted: reclaimed %d of %d entr%s for project roots that no longer exist\n",
			dir, compaction.Reclaimed, compaction.Before, plural(compaction.Before))
		if compaction.FirstRetainedErr != nil {
			fmt.Fprintf(w, "  present  governance hook state store %s kept a generation it cannot reclaim: %v\n", dir, compaction.FirstRetainedErr)
		}
		return reportHookStateCount(w, dir, compaction.Remaining), nil
	}
	count, err := countHookStateEntries(dir)
	if err != nil {
		fmt.Fprintf(w, "  ERROR    governance hook state store %s cannot be enumerated: %v\n", dir, err)
		return false, nil
	}
	return reportHookStateCount(w, dir, count), nil
}

func reportHookStateCount(w io.Writer, dir string, count int) bool {
	if count >= hookStateDirMaxEntries {
		fmt.Fprintf(w, "  ERROR    governance hook state store %s holds %d entries, at or above its %d-entry limit; every governance hook fails closed on it. Run 'machinery doctor --repair', which reclaims obligations whose project root no longer exists and keeps every obligation whose root still does\n",
			dir, count, hookStateDirMaxEntries)
		return false
	}
	if count > hookStateDirRetentionCeiling {
		fmt.Fprintf(w, "  present  governance hook state store %s holds %d entries, above its %d-entry retention ceiling but below the %d-entry limit; obligations for live project roots are never reclaimed\n",
			dir, count, hookStateDirRetentionCeiling, hookStateDirMaxEntries)
		return true
	}
	fmt.Fprintf(w, "  ok       governance hook state store %s holds %d entr%s (retention ceiling %d, fail-closed limit %d)\n",
		dir, count, plural(count), hookStateDirRetentionCeiling, hookStateDirMaxEntries)
	return true
}

func plural(count int) string {
	if count == 1 {
		return "y"
	}
	return "ies"
}

// countHookStateEntries reads the store under the repair ceiling so a doctor
// report states the real count of a store that is already over the
// fail-closed limit instead of failing on it.
func countHookStateEntries(dir string) (int, error) {
	entries, err := dirscan.Read(dir, hookStateDirRepairMaxEntries)
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}
