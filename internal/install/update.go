package install

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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
	Out         io.Writer

	run      commandRunner
	lookPath pathLookup
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

// Update checksum-verifies and forcefully replaces the machinery binary, then
// asks that new binary to refresh every selected direct installation from the
// exact same release source. Host-owned plugin caches are refreshed through
// their CLIs on a best-effort basis and are never edited directly.
func Update(opts UpdateOptions) (UpdateResult, error) {
	out := opts.Out
	if out == nil {
		out = io.Discard
	}
	run := opts.run
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
	tag, err := resolveTag(repo, opts.Version)
	if err != nil {
		return UpdateResult{}, err
	}

	candidate, cleanupCandidate, err := fetchReleaseBinary(repo, tag, out)
	if err != nil {
		return UpdateResult{}, err
	}
	defer cleanupCandidate()
	if err := validateReleaseBinary(candidate, tag, run); err != nil {
		return UpdateResult{}, err
	}

	var source string
	if len(plan.HomeInstalls) > 0 || len(plan.Targets) > 0 {
		var cleanupSource func()
		source, cleanupSource, err = fetchSource(repo, tag, out)
		if err != nil {
			return UpdateResult{}, err
		}
		defer cleanupSource()
	}

	destination := opts.Executable
	if destination == "" {
		destination, err = os.Executable()
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
	if err := replaceExecutable(candidate, destination); err != nil {
		return UpdateResult{}, fmt.Errorf("replace machinery binary at %s: %w", destination, err)
	}
	fmt.Fprintf(out, "updated machinery binary -> %s (%s)\n", destination, tag)

	result := UpdateResult{
		Version:        tag,
		Executable:     destination,
		HomeInstalls:   len(plan.HomeInstalls),
		TargetInstalls: len(plan.Targets),
	}
	if source != "" {
		if err := refreshDirectInstalls(destination, source, plan, run, out); err != nil {
			return result, fmt.Errorf("binary updated to %s, but direct harness refresh failed: %w", tag, err)
		}
	}
	if !opts.SkipPlugins {
		updates, warnings := refreshHostPlugins(plan, run, lookPath, out)
		result.PluginUpdates = updates
		result.Warnings = warnings
	}
	if len(plan.HomeInstalls) == 0 && len(plan.Targets) == 0 && result.PluginUpdates == 0 {
		fmt.Fprintln(out, "no direct harness installation was detected; the binary was updated only (run 'machinery install --target all' to add adapters)")
	}
	fmt.Fprintf(out, "machinery update complete: %s; %d home group(s), %d native target(s), %d plugin refresh(es)\n",
		tag, result.HomeInstalls, result.TargetInstalls, result.PluginUpdates)
	return result, nil
}

func updatePlan(opts UpdateOptions) (refreshPlan, error) {
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
	home, err := os.UserHomeDir()
	if err == nil {
		plan.ClaudePlugin = pluginInstalled(filepath.Join(home, ".claude"))
	}
	return plan, nil
}

func fetchReleaseBinary(repo, tag string, out io.Writer) (string, func(), error) {
	asset, err := releaseAssetName()
	if err != nil {
		return "", func() {}, err
	}
	tmp, err := os.MkdirTemp("", "machinery-update")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	base := githubBase + "/" + repo + "/releases/download/" + tag
	binary := filepath.Join(tmp, asset)
	checksums := filepath.Join(tmp, "checksums-sha256.txt")
	fmt.Fprintf(out, "fetching machinery %s (%s/%s)\n", tag, runtime.GOOS, runtime.GOARCH)
	if err := download(base+"/"+asset, binary); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("download %s: %w", asset, err)
	}
	if err := download(base+"/checksums-sha256.txt", checksums); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("release %s has no checksums-sha256.txt: %w", tag, err)
	}
	want, err := checksumForAsset(checksums, asset)
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	raw, err := os.ReadFile(binary)
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	got := fmt.Sprintf("%x", sha256.Sum256(raw))
	if got != want {
		cleanup()
		return "", func() {}, fmt.Errorf("checksum mismatch for %s (want %s, got %s)", asset, want, got)
	}
	if err := os.Chmod(binary, 0o755); err != nil {
		cleanup()
		return "", func() {}, err
	}
	fmt.Fprintln(out, "checksum verified")
	return binary, cleanup, nil
}

func releaseAssetName() (string, error) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		return "", fmt.Errorf("unsupported architecture for self-update: %s", runtime.GOARCH)
	}
	switch runtime.GOOS {
	case "darwin", "linux":
		return "machinery-" + runtime.GOOS + "-" + runtime.GOARCH, nil
	case "windows":
		return "machinery-windows-" + runtime.GOARCH + ".exe", nil
	default:
		return "", fmt.Errorf("unsupported operating system for self-update: %s", runtime.GOOS)
	}
}

func checksumForAsset(path, asset string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if name == asset {
			return strings.ToLower(fields[0]), nil
		}
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

func replaceExecutable(candidate, destination string) error {
	dir := filepath.Dir(destination)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	in, err := os.Open(candidate)
	if err != nil {
		return err
	}
	defer in.Close()
	staged, err := os.CreateTemp(dir, ".machinery-update-*")
	if err != nil {
		return err
	}
	stagedPath := staged.Name()
	defer os.Remove(stagedPath)
	if _, err := io.Copy(staged, in); err != nil {
		staged.Close()
		return err
	}
	if err := staged.Chmod(0o755); err != nil {
		staged.Close()
		return err
	}
	if err := staged.Sync(); err != nil {
		staged.Close()
		return err
	}
	if err := staged.Close(); err != nil {
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

func refreshHostPlugins(plan refreshPlan, run commandRunner, lookPath pathLookup, out io.Writer) (int, []string) {
	updates := 0
	var warnings []string
	warn := func(message string) {
		warnings = append(warnings, message)
		fmt.Fprintln(out, "warning: "+message)
	}

	if plan.ClaudePlugin {
		claude, err := lookPath("claude")
		if err != nil {
			warn("Claude Code machinery plugin detected, but 'claude' is not on PATH; run 'claude plugin update machinery@machinery'")
		} else {
			_, _ = run(claude, "plugin", "marketplace", "update", "machinery")
			var failures []string
			claudeUpdates := 0
			for _, scope := range []string{"user", "project", "local"} {
				if output, err := run(claude, "plugin", "update", "machinery@machinery", "--scope", scope); err != nil {
					if text := strings.TrimSpace(output); text != "" {
						failures = append(failures, scope+": "+text)
					}
					continue
				}
				claudeUpdates++
			}
			if claudeUpdates == 0 {
				detail := strings.Join(failures, "; ")
				if detail != "" {
					detail = " (" + detail + ")"
				}
				warn("Claude Code plugin refresh failed; run 'claude plugin update machinery@machinery'" + detail)
			} else {
				updates += claudeUpdates
				fmt.Fprintf(out, "refreshed Claude Code plugin machinery@machinery in %d scope(s)\n", claudeUpdates)
			}
		}
	}

	if codex, err := lookPath("codex"); err == nil {
		listed, listErr := run(codex, "plugin", "list")
		if listErr == nil && strings.Contains(strings.ToLower(listed), "machinery") {
			if output, err := run(codex, "plugin", "add", "machinery@machinery"); err != nil {
				warn("Codex machinery plugin refresh failed; run 'codex plugin add machinery@machinery' (" + strings.TrimSpace(output) + ")")
			} else {
				updates++
				fmt.Fprintln(out, "refreshed Codex plugin machinery@machinery")
			}
		}
	}
	return updates, warnings
}

func runCombined(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(output), err
}
