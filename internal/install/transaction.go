package install

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
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/RamXX/machinery/internal/filelock"
	"github.com/RamXX/machinery/internal/fsatomic"
)

const (
	installJournalSchema       = 4
	installJournalDir          = ".machinery-install-journal"
	installJournalTombstone    = ".machinery-install-journal.committed"
	installJournalMetadata     = "journal.json"
	installJournalPrepared     = "PREPARED"
	installJournalCommitted    = "COMMITTED"
	installJournalScratch      = "scratch"
	installJournalDeletePrefix = ".machinery-journal-delete-"
	installRestoreStagePrefix  = ".machinery-install-restore-stage-"
	installRestorePrepareTag   = ".prepare-"
	installRestoreIntentFile   = "CREATION.json"
	installPrepareDeletePrefix = ".machinery-install-prep-delete-"
	installRecordDeletePrefix  = ".machinery-install-record-delete-"
	installMetadataTempPrefix  = ".machinery-journal-create-"
	installMetadataNextPrefix  = ".machinery-journal-next-"
	installMarkerTempPrefix    = ".machinery-marker-create-"
	installWitnessTempPrefix   = ".machinery-witness-create-"
	installRestoreDeletePrefix = ".machinery-install-restore-delete-"
	installActivationDir       = ".machinery-install-activation"
	installJournalMaxBytes     = 4 << 20
	installJournalMaxTargets   = 4096
)

type artifactTransaction struct {
	root        string
	journal     installJournal
	delegated   bool
	anchors     []*installAnchorCapability
	journalMu   sync.Mutex
	rollingBack bool
}

type installAnchorCapability struct {
	path  string
	id    string
	root  *os.Root
	items []installJournalItem
	tx    *artifactTransaction
}

var activeInstallAnchors = struct {
	sync.Mutex
	byPath map[string][]*installAnchorCapability
}{byPath: map[string][]*installAnchorCapability{}}

// afterInstallMutationValidation is a deterministic adversarial test hook.
// Production leaves it nil.
var afterInstallMutationValidation func(string)

// afterInstallPrivateDeletionValidation is a deterministic adversarial test
// hook. Production leaves it nil.
var afterInstallPrivateDeletionValidation func(*os.Root, string)

// afterInstallPostImageValidation interleaves adversarial changes after the
// first rollback ownership proof. Production leaves it nil; rollback repeats
// the proof after the hook before removing any live target.
var afterInstallPostImageValidation func(string)

// beforeInstallRestoreRename interleaves a concurrent target creation at the
// last validation/system-call boundary. Production leaves it nil.
var beforeInstallRestoreRename func(string)

// afterInstallRestoreBoundary simulates interruption immediately before or
// after publication of a fully witnessed private restoration stage.
var afterInstallRestoreBoundary func(string, string) error

// closeInstallFile is replaceable only by deterministic failure-injection
// tests. Durability helpers must surface both flush and close failures.
var closeInstallFile = func(f *os.File) error { return f.Close() }

var renameInstallNoReplace = fsatomic.RenameNoReplaceBetween

var errInstallForeignAuthority = errors.New("foreign install authority")

// afterInstallStageCreationBoundary is a deterministic subprocess crash-test
// seam. Production leaves it nil.
var afterInstallStageCreationBoundary func(string, string)

// afterInstallAuthorityRecordBoundary is a deterministic subprocess crash-test
// seam for fixed authority-record publication. Production leaves it nil.
var afterInstallAuthorityRecordBoundary func(string, string)

// removeInstallScratchDir and removeInstallScratchFile are replaceable only
// by deterministic failure-injection tests. Every returned cleanup callback
// must surface their errors to its caller.
var (
	removeInstallScratchDir = func(path string, durable bool) error {
		if durable {
			return durableRemoveAll(path)
		}
		return os.RemoveAll(path)
	}
	removeInstallScratchFile = func(path string, durable bool) error {
		var err error
		if durable {
			err = durableRemove(path)
		} else {
			err = os.Remove(path)
		}
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
)

type installJournal struct {
	Schema int                  `json:"schema"`
	Phase  string               `json:"phase"`
	Items  []installJournalItem `json:"items"`
}

type installJournalItem struct {
	Target      string   `json:"target"`
	Backup      string   `json:"backup"`
	Existed     bool     `json:"existed"`
	CreatedDirs []string `json:"created_dirs,omitempty"`
	Anchor      string   `json:"anchor"`
	AnchorID    string   `json:"anchor_id"`
	Digest      string   `json:"digest,omitempty"`
	PostState   string   `json:"post_state,omitempty"`
	PostDigest  string   `json:"post_digest,omitempty"`
	StageName   string   `json:"stage_name,omitempty"`
	StageID     string   `json:"stage_id,omitempty"`
	StageObject string   `json:"stage_object_id,omitempty"`
	StageDigest string   `json:"stage_digest,omitempty"`
	StageUse    string   `json:"stage_use,omitempty"`
}

type installStageCreationWitness struct {
	StageName string `json:"stage_name"`
	StageID   string `json:"stage_id"`
}

const (
	installPostUnknown  = "unknown"
	installPostAbsent   = "absent"
	installPostPresent  = "present"
	installStagePublish = "publish"
	installStageRestore = "restore"
)

func beginArtifactTransaction(paths []string) (*artifactTransaction, error) {
	clean, err := normalizeTransactionPaths(paths)
	if err != nil {
		return nil, err
	}
	if delegatedInstallOperation() {
		return delegatedArtifactTransaction(clean)
	}
	root, tombstone, err := installJournalPaths()
	if err != nil {
		return nil, err
	}
	if _, err := os.Lstat(root); err == nil {
		return nil, fmt.Errorf("install transaction journal already exists at %s; acquire the operation lock to recover it first", root)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if _, err := os.Lstat(tombstone); err == nil {
		return nil, fmt.Errorf("committed install transaction cleanup is still pending at %s", tombstone)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(root), 0o700); err != nil {
		return nil, err
	}
	tx := &artifactTransaction{root: root, journal: installJournal{Schema: installJournalSchema, Phase: "snapshotting"}}
	fail := func(primary error) (*artifactTransaction, error) {
		if errors.Is(primary, errInstallForeignAuthority) {
			return nil, errors.Join(primary, tx.closeAnchors())
		}
		return nil, errors.Join(primary, tx.abandonUnprepared())
	}
	for i, target := range clean {
		createdDirs, err := missingAncestorDirs(filepath.Dir(target))
		if err != nil {
			return fail(fmt.Errorf("inspect ancestors for install artifact %s: %w", target, err))
		}
		anchor, anchorID, err := installArtifactAnchor(filepath.Dir(target))
		if err != nil {
			return fail(fmt.Errorf("anchor install artifact %s: %w", target, err))
		}
		item := installJournalItem{Target: target, Backup: filepath.ToSlash(filepath.Join("snapshots", fmt.Sprintf("%06d", i))), CreatedDirs: createdDirs, Anchor: anchor, AnchorID: anchorID}
		tx.journal.Items = append(tx.journal.Items, item)
	}
	// Retain and verify every original parent before inspecting or snapshotting
	// targets. This makes the capability cover the complete read/snapshot/write
	// transaction, not merely the forward mutation phase.
	if err := tx.openAnchors(); err != nil {
		return fail(fmt.Errorf("retain install transaction parent capabilities: %w", err))
	}
	for i := range tx.journal.Items {
		item := &tx.journal.Items[i]
		info, statErr := tx.lstatItem(*item)
		switch {
		case statErr == nil:
			if !supportedArtifactMode(info.Mode()) {
				return fail(fmt.Errorf("unsupported install artifact type %s at %s", info.Mode().Type(), item.Target))
			}
			item.Existed = true
		case os.IsNotExist(statErr):
		case statErr != nil:
			return fail(fmt.Errorf("inspect install artifact %s: %w", item.Target, statErr))
		}
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		return fail(fmt.Errorf("create install transaction journal: %w", err))
	}
	if err := syncDir(filepath.Dir(root)); err != nil {
		return fail(fmt.Errorf("sync install journal parent: %w", err))
	}
	if err := writeJournalMetadata(root, tx.journal); err != nil {
		return fail(err)
	}
	if err := os.Mkdir(filepath.Join(root, "snapshots"), 0o700); err != nil {
		return fail(err)
	}
	for _, item := range tx.journal.Items {
		if item.Existed {
			if err := tx.snapshotItem(item, filepath.Join(root, filepath.FromSlash(item.Backup))); err != nil {
				return fail(fmt.Errorf("snapshot install artifact %s: %w", item.Target, err))
			}
		}
	}
	for i := range tx.journal.Items {
		item := &tx.journal.Items[i]
		if item.Existed {
			digest, err := stableArtifactTreeDigest(filepath.Join(root, filepath.FromSlash(item.Backup)))
			if err != nil {
				return fail(fmt.Errorf("digest install artifact snapshot %s: %w", item.Target, err))
			}
			item.Digest = digest
			item.PostState = installPostPresent
			postDigest, err := stableArtifactPostImageDigest(filepath.Join(root, filepath.FromSlash(item.Backup)))
			if err != nil {
				return fail(fmt.Errorf("digest exact install artifact post-image %s: %w", item.Target, err))
			}
			item.PostDigest = postDigest
		} else {
			item.PostState = installPostAbsent
		}
	}
	if err := replaceJournalMetadata(root, tx.journal); err != nil {
		return fail(fmt.Errorf("bind install transaction snapshot integrity: %w", err))
	}
	if err := syncTree(filepath.Join(root, "snapshots")); err != nil {
		return fail(fmt.Errorf("sync install transaction snapshots: %w", err))
	}
	if err := createJournalMarker(root, installJournalPrepared); err != nil {
		return fail(fmt.Errorf("prepare install transaction journal: %w", err))
	}
	return tx, nil
}

func installArtifactAnchor(path string) (string, string, error) {
	for {
		info, err := os.Lstat(path)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return "", "", fmt.Errorf("anchor %s is not a real directory", path)
			}
			identity, err := stableInstallDirIdentity(path, info)
			return path, identity, err
		}
		if !os.IsNotExist(err) {
			return "", "", err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", "", fmt.Errorf("no existing anchor for %s", path)
		}
		path = parent
	}
}

func (tx *artifactTransaction) openAnchors() error {
	seen := map[string]bool{}
	for _, item := range tx.journal.Items {
		if seen[item.Anchor] {
			continue
		}
		root, err := os.OpenRoot(item.Anchor)
		if err != nil {
			return errors.Join(err, tx.closeAnchors())
		}
		openedInfo, openedErr := root.Stat(".")
		pathInfo, pathErr := os.Lstat(item.Anchor)
		identity, identityErr := "", pathErr
		if pathErr == nil {
			identity, identityErr = stableInstallDirIdentity(item.Anchor, pathInfo)
		}
		if openedErr != nil || pathErr != nil || pathInfo.Mode()&os.ModeSymlink != 0 || !pathInfo.IsDir() || !os.SameFile(openedInfo, pathInfo) || identityErr != nil || identity != item.AnchorID {
			return errors.Join(fmt.Errorf("install transaction anchor %s changed while opening", item.Anchor), root.Close(), tx.closeAnchors())
		}
		capability := &installAnchorCapability{path: item.Anchor, id: item.AnchorID, root: root, tx: tx}
		for _, candidate := range tx.journal.Items {
			if candidate.Anchor == item.Anchor && candidate.AnchorID == item.AnchorID {
				capability.items = append(capability.items, candidate)
			}
		}
		tx.anchors = append(tx.anchors, capability)
		activeInstallAnchors.Lock()
		activeInstallAnchors.byPath[item.Anchor] = append(activeInstallAnchors.byPath[item.Anchor], capability)
		activeInstallAnchors.Unlock()
		seen[item.Anchor] = true
	}
	return nil
}

func (tx *artifactTransaction) capabilityForItem(item installJournalItem) (*os.Root, string, error) {
	for _, capability := range tx.anchors {
		if capability.path != item.Anchor || capability.id != item.AnchorID {
			continue
		}
		rel, err := filepath.Rel(item.Anchor, item.Target)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return nil, "", fmt.Errorf("install target %s escapes retained anchor %s", item.Target, item.Anchor)
		}
		return capability.root, rel, nil
	}
	return nil, "", fmt.Errorf("no retained parent capability for install target %s", item.Target)
}

func (tx *artifactTransaction) lstatItem(item installJournalItem) (os.FileInfo, error) {
	root, rel, err := tx.capabilityForItem(item)
	if err != nil {
		return nil, err
	}
	return root.Lstat(rel)
}

func (tx *artifactTransaction) snapshotItem(item installJournalItem, dst string) error {
	root, rel, err := tx.capabilityForItem(item)
	if err != nil {
		return err
	}
	return copyEntryFromRoot(root, rel, dst)
}

func (tx *artifactTransaction) closeAnchors() error {
	var errs []error
	for _, capability := range tx.anchors {
		activeInstallAnchors.Lock()
		entries := activeInstallAnchors.byPath[capability.path]
		for i, entry := range entries {
			if entry == capability {
				entries = append(entries[:i], entries[i+1:]...)
				break
			}
		}
		if len(entries) == 0 {
			delete(activeInstallAnchors.byPath, capability.path)
		} else {
			activeInstallAnchors.byPath[capability.path] = entries
		}
		activeInstallAnchors.Unlock()
		errs = append(errs, capability.root.Close())
	}
	tx.anchors = nil
	return errors.Join(errs...)
}

func installScratchDir(prefix string) (string, func() error, error) {
	root, active, err := transactionScratchRoot()
	if err != nil {
		return "", func() error { return nil }, err
	}
	if active {
		base := filepath.Join(root, installJournalScratch)
		if err := os.MkdirAll(base, 0o700); err != nil {
			return "", func() error { return nil }, err
		}
		dir, err := os.MkdirTemp(base, prefix+"-")
		if err != nil {
			return "", func() error { return nil }, err
		}
		if err := syncDir(base); err != nil {
			return "", func() error { return nil }, errors.Join(err, removeInstallScratchDir(dir, true))
		}
		return dir, func() error { return removeInstallScratchDir(dir, true) }, nil
	}
	dir, err := os.MkdirTemp("", prefix+"-")
	if err != nil {
		return "", func() error { return nil }, err
	}
	return dir, func() error { return removeInstallScratchDir(dir, false) }, nil
}

func installScratchFile(fallbackDir, prefix string) (*os.File, func() error, error) {
	root, active, err := transactionScratchRoot()
	if err != nil {
		return nil, func() error { return nil }, err
	}
	if active {
		base := filepath.Join(root, installJournalScratch)
		if err := os.MkdirAll(base, 0o700); err != nil {
			return nil, func() error { return nil }, err
		}
		f, err := os.CreateTemp(base, prefix+"-")
		if err != nil {
			return nil, func() error { return nil }, err
		}
		if err := syncDir(base); err != nil {
			return nil, func() error { return nil }, errors.Join(err, closeInstallFile(f), removeInstallScratchFile(f.Name(), true))
		}
		return f, func() error { return removeInstallScratchFile(f.Name(), true) }, nil
	}
	f, err := os.CreateTemp(fallbackDir, prefix+"-")
	if err != nil {
		return nil, func() error { return nil }, err
	}
	return f, func() error { return removeInstallScratchFile(f.Name(), false) }, nil
}

func transactionScratchRoot() (string, bool, error) {
	root, _, err := installJournalPaths()
	if err != nil {
		return "", false, err
	}
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if err := filelock.ValidatePrivateDir(root, info); err != nil {
		return "", false, err
	}
	prepared, err := validJournalMarker(root, installJournalPrepared)
	if err != nil {
		return "", false, err
	}
	if !prepared {
		return "", false, fmt.Errorf("install transaction scratch requested before PREPARED")
	}
	return root, true, nil
}

func normalizeTransactionPaths(paths []string) ([]string, error) {
	unique := map[string]string{}
	clean := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("transaction artifact path is empty")
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		abs = filepath.Clean(abs)
		accessPath, err := installArtifactAccessPath(abs)
		if err != nil {
			return nil, err
		}
		abs = accessPath
		identity, err := installArtifactPathIdentity(abs)
		if err != nil {
			return nil, err
		}
		if prior, ok := unique[identity]; ok {
			if prior != abs {
				return nil, fmt.Errorf("transaction artifacts resolve to the same path: %s and %s", prior, abs)
			}
			continue
		}
		unique[identity] = abs
		clean = append(clean, abs)
	}
	sort.Strings(clean)
	for i, path := range clean {
		for _, prior := range clean[:i] {
			if artifactSameOrNestedPath(prior, path) {
				return nil, fmt.Errorf("transaction artifacts overlap: %s and %s", prior, path)
			}
		}
	}
	return clean, nil
}

func artifactSameOrNestedPath(a, b string) bool {
	aID, aErr := installArtifactResolvedPath(a)
	bID, bErr := installArtifactResolvedPath(b)
	if aErr == nil && bErr == nil {
		a, b = aID, bID
	}
	return lexicalSameOrNestedPath(a, b)
}

func lexicalSameOrNestedPath(a, b string) bool {
	rel, err := filepath.Rel(a, b)
	if err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))) {
		return true
	}
	rel, err = filepath.Rel(b, a)
	return err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))))
}

func delegatedInstallOperation() bool {
	encoded := os.Getenv(installLockCapabilityEnv)
	if encoded == "" {
		return false
	}
	scope, err := installOperationScope()
	return err == nil && validateInstallLockCapability(encoded, scope) == nil
}

func delegatedArtifactTransaction(paths []string) (*artifactTransaction, error) {
	root, _, err := installJournalPaths()
	if err != nil {
		return nil, err
	}
	journal, phase, err := loadInstallJournal(root)
	if err != nil {
		return nil, fmt.Errorf("load parent install transaction: %w", err)
	}
	if phase != "prepared" {
		return nil, fmt.Errorf("parent install transaction is %s, want prepared", phase)
	}
	covered := map[string]bool{}
	for _, item := range journal.Items {
		identity, err := installArtifactPathIdentity(item.Target)
		if err != nil {
			return nil, err
		}
		covered[identity] = true
	}
	for _, path := range paths {
		identity, err := installArtifactPathIdentity(path)
		if err != nil || !covered[identity] {
			return nil, fmt.Errorf("lock-capability child attempted install artifact outside parent journal: %s", path)
		}
	}
	tx := &artifactTransaction{root: root, journal: journal, delegated: true}
	if err := tx.openAnchors(); err != nil {
		return nil, err
	}
	return tx, nil
}

func (tx *artifactTransaction) rollback() error {
	if tx == nil {
		return nil
	}
	if tx.delegated {
		return tx.closeAnchors()
	}
	tx.journalMu.Lock()
	tx.rollingBack = true
	tx.journalMu.Unlock()
	journal, phase, loadErr := loadInstallJournal(tx.root)
	if loadErr != nil || phase != "prepared" {
		return errors.Join(fmt.Errorf("load prepared install transaction for rollback: phase=%s: %w", phase, loadErr), tx.closeAnchors())
	}
	tx.journal = journal
	err := rollbackInstallJournal(tx.root, journal)
	err = errors.Join(err, tx.closeAnchors())
	if err == nil {
		tx.root = ""
	}
	return err
}

func (tx *artifactTransaction) commit() error {
	if tx == nil {
		return nil
	}
	if tx.delegated {
		return tx.closeAnchors()
	}
	journal, phase, err := loadInstallJournal(tx.root)
	if err != nil || phase != "prepared" {
		return errors.Join(fmt.Errorf("load prepared install transaction for commit: phase=%s: %w", phase, err), tx.rollback())
	}
	for _, item := range journal.Items {
		if item.PostState == installPostUnknown {
			return errors.Join(fmt.Errorf("install transaction post-image for %s is unknown", item.Target), tx.rollback())
		}
		matches, matchErr := installJournalPostImageMatches(item)
		if matchErr != nil {
			return errors.Join(fmt.Errorf("verify install transaction post-image for %s before commit: %w", item.Target, matchErr), tx.rollback())
		}
		if !matches {
			return errors.Join(fmt.Errorf("install transaction post-image for %s changed before commit", item.Target), tx.rollback())
		}
	}
	tx.journal = journal
	if err := createJournalMarker(tx.root, installJournalCommitted); err != nil {
		return errors.Join(fmt.Errorf("persist committed install transaction: %w", err), tx.rollback())
	}
	if err := finalizeInstallJournal(tx.root); err != nil {
		return errors.Join(fmt.Errorf("finalize committed install transaction: %w", err), tx.closeAnchors())
	}
	if err := tx.closeAnchors(); err != nil {
		return fmt.Errorf("close committed install transaction parent capabilities: %w", err)
	}
	tx.root = ""
	return nil
}

func (tx *artifactTransaction) abandonUnprepared() error {
	if tx == nil || tx.root == "" {
		return nil
	}
	err := errors.Join(finalizeInstallJournal(tx.root), tx.closeAnchors())
	if err == nil {
		tx.root = ""
	}
	return err
}

type installRecovery struct {
	restoredExecutable string
	executableIdentity string
	executableFile     *os.File
	activationPath     string
}

func recoverInstallTransaction() (installRecovery, error) {
	root, tombstone, err := installJournalPaths()
	if err != nil {
		return installRecovery{}, err
	}
	if err := cleanupJournalTombstone(tombstone); err != nil {
		return installRecovery{}, err
	}
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return installRecovery{}, nil
	}
	if err != nil {
		return installRecovery{}, err
	}
	if err := filelock.ValidatePrivateDir(root, info); err != nil {
		return installRecovery{}, fmt.Errorf("reject install transaction journal: %w", err)
	}
	empty, err := recoverInitialJournalMetadata(root)
	if err != nil {
		return installRecovery{}, fmt.Errorf("recover initial install journal metadata: %w", err)
	}
	if empty {
		return installRecovery{}, finalizeInstallJournal(root)
	}
	if err := recoverJournalMetadataReplacement(root); err != nil {
		return installRecovery{}, fmt.Errorf("recover install journal metadata replacement: %w", err)
	}
	journal, phase, err := loadInstallJournal(root)
	if err != nil {
		return installRecovery{}, fmt.Errorf("reject install transaction journal: %w", err)
	}
	switch phase {
	case "snapshotting":
		return installRecovery{}, finalizeInstallJournal(root)
	case "prepared":
		recovery := installRecovery{restoredExecutable: journalRunningExecutable(journal)}
		tx := &artifactTransaction{root: root, journal: journal}
		if err := tx.openAnchors(); err != nil {
			return installRecovery{}, err
		}
		tx.journalMu.Lock()
		tx.rollingBack = true
		tx.journalMu.Unlock()
		if err := errors.Join(rollbackInstallJournal(root, journal), tx.closeAnchors()); err != nil {
			return installRecovery{}, err
		}
		if recovery.restoredExecutable != "" {
			file, identity, err := retainActivationExecutable(recovery.restoredExecutable)
			if err != nil {
				return installRecovery{}, fmt.Errorf("bind restored executable identity: %w", err)
			}
			recovery.executableFile = file
			recovery.executableIdentity = identity
			recovery.activationPath, err = stageActivationExecutable(recovery.restoredExecutable, file, identity)
			if err != nil {
				return installRecovery{}, errors.Join(fmt.Errorf("stage restored executable activation: %w", err), closeInstallFile(file))
			}
		}
		return recovery, nil
	case "committed":
		return installRecovery{}, finalizeInstallJournal(root)
	default:
		return installRecovery{}, fmt.Errorf("reject install transaction journal: unsupported phase %q", phase)
	}
}

func recoverInitialJournalMetadata(path string) (bool, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return false, err
	}
	defer root.Close() //nolint:errcheck // recovery result is authoritative
	if _, err := root.Lstat(installJournalMetadata); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	dir, err := root.Open(".")
	if err != nil {
		return false, err
	}
	entries, readErr := readInstallDirBounded(dir, installArtifactMaxEntries, "initial install journal recovery")
	closeErr := closeInstallFile(dir)
	if err := errors.Join(readErr, closeErr); err != nil {
		return false, err
	}
	var expected []byte
	for _, entry := range entries {
		if !validInstallAuthorityTempName(installMetadataTempPrefix, entry.Name()) {
			continue
		}
		raw, err := readInstallRootRecord(root, entry.Name(), installJournalMaxBytes)
		if err != nil || !validInitialInstallJournalRecord(path, raw) {
			if retireErr := retireInstallAuthorityTemp(root, entry.Name()); retireErr != nil {
				return false, errors.Join(err, retireErr)
			}
			continue
		}
		if expected != nil && !bytes.Equal(expected, raw) {
			return false, fmt.Errorf("multiple distinct complete initial install journal records exist; preserving them")
		}
		expected = raw
	}
	if expected != nil {
		if err := recoverInstallAuthorityRecord(root, installJournalMetadata, installMetadataTempPrefix, expected, installJournalMaxBytes); err != nil {
			return false, err
		}
		return false, nil
	}
	dir, err = root.Open(".")
	if err != nil {
		return false, err
	}
	remaining, readErr := readInstallDirBounded(dir, 0, "abandoned initial install journal")
	closeErr = closeInstallFile(dir)
	if err := errors.Join(readErr, closeErr); err != nil || len(remaining) != 0 {
		return false, errors.Join(err, fmt.Errorf("initial install journal has foreign residue; preserving it"))
	}
	return true, nil
}

func validInitialInstallJournalRecord(root string, raw []byte) bool {
	var journal installJournal
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&journal); err != nil {
		return false
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return false
	}
	return journal.Schema == installJournalSchema && journal.Phase == "snapshotting" && len(journal.Items) <= installJournalMaxTargets && validateJournalItems(root, journal.Items, false) == nil
}

func activationExecutableIdentity(path string) (string, error) {
	f, identity, err := retainActivationExecutable(path)
	if err != nil {
		return "", err
	}
	return identity, closeInstallFile(f)
}

func retainActivationExecutable(path string) (*os.File, string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, "", fmt.Errorf("restored executable %s is not a real regular file", path)
	}
	f, err := openActivationExecutable(path)
	if err != nil {
		return nil, "", err
	}
	opened, statErr := f.Stat()
	if statErr != nil || !os.SameFile(info, opened) {
		return nil, "", errors.Join(fmt.Errorf("restored executable %s changed while opening", path), statErr, closeInstallFile(f))
	}
	identity, hashErr := activationOpenFileIdentity(f)
	current, currentErr := os.Lstat(path)
	if hashErr != nil || currentErr != nil || !os.SameFile(info, current) {
		return nil, "", errors.Join(hashErr, currentErr, fmt.Errorf("restored executable %s changed while hashing", path), closeInstallFile(f))
	}
	return f, identity, nil
}

func activationOpenFileIdentity(f *os.File) (string, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > installArtifactMaxFileBytes {
		return "", fmt.Errorf("activation executable exceeds fixed file bounds")
	}
	hash := sha256.New()
	written, copyErr := io.Copy(hash, io.LimitReader(f, info.Size()+1))
	after, statErr := f.Stat()
	if err := errors.Join(copyErr, statErr); err != nil {
		return "", err
	}
	if written != info.Size() || !sameInstallArtifactInfo(info, after) {
		return "", fmt.Errorf("activation executable changed while hashing")
	}
	return fmt.Sprintf("sha256:%x:size:%d:mode:%o", hash.Sum(nil), info.Size(), uint32(info.Mode())), nil
}

func journalRunningExecutable(journal installJournal) string {
	running, err := runningInstallExecutable()
	if err != nil {
		return ""
	}
	running, err = filepath.Abs(running)
	if err != nil {
		return ""
	}
	for _, item := range journal.Items {
		if item.Existed && sameInstallPath(item.Target, running) {
			return item.Target
		}
	}
	return ""
}

func runningInstallExecutable() (string, error) {
	if canonical := os.Getenv(activationCanonicalExecutableEnv); canonical != "" {
		if !filepath.IsAbs(canonical) || filepath.Clean(canonical) != canonical {
			return "", fmt.Errorf("invalid canonical activation executable %q", canonical)
		}
		return canonical, nil
	}
	return os.Executable()
}

func rollbackInstallJournal(root string, journal installJournal) error {
	var errs []error
	created := map[string]bool{}
	protectedCreated := map[string]bool{}
	for i := len(journal.Items) - 1; i >= 0; i-- {
		item := journal.Items[i]
		for _, dir := range item.CreatedDirs {
			created[dir] = true
		}
		handled, stagedErr := rollbackStagedInstallItem(root, item)
		if stagedErr != nil {
			for _, dir := range item.CreatedDirs {
				protectedCreated[dir] = true
			}
			errs = append(errs, fmt.Errorf("resume staged rollback for %s: %w", item.Target, stagedErr))
			continue
		}
		if handled {
			continue
		}
		matches, matchErr := installJournalPostImageMatches(item)
		if matchErr != nil {
			for _, dir := range item.CreatedDirs {
				protectedCreated[dir] = true
			}
			errs = append(errs, fmt.Errorf("verify transaction-written post-image %s: %w", item.Target, matchErr))
			continue
		}
		if !matches {
			for _, dir := range item.CreatedDirs {
				protectedCreated[dir] = true
			}
			errs = append(errs, fmt.Errorf("preserve concurrently changed install artifact %s: live state does not match durable transaction post-image", item.Target))
			continue
		}
		if afterInstallPostImageValidation != nil {
			afterInstallPostImageValidation(item.Target)
		}
		matches, matchErr = installJournalPostImageMatches(item)
		if matchErr != nil {
			for _, dir := range item.CreatedDirs {
				protectedCreated[dir] = true
			}
			errs = append(errs, fmt.Errorf("reverify transaction-written post-image %s: %w", item.Target, matchErr))
			continue
		}
		if !matches {
			for _, dir := range item.CreatedDirs {
				protectedCreated[dir] = true
			}
			errs = append(errs, fmt.Errorf("preserve concurrently changed install artifact %s: live state changed after post-image validation", item.Target))
			continue
		}
		if err := durableRemoveAll(item.Target); err != nil {
			for _, dir := range item.CreatedDirs {
				protectedCreated[dir] = true
			}
			errs = append(errs, fmt.Errorf("remove changed %s: %w", item.Target, err))
			continue
		}
		if item.Existed {
			if err := stageInstallEntryNoReplace(filepath.Join(root, filepath.FromSlash(item.Backup)), item.Target, installStageRestore); err != nil {
				for _, dir := range item.CreatedDirs {
					protectedCreated[dir] = true
				}
				errs = append(errs, fmt.Errorf("restore %s: %w", item.Target, err))
			}
		}
	}
	var dirs []string
	for dir := range created {
		dirs = append(dirs, dir)
	}
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, dir := range dirs {
		if protectedCreated[dir] {
			continue
		}
		if err := durableRemoveEmptyDir(dir); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) != 0 {
		return fmt.Errorf("install rollback incomplete; recoverable journal retained at %s: %w", root, errors.Join(errs...))
	}
	return finalizeInstallJournal(root)
}

func installJournalPostImageMatches(item installJournalItem) (bool, error) {
	info, err := os.Lstat(item.Target)
	switch item.PostState {
	case installPostAbsent:
		if os.IsNotExist(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	case installPostPresent:
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
		}
		if !supportedArtifactMode(info.Mode()) || item.PostDigest == "" {
			return false, nil
		}
		digest, err := stableArtifactPostImageDigest(item.Target)
		if err != nil {
			return false, err
		}
		return digest == item.PostDigest, nil
	case installPostUnknown:
		return false, nil
	default:
		return false, fmt.Errorf("invalid post-image state %q", item.PostState)
	}
}

func installJournalPaths() (string, string, error) {
	receipt, err := installationReceiptPath()
	if err != nil {
		return "", "", err
	}
	dir := filepath.Dir(receipt)
	if info, statErr := os.Stat(dir); statErr == nil && info.IsDir() {
		if resolved, resolveErr := filepath.EvalSymlinks(dir); resolveErr == nil {
			dir = resolved
		}
	}
	return filepath.Join(dir, installJournalDir), filepath.Join(dir, installJournalTombstone), nil
}

func installAuthorityRecordPoint(family, point string) {
	if afterInstallAuthorityRecordBoundary != nil {
		afterInstallAuthorityRecordBoundary(family, point)
	}
}

func newInstallAuthorityTempName(prefix string) (string, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(nonce[:]), nil
}

func validInstallAuthorityTempName(prefix, name string) bool {
	suffix := strings.TrimPrefix(name, prefix)
	if suffix == name || len(suffix) != 32 || suffix != strings.ToLower(suffix) {
		return false
	}
	_, err := hex.DecodeString(suffix)
	return err == nil
}

func readInstallRootRecord(root *os.Root, name string, limit int64) ([]byte, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Size() < 0 || before.Size() > limit || !privateFilePermissionsOK(before) {
		return nil, fmt.Errorf("install authority record %s is not a bounded private regular file", name)
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	opened, statErr := f.Stat()
	raw, readErr := io.ReadAll(io.LimitReader(f, limit+1))
	after, afterErr := f.Stat()
	live, liveErr := root.Lstat(name)
	closeErr := closeInstallFile(f)
	if err := errors.Join(statErr, readErr, afterErr, liveErr, closeErr); err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit || int64(len(raw)) != before.Size() || !sameInstallArtifactInfo(before, opened) || !sameInstallArtifactInfo(opened, after) || !sameInstallArtifactInfo(after, live) {
		return nil, fmt.Errorf("install authority record %s changed while reading", name)
	}
	return raw, nil
}

func retireInstallAuthorityTemp(root *os.Root, name string) error {
	witness, exists, err := captureInstallRemovalWitness(root, name)
	if err != nil || !exists {
		return errors.Join(err, fmt.Errorf("install authority temp %s has no exact deletion witness", name))
	}
	quarantined, err := fsatomic.Quarantine(root, name, installRecordDeletePrefix)
	if err != nil {
		return err
	}
	privateWitness, exists, err := captureInstallRemovalWitness(quarantined.Root(), quarantined.Name())
	if err != nil || !exists || !sameInstallRemovalWitnessAfterMove(witness, privateWitness) {
		return errors.Join(err, fmt.Errorf("install authority temp %s changed during retirement; preserving it", name), quarantined.Close())
	}
	return quarantined.Remove()
}

func recoverInstallAuthorityRecord(root *os.Root, name, prefix string, expected []byte, limit int64) error {
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := readInstallDirBounded(dir, installArtifactMaxEntries, "install authority record recovery")
	closeErr := closeInstallFile(dir)
	if err := errors.Join(readErr, closeErr); err != nil {
		return err
	}
	valid := ""
	var invalid []string
	for _, entry := range entries {
		if !validInstallAuthorityTempName(prefix, entry.Name()) {
			continue
		}
		raw, err := readInstallRootRecord(root, entry.Name(), limit)
		if err == nil && bytes.Equal(raw, expected) {
			if valid != "" {
				return fmt.Errorf("multiple complete install authority temps exist for %s; preserving them", name)
			}
			valid = entry.Name()
			continue
		}
		invalid = append(invalid, entry.Name())
	}
	live, liveErr := readInstallRootRecord(root, name, limit)
	switch {
	case liveErr == nil:
		if !bytes.Equal(live, expected) {
			return fmt.Errorf("%w: fixed install authority record %s contains foreign data; preserving it", errInstallForeignAuthority, name)
		}
	case !os.IsNotExist(liveErr):
		return liveErr
	case valid != "":
		if err := fsatomic.RenameNoReplace(root, valid, name); err != nil {
			return fmt.Errorf("recover install authority record %s without replacement: %w", name, err)
		}
		if err := syncRootRelativeDir(root, "."); err != nil {
			return err
		}
		valid = ""
	}
	for _, candidate := range invalid {
		if err := retireInstallAuthorityTemp(root, candidate); err != nil {
			return err
		}
	}
	if valid != "" {
		if err := retireInstallAuthorityTemp(root, valid); err != nil {
			return err
		}
	}
	return nil
}

func publishInstallAuthorityRecord(root *os.Root, name, prefix, family string, raw []byte, limit int64) error {
	if int64(len(raw)) > limit {
		return fmt.Errorf("install authority record %s exceeds %d bytes", name, limit)
	}
	if err := recoverInstallAuthorityRecord(root, name, prefix, raw, limit); err != nil {
		return err
	}
	if live, err := readInstallRootRecord(root, name, limit); err == nil {
		if bytes.Equal(live, raw) {
			return nil
		}
		return fmt.Errorf("%w: fixed install authority record %s contains foreign data; preserving it", errInstallForeignAuthority, name)
	} else if !os.IsNotExist(err) {
		return err
	}
	temp, err := newInstallAuthorityTempName(prefix)
	if err != nil {
		return err
	}
	f, err := root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	installAuthorityRecordPoint(family, "after-create")
	split := len(raw) / 2
	if split == 0 {
		split = len(raw)
	}
	if _, err := f.Write(raw[:split]); err != nil {
		return errors.Join(err, closeInstallFile(f))
	}
	installAuthorityRecordPoint(family, "partial-write")
	if _, err := f.Write(raw[split:]); err != nil {
		return errors.Join(err, closeInstallFile(f))
	}
	installAuthorityRecordPoint(family, "after-write")
	if err := f.Sync(); err != nil {
		return errors.Join(err, closeInstallFile(f))
	}
	installAuthorityRecordPoint(family, "fsync")
	if err := closeInstallFile(f); err != nil {
		return err
	}
	installAuthorityRecordPoint(family, "close")
	if err := syncRootRelativeDir(root, "."); err != nil {
		return err
	}
	installAuthorityRecordPoint(family, "pre-rename")
	if err := fsatomic.RenameNoReplace(root, temp, name); err != nil {
		return fmt.Errorf("%w: publish install authority record %s without replacement: %w", errInstallForeignAuthority, name, err)
	}
	installAuthorityRecordPoint(family, "post-rename")
	return syncRootRelativeDir(root, ".")
}

func writeJournalMetadata(root string, journal installJournal) error {
	raw, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	rooted, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer rooted.Close() //nolint:errcheck // publication result is authoritative
	return publishInstallAuthorityRecord(rooted, installJournalMetadata, installMetadataTempPrefix, "metadata", raw, installJournalMaxBytes)
}

func replaceJournalMetadata(root string, journal installJournal) error {
	if err := recoverJournalMetadataReplacement(root); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	path := filepath.Join(root, installJournalMetadata)
	next := path + ".next"
	rooted, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	if err := publishInstallAuthorityRecord(rooted, filepath.Base(next), installMetadataNextPrefix, "metadata-next", raw, installJournalMaxBytes); err != nil {
		return errors.Join(err, rooted.Close())
	}
	defer rooted.Close()
	oldWitness, exists, err := captureInstallRemovalWitness(rooted, installJournalMetadata)
	if err != nil || !exists {
		return errors.Join(err, fmt.Errorf("install journal metadata disappeared before replacement"), retireInstallAuthorityTemp(rooted, filepath.Base(next)))
	}
	if err := revalidateInstallRemovalWitness(rooted, installJournalMetadata, oldWitness); err != nil {
		return errors.Join(err, retireInstallAuthorityTemp(rooted, filepath.Base(next)))
	}
	quarantined, err := fsatomic.Quarantine(rooted, installJournalMetadata, installJournalDeletePrefix)
	if err != nil {
		return errors.Join(err, retireInstallAuthorityTemp(rooted, filepath.Base(next)))
	}
	privateWitness, exists, err := captureInstallRemovalWitness(quarantined.Root(), quarantined.Name())
	if err != nil || !exists || !sameInstallRemovalWitnessAfterMove(oldWitness, privateWitness) {
		return errors.Join(err, fmt.Errorf("install journal metadata changed while entering private replacement authority"), quarantined.Close())
	}
	if err := fsatomic.RenameNoReplace(rooted, filepath.Base(next), installJournalMetadata); err != nil {
		return errors.Join(err, quarantined.Restore(), quarantined.Close())
	}
	if err := syncRootRelativeDir(rooted, "."); err != nil {
		return errors.Join(err, quarantined.Close())
	}
	if err := revalidateInstallRemovalWitness(quarantined.Root(), quarantined.Name(), privateWitness); err != nil {
		return errors.Join(fmt.Errorf("retired install journal metadata changed; preserving it: %w", err), quarantined.Close())
	}
	return quarantined.Remove()
}

func recoverJournalMetadataReplacement(root string) error {
	rooted, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer rooted.Close()
	dir, err := rooted.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := readInstallDirBounded(dir, installArtifactMaxEntries, "install journal replacement")
	closeErr := closeInstallFile(dir)
	if err := errors.Join(readErr, closeErr); err != nil {
		return err
	}
	var quarantined *fsatomic.Quarantined
	for _, entry := range entries {
		if validInstallAuthorityTempName(installMetadataNextPrefix, entry.Name()) {
			if err := retireInstallAuthorityTemp(rooted, entry.Name()); err != nil {
				return err
			}
			continue
		}
		if !strings.HasPrefix(entry.Name(), installJournalDeletePrefix) {
			continue
		}
		candidate, err := fsatomic.ResumeQuarantine(rooted, entry.Name(), installJournalMetadata)
		if err != nil {
			return err
		}
		if quarantined != nil {
			return errors.Join(fmt.Errorf("multiple install journal metadata replacement authorities exist"), candidate.Close(), quarantined.Close())
		}
		quarantined = candidate
	}
	if quarantined == nil {
		if _, nextErr := rooted.Lstat(installJournalMetadata + ".next"); nextErr == nil {
			if _, liveErr := rooted.Lstat(installJournalMetadata); liveErr != nil {
				return errors.Join(liveErr, fmt.Errorf("staged install journal metadata exists without live or retired authority; preserving it"))
			}
			return retireInstallAuthorityTemp(rooted, installJournalMetadata+".next")
		} else if !os.IsNotExist(nextErr) {
			return nextErr
		}
		return nil
	}
	if _, err := quarantined.Root().Lstat(quarantined.Name()); errors.Is(err, os.ErrNotExist) {
		return quarantined.FinishEmpty()
	} else if err != nil {
		return errors.Join(err, quarantined.Close())
	}
	if err := validateInstallDeletionQuarantine(quarantined, true); err != nil {
		return errors.Join(err, quarantined.Close())
	}
	witness, exists, err := captureInstallRemovalWitness(quarantined.Root(), quarantined.Name())
	if err != nil || !exists {
		return errors.Join(err, fmt.Errorf("retired install journal metadata has no stable witness"), quarantined.Close())
	}
	_, liveErr := rooted.Lstat(installJournalMetadata)
	_, nextErr := rooted.Lstat(installJournalMetadata + ".next")
	switch {
	case errors.Is(liveErr, os.ErrNotExist):
		if err := quarantined.Restore(); err != nil {
			return errors.Join(err, quarantined.Close())
		}
		if nextErr == nil {
			return retireInstallAuthorityTemp(rooted, installJournalMetadata+".next")
		}
		if !errors.Is(nextErr, os.ErrNotExist) {
			return nextErr
		}
		return nil
	case liveErr != nil:
		return errors.Join(liveErr, quarantined.Close())
	case nextErr == nil:
		return errors.Join(fmt.Errorf("ambiguous install journal replacement has live and staged metadata"), quarantined.Close())
	case !errors.Is(nextErr, os.ErrNotExist):
		return errors.Join(nextErr, quarantined.Close())
	}
	if err := revalidateInstallRemovalWitness(quarantined.Root(), quarantined.Name(), witness); err != nil {
		return errors.Join(err, quarantined.Close())
	}
	return quarantined.Remove()
}

func createJournalMarker(root, name string) error {
	rooted, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer rooted.Close() //nolint:errcheck // publication result is authoritative
	prefix := installMarkerTempPrefix + strings.ToLower(name) + "-"
	raw := []byte(name + "\n")
	return publishInstallAuthorityRecord(rooted, name, prefix, "marker:"+name, raw, int64(len(raw)))
}

func loadInstallJournal(root string) (installJournal, string, error) {
	metadata := filepath.Join(root, installJournalMetadata)
	raw, err := readPrivateRegularFile(metadata, installJournalMaxBytes)
	if err != nil {
		return installJournal{}, "", err
	}
	var journal installJournal
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&journal); err != nil {
		return installJournal{}, "", fmt.Errorf("parse %s: %w", metadata, err)
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = fmt.Errorf("trailing JSON value")
		}
		return installJournal{}, "", fmt.Errorf("parse %s: %w", metadata, err)
	}
	if journal.Schema == 1 {
		return installJournal{}, "", fmt.Errorf("legacy install transaction journal schema 1 cannot be recovered safely; no artifact was mutated")
	}
	if journal.Schema == 2 {
		return installJournal{}, "", fmt.Errorf("legacy install transaction journal schema 2 has no durable post-images and cannot be recovered safely; no artifact was mutated")
	}
	if journal.Schema == 3 {
		return installJournal{}, "", fmt.Errorf("legacy install transaction journal schema 3 has no staged-restoration authority and cannot be recovered safely; no artifact was mutated")
	}
	if journal.Schema != installJournalSchema || journal.Phase != "snapshotting" || len(journal.Items) > installJournalMaxTargets {
		return installJournal{}, "", fmt.Errorf("invalid install transaction journal header")
	}
	prepared, err := validJournalMarker(root, installJournalPrepared)
	if err != nil {
		return installJournal{}, "", err
	}
	committed, err := validJournalMarker(root, installJournalCommitted)
	if err != nil {
		return installJournal{}, "", err
	}
	if committed && !prepared {
		return installJournal{}, "", fmt.Errorf("COMMITTED marker exists without PREPARED")
	}
	if err := validateJournalItems(root, journal.Items, prepared); err != nil {
		return installJournal{}, "", err
	}
	if committed {
		return journal, "committed", nil
	}
	if prepared {
		return journal, "prepared", nil
	}
	return journal, "snapshotting", nil
}

func validateJournalItems(root string, items []installJournalItem, requireBackups bool) error {
	type validatedTarget struct {
		identity string
		path     string
	}
	// Preserve journal order explicitly. A corrupt later target may overlap
	// several prior targets; its diagnostic must identify the first declared
	// prior target, never whichever key a Go map happens to return first.
	var seenTargets []validatedTarget
	seenBackups := map[string]bool{}
	for _, item := range items {
		if !filepath.IsAbs(item.Target) || filepath.Clean(item.Target) != item.Target {
			return fmt.Errorf("journal target is not a clean absolute path: %q", item.Target)
		}
		identity, err := installArtifactPathIdentity(item.Target)
		if err != nil {
			return err
		}
		for _, prior := range seenTargets {
			if identity == prior.identity || artifactSameOrNestedPath(prior.path, item.Target) {
				return fmt.Errorf("journal targets overlap or repeat: %s and %s", prior.path, item.Target)
			}
		}
		seenTargets = append(seenTargets, validatedTarget{identity: identity, path: item.Target})
		if !filepath.IsAbs(item.Anchor) || filepath.Clean(item.Anchor) != item.Anchor || item.AnchorID == "" || !pathAtOrBelow(foldedInstallPath(item.Anchor), foldedInstallPath(item.Target)) {
			return fmt.Errorf("journal anchor is invalid for target %s", item.Target)
		}
		anchorInfo, err := os.Lstat(item.Anchor)
		if err != nil || anchorInfo.Mode()&os.ModeSymlink != 0 || !anchorInfo.IsDir() {
			return fmt.Errorf("journal anchor %s is missing or not a real directory", item.Anchor)
		}
		anchorID, err := stableInstallDirIdentity(item.Anchor, anchorInfo)
		if err != nil || anchorID != item.AnchorID {
			return fmt.Errorf("journal anchor %s changed identity", item.Anchor)
		}
		backup := filepath.Clean(filepath.FromSlash(item.Backup))
		if filepath.IsAbs(backup) || backup == "." || backup == ".." || strings.HasPrefix(backup, ".."+string(os.PathSeparator)) || filepath.Dir(backup) != "snapshots" {
			return fmt.Errorf("journal backup escapes snapshots: %q", item.Backup)
		}
		if seenBackups[backup] {
			return fmt.Errorf("journal backup repeats: %q", item.Backup)
		}
		seenBackups[backup] = true
		full := filepath.Join(root, backup)
		info, statErr := os.Lstat(full)
		if item.Existed {
			if statErr != nil && (!os.IsNotExist(statErr) || requireBackups) {
				return fmt.Errorf("journal backup %s is missing: %w", full, statErr)
			}
			if statErr == nil && !supportedArtifactMode(info.Mode()) {
				return fmt.Errorf("journal backup %s is missing or unsupported", full)
			}
			if requireBackups {
				if item.Digest == "" {
					return fmt.Errorf("journal backup %s has no integrity digest", full)
				}
				digest, err := stableArtifactTreeDigest(full)
				if err != nil {
					return fmt.Errorf("digest journal backup %s: %w", full, err)
				}
				if digest != item.Digest {
					return fmt.Errorf("journal backup %s failed integrity verification", full)
				}
			}
		} else if statErr == nil {
			return fmt.Errorf("journal says missing target %s but carries backup %s", item.Target, full)
		} else if !os.IsNotExist(statErr) {
			return statErr
		}
		if requireBackups {
			switch item.PostState {
			case installPostAbsent, installPostUnknown:
				if item.PostDigest != "" {
					return fmt.Errorf("journal post-image for %s has a digest in state %s", item.Target, item.PostState)
				}
			case installPostPresent:
				if !validArtifactDigest(item.PostDigest) {
					return fmt.Errorf("journal post-image digest for %s is malformed", item.Target)
				}
			default:
				return fmt.Errorf("journal post-image state for %s is invalid", item.Target)
			}
		}
		if err := validateInstallStageJournalItem(item); err != nil {
			return fmt.Errorf("journal restoration stage for %s is invalid: %w", item.Target, err)
		}
		for _, dir := range item.CreatedDirs {
			if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir || !pathContains(dir, item.Target) {
				return fmt.Errorf("journal created directory %q is not a clean ancestor of %s", dir, item.Target)
			}
		}
	}
	return nil
}

func validateInstallStageJournalItem(item installJournalItem) error {
	allEmpty := item.StageName == "" && item.StageID == "" && item.StageObject == "" && item.StageDigest == "" && item.StageUse == ""
	if allEmpty {
		return nil
	}
	if filepath.Base(item.StageName) != item.StageName || !strings.HasPrefix(item.StageName, installRestoreStagePrefix) {
		return fmt.Errorf("stage name is not a private basename")
	}
	suffix := strings.TrimPrefix(item.StageName, installRestoreStagePrefix)
	if len(suffix) != 32 {
		return fmt.Errorf("stage name has an invalid nonce")
	}
	if _, err := hex.DecodeString(suffix); err != nil {
		return fmt.Errorf("stage name has an invalid nonce")
	}
	if item.StageUse != installStagePublish && item.StageUse != installStageRestore {
		return fmt.Errorf("stage purpose is invalid")
	}
	if !validArtifactDigest(item.StageDigest) {
		return fmt.Errorf("stage digest is invalid")
	}
	if item.StageID == "" && item.StageObject != "" {
		return fmt.Errorf("stage identities are incomplete")
	}
	return nil
}

func readPrivateRegularFile(path string, limit int64) ([]byte, error) {
	return readInstallRegularFileExact(path, limit, true)
}

func readInstallRegularFileExact(path string, limit int64, requirePrivate bool) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || requirePrivate && !privateFilePermissionsOK(before) {
		return nil, fmt.Errorf("%s is not an allowed regular file", path)
	}
	if before.Size() < 0 || before.Size() > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", path, limit)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !sameInstallArtifactInfo(before, opened) {
		return nil, fmt.Errorf("%s changed while opening", path)
	}
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", path, limit)
	}
	heldAfter, heldErr := f.Stat()
	liveAfter, liveErr := os.Lstat(path)
	if err := errors.Join(heldErr, liveErr); err != nil {
		return nil, err
	}
	if int64(len(raw)) != before.Size() || !sameInstallArtifactInfo(before, heldAfter) || !sameInstallArtifactInfo(before, liveAfter) {
		return nil, fmt.Errorf("%s changed while reading", path)
	}
	return raw, nil
}

func validJournalMarker(root, name string) (bool, error) {
	rooted, err := os.OpenRoot(root)
	if err != nil {
		return false, err
	}
	prefix := installMarkerTempPrefix + strings.ToLower(name) + "-"
	raw := []byte(name + "\n")
	recoverErr := recoverInstallAuthorityRecord(rooted, name, prefix, raw, int64(len(raw)))
	closeErr := rooted.Close()
	if err := errors.Join(recoverErr, closeErr); err != nil {
		return false, err
	}
	path := filepath.Join(root, name)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || !privateFilePermissionsOK(info) {
		return false, fmt.Errorf("journal marker %s is not a private regular file", path)
	}
	raw, err = readPrivateRegularFile(path, int64(len(name)+1))
	if err != nil || string(raw) != name+"\n" {
		return false, fmt.Errorf("journal marker %s is malformed", path)
	}
	return true, nil
}

func finalizeInstallJournal(root string) error {
	if root == "" {
		return nil
	}
	tombstone := filepath.Join(filepath.Dir(root), installJournalTombstone)
	if err := cleanupJournalTombstone(tombstone); err != nil {
		return err
	}
	if err := durableRename(root, tombstone); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return cleanupJournalTombstone(tombstone)
}

func cleanupJournalTombstone(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("install journal tombstone %s is not a real directory", path)
	}
	return durableRemoveAll(path)
}

func supportedArtifactMode(mode os.FileMode) bool {
	return mode.IsRegular() || mode.IsDir() || mode&os.ModeSymlink != 0
}

func artifactTreeDigest(path string) (string, error) {
	return artifactDigest(path, false)
}

func artifactDigest(path string, exactMetadata bool) (string, error) {
	h := sha256.New()
	budget := &installArtifactBudget{}
	if err := digestArtifactEntry(h, path, ".", exactMetadata, budget, 0); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

func stableArtifactTreeDigest(path string) (string, error) {
	return stableArtifactDigest(path, false)
}

func stableArtifactPostImageDigest(path string) (string, error) {
	return stableArtifactDigest(path, true)
}

func stableArtifactDigest(path string, exactMetadata bool) (string, error) {
	var prior string
	for pass := 1; pass <= 3; pass++ {
		digest, err := artifactDigest(path, exactMetadata)
		if err != nil {
			return "", err
		}
		if pass > 1 && digest != prior {
			return "", fmt.Errorf("artifact tree changed during digest witness")
		}
		prior = digest
	}
	return prior, nil
}

func validArtifactDigest(digest string) bool {
	if len(digest) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(digest, "sha256:") {
		return false
	}
	for _, r := range strings.TrimPrefix(digest, "sha256:") {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

type digestWriter interface {
	Write([]byte) (int, error)
}

const (
	installArtifactMaxEntries          = 100_000
	installArtifactMaxFileBytes  int64 = 1 << 30
	installArtifactMaxTotalBytes int64 = 4 << 30
	// A traversal root is depth zero. This single portable ceiling applies to
	// every recursive installer authority walk, independent of branching.
	installMaxTraversalDepth = 64
)

type installArtifactBudget struct {
	entries int
	bytes   int64
}

var installArtifactCopyAfterOpen = func(string) {}

func validateInstallTraversalDepth(depth int, path string) error {
	if depth < 0 || depth > installMaxTraversalDepth {
		return fmt.Errorf("install tree %s exceeds %d-level traversal depth limit", filepath.ToSlash(path), installMaxTraversalDepth)
	}
	return nil
}

func installRelativeTraversalDepth(path string) int {
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || clean == "/" {
		return 0
	}
	return len(strings.Split(strings.Trim(clean, "/"), "/"))
}

func digestArtifactEntry(h digestWriter, path, rel string, exactMetadata bool, budget *installArtifactBudget, depth int) error {
	if err := validateInstallTraversalDepth(depth, rel); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !supportedArtifactMode(info.Mode()) {
		return fmt.Errorf("unsupported artifact type %s in snapshot", info.Mode().Type())
	}
	if budget.entries >= installArtifactMaxEntries {
		return fmt.Errorf("artifact tree exceeds %d-entry limit", installArtifactMaxEntries)
	}
	budget.entries++
	kind := "file"
	if info.Mode()&os.ModeSymlink != 0 {
		kind = "symlink"
	} else if info.IsDir() {
		kind = "dir"
	}
	// Modification time is ambient extraction/install metadata, not semantic
	// artifact identity. Hash canonical path, type, mode, link target, and file
	// bytes only so identical releases installed at different times converge.
	if _, err := fmt.Fprintf(h, "%s\x00%s\x00%o\x00", filepath.ToSlash(rel), kind, uint32(info.Mode())); err != nil {
		return err
	}
	if exactMetadata && kind != "symlink" {
		if _, err := fmt.Fprintf(h, "mtime:%d\x00", info.ModTime().UnixNano()); err != nil {
			return err
		}
	}
	if kind == "symlink" {
		target, err := os.Readlink(path)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(h, "%s\x00", target)
		return err
	}
	if kind == "file" {
		if info.Size() < 0 || info.Size() > installArtifactMaxFileBytes {
			return fmt.Errorf("artifact file %s exceeds %d-byte limit", filepath.ToSlash(rel), installArtifactMaxFileBytes)
		}
		if info.Size() > installArtifactMaxTotalBytes-budget.bytes {
			return fmt.Errorf("artifact tree exceeds %d-byte limit", installArtifactMaxTotalBytes)
		}
		if _, err := fmt.Fprintf(h, "%d\x00", info.Size()); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		opened, statErr := f.Stat()
		if statErr != nil || !sameInstallArtifactInfo(info, opened) {
			return errors.Join(statErr, fmt.Errorf("artifact file %s changed while opening", filepath.ToSlash(rel)), closeInstallFile(f))
		}
		written, copyErr := io.Copy(h, io.LimitReader(f, info.Size()+1))
		heldAfter, heldErr := f.Stat()
		liveAfter, liveErr := os.Lstat(path)
		closeErr := closeInstallFile(f)
		if err := errors.Join(copyErr, heldErr, liveErr, closeErr); err != nil {
			return err
		}
		if written != info.Size() || !sameInstallArtifactInfo(info, heldAfter) || !sameInstallArtifactInfo(info, liveAfter) {
			return fmt.Errorf("artifact file %s changed while hashing", filepath.ToSlash(rel))
		}
		budget.bytes += written
		return nil
	}
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	opened, statErr := dir.Stat()
	entries, readErr := readInstallDirBounded(dir, installArtifactMaxEntries-budget.entries, "artifact directory")
	closeErr := closeInstallFile(dir)
	if err := errors.Join(statErr, readErr, closeErr); err != nil {
		return err
	}
	if !sameInstallArtifactInfo(info, opened) {
		return fmt.Errorf("artifact directory %s changed while opening", filepath.ToSlash(rel))
	}
	for _, entry := range entries {
		if err := digestArtifactEntry(h, filepath.Join(path, entry.Name()), filepath.Join(rel, entry.Name()), exactMetadata, budget, depth+1); err != nil {
			return err
		}
	}
	finalDir, err := os.Open(path)
	if err != nil {
		return err
	}
	finalOpened, finalStatErr := finalDir.Stat()
	finalEntries, finalReadErr := readInstallDirBounded(finalDir, len(entries), "artifact directory revalidation")
	finalCloseErr := closeInstallFile(finalDir)
	pathAfter, pathErr := os.Lstat(path)
	if err := errors.Join(finalStatErr, finalReadErr, finalCloseErr, pathErr); err != nil {
		return err
	}
	if !sameInstallArtifactInfo(info, finalOpened) || !sameInstallArtifactInfo(info, pathAfter) || len(entries) != len(finalEntries) {
		return fmt.Errorf("artifact directory %s changed while hashing", filepath.ToSlash(rel))
	}
	for i := range entries {
		if entries[i].Name() != finalEntries[i].Name() {
			return fmt.Errorf("artifact directory %s changed while hashing", filepath.ToSlash(rel))
		}
	}
	return nil
}

func sameInstallArtifactInfo(before, after os.FileInfo) bool {
	return before != nil && after != nil && os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size() && before.ModTime().Equal(after.ModTime()) && (installFileChangeID(before) == "" || installFileChangeID(after) == "" || installFileChangeID(before) == installFileChangeID(after))
}

func missingAncestorDirs(path string) ([]string, error) {
	var dirs []string
	for {
		info, err := os.Stat(path)
		if err == nil {
			if !info.IsDir() {
				return nil, fmt.Errorf("existing ancestor is not a directory: %s", path)
			}
			return dirs, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
		dirs = append(dirs, path)
		parent := filepath.Dir(path)
		if parent == path {
			return nil, fmt.Errorf("no existing directory ancestor for %s", path)
		}
		path = parent
	}
}

func activeInstallTransactionItem(path string) (*artifactTransaction, error) {
	stable := stableInstallMutationPath(path)
	activeInstallAnchors.Lock()
	defer activeInstallAnchors.Unlock()
	var found *artifactTransaction
	foundAnchor := ""
	for _, capabilities := range activeInstallAnchors.byPath {
		for _, capability := range capabilities {
			for _, item := range capability.items {
				if stableInstallMutationPath(item.Target) != stable {
					continue
				}
				if found == nil || len(capability.path) > len(foundAnchor) || capability.path == foundAnchor {
					found = capability.tx
					foundAnchor = capability.path
				}
			}
		}
	}
	if found == nil {
		return nil, fmt.Errorf("no active install transaction covers staged publication %s", path)
	}
	return found, nil
}

func mutateInstallStageItem(path string, mutate func(*installJournalItem) error) (installJournalItem, error) {
	tx, err := activeInstallTransactionItem(path)
	if err != nil {
		return installJournalItem{}, err
	}
	tx.journalMu.Lock()
	defer tx.journalMu.Unlock()
	latest, err := loadRetainedInstallJournal(tx.root, tx.journal)
	if err != nil {
		return installJournalItem{}, err
	}
	index := -1
	stable := stableInstallMutationPath(path)
	for i := range latest.Items {
		if stableInstallMutationPath(latest.Items[i].Target) == stable {
			index = i
			break
		}
	}
	if index < 0 {
		return installJournalItem{}, fmt.Errorf("active install transaction lost target %s", path)
	}
	if err := mutate(&latest.Items[index]); err != nil {
		return installJournalItem{}, err
	}
	if err := validateInstallStageJournalItem(latest.Items[index]); err != nil {
		return installJournalItem{}, err
	}
	if err := replaceJournalMetadata(tx.root, latest); err != nil {
		return installJournalItem{}, err
	}
	tx.journal = latest
	activeInstallAnchors.Lock()
	for _, capability := range tx.anchors {
		for i := range capability.items {
			if stableInstallMutationPath(capability.items[i].Target) == stable {
				capability.items[i] = latest.Items[index]
			}
		}
	}
	activeInstallAnchors.Unlock()
	return latest.Items[index], nil
}

func newInstallRestoreStageName() (string, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	return installRestoreStagePrefix + hex.EncodeToString(nonce[:]), nil
}

func newInstallRestorePrepareName(stageName string) (string, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	return stageName + installRestorePrepareTag + hex.EncodeToString(nonce[:]), nil
}

func validInstallRestorePrepareName(stageName, name string) bool {
	prefix := stageName + installRestorePrepareTag
	suffix := strings.TrimPrefix(name, prefix)
	if suffix == name || len(suffix) != 32 || suffix != strings.ToLower(suffix) {
		return false
	}
	_, err := hex.DecodeString(suffix)
	return err == nil
}

func installStageCreationPoint(point, target string) {
	if afterInstallStageCreationBoundary != nil {
		afterInstallStageCreationBoundary(point, target)
	}
}

func writeInstallStageCreationWitness(stage *os.Root, witness installStageCreationWitness) error {
	raw, err := marshalInstallStageCreationWitness(witness)
	if err != nil {
		return err
	}
	return publishInstallAuthorityRecord(stage, installRestoreIntentFile, installWitnessTempPrefix, "stage-witness", raw, 1024)
}

func marshalInstallStageCreationWitness(witness installStageCreationWitness) ([]byte, error) {
	raw, err := json.Marshal(witness)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func readInstallStageCreationWitness(stage *os.Root) (installStageCreationWitness, error) {
	const limit = 1024
	before, err := stage.Lstat(installRestoreIntentFile)
	if err != nil {
		return installStageCreationWitness{}, err
	}
	if !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 || before.Size() < 0 || before.Size() > limit {
		return installStageCreationWitness{}, fmt.Errorf("install stage creation witness is not a bounded regular file")
	}
	f, err := stage.Open(installRestoreIntentFile)
	if err != nil {
		return installStageCreationWitness{}, err
	}
	opened, statErr := f.Stat()
	raw, readErr := io.ReadAll(io.LimitReader(f, limit+1))
	after, afterErr := f.Stat()
	live, liveErr := stage.Lstat(installRestoreIntentFile)
	closeErr := closeInstallFile(f)
	if err := errors.Join(statErr, readErr, afterErr, liveErr, closeErr); err != nil {
		return installStageCreationWitness{}, err
	}
	if int64(len(raw)) > limit || !sameInstallArtifactInfo(before, opened) || !sameInstallArtifactInfo(opened, after) || !sameInstallArtifactInfo(after, live) {
		return installStageCreationWitness{}, fmt.Errorf("install stage creation witness changed while reading")
	}
	var witness installStageCreationWitness
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&witness); err != nil {
		return installStageCreationWitness{}, err
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = fmt.Errorf("trailing JSON value")
		}
		return installStageCreationWitness{}, err
	}
	return witness, nil
}

func openPreparedInstallStage(parent *os.Root, target, name, stageName string) (*os.Root, string, error) {
	info, err := parent.Lstat(name)
	if err != nil {
		return nil, "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return nil, "", fmt.Errorf("install stage creation envelope is not a private real directory")
	}
	path := filepath.Join(filepath.Dir(target), name)
	identity, err := installArtifactNativeIdentity(path, info)
	if err != nil {
		return nil, "", err
	}
	stage, err := parent.OpenRoot(name)
	if err != nil {
		return nil, "", err
	}
	opened, statErr := stage.Stat(".")
	if statErr != nil || !sameInstallArtifactInfo(info, opened) {
		return nil, "", errors.Join(statErr, fmt.Errorf("install stage creation envelope changed while opening"), stage.Close())
	}
	expected, err := marshalInstallStageCreationWitness(installStageCreationWitness{StageName: stageName, StageID: identity})
	if err != nil {
		return nil, "", errors.Join(err, stage.Close())
	}
	if err := recoverInstallAuthorityRecord(stage, installRestoreIntentFile, installWitnessTempPrefix, expected, 1024); err != nil {
		return nil, "", errors.Join(err, stage.Close())
	}
	witness, err := readInstallStageCreationWitness(stage)
	if err != nil || witness.StageName != stageName || witness.StageID != identity {
		return nil, "", errors.Join(err, fmt.Errorf("install stage creation witness does not match its native envelope"), stage.Close())
	}
	return stage, identity, nil
}

func installArtifactNativeIdentity(path string, info os.FileInfo) (string, error) {
	identity, err := stableInstallDirIdentity(path, info)
	if err != nil || identity == "" {
		return "", errors.Join(err, fmt.Errorf("install artifact %s lacks a stable native identity", path))
	}
	return identity, nil
}

func installStageAbsolutePath(target, stage string) string {
	return filepath.Join(filepath.Dir(target), stage)
}

func openInstallRestoreStage(parent *os.Root, target string, item installJournalItem) (*os.Root, error) {
	info, err := parent.Lstat(item.StageName)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("install restoration stage is not a real directory")
	}
	identity, err := installArtifactNativeIdentity(installStageAbsolutePath(target, item.StageName), info)
	if err != nil || identity != item.StageID {
		return nil, errors.Join(err, fmt.Errorf("install restoration stage changed identity; preserving it"))
	}
	root, err := parent.OpenRoot(item.StageName)
	if err != nil {
		return nil, err
	}
	opened, statErr := root.Stat(".")
	if statErr != nil || !sameInstallArtifactInfo(info, opened) {
		return nil, errors.Join(statErr, fmt.Errorf("install restoration stage changed while opening"), root.Close())
	}
	witness, witnessErr := readInstallStageCreationWitness(root)
	legacyWitnessless := os.IsNotExist(witnessErr) && item.StageID != ""
	if !legacyWitnessless && (witnessErr != nil || witness.StageName != item.StageName || witness.StageID != item.StageID) {
		return nil, errors.Join(witnessErr, fmt.Errorf("install restoration stage lost its exact creation witness"), root.Close())
	}
	return root, nil
}

func installStageObjectMatches(root *os.Root, target string, item installJournalItem) (bool, bool, error) {
	before, err := root.Lstat("object")
	if os.IsNotExist(err) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	path := filepath.Join(installStageAbsolutePath(target, item.StageName), "object")
	identity, err := installArtifactNativeIdentity(path, before)
	if err != nil {
		return false, true, err
	}
	digest, err := stableArtifactDigestRoot(root, "object")
	if err != nil {
		return false, true, err
	}
	after, err := root.Lstat("object")
	if err != nil || !sameInstallArtifactInfo(before, after) {
		return false, true, errors.Join(err, fmt.Errorf("private restoration object changed while being validated"))
	}
	afterIdentity, err := installArtifactNativeIdentity(path, after)
	if err != nil || identity != afterIdentity {
		return false, true, errors.Join(err, fmt.Errorf("private restoration object changed native identity while being validated"))
	}
	return afterIdentity == item.StageObject && digest == item.StageDigest, true, nil
}

func installPublishedStageMatches(parent *os.Root, target string, item installJournalItem) (bool, bool, error) {
	base := filepath.Base(target)
	before, err := parent.Lstat(base)
	if os.IsNotExist(err) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	identity, err := installArtifactNativeIdentity(target, before)
	if err != nil {
		return false, true, err
	}
	digest, err := stableArtifactDigestRoot(parent, base)
	if err != nil {
		return false, true, err
	}
	after, err := parent.Lstat(base)
	if err != nil || !sameInstallArtifactInfo(before, after) {
		return false, true, errors.Join(err, fmt.Errorf("published restoration changed while being validated"))
	}
	afterIdentity, err := installArtifactNativeIdentity(target, after)
	if err != nil || identity != afterIdentity {
		return false, true, errors.Join(err, fmt.Errorf("published restoration changed native identity while being validated"))
	}
	return afterIdentity == item.StageObject && digest == item.StageDigest, true, nil
}

func recoverInstallRestoreStageQuarantine(parent *os.Root, target string, item installJournalItem) (bool, error) {
	dir, err := parent.Open(".")
	if err != nil {
		return false, err
	}
	entries, readErr := readInstallDirBounded(dir, installArtifactMaxEntries, "install restoration quarantine inventory")
	closeErr := closeInstallFile(dir)
	if err := errors.Join(readErr, closeErr); err != nil {
		return false, err
	}
	var match string
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), installRestoreDeletePrefix) {
			continue
		}
		quarantine, err := fsatomic.ResumeQuarantine(parent, entry.Name(), "")
		if err != nil {
			return false, err
		}
		source := quarantine.Source()
		if err := quarantine.Close(); err != nil {
			return false, err
		}
		if source != item.StageName {
			continue
		}
		if match != "" {
			return false, fmt.Errorf("multiple restoration quarantines cover %s; preserving both", target)
		}
		match = entry.Name()
	}
	if match == "" {
		return false, nil
	}
	quarantine, err := fsatomic.ResumeQuarantine(parent, match, item.StageName)
	if err != nil {
		return false, err
	}
	info, err := quarantine.Root().Lstat(quarantine.Name())
	if os.IsNotExist(err) {
		if err := quarantine.FinishEmpty(); err != nil {
			return false, errors.Join(err, quarantine.Close())
		}
		return true, nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false, errors.Join(err, fmt.Errorf("restoration quarantine is invalid; preserving it"), quarantine.Close())
	}
	identity, err := installArtifactNativeIdentity(filepath.Join(filepath.Dir(target), match, quarantine.Name()), info)
	if err != nil || identity != item.StageID {
		return false, errors.Join(err, fmt.Errorf("restoration quarantine changed identity; preserving it"), quarantine.Close())
	}
	stage, err := quarantine.Root().OpenRoot(quarantine.Name())
	if err != nil {
		return false, errors.Join(err, quarantine.Close())
	}
	object, objectErr := stage.Lstat("object")
	if objectErr == nil {
		objectPath := filepath.Join(filepath.Dir(target), match, quarantine.Name(), "object")
		objectID, identityErr := installArtifactNativeIdentity(objectPath, object)
		digest, digestErr := stableArtifactDigestRoot(stage, "object")
		after, afterErr := stage.Lstat("object")
		if identityErr != nil || digestErr != nil || afterErr != nil || objectID != item.StageObject || digest != item.StageDigest || !sameInstallArtifactInfo(object, after) {
			return false, errors.Join(identityErr, digestErr, afterErr, fmt.Errorf("restoration quarantine object changed; preserving it"), stage.Close(), quarantine.Close())
		}
	} else if !os.IsNotExist(objectErr) {
		return false, errors.Join(objectErr, stage.Close(), quarantine.Close())
	}
	if err := stage.Close(); err != nil {
		return false, errors.Join(err, quarantine.Close())
	}
	if err := quarantine.RemoveAll(); err != nil {
		return false, errors.Join(err, quarantine.Close())
	}
	return true, nil
}

func retireInstallRestoreStage(parent *os.Root, target string, item installJournalItem, stage *os.Root) (retErr error) {
	stageClosed := false
	defer func() {
		if !stageClosed {
			retErr = errors.Join(retErr, stage.Close())
		}
	}()
	objectMatches, objectExists, err := installStageObjectMatches(stage, target, item)
	if err != nil || objectExists && !objectMatches {
		return errors.Join(err, fmt.Errorf("restoration object changed before stage retirement; preserving it"))
	}
	var objectBefore os.FileInfo
	if objectExists {
		objectBefore, err = stage.Lstat("object")
		if err != nil {
			return err
		}
	}
	if err := syncRootRelativeDir(stage, "."); err != nil {
		return err
	}
	before, err := parent.Lstat(item.StageName)
	if err != nil {
		return err
	}
	identity, err := installArtifactNativeIdentity(installStageAbsolutePath(target, item.StageName), before)
	if err != nil || identity != item.StageID {
		return errors.Join(err, fmt.Errorf("restoration stage changed before retirement; preserving it"))
	}
	if err := stage.Close(); err != nil {
		return err
	}
	stageClosed = true
	quarantine, err := fsatomic.Quarantine(parent, item.StageName, installRestoreDeletePrefix)
	if err != nil {
		return err
	}
	info, err := quarantine.Root().Lstat(quarantine.Name())
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.Join(err, fmt.Errorf("retired restoration stage is invalid; preserving it"), quarantine.Close())
	}
	if !os.SameFile(before, info) || before.Mode() != info.Mode() || before.Size() != info.Size() || !before.ModTime().Equal(info.ModTime()) {
		return errors.Join(fmt.Errorf("retired restoration stage changed identity; preserving it"), quarantine.Close())
	}
	retiredStage, err := quarantine.Root().OpenRoot(quarantine.Name())
	if err != nil {
		return errors.Join(err, quarantine.Close())
	}
	objectAfter, objectErr := retiredStage.Lstat("object")
	switch {
	case objectExists && objectErr == nil:
		digest, digestErr := stableArtifactDigestRoot(retiredStage, "object")
		finalInfo, finalErr := retiredStage.Lstat("object")
		if digestErr != nil || finalErr != nil || digest != item.StageDigest || !sameInstallArtifactInfo(objectBefore, objectAfter) || !sameInstallArtifactInfo(objectAfter, finalInfo) {
			return errors.Join(digestErr, finalErr, fmt.Errorf("retired restoration object changed identity or content; preserving it"), retiredStage.Close(), quarantine.Close())
		}
	case objectExists:
		return errors.Join(objectErr, fmt.Errorf("retired restoration object disappeared; preserving stage"), retiredStage.Close(), quarantine.Close())
	case objectErr == nil:
		return errors.Join(fmt.Errorf("restoration object appeared during stage retirement; preserving it"), retiredStage.Close(), quarantine.Close())
	case !os.IsNotExist(objectErr):
		return errors.Join(objectErr, retiredStage.Close(), quarantine.Close())
	}
	if err := retiredStage.Close(); err != nil {
		return errors.Join(err, quarantine.Close())
	}
	if err := quarantine.RemoveAll(); err != nil {
		return errors.Join(err, quarantine.Close())
	}
	return syncRootRelativeDir(parent, ".")
}

func prepareInstallRestoreStage(parent *os.Root, target string, item installJournalItem) (installJournalItem, error) {
	if item.StageID != "" {
		return item, nil
	}
	if _, err := parent.Lstat(item.StageName); err == nil {
		stage, identity, err := openPreparedInstallStage(parent, target, item.StageName, item.StageName)
		if err != nil {
			return item, fmt.Errorf("planned restoration stage appeared without its exact creation witness; preserving it: %w", err)
		}
		if err := stage.Close(); err != nil {
			return item, err
		}
		return persistInstallRestoreStageIdentity(target, item, identity)
	} else if !os.IsNotExist(err) {
		return item, err
	}
	prefix := item.StageName + installRestorePrepareTag
	dir, err := parent.Open(".")
	if err != nil {
		return item, err
	}
	entries, readErr := readInstallDirBounded(dir, installArtifactMaxEntries, "install stage preparation inventory")
	closeErr := closeInstallFile(dir)
	if err := errors.Join(readErr, closeErr); err != nil {
		return item, err
	}
	preparedName := ""
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), prefix) || !validInstallRestorePrepareName(item.StageName, entry.Name()) {
			continue
		}
		stage, _, err := openPreparedInstallStage(parent, target, entry.Name(), item.StageName)
		if err != nil {
			if retireErr := retireEmptyInstallPrepare(parent, entry.Name()); retireErr != nil {
				return item, errors.Join(fmt.Errorf("untrusted install stage preparation is not safely empty; preserving it: %w", err), retireErr)
			}
			continue
		}
		if err := stage.Close(); err != nil {
			return item, err
		}
		if preparedName != "" {
			return item, fmt.Errorf("multiple exact install stage preparation envelopes exist; preserving them")
		}
		preparedName = entry.Name()
	}
	if preparedName == "" {
		preparedName, err = newInstallRestorePrepareName(item.StageName)
		if err != nil {
			return item, err
		}
		installStageCreationPoint("before-mkdir", target)
		if err := parent.Mkdir(preparedName, 0o700); err != nil {
			return item, err
		}
		installStageCreationPoint("after-mkdir", target)
		if err := syncRootRelativeDir(parent, "."); err != nil {
			return item, err
		}
		installStageCreationPoint("after-parent-sync", target)
		info, err := parent.Lstat(preparedName)
		if err != nil {
			return item, err
		}
		identity, err := installArtifactNativeIdentity(filepath.Join(filepath.Dir(target), preparedName), info)
		if err != nil {
			return item, err
		}
		stage, err := parent.OpenRoot(preparedName)
		if err != nil {
			return item, err
		}
		if err := writeInstallStageCreationWitness(stage, installStageCreationWitness{StageName: item.StageName, StageID: identity}); err != nil {
			return item, errors.Join(err, stage.Close())
		}
		if err := stage.Close(); err != nil {
			return item, err
		}
		if err := syncRootRelativeDir(parent, "."); err != nil {
			return item, err
		}
	}
	if err := fsatomic.RenameNoReplace(parent, preparedName, item.StageName); err != nil {
		return item, fmt.Errorf("promote prepared install stage without replacement: %w", err)
	}
	if err := syncRootRelativeDir(parent, "."); err != nil {
		return item, err
	}
	stage, identity, err := openPreparedInstallStage(parent, target, item.StageName, item.StageName)
	if err != nil {
		return item, err
	}
	if err := stage.Close(); err != nil {
		return item, err
	}
	installStageCreationPoint("before-stageid-persist", target)
	item, err = persistInstallRestoreStageIdentity(target, item, identity)
	if err != nil {
		return item, err
	}
	installStageCreationPoint("after-stageid-persist", target)
	return item, nil
}

func retireEmptyInstallPrepare(parent *os.Root, name string) error {
	before, err := parent.Lstat(name)
	if err != nil {
		return err
	}
	if !before.IsDir() || before.Mode()&os.ModeSymlink != 0 || before.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("untrusted stage preparation is not a private real directory")
	}
	stage, err := parent.OpenRoot(name)
	if err != nil {
		return err
	}
	opened, statErr := stage.Stat(".")
	dir, openErr := stage.Open(".")
	if statErr != nil || openErr != nil || !sameInstallArtifactInfo(before, opened) {
		return errors.Join(statErr, openErr, fmt.Errorf("untrusted stage preparation changed while opening"), stage.Close())
	}
	_, readErr := readInstallDirBounded(dir, 0, "incomplete install stage preparation")
	closeDirErr := closeInstallFile(dir)
	if err := errors.Join(readErr, closeDirErr); err != nil {
		return errors.Join(err, stage.Close())
	}
	if err := stage.Close(); err != nil {
		return err
	}
	quarantine, err := fsatomic.Quarantine(parent, name, installPrepareDeletePrefix)
	if err != nil {
		return err
	}
	after, err := quarantine.Root().Lstat(quarantine.Name())
	if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return errors.Join(err, fmt.Errorf("incomplete install stage preparation changed during retirement; preserving it"), quarantine.Close())
	}
	retired, err := quarantine.Root().OpenRoot(quarantine.Name())
	if err != nil {
		return errors.Join(err, quarantine.Close())
	}
	retiredDir, openErr := retired.Open(".")
	if openErr != nil {
		return errors.Join(openErr, retired.Close(), quarantine.Close())
	}
	_, readErr = readInstallDirBounded(retiredDir, 0, "retired incomplete install stage preparation")
	closeDirErr = closeInstallFile(retiredDir)
	if err := errors.Join(readErr, closeDirErr, retired.Close()); err != nil {
		return errors.Join(err, quarantine.Close())
	}
	return quarantine.RemoveAll()
}

func persistInstallRestoreStageIdentity(target string, item installJournalItem, identity string) (installJournalItem, error) {
	return mutateInstallStageItem(target, func(current *installJournalItem) error {
		if current.StageName != item.StageName || current.StageID != "" {
			return fmt.Errorf("restoration stage authority changed while recording identity")
		}
		current.StageID = identity
		return nil
	})
}

func stageInstallEntryNoReplace(src, target, use string) error {
	digest, err := stableArtifactPostImageDigest(src)
	if err != nil {
		return err
	}
	item, err := mutateInstallStageItem(target, func(item *installJournalItem) error {
		if item.StageName != "" {
			if item.StageUse != use || item.StageDigest != digest {
				return fmt.Errorf("another staged publication already covers %s", target)
			}
			return nil
		}
		name, err := newInstallRestoreStageName()
		if err != nil {
			return err
		}
		item.StageName, item.StageDigest, item.StageUse = name, digest, use
		return nil
	})
	if err != nil {
		return err
	}
	parent, base, closeParent, err := installDeletionParent(target)
	if err != nil {
		return err
	}
	if closeParent {
		defer parent.Close() //nolint:errcheck // error paths preserve the staged authority
	}
	if cleaned, err := recoverInstallRestoreStageQuarantine(parent, target, item); err != nil {
		return err
	} else if cleaned {
		published, exists, matchErr := installPublishedStageMatches(parent, target, item)
		if matchErr != nil || !exists || !published {
			return errors.Join(matchErr, fmt.Errorf("retired restoration stage has no matching published target; preserving journal authority"))
		}
		_, clearErr := mutateInstallStageItem(target, func(current *installJournalItem) error {
			if use == installStagePublish {
				current.PostState, current.PostDigest = installPostPresent, digest
			}
			return clearInstallStageItem(current)
		})
		return clearErr
	}
	if item.StageID == "" {
		item, err = prepareInstallRestoreStage(parent, target, item)
		if err != nil {
			return err
		}
	}
	stage, err := openInstallRestoreStage(parent, target, item)
	if err != nil {
		published, exists, matchErr := installPublishedStageMatches(parent, target, item)
		if matchErr == nil && exists && published {
			_, clearErr := mutateInstallStageItem(target, clearInstallStageItem)
			return clearErr
		}
		return errors.Join(err, matchErr)
	}
	stageClosed := false
	defer func() {
		if !stageClosed {
			_ = stage.Close()
		}
	}()
	if item.StageObject != "" {
		matches, objectExists, matchErr := installStageObjectMatches(stage, target, item)
		if matchErr != nil {
			return matchErr
		}
		if !objectExists {
			published, targetExists, publishErr := installPublishedStageMatches(parent, target, item)
			if publishErr != nil || !targetExists || !published {
				return errors.Join(publishErr, fmt.Errorf("restoration object disappeared without its exact published target; preserving stage"))
			}
			if err := retireInstallRestoreStage(parent, target, item, stage); err != nil {
				return err
			}
			stageClosed = true
			_, err = mutateInstallStageItem(target, func(current *installJournalItem) error {
				if use == installStagePublish {
					current.PostState, current.PostDigest = installPostPresent, digest
				}
				return clearInstallStageItem(current)
			})
			return err
		}
		if !matches {
			return fmt.Errorf("private restoration object changed before resumed publication; preserving it")
		}
	}
	if item.StageObject == "" {
		if err := stage.RemoveAll("object"); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := copyEntryToRootBounded(src, stage, "object", &installArtifactBudget{}, 0); err != nil {
			return err
		}
		if err := syncRootRelativeDir(stage, "."); err != nil {
			return err
		}
		gotDigest, err := stableArtifactDigestRoot(stage, "object")
		sourceDigest, sourceErr := stableArtifactPostImageDigest(src)
		if err != nil || sourceErr != nil || gotDigest != item.StageDigest || sourceDigest != item.StageDigest {
			return errors.Join(err, sourceErr, fmt.Errorf("private restoration stage or source differs from its retained witness"))
		}
		info, err := stage.Lstat("object")
		if err != nil {
			return err
		}
		identity, err := installArtifactNativeIdentity(filepath.Join(installStageAbsolutePath(target, item.StageName), "object"), info)
		if err != nil {
			return err
		}
		item, err = mutateInstallStageItem(target, func(current *installJournalItem) error {
			if current.StageName != item.StageName || current.StageID != item.StageID || current.StageObject != "" {
				return fmt.Errorf("restoration object authority changed while recording identity")
			}
			current.StageObject = identity
			return nil
		})
		if err != nil {
			return err
		}
	}
	if err := afterInstallRestorePoint("ready", target); err != nil {
		return err
	}
	if use == installStagePublish {
		if err := durableRemoveAll(target); err != nil {
			return err
		}
		if err := afterInstallRestorePoint("removed", target); err != nil {
			return err
		}
	}
	if beforeInstallRestoreRename != nil {
		beforeInstallRestoreRename(target)
	}
	matches, exists, err := installStageObjectMatches(stage, target, item)
	if err != nil || !exists || !matches {
		return errors.Join(err, fmt.Errorf("private restoration object changed at publication boundary; preserving it"))
	}
	if err := renameInstallNoReplace(stage, "object", parent, base); err != nil {
		return fmt.Errorf("publish restored install artifact without replacement: %w", err)
	}
	if err := errors.Join(syncRootRelativeDir(parent, "."), syncRootRelativeDir(stage, ".")); err != nil {
		return err
	}
	if err := afterInstallRestorePoint("published", target); err != nil {
		return err
	}
	published, exists, err := installPublishedStageMatches(parent, target, item)
	if err != nil || !exists || !published {
		return errors.Join(err, fmt.Errorf("published restoration changed at the syscall boundary; preserving it"))
	}
	if err := retireInstallRestoreStage(parent, target, item, stage); err != nil {
		return err
	}
	stageClosed = true
	_, err = mutateInstallStageItem(target, func(current *installJournalItem) error {
		if use == installStagePublish {
			current.PostState, current.PostDigest = installPostPresent, digest
		}
		return clearInstallStageItem(current)
	})
	return err
}

func afterInstallRestorePoint(point, target string) error {
	if afterInstallRestoreBoundary == nil {
		return nil
	}
	return afterInstallRestoreBoundary(point, target)
}

func clearInstallStageItem(item *installJournalItem) error {
	item.StageName, item.StageID, item.StageObject, item.StageDigest, item.StageUse = "", "", "", "", ""
	return nil
}

func rollbackStagedInstallItem(root string, item installJournalItem) (bool, error) {
	if item.StageName == "" {
		return false, nil
	}
	backup := filepath.Join(root, filepath.FromSlash(item.Backup))
	if item.StageUse == installStageRestore {
		return true, stageInstallEntryNoReplace(backup, item.Target, installStageRestore)
	}
	parent, _, closeParent, err := installDeletionParent(item.Target)
	if err != nil {
		return false, err
	}
	if closeParent {
		defer parent.Close() //nolint:errcheck // error paths preserve authority
	}
	if item.StageID == "" {
		item, err = prepareInstallRestoreStage(parent, item.Target, item)
		if err != nil {
			return false, err
		}
	}
	published, exists, err := installPublishedStageMatches(parent, item.Target, item)
	if err != nil {
		return false, err
	}
	needsRestore := !exists && item.Existed
	if exists && published {
		if err := durableRemoveAll(item.Target); err != nil {
			return false, err
		}
		needsRestore = item.Existed
	} else if exists {
		matches, matchErr := installJournalPostImageMatches(item)
		if matchErr != nil || !matches {
			return false, errors.Join(matchErr, fmt.Errorf("staged publication target was concurrently replaced; preserving target and stage"))
		}
	}
	cleaned, err := recoverInstallRestoreStageQuarantine(parent, item.Target, item)
	if err != nil {
		return false, err
	}
	if !cleaned {
		stage, err := openInstallRestoreStage(parent, item.Target, item)
		if err != nil {
			return false, err
		}
		if err := retireInstallRestoreStage(parent, item.Target, item, stage); err != nil {
			return false, err
		}
	}
	if _, err := mutateInstallStageItem(item.Target, clearInstallStageItem); err != nil {
		return false, err
	}
	if needsRestore {
		backup := filepath.Join(root, filepath.FromSlash(item.Backup))
		if err := stageInstallEntryNoReplace(backup, item.Target, installStageRestore); err != nil {
			return false, err
		}
	}
	return true, nil
}

func copyEntryNoFollow(src, dst string) error {
	return copyEntryNoFollowBounded(src, dst, &installArtifactBudget{}, 0)
}

func copyEntryNoFollowBounded(src, dst string, budget *installArtifactBudget, depth int) error {
	if err := validateInstallTraversalDepth(depth, src); err != nil {
		return err
	}
	if err := declareInstallPresentPostImage(dst, src); err != nil {
		return err
	}
	if err := validateActiveInstallMutation(dst); err != nil {
		return err
	}
	capability, rel, confined, err := retainedCapabilityForMutation(dst)
	if err != nil {
		return err
	}
	if afterInstallMutationValidation != nil {
		afterInstallMutationValidation(dst)
	}
	if confined {
		return copyEntryToRootBounded(src, capability, rel, budget, depth)
	}
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if budget.entries >= installArtifactMaxEntries {
		return fmt.Errorf("install artifact copy exceeds %d-entry limit", installArtifactMaxEntries)
	}
	budget.entries++
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.Symlink(target, dst); err != nil {
			return err
		}
		return syncDir(filepath.Dir(dst))
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		dir, err := os.Open(src)
		if err != nil {
			return err
		}
		opened, statErr := dir.Stat()
		entries, readErr := readInstallDirBounded(dir, installArtifactMaxEntries-budget.entries, "install artifact copy")
		closeErr := closeInstallFile(dir)
		if err := errors.Join(statErr, readErr, closeErr); err != nil {
			return err
		}
		if !sameInstallArtifactInfo(info, opened) {
			return fmt.Errorf("install artifact directory %s changed while opening", src)
		}
		for _, entry := range entries {
			if err := copyEntryNoFollowBounded(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()), budget, depth+1); err != nil {
				return err
			}
		}
		if err := os.Chmod(dst, info.Mode().Perm()); err != nil {
			return err
		}
		if err := os.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
			return err
		}
		if err := syncDir(dst); err != nil {
			return err
		}
		return syncDir(filepath.Dir(dst))
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported install artifact type %s at %s", info.Mode().Type(), src)
	}
	if info.Size() < 0 || info.Size() > installArtifactMaxFileBytes || info.Size() > installArtifactMaxTotalBytes-budget.bytes {
		return fmt.Errorf("install artifact file %s exceeds copy bounds", src)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	opened, err := in.Stat()
	if err != nil || !sameInstallArtifactInfo(info, opened) {
		return errors.Join(err, fmt.Errorf("install artifact file %s changed while opening", src))
	}
	installArtifactCopyAfterOpen(src)
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(out, io.LimitReader(in, info.Size()+1))
	heldAfter, heldErr := in.Stat()
	liveAfter, liveErr := os.Lstat(src)
	if err := errors.Join(copyErr, heldErr, liveErr); err != nil {
		_ = out.Close()
		return err
	}
	if written != info.Size() || !sameInstallArtifactInfo(info, heldAfter) || !sameInstallArtifactInfo(info, liveAfter) {
		_ = out.Close()
		return fmt.Errorf("install artifact file %s changed while copying", src)
	}
	budget.bytes += written
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
		return err
	}
	return syncDir(filepath.Dir(dst))
}

func copyEntryToRoot(src string, root *os.Root, dst string) error {
	if _, err := stableArtifactPostImageDigest(src); err != nil {
		return err
	}
	return copyEntryToRootBounded(src, root, dst, &installArtifactBudget{}, 0)
}

func copyEntryToRootBounded(src string, root *os.Root, dst string, budget *installArtifactBudget, depth int) error {
	if err := validateInstallTraversalDepth(depth, src); err != nil {
		return err
	}
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if budget.entries >= installArtifactMaxEntries {
		return fmt.Errorf("install artifact copy exceeds %d-entry limit", installArtifactMaxEntries)
	}
	budget.entries++
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := root.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := root.Symlink(target, dst); err != nil {
			return err
		}
		return syncRootRelativeDir(root, filepath.Dir(dst))
	}
	if info.IsDir() {
		if err := root.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		dir, err := os.Open(src)
		if err != nil {
			return err
		}
		opened, statErr := dir.Stat()
		entries, readErr := readInstallDirBounded(dir, installArtifactMaxEntries-budget.entries, "install artifact copy")
		closeErr := closeInstallFile(dir)
		if err := errors.Join(statErr, readErr, closeErr); err != nil {
			return err
		}
		if !sameInstallArtifactInfo(info, opened) {
			return fmt.Errorf("install artifact directory %s changed while opening", src)
		}
		for _, entry := range entries {
			if err := copyEntryToRootBounded(filepath.Join(src, entry.Name()), root, filepath.Join(dst, entry.Name()), budget, depth+1); err != nil {
				return err
			}
		}
		if err := root.Chmod(dst, info.Mode().Perm()); err != nil {
			return err
		}
		if err := root.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
			return err
		}
		return syncRootRelativeDir(root, dst)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported install artifact type %s at %s", info.Mode().Type(), src)
	}
	if info.Size() < 0 || info.Size() > installArtifactMaxFileBytes || info.Size() > installArtifactMaxTotalBytes-budget.bytes {
		return fmt.Errorf("install artifact file %s exceeds copy bounds", src)
	}
	if err := root.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	opened, err := in.Stat()
	if err != nil || !sameInstallArtifactInfo(info, opened) {
		return errors.Join(err, fmt.Errorf("install artifact file %s changed while opening", src))
	}
	installArtifactCopyAfterOpen(src)
	out, err := root.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(out, io.LimitReader(in, info.Size()+1))
	heldAfter, heldErr := in.Stat()
	liveAfter, liveErr := os.Lstat(src)
	if err := errors.Join(copyErr, heldErr, liveErr); err != nil {
		_ = out.Close()
		return err
	}
	if written != info.Size() || !sameInstallArtifactInfo(info, heldAfter) || !sameInstallArtifactInfo(info, liveAfter) {
		_ = out.Close()
		return fmt.Errorf("install artifact file %s changed while copying", src)
	}
	budget.bytes += written
	if err := out.Chmod(info.Mode().Perm()); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := root.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
		return err
	}
	return syncRootRelativeDir(root, filepath.Dir(dst))
}

// copyEntryFromRoot snapshots a target through its retained parent capability.
// The source namespace can move while the transaction runs without redirecting
// reads outside the parent that was verified during planning.
func copyEntryFromRoot(root *os.Root, src, dst string) error {
	if _, err := stableArtifactDigestRoot(root, src); err != nil {
		return err
	}
	return copyEntryFromRootBounded(root, src, dst, &installArtifactBudget{}, 0)
}

func copyEntryFromRootBounded(root *os.Root, src, dst string, budget *installArtifactBudget, depth int) error {
	if err := validateInstallTraversalDepth(depth, src); err != nil {
		return err
	}
	info, err := root.Lstat(src)
	if err != nil {
		return err
	}
	if budget.entries >= installArtifactMaxEntries {
		return fmt.Errorf("install artifact copy exceeds %d-entry limit", installArtifactMaxEntries)
	}
	budget.entries++
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := root.Readlink(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.Symlink(target, dst); err != nil {
			return err
		}
		return syncDir(filepath.Dir(dst))
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		dir, err := root.Open(src)
		if err != nil {
			return err
		}
		opened, statErr := dir.Stat()
		entries, readErr := readInstallDirBounded(dir, installArtifactMaxEntries-budget.entries, "install artifact copy")
		closeErr := dir.Close()
		if err := errors.Join(statErr, readErr, closeErr); err != nil {
			return err
		}
		if !sameInstallArtifactInfo(info, opened) {
			return fmt.Errorf("install artifact directory %s changed while opening", src)
		}
		for _, entry := range entries {
			if err := copyEntryFromRootBounded(root, filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()), budget, depth+1); err != nil {
				return err
			}
		}
		if err := os.Chmod(dst, info.Mode().Perm()); err != nil {
			return err
		}
		if err := os.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
			return err
		}
		if err := syncDir(dst); err != nil {
			return err
		}
		return syncDir(filepath.Dir(dst))
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported install artifact type %s at %s", info.Mode().Type(), src)
	}
	if info.Size() < 0 || info.Size() > installArtifactMaxFileBytes || info.Size() > installArtifactMaxTotalBytes-budget.bytes {
		return fmt.Errorf("install artifact file %s exceeds copy bounds", src)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := root.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	opened, err := in.Stat()
	if err != nil || !sameInstallArtifactInfo(info, opened) {
		return errors.Join(err, fmt.Errorf("install artifact file %s changed while opening", src))
	}
	installArtifactCopyAfterOpen(src)
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(out, io.LimitReader(in, info.Size()+1))
	heldAfter, heldErr := in.Stat()
	liveAfter, liveErr := root.Lstat(src)
	if err := errors.Join(copyErr, heldErr, liveErr); err != nil {
		_ = out.Close()
		return err
	}
	if written != info.Size() || !sameInstallArtifactInfo(info, heldAfter) || !sameInstallArtifactInfo(info, liveAfter) {
		_ = out.Close()
		return fmt.Errorf("install artifact file %s changed while copying", src)
	}
	budget.bytes += written
	if err := out.Chmod(info.Mode().Perm()); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
		return err
	}
	return syncDir(filepath.Dir(dst))
}

func syncTree(root string) error {
	return walkInstallTreeBounded(root, installArtifactMaxEntries, func(path string, info os.FileInfo) error {
		if info.IsDir() {
			return syncDir(path)
		}
		return nil
	})
}

func walkInstallTreeBounded(root string, limit int, visit func(string, os.FileInfo) error) error {
	if limit <= 0 {
		return fmt.Errorf("install tree entry limit must be positive")
	}
	type installTreeMember struct {
		path string
		info os.FileInfo
	}
	seen := 0
	inventory := make([]installTreeMember, 0, min(limit, installSourceReadDirPage))
	var walk func(string, int) error
	walk = func(path string, depth int) error {
		if err := validateInstallTraversalDepth(depth, path); err != nil {
			return err
		}
		before, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if seen >= limit {
			return fmt.Errorf("install tree exceeds %d-entry limit", limit)
		}
		seen++
		inventory = append(inventory, installTreeMember{path: path, info: before})
		if !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		dir, err := os.Open(path)
		if err != nil {
			return err
		}
		opened, statErr := dir.Stat()
		entries, readErr := readInstallDirBounded(dir, limit-seen, "install tree")
		closeErr := closeInstallFile(dir)
		if err := errors.Join(statErr, readErr, closeErr); err != nil {
			return err
		}
		if !sameInstallArtifactInfo(before, opened) {
			return fmt.Errorf("install directory %s changed while being enumerated", path)
		}
		for _, entry := range entries {
			if err := walk(filepath.Join(path, entry.Name()), depth+1); err != nil {
				return err
			}
		}
		after, err := os.Lstat(path)
		if err != nil || !sameInstallArtifactInfo(before, after) {
			return errors.Join(err, fmt.Errorf("install directory %s changed while being traversed", path))
		}
		final, err := os.Open(path)
		if err != nil {
			return err
		}
		finalInfo, statErr := final.Stat()
		finalEntries, readErr := readInstallDirBounded(final, len(entries), "install tree revalidation")
		closeErr = closeInstallFile(final)
		if err := errors.Join(statErr, readErr, closeErr); err != nil {
			return err
		}
		if !sameInstallArtifactInfo(before, finalInfo) || len(entries) != len(finalEntries) {
			return fmt.Errorf("install directory %s inventory changed while being traversed", path)
		}
		for index := range entries {
			if entries[index].Name() != finalEntries[index].Name() {
				return fmt.Errorf("install directory %s inventory changed while being traversed", path)
			}
		}
		return nil
	}
	if err := walk(root, 0); err != nil {
		return err
	}
	// Traversal bounds and the complete source inventory are established before
	// the visitor can mutate a destination or issue durability syscalls.
	for _, member := range inventory {
		current, err := os.Lstat(member.path)
		if err != nil || !sameInstallArtifactInfo(member.info, current) {
			return errors.Join(err, fmt.Errorf("install tree member %s changed before visitation", member.path))
		}
		if err := visit(member.path, current); err != nil {
			return err
		}
	}
	return nil
}

func syncDir(path string) error {
	return syncDirectoryPath(path)
}

func durableRename(oldPath, newPath string) (retErr error) {
	if err := declareInstallPresentPostImage(newPath, oldPath); err != nil {
		return err
	}
	if err := validateActiveInstallMutation(newPath); err != nil {
		return err
	}
	newRoot, newBase, closeNew, err := installDeletionParent(newPath)
	if err != nil {
		return err
	}
	if closeNew {
		defer func() { retErr = errors.Join(retErr, newRoot.Close()) }()
	}
	oldRoot, oldBase, closeOld, err := openInstallPathParent(oldPath)
	if err != nil {
		return err
	}
	if closeOld {
		defer func() { retErr = errors.Join(retErr, oldRoot.Close()) }()
	}
	if info, err := newRoot.Lstat(newBase); err == nil {
		return fmt.Errorf("install rename destination %s already exists (%s)", newPath, info.Mode())
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := renameInstallNoReplace(oldRoot, oldBase, newRoot, newBase); err != nil {
		return err
	}
	if err := syncRootRelativeDir(newRoot, "."); err != nil {
		return err
	}
	return syncRootRelativeDir(oldRoot, ".")
}

func openInstallPathParent(path string) (*os.Root, string, bool, error) {
	clean, err := filepath.Abs(path)
	if err != nil {
		return nil, "", false, err
	}
	parentPath := filepath.Dir(clean)
	before, err := os.Lstat(parentPath)
	if err != nil {
		return nil, "", false, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, "", false, fmt.Errorf("install rename parent %s must be a real directory", parentPath)
	}
	root, err := os.OpenRoot(parentPath)
	if err != nil {
		return nil, "", false, err
	}
	inside, err := root.Stat(".")
	if err != nil || !sameInstallArtifactInfo(before, inside) {
		return nil, "", false, errors.Join(err, fmt.Errorf("install rename parent %s changed while opening", parentPath), root.Close())
	}
	return root, filepath.Base(clean), true, nil
}

func durableRemoveAll(path string) error {
	return durableRemoveInstallPath(path, false)
}

func durableRemove(path string) error {
	return durableRemoveInstallPath(path, true)
}

const installDeletionQuarantinePrefix = ".machinery-install-delete-"

type installRemovalWitness struct {
	info   os.FileInfo
	digest string
}

func durableRemoveInstallPath(path string, emptyOnly bool) (retErr error) {
	if err := declareInstallPostImage(path, installPostAbsent, ""); err != nil {
		return err
	}
	if err := validateActiveInstallMutation(path); err != nil {
		return err
	}
	parent, base, closeParent, err := installDeletionParent(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if closeParent {
		defer func() { retErr = errors.Join(retErr, parent.Close()) }()
	}
	if err := recoverInstallDeletionQuarantines(parent, base); err != nil {
		return err
	}
	witness, exists, err := captureInstallRemovalWitness(parent, base)
	if err != nil || !exists {
		return err
	}
	if emptyOnly && witness.info.IsDir() {
		empty, err := installRootDirectoryEmpty(parent, base)
		if err != nil {
			return err
		}
		if !empty {
			return nil
		}
	}
	if afterInstallMutationValidation != nil {
		afterInstallMutationValidation(path)
	}
	if err := validateActiveInstallMutation(path); err != nil {
		return err
	}
	if err := revalidateInstallRemovalWitness(parent, base, witness); err != nil {
		return fmt.Errorf("install deletion target %s changed at the mutation boundary; preserving it: %w", path, err)
	}
	quarantined, err := fsatomic.Quarantine(parent, base, installDeletionQuarantinePrefix)
	if err != nil {
		return err
	}
	if err := validateInstallDeletionQuarantine(quarantined, true); err != nil {
		return errors.Join(err, quarantined.Close())
	}
	privateWitness, exists, err := captureInstallRemovalWitness(quarantined.Root(), quarantined.Name())
	if err != nil || !exists || !sameInstallRemovalWitnessAfterMove(witness, privateWitness) {
		return errors.Join(err, fmt.Errorf("install deletion target %s changed while entering private authority; preserving it", path), quarantined.Close())
	}
	if afterInstallPrivateDeletionValidation != nil {
		afterInstallPrivateDeletionValidation(quarantined.Root(), quarantined.Name())
	}
	if err := validateInstallDeletionQuarantine(quarantined, true); err != nil {
		return errors.Join(err, quarantined.Close())
	}
	if err := revalidateInstallRemovalWitness(quarantined.Root(), quarantined.Name(), privateWitness); err != nil {
		return errors.Join(fmt.Errorf("private install deletion authority changed; preserving it: %w", err), quarantined.Close())
	}
	if privateWitness.info.IsDir() {
		return quarantined.RemoveAll()
	}
	return quarantined.Remove()
}

func installDeletionParent(path string) (*os.Root, string, bool, error) {
	capability, rel, confined, err := retainedCapabilityForMutation(path)
	if err != nil {
		return nil, "", false, err
	}
	if confined {
		parentRel := filepath.Dir(rel)
		if parentRel == "." {
			return capability, filepath.Base(rel), false, nil
		}
		parent, err := capability.OpenRoot(parentRel)
		return parent, filepath.Base(rel), true, err
	}
	clean, err := filepath.Abs(path)
	if err != nil {
		return nil, "", false, err
	}
	parentPath := filepath.Dir(clean)
	before, err := os.Lstat(parentPath)
	if err != nil {
		return nil, "", false, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, "", false, fmt.Errorf("install deletion parent %s must be a real directory", parentPath)
	}
	root, err := os.OpenRoot(parentPath)
	if err != nil {
		return nil, "", false, err
	}
	inside, err := root.Stat(".")
	if err != nil || !sameInstallArtifactInfo(before, inside) {
		return nil, "", false, errors.Join(err, fmt.Errorf("install deletion parent %s changed while opening", parentPath), root.Close())
	}
	return root, filepath.Base(clean), true, nil
}

func captureInstallRemovalWitness(root *os.Root, name string) (installRemovalWitness, bool, error) {
	before, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return installRemovalWitness{}, false, nil
	}
	if err != nil {
		return installRemovalWitness{}, false, err
	}
	if !supportedArtifactMode(before.Mode()) {
		return installRemovalWitness{}, false, fmt.Errorf("install deletion target %s has unsupported type %s", name, before.Mode().Type())
	}
	digest, err := stableArtifactDigestRoot(root, name)
	if err != nil {
		return installRemovalWitness{}, false, err
	}
	after, err := root.Lstat(name)
	if err != nil || !sameInstallArtifactInfo(before, after) {
		return installRemovalWitness{}, false, errors.Join(err, fmt.Errorf("install deletion target %s changed while being witnessed", name))
	}
	return installRemovalWitness{info: after, digest: digest}, true, nil
}

func revalidateInstallRemovalWitness(root *os.Root, name string, want installRemovalWitness) error {
	got, exists, err := captureInstallRemovalWitness(root, name)
	if err != nil {
		return err
	}
	if !exists || got.digest != want.digest || !sameInstallArtifactInfo(want.info, got.info) {
		return fmt.Errorf("install deletion target no longer matches its exact identity and content")
	}
	return nil
}

func sameInstallRemovalWitnessAfterMove(before, after installRemovalWitness) bool {
	return before.info != nil && after.info != nil && os.SameFile(before.info, after.info) &&
		before.info.Mode() == after.info.Mode() && before.info.Size() == after.info.Size() &&
		before.info.ModTime().Equal(after.info.ModTime()) && before.digest == after.digest
}

func stableArtifactDigestRoot(root *os.Root, path string) (string, error) {
	hash := sha256.New()
	budget := &installArtifactBudget{}
	if err := digestArtifactRootEntry(hash, root, path, ".", true, budget, 0); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

func digestArtifactRootEntry(hash digestWriter, root *os.Root, path, rel string, exactMetadata bool, budget *installArtifactBudget, depth int) error {
	if err := validateInstallTraversalDepth(depth, rel); err != nil {
		return err
	}
	info, err := root.Lstat(path)
	if err != nil {
		return err
	}
	if !supportedArtifactMode(info.Mode()) || budget.entries >= installArtifactMaxEntries {
		return fmt.Errorf("rooted artifact %s exceeds inventory or type bounds", filepath.ToSlash(rel))
	}
	budget.entries++
	kind := "file"
	if info.Mode()&os.ModeSymlink != 0 {
		kind = "symlink"
	} else if info.IsDir() {
		kind = "dir"
	}
	if _, err := fmt.Fprintf(hash, "%s\x00%s\x00%o\x00", filepath.ToSlash(rel), kind, uint32(info.Mode())); err != nil {
		return err
	}
	if exactMetadata && kind != "symlink" {
		if _, err := fmt.Fprintf(hash, "mtime:%d\x00", info.ModTime().UnixNano()); err != nil {
			return err
		}
	}
	if kind == "symlink" {
		target, err := root.Readlink(path)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(hash, "%s\x00", target)
		return err
	}
	if kind == "file" {
		if info.Size() < 0 || info.Size() > installArtifactMaxFileBytes || info.Size() > installArtifactMaxTotalBytes-budget.bytes {
			return fmt.Errorf("rooted artifact file %s exceeds byte bounds", filepath.ToSlash(rel))
		}
		if _, err := fmt.Fprintf(hash, "%d\x00", info.Size()); err != nil {
			return err
		}
		file, err := root.Open(path)
		if err != nil {
			return err
		}
		opened, statErr := file.Stat()
		written, copyErr := io.Copy(hash, io.LimitReader(file, info.Size()+1))
		heldAfter, heldErr := file.Stat()
		pathAfter, pathErr := root.Lstat(path)
		closeErr := closeInstallFile(file)
		if err := errors.Join(statErr, copyErr, heldErr, pathErr, closeErr); err != nil {
			return err
		}
		if written != info.Size() || !sameInstallArtifactInfo(info, opened) || !sameInstallArtifactInfo(info, heldAfter) || !sameInstallArtifactInfo(info, pathAfter) {
			return fmt.Errorf("rooted artifact file %s changed while hashing", filepath.ToSlash(rel))
		}
		budget.bytes += written
		return nil
	}
	dir, err := root.Open(path)
	if err != nil {
		return err
	}
	opened, statErr := dir.Stat()
	entries, readErr := readInstallDirBounded(dir, installArtifactMaxEntries-budget.entries, "rooted artifact directory")
	closeErr := closeInstallFile(dir)
	if err := errors.Join(statErr, readErr, closeErr); err != nil {
		return err
	}
	if !sameInstallArtifactInfo(info, opened) {
		return fmt.Errorf("rooted artifact directory %s changed while opening", filepath.ToSlash(rel))
	}
	for _, entry := range entries {
		if err := digestArtifactRootEntry(hash, root, filepath.Join(path, entry.Name()), filepath.Join(rel, entry.Name()), exactMetadata, budget, depth+1); err != nil {
			return err
		}
	}
	final, err := root.Open(path)
	if err != nil {
		return err
	}
	finalInfo, statErr := final.Stat()
	finalEntries, readErr := readInstallDirBounded(final, len(entries), "rooted artifact directory revalidation")
	closeErr = closeInstallFile(final)
	pathAfter, pathErr := root.Lstat(path)
	if err := errors.Join(statErr, readErr, closeErr, pathErr); err != nil {
		return err
	}
	if !sameInstallArtifactInfo(info, finalInfo) || !sameInstallArtifactInfo(info, pathAfter) || len(entries) != len(finalEntries) {
		return fmt.Errorf("rooted artifact directory %s changed while hashing", filepath.ToSlash(rel))
	}
	for index := range entries {
		if entries[index].Name() != finalEntries[index].Name() {
			return fmt.Errorf("rooted artifact directory %s inventory changed while hashing", filepath.ToSlash(rel))
		}
	}
	return nil
}

func recoverInstallDeletionQuarantines(parent *os.Root, source string) error {
	dir, err := parent.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := readInstallDirBounded(dir, installArtifactMaxEntries, "install deletion parent")
	closeErr := closeInstallFile(dir)
	if err := errors.Join(readErr, closeErr); err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, installDeletionQuarantinePrefix) {
			continue
		}
		quarantined, err := fsatomic.ResumeQuarantine(parent, name, "")
		if err != nil {
			return err
		}
		if quarantined.Source() != source {
			if err := quarantined.Close(); err != nil {
				return err
			}
			continue
		}
		if err := recoverInstallDeletionQuarantine(quarantined); err != nil {
			return err
		}
	}
	return nil
}

func recoverInstallDeletionQuarantine(quarantined *fsatomic.Quarantined) error {
	if _, err := quarantined.Root().Lstat(quarantined.Name()); errors.Is(err, os.ErrNotExist) {
		if err := validateInstallDeletionQuarantine(quarantined, false); err != nil {
			return errors.Join(err, quarantined.Close())
		}
		return quarantined.FinishEmpty()
	} else if err != nil {
		return errors.Join(err, quarantined.Close())
	}
	if err := validateInstallDeletionQuarantine(quarantined, true); err != nil {
		return errors.Join(err, quarantined.Close())
	}
	witness, exists, err := captureInstallRemovalWitness(quarantined.Root(), quarantined.Name())
	if err != nil || !exists {
		return errors.Join(err, fmt.Errorf("install deletion quarantine has no stable object"), quarantined.Close())
	}
	if afterInstallPrivateDeletionValidation != nil {
		afterInstallPrivateDeletionValidation(quarantined.Root(), quarantined.Name())
	}
	if err := validateInstallDeletionQuarantine(quarantined, true); err != nil {
		return errors.Join(err, quarantined.Close())
	}
	if err := revalidateInstallRemovalWitness(quarantined.Root(), quarantined.Name(), witness); err != nil {
		return errors.Join(fmt.Errorf("install deletion quarantine changed; preserving it: %w", err), quarantined.Close())
	}
	if witness.info.IsDir() {
		return quarantined.RemoveAll()
	}
	return quarantined.Remove()
}

func validateInstallDeletionQuarantine(quarantined *fsatomic.Quarantined, objectExpected bool) error {
	dir, err := quarantined.Root().Open(".")
	if err != nil {
		return err
	}
	var want []string
	if objectExpected {
		want = []string{"object"}
	}
	entries, readErr := readInstallDirBounded(dir, len(want), "install deletion quarantine")
	closeErr := closeInstallFile(dir)
	if err := errors.Join(readErr, closeErr); err != nil {
		return err
	}
	if len(entries) != len(want) {
		return fmt.Errorf("install deletion quarantine has unexpected inventory")
	}
	for index := range want {
		if entries[index].Name() != want[index] {
			return fmt.Errorf("install deletion quarantine has unexpected inventory")
		}
	}
	return nil
}

func installRootDirectoryEmpty(root *os.Root, name string) (bool, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return false, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return false, nil
	}
	dir, err := root.Open(name)
	if err != nil {
		return false, err
	}
	opened, statErr := dir.Stat()
	entries, readErr := readInstallDirBounded(dir, 1, "install empty directory")
	closeErr := closeInstallFile(dir)
	after, pathErr := root.Lstat(name)
	if err := errors.Join(statErr, readErr, closeErr, pathErr); err != nil {
		return false, err
	}
	if !sameInstallArtifactInfo(before, opened) || !sameInstallArtifactInfo(before, after) {
		return false, fmt.Errorf("install directory changed while checking emptiness")
	}
	return len(entries) == 0, nil
}

func retainedCapabilityForMutation(path string) (*os.Root, string, bool, error) {
	root, tombstone, err := installJournalPaths()
	if err != nil {
		return nil, "", false, err
	}
	clean, err := filepath.Abs(path)
	if err != nil {
		return nil, "", false, err
	}
	if capability, rel, ok := retainedActiveCapability(clean); ok {
		return capability, rel, true, nil
	}
	resolved, err := installArtifactAccessPath(clean)
	if err != nil {
		return nil, "", false, err
	}
	rootResolved, _ := installArtifactAccessPath(root)
	tombstoneResolved, _ := installArtifactAccessPath(tombstone)
	if pathAtOrBelow(rootResolved, resolved) || pathAtOrBelow(tombstoneResolved, resolved) {
		return nil, "", false, nil
	}
	journal, phase, err := loadInstallJournal(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", false, nil
		}
		return nil, "", false, err
	}
	if phase != "prepared" && phase != "committed" {
		return nil, "", false, nil
	}
	folded := foldedInstallPath(resolved)
	for _, item := range journal.Items {
		target := foldedInstallPath(item.Target)
		matches := pathAtOrBelow(target, folded) || pathAtOrBelow(folded, target)
		if !matches {
			for _, dir := range item.CreatedDirs {
				matches = foldedInstallPath(dir) == folded
				if matches {
					break
				}
			}
		}
		if !matches {
			continue
		}
		activeInstallAnchors.Lock()
		var found *installAnchorCapability
		for _, capability := range activeInstallAnchors.byPath[item.Anchor] {
			if capability.id == item.AnchorID {
				found = capability
				break
			}
		}
		activeInstallAnchors.Unlock()
		if found == nil {
			return nil, "", false, fmt.Errorf("no retained parent capability for install mutation %s", path)
		}
		rel, err := filepath.Rel(item.Anchor, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			continue
		}
		return found.root, rel, true, nil
	}
	return nil, "", false, nil
}

func retainedActiveCapability(path string) (*os.Root, string, bool) {
	stable := stableInstallMutationPath(path)
	activeInstallAnchors.Lock()
	defer activeInstallAnchors.Unlock()
	keys := make([]string, 0, len(activeInstallAnchors.byPath))
	for key := range activeInstallAnchors.byPath {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left, right := stableInstallMutationPath(keys[i]), stableInstallMutationPath(keys[j])
		if left != right {
			return left < right
		}
		return keys[i] < keys[j]
	})
	for _, key := range keys {
		capabilities := append([]*installAnchorCapability(nil), activeInstallAnchors.byPath[key]...)
		sort.SliceStable(capabilities, func(i, j int) bool {
			left, right := stableInstallMutationPath(capabilities[i].path), stableInstallMutationPath(capabilities[j].path)
			if left != right {
				return left < right
			}
			return capabilities[i].id < capabilities[j].id
		})
		for _, capability := range capabilities {
			anchor := stableInstallMutationPath(capability.path)
			for _, item := range capability.items {
				target := stableInstallMutationPath(item.Target)
				if !pathAtOrBelow(target, stable) && !pathAtOrBelow(stable, target) {
					matchedCreated := false
					for _, dir := range item.CreatedDirs {
						if stableInstallMutationPath(dir) == stable {
							matchedCreated = true
							break
						}
					}
					if !matchedCreated {
						continue
					}
				}
				rel, err := filepath.Rel(anchor, stable)
				if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
					return capability.root, rel, true
				}
			}
		}
	}
	return nil, "", false
}

func declareInstallPostImage(path, state, digest string) error {
	clean, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	stable := stableInstallMutationPath(clean)
	activeInstallAnchors.Lock()
	var tx *artifactTransaction
	for _, capabilities := range activeInstallAnchors.byPath {
		for _, capability := range capabilities {
			for _, item := range capability.items {
				if stableInstallMutationPath(item.Target) == stable {
					tx = capability.tx
					break
				}
			}
			if tx != nil {
				break
			}
		}
		if tx != nil {
			break
		}
	}
	activeInstallAnchors.Unlock()
	if tx == nil {
		return nil
	}
	if state != installPostAbsent && state != installPostPresent && state != installPostUnknown {
		return fmt.Errorf("invalid install post-image state %q", state)
	}
	if state == installPostPresent && !validArtifactDigest(digest) {
		return fmt.Errorf("invalid install post-image digest for %s", clean)
	}
	if state != installPostPresent && digest != "" {
		return fmt.Errorf("install post-image state %s cannot carry a digest", state)
	}
	tx.journalMu.Lock()
	defer tx.journalMu.Unlock()
	if tx.rollingBack {
		return nil
	}
	latest, err := loadRetainedInstallJournal(tx.root, tx.journal)
	if err != nil {
		return fmt.Errorf("refresh retained transaction before recording post-image for %s: %w", clean, err)
	}
	tx.journal = latest
	found := false
	for i := range tx.journal.Items {
		item := &tx.journal.Items[i]
		if stableInstallMutationPath(item.Target) != stable {
			continue
		}
		item.PostState = state
		item.PostDigest = digest
		found = true
	}
	if !found {
		return nil
	}
	if err := replaceJournalMetadata(tx.root, tx.journal); err != nil {
		return fmt.Errorf("persist transaction post-image for %s: %w", clean, err)
	}
	for _, capability := range tx.anchors {
		for i := range capability.items {
			if stableInstallMutationPath(capability.items[i].Target) == stable {
				capability.items[i].PostState = state
				capability.items[i].PostDigest = digest
			}
		}
	}
	return nil
}

func loadRetainedInstallJournal(root string, expected installJournal) (installJournal, error) {
	raw, err := readPrivateRegularFile(filepath.Join(root, installJournalMetadata), installJournalMaxBytes)
	if err != nil {
		return installJournal{}, err
	}
	var journal installJournal
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&journal); err != nil {
		return installJournal{}, err
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = fmt.Errorf("trailing JSON value")
		}
		return installJournal{}, err
	}
	if journal.Schema != installJournalSchema || journal.Phase != "snapshotting" || len(journal.Items) != len(expected.Items) {
		return installJournal{}, fmt.Errorf("retained install transaction metadata changed shape")
	}
	prepared, err := validJournalMarker(root, installJournalPrepared)
	if err != nil || !prepared {
		return installJournal{}, errors.Join(fmt.Errorf("retained install transaction is not prepared"), err)
	}
	for i, item := range journal.Items {
		want := expected.Items[i]
		if item.Target != want.Target || item.Backup != want.Backup || item.Existed != want.Existed || item.Anchor != want.Anchor ||
			item.AnchorID != want.AnchorID || item.Digest != want.Digest || strings.Join(item.CreatedDirs, "\x00") != strings.Join(want.CreatedDirs, "\x00") ||
			item.StageName != want.StageName || item.StageID != want.StageID || item.StageObject != want.StageObject || item.StageDigest != want.StageDigest || item.StageUse != want.StageUse {
			return installJournal{}, fmt.Errorf("retained install transaction immutable item %d changed", i)
		}
		switch item.PostState {
		case installPostAbsent, installPostUnknown:
			if item.PostDigest != "" {
				return installJournal{}, fmt.Errorf("retained install transaction item %d has an invalid post-image", i)
			}
		case installPostPresent:
			if !validArtifactDigest(item.PostDigest) {
				return installJournal{}, fmt.Errorf("retained install transaction item %d has an invalid post-image", i)
			}
		default:
			return installJournal{}, fmt.Errorf("retained install transaction item %d has an invalid post-image state", i)
		}
	}
	return journal, nil
}

func declareInstallPresentPostImage(path, source string) error {
	digest, err := stableArtifactPostImageDigest(source)
	if err != nil {
		return fmt.Errorf("digest transaction post-image source %s: %w", source, err)
	}
	return declareInstallPostImage(path, installPostPresent, digest)
}

func stableInstallMutationPath(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "darwin" {
		for _, alias := range []string{"/var", "/tmp", "/etc"} {
			if path == alias || strings.HasPrefix(path, alias+string(os.PathSeparator)) {
				path = "/private" + path
				break
			}
		}
	}
	return foldedInstallPath(path)
}

func syncRootRelativeDir(root *os.Root, rel string) error {
	if rel == "" {
		rel = "."
	}
	f, err := root.Open(rel)
	if err != nil {
		return err
	}
	return errors.Join(syncRootDirectoryFile(f, rel), closeInstallFile(f))
}

func syncRootRelativeDirChain(root *os.Root, rel string) error {
	rel = filepath.Clean(rel)
	for {
		if err := syncRootRelativeDir(root, rel); err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.Dir(rel)
	}
}

func durableMkdirAll(path string) error {
	return durableMkdirAllMode(path, 0o755)
}

func durableMkdirAllPrivate(path string) error {
	if err := durableMkdirAllMode(path, 0o700); err != nil {
		return err
	}
	return durableChmodDirectory(path, 0o700)
}

func durableChmodDirectory(path string, perm os.FileMode) error {
	if err := validateActiveInstallMutation(path); err != nil {
		return err
	}
	capability, rel, confined, err := retainedCapabilityForMutation(path)
	if err != nil {
		return err
	}
	if afterInstallMutationValidation != nil {
		afterInstallMutationValidation(path)
	}
	if confined {
		if err := capability.Chmod(rel, perm); err != nil {
			return err
		}
		return syncRootRelativeDir(capability, rel)
	}
	if err := os.Chmod(path, perm); err != nil {
		return err
	}
	return syncDir(path)
}

func durableMkdirAllMode(path string, perm os.FileMode) error {
	if err := validateActiveInstallMutation(path); err != nil {
		return err
	}
	capability, rel, confined, err := retainedCapabilityForMutation(path)
	if err != nil {
		return err
	}
	if afterInstallMutationValidation != nil {
		afterInstallMutationValidation(path)
	}
	if confined {
		if err := capability.MkdirAll(rel, perm); err != nil {
			return err
		}
		return syncRootRelativeDirChain(capability, rel)
	}
	if err := os.MkdirAll(path, perm); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

func durableRemoveEmptyDir(path string) error {
	return durableRemoveInstallPath(path, true)
}

func validateActiveInstallMutation(path string) error {
	root, tombstone, err := installJournalPaths()
	if err != nil {
		return err
	}
	clean, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	clean = filepath.Clean(clean)
	if _, _, ok := retainedActiveCapability(clean); ok {
		return nil
	}
	resolvedClean, resolveErr := installArtifactResolvedPath(clean)
	resolvedRoot, rootResolveErr := installArtifactResolvedPath(root)
	resolvedTombstone, tombstoneResolveErr := installArtifactResolvedPath(tombstone)
	activation := filepath.Join(filepath.Dir(root), installActivationDir)
	resolvedActivation, activationResolveErr := installArtifactResolvedPath(activation)
	if resolveErr == nil && rootResolveErr == nil && tombstoneResolveErr == nil &&
		(pathAtOrBelow(resolvedRoot, resolvedClean) || pathAtOrBelow(resolvedTombstone, resolvedClean) ||
			(activationResolveErr == nil && pathAtOrBelow(resolvedActivation, resolvedClean))) {
		return nil
	}
	if _, err := os.Lstat(root); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	prepared, err := validJournalMarker(root, installJournalPrepared)
	if err != nil {
		return err
	}
	if !prepared {
		return fmt.Errorf("install mutation attempted before transaction PREPARED: %s", clean)
	}
	journal, _, err := loadInstallJournal(root)
	if err != nil {
		return err
	}
	resolved := resolvedClean
	if resolveErr != nil {
		return resolveErr
	}
	for _, item := range journal.Items {
		target := foldedInstallPath(item.Target)
		if pathAtOrBelow(target, resolved) || pathAtOrBelow(resolved, target) {
			return nil
		}
		for _, dir := range item.CreatedDirs {
			created := foldedInstallPath(dir)
			if created == resolved {
				return nil
			}
		}
	}
	return fmt.Errorf("install mutation path escaped prepared transaction: %s", clean)
}

func foldedInstallPath(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func pathAtOrBelow(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))))
}

func pathContains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func homeInstallArtifactPaths(homes []string) []string {
	var paths []string
	for _, home := range homes {
		paths = append(paths, filepath.Join(home, "skills", "machinery"))
		for _, doc := range RoleDocs {
			paths = append(paths, filepath.Join(home, "agents", doc))
		}
	}
	return paths
}

func targetInstallArtifactPaths(names []string) ([]string, error) {
	artifacts, err := TargetArtifacts(names)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		paths = append(paths, artifact.Path)
	}
	return paths, nil
}
