package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoutingSnapshotTransientWriter(t *testing.T) {
	for _, ignored := range []bool{true, false} {
		t.Run(map[bool]string{true: "ignored", false: "source"}[ignored], func(t *testing.T) {
			isolateHookState(t)
			root := managedRoot(t)
			design := filepath.Join(root, "design")
			if err := os.WriteFile(filepath.Join(design, ".machineryignore"), []byte("logs/\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(design, "logs"), 0700); err != nil {
				t.Fatal(err)
			}
			rel := "logs/run.log"
			if !ignored {
				rel = "BUILD.md"
			}
			target := filepath.Join(design, filepath.FromSlash(rel))
			if err := os.WriteFile(target, []byte("before\n"), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, ok, _ := Load(root)
			if !ok {
				t.Fatal("missing config")
			}
			err := withRoutingSnapshot(root, cfg, func(Config) error {
				done := make(chan error, 1)
				go func() {
					f, e := os.OpenFile(target, os.O_APPEND|os.O_WRONLY, 0600)
					if e == nil {
						_, e = f.WriteString("after\n")
						ce := f.Close()
						if e == nil {
							e = ce
						}
					}
					done <- e
				}()
				return <-done
			})
			if ignored && err != nil {
				t.Fatalf("ignored writer denied unrelated event: %v", err)
			}
			if !ignored && (err == nil || !strings.Contains(err.Error(), rel)) {
				t.Fatalf("design source change must fail closed and name path: %v", err)
			}
		})
	}
}

func TestProcessControlDoesNotReadChangingDesign(t *testing.T) {
	isolateHookState(t)
	root := managedRoot(t)
	// An unsupported in-tree writer must not strand the command that stops it.
	if err := os.Symlink("missing", filepath.Join(root, "design", "run.log")); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"kill 12345", "kill -TERM 12345", "pkill -f worker", "kill %1"} {
		if out := runEvent(t, root, Input{HookEventName: "PreToolUse", SessionID: "stop-writer", ToolName: "Bash", ToolInput: toolInput{Command: command}}); out != "" {
			t.Fatalf("process control denied: %s", out)
		}
	}
}

func TestProcessControlRejectsShellEvaluation(t *testing.T) {
	for _, command := range []string{"kill 123; echo write", "kill $(cat pid)", "kill 123 > design/BUILD.md", "pkill `echo worker`", "kill 123 && touch design/BUILD.md"} {
		if processControl(Input{ToolName: "Bash", ToolInput: toolInput{Command: command}}) {
			t.Fatalf("shell evaluation bypassed governance: %s", command)
		}
	}
}

func TestRoutingSnapshotGuardsIgnorePolicy(t *testing.T) {
	isolateHookState(t)
	root := managedRoot(t)
	target := filepath.Join(root, "design", ".machineryignore")
	if err := os.WriteFile(target, []byte("logs/\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := Load(root)
	err := withRoutingSnapshot(root, cfg, func(Config) error { return os.WriteFile(target, []byte("**\n"), 0600) })
	if err == nil || !strings.Contains(err.Error(), ".machineryignore") {
		t.Fatalf("ignore policy change escaped snapshot: %v", err)
	}
}

func TestRoutingSnapshotOmitsLargeIgnoredLog(t *testing.T) {
	isolateHookState(t)
	root := managedRoot(t)
	design := filepath.Join(root, "design")
	writeFile(t, filepath.Join(design, ".machineryignore"), "logs/\n")
	if err := os.Mkdir(filepath.Join(design, "logs"), 0700); err != nil {
		t.Fatal(err)
	}
	log, err := os.Create(filepath.Join(design, "logs", "run.log"))
	if err != nil {
		t.Fatal(err)
	}
	if err := log.Truncate(2 << 30); err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := Load(root)
	if err := withRoutingSnapshot(root, cfg, func(Config) error { return nil }); err != nil {
		t.Fatalf("ignored log metadata blocked routing: %v", err)
	}
}
