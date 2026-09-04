package install

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/processcontrol"
)

var sha256Hex = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

const pluginCommandOutputLimit = 1 << 20

var (
	pluginCommandTimeout   = 2 * time.Minute
	pluginCommandWaitDelay = processcontrol.DefaultWaitDelay
)

type commandRunner func(name string, args ...string) (string, error)
type pathLookup func(name string) (string, error)

// UpdateOptions configures a forced release refresh. Homes/Targets, when
// present, restrict the asset refresh to those explicit placements; otherwise
// the persisted receipt plus standard-path discovery determines the plan.
type UpdateOptions struct {
	Version     string
	Repo        string
	Executable  string
	Homes       []string
	Targets     []string
	Copy        bool
	SkipPlugins bool
	// BootstrapDefaults asks a fresh installer to use Install's plugin-aware
	// default direct homes instead of receipt/discovery planning.
	BootstrapDefaults bool
	Out               io.Writer

	run      commandRunner
	lookPath pathLookup
	// allowNonAtomicDirectForTest preserves deep rollback/failure-injection
	// coverage for the legacy sequential direct-refresh implementation. No
	// production caller can opt out of the atomic-observability invariant.
	allowNonAtomicDirectForTest bool
}

// UpdateResult summarizes a completed release refresh.
type UpdateResult struct {
	Version        string
	Executable     string
	HomeInstalls   int
	TargetInstalls int
	PluginUpdates  int
	Warnings       []string
}

// PluginRefreshError reports host-owned plugin refresh failures. Host plugin
// stores cannot be rolled back by machinery, so every detected failure is
// retained and exposed as an aggregate instead of being downgraded to a
// successful update warning.
type PluginRefreshError struct {
	Failures []error
}

func (e *PluginRefreshError) Error() string {
	return fmt.Sprintf("host plugin refresh failed (%d error(s))", len(e.Failures))
}

func (e *PluginRefreshError) Unwrap() []error { return e.Failures }

// Update checksum-verifies and replaces the machinery binary. An existing
// binary plus direct homes/native targets is rejected before mutation because
// those independent roots cannot be atomically observed by ambient host
// processes; callers must uninstall them, update, then reinstall. Fresh
// bootstrap may establish the complete topology because no prior version can
// be observed. Host-owned plugin caches are refreshed through their CLIs and
// are never edited directly.
func Update(opts UpdateOptions) (result UpdateResult, retErr error) {
	operationLock, err := acquireInstallOperationLock()
	if err != nil {
		return UpdateResult{}, err
	}
	defer func() { retErr = errors.Join(retErr, operationLock.Release()) }()
	return updateLocked(opts)
}

func updateLocked(opts UpdateOptions) (result UpdateResult, retErr error) {
	out := opts.Out
	if out == nil {
		out = io.Discard
	}
	run := opts.run
	defaultRunner := run == nil
	if run == nil {
		run = runCombined
	}
	lookPath := opts.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	repo := opts.Repo
	if repo == "" {
		repo = defaultRepo
	}
	plan, err := updatePlan(opts)
	if err != nil {
		return UpdateResult{}, err
	}
	destination := opts.Executable
	if destination == "" {
		destination, err = runningInstallExecutable()
		if err != nil {
			return UpdateResult{}, err
		}
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return UpdateResult{}, err
	}
	if resolved, resolveErr := filepath.EvalSymlinks(destination); resolveErr == nil {
		destination = resolved
	}
	transactionPaths := []string{destination}
	for _, group := range plan.HomeInstalls {
		transactionPaths = append(transactionPaths, homeInstallArtifactPaths(group.Homes)...)
	}
	if len(plan.Targets) > 0 {
		var targetNames []string
		for _, target := range plan.Targets {
			targetNames = append(targetNames, target.Target)
		}
		targetPaths, pathErr := targetInstallArtifactPaths(targetNames)
		if pathErr != nil {
			return UpdateResult{}, pathErr
		}
		transactionPaths = append(transactionPaths, targetPaths...)
	}
	direct := len(plan.HomeInstalls) > 0 || len(plan.Targets) > 0
	if direct && !opts.allowNonAtomicDirectForTest {
		if _, statErr := os.Lstat(destination); statErr == nil {
			return UpdateResult{}, fmt.Errorf("refuse non-atomic multi-root update: an existing machinery binary cannot be activated together with %d direct home group(s) and %d native target(s) without exposing mixed versions; stop agent hosts, uninstall the recorded direct placements, update the binary, then reinstall them", len(plan.HomeInstalls), len(plan.Targets))
		} else if !os.IsNotExist(statErr) {
			return UpdateResult{}, fmt.Errorf("inspect update destination before atomic activation: %w", statErr)
		}
	}
	if direct {
		receipt, pathErr := installationReceiptPath()
		if pathErr != nil {
			return UpdateResult{}, pathErr
		}
		transactionPaths = append(transactionPaths, receipt)
	}
	// The parent retains the exclusive operation lock while it validates the
	// downloaded binary, even for a binary-only update. Give every real child
	// process the scoped capability so its startup consistency barrier can
	// authenticate the parent-held lock instead of waiting on its own parent.
	if defaultRunner {
		scope, scopeErr := installOperationScope()
		if scopeErr != nil {
			return UpdateResult{}, scopeErr
		}
		capability, cleanupCapability, capabilityErr := createInstallLockCapability(scope)
		if capabilityErr != nil {
			return UpdateResult{}, capabilityErr
		}
		defer cleanupCapability()
		run = runCombinedWithInstallLockCapability(capability)
	}
	tx, err := beginArtifactTransaction(transactionPaths)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("snapshot update transaction: %w", err)
	}
	// Downloads and validation are journal-owned preflight work. A process kill
	// leaves them beneath the same scratch root recovered on the next lock
	// acquisition, and every ordinary preflight failure rolls the journal back.
	tag, err := resolveTag(repo, opts.Version, true)
	if err != nil {
		return UpdateResult{}, rollbackUpdateTransaction(tx, err)
	}
	candidate, cleanupCandidate, err := fetchReleaseBinary(repo, tag, out)
	if err != nil {
		return UpdateResult{}, rollbackUpdateTransaction(tx, err)
	}
	defer func() { retErr = errors.Join(retErr, cleanupCandidate()) }()
	if err := validateReleaseBinary(candidate, tag, run); err != nil {
		return UpdateResult{}, rollbackUpdateTransaction(tx, err)
	}
	var source string
	if direct {
		var cleanupSource func() error
		source, cleanupSource, err = fetchSource(repo, tag, true, out)
		if err != nil {
			return UpdateResult{}, rollbackUpdateTransaction(tx, err)
		}
		defer func() { retErr = errors.Join(retErr, cleanupSource()) }()
	}
	if err := replaceExecutable(candidate, destination); err != nil {
		return UpdateResult{}, rollbackUpdateTransaction(tx, fmt.Errorf("replace machinery binary at %s: %w", destination, err))
	}
	fmt.Fprintf(out, "updated machinery binary -> %s (%s)\n", destination, tag)

	result = UpdateResult{
		Version:        tag,
		Executable:     destination,
		HomeInstalls:   len(plan.HomeInstalls),
		TargetInstalls: len(plan.Targets),
	}
	if source != "" {
		if err := refreshDirectInstalls(destination, source, plan, run, out); err != nil {
			return result, rollbackUpdateTransaction(tx, fmt.Errorf("binary updated to %s, but direct harness refresh failed: %w", tag, err))
		}
	}
	if err := tx.commit(); err != nil {
		return result, fmt.Errorf("commit update transaction: %w", err)
	}
	if !opts.SkipPlugins {
		updates, warnings, obligations, pluginErr := refreshHostPlugins(plan, run, lookPath, out)
		result.PluginUpdates = updates
		result.Warnings = warnings
		if obligationErr := recordHostPluginObligationsLocked(obligations); obligationErr != nil {
			pluginErr = errors.Join(pluginErr, fmt.Errorf("persist host plugin refresh obligation: %w", obligationErr))
		}
		if pluginErr != nil {
			return result, pluginErr
		}
	}
	if len(plan.HomeInstalls) == 0 && len(plan.Targets) == 0 && result.PluginUpdates == 0 {
		fmt.Fprintln(out, "no direct harness installation was detected; the binary was updated only (run 'machinery install --target all' to add adapters)")
	}
	fmt.Fprintf(out, "machinery update complete: %s; %d home group(s), %d native target(s), %d plugin refresh(es)\n",
		tag, result.HomeInstalls, result.TargetInstalls, result.PluginUpdates)
	return result, nil
}

func rollbackUpdateTransaction(tx *artifactTransaction, cause error) error {
	if rollbackErr := tx.rollback(); rollbackErr != nil {
		return errors.Join(fmt.Errorf("update failed: %w", cause), fmt.Errorf("rollback update transaction: %w", rollbackErr))
	}
	return cause
}

func updatePlan(opts UpdateOptions) (refreshPlan, error) {
	if opts.BootstrapDefaults {
		if len(opts.Homes) != 0 || len(opts.Targets) != 0 {
			return refreshPlan{}, fmt.Errorf("bootstrap defaults cannot be combined with explicit homes or targets")
		}
		defaults, err := DefaultHomes()
		if err != nil {
			return refreshPlan{}, err
		}
		homes, err := absHomes(defaults)
		if err != nil {
			return refreshPlan{}, err
		}
		kept := homes[:0]
		for _, home := range homes {
			installed, pluginErr := pluginInstalled(home)
			if pluginErr != nil {
				return refreshPlan{}, fmt.Errorf("discover machinery plugin under %s: %w", home, pluginErr)
			}
			if !installed {
				kept = append(kept, home)
			}
		}
		plan := refreshPlan{}
		if len(kept) != 0 {
			plan.HomeInstalls = []homeInstall{{Homes: kept}}
		}
		userHome, err := userHomeDir()
		if err != nil {
			return refreshPlan{}, err
		}
		plan.ClaudePlugin, err = pluginInstalled(filepath.Join(userHome, ".claude"))
		if err != nil {
			return refreshPlan{}, fmt.Errorf("discover machinery plugin: %w", err)
		}
		if err := applyRecordedPluginObligations(&plan); err != nil {
			return refreshPlan{}, err
		}
		return plan, nil
	}
	if len(opts.Homes) == 0 && len(opts.Targets) == 0 {
		return buildRefreshPlan()
	}
	var plan refreshPlan
	if len(opts.Homes) > 0 {
		homes, err := absHomes(opts.Homes)
		if err != nil {
			return refreshPlan{}, err
		}
		if len(homes) == 0 {
			return refreshPlan{}, fmt.Errorf("at least one non-empty --home is required")
		}
		plan.HomeInstalls = []homeInstall{{Homes: homes, Copy: opts.Copy}}
	}
	if len(opts.Targets) > 0 {
		set, err := parseTargets(opts.Targets)
		if err != nil {
			return refreshPlan{}, err
		}
		for _, target := range targetOrder {
			if set[target] {
				plan.Targets = append(plan.Targets, targetInstall{Target: string(target), Copy: opts.Copy})
			}
		}
	}
	home, err := userHomeDir()
	if err != nil {
		return refreshPlan{}, err
	}
	plan.ClaudePlugin, err = pluginInstalled(filepath.Join(home, ".claude"))
	if err != nil {
		return refreshPlan{}, fmt.Errorf("discover machinery plugin: %w", err)
	}
	if err := applyRecordedPluginObligations(&plan); err != nil {
		return refreshPlan{}, err
	}
	return plan, nil
}

func applyRecordedPluginObligations(plan *refreshPlan) error {
	receipt, exists, err := loadReceipt()
	if err != nil || !exists {
		return err
	}
	for _, plugin := range receipt.HostPlugins {
		plan.ClaudePlugin = plan.ClaudePlugin || plugin == "claude"
		plan.CodexPlugin = plan.CodexPlugin || plugin == "codex"
	}
	return nil
}

func fetchReleaseBinary(repo, tag string, out io.Writer) (string, func() error, error) {
	asset, err := releaseAssetName()
	if err != nil {
		return "", func() error { return nil }, err
	}
	tmp, cleanup, err := installScratchDir("update")
	if err != nil {
		return "", func() error { return nil }, err
	}
	base := githubBase + "/" + repo + "/releases/download/" + tag
	binary := filepath.Join(tmp, asset)
	checksums := filepath.Join(tmp, "checksums-sha256.txt")
	fmt.Fprintf(out, "fetching machinery %s (%s/%s)\n", tag, runtime.GOOS, runtime.GOARCH)
	if err := download(base+"/"+asset, binary, releaseBinaryDownload); err != nil {
		return "", func() error { return nil }, errors.Join(fmt.Errorf("download %s: %w", asset, err), cleanup())
	}
	if err := download(base+"/checksums-sha256.txt", checksums, releaseChecksumsDownload); err != nil {
		return "", func() error { return nil }, errors.Join(fmt.Errorf("release %s has no checksums-sha256.txt: %w", tag, err), cleanup())
	}
	want, err := checksumForAsset(checksums, asset)
	if err != nil {
		return "", func() error { return nil }, errors.Join(err, cleanup())
	}
	got, err := hashDownloadedFile(binary, releaseBinaryDownload)
	if err != nil {
		return "", func() error { return nil }, errors.Join(err, cleanup())
	}
	if got != want {
		return "", func() error { return nil }, errors.Join(fmt.Errorf("checksum mismatch for %s (want %s, got %s)", asset, want, got), cleanup())
	}
	if err := os.Chmod(binary, 0o755); err != nil {
		return "", func() error { return nil }, errors.Join(err, cleanup())
	}
	fmt.Fprintln(out, "checksum verified")
	return binary, cleanup, nil
}

func releaseAssetName() (string, error) {
	return releaseAssetNameFor(runtime.GOOS, runtime.GOARCH)
}

func releaseAssetNameFor(goos, goarch string) (string, error) {
	if goarch != "amd64" && goarch != "arm64" {
		return "", fmt.Errorf("unsupported architecture for self-update: %s", goarch)
	}
	switch goos {
	case "darwin", "linux":
		return "machinery-" + goos + "-" + goarch, nil
	case "windows":
		return "", fmt.Errorf("unsupported operating system for self-update: windows (v0.6.7 publishes Linux and macOS binaries only)")
	default:
		return "", fmt.Errorf("unsupported operating system for self-update: %s", goos)
	}
}

func checksumForAsset(path, asset string) (string, error) {
	raw, err := readDownloadedFile(path, releaseChecksumsDownload)
	if err != nil {
		return "", err
	}
	var found string
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if name == asset {
			if !sha256Hex.MatchString(fields[0]) {
				return "", fmt.Errorf("invalid SHA-256 checksum for %s", asset)
			}
			if found != "" {
				return "", fmt.Errorf("duplicate checksum entries for %s", asset)
			}
			found = strings.ToLower(fields[0])
		}
	}
	if found != "" {
		return found, nil
	}
	return "", fmt.Errorf("no checksum listed for %s", asset)
}

func validateReleaseBinary(candidate, tag string, run commandRunner) error {
	out, err := run(candidate, "version")
	if err != nil {
		return fmt.Errorf("validate downloaded binary: %w (%s)", err, strings.TrimSpace(out))
	}
	want := "machinery version " + tag
	if strings.TrimSpace(out) != want {
		return fmt.Errorf("downloaded binary reports %q, want %q", strings.TrimSpace(out), want)
	}
	return nil
}

func replaceExecutable(candidate, destination string) (retErr error) {
	dir := filepath.Dir(destination)
	if err := durableMkdirAll(dir); err != nil {
		return err
	}
	in, candidateInfo, err := openDownloadedFile(candidate, releaseBinaryDownload)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, closeInstallFile(in)) }()
	staged, cleanupStaged, err := installScratchFile(dir, "executable")
	if err != nil {
		return err
	}
	stagedPath := staged.Name()
	defer func() { retErr = errors.Join(retErr, cleanupStaged()) }()
	written, copyErr := io.Copy(staged, io.LimitReader(in, releaseBinaryMaxBytes+1))
	if copyErr != nil {
		return errors.Join(copyErr, closeInstallFile(staged))
	}
	if written > releaseBinaryMaxBytes || written != candidateInfo.Size() {
		return errors.Join(fmt.Errorf("update binary changed size while staging (%d bytes, expected %d)", written, candidateInfo.Size()), closeInstallFile(staged))
	}
	if err := revalidateDownloadedFile(candidate, in, candidateInfo, releaseBinaryDownload); err != nil {
		return errors.Join(err, closeInstallFile(staged))
	}
	if err := staged.Chmod(0o755); err != nil {
		return errors.Join(err, closeInstallFile(staged))
	}
	if err := staged.Sync(); err != nil {
		return errors.Join(err, closeInstallFile(staged))
	}
	if err := closeInstallFile(staged); err != nil {
		return err
	}
	return renameReplace(stagedPath, destination)
}

func refreshDirectInstalls(binary, source string, plan refreshPlan, run commandRunner, out io.Writer) error {
	for _, group := range plan.HomeInstalls {
		args := []string{"install", "--from", source}
		for _, home := range group.Homes {
			args = append(args, "--home", home)
		}
		if group.Copy {
			args = append(args, "--copy")
		}
		if err := runAndRelay(binary, args, run, out); err != nil {
			return err
		}
	}
	for _, copyAll := range []bool{false, true} {
		args := []string{"install", "--from", source}
		count := 0
		for _, target := range plan.Targets {
			if target.Copy == copyAll {
				args = append(args, "--target", target.Target)
				count++
			}
		}
		if count == 0 {
			continue
		}
		if copyAll {
			args = append(args, "--copy")
		}
		if err := runAndRelay(binary, args, run, out); err != nil {
			return err
		}
	}
	return nil
}

func runAndRelay(name string, args []string, run commandRunner, out io.Writer) error {
	output, err := run(name, args...)
	if output != "" {
		fmt.Fprint(out, output)
		if !strings.HasSuffix(output, "\n") {
			fmt.Fprintln(out)
		}
	}
	if err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func refreshHostPlugins(plan refreshPlan, run commandRunner, lookPath pathLookup, out io.Writer) (int, []string, []string, error) {
	updates := 0
	var warnings []string
	var obligations []string
	var failures []error
	fail := func(message string, cause error) {
		warnings = append(warnings, message)
		failure := errors.New(message)
		if cause != nil {
			failure = fmt.Errorf("%s: %w", message, cause)
		}
		failures = append(failures, failure)
		fmt.Fprintln(out, "error: "+failure.Error())
	}

	if plan.ClaudePlugin {
		claude, err := lookPath("claude")
		if err != nil {
			fail("Claude Code machinery plugin detected, but 'claude' is not on PATH; run 'claude plugin update machinery@machinery'", err)
		} else {
			listed, listErr := run(claude, "plugin", "list", "--json")
			scopes, parseErr := claudeMachineryScopes(listed)
			if listErr != nil {
				fail("Claude Code plugin inventory failed", listErr)
			} else if parseErr != nil {
				fail("Claude Code plugin inventory was not understood", parseErr)
			} else if len(scopes) == 0 {
				fail("Claude Code machinery plugin obligation exists, but inventory returned no installed machinery scope", nil)
			}
			if listErr != nil || parseErr != nil || len(scopes) == 0 {
				goto codexRefresh
			}
			obligations = append(obligations, "claude")
			output, marketplaceErr := run(claude, "plugin", "marketplace", "update", "machinery")
			if marketplaceErr == nil {
				marketplaceErr = validateClaudeMarketplaceUpdateOutput(output)
			}
			if marketplaceErr != nil {
				message := "Claude Code machinery marketplace refresh failed"
				if detail := strings.TrimSpace(output); detail != "" {
					message += " (" + detail + ")"
				}
				fail(message, marketplaceErr)
				goto codexRefresh
			}
			claudeUpdates := 0
			for _, scope := range scopes {
				output, err := run(claude, "plugin", "update", "machinery@machinery", "--scope", scope)
				if err == nil {
					err = validateClaudePluginUpdateOutput(output, scope)
				}
				if err != nil {
					message := "Claude Code machinery plugin refresh failed for " + scope + " scope"
					if detail := strings.TrimSpace(output); detail != "" {
						message += " (" + detail + ")"
					}
					fail(message, err)
					continue
				}
				claudeUpdates++
			}
			if claudeUpdates > 0 {
				updates += claudeUpdates
				fmt.Fprintf(out, "refreshed Claude Code plugin machinery@machinery in %d scope(s)\n", claudeUpdates)
			}
		}
	}

codexRefresh:
	if codex, err := lookPath("codex"); err == nil {
		listed, listErr := run(codex, "plugin", "list", "--json")
		if listErr != nil {
			fail("Codex plugin inventory failed", listErr)
		} else if installed, parseErr := codexMachineryInstalled(listed); parseErr != nil {
			fail("Codex plugin inventory was not understood", parseErr)
		} else if installed {
			obligations = append(obligations, "codex")
			output, err := run(codex, "plugin", "add", "machinery@machinery", "--json")
			if err == nil {
				err = validateCodexPluginAddOutput(output)
			}
			if err != nil {
				message := "Codex machinery plugin refresh failed; run 'codex plugin add machinery@machinery'"
				if detail := strings.TrimSpace(output); detail != "" {
					message += " (" + detail + ")"
				}
				fail(message, err)
			} else {
				updates++
				fmt.Fprintln(out, "refreshed Codex plugin machinery@machinery")
			}
		}
	} else if plan.CodexPlugin {
		fail("Codex machinery plugin obligation exists, but 'codex' is not on PATH; run 'codex plugin add machinery@machinery'", err)
	}
	if len(failures) != 0 {
		return updates, warnings, obligations, &PluginRefreshError{Failures: failures}
	}
	return updates, warnings, obligations, nil
}

var (
	pluginVersionRE                 = `[0-9A-Za-z][0-9A-Za-z.+-]*`
	claudeMarketplaceUpdateOutputRE = regexp.MustCompile(
		`^Updating marketplace: machinery\.\.\.(?:Validating local marketplace\n)?✔ Successfully updated marketplace: machinery\n?$`,
	)
)

func validateClaudeMarketplaceUpdateOutput(output string) error {
	if !claudeMarketplaceUpdateOutputRE.MatchString(strings.ReplaceAll(output, "\r\n", "\n")) {
		return fmt.Errorf("non-canonical Claude marketplace success output")
	}
	return nil
}

func validateClaudePluginUpdateOutput(output, scope string) error {
	if !map[string]bool{"managed": true, "user": true, "project": true, "local": true}[scope] {
		return fmt.Errorf("unsupported Claude plugin scope %q", scope)
	}
	prefix := `Checking for updates for plugin "machinery@machinery" at ` + regexp.QuoteMeta(scope) + ` scope(?:…|\.\.\.)\n✔ `
	upToDate := `machinery is already at the latest version \(` + pluginVersionRE + `\)\.`
	updated := `Plugin "machinery" updated from (?:unknown|` + pluginVersionRE + `) to ` + pluginVersionRE + ` for scope ` + regexp.QuoteMeta(scope) + `\. Restart to apply changes\.`
	pattern := regexp.MustCompile(`^` + prefix + `(?:` + upToDate + `|` + updated + `)\n?$`)
	if !pattern.MatchString(strings.ReplaceAll(output, "\r\n", "\n")) {
		return fmt.Errorf("non-canonical Claude plugin success output for %s scope", scope)
	}
	return nil
}

func validateCodexPluginAddOutput(output string) error {
	var result map[string]json.RawMessage
	if err := decodePluginInventory(output, &result); err != nil {
		return fmt.Errorf("codex plugin add result: %w", err)
	}
	fields := []string{"pluginId", "name", "marketplaceName", "version", "installedPath", "authPolicy"}
	if err := requirePluginFields(result, fields, "codex plugin add result"); err != nil {
		return err
	}
	values := make(map[string]string, len(fields))
	for _, field := range fields {
		var value string
		if err := json.Unmarshal(result[field], &value); err != nil || value == "" {
			return fmt.Errorf("codex plugin add result field %q is not a nonempty string", field)
		}
		values[field] = value
	}
	if values["pluginId"] != "machinery@machinery" || values["name"] != "machinery" || values["marketplaceName"] != "machinery" {
		return fmt.Errorf("codex plugin add result identity does not match machinery@machinery")
	}
	if !filepath.IsAbs(values["installedPath"]) {
		return fmt.Errorf("codex plugin add result installedPath is not absolute")
	}
	if values["authPolicy"] != "ON_INSTALL" && values["authPolicy"] != "ON_USE" {
		return fmt.Errorf("codex plugin add result authPolicy %q is unsupported", values["authPolicy"])
	}
	return nil
}

func claudeMachineryScopes(raw string) ([]string, error) {
	var entries []map[string]json.RawMessage
	if err := decodePluginInventory(raw, &entries); err != nil {
		return nil, err
	}
	found := map[string]bool{}
	validScope := map[string]bool{"managed": true, "user": true, "project": true, "local": true}
	for i, entry := range entries {
		if err := requirePluginFields(entry, []string{"id", "scope"}, fmt.Sprintf("Claude plugin entry %d", i)); err != nil {
			return nil, err
		}
		var id, scope string
		if err := json.Unmarshal(entry["id"], &id); err != nil || id == "" {
			return nil, fmt.Errorf("entry has no string id")
		}
		if err := json.Unmarshal(entry["scope"], &scope); err != nil || !validScope[scope] {
			return nil, fmt.Errorf("plugin %s has unsupported scope", id)
		}
		if id == "machinery@machinery" {
			if found[scope] {
				return nil, fmt.Errorf("duplicate machinery plugin entry for %s scope", scope)
			}
			found[scope] = true
		}
	}
	var scopes []string
	for _, scope := range []string{"managed", "user", "project", "local"} {
		if found[scope] {
			scopes = append(scopes, scope)
		}
	}
	return scopes, nil
}

func codexMachineryInstalled(raw string) (bool, error) {
	var root map[string]json.RawMessage
	if err := decodePluginInventory(raw, &root); err != nil {
		return false, err
	}
	if err := requirePluginFields(root, []string{"installed"}, "Codex plugin inventory", "available"); err != nil {
		return false, err
	}
	installedRaw, ok := root["installed"]
	if !ok {
		return false, fmt.Errorf("missing installed inventory")
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(installedRaw, &entries); err != nil {
		return false, fmt.Errorf("installed inventory: %w", err)
	}
	found := false
	seen := map[string]bool{}
	for i, entry := range entries {
		context := fmt.Sprintf("Codex installed entry %d", i)
		if err := requirePluginFields(entry, []string{
			"pluginId", "name", "marketplaceName", "version", "installed", "enabled", "source", "installPolicy", "authPolicy",
		}, context, "marketplaceSource"); err != nil {
			return false, err
		}
		values := map[string]string{}
		for _, field := range []string{"pluginId", "name", "marketplaceName", "version", "installPolicy", "authPolicy"} {
			var value string
			if err := json.Unmarshal(entry[field], &value); err != nil || strings.TrimSpace(value) == "" {
				return false, fmt.Errorf("%s field %q is not a nonempty string", context, field)
			}
			values[field] = value
		}
		id := values["pluginId"]
		if seen[id] {
			return false, fmt.Errorf("duplicate installed plugin entry %q", id)
		}
		seen[id] = true
		var installed bool
		if err := json.Unmarshal(entry["installed"], &installed); err != nil {
			return false, fmt.Errorf("plugin %s has no boolean installed status", id)
		}
		var enabled bool
		if err := json.Unmarshal(entry["enabled"], &enabled); err != nil {
			return false, fmt.Errorf("plugin %s has no boolean enabled status", id)
		}
		if enabled && !installed {
			return false, fmt.Errorf("plugin %s claims enabled without being installed", id)
		}
		if !validCodexInstallPolicy(values["installPolicy"]) {
			return false, fmt.Errorf("plugin %s has unsupported installPolicy %q", id, values["installPolicy"])
		}
		if values["authPolicy"] != "ON_INSTALL" && values["authPolicy"] != "ON_USE" {
			return false, fmt.Errorf("plugin %s has unsupported authPolicy %q", id, values["authPolicy"])
		}
		if err := validateCodexSourceObject(entry["source"], context+" source", "source", "path"); err != nil {
			return false, err
		}
		if rawMarketplace, ok := entry["marketplaceSource"]; ok {
			if err := validateCodexSourceObject(rawMarketplace, context+" marketplaceSource", "sourceType", "source"); err != nil {
				return false, err
			}
		}
		if id == "machinery@machinery" {
			if values["name"] != "machinery" || values["marketplaceName"] != "machinery" {
				return false, fmt.Errorf("codex machinery plugin identity does not match machinery@machinery")
			}
			found = installed && enabled
		}
	}
	if availableRaw, ok := root["available"]; ok {
		var available []map[string]json.RawMessage
		if err := json.Unmarshal(availableRaw, &available); err != nil {
			return false, fmt.Errorf("available inventory: %w", err)
		}
		for i, entry := range available {
			context := fmt.Sprintf("Codex available entry %d", i)
			if err := requirePluginFields(entry, []string{
				"pluginId", "marketplaceName", "installed", "enabled", "marketplaceSource", "installPolicy", "authPolicy",
			}, context); err != nil {
				return false, err
			}
			values := map[string]string{}
			for _, field := range []string{"pluginId", "marketplaceName", "installPolicy", "authPolicy"} {
				var value string
				if err := json.Unmarshal(entry[field], &value); err != nil || strings.TrimSpace(value) == "" {
					return false, fmt.Errorf("%s field %q is not a nonempty string", context, field)
				}
				values[field] = value
			}
			id := values["pluginId"]
			if seen[id] {
				return false, fmt.Errorf("duplicate Codex plugin entry %q across installed and available inventories", id)
			}
			seen[id] = true
			var installed, enabled bool
			if err := json.Unmarshal(entry["installed"], &installed); err != nil {
				return false, fmt.Errorf("available plugin %s has no boolean installed status", id)
			}
			if err := json.Unmarshal(entry["enabled"], &enabled); err != nil {
				return false, fmt.Errorf("available plugin %s has no boolean enabled status", id)
			}
			if installed || enabled {
				return false, fmt.Errorf("available plugin %s claims installed=%t enabled=%t", id, installed, enabled)
			}
			if !validCodexInstallPolicy(values["installPolicy"]) {
				return false, fmt.Errorf("available plugin %s has unsupported installPolicy %q", id, values["installPolicy"])
			}
			if values["authPolicy"] != "ON_INSTALL" && values["authPolicy"] != "ON_USE" {
				return false, fmt.Errorf("available plugin %s has unsupported authPolicy %q", id, values["authPolicy"])
			}
			if err := validateCodexSourceObject(entry["marketplaceSource"], context+" marketplaceSource", "sourceType", "source"); err != nil {
				return false, err
			}
		}
	}
	return found, nil
}

func validCodexInstallPolicy(value string) bool {
	return value == "NOT_AVAILABLE" || value == "AVAILABLE" || value == "INSTALLED_BY_DEFAULT"
}

func validateCodexSourceObject(raw json.RawMessage, context, kindField, pathField string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return fmt.Errorf("%s: %w", context, err)
	}
	if err := requirePluginFields(object, []string{kindField, pathField}, context); err != nil {
		return err
	}
	for _, field := range []string{kindField, pathField} {
		var value string
		if err := json.Unmarshal(object[field], &value); err != nil || strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s field %q is not a nonempty string", context, field)
		}
	}
	return nil
}

func decodePluginInventory(raw string, destination any) error {
	data := []byte(raw)
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = fmt.Errorf("trailing JSON value")
		}
		return err
	}
	return nil
}

// requirePluginFields validates a closed object schema and emits all unknown
// fields in lexical order, independent of Go map iteration or producer order.
func requirePluginFields(object map[string]json.RawMessage, required []string, context string, optional ...string) error {
	allowed := make(map[string]bool, len(required)+len(optional))
	for _, key := range required {
		allowed[key] = true
	}
	for _, key := range optional {
		allowed[key] = true
	}
	var unknown []string
	for key := range object {
		if !allowed[key] {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) != 0 {
		sort.Strings(unknown)
		return fmt.Errorf("%s has unknown fields %q", context, unknown)
	}
	for _, key := range required {
		if _, ok := object[key]; !ok {
			return fmt.Errorf("%s is missing required field %q", context, key)
		}
	}
	return nil
}

func runCombined(name string, args ...string) (string, error) {
	return runBoundedPluginCommand(nil, name, args...)
}

func runBoundedPluginCommand(environment []string, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pluginCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = pluginCommandWaitDelay
	if environment != nil {
		cmd.Env = environment
	}
	output, stderr, err := processcontrol.RunCapturedStreams(ctx, cmd, pluginCommandOutputLimit)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return output, fmt.Errorf("%s timed out after %s; process tree was terminated: %w%s", filepath.Base(name), pluginCommandTimeout, errors.Join(context.DeadlineExceeded, err), pluginStderrDiagnostic(stderr))
	}
	if errors.Is(err, exec.ErrWaitDelay) {
		return output, fmt.Errorf("%s descendant held output pipes open beyond %s; process tree was terminated: %w%s", filepath.Base(name), pluginCommandWaitDelay, err, pluginStderrDiagnostic(stderr))
	}
	if err != nil {
		return output, fmt.Errorf("%w%s", err, pluginStderrDiagnostic(stderr))
	}
	if stderr != "" {
		return output, fmt.Errorf("%s wrote to stderr: %q", filepath.Base(name), stderr)
	}
	return output, nil
}

func pluginStderrDiagnostic(stderr string) string {
	if stderr == "" {
		return ""
	}
	return fmt.Sprintf("; stderr: %q", stderr)
}
