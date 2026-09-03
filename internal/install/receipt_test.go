package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestReceiptReadModifyWriteIsSerialized(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	root := t.TempDir()
	const count = 16
	for i := 0; i < count; i++ {
		seedHomeArtifactInventory(t, filepath.Join(root, fmt.Sprintf("home-%02d", i)))
	}
	var wg sync.WaitGroup
	errCh := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errCh <- recordHomeInstall([]string{filepath.Join(root, fmt.Sprintf("home-%02d", i))}, false)
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	receipt, _, err := loadReceipt()
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.HomeInstalls) != count {
		t.Fatalf("receipt retained %d/%d concurrent updates", len(receipt.HomeInstalls), count)
	}
}

func TestLoadReceiptRejectsSymlinkOversizeAndUnstableSwap(t *testing.T) {
	t.Run("symlink", func(t *testing.T) {
		config := privateConfigDir(t)
		t.Setenv("MACHINERY_CONFIG_DIR", config)
		outside := filepath.Join(t.TempDir(), "outside.json")
		if err := os.WriteFile(outside, []byte(`{"schema_version":2}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(config, "install.json")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, exists, err := loadReceipt(); !exists || err == nil || !strings.Contains(err.Error(), "private regular file") {
			t.Fatalf("symlink receipt: exists=%v err=%v", exists, err)
		}
	})

	t.Run("oversize", func(t *testing.T) {
		config := privateConfigDir(t)
		t.Setenv("MACHINERY_CONFIG_DIR", config)
		if err := os.WriteFile(filepath.Join(config, "install.json"), make([]byte, receiptMaxBytes+1), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, exists, err := loadReceipt(); !exists || err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("oversize receipt: exists=%v err=%v", exists, err)
		}
	})

	t.Run("config directory swap", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("fixture uses POSIX rename-over-open-directory semantics")
		}
		base := t.TempDir()
		config := filepath.Join(base, "config")
		if err := os.Mkdir(config, 0o700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("MACHINERY_CONFIG_DIR", config)
		writeLegacyReceipt(t, installReceipt{})
		outside := t.TempDir()
		sentinel := filepath.Join(outside, "sentinel")
		write(t, sentinel, "outside")
		parked := filepath.Join(base, "config-parked")
		afterReceiptRootOpen = func() {
			afterReceiptRootOpen = nil
			if err := os.Rename(config, parked); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, config); err != nil {
				t.Fatal(err)
			}
		}
		t.Cleanup(func() {
			afterReceiptRootOpen = nil
			_ = os.Remove(config)
			_ = os.Rename(parked, config)
		})
		if _, exists, err := loadReceipt(); !exists || err == nil || !strings.Contains(err.Error(), "changed during read") {
			t.Fatalf("directory swap receipt: exists=%v err=%v", exists, err)
		}
		if got, err := os.ReadFile(sentinel); err != nil || string(got) != "outside" {
			t.Fatalf("outside sentinel changed: %q, %v", got, err)
		}
	})

	t.Run("entry swap", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("fixture uses POSIX rename-over-open-file semantics")
		}
		config := privateConfigDir(t)
		t.Setenv("MACHINERY_CONFIG_DIR", config)
		writeLegacyReceipt(t, installReceipt{})
		path := filepath.Join(config, "install.json")
		parked := path + ".parked"
		outside := filepath.Join(t.TempDir(), "outside.json")
		if err := os.WriteFile(outside, []byte(`{"schema_version":2}`), 0o600); err != nil {
			t.Fatal(err)
		}
		afterReceiptEntryLstat = func() {
			afterReceiptEntryLstat = nil
			if err := os.Rename(path, parked); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, path); err != nil {
				t.Fatal(err)
			}
		}
		t.Cleanup(func() {
			afterReceiptEntryLstat = nil
			_ = os.Remove(path)
			_ = os.Rename(parked, path)
		})
		if _, exists, err := loadReceipt(); !exists || err == nil {
			t.Fatalf("entry swap receipt: exists=%v err=%v", exists, err)
		}
		if got, err := os.ReadFile(outside); err != nil || string(got) != `{"schema_version":2}` {
			t.Fatalf("outside receipt changed: %q, %v", got, err)
		}
	})
}

func TestInstallRecordsCustomHomesAndNativeTargets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override does not steer os.UserHomeDir on Windows")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	t.Setenv("HOME", home)
	src := fakeSource(t)
	customA := filepath.Join(home, "custom-a")
	customB := filepath.Join(home, "custom-b")

	if err := Install(Options{Homes: []string{customA, customB}, From: src, Record: true}); err != nil {
		t.Fatal(err)
	}
	if err := Install(Options{Targets: []string{"codex", "opencode"}, From: src, Copy: true, Record: true}); err != nil {
		t.Fatal(err)
	}
	receipt, exists, err := loadReceipt()
	if err != nil {
		t.Fatal(err)
	}
	if !exists || receipt.SchemaVersion != receiptSchema {
		t.Fatalf("receipt = %+v, exists=%v", receipt, exists)
	}
	if len(receipt.HomeInstalls) != 1 || len(receipt.HomeInstalls[0].Homes) != 2 || receipt.HomeInstalls[0].Copy {
		t.Fatalf("home receipt = %+v", receipt.HomeInstalls)
	}
	if len(receipt.Targets) != 2 || !receipt.Targets[0].Copy || !receipt.Targets[1].Copy {
		t.Fatalf("target receipt = %+v", receipt.Targets)
	}

	if err := ForgetTargetInstalls([]string{"opencode"}); err != nil {
		t.Fatal(err)
	}
	if err := ForgetHomeInstalls([]string{customB}); err != nil {
		t.Fatal(err)
	}
	receipt, _, err = loadReceipt()
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Targets) != 1 || receipt.Targets[0].Target != "codex" {
		t.Fatalf("target receipt after removal = %+v", receipt.Targets)
	}
	if len(receipt.HomeInstalls) != 1 || len(receipt.HomeInstalls[0].Homes) != 1 || receipt.HomeInstalls[0].Homes[0] != customA {
		t.Fatalf("home receipt after secondary removal = %+v", receipt.HomeInstalls)
	}
}

func TestForgetReceiptUsesNativeCaseAliasIdentity(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("case-folded path identity is a Darwin/Windows contract")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	root := t.TempDir()
	recorded := filepath.Join(root, "Recorded", "Home")
	seedHomeArtifactInventory(t, recorded)
	if err := recordHomeInstall([]string{recorded}, false); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "recorded", "home")
	if err := ForgetHomeInstalls([]string{alias}); err != nil {
		t.Fatal(err)
	}
	_, exists, err := loadReceipt()
	if err != nil || exists {
		t.Fatalf("case-alias receipt was not removed: exists=%v err=%v", exists, err)
	}
}

func seedHomeArtifactInventory(t *testing.T, home string) {
	t.Helper()
	write(t, filepath.Join(home, "skills", "machinery", "SKILL.md"), "seed skill")
	for _, doc := range RoleDocs {
		write(t, filepath.Join(home, "agents", doc), "seed role")
	}
}

func writeLegacyReceipt(t *testing.T, receipt installReceipt) {
	t.Helper()
	receipt.SchemaVersion = 1
	receipt.Artifacts = nil
	path, err := installationReceiptPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestBuildRefreshPlanMigratesLegacyStandardTopology(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink topology test is POSIX-specific")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	t.Setenv("HOME", home)
	agentsSkill := filepath.Join(home, ".agents", "skills", "machinery")
	if err := os.MkdirAll(agentsSkill, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(agentsSkill, "SKILL.md"), "skill")
	claudeSkills := filepath.Join(home, ".claude", "skills")
	if err := os.MkdirAll(claudeSkills, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(agentsSkill, filepath.Join(claudeSkills, "machinery")); err != nil {
		t.Fatal(err)
	}

	plan, err := buildRefreshPlan()
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.HomeInstalls) != 1 || plan.HomeInstalls[0].Copy || len(plan.HomeInstalls[0].Homes) != 2 {
		t.Fatalf("legacy plan = %+v", plan.HomeInstalls)
	}
}

func TestBuildRefreshPlanRecognizesNativeTargetTopology(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override does not steer os.UserHomeDir on Windows")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	t.Setenv("HOME", home)
	write(t, filepath.Join(home, ".agents", "skills", "machinery", "SKILL.md"), "skill")
	write(t, filepath.Join(home, ".codex", "agents", "machinery-fsm-author.toml"), "agent")

	plan, err := buildRefreshPlan()
	if err != nil {
		t.Fatal(err)
	}
	if !plan.hasTarget(TargetCodex) {
		t.Fatalf("Codex target not discovered: %+v", plan.Targets)
	}
	if len(plan.HomeInstalls) != 0 {
		t.Fatalf("shared target skill must not be misclassified as a legacy home: %+v", plan.HomeInstalls)
	}
}

func TestCorruptReceiptFailsLoudly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MACHINERY_CONFIG_DIR", dir)
	write(t, filepath.Join(dir, "install.json"), "{not-json")
	if _, _, err := loadReceipt(); err == nil {
		t.Fatal("corrupt receipt must not be silently replaced")
	}
}

func TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"unknown root", `{"schema_version":1,"home_installz":[]}`},
		{"unknown home field", `{"schema_version":1,"home_installs":[{"homes":[],"copies":true}]}`},
		{"unknown target field", `{"schema_version":1,"targets":[{"target":"codex","copied":true}]}`},
		{"duplicate root", `{"schema_version":1,"schema_version":1}`},
		{"duplicate nested", `{"schema_version":1,"targets":[{"target":"codex","target":"opencode"}]}`},
		{"wrong homes type", `{"schema_version":1,"home_installs":[{"homes":"/tmp/home"}]}`},
		{"wrong copy type", `{"schema_version":1,"targets":[{"target":"codex","copy":"yes"}]}`},
		{"trailing value", `{"schema_version":1} {"schema_version":1}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("MACHINERY_CONFIG_DIR", dir)
			write(t, filepath.Join(dir, "install.json"), tc.raw)
			if _, _, err := loadReceipt(); err == nil {
				t.Fatalf("receipt accepted: %s", tc.raw)
			}
		})
	}
}

func TestSemanticallyInvalidReceiptFailsBeforeUpdate(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MACHINERY_CONFIG_DIR", dir)
	write(t, filepath.Join(dir, "install.json"), `{"schema_version":1,"targets":[{"target":"cursor"}]}`)
	if _, _, err := loadReceipt(); err == nil {
		t.Fatal("unknown receipt target must fail before binary replacement")
	}
}
