package hook

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// Crash-temp recovery (B8). Every hook state write goes through a durable
// temp (".<state>.tmp-*" for the dirty ledger, ".<route>.json.tmp-*" for a
// route snapshot) that is renamed into place. A write interrupted between the
// temp and the rename (a full disk, a killed hook process) leaves the temp
// behind. Treating every such temp as untouchable crash evidence made one
// crashed write block every later governed command in the project forever.
//
// Recovery runs under the project's state lock, so no hook writer of this
// project can be between its own temp and its own rename:
//
//   - an empty temp, or a provably stale one, is removed through the same
//     witness-checked deletion every other hook file uses. Provably stale
//     means its content is already the live state (byte-identical), or, for
//     the ledger, a canonical record whose revision is not newer than the
//     live one, or, for a route, a snapshot whose digest the live ledger does
//     not reference (no obligation was ever bound to it);
//   - any other temp is moved aside as crash evidence
//     (".<base>.crashed-<hex>", mode 0600, a name no temp inventory matches),
//     the project is marked dirty for design and impl so the stop-time checks
//     still run, and the one command that found it is refused with a single
//     line naming the preserved file and how to remove it. The next command
//     proceeds normally.
//
// The stop path does not recover: readStateRecord still refuses to read a
// project with a temp present as untouched, and the next governed shell or
// file command performs the recovery.

// hookCrashEvidence is one temp moved aside during recovery.
type hookCrashEvidence struct {
	path string
	kind string // "state" or "route"
}

// hookCrashRefusal refuses the one command that found unprovable crash
// evidence. Its message is a single line by contract: hosts show it verbatim.
type hookCrashRefusal struct {
	evidence []hookCrashEvidence
}

func (r *hookCrashRefusal) Error() string {
	var kinds []string
	quoted := make([]string, 0, len(r.evidence))
	for _, e := range r.evidence {
		if !slices.Contains(kinds, e.kind) {
			kinds = append(kinds, e.kind)
		}
		quoted = append(quoted, shellQuote(e.path))
	}
	sort.Strings(kinds)
	it := "its temp"
	if len(r.evidence) > 1 {
		it = "the temps"
	}
	return fmt.Sprintf("machinery hook: incomplete hook %s transaction (an interrupted or crashed write); preserved %s as %s, marked this project dirty for design and impl so the stop-time checks still run, and refused this one command; retry it, the next command proceeds. After inspecting the evidence, remove it: rm %s",
		strings.Join(kinds, " and "), it, strings.Join(quoted, ", "), strings.Join(quoted, " "))
}

// asError keeps a nil refusal a nil error (never a typed nil in an interface).
func (r *hookCrashRefusal) asError() error {
	if r == nil {
		return nil
	}
	return r
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// recoverHookCrashTemps takes the project state lock and recovers.
func recoverHookCrashTemps(root string) (refusal *hookCrashRefusal, returnErr error) {
	p := statePath(root, "")
	lock, err := acquireHookStateLock(p)
	if err != nil {
		return nil, err
	}
	defer func() { returnErr = errors.Join(returnErr, lock.release()) }()
	return recoverHookCrashTempsLocked(root, p)
}

// recoverHookCrashTempsLocked recovers every state and route temp of the
// project whose ledger is p. The caller holds p's state lock. It returns a
// refusal when evidence was preserved (the ledger is then already dirty), and
// an error only when the store itself cannot be read or written safely.
func recoverHookCrashTempsLocked(root, p string) (*hookCrashRefusal, error) {
	// an interrupted witness-checked deletion is resumed (or restored) first,
	// so a temp it was removing is seen exactly once
	if err := recoverHookDeletionQuarantines(p); err != nil {
		return nil, err
	}
	stateTemps, err := hookStateTemps(p)
	if err != nil {
		return nil, err
	}
	routeTemps, err := routeStateTemps(root)
	if err != nil {
		return nil, err
	}
	if len(stateTemps) == 0 && len(routeTemps) == 0 {
		return nil, nil
	}
	liveWitness, err := readBoundedHookStateFile(p, "hook state", hookStateMaxBytes)
	if err != nil {
		return nil, err
	}
	// liveRecord stays nil when the live ledger is absent or unreadable;
	// staleness against an unreadable ledger is unprovable, never assumed
	var liveRecord *hookStateRecord
	liveParsed := liveWitness == nil
	if liveWitness != nil {
		if record, parseErr := parseHookStateRecord(liveWitness.body); parseErr == nil {
			liveRecord, liveParsed = &record, true
		}
	}

	var evidence []hookCrashEvidence
	changed := false
	for _, temp := range stateTemps {
		stale := func(body []byte) bool {
			if liveWitness == nil {
				return false // a first write crashed: the temp is the only record
			}
			if bytes.Equal(body, liveWitness.body) {
				return true
			}
			record, err := parseHookStateRecord(body)
			return err == nil && liveRecord != nil && record.revision <= liveRecord.revision
		}
		preserved, err := recoverOneHookCrashTemp(temp, "hook state", hookStateMaxBytes, stale)
		if err != nil {
			return nil, err
		}
		changed = true
		if preserved != "" {
			evidence = append(evidence, hookCrashEvidence{path: preserved, kind: "state"})
		}
	}
	for _, temp := range routeTemps {
		live := routeTempTarget(temp)
		liveRoute, err := readBoundedHookStateFile(live, "hook route snapshot", hookRouteMaxBytes)
		if err != nil {
			return nil, err
		}
		stale := func(body []byte) bool {
			if liveRoute != nil && bytes.Equal(body, liveRoute.body) {
				return true
			}
			if !liveParsed {
				return false
			}
			// an unreadable snapshot is not provably anything
			if _, err := decodeConfig(body); err != nil {
				return false
			}
			// the ledger binds a route digest before its snapshot is written,
			// so a snapshot the live ledger never referenced carried no
			// obligation; one it does reference is not provably stale
			if liveRecord == nil {
				return true
			}
			return !slices.Contains(liveRecord.routes, routeSnapshotDigest(body))
		}
		preserved, err := recoverOneHookCrashTemp(temp, "hook route", hookRouteMaxBytes, stale)
		if err != nil {
			return nil, err
		}
		changed = true
		if preserved != "" {
			evidence = append(evidence, hookCrashEvidence{path: preserved, kind: "route"})
		}
	}
	if changed {
		if err := syncStateDirectory(filepath.Dir(p)); err != nil {
			return nil, err
		}
	}
	if len(evidence) == 0 {
		return nil, nil
	}
	// conservative: whatever the lost write meant to record, the stop-time
	// checks must run over both trees
	if err := updateStateLocked(p, root, ledgerMutation{addDesign: true, addImpl: true}); err != nil {
		return nil, err
	}
	keep := make([]string, 0, len(evidence))
	for _, e := range evidence {
		keep = append(keep, e.path)
	}
	if err := retainHookCrashEvidence(p, keep); err != nil {
		return nil, err
	}
	if err := boundHookStateStore(); err != nil {
		return nil, err
	}
	return &hookCrashRefusal{evidence: evidence}, nil
}

// routeTempTarget is the live route snapshot a route temp would have replaced.
func routeTempTarget(temp string) string {
	name := strings.TrimPrefix(filepath.Base(temp), ".")
	if i := strings.Index(name, ".json.tmp-"); i >= 0 {
		name = name[:i+len(".json")]
	}
	return filepath.Join(filepath.Dir(temp), name)
}

// recoverOneHookCrashTemp removes one temp when it is empty or stale and
// otherwise moves it aside, returning the preserved path ("" when removed or
// already gone). It never follows a symlink and never reads past limit: an
// oversize temp is unprovable and preserved without reading it.
func recoverOneHookCrashTemp(temp, kind string, limit int64, stale func([]byte) bool) (string, error) {
	kind += " temp"
	info, err := os.Lstat(temp)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s %s must be a regular file, not a symlink or special file; refusing to recover it", kind, temp)
	}
	if info.Size() <= limit {
		witness, err := readBoundedHookStateFile(temp, kind, limit)
		if err != nil {
			return "", err
		}
		if witness == nil {
			return "", nil
		}
		if len(witness.body) == 0 || stale(witness.body) {
			return "", removeHookCrashTempWitness(temp, kind, limit, witness)
		}
		info = witness.info
	}
	return preserveHookCrashTemp(temp, kind, info)
}

// removeHookCrashTempWitness removes one stale temp that is still exactly the
// file it was judged as. A ledger temp goes through the restorable quarantined
// deletion every hook state file uses. A route temp's name is too long for a
// portable quarantine name (it carries two digests), so it is removed the way
// retention removes route snapshots: two independent reads must agree with
// the judged witness, and the name must be gone afterwards.
func removeHookCrashTempWitness(temp, kind string, limit int64, want *hookFileWitness) (retErr error) {
	if !strings.Contains(filepath.Base(temp), ".route-") {
		return removeHookFileWitness(temp, kind, limit, want)
	}
	root, err := os.OpenRoot(filepath.Dir(temp))
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	got, err := readBoundedHookRootFile(root, filepath.Base(temp), temp, kind, limit)
	if err != nil {
		return err
	}
	if !sameHookFileWitness(want, got) {
		return fmt.Errorf("%s %s changed before deletion; preserving it", kind, temp)
	}
	return removeWitnessedHookStateFile(root, filepath.Base(temp), temp, kind, limit)
}

// preserveHookCrashTemp renames one temp, as the same file object it was
// inspected as, to a private crash-evidence name in the same directory.
func preserveHookCrashTemp(temp, kind string, want os.FileInfo) (_ string, retErr error) {
	base, err := hookProjectBase(temp)
	if err != nil {
		return "", err
	}
	var random [8]byte
	if _, err := cryptorand.Read(random[:]); err != nil {
		return "", err
	}
	dir, name := filepath.Dir(temp), filepath.Base(temp)
	preserved := "." + base + ".crashed-" + hex.EncodeToString(random[:])
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	if _, err := root.Lstat(preserved); !errors.Is(err, os.ErrNotExist) {
		return "", errors.Join(err, fmt.Errorf("crash evidence name %s is already taken", filepath.Join(dir, preserved)))
	}
	before, err := root.Lstat(name)
	if err != nil {
		return "", err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || !os.SameFile(want, before) {
		return "", fmt.Errorf("%s %s changed before it could be preserved; preserving it in place", kind, temp)
	}
	if err := root.Rename(name, preserved); err != nil {
		return "", err
	}
	after, err := root.Lstat(preserved)
	if err != nil {
		return "", err
	}
	if after.Mode()&os.ModeSymlink != 0 || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return "", fmt.Errorf("%s %s changed while it was being preserved as %s; inspect both", kind, temp, filepath.Join(dir, preserved))
	}
	if after.Mode().Perm() != 0o600 {
		if err := root.Chmod(preserved, 0o600); err != nil {
			return "", err
		}
	}
	return filepath.Join(dir, preserved), nil
}

// looksLikeHookCrashEvidence reports a preserved crash-temp name for base.
func looksLikeHookCrashEvidence(name, base string) bool {
	rest, ok := strings.CutPrefix(name, "."+base+".crashed-")
	return ok && len(rest) == 16 && validLowerHex(rest)
}

// retainHookCrashEvidence bounds one project's preserved crash temps at
// hookCrashEvidenceRetention, newest first, never removing the ones keep
// names (the refusal the caller is about to emit names them). Every one of
// them already marked the project dirty, so an older one is evidence only.
func retainHookCrashEvidence(p string, keep []string) (retErr error) {
	base, err := hookProjectBase(p)
	if err != nil {
		return err
	}
	dir := filepath.Dir(p)
	entries, err := readHookStateDir(dir)
	if err != nil {
		return err
	}
	type aged struct {
		name string
		info os.FileInfo
	}
	var others []aged
	for _, entry := range entries {
		if !looksLikeHookCrashEvidence(entry.Name(), base) || slices.Contains(keep, filepath.Join(dir, entry.Name())) {
			continue
		}
		info, err := os.Lstat(filepath.Join(dir, entry.Name()))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		others = append(others, aged{entry.Name(), info})
	}
	room := max(hookCrashEvidenceRetention-len(keep), 0)
	if len(others) <= room {
		return nil
	}
	sort.Slice(others, func(i, j int) bool {
		if !others[i].info.ModTime().Equal(others[j].info.ModTime()) {
			return others[i].info.ModTime().After(others[j].info.ModTime())
		}
		return others[i].name > others[j].name
	})
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	for _, old := range others[room:] {
		if err := removeHookCrashEvidence(root, dir, old.name, old.info); err != nil {
			return err
		}
	}
	return syncStateDirectory(dir)
}

// removeHookCrashEvidence removes one preserved crash temp that is still the
// regular file it was inventoried as. Its content was already accounted for
// (the project was marked dirty when it was preserved), so identity, not
// bytes, is what the removal binds; an oversize file is removable too.
func removeHookCrashEvidence(root *os.Root, dir, name string, want os.FileInfo) error {
	got, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if got.Mode()&os.ModeSymlink != 0 || !got.Mode().IsRegular() || (want != nil && !os.SameFile(want, got)) {
		return fmt.Errorf("hook crash evidence %s changed before removal; preserving it", filepath.Join(dir, name))
	}
	if err := root.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
