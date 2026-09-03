package pack

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
	packJournalName       = ".machinery-pack-transaction.jsonl"
	packJournalRetirement = ".machinery-pack-transaction.recovering.jsonl"
	packJournalMax        = 1 << 20
	packAbsentTree        = "absent"
)

var (
	packJournalPoint = func(string) error { return nil }
	packJournalWrite = func(file *os.File, body []byte) (int, error) { return file.Write(body) }
	packJournalSync  = func(file *os.File) error { return file.Sync() }
	packJournalClose = func(file *os.File) error { return file.Close() }
)

type packTreeWitness struct {
	Tree     string `json:"tree"`
	Identity string `json:"identity"`
}

type packJournalEntry struct {
	Target    string          `json:"target"`
	Stage     string          `json:"stage"`
	Backup    string          `json:"backup"`
	Retire    string          `json:"retire"`
	Existed   bool            `json:"existed"`
	Before    packTreeWitness `json:"before"`
	AfterTree string          `json:"after_tree"`
	after     packTreeWitness
}

type packJournalHeader struct {
	Version int                `json:"version"`
	Phase   string             `json:"phase"`
	Entries []packJournalEntry `json:"entries"`
}

type packPhaseRecord struct {
	Phase string `json:"phase"`
}

type packStagedEntry struct {
	Target string          `json:"target"`
	After  packTreeWitness `json:"after"`
}

type packStagedRecord struct {
	Phase   string            `json:"phase"`
	Entries []packStagedEntry `json:"entries"`
}

type packStageRecord struct {
	Phase  string          `json:"phase"`
	Target string          `json:"target"`
	After  packTreeWitness `json:"after"`
}

// packTreeRemovalPoint is a test-only boundary immediately before a retired
// tree is revalidated for destructive removal.
var packTreeRemovalPoint = func(string, string) error { return nil }

func packScratchName(kind, target string) string {
	sum := sha256.Sum256([]byte(target))
	return ".machinery-pack-" + kind + "-" + hex.EncodeToString(sum[:])
}

func packEntryDeletes(entry packJournalEntry) bool {
	return entry.Stage == packScratchName("stage-delete", entry.Target)
}

func syncPackDir(root *os.Root) error { return syncPackDirectory(root) }

func encodePackRecord(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

type packTreeEntryState struct {
	info os.FileInfo
}

type packTreeState struct {
	root    string
	witness packTreeWitness
	entries map[string]packTreeEntryState
}

type packJournalFileState struct {
	info os.FileInfo
	body []byte
}

func samePackJournalMetadata(a, b os.FileInfo) bool {
	return a != nil && b != nil && a.Mode() == b.Mode() && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime()) && packChangeID(a) == packChangeID(b)
}

func capturePackJournalFileAt(root *os.Root, name string) (packJournalFileState, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return packJournalFileState{}, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > packJournalMax {
		return packJournalFileState{}, fmt.Errorf("pack transaction journal must be a bounded regular non-symlink file")
	}
	file, err := root.Open(name)
	if err != nil {
		return packJournalFileState{}, err
	}
	opened, statErr := file.Stat()
	body, readErr := io.ReadAll(io.LimitReader(file, packJournalMax+1))
	closeErr := file.Close()
	after, pathErr := root.Lstat(name)
	if err := errors.Join(statErr, readErr, closeErr, pathErr); err != nil {
		return packJournalFileState{}, err
	}
	if len(body) > packJournalMax || !os.SameFile(before, opened) || !os.SameFile(opened, after) || !samePackJournalMetadata(before, opened) || !samePackJournalMetadata(opened, after) {
		return packJournalFileState{}, fmt.Errorf("pack transaction journal changed while being witnessed")
	}
	return packJournalFileState{info: after, body: body}, nil
}

func capturePackJournalFile(root *os.Root) (packJournalFileState, error) {
	return capturePackJournalFileAt(root, packJournalName)
}

func (state packJournalFileState) revalidateAt(root *os.Root, name string) error {
	current, err := capturePackJournalFileAt(root, name)
	if err != nil {
		return err
	}
	if !os.SameFile(state.info, current.info) || !samePackJournalMetadata(state.info, current.info) || !bytes.Equal(state.body, current.body) {
		return fmt.Errorf("pack transaction journal changed content, identity, size, or metadata")
	}
	return nil
}

func (state packJournalFileState) revalidate(root *os.Root) error {
	return state.revalidateAt(root, packJournalName)
}

func validPackTreeDigest(value string) bool {
	if value == packAbsentTree {
		return true
	}
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func validPackTreeWitness(witness packTreeWitness, allowAbsent bool) bool {
	if witness.Tree == packAbsentTree {
		return allowAbsent && witness.Identity == ""
	}
	return validPackTreeDigest(witness.Tree) && validPackTreeDigest(witness.Identity) && witness.Identity != packAbsentTree
}

func readPackDir(root *os.Root, name string) ([]os.DirEntry, error) {
	dir, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	entries, readErr := dir.ReadDir(-1)
	closeErr := dir.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func packInfoField(info os.FileInfo, names ...string) string {
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

func packNestedInfoField(info os.FileInfo, outer string, names ...string) string {
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

func packStableFileID(info os.FileInfo) string {
	if value := packInfoField(info, "Dev", "Ino"); value != "" {
		return value
	}
	if value := packInfoField(info, "VolumeSerialNumber", "FileIndexHigh", "FileIndexLow"); value != "" {
		return value
	}
	if value := packNestedInfoField(info, "CreationTime", "HighDateTime", "LowDateTime"); value != "" {
		return "creation:" + value
	}
	return ""
}

func packChangeID(info os.FileInfo) string {
	if value := packInfoField(info, "Ctime", "Ctimensec"); value != "" {
		return value
	}
	value := reflect.ValueOf(info.Sys())
	if value.Kind() == reflect.Pointer && !value.IsNil() {
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

func capturePackTree(root *os.Root, name string) (*packTreeState, error) {
	state := &packTreeState{root: name, entries: map[string]packTreeEntryState{}}
	treeHash, identityHash := sha256.New(), sha256.New()
	if err := state.capture(root, name, treeHash, identityHash); err != nil {
		return nil, err
	}
	state.witness = packTreeWitness{
		Tree:     "sha256:" + hex.EncodeToString(treeHash.Sum(nil)),
		Identity: "sha256:" + hex.EncodeToString(identityHash.Sum(nil)),
	}
	return state, nil
}

func (state *packTreeState) capture(root *os.Root, name string, treeHash, identityHash io.Writer) error {
	before, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if before.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("pack tree entry %s is a symlink", filepath.ToSlash(name))
	}
	display := strings.TrimPrefix(name, state.root+string(filepath.Separator))
	if name == state.root {
		display = ""
	}
	kind := "F"
	if before.IsDir() {
		kind = "D"
	} else if !before.Mode().IsRegular() {
		return fmt.Errorf("pack tree entry %s is special (%s)", filepath.ToSlash(name), before.Mode())
	}
	if _, err := fmt.Fprintf(treeHash, "%s\x00%s\x00%04o\x00", kind, filepath.ToSlash(display), before.Mode().Perm()); err != nil {
		return err
	}
	if before.IsDir() {
		initial, err := readPackDir(root, name)
		if err != nil {
			return err
		}
		for _, child := range initial {
			if err := state.capture(root, filepath.Join(name, child.Name()), treeHash, identityHash); err != nil {
				return err
			}
		}
		final, err := readPackDir(root, name)
		if err != nil {
			return err
		}
		if len(initial) != len(final) {
			return fmt.Errorf("pack tree directory %s changed inventory while fingerprinting", filepath.ToSlash(name))
		}
		for i := range initial {
			if initial[i].Name() != final[i].Name() {
				return fmt.Errorf("pack tree directory %s changed inventory while fingerprinting", filepath.ToSlash(name))
			}
		}
	} else {
		file, err := root.Open(name)
		if err != nil {
			return err
		}
		opened, statErr := file.Stat()
		fileHash := sha256.New()
		_, readErr := io.Copy(fileHash, file)
		closeErr := file.Close()
		if err := errors.Join(statErr, readErr, closeErr); err != nil {
			return err
		}
		if !opened.Mode().IsRegular() || !os.SameFile(before, opened) || opened.Mode() != before.Mode() {
			return fmt.Errorf("pack tree file %s changed identity while fingerprinting", filepath.ToSlash(name))
		}
		if _, err := fmt.Fprintf(treeHash, "%x\x00", fileHash.Sum(nil)); err != nil {
			return err
		}
	}
	after, err := root.Lstat(name)
	if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return errors.Join(err, fmt.Errorf("pack tree entry %s changed while fingerprinting", filepath.ToSlash(name)))
	}
	changeID := packChangeID(after)
	if display == "" {
		// Renaming the witnessed tree changes the root directory's ctime on
		// Unix even though the directory identity and contents are unchanged.
		changeID = ""
	}
	if _, err := fmt.Fprintf(identityHash, "%s\x00%s\x00%04o\x00%d\x00%d\x00%s\x00%s\x00", kind, filepath.ToSlash(display), after.Mode().Perm(), after.Size(), after.ModTime().UnixNano(), packStableFileID(after), changeID); err != nil {
		return err
	}
	state.entries[name] = packTreeEntryState{info: after}
	return nil
}

func (state *packTreeState) revalidateAt(root *os.Root, actual string) error {
	current, err := capturePackTree(root, actual)
	if err != nil {
		return err
	}
	if current.witness != state.witness || len(current.entries) != len(state.entries) {
		return fmt.Errorf("pack tree changed content, inventory, identity, or metadata")
	}
	for expectedName, expected := range state.entries {
		rel := strings.TrimPrefix(expectedName, state.root)
		actualName := actual + rel
		got, ok := current.entries[actualName]
		if !ok || !os.SameFile(expected.info, got.info) {
			return fmt.Errorf("pack tree entry %s changed identity", filepath.ToSlash(actualName))
		}
	}
	return nil
}

func generatedPackTreeDigest(names []string, files map[string]string) string {
	hash := sha256.New()
	fmt.Fprintf(hash, "D\x00\x00%04o\x00", os.FileMode(0o755))
	for _, name := range names {
		fmt.Fprintf(hash, "F\x00%s\x00%04o\x00", filepath.ToSlash(name), os.FileMode(0o644))
		sum := sha256.Sum256([]byte(files[name]))
		fmt.Fprintf(hash, "%x\x00", sum)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func createPackJournal(root *os.Root, entries []packJournalEntry) error {
	header := packJournalHeader{Version: 2, Phase: "prepared", Entries: entries}
	if err := validatePackJournal(header, "prepared"); err != nil {
		return err
	}
	body, err := encodePackRecord(header)
	if err != nil {
		return err
	}
	if len(body) > packJournalMax {
		return fmt.Errorf("pack transaction journal exceeds %d bytes", packJournalMax)
	}
	f, err := root.OpenFile(packJournalName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	created, statErr := f.Stat()
	if statErr != nil || !created.Mode().IsRegular() {
		return errors.Join(statErr, packJournalClose(f), cleanupCreatedPackJournal(root, created))
	}
	written, writeErr := packJournalWrite(f, body)
	if writeErr == nil && written != len(body) {
		writeErr = io.ErrShortWrite
	}
	syncErr := packJournalSync(f)
	closeErr := packJournalClose(f)
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return errors.Join(err, cleanupCreatedPackJournal(root, created))
	}
	if err := packJournalPoint("create-before-rebind"); err != nil {
		return err
	}
	state, err := capturePackJournalFile(root)
	if err != nil {
		return err
	}
	if !os.SameFile(created, state.info) || !bytes.Equal(state.body, body) || state.info.Mode().Perm() != 0o600 {
		return fmt.Errorf("created pack transaction journal changed before authority rebind; preserving it")
	}
	return syncPackDir(root)
}

func cleanupCreatedPackJournal(root *os.Root, created os.FileInfo) error {
	if created == nil {
		return fmt.Errorf("cannot prove failed pack journal creation path identity; preserving it")
	}
	live, err := root.Lstat(packJournalName)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !os.SameFile(created, live) || !live.Mode().IsRegular() || live.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("failed pack journal path was replaced; preserving replacement")
	}
	if err := root.Remove(packJournalName); err != nil {
		return err
	}
	return syncPackDir(root)
}

func appendPackPhase(root *os.Root, phase string) error {
	if phase != "parking" && phase != "installing" && phase != "committed" {
		return fmt.Errorf("invalid pack transaction phase %q", phase)
	}
	body, err := encodePackRecord(packPhaseRecord{Phase: phase})
	if err != nil {
		return err
	}
	return appendPackRecord(root, body)
}

func appendPackStaged(root *os.Root, entries []packStagedEntry) error {
	record := packStagedRecord{Phase: "staged", Entries: entries}
	body, err := encodePackRecord(record)
	if err != nil {
		return err
	}
	return appendPackRecord(root, body)
}

func appendPackStage(root *os.Root, entry packStagedEntry) error {
	record := packStageRecord{Phase: "stage", Target: entry.Target, After: entry.After}
	body, err := encodePackRecord(record)
	if err != nil {
		return err
	}
	return appendPackRecord(root, body)
}

func appendPackRecord(root *os.Root, body []byte) error {
	before, err := capturePackJournalFile(root)
	if err != nil {
		return err
	}
	if len(before.body)+len(body) > packJournalMax {
		return fmt.Errorf("pack transaction journal exceeds %d bytes", packJournalMax)
	}
	f, err := root.OpenFile(packJournalName, os.O_RDWR|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	opened, statErr := f.Stat()
	if statErr != nil || !os.SameFile(before.info, opened) || !samePackJournalMetadata(before.info, opened) {
		return errors.Join(statErr, f.Close(), fmt.Errorf("pack transaction journal changed while opening"))
	}
	if err := packJournalPoint("append-after-open"); err != nil {
		return errors.Join(err, f.Close())
	}
	if err := before.revalidate(root); err != nil {
		return errors.Join(fmt.Errorf("pack transaction journal path changed after append open: %w", err), f.Close())
	}
	written, writeErr := f.Write(body)
	if writeErr == nil && written != len(body) {
		writeErr = io.ErrShortWrite
	}
	syncErr := f.Sync()
	if err := packJournalPoint("append-after-sync"); err != nil {
		return errors.Join(writeErr, syncErr, err, f.Close())
	}
	_, seekErr := f.Seek(0, io.SeekStart)
	got, readErr := io.ReadAll(io.LimitReader(f, packJournalMax+1))
	heldAfter, statAfterErr := f.Stat()
	liveAfter, pathErr := root.Lstat(packJournalName)
	closeErr := f.Close()
	if err := errors.Join(writeErr, syncErr, seekErr, readErr, statAfterErr, pathErr, closeErr); err != nil {
		return err
	}
	want := append(append([]byte(nil), before.body...), body...)
	if !bytes.Equal(got, want) || heldAfter.Size() != int64(len(want)) || heldAfter.Mode() != before.info.Mode() || !os.SameFile(before.info, heldAfter) || !os.SameFile(heldAfter, liveAfter) || !samePackJournalMetadata(heldAfter, liveAfter) {
		return fmt.Errorf("pack transaction journal append lost live path authority or wrote unexpected bytes")
	}
	if err := syncPackDir(root); err != nil {
		return err
	}
	if err := packJournalPoint("append-after-directory-sync"); err != nil {
		return err
	}
	after, err := capturePackJournalFile(root)
	if err != nil {
		return err
	}
	if !os.SameFile(heldAfter, after.info) || !bytes.Equal(after.body, want) || after.info.Size() != int64(len(want)) {
		return fmt.Errorf("pack transaction journal changed after durable append")
	}
	return nil
}

func decodePackJSON(line []byte, dst any) error {
	if err := rejectPackDuplicateKeys(line); err != nil {
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

func requirePackJSONKeys(line []byte, allowed ...string) (map[string]json.RawMessage, error) {
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

func decodePackHeader(line []byte, header *packJournalHeader) error {
	if err := rejectPackDuplicateKeys(line); err != nil {
		return err
	}
	object, err := requirePackJSONKeys(line, "version", "phase", "entries")
	if err != nil {
		return err
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(object["entries"], &entries); err != nil {
		return err
	}
	for _, entry := range entries {
		if _, err := requirePackJSONKeys(entry, "target", "stage", "backup", "retire", "existed", "before", "after_tree"); err != nil {
			return err
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(entry, &fields); err != nil {
			return err
		}
		if _, err := requirePackJSONKeys(fields["before"], "tree", "identity"); err != nil {
			return err
		}
	}
	return decodePackJSON(line, header)
}

func decodePackPhase(line []byte, phase *packPhaseRecord) error {
	if err := rejectPackDuplicateKeys(line); err != nil {
		return err
	}
	if _, err := requirePackJSONKeys(line, "phase"); err != nil {
		return err
	}
	return decodePackJSON(line, phase)
}

func decodePackStaged(line []byte, staged *packStagedRecord) error {
	if err := rejectPackDuplicateKeys(line); err != nil {
		return err
	}
	object, err := requirePackJSONKeys(line, "phase", "entries")
	if err != nil {
		return err
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(object["entries"], &entries); err != nil {
		return err
	}
	for _, entry := range entries {
		fields, err := requirePackJSONKeys(entry, "target", "after")
		if err != nil {
			return err
		}
		if _, err := requirePackJSONKeys(fields["after"], "tree", "identity"); err != nil {
			return err
		}
	}
	return decodePackJSON(line, staged)
}

func decodePackStage(line []byte, staged *packStageRecord) error {
	if err := rejectPackDuplicateKeys(line); err != nil {
		return err
	}
	object, err := requirePackJSONKeys(line, "phase", "target", "after")
	if err != nil {
		return err
	}
	if _, err := requirePackJSONKeys(object["after"], "tree", "identity"); err != nil {
		return err
	}
	return decodePackJSON(line, staged)
}

func rejectPackDuplicateKeys(line []byte) error {
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

func readPackJournalAt(root *os.Root, name string) (packJournalHeader, string, packJournalFileState, error) {
	authority, err := capturePackJournalFileAt(root, name)
	if err != nil {
		return packJournalHeader{}, "", packJournalFileState{}, err
	}
	body := authority.body
	complete := body
	fragment := []byte(nil)
	if body[len(body)-1] != '\n' {
		if cut := bytes.LastIndexByte(body, '\n'); cut >= 0 {
			complete, fragment = body[:cut+1], body[cut+1:]
		} else {
			return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction journal header is incomplete")
		}
	}
	lines := bytes.Split(bytes.TrimSuffix(complete, []byte{'\n'}), []byte{'\n'})
	var header packJournalHeader
	if len(lines) == 0 || decodePackHeader(lines[0], &header) != nil {
		return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction journal header is malformed")
	}
	phases := []string{"prepared"}
	var staged packStagedRecord
	nextStage := 0
	for _, line := range lines[1:] {
		var rec packPhaseRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction journal phase is malformed: %w", err)
		}
		if rec.Phase == "stage" {
			if len(phases) != 1 {
				return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction stage witness appears after staging completed")
			}
			var stage packStageRecord
			if err := decodePackStage(line, &stage); err != nil {
				return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction stage witness is malformed: %w", err)
			}
			for nextStage < len(header.Entries) && packEntryDeletes(header.Entries[nextStage]) {
				nextStage++
			}
			if nextStage >= len(header.Entries) || stage.Target != header.Entries[nextStage].Target || stage.After.Tree != header.Entries[nextStage].AfterTree || !validPackTreeWitness(stage.After, false) {
				return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction stage witness does not match the next target")
			}
			header.Entries[nextStage].after = stage.After
			nextStage++
			continue
		}
		if rec.Phase == "staged" {
			if err := decodePackStaged(line, &staged); err != nil {
				return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction staged witness is malformed: %w", err)
			}
		} else if err := decodePackPhase(line, &rec); err != nil {
			return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction journal phase is malformed: %w", err)
		}
		phases = append(phases, rec.Phase)
	}
	want := []string{"prepared", "staged", "parking", "installing", "committed"}
	if len(phases) > len(want) {
		return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction journal has too many phase records")
	}
	for i, phase := range phases {
		if phase != want[i] {
			return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction journal phase sequence is invalid")
		}
	}
	if len(fragment) > 0 {
		if len(phases) == len(want) {
			return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction journal has trailing data")
		}
		nextPhase := want[len(phases)]
		validFragment := false
		if nextPhase == "staged" {
			for _, prefix := range [][]byte{[]byte(`{"phase":"stage"`), []byte(`{"phase":"staged"`)} {
				validFragment = validFragment || bytes.HasPrefix(prefix, fragment) || bytes.HasPrefix(fragment, prefix)
			}
		} else {
			next, _ := encodePackRecord(packPhaseRecord{Phase: nextPhase})
			validFragment = bytes.HasPrefix(next, fragment)
		}
		if !validFragment {
			return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction journal has malformed trailing data")
		}
	}
	phase := phases[len(phases)-1]
	if phase != "prepared" {
		if len(staged.Entries) != len(header.Entries) {
			return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction staged witness inventory is incomplete")
		}
		for i := range header.Entries {
			if staged.Entries[i].Target != header.Entries[i].Target || staged.Entries[i].After.Tree != header.Entries[i].AfterTree || !validPackTreeWitness(staged.Entries[i].After, true) {
				return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction staged witness does not match target %q", header.Entries[i].Target)
			}
			if validPackTreeWitness(header.Entries[i].after, true) && header.Entries[i].after != staged.Entries[i].After {
				return packJournalHeader{}, "", packJournalFileState{}, fmt.Errorf("pack transaction staged witness contradicts durable stage witness for target %q", header.Entries[i].Target)
			}
			header.Entries[i].after = staged.Entries[i].After
		}
	}
	if err := validatePackJournal(header, phase); err != nil {
		return packJournalHeader{}, "", packJournalFileState{}, err
	}
	return header, phase, authority, nil
}

func validatePackJournal(header packJournalHeader, phase string) error {
	if header.Version != 2 || header.Phase != "prepared" || len(header.Entries) == 0 {
		return fmt.Errorf("pack transaction journal header is invalid")
	}
	if phase != "prepared" && phase != "staged" && phase != "parking" && phase != "installing" && phase != "committed" {
		return fmt.Errorf("pack transaction journal phase %q is invalid", phase)
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
			return fmt.Errorf("pack transaction journal target is unsafe: %w", err)
		}
		if !strings.HasSuffix(entry.Target, ".pack") || validateSubsystemID(strings.TrimSuffix(entry.Target, ".pack")) != nil {
			return fmt.Errorf("pack transaction journal target %q is not a portable subsystem pack", entry.Target)
		}
		if (entry.Stage != packScratchName("stage", entry.Target) && entry.Stage != packScratchName("stage-delete", entry.Target)) || entry.Backup != packScratchName("backup", entry.Target) || entry.Retire != packScratchName("retire", entry.Target) {
			return fmt.Errorf("pack transaction journal scratch path does not match target %q", entry.Target)
		}
		if !validPackTreeWitness(entry.Before, true) || !validPackTreeDigest(entry.AfterTree) || entry.Existed != (entry.Before.Tree != packAbsentTree) || packEntryDeletes(entry) != (entry.AfterTree == packAbsentTree) {
			return fmt.Errorf("pack transaction journal tree witnesses are invalid for target %q", entry.Target)
		}
		for _, name := range []string{entry.Target, entry.Stage, entry.Backup, entry.Retire} {
			if filepath.Base(name) != name || name == packJournalName {
				return fmt.Errorf("pack transaction journal contains unsafe path %q", name)
			}
			fold := strings.ToLower(name)
			if prior, ok := folded[fold]; ok {
				return fmt.Errorf("pack transaction journal paths %q and %q alias on case-insensitive filesystems", prior, name)
			}
			folded[fold] = name
		}
	}
	if !ordered {
		return fmt.Errorf("pack transaction journal targets are not in deterministic order")
	}
	return nil
}

type packRecoveryState struct {
	entry                          packJournalEntry
	targetExists, stageExists      bool
	backupExists, retirementExists bool
	targetInstalled                bool
}

func packDirectoryState(root *os.Root, name, label string) (bool, error) {
	info, err := root.Lstat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return false, fmt.Errorf("pack transaction %s must be a real non-symlink directory", label)
	}
	return true, nil
}

type packRootRename func(*os.Root, string, string) error

func renamePackRoot(root *os.Root, oldname, newname string) error {
	return root.Rename(oldname, newname)
}

func captureExpectedPackTree(root *os.Root, name string, expected packTreeWitness, label string) (*packTreeState, error) {
	state, err := capturePackTree(root, name)
	if err != nil {
		return nil, fmt.Errorf("verify %s: %w", label, err)
	}
	if state.witness != expected {
		return nil, fmt.Errorf("%s changed after the transaction witness was recorded; preserving it", label)
	}
	return state, nil
}

func isolateAndRemovePackTree(root *os.Root, rename packRootRename, name, retirement string, expected packTreeWitness, label string) error {
	state, err := captureExpectedPackTree(root, name, expected, label)
	if err != nil {
		return err
	}
	if _, err := root.Lstat(retirement); !os.IsNotExist(err) {
		if err == nil {
			return fmt.Errorf("cannot retire %s: reserved retirement directory already exists", label)
		}
		return err
	}
	if err := state.revalidateAt(root, name); err != nil {
		return fmt.Errorf("%s changed before retirement; preserving it: %w", label, err)
	}
	if err := rename(root, name, retirement); err != nil {
		return err
	}
	if err := syncPackDir(root); err != nil {
		return err
	}
	if err := state.revalidateAt(root, retirement); err != nil {
		restoreErr := rename(root, retirement, name)
		return errors.Join(fmt.Errorf("%s changed at retirement boundary; preserving it: %w", label, err), restoreErr)
	}
	if err := packTreeRemovalPoint(name, retirement); err != nil {
		return err
	}
	if err := state.revalidateAt(root, retirement); err != nil {
		restoreErr := rename(root, retirement, name)
		return errors.Join(fmt.Errorf("%s changed at deletion boundary; preserving it: %w", label, err), restoreErr)
	}
	if err := root.RemoveAll(retirement); err != nil {
		return err
	}
	return syncPackDir(root)
}

func restorePackBackup(root *os.Root, rename packRootRename, entry packJournalEntry) error {
	state, err := captureExpectedPackTree(root, entry.Backup, entry.Before, "parked backup "+entry.Target)
	if err != nil {
		return err
	}
	if _, err := root.Lstat(entry.Target); !os.IsNotExist(err) {
		if err == nil {
			return fmt.Errorf("refuse to overwrite live target %s while restoring its backup", entry.Target)
		}
		return err
	}
	if err := state.revalidateAt(root, entry.Backup); err != nil {
		return fmt.Errorf("parked backup %s changed before restoration; preserving it: %w", entry.Target, err)
	}
	if err := rename(root, entry.Backup, entry.Target); err != nil {
		return err
	}
	if err := syncPackDir(root); err != nil {
		return err
	}
	if err := state.revalidateAt(root, entry.Target); err != nil {
		restoreErr := rename(root, entry.Target, entry.Backup)
		return errors.Join(fmt.Errorf("restored target %s changed at restoration boundary; preserving backup authority: %w", entry.Target, err), restoreErr)
	}
	return nil
}

func samePackJournalAcrossRename(before, after packJournalFileState) bool {
	return os.SameFile(before.info, after.info) && before.info.Mode() == after.info.Mode() && before.info.Size() == after.info.Size() && before.info.ModTime().Equal(after.info.ModTime()) && bytes.Equal(before.body, after.body)
}

func isolatePackRecoveryJournal(root *os.Root, rename packRootRename, authority packJournalFileState) (packJournalFileState, error) {
	if _, err := root.Lstat(packJournalRetirement); !os.IsNotExist(err) {
		if err == nil {
			return packJournalFileState{}, fmt.Errorf("reserved pack recovery journal already exists; preserving both journal paths")
		}
		return packJournalFileState{}, err
	}
	if err := authority.revalidate(root); err != nil {
		return packJournalFileState{}, fmt.Errorf("pack recovery journal changed before isolation; preserving it: %w", err)
	}
	if err := packJournalPoint("recovery-before-journal-isolate"); err != nil {
		return packJournalFileState{}, err
	}
	if err := authority.revalidate(root); err != nil {
		return packJournalFileState{}, fmt.Errorf("pack recovery journal changed at isolation boundary; preserving it: %w", err)
	}
	if err := rename(root, packJournalName, packJournalRetirement); err != nil {
		return packJournalFileState{}, err
	}
	if err := syncPackDir(root); err != nil {
		return packJournalFileState{}, err
	}
	if err := packJournalPoint("recovery-after-journal-isolate"); err != nil {
		return packJournalFileState{}, err
	}
	isolated, err := capturePackJournalFileAt(root, packJournalRetirement)
	if err != nil || !samePackJournalAcrossRename(authority, isolated) {
		var restoreErr error
		if _, liveErr := root.Lstat(packJournalName); os.IsNotExist(liveErr) {
			restoreErr = rename(root, packJournalRetirement, packJournalName)
		} else if liveErr == nil {
			restoreErr = fmt.Errorf("cannot restore changed isolated journal over a replacement; preserving both")
		} else {
			restoreErr = liveErr
		}
		if err == nil {
			err = fmt.Errorf("identity, content, mode, size, or modification time changed")
		}
		return packJournalFileState{}, errors.Join(fmt.Errorf("pack recovery journal changed during isolation; preserving it: %w", err), restoreErr)
	}
	if _, err := root.Lstat(packJournalName); !os.IsNotExist(err) {
		if err == nil {
			return packJournalFileState{}, fmt.Errorf("pack recovery journal path was replaced during isolation; preserving both authorities")
		}
		return packJournalFileState{}, err
	}
	return isolated, nil
}

func requirePackRecoveryJournal(root *os.Root, authority packJournalFileState, point string) error {
	if err := packJournalPoint("recovery-before-" + point); err != nil {
		return err
	}
	if _, err := root.Lstat(packJournalName); !os.IsNotExist(err) {
		if err == nil {
			return fmt.Errorf("pack recovery journal path reappeared before %s; preserving live state", point)
		}
		return err
	}
	if err := authority.revalidateAt(root, packJournalRetirement); err != nil {
		return fmt.Errorf("isolated pack recovery journal changed before %s; preserving live state: %w", point, err)
	}
	return nil
}

func restorePackRecoveryJournal(root *os.Root, rename packRootRename, authority packJournalFileState) error {
	if err := authority.revalidateAt(root, packJournalRetirement); err != nil {
		return fmt.Errorf("cannot restore changed isolated pack recovery journal; preserving it: %w", err)
	}
	if _, err := root.Lstat(packJournalName); !os.IsNotExist(err) {
		if err == nil {
			return fmt.Errorf("cannot restore pack recovery journal over a replacement; preserving both authorities")
		}
		return err
	}
	if err := rename(root, packJournalRetirement, packJournalName); err != nil {
		return err
	}
	if err := syncPackDir(root); err != nil {
		return err
	}
	restored, err := capturePackJournalFile(root)
	if err != nil {
		return err
	}
	if !samePackJournalAcrossRename(authority, restored) {
		return fmt.Errorf("restored pack recovery journal lost exact authority")
	}
	return nil
}

func removePackRecoveryJournal(root *os.Root, authority packJournalFileState) (bool, error) {
	if err := requirePackRecoveryJournal(root, authority, "journal-remove"); err != nil {
		return false, err
	}
	if err := root.Remove(packJournalRetirement); err != nil {
		return false, err
	}
	return true, syncPackDir(root)
}

func recoverPackTransaction(root *os.Root, rename packRootRename) (retErr error) {
	_, journalErr := root.Lstat(packJournalName)
	_, retirementErr := root.Lstat(packJournalRetirement)
	journalExists := journalErr == nil
	retirementExists := retirementErr == nil
	if journalErr != nil && !os.IsNotExist(journalErr) {
		return journalErr
	}
	if retirementErr != nil && !os.IsNotExist(retirementErr) {
		return retirementErr
	}
	if journalExists && retirementExists {
		return fmt.Errorf("pack transaction has both live and isolated recovery journals; preserving both")
	}
	if !journalExists && !retirementExists {
		return nil
	}
	journalPath := packJournalName
	if retirementExists {
		journalPath = packJournalRetirement
	}
	header, phase, authority, err := readPackJournalAt(root, journalPath)
	if err != nil {
		return err
	}
	if err := packJournalPoint("recovery-after-journal-read"); err != nil {
		return err
	}
	if journalPath == packJournalName {
		authority, err = isolatePackRecoveryJournal(root, rename, authority)
		if err != nil {
			return err
		}
	} else if err := requirePackRecoveryJournal(root, authority, "resume-isolated"); err != nil {
		return err
	}
	isolated := true
	defer func() {
		if retErr != nil && isolated {
			retErr = errors.Join(retErr, restorePackRecoveryJournal(root, rename, authority))
		}
	}()
	states := make([]packRecoveryState, 0, len(header.Entries))
	for _, entry := range header.Entries {
		state := packRecoveryState{entry: entry}
		if state.targetExists, err = packDirectoryState(root, entry.Target, "target "+entry.Target); err != nil {
			return err
		}
		if state.stageExists, err = packDirectoryState(root, entry.Stage, "stage "+entry.Stage); err != nil {
			return err
		}
		if state.backupExists, err = packDirectoryState(root, entry.Backup, "backup "+entry.Backup); err != nil {
			return err
		}
		if state.retirementExists, err = packDirectoryState(root, entry.Retire, "retirement "+entry.Retire); err != nil {
			return err
		}
		if phase == "committed" && packEntryDeletes(entry) == state.targetExists {
			if packEntryDeletes(entry) {
				return fmt.Errorf("committed pack transaction retained deleted target %s", entry.Target)
			}
			return fmt.Errorf("committed pack transaction is missing target %s", entry.Target)
		}
		if phase != "committed" && entry.Existed && !state.targetExists && !state.backupExists {
			return fmt.Errorf("uncommitted pack transaction lost both target and backup for %s", entry.Target)
		}
		if phase != "committed" && !entry.Existed && state.backupExists {
			return fmt.Errorf("uncommitted pack transaction has an impossible backup for new target %s", entry.Target)
		}
		states = append(states, state)
	}
	// Validate the entire live authority before mutating any entry. A recovery
	// blocker must preserve every user-visible target and every usable backup.
	for i := range states {
		state := &states[i]
		entry := state.entry
		if state.stageExists {
			if packEntryDeletes(entry) {
				return fmt.Errorf("pack deletion entry %s has an unexpected stage; preserving it", entry.Target)
			}
			if !validPackTreeWitness(entry.after, false) {
				return fmt.Errorf("staged target %s has no durable exact witness; preserving it", entry.Target)
			}
			if _, err := captureExpectedPackTree(root, entry.Stage, entry.after, "staged target "+entry.Target); err != nil {
				return err
			}
			if phase == "committed" {
				return fmt.Errorf("committed pack transaction has an unexpected stage for %s; preserving it", entry.Target)
			}
		}
		if phase == "committed" {
			if packEntryDeletes(entry) {
				if state.targetExists {
					return fmt.Errorf("committed pack transaction retained deleted target %s", entry.Target)
				}
			} else if !state.targetExists {
				return fmt.Errorf("committed pack transaction is missing target %s", entry.Target)
			} else if _, err := captureExpectedPackTree(root, entry.Target, entry.after, "installed target "+entry.Target); err != nil {
				return err
			}
			if state.backupExists {
				if _, err := captureExpectedPackTree(root, entry.Backup, entry.Before, "parked backup "+entry.Target); err != nil {
					return err
				}
			}
			if state.retirementExists {
				if _, err := captureExpectedPackTree(root, entry.Retire, entry.Before, "retiring backup "+entry.Target); err != nil {
					return err
				}
			}
			continue
		}
		if state.retirementExists {
			if _, err := captureExpectedPackTree(root, entry.Retire, entry.after, "retiring installed target "+entry.Target); err != nil {
				return err
			}
		}
		if state.backupExists {
			if _, err := captureExpectedPackTree(root, entry.Backup, entry.Before, "parked backup "+entry.Target); err != nil {
				return err
			}
		}
		if phase == "installing" && state.targetExists {
			if entry.Existed && !state.backupExists {
				if _, err := captureExpectedPackTree(root, entry.Target, entry.Before, "already restored target "+entry.Target); err != nil {
					return err
				}
			} else {
				if packEntryDeletes(entry) {
					return fmt.Errorf("uncommitted pack deletion has a concurrent target %s; preserving it", entry.Target)
				}
				if _, err := captureExpectedPackTree(root, entry.Target, entry.after, "installed target "+entry.Target); err != nil {
					return err
				}
				state.targetInstalled = true
			}
		} else if !entry.Existed && state.targetExists {
			return fmt.Errorf("uncommitted pack transaction has a concurrent new target %s; preserving it", entry.Target)
		} else if state.backupExists && state.targetExists {
			return fmt.Errorf("uncommitted pack transaction has both target and backup for %s before installation; preserving both", entry.Target)
		}
	}
	if phase == "committed" {
		for _, state := range states {
			if state.backupExists {
				if err := requirePackRecoveryJournal(root, authority, "remove-backup-"+state.entry.Target); err != nil {
					return err
				}
				if err := isolateAndRemovePackTree(root, rename, state.entry.Backup, state.entry.Retire, state.entry.Before, "parked backup "+state.entry.Target); err != nil {
					return err
				}
			} else if state.retirementExists {
				if err := requirePackRecoveryJournal(root, authority, "remove-retiring-backup-"+state.entry.Target); err != nil {
					return err
				}
				if err := isolateAndRemovePackTree(root, rename, state.entry.Retire, state.entry.Backup, state.entry.Before, "retiring backup "+state.entry.Target); err != nil {
					return err
				}
			}
		}
	} else {
		for i := len(states) - 1; i >= 0; i-- {
			state := states[i]
			if state.targetInstalled {
				if err := requirePackRecoveryJournal(root, authority, "remove-installed-"+state.entry.Target); err != nil {
					return err
				}
				if err := isolateAndRemovePackTree(root, rename, state.entry.Target, state.entry.Retire, state.entry.after, "installed target "+state.entry.Target); err != nil {
					return err
				}
			}
			if state.retirementExists {
				if err := requirePackRecoveryJournal(root, authority, "remove-retiring-installed-"+state.entry.Target); err != nil {
					return err
				}
				if err := isolateAndRemovePackTree(root, rename, state.entry.Retire, state.entry.Stage, state.entry.after, "retiring installed target "+state.entry.Target); err != nil {
					return err
				}
			}
			if state.entry.Existed && state.backupExists {
				if err := requirePackRecoveryJournal(root, authority, "restore-backup-"+state.entry.Target); err != nil {
					return err
				}
				if err := restorePackBackup(root, rename, state.entry); err != nil {
					return err
				}
			}
			if state.stageExists {
				if err := requirePackRecoveryJournal(root, authority, "remove-stage-"+state.entry.Target); err != nil {
					return err
				}
				if err := isolateAndRemovePackTree(root, rename, state.entry.Stage, state.entry.Retire, state.entry.after, "staged target "+state.entry.Target); err != nil {
					return err
				}
			}
		}
	}
	removed, err := removePackRecoveryJournal(root, authority)
	if removed {
		isolated = false
	}
	return err
}
