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
