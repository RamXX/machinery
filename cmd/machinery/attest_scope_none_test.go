package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestAttestationScopeNone treats the digest descriptor "none" as grammar,
// never as authority to omit a real entry. Each leaf owns its fixture: a
// failed inventory assertion cannot prevent a different mutation from running.
func TestAttestationScopeNone(t *testing.T) {
	binary := goldenBin(t)
	t.Logf("ordinary CLI binary=%s hash=%s", binary, cliReviewHash(cliReviewRead(t, binary)))
	for _, topology := range []string{"disjoint", "inside-design", "equal", "ancestor"} {
		for _, kind := range []string{"file", "empty-directory"} {
			for _, challenge := range []string{
				"inventory-absent-evidence", "unchanged", "independent-full-manifest",
				"content-or-mode", "addition", "removal", "rename",
				"omission-old-digest", "omission-rehashed",
				"evidence-only", "sibling-evidence", "nested-evidence",
				"git-directory", "gitfile", "nested-git",
			} {
				t.Run(topology+"/"+kind+"/"+challenge, func(t *testing.T) {
					f := scopeNoneFixture(t, topology, kind, challenge == "gitfile", challenge != "addition")
					row := cliReviewGenerate(t, binary, f, "gt.conformance-test-shape", "current")
					scopeNoneLogRow(t, "GENERATED_ROW", row)
					want := scopeNoneManifest(t, f)
					if challenge == "inventory-absent-evidence" {
						if _, err := os.Lstat(filepath.Join(f.design, "attestations.yaml")); !os.IsNotExist(err) {
							t.Fatalf("SETUP: evidence must initially be absent: %v", err)
						}
						scopeNoneEqualManifest(t, row["implementation"], want)
						return
					}
					cliReviewSave(t, f, 2, row)
					scopeNoneCurrent(t, cliReviewCheck(t, binary, f))
					t.Log("CONTROL_REACHED: generated receipt accepted unchanged by actual CLI")
					if challenge == "unchanged" {
						return
					}
					if challenge == "independent-full-manifest" {
						row["implementation"] = want
						cliReviewSave(t, f, 2, row)
						scopeNoneLogRow(t, "SUBMITTED_FULL_ROW", row)
						t.Log("CHALLENGE_REACHED: independently enumerated complete manifest")
						scopeNoneCurrent(t, cliReviewCheck(t, binary, f))
						return
					}
					path := filepath.Join(f.impl, "none")
					category, diagnostic := "GV_SCOPE_INVENTORY", "scope path none added or removed"
					switch challenge {
					case "content-or-mode":
						if kind == "file" {
							cliReviewWrite(t, path, "return unchecked behavior\n")
							if string(cliReviewRead(t, path)) != "return unchecked behavior\n" {
								t.Fatal("SETUP: file mutation did not persist")
							}
						} else {
							scopeNoneChmod(t, path, 0o700)
						}
						scopeNoneEntry(t, path, kind, map[string]fs.FileMode{"file": 0o644, "empty-directory": 0o700}[kind])
						category, diagnostic = "GV_STALE_CONTENT", "scope path none changed"
					case "addition":
						scopeNoneCreate(t, path, kind)
					case "removal":
						if err := os.Remove(path); err != nil {
							t.Fatalf("SETUP: remove literal none: %v", err)
						}
						if _, err := os.Lstat(path); !os.IsNotExist(err) {
							t.Fatalf("SETUP: none still exists: %v", err)
						}
					case "rename":
						if err := os.Rename(path, path+"-renamed"); err != nil {
							t.Fatalf("SETUP: rename literal none: %v", err)
						}
						scopeNoneEntry(t, path+"-renamed", kind, map[string]fs.FileMode{"file": 0o644, "empty-directory": 0o755}[kind])
					case "omission-old-digest", "omission-rehashed":
						// Construct the full original digest from actual input bytes,
						// independently of the generator's possibly incomplete receipt.
						// Its unedited acceptance has its own independent leaf above.
						entries := want["entries"].([]any)
						var narrowed []any
						removed := 0
						for _, value := range entries {
							if value.(map[string]any)["path"] == "none" {
								removed++
							} else {
								narrowed = append(narrowed, value)
							}
						}
						if removed != 1 {
							t.Fatalf("SETUP: independent full inventory omitted %d literal none entries", removed)
						}
						want["entries"] = narrowed
						if challenge == "omission-rehashed" {
							want["hash"] = scopeNoneDigest(t, f, want)
						} else {
							category, diagnostic = "GV_SCOPE_HASH", "submitted inventory does not match its recorded hash"
						}
						row["implementation"] = want
						cliReviewSave(t, f, 2, row)
						scopeNoneLogRow(t, "SUBMITTED_OMISSION_ROW", row)
					case "sibling-evidence", "nested-evidence":
						rel := scopeNoneSibling(f)
						if challenge == "nested-evidence" {
							rel = "nested/inner/attestations.yaml"
						}
						cliReviewWrite(t, filepath.Join(f.impl, rel), "changed ordinary runtime input\n")
						category, diagnostic = "GV_STALE_CONTENT", "scope path "+rel+" changed"
					case "nested-git":
						cliReviewWrite(t, filepath.Join(f.impl, "nested", ".git", "config"), "nested metadata\n")
						category = "GV_SCOPE_UNSUPPORTED_METADATA"
						diagnostic = "nested metadata " + filepath.Join(f.impl, "nested", ".git")
					case "evidence-only", "git-directory", "gitfile":
						before := row["implementation"].(map[string]any)["hash"]
						row["note"] = "diagnostic attribution corrected; no new implementation review"
						cliReviewSave(t, f, 2, row)
						if challenge == "evidence-only" {
							scopeNoneChmod(t, filepath.Join(f.design, "attestations.yaml"), 0o600)
							repository := f.impl
							evidence, err := filepath.Rel(repository, filepath.Join(f.design, "attestations.yaml"))
							if err != nil {
								t.Fatal(err)
							}
							if strings.HasPrefix(filepath.ToSlash(evidence), "../") {
								// The evidence is outside the implementation repository.
								// A separate real design repository records only evidence.
								repository, evidence = f.design, "attestations.yaml"
								cliReviewGit(t, repository, "init", "-q")
							}
							cliReviewGit(t, repository, "add", "--", evidence)
							cliReviewGit(t, repository, "commit", "-q", "-m", "evidence-only diagnostic correction")
						} else {
							cliReviewGit(t, f.impl, "-c", "core.quotePath=false", "commit", "--allow-empty", "-q", "-m", "metadata-only local commit")
						}
						t.Logf("CHALLENGE_REACHED: %s; evidence presence/content/mode or real Git metadata changed", challenge)
						scopeNoneCurrent(t, cliReviewCheck(t, binary, f))
						again := cliReviewGenerate(t, binary, f, "gt.conformance-test-shape", "current")
						if again["implementation"].(map[string]any)["hash"] != before {
							t.Fatal("evidence/metadata-only change altered scope digest")
						}
						// Independent expected inventory checks the exclusion descriptor
						// with the evidence present, as well as the absent-evidence leaf.
						scopeNoneEqualManifest(t, again["implementation"], scopeNoneManifest(t, f))
						return
					}
					t.Logf("CHALLENGE_REACHED: %s; expected %s naming %q", challenge, category, diagnostic)
					scopeNoneRejected(t, cliReviewCheck(t, binary, f), category, diagnostic)
				})
			}
		}
	}
}

func scopeNoneFixture(t *testing.T, topology, kind string, gitfile, includeNone bool) *cliReviewFixture {
	t.Helper()
	outer := t.TempDir()
	root := filepath.Join(outer, "checkout")
	f := &cliReviewFixture{root: root, design: filepath.Join(root, "design"), impl: filepath.Join(root, "impl")}
	switch topology {
	case "inside-design":
		f.impl = filepath.Join(f.design, "impl")
	case "equal":
		f.impl = f.design
	case "ancestor":
		f.impl = root
	}
	for _, dir := range []string{root, f.design, f.impl} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("SETUP: topology directory: %v", err)
		}
		scopeNoneChmod(t, dir, 0o755)
	}
	cliReviewWrite(t, filepath.Join(f.design, "BUILD.md"), "# Build\n")
	cliReviewWrite(t, filepath.Join(f.impl, "handler.go"), cliReviewHandler)
	cliReviewWrite(t, filepath.Join(f.impl, scopeNoneSibling(f)), "ordinary sibling input\n")
	cliReviewWrite(t, filepath.Join(f.impl, "nested", "inner", "attestations.yaml"), "ordinary nested input\n")
	if includeNone {
		scopeNoneCreate(t, filepath.Join(f.impl, "none"), kind)
	}
	args := []string{"init", "-q"}
	if gitfile {
		args = append(args, "--separate-git-dir", filepath.Join(outer, "git-admin"))
	}
	cliReviewGit(t, f.impl, args...)
	info, err := os.Lstat(filepath.Join(f.impl, ".git"))
	if err != nil || gitfile && !info.Mode().IsRegular() || !gitfile && !info.IsDir() {
		t.Fatalf("SETUP: actual top-level Git metadata kind: %v %v", info, err)
	}
	cliReviewGit(t, f.impl, "add", ".")
	cliReviewGit(t, f.impl, "commit", "-q", "-m", "diagnostic fixture baseline")
	return f
}

func scopeNoneSibling(f *cliReviewFixture) string {
	if f.impl == f.design {
		return "adjacent/attestations.yaml"
	}
	return "attestations.yaml"
}

func scopeNoneCreate(t *testing.T, path, kind string) {
	t.Helper()
	mode := fs.FileMode(0o644)
	if kind == "file" {
		cliReviewWrite(t, path, "assert actual behavior\n")
	} else {
		mode = 0o755
		if err := os.Mkdir(path, mode); err != nil {
			t.Fatalf("SETUP: empty literal none directory: %v", err)
		}
	}
	scopeNoneChmod(t, path, mode)
	scopeNoneEntry(t, path, kind, mode)
}

func scopeNoneChmod(t *testing.T, path string, mode fs.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("SETUP: chmod %s: %v", path, err)
	}
}

func scopeNoneEntry(t *testing.T, path, kind string, mode fs.FileMode) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("SETUP: literal entry unavailable: %v", err)
	}
	if info.Mode().Perm() != mode || kind == "file" && !info.Mode().IsRegular() || kind == "empty-directory" && !info.IsDir() {
		t.Fatalf("SETUP: actual literal entry %s type/mode=%s, want %s/%04o", path, info.Mode(), kind, mode)
	}
	if info.IsDir() {
		children, err := os.ReadDir(path)
		if err != nil || len(children) != 0 {
			t.Fatalf("SETUP: literal directory must remain empty: %v %v", children, err)
		}
	} else {
		t.Logf("ACTUAL_FILE_HASH: %s %s", path, cliReviewHash(cliReviewRead(t, path)))
	}
	t.Logf("ACTUAL_ENTRY: %s type=%s mode=%04o size=%d", path, kind, mode, info.Size())
}

// This oracle walks original inputs with standard filesystem calls, without
// calling Machinery's inventory, projection, manifest or digest functions.
func scopeNoneManifest(t *testing.T, f *cliReviewFixture) map[string]any {
	t.Helper()
	var entries []any
	err := filepath.WalkDir(f.impl, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == filepath.Join(f.impl, ".git") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == filepath.Join(f.design, "attestations.yaml") {
			return nil
		}
		rel, err := filepath.Rel(f.impl, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		entry := map[string]any{"path": filepath.ToSlash(rel), "type": "directory", "mode": fmt.Sprintf("%04o", info.Mode().Perm())}
		if !info.IsDir() {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("SETUP: oracle encountered nonregular input %s", rel)
			}
			entry["type"], entry["size"], entry["hash"] = "file", info.Size(), cliReviewHash(cliReviewRead(t, path))
		}
		entries = append(entries, entry)
		return nil
	})
	if err != nil {
		t.Fatalf("SETUP: independent inventory: %v", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].(map[string]any)["path"].(string) < entries[j].(map[string]any)["path"].(string)
	})
	locator, err := filepath.Rel(f.design, f.impl)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"root": filepath.ToSlash(locator), "policy": "full-root-v1", "entries": entries}
	want["hash"] = scopeNoneDigest(t, f, want)
	return want
}

func scopeNoneDigest(t *testing.T, f *cliReviewFixture, scope map[string]any) string {
	t.Helper()
	evidence, err := filepath.Rel(f.impl, filepath.Join(f.design, "attestations.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	descriptor := filepath.ToSlash(evidence)
	if descriptor == ".." || strings.HasPrefix(descriptor, "../") {
		descriptor = "none"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "machinery-attestation-scope-v1\nroot\t%s\npolicy\tfull-root-v1\nexclude\tvcs-root:.git\nexclude\tevidence:%s\n", scope["root"], descriptor)
	for _, value := range scope["entries"].([]any) {
		e := value.(map[string]any)
		if e["type"] == "directory" {
			fmt.Fprintf(&b, "directory\t%s\t%s\n", e["path"], e["mode"])
		} else {
			fmt.Fprintf(&b, "file\t%s\t%s\t%v\t%s\n", e["path"], e["mode"], e["size"], e["hash"])
		}
	}
	return cliReviewHash([]byte(b.String()))
}

func scopeNoneEqualManifest(t *testing.T, actual any, expected map[string]any) {
	t.Helper()
	// JSON normalizes YAML integer widths while preserving every key/value.
	a, errA := json.Marshal(actual)
	w, errW := json.Marshal(expected)
	if errA != nil || errW != nil {
		t.Fatalf("SETUP: compare manifest JSON: %v %v", errA, errW)
	}
	if !reflect.DeepEqual(a, w) {
		t.Fatalf("INVENTORY_REACHED: generated full-root manifest differs from independent filesystem metadata/hash oracle\nactual=%s\nexpected=%s", a, w)
	}
	t.Logf("INVENTORY_REACHED: complete generated manifest matches independent oracle %s", w)
}

func scopeNoneCurrent(t *testing.T, r cliReviewResult) {
	t.Helper()
	t.Logf("ACTUAL_RESULT: exit=%d stdout=%q stderr=%q", r.code, r.out, r.stderr)
	if r.code != 0 || !regexp.MustCompile(`\b1 current implementation reviews\b`).MatchString(r.out) || !strings.Contains(r.out, cliReviewLimits) {
		t.Fatalf("CURRENT_REACHED: want exit 0, exact current count 1 and scope limits; actual exit=%d stdout=%q stderr=%q", r.code, r.out, r.stderr)
	}
}

func scopeNoneLogRow(t *testing.T, stage string, row map[string]any) {
	t.Helper()
	b, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("SETUP: record row: %v", err)
	}
	t.Logf("%s: %s", stage, b)
}

func scopeNoneRejected(t *testing.T, r cliReviewResult, category, diagnostic string) {
	t.Helper()
	t.Logf("ACTUAL_RESULT: exit=%d stdout=%q stderr=%q", r.code, r.out, r.stderr)
	text := r.out + r.stderr
	if r.code != 1 || !strings.Contains(text, category+": "+diagnostic) || strings.Contains(text, "1 current implementation reviews") {
		t.Fatalf("BEHAVIOR_REACHED: want exit 1 and %s: %s with no current success; actual exit=%d stdout=%q stderr=%q", category, diagnostic, r.code, r.out, r.stderr)
	}
}
