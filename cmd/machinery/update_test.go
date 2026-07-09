package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateCommandHelpAndPreflightValidation(t *testing.T) {
	out, errOut, code := runBin(t, "update", "--help")
	if code != 0 {
		t.Fatalf("update --help exited %d: %s", code, errOut)
	}
	for _, flag := range []string{"--version", "--install-dir", "--home", "--target", "--skip-plugins"} {
		if !strings.Contains(out, flag) {
			t.Errorf("update help missing %s:\n%s", flag, out)
		}
	}
	_, errOut, code = runBin(t, "update", "--copy")
	if code == 0 || !strings.Contains(errOut, "--copy requires") {
		t.Fatalf("update --copy without selectors: code=%d stderr=%q", code, errOut)
	}
}

func TestInstallAndUninstallMaintainUpdateReceipt(t *testing.T) {
	root := repoRootDir(t)
	config := t.TempDir()
	t.Setenv("MACHINERY_CONFIG_DIR", config)
	home := filepath.Join(t.TempDir(), "agent-home")
	if _, errOut, code := runBin(t, "install", "--from", root, "--home", home); code != 0 {
		t.Fatalf("install exited %d: %s", code, errOut)
	}
	receipt := filepath.Join(config, "install.json")
	raw, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatalf("installation receipt missing: %v", err)
	}
	if !strings.Contains(string(raw), home) || !strings.Contains(string(raw), `"schema_version": 1`) {
		t.Fatalf("receipt does not describe the custom home:\n%s", raw)
	}
	if _, errOut, code := runBin(t, "uninstall", "--home", home); code != 0 {
		t.Fatalf("uninstall exited %d: %s", code, errOut)
	}
	if _, err := os.Stat(receipt); !os.IsNotExist(err) {
		t.Fatalf("empty receipt remains after uninstall: %v", err)
	}
}
