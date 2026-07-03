package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestInstallScript exercises install.sh offline: it points the script at the
// working tree as the skill source (MACHINERY_SKILL_SRC), skips the binary
// download (MACHINERY_SKIP_BINARY), and asserts the canonical-copy + symlink
// topology it lays down across two agent homes. This locks the installer's
// contract: real files in the first home, symlinks to it in the rest.
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
	agents := filepath.Join(home, ".agents") // canonical (first home)
	claude := filepath.Join(home, ".claude") // symlinked to canonical

	cmd := exec.Command(sh, script)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"MACHINERY_SKIP_BINARY=1",
		"MACHINERY_SKILL_SRC="+root,
		"MACHINERY_HOMES="+agents+" "+claude,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}

	// Canonical home holds the real skill directory (not a symlink).
	canonSkill := filepath.Join(agents, "skills", "machinery")
	if fi, err := os.Lstat(canonSkill); err != nil {
		t.Fatalf("canonical skill missing: %v", err)
	} else if fi.Mode()&os.ModeSymlink != 0 {
		t.Errorf("canonical skill should be a real directory, got a symlink: %s", canonSkill)
	}
	if _, err := os.Stat(filepath.Join(canonSkill, "SKILL.md")); err != nil {
		t.Errorf("canonical skill has no SKILL.md: %v", err)
	}

	// Secondary home symlinks to the canonical copy.
	linkSkill := filepath.Join(claude, "skills", "machinery")
	if lfi, err := os.Lstat(linkSkill); err != nil {
		t.Fatalf("secondary skill link missing: %v", err)
	} else if lfi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("secondary skill should be a symlink: %s", linkSkill)
	}
	if target, err := os.Readlink(linkSkill); err != nil {
		t.Fatalf("readlink: %v", err)
	} else if target != canonSkill {
		t.Errorf("symlink target = %s, want %s", target, canonSkill)
	}

	// Role docs: real files in canonical home, symlinks in the secondary.
	for _, name := range []string{"machinery-fsm-author.md", "machinery-build-writer.md"} {
		real := filepath.Join(agents, "agents", name)
		if rfi, err := os.Lstat(real); err != nil {
			t.Errorf("canonical agent %s missing: %v", name, err)
		} else if rfi.Mode()&os.ModeSymlink != 0 {
			t.Errorf("canonical agent %s should be a real file, got a symlink", name)
		}
		link := filepath.Join(claude, "agents", name)
		if lfi, err := os.Lstat(link); err != nil {
			t.Errorf("secondary agent %s missing: %v", name, err)
		} else if lfi.Mode()&os.ModeSymlink == 0 {
			t.Errorf("secondary agent %s should be a symlink", name)
		}
	}
}
