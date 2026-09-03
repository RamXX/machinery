package gates

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/filelock"
	"github.com/RamXX/machinery/internal/portablepath"
)

const (
	embedTxJournalName = ".machinery-embed-refresh-transaction.json"
	embedTxStageName   = ".machinery-embed-refresh-transaction.stage"
	embedTxVersion     = 1
	embedTxPrepared    = "prepared"
	embedTxCommitted   = "committed"
	embedTxMaxJournal  = 64 << 20
	embedTxNewPrefix   = ".machinery-embed-new-"
	embedTxOldPrefix   = ".machinery-embed-old-"
)

// embedTransactionPoint is a test-only power-loss/error boundary. A returned
// error takes the ordinary rollback path. A panic models process death and
// deliberately leaves the durable journal and outer publication sentinel.
var embedTransactionPoint = func(string) error { return nil }

type embedTxItem struct {
	Path     string `json:"path"`
	Temp     string `json:"temp"`
	Backup   string `json:"backup"`
	Old      []byte `json:"old"`
	New      []byte `json:"new"`
	OldHash  string `json:"old_hash"`
	NewHash  string `json:"new_hash"`
	OldMode  uint32 `json:"old_mode"`
	NewMode  uint32 `json:"new_mode"`
	Deletion bool   `json:"deletion"`
}

type embedTxJournal struct {
	Version   int           `json:"version"`
	Operation string        `json:"operation"`
	Phase     string        `json:"phase"`
	Scope     string        `json:"scope"`
	Items     []embedTxItem `json:"items"`
	Checksum  string        `json:"checksum"`
}

type embedTxPayload struct {
	Version   int           `json:"version"`
	Operation string        `json:"operation"`
	Phase     string        `json:"phase"`
	Scope     string        `json:"scope"`
	Items     []embedTxItem `json:"items"`
}

type embedRootTransaction struct {
	design string
	scope  string
	root   *os.Root
}

func openEmbedRootTransaction(design string) (*embedRootTransaction, error) {
	abs, err := filepath.Abs(design)
	if err != nil {
		return nil, fmt.Errorf("resolve embed transaction root: %w", err)
	}
	before, err := os.Lstat(abs)
	if err != nil {
		return nil, fmt.Errorf("inspect embed transaction root: %w", err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, fmt.Errorf("embed transaction root must be a real directory")
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, fmt.Errorf("open embed transaction root: %w", err)
	}
	after, err := root.Lstat(".")
	if err != nil || !os.SameFile(before, after) {
		_ = root.Close()
		return nil, fmt.Errorf("embed transaction root changed identity while opening")
	}
	scope, err := filelock.ScopeIdentity(abs)
	if err != nil {
		_ = root.Close()
		return nil, fmt.Errorf("resolve embed transaction scope: %w", err)
	}
	return &embedRootTransaction{design: abs, scope: scope, root: root}, nil
}

func openEmbedRootTransactionRetained(design string, root *os.Root) (*embedRootTransaction, error) {
	abs, err := filepath.Abs(design)
	if err != nil {
		return nil, fmt.Errorf("resolve embed transaction root: %w", err)
	}
	inside, err := root.Lstat(".")
	if err != nil || !inside.IsDir() {
		return nil, fmt.Errorf("inspect retained embed transaction root")
	}
	scope, err := filelock.ScopeIdentity(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve embed transaction scope: %w", err)
	}
	return &embedRootTransaction{design: abs, scope: scope, root: root}, nil
}

func (tx *embedRootTransaction) withRoot(root *os.Root, fn func() error) error {
	prior := tx.root
	tx.root = root
	defer func() { tx.root = prior }()
	return fn()
}

func (tx *embedRootTransaction) Close() error { return tx.root.Close() }

func (tx *embedRootTransaction) pending() (embedTxJournal, bool, error) {
	residue, err := tx.embedResiduePaths()
	if err != nil {
		return embedTxJournal{}, false, err
	}
	hasJournal, hasStage := false, false
	for _, rel := range residue {
		hasJournal = hasJournal || rel == embedTxJournalName
		hasStage = hasStage || rel == embedTxStageName
	}
	if !hasJournal && !hasStage && len(residue) != 0 {
		return embedTxJournal{}, false, fmt.Errorf("embed transaction scratch exists without its root journal: %s", strings.Join(residue, ", "))
	}
	if !hasJournal && !hasStage {
		return embedTxJournal{}, false, nil
	}
	var current, staged embedTxJournal
	if hasJournal {
		current, err = tx.readJournalFile(embedTxJournalName)
		if err != nil {
			return embedTxJournal{}, false, err
		}
	}
	journal := current
	if hasStage {
		staged, err = tx.readJournalFile(embedTxStageName)
		if err != nil {
			return embedTxJournal{}, false, err
		}
		if hasJournal && (!embedTxSamePlan(current, staged) || (current.Phase == embedTxCommitted && staged.Phase != embedTxCommitted)) {
			return embedTxJournal{}, false, fmt.Errorf("embed refresh journal stage has a different plan or regresses phase")
		}
		journal = staged
	}
	if journal.Scope != tx.scope {
		return embedTxJournal{}, false, fmt.Errorf("embed refresh journal belongs to foreign design scope %q, not %q", journal.Scope, tx.scope)
	}
	if err := tx.validatePhysicalInventory(journal, hasStage); err != nil {
		return embedTxJournal{}, false, err
	}
	return journal, true, nil
}

func (tx *embedRootTransaction) readJournalFile(name string) (embedTxJournal, error) {
	info, err := tx.root.Lstat(name)
	if err != nil {
		return embedTxJournal{}, fmt.Errorf("inspect embed refresh journal %s: %w", name, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > embedTxMaxJournal {
		return embedTxJournal{}, fmt.Errorf("embed refresh journal %s must be a bounded regular file, not a symlink or special file", name)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		return embedTxJournal{}, fmt.Errorf("embed refresh journal %s must have mode 0600", name)
	}
	f, err := tx.root.Open(name)
	if err != nil {
		return embedTxJournal{}, fmt.Errorf("open embed refresh journal %s: %w", name, err)
	}
	opened, statErr := f.Stat()
	if statErr != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		_ = f.Close()
		return embedTxJournal{}, fmt.Errorf("embed refresh journal %s changed identity while opening", name)
	}
	body, readErr := io.ReadAll(io.LimitReader(f, embedTxMaxJournal+1))
	closeErr := f.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return embedTxJournal{}, fmt.Errorf("read embed refresh journal %s: %w", name, err)
	}
	journal, err := decodeEmbedTxJournal(body)
	if err != nil {
		return embedTxJournal{}, fmt.Errorf("invalid embed refresh journal %s: %w", name, err)
	}
	return journal, nil
}

func (tx *embedRootTransaction) reconcileJournalStage(hasJournal bool) error {
	stageState, err := tx.readState(embedTxStageName)
	if err != nil {
		return err
	}
	if !stageState.exists {
		return fmt.Errorf("recoverable embed refresh journal stage disappeared")
	}
	staged, err := tx.readJournalFile(embedTxStageName)
	if err != nil {
		return err
	}
	if staged.Scope != tx.scope {
		return fmt.Errorf("embed refresh journal stage belongs to foreign design scope %q", staged.Scope)
	}
	journalState := embedFileState{}
	if hasJournal {
		journalState, err = tx.readState(embedTxJournalName)
		if err != nil {
			return err
		}
		if !journalState.exists {
			return fmt.Errorf("embed refresh journal disappeared before stage reconciliation")
		}
		current, err := tx.readJournalFile(embedTxJournalName)
		if err != nil {
			return err
		}
		if !embedTxSamePlan(current, staged) || (current.Phase == embedTxCommitted && staged.Phase != embedTxCommitted) {
			return fmt.Errorf("embed refresh journal stage has a different plan or regresses phase")
		}
	}
	if err := embedTransactionPoint("reconcile-before-journal-promote"); err != nil {
		return err
	}
	if err := tx.revalidateState(embedTxStageName, stageState); err != nil {
		return fmt.Errorf("embed refresh journal stage changed before reconciliation: %w", err)
	}
	if err := tx.revalidateState(embedTxJournalName, journalState); err != nil {
		return fmt.Errorf("embed refresh journal changed before stage reconciliation: %w", err)
	}
	if err := tx.root.Rename(embedTxStageName, embedTxJournalName); err != nil {
		return fmt.Errorf("promote recoverable embed refresh journal stage: %w", err)
	}
	return tx.syncDir(".")
}

func embedTxSamePlan(a, b embedTxJournal) bool {
	return a.Version == b.Version && a.Operation == b.Operation && a.Scope == b.Scope && reflect.DeepEqual(a.Items, b.Items)
}

func newEmbedTxJournal(scope string, items []embedTxItem) (embedTxJournal, error) {
	journal := embedTxJournal{Version: embedTxVersion, Operation: "embed-refresh", Phase: embedTxPrepared, Scope: scope, Items: append([]embedTxItem(nil), items...)}
	for i := range journal.Items {
		item := &journal.Items[i]
		item.Path = filepath.ToSlash(item.Path)
		dir := path.Dir(item.Path)
		if dir == "." {
			dir = ""
		}
		token := fmt.Sprintf("%06d", i)
		item.Temp = path.Join(dir, embedTxNewPrefix+token)
		item.Backup = path.Join(dir, embedTxOldPrefix+token)
		item.OldHash = embedTxHash(item.Old)
		item.NewHash = embedTxHash(item.New)
	}
	journal.Checksum = embedTxChecksum(journal)
	if err := validateEmbedTxJournal(journal); err != nil {
		return embedTxJournal{}, err
	}
	if _, err := encodeEmbedTxJournal(journal); err != nil {
		return embedTxJournal{}, err
	}
	return journal, nil
}

func (tx *embedRootTransaction) commit(journal embedTxJournal) error {
	if journal.Scope != tx.scope {
		return fmt.Errorf("refuse embed transaction for foreign design scope")
	}
	if err := tx.preflightOriginal(journal); err != nil {
		return err
	}
	if err := tx.persistJournal(journal); err != nil {
		return err
	}
	if err := embedTransactionPoint("prepared-journal"); err != nil {
		return tx.rollbackAfterError(journal, err)
	}
	if err := tx.forwardPrepared(journal, true); err != nil {
		return tx.rollbackAfterError(journal, err)
	}
	journal.Phase = embedTxCommitted
	if err := tx.persistJournal(journal); err != nil {
		// Persistence may already have durably installed the committed stage or
		// journal. Leave the outer sentinel and exact plan for restart recovery;
		// rolling back here could contradict the higher durable phase.
		return err
	}
	if err := embedTransactionPoint("committed-journal"); err != nil {
		return err
	}
	return tx.finalizeAndMarkBoundary(journal)
}

func (tx *embedRootTransaction) recover(journal embedTxJournal) error {
	if journal.Scope != tx.scope {
		return fmt.Errorf("refuse recovery for foreign embed refresh journal")
	}
	if _, err := tx.root.Lstat(embedTxStageName); err == nil {
		_, currentErr := tx.root.Lstat(embedTxJournalName)
		if currentErr != nil && !errors.Is(currentErr, fs.ErrNotExist) {
			return currentErr
		}
		if err := tx.reconcileJournalStage(currentErr == nil); err != nil {
			return err
		}
		var readErr error
		journal, readErr = tx.readJournalFile(embedTxJournalName)
		if readErr != nil {
			return readErr
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if journal.Phase == embedTxPrepared {
		if err := tx.forwardPrepared(journal, false); err != nil {
			return fmt.Errorf("replay original embed refresh plan: %w", err)
		}
		journal.Phase = embedTxCommitted
		if err := tx.persistJournal(journal); err != nil {
			return fmt.Errorf("persist replayed embed refresh plan: %w", err)
		}
	}
	return tx.finalizeAndMarkBoundary(journal)
}

// finalizeAndMarkBoundary exposes the last inner/outer durability boundary to
// power-loss tests. Once finalize returns, every target is exact and the inner
// journal is gone, but the enclosing design publication sentinel has not yet
// been validated and cleared.
func (tx *embedRootTransaction) finalizeAndMarkBoundary(journal embedTxJournal) error {
	if err := tx.finalize(journal); err != nil {
		return err
	}
	return embedTransactionPoint("finalized-before-publication-clear")
}

func (tx *embedRootTransaction) preflightOriginal(journal embedTxJournal) error {
	for _, item := range journal.Items {
		if err := tx.validateParent(item.Path); err != nil {
			return err
		}
		if err := tx.ensureReservedAbsent(item.Temp, item.Backup); err != nil {
			return err
		}
		state, err := tx.readState(item.Path)
		if err != nil {
			return err
		}
		if !state.exists || state.hash != item.OldHash || state.mode != fs.FileMode(item.OldMode) {
			return fmt.Errorf("embed refresh target %s changed after immutable planning", item.Path)
		}
	}
	return nil
}

func (tx *embedRootTransaction) ensureReservedAbsent(paths ...string) error {
	for _, rel := range paths {
		if _, err := tx.root.Lstat(rel); err == nil {
			return fmt.Errorf("reserved embed transaction path already exists: %s", rel)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("inspect reserved embed transaction path %s: %w", rel, err)
		}
	}
	return nil
}

func (tx *embedRootTransaction) forwardPrepared(journal embedTxJournal, inject bool) error {
	for first := 0; first < len(journal.Items); {
		dir := path.Dir(journal.Items[first].Path)
		last := first
		for last < len(journal.Items) && path.Dir(journal.Items[last].Path) == dir {
			if err := tx.forwardItem(journal.Items[last], inject); err != nil {
				return err
			}
			last++
		}
		if inject {
			if err := embedTransactionPoint("directory:" + dir); err != nil {
				return err
			}
		}
		first = last
	}
	return nil
}

func (tx *embedRootTransaction) forwardItem(item embedTxItem, inject bool) error {
	target, err := tx.readState(item.Path)
	if err != nil {
		return err
	}
	temp, err := tx.readState(item.Temp)
	if err != nil {
		return err
	}
	backup, err := tx.readState(item.Backup)
	if err != nil {
		return err
	}
	if temp.exists && (temp.hash != item.NewHash || temp.mode != fs.FileMode(item.NewMode)) {
		return fmt.Errorf("embed transaction temp %s has unexpected bytes or mode", item.Temp)
	}
	if backup.exists && (backup.hash != item.OldHash || backup.mode != fs.FileMode(item.OldMode)) {
		return fmt.Errorf("embed transaction backup %s has unexpected bytes or mode", item.Backup)
	}
	// A power loss after the durable link and before unlinking the source can
	// leave both names bound to the same exact original inode. Complete that
	// no-clobber move rather than treating the witnessed intermediate state as
	// an ambiguous external edit.
	if target.exists && backup.exists && target.hash == item.OldHash && target.mode == fs.FileMode(item.OldMode) && sameEmbedFileIdentity(target, backup) {
		if err := tx.removeExact(item.Path, target, "forward-before-linked-source-remove:"+item.Path, "linked original embed "+item.Path); err != nil {
			return err
		}
		if err := tx.syncDir(path.Dir(item.Path)); err != nil {
			return err
		}
		target = embedFileState{}
	}
	switch {
	case target.exists && target.hash == item.NewHash && target.mode == fs.FileMode(item.NewMode):
		if !backup.exists {
			return fmt.Errorf("embed target %s is new but its original backup is absent", item.Path)
		}
		if temp.exists {
			if err := tx.removeExact(item.Temp, temp, "forward-before-temp-remove:"+item.Temp, "redundant embed transaction temp "+item.Temp); err != nil {
				return err
			}
			return tx.syncDir(path.Dir(item.Path))
		}
		return nil
	case target.exists && target.hash == item.OldHash && target.mode == fs.FileMode(item.OldMode):
		if backup.exists {
			return fmt.Errorf("embed target %s and its original backup both exist", item.Path)
		}
		if !temp.exists && !item.Deletion {
			if err := tx.writeTemp(item.Temp, item.New, fs.FileMode(item.NewMode)); err != nil {
				return err
			}
			if inject {
				if err := embedTransactionPoint("stage:" + item.Temp); err != nil {
					return err
				}
			}
		}
		if err := tx.renameExact(item.Path, item.Backup, target, "forward-before-backup:"+item.Path, "original embed "+item.Path); err != nil {
			return fmt.Errorf("park original embed %s: %w", item.Path, err)
		}
		if err := tx.syncDir(path.Dir(item.Path)); err != nil {
			return err
		}
		if item.Deletion {
			return nil
		}
		temp, err = tx.readState(item.Temp)
		if err != nil {
			return err
		}
		if !temp.exists || temp.hash != item.NewHash || temp.mode != fs.FileMode(item.NewMode) {
			return fmt.Errorf("embed transaction temp %s changed before install", item.Temp)
		}
		if err := tx.renameExact(item.Temp, item.Path, temp, "forward-before-install:"+item.Path, "refreshed embed "+item.Path); err != nil {
			return fmt.Errorf("install refreshed embed %s: %w", item.Path, err)
		}
		return tx.syncDir(path.Dir(item.Path))
	case !target.exists && backup.exists:
		if item.Deletion {
			return nil
		}
		if !temp.exists {
			if err := tx.writeTemp(item.Temp, item.New, fs.FileMode(item.NewMode)); err != nil {
				return err
			}
			if inject {
				if err := embedTransactionPoint("stage:" + item.Temp); err != nil {
					return err
				}
			}
		}
		temp, err = tx.readState(item.Temp)
		if err != nil {
			return err
		}
		if !temp.exists || temp.hash != item.NewHash || temp.mode != fs.FileMode(item.NewMode) {
			return fmt.Errorf("embed transaction temp %s changed before install", item.Temp)
		}
		if err := tx.renameExact(item.Temp, item.Path, temp, "forward-before-install:"+item.Path, "refreshed embed "+item.Path); err != nil {
			return fmt.Errorf("finish refreshed embed %s: %w", item.Path, err)
		}
		return tx.syncDir(path.Dir(item.Path))
	default:
		return fmt.Errorf("embed transaction target %s has ambiguous or externally modified state", item.Path)
	}
}

func (tx *embedRootTransaction) rollbackAfterError(journal embedTxJournal, cause error) error {
	if rollbackErr := tx.rollback(journal); rollbackErr != nil {
		return fmt.Errorf("embed refresh transaction failed: %w; durable rollback failed: %w", cause, rollbackErr)
	}
	return cause
}

func (tx *embedRootTransaction) rollback(journal embedTxJournal) error {
	dirs := map[string]bool{}
	for i := len(journal.Items) - 1; i >= 0; i-- {
		item := journal.Items[i]
		dir := path.Dir(item.Path)
		dirs[dir] = true
		target, err := tx.readState(item.Path)
		if err != nil {
			return err
		}
		backup, err := tx.readState(item.Backup)
		if err != nil {
			return err
		}
		if backup.exists {
			if target.exists && target.hash == item.OldHash && target.mode == fs.FileMode(item.OldMode) && sameEmbedFileIdentity(target, backup) {
				if err := tx.removeExact(item.Backup, backup, "rollback-before-linked-backup-remove:"+item.Backup, "linked embed backup "+item.Backup); err != nil {
					return err
				}
				backup = embedFileState{}
			} else {
				if target.exists {
					if target.hash != item.NewHash || target.mode != fs.FileMode(item.NewMode) {
						return fmt.Errorf("cannot roll back externally modified embed %s", item.Path)
					}
					if err := tx.removeExact(item.Path, target, "rollback-before-target-remove:"+item.Path, "refreshed embed "+item.Path); err != nil {
						return err
					}
				}
				if err := tx.renameExact(item.Backup, item.Path, backup, "rollback-before-backup-restore:"+item.Path, "original embed "+item.Path); err != nil {
					return fmt.Errorf("restore original embed %s: %w", item.Path, err)
				}
			}
		} else if !target.exists || target.hash != item.OldHash || target.mode != fs.FileMode(item.OldMode) {
			return fmt.Errorf("cannot roll back embed %s without its exact original", item.Path)
		}
		if temp, err := tx.readState(item.Temp); err != nil {
			return err
		} else if temp.exists {
			if temp.hash != item.NewHash || temp.mode != fs.FileMode(item.NewMode) {
				return fmt.Errorf("cannot remove externally modified embed temp %s", item.Temp)
			}
			if err := tx.removeExact(item.Temp, temp, "rollback-before-temp-remove:"+item.Temp, "embed transaction temp "+item.Temp); err != nil {
				return err
			}
		}
	}
	for _, dir := range sortedEmbedDirs(dirs) {
		if err := tx.syncDir(dir); err != nil {
			return err
		}
	}
	return tx.removeJournal(journal)
}

func (tx *embedRootTransaction) finalize(journal embedTxJournal) error {
	dirs := map[string]bool{}
	for _, item := range journal.Items {
		target, err := tx.readState(item.Path)
		if err != nil {
			return err
		}
		if item.Deletion {
			if target.exists {
				return fmt.Errorf("committed embed deletion target still exists: %s", item.Path)
			}
		} else if !target.exists || target.hash != item.NewHash || target.mode != fs.FileMode(item.NewMode) {
			return fmt.Errorf("committed embed target %s does not match journal", item.Path)
		}
		for _, residue := range []struct {
			path string
			hash string
			mode fs.FileMode
			kind string
		}{
			{path: item.Temp, hash: item.NewHash, mode: fs.FileMode(item.NewMode), kind: "temp"},
			{path: item.Backup, hash: item.OldHash, mode: fs.FileMode(item.OldMode), kind: "backup"},
		} {
			state, err := tx.readState(residue.path)
			if err != nil {
				return err
			}
			if !state.exists {
				continue
			}
			if state.hash != residue.hash || state.mode != residue.mode {
				return fmt.Errorf("embed transaction %s %s changed after commit; preserving it", residue.kind, residue.path)
			}
			if err := embedTransactionPoint("finalize-before-residue-remove:" + residue.path); err != nil {
				return err
			}
			if err := tx.revalidateCommittedTarget(item, target); err != nil {
				return err
			}
			if err := tx.removeExact(residue.path, state, "", "embed transaction residue "+residue.path); err != nil {
				return err
			}
		}
		dirs[path.Dir(item.Path)] = true
	}
	for _, dir := range sortedEmbedDirs(dirs) {
		if err := tx.syncDir(dir); err != nil {
			return err
		}
	}
	return tx.removeJournal(journal)
}

func validateEmbedJournalTarget(phase string, item embedTxItem, state embedFileState) error {
	wantHash, wantMode, wantExists := item.OldHash, fs.FileMode(item.OldMode), true
	if phase == embedTxCommitted {
		wantHash, wantMode, wantExists = item.NewHash, fs.FileMode(item.NewMode), !item.Deletion
	}
	if state.exists != wantExists {
		return fmt.Errorf("embed target %s does not match %s journal before removal", item.Path, phase)
	}
	if wantExists && (state.hash != wantHash || state.mode != wantMode) {
		return fmt.Errorf("embed target %s changed before %s journal removal; preserving journal", item.Path, phase)
	}
	return nil
}

func (tx *embedRootTransaction) revalidateCommittedTarget(item embedTxItem, expected embedFileState) error {
	if err := tx.revalidateState(item.Path, expected); err != nil {
		return fmt.Errorf("committed embed target changed before residue removal: %w", err)
	}
	return nil
}

func (tx *embedRootTransaction) removeJournal(journal embedTxJournal) error {
	state, err := tx.readState(embedTxJournalName)
	if err != nil {
		return err
	}
	if !state.exists {
		return tx.syncDir(".")
	}
	body, err := encodeEmbedTxJournal(journal)
	if err != nil {
		return err
	}
	if state.hash != embedTxHash(body) || state.mode != 0o600 {
		return fmt.Errorf("embed refresh journal changed before removal; preserving it")
	}
	targets := make([]embedFileState, len(journal.Items))
	for i, item := range journal.Items {
		target, err := tx.readState(item.Path)
		if err != nil {
			return err
		}
		if err := validateEmbedJournalTarget(journal.Phase, item, target); err != nil {
			return err
		}
		targets[i] = target
	}
	if err := embedTransactionPoint("finalize-before-journal-remove"); err != nil {
		return err
	}
	for i, item := range journal.Items {
		if err := tx.revalidateState(item.Path, targets[i]); err != nil {
			return fmt.Errorf("embed target %s changed before journal removal; preserving journal: %w", item.Path, err)
		}
	}
	if err := tx.removeExact(embedTxJournalName, state, "", "embed refresh journal"); err != nil {
		return err
	}
	return tx.syncDir(".")
}

func (tx *embedRootTransaction) persistJournal(journal embedTxJournal) error {
	body, err := encodeEmbedTxJournal(journal)
	if err != nil {
		return err
	}
	if _, err := tx.root.Lstat(embedTxStageName); err == nil {
		return fmt.Errorf("embed refresh journal stage already exists; preserving it for recovery")
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	f, err := tx.root.OpenFile(embedTxStageName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create embed refresh journal stage: %w", err)
	}
	writeErr := func() error {
		if err := f.Chmod(0o600); err != nil {
			return err
		}
		if _, err := f.Write(body); err != nil {
			return err
		}
		return f.Sync()
	}()
	closeErr := f.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return fmt.Errorf("persist embed refresh journal stage: %w", err)
	}
	if err := embedTransactionPoint("journal-stage-synced:" + journal.Phase); err != nil {
		return err
	}
	if err := tx.syncDir("."); err != nil {
		return err
	}
	if err := embedTransactionPoint("journal-stage-dir-synced:" + journal.Phase); err != nil {
		return err
	}
	stageState, err := tx.readState(embedTxStageName)
	if err != nil {
		return err
	}
	if !stageState.exists || stageState.hash != embedTxHash(body) || stageState.mode != 0o600 {
		return fmt.Errorf("embed refresh journal stage changed before promotion; preserving it")
	}
	journalState, err := tx.readState(embedTxJournalName)
	if err != nil {
		return err
	}
	if journalState.exists {
		current, err := tx.readJournalFile(embedTxJournalName)
		if err != nil {
			return err
		}
		if !embedTxSamePlan(current, journal) || (current.Phase == embedTxCommitted && journal.Phase != embedTxCommitted) {
			return fmt.Errorf("existing embed refresh journal has a different plan or would regress phase")
		}
	}
	if err := embedTransactionPoint("journal-before-promote:" + journal.Phase); err != nil {
		return err
	}
	if err := tx.revalidateState(embedTxStageName, stageState); err != nil {
		return fmt.Errorf("embed refresh journal stage changed before promotion; preserving it: %w", err)
	}
	if err := tx.revalidateState(embedTxJournalName, journalState); err != nil {
		return fmt.Errorf("embed refresh journal changed before promotion; preserving stage: %w", err)
	}
	if err := tx.root.Rename(embedTxStageName, embedTxJournalName); err != nil {
		return fmt.Errorf("install embed refresh journal: %w", err)
	}
	if err := embedTransactionPoint("journal-renamed:" + journal.Phase); err != nil {
		return err
	}
	if err := tx.syncDir("."); err != nil {
		return err
	}
	return embedTransactionPoint("journal-dir-synced:" + journal.Phase)
}

func (tx *embedRootTransaction) writeTemp(rel string, body []byte, mode fs.FileMode) error {
	if err := tx.validateParent(rel); err != nil {
		return err
	}
	f, err := tx.root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create embed transaction temp %s: %w", rel, err)
	}
	writeErr := func() error {
		if _, err := f.Write(body); err != nil {
			return err
		}
		if err := f.Chmod(mode); err != nil {
			return err
		}
		return f.Sync()
	}()
	closeErr := f.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return fmt.Errorf("persist embed transaction temp %s: %w", rel, err)
	}
	return tx.syncDir(path.Dir(rel))
}

type embedFileState struct {
	exists bool
	hash   string
	mode   fs.FileMode
	info   os.FileInfo
	size   int64
	mtime  int64
	change string
}

func embedInfoChangeID(info os.FileInfo) string {
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
	ctime, nsec := value.FieldByName("Ctime"), value.FieldByName("Ctimensec")
	if ctime.IsValid() && nsec.IsValid() && ctime.CanInt() && nsec.CanInt() {
		return fmt.Sprintf("%d:%d", ctime.Int(), nsec.Int())
	}
	return ""
}

func sameEmbedFileState(expected, current embedFileState) bool {
	if expected.exists != current.exists {
		return false
	}
	if !expected.exists {
		return true
	}
	return expected.hash == current.hash && expected.mode == current.mode && expected.size == current.size && expected.mtime == current.mtime &&
		os.SameFile(expected.info, current.info) && (expected.change == "" || current.change == "" || expected.change == current.change)
}

func sameEmbedFileIdentity(a, b embedFileState) bool {
	return a.exists && b.exists && a.info != nil && b.info != nil && os.SameFile(a.info, b.info) &&
		a.hash == b.hash && a.mode == b.mode && a.size == b.size
}

func (tx *embedRootTransaction) revalidateState(rel string, expected embedFileState) error {
	current, err := tx.readState(rel)
	if err != nil {
		return err
	}
	if !sameEmbedFileState(expected, current) {
		return fmt.Errorf("embed transaction path %s changed content, identity, mode, or metadata", rel)
	}
	return nil
}

func (tx *embedRootTransaction) removeExact(rel string, expected embedFileState, point, label string) error {
	if point != "" {
		if err := embedTransactionPoint(point); err != nil {
			return err
		}
	}
	if err := tx.revalidateState(rel, expected); err != nil {
		return fmt.Errorf("refuse to remove changed %s: %w", label, err)
	}
	if err := tx.root.Remove(rel); err != nil {
		return fmt.Errorf("remove %s: %w", label, err)
	}
	return nil
}

func (tx *embedRootTransaction) renameExact(source, destination string, sourceState embedFileState, point, label string) error {
	if err := embedTransactionPoint(point); err != nil {
		return err
	}
	if err := tx.revalidateState(source, sourceState); err != nil {
		return fmt.Errorf("refuse to rename changed %s: %w", label, err)
	}
	if state, err := tx.readState(destination); err != nil {
		return err
	} else if state.exists {
		return fmt.Errorf("refuse to overwrite concurrent embed transaction path %s", destination)
	}
	if err := tx.root.Link(source, destination); err != nil {
		return fmt.Errorf("install no-clobber link for %s: %w", label, err)
	}
	if err := tx.syncDir(path.Dir(source)); err != nil {
		return err
	}
	// Creating a hard link legitimately changes ctime/link-count metadata on
	// the source inode. Refresh that expected metadata only after proving the
	// source name still resolves to the originally witnessed inode and bytes.
	linkedSource, err := tx.readState(source)
	if err != nil {
		return err
	}
	if !sameEmbedFileIdentity(sourceState, linkedSource) {
		return fmt.Errorf("source for %s changed while linking; preserving both names", label)
	}
	sourceState = linkedSource
	if err := embedTransactionPoint("rename-linked:" + source + "->" + destination); err != nil {
		return err
	}
	destinationState, err := tx.readState(destination)
	if err != nil {
		return err
	}
	if !sameEmbedFileIdentity(sourceState, destinationState) {
		return fmt.Errorf("linked destination for %s changed before source removal; preserving both names", label)
	}
	if err := tx.revalidateState(source, sourceState); err != nil {
		return fmt.Errorf("source for %s changed after linking; preserving both names: %w", label, err)
	}
	if err := tx.root.Remove(source); err != nil {
		return fmt.Errorf("remove linked source for %s: %w", label, err)
	}
	if err := tx.syncDir(path.Dir(source)); err != nil {
		return err
	}
	return nil
}

func (tx *embedRootTransaction) readState(rel string) (embedFileState, error) {
	info, err := tx.root.Lstat(rel)
	if errors.Is(err, fs.ErrNotExist) {
		return embedFileState{}, nil
	}
	if err != nil {
		return embedFileState{}, fmt.Errorf("inspect embed transaction path %s: %w", rel, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return embedFileState{}, fmt.Errorf("embed transaction path %s must be a regular file, not a symlink or special file", rel)
	}
	f, err := tx.root.Open(rel)
	if err != nil {
		return embedFileState{}, err
	}
	opened, statErr := f.Stat()
	if statErr != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		_ = f.Close()
		return embedFileState{}, fmt.Errorf("embed transaction path %s changed identity while opening", rel)
	}
	body, readErr := io.ReadAll(f)
	closeErr := f.Close()
	after, pathErr := tx.root.Lstat(rel)
	if err := errors.Join(readErr, closeErr, pathErr); err != nil {
		return embedFileState{}, err
	}
	if !os.SameFile(info, opened) || !os.SameFile(opened, after) || opened.Mode() != after.Mode() || opened.Size() != after.Size() || !opened.ModTime().Equal(after.ModTime()) || embedInfoChangeID(opened) != embedInfoChangeID(after) {
		return embedFileState{}, fmt.Errorf("embed transaction path %s changed while reading", rel)
	}
	return embedFileState{exists: true, hash: embedTxHash(body), mode: opened.Mode().Perm(), info: after, size: after.Size(), mtime: after.ModTime().UnixNano(), change: embedInfoChangeID(after)}, nil
}

func (tx *embedRootTransaction) validateParent(rel string) error {
	dir := path.Dir(rel)
	if dir == "." {
		return nil
	}
	current := ""
	for _, part := range strings.Split(dir, "/") {
		current = path.Join(current, part)
		info, err := tx.root.Lstat(current)
		if err != nil {
			return fmt.Errorf("inspect embed target parent %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("embed target parent %s must be a real directory", current)
		}
	}
	return nil
}

func (tx *embedRootTransaction) syncDir(rel string) error {
	if rel == "" {
		rel = "."
	}
	f, err := tx.root.Open(rel)
	if err != nil {
		return fmt.Errorf("open embed transaction directory %s for sync: %w", rel, err)
	}
	return errors.Join(syncEmbedDirectoryHandle(f), f.Close())
}

func (tx *embedRootTransaction) validatePhysicalInventory(journal embedTxJournal, allowStage bool) error {
	wanted := map[string]bool{embedTxJournalName: true}
	if allowStage {
		wanted[embedTxStageName] = true
	}
	for _, item := range journal.Items {
		wanted[item.Temp], wanted[item.Backup] = true, true
		if err := tx.validateParent(item.Path); err != nil {
			return err
		}
	}
	residue, err := tx.embedResiduePaths()
	if err != nil {
		return err
	}
	for _, rel := range residue {
		if !wanted[rel] {
			return fmt.Errorf("unreferenced embed transaction residue %s", rel)
		}
	}
	return nil
}

func (tx *embedRootTransaction) embedResiduePaths() ([]string, error) {
	var residue []string
	var walk func(string) error
	walk = func(dir string) error {
		f, err := tx.root.Open(dir)
		if err != nil {
			return err
		}
		entries, readErr := f.ReadDir(-1)
		closeErr := f.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			rel := entry.Name()
			if dir != "." {
				rel = path.Join(dir, rel)
			}
			info, err := tx.root.Lstat(rel)
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("embed transaction inventory entry %s is a symlink", rel)
			}
			base := entry.Name()
			reserved := base == embedTxJournalName || base == embedTxStageName || strings.HasPrefix(base, embedTxNewPrefix) || strings.HasPrefix(base, embedTxOldPrefix)
			if reserved {
				if !info.Mode().IsRegular() {
					return fmt.Errorf("embed transaction residue %s must be a regular file", rel)
				}
				residue = append(residue, rel)
				continue
			}
			if info.IsDir() {
				if entry.Name() == ".git" {
					continue
				}
				if err := walk(rel); err != nil {
					return err
				}
				continue
			}
		}
		return nil
	}
	if err := walk("."); err != nil {
		return nil, fmt.Errorf("inspect embed transaction residue: %w", err)
	}
	sort.Strings(residue)
	return residue, nil
}

func encodeEmbedTxJournal(journal embedTxJournal) ([]byte, error) {
	journal.Checksum = embedTxChecksum(journal)
	encoded, err := json.Marshal(journal)
	if err != nil {
		return nil, err
	}
	encoded = append(encoded, '\n')
	if len(encoded) == 0 || len(encoded) > embedTxMaxJournal {
		return nil, fmt.Errorf("journal size %d is outside 1..%d", len(encoded), embedTxMaxJournal)
	}
	return encoded, nil
}

func decodeEmbedTxJournal(body []byte) (embedTxJournal, error) {
	if len(body) == 0 || len(body) > embedTxMaxJournal {
		return embedTxJournal{}, fmt.Errorf("journal size %d is outside 1..%d", len(body), embedTxMaxJournal)
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	var journal embedTxJournal
	if err := dec.Decode(&journal); err != nil {
		return embedTxJournal{}, err
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return embedTxJournal{}, fmt.Errorf("journal contains multiple JSON values")
		}
		return embedTxJournal{}, err
	}
	canonical, err := encodeEmbedTxJournal(journal)
	if err != nil {
		return embedTxJournal{}, err
	}
	if !bytes.Equal(canonical, body) {
		return embedTxJournal{}, fmt.Errorf("journal is not in its unique canonical encoding")
	}
	if err := validateEmbedTxJournal(journal); err != nil {
		return embedTxJournal{}, err
	}
	return journal, nil
}

func validateEmbedTxJournal(journal embedTxJournal) error {
	if journal.Version != embedTxVersion || journal.Operation != "embed-refresh" || (journal.Phase != embedTxPrepared && journal.Phase != embedTxCommitted) || journal.Scope == "" {
		return fmt.Errorf("unsupported journal version, operation, phase, or empty scope")
	}
	if len(journal.Items) == 0 || len(journal.Items) > 100_000 {
		return fmt.Errorf("journal item count %d is outside 1..100000", len(journal.Items))
	}
	seen := map[string]string{}
	prior := ""
	for i, item := range journal.Items {
		if err := portablepath.ValidateRelative(item.Path); err != nil {
			return fmt.Errorf("journal item %d path: %w", i, err)
		}
		if i > 0 && item.Path <= prior {
			return fmt.Errorf("journal item paths are not strictly byte-sorted")
		}
		prior = item.Path
		for _, candidate := range []struct{ kind, rel string }{
			{kind: "target", rel: item.Path},
			{kind: "temp", rel: item.Temp},
			{kind: "backup", rel: item.Backup},
		} {
			kind, rel := candidate.kind, candidate.rel
			if err := portablepath.ValidateRelative(rel); err != nil {
				return fmt.Errorf("journal item %d %s: %w", i, kind, err)
			}
			fold := strings.ToLower(rel)
			if previous, ok := seen[fold]; ok {
				return fmt.Errorf("journal item %d %s %s aliases %s", i, kind, rel, previous)
			}
			seen[fold] = kind + " " + rel
		}
		dir := path.Dir(item.Path)
		if path.Dir(item.Temp) != dir || path.Dir(item.Backup) != dir || path.Base(item.Temp) != embedTxNewPrefix+fmt.Sprintf("%06d", i) || path.Base(item.Backup) != embedTxOldPrefix+fmt.Sprintf("%06d", i) {
			return fmt.Errorf("journal item %d temp/backup names are not canonically derived", i)
		}
		if item.OldMode == 0 || item.OldMode&^0o777 != 0 || (!item.Deletion && (item.NewMode == 0 || item.NewMode&^0o777 != 0)) || (item.Deletion && (item.NewMode != 0 || len(item.New) != 0)) {
			return fmt.Errorf("journal item %d has invalid modes or deletion payload", i)
		}
		if item.OldHash != embedTxHash(item.Old) || item.NewHash != embedTxHash(item.New) {
			return fmt.Errorf("journal item %d content hash mismatch", i)
		}
	}
	if journal.Checksum != embedTxChecksum(journal) {
		return fmt.Errorf("journal checksum mismatch")
	}
	return nil
}

func embedTxChecksum(journal embedTxJournal) string {
	payload := embedTxPayload{Version: journal.Version, Operation: journal.Operation, Phase: journal.Phase, Scope: journal.Scope, Items: journal.Items}
	body, _ := json.Marshal(payload)
	return embedTxHash(body)
}

func embedTxHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func sortedEmbedDirs(set map[string]bool) []string {
	dirs := make([]string, 0, len(set))
	for dir := range set {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs
}
