package checker

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

const checkerDeterminismMode = "MACHINERY_CHECKER_DETERMINISM_MODE"

func TestCheckerDeterminismHelper(t *testing.T) {
	switch os.Getenv(checkerDeterminismMode) {
	case "generate":
		model, err := LoadModel(writeTemp(t, "determinism.modelith.yaml", sampleModel))
		if err != nil {
			t.Fatal(err)
		}
		_, err = Generate(model, manifestWith([]string{"machines", "actions", "scenarios"}, nil), validTestDesignID, "v0")
		fmt.Print(err)
	case "root":
		root, err := openDesignRoot(os.Getenv("MACHINERY_CHECKER_DETERMINISM_DESIGN"))
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := root.close(); err != nil {
				t.Error(err)
			}
		}()
		if os.Getenv("MACHINERY_CHECKER_DETERMINISM_DISCOVERY") == "models" {
			_, err = root.modelPaths()
		} else {
			_, err = root.manifestPaths()
		}
		fmt.Print(err)
	}
}

func repeatedCheckerDiagnostic(t *testing.T, env ...string) string {
	t.Helper()
	var want string
	for i := 0; i < 30; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCheckerDeterminismHelper$")
		cmd.Env = append(os.Environ(), env...)
		gotBytes, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			t.Fatalf("determinism helper failed: %v\n%s", err, gotBytes)
		}
		got := string(gotBytes)
		if i == 0 {
			want = got
		} else if got != want {
			t.Fatalf("diagnostic changed between process 0 and %d:\nfirst: %q\nnext:  %q", i, want, got)
		}
	}
	return want
}

func TestGenerateUnsupportedLayerDiagnosticIsProcessDeterministic(t *testing.T) {
	got := repeatedCheckerDiagnostic(t, checkerDeterminismMode+"=generate")
	if !strings.Contains(got, `layer "actions" is not yet supported`) {
		t.Fatalf("unsupported layers did not use canonical priority: %q", got)
	}
}

func TestRootDiscoveryDiagnosticsAreProcessDeterministic(t *testing.T) {
	design := t.TempDir()
	for _, name := range []string{"z?.modelith.yaml", "a?.modelith.yaml"} {
		if err := os.WriteFile(filepath.Join(design, name), []byte(sampleModel), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	checkerDir := filepath.Join(design, "checkers")
	if err := os.Mkdir(checkerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"z?.checker.yaml", "a?.checker.yaml"} {
		if err := os.WriteFile(filepath.Join(checkerDir, name), []byte("invalid"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, discovery := range []string{"models", "manifests"} {
		t.Run(discovery, func(t *testing.T) {
			got := repeatedCheckerDiagnostic(t,
				checkerDeterminismMode+"=root",
				"MACHINERY_CHECKER_DETERMINISM_DESIGN="+design,
				"MACHINERY_CHECKER_DETERMINISM_DISCOVERY="+discovery,
			)
			if !strings.Contains(got, `name "a?`) {
				t.Fatalf("discovery did not report the lexically first invalid entry: %q", got)
			}
		})
	}
}
