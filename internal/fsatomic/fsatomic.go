// Package fsatomic provides identity-relative, non-clobbering filesystem
// transitions for transaction recovery code.
package fsatomic

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	quarantineAttempts      = 16
	quarantineNameMax       = 240
	quarantineSourceMarker  = ".fsatomic."
	quarantineNonceBytes    = 16
	quarantineNonceHexBytes = 32
)

// fsatomicBeforePrivateRemove is a test seam at the protocol check/system-call
// boundary. Production code always leaves it nil.
var fsatomicBeforePrivateRemove func()

var fsatomicRandomRead = rand.Read

var fsatomicRemovePrivateDirectory = removePrivateDirectory

var fsatomicBeforeQuarantineMove func(string) error

var fsatomicAfterQuarantineMove func(string)

func validateRenameNames(oldname, newname string) error {
	for _, candidate := range []struct{ label, name string }{{"source", oldname}, {"destination", newname}} {
		label, name := candidate.label, candidate.name
		slash := filepath.ToSlash(name)
		if !utf8.ValidString(name) || strings.ContainsRune(name, 0) || filepath.IsAbs(name) || strings.Contains(name, ":") || (filepath.Separator == '/' && strings.Contains(name, `\`)) || !fs.ValidPath(slash) || slash == "." {
			return fmt.Errorf("atomic rename %s %q must be a clean relative path", label, name)
		}
	}
	return nil
}

// RenameNoReplace moves oldname within root without replacing newname.
func RenameNoReplace(root *os.Root, oldname, newname string) error {
	return RenameNoReplaceBetween(root, oldname, root, newname)
}

// RenameNoReplaceBetween moves oldname between two opened directory
// authorities without replacing an existing destination. Nested paths are
// resolved into opened, non-symlink parent roots before the platform syscall;
// the syscall itself receives basename-only arguments.
func RenameNoReplaceBetween(oldRoot *os.Root, oldname string, newRoot *os.Root, newname string) error {
	if err := validateRenameNames(oldname, newname); err != nil {
		return err
	}
	if oldRoot == nil || newRoot == nil {
		return errors.New("atomic rename root is nil")
	}
	oldParent, oldBase, oldOwned, err := openRenameParent(oldRoot, oldname)
	if err != nil {
		return fmt.Errorf("open atomic rename source parent: %w", err)
	}
	newParent, newBase, newOwned, err := openRenameParent(newRoot, newname)
	if err != nil {
		if oldOwned {
			return errors.Join(fmt.Errorf("open atomic rename destination parent: %w", err), oldParent.Close())
		}
		return fmt.Errorf("open atomic rename destination parent: %w", err)
	}
	if !validBaseName(oldBase) || !validBaseName(newBase) {
		var closeErr error
		if newOwned {
			closeErr = newParent.Close()
		}
		if oldOwned {
			closeErr = errors.Join(closeErr, oldParent.Close())
		}
		return errors.Join(fmt.Errorf("atomic rename syscall arguments must be base names"), closeErr)
	}
	renameErr := renameNoReplaceBase(oldParent, oldBase, newParent, newBase)
	var closeErr error
	if newOwned {
		closeErr = newParent.Close()
	}
	if oldOwned {
		closeErr = errors.Join(closeErr, oldParent.Close())
	}
	return errors.Join(renameErr, closeErr)
}

func openRenameParent(root *os.Root, name string) (*os.Root, string, bool, error) {
	dir, base := filepath.Dir(name), filepath.Base(name)
	if dir == "." {
		return root, base, false, nil
	}
	current := root
	owned := false
	for _, component := range strings.Split(filepath.ToSlash(dir), "/") {
		info, err := current.Lstat(component)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			if owned {
				err = errors.Join(err, current.Close())
			}
			return nil, "", false, errors.Join(err, fmt.Errorf("atomic rename parent component %q must be a real directory", component))
		}
		next, err := current.OpenRoot(component)
		if err != nil {
			if owned {
				err = errors.Join(err, current.Close())
			}
			return nil, "", false, err
		}
		inside, statErr := next.Lstat(".")
		if statErr != nil || !inside.IsDir() || !os.SameFile(info, inside) {
			closeErr := next.Close()
			if owned {
				closeErr = errors.Join(closeErr, current.Close())
			}
			return nil, "", false, errors.Join(statErr, closeErr, fmt.Errorf("atomic rename parent component %q changed while opening", component))
		}
		if owned {
			if err := current.Close(); err != nil {
				return nil, "", false, errors.Join(err, next.Close())
			}
		}
		current = next
		owned = true
	}
	return current, base, owned, nil
}

// Quarantined is an object moved out of its public protocol name and into an
// unpredictable private directory. Validation and deletion through Root are
// bound to the opened directory, so swapping the public directory name cannot
// redirect cleanup to a replacement.
type Quarantined struct {
	root      *os.Root
	private   *os.Root
	source    string
	directory string
	entry     string
	closed    bool
}

// Quarantine atomically moves source, without replacement, into a newly
// created mode-0700 directory. The random directory is an authority boundary,
// not an identifier: callers must retain the returned opened Root for all
// validation and removal.
func Quarantine(root *os.Root, source, prefix string) (*Quarantined, error) {
	if root == nil {
		return nil, fmt.Errorf("atomic quarantine root is nil")
	}
	if !validBaseName(source) {
		return nil, fmt.Errorf("atomic quarantine source %q must be one base name", source)
	}
	if !validBaseName(prefix) {
		return nil, fmt.Errorf("atomic quarantine prefix %q must be one base name", prefix)
	}
	var directory string
	for range quarantineAttempts {
		var err error
		directory, err = newQuarantineDirectory(prefix, source)
		if err != nil {
			return nil, err
		}
		if err := root.Mkdir(directory, 0o700); err == nil {
			break
		} else if !errors.Is(err, fs.ErrExist) {
			return nil, fmt.Errorf("create atomic quarantine: %w", err)
		}
		directory = ""
	}
	if directory == "" {
		return nil, fmt.Errorf("could not allocate a unique atomic quarantine after %d attempts", quarantineAttempts)
	}
	private, err := openBoundDirectory(root, directory)
	if err != nil {
		// Without an opened directory authority, cleanup by the public random
		// name could remove a replacement. Preserve the directory for explicit
		// inspection instead.
		return nil, fmt.Errorf("open atomic quarantine; preserving unbound directory %s: %w", directory, err)
	}
	q := &Quarantined{root: root, private: private, source: source, directory: directory, entry: "object"}
	if err := syncDirectory(q.private); err != nil {
		return nil, errors.Join(fmt.Errorf("sync atomic quarantine authority: %w", err), q.closeEmpty(), q.Close())
	}
	if fsatomicBeforeQuarantineMove != nil {
		if err := fsatomicBeforeQuarantineMove(directory); err != nil {
			return nil, errors.Join(err, q.closeEmpty(), q.Close())
		}
	}
	if err := q.validateBinding(); err != nil {
		return nil, errors.Join(fmt.Errorf("atomic quarantine changed before source move: %w", err), q.Close())
	}
	if err := RenameNoReplaceBetween(root, source, q.private, q.entry); err != nil {
		return nil, errors.Join(fmt.Errorf("move %s into atomic quarantine: %w", source, err), q.closeEmpty(), q.Close())
	}
	if fsatomicAfterQuarantineMove != nil {
		fsatomicAfterQuarantineMove(directory)
	}
	if err := q.validateBinding(); err != nil {
		return nil, errors.Join(fmt.Errorf("atomic quarantine changed after source move: %w", err), q.Restore(), q.Close())
	}
	if err := errors.Join(syncDirectory(root), syncDirectory(q.private)); err != nil {
		return nil, errors.Join(fmt.Errorf("sync atomic quarantine move: %w", err), q.Restore(), q.Close())
	}
	return q, nil
}

// ResumeQuarantine reopens a private namespace discovered by a caller's
// bounded recovery inventory. directory and source must each be one base name;
// the caller owns the protocol-specific prefix and journal binding.
func ResumeQuarantine(root *os.Root, directory, source string) (*Quarantined, error) {
	if root == nil {
		return nil, fmt.Errorf("atomic quarantine root is nil")
	}
	for _, candidate := range []struct{ label, name string }{{"directory", directory}} {
		label, name := candidate.label, candidate.name
		if !validBaseName(name) {
			return nil, fmt.Errorf("atomic quarantine %s %q must be one base name", label, name)
		}
	}
	recorded, err := quarantineSource(directory)
	if err != nil {
		return nil, err
	}
	if source != "" && recorded != source {
		return nil, fmt.Errorf("atomic quarantine source is %q, want %q", recorded, source)
	}
	private, err := openBoundDirectory(root, directory)
	if err != nil {
		return nil, fmt.Errorf("reopen atomic quarantine: %w", err)
	}
	return &Quarantined{root: root, private: private, source: recorded, directory: directory, entry: "object"}, nil
}

func validBaseName(name string) bool {
	return name != "" && name != "." && utf8.ValidString(name) && filepath.Base(name) == name && fs.ValidPath(name) && !strings.ContainsAny(name, `/\:`)
}

func openBoundDirectory(root *os.Root, name string) (*os.Root, error) {
	outside, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !outside.IsDir() || outside.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("atomic quarantine %q must be a real directory", name)
	}
	private, err := root.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	inside, statErr := private.Lstat(".")
	if statErr != nil || !inside.IsDir() || !os.SameFile(outside, inside) {
		return nil, errors.Join(statErr, private.Close(), fmt.Errorf("atomic quarantine %q changed while opening", name))
	}
	return private, nil
}

func randomNonce(label string) ([quarantineNonceBytes]byte, error) {
	var nonce [quarantineNonceBytes]byte
	n, err := fsatomicRandomRead(nonce[:])
	if err != nil || n != len(nonce) {
		return nonce, errors.Join(err, fmt.Errorf("generate atomic quarantine %s: random source returned %d of %d bytes", label, n, len(nonce)))
	}
	return nonce, nil
}

func newQuarantineDirectory(prefix, source string) (string, error) {
	nonce, err := randomNonce("name")
	if err != nil {
		return "", err
	}
	directory := prefix + hex.EncodeToString(nonce[:]) + quarantineSourceMarker + base64.RawURLEncoding.EncodeToString([]byte(source))
	if len(directory) > quarantineNameMax {
		return "", fmt.Errorf("atomic quarantine name requires %d bytes, exceeding %d-byte portable limit", len(directory), quarantineNameMax)
	}
	return directory, nil
}

func quarantineSource(directory string) (string, error) {
	marker := strings.LastIndex(directory, quarantineSourceMarker)
	if marker < quarantineNonceHexBytes {
		return "", fmt.Errorf("atomic quarantine directory %q has no canonical provenance", directory)
	}
	nonce := directory[marker-quarantineNonceHexBytes : marker]
	if _, err := hex.DecodeString(nonce); err != nil || nonce != strings.ToLower(nonce) {
		return "", fmt.Errorf("atomic quarantine directory %q has no canonical nonce", directory)
	}
	encoded := directory[marker+len(quarantineSourceMarker):]
	body, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || base64.RawURLEncoding.EncodeToString(body) != encoded || !validBaseName(string(body)) {
		return "", errors.Join(err, fmt.Errorf("atomic quarantine directory %q has invalid provenance", directory))
	}
	return string(body), nil
}

func nextQuarantineDirectory(directory string) (string, error) {
	marker := strings.LastIndex(directory, quarantineSourceMarker)
	if marker < quarantineNonceHexBytes {
		return "", fmt.Errorf("atomic quarantine directory %q has no replaceable nonce", directory)
	}
	nonce, err := randomNonce("retirement name")
	if err != nil {
		return "", err
	}
	return directory[:marker-quarantineNonceHexBytes] + hex.EncodeToString(nonce[:]) + directory[marker:], nil
}

// Root returns the opened private namespace used for validation.
func (q *Quarantined) Root() *os.Root { return q.private }

// Name returns the quarantined object's name relative to Root.
func (q *Quarantined) Name() string { return q.entry }

// Source returns the original public base name recorded in the quarantine.
func (q *Quarantined) Source() string { return q.source }

// Restore moves the quarantined object back to its original public name
// without overwriting anything that appeared there concurrently. If the
// public name is occupied, the quarantine remains accessible and Close
// preserves it.
func (q *Quarantined) Restore() error {
	if q == nil || q.closed || q.private == nil {
		return fmt.Errorf("atomic quarantine is closed")
	}
	if err := RenameNoReplaceBetween(q.private, q.entry, q.root, q.source); err != nil {
		return fmt.Errorf("restore atomic quarantine without replacement: %w", err)
	}
	if err := errors.Join(syncDirectory(q.root), syncDirectory(q.private)); err != nil {
		return fmt.Errorf("sync restored atomic quarantine: %w", err)
	}
	return q.closeEmpty()
}

// Remove deletes a quarantined non-directory entry through the held private
// namespace, then retires that empty namespace.
func (q *Quarantined) Remove() error {
	if q == nil || q.closed || q.private == nil {
		return fmt.Errorf("atomic quarantine is closed")
	}
	if err := q.private.Remove(q.entry); err != nil {
		return err
	}
	if err := syncDirectory(q.private); err != nil {
		return err
	}
	return q.closeEmpty()
}

// RemoveAll deletes a quarantined tree through the held private namespace,
// then retires that empty namespace.
func (q *Quarantined) RemoveAll() error {
	if q == nil || q.closed || q.private == nil {
		return fmt.Errorf("atomic quarantine is closed")
	}
	if err := q.private.RemoveAll(q.entry); err != nil {
		return err
	}
	if err := syncDirectory(q.private); err != nil {
		return err
	}
	return q.closeEmpty()
}

// FinishEmpty retires a quarantine that crashed before an object was moved
// into it, or after the object was deleted but before the directory cleanup.
func (q *Quarantined) FinishEmpty() error {
	if q == nil || q.closed || q.private == nil {
		return fmt.Errorf("atomic quarantine is closed")
	}
	if _, err := q.private.Lstat(q.entry); err == nil {
		return fmt.Errorf("atomic quarantine is not empty")
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return q.closeEmpty()
}

// Close releases the private root. A nonempty quarantine deliberately remains
// in place so evidence that could not be validated or restored is preserved.
func (q *Quarantined) Close() error {
	if q == nil || q.closed {
		return nil
	}
	q.closed = true
	err := q.private.Close()
	q.private = nil
	return err
}

func (q *Quarantined) validateBinding() error {
	inside, insideErr := q.private.Lstat(".")
	outside, outsideErr := q.root.Lstat(q.directory)
	if err := errors.Join(insideErr, outsideErr); err != nil {
		return err
	}
	if !inside.IsDir() || !outside.IsDir() || outside.Mode()&os.ModeSymlink != 0 || !os.SameFile(inside, outside) {
		return fmt.Errorf("atomic quarantine directory changed identity; preserving replacement")
	}
	return nil
}

func (q *Quarantined) closeEmpty() error {
	if q == nil || q.closed || q.private == nil {
		return fmt.Errorf("atomic quarantine is closed")
	}
	if recorded, err := quarantineSource(q.directory); err != nil || recorded != q.source {
		return errors.Join(err, fmt.Errorf("atomic quarantine provenance changed; preserving it"))
	}
	dir, err := q.private.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := dir.ReadDir(1)
	closeErr := dir.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return errors.Join(readErr, closeErr)
	}
	if closeErr != nil {
		return closeErr
	}
	if len(entries) != 0 {
		return fmt.Errorf("atomic quarantine private inventory is not empty")
	}
	if err := q.validateBinding(); err != nil {
		return err
	}
	inside, err := q.private.Lstat(".")
	if err != nil {
		return err
	}
	if fsatomicBeforePrivateRemove != nil {
		fsatomicBeforePrivateRemove()
	}
	// The protocol-visible name is never removed. Move the exact candidate,
	// without replacement, to a newly generated name after the final test seam,
	// then rebind that name to the held directory identity. On Unix, which has
	// no handle-only rmdir, this unpredictable retirement name is the deletion
	// authority; a replacement at the old public name is left untouched. The
	// name continues to encode source provenance until removal succeeds.
	var retirement string
	for range quarantineAttempts {
		retirement, err = nextQuarantineDirectory(q.directory)
		if err != nil {
			return err
		}
		if retirement == q.directory {
			continue
		}
		err = RenameNoReplace(q.root, q.directory, retirement)
		if err == nil {
			q.directory = retirement
			break
		}
		if !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("isolate empty atomic quarantine for retirement: %w", err)
		}
		retirement = ""
	}
	if retirement == "" {
		return fmt.Errorf("could not allocate a unique atomic quarantine retirement after %d attempts", quarantineAttempts)
	}
	if err := syncDirectory(q.root); err != nil {
		return fmt.Errorf("sync isolated atomic quarantine retirement: %w", err)
	}
	retired, err := q.root.Lstat(retirement)
	if err != nil || !retired.IsDir() || !os.SameFile(inside, retired) {
		return errors.Join(err, fmt.Errorf("atomic quarantine changed at retirement isolation; preserving it"))
	}
	if err := fsatomicRemovePrivateDirectory(q.root, q.private, retirement); err != nil {
		return fmt.Errorf("remove isolated empty atomic quarantine: %w", err)
	}
	q.closed = true
	closeErr = q.private.Close()
	q.private = nil
	return errors.Join(closeErr, syncDirectory(q.root))
}
