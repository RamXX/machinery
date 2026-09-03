package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/checker"
	"github.com/RamXX/machinery/internal/install"
	"github.com/RamXX/machinery/internal/processcontrol"
	"github.com/RamXX/machinery/internal/runtimeclosure"
	machversion "github.com/RamXX/machinery/internal/version"
)

const modelithVersion = "v0.4.0"

const diagnosticCommandOutputLimit = 1 << 20

var (
	diagnosticCommandTimeout   = 10 * time.Second
	diagnosticCommandWaitDelay = processcontrol.DefaultWaitDelay
)

// preflightRun mirrors the Makefile `preflight` target: checks the same labels
// (ok/MISSING/optional/auto) so `machinery preflight` and `make preflight` agree.
func preflightRun() error {
	return preflightRunTo(stdoutW)
}

func preflightRunTo(out io.Writer) error {
	return install.WithInstallInspectionLock(func() error { return preflightRunUnlockedTo(out) })
}

func preflightRunUnlocked() error { return preflightRunUnlockedTo(stdoutW) }

func preflightRunUnlockedTo(out io.Writer) error {
	var failures []error
	fmt.Fprintln(out, "machinery prerequisites:")

	// modelith
	if p, absent, lookupErr := lookupDiagnosticExecutable("modelith"); lookupErr != nil {
		fmt.Fprintf(out, "  ERROR    modelith executable resolution failed: %v\n", lookupErr)
		failures = append(failures, lookupErr)
	} else if !absent {
		versionOutput, runErr := runCommand(p, false, "--version")
		if runErr != nil {
			fmt.Fprintf(out, "  ERROR    modelith probe failed: %v\n", runErr)
			failures = append(failures, fmt.Errorf("modelith probe: %w", runErr))
		} else if installed, parseErr := parseModelithVersion(versionOutput); parseErr != nil {
			fmt.Fprintf(out, "  ERROR    modelith probe returned non-canonical identity: %v\n", parseErr)
			failures = append(failures, fmt.Errorf("modelith identity: %w", parseErr))
		} else if strings.TrimPrefix(installed, "v") == strings.TrimPrefix(modelithVersion, "v") {
			fmt.Fprintf(out, "  ok       modelith %s (pinned %s)\n", installed, modelithVersion)
		} else {
			fmt.Fprintf(out, "  ERROR    modelith %s does not match the pin %s -- install: go install github.com/stacklok/modelith/cmd/modelith@%s (or make install-modelith)\n", installed, modelithVersion, modelithVersion)
			failures = append(failures, fmt.Errorf("modelith %q does not match pin %s", installed, modelithVersion))
		}
	} else {
		fmt.Fprintf(out, "  MISSING  modelith (Phase 1 domain model lint/render) -- install: go install github.com/stacklok/modelith/cmd/modelith@%s (or make install-modelith)\n", modelithVersion)
		failures = append(failures, errors.New("modelith is not on PATH"))
	}

	// the gate tools and generators are this binary itself: nothing else needed
	fmt.Fprintln(out, "  ok       machinery "+version+" (the deterministic gate tools and formal generators are this binary; no Python, no other runtime)")

	// Java is optional, but when present it must be the exact runtime policy
	// used by CI so solver output is not dependent on an ambient JVM.
	var java *runtimeclosure.Java
	var javaEnv []string
	javaAvailable := os.Getenv(runtimeclosure.JavaEnv) != ""
	javaLookupFailed := false
	if !javaAvailable {
		_, absent, lookupErr := lookupDiagnosticExecutable("java")
		switch {
		case lookupErr != nil:
			fmt.Fprintf(out, "  ERROR    java executable resolution failed: %v\n", lookupErr)
			failures = append(failures, lookupErr)
			javaLookupFailed = true
		case !absent:
			javaAvailable = true
		}
	}
	javaRuntimeDir := ""
	if javaAvailable {
		var err error
		javaRuntimeDir, err = os.MkdirTemp("", "machinery-preflight-java-")
		if err == nil {
			java, err = openC4Java(javaRuntimeDir)
		}
		if err != nil {
			diagnostic := strings.ReplaceAll(err.Error(), javaRuntimeDir, "<java-home>")
			fmt.Fprintf(out, "  ERROR    java runtime policy probe failed: %v\n", diagnostic)
			failures = append(failures, fmt.Errorf("java runtime policy: %w", err))
		} else {
			javaEnv = runtimeclosure.Environment(javaRuntimeDir, javaRuntimeDir, java.Path())
			fmt.Fprintf(out, "  ok       Java %d (runtime identity bound; CI uses %s)\n", runtimeclosure.RequiredJavaMajor, runtimeclosure.CIJavaVendor)
		}
	} else if !javaLookupFailed {
		fmt.Fprintf(out, "  optional OpenJDK/HotSpot Java %s -- needed only for engine-backed formal/C4 verification. https://adoptium.net/\n", runtimeclosure.RequiredJavaRelease)
	}
	fmt.Fprintln(out, "  auto     'machinery verify-formal' downloads the TLA+ tools (tla2tools.jar) and, for designs with a relational annotation (policy, integrity, or isolation), the Alloy analyzer (org.alloytools.alloy.dist.jar) on first use, pinned and checksum-verified (that step needs Java)")

	// Structurizr is auto-provisioned by verify-c4. Doctor probes only an
	// explicit override, and only with its paired closure trust root.
	structurizr := os.Getenv(structurizrEnv)
	if structurizr == "" {
		fmt.Fprintln(out, "  auto     verify-c4 provisions the checksum-pinned Structurizr CLI; no ambient executable is trusted")
	} else {
		want := machversion.StructurizrVersion
		var err error
		var cleanup func() error
		structurizr, cleanup, err = snapshotStructurizrExecutable(structurizr)
		if err == nil {
			digest, digestErr := fingerprintStructurizrTree(filepath.Dir(structurizr))
			digestText := fmt.Sprintf("%x", digest)
			paired := os.Getenv(structurizrClosureSHAEnv)
			if digestErr != nil || len(paired) != sha256.Size*2 || paired != strings.ToLower(paired) || paired != digestText {
				err = errors.Join(digestErr, fmt.Errorf("explicit Structurizr override does not match paired %s trust root", structurizrClosureSHAEnv))
			}
		}
		if err == nil && java == nil {
			err = fmt.Errorf("structurizr requires the supported OpenJDK/HotSpot Java %s runtime", runtimeclosure.RequiredJavaRelease)
		} else if err == nil && java != nil {
			if validateErr := java.Validate(); validateErr != nil {
				err = validateErr
			}
		}
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err = verifyStructurizrVersion(ctx, structurizr, want, javaEnv)
			cancel()
		}
		if cleanup != nil {
			err = errors.Join(err, cleanup())
		}
		if err != nil {
			fmt.Fprintf(out, "  ERROR    structurizr-cli probe failed: %v\n", err)
			failures = append(failures, err)
		} else {
			fmt.Fprintf(out, "  ok       structurizr-cli %s (matches the binary's supported engine and paired closure trust root)\n", want)
		}
	}

	// scorecard (optional)
	if p, absent, lookupErr := lookupDiagnosticExecutable("scorecard"); lookupErr != nil {
		fmt.Fprintf(out, "  ERROR    scorecard executable resolution failed: %v\n", lookupErr)
		failures = append(failures, lookupErr)
	} else if !absent {
		versionOutput, runErr := runCommand(p, false, "version")
		if runErr != nil {
			fmt.Fprintf(out, "  ERROR    scorecard probe failed: %v\n", runErr)
			failures = append(failures, fmt.Errorf("scorecard probe: %w", runErr))
		} else if v, parseErr := parseScorecardVersion(versionOutput); parseErr != nil {
			fmt.Fprintf(out, "  ERROR    scorecard probe returned non-canonical identity: %v\n", parseErr)
			failures = append(failures, fmt.Errorf("scorecard identity: %w", parseErr))
		} else {
			fmt.Fprintf(out, "  ok       scorecard %s (OpenSSF Scorecard: Phase 2 adoption-closure risk evidence; needs GITHUB_AUTH_TOKEN at run time)\n", v)
		}
	} else {
		fmt.Fprintln(out, "  optional scorecard (OpenSSF Scorecard, Phase 2 adoption-closure risk evidence; public GitHub repos need NO install: curl https://api.securityscorecards.dev/projects/github.com/<org>/<repo>) -- install: go install github.com/ossf/scorecard/v5@latest (needs GITHUB_AUTH_TOKEN at run time)")
	}
	if java != nil {
		failures = append(failures, java.Validate(), java.Close())
	}
	if javaRuntimeDir != "" {
		failures = append(failures, os.RemoveAll(javaRuntimeDir))
	}
	return errors.Join(failures...)
}

// lookupDiagnosticExecutable distinguishes a genuinely absent optional tool
// from an unsafe or otherwise failed PATH resolution. In particular ErrDot is
// never reported as "optional" or "missing": executing from the current
// directory would make doctor depend on ambient mutable state.
func lookupDiagnosticExecutable(name string) (path string, absent bool, err error) {
	path, err = exec.LookPath(name)
	if err == nil {
		return path, false, nil
	}
	if errors.Is(err, exec.ErrNotFound) {
		// LookPath deliberately collapses permission, directory, broken-link,
		// and other candidate failures into ErrNotFound while searching PATH.
		// Inspect shadowing candidates so only genuine absence stays optional.
		if candidateErr := diagnosticPATHCandidateError(name); candidateErr != nil {
			return "", false, fmt.Errorf("resolve %s on PATH: %w", name, candidateErr)
		}
		return "", true, nil
	}
	return "", false, fmt.Errorf("resolve %s on PATH: %w", name, err)
}

func diagnosticPATHCandidateError(name string) error {
	names := []string{name}
	if runtime.GOOS == "windows" && filepath.Ext(name) == "" {
		extensions := os.Getenv("PATHEXT")
		if extensions == "" {
			extensions = ".COM;.EXE;.BAT;.CMD"
		}
		for _, extension := range strings.Split(extensions, ";") {
			if extension != "" {
				names = append(names, name+strings.ToLower(extension), name+strings.ToUpper(extension))
			}
		}
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			dir = "."
		}
		for _, candidateName := range names {
			candidate := filepath.Join(dir, candidateName)
			info, err := os.Lstat(candidate)
			switch {
			case err == nil:
				return fmt.Errorf("PATH candidate %s exists but is not an executable regular file (%s)", candidate, info.Mode())
			case errors.Is(err, os.ErrNotExist):
				continue
			default:
				return fmt.Errorf("inspect PATH candidate %s: %w", candidate, err)
			}
		}
	}
	return nil
}

// doctorRun mirrors the Makefile `doctor` target: preflight + install status.
// With targets it checks each host's native adapter; without them it preserves
// the original ~/.claude + ~/.agents report.
func doctorRunTo(targets []string, out io.Writer) error {
	return install.WithInstallInspectionLock(func() error { return doctorRunUnlockedTo(targets, out) })
}

type diagnosticOutput struct {
	destination io.Writer
	err         error
}

func (output *diagnosticOutput) Write(p []byte) (int, error) {
	if output.err != nil {
		return 0, output.err
	}
	written, err := output.destination.Write(p)
	if err == nil && written != len(p) {
		err = io.ErrShortWrite
	}
	if err != nil {
		output.err = fmt.Errorf("write doctor output: %w", err)
		return written, output.err
	}
	return written, nil
}

func doctorRunUnlockedTo(targets []string, out io.Writer) (result error) {
	output := &diagnosticOutput{destination: out}
	out = output
	defer func() { result = errors.Join(result, output.err) }()
	var failures []error
	if len(targets) > 0 {
		artifacts, err := install.TargetArtifacts(targets)
		if err != nil {
			return err
		}
		failures = append(failures, preflightRunUnlockedTo(out))
		if !reportCheckerBinaries(out) {
			failures = append(failures, errors.New("one or more checker prerequisites are unavailable"))
		}
		fmt.Fprintln(out, "install status:")
		for _, artifact := range artifacts {
			if err := install.ValidateArtifact(artifact); err == nil {
				fmt.Fprintf(out, "  ok       [%s] %s at %s\n", artifact.Target, artifact.Label, artifact.Path)
			} else {
				fmt.Fprintf(out, "  ERROR    [%s] %s at %s is invalid: %v -- run machinery install --target %s\n", artifact.Target, artifact.Label, artifact.Path, err, strings.Join(targets, " --target "))
				failures = append(failures, fmt.Errorf("invalid %s artifact %s: %w", artifact.Target, artifact.Path, err))
			}
		}
		if !reportHookWiring(out) {
			failures = append(failures, errors.New("governance hook wiring is incomplete"))
		}
		if !reportUpdateReceipt(out) {
			failures = append(failures, errors.New("update receipt is unreadable"))
		}
		return errors.Join(failures...)
	}

	failures = append(failures, preflightRunUnlockedTo(out))
	if !reportCheckerBinaries(out) {
		failures = append(failures, errors.New("one or more checker prerequisites are unavailable"))
	}
	fmt.Fprintln(out, "install status:")
	homes, homesErr := install.DefaultHomes()
	if homesErr != nil {
		fmt.Fprintf(out, "  ERROR    default install homes cannot be resolved: %v\n", homesErr)
		failures = append(failures, homesErr)
	}
	for _, home := range homes {
		target := "shared"
		if filepath.Base(home) == ".claude" {
			target = "claude"
		}
		if err := install.ValidateArtifact(install.Artifact{Target: target, Label: "machinery skill", Path: filepath.Join(home, "skills", "machinery")}); err == nil {
			fmt.Fprintf(out, "  ok       skill at %s/skills/machinery\n", home)
		} else {
			fmt.Fprintf(out, "  ERROR    skill at %s/skills/machinery is invalid: %v -- run machinery install --target all\n", home, err)
			failures = append(failures, fmt.Errorf("invalid machinery skill under %s: %w", home, err))
		}
		if err := install.ValidateArtifact(install.Artifact{Target: target, Label: "machinery-fsm-author agent", Path: filepath.Join(home, "agents", "machinery-fsm-author.md")}); err == nil {
			fmt.Fprintf(out, "  ok       fsm-author role at %s/agents\n", home)
		} else {
			fmt.Fprintf(out, "  ERROR    fsm-author role at %s/agents is invalid: %v -- run machinery install --target all\n", home, err)
			failures = append(failures, fmt.Errorf("invalid fsm-author role under %s: %w", home, err))
		}
		if err := install.ValidateArtifact(install.Artifact{Target: target, Label: "machinery-build-writer agent", Path: filepath.Join(home, "agents", "machinery-build-writer.md")}); err == nil {
			fmt.Fprintf(out, "  ok       build-writer role at %s/agents\n", home)
		} else {
			fmt.Fprintf(out, "  ERROR    build-writer role at %s/agents is invalid: %v -- run machinery install --target all\n", home, err)
			failures = append(failures, fmt.Errorf("invalid build-writer role under %s: %w", home, err))
		}
	}
	if !reportHookWiring(out) {
		failures = append(failures, errors.New("governance hook wiring is incomplete"))
	}
	if !reportUpdateReceipt(out) {
		failures = append(failures, errors.New("update receipt is unreadable"))
	}
	return errors.Join(failures...)
}

// reportCheckerBinaries checks that every checker configured in the local
// registry has its run binary on PATH, mirroring the java/structurizr/scorecard
// prerequisite style. It runs ONLY when a registry exists in the current
// directory; with no registry the doctor output is byte-for-byte unchanged, so
// a design that never opted into external checkers sees nothing new.
func reportCheckerBinaries(out io.Writer) bool {
	path := checker.DefaultRegistryPath
	if _, err := os.Stat(path); err != nil {
		return os.IsNotExist(err)
	}
	reg, err := checker.LoadRegistry(path)
	if err != nil {
		fmt.Fprintf(out, "  MISSING  checker registry %s is unreadable: %v\n", path, err)
		return false
	}
	ids := reg.IDs()
	if len(ids) == 0 {
		fmt.Fprintf(out, "  auto     checker registry %s has no checkers configured\n", path)
		return true
	}
	ok := true
	for _, id := range ids {
		entry, _ := reg.Resolve(id)
		engine := entry.Runtime.Engine[0]
		if p, err := doctorCheckerExecutable(engine); err == nil {
			fmt.Fprintf(out, "  present  checker %s OCI engine %s is snapshot-safe at %s for immutable image %s on %s (not executed by doctor)\n", id, engine, p, entry.Runtime.Image, entry.Runtime.Platform)
		} else {
			fmt.Fprintf(out, "  MISSING  checker %s OCI engine %s is unavailable or unsafe: %v -- 'machinery verify-checkers' cannot inspect or run %s\n", id, engine, err, entry.Runtime.Image)
			ok = false
		}
	}
	return ok
}

func doctorCheckerExecutable(command string) (string, error) {
	resolved, err := exec.LookPath(command)
	if err != nil {
		return "", err
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("resolved path %s is not a regular, non-symlink file", resolved)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("resolved path %s has no execute bit", resolved)
	}
	return resolved, nil
}

// reportHookWiring checks the machinery plugin hook plumbing wherever a
// plugin layout exists (a .claude-plugin/plugin.json with a hooks/ sibling):
// hooks.json must be present and the shim executable, or every governance
// hook silently never fires (GATE-11 doctor check).
func reportHookWiring(out io.Writer) bool {
	roots, discoveryErr := pluginRoots()
	ok := discoveryErr == nil
	if discoveryErr != nil {
		fmt.Fprintf(out, "  ERROR    machinery plugin discovery failed: %v\n", discoveryErr)
	}
	if len(roots) == 0 {
		if !ok {
			return false
		}
		fmt.Fprintln(out, "  auto     no machinery plugin layout found (.claude-plugin/ + hooks/); governance hooks run only where the plugin is installed")
		return true
	}
	for _, root := range roots {
		pluginManifest := filepath.Join(root, ".claude-plugin", "plugin.json")
		var pluginIdentity doctorPluginManifest
		identityErr := readDoctorJSON(pluginManifest, &pluginIdentity)
		if identityErr == nil && pluginIdentity.Name != "machinery" {
			identityErr = fmt.Errorf("name is %q, want machinery", pluginIdentity.Name)
		}
		if identityErr == nil && strings.TrimPrefix(pluginIdentity.Version, "v") != strings.TrimPrefix(version, "v") {
			identityErr = fmt.Errorf("version is %q, want %s", pluginIdentity.Version, version)
		}
		if identityErr != nil {
			fmt.Fprintf(out, "  ERROR    plugin manifest at %s has the wrong identity or is corrupt: %v\n", pluginManifest, identityErr)
			ok = false
		}
		manifest := filepath.Join(root, "hooks", "hooks.json")
		if err := validateDoctorHookManifest(manifest); err == nil {
			fmt.Fprintf(out, "  ok       hook manifest at %s\n", manifest)
		} else {
			fmt.Fprintf(out, "  ERROR    hook manifest at %s is missing, corrupt, or empty: %v -- no governance hook will fire\n", manifest, err)
			ok = false
		}
		shim := filepath.Join(root, "hooks", "machinery-hook.sh")
		if fi, err := os.Lstat(shim); err != nil {
			fmt.Fprintf(out, "  MISSING  hook shim at %s -- the hooks.json entries point at nothing\n", shim)
			ok = false
		} else if !fi.Mode().IsRegular() || fi.Mode()&os.ModeSymlink != 0 {
			fmt.Fprintf(out, "  ERROR    hook shim at %s is not a real regular file\n", shim)
			ok = false
		} else if fi.Mode()&0o111 == 0 {
			fmt.Fprintf(out, "  ERROR    hook shim at %s is not executable -- chmod +x it or every hook invocation fails silently\n", shim)
			ok = false
		} else if err := validateDoctorHookShim(shim); err != nil {
			fmt.Fprintf(out, "  ERROR    hook shim at %s has the wrong identity: %v\n", shim, err)
			ok = false
		} else {
			fmt.Fprintf(out, "  ok       hook shim at %s (executable, canonical digest)\n", shim)
		}
	}
	return ok
}

func readDoctorJSON(path string, dst any) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("not a real regular file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := rejectDoctorDuplicateJSONKeys(raw); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("trailing JSON content")
	}
	return nil
}

type doctorHookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
	Async   *bool  `json:"async,omitempty"`
}

type doctorHookBinding struct {
	Matcher string              `json:"matcher"`
	Hooks   []doctorHookCommand `json:"hooks"`
}

type doctorHookManifest struct {
	Description string                         `json:"description"`
	Hooks       map[string][]doctorHookBinding `json:"hooks"`
}

type doctorPluginAuthor struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type doctorPluginManifest struct {
	Name        string             `json:"name"`
	Version     string             `json:"version"`
	Description string             `json:"description"`
	Author      doctorPluginAuthor `json:"author"`
	Homepage    string             `json:"homepage"`
	Repository  string             `json:"repository"`
	License     string             `json:"license"`
	Keywords    []string           `json:"keywords"`
}

func validateDoctorHookManifest(path string) error {
	var manifest doctorHookManifest
	if err := readDoctorJSON(path, &manifest); err != nil {
		return err
	}
	type expectedHookEvent struct {
		event   string
		matcher string
		timeout int
	}
	expected := []expectedHookEvent{
		{"PreToolUse", "Edit|Write|MultiEdit|NotebookEdit|apply_patch|edit|write|patch|Bash|bash|Shell|shell", 15},
		{"PostToolUse", "Edit|Write|MultiEdit|NotebookEdit|apply_patch|edit|write|patch|Bash|bash|Shell|shell", 15},
		{"PostToolUseFailure", "Edit|Write|MultiEdit|NotebookEdit|apply_patch|edit|write|patch|Bash|bash|Shell|shell", 15},
		{"Stop", "*", 180},
		{"SubagentStop", "*", 180},
		{"SessionStart", "startup|resume|clear|compact", 15},
	}
	if strings.TrimSpace(manifest.Description) == "" || len(manifest.Hooks) != len(expected) {
		return fmt.Errorf("hook inventory is incomplete or has unexpected events")
	}
	for _, want := range expected {
		bindings, ok := manifest.Hooks[want.event]
		if !ok || len(bindings) != 1 || bindings[0].Matcher != want.matcher || len(bindings[0].Hooks) != 1 {
			return fmt.Errorf("event %s does not have the canonical matcher and single command", want.event)
		}
		command := bindings[0].Hooks[0]
		if command.Type != "command" || command.Command != "${CLAUDE_PLUGIN_ROOT}/hooks/machinery-hook.sh" || command.Timeout != want.timeout {
			return fmt.Errorf("event %s does not wire the canonical shim contract", want.event)
		}
		if command.Async != nil {
			return fmt.Errorf("event %s must use the canonical synchronous topology (no async override)", want.event)
		}
	}
	return nil
}

// canonicalHookShimSHA256 is the release identity of hooks/machinery-hook.sh.
// TestCanonicalDoctorHookAssetsMatchRepository makes source edits fail until
// this release contract is advanced deliberately with the shipped shim.
const canonicalHookShimSHA256 = "9dd7543bd7c3bf17adc56228e10174a6d5f1a859585c5a5b1d11906714b6ec5c"

func validateDoctorHookShim(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(raw))
	if digest != canonicalHookShimSHA256 {
		return fmt.Errorf("sha256 is %s, want %s", digest, canonicalHookShimSHA256)
	}
	return nil
}

func rejectDoctorDuplicateJSONKeys(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	var parseValue func() error
	parseValue = func() error {
		token, err := dec.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
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
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return err
	}
	return nil
}

// pluginRoots finds machinery plugin layouts: $CLAUDE_PLUGIN_ROOT, the
// current directory, and any ~/.claude/plugins entry that carries a
// machinery hook shim. A root qualifies when .claude-plugin/plugin.json and
// a hooks/ directory both exist.
func pluginRoots() ([]string, error) {
	var roots []string
	seen := map[string]bool{}
	add := func(dir string, explicit bool) {
		if dir == "" || seen[dir] {
			return
		}
		if !explicit {
			_, manifestErr := os.Lstat(filepath.Join(dir, ".claude-plugin", "plugin.json"))
			_, hooksErr := os.Lstat(filepath.Join(dir, "hooks"))
			if os.IsNotExist(manifestErr) && os.IsNotExist(hooksErr) {
				return
			}
		}
		seen[dir] = true
		roots = append(roots, dir)
	}
	add(os.Getenv("CLAUDE_PLUGIN_ROOT"), true)
	if wd, err := os.Getwd(); err == nil {
		add(wd, false)
	}
	plugins := filepath.Join(os.Getenv("HOME"), ".claude", "plugins")
	if entries, err := os.ReadDir(plugins); err == nil {
		for _, e := range entries {
			if strings.Contains(strings.ToLower(e.Name()), "machinery") {
				add(filepath.Join(plugins, e.Name()), true)
			}
		}
	} else if !os.IsNotExist(err) {
		return roots, fmt.Errorf("enumerate %s: %w", plugins, err)
	}
	return roots, nil
}

func reportUpdateReceipt(out io.Writer) bool {
	status, err := install.InstallationReceiptStatus()
	switch {
	case err != nil:
		fmt.Fprintf(out, "  ERROR    installation receipt at %s is unreadable or unsafe: %v -- run machinery install or machinery update\n", status.Path, err)
		return false
	case status.Exists && status.SchemaVersion == 2:
		fmt.Fprintf(out, "  ok       update receipt at %s (%d home group(s), %d native target(s))\n", status.Path, status.HomeInstalls, status.Targets)
	case status.Exists:
		fmt.Fprintf(out, "  ERROR    installation receipt at %s uses legacy schema %d without artifact digests -- run machinery update\n", status.Path, status.SchemaVersion)
		return false
	default:
		fmt.Fprintf(out, "  ERROR    no schema-2 installation receipt at %s -- run machinery install or machinery update\n", status.Path)
		return false
	}
	return true
}

func runCommand(name string, combined bool, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), diagnosticCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = diagnosticCommandWaitDelay
	var output, stderr string
	var err error
	if combined {
		output, err = processcontrol.RunCaptured(ctx, cmd, diagnosticCommandOutputLimit, true)
	} else {
		output, stderr, err = processcontrol.RunCapturedStreams(ctx, cmd, diagnosticCommandOutputLimit)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return output, fmt.Errorf("%s %s timed out after %s; process tree was terminated: %w", name, strings.Join(args, " "), diagnosticCommandTimeout, errors.Join(context.DeadlineExceeded, err))
	}
	if errors.Is(err, exec.ErrWaitDelay) {
		return output, fmt.Errorf("%s %s descendant held output pipes open beyond %s; process tree was terminated: %w", name, strings.Join(args, " "), diagnosticCommandWaitDelay, err)
	}
	if err != nil {
		if stderr != "" {
			return output, fmt.Errorf("%s %s: %w; stderr: %q", name, strings.Join(args, " "), err, stderr)
		}
		return output, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	if stderr != "" {
		return output, fmt.Errorf("%s %s wrote to stderr: %q", name, strings.Join(args, " "), stderr)
	}
	return output, nil
}

var (
	modelithVersionOutputRE = regexp.MustCompile(`^modelith version (v?[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?)$`)
	scorecardVersionLineRE  = regexp.MustCompile(`^GitVersion:[ \t]+(v?[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?)$`)
)

func parseModelithVersion(output string) (string, error) {
	line := strings.TrimSuffix(output, "\n")
	line = strings.TrimSuffix(line, "\r")
	match := modelithVersionOutputRE.FindStringSubmatch(line)
	if len(match) != 2 {
		return "", fmt.Errorf("expected exactly %q, got %q", "modelith version <semver>", output)
	}
	return match[1], nil
}

func parseScorecardVersion(output string) (string, error) {
	var version string
	for _, rawLine := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		match := scorecardVersionLineRE.FindStringSubmatch(rawLine)
		if !strings.HasPrefix(strings.TrimSpace(rawLine), "GitVersion:") {
			continue
		}
		if len(match) != 2 {
			return "", fmt.Errorf("non-canonical GitVersion identity line %q", rawLine)
		}
		if version != "" {
			return "", fmt.Errorf("duplicate GitVersion identity")
		}
		version = match[1]
	}
	if version == "" {
		return "", fmt.Errorf("missing canonical GitVersion: <semver> line")
	}
	return version, nil
}

var javaVersionRE = regexp.MustCompile(`(?i)version\s+"?([0-9]+)(?:\.([0-9]+))?`)

func javaMajor(versionLine string) (int, error) {
	match := javaVersionRE.FindStringSubmatch(versionLine)
	if len(match) == 0 {
		return 0, fmt.Errorf("unrecognized java version output")
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("parse java major version: %w", err)
	}
	if major == 1 && match[2] != "" {
		major, err = strconv.Atoi(match[2])
		if err != nil {
			return 0, fmt.Errorf("parse legacy java major version: %w", err)
		}
	}
	return major, nil
}
