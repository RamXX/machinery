package gates

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/designlock"
	"github.com/RamXX/machinery/internal/ir"
)

func TestEmbedInventoryRejectsEntryBeyondFixedCeiling(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"c", "a", "b"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	f, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() //nolint:errcheck // test cleanup
	if _, err := readEmbedTxDir(f, 2); err == nil || !strings.Contains(err.Error(), "entry limit") {
		t.Fatalf("high-entry embed inventory was accepted: %v", err)
	}
}

func TestEmbedReadStateRejectsContinuousAppender(t *testing.T) {
	dir := t.TempDir()
	name := "target.md"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 1<<20)), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	tx := &embedRootTransaction{design: dir, root: root}
	old := embedReadStateAfterOpen
	t.Cleanup(func() { embedReadStateAfterOpen = old })
	done := make(chan struct{})
	stopped := make(chan struct{})
	embedReadStateAfterOpen = func(rel string) {
		if rel != name {
			return
		}
		embedReadStateAfterOpen = func(string) {}
		first := make(chan struct{})
		go func() {
			defer close(stopped)
			f, openErr := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
			if openErr != nil {
				close(first)
				return
			}
			defer f.Close() //nolint:errcheck // test mutation
			for i := 0; ; i++ {
				_, _ = f.Write([]byte("growth"))
				if i == 0 {
					close(first)
				}
				select {
				case <-done:
					return
				default:
				}
			}
		}()
		<-first
	}
	_, err = tx.readState(name)
	close(done)
	<-stopped
	if err == nil || !strings.Contains(err.Error(), "changed while reading") {
		t.Fatalf("continuous appender was accepted: %v", err)
	}
}

// refreshDesign writes a two-document design: a source and one embedding
// document whose marker and table are given verbatim.
func refreshDesign(t *testing.T, source, embedding string) string {
	t.Helper()
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "ARCHITECTURE.md"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "SHARD.md"), []byte(embedding), 0644); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestRefreshEmbedsDryRunRejectsInterruptedPublicationBeforeReads(t *testing.T) {
	design := t.TempDir()
	journal := filepath.Join(design, "formal", ".machinery-formal-transaction.jsonl")
	if err := os.MkdirAll(filepath.Dir(journal), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(journal, []byte("seeded\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := RefreshEmbeds(design, true); err == nil || !strings.Contains(err.Error(), "interrupted Machinery publication") {
		t.Fatalf("embed --dry-run read through interrupted publication: %v", err)
	}
}

func shardText(t *testing.T, design string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(design, "SHARD.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const keyedSource = "# root\n\n" +
	"| name | note |\n|---|---|\n" +
	"| `alpha` | the alpha note, current |\n" +
	"| `beta` | the beta note, current |\n"

const eventSource = "# root\n\n" +
	"| event | producer | consumer | payload |\n|---|---|---|---|\n" +
	"| `job.done` | ops | core (sse lane) | id |\n" +
	"| `job.done` | ops | core (durable lane) | id + hash |\n" +
	"| `job.done` | ops | risk (durable lane) | id |\n"

func TestRefreshEmbeds(t *testing.T) {
	cases := []struct {
		name        string
		source      string
		embedding   string
		wantIn      []string
		wantNotIn   []string
		recopied    int
		appended    int
		orphans     int
		kept        int
		wantProblem string
	}{
		{
			name:      "a drifted row is re-copied from its source",
			source:    keyedSource,
			embedding: "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `alpha` | the alpha note, STALE |\n",
			wantIn:    []string{"| `alpha` | the alpha note, current |"},
			wantNotIn: []string{"STALE"},
			recopied:  1,
		},
		{
			name:      "an already-current row is left byte-identical",
			source:    keyedSource,
			embedding: "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `alpha` | the alpha note, current |\n",
			wantIn:    []string{"| `alpha` | the alpha note, current |"},
			recopied:  0,
		},
		{
			name:      "a localized row is never touched",
			source:    keyedSource,
			embedding: "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `alpha` (shard-local: this shard restates it) | wholly local |\n",
			wantIn:    []string{"| `alpha` (shard-local: this shard restates it) | wholly local |"},
			kept:      1,
			recopied:  0,
		},
		{
			name:      "a row with no source row is kept and reported",
			source:    keyedSource,
			embedding: "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `gamma` | renamed away |\n",
			wantIn:    []string{"| `gamma` | renamed away |"},
			orphans:   1,
		},
		{
			name:      "complete plus where appends the missing selected rows",
			source:    eventSource,
			embedding: "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"event,producer,consumer,payload\" where=\"consumer=core\" claims=\"subset,complete\" -->\n| event | producer | consumer | payload |\n|---|---|---|---|\n| `job.done` | ops | core (sse lane) | id |\n",
			wantIn:    []string{"| `job.done` | ops | core (durable lane) | id + hash |"},
			wantNotIn: []string{"risk (durable lane)"},
			appended:  1,
		},
		{
			name:      "subset alone never appends",
			source:    eventSource,
			embedding: "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"event,producer,consumer,payload\" where=\"consumer=core\" claims=\"subset\" -->\n| event | producer | consumer | payload |\n|---|---|---|---|\n| `job.done` | ops | core (sse lane) | id |\n",
			wantNotIn: []string{"durable lane"},
			appended:  0,
		},
		{
			name:      "event lanes sharing an event and a consumer stay distinct",
			source:    eventSource,
			embedding: "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"event,producer,consumer,payload\" claims=\"subset\" -->\n| event | producer | consumer | payload |\n|---|---|---|---|\n| `job.done` | ops | core (durable lane) | STALE |\n| `job.done` | ops | core (sse lane) | id |\n",
			wantIn:    []string{"| `job.done` | ops | core (durable lane) | id + hash |", "| `job.done` | ops | core (sse lane) | id |"},
			wantNotIn: []string{"STALE"},
			recopied:  1,
		},
		{
			name:        "a header that is not the source's is refused, not rewritten",
			source:      keyedSource,
			embedding:   "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note | extra |\n|---|---|---|\n| `alpha` | x | y |\n",
			wantIn:      []string{"| `alpha` | x | y |"},
			wantProblem: "does not carry the source's columns",
		},
		{
			name:        "a missing source is refused, not rewritten",
			source:      keyedSource,
			embedding:   "<!-- machinery:embed from=\"NOPE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `alpha` | x |\n",
			wantIn:      []string{"| `alpha` | x |"},
			wantProblem: "does not exist or is empty",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := refreshDesign(t, tc.source, tc.embedding)
			reports, changed, err := RefreshEmbeds(d, false)
			if err != nil {
				t.Fatal(err)
			}
			if len(reports) != 1 {
				t.Fatalf("reports = %d, want 1", len(reports))
			}
			r := reports[0]
			if tc.wantProblem != "" {
				if !strings.Contains(r.Problem, tc.wantProblem) {
					t.Fatalf("problem = %q, want it to name %q", r.Problem, tc.wantProblem)
				}
				if len(changed) != 0 {
					t.Fatalf("a refused marker changed files: %v", changed)
				}
			} else if r.Problem != "" {
				t.Fatalf("unexpected problem: %s", r.Problem)
			}
			if r.Recopied != tc.recopied || r.Appended != tc.appended || r.Kept != tc.kept || len(r.Orphans) != tc.orphans {
				t.Fatalf("report = (recopied %d, appended %d, kept %d, orphans %d), want (%d, %d, %d, %d)",
					r.Recopied, r.Appended, r.Kept, len(r.Orphans), tc.recopied, tc.appended, tc.kept, tc.orphans)
			}
			got := shardText(t, d)
			for _, want := range tc.wantIn {
				if !strings.Contains(got, want) {
					t.Fatalf("result does not carry %q:\n%s", want, got)
				}
			}
			for _, no := range tc.wantNotIn {
				if strings.Contains(got, no) {
					t.Fatalf("result still carries %q:\n%s", no, got)
				}
			}
		})
	}
}

func TestRefreshEmbedSourceCannotEscapeRetainedWorkspace(t *testing.T) {
	parent := t.TempDir()
	workspace := filepath.Join(parent, "workspace")
	design := filepath.Join(workspace, "child", "design")
	outside := filepath.Join(parent, "outside.md")
	if err := os.MkdirAll(design, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte(keyedSource), 0o644); err != nil {
		t.Fatal(err)
	}
	embedding := "<!-- machinery:embed from=\"../../../outside.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `alpha` | STALE |\n"
	if err := os.WriteFile(filepath.Join(design, "SHARD.md"), []byte(embedding), 0o644); err != nil {
		t.Fatal(err)
	}

	reports, changed, err := RefreshEmbeds(design, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || !strings.Contains(reports[0].Problem, "escapes the retained design workspace") {
		t.Fatalf("over-climbing refresh source was not rejected: %+v", reports)
	}
	if len(changed) != 0 || !strings.Contains(shardText(t, design), "STALE") {
		t.Fatalf("rejected outside source changed the design: changed=%v", changed)
	}
}

func TestRefreshEmbedsRejectsConcurrentSiblingSourceMutation(t *testing.T) {
	workspace := t.TempDir()
	source := filepath.Join(workspace, "parent", "ARCHITECTURE.md")
	design := filepath.Join(workspace, "child", "design")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(design, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte(keyedSource), 0o644); err != nil {
		t.Fatal(err)
	}
	mustCopyChildPackCapability(t, design)
	embedding := "<!-- machinery:embed from=\"../../parent/ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `alpha` | STALE |\n"
	if err := os.WriteFile(filepath.Join(design, "SHARD.md"), []byte(embedding), 0o644); err != nil {
		t.Fatal(err)
	}
	before := shardText(t, design)
	prior := embedRefreshAfterWorkspaceSnapshot
	embedRefreshAfterWorkspaceSnapshot = func() {
		if err := os.WriteFile(source, []byte(strings.Replace(keyedSource, "current", "mutated", 1)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { embedRefreshAfterWorkspaceSnapshot = prior }()

	if _, _, err := RefreshEmbeds(design, false); err == nil || !strings.Contains(err.Error(), "external tree changed") {
		t.Fatalf("concurrent sibling source mutation was not rejected: %v", err)
	}
	if got := shardText(t, design); got != before {
		t.Fatalf("refresh published from an unstable sibling source:\n%s", got)
	}
}

func TestRefreshIsIdempotent(t *testing.T) {
	d := refreshDesign(t, eventSource,
		"<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"event,producer,consumer,payload\" where=\"consumer=core\" claims=\"subset,complete\" -->\n"+
			"| event | producer | consumer | payload |\n|---|---|---|---|\n| `job.done` | ops | core (sse lane) | STALE |\n")
	if _, changed, err := RefreshEmbeds(d, false); err != nil || len(changed) != 1 {
		t.Fatalf("first run: changed %v, err %v", changed, err)
	}
	first := shardText(t, d)
	reports, changed, err := RefreshEmbeds(d, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 0 {
		t.Fatalf("second run changed %v; the refresh is not idempotent", changed)
	}
	if reports[0].Recopied != 0 || reports[0].Appended != 0 {
		t.Fatalf("second run reported work: %+v", reports[0])
	}
	if got := shardText(t, d); got != first {
		t.Fatalf("second run rewrote the document:\n%s\n---\n%s", first, got)
	}
}

func TestRefreshCrashAfterFirstDirectoryExactRetryClearsSentinel(t *testing.T) {
	design := multiDirectoryRefreshDesign(t)
	prior := embedTransactionPoint
	embedTransactionPoint = func(point string) error {
		if point == "directory:a" {
			panic("simulated process death after first directory")
		}
		return nil
	}
	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("simulated crash did not fire")
			}
		}()
		_, _, _ = RefreshEmbeds(design, false)
	}()
	embedTransactionPoint = prior
	defer func() { embedTransactionPoint = prior }()

	aBody, err := os.ReadFile(filepath.Join(design, "a", "SHARD.md"))
	if err != nil || strings.Contains(string(aBody), "STALE") {
		t.Fatalf("first directory did not commit before crash: %q, %v", aBody, err)
	}
	bBody, err := os.ReadFile(filepath.Join(design, "b", "SHARD.md"))
	if err != nil || !strings.Contains(string(bBody), "STALE") {
		t.Fatalf("second directory changed across crash: %q, %v", bBody, err)
	}
	if reader, err := designlock.AcquireReader(design); err == nil {
		_ = reader.Release()
		t.Fatal("reader accepted crash-partial embed publication")
	}
	if _, changed, err := RefreshEmbeds(design, false); err != nil {
		t.Fatalf("exact retry failed: %v", err)
	} else if len(changed) != 2 || changed[0] != "a/SHARD.md" || changed[1] != "b/SHARD.md" {
		t.Fatalf("retry changed = %v, want the immutable original two-file plan", changed)
	}
	reader, err := designlock.AcquireReader(design)
	if err != nil {
		t.Fatalf("reader remained stranded after exact retry: %v", err)
	}
	if err := reader.Release(); err != nil {
		t.Fatal(err)
	}
}

func multiDirectoryRefreshDesign(t *testing.T) string {
	t.Helper()
	design := t.TempDir()
	stale := "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n" +
		"| name | note |\n|---|---|\n| `alpha` | STALE |\n"
	for _, dir := range []string{"a", "b"} {
		base := filepath.Join(design, dir)
		if err := os.MkdirAll(base, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, "ARCHITECTURE.md"), []byte(keyedSource), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, "SHARD.md"), []byte(stale), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return design
}

func TestRefreshPowerLossAtEveryDurableBoundaryReplaysImmutablePlan(t *testing.T) {
	points := []string{
		"journal-stage-synced:prepared",
		"journal-stage-dir-synced:prepared",
		"journal-renamed:prepared",
		"journal-dir-synced:prepared",
		"prepared-journal",
		"stage:a/.machinery-embed-new-000000",
		"rename-linked:a/SHARD.md->a/.machinery-embed-old-000000",
		"rename-linked:a/.machinery-embed-new-000000->a/SHARD.md",
		"directory:a",
		"stage:b/.machinery-embed-new-000001",
		"directory:b",
		"journal-stage-synced:committed",
		"journal-stage-dir-synced:committed",
		"journal-renamed:committed",
		"journal-dir-synced:committed",
		"committed-journal",
		"finalized-before-publication-clear",
	}
	for _, crashPoint := range points {
		t.Run(strings.ReplaceAll(crashPoint, "/", "_"), func(t *testing.T) {
			design := multiDirectoryRefreshDesign(t)
			prior := embedTransactionPoint
			embedTransactionPoint = func(point string) error {
				if point == crashPoint {
					panic("power loss at " + point)
				}
				return nil
			}
			func() {
				defer func() {
					if recovered := recover(); recovered == nil {
						t.Fatalf("power-loss point %s did not fire", crashPoint)
					}
				}()
				_, _, _ = RefreshEmbeds(design, false)
			}()
			embedTransactionPoint = prior
			t.Cleanup(func() { embedTransactionPoint = prior })

			if reader, err := designlock.AcquireReader(design); err == nil {
				_ = reader.Release()
				t.Fatalf("reader crossed interrupted transaction at %s", crashPoint)
			}
			if _, changed, err := RefreshEmbeds(design, false); err != nil {
				t.Fatalf("exact retry after %s: %v", crashPoint, err)
			} else if crashPoint == "finalized-before-publication-clear" && len(changed) != 0 {
				t.Fatalf("finalized retry after %s reported already-published files as newly changed: %v", crashPoint, changed)
			} else if crashPoint != "finalized-before-publication-clear" && strings.Join(changed, ",") != "a/SHARD.md,b/SHARD.md" {
				t.Fatalf("retry after %s did not replay original plan: %v", crashPoint, changed)
			}
			for _, dir := range []string{"a", "b"} {
				body, err := os.ReadFile(filepath.Join(design, dir, "SHARD.md"))
				if err != nil || strings.Contains(string(body), "STALE") {
					t.Fatalf("%s not converged after %s: %q, %v", dir, crashPoint, body, err)
				}
			}
			assertNoEmbedTransactionResidue(t, design)
			if _, changed, err := RefreshEmbeds(design, false); err != nil || len(changed) != 0 {
				t.Fatalf("post-recovery retry not idempotent after %s: changed=%v err=%v", crashPoint, changed, err)
			}
		})
	}
}

func assertNoEmbedTransactionResidue(t *testing.T, design string) {
	t.Helper()
	err := filepath.WalkDir(design, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := entry.Name()
		if name == embedTxJournalName || name == embedTxStageName || strings.HasPrefix(name, embedTxNewPrefix) || strings.HasPrefix(name, embedTxOldPrefix) || name == ".machinery-design-publish.json" {
			return fmt.Errorf("transaction residue remains: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEmbedRefreshJournalCarriesExactBytesHashesModesAndRejectsForeignOrMalformed(t *testing.T) {
	t.Run("exact immutable payload", func(t *testing.T) {
		design := multiDirectoryRefreshDesign(t)
		prior := embedTransactionPoint
		embedTransactionPoint = func(point string) error {
			if point == "prepared-journal" {
				panic("inspect journal")
			}
			return nil
		}
		func() {
			defer func() { _ = recover() }()
			_, _, _ = RefreshEmbeds(design, false)
		}()
		embedTransactionPoint = prior
		t.Cleanup(func() { embedTransactionPoint = prior })
		body, err := os.ReadFile(filepath.Join(design, embedTxJournalName))
		if err != nil {
			t.Fatal(err)
		}
		journal, err := decodeEmbedTxJournal(body)
		if err != nil || len(journal.Items) != 2 {
			t.Fatalf("decode exact journal: %v, %#v", err, journal)
		}
		for _, item := range journal.Items {
			if !bytes.Contains(item.Old, []byte("STALE")) || bytes.Contains(item.New, []byte("STALE")) || item.OldHash != embedTxHash(item.Old) || item.NewHash != embedTxHash(item.New) || item.OldMode != 0o644 || item.NewMode != 0o644 || item.Deletion {
				t.Fatalf("journal omitted exact old/new identity: %#v", item)
			}
		}
	})

	t.Run("malformed", func(t *testing.T) {
		design := refreshDesign(t, keyedSource, "clean\n")
		if err := os.WriteFile(filepath.Join(design, embedTxJournalName), []byte(`{"version":1,"unknown":true}`+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := RefreshEmbeds(design, false); err == nil || !strings.Contains(err.Error(), "invalid embed refresh journal") {
			t.Fatalf("malformed journal was accepted: %v", err)
		}
	})

	t.Run("foreign scope", func(t *testing.T) {
		design := refreshDesign(t, keyedSource, "clean\n")
		item := embedTxItem{Path: "SHARD.md", Old: []byte("clean\n"), New: []byte("new\n"), OldMode: 0o644, NewMode: 0o644}
		journal, err := newEmbedTxJournal("foreign-design-scope", []embedTxItem{item})
		if err != nil {
			t.Fatal(err)
		}
		body, err := encodeEmbedTxJournal(journal)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(design, embedTxJournalName), body, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := RefreshEmbeds(design, false); err == nil || !strings.Contains(err.Error(), "foreign design scope") {
			t.Fatalf("foreign journal was accepted: %v", err)
		}
	})

	t.Run("symlink journal", func(t *testing.T) {
		design := refreshDesign(t, keyedSource, "clean\n")
		outside := filepath.Join(t.TempDir(), "journal")
		if err := os.WriteFile(outside, []byte("outside sentinel\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(design, embedTxJournalName)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, _, err := RefreshEmbeds(design, false); err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("symlink journal was accepted: %v", err)
		}
		body, err := os.ReadFile(outside)
		if err != nil || string(body) != "outside sentinel\n" {
			t.Fatalf("outside journal referent changed: %q, %v", body, err)
		}
	})

	t.Run("directory journal", func(t *testing.T) {
		design := refreshDesign(t, keyedSource, "clean\n")
		if err := os.Mkdir(filepath.Join(design, embedTxJournalName), 0o700); err != nil {
			t.Fatal(err)
		}
		if _, _, err := RefreshEmbeds(design, false); err == nil || !strings.Contains(err.Error(), "must be a regular file") {
			t.Fatalf("directory journal was accepted: %v", err)
		}
	})
}

func TestEmbedTransactionDiagnosticsAndRollbackAreDeterministicAndExact(t *testing.T) {
	t.Run("alias diagnostic", func(t *testing.T) {
		item := embedTxItem{Path: "A.md", Temp: "A.md", Backup: ".machinery-embed-old-000000", Old: []byte("old"), New: []byte("new"), OldMode: 0o644, NewMode: 0o644}
		item.OldHash, item.NewHash = embedTxHash(item.Old), embedTxHash(item.New)
		journal := embedTxJournal{Version: embedTxVersion, Operation: "embed-refresh", Phase: embedTxPrepared, Scope: "scope", Items: []embedTxItem{item}}
		journal.Checksum = embedTxChecksum(journal)
		const want = "journal item 0 temp A.md aliases target A.md"
		for i := 0; i < 100; i++ {
			if err := validateEmbedTxJournal(journal); err == nil || err.Error() != want {
				t.Fatalf("run %d: err=%v, want %q", i, err, want)
			}
		}
	})

	t.Run("physical residue diagnostic", func(t *testing.T) {
		design := t.TempDir()
		for _, dir := range []string{"a", "b"} {
			if err := os.Mkdir(filepath.Join(design, dir), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(design, dir, ".machinery-embed-new-999999"), []byte("foreign"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		tx, err := openEmbedRootTransaction(design)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Close()
		journal, err := newEmbedTxJournal(tx.scope, []embedTxItem{{Path: "target.md", Old: []byte("old"), New: []byte("new"), OldMode: 0o644, NewMode: 0o644}})
		if err != nil {
			t.Fatal(err)
		}
		const want = "unreferenced embed transaction residue a/.machinery-embed-new-999999"
		for i := 0; i < 100; i++ {
			if err := tx.validatePhysicalInventory(journal, false); err == nil || err.Error() != want {
				t.Fatalf("run %d: err=%v, want %q", i, err, want)
			}
		}
	})

	t.Run("chmod prevents rollback overwrite", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("POSIX permission mutation contract")
		}
		design := multiDirectoryRefreshDesign(t)
		target := filepath.Join(design, "a", "SHARD.md")
		prior := embedTransactionPoint
		embedTransactionPoint = func(point string) error {
			if point == "directory:a" {
				if err := os.Chmod(target, 0o600); err != nil {
					t.Fatal(err)
				}
				return fmt.Errorf("injected failure after concurrent chmod")
			}
			return nil
		}
		_, _, err := RefreshEmbeds(design, false)
		embedTransactionPoint = prior
		t.Cleanup(func() { embedTransactionPoint = prior })
		if err == nil || !strings.Contains(err.Error(), "durable rollback failed") {
			t.Fatalf("rollback silently overwrote mode mutation: %v", err)
		}
		info, statErr := os.Stat(target)
		if statErr != nil {
			t.Fatalf("stat concurrently modified target: %v", statErr)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("concurrent mode mutation was overwritten: mode=%v", info.Mode())
		}
	})
}

func replaceEmbedTestFile(t *testing.T, target string, body []byte, mode os.FileMode) {
	t.Helper()
	replacement := target + ".concurrent-replacement"
	if err := os.WriteFile(replacement, body, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(replacement, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, target); err != nil {
		t.Fatal(err)
	}
}

func TestEmbedRollbackPreservesLateTargetReplacementAndABA(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mutate  func([]byte) []byte
		message string
	}{
		{name: "content", mutate: func([]byte) []byte { return []byte("concurrent user edit\n") }, message: "content"},
		{name: "same-byte ABA", mutate: func(body []byte) []byte { return append([]byte(nil), body...) }, message: "identity"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			target := filepath.Join(design, "SHARD.md")
			if err := os.WriteFile(target, []byte("old\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			tx, err := openEmbedRootTransaction(design)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Close()
			journal, err := newEmbedTxJournal(tx.scope, []embedTxItem{{Path: "SHARD.md", Old: []byte("old\n"), New: []byte("new\n"), OldMode: 0o644, NewMode: 0o644}})
			if err != nil {
				t.Fatal(err)
			}
			prior := embedTransactionPoint
			embedTransactionPoint = func(point string) error {
				switch point {
				case "directory:.":
					return fmt.Errorf("injected failure")
				case "rollback-before-target-remove:SHARD.md":
					body, readErr := os.ReadFile(target)
					if readErr != nil {
						t.Fatal(readErr)
					}
					replaceEmbedTestFile(t, target, tc.mutate(body), 0o644)
				}
				return nil
			}
			err = tx.commit(journal)
			embedTransactionPoint = prior
			t.Cleanup(func() { embedTransactionPoint = prior })
			if err == nil || !strings.Contains(err.Error(), "durable rollback failed") || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("late %s replacement was not rejected exactly: %v", tc.name, err)
			}
			got, readErr := os.ReadFile(target)
			if readErr != nil || !bytes.Equal(got, tc.mutate([]byte("new\n"))) {
				t.Fatalf("late replacement was not preserved: %q, %v", got, readErr)
			}
			for _, residue := range []string{embedTxJournalName, journal.Items[0].Backup} {
				if _, statErr := os.Lstat(filepath.Join(design, filepath.FromSlash(residue))); statErr != nil {
					t.Fatalf("recovery evidence %s was removed: %v", residue, statErr)
				}
			}
		})
	}
}

func TestEmbedRollbackPreservesLateBackupReplacementAtRestore(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{name: "content", mutate: func([]byte) []byte { return []byte("concurrent original edit\n") }},
		{name: "same-byte ABA", mutate: func(body []byte) []byte { return append([]byte(nil), body...) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			target := filepath.Join(design, "SHARD.md")
			if err := os.WriteFile(target, []byte("old\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			tx, err := openEmbedRootTransaction(design)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Close()
			journal, err := newEmbedTxJournal(tx.scope, []embedTxItem{{Path: "SHARD.md", Old: []byte("old\n"), New: []byte("new\n"), OldMode: 0o644, NewMode: 0o644}})
			if err != nil {
				t.Fatal(err)
			}
			backup := filepath.Join(design, filepath.FromSlash(journal.Items[0].Backup))
			prior := embedTransactionPoint
			embedTransactionPoint = func(point string) error {
				switch point {
				case "directory:.":
					return fmt.Errorf("injected failure")
				case "rollback-before-backup-restore:SHARD.md":
					body, readErr := os.ReadFile(backup)
					if readErr != nil {
						t.Fatal(readErr)
					}
					replaceEmbedTestFile(t, backup, tc.mutate(body), 0o644)
				}
				return nil
			}
			err = tx.commit(journal)
			embedTransactionPoint = prior
			t.Cleanup(func() { embedTransactionPoint = prior })
			if err == nil || !strings.Contains(err.Error(), "durable rollback failed") || !strings.Contains(err.Error(), "changed content, identity") {
				t.Fatalf("late %s backup replacement was not rejected: %v", tc.name, err)
			}
			got, readErr := os.ReadFile(backup)
			if readErr != nil || !bytes.Equal(got, tc.mutate([]byte("old\n"))) {
				t.Fatalf("late backup replacement was not preserved: %q, %v", got, readErr)
			}
			if _, statErr := os.Lstat(filepath.Join(design, embedTxJournalName)); statErr != nil {
				t.Fatalf("journal was removed after ambiguous restore: %v", statErr)
			}
		})
	}
}

func TestEmbedRollbackDoesNotOverwriteLateTargetAtRestore(t *testing.T) {
	design := t.TempDir()
	target := filepath.Join(design, "SHARD.md")
	if err := os.WriteFile(target, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tx, err := openEmbedRootTransaction(design)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Close()
	journal, err := newEmbedTxJournal(tx.scope, []embedTxItem{{Path: "SHARD.md", Old: []byte("old\n"), New: []byte("new\n"), OldMode: 0o644, NewMode: 0o644}})
	if err != nil {
		t.Fatal(err)
	}
	prior := embedTransactionPoint
	embedTransactionPoint = func(point string) error {
		switch point {
		case "directory:.":
			return fmt.Errorf("injected failure")
		case "rollback-before-backup-restore:SHARD.md":
			if err := os.WriteFile(target, []byte("late target\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return nil
	}
	err = tx.commit(journal)
	embedTransactionPoint = prior
	t.Cleanup(func() { embedTransactionPoint = prior })
	if err == nil || !strings.Contains(err.Error(), "refuse to overwrite concurrent embed transaction path SHARD.md") {
		t.Fatalf("late restore target was overwritten: %v", err)
	}
	if got, readErr := os.ReadFile(target); readErr != nil || string(got) != "late target\n" {
		t.Fatalf("late target was not preserved: %q, %v", got, readErr)
	}
	if got, readErr := os.ReadFile(filepath.Join(design, filepath.FromSlash(journal.Items[0].Backup))); readErr != nil || string(got) != "old\n" {
		t.Fatalf("original backup was not preserved: %q, %v", got, readErr)
	}
}

func TestEmbedFinalizePreservesLateBackupReplacementAndABA(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mutate  func([]byte) []byte
		message string
	}{
		{name: "content", mutate: func([]byte) []byte { return []byte("concurrent backup edit\n") }, message: "content"},
		{name: "same-byte ABA", mutate: func(body []byte) []byte { return append([]byte(nil), body...) }, message: "identity"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			target := filepath.Join(design, "SHARD.md")
			if err := os.WriteFile(target, []byte("old\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			tx, err := openEmbedRootTransaction(design)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Close()
			journal, err := newEmbedTxJournal(tx.scope, []embedTxItem{{Path: "SHARD.md", Old: []byte("old\n"), New: []byte("new\n"), OldMode: 0o644, NewMode: 0o644}})
			if err != nil {
				t.Fatal(err)
			}
			backup := filepath.Join(design, filepath.FromSlash(journal.Items[0].Backup))
			prior := embedTransactionPoint
			embedTransactionPoint = func(point string) error {
				if point == "finalize-before-residue-remove:"+journal.Items[0].Backup {
					body, readErr := os.ReadFile(backup)
					if readErr != nil {
						t.Fatal(readErr)
					}
					replaceEmbedTestFile(t, backup, tc.mutate(body), 0o644)
				}
				return nil
			}
			err = tx.commit(journal)
			embedTransactionPoint = prior
			t.Cleanup(func() { embedTransactionPoint = prior })
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("late %s backup replacement was not rejected exactly: %v", tc.name, err)
			}
			got, readErr := os.ReadFile(backup)
			if readErr != nil || !bytes.Equal(got, tc.mutate([]byte("old\n"))) {
				t.Fatalf("late backup replacement was not preserved: %q, %v", got, readErr)
			}
			if _, statErr := os.Lstat(filepath.Join(design, embedTxJournalName)); statErr != nil {
				t.Fatalf("journal was removed after ambiguous cleanup: %v", statErr)
			}
		})
	}
}

func TestEmbedRollbackPreservesLateTempReplacementAndABA(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{name: "content", mutate: func([]byte) []byte { return []byte("concurrent temp edit\n") }},
		{name: "same-byte ABA", mutate: func(body []byte) []byte { return append([]byte(nil), body...) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			target := filepath.Join(design, "SHARD.md")
			if err := os.WriteFile(target, []byte("old\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			tx, err := openEmbedRootTransaction(design)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Close()
			journal, err := newEmbedTxJournal(tx.scope, []embedTxItem{{Path: "SHARD.md", Old: []byte("old\n"), New: []byte("new\n"), OldMode: 0o644, NewMode: 0o644}})
			if err != nil {
				t.Fatal(err)
			}
			temp := filepath.Join(design, filepath.FromSlash(journal.Items[0].Temp))
			prior := embedTransactionPoint
			embedTransactionPoint = func(point string) error {
				switch point {
				case "stage:" + journal.Items[0].Temp:
					return fmt.Errorf("injected failure")
				case "rollback-before-temp-remove:" + journal.Items[0].Temp:
					body, readErr := os.ReadFile(temp)
					if readErr != nil {
						t.Fatal(readErr)
					}
					replaceEmbedTestFile(t, temp, tc.mutate(body), 0o644)
				}
				return nil
			}
			err = tx.commit(journal)
			embedTransactionPoint = prior
			t.Cleanup(func() { embedTransactionPoint = prior })
			if err == nil || !strings.Contains(err.Error(), "durable rollback failed") || !strings.Contains(err.Error(), "changed content, identity") {
				t.Fatalf("late %s temp replacement was not rejected: %v", tc.name, err)
			}
			got, readErr := os.ReadFile(temp)
			if readErr != nil || !bytes.Equal(got, tc.mutate([]byte("new\n"))) {
				t.Fatalf("late temp replacement was not preserved: %q, %v", got, readErr)
			}
			if _, statErr := os.Lstat(filepath.Join(design, embedTxJournalName)); statErr != nil {
				t.Fatalf("journal was removed after ambiguous temp cleanup: %v", statErr)
			}
		})
	}
}

func TestEmbedFinalizePreservesLateTargetReplacementAndJournalABA(t *testing.T) {
	t.Run("target before residue removal", func(t *testing.T) {
		for _, sameBytes := range []bool{false, true} {
			name := "content"
			if sameBytes {
				name = "same-byte ABA"
			}
			t.Run(name, func(t *testing.T) {
				design := t.TempDir()
				target := filepath.Join(design, "SHARD.md")
				if err := os.WriteFile(target, []byte("old\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				tx, err := openEmbedRootTransaction(design)
				if err != nil {
					t.Fatal(err)
				}
				defer tx.Close()
				journal, err := newEmbedTxJournal(tx.scope, []embedTxItem{{Path: "SHARD.md", Old: []byte("old\n"), New: []byte("new\n"), OldMode: 0o644, NewMode: 0o644}})
				if err != nil {
					t.Fatal(err)
				}
				prior := embedTransactionPoint
				embedTransactionPoint = func(point string) error {
					if point == "finalize-before-residue-remove:"+journal.Items[0].Backup {
						body := []byte("concurrent target edit\n")
						if sameBytes {
							body = []byte("new\n")
						}
						replaceEmbedTestFile(t, target, body, 0o644)
					}
					return nil
				}
				err = tx.commit(journal)
				embedTransactionPoint = prior
				t.Cleanup(func() { embedTransactionPoint = prior })
				if err == nil || !strings.Contains(err.Error(), "target changed before residue removal") {
					t.Fatalf("late target replacement was not rejected: %v", err)
				}
				for _, residue := range []string{embedTxJournalName, journal.Items[0].Backup} {
					if _, statErr := os.Lstat(filepath.Join(design, filepath.FromSlash(residue))); statErr != nil {
						t.Fatalf("recovery evidence %s was removed: %v", residue, statErr)
					}
				}
			})
		}
	})

	t.Run("journal", func(t *testing.T) {
		design := t.TempDir()
		target := filepath.Join(design, "SHARD.md")
		if err := os.WriteFile(target, []byte("old\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		tx, err := openEmbedRootTransaction(design)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Close()
		journal, err := newEmbedTxJournal(tx.scope, []embedTxItem{{Path: "SHARD.md", Old: []byte("old\n"), New: []byte("new\n"), OldMode: 0o644, NewMode: 0o644}})
		if err != nil {
			t.Fatal(err)
		}
		journalPath := filepath.Join(design, embedTxJournalName)
		prior := embedTransactionPoint
		embedTransactionPoint = func(point string) error {
			if point == "finalize-before-journal-remove" {
				body, readErr := os.ReadFile(journalPath)
				if readErr != nil {
					t.Fatal(readErr)
				}
				replaceEmbedTestFile(t, journalPath, body, 0o600)
			}
			return nil
		}
		err = tx.commit(journal)
		embedTransactionPoint = prior
		t.Cleanup(func() { embedTransactionPoint = prior })
		if err == nil || !strings.Contains(err.Error(), "refuse to remove changed embed refresh journal") {
			t.Fatalf("late journal ABA was not rejected: %v", err)
		}
		if _, statErr := os.Lstat(journalPath); statErr != nil {
			t.Fatalf("replaced journal was not preserved: %v", statErr)
		}
	})
}

func TestEmbedRollbackPowerLossAfterRestoreLinkRecovers(t *testing.T) {
	design := multiDirectoryRefreshDesign(t)
	prior := embedTransactionPoint
	embedTransactionPoint = func(point string) error {
		if point == "directory:a" {
			return fmt.Errorf("force ordinary rollback")
		}
		if point == "rename-linked:a/.machinery-embed-old-000000->a/SHARD.md" {
			panic("power loss after no-clobber restore link")
		}
		return nil
	}
	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("restore-link power loss did not fire")
			}
		}()
		_, _, _ = RefreshEmbeds(design, false)
	}()
	embedTransactionPoint = prior
	t.Cleanup(func() { embedTransactionPoint = prior })

	if _, changed, err := RefreshEmbeds(design, false); err != nil {
		t.Fatalf("restart did not recover exact linked rollback state: %v", err)
	} else if strings.Join(changed, ",") != "a/SHARD.md,b/SHARD.md" {
		t.Fatalf("restart changed = %v", changed)
	}
	assertNoEmbedTransactionResidue(t, design)
}

func TestEmbedJournalPromotionRejectsLateStageAndJournalABA(t *testing.T) {
	t.Run("stage", func(t *testing.T) {
		design := t.TempDir()
		tx, err := openEmbedRootTransaction(design)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Close()
		journal, err := newEmbedTxJournal(tx.scope, []embedTxItem{{Path: "SHARD.md", Old: []byte("old\n"), New: []byte("new\n"), OldMode: 0o644, NewMode: 0o644}})
		if err != nil {
			t.Fatal(err)
		}
		stage := filepath.Join(design, embedTxStageName)
		prior := embedTransactionPoint
		embedTransactionPoint = func(point string) error {
			if point == "journal-before-promote:prepared" {
				body, readErr := os.ReadFile(stage)
				if readErr != nil {
					t.Fatal(readErr)
				}
				replaceEmbedTestFile(t, stage, body, 0o600)
			}
			return nil
		}
		err = tx.persistJournal(journal)
		embedTransactionPoint = prior
		t.Cleanup(func() { embedTransactionPoint = prior })
		if err == nil || !strings.Contains(err.Error(), "stage changed before promotion") {
			t.Fatalf("late staged-journal ABA was not rejected: %v", err)
		}
		if _, statErr := os.Lstat(stage); statErr != nil {
			t.Fatalf("replaced stage was not preserved: %v", statErr)
		}
	})

	t.Run("current journal", func(t *testing.T) {
		design := t.TempDir()
		tx, err := openEmbedRootTransaction(design)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Close()
		journal, err := newEmbedTxJournal(tx.scope, []embedTxItem{{Path: "SHARD.md", Old: []byte("old\n"), New: []byte("new\n"), OldMode: 0o644, NewMode: 0o644}})
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.persistJournal(journal); err != nil {
			t.Fatal(err)
		}
		current := filepath.Join(design, embedTxJournalName)
		journal.Phase = embedTxCommitted
		prior := embedTransactionPoint
		embedTransactionPoint = func(point string) error {
			if point == "journal-before-promote:committed" {
				body, readErr := os.ReadFile(current)
				if readErr != nil {
					t.Fatal(readErr)
				}
				replaceEmbedTestFile(t, current, body, 0o600)
			}
			return nil
		}
		err = tx.persistJournal(journal)
		embedTransactionPoint = prior
		t.Cleanup(func() { embedTransactionPoint = prior })
		if err == nil || !strings.Contains(err.Error(), "journal changed before promotion") {
			t.Fatalf("late current-journal ABA was not rejected: %v", err)
		}
		for _, name := range []string{embedTxJournalName, embedTxStageName} {
			if _, statErr := os.Lstat(filepath.Join(design, name)); statErr != nil {
				t.Fatalf("%s was not preserved: %v", name, statErr)
			}
		}
	})
}

func TestEmbedJournalInstallPreservesAtomicDestinationCollisionAndRecovers(t *testing.T) {
	design := t.TempDir()
	target := filepath.Join(design, "SHARD.md")
	if err := os.WriteFile(target, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tx, err := openEmbedRootTransaction(design)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Close()
	journal, err := newEmbedTxJournal(tx.scope, []embedTxItem{{Path: "SHARD.md", Old: []byte("old\n"), New: []byte("new\n"), OldMode: 0o644, NewMode: 0o644}})
	if err != nil {
		t.Fatal(err)
	}
	collision := []byte("concurrent journal authority\n")
	prior := embedRenameNoReplace
	embedRenameNoReplace = func(root *os.Root, from, to string) error {
		if from == embedTxStageName && to == embedTxJournalName {
			file, openErr := root.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
			if openErr != nil {
				return openErr
			}
			_, writeErr := file.Write(collision)
			if err := errors.Join(writeErr, file.Close()); err != nil {
				return err
			}
		}
		return prior(root, from, to)
	}
	err = tx.persistJournal(journal)
	embedRenameNoReplace = prior
	t.Cleanup(func() { embedRenameNoReplace = prior })
	if err == nil || !strings.Contains(err.Error(), "install embed refresh journal") {
		t.Fatalf("destination collision was overwritten or accepted: %v", err)
	}
	journalBody, readErr := os.ReadFile(filepath.Join(design, embedTxJournalName))
	if readErr != nil || !bytes.Equal(journalBody, collision) {
		t.Fatalf("concurrent journal = %q, %v", journalBody, readErr)
	}
	if _, statErr := os.Lstat(filepath.Join(design, embedTxStageName)); statErr != nil {
		t.Fatalf("staged recovery authority was not preserved: %v", statErr)
	}
	if err := os.Remove(filepath.Join(design, embedTxJournalName)); err != nil {
		t.Fatal(err)
	}
	pending, ok, err := tx.pending()
	if err != nil || !ok {
		t.Fatalf("pending staged journal = ok %v, err %v", ok, err)
	}
	if err := tx.recover(pending); err != nil {
		t.Fatalf("retry did not recover preserved stage: %v", err)
	}
	body, err := os.ReadFile(target)
	if err != nil || string(body) != "new\n" {
		t.Fatalf("recovered target = %q, %v", body, err)
	}
}

func TestEmbedCommittedStageRecoversAfterOlderJournalRemovalCrash(t *testing.T) {
	design := refreshDesign(t, keyedSource,
		"<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `alpha` | STALE |\n")
	prior := embedTransactionPoint
	embedTransactionPoint = func(point string) error {
		if point == "finalize-journal-removed-before-stage" {
			panic("power loss after older journal removal")
		}
		return nil
	}
	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("journal-removal crash boundary did not fire")
			}
		}()
		_, _, _ = RefreshEmbeds(design, false)
	}()
	embedTransactionPoint = prior
	t.Cleanup(func() { embedTransactionPoint = prior })
	if _, err := os.Lstat(filepath.Join(design, embedTxJournalName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("older journal remained after crash boundary: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(design, embedTxStageName)); err != nil {
		t.Fatalf("committed recovery stage missing: %v", err)
	}
	if _, _, err := RefreshEmbeds(design, false); err != nil {
		t.Fatalf("restart did not recover committed stage: %v", err)
	}
	assertNoEmbedTransactionResidue(t, design)
}

func TestEmbedRemoveExactPreservesLateSameByteReplacementAndQuarantine(t *testing.T) {
	design := t.TempDir()
	path := filepath.Join(design, "victim.txt")
	if err := os.WriteFile(path, []byte("owned\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tx, err := openEmbedRootTransaction(design)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Close()
	state, err := tx.readState("victim.txt")
	if err != nil {
		t.Fatal(err)
	}
	prior := embedTransactionPoint
	embedTransactionPoint = func(point string) error {
		if point == "remove-quarantined:victim.txt" {
			return os.WriteFile(path, []byte("owned\n"), 0o600)
		}
		return nil
	}
	err = tx.removeExact("victim.txt", state, "", "test victim")
	embedTransactionPoint = prior
	t.Cleanup(func() { embedTransactionPoint = prior })
	if err == nil || !strings.Contains(err.Error(), "was repopulated") {
		t.Fatalf("late same-byte replacement was removed or accepted: %v", err)
	}
	body, readErr := os.ReadFile(path)
	if readErr != nil || string(body) != "owned\n" {
		t.Fatalf("public replacement = %q, %v", body, readErr)
	}
	residue, err := tx.embedResiduePaths()
	if err != nil {
		t.Fatal(err)
	}
	quarantines := 0
	for _, rel := range residue {
		if embedDeleteQuarantineRel(rel) {
			quarantines++
		}
	}
	if quarantines != 1 {
		t.Fatalf("private deletion quarantines = %v, want exactly one", residue)
	}
}

func TestEmbedRemoveExactRestoresABAReplacementMovedAtQuarantineBoundary(t *testing.T) {
	design := t.TempDir()
	path := filepath.Join(design, "victim.txt")
	if err := os.WriteFile(path, []byte("owned\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tx, err := openEmbedRootTransaction(design)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Close()
	state, err := tx.readState("victim.txt")
	if err != nil {
		t.Fatal(err)
	}
	prior := embedTransactionPoint
	embedTransactionPoint = func(point string) error {
		if point == "remove-before-quarantine:victim.txt" {
			replaceEmbedTestFile(t, path, []byte("owned\n"), 0o600)
		}
		return nil
	}
	err = tx.removeExact("victim.txt", state, "", "test victim")
	embedTransactionPoint = prior
	t.Cleanup(func() { embedTransactionPoint = prior })
	if err == nil || !strings.Contains(err.Error(), "differs from its exact retained witness") {
		t.Fatalf("same-byte ABA replacement was removed or accepted: %v", err)
	}
	body, readErr := os.ReadFile(path)
	if readErr != nil || string(body) != "owned\n" {
		t.Fatalf("ABA replacement was not restored to its public name: %q, %v", body, readErr)
	}
	residue, err := tx.embedResiduePaths()
	if err != nil {
		t.Fatal(err)
	}
	if len(residue) != 0 {
		t.Fatalf("restored ABA left deletion residue: %v", residue)
	}
}

func TestEmbedDeletionQuarantineCrashResidueRecovers(t *testing.T) {
	for _, crashPoint := range []string{
		"remove-quarantined:.machinery-embed-old-000000",
		"remove-quarantined:" + embedTxJournalName,
		"remove-quarantined:" + embedTxStageName,
	} {
		t.Run(crashPoint, func(t *testing.T) {
			design := refreshDesign(t, keyedSource,
				"<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `alpha` | STALE |\n")
			prior := embedTransactionPoint
			embedTransactionPoint = func(point string) error {
				if point == crashPoint {
					panic("power loss with retained deletion quarantine")
				}
				return nil
			}
			func() {
				defer func() {
					if recovered := recover(); recovered == nil {
						t.Fatalf("crash point %s did not fire", crashPoint)
					}
				}()
				_, _, _ = RefreshEmbeds(design, false)
			}()
			embedTransactionPoint = prior
			t.Cleanup(func() { embedTransactionPoint = prior })
			if _, _, err := RefreshEmbeds(design, false); err != nil {
				t.Fatalf("restart did not resume deletion quarantine: %v", err)
			}
			assertNoEmbedTransactionResidue(t, design)
		})
	}
}

func TestEmbedRefreshRootedTransactionCannotEscapeSwappedParent(t *testing.T) {
	design := multiDirectoryRefreshDesign(t)
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "SHARD.md")
	if err := os.WriteFile(sentinel, []byte("outside sentinel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prior := embedTransactionPoint
	embedTransactionPoint = func(point string) error {
		if point != "stage:a/.machinery-embed-new-000000" {
			return nil
		}
		if err := os.Rename(filepath.Join(design, "a"), filepath.Join(design, "a-parked")); err != nil {
			return err
		}
		return os.Symlink(outside, filepath.Join(design, "a"))
	}
	_, _, err := RefreshEmbeds(design, false)
	embedTransactionPoint = prior
	t.Cleanup(func() { embedTransactionPoint = prior })
	if err == nil {
		t.Fatal("parent swap was not rejected")
	}
	body, readErr := os.ReadFile(sentinel)
	if readErr != nil || string(body) != "outside sentinel\n" {
		t.Fatalf("rooted transaction escaped to outside sentinel: %q, %v", body, readErr)
	}
}

func TestRefreshDryRunWritesNothing(t *testing.T) {
	d := refreshDesign(t, keyedSource,
		"<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `alpha` | STALE |\n")
	before := shardText(t, d)
	reports, changed, err := RefreshEmbeds(d, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 1 {
		t.Fatalf("dry run reported %d changed files, want 1", len(changed))
	}
	if reports[0].Recopied != 1 {
		t.Fatalf("dry run reported %d re-copied rows, want 1", reports[0].Recopied)
	}
	if got := shardText(t, d); got != before {
		t.Fatalf("dry run wrote the file")
	}
}

func TestRefreshLeavesProseAndOtherTablesAlone(t *testing.T) {
	embedding := "# shard\n\nSome prose.\n\n| unrelated | table |\n|---|---|\n| a | b |\n\n" +
		"<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n\n" +
		"| name | note |\n|---|---|\n| `alpha` | STALE |\n\nTrailing prose.\n"
	d := refreshDesign(t, keyedSource, embedding)
	if _, _, err := RefreshEmbeds(d, false); err != nil {
		t.Fatal(err)
	}
	got := shardText(t, d)
	for _, want := range []string{"Some prose.", "| unrelated | table |", "| a | b |", "Trailing prose.", "| `alpha` | the alpha note, current |"} {
		if !strings.Contains(got, want) {
			t.Fatalf("result lost %q:\n%s", want, got)
		}
	}
}

func TestEmbedRowKey(t *testing.T) {
	cases := []struct {
		name  string
		cells []string
		cols  []string
		want  string
	}{
		{"backticked first cell", []string{"`alpha`", "x"}, []string{"name", "note"}, "alpha"},
		{"bare first cell", []string{"alpha thing", "x"}, []string{"name", "note"}, "alpha"},
		{"event table keys on three cells", []string{"`e`", "ops", "core (sse)", "p"}, []string{"event", "producer", "consumer", "payload"},
			"e\x00ops\x00core\x00core (sse)"},
		{"empty row", nil, []string{"name"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := embedRowKey(tc.cells, tc.cols); got != tc.want {
				t.Fatalf("embedRowKey = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLeadToken(t *testing.T) {
	cases := []struct{ in, want string }{
		{"`alpha`", "alpha"},
		{"  `alpha.beta` (note)", "alpha.beta"},
		{"plain-token rest", "plain-token"},
		{"(only a parenthetical)", "(only a parenthetical)"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := leadToken(tc.in); got != tc.want {
			t.Fatalf("leadToken(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsSeparatorRow(t *testing.T) {
	for _, in := range []string{"|---|---|", "| --- | :---: |"} {
		if !isSeparatorRow(in) {
			t.Fatalf("%q is a separator row", in)
		}
	}
	for _, in := range []string{"| a | b |", "not a table"} {
		if isSeparatorRow(in) {
			t.Fatalf("%q is not a separator row", in)
		}
	}
}

func TestFilterRowIdxMatchesFilterRows(t *testing.T) {
	tbls := ir.ParseMdTables(eventSource)
	if len(tbls) != 1 {
		t.Fatalf("fixture parsed to %d tables", len(tbls))
	}
	tbl := tbls[0]
	idx, err := filterRowIdx(tbl, "consumer=core")
	if err != "" {
		t.Fatal(err)
	}
	rows, err := filterRows(tbl, "consumer=core")
	if err != "" {
		t.Fatal(err)
	}
	if len(idx) != len(rows) {
		t.Fatalf("index count %d != row count %d", len(idx), len(rows))
	}
	for i := range idx {
		if tbl.Rows[idx[i]][0] != rows[i][0] {
			t.Fatalf("index %d does not address row %d", idx[i], i)
		}
	}
}

const collidingSource = "# root\n\n" +
	"| failure | detection | recovery |\n|---|---|---|\n" +
	"| duplicate `request` redelivery | dedupe by orderId | drop |\n" +
	"| duplicate `capture` redelivery | dedupe by id | drop |\n"

func TestRefreshRefusesAmbiguousKeys(t *testing.T) {
	// Both source rows lead with the bare token "duplicate", so the key does
	// not separate them. A drifted copy of one must NOT be rewritten with the
	// other's text; it is reported instead.
	d := refreshDesign(t, collidingSource,
		"<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"failure,detection,recovery\" claims=\"subset\" -->\n"+
			"| failure | detection | recovery |\n|---|---|---|\n| duplicate `capture` redelivery | dedupe by id | STALE |\n")
	reports, changed, err := RefreshEmbeds(d, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 0 {
		t.Fatalf("an ambiguous key rewrote the document: %v", changed)
	}
	if len(reports[0].Ambiguous) != 1 {
		t.Fatalf("ambiguity not reported: %+v", reports[0])
	}
	if got := shardText(t, d); !strings.Contains(got, "STALE") {
		t.Fatalf("the row was rewritten from a colliding source row:\n%s", got)
	}
}

func TestRefreshLeavesADuplicatedCopyAlone(t *testing.T) {
	// A copy that carries the same source row twice is a de-duplication
	// judgment, not a re-copy: the second occurrence is already a byte copy
	// and must not be rewritten with the next unclaimed source row.
	d := refreshDesign(t, keyedSource,
		"<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n"+
			"| name | note |\n|---|---|\n| `alpha` | the alpha note, current |\n| `alpha` | the alpha note, current |\n")
	reports, changed, err := RefreshEmbeds(d, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 0 || reports[0].Recopied != 0 {
		t.Fatalf("the duplicated row was rewritten: changed %v, report %+v", changed, reports[0])
	}
	if got := shardText(t, d); strings.Count(got, "the alpha note, current") != 2 {
		t.Fatalf("the duplicate did not survive untouched:\n%s", got)
	}
}

func TestRefreshRefusesAnUnresolvableSelector(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		wantSub string
	}{
		{
			name:    "no source table has the columns",
			source:  "# root\n\n| other | columns |\n|---|---|\n| a | b |\n",
			wantSub: "resolves to 0 source tables",
		},
		{
			name:    "two source tables have them",
			source:  keyedSource + "\nand again\n\n| name | note |\n|---|---|\n| `alpha` | a second table |\n",
			wantSub: "resolves to 2 source tables",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := refreshDesign(t, tc.source,
				"<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n"+
					"| name | note |\n|---|---|\n| `alpha` | STALE |\n")
			reports, changed, err := RefreshEmbeds(d, false)
			if err != nil {
				t.Fatal(err)
			}
			if len(changed) != 0 {
				t.Fatalf("an unresolvable selector rewrote the document: %v", changed)
			}
			if !strings.Contains(reports[0].Problem, tc.wantSub) {
				t.Fatalf("problem = %q, want it to name %q", reports[0].Problem, tc.wantSub)
			}
		})
	}
}

func TestSortReportsIsDeterministic(t *testing.T) {
	rs := []RefreshReport{
		{File: "b.md", From: "y.md"},
		{File: "a.md", From: "z.md"},
		{File: "a.md", From: "a.md"},
	}
	SortReports(rs)
	want := []string{"a.md:a.md", "a.md:z.md", "b.md:y.md"}
	for i, r := range rs {
		if got := r.File + ":" + r.From; got != want[i] {
			t.Fatalf("report %d = %q, want %q", i, got, want[i])
		}
	}
}
