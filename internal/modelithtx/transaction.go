// Package modelithtx publishes the authoritative Modelith corpus as one
// durable, crash-recoverable directory transaction.
package modelithtx

import (
	"bytes"
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
)

const (
	stageName            = ".machinery-modelith-stage"
	stageCorpus          = stageName + string(filepath.Separator) + "examples"
	backupName           = ".machinery-modelith-backup"
	retireName           = ".machinery-modelith-retire"
	journalName          = ".machinery-modelith-journal"
	journalNextName      = journalName + ".new"
	journalAuthorityName = journalName + ".authority"
	journalPreviousName  = journalAuthorityName + ".previous"
	journalRetireName    = journalAuthorityName + ".retire"
	targetName           = "examples"
	journalVersion       = 2
	journalPrepared      = "prepared"
	journalRestoring     = "restoring"
	modelithDirPageSize  = 128
	modelithDirEntryMax  = 4096
	modelithTreeEntryMax = 65536
	modelithTreeDepthMax = 256
	modelithFileSizeMax  = 16 << 20
	modelithTreeBytesMax = 256 << 20
)

type journal struct {
	Version          int    `json:"version"`
	Phase            string `json:"phase"`
	ExpectedDigest   string `json:"expected_digest"`
	ExpectedIdentity string `json:"expected_identity"`
	StagedDigest     string `json:"staged_digest"`
}

type hooks struct {
	afterJournal          func() error
	beforeAbortCleanup    func() error
	afterLiveRevalidate   func() error
	afterPark             func() error
	beforeInstalledVerify func() error
	afterInstall          func() error
	beforeRetireVerify    func() error
	beforeRetireMove      func() error
	afterBackup           func() error
}

type corpusEntry struct {
	info   os.FileInfo
	digest [sha256.Size]byte
	dir    bool
}

type corpusState struct {
	root           string
	digest         string
	identityDigest string
	entries        map[string]corpusEntry
	entryCount     int
	fileBytes      int64
}

type recoveryHooks struct {
	beforeRetireVerify        func() error
	beforeRetireMove          func() error
	beforeBackupRestoreVerify func() error
	afterBackupRestore        func() error
	afterJournalIsolation     func() error
}

type regularFileState struct {
	body    []byte
	info    os.FileInfo
	mode    os.FileMode
	size    int64
	modTime int64
	native  string
	change  string
}

// modelithBeforeJournalPhaseMove is a test seam at the final authority
// revalidation/rename boundary. Production code always leaves it nil.
var modelithBeforeJournalPhaseMove func() error

// Fingerprint returns the deterministic content, type, path, and permission
// digest of a strict real tree. Symlinks and special entries are rejected.
func Fingerprint(path string) (digest string, retErr error) {
	before, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("inspect Modelith corpus: %w", err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return "", fmt.Errorf("modelith corpus %s must be a real directory", path)
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return "", fmt.Errorf("open Modelith corpus: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	inside, err := root.Lstat(".")
	if err != nil || !os.SameFile(before, inside) {
		return "", errors.Join(err, fmt.Errorf("modelith corpus changed identity while opening"))
	}
	hash := sha256.New()
	budget := modelithTreeBudget{}
	if err := fingerprintDir(root, ".", ".", hash, &budget, 0); err != nil {
		return "", err
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(before, after) {
		return "", errors.Join(err, fmt.Errorf("modelith corpus changed identity while fingerprinting"))
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

type modelithTreeBudget struct {
	entries int
	bytes   int64
}

func (budget *modelithTreeBudget) add(info os.FileInfo, display string) error {
	budget.entries++
	if budget.entries > modelithTreeEntryMax {
		return fmt.Errorf("modelith corpus exceeds %d-entry limit at %s", modelithTreeEntryMax, display)
	}
	if info.Mode().IsRegular() {
		if info.Size() < 0 || info.Size() > modelithFileSizeMax {
			return fmt.Errorf("modelith corpus file %s has size %d, exceeding %d-byte limit", display, info.Size(), modelithFileSizeMax)
		}
		if budget.bytes > modelithTreeBytesMax-info.Size() {
			return fmt.Errorf("modelith corpus exceeds %d-byte aggregate limit at %s", modelithTreeBytesMax, display)
		}
		budget.bytes += info.Size()
	}
	return nil
}

func fingerprintDir(root *os.Root, dir, displayRoot string, hash io.Writer, budget *modelithTreeBudget, depth int) error {
	if depth > modelithTreeDepthMax {
		return fmt.Errorf("modelith corpus exceeds %d-directory depth limit at %s", modelithTreeDepthMax, filepath.ToSlash(dir))
	}
	dirBefore, err := root.Lstat(dir)
	if err != nil || dirBefore.Mode()&os.ModeSymlink != 0 || !dirBefore.IsDir() {
		return errors.Join(err, fmt.Errorf("modelith corpus directory %s changed type or identity before enumeration", filepath.ToSlash(dir)))
	}
	entries, err := readDir(root, dir)
	if err != nil {
		return fmt.Errorf("read Modelith corpus directory %s: %w", filepath.ToSlash(dir), err)
	}
	for _, entry := range entries {
		name := filepath.Join(dir, entry.Name())
		display := name
		if displayRoot == "." {
			display = strings.TrimPrefix(display, "."+string(filepath.Separator))
		} else {
			display = strings.TrimPrefix(display, displayRoot+string(filepath.Separator))
		}
		display = filepath.ToSlash(display)
		info, err := root.Lstat(name)
		if err != nil {
			return fmt.Errorf("inspect Modelith corpus entry %s: %w", display, err)
		}
		if err := budget.add(info, display); err != nil {
			return err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("modelith corpus entry %s must not be a symlink", display)
		case info.IsDir():
			if _, err := fmt.Fprintf(hash, "D\x00%s\x00%04o\x00", display, info.Mode().Perm()); err != nil {
				return err
			}
			if err := fingerprintDir(root, name, displayRoot, hash, budget, depth+1); err != nil {
				return err
			}
		case info.Mode().IsRegular():
			if _, err := fmt.Fprintf(hash, "F\x00%s\x00%04o\x00", display, info.Mode().Perm()); err != nil {
				return err
			}
			file, err := root.Open(name)
			if err != nil {
				return fmt.Errorf("open Modelith corpus file %s: %w", display, err)
			}
			opened, statErr := file.Stat()
			if statErr != nil || !os.SameFile(info, opened) || opened.Mode() != info.Mode() || opened.Size() != info.Size() {
				return errors.Join(statErr, file.Close(), fmt.Errorf("modelith corpus file %s changed identity while opening", display))
			}
			fileHash := sha256.New()
			copyErr := copyModelithFileExact(fileHash, file, info.Size(), display)
			closeErr := file.Close()
			if err := errors.Join(copyErr, closeErr); err != nil {
				return fmt.Errorf("hash Modelith corpus file %s: %w", display, err)
			}
			after, err := root.Lstat(name)
			if err != nil || !os.SameFile(info, after) || after.Mode() != info.Mode() || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
				return errors.Join(err, fmt.Errorf("modelith corpus file %s changed identity while hashing", display))
			}
			if _, err := fmt.Fprintf(hash, "%x\x00", fileHash.Sum(nil)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("modelith corpus entry %s is special (%s)", display, info.Mode())
		}
	}
	final, err := readDir(root, dir)
	if err != nil {
		return fmt.Errorf("re-read Modelith corpus directory %s: %w", filepath.ToSlash(dir), err)
	}
	dirAfter, statErr := root.Lstat(dir)
	if statErr != nil {
		return statErr
	}
	if !sameDirInventory(entries, final) || !os.SameFile(dirBefore, dirAfter) || dirBefore.Mode() != dirAfter.Mode() || !dirBefore.ModTime().Equal(dirAfter.ModTime()) {
		return fmt.Errorf("modelith corpus directory %s changed while fingerprinting", filepath.ToSlash(dir))
	}
	return nil
}

func copyModelithFileExact(dst io.Writer, src *os.File, size int64, display string) error {
	written, err := io.CopyN(dst, src, size)
	if err != nil {
		return fmt.Errorf("read exact Modelith corpus file %s: copied %d of %d bytes: %w", display, written, size, err)
	}
	var extra [1]byte
	n, probeErr := src.Read(extra[:])
	if n != 0 || probeErr != io.EOF {
		return errors.Join(probeErr, fmt.Errorf("modelith corpus file %s grew while being read", display))
	}
	return nil
}

// Recover rolls back an interrupted pre-install transaction or finalizes an
// already-installed transaction. An unjournaled private stage is discarded.
func Recover(repo string) error {
	return recoverTransaction(repo, nil)
}

func recoverTransaction(repo string, beforeRetireVerify func() error) error {
	return recoverTransactionWithHooks(repo, recoveryHooks{beforeRetireVerify: beforeRetireVerify})
}

func recoverTransactionWithHooks(repo string, testHooks recoveryHooks) error {
	return withRoot(repo, func(root *os.Root) error {
		return recoverRoot(root, testHooks)
	})
}

// Publish atomically replaces examples with the already-validated staged
// corpus. expectedDigest binds the live corpus to the snapshot used to render.
func Publish(repo, expectedDigest string) error {
	return publish(repo, expectedDigest, hooks{})
}

func publish(repo, expectedDigest string, testHooks hooks) error {
	if !validDigest(expectedDigest) {
		return fmt.Errorf("invalid expected Modelith corpus digest %q", expectedDigest)
	}
	return withRoot(repo, func(root *os.Root) (returnErr error) {
		if err := validateReservedInventory(root); err != nil {
			return err
		}
		for _, residue := range []string{journalName, journalNextName, journalAuthorityName, journalPreviousName, journalRetireName, backupName, retireName} {
			exists, err := pathExists(root, residue)
			if err != nil {
				return fmt.Errorf("inspect reserved entry %s before publishing: %w", residue, err)
			}
			if exists {
				return fmt.Errorf("recover interrupted Modelith transaction before publishing: reserved entry %s exists", residue)
			}
		}
		live, err := captureCorpusState(root, targetName)
		if err != nil {
			return err
		}
		if live.digest != expectedDigest {
			return fmt.Errorf("modelith corpus changed while rendering: expected %s, found %s", expectedDigest, live.digest)
		}
		if err := validateReservedDir(root, stageName); err != nil {
			return err
		}
		if err := validateTree(root, stageCorpus); err != nil {
			return fmt.Errorf("validate staged Modelith corpus: %w", err)
		}
		staged, err := captureCorpusState(root, stageCorpus)
		if err != nil {
			return fmt.Errorf("fingerprint staged Modelith corpus: %w", err)
		}
		stagedDigest := staged.digest
		if _, err := root.Lstat(backupName); !os.IsNotExist(err) {
			if err == nil {
				return fmt.Errorf("reserved Modelith backup already exists")
			}
			return fmt.Errorf("inspect reserved Modelith backup: %w", err)
		}
		if err := syncTree(root, stageCorpus); err != nil {
			return fmt.Errorf("sync staged Modelith corpus: %w", err)
		}
		authority, err := writeJournal(root, journal{
			Version:          journalVersion,
			Phase:            journalPrepared,
			ExpectedDigest:   expectedDigest,
			ExpectedIdentity: live.identityDigest,
			StagedDigest:     stagedDigest,
		})
		if err != nil {
			return err
		}
		defer func() {
			if returnErr != nil {
				returnErr = errors.Join(returnErr, restoreJournalAuthorityPath(root))
			}
		}()
		if testHooks.afterJournal != nil {
			if err := testHooks.afterJournal(); err != nil {
				return err
			}
		}
		if err := authority.revalidateAt(root, journalAuthorityName); err != nil {
			return fmt.Errorf("modelith journal authority changed after installation; preserving it: %w", err)
		}
		if err := live.revalidate(root); err != nil {
			if testHooks.beforeAbortCleanup != nil {
				if hookErr := testHooks.beforeAbortCleanup(); hookErr != nil {
					return errors.Join(fmt.Errorf("modelith live corpus changed before publication: %w", err), hookErr)
				}
			}
			return abortUnparked(root, authority, fmt.Errorf("modelith live corpus changed before publication: %w", err))
		}
		if testHooks.afterLiveRevalidate != nil {
			if err := testHooks.afterLiveRevalidate(); err != nil {
				return err
			}
		}
		if err := authority.revalidateAt(root, journalAuthorityName); err != nil {
			return fmt.Errorf("modelith journal authority changed before parking live corpus; preserving it: %w", err)
		}
		if err := fsatomic.RenameNoReplace(root, targetName, backupName); err != nil {
			return fmt.Errorf("park old Modelith corpus: %w", err)
		}
		if err := syncModelithDirectory(root, "."); err != nil {
			return fmt.Errorf("sync parked Modelith corpus: %w", err)
		}
		if testHooks.afterPark != nil {
			if err := testHooks.afterPark(); err != nil {
				return err
			}
		}
		if err := live.revalidateAt(root, backupName); err != nil {
			return abortParked(root, authority, fmt.Errorf("parked Modelith corpus changed before staged publication: %w", err))
		}
		if err := authority.revalidateAt(root, journalAuthorityName); err != nil {
			return abortParked(root, authority, fmt.Errorf("modelith journal authority changed before staged publication; preserving it: %w", err))
		}
		if err := fsatomic.RenameNoReplace(root, stageCorpus, targetName); err != nil {
			return fmt.Errorf("install staged Modelith corpus: %w", err)
		}
		if err := syncModelithDirectory(root, "."); err != nil {
			return fmt.Errorf("sync installed Modelith corpus: %w", err)
		}
		if testHooks.beforeInstalledVerify != nil {
			if err := testHooks.beforeInstalledVerify(); err != nil {
				return err
			}
		}
		if err := staged.revalidateAt(root, targetName); err != nil {
			return fmt.Errorf("installed Modelith corpus changed before verification completed; preserving installed target and backup: %w", err)
		}
		if testHooks.afterInstall != nil {
			if err := testHooks.afterInstall(); err != nil {
				return err
			}
		}
		if err := staged.revalidateAt(root, targetName); err != nil {
			return fmt.Errorf("installed Modelith corpus changed after verification; preserving installed target and backup: %w", err)
		}
		if err := authority.revalidateAt(root, journalAuthorityName); err != nil {
			return fmt.Errorf("modelith journal authority changed before backup retirement; preserving installed target and backup: %w", err)
		}
		if err := retireCorpusIfUnchanged(root, live, staged, testHooks.beforeRetireVerify, testHooks.beforeRetireMove); err != nil {
			return err
		}
		if testHooks.afterBackup != nil {
			if err := testHooks.afterBackup(); err != nil {
				return err
			}
		}
		return finishRecovery(root, authority)
	})
}

func retireCorpusIfUnchanged(root *os.Root, expectedBackup, expectedTarget *corpusState, beforeVerify, beforeMove func() error) error {
	if _, err := root.Lstat(retireName); err == nil {
		return fmt.Errorf("reserved Modelith retirement entry already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect reserved Modelith retirement entry: %w", err)
	}
	if err := fsatomic.RenameNoReplace(root, backupName, retireName); err != nil {
		return fmt.Errorf("isolate old Modelith corpus for retirement: %w", err)
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return fmt.Errorf("sync isolated Modelith corpus retirement: %w", err)
	}
	return deleteRetirementIfUnchanged(root, expectedBackup, expectedTarget, beforeVerify, beforeMove)
}

func deleteRetirementIfUnchanged(root *os.Root, expectedRetirement, expectedTarget *corpusState, beforeVerify, beforeMove func() error) error {
	if beforeVerify != nil {
		if err := beforeVerify(); err != nil {
			return err
		}
	}
	if err := expectedTarget.revalidateAt(root, targetName); err != nil {
		return restoreRetirement(root, fmt.Errorf("installed Modelith corpus changed at backup retirement boundary; preserving installed target and backup: %w", err))
	}
	if err := expectedRetirement.revalidateAt(root, retireName); err != nil {
		return restoreRetirement(root, fmt.Errorf("parked Modelith backup changed at retirement boundary; preserving installed target and backup: %w", err))
	}
	if beforeMove != nil {
		if err := beforeMove(); err != nil {
			return err
		}
	}
	if err := removeCorpusStateExact(root, retireName, expectedRetirement); err != nil {
		return fmt.Errorf("remove verified old Modelith corpus: %w", err)
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return fmt.Errorf("sync old Modelith corpus retirement: %w", err)
	}
	return nil
}

func restoreRetirement(root *os.Root, cause error) error {
	if _, err := root.Lstat(backupName); err == nil {
		return errors.Join(cause, fmt.Errorf("reserved backup reappeared; changed retirement tree remains at %s", retireName))
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.Join(cause, fmt.Errorf("inspect backup before restoring retirement tree: %w", err))
	}
	if err := fsatomic.RenameNoReplace(root, retireName, backupName); err != nil {
		return errors.Join(cause, fmt.Errorf("restore changed retirement tree as backup: %w", err))
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return errors.Join(cause, fmt.Errorf("sync restored changed Modelith backup: %w", err))
	}
	return cause
}

func abortUnparked(root *os.Root, authority *regularFileState, cause error) error {
	if err := clearAbortedUnparked(root, authority); err != nil {
		return errors.Join(cause, fmt.Errorf("abort pre-publication Modelith transaction: %w", err))
	}
	return cause
}

func abortParked(root *os.Root, authority *regularFileState, cause error) error {
	if _, err := root.Lstat(targetName); err == nil {
		return errors.Join(cause, fmt.Errorf("refuse to overwrite concurrent %s; parked live corpus remains at %s", targetName, backupName))
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.Join(cause, fmt.Errorf("inspect publication target before restoring parked corpus: %w", err))
	}
	if err := fsatomic.RenameNoReplace(root, backupName, targetName); err != nil {
		return errors.Join(cause, fmt.Errorf("restore changed parked Modelith corpus: %w", err))
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return errors.Join(cause, fmt.Errorf("sync restored Modelith corpus: %w", err))
	}
	if err := clearAbortedUnparked(root, authority); err != nil {
		return errors.Join(cause, fmt.Errorf("clean restored Modelith transaction: %w", err))
	}
	return cause
}

// clearAbortedUnparked removes the authoritative journal before its private
// stage. Once that removal is durable, recovery can always recognize any
// remaining stage as unjournaled scratch and discard it without interpreting
// or deleting the live corpus.
func clearAbortedUnparked(root *os.Root, authority *regularFileState) error {
	if err := retireJournalAuthority(root, authority); err != nil {
		return err
	}
	if err := removeIfPresent(root, journalNextName); err != nil {
		return err
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return fmt.Errorf("sync aborted Modelith journal cleanup: %w", err)
	}
	if err := removeIfPresent(root, stageName); err != nil {
		return err
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return fmt.Errorf("sync aborted Modelith stage removal: %w", err)
	}
	return nil
}

func captureCorpusState(root *os.Root, dir string) (*corpusState, error) {
	state := &corpusState{root: dir, entries: make(map[string]corpusEntry)}
	hash := sha256.New()
	identityHash := sha256.New()
	if err := state.captureEntry(root, dir, hash, identityHash, 0); err != nil {
		return nil, err
	}
	state.digest = "sha256:" + hex.EncodeToString(hash.Sum(nil))
	state.identityDigest = "sha256:" + hex.EncodeToString(identityHash.Sum(nil))
	return state, nil
}

func (state *corpusState) captureEntry(root *os.Root, name string, hash, identityHash io.Writer, depth int) error {
	if depth > modelithTreeDepthMax {
		return fmt.Errorf("live Modelith corpus exceeds %d-directory depth limit at %s", modelithTreeDepthMax, filepath.ToSlash(name))
	}
	before, err := root.Lstat(name)
	if err != nil {
		return fmt.Errorf("inspect live Modelith corpus entry %s: %w", filepath.ToSlash(name), err)
	}
	if before.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("live Modelith corpus entry %s is a symlink", filepath.ToSlash(name))
	}
	state.entryCount++
	if state.entryCount > modelithTreeEntryMax {
		return fmt.Errorf("live Modelith corpus exceeds %d-entry limit at %s", modelithTreeEntryMax, filepath.ToSlash(name))
	}
	if before.Mode().IsRegular() {
		if before.Size() < 0 || before.Size() > modelithFileSizeMax {
			return fmt.Errorf("live Modelith corpus file %s has size %d, exceeding %d-byte limit", filepath.ToSlash(name), before.Size(), modelithFileSizeMax)
		}
		if state.fileBytes > modelithTreeBytesMax-before.Size() {
			return fmt.Errorf("live Modelith corpus exceeds %d-byte aggregate limit at %s", modelithTreeBytesMax, filepath.ToSlash(name))
		}
		state.fileBytes += before.Size()
	}
	display := strings.TrimPrefix(name, state.root+string(filepath.Separator))
	if name == state.root {
		display = ""
	}
	if before.IsDir() {
		if display != "" {
			if _, err := fmt.Fprintf(hash, "D\x00%s\x00%04o\x00", filepath.ToSlash(display), before.Mode().Perm()); err != nil {
				return err
			}
		}
		initial, err := readDir(root, name)
		if err != nil {
			return fmt.Errorf("read live Modelith corpus directory %s: %w", filepath.ToSlash(name), err)
		}
		for _, child := range initial {
			if err := state.captureEntry(root, filepath.Join(name, child.Name()), hash, identityHash, depth+1); err != nil {
				return err
			}
		}
		final, err := readDir(root, name)
		if err != nil {
			return fmt.Errorf("re-read live Modelith corpus directory %s: %w", filepath.ToSlash(name), err)
		}
		if !sameDirInventory(initial, final) {
			return fmt.Errorf("live Modelith corpus directory %s changed inventory while snapshotting", filepath.ToSlash(name))
		}
		after, err := root.Lstat(name)
		if err != nil || !after.IsDir() || !os.SameFile(before, after) || after.Mode() != before.Mode() || !after.ModTime().Equal(before.ModTime()) {
			return errors.Join(err, fmt.Errorf("live Modelith corpus directory %s changed identity or metadata while snapshotting", filepath.ToSlash(name)))
		}
		if err := writeCorpusIdentity(root, name, display, after, identityHash); err != nil {
			return err
		}
		state.entries[name] = corpusEntry{info: after, dir: true}
		return nil
	}
	if !before.Mode().IsRegular() {
		return fmt.Errorf("live Modelith corpus entry %s is special (%s)", filepath.ToSlash(name), before.Mode())
	}
	if _, err := fmt.Fprintf(hash, "F\x00%s\x00%04o\x00", filepath.ToSlash(display), before.Mode().Perm()); err != nil {
		return err
	}
	file, err := root.Open(name)
	if err != nil {
		return err
	}
	opened, statErr := file.Stat()
	if statErr != nil || !os.SameFile(before, opened) || opened.Mode() != before.Mode() || opened.Size() != before.Size() {
		return errors.Join(statErr, file.Close(), fmt.Errorf("live Modelith corpus file %s changed identity or size while opening", filepath.ToSlash(name)))
	}
	fileHash := sha256.New()
	readErr := copyModelithFileExact(fileHash, file, before.Size(), filepath.ToSlash(name))
	closeErr := file.Close()
	after, pathErr := root.Lstat(name)
	if err := errors.Join(statErr, readErr, closeErr, pathErr); err != nil {
		return err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) || !os.SameFile(before, after) || after.Mode() != before.Mode() ||
		after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return fmt.Errorf("live Modelith corpus file %s changed while snapshotting", filepath.ToSlash(name))
	}
	var digest [sha256.Size]byte
	copy(digest[:], fileHash.Sum(nil))
	if _, err := fmt.Fprintf(hash, "%x\x00", digest); err != nil {
		return err
	}
	if err := writeCorpusIdentity(root, name, display, after, identityHash); err != nil {
		return err
	}
	state.entries[name] = corpusEntry{info: after, digest: digest}
	return nil
}

func writeCorpusIdentity(root *os.Root, name, display string, expected os.FileInfo, hash io.Writer) (retErr error) {
	file, err := root.Open(name)
	if err != nil {
		return fmt.Errorf("open live Modelith corpus entry %s for native identity: %w", filepath.ToSlash(name), err)
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(expected, opened) || opened.Mode() != expected.Mode() {
		return errors.Join(err, fmt.Errorf("live Modelith corpus entry %s changed before native identity inspection", filepath.ToSlash(name)))
	}
	witness, err := modelithNativeEntryWitness(file, opened)
	if err != nil {
		return fmt.Errorf("inspect native identity of live Modelith corpus entry %s: %w", filepath.ToSlash(name), err)
	}
	after, err := root.Lstat(name)
	if err != nil || !os.SameFile(expected, after) || after.Mode() != expected.Mode() ||
		after.Size() != expected.Size() || !after.ModTime().Equal(expected.ModTime()) {
		return errors.Join(err, fmt.Errorf("live Modelith corpus entry %s changed during native identity inspection", filepath.ToSlash(name)))
	}
	_, err = fmt.Fprintf(hash, "%s\x00%s\x00", filepath.ToSlash(display), witness)
	return err
}

func (state *corpusState) revalidate(root *os.Root) error {
	return state.revalidateAt(root, state.root)
}

func (state *corpusState) revalidateAt(root *os.Root, actualRoot string) error {
	seen := make(map[string]bool, len(state.entries))
	if err := state.revalidateEntry(root, state.root, actualRoot, seen); err != nil {
		return err
	}
	if len(seen) != len(state.entries) {
		return fmt.Errorf("live Modelith corpus inventory changed: found %d retained entries, want %d", len(seen), len(state.entries))
	}
	return nil
}

func (state *corpusState) revalidateEntry(root *os.Root, expectedName, actualName string, seen map[string]bool) error {
	want, expected := state.entries[expectedName]
	info, err := root.Lstat(actualName)
	if err != nil {
		return fmt.Errorf("revalidate live Modelith corpus entry %s: %w", filepath.ToSlash(actualName), err)
	}
	if !expected || info.Mode()&os.ModeSymlink != 0 || info.IsDir() != want.dir || !os.SameFile(want.info, info) ||
		info.Mode() != want.info.Mode() || info.Size() != want.info.Size() || !info.ModTime().Equal(want.info.ModTime()) {
		return fmt.Errorf("live Modelith corpus entry %s changed identity, type, or metadata", filepath.ToSlash(actualName))
	}
	seen[expectedName] = true
	if want.dir {
		initial, err := readDir(root, actualName)
		if err != nil {
			return err
		}
		for _, child := range initial {
			expectedChild := filepath.Join(expectedName, child.Name())
			actualChild := filepath.Join(actualName, child.Name())
			if _, exists := state.entries[expectedChild]; !exists {
				return fmt.Errorf("live Modelith corpus gained unexpected entry %s", filepath.ToSlash(actualChild))
			}
			if err := state.revalidateEntry(root, expectedChild, actualChild, seen); err != nil {
				return err
			}
		}
		final, err := readDir(root, actualName)
		if err != nil {
			return err
		}
		if !sameDirInventory(initial, final) {
			return fmt.Errorf("live Modelith corpus directory %s changed inventory during final revalidation", filepath.ToSlash(actualName))
		}
		after, err := root.Lstat(actualName)
		if err != nil || !os.SameFile(want.info, after) || after.Mode() != want.info.Mode() || !after.ModTime().Equal(want.info.ModTime()) {
			return errors.Join(err, fmt.Errorf("live Modelith corpus directory %s changed during final revalidation", filepath.ToSlash(actualName)))
		}
		return nil
	}
	file, err := root.Open(actualName)
	if err != nil {
		return err
	}
	hash := sha256.New()
	readErr := copyModelithFileExact(hash, file, want.info.Size(), filepath.ToSlash(actualName))
	after, statErr := file.Stat()
	closeErr := file.Close()
	pathAfter, pathErr := root.Lstat(actualName)
	if err := errors.Join(readErr, statErr, closeErr, pathErr); err != nil {
		return err
	}
	var digest [sha256.Size]byte
	copy(digest[:], hash.Sum(nil))
	if !os.SameFile(want.info, after) || !os.SameFile(want.info, pathAfter) || after.Mode() != want.info.Mode() ||
		pathAfter.Mode() != want.info.Mode() || after.Size() != want.info.Size() || pathAfter.Size() != want.info.Size() ||
		!after.ModTime().Equal(want.info.ModTime()) || !pathAfter.ModTime().Equal(want.info.ModTime()) || digest != want.digest {
		return fmt.Errorf("live Modelith corpus file %s changed content, identity, or metadata", filepath.ToSlash(actualName))
	}
	return nil
}

func sameDirInventory(left, right []os.DirEntry) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Name() != right[i].Name() {
			return false
		}
	}
	return true
}

func withRoot(repo string, operation func(*os.Root) error) (retErr error) {
	abs, err := filepath.Abs(repo)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	before, err := os.Lstat(abs)
	if err != nil {
		return fmt.Errorf("inspect repository root: %w", err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return fmt.Errorf("repository root %s must be a real directory", abs)
	}
	lock, err := filelock.AcquireWait(filepath.Join(abs, ".machinery-modelith-render"))
	if err != nil {
		return fmt.Errorf("acquire Modelith publication lock: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, lock.Release()) }()
	root, err := os.OpenRoot(abs)
	if err != nil {
		return fmt.Errorf("open repository root: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	inside, err := root.Lstat(".")
	if err != nil || !os.SameFile(before, inside) {
		return errors.Join(err, fmt.Errorf("repository root changed identity while opening"))
	}
	if err := operation(root); err != nil {
		return err
	}
	after, err := os.Lstat(abs)
	if err != nil || !os.SameFile(before, after) {
		return errors.Join(err, fmt.Errorf("repository root changed identity during Modelith publication"))
	}
	return nil
}

func recoverRoot(root *os.Root, testHooks recoveryHooks) (returnErr error) {
	if err := recoverModelithQuarantines(root); err != nil {
		return err
	}
	if err := recoverJournalReplacement(root); err != nil {
		return err
	}
	if err := validateReservedInventory(root); err != nil {
		return err
	}
	authority, exists, err := isolateJournalAuthority(root)
	if err != nil {
		return err
	}
	if !exists {
		for _, retained := range []string{backupName, retireName} {
			present, err := pathExists(root, retained)
			if err != nil {
				return fmt.Errorf("inspect reserved Modelith recovery entry %s: %w", retained, err)
			}
			if present {
				return fmt.Errorf("reserved Modelith recovery entry %s exists without an authoritative journal", retained)
			}
		}
		removed := false
		for _, name := range []string{stageName, journalNextName} {
			present, err := pathExists(root, name)
			if err != nil {
				return fmt.Errorf("inspect unjournaled Modelith transaction entry %s: %w", name, err)
			}
			if present {
				if err := removeValidated(root, name); err != nil {
					return err
				}
				removed = true
			}
		}
		if removed {
			return syncModelithDirectory(root, ".")
		}
		return nil
	}
	defer func() {
		if returnErr != nil {
			returnErr = errors.Join(returnErr, restoreJournalAuthorityPath(root))
		}
	}()
	record, err := decodeJournal(authority.body)
	if err != nil {
		return err
	}
	if testHooks.afterJournalIsolation != nil {
		if err := testHooks.afterJournalIsolation(); err != nil {
			return err
		}
	}
	if err := authority.revalidateAt(root, journalAuthorityName); err != nil {
		return fmt.Errorf("authoritative Modelith recovery journal changed after isolation; preserving it: %w", err)
	}
	target, targetErr := pathExists(root, targetName)
	backup, backupErr := pathExists(root, backupName)
	retiring, retireErr := pathExists(root, retireName)
	staged, stagedErr := pathExists(root, stageCorpus)
	if err := errors.Join(targetErr, backupErr, retireErr, stagedErr); err != nil {
		return fmt.Errorf("inspect Modelith recovery state: %w", err)
	}
	switch {
	case retiring && target && !backup && !staged:
		installed, err := captureCorpusState(root, targetName)
		if err != nil {
			return fmt.Errorf("verify installed Modelith corpus during retirement recovery: %w", err)
		}
		if installed.digest != record.StagedDigest {
			return fmt.Errorf("installed Modelith corpus changed during retirement recovery; preserving installed target and retirement tree")
		}
		retirement, err := captureCorpusState(root, retireName)
		if err != nil {
			return fmt.Errorf("verify retained Modelith retirement tree: %w", err)
		}
		if retirement.digest != record.ExpectedDigest {
			return restoreRetirement(root, fmt.Errorf("retained Modelith retirement digest %s does not match expected digest %s", retirement.digest, record.ExpectedDigest))
		}
		if retirement.identityDigest != record.ExpectedIdentity {
			return restoreRetirement(root, fmt.Errorf("retained Modelith retirement identity does not match the journaled backup identity"))
		}
		if err := authority.revalidateAt(root, journalAuthorityName); err != nil {
			return fmt.Errorf("modelith recovery journal changed before retirement recovery: %w", err)
		}
		if err := deleteRetirementIfUnchanged(root, retirement, installed, testHooks.beforeRetireVerify, testHooks.beforeRetireMove); err != nil {
			return err
		}
	case backup && !target && !retiring:
		retained, err := captureCorpusState(root, backupName)
		if err != nil {
			return fmt.Errorf("verify parked Modelith corpus before restoration: %w", err)
		}
		if retained.digest != record.ExpectedDigest {
			return fmt.Errorf("parked Modelith backup digest %s does not match expected digest %s; preserving backup and recovery journal", retained.digest, record.ExpectedDigest)
		}
		if retained.identityDigest != record.ExpectedIdentity {
			return fmt.Errorf("parked Modelith backup identity does not match the journaled live corpus identity; preserving backup and recovery journal")
		}
		record.Phase = journalRestoring
		authority, err = replaceJournalAuthority(root, authority, record)
		if err != nil {
			return err
		}
		if testHooks.beforeBackupRestoreVerify != nil {
			if err := testHooks.beforeBackupRestoreVerify(); err != nil {
				return err
			}
		}
		if err := retained.revalidateAt(root, backupName); err != nil {
			return fmt.Errorf("parked Modelith backup changed at the atomic restoration boundary; preserving backup and recovery journal: %w", err)
		}
		if err := authority.revalidateAt(root, journalAuthorityName); err != nil {
			return fmt.Errorf("modelith recovery journal changed before restoring backup: %w", err)
		}
		if err := fsatomic.RenameNoReplace(root, backupName, targetName); err != nil {
			return fmt.Errorf("restore parked Modelith corpus: %w", err)
		}
		if err := syncModelithDirectory(root, "."); err != nil {
			return fmt.Errorf("sync restored Modelith corpus: %w", err)
		}
		if testHooks.afterBackupRestore != nil {
			if err := testHooks.afterBackupRestore(); err != nil {
				return err
			}
		}
		if err := retained.revalidateAt(root, targetName); err != nil {
			return fmt.Errorf("restored Modelith corpus changed before restoration completed; preserving target and recovery journal: %w", err)
		}
		if err := authority.revalidateAt(root, journalAuthorityName); err != nil {
			return fmt.Errorf("modelith recovery journal changed before removing restored stage: %w", err)
		}
		if err := removeIfPresent(root, stageName); err != nil {
			return err
		}
	case backup && target && !staged && !retiring:
		installed, err := captureCorpusState(root, targetName)
		if err != nil {
			return fmt.Errorf("verify interrupted installed Modelith corpus: %w", err)
		}
		if installed.digest != record.StagedDigest {
			return fmt.Errorf("interrupted installed Modelith corpus digest %s does not match staged digest %s; preserving installed target and backup for explicit recovery", installed.digest, record.StagedDigest)
		}
		retained, err := captureCorpusState(root, backupName)
		if err != nil {
			return fmt.Errorf("verify interrupted Modelith backup before retirement: %w", err)
		}
		if retained.digest != record.ExpectedDigest {
			return fmt.Errorf("interrupted Modelith backup digest %s does not match expected digest %s; preserving installed target and backup for explicit recovery", retained.digest, record.ExpectedDigest)
		}
		if retained.identityDigest != record.ExpectedIdentity {
			return fmt.Errorf("interrupted Modelith backup identity does not match the journaled live corpus identity; preserving installed target and backup for explicit recovery")
		}
		if err := authority.revalidateAt(root, journalAuthorityName); err != nil {
			return fmt.Errorf("modelith recovery journal changed before retiring backup: %w", err)
		}
		if err := retireCorpusIfUnchanged(root, retained, installed, testHooks.beforeRetireVerify, testHooks.beforeRetireMove); err != nil {
			return err
		}
		if err := removeIfPresent(root, stageName); err != nil {
			return err
		}
	case !backup && target && !retiring:
		if staged {
			if record.Phase == journalRestoring {
				restored, err := captureCorpusState(root, targetName)
				if err != nil {
					return fmt.Errorf("verify restored Modelith corpus: %w", err)
				}
				if restored.digest != record.ExpectedDigest || restored.identityDigest != record.ExpectedIdentity {
					return fmt.Errorf("restored Modelith corpus does not match the journaled backup witness; preserving target, stage, and recovery journal")
				}
				if err := authority.revalidateAt(root, journalAuthorityName); err != nil {
					return fmt.Errorf("modelith recovery journal changed before removing restored stage: %w", err)
				}
				if err := removeIfPresent(root, stageName); err != nil {
					return err
				}
				break
			}
			// No park has happened while the private stage still exists. The
			// live corpus may have been legitimately edited after journal
			// installation; preserve it and revoke journal authority first.
			return clearAbortedUnparked(root, authority)
		}
		digest, err := fingerprintRootDir(root, targetName)
		if err != nil {
			return fmt.Errorf("verify recovered Modelith corpus: %w", err)
		}
		if digest != record.StagedDigest {
			return fmt.Errorf("recovered Modelith corpus digest %s does not match journal digest %s", digest, record.StagedDigest)
		}
		if err := authority.revalidateAt(root, journalAuthorityName); err != nil {
			return fmt.Errorf("modelith recovery journal changed before recovered-stage cleanup: %w", err)
		}
		if err := removeIfPresent(root, stageName); err != nil {
			return err
		}
	default:
		return fmt.Errorf("modelith transaction journal has an impossible target/stage/backup state")
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return fmt.Errorf("sync recovered Modelith transaction: %w", err)
	}
	return finishRecovery(root, authority)
}

func restoreJournalAuthorityPath(root *os.Root) error {
	if exists, err := pathExists(root, journalName); err != nil {
		return err
	} else if exists {
		return nil
	}
	var source string
	for _, candidate := range []string{journalAuthorityName, journalRetireName} {
		exists, err := pathExists(root, candidate)
		if err != nil {
			return err
		}
		if exists {
			if source != "" {
				return fmt.Errorf("multiple isolated Modelith journal authorities remain after failed recovery")
			}
			source = candidate
		}
	}
	if source == "" {
		return nil
	}
	if err := fsatomic.RenameNoReplace(root, source, journalName); err != nil {
		return fmt.Errorf("restore Modelith journal authority after failed recovery: %w", err)
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return fmt.Errorf("sync restored Modelith journal authority: %w", err)
	}
	return nil
}

// recoverJournalReplacement resolves the two durable states of a phase
// transition. The previous journal is moved aside without replacement before
// the next journal is installed, so a concurrent fixed-name replacement is
// never overwritten. Crashes either restore the previous phase or finish an
// already-staged exact successor.
func recoverJournalReplacement(root *os.Root) error {
	previous, previousExists, err := readRegularState(root, journalPreviousName)
	if err != nil || !previousExists {
		return err
	}
	previousRecord, err := decodeJournal(previous.body)
	if err != nil {
		return fmt.Errorf("decode previous Modelith journal phase: %w", err)
	}
	authority, authorityExists, err := readRegularState(root, journalAuthorityName)
	if err != nil {
		return err
	}
	next, nextExists, err := readRegularState(root, journalNextName)
	if err != nil {
		return err
	}
	if authorityExists {
		if nextExists {
			return fmt.Errorf("modelith journal phase transition has both an installed authority and a staged successor; preserving all entries")
		}
		authorityRecord, err := decodeJournal(authority.body)
		if err != nil || !isJournalPhaseSuccessor(previousRecord, authorityRecord) {
			return errors.Join(err, fmt.Errorf("installed Modelith journal is not the exact successor of its previous phase; preserving both"))
		}
		if err := previous.revalidateAt(root, journalPreviousName); err != nil {
			return fmt.Errorf("previous Modelith journal changed before phase retirement; preserving it: %w", err)
		}
		if err := removeRegularStateExact(root, journalPreviousName, previous); err != nil {
			return fmt.Errorf("finish previous Modelith journal phase retirement: %w", err)
		}
		return syncModelithDirectory(root, ".")
	}
	if !nextExists {
		if err := fsatomic.RenameNoReplace(root, journalPreviousName, journalAuthorityName); err != nil {
			return fmt.Errorf("restore previous Modelith journal phase: %w", err)
		}
		return syncModelithDirectory(root, ".")
	}
	nextRecord, err := decodeJournal(next.body)
	if err != nil || !isJournalPhaseSuccessor(previousRecord, nextRecord) {
		return errors.Join(err, fmt.Errorf("staged Modelith journal is not the exact successor of its previous phase; preserving both"))
	}
	if err := fsatomic.RenameNoReplace(root, journalNextName, journalAuthorityName); err != nil {
		return fmt.Errorf("finish installing next Modelith journal phase: %w", err)
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return fmt.Errorf("sync recovered Modelith journal phase: %w", err)
	}
	installed, exists, err := readRegularState(root, journalAuthorityName)
	if err != nil || !exists || !sameRegularFileAfterRename(next, installed) {
		return errors.Join(err, fmt.Errorf("recovered Modelith journal successor changed at installation; preserving both phases"))
	}
	if err := removeRegularStateExact(root, journalPreviousName, previous); err != nil {
		return fmt.Errorf("retire recovered previous Modelith journal phase: %w", err)
	}
	return syncModelithDirectory(root, ".")
}

func writeJournal(root *os.Root, record journal) (*regularFileState, error) {
	if err := installJournal(root, record); err != nil {
		return nil, err
	}
	authority, exists, err := isolateJournalAuthority(root)
	if err != nil || !exists {
		return nil, errors.Join(err, fmt.Errorf("installed Modelith journal could not be isolated as authority"))
	}
	return authority, nil
}

func replaceJournalAuthority(root *os.Root, authority *regularFileState, record journal) (*regularFileState, error) {
	if err := authority.revalidateAt(root, journalAuthorityName); err != nil {
		return nil, fmt.Errorf("modelith journal authority changed before phase transition; preserving it: %w", err)
	}
	if exists, err := pathExists(root, journalPreviousName); err != nil {
		return nil, fmt.Errorf("inspect Modelith journal phase-transition entry %s: %w", journalPreviousName, err)
	} else if exists {
		return nil, fmt.Errorf("modelith journal phase-transition entry %s already exists; preserving it", journalPreviousName)
	}
	if err := removeIfPresent(root, journalNextName); err != nil {
		return nil, fmt.Errorf("clear incomplete Modelith journal successor: %w", err)
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return nil, fmt.Errorf("sync incomplete Modelith journal successor cleanup: %w", err)
	}
	if err := createJournalNext(root, record); err != nil {
		return nil, err
	}
	if modelithBeforeJournalPhaseMove != nil {
		if err := modelithBeforeJournalPhaseMove(); err != nil {
			return nil, err
		}
	}
	if err := fsatomic.RenameNoReplace(root, journalAuthorityName, journalPreviousName); err != nil {
		return nil, fmt.Errorf("isolate previous Modelith journal phase: %w", err)
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return nil, fmt.Errorf("sync previous Modelith journal phase: %w", err)
	}
	previous, exists, err := readRegularState(root, journalPreviousName)
	if err != nil || !exists || !sameRegularFileAfterRename(authority, previous) {
		return nil, errors.Join(err, fmt.Errorf("modelith journal authority changed at phase-transition isolation; preserving it"))
	}
	if err := fsatomic.RenameNoReplace(root, journalNextName, journalAuthorityName); err != nil {
		return nil, fmt.Errorf("install next Modelith journal phase without replacement: %w", err)
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return nil, fmt.Errorf("sync next Modelith journal phase: %w", err)
	}
	updated, exists, err := readRegularState(root, journalAuthorityName)
	if err != nil || !exists {
		return nil, errors.Join(err, fmt.Errorf("replacement Modelith journal authority disappeared"))
	}
	previousRecord, previousErr := decodeJournal(previous.body)
	updatedRecord, updatedErr := decodeJournal(updated.body)
	if err := errors.Join(previousErr, updatedErr); err != nil || !isJournalPhaseSuccessor(previousRecord, updatedRecord) {
		return nil, errors.Join(err, fmt.Errorf("replacement Modelith journal is not the exact successor of its previous authority; preserving both"))
	}
	if err := previous.revalidateAt(root, journalPreviousName); err != nil {
		return nil, fmt.Errorf("previous Modelith journal changed before phase retirement; preserving it: %w", err)
	}
	if err := removeRegularStateExact(root, journalPreviousName, previous); err != nil {
		return nil, fmt.Errorf("retire previous Modelith journal phase: %w", err)
	}
	return updated, nil
}

func installJournal(root *os.Root, record journal) error {
	return installJournalAt(root, record, journalName)
}

func installJournalAt(root *os.Root, record journal, target string) error {
	if err := createJournalNext(root, record); err != nil {
		return err
	}
	if err := fsatomic.RenameNoReplace(root, journalNextName, target); err != nil {
		return fmt.Errorf("install Modelith transaction journal: %w", err)
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return fmt.Errorf("sync Modelith transaction journal directory: %w", err)
	}
	return nil
}

func createJournalNext(root *os.Root, record journal) error {
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}
	body = append(body, '\n')
	file, err := root.OpenFile(journalNextName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create Modelith transaction journal: %w", err)
	}
	if _, writeErr := file.Write(body); writeErr != nil {
		return errors.Join(fmt.Errorf("write Modelith transaction journal: %w", writeErr), file.Close())
	}
	if err := errors.Join(file.Sync(), file.Close()); err != nil {
		return fmt.Errorf("sync Modelith transaction journal: %w", err)
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return fmt.Errorf("sync staged Modelith transaction journal: %w", err)
	}
	return nil
}

func isJournalPhaseSuccessor(previous, next journal) bool {
	validPhase := previous.Phase == journalPrepared && next.Phase == journalRestoring || previous.Phase == journalRestoring && next.Phase == journalRestoring
	return validPhase && previous.Version == next.Version &&
		previous.ExpectedDigest == next.ExpectedDigest && previous.ExpectedIdentity == next.ExpectedIdentity && previous.StagedDigest == next.StagedDigest
}

func decodeJournal(body []byte) (journal, error) {
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	var record journal
	if err := decoder.Decode(&record); err != nil {
		return journal{}, fmt.Errorf("decode Modelith transaction journal: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return journal{}, fmt.Errorf("modelith transaction journal contains trailing content")
	}
	canonical, err := json.Marshal(record)
	if err != nil {
		return journal{}, err
	}
	canonical = append(canonical, '\n')
	validPhase := record.Phase == journalPrepared || record.Phase == journalRestoring
	if !bytes.Equal(canonical, body) || record.Version != journalVersion || !validPhase || !validDigest(record.ExpectedDigest) ||
		!validDigest(record.ExpectedIdentity) || !validDigest(record.StagedDigest) {
		return journal{}, fmt.Errorf("modelith transaction journal is not canonical or supported")
	}
	return record, nil
}

func finishRecovery(root *os.Root, authority *regularFileState) error {
	if err := authority.revalidateAt(root, journalAuthorityName); err != nil {
		return fmt.Errorf("modelith recovery journal changed before completion cleanup; preserving it: %w", err)
	}
	if err := removeIfPresent(root, stageName); err != nil {
		return err
	}
	if err := authority.revalidateAt(root, journalAuthorityName); err != nil {
		return fmt.Errorf("modelith recovery journal changed before scratch cleanup; preserving it: %w", err)
	}
	if err := removeIfPresent(root, journalNextName); err != nil {
		return err
	}
	return retireJournalAuthority(root, authority)
}

func retireJournalAuthority(root *os.Root, authority *regularFileState) error {
	if _, err := root.Lstat(journalRetireName); err == nil {
		return fmt.Errorf("modelith journal retirement entry already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := fsatomic.RenameNoReplace(root, journalAuthorityName, journalRetireName); err != nil {
		return fmt.Errorf("atomically retire Modelith recovery journal: %w", err)
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return fmt.Errorf("sync retired Modelith recovery journal: %w", err)
	}
	retired, exists, err := readRegularState(root, journalRetireName)
	if err != nil || !exists || !sameRegularFileAfterRename(authority, retired) {
		return errors.Join(err, fmt.Errorf("modelith recovery journal changed at retirement boundary; preserving it"))
	}
	if err := removeRegularStateExact(root, journalRetireName, retired); err != nil {
		return fmt.Errorf("remove retired Modelith recovery journal: %w", err)
	}
	return nil
}

func sameRegularFileAfterRename(before, after *regularFileState) bool {
	return before != nil && after != nil && os.SameFile(before.info, after.info) && before.mode == after.mode && before.size == after.size &&
		before.modTime == after.modTime && before.native == after.native && bytes.Equal(before.body, after.body)
}

func validateReservedInventory(root *os.Root) error {
	entries, err := readDir(root, ".")
	if err != nil {
		return err
	}
	allowed := map[string]bool{stageName: true, backupName: true, retireName: true, journalName: true, journalNextName: true, journalAuthorityName: true, journalPreviousName: true, journalRetireName: true}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(strings.ToLower(name), ".machinery-modelith-") {
			continue
		}
		if !allowed[name] {
			return fmt.Errorf("unexpected reserved Modelith transaction entry %q", name)
		}
		info, err := root.Lstat(name)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("reserved Modelith transaction entry %s must not be a symlink", name)
		}
		if (name == stageName || name == backupName || name == retireName) && !info.IsDir() {
			return fmt.Errorf("reserved Modelith transaction entry %s must be a real directory", name)
		}
		if (name == journalName || name == journalNextName || name == journalAuthorityName || name == journalPreviousName || name == journalRetireName) && !info.Mode().IsRegular() {
			return fmt.Errorf("reserved Modelith transaction entry %s must be a regular file", name)
		}
	}
	return nil
}

func validateReservedDir(root *os.Root, name string) error {
	info, err := root.Lstat(name)
	if err != nil {
		return fmt.Errorf("inspect reserved Modelith stage: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("reserved Modelith stage must be a real directory")
	}
	return nil
}

func validateTree(root *os.Root, dir string) error {
	budget := modelithTreeBudget{}
	return validateTreeWithBudget(root, dir, &budget, 0)
}

func validateTreeWithBudget(root *os.Root, dir string, budget *modelithTreeBudget, depth int) error {
	if depth > modelithTreeDepthMax {
		return fmt.Errorf("modelith transaction tree exceeds %d-directory depth limit at %s", modelithTreeDepthMax, filepath.ToSlash(dir))
	}
	entries, err := readDir(root, dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := filepath.Join(dir, entry.Name())
		info, err := root.Lstat(name)
		if err != nil {
			return err
		}
		if err := budget.add(info, filepath.ToSlash(name)); err != nil {
			return err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("%s is a symlink", filepath.ToSlash(name))
		case info.IsDir():
			if err := validateTreeWithBudget(root, name, budget, depth+1); err != nil {
				return err
			}
		case info.Mode().IsRegular():
		default:
			return fmt.Errorf("%s is special (%s)", filepath.ToSlash(name), info.Mode())
		}
	}
	return nil
}

func syncTree(root *os.Root, dir string) error {
	budget := modelithTreeBudget{}
	return syncTreeWithBudget(root, dir, &budget, 0)
}

func syncTreeWithBudget(root *os.Root, dir string, budget *modelithTreeBudget, depth int) error {
	if depth > modelithTreeDepthMax {
		return fmt.Errorf("staged Modelith tree exceeds %d-directory depth limit at %s", modelithTreeDepthMax, filepath.ToSlash(dir))
	}
	entries, err := readDir(root, dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := filepath.Join(dir, entry.Name())
		info, err := root.Lstat(name)
		if err != nil {
			return err
		}
		if err := budget.add(info, filepath.ToSlash(name)); err != nil {
			return err
		}
		if info.IsDir() {
			if err := syncTreeWithBudget(root, name, budget, depth+1); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("staged Modelith entry %s is not a regular file", filepath.ToSlash(name))
		}
		file, err := root.Open(name)
		if err != nil {
			return err
		}
		opened, statErr := file.Stat()
		if statErr != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
			return errors.Join(statErr, file.Close(), fmt.Errorf("staged Modelith entry %s changed identity while syncing", filepath.ToSlash(name)))
		}
		if err := errors.Join(file.Sync(), file.Close()); err != nil {
			return err
		}
	}
	return syncModelithDirectory(root, dir)
}

func fingerprintRootDir(root *os.Root, dir string) (string, error) {
	info, err := root.Lstat(dir)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("modelith corpus must be a real directory")
	}
	hash := sha256.New()
	budget := modelithTreeBudget{}
	if err := fingerprintDir(root, dir, dir, hash, &budget, 0); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func readDir(root *os.Root, dir string) ([]os.DirEntry, error) {
	file, err := root.Open(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]os.DirEntry, 0, modelithDirPageSize)
	var readErr error
	for {
		page, err := file.ReadDir(modelithDirPageSize)
		if len(entries) > modelithDirEntryMax-len(page) {
			readErr = fmt.Errorf("modelith directory %s exceeds %d-entry limit", filepath.ToSlash(dir), modelithDirEntryMax)
			break
		}
		entries = append(entries, page...)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			readErr = err
			break
		}
	}
	closeErr := file.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func readRegularState(root *os.Root, name string) (*regularFileState, bool, error) {
	info, err := root.Lstat(name)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, false, fmt.Errorf("reserved Modelith transaction entry %s must be a regular file", name)
	}
	if info.Size() < 0 || info.Size() > 4096 {
		return nil, false, fmt.Errorf("reserved Modelith transaction entry %s exceeds the 4096-byte limit", name)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, false, err
	}
	opened, statErr := file.Stat()
	if statErr != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() || opened.Size() != info.Size() {
		return nil, false, errors.Join(statErr, file.Close(), fmt.Errorf("reserved Modelith transaction entry %s changed identity while opening", name))
	}
	native, nativeErr := modelithNativeEntryWitness(file, opened)
	if nativeErr != nil {
		return nil, false, errors.Join(nativeErr, file.Close())
	}
	var body bytes.Buffer
	body.Grow(int(opened.Size()))
	readErr := copyModelithFileExact(&body, file, opened.Size(), name)
	openedAfter, statErr := file.Stat()
	pathAfter, pathErr := root.Lstat(name)
	closeErr := file.Close()
	if err := errors.Join(readErr, statErr, pathErr, closeErr); err != nil {
		return nil, false, err
	}
	if !os.SameFile(opened, openedAfter) || !os.SameFile(opened, pathAfter) || opened.Mode() != openedAfter.Mode() || opened.Mode() != pathAfter.Mode() ||
		opened.Size() != openedAfter.Size() || opened.Size() != pathAfter.Size() || !opened.ModTime().Equal(openedAfter.ModTime()) || !opened.ModTime().Equal(pathAfter.ModTime()) ||
		modelithJournalChangeID(opened) != modelithJournalChangeID(openedAfter) || modelithJournalChangeID(opened) != modelithJournalChangeID(pathAfter) {
		return nil, false, fmt.Errorf("reserved Modelith transaction entry %s changed while being read", name)
	}
	return &regularFileState{body: body.Bytes(), info: pathAfter, mode: pathAfter.Mode(), size: pathAfter.Size(), modTime: pathAfter.ModTime().UnixNano(), native: native, change: modelithJournalChangeID(pathAfter)}, true, nil
}

func (state *regularFileState) revalidateAt(root *os.Root, name string) error {
	current, exists, err := readRegularState(root, name)
	if err != nil {
		return err
	}
	if !exists || current == nil || !os.SameFile(state.info, current.info) || state.mode != current.mode || state.size != current.size ||
		state.modTime != current.modTime || state.native != current.native || state.change != current.change || !bytes.Equal(state.body, current.body) {
		return fmt.Errorf("reserved Modelith transaction entry %s changed identity, metadata, or content", name)
	}
	return nil
}

func modelithJournalChangeID(info os.FileInfo) string {
	if info == nil || info.Sys() == nil {
		return ""
	}
	value := reflect.Indirect(reflect.ValueOf(info.Sys()))
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return ""
	}
	for _, name := range []string{"Ctim", "Ctimespec", "LastWriteTime"} {
		field := value.FieldByName(name)
		if !field.IsValid() || field.Kind() != reflect.Struct {
			continue
		}
		for _, pair := range [][2]string{{"Sec", "Nsec"}, {"HighDateTime", "LowDateTime"}} {
			left, leftOK := modelithReflectInteger(field.FieldByName(pair[0]))
			right, rightOK := modelithReflectInteger(field.FieldByName(pair[1]))
			if leftOK && rightOK {
				return fmt.Sprintf("%x:%x", left, right)
			}
		}
	}
	left, leftOK := modelithReflectInteger(value.FieldByName("Ctime"))
	right, rightOK := modelithReflectInteger(value.FieldByName("Ctimensec"))
	if leftOK && rightOK {
		return fmt.Sprintf("%x:%x", left, right)
	}
	return ""
}

func modelithReflectInteger(value reflect.Value) (uint64, bool) {
	if !value.IsValid() {
		return 0, false
	}
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint64(value.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Uint(), true
	default:
		return 0, false
	}
}

func isolateJournalAuthority(root *os.Root) (*regularFileState, bool, error) {
	present := make([]string, 0, 3)
	for _, name := range []string{journalName, journalAuthorityName, journalRetireName} {
		exists, err := pathExists(root, name)
		if err != nil {
			return nil, false, fmt.Errorf("inspect Modelith journal authority %s: %w", name, err)
		}
		if exists {
			present = append(present, name)
		}
	}
	if len(present) == 0 {
		return nil, false, nil
	}
	if len(present) != 1 {
		return nil, false, fmt.Errorf("multiple Modelith journal authority entries exist: %s", strings.Join(present, ", "))
	}
	if present[0] != journalAuthorityName {
		if err := fsatomic.RenameNoReplace(root, present[0], journalAuthorityName); err != nil {
			return nil, false, fmt.Errorf("atomically isolate Modelith recovery journal: %w", err)
		}
		if err := syncModelithDirectory(root, "."); err != nil {
			return nil, false, fmt.Errorf("sync isolated Modelith recovery journal: %w", err)
		}
	}
	state, exists, err := readRegularState(root, journalAuthorityName)
	if err != nil || !exists {
		return nil, false, errors.Join(err, fmt.Errorf("isolated Modelith recovery journal disappeared"))
	}
	return state, true, nil
}

func removeIfPresent(root *os.Root, name string) error {
	exists, err := pathExists(root, name)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return removeValidated(root, name)
}

func removeValidated(root *os.Root, name string) error {
	info, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refuse to remove symlinked Modelith transaction entry %s", name)
	}
	if info.IsDir() {
		state, err := captureCorpusState(root, name)
		if err != nil {
			return err
		}
		return removeCorpusStateExact(root, name, state)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refuse to remove special Modelith transaction entry %s", name)
	}
	state, exists, err := readRegularState(root, name)
	if err != nil || !exists {
		return errors.Join(err, fmt.Errorf("modelith transaction entry %s disappeared before deletion", name))
	}
	return removeRegularStateExact(root, name, state)
}

func removeCorpusStateExact(root *os.Root, name string, state *corpusState) error {
	if state == nil {
		return fmt.Errorf("cannot remove Modelith transaction tree %s without exact authority", name)
	}
	q, err := fsatomic.Quarantine(root, name, modelithQuarantinePrefix(name))
	if err != nil {
		return err
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return errors.Join(err, q.Close())
	}
	if err := state.revalidateAt(q.Root(), q.Name()); err != nil {
		return errors.Join(fmt.Errorf("modelith transaction tree %s changed at deletion isolation; preserving it: %w", name, err), q.Close())
	}
	if err := q.RemoveAll(); err != nil {
		return err
	}
	return syncModelithDirectory(root, ".")
}

func removeRegularStateExact(root *os.Root, name string, state *regularFileState) error {
	q, err := fsatomic.Quarantine(root, name, modelithQuarantinePrefix(name))
	if err != nil {
		return err
	}
	if err := syncModelithDirectory(root, "."); err != nil {
		return errors.Join(err, q.Close())
	}
	isolated, exists, err := readRegularState(q.Root(), q.Name())
	if err != nil || !exists || !sameRegularFileAfterRename(state, isolated) {
		return errors.Join(err, fmt.Errorf("modelith transaction entry %s changed at deletion isolation; preserving it", name), q.Close())
	}
	if err := q.Remove(); err != nil {
		return err
	}
	return syncModelithDirectory(root, ".")
}

func modelithQuarantinePrefix(source string) string {
	sum := sha256.Sum256([]byte(source))
	return ".machinery-modelith-delete-" + hex.EncodeToString(sum[:8]) + "-"
}

func recoverModelithQuarantines(root *os.Root) error {
	entries, err := readDir(root, ".")
	if err != nil {
		return err
	}
	sources := []string{retireName, journalRetireName, journalPreviousName, stageName, journalNextName}
	for _, entry := range entries {
		matched := ""
		for _, source := range sources {
			if strings.HasPrefix(entry.Name(), modelithQuarantinePrefix(source)) {
				matched = source
				break
			}
		}
		if matched == "" {
			continue
		}
		q, err := fsatomic.ResumeQuarantine(root, entry.Name(), matched)
		if err != nil {
			return err
		}
		if _, err := q.Root().Lstat(q.Name()); errors.Is(err, os.ErrNotExist) {
			if err := q.FinishEmpty(); err != nil {
				return err
			}
			continue
		} else if err != nil {
			return errors.Join(err, q.Close())
		}
		if err := q.Restore(); err != nil {
			return errors.Join(fmt.Errorf("restore interrupted Modelith deletion quarantine %s: %w", entry.Name(), err), q.Close())
		}
		if err := syncModelithDirectory(root, "."); err != nil {
			return err
		}
	}
	return nil
}

func pathExists(root *os.Root, name string) (bool, error) {
	_, err := root.Lstat(name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
