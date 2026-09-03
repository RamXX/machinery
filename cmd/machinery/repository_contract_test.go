package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	machversion "github.com/RamXX/machinery/internal/version"
)

// TestRepositoryVersionContracts prevents the documentation/tool metadata
// drift that is otherwise easy to introduce during a release or Go bump.
func TestRepositoryVersionContracts(t *testing.T) {
	root := repoRootDir(t)
	goMod := mustRepositoryFile(t, filepath.Join(root, "go.mod"))
	match := regexp.MustCompile(`(?m)^go ([0-9]+\.[0-9]+\.[0-9]+)$`).FindStringSubmatch(goMod)
	if len(match) != 2 {
		t.Fatal("go.mod must pin a full Go patch version")
	}
	readme := mustRepositoryFile(t, filepath.Join(root, "README.md"))
	if !strings.Contains(readme, "`go.mod` pins "+match[1]) {
		t.Fatalf("README Go pin is stale; want %s", match[1])
	}

	var plugin struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(mustRepositoryFile(t, filepath.Join(root, ".claude-plugin", "plugin.json"))), &plugin); err != nil {
		t.Fatal(err)
	}
	// Versions are numeric only: the binary's bare-build default is the plain
	// plugin version, with no -dev or other pre-release suffix, so a local
	// build reports the same version as the release it corresponds to.
	want := "v" + plugin.Version
	var codexPlugin struct {
		Version string `json:"version"`
		Skills  string `json:"skills"`
	}
	if err := json.Unmarshal([]byte(mustRepositoryFile(t, filepath.Join(root, ".codex-plugin", "plugin.json"))), &codexPlugin); err != nil {
		t.Fatal(err)
	}
	if codexPlugin.Version != plugin.Version || codexPlugin.Skills != "./skills/" {
		t.Fatalf("Codex manifest = %+v, want Claude version %s and shared skills path", codexPlugin, plugin.Version)
	}
	if version != want {
		t.Fatalf("binary default version = %q, plugin metadata requires %q", version, want)
	}
	if strings.Contains(version, "-") {
		t.Fatalf("binary default version %q carries a suffix; versions are numeric only", version)
	}
	makefile := mustRepositoryFile(t, filepath.Join(root, "Makefile"))
	if !strings.Contains(makefile, "INTERNAL_VERSION := "+want+"\n") {
		t.Fatalf("Makefile INTERNAL_VERSION must be %s", want)
	}
}

func TestOpenCodeAdapterContracts(t *testing.T) {
	root := repoRootDir(t)
	plugin := mustRepositoryFile(t, filepath.Join(root, "adapters", "opencode", "plugins", "machinery.js"))
	for _, required := range []string{
		`"tool.execute.before"`,
		`"tool.execute.after"`,
		`"session.idle"`,
		`machinery hook --root`,
		`args.patchText`,
		`input.args`,
		`tool_use_id`,
		`client.tui.showToast`,
		`Machinery governance failed closed`,
		`if (failure) throw new Error(failure)`,
		`Retain the`,
		`until the underlying check is green`,
	} {
		if !strings.Contains(plugin, required) {
			t.Errorf("OpenCode adapter is missing protocol behavior %q", required)
		}
	}
	for _, command := range []string{"design.md", "check.md", "init.md", "status.md"} {
		doc := mustRepositoryFile(t, filepath.Join(root, "adapters", "opencode", "commands", command))
		if !strings.HasPrefix(doc, "---\n") || !strings.Contains(doc, "description:") {
			t.Errorf("OpenCode command %s has no valid frontmatter", command)
		}
	}
}

func TestOpenCodeAdapterPostToolFailures(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is unavailable; static OpenCode adapter contract remains enforced")
	}
	testFile := filepath.Join(repoRootDir(t), "adapters", "opencode", "plugins", "machinery.test.mjs")
	cmd := exec.CommandContext(t.Context(), node, "--test", testFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("OpenCode adapter behavior tests failed: %v\n%s", err, out)
	}
}

func TestModelithInventoryDiscoveryFailsClosed(t *testing.T) {
	repo := repoRootDir(t)
	script := filepath.Join(repo, "scripts", "modelith-inventory.sh")
	makefile := mustRepositoryFile(t, filepath.Join(repo, "Makefile"))
	renderScript := mustRepositoryFile(t, filepath.Join(repo, "scripts", "modelith-render.sh"))
	if strings.Contains(makefile, "$(shell find examples") {
		t.Fatal("Makefile must not discard Modelith discovery exit status through $(shell find ...)")
	}
	for _, required := range []string{".SHELLFLAGS := -eu -o pipefail -c", "$(MODELITH_RENDER) render", "$(MODELITH_INVENTORY) check", "$(MODELITH_INVENTORY) git-diff"} {
		if !strings.Contains(makefile, required) {
			t.Errorf("Makefile is missing checked Modelith inventory contract %q", required)
		}
	}
	for _, required := range []string{"modelith-inventory.sh\" sources", "modelith --version emitted stderr", "modelith render emitted stderr", "modelith-tx", "scripts/run-safe", "publish \"$repo_root\""} {
		if !strings.Contains(renderScript, required) {
			t.Errorf("Modelith renderer is missing closed transaction contract %q", required)
		}
	}
	inventoryScript := mustRepositoryFile(t, script)
	for _, required := range []string{"./scripts/tree-inventory", "-max-entries 100000", "-max-depth 64", "-max-bytes 33554432", "-timeout 15s"} {
		if !strings.Contains(inventoryScript, required) {
			t.Errorf("Modelith inventory lacks bounded traversal contract %q", required)
		}
	}

	corpus := t.TempDir()
	for _, name := range []string{"domain.modelith.yaml", "domain.modelith.md"} {
		if err := os.WriteFile(filepath.Join(corpus, name), []byte("fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := exec.CommandContext(t.Context(), "bash", script, "check", corpus).CombinedOutput(); err != nil {
		t.Fatalf("valid source/render inventory failed: %v\n%s", err, out)
	}

	t.Run("symlink source", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, "real.modelith.yaml")
		if err := os.WriteFile(target, []byte("fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(root, "domain.modelith.yaml")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, "domain.modelith.md"), []byte("fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := exec.CommandContext(t.Context(), "bash", script, "check", root).CombinedOutput()
		if err == nil || !strings.Contains(string(out), "must not contain symlinks") {
			t.Fatalf("symlink source must fail closed: err=%v out=%s", err, out)
		}
	})

	t.Run("symlink render", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "domain.modelith.yaml"), []byte("fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(root, "real.modelith.md")
		if err := os.WriteFile(target, []byte("fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(root, "domain.modelith.md")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		out, err := exec.CommandContext(t.Context(), "bash", script, "check", root).CombinedOutput()
		if err == nil || !strings.Contains(string(out), "must not contain symlinks") {
			t.Fatalf("symlink render must fail closed: err=%v out=%s", err, out)
		}
	})

	t.Run("symlink parent", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "domain.modelith.yaml"), []byte("fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "domain.modelith.md"), []byte("fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		outside := t.TempDir()
		if err := os.WriteFile(filepath.Join(outside, "hidden.modelith.yaml"), []byte("fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(outside, "hidden.modelith.md"), []byte("fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(root, "hidden")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		out, err := exec.CommandContext(t.Context(), "bash", script, "check", root).CombinedOutput()
		if err == nil || !strings.Contains(string(out), "must not contain symlinks") {
			t.Fatalf("symlink parent must fail closed: err=%v out=%s", err, out)
		}
	})

	t.Run("unrelated special entry", func(t *testing.T) {
		mkfifo, err := exec.LookPath("mkfifo")
		if err != nil {
			t.Skip("mkfifo is unavailable on this platform")
		}
		root := t.TempDir()
		for _, name := range []string{"domain.modelith.yaml", "domain.modelith.md"} {
			if err := os.WriteFile(filepath.Join(root, name), []byte("fixture\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		special := filepath.Join(root, "unrelated-special")
		if output, err := exec.CommandContext(t.Context(), mkfifo, special).CombinedOutput(); err != nil {
			t.Skipf("cannot create FIFO fixture: %v\n%s", err, output)
		}
		out, err := exec.CommandContext(t.Context(), "bash", script, "check", root).CombinedOutput()
		if err == nil || !strings.Contains(string(out), "regular files or real directories") {
			t.Fatalf("unrelated special corpus entry must fail closed: err=%v out=%s", err, out)
		}
	})

	empty := t.TempDir()
	if out, err := exec.CommandContext(t.Context(), "bash", script, "check", empty).CombinedOutput(); err == nil || !strings.Contains(string(out), "unexpected empty corpus") {
		t.Fatalf("empty discovery must fail explicitly: err=%v out=%s", err, out)
	}

	t.Run("bounded helper build failure", func(t *testing.T) {
		bin := t.TempDir()
		if err := os.WriteFile(filepath.Join(bin, "go"), []byte("#!/bin/sh\nexit 23\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(t.Context(), "bash", script, "check", corpus)
		cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "build bounded tree inventory helper failed") {
			t.Fatalf("helper build failure must invalidate inventory: err=%v out=%s", err, out)
		}
	})
}

func TestPreflightC4DiscoveryFailsClosed(t *testing.T) {
	repo := repoRootDir(t)
	preflight := mustRepositoryFile(t, filepath.Join(repo, "scripts", "preflight.sh"))
	for _, required := range []string{
		`preflight_work=$(mktemp -d)`,
		`trap cleanup EXIT`,
		`c4_inventory=$preflight_work/c4.inventory`,
		`scripts/c4-inventory.sh examples >"$c4_inventory"`,
		`done <"$c4_inventory"`,
	} {
		if !strings.Contains(preflight, required) {
			t.Errorf("preflight lacks checked C4 discovery contract %q", required)
		}
	}
	if strings.Contains(preflight, "done < <(find examples") {
		t.Fatal("preflight must not hide C4 find/sort failure in process substitution")
	}
	if !strings.Contains(preflight, "unset MACHINERY_STRUCTURIZR_CLI MACHINERY_STRUCTURIZR_CLI_CLOSURE_SHA256") {
		t.Fatal("preflight must use verify-c4's pinned provisioner instead of forwarding an unbound ambient Structurizr executable")
	}
	ci := mustRepositoryFile(t, filepath.Join(repo, ".github", "workflows", "ci.yml"))
	for _, required := range []string{"scripts/c4-inventory.sh examples", `test -s "$c4_inventory"`, `done <"$c4_inventory"`} {
		if !strings.Contains(ci, required) {
			t.Errorf("CI lacks checked nonempty C4 discovery contract %q", required)
		}
	}
	if strings.Contains(ci, "find examples -type f -name workspace.dsl") {
		t.Fatal("CI must use the same checked C4 inventory helper as preflight")
	}

	script := filepath.Join(repo, "scripts", "c4-inventory.sh")
	scriptBody := mustRepositoryFile(t, script)
	for _, required := range []string{"./scripts/tree-inventory", "-file-name workspace.dsl", "-regular-files-only", "-max-entries 4096", "-max-depth 64", "-max-bytes 8388608", "-timeout 15s"} {
		if !strings.Contains(scriptBody, required) {
			t.Errorf("custom-root C4 inventory lacks bounded exact contract %q", required)
		}
	}
	if strings.Contains(scriptBody, "find \"$root\"") {
		t.Fatal("custom-root C4 inventory must not use recursive find")
	}
	corpus := t.TempDir()
	if out, err := exec.CommandContext(t.Context(), "bash", script, corpus).CombinedOutput(); err == nil || !strings.Contains(string(out), "unexpected empty corpus") {
		t.Fatalf("empty C4 discovery must fail: err=%v out=%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(corpus, "workspace.dsl"), []byte("workspace {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(corpus, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "workspace.dsl"), []byte("workspace {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "notworkspace.dsl"), []byte("workspace {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.CommandContext(t.Context(), "bash", script, corpus).CombinedOutput()
	if err != nil {
		t.Fatalf("valid C4 discovery failed: err=%v out=%s", err, out)
	}
	want := filepath.Join(corpus, "nested", "workspace.dsl") + "\n" + filepath.Join(corpus, "workspace.dsl") + "\n"
	if string(out) != want {
		t.Fatalf("custom-root C4 inventory = %q, want exact byte-sorted regular workspaces %q", out, want)
	}

	t.Run("bounded helper build failure", func(t *testing.T) {
		bin := t.TempDir()
		if err := os.WriteFile(filepath.Join(bin, "go"), []byte("#!/bin/sh\nexit 29\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(t.Context(), "bash", script, corpus)
		cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "build bounded tree inventory helper failed") {
			t.Fatalf("helper build failure must invalidate C4 corpus: err=%v out=%s", err, out)
		}
	})

	t.Run("high-entry custom root", func(t *testing.T) {
		root := t.TempDir()
		for index := 0; index <= 4096; index++ {
			if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("entry-%05d", index)), nil, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		out, err := exec.CommandContext(t.Context(), "bash", script, root).CombinedOutput()
		if err == nil || !strings.Contains(string(out), "entry limit") {
			t.Fatalf("high-entry custom C4 root was accepted: err=%v out=%s", err, out)
		}
	})

	t.Run("deep custom root", func(t *testing.T) {
		root := t.TempDir()
		deep := root
		for depth := 0; depth < 64; depth++ {
			deep = filepath.Join(deep, "d")
		}
		if err := os.MkdirAll(deep, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(deep, "workspace.dsl"), []byte("workspace {}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		out, err := exec.CommandContext(t.Context(), "bash", script, root).CombinedOutput()
		if err == nil || !strings.Contains(string(out), "depth limit") {
			t.Fatalf("deep custom C4 root was accepted: err=%v out=%s", err, out)
		}
	})
}

func TestExampleInventoryIsClosedAndDrivesEveryRunner(t *testing.T) {
	repo := repoRootDir(t)
	script := filepath.Join(repo, "scripts", "example-inventory.sh")
	cmd := exec.CommandContext(t.Context(), "bash", script, "rows")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("canonical example inventory is invalid: %v\n%s", err, out)
	}
	scriptBody := mustRepositoryFile(t, script)
	for _, required := range []string{"./scripts/tree-inventory", "-max-entries 100000", "-max-depth 64", "-max-bytes 33554432", "-timeout 15s", `if [[ "$entry" == */design ]]`, "example design root must be a real directory"} {
		if !strings.Contains(scriptBody, required) {
			t.Errorf("example inventory does not classify the design-root entry itself: missing %q", required)
		}
	}
	rows := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(rows) != 8 {
		t.Fatalf("registered design rows = %d, want 8: %s", len(rows), out)
	}
	joined := string(out)
	for _, exact := range []string{
		"examples/go-crm/design\texamples/go-crm/impl\tyes\t-\tyes\tyes\t-\tgo\tyes",
		"examples/pii-flow/design\t-\tyes\texamples/pii-flow/checkers.local.example.yaml\tyes\tno\t-\t-\tno",
		"examples/checkout-split/parent/design\t-\tno\t-\tyes\tno\tparent\t-\tno",
		"examples/checkout-split/orders/design\t-\tyes\t-\tyes\tno\tchild:examples/checkout-split/parent/design\t-\tno",
	} {
		if !strings.Contains(joined, exact) {
			t.Errorf("example capability registry lacks exact row %q", exact)
		}
	}

	manifest := mustRepositoryFile(t, filepath.Join(repo, "examples", "inventory.tsv"))
	missing := strings.Replace(manifest, "examples/fulfillment/design\t-\tyes\t-\tyes\tno\t-\t-\tno\n", "", 1)
	badManifest := filepath.Join(t.TempDir(), "inventory.tsv")
	if err := os.WriteFile(badManifest, []byte(missing), 0o644); err != nil {
		t.Fatal(err)
	}
	missingCmd := exec.CommandContext(t.Context(), "bash", script, "rows", badManifest)
	missingCmd.Dir = repo
	if out, err := missingCmd.CombinedOutput(); err == nil || !strings.Contains(string(out), "does not exactly match discovered") {
		t.Fatalf("unregistered design root did not fail reverse coverage: err=%v out=%s", err, out)
	}

	t.Run("design root entry is closed", func(t *testing.T) {
		fixture := t.TempDir()
		fixtureExamples := filepath.Join(fixture, "examples")
		if err := os.CopyFS(fixtureExamples, os.DirFS(filepath.Join(repo, "examples"))); err != nil {
			t.Fatalf("copy example fixture: %v", err)
		}
		unregistered := filepath.Join(fixtureExamples, "unregistered")
		populatedDesign := filepath.Join(unregistered, "design")
		if err := os.MkdirAll(populatedDesign, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(populatedDesign, "unexpected.md"), []byte("unexpected\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(t.Context(), "bash", script, "rows")
		cmd.Dir = fixture
		if out, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(out), "does not exactly match discovered") {
			t.Fatalf("populated unregistered design root must fail: err=%v out=%s", err, out)
		}

		if err := os.RemoveAll(unregistered); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(unregistered, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(fixtureExamples, "go-crm", "design"), filepath.Join(unregistered, "design")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		cmd = exec.CommandContext(t.Context(), "bash", script, "rows")
		cmd.Dir = fixture
		if out, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(out), "must not be a symlink") {
			t.Fatalf("symlinked design root must fail: err=%v out=%s", err, out)
		}
	})

	for _, file := range []string{"Makefile", "scripts/preflight.sh", ".github/workflows/ci.yml", ".github/workflows/formal.yml", ".github/workflows/nightly.yml", ".github/workflows/security.yml"} {
		body := mustRepositoryFile(t, filepath.Join(repo, filepath.FromSlash(file)))
		if !strings.Contains(body, "example-inventory.sh") && file != "Makefile" {
			t.Errorf("%s does not consume the checked example inventory", file)
		}
		if strings.Contains(body, ".bin/machinery check examples/") || strings.Contains(body, ".bin/machinery verify-formal examples/") {
			t.Errorf("%s retains a hardcoded example loop that can omit new registry rows", file)
		}
		for _, stale := range []string{
			"pack generate examples/checkout-split",
			"pack refine examples/checkout-split",
			"working-directory: examples/go-crm/impl",
			"( cd examples/go-crm/impl",
		} {
			if strings.Contains(body, stale) {
				t.Errorf("%s retains hardcoded example capability %q instead of the checked inventory", file, stale)
			}
		}
	}
	for _, action := range []string{"formal", "c4", "impl-modules", "checkers", "pack-parents", "pack-children", "security"} {
		cmd := exec.CommandContext(t.Context(), "bash", script, action)
		cmd.Dir = repo
		if actionOut, err := cmd.CombinedOutput(); err != nil || len(bytes.TrimSpace(actionOut)) == 0 {
			t.Errorf("example inventory action %s is empty or invalid: err=%v out=%s", action, err, actionOut)
		}
	}
}

func TestDependencyReviewHasNoWarningOnlySeverityTier(t *testing.T) {
	workflow := mustRepositoryFile(t, filepath.Join(repoRootDir(t), ".github", "workflows", "security.yml"))
	if !strings.Contains(workflow, "fail-on-severity: low") {
		t.Fatal("dependency review must fail on every introduced vulnerability severity")
	}
	for _, weaker := range []string{"fail-on-severity: moderate", "fail-on-severity: high", "fail-on-severity: critical"} {
		if strings.Contains(workflow, weaker) {
			t.Fatalf("dependency review retains weaker policy %q", weaker)
		}
	}
}

func TestPreflightFailsClosedWithoutTrustedMergeBaseAndUsesBash32Syntax(t *testing.T) {
	repo := repoRootDir(t)
	script := filepath.Join(repo, "scripts", "preflight.sh")
	body := mustRepositoryFile(t, script)
	for _, forbidden := range []string{"git rev-parse HEAD^", "mapfile"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("preflight retains non-deterministic or non-portable construct %q", forbidden)
		}
	}
	for _, required := range []string{"PREFLIGHT_BASE_REF:-origin/main", "cannot resolve trusted aggregate-diff base", "while IFS= read -r shell_file", `shell_files+=("$shell_file")`} {
		if !strings.Contains(body, required) {
			t.Errorf("preflight lacks fail-closed/Bash 3.2 contract %q", required)
		}
	}
	ci := mustRepositoryFile(t, filepath.Join(repo, ".github", "workflows", "ci.yml"))
	if strings.Contains(ci, "mapfile") {
		t.Fatal("CI retains mapfile, which is unavailable in the stock macOS Bash 3.2 runtime")
	}
	for _, required := range []string{"while IFS= read -r shell_file", `shell_files+=("$shell_file")`, "grep_status=0", `case "$grep_status" in`} {
		if !strings.Contains(ci, required) {
			t.Errorf("CI lacks Bash 3.2/fail-closed scan contract %q", required)
		}
	}
	for _, required := range []string{"reject_matches()", "scan failed", `*) fail "$scan_failure (status $status)"`} {
		if !strings.Contains(body, required) {
			t.Errorf("preflight lacks fail-closed negative-scan contract %q", required)
		}
	}
	if output, err := exec.CommandContext(t.Context(), "/bin/bash", "-n", script).CombinedOutput(); err != nil {
		t.Fatalf("preflight is not accepted by the platform Bash parser: %v\n%s", err, output)
	}

	missingBase := "refs/machinery-test/missing-trusted-base"
	cmd := exec.CommandContext(t.Context(), "/bin/bash", script)
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "PREFLIGHT_BASE_REF="+missingBase)
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "cannot resolve trusted aggregate-diff base "+missingBase) {
		t.Fatalf("missing trusted base did not fail before partial diff: err=%v out=%s", err, output)
	}
}

func TestShellGitOperationsUseBoundedSanitizedRunner(t *testing.T) {
	repo := repoRootDir(t)
	helper := mustRepositoryFile(t, filepath.Join(repo, "scripts", "git-safe", "main.go"))
	for _, required := range []string{
		"gitcontrol.Environment(environ)",
		"processcontrol.RunCapturedStreams",
		"context.WithTimeout",
		"successful Git command emitted stderr",
	} {
		if !strings.Contains(helper, required) {
			t.Errorf("bounded Git helper lacks contract %q", required)
		}
	}
	for _, rel := range []string{"scripts/preflight.sh", "scripts/modelith-inventory.sh", "scripts/modelith-render.sh"} {
		body := mustRepositoryFile(t, filepath.Join(repo, filepath.FromSlash(rel)))
		for _, required := range []string{"git-safe.sh", "git_safe"} {
			if !strings.Contains(body, required) {
				t.Errorf("%s lacks bounded Git adapter %q", rel, required)
			}
		}
		bareGit := regexp.MustCompile(`(?m)^[[:space:]]*git[[:space:]]+(merge-base|diff|status|rev-parse|rev-list|show|fetch)\b`)
		if match := bareGit.FindString(body); match != "" {
			t.Errorf("%s retains unbounded ambient Git invocation %q", rel, strings.TrimSpace(match))
		}
	}
	for _, rel := range []string{
		".github/workflows/ci.yml",
		".github/workflows/formal.yml",
		".github/workflows/nightly.yml",
		".github/workflows/release.yml",
	} {
		body := mustRepositoryFile(t, filepath.Join(repo, filepath.FromSlash(rel)))
		if !strings.Contains(body, "go run ./scripts/git-safe -root .") {
			t.Errorf("%s does not route correctness-critical Git through scripts/git-safe", rel)
		}
	}
}

func TestExternalEnginesUseBoundedClosedRunnerAndImmutableModelith(t *testing.T) {
	repo := repoRootDir(t)
	runner := mustRepositoryFile(t, filepath.Join(repo, "scripts", "run-safe", "main.go"))
	closure := mustRepositoryFile(t, filepath.Join(repo, "scripts", "run-safe", "closure.go"))
	for _, required := range []string{
		"context.WithTimeout",
		"processcontrol.RunCapturedStreamLimits",
		"successful command emitted stderr",
		"expect-stdout-file",
		"expect-stderr-file",
		"executable-receipt",
	} {
		if !strings.Contains(runner, required) {
			t.Errorf("bounded external-command runner lacks contract %q", required)
		}
	}
	for _, required := range []string{
		"regular non-symlink file",
		"maximumSymlinkDepth",
		"captureExecutableChain",
		"source executable symlink chain changed identity, metadata, or link target",
		"source executable changed identity, metadata, or content",
		"executable snapshot changed identity, metadata, or content",
		"nativeExecutableWitness",
	} {
		if !strings.Contains(closure, required) {
			t.Errorf("executable snapshot helper lacks contract %q", required)
		}
	}
	render := mustRepositoryFile(t, filepath.Join(repo, "scripts", "modelith-render.sh"))
	for _, required := range []string{
		"snapshot-executable -source \"$modelith_source\"",
		"-executable-receipt \"$modelith_receipt\"",
		"verify-executable -receipt \"$modelith_receipt\"",
	} {
		if !strings.Contains(render, required) {
			t.Errorf("Modelith render script lacks immutable executable contract %q", required)
		}
	}
	if strings.Count(render, "verify-executable -receipt \"$modelith_receipt\"") < 2 {
		t.Error("Modelith executable source is not revalidated on both sides of publication")
	}
	preflight := mustRepositoryFile(t, filepath.Join(repo, "scripts", "preflight.sh"))
	ci := mustRepositoryFile(t, filepath.Join(repo, ".github", "workflows", "ci.yml"))
	for name, body := range map[string]string{"preflight": preflight, "CI": ci} {
		for _, required := range []string{"./scripts/run-safe", "pull --quiet --platform", "image inspect --platform", "run --rm --pull=never"} {
			if !strings.Contains(body, required) {
				t.Errorf("%s external-engine gate lacks bounded runner contract %q", name, required)
			}
		}
	}
	for _, workflow := range []string{"ci.yml", "nightly.yml"} {
		body := mustRepositoryFile(t, filepath.Join(repo, ".github", "workflows", workflow))
		if !strings.Contains(body, "timeout-minutes:") {
			t.Errorf("%s lacks an explicit workflow job timeout", workflow)
		}
	}
}

func TestShellcheckInventoryIsClosedAndSharedByLocalAndCI(t *testing.T) {
	repo := repoRootDir(t)
	script := filepath.Join(repo, "scripts", "shellcheck-inventory.sh")
	cmd := exec.CommandContext(t.Context(), "bash", script)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("canonical ShellCheck inventory is invalid: %v\n%s", err, out)
	}
	for _, required := range []string{
		"install.sh",
		"hooks/machinery-hook.sh",
		"scripts/preflight.sh",
		"skills/machinery/tools/tlc.sh",
	} {
		if !strings.Contains(string(out), required+"\n") {
			t.Errorf("ShellCheck inventory omitted %s: %s", required, out)
		}
	}

	manifest := mustRepositoryFile(t, filepath.Join(repo, "scripts", "shellcheck-files.txt"))
	missing := strings.Replace(manifest, "hooks/machinery-hook.sh\n", "", 1)
	badManifest := filepath.Join(t.TempDir(), "shellcheck-files.txt")
	if err := os.WriteFile(badManifest, []byte(missing), 0o644); err != nil {
		t.Fatal(err)
	}
	bad := exec.CommandContext(t.Context(), "bash", script, badManifest)
	bad.Dir = repo
	if badOut, err := bad.CombinedOutput(); err == nil || !strings.Contains(string(badOut), "does not exactly match") {
		t.Fatalf("incomplete ShellCheck corpus did not fail reverse coverage: err=%v out=%s", err, badOut)
	}

	for _, rel := range []string{"scripts/preflight.sh", ".github/workflows/ci.yml"} {
		body := mustRepositoryFile(t, filepath.Join(repo, filepath.FromSlash(rel)))
		for _, required := range []string{"scripts/shellcheck-inventory.sh", ".shellcheck-version"} {
			if !strings.Contains(body, required) {
				t.Errorf("%s does not consume shared ShellCheck contract %q", rel, required)
			}
		}
	}
}

func TestAgentPortabilityDocumentationContracts(t *testing.T) {
	root := repoRootDir(t)
	readme := mustRepositoryFile(t, filepath.Join(root, "README.md"))
	if !strings.Contains(readme, "[agent portability guide](docs/agent-portability.md)") {
		t.Fatal("README must link the agent portability guide")
	}
	guide := mustRepositoryFile(t, filepath.Join(root, "docs", "agent-portability.md"))
	for _, required := range []string{
		"form one serialized transaction",
		"restores the complete previous local generation",
		"detected host plugin",
		"also makes the command fail",
	} {
		if !strings.Contains(guide, required) {
			t.Errorf("agent portability guide is missing recovery contract %q", required)
		}
	}
	for _, required := range []string{
		"machinery install --target all",
		"machinery update --version",
		"machinery doctor --target all",
		"machinery uninstall --target all",
		"Missing subagent support falls back",
		"OpenCode's event API",
		"CI remains authoritative",
	} {
		if !strings.Contains(guide, required) {
			t.Errorf("agent portability guide is missing %q", required)
		}
	}
}

func TestAcceptanceDocumentationUsesSatisfiableHistoryBinding(t *testing.T) {
	root := repoRootDir(t)
	for _, rel := range []string{
		"docs/acceptance-gate.md",
		"skills/machinery/references/build-md-template.md",
		"skills/machinery/references/verification-evidence.md",
	} {
		body := mustRepositoryFile(t, filepath.Join(root, filepath.FromSlash(rel)))
		for _, impossible := range []string{"--commit $(git rev-parse HEAD)", `--commit "${{ github.sha }}"`} {
			if strings.Contains(body, impossible) {
				t.Errorf("%s prescribes self-referential acceptance binding %q", rel, impossible)
			}
		}
		for _, required := range []string{"ancestr", "ORACLESET{"} {
			if !strings.Contains(body, required) {
				t.Errorf("%s is missing acceptance contract %q", rel, required)
			}
		}
	}
}

// TestRepositoryDeterminismSurfaceContracts holds the release, CI, and role
// prompts to the deterministic obligations they advertise. These files are
// executable interfaces even though they are not Go production code.
func TestRepositoryDeterminismSurfaceContracts(t *testing.T) {
	root := repoRootDir(t)
	requireAll := func(name, body string, required ...string) {
		t.Helper()
		for _, want := range required {
			if !strings.Contains(body, want) {
				t.Errorf("%s is missing deterministic contract %q", name, want)
			}
		}
	}

	formal := mustRepositoryFile(t, filepath.Join(root, ".github", "workflows", "formal.yml"))
	requireAll("formal workflow", formal,
		"scripts/example-inventory.sh formal",
		"go run ./scripts/git-safe -root . -- status --porcelain --untracked-files=all",
		"Assert formal generation left no diff",
	)
	nightly := mustRepositoryFile(t, filepath.Join(root, ".github", "workflows", "nightly.yml"))
	requireAll("nightly workflow", nightly,
		"scripts/example-inventory.sh formal",
		`oracle "$design/machines"`,
	)
	ci := mustRepositoryFile(t, filepath.Join(root, ".github", "workflows", "ci.yml"))
	requireAll("CI workflow", ci,
		"os: [macos-latest, windows-latest]",
		"go test -count=1 -run TestGolden ./cmd/machinery",
		"go install github.com/stacklok/modelith/cmd/modelith@v0.4.0",
		"make modelith-render-check",
		"Assert committed engine trust roots",
		"test -s .java-runtime-pin",
		"test -s .structurizr-pin",
		"verify-c4 \"$(dirname \"$dsl\")\"",
		"python@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc",
		"docker pull --quiet --platform \"$platform\" \"$image\"",
		"docker image inspect --platform \"$platform\" --format '{{json .RepoDigests}} {{.Os}}/{{.Architecture}}'",
		"docker run --rm --pull=never --platform \"$platform\"",
		"go build -o \"$runner\" ./scripts/run-safe",
		"timeout-minutes: 30",
		"MACHINERY_REQUIRE_OCI_GOLDEN: \"1\"",
		"Souffl(e|é).*(external.checker|checker engine|CI pin|required)",
		"stale host checker-runtime contract found",
		"scripts/example-inventory.sh rows",
		"scripts/example-inventory.sh checkers",
		`args+=(--impl "$impl")`,
		"args+=(--complete)",
		"go install github.com/rhysd/actionlint/cmd/actionlint@\"$version\"",
		"actionlint .github/workflows/*.yml",
		"scripts/shellcheck-inventory.sh",
		".shellcheck-linux-x86_64.sha256",
		`"$install_dir/shellcheck" "${shell_files[@]}"`,
	)
	actionlintPin := strings.TrimSpace(mustRepositoryFile(t, filepath.Join(root, ".actionlint-version")))
	if actionlintPin != "v1.7.11" {
		t.Fatalf("actionlint pin = %q, want v1.7.11", actionlintPin)
	}
	shellcheckPin := strings.TrimSpace(mustRepositoryFile(t, filepath.Join(root, ".shellcheck-version")))
	if shellcheckPin != "0.11.0" {
		t.Fatalf("ShellCheck pin = %q, want 0.11.0", shellcheckPin)
	}
	shellcheckChecksum := strings.TrimSpace(mustRepositoryFile(t, filepath.Join(root, ".shellcheck-linux-x86_64.sha256")))
	if shellcheckChecksum != "8c3be12b05d5c177a04c29e3c78ce89ac86f1595681cab149b65b97c4e227198" {
		t.Fatalf("ShellCheck checksum pin drifted: %q", shellcheckChecksum)
	}
	structurizrPin := mustRepositoryFile(t, filepath.Join(root, ".structurizr-pin"))
	requireAll("Structurizr pin", structurizrPin,
		"STRUCTURIZR_VERSION="+machversion.StructurizrVersion,
		"STRUCTURIZR_LINUX_ZIP_SHA256="+machversion.StructurizrLinuxZipSHA256,
	)
	javaPin := mustRepositoryFile(t, filepath.Join(root, ".java-runtime-pin"))
	requireAll("Java runtime pin", javaPin,
		"JAVA_RUNTIME_VERSION=21.0.12.1+1",
		"JAVA_RUNTIME_DARWIN_ARM64_SHA256=3623232f33a9c3baadf304480b2535f9a3cba8a58d42ecbb438ba267315d9998",
		"JAVA_RUNTIME_LINUX_AMD64_SHA256=ce79869e1307ed8ee1e2baa86a412b1eb5b75d10a01006d788a6f968bcfaee94",
	)

	release := mustRepositoryFile(t, filepath.Join(root, ".github", "workflows", "release.yml"))
	requireAll("release workflow", release,
		"group: release-${{ github.event_name == 'workflow_dispatch' && inputs.version || github.ref_name }}",
		"cancel-in-progress: false",
		"SOURCE_DATE_EPOCH",
		"go build -a -trimpath",
		"cmp \"$first/$binary\" \"$second/$binary\"",
		"refs/tags/${RELEASE_VERSION}:refs/tags/${RELEASE_VERSION}",
		"go run ./scripts/git-safe -root . -- rev-list -n 1 \"refs/tags/${RELEASE_VERSION}\"",
		`[ "$head_sha" != "$GITHUB_SHA" ]`,
		"go-version-file: go.mod",
		"go run ./cmd/release-archive",
		"-epoch \"$SOURCE_DATE_EPOCH\"",
		"-git-root .",
		"machinery-source.tar.gz",
		"Reject release/repository/plugin/binary version mismatch",
		"cmd-default:$cmd_default",
		"internal-default:$internal_default",
		"dist/machinery-linux-amd64 version",
		"LC_ALL=C sort | xargs sha256sum",
		"if-no-files-found: error",
		"Require the exact release artifact inventory",
		"find dist -mindepth 1 -maxdepth 1 -printf '%f\\n'",
		`if ! cmp -s "$expected" "$discovered"`,
		`[ -L "dist/$artifact" ]`,
	)
	packageStart := strings.Index(release, "      - name: Package tarball (pvg-compatible)")
	if packageStart < 0 {
		t.Fatal("release workflow has no pvg-compatible package step")
	}
	packageEnd := strings.Index(release[packageStart:], "\n      - uses:")
	if packageEnd < 0 {
		t.Fatal("release workflow package step has no terminating action")
	}
	packageStep := release[packageStart : packageStart+packageEnd]
	requireAll("release package step", packageStep,
		"TARGET_GOOS: ${{ matrix.goos }}",
		"TARGET_GOARCH: ${{ matrix.goarch }}",
		"machinery-${TARGET_GOOS}-${TARGET_GOARCH}",
	)
	for _, crossEnv := range []string{"\n          GOOS:", "\n          GOARCH:"} {
		if strings.Contains(packageStep, crossEnv) {
			t.Errorf("release package step leaks cross-build environment %q into host go run", strings.TrimSpace(crossEnv))
		}
	}

	fsm := mustRepositoryFile(t, filepath.Join(root, "agents", "machinery-fsm-author.md"))
	requireAll("FSM author role", fsm,
		"design/formal/<C>.semantics.yaml",
		"design/formal/<name>.composition.yaml",
		"machinery check design --gate g3",
		"machinery verify-formal design",
		"machinery verify-formal --gen-only design",
		"`\\* machinery:manual`",
		"same-basename `.cfg` sibling",
		"without its cfg",
	)
	build := mustRepositoryFile(t, filepath.Join(root, "agents", "machinery-build-writer.md"))
	requireAll("BUILD writer role", build,
		"design/formal/*.semantics.yaml",
		"design/formal/*.composition.yaml",
		"machinery check design --gate g3",
		"machinery verify-formal design",
		"g4.pack-event-discipline",
		"`\\* machinery:manual`",
	)
	for name, body := range map[string]string{"FSM author role": fsm, "BUILD writer role": build} {
		if strings.Contains(body, "make install") {
			t.Errorf("%s references nonexistent make install target", name)
		}
	}

	makefile := mustRepositoryFile(t, filepath.Join(root, "Makefile"))
	requireAll("Makefile formal suite", makefile,
		"EXAMPLE_INVENTORY := scripts/example-inventory.sh",
		"MODELITH_VERSION := v0.4.0",
		"MODELITH_RENDER := scripts/modelith-render.sh",
		"modelith-render-check: modelith-inventory modelith-render",
		"$(EXAMPLE_INVENTORY) rows",
		"$(EXAMPLE_INVENTORY) formal",
	)
	preflight := mustRepositoryFile(t, filepath.Join(root, "scripts", "preflight.sh"))
	requireAll("local preflight", preflight,
		"set -euo pipefail",
		"make modelith-render-check",
		"golangci-lint is required at the version pinned",
		"does not match pin $want",
		"scripts/example-inventory.sh rows",
		"scripts/example-inventory.sh checkers",
		"scripts/shellcheck-inventory.sh",
		`shellcheck "${shell_files[@]}"`,
		`go build -o "$run_safe" ./scripts/run-safe`,
		`"$docker_bin" pull --quiet --platform "$checker_platform" "$checker_image"`,
	)
	readme := mustRepositoryFile(t, filepath.Join(root, "README.md"))
	requireAll("README formal proof count", readme,
		"all 35 TLC proofs",
		"seven example designs that carry formal suites",
		"`\\* machinery:manual`",
		"declared/not-regenerated",
	)
}

// TestLeafCobraCommandsDeclareArgs prevents Cobra's permissive default from
// silently accepting and ignoring positional arguments. The AST walk makes
// this independent of formatting and variable names.
func TestLeafCobraCommandsDeclareArgs(t *testing.T) {
	root := filepath.Join(repoRootDir(t), "cmd", "machinery")
	fset := token.NewFileSet()
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			unary, ok := node.(*ast.UnaryExpr)
			if !ok || unary.Op != token.AND {
				return true
			}
			literal, ok := unary.X.(*ast.CompositeLit)
			if !ok {
				return true
			}
			selector, ok := literal.Type.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Command" {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if !ok || pkg.Name != "cobra" {
				return true
			}
			hasRun := false
			hasArgs := false
			for _, elt := range literal.Elts {
				field, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := field.Key.(*ast.Ident)
				if !ok {
					continue
				}
				switch key.Name {
				case "Run", "RunE":
					hasRun = true
				case "Args":
					hasArgs = true
				}
			}
			if hasRun && !hasArgs {
				t.Errorf("%s: executable Cobra command has Run/RunE but no Args validator", fset.Position(literal.Pos()))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Every mutating design publication must predeclare exact output identity.
// Keeping this as an AST repository contract prevents a new generator from
// silently falling back to the legacy path-only publication API.
func TestProductionDesignPublishersBindExactOutputs(t *testing.T) {
	root := repoRootDir(t)
	fset := token.NewFileSet()
	for _, subtree := range []string{"cmd", "internal"} {
		err := filepath.Walk(filepath.Join(root, subtree), func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			// Package-level Publish functions in unrelated transaction helpers
			// are not designlock publication callbacks. Method receivers are not
			// package imports, so this still catches snapshot.Publish and
			// snapshot.PublishExpected while avoiding false positives such as
			// cachestage.Publish.
			packageNames := make(map[string]bool)
			for _, spec := range file.Imports {
				if spec.Name != nil {
					if spec.Name.Name != "." && spec.Name.Name != "_" {
						packageNames[spec.Name.Name] = true
					}
					continue
				}
				packageNames[filepath.Base(strings.Trim(spec.Path.Value, `"`))] = true
			}
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || (selector.Sel.Name != "Publish" && selector.Sel.Name != "PublishExpected") {
					return true
				}
				if receiver, importedPackage := selector.X.(*ast.Ident); importedPackage && packageNames[receiver.Name] {
					return true
				}
				t.Errorf("%s: production writer uses an unrooted publication callback; use PublishExpectedRooted with exact output identity and retained output capabilities", fset.Position(call.Pos()))
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// Every executable Cobra command must join stdout/stderr failures into its
// returned status. This closes the class where successful mutation followed
// by a broken pipe was reported as success, and keeps new command classes
// inside the same checked-output contract automatically.
func TestEveryProductionRunETracksOutputFailures(t *testing.T) {
	root := filepath.Join(repoRootDir(t), "cmd", "machinery")
	fset := token.NewFileSet()
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		check := func(literal *ast.FuncLit) {
			tracked := false
			ast.Inspect(literal.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				name, ok := call.Fun.(*ast.Ident)
				if ok && (name.Name == "trackCommandOutput" || name.Name == "trackOutput") {
					tracked = true
				}
				return true
			})
			if !tracked {
				t.Errorf("%s: Cobra RunE does not join checked output failures", fset.Position(literal.Pos()))
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch current := node.(type) {
			case *ast.KeyValueExpr:
				key, ok := current.Key.(*ast.Ident)
				literal, isFunc := current.Value.(*ast.FuncLit)
				if ok && isFunc && key.Name == "RunE" {
					check(literal)
				}
			case *ast.AssignStmt:
				for i, lhs := range current.Lhs {
					selector, ok := lhs.(*ast.SelectorExpr)
					if !ok || selector.Sel.Name != "RunE" || i >= len(current.Rhs) {
						continue
					}
					if literal, ok := current.Rhs[i].(*ast.FuncLit); ok {
						check(literal)
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func mustRepositoryFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// The stamp default in internal/version must match the binary's default:
// main() copies the ldflags value over at startup, but in-process library use
// (tests, gates) sees the internal default, and a drift between the two would
// stamp artifacts with a version the binary never reports.
func TestInternalVersionDefaultMatchesBinaryDefault(t *testing.T) {
	if machversion.Version != version {
		t.Fatalf("internal/version default %q != cmd/machinery default %q", machversion.Version, version)
	}
}
