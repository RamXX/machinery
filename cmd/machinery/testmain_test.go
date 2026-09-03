package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/install"
)

const (
	cmdTestControlRootEnv = "MACHINERY_TEST_CONTROL_ROOT"
	cmdTestRootReportEnv  = "MACHINERY_TEST_CONTROL_ROOT_REPORT"
)

var (
	cmdTestControlRoot     string
	cmdTestNeutralTempRoot string
)

// TestMain redirects every user-scoped control path before any test, golden
// binary, or helper subprocess can acquire Machinery's startup/install lock.
// Explicit t.Setenv overrides inside individual tests remain authoritative and
// are restored to this sandbox, never to the invoking user's environment.
func TestMain(m *testing.M) {
	originalHome, _ := os.UserHomeDir()
	originalCache, _ := os.UserCacheDir()
	cmdTestNeutralTempRoot = os.TempDir()
	preserveGoToolPaths(originalHome, originalCache)

	root, err := os.MkdirTemp("", "machinery-cmd-test-control-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "create cmd test control root:", err)
		os.Exit(2)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		fmt.Fprintln(os.Stderr, "confine cmd test control root:", err)
		_ = os.RemoveAll(root)
		os.Exit(2)
	}
	cmdTestControlRoot = root
	type namedDirectory struct {
		name string
		path string
	}
	directoryList := []namedDirectory{
		{"home", filepath.Join(root, "home")},
		{"config", filepath.Join(root, "config")},
		{"data", filepath.Join(root, "data")},
		{"cache", filepath.Join(root, "cache")},
		{"state", filepath.Join(root, "state")},
		{"temp", filepath.Join(root, "temp")},
		{"appdata", filepath.Join(root, "appdata")},
		{"localappdata", filepath.Join(root, "localappdata")},
		{"install", filepath.Join(root, "install-control")},
	}
	directories := make(map[string]string, len(directoryList))
	for _, directory := range directoryList {
		directories[directory.name] = directory.path
		if err := os.MkdirAll(directory.path, 0o700); err != nil {
			fmt.Fprintln(os.Stderr, "create cmd test sandbox directory:", err)
			_ = os.RemoveAll(root)
			os.Exit(2)
		}
		if err := os.Chmod(directory.path, 0o700); err != nil {
			fmt.Fprintln(os.Stderr, "confine cmd test sandbox directory:", err)
			_ = os.RemoveAll(root)
			os.Exit(2)
		}
	}
	environment := []struct{ key, value string }{
		{"HOME", directories["home"]},
		{"USERPROFILE", directories["home"]},
		{"XDG_CONFIG_HOME", directories["config"]},
		{"XDG_DATA_HOME", directories["data"]},
		{"XDG_CACHE_HOME", directories["cache"]},
		{"XDG_STATE_HOME", directories["state"]},
		{"APPDATA", directories["appdata"]},
		{"LOCALAPPDATA", directories["localappdata"]},
		{"TMPDIR", directories["temp"]},
		{"TMP", directories["temp"]},
		{"TEMP", directories["temp"]},
		{"MACHINERY_CONFIG_DIR", directories["install"]},
		{cmdTestControlRootEnv, root},
	}
	preserveCheckerLogicalPaths := checkerFixtureInvocation()
	for _, item := range environment {
		if preserveCheckerLogicalPaths && (item.key == "HOME" || item.key == "USERPROFILE" || item.key == "TMPDIR" || item.key == "TMP" || item.key == "TEMP") {
			continue
		}
		if err := os.Setenv(item.key, item.value); err != nil {
			fmt.Fprintf(os.Stderr, "set cmd test sandbox %s: %v\n", item.key, err)
			_ = os.RemoveAll(root)
			os.Exit(2)
		}
	}
	for _, key := range []string{
		"MACHINERY_INTERNAL_ACTIVATION_REEXEC_GUARD",
		"MACHINERY_INTERNAL_CANONICAL_EXECUTABLE",
		"MACHINERY_INTERNAL_INSTALL_LOCK_CAPABILITY",
		"MACHINERY_BIN",
		"MACHINERY_HOMES",
		"MACHINERY_SKILL_SRC",
		"MACHINERY_TARGETS",
	} {
		_ = os.Unsetenv(key)
	}
	if report := os.Getenv(cmdTestRootReportEnv); report != "" {
		if err := os.WriteFile(report, []byte(root+"\n"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, "report cmd test control root:", err)
			_ = os.RemoveAll(root)
			os.Exit(2)
		}
	}

	code := m.Run()
	if err := os.RemoveAll(root); err != nil && code == 0 {
		fmt.Fprintln(os.Stderr, "remove cmd test control root:", err)
		code = 2
	}
	os.Exit(code)
}

func checkerFixtureInvocation() bool {
	for _, argument := range os.Args {
		if argument == checkerProcessFixtureMarker {
			return true
		}
	}
	return false
}

// Go compiler/module caches are toolchain state, not Machinery user control
// state. Preserve their pre-sandbox locations so golden tests can build the
// CLI without downloading dependencies into the isolated HOME.
func preserveGoToolPaths(home, cache string) {
	goPath := os.Getenv("GOPATH")
	if goPath == "" && home != "" {
		goPath = filepath.Join(home, "go")
		_ = os.Setenv("GOPATH", goPath)
	}
	if os.Getenv("GOMODCACHE") == "" && goPath != "" {
		first, _, _ := strings.Cut(goPath, string(os.PathListSeparator))
		_ = os.Setenv("GOMODCACHE", filepath.Join(first, "pkg", "mod"))
	}
	if os.Getenv("GOCACHE") == "" && cache != "" {
		_ = os.Setenv("GOCACHE", filepath.Join(cache, "go-build"))
	}
}

func TestCmdTestControlStateIsPrivate(t *testing.T) {
	root := os.Getenv(cmdTestControlRootEnv)
	if root == "" || root != cmdTestControlRoot {
		t.Fatalf("cmd test control root = %q, process root %q", root, cmdTestControlRoot)
	}
	for _, key := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME", "APPDATA", "LOCALAPPDATA", "TMPDIR", "TMP", "TEMP", "MACHINERY_CONFIG_DIR"} {
		value := os.Getenv(key)
		if !pathWithin(value, root) {
			t.Errorf("%s escaped cmd test control root: %q", key, value)
		}
	}
	status, err := install.InstallationReceiptStatus()
	if err != nil {
		t.Fatal(err)
	}
	if !pathWithin(status.Path, root) {
		t.Fatalf("installation receipt escaped test root: %s", status.Path)
	}
	if err := install.WithInstallInspectionLock(func() error { return nil }); err != nil {
		t.Fatalf("private startup/install lock: %v", err)
	}
}

func TestConcurrentCmdTestBinariesUseDisjointControlState(t *testing.T) {
	reports := []string{filepath.Join(t.TempDir(), "first-root"), filepath.Join(t.TempDir(), "second-root")}
	fakeUserRoot := t.TempDir()
	marker := filepath.Join(fakeUserRoot, "must-remain-only-entry")
	if err := os.WriteFile(marker, []byte("sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	type result struct {
		output []byte
		err    error
	}
	results := make(chan result, len(reports))
	ctx := t.Context()
	for _, report := range reports {
		go func(report string) {
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCmdTestLockSubprocess$", "-test.count=1")
			command.Env = cmdTestSubprocessEnvironment(fakeUserRoot, report)
			output, err := command.CombinedOutput()
			results <- result{output: output, err: err}
		}(report)
	}
	for range reports {
		if result := <-results; result.err != nil {
			t.Fatalf("concurrent cmd test subprocess: %v\n%s", result.err, result.output)
		}
	}
	first, err := os.ReadFile(reports[0])
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(reports[1])
	if err != nil {
		t.Fatal(err)
	}
	firstRoot, secondRoot := strings.TrimSpace(string(first)), strings.TrimSpace(string(second))
	if firstRoot == "" || secondRoot == "" || firstRoot == secondRoot {
		t.Fatalf("concurrent subprocess roots = %q and %q", firstRoot, secondRoot)
	}
	entries, err := os.ReadDir(fakeUserRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(marker) {
		t.Fatalf("subprocesses touched inherited user state: %v", entries)
	}
}

func TestCmdTestLockSubprocess(t *testing.T) {
	if err := install.WithInstallInspectionLock(func() error {
		status, err := install.InstallationReceiptStatus()
		if err != nil {
			return err
		}
		if !pathWithin(status.Path, os.Getenv(cmdTestControlRootEnv)) {
			return fmt.Errorf("receipt %s escaped subprocess root", status.Path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func cmdTestSubprocessEnvironment(fakeUserRoot, report string) []string {
	overrides := []struct{ key, value string }{
		{"HOME", fakeUserRoot},
		{"USERPROFILE", fakeUserRoot},
		{"XDG_CONFIG_HOME", fakeUserRoot},
		{"XDG_DATA_HOME", fakeUserRoot},
		{"XDG_CACHE_HOME", fakeUserRoot},
		{"XDG_STATE_HOME", fakeUserRoot},
		{"APPDATA", fakeUserRoot},
		{"LOCALAPPDATA", fakeUserRoot},
		{"MACHINERY_CONFIG_DIR", fakeUserRoot},
		{cmdTestRootReportEnv, report},
	}
	result := make([]string, 0, len(os.Environ())+len(overrides))
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		replaced := false
		for _, override := range overrides {
			if key == override.key {
				replaced = true
				break
			}
		}
		if !replaced {
			result = append(result, item)
		}
	}
	for _, override := range overrides {
		result = append(result, override.key+"="+override.value)
	}
	return result
}

func pathWithin(path, root string) bool {
	path, pathErr := filepath.Abs(path)
	root, rootErr := filepath.Abs(root)
	if pathErr != nil || rootErr != nil {
		return false
	}
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}
