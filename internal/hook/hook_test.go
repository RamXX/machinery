package hook

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/testgit"

	"github.com/RamXX/machinery/internal/designlock"
	"github.com/RamXX/machinery/internal/gates"
)

// --- helpers ---

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// managedRoot returns a temp project root marked machinery-managed by
// convention (design/domain.modelith.yaml).
func managedRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "design", "domain.modelith.yaml"), "model: {}\n")
	return root
}

func writeSparseFile(t *testing.T, path string, size int64) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(file.Truncate(size), file.Close()); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeInputRejectsOversizeStringsAndArraysDeterministically(t *testing.T) {
	largeString := `{"hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"content":"` +
		strings.Repeat("x", int(hookInputMaxBytes)) + `"}}`
	arrayItems := int(hookInputMaxBytes)/len(`{"id":"x"},`) + 1
	largeArray := `{"hook_event_name":"Stop","background_tasks":[` +
		strings.Repeat(`{"id":"x"},`, arrayItems) + `{"id":"x"}]}`
	want := fmt.Sprintf("hook-event JSON exceeds %d-byte limit", hookInputMaxBytes)
	for _, tt := range []struct {
		name string
		body string
	}{
		{name: "tool payload string", body: largeString},
		{name: "object array", body: largeArray},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var prior string
			for attempt := 0; attempt < 2; attempt++ {
				_, err := decodeInput(strings.NewReader(tt.body))
				if err == nil || err.Error() != want {
					t.Fatalf("oversize hook input error = %v, want %q", err, want)
				}
				if attempt > 0 && err.Error() != prior {
					t.Fatalf("oversize diagnostic changed between runs: %q != %q", err.Error(), prior)
				}
				prior = err.Error()
			}
		})
	}
}

func TestConfinedGovernanceReadsRejectOversizeSparseFiles(t *testing.T) {
	t.Run("config", func(t *testing.T) {
		root := t.TempDir()
		writeSparseFile(t, filepath.Join(root, ConfigName), hookConfigMaxBytes+1)
		cfg, ok, first := Load(root)
		_, _, second := Load(root)
		want := fmt.Sprintf("%s exceeds %d-byte limit", ConfigName, hookConfigMaxBytes)
		if !ok || !strings.Contains(cfg.loadError, want) || first != second {
			t.Fatalf("oversize config did not fail closed deterministically: cfg=%+v ok=%v first=%q second=%q", cfg, ok, first, second)
		}
	})

	t.Run("conventional marker", func(t *testing.T) {
		root := t.TempDir()
		writeSparseFile(t, filepath.Join(root, filepath.FromSlash(conventionalMarker)), hookMarkerMaxBytes+1)
		cfg, ok, first := Load(root)
		_, _, second := Load(root)
		want := fmt.Sprintf("%s exceeds %d-byte limit", conventionalMarker, hookMarkerMaxBytes)
		if !ok || !strings.Contains(cfg.loadError, want) || first != second {
			t.Fatalf("oversize marker did not fail closed deterministically: cfg=%+v ok=%v first=%q second=%q", cfg, ok, first, second)
		}
	})

	t.Run("wave sentinel", func(t *testing.T) {
		design := t.TempDir()
		writeSparseFile(t, filepath.Join(design, waveSentinelName), hookWaveMaxBytes+1)
		if left, stale, active := waveSentinel(design); left != "" || !stale || !active {
			t.Fatalf("oversize wave sentinel did not fail closed: left=%q stale=%v active=%v", left, stale, active)
		}
		_, present, first := readConfinedRegular(design, waveSentinelName, hookWaveMaxBytes)
		_, _, second := readConfinedRegular(design, waveSentinelName, hookWaveMaxBytes)
		want := fmt.Sprintf("%s exceeds %d-byte limit", waveSentinelName, hookWaveMaxBytes)
		if !present || first == nil || first.Error() != want || second == nil || second.Error() != want {
			t.Fatalf("oversize wave diagnostic is not stable: present=%v first=%v second=%v", present, first, second)
		}
	})
}

func isolateHookState(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
}

func mutateHookStateReadOnce(t *testing.T, phase, path string, mutation func() error) {
	t.Helper()
	prior := hookStateReadPhase
	var once sync.Once
	var mutationErr error
	hookStateReadPhase = func(gotPhase, gotPath string) {
		if gotPhase == phase && gotPath == path {
			once.Do(func() { mutationErr = mutation() })
		}
	}
	t.Cleanup(func() {
		hookStateReadPhase = prior
		if mutationErr != nil {
			t.Errorf("hook state read mutation failed: %v", mutationErr)
		}
	})
}

func replacePathPreservingOriginal(path, parked string, replacement []byte) error {
	if err := os.Rename(path, parked); err != nil {
		return err
	}
	return os.WriteFile(path, replacement, 0o600)
}

// copyTree copies the go-crm example design into dst for gate-level tests.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	if err := copyDesignTree(src, dst); err != nil {
		t.Fatal(err)
	}
}

// copyDesignTree takes a governed reader snapshot before copying a shared
// example. Full-package tests run concurrently with writer tests and developer
// processes; a raw Walk can otherwise capture the private publication sentinel
// or a cross-file partial state. A genuinely stale/invalid sentinel remains a
// loud fixture error instead of being skipped or deleted.
func copyDesignTree(src, dst string) (retErr error) {
	snapshot, err := designlock.AcquireReader(src)
	if err != nil {
		return fmt.Errorf("acquire coherent design fixture %s: %w", src, err)
	}
	defer func() {
		retErr = errors.Join(retErr, snapshot.Release())
	}()
	source := snapshot.SourceRoot()
	return filepath.Walk(source, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func TestCopyDesignTreeWaitsForConcurrentPublication(t *testing.T) {
	source := t.TempDir()
	destination := filepath.Join(t.TempDir(), "copy")
	writeFile(t, filepath.Join(source, "ARCHITECTURE.md"), "before\n")

	writer, err := designlock.Acquire(source)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		done <- copyDesignTree(source, destination)
	}()
	<-started
	select {
	case err := <-done:
		_ = writer.Release()
		t.Fatalf("fixture copy crossed an active design writer: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	output := filepath.Join(source, "generated.md")
	if err := writer.PublishExpected("fixture-writer", "rerun fixture writer", []designlock.OutputExpectation{
		designlock.ExpectFile(output, []byte("complete\n"), 0o644),
	}, func() error {
		return os.WriteFile(output, []byte("complete\n"), 0o644)
	}); err != nil {
		_ = writer.Release()
		t.Fatal(err)
	}
	if err := writer.Release(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("fixture copy remained blocked after publication completed")
	}
	if body, err := os.ReadFile(filepath.Join(destination, "generated.md")); err != nil || string(body) != "complete\n" {
		t.Fatalf("fixture copy did not observe the completed publication: %q, %v", body, err)
	}
	if _, err := os.Lstat(filepath.Join(destination, ".machinery-design-publish.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fixture copy captured a private publication sentinel: %v", err)
	}
}

func TestCopyDesignTreeRejectsStaleSentinelWithoutDeletingIt(t *testing.T) {
	source := t.TempDir()
	destination := filepath.Join(t.TempDir(), "copy")
	writeFile(t, filepath.Join(source, "ARCHITECTURE.md"), "stable\n")
	sentinel := filepath.Join(source, ".machinery-design-publish.json")
	body := []byte("not a valid publication record\n")
	if err := os.WriteFile(sentinel, body, 0o600); err != nil {
		t.Fatal(err)
	}

	err := copyDesignTree(source, destination)
	if err == nil || !strings.Contains(err.Error(), "invalid interrupted Machinery publication sentinel") {
		t.Fatalf("stale fixture sentinel did not fail closed: %v", err)
	}
	if got, readErr := os.ReadFile(sentinel); readErr != nil || !bytes.Equal(got, body) {
		t.Fatalf("fixture copy deleted or rewrote unrelated residue: %q, %v", got, readErr)
	}
	if _, statErr := os.Lstat(destination); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("copy began before stale residue validation: %v", statErr)
	}
}

// runEvent pipes one synthesized event through Run and returns stdout.
func runEvent(t *testing.T, root string, in Input) string {
	t.Helper()
	if (in.HookEventName == "PreToolUse" || in.HookEventName == "PostToolUse" || in.HookEventName == "PostToolUseFailure") && in.ToolUseID == "" {
		body, err := json.Marshal(in.ToolInput)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		in.ToolUseID = fmt.Sprintf("test-%s-%x", in.ToolName, sum[:8])
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(bytes.NewReader(raw), &out, root); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return out.String()
}

func TestRunRejectsOpenOrAmbiguousHookProtocol(t *testing.T) {
	root := managedRoot(t)
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"duplicate outer key", `{"hook_event_name":"SessionStart","hook_event_name":"Stop"}`, `duplicate hook-event key "hook_event_name"`},
		{"unknown outer key", `{"hook_event_name":"SessionStart","hook_event_nam":"Stop"}`, `unknown hook-event key "hook_event_nam"`},
		{"trailing document", `{"hook_event_name":"SessionStart"} {}`, "trailing JSON value"},
		{"wrong scalar type", `{"hook_event_name":7}`, `key "hook_event_name" must be a string`},
		{"duplicate tool key", `{"hook_event_name":"PreToolUse","tool_input":{"file_path":"a","file_path":"b"}}`, `duplicate tool_input key "file_path"`},
		{"unknown tool key", `{"hook_event_name":"PreToolUse","tool_input":{"file_pat":"a"}}`, `unknown tool_input key "file_pat"`},
		{"unsupported event", `{"hook_event_name":"PostToolUes"}`, `unsupported hook event "PostToolUes"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Run(strings.NewReader(tc.raw), io.Discard, root)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ambiguous hook input did not fail closed: err=%v, want %q", err, tc.want)
			}
		})
	}
}

func TestRunAcceptsKnownAdapterMetadata(t *testing.T) {
	isolateHookState(t)
	root := managedRoot(t)
	raw := `{"session_id":"s","prompt_id":"550e8400-e29b-41d4-a716-446655440000","cwd":"` + filepath.ToSlash(root) + `","hook_event_name":"SessionStart","transcript_path":"transcript.jsonl","permission_mode":"default","effort":{"level":"high"},"source":"resume","model":"claude-opus-4-6","session_title":"hardening","seconds_since_last_response":12.5,"context_tokens":4096,"prompt_cache_likely_expired":false,"estimated_cache_write_usd":0.0125}`
	var out bytes.Buffer
	if err := Run(strings.NewReader(raw), &out, root); err != nil {
		t.Fatalf("official adapter metadata was rejected: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("valid SessionStart did not dispatch")
	}
	stop := `{"session_id":"s","prompt_id":"550e8400-e29b-41d4-a716-446655440000","cwd":"` + filepath.ToSlash(root) + `","hook_event_name":"Stop","stop_hook_active":false,"last_assistant_message":"done","background_tasks":[],"session_crons":[{"id":"cron-1","schedule":"0 9 * * 1-5","recurring":true,"prompt":"check the build"}]}`
	if err := Run(strings.NewReader(stop), io.Discard, root); err != nil {
		t.Fatalf("current official Stop metadata was rejected: %v", err)
	}
}

func TestRunRejectsInvalidClosedCurrentMetadata(t *testing.T) {
	root := managedRoot(t)
	for _, tc := range []struct {
		name, raw, want string
	}{
		{"unknown effort key", `{"hook_event_name":"Stop","effort":{"level":"high","typo":true}}`, `unknown hook-event effort key "typo"`},
		{"unknown background task key", `{"hook_event_name":"Stop","background_tasks":[{"id":"task","typo":"x"}]}`, `unknown hook-event background_tasks item key "typo"`},
		{"wrong cron boolean", `{"hook_event_name":"Stop","session_crons":[{"recurring":"yes"}]}`, `session_crons item.recurring`},
		{"negative duration", `{"hook_event_name":"PostToolUse","duration_ms":-0.5}`, `nonnegative finite number`},
		{"string duration", `{"hook_event_name":"PostToolUse","duration_ms":"12"}`, `nonnegative finite number`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := Run(strings.NewReader(tc.raw), io.Discard, root); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid current metadata was accepted: err=%v want=%q", err, tc.want)
			}
		})
	}
}

func TestPostToolUseFailureCompletesOnlyItsExactPendingOperation(t *testing.T) {
	isolateHookState(t)
	root := managedRoot(t)
	target := filepath.Join(root, "design", "notes.txt")
	pre := editEvent("PreToolUse", "Write", "failed-tool-session", target)
	pre.ToolUseID = "failed-tool-use"
	if out := runEvent(t, root, pre); out != "" {
		t.Fatalf("allowed PreToolUse unexpectedly emitted output: %s", out)
	}
	state, err := readStateRecord(root, "replacement-session")
	if err != nil || !state.design || len(state.pending) != 1 {
		t.Fatalf("PreToolUse did not durably arm one exact operation: state=%+v err=%v", state, err)
	}

	failureRaw := `{"session_id":"failed-tool-session","tool_use_id":"failed-tool-use","cwd":"` + filepath.ToSlash(root) + `","hook_event_name":"PostToolUseFailure","tool_name":"Write","tool_input":{"file_path":"` + filepath.ToSlash(target) + `"},"error":"permission denied","is_interrupt":false,"duration_ms":1.5}`
	if err := Run(strings.NewReader(failureRaw), io.Discard, root); err != nil {
		t.Fatalf("PostToolUseFailure did not durably terminate its operation: %v", err)
	}
	state, err = readStateRecord(root, "replacement-session")
	if err != nil || !state.design || len(state.pending) != 0 {
		t.Fatalf("failed operation completion cleared the obligation or remained in flight: state=%+v err=%v", state, err)
	}
}

func editEvent(event, tool, sessionID, file string) Input {
	return Input{
		SessionID:     sessionID,
		HookEventName: event,
		ToolName:      tool,
		ToolInput:     toolInput{FilePath: file},
	}
}

func codexPatchEvent(event, sessionID, patch string) Input {
	return Input{
		SessionID:     sessionID,
		HookEventName: event,
		ToolName:      "apply_patch",
		ToolInput:     toolInput{Command: patch},
	}
}

// --- detection: the no-op guarantee ---

func TestLoadDetection(t *testing.T) {
	t.Run("unmanaged dir is not managed", func(t *testing.T) {
		_, ok, _ := Load(t.TempDir())
		if ok {
			t.Fatal("bare directory must not count as machinery-managed")
		}
	})
	t.Run("conventional design marks managed", func(t *testing.T) {
		root := managedRoot(t)
		cfg, ok, warn := Load(root)
		if !ok || cfg.Design != "design" || warn != "" {
			t.Fatalf("got cfg=%+v ok=%v warn=%q", cfg, ok, warn)
		}
	})
	t.Run("config marks managed with custom design dir", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ConfigName), `{"design":"blueprint","gates":"g2,g3","impl":".","strict":true}`)
		cfg, ok, _ := Load(root)
		if !ok || cfg.Design != "blueprint" || cfg.Gates != "g2,g3" || cfg.Impl != "." || !cfg.Strict {
			t.Fatalf("got cfg=%+v ok=%v", cfg, ok)
		}
	})
	t.Run("empty config defaults design before validation", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ConfigName), `{}`)
		cfg, ok, warn := Load(root)
		if !ok || cfg.Design != "design" || cfg.loadError != "" || warn != "" {
			t.Fatalf("empty config did not retain canonical default: cfg=%+v ok=%v warn=%q", cfg, ok, warn)
		}
	})
	t.Run("hooks false opts out", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ConfigName), `{"hooks": false}`)
		if _, ok, _ := Load(root); ok {
			t.Fatal("hooks:false must disable governance")
		}
	})
	t.Run("unparseable config stays managed and warns", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ConfigName), `{not json`)
		cfg, ok, warn := Load(root)
		if !ok || warn == "" || cfg.Design != "design" {
			t.Fatalf("a config typo must degrade loudly, not disable governance: cfg=%+v ok=%v warn=%q", cfg, ok, warn)
		}
	})
	t.Run("invalid config entry stays managed and cannot opt out", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			setup func(*testing.T, string)
		}{
			{"symlink", func(t *testing.T, root string) {
				outside := filepath.Join(t.TempDir(), "config.json")
				writeFile(t, outside, `{"hooks":false}`)
				if err := os.Symlink(outside, filepath.Join(root, ConfigName)); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			}},
			{"directory", func(t *testing.T, root string) {
				if err := os.Mkdir(filepath.Join(root, ConfigName), 0o755); err != nil {
					t.Fatal(err)
				}
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				root := t.TempDir()
				tc.setup(t, root)
				cfg, ok, warn := Load(root)
				if !ok || cfg.Hooks != nil || !strings.Contains(warn, "invalid or unreadable") {
					t.Fatalf("invalid marker disabled governance: cfg=%+v ok=%v warn=%q", cfg, ok, warn)
				}
			})
		}
	})
	t.Run("invalid conventional marker stays managed and cannot opt out", func(t *testing.T) {
		for _, kind := range []string{"directory", "symlink"} {
			t.Run(kind, func(t *testing.T) {
				root := t.TempDir()
				marker := filepath.Join(root, conventionalMarker)
				if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
					t.Fatal(err)
				}
				if kind == "directory" {
					if err := os.Mkdir(marker, 0o755); err != nil {
						t.Fatal(err)
					}
				} else {
					outside := filepath.Join(t.TempDir(), "model.yaml")
					writeFile(t, outside, "model: {}\n")
					if err := os.Symlink(outside, marker); err != nil {
						t.Skipf("symlinks unavailable: %v", err)
					}
				}
				cfg, ok, warn := Load(root)
				if !ok || cfg.loadError == "" || !strings.Contains(warn, "invalid or unreadable") {
					t.Fatalf("invalid conventional marker disabled governance: cfg=%+v ok=%v warn=%q", cfg, ok, warn)
				}
			})
		}
	})
	t.Run("unreadable config stays managed where permissions are enforced", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, ConfigName)
		writeFile(t, path, `{"hooks":false}`)
		if err := os.Chmod(path, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
		if f, err := os.Open(path); err == nil {
			f.Close()
			t.Skip("host privileges bypass file permission checks")
		}
		cfg, ok, warn := Load(root)
		if !ok || cfg.Hooks != nil || !strings.Contains(warn, "invalid or unreadable") {
			t.Fatalf("unreadable marker disabled governance: cfg=%+v ok=%v warn=%q", cfg, ok, warn)
		}
	})
	t.Run("unknown duplicate and mistyped keys stay managed with defaults", func(t *testing.T) {
		for _, tc := range []struct {
			name, body, want string
		}{
			{"unknown", `{"desgin":"blueprint"}`, "unknown config key"},
			{"duplicate", `{"hooks":false,"hooks":true}`, "duplicate config key"},
			{"string type", `{"design":42}`, "must be a string"},
			{"string null", `{"gates":null}`, "must be a string"},
			{"bool type", `{"strict":"true"}`, "must be a boolean"},
			{"bool null", `{"hooks":null}`, "must be a boolean"},
			{"root type", `[]`, "root must be a JSON object"},
			{"trailing", `{} {}`, "trailing JSON value"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				root := t.TempDir()
				writeFile(t, filepath.Join(root, ConfigName), tc.body)
				cfg, ok, warn := Load(root)
				if !ok || cfg.Design != "design" || cfg.Hooks != nil || !strings.Contains(warn, tc.want) {
					t.Fatalf("invalid config did not fail closed: cfg=%+v ok=%v warn=%q, want %q", cfg, ok, warn, tc.want)
				}
			})
		}
	})
	t.Run("unknown gate in list is a hard managed error", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ConfigName), `{"gates":"g2,gg"}`)
		cfg, ok, warn := Load(root)
		if !ok || cfg.loadError == "" || !strings.Contains(warn, `unknown gate "gg"`) {
			t.Fatalf("got cfg=%+v ok=%v warn=%q", cfg, ok, warn)
		}
	})
	t.Run("hooks false cannot hide an invalid closed value", func(t *testing.T) {
		for _, body := range []string{
			`{"hooks":false,"gates":"g2,gg"}`,
			`{"hooks":false,"dialog":"terse"}`,
		} {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ConfigName), body)
			cfg, ok, warn := Load(root)
			if !ok || cfg.loadError == "" || warn == "" {
				t.Fatalf("invalid opt-out config was not a hard managed error: body=%s cfg=%+v ok=%v warn=%q", body, cfg, ok, warn)
			}
		}
	})
	t.Run("escaping design and impl paths are hard config errors", func(t *testing.T) {
		for _, body := range []string{`{"design":".."}`, `{"design":"../elsewhere"}`, `{"design":"."}`, `{"design":"/abs"}`, `{"impl":"../outside"}`} {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ConfigName), body)
			cfg, ok, _ := Load(root)
			if !ok || cfg.loadError == "" || !strings.Contains(cfg.loadError, "path is invalid") {
				t.Fatalf("unsafe config path was not a hard managed error: body=%s cfg=%+v ok=%v", body, cfg, ok)
			}
		}
	})
}

func TestRunIsNoopWhenUnmanaged(t *testing.T) {
	root := t.TempDir()
	for _, ev := range []string{"PreToolUse", "PostToolUse", "PostToolUseFailure", "Stop", "SubagentStop", "SessionStart"} {
		out := runEvent(t, root, editEvent(ev, "Write", "s1", filepath.Join(root, "design", "machines", "X.oracle.md")))
		if out != "" {
			t.Fatalf("%s in an unmanaged repo must produce no output, got %q", ev, out)
		}
	}
}

func TestRunIsNoopWhenUnmanagedWithoutHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	for _, ev := range []string{"PreToolUse", "PostToolUse", "PostToolUseFailure", "Stop", "SubagentStop", "SessionStart"} {
		out := runEvent(t, root, editEvent(ev, "Write", "no-home-unmanaged", filepath.Join(root, "notes.txt")))
		if out != "" {
			t.Fatalf("%s disturbed an unmanaged repo without HOME: %q", ev, out)
		}
	}
}

// --- PreToolUse: generated artifacts are read-only ---

func TestPreDeniesGeneratedArtifacts(t *testing.T) {
	root := managedRoot(t)
	writeFile(t, filepath.Join(root, "design", "checkers", "pii.checker.yaml"), `checker:
  id: pii
  description: PII flow checker
  runtime_closure: sha256:1111111111111111111111111111111111111111111111111111111111111111
projection:
  include: [model]
coverage:
  claim: []
evidence:
  projection_out: checkers/pii/custom-projection.json
  evidence_in: checkers/pii/custom-evidence.json
`)
	writeFile(t, filepath.Join(root, "design", "checkers", "pii", "generated", "custom-trace.json"), `{}`)
	writeFile(t, filepath.Join(root, "design", "checkers", "pii", "custom-evidence.json"), `{
  "evidence_schema":"1.0",
  "checker":{"id":"pii","version":"1"},
  "input_hash":"sha256:0000000000000000000000000000000000000000000000000000000000000000",
  "runtime_closure":"sha256:1111111111111111111111111111111111111111111111111111111111111111",
  "verdict":"pass",
  "coverage":[],
  "trace_ref":"generated/custom-trace.json"
}`)
	cases := []struct {
		rel  string
		tool string
		deny bool
		hint string
	}{
		{"design/machines/Deal.oracle.md", "Edit", true, "machinery oracle"},
		{"design/machines/Deal.oracle.md", "Write", true, "machinery oracle"},
		{"design/formal/Deal.tla", "Write", true, "verify-formal"},
		{"design/formal/Deal.cfg", "MultiEdit", true, "verify-formal"},
		{"design/packs/billing.pack/domain.yaml", "Write", true, "pack generate"},
		{"design/pack/contract.machine.json", "Edit", true, "frozen pack"},
		{"design/ratchet.json", "Edit", true, "machinery baseline"},
		{"design/ratchet.json", "Write", true, "defeats the ratchet"},
		{"design/formal/Policy.als", "Write", true, "machinery alloy"},
		{"design/checkers/pii/custom-projection.json", "Write", true, "machinery project"},
		{"design/checkers/pii/custom-evidence.json", "Edit", true, "verify-checkers"},
		{"design/checkers/pii/generated/custom-trace.json", "Write", true, "verify-checkers"},
		{"design/formal/Deal.semantics.yaml", "Edit", false, ""},    // annotation source
		{"design/formal/policy.relational.yaml", "Edit", false, ""}, // annotation source
		{"design/machines/Deal.machine.json", "Edit", false, ""},    // machine source
		{"design/machines/Deal.matrix.md", "Edit", false, ""},       // hand matrix
		{"design/domain.modelith.md", "Edit", false, ""},            // rendered, but post-processed by hand
		{"src/main.go", "Write", false, ""},
	}
	for _, c := range cases {
		t.Run(c.tool+" "+c.rel, func(t *testing.T) {
			out := runEvent(t, root, editEvent("PreToolUse", c.tool, "s-pre", filepath.Join(root, c.rel)))
			if !c.deny {
				if out != "" {
					t.Fatalf("expected allow (no output), got %q", out)
				}
				return
			}
			var got preOut
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("deny output is not JSON: %v (%q)", err, out)
			}
			if got.HookSpecificOutput.PermissionDecision != "deny" {
				t.Fatalf("expected deny, got %+v", got)
			}
			if !strings.Contains(got.HookSpecificOutput.PermissionDecisionReason, c.hint) {
				t.Fatalf("reason %q missing hint %q", got.HookSpecificOutput.PermissionDecisionReason, c.hint)
			}
		})
	}
}

func TestInvalidConfigBlocksEveryGovernanceEventWithoutDefaultRouting(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"blueprint"`)
	custom := filepath.Join(root, "blueprint", "machines", "Order.machine.json")
	writeFile(t, custom, `{}`)

	preOutRaw := runEvent(t, root, editEvent("PreToolUse", "Write", "bad-config", custom))
	var preResult preOut
	if err := json.Unmarshal([]byte(preOutRaw), &preResult); err != nil || preResult.HookSpecificOutput.PermissionDecision != "deny" {
		t.Fatalf("invalid config did not deny PreToolUse: output=%q err=%v", preOutRaw, err)
	}
	if !strings.Contains(preResult.HookSpecificOutput.PermissionDecisionReason, "guessed defaults") {
		t.Fatalf("PreToolUse denial did not explain invalid routing: %+v", preResult)
	}

	postInput := editEvent("PostToolUse", "Write", "bad-config", custom)
	postRaw, err := json.Marshal(postInput)
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(bytes.NewReader(postRaw), io.Discard, root); err == nil || !strings.Contains(err.Error(), "guessed defaults") {
		t.Fatalf("invalid config did not fail PostToolUse: %v", err)
	}

	stopRaw := runEvent(t, root, Input{HookEventName: "Stop", SessionID: "bad-config"})
	var stopResult stopOut
	if err := json.Unmarshal([]byte(stopRaw), &stopResult); err != nil || stopResult.Decision != "block" {
		t.Fatalf("invalid config did not block Stop: output=%q err=%v", stopRaw, err)
	}
}

func TestInvalidClosedConfigValuesDenyPreAndBlockStop(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"unknown gate", `{"gates":"g2,gg"}`},
		{"invalid dialog", `{"dialog":"terse"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ConfigName), tc.body)
			preRaw := runEvent(t, root, editEvent("PreToolUse", "Write", "closed-config", filepath.Join(root, "design", "BUILD.md")))
			var preResult preOut
			if err := json.Unmarshal([]byte(preRaw), &preResult); err != nil || preResult.HookSpecificOutput.PermissionDecision != "deny" {
				t.Fatalf("invalid closed config did not deny PreToolUse: output=%q err=%v", preRaw, err)
			}
			stopRaw := runEvent(t, root, Input{HookEventName: "Stop", SessionID: "closed-config"})
			var stopResult stopOut
			if err := json.Unmarshal([]byte(stopRaw), &stopResult); err != nil || stopResult.Decision != "block" {
				t.Fatalf("invalid closed config did not block Stop: output=%q err=%v", stopRaw, err)
			}
		})
	}
}

func TestPreIgnoresPathsOutsideRoot(t *testing.T) {
	root := managedRoot(t)
	other := t.TempDir()
	out := runEvent(t, root, editEvent("PreToolUse", "Edit", "s-out", filepath.Join(other, "design", "machines", "X.oracle.md")))
	if out != "" {
		t.Fatalf("a path outside the project root is not ours to police, got %q", out)
	}
}

func TestCodexApplyPatchDeniesAnyGeneratedArtifact(t *testing.T) {
	root := managedRoot(t)
	patch := "*** Begin Patch\n" +
		"*** Update File: src/main.go\n" +
		"@@\n-old\n+new\n" +
		"*** Update File: design/machines/Deal.oracle.md\n" +
		"@@\n-old\n+new\n" +
		"*** End Patch"
	out := runEvent(t, root, codexPatchEvent("PreToolUse", "s-codex-pre", patch))
	var got preOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Codex deny output is not JSON: %v (%q)", err, out)
	}
	if got.HookSpecificOutput.PermissionDecision != "deny" {
		t.Fatalf("a multi-file patch touching a generated artifact must be denied: %+v", got)
	}
	if !strings.Contains(got.HookSpecificOutput.PermissionDecisionReason, "machinery oracle") {
		t.Fatalf("deny reason must identify the regeneration command: %+v", got)
	}
}

func TestEditedPathsParsesCodexPatchOperations(t *testing.T) {
	in := codexPatchEvent("PreToolUse", "s-paths", "*** Begin Patch\n"+
		"*** Add File: src/new.go\n"+
		"*** Update File: src/old.go\n"+
		"*** Move to: src/moved.go\n"+
		"*** Delete File: src/gone.go\n"+
		"*** Update File: src/old.go\n"+
		"*** End Patch")
	got := editedPaths(in)
	want := []string{"src/new.go", "src/old.go", "src/gone.go", "src/moved.go"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("editedPaths = %v, want %v", got, want)
	}
}

// --- PostToolUse: the touched ledger ---

func TestPostRecordsTouches(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"impl":"."}`)
	sid := "s-post"
	t.Cleanup(func() { clearState(root, sid) })

	runEvent(t, root, editEvent("PostToolUse", "Write", sid, filepath.Join(root, "design", "machines", "Deal.machine.json")))
	if d, i := readState(root, sid); !d || i {
		t.Fatalf("design edit: got design=%v impl=%v", d, i)
	}
	runEvent(t, root, editEvent("PostToolUse", "Edit", sid, filepath.Join(root, "src", "main.go")))
	if d, i := readState(root, sid); !d || !i {
		t.Fatalf("source edit: got design=%v impl=%v", d, i)
	}
}

func TestCodexApplyPatchRecordsAllTouchedClasses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"impl":"."}`)
	sid := "s-codex-post"
	t.Cleanup(func() { clearState(root, sid) })
	patch := "*** Begin Patch\n" +
		"*** Update File: design/machines/Deal.machine.json\n" +
		"@@\n-old\n+new\n" +
		"*** Add File: src/main.go\n" +
		"+package main\n" +
		"*** End Patch"
	runEvent(t, root, codexPatchEvent("PostToolUse", sid, patch))
	if d, i := readState(root, sid); !d || !i {
		t.Fatalf("Codex multi-file patch: got design=%v impl=%v", d, i)
	}
}

func TestPostIgnoresNonSourceAndUnwatched(t *testing.T) {
	root := managedRoot(t) // no impl configured
	sid := "s-post2"
	t.Cleanup(func() { clearState(root, sid) })
	runEvent(t, root, editEvent("PostToolUse", "Write", sid, filepath.Join(root, "README.md")))
	runEvent(t, root, editEvent("PostToolUse", "Write", sid, filepath.Join(root, "src", "main.go"))) // impl not set
	if d, i := readState(root, sid); d || i {
		t.Fatalf("nothing watched was edited: got design=%v impl=%v", d, i)
	}
}

// --- Stop: gates run before the turn ends ---

const crmDesign = "../../examples/go-crm/design"

func TestStopSilentWhenNothingTouched(t *testing.T) {
	root := managedRoot(t)
	out := runEvent(t, root, Input{SessionID: "s-idle", HookEventName: "Stop"})
	if out != "" {
		t.Fatalf("a session that touched nothing must stop silently, got %q", out)
	}
}

func TestStopGreenDesignClearsStateSilently(t *testing.T) {
	root := t.TempDir()
	copyTree(t, crmDesign, filepath.Join(root, "design"))
	sid := "s-green"
	appendState(root, sid, "design")
	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	if out != "" {
		t.Fatalf("green gates must be silent, got %q", out)
	}
	if d, i := readState(root, sid); d || i {
		t.Fatal("state must clear after a green pass")
	}
}

func TestPreEditObligationSurvivesLostPostAndReplacementSession(t *testing.T) {
	root := t.TempDir()
	copyTree(t, crmDesign, filepath.Join(root, "design"))
	before := "session-before-crash"
	after := "session-after-crash"
	t.Cleanup(func() { _ = clearState(root, after) })
	target := filepath.Join(root, "design", "notes.txt")
	pre := editEvent("PreToolUse", "Write", before, target)
	pre.ToolUseID = "crashed-file-tool"
	if out := runEvent(t, root, pre); out != "" {
		t.Fatalf("allowed edit preflight unexpectedly emitted output: %s", out)
	}
	// Simulate the tool completing and the host crashing before PostToolUse.
	writeFile(t, target, "ordinary design note\n")
	if design, impl, err := readStateErr(root, after); err != nil || !design || impl {
		t.Fatalf("replacement session did not inherit the project obligation: design=%v impl=%v err=%v", design, impl, err)
	}
	if out := runEvent(t, root, Input{SessionID: after, HookEventName: "Stop"}); !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "in-flight tool") {
		t.Fatalf("replacement session discharged an operation whose Post event was lost: %s", out)
	}
	post := editEvent("PostToolUse", "Write", before, target)
	post.ToolUseID = pre.ToolUseID
	if out := runEvent(t, root, post); out != "" {
		t.Fatalf("late durable Post completion failed: %s", out)
	}
	if out := runEvent(t, root, Input{SessionID: after, HookEventName: "Stop"}); out != "" {
		t.Fatalf("replacement session failed to discharge the completed green obligation: %s", out)
	}
	if design, impl, err := readStateErr(root, before); err != nil || design || impl {
		t.Fatalf("successful gates did not clear the project obligation for every session: design=%v impl=%v err=%v", design, impl, err)
	}
}

func TestStopRetainsCompletedObligationWhileBackgroundTasksRun(t *testing.T) {
	isolateHookState(t)
	root := managedRoot(t)
	target := filepath.Join(root, "design", "notes.txt")
	pre := editEvent("PreToolUse", "Write", "background-session", target)
	pre.ToolUseID = "background-tool-use"
	if out := runEvent(t, root, pre); out != "" {
		t.Fatalf("allowed PreToolUse unexpectedly emitted output: %s", out)
	}
	post := editEvent("PostToolUse", "Write", "background-session", target)
	post.ToolUseID = pre.ToolUseID
	if out := runEvent(t, root, post); out != "" {
		t.Fatalf("PostToolUse unexpectedly emitted output: %s", out)
	}

	stopRaw := `{"session_id":"background-session","cwd":"` + filepath.ToSlash(root) + `","hook_event_name":"Stop","background_tasks":[{"id":"task-1","type":"local","status":"running","description":"mutating files"}]}`
	var stopOutput bytes.Buffer
	if err := Run(strings.NewReader(stopRaw), &stopOutput, root); err != nil {
		t.Fatal(err)
	}
	out := stopOutput.String()
	if !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "background task") {
		t.Fatalf("Stop discharged a project obligation while a background task remained: %s", out)
	}
	state, err := readStateRecord(root, "replacement-session")
	if err != nil || !state.design || len(state.pending) != 0 {
		t.Fatalf("background-task block lost the completed dirty obligation: state=%+v err=%v", state, err)
	}
}

func TestWaveDeferralSurvivesCrashAndOutOfBandClose(t *testing.T) {
	root := t.TempDir()
	copyTree(t, crmDesign, filepath.Join(root, "design"))
	oracle := filepath.Join(root, "design", "machines", "Deal.oracle.md")
	raw, err := os.ReadFile(oracle)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, oracle, string(raw)+"\ntampered during wave\n")
	wave := filepath.Join(root, "design", waveSentinelName)
	writeFile(t, wave, "open\n")
	before := "wave-session-before-crash"
	after := "wave-session-after-crash"
	t.Cleanup(func() { _ = clearState(root, after) })
	if err := appendState(root, before, "design"); err != nil {
		t.Fatal(err)
	}
	deferred := runEvent(t, root, Input{SessionID: before, HookEventName: "Stop"})
	if !strings.Contains(deferred, "deferred to wave close") || strings.Contains(deferred, `"decision":"block"`) {
		t.Fatalf("open wave did not defer while retaining the obligation: %s", deferred)
	}
	if err := os.Remove(wave); err != nil {
		t.Fatal(err)
	}
	closed := runEvent(t, root, Input{SessionID: after, HookEventName: "Stop"})
	if !strings.Contains(closed, `"decision":"block"`) || !strings.Contains(closed, "DRIFT") {
		t.Fatalf("replacement session skipped the gate after out-of-band wave close: %s", closed)
	}
	if design, _, err := readStateErr(root, after); err != nil || !design {
		t.Fatalf("red post-close gates cleared the durable obligation: design=%v err=%v", design, err)
	}
}

func TestStopDriftBlocks(t *testing.T) {
	root := t.TempDir()
	copyTree(t, crmDesign, filepath.Join(root, "design"))
	oracle := filepath.Join(root, "design", "machines", "Deal.oracle.md")
	raw, err := os.ReadFile(oracle)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, oracle, string(raw)+"\ntampered\n")
	sid := "s-drift"
	t.Cleanup(func() { clearState(root, sid) })
	appendState(root, sid, "design")

	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision != "block" {
		t.Fatalf("a stale oracle must block the stop, got %+v", got)
	}
	if !strings.Contains(got.Reason, "DRIFT") {
		t.Fatalf("block reason must carry the gate output, got %q", got.Reason)
	}
	if d, _ := readState(root, sid); !d {
		t.Fatal("state must survive a block so the re-check runs after the fix")
	}

	// A repeated Stop is still blocking. The host must not be able to clear
	// governance state merely by setting stop_hook_active while gates are red.
	out = runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop", StopHookActive: true})
	got = stopOut{}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision != "block" || !strings.Contains(got.Reason, "remain red") {
		t.Fatalf("with stop_hook_active the hook must remain blocking: %+v", got)
	}
	if d, _ := readState(root, sid); !d {
		t.Fatal("blocking state must survive every repeated Stop until gates are green")
	}
}

func TestStopMidPhaseErrorsWarnOnly(t *testing.T) {
	root := managedRoot(t)
	// Phase 2 in flight: an ARCHITECTURE.md with no parseable contract is an
	// ERROR, but no machines and no BUILD.md exist, so g3/gx do not apply.
	writeFile(t, filepath.Join(root, "design", "ARCHITECTURE.md"), "# Architecture\n(draft)\n")
	sid := "s-midphase"
	t.Cleanup(func() { clearState(root, sid) })
	appendState(root, sid, "design")

	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision == "block" {
		t.Fatalf("mid-phase ERRORs must not block a non-strict stop: %+v", got)
	}
	if !strings.Contains(got.SystemMessage, "ERROR") {
		t.Fatalf("the warning must still surface the red gates: %+v", got)
	}
}

func TestStopStrictBlocksOnAnyFinding(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"strict": true}`)
	writeFile(t, filepath.Join(root, "design", "domain.modelith.yaml"), "model: {}\n")
	writeFile(t, filepath.Join(root, "design", "ARCHITECTURE.md"), "# Architecture\n(draft)\n")
	sid := "s-strict"
	t.Cleanup(func() { clearState(root, sid) })
	appendState(root, sid, "design")

	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision != "block" {
		t.Fatalf("strict mode must block on any blocking finding: %+v", got)
	}
}

func TestStopBeforeAnyGateApplies(t *testing.T) {
	// Phase 1 skeleton: the conventional marker exists, so gc applies from
	// turn one exactly as the CLI default suite arms it. An empty model is a
	// finding ("an empty check is a failure, not a pass"), surfaced as the
	// informational mid-phase message, never a block (no DRIFT, not strict).
	root := managedRoot(t)
	sid := "s-phase1"
	appendState(root, sid, "design")
	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision != "" {
		t.Fatalf("a mid-phase ERROR must not block: %+v", got)
	}
	if !strings.Contains(got.SystemMessage, "ERROR finding(s)") {
		t.Fatalf("the empty-model carrier finding must surface in the message: %+v", got)
	}
	if d, _ := readState(root, sid); d {
		t.Fatal("state must clear after a non-blocking stop")
	}
}

// TestSelectGatesProgressiveOptional locks progressive opt-in behavior: each
// relational annotation and the migration contract turns on its own gate at
// stop time without requiring configuration.
func TestSelectGatesProgressiveOptional(t *testing.T) {
	dir := t.TempDir()
	formal := filepath.Join(dir, "formal")
	if err := os.MkdirAll(formal, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "legacy"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Strictness changes blocking policy, not the progressive activation set.
	sel, _ := selectGates(dir, Config{Strict: true})
	for _, g := range []string{"gm", "gs", "gp", "gi", "gn"} {
		if sel.Run[g] {
			t.Errorf("%s must not run before its annotation exists", g)
		}
	}
	writeFile(t, filepath.Join(dir, "migration.yaml"), "contract_version: 1\n")
	writeFile(t, filepath.Join(dir, "legacy", "surface.yaml"), "surface_version: 1\n")
	writeFile(t, filepath.Join(formal, "policy.relational.yaml"), "subjects: {}\n")
	writeFile(t, filepath.Join(formal, "integrity.relational.yaml"), "entities: []\n")
	writeFile(t, filepath.Join(formal, "isolation.relational.yaml"), "tenant: {}\n")
	sel, _ = selectGates(dir, Config{})
	for _, g := range []string{"gm", "gs", "gp", "gi", "gn"} {
		if !sel.Run[g] {
			t.Errorf("%s must run once its opt-in artifact exists", g)
		}
	}
}

// The stop hook mirrors the CLI default suite for every checkable-from-design
// gate: gc arms on the domain model, gd on machines, gk on the external-
// checker layer. Omitting gk once let checker DRIFT (a stale committed
// projection after a mid-session model edit) pass the turn end green while
// the CLI reported it and exited 1.
func TestSelectGatesArmsCarrierIdciteCheckers(t *testing.T) {
	dir := t.TempDir()
	sel, _ := selectGates(dir, Config{})
	for _, g := range []string{"gc", "gd", "gk"} {
		if sel.Run[g] {
			t.Errorf("%s must not run before its artifact exists: %v", g, sel.Run)
		}
	}
	writeFile(t, filepath.Join(dir, "domain.modelith.yaml"), "entities: {}\n")
	writeFile(t, filepath.Join(dir, "machines", "Order.machine.json"), "{}\n")
	writeFile(t, filepath.Join(dir, "checkers", "pii.checker.yaml"), "checker_version: 1\n")
	sel, _ = selectGates(dir, Config{})
	for _, g := range []string{"gc", "gd", "gk"} {
		if !sel.Run[g] {
			t.Errorf("%s must run once its artifact exists (CLI parity): %v", g, sel.Run)
		}
	}
}

// A machine-less decomposed parent (decomposition.yaml, BUILD.md, an empty
// machines/ directory) must not select Gx at stop time: its behavior layer is
// the children's, and Gx against the parent's BUILD.md would fail it for
// phases that live in the child designs. The artifact-activated gates
// (gm/gs/gp/gi/gn) keep their auto-activation on that same parent (the v0.3.0
// CLI narrowing regression dropped gp/gi/gn; the hook path must never copy
// that). Once machines exist the full selection returns.
func TestSelectGatesSkipsGxOnMachinelessDecomposedParent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "checkout.modelith.yaml"), "entities: {}\n")
	writeFile(t, filepath.Join(dir, "decomposition.yaml"), "decomposition_version: 1\n")
	writeFile(t, filepath.Join(dir, "BUILD.md"), "Mode: full\n")
	writeFile(t, filepath.Join(dir, "ARCHITECTURE.md"), "# arch\n")
	writeFile(t, filepath.Join(dir, "migration.yaml"), "contract_version: 1\n")
	writeFile(t, filepath.Join(dir, "legacy", "surface.yaml"), "surface_version: 1\n")
	writeFile(t, filepath.Join(dir, "formal", "policy.relational.yaml"), "subjects: {}\n")
	writeFile(t, filepath.Join(dir, "formal", "integrity.relational.yaml"), "entities: []\n")
	writeFile(t, filepath.Join(dir, "formal", "isolation.relational.yaml"), "tenant: {}\n")
	if err := os.MkdirAll(filepath.Join(dir, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	sel, _ := selectGates(dir, Config{})
	if sel.Run["gx"] || sel.Run["g3"] {
		t.Errorf("machine-less decomposed parent must not select g3/gx: %v", sel.Run)
	}
	if !sel.Run["g2"] || !sel.Run["g5"] {
		t.Errorf("machine-less decomposed parent must select g2,g5: %v", sel.Run)
	}
	for _, g := range []string{"gm", "gs", "gp", "gi", "gn"} {
		if !sel.Run[g] {
			t.Errorf("machine-less decomposed parent dropped artifact-activated gate %s: %v", g, sel.Run)
		}
	}
	writeFile(t, filepath.Join(dir, "machines", "Order.machine.json"), "{}\n")
	sel, _ = selectGates(dir, Config{})
	if !sel.Run["gx"] || !sel.Run["g3"] {
		t.Errorf("with machines present g3,gx must return: %v", sel.Run)
	}
}

// The stop hook selects Gx as soon as the model and machines exist: BUILD.md
// is not a precondition, or phase-3 Gx DRIFT (a stale maps-to reference)
// escapes the drift-blocking contract until Phase 4 (GATE-6).
func TestSelectGatesGxWithoutBuildDoc(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "domain.modelith.yaml"), "entities: {}\n")
	writeFile(t, filepath.Join(dir, "machines", "Order.machine.json"), "{}\n")
	sel, _ := selectGates(dir, Config{})
	if !sel.Run["gx"] {
		t.Errorf("modelith + machines must select gx even without BUILD.md: %v", sel.Run)
	}
	if sel.Run["gb"] {
		t.Errorf("gb still needs BUILD.md: %v", sel.Run)
	}
}

// Machine detection must survive glob metacharacters in the project path: a
// design under "pr[1]" once defeated filepath.Glob, silently dropping g3 and
// letting committed-oracle DRIFT pass at stop time (GATE-3/GATE-7 hook half).
func TestSelectGatesMetacharDesignPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pr[1]", "design")
	writeFile(t, filepath.Join(dir, "domain.modelith.yaml"), "entities: {}\n")
	writeFile(t, filepath.Join(dir, "machines", "Order.machine.json"), "{}\n")
	sel, _ := selectGates(dir, Config{})
	if !sel.Run["g3"] || !sel.Run["gx"] {
		t.Errorf("a metachar path must not defeat machine detection: %v", sel.Run)
	}
}

// The governance configuration itself is agent-read-only: a Write of
// {"hooks": false} to .machinery.json is how an agent would switch machinery
// off (GATE-10).
func TestPreDeniesGovernanceConfigEdits(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), "{}\n")
	for _, tool := range []string{"Write", "Edit"} {
		out := runEvent(t, root, editEvent("PreToolUse", tool, "s-gov", filepath.Join(root, ConfigName)))
		var got preOut
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("%s deny output is not JSON: %v (%q)", tool, err, out)
		}
		if got.HookSpecificOutput.PermissionDecision != "deny" {
			t.Fatalf("%s of %s must be denied: %+v", tool, ConfigName, got)
		}
		if !strings.Contains(got.HookSpecificOutput.PermissionDecisionReason, "governance") {
			t.Fatalf("the reason must say why: %+v", got)
		}
	}
}

// Deleting a governance marker via a Codex apply_patch switches machinery off
// for the whole repo; updating the domain model is Phase 1 authoring and
// stays allowed (GATE-10).
func TestCodexDeleteOfGovernanceMarkerDenied(t *testing.T) {
	root := managedRoot(t)
	deny := codexPatchEvent("PreToolUse", "s-del", "*** Begin Patch\n"+
		"*** Delete File: design/domain.modelith.yaml\n"+
		"*** End Patch")
	out := runEvent(t, root, deny)
	var got preOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("deny output is not JSON: %v (%q)", err, out)
	}
	if got.HookSpecificOutput.PermissionDecision != "deny" {
		t.Fatalf("deleting the design marker must be denied: %+v", got)
	}
	allow := codexPatchEvent("PreToolUse", "s-upd", "*** Begin Patch\n"+
		"*** Update File: design/domain.modelith.yaml\n"+
		"@@\n-old\n+new\n"+
		"*** End Patch")
	if out := runEvent(t, root, allow); out != "" {
		t.Fatalf("updating the domain model is Phase 1 authoring and must stay allowed, got %q", out)
	}
}

// The wave sentinel defers stop-time gating while it is fresh, so an agent
// that could touch it would defer the gates indefinitely. It is operator-only:
// file-tool creates and edits are denied wherever the base name appears, while
// deleting it (the documented way to close a wave) stays allowed.
func TestPreDeniesWaveSentinelWrites(t *testing.T) {
	root := managedRoot(t)
	cases := []struct {
		name string
		rel  string
		tool string
		deny bool
	}{
		{"root design dir, Write", "design/.machinery-wave", "Write", true},
		{"root design dir, Edit", "design/.machinery-wave", "Edit", true},
		{"root design dir, MultiEdit", "design/.machinery-wave", "MultiEdit", true},
		{"nested child design", "design/children/billing/.machinery-wave", "Write", true},
		{"outside the design dir", "ops/.machinery-wave", "Write", true},
		{"lookalike suffix is not the sentinel", "design/.machinery-wave.bak", "Write", false},
		{"lookalike prefix is not the sentinel", "design/wave.md", "Write", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := runEvent(t, root, editEvent("PreToolUse", c.tool, "s-wave", filepath.Join(root, c.rel)))
			if !c.deny {
				if out != "" {
					t.Fatalf("expected allow (no output), got %q", out)
				}
				return
			}
			var got preOut
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("deny output is not JSON: %v (%q)", err, out)
			}
			if got.HookSpecificOutput.PermissionDecision != "deny" {
				t.Fatalf("writing the wave sentinel must be denied: %+v", got)
			}
			reason := got.HookSpecificOutput.PermissionDecisionReason
			for _, want := range []string{c.rel, "wave sentinel", "operator-created"} {
				if !strings.Contains(reason, want) {
					t.Fatalf("reason %q missing %q", reason, want)
				}
			}
		})
	}
}

func TestShellGovernanceDeniesProtectedTargetsAndArmsStop(t *testing.T) {
	root := managedRoot(t)
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"design","impl":"impl","strict":true}`)
	writeFile(t, filepath.Join(root, "impl", "main.go"), "package main\n")
	for _, tc := range []struct {
		name, command, want string
	}{
		{"generated artifact", "printf x > design/machines/Deal.oracle.md", "generated or frozen"},
		{"wave sentinel", "touch design/.machinery-wave", "protected machinery governance"},
		{"routing config", "printf '{}\\n' > .machinery.json", "protected machinery governance"},
		{"awk inplace", "awk -i inplace '{print}' design/ratchet.json", "generated or frozen"},
		{"ed", "ed design/machines/Deal.oracle.md", "generated or frozen"},
		{"sponge", "generate | sponge design/formal/Deal.tla", "generated or frozen"},
		{"script wrapper", "./scripts/rewrite design/pack/domain.yaml", "generated or frozen"},
		{"make output", "make OUTPUT=design/formal/Policy.als", "generated or frozen"},
		{"read verb cannot prove safety", "cat design/ratchet.json", "generated or frozen"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := runEvent(t, root, Input{HookEventName: "PreToolUse", ToolName: "Bash", SessionID: "shell-pre", ToolInput: toolInput{Command: tc.command}})
			if !strings.Contains(out, `"permissionDecision":"deny"`) || !strings.Contains(out, tc.want) {
				t.Fatalf("protected shell mutation was not denied: %s", out)
			}
		})
	}
	sid := "shell-post"
	writeFile(t, filepath.Join(root, "design", "machines", "Deal.oracle.md"), "shell mutation\n")
	if out := runEvent(t, root, Input{HookEventName: "PostToolUse", ToolName: "Bash", SessionID: sid, ToolInput: toolInput{Command: "computed shell mutation"}}); out != "" {
		t.Fatalf("PostToolUse should record silently, got %s", out)
	}
	designTouched, implTouched, err := readStateErr(root, sid)
	if err != nil || !designTouched || !implTouched {
		t.Fatalf("shell PostToolUse must conservatively arm both ledgers: design=%v impl=%v err=%v", designTouched, implTouched, err)
	}
	out := runEvent(t, root, Input{HookEventName: "Stop", SessionID: sid})
	if !strings.Contains(out, `"decision":"block"`) {
		t.Fatalf("Stop skipped a shell-authored governed-tree mutation: %s", out)
	}
}

func TestPreShellRouteSurvivesGovernanceMarkerDeletion(t *testing.T) {
	root := managedRoot(t)
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"design","impl":"impl","strict":true}`)
	writeFile(t, filepath.Join(root, "impl", "main.go"), "package main\n")
	sid := "shell-disarm"
	pre := Input{
		HookEventName: "PreToolUse", ToolName: "Bash", SessionID: sid, ToolUseID: "quoted-marker-delete",
		ToolInput: toolInput{Command: `env rm .machine""ry.json design/domain.modelith.""yaml`},
	}
	out := runEvent(t, root, pre)
	if out != "" {
		t.Fatalf("quoted-concatenation command should exercise durable recovery instead of the literal pre-deny: %s", out)
	}
	designTouched, implTouched, err := readStateErr(root, sid)
	if err != nil || !designTouched || !implTouched {
		t.Fatalf("PreToolUse did not durably arm both ledgers before denial: design=%v impl=%v err=%v", designTouched, implTouched, err)
	}
	if _, present, err := loadRouteSnapshot(root, sid); err != nil || !present {
		t.Fatalf("pre-shell routing snapshot was not durable: present=%v err=%v", present, err)
	}

	// Execute the exact dynamically assembled targets. Neither protected name
	// is a literal substring, so Post and Stop must route from the durable
	// pre-event snapshot outside the mutable project markers.
	cmd := exec.CommandContext(t.Context(), "/bin/sh", "-c", pre.ToolInput.Command)
	cmd.Dir = root
	if commandOut, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("execute quoted marker deletion: %v: %s", err, commandOut)
	}
	postRaw, err := json.Marshal(Input{HookEventName: "PostToolUse", ToolName: "Bash", SessionID: sid, ToolUseID: "quoted-marker-delete"})
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(bytes.NewReader(postRaw), io.Discard, root); err == nil || !strings.Contains(err.Error(), "routing config changed") {
		t.Fatalf("PostToolUse forgot the pre-shell route after marker deletion: %v", err)
	}
	stopRaw, err := json.Marshal(Input{HookEventName: "Stop", SessionID: "replacement-stop-session"})
	if err != nil {
		t.Fatal(err)
	}
	var stopOutput bytes.Buffer
	if err := Run(bytes.NewReader(stopRaw), &stopOutput, root); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stopOutput.String(), `"decision":"block"`) || !strings.Contains(stopOutput.String(), "in-flight tool") {
		t.Fatalf("Stop silently unmanaged a pre-shell governed session: %s", stopOutput.String())
	}
	t.Cleanup(func() {
		_ = clearState(root, sid)
	})
}

func TestDurableStateRecoversManagedAncestorFromNestedDirectoryAfterMarkerDeletion(t *testing.T) {
	isolateHookState(t)
	root := managedRoot(t)
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"design","impl":"impl","strict":true}`)
	writeFile(t, filepath.Join(root, "impl", "main.go"), "package main\n")
	nested := filepath.Join(root, "nested", "work")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	pre := Input{
		SessionID: "nested-marker-delete", ToolUseID: "nested-shell-tool",
		Cwd: nested, HookEventName: "PreToolUse", ToolName: "Bash",
		ToolInput: toolInput{Command: `env rm ../../.machine""ry.json ../../design/domain.modelith.""yaml`},
	}
	if out := runEvent(t, nested, pre); out != "" {
		t.Fatalf("allowed nested PreToolUse unexpectedly emitted output: %s", out)
	}
	if err := os.Remove(filepath.Join(root, ConfigName)); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, conventionalMarker)); err != nil {
		t.Fatal(err)
	}

	wantRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	gotRoot, err := canonicalHookRoot(nested)
	if err != nil || gotRoot != wantRoot {
		t.Fatalf("durable project identity did not recover the managed ancestor: got=%q want=%q err=%v", gotRoot, wantRoot, err)
	}
	out := runEvent(t, nested, Input{SessionID: "replacement-session", Cwd: nested, HookEventName: "Stop"})
	if !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "in-flight tool") {
		t.Fatalf("nested Stop silently unmanaged an in-flight operation after marker deletion: %s", out)
	}
}

func TestShimDispatchesAfterMarkersDisappearWhenBinaryExists(t *testing.T) {
	root := t.TempDir()
	binDir := t.TempDir()
	fake := filepath.Join(binDir, "machinery")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nprintf '%s' \"$*\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	shim, err := filepath.Abs(repoPath("hooks", "machinery-hook.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []string{"PostToolUse", "Stop", "SubagentStop", "SessionStart"} {
		t.Run(event, func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), "/bin/sh", shim)
			cmd.Dir = root
			cmd.Env = []string{
				"CLAUDE_PROJECT_DIR=" + root,
				"HOME=" + t.TempDir(),
				"PATH=" + binDir + string(os.PathListSeparator) + "/usr/bin:/bin",
			}
			cmd.Stdin = strings.NewReader(`{"hook_event_name":"` + event + `"}`)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("markerless %s was not dispatched: %v: %s", event, err, out)
			}
			want := "hook --root " + root
			if string(out) != want {
				t.Fatalf("shim invocation = %q, want %q", out, want)
			}
		})
	}
}

// The filesystems this hook guards on (APFS, NTFS) resolve names case-
// insensitively, so the enforcement reads (os.Stat of the sentinel,
// os.ReadFile of the config) find a case-variant spelling; the deny must
// fold case too or the guard is bypassable by writing .MACHINERY-WAVE.
func TestPreDeniesCaseVariants(t *testing.T) {
	root := managedRoot(t)
	cases := []struct{ name, rel, want string }{
		{"wave sentinel upper", "design/.MACHINERY-WAVE", "wave sentinel"},
		{"governance config mixed", ".Machinery.json", "governance configuration"},
		{"ratchet mixed", "design/Ratchet.json", "machinery baseline"},
		{"oracle mixed suffix", "design/machines/Thing.Oracle.MD", "machinery oracle"},
		{"frozen pack mixed", "design/Pack/events.md", "frozen pack"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := runEvent(t, root, editEvent("PreToolUse", "Write", "s-fold", filepath.Join(root, c.rel)))
			var got preOut
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("deny output is not JSON: %v (%q)", err, out)
			}
			if got.HookSpecificOutput.PermissionDecision != "deny" {
				t.Fatalf("case variant %s must be denied: %+v", c.rel, got)
			}
			if !strings.Contains(got.HookSpecificOutput.PermissionDecisionReason, c.want) {
				t.Fatalf("reason %q missing %q", got.HookSpecificOutput.PermissionDecisionReason, c.want)
			}
		})
	}
}

func TestCodexPatchWaveSentinel(t *testing.T) {
	root := managedRoot(t)
	add := codexPatchEvent("PreToolUse", "s-wave-add", "*** Begin Patch\n"+
		"*** Add File: design/.machinery-wave\n"+
		"+240\n"+
		"*** End Patch")
	out := runEvent(t, root, add)
	var got preOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("deny output is not JSON: %v (%q)", err, out)
	}
	if got.HookSpecificOutput.PermissionDecision != "deny" {
		t.Fatalf("an apply_patch that creates the sentinel must be denied: %+v", got)
	}
	del := codexPatchEvent("PreToolUse", "s-wave-del", "*** Begin Patch\n"+
		"*** Delete File: design/.machinery-wave\n"+
		"*** End Patch")
	if out := runEvent(t, root, del); out != "" {
		t.Fatalf("deleting the sentinel closes the wave and must stay allowed, got %q", out)
	}
}

// A delete of the sentinel is allowed because it closes the wave, but the
// delete must not launder any other write to the same path in the same tool
// call: a patch that deletes and re-adds .machinery-wave reopens a fresh
// full-TTL wave in one call and must be denied.
func TestCodexPatchWaveSentinelDeleteDoesNotLaunderRewrite(t *testing.T) {
	root := managedRoot(t)
	cases := []struct {
		name  string
		patch string
	}{
		{"delete then add", "*** Begin Patch\n" +
			"*** Delete File: design/.machinery-wave\n" +
			"*** Add File: design/.machinery-wave\n" +
			"+240\n" +
			"*** End Patch"},
		{"add then delete", "*** Begin Patch\n" +
			"*** Add File: design/.machinery-wave\n" +
			"+240\n" +
			"*** Delete File: design/.machinery-wave\n" +
			"*** End Patch"},
		{"delete then update", "*** Begin Patch\n" +
			"*** Delete File: design/.machinery-wave\n" +
			"*** Update File: design/.machinery-wave\n" +
			"@@\n-45\n+240\n" +
			"*** End Patch"},
		{"delete one sentinel, add another", "*** Begin Patch\n" +
			"*** Delete File: design/.machinery-wave\n" +
			"*** Add File: design/children/billing/.machinery-wave\n" +
			"+240\n" +
			"*** End Patch"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := runEvent(t, root, codexPatchEvent("PreToolUse", "s-wave-relaunder", c.patch))
			var got preOut
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("deny output is not JSON: %v (%q)", err, out)
			}
			if got.HookSpecificOutput.PermissionDecision != "deny" {
				t.Fatalf("re-creating the sentinel in the same patch that deletes it must be denied: %+v", got)
			}
			reason := got.HookSpecificOutput.PermissionDecisionReason
			for _, want := range []string{"wave sentinel", "operator-created"} {
				if !strings.Contains(reason, want) {
					t.Fatalf("reason %q missing %q", reason, want)
				}
			}
		})
	}
}

// editedOps is the substrate the per-operation deny rests on: a single patch
// that deletes and re-adds one path must report both operations, and a plain
// file-tool write must never be classified as a delete.
func TestEditedOpsReportsOperationPerPath(t *testing.T) {
	patch := "*** Begin Patch\n" +
		"*** Delete File: design/.machinery-wave\n" +
		"*** Add File: design/.machinery-wave\n" +
		"+240\n" +
		"*** Update File: design/notes.md\n" +
		"*** End Patch"
	got := editedOps(Input{ToolName: "apply_patch", ToolInput: toolInput{Command: patch}})
	want := []editedPath{
		{Path: "design/.machinery-wave", Op: opDelete},
		{Path: "design/.machinery-wave", Op: opAdd},
		{Path: "design/notes.md", Op: opUpdate},
	}
	if len(got) != len(want) {
		t.Fatalf("editedOps = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("editedOps[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	if paths := editedPaths(Input{ToolName: "apply_patch", ToolInput: toolInput{Command: patch}}); len(paths) != 2 {
		t.Fatalf("editedPaths must stay deduplicated by path, got %v", paths)
	}
	write := editedOps(Input{ToolName: "Write", ToolInput: toolInput{FilePath: "design/.machinery-wave"}})
	if len(write) != 1 || write[0].Op != opWrite {
		t.Fatalf("a file-tool write must report opWrite, got %v", write)
	}
}

func TestStopMissingDesignDirFailsClosed(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"blueprint"}`)
	sid := "s-nodir"
	appendState(root, sid, "design")
	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision != "block" || !strings.Contains(got.Reason, "blueprint") {
		t.Fatalf("a missing design dir must fail closed, got %+v", got)
	}
	if d, _ := readState(root, sid); !d {
		t.Fatal("missing-design block must retain touched state")
	}
}

func TestStopAutoSelectionRejectsInvalidActivationMarkers(t *testing.T) {
	for _, rel := range []string{
		"migration.yaml",
		"domain.modelith.yaml",
		filepath.Join("checkers", "pii.checker.yaml"),
	} {
		t.Run(rel, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ConfigName), `{}`)
			design := filepath.Join(root, "design")
			if err := os.MkdirAll(filepath.Dir(filepath.Join(design, rel)), 0o755); err != nil {
				t.Fatal(err)
			}
			outside := filepath.Join(t.TempDir(), filepath.Base(rel))
			writeFile(t, outside, "outside-sentinel\n")
			if err := os.Symlink(outside, filepath.Join(design, rel)); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			sid := "s-auto-preflight-" + strings.ReplaceAll(rel, string(filepath.Separator), "-")
			appendState(root, sid, "design")
			out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
			var got stopOut
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("stop output is not JSON: %v (%q)", err, out)
			}
			if got.Decision != "block" || !strings.Contains(got.Reason, "inventory") || !strings.Contains(got.Reason, "symlink") {
				t.Fatalf("invalid auto-activation marker bypassed universal inventory: %+v", got)
			}
			if done, _ := readState(root, sid); !done {
				t.Fatal("inventory block must retain touched state for re-check")
			}
		})
	}
}

// writeG4Fixture builds a managed root whose impl has one undeclared
// cross-boundary import (alpha -> beta), so G4 is red at stop time.
func writeG4Fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"impl":"."}`)
	arch := "# Architecture\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: alpha\n    code: [\"alpha/**\"]\n  - id: beta\n    code: [\"beta/**\"]\n```\n"
	writeFile(t, filepath.Join(root, "design", "ARCHITECTURE.md"), arch)
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/m\n")
	writeFile(t, filepath.Join(root, "alpha", "a.go"), "package alpha\n\nimport \"example.com/m/beta\"\n")
	writeFile(t, filepath.Join(root, "beta", "b.go"), "package beta\n")
	return root
}

func TestStopImportFindingsDisarmedThenArmed(t *testing.T) {
	root := writeG4Fixture(t)
	sid := "s-arming"
	t.Cleanup(func() { clearState(root, sid) })

	// no ratchet.json: pre-baseline debt warns with the arming instruction
	appendState(root, sid, "impl")
	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision == "block" {
		t.Fatalf("import findings must not block before a baseline exists: %+v", got)
	}
	if !strings.Contains(got.SystemMessage, "disarmed") || !strings.Contains(got.SystemMessage, "machinery baseline") {
		t.Fatalf("the warning must name the arming step: %+v", got)
	}

	// an empty snapshot (greenfield arming) makes the same finding block
	writeFile(t, filepath.Join(root, "design", "ratchet.json"), `{"date":"2026-07","edges":{}}`)
	appendState(root, sid, "impl")
	out = runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	got = stopOut{}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision != "block" {
		t.Fatalf("with the baseline recorded an import finding must block: %+v", got)
	}
	if !strings.Contains(got.Reason, "undeclared cross-boundary edge") {
		t.Fatalf("the block must carry the gate output: %q", got.Reason)
	}
}

// A staged gates list naming the impl-facing gates (gt, g4) with no impl
// configured must not fail the stop, but the drop has to stay visible: a
// silently skipped gate is a configured-but-never-run gate.
func TestStopWarnsWhenStagedImplGatesLackImpl(t *testing.T) {
	// green design, otherwise-silent stop: the warning must still surface
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"gates":"g2,g3,gt"}`)
	copyTree(t, crmDesign, filepath.Join(root, "design"))
	sid := "s-dropped"
	t.Cleanup(func() { clearState(root, sid) })
	appendState(root, sid, "design")
	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision == "block" {
		t.Fatalf("a config gap must warn, never block: %+v", got)
	}
	if !strings.Contains(got.SystemMessage, "gt") || !strings.Contains(got.SystemMessage, "impl") {
		t.Fatalf("the dropped gate and the missing impl setting must be named: %+v", got)
	}

	// the whole staged list impl-facing: nothing runs, the warning names both
	writeFile(t, filepath.Join(root, ConfigName), `{"gates":"g4,gt"}`)
	appendState(root, sid, "design")
	out = runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	got = stopOut{}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if !strings.Contains(got.SystemMessage, "g4,gt") {
		t.Fatalf("an all-dropped list must still warn: %+v", got)
	}
}

// --- SessionStart: the governance announcement ---

func TestSessionStartAnnouncesGovernance(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"gates":"g2,g4","impl":"."}`)
	writeFile(t, filepath.Join(root, "design", "STATE.md"), "Phase 1: gate-passed\nPhase 2: in-progress\n")
	out := runEvent(t, root, Input{HookEventName: "SessionStart"})
	for _, want := range []string{"machinery-managed", "g2,g4", "oracle.md", "STATE.md", "Phase 2: in-progress"} {
		if !strings.Contains(out, want) {
			t.Fatalf("session context missing %q:\n%s", want, out)
		}
	}
}

func TestSessionStartSilentWhenUnmanaged(t *testing.T) {
	out := runEvent(t, t.TempDir(), Input{HookEventName: "SessionStart"})
	if out != "" {
		t.Fatalf("unmanaged repos get no session context, got %q", out)
	}
}

// --- project-wide durable state ledger ---

func TestStatePathIsolatesRootsAndJoinsSessions(t *testing.T) {
	rootA, rootB := t.TempDir(), t.TempDir()
	if statePath(rootA, "s1") == statePath(rootB, "s1") {
		t.Fatal("different roots must not share a ledger")
	}
	if statePath(rootA, "s1") != statePath(rootA, "s2") {
		t.Fatal("replacement sessions must inherit one project gate obligation")
	}
	if routeStatePath(rootA, "s1") == routeStatePath(rootA, "s2") {
		t.Fatal("session routing snapshots must not overwrite each other")
	}
	if p := statePath(rootA, "../../etc/passwd"); strings.Contains(filepath.Base(p), "/") {
		t.Fatalf("session id must be sanitized into the filename, got %q", p)
	}
	configDir, err := os.UserConfigDir()
	if err == nil {
		rel, relErr := filepath.Rel(configDir, stateDirPath())
		if relErr != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
			t.Fatalf("durable hook state is not rooted in the user config directory: %s", stateDirPath())
		}
	}
	if filepath.Clean(stateDirPath()) == filepath.Clean(os.TempDir()) || strings.HasPrefix(stateDirPath(), filepath.Clean(os.TempDir())+string(filepath.Separator)) {
		t.Fatalf("project obligations under %s would disappear across temp cleanup or reboot", stateDirPath())
	}
}

func TestStateLossMarkerRequiresIndependentParent(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	if err := os.Mkdir(home, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name      string
		configDir string
		want      bool
	}{
		{"config below home", filepath.Join(home, ".config"), true},
		{"config beside home", filepath.Join(base, "config"), true},
		{"config equals home", home, false},
		{"config contains home", base, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := stateMarkerHasIndependentParent(home, tc.configDir); got != tc.want {
				t.Fatalf("stateMarkerHasIndependentParent(%q, %q)=%v want %v", home, tc.configDir, got, tc.want)
			}
		})
	}
}

func TestFirstStateInitializationIsSafeButSubsequentStoreLossBlocks(t *testing.T) {
	isolateHookState(t)
	root := managedRoot(t)

	if out := runEvent(t, root, Input{SessionID: "first-stop", HookEventName: "Stop"}); out != "" {
		t.Fatalf("first safe initialization changed an untouched Stop decision: %s", out)
	}
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		t.Fatal(err)
	}
	if present, err := validateStateInitializationMarker(marker); err != nil || !present {
		t.Fatalf("first initialization did not durably record store identity: present=%v err=%v", present, err)
	}

	target := filepath.Join(root, "design", "notes.txt")
	pre := editEvent("PreToolUse", "Write", "state-loss-session", target)
	pre.ToolUseID = "state-loss-tool"
	if out := runEvent(t, root, pre); out != "" {
		t.Fatalf("allowed PreToolUse unexpectedly emitted output: %s", out)
	}
	dir := stateDirPath()
	lost := dir + ".lost"
	if err := os.Rename(dir, lost); err != nil {
		t.Fatal(err)
	}
	secondPre := editEvent("PreToolUse", "Write", "replacement-session", target)
	secondPre.ToolUseID = "replacement-tool"
	if out := runEvent(t, root, secondPre); !strings.Contains(out, `"permissionDecision":"deny"`) || !strings.Contains(out, "missing after prior initialization") {
		t.Fatalf("PreToolUse recreated or ignored a missing durable store: %s", out)
	}
	if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("PreToolUse recreated a missing durable store as empty: %v", err)
	}

	out := runEvent(t, root, Input{SessionID: "replacement-session", HookEventName: "Stop"})
	if !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "missing after prior initialization") {
		t.Fatalf("missing durable store was treated as an untouched project: %s", out)
	}
	if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stop recreated a missing durable store as empty: %v", err)
	}
	unmanaged := t.TempDir()
	if out := runEvent(t, unmanaged, Input{SessionID: "unmanaged", HookEventName: "Stop"}); out != "" {
		t.Fatalf("loss of another project's state disturbed an unmanaged directory: %s", out)
	}
}

func TestManagedSubdirectoryFailsClosedAfterEntireStateParentLoss(t *testing.T) {
	isolateHookState(t)
	root := managedRoot(t)
	nested := filepath.Join(root, "nested", "work")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "design", "notes.txt")
	pre := editEvent("PreToolUse", "Write", "parent-loss-session", target)
	pre.ToolUseID = "parent-loss-tool"
	pre.Cwd = nested
	if out := runEvent(t, nested, pre); out != "" {
		t.Fatalf("allowed nested PreToolUse unexpectedly emitted output: %s", out)
	}

	dir := stateDirPath()
	configDir := filepath.Dir(dir)
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		t.Fatal(err)
	}
	if rel, err := filepath.Rel(configDir, marker); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("loss sentinel %s shares the state parent %s it must survive", marker, configDir)
	}
	lostParent := configDir + ".lost"
	if err := os.Rename(configDir, lostParent); err != nil {
		t.Fatal(err)
	}
	if present, err := validateStateInitializationMarker(marker); err != nil || !present {
		t.Fatalf("external loss sentinel did not survive config-parent loss: present=%v err=%v", present, err)
	}

	retry := editEvent("PreToolUse", "Write", "replacement-session", target)
	retry.ToolUseID = "replacement-parent-loss-tool"
	retry.Cwd = nested
	if out := runEvent(t, nested, retry); !strings.Contains(out, `"permissionDecision":"deny"`) || !strings.Contains(out, "parent directory") || !strings.Contains(out, "missing after prior initialization") {
		t.Fatalf("nested PreToolUse treated parent loss as first initialization: %s", out)
	}
	if _, err := os.Lstat(configDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("nested PreToolUse recreated the lost state parent: %v", err)
	}
	if out := runEvent(t, nested, Input{SessionID: "replacement-session", Cwd: nested, HookEventName: "Stop"}); !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "parent directory") || !strings.Contains(out, "missing after prior initialization") {
		t.Fatalf("nested Stop silently discharged after state-parent loss: %s", out)
	}
	if _, err := os.Lstat(configDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("nested Stop recreated the lost state parent: %v", err)
	}

	unmanaged := t.TempDir()
	if out := runEvent(t, unmanaged, Input{SessionID: "unmanaged", HookEventName: "Stop"}); out != "" {
		t.Fatalf("another project's parent loss disturbed an unmanaged directory: %s", out)
	}
}

func TestManagedSubdirectoryFailsClosedAfterStateDirectoryReplacement(t *testing.T) {
	isolateHookState(t)
	root := managedRoot(t)
	nested := filepath.Join(root, "nested", "work")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "design", "notes.txt")
	pre := editEvent("PreToolUse", "Write", "directory-replacement", target)
	pre.ToolUseID = "directory-replacement-first"
	pre.Cwd = nested
	if out := runEvent(t, nested, pre); out != "" {
		t.Fatalf("initial nested PreToolUse unexpectedly emitted output: %s", out)
	}
	dir := stateDirPath()
	parked := dir + ".parked"
	if err := os.Rename(dir, parked); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	retry := editEvent("PreToolUse", "Write", "directory-replacement", target)
	retry.ToolUseID = "directory-replacement-second"
	retry.Cwd = nested
	if out := runEvent(t, nested, retry); !strings.Contains(out, `"permissionDecision":"deny"`) || !strings.Contains(out, "bound identity") {
		t.Fatalf("nested PreToolUse accepted replacement state directory: %s", out)
	}
	if out := runEvent(t, nested, Input{SessionID: "directory-replacement", Cwd: nested, HookEventName: "Stop"}); !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "bound identity") {
		t.Fatalf("nested Stop discharged through replacement state directory: %s", out)
	}
	if err := clearState(root, "directory-replacement"); err == nil || !strings.Contains(err.Error(), "bound identity") {
		t.Fatalf("clear accepted replacement state directory: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("replacement directory was populated or removed: entries=%v err=%v", entries, err)
	}
	if _, err := os.Lstat(filepath.Join(parked, stateDirectoryIdentityName)); err != nil {
		t.Fatalf("original bound store was modified: %v", err)
	}
}

func TestCopiedStateIdentityCannotAuthorizeReplacementDirectory(t *testing.T) {
	isolateHookState(t)
	if err := ensureStateDir(); err != nil {
		t.Fatal(err)
	}
	dir := stateDirPath()
	identity, err := os.ReadFile(filepath.Join(dir, stateDirectoryIdentityName))
	if err != nil {
		t.Fatal(err)
	}
	parked := dir + ".parked"
	if err := os.Rename(dir, parked); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, stateDirectoryIdentityName), identity, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureStateDir(); err == nil || !strings.Contains(err.Error(), "changed native identity") {
		t.Fatalf("copied generation authorized a replacement directory: %v", err)
	}
}

func TestStateDirectoryReplacementDuringIdentityValidationFailsClosed(t *testing.T) {
	isolateHookState(t)
	if err := ensureStateDir(); err != nil {
		t.Fatal(err)
	}
	dir := stateDirPath()
	parked := dir + ".parked"
	prior := hookStateDirectoryPhase
	var once sync.Once
	var mutationErr error
	hookStateDirectoryPhase = func(phase, path string) {
		if phase == "after-open" && path == dir {
			once.Do(func() {
				if err := os.Rename(dir, parked); err != nil {
					mutationErr = err
					return
				}
				mutationErr = os.Mkdir(dir, 0o700)
			})
		}
	}
	t.Cleanup(func() { hookStateDirectoryPhase = prior })
	if err := ensureStateDir(); err == nil || !strings.Contains(err.Error(), "changed identity during validation") {
		t.Fatalf("directory replacement race was accepted: %v", err)
	}
	if mutationErr != nil {
		t.Fatal(mutationErr)
	}
}

func TestStateDirectoryReplacementAfterBindingValidationFailsClosed(t *testing.T) {
	for _, operation := range []string{"stop", "clear"} {
		t.Run(operation, func(t *testing.T) {
			isolateHookState(t)
			root := managedRoot(t)
			if err := appendState(root, "binding-race", "design"); err != nil {
				t.Fatal(err)
			}
			dir := stateDirPath()
			parked := dir + ".parked"
			prior := hookStateBindingPhase
			var once sync.Once
			var mutationErr error
			hookStateBindingPhase = func(phase, path string) {
				if phase == "validated" && path == dir {
					once.Do(func() {
						if err := os.Rename(dir, parked); err != nil {
							mutationErr = err
							return
						}
						mutationErr = os.Mkdir(dir, 0o700)
					})
				}
			}
			t.Cleanup(func() { hookStateBindingPhase = prior })
			switch operation {
			case "stop":
				out := runEvent(t, root, Input{SessionID: "binding-race", HookEventName: "Stop"})
				if !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "bound hook state directory identity") {
					t.Fatalf("Stop accepted directory replacement after binding validation: %s", out)
				}
			case "clear":
				if err := clearState(root, "binding-race"); err == nil || !strings.Contains(err.Error(), "bound hook state directory identity") {
					t.Fatalf("clear accepted directory replacement after binding validation: %v", err)
				}
			}
			if mutationErr != nil {
				t.Fatal(mutationErr)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("replacement directory changed: entries=%v err=%v", entries, err)
			}
			if _, err := os.Lstat(filepath.Join(parked, stateDirectoryIdentityName)); err != nil {
				t.Fatalf("original directory identity was lost: %v", err)
			}
		})
	}
}

func TestLegacySiblingStateMarkerMigratesOutsideStateParent(t *testing.T) {
	isolateHookState(t)
	dir, err := stateDirPathExact()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	legacy, err := legacyStateInitializationMarkerPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte(stateInitializationMarkerBody), 0o600); err != nil {
		t.Fatal(err)
	}
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		t.Fatal(err)
	}
	if present, err := validateStateInitializationMarker(marker); err != nil || present {
		t.Fatalf("external marker unexpectedly pre-existed: present=%v err=%v", present, err)
	}
	if err := ensureStateDir(); err != nil {
		t.Fatal(err)
	}
	if present, err := validateStateInitializationMarker(marker); err != nil || !present {
		t.Fatalf("legacy state marker did not migrate outside its state parent: present=%v err=%v", present, err)
	}
}

func TestConcurrentFirstStateInitializationProducesOneCanonicalMarker(t *testing.T) {
	isolateHookState(t)
	const workers = 24
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- ensureStateDir()
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent state initialization failed: %v", err)
		}
	}
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(marker)
	binding, parseErr := parseStateDirectoryBinding(body, "machinery-hook-state-v2")
	if err != nil || parseErr != nil || binding.native == "" || binding.generation == "" {
		t.Fatalf("state initialization marker is not canonical: %q err=%v", body, err)
	}
	info, err := os.Lstat(stateDirPath())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || (runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0) {
		t.Fatalf("concurrent initialization produced an unsafe store: mode=%v", info.Mode())
	}
}

func TestStateInitializationWaiterDoesNotReadIncompleteMarker(t *testing.T) {
	isolateHookState(t)
	previous := hookStateMarkerPhase
	t.Cleanup(func() { hookStateMarkerPhase = previous })
	created := make(chan struct{})
	resume := make(chan struct{})
	var once sync.Once
	hookStateMarkerPhase = func(phase string) {
		if phase != "created" {
			return
		}
		once.Do(func() {
			close(created)
			<-resume
		})
	}
	first := make(chan error, 1)
	go func() { first <- ensureStateDir() }()
	select {
	case <-created:
	case <-time.After(2 * time.Second):
		t.Fatal("marker creator did not reach the incomplete-file boundary")
	}
	second := make(chan error, 1)
	go func() { second <- ensureStateDir() }()
	select {
	case err := <-second:
		close(resume)
		t.Fatalf("waiter inspected the incomplete marker before acquiring the lock: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(resume)
	if err := <-first; err != nil {
		t.Fatalf("marker creator failed: %v", err)
	}
	if err := <-second; err != nil {
		t.Fatalf("marker waiter failed after creator durability: %v", err)
	}
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		t.Fatal(err)
	}
	if present, err := validateStateInitializationMarker(marker); err != nil || !present {
		t.Fatalf("concurrent initialization did not leave one canonical marker: present=%v err=%v", present, err)
	}
}

func TestProjectObligationKeepsInterleavedSessionRoutesDistinct(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(func() { _ = clearState(root, "cleanup") })
	if err := appendState(root, "session-a", "design"); err != nil {
		t.Fatal(err)
	}
	if err := writeRouteSnapshot(root, "session-a", []byte(`{"design":"design","strict":false}`+"\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeRouteSnapshot(root, "session-b", []byte(`{"design":"blueprint","strict":true}`+"\n")); err != nil {
		t.Fatal(err)
	}
	if design, _, err := readStateErr(root, "replacement"); err != nil || !design {
		t.Fatalf("dirty obligation was not project-scoped: design=%v err=%v", design, err)
	}
	first, present, err := loadRouteSnapshot(root, "session-a")
	if err != nil || !present || first.Design != "design" || first.Strict {
		t.Fatalf("session-a route was overwritten: cfg=%+v present=%v err=%v", first, present, err)
	}
	second, present, err := loadRouteSnapshot(root, "session-b")
	if err != nil || !present || second.Design != "blueprint" || !second.Strict {
		t.Fatalf("session-b route was overwritten: cfg=%+v present=%v err=%v", second, present, err)
	}
	if _, _, err := loadRouteSnapshot(root, "replacement"); err == nil || !strings.Contains(err.Error(), "conflicting") {
		t.Fatalf("a replacement session guessed between conflicting route revisions: %v", err)
	}
	if err := clearState(root, "session-b"); err != nil {
		t.Fatal(err)
	}
	if paths, err := routeStatePaths(root); err != nil || len(paths) != 0 {
		t.Fatalf("successful project discharge retained route residue: paths=%v err=%v", paths, err)
	}
}

func TestHookStateReadsRejectPathAndContentRaces(t *testing.T) {
	t.Run("route regular becomes symlink", func(t *testing.T) {
		isolateHookState(t)
		root := managedRoot(t)
		if err := appendState(root, "route-swap", "design"); err != nil {
			t.Fatal(err)
		}
		body, err := routeSnapshotBody(Config{Design: "design"})
		if err != nil {
			t.Fatal(err)
		}
		if err := writeRouteSnapshot(root, "route-swap", body); err != nil {
			t.Fatal(err)
		}
		path := routeStatePath(root, "route-swap")
		parked := path + ".parked"
		foreign := path + ".foreign"
		if err := os.WriteFile(foreign, body, 0o600); err != nil {
			t.Fatal(err)
		}
		mutateHookStateReadOnce(t, "after-lstat", path, func() error {
			if err := os.Rename(path, parked); err != nil {
				return err
			}
			return os.Symlink(filepath.Base(foreign), path)
		})
		if _, _, err := loadRouteSnapshot(root, "route-swap"); err == nil {
			t.Fatal("route path replaced by a symlink was accepted")
		}
		if got, err := os.ReadFile(foreign); err != nil || !bytes.Equal(got, body) {
			t.Fatalf("foreign route was modified: %q err=%v", got, err)
		}
	})

	t.Run("update path ABA", func(t *testing.T) {
		isolateHookState(t)
		root := t.TempDir()
		if err := appendState(root, "update-aba", "design"); err != nil {
			t.Fatal(err)
		}
		path := statePath(root, "update-aba")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		parked := path + ".parked"
		mutateHookStateReadOnce(t, "after-lstat", path, func() error {
			return replacePathPreservingOriginal(path, parked, body)
		})
		if err := appendState(root, "update-aba", "impl"); err == nil || !strings.Contains(err.Error(), "changed identity") {
			t.Fatalf("state path ABA was accepted by update: %v", err)
		}
		if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, body) {
			t.Fatalf("failed update overwrote replacement: %q err=%v", got, err)
		}
	})

	t.Run("stop content ABA", func(t *testing.T) {
		isolateHookState(t)
		root := managedRoot(t)
		if err := appendState(root, "stop-aba", "design"); err != nil {
			t.Fatal(err)
		}
		path := statePath(root, "stop-aba")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		mutateHookStateReadOnce(t, "after-open", path, func() error {
			changed := append([]byte(nil), body...)
			changed[len(changed)-2] ^= 1
			if err := os.WriteFile(path, changed, 0o600); err != nil {
				return err
			}
			return os.WriteFile(path, body, 0o600)
		})
		out := runEvent(t, root, Input{SessionID: "stop-aba", HookEventName: "Stop"})
		if !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "changed identity, metadata, or content") {
			t.Fatalf("Stop accepted a content ABA during state read: %s", out)
		}
	})

	t.Run("clear path ABA", func(t *testing.T) {
		isolateHookState(t)
		root := t.TempDir()
		if err := appendState(root, "clear-aba", "design"); err != nil {
			t.Fatal(err)
		}
		body, err := routeSnapshotBody(Config{Design: "design"})
		if err != nil {
			t.Fatal(err)
		}
		if err := writeRouteSnapshot(root, "clear-aba", body); err != nil {
			t.Fatal(err)
		}
		route := routeStatePath(root, "clear-aba")
		parked := route + ".parked"
		mutateHookStateReadOnce(t, "before-remove", route, func() error {
			return replacePathPreservingOriginal(route, parked, body)
		})
		if err := clearState(root, "clear-aba"); err == nil || !strings.Contains(err.Error(), "preserving it") {
			t.Fatalf("clear accepted a route path ABA: %v", err)
		}
		if _, err := os.Lstat(statePath(root, "clear-aba")); err != nil {
			t.Fatalf("clear discharged dirty ledger after route race: %v", err)
		}
		if got, err := os.ReadFile(route); err != nil || !bytes.Equal(got, body) {
			t.Fatalf("clear deleted route replacement: %q err=%v", got, err)
		}
	})

	t.Run("marker regular becomes symlink", func(t *testing.T) {
		isolateHookState(t)
		if err := ensureStateDir(); err != nil {
			t.Fatal(err)
		}
		marker, err := stateInitializationMarkerPath()
		if err != nil {
			t.Fatal(err)
		}
		parked := marker + ".parked"
		foreign := marker + ".foreign"
		if err := os.WriteFile(foreign, []byte(stateInitializationMarkerBody), 0o600); err != nil {
			t.Fatal(err)
		}
		mutateHookStateReadOnce(t, "after-lstat", marker, func() error {
			if err := os.Rename(marker, parked); err != nil {
				return err
			}
			return os.Symlink(filepath.Base(foreign), marker)
		})
		if err := ensureStateDir(); err == nil {
			t.Fatal("symlink-swapped initialization marker was accepted")
		}
		if got, err := os.ReadFile(foreign); err != nil || string(got) != stateInitializationMarkerBody {
			t.Fatalf("foreign marker was modified: %q err=%v", got, err)
		}
	})
}

func TestHookStateReadsRejectOversizeSparseFiles(t *testing.T) {
	t.Run("state update stop and clear", func(t *testing.T) {
		isolateHookState(t)
		root := managedRoot(t)
		if err := ensureStateDir(); err != nil {
			t.Fatal(err)
		}
		path := statePath(root, "oversize-state")
		writeSparseFile(t, path, hookStateMaxBytes+1)
		want := fmt.Sprintf("exceeds %d-byte limit", hookStateMaxBytes)
		if err := appendState(root, "oversize-state", "design"); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("update accepted oversized state: %v", err)
		}
		out := runEvent(t, root, Input{SessionID: "oversize-state", HookEventName: "Stop"})
		if !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, want) {
			t.Fatalf("Stop accepted oversized state: %s", out)
		}
		if err := clearState(root, "oversize-state"); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("clear accepted oversized state: %v", err)
		}
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("oversized state was deleted: %v", err)
		}
	})

	t.Run("route load and clear", func(t *testing.T) {
		isolateHookState(t)
		root := t.TempDir()
		if err := appendState(root, "oversize-route", "design"); err != nil {
			t.Fatal(err)
		}
		path := routeStatePath(root, "oversize-route")
		writeSparseFile(t, path, hookRouteMaxBytes+1)
		want := fmt.Sprintf("exceeds %d-byte limit", hookRouteMaxBytes)
		if _, _, err := loadRouteSnapshot(root, "oversize-route"); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("route load accepted oversized route: %v", err)
		}
		if err := clearState(root, "oversize-route"); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("clear accepted oversized route: %v", err)
		}
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("oversized route was deleted: %v", err)
		}
		if _, err := os.Lstat(statePath(root, "oversize-route")); err != nil {
			t.Fatalf("dirty ledger was discharged after route rejection: %v", err)
		}
	})

	t.Run("initialization marker", func(t *testing.T) {
		isolateHookState(t)
		dir, err := stateDirPathExact()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
			t.Fatal(err)
		}
		marker, err := stateInitializationMarkerPath()
		if err != nil {
			t.Fatal(err)
		}
		writeSparseFile(t, marker, hookStateMarkerMaxBytes+1)
		want := fmt.Sprintf("exceeds %d-byte limit", hookStateMarkerMaxBytes)
		if err := ensureStateDir(); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("initialization accepted oversized marker: %v", err)
		}
		if _, err := os.Lstat(marker); err != nil {
			t.Fatalf("oversized marker was deleted: %v", err)
		}
	})
}

func TestGreenStopCannotClearNewerProjectRevision(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(func() { _ = clearState(root, "cleanup") })
	if err := appendState(root, "first", "design"); err != nil {
		t.Fatal(err)
	}
	checked, err := readStateRecord(root, "first")
	if err != nil {
		t.Fatal(err)
	}
	if err := appendState(root, "second", "design"); err != nil {
		t.Fatal(err)
	}
	if err := clearCheckedState(root, "first", checked.revision); err == nil || !strings.Contains(err.Error(), "new governed edits arrived") {
		t.Fatalf("stale green check cleared a newer project revision: %v", err)
	}
	if state, err := readStateRecord(root, "replacement"); err != nil || !state.design || state.revision <= checked.revision {
		t.Fatalf("newer obligation was not retained: %+v err=%v", state, err)
	}
}

func TestRouteWithoutDirtyLedgerBlocksStop(t *testing.T) {
	root := managedRoot(t)
	raw, err := routeSnapshotBody(Config{Design: "design"})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRouteSnapshot(root, "orphan-route", raw); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = clearState(root, "cleanup") })
	out := runEvent(t, root, Input{SessionID: "orphan-route", HookEventName: "Stop"})
	if !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "without the project dirty ledger") {
		t.Fatalf("orphan route snapshot allowed a nil stop: %s", out)
	}
}

func TestRouteTempCrashBlocksWritersAndStop(t *testing.T) {
	if os.Getenv("MACHINERY_HOOK_ROUTE_CRASH_CHILD") == "1" {
		hookRoutePhase = func(phase string) {
			if phase == "temp-synced" {
				os.Exit(98)
			}
		}
		root := os.Getenv("MACHINERY_HOOK_ROUTE_CRASH_ROOT")
		in := editEvent("PreToolUse", "Write", "route-crash", filepath.Join(root, "design", "note.txt"))
		in.ToolUseID = "route-crash-operation"
		raw, _ := json.Marshal(in)
		_ = Run(bytes.NewReader(raw), io.Discard, root)
		os.Exit(0)
	}
	root := managedRoot(t)
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestRouteTempCrashBlocksWritersAndStop$")
	cmd.Env = append(os.Environ(), "MACHINERY_HOOK_ROUTE_CRASH_CHILD=1", "MACHINERY_HOOK_ROUTE_CRASH_ROOT="+root)
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 98 {
		t.Fatalf("route crash child exit = %v, want 98", err)
	}
	temps, err := routeStateTemps(root)
	if err != nil || len(temps) != 1 {
		t.Fatalf("route crash did not leave one durable temp: %v err=%v", temps, err)
	}
	stopRaw, _ := json.Marshal(Input{SessionID: "replacement", HookEventName: "Stop"})
	var stopOutput bytes.Buffer
	if err := Run(bytes.NewReader(stopRaw), &stopOutput, root); err != nil || !strings.Contains(stopOutput.String(), `"decision":"block"`) || !strings.Contains(stopOutput.String(), "incomplete hook route transaction") {
		t.Fatalf("Stop ignored route crash residue: output=%s err=%v", stopOutput.String(), err)
	}
	pre := editEvent("PreToolUse", "Write", "new-writer", filepath.Join(root, "design", "other.txt"))
	pre.ToolUseID = "new-writer-operation"
	out := runEvent(t, root, pre)
	if !strings.Contains(out, `"permissionDecision":"deny"`) || !strings.Contains(out, "route transaction") {
		t.Fatalf("new writer overwrote route crash residue: %s", out)
	}
	for _, temp := range temps {
		if err := os.Remove(temp); err != nil {
			t.Fatal(err)
		}
	}
	if err := clearState(root, "cleanup"); err != nil {
		t.Fatal(err)
	}
}

func TestManagedEventsFailClosedWithoutAbsolutePersistentStateRoot(t *testing.T) {
	root := managedRoot(t)
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	pre := editEvent("PreToolUse", "Write", "no-home", filepath.Join(root, "design", "note.txt"))
	pre.ToolUseID = "no-home-operation"
	if out := runEvent(t, root, pre); !strings.Contains(out, `"permissionDecision":"deny"`) || !strings.Contains(out, "durable project-obligation store") {
		t.Fatalf("PreToolUse guessed a relative state path without HOME: %s", out)
	}
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	if err := os.Chdir(other); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldCwd) }()
	if out := runEvent(t, root, Input{SessionID: "replacement", HookEventName: "Stop"}); !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "resolve absolute user home") {
		t.Fatalf("Stop changed state identity with cwd or treated missing HOME as untouched: %s", out)
	}
}

func TestStateDirectoryIsPrivateAndSymlinkLedgerFailsClosed(t *testing.T) {
	if err := ensureStateDir(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(stateDirPath())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || (runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0) {
		t.Fatalf("hook state directory is not private and real: mode=%v", info.Mode())
	}
	root := t.TempDir()
	sid := "symlink-ledger"
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, statePath(root, sid)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(statePath(root, sid)) })
	if err := appendState(root, sid, "design"); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlinked hook state accepted: %v", err)
	}
	got, err := os.ReadFile(outside)
	if err != nil || string(got) != "sentinel" {
		t.Fatalf("symlink target changed: %q, %v", got, err)
	}
}

func TestConcurrentStateUpdatesPreserveEveryTouchClass(t *testing.T) {
	root := t.TempDir()
	sid := "parallel-ledger"
	clearState(root, sid)
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		kind := "design"
		if i%2 == 1 {
			kind = "impl"
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- appendState(root, sid, kind)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent state update failed: %v", err)
		}
	}
	design, impl, err := readStateErr(root, sid)
	if err != nil || !design || !impl {
		t.Fatalf("parallel touches were lost: design=%v impl=%v err=%v", design, impl, err)
	}
	clearState(root, sid)
}

func TestHookStateSchemaRejectsEveryNoncanonicalExistingLedger(t *testing.T) {
	for _, raw := range []string{"", "\n", "design", "design\n\n", "impl\ndesign\n", "design\ndesign\n", "unknown\n"} {
		t.Run(fmt.Sprintf("%q", raw), func(t *testing.T) {
			root := t.TempDir()
			sid := "corrupt-schema"
			if err := ensureStateDir(); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(statePath(root, sid), []byte(raw), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := readStateErr(root, sid); err == nil || !strings.Contains(err.Error(), "corrupt or noncanonical") {
				t.Fatalf("noncanonical ledger %q was accepted: %v", raw, err)
			}
		})
	}
}

func TestStopBlocksOnCorruptExistingStateLedger(t *testing.T) {
	root := managedRoot(t)
	sid := "corrupt-stop"
	if err := ensureStateDir(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath(root, sid), []byte("design\nunknown\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision != "block" || !strings.Contains(got.Reason, "touched-file state") || !strings.Contains(got.Reason, "noncanonical") {
		t.Fatalf("corrupt state ledger allowed Stop to skip gates: %+v", got)
	}
}

func TestAppendStatePreservesPriorLedgerOnReplaceFailure(t *testing.T) {
	root := t.TempDir()
	sid := "replace-failure"
	if err := appendState(root, sid, "design"); err != nil {
		t.Fatal(err)
	}
	prior := replaceHookStateFile
	replaceHookStateFile = func(_, _ string) error { return errors.New("injected durable replace failure") }
	t.Cleanup(func() { replaceHookStateFile = prior })
	if err := appendState(root, sid, "impl"); err == nil || !strings.Contains(err.Error(), "injected durable replace failure") {
		t.Fatalf("replace failure was not propagated: %v", err)
	}
	design, impl, err := readStateErr(root, sid)
	if err != nil || !design || impl {
		t.Fatalf("failed replace changed prior ledger: design=%v impl=%v err=%v", design, impl, err)
	}
}

func TestReplaceStateFileNativeReplacement(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "session.state")
	temp := filepath.Join(dir, "session.state.new")
	writeFile(t, target, "old\n")
	writeFile(t, temp, "new\n")
	if err := replaceStateFile(temp, target); err != nil {
		t.Fatalf("native durable replacement failed: %v", err)
	}
	body, err := os.ReadFile(target)
	if err != nil || string(body) != "new\n" {
		t.Fatalf("replacement content = %q, %v", body, err)
	}
	if _, err := os.Lstat(temp); !os.IsNotExist(err) {
		t.Fatalf("replacement left source temp behind: %v", err)
	}
}

func TestAppendStatePropagatesLockReleaseFailure(t *testing.T) {
	root := t.TempDir()
	prior := acquireHookStateLock
	acquireHookStateLock = func(string) (*stateLock, error) {
		return &stateLock{releaseFn: func() error { return errors.New("injected state unlock failure") }}, nil
	}
	t.Cleanup(func() { acquireHookStateLock = prior })
	if err := appendState(root, "unlock-failure", "design"); err == nil || !strings.Contains(err.Error(), "injected state unlock failure") {
		t.Fatalf("lock release failure was discarded: %v", err)
	}
}

func TestAppendStateCrashBeforeReplacePreservesPriorLedger(t *testing.T) {
	if os.Getenv("MACHINERY_HOOK_STATE_CRASH_CHILD") == "1" {
		hookStatePhase = func(phase string) {
			if phase == "temp-synced" {
				os.Exit(97)
			}
		}
		_ = appendState(os.Getenv("MACHINERY_HOOK_STATE_CRASH_ROOT"), "phase-crash", "impl")
		os.Exit(0)
	}
	root := t.TempDir()
	if err := appendState(root, "phase-crash", "design"); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestAppendStateCrashBeforeReplacePreservesPriorLedger$")
	cmd.Env = append(os.Environ(),
		"MACHINERY_HOOK_STATE_CRASH_CHILD=1",
		"MACHINERY_HOOK_STATE_CRASH_ROOT="+root,
	)
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 97 {
		t.Fatalf("crash child exit = %v, want 97", err)
	}
	design, impl, readErr := readStateErr(root, "phase-crash")
	if readErr == nil || !strings.Contains(readErr.Error(), "incomplete hook state transaction") || design || impl {
		t.Fatalf("pre-replace crash evidence was ignored: design=%v impl=%v err=%v", design, impl, readErr)
	}
	stateFile := statePath(root, "phase-crash")
	if err := cleanupHookStateTemps(stateFile); err != nil {
		t.Fatalf("recover crash temp: %v", err)
	}
	left, err := filepath.Glob(filepath.Join(filepath.Dir(stateFile), "."+filepath.Base(stateFile)+".tmp-*"))
	if err != nil || len(left) != 0 {
		t.Fatalf("crash temp recovery left %v: %v", left, err)
	}
	design, impl, readErr = readStateErr(root, "phase-crash")
	if readErr != nil || !design || impl {
		t.Fatalf("recovering the temp did not preserve prior canonical state: design=%v impl=%v err=%v", design, impl, readErr)
	}
}

func TestFirstAppendCrashLeavesDurableEvidenceThatBlocksStop(t *testing.T) {
	if os.Getenv("MACHINERY_HOOK_FIRST_STATE_CRASH_CHILD") == "1" {
		hookStatePhase = func(phase string) {
			if phase == "temp-synced" {
				os.Exit(98)
			}
		}
		_ = appendState(os.Getenv("MACHINERY_HOOK_FIRST_STATE_CRASH_ROOT"), "first-phase-crash", "design")
		os.Exit(0)
	}
	root := managedRoot(t)
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestFirstAppendCrashLeavesDurableEvidenceThatBlocksStop$")
	cmd.Env = append(os.Environ(),
		"MACHINERY_HOOK_FIRST_STATE_CRASH_CHILD=1",
		"MACHINERY_HOOK_FIRST_STATE_CRASH_ROOT="+root,
	)
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 98 {
		t.Fatalf("first-write crash child exit = %v, want 98", err)
	}
	out := runEvent(t, root, Input{HookEventName: "Stop", SessionID: "first-phase-crash"})
	var got stopOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stop output is not JSON: %v (%q)", err, out)
	}
	if got.Decision != "block" || !strings.Contains(got.Reason, "incomplete hook state transaction") || !strings.Contains(got.Reason, "untouched") {
		t.Fatalf("first-write crash evidence did not block Stop: %+v", got)
	}
	if err := cleanupHookStateTemps(statePath(root, "first-phase-crash")); err != nil {
		t.Fatal(err)
	}
}

func TestStatePathCanonicalizesSymlinkRoot(t *testing.T) {
	real := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(real, alias); err != nil {
		t.Fatal(err)
	}
	if statePath(real, "s1") != statePath(alias, "s1") {
		t.Fatal("a symlink alias of the same project must share hook state")
	}
}

func TestStatePathCanonicalizesNativeCaseAliases(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("case-alias contract applies to Darwin and Windows")
	}
	root := t.TempDir()
	if statePath(root, "s1") != statePath(strings.ToUpper(root), "s1") {
		t.Fatal("native case aliases of one project must share hook state")
	}
}

func TestRelativeEditPathUsesEventCwd(t *testing.T) {
	root := managedRoot(t)
	sub := filepath.Join(root, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	in := editEvent("PreToolUse", "Write", "s-cwd", filepath.Join("..", "design", "machines", "Order.oracle.md"))
	in.Cwd = sub
	out := runEvent(t, root, in)
	var got preOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("deny output is not JSON: %v (%q)", err, out)
	}
	if got.HookSpecificOutput.PermissionDecision != "deny" {
		t.Fatalf("event-cwd-relative generated artifact edit escaped governance: %+v", got)
	}
}

func TestPostStateWriteFailureFailsClosed(t *testing.T) {
	root := managedRoot(t)
	sid := "s-state-write-failure"
	if err := ensureStateDir(); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(statePath(root, sid), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(statePath(root, sid)) })
	in := editEvent("PostToolUse", "Write", sid, filepath.Join(root, "design", "domain.modelith.yaml"))
	in.ToolUseID = "state-write-failure"
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = Run(bytes.NewReader(raw), &out, root)
	if err == nil || !strings.Contains(err.Error(), "durable tool tracking") {
		t.Fatalf("Run error = %v, want fail-closed state diagnostic", err)
	}
}

// --- the plugin wiring itself: a regression net over the shipped files ---

func repoPath(parts ...string) string {
	return filepath.Join(append([]string{"..", ".."}, parts...)...)
}

func TestPluginHooksJSONWiring(t *testing.T) {
	raw, err := os.ReadFile(repoPath("hooks", "hooks.json"))
	if err != nil {
		t.Fatalf("hooks/hooks.json must ship with the plugin: %v", err)
	}
	var doc struct {
		Description string `json:"description"`
		Hooks       map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
				Timeout int    `json:"timeout"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("hooks.json does not parse: %v", err)
	}
	if len(doc.Hooks) == 0 {
		t.Fatal("hooks.json must use the plugin wrapper format with a hooks key")
	}
	for _, ev := range []string{"PreToolUse", "PostToolUse", "Stop", "SubagentStop", "SessionStart"} {
		entries, ok := doc.Hooks[ev]
		if !ok || len(entries) == 0 {
			t.Fatalf("hooks.json missing event %s", ev)
		}
		for _, e := range entries {
			if (ev == "PreToolUse" || ev == "PostToolUse" || ev == "PostToolUseFailure") && !strings.Contains(e.Matcher, "apply_patch") {
				t.Fatalf("%s matcher must include the Codex apply_patch tool, got %q", ev, e.Matcher)
			}
			if (ev == "PreToolUse" || ev == "PostToolUse" || ev == "PostToolUseFailure") && !strings.Contains(e.Matcher, "Bash") {
				t.Fatalf("%s matcher must include shell tools, got %q", ev, e.Matcher)
			}
			for _, h := range e.Hooks {
				if h.Type != "command" {
					t.Fatalf("%s: only command hooks are shipped, got %q", ev, h.Type)
				}
				if h.Command != "${CLAUDE_PLUGIN_ROOT}/hooks/machinery-hook.sh" {
					t.Fatalf("%s: every hook must route through the shim, got %q", ev, h.Command)
				}
				if h.Timeout <= 0 {
					t.Fatalf("%s: hooks must carry an explicit timeout", ev)
				}
			}
		}
	}
	fi, err := os.Stat(repoPath("hooks", "machinery-hook.sh"))
	if err != nil {
		t.Fatalf("the shim must exist: %v", err)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Fatal("hooks/machinery-hook.sh must be executable")
	}
}

func TestPluginManifests(t *testing.T) {
	var plugin struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Skills  string `json:"skills"`
	}
	raw, err := os.ReadFile(repoPath(".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &plugin); err != nil {
		t.Fatalf("plugin.json does not parse: %v", err)
	}
	if plugin.Name != "machinery" || plugin.Version == "" {
		t.Fatalf("plugin.json must name and version the plugin, got %+v", plugin)
	}
	claudeVersion := plugin.Version

	raw, err = os.ReadFile(repoPath(".codex-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &plugin); err != nil {
		t.Fatalf("Codex plugin.json does not parse: %v", err)
	}
	if plugin.Name != "machinery" || plugin.Version != claudeVersion || plugin.Skills != "./skills/" {
		t.Fatalf("Codex manifest must reuse the shared skill and match the Claude version, got %+v", plugin)
	}
	skillRaw, err := os.ReadFile(repoPath("skills", "machinery", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(skillRaw), "version: \""+claudeVersion+"\"") {
		t.Fatalf("plugin version %s and skill metadata version diverge", claudeVersion)
	}

	var mkt struct {
		Name    string `json:"name"`
		Plugins []struct {
			Name   string `json:"name"`
			Source string `json:"source"`
		} `json:"plugins"`
	}
	raw, err = os.ReadFile(repoPath(".claude-plugin", "marketplace.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &mkt); err != nil {
		t.Fatalf("marketplace.json does not parse: %v", err)
	}
	if len(mkt.Plugins) != 1 || mkt.Plugins[0].Name != "machinery" || mkt.Plugins[0].Source != "./" {
		t.Fatalf("marketplace must list the repo root as the machinery plugin, got %+v", mkt)
	}

	// the plugin reuses the repo's own skill, agents, and commands
	for _, p := range [][]string{
		{"skills", "machinery", "SKILL.md"},
		{"agents", "machinery-fsm-author.md"},
		{"agents", "machinery-build-writer.md"},
		{"commands", "design.md"},
		{"commands", "check.md"},
		{"commands", "init.md"},
		{"commands", "status.md"},
	} {
		if _, err := os.Stat(repoPath(p...)); err != nil {
			t.Fatalf("plugin component missing: %s", filepath.Join(p...))
		}
	}
}

// TestShimNoopContract documents the shim's stdin-independence: for an
// unmanaged root the shim must exit before it ever reads stdin or looks for
// the binary. Exercised here by running the shim when sh is available.
func TestShimNoopContract(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no /bin/sh")
	}
	root := t.TempDir()
	out, errOut, code := runShim(t, root, `{"hook_event_name":"Stop"}`)
	if code != 0 || out != "" || errOut != "" {
		t.Fatalf("unmanaged root: shim must be a silent no-op, got code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestShimWithUnsetHomeIsSilentWhenUnmanagedAndBlocksManagedAncestor(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no /bin/sh")
	}
	shim, err := filepath.Abs(repoPath("hooks", "machinery-hook.sh"))
	if err != nil {
		t.Fatal(err)
	}
	run := func(t *testing.T, dir string) (string, error) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "/bin/sh", shim)
		cmd.Dir = dir
		cmd.Env = []string{"PATH=" + t.TempDir()}
		cmd.Stdin = strings.NewReader(`{"hook_event_name":"Stop"}`)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	unmanaged := t.TempDir()
	if out, err := run(t, unmanaged); err != nil || out != "" {
		t.Fatalf("unset HOME disturbed an unmanaged project: err=%v output=%q", err, out)
	}

	managed := t.TempDir()
	writeFile(t, filepath.Join(managed, ConfigName), `{}`)
	nested := filepath.Join(managed, "nested", "work")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, nested)
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
		t.Fatalf("unset HOME bypassed a managed ancestor: err=%v output=%q", err, out)
	}
	if !strings.Contains(out, "BLOCKED") || !strings.Contains(out, "binary is unavailable") {
		t.Fatalf("unset-HOME managed diagnostic is not actionable: %q", out)
	}
}

func TestShimFailsClosedWhenBinaryMissing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{}`)
	shim, err := filepath.Abs(repoPath("hooks", "machinery-hook.sh"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "/bin/sh", shim)
	cmd.Dir = root
	cmd.Env = []string{"CLAUDE_PROJECT_DIR=" + root, "HOME=" + t.TempDir(), "PATH=/usr/bin:/bin"}
	cmd.Stdin = strings.NewReader(`{"hook_event_name":"Stop"}`)
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
		t.Fatalf("missing binary must exit 2, err=%v output=%s", err, out)
	}
	if !strings.Contains(string(out), "BLOCKED") || !strings.Contains(string(out), "binary is unavailable") {
		t.Fatalf("missing-binary diagnostic is not actionable: %s", out)
	}
}

func TestShimDispatchesEveryPresentMarkerToGoValidation(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no /bin/sh")
	}
	for _, tc := range []struct {
		name  string
		rel   string
		setup func(*testing.T, string) func()
	}{
		{"config directory", ConfigName, func(t *testing.T, marker string) func() {
			if err := os.Mkdir(marker, 0o755); err != nil {
				t.Fatal(err)
			}
			return func() {}
		}},
		{"config symlink", ConfigName, func(t *testing.T, marker string) func() {
			outside := filepath.Join(t.TempDir(), "config.json")
			writeFile(t, outside, `{}`)
			if err := os.Symlink(outside, marker); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			return func() {}
		}},
		{"dangling config symlink", ConfigName, func(t *testing.T, marker string) func() {
			if err := os.Symlink(filepath.Join(t.TempDir(), "absent"), marker); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			return func() {}
		}},
		{"config special file", ConfigName, func(t *testing.T, marker string) func() {
			if runtime.GOOS == "windows" {
				t.Skip("Unix-domain socket marker is unavailable on Windows")
			}
			var listenConfig net.ListenConfig
			listener, err := listenConfig.Listen(t.Context(), "unix", marker)
			if err != nil {
				t.Skipf("Unix-domain sockets unavailable: %v", err)
			}
			return func() { _ = listener.Close() }
		}},
		{"conventional model directory", filepath.Join("design", "domain.modelith.yaml"), func(t *testing.T, marker string) func() {
			if err := os.MkdirAll(marker, 0o755); err != nil {
				t.Fatal(err)
			}
			return func() {}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			marker := filepath.Join(root, tc.rel)
			if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
				t.Fatal(err)
			}
			cleanup := tc.setup(t, marker)
			defer cleanup()

			binDir := t.TempDir()
			fake := filepath.Join(binDir, "machinery")
			if err := os.WriteFile(fake, []byte("#!/bin/sh\nprintf '%s' \"$*\"\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			shim, err := filepath.Abs(repoPath("hooks", "machinery-hook.sh"))
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.CommandContext(t.Context(), "/bin/sh", shim)
			cmd.Dir = root
			cmd.Env = []string{
				"CLAUDE_PROJECT_DIR=" + root,
				"HOME=" + t.TempDir(),
				"PATH=" + binDir + string(os.PathListSeparator) + "/usr/bin:/bin",
			}
			cmd.Stdin = strings.NewReader(`{"hook_event_name":"Stop"}`)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("present invalid marker was not dispatched: %v: %s", err, out)
			}
			want := "hook --root " + root
			if string(out) != want {
				t.Fatalf("shim invocation = %q, want %q", out, want)
			}
		})
	}
}

func TestShimDelegatesSubdirectoryRootDiscoveryToGo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{}`)
	subdir := filepath.Join(root, "nested", "work")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	fake := filepath.Join(binDir, "machinery")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nprintf '%s' \"$*\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	shim, err := filepath.Abs(repoPath("hooks", "machinery-hook.sh"))
	if err != nil {
		t.Fatal(err)
	}
	hookCmd := exec.CommandContext(t.Context(), "/bin/sh", shim)
	hookCmd.Dir = subdir
	hookCmd.Env = withoutEnv(os.Environ(), "CLAUDE_PROJECT_DIR")
	hookCmd.Env = append(hookCmd.Env, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	hookCmd.Stdin = strings.NewReader(`{"hook_event_name":"SessionStart"}`)
	out, err := hookCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shim: %v: %s", err, out)
	}
	realSubdir, err := filepath.EvalSymlinks(subdir)
	if err != nil {
		t.Fatal(err)
	}
	want := "hook --root " + realSubdir
	if string(out) != want {
		t.Fatalf("shim invocation = %q, want %q", out, want)
	}
	shimBody, err := os.ReadFile(shim)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(shimBody, []byte("git -C")) || bytes.Contains(shimBody, []byte("rev-parse")) {
		t.Fatal("shell shim must not use ambient Git for governance routing")
	}
}

func TestShimBlocksMissingBinaryFromManagedSubdirectory(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no /bin/sh")
	}
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{}`)
	subdir := filepath.Join(root, "nested", "work")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	shim, err := filepath.Abs(repoPath("hooks", "machinery-hook.sh"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(t.Context(), "/bin/sh", shim)
	command.Dir = subdir
	command.Env = []string{"HOME=" + t.TempDir(), "PATH=/usr/bin:/bin"}
	command.Stdin = strings.NewReader(`{"hook_event_name":"Stop"}`)
	out, err := command.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
		t.Fatalf("missing binary under a managed ancestor must exit 2: err=%v output=%s", err, out)
	}
	if !strings.Contains(string(out), "BLOCKED") || !strings.Contains(string(out), "binary is unavailable") {
		t.Fatalf("missing-binary ancestor diagnostic is not actionable: %s", out)
	}
}

func TestCanonicalHookRootFindsManagedParentWithoutAmbientGit(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{}`)
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(root, "nested", "work")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		env  map[string]string
	}{
		{"git absent", map[string]string{"PATH": t.TempDir()}},
		{"git operational failure", map[string]string{"PATH": makeFailingGitPath(t)}},
		{"injected repository", map[string]string{"GIT_DIR": filepath.Join(t.TempDir(), ".git"), "GIT_WORK_TREE": t.TempDir()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			got, err := canonicalHookRoot(subdir)
			if err != nil {
				t.Fatal(err)
			}
			if got != realRoot {
				t.Fatalf("canonical root = %q, want managed parent %q", got, realRoot)
			}
		})
	}
}

func makeFailingGitPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	name := "git"
	body := []byte("#!/bin/sh\nexit 128\n")
	if runtime.GOOS == "windows" {
		name = "git.bat"
		body = []byte("@exit /b 128\r\n")
	}
	if err := os.WriteFile(filepath.Join(dir, name), body, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func withoutEnv(env []string, key string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env))
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			out = append(out, entry)
		}
	}
	return out
}

func runShim(t *testing.T, projectDir, stdin string) (stdout, stderr string, code int) {
	t.Helper()
	shim, err := filepath.Abs(repoPath("hooks", "machinery-hook.sh"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "/bin/sh", shim)
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
	cmd.Stdin = strings.NewReader(stdin)
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	err = cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
	default:
		t.Fatalf("shim: %v", err)
	}
	return so.String(), se.String(), code
}

// The stop-time selection matches `machinery check`'s default activation for
// Ga: neither artifact, no gate; the acceptance directory or a milestone
// marked closed, gate.
func TestSelectGatesActivatesGaOnAcceptanceArtifacts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "BUILD.md"),
		"# B\n\nMode: full\n\n## Build plan\n\n**M0 - Walking skeleton.** DoD: T-X-01 green.\n")
	sel, _ := selectGates(dir, Config{})
	if sel.Run["ga"] {
		t.Error("ga must not run before a milestone is closed or evidence exists")
	}

	writeFile(t, filepath.Join(dir, "acceptance", "M0.yaml"), "milestone: 0\n")
	sel, _ = selectGates(dir, Config{})
	if !sel.Run["ga"] {
		t.Error("committed acceptance evidence must activate ga")
	}

	bare := t.TempDir()
	writeFile(t, filepath.Join(bare, "BUILD.md"),
		"# B\n\nMode: full\n\n## Build plan\n\n**M0 - Walking skeleton.** DoD: T-X-01 green.\nStatus: closed\n")
	sel, _ = selectGates(bare, Config{})
	if !sel.Run["ga"] {
		t.Error("a milestone marked closed must activate ga on its own")
	}
}

// The stop-time selection matches `machinery check`'s default activation for
// Gv: no evidence file, no gate; a committed attestations.yaml, gate. The
// stop hook is where staleness must surface, because the turn that edited a
// covered artifact is the turn that invalidated the judgment over it.
func TestSelectGatesActivatesGvOnAttestationEvidence(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ARCHITECTURE.md"), "# A\n")
	sel, _ := selectGates(dir, Config{})
	if sel.Run["gv"] {
		t.Error("gv must not run before attestation evidence is committed")
	}

	writeFile(t, filepath.Join(dir, gates.AttestationsFileName), "attestation_version: 1\nattestations: []\n")
	sel, _ = selectGates(dir, Config{})
	if !sel.Run["gv"] {
		t.Error("committed attestation evidence must activate gv")
	}
}

func TestWaveSentinel(t *testing.T) {
	d := t.TempDir()
	if _, _, active := waveSentinel(d); active {
		t.Fatal("absent sentinel reported active")
	}
	p := filepath.Join(d, ".machinery-wave")
	if err := os.WriteFile(p, []byte("open\n"), 0644); err != nil {
		t.Fatal(err)
	}
	left, stale, active := waveSentinel(d)
	if !active || stale || left == "" {
		t.Fatalf("fresh sentinel: left=%q stale=%v active=%v", left, stale, active)
	}
	if err := os.WriteFile(p, []byte("closed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, stale, active = waveSentinel(d)
	if !active || !stale {
		t.Fatalf("non-open sentinel: stale=%v active=%v", stale, active)
	}
}

func TestWaveSentinelRejectsMalformedAndNonregularEntries(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*testing.T, string)
	}{
		{"malformed", func(t *testing.T, path string) { writeFile(t, path, "wave open\n") }},
		{"directory", func(t *testing.T, path string) {
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"symlink", func(t *testing.T, path string) {
			outside := filepath.Join(t.TempDir(), "outside-wave")
			writeFile(t, outside, "240\n")
			if err := os.Symlink(outside, path); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			tc.setup(t, filepath.Join(design, waveSentinelName))
			left, stale, active := waveSentinel(design)
			if !active || !stale || left != "" {
				t.Fatalf("invalid sentinel could suppress blocking: left=%q stale=%v active=%v", left, stale, active)
			}
		})
	}
}

// The upgrade protocol forbids mixing a binary upgrade with a design change;
// the warning fires exactly when a regenerated artifact's machinery-version
// stamp changed AND a hand-written design file changed in the same tree.
func TestUpgradeMixWarning(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		base := []string{"-c", "user.name=hook test", "-c", "user.email=hook@example.invalid"}
		if out, err := testgit.Run(t.Context(), root, append(base, args...)...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q")
	writeFile(t, filepath.Join(root, "design", "domain.modelith.yaml"), "entities: {}\n")
	writeFile(t, filepath.Join(root, "design", "machines", "Order.oracle.md"),
		"<!-- machinery-version: v0.1.0 -->\n| test id | stable id |\n|---|---|\n| T-1 | ORDE-aaa |\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	baseCommit, err := resolveUpgradeCommit(root)
	if err != nil {
		t.Fatal(err)
	}
	warning := func() string {
		t.Helper()
		w, err := upgradeMixWarning(root)
		if err != nil {
			t.Fatalf("upgrade mix proof failed: %v", err)
		}
		return w
	}

	if w := warning(); w != "" {
		t.Fatalf("clean tree must not warn: %q", w)
	}
	// stamp change alone: an upgrade in flight, no mixing
	writeFile(t, filepath.Join(root, "design", "machines", "Order.oracle.md"),
		"<!-- machinery-version: v0.2.0 -->\n| test id | stable id |\n|---|---|\n| T-1 | ORDE-aaa |\n")
	if w := warning(); w != "" {
		t.Fatalf("an upgrade alone must not warn: %q", w)
	}
	// plus a hand-written edit: the mix the protocol forbids
	writeFile(t, filepath.Join(root, "design", "domain.modelith.yaml"), "entities: {A: {}}\n")
	w := warning()
	if !strings.Contains(w, "mixes a binary upgrade") || !strings.Contains(w, "machines/Order.oracle.md") {
		t.Fatalf("mixed change set must warn naming the artifact: %q", w)
	}
	if !strings.Contains(w, "with 1 hand-written design edit(s)") {
		t.Fatalf("mixed change set must count each changed hand-written path exactly once: %q", w)
	}
	stable := t.TempDir()
	writeFile(t, filepath.Join(stable, "domain.modelith.yaml"), "entities: {A: {}}\n")
	writeFile(t, filepath.Join(stable, "machines", "Order.oracle.md"),
		"<!-- machinery-version: v0.2.0 -->\n| test id | stable id |\n|---|---|\n| T-1 | ORDE-aaa |\n")
	git("add", "-A")
	git("commit", "-q", "-m", "move baseline ref")
	afterMove, err := upgradeMixWarningAt(root, "design", stable, baseCommit)
	if err != nil {
		t.Fatal(err)
	}
	if afterMove != w {
		t.Fatalf("moving HEAD after baseline acquisition changed the decision:\nbefore: %q\nafter:  %q", w, afterMove)
	}
	// hand edit alone (stamp reverted): no warning
	writeFile(t, filepath.Join(root, "design", "machines", "Order.oracle.md"),
		"<!-- machinery-version: v0.1.0 -->\n| test id | stable id |\n|---|---|\n| T-1 | ORDE-aaa |\n")
	if w := warning(); w != "" {
		t.Fatalf("a design edit alone must not warn: %q", w)
	}
}

func installHookFakeGit(t *testing.T, script string, mode os.FileMode) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake git executable uses a POSIX shell")
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte("#!/bin/sh\n"+script+"\n"), mode); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestUpgradeMixProofFailuresAreErrors(t *testing.T) {
	gitRoot := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		return root
	}
	t.Run("genuine non-repository", func(t *testing.T) {
		if warning, err := upgradeMixWarning(t.TempDir()); err != nil || warning != "" {
			t.Fatalf("non-repository has no upgrade-mix invariant: warning=%q err=%v", warning, err)
		}
	})
	t.Run("missing git", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		if _, err := upgradeMixWarning(gitRoot(t)); err == nil || !strings.Contains(err.Error(), "executable file not found") {
			t.Fatalf("missing git was treated as no mix: %v", err)
		}
	})
	t.Run("permission", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("executable permission bits are POSIX-specific")
		}
		bin := t.TempDir()
		if err := os.WriteFile(filepath.Join(bin, "git"), []byte("not executable\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", bin)
		if _, err := upgradeMixWarning(gitRoot(t)); err == nil {
			t.Fatalf("unexecutable git was treated as no mix: %v", err)
		}
	})
	t.Run("timeout", func(t *testing.T) {
		installHookFakeGit(t, "exec /bin/sleep 10", 0o755)
		old := upgradeGitTimeout
		upgradeGitTimeout = 25 * time.Millisecond
		t.Cleanup(func() { upgradeGitTimeout = old })
		if _, err := upgradeMixWarning(gitRoot(t)); err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Fatalf("hung git was treated as no mix: %v", err)
		}
	})
	t.Run("malformed baseline", func(t *testing.T) {
		installHookFakeGit(t, `printf 'M\000'`, 0o755)
		if _, err := upgradeMixWarning(gitRoot(t)); err == nil || !strings.Contains(err.Error(), "noncanonical object id") {
			t.Fatalf("corrupt baseline output was treated as no mix: %v", err)
		}
	})
	t.Run("show failure", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(root, "design", "machines", "Order.oracle.md"), "<!-- machinery-version: v2.0.0 -->\n")
		installHookFakeGit(t, `
if [ "$3" = diff ]; then
  printf 'M\000design/machines/Order.oracle.md\000'
  exit 0
fi
echo 'committed object unreadable' >&2
exit 19`, 0o755)
		if _, err := upgradeMixWarning(root); err == nil || !strings.Contains(err.Error(), "committed object unreadable") {
			t.Fatalf("git show failure was treated as a new file/no mix: %v", err)
		}
	})
}

func TestRunUpgradeGitForcesStableLocaleWithoutDuplicateOverrides(t *testing.T) {
	installHookFakeGit(t, `printf 'locale=%s/%s\n' "$LC_ALL" "$LANG" >&2
exit 19`, 0o755)
	root := t.TempDir()
	t.Setenv("LC_ALL", "tr_TR.UTF-8")
	t.Setenv("LANG", "de_DE.UTF-8")
	_, first := runUpgradeGit(root, "status")
	t.Setenv("LC_ALL", "ja_JP.UTF-8")
	t.Setenv("LANG", "fr_FR.UTF-8")
	_, second := runUpgradeGit(root, "status")
	if first == nil || second == nil {
		t.Fatalf("fake git failure was lost: first=%v second=%v", first, second)
	}
	if first.Error() != second.Error() || !strings.Contains(first.Error(), "locale=C/C") {
		t.Fatalf("git diagnostic varied with ambient locale:\nfirst:  %v\nsecond: %v", first, second)
	}
}

func TestRunUpgradeGitIgnoresAmbientRepositoryRedirectionAndSuccessWarnings(t *testing.T) {
	makeRepo := func(t *testing.T, name string) (string, string) {
		t.Helper()
		repo := t.TempDir()
		git := func(args ...string) string {
			t.Helper()
			base := []string{"-c", "user.name=hook test", "-c", "user.email=hook@example.invalid"}
			out, err := testgit.Run(t.Context(), repo, append(base, args...)...)
			if err != nil {
				t.Fatalf("git %v: %v: %s", args, err, out)
			}
			return strings.TrimSpace(string(out))
		}
		git("init", "-q")
		writeFile(t, filepath.Join(repo, name), name+"\n")
		git("add", name)
		git("commit", "-q", "-m", name)
		return repo, git("rev-parse", "HEAD")
	}
	repoA, headA := makeRepo(t, "a")
	repoB, _ := makeRepo(t, "b")
	t.Setenv("GIT_DIR", filepath.Join(repoB, ".git"))
	t.Setenv("GIT_WORK_TREE", repoB)
	t.Setenv("GIT_TRACE", "1")
	got, err := resolveUpgradeCommit(repoA)
	if err != nil || got != headA {
		t.Fatalf("ambient Git redirection changed hook repository: got=%q want=%q err=%v", got, headA, err)
	}

	installHookFakeGit(t, "printf '%040d\\n' 0; echo 'warning: injected config' >&2; exit 0", 0o755)
	if _, err := runUpgradeGit(t.TempDir(), "rev-parse", "HEAD"); err == nil || !strings.Contains(err.Error(), "emitted stderr on success") {
		t.Fatalf("successful Git warning was discarded: %v", err)
	}
}

func TestRunUpgradeGitKillsBackgroundDescendantsOnTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake git executable uses a POSIX shell")
	}
	sentinel := filepath.Join(t.TempDir(), "survived")
	t.Setenv("MACHINERY_UPGRADE_GIT_SENTINEL", sentinel)
	installHookFakeGit(t, `(
  /bin/sleep 1
  printf survived > "$MACHINERY_UPGRADE_GIT_SENTINEL"
) &
/bin/sleep 10`, 0o755)
	old := upgradeGitTimeout
	upgradeGitTimeout = 50 * time.Millisecond
	t.Cleanup(func() { upgradeGitTimeout = old })
	started := time.Now()
	if _, err := runUpgradeGit(t.TempDir(), "status"); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("timed-out git process tree was not reported: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("background child retained git output pipes for %s", elapsed)
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err := os.Lstat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("background git descendant survived timeout cleanup: %v", err)
	}
}

// The plain dialog register swaps only the USER-FACING stop messages and adds
// a register reminder to session start; deny reasons, block reasons, and the
// default strings stay byte-identical (every other test in this file runs
// with the default register and pins that).
func TestDialogPlainRegister(t *testing.T) {
	t.Run("unknown value is a hard managed error", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ConfigName), `{"dialog":"terse"}`)
		cfg, ok, warn := Load(root)
		if !ok || cfg.loadError == "" || !strings.Contains(warn, "dialog value") {
			t.Fatalf("cfg=%+v ok=%v warn=%q", cfg, ok, warn)
		}
	})
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ConfigName), `{"dialog":"plain"}`)
	writeFile(t, filepath.Join(root, "design", "domain.modelith.yaml"), "model: {}\n")
	t.Run("mid-phase stop message is plain", func(t *testing.T) {
		sid := "s-plain"
		appendState(root, sid, "design")
		out := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
		var got stopOut
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("stop output is not JSON: %v (%q)", err, out)
		}
		if got.Decision != "" {
			t.Fatalf("plain register never changes blocking behavior: %+v", got)
		}
		if !strings.Contains(got.SystemMessage, "design-check item(s) are still open") {
			t.Fatalf("plain message expected: %+v", got)
		}
		for _, jargon := range []string{"gate ERROR", "DRIFT", "machinery check"} {
			if strings.Contains(got.SystemMessage, jargon) {
				t.Fatalf("plain message leaks %q: %+v", jargon, got)
			}
		}
	})
	t.Run("session start carries the register reminder", func(t *testing.T) {
		out := runEvent(t, root, Input{HookEventName: "SessionStart"})
		if !strings.Contains(out, "Dialog register: PLAIN") {
			t.Fatalf("session start must remind the conductor of the register, got %q", out)
		}
	})
}
