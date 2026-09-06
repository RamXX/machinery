package gates

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func greenReview() AttestationReview {
	return AttestationReview{Claim: "gt.conformance-test-shape", Kind: "current", Attestor: "reviewer", Date: "2026-09-03"}
}

func greenAttestExercise(t *testing.T, route, fault, temporary string) {
	t.Helper()
	f := newReviewFixture(t, "disjoint")
	sentinel := filepath.Join(t.TempDir(), "sentinel")
	reviewWrite(t, sentinel, "must survive")
	prior := attestationBeforeFinalRelease
	t.Cleanup(func() { attestationBeforeFinalRelease = prior })
	fired := 0
	var observed *Snapshot
	attestationBeforeFinalRelease = func(s *Snapshot) {
		fired++
		observed = s
		for _, pending := range s.attestationPending {
			reviewNoCurrent(t, pending.gate)
		}
		switch fault {
		case "original":
			reviewWrite(t, filepath.Join(f.impl, "handler.go"), "package example\n// changed at first final release\n")
		case "cleanup":
			candidate := filepath.Join(s.DesignPath(), "BUILD.md")
			rel, err := filepath.Rel(temporary, candidate)
			if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") || candidate == filepath.Join(f.design, "BUILD.md") {
				t.Fatalf("unowned fault target: %s", candidate)
			}
			info, err := os.Lstat(candidate)
			if err != nil || !info.Mode().IsRegular() {
				t.Fatalf("fault target not regular: %v", err)
			}
			if err := os.Remove(candidate); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(sentinel, candidate); err != nil {
				t.Fatal(err)
			}
		}
	}
	var g *Gate
	var body []byte
	var err error
	switch route {
	case "Render":
		body, err = RenderAttestation(f.design, f.impl, greenReview())
	case "WithImplementation":
		g = CheckAttestationsWithImplementation(f.design, f.impl)
	case "RunSelected":
		g = reviewGate(t, RunSelected(f.design, f.impl, Selection{Run: map[string]bool{"gv": true}, Explicit: true}, RunOptions{}))
	case "SelectRunAndNote":
		var run []*Gate
		_, run, _, err = SelectRunAndNote(f.design, f.impl, "gv", RunOptions{})
		if len(run) > 0 {
			g = reviewGate(t, run)
		}
	default:
		t.Fatal("unknown route")
	}
	if fired != 1 || observed == nil {
		t.Fatalf("callback=%d; actual finalization not reached", fired)
	}
	if got := string(reviewRead(t, sentinel)); got != "must survive" {
		t.Fatalf("sentinel changed: %q", got)
	}
	if fault == "none" {
		if err != nil {
			t.Fatalf("matched unchanged control: %v", err)
		}
		if route == "Render" {
			if len(body) == 0 {
				t.Fatal("renderer control emitted no document")
			}
		} else {
			if g == nil {
				t.Fatal("control omitted Gv")
			}
			reviewGood(t, g)
		}
	} else {
		text := fmt.Sprint(err)
		if g != nil {
			text += strings.Join(g.Errs, "\n")
			reviewNoCurrent(t, g)
		}
		if route == "Render" && (err == nil || body != nil) {
			t.Fatalf("renderer published before actual finalization fault: bytes=%d error=%v", len(body), err)
		}
		if fault == "original" && !strings.Contains(text, "changed") {
			t.Fatalf("original mutation cause missing: %s", text)
		}
		if fault == "cleanup" && (!strings.Contains(text, "private snapshot cleanup") || !strings.Contains(text, "symlink")) {
			t.Fatalf("real owned-copy cleanup cause missing: %s", text)
		}
		for _, prefix := range []string{"machinery-design-source-", "machinery-impl-snapshot-", "machinery-attestation-"} {
			if strings.Contains(text, prefix) {
				t.Fatalf("private path leaked: %s", text)
			}
		}
	}
	first := observed.Release()
	second := observed.Release()
	if fmt.Sprint(first) != fmt.Sprint(second) || fired != 1 || (first != nil) != (fault != "none") {
		t.Fatalf("repeated release changed result: %v / %v fired=%d", first, second, fired)
	}
	if observed.CheckUnchanged() == nil {
		t.Fatal("check after release accepted")
	}
	if _, err := observed.Select("gv", f.impl); err == nil {
		t.Fatal("select after release accepted")
	}
	late := observed.RunSelected(f.impl, Selection{Run: map[string]bool{"gv": true}, Explicit: true}, RunOptions{})
	if len(late) != 1 || len(late[0].Errs) == 0 {
		t.Fatal("run after release accepted")
	}
	t.Logf("OBSERVED route=%s fault=%s callback=%d renderer-bytes=%d error=%v", route, fault, fired, len(body), err)
}

// Each destructive copy challenge has a fired unchanged control in the same
// bounded subprocess. Its private TMPDIR is established before acquisition;
// process exit closes handles and the parent owns this exact temporary tree.
func TestAttestGreenFinalization(t *testing.T) {
	for i, arg := range os.Args {
		if arg == "attestation-green-child" {
			if len(os.Args) != i+4 || os.TempDir() != os.Args[i+3] {
				t.Fatal("invalid isolated child invocation")
			}
			route, fault, temporary := os.Args[i+1], os.Args[i+2], os.Args[i+3]
			greenAttestExercise(t, route, "none", temporary)
			if fault != "none" {
				t.Log("fired matched unchanged control passed; actual fault begins")
				greenAttestExercise(t, route, fault, temporary)
			}
			return
		}
	}
	for _, route := range []string{"Render", "WithImplementation", "RunSelected", "SelectRunAndNote"} {
		for _, fault := range []string{"none", "original", "cleanup"} {
			t.Run(route+"/"+fault, func(t *testing.T) {
				temporary := t.TempDir()
				ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAttestGreenFinalization$", "-test.v", "--", "attestation-green-child", route, fault, temporary)
				for _, e := range os.Environ() {
					if !strings.HasPrefix(e, "TMPDIR=") {
						cmd.Env = append(cmd.Env, e)
					}
				}
				cmd.Env = append(cmd.Env, "TMPDIR="+temporary)
				out, err := cmd.CombinedOutput()
				t.Logf("native helper output:\n%s", out)
				if ctx.Err() != nil || err != nil {
					t.Fatalf("native finalization assertion: %v timeout=%v", err, ctx.Err())
				}
			})
		}
	}
}

func TestAttestGreenEvidenceLimits(t *testing.T) {
	const limit = 16 << 20
	if designArtifactMaxBytes != limit {
		t.Fatal("fixed design/document limit changed")
	}
	t.Run("generated-document-at-and-plus-one", func(t *testing.T) {
		f := newReviewFixture(t, "disjoint")
		r := greenReview()
		r.Note = "x"
		base, err := RenderAttestation(f.design, f.impl, r)
		if err != nil {
			t.Fatal(err)
		}
		r.Note = strings.Repeat("x", limit-len(base)+1)
		body, err := RenderAttestation(f.design, f.impl, r)
		if err != nil || len(body) != limit {
			t.Fatalf("actual exact-limit control bytes=%d error=%v", len(body), err)
		}
		r.Note += "x"
		body, err = RenderAttestation(f.design, f.impl, r)
		if body != nil || err == nil || !strings.Contains(err.Error(), "GV_EVIDENCE_LIMIT") {
			t.Fatalf("actual generated plus-one accepted: bytes=%d error=%v", len(body), err)
		}
	})
	t.Run("design-cover-at-and-plus-one", func(t *testing.T) {
		f := newReviewFixture(t, "disjoint")
		r := greenReview()
		r.Kind = "plan"
		reviewWrite(t, filepath.Join(f.design, "BUILD.md"), strings.Repeat("x", limit))
		body, err := RenderAttestation(f.design, "", r)
		if err != nil || len(body) == 0 {
			t.Fatalf("actual exact-limit cover control: %v", err)
		}
		reviewWrite(t, filepath.Join(f.design, "BUILD.md"), strings.Repeat("x", limit+1))
		body, err = RenderAttestation(f.design, "", r)
		if body != nil || err == nil || !strings.Contains(err.Error(), "GV_EVIDENCE_LIMIT") {
			t.Fatalf("actual cover plus-one accepted: bytes=%d error=%v", len(body), err)
		}
	})
	t.Run("input-document-at-and-plus-one", func(t *testing.T) {
		f := newReviewFixture(t, "disjoint")
		r := greenReview()
		r.Kind = "plan"
		base, err := RenderAttestation(f.design, "", r)
		if err != nil {
			t.Fatal(err)
		}
		body := string(base) + "_comment: " + strings.Repeat("x", limit-len(base)-len("_comment: \n")) + "\n"
		if len(body) != limit {
			t.Fatalf("fixture length=%d", len(body))
		}
		path := filepath.Join(f.design, AttestationsFileName)
		reviewWrite(t, path, body)
		g := CheckAttestations(f.design)
		if len(g.Errs) != 0 {
			t.Fatalf("actual input exact-limit control: %v", g.Errs)
		}
		reviewWrite(t, path, body+"\n")
		g = CheckAttestations(f.design)
		if !strings.Contains(strings.Join(g.Errs, "\n"), "GV_EVIDENCE_LIMIT") {
			t.Fatalf("actual input plus-one accepted: %v", g.Errs)
		}
	})
}

func TestAttestGreenExactSchema(t *testing.T) {
	for _, field := range []string{"claim", "date", "cover-hash"} {
		t.Run(field, func(t *testing.T) {
			f := newReviewFixture(t, "disjoint")
			reviewGood(t, CheckAttestationsWithImplementation(f.design, f.impl))
			switch field {
			case "claim":
				f.doc.Rows[0].Claim = " " + f.doc.Rows[0].Claim
			case "date":
				f.doc.Rows[0].Date += " "
			case "cover-hash":
				f.doc.Rows[0].Covers[0].Hash += " "
			}
			f.save(t)
			g := CheckAttestationsWithImplementation(f.design, f.impl)
			reviewFailure(t, g, "GV_SCHEMA", "")
		})
	}
}
