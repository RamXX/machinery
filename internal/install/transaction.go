package install

import (
	"crypto/sha256"
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
)

const (
	installJournalSchema     = 3
	installJournalDir        = ".machinery-install-journal"
	installJournalTombstone  = ".machinery-install-journal.committed"
	installJournalMetadata   = "journal.json"
	installJournalPrepared   = "PREPARED"
	installJournalCommitted  = "COMMITTED"
	installJournalScratch    = "scratch"
	installJournalMaxBytes   = 4 << 20
	installJournalMaxTargets = 4096
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

// afterInstallPostImageValidation interleaves adversarial changes after the
// first rollback ownership proof. Production leaves it nil; rollback repeats
// the proof after the hook before removing any live target.
var afterInstallPostImageValidation func(string)

// closeInstallFile is replaceable only by deterministic failure-injection
// tests. Durability helpers must surface both flush and close failures.
var closeInstallFile = func(f *os.File) error { return f.Close() }

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
}

const (
	installPostUnknown = "unknown"
	installPostAbsent  = "absent"
	installPostPresent = "present"
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
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
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
			if err := copyEntryNoFollow(filepath.Join(root, filepath.FromSlash(item.Backup)), item.Target); err != nil {
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

func writeJournalMetadata(root string, journal installJournal) error {
	raw, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	f, err := os.OpenFile(filepath.Join(root, installJournalMetadata), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(raw); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return syncDir(root)
}

func replaceJournalMetadata(root string, journal installJournal) error {
	raw, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	path := filepath.Join(root, installJournalMetadata)
	next := path + ".next"
	f, err := os.OpenFile(next, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(raw); err != nil {
		return errors.Join(err, closeInstallFile(f), durableRemove(next))
	}
	if err := f.Sync(); err != nil {
		return errors.Join(err, closeInstallFile(f), durableRemove(next))
	}
	if err := closeInstallFile(f); err != nil {
		return errors.Join(err, durableRemove(next))
	}
	if err := renameInstallPath(next, path); err != nil {
		return errors.Join(err, durableRemove(next))
	}
	return syncDir(root)
}

func createJournalMarker(root, name string) error {
	f, err := os.OpenFile(filepath.Join(root, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(name + "\n"); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return syncDir(root)
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
		for _, dir := range item.CreatedDirs {
			if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir || !pathContains(dir, item.Target) {
				return fmt.Errorf("journal created directory %q is not a clean ancestor of %s", dir, item.Target)
			}
		}
	}
	return nil
}

func readPrivateRegularFile(path string, limit int64) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || !privateFilePermissionsOK(before) {
		return nil, fmt.Errorf("%s is not a private regular file", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || !os.SameFile(before, after) {
		return nil, fmt.Errorf("%s changed while opening", path)
	}
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", path, limit)
	}
	return raw, nil
}

func validJournalMarker(root, name string) (bool, error) {
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
	raw, err := os.ReadFile(path)
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
	if err := digestArtifactEntry(h, path, ".", exactMetadata); err != nil {
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

func digestArtifactEntry(h digestWriter, path, rel string, exactMetadata bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !supportedArtifactMode(info.Mode()) {
		return fmt.Errorf("unsupported artifact type %s in snapshot", info.Mode().Type())
	}
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
		if _, err := fmt.Fprintf(h, "%d\x00", info.Size()); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(h, f)
		return errors.Join(copyErr, closeInstallFile(f))
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := digestArtifactEntry(h, filepath.Join(path, entry.Name()), filepath.Join(rel, entry.Name()), exactMetadata); err != nil {
			return err
		}
	}
	return nil
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

func copyEntryNoFollow(src, dst string) error {
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
		return copyEntryToRoot(src, capability, rel)
	}
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
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
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyEntryNoFollow(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
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
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
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

func copyEntryToRoot(src string, root *os.Root, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
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
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyEntryToRoot(filepath.Join(src, entry.Name()), root, filepath.Join(dst, entry.Name())); err != nil {
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
	if err := root.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := root.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
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
	info, err := root.Lstat(src)
	if err != nil {
		return err
	}
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
		entries, readErr := dir.ReadDir(-1)
		closeErr := dir.Close()
		if readErr != nil || closeErr != nil {
			return errors.Join(readErr, closeErr)
		}
		// os.Root returns the native directory order. Canonicalize it before
		// recursion so snapshot bytes, partial crash state, and the first
		// reported invalid entry do not depend on filesystem enumeration order.
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			if err := copyEntryFromRoot(root, filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
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
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := root.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
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
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return syncDir(path)
		}
		return nil
	})
}

func syncDir(path string) error {
	return syncDirectoryPath(path)
}

func durableRename(oldPath, newPath string) error {
	if err := declareInstallPresentPostImage(newPath, oldPath); err != nil {
		return err
	}
	if err := validateActiveInstallMutation(newPath); err != nil {
		return err
	}
	_, _, confined, err := retainedCapabilityForMutation(newPath)
	if err != nil {
		return err
	}
	if confined {
		return fmt.Errorf("install target rename requires root-relative replacement")
	}
	if err := renameInstallPath(oldPath, newPath); err != nil {
		return err
	}
	if err := syncDir(filepath.Dir(newPath)); err != nil {
		return err
	}
	if filepath.Dir(oldPath) != filepath.Dir(newPath) {
		return syncDir(filepath.Dir(oldPath))
	}
	return nil
}

func durableRemoveAll(path string) error {
	if err := declareInstallPostImage(path, installPostAbsent, ""); err != nil {
		return err
	}
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
		if _, statErr := capability.Lstat(rel); os.IsNotExist(statErr) {
			return nil
		} else if statErr != nil {
			return statErr
		}
		if err := capability.RemoveAll(rel); err != nil && !os.IsNotExist(err) {
			return err
		}
		return syncRootRelativeDir(capability, filepath.Dir(rel))
	}
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

func durableRemove(path string) error {
	if err := declareInstallPostImage(path, installPostAbsent, ""); err != nil {
		return err
	}
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
		err := capability.Remove(rel)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		return syncRootRelativeDir(capability, filepath.Dir(rel))
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
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
			item.AnchorID != want.AnchorID || item.Digest != want.Digest || strings.Join(item.CreatedDirs, "\x00") != strings.Join(want.CreatedDirs, "\x00") {
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
	if err := declareInstallPostImage(path, installPostAbsent, ""); err != nil {
		return err
	}
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
		err = capability.Remove(rel)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			dir, openErr := capability.Open(rel)
			if openErr == nil {
				entries, readErr := dir.ReadDir(1)
				_ = dir.Close()
				if readErr == nil && len(entries) != 0 {
					return nil
				}
			}
			return fmt.Errorf("remove transaction-created directory %s: %w", path, err)
		}
		return syncRootRelativeDir(capability, filepath.Dir(rel))
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		entries, readErr := os.ReadDir(path)
		if readErr == nil && len(entries) != 0 {
			return nil
		}
		return fmt.Errorf("remove transaction-created directory %s: %w", path, err)
	}
	return syncDir(filepath.Dir(path))
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
	if resolveErr == nil && rootResolveErr == nil && tombstoneResolveErr == nil &&
		(pathAtOrBelow(resolvedRoot, resolvedClean) || pathAtOrBelow(resolvedTombstone, resolvedClean)) {
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
