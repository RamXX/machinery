package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func privateConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

// fakeSource builds a minimal source tree containing the shared skill, the
// canonical role bodies, and the OpenCode adapter assets.
func fakeSource(t *testing.T) string {
	t.Helper()
	// Install operations persist a receipt. Never let a unit test that did not
	// explicitly select a fixture config directory touch the developer's real
	// user configuration.
	if os.Getenv("MACHINERY_CONFIG_DIR") == "" {
		t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	}
	src := t.TempDir()
	skill := filepath.Join(src, "skills", "machinery")
	if err := os.MkdirAll(filepath.Join(skill, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(skill, "SKILL.md"), "---\nname: machinery\n---\n")
	write(t, filepath.Join(skill, "references", "x.md"), "ref\n")
	if err := os.MkdirAll(filepath.Join(src, "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, d := range RoleDocs {
		write(t, filepath.Join(src, "agents", d), "---\nname: role\ndescription: role\ntools: Read, Write\nmodel: opus\n---\n\ncanonical role body for "+d+"\n")
	}
	for _, d := range openCodeCommands {
		write(t, filepath.Join(src, "adapters", "opencode", "commands", d), "---\ndescription: command\n---\n\ncommand "+d+"\n")
	}
	write(t, filepath.Join(src, "adapters", "opencode", "plugins", "machinery.js"), "export const MachineryPlugin = async () => ({})\n")
	return src
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func isSymlink(t *testing.T, path string) bool {
	t.Helper()
	fi, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat %s: %v", path, err)
	}
	return fi.Mode()&os.ModeSymlink != 0
}

func TestInstallCanonicalSymlink(t *testing.T) {
	src := fakeSource(t)
	root := t.TempDir()
	agents := filepath.Join(root, ".agents")
	claude := filepath.Join(root, ".claude")

	if err := Install(Options{Homes: []string{agents, claude}, From: src}); err != nil {
		t.Fatal(err)
	}

	canonSkill := filepath.Join(agents, "skills", "machinery")
	if isSymlink(t, canonSkill) {
		t.Errorf("canonical skill should be a real directory, got a symlink")
	}
	if _, err := os.Stat(filepath.Join(canonSkill, "SKILL.md")); err != nil {
		t.Errorf("canonical skill missing SKILL.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(canonSkill, "references", "x.md")); err != nil {
		t.Errorf("copyTree did not recurse into references/: %v", err)
	}

	linkSkill := filepath.Join(claude, "skills", "machinery")
	if !isSymlink(t, linkSkill) {
		t.Errorf("secondary skill should be a symlink")
	}
	target, err := os.Readlink(linkSkill)
	if err != nil {
		t.Fatal(err)
	}
	if target != canonSkill {
		t.Errorf("symlink target = %s, want %s", target, canonSkill)
	}
	for _, d := range RoleDocs {
		if isSymlink(t, filepath.Join(agents, "agents", d)) {
			t.Errorf("canonical role doc %s should be a real file", d)
		}
		if !isSymlink(t, filepath.Join(claude, "agents", d)) {
			t.Errorf("secondary role doc %s should be a symlink", d)
		}
	}
}

func TestInstallReplacesSymlinkedRoleDoc(t *testing.T) {
	// Regression: a prior symlink-based install leaves the role docs as symlinks.
	// Re-installing must replace them with real files, never write through the
	// symlink into whatever it pointed at (which was the repo on a dev machine).
	src := fakeSource(t)
	root := t.TempDir()
	home := filepath.Join(root, "home")
	external := filepath.Join(root, "external")
	if err := os.MkdirAll(external, 0o755); err != nil {
		t.Fatal(err)
	}
	extDoc := filepath.Join(external, RoleDocs[0])
	if err := os.WriteFile(extDoc, []byte("ORIGINAL"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, "agents", RoleDocs[0])
	if err := os.Symlink(extDoc, link); err != nil {
		t.Fatal(err)
	}

	if err := Install(Options{Homes: []string{home}, From: src, Copy: true}); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(link); err != nil {
		t.Fatal(err)
	} else if fi.Mode()&os.ModeSymlink != 0 {
		t.Errorf("role doc must be a real file after install, still a symlink")
	}
	if b, _ := os.ReadFile(extDoc); string(b) != "ORIGINAL" {
		t.Errorf("install wrote through the symlink into the external file: %q", b)
	}
}

func TestInstallCopyMode(t *testing.T) {
	src := fakeSource(t)
	root := t.TempDir()
	a := filepath.Join(root, ".a")
	b := filepath.Join(root, ".b")

	if err := Install(Options{Homes: []string{a, b}, From: src, Copy: true}); err != nil {
		t.Fatal(err)
	}
	for _, home := range []string{a, b} {
		skill := filepath.Join(home, "skills", "machinery")
		if isSymlink(t, skill) {
			t.Errorf("--copy: %s should be a real directory, got a symlink", skill)
		}
		if _, err := os.Stat(filepath.Join(skill, "SKILL.md")); err != nil {
			t.Errorf("--copy: %s missing SKILL.md: %v", home, err)
		}
	}
}

func TestInstallTargetCodex(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override does not steer os.UserHomeDir on Windows")
	}
	src := fakeSource(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := Install(Options{Targets: []string{"codex"}, From: src}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "machinery", "SKILL.md")); err != nil {
		t.Fatalf("Codex target must install the shared skill: %v", err)
	}
	for _, spec := range roleSpecs {
		path := filepath.Join(home, ".codex", "agents", spec.Name+".toml")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Codex agent missing: %v", err)
		}
		doc := string(raw)
		if !strings.Contains(doc, "developer_instructions = '''") || !strings.Contains(doc, "canonical role body for "+spec.File) {
			t.Fatalf("Codex agent does not embed the canonical role body:\n%s", doc)
		}
		if strings.Contains(doc, "model: opus") || strings.Contains(doc, "tools: Read") {
			t.Fatalf("Claude frontmatter leaked into the Codex agent:\n%s", doc)
		}
	}
}

func TestInstallTargetAllAddsOpenCodeWithoutChangingLegacyTopology(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override does not steer os.UserHomeDir on Windows")
	}
	src := fakeSource(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := Install(Options{Targets: []string{"all"}, From: src}); err != nil {
		t.Fatal(err)
	}
	shared := filepath.Join(home, ".agents")
	claude := filepath.Join(home, ".claude")
	if isSymlink(t, filepath.Join(shared, "skills", "machinery")) {
		t.Fatal("the shared skill must remain the real canonical copy")
	}
	if !isSymlink(t, filepath.Join(claude, "skills", "machinery")) {
		t.Fatal("the all-target install must preserve the Claude symlink topology")
	}
	for _, spec := range roleSpecs {
		path := filepath.Join(home, ".config", "opencode", "agents", spec.Name+".md")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("OpenCode agent missing: %v", err)
		}
		doc := string(raw)
		if !strings.Contains(doc, "mode: subagent") || !strings.Contains(doc, "canonical role body for "+spec.File) {
			t.Fatalf("OpenCode agent does not wrap the canonical role body:\n%s", doc)
		}
		if strings.Contains(doc, "model: opus") {
			t.Fatalf("Claude model pin leaked into the OpenCode agent:\n%s", doc)
		}
	}
	for _, command := range openCodeCommands {
		if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "commands", command)); err != nil {
			t.Errorf("OpenCode command %s missing: %v", command, err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "plugins", "machinery.js")); err != nil {
		t.Fatalf("OpenCode governance adapter missing: %v", err)
	}
}

func TestInstallTargetsRejectInvalidAndAmbiguousOptions(t *testing.T) {
	src := fakeSource(t)
	if err := Install(Options{Targets: []string{"cursor"}, From: src}); err == nil || !strings.Contains(err.Error(), "unknown install target") {
		t.Fatalf("unknown target error = %v", err)
	}
	if err := Install(Options{Targets: []string{"codex"}, Homes: []string{t.TempDir()}, From: src}); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("target/home conflict error = %v", err)
	}
}

func TestTargetArtifactsMatchInstalledTopology(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override does not steer os.UserHomeDir on Windows")
	}
	src := fakeSource(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := Install(Options{Targets: []string{"all"}, From: src}); err != nil {
		t.Fatal(err)
	}
	artifacts, err := TargetArtifacts([]string{"all"})
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 15 {
		t.Fatalf("all-target artifact count = %d, want 15", len(artifacts))
	}
	for _, artifact := range artifacts {
		if _, err := os.Stat(artifact.Path); err != nil {
			t.Errorf("doctor artifact missing after install: [%s] %s at %s: %v", artifact.Target, artifact.Label, artifact.Path, err)
		}
	}
}

func TestValidateArtifactRejectsWrongTypeTargetAndIdentity(t *testing.T) {
	root := t.TempDir()
	wrongType := filepath.Join(root, "skill")
	write(t, wrongType, "not a directory")
	if err := ValidateArtifact(Artifact{Target: "shared", Label: "machinery skill", Path: wrongType}); err == nil {
		t.Fatal("skill regular file was accepted")
	}
	badRole := filepath.Join(root, "machinery-fsm-author.toml")
	write(t, badRole, `name = "someone-else"`)
	if err := ValidateArtifact(Artifact{Target: "codex", Label: "agent", Path: badRole}); err == nil {
		t.Fatal("wrong rendered role identity was accepted")
	}
	if runtime.GOOS != "windows" {
		home := t.TempDir()
		t.Setenv("HOME", home)
		wrongTarget := filepath.Join(root, "outside")
		if err := os.Symlink(wrongTarget, filepath.Join(root, "machinery")); err != nil {
			t.Fatal(err)
		}
		if err := ValidateArtifact(Artifact{Target: "claude", Label: "machinery skill", Path: filepath.Join(root, "machinery")}); err == nil {
			t.Fatal("wrong Claude symlink referent was accepted")
		}
	}
}

func TestValidateArtifactBindsRecordedContentAndRejectsTruncatedTokens(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	if err := Install(Options{Homes: []string{home}, From: fakeSource(t), Record: true}); err != nil {
		t.Fatal(err)
	}
	role := filepath.Join(home, "agents", "machinery-fsm-author.md")
	artifact := Artifact{Target: "shared", Label: "machinery-fsm-author agent", Path: role}
	if err := ValidateArtifact(artifact); err != nil {
		t.Fatalf("receipt-bound installed role rejected: %v", err)
	}
	raw, err := os.ReadFile(role)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(role, append(raw, []byte("\n# tampered\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateArtifact(artifact); err == nil || !strings.Contains(err.Error(), "receipt-bound") {
		t.Fatalf("tampered receipt-bound role error = %v", err)
	}

	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	truncated := filepath.Join(t.TempDir(), "machinery-fsm-author.md")
	write(t, truncated, "---\nname: machinery-fsm-author\ndescription: >\n  Phase 3\n---\n")
	if err := ValidateArtifact(Artifact{Target: "shared", Label: "agent", Path: truncated}); err == nil {
		t.Fatalf("truncated token-bearing role error = %v", err)
	}
}

func TestReceiptArtifactDigestIgnoresInstallTimeButDetectsModeAndContent(t *testing.T) {
	installAt := func(timestamp time.Time) ([]receiptArtifact, string, string) {
		t.Helper()
		config := privateConfigDir(t)
		t.Setenv("MACHINERY_CONFIG_DIR", config)
		source := fakeSource(t)
		if err := filepath.Walk(source, func(path string, _ os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			return os.Chtimes(path, timestamp, timestamp)
		}); err != nil {
			t.Fatal(err)
		}
		home := t.TempDir()
		if err := Install(Options{Homes: []string{home}, From: source, Record: true}); err != nil {
			t.Fatal(err)
		}
		receipt, exists, err := loadReceipt()
		if err != nil || !exists {
			t.Fatalf("receipt exists=%v err=%v", exists, err)
		}
		return append([]receiptArtifact(nil), receipt.Artifacts...), filepath.Join(home, "agents", "machinery-fsm-author.md"), config
	}
	first, firstRole, firstConfig := installAt(time.Unix(1_600_000_000, 0))
	second, _, _ := installAt(time.Unix(1_900_000_000, 0))
	if len(first) != len(second) {
		t.Fatalf("artifact counts differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Digest != second[i].Digest {
			t.Fatalf("install-time digest %d differs: %s vs %s", i, first[i].Digest, second[i].Digest)
		}
	}
	if err := os.Chtimes(firstRole, time.Unix(2_000_000_000, 0), time.Unix(2_000_000_000, 0)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MACHINERY_CONFIG_DIR", firstConfig)
	if err := ValidateArtifact(Artifact{Target: "shared", Label: "agent", Path: firstRole}); err != nil {
		t.Fatalf("metadata-only touch invalidated artifact: %v", err)
	}
	if err := os.Chmod(firstRole, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateArtifact(Artifact{Target: "shared", Label: "agent", Path: firstRole}); err == nil {
		t.Fatal("semantic mode change was not detected")
	}
}

func TestUninstallTargetsPreservesSharedAssetsUntilCompleteRemoval(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override does not steer os.UserHomeDir on Windows")
	}
	src := fakeSource(t)
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := Install(Options{Targets: []string{"all"}, From: src}); err != nil {
		t.Fatal(err)
	}

	if err := UninstallTargets([]string{"opencode"}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "machinery", "SKILL.md")); err != nil {
		t.Fatalf("single-host removal must preserve the shared skill: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", "agents", "machinery-fsm-author.toml")); err != nil {
		t.Fatalf("OpenCode removal must preserve Codex assets: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "plugins", "machinery.js")); !os.IsNotExist(err) {
		t.Fatalf("OpenCode adapter remains after removal: %v", err)
	}

	if err := Install(Options{Targets: []string{"opencode"}, From: src}); err != nil {
		t.Fatal(err)
	}
	if err := UninstallTargets([]string{"all"}, nil); err != nil {
		t.Fatal(err)
	}
	artifacts, err := TargetArtifacts([]string{"all"})
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range artifacts {
		if _, err := os.Stat(artifact.Path); !os.IsNotExist(err) {
			t.Errorf("artifact remains after complete target removal: %s (err=%v)", artifact.Path, err)
		}
	}
}

func TestUninstall(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	src := fakeSource(t)
	root := t.TempDir()
	agents := filepath.Join(root, ".agents")
	claude := filepath.Join(root, ".claude")
	if err := Install(Options{Homes: []string{agents, claude}, From: src}); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall([]string{agents, claude}, nil); err != nil {
		t.Fatal(err)
	}
	for _, home := range []string{agents, claude} {
		if _, err := os.Lstat(filepath.Join(home, "skills", "machinery")); !os.IsNotExist(err) {
			t.Errorf("skill still present in %s after uninstall (err=%v)", home, err)
		}
		for _, d := range RoleDocs {
			if _, err := os.Lstat(filepath.Join(home, "agents", d)); !os.IsNotExist(err) {
				t.Errorf("role doc %s still present in %s after uninstall", d, home)
			}
		}
	}
	// Uninstall on an already-clean home must be a no-op, not an error.
	if err := Uninstall([]string{agents}, nil); err != nil {
		t.Errorf("second uninstall should be a no-op: %v", err)
	}
}

func TestUninstallCanonicalRemovesEntireRecordedGroup(t *testing.T) {
	for _, copyAll := range []bool{false, true} {
		t.Run(fmt.Sprintf("copy=%v", copyAll), func(t *testing.T) {
			t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
			src := fakeSource(t)
			root := t.TempDir()
			canonical := filepath.Join(root, "canonical")
			secondary := filepath.Join(root, "secondary")
			if err := Install(Options{Homes: []string{canonical, secondary}, From: src, Copy: copyAll, Record: true}); err != nil {
				t.Fatal(err)
			}
			if err := Uninstall([]string{canonical}, io.Discard); err != nil {
				t.Fatal(err)
			}
			for _, home := range []string{canonical, secondary} {
				if _, err := os.Lstat(filepath.Join(home, "skills", "machinery")); !os.IsNotExist(err) {
					t.Fatalf("recorded group artifact remains under %s: %v", home, err)
				}
			}
			receipt, exists, err := loadReceipt()
			if err != nil || exists || len(receipt.HomeInstalls) != 0 {
				t.Fatalf("receipt after group uninstall = %+v, exists=%v, err=%v", receipt, exists, err)
			}
		})
	}
}

func TestUninstallDeletionFailureRollsBackArtifactsAndReceipt(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	src := fakeSource(t)
	home := filepath.Join(t.TempDir(), "home")
	if err := Install(Options{Homes: []string{home}, From: src, Record: true}); err != nil {
		t.Fatal(err)
	}
	receiptPath, err := installationReceiptPath()
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	originalRemove := removeInstallArtifact
	calls := 0
	removeInstallArtifact = func(path string) error {
		calls++
		if calls == 2 {
			return errors.New("injected deletion failure")
		}
		return durableRemoveAll(path)
	}
	t.Cleanup(func() { removeInstallArtifact = originalRemove })
	if err := Uninstall([]string{home}, io.Discard); err == nil || !strings.Contains(err.Error(), "injected deletion failure") {
		t.Fatalf("Uninstall error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "skills", "machinery", "SKILL.md")); err != nil {
		t.Fatalf("skill was not restored: %v", err)
	}
	after, err := os.ReadFile(receiptPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("receipt changed after rollback: %q, %v", after, err)
	}
}

func TestValidateSourceRejectsIncomplete(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	empty := t.TempDir()
	if err := Install(Options{Homes: []string{filepath.Join(empty, "home")}, From: empty}); err == nil {
		t.Fatal("expected an error installing from a source with no skills/machinery")
	}
}

func TestValidateSourceMissingRoleDoc(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	part := t.TempDir()
	if err := os.MkdirAll(filepath.Join(part, "skills", "machinery"), 0o755); err != nil {
		t.Fatal(err)
	}
	// skills/machinery exists but the role docs do not.
	if err := Install(Options{Homes: []string{filepath.Join(part, "h")}, From: part}); err == nil {
		t.Fatal("expected an error when a role doc is missing from the source")
	}
}

func TestUninstallFailsOnUnwritableHome(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permission enforcement requires a non-root POSIX environment")
	}
	src := fakeSource(t)
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	if err := Install(Options{Homes: []string{home}, From: src}); err != nil {
		t.Fatal(err)
	}
	skills := filepath.Join(home, "skills")
	if err := os.Chmod(skills, 0o555); err != nil { // can't remove the child dir
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(skills, 0o755) })
	if err := Uninstall([]string{home}, nil); err == nil {
		t.Error("expected an error removing from an unwritable home")
	}
}

func TestResolveTagExplicitIsExact(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/releases/tags/v0.1.1-rc.1") {
			_, _ = w.Write([]byte(`{"tag_name":"v0.1.1-rc.1"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	old := apiBase
	apiBase = srv.URL
	defer func() { apiBase = old }()
	_, err := resolveTag("RamXX/machinery", "v0.1.1", true)
	if err == nil {
		t.Fatal("missing exact tag should fail")
	}
	got, err := resolveTag("RamXX/machinery", "v0.1.1-rc.1", true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v0.1.1-rc.1" {
		t.Errorf("resolveTag = %q, want v0.1.1-rc.1", got)
	}
}

// An implicit version (the binary's own default) whose release exists
// resolves to itself, not to the latest release.
func TestResolveTagImplicitUsesExistingRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/tags/v0.1.1"):
			_, _ = w.Write([]byte(`{"tag_name":"v0.1.1"}`))
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			_, _ = w.Write([]byte(`{"tag_name":"v9.9.9"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	old := apiBase
	apiBase = srv.URL
	defer func() { apiBase = old }()
	got, err := resolveTag("a/b", "v0.1.1", false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v0.1.1" {
		t.Errorf("resolveTag = %q, want the binary's own published release v0.1.1", got)
	}
}

// An implicit version with no published release falls back to the latest
// release: a locally built binary ahead of its tag must still install.
func TestResolveTagImplicitFallsBackWhenUnpublished(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/releases/latest") {
			_, _ = w.Write([]byte(`{"tag_name":"v9.9.9"}`))
			return
		}
		http.NotFound(w, r) // the tag probe 404s
	}))
	defer srv.Close()
	old := apiBase
	apiBase = srv.URL
	defer func() { apiBase = old }()
	got, err := resolveTag("a/b", "v0.0.9", false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v9.9.9" {
		t.Errorf("resolveTag = %q, want fallback to latest v9.9.9", got)
	}
}

// An explicit --version whose release does not exist must fail loudly, never
// be silently substituted with the latest release.
func TestInstallExplicitMissingVersionFailsLoudly(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	tarball := sourceTarball(t, "machinery")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			_, _ = w.Write([]byte(`{"tag_name":"v9.9.9"}`))
		case strings.HasSuffix(r.URL.Path, "/releases/download/v9.9.9/machinery-source.tar.gz"):
			_, _ = w.Write(tarball)
		case strings.HasSuffix(r.URL.Path, "/releases/download/v9.9.9/checksums-sha256.txt"):
			_, _ = fmt.Fprintf(w, "%x  machinery-source.tar.gz\n", sha256.Sum256(tarball))
		default:
			http.NotFound(w, r) // v0.0.9 has no release and no tarball
		}
	}))
	defer srv.Close()
	oldGH, oldAPI := githubBase, apiBase
	githubBase, apiBase = srv.URL, srv.URL
	defer func() { githubBase, apiBase = oldGH, oldAPI }()

	root := t.TempDir()
	err := Install(Options{Homes: []string{filepath.Join(root, ".a")}, Repo: "a/b", Version: "v0.0.9", VersionExplicit: true})
	if err == nil {
		t.Fatal("explicit missing version must fail, not fall back to latest")
	}
	if !strings.Contains(err.Error(), "v0.0.9") {
		t.Errorf("error should name the requested version: %v", err)
	}
}

// The same missing version arriving implicitly (the binary's own default)
// installs from the latest release instead.
func TestInstallImplicitUnpublishedVersionFallsBackToLatest(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	tarball := sourceTarball(t, "machinery")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			_, _ = w.Write([]byte(`{"tag_name":"v9.9.9"}`))
		case strings.HasSuffix(r.URL.Path, "/releases/download/v9.9.9/machinery-source.tar.gz"):
			_, _ = w.Write(tarball)
		case strings.HasSuffix(r.URL.Path, "/releases/download/v9.9.9/checksums-sha256.txt"):
			_, _ = fmt.Fprintf(w, "%x  machinery-source.tar.gz\n", sha256.Sum256(tarball))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	oldGH, oldAPI := githubBase, apiBase
	githubBase, apiBase = srv.URL, srv.URL
	defer func() { githubBase, apiBase = oldGH, oldAPI }()

	home := filepath.Join(t.TempDir(), ".a")
	if err := Install(Options{Homes: []string{home}, Repo: "a/b", Version: "v0.0.9"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, "skills", "machinery", "SKILL.md")); err != nil {
		t.Errorf("fallback install missing skill: %v", err)
	}
}

// sourceTarball builds a gzipped tar that mirrors a GitHub source archive:
// a single top-level dir holding skills/machinery + agents role docs.
func sourceTarball(t *testing.T, top string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	add := func(name, body string) {
		if err := tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	add(top+"/skills/machinery/SKILL.md", "---\nname: machinery\n---\n")
	add(top+"/skills/machinery/references/x.md", "ref\n")
	for _, d := range RoleDocs {
		add(top+"/agents/"+d, "role\n")
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestInstallFetchesFromRelease exercises the full network path (resolve latest
// tag -> download source tarball -> extract -> lay down) against a local
// httptest server, so no real GitHub calls are made.
func TestInstallFetchesFromRelease(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	const repo = "acme/machinery"
	tarball := sourceTarball(t, "machinery")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tag_name":"v9.9.9"}`))
		case strings.HasSuffix(r.URL.Path, "/releases/download/v9.9.9/machinery-source.tar.gz"):
			_, _ = w.Write(tarball)
		case strings.HasSuffix(r.URL.Path, "/releases/download/v9.9.9/checksums-sha256.txt"):
			_, _ = fmt.Fprintf(w, "%x  machinery-source.tar.gz\n", sha256.Sum256(tarball))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	oldGH, oldAPI := githubBase, apiBase
	githubBase, apiBase = srv.URL, srv.URL
	defer func() { githubBase, apiBase = oldGH, oldAPI }()

	root := t.TempDir()
	agents := filepath.Join(root, ".agents")
	claude := filepath.Join(root, ".claude")
	// Version "" with a non-release value forces the latest-release lookup.
	if err := Install(Options{Homes: []string{agents, claude}, Repo: repo, Version: "latest"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(agents, "skills", "machinery", "SKILL.md")); err != nil {
		t.Errorf("fetched skill missing SKILL.md: %v", err)
	}
	if fi, err := os.Lstat(filepath.Join(claude, "skills", "machinery")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("secondary home should be a symlink after a fetched install (err=%v)", err)
	}
}

func TestInstallDefaultHomes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override does not steer os.UserHomeDir on Windows")
	}
	src := fakeSource(t)
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := Install(Options{From: src}); err != nil { // no Homes -> DefaultHomes under $HOME
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "machinery", "SKILL.md")); err != nil {
		t.Errorf("default canonical home (~/.agents) missing skill: %v", err)
	}
	if fi, err := os.Lstat(filepath.Join(home, ".claude", "skills", "machinery")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("default secondary home (~/.claude) should be a symlink (err=%v)", err)
	}
	if err := Uninstall(nil, nil); err != nil { // default homes
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(home, ".agents", "skills", "machinery")); !os.IsNotExist(err) {
		t.Errorf("default uninstall left the skill behind")
	}
}

func TestAbsHomesRejectsDuplicatesOverlapsAndSymlinkAliases(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(real, alias); err != nil {
		t.Fatal(err)
	}
	for name, homes := range map[string][]string{
		"duplicate": {real, real},
		"nested":    {real, filepath.Join(real, "child")},
		"alias":     {real, alias},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := absHomes(homes); err == nil || !strings.Contains(err.Error(), "overlap") {
				t.Fatalf("absHomes(%v) = %v, want overlap error", homes, err)
			}
		})
	}
}

func TestAbsHomesRejectsNativeCaseAliasesWithMissingTails(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("case-folded path identity is a Darwin/Windows contract")
	}
	root := t.TempDir()
	upper := filepath.Join(root, "Missing", "Home")
	lower := filepath.Join(root, "missing", "home")
	if _, err := absHomes([]string{upper, lower}); err == nil || !strings.Contains(err.Error(), "overlap") {
		t.Fatalf("case aliases accepted: %v", err)
	}
}

func TestDefaultHomesFailsWithoutUserHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows resolves the profile through platform APIs")
	}
	t.Setenv("HOME", "")
	if _, err := DefaultHomes(); err == nil {
		t.Fatal("missing HOME must not produce relative default install paths")
	}
}

func TestNativeTargetPlanningFailsWithoutUserHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows resolves the profile through platform APIs")
	}
	t.Setenv("HOME", "")
	if _, err := TargetArtifacts([]string{"codex"}); err == nil {
		t.Fatal("target artifact planning accepted an empty user home")
	}
	if err := UninstallTargets([]string{"codex"}, io.Discard); err == nil {
		t.Fatal("target uninstall accepted an empty user home")
	}
	if _, err := updatePlan(UpdateOptions{Homes: []string{t.TempDir()}}); err == nil {
		t.Fatal("update planning silently skipped native plugin discovery with an empty user home")
	}
}

func TestFetchSourceRejectsChecksumMismatch(t *testing.T) {
	tarball := sourceTarball(t, "machinery")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			_, _ = w.Write([]byte(`{"tag_name":"v1.2.3"}`))
		case strings.HasSuffix(r.URL.Path, "/machinery-source.tar.gz"):
			_, _ = w.Write(tarball)
		case strings.HasSuffix(r.URL.Path, "/checksums-sha256.txt"):
			_, _ = fmt.Fprintf(w, "%s  machinery-source.tar.gz\n", strings.Repeat("0", 64))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	oldGH, oldAPI := githubBase, apiBase
	githubBase, apiBase = srv.URL, srv.URL
	defer func() { githubBase, apiBase = oldGH, oldAPI }()
	if _, _, err := fetchSource("a/b", "latest", false, nil); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("fetchSource error = %v, want checksum mismatch", err)
	}
}

func TestResolveTagDistinguishesNetworkFailureFromNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/releases/tags/v1.2.3") {
			http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	old := apiBase
	apiBase = srv.URL
	defer func() { apiBase = old }()
	_, err := resolveTag("a/b", "v1.2.3", true)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("resolveTag error = %v, want upstream status", err)
	}
}

func TestInstallFetchErrorWhenTarballMissing(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/releases/latest") {
			_, _ = w.Write([]byte(`{"tag_name":"v9.9.9"}`))
			return
		}
		http.NotFound(w, r) // tarball 404s
	}))
	defer srv.Close()
	oldGH, oldAPI := githubBase, apiBase
	githubBase, apiBase = srv.URL, srv.URL
	defer func() { githubBase, apiBase = oldGH, oldAPI }()

	root := t.TempDir()
	if err := Install(Options{Homes: []string{filepath.Join(root, ".a")}, Repo: "a/b", Version: "latest"}); err == nil {
		t.Fatal("expected an error when the source tarball is missing")
	}
}

func TestResolveTagRejectsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":""}`))
	}))
	defer srv.Close()
	old := apiBase
	apiBase = srv.URL
	defer func() { apiBase = old }()
	if _, err := resolveTag("a/b", "latest", false); err == nil {
		t.Fatal("expected an error for an empty tag_name")
	}
}

func TestResolveTagRejectsBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer srv.Close()
	old := apiBase
	apiBase = srv.URL
	defer func() { apiBase = old }()
	if _, err := resolveTag("a/b", "", false); err == nil {
		t.Fatal("expected an error for malformed release JSON")
	}
}

func TestInstallFailsOnUnwritableHome(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permission enforcement requires a non-root POSIX environment")
	}
	src := fakeSource(t)
	ro := t.TempDir()
	if err := os.Chmod(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ro, 0o755) })
	if err := Install(Options{Homes: []string{filepath.Join(ro, "home")}, From: src}); err == nil {
		t.Fatal("expected an error installing into an unwritable home")
	}
}

func TestErrorPaths(t *testing.T) {
	if err := copyFile(filepath.Join(t.TempDir(), "nope"), filepath.Join(t.TempDir(), "dst")); err == nil {
		t.Error("copyFile should fail on a missing source")
	}
	srcF := filepath.Join(t.TempDir(), "s")
	write(t, srcF, "x")
	blocker := filepath.Join(t.TempDir(), "blk")
	write(t, blocker, "x")
	if err := copyFile(srcF, filepath.Join(blocker, "child")); err == nil {
		t.Error("copyFile should fail when the destination parent is a regular file")
	}
	nonEmptyDir := t.TempDir()
	write(t, filepath.Join(nonEmptyDir, "child"), "x")
	if err := copyFile(srcF, nonEmptyDir); err == nil {
		t.Error("copyFile should fail when the destination is a non-empty directory")
	}
	if err := copyTree(filepath.Join(t.TempDir(), "nope"), t.TempDir()); err == nil {
		t.Error("copyTree should fail on a missing source")
	}
	bad := filepath.Join(t.TempDir(), "bad.tar.gz")
	write(t, bad, "not gzip")
	if err := extractTarGz(bad, t.TempDir()); err == nil {
		t.Error("extractTarGz should fail on non-gzip input")
	}
	if err := extractTarGz(filepath.Join(t.TempDir(), "missing.tgz"), t.TempDir()); err == nil {
		t.Error("extractTarGz should fail on a missing archive")
	}
	if _, err := singleChildDir(t.TempDir()); err == nil {
		t.Error("singleChildDir should fail with no subdirs")
	}
	if err := download("http://127.0.0.1:0/nope", filepath.Join(t.TempDir(), "x"), releaseAPIDownload); err == nil {
		t.Error("download should fail on an unreachable host")
	}
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("data")) }))
	defer okSrv.Close()
	if err := download(okSrv.URL, filepath.Join(blocker, "child"), releaseAPIDownload); err == nil {
		t.Error("download should fail when the destination cannot be created")
	}
	if got, err := absHomes([]string{"", "  "}); err != nil || len(got) != 0 {
		t.Errorf("absHomes should skip blank entries, got %v (err %v)", got, err)
	}
}

func TestInstallFailsOnUnwritableSecondary(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permission enforcement requires a non-root POSIX environment")
	}
	src := fakeSource(t)
	root := t.TempDir()
	canon := filepath.Join(root, ".agents") // writable canonical home
	write(t, filepath.Join(canon, "skills", "machinery", "SKILL.md"), "previous\n")
	roParent := t.TempDir()
	if err := os.Chmod(roParent, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(roParent, 0o755) })
	secondary := filepath.Join(roParent, ".claude") // parent is unwritable
	if err := Install(Options{Homes: []string{canon, secondary}, From: src}); err == nil {
		t.Fatal("expected an error linking into an unwritable secondary home")
	}
	raw, err := os.ReadFile(filepath.Join(canon, "skills", "machinery", "SKILL.md"))
	if err != nil || string(raw) != "previous\n" {
		t.Fatalf("canonical install changed before secondary preflight failed: %q, %v", raw, err)
	}
}

func TestInstallRollsBackEarlierHomeWhenLaterHomeFails(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	src := fakeSource(t)
	root := t.TempDir()
	first, second := filepath.Join(root, "first"), filepath.Join(root, "second")
	for _, home := range []string{first, second} {
		write(t, filepath.Join(home, "skills", "machinery", "SKILL.md"), "old "+filepath.Base(home)+"\n")
		for _, doc := range RoleDocs {
			write(t, filepath.Join(home, "agents", doc), "old "+filepath.Base(home)+" "+doc+"\n")
		}
	}
	if err := saveReceipt(installReceipt{
		SchemaVersion: receiptSchema,
		HomeInstalls:  []homeInstall{{Homes: []string{first, second}, Copy: true}},
	}); err != nil {
		t.Fatal(err)
	}
	receiptPath, err := installationReceiptPath()
	if err != nil {
		t.Fatal(err)
	}
	receiptBefore, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}

	err = Install(Options{
		Homes:  []string{first, second},
		From:   src,
		Copy:   true,
		Record: true,
		beforeCommit: func(home string) error {
			if home == second {
				return errors.New("injected later-home failure")
			}
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "injected later-home failure") {
		t.Fatalf("Install error = %v", err)
	}
	for _, home := range []string{first, second} {
		got, readErr := os.ReadFile(filepath.Join(home, "skills", "machinery", "SKILL.md"))
		want := "old " + filepath.Base(home) + "\n"
		if readErr != nil || string(got) != want {
			t.Errorf("%s was not restored: got %q, err %v, want %q", home, got, readErr, want)
		}
	}
	receiptAfter, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(receiptAfter, receiptBefore) {
		t.Fatalf("receipt changed after rollback:\nbefore=%s\nafter=%s", receiptBefore, receiptAfter)
	}
}

func TestInstallFromRejectsSourceMutationAndRollsBack(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	src := fakeSource(t)
	home := filepath.Join(t.TempDir(), "home")
	role := filepath.Join(src, agentsRel, RoleDocs[0])
	err := Install(Options{
		Homes: []string{home},
		From:  src,
		beforeCommit: func(string) error {
			write(t, role, "---\nname: role\ndescription: role\n---\n\nmutated after snapshot\n")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "changed after the immutable snapshot") {
		t.Fatalf("source mutation error = %v", err)
	}
	if _, statErr := os.Lstat(filepath.Join(home, "skills", "machinery")); !os.IsNotExist(statErr) {
		t.Fatalf("source mutation was committed instead of rolled back: %v", statErr)
	}
}

func TestInstallFromRejectsConcurrentSourceReplacement(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	src := fakeSource(t)
	home := filepath.Join(t.TempDir(), "home")
	roleRel := filepath.Join(agentsRel, RoleDocs[0])
	role := filepath.Join(src, roleRel)
	opened := make(chan struct{})
	mutated := make(chan error, 1)
	oldHook := sourceSnapshotAfterOpen
	t.Cleanup(func() { sourceSnapshotAfterOpen = oldHook })
	sourceSnapshotAfterOpen = func(rel string) {
		if rel != roleRel {
			return
		}
		sourceSnapshotAfterOpen = func(string) {}
		close(opened)
		if err := <-mutated; err != nil {
			t.Errorf("concurrent mutation failed: %v", err)
		}
	}
	go func() {
		<-opened
		if err := os.Rename(role, role+".original"); err != nil {
			mutated <- err
			return
		}
		mutated <- os.WriteFile(role, []byte("---\nname: role\ndescription: role\n---\n\nconcurrent replacement\n"), 0o644)
	}()
	err := Install(Options{Homes: []string{home}, From: src})
	if err == nil || !strings.Contains(err.Error(), "changed while being snapshotted") {
		t.Fatalf("concurrent source replacement error = %v", err)
	}
	if _, statErr := os.Lstat(filepath.Join(home, "skills", "machinery")); !os.IsNotExist(statErr) {
		t.Fatalf("concurrent source replacement was committed: %v", statErr)
	}
}

func TestInstallSourceSnapshotRejectsBehindCursorNestedMutation(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, string) func() error
	}{
		{
			name: "late nested addition",
			prepare: func(_ *testing.T, first string) func() error {
				return func() error { return os.WriteFile(filepath.Join(first, "added.md"), []byte("late\n"), 0o644) }
			},
		},
		{
			name: "late nested removal",
			prepare: func(_ *testing.T, first string) func() error {
				return func() error { return os.Remove(filepath.Join(first, "kept.md")) }
			},
		},
		{
			name: "late nested ABA",
			prepare: func(t *testing.T, first string) func() error {
				info, err := os.Stat(first)
				if err != nil {
					t.Fatal(err)
				}
				if installFileChangeID(info) == "" {
					t.Skip("platform does not expose a directory change identity")
				}
				transient := filepath.Join(first, "transient.md")
				return func() error {
					if err := os.WriteFile(transient, []byte("transient\n"), 0o644); err != nil {
						return err
					}
					if err := os.Remove(transient); err != nil {
						return err
					}
					return os.Chtimes(first, info.ModTime(), info.ModTime())
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := fakeSource(t)
			firstRel := filepath.Join(skillRel, "aa-first")
			laterRel := filepath.Join(skillRel, "zz-later")
			first := filepath.Join(src, firstRel)
			write(t, filepath.Join(first, "kept.md"), "kept\n")
			write(t, filepath.Join(src, laterRel, "kept.md"), "later\n")
			snapshot, err := acquireInstallSourceSnapshot(src, nil)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := snapshot.cleanup(); err != nil {
					t.Errorf("cleanup source snapshot: %v", err)
				}
			})
			mutate := tc.prepare(t, first)
			start := make(chan struct{})
			done := make(chan error, 1)
			oldHook := sourceSnapshotAfterVerifyDirectory
			t.Cleanup(func() { sourceSnapshotAfterVerifyDirectory = oldHook })
			triggered := false
			laterScanned := false
			sourceSnapshotAfterVerifyDirectory = func(rel string) {
				if triggered {
					if rel == laterRel {
						laterScanned = true
					}
					return
				}
				if rel != firstRel {
					return
				}
				triggered = true
				close(start)
				if err := <-done; err != nil {
					t.Errorf("concurrent source mutation: %v", err)
				}
			}
			go func() {
				<-start
				done <- mutate()
			}()
			if err := snapshot.verifyUnchanged(); err == nil || !strings.Contains(err.Error(), "install source") {
				t.Fatalf("behind-cursor source mutation error = %v", err)
			}
			if !laterScanned {
				t.Fatal("source mutation did not occur while a later nested directory remained to scan")
			}
		})
	}
}

func TestInstallFromNestedMutationCannotCrossCommitBoundary(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	src := fakeSource(t)
	firstRel := filepath.Join(skillRel, "aa-first")
	laterRel := filepath.Join(skillRel, "zz-later")
	first := filepath.Join(src, firstRel)
	write(t, filepath.Join(first, "kept.md"), "kept\n")
	write(t, filepath.Join(src, laterRel, "kept.md"), "later\n")
	home := filepath.Join(t.TempDir(), "home")
	oldSkill := filepath.Join(home, "skills", "machinery", "SKILL.md")
	write(t, oldSkill, "old installed skill\n")
	oldHook := sourceSnapshotAfterVerifyDirectory
	t.Cleanup(func() { sourceSnapshotAfterVerifyDirectory = oldHook })
	mutated := false
	sourceSnapshotAfterVerifyDirectory = func(rel string) {
		if rel != firstRel || mutated {
			return
		}
		mutated = true
		write(t, filepath.Join(first, "late.md"), "late source mutation\n")
	}
	err := Install(Options{Homes: []string{home}, From: src, Copy: true})
	if err == nil || !strings.Contains(err.Error(), "install source") {
		t.Fatalf("late nested source mutation error = %v", err)
	}
	if !mutated {
		t.Fatal("late nested source mutation hook did not run")
	}
	if got, err := os.ReadFile(oldSkill); err != nil || string(got) != "old installed skill\n" {
		t.Fatalf("source mutation crossed commit boundary or rollback lost prior target: %q, %v", got, err)
	}
}

func TestInstallFromABASwapNeverReachesTargetRenderers(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	t.Setenv("HOME", home)
	src := fakeSource(t)
	role := filepath.Join(src, agentsRel, RoleDocs[0])
	original, err := os.ReadFile(role)
	if err != nil {
		t.Fatal(err)
	}
	saved := role + ".saved"
	mutated := false
	err = Install(Options{
		Targets: []string{"codex"},
		From:    src,
		beforeCommit: func(point string) error {
			if point != string(TargetCodex) || mutated {
				return nil
			}
			mutated = true
			if err := os.Rename(role, saved); err != nil {
				return err
			}
			if err := os.WriteFile(role, []byte("---\nname: role\ndescription: role\n---\n\nMALICIOUS ABA BODY\n"), 0o644); err != nil {
				return err
			}
			if err := os.Remove(role); err != nil {
				return err
			}
			return os.Rename(saved, role)
		},
	})
	if err == nil || !strings.Contains(err.Error(), "changed after the immutable snapshot") {
		t.Fatalf("source ABA error = %v", err)
	}
	if !mutated {
		t.Fatal("ABA hook did not run")
	}
	if _, statErr := os.Lstat(filepath.Join(home, ".codex", "agents", "machinery-fsm-author.toml")); !os.IsNotExist(statErr) {
		t.Fatalf("source ABA reached target commit: %v", statErr)
	}
	current, err := os.ReadFile(role)
	if err != nil || !bytes.Equal(current, original) {
		t.Fatalf("ABA fixture did not restore source: %v", err)
	}
}

func TestInstallFromRejectsSymlinkedSourceMembers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink fixture requires unprivileged symlink support")
	}
	for _, tc := range []struct {
		name string
		rel  string
	}{
		{name: "skill member", rel: filepath.Join(skillRel, "references", "x.md")},
		{name: "role", rel: filepath.Join(agentsRel, RoleDocs[0])},
		{name: "OpenCode adapter", rel: filepath.Join("adapters", "opencode", "plugins", "machinery.js")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := fakeSource(t)
			target := filepath.Join(t.TempDir(), "outside")
			write(t, target, "outside\n")
			member := filepath.Join(src, tc.rel)
			if err := os.Remove(member); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, member); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			opts := Options{Homes: []string{filepath.Join(t.TempDir(), "home")}, From: src}
			if tc.name == "OpenCode adapter" {
				opts.Homes = nil
				opts.Targets = []string{"opencode"}
				t.Setenv("HOME", t.TempDir())
			}
			if err := Install(opts); err == nil || !strings.Contains(err.Error(), "symlink") {
				t.Fatalf("symlinked source member accepted: %v", err)
			}
		})
	}
}

func TestArtifactTransactionRollbackPreservesSymlinkLeaf(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink fixture is POSIX-specific")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	root := t.TempDir()
	external := filepath.Join(root, "external.md")
	write(t, external, "external remains untouched\n")
	target := filepath.Join(root, "home", "agents", RoleDocs[0])
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, target); err != nil {
		t.Fatal(err)
	}
	tx, err := beginArtifactTransaction([]string{target})
	if err != nil {
		t.Fatal(err)
	}
	transactionReplaceForTest(t, target, "replacement\n")
	if err := tx.rollback(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(target)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("rollback did not restore symlink leaf: info=%v err=%v", info, err)
	}
	link, err := os.Readlink(target)
	if err != nil || link != external {
		t.Fatalf("restored link = %q, err %v; want %q", link, err, external)
	}
	if got, err := os.ReadFile(external); err != nil || string(got) != "external remains untouched\n" {
		t.Fatalf("external target changed: %q, err %v", got, err)
	}
}

func TestExtractTarGzRejectsTraversalBeforeCreatingPartialTree(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	entries := []struct {
		name, body string
		dir        bool
	}{
		{name: "pkg/", dir: true},
		{name: "pkg/a.txt", body: "hello"},
		{name: "../escape.txt", body: "nope"}, // must be rejected before extraction, never clamped
	}
	for _, e := range entries {
		if e.dir {
			if err := tw.WriteHeader(&tar.Header{Name: e.name, Typeflag: tar.TypeDir, Mode: 0o755}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := tw.WriteHeader(&tar.Header{Name: e.name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(e.body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	gz.Close()

	root := t.TempDir()
	archive := filepath.Join(root, "src.tar.gz")
	write(t, archive, buf.String())
	dest := filepath.Join(root, "out")
	if err := extractTarGz(archive, dest); err == nil || !strings.Contains(err.Error(), "portable canonical") {
		t.Fatalf("traversal archive error = %v", err)
	}
	if _, err := os.Lstat(dest); !os.IsNotExist(err) {
		t.Fatalf("invalid archive created a partial tree: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "escape.txt")); !os.IsNotExist(err) {
		t.Errorf("traversal entry escaped dest: %v", err)
	}
}

func TestExtractTarGzRejectsUnsupportedAndAliasedMembersBeforeWrites(t *testing.T) {
	tests := []struct {
		name    string
		entries []*tar.Header
		marker  string
	}{
		{"symlink", []*tar.Header{{Name: "pkg/ok", Typeflag: tar.TypeReg, Mode: 0o644}, {Name: "pkg/link", Typeflag: tar.TypeSymlink, Linkname: "ok"}}, "unsupported type"},
		{"hardlink", []*tar.Header{{Name: "pkg/ok", Typeflag: tar.TypeReg, Mode: 0o644}, {Name: "pkg/link", Typeflag: tar.TypeLink, Linkname: "pkg/ok"}}, "unsupported type"},
		{"character device", []*tar.Header{{Name: "pkg/device", Typeflag: tar.TypeChar, Mode: 0o600}}, "unsupported type"},
		{"block device", []*tar.Header{{Name: "pkg/device", Typeflag: tar.TypeBlock, Mode: 0o600}}, "unsupported type"},
		{"fifo", []*tar.Header{{Name: "pkg/fifo", Typeflag: tar.TypeFifo, Mode: 0o600}}, "unsupported type"},
		{"exact duplicate", []*tar.Header{{Name: "pkg/a", Typeflag: tar.TypeReg}, {Name: "pkg/a", Typeflag: tar.TypeReg}}, "repeats canonical"},
		{"case alias", []*tar.Header{{Name: "pkg/Agent.md", Typeflag: tar.TypeReg}, {Name: "pkg/agent.md", Typeflag: tar.TypeReg}}, "aliases prior"},
		{"unicode normalization alias", []*tar.Header{{Name: "pkg/café", Typeflag: tar.TypeReg}, {Name: "pkg/cafe\u0301", Typeflag: tar.TypeReg}}, "portable ASCII"},
		{"backslash", []*tar.Header{{Name: `pkg\agent.md`, Typeflag: tar.TypeReg}}, "portable relative"},
		{"windows reserved con", []*tar.Header{{Name: "pkg/CON", Typeflag: tar.TypeReg}}, "Windows-reserved"},
		{"windows reserved suffix", []*tar.Header{{Name: "pkg/aux.txt", Typeflag: tar.TypeReg}}, "Windows-reserved"},
		{"trailing dot", []*tar.Header{{Name: "pkg/name.", Typeflag: tar.TypeReg}}, "portable filename"},
		{"trailing space", []*tar.Header{{Name: "pkg/name ", Typeflag: tar.TypeReg}}, "portable filename"},
		{"control", []*tar.Header{{Name: "pkg/name\x01", Typeflag: tar.TypeReg}}, "portable ASCII"},
		{"file parent", []*tar.Header{{Name: "pkg", Typeflag: tar.TypeReg}, {Name: "pkg/a", Typeflag: tar.TypeReg}}, "nested beneath regular"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var archive bytes.Buffer
			gz := gzip.NewWriter(&archive)
			writer := tar.NewWriter(gz)
			for _, header := range tc.entries {
				copy := *header
				if copy.Typeflag == tar.TypeReg {
					copy.Size = 0
				}
				if err := writer.WriteHeader(&copy); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err := gz.Close(); err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			archivePath := filepath.Join(root, "fixture.tar.gz")
			if err := os.WriteFile(archivePath, archive.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			destination := filepath.Join(root, "out")
			var diagnostic string
			for iteration := 0; iteration < 20; iteration++ {
				err := extractTarGz(archivePath, destination)
				if err == nil || !strings.Contains(err.Error(), tc.marker) {
					t.Fatalf("iteration %d error = %v, want %q", iteration, err, tc.marker)
				}
				if diagnostic == "" {
					diagnostic = err.Error()
				} else if err.Error() != diagnostic {
					t.Fatalf("nondeterministic diagnostic: %q vs %q", diagnostic, err)
				}
				if _, err := os.Lstat(destination); !os.IsNotExist(err) {
					t.Fatalf("invalid archive created partial destination: %v", err)
				}
			}
		})
	}
}

func TestSingleChildDirRequiresExactSoleMachineryRoot(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, string)
	}{
		{"empty", func(*testing.T, string) {}},
		{"alternate root", func(t *testing.T, root string) {
			t.Helper()
			if err := os.Mkdir(filepath.Join(root, "machinery-v1"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"root file", func(t *testing.T, root string) {
			t.Helper()
			write(t, filepath.Join(root, "machinery"), "not a directory")
		}},
		{"injected sibling file", func(t *testing.T, root string) {
			t.Helper()
			if err := os.Mkdir(filepath.Join(root, "machinery"), 0o755); err != nil {
				t.Fatal(err)
			}
			write(t, filepath.Join(root, "payload"), "injected")
		}},
		{"injected sibling directory", func(t *testing.T, root string) {
			t.Helper()
			for _, name := range []string{"machinery", "other"} {
				if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
					t.Fatal(err)
				}
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			tc.setup(t, root)
			if _, err := singleChildDir(root); err == nil || !strings.Contains(err.Error(), "machinery/") {
				t.Fatalf("singleChildDir error = %v, want exact machinery root diagnostic", err)
			}
		})
	}
	ok := t.TempDir()
	if err := os.Mkdir(filepath.Join(ok, "machinery"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := singleChildDir(ok); err != nil || got != filepath.Join(ok, "machinery") {
		t.Fatalf("singleChildDir = %q, %v", got, err)
	}
}

// pluginInstalled must survive a home path containing glob metacharacters:
// filepath.Glob treats "[...]" in $HOME as a character class and silently
// reports no plugin, which re-enables the duplicate-skill fallback.
func TestPluginInstalledSurvivesGlobMetacharHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "[claude] home")
	seedCachedMachineryPlugin(t, home, "machinery-marketplace")
	if installed, err := pluginInstalled(home); err != nil || !installed {
		t.Fatal("plugin cache present but not detected under a metachar home path")
	}
	empty := filepath.Join(t.TempDir(), "[claude] empty")
	if err := os.MkdirAll(filepath.Join(empty, "plugins", "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	if installed, err := pluginInstalled(empty); err != nil || installed {
		t.Fatal("empty cache reported as installed")
	}
	if installed, err := pluginInstalled(filepath.Join(t.TempDir(), "nonexistent")); err != nil || installed {
		t.Fatal("missing home reported as installed")
	}
}
