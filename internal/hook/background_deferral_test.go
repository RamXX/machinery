package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackgroundStopDefersWithoutChangingLedger(t *testing.T) {
	for _, event := range []string{"Stop", "SubagentStop"} {
		for _, pending := range []bool{true, false} {
			t.Run(event+"/pending="+map[bool]string{true: "yes", false: "no"}[pending], func(t *testing.T) {
				isolateHookState(t)
				root := managedRoot(t)
				pre := editEvent("PreToolUse", "Write", "deferred", filepath.Join(root, "design", "notes.txt"))
				pre.ToolUseID = "deferred-operation"
				runEvent(t, root, pre)
				impl := editEvent("PostToolUse", "Write", "deferred", filepath.Join(root, "src", "main.go"))
				runEvent(t, root, impl)
				if !pending {
					pre.HookEventName = "PostToolUse"
					runEvent(t, root, pre)
				}
				ledger := statePath(root, "deferred")
				before, err := os.ReadFile(ledger)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := json.Marshal(Input{SessionID: "deferred", HookEventName: event})
				if err != nil {
					t.Fatal(err)
				}
				raw = append(raw[:len(raw)-1], []byte(`,"background_tasks":[{"id":"running","status":"running"}]}`)...)
				var out bytes.Buffer
				if err := Run(bytes.NewReader(raw), &out, root); err != nil {
					t.Fatal(err)
				}
				if out.Len() != 0 {
					t.Fatalf("background Stop must defer: %s", out.String())
				}
				after, err := os.ReadFile(ledger)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(before, after) {
					t.Fatalf("deferral changed ledger: before=%s after=%s", before, after)
				}
				later := runEvent(t, root, Input{SessionID: "deferred", HookEventName: event})
				if pending && !strings.Contains(later, "in-flight tool") {
					t.Fatalf("later Stop did not enforce pending obligation: %s", later)
				}
				if !pending && !strings.Contains(later, "gate ERROR") {
					t.Fatalf("later Stop did not run gates: %s", later)
				}
				if !pending {
					state, err := readStateRecord(root, "deferred")
					if err != nil || state.design || state.impl {
						t.Fatalf("idle Stop did not discharge: %+v %v", state, err)
					}
				}
			})
		}
	}
}
