package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStableRegularRejectsOversizedSparseInputBeforeRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized.modelith.yaml")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(stableRegularMaxBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := openStableRegular(path); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("stable reader accepted oversized sparse input: %v", err)
	}
}

func TestScalePropagatesMachineParseFailure(t *testing.T) {
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "domain.modelith.yaml"), []byte("entities: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "machines", "Broken.machine.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var want string
	for i := 0; i < 10; i++ {
		cmd := newScaleCmd()
		cmd.SetArgs([]string{design})
		err := executeCapturedCommand(cmd)
		if err == nil || !strings.Contains(err.Error(), "read machine Broken.machine.json") || strings.Contains(err.Error(), "machinery-design-reader-") {
			t.Fatalf("scale error = %v, want stable machine parse failure", err)
		}
		if i == 0 {
			want = err.Error()
		} else if err.Error() != want {
			t.Fatalf("run %d diagnostic changed:\nwant %s\n got %s", i, want, err)
		}
	}
}

func TestScalePropagatesModelParseFailure(t *testing.T) {
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "domain.modelith.yaml"), []byte("entities: [unterminated"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newScaleCmd()
	cmd.SetArgs([]string{design})
	err := executeCapturedCommand(cmd)
	if err == nil || !strings.Contains(err.Error(), "parse model domain.modelith.yaml") {
		t.Fatalf("scale error = %v, want model parse failure", err)
	}
}

func writeMinimalScaleDesign(t *testing.T, design string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(design, "domain.modelith.yaml"), []byte("entities: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "ARCHITECTURE.md"), []byte("# Architecture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScaleRejectsMutationAfterSnapshot(t *testing.T) {
	design := t.TempDir()
	writeMinimalScaleDesign(t, design)
	prior := designReaderAfterSnapshot
	designReaderAfterSnapshot = func() {
		if err := os.WriteFile(filepath.Join(design, "domain.modelith.yaml"), []byte("entities:\n  Changed: {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { designReaderAfterSnapshot = prior }()
	cmd := newScaleCmd()
	cmd.SetArgs([]string{design})
	err := executeCapturedCommand(cmd)
	if err == nil || !strings.Contains(err.Error(), "design changed outside the snapshot lock") {
		t.Fatalf("scale mutation error = %v", err)
	}
}

func TestScaleFailsClosedOnMissingEventSource(t *testing.T) {
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "domain.modelith.yaml"), []byte("entities: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newScaleCmd()
	cmd.SetArgs([]string{design})
	err := executeCapturedCommand(cmd)
	if err == nil || !strings.Contains(err.Error(), "read event contracts") {
		t.Fatalf("missing ARCHITECTURE.md was measured as zero rows: %v", err)
	}
}

func TestScaleRejectsSymlinkInventory(t *testing.T) {
	design := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.modelith.yaml")
	if err := os.WriteFile(outside, []byte("entities: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(design, "domain.modelith.yaml")); err != nil {
		t.Fatal(err)
	}
	cmd := newScaleCmd()
	cmd.SetArgs([]string{design})
	err := executeCapturedCommand(cmd)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("scale accepted symlinked design inventory: %v", err)
	}
}

func TestLintAndIRDumpRejectMutationAfterSnapshot(t *testing.T) {
	for _, command := range []string{"lint", "ir-dump"} {
		t.Run(command, func(t *testing.T) {
			_, _, _ = withCapturedIO(t)
			design := t.TempDir()
			machines := filepath.Join(design, "machines")
			if err := os.Mkdir(machines, 0o755); err != nil {
				t.Fatal(err)
			}
			machine := filepath.Join(machines, "Thing.machine.json")
			body := []byte(`{"id":"thing","initial":"A","states":{"A":{"type":"final"}}}`)
			if err := os.WriteFile(machine, body, 0o644); err != nil {
				t.Fatal(err)
			}
			prior := designReaderAfterSnapshot
			designReaderAfterSnapshot = func() {
				if err := os.WriteFile(machine, append(body, '\n'), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			defer func() { designReaderAfterSnapshot = prior }()
			var err error
			if command == "lint" {
				cmd := newLintCmd()
				cmd.SetArgs([]string{machines})
				err = executeCapturedCommand(cmd)
			} else {
				err = irDumpRun(machine)
			}
			if err == nil || !strings.Contains(err.Error(), "design changed outside the snapshot lock") {
				t.Fatalf("%s hid paused mutation: %v", command, err)
			}
		})
	}
}
