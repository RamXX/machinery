package checker

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/designlock"
)

const (
	projectionTransactionVersion = 2
	projectionControlDirName     = ".machinery"
	projectionPreparedName       = "checker-project-transaction.json"
	projectionBoundName          = "checker-project-transaction.bound.json"
	projectionCommittedName      = "checker-project-transaction.committed.json"
	projectionMaxControlRecord   = 16 << 20
)

var errSimulatedProjectionCrash = errors.New("simulated checker projection process crash")

type projectionTransactionRecord struct {
	Version int                          `json:"version"`
	Phase   string                       `json:"phase"`
	Entries []projectionTransactionEntry `json:"entries"`
}

type projectionTransactionEntry struct {
	Target  string                `json:"target"`
	Stage   string                `json:"stage"`
	Backup  string                `json:"backup"`
	Before  projectionFileWitness `json:"before"`
	After   projectionFileWitness `json:"after"`
	Existed bool                  `json:"existed"`
}

type projectionFileWitness struct {
	SHA256      string `json:"sha256"`
	Size        int64  `json:"size"`
	Mode        uint32 `json:"mode"`
	ModTimeNano int64  `json:"mod_time_nano"`
	Identity    string `json:"identity"`
}

type projectionTransactionHooks struct {
	beforeRename func(string, string) error
	beforeRemove func(string) error
	fault        func(string) error
	authorize    func() error
}

type projectionControlSnapshot struct {
	path    string
	body    []byte
	witness projectionFileWitness
	change  string
}

type projectionTransactionAuthority struct {
	control   string
	snapshots map[string]projectionControlSnapshot
	retiring  bool
}

type resolvedProjectionTransactionEntry struct {
	projectionTransactionEntry
	target string
	stage  string
	backup string
}

func commitProjectionPlansWithRename(design string, plans []plannedProjection, beforeRename func(string, string) error) error {
	return designlock.With(design, func() error {
		return commitProjectionPlansWithHooks(design, plans, projectionTransactionHooks{beforeRename: beforeRename})
	})
}

func commitProjectionPlansWithHooks(design string, plans []plannedProjection, hooks projectionTransactionHooks) (retErr error) {
	root, err := openDesignRoot(design)
	if err != nil {
		return err
	}
	defer closeRoot(&retErr, root)
	return commitProjectionPlansRoot(root, plans, hooks)
}

func commitProjectionPlansRoot(root *designRoot, plans []plannedProjection, hooks projectionTransactionHooks) (retErr error) {
	if err := recoverProjectionTransactionRoot(root); err != nil {
		return err
	}
	control, err := ensureProjectionControlDir(root)
	if err != nil {
		return err
	}
	preparedPath := filepath.Join(control, projectionPreparedName)
	boundPath := filepath.Join(control, projectionBoundName)
	committedPath := filepath.Join(control, projectionCommittedName)
	journaled := false
	crashed := false
	defer func() {
		if crashed || journaled {
			return
		}
		retErr = errors.Join(retErr, removeProjectionControlDirIfEmpty(root, control))
	}()

	sortedPlans := append([]plannedProjection(nil), plans...)
	sort.Slice(sortedPlans, func(i, j int) bool { return sortedPlans[i].dest < sortedPlans[j].dest })
	record := projectionTransactionRecord{Version: projectionTransactionVersion, Phase: "prepared"}
	byTarget := make(map[string]plannedProjection, len(sortedPlans))
	for _, plan := range sortedPlans {
		target, rel, err := projectionPlanTarget(root, plan.dest)
		if err != nil {
			return err
		}
		if err := root.validateNoSymlink(target); err != nil {
			return fmt.Errorf("projection target %s is unsafe: %w", root.display(target), err)
		}
		folded := strings.ToLower(filepath.ToSlash(rel))
		if _, exists := byTarget[folded]; exists {
			return fmt.Errorf("projection transaction target %s appears more than once", rel)
		}
		byTarget[folded] = plan
		digest := fmt.Sprintf("%x", sha256.Sum256([]byte(filepath.ToSlash(rel))))[:16]
		dirRel := filepath.Dir(rel)
		stageRel := filepath.Join(dirRel, ".machinery-project-stage-"+digest)
		backupRel := filepath.Join(dirRel, ".machinery-project-backup-"+digest)
		before, existed, err := captureProjectionFile(root, target, "projection target")
		if err != nil {
			return err
		}
		for _, transient := range []string{stageRel, backupRel} {
			if err := root.validateNoSymlink(transient); err != nil {
				return err
			}
			if _, err := root.root.Lstat(transient); err == nil {
				return fmt.Errorf("unowned checker projection transaction artifact already exists: %s", root.display(transient))
			} else if !os.IsNotExist(err) {
				return err
			}
		}
		record.Entries = append(record.Entries, projectionTransactionEntry{
			Target: filepath.ToSlash(rel),
			Stage:  filepath.ToSlash(stageRel),
			Backup: filepath.ToSlash(backupRel),
			Before: before,
			After: projectionFileWitness{
				SHA256: fmt.Sprintf("%x", sha256.Sum256(plan.rendered)),
				Size:   int64(len(plan.rendered)),
			},
			Existed: existed,
		})
	}

	resolved, err := validateProjectionTransactionRecord(root, record, "prepared")
	if err != nil {
		return err
	}
	if err := writeProjectionTransactionRecord(root, preparedPath, record); err != nil {
		return err
	}
	journaled = true
	rollback := func(cause error) error {
		recoveryErr := recoverProjectionTransactionRootWithHooks(root, hooks)
		journaled = recoveryErr != nil
		return errors.Join(cause, recoveryErr)
	}
	for index, entry := range resolved {
		if err := createProjectionDirectoryTree(root, filepath.Dir(entry.target)); err != nil {
			return rollback(err)
		}
		plan := byTarget[strings.ToLower(entry.Target)]
		stage, err := root.root.OpenFile(entry.stage, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return rollback(err)
		}
		if _, err := stage.Write(plan.rendered); err != nil {
			_ = stage.Close()
			return rollback(err)
		}
		if err := stage.Sync(); err != nil {
			_ = stage.Close()
			return rollback(err)
		}
		if err := stage.Close(); err != nil {
			return rollback(err)
		}
		if err := root.syncDir(filepath.Dir(entry.stage)); err != nil {
			return rollback(err)
		}
		witness, exists, err := captureProjectionFile(root, entry.stage, "staged checker projection")
		if err != nil {
			return rollback(err)
		}
		if !exists || witness.SHA256 != record.Entries[index].After.SHA256 || witness.Size != record.Entries[index].After.Size {
			return rollback(fmt.Errorf("staged checker projection %s does not match its intended post-image", root.display(entry.stage)))
		}
		record.Entries[index].After = witness
	}
	bound := record
	bound.Phase = "bound"
	if err := writeProjectionTransactionRecord(root, boundPath, bound); err != nil {
		return rollback(err)
	}
	resolved, err = validateProjectionTransactionRecord(root, bound, "bound")
	if err != nil {
		return rollback(err)
	}
	if err := projectionFault(hooks, "prepared"); err != nil {
		crashed = true
		return err
	}
	for index, entry := range resolved {
		if !entry.Existed {
			continue
		}
		if err := renameProjectionPathRoot(root, hooks, entry.target, entry.backup); err != nil {
			return rollback(err)
		}
		if err := root.syncDir(filepath.Dir(entry.target)); err != nil {
			return rollback(err)
		}
		if err := projectionFault(hooks, fmt.Sprintf("parked:%d", index)); err != nil {
			crashed = true
			return err
		}
	}
	for index, entry := range resolved {
		if err := renameProjectionPathRoot(root, hooks, entry.stage, entry.target); err != nil {
			return rollback(err)
		}
		if err := root.syncDir(filepath.Dir(entry.target)); err != nil {
			return rollback(err)
		}
		if err := projectionFault(hooks, fmt.Sprintf("installed:%d", index)); err != nil {
			crashed = true
			return err
		}
	}
	committed := bound
	committed.Phase = "committed"
	if err := writeProjectionTransactionRecord(root, committedPath, committed); err != nil {
		return rollback(err)
	}
	if err := projectionFault(hooks, "committed"); err != nil {
		crashed = true
		return err
	}
	if err := recoverProjectionTransactionRootWithHooks(root, hooks); err != nil {
		return err
	}
	journaled = false
	return nil
}

func renameProjectionPathRoot(root *designRoot, hooks projectionTransactionHooks, from, to string) error {
	if hooks.beforeRename != nil {
		if err := hooks.beforeRename(root.display(from), root.display(to)); err != nil {
			return err
		}
	}
	if err := root.validateNoSymlink(from); err != nil {
		return err
	}
	if err := root.validateNoSymlink(filepath.Dir(to)); err != nil {
		return err
	}
	return root.root.Rename(from, to)
}

func projectionFault(hooks projectionTransactionHooks, point string) error {
	if hooks.fault == nil {
		return nil
	}
	return hooks.fault(point)
}

func recoverProjectionTransaction(design string) (retErr error) {
	root, err := openDesignRoot(design)
	if err != nil {
		return err
	}
	defer closeRoot(&retErr, root)
	return recoverProjectionTransactionRoot(root)
}

func recoverProjectionTransactionRoot(root *designRoot) error {
	return recoverProjectionTransactionRootWithHooks(root, projectionTransactionHooks{})
}

func recoverProjectionTransactionRootWithHooks(root *designRoot, hooks projectionTransactionHooks) error {
	control := projectionControlDirName
	if err := root.validateNoSymlink(control); err != nil {
		return err
	}
	info, err := root.root.Lstat(control)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || !projectionControlPermissionsSafe(info.Mode()) {
		return fmt.Errorf("checker projection transaction control %s must be a private real directory", root.display(control))
	}
	if err := cleanupIncompleteProjectionTransactionRecords(root, control, hooks); err != nil {
		return err
	}
	authority, err := bindProjectionTransactionAuthority(root, control)
	if err != nil {
		return err
	}
	if authority.retiring {
		return removeProjectionTransactionRecords(root, control, hooks, authority)
	}
	preparedSnapshot, preparedExists := authority.snapshots[projectionPreparedName]
	boundSnapshot, boundExists := authority.snapshots[projectionBoundName]
	committedSnapshot, committedExists := authority.snapshots[projectionCommittedName]
	if !preparedExists && !boundExists && !committedExists {
		return nil
	}
	hooks.authorize = func() error { return validateProjectionTransactionAuthority(root, authority) }
	if !preparedExists {
		if !committedExists {
			return fmt.Errorf("checker projection transaction journals have an impossible durable state")
		}
		committed, err := loadProjectionTransactionRecord(committedSnapshot, "committed")
		if err != nil {
			return err
		}
		resolved, err := validateProjectionTransactionRecord(root, committed, "committed")
		if err != nil {
			return err
		}
		if boundExists {
			bound, err := loadProjectionTransactionRecord(boundSnapshot, "bound")
			if err != nil {
				return err
			}
			if !sameProjectionTransaction(bound, committed) {
				return fmt.Errorf("committed checker projection transaction does not match bound journal")
			}
		}
		return finalizeCommittedProjectionTransaction(root, control, resolved, hooks, authority)
	}
	prepared, err := loadProjectionTransactionRecord(preparedSnapshot, "prepared")
	if err != nil {
		return err
	}
	preparedResolved, err := validateProjectionTransactionRecord(root, prepared, "prepared")
	if err != nil {
		return err
	}
	if committedExists {
		if !boundExists {
			return fmt.Errorf("committed checker projection transaction is missing its bound post-image journal")
		}
		committed, err := loadProjectionTransactionRecord(committedSnapshot, "committed")
		if err != nil {
			return err
		}
		if !sameProjectionIntent(prepared, committed) {
			return fmt.Errorf("committed checker projection transaction does not match prepared journal")
		}
		bound, err := loadProjectionTransactionRecord(boundSnapshot, "bound")
		if err != nil {
			return err
		}
		if !sameProjectionIntent(prepared, bound) || !sameProjectionTransaction(bound, committed) {
			return fmt.Errorf("committed checker projection transaction does not match bound journal")
		}
		resolved, err := validateProjectionTransactionRecord(root, committed, "committed")
		if err != nil {
			return err
		}
		return finalizeCommittedProjectionTransaction(root, control, resolved, hooks, authority)
	}
	if !boundExists {
		return recoverUnboundProjectionTransaction(root, control, preparedResolved, hooks, authority)
	}
	bound, err := loadProjectionTransactionRecord(boundSnapshot, "bound")
	if err != nil {
		return err
	}
	if !sameProjectionIntent(prepared, bound) {
		return fmt.Errorf("bound checker projection transaction does not match prepared journal")
	}
	resolved, err := validateProjectionTransactionRecord(root, bound, "bound")
	if err != nil {
		return err
	}

	var recoveryErrs []error
	for i := len(resolved) - 1; i >= 0; i-- {
		entry := resolved[i]
		_, backupExists, err := captureProjectionFile(root, entry.backup, "checker projection transaction backup")
		if err != nil {
			return err
		}
		_, targetExists, err := captureProjectionFile(root, entry.target, "checker projection transaction target")
		if err != nil {
			return err
		}
		if backupExists {
			matches, err := projectionFileMatches(root, entry.backup, entry.Before, "checker projection transaction backup")
			if err != nil || !matches {
				recoveryErrs = append(recoveryErrs, errors.Join(err, fmt.Errorf("parked projection %s no longer matches its exact pre-image; preserving live and backup files", root.display(entry.backup))))
				continue
			}
			if targetExists {
				if err := removeProjectionFileIfMatch(root, hooks, entry.target, entry.After, "installed checker projection"); err != nil {
					recoveryErrs = append(recoveryErrs, err)
					continue
				}
			}
			if err := restoreProjectionBackup(root, hooks, entry); err != nil {
				recoveryErrs = append(recoveryErrs, err)
				continue
			}
		} else if entry.Existed {
			if !targetExists {
				recoveryErrs = append(recoveryErrs, fmt.Errorf("cannot recover missing original projection %s", root.display(entry.target)))
			} else if matches, err := projectionFileMatches(root, entry.target, entry.Before, "restored checker projection"); err != nil || !matches {
				recoveryErrs = append(recoveryErrs, errors.Join(err, fmt.Errorf("projection %s does not match its exact restored pre-image; preserving it", root.display(entry.target))))
			}
		} else if !entry.Existed && targetExists {
			if err := removeProjectionFileIfMatch(root, hooks, entry.target, entry.After, "installed checker projection"); err != nil {
				recoveryErrs = append(recoveryErrs, err)
			}
		}
		if _, stageExists, err := captureProjectionFile(root, entry.stage, "checker projection transaction stage"); err != nil {
			recoveryErrs = append(recoveryErrs, err)
		} else if stageExists {
			if err := removeProjectionFileIfMatch(root, hooks, entry.stage, entry.After, "staged checker projection"); err != nil {
				recoveryErrs = append(recoveryErrs, err)
			}
		}
	}
	if err := errors.Join(recoveryErrs...); err != nil {
		return fmt.Errorf("rollback checker projection transaction: %w", err)
	}
	return removeProjectionTransactionRecords(root, control, hooks, authority)
}

func finalizeCommittedProjectionTransaction(root *designRoot, control string, entries []resolvedProjectionTransactionEntry, hooks projectionTransactionHooks, authority projectionTransactionAuthority) error {
	for _, entry := range entries {
		exists, err := root.lstatRegular(entry.target, "checker projection transaction artifact", false)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("committed projection target is missing: %s", root.display(entry.target))
		}
		if _, stageExists, err := captureProjectionFile(root, entry.stage, "checker projection transaction stage"); err != nil {
			return err
		} else if stageExists {
			if err := removeProjectionFileIfMatch(root, hooks, entry.stage, entry.After, "staged checker projection"); err != nil {
				return err
			}
		}
		if _, backupExists, err := captureProjectionFile(root, entry.backup, "checker projection transaction backup"); err != nil {
			return err
		} else if backupExists {
			if err := removeProjectionFileIfMatch(root, hooks, entry.backup, entry.Before, "parked checker projection"); err != nil {
				return err
			}
		}
		if err := root.syncDir(filepath.Dir(entry.target)); err != nil {
			return err
		}
	}
	return removeProjectionTransactionRecords(root, control, hooks, authority)
}

func recoverUnboundProjectionTransaction(root *designRoot, control string, entries []resolvedProjectionTransactionEntry, hooks projectionTransactionHooks, authority projectionTransactionAuthority) error {
	var recoveryErrs []error
	for _, entry := range entries {
		if _, exists, err := captureProjectionFile(root, entry.backup, "unbound checker projection backup"); err != nil {
			recoveryErrs = append(recoveryErrs, err)
		} else if exists {
			recoveryErrs = append(recoveryErrs, fmt.Errorf("unbound checker projection transaction contains %s; preserving ambiguous state", root.display(entry.backup)))
		}
		if _, exists, err := captureProjectionFile(root, entry.stage, "unbound checker projection stage"); err != nil {
			recoveryErrs = append(recoveryErrs, err)
		} else if exists {
			recoveryErrs = append(recoveryErrs, fmt.Errorf("unbound checker projection transaction contains %s; preserving ambiguous state", root.display(entry.stage)))
		}
		current, exists, err := captureProjectionFile(root, entry.target, "unbound checker projection target")
		if err != nil {
			recoveryErrs = append(recoveryErrs, err)
			continue
		}
		if exists != entry.Existed || (exists && current != entry.Before) {
			recoveryErrs = append(recoveryErrs, fmt.Errorf("projection %s changed before its post-image was durably bound; preserving it", root.display(entry.target)))
		}
	}
	if err := errors.Join(recoveryErrs...); err != nil {
		return fmt.Errorf("recover unbound checker projection transaction: %w", err)
	}
	return removeProjectionTransactionRecords(root, control, hooks, authority)
}

func captureProjectionFile(root *designRoot, rel, kind string) (witness projectionFileWitness, exists bool, retErr error) {
	if err := root.validateNoSymlink(rel); err != nil {
		return projectionFileWitness{}, false, err
	}
	before, err := root.root.Lstat(rel)
	if os.IsNotExist(err) {
		return projectionFileWitness{}, false, nil
	}
	if err != nil {
		return projectionFileWitness{}, false, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return projectionFileWitness{}, false, fmt.Errorf("%s %s must be a regular, non-symlink file", kind, root.display(rel))
	}
	file, err := root.root.Open(rel)
	if err != nil {
		return projectionFileWitness{}, false, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return projectionFileWitness{}, false, errors.Join(err, fmt.Errorf("%s %s changed identity while opening", kind, root.display(rel)))
	}
	identity, err := projectionFileIdentity(file, opened)
	if err != nil {
		return projectionFileWitness{}, false, fmt.Errorf("identify %s %s: %w", kind, root.display(rel), err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return projectionFileWitness{}, false, fmt.Errorf("hash %s %s: %w", kind, root.display(rel), err)
	}
	after, statErr := file.Stat()
	pathAfter, pathErr := root.root.Lstat(rel)
	if err := errors.Join(statErr, pathErr); err != nil {
		return projectionFileWitness{}, false, err
	}
	afterIdentity, identityErr := projectionFileIdentity(file, after)
	if identityErr != nil {
		return projectionFileWitness{}, false, identityErr
	}
	if !after.Mode().IsRegular() || !pathAfter.Mode().IsRegular() || pathAfter.Mode()&os.ModeSymlink != 0 ||
		!os.SameFile(opened, after) || !os.SameFile(opened, pathAfter) ||
		after.Mode() != opened.Mode() || after.Size() != opened.Size() || !after.ModTime().Equal(opened.ModTime()) || afterIdentity != identity {
		return projectionFileWitness{}, false, fmt.Errorf("%s %s changed while being witnessed", kind, root.display(rel))
	}
	return projectionFileWitness{
		SHA256:      fmt.Sprintf("%x", hash.Sum(nil)),
		Size:        after.Size(),
		Mode:        uint32(after.Mode().Perm()),
		ModTimeNano: after.ModTime().UnixNano(),
		Identity:    identity,
	}, true, nil
}

func projectionFileMatches(root *designRoot, rel string, want projectionFileWitness, kind string) (bool, error) {
	got, exists, err := captureProjectionFile(root, rel, kind)
	return exists && got == want, err
}

func removeProjectionFileIfMatch(root *designRoot, hooks projectionTransactionHooks, rel string, want projectionFileWitness, kind string) error {
	matches, err := projectionFileMatches(root, rel, want, kind)
	if err != nil {
		return err
	}
	if !matches {
		return fmt.Errorf("%s %s no longer matches the exact transaction image; preserving it", kind, root.display(rel))
	}
	if hooks.beforeRemove != nil {
		if err := hooks.beforeRemove(root.display(rel)); err != nil {
			return err
		}
	}
	matches, err = projectionFileMatches(root, rel, want, kind)
	if err != nil {
		return err
	}
	if !matches {
		return fmt.Errorf("%s %s changed at the removal boundary; preserving it", kind, root.display(rel))
	}
	if hooks.authorize != nil {
		if err := hooks.authorize(); err != nil {
			return fmt.Errorf("checker projection transaction authority changed before removing %s: %w", root.display(rel), err)
		}
	}
	if err := root.root.Remove(rel); err != nil {
		return err
	}
	return root.syncDir(filepath.Dir(rel))
}

func restoreProjectionBackup(root *designRoot, hooks projectionTransactionHooks, entry resolvedProjectionTransactionEntry) error {
	if hooks.beforeRename != nil {
		if err := hooks.beforeRename(root.display(entry.backup), root.display(entry.target)); err != nil {
			return err
		}
	}
	if err := root.validateNoSymlink(entry.backup); err != nil {
		return err
	}
	if err := root.validateNoSymlink(filepath.Dir(entry.target)); err != nil {
		return err
	}
	if hooks.authorize != nil {
		if err := hooks.authorize(); err != nil {
			return fmt.Errorf("checker projection transaction authority changed before restoring %s: %w", root.display(entry.target), err)
		}
	}
	if err := root.root.Link(entry.backup, entry.target); err != nil {
		return fmt.Errorf("restore parked checker projection without replacing concurrent work: %w", err)
	}
	if err := root.syncDir(filepath.Dir(entry.target)); err != nil {
		return err
	}
	if matches, err := projectionFileMatches(root, entry.target, entry.Before, "restored checker projection"); err != nil || !matches {
		return errors.Join(err, fmt.Errorf("restored projection %s changed while being rebound; preserving target and backup", root.display(entry.target)))
	}
	return removeProjectionFileIfMatch(root, hooks, entry.backup, entry.Before, "parked checker projection")
}

func projectionTransactionControlNames(root *designRoot, control string) ([]string, error) {
	dir, err := root.root.Open(control)
	if err != nil {
		return nil, err
	}
	entries, readErr := dir.ReadDir(-1)
	closeErr := dir.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, name := range []string{projectionPreparedName, projectionBoundName, projectionCommittedName} {
		allowed[name] = true
		allowed[name+".new"] = true
		allowed[name+".retired"] = true
		allowed[name+".new.retired"] = true
	}
	prefix := strings.TrimSuffix(projectionPreparedName, ".json")
	names := make([]string, 0, len(allowed))
	folded := map[string]string{}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(strings.ToLower(name), prefix) {
			continue
		}
		if prior, exists := folded[strings.ToLower(name)]; exists {
			return nil, fmt.Errorf("checker projection transaction control paths %q and %q alias", prior, name)
		}
		folded[strings.ToLower(name)] = name
		if !allowed[name] {
			return nil, fmt.Errorf("unknown checker projection transaction control path %s", root.display(filepath.Join(control, name)))
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func captureProjectionControlSnapshot(root *designRoot, path string) (snapshot projectionControlSnapshot, retErr error) {
	if err := root.validateNoSymlink(path); err != nil {
		return projectionControlSnapshot{}, err
	}
	before, err := root.root.Lstat(path)
	if err != nil {
		return projectionControlSnapshot{}, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || !projectionControlPermissionsSafe(before.Mode()) {
		return projectionControlSnapshot{}, fmt.Errorf("checker projection transaction control %s must be a private regular, non-symlink file", root.display(path))
	}
	if before.Size() < 0 || before.Size() > projectionMaxControlRecord {
		return projectionControlSnapshot{}, fmt.Errorf("checker projection transaction control %s exceeds %d-byte limit", root.display(path), projectionMaxControlRecord)
	}
	file, err := root.root.Open(path)
	if err != nil {
		return projectionControlSnapshot{}, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil {
		return projectionControlSnapshot{}, err
	}
	identity, err := projectionFileIdentity(file, opened)
	if err != nil {
		return projectionControlSnapshot{}, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) || before.Mode() != opened.Mode() || before.Size() != opened.Size() ||
		!before.ModTime().Equal(opened.ModTime()) || projectionControlChangeID(before) != projectionControlChangeID(opened) {
		return projectionControlSnapshot{}, fmt.Errorf("checker projection transaction control %s changed identity while opening", root.display(path))
	}
	body, err := io.ReadAll(io.LimitReader(file, projectionMaxControlRecord+1))
	if err != nil {
		return projectionControlSnapshot{}, err
	}
	if int64(len(body)) != before.Size() {
		return projectionControlSnapshot{}, fmt.Errorf("checker projection transaction control %s changed size while reading", root.display(path))
	}
	openedAfter, statErr := file.Stat()
	pathAfter, pathErr := root.root.Lstat(path)
	if err := errors.Join(statErr, pathErr); err != nil {
		return projectionControlSnapshot{}, err
	}
	afterIdentity, err := projectionFileIdentity(file, openedAfter)
	if err != nil {
		return projectionControlSnapshot{}, err
	}
	if !openedAfter.Mode().IsRegular() || pathAfter.Mode()&os.ModeSymlink != 0 || !pathAfter.Mode().IsRegular() ||
		!os.SameFile(opened, openedAfter) || !os.SameFile(opened, pathAfter) || identity != afterIdentity ||
		opened.Mode() != openedAfter.Mode() || opened.Mode() != pathAfter.Mode() || opened.Size() != openedAfter.Size() || opened.Size() != pathAfter.Size() ||
		!opened.ModTime().Equal(openedAfter.ModTime()) || !opened.ModTime().Equal(pathAfter.ModTime()) ||
		projectionControlChangeID(opened) != projectionControlChangeID(openedAfter) || projectionControlChangeID(opened) != projectionControlChangeID(pathAfter) {
		return projectionControlSnapshot{}, fmt.Errorf("checker projection transaction control %s changed while being read", root.display(path))
	}
	return projectionControlSnapshot{
		path: path,
		body: append([]byte(nil), body...),
		witness: projectionFileWitness{
			SHA256:      fmt.Sprintf("%x", sha256.Sum256(body)),
			Size:        openedAfter.Size(),
			Mode:        uint32(openedAfter.Mode()),
			ModTimeNano: openedAfter.ModTime().UnixNano(),
			Identity:    identity,
		},
		change: projectionControlChangeID(openedAfter),
	}, nil
}

func projectionControlChangeID(info os.FileInfo) string {
	if info == nil || info.Sys() == nil {
		return ""
	}
	value := reflect.ValueOf(info.Sys())
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return ""
	}
	for _, name := range []string{"Ctim", "Ctimespec"} {
		field := value.FieldByName(name)
		if !field.IsValid() || field.Kind() != reflect.Struct {
			continue
		}
		sec, nsec := field.FieldByName("Sec"), field.FieldByName("Nsec")
		if sec.IsValid() && nsec.IsValid() && sec.CanInt() && nsec.CanInt() {
			return fmt.Sprintf("%d:%d", sec.Int(), nsec.Int())
		}
	}
	return ""
}

func sameProjectionControlSnapshot(first, second projectionControlSnapshot) bool {
	return first.path == second.path && first.witness == second.witness && first.change == second.change && bytes.Equal(first.body, second.body)
}

func sameProjectionControlObject(first, second projectionControlSnapshot) bool {
	// A pathname rename may update native change time. Stable file identity,
	// exact bytes, size, mode, and mtime bind the moved object; the post-rename
	// change witness becomes authoritative for subsequent revalidation.
	return first.witness == second.witness && bytes.Equal(first.body, second.body)
}

func cleanupIncompleteProjectionTransactionRecords(root *designRoot, control string, hooks projectionTransactionHooks) error {
	names, err := projectionTransactionControlNames(root, control)
	if err != nil {
		return err
	}
	present := make(map[string]bool, len(names))
	for _, name := range names {
		present[name] = true
	}
	for _, recordName := range []string{projectionPreparedName, projectionBoundName, projectionCommittedName} {
		incompleteName := recordName + ".new"
		retiredName := incompleteName + ".retired"
		if present[incompleteName] && present[retiredName] {
			return fmt.Errorf("dual incomplete checker projection transaction controls exist for %s", recordName)
		}
		name := incompleteName
		if present[retiredName] {
			name = retiredName
		} else if !present[incompleteName] {
			continue
		}
		path := filepath.Join(control, name)
		snapshot, err := captureProjectionControlSnapshot(root, path)
		if err != nil {
			return err
		}
		if name == incompleteName {
			if hooks.beforeRemove != nil {
				if err := hooks.beforeRemove(root.display(path)); err != nil {
					return err
				}
			}
			if hooks.beforeRename != nil {
				if err := hooks.beforeRename(root.display(path), root.display(filepath.Join(control, retiredName))); err != nil {
					return err
				}
			}
			current, err := captureProjectionControlSnapshot(root, path)
			if err != nil || !sameProjectionControlSnapshot(snapshot, current) {
				return errors.Join(err, fmt.Errorf("incomplete checker projection transaction control %s changed at the removal boundary; preserving it", root.display(path)))
			}
			retiredPath := filepath.Join(control, retiredName)
			if err := root.root.Rename(path, retiredPath); err != nil {
				return err
			}
			if err := root.syncDir(control); err != nil {
				return err
			}
			retired, err := captureProjectionControlSnapshot(root, retiredPath)
			if err != nil || !sameProjectionControlObject(snapshot, retired) {
				return errors.Join(err, fmt.Errorf("retired incomplete checker projection transaction control %s differs from its bound object; preserving it", root.display(retiredPath)))
			}
			path, snapshot = retiredPath, retired
		}
		if hooks.beforeRemove != nil {
			if err := hooks.beforeRemove(root.display(path)); err != nil {
				return err
			}
		}
		current, err := captureProjectionControlSnapshot(root, path)
		if err != nil || !sameProjectionControlSnapshot(snapshot, current) {
			return errors.Join(err, fmt.Errorf("retired incomplete checker projection transaction control %s changed at the deletion boundary; preserving it", root.display(path)))
		}
		if err := root.root.Remove(path); err != nil {
			return err
		}
		if err := root.syncDir(control); err != nil {
			return err
		}
	}
	return nil
}

func bindProjectionTransactionAuthority(root *designRoot, control string) (projectionTransactionAuthority, error) {
	names, err := projectionTransactionControlNames(root, control)
	if err != nil {
		return projectionTransactionAuthority{}, err
	}
	authority := projectionTransactionAuthority{control: control, snapshots: make(map[string]projectionControlSnapshot, len(names))}
	for _, name := range names {
		if strings.Contains(name, ".new") {
			return projectionTransactionAuthority{}, fmt.Errorf("incomplete checker projection transaction control appeared during authority binding: %s", root.display(filepath.Join(control, name)))
		}
		logicalName := strings.TrimSuffix(name, ".retired")
		if _, duplicate := authority.snapshots[logicalName]; duplicate {
			return projectionTransactionAuthority{}, fmt.Errorf("dual checker projection transaction controls exist for %s", logicalName)
		}
		snapshot, err := captureProjectionControlSnapshot(root, filepath.Join(control, name))
		if err != nil {
			return projectionTransactionAuthority{}, err
		}
		authority.snapshots[logicalName] = snapshot
		authority.retiring = authority.retiring || name != logicalName
	}
	if err := validateProjectionTransactionAuthority(root, authority); err != nil {
		return projectionTransactionAuthority{}, err
	}
	return authority, nil
}

func validateProjectionTransactionAuthority(root *designRoot, authority projectionTransactionAuthority) error {
	names, err := projectionTransactionControlNames(root, authority.control)
	if err != nil {
		return err
	}
	expectedByName := make(map[string]projectionControlSnapshot, len(authority.snapshots))
	for _, snapshot := range authority.snapshots {
		expectedByName[filepath.Base(snapshot.path)] = snapshot
	}
	if len(names) != len(expectedByName) {
		return fmt.Errorf("checker projection transaction control inventory changed")
	}
	for _, name := range names {
		expected, exists := expectedByName[name]
		if !exists || strings.Contains(name, ".new") {
			return fmt.Errorf("checker projection transaction control inventory changed at %s", root.display(filepath.Join(authority.control, name)))
		}
		current, err := captureProjectionControlSnapshot(root, expected.path)
		if err != nil {
			return err
		}
		if !sameProjectionControlSnapshot(expected, current) {
			return fmt.Errorf("checker projection transaction control %s changed since authority binding", root.display(expected.path))
		}
	}
	// Reverse revalidation binds entries that were enumerated before later
	// siblings and catches add/remove/ABA mutations behind the forward cursor.
	for i := len(names) - 1; i >= 0; i-- {
		expected := expectedByName[names[i]]
		current, err := captureProjectionControlSnapshot(root, expected.path)
		if err != nil {
			return err
		}
		if !sameProjectionControlSnapshot(expected, current) {
			return fmt.Errorf("checker projection transaction control %s changed since authority binding", root.display(expected.path))
		}
	}
	finalNames, err := projectionTransactionControlNames(root, authority.control)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(names, finalNames) {
		return fmt.Errorf("checker projection transaction control inventory changed while validating authority")
	}
	return nil
}

func removeProjectionTransactionRecords(root *designRoot, control string, hooks projectionTransactionHooks, authority projectionTransactionAuthority) error {
	// First atomically retire every remaining record. The presence of any
	// retired record is a durable witness that target recovery completed, so a
	// crash during retirement or deletion can only resume this cleanup phase.
	for _, name := range []string{projectionPreparedName, projectionBoundName, projectionCommittedName} {
		snapshot, exists := authority.snapshots[name]
		if !exists {
			continue
		}
		if strings.HasSuffix(snapshot.path, ".retired") {
			continue
		}
		if err := validateProjectionTransactionAuthority(root, authority); err != nil {
			return fmt.Errorf("checker projection transaction authority changed before retiring %s: %w", root.display(snapshot.path), err)
		}
		if hooks.beforeRemove != nil {
			if err := hooks.beforeRemove(root.display(snapshot.path)); err != nil {
				return err
			}
		}
		retiredPath := snapshot.path + ".retired"
		if hooks.beforeRename != nil {
			if err := hooks.beforeRename(root.display(snapshot.path), root.display(retiredPath)); err != nil {
				return err
			}
		}
		if err := validateProjectionTransactionAuthority(root, authority); err != nil {
			return fmt.Errorf("checker projection transaction authority changed at the retirement boundary for %s: %w", root.display(snapshot.path), err)
		}
		if err := root.root.Rename(snapshot.path, retiredPath); err != nil {
			return err
		}
		if err := root.syncDir(control); err != nil {
			return err
		}
		retired, err := captureProjectionControlSnapshot(root, retiredPath)
		if err != nil || !sameProjectionControlObject(snapshot, retired) {
			return errors.Join(err, fmt.Errorf("retired checker projection transaction control %s differs from its bound authority; preserving it", root.display(retiredPath)))
		}
		authority.snapshots[name] = retired
		authority.retiring = true
	}
	if err := validateProjectionTransactionAuthority(root, authority); err != nil {
		return fmt.Errorf("checker projection transaction authority changed after retirement: %w", err)
	}
	// Retired names are isolated from the live control protocol. Conditional
	// deletion preserves any object replaced after retirement.
	for _, name := range []string{projectionPreparedName, projectionBoundName, projectionCommittedName} {
		snapshot, exists := authority.snapshots[name]
		if !exists {
			continue
		}
		if !strings.HasSuffix(snapshot.path, ".retired") {
			return fmt.Errorf("checker projection transaction control %s was not retired", root.display(snapshot.path))
		}
		if hooks.beforeRemove != nil {
			if err := hooks.beforeRemove(root.display(snapshot.path)); err != nil {
				return err
			}
		}
		if err := validateProjectionTransactionAuthority(root, authority); err != nil {
			return fmt.Errorf("checker projection transaction authority changed at the deletion boundary for %s: %w", root.display(snapshot.path), err)
		}
		if err := root.root.Remove(snapshot.path); err != nil {
			return err
		}
		if err := root.syncDir(control); err != nil {
			return err
		}
		delete(authority.snapshots, name)
	}
	return removeProjectionControlDirIfEmpty(root, control)
}

func removeProjectionControlDirIfEmpty(root *designRoot, control string) error {
	err := root.root.Remove(control)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		// A non-empty control directory is retryable state, not a cleanup error.
		dir, openErr := root.root.Open(control)
		if openErr == nil {
			entries, readErr := dir.ReadDir(1)
			closeErr := dir.Close()
			if readErr == nil && closeErr == nil && len(entries) > 0 {
				return nil
			}
		}
		return err
	}
	if err := root.syncDir("."); err != nil {
		// Keep an explicit retry point when the parent-directory durability
		// barrier fails after removal. Recovery will see the empty private
		// directory and retry the removal plus parent sync on the next run.
		recreateErr := root.root.Mkdir(control, 0o700)
		if recreateErr != nil && !os.IsExist(recreateErr) {
			return errors.Join(err, fmt.Errorf("restore checker transaction cleanup retry point: %w", recreateErr))
		}
		return err
	}
	return nil
}

func cleanDesignRoot(design string) (string, error) {
	abs, err := filepath.Abs(design)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("checker design root %s must be a real directory", abs)
	}
	return filepath.Clean(abs), nil
}

func ensureProjectionControlDir(root *designRoot) (string, error) {
	control := projectionControlDirName
	if err := root.root.Mkdir(control, 0o700); err != nil && !os.IsExist(err) {
		return "", err
	}
	if err := root.validateNoSymlink(control); err != nil {
		return "", err
	}
	info, err := root.root.Lstat(control)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || !projectionControlPermissionsSafe(info.Mode()) {
		return "", fmt.Errorf("checker projection transaction control %s must be a private real directory", root.display(control))
	}
	if err := root.syncDir("."); err != nil {
		return "", err
	}
	return control, nil
}

func createProjectionDirectoryTree(root *designRoot, dir string) error {
	current := ""
	for _, component := range strings.Split(dir, string(os.PathSeparator)) {
		if component == "." || component == "" {
			continue
		}
		parent := current
		if current == "" {
			current = component
		} else {
			current = filepath.Join(current, component)
		}
		info, err := root.root.Lstat(current)
		if os.IsNotExist(err) {
			if err := root.root.Mkdir(current, 0o755); err != nil {
				return err
			}
			if parent == "" {
				parent = "."
			}
			if err := root.syncDir(parent); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("projection directory component %s must be a real directory", root.display(current))
		}
	}
	return nil
}

func projectionPlanTarget(root *designRoot, dest string) (string, string, error) {
	rel, err := root.rel(dest)
	if err != nil || rel == "." {
		return "", "", fmt.Errorf("projection target %s escapes design %s", dest, root.abs)
	}
	return rel, rel, nil
}

func writeProjectionTransactionRecord(root *designRoot, path string, record projectionTransactionRecord) error {
	if exists, err := root.lstatRegular(path, "checker projection transaction control", true); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("checker projection transaction record already exists: %s", root.display(path))
	}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmpPath := path + ".new"
	tmp, err := root.root.OpenFile(tmpPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = root.root.Remove(tmpPath) }()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := renameProjectionPathRoot(root, projectionTransactionHooks{}, tmpPath, path); err != nil {
		return err
	}
	return root.syncDir(filepath.Dir(path))
}

func loadProjectionTransactionRecord(snapshot projectionControlSnapshot, phase string) (projectionTransactionRecord, error) {
	var record projectionTransactionRecord
	if err := decodeStrictJSON(snapshot.body, &record); err != nil {
		return projectionTransactionRecord{}, fmt.Errorf("parse checker projection transaction %s: %w", snapshot.path, err)
	}
	if record.Version != projectionTransactionVersion || record.Phase != phase || len(record.Entries) == 0 {
		return projectionTransactionRecord{}, fmt.Errorf("invalid checker projection transaction %s", snapshot.path)
	}
	return record, nil
}

func validateProjectionTransactionRecord(root *designRoot, record projectionTransactionRecord, phase string) ([]resolvedProjectionTransactionEntry, error) {
	if record.Version != projectionTransactionVersion || record.Phase != phase || len(record.Entries) == 0 {
		return nil, fmt.Errorf("invalid checker projection transaction record")
	}
	seen := map[string]string{}
	resolved := make([]resolvedProjectionTransactionEntry, 0, len(record.Entries))
	priorTarget := ""
	for _, entry := range record.Entries {
		if entry.Target <= priorTarget {
			return nil, fmt.Errorf("checker projection transaction entries are not in canonical target order")
		}
		priorTarget = entry.Target
		paths := []struct {
			kind string
			rel  string
		}{
			{"target", entry.Target}, {"stage", entry.Stage}, {"backup", entry.Backup},
		}
		rooted := make(map[string]string, 3)
		for _, item := range paths {
			if err := validatePortableRelativePath(item.rel); err != nil {
				return nil, fmt.Errorf("checker projection transaction %s %q is unsafe: %w", item.kind, item.rel, err)
			}
			folded := strings.ToLower(filepath.ToSlash(filepath.Clean(item.rel)))
			if prior, ok := seen[folded]; ok {
				return nil, fmt.Errorf("checker projection transaction paths %q and %q alias", prior, item.rel)
			}
			seen[folded] = item.rel
			rel := filepath.Clean(filepath.FromSlash(item.rel))
			if err := root.validateNoSymlink(rel); err != nil {
				return nil, fmt.Errorf("checker projection transaction path %q is unsafe: %w", item.rel, err)
			}
			rooted[item.kind] = rel
		}
		targetParts := strings.Split(entry.Target, "/")
		if len(targetParts) < 3 || targetParts[0] != "checkers" {
			return nil, fmt.Errorf("checker projection transaction target %q is outside a checker-owned directory", entry.Target)
		}
		if err := validatePortableComponent(targetParts[1]); err != nil {
			return nil, fmt.Errorf("checker projection transaction target %q has invalid checker owner: %w", entry.Target, err)
		}
		digest := fmt.Sprintf("%x", sha256.Sum256([]byte(entry.Target)))[:16]
		wantStage := filepath.ToSlash(filepath.Join(filepath.Dir(filepath.FromSlash(entry.Target)), ".machinery-project-stage-"+digest))
		wantBackup := filepath.ToSlash(filepath.Join(filepath.Dir(filepath.FromSlash(entry.Target)), ".machinery-project-backup-"+digest))
		if entry.Stage != wantStage || entry.Backup != wantBackup {
			return nil, fmt.Errorf("checker projection transaction transient paths do not belong to target %q", entry.Target)
		}
		if entry.Existed {
			if !validProjectionFileWitness(entry.Before, true) {
				return nil, fmt.Errorf("checker projection transaction target %q has an invalid exact pre-image witness", entry.Target)
			}
		} else if entry.Before != (projectionFileWitness{}) {
			return nil, fmt.Errorf("checker projection transaction target %q has a pre-image witness despite being absent", entry.Target)
		}
		exactAfter := phase == "bound" || phase == "committed"
		if !validProjectionFileWitness(entry.After, exactAfter) {
			return nil, fmt.Errorf("checker projection transaction target %q has an invalid post-image witness", entry.Target)
		}
		resolved = append(resolved, resolvedProjectionTransactionEntry{
			projectionTransactionEntry: entry,
			target:                     rooted["target"],
			stage:                      rooted["stage"],
			backup:                     rooted["backup"],
		})
	}
	return resolved, nil
}

func sameProjectionTransaction(prepared, committed projectionTransactionRecord) bool {
	if prepared.Version != committed.Version || len(prepared.Entries) != len(committed.Entries) {
		return false
	}
	for i := range prepared.Entries {
		if prepared.Entries[i] != committed.Entries[i] {
			return false
		}
	}
	return true
}

func sameProjectionIntent(prepared, later projectionTransactionRecord) bool {
	if prepared.Version != later.Version || len(prepared.Entries) != len(later.Entries) {
		return false
	}
	for i := range prepared.Entries {
		first, second := prepared.Entries[i], later.Entries[i]
		if first.Target != second.Target || first.Stage != second.Stage || first.Backup != second.Backup || first.Existed != second.Existed || first.Before != second.Before ||
			first.After.SHA256 != second.After.SHA256 || first.After.Size != second.After.Size {
			return false
		}
	}
	return true
}

func validProjectionFileWitness(witness projectionFileWitness, exact bool) bool {
	if len(witness.SHA256) != sha256.Size*2 || witness.Size < 0 {
		return false
	}
	for _, char := range witness.SHA256 {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	if !exact {
		return witness.Mode == 0 && witness.ModTimeNano == 0 && witness.Identity == ""
	}
	if witness.Mode > 0o777 || witness.Identity == "" || len(witness.Identity) > 128 {
		return false
	}
	for _, char := range witness.Identity {
		if (char < '0' || char > '9') && char != ':' {
			return false
		}
	}
	return true
}
