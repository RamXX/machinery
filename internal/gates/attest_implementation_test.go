package gates

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"
)

// These fixtures implement the reviewed public receipt grammar independently
// of production hashing/parsing. A/B/C/D names identify the frozen RED stage.
const reviewLimits = "Hashes bind the observed files and scope; they do not prove tests ran, reviewer identity, or judgment correctness. Top-level Git administration and the exact Machinery attestation record are excluded; applications that use them as runtime inputs are outside this review boundary."
const reviewHandler = "package example\nfunc Handle() string { return \"accepted\" }\n"
const reviewTest = "package example\nimport \"testing\"\n// ORACLESET{machines/Thing.oracle.md}\nfunc TestHandle(t *testing.T) { if Handle() != \"accepted\" { t.Fatal(\"wrong action\") } }\n"

type reviewEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Mode string `json:"mode"`
	Size *int64 `json:"size,omitempty"`
	Hash string `json:"hash,omitempty"`
}
type reviewScope struct {
	Root    string        `json:"root"`
	Policy  string        `json:"policy"`
	Entries []reviewEntry `json:"entries"`
	Hash    string        `json:"hash"`
}
type reviewCover struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}
type reviewRecord struct {
	Claim          string        `json:"claim"`
	Kind           string        `json:"kind,omitempty"`
	Attestor       string        `json:"attestor"`
	Date           string        `json:"date"`
	Covers         []reviewCover `json:"covers"`
	Implementation *reviewScope  `json:"implementation,omitempty"`
	Note           string        `json:"note,omitempty"`
}
type reviewDocument struct {
	Version int            `json:"attestation_version"`
	Rows    []reviewRecord `json:"attestations"`
}
type reviewFixture struct {
	design, impl string
	doc          reviewDocument
}

func reviewHash(body []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(body)) }
func reviewRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func reviewWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
func reviewScopeHash(scope reviewScope, exclusion string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "machinery-attestation-scope-v1\nroot\t%s\npolicy\t%s\nexclude\tvcs-root:.git\nexclude\tevidence:%s\n", scope.Root, scope.Policy, exclusion)
	for _, e := range scope.Entries {
		if e.Type == "directory" {
			fmt.Fprintf(&b, "directory\t%s\t%s\n", e.Path, e.Mode)
		} else {
			fmt.Fprintf(&b, "file\t%s\t%s\t%d\t%s\n", e.Path, e.Mode, *e.Size, e.Hash)
		}
	}
	return reviewHash([]byte(b.String()))
}
func reviewInventory(t *testing.T, design, impl string) *reviewScope {
	t.Helper()
	locator, err := filepath.Rel(design, impl)
	if err != nil {
		t.Fatal(err)
	}
	s := &reviewScope{Root: filepath.ToSlash(locator), Policy: "full-root-v1"}
	evidence := filepath.Join(design, AttestationsFileName)
	exclusion := "none"
	if rel, err := filepath.Rel(impl, evidence); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		exclusion = filepath.ToSlash(rel)
	}
	err = filepath.WalkDir(impl, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(impl, path)
		if err != nil {
			return err
		}
		if rel == ".git" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == evidence {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		e := reviewEntry{Path: filepath.ToSlash(rel), Mode: fmt.Sprintf("%04o", info.Mode().Perm())}
		if d.IsDir() {
			e.Type = "directory"
		} else {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("fixture inventory requires a regular file: %s", path)
			}
			e.Type = "file"
			size := info.Size()
			e.Size = &size
			e.Hash = reviewHash(reviewRead(t, path))
		}
		s.Entries = append(s.Entries, e)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(s.Entries, func(i, j int) bool { return s.Entries[i].Path < s.Entries[j].Path })
	s.Hash = reviewScopeHash(*s, exclusion)
	return s
}
func reviewRow(t *testing.T, design, claim, kind string, paths ...string) reviewRecord {
	t.Helper()
	row := reviewRecord{Claim: claim, Kind: kind, Attestor: "R", Date: "2026-09-05"}
	for _, path := range paths {
		row.Covers = append(row.Covers, reviewCover{path, reviewHash(reviewRead(t, filepath.Join(design, path)))})
	}
	return row
}
func (f *reviewFixture) save(t *testing.T) {
	t.Helper()
	b, err := json.MarshalIndent(f.doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	reviewWrite(t, filepath.Join(f.design, AttestationsFileName), string(b)+"\n")
}
func newReviewFixture(t *testing.T, topology string) *reviewFixture {
	t.Helper()
	root := t.TempDir()
	f := &reviewFixture{design: filepath.Join(root, "design"), impl: filepath.Join(root, "src")}
	switch topology {
	case "ancestor":
		f.impl = root
	case "equal":
		f.impl = f.design
	case "inside":
		f.impl = filepath.Join(f.design, "src")
	}
	reviewWrite(t, filepath.Join(f.design, "BUILD.md"), "# Build\n")
	for path, body := range map[string]string{
		"handler.go": reviewHandler, "handler_test.go": reviewTest, "config.yaml": "enabled: true\n", ".gitignore": ".ignored/\nvendor/\nbuild/\n",
		".ignored/handler": "hidden behavior\n", "vendor/input": "vendor behavior\n", "build/generated_test": "assert expected action\n", "evidence/attestations.yaml": "ordinary application input\n",
	} {
		reviewWrite(t, filepath.Join(f.impl, path), body)
	}
	if err := os.MkdirAll(filepath.Join(f.impl, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	current := reviewRow(t, f.design, "gt.conformance-test-shape", "current", "BUILD.md")
	current.Implementation = reviewInventory(t, f.design, f.impl)
	f.doc = reviewDocument{2, []reviewRecord{current, reviewRow(t, f.design, "g4.zero-context", "plan", "BUILD.md")}}
	f.save(t)
	return f
}
func reviewGate(t *testing.T, run []*Gate) *Gate {
	t.Helper()
	for _, g := range run {
		if strings.HasPrefix(g.Title, "Gv-attest") {
			return g
		}
	}
	t.Fatalf("Gv did not run: %+v", run)
	return nil
}
func reviewRun(t *testing.T, f *reviewFixture) *Gate {
	t.Helper()
	run := RunSelected(f.design, f.impl, Selection{Run: map[string]bool{"gv": true}, Explicit: true}, RunOptions{})
	if len(run) == 1 && strings.HasPrefix(run[0].Title, "G0-snapshot") {
		return run[0]
	}
	return reviewGate(t, run)
}
func reviewNoCurrent(t *testing.T, g *Gate) {
	t.Helper()
	for key, n := range g.Counts {
		if n > 0 && (strings.Contains(key, "implementation") || strings.Contains(key, "scope")) {
			t.Errorf("published current scope counter %q=%d despite pending/invalid review", key, n)
		}
	}
}
func reviewGood(t *testing.T, g *Gate) {
	t.Helper()
	if len(g.Errs)+len(g.Drift)+len(g.Warns) != 0 || g.Counts["current implementation reviews"] != 1 {
		t.Fatalf("valid-v2 unchanged control failed; mutation NOT YET EXERCISED: errors=%v warnings=%v drift=%v counts=%v", g.Errs, g.Warns, g.Drift, g.Counts)
	}
	if !strings.Contains(strings.Join(g.Notes, "\n"), reviewLimits) {
		t.Fatalf("current review omits exact scope limits: %v", g.Notes)
	}
}
func reviewFailure(t *testing.T, g *Gate, category, path string) {
	t.Helper()
	errs := strings.Join(g.Errs, "\n")
	custody := category == "GV_SCOPE_CUSTODY" && strings.HasPrefix(g.Title, "G0-snapshot") && strings.Contains(strings.ToLower(errs), "symlink")
	if (!strings.Contains(errs, category) && !custody) || (path != "" && !strings.Contains(errs, path)) {
		t.Fatalf("want %s naming %q, got %v", category, path, g.Errs)
	}
	if len(g.Drift) != 0 {
		t.Errorf("review failure is not regenerable DRIFT: %v", g.Drift)
	}
	reviewNoCurrent(t, g)
}

func TestAttestationARejectsLegacyBehavior(t *testing.T) {
	for _, claim := range []string{"gt.conformance-test-shape", "g4.standin-coverage", "g4.pack-event-discipline"} {
		for _, mutated := range []bool{false, true} {
			for _, route := range []string{"direct", "suite-with-impl", "suite-without-impl"} {
				t.Run(fmt.Sprintf("%s/mutated=%t/%s", claim, mutated, route), func(t *testing.T) {
					f := newReviewFixture(t, "disjoint")
					paths := []string{"BUILD.md"}
					if claim == "g4.pack-event-discipline" {
						reviewWrite(t, filepath.Join(f.design, "pack", "pack.yaml"), "events: []\n")
						paths = []string{"pack/pack.yaml"}
					}
					f.doc = reviewDocument{1, []reviewRecord{reviewRow(t, f.design, claim, "", paths...)}}
					f.save(t)
					if mutated {
						reviewWrite(t, filepath.Join(f.impl, "handler.go"), "package example\nfunc Handle() string { return \"forbidden\" }\n")
						reviewWrite(t, filepath.Join(f.impl, "handler_test.go"), "package example\n// ORACLESET{machines/Thing.oracle.md}\n")
					}
					g := CheckAttestations(f.design)
					if route != "direct" {
						impl := f.impl
						if route == "suite-without-impl" {
							impl = ""
						}
						g = reviewGate(t, RunSelected(f.design, impl, Selection{Run: map[string]bool{"gv": true}, Explicit: true}, RunOptions{}))
					}
					t.Logf("A observed legacy errors=%v counts=%v mutated=%t", g.Errs, g.Counts, mutated)
					reviewFailure(t, g, "GV_MISSING_IMPLEMENTATION_SUBJECT", claim)
					for _, word := range []string{"machinery attest", "--impl", "current", "plan"} {
						if !strings.Contains(strings.Join(g.Errs, "\n"), word) {
							t.Errorf("migration remedy omits %q: %v", word, g.Errs)
						}
					}
				})
			}
		}
	}
}

func TestAttestationDBaselinePlanControls(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(fmt.Sprintf("partial=%t", partial), func(t *testing.T) {
			rows := attestAllG2()
			if partial {
				rows = rows[:1]
			}
			design, _ := attestFixture(t, attestEvidence(rows...))
			g := CheckAttestations(design)
			wantWarnings := 0
			if partial {
				wantWarnings = 5
			}
			if len(g.Errs)+len(g.Drift) != 0 || len(g.Warns) != wantWarnings || g.Counts["attested claims"] != len(rows) {
				t.Fatalf("D compatibility errors=%v warnings=%v counts=%v", g.Errs, g.Warns, g.Counts)
			}
			reviewNoCurrent(t, g)
		})
	}
}

func TestAttestationBIndependentDigestVector(t *testing.T) {
	size := int64(16)
	hash := "sha256:e0e0431b63a883552b05817d33ba13f79262019cdefb4e5ae060c299c1de1eb8"
	s := reviewScope{Root: "../src", Policy: "full-root-v1", Entries: []reviewEntry{{Path: ".", Type: "directory", Mode: "0755"}, {Path: "handler.go", Type: "file", Mode: "0644", Size: &size, Hash: hash}, {Path: "handler_test.go", Type: "file", Mode: "0644", Size: &size, Hash: hash}}}
	if got := reviewScopeHash(s, "none"); got != "sha256:4e0ca743a34c585082fdcfb7f57ca6a47a1e3448510c3033f5dd7281a351ac2a" {
		t.Fatalf("independent reviewed digest vector = %s", got)
	}
}

func TestAttestationCScopeMutations(t *testing.T) {
	cases := []struct {
		name, category, path string
		mutate               func(*testing.T, *reviewFixture)
	}{
		{"remove-assertion", "GV_STALE_CONTENT", "handler_test.go", func(t *testing.T, f *reviewFixture) {
			reviewWrite(t, filepath.Join(f.impl, "handler_test.go"), "package example\n// ORACLESET{machines/Thing.oracle.md}\n")
		}},
		{"alter-handler", "GV_STALE_CONTENT", "handler.go", func(t *testing.T, f *reviewFixture) {
			reviewWrite(t, filepath.Join(f.impl, "handler.go"), strings.ReplaceAll(reviewHandler, "accepted", "rejected"))
		}},
		{"gitignore-policy", "GV_STALE_CONTENT", ".gitignore", func(t *testing.T, f *reviewFixture) { reviewWrite(t, filepath.Join(f.impl, ".gitignore"), "**\n") }},
		{"file-mode", "GV_STALE_CONTENT", "handler.go", func(t *testing.T, f *reviewFixture) {
			if err := os.Chmod(filepath.Join(f.impl, "handler.go"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"directory-mode", "GV_STALE_CONTENT", "empty", func(t *testing.T, f *reviewFixture) {
			if err := os.Chmod(filepath.Join(f.impl, "empty"), 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{"nested-git", "GV_SCOPE_UNSUPPORTED_METADATA", ".git", func(t *testing.T, f *reviewFixture) {
			reviewWrite(t, filepath.Join(f.impl, "empty", ".git", "handler.go"), reviewHandler)
		}},
		{"file-symlink", "GV_SCOPE_CUSTODY", "handler.go", func(t *testing.T, f *reviewFixture) {
			p := filepath.Join(f.impl, "handler.go")
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(f.impl, "config.yaml"), p); err != nil {
				t.Fatal(err)
			}
		}},
		{"directory-symlink", "GV_SCOPE_CUSTODY", "empty", func(t *testing.T, f *reviewFixture) {
			p := filepath.Join(f.impl, "empty")
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(t.TempDir(), p); err != nil {
				t.Fatal(err)
			}
		}},
		{"root-symlink", "GV_SCOPE_CUSTODY", "src", func(t *testing.T, f *reviewFixture) {
			old := f.impl + "-held"
			if err := os.Rename(f.impl, old); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(old, f.impl); err != nil {
				t.Fatal(err)
			}
		}},
		{"hardlink", "GV_SCOPE_ALIAS", "handler", func(t *testing.T, f *reviewFixture) {
			if err := os.Link(filepath.Join(f.impl, "handler.go"), filepath.Join(f.impl, "handler-alias.go")); err != nil {
				t.Fatalf("required hardlink fixture: %v", err)
			}
		}},
		{"fifo", "GV_SCOPE_CUSTODY", "pipe", func(t *testing.T, f *reviewFixture) {
			if err := syscall.Mkfifo(filepath.Join(f.impl, "pipe"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"scope-narrowing", "GV_SCOPE_ROOT", "empty", func(t *testing.T, f *reviewFixture) { f.impl = filepath.Join(f.impl, "empty") }},
	}
	for _, path := range []string{"extra.go", "extra_test.go", "extra.yaml", "LICENSE", ".ignored/new-handler", "vendor/new-handler", "build/new_test", "evidence/other.yaml", "other/attestations.yaml"} {
		p := path
		cases = append(cases, struct {
			name, category, path string
			mutate               func(*testing.T, *reviewFixture)
		}{"add-" + p, "GV_SCOPE_INVENTORY", p, func(t *testing.T, f *reviewFixture) { reviewWrite(t, filepath.Join(f.impl, p), "new behavior\n") }})
	}
	for _, path := range []string{"handler.go", "handler_test.go", "config.yaml", ".ignored/handler", "build/generated_test"} {
		for _, op := range []string{"delete", "rename"} {
			p, o := path, op
			cases = append(cases, struct {
				name, category, path string
				mutate               func(*testing.T, *reviewFixture)
			}{o + "-" + p, "GV_SCOPE_INVENTORY", p, func(t *testing.T, f *reviewFixture) {
				var err error
				if o == "delete" {
					err = os.Remove(filepath.Join(f.impl, p))
				} else {
					err = os.Rename(filepath.Join(f.impl, p), filepath.Join(f.impl, p+".renamed"))
				}
				if err != nil {
					t.Fatal(err)
				}
			}})
		}
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newReviewFixture(t, "disjoint")
			reviewGood(t, reviewRun(t, f))
			t.Log("C unchanged control passed; applying mutation")
			tc.mutate(t, f)
			g := reviewRun(t, f)
			reviewFailure(t, g, tc.category, tc.path)
		})
	}
}

func TestAttestationCTopologyAndEvidenceBoundary(t *testing.T) {
	for _, topology := range []string{"disjoint", "ancestor", "equal", "inside"} {
		for _, operation := range []string{"unchanged", "evidence-only", "design-code-change", "design-code-add"} {
			t.Run(topology+"/"+operation, func(t *testing.T) {
				f := newReviewFixture(t, topology)
				// Place a behavioral input inside the design only when it is in scope.
				path := filepath.Join(f.impl, "helper.go")
				if topology == "ancestor" || topology == "equal" {
					path = filepath.Join(f.design, "helpers", "test.go")
				}
				reviewWrite(t, path, reviewTest)
				f.doc.Rows[0].Implementation = reviewInventory(t, f.design, f.impl)
				f.save(t)
				reviewGood(t, reviewRun(t, f))
				oldHash := f.doc.Rows[0].Implementation.Hash
				switch operation {
				case "evidence-only":
					f.doc.Rows[0].Note = "review attribution corrected"
					f.save(t)
				case "design-code-change":
					reviewWrite(t, path, "package example\n")
				case "design-code-add":
					path = filepath.Join(filepath.Dir(path), "new_test.go")
					reviewWrite(t, path, reviewTest)
				}
				g := reviewRun(t, f)
				if operation == "unchanged" || operation == "evidence-only" {
					reviewGood(t, g)
					if got := reviewInventory(t, f.design, f.impl).Hash; got != oldHash {
						t.Fatalf("reserved evidence-only update changed digest %s -> %s", oldHash, got)
					}
				} else {
					category := "GV_STALE_CONTENT"
					if operation == "design-code-add" {
						category = "GV_SCOPE_INVENTORY"
					}
					reviewFailure(t, g, category, filepath.Base(path))
				}
			})
		}
	}
}

func TestAttestationCReceiptCannotNarrowScope(t *testing.T) {
	for _, change := range []string{"omit-old-hash", "omit-rehashed", "root", "policy", "exclude", "duplicate", "dot-alias", "case-alias", "unsorted", "missing-parent", "self-reference"} {
		t.Run(change, func(t *testing.T) {
			f := newReviewFixture(t, "disjoint")
			reviewGood(t, reviewRun(t, f))
			s := f.doc.Rows[0].Implementation
			category := "GV_SCOPE_PATH"
			switch change {
			case "omit-old-hash", "omit-rehashed":
				for i, e := range s.Entries {
					if e.Path == "handler_test.go" {
						s.Entries = append(s.Entries[:i], s.Entries[i+1:]...)
						break
					}
				}
				category = "GV_SCOPE_HASH"
				if change == "omit-rehashed" {
					s.Hash = reviewScopeHash(*s, "none")
					category = "GV_SCOPE_INVENTORY"
				}
			case "root":
				s.Root = "../src/empty"
				category = "GV_SCOPE_ROOT"
			case "policy":
				s.Policy = "include-only-tests"
				category = "GV_SCHEMA"
			case "duplicate":
				s.Entries = append(s.Entries, s.Entries[len(s.Entries)-1])
			case "dot-alias":
				s.Entries[1].Path = "./" + s.Entries[1].Path
			case "case-alias":
				e := s.Entries[len(s.Entries)-1]
				e.Path = strings.ToUpper(e.Path)
				s.Entries = append(s.Entries, e)
			case "unsorted":
				s.Entries[0], s.Entries[1] = s.Entries[1], s.Entries[0]
			case "missing-parent":
				s.Entries[1].Path = "unlisted/entry"
			case "self-reference":
				f.doc.Rows[0].Covers = append(f.doc.Rows[0].Covers, reviewCover{AttestationsFileName, reviewHash(reviewRead(t, filepath.Join(f.design, AttestationsFileName)))})
				category = "GV_SCHEMA"
			case "exclude":
				category = "GV_SCHEMA"
			}
			f.save(t)
			if change == "exclude" {
				p := filepath.Join(f.design, AttestationsFileName)
				body := string(reviewRead(t, p))
				reviewWrite(t, p, strings.Replace(body, "\"policy\":", "\"exclude\": [\"**\"], \"policy\":", 1))
			}
			reviewFailure(t, reviewRun(t, f), category, "")
		})
	}
}

func TestAttestationBKindAndMissingRoot(t *testing.T) {
	for _, kind := range []string{"current", "plan", "historical"} {
		t.Run(kind, func(t *testing.T) {
			f := newReviewFixture(t, "disjoint")
			f.doc.Rows[0].Kind = kind
			if kind != "current" {
				f.doc.Rows[0].Implementation = nil
			}
			f.save(t)
			g := CheckAttestations(f.design)
			switch kind {
			case "current":
				reviewFailure(t, g, "GV_IMPL_REQUIRED", "")
			case "historical":
				reviewFailure(t, g, "GV_KIND", "")
			case "plan":
				if len(g.Errs) != 0 || !strings.Contains(strings.Join(g.Warns, "\n"), "plan only; current implementation review missing") {
					t.Fatalf("behavioral plan must remain incomplete: %v %v", g.Errs, g.Warns)
				}
				reviewNoCurrent(t, g)
			}
		})
	}
}

func TestAttestationCReleaseFinalizesAllOrNone(t *testing.T) {
	for _, fault := range []string{"none", "original-content", "root-replacement", "supplementary", "latched-restoration"} {
		t.Run(fault, func(t *testing.T) {
			f := newReviewFixture(t, "disjoint")
			reviewGood(t, reviewRun(t, f))
			s, err := AcquireSnapshot(f.design)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = s.Release() })
			external := filepath.Join(t.TempDir(), "routing.yaml")
			reviewWrite(t, external, "original\n")
			if err := s.TrackExternal(external); err != nil {
				t.Fatal(err)
			}
			sel := Selection{Run: map[string]bool{"gv": true}, Explicit: true}
			var pending []*Gate
			for i := 0; i < 2; i++ {
				g := reviewGate(t, s.RunSelected(f.impl, sel, RunOptions{}))
				if len(g.Errs) != 0 {
					t.Fatalf("pending valid review errors=%v", g.Errs)
				}
				reviewNoCurrent(t, g)
				if !strings.Contains(strings.Join(g.Notes, "\n"), "current implementation review pending final snapshot release") {
					t.Fatalf("pending state not visible: %v", g.Notes)
				}
				pending = append(pending, g)
			}
			switch fault {
			case "original-content":
				reviewWrite(t, filepath.Join(f.impl, "handler.go"), "package example\n")
			case "root-replacement":
				if err := os.Rename(f.impl, f.impl+"-parked"); err != nil {
					t.Fatal(err)
				}
				reviewWrite(t, filepath.Join(f.impl, "handler.go"), reviewHandler)
			case "supplementary", "latched-restoration":
				reviewWrite(t, external, "changed!\n")
				if fault == "latched-restoration" {
					if err := s.CheckUnchanged(); err == nil {
						t.Fatal("changed tracked external file was not observed")
					}
					reviewWrite(t, external, "original\n")
				}
			}
			err = s.Release()
			if fault == "none" {
				if err != nil {
					t.Fatal(err)
				}
				for _, g := range pending {
					reviewGood(t, g)
				}
			} else {
				if err == nil {
					t.Fatal("late custody failure disappeared at Release")
				}
				for _, g := range pending {
					reviewFailure(t, g, "GV_SCOPE_CUSTODY", "")
				}
			}
			before := fmt.Sprint(pending[0].Counts, pending[0].Notes, pending[0].Errs)
			again := s.Release()
			if fmt.Sprint(again) != fmt.Sprint(err) || before != fmt.Sprint(pending[0].Counts, pending[0].Notes, pending[0].Errs) {
				t.Fatal("repeated Release changed final disposition or findings")
			}
			for _, g := range pending {
				text := strings.Join(append(append([]string{}, g.Errs...), g.Notes...), "\n")
				for _, private := range []string{"machinery-design-source-", "machinery-impl-snapshot-", "machinery-attestation-"} {
					if strings.Contains(text, private) {
						t.Errorf("private snapshot path leaked: %s", text)
					}
				}
			}
			late := s.RunSelected(f.impl, sel, RunOptions{})
			errs := ""
			for _, g := range late {
				errs += strings.Join(g.Errs, "\n")
				reviewNoCurrent(t, g)
			}
			if !strings.Contains(strings.ToLower(errs), "releas") {
				t.Fatalf("RunSelected after release did not fail closed: %s", errs)
			}
		})
	}
}
