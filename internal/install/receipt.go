package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const receiptSchema = 1

type homeInstall struct {
	Homes []string `json:"homes"`
	Copy  bool     `json:"copy,omitempty"`
}

type targetInstall struct {
	Target string `json:"target"`
	Copy   bool   `json:"copy,omitempty"`
}

type installReceipt struct {
	SchemaVersion int             `json:"schema_version"`
	HomeInstalls  []homeInstall   `json:"home_installs,omitempty"`
	Targets       []targetInstall `json:"targets,omitempty"`
}

type refreshPlan struct {
	HomeInstalls []homeInstall
	Targets      []targetInstall
	ClaudePlugin bool
}

// ReceiptStatus is the user-facing summary reported by machinery doctor.
type ReceiptStatus struct {
	Path         string
	Exists       bool
	HomeInstalls int
	Targets      int
}

// InstallationReceiptStatus reports which direct placements `machinery
// update` will remember without exposing the receipt's internal schema.
func InstallationReceiptStatus() (ReceiptStatus, error) {
	path, err := installationReceiptPath()
	if err != nil {
		return ReceiptStatus{}, err
	}
	receipt, exists, err := loadReceipt()
	if err != nil {
		return ReceiptStatus{Path: path, Exists: exists}, err
	}
	return ReceiptStatus{
		Path:         path,
		Exists:       exists,
		HomeInstalls: len(receipt.HomeInstalls),
		Targets:      len(receipt.Targets),
	}, nil
}

func installationReceiptPath() (string, error) {
	if dir := os.Getenv("MACHINERY_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "install.json"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "machinery", "install.json"), nil
}

func loadReceipt() (installReceipt, bool, error) {
	path, err := installationReceiptPath()
	if err != nil {
		return installReceipt{}, false, err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return installReceipt{SchemaVersion: receiptSchema}, false, nil
	}
	if err != nil {
		return installReceipt{}, false, err
	}
	var receipt installReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return installReceipt{}, true, fmt.Errorf("parse installation receipt %s: %w", path, err)
	}
	if receipt.SchemaVersion != receiptSchema {
		return installReceipt{}, true, fmt.Errorf("installation receipt %s uses schema %d; this binary supports schema %d", path, receipt.SchemaVersion, receiptSchema)
	}
	normalizeReceipt(&receipt)
	if err := validateReceipt(receipt); err != nil {
		return installReceipt{}, true, fmt.Errorf("invalid installation receipt %s: %w", path, err)
	}
	return receipt, true, nil
}

func saveReceipt(receipt installReceipt) error {
	receipt.SchemaVersion = receiptSchema
	normalizeReceipt(&receipt)
	path, err := installationReceiptPath()
	if err != nil {
		return err
	}
	if len(receipt.HomeInstalls) == 0 && len(receipt.Targets) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".install-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return renameReplace(tmpPath, path)
}

func recordHomeInstall(homes []string, copyAll bool) error {
	abs, err := absHomes(homes)
	if err != nil {
		return err
	}
	if len(abs) == 0 {
		return nil
	}
	receipt, _, err := loadReceipt()
	if err != nil {
		return err
	}
	next := homeInstall{Homes: abs, Copy: copyAll}
	replaced := false
	for i := range receipt.HomeInstalls {
		if len(receipt.HomeInstalls[i].Homes) > 0 && receipt.HomeInstalls[i].Homes[0] == abs[0] {
			receipt.HomeInstalls[i] = next
			replaced = true
			break
		}
	}
	if !replaced {
		receipt.HomeInstalls = append(receipt.HomeInstalls, next)
	}
	return saveReceipt(receipt)
}

func recordTargetInstall(names []string, copyAll bool) error {
	set, err := parseTargets(names)
	if err != nil {
		return err
	}
	receipt, _, err := loadReceipt()
	if err != nil {
		return err
	}
	byName := map[string]targetInstall{}
	for _, target := range receipt.Targets {
		byName[target.Target] = target
	}
	for _, target := range targetOrder {
		if set[target] {
			byName[string(target)] = targetInstall{Target: string(target), Copy: copyAll}
		}
	}
	receipt.Targets = receipt.Targets[:0]
	for _, target := range targetOrder {
		if record, ok := byName[string(target)]; ok {
			receipt.Targets = append(receipt.Targets, record)
		}
	}
	return saveReceipt(receipt)
}

// ForgetHomeInstalls removes physically uninstalled homes from the update
// receipt. Removing a canonical home drops its whole symlink group; removing a
// secondary home keeps the remaining group intact.
func ForgetHomeInstalls(homes []string) error {
	abs, err := absHomes(homes)
	if err != nil {
		return err
	}
	if len(abs) == 0 {
		abs, err = absHomes(DefaultHomes())
		if err != nil {
			return err
		}
	}
	remove := map[string]bool{}
	for _, home := range abs {
		remove[home] = true
	}
	receipt, exists, err := loadReceipt()
	if err != nil || !exists {
		return err
	}
	var groups []homeInstall
	for _, group := range receipt.HomeInstalls {
		if len(group.Homes) == 0 || remove[group.Homes[0]] {
			continue
		}
		kept := group.Homes[:0]
		for _, home := range group.Homes {
			if !remove[home] {
				kept = append(kept, home)
			}
		}
		if len(kept) > 0 {
			group.Homes = kept
			groups = append(groups, group)
		}
	}
	receipt.HomeInstalls = groups
	return saveReceipt(receipt)
}

// ForgetTargetInstalls removes host-native adapters from the update receipt.
func ForgetTargetInstalls(names []string) error {
	set, err := parseTargets(names)
	if err != nil {
		return err
	}
	receipt, exists, err := loadReceipt()
	if err != nil || !exists {
		return err
	}
	kept := receipt.Targets[:0]
	for _, record := range receipt.Targets {
		if !set[Target(record.Target)] {
			kept = append(kept, record)
		}
	}
	receipt.Targets = kept
	return saveReceipt(receipt)
}

func buildRefreshPlan() (refreshPlan, error) {
	receipt, _, err := loadReceipt()
	if err != nil {
		return refreshPlan{}, err
	}
	plan := refreshPlan{
		HomeInstalls: append([]homeInstall(nil), receipt.HomeInstalls...),
		Targets:      append([]targetInstall(nil), receipt.Targets...),
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return refreshPlan{}, err
	}

	// Native adapters can be discovered reliably at their standard paths. This
	// migrates installs made before the receipt existed.
	if targetInstalled(filepath.Join(home, ".codex", "agents"), ".toml") {
		plan.addTarget(TargetCodex, false)
	}
	openCodeBase := filepath.Join(home, ".config", "opencode")
	if fileExists(filepath.Join(openCodeBase, "plugins", "machinery.js")) ||
		targetInstalled(filepath.Join(openCodeBase, "agents"), ".md") {
		plan.addTarget(TargetOpenCode, false)
	}
	plan.ClaudePlugin = pluginInstalled(filepath.Join(home, ".claude"))

	// A receipt is authoritative for direct home groups. Without one, infer the
	// original ~/.agents + ~/.claude topology, taking care not to reinterpret the
	// shared ~/.agents copy of a native target as a separate legacy install.
	if len(plan.HomeInstalls) == 0 {
		agentsHome := filepath.Join(home, ".agents")
		claudeHome := filepath.Join(home, ".claude")
		agentsInstalled := skillInstalled(agentsHome)
		claudeInstalled := skillInstalled(claudeHome)
		sharedCovered := plan.hasTarget(TargetCodex) || plan.hasTarget(TargetOpenCode)
		claudeLinkedToShared := claudeInstalled && isSymlinkPath(filepath.Join(claudeHome, "skills", "machinery"))
		if sharedCovered && claudeLinkedToShared {
			plan.addTarget(TargetClaude, false)
			claudeInstalled = false
		}
		switch {
		case agentsInstalled && claudeInstalled && !sharedCovered:
			copyAll := !claudeLinkedToShared
			plan.HomeInstalls = append(plan.HomeInstalls, homeInstall{Homes: []string{agentsHome, claudeHome}, Copy: copyAll})
		case agentsInstalled && !sharedCovered:
			plan.HomeInstalls = append(plan.HomeInstalls, homeInstall{Homes: []string{agentsHome}})
		case claudeInstalled:
			plan.HomeInstalls = append(plan.HomeInstalls, homeInstall{Homes: []string{claudeHome}})
		}
	}
	return plan, nil
}

func (p *refreshPlan) addTarget(target Target, copyAll bool) {
	for i := range p.Targets {
		if p.Targets[i].Target == string(target) {
			return
		}
	}
	p.Targets = append(p.Targets, targetInstall{Target: string(target), Copy: copyAll})
	sort.SliceStable(p.Targets, func(i, j int) bool { return p.Targets[i].Target < p.Targets[j].Target })
}

func (p refreshPlan) hasTarget(target Target) bool {
	for _, record := range p.Targets {
		if record.Target == string(target) {
			return true
		}
	}
	return false
}

func skillInstalled(home string) bool {
	return fileExists(filepath.Join(home, "skills", "machinery", "SKILL.md"))
}

func targetInstalled(dir, suffix string) bool {
	for _, spec := range roleSpecs {
		if fileExists(filepath.Join(dir, spec.Name+suffix)) {
			return true
		}
	}
	return false
}

func isSymlinkPath(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func normalizeReceipt(receipt *installReceipt) {
	receipt.SchemaVersion = receiptSchema
	sort.SliceStable(receipt.HomeInstalls, func(i, j int) bool {
		if len(receipt.HomeInstalls[i].Homes) == 0 {
			return false
		}
		if len(receipt.HomeInstalls[j].Homes) == 0 {
			return true
		}
		return receipt.HomeInstalls[i].Homes[0] < receipt.HomeInstalls[j].Homes[0]
	})
	sort.SliceStable(receipt.Targets, func(i, j int) bool {
		return receipt.Targets[i].Target < receipt.Targets[j].Target
	})
}

func validateReceipt(receipt installReceipt) error {
	canonicals := map[string]bool{}
	for _, group := range receipt.HomeInstalls {
		if len(group.Homes) == 0 {
			return fmt.Errorf("home install has no homes")
		}
		if canonicals[group.Homes[0]] {
			return fmt.Errorf("duplicate canonical home %s", group.Homes[0])
		}
		canonicals[group.Homes[0]] = true
		for _, home := range group.Homes {
			if !filepath.IsAbs(home) {
				return fmt.Errorf("home path is not absolute: %s", home)
			}
		}
	}
	targets := map[string]bool{}
	for _, target := range receipt.Targets {
		name := Target(target.Target)
		if name != TargetClaude && name != TargetCodex && name != TargetOpenCode {
			return fmt.Errorf("unknown target %q", target.Target)
		}
		if targets[target.Target] {
			return fmt.Errorf("duplicate target %q", target.Target)
		}
		targets[target.Target] = true
	}
	return nil
}

func renameReplace(staged, destination string) error {
	if err := os.Rename(staged, destination); err == nil {
		return nil
	}
	backup := destination + ".pre-update"
	_ = os.Remove(backup)
	if err := os.Rename(destination, backup); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(staged, destination); err != nil {
		_ = os.Rename(backup, destination)
		return err
	}
	_ = os.Remove(backup)
	return nil
}
