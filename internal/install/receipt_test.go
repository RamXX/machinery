package install

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInstallRecordsCustomHomesAndNativeTargets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override does not steer os.UserHomeDir on Windows")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", t.TempDir())
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

func TestBuildRefreshPlanMigratesLegacyStandardTopology(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink topology test is POSIX-specific")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", t.TempDir())
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
	t.Setenv("MACHINERY_CONFIG_DIR", t.TempDir())
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

func TestSemanticallyInvalidReceiptFailsBeforeUpdate(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MACHINERY_CONFIG_DIR", dir)
	write(t, filepath.Join(dir, "install.json"), `{"schema_version":1,"targets":[{"target":"cursor"}]}`)
	if _, _, err := loadReceipt(); err == nil {
		t.Fatal("unknown receipt target must fail before binary replacement")
	}
}
