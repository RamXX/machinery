package designlock

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func attestationWrite(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func attestationFixture(t *testing.T, topology string) (string, string, *Lock) {
	t.Helper()
	base := t.TempDir()
	design, impl := filepath.Join(base, "design"), filepath.Join(base, "impl")
	switch topology {
	case "ancestor":
		impl = base
	case "equal":
		impl = design
	case "inside":
		impl = filepath.Join(design, "impl")
	}
	attestationWrite(t, filepath.Join(design, "BUILD.md"), []byte("build"))
	attestationWrite(t, filepath.Join(impl, "input"), []byte("input"))
	l, err := AcquireReader(design)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := l.Release(); err != nil {
			t.Error(err)
		}
	})
	return design, impl, l
}

func TestAttestationCapabilityTopologies(t *testing.T) {
	for _, topology := range []string{"disjoint", "ancestor", "equal", "inside"} {
		t.Run(topology, func(t *testing.T) {
			design, impl, l := attestationFixture(t, topology)
			s, err := l.MaterializeAttestationTree(impl)
			if err != nil {
				t.Fatal(err)
			}
			if s.Logical() != impl || s.Path() == impl {
				t.Fatalf("not an isolated logical subject: %s %s", s.Logical(), s.Path())
			}
			entries := s.Entries()
			if len(entries) < 2 || entries[0].Path != "." || !entries[0].Directory {
				t.Fatalf("incomplete inventory: %+v", entries)
			}
			found := false
			for _, e := range entries {
				if e.Path == "input" {
					found = !e.Directory && e.Size == 5 && e.SHA256 == sha256.Sum256([]byte("input"))
				}
			}
			if !found {
				t.Fatal("input bytes not bound")
			}
			if topology == "ancestor" || topology == "equal" {
				want, _ := filepath.Rel(impl, filepath.Join(design, "BUILD.md"))
				found = false
				for _, e := range entries {
					if e.Path == filepath.ToSlash(want) {
						found = e.SHA256 == sha256.Sum256([]byte("build"))
					}
				}
				if !found {
					t.Fatal("original design overlay missing")
				}
			}
			entries[0].Path = "corrupted"
			if s.Entries()[0].Path != "." {
				t.Fatal("Entries leaked mutable authority")
			}
			if err := s.CheckUnchanged(); err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil || s.Entries() != nil || s.CheckUnchanged() == nil {
				t.Fatalf("release not idempotent/fail closed: %v", err)
			}
			if err := l.Release(); err != nil {
				t.Fatal(err)
			}
			if captured, err := l.MaterializeAttestationTree(impl); err == nil || captured != nil {
				t.Fatal("capture after lock release succeeded")
			}
		})
	}
}

func TestAttestationCapabilityRetainedAuthority(t *testing.T) {
	for _, fault := range []string{"original-file", "design-file-identity", "original-root", "design-root", "owned-copy"} {
		t.Run(fault, func(t *testing.T) {
			design, impl, l := attestationFixture(t, "disjoint")
			s, err := l.MaterializeAttestationTree(impl)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := s.Close(); err != nil {
					t.Error(err)
				}
			}()
			if err := s.CheckUnchanged(); err != nil {
				t.Fatalf("unchanged control: %v", err)
			}
			switch fault {
			case "original-file":
				attestationWrite(t, filepath.Join(impl, "input"), []byte("other"))
			case "owned-copy":
				attestationWrite(t, filepath.Join(s.Path(), "input"), []byte("other"))
			case "design-file-identity":
				p := filepath.Join(design, "BUILD.md")
				info, err := os.Stat(p)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(p, p+".old"); err != nil {
					t.Fatal(err)
				}
				attestationWrite(t, p, []byte("build"))
				if err := os.Chtimes(p, info.ModTime(), info.ModTime()); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(p + ".old"); err != nil {
					t.Fatal(err)
				}
			default:
				root := impl
				if fault == "design-root" {
					root = design
				}
				if err := os.Rename(root, root+"-original"); err != nil {
					t.Fatal(err)
				}
				attestationWrite(t, filepath.Join(root, "replacement"), []byte("replacement"))
			}
			if err := s.CheckUnchanged(); err == nil || !strings.Contains(err.Error(), "GV_SCOPE_CUSTODY") {
				t.Fatalf("actual %s not rejected: %v", fault, err)
			}
			t.Logf("unchanged control then actual %s rejected", fault)
		})
	}
	t.Run("original-generation-gap", func(t *testing.T) {
		design, impl, l := attestationFixture(t, "disjoint")
		attestationWrite(t, filepath.Join(design, "BUILD.md"), []byte("changed after reader acquisition"))
		if s, err := l.MaterializeAttestationTree(impl); err == nil || s != nil || !strings.Contains(err.Error(), "generation") {
			t.Fatalf("generation gap accepted: %v", err)
		}
	})
}

func TestAttestationCapabilityReadChunk(t *testing.T) {
	for _, fault := range []string{"none", "file", "root"} {
		t.Run(fault, func(t *testing.T) {
			_, impl, l := attestationFixture(t, "disjoint")
			path := filepath.Join(impl, "input")
			attestationWrite(t, path, bytes.Repeat([]byte("a"), 2*fingerprintBufferBytes))
			prior := testAfterSnapshotCopyReadChunk
			t.Cleanup(func() { testAfterSnapshotCopyReadChunk = prior })
			fired := 0
			testAfterSnapshotCopyReadChunk = func(label string) {
				if label != path {
					return
				}
				fired++
				if fired != 1 {
					return
				}
				switch fault {
				case "file":
					attestationWrite(t, path, bytes.Repeat([]byte("b"), 2*fingerprintBufferBytes))
				case "root":
					if err := os.Rename(impl, impl+"-original"); err != nil {
						t.Fatal(err)
					}
					attestationWrite(t, path, bytes.Repeat([]byte("a"), 2*fingerprintBufferBytes))
				}
			}
			s, err := l.MaterializeAttestationTree(impl)
			if fired == 0 {
				t.Fatal("real first-chunk callback never fired")
			}
			if fault == "none" {
				if err != nil {
					t.Fatal(err)
				}
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
			} else if err == nil || s != nil {
				t.Fatalf("actual mid-read %s accepted", fault)
			}
			t.Logf("real bounded copy callback=%d fault=%s result=%v", fired, fault, err)
		})
	}
}

func TestAttestationCapabilityFixedLimits(t *testing.T) {
	defaults := newAttestationSnapshotBudget()
	if defaults.maxEntries != 100000 || defaults.maxDepth != 64 || defaults.maxBytes != 8<<30 || snapshotRegularFileMaxBytes != 1<<30 {
		t.Fatalf("fixed production limits changed: %+v", defaults)
	}
	for _, limit := range []string{"entries", "depth", "aggregate", "aggregate-overlay"} {
		for _, plus := range []bool{false, true} {
			name := limit + "/at"
			if plus {
				name = limit + "/plus-one"
			}
			t.Run(name, func(t *testing.T) {
				topology := "disjoint"
				if limit == "aggregate-overlay" {
					topology = "ancestor"
				}
				_, impl, l := attestationFixture(t, topology)
				budget := defaults
				want := ""
				switch limit {
				case "entries":
					budget.maxEntries = 2
					want = "2-entry"
					if plus {
						attestationWrite(t, filepath.Join(impl, "extra"), nil)
					}
				case "depth":
					budget.maxDepth = 2
					want = "2-level"
					path := filepath.Join(impl, "a", "b")
					if plus {
						path = filepath.Join(path, "c")
					}
					if err := os.MkdirAll(path, 0o755); err != nil {
						t.Fatal(err)
					}
				default:
					budget.maxBytes = 5
					if limit == "aggregate-overlay" {
						budget.maxBytes = 10
					}
					want = "aggregate limit"
					if plus {
						attestationWrite(t, filepath.Join(impl, "input"), []byte("input!"))
					}
				}
				prior := newAttestationSnapshotBudget
				t.Cleanup(func() { newAttestationSnapshotBudget = prior })
				newAttestationSnapshotBudget = func() snapshotBudget { return budget }
				s, err := l.MaterializeAttestationTree(impl)
				if plus {
					if err == nil || s != nil || !strings.Contains(err.Error(), want) {
						t.Fatalf("real plus-one limit=%s: %v", limit, err)
					}
				} else {
					if err != nil {
						t.Fatalf("real at-limit control=%s: %v", limit, err)
					}
					if err := s.CheckUnchanged(); err != nil {
						t.Fatal(err)
					}
					if err := s.Close(); err != nil {
						t.Fatal(err)
					}
				}
				t.Logf("actual inventory limit=%s plus-one=%t error=%v", limit, plus, err)
			})
		}
	}
	t.Run("sparse-file-plus-one", func(t *testing.T) {
		_, impl, l := attestationFixture(t, "disjoint")
		path := filepath.Join(impl, "oversize")
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := file.Truncate((1 << 30) + 1); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		prior := testAfterSnapshotCopyReadChunk
		t.Cleanup(func() { testAfterSnapshotCopyReadChunk = prior })
		readOversize := false
		testAfterSnapshotCopyReadChunk = func(label string) {
			if label == path {
				readOversize = true
			}
		}
		s, err := l.MaterializeAttestationTree(impl)
		if err == nil || s != nil || readOversize || !strings.Contains(err.Error(), "1073741824") {
			t.Fatalf("oversize was not rejected before read: %v read=%t", err, readOversize)
		}
	})
}
