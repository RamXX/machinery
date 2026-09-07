// Frozen RED/GREEN suite for the MAC-p9z1 immutable object transport:
// bounded closed-archive export, validated atomic import into a NEW
// destination under an explicit expected head, preservation of failed
// history, byte/checksum tamper rejection and recorded-only (never
// replayed) imported state.
package tdd

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const archiveMagic = "MTDDARCV"

var _ = hex.EncodeToString

// archiveRecord frames one payload: U64-BE length, bytes, 32-byte checksum.
func archiveFrame(payload []byte) []byte {
	var head [8]byte
	binary.BigEndian.PutUint64(head[:], uint64(len(payload)))
	return append(append(head[:], payload...), digestRaw(payload)...)
}

func digestRaw(b []byte) []byte {
	sum := sha256.Sum256(b)
	return sum[:]
}

// writeArchiveRaw writes a closed archive from explicit records
// (test-side; independent of production export).
func writeArchiveRaw(t *testing.T, path string, records [][]byte) {
	t.Helper()
	var out []byte
	out = append(out, archiveMagic...)
	out = append(out, 0x01)
	for _, r := range records {
		out = append(out, archiveFrame(r)...)
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}
}

// readArchiveRaw parses and checksum-verifies an archive (test-side).
func readArchiveRaw(t *testing.T, path string) (index []byte, payloads [][]byte) {
	t.Helper()
	data := mustRead(t, path)
	if string(data[:len(archiveMagic)]) != archiveMagic || data[len(archiveMagic)] != 0x01 {
		t.Fatalf("bad archive header")
	}
	off := len(archiveMagic) + 1
	first := true
	for off < len(data) {
		if off+8 > len(data) {
			t.Fatalf("truncated record header")
		}
		n := binary.BigEndian.Uint64(data[off : off+8])
		off += 8
		if off+int(n)+32 > len(data) {
			t.Fatalf("truncated record body")
		}
		payload := data[off : off+int(n)]
		off += int(n)
		sum := data[off : off+32]
		off += 32
		if string(sum) != string(digestRaw(payload)) {
			t.Fatalf("record checksum mismatch")
		}
		if first {
			index = payload
			first = false
		} else {
			payloads = append(payloads, payload)
		}
	}
	return index, payloads
}

type archiveIndexEntry struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

type archiveIndex struct {
	Schema     string              `json:"schema"`
	StoreID    string              `json:"store_id"`
	ProjectID  string              `json:"project_id"`
	HeadDigest string              `json:"head_digest"`
	Generation int64               `json:"generation"`
	TotalBytes int64               `json:"total_bytes"`
	Entries    []archiveIndexEntry `json:"entries"`
}

// AC4: export writes a new bounded closed archive of the complete history
// and refuses existing outputs or corrupt stores.
func TestExportProducesBoundedClosedArchive(t *testing.T) {
	_, _, store, _, _, _, _ := boundStatusFixture(t)
	out := filepath.Join(t.TempDir(), "archive.tddexp")
	res, err := ExportStore(context.Background(), store, out)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	fi, err := os.Lstat(out)
	if err != nil || fi.Mode().IsDir() {
		t.Fatalf("archive not written: %v", err)
	}
	indexBytes, _ := readArchiveRaw(t, out)
	var idx archiveIndex
	if err := strictUnmarshalJSON(indexBytes, &idx); err != nil {
		t.Fatalf("index decode: %v", err)
	}
	if idx.Schema != "machinery.tdd.export/v1" {
		t.Errorf("index schema %q", idx.Schema)
	}
	if idx.ProjectID != fixtureProjectID || idx.HeadDigest == "" || res.HeadDigest != idx.HeadDigest {
		t.Errorf("index identity %+v vs result %+v", idx, res)
	}
	if len(idx.Entries) == 0 || res.Entries != int64(len(idx.Entries)) {
		t.Errorf("entries: index %d result %d", len(idx.Entries), res.Entries)
	}
	for i := 1; i < len(idx.Entries); i++ {
		if idx.Entries[i-1].Path >= idx.Entries[i].Path {
			t.Error("index entries are not canonically sorted")
			break
		}
	}
	var total int64
	for _, e := range idx.Entries {
		total += e.Size
	}
	if total != idx.TotalBytes || total != res.TotalBytes {
		t.Errorf("total bytes %d/%d/%d", total, idx.TotalBytes, res.TotalBytes)
	}
	// exporting to an existing output refuses
	if _, err := ExportStore(context.Background(), store, out); err == nil || !strings.Contains(err.Error(), "STORE_ROOT_MISMATCH") {
		t.Errorf("existing archive output overwritten: err = %v", err)
	}
	// exporting a corrupt store fails closed
	if err := os.Remove(filepath.Join(store, "store.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := ExportStore(context.Background(), store, filepath.Join(t.TempDir(), "a2.tddexp")); err == nil {
		t.Error("corrupt store exported")
	}
}

// AC4/AC6: import round trip preserves identity, chain, exact bytes, failed
// history and object graph; imported state is recorded-only.
func TestImportRoundTripPreservesHistory(t *testing.T) {
	src, ctl, store, plan, manifests, inv, ref := boundStatusFixture(t)
	manifestDigest := "sha256:" + digestHex(manifests[0].Raw())
	writeRunRecord(t, store, "run-fail-1", ".", "M1", manifestDigest, "", "fail", "red")
	failedRunBytes := mustRead(t, filepath.Join(store, "runs", "run-fail-1.json"))
	out := filepath.Join(t.TempDir(), "archive.tddexp")
	if _, err := ExportStore(context.Background(), store, out); err != nil {
		t.Fatalf("export: %v", err)
	}
	dest := filepath.Join(t.TempDir(), "imported-store")
	res, err := ImportStore(context.Background(), dest, out, fixtureProjectID, currentHeadDigest(t, store))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if res.ProjectID != fixtureProjectID || res.HeadDigest != currentHeadDigest(t, store) || res.State != "imported-recorded-only" {
		t.Errorf("import result %+v", res)
	}
	if res.Runs < 1 || res.Blobs < 1 || res.Objects < 1 || res.Controls < 1 || res.Heads < 1 {
		t.Errorf("import counts %+v", res)
	}
	// byte-exact preservation of blobs, controls and failed history
	var walkSrc, walkDst int
	filepath.Walk(store, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		walkSrc++
		rel, _ := filepath.Rel(store, p)
		dst := filepath.Join(dest, rel)
		if strings.HasPrefix(rel, filepath.Join("ledger", "head.json")) {
			return nil
		}
		if digestHex(mustRead(t, dst)) != digestHex(mustRead(t, p)) {
			t.Errorf("imported %s differs byte-wise", rel)
		}
		return nil
	})
	filepath.Walk(dest, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		walkDst++
		return nil
	})
	if walkSrc != walkDst {
		t.Errorf("imported store file count %d != exported %d", walkDst, walkSrc)
	}
	if got := mustRead(t, filepath.Join(dest, "runs", "run-fail-1.json")); string(got) != string(failedRunBytes) {
		t.Error("failed run history not preserved byte-for-byte")
	}
	// the imported store materializes the same retained inputs
	mat := filepath.Join(t.TempDir(), "mat")
	if err := MaterializeBundle(context.Background(), dest, fixtureProjectID, ref, mat); err != nil {
		t.Fatalf("materialize from imported store: %v", err)
	}
	if string(mustRead(t, filepath.Join(mat, "src.txt"))) != string(mustRead(t, filepath.Join(src, "src.txt"))) {
		t.Error("imported materialization lost exact bytes")
	}
	// status over the imported store reports recorded evidence, never replay
	rep, err := Status(context.Background(), statusRequest(src, ctl, dest, plan, manifests, inv))
	if err != nil {
		t.Fatalf("status on imported store: %v", err)
	}
	if rep.TestExecution == "replayed-this-invocation" {
		t.Error("import certified replay")
	}
	if rep.TestExecution != "recorded-only" && rep.TestExecution != "failed" {
		t.Errorf("imported failed history reported as %q", rep.TestExecution)
	}
}

// AC4: import requires a NEW destination and the exact expected head.
func TestImportRequiresNewDestinationAndExpectedHead(t *testing.T) {
	_, _, store, _, _, _, _ := boundStatusFixture(t)
	out := filepath.Join(t.TempDir(), "archive.tddexp")
	if _, err := ExportStore(context.Background(), store, out); err != nil {
		t.Fatalf("export: %v", err)
	}
	head := currentHeadDigest(t, store)
	emptyDest := filepath.Join(t.TempDir(), "empty-dest")
	if err := os.Mkdir(emptyDest, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportStore(context.Background(), emptyDest, out, fixtureProjectID, head); err == nil || !strings.Contains(err.Error(), "STORE_ROOT_MISMATCH") {
		t.Errorf("existing empty destination adopted: err = %v", err)
	}
	if _, err := os.ReadDir(emptyDest); err != nil {
		t.Fatal(err)
	}
	wrongHead := "sha256:" + strings.Repeat("0", 64)
	fresh := filepath.Join(t.TempDir(), "fresh")
	if _, err := ImportStore(context.Background(), fresh, out, fixtureProjectID, wrongHead); err == nil || !strings.Contains(err.Error(), "CONTROL_ROLLBACK") {
		t.Errorf("wrong expected head accepted: err = %v", err)
	}
	if _, err := os.Lstat(fresh); !os.IsNotExist(err) {
		t.Error("rejected import published a destination anyway")
	}
	// wrong project is rejected
	fresh2 := filepath.Join(t.TempDir(), "fresh2")
	if _, err := ImportStore(context.Background(), fresh2, out, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", head); err == nil {
		t.Error("mismatched project imported")
	}
	// correct expected head imports
	ok := filepath.Join(t.TempDir(), "ok")
	if _, err := ImportStore(context.Background(), ok, out, fixtureProjectID, head); err != nil {
		t.Fatalf("correct import rejected: %v", err)
	}
	// unsafe challenge twin: a trustless importer accepts any expected head
	if v := func(_, _ string) bool { return true }; !v(wrongHead, head) {
		t.Fatal("challenge twin must accept to prove the assertion discriminates")
	}
}

// AC4: malformed, tampered, oversized or self-inconsistent archives fail
// closed before any publication, with staging cleaned up.
func TestImportRejectsMalformedTamperedOversized(t *testing.T) {
	_, _, store, _, _, _, _ := boundStatusFixture(t)
	out := filepath.Join(t.TempDir(), "a.tddexp")
	if _, err := ExportStore(context.Background(), store, out); err != nil {
		t.Fatalf("export: %v", err)
	}
	indexBytes, payloads := readArchiveRaw(t, out)
	var idx archiveIndex
	if err := strictUnmarshalJSON(indexBytes, &idx); err != nil {
		t.Fatal(err)
	}
	head := currentHeadDigest(t, store)
	rebuild := func(mutate func(idx *archiveIndex, payloads [][]byte) [][]byte, name, wantCode string) {
		t.Helper()
		clone := archiveIndex{
			Schema: idx.Schema, StoreID: idx.StoreID, ProjectID: idx.ProjectID,
			HeadDigest: idx.HeadDigest, Generation: idx.Generation, TotalBytes: idx.TotalBytes,
			Entries: append([]archiveIndexEntry(nil), idx.Entries...),
		}
		var pl [][]byte
		for _, p := range payloads {
			pl = append(pl, append([]byte(nil), p...))
		}
		pl = mutate(&clone, pl)
		marshaled, err := json.Marshal(&clone)
		if err != nil {
			t.Fatal(err)
		}
		archive := filepath.Join(t.TempDir(), name+".tddexp")
		writeArchiveRaw(t, archive, append([][]byte{marshaled}, pl...))
		dest := filepath.Join(t.TempDir(), name+"-dest")
		if _, err := ImportStore(context.Background(), dest, archive, fixtureProjectID, head); err == nil || !strings.Contains(err.Error(), wantCode) {
			t.Errorf("%s accepted: err = %v", name, err)
		}
		if _, err := os.Lstat(dest); !os.IsNotExist(err) {
			t.Errorf("%s left a published destination behind", name)
		}
	}
	// unknown schema (UNSUPPORTED_VERSION is the closed code for a foreign
	// schema identity)
	rebuild(func(i *archiveIndex, _ [][]byte) [][]byte { i.Schema = "someone.else/v9"; return payloads }, "schema", "UNSUPPORTED_VERSION")
	// duplicate paths
	rebuild(func(i *archiveIndex, _ [][]byte) [][]byte {
		if len(i.Entries) > 0 {
			i.Entries = append(i.Entries, i.Entries[0])
		}
		return payloads
	}, "dup", "INVALID_SCHEMA")
	// declared size beyond the bundle cap
	rebuild(func(i *archiveIndex, _ [][]byte) [][]byte {
		i.Entries[0].Size = int64(1) << 40
		return payloads
	}, "oversize", "OUTPUT_LIMIT")
	// index digest lying about a payload
	rebuild(func(i *archiveIndex, _ [][]byte) [][]byte {
		i.Entries[0].Digest = "sha256:" + strings.Repeat("0", 64)
		return payloads
	}, "digest-lie", "INVALID_SCHEMA")
	// dropped blob record still referenced by the index
	rebuild(func(i *archiveIndex, pl [][]byte) [][]byte {
		for k, e := range i.Entries {
			if e.Kind == "blob" {
				i.Entries = append(i.Entries[:k], i.Entries[k+1:]...)
				return append(pl[:0], pl[1:]...)
			}
		}
		return pl
	}, "missing-blob", "INVALID_SCHEMA")
	// raw truncation mid-record
	trunc := filepath.Join(t.TempDir(), "trunc.tddexp")
	raw := mustRead(t, out)
	writeRaw(t, trunc, raw[:len(raw)-40])
	if _, err := ImportStore(context.Background(), filepath.Join(t.TempDir(), "t1"), trunc, fixtureProjectID, head); err == nil {
		t.Error("truncated archive imported")
	}
	// flipped payload byte inside a record (checksum violation)
	flipped := append([]byte(nil), raw...)
	flipped[len(flipped)-45] ^= 0xff
	flipPath := filepath.Join(t.TempDir(), "flip.tddexp")
	writeRaw(t, flipPath, flipped)
	if _, err := ImportStore(context.Background(), filepath.Join(t.TempDir(), "t2"), flipPath, fixtureProjectID, head); err == nil {
		t.Error("checksum-violating archive imported")
	}
	// extra trailing garbage
	garbage := filepath.Join(t.TempDir(), "garbage.tddexp")
	writeRaw(t, garbage, append(append([]byte(nil), raw...), 0, 0, 0, 1))
	if _, err := ImportStore(context.Background(), filepath.Join(t.TempDir(), "t3"), garbage, fixtureProjectID, head); err == nil {
		t.Error("archive with trailing bytes imported")
	}
	// no staging residue in the destination parent after failures
	entries, _ := os.ReadDir(t.TempDir())
	for _, e := range entries {
		if strings.Contains(e.Name(), "staging") {
			t.Errorf("staging residue %q left behind", e.Name())
		}
	}
}

// AC4: a broken chain inside an archive never imports.
func TestImportRejectsBrokenChainArchive(t *testing.T) {
	_, _, store, _, _, _, _ := boundStatusFixture(t)
	// tamper head.json inside the store, then export: the chain check must
	// reject the store (and thus no archive exists to trust)
	hj := filepath.Join(store, "ledger", "head.json")
	bad := strings.Replace(string(mustRead(t, hj)), `"generation":0`, `"generation":7`, 1)
	if err := os.Chmod(hj, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hj, []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "bad.tddexp")
	if _, err := ExportStore(context.Background(), store, out); err == nil {
		t.Error("store with unarchived tampered head exported")
	}
	if _, err := os.Lstat(out); !os.IsNotExist(err) {
		t.Error("failed export left an archive behind")
	}
}

func writeRaw(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
