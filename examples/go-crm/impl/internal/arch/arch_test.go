package arch_test

// C-ARCH-01 (BUILD.md 4.5, 7.2): static import check. Every cross-boundary
// import edge among the five boundaries must be in the section 4.5 allow list;
// internal/cli must not import internal/authz; only internal/repo may import
// go-ladybug. Uses `go list` (offline) for the default-build import graph plus a
// source scan for the build-tagged go-ladybug importer.
//
// This is a structural test, so it is GREEN by construction against correct
// scaffolding: it validates the package layout the implementer must preserve.

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// boundary maps an internal package import path to its BUILD.md 4.5 boundary id.
var boundary = map[string]string{
	"crm/internal/cli":     "commands",
	"crm/internal/session": "session",
	"crm/internal/authz":   "authz",
	"crm/internal/domain":  "domain",
	"crm/internal/repo":    "repo",
}

// allowed is the section 4.5 dependency_rules.allow list, as importer->imported.
var allowed = map[string]bool{
	"commands->session": true,
	"commands->domain":  true,
	"commands->repo":    true,
	"session->repo":     true,
	"domain->authz":     true,
	"domain->repo":      true,
}

const ladybugImport = "github.com/LadybugDB/go-ladybug"

// TestCArch01Boundaries checks the default-build import graph against the allow
// list and the commands->authz deny rule.
func TestCArch01Boundaries(t *testing.T) {
	out, err := exec.Command("go", "list", "-f", "{{.ImportPath}}|{{join .Imports \",\"}}", "crm/...").CombinedOutput()
	if err != nil {
		t.Fatalf("go list failed: %v\n%s", err, out)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "|", 2)
		src := boundary[parts[0]]
		if src == "" || len(parts) < 2 {
			continue
		}
		for _, dep := range strings.Split(parts[1], ",") {
			if dst := boundary[dep]; dst != "" {
				edge := src + "->" + dst
				if !allowed[edge] {
					t.Errorf("C-ARCH-01: disallowed cross-boundary edge %s (%s imports %s)", edge, parts[0], dep)
				}
			}
			if dep == ladybugImport && src != "repo" {
				t.Errorf("C-ARCH-01: %s imports go-ladybug; only crm.repo may (deny crm.* -> external.ladybug)", parts[0])
			}
		}
	}
}

// TestCArch01LadybugIsolatedInSource ensures the build-tagged go-ladybug importer
// (excluded from the default `go list`) lives only under internal/repo.
func TestCArch01LadybugIsolatedInSource(t *testing.T) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "crm").Output()
	if err != nil {
		t.Fatalf("go list -m failed: %v", err)
	}
	root := strings.TrimSpace(string(out))
	internal := filepath.Join(root, "internal")
	repoDir := filepath.Join(internal, "repo") + string(os.PathSeparator)

	err = filepath.WalkDir(internal, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(b), `"`+ladybugImport+`"`) && !strings.HasPrefix(path, repoDir) {
			t.Errorf("C-ARCH-01: %s imports go-ladybug outside internal/repo", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
}
