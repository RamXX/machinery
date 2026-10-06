package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPluginCacheTopologyRejectsBlindStampABA(t *testing.T) {
	prior := installFileChangeID
	t.Cleanup(func() { installFileChangeID = prior })
	installFileChangeID = func(os.FileInfo) string { return "blind" }
	cache := filepath.Join(t.TempDir(), "cache")
	container := filepath.Join(cache, "market", "plugin")
	if err := os.MkdirAll(filepath.Join(container, "version"), 0o700); err != nil {
		t.Fatal(err)
	}
	initial, err := os.Stat(container)
	if err != nil {
		t.Fatal(err)
	}
	triggered := false
	_, err = capturePluginCacheTopologyWithHook(cache, func(pass int, directory string) {
		if triggered || pass != 2 || directory != "." {
			return
		}
		triggered = true
		path := filepath.Join(container, "version")
		parked := filepath.Join(container, "parked")
		if err := os.Rename(path, parked); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(parked, path); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(container, initial.ModTime(), initial.ModTime()); err != nil {
			t.Fatal(err)
		}
	})
	if !triggered {
		t.Fatal("mutation hook did not run")
	}
	if err == nil || !strings.Contains(err.Error(), "topology") {
		t.Fatalf("behind-cursor topology ABA accepted: %v", err)
	}
}

func TestPluginInstalledRejectsABAInGapBetweenTopologyPasses(t *testing.T) {
	preserveInstallDiscoveryHooks(t)
	prior := installFileChangeID
	t.Cleanup(func() { installFileChangeID = prior })
	installFileChangeID = func(os.FileInfo) string { return "blind" }
	home := t.TempDir()
	plugin := seedCachedMachineryPlugin(t, home, "market")
	container := filepath.Join(filepath.Dir(filepath.Dir(plugin)), "sibling")
	path := filepath.Join(container, "version")
	parked := filepath.Join(container, "parked")
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
	initial, err := os.Stat(container)
	if err != nil {
		t.Fatal(err)
	}
	triggered := false
	cachedPluginAfterWitnessMember = func(pass int, _ string) {
		if triggered || pass != 3 {
			return
		}
		triggered = true
		if err := os.Rename(path, parked); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(parked, path); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(container, initial.ModTime(), initial.ModTime()); err != nil {
			t.Fatal(err)
		}
	}
	installed, err := pluginInstalled(home)
	if !triggered {
		t.Fatal("mutation hook did not run")
	}
	if installed || err == nil || !strings.Contains(err.Error(), "topology") {
		t.Fatalf("topology ABA between passes accepted: installed=%v err=%v", installed, err)
	}
}
