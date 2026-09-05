package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// These fixtures exercise the actual Cobra CLI, release downloads, child binary
// validation, delegated install lock, receipt persistence, and filesystem writes.
// Only the existing release endpoint strings and release version are linked to
// local values; no command runner or replacement executable is simulated.
type bootstrapRelease struct {
	old, next, source string
	endpoint          string
	nextBytes         []byte
	hold              atomic.Bool
	broken            atomic.Bool
	requested         chan struct{}
	release           chan struct{}
}

func bootstrapCommand(t *testing.T, dir string, env []string, name string, args ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, name, args...)
	c.Dir, c.Env = dir, env
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	return out
}

func bootstrapReleaseFixture(t *testing.T) *bootstrapRelease {
	t.Helper()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	f := &bootstrapRelease{requested: make(chan struct{}, 1), release: make(chan struct{})}
	root := t.TempDir()
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(root, "source.tar.gz")
	bootstrapCommand(t, repo, os.Environ(), "go", "run", "./cmd/release-archive", "-git-root", repo, "-epoch", "1700000000", "-output", archivePath)
	archive := bootstrapRead(t, archivePath)
	broken := bootstrapWithoutAdapter(t, archive)
	asset, err := releaseAssetName()
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		source := archive
		if f.broken.Load() {
			source = broken
		}
		switch {
		case strings.HasPrefix(r.URL.Path, "/repos/fixture/machinery/releases/tags/"):
			fmt.Fprint(w, `{ "tag_name": "v9.9.2" }`)
		case strings.HasSuffix(r.URL.Path, "/"+asset):
			w.Write(f.nextBytes)
		case strings.HasSuffix(r.URL.Path, "/machinery-source.tar.gz"):
			if f.hold.Load() {
				f.requested <- struct{}{}
				select {
				case <-f.release:
				case <-r.Context().Done():
					return
				}
			}
			w.Write(source)
		case strings.HasSuffix(r.URL.Path, "/checksums-sha256.txt"):
			fmt.Fprintf(w, "%x  %s\n%x  machinery-source.tar.gz\n", sha256.Sum256(f.nextBytes), asset, sha256.Sum256(source))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	f.endpoint = server.URL
	t.Cleanup(func() { close(f.release) })
	f.old, f.next = filepath.Join(root, "old"), filepath.Join(root, "next")
	for i, binary := range []string{f.old, f.next} {
		flags := fmt.Sprintf("-X main.version=v9.9.%d -X github.com/RamXX/machinery/internal/install.githubBase=%s -X github.com/RamXX/machinery/internal/install.apiBase=%s", i+1, server.URL, server.URL)
		bootstrapCommand(t, repo, os.Environ(), "go", "build", "-ldflags", flags, "-o", binary, "./cmd/machinery")
	}
	f.nextBytes = bootstrapRead(t, f.next)
	if err := extractTarGz(archivePath, filepath.Join(root, "source")); err != nil {
		t.Fatal(err)
	}
	f.source = filepath.Join(root, "source", "machinery")
	// A historical source fixture retains the complete real shipped inventory.
	// Its visible marker makes stale skill, role, command, and plugin bytes observable.
	err = filepath.WalkDir(f.source, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(path, append(body, []byte("\n// bootstrap historical asset\n")...), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func bootstrapWithoutAdapter(t *testing.T, archive []byte) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var out bytes.Buffer
	gz := gzip.NewWriter(&out)
	writer := tar.NewWriter(gz)
	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Name == "machinery/adapters/opencode/plugins/machinery.js" {
			continue
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(writer, tr); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func bootstrapRead(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

type bootstrapInstall struct {
	root, home, config, binary string
	env                        []string
	receipt                    installReceipt
}

func bootstrapSeed(t *testing.T, release *bootstrapRelease) bootstrapInstall {
	t.Helper()
	f := bootstrapInstall{root: t.TempDir()}
	f.home, f.config, f.binary = filepath.Join(f.root, "home"), filepath.Join(f.root, "config"), filepath.Join(f.root, "bin", "machinery")
	for _, path := range []string{f.home, f.config, filepath.Dir(f.binary), filepath.Join(f.root, "tmp")} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(f.binary, bootstrapRead(t, release.old), 0o755); err != nil {
		t.Fatal(err)
	}
	// Set both process and child roots: helpers and actual CLI use the same isolated scope.
	for key, value := range map[string]string{"HOME": f.home, "USERPROFILE": f.home, "MACHINERY_CONFIG_DIR": f.config, "XDG_CONFIG_HOME": filepath.Join(f.home, ".config"), "TMPDIR": filepath.Join(f.root, "tmp")} {
		t.Setenv(key, value)
	}
	f.env = os.Environ()
	for _, args := range [][]string{
		{"--home", filepath.Join(f.home, "custom-a"), "--home", filepath.Join(f.home, "custom-b")},
		{"--home", filepath.Join(f.home, "copy-a"), "--home", filepath.Join(f.home, "copy-b"), "--copy"},
		{"--target", "codex"}, {"--target", "opencode", "--copy"},
	} {
		bootstrapCommand(t, f.root, f.env, f.binary, append([]string{"install", "--from", release.source}, args...)...)
	}
	write(t, filepath.Join(f.home, "unrelated", "sentinel"), "unrelated host content")
	var err error
	f.receipt, _, err = loadReceipt()
	if err != nil {
		t.Fatal(err)
	}
	if len(f.receipt.HomeInstalls) != 2 || len(f.receipt.Targets) != 2 {
		t.Fatalf("real install omitted topology: %+v", f.receipt)
	}
	f.receipt.HostPlugins = []string{"claude", "codex"}
	raw, err := json.Marshal(f.receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.config, "install.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f bootstrapInstall) args(bootstrap bool) []string {
	args := []string{"update", "--version", "v9.9.2", "--repo", "fixture/machinery", "--install-dir", filepath.Dir(f.binary), "--skip-plugins"}
	if bootstrap {
		args = append(args, "--bootstrap-defaults")
	}
	return args
}

func bootstrapState(t *testing.T, f bootstrapInstall) map[string]string {
	t.Helper()
	result := map[string]string{}
	for _, root := range []string{f.home, filepath.Dir(f.binary)} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			value := ""
			if info.Mode()&os.ModeSymlink != 0 {
				value, err = os.Readlink(path)
			} else {
				var raw []byte
				raw, err = os.ReadFile(path)
				value = fmt.Sprintf("%x", sha256.Sum256(raw))
			}
			result[path] = fmt.Sprintf("%v:%s", info.Mode(), value)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	result["receipt"] = bootstrapReceiptState(t, filepath.Join(f.config, "install.json"))
	return result
}

func bootstrapReceiptState(t *testing.T, path string) string {
	t.Helper()
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "absent"
	}
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%v:%s", info.Mode(), bootstrapRead(t, path))
}

func bootstrapRootAlias(path, root, canonical string) string {
	if path == root || strings.HasPrefix(path, root+string(os.PathSeparator)) {
		return canonical + strings.TrimPrefix(path, root)
	}
	return path
}

func TestBootstrapReceiptRootAlias(t *testing.T) {
	for _, tc := range []struct{ name, path, want string }{
		{"root", "/alias", "/canonical/alias"},
		{"descendant", "/alias/config/scratch", "/canonical/alias/config/scratch"},
		{"canonical root", "/canonical/alias", "/canonical/alias"},
		{"canonical descendant", "/canonical/alias/config/scratch", "/canonical/alias/config/scratch"},
		{"similar sibling", "/alias-other/config/scratch", "/alias-other/config/scratch"},
		{"interior alias", "/foreign/alias/config/scratch", "/foreign/alias/config/scratch"},
		{"repeated suffix", "/alias/nested/alias/file", "/canonical/alias/nested/alias/file"},
		{"sibling journal", "/alias/config/other-journal/scratch", "/canonical/alias/config/other-journal/scratch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := bootstrapRootAlias(tc.path, "/alias", "/canonical/alias")
			if got != tc.want || bootstrapRootAlias(got, "/alias", "/canonical/alias") != got {
				t.Fatalf("alias mapped foreign path or was not idempotent: %q want %q", got, tc.want)
			}
		})
	}
}

func TestBootstrapReceiptPlan(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	t.Run("first bootstrap defaults control", func(t *testing.T) {
		got, err := updatePlan(UpdateOptions{BootstrapDefaults: true})
		if err != nil {
			t.Fatal(err)
		}
		want, _ := DefaultHomes()
		if len(got.HomeInstalls) != 1 || !reflect.DeepEqual(got.HomeInstalls[0].Homes, want) {
			t.Fatalf("defaults: %+v want %v", got, want)
		}
	})
	t.Run("supported schema one control", func(t *testing.T) {
		t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
		writeLegacyReceipt(t, installReceipt{})
		if _, err := updatePlan(UpdateOptions{BootstrapDefaults: true}); err != nil {
			t.Fatalf("supported schema one rejected: %v", err)
		}
	})
	for _, opts := range []UpdateOptions{{Homes: []string{"/explicit"}}, {Targets: []string{"codex"}}} {
		opts.BootstrapDefaults = true
		if _, err := updatePlan(opts); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
			t.Fatalf("explicit selector accepted: %v", err)
		}
	}
	t.Run("complete recorded mixed plan", func(t *testing.T) {
		home := os.Getenv("HOME")
		repo, err := filepath.Abs("../..")
		if err != nil {
			t.Fatal(err)
		}
		for _, opts := range []Options{{Homes: []string{filepath.Join(home, "a"), filepath.Join(home, "b")}}, {Homes: []string{filepath.Join(home, "c")}, Copy: true}, {Targets: []string{"codex"}}, {Targets: []string{"opencode"}, Copy: true}} {
			opts.From, opts.Record = repo, true
			if err := Install(opts); err != nil {
				t.Fatal(err)
			}
		}
		r, _, err := loadReceipt()
		if err != nil {
			t.Fatal(err)
		}
		r.HostPlugins = []string{"claude", "codex"}
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(os.Getenv("MACHINERY_CONFIG_DIR"), "install.json"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
		want, err := updatePlan(UpdateOptions{})
		if err != nil {
			t.Fatal(err)
		}
		got, err := updatePlan(UpdateOptions{BootstrapDefaults: true})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("bootstrap lost receipt plan\ngot: %+v\nwant: %+v", got, want)
		}
	})
}

// The reference is installed independently by the real release CLI from the
// current source, before the tested updater runs. It includes every file and
// link, rather than trusting whichever inventory that updater publishes.
func bootstrapContentState(t *testing.T, f bootstrapInstall) map[string]string {
	t.Helper()
	state := bootstrapState(t, f)
	delete(state, "receipt")
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	// Coordination lock names encode each fixture's distinct config scope.
	locks := filepath.Join(cache, "machinery", "locks") + string(os.PathSeparator)
	normalized := map[string]string{}
	for path, value := range state {
		if strings.HasPrefix(path, locks) {
			continue
		}
		normalized[strings.ReplaceAll(path, f.root, "$ROOT")] = strings.ReplaceAll(value, f.root, "$ROOT")
	}
	return normalized
}

func bootstrapAssertConvergence(t *testing.T, f bootstrapInstall, release *bootstrapRelease, expected map[string]string, bootstrap bool) {
	t.Helper()
	before := bootstrapState(t, f)
	bootstrapCommand(t, f.root, f.env, f.binary, f.args(bootstrap)...)
	if !bytes.Equal(bootstrapRead(t, f.binary), release.nextBytes) {
		t.Error("binary does not match verified release digest")
	}
	if got := string(bootstrapCommand(t, f.root, f.env, f.binary, "version")); got != "machinery version v9.9.2\n" {
		t.Errorf("wrong release version: %s", got)
	}
	r, _, err := loadReceipt()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.HomeInstalls, f.receipt.HomeInstalls) || !reflect.DeepEqual(r.Targets, f.receipt.Targets) || !reflect.DeepEqual(r.HostPlugins, f.receipt.HostPlugins) {
		t.Errorf("rerun changed recorded topology: %+v", r)
	}
	wantPaths, gotPaths := []string{}, []string{}
	for _, a := range f.receipt.Artifacts {
		wantPaths = append(wantPaths, a.Path)
	}
	for _, a := range r.Artifacts {
		gotPaths = append(gotPaths, a.Path)
	}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Errorf("incomplete receipt inventory: got %v want %v", gotPaths, wantPaths)
	}
	if got := bootstrapContentState(t, f); !reflect.DeepEqual(got, expected) {
		t.Error("installed complete content/modes/link destinations differ from independent current-release installation")
		for path, want := range expected {
			if got[path] != want {
				t.Errorf("release content %s: got %s want %s", path, got[path], want)
			}
		}
		for path := range got {
			if _, ok := expected[path]; !ok {
				t.Errorf("unexpected installed content: %s", path)
			}
		}
	}
	for _, a := range f.receipt.Artifacts {
		err := filepath.WalkDir(a.Path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if bytes.Contains(bootstrapRead(t, path), []byte("bootstrap historical asset")) {
				t.Errorf("recorded target remains stale: %s", path)
			}
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	}
	for _, a := range r.Artifacts {
		digest, err := artifactTreeDigest(a.Path)
		if err != nil || digest != a.Digest {
			t.Errorf("receipt digest mismatch %s: %s %v", a.Path, digest, err)
		}
	}
	after := bootstrapState(t, f)
	for path, value := range before {
		if strings.Contains(path, "unrelated") && after[path] != value {
			t.Errorf("unrelated file changed: %s", path)
		}
	}
	bootstrapCommand(t, f.root, f.env, f.binary, f.args(bootstrap)...)
	if !reflect.DeepEqual(after, bootstrapState(t, f)) {
		t.Error("same release rerun is not byte/topology idempotent")
	}
}

func TestBootstrapReceiptCLI(t *testing.T) {
	release := bootstrapReleaseFixture(t)
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	reference := bootstrapSeed(t, &bootstrapRelease{old: release.next, source: repo})
	expected := bootstrapContentState(t, reference)
	t.Run("standalone_receipt", func(t *testing.T) { bootstrapStandaloneReceiptCases(t, release) })
	t.Run("parent_finalization", func(t *testing.T) {
		// A missing receipt has no remembered copy modes. Use an explicit all-copy
		// plan and an independently installed release reference for that plan.
		allCopy := bootstrapSeed(t, &bootstrapRelease{old: release.next, source: repo})
		if err := os.Remove(filepath.Join(allCopy.config, "install.json")); err != nil {
			t.Fatal(err)
		}
		args := []string{"install", "--from", repo, "--copy"}
		for _, name := range []string{"custom-a", "custom-b", "copy-a", "copy-b"} {
			args = append(args, "--home", filepath.Join(allCopy.home, name))
		}
		bootstrapCommand(t, allCopy.root, allCopy.env, allCopy.binary, args...)
		bootstrapCommand(t, allCopy.root, allCopy.env, allCopy.binary, "install", "--from", repo, "--target", "codex", "--target", "opencode", "--copy")
		copyExpected := bootstrapContentState(t, allCopy)
		for _, absent := range []bool{false, true} {
			for _, fault := range []bool{false, true} {
				t.Run(fmt.Sprintf("absent_%t_close_fault_%t", absent, fault), func(t *testing.T) {
					want := expected
					if absent {
						want = copyExpected
					}
					bootstrapFinalizationCase(t, release, want, absent, fault)
				})
			}
		}
	})
	for _, authority := range []string{"absent", "forged", "unprepared", "out_of_scope"} {
		t.Run("receipt_authority_"+authority, func(t *testing.T) { bootstrapAuthorityCase(t, release, repo, authority) })
	}
	for _, bootstrap := range []bool{false, true} {
		t.Run(fmt.Sprintf("converges_bootstrap_%t", bootstrap), func(t *testing.T) {
			f := bootstrapSeed(t, release)
			bootstrapAssertConvergence(t, f, release, expected, bootstrap)
		})
	}
	for _, repair := range []string{"edited_owned_artifact", "missing_owned_artifact", "both_native_groups_missing"} {
		for _, bootstrap := range []bool{false, true} {
			t.Run(fmt.Sprintf("repair_%s_bootstrap_%t", repair, bootstrap), func(t *testing.T) {
				f := bootstrapSeed(t, release)
				path := filepath.Join(f.home, ".codex", "agents", "machinery-fsm-author.toml")
				if repair == "edited_owned_artifact" {
					write(t, path, "externally modified target")
				} else if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if repair == "both_native_groups_missing" {
					if err := os.Remove(filepath.Join(f.home, ".config", "opencode", "plugins", "machinery.js")); err != nil {
						t.Fatal(err)
					}
				}
				bootstrapAssertConvergence(t, f, release, expected, bootstrap)
			})
		}
	}
	for _, missing := range []bool{false, true} {
		for _, bootstrap := range []bool{false, true} {
			name := fmt.Sprintf("later_target_failure_rolls_back_bootstrap_%t", bootstrap)
			if missing {
				name = "missing_prestate_" + name
			}
			t.Run(name, func(t *testing.T) {
				f := bootstrapSeed(t, release)
				missingPath := filepath.Join(f.home, ".codex", "agents", "machinery-fsm-author.toml")
				if missing {
					if err := os.Remove(missingPath); err != nil {
						t.Fatal(err)
					}
				}
				before := bootstrapState(t, f)
				release.broken.Store(true)
				defer release.broken.Store(false)
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, f.binary, f.args(bootstrap)...)
				cmd.Dir, cmd.Env = f.root, f.env
				out, err := cmd.CombinedOutput()
				if err == nil {
					t.Errorf("missing later OpenCode source unexpectedly succeeded: %s", out)
				}
				if !strings.Contains(string(out), "updated machinery binary") || !strings.Contains(string(out), "custom-a") || !strings.Contains(string(out), "installed Codex agents ->") {
					t.Errorf("did not prove binary and earlier placements changed before later failure: %s", out)
				}
				if !strings.Contains(string(out), "adapters/opencode/plugins/machinery.js") || !strings.Contains(string(out), "no such file") {
					t.Errorf("did not reach intended later source failure: %s", out)
				}
				if !reflect.DeepEqual(before, bootstrapState(t, f)) {
					t.Error("later target failure did not restore old binary, all homes/native targets, and receipt")
				}
				if missing {
					if _, err := os.Lstat(missingPath); !os.IsNotExist(err) {
						t.Errorf("original absence was not restored: %v", err)
					}
				}
			})
		}
	}
	for _, negative := range []string{"corrupt receipt", "unsafe receipt", "stale schema", "plugin discovery failure", "symlink native artifact", "non-directory native parent"} {
		t.Run(negative, func(t *testing.T) {
			f := bootstrapSeed(t, release)
			path := filepath.Join(f.config, "install.json")
			switch negative {
			case "corrupt receipt":
				if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "unsafe receipt":
				if err := os.Chmod(path, 0o666); err != nil {
					t.Fatal(err)
				}
			case "stale schema":
				r := f.receipt
				r.SchemaVersion = 999
				raw, _ := json.Marshal(r)
				if err := os.WriteFile(path, raw, 0o600); err != nil {
					t.Fatal(err)
				}
			case "plugin discovery failure":
				write(t, filepath.Join(f.home, ".claude", "plugins", "cache"), "not a directory")
			case "symlink native artifact":
				target := filepath.Join(f.home, ".codex", "agents", "machinery-fsm-author.toml")
				if err := os.Remove(target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(f.home, "unrelated", "sentinel"), target); err != nil {
					t.Fatal(err)
				}
			case "non-directory native parent":
				parent := filepath.Join(f.home, ".codex", "agents")
				if err := os.Rename(parent, filepath.Join(f.root, "parked-agents")); err != nil {
					t.Fatal(err)
				}
				write(t, parent, "unsafe non-directory parent")
			}
			before := bootstrapState(t, f)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, f.binary, f.args(true)...)
			cmd.Dir, cmd.Env = f.root, f.env
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Errorf("unsafe bootstrap unexpectedly succeeded: %s", out)
			} else if !strings.Contains(string(out), "receipt") && !strings.Contains(string(out), "plugin") && !strings.Contains(string(out), "target") && !strings.Contains(string(out), "artifact") && !strings.Contains(string(out), "digest") {
				t.Errorf("missing actionable diagnostic: %s", out)
			}
			if !reflect.DeepEqual(before, bootstrapState(t, f)) {
				t.Error("rejected bootstrap altered binary, recorded targets, receipt, or unrelated host files")
			}
		})
	}
	t.Run("interrupted download recovers real transaction", func(t *testing.T) {
		f := bootstrapSeed(t, release)
		before := bootstrapState(t, f)
		release.hold.Store(true)
		defer release.hold.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, f.binary, f.args(true)...)
		cmd.Dir, cmd.Env = f.root, f.env
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = cmd.Process.Kill() }()
		select {
		case <-release.requested:
		case <-ctx.Done():
			t.Fatal("update never reached real source download")
		}
		if err := cmd.Process.Kill(); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Wait(); err == nil {
			t.Fatal("interrupted update exited successfully")
		}
		if _, err := os.Stat(filepath.Join(f.config, installJournalDir, installJournalMetadata)); err != nil {
			t.Fatalf("interruption did not leave recovery journal: %v", err)
		}
		bootstrapCommand(t, f.root, f.env, f.binary, "version")
		if !reflect.DeepEqual(before, bootstrapState(t, f)) {
			t.Error("startup recovery did not restore complete pre-update state")
		}
		if _, err := os.Stat(filepath.Join(f.config, installJournalDir)); !os.IsNotExist(err) {
			t.Errorf("recovery journal remains: %v", err)
		}
	})
}

type bootstrapObserver func([]byte) (int, error)

func (observe bootstrapObserver) Write(p []byte) (int, error) { return observe(p) }

func bootstrapAssertRecorded(t *testing.T, want installReceipt, source string) {
	t.Helper()
	got, exists, err := loadReceipt()
	if err != nil || !exists {
		t.Errorf("successful recording not loadable: %v", err)
		return
	}
	want.SchemaVersion = receiptSchema
	want.HomeInstalls = append([]homeInstall(nil), want.HomeInstalls...)
	want.Targets = append([]targetInstall(nil), want.Targets...)
	want.HostPlugins = append([]string(nil), want.HostPlugins...)
	paths, err := receiptArtifactPaths(want) // fixed expected plan, not emitted membership
	if err != nil {
		t.Fatal(err)
	}
	want.Artifacts = nil
	for _, path := range paths {
		digest, err := artifactTreeDigest(path)
		if err != nil {
			t.Fatal(err)
		}
		want.Artifacts = append(want.Artifacts, receiptArtifact{Path: path, Digest: digest})
	}
	normalizeReceipt(&want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("recorded topology/full inventory/real digests differ: got %+v want %+v", got, want)
	}
	sourceContent := bootstrapResolvedHomes(t, []string{source})
	for _, group := range want.HomeInstalls {
		for i, home := range group.Homes {
			content := bootstrapResolvedHomes(t, []string{home})
			if len(content) != len(sourceContent) {
				t.Errorf("home release inventory differs from source: %s", home)
			}
			for path, digest := range sourceContent {
				if content[home+strings.TrimPrefix(path, source)] != digest {
					t.Errorf("home content differs from actual source: %s", path)
				}
			}
			for _, path := range homeInstallArtifactPaths([]string{home}) {
				info, err := os.Lstat(path)
				if err != nil || (info.Mode()&os.ModeSymlink != 0) != (i > 0 && !group.Copy) {
					t.Errorf("home copy/link topology differs: %s %v", path, err)
				}
				if i > 0 && !group.Copy {
					target, err := filepath.EvalSymlinks(path)
					canonical, canonicalErr := filepath.EvalSymlinks(group.Homes[0] + strings.TrimPrefix(path, home))
					if err != nil || canonicalErr != nil || target != canonical {
						t.Errorf("home link has wrong canonical destination: %s", path)
					}
				}
			}
		}
	}
}

func bootstrapResolvedHomes(t *testing.T, homes []string) map[string]string {
	t.Helper()
	result := map[string]string{}
	for _, artifact := range homeInstallArtifactPaths(homes) {
		resolved, err := filepath.EvalSymlinks(artifact)
		if err != nil {
			t.Fatal(err)
		}
		err = filepath.WalkDir(resolved, func(path string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				result[artifact+strings.TrimPrefix(path, resolved)] = fmt.Sprintf("%x", sha256.Sum256(bootstrapRead(t, path)))
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func bootstrapStandaloneReceiptCases(t *testing.T, release *bootstrapRelease) {
	wantFor := func(f bootstrapInstall) installReceipt {
		return installReceipt{HomeInstalls: []homeInstall{
			{Homes: []string{filepath.Join(f.home, "custom-a"), filepath.Join(f.home, "custom-b")}},
			{Homes: []string{filepath.Join(f.home, "copy-a"), filepath.Join(f.home, "copy-b")}, Copy: true},
		}, Targets: []targetInstall{{Target: "codex"}, {Target: "opencode", Copy: true}}, HostPlugins: []string{"claude", "codex"}}
	}
	unchanged := func(t *testing.T, before, after map[string]string, selected []string) {
		t.Helper()
		for path, value := range before {
			if path == "receipt" {
				continue
			}
			owned := false
			for _, home := range selected {
				owned = owned || strings.HasPrefix(path, home+string(os.PathSeparator))
			}
			if !owned && after[path] != value {
				t.Errorf("unselected content/topology changed: %s", path)
			}
		}
	}
	t.Run("supported_recording", func(t *testing.T) {
		f := bootstrapSeed(t, release)
		want := wantFor(f)
		t.Run("fresh_mixed_groups", func(t *testing.T) { bootstrapAssertRecorded(t, want, release.source) })
		t.Run("disjoint_additional_group", func(t *testing.T) {
			before := bootstrapState(t, f)
			homes := []string{filepath.Join(f.home, "new-a"), filepath.Join(f.home, "new-b")}
			bootstrapCommand(t, f.root, f.env, f.binary, "install", "--from", release.source, "--home", homes[0], "--home", homes[1])
			want.HomeInstalls = append(want.HomeInstalls, homeInstall{Homes: homes})
			bootstrapAssertRecorded(t, want, release.source)
			unchanged(t, before, bootstrapState(t, f), homes)
			for _, path := range homeInstallArtifactPaths([]string{homes[1]}) {
				info, err := os.Lstat(path)
				if err != nil || info.Mode()&os.ModeSymlink == 0 {
					t.Errorf("disjoint linked topology missing: %s %v", path, err)
				}
			}
		})
		t.Run("same_ordered_group_copy_change", func(t *testing.T) {
			homes := want.HomeInstalls[0].Homes
			before, contents := bootstrapState(t, f), bootstrapResolvedHomes(t, homes)
			bootstrapCommand(t, f.root, f.env, f.binary, "install", "--from", release.source, "--home", homes[0], "--home", homes[1], "--copy")
			want.HomeInstalls[0].Copy = true
			bootstrapAssertRecorded(t, want, release.source)
			unchanged(t, before, bootstrapState(t, f), homes)
			if !reflect.DeepEqual(contents, bootstrapResolvedHomes(t, homes)) {
				t.Error("supported copy change altered release content")
			}
			for _, path := range homeInstallArtifactPaths(homes) {
				info, err := os.Lstat(path)
				if err != nil || info.Mode()&os.ModeSymlink != 0 {
					t.Errorf("same-group copy placement is not independent: %s %v", path, err)
				}
			}
		})
	})
	for _, repeated := range []bool{false, true} {
		t.Run(fmt.Sprintf("reject_cross_group_repeated_%t_then_native", repeated), func(t *testing.T) {
			f := bootstrapSeed(t, release)
			want := wantFor(f)
			bootstrapAssertRecorded(t, want, release.source)
			before := bootstrapState(t, f)
			args := []string{"install", "--from", release.source, "--copy"}
			names := []string{"custom-a", "custom-b", "copy-a", "copy-b"}
			if repeated {
				names = []string{"custom-a", "copy-a"}
			}
			for _, name := range names {
				args = append(args, "--home", filepath.Join(f.home, name))
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, f.binary, args...)
			cmd.Dir, cmd.Env = f.root, f.env
			out, err := cmd.CombinedOutput()
			diagnostic := strings.ToLower(string(out))
			if err == nil || ctx.Err() != nil || !strings.Contains(string(out), filepath.Join(f.home, "copy-a")) ||
				!(strings.Contains(diagnostic, "overlap") || strings.Contains(diagnostic, "conflict") || strings.Contains(diagnostic, "repeat")) ||
				!(strings.Contains(diagnostic, "retry") || strings.Contains(diagnostic, "reconfigur") || strings.Contains(diagnostic, "nonconflicting")) {
				t.Errorf("expected actionable recorded-group conflict, got err=%v deadline=%v output=%s", err, ctx.Err(), out)
			}
			if !reflect.DeepEqual(before, bootstrapState(t, f)) {
				t.Error("rejected overlap did not restore exact installation and valid receipt")
			}
			// No manual receipt rewrite or artifact repair between conflict and next install.
			bootstrapCommand(t, f.root, f.env, f.binary, "install", "--from", release.source, "--target", "codex", "--target", "opencode", "--copy")
			want.Targets[0].Copy = true
			bootstrapAssertRecorded(t, want, release.source)
			unchanged(t, before, bootstrapState(t, f), []string{filepath.Join(f.home, ".agents"), filepath.Join(f.home, ".codex"), filepath.Join(f.home, ".config", "opencode")})
		})
	}
}

func bootstrapFinalizationCase(t *testing.T, release *bootstrapRelease, expected map[string]string, absent, fault bool) {
	t.Helper()
	f := bootstrapSeed(t, release)
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MACHINERY_INTERNAL_TEST_LOCK_ROOT", cache)
	receiptPath := filepath.Join(f.config, "install.json")
	opts := UpdateOptions{Version: "v9.9.2", Repo: "fixture/machinery", Executable: f.binary, SkipPlugins: true}
	wantReceipt := f.receipt
	wantReceipt.Artifacts = append([]receiptArtifact(nil), f.receipt.Artifacts...)
	children := 4 // two home groups, then the two distinct native copy groups
	if absent {
		if err := os.Remove(receiptPath); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"custom-a", "custom-b", "copy-a", "copy-b"} {
			opts.Homes = append(opts.Homes, filepath.Join(f.home, name))
		}
		opts.Targets, opts.Copy = []string{"codex", "opencode"}, true
		wantReceipt.HomeInstalls = []homeInstall{{Homes: opts.Homes, Copy: true}}
		wantReceipt.Targets = []targetInstall{{Target: "codex", Copy: true}, {Target: "opencode", Copy: true}}
		wantReceipt.HostPlugins = nil
		children = 2
	}
	before := bootstrapState(t, f)
	journalRoot := filepath.Join(f.config, installJournalDir)
	canonicalRoot, err := filepath.EvalSymlinks(f.root)
	if err != nil {
		t.Fatal(err)
	}
	canonical := func(path string) string { return bootstrapRootAlias(path, f.root, canonicalRoot) }
	checkPrepared := func() {
		journal, phase, err := loadInstallJournal(journalRoot)
		if err != nil || phase != "prepared" {
			t.Fatalf("finalization is not inside prepared uncommitted journal: %s %v", phase, err)
		}
		covered := map[string]bool{}
		for _, item := range journal.Items {
			covered[canonical(item.Target)] = true
		}
		paths := []string{f.binary, receiptPath}
		for _, artifact := range wantReceipt.Artifacts {
			paths = append(paths, artifact.Path)
		}
		for _, path := range paths {
			if !covered[canonical(path)] {
				t.Errorf("prepared journal omits selected artifact %s", path)
			}
		}
	}
	completed, publications := 0, 0
	unchanged, placements, desiredReady := true, false, false
	var output bytes.Buffer
	opts.Out = bootstrapObserver(func(p []byte) (int, error) {
		output.Write(p)
		if bytes.Contains(p, []byte("skill + agents ->")) || bytes.Contains(p, []byte("installed Codex agents ->")) || bytes.Contains(p, []byte("installed OpenCode agents + commands")) {
			completed++ // output is relayed only after the actual child returns
			if bootstrapReceiptState(t, receiptPath) != before["receipt"] {
				t.Errorf("child %d published receipt before parent finalization", completed)
				unchanged = false
			}
			if completed == children {
				checkPrepared()
				placements = reflect.DeepEqual(bootstrapContentState(t, f), expected) && bytes.Equal(bootstrapRead(t, f.binary), release.nextBytes)
				if !placements {
					t.Error("final child did not leave independently expected complete release placements")
				} else {
					wantReceipt.HomeInstalls = append([]homeInstall(nil), wantReceipt.HomeInstalls...)
					wantReceipt.Targets = append([]targetInstall(nil), wantReceipt.Targets...)
					wantReceipt.HostPlugins = append([]string(nil), wantReceipt.HostPlugins...)
					wantReceipt.SchemaVersion = receiptSchema
					for i := range wantReceipt.Artifacts {
						digest, err := artifactTreeDigest(wantReceipt.Artifacts[i].Path)
						if err != nil {
							t.Fatal(err)
						}
						wantReceipt.Artifacts[i].Digest = digest
					}
					normalizeReceipt(&wantReceipt)
					desiredReady = true
				}
			}
		}
		return len(p), nil
	})
	oldBase, oldAPI, oldClose := githubBase, apiBase, closeInstallFile
	githubBase, apiBase = release.endpoint, release.endpoint
	t.Cleanup(func() { githubBase, apiBase, closeInstallFile = oldBase, oldAPI, oldClose })
	var closeErr error
	closeInstallFile = func(file *os.File) error {
		if canonical(filepath.Dir(file.Name())) != canonical(filepath.Join(journalRoot, installJournalScratch)) || !strings.HasPrefix(filepath.Base(file.Name()), "receipt-") {
			return oldClose(file)
		}
		publications++
		if completed != children || !placements || !unchanged || !desiredReady {
			t.Error("parent receipt publication preceded complete children/placements/unchanged receipt")
			return oldClose(file)
		}
		checkPrepared()
		if bootstrapReceiptState(t, receiptPath) != before["receipt"] {
			t.Error("persisted receipt changed before parent receipt close")
		}
		want, err := json.MarshalIndent(wantReceipt, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(bootstrapRead(t, file.Name()), append(want, '\n')) {
			t.Error("parent final scratch receipt lacks exact normalized topology/inventory/current digests")
		}
		if err := oldClose(file); err != nil {
			t.Fatalf("first real receipt close failed: %v", err)
		}
		if fault {
			closeErr = file.Close()
			t.Logf("injected real final receipt close at %s: %v", file.Name(), closeErr)
			return closeErr
		}
		return nil
	}
	// Bound the real in-process parent too; no replacement runner is supplied.
	deadline := time.AfterFunc(90*time.Second, func() { panic("real parent Update exceeded 90-second operation bound") })
	defer deadline.Stop()
	_, updateErr := Update(opts)
	if !desiredReady {
		t.Error("independent complete desired receipt was not established after final child")
	}
	if completed != children || publications != 1 {
		t.Errorf("missing parent finalization boundary: children=%d/%d publications=%d; err=%v", completed, children, publications, updateErr)
	}
	if fault {
		if !errors.Is(closeErr, os.ErrClosed) || !errors.Is(updateErr, os.ErrClosed) {
			t.Errorf("real late publication close fault was not exercised/retained: injected=%v returned=%v", closeErr, updateErr)
		}
		if strings.Contains(output.String(), "machinery update complete:") || !reflect.DeepEqual(before, bootstrapState(t, f)) {
			t.Error("publication fault committed or failed exact binary/home/native/receipt rollback")
		}
	} else if updateErr != nil {
		t.Errorf("no-fault complete publication failed: %v", updateErr)
	} else {
		got, exists, err := loadReceipt()
		if err != nil || !exists || !reflect.DeepEqual(got, wantReceipt) || !strings.Contains(output.String(), "machinery update complete:") {
			t.Errorf("no-fault final receipt/commit differs from expected complete release: %v", err)
		}
	}
	if _, err := os.Lstat(journalRoot); !os.IsNotExist(err) {
		t.Errorf("finalization left journal behind: %v", err)
	}
	// Same test-only cache mapping as the real children: actual lock reacquisition.
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
}

func bootstrapAuthorityCase(t *testing.T, release *bootstrapRelease, source, authority string) {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	config := filepath.Join(root, "config")
	if err := os.MkdirAll(config, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("MACHINERY_CONFIG_DIR", config)
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	// Go test isolates lock storage by default; point that existing test-only
	// root at the real CLI's temporary cache so both contend on the same lock.
	t.Setenv("MACHINERY_INTERNAL_TEST_LOCK_ROOT", cache)
	env := os.Environ()
	bootstrapCommand(t, root, env, release.next, "install", "--from", source, "--home", home)
	receipt, exists, err := loadReceipt()
	if err != nil || !exists || len(receipt.HomeInstalls) != 1 || len(receipt.Artifacts) != 3 {
		t.Fatalf("standalone receipt missing/invalid: %+v %v", receipt, err)
	}
	for _, artifact := range receipt.Artifacts {
		digest, err := artifactTreeDigest(artifact.Path)
		if err != nil || digest != artifact.Digest {
			t.Fatalf("standalone receipt digest invalid: %s %v", artifact.Path, err)
		}
	}
	receiptPath := filepath.Join(config, "install.json")
	beforeReceipt := bootstrapRead(t, receiptPath)
	beforeHome, err := artifactTreeDigest(home)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := acquireInstallOperationLock()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := lock.Release(); err != nil {
			t.Error(err)
		}
		reopened, err := acquireInstallOperationLock()
		if err != nil {
			t.Errorf("parent lock not released: %v", err)
		} else if err := reopened.Release(); err != nil {
			t.Error(err)
		}
	})
	tx, err := beginArtifactTransaction(append(homeInstallArtifactPaths([]string{home}), receiptPath))
	if err != nil {
		t.Fatal(err)
	}
	journalRoot := tx.root
	t.Cleanup(func() {
		if err := tx.rollback(); err != nil {
			t.Error(err)
		}
		if _, err := os.Lstat(journalRoot); !os.IsNotExist(err) {
			t.Errorf("parent journal cleanup incomplete: %v", err)
		}
	})
	scope, err := installOperationScope()
	if err != nil {
		t.Fatal(err)
	}
	capability, cleanup, err := createInstallLockCapability(scope)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	encoded := capability.String()
	selected := home
	switch authority {
	case "absent":
		encoded = ""
	case "forged":
		encoded = "forged-parent-capability"
	case "out_of_scope":
		selected = filepath.Join(root, "outside-coverage")
	case "unprepared":
		marker := filepath.Join(tx.root, installJournalPrepared)
		raw := bootstrapRead(t, marker)
		if err := os.Remove(marker); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.WriteFile(marker, raw, 0o600); err != nil {
				t.Error(err)
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, release.next, "install", "--from", release.source, "--home", selected)
	cmd.Dir, cmd.Env = root, append(env, installLockCapabilityEnv+"="+encoded)
	out, err := cmd.CombinedOutput()
	if err == nil || ctx.Err() != nil {
		t.Errorf("unauthorized receipt participant: err=%v deadline=%v output=%s", err, ctx.Err(), out)
	}
	afterHome, err := artifactTreeDigest(home)
	if err != nil || beforeHome != afterHome || !bytes.Equal(beforeReceipt, bootstrapRead(t, receiptPath)) {
		t.Errorf("rejected child changed standalone content/receipt: %v", err)
	}
	if selected != home {
		if _, err := os.Lstat(selected); !os.IsNotExist(err) {
			t.Errorf("out-of-scope child created target: %v", err)
		}
	}
}
