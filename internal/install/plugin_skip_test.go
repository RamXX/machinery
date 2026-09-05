package install

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	machversion "github.com/RamXX/machinery/internal/version"
)

// TestInstallSkipsHomeServedByPlugin: when the machinery Claude Code plugin
// is already cached under ~/.claude, the default install must not lay a
// duplicate skill there; ~/.agents (and any explicit --home) still get one.
func TestInstallSkipsHomeServedByPlugin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// the plugin cache layout: <home>/.claude/plugins/cache/<marketplace>/<plugin>/<version>
	seedCachedMachineryPlugin(t, filepath.Join(home, ".claude"), "machinery")

	var out bytes.Buffer
	if err := Install(Options{From: "../..", Out: &out}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "machinery", "SKILL.md")); err != nil {
		t.Fatalf("~/.agents must still receive the canonical copy: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "machinery")); !os.IsNotExist(err) {
		t.Fatalf("~/.claude must be skipped when the plugin serves it (err=%v)", err)
	}
	if !strings.Contains(out.String(), "skipping") {
		t.Fatalf("the skip must be announced, got %q", out.String())
	}
}

func TestPluginInstalledIgnoresCompleteOldVersion(t *testing.T) {
	home := t.TempDir()
	root := seedCachedMachineryPlugin(t, home, "market")
	old := filepath.Join(filepath.Dir(root), "0.1.0")
	if err := os.Rename(root, old); err != nil {
		t.Fatal(err)
	}
	if installed, err := pluginInstalled(home); err != nil || installed {
		t.Fatalf("old-only cache must not suppress the current direct install: installed=%v err=%v", installed, err)
	}
}

// TestInstallExplicitHomeWinsOverPluginSkip: an explicit --home is honored
// even when the plugin is present; the filter applies to defaults only.
func TestInstallExplicitHomeWinsOverPluginSkip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	claude := filepath.Join(home, ".claude")
	seedCachedMachineryPlugin(t, claude, "machinery")
	if err := Install(Options{Homes: []string{claude}, From: "../.."}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(claude, "skills", "machinery", "SKILL.md")); err != nil {
		t.Fatalf("an explicit home must be honored: %v", err)
	}
}

func TestPluginInstalledRejectsMixedVersionAndUnexpectedInventory(t *testing.T) {
	t.Run("mixed skill version", func(t *testing.T) {
		home := t.TempDir()
		root := seedCachedMachineryPlugin(t, home, "market")
		skill := filepath.Join(root, "skills", "machinery", "SKILL.md")
		raw, err := os.ReadFile(skill)
		if err != nil {
			t.Fatal(err)
		}
		current := []byte(`version: "` + strings.TrimPrefix(machversion.Version, "v") + `"`)
		raw = bytes.Replace(raw, current, []byte(`version: "0.1.0"`), 1)
		if err := os.WriteFile(skill, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		if installed, err := pluginInstalled(home); err == nil || installed || !strings.Contains(err.Error(), "metadata version") {
			t.Fatalf("mixed-version cache = installed %v, error %v", installed, err)
		}
	})

	t.Run("host-owned members are tolerated", func(t *testing.T) {
		home := t.TempDir()
		root := seedCachedMachineryPlugin(t, home, "market")
		// Members machinery does not own, in every walked root: a file the host
		// or marketplace adds next to an owned one, and a whole directory.
		write(t, filepath.Join(root, "agents", "host-added-agent.md"), "host\n")
		write(t, filepath.Join(root, "hooks", "README.md"), "host\n")
		write(t, filepath.Join(root, "skills", "machinery", "extras", "notes.md"), "host\n")
		if installed, err := pluginInstalled(home); err != nil || !installed {
			t.Fatalf("cache with host-owned members = installed %v, error %v; want installed", installed, err)
		}
	})

	t.Run("symlinked host member still fails closed", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink fixture")
		}
		home := t.TempDir()
		root := seedCachedMachineryPlugin(t, home, "market")
		if err := os.Symlink(filepath.Join(root, "skills", "machinery", "SKILL.md"), filepath.Join(root, "agents", "link.md")); err != nil {
			t.Fatal(err)
		}
		if installed, err := pluginInstalled(home); err == nil || installed || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("symlinked member = installed %v, error %v", installed, err)
		}
	})
}

func TestPluginInstalledRetainedReadRejectsSwapAndDetectsABA(t *testing.T) {
	t.Run("lingering replacement", func(t *testing.T) {
		preserveInstallDiscoveryHooks(t)
		home := t.TempDir()
		root := seedCachedMachineryPlugin(t, home, "market")
		skillRel := filepath.Join("skills", "machinery", "SKILL.md")
		skill := filepath.Join(root, skillRel)
		cachedPluginAfterOpen = func(path string) {
			if path != skillRel {
				return
			}
			cachedPluginAfterOpen = func(string) {}
			if err := os.Rename(skill, skill+".original"); err != nil {
				t.Fatal(err)
			}
			write(t, skill, strings.Repeat("malicious\n", 256))
		}
		if installed, err := pluginInstalled(home); err == nil || installed || !strings.Contains(err.Error(), "changed while being read") {
			t.Fatalf("swapped cache member = installed %v, error %v", installed, err)
		}
	})

	t.Run("ABA replacement", func(t *testing.T) {
		preserveInstallDiscoveryHooks(t)
		home := t.TempDir()
		root := seedCachedMachineryPlugin(t, home, "market")
		skillRel := filepath.Join("skills", "machinery", "SKILL.md")
		skill := filepath.Join(root, skillRel)
		cachedPluginAfterOpen = func(path string) {
			if path != skillRel {
				return
			}
			cachedPluginAfterOpen = func(string) {}
			if err := os.Rename(skill, skill+".original"); err != nil {
				t.Fatal(err)
			}
			write(t, skill, strings.Repeat("malicious\n", 256))
			if err := os.Remove(skill); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(skill+".original", skill); err != nil {
				t.Fatal(err)
			}
		}
		if installed, err := pluginInstalled(home); err == nil || installed || !strings.Contains(err.Error(), "inventory changed") {
			t.Fatalf("cached member ABA was not detected: installed %v, error %v", installed, err)
		}
	})
}

func TestPluginInstalledRejectsConcurrentFinalInventoryMutation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(string) error
	}{
		{
			name: "added member",
			mutate: func(root string) error {
				return os.WriteFile(filepath.Join(root, "agents", "unexpected.md"), []byte("unexpected\n"), 0o644)
			},
		},
		{
			name: "removed member",
			mutate: func(root string) error {
				return os.Remove(filepath.Join(root, "agents", RoleDocs[0]))
			},
		},
		{
			name: "replaced member",
			mutate: func(root string) error {
				path := filepath.Join(root, "agents", RoleDocs[0])
				raw, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				saved := filepath.Join(filepath.Dir(root), "saved-role")
				if err := os.Rename(path, saved); err != nil {
					return err
				}
				return os.WriteFile(path, raw, 0o644)
			},
		},
		{
			name: "ABA member",
			mutate: func(root string) error {
				path := filepath.Join(root, "agents", RoleDocs[0])
				saved := filepath.Join(filepath.Dir(root), "saved-role")
				if err := os.Rename(path, saved); err != nil {
					return err
				}
				if err := os.WriteFile(path, []byte("malicious transient replacement\n"), 0o644); err != nil {
					return err
				}
				if err := os.Remove(path); err != nil {
					return err
				}
				return os.Rename(saved, path)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preserveInstallDiscoveryHooks(t)
			home := t.TempDir()
			root := seedCachedMachineryPlugin(t, home, "market")
			if err := os.Chtimes(filepath.Join(root, "agents"), time.Unix(1, 0), time.Unix(1, 0)); err != nil {
				t.Fatal(err)
			}
			start := make(chan struct{})
			done := make(chan error, 1)
			cachedPluginBeforeFinalInventory = func(gotRoot string) {
				cachedPluginBeforeFinalInventory = func(string) {}
				if gotRoot != root {
					t.Errorf("final inventory root = %s, want %s", gotRoot, root)
				}
				close(start)
				if err := <-done; err != nil {
					t.Errorf("concurrent inventory mutation: %v", err)
				}
			}
			go func() {
				<-start
				done <- tc.mutate(root)
			}()
			installed, err := pluginInstalled(home)
			if err == nil || installed || !strings.Contains(err.Error(), "inventory") {
				t.Fatalf("concurrent %s = installed %v, error %v", tc.name, installed, err)
			}
		})
	}
}

func TestPluginInstalledRejectsMutationAfterWitnessMemberWasScanned(t *testing.T) {
	tests := []struct {
		name string
		pass int
		aba  bool
	}{
		{name: "first witness pass", pass: 1},
		{name: "second witness pass", pass: 2},
		{name: "last digest witness pass", pass: 3},
		{name: "last digest witness pass ABA", pass: 3, aba: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preserveInstallDiscoveryHooks(t)
			home := t.TempDir()
			root := seedCachedMachineryPlugin(t, home, "market")
			relative := filepath.Join(".claude-plugin", "plugin.json")
			path := filepath.Join(root, relative)
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			originalInfo, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if tc.aba && installFileChangeID(originalInfo) == "" {
				t.Skip("platform does not expose a file change identity")
			}
			replacement := append([]byte(nil), original...)
			replacement[len(replacement)/2] ^= 1
			start := make(chan struct{})
			done := make(chan error, 1)
			triggered := false
			laterMemberScanned := false
			cachedPluginAfterWitnessMember = func(pass int, gotPath string) {
				if pass != tc.pass {
					return
				}
				if triggered {
					laterMemberScanned = true
					return
				}
				if gotPath != relative {
					return
				}
				triggered = true
				close(start)
				if err := <-done; err != nil {
					t.Errorf("concurrent witness mutation: %v", err)
				}
			}
			go func() {
				<-start
				if err := os.WriteFile(path, replacement, originalInfo.Mode().Perm()); err != nil {
					done <- err
					return
				}
				if tc.aba {
					if err := os.WriteFile(path, original, originalInfo.Mode().Perm()); err != nil {
						done <- err
						return
					}
					if err := os.Chtimes(path, originalInfo.ModTime(), originalInfo.ModTime()); err != nil {
						done <- err
						return
					}
				}
				done <- nil
			}()
			installed, err := pluginInstalled(home)
			if err == nil || installed || !strings.Contains(err.Error(), "success witness") {
				t.Fatalf("post-scan witness mutation = installed %v, error %v", installed, err)
			}
			if !laterMemberScanned {
				t.Fatal("mutation did not occur while later inventory members remained to scan")
			}
		})
	}
}

func TestPluginInstalledSuccessWitnessCoversLateGlobalTopologyMutation(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, string, string) func() error
	}{
		{
			name: "sibling marketplace added during last selected pass",
			prepare: func(t *testing.T, home, _ string) func() error {
				staging := t.TempDir()
				stagedRoot := seedCachedMachineryPlugin(t, staging, "sibling")
				stagedMarketplace := filepath.Dir(filepath.Dir(stagedRoot))
				cache := filepath.Join(home, "plugins", "cache")
				return func() error {
					return os.Rename(stagedMarketplace, filepath.Join(cache, "sibling"))
				}
			},
		},
		{
			name: "sibling plugin version added during last selected pass",
			prepare: func(t *testing.T, _, root string) func() error {
				container := filepath.Join(filepath.Dir(filepath.Dir(root)), "sibling-plugin")
				if err := os.MkdirAll(filepath.Join(container, "0.1.0"), 0o755); err != nil {
					t.Fatal(err)
				}
				return func() error {
					return os.Mkdir(filepath.Join(container, "0.2.0"), 0o755)
				}
			},
		},
		{
			name: "sibling plugin version ABA during last selected pass",
			prepare: func(t *testing.T, _, root string) func() error {
				container := filepath.Join(filepath.Dir(filepath.Dir(root)), "sibling-plugin")
				if err := os.MkdirAll(filepath.Join(container, "0.1.0"), 0o755); err != nil {
					t.Fatal(err)
				}
				originalInfo, err := os.Stat(container)
				if err != nil {
					t.Fatal(err)
				}
				if installFileChangeID(originalInfo) == "" {
					t.Skip("platform does not expose a directory change identity")
				}
				transient := filepath.Join(container, "0.2.0")
				return func() error {
					if err := os.Mkdir(transient, 0o755); err != nil {
						return err
					}
					if err := os.Remove(transient); err != nil {
						return err
					}
					return os.Chtimes(container, originalInfo.ModTime(), originalInfo.ModTime())
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preserveInstallDiscoveryHooks(t)
			home := t.TempDir()
			root := seedCachedMachineryPlugin(t, home, "market")
			mutate := tc.prepare(t, home, root)
			relative := filepath.Join(".claude-plugin", "plugin.json")
			start := make(chan struct{})
			done := make(chan error, 1)
			triggered := false
			laterMemberScanned := false
			cachedPluginAfterWitnessMember = func(pass int, gotPath string) {
				if pass != 3 {
					return
				}
				if triggered {
					laterMemberScanned = true
					return
				}
				if gotPath != relative {
					return
				}
				triggered = true
				close(start)
				if err := <-done; err != nil {
					t.Errorf("concurrent topology mutation: %v", err)
				}
			}
			go func() {
				<-start
				done <- mutate()
			}()
			installed, err := pluginInstalled(home)
			if err == nil || installed || !strings.Contains(err.Error(), "topology") {
				t.Fatalf("late global topology mutation = installed %v, error %v", installed, err)
			}
			if !laterMemberScanned {
				t.Fatal("topology mutation did not occur while later selected members remained to scan")
			}
		})
	}
}

func TestPluginInstalledCommitTopologyRejectsBehindCursorNestedMutation(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, string) func() error
	}{
		{
			name: "late nested version addition",
			prepare: func(_ *testing.T, container string) func() error {
				return func() error { return os.Mkdir(filepath.Join(container, "0.1.0"), 0o755) }
			},
		},
		{
			name: "late nested version removal",
			prepare: func(t *testing.T, container string) func() error {
				old := filepath.Join(container, "0.1.0")
				if err := os.Mkdir(old, 0o755); err != nil {
					t.Fatal(err)
				}
				return func() error { return os.Remove(old) }
			},
		},
		{
			name: "late nested version ABA",
			prepare: func(t *testing.T, container string) func() error {
				originalInfo, err := os.Stat(container)
				if err != nil {
					t.Fatal(err)
				}
				if installFileChangeID(originalInfo) == "" {
					t.Skip("platform does not expose a directory change identity")
				}
				transient := filepath.Join(container, "0.1.0")
				return func() error {
					if err := os.Mkdir(transient, 0o755); err != nil {
						return err
					}
					if err := os.Remove(transient); err != nil {
						return err
					}
					return os.Chtimes(container, originalInfo.ModTime(), originalInfo.ModTime())
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preserveInstallDiscoveryHooks(t)
			home := t.TempDir()
			root := seedCachedMachineryPlugin(t, home, "market")
			container := filepath.Dir(root)
			laterContainer := filepath.Join(filepath.Dir(container), "zz-sibling", "0.1.0")
			if err := os.MkdirAll(laterContainer, 0o755); err != nil {
				t.Fatal(err)
			}
			mutate := tc.prepare(t, container)
			target := filepath.Join("market", "machinery")
			start := make(chan struct{})
			done := make(chan error, 1)
			triggered := false
			laterDirectoryScanned := false
			cachedPluginAfterCommitTopology = func(pass int, directory string) {
				if pass != 2 {
					return
				}
				if triggered {
					laterDirectoryScanned = true
					return
				}
				if directory != target {
					return
				}
				triggered = true
				close(start)
				if err := <-done; err != nil {
					t.Errorf("concurrent nested topology mutation: %v", err)
				}
			}
			go func() {
				<-start
				done <- mutate()
			}()
			installed, err := pluginInstalled(home)
			if err == nil || installed || !strings.Contains(err.Error(), "topology") {
				t.Fatalf("behind-cursor nested mutation = installed %v, error %v", installed, err)
			}
			if !laterDirectoryScanned {
				t.Fatal("nested mutation did not occur while a later topology directory remained to scan")
			}
		})
	}
}

func TestPluginInstalledRejectsConcurrentFinalTopologyMutation(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, string, string) (func() error, []string)
	}{
		{
			name: "sibling marketplace addition",
			prepare: func(t *testing.T, home, _ string) (func() error, []string) {
				staging := t.TempDir()
				stagedRoot := seedCachedMachineryPlugin(t, staging, "sibling")
				stagedMarketplace := filepath.Dir(filepath.Dir(stagedRoot))
				cache := filepath.Join(home, "plugins", "cache")
				return func() error {
					return os.Rename(stagedMarketplace, filepath.Join(cache, "sibling"))
				}, []string{cache}
			},
		},
		{
			name: "malformed cache entry addition",
			prepare: func(_ *testing.T, home, _ string) (func() error, []string) {
				cache := filepath.Join(home, "plugins", "cache")
				return func() error {
					return os.WriteFile(filepath.Join(cache, "malformed"), []byte("not a marketplace\n"), 0o644)
				}, []string{cache}
			},
		},
		{
			name: "sibling plugin version addition",
			prepare: func(t *testing.T, _, root string) (func() error, []string) {
				marketplace := filepath.Dir(filepath.Dir(root))
				container := filepath.Join(marketplace, "sibling-plugin")
				if err := os.MkdirAll(filepath.Join(container, "0.1.0"), 0o755); err != nil {
					t.Fatal(err)
				}
				return func() error {
					return os.Mkdir(filepath.Join(container, "0.2.0"), 0o755)
				}, []string{container}
			},
		},
		{
			name: "current version removal",
			prepare: func(_ *testing.T, _, root string) (func() error, []string) {
				container := filepath.Dir(root)
				return func() error { return os.RemoveAll(root) }, []string{container}
			},
		},
		{
			name: "marketplace replacement",
			prepare: func(t *testing.T, _, root string) (func() error, []string) {
				staging := t.TempDir()
				stagedRoot := seedCachedMachineryPlugin(t, staging, "market")
				stagedMarketplace := filepath.Dir(filepath.Dir(stagedRoot))
				marketplace := filepath.Dir(filepath.Dir(root))
				saved := filepath.Join(t.TempDir(), "original-marketplace")
				return func() error {
					if err := os.Rename(marketplace, saved); err != nil {
						return err
					}
					return os.Rename(stagedMarketplace, marketplace)
				}, []string{filepath.Dir(marketplace)}
			},
		},
		{
			name: "version container ABA",
			prepare: func(_ *testing.T, _, root string) (func() error, []string) {
				container := filepath.Dir(root)
				transient := filepath.Join(container, "0.1.0")
				return func() error {
					if err := os.Mkdir(transient, 0o755); err != nil {
						return err
					}
					return os.Remove(transient)
				}, []string{container}
			},
		},
		{
			name: "current cache ABA",
			prepare: func(t *testing.T, home, _ string) (func() error, []string) {
				cache := filepath.Join(home, "plugins", "cache")
				saved := filepath.Join(t.TempDir(), "original-cache")
				return func() error {
					if err := os.Rename(cache, saved); err != nil {
						return err
					}
					if err := os.Mkdir(cache, 0o755); err != nil {
						return err
					}
					if err := os.Remove(cache); err != nil {
						return err
					}
					return os.Rename(saved, cache)
				}, []string{filepath.Dir(cache)}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preserveInstallDiscoveryHooks(t)
			home := t.TempDir()
			root := seedCachedMachineryPlugin(t, home, "market")
			mutate, parents := tc.prepare(t, home, root)
			for _, parent := range parents {
				if err := os.Chtimes(parent, time.Unix(1, 0), time.Unix(1, 0)); err != nil {
					t.Fatal(err)
				}
			}
			start := make(chan struct{})
			done := make(chan error, 1)
			cache := filepath.Join(home, "plugins", "cache")
			cachedPluginBeforeFinalTopology = func(gotCache string) {
				cachedPluginBeforeFinalTopology = func(string) {}
				if gotCache != cache {
					t.Errorf("final topology cache = %s, want %s", gotCache, cache)
				}
				close(start)
				if err := <-done; err != nil {
					t.Errorf("concurrent topology mutation: %v", err)
				}
			}
			go func() {
				<-start
				done <- mutate()
			}()
			installed, err := pluginInstalled(home)
			if err == nil || installed || !strings.Contains(err.Error(), "topology") {
				t.Fatalf("concurrent %s = installed %v, error %v", tc.name, installed, err)
			}
		})
	}
}

func seedCachedMachineryPlugin(t *testing.T, claudeHome, marketplace string) string {
	t.Helper()
	root := filepath.Join(claudeHome, "plugins", "cache", marketplace, "machinery", strings.TrimPrefix(machversion.Version, "v"))
	files := []string{
		filepath.Join(".claude-plugin", "plugin.json"),
		// Every real cache entry is a copy of the plugin source tree and carries
		// the marketplace manifest next to plugin.json; the fixture mirrors that.
		filepath.Join(".claude-plugin", "marketplace.json"),
		filepath.Join("agents", "machinery-fsm-author.md"),
		filepath.Join("agents", "machinery-build-writer.md"),
		filepath.Join("hooks", "hooks.json"),
		filepath.Join("hooks", "machinery-hook.sh"),
		filepath.Join("skills", "machinery", "SKILL.md"),
		filepath.Join("skills", "machinery", "references", "archaeology-classification.md"),
		filepath.Join("skills", "machinery", "references", "build-md-template.md"),
		filepath.Join("skills", "machinery", "references", "c4-standalone.md"),
		filepath.Join("skills", "machinery", "references", "execution-packets.md"),
		filepath.Join("skills", "machinery", "references", "rebuild-guide.md"),
		filepath.Join("skills", "machinery", "references", "surface-ledger.md"),
		filepath.Join("skills", "machinery", "references", "target-surfaces.md"),
		filepath.Join("skills", "machinery", "references", "verification-evidence.md"),
		filepath.Join("skills", "machinery", "references", "xstate-format.md"),
		filepath.Join("skills", "machinery", "tools", "README.md"),
		filepath.Join("skills", "machinery", "tools", "tlc.sh"),
		filepath.Join("skills", "machinery", "tools", "verify_formal.sh"),
	}
	for _, file := range files {
		source := filepath.Join("..", "..", file)
		destination := filepath.Join(root, file)
		raw, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(source)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, raw, info.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
