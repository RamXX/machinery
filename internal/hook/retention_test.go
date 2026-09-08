package hook

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/dirscan"
)

// isolateHookRetention gives one test its own store and its own copy of the
// per-process compaction flags. Those flags exist so a single hook process
// repairs at most once; a test binary runs many events in one process and
// must start each case from the same state a fresh hook process would.
func isolateHookRetention(t *testing.T) string {
	t.Helper()
	isolateHookState(t)
	hookStateCompactionMu.Lock()
	priorAuto, priorBound := hookStateAutoCompacted, hookStateBoundExhausted
	hookStateAutoCompacted, hookStateBoundExhausted = false, false
	hookStateCompactionMu.Unlock()
	t.Cleanup(func() {
		hookStateCompactionMu.Lock()
		hookStateAutoCompacted, hookStateBoundExhausted = priorAuto, priorBound
		hookStateCompactionMu.Unlock()
	})
	if err := ensureStateDir(); err != nil {
		t.Fatal(err)
	}
	dir, err := stateDirPathExact()
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func shellEvent(sessionID, command string) Input {
	return Input{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
		ToolInput:     toolInput{Command: command},
		ToolUseID:     "operation-" + sessionID,
	}
}

func hookStoreEntries(t *testing.T, dir string) int {
	t.Helper()
	count, err := countHookStateEntries(dir)
	if err != nil {
		t.Fatalf("count hook state entries: %v", err)
	}
	return count
}

// writeDeadGenerations fabricates ledgers whose recorded project root does not
// exist. This is exactly what a heavy local sweep leaves behind: an armed
// obligation for a temporary project root that was deleted before any Stop
// event could discharge it.
func writeDeadGenerations(t *testing.T, dir, deadRoot string, count int) {
	t.Helper()
	body := []byte("revision 1\n" + hookStateRootLine(deadRoot) + "\ndesign\n")
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("%064x.state", i)
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHookStateLedgerBindsItsCanonicalProjectRoot(t *testing.T) {
	isolateHookRetention(t)
	root := managedRoot(t)
	if out := runEvent(t, root, shellEvent("bind", "echo governed")); out != "" {
		t.Fatalf("governed shell event was not allowed: %s", out)
	}
	canonical, err := canonicalHookRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(statePath(root, "bind"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(lines) < 2 || lines[1] != hookStateRootLine(canonical) {
		t.Fatalf("ledger does not bind its project root on line two: %q", string(raw))
	}
	record, err := parseHookStateRecord(raw)
	if err != nil || record.root != canonical {
		t.Fatalf("parsed ledger root = %q err=%v, want %q", record.root, err, canonical)
	}
}

func TestHookStateLedgerRejectsEveryNoncanonicalProjectRootLine(t *testing.T) {
	valid := hookStateRootLine("/tmp/project")
	for name, raw := range map[string]string{
		"two root lines":       "revision 1\n" + valid + "\n" + valid + "\ndesign\n",
		"root after design":    "revision 1\ndesign\n" + valid + "\n",
		"root after route":     "revision 1\nroute " + strings.Repeat("a", 64) + "\n" + valid + "\ndesign\n",
		"not hex":              "revision 1\nroot zz\ndesign\n",
		"uppercase hex":        "revision 1\nroot " + strings.ToUpper(hex.EncodeToString([]byte("/tmp/project"))) + "\ndesign\n",
		"empty root":           "revision 1\nroot \ndesign\n",
		"relative path":        "revision 1\n" + hookStateRootLine("project") + "\ndesign\n",
		"uncleaned path":       "revision 1\n" + hookStateRootLine("/tmp/project/../project") + "\ndesign\n",
		"root without a class": "revision 1\n" + valid + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseHookStateRecord([]byte(raw)); err == nil || !strings.Contains(err.Error(), "corrupt or noncanonical") {
				t.Fatalf("noncanonical ledger %q was accepted: %v", raw, err)
			}
		})
	}
}

func TestRouteSnapshotsAreBoundedPerProjectRoot(t *testing.T) {
	isolateHookRetention(t)
	root := managedRoot(t)
	const sessions = hookStateRouteRetention + 12
	last := ""
	for i := 0; i < sessions; i++ {
		session := fmt.Sprintf("session-%03d", i)
		if out := runEvent(t, root, shellEvent(session, "echo governed")); out != "" {
			t.Fatalf("governed shell event was not allowed: %s", out)
		}
		last = session
		routes, err := routeStatePaths(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(routes) > hookStateRouteRetention {
			t.Fatalf("after %d sessions the project kept %d route snapshots, above the %d bound", i+1, len(routes), hookStateRouteRetention)
		}
	}
	if _, err := os.Lstat(routeStatePath(root, last)); err != nil {
		t.Fatalf("retention reclaimed the snapshot it had just written: %v", err)
	}
	record, err := readStateRecord(root, last)
	if err != nil {
		t.Fatal(err)
	}
	if !record.design || len(record.routes) != 1 {
		t.Fatalf("route retention changed the project obligation: %+v", record)
	}
}

// TestHookStateStoreStaysBoundedAcrossThousandsOfInvocations is the sweep this
// bug was filed for: thousands of governed events against project roots that
// are deleted immediately afterwards, exactly like a local test sweep. It runs
// them one at a time, the way a host runs one hook process per event.
func TestHookStateStoreStaysBoundedAcrossThousandsOfInvocations(t *testing.T) {
	dir := isolateHookRetention(t)
	base := t.TempDir()
	const invocations = 2000
	peak := 0
	for i := 0; i < invocations; i++ {
		name := fmt.Sprintf("sweep-%04d", i)
		root := filepath.Join(base, name)
		if err := os.MkdirAll(filepath.Join(root, "design"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "design", "domain.modelith.yaml"), []byte("model: {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if out := runEvent(t, root, shellEvent(name, "echo governed")); out != "" {
			t.Fatalf("governed event %s was not allowed: %s", name, out)
		}
		if err := os.RemoveAll(root); err != nil {
			t.Fatal(err)
		}
		if i%25 != 0 && i != invocations-1 {
			continue
		}
		count := hookStoreEntries(t, dir)
		if count > peak {
			peak = count
		}
		if count >= hookStateDirMaxEntries {
			t.Fatalf("after %d invocations the store held %d entries, at the %d-entry fail-closed limit", i+1, count, hookStateDirMaxEntries)
		}
	}
	if peak > hookStateDirRetentionCeiling {
		t.Fatalf("store peaked at %d entries, above the %d-entry retention ceiling", peak, hookStateDirRetentionCeiling)
	}
	if final := hookStoreEntries(t, dir); final > hookStateDirRetentionCeiling {
		t.Fatalf("store finished at %d entries, above the %d-entry retention ceiling", final, hookStateDirRetentionCeiling)
	}
}

// TestStoreAtTheEntryLimitFailsClosedAndThenSelfRepairs pins both halves of
// the incident: the old behavior on a full store, and the recovery that
// replaces the manual prune it used to require.
func TestStoreAtTheEntryLimitFailsClosedAndThenSelfRepairs(t *testing.T) {
	dir := isolateHookRetention(t)
	root := managedRoot(t)
	dead := filepath.Join(t.TempDir(), "root-that-no-longer-exists")
	writeDeadGenerations(t, dir, dead, hookStateDirMaxEntries+8)

	// Negative: the bounded inventory every governance path takes is exactly
	// the read that fails closed on a store this size.
	if _, err := dirscan.Read(dir, hookStateDirMaxEntries); !errors.Is(err, dirscan.ErrTooManyEntries) {
		t.Fatalf("a store above its entry limit did not fail closed: %v", err)
	}

	// Negative: with this process's one repair attempt already spent, which is
	// the state every hook process was permanently in before this fix, the
	// store stays closed and says how to open it.
	if !beginHookStateAutoCompaction() {
		t.Fatal("expected an unspent repair attempt")
	}
	_, err := routeStatePaths(root)
	if err == nil || !errors.Is(err, dirscan.ErrTooManyEntries) {
		t.Fatalf("hook inventory on a full store = %v, want the entry limit", err)
	}
	for _, want := range []string{dir, "machinery doctor --repair"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("fail-closed diagnostic does not name %q: %v", want, err)
		}
	}
	out := runEvent(t, root, shellEvent("closed", "echo governed"))
	if !strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Fatalf("a governed event on an unrepairable full store was not denied: %q", out)
	}

	// Positive: a fresh hook process repairs the store it owns and the same
	// governed event is allowed again.
	hookStateCompactionMu.Lock()
	hookStateAutoCompacted = false
	hookStateCompactionMu.Unlock()
	if out := runEvent(t, root, shellEvent("recovered", "echo governed")); out != "" {
		t.Fatalf("governed event after self-repair was not allowed: %s", out)
	}
	count := hookStoreEntries(t, dir)
	if count > hookStateDirRetentionCeiling {
		t.Fatalf("self-repair left %d entries, above the %d-entry retention ceiling", count, hookStateDirRetentionCeiling)
	}
	if _, err := os.Lstat(statePath(root, "recovered")); err != nil {
		t.Fatalf("self-repair discarded the live project obligation it had just armed: %v", err)
	}
}

func TestCompactionNeverReclaimsLiveCorruptOrCrashEvidenceGenerations(t *testing.T) {
	dir := isolateHookRetention(t)
	live := managedRoot(t)
	if out := runEvent(t, live, shellEvent("live", "echo governed")); out != "" {
		t.Fatalf("governed shell event was not allowed: %s", out)
	}
	dead := filepath.Join(t.TempDir(), "root-that-no-longer-exists")
	reclaimable := filepath.Join(dir, fmt.Sprintf("%064x.state", 1))
	if err := os.WriteFile(reclaimable, []byte("revision 1\n"+hookStateRootLine(dead)+"\ndesign\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(dir, fmt.Sprintf("%064x.state", 2))
	if err := os.WriteFile(legacy, []byte("revision 1\ndesign\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	corrupt := filepath.Join(dir, fmt.Sprintf("%064x.state", 3))
	if err := os.WriteFile(corrupt, []byte("revision 1\n"+hookStateRootLine(dead)+"\nunknown\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	crashed := filepath.Join(dir, fmt.Sprintf("%064x.state", 4))
	if err := os.WriteFile(crashed, []byte("revision 1\n"+hookStateRootLine(dead)+"\ndesign\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	crashedTemp := filepath.Join(dir, "."+filepath.Base(crashed)+".tmp-crash")
	if err := os.WriteFile(crashedTemp, []byte("revision 2\n"+hookStateRootLine(dead)+"\ndesign\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	compaction, err := compactHookStateDir(dir, 0)
	if err != nil {
		t.Fatalf("compaction: %v", err)
	}
	if compaction.Reclaimed != 1 {
		t.Fatalf("compaction reclaimed %d entries, want exactly the one dead generation", compaction.Reclaimed)
	}
	if compaction.FirstRetainedErr == nil || !strings.Contains(compaction.FirstRetainedErr.Error(), "corrupt") {
		t.Fatalf("compaction did not report the corrupt ledger it kept: %v", compaction.FirstRetainedErr)
	}
	if _, err := os.Lstat(reclaimable); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dead generation survived compaction: %v", err)
	}
	for name, path := range map[string]string{
		"legacy ledger without a recorded root": legacy,
		"corrupt ledger":                        corrupt,
		"ledger with durable crash evidence":    crashed,
		"crash temp":                            crashedTemp,
		"live project ledger":                   statePath(live, "live"),
		"live project route snapshot":           routeStatePath(live, "live"),
	} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("compaction reclaimed the %s: %v", name, err)
		}
	}
	record, err := readStateRecord(live, "live")
	if err != nil || !record.design {
		t.Fatalf("compaction disturbed the live obligation: %+v err=%v", record, err)
	}
}

func TestStateReportStatesTheStoreAndRepairsItAboveTheLimit(t *testing.T) {
	dir := isolateHookRetention(t)
	var quiet strings.Builder
	ok, err := StateReport(&quiet, false)
	if err != nil || !ok || !strings.Contains(quiet.String(), dir) {
		t.Fatalf("empty-store report ok=%v err=%v out=%q", ok, err, quiet.String())
	}
	dead := filepath.Join(t.TempDir(), "root-that-no-longer-exists")
	writeDeadGenerations(t, dir, dead, hookStateDirMaxEntries+8)
	before := hookStoreEntries(t, dir)

	var full strings.Builder
	ok, err = StateReport(&full, false)
	if err != nil {
		t.Fatal(err)
	}
	if ok || !strings.Contains(full.String(), "at or above its 4096-entry limit") {
		t.Fatalf("report on a full store ok=%v out=%q", ok, full.String())
	}
	if strings.Contains(full.String(), "compacted: reclaimed") {
		t.Fatalf("a report without repair mutated the store: %q", full.String())
	}
	if count := hookStoreEntries(t, dir); count != before {
		t.Fatalf("a report without repair changed the store: %d entries, was %d", count, before)
	}

	var repaired strings.Builder
	ok, err = StateReport(&repaired, true)
	if err != nil || !ok {
		t.Fatalf("repair report ok=%v err=%v out=%q", ok, err, repaired.String())
	}
	if !strings.Contains(repaired.String(), "compacted: reclaimed") {
		t.Fatalf("repair report does not state what it reclaimed: %q", repaired.String())
	}
	if count := hookStoreEntries(t, dir); count > hookStateDirRetentionTarget {
		t.Fatalf("repair left %d entries, above the %d-entry target", count, hookStateDirRetentionTarget)
	}
}

// TestCompactionExcludesConcurrentEnumerationInAnotherProcess is the reason
// reclamation holds a store-wide lock in bounded batches. One store is shared
// by every repository and every agent on the machine: a compaction that
// removed files while a governed event was enumerating the same directory
// would exhaust that event's bounded retries and deny an unrelated tool call.
func TestCompactionExcludesConcurrentEnumerationInAnotherProcess(t *testing.T) {
	if os.Getenv("MACHINERY_HOOK_COMPACTION_CHILD") == "1" {
		dir := os.Getenv("MACHINERY_HOOK_COMPACTION_DIR")
		compaction, err := compactHookStateDir(dir, 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, "child compaction:", err)
			os.Exit(3)
		}
		if compaction.Reclaimed == 0 {
			fmt.Fprintln(os.Stderr, "child compaction reclaimed nothing")
			os.Exit(4)
		}
		os.Exit(0)
	}
	dir := isolateHookRetention(t)
	dead := filepath.Join(t.TempDir(), "root-that-no-longer-exists")
	const generations = 1200
	writeDeadGenerations(t, dir, dead, generations)

	command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestCompactionExcludesConcurrentEnumerationInAnotherProcess$")
	command.Env = append(os.Environ(),
		"MACHINERY_HOOK_COMPACTION_CHILD=1",
		"MACHINERY_HOOK_COMPACTION_DIR="+dir,
	)
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() { finished <- command.Wait() }()
	enumerations := 0
	for running := true; running; {
		select {
		case err := <-finished:
			if err != nil {
				t.Fatalf("compaction child: %v", err)
			}
			running = false
		default:
			if _, err := readHookStateDir(dir); err != nil {
				t.Fatalf("a governed enumeration failed while another process compacted the store: %v", err)
			}
			enumerations++
			time.Sleep(time.Millisecond)
		}
	}
	if enumerations < 5 {
		t.Fatalf("the parent enumerated the store only %d time(s) while the child compacted it", enumerations)
	}
	remaining, err := countHookStateEntries(dir)
	if err != nil {
		t.Fatal(err)
	}
	if remaining > 4 {
		t.Fatalf("the child process left %d of %d fabricated entries behind", remaining, generations)
	}
}

// TestCompactionNeverReclaimsAGenerationThisProcessHasLocked pins the
// reentrancy rule: a generation this process is itself arming is retained, and
// the same generation becomes reclaimable once that lock is gone.
func TestCompactionNeverReclaimsAGenerationThisProcessHasLocked(t *testing.T) {
	dir := isolateHookRetention(t)
	dead := filepath.Join(t.TempDir(), "root-that-no-longer-exists")
	ledger := filepath.Join(dir, fmt.Sprintf("%064x.state", 7))
	if err := os.WriteFile(ledger, []byte("revision 1\n"+hookStateRootLine(dead)+"\ndesign\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := acquireHookStateLock(ledger)
	if err != nil {
		t.Fatal(err)
	}
	compaction, compactErr := compactHookStateDir(dir, 0)
	releaseErr := lock.release()
	if compactErr != nil || releaseErr != nil {
		t.Fatalf("compaction=%v release=%v", compactErr, releaseErr)
	}
	if compaction.Reclaimed != 0 {
		t.Fatalf("compaction reclaimed %d entries of a generation this process had locked", compaction.Reclaimed)
	}
	if _, err := os.Lstat(ledger); err != nil {
		t.Fatalf("compaction removed a ledger this process had locked: %v", err)
	}
	// With the lock gone the same generation is reclaimable, so the guard is
	// the lock and not some unrelated property of the fixture.
	compaction, err = compactHookStateDir(dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if compaction.Reclaimed != 1 {
		t.Fatalf("compaction reclaimed %d entries after the lock was released, want 1", compaction.Reclaimed)
	}
}

// TestStoreFullOfLiveObligationsStaysIntactAndStillArms is the other side of
// the bound: when every obligation in the store belongs to a project root that
// still exists, there is nothing safe to reclaim. Governance must keep working
// and every obligation must survive, because discharging one here is exactly
// the fail-open this retention policy refuses.
func TestStoreFullOfLiveObligationsStaysIntactAndStillArms(t *testing.T) {
	dir := isolateHookRetention(t)
	live := managedRoot(t)
	canonical, err := canonicalHookRoot(live)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("revision 1\n" + hookStateRootLine(canonical) + "\ndesign\n")
	for i := 0; i < hookStateDirRetentionCeiling+64; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%064x.state", i)), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	before := hookStoreEntries(t, dir)
	if out := runEvent(t, live, shellEvent("pressure", "echo governed")); out != "" {
		t.Fatalf("a governed event under store pressure was not allowed: %s", out)
	}
	if after := hookStoreEntries(t, dir); after < before {
		t.Fatalf("obligations for a live project root were reclaimed: %d entries, was %d", after, before)
	}
	if !hookStateBoundIsExhausted() {
		t.Fatal("a compaction that reclaimed nothing did not stop this process from rescanning on every later write")
	}
	if out := runEvent(t, live, shellEvent("pressure-again", "echo governed")); out != "" {
		t.Fatalf("a second governed event under store pressure was not allowed: %s", out)
	}
	if _, err := os.Lstat(filepath.Join(dir, fmt.Sprintf("%064x.state", 0))); err != nil {
		t.Fatalf("a live obligation was discharged under store pressure: %v", err)
	}
}

// TestStateReportNeverCreatesTheStore protects the durable-loss evidence. A
// machine that never armed an obligation has no store, and a report that
// materialized one would fabricate the very marker that tells a first
// initialization from a store that was lost.
func TestStateReportNeverCreatesTheStore(t *testing.T) {
	isolateHookState(t)
	dir, err := stateDirPathExact()
	if err != nil {
		t.Fatal(err)
	}
	for _, repair := range []bool{false, true} {
		var report strings.Builder
		ok, err := StateReport(&report, repair)
		if err != nil || !ok {
			t.Fatalf("report on an absent store (repair=%v) ok=%v err=%v", repair, ok, err)
		}
		if !strings.Contains(report.String(), "no governance hook state store") {
			t.Fatalf("report on an absent store (repair=%v) does not say so: %q", repair, report.String())
		}
		if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("a doctor report (repair=%v) created the store at %s: %v", repair, dir, err)
		}
	}
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a doctor report created the store initialization marker at %s: %v", marker, err)
	}
}

// TestStoreOfLegacyRootlessLedgersIsRetainedAndKeepsFailingClosed pins the
// limit of self-repair, so it cannot later be turned into a fail-open without
// someone deciding to. A ledger written before this version carries no project
// root, nothing can tell its obligation from a live one, and reclamation
// therefore refuses every one of them. A store filled past the limit by an
// older version stays failed closed until a human removes those files.
func TestStoreOfLegacyRootlessLedgersIsRetainedAndKeepsFailingClosed(t *testing.T) {
	dir := isolateHookRetention(t)
	root := managedRoot(t)
	legacy := []byte("revision 1\ndesign\n")
	for i := 0; i < hookStateDirMaxEntries+8; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%064x.state", i)), legacy, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	before := hookStoreEntries(t, dir)

	compaction, err := compactHookStateDir(dir, 0)
	if err != nil {
		t.Fatalf("compaction over a pre-upgrade store: %v", err)
	}
	if compaction.Reclaimed != 0 {
		t.Fatalf("compaction reclaimed %d entries whose obligation it cannot tell from a live one", compaction.Reclaimed)
	}
	if compaction.Retained < hookStateDirMaxEntries {
		t.Fatalf("compaction retained only %d of %d generations", compaction.Retained, before)
	}

	var repaired strings.Builder
	ok, err := StateReport(&repaired, true)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("doctor --repair reported a pre-upgrade store as healthy: %q", repaired.String())
	}
	if !strings.Contains(repaired.String(), "at or above its 4096-entry limit") {
		t.Fatalf("doctor --repair does not state that the store is still over the limit: %q", repaired.String())
	}
	if count := hookStoreEntries(t, dir); count != before {
		t.Fatalf("a pre-upgrade store changed size: %d entries, was %d", count, before)
	}

	out := runEvent(t, root, shellEvent("legacy", "echo governed"))
	if !strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Fatalf("a governed event on a pre-upgrade store over the limit was not denied: %q", out)
	}
	if count := hookStoreEntries(t, dir); count != before {
		t.Fatalf("a denied event changed a pre-upgrade store: %d entries, was %d", count, before)
	}
}

// TestStopRecoversRoutingAfterRetentionPrunedItsSnapshot covers what the route
// bound does downstream. A session whose snapshot was reclaimed no longer has
// an exact match at Stop time and falls into the shared-route recovery branch,
// which must recover the same configuration rather than block.
func TestStopRecoversRoutingAfterRetentionPrunedItsSnapshot(t *testing.T) {
	isolateHookRetention(t)
	root := managedRoot(t)
	pruned := "session-000"
	for i := 0; i < hookStateRouteRetention+4; i++ {
		session := fmt.Sprintf("session-%03d", i)
		if out := runEvent(t, root, shellEvent(session, "echo governed")); out != "" {
			t.Fatalf("governed shell event was not allowed: %s", out)
		}
	}
	if _, err := os.Lstat(routeStatePath(root, pruned)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("retention did not reclaim the oldest session's snapshot: %v", err)
	}
	cfg, present, err := loadRouteSnapshot(root, pruned)
	if err != nil || !present {
		t.Fatalf("routing recovery for a pruned session: present=%v err=%v", present, err)
	}
	if cfg.Design != "design" {
		t.Fatalf("recovered routing = %+v, want the project's own design directory", cfg)
	}
	out := runEvent(t, root, Input{SessionID: pruned, HookEventName: "Stop"})
	for _, forbidden := range []string{"routing", "route snapshot", "conflicting"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("Stop for a pruned session blocked on routing: %s", out)
		}
	}
}

// TestRouteRetentionReclaimsABoundedShareOfSurplusPerWrite pins the cost the
// bound may impose on one governed event. Surplus was unbounded before this
// policy existed, so the first event after an upgrade can meet hundreds of
// snapshots; it must reclaim a bounded share and let the project converge over
// the next few events instead of paying for the whole backlog inside one tool
// call, with every other repository's enumeration waiting behind it.
func TestRouteRetentionReclaimsABoundedShareOfSurplusPerWrite(t *testing.T) {
	isolateHookRetention(t)
	root := managedRoot(t)
	if out := runEvent(t, root, shellEvent("first", "echo governed")); out != "" {
		t.Fatalf("governed shell event was not allowed: %s", out)
	}
	body, err := os.ReadFile(routeStatePath(root, "first"))
	if err != nil {
		t.Fatal(err)
	}
	prefix := routeStatePrefix(root)
	const surplus = 2*hookStateRouteReclaimBudget + hookStateRouteRetention
	for i := 0; i < surplus; i++ {
		if err := os.WriteFile(prefix+fmt.Sprintf("%064x", i)+".json", body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	before := len(routeSnapshotPaths(t, root))

	if out := runEvent(t, root, shellEvent("second", "echo governed")); out != "" {
		t.Fatalf("governed shell event was not allowed: %s", out)
	}
	after := len(routeSnapshotPaths(t, root))
	if reclaimed := before + 1 - after; reclaimed > hookStateRouteReclaimBudget {
		t.Fatalf("one write reclaimed %d surplus snapshots, above the %d budget", reclaimed, hookStateRouteReclaimBudget)
	}
	if after <= hookStateRouteRetention {
		t.Fatalf("one write reclaimed the whole backlog (%d snapshots left); the per-write budget did not apply", after)
	}

	for i := 0; i < surplus; i++ {
		session := fmt.Sprintf("converge-%03d", i)
		if out := runEvent(t, root, shellEvent(session, "echo governed")); out != "" {
			t.Fatalf("governed shell event was not allowed: %s", out)
		}
		if len(routeSnapshotPaths(t, root)) <= hookStateRouteRetention {
			return
		}
	}
	t.Fatalf("route retention did not converge to %d snapshots over %d writes", hookStateRouteRetention, surplus)
}

func routeSnapshotPaths(t *testing.T, root string) []string {
	t.Helper()
	paths, err := routeStatePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	return paths
}
