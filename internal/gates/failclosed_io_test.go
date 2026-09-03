package gates

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/filelock"
)

func writeFailClosedFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func symlinkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

func TestSelectPreflightsExplicitGateInventory(t *testing.T) {
	design := t.TempDir()
	outside := filepath.Join(t.TempDir(), "workspace.dsl")
	writeFailClosedFile(t, outside, "workspace outside")
	symlinkOrSkip(t, outside, filepath.Join(design, "workspace.dsl"))
	if _, err := Select(design, "g2", ""); err == nil || !strings.Contains(err.Error(), "workspace.dsl") || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("explicit selection followed an outside artifact: %v", err)
	}
}

func TestRootedMachineReadRejectsParentSymlinkSwap(t *testing.T) {
	design := t.TempDir()
	safeDir := filepath.Join(design, "nested")
	machineName := "Order.machine.json"
	writeFailClosedFile(t, filepath.Join(safeDir, machineName), `{"id":"inside-sentinel","initial":"A","states":{"A":{"type":"final"}}}`)
	outside := t.TempDir()
	writeFailClosedFile(t, filepath.Join(outside, machineName), `{"id":"outside-sentinel","initial":"A","states":{"A":{"type":"final"}}}`)

	root, _, err := openRealRoot(design)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	rel := filepath.Join("nested", machineName)
	if ok, err := inspectRootPath(root, rel, false); err != nil || !ok {
		t.Fatalf("preflight failed before adversarial swap: ok=%v err=%v", ok, err)
	}

	moved := filepath.Join(design, "nested-safe")
	if err := os.Rename(safeDir, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, safeDir); err != nil {
		if restoreErr := os.Rename(moved, safeDir); restoreErr != nil {
			t.Fatalf("symlinks unavailable (%v) and safe directory restore failed: %v", err, restoreErr)
		}
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(safeDir)
		_ = os.Rename(moved, safeDir)
	})

	body, err := readRootRegularFile(root, rel)
	if err == nil {
		t.Fatalf("rooted read accepted swapped parent and returned %q", body)
	}
	if strings.Contains(string(body), "outside-sentinel") {
		t.Fatalf("rooted read escaped to outside sentinel: %q", body)
	}
	outsideBody, readErr := os.ReadFile(filepath.Join(outside, machineName))
	if readErr != nil || !strings.Contains(string(outsideBody), "outside-sentinel") {
		t.Fatalf("outside sentinel changed: %q, %v", outsideBody, readErr)
	}
}

func TestRootedReadRejectsOversizedSparseArtifactBeforeAllocation(t *testing.T) {
	design := t.TempDir()
	path := filepath.Join(design, "oversized.md")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(designArtifactMaxBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegularFile(path); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("rooted reader accepted oversized sparse artifact: %v", err)
	}
}

func TestRootedReadRejectsSameSizeConcurrentMutation(t *testing.T) {
	design := t.TempDir()
	path := filepath.Join(design, "mutable.md")
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	prior := rootReadAfterInitial
	t.Cleanup(func() { rootReadAfterInitial = prior })
	rootReadAfterInitial = func(rel string) {
		if rel != "mutable.md" {
			return
		}
		rootReadAfterInitial = func(string) {}
		if err := os.WriteFile(path, []byte("change"), 0o600); err != nil {
			t.Error(err)
		}
	}
	if _, err := readRegularFile(path); err == nil || !strings.Contains(err.Error(), "changed while reading") {
		t.Fatalf("rooted reader accepted same-size mutation: %v", err)
	}
}

func TestBoundedTreeWalkerRejectsHighEntryInventory(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a", "b", "c"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	err := walkTreeDirBounded(dir, 2, designInventoryMaxDepth, func(string, os.DirEntry, error) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "-entry limit") {
		t.Fatalf("bounded tree walker accepted high-entry inventory: %v", err)
	}
}

func TestSelectRejectsNonportableAndAliasedDesignPaths(t *testing.T) {
	for _, name := range []string{"CON.md", "naïve.md", "trailing."} {
		t.Run(name, func(t *testing.T) {
			design := t.TempDir()
			writeFailClosedFile(t, filepath.Join(design, name), "artifact\n")
			if _, err := Select(design, "g2", ""); err == nil || !strings.Contains(err.Error(), "non-portable design path") {
				t.Fatalf("accepted nonportable path %q: %v", name, err)
			}
		})
	}
	t.Run("case folded collision", func(t *testing.T) {
		design := t.TempDir()
		writeFailClosedFile(t, filepath.Join(design, "BUILD.md"), "first\n")
		writeFailClosedFile(t, filepath.Join(design, "build.md"), "second\n")
		entries, err := os.ReadDir(design)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) < 2 {
			t.Skip("host filesystem aliases case; collision regression runs on case-sensitive CI hosts")
		}
		if _, err := Select(design, "g2", ""); err == nil || !strings.Contains(err.Error(), "portable design-path collision") {
			t.Fatalf("accepted case-folded collision: %v", err)
		}
	})
}

func TestPortableInventoryCollisionIsHostIndependent(t *testing.T) {
	seen := map[string]string{}
	if err := validatePortableInventoryPath(seen, "BUILD/Plan.md"); err != nil {
		t.Fatal(err)
	}
	if err := validatePortableInventoryPath(seen, "build/plan.md"); err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("case-folded alias was accepted: %v", err)
	}
}

func TestSelectFailsClosedOnMarkdownAndActivationSymlinks(t *testing.T) {
	outside := t.TempDir()
	writeFailClosedFile(t, filepath.Join(outside, "outside.md"), "<!-- machinery:embed -->")
	writeFailClosedFile(t, filepath.Join(outside, "policy.yaml"), "policy_version: 1\n")
	for _, tc := range []struct {
		name, rel, target string
	}{
		{"markdown", "outside.md", filepath.Join(outside, "outside.md")},
		{"policy", filepath.Join("formal", "policy.yaml"), filepath.Join(outside, "policy.yaml")},
		{"build", "BUILD.md", filepath.Join(outside, "outside.md")},
		{"acceptance", AcceptanceDirName, outside},
		{"adjudication", AdjudicationDirName, outside},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			if err := os.MkdirAll(filepath.Dir(filepath.Join(design, tc.rel)), 0o755); err != nil {
				t.Fatal(err)
			}
			symlinkOrSkip(t, tc.target, filepath.Join(design, tc.rel))
			if _, err := Select(design, "", ""); err == nil || !strings.Contains(err.Error(), "symlink") {
				t.Fatalf("activation discovery followed %s symlink: %v", tc.rel, err)
			}
		})
	}
}

func TestTraceabilityRejectsSymlinkedModelith(t *testing.T) {
	design := t.TempDir()
	outside := filepath.Join(t.TempDir(), "domain.modelith.yaml")
	writeFailClosedFile(t, outside, "modelith_version: 2\nmodel: x\n")
	symlinkOrSkip(t, outside, filepath.Join(design, "domain.modelith.yaml"))
	g := CheckTraceability(design)
	if got := strings.Join(g.Errs, "\n"); !strings.Contains(got, "modelith source must be a regular file") {
		t.Fatalf("Gx followed modelith symlink: %v", g.Errs)
	}
}

func TestMachineInventoryRequiresPortableIdentity(t *testing.T) {
	t.Run("portable basename", func(t *testing.T) {
		design := t.TempDir()
		writeFailClosedFile(t, filepath.Join(design, "machines", "naïve.machine.json"), `{"id":"naïve","initial":"A","states":{"A":{}}}`)
		g := CheckMachines(design)
		if got := strings.Join(g.Errs, "\n"); !strings.Contains(got, "filename is not portable") {
			t.Fatalf("G3 accepted nonportable basename: %v", g.Errs)
		}
	})
	t.Run("canonical id", func(t *testing.T) {
		design := t.TempDir()
		writeFailClosedFile(t, filepath.Join(design, "machines", "Order.machine.json"), `{"id":"payment","initial":"A","states":{"A":{}}}`)
		g := CheckMachines(design)
		if got := strings.Join(g.Errs, "\n"); !strings.Contains(got, "filename stem 'Order' does not identify canonical machine id 'payment'") {
			t.Fatalf("G3 accepted divergent filename/id: %v", g.Errs)
		}
	})
	t.Run("portable stem", func(t *testing.T) {
		design := t.TempDir()
		writeFailClosedFile(t, filepath.Join(design, "machines", "Order..machine.json"), `{"id":"order","initial":"A","states":{"A":{}}}`)
		g := CheckMachines(design)
		if got := strings.Join(g.Errs, "\n"); !strings.Contains(got, "stem is not portable") {
			t.Fatalf("G3 accepted nonportable stem: %v", g.Errs)
		}
	})
}

func TestTraversalAuditsPropagateSymlinkReadErrors(t *testing.T) {
	design := t.TempDir()
	writeFailClosedFile(t, filepath.Join(design, "machines", "Deal.matrix.md"), "| name | kind | contract |\n|---|---|---|\n| `canWin` | guard | CLAUSES{ready, funded} |\n| `deal.won` | event | READS{amount} |\n")
	outside := filepath.Join(t.TempDir(), "outside.md")
	writeFailClosedFile(t, outside, "(2 rows)\n| event | a | b | c | d | e |\n|---|---|---|---|---|---|\n| `deal.won` | x | x | x | x | amount |\n")
	symlinkOrSkip(t, outside, filepath.Join(design, "hidden.md"))

	for _, tc := range []struct {
		name string
		run  func(*Gate)
	}{
		{"clauses", func(g *Gate) { checkClauseDrift(g, design) }},
		{"counts", func(g *Gate) { checkProseCounts(g, design) }},
		{"payload", func(g *Gate) { checkPayloadReads(g, design) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := NewGate(tc.name)
			tc.run(g)
			if got := strings.Join(g.Errs, "\n"); !strings.Contains(got, "symlink") {
				t.Fatalf("%s audit swallowed traversal/read error: %v", tc.name, g.Errs)
			}
		})
	}
}

func TestAttestationPackTraversalErrorIsBlocking(t *testing.T) {
	design := t.TempDir()
	if err := os.Mkdir(filepath.Join(design, "pack"), 0o755); err != nil {
		t.Fatal(err)
	}
	symlinkOrSkip(t, filepath.Join(t.TempDir(), "missing"), filepath.Join(design, "pack", "outside.pack"))
	g := NewGate("attest")
	_ = attestationRequiredPaths(g, design, "g4.pack-event-discipline")
	if got := strings.Join(g.Errs, "\n"); !strings.Contains(got, "pack attestation subject inventory failed") {
		t.Fatalf("attestation inventory swallowed pack traversal error: %v", g.Errs)
	}
}

func TestWriteRatchetRejectsSymlinkTarget(t *testing.T) {
	design := t.TempDir()
	outside := filepath.Join(t.TempDir(), "ratchet.json")
	writeFailClosedFile(t, outside, "sentinel\n")
	symlinkOrSkip(t, outside, filepath.Join(design, RatchetFile))
	err := WriteRatchet(design, &Ratchet{Date: "2026-09-02", Edges: map[string][]string{}})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("ratchet writer followed symlink: %v", err)
	}
	body, readErr := os.ReadFile(outside)
	if readErr != nil || string(body) != "sentinel\n" {
		t.Fatalf("outside ratchet changed: %q, %v", body, readErr)
	}
}

func TestRefreshEmbedsRollsBackEarlierDirectory(t *testing.T) {
	design := t.TempDir()
	source := "| event | value |\n|---|---|\n| `deal.won` | fresh |\n"
	writeFailClosedFile(t, filepath.Join(design, "source.md"), source)
	doc := "<!-- machinery:embed from=\"../source.md\" table=\"event,value\" claims=\"subset\" -->\n| event | value |\n|---|---|\n| `deal.won` | stale |\n"
	aPath := filepath.Join(design, "a", "copy.md")
	zPath := filepath.Join(design, "z", "copy.md")
	writeFailClosedFile(t, aPath, doc)
	writeFailClosedFile(t, zPath, doc)

	priorCommit := embedTransactionPoint
	failed := false
	embedTransactionPoint = func(point string) error {
		if point == "directory:a" && !failed {
			failed = true
			return errors.New("injected late commit failure")
		}
		return nil
	}
	t.Cleanup(func() { embedTransactionPoint = priorCommit })

	if _, _, err := RefreshEmbeds(design, false); err == nil || !strings.Contains(err.Error(), "injected late commit failure") {
		t.Fatalf("refresh did not surface late failure: %v", err)
	}
	for _, path := range []string{aPath, zPath} {
		body, err := os.ReadFile(path)
		if err != nil || string(body) != doc {
			t.Fatalf("%s was left partially refreshed: %q, %v", path, body, err)
		}
	}
}

func TestRefreshEmbedsRejectsConcurrentDesignRefresh(t *testing.T) {
	design := t.TempDir()
	writeFailClosedFile(t, filepath.Join(design, "doc.md"), "unchanged\n")
	lock, err := filelock.Acquire(embedRefreshLockScope(design))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := lock.Release(); err != nil {
			t.Error(err)
		}
	}()
	if _, _, err := RefreshEmbeds(design, false); err == nil || !strings.Contains(err.Error(), "another operation holds the lock") {
		t.Fatalf("concurrent refresh was not rejected: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(design, "doc.md"))
	if err != nil || string(body) != "unchanged\n" {
		t.Fatalf("contended refresh mutated document: %q, %v", body, err)
	}
}
