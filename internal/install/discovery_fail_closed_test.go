package install

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPluginInstalledRejectsCorruptAndAmbiguousCaches(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, string)
	}{
		{"cache root is symlink", func(t *testing.T, home string) {
			target := filepath.Join(t.TempDir(), "cache")
			if err := os.MkdirAll(target, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(home, "plugins"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, filepath.Join(home, "plugins", "cache")); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
		}},
		{"marketplace is symlink", func(t *testing.T, home string) {
			target := filepath.Join(t.TempDir(), "market")
			if err := os.MkdirAll(target, 0o755); err != nil {
				t.Fatal(err)
			}
			cache := filepath.Join(home, "plugins", "cache")
			if err := os.MkdirAll(cache, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, filepath.Join(cache, "market")); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
		}},
		{"machinery object is file", func(t *testing.T, home string) {
			write(t, filepath.Join(home, "plugins", "cache", "market", "machinery"), "not a plugin")
		}},
		{"machinery object is symlink", func(t *testing.T, home string) {
			target := seedCachedMachineryPlugin(t, filepath.Join(t.TempDir(), "target"), "market")
			parent := filepath.Join(home, "plugins", "cache", "market")
			if err := os.MkdirAll(parent, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, filepath.Join(parent, "machinery")); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
		}},
		{"partial plugin directory", func(t *testing.T, home string) {
			if err := os.MkdirAll(filepath.Join(home, "plugins", "cache", "market", "machinery"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"wrong manifest identity", func(t *testing.T, home string) {
			root := seedCachedMachineryPlugin(t, home, "market")
			write(t, filepath.Join(root, ".claude-plugin", "plugin.json"), `{"name":"other","version":"0.6.2"}`)
		}},
		{"stale valid manifest", func(t *testing.T, home string) {
			root := seedCachedMachineryPlugin(t, home, "market")
			write(t, filepath.Join(root, ".claude-plugin", "plugin.json"), `{"name":"machinery","version":"0.1.0"}`)
		}},
		{"truncated skill", func(t *testing.T, home string) {
			root := seedCachedMachineryPlugin(t, home, "market")
			write(t, filepath.Join(root, "skills", "machinery", "SKILL.md"), "---\nname: machinery\n---\n")
		}},
		{"malformed hook topology", func(t *testing.T, home string) {
			root := seedCachedMachineryPlugin(t, home, "market")
			write(t, filepath.Join(root, "hooks", "hooks.json"), `{"description":"present","hooks":[]}`)
		}},
		{"ambiguous valid caches", func(t *testing.T, home string) {
			seedCachedMachineryPlugin(t, home, "first")
			seedCachedMachineryPlugin(t, home, "second")
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			tc.setup(t, home)
			installed, err := pluginInstalled(home)
			if err == nil || installed {
				t.Fatalf("pluginInstalled(%s) = %v, %v; corrupt cache must block", home, installed, err)
			}
		})
	}
}

func TestCachedHookManifestRejectsMalformedTopology(t *testing.T) {
	canonical, err := os.ReadFile(filepath.Join("..", "..", "hooks", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	mutate := func(change func(*cachedHookManifest)) []byte {
		var manifest cachedHookManifest
		if err := json.Unmarshal(canonical, &manifest); err != nil {
			t.Fatal(err)
		}
		change(&manifest)
		raw, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	tests := []struct {
		name string
		raw  []byte
	}{
		{"wrong hooks type", []byte(`{"description":"present","hooks":[]}`)},
		{"unknown root field", append([]byte(`{"unknown":true,`), canonical[1:]...)},
		{"duplicate root field", []byte(`{"description":"one","description":"two","hooks":{}}`)},
		{"missing event", mutate(func(manifest *cachedHookManifest) { delete(manifest.Hooks, "Stop") })},
		{"extra event", mutate(func(manifest *cachedHookManifest) { manifest.Hooks["Other"] = manifest.Hooks["Stop"] })},
		{"wrong matcher", mutate(func(manifest *cachedHookManifest) { manifest.Hooks["Stop"][0].Matcher = "changed" })},
		{"multiple bindings", mutate(func(manifest *cachedHookManifest) {
			manifest.Hooks["Stop"] = append(manifest.Hooks["Stop"], manifest.Hooks["Stop"][0])
		})},
		{"multiple commands", mutate(func(manifest *cachedHookManifest) {
			manifest.Hooks["Stop"][0].Hooks = append(manifest.Hooks["Stop"][0].Hooks, manifest.Hooks["Stop"][0].Hooks[0])
		})},
		{"wrong command type", mutate(func(manifest *cachedHookManifest) { manifest.Hooks["Stop"][0].Hooks[0].Type = "prompt" })},
		{"wrong command path", mutate(func(manifest *cachedHookManifest) { manifest.Hooks["Stop"][0].Hooks[0].Command = "other" })},
		{"wrong timeout", mutate(func(manifest *cachedHookManifest) { manifest.Hooks["Stop"][0].Hooks[0].Timeout = 15 })},
		{"async override", mutate(func(manifest *cachedHookManifest) { value := false; manifest.Hooks["Stop"][0].Hooks[0].Async = &value })},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateCachedHookManifest(tc.raw); err == nil {
				t.Fatal("malformed hook topology was accepted")
			}
		})
	}
}

func TestPluginInstalledRequiresEveryOwnedPluginMember(t *testing.T) {
	required := []string{
		filepath.Join(".claude-plugin", "plugin.json"),
		filepath.Join("agents", "machinery-fsm-author.md"),
		filepath.Join("agents", "machinery-build-writer.md"),
		filepath.Join("hooks", "hooks.json"),
		filepath.Join("hooks", "machinery-hook.sh"),
		filepath.Join("skills", "machinery", "SKILL.md"),
		filepath.Join("skills", "machinery", "references", "build-md-template.md"),
		filepath.Join("skills", "machinery", "references", "c4-standalone.md"),
		filepath.Join("skills", "machinery", "references", "rebuild-guide.md"),
		filepath.Join("skills", "machinery", "references", "surface-ledger.md"),
		filepath.Join("skills", "machinery", "references", "target-surfaces.md"),
		filepath.Join("skills", "machinery", "references", "xstate-format.md"),
		filepath.Join("skills", "machinery", "tools", "README.md"),
		filepath.Join("skills", "machinery", "tools", "tlc.sh"),
		filepath.Join("skills", "machinery", "tools", "verify_formal.sh"),
	}
	for _, missing := range required {
		t.Run(filepath.ToSlash(missing), func(t *testing.T) {
			home := t.TempDir()
			root := seedCachedMachineryPlugin(t, home, "market")
			if err := os.Remove(filepath.Join(root, missing)); err != nil {
				t.Fatal(err)
			}
			if installed, err := pluginInstalled(home); err == nil || installed {
				t.Fatalf("cache missing %s = %v, %v; incomplete plugin must block", missing, installed, err)
			}
		})
	}
}

func TestPluginInstalledPropagatesEnumerationAndChildStatErrors(t *testing.T) {
	sentinel := errors.New("injected plugin discovery failure")
	t.Run("cache enumeration", func(t *testing.T) {
		preserveInstallDiscoveryHooks(t)
		home := t.TempDir()
		if err := os.MkdirAll(filepath.Join(home, "plugins", "cache"), 0o755); err != nil {
			t.Fatal(err)
		}
		readPluginCache = func(string) ([]fs.DirEntry, error) { return nil, sentinel }
		if installed, err := pluginInstalled(home); !errors.Is(err, sentinel) || installed {
			t.Fatalf("plugin enumeration = %v, %v", installed, err)
		}
	})
	t.Run("cache child stat", func(t *testing.T) {
		preserveInstallDiscoveryHooks(t)
		home := t.TempDir()
		marketplace := filepath.Join(home, "plugins", "cache", "market")
		if err := os.MkdirAll(marketplace, 0o755); err != nil {
			t.Fatal(err)
		}
		installDiscoveryLstat = func(path string) (os.FileInfo, error) {
			if path == marketplace {
				return nil, sentinel
			}
			return os.Lstat(path)
		}
		if installed, err := pluginInstalled(home); !errors.Is(err, sentinel) || installed {
			t.Fatalf("plugin child stat = %v, %v", installed, err)
		}
	})
}

func TestUpdateDiscoveryErrorsPrecedeDownloadJournalAndMutation(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("HOME override does not steer standard discovery paths on Windows")
	}
	sentinel := errors.New("injected direct discovery failure")
	tests := []struct {
		name      string
		configure func(*testing.T, string)
	}{
		{"plugin cache enumeration", func(t *testing.T, home string) {
			failed := filepath.Join(home, ".claude", "plugins", "cache")
			if err := os.MkdirAll(failed, 0o755); err != nil {
				t.Fatal(err)
			}
			readPluginCache = func(path string) ([]fs.DirEntry, error) {
				if path == failed {
					return nil, sentinel
				}
				return os.ReadDir(path)
			}
		}},
		{"plugin cache child stat", func(t *testing.T, home string) {
			seedCachedMachineryPlugin(t, filepath.Join(home, ".claude"), "market")
			failed := filepath.Join(home, ".claude", "plugins", "cache", "market")
			installDiscoveryLstat = failingDiscoveryLstat(failed, sentinel)
		}},
		{"shared agents skill", func(_ *testing.T, home string) {
			failed := filepath.Join(home, ".agents", "skills", "machinery", "SKILL.md")
			installDiscoveryLstat = failingDiscoveryLstat(failed, sentinel)
		}},
		{"Claude direct skill", func(_ *testing.T, home string) {
			failed := filepath.Join(home, ".claude", "skills", "machinery", "SKILL.md")
			installDiscoveryLstat = failingDiscoveryLstat(failed, sentinel)
		}},
		{"Claude direct symlink topology", func(t *testing.T, home string) {
			write(t, filepath.Join(home, ".agents", "skills", "machinery", "SKILL.md"), "skill")
			write(t, filepath.Join(home, ".claude", "skills", "machinery", "SKILL.md"), "skill")
			failed := filepath.Join(home, ".claude", "skills", "machinery")
			installDiscoveryLstat = failingDiscoveryLstat(failed, sentinel)
		}},
		{"Codex role", func(_ *testing.T, home string) {
			failed := filepath.Join(home, ".codex", "agents", roleSpecs[0].Name+".toml")
			installDiscoveryLstat = failingDiscoveryLstat(failed, sentinel)
		}},
		{"OpenCode plugin", func(_ *testing.T, home string) {
			failed := filepath.Join(home, ".config", "opencode", "plugins", "machinery.js")
			installDiscoveryLstat = failingDiscoveryLstat(failed, sentinel)
		}},
		{"OpenCode role", func(_ *testing.T, home string) {
			failed := filepath.Join(home, ".config", "opencode", "agents", roleSpecs[0].Name+".md")
			installDiscoveryLstat = failingDiscoveryLstat(failed, sentinel)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preserveInstallDiscoveryHooks(t)
			config := privateConfigDir(t)
			home := t.TempDir()
			t.Setenv("MACHINERY_CONFIG_DIR", config)
			t.Setenv("HOME", home)
			tc.configure(t, home)
			destination := filepath.Join(t.TempDir(), "machinery")
			_, err := Update(UpdateOptions{
				Executable:  destination,
				Repo:        "invalid/discovery-must-precede-network",
				Version:     "v1.2.3",
				SkipPlugins: true,
			})
			if !errors.Is(err, sentinel) {
				t.Fatalf("Update error = %v, want injected discovery failure", err)
			}
			if _, err := os.Lstat(destination); !os.IsNotExist(err) {
				t.Fatalf("discovery failure mutated destination: %v", err)
			}
			entries, err := os.ReadDir(config)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.Contains(entry.Name(), "journal") {
					t.Fatalf("discovery failure created journal %s", entry.Name())
				}
			}
		})
	}
}

func TestInstallPluginDiscoveryErrorPrecedesSourceJournalAndMutation(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("HOME override does not steer standard discovery paths on Windows")
	}
	preserveInstallDiscoveryHooks(t)
	sentinel := errors.New("injected install plugin discovery failure")
	config := privateConfigDir(t)
	home := t.TempDir()
	t.Setenv("MACHINERY_CONFIG_DIR", config)
	t.Setenv("HOME", home)
	failed := filepath.Join(home, ".claude", "plugins", "cache")
	if err := os.MkdirAll(failed, 0o755); err != nil {
		t.Fatal(err)
	}
	readPluginCache = func(path string) ([]fs.DirEntry, error) {
		if path == failed {
			return nil, sentinel
		}
		return os.ReadDir(path)
	}
	err := Install(Options{From: filepath.Join(t.TempDir(), "missing-source")})
	if !errors.Is(err, sentinel) {
		t.Fatalf("Install error = %v, want discovery failure before source resolution", err)
	}
	for _, path := range []string{
		filepath.Join(home, ".agents", "skills", "machinery"),
		filepath.Join(home, ".claude", "skills", "machinery"),
	} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("discovery failure mutated %s: %v", path, err)
		}
	}
	entries, err := os.ReadDir(config)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "journal") {
			t.Fatalf("discovery failure created journal %s", entry.Name())
		}
	}
}

func TestBuildRefreshPlanRejectsWrongDiscoveryTypes(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("HOME override does not steer standard discovery paths on Windows")
	}
	tests := []struct {
		name string
		path func(string) string
	}{
		{"shared skill", func(home string) string { return filepath.Join(home, ".agents", "skills", "machinery", "SKILL.md") }},
		{"Claude skill", func(home string) string { return filepath.Join(home, ".claude", "skills", "machinery", "SKILL.md") }},
		{"Codex role", func(home string) string { return filepath.Join(home, ".codex", "agents", roleSpecs[0].Name+".toml") }},
		{"OpenCode plugin", func(home string) string { return filepath.Join(home, ".config", "opencode", "plugins", "machinery.js") }},
		{"OpenCode role", func(home string) string {
			return filepath.Join(home, ".config", "opencode", "agents", roleSpecs[0].Name+".md")
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
			home := t.TempDir()
			t.Setenv("HOME", home)
			if err := os.MkdirAll(tc.path(home), 0o755); err != nil {
				t.Fatal(err)
			}
			if _, err := buildRefreshPlan(); err == nil || !strings.Contains(err.Error(), "expected regular file") {
				t.Fatalf("wrong-type discovery error = %v", err)
			}
		})
	}
}

func TestBuildRefreshPlanRejectsSymlinkedNativeDiscoveryFile(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("symlink and HOME discovery fixture is POSIX-specific")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(t.TempDir(), "role.toml")
	write(t, target, "role")
	link := filepath.Join(home, ".codex", "agents", roleSpecs[0].Name+".toml")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := buildRefreshPlan(); err == nil || !strings.Contains(err.Error(), "expected regular file") {
		t.Fatalf("symlinked native discovery error = %v", err)
	}
}

func preserveInstallDiscoveryHooks(t *testing.T) {
	t.Helper()
	oldReadDir := readPluginCache
	oldLstat := installDiscoveryLstat
	oldRead := installDiscoveryRead
	oldAfterOpen := cachedPluginAfterOpen
	oldBeforeFinalInventory := cachedPluginBeforeFinalInventory
	oldBeforeFinalTopology := cachedPluginBeforeFinalTopology
	oldAfterWitnessMember := cachedPluginAfterWitnessMember
	oldAfterCommitTopology := cachedPluginAfterCommitTopology
	t.Cleanup(func() {
		readPluginCache = oldReadDir
		installDiscoveryLstat = oldLstat
		installDiscoveryRead = oldRead
		cachedPluginAfterOpen = oldAfterOpen
		cachedPluginBeforeFinalInventory = oldBeforeFinalInventory
		cachedPluginBeforeFinalTopology = oldBeforeFinalTopology
		cachedPluginAfterWitnessMember = oldAfterWitnessMember
		cachedPluginAfterCommitTopology = oldAfterCommitTopology
	})
}

func failingDiscoveryLstat(failed string, sentinel error) func(string) (os.FileInfo, error) {
	return func(path string) (os.FileInfo, error) {
		if path == failed {
			return nil, sentinel
		}
		return os.Lstat(path)
	}
}
