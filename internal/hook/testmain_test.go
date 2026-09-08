package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hookTestControlRootEnv names the sandbox this package's tests share. A test
// that re-executes the test binary to reproduce a crash must land in the same
// store as its parent, so an inherited root is honored instead of replaced.
const hookTestControlRootEnv = "MACHINERY_INTERNAL_TEST_HOOK_CONTROL_ROOT"

var hookTestControlRoot string

// TestMain redirects every user-scoped path this package writes to before any
// test runs. Without it the suite arms real governance obligations in the
// invoking user's own hook state store: one ledger, and usually one route
// snapshot, per temporary project root, none of which any Stop event ever
// discharges because the root is deleted when the test ends. A repeated local
// sweep is then enough to push that store past its fail-closed entry limit and
// block every shell and write tool on the machine. Test state belongs to the
// test binary, never to the user running it.
func TestMain(m *testing.M) {
	if inherited := os.Getenv(hookTestControlRootEnv); inherited != "" {
		hookTestControlRoot = inherited
		os.Exit(m.Run())
	}
	root, err := os.MkdirTemp("", "machinery-hook-test-control-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "create hook test control root:", err)
		os.Exit(2)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		fmt.Fprintln(os.Stderr, "confine hook test control root:", err)
		_ = os.RemoveAll(root)
		os.Exit(2)
	}
	hookTestControlRoot = root
	directories := map[string]string{
		"home":         filepath.Join(root, "home"),
		"config":       filepath.Join(root, "config"),
		"data":         filepath.Join(root, "data"),
		"cache":        filepath.Join(root, "cache"),
		"state":        filepath.Join(root, "state"),
		"appdata":      filepath.Join(root, "appdata"),
		"localappdata": filepath.Join(root, "localappdata"),
		"temp":         filepath.Join(root, "temp"),
	}
	for _, directory := range directories {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			fmt.Fprintln(os.Stderr, "create hook test sandbox directory:", err)
			_ = os.RemoveAll(root)
			os.Exit(2)
		}
		if err := os.Chmod(directory, 0o700); err != nil {
			fmt.Fprintln(os.Stderr, "confine hook test sandbox directory:", err)
			_ = os.RemoveAll(root)
			os.Exit(2)
		}
	}
	for _, item := range []struct{ key, value string }{
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
		{hookTestControlRootEnv, root},
	} {
		if err := os.Setenv(item.key, item.value); err != nil {
			fmt.Fprintf(os.Stderr, "set hook test sandbox %s: %v\n", item.key, err)
			_ = os.RemoveAll(root)
			os.Exit(2)
		}
	}
	code := m.Run()
	if err := os.RemoveAll(root); err != nil && code == 0 {
		fmt.Fprintln(os.Stderr, "remove hook test control root:", err)
		code = 2
	}
	os.Exit(code)
}

// TestHookTestStateStaysInsideItsOwnSandbox is the standing guard for the
// leak this package caused: it proves that the store these tests arm is the
// test binary's own, not the store the user's editor and agents depend on.
func TestHookTestStateStaysInsideItsOwnSandbox(t *testing.T) {
	if hookTestControlRoot == "" || os.Getenv(hookTestControlRootEnv) != hookTestControlRoot {
		t.Fatalf("hook test control root = %q, environment %q", hookTestControlRoot, os.Getenv(hookTestControlRootEnv))
	}
	for _, key := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME", "APPDATA", "LOCALAPPDATA", "TMPDIR", "TMP", "TEMP"} {
		if value := os.Getenv(key); !hookTestPathWithin(value, hookTestControlRoot) {
			t.Errorf("%s escaped the hook test control root: %q", key, value)
		}
	}
	dir, err := stateDirPathExact()
	if err != nil {
		t.Fatal(err)
	}
	if !hookTestPathWithin(dir, hookTestControlRoot) {
		t.Fatalf("hook state store escaped the test control root: %s", dir)
	}
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		t.Fatal(err)
	}
	if !hookTestPathWithin(marker, hookTestControlRoot) {
		t.Fatalf("hook state initialization marker escaped the test control root: %s", marker)
	}
}

func hookTestPathWithin(path, root string) bool {
	path, pathErr := filepath.Abs(path)
	root, rootErr := filepath.Abs(root)
	if pathErr != nil || rootErr != nil {
		return false
	}
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}
