// Package install places the machinery agent skill and role docs into agent
// homes, and removes them again. It is the single implementation behind the
// `machinery install` / `machinery uninstall` subcommands and the install.sh
// bootstrap: real files land in the first (canonical) home and the rest are
// symlinked to it, so there is exactly one copy on disk to update.
package install

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	defaultRepo = "RamXX/machinery"
	skillRel    = "skills/machinery" // path of the skill within a source tree
	agentsRel   = "agents"           // dir of the role docs within a source tree
)

// RoleDocs are the two synthesis subagent files shipped next to the skill.
var RoleDocs = []string{"machinery-fsm-author.md", "machinery-build-writer.md"}

var releaseTag = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// Base URLs, overridable in tests to point at a local httptest server.
var (
	githubBase = "https://github.com"
	apiBase    = "https://api.github.com"
)

// Options configures Install.
type Options struct {
	Homes   []string  // target agent homes; the first is canonical (real files)
	Targets []string  // optional first-class host adapters: claude, codex, opencode, all
	From    string    // local source dir (contains skills/ and agents/); skips download
	Copy    bool      // copy into every home instead of symlinking the non-canonical ones
	Version string    // release tag to fetch when From is empty; "", "latest", or a non-release tag -> newest release
	Repo    string    // source repo owner/name (default RamXX/machinery)
	Out     io.Writer // progress messages (nil -> discarded)
	Record  bool      // persist this successful topology for `machinery update`
}

// DefaultHomes is the canonical-first default target list: ~/.agents then ~/.claude.
func DefaultHomes() []string {
	home, _ := os.UserHomeDir()
	return []string{filepath.Join(home, ".agents"), filepath.Join(home, ".claude")}
}

// Install lays the skill + role docs into opts.Homes. The first home holds the
// real files; the rest are symlinked to it (or copied, with opts.Copy).
func Install(opts Options) error {
	out := opts.Out
	if out == nil {
		out = io.Discard
	}
	if len(opts.Targets) > 0 {
		if len(opts.Homes) > 0 {
			return fmt.Errorf("--home and --target cannot be combined")
		}
		src, cleanup, err := resolveInstallSource(opts, out)
		if err != nil {
			return err
		}
		defer cleanup()
		if err := installTargets(opts.Targets, src, opts.Copy, out); err != nil {
			return err
		}
		if opts.Record {
			return recordTargetInstall(opts.Targets, opts.Copy)
		}
		return nil
	}
	homes, err := absHomes(opts.Homes)
	if err != nil {
		return err
	}
	if len(homes) == 0 {
		if homes, err = absHomes(DefaultHomes()); err != nil {
			return err
		}
		// The Claude Code plugin ships the same skill + role docs; placing a
		// second copy into a home the plugin already serves would duplicate
		// them. Only the default list is filtered: an explicit --home wins.
		kept := homes[:0]
		for _, h := range homes {
			if pluginInstalled(h) {
				fmt.Fprintf(out, "skipping %s: the machinery Claude Code plugin already provides the skill + role docs there\n", h)
				continue
			}
			kept = append(kept, h)
		}
		homes = kept
		if len(homes) == 0 {
			return nil
		}
	}

	src, cleanup, err := resolveInstallSource(opts, out)
	if err != nil {
		return err
	}
	defer cleanup()

	canon := homes[0]
	if err := placeReal(canon, src, out); err != nil {
		return err
	}
	for _, home := range homes[1:] {
		if opts.Copy {
			if err := placeReal(home, src, out); err != nil {
				return err
			}
			continue
		}
		if err := placeLinks(home, canon, out); err != nil {
			return err
		}
	}
	if opts.Record {
		return recordHomeInstall(homes, opts.Copy)
	}
	return nil
}

func resolveInstallSource(opts Options, out io.Writer) (string, func(), error) {
	if opts.From != "" {
		src, err := filepath.Abs(opts.From)
		if err != nil {
			return "", func() {}, err
		}
		if err := validateSource(src); err != nil {
			return "", func() {}, err
		}
		return src, func() {}, nil
	}

	repo := opts.Repo
	if repo == "" {
		repo = defaultRepo
	}
	src, cleanup, err := fetchSource(repo, opts.Version, out)
	if err != nil {
		return "", func() {}, err
	}
	if err := validateSource(src); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return src, cleanup, nil
}

// Uninstall removes the skill and role docs from every home.
func Uninstall(homes []string, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	abs, err := absHomes(homes)
	if err != nil {
		return err
	}
	if len(abs) == 0 {
		if abs, err = absHomes(DefaultHomes()); err != nil {
			return err
		}
	}
	for _, home := range abs {
		if err := os.RemoveAll(filepath.Join(home, "skills", "machinery")); err != nil {
			return err
		}
		for _, d := range RoleDocs {
			if err := os.Remove(filepath.Join(home, "agents", d)); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		fmt.Fprintf(out, "removed machinery -> %s\n", home)
	}
	return nil
}

// pluginInstalled reports whether the machinery Claude Code plugin is cached
// under home (a ~/.claude-style config dir). The glob follows the plugin
// cache layout <home>/plugins/cache/<marketplace>/<plugin>; if that layout
// ever changes the worst case is a benign duplicate skill, exactly the
// pre-plugin behavior.
func pluginInstalled(home string) bool {
	m, _ := filepath.Glob(filepath.Join(home, "plugins", "cache", "*", "machinery"))
	return len(m) > 0
}

func absHomes(homes []string) ([]string, error) {
	out := make([]string, 0, len(homes))
	for _, h := range homes {
		if strings.TrimSpace(h) == "" {
			continue
		}
		a, err := filepath.Abs(h)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func validateSource(src string) error {
	if fi, err := os.Stat(filepath.Join(src, skillRel)); err != nil || !fi.IsDir() {
		return fmt.Errorf("source has no %s: %s", skillRel, src)
	}
	for _, d := range RoleDocs {
		if _, err := os.Stat(filepath.Join(src, agentsRel, d)); err != nil {
			return fmt.Errorf("source is missing role doc %s: %w", d, err)
		}
	}
	return nil
}

// placeReal copies the real skill + role docs into home.
func placeReal(home, src string, out io.Writer) error {
	if err := os.MkdirAll(filepath.Join(home, "skills"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(home, "agents"), 0o755); err != nil {
		return err
	}
	dst := filepath.Join(home, "skills", "machinery")
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	if err := copyTree(filepath.Join(src, skillRel), dst); err != nil {
		return err
	}
	for _, d := range RoleDocs {
		if err := copyFile(filepath.Join(src, agentsRel, d), filepath.Join(home, "agents", d)); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "installed skill + agents -> %s\n", home)
	return nil
}

// placeLinks symlinks home's skill + role docs to the canonical copy.
func placeLinks(home, canon string, out io.Writer) error {
	if err := os.MkdirAll(filepath.Join(home, "skills"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(home, "agents"), 0o755); err != nil {
		return err
	}
	dst := filepath.Join(home, "skills", "machinery")
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	if err := os.Symlink(filepath.Join(canon, "skills", "machinery"), dst); err != nil {
		return err
	}
	for _, d := range RoleDocs {
		link := filepath.Join(home, "agents", d)
		if err := os.Remove(link); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.Symlink(filepath.Join(canon, "agents", d), link); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "linked skill + agents -> %s (-> %s)\n", home, canon)
	return nil
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	// Replace an existing destination (in particular a symlink left by a prior
	// install) rather than following it and writing through to its target.
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// fetchSource downloads and extracts the source tarball for the resolved tag,
// returning the extracted top-level directory and a cleanup func.
func fetchSource(repo, version string, out io.Writer) (string, func(), error) {
	tag, err := resolveTag(repo, version)
	if err != nil {
		return "", nil, err
	}
	fmt.Fprintf(out, "fetching skill + agents from %s %s\n", repo, tag)
	tmp, err := os.MkdirTemp("", "machinery-src")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(tmp) }
	url := githubBase + "/" + repo + "/archive/refs/tags/" + tag + ".tar.gz"
	archive := filepath.Join(tmp, "src.tar.gz")
	if err := download(url, archive); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("download source tarball for %s: %w", tag, err)
	}
	dest := filepath.Join(tmp, "extracted")
	if err := extractTarGz(archive, dest); err != nil {
		cleanup()
		return "", nil, err
	}
	top, err := singleChildDir(dest)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return top, cleanup, nil
}

// resolveTag returns an explicit release tag as-is, otherwise the newest release.
func resolveTag(repo, version string) (string, error) {
	v := strings.TrimSpace(version)
	if releaseTag.MatchString(v) {
		return v, nil
	}
	// A blank, "latest", or non-release version (e.g. a -dev binary) resolves
	// to the newest published release.
	tmp, err := os.CreateTemp("", "machinery-rel-*.json")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	tmp.Close()
	if err := download(apiBase+"/repos/"+repo+"/releases/latest", tmp.Name()); err != nil {
		return "", fmt.Errorf("resolve latest release for %s: %w", repo, err)
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		return "", err
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(data, &rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("no published release found for %s", repo)
	}
	return rel.TagName, nil
}

func download(url, dst string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// extractTarGz unpacks a gzipped tar into dest, taking only regular files and
// directories and rejecting any entry that would escape dest.
func extractTarGz(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	root := filepath.Clean(dest) + string(os.PathSeparator)
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.Clean("/"+hdr.Name))
		if target != filepath.Clean(dest) && !strings.HasPrefix(target, root) {
			continue // guard against tar-slip
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode).Perm())
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil { //nolint:gosec // our own release tarball
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

func singleChildDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no top-level directory in extracted archive %s", dir)
}
