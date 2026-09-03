package pack

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/ir"
)

func TestPackMutableFileReadsAreBoundedAgainstContinuousAppender(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*testing.T, *os.Root) error
	}{
		{
			name: "design source",
			run: func(_ *testing.T, root *os.Root) error {
				_, err := readPackRegularRoot(root, "source", "test source")
				return err
			},
		},
		{
			name: "tree fingerprint",
			run: func(_ *testing.T, root *os.Root) error {
				_, err := capturePackTree(root, "tree")
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "source")
			if tc.name == "tree fingerprint" {
				if err := os.Mkdir(filepath.Join(dir, "tree"), 0o755); err != nil {
					t.Fatal(err)
				}
				path = filepath.Join(dir, "tree", "member")
			}
			if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close() //nolint:errcheck // test cleanup
			stop := make(chan struct{})
			appenderDone := make(chan error, 1)
			oldPoint := packFileReadPoint
			started := false
			packFileReadPoint = func(string) error {
				if started {
					return nil
				}
				started = true
				appender, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
				if err != nil {
					return err
				}
				chunk := bytes.Repeat([]byte("a"), 64<<10)
				if _, err := appender.Write(chunk); err != nil {
					_ = appender.Close()
					return err
				}
				go func() {
					var appendErr error
					defer func() { appenderDone <- errors.Join(appendErr, appender.Close()) }()
					for {
						select {
						case <-stop:
							return
						default:
							if _, err := appender.Write(chunk); err != nil {
								appendErr = err
								return
							}
						}
					}
				}()
				return nil
			}
			defer func() { packFileReadPoint = oldPoint }()
			result := make(chan error, 1)
			go func() { result <- tc.run(t, root) }()
			select {
			case err := <-result:
				close(stop)
				if appendErr := <-appenderDone; appendErr != nil {
					t.Fatal(appendErr)
				}
				if !started || err == nil || !strings.Contains(err.Error(), "changed") {
					t.Fatalf("continuous growth was accepted: started=%v err=%v", started, err)
				}
			case <-time.After(2 * time.Second):
				close(stop)
				<-appenderDone
				t.Fatal("pack file read followed a continuous appender instead of stopping at the witnessed size")
			}
		})
	}
}

func TestPackDirectoryEnumerationStopsAtFixedCeiling(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a", "b", "c"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	_, err = readPackDirBounded(root, ".", 2)
	if err == nil || !strings.Contains(err.Error(), "2-entry limit") {
		t.Fatalf("over-limit pack directory inventory was accepted: %v", err)
	}
}

func TestPackWriterRejectsPackAndMemberInventoriesBeforePlanning(t *testing.T) {
	t.Run("pack count", func(t *testing.T) {
		packs := make(map[string]map[string]string, packDirMaxEntries+1)
		for i := 0; i <= packDirMaxEntries; i++ {
			packs[fmt.Sprintf("p%d", i)] = nil
		}
		if _, err := writePacksRootedWithRename(t.TempDir(), nil, packs, nil); err == nil || !strings.Contains(err.Error(), "pack limit") {
			t.Fatalf("over-limit pack inventory was accepted: %v", err)
		}
	})
	t.Run("member count", func(t *testing.T) {
		members := make(map[string]string, packTreeMaxEntries)
		for i := 0; i < packTreeMaxEntries; i++ {
			members[fmt.Sprintf("m%d", i)] = "x"
		}
		if _, err := writePacksRootedWithRename(t.TempDir(), nil, map[string]map[string]string{"orders": members}, nil); err == nil || !strings.Contains(err.Error(), "member limit") {
			t.Fatalf("over-limit pack member inventory was accepted: %v", err)
		}
	})
}

func TestPackRegularFileLimitsRejectSparseMembers(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "tree"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "tree", "oversized")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(file.Truncate(packFileMaxBytes+1), file.Close()); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	if _, err := readPackRegularRoot(root, filepath.Join("tree", "oversized"), "test source"); err == nil || !strings.Contains(err.Error(), "byte limit") {
		t.Fatalf("oversized direct read was accepted: %v", err)
	}
	if _, err := capturePackTree(root, "tree"); err == nil || !strings.Contains(err.Error(), "member limit") {
		t.Fatalf("oversized tree member was accepted: %v", err)
	}
}

func parentDesign() string {
	return filepath.Join("..", "..", "examples", "checkout-split", "parent", "design")
}

// copyParentDesign copies the checkout-split parent design into a temp dir so
// a test can mutate the sources (ARCHITECTURE.md table cells, decomposition
// waivers) without touching the shipped example.
func copyParentDesign(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	src := parentDesign()
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(src, p)
		if rerr != nil {
			return rerr
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

// editDesignFile replaces old with new in one design file, failing when the
// needle is absent (a silent no-op mutation would vacuously pass the test).
func editDesignFile(t *testing.T, design, name, old, new string) {
	t.Helper()
	p := filepath.Join(design, name)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("%s does not contain %q", name, old)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(data), old, new, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDecompositionValidates(t *testing.T) {
	d, err := LoadDecomposition(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Subsystems) != 2 {
		t.Fatalf("subsystems=%d", len(d.Subsystems))
	}
	if d.Subsystems[0].ID != "orders" || d.Subsystems[1].ID != "payments" {
		t.Fatalf("ids=%v,%v", d.Subsystems[0].ID, d.Subsystems[1].ID)
	}
	if d.Subsystems[0].DelegatedInvariants[0] != "no-ship-without-capture" {
		t.Fatal("delegated invariant lost")
	}
}

func TestLoadDecompositionRejectsUnknownAndWrongTypedSchemaDeterministically(t *testing.T) {
	for _, tc := range []struct {
		name, old, replacement, want string
	}{
		{"unknown root", "decomposition_version: 1\n", "decomposition_version: 1\nzzz: true\naaa: true\n", "unknown root key 'aaa'"},
		{"wrong subsystems type", "subsystems:\n", "subsystems: null\n", "subsystems must be an array"},
		{"unknown subsystem", "  - id: orders\n", "  - id: orders\n    zzz: true\n    aaa: true\n", "unknown key 'aaa'"},
		{"wrong list member", "    owns: [Order]\n", "    owns: [Order, null]\n", "owns[1] must be a string"},
		{"wrong boundary waiver", "    child_design: ../../orders/design\n", "    child_design: ../../orders/design\n    boundary_events:\n      none: reason\n      typo: value\n", "boundary_events must be a mapping"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := copyParentDesign(t)
			if tc.name == "wrong subsystems type" {
				if err := os.WriteFile(filepath.Join(design, "decomposition.yaml"), []byte("decomposition_version: 1\nsubsystems: null\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			} else {
				editDesignFile(t, design, "decomposition.yaml", tc.old, tc.replacement)
			}
			var stable string
			for i := 0; i < 100; i++ {
				_, err := LoadDecomposition(design)
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("invalid decomposition accepted or wrong diagnostic: %v", err)
				}
				if i == 0 {
					stable = err.Error()
				} else if err.Error() != stable {
					t.Fatalf("diagnostic changed on run %d:\nwant %s\n got %s", i, stable, err)
				}
			}
		})
	}
}

func TestLoadPackMapRejectsOpenOrWrongTypedSchema(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{"unknown sorted", "subsystem: x\npack_hash: h\nmachine: M\nmapping: {}\nzzz: 1\naaa: 1\n", "unknown root key 'aaa'"},
		{"missing mapping", "subsystem: x\npack_hash: h\nmachine: M\n", "mapping is required"},
		{"null mapping", "subsystem: x\npack_hash: h\nmachine: M\nmapping: null\n", "mapping is required"},
		{"wrong mapping value", "subsystem: x\npack_hash: h\nmachine: M\nmapping:\n  Ready: null\n", "mapping value for 'Ready'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			if err := os.WriteFile(filepath.Join(design, "packmap.yaml"), []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadPackMap(design)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid packmap accepted or wrong diagnostic: %v", err)
			}
		})
	}
}

func TestLoadPackManifestRejectsOpenOrWrongTypedSchema(t *testing.T) {
	valid := "pack_version: 1\npack_revision: 1\nsubsystem: x\ncontract_module: XContract\nowns: []\ncomponents: []\nboundaries: []\ndelegated_invariants: []\ncontent_hash: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n"
	for _, tc := range []struct {
		name, body, want string
	}{
		{"unknown sorted", valid + "zzz: 1\naaa: 1\n", "unknown root key 'aaa'"},
		{"wrong revision", strings.Replace(valid, "pack_revision: 1", "pack_revision: null", 1), "pack_revision"},
		{"missing list", strings.Replace(valid, "owns: []\n", "", 1), "owns is required"},
		{"wrong list member", strings.Replace(valid, "owns: []", "owns: [null]", 1), "owns[0]"},
		{"wrong hash", strings.Replace(valid, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "ABC", 1), "64 lowercase"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			if err := os.Mkdir(filepath.Join(design, "pack"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(design, "pack", "pack.yaml"), []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadPackManifest(design)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid manifest accepted or wrong diagnostic: %v", err)
			}
		})
	}
}

func TestGenerateTLAFromMachineBytesUsesStableLogicalSource(t *testing.T) {
	const logical = "design/contracts/Stable.machine.json"
	var stable string
	for i := 0; i < 100; i++ {
		_, _, _, err := generateTLAFromMachineBytes(logical, []byte("{"))
		if err == nil || !strings.Contains(err.Error(), logical) || strings.Contains(err.Error(), "machinery-pack-machine-") {
			t.Fatalf("logical source missing or scratch leaked: %v", err)
		}
		if i == 0 {
			stable = err.Error()
		} else if err.Error() != stable {
			t.Fatalf("diagnostic changed on run %d:\nwant %s\n got %s", i, stable, err)
		}
	}
}

func TestLoadDecompositionRequiresAuthoritativeArchitectureContract(t *testing.T) {
	for _, tc := range []struct {
		name, mutate, want string
	}{
		{"missing", "missing", "read ARCHITECTURE.md"},
		{"bad fence", "fence", "no heading-anchored Architecture Contract YAML fence"},
		{"bad yaml", "yaml", "parse Architecture Contract YAML"},
		{"wrong boundaries type", "type", "boundaries is required and must be an array"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := copyParentDesign(t)
			architecture := filepath.Join(design, "ARCHITECTURE.md")
			switch tc.mutate {
			case "missing":
				if err := os.Remove(architecture); err != nil {
					t.Fatal(err)
				}
			case "fence":
				if err := os.WriteFile(architecture, []byte("# Architecture\n\nNo authoritative contract.\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			case "yaml":
				if err := os.WriteFile(architecture, []byte("## Architecture Contract\n\n```yaml\nboundaries: [\n```\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			case "type":
				if err := os.WriteFile(architecture, []byte("## Architecture Contract\n\n```yaml\ncontract_version: 1\nboundaries: null\n```\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			var stable string
			for i := 0; i < 20; i++ {
				_, err := LoadDecomposition(design)
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("invalid Architecture Contract accepted or wrong diagnostic: %v", err)
				}
				if i == 0 {
					stable = err.Error()
				} else if err.Error() != stable {
					t.Fatalf("diagnostic changed: want %s, got %v", stable, err)
				}
			}
		})
	}
}

func TestContractBoundariesAcceptsLegitimateEmptyContract(t *testing.T) {
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "ARCHITECTURE.md"), []byte("## Architecture Contract\n\n```yaml\ncontract_version: 1\nboundaries: []\n```\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := openDesignRoot(design)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	boundaries, err := contractBoundariesRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(boundaries) != 0 {
		t.Fatalf("empty contract has %d boundaries", len(boundaries))
	}
}

func TestLoadDecompositionRejectsPortableSubsystemAlias(t *testing.T) {
	design := copyParentDesign(t)
	editDesignFile(t, design, "decomposition.yaml", "  - id: payments", "  - id: Orders")
	_, err := LoadDecomposition(design)
	if err == nil || !strings.Contains(err.Error(), "case-insensitive") {
		t.Fatalf("portable subsystem alias accepted: %v", err)
	}
}

func TestLoadDecompositionRejectsWindowsDeviceSubsystemName(t *testing.T) {
	design := copyParentDesign(t)
	editDesignFile(t, design, "decomposition.yaml", "  - id: payments", "  - id: CON")
	_, err := LoadDecomposition(design)
	if err == nil || !strings.Contains(err.Error(), "reserved Windows device") {
		t.Fatalf("Windows device subsystem name accepted: %v", err)
	}
}

func TestRetainedDiagnosticsAreDeterministic(t *testing.T) {
	design := copyParentDesign(t)
	editDesignFile(t, design, "decomposition.yaml", "decomposition_version: 1\n", "decomposition_version: 1\nretained:\n  zzz-unknown: z\n  no-ship-without-capture: conflict\n  aaa-unknown: a\n")
	var want string
	for i := 0; i < 100; i++ {
		_, err := LoadDecomposition(design)
		if err == nil {
			t.Fatal("multi-invalid retained inventory passed")
		}
		if i == 0 {
			want = err.Error()
		} else if err.Error() != want {
			t.Fatalf("retained diagnostics changed on run %d:\nwant %s\n got %s", i, want, err)
		}
	}
	for _, fragment := range []string{"aaa-unknown", "zzz-unknown", "both retained and delegated"} {
		if !strings.Contains(want, fragment) {
			t.Fatalf("deterministic diagnostic omitted %q: %s", fragment, want)
		}
	}
}

func TestRootedDesignReadsCannotHybridizeAfterParentSwap(t *testing.T) {
	base := t.TempDir()
	design := filepath.Join(base, "design")
	held := filepath.Join(base, "held-design")
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(design, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "first"), []byte("inside-first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "second"), []byte("inside-second"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "second"), []byte("outside-second"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := openDesignRoot(design)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	first, err := readDesignFileRoot(root, "first")
	if err != nil || string(first) != "inside-first" {
		t.Fatalf("first snapshot read = %q, %v", first, err)
	}
	if err := os.Rename(design, held); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, design); err != nil {
		t.Fatal(err)
	}
	second, err := readDesignFileRoot(root, "second")
	if err != nil || string(second) != "inside-second" {
		t.Fatalf("rooted snapshot hybridized after swap: %q, %v", second, err)
	}
}

func TestGeneratePacksIsDeterministic(t *testing.T) {
	a, err := GeneratePacks(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	b, err := GeneratePacks(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	for id := range a {
		if ContentHash(a[id]) != ContentHash(b[id]) {
			t.Fatalf("pack %s not byte-deterministic", id)
		}
		for name, body := range a[id] {
			if b[id][name] != body {
				t.Fatalf("pack %s file %s differs across runs", id, name)
			}
		}
	}
}

func TestLoadDecompositionRejectsSymlinkedContractSource(t *testing.T) {
	design := copyParentDesign(t)
	contract := filepath.Join(design, "contracts", "OrdersContract.machine.json")
	outside := filepath.Join(t.TempDir(), "outside.machine.json")
	raw, err := os.ReadFile(contract)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(contract); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, contract); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDecomposition(design); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlinked contract source accepted: %v", err)
	}
}

func TestPackFilesOnDiskRejectsSymlinkMember(t *testing.T) {
	design := t.TempDir()
	packDir := filepath.Join(design, "pack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("smuggled"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(packDir, "events.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := PackFilesOnDisk(design); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlinked pack member accepted: %v", err)
	}
}

func TestPackFilesOnDiskDiagnosticIsDeterministicWithMultipleBadEntries(t *testing.T) {
	design := t.TempDir()
	packDir := filepath.Join(design, "pack")
	if err := os.Mkdir(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(packDir, "z-dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(packDir, "a-dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	const want = `pack/ contains a directory 'a-dir'`
	for i := 0; i < 100; i++ {
		_, err := PackFilesOnDisk(design)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("run %d diagnostic = %v, want %q", i, err, want)
		}
	}
}

func TestLoadModelithDiagnosticIsDeterministicWithMultipleBadEntries(t *testing.T) {
	design := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("kind: modelith\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"z.modelith.yaml", "a.modelith.yaml"} {
		if err := os.Symlink(outside, filepath.Join(design, name)); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
	}
	for i := 0; i < 100; i++ {
		root, err := openDesignRoot(design)
		if err != nil {
			t.Fatal(err)
		}
		_, loadErr := loadModelithRoot(design, root)
		closeErr := root.Close()
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		if loadErr == nil || !strings.Contains(loadErr.Error(), `'a.modelith.yaml' is a symlink`) {
			t.Fatalf("run %d diagnostic = %v", i, loadErr)
		}
	}
}

func TestPackOutputAliasDiagnosticIsDeterministic(t *testing.T) {
	design := t.TempDir()
	packsDir := filepath.Join(design, "packs")
	if err := os.Mkdir(packsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"PAYMENTS.pack", "ORDERS.pack"} {
		if err := os.Mkdir(filepath.Join(packsDir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	packs := map[string]map[string]string{
		"orders":   {"pack.yaml": "new"},
		"payments": {"pack.yaml": "new"},
	}
	for i := 0; i < 100; i++ {
		_, err := writePacksWithRename(design, packs, nil)
		if err == nil || !strings.Contains(err.Error(), `'ORDERS.pack' aliases generated target 'orders.pack'`) {
			t.Fatalf("run %d diagnostic = %v", i, err)
		}
	}
}

func TestWriteRefinementArtifactsRejectsSymlinkedFormalDir(t *testing.T) {
	design := t.TempDir()
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "Refinement.tla")
	if err := os.WriteFile(sentinel, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(design, "formal")); err != nil {
		t.Fatal(err)
	}
	_, err := writeRefinementArtifacts(design, map[string]string{"Refinement.tla": "replacement"})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlinked formal directory accepted: %v", err)
	}
	got, readErr := os.ReadFile(sentinel)
	if readErr != nil || string(got) != "outside" {
		t.Fatalf("outside target changed: %q, %v", got, readErr)
	}
}

func TestWriteRefinementArtifactsRejectsSymlinkTargetBeforeWrites(t *testing.T) {
	design := t.TempDir()
	fdir := filepath.Join(design, "formal")
	if err := os.Mkdir(fdir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(fdir, "B.cfg")); err != nil {
		t.Fatal(err)
	}
	_, err := writeRefinementArtifacts(design, map[string]string{"A.tla": "new-a", "B.cfg": "new-b"})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlinked generated target accepted: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(fdir, "A.tla")); !os.IsNotExist(statErr) {
		t.Fatalf("another target was written before symlink rejection: %v", statErr)
	}
}

func TestWriteRefinementArtifactsPreservesForeignSuffixPairAndIsIdempotent(t *testing.T) {
	design := t.TempDir()
	fdir := filepath.Join(design, "formal")
	if err := os.Mkdir(fdir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"OldPackRefinement.tla", "OldPackRefinement.cfg"} {
		if err := os.WriteFile(filepath.Join(fdir, name), []byte("stale"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"NewPackRefinement.tla": "new-tla",
		"NewPackRefinement.cfg": "new-cfg",
	}
	for i := 0; i < 2; i++ {
		names, err := writeRefinementArtifacts(design, files)
		if err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
		if !reflect.DeepEqual(names, []string{"NewPackRefinement.cfg", "NewPackRefinement.tla"}) {
			t.Fatalf("write %d names=%v", i, names)
		}
	}
	for _, name := range []string{"OldPackRefinement.tla", "OldPackRefinement.cfg"} {
		if got, err := os.ReadFile(filepath.Join(fdir, name)); err != nil || string(got) != "stale" {
			t.Fatalf("foreign suffix artifact %s changed: %q, %v", name, got, err)
		}
	}
}

func TestStaleRefinementArtifactsConvergesMissingAnchorConfig(t *testing.T) {
	design := t.TempDir()
	fdir := filepath.Join(design, "formal")
	if err := os.MkdirAll(fdir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "\\* machinery-version: v1\nCONSTANT MaxRetries = 3\nSPECIFICATION Spec\nPROPERTY CSpecHolds\n"
	if err := os.WriteFile(filepath.Join(fdir, "GonePackRefinement.cfg"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, err := staleRefinementArtifacts(design, design, map[string]string{})
	if err != nil || len(stale) != 1 || stale[0].Name != "GonePackRefinement.cfg" {
		t.Fatalf("missing-anchor pack cfg not reconciled: stale=%v err=%v", stale, err)
	}
}

func TestWritePacksReplacesEachPackAsACompleteSet(t *testing.T) {
	design := copyParentDesign(t)
	stale := filepath.Join(design, "packs", "orders.pack", "stale.txt")
	if err := os.WriteFile(stale, []byte("must disappear"), 0o644); err != nil {
		t.Fatal(err)
	}
	ids, err := WritePacks(design)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("ids=%v", ids)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale member survived atomic directory replacement: %v", err)
	}
	if _, err := os.Stat(filepath.Join(design, "packs", "orders.pack", "pack.yaml")); err != nil {
		t.Fatalf("replacement pack incomplete: %v", err)
	}
}

func TestWritePacksRemovesRenamedPackAndIsIdempotent(t *testing.T) {
	design := copyParentDesign(t)
	editDesignFile(t, design, "decomposition.yaml", "  - id: orders", "  - id: fulfillment")
	if _, err := WritePacks(design); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(design, "packs", "orders.pack")); !os.IsNotExist(err) {
		t.Fatalf("renamed orders.pack survived convergence: %v", err)
	}
	if info, err := os.Stat(filepath.Join(design, "packs", "fulfillment.pack")); err != nil || !info.IsDir() {
		t.Fatalf("renamed fulfillment.pack missing: %v", err)
	}
	want := snapshotPackDirs(t, filepath.Join(design, "packs"))
	if _, err := WritePacks(design); err != nil {
		t.Fatal(err)
	}
	got := snapshotPackDirs(t, filepath.Join(design, "packs"))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("second convergence changed pack bytes:\nwant %#v\n got %#v", want, got)
	}
}

func TestWritePacksMetadataDoesNotRereadUnlockedSources(t *testing.T) {
	design := copyParentDesign(t)
	results, err := writePacksWithMetadataHook(design, func() {
		if err := os.WriteFile(filepath.Join(design, "ARCHITECTURE.md"), []byte("mutated after commit\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	if err != nil {
		t.Fatalf("post-commit reporting reread mutated sources: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("reported %d packs, want 2", len(results))
	}
	for _, result := range results {
		dir := filepath.Join(design, "packs", result.ID+".pack")
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		files := map[string]string{}
		for _, entry := range entries {
			body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			files[entry.Name()] = string(body)
		}
		if result.FileCount != len(files) || result.Hash != ContentHash(files) {
			t.Fatalf("metadata for %s does not bind committed bytes: %#v", result.ID, result)
		}
	}
}

func TestStalePackDeletionRollsBackWithInstallFailure(t *testing.T) {
	design := t.TempDir()
	root := filepath.Join(design, "packs")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"orders.pack": "old-orders", "stale.pack": "old-stale"} {
		writePackRecoveryDir(t, root, name, body)
	}
	injected := errors.New("install failed")
	rename := func(opened *os.Root, old, new string) error {
		if strings.HasPrefix(old, ".machinery-pack-stage-") && new == "orders.pack" {
			return injected
		}
		return nil
	}
	_, err := writePacksWithRename(design, map[string]map[string]string{
		"orders": {"marker": "new-orders"},
	}, rename)
	if !errors.Is(err, injected) {
		t.Fatalf("install failure hidden: %v", err)
	}
	for name, want := range map[string]string{"orders.pack": "old-orders", "stale.pack": "old-stale"} {
		got, err := os.ReadFile(filepath.Join(root, name, "marker"))
		if err != nil || string(got) != want {
			t.Fatalf("rollback %s = %q, %v; want %q", name, got, err, want)
		}
	}
}

func TestWritePacksPreflightsLaterTargetBeforeStaging(t *testing.T) {
	design := copyParentDesign(t)
	root := filepath.Join(design, "packs")
	orders := filepath.Join(root, "orders.pack")
	want := snapshotPackDirs(t, orders)
	paymentTarget := filepath.Join(root, "payments.pack")
	if err := os.RemoveAll(paymentTarget); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), paymentTarget); err != nil {
		t.Fatal(err)
	}
	_, firstErr := WritePacks(design)
	_, secondErr := WritePacks(design)
	if firstErr == nil || !strings.Contains(firstErr.Error(), "real directory") {
		t.Fatalf("later symlink target was accepted: %v", firstErr)
	}
	if secondErr == nil || firstErr.Error() != secondErr.Error() || strings.Contains(firstErr.Error(), "machinery-design-source-") {
		t.Fatalf("invalid pack diagnostic is unstable or leaks private source root:\nfirst: %v\nsecond: %v", firstErr, secondErr)
	}
	if got := snapshotPackDirs(t, orders); !reflect.DeepEqual(got, want) {
		t.Fatalf("earlier target changed before later-target preflight failed:\nwant: %#v\n got: %#v", want, got)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "pack-stage") || strings.Contains(entry.Name(), "pack-backup") {
			t.Fatalf("preflight failure left transaction scratch %s", entry.Name())
		}
	}
}

func snapshotPackDirs(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			snapshot[filepath.ToSlash(rel)+"/"] = ""
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[filepath.ToSlash(rel)] = string(body)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestWritePacksLaterInstallFailureRestoresWholeSet(t *testing.T) {
	design := copyParentDesign(t)
	root := filepath.Join(design, "packs")
	if err := os.WriteFile(filepath.Join(root, "orders.pack", "old-only.txt"), []byte("orders sentinel"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "payments.pack", "old-only.txt"), []byte("payments sentinel"), 0o644); err != nil {
		t.Fatal(err)
	}
	want := snapshotPackDirs(t, root)
	packs, err := GeneratePacks(design)
	if err != nil {
		t.Fatal(err)
	}
	failed := false
	rename := func(root *os.Root, old, new string) error {
		if !failed && new == "payments.pack" && strings.HasPrefix(old, ".machinery-pack-stage-") {
			failed = true
			return errors.New("injected later pack install failure")
		}
		return nil
	}
	if _, err := writePacksWithRename(design, packs, rename); err == nil || !strings.Contains(err.Error(), "injected later pack install failure") {
		t.Fatalf("later install failure was ignored: %v", err)
	}
	if !failed {
		t.Fatal("failure injection did not reach the later target")
	}
	got := snapshotPackDirs(t, root)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pack set was not restored byte-for-byte:\nwant: %#v\n got: %#v", want, got)
	}
}

func TestWritePacksJoinsCommitAndRollbackErrors(t *testing.T) {
	design := copyParentDesign(t)
	packs, err := GeneratePacks(design)
	if err != nil {
		t.Fatal(err)
	}
	commitErr := errors.New("injected later install failure")
	rollbackErr := errors.New("injected later restore failure")
	rename := func(root *os.Root, old, new string) error {
		oldBase := old
		newBase := new
		if newBase == "payments.pack" && strings.HasPrefix(oldBase, ".machinery-pack-stage-") {
			return commitErr
		}
		if newBase == "payments.pack" && strings.HasPrefix(oldBase, ".machinery-pack-backup-") {
			return rollbackErr
		}
		return nil
	}
	_, err = writePacksWithRename(design, packs, rename)
	if !errors.Is(err, commitErr) || !errors.Is(err, rollbackErr) {
		t.Fatalf("combined failure does not expose both causes: %v", err)
	}
}

func TestWritePacksRollbackPreservesLateMutatedInstalledTarget(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(string, string) error
	}{
		{
			name: "content mutation",
			mutate: func(path, _ string) error {
				return os.WriteFile(path, []byte("user edit"), 0o644)
			},
		},
		{
			name: "ABA identity mutation",
			mutate: func(path, original string) error {
				if err := os.Remove(path); err != nil {
					return err
				}
				return os.WriteFile(path, []byte(original), 0o644)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			design := t.TempDir()
			root := filepath.Join(design, "packs")
			if err := os.Mkdir(root, 0o755); err != nil {
				t.Fatal(err)
			}
			writePackRecoveryDir(t, root, "orders.pack", "old-orders")
			writePackRecoveryDir(t, root, "payments.pack", "old-payments")
			newOrders := "new-orders"
			injected := errors.New("injected later install failure")
			rename := func(opened *os.Root, old, new string) error {
				if new == "payments.pack" && strings.HasPrefix(old, ".machinery-pack-stage-") {
					if err := test.mutate(filepath.Join(root, "orders.pack", "marker"), newOrders); err != nil {
						return err
					}
					return injected
				}
				return nil
			}
			_, err := writePacksWithRename(design, map[string]map[string]string{
				"orders":   {"marker": newOrders},
				"payments": {"marker": "new-payments"},
			}, rename)
			if !errors.Is(err, injected) || !strings.Contains(err.Error(), "preserving it") {
				t.Fatalf("late mutation was not a preservation blocker: %v", err)
			}
			got, readErr := os.ReadFile(filepath.Join(root, "orders.pack", "marker"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			want := "user edit"
			if test.name == "ABA identity mutation" {
				want = newOrders
			}
			if string(got) != want {
				t.Fatalf("late-mutated installed target was overwritten or removed: got %q, want %q", got, want)
			}
			for _, target := range []string{"orders.pack", "payments.pack"} {
				backup := filepath.Join(root, packScratchName("backup", target), "marker")
				if _, err := os.ReadFile(backup); err != nil {
					t.Fatalf("usable backup %s was not preserved: %v", target, err)
				}
			}
			if _, err := os.Lstat(filepath.Join(root, packJournalName)); err != nil {
				t.Fatalf("authoritative recovery journal was removed after preservation blocker: %v", err)
			}
		})
	}
}

func writePackRecoveryDir(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "marker"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func seedPackCrash(t *testing.T, root, phase string) []packJournalEntry {
	t.Helper()
	entries := []packJournalEntry{
		{Target: "orders.pack", Stage: packScratchName("stage", "orders.pack"), Backup: packScratchName("backup", "orders.pack"), Retire: packScratchName("retire", "orders.pack"), Existed: true},
		{Target: "payments.pack", Stage: packScratchName("stage", "payments.pack"), Backup: packScratchName("backup", "payments.pack"), Retire: packScratchName("retire", "payments.pack"), Existed: true},
		{Target: "shipping.pack", Stage: packScratchName("stage", "shipping.pack"), Backup: packScratchName("backup", "shipping.pack"), Retire: packScratchName("retire", "shipping.pack"), Existed: false},
	}
	switch phase {
	case "prepared":
		writePackRecoveryDir(t, root, "orders.pack", "old-orders")
		writePackRecoveryDir(t, root, "payments.pack", "old-payments")
		for _, entry := range entries {
			writePackRecoveryDir(t, root, entry.Stage, "new-"+entry.Target)
		}
	case "parking":
		writePackRecoveryDir(t, root, entries[0].Backup, "old-orders")
		writePackRecoveryDir(t, root, "payments.pack", "old-payments")
		for _, entry := range entries {
			writePackRecoveryDir(t, root, entry.Stage, "new-"+entry.Target)
		}
	case "installing":
		writePackRecoveryDir(t, root, "orders.pack", "new-orders")
		writePackRecoveryDir(t, root, "payments.pack", "new-payments")
		writePackRecoveryDir(t, root, "shipping.pack", "new-shipping")
		writePackRecoveryDir(t, root, entries[0].Backup, "old-orders")
		writePackRecoveryDir(t, root, entries[1].Backup, "old-payments")
	case "committed":
		writePackRecoveryDir(t, root, "orders.pack", "new-orders")
		writePackRecoveryDir(t, root, "payments.pack", "new-payments")
		writePackRecoveryDir(t, root, "shipping.pack", "new-shipping")
		writePackRecoveryDir(t, root, entries[0].Backup, "old-orders")
		writePackRecoveryDir(t, root, entries[1].Backup, "old-payments")
	default:
		t.Fatalf("unknown crash phase %s", phase)
	}
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := rootHandle.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	staged := make([]packStagedEntry, 0, len(entries))
	for i := range entries {
		entry := &entries[i]
		entry.Before = packTreeWitness{Tree: packAbsentTree}
		if entry.Existed {
			beforeName := entry.Target
			if _, err := rootHandle.Lstat(entry.Backup); err == nil {
				beforeName = entry.Backup
			}
			state, err := capturePackTree(rootHandle, beforeName)
			if err != nil {
				t.Fatal(err)
			}
			entry.Before = state.witness
		}
		afterName := entry.Stage
		if _, err := rootHandle.Lstat(afterName); os.IsNotExist(err) {
			afterName = entry.Target
		}
		after, err := capturePackTree(rootHandle, afterName)
		if err != nil {
			t.Fatal(err)
		}
		entry.AfterTree = after.witness.Tree
		entry.after = after.witness
		staged = append(staged, packStagedEntry{Target: entry.Target, After: after.witness})
	}
	if err := createPackJournal(rootHandle, entries); err != nil {
		t.Fatal(err)
	}
	for _, entry := range staged {
		if entry.After.Tree == packAbsentTree {
			continue
		}
		if err := appendPackStage(rootHandle, entry); err != nil {
			t.Fatal(err)
		}
	}
	for _, recorded := range []string{"staged", "parking", "installing", "committed"} {
		if phase == "prepared" {
			break
		}
		var err error
		if recorded == "staged" {
			err = appendPackStaged(rootHandle, staged)
		} else {
			err = appendPackPhase(rootHandle, recorded)
		}
		if err != nil {
			t.Fatal(err)
		}
		if phase == recorded {
			break
		}
	}
	return entries
}

func TestPackJournalRecoversEveryCrashPhaseOnLockAcquisition(t *testing.T) {
	for _, phase := range []string{"prepared", "parking", "installing", "committed"} {
		t.Run(phase, func(t *testing.T) {
			root := t.TempDir()
			entries := seedPackCrash(t, root, phase)
			lock, err := acquirePackWriteLock(root)
			if err != nil {
				t.Fatal(err)
			}
			if err := lock.releaseAll(); err != nil {
				t.Fatal(err)
			}
			want := map[string]string{"orders.pack": "old-orders", "payments.pack": "old-payments"}
			if phase == "committed" {
				want = map[string]string{"orders.pack": "new-orders", "payments.pack": "new-payments", "shipping.pack": "new-shipping"}
			}
			for name, body := range want {
				got, err := os.ReadFile(filepath.Join(root, name, "marker"))
				if err != nil || string(got) != body {
					t.Fatalf("%s after %s recovery = %q, %v; want %q", name, phase, got, err, body)
				}
			}
			if phase != "committed" {
				if _, err := os.Lstat(filepath.Join(root, "shipping.pack")); !os.IsNotExist(err) {
					t.Fatalf("new pack survived uncommitted %s recovery: %v", phase, err)
				}
			}
			for _, entry := range entries {
				for _, scratch := range []string{entry.Stage, entry.Backup} {
					if _, err := os.Lstat(filepath.Join(root, scratch)); !os.IsNotExist(err) {
						t.Fatalf("%s survived recovery: %v", scratch, err)
					}
				}
			}
			if _, err := os.Lstat(filepath.Join(root, packJournalName)); !os.IsNotExist(err) {
				t.Fatalf("journal survived recovery: %v", err)
			}
		})
	}
}

func TestPackPreparedRecoveryPreservesForeignStageMutationReplacementAndABA(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string)
		want   string
	}{
		{
			name: "content mutation",
			mutate: func(t *testing.T, stage string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(stage, "marker"), []byte("foreign content"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			want: "foreign content",
		},
		{
			name: "foreign replacement",
			mutate: func(t *testing.T, stage string) {
				t.Helper()
				if err := os.RemoveAll(stage); err != nil {
					t.Fatal(err)
				}
				writePackRecoveryDir(t, filepath.Dir(stage), filepath.Base(stage), "foreign replacement")
			},
			want: "foreign replacement",
		},
		{
			name: "same-byte ABA",
			mutate: func(t *testing.T, stage string) {
				t.Helper()
				if err := os.RemoveAll(stage); err != nil {
					t.Fatal(err)
				}
				writePackRecoveryDir(t, filepath.Dir(stage), filepath.Base(stage), "new-payments.pack")
			},
			want: "new-payments.pack",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			entries := seedPackCrash(t, root, "prepared")
			stage := filepath.Join(root, entries[1].Stage)
			test.mutate(t, stage)

			if _, err := acquirePackWriteLock(root); err == nil || !strings.Contains(err.Error(), "preserving it") {
				t.Fatalf("prepared recovery did not preserve foreign stage: %v", err)
			}
			if got, err := os.ReadFile(filepath.Join(stage, "marker")); err != nil || string(got) != test.want {
				t.Fatalf("foreign stage was changed or deleted: got %q, err %v", got, err)
			}
			for _, entry := range entries {
				if _, err := os.Lstat(filepath.Join(root, entry.Stage)); err != nil {
					t.Fatalf("whole-set preflight removed stage %s: %v", entry.Stage, err)
				}
			}
			for target, want := range map[string]string{"orders.pack": "old-orders", "payments.pack": "old-payments"} {
				if got, err := os.ReadFile(filepath.Join(root, target, "marker")); err != nil || string(got) != want {
					t.Fatalf("prepared recovery changed %s: got %q, err %v", target, got, err)
				}
			}
			if _, err := os.Lstat(filepath.Join(root, packJournalName)); err != nil {
				t.Fatalf("prepared recovery removed authoritative journal: %v", err)
			}
		})
	}
}

func TestPackPreparedRecoveryPreservesUnwitnessedPartialStage(t *testing.T) {
	root := t.TempDir()
	writePackRecoveryDir(t, root, "orders.pack", "old-orders")
	writePackRecoveryDir(t, root, packScratchName("stage", "orders.pack"), "partial-stage")
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	before, err := capturePackTree(rootHandle, "orders.pack")
	if err != nil {
		t.Fatal(err)
	}
	entry := packJournalEntry{
		Target: "orders.pack", Stage: packScratchName("stage", "orders.pack"), Backup: packScratchName("backup", "orders.pack"), Retire: packScratchName("retire", "orders.pack"),
		Existed: true, Before: before.witness, AfterTree: generatedPackTreeDigest([]string{"marker"}, map[string]string{"marker": "intended-stage"}),
	}
	if err := createPackJournal(rootHandle, []packJournalEntry{entry}); err != nil {
		t.Fatal(err)
	}
	if err := rootHandle.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := acquirePackWriteLock(root); err == nil || !strings.Contains(err.Error(), "no durable exact witness; preserving it") {
		t.Fatalf("unwitnessed partial stage was not a recovery blocker: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(root, entry.Stage, "marker")); err != nil || string(got) != "partial-stage" {
		t.Fatalf("unwitnessed partial stage was changed: got %q, err %v", got, err)
	}
	if _, err := os.Lstat(filepath.Join(root, packJournalName)); err != nil {
		t.Fatalf("journal was removed despite unwitnessed stage: %v", err)
	}
}

func TestPackStageCleanupPreservesLateDeletionBoundaryMutationAndABA(t *testing.T) {
	for _, phase := range []string{"prepared", "parking"} {
		for _, test := range []struct {
			name   string
			mutate func(*testing.T, string)
			want   string
		}{
			{
				name: "content mutation",
				mutate: func(t *testing.T, retirement string) {
					t.Helper()
					if err := os.WriteFile(filepath.Join(retirement, "marker"), []byte("late stage edit"), 0o644); err != nil {
						t.Fatal(err)
					}
				},
				want: "late stage edit",
			},
			{
				name: "same-byte ABA",
				mutate: func(t *testing.T, retirement string) {
					t.Helper()
					if err := os.RemoveAll(retirement); err != nil {
						t.Fatal(err)
					}
					writePackRecoveryDir(t, filepath.Dir(retirement), filepath.Base(retirement), "new-payments.pack")
				},
				want: "new-payments.pack",
			},
		} {
			t.Run(phase+" "+test.name, func(t *testing.T) {
				root := t.TempDir()
				entries := seedPackCrash(t, root, phase)
				entry := entries[1]
				prior := packTreeRemovalPoint
				packTreeRemovalPoint = func(name, retirement string) error {
					if name == entry.Stage {
						test.mutate(t, filepath.Join(root, retirement))
					}
					return nil
				}
				_, err := acquirePackWriteLock(root)
				packTreeRemovalPoint = prior
				t.Cleanup(func() { packTreeRemovalPoint = prior })
				if err == nil || !strings.Contains(err.Error(), "deletion boundary") || !strings.Contains(err.Error(), "preserving it") {
					t.Fatalf("late stage mutation was not a preservation blocker: %v", err)
				}
				if got, readErr := os.ReadFile(filepath.Join(root, entry.Stage, "marker")); readErr != nil || string(got) != test.want {
					t.Fatalf("late-mutated stage was not restored for inspection: got %q, err %v", got, readErr)
				}
				if _, statErr := os.Lstat(filepath.Join(root, packJournalName)); statErr != nil {
					t.Fatalf("journal was removed after deletion-boundary ambiguity: %v", statErr)
				}
			})
		}
	}
}

func TestPackRestartRecoveryPreservesLateMutatedInstalledTarget(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(string) error
		want   string
	}{
		{
			name: "content mutation",
			mutate: func(path string) error {
				return os.WriteFile(path, []byte("user edit after crash"), 0o644)
			},
			want: "user edit after crash",
		},
		{
			name: "ABA identity mutation",
			mutate: func(path string) error {
				if err := os.Remove(path); err != nil {
					return err
				}
				return os.WriteFile(path, []byte("new-orders"), 0o644)
			},
			want: "new-orders",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			entries := seedPackCrash(t, root, "installing")
			target := filepath.Join(root, "orders.pack", "marker")
			if err := test.mutate(target); err != nil {
				t.Fatal(err)
			}
			if _, err := acquirePackWriteLock(root); err == nil || !strings.Contains(err.Error(), "preserving it") {
				t.Fatalf("restart recovery did not fail closed on late mutation: %v", err)
			}
			got, err := os.ReadFile(target)
			if err != nil || string(got) != test.want {
				t.Fatalf("restart recovery changed late-mutated target: got %q, err %v; want %q", got, err, test.want)
			}
			backup := filepath.Join(root, entries[0].Backup, "marker")
			if got, err := os.ReadFile(backup); err != nil || string(got) != "old-orders" {
				t.Fatalf("restart recovery changed usable backup: got %q, err %v", got, err)
			}
			if _, err := os.Lstat(filepath.Join(root, packJournalName)); err != nil {
				t.Fatalf("restart recovery removed journal after preservation blocker: %v", err)
			}
		})
	}
}

func TestPackRestartRecoveryConvergesAfterBackupWasAlreadyRestored(t *testing.T) {
	root := t.TempDir()
	entries := seedPackCrash(t, root, "installing")
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := rootHandle.RemoveAll(entries[0].Target); err != nil {
		t.Fatal(err)
	}
	if err := rootHandle.Rename(entries[0].Backup, entries[0].Target); err != nil {
		t.Fatal(err)
	}
	if err := rootHandle.Close(); err != nil {
		t.Fatal(err)
	}

	lock, err := acquirePackWriteLock(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.releaseAll(); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"orders.pack": "old-orders", "payments.pack": "old-payments"} {
		got, err := os.ReadFile(filepath.Join(root, name, "marker"))
		if err != nil || string(got) != want {
			t.Fatalf("%s after resumed rollback = %q, %v; want %q", name, got, err, want)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, "shipping.pack")); !os.IsNotExist(err) {
		t.Fatalf("new target survived resumed rollback: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, packJournalName)); !os.IsNotExist(err) {
		t.Fatalf("journal survived completed resumed rollback: %v", err)
	}
}

func TestPackRecoveryStaysOnOpenedRootDuringParentSwap(t *testing.T) {
	base := t.TempDir()
	rootPath := filepath.Join(base, "packs")
	held := filepath.Join(base, "held-packs")
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(rootPath, 0o755); err != nil {
		t.Fatal(err)
	}
	seedPackCrash(t, rootPath, "parking")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	swapped := false
	rename := func(opened *os.Root, old, new string) error {
		if !swapped {
			swapped = true
			if err := os.Rename(rootPath, held); err != nil {
				return err
			}
			if err := os.Symlink(outside, rootPath); err != nil {
				return err
			}
		}
		return nil
	}
	if err := recoverPackTransaction(root, rename); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(held, "orders.pack", "marker"))
	if err != nil || string(got) != "old-orders" {
		t.Fatalf("rooted recovery restored %q, %v", got, err)
	}
	got, err = os.ReadFile(sentinel)
	if err != nil || string(got) != "outside" {
		t.Fatalf("outside sentinel changed: %q, %v", got, err)
	}
}

func TestPackJournalBytesAreDeterministic(t *testing.T) {
	witness := packTreeWitness{Tree: "sha256:" + strings.Repeat("0", 64), Identity: "sha256:" + strings.Repeat("1", 64)}
	entries := []packJournalEntry{
		{Target: "orders.pack", Stage: packScratchName("stage", "orders.pack"), Backup: packScratchName("backup", "orders.pack"), Retire: packScratchName("retire", "orders.pack"), Existed: true, Before: witness, AfterTree: witness.Tree},
		{Target: "payments.pack", Stage: packScratchName("stage", "payments.pack"), Backup: packScratchName("backup", "payments.pack"), Retire: packScratchName("retire", "payments.pack"), Before: packTreeWitness{Tree: packAbsentTree}, AfterTree: witness.Tree},
	}
	var bodies [][]byte
	for i := 0; i < 2; i++ {
		root := t.TempDir()
		rootHandle, err := os.OpenRoot(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := createPackJournal(rootHandle, entries); err != nil {
			t.Fatal(err)
		}
		staged := []packStagedEntry{{Target: "orders.pack", After: witness}, {Target: "payments.pack", After: witness}}
		if err := appendPackStaged(rootHandle, staged); err != nil {
			t.Fatal(err)
		}
		if err := appendPackPhase(rootHandle, "parking"); err != nil {
			t.Fatal(err)
		}
		if err := rootHandle.Close(); err != nil {
			t.Fatal(err)
		}
		body, err := os.ReadFile(filepath.Join(root, packJournalName))
		if err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, body)
	}
	if !bytes.Equal(bodies[0], bodies[1]) {
		t.Fatalf("same pack transaction produced different journals:\n%s\n%s", bodies[0], bodies[1])
	}
}

func replacePackJournalForTest(t *testing.T, root, body string) {
	t.Helper()
	replacePackJournalPathForTest(t, root, packJournalName, body)
}

func replacePackJournalPathForTest(t *testing.T, root, name, body string) {
	t.Helper()
	replacement := filepath.Join(root, ".pack-journal-replacement")
	if err := os.WriteFile(replacement, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, filepath.Join(root, name)); err != nil {
		t.Fatal(err)
	}
}

func testPackJournalEntry() packJournalEntry {
	target := "orders.pack"
	return packJournalEntry{
		Target: target, Stage: packScratchName("stage", target), Backup: packScratchName("backup", target), Retire: packScratchName("retire", target),
		Existed: false, Before: packTreeWitness{Tree: packAbsentTree}, AfterTree: "sha256:" + strings.Repeat("1", sha256.Size*2),
	}
}

func TestAppendPackJournalRejectsAfterOpenAndAfterSyncPathABA(t *testing.T) {
	for _, point := range []string{"append-after-open", "append-after-sync"} {
		t.Run(point, func(t *testing.T) {
			root := t.TempDir()
			rootHandle, err := os.OpenRoot(root)
			if err != nil {
				t.Fatal(err)
			}
			defer rootHandle.Close()
			if err := createPackJournal(rootHandle, []packJournalEntry{testPackJournalEntry()}); err != nil {
				t.Fatal(err)
			}
			prior := packJournalPoint
			packJournalPoint = func(got string) error {
				if got == point {
					body, readErr := os.ReadFile(filepath.Join(root, packJournalName))
					if readErr != nil {
						t.Fatal(readErr)
					}
					replacePackJournalForTest(t, root, string(body))
				}
				return nil
			}
			err = appendPackPhase(rootHandle, "parking")
			packJournalPoint = prior
			t.Cleanup(func() { packJournalPoint = prior })
			if err == nil || !strings.Contains(err.Error(), "journal") || !strings.Contains(err.Error(), "authority") && !strings.Contains(err.Error(), "changed") {
				t.Fatalf("journal ABA at %s was accepted: %v", point, err)
			}
			if _, statErr := os.Lstat(filepath.Join(root, packJournalName)); statErr != nil {
				t.Fatalf("replacement journal was not preserved: %v", statErr)
			}
		})
	}
}

func TestWritePacksRejectsJournalABABeforePublicRename(t *testing.T) {
	design := t.TempDir()
	root := filepath.Join(design, "packs")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	writePackRecoveryDir(t, root, "orders.pack", "old-orders")
	prior := packJournalPoint
	packJournalPoint = func(point string) error {
		if strings.HasPrefix(point, "before-public-rename:") {
			body, err := os.ReadFile(filepath.Join(root, packJournalName))
			if err != nil {
				t.Fatal(err)
			}
			replacePackJournalForTest(t, root, string(body))
		}
		return nil
	}
	renameCalled := false
	rename := func(opened *os.Root, oldName, newName string) error {
		renameCalled = true
		return nil
	}
	_, err := writePacksWithRename(design, map[string]map[string]string{"orders": {"marker": "new-orders"}}, rename)
	packJournalPoint = prior
	t.Cleanup(func() { packJournalPoint = prior })
	if err == nil || !strings.Contains(err.Error(), "journal authority changed") || !strings.Contains(err.Error(), "preserving live state") {
		t.Fatalf("pre-publication journal ABA was accepted: %v", err)
	}
	if renameCalled {
		t.Fatal("public pack rename ran after journal authority was lost")
	}
	if got, readErr := os.ReadFile(filepath.Join(root, "orders.pack", "marker")); readErr != nil || string(got) != "old-orders" {
		t.Fatalf("live target changed after journal ABA: got %q, err %v", got, readErr)
	}
	if _, statErr := os.Lstat(filepath.Join(root, packJournalName)); statErr != nil {
		t.Fatalf("replacement journal was not preserved: %v", statErr)
	}
}

func TestPackRecoveryRejectsJournalReplacementAfterParse(t *testing.T) {
	root := t.TempDir()
	seedPackCrash(t, root, "prepared")
	before, err := os.Lstat(filepath.Join(root, packJournalName))
	if err != nil {
		t.Fatal(err)
	}
	prior := packJournalPoint
	packJournalPoint = func(point string) error {
		if point == "recovery-after-journal-read" {
			body, readErr := os.ReadFile(filepath.Join(root, packJournalName))
			if readErr != nil {
				t.Fatal(readErr)
			}
			replacePackJournalForTest(t, root, string(body))
		}
		return nil
	}
	t.Cleanup(func() { packJournalPoint = prior })
	_, err = acquirePackWriteLock(root)
	packJournalPoint = prior
	if err == nil || !strings.Contains(err.Error(), "journal changed") || !strings.Contains(err.Error(), "preserving") {
		t.Fatalf("post-parse journal replacement was accepted: %v", err)
	}
	after, statErr := os.Lstat(filepath.Join(root, packJournalName))
	if statErr != nil || os.SameFile(before, after) {
		t.Fatalf("replacement journal was not preserved: before=%v after=%v err=%v", before, after, statErr)
	}
	if got, readErr := os.ReadFile(filepath.Join(root, "orders.pack", "marker")); readErr != nil || string(got) != "old-orders" {
		t.Fatalf("stale parsed journal drove recovery: got %q, err %v", got, readErr)
	}
}

func TestPackJournalIsolationNeverOverwritesDestinationCollision(t *testing.T) {
	root := t.TempDir()
	seedPackCrash(t, root, "prepared")
	prior := packJournalPoint
	packJournalPoint = func(point string) error {
		if point == "recovery-before-journal-isolate" {
			return os.WriteFile(filepath.Join(root, packJournalRetirement), []byte("foreign destination\n"), 0o600)
		}
		return nil
	}
	t.Cleanup(func() { packJournalPoint = prior })
	_, err := acquirePackWriteLock(root)
	packJournalPoint = prior
	if err == nil {
		t.Fatal("journal retirement destination collision was accepted")
	}
	if _, statErr := os.Lstat(filepath.Join(root, packJournalName)); statErr != nil {
		t.Fatalf("source journal authority was lost: %v", statErr)
	}
	if got, readErr := os.ReadFile(filepath.Join(root, packJournalRetirement)); readErr != nil || string(got) != "foreign destination\n" {
		t.Fatalf("journal retirement collision was overwritten: %q, %v", got, readErr)
	}
}

func TestPackTreeRetirementNeverOverwritesDestinationCollision(t *testing.T) {
	root := t.TempDir()
	seedPackCrash(t, root, "prepared")
	prior := packTreeRetirementPoint
	var source, retirement string
	packTreeRetirementPoint = func(gotSource, gotRetirement string) error {
		if retirement != "" {
			return nil
		}
		source, retirement = gotSource, gotRetirement
		writePackRecoveryDir(t, root, retirement, "foreign destination")
		return nil
	}
	t.Cleanup(func() { packTreeRetirementPoint = prior })
	_, err := acquirePackWriteLock(root)
	packTreeRetirementPoint = prior
	if err == nil || source == "" || retirement == "" {
		t.Fatalf("tree retirement destination collision was not rejected: source=%q retirement=%q err=%v", source, retirement, err)
	}
	if _, statErr := os.Lstat(filepath.Join(root, source)); statErr != nil {
		t.Fatalf("tree retirement source was lost: %v", statErr)
	}
	if got, readErr := os.ReadFile(filepath.Join(root, retirement, "marker")); readErr != nil || string(got) != "foreign destination" {
		t.Fatalf("tree retirement collision was overwritten: %q, %v", got, readErr)
	}
}

func TestPackBackupRestoreNeverOverwritesBoundaryCollision(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	const backup = ".machinery-pack-backup-orders"
	if err := root.Mkdir(backup, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, backup, "pack.yaml"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := capturePackTree(root, backup)
	if err != nil {
		t.Fatal(err)
	}
	entry := packJournalEntry{Target: "orders.pack", Backup: backup, Before: before.witness}
	called := false
	rename := func(root *os.Root, oldName, newName string) error {
		called = true
		if err := root.Mkdir(newName, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, newName, "foreign"), []byte("concurrent"), 0o600); err != nil {
			return err
		}
		return nil
	}
	err = restorePackBackup(root, rename, entry)
	if err == nil || !called {
		t.Fatalf("backup restoration collision was not rejected: called=%v err=%v", called, err)
	}
	for path, want := range map[string]string{
		filepath.Join(dir, backup, "pack.yaml"):      "old",
		filepath.Join(dir, "orders.pack", "foreign"): "concurrent",
	} {
		body, readErr := os.ReadFile(path)
		if readErr != nil || string(body) != want {
			t.Fatalf("%s = %q, %v; want %q", path, body, readErr, want)
		}
	}
}

func TestPackQuarantineDeletionCannotDeletePublicReplacement(t *testing.T) {
	for _, tc := range []struct {
		name    string
		install func(root string) func()
	}{
		{
			name: "tree",
			install: func(root string) func() {
				prior := packTreeQuarantinePoint
				packTreeQuarantinePoint = func(retirement string) error {
					name := findPackTestEntry(t, root, func(name string) bool { return validPackQuarantineName(name, packTreeQuarantinePrefix) })
					return replacePackTestQuarantine(root, name)
				}
				return func() { packTreeQuarantinePoint = prior }
			},
		},
		{
			name: "journal",
			install: func(root string) func() {
				prior := packJournalPoint
				packJournalPoint = func(point string) error {
					if point != "recovery-before-journal-quarantine-remove" {
						return nil
					}
					name := findPackTestEntry(t, root, func(name string) bool { return validPackQuarantineName(name, packJournalQuarantinePrefix) })
					return replacePackTestQuarantine(root, name)
				}
				return func() { packJournalPoint = prior }
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			seedPackCrash(t, root, "committed")
			restore := tc.install(root)
			t.Cleanup(restore)
			_, err := acquirePackWriteLock(root)
			restore()
			if err == nil {
				t.Fatal("public quarantine replacement did not block namespace retirement")
			}
			name := findPackTestEntry(t, root, func(name string) bool {
				body, readErr := os.ReadFile(filepath.Join(root, name, "object"))
				return readErr == nil && string(body) == "foreign replacement\n"
			})
			if got, readErr := os.ReadFile(filepath.Join(root, name, "object")); readErr != nil || string(got) != "foreign replacement\n" {
				t.Fatalf("public quarantine replacement was deleted: %q, %v", got, readErr)
			}
		})
	}
}

func TestPackRecoveryResumesJournalAndTreeQuarantines(t *testing.T) {
	for _, tc := range []struct {
		name, phase string
		install     func(error) func(string) error
	}{
		{
			name: "journal", phase: "committed",
			install: func(injected error) func(string) error {
				return func(point string) error {
					if point == "recovery-after-journal-quarantine" {
						return injected
					}
					return nil
				}
			},
		},
		{
			name: "tree", phase: "prepared",
			install: func(injected error) func(string) error {
				return func(string) error { return injected }
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			seedPackCrash(t, root, tc.phase)
			injected := errors.New("crash after quarantine")
			priorJournal, priorTree := packJournalPoint, packTreeQuarantinePoint
			if tc.name == "journal" {
				packJournalPoint = tc.install(injected)
			} else {
				packTreeQuarantinePoint = tc.install(injected)
			}
			_, err := acquirePackWriteLock(root)
			packJournalPoint, packTreeQuarantinePoint = priorJournal, priorTree
			t.Cleanup(func() { packJournalPoint, packTreeQuarantinePoint = priorJournal, priorTree })
			if !errors.Is(err, injected) {
				t.Fatalf("quarantine crash was not reported: %v", err)
			}
			lock, err := acquirePackWriteLock(root)
			if err != nil {
				t.Fatalf("resume quarantine: %v", err)
			}
			if err := lock.releaseAll(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func findPackTestEntry(t *testing.T, root string, match func(string) bool) string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if match(entry.Name()) {
			return entry.Name()
		}
	}
	t.Fatal("matching pack test entry not found")
	return ""
}

func replacePackTestQuarantine(root, name string) error {
	public := filepath.Join(root, name)
	held := filepath.Join(root, name+"-held")
	if err := os.Rename(public, held); err != nil {
		return err
	}
	if err := os.Mkdir(public, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(public, "object"), []byte("foreign replacement\n"), 0o600)
}

func TestPackRecoveryRejectsChangedIsolatedJournalBeforeDestructiveAction(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{
			name: "content mutation",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
				if err != nil {
					t.Fatal(err)
				}
				_, writeErr := file.WriteString("foreign mutation")
				if err := errors.Join(writeErr, file.Close()); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "path ABA",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				replacePackJournalPathForTest(t, filepath.Dir(path), filepath.Base(path), string(body))
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			seedPackCrash(t, root, "prepared")
			prior := packJournalPoint
			packJournalPoint = func(point string) error {
				if point == "recovery-before-remove-stage-shipping.pack" {
					test.mutate(t, filepath.Join(root, packJournalRetirement))
				}
				return nil
			}
			t.Cleanup(func() { packJournalPoint = prior })
			_, err := acquirePackWriteLock(root)
			packJournalPoint = prior
			if err == nil || !strings.Contains(err.Error(), "isolated pack recovery journal changed") || !strings.Contains(err.Error(), "preserving") {
				t.Fatalf("changed isolated journal was accepted: %v", err)
			}
			if _, statErr := os.Lstat(filepath.Join(root, packJournalRetirement)); statErr != nil {
				t.Fatalf("changed isolated journal was not preserved: %v", statErr)
			}
			if got, readErr := os.ReadFile(filepath.Join(root, "orders.pack", "marker")); readErr != nil || string(got) != "old-orders" {
				t.Fatalf("changed authority drove recovery: got %q, err %v", got, readErr)
			}
		})
	}
}

func TestPackRecoveryRejectsReappearingJournalPath(t *testing.T) {
	root := t.TempDir()
	seedPackCrash(t, root, "prepared")
	prior := packJournalPoint
	packJournalPoint = func(point string) error {
		if point == "recovery-before-remove-stage-shipping.pack" {
			if err := os.WriteFile(filepath.Join(root, packJournalName), []byte("foreign replacement\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return nil
	}
	t.Cleanup(func() { packJournalPoint = prior })
	_, err := acquirePackWriteLock(root)
	packJournalPoint = prior
	if err == nil || !strings.Contains(err.Error(), "journal path reappeared") || !strings.Contains(err.Error(), "preserving") {
		t.Fatalf("reappearing journal path was accepted: %v", err)
	}
	for _, name := range []string{packJournalName, packJournalRetirement} {
		if _, statErr := os.Lstat(filepath.Join(root, name)); statErr != nil {
			t.Fatalf("journal authority %s was not preserved: %v", name, statErr)
		}
	}
	if got, readErr := os.ReadFile(filepath.Join(root, "orders.pack", "marker")); readErr != nil || string(got) != "old-orders" {
		t.Fatalf("replacement journal drove recovery: got %q, err %v", got, readErr)
	}
}

func TestPackRecoveryConditionallyRemovesExactIsolatedJournal(t *testing.T) {
	root := t.TempDir()
	seedPackCrash(t, root, "committed")
	prior := packJournalPoint
	packJournalPoint = func(point string) error {
		if point == "recovery-before-journal-remove" {
			path := filepath.Join(root, packJournalRetirement)
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			replacePackJournalPathForTest(t, root, packJournalRetirement, string(body))
		}
		return nil
	}
	t.Cleanup(func() { packJournalPoint = prior })
	_, err := acquirePackWriteLock(root)
	packJournalPoint = prior
	if err == nil || !strings.Contains(err.Error(), "isolated pack recovery journal changed") || !strings.Contains(err.Error(), "preserving") {
		t.Fatalf("final journal replacement was removed or accepted: %v", err)
	}
	if _, statErr := os.Lstat(filepath.Join(root, packJournalRetirement)); statErr != nil {
		t.Fatalf("final replacement journal was not preserved: %v", statErr)
	}
}

func TestPackRecoveryResumesAnExactlyIsolatedJournalAfterCrash(t *testing.T) {
	root := t.TempDir()
	seedPackCrash(t, root, "prepared")
	injected := errors.New("injected crash after journal isolation")
	prior := packJournalPoint
	packJournalPoint = func(point string) error {
		if point == "recovery-after-journal-isolate" {
			return injected
		}
		return nil
	}
	_, err := acquirePackWriteLock(root)
	packJournalPoint = prior
	t.Cleanup(func() { packJournalPoint = prior })
	if !errors.Is(err, injected) {
		t.Fatalf("journal isolation crash was not reported: %v", err)
	}
	if _, statErr := os.Lstat(filepath.Join(root, packJournalName)); !os.IsNotExist(statErr) {
		t.Fatalf("live journal survived atomic isolation: %v", statErr)
	}
	if _, statErr := os.Lstat(filepath.Join(root, packJournalRetirement)); statErr != nil {
		t.Fatalf("isolated authority was not durable after crash: %v", statErr)
	}
	lock, err := acquirePackWriteLock(root)
	if err != nil {
		t.Fatalf("resuming isolated recovery failed: %v", err)
	}
	if err := lock.releaseAll(); err != nil {
		t.Fatal(err)
	}
	if got, readErr := os.ReadFile(filepath.Join(root, "orders.pack", "marker")); readErr != nil || string(got) != "old-orders" {
		t.Fatalf("resumed recovery restored %q, %v", got, readErr)
	}
	for _, name := range []string{packJournalName, packJournalRetirement} {
		if _, statErr := os.Lstat(filepath.Join(root, name)); !os.IsNotExist(statErr) {
			t.Fatalf("journal path %s survived resumed recovery: %v", name, statErr)
		}
	}
}

func TestCreatePackJournalFailurePreservesReplacement(t *testing.T) {
	injected := errors.New("injected journal persistence failure")
	for _, operation := range []string{"write", "sync", "close"} {
		t.Run(operation, func(t *testing.T) {
			root := t.TempDir()
			rootHandle, err := os.OpenRoot(root)
			if err != nil {
				t.Fatal(err)
			}
			defer rootHandle.Close()
			priorWrite, priorSync, priorClose := packJournalWrite, packJournalSync, packJournalClose
			replace := func() { replacePackJournalForTest(t, root, "foreign replacement\n") }
			switch operation {
			case "write":
				packJournalWrite = func(*os.File, []byte) (int, error) {
					replace()
					return 0, injected
				}
			case "sync":
				packJournalSync = func(*os.File) error {
					replace()
					return injected
				}
			case "close":
				packJournalClose = func(file *os.File) error {
					closeErr := file.Close()
					replace()
					return errors.Join(closeErr, injected)
				}
			}
			err = createPackJournal(rootHandle, []packJournalEntry{testPackJournalEntry()})
			packJournalWrite, packJournalSync, packJournalClose = priorWrite, priorSync, priorClose
			t.Cleanup(func() {
				packJournalWrite, packJournalSync, packJournalClose = priorWrite, priorSync, priorClose
			})
			if !errors.Is(err, injected) || !strings.Contains(err.Error(), "preserving replacement") {
				t.Fatalf("%s failure did not preserve and report replacement: %v", operation, err)
			}
			if body, readErr := os.ReadFile(filepath.Join(root, packJournalName)); readErr != nil || string(body) != "foreign replacement\n" {
				t.Fatalf("%s failure removed replacement journal: got %q, err %v", operation, body, readErr)
			}
		})
	}
}

func TestPackJournalRecoversPastTornPhaseRecord(t *testing.T) {
	root := t.TempDir()
	seedPackCrash(t, root, "prepared")
	f, err := os.OpenFile(filepath.Join(root, packJournalName), os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"phase":"stag`); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	lock, err := acquirePackWriteLock(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.releaseAll(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "orders.pack", "marker"))
	if err != nil || string(got) != "old-orders" {
		t.Fatalf("torn record recovery restored %q, %v", got, err)
	}
}

func TestPackJournalRejectsUnsafeAuthority(t *testing.T) {
	t.Run("symlink journal", func(t *testing.T) {
		root := t.TempDir()
		outside := filepath.Join(t.TempDir(), "sentinel")
		if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(root, packJournalName)); err != nil {
			t.Fatal(err)
		}
		if _, err := acquirePackWriteLock(root); err == nil || !strings.Contains(err.Error(), "non-symlink") {
			t.Fatalf("symlink journal accepted: %v", err)
		}
		got, _ := os.ReadFile(outside)
		if string(got) != "outside" {
			t.Fatalf("outside sentinel changed: %q", got)
		}
	})
	t.Run("special journal", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, packJournalName), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := acquirePackWriteLock(root); err == nil || !strings.Contains(err.Error(), "regular") {
			t.Fatalf("directory journal accepted: %v", err)
		}
	})
	t.Run("malformed journal", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, packJournalName), []byte("not-json\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := acquirePackWriteLock(root); err == nil || !strings.Contains(err.Error(), "malformed") {
			t.Fatalf("malformed journal accepted: %v", err)
		}
	})
	t.Run("duplicate journal key", func(t *testing.T) {
		root := t.TempDir()
		body := `{"version":1,"version":1,"phase":"prepared","entries":[]}` + "\n"
		if err := os.WriteFile(filepath.Join(root, packJournalName), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := acquirePackWriteLock(root); err == nil || !strings.Contains(err.Error(), "malformed") {
			t.Fatalf("duplicate journal key accepted: %v", err)
		}
	})
	t.Run("case-aliased journal key", func(t *testing.T) {
		root := t.TempDir()
		body := `{"Version":1,"phase":"prepared","entries":[]}` + "\n"
		if err := os.WriteFile(filepath.Join(root, packJournalName), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := acquirePackWriteLock(root); err == nil || !strings.Contains(err.Error(), "malformed") {
			t.Fatalf("case-aliased journal key accepted: %v", err)
		}
	})
	t.Run("portable aliases", func(t *testing.T) {
		root := t.TempDir()
		witness := packTreeWitness{Tree: "sha256:" + strings.Repeat("0", 64), Identity: "sha256:" + strings.Repeat("1", 64)}
		entries := []packJournalEntry{
			{Target: "orders.pack", Stage: packScratchName("stage", "orders.pack"), Backup: packScratchName("backup", "orders.pack"), Retire: packScratchName("retire", "orders.pack"), Existed: true, Before: witness, AfterTree: witness.Tree},
			{Target: "Orders.pack", Stage: packScratchName("stage", "Orders.pack"), Backup: packScratchName("backup", "Orders.pack"), Retire: packScratchName("retire", "Orders.pack"), Existed: true, Before: witness, AfterTree: witness.Tree},
		}
		body, _ := encodePackRecord(packJournalHeader{Version: 2, Phase: "prepared", Entries: entries})
		if err := os.WriteFile(filepath.Join(root, packJournalName), body, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := acquirePackWriteLock(root); err == nil || !strings.Contains(err.Error(), "alias") {
			t.Fatalf("portable alias journal accepted: %v", err)
		}
	})
	t.Run("outside path", func(t *testing.T) {
		base := t.TempDir()
		root := filepath.Join(base, "packs")
		if err := os.Mkdir(root, 0o755); err != nil {
			t.Fatal(err)
		}
		outside := filepath.Join(base, "outside")
		if err := os.WriteFile(outside, []byte("sentinel"), 0o644); err != nil {
			t.Fatal(err)
		}
		target := "../outside"
		witness := packTreeWitness{Tree: "sha256:" + strings.Repeat("0", 64), Identity: "sha256:" + strings.Repeat("1", 64)}
		entry := packJournalEntry{Target: target, Stage: packScratchName("stage", target), Backup: packScratchName("backup", target), Retire: packScratchName("retire", target), Existed: true, Before: witness, AfterTree: witness.Tree}
		body, _ := encodePackRecord(packJournalHeader{Version: 2, Phase: "prepared", Entries: []packJournalEntry{entry}})
		if err := os.WriteFile(filepath.Join(root, packJournalName), body, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := acquirePackWriteLock(root); err == nil || !strings.Contains(err.Error(), "unsafe") {
			t.Fatalf("outside journal path accepted: %v", err)
		}
		got, _ := os.ReadFile(outside)
		if string(got) != "sentinel" {
			t.Fatalf("outside sentinel changed: %q", got)
		}
	})
}

func TestContentHashCoversManifestMinusHashLine(t *testing.T) {
	manifest := "subsystem: s\ndelegated_invariants:\n  - inv-1\ncontent_hash: aaa\n"
	files := map[string]string{"a.txt": "x", "b.txt": "y", "pack.yaml": manifest}
	h1 := ContentHash(files)
	// changing only the content_hash line must not change the hash (the hash
	// is written into the manifest, so it cannot feed back into itself)
	files["pack.yaml"] = strings.Replace(manifest, "content_hash: aaa", "content_hash: bbb", 1)
	if ContentHash(files) != h1 {
		t.Fatal("the content_hash line fed back into the hash")
	}
	// editing any OTHER manifest line (e.g. deleting a delegated invariant)
	// must change the hash: the manifest is covered
	files["pack.yaml"] = strings.Replace(manifest, "  - inv-1\n", "", 1)
	if ContentHash(files) == h1 {
		t.Fatal("manifest edit did not change the hash; the manifest is not covered")
	}
	// file contents stay covered, and the hash is order-independent by name
	files["pack.yaml"] = manifest
	files["a.txt"] = "z"
	if ContentHash(files) == h1 {
		t.Fatal("file change did not change the hash")
	}
}

// The hash a fresh generation writes into the manifest must verify against
// the very files it wrote (the child-side check reads them back from disk).
func TestGeneratedManifestHashVerifies(t *testing.T) {
	packs, err := GeneratePacks(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	for id, files := range packs {
		v, err := ir.LoadYAML([]byte(files["pack.yaml"]))
		if err != nil || v.AsObject() == nil {
			t.Fatalf("pack %s manifest does not parse", id)
		}
		if got := v.AsObject().GetString("content_hash"); got != ContentHash(files) {
			t.Fatalf("pack %s manifest hash %s does not verify against its own files (%s)", id, got, ContentHash(files))
		}
	}
}

func TestSliceModelithQuotesTitleAndPreservesVersion(t *testing.T) {
	src := `kind: modelith
version: 1
title: "Checkout: Split"
enums: {}
entities:
  Order:
    attributes:
      - name: status
        type: OrderStatus
invariants: []
scenarios: []
`
	dm, err := ir.LoadYAML([]byte(src))
	if err != nil || dm.AsObject() == nil {
		t.Fatal(err)
	}
	out := sliceModelith(dm, Subsystem{ID: "orders", Owns: []string{"Order"}})
	v, err := ir.LoadYAML([]byte(out))
	if err != nil || v.AsObject() == nil {
		t.Fatalf("slice with a colon title does not round-trip: %v\n%s", err, out)
	}
	o := v.AsObject()
	if got := o.GetString("title"); got != "Checkout: Split / orders" {
		t.Errorf("title mangled: %q", got)
	}
	ver := o.Get2("version")
	if ver == nil || ver.Kind != ir.KindNumber || string(ver.AsNumber()) != "1" {
		t.Errorf("numeric version not preserved: %s", ir.Repr(ver))
	}
}

func TestYamlQuoteQuotesNumberLikeStrings(t *testing.T) {
	for _, s := range []string{"1e5", "1.5", "42", "-3", "0x1F"} {
		if got := yamlQuote(s); got != `"`+s+`"` {
			t.Errorf("yamlQuote(%q) = %s; a number-like string type-flips on reparse", s, got)
		}
	}
	if got := yamlQuote("plainWord"); got != "plainWord" {
		t.Errorf("plain string quoted needlessly: %s", got)
	}
}

func TestDomainSliceRoundTripsAndFreezesShape(t *testing.T) {
	packs, err := GeneratePacks(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	slice := packs["orders"]["domain.modelith.yaml"]
	v, err := ir.LoadYAML([]byte(slice))
	if err != nil || v.AsObject() == nil {
		t.Fatalf("slice does not round-trip through the yaml loader: %v", err)
	}
	o := v.AsObject()
	if !o.GetObject("entities").Has("Order") {
		t.Fatal("owned entity missing from the slice")
	}
	if o.GetObject("entities").Has("Payment") {
		t.Fatal("foreign entity leaked into the slice")
	}
	if !o.GetObject("enums").Has("OrderStatus") {
		t.Fatal("referenced enum missing from the slice")
	}
	if o.GetObject("enums").Has("PaymentStatus") {
		t.Fatal("unreferenced enum leaked into the slice")
	}
	if !strings.Contains(slice, "no-ship-without-capture") {
		t.Fatal("delegated invariant missing from the slice")
	}
}

func TestEventsSliceDirections(t *testing.T) {
	packs, err := GeneratePacks(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	ev := packs["payments"]["events.md"]
	if !strings.Contains(ev, "| request | consumes | orders |") {
		t.Fatalf("payments should consume request:\n%s", ev)
	}
	if !strings.Contains(ev, "| markPaid | produces | orders |") {
		t.Fatalf("payments should produce markPaid:\n%s", ev)
	}
}

// The event-contract extraction is strict: a producer or consumer cell that
// does not resolve to exactly one known participant fails generation with a
// finding naming the row and the offending cell text. Silent drops shipped
// packs asserting boundary completeness over an almost-empty table once; the
// cases below are the observed lossy shapes.
func TestStrictEventCellValidation(t *testing.T) {
	cases := []struct {
		name     string
		old, new string
		wantSubs []string
	}{
		{
			name: "comma multi-consumer",
			old:  "| request | orders | payments |",
			new:  "| request | orders | orders (drafts), payments (audit) |",
			wantSubs: []string{
				"event 'request'",
				"consumer cell 'orders (drafts), payments (audit)'",
				"names more than one component",
				"one row per producer-consumer pair",
			},
		},
		{
			name:     "slash multi-consumer",
			old:      "| request | orders | payments |",
			new:      "| request | orders | orders/payments |",
			wantSubs: []string{"names more than one component"},
		},
		{
			name:     "arrow in cell",
			old:      "| request | orders | payments |",
			new:      "| request | orders | orders -> payments |",
			wantSubs: []string{"names more than one component"},
		},
		{
			name:     "fan-out phrase",
			old:      "| request | orders | payments |",
			new:      "| request | orders | ALL components (fan-out) |",
			wantSubs: []string{"names more than one component"},
		},
		{
			name:     "event name embedded in producer cell",
			old:      "| markPaid | payments | orders |",
			new:      "| markPaid | payments `markPaid` | orders |",
			wantSubs: []string{"event 'markPaid'", "names more than one component"},
		},
		{
			name: "unknown single name",
			old:  "| request | orders | payments |",
			new:  "| request | warehouse | payments |",
			wantSubs: []string{
				"producer cell 'warehouse'",
				"is not a known component",
				"orders, payments", // the known-participant list teaches the fix
			},
		},
		{
			name:     "empty producer cell",
			old:      "| request | orders | payments |",
			new:      "| request |  | payments |",
			wantSubs: []string{"event 'request'", "empty producer cell"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			design := copyParentDesign(t)
			editDesignFile(t, design, "ARCHITECTURE.md", c.old, c.new)
			_, err := GeneratePacks(design)
			if err == nil {
				t.Fatal("lossy event-contract cell accepted; extraction must fail loudly")
			}
			for _, sub := range c.wantSubs {
				if !strings.Contains(err.Error(), sub) {
					t.Errorf("error %q\nmissing %q", err.Error(), sub)
				}
			}
		})
	}
}

// Parenthetical annotations are the sanctioned annotation channel: the cell
// resolves to the component and the pack row keeps the pairwise direction.
func TestParentheticalAnnotationsAccepted(t *testing.T) {
	design := copyParentDesign(t)
	editDesignFile(t, design, "ARCHITECTURE.md",
		"| request | orders | payments |",
		"| request | orders (command) | payments (worker) |")
	packs, err := GeneratePacks(design)
	if err != nil {
		t.Fatal(err)
	}
	if ev := packs["payments"]["events.md"]; !strings.Contains(ev, "| request | consumes | orders |") {
		t.Errorf("annotated producer cell lost the pairwise contract:\n%s", ev)
	}
	if ev := packs["orders"]["events.md"]; !strings.Contains(ev, "| request | produces | payments |") {
		t.Errorf("annotated consumer cell lost the pairwise contract:\n%s", ev)
	}
}

// A row between two non-pack participants (declared only as Architecture
// Contract boundary elements) validates like any other row and emits no pack
// rows.
func TestNonPackToNonPackRowValidatesAndEmitsNothing(t *testing.T) {
	design := copyParentDesign(t)
	editDesignFile(t, design, "ARCHITECTURE.md",
		"  - id: payments.svc",
		"  - id: gw.svc\n    kind: container\n    element: gw\n    code: [ \"gw/**\" ]\n  - id: payments.svc")
	editDesignFile(t, design, "ARCHITECTURE.md",
		"| markDeclined | payments | orders | Payment.orderId | at-least-once | none | Payment.id |",
		"| markDeclined | payments | orders | Payment.orderId | at-least-once | none | Payment.id |\n| ping | gw | gw | n/a | at-least-once | none | n/a |")
	packs, err := GeneratePacks(design)
	if err != nil {
		t.Fatalf("non-pack-to-non-pack row rejected: %v", err)
	}
	for id, files := range packs {
		if strings.Contains(files["events.md"], "ping") {
			t.Errorf("pack %s picked up a row between two non-pack participants:\n%s", id, files["events.md"])
		}
	}
}

// zeroEventPayments rewrites the table so no row names payments: extraction
// yields zero boundary events for that subsystem. The rows route through a
// declared non-pack boundary element (gw) because a row whose producer and
// consumer belong to one subsystem is itself a generation error now.
func zeroEventPayments(t *testing.T, design string) {
	t.Helper()
	editDesignFile(t, design, "ARCHITECTURE.md",
		"  - id: payments.svc",
		"  - id: gw.svc\n    kind: container\n    element: gw\n    code: [ \"gw/**\" ]\n  - id: payments.svc")
	editDesignFile(t, design, "ARCHITECTURE.md",
		"| request | orders | payments |", "| request | orders | gw |")
	editDesignFile(t, design, "ARCHITECTURE.md",
		"| markPaid | payments | orders |", "| markPaid | gw | orders |")
	editDesignFile(t, design, "ARCHITECTURE.md",
		"| markDeclined | payments | orders |", "| markDeclined | gw | orders |")
}

// waivePayments declares the payments boundary_events waiver.
func waivePayments(t *testing.T, design, reason string) {
	t.Helper()
	editDesignFile(t, design, "decomposition.yaml",
		"    delegated_invariants: []\n",
		"    delegated_invariants: []\n    boundary_events:\n      none: \""+reason+"\"\n")
}

func TestZeroBoundaryEventsIsGenerationError(t *testing.T) {
	design := copyParentDesign(t)
	zeroEventPayments(t, design)
	_, err := GeneratePacks(design)
	if err == nil {
		t.Fatal("a zero-boundary-event subsystem generated silently")
	}
	for _, sub := range []string{"'payments'", "extracts zero boundary events", "boundary_events"} {
		if !strings.Contains(err.Error(), sub) {
			t.Errorf("error %q\nmissing %q", err.Error(), sub)
		}
	}
}

func TestZeroBoundaryEventsWaiverGeneratesWithReason(t *testing.T) {
	design := copyParentDesign(t)
	zeroEventPayments(t, design)
	reason := "payments is store-driven in this variant; no events cross its boundary"
	waivePayments(t, design, reason)
	packs, err := GeneratePacks(design)
	if err != nil {
		t.Fatalf("waived zero-event subsystem still fails: %v", err)
	}
	ev := packs["payments"]["events.md"]
	if !strings.Contains(ev, reason) {
		t.Errorf("waiver reason missing from events.md:\n%s", ev)
	}
	if strings.Contains(ev, "there are no other cross-boundary events") {
		t.Errorf("waived events.md still emits the completeness claim:\n%s", ev)
	}
	if !strings.Contains(ev, "Boundary events: 0 (waived)") {
		t.Errorf("waived events.md missing the visible zero count:\n%s", ev)
	}
	if got := CountBoundaryEvents(ev); got != 0 {
		t.Errorf("CountBoundaryEvents(waived) = %d, want 0", got)
	}
}

func TestStaleWaiverIsGenerationError(t *testing.T) {
	design := copyParentDesign(t)
	waivePayments(t, design, "stale: rows still name payments")
	_, err := GeneratePacks(design)
	if err == nil || !strings.Contains(err.Error(), "remove the stale waiver") {
		t.Fatalf("stale waiver accepted: %v", err)
	}
}

func TestMalformedWaiverIsLoadError(t *testing.T) {
	design := copyParentDesign(t)
	editDesignFile(t, design, "decomposition.yaml",
		"    delegated_invariants: []\n",
		"    delegated_invariants: []\n    boundary_events: nonsense\n")
	_, err := LoadDecomposition(design)
	if err == nil || !strings.Contains(err.Error(), "boundary_events must be a mapping") {
		t.Fatalf("malformed boundary_events accepted: %v", err)
	}
}

func TestCountBoundaryEvents(t *testing.T) {
	packs, err := GeneratePacks(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	if got := CountBoundaryEvents(packs["payments"]["events.md"]); got != 3 {
		t.Errorf("payments boundary events = %d, want 3", got)
	}
	if got := CountBoundaryEvents("not a generated events file"); got != -1 {
		t.Errorf("absent count line = %d, want -1", got)
	}
}

// A parent model with no title yields just the subsystem id; the old
// unconditional concatenation emitted the nonsense title " / core".
func TestSliceModelithTitleWithoutParentTitle(t *testing.T) {
	src := `kind: modelith
version: 1
enums: {}
entities:
  Order:
    attributes:
      - name: status
        type: OrderStatus
invariants: []
scenarios: []
`
	dm, err := ir.LoadYAML([]byte(src))
	if err != nil || dm.AsObject() == nil {
		t.Fatal(err)
	}
	out := sliceModelith(dm, Subsystem{ID: "orders", Owns: []string{"Order"}})
	v, err := ir.LoadYAML([]byte(out))
	if err != nil || v.AsObject() == nil {
		t.Fatalf("slice does not round-trip: %v\n%s", err, out)
	}
	if got := v.AsObject().GetString("title"); got != "orders" {
		t.Errorf("title = %q, want %q", got, "orders")
	}
}

// P-F10: pack files are covered by the content hash, so they must NEVER carry
// a version stamp (it would churn the hash every release and re-red every
// pinned child).
func TestGeneratedPackFilesAreUnstamped(t *testing.T) {
	packs, err := GeneratePacks(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	for id, files := range packs {
		for name, body := range files {
			if strings.Contains(body, "machinery-version:") {
				t.Errorf("pack %s file %s carries a version stamp; the content hash covers it", id, name)
			}
		}
	}
}

func TestWritePacksStaysOnOpenedRootDuringParentSwap(t *testing.T) {
	base := t.TempDir()
	design := filepath.Join(base, "design")
	root := filepath.Join(design, "packs")
	held := filepath.Join(base, "held-packs")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(filepath.Join(root, "orders.pack"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "orders.pack", "pack.yaml"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	swapped := false
	rename := func(opened *os.Root, old, new string) error {
		if !swapped {
			swapped = true
			if err := os.Rename(root, held); err != nil {
				return err
			}
			if err := os.Symlink(outside, root); err != nil {
				return err
			}
		}
		return nil
	}
	ids, err := writePacksWithRename(design, map[string]map[string]string{
		"orders": {"pack.yaml": "new"},
	}, rename)
	if err != nil || !reflect.DeepEqual(ids, []string{"orders"}) {
		t.Fatalf("rooted pack write failed: ids=%v err=%v", ids, err)
	}
	got, err := os.ReadFile(filepath.Join(held, "orders.pack", "pack.yaml"))
	if err != nil || string(got) != "new" {
		t.Fatalf("opened root received %q, %v", got, err)
	}
	got, err = os.ReadFile(sentinel)
	if err != nil || string(got) != "outside" {
		t.Fatalf("outside sentinel changed: %q, %v", got, err)
	}
	if _, err := os.Lstat(filepath.Join(outside, "orders.pack")); !os.IsNotExist(err) {
		t.Fatalf("ambient replacement received transaction output: %v", err)
	}
}

func TestApplyPackReleaseMakesSuccessFailAndJoinsCauses(t *testing.T) {
	primary := errors.New("primary")
	releaseErr := errors.New("injected pack lock close failure")
	ids, err := applyPackRelease([]string{"orders"}, primary, func() error { return releaseErr })
	if ids != nil || !errors.Is(err, primary) || !errors.Is(err, releaseErr) {
		t.Fatalf("release failure was hidden or causes were lost: ids=%v err=%v", ids, err)
	}
}

// P-F10: the refinement modules the child commits to design/formal DO carry
// the stamp (G5 strips it before the freshness diff); the contract module
// stays a byte-copy of the pack's hash-covered file.
func TestGenerateRefinementStampsGeneratedModules(t *testing.T) {
	design := filepath.Join("..", "..", "examples", "checkout-split", "orders", "design")
	files, err := GenerateRefinement(design)
	if err != nil {
		t.Fatal(err)
	}
	stamped := 0
	for name, body := range files {
		switch {
		case strings.HasSuffix(name, "PackRefinement.tla"), strings.HasSuffix(name, "PackRefinement.cfg"):
			if got := strings.Count(body, "machinery-version:"); got != 1 {
				t.Errorf("%s carries %d stamp lines, want exactly 1", name, got)
			}
			if strings.HasSuffix(name, ".tla") && !strings.HasPrefix(body, "---- MODULE ") {
				t.Errorf("%s no longer opens with the MODULE line", name)
			}
			stamped++
		default:
			// the contract module copied from the pack
			if strings.Contains(body, "machinery-version:") {
				t.Errorf("%s must stay byte-identical to the pack's copy (no stamp)", name)
			}
			packCopy, rerr := os.ReadFile(filepath.Join(design, "pack", name))
			if rerr != nil {
				t.Fatal(rerr)
			}
			if body != string(packCopy) {
				t.Errorf("%s differs from the pack's copy", name)
			}
		}
	}
	if stamped != 2 {
		t.Errorf("expected 2 stamped refinement artifacts, got %d", stamped)
	}
}
