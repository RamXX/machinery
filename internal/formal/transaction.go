package formal

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

	"github.com/RamXX/machinery/internal/portablepath"
)

const (
	formalJournalName        = ".machinery-formal-transaction.jsonl"
	formalRetiredJournalName = ".machinery-formal-transaction.recovering.jsonl"
	formalJournalMax         = 1 << 20
	formalMaxWitness         = "unix:ffffffffffffffff:ffffffffffffffff:ffffffffffffffff:ffffffffffffffff"
)

type formalJournalEntry struct {
	Target      string `json:"target"`
	Stage       string `json:"stage"`
	Backup      string `json:"backup"`
	Existed     bool   `json:"existed"`
	OldIdentity string `json:"old_identity"`
	NewIdentity string `json:"new_identity"`
	OldWitness  string `json:"old_witness"`
	NewWitness  string `json:"new_witness"`
	OldMode     uint32 `json:"old_mode"`
	NewMode     uint32 `json:"new_mode"`
}

type formalJournalHeader struct {
	Version int                  `json:"version"`
	Phase   string               `json:"phase"`
	Entries []formalJournalEntry `json:"entries"`
}

type formalPhaseRecord struct {
	Phase string `json:"phase"`
}

type formalWitnessRecord struct {
	Target  string `json:"target"`
	Image   string `json:"image"`
	Witness string `json:"witness"`
}

type formalJournalOps struct {
	write     func(*os.File, []byte) (int, error)
	sync      func(*os.File) error
	close     func(*os.File) error
	afterOpen func(*os.Root, *os.File) error
	afterSync func(*os.Root, *os.File) error
}

func defaultFormalJournalOps() formalJournalOps {
	return formalJournalOps{
		write: func(file *os.File, body []byte) (int, error) { return file.Write(body) },
		sync:  func(file *os.File) error { return file.Sync() },
		close: func(file *os.File) error { return file.Close() },
	}
}

func formalScratchName(kind, target string) string {
	sum := sha256.Sum256([]byte(target))
	return ".machinery-formal-" + kind + "-" + hex.EncodeToString(sum[:])
}

func formalEntryDeletes(entry formalJournalEntry) bool {
	return entry.Stage == formalScratchName("stage-delete", entry.Target)
}

const formalAbsentIdentity = "absent"

func formalBodyIdentity(body []byte) string {
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validFormalIdentity(identity string) bool {
	if len(identity) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(identity, "sha256:") || identity != strings.ToLower(identity) {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(identity, "sha256:"))
	return err == nil
}

func validFormalNativeWitness(witness string) bool {
	if len(witness) == 0 || len(witness) > len(formalMaxWitness) {
		return false
	}
	parts := strings.Split(witness, ":")
	if len(parts) != 5 && len(parts) != 4 || len(parts) == 5 && parts[0] != "unix" || len(parts) == 4 && parts[0] != "windows" {
		return false
	}
	for _, part := range parts[1:] {
		if len(part) == 0 || len(part) > 16 {
			return false
		}
		if part != strings.ToLower(part) {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				if r < 'a' || r > 'f' {
					return false
				}
			}
		}
	}
	return true
}

func sameFormalNativeObject(left, right string) bool {
	leftParts, rightParts := strings.Split(left, ":"), strings.Split(right, ":")
	return len(leftParts) >= 3 && len(rightParts) >= 3 && leftParts[0] == rightParts[0] && leftParts[1] == rightParts[1] && leftParts[2] == rightParts[2]
}

func syncFormalDir(root *os.Root) error {
	return syncFormalDirectory(root)
}

func encodeFormalRecord(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func createFormalJournal(root *os.Root, entries []formalJournalEntry) error {
	return createFormalJournalWithOps(root, entries, defaultFormalJournalOps())
}

func createFormalJournalWithOps(root *os.Root, entries []formalJournalEntry, ops formalJournalOps) error {
	header := formalJournalHeader{Version: 2, Phase: "prepared", Entries: entries}
	if err := validateFormalJournal(header, "prepared"); err != nil {
		return err
	}
	body, err := encodeFormalRecord(header)
	if err != nil {
		return err
	}
	required, err := formalJournalRequiredSize(body, entries)
	if err != nil {
		return err
	}
	if required > formalJournalMax {
		return fmt.Errorf("formal transaction journal requires %d bytes, exceeding %d-byte limit", required, formalJournalMax)
	}
	f, err := root.OpenFile(formalJournalName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	createdInfo, statErr := f.Stat()
	createdWitness := ""
	var witnessErr error
	if statErr == nil && createdInfo != nil {
		createdWitness, witnessErr = formalNativeFileWitness(f, createdInfo)
	}
	if statErr != nil || witnessErr != nil || createdInfo == nil || !createdInfo.Mode().IsRegular() || createdInfo.Size() != 0 {
		primary := errors.Join(statErr, witnessErr, fmt.Errorf("created formal transaction journal lacks an exact empty regular-file identity"))
		return errors.Join(primary, ops.close(f))
	}
	if ops.afterOpen != nil {
		if err := ops.afterOpen(root, f); err != nil {
			post, postErr := snapshotOpenedFormalJournal(f, true)
			closeErr := ops.close(f)
			return errors.Join(err, closeErr, cleanupCreatedFormalJournal(root, createdInfo, createdWitness, post, postErr))
		}
	}
	written, writeErr := ops.write(f, body)
	if writeErr == nil && written != len(body) {
		writeErr = io.ErrShortWrite
	}
	syncErr := ops.sync(f)
	if ops.afterSync != nil {
		syncErr = errors.Join(syncErr, ops.afterSync(root, f))
	}
	post, snapshotErr := snapshotOpenedFormalJournal(f, true)
	validationErr := snapshotErr
	if validationErr == nil && !bytes.Equal(post.body, body) {
		validationErr = fmt.Errorf("created formal transaction journal bytes differ from the encoded header")
	}
	closeErr := ops.close(f)
	if primary := errors.Join(writeErr, syncErr, validationErr, closeErr); primary != nil {
		return errors.Join(primary, cleanupCreatedFormalJournal(root, createdInfo, createdWitness, post, snapshotErr))
	}
	if err := syncFormalDir(root); err != nil {
		return errors.Join(err, cleanupCreatedFormalJournal(root, createdInfo, createdWitness, post, nil))
	}
	if err := requireFormalJournalLive(root, post, "created formal transaction journal"); err != nil {
		return errors.Join(err, cleanupCreatedFormalJournal(root, createdInfo, createdWitness, post, nil))
	}
	return nil
}

func appendFormalPhase(root *os.Root, phase string) error {
	if phase != "parking" && phase != "installing" && phase != "committed" {
		return fmt.Errorf("invalid formal transaction phase %q", phase)
	}
	body, err := encodeFormalRecord(formalPhaseRecord{Phase: phase})
	if err != nil {
		return err
	}
	return appendFormalJournalRecord(root, body)
}

func appendFormalWitness(root *os.Root, target, image, witness string) error {
	if image != "old" && image != "new" || !validFormalNativeWitness(witness) {
		return fmt.Errorf("invalid formal transaction witness record for %s", target)
	}
	body, err := encodeFormalRecord(formalWitnessRecord{Target: target, Image: image, Witness: witness})
	if err != nil {
		return err
	}
	return appendFormalJournalRecord(root, body)
}

func appendFormalJournalRecord(root *os.Root, body []byte) error {
	return appendFormalJournalRecordWithOps(root, body, defaultFormalJournalOps())
}

func appendFormalJournalRecordWithOps(root *os.Root, body []byte, ops formalJournalOps) error {
	if len(body) == 0 || body[len(body)-1] != '\n' {
		return fmt.Errorf("formal transaction journal append must be one nonempty newline-terminated record")
	}
	info, err := root.Lstat(formalJournalName)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("formal transaction journal must be a regular non-symlink file")
	}
	if info.Size() <= 0 || info.Size() > formalJournalMax || int64(len(body)) > int64(formalJournalMax)-info.Size() {
		return fmt.Errorf("formal transaction journal append would exceed %d-byte limit", formalJournalMax)
	}
	f, err := root.OpenFile(formalJournalName, os.O_RDWR|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	before, snapshotErr := snapshotOpenedFormalJournal(f, false)
	preWitness := ""
	if snapshotErr == nil {
		preWitness, err = formalNativeFileWitness(f, info)
	}
	if snapshotErr != nil || err != nil || !sameFormalJournalMetadata(info, before.info) || preWitness != before.witness {
		return errors.Join(snapshotErr, err, fmt.Errorf("formal transaction journal changed while opening"), ops.close(f))
	}
	if ops.afterOpen != nil {
		if err := ops.afterOpen(root, f); err != nil {
			return errors.Join(err, ops.close(f))
		}
	}
	if err := requireFormalJournalLive(root, before, "formal transaction journal before append"); err != nil {
		return errors.Join(err, ops.close(f))
	}
	written, writeErr := ops.write(f, body)
	if writeErr == nil && written != len(body) {
		writeErr = io.ErrShortWrite
	}
	syncErr := ops.sync(f)
	if ops.afterSync != nil {
		syncErr = errors.Join(syncErr, ops.afterSync(root, f))
	}
	after, snapshotErr := snapshotOpenedFormalJournal(f, false)
	expected := append(append(make([]byte, 0, len(before.body)+len(body)), before.body...), body...)
	if snapshotErr == nil && !bytes.Equal(after.body, expected) {
		snapshotErr = fmt.Errorf("formal transaction journal append did not produce the exact expected bytes")
	}
	closeErr := ops.close(f)
	if err := errors.Join(writeErr, syncErr, snapshotErr, closeErr); err != nil {
		return err
	}
	if err := requireFormalJournalLive(root, after, "formal transaction journal after append"); err != nil {
		return err
	}
	if err := syncFormalDir(root); err != nil {
		return err
	}
	return requireFormalJournalLive(root, after, "durable formal transaction journal append")
}

type formalJournalSnapshot struct {
	body    []byte
	info    os.FileInfo
	witness string
}

type formalJournalAuthority struct {
	file     *os.File
	name     string
	snapshot formalJournalSnapshot
}

func openNamedFormalJournalAuthority(root *os.Root, name string) (*formalJournalAuthority, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("formal transaction journal must be a regular non-symlink file")
	}
	if info.Size() <= 0 || info.Size() > formalJournalMax {
		return nil, fmt.Errorf("formal transaction journal has invalid size %d", info.Size())
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	snapshot, snapshotErr := snapshotOpenedFormalJournal(file, false)
	preWitness := ""
	if snapshotErr == nil {
		preWitness, err = formalNativeFileWitness(file, info)
	}
	if snapshotErr != nil || err != nil || !sameFormalJournalMetadata(info, snapshot.info) || preWitness != snapshot.witness {
		return nil, errors.Join(snapshotErr, err, fmt.Errorf("formal transaction journal changed while opening"), file.Close())
	}
	authority := &formalJournalAuthority{file: file, name: name, snapshot: snapshot}
	if err := authority.requireLive(root, "opened formal transaction journal"); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return authority, nil
}

func (authority *formalJournalAuthority) requireHeld() error {
	if authority == nil || authority.file == nil {
		return fmt.Errorf("formal transaction journal authority is unavailable")
	}
	current, err := snapshotOpenedFormalJournal(authority.file, false)
	if err != nil {
		return err
	}
	if !sameFormalJournalMetadata(authority.snapshot.info, current.info) || authority.snapshot.witness != current.witness || !bytes.Equal(authority.snapshot.body, current.body) {
		return fmt.Errorf("formal transaction journal authority changed identity, metadata, or exact bytes")
	}
	return nil
}

func (authority *formalJournalAuthority) requireHeldAfterUnlink() error {
	if authority == nil || authority.file == nil {
		return fmt.Errorf("formal transaction journal authority is unavailable")
	}
	current, err := snapshotOpenedFormalJournal(authority.file, false)
	if err != nil {
		return err
	}
	if !sameFormalJournalObjectMetadata(authority.snapshot.info, current.info) || !sameFormalNativeObject(authority.snapshot.witness, current.witness) || !bytes.Equal(authority.snapshot.body, current.body) {
		return fmt.Errorf("formal transaction journal authority changed object identity, metadata, or exact bytes")
	}
	return nil
}

func (authority *formalJournalAuthority) refreshAfterRename(root *os.Root, label string) error {
	if authority == nil || authority.file == nil {
		return fmt.Errorf("formal transaction journal authority is unavailable")
	}
	current, err := snapshotOpenedFormalJournal(authority.file, false)
	if err != nil {
		return err
	}
	if !sameFormalJournalObjectMetadata(authority.snapshot.info, current.info) || !sameFormalNativeObject(authority.snapshot.witness, current.witness) || !bytes.Equal(authority.snapshot.body, current.body) {
		return fmt.Errorf("%s changed retained journal object, metadata, or exact bytes", label)
	}
	authority.snapshot = current
	return authority.requireLive(root, label)
}

func (authority *formalJournalAuthority) requireLive(root *os.Root, label string) error {
	if err := authority.requireHeld(); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	return requireNamedFormalJournalLive(root, authority.name, authority.snapshot, label)
}

func (authority *formalJournalAuthority) close() error {
	if authority == nil || authority.file == nil {
		return nil
	}
	err := authority.file.Close()
	authority.file = nil
	return err
}

func snapshotOpenedFormalJournal(file *os.File, allowEmpty bool) (formalJournalSnapshot, error) {
	info, err := file.Stat()
	if err != nil {
		return formalJournalSnapshot{}, err
	}
	if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > formalJournalMax || !allowEmpty && info.Size() == 0 {
		return formalJournalSnapshot{}, fmt.Errorf("formal transaction journal has invalid size %d", info.Size())
	}
	witness, err := formalNativeFileWitness(file, info)
	if err != nil {
		return formalJournalSnapshot{}, err
	}
	body := make([]byte, int(info.Size()))
	if len(body) > 0 {
		n, readErr := file.ReadAt(body, 0)
		if readErr != nil && readErr != io.EOF {
			return formalJournalSnapshot{}, readErr
		}
		if n != len(body) {
			return formalJournalSnapshot{}, io.ErrUnexpectedEOF
		}
	}
	after, err := file.Stat()
	if err != nil {
		return formalJournalSnapshot{}, err
	}
	afterWitness, err := formalNativeFileWitness(file, after)
	if err != nil {
		return formalJournalSnapshot{}, err
	}
	if !sameFormalJournalMetadata(info, after) || witness != afterWitness {
		return formalJournalSnapshot{}, fmt.Errorf("formal transaction journal changed while reading")
	}
	return formalJournalSnapshot{body: body, info: after, witness: afterWitness}, nil
}

func sameFormalJournalMetadata(before, after os.FileInfo) bool {
	return sameFormalJournalObjectMetadata(before, after) && formalJournalChangeID(before) == formalJournalChangeID(after)
}

func sameFormalJournalObjectMetadata(before, after os.FileInfo) bool {
	return before != nil && after != nil && os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}

func formalJournalChangeID(info os.FileInfo) string {
	if info == nil || info.Sys() == nil {
		return ""
	}
	value := reflect.Indirect(reflect.ValueOf(info.Sys()))
	if !value.IsValid() || value.Kind() != reflect.Struct {
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
	sec, nsec := value.FieldByName("Ctime"), value.FieldByName("Ctimensec")
	if sec.IsValid() && nsec.IsValid() {
		var secValue, nsecValue int64
		var secOK, nsecOK bool
		if sec.CanInt() {
			secValue, secOK = sec.Int(), true
		} else if sec.CanUint() {
			secValue, secOK = int64(sec.Uint()), true
		}
		if nsec.CanInt() {
			nsecValue, nsecOK = nsec.Int(), true
		} else if nsec.CanUint() {
			nsecValue, nsecOK = int64(nsec.Uint()), true
		}
		if secOK && nsecOK {
			return fmt.Sprintf("ctime:%d:%d", secValue, nsecValue)
		}
	}
	return ""
}

func requireFormalJournalLive(root *os.Root, expected formalJournalSnapshot, label string) error {
	return requireNamedFormalJournalLive(root, formalJournalName, expected, label)
}

func requireNamedFormalJournalLive(root *os.Root, name string, expected formalJournalSnapshot, label string) error {
	info, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || !sameFormalJournalMetadata(expected.info, info) {
		return fmt.Errorf("%s changed identity, mode, size, or modification time", label)
	}
	witness, err := formalNativeWitness(root, name, label, info)
	if err != nil {
		return err
	}
	if witness != expected.witness {
		return fmt.Errorf("%s changed native identity or change metadata", label)
	}
	return nil
}

func cleanupCreatedFormalJournal(root *os.Root, createdInfo os.FileInfo, createdWitness string, post formalJournalSnapshot, snapshotErr error) error {
	if snapshotErr != nil || createdInfo == nil || post.info == nil || !os.SameFile(createdInfo, post.info) || !sameFormalNativeObject(createdWitness, post.witness) {
		return errors.Join(snapshotErr, fmt.Errorf("cannot prove ownership of failed formal transaction journal; preserving it"))
	}
	if err := requireFormalJournalLive(root, post, "failed formal transaction journal cleanup candidate"); err != nil {
		return errors.Join(err, fmt.Errorf("preserving changed formal transaction journal"))
	}
	if err := root.Remove(formalJournalName); err != nil {
		return err
	}
	return syncFormalDir(root)
}

func formalJournalRequiredSize(header []byte, entries []formalJournalEntry) (int64, error) {
	total := int64(len(header))
	appendRecord := func(value any) error {
		body, err := encodeFormalRecord(value)
		if err != nil {
			return err
		}
		total += int64(len(body))
		return nil
	}
	if err := appendRecord(formalPhaseRecord{Phase: "parking"}); err != nil {
		return 0, err
	}
	for _, entry := range entries {
		if entry.Existed {
			if err := appendRecord(formalWitnessRecord{Target: entry.Target, Image: "old", Witness: formalMaxWitness}); err != nil {
				return 0, err
			}
		}
	}
	if err := appendRecord(formalPhaseRecord{Phase: "installing"}); err != nil {
		return 0, err
	}
	for _, entry := range entries {
		if !formalEntryDeletes(entry) {
			if err := appendRecord(formalWitnessRecord{Target: entry.Target, Image: "new", Witness: formalMaxWitness}); err != nil {
				return 0, err
			}
		}
	}
	if err := appendRecord(formalPhaseRecord{Phase: "committed"}); err != nil {
		return 0, err
	}
	return total, nil
}

func decodeStrictJSON(line []byte, dst any) error {
	if err := rejectFormalDuplicateKeys(line); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func requireFormalJSONKeys(line []byte, allowed ...string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(line, &object); err != nil {
		return nil, err
	}
	want := make(map[string]bool, len(allowed))
	for _, key := range allowed {
		want[key] = true
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !want[key] {
			return nil, fmt.Errorf("unknown exact JSON key %q", key)
		}
	}
	for _, key := range allowed {
		if _, ok := object[key]; !ok {
			return nil, fmt.Errorf("missing JSON key %q", key)
		}
	}
	return object, nil
}

func decodeFormalHeader(line []byte, header *formalJournalHeader) error {
	if err := rejectFormalDuplicateKeys(line); err != nil {
		return err
	}
	object, err := requireFormalJSONKeys(line, "version", "phase", "entries")
	if err != nil {
		return err
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(object["entries"], &entries); err != nil {
		return err
	}
	for _, entry := range entries {
		if _, err := requireFormalJSONKeys(entry, "target", "stage", "backup", "existed", "old_identity", "new_identity", "old_witness", "new_witness", "old_mode", "new_mode"); err != nil {
			return err
		}
	}
	return decodeStrictJSON(line, header)
}

func decodeFormalPhase(line []byte, phase *formalPhaseRecord) error {
	if err := rejectFormalDuplicateKeys(line); err != nil {
		return err
	}
	if _, err := requireFormalJSONKeys(line, "phase"); err != nil {
		return err
	}
	return decodeStrictJSON(line, phase)
}

func decodeFormalWitness(line []byte, witness *formalWitnessRecord) error {
	if err := rejectFormalDuplicateKeys(line); err != nil {
		return err
	}
	if _, err := requireFormalJSONKeys(line, "target", "image", "witness"); err != nil {
		return err
	}
	return decodeStrictJSON(line, witness)
}

func rejectFormalDuplicateKeys(line []byte) error {
	dec := json.NewDecoder(bytes.NewReader(line))
	var walk func() error
	walk = func() error {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for dec.More() {
				keyToken, err := dec.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return fmt.Errorf("object key is not a string")
				}
				if seen[key] {
					return fmt.Errorf("duplicate JSON key %q", key)
				}
				seen[key] = true
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		case '[':
			for dec.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		default:
			return fmt.Errorf("unexpected JSON delimiter %q", delim)
		}
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func readFormalJournal(root *os.Root) (formalJournalHeader, string, error) {
	header, phase, authority, err := readFormalJournalAuthority(root)
	if authority != nil {
		err = errors.Join(err, authority.close())
	}
	return header, phase, err
}

func readFormalJournalAuthority(root *os.Root) (formalJournalHeader, string, *formalJournalAuthority, error) {
	return readNamedFormalJournalAuthority(root, formalJournalName)
}

func readNamedFormalJournalAuthority(root *os.Root, name string) (formalJournalHeader, string, *formalJournalAuthority, error) {
	authority, err := openNamedFormalJournalAuthority(root, name)
	if err != nil {
		return formalJournalHeader{}, "", nil, err
	}
	header, phase, err := parseFormalJournal(root, authority.snapshot.body)
	if err != nil {
		return formalJournalHeader{}, "", nil, errors.Join(err, authority.close())
	}
	if err := authority.requireLive(root, "parsed formal transaction journal"); err != nil {
		return formalJournalHeader{}, "", nil, errors.Join(err, authority.close())
	}
	return header, phase, authority, nil
}

func acquireFormalRecoveryJournal(root *os.Root, rename formalRootRename) (formalJournalHeader, string, *formalJournalAuthority, error) {
	_, canonicalErr := root.Lstat(formalJournalName)
	canonicalExists := canonicalErr == nil
	if canonicalErr != nil && !os.IsNotExist(canonicalErr) {
		return formalJournalHeader{}, "", nil, canonicalErr
	}
	_, retiredErr := root.Lstat(formalRetiredJournalName)
	retiredExists := retiredErr == nil
	if retiredErr != nil && !os.IsNotExist(retiredErr) {
		return formalJournalHeader{}, "", nil, retiredErr
	}
	if canonicalExists && retiredExists {
		return formalJournalHeader{}, "", nil, fmt.Errorf("formal transaction has both canonical and recovering journals; preserving both")
	}
	if retiredExists {
		header, phase, authority, err := readNamedFormalJournalAuthority(root, formalRetiredJournalName)
		if err != nil {
			return formalJournalHeader{}, "", nil, err
		}
		if err := requireCanonicalFormalJournalAbsent(root, "canonical journal beside recovery tombstone"); err != nil {
			return formalJournalHeader{}, "", nil, errors.Join(err, authority.close())
		}
		return header, phase, authority, nil
	}
	if !canonicalExists {
		return formalJournalHeader{}, "", nil, os.ErrNotExist
	}
	header, phase, authority, err := readNamedFormalJournalAuthority(root, formalJournalName)
	if err != nil {
		return formalJournalHeader{}, "", nil, err
	}
	if err := rename(root, formalJournalName, formalRetiredJournalName); err != nil {
		authority.name = formalRetiredJournalName
		if retainedErr := authority.requireLive(root, "recovery journal after interrupted isolation rename"); retainedErr == nil {
			return formalJournalHeader{}, "", nil, errors.Join(err, syncFormalDir(root), authority.close())
		}
		authority.name = formalJournalName
		return formalJournalHeader{}, "", nil, errors.Join(err, authority.close())
	}
	authority.name = formalRetiredJournalName
	if err := authority.refreshAfterRename(root, "isolated formal recovery journal"); err != nil {
		restoreErr := restoreMismatchedFormalJournalIsolation(root)
		return formalJournalHeader{}, "", nil, errors.Join(err, restoreErr, authority.close())
	}
	if err := syncFormalDir(root); err != nil {
		return formalJournalHeader{}, "", nil, errors.Join(err, authority.close())
	}
	if err := authority.requireLive(root, "durable isolated formal recovery journal"); err != nil {
		return formalJournalHeader{}, "", nil, errors.Join(err, authority.close())
	}
	if err := requireCanonicalFormalJournalAbsent(root, "canonical journal after recovery isolation"); err != nil {
		return formalJournalHeader{}, "", nil, errors.Join(err, authority.close())
	}
	return header, phase, authority, nil
}

func restoreMismatchedFormalJournalIsolation(root *os.Root) error {
	if err := requireCanonicalFormalJournalAbsent(root, "canonical journal while restoring failed isolation"); err != nil {
		return fmt.Errorf("preserve mismatched recovery tombstone: %w", err)
	}
	if err := root.Rename(formalRetiredJournalName, formalJournalName); err != nil {
		return fmt.Errorf("restore mismatched recovery tombstone: %w", err)
	}
	return syncFormalDir(root)
}

func requireCanonicalFormalJournalAbsent(root *os.Root, label string) error {
	if _, err := root.Lstat(formalJournalName); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("%s unexpectedly exists", label)
}

func restoreFormalRecoveryAuthority(root *os.Root, authority *formalJournalAuthority) error {
	if authority == nil || authority.name != formalRetiredJournalName {
		return nil
	}
	if err := authority.requireLive(root, "failed formal recovery journal"); err != nil {
		return fmt.Errorf("preserve failed recovery tombstone: %w", err)
	}
	if err := requireCanonicalFormalJournalAbsent(root, "canonical journal while restoring failed recovery"); err != nil {
		return fmt.Errorf("preserve failed recovery tombstone: %w", err)
	}
	if err := root.Rename(formalRetiredJournalName, formalJournalName); err != nil {
		return fmt.Errorf("restore failed formal recovery journal: %w", err)
	}
	authority.name = formalJournalName
	if err := authority.refreshAfterRename(root, "restored failed formal recovery journal"); err != nil {
		return err
	}
	if err := syncFormalDir(root); err != nil {
		return err
	}
	return authority.requireLive(root, "durable restored failed formal recovery journal")
}

func parseFormalJournal(root *os.Root, body []byte) (formalJournalHeader, string, error) {
	complete := body
	fragment := []byte(nil)
	if body[len(body)-1] != '\n' {
		if cut := bytes.LastIndexByte(body, '\n'); cut >= 0 {
			complete, fragment = body[:cut+1], body[cut+1:]
		} else {
			return formalJournalHeader{}, "", fmt.Errorf("formal transaction journal header is incomplete")
		}
	}
	lines := bytes.Split(bytes.TrimSuffix(complete, []byte{'\n'}), []byte{'\n'})
	var header formalJournalHeader
	if len(lines) == 0 || decodeFormalHeader(lines[0], &header) != nil {
		return formalJournalHeader{}, "", fmt.Errorf("formal transaction journal header is malformed")
	}
	phases := []string{"prepared"}
	entryByTarget := make(map[string]int, len(header.Entries))
	seenWitness := map[string]bool{}
	lastWitnessTarget := map[string]string{}
	for i := range header.Entries {
		entryByTarget[header.Entries[i].Target] = i
	}
	for _, line := range lines[1:] {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(line, &object); err != nil {
			return formalJournalHeader{}, "", fmt.Errorf("formal transaction journal record is malformed: %w", err)
		}
		if _, phaseRecord := object["phase"]; phaseRecord {
			var rec formalPhaseRecord
			if err := decodeFormalPhase(line, &rec); err != nil {
				return formalJournalHeader{}, "", fmt.Errorf("formal transaction journal phase is malformed: %w", err)
			}
			phases = append(phases, rec.Phase)
			continue
		}
		var rec formalWitnessRecord
		if err := decodeFormalWitness(line, &rec); err != nil {
			return formalJournalHeader{}, "", fmt.Errorf("formal transaction journal witness is malformed: %w", err)
		}
		idx, ok := entryByTarget[rec.Target]
		if !ok || !validFormalNativeWitness(rec.Witness) {
			return formalJournalHeader{}, "", fmt.Errorf("formal transaction journal witness is invalid")
		}
		key := rec.Image + "\x00" + rec.Target
		if seenWitness[key] || lastWitnessTarget[rec.Image] != "" && rec.Target <= lastWitnessTarget[rec.Image] {
			return formalJournalHeader{}, "", fmt.Errorf("formal transaction journal witness records are duplicated or out of order")
		}
		phase := phases[len(phases)-1]
		switch {
		case phase == "parking" && rec.Image == "old" && header.Entries[idx].Existed:
			header.Entries[idx].OldWitness = rec.Witness
		case phase == "installing" && rec.Image == "new" && !formalEntryDeletes(header.Entries[idx]):
			header.Entries[idx].NewWitness = rec.Witness
		default:
			return formalJournalHeader{}, "", fmt.Errorf("formal transaction journal witness sequence is invalid")
		}
		seenWitness[key] = true
		lastWitnessTarget[rec.Image] = rec.Target
	}
	want := []string{"prepared", "parking", "installing", "committed"}
	if len(phases) > len(want) {
		return formalJournalHeader{}, "", fmt.Errorf("formal transaction journal has too many phase records")
	}
	for i, phase := range phases {
		if phase != want[i] {
			return formalJournalHeader{}, "", fmt.Errorf("formal transaction journal phase sequence is invalid")
		}
	}
	if len(fragment) > 0 {
		next, witness, err := recoverableFormalJournalTail(root, header, phases[len(phases)-1], seenWitness)
		if err != nil {
			return formalJournalHeader{}, "", err
		}
		if !bytes.HasPrefix(next, fragment) {
			return formalJournalHeader{}, "", fmt.Errorf("formal transaction journal has malformed trailing data")
		}
		if witness != nil {
			idx := entryByTarget[witness.Target]
			if witness.Image == "old" {
				header.Entries[idx].OldWitness = witness.Witness
			} else {
				header.Entries[idx].NewWitness = witness.Witness
			}
		}
	}
	phase := phases[len(phases)-1]
	if err := validateFormalJournal(header, phase); err != nil {
		return formalJournalHeader{}, "", err
	}
	return header, phase, nil
}

func recoverableFormalJournalTail(root *os.Root, header formalJournalHeader, phase string, seen map[string]bool) ([]byte, *formalWitnessRecord, error) {
	phaseRecord := func(next string) ([]byte, *formalWitnessRecord, error) {
		body, err := encodeFormalRecord(formalPhaseRecord{Phase: next})
		return body, nil, err
	}
	switch phase {
	case "prepared":
		return phaseRecord("parking")
	case "parking":
		for _, entry := range header.Entries {
			if !entry.Existed || seen["old\x00"+entry.Target] {
				continue
			}
			return recoverableFormalWitnessTail(root, entry, "old")
		}
		return phaseRecord("installing")
	case "installing":
		for _, entry := range header.Entries {
			if formalEntryDeletes(entry) || seen["new\x00"+entry.Target] {
				continue
			}
			return recoverableFormalWitnessTail(root, entry, "new")
		}
		return phaseRecord("committed")
	case "committed":
		return nil, nil, fmt.Errorf("formal transaction journal has trailing data")
	default:
		return nil, nil, fmt.Errorf("formal transaction journal has unknown trailing phase %q", phase)
	}
}

func recoverableFormalWitnessTail(root *os.Root, entry formalJournalEntry, image string) ([]byte, *formalWitnessRecord, error) {
	path, absentPath := entry.Backup, entry.Target
	expectedIdentity, expectedWitness, expectedMode := entry.OldIdentity, entry.OldWitness, entry.OldMode
	if image == "new" {
		path, absentPath = entry.Target, entry.Stage
		expectedIdentity, expectedWitness, expectedMode = entry.NewIdentity, entry.NewWitness, entry.NewMode
	}
	if _, err := root.Lstat(absentPath); err == nil {
		return nil, nil, fmt.Errorf("formal transaction journal partial %s witness has ambiguous live state for %s", image, entry.Target)
	} else if !os.IsNotExist(err) {
		return nil, nil, err
	}
	identity, exists, info, err := formalRegularSnapshot(root, path, "partial journal "+image+" image "+entry.Target)
	if err != nil {
		return nil, nil, err
	}
	if !exists || identity != expectedIdentity || uint32(info.Mode()) != expectedMode {
		return nil, nil, fmt.Errorf("formal transaction journal partial %s witness cannot be derived from durable content for %s", image, entry.Target)
	}
	witness, err := formalNativeWitness(root, path, "partial journal "+image+" image "+entry.Target, info)
	if err != nil {
		return nil, nil, err
	}
	if !sameFormalNativeObject(expectedWitness, witness) {
		return nil, nil, fmt.Errorf("formal transaction journal partial %s witness cannot be derived from the durable native object for %s", image, entry.Target)
	}
	record := &formalWitnessRecord{Target: entry.Target, Image: image, Witness: witness}
	body, err := encodeFormalRecord(*record)
	return body, record, err
}

func validateFormalJournal(header formalJournalHeader, phase string) error {
	if header.Version != 2 || header.Phase != "prepared" || len(header.Entries) == 0 {
		return fmt.Errorf("formal transaction journal header is invalid")
	}
	if phase != "prepared" && phase != "parking" && phase != "installing" && phase != "committed" {
		return fmt.Errorf("formal transaction journal phase %q is invalid", phase)
	}
	folded := map[string]string{}
	ordered := true
	previous := ""
	for _, entry := range header.Entries {
		if previous != "" && entry.Target <= previous {
			ordered = false
		}
		previous = entry.Target
		if err := portablepath.ValidateBase(entry.Target); err != nil {
			return fmt.Errorf("formal transaction journal target is unsafe: %w", err)
		}
		if (entry.Stage != formalScratchName("stage", entry.Target) && entry.Stage != formalScratchName("stage-delete", entry.Target)) || entry.Backup != formalScratchName("backup", entry.Target) {
			return fmt.Errorf("formal transaction journal scratch path does not match target %q", entry.Target)
		}
		if entry.Existed != (entry.OldIdentity != formalAbsentIdentity) || entry.Existed && !validFormalIdentity(entry.OldIdentity) {
			return fmt.Errorf("formal transaction journal old identity is invalid for target %q", entry.Target)
		}
		if entry.Existed != (entry.OldWitness != formalAbsentIdentity) || entry.Existed && !validFormalNativeWitness(entry.OldWitness) {
			return fmt.Errorf("formal transaction journal old native witness is invalid for target %q", entry.Target)
		}
		if !entry.Existed && entry.OldMode != 0 || entry.Existed && !validFormalRegularMode(entry.OldMode) {
			return fmt.Errorf("formal transaction journal old mode is invalid for target %q", entry.Target)
		}
		if formalEntryDeletes(entry) != (entry.NewIdentity == formalAbsentIdentity) || !formalEntryDeletes(entry) && !validFormalIdentity(entry.NewIdentity) {
			return fmt.Errorf("formal transaction journal new identity is invalid for target %q", entry.Target)
		}
		if formalEntryDeletes(entry) != (entry.NewWitness == formalAbsentIdentity) || !formalEntryDeletes(entry) && !validFormalNativeWitness(entry.NewWitness) {
			return fmt.Errorf("formal transaction journal new native witness is invalid for target %q", entry.Target)
		}
		if formalEntryDeletes(entry) && entry.NewMode != 0 || !formalEntryDeletes(entry) && !validFormalRegularMode(entry.NewMode) {
			return fmt.Errorf("formal transaction journal new mode is invalid for target %q", entry.Target)
		}
		for _, name := range []string{entry.Target, entry.Stage, entry.Backup} {
			if filepath.Base(name) != name || name == formalJournalName || name == formalRetiredJournalName {
				return fmt.Errorf("formal transaction journal contains unsafe path %q", name)
			}
			fold := strings.ToLower(name)
			if prior, ok := folded[fold]; ok {
				return fmt.Errorf("formal transaction journal paths %q and %q alias on case-insensitive filesystems", prior, name)
			}
			folded[fold] = name
		}
	}
	if !ordered {
		return fmt.Errorf("formal transaction journal targets are not in deterministic order")
	}
	return nil
}

func validFormalRegularMode(mode uint32) bool {
	allowed := uint32(os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
	return mode&^allowed == 0
}

type formalRecoveryState struct {
	entry                         formalJournalEntry
	targetIdentity, stageIdentity string
	backupIdentity                string
	targetExists, stageExists     bool
	backupExists                  bool
	targetInfo, stageInfo         os.FileInfo
	backupInfo                    os.FileInfo
	targetWitness, stageWitness   string
	backupWitness                 string
}

func formalRegularState(root *os.Root, name, label string) (identity string, exists bool, returnErr error) {
	identity, exists, _, returnErr = formalRegularSnapshot(root, name, label)
	return identity, exists, returnErr
}

func formalRegularSnapshot(root *os.Root, name, label string) (identity string, exists bool, info os.FileInfo, returnErr error) {
	before, err := root.Lstat(name)
	if os.IsNotExist(err) {
		return formalAbsentIdentity, false, nil, nil
	}
	if err != nil {
		return "", false, nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return "", false, nil, fmt.Errorf("formal transaction %s must be a regular non-symlink file", label)
	}
	f, err := root.Open(name)
	if err != nil {
		return "", false, nil, fmt.Errorf("open formal transaction %s: %w", label, err)
	}
	defer func() { returnErr = errors.Join(returnErr, f.Close()) }()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(before, opened) || !opened.Mode().IsRegular() {
		return "", false, nil, errors.Join(err, fmt.Errorf("formal transaction %s changed identity while opening", label))
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", false, nil, fmt.Errorf("hash formal transaction %s: %w", label, err)
	}
	after, err := root.Lstat(name)
	if err != nil || !os.SameFile(before, after) || after.Mode() != before.Mode() || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return "", false, nil, errors.Join(err, fmt.Errorf("formal transaction %s changed while hashing", label))
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), true, after, nil
}

func formalNativeWitness(root *os.Root, name, label string, expectedInfo os.FileInfo) (witness string, returnErr error) {
	if expectedInfo == nil {
		return "", fmt.Errorf("formal transaction %s lacks a preflight identity", label)
	}
	f, err := root.Open(name)
	if err != nil {
		return "", fmt.Errorf("open formal transaction %s for native identity: %w", label, err)
	}
	defer func() { returnErr = errors.Join(returnErr, f.Close()) }()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(expectedInfo, opened) || !opened.Mode().IsRegular() {
		return "", errors.Join(err, fmt.Errorf("formal transaction %s changed before native identity inspection", label))
	}
	witness, err = formalNativeFileWitness(f, opened)
	if err != nil {
		return "", err
	}
	after, err := root.Lstat(name)
	if err != nil || !os.SameFile(expectedInfo, after) || after.Mode() != expectedInfo.Mode() {
		return "", errors.Join(err, fmt.Errorf("formal transaction %s changed during native identity inspection", label))
	}
	return witness, nil
}

type formalRootRename func(*os.Root, string, string) error

func renameFormalRoot(root *os.Root, oldname, newname string) error {
	return root.Rename(oldname, newname)
}

func recoverFormalTransaction(root *os.Root, rename formalRootRename) (returnErr error) {
	_, canonicalErr := root.Lstat(formalJournalName)
	_, retiredErr := root.Lstat(formalRetiredJournalName)
	if os.IsNotExist(canonicalErr) && os.IsNotExist(retiredErr) {
		return nil
	}
	header, phase, journal, err := acquireFormalRecoveryJournal(root, rename)
	if err != nil {
		return err
	}
	defer func() {
		if returnErr != nil {
			returnErr = errors.Join(returnErr, restoreFormalRecoveryAuthority(root, journal))
		}
		returnErr = errors.Join(returnErr, journal.close())
	}()
	mutate := func(label string, action func() error) error {
		if err := journal.requireLive(root, "before "+label); err != nil {
			return err
		}
		if err := requireCanonicalFormalJournalAbsent(root, "canonical journal before "+label); err != nil {
			return err
		}
		actionErr := action()
		authorityErr := journal.requireLive(root, "after "+label)
		canonicalErr := requireCanonicalFormalJournalAbsent(root, "canonical journal after "+label)
		return errors.Join(actionErr, authorityErr, canonicalErr)
	}
	states := make([]formalRecoveryState, 0, len(header.Entries))
	for _, entry := range header.Entries {
		state := formalRecoveryState{entry: entry}
		if state.targetIdentity, state.targetExists, state.targetInfo, err = formalRegularSnapshot(root, entry.Target, "target "+entry.Target); err != nil {
			return err
		}
		if state.targetExists {
			if state.targetWitness, err = formalNativeWitness(root, entry.Target, "target "+entry.Target, state.targetInfo); err != nil {
				return err
			}
		}
		if state.stageIdentity, state.stageExists, state.stageInfo, err = formalRegularSnapshot(root, entry.Stage, "stage "+entry.Stage); err != nil {
			return err
		}
		if state.stageExists {
			if state.stageWitness, err = formalNativeWitness(root, entry.Stage, "stage "+entry.Stage, state.stageInfo); err != nil {
				return err
			}
		}
		if state.backupIdentity, state.backupExists, state.backupInfo, err = formalRegularSnapshot(root, entry.Backup, "backup "+entry.Backup); err != nil {
			return err
		}
		if state.backupExists {
			if state.backupWitness, err = formalNativeWitness(root, entry.Backup, "backup "+entry.Backup, state.backupInfo); err != nil {
				return err
			}
		}
		if state.stageExists && (state.stageIdentity != entry.NewIdentity || state.stageWitness != entry.NewWitness || uint32(state.stageInfo.Mode()) != entry.NewMode) {
			return fmt.Errorf("formal transaction stage %s identity does not match durable new identity; preserving transaction evidence", entry.Stage)
		}
		if state.backupExists {
			if state.backupIdentity != entry.OldIdentity || uint32(state.backupInfo.Mode()) != entry.OldMode {
				return fmt.Errorf("formal transaction backup %s content does not match durable old identity; preserving transaction evidence", entry.Backup)
			}
			if state.backupWitness != entry.OldWitness {
				if !sameFormalNativeObject(state.backupWitness, entry.OldWitness) {
					return fmt.Errorf("formal transaction backup %s native object does not match durable old identity; preserving transaction evidence", entry.Backup)
				}
				// A rename may advance change metadata on filesystems without a
				// stable birth/generation field. The still-present backup keeps
				// the inode allocated, so an equal device+inode identifies the
				// same durable object rather than a reusable-name ABA.
				state.entry.OldWitness = state.backupWitness
			}
		}
		if phase == "installing" && state.targetExists && !state.stageExists && !formalEntryDeletes(state.entry) && state.targetIdentity == state.entry.NewIdentity && uint32(state.targetInfo.Mode()) == state.entry.NewMode && state.targetWitness != state.entry.NewWitness {
			if sameFormalNativeObject(state.targetWitness, state.entry.NewWitness) {
				// The install rename can durably complete before any bytes of
				// its witness record reach the journal. As with a parked old
				// image, the still-allocated device+inode binds the target to
				// the pre-rename staged object while permitting rename-induced
				// change metadata to advance.
				state.entry.NewWitness = state.targetWitness
			}
		}
		if err := validateFormalRecoveryState(state, phase); err != nil {
			return err
		}
		states = append(states, state)
	}
	if phase == "committed" {
		for _, state := range states {
			if err := requireFormalSnapshot(root, state.entry.Target, state.targetIdentity, state.targetWitness, state.targetExists, state.targetInfo, "committed target "+state.entry.Target); err != nil {
				return err
			}
			for _, scratch := range []struct {
				name     string
				exists   bool
				identity string
				witness  string
				info     os.FileInfo
			}{{state.entry.Stage, state.stageExists, state.entry.NewIdentity, state.stageWitness, state.stageInfo}, {state.entry.Backup, state.backupExists, state.entry.OldIdentity, state.backupWitness, state.backupInfo}} {
				if scratch.exists {
					if err := requireFormalSnapshot(root, scratch.name, scratch.identity, scratch.witness, true, scratch.info, "committed scratch "+scratch.name); err != nil {
						return err
					}
					if err := mutate("remove committed scratch "+scratch.name, func() error { return root.Remove(scratch.name) }); err != nil {
						return err
					}
					if err := syncFormalDir(root); err != nil {
						return err
					}
					if err := requireFormalIdentity(root, scratch.name, formalAbsentIdentity, "removed committed scratch "+scratch.name); err != nil {
						return err
					}
				}
			}
		}
	} else {
		for i := len(states) - 1; i >= 0; i-- {
			state := &states[i]
			if state.entry.Existed && state.backupExists {
				if state.targetExists {
					if err := requireFormalSnapshot(root, state.entry.Target, state.targetIdentity, state.targetWitness, true, state.targetInfo, "uncommitted installed target "+state.entry.Target); err != nil {
						return err
					}
					if err := mutate("remove uncommitted target "+state.entry.Target, func() error { return root.Remove(state.entry.Target) }); err != nil {
						return err
					}
					if err := syncFormalDir(root); err != nil {
						return err
					}
					if err := requireFormalIdentity(root, state.entry.Target, formalAbsentIdentity, "removed uncommitted target "+state.entry.Target); err != nil {
						return err
					}
				}
				if err := requireFormalIdentity(root, state.entry.Target, formalAbsentIdentity, "rollback destination "+state.entry.Target); err != nil {
					return err
				}
				if err := requireFormalSnapshot(root, state.entry.Backup, state.backupIdentity, state.backupWitness, true, state.backupInfo, "rollback backup "+state.entry.Backup); err != nil {
					return err
				}
				if err := mutate("restore backup "+state.entry.Backup, func() error { return rename(root, state.entry.Backup, state.entry.Target) }); err != nil {
					return err
				}
				if err := syncFormalDir(root); err != nil {
					return err
				}
				if err := requireFormalIdentity(root, state.entry.Backup, formalAbsentIdentity, "restored backup source "+state.entry.Backup); err != nil {
					return err
				}
				identity, exists, info, err := formalRegularSnapshot(root, state.entry.Target, "restored old target "+state.entry.Target)
				if err != nil || !exists || identity != state.entry.OldIdentity || uint32(info.Mode()) != state.entry.OldMode || !os.SameFile(state.backupInfo, info) {
					return errors.Join(err, fmt.Errorf("restored old target %s does not retain the preflight backup identity", state.entry.Target))
				}
				witness, err := formalNativeWitness(root, state.entry.Target, "restored old target "+state.entry.Target, info)
				if err != nil {
					return err
				}
				state.entry.OldWitness = witness
				state.targetIdentity, state.targetWitness, state.targetExists, state.targetInfo = identity, witness, true, info
				state.backupIdentity, state.backupWitness, state.backupExists, state.backupInfo = "", "", false, nil
			} else if !state.entry.Existed && state.targetExists {
				if err := requireFormalSnapshot(root, state.entry.Target, state.targetIdentity, state.targetWitness, true, state.targetInfo, "uncommitted new target "+state.entry.Target); err != nil {
					return err
				}
				if err := mutate("remove uncommitted new target "+state.entry.Target, func() error { return root.Remove(state.entry.Target) }); err != nil {
					return err
				}
				if err := syncFormalDir(root); err != nil {
					return err
				}
				if err := requireFormalIdentity(root, state.entry.Target, formalAbsentIdentity, "removed uncommitted new target "+state.entry.Target); err != nil {
					return err
				}
			}
			if state.stageExists {
				if err := requireFormalSnapshot(root, state.entry.Stage, state.stageIdentity, state.stageWitness, true, state.stageInfo, "uncommitted stage "+state.entry.Stage); err != nil {
					return err
				}
				if err := mutate("remove uncommitted stage "+state.entry.Stage, func() error { return root.Remove(state.entry.Stage) }); err != nil {
					return err
				}
				if err := syncFormalDir(root); err != nil {
					return err
				}
				if err := requireFormalIdentity(root, state.entry.Stage, formalAbsentIdentity, "removed uncommitted stage "+state.entry.Stage); err != nil {
					return err
				}
			}
		}
	}
	for _, state := range states {
		targetIdentity := state.entry.OldIdentity
		targetWitness := state.entry.OldWitness
		targetMode := state.entry.OldMode
		if phase == "committed" {
			targetIdentity = state.entry.NewIdentity
			targetWitness = state.entry.NewWitness
			targetMode = state.entry.NewMode
		}
		if err := requireFormalDurableWitness(root, state.entry.Target, targetIdentity, targetWitness, targetMode, "final recovered target "+state.entry.Target); err != nil {
			return err
		}
		for _, scratch := range []string{state.entry.Stage, state.entry.Backup} {
			if err := requireFormalIdentity(root, scratch, formalAbsentIdentity, "final recovered scratch "+scratch); err != nil {
				return err
			}
		}
	}
	if err := journal.requireLive(root, "before final formal transaction journal removal"); err != nil {
		return err
	}
	if err := requireCanonicalFormalJournalAbsent(root, "canonical journal before final recovery cleanup"); err != nil {
		return err
	}
	if err := root.Remove(formalRetiredJournalName); err != nil {
		return err
	}
	if err := journal.requireHeldAfterUnlink(); err != nil {
		return fmt.Errorf("removed formal transaction journal changed through retained authority: %w", err)
	}
	if err := syncFormalDir(root); err != nil {
		return err
	}
	for _, name := range []string{formalJournalName, formalRetiredJournalName} {
		if _, err := root.Lstat(name); !os.IsNotExist(err) {
			if err == nil {
				return fmt.Errorf("formal transaction journal %s reappeared after recovery", name)
			}
			return err
		}
	}
	return nil
}

func validateFormalRecoveryState(state formalRecoveryState, phase string) error {
	entry := state.entry
	targetOld := state.targetExists && state.targetIdentity == entry.OldIdentity && state.targetWitness == entry.OldWitness && uint32(state.targetInfo.Mode()) == entry.OldMode
	targetNew := state.targetExists && state.targetIdentity == entry.NewIdentity && state.targetWitness == entry.NewWitness && uint32(state.targetInfo.Mode()) == entry.NewMode
	switch phase {
	case "prepared":
		if state.backupExists || entry.Existed && !targetOld || !entry.Existed && state.targetExists {
			return fmt.Errorf("prepared formal transaction target %s no longer matches its durable old identity; preserving live files and transaction evidence", entry.Target)
		}
	case "parking":
		if entry.Existed {
			if targetNew || targetOld == state.backupExists {
				return fmt.Errorf("parking formal transaction has ambiguous old-image state for %s; preserving live files and transaction evidence", entry.Target)
			}
		} else if state.targetExists || state.backupExists {
			return fmt.Errorf("parking formal transaction has an unexpected target or backup for new file %s; preserving live files and transaction evidence", entry.Target)
		}
	case "installing":
		if entry.Existed {
			switch {
			case state.backupExists && (!state.targetExists || targetNew):
			case !state.backupExists && targetOld:
			default:
				return fmt.Errorf("installing formal transaction target %s does not match a durable old/new state; preserving live files and transaction evidence", entry.Target)
			}
		} else if state.backupExists || state.targetExists && !targetNew {
			return fmt.Errorf("installing formal transaction target %s does not match its durable new identity; preserving live files and transaction evidence", entry.Target)
		}
	case "committed":
		if entry.NewIdentity == formalAbsentIdentity {
			if state.targetExists {
				return fmt.Errorf("committed formal deletion target %s was recreated; preserving live file and transaction evidence", entry.Target)
			}
		} else if !targetNew {
			return fmt.Errorf("committed formal target %s does not match its durable new identity; preserving live file and transaction evidence", entry.Target)
		}
		if state.stageExists {
			return fmt.Errorf("committed formal transaction retained impossible stage %s", entry.Stage)
		}
	default:
		return fmt.Errorf("unknown formal recovery phase %q", phase)
	}
	return nil
}

func requireFormalIdentity(root *os.Root, name, expected, label string) error {
	identity, exists, err := formalRegularState(root, name, label)
	if err != nil {
		return err
	}
	if expected == formalAbsentIdentity {
		if exists {
			return fmt.Errorf("formal transaction %s appeared after validation; preserving it and transaction evidence", label)
		}
		return nil
	}
	if !exists || identity != expected {
		return fmt.Errorf("formal transaction %s changed after validation; preserving live files and transaction evidence", label)
	}
	return nil
}

func requireFormalDurableWitness(root *os.Root, name, expectedIdentity, expectedWitness string, expectedMode uint32, label string) error {
	identity, exists, info, err := formalRegularSnapshot(root, name, label)
	if err != nil {
		return err
	}
	if expectedIdentity == formalAbsentIdentity {
		if exists {
			return fmt.Errorf("formal transaction %s appeared after validation; preserving it and transaction evidence", label)
		}
		return nil
	}
	if !exists || identity != expectedIdentity {
		return fmt.Errorf("formal transaction %s changed after validation; preserving live files and transaction evidence", label)
	}
	witness, err := formalNativeWitness(root, name, label, info)
	if err != nil {
		return err
	}
	if witness != expectedWitness || uint32(info.Mode()) != expectedMode {
		return fmt.Errorf("formal transaction %s has a foreign native identity (got %s, expected %s); preserving live files and transaction evidence", label, witness, expectedWitness)
	}
	return nil
}

func requireFormalSnapshot(root *os.Root, name, expectedIdentity, expectedWitness string, expectedExists bool, expectedInfo os.FileInfo, label string) error {
	identity, exists, info, err := formalRegularSnapshot(root, name, label)
	if err != nil {
		return err
	}
	if exists != expectedExists {
		return fmt.Errorf("formal transaction %s changed existence since preflight; preserving live files and transaction evidence", label)
	}
	if !expectedExists {
		return nil
	}
	witness, err := formalNativeWitness(root, name, label, info)
	if err != nil {
		return err
	}
	if expectedInfo == nil || info == nil || !os.SameFile(expectedInfo, info) || expectedInfo.Mode() != info.Mode() || identity != expectedIdentity || witness != expectedWitness {
		return fmt.Errorf("formal transaction %s changed identity, mode, or content since preflight; preserving live files and transaction evidence", label)
	}
	return nil
}
