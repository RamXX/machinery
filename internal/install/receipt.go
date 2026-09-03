package install

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/filelock"
)

const receiptSchema = 2

const receiptMaxBytes = 1 << 20

// Deterministic adversarial hooks used only by receipt confinement tests.
var (
	afterReceiptRootOpen   func()
	afterReceiptEntryLstat func()
)

type homeInstall struct {
	Homes []string `json:"homes"`
	Copy  bool     `json:"copy,omitempty"`
}

type targetInstall struct {
	Target string `json:"target"`
	Copy   bool   `json:"copy,omitempty"`
}

type installReceipt struct {
	SchemaVersion int               `json:"schema_version"`
	HomeInstalls  []homeInstall     `json:"home_installs,omitempty"`
	Targets       []targetInstall   `json:"targets,omitempty"`
	HostPlugins   []string          `json:"host_plugins,omitempty"`
	Artifacts     []receiptArtifact `json:"artifacts,omitempty"`
}

type receiptArtifact struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type refreshPlan struct {
	HomeInstalls []homeInstall
	Targets      []targetInstall
	ClaudePlugin bool
	CodexPlugin  bool
}

// ReceiptStatus is the user-facing summary reported by machinery doctor.
type ReceiptStatus struct {
	Path          string
	Exists        bool
	SchemaVersion int
	HomeInstalls  int
	Targets       int
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
		Path:          path,
		Exists:        exists,
		SchemaVersion: receipt.SchemaVersion,
		HomeInstalls:  len(receipt.HomeInstalls),
		Targets:       len(receipt.Targets),
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
	raw, exists, err := readReceiptFile(path)
	if !exists && err == nil {
		return installReceipt{SchemaVersion: receiptSchema}, false, nil
	}
	if err != nil {
		return installReceipt{}, exists, err
	}
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return installReceipt{}, true, fmt.Errorf("parse installation receipt %s: %w", path, err)
	}
	var receipt installReceipt
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&receipt); err != nil {
		return installReceipt{}, true, fmt.Errorf("parse installation receipt %s: %w", path, err)
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = fmt.Errorf("trailing JSON value")
		}
		return installReceipt{}, true, fmt.Errorf("parse installation receipt %s: %w", path, err)
	}
	if receipt.SchemaVersion != 1 && receipt.SchemaVersion != receiptSchema {
		return installReceipt{}, true, fmt.Errorf("installation receipt %s uses schema %d; this binary supports schemas 1 and %d", path, receipt.SchemaVersion, receiptSchema)
	}
	normalizeReceipt(&receipt)
	if err := validateReceipt(receipt); err != nil {
		return installReceipt{}, true, fmt.Errorf("invalid installation receipt %s: %w", path, err)
	}
	return receipt, true, nil
}

func readReceiptFile(path string) ([]byte, bool, error) {
	dir, base := filepath.Dir(path), filepath.Base(path)
	dirInfo, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if err := filelock.ValidatePrivateDir(dir, dirInfo); err != nil {
		return nil, false, fmt.Errorf("installation receipt directory is not confined: %w", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, false, err
	}
	openedDir, openedErr := root.Stat(".")
	if openedErr != nil || !os.SameFile(dirInfo, openedDir) {
		return nil, false, errors.Join(fmt.Errorf("installation receipt directory %s changed while opening", dir), openedErr, root.Close())
	}
	if afterReceiptRootOpen != nil {
		afterReceiptRootOpen()
	}
	entryInfo, err := root.Lstat(base)
	if os.IsNotExist(err) {
		return nil, false, errors.Join(root.Close(), verifyReceiptRootIdentity(dir, openedDir))
	}
	if err != nil {
		return nil, true, errors.Join(err, root.Close())
	}
	if entryInfo.Mode()&os.ModeSymlink != 0 || !entryInfo.Mode().IsRegular() || !privateFilePermissionsOK(entryInfo) {
		return nil, true, errors.Join(fmt.Errorf("installation receipt %s is not a private regular file", path), root.Close())
	}
	if entryInfo.Size() > receiptMaxBytes {
		return nil, true, errors.Join(fmt.Errorf("installation receipt %s exceeds %d bytes", path, receiptMaxBytes), root.Close())
	}
	if afterReceiptEntryLstat != nil {
		afterReceiptEntryLstat()
	}
	f, err := root.Open(base)
	if err != nil {
		return nil, true, errors.Join(err, root.Close())
	}
	openedEntry, statErr := f.Stat()
	if statErr != nil || !os.SameFile(entryInfo, openedEntry) || !openedEntry.Mode().IsRegular() || !privateFilePermissionsOK(openedEntry) {
		return nil, true, errors.Join(fmt.Errorf("installation receipt %s changed while opening", path), statErr, closeInstallFile(f), root.Close())
	}
	raw, readErr := io.ReadAll(io.LimitReader(f, receiptMaxBytes+1))
	closeErr := closeInstallFile(f)
	rootIdentityErr := verifyReceiptRootIdentity(dir, openedDir)
	rootCloseErr := root.Close()
	if readErr != nil || closeErr != nil || rootIdentityErr != nil || rootCloseErr != nil {
		return nil, true, errors.Join(readErr, closeErr, rootIdentityErr, rootCloseErr)
	}
	if len(raw) > receiptMaxBytes {
		return nil, true, fmt.Errorf("installation receipt %s exceeds %d bytes", path, receiptMaxBytes)
	}
	return raw, true, nil
}

func verifyReceiptRootIdentity(dir string, opened os.FileInfo) error {
	current, err := os.Lstat(dir)
	if err != nil || current.Mode()&os.ModeSymlink != 0 || !current.IsDir() || !os.SameFile(opened, current) {
		return errors.Join(fmt.Errorf("installation receipt directory %s changed during read", dir), err)
	}
	return nil
}

func rejectDuplicateJSONKeys(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	var parseValue func() error
	parseValue = func() error {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for dec.More() {
				keyToken, err := dec.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return fmt.Errorf("object key is not a string")
				}
				if seen[key] {
					return fmt.Errorf("duplicate JSON field %q", key)
				}
				seen[key] = true
				if err := parseValue(); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		case '[':
			for dec.More() {
				if err := parseValue(); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		default:
			return fmt.Errorf("unexpected JSON delimiter %q", delim)
		}
	}
	if err := parseValue(); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return err
	}
	return nil
}

func saveReceipt(receipt installReceipt) (retErr error) {
	receipt.SchemaVersion = receiptSchema
	normalizeReceipt(&receipt)
	path, err := installationReceiptPath()
	if err != nil {
		return err
	}
	if len(receipt.HomeInstalls) == 0 && len(receipt.Targets) == 0 && len(receipt.HostPlugins) == 0 {
		return durableRemove(path)
	}
	if err := refreshReceiptArtifacts(&receipt); err != nil {
		return fmt.Errorf("inventory installed artifacts for receipt: %w", err)
	}
	normalizeReceipt(&receipt)
	if err := durableMkdirAllPrivate(filepath.Dir(path)); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp, cleanupTmp, err := installScratchFile(filepath.Dir(path), "receipt")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { retErr = errors.Join(retErr, cleanupTmp()) }()
	if err := tmp.Chmod(0o600); err != nil {
		return errors.Join(err, closeInstallFile(tmp))
	}
	if _, err := tmp.Write(raw); err != nil {
		return errors.Join(err, closeInstallFile(tmp))
	}
	if err := tmp.Sync(); err != nil {
		return errors.Join(err, closeInstallFile(tmp))
	}
	if err := closeInstallFile(tmp); err != nil {
		return err
	}
	return renameReplace(tmpPath, path)
}

func recordHomeInstall(homes []string, copyAll bool) error {
	return withInstallOperationLock(func() error { return recordHomeInstallLocked(homes, copyAll) })
}

func recordHomeInstallLocked(homes []string, copyAll bool) error {
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
		if len(receipt.HomeInstalls[i].Homes) > 0 && sameInstallPath(receipt.HomeInstalls[i].Homes[0], abs[0]) {
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

func recordTargetInstallLocked(names []string, copyAll bool) error {
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

func recordHostPluginObligationsLocked(obligations []string) error {
	if len(obligations) == 0 {
		return nil
	}
	receipt, _, err := loadReceipt()
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, plugin := range receipt.HostPlugins {
		seen[plugin] = true
	}
	changed := false
	for _, plugin := range obligations {
		if (plugin != "claude" && plugin != "codex") || seen[plugin] {
			continue
		}
		receipt.HostPlugins = append(receipt.HostPlugins, plugin)
		seen[plugin] = true
		changed = true
	}
	if !changed {
		return nil
	}
	path, err := installationReceiptPath()
	if err != nil {
		return err
	}
	tx, err := beginArtifactTransaction([]string{path})
	if err != nil {
		return err
	}
	if err := saveReceipt(receipt); err != nil {
		return errors.Join(err, tx.rollback())
	}
	return tx.commit()
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
		defaults, derr := DefaultHomes()
		if derr != nil {
			return derr
		}
		abs, err = absHomes(defaults)
		if err != nil {
			return err
		}
	}
	return withInstallOperationLock(func() error {
		return forgetHomeInstallsLocked(abs)
	})
}

func forgetHomeInstallsLocked(homes []string) error {
	receipt, exists, err := loadReceipt()
	if err != nil || !exists {
		return err
	}
	changed := forgetHomesFromReceipt(&receipt, homes)
	if !changed {
		return nil
	}
	return saveReceipt(receipt)
}

func forgetHomesFromReceipt(receipt *installReceipt, homes []string) bool {
	var groups []homeInstall
	changed := false
	for _, group := range receipt.HomeInstalls {
		if len(group.Homes) == 0 || installPathIn(group.Homes[0], homes) {
			changed = true
			continue
		}
		kept := make([]string, 0, len(group.Homes))
		for _, home := range group.Homes {
			if installPathIn(home, homes) {
				changed = true
				continue
			}
			kept = append(kept, home)
		}
		if len(kept) > 0 {
			group.Homes = kept
			groups = append(groups, group)
		}
	}
	receipt.HomeInstalls = groups
	return changed
}

func expandCanonicalHomeGroups(receipt installReceipt, homes []string) []string {
	expanded := append([]string(nil), homes...)
	for _, group := range receipt.HomeInstalls {
		if len(group.Homes) == 0 || !installPathIn(group.Homes[0], homes) {
			continue
		}
		for _, home := range group.Homes {
			if !installPathIn(home, expanded) {
				expanded = append(expanded, home)
			}
		}
	}
	return expanded
}

// ForgetTargetInstalls removes host-native adapters from the update receipt.
func ForgetTargetInstalls(names []string) error {
	set, err := parseTargets(names)
	if err != nil {
		return err
	}
	return withInstallOperationLock(func() error {
		return forgetTargetInstallsLocked(set)
	})
}

func forgetTargetInstallsLocked(set map[Target]bool) error {
	receipt, exists, err := loadReceipt()
	if err != nil || !exists {
		return err
	}
	changed := forgetTargetsFromReceipt(&receipt, set)
	if !changed {
		return nil
	}
	return saveReceipt(receipt)
}

func forgetTargetsFromReceipt(receipt *installReceipt, set map[Target]bool) bool {
	kept := receipt.Targets[:0]
	changed := false
	for _, record := range receipt.Targets {
		if set[Target(record.Target)] {
			changed = true
			continue
		}
		kept = append(kept, record)
	}
	receipt.Targets = kept
	return changed
}

func installPathIn(path string, paths []string) bool {
	for _, candidate := range paths {
		if sameInstallPath(path, candidate) {
			return true
		}
	}
	return false
}

func sameInstallPath(a, b string) bool {
	aID, aErr := installPathIdentity(a)
	bID, bErr := installPathIdentity(b)
	if aErr == nil && bErr == nil {
		return aID == bID
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func withInstallOperationLock(fn func() error) (retErr error) {
	operationLock, err := acquireInstallOperationLockWait()
	if err != nil {
		return err
	}
	defer func() {
		if releaseErr := operationLock.Release(); releaseErr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("release install operation lock: %w", releaseErr))
		}
	}()
	receipt, err := installationReceiptPath()
	if err != nil {
		return err
	}
	tx, err := beginArtifactTransaction([]string{receipt})
	if err != nil {
		return err
	}
	if err := fn(); err != nil {
		if rollbackErr := tx.rollback(); rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("rollback receipt transaction: %w", rollbackErr))
		}
		return err
	}
	return tx.commit()
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
	for _, plugin := range receipt.HostPlugins {
		plan.ClaudePlugin = plan.ClaudePlugin || plugin == "claude"
		plan.CodexPlugin = plan.CodexPlugin || plugin == "codex"
	}
	home, err := userHomeDir()
	if err != nil {
		return refreshPlan{}, err
	}

	// Native adapters can be discovered reliably at their standard paths. This
	// migrates installs made before the receipt existed.
	codexInstalled, err := targetInstalled(filepath.Join(home, ".codex", "agents"), ".toml")
	if err != nil {
		return refreshPlan{}, fmt.Errorf("discover Codex target: %w", err)
	}
	if codexInstalled {
		plan.addTarget(TargetCodex, false)
	}
	openCodeBase := filepath.Join(home, ".config", "opencode")
	openCodePlugin, err := fileExists(filepath.Join(openCodeBase, "plugins", "machinery.js"))
	if err != nil {
		return refreshPlan{}, fmt.Errorf("discover OpenCode plugin: %w", err)
	}
	openCodeAgents, err := targetInstalled(filepath.Join(openCodeBase, "agents"), ".md")
	if err != nil {
		return refreshPlan{}, fmt.Errorf("discover OpenCode agents: %w", err)
	}
	if openCodePlugin || openCodeAgents {
		plan.addTarget(TargetOpenCode, false)
	}
	claudePlugin, err := pluginInstalled(filepath.Join(home, ".claude"))
	if err != nil {
		return refreshPlan{}, fmt.Errorf("discover Claude plugin: %w", err)
	}
	plan.ClaudePlugin = plan.ClaudePlugin || claudePlugin

	// A receipt is authoritative for direct home groups. Without one, infer the
	// original ~/.agents + ~/.claude topology, taking care not to reinterpret the
	// shared ~/.agents copy of a native target as a separate legacy install.
	if len(plan.HomeInstalls) == 0 {
		agentsHome := filepath.Join(home, ".agents")
		claudeHome := filepath.Join(home, ".claude")
		agentsInstalled, err := skillInstalled(agentsHome)
		if err != nil {
			return refreshPlan{}, fmt.Errorf("discover shared machinery skill: %w", err)
		}
		claudeInstalled, err := skillInstalled(claudeHome)
		if err != nil {
			return refreshPlan{}, fmt.Errorf("discover Claude machinery skill: %w", err)
		}
		sharedCovered := plan.hasTarget(TargetCodex) || plan.hasTarget(TargetOpenCode)
		claudeLinkedToShared := false
		if claudeInstalled {
			claudeLinkedToShared, err = isSymlinkPath(filepath.Join(claudeHome, "skills", "machinery"))
			if err != nil {
				return refreshPlan{}, fmt.Errorf("discover Claude machinery skill topology: %w", err)
			}
		}
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

func skillInstalled(home string) (bool, error) {
	return fileExists(filepath.Join(home, "skills", "machinery", "SKILL.md"))
}

func targetInstalled(dir, suffix string) (bool, error) {
	installed := false
	for _, spec := range roleSpecs {
		found, err := fileExists(filepath.Join(dir, spec.Name+suffix))
		if err != nil {
			return false, err
		}
		if found {
			installed = true
		}
	}
	return installed, nil
}

func isSymlinkPath(path string) (bool, error) {
	info, err := installDiscoveryLstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return true, nil
	}
	if !info.IsDir() {
		return false, fmt.Errorf("expected directory or symlink at %s, got %s", path, info.Mode().Type())
	}
	return false, nil
}

func fileExists(path string) (bool, error) {
	info, err := installDiscoveryLstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false, fmt.Errorf("expected regular file at %s, got %s", path, info.Mode().Type())
	}
	return true, nil
}

func normalizeReceipt(receipt *installReceipt) {
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
	sort.Strings(receipt.HostPlugins)
	sort.SliceStable(receipt.Artifacts, func(i, j int) bool {
		return receipt.Artifacts[i].Path < receipt.Artifacts[j].Path
	})
}

func refreshReceiptArtifacts(receipt *installReceipt) error {
	paths, err := receiptArtifactPaths(*receipt)
	if err != nil {
		return err
	}
	receipt.Artifacts = make([]receiptArtifact, 0, len(paths))
	for _, path := range paths {
		digest, err := artifactTreeDigest(path)
		if err != nil {
			return fmt.Errorf("digest %s: %w", path, err)
		}
		receipt.Artifacts = append(receipt.Artifacts, receiptArtifact{Path: path, Digest: digest})
	}
	return nil
}

func receiptArtifactPaths(receipt installReceipt) ([]string, error) {
	var paths []string
	for _, group := range receipt.HomeInstalls {
		paths = append(paths, homeInstallArtifactPaths(group.Homes)...)
	}
	if len(receipt.Targets) > 0 {
		names := make([]string, 0, len(receipt.Targets))
		for _, target := range receipt.Targets {
			names = append(names, target.Target)
		}
		targetPaths, err := targetInstallArtifactPaths(names)
		if err != nil {
			return nil, err
		}
		paths = append(paths, targetPaths...)
	}
	unique := map[string]string{}
	for _, path := range paths {
		path = filepath.Clean(path)
		identity, err := installArtifactPathIdentity(path)
		if err != nil {
			return nil, err
		}
		unique[identity] = path
	}
	paths = paths[:0]
	for _, path := range unique {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func validateReceipt(receipt installReceipt) error {
	canonicals := map[string]bool{}
	var allHomes []string
	for _, group := range receipt.HomeInstalls {
		if len(group.Homes) == 0 {
			return fmt.Errorf("home install has no homes")
		}
		canonicalID, err := installPathIdentity(group.Homes[0])
		if err != nil {
			return fmt.Errorf("resolve canonical home %s: %w", group.Homes[0], err)
		}
		if canonicals[canonicalID] {
			return fmt.Errorf("duplicate canonical home %s", group.Homes[0])
		}
		canonicals[canonicalID] = true
		for _, home := range group.Homes {
			if !filepath.IsAbs(home) || filepath.Clean(home) != home {
				return fmt.Errorf("home path is not a clean absolute path: %s", home)
			}
			for _, prior := range allHomes {
				if sameOrNestedPath(prior, home) {
					return fmt.Errorf("home install paths overlap or repeat: %s and %s", prior, home)
				}
			}
			allHomes = append(allHomes, home)
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
	plugins := map[string]bool{}
	for _, plugin := range receipt.HostPlugins {
		if plugin != "claude" && plugin != "codex" {
			return fmt.Errorf("unknown host plugin obligation %q", plugin)
		}
		if plugins[plugin] {
			return fmt.Errorf("duplicate host plugin obligation %q", plugin)
		}
		plugins[plugin] = true
	}
	if receipt.SchemaVersion == 1 {
		if len(receipt.HostPlugins) != 0 {
			return fmt.Errorf("schema 1 receipt cannot carry host plugin obligations")
		}
		if len(receipt.Artifacts) != 0 {
			return fmt.Errorf("schema 1 receipt cannot carry artifact digests")
		}
		return nil
	}
	wantPaths, err := receiptArtifactPaths(receipt)
	if err != nil {
		return err
	}
	if len(receipt.Artifacts) != len(wantPaths) {
		return fmt.Errorf("artifact inventory has %d entries, want %d for recorded topology", len(receipt.Artifacts), len(wantPaths))
	}
	for i, artifact := range receipt.Artifacts {
		if !filepath.IsAbs(artifact.Path) || filepath.Clean(artifact.Path) != artifact.Path || artifact.Path != wantPaths[i] {
			return fmt.Errorf("artifact inventory path %q does not match recorded topology path %q", artifact.Path, wantPaths[i])
		}
		if len(artifact.Digest) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(artifact.Digest, "sha256:") {
			return fmt.Errorf("artifact inventory digest for %s is malformed", artifact.Path)
		}
		if _, err := hex.DecodeString(strings.TrimPrefix(artifact.Digest, "sha256:")); err != nil {
			return fmt.Errorf("artifact inventory digest for %s is malformed: %w", artifact.Path, err)
		}
	}
	return nil
}

func renameReplace(staged, destination string) error {
	if err := declareInstallPresentPostImage(destination, staged); err != nil {
		return err
	}
	if err := validateActiveInstallMutation(destination); err != nil {
		return err
	}
	if err := durableRemove(destination); err != nil {
		return err
	}
	if err := durableRename(staged, destination); err != nil {
		// A cross-device staged path cannot be renamed. The copy is synced before
		// it becomes the committed transaction state; rollback restores the old
		// destination if the copy or subsequent commit fails. O_EXCL publication
		// preserves anything that appears at the destination boundary.
		if copyErr := copyReplacement(staged, destination); copyErr != nil {
			return errors.Join(fmt.Errorf("rename replacement: %w", err), fmt.Errorf("copy fallback: %w", copyErr), durableRemove(destination))
		}
		if removeErr := durableRemove(staged); removeErr != nil {
			return removeErr
		}
	}
	return nil
}

func copyReplacement(src, dst string) error {
	if err := declareInstallPresentPostImage(dst, src); err != nil {
		return err
	}
	if err := validateActiveInstallMutation(dst); err != nil {
		return err
	}
	before, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Size() < 0 || before.Size() > installArtifactMaxFileBytes {
		return fmt.Errorf("replacement source %s must be a bounded regular non-symlink file", src)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if !sameInstallArtifactInfo(before, info) {
		return fmt.Errorf("replacement source %s changed while opening", src)
	}
	capability, rel, confined, err := retainedCapabilityForMutation(dst)
	if err != nil {
		return err
	}
	if afterInstallMutationValidation != nil {
		afterInstallMutationValidation(dst)
	}
	var out *os.File
	if confined {
		out, err = capability.OpenFile(rel, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	} else {
		out, err = os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	}
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(out, io.LimitReader(in, info.Size()+1))
	heldAfter, heldErr := in.Stat()
	liveAfter, liveErr := os.Lstat(src)
	if err := errors.Join(copyErr, heldErr, liveErr); err != nil {
		out.Close()
		return err
	}
	if written != info.Size() || !sameInstallArtifactInfo(info, heldAfter) || !sameInstallArtifactInfo(info, liveAfter) {
		out.Close()
		return fmt.Errorf("replacement source %s changed while copying", src)
	}
	if err := out.Chmod(info.Mode().Perm()); err != nil {
		out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if confined {
		if err := capability.Chtimes(rel, info.ModTime(), info.ModTime()); err != nil {
			return err
		}
		return syncRootRelativeDir(capability, filepath.Dir(rel))
	}
	if err := os.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
		return err
	}
	return syncDir(filepath.Dir(dst))
}
