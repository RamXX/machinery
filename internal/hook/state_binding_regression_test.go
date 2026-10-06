package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStateMarkerMismatchNamesGenerationAndDoctorSeesIt(t *testing.T) {
	isolateHookState(t)
	root := managedRoot(t)
	pre := editEvent("PreToolUse", "Write", "first-seat", filepath.Join(root, "design", "BUILD.md"))
	runEvent(t, root, pre)
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		t.Fatal(err)
	}
	present, legacy, binding, err := readStateInitializationMarker(marker)
	if err != nil || !present || legacy {
		t.Fatalf("marker: %v", err)
	}
	binding.generation = strings.Repeat("0", 64)
	if err := os.WriteFile(marker, binding.markerBody(), 0600); err != nil {
		t.Fatal(err)
	}
	pre.SessionID = "second-seat"
	denied := runEvent(t, root, pre)
	if !strings.Contains(denied, "generation") || strings.Contains(denied, "replacement store") {
		t.Errorf("refusal misidentifies the component: %s", denied)
	}
	var report bytes.Buffer
	ok, err := StateReport(&report, false)
	if err != nil {
		t.Fatal(err)
	}
	if ok || !strings.Contains(report.String(), "generation") || !strings.Contains(report.String(), "hook-state adopt") {
		t.Fatalf("doctor missed binding failure: ok=%v report=%s", ok, report.String())
	}
}

func TestSeatSessionChangeKeepsStoreBinding(t *testing.T) {
	isolateHookState(t)
	root := managedRoot(t)
	for _, seat := range []string{"first-seat", "second-seat"} {
		event := editEvent("PreToolUse", "Write", seat, filepath.Join(root, "design", "BUILD.md"))
		if out := runEvent(t, root, event); out != "" {
			t.Fatalf("seat denied: %s", out)
		}
	}
}

func TestAdoptionPreservesLedgerAndRestoresQuarantine(t *testing.T) {
	for _, quarantine := range []bool{false, true} {
		t.Run(map[bool]string{false: "marker", true: "quarantine"}[quarantine], func(t *testing.T) {
			isolateHookState(t)
			root := managedRoot(t)
			pre := editEvent("PreToolUse", "Write", "first-seat", filepath.Join(root, "design", "BUILD.md"))
			runEvent(t, root, pre)
			ledger := statePath(root, "")
			before, err := os.ReadFile(ledger)
			if err != nil {
				t.Fatal(err)
			}
			marker, err := stateInitializationMarkerPath()
			if err != nil {
				t.Fatal(err)
			}
			_, _, binding, err := readStateInitializationMarker(marker)
			if err != nil {
				t.Fatal(err)
			}
			from := ""
			if quarantine {
				from = stateDirPath() + ".parked"
				if err := os.Rename(stateDirPath(), from); err != nil {
					t.Fatal(err)
				}
				var report bytes.Buffer
				ok, err := StateReport(&report, false)
				if err != nil || ok || !strings.Contains(report.String(), "missing after prior initialization") {
					t.Fatalf("doctor missed missing initialized store: %v %v %s", ok, err, report.String())
				}
			} else {
				binding.generation = strings.Repeat("0", 64)
				if err := os.WriteFile(marker, binding.markerBody(), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var output bytes.Buffer
			if err := AdoptState(&output, root, from); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), "pending=1") || !strings.Contains(output.String(), "routes=1") {
				t.Fatalf("handoff did not report obligations: %s", output.String())
			}
			after, err := os.ReadFile(ledger)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("handoff lost ledger: %v before=%s after=%s", err, before, after)
			}
			journal, err := os.ReadFile(marker + ".handoffs")
			if err != nil || !strings.Contains(string(journal), "prior_generation") {
				t.Fatalf("handoff was not journaled: %s %v", journal, err)
			}
			pre.SessionID = "second-seat"
			if out := runEvent(t, root, pre); out != "" {
				t.Fatalf("adopted store denies second seat: %s", out)
			}
			var report bytes.Buffer
			if ok, err := StateReport(&report, false); err != nil || !ok {
				t.Fatalf("doctor denies adopted store: %s %v", report.String(), err)
			}
		})
	}
}

func TestAdoptionRefusesReplacementAndCorruptLedger(t *testing.T) {
	for _, replacement := range []bool{true, false} {
		t.Run(map[bool]string{true: "replacement", false: "corrupt"}[replacement], func(t *testing.T) {
			isolateHookState(t)
			root := managedRoot(t)
			runEvent(t, root, editEvent("PreToolUse", "Write", "seat", filepath.Join(root, "design", "BUILD.md")))
			if replacement {
				dir := stateDirPath()
				identity, err := os.ReadFile(filepath.Join(dir, stateDirectoryIdentityName))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(dir, dir+".parked"); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, stateDirectoryIdentityName), identity, 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(statePath(root, ""), []byte("corrupt\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var output bytes.Buffer
			err := AdoptState(&output, root, "")
			if err == nil {
				t.Fatal("invalid store adopted")
			}
			if replacement && !strings.Contains(err.Error(), "replacement store") {
				t.Fatalf("replacement not identified: %v", err)
			}
		})
	}
}

func TestMissingIdentityInOriginalStoreNamesMissingFile(t *testing.T) {
	isolateHookState(t)
	root := managedRoot(t)
	event := editEvent("PreToolUse", "Write", "seat", filepath.Join(root, "design", "BUILD.md"))
	runEvent(t, root, event)
	if err := os.Remove(filepath.Join(stateDirPath(), stateDirectoryIdentityName)); err != nil {
		t.Fatal(err)
	}
	out := runEvent(t, root, event)
	if !strings.Contains(out, "identity file") || strings.Contains(out, "replacement store") {
		t.Fatalf("missing identity in original directory misreported as replacement: %s", out)
	}
}

func TestAdoptionPreservesLegacyRootlessLedger(t *testing.T) {
	isolateHookState(t)
	root := managedRoot(t)
	event := editEvent("PreToolUse", "Write", "seat", filepath.Join(root, "design", "BUILD.md"))
	runEvent(t, root, event)
	ledger := statePath(root, "")
	raw, err := os.ReadFile(ledger)
	if err != nil {
		t.Fatal(err)
	}
	var legacy strings.Builder
	for _, line := range strings.Split(string(raw), "\n") {
		if line != "" && !strings.HasPrefix(line, "root ") {
			legacy.WriteString(line + "\n")
		}
	}
	before := []byte(legacy.String())
	if err := os.WriteFile(ledger, before, 0600); err != nil {
		t.Fatal(err)
	}
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		t.Fatal(err)
	}
	_, _, binding, err := readStateInitializationMarker(marker)
	if err != nil {
		t.Fatal(err)
	}
	binding.generation = strings.Repeat("0", 64)
	if err := os.WriteFile(marker, binding.markerBody(), 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := AdoptState(&output, root, ""); err != nil {
		t.Fatalf("valid legacy ledger refused: %v", err)
	}
	after, err := os.ReadFile(ledger)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("legacy ledger rewritten or lost: %s %v", after, err)
	}
	if !strings.Contains(output.String(), "pending=1") || !strings.Contains(output.String(), root) {
		t.Fatalf("legacy obligations not reported: %s", output.String())
	}
	if out := runEvent(t, root, event); out != "" {
		t.Fatalf("adopted legacy store denied tool: %s", out)
	}
}
