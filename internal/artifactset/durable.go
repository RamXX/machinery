package artifactset

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/filelock"
	"github.com/RamXX/machinery/internal/fsatomic"
	"github.com/RamXX/machinery/internal/portablepath"
)

const (
	txPrefix                              = ".machinery-artifact-"
	txNewPrefix                           = txPrefix + "new-"
	txOldPrefix                           = txPrefix + "old-"
	txJournalName                         = txPrefix + "set.journal"
	txRecoveryName                        = txPrefix + "set.recovery"
	txRetiredName                         = txPrefix + "set.retired"
	txStagePrefix                         = txPrefix + "journal-stage-"
	txJournalUpdateQuarantinePrefix       = txPrefix + "journal-update-quarantine-"
	txJournalQuarantinePrefix             = txPrefix + "journal-quarantine-"
	txObjectQuarantinePrefix              = txPrefix + "object-quarantine-"
	txVersion                             = 2
	txPrepared                            = "prepared"
	txCommitted                           = "committed"
	txMaxJournal                          = 16 << 20
	txMaxItemCount                        = 100_000
	txMaxFileBytes                  int64 = 64 << 20
	txMaxDirEntries                       = txMaxItemCount*4 + 16
	txDirPageSize                         = 256
)

type txOps struct {
	root          *os.Root
	syncFile      *os.File
	acquire       func(string) (txLocker, error)
	renameHook    func(string, string) error
	hashAfterOpen func(string) error
	syncObserve   func(string)
	faultAfter    func(string) error
}

type txLocker interface {
	Release() error
}

type txCrash struct {
	point string
	cause error
}

func (e *txCrash) Error() string {
	return fmt.Sprintf("simulated crash after %s: %v", e.point, e.cause)
}
func (e *txCrash) Unwrap() error { return e.cause }

type txJournal struct {
	Version  int      `json:"version"`
	Phase    string   `json:"phase"`
	Items    []txItem `json:"items"`
	Checksum string   `json:"checksum"`
}

type txItem struct {
	Target      string `json:"target"`
	Temp        string `json:"temp"`
	Backup      string `json:"backup"`
	HadOld      bool   `json:"had_old"`
	OldHash     string `json:"old_hash,omitempty"`
	OldIdentity string `json:"old_identity,omitempty"`
	OldChange   string `json:"old_change,omitempty"`
	NewHash     string `json:"new_hash"`
	NewIdentity string `json:"new_identity"`
	NewChange   string `json:"new_change,omitempty"`
	Delete      bool   `json:"delete,omitempty"`
}

type txPayload struct {
	Version int      `json:"version"`
	Phase   string   `json:"phase"`
	Items   []txItem `json:"items"`
}

type txSnapshot struct {
	exists   bool
	hash     string
	identity string
	change   string
	info     os.FileInfo
}

type txInventory struct {
	journal            bool
	recovery           bool
	retired            bool
	stages             []string
	temps              []string
	backups            []string
	journalUpdates     []string
	journalQuarantines []string
	objectQuarantines  []string
}

type txJournalAuthority struct {
	name     string
	journal  txJournal
	snapshot txSnapshot
}

func txDefaultOps(renameHook func(string, string) error) txOps {
	return txOps{
		acquire:    func(scope string) (txLocker, error) { return filelock.Acquire(scope) },
		renameHook: renameHook,
	}
}

func txCommit(dir string, files map[string][]byte, ops txOps) (returnErr error) {
	return txReconcile(dir, files, nil, ops)
}

func txReconcile(dir string, files map[string][]byte, remove []string, ops txOps) (returnErr error) {
	conditions := make([]RemovalPrecondition, len(remove))
	for i, name := range remove {
		conditions[i] = RemovalPrecondition{Name: name}
	}
	return txReconcileConditions(dir, files, conditions, false, ops)
}

func txReconcilePlanned(dir string, files map[string][]byte, remove []RemovalPrecondition, ops txOps) (returnErr error) {
	return txReconcileConditions(dir, files, remove, true, ops)
}

func txReconcileConditions(dir string, files map[string][]byte, remove []RemovalPrecondition, requireInspected bool, ops txOps) (returnErr error) {
	resolved, err := ensureRealDir(dir)
	if err != nil {
		return err
	}
	root, syncFile, err := txOpenRoot(resolved)
	if err != nil {
		return err
	}
	return txReconcilePlannedOpened(resolved, root, syncFile, true, files, remove, nil, requireInspected, ops)
}

func txReconcilePlannedRooted(scope string, root *os.Root, files map[string][]byte, remove []RemovalPrecondition, ops txOps) error {
	if root == nil {
		return fmt.Errorf("artifact output root is nil")
	}
	syncFile, err := txOpenSyncRoot(root)
	if err != nil {
		return err
	}
	return txReconcilePlannedOpened(scope, root, syncFile, false, files, remove, nil, true, ops)
}

func txReconcileGuardedRooted(scope string, root *os.Root, files map[string][]byte, remove, replace []RemovalPrecondition, ops txOps) error {
	syncFile, err := txOpenSyncRoot(root)
	if err != nil {
		return err
	}
	return txReconcilePlannedOpened(scope, root, syncFile, false, files, remove, replace, true, ops)
}

func txReconcilePlannedOpened(scope string, root *os.Root, syncFile *os.File, closeRoot bool, files map[string][]byte, remove, replace []RemovalPrecondition, requireInspected bool, ops txOps) (returnErr error) {
	lock, err := ops.acquire(scope)
	if err != nil {
		return fmt.Errorf("acquire artifact transaction lock: %w", err)
	}
	defer func() {
		if err := lock.Release(); err != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("release artifact transaction lock: %w", err))
		}
	}()
	ops.root, ops.syncFile = root, syncFile
	defer func() {
		if err := syncFile.Close(); err != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("close artifact directory sync handle: %w", err))
		}
		if closeRoot {
			if err := root.Close(); err != nil {
				returnErr = errors.Join(returnErr, fmt.Errorf("close artifact directory root: %w", err))
			}
		}
	}()
	if err := txPoint(ops, "root-open"); err != nil {
		return err
	}
	if err := txRecover(scope, ops); err != nil {
		return fmt.Errorf("recover artifact transaction: %w", err)
	}
	removeNames := make([]string, len(remove))
	preconditions := make(map[string]RemovalPrecondition, len(remove))
	for i, condition := range remove {
		removeNames[i] = condition.Name
		if condition.info != nil {
			preconditions[condition.Name] = condition
		} else if requireInspected {
			return fmt.Errorf("planned removal target %q lacks an inspected identity", condition.Name)
		}
	}
	if err := txValidateRemovalPreconditions(ops, preconditions); err != nil {
		return err
	}
	for _, condition := range replace {
		if condition.info == nil || condition.Name == "" {
			return fmt.Errorf("guarded replacement target %q lacks an inspected identity", condition.Name)
		}
		if _, ok := files[condition.Name]; !ok {
			return fmt.Errorf("guarded replacement target %q is not in the install set", condition.Name)
		}
		if _, duplicate := preconditions[condition.Name]; duplicate {
			return fmt.Errorf("artifact %q cannot be both a guarded replacement and removal", condition.Name)
		}
		preconditions[condition.Name] = condition
	}
	if len(replace) > 0 {
		if err := txValidateRemovalPreconditions(ops, preconditions); err != nil {
			return fmt.Errorf("validate guarded replacement ownership: %w", err)
		}
	}
	names, deletes, err := txValidateReconcileTargets(ops, files, removeNames)
	if err != nil || len(names) == 0 {
		return err
	}
	journal, err := txStage(names, files, deletes, ops)
	if err != nil {
		return err
	}
	journalInstalled, err := txPersistJournal(scope, journal, ops)
	if err != nil {
		if journalInstalled {
			return fmt.Errorf("persist prepared journal: %w", err)
		}
		cleanupErr := txRemoveUnjournaled(journal, ops)
		return errors.Join(fmt.Errorf("write prepared journal: %w", err), cleanupErr)
	}
	if err := txPoint(ops, "prepared-journal"); err != nil {
		return err
	}
	failPrepared := func(context string, cause error) error {
		authority, authorityErr := txBeginJournalAuthority(scope, ops)
		if authorityErr != nil {
			return fmt.Errorf("%s: %w; retain prepared journal authority: %w", context, cause, authorityErr)
		}
		rollbackErr := txRollback(scope, authority, nil, ops)
		if rollbackErr != nil {
			return fmt.Errorf("%s: %w; durable rollback also failed: %w", context, cause, rollbackErr)
		}
		return fmt.Errorf("%s: %w", context, cause)
	}
	for _, item := range journal.Items {
		if !item.HadOld {
			continue
		}
		if err := txRename(scope, item.Target, item.Backup, ops); err != nil {
			return failPrepared("park old "+item.Target, err)
		}
		if condition, ok := preconditions[item.Target]; ok {
			if err := txValidateRemovalPath(ops, item.Backup, condition); err != nil {
				return failPrepared("verify parked stale artifact "+item.Target, err)
			}
		}
		if err := txPoint(ops, "park:"+item.Target); err != nil {
			return err
		}
	}
	for _, item := range journal.Items {
		if item.Delete {
			temp, err := txSnapshotPath(item.Temp, "staged deletion "+item.Temp, ops)
			if err != nil {
				return failPrepared("inspect deletion "+item.Target, err)
			}
			if err := txValidateDurableSnapshot(temp, item.NewIdentity, item.NewChange, true); err != nil {
				return failPrepared("verify deletion "+item.Target, err)
			}
			if err := txRemoveSnapshot(scope, item.Temp, "staged deletion "+item.Temp, temp, ops); err != nil {
				return failPrepared("install deletion "+item.Target, err)
			}
		} else if err := txRename(scope, item.Temp, item.Target, ops); err != nil {
			return failPrepared("install "+item.Target, err)
		}
		if err := txPoint(ops, "install:"+item.Target); err != nil {
			return err
		}
	}
	journal.Phase = txCommitted
	journalInstalled, err = txPersistJournal(scope, journal, ops)
	if err != nil {
		if journalInstalled {
			return fmt.Errorf("persist committed journal: %w", err)
		}
		return failPrepared("write committed journal", err)
	}
	if err := txPoint(ops, "committed-journal"); err != nil {
		return err
	}
	authority, err := txBeginJournalAuthority(scope, ops)
	if err != nil {
		return fmt.Errorf("retain committed journal authority: %w", err)
	}
	if err := txFinalize(scope, authority, nil, ops); err != nil {
		return fmt.Errorf("finalize committed transaction: %w", err)
	}
	return nil
}

func txValidateRemovalPreconditions(ops txOps, conditions map[string]RemovalPrecondition) error {
	names := make([]string, 0, len(conditions))
	for name := range conditions {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := txValidateRemovalPath(ops, name, conditions[name]); err != nil {
			return fmt.Errorf("stale artifact %s no longer matches its ownership plan: %w", name, err)
		}
	}
	return nil
}

func txValidateRemovalPath(ops txOps, name string, condition RemovalPrecondition) error {
	if condition.info == nil || condition.digest == "" {
		return fmt.Errorf("missing inspected file identity")
	}
	info, err := ops.root.Lstat(name)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode() != condition.mode || !os.SameFile(info, condition.info) {
		return fmt.Errorf("changed identity or mode")
	}
	hash, exists, err := txHashPath(name, "planned stale artifact "+name, ops)
	if err != nil {
		return err
	}
	if !exists || hash != condition.digest {
		return fmt.Errorf("changed content")
	}
	after, err := ops.root.Lstat(name)
	if err != nil || !os.SameFile(info, after) || after.Mode() != info.Mode() {
		return fmt.Errorf("changed identity while hashing")
	}
	return nil
}

func txValidateTargets(ops txOps, files map[string][]byte) ([]string, error) {
	names, _, err := txValidateReconcileTargets(ops, files, nil)
	return names, err
}

func txValidateReconcileTargets(ops txOps, files map[string][]byte, remove []string) ([]string, map[string]bool, error) {
	if len(files) > txMaxItemCount || len(remove) > txMaxItemCount-len(files) {
		return nil, nil, fmt.Errorf("artifact transaction exceeds %d-item limit", txMaxItemCount)
	}
	names := make([]string, 0, len(files)+len(remove))
	deletes := make(map[string]bool, len(remove))
	for name := range files {
		names = append(names, name)
	}
	for _, name := range remove {
		if _, exists := files[name]; exists {
			return nil, nil, fmt.Errorf("artifact %q cannot be installed and removed in the same transaction", name)
		}
		if deletes[name] {
			return nil, nil, fmt.Errorf("artifact removal target %q appears more than once", name)
		}
		deletes[name] = true
		names = append(names, name)
	}
	sort.Strings(names)
	folded := make(map[string]string, len(files))
	for _, name := range names {
		if err := txValidateTarget(name); err != nil {
			return nil, nil, err
		}
		fold := strings.ToLower(name)
		if prior, ok := folded[fold]; ok {
			return nil, nil, fmt.Errorf("portable artifact-name collision: %q and %q alias on case-insensitive filesystems", prior, name)
		}
		folded[fold] = name
	}
	entries, err := txReadDir(ops)
	if err != nil {
		return nil, nil, fmt.Errorf("read artifact directory: %w", err)
	}
	for _, entry := range entries {
		if want, ok := folded[strings.ToLower(entry.Name())]; ok && entry.Name() != want {
			return nil, nil, fmt.Errorf("portable artifact-name collision: existing %q aliases generated %q", entry.Name(), want)
		}
	}
	kept := names[:0]
	for _, name := range names {
		if !deletes[name] {
			kept = append(kept, name)
			continue
		}
		_, exists, err := txHashPath(name, "artifact removal target "+name, ops)
		if err != nil {
			return nil, nil, err
		}
		if exists {
			kept = append(kept, name)
		} else {
			delete(deletes, name)
		}
	}
	return kept, deletes, nil
}

func txValidateTarget(name string) error {
	if name == "" || name == "." || filepath.Base(name) != name {
		return fmt.Errorf("unsafe artifact name %q", name)
	}
	if strings.HasPrefix(strings.ToLower(name), txPrefix) {
		return fmt.Errorf("artifact name %q uses reserved transaction namespace %q", name, txPrefix)
	}
	if err := portablepath.ValidateBase(name); err != nil {
		return fmt.Errorf("validate portable artifact name %q: %w", name, err)
	}
	return nil
}

func txStage(names []string, files map[string][]byte, deletes map[string]bool, ops txOps) (txJournal, error) {
	journal := txJournal{Version: txVersion, Phase: txPrepared}
	for _, name := range names {
		item, err := txStageItem(name, files[name], deletes[name], ops)
		if err != nil {
			cleanupErr := txRemoveUnjournaled(journal, ops)
			return txJournal{}, errors.Join(fmt.Errorf("stage artifact %s: %w", name, err), cleanupErr)
		}
		journal.Items = append(journal.Items, item)
	}
	return journal, nil
}

func txStageItem(name string, body []byte, deleteTarget bool, ops txOps) (_ txItem, returnErr error) {
	item := txItem{Target: name, NewHash: txHash(body), Delete: deleteTarget}
	old, err := txSnapshotPath(name, "artifact target "+name, ops)
	if err != nil {
		return txItem{}, err
	}
	item.HadOld, item.OldHash = old.exists, old.hash
	item.OldIdentity, item.OldChange = old.identity, old.change
	if deleteTarget && !old.exists {
		return txItem{}, fmt.Errorf("artifact removal target %s disappeared before staging", name)
	}
	tmp, tempName, err := txCreateTemp(ops, txNewPrefix)
	if err != nil {
		return txItem{}, fmt.Errorf("create staged artifact: %w", err)
	}
	item.Temp = tempName
	item.Backup = txOldPrefix + strings.TrimPrefix(item.Temp, txNewPrefix)
	created, err := tmp.Stat()
	if err != nil {
		return txItem{}, fmt.Errorf("inspect created staged artifact: %w", err)
	}
	ok := false
	closed := false
	defer func() {
		if !ok {
			var closeErr error
			if !closed {
				closeErr = tmp.Close()
			}
			cleanupErr := txCleanupCreated("", item.Temp, "failed staged artifact "+item.Temp, created, ops)
			returnErr = errors.Join(returnErr, closeErr, cleanupErr)
		}
	}()
	if _, err := ops.root.Lstat(item.Backup); err == nil {
		return txItem{}, fmt.Errorf("derived backup %q already exists", item.Backup)
	} else if !os.IsNotExist(err) {
		return txItem{}, fmt.Errorf("inspect derived backup: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		return txItem{}, fmt.Errorf("set staged artifact mode: %w", err)
	}
	if _, err := tmp.Write(body); err != nil {
		return txItem{}, fmt.Errorf("write staged artifact: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return txItem{}, fmt.Errorf("sync staged artifact: %w", err)
	}
	err = tmp.Close()
	closed = true
	if err != nil {
		return txItem{}, fmt.Errorf("close staged artifact: %w", err)
	}
	staged, err := txSnapshotPath(item.Temp, "staged artifact "+item.Temp, ops)
	if err != nil {
		return txItem{}, err
	}
	if !staged.exists || staged.hash != item.NewHash {
		return txItem{}, fmt.Errorf("staged artifact %s changed before journaling", item.Temp)
	}
	item.NewIdentity, item.NewChange = staged.identity, staged.change
	ok = true
	return item, nil
}

func txRemoveUnjournaled(journal txJournal, ops txOps) error {
	var errs []error
	for _, item := range journal.Items {
		snapshot, err := txSnapshotPath(item.Temp, "unjournaled temp "+item.Temp, ops)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !snapshot.exists {
			continue
		}
		if err := txRemoveSnapshot("", item.Temp, "unjournaled temp "+item.Temp, snapshot, ops); err != nil {
			errs = append(errs, fmt.Errorf("remove unjournaled temp %s: %w", item.Temp, err))
		}
	}
	return errors.Join(errs...)
}

func txPersistJournal(dir string, journal txJournal, ops txOps) (installed bool, returnErr error) {
	body, err := txEncode(journal)
	if err != nil {
		return false, err
	}
	stage, stageName, err := txCreateTemp(ops, txStagePrefix)
	if err != nil {
		return false, fmt.Errorf("create journal stage: %w", err)
	}
	created, err := stage.Stat()
	if err != nil {
		return false, fmt.Errorf("inspect created journal stage: %w", err)
	}
	ok := false
	closed := false
	defer func() {
		if !ok {
			var closeErr error
			if !closed {
				closeErr = stage.Close()
			}
			cleanupErr := txCleanupCreated(dir, stageName, "failed journal stage "+stageName, created, ops)
			returnErr = errors.Join(returnErr, closeErr, cleanupErr)
		}
	}()
	if err := stage.Chmod(0o600); err != nil {
		return false, fmt.Errorf("set journal mode: %w", err)
	}
	if _, err := stage.Write(body); err != nil {
		return false, fmt.Errorf("write journal stage: %w", err)
	}
	if err := stage.Sync(); err != nil {
		return false, fmt.Errorf("sync journal stage: %w", err)
	}
	err = stage.Close()
	closed = true
	if err != nil {
		return false, fmt.Errorf("close journal stage: %w", err)
	}
	staged, err := txSnapshotPath(stageName, "completed journal stage "+stageName, ops)
	if err != nil {
		return false, err
	}
	if !staged.exists || staged.hash != txHash(body) || staged.identity != txStableFileID(created) {
		return false, fmt.Errorf("completed journal stage changed before installation")
	}
	var previous *fsatomic.Quarantined
	var previousSnapshot txSnapshot
	if journal.Phase == txCommitted {
		current, readErr := txReadJournal(txJournalName, ops)
		if readErr != nil {
			return false, fmt.Errorf("read prepared journal before committed update: %w", readErr)
		}
		if current.journal.Phase != txPrepared || !txSameJournalTransaction(current.journal, journal) {
			return false, fmt.Errorf("prepared journal does not authorize committed update")
		}
		previousSnapshot = current.snapshot
		previous, err = fsatomic.Quarantine(ops.root, txJournalName, txJournalUpdateQuarantinePrefix)
		if err != nil {
			return false, fmt.Errorf("isolate prepared journal for committed update: %w", err)
		}
		if err := txSyncHeld(ops.syncFile); err != nil {
			return false, errors.Join(err, previous.Close())
		}
		qops := ops
		qops.root = previous.Root()
		isolated, readErr := txReadJournal(previous.Name(), qops)
		if readErr != nil || !txSameMovedSnapshot(previousSnapshot, isolated.snapshot) {
			restoreErr := previous.Restore()
			return false, errors.Join(fmt.Errorf("prepared journal changed at update isolation boundary"), readErr, restoreErr, previous.Close())
		}
		previousSnapshot = isolated.snapshot
		if err := txPoint(ops, "journal-update-isolated"); err != nil {
			return false, errors.Join(err, previous.Close())
		}
	}
	if ops.renameHook != nil {
		if err := ops.renameHook(stageName, txJournalName); err != nil {
			var restoreErr error
			if previous != nil {
				restoreErr = previous.Restore()
			}
			return false, errors.Join(fmt.Errorf("journal install hook for %s to %s: %w", stageName, txJournalName, err), restoreErr, closeQuarantine(previous))
		}
	}
	if err := txValidateSnapshot(stageName, "journal stage at installation boundary", staged, ops); err != nil {
		var restoreErr error
		if previous != nil {
			restoreErr = previous.Restore()
		}
		return false, errors.Join(err, restoreErr, closeQuarantine(previous))
	}
	if err := fsatomic.RenameNoReplace(ops.root, stageName, txJournalName); err != nil {
		var restoreErr error
		if previous != nil {
			restoreErr = previous.Restore()
		}
		return false, errors.Join(fmt.Errorf("install transaction journal: %w", err), restoreErr, closeQuarantine(previous))
	}
	ok = true
	if err := txSyncHeld(ops.syncFile); err != nil {
		return true, fmt.Errorf("sync transaction journal directory: %w", err)
	}
	if ops.syncObserve != nil {
		ops.syncObserve(dir)
	}
	installedJournal, err := txReadJournal(txJournalName, ops)
	if err != nil || installedJournal.snapshot.hash != staged.hash || installedJournal.snapshot.identity != staged.identity {
		return true, errors.Join(fmt.Errorf("installed transaction journal changed at publication boundary"), err, closeQuarantine(previous))
	}
	if previous != nil {
		if err := txPoint(ops, "journal-update-installed"); err != nil {
			return true, errors.Join(err, previous.Close())
		}
		qops := ops
		qops.root = previous.Root()
		if err := txValidateSnapshot(previous.Name(), "isolated prepared journal before retirement", previousSnapshot, qops); err != nil {
			return true, errors.Join(err, previous.Close())
		}
		if err := previous.Remove(); err != nil {
			return true, errors.Join(err, previous.Close())
		}
		if err := txSyncHeld(ops.syncFile); err != nil {
			return true, err
		}
	}
	return true, nil
}

func closeQuarantine(quarantine *fsatomic.Quarantined) error {
	if quarantine == nil {
		return nil
	}
	return quarantine.Close()
}

func txSameJournalTransaction(left, right txJournal) bool {
	return left.Version == right.Version && reflect.DeepEqual(left.Items, right.Items)
}

func txSameMovedSnapshot(before, after txSnapshot) bool {
	return before.exists && after.exists && before.hash == after.hash && before.identity != "" && before.identity == after.identity &&
		before.info != nil && after.info != nil && before.info.Mode() == after.info.Mode() && before.info.Size() == after.info.Size() && before.info.ModTime().Equal(after.info.ModTime())
}

func txEncode(journal txJournal) ([]byte, error) {
	payload := txPayload{Version: journal.Version, Phase: journal.Phase, Items: journal.Items}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal journal payload: %w", err)
	}
	journal.Checksum = txHash(payloadBytes)
	body, err := json.Marshal(journal)
	if err != nil {
		return nil, fmt.Errorf("marshal journal: %w", err)
	}
	return append(body, '\n'), nil
}

func txDecode(body []byte) (txJournal, error) {
	if len(body) == 0 || len(body) > txMaxJournal {
		return txJournal{}, fmt.Errorf("journal size %d is outside 1..%d", len(body), txMaxJournal)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var journal txJournal
	if err := decoder.Decode(&journal); err != nil {
		return txJournal{}, fmt.Errorf("decode journal: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return txJournal{}, fmt.Errorf("journal contains multiple JSON values")
		}
		return txJournal{}, fmt.Errorf("decode trailing journal content: %w", err)
	}
	canonical, err := txEncode(journal)
	if err != nil {
		return txJournal{}, err
	}
	if !bytes.Equal(body, canonical) {
		return txJournal{}, fmt.Errorf("journal is not in its unique canonical encoding")
	}
	if err := txValidateJournal(journal); err != nil {
		return txJournal{}, err
	}
	return journal, nil
}

func txValidateJournal(journal txJournal) error {
	if journal.Version != txVersion || (journal.Phase != txPrepared && journal.Phase != txCommitted) {
		return fmt.Errorf("unsupported journal version or phase")
	}
	if len(journal.Items) == 0 || len(journal.Items) > txMaxItemCount {
		return fmt.Errorf("journal item count %d is outside 1..%d", len(journal.Items), txMaxItemCount)
	}
	seen := make(map[string]string, len(journal.Items)*3)
	prior := ""
	for i, item := range journal.Items {
		if err := txValidateTarget(item.Target); err != nil {
			return fmt.Errorf("journal item %d target: %w", i, err)
		}
		if i > 0 && item.Target <= prior {
			return fmt.Errorf("journal targets are not strictly sorted")
		}
		prior = item.Target
		if err := txValidateStage(item.Temp, txNewPrefix); err != nil {
			return fmt.Errorf("journal item %d temp: %w", i, err)
		}
		if err := txValidateStage(item.Backup, txOldPrefix); err != nil {
			return fmt.Errorf("journal item %d backup: %w", i, err)
		}
		if item.Backup != txOldPrefix+strings.TrimPrefix(item.Temp, txNewPrefix) {
			return fmt.Errorf("journal item %d backup is not derived from temp", i)
		}
		if !txValidHash(item.NewHash) || !txValidNativeWitness(item.NewIdentity) || !txValidOptionalNativeWitness(item.NewChange) ||
			item.HadOld != (item.OldHash != "") || item.HadOld != (item.OldIdentity != "") ||
			(item.OldHash != "" && !txValidHash(item.OldHash)) || !txValidOptionalNativeWitness(item.OldChange) ||
			item.Delete && (!item.HadOld || item.NewHash != txHash(nil)) {
			return fmt.Errorf("journal item %d has invalid content hashes", i)
		}
		orderedNames := []struct {
			kind string
			name string
		}{
			{kind: "target", name: item.Target},
			{kind: "temp", name: item.Temp},
			{kind: "backup", name: item.Backup},
		}
		for _, candidate := range orderedNames {
			kind, name := candidate.kind, candidate.name
			fold := strings.ToLower(name)
			if previous, ok := seen[fold]; ok {
				return fmt.Errorf("journal %s %q aliases %s", kind, name, previous)
			}
			seen[fold] = kind + " " + name
		}
	}
	payload := txPayload{Version: journal.Version, Phase: journal.Phase, Items: journal.Items}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal journal checksum payload: %w", err)
	}
	if journal.Checksum != txHash(body) {
		return fmt.Errorf("journal checksum mismatch")
	}
	return nil
}

func txValidateStage(name, prefix string) error {
	if filepath.Base(name) != name || !strings.HasPrefix(name, prefix) || len(name) == len(prefix) {
		return fmt.Errorf("%q is not a confined %q basename", name, prefix)
	}
	if err := portablepath.ValidateBase(name); err != nil {
		return fmt.Errorf("validate internal basename %q: %w", name, err)
	}
	return nil
}

func txRecover(dir string, ops txOps) error {
	inv, err := txInspect(ops)
	if err != nil {
		return err
	}
	if len(inv.journalUpdates) != 0 {
		if len(inv.journalUpdates) != 1 {
			return fmt.Errorf("transaction has ambiguous journal-update quarantine topology")
		}
		if err := txRecoverJournalUpdate(dir, inv.journalUpdates[0], inv, ops); err != nil {
			return fmt.Errorf("resume committed-journal update: %w", err)
		}
		inv, err = txInspect(ops)
		if err != nil {
			return err
		}
	}
	for _, name := range inv.objectQuarantines {
		if err := txRestoreObjectQuarantine(dir, name, ops); err != nil {
			return fmt.Errorf("resume artifact object quarantine: %w", err)
		}
	}
	if len(inv.objectQuarantines) != 0 {
		inv, err = txInspect(ops)
		if err != nil {
			return err
		}
	}
	if len(inv.journalQuarantines) != 0 {
		if len(inv.journalQuarantines) != 1 {
			return fmt.Errorf("transaction has ambiguous completed-journal quarantine topology")
		}
		quarantine, err := fsatomic.ResumeQuarantine(ops.root, inv.journalQuarantines[0], "")
		if err != nil {
			return err
		}
		if quarantine.Source() != txRetiredName {
			return errors.Join(fmt.Errorf("transaction quarantine records unexpected source %q", quarantine.Source()), quarantine.Close())
		}
		_, objectErr := quarantine.Root().Lstat(quarantine.Name())
		if os.IsNotExist(objectErr) {
			if err := quarantine.FinishEmpty(); err != nil {
				return errors.Join(err, quarantine.Close())
			}
			if err := txSyncHeld(ops.syncFile); err != nil {
				return err
			}
		} else if objectErr != nil {
			return errors.Join(objectErr, quarantine.Close())
		} else {
			if inv.journal || inv.recovery || inv.retired || len(inv.stages) != 0 || len(inv.temps) != 0 || len(inv.backups) != 0 {
				return errors.Join(fmt.Errorf("transaction has ambiguous completed-journal quarantine topology"), quarantine.Close())
			}
			if err := txRemoveOpenRecoveredJournalQuarantine(dir, quarantine, ops); err != nil {
				return fmt.Errorf("resume completed-journal quarantine cleanup: %w", err)
			}
		}
		inv, err = txInspect(ops)
		if err != nil {
			return err
		}
	}
	authorities := 0
	for _, exists := range []bool{inv.journal, inv.recovery, inv.retired} {
		if exists {
			authorities++
		}
	}
	if authorities > 1 {
		return fmt.Errorf("multiple transaction journal authorities exist")
	}
	if authorities == 0 {
		if len(inv.backups) != 0 {
			return fmt.Errorf("transaction backups exist without journal: %s", strings.Join(inv.backups, ", "))
		}
		orphans := append(append([]string{}, inv.temps...), inv.stages...)
		sort.Strings(orphans)
		for _, name := range orphans {
			snapshot, err := txSnapshotPath(name, "orphan transaction stage "+name, ops)
			if err != nil {
				return err
			}
			if err := txRemoveSnapshot(dir, name, "orphan transaction stage "+name, snapshot, ops); err != nil {
				return fmt.Errorf("remove orphan transaction stage %s: %w", name, err)
			}
			if err := txPoint(ops, "orphan-cleanup:"+name); err != nil {
				return err
			}
		}
		return nil
	}
	authority, err := txBeginJournalAuthority(dir, ops)
	if err != nil {
		return err
	}
	journal := authority.journal
	states, err := txPreflight(journal, ops)
	if err != nil {
		return err
	}
	if journal.Phase == txPrepared {
		return txRollback(dir, authority, states, ops)
	}
	return txFinalize(dir, authority, states, ops)
}

func txRecoverJournalUpdate(dir, name string, inv txInventory, ops txOps) error {
	quarantine, err := fsatomic.ResumeQuarantine(ops.root, name, txJournalName)
	if err != nil {
		return err
	}
	if _, err := quarantine.Root().Lstat(quarantine.Name()); os.IsNotExist(err) {
		if !inv.journal || inv.recovery || inv.retired {
			return errors.Join(fmt.Errorf("empty journal-update quarantine lacks one live committed journal"), quarantine.Close())
		}
		live, readErr := txReadJournal(txJournalName, ops)
		if readErr != nil || live.journal.Phase != txCommitted {
			return errors.Join(fmt.Errorf("empty journal-update quarantine live journal is not committed"), readErr, quarantine.Close())
		}
		if err := quarantine.FinishEmpty(); err != nil {
			return errors.Join(err, quarantine.Close())
		}
		return txSyncHeld(ops.syncFile)
	} else if err != nil {
		return errors.Join(err, quarantine.Close())
	}
	qops := ops
	qops.root = quarantine.Root()
	prepared, err := txReadJournal(quarantine.Name(), qops)
	if err != nil || prepared.journal.Phase != txPrepared {
		return errors.Join(fmt.Errorf("journal-update quarantine does not contain a prepared journal"), err, quarantine.Close())
	}
	if inv.recovery || inv.retired {
		return errors.Join(fmt.Errorf("journal-update quarantine overlaps another journal authority"), quarantine.Close())
	}
	if !inv.journal {
		if err := quarantine.Restore(); err != nil {
			return errors.Join(err, quarantine.Close())
		}
		if err := txSyncHeld(ops.syncFile); err != nil {
			return err
		}
		if ops.syncObserve != nil {
			ops.syncObserve(dir)
		}
		return nil
	}
	live, err := txReadJournal(txJournalName, ops)
	if err != nil || live.journal.Phase != txCommitted || !txSameJournalTransaction(prepared.journal, live.journal) {
		return errors.Join(fmt.Errorf("live journal does not complete the isolated prepared transaction"), err, quarantine.Close())
	}
	if err := txValidateSnapshot(quarantine.Name(), "isolated prepared journal before resumed retirement", prepared.snapshot, qops); err != nil {
		return errors.Join(err, quarantine.Close())
	}
	if err := quarantine.Remove(); err != nil {
		return errors.Join(err, quarantine.Close())
	}
	return txSyncHeld(ops.syncFile)
}

func txInspect(ops txOps) (txInventory, error) {
	entries, err := txReadDir(ops)
	if err != nil {
		return txInventory{}, fmt.Errorf("read transaction directory: %w", err)
	}
	var inv txInventory
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(strings.ToLower(name), txPrefix) {
			continue
		}
		if !strings.HasPrefix(name, txPrefix) {
			return txInventory{}, fmt.Errorf("reserved transaction path %q uses noncanonical case", name)
		}
		if err := portablepath.ValidateBase(name); err != nil {
			return txInventory{}, fmt.Errorf("reserved transaction path %q is not portable: %w", name, err)
		}
		info, err := ops.root.Lstat(name)
		if err != nil {
			return txInventory{}, fmt.Errorf("inspect reserved transaction path %q: %w", name, err)
		}
		if txValidQuarantineName(name, txJournalUpdateQuarantinePrefix) || txValidQuarantineName(name, txJournalQuarantinePrefix) || txValidQuarantineName(name, txObjectQuarantinePrefix) {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return txInventory{}, fmt.Errorf("reserved transaction quarantine %q must be a real directory", name)
			}
			if txValidQuarantineName(name, txJournalUpdateQuarantinePrefix) {
				inv.journalUpdates = append(inv.journalUpdates, name)
			} else if txValidQuarantineName(name, txJournalQuarantinePrefix) {
				inv.journalQuarantines = append(inv.journalQuarantines, name)
			} else {
				inv.objectQuarantines = append(inv.objectQuarantines, name)
			}
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return txInventory{}, fmt.Errorf("reserved transaction path %q must be a regular file, not a symlink or special file", name)
		}
		switch {
		case name == txJournalName:
			inv.journal = true
		case name == txRecoveryName:
			inv.recovery = true
		case name == txRetiredName:
			inv.retired = true
		case strings.HasPrefix(name, txStagePrefix) && len(name) > len(txStagePrefix):
			inv.stages = append(inv.stages, name)
		case strings.HasPrefix(name, txNewPrefix) && len(name) > len(txNewPrefix):
			inv.temps = append(inv.temps, name)
		case strings.HasPrefix(name, txOldPrefix) && len(name) > len(txOldPrefix):
			inv.backups = append(inv.backups, name)
		default:
			return txInventory{}, fmt.Errorf("unknown reserved transaction path %q", name)
		}
	}
	sort.Strings(inv.stages)
	sort.Strings(inv.temps)
	sort.Strings(inv.backups)
	sort.Strings(inv.journalUpdates)
	sort.Strings(inv.journalQuarantines)
	sort.Strings(inv.objectQuarantines)
	return inv, nil
}

func txValidQuarantineName(name, prefix string) bool {
	return strings.HasPrefix(name, prefix) && len(name) > len(prefix)
}

func txReadJournal(path string, ops txOps) (txJournalAuthority, error) {
	name := filepath.Base(path)
	before, err := ops.root.Lstat(name)
	if err != nil {
		return txJournalAuthority{}, fmt.Errorf("inspect journal: %w", err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > txMaxJournal {
		return txJournalAuthority{}, fmt.Errorf("transaction journal must be a bounded regular file, not a symlink or special file")
	}
	f, err := ops.root.Open(name)
	if err != nil {
		return txJournalAuthority{}, fmt.Errorf("open journal: %w", err)
	}
	opened, statErr := f.Stat()
	if statErr != nil {
		_ = f.Close()
		return txJournalAuthority{}, fmt.Errorf("inspect opened journal: %w", statErr)
	}
	if !os.SameFile(before, opened) || !opened.Mode().IsRegular() || before.Mode() != opened.Mode() || before.Size() != opened.Size() || !before.ModTime().Equal(opened.ModTime()) || txChangeID(before) != txChangeID(opened) {
		_ = f.Close()
		return txJournalAuthority{}, fmt.Errorf("transaction journal changed identity while opening")
	}
	body, readErr := io.ReadAll(io.LimitReader(f, txMaxJournal+1))
	openedAfter, retainedStatErr := f.Stat()
	closeErr := f.Close()
	if readErr != nil {
		return txJournalAuthority{}, fmt.Errorf("read journal: %w", readErr)
	}
	if retainedStatErr != nil {
		return txJournalAuthority{}, fmt.Errorf("reinspect opened journal: %w", retainedStatErr)
	}
	if closeErr != nil {
		return txJournalAuthority{}, fmt.Errorf("close journal: %w", closeErr)
	}
	if int64(len(body)) != before.Size() {
		return txJournalAuthority{}, fmt.Errorf("transaction journal changed size while reading")
	}
	pathAfter, pathErr := ops.root.Lstat(name)
	if pathErr != nil || !os.SameFile(before, openedAfter) || !os.SameFile(before, pathAfter) ||
		before.Mode() != openedAfter.Mode() || before.Mode() != pathAfter.Mode() || before.Size() != openedAfter.Size() || before.Size() != pathAfter.Size() ||
		!before.ModTime().Equal(openedAfter.ModTime()) || !before.ModTime().Equal(pathAfter.ModTime()) ||
		txChangeID(before) != txChangeID(openedAfter) || txChangeID(before) != txChangeID(pathAfter) {
		return txJournalAuthority{}, errors.Join(fmt.Errorf("transaction journal changed identity, content, or metadata while reading"), pathErr)
	}
	journal, err := txDecode(body)
	if err != nil {
		return txJournalAuthority{}, err
	}
	snapshot := txSnapshot{exists: true, hash: txHash(body), identity: txStableFileID(pathAfter), change: txChangeID(pathAfter), info: pathAfter}
	if snapshot.identity == "" {
		return txJournalAuthority{}, fmt.Errorf("transaction journal lacks a stable native file identity")
	}
	return txJournalAuthority{name: name, journal: journal, snapshot: snapshot}, nil
}

func txBeginJournalAuthority(dir string, ops txOps) (txJournalAuthority, error) {
	inv, err := txInspect(ops)
	if err != nil {
		return txJournalAuthority{}, err
	}
	names := make([]string, 0, 3)
	if inv.journal {
		names = append(names, txJournalName)
	}
	if inv.recovery {
		names = append(names, txRecoveryName)
	}
	if inv.retired {
		names = append(names, txRetiredName)
	}
	if len(names) != 1 {
		return txJournalAuthority{}, fmt.Errorf("expected exactly one transaction journal authority, found %d", len(names))
	}
	authority, err := txReadJournal(names[0], ops)
	if err != nil {
		return txJournalAuthority{}, err
	}
	if err := txReconcileInventory(authority.journal, inv); err != nil {
		return txJournalAuthority{}, err
	}
	if authority.name != txJournalName {
		return authority, nil
	}
	destination, err := txSnapshotPath(txRecoveryName, "recovery journal destination", ops)
	if err != nil {
		return txJournalAuthority{}, err
	}
	if destination.exists {
		return txJournalAuthority{}, fmt.Errorf("recovery journal destination already exists")
	}
	if err := txRenameNoReplaceSnapshots(dir, txJournalName, txRecoveryName, authority.snapshot, destination, ops); err != nil {
		return txJournalAuthority{}, fmt.Errorf("isolate parsed transaction journal: %w", err)
	}
	if err := txPoint(ops, "journal-isolated"); err != nil {
		return txJournalAuthority{}, err
	}
	isolated, err := txReadJournal(txRecoveryName, ops)
	if err != nil {
		return txJournalAuthority{}, fmt.Errorf("verify isolated transaction journal: %w", err)
	}
	if isolated.snapshot.hash != authority.snapshot.hash || isolated.snapshot.identity != authority.snapshot.identity ||
		isolated.snapshot.info == nil || authority.snapshot.info == nil ||
		isolated.snapshot.info.Mode() != authority.snapshot.info.Mode() || isolated.snapshot.info.Size() != authority.snapshot.info.Size() ||
		!isolated.snapshot.info.ModTime().Equal(authority.snapshot.info.ModTime()) {
		return txJournalAuthority{}, fmt.Errorf("isolated transaction journal differs from parsed authority")
	}
	return isolated, nil
}

func txReconcileInventory(journal txJournal, inv txInventory) error {
	wantTemps, wantBackups := map[string]bool{}, map[string]bool{}
	for _, item := range journal.Items {
		wantTemps[item.Temp], wantBackups[item.Backup] = true, true
	}
	for _, name := range inv.temps {
		if !wantTemps[name] {
			return fmt.Errorf("unreferenced transaction temp %q", name)
		}
	}
	for _, name := range inv.backups {
		if !wantBackups[name] {
			return fmt.Errorf("unreferenced transaction backup %q", name)
		}
	}
	return nil
}

func txPreflight(journal txJournal, ops txOps) ([]txSnapshot, error) {
	folded := map[string]string{}
	for _, item := range journal.Items {
		folded[strings.ToLower(item.Target)] = item.Target
	}
	entries, err := txReadDir(ops)
	if err != nil {
		return nil, fmt.Errorf("read transaction targets: %w", err)
	}
	for _, entry := range entries {
		if want, ok := folded[strings.ToLower(entry.Name())]; ok && entry.Name() != want {
			return nil, fmt.Errorf("transaction target %q has portable alias %q", want, entry.Name())
		}
	}
	states := make([]txSnapshot, 0, len(journal.Items)*3)
	for _, item := range journal.Items {
		target, err := txSnapshotPath(item.Target, "target "+item.Target, ops)
		if err != nil {
			return nil, err
		}
		temp, err := txSnapshotPath(item.Temp, "temp "+item.Temp, ops)
		if err != nil {
			return nil, err
		}
		backup, err := txSnapshotPath(item.Backup, "backup "+item.Backup, ops)
		if err != nil {
			return nil, err
		}
		if temp.exists && temp.hash != item.NewHash {
			return nil, fmt.Errorf("transaction temp %q hash mismatch", item.Temp)
		}
		if backup.exists && (!item.HadOld || backup.hash != item.OldHash) {
			return nil, fmt.Errorf("transaction backup %q hash mismatch", item.Backup)
		}
		if !txValidState(journal.Phase, item, target, temp, backup) {
			return nil, fmt.Errorf("transaction item %q has ambiguous physical state for phase %s", item.Target, journal.Phase)
		}
		if err := txValidateDurableItemWitness(journal.Phase, item, target, temp, backup); err != nil {
			return nil, err
		}
		states = append(states, target, temp, backup)
	}
	return states, nil
}

func txValidateDurableItemWitness(phase string, item txItem, target, temp, backup txSnapshot) error {
	if temp.exists {
		if err := txValidateDurableSnapshot(temp, item.NewIdentity, item.NewChange, true); err != nil {
			return fmt.Errorf("transaction temp %q %w", item.Temp, err)
		}
	}
	if backup.exists {
		if err := txValidateDurableSnapshot(backup, item.OldIdentity, item.OldChange, false); err != nil {
			return fmt.Errorf("transaction backup %q %w", item.Backup, err)
		}
	}
	if !target.exists {
		return nil
	}
	expectIdentity := item.NewIdentity
	if phase == txPrepared && (item.Delete || item.HadOld && !backup.exists) {
		expectIdentity = item.OldIdentity
	}
	// A target may already have crossed a rename boundary in either the forward
	// or rollback direction; native ctime behavior for rename is platform
	// specific. Stable file identity plus the journaled content digest binds
	// that moved object without rejecting a legitimate recovery.
	if err := txValidateDurableSnapshot(target, expectIdentity, "", false); err != nil {
		return fmt.Errorf("transaction target %q %w", item.Target, err)
	}
	return nil
}

func txValidateDurableSnapshot(snapshot txSnapshot, identity, change string, checkChange bool) error {
	if snapshot.identity == "" || identity == "" || snapshot.identity != identity {
		return fmt.Errorf("native identity mismatch")
	}
	if checkChange && change != "" && snapshot.change != change {
		return fmt.Errorf("native change identity mismatch")
	}
	return nil
}

func txValidState(phase string, item txItem, target, temp, backup txSnapshot) bool {
	if item.Delete {
		if phase == txCommitted {
			return !target.exists && !temp.exists && (!backup.exists || backup.hash == item.OldHash)
		}
		return target.exists && target.hash == item.OldHash && temp.exists && !backup.exists ||
			!target.exists && temp.exists && backup.exists && backup.hash == item.OldHash ||
			!target.exists && !temp.exists && backup.exists && backup.hash == item.OldHash ||
			target.exists && target.hash == item.OldHash && !temp.exists && !backup.exists
	}
	if phase == txCommitted {
		return target.exists && target.hash == item.NewHash && !temp.exists &&
			(!backup.exists || item.HadOld && backup.hash == item.OldHash)
	}
	if item.HadOld {
		return target.exists && target.hash == item.OldHash && temp.exists && !backup.exists ||
			!target.exists && temp.exists && backup.exists ||
			target.exists && target.hash == item.NewHash && !temp.exists && backup.exists ||
			!target.exists && !temp.exists && backup.exists ||
			target.exists && target.hash == item.OldHash && !temp.exists && !backup.exists
	}
	return !target.exists && temp.exists && !backup.exists ||
		target.exists && target.hash == item.NewHash && !temp.exists && !backup.exists ||
		!target.exists && !temp.exists && !backup.exists
}

func txValidateJournalAuthority(authority txJournalAuthority, ops txOps) error {
	if authority.name != txRecoveryName && authority.name != txRetiredName {
		return fmt.Errorf("transaction journal authority %q is not isolated", authority.name)
	}
	return txValidateSnapshot(authority.name, "transaction journal authority", authority.snapshot, ops)
}

func txRollback(dir string, authority txJournalAuthority, states []txSnapshot, ops txOps) error {
	journal := authority.journal
	if err := txValidateJournalAuthority(authority, ops); err != nil {
		return err
	}
	if states == nil {
		var err error
		states, err = txPreflight(journal, ops)
		if err != nil {
			return err
		}
	}
	for i := len(journal.Items) - 1; i >= 0; i-- {
		item := journal.Items[i]
		target, temp, backup := states[i*3], states[i*3+1], states[i*3+2]
		if item.HadOld && backup.exists {
			if target.exists {
				if err := txValidateJournalAuthority(authority, ops); err != nil {
					return err
				}
				if err := txRemoveSnapshot(dir, item.Target, "rollback target "+item.Target, target, ops); err != nil {
					return fmt.Errorf("remove uncommitted target %s: %w", item.Target, err)
				}
				if err := txPoint(ops, "rollback-remove-target:"+item.Target); err != nil {
					return err
				}
			}
			if err := txValidateJournalAuthority(authority, ops); err != nil {
				return err
			}
			if err := txRenameSnapshots(dir, item.Backup, item.Target, backup, txSnapshot{}, ops); err != nil {
				return fmt.Errorf("restore old target %s: %w", item.Target, err)
			}
			if err := txPoint(ops, "rollback-restore:"+item.Target); err != nil {
				return err
			}
		} else if !item.HadOld && target.exists {
			if err := txValidateJournalAuthority(authority, ops); err != nil {
				return err
			}
			if err := txRemoveSnapshot(dir, item.Target, "rollback target "+item.Target, target, ops); err != nil {
				return fmt.Errorf("remove new target %s: %w", item.Target, err)
			}
			if err := txPoint(ops, "rollback-remove-target:"+item.Target); err != nil {
				return err
			}
		}
		if temp.exists {
			if err := txValidateJournalAuthority(authority, ops); err != nil {
				return err
			}
			if err := txRemoveSnapshot(dir, item.Temp, "rollback temp "+item.Temp, temp, ops); err != nil {
				return fmt.Errorf("remove uncommitted temp %s: %w", item.Temp, err)
			}
			if err := txPoint(ops, "rollback-temp:"+item.Target); err != nil {
				return err
			}
		}
	}
	return txFinish(dir, authority, ops)
}

func txFinalize(dir string, authority txJournalAuthority, states []txSnapshot, ops txOps) error {
	journal := authority.journal
	if err := txValidateJournalAuthority(authority, ops); err != nil {
		return err
	}
	if states == nil {
		var err error
		states, err = txPreflight(journal, ops)
		if err != nil {
			return err
		}
	}
	for i, item := range journal.Items {
		backup := states[i*3+2]
		if !backup.exists {
			continue
		}
		if err := txValidateJournalAuthority(authority, ops); err != nil {
			return err
		}
		if err := txRemoveSnapshot(dir, item.Backup, "committed backup "+item.Backup, backup, ops); err != nil {
			return fmt.Errorf("remove committed backup for %s: %w", item.Target, err)
		}
		if err := txPoint(ops, "commit-cleanup:"+item.Target); err != nil {
			return err
		}
	}
	return txFinish(dir, authority, ops)
}

func txFinish(dir string, authority txJournalAuthority, ops txOps) error {
	if err := txValidateJournalAuthority(authority, ops); err != nil {
		return err
	}
	inv, err := txInspect(ops)
	if err != nil {
		return err
	}
	if inv.journal || inv.recovery != (authority.name == txRecoveryName) || inv.retired != (authority.name == txRetiredName) {
		return fmt.Errorf("transaction journal authority topology changed before cleanup")
	}
	for _, name := range inv.stages {
		if err := txValidateJournalAuthority(authority, ops); err != nil {
			return err
		}
		snapshot, err := txSnapshotPath(name, "journal stage "+name, ops)
		if err != nil {
			return err
		}
		if err := txRemoveSnapshot(dir, name, "journal stage "+name, snapshot, ops); err != nil {
			return fmt.Errorf("remove journal stage %s: %w", name, err)
		}
		if err := txPoint(ops, "journal-stage-cleanup:"+name); err != nil {
			return err
		}
	}
	if authority.name != txRetiredName {
		retired, err := txSnapshotPath(txRetiredName, "retired journal destination", ops)
		if err != nil {
			return err
		}
		if retired.exists {
			return fmt.Errorf("retired journal destination already exists")
		}
		if err := txRenameNoReplaceSnapshots(dir, authority.name, txRetiredName, authority.snapshot, retired, ops); err != nil {
			return fmt.Errorf("retire completed journal authority: %w", err)
		}
		if err := txPoint(ops, "journal-retired"); err != nil {
			return err
		}
		retiredAuthority, err := txReadJournal(txRetiredName, ops)
		if err != nil {
			return fmt.Errorf("verify retired journal authority: %w", err)
		}
		if retiredAuthority.snapshot.hash != authority.snapshot.hash || retiredAuthority.snapshot.identity != authority.snapshot.identity ||
			retiredAuthority.snapshot.info == nil || authority.snapshot.info == nil ||
			retiredAuthority.snapshot.info.Mode() != authority.snapshot.info.Mode() || retiredAuthority.snapshot.info.Size() != authority.snapshot.info.Size() ||
			!retiredAuthority.snapshot.info.ModTime().Equal(authority.snapshot.info.ModTime()) {
			return fmt.Errorf("retired journal differs from recovery authority")
		}
		authority = retiredAuthority
	}
	if err := txQuarantineAndRemoveJournal(dir, authority, ops); err != nil {
		return fmt.Errorf("remove completed journal authority: %w", err)
	}
	if err := txPoint(ops, "journal-cleanup"); err != nil {
		return err
	}
	return nil
}

func txQuarantineAndRemoveJournal(dir string, authority txJournalAuthority, ops txOps) error {
	if err := txValidateJournalAuthority(authority, ops); err != nil {
		return err
	}
	quarantine, err := fsatomic.Quarantine(ops.root, authority.name, txJournalQuarantinePrefix)
	if err != nil {
		return err
	}
	if err := txSyncHeld(ops.syncFile); err != nil {
		return errors.Join(err, quarantine.Close())
	}
	if ops.syncObserve != nil {
		ops.syncObserve(dir)
	}
	if err := txPoint(ops, "journal-quarantined"); err != nil {
		return errors.Join(err, quarantine.Close())
	}
	return txRemoveOpenJournalQuarantine(dir, quarantine, authority.snapshot, ops)
}

func txRemoveOpenRecoveredJournalQuarantine(dir string, quarantine *fsatomic.Quarantined, ops txOps) error {
	qops := ops
	qops.root = quarantine.Root()
	authority, err := txReadJournal(quarantine.Name(), qops)
	if err != nil {
		return errors.Join(err, quarantine.Close())
	}
	return txRemoveOpenJournalQuarantine(dir, quarantine, authority.snapshot, ops)
}

func txRestoreObjectQuarantine(dir, name string, ops txOps) error {
	quarantine, err := fsatomic.ResumeQuarantine(ops.root, name, "")
	if err != nil {
		return err
	}
	source := quarantine.Source()
	if filepath.Base(source) != source || source == "" || source == "." {
		return errors.Join(fmt.Errorf("artifact object quarantine records unsafe source %q", source), quarantine.Close())
	}
	_, objectErr := quarantine.Root().Lstat(quarantine.Name())
	if os.IsNotExist(objectErr) {
		if err := quarantine.FinishEmpty(); err != nil {
			return errors.Join(err, quarantine.Close())
		}
		return txSyncHeld(ops.syncFile)
	}
	if objectErr != nil {
		return errors.Join(objectErr, quarantine.Close())
	}
	qops := ops
	qops.root = quarantine.Root()
	if _, err := txSnapshotPath(quarantine.Name(), "quarantined artifact object", qops); err != nil {
		return errors.Join(err, quarantine.Close())
	}
	if _, err := ops.root.Lstat(source); err == nil {
		return errors.Join(fmt.Errorf("artifact object quarantine source %s was repopulated; preserving both", source), quarantine.Close())
	} else if !os.IsNotExist(err) {
		return errors.Join(err, quarantine.Close())
	}
	if err := quarantine.Restore(); err != nil {
		return errors.Join(err, quarantine.Close())
	}
	if err := txSyncHeld(ops.syncFile); err != nil {
		return err
	}
	if ops.syncObserve != nil {
		ops.syncObserve(dir)
	}
	return nil
}

func txRemoveOpenJournalQuarantine(dir string, quarantine *fsatomic.Quarantined, expected txSnapshot, ops txOps) error {
	qops := ops
	qops.root = quarantine.Root()
	qsync, err := txOpenSyncRoot(quarantine.Root())
	if err != nil {
		return errors.Join(err, quarantine.Close())
	}
	qops.syncFile = qsync
	current, err := txSnapshotPath(quarantine.Name(), "quarantined completed journal authority", qops)
	if err != nil || current.hash != expected.hash || current.identity != expected.identity || current.info == nil || expected.info == nil ||
		current.info.Mode() != expected.info.Mode() || current.info.Size() != expected.info.Size() || !current.info.ModTime().Equal(expected.info.ModTime()) {
		return errors.Join(fmt.Errorf("quarantined completed journal differs from its bound authority"), err, qsync.Close(), quarantine.Close())
	}
	if err := txPoint(ops, "journal-quarantine-delete"); err != nil {
		return errors.Join(err, qsync.Close(), quarantine.Close())
	}
	if err := txValidateSnapshot(quarantine.Name(), "quarantined completed journal authority", current, qops); err != nil {
		return errors.Join(fmt.Errorf("quarantined completed journal changed at deletion boundary; preserving it: %w", err), qsync.Close(), quarantine.Close())
	}
	removeErr := quarantine.Remove()
	privateSyncErr := txSyncHeld(qsync)
	privateCloseErr := qsync.Close()
	parentSyncErr := txSyncHeld(ops.syncFile)
	if ops.syncObserve != nil {
		ops.syncObserve(dir)
	}
	return errors.Join(removeErr, privateSyncErr, privateCloseErr, parentSyncErr)
}

func txSnapshotPath(path, label string, ops txOps) (txSnapshot, error) {
	name := filepath.Base(path)
	before, err := ops.root.Lstat(name)
	if os.IsNotExist(err) {
		return txSnapshot{}, nil
	}
	if err != nil {
		return txSnapshot{}, fmt.Errorf("inspect %s: %w", label, err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return txSnapshot{}, fmt.Errorf("%s must be a regular file, not a symlink or special file", label)
	}
	hash, exists, err := txHashPath(path, label, ops)
	if err != nil {
		return txSnapshot{}, err
	}
	if !exists {
		return txSnapshot{}, fmt.Errorf("%s disappeared while snapshotting", label)
	}
	after, err := ops.root.Lstat(name)
	if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || txChangeID(before) != txChangeID(after) {
		return txSnapshot{}, fmt.Errorf("%s changed identity or mode while snapshotting", label)
	}
	identity := txStableFileID(after)
	if identity == "" {
		return txSnapshot{}, fmt.Errorf("%s lacks a stable native file identity", label)
	}
	return txSnapshot{exists: true, hash: hash, identity: identity, change: txChangeID(after), info: after}, nil
}

func txValidateSnapshot(path, label string, expected txSnapshot, ops txOps) error {
	current, err := txSnapshotPath(path, label, ops)
	if err != nil {
		return err
	}
	if current.exists != expected.exists {
		return fmt.Errorf("%s changed existence since preflight", label)
	}
	if !expected.exists {
		return nil
	}
	if expected.info == nil || current.info == nil || !os.SameFile(expected.info, current.info) || expected.info.Mode() != current.info.Mode() || expected.identity == "" || current.identity != expected.identity {
		return fmt.Errorf("%s changed identity or mode since preflight", label)
	}
	if current.hash != expected.hash {
		return fmt.Errorf("%s changed content since preflight", label)
	}
	if expected.change != "" && current.change != expected.change {
		return fmt.Errorf("%s changed native metadata since preflight", label)
	}
	return nil
}

func txHashPath(path, label string, ops txOps) (string, bool, error) {
	name := filepath.Base(path)
	before, err := ops.root.Lstat(name)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("inspect %s: %w", label, err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return "", false, fmt.Errorf("%s must be a regular file, not a symlink or special file", label)
	}
	if before.Size() < 0 || before.Size() > txMaxFileBytes {
		return "", false, fmt.Errorf("%s exceeds %d-byte transaction file limit", label, txMaxFileBytes)
	}
	f, err := ops.root.Open(name)
	if err != nil {
		return "", false, fmt.Errorf("open %s: %w", label, err)
	}
	after, statErr := f.Stat()
	if statErr != nil {
		_ = f.Close()
		return "", false, fmt.Errorf("inspect opened %s: %w", label, statErr)
	}
	if !os.SameFile(before, after) || !after.Mode().IsRegular() {
		_ = f.Close()
		return "", false, fmt.Errorf("%s changed identity while opening", label)
	}
	if ops.hashAfterOpen != nil {
		if err := ops.hashAfterOpen(name); err != nil {
			return "", false, errors.Join(err, f.Close())
		}
	}
	hasher := sha256.New()
	written, copyErr := io.Copy(hasher, io.LimitReader(f, before.Size()+1))
	openedAfter, afterStatErr := f.Stat()
	closeErr := f.Close()
	pathAfter, pathErr := ops.root.Lstat(name)
	if err := errors.Join(copyErr, afterStatErr, closeErr, pathErr); err != nil {
		return "", false, fmt.Errorf("hash %s: %w", label, err)
	}
	if written != before.Size() || !os.SameFile(before, openedAfter) || !os.SameFile(before, pathAfter) ||
		before.Mode() != openedAfter.Mode() || before.Mode() != pathAfter.Mode() ||
		before.Size() != openedAfter.Size() || before.Size() != pathAfter.Size() ||
		!before.ModTime().Equal(openedAfter.ModTime()) || !before.ModTime().Equal(pathAfter.ModTime()) ||
		txChangeID(before) != txChangeID(openedAfter) || txChangeID(before) != txChangeID(pathAfter) {
		return "", false, fmt.Errorf("%s changed identity, size, or metadata while hashing", label)
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), true, nil
}

func txHash(body []byte) string {
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func txValidHash(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func txValidNativeWitness(value string) bool {
	if value == "" || len(value) > 256 {
		return false
	}
	for _, char := range value {
		if char != ':' && char != '-' && (char < '0' || char > '9') && (char < 'a' || char > 'z') {
			return false
		}
	}
	return true
}

func txValidOptionalNativeWitness(value string) bool {
	return value == "" || txValidNativeWitness(value)
}

func txStableFileID(info os.FileInfo) string {
	if value := txInfoFields(info, "Dev", "Ino"); value != "" {
		return "unix:" + value
	}
	if value := txInfoFields(info, "VolumeSerialNumber", "FileIndexHigh", "FileIndexLow"); value != "" {
		return "windows:" + value
	}
	if value := txNestedInfoFields(info, "CreationTime", "HighDateTime", "LowDateTime"); value != "" {
		return "windows-creation:" + value
	}
	return ""
}

func txChangeID(info os.FileInfo) string {
	if value := txInfoFields(info, "Ctime", "Ctimensec"); value != "" {
		return "ctime:" + value
	}
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
			return fmt.Sprintf("ctime:%d:%d", sec.Int(), nsec.Int())
		}
	}
	return ""
}

func txInfoFields(info os.FileInfo, names ...string) string {
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
	parts := make([]string, 0, len(names))
	for _, name := range names {
		field := value.FieldByName(name)
		if !field.IsValid() {
			return ""
		}
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			parts = append(parts, fmt.Sprintf("%d", field.Int()))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			parts = append(parts, fmt.Sprintf("%d", field.Uint()))
		default:
			return ""
		}
	}
	return strings.Join(parts, ":")
}

func txNestedInfoFields(info os.FileInfo, outer string, names ...string) string {
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
	value = value.FieldByName(outer)
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return ""
	}
	parts := make([]string, 0, len(names))
	for _, name := range names {
		field := value.FieldByName(name)
		if !field.IsValid() {
			return ""
		}
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			parts = append(parts, fmt.Sprintf("%d", field.Int()))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			parts = append(parts, fmt.Sprintf("%d", field.Uint()))
		default:
			return ""
		}
	}
	return strings.Join(parts, ":")
}

func txRename(dir, oldPath, newPath string, ops txOps) error {
	oldName, newName := oldPath, newPath
	if ops.renameHook != nil {
		if err := ops.renameHook(oldName, newName); err != nil {
			return fmt.Errorf("rename hook for %s to %s: %w", oldName, newName, err)
		}
	}
	return txRenameApplied(dir, oldPath, newPath, ops)
}

func txRenameSnapshots(dir, oldPath, newPath string, oldSnapshot, newSnapshot txSnapshot, ops txOps) error {
	if ops.renameHook != nil {
		if err := ops.renameHook(oldPath, newPath); err != nil {
			return fmt.Errorf("rename hook for %s to %s: %w", oldPath, newPath, err)
		}
	}
	if err := txValidateSnapshot(oldPath, "rollback rename source "+oldPath, oldSnapshot, ops); err != nil {
		return err
	}
	if err := txValidateSnapshot(newPath, "rollback rename destination "+newPath, newSnapshot, ops); err != nil {
		return err
	}
	return txRenameApplied(dir, oldPath, newPath, ops)
}

func txRenameNoReplaceSnapshots(dir, oldPath, newPath string, oldSnapshot, newSnapshot txSnapshot, ops txOps) error {
	if ops.renameHook != nil {
		if err := ops.renameHook(oldPath, newPath); err != nil {
			return fmt.Errorf("rename hook for %s to %s: %w", oldPath, newPath, err)
		}
	}
	if err := txValidateSnapshot(oldPath, "no-replace rename source "+oldPath, oldSnapshot, ops); err != nil {
		return err
	}
	if err := txValidateSnapshot(newPath, "no-replace rename destination "+newPath, newSnapshot, ops); err != nil {
		return err
	}
	if err := fsatomic.RenameNoReplace(ops.root, oldPath, newPath); err != nil {
		return fmt.Errorf("rename %s to %s without replacement: %w", filepath.Base(oldPath), filepath.Base(newPath), err)
	}
	if err := txSyncHeld(ops.syncFile); err != nil {
		return fmt.Errorf("sync directory after no-replace rename: %w", err)
	}
	if ops.syncObserve != nil {
		ops.syncObserve(dir)
	}
	return nil
}

func txRenameApplied(dir, oldPath, newPath string, ops txOps) error {
	if err := fsatomic.RenameNoReplace(ops.root, oldPath, newPath); err != nil {
		return fmt.Errorf("rename %s to %s without replacement: %w", filepath.Base(oldPath), filepath.Base(newPath), err)
	}
	if err := txSyncHeld(ops.syncFile); err != nil {
		return fmt.Errorf("sync directory after rename: %w", err)
	}
	if ops.syncObserve != nil {
		ops.syncObserve(dir)
	}
	return nil
}

func txRemoveSnapshot(dir, path, label string, snapshot txSnapshot, ops txOps) error {
	if err := txValidateSnapshot(path, label, snapshot, ops); err != nil {
		return err
	}
	quarantine, err := fsatomic.Quarantine(ops.root, path, txObjectQuarantinePrefix)
	if err != nil {
		return fmt.Errorf("quarantine %s: %w", label, err)
	}
	if err := txSyncHeld(ops.syncFile); err != nil {
		return errors.Join(err, quarantine.Close())
	}
	if ops.syncObserve != nil {
		ops.syncObserve(dir)
	}
	qops := ops
	qops.root = quarantine.Root()
	qsync, err := txOpenSyncRoot(quarantine.Root())
	if err != nil {
		return errors.Join(err, quarantine.Close())
	}
	qops.syncFile = qsync
	current, err := txSnapshotPath(quarantine.Name(), "quarantined "+label, qops)
	if err != nil || current.hash != snapshot.hash || current.identity != snapshot.identity || current.info == nil || snapshot.info == nil ||
		current.info.Mode() != snapshot.info.Mode() || current.info.Size() != snapshot.info.Size() || !current.info.ModTime().Equal(snapshot.info.ModTime()) {
		return errors.Join(fmt.Errorf("quarantined %s differs from its bound authority; preserving it", label), err, qsync.Close(), quarantine.Close())
	}
	if err := txPoint(ops, "object-quarantine-delete:"+filepath.Base(path)); err != nil {
		return errors.Join(err, qsync.Close(), quarantine.Close())
	}
	if err := txValidateSnapshot(quarantine.Name(), "quarantined "+label, current, qops); err != nil {
		return errors.Join(fmt.Errorf("quarantined %s changed at deletion boundary; preserving it: %w", label, err), qsync.Close(), quarantine.Close())
	}
	removeErr := quarantine.Remove()
	privateSyncErr := txSyncHeld(qsync)
	privateCloseErr := qsync.Close()
	parentSyncErr := txSyncHeld(ops.syncFile)
	if ops.syncObserve != nil {
		ops.syncObserve(dir)
	}
	return errors.Join(removeErr, privateSyncErr, privateCloseErr, parentSyncErr)
}

func txCleanupCreated(dir, path, label string, created os.FileInfo, ops txOps) error {
	current, err := txSnapshotPath(path, label, ops)
	if err != nil {
		return err
	}
	if !current.exists {
		return nil
	}
	createdIdentity := txStableFileID(created)
	if created == nil || current.info == nil || !os.SameFile(created, current.info) || createdIdentity == "" || current.identity != createdIdentity {
		return fmt.Errorf("%s changed identity before cleanup; preserving it", label)
	}
	return txRemoveSnapshot(dir, path, label, current, ops)
}

func txReadDir(ops txOps) ([]os.DirEntry, error) {
	return txReadDirBounded(ops, txMaxDirEntries)
}

func txReadDirBounded(ops txOps, maxEntries int) ([]os.DirEntry, error) {
	if maxEntries <= 0 {
		return nil, fmt.Errorf("transaction directory entry limit must be positive")
	}
	dir, err := ops.root.Open(".")
	if err != nil {
		return nil, fmt.Errorf("open rooted transaction directory: %w", err)
	}
	entries := make([]os.DirEntry, 0, min(txDirPageSize, maxEntries))
	for {
		page, readErr := dir.ReadDir(txDirPageSize)
		if len(page) > maxEntries-len(entries) {
			return nil, errors.Join(fmt.Errorf("rooted transaction directory exceeds %d-entry limit", maxEntries), dir.Close())
		}
		entries = append(entries, page...)
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, errors.Join(fmt.Errorf("read rooted transaction directory: %w", readErr), dir.Close())
		}
	}
	if closeErr := dir.Close(); closeErr != nil {
		return nil, fmt.Errorf("close rooted transaction directory: %w", closeErr)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	return entries, nil
}

func txCreateTemp(ops txOps, prefix string) (*os.File, string, error) {
	for attempt := 0; attempt < 100; attempt++ {
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			return nil, "", fmt.Errorf("read temporary-name entropy: %w", err)
		}
		name := prefix + hex.EncodeToString(nonce[:])
		file, err := ops.root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
		if err == nil {
			return file, name, nil
		}
		if !os.IsExist(err) {
			return nil, "", fmt.Errorf("create rooted temporary file %s: %w", name, err)
		}
	}
	return nil, "", fmt.Errorf("could not allocate a unique rooted temporary file after 100 attempts")
}

func txPoint(ops txOps, point string) error {
	if ops.faultAfter == nil {
		return nil
	}
	if err := ops.faultAfter(point); err != nil {
		return &txCrash{point: point, cause: err}
	}
	return nil
}
