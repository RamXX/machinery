package main

import (
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
	root := repoRootDir(t)
	home := t.TempDir()
	agents := filepath.Join(home, ".agents")
	claude := filepath.Join(home, ".claude")

	if _, errS, code := runBin(t, "install", "--from", root, "--home", agents, "--home", claude); code != 0 {
		t.Fatalf("machinery install exited %d: %s", code, errS)
	}
	assertTopology(t, agents, claude)

	if _, errS, code := runBin(t, "uninstall", "--home", agents, "--home", claude); code != 0 {
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
	root := repoRootDir(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	if out, errS, code := runBin(t, "install", "--from", root, "--target", "all"); code != 0 {
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

	out, errS, code := runBin(t, "doctor", "--target", "all")
	if code != 0 {
		t.Fatalf("machinery doctor --target all exited %d: %s", code, errS)
	}
	for _, marker := range []string{"[claude]", "[codex]", "[opencode]", "[shared]"} {
		if !strings.Contains(out, marker) {
			t.Errorf("doctor output missing %s:\n%s", marker, out)
		}
	}

	if out, errS, code := runBin(t, "uninstall", "--target", "all"); code != 0 {
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
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"MACHINERY_BIN="+goldenBin(t),
		"MACHINERY_SKILL_SRC="+root,
		"MACHINERY_HOMES="+agents+" "+claude,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}
	assertTopology(t, agents, claude)
}

func TestInstallScriptHostTargets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is a POSIX shell installer")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no POSIX sh available")
	}
	root := repoRootDir(t)
	home := t.TempDir()
	cmd := exec.CommandContext(t.Context(), sh, filepath.Join(root, "install.sh"))
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"MACHINERY_BIN="+goldenBin(t),
		"MACHINERY_SKILL_SRC="+root,
		"MACHINERY_TARGETS=codex opencode",
	)
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
}
