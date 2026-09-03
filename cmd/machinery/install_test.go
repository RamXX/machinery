package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// assertTopology checks the canonical-copy + symlink layout across two homes:
// real skill dir + role docs in canon, symlinks to them in secondary.
func assertTopology(t *testing.T, canon, secondary string) {
	t.Helper()
	canonSkill := filepath.Join(canon, "skills", "machinery")
	if fi, err := os.Lstat(canonSkill); err != nil {
		t.Fatalf("canonical skill missing: %v", err)
	} else if fi.Mode()&os.ModeSymlink != 0 {
		t.Errorf("canonical skill should be a real directory, got a symlink")
	}
	if _, err := os.Stat(filepath.Join(canonSkill, "SKILL.md")); err != nil {
		t.Errorf("canonical skill has no SKILL.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(canonSkill, "references", "rebuild-guide.md")); err != nil {
		t.Errorf("canonical skill has no installed rebuild reference: %v", err)
	}

	linkSkill := filepath.Join(secondary, "skills", "machinery")
	if fi, err := os.Lstat(linkSkill); err != nil {
		t.Fatalf("secondary skill link missing: %v", err)
	} else if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("secondary skill should be a symlink")
	}
	if target, err := os.Readlink(linkSkill); err != nil {
		t.Fatalf("readlink: %v", err)
	} else if target != canonSkill {
		t.Errorf("symlink target = %s, want %s", target, canonSkill)
	}

	for _, name := range []string{"machinery-fsm-author.md", "machinery-build-writer.md"} {
		if fi, err := os.Lstat(filepath.Join(canon, "agents", name)); err != nil {
			t.Errorf("canonical role doc %s missing: %v", name, err)
		} else if fi.Mode()&os.ModeSymlink != 0 {
			t.Errorf("canonical role doc %s should be a real file", name)
		}
		if fi, err := os.Lstat(filepath.Join(secondary, "agents", name)); err != nil {
			t.Errorf("secondary role doc %s missing: %v", name, err)
		} else if fi.Mode()&os.ModeSymlink == 0 {
			t.Errorf("secondary role doc %s should be a symlink", name)
		}
	}
}

// TestInstallCommand drives `machinery install` directly against the working
// tree (--from) and asserts the topology, then checks uninstall removes it.
func TestInstallCommand(t *testing.T) {
	t.Parallel()
	root := repoRootDir(t)
	config := privateTestConfigDir(t)
	environment := []string{"MACHINERY_CONFIG_DIR=" + config}
	home := filepath.Join(t.TempDir(), "home with spaces")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	agents := filepath.Join(home, ".agents")
	claude := filepath.Join(home, ".claude")

	if _, errS, code := runBinWithEnv(t, environment, "install", "--from", root, "--home", agents, "--home", claude); code != 0 {
		t.Fatalf("machinery install exited %d: %s", code, errS)
	}
	assertTopology(t, agents, claude)

	if _, errS, code := runBinWithEnv(t, environment, "uninstall", "--home", agents, "--home", claude); code != 0 {
		t.Fatalf("machinery uninstall exited %d: %s", code, errS)
	}
	if _, err := os.Lstat(filepath.Join(agents, "skills", "machinery")); !os.IsNotExist(err) {
		t.Errorf("skill still present after uninstall (err=%v)", err)
	}
}

func TestInstallAndDoctorTargetAll(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override does not steer agent config paths on Windows")
	}
	t.Parallel()
	root := repoRootDir(t)
	// Build before HOME is redirected; otherwise Go's default module cache is
	// created beneath the fixture home with read-only module files that the
	// testing package cannot remove on cleanup.
	_ = goldenBin(t)
	home := t.TempDir()
	// Install, doctor, and uninstall are one topology lifecycle and therefore
	// must share the authoritative schema-2 receipt location.
	environment := []string{"HOME=" + home, "MACHINERY_CONFIG_DIR=" + privateTestConfigDir(t), fakeModelithPath(t)}

	if out, errS, code := runBinWithEnv(t, environment, "install", "--from", root, "--target", "all"); code != 0 {
		t.Fatalf("machinery install --target all exited %d: %s\n%s", code, errS, out)
	}
	for _, path := range []string{
		filepath.Join(home, ".agents", "skills", "machinery", "SKILL.md"),
		filepath.Join(home, ".claude", "agents", "machinery-fsm-author.md"),
		filepath.Join(home, ".codex", "agents", "machinery-fsm-author.toml"),
		filepath.Join(home, ".config", "opencode", "agents", "machinery-fsm-author.md"),
		filepath.Join(home, ".config", "opencode", "commands", "design.md"),
		filepath.Join(home, ".config", "opencode", "plugins", "machinery.js"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("target install missing %s: %v", path, err)
		}
	}

	out, errS, code := runBinWithEnv(t, environment, "doctor", "--target", "all")
	if code != 0 {
		t.Fatalf("machinery doctor --target all exited %d: %s", code, errS)
	}
	for _, marker := range []string{"[claude]", "[codex]", "[opencode]", "[shared]"} {
		if !strings.Contains(out, marker) {
			t.Errorf("doctor output missing %s:\n%s", marker, out)
		}
	}

	if out, errS, code := runBinWithEnv(t, environment, "uninstall", "--target", "all"); code != 0 {
		t.Fatalf("machinery uninstall --target all exited %d: %s\n%s", code, errS, out)
	}
	for _, path := range []string{
		filepath.Join(home, ".agents", "skills", "machinery"),
		filepath.Join(home, ".claude", "agents", "machinery-fsm-author.md"),
		filepath.Join(home, ".codex", "agents", "machinery-fsm-author.toml"),
		filepath.Join(home, ".config", "opencode", "plugins", "machinery.js"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("target uninstall left %s (err=%v)", path, err)
		}
	}
}

// TestInstallScript exercises the install.sh bootstrap end to end offline: it
// hands the script the built binary (MACHINERY_BIN) and the working tree as the
// skill source (MACHINERY_SKILL_SRC), so it delegates to `machinery install`
// with no network, and asserts the resulting topology.
func TestInstallScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is a POSIX shell installer")
	}
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no POSIX sh available")
	}
	root := repoRootDir(t)
	script := filepath.Join(root, "install.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("install.sh not found at %s: %v", script, err)
	}

	home := t.TempDir()
	agents := filepath.Join(home, ".agents")
	claude := filepath.Join(home, ".claude")

	cmd := exec.CommandContext(t.Context(), sh, script)
	cmd.Env = environmentWithOverrides(os.Environ(), []string{
		"HOME=" + home,
		"MACHINERY_CONFIG_DIR=" + privateTestConfigDir(t),
		"MACHINERY_BIN=" + goldenBin(t),
		"MACHINERY_SKILL_SRC=" + root,
		"MACHINERY_HOMES=" + agents + "\n" + claude,
		fakeModelithPath(t),
	})
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}
	assertTopology(t, agents, claude)
}

func TestInstallScriptHostTargets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is a POSIX shell installer")
	}
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no POSIX sh available")
	}
	root := repoRootDir(t)
	home := t.TempDir()
	config := privateTestConfigDir(t)
	cmd := exec.CommandContext(t.Context(), sh, filepath.Join(root, "install.sh"))
	cmd.Env = environmentWithOverrides(os.Environ(), []string{
		"HOME=" + home,
		"MACHINERY_CONFIG_DIR=" + config,
		"MACHINERY_BIN=" + goldenBin(t),
		"MACHINERY_SKILL_SRC=" + root,
		"MACHINERY_TARGETS=codex opencode",
		fakeModelithPath(t),
	})
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("targeted install.sh failed: %v\n%s", err, out)
	}
	for _, path := range []string{
		filepath.Join(home, ".agents", "skills", "machinery", "SKILL.md"),
		filepath.Join(home, ".codex", "agents", "machinery-fsm-author.toml"),
		filepath.Join(home, ".config", "opencode", "commands", "design.md"),
		filepath.Join(home, ".config", "opencode", "plugins", "machinery.js"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("targeted bootstrap missing %s: %v", path, err)
		}
	}
	assertReceiptTargets(t, filepath.Join(config, "install.json"), "codex", "opencode")
}

func fakeModelithPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "modelith")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n[ \"${1:-}\" = --version ] || exit 2\nprintf 'modelith version 0.4.0\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return "PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")
}

func assertReceiptTargets(t *testing.T, path string, expected ...string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read isolated install receipt: %v", err)
	}
	var receipt struct {
		SchemaVersion int `json:"schema_version"`
		Targets       []struct {
			Target string `json:"target"`
		} `json:"targets"`
		HomeInstalls []json.RawMessage `json:"home_installs"`
	}
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatalf("decode isolated install receipt: %v", err)
	}
	if receipt.SchemaVersion != 2 {
		t.Fatalf("install receipt schema = %d, want 2", receipt.SchemaVersion)
	}
	if len(receipt.HomeInstalls) != 0 {
		t.Fatalf("target-only receipt was contaminated by %d home installs", len(receipt.HomeInstalls))
	}
	if len(receipt.Targets) != len(expected) {
		t.Fatalf("install receipt targets = %#v, want %v", receipt.Targets, expected)
	}
	for index, want := range expected {
		if got := receipt.Targets[index].Target; got != want {
			t.Fatalf("install receipt target[%d] = %q, want %q", index, got, want)
		}
	}
}

func TestInstallScriptPropagatesPreflightFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is a POSIX shell installer")
	}
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no POSIX sh available")
	}
	root := repoRootDir(t)
	home := t.TempDir()
	fake := filepath.Join(t.TempDir(), "machinery")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\ncase \"$1\" in install) exit 0;; preflight) exit 23;; *) exit 2;; esac\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), sh, filepath.Join(root, "install.sh"))
	cmd.Env = environmentWithOverrides(os.Environ(), []string{
		"HOME=" + home,
		"MACHINERY_CONFIG_DIR=" + privateTestConfigDir(t),
		"MACHINERY_BIN=" + fake,
		"MACHINERY_SKILL_SRC=" + root,
		"MACHINERY_HOMES=" + filepath.Join(home, ".agents"),
	})
	if output, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("install.sh masked preflight failure:\n%s", output)
	}
}
