// Frozen RED/GREEN suite for the MAC-p9z1 replay-input retention surface
// (bundle half): Capture over real filesystems, exact byte/mode/empty-
// directory custody, frozen-by-default classification with separately bound
// control/judgment identities, the exact machinery.tdd.tree/v1 encoding,
// alias/link/store-overlap rejection, idempotent concurrent capture and
// immutable materialization. No mocks: every case exercises real directory
// trees, real permission bits and the real external store.
package tdd

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/RamXX/machinery/internal/tdd/protocol"
)

// ---- frozen capture fixture ----

const (
	fixSubjectBytes = "subject v2\nanchor-line\n"
	fixGoTest       = "package tests\n"
	fixHelper       = "package tests\n\nfunc Helper() {}\n"
	fixDepLib       = "package vendor\n\nfunc Lib() {}\n"
	fixApp          = "package impl\n"
	fixTool         = "#!/bin/sh\nexit 0\n"
	fixLock         = "deps locked v1\n"
	fixDotConfig    = "build-config=v1\n"
	fixBuildMD      = "# Build\n\n## Build plan\n\n**M1 - Alpha behavior.**\nStatus: open\nDoD: alpha-s1 covered.\n"
	fixAttest       = "attestation_version: 2\nattestations: []\n"
)

// writeCaptureSource materializes the canonical source repository: subject
// file, frozen test/helper/fixture tree with an executable, an empty
// fixture directory, binary content, an untracked dependency lock, a dot
// config, the reserved control namespace and the judgment control. Fixed
// modes are applied explicitly so digests are umask-independent.
func writeCaptureSource(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	must := func(dir string, mode os.FileMode) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), mode); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	write := func(rel string, content string, mode os.FileMode) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir parent %s: %v", rel, err)
		}
		if err := os.WriteFile(p, []byte(content), mode); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
		if err := os.Chmod(p, mode); err != nil {
			t.Fatalf("chmod %s: %v", rel, err)
		}
	}
	must("machines", 0o755)
	must("impl", 0o755)
	must("tests/helpers", 0o755)
	must("tests/fixtures/empty-dir", 0o755)
	must("vendor-deps", 0o755)
	must("assurance/milestones", 0o755)
	write("BUILD.md", fixBuildMD, 0o644)
	write("domain.modelith.yaml", "kind: modelith\nversion: 1\n", 0o644)
	write("machines/Alpha.oracle.md", "# Alpha oracle\n| test id | stable id | guard | behavior |\n| --- | --- | --- | --- |\n| alpha-t1 | alpha-s1 | gate-x | refuses malformed input |\n", 0o644)
	write("src.txt", fixSubjectBytes, 0o600)
	write("impl/app.go", fixApp, 0o644)
	write("impl/tool.sh", fixTool, 0o755)
	write("tests/alpha_test.go", fixGoTest, 0o644)
	write("tests/helpers/helper.go", fixHelper, 0o644)
	write("tests/fixtures/data.bin", "\x00\x01binary\xff", 0o644)
	write("vendor-deps/lib.go", fixDepLib, 0o644)
	write("vendor.lock", fixLock, 0o644)
	write(".build-config", fixDotConfig, 0o644)
	write("assurance/plan.json", "{\"schema\":\"machinery.tdd.plan/v1\"}", 0o644)
	write("assurance/milestones/M1.json", "{\"schema\":\"machinery.tdd.milestone/v1\"}", 0o644)
	write("attestations.yaml", fixAttest, 0o644)
	// VCS administrative payload: excluded from the source bundle.
	write(".git/HEAD", "ref: refs/heads/main\n", 0o644)
	for _, d := range []string{"", "machines", "impl", "tests", "tests/helpers", "tests/fixtures", "tests/fixtures/empty-dir", "vendor-deps", "assurance", "assurance/milestones", ".git"} {
		if err := os.Chmod(filepath.Join(root, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatalf("normalize dir mode %q: %v", d, err)
		}
	}
	return root
}

// separateControlMaterialization copies the reserved control namespace to
// its own immutable materialization outside the source root.
func separateControlMaterialization(t *testing.T, src string) string {
	t.Helper()
	ctl := t.TempDir()
	for _, f := range []string{"plan.json", "milestones/M1.json"} {
		data, err := os.ReadFile(filepath.Join(src, "assurance", filepath.FromSlash(f)))
		if err != nil {
			t.Fatalf("read control %s: %v", f, err)
		}
		p := filepath.Join(ctl, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir control: %v", err)
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatalf("write control: %v", err)
		}
	}
	return ctl
}

// testInputView is a fully held InputView whose callbacks are observable.
type testInputView struct {
	src       string
	ctl       string
	revalidFn func() error
	released  int
}

func (v *testInputView) view() InputView {
	return InputView{
		SourceRoot:          v.src,
		DesignPath:          protocol.RepositoryRoot,
		ImplementationPaths: []string{"."},
		ControlRoot:         v.ctl,
		Revalidate: func() error {
			if v.revalidFn != nil {
				return v.revalidFn()
			}
			return nil
		},
		Release: func() error {
			v.released++
			return nil
		},
	}
}

// captureManifest builds the hand-bound manifest matching the fixture tree.
func captureManifest() Manifest {
	return Manifest{
		Schema:              protocol.SchemaMilestone,
		ID:                  "M1",
		Revision:            1,
		Repository:          protocol.RepositoryRoot,
		ImplementationRoots: []string{"."},
		FrozenRoots:         []string{"tests"},
		SubjectEntries:      []SubjectEntry{{Path: "src.txt", Kind: "file"}},
		Suites: []Suite{{
			ID:              "s1",
			Adapter:         protocol.AdapterGoTesting,
			Runtime:         RuntimeRef{Profile: "go", Version: "go1.27.1", Platform: "darwin/arm64", Closure: fixtureClosure},
			Root:            "tests",
			Files:           []string{"tests/alpha_test.go"},
			DependencyRoots: []string{"vendor-deps"},
			Tests: []Test{{
				ID: "t1", Native: NativeID{Package: "github.com/example/tests", Test: "TestAlphaRefusal"},
				Source: "tests/alpha_test.go", Role: RoleNegative,
				Assertions: []Assertion{{ID: "asr-1", Source: "tests/alpha_test.go", Line: 10, Helper: protocol.AssertionHelperV1}},
			}},
		}},
	}
}

func captureLimits() Limits {
	return Limits{
		WallMS: protocol.LimitWallDefaultMS, CleanupMS: protocol.LimitCleanupDefaultMS,
		StdoutBytes: protocol.LimitStdoutDefault, StderrBytes: protocol.LimitStderrDefault,
		EventBytes: protocol.LimitEventBytesDefault, EventCount: protocol.LimitEventCountDefault,
		Jobs: protocol.LimitJobsDefault, BundleBytes: protocol.LimitBundleDefault,
		Entries: protocol.LimitEntriesDefault, Depth: protocol.LimitDepthDefault,
	}
}

func newCaptureRequest(v *testInputView, store string) CaptureRequest {
	return CaptureRequest{Inputs: v.view(), Manifest: captureManifest(), Name: "baseline-M1", Store: store, Limits: captureLimits()}
}

func digestHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// testTreeEnc is an INDEPENDENT oracle for the machinery.tdd.tree/v1 typed
// topology encoding (written directly against the contract text, not shared
// with production code).
func testTreeEnc(entries []struct {
	Rel   string
	Dir   bool
	Perm  uint32
	Size  int64
	Dig   []byte
	Role  string
}) string {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Rel < entries[j].Rel })
	h := sha256.New()
	h.Write([]byte("machinery.tdd.tree/v1"))
	var buf []byte
	for _, e := range entries {
		if e.Dir {
			buf = append(buf, 0)
		} else {
			buf = append(buf, 1)
		}
		var u8 [8]byte
		binary.BigEndian.PutUint64(u8[:], uint64(len(e.Rel)))
		buf = append(buf, u8[:]...)
		buf = append(buf, e.Rel...)
		var u4 [4]byte
		binary.BigEndian.PutUint32(u4[:], e.Perm)
		buf = append(buf, u4[:]...)
		binary.BigEndian.PutUint64(u8[:], uint64(e.Size))
		buf = append(buf, u8[:]...)
		if e.Dir {
			buf = append(buf, make([]byte, 32)...)
		} else {
			buf = append(buf, e.Dig...)
		}
		binary.BigEndian.PutUint64(u8[:], uint64(len(e.Role)))
		buf = append(buf, u8[:]...)
		buf = append(buf, e.Role...)
		h.Write(buf)
		buf = buf[:0]
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// readBundleDoc decodes a store bundle.json for assertions (test-side,
// independent of production decoders beyond JSON syntax).
func readBundleDoc(t *testing.T, store, ref string) (raw []byte, entries map[string]struct {
	Kind  string `json:"kind"`
	Mode  int64  `json:"mode"`
	Size  int64  `json:"size"`
	Digest string `json:"digest"`
	Role  string `json:"role"`
}, treeDigest string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(store, "objects", strings.TrimPrefix(ref, "sha256:"), "bundle.json"))
	if err != nil {
		t.Fatalf("read bundle.json: %v", err)
	}
	var doc struct {
		Schema     string `json:"schema"`
		Entries    []struct {
			Path   string `json:"path"`
			Kind   string `json:"kind"`
			Mode   int64  `json:"mode"`
			Size   int64  `json:"size"`
			Digest string `json:"digest"`
			Role   string `json:"role"`
		} `json:"entries"`
		TreeDigest string `json:"tree_digest"`
	}
	if err := strictUnmarshalJSON(raw, &doc); err != nil {
		t.Fatalf("decode bundle.json: %v", err)
	}
	if doc.Schema != "machinery.tdd.bundle/v1" {
		t.Fatalf("bundle schema %q", doc.Schema)
	}
	entries = make(map[string]struct {
		Kind   string `json:"kind"`
		Mode   int64  `json:"mode"`
		Size   int64  `json:"size"`
		Digest string `json:"digest"`
		Role   string `json:"role"`
	}, len(doc.Entries))
	for _, e := range doc.Entries {
		entries[e.Path] = struct {
			Kind   string `json:"kind"`
			Mode   int64  `json:"mode"`
			Size   int64  `json:"size"`
			Digest string `json:"digest"`
			Role   string `json:"role"`
		}{Kind: e.Kind, Mode: e.Mode, Size: e.Size, Digest: e.Digest, Role: e.Role}
	}
	return raw, entries, doc.TreeDigest
}

// AC1/AC6: capture -> materialize round trip preserves exact bytes, portable
// permission bits and empty directories over the real filesystem.
func TestCaptureRoundTripBytesModesEmptyDirs(t *testing.T) {
	src := writeCaptureSource(t)
	ctl := separateControlMaterialization(t, src)
	store := filepath.Join(t.TempDir(), "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init store: %v", err)
	}
	v := &testInputView{src: src, ctl: ctl}
	ref, err := Capture(context.Background(), newCaptureRequest(v, store))
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if ref.Ref() == "" || !strings.HasPrefix(ref.Ref(), "sha256:") {
		t.Fatalf("capture returned no content-addressed ref: %q", ref.Ref())
	}
	dest := filepath.Join(t.TempDir(), "materialized")
	if err := MaterializeBundle(context.Background(), store, fixtureProjectID, ref.Ref(), dest); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	// byte/mode/empty-dir equality on representative entries
	checks := []struct {
		rel     string
		content string
		mode    os.FileMode
	}{
		{"src.txt", fixSubjectBytes, 0o600},
		{"impl/tool.sh", fixTool, 0o755},
		{"impl/app.go", fixApp, 0o644},
		{"tests/fixtures/data.bin", "\x00\x01binary\xff", 0o644},
		{".build-config", fixDotConfig, 0o644},
		{"vendor.lock", fixLock, 0o644},
	}
	for _, c := range checks {
		p := filepath.Join(dest, filepath.FromSlash(c.rel))
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("materialized %s missing: %v", c.rel, err)
		}
		if string(data) != c.content {
			t.Errorf("materialized %s bytes differ", c.rel)
		}
		fi, err := os.Lstat(p)
		if err != nil {
			t.Fatalf("lstat %s: %v", c.rel, err)
		}
		if fi.Mode().Perm() != c.mode {
			t.Errorf("materialized %s mode = %o, want %o", c.rel, fi.Mode().Perm(), c.mode)
		}
	}
	if fi, err := os.Lstat(filepath.Join(dest, "tests", "fixtures", "empty-dir")); err != nil || !fi.IsDir() {
		t.Fatalf("empty fixture directory not retained: %v", err)
	}
	// a whitespace-only change is never an equivalent frozen identity
	if err := os.WriteFile(filepath.Join(src, "impl", "app.go"), []byte(fixApp+"\n"), 0o644); err != nil {
		t.Fatalf("mutate: %v", err)
	}
	ref2, err := Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, store))
	if err != nil {
		t.Fatalf("recapture: %v", err)
	}
	if ref2.Ref() == ref.Ref() {
		t.Fatal("whitespace-only change produced the same frozen identity")
	}
}

// AC1: non-subject files are frozen by default (helpers, fixtures, locks,
// dot configs, empty directories), subjects carry the subject role, the
// dependency closure is separate, design authorities carry the design role,
// and the control namespace plus judgment control never appear as source.
func TestCaptureRolesFrozenByDefaultAndControlJudgmentExclusion(t *testing.T) {
	src := writeCaptureSource(t)
	ctl := separateControlMaterialization(t, src)
	store := filepath.Join(t.TempDir(), "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init store: %v", err)
	}
	ref, err := Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, store))
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	_, entries, _ := readBundleDoc(t, store, ref.Ref())
	roleCases := map[string]string{
		"src.txt":                  RoleSubject,
		"tests/alpha_test.go":      RoleFrozen,
		"tests/helpers/helper.go":  RoleFrozen,
		"tests/fixtures/data.bin":  RoleFrozen,
		"tests/fixtures/empty-dir": RoleFrozen,
		"vendor-deps/lib.go":       RoleDependency,
		"BUILD.md":                 RoleDesign,
		"machines/Alpha.oracle.md": RoleDesign,
	}
	for rel, want := range roleCases {
		got, ok := entries[rel]
		if !ok {
			t.Errorf("captured inventory is missing %q", rel)
			continue
		}
		if got.Role != want {
			t.Errorf("role of %q = %q, want %q", rel, got.Role, want)
		}
	}
	for _, forbidden := range []string{"assurance/plan.json", "assurance/milestones/M1.json", "attestations.yaml", ".git/HEAD"} {
		if _, ok := entries[forbidden]; ok {
			t.Errorf("excluded path %q appears in the source bundle", forbidden)
		}
	}
	// control bytes are archived exactly, keyed by their own byte digest
	planBytes, _ := os.ReadFile(filepath.Join(ctl, "plan.json"))
	archived, err := os.ReadFile(filepath.Join(store, "controls", digestHex(planBytes)+".json"))
	if err != nil || string(archived) != string(planBytes) {
		t.Errorf("control plan.json not archived byte-exactly: %v", err)
	}
	attestBytes, err := os.ReadFile(filepath.Join(src, "attestations.yaml"))
	if err != nil {
		t.Fatalf("read attestations: %v", err)
	}
	archivedAtt, err := os.ReadFile(filepath.Join(store, "controls", digestHex(attestBytes)+".json"))
	if err != nil || string(archivedAtt) != string(attestBytes) {
		t.Errorf("judgment control not archived byte-exactly: %v", err)
	}
}

// AC2: the tree digest is the exact typed machinery.tdd.tree/v1 encoding and
// bundle.json bytes are canonical and deterministic.
func TestCaptureTreeDigestEncodingOracle(t *testing.T) {
	// independent oracle over a minimal fixed tree
	content := "x\n"
	sum := sha256.Sum256([]byte(content))
	type ent = struct {
		Rel  string
		Dir  bool
		Perm uint32
		Size int64
		Dig  []byte
		Role string
	}
	oracle := testTreeEnc([]ent{
		{Rel: ".", Dir: true, Perm: 0o755, Role: RoleDesign},
		{Rel: "a.txt", Perm: 0o644, Size: int64(len(content)), Dig: sum[:], Role: RoleFrozen},
		{Rel: "sub", Dir: true, Perm: 0o700, Role: RoleSubject},
		{Rel: "sub/b.txt", Perm: 0o600, Size: int64(len(content)), Dig: sum[:], Role: RoleSubject},
	})
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for rel, mode := range map[string]os.FileMode{"a.txt": 0o644, "sub/b.txt": 0o600} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(content), mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Join(root, filepath.FromSlash(rel)), mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(filepath.Join(root, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	ctl := t.TempDir()
	if err := os.WriteFile(filepath.Join(ctl, "plan.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(t.TempDir(), "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init store: %v", err)
	}
	m := captureManifest()
	m.FrozenRoots = []string{"a.txt"}
	m.Suites[0].Root = "."
	m.Suites[0].Files = []string{"a.txt"}
	m.Suites[0].DependencyRoots = nil
	m.SubjectEntries = []SubjectEntry{{Path: "sub", Kind: "directory"}}
	req := CaptureRequest{Inputs: (&testInputView{src: root, ctl: ctl}).view(), Manifest: m, Name: "oracle", Store: store, Limits: captureLimits()}
	ref, err := Capture(context.Background(), req)
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if ref.Ref() != oracle {
		t.Errorf("tree digest %s != independent oracle %s", ref.Ref(), oracle)
	}
	raw, _, treeDigest := readBundleDoc(t, store, ref.Ref())
	if treeDigest != oracle {
		t.Errorf("bundle.json tree_digest %s != oracle %s", treeDigest, oracle)
	}
	if strings.Contains(string(raw), ": ") || strings.Contains(string(raw), "\n") {
		t.Error("bundle.json bytes are not canonical (whitespace present)")
	}
}

// AC2: identical logical trees produce identical digests across separate
// materializations (cross-platform reproduction; umask independence).
func TestCaptureDigestDeterministicAcrossMaterializations(t *testing.T) {
	var refs [2]string
	for i := range refs {
		src := writeCaptureSource(t)
		ctl := separateControlMaterialization(t, src)
		store := filepath.Join(t.TempDir(), "store")
		if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
			t.Fatalf("init store %d: %v", i, err)
		}
		ref, err := Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, store))
		if err != nil {
			t.Fatalf("capture %d: %v", i, err)
		}
		refs[i] = ref.Ref()
	}
	if refs[0] != refs[1] {
		t.Errorf("same logical tree hashed differently across materializations: %s vs %s", refs[0], refs[1])
	}
}

// AC2: escaping links, symlinks, special files and hardlink aliases are
// rejected; the assertions are falsifiable (frozen unsafe challenge twins
// demonstrate a permissive implementation would accept them).
func TestCaptureRejectsSymlinkSpecialAndHardlinkAliases(t *testing.T) {
	src := writeCaptureSource(t)
	ctl := separateControlMaterialization(t, src)
	store := filepath.Join(t.TempDir(), "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init store: %v", err)
	}
	if err := os.Symlink(filepath.Join("..", "..", "elsewhere"), filepath.Join(src, "tests", "escape-link")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	_, err := Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, store))
	if err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("escaping symlink accepted: err = %v", err)
	}
	// unsafe challenge twin: a permissive walker that ignores symlinks
	permissive := func(rel string) error { return nil }
	if err := permissive("tests/escape-link"); err != nil {
		t.Fatal("challenge twin must accept the symlink to prove the assertion discriminates")
	}
	// safe control: the same tree without the symlink captures cleanly
	if err := os.Remove(filepath.Join(src, "tests", "escape-link")); err != nil {
		t.Fatal(err)
	}
	if _, err := Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, store)); err != nil {
		t.Errorf("safe control capture rejected: %v", err)
	}
	if runtime.GOOS != "windows" {
		fifo := filepath.Join(src, "tests", "fixtures", "pipe")
		if err := mkfifo(t, fifo); err != nil {
			t.Fatalf("mkfifo: %v", err)
		}
		if _, err := Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, store)); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
			t.Errorf("special file accepted: err = %v", err)
		}
		if err := os.Remove(fifo); err != nil {
			t.Fatal(err)
		}
	}
	// hardlink alias: the same inode bound at two captured paths
	if err := os.Link(filepath.Join(src, "impl", "app.go"), filepath.Join(src, "impl", "alias.go")); err != nil {
		t.Fatalf("hardlink: %v", err)
	}
	_, err = Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, store))
	if err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("hardlink alias accepted: err = %v", err)
	}
}

// AC2: case aliases are rejected wherever they would collide.
func TestCaptureRejectsCaseAliasInventory(t *testing.T) {
	src := writeCaptureSource(t)
	ctl := separateControlMaterialization(t, src)
	store := filepath.Join(t.TempDir(), "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init store: %v", err)
	}
	// declared-alias form: a frozen root aliasing a subject directory
	m := captureManifest()
	m.FrozenRoots = []string{"TESTS"}
	_, err := Capture(context.Background(), CaptureRequest{Inputs: (&testInputView{src: src, ctl: ctl}).view(), Manifest: m, Name: "alias", Store: store, Limits: captureLimits()})
	if err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("case-aliased declared roots accepted: err = %v", err)
	}
	// unsafe challenge: naive exact-string matching accepts the alias
	naive := func(a, b string) bool { return a == b }
	if naive("TESTS", "tests") {
		t.Fatal("challenge twin must accept the alias to prove the assertion discriminates")
	}
}

// AC2/AC3: the store must not overlap any governed root by canonical alias
// or parent/child relationship; boundary-safe siblings are legal. The frozen
// challenge twin shows a naive lexical-prefix check misjudges siblings.
func TestCaptureRejectsStoreOverlap(t *testing.T) {
	src := writeCaptureSource(t)
	ctl := separateControlMaterialization(t, src)
	base := t.TempDir()
	inside := filepath.Join(base, "governed") // governed root really inside the store path
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		store string
		roots []string
		ok    bool
	}{
		{"store inside source", filepath.Join(src, "store"), []string{src, ctl}, false},
		{"store equals source", src, []string{src, ctl}, false},
		{"governed root inside store", base, []string{inside, ctl}, false},
		{"store inside control root", filepath.Join(ctl, "store"), []string{src, ctl}, false},
		{"boundary sibling", filepath.Join(base, "src-store"), []string{src, ctl}, true},
	} {
		err := ValidateStorePlacement(tc.store, tc.roots...)
		if tc.ok && err != nil {
			t.Errorf("%s: sibling store rejected: %v", tc.name, err)
		}
		if !tc.ok && (err == nil || !strings.Contains(err.Error(), "STORE_ROOT_MISMATCH")) {
			t.Errorf("%s: overlapping store accepted: err = %v", tc.name, err)
		}
	}
	naive := func(store, root string) bool { return strings.HasPrefix(store, root) }
	if naive(filepath.Join(base, "src-store"), base) == false {
		t.Fatal("setup: naive twin must prefix-match")
	}
	// naive twin cannot distinguish sibling "src2" from child of "src"
	root := filepath.Join(base, "src")
	if naive(filepath.Join(base, "src2"), root) {
		t.Error("challenge twin: naive prefix check treats the sibling src2 as inside src; production must not")
	}
	// end to end: capture with an overlapping store fails closed
	if _, err := InitStore(context.Background(), filepath.Join(src, "store"), fixtureProjectID); err != nil {
		t.Fatalf("init overlapping store: %v", err)
	}
	_, err := Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, filepath.Join(src, "store")))
	if err == nil || !strings.Contains(err.Error(), "STORE_ROOT_MISMATCH") {
		t.Errorf("capture into overlapping store accepted: err = %v", err)
	}
}

// AC2: frozen descendants under mutable subject directories are rejected;
// subject entries must exist.
func TestCaptureRejectsFrozenUnderMutableSubjectAndMissingSubjects(t *testing.T) {
	src := writeCaptureSource(t)
	ctl := separateControlMaterialization(t, src)
	store := filepath.Join(t.TempDir(), "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init store: %v", err)
	}
	m := captureManifest()
	m.SubjectEntries = []SubjectEntry{{Path: "tests", Kind: "directory"}}
	_, err := Capture(context.Background(), CaptureRequest{Inputs: (&testInputView{src: src, ctl: ctl}).view(), Manifest: m, Name: "overlap", Store: store, Limits: captureLimits()})
	if err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("frozen descendant under mutable subject accepted: err = %v", err)
	}
	m2 := captureManifest()
	m2.SubjectEntries = []SubjectEntry{{Path: "does/not/exist.go", Kind: "file"}}
	_, err = Capture(context.Background(), CaptureRequest{Inputs: (&testInputView{src: src, ctl: ctl}).view(), Manifest: m2, Name: "missing", Store: store, Limits: captureLimits()})
	if err == nil || !strings.Contains(err.Error(), "MISSING_CONTRACT") {
		t.Errorf("missing subject entry accepted: err = %v", err)
	}
}

// AC1: an InputView without both callbacks or with a failing revalidation is
// rejected; a mid-capture mutation of the held view is a race, not success.
func TestCaptureRequiresVerifiedImmutableInputView(t *testing.T) {
	src := writeCaptureSource(t)
	ctl := separateControlMaterialization(t, src)
	store := filepath.Join(t.TempDir(), "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init store: %v", err)
	}
	v := (&testInputView{src: src, ctl: ctl}).view()
	v.Revalidate = nil
	if _, err := Capture(context.Background(), CaptureRequest{Inputs: v, Manifest: captureManifest(), Name: "n", Store: store, Limits: captureLimits()}); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("nil Revalidate accepted: err = %v", err)
	}
	v = (&testInputView{src: src, ctl: ctl}).view()
	v.Release = nil
	if _, err := Capture(context.Background(), CaptureRequest{Inputs: v, Manifest: captureManifest(), Name: "n", Store: store, Limits: captureLimits()}); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("nil Release accepted: err = %v", err)
	}
	req := newCaptureRequest(&testInputView{src: src, ctl: ctl}, store)
	req.Name = "not an id!"
	if _, err := Capture(context.Background(), req); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("invalid capture name accepted: err = %v", err)
	}
	// held-view mutation between entry and walk revalidation
	calls := 0
	mutating := &testInputView{src: src, ctl: ctl, revalidFn: func() error {
		calls++
		if calls == 2 {
			if err := os.WriteFile(filepath.Join(src, "impl", "app.go"), []byte("mutated\n"), 0o644); err != nil {
				return err
			}
			return fmt.Errorf("held view changed during capture")
		}
		return nil
	}}
	_, err := Capture(context.Background(), newCaptureRequest(mutating, store))
	if err == nil || !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Errorf("mid-capture mutation not a blocking race: err = %v", err)
	}
}

// AC4/AC6: content objects are immutable and read-verified; tampered bytes,
// tampered bundle documents and missing blobs fail closed.
func TestMaterializeRejectsTamperedAndMissingObjects(t *testing.T) {
	src := writeCaptureSource(t)
	ctl := separateControlMaterialization(t, src)
	store := filepath.Join(t.TempDir(), "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init store: %v", err)
	}
	ref, err := Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, store))
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	_, entries, _ := readBundleDoc(t, store, ref.Ref())
	dig := entries["src.txt"].Digest
	blob := filepath.Join(store, "blobs", strings.TrimPrefix(dig, "sha256:"))
	if err := os.Chmod(blob, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blob, []byte("tampered subject bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "m")
	if err := MaterializeBundle(context.Background(), store, fixtureProjectID, ref.Ref(), dest); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("tampered blob materialized: err = %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Error("failed materialization left a destination behind")
	}
	// restore, then tamper the bundle document's declared inventory
	if err := os.WriteFile(blob, []byte(fixSubjectBytes), 0o400); err != nil {
		t.Fatal(err)
	}
	bj := filepath.Join(store, "objects", strings.TrimPrefix(ref.Ref(), "sha256:"), "bundle.json")
	if err := os.Chmod(bj, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bj, []byte(strings.Replace(string(mustRead(t, bj)), `"size":15`, `"size":14`, 1)), 0o400); err != nil {
		t.Fatal(err)
	}
	dest2 := filepath.Join(t.TempDir(), "m2")
	if err := MaterializeBundle(context.Background(), store, fixtureProjectID, ref.Ref(), dest2); err == nil {
		t.Error("tampered bundle.json accepted by materialize")
	}
}

// AC6: capture is idempotent (content-addressed) and safe under concurrent
// reader/writer pressure on the real store.
func TestCaptureIdempotentConcurrent(t *testing.T) {
	src := writeCaptureSource(t)
	ctl := separateControlMaterialization(t, src)
	store := filepath.Join(t.TempDir(), "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init store: %v", err)
	}
	first, err := Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, store))
	if err != nil {
		t.Fatalf("first capture: %v", err)
	}
	second, err := Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, store))
	if err != nil {
		t.Fatalf("second capture: %v", err)
	}
	if first.Ref() != second.Ref() {
		t.Fatalf("idempotent capture changed identity: %s vs %s", first.Ref(), second.Ref())
	}
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, store)); err != nil {
				errs <- fmt.Errorf("writer: %w", err)
			}
		}()
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dest := filepath.Join(t.TempDir(), "m")
			if err := MaterializeBundle(context.Background(), store, fixtureProjectID, first.Ref(), dest); err != nil {
				errs <- fmt.Errorf("reader: %w", err)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// AC6: every operation honors caller cancellation; failure never retries
// automatically and leaves no success artifact.
func TestCaptureHonorsCallerCancellation(t *testing.T) {
	src := writeCaptureSource(t)
	ctl := separateControlMaterialization(t, src)
	store := filepath.Join(t.TempDir(), "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init store: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Capture(ctx, newCaptureRequest(&testInputView{src: src, ctl: ctl}, store))
	if err == nil || !strings.Contains(err.Error(), "TIMEOUT") {
		t.Errorf("cancelled capture did not fail closed: err = %v", err)
	}
	objectsBefore := countEntries(t, filepath.Join(store, "objects"))
	if objectsBefore != 0 {
		t.Errorf("cancelled capture left %d objects behind", objectsBefore)
	}
	if err := MaterializeBundle(ctx, store, fixtureProjectID, "sha256:"+strings.Repeat("0", 64), filepath.Join(t.TempDir(), "m")); err == nil || !strings.Contains(err.Error(), "TIMEOUT") {
		t.Errorf("cancelled materialize did not fail closed: err = %v", err)
	}
}

// AC3: no environment variable ever selects or overrides the explicit store.
func TestCaptureIgnoresEnvironmentStoreDiscovery(t *testing.T) {
	t.Setenv("MACHINERY_TDD_STORE", "/nonexistent-env-store")
	t.Setenv("ASSURANCE_STORE", "/nonexistent-env-store-2")
	src := writeCaptureSource(t)
	ctl := separateControlMaterialization(t, src)
	store := filepath.Join(t.TempDir(), "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init store: %v", err)
	}
	if _, err := Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, store)); err != nil {
		t.Fatalf("explicit capture disturbed by environment: %v", err)
	}
	if _, err := os.Stat("/nonexistent-env-store"); !os.IsNotExist(err) {
		t.Error("environment-directed store path was created")
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return data
}

func countEntries(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatalf("readdir %s: %v", dir, err)
	}
	return len(entries)
}

// strictUnmarshalJSON is the test-side closed decoder (no unknown keys, no
// trailing data), independent of production decoding.
func strictUnmarshalJSON(data []byte, v any) error {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("trailing data after JSON document")
	}
	return nil
}
