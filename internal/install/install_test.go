package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeSource builds a minimal source tree containing the shared skill, the
// canonical role bodies, and the OpenCode adapter assets.
func fakeSource(t *testing.T) string {
	t.Helper()
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

func TestUninstallTargetsPreservesSharedAssetsUntilCompleteRemoval(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override does not steer os.UserHomeDir on Windows")
	}
	src := fakeSource(t)
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

func TestValidateSourceRejectsIncomplete(t *testing.T) {
	empty := t.TempDir()
	if err := Install(Options{Homes: []string{filepath.Join(empty, "home")}, From: empty}); err == nil {
		t.Fatal("expected an error installing from a source with no skills/machinery")
	}
}

func TestValidateSourceMissingRoleDoc(t *testing.T) {
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

func TestResolveTagExplicitIsOffline(t *testing.T) {
	// A well-formed release tag returns as-is with no network call.
	got, err := resolveTag("RamXX/machinery", "v0.1.1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "v0.1.1" {
		t.Errorf("resolveTag = %q, want v0.1.1", got)
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
	const repo = "acme/machinery"
	tarball := sourceTarball(t, "machinery-9.9.9")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tag_name":"v9.9.9"}`))
		case strings.HasSuffix(r.URL.Path, "/archive/refs/tags/v9.9.9.tar.gz"):
			_, _ = w.Write(tarball)
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

func TestInstallFetchErrorWhenTarballMissing(t *testing.T) {
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
	if _, err := resolveTag("a/b", "latest"); err == nil {
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
	if _, err := resolveTag("a/b", ""); err == nil {
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
	if err := download("http://127.0.0.1:0/nope", filepath.Join(t.TempDir(), "x")); err == nil {
		t.Error("download should fail on an unreachable host")
	}
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("data")) }))
	defer okSrv.Close()
	if err := download(okSrv.URL, filepath.Join(blocker, "child")); err == nil {
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
	roParent := t.TempDir()
	if err := os.Chmod(roParent, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(roParent, 0o755) })
	secondary := filepath.Join(roParent, ".claude") // parent is unwritable
	if err := Install(Options{Homes: []string{canon, secondary}, From: src}); err == nil {
		t.Fatal("expected an error linking into an unwritable secondary home")
	}
}

func TestExtractTarGzClampsTraversal(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	entries := []struct {
		name, body string
		dir        bool
	}{
		{name: "pkg/", dir: true},
		{name: "pkg/a.txt", body: "hello"},
		{name: "../escape.txt", body: "nope"}, // must be clamped inside dest, never escape
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
	if err := extractTarGz(archive, dest); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dest, "pkg", "a.txt")); err != nil || string(b) != "hello" {
		t.Errorf("pkg/a.txt = %q, %v; want hello", b, err)
	}
	// The traversal entry must not have written above dest.
	if _, err := os.Stat(filepath.Join(root, "escape.txt")); !os.IsNotExist(err) {
		t.Errorf("traversal entry escaped dest: %v", err)
	}
}
