// Package install places the machinery agent skill and role docs into agent
// homes, and removes them again. It is the single implementation behind the
// `machinery install` / `machinery uninstall` subcommands and the install.sh
// bootstrap: real files land in the first (canonical) home and the rest are
// symlinked to it, so there is exactly one copy on disk to update.
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
	"net/http"
	"os"
	archivepath "path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/filelock"
	"github.com/RamXX/machinery/internal/portablepath"
	machversion "github.com/RamXX/machinery/internal/version"
)

const (
	defaultRepo = "RamXX/machinery"
	skillRel    = "skills/machinery" // path of the skill within a source tree
	agentsRel   = "agents"           // dir of the role docs within a source tree

	releaseAPIMaxBytes           = int64(1 << 20)
	releaseChecksumsMaxBytes     = int64(4 << 20)
	releaseSourceArchiveMaxBytes = int64(128 << 20)
	releaseBinaryMaxBytes        = int64(256 << 20)
	releaseSourceTreeMaxFiles    = 10_000
	releaseSourceTreeMaxBytes    = int64(512 << 20)
	releaseSourceMemberMaxBytes  = int64(64 << 20)
)

type downloadPolicy struct {
	label    string
	maxBytes int64
}

var (
	releaseAPIDownload       = downloadPolicy{label: "release API response", maxBytes: releaseAPIMaxBytes}
	releaseChecksumsDownload = downloadPolicy{label: "release checksums", maxBytes: releaseChecksumsMaxBytes}
	releaseSourceDownload    = downloadPolicy{label: "source archive", maxBytes: releaseSourceArchiveMaxBytes}
	releaseBinaryDownload    = downloadPolicy{label: "update binary", maxBytes: releaseBinaryMaxBytes}
)

// RoleDocs are the two synthesis subagent files shipped next to the skill.
var RoleDocs = []string{"machinery-fsm-author.md", "machinery-build-writer.md"}

var releaseTag = regexp.MustCompile(`^v\d+\.\d+\.\d+(?:-[0-9A-Za-z][0-9A-Za-z.-]*)?(?:\+[0-9A-Za-z][0-9A-Za-z.-]*)?$`)

// Base URLs, overridable in tests to point at a local httptest server.
var (
	githubBase = "https://github.com"
	apiBase    = "https://api.github.com"
)

// Options configures Install.
type Options struct {
	Homes   []string // target agent homes; the first is canonical (real files)
	Targets []string // optional first-class host adapters: claude, codex, opencode, all
	From    string   // local source dir (contains skills/ and agents/); skips download
	Copy    bool     // copy into every home instead of symlinking the non-canonical ones
	Version string   // release tag to fetch when From is empty; "", "latest", or a non-release tag -> newest release
	// VersionExplicit records that the user asked for Version (--version)
	// rather than it defaulting to the binary's own version. An explicit tag
	// with no published release fails loudly; the binary's own version falls
	// back to the newest release (a local build ahead of its tag).
	VersionExplicit bool
	Repo            string    // source repo owner/name (default RamXX/machinery)
	Out             io.Writer // progress messages (nil -> discarded)
	Record          bool      // persist this successful topology for `machinery update`

	// beforeCommit is a deterministic failure-injection hook for transaction
	// tests. Production callers leave it nil.
	beforeCommit func(string) error
}

// DefaultHomes is the canonical-first default target list: ~/.agents then ~/.claude.
// It fails rather than turning a missing user home into process-relative paths.
func DefaultHomes() ([]string, error) {
	home, err := userHomeDir()
	if err != nil {
		return nil, err
	}
	return []string{filepath.Join(home, ".agents"), filepath.Join(home, ".claude")}, nil
}

func userHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home for installs: %w", err)
	}
	if strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("resolve user home for installs: empty home")
	}
	return home, nil
}

// Install lays the skill + role docs into opts.Homes. The first home holds the
// real files; the rest are symlinked to it (or copied, with opts.Copy).
func Install(opts Options) (retErr error) {
	operationLock, err := acquireInstallOperationLock()
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, operationLock.Release()) }()
	return installLocked(opts)
}

func installLocked(opts Options) (retErr error) {
	out := opts.Out
	if out == nil {
		out = io.Discard
	}
	if len(opts.Targets) > 0 {
		if len(opts.Homes) > 0 {
			return fmt.Errorf("--home and --target cannot be combined")
		}
		paths, err := targetInstallArtifactPaths(opts.Targets)
		if err != nil {
			return err
		}
		if opts.Record {
			receipt, err := installationReceiptPath()
			if err != nil {
				return err
			}
			paths = append(paths, receipt)
		}
		tx, err := beginArtifactTransaction(paths)
		if err != nil {
			return err
		}
		source, err := resolveInstallSource(opts, out)
		if err != nil {
			return rollbackInstallTransaction(tx, err)
		}
		defer func() { retErr = errors.Join(retErr, source.cleanup()) }()
		if err := installTargets(opts.Targets, source.path, opts.Copy, out, opts.beforeCommit); err != nil {
			return rollbackInstallTransaction(tx, err)
		}
		if opts.Record {
			if err := recordTargetInstallLocked(opts.Targets, opts.Copy); err != nil {
				return rollbackInstallTransaction(tx, err)
			}
		}
		if err := source.verify(); err != nil {
			return rollbackInstallTransaction(tx, err)
		}
		if err := tx.commit(); err != nil {
			return fmt.Errorf("commit install transaction: %w", err)
		}
		return nil
	}
	homes, err := absHomes(opts.Homes)
	if err != nil {
		return err
	}
	if len(homes) == 0 {
		defaults, derr := DefaultHomes()
		if derr != nil {
			return derr
		}
		if homes, err = absHomes(defaults); err != nil {
			return err
		}
		// The Claude Code plugin ships the same skill + role docs; placing a
		// second copy into a home the plugin already serves would duplicate
		// them. Only the default list is filtered: an explicit --home wins.
		kept := homes[:0]
		for _, h := range homes {
			installed, pluginErr := pluginInstalled(h)
			if pluginErr != nil {
				return fmt.Errorf("discover machinery plugin under %s: %w", h, pluginErr)
			}
			if installed {
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

	canon := homes[0]
	paths := homeInstallArtifactPaths(homes)
	if opts.Record {
		receipt, err := installationReceiptPath()
		if err != nil {
			return err
		}
		paths = append(paths, receipt)
	}
	tx, err := beginArtifactTransaction(paths)
	if err != nil {
		return err
	}
	source, err := resolveInstallSource(opts, out)
	if err != nil {
		return rollbackInstallTransaction(tx, err)
	}
	defer func() { retErr = errors.Join(retErr, source.cleanup()) }()
	before := func(home string) error {
		if opts.beforeCommit != nil {
			return opts.beforeCommit(home)
		}
		return nil
	}
	if err := before(canon); err != nil {
		return rollbackInstallTransaction(tx, err)
	}
	if err := placeReal(canon, source.path, out); err != nil {
		return rollbackInstallTransaction(tx, err)
	}
	for _, home := range homes[1:] {
		if err := before(home); err != nil {
			return rollbackInstallTransaction(tx, err)
		}
		if opts.Copy {
			if err := placeReal(home, source.path, out); err != nil {
				return rollbackInstallTransaction(tx, err)
			}
			continue
		}
		if err := placeLinks(home, canon, out); err != nil {
			return rollbackInstallTransaction(tx, err)
		}
	}
	if opts.Record {
		if err := recordHomeInstallLocked(homes, opts.Copy); err != nil {
			return rollbackInstallTransaction(tx, err)
		}
	}
	if err := source.verify(); err != nil {
		return rollbackInstallTransaction(tx, err)
	}
	if err := tx.commit(); err != nil {
		return fmt.Errorf("commit install transaction: %w", err)
	}
	return nil
}

func rollbackInstallTransaction(tx *artifactTransaction, cause error) error {
	if rollbackErr := tx.rollback(); rollbackErr != nil {
		return errors.Join(fmt.Errorf("install failed: %w", cause), fmt.Errorf("rollback install transaction: %w", rollbackErr))
	}
	return cause
}

func resolveInstallSource(opts Options, out io.Writer) (*resolvedSource, error) {
	if opts.From != "" {
		src, err := filepath.Abs(opts.From)
		if err != nil {
			return nil, err
		}
		snapshot, err := acquireInstallSourceSnapshot(src, opts.Targets)
		if err != nil {
			return nil, err
		}
		if err := validateSource(snapshot.materialized); err != nil {
			return nil, errors.Join(err, snapshot.cleanup())
		}
		if set, err := parseTargetsOptional(opts.Targets); err != nil {
			return nil, errors.Join(err, snapshot.cleanup())
		} else if err := validateTargetSource(snapshot.materialized, set); err != nil {
			return nil, errors.Join(err, snapshot.cleanup())
		}
		return &resolvedSource{path: snapshot.materialized, verify: snapshot.verifyUnchanged, cleanup: snapshot.cleanup}, nil
	}

	repo := opts.Repo
	if repo == "" {
		repo = defaultRepo
	}
	src, cleanup, err := fetchSource(repo, opts.Version, opts.VersionExplicit, out)
	if err != nil {
		return nil, err
	}
	snapshot, err := acquireInstallSourceSnapshot(src, opts.Targets)
	if err != nil {
		return nil, errors.Join(err, cleanup())
	}
	if err := validateSource(snapshot.materialized); err != nil {
		return nil, errors.Join(err, snapshot.cleanup(), cleanup())
	}
	if set, err := parseTargetsOptional(opts.Targets); err != nil {
		return nil, errors.Join(err, snapshot.cleanup(), cleanup())
	} else if err := validateTargetSource(snapshot.materialized, set); err != nil {
		return nil, errors.Join(err, snapshot.cleanup(), cleanup())
	}
	return &resolvedSource{
		path:   snapshot.materialized,
		verify: snapshot.verifyUnchanged,
		cleanup: func() error {
			return errors.Join(snapshot.cleanup(), cleanup())
		},
	}, nil
}

// Uninstall removes the skill and role docs from every home.
func Uninstall(homes []string, out io.Writer) (retErr error) {
	if out == nil {
		out = io.Discard
	}
	abs, err := absHomes(homes)
	if err != nil {
		return err
	}
	if len(abs) == 0 {
		defaults, derr := DefaultHomes()
		if derr != nil {
			return derr
		}
		if abs, err = absHomes(defaults); err != nil {
			return err
		}
	}
	operationLock, err := acquireInstallOperationLock()
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, operationLock.Release()) }()

	receiptPath, err := installationReceiptPath()
	if err != nil {
		return err
	}
	receipt, receiptExists, err := loadReceipt()
	if err != nil {
		return err
	}
	if receiptExists {
		abs = expandCanonicalHomeGroups(receipt, abs)
	}
	paths := append(homeInstallArtifactPaths(abs), receiptPath)
	tx, err := beginArtifactTransaction(paths)
	if err != nil {
		return fmt.Errorf("snapshot uninstall transaction: %w", err)
	}
	for _, path := range homeInstallArtifactPaths(abs) {
		if err := removeInstallArtifact(path); err != nil {
			return rollbackUninstallTransaction(tx, fmt.Errorf("remove install artifact %s: %w", path, err))
		}
	}
	if receiptExists && forgetHomesFromReceipt(&receipt, abs) {
		if err := saveReceipt(receipt); err != nil {
			return rollbackUninstallTransaction(tx, fmt.Errorf("update installation receipt: %w", err))
		}
	}
	if err := tx.commit(); err != nil {
		return fmt.Errorf("commit uninstall transaction: %w", err)
	}
	for _, home := range abs {
		fmt.Fprintf(out, "removed machinery -> %s\n", home)
	}
	return nil
}

var removeInstallArtifact = durableRemoveAll

func rollbackUninstallTransaction(tx *artifactTransaction, cause error) error {
	if rollbackErr := tx.rollback(); rollbackErr != nil {
		return errors.Join(cause, fmt.Errorf("rollback uninstall transaction: %w", rollbackErr))
	}
	return cause
}

// pluginInstalled reports whether the machinery Claude Code plugin is cached
// under home (a ~/.claude-style config dir), following the plugin cache
// layout <home>/plugins/cache/<marketplace>/<plugin>/<version>. It lists the cache
// directory instead of filepath.Glob: a glob metacharacter in $HOME ("[",
// "*", "?") turns the pattern into a character class and silently defeats
// detection. Discovery fails closed on unreadable, ambiguous, stale, or
// partial cache entries: skipping a direct install is safe only when the
// current plugin's complete owned artifact inventory is positively proven.
var (
	readPluginCache                  = readPluginCacheBoundedPath
	installDiscoveryLstat            = os.Lstat
	installDiscoveryRead             = os.ReadFile
	cachedPluginAfterOpen            = func(string) {}
	cachedPluginBeforeFinalInventory = func(string) {}
	cachedPluginBeforeFinalTopology  = func(string) {}
	cachedPluginAfterWitnessMember   = func(int, string) {}
	cachedPluginAfterCommitTopology  = func(int, string) {}
)

const pluginCacheMaxEntries = 10_000

func readPluginCacheBoundedPath(directory string) ([]os.DirEntry, error) {
	dir, err := os.Open(directory)
	if err != nil {
		return nil, err
	}
	entries, readErr := readInstallDirBounded(dir, pluginCacheMaxEntries, "plugin cache directory")
	closeErr := closeInstallFile(dir)
	return entries, errors.Join(readErr, closeErr)
}

func pluginInstalled(home string) (bool, error) {
	cache := filepath.Join(home, "plugins", "cache")
	cacheInfo, err := installDiscoveryLstat(cache)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("inspect plugin cache %s: %w", cache, err)
	}
	if cacheInfo.Mode()&os.ModeSymlink != 0 || !cacheInfo.IsDir() {
		return false, fmt.Errorf("plugin cache %s is not a real directory", cache)
	}
	initialTopology, err := capturePluginCacheTopology(cache)
	if err != nil {
		return false, fmt.Errorf("snapshot plugin cache topology %s: %w", cache, err)
	}
	entries, err := readPluginCache(cache)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("enumerate plugin cache %s: %w", cache, err)
	}
	found := ""
	for _, e := range entries {
		marketplace := filepath.Join(cache, e.Name())
		entryInfo, statErr := installDiscoveryLstat(marketplace)
		if statErr != nil {
			return false, fmt.Errorf("inspect plugin cache entry %s: %w", marketplace, statErr)
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 || !entryInfo.IsDir() {
			return false, fmt.Errorf("plugin cache entry %s is not a real directory", marketplace)
		}
		pluginContainer := filepath.Join(marketplace, "machinery")
		pluginInfo, pluginErr := installDiscoveryLstat(pluginContainer)
		if pluginErr != nil {
			if os.IsNotExist(pluginErr) {
				continue
			}
			return false, fmt.Errorf("inspect cached machinery plugin container %s: %w", pluginContainer, pluginErr)
		}
		if pluginInfo.Mode()&os.ModeSymlink != 0 || !pluginInfo.IsDir() {
			return false, fmt.Errorf("cached machinery plugin container %s is not a real directory", pluginContainer)
		}
		versions, versionListErr := readPluginCache(pluginContainer)
		if versionListErr != nil {
			return false, fmt.Errorf("enumerate cached machinery plugin versions %s: %w", pluginContainer, versionListErr)
		}
		if len(versions) == 0 {
			return false, fmt.Errorf("cached machinery plugin container %s has no version directories", pluginContainer)
		}
		for _, versionEntry := range versions {
			versionPath := filepath.Join(pluginContainer, versionEntry.Name())
			info, statErr := installDiscoveryLstat(versionPath)
			if statErr != nil {
				return false, fmt.Errorf("inspect cached machinery plugin version %s: %w", versionPath, statErr)
			}
			if !releaseTag.MatchString("v"+versionEntry.Name()) || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return false, fmt.Errorf("cached machinery plugin version %s is not a real, valid version directory", versionPath)
			}
		}
		plugin := filepath.Join(pluginContainer, strings.TrimPrefix(machversion.Version, "v"))
		versionInfo, versionErr := installDiscoveryLstat(plugin)
		if versionErr != nil {
			if os.IsNotExist(versionErr) {
				continue
			}
			return false, fmt.Errorf("inspect cached machinery plugin version %s: %w", plugin, versionErr)
		}
		if versionInfo.Mode()&os.ModeSymlink != 0 || !versionInfo.IsDir() {
			return false, fmt.Errorf("cached machinery plugin version %s is not a real directory", plugin)
		}
		if found != "" {
			return false, fmt.Errorf("ambiguous machinery plugin caches %s and %s", found, plugin)
		}
		found = plugin
	}
	if found == "" {
		return false, nil
	}
	validatedInventory := map[string]cachedPluginInventoryEntry{}
	if err := validateCachedMachineryPlugin(found, &validatedInventory); err != nil {
		return false, fmt.Errorf("validate cached machinery plugin %s: %w", found, err)
	}
	cachedPluginBeforeFinalTopology(cache)
	finalTopology, err := capturePluginCacheTopology(cache)
	if err != nil {
		return false, fmt.Errorf("revalidate plugin cache topology %s: %w", cache, err)
	}
	if err := comparePluginCacheTopologies(initialTopology, finalTopology); err != nil {
		return false, err
	}
	if err := validateCachedPluginSuccessWitness(found, cache, validatedInventory, finalTopology); err != nil {
		return false, fmt.Errorf("validate cached machinery plugin success witness %s: %w", found, err)
	}
	return true, nil
}

type pluginCacheTopology struct {
	parentInfo     os.FileInfo
	parentChangeID string
	entries        map[string]pluginCacheTopologyEntry
}

type pluginCacheTopologyEntry struct {
	info     os.FileInfo
	depth    int
	changeID string
}

func capturePluginCacheTopology(cache string) (snapshot pluginCacheTopology, retErr error) {
	return capturePluginCacheTopologyWithHook(cache, nil)
}

func capturePluginCacheTopologyWithHook(cache string, afterDirectory func(int, string)) (snapshot pluginCacheTopology, retErr error) {
	parent := filepath.Dir(cache)
	parentInfo, err := installDiscoveryLstat(parent)
	if err != nil {
		return snapshot, fmt.Errorf("inspect plugin cache parent %s: %w", parent, err)
	}
	if parentInfo.Mode()&os.ModeSymlink != 0 || !parentInfo.IsDir() {
		return snapshot, fmt.Errorf("plugin cache parent %s is not a real directory", parent)
	}
	parentRoot, err := os.OpenRoot(parent)
	if err != nil {
		return snapshot, fmt.Errorf("retain plugin cache parent %s: %w", parent, err)
	}
	defer func() { retErr = errors.Join(retErr, parentRoot.Close()) }()
	retainedParentInfo, err := parentRoot.Stat(".")
	if err != nil || !sameInstallTopologyEntry(parentInfo, retainedParentInfo) {
		return snapshot, errors.Join(fmt.Errorf("plugin cache parent %s changed while being retained", parent), err)
	}

	cacheName := filepath.Base(cache)
	cacheInfo, err := parentRoot.Lstat(cacheName)
	if err != nil {
		return snapshot, fmt.Errorf("inspect retained plugin cache %s: %w", cache, err)
	}
	if cacheInfo.Mode()&os.ModeSymlink != 0 || !cacheInfo.IsDir() {
		return snapshot, fmt.Errorf("plugin cache %s is not a real directory", cache)
	}
	cacheRoot, err := parentRoot.OpenRoot(cacheName)
	if err != nil {
		return snapshot, fmt.Errorf("retain plugin cache %s: %w", cache, err)
	}
	defer func() { retErr = errors.Join(retErr, cacheRoot.Close()) }()
	retainedCacheInfo, err := cacheRoot.Stat(".")
	if err != nil || !sameInstallTopologyEntry(cacheInfo, retainedCacheInfo) {
		return snapshot, errors.Join(fmt.Errorf("plugin cache %s changed while being retained", cache), err)
	}

	first := map[string]pluginCacheTopologyEntry{}
	if err := walkPluginCacheTopology(cacheRoot, ".", 0, 1, first, afterDirectory); err != nil {
		return snapshot, err
	}
	second := map[string]pluginCacheTopologyEntry{}
	if err := walkPluginCacheTopology(cacheRoot, ".", 0, 2, second, afterDirectory); err != nil {
		return snapshot, err
	}
	if err := comparePluginCacheTopologyEntries(first, second); err != nil {
		return snapshot, fmt.Errorf("plugin cache topology changed while being snapshotted: %w", err)
	}
	if err := revalidatePluginCacheTopologyCensus(cacheRoot, second); err != nil {
		return snapshot, fmt.Errorf("plugin cache topology changed during enumeration: %w", err)
	}

	currentParentInfo, parentErr := installDiscoveryLstat(parent)
	currentCacheInfo, cacheErr := installDiscoveryLstat(cache)
	if parentErr != nil || !sameInstallTopologyEntry(parentInfo, currentParentInfo) {
		return snapshot, errors.Join(fmt.Errorf("plugin cache parent %s changed while being snapshotted", parent), parentErr)
	}
	if cacheErr != nil || !sameInstallTopologyEntry(cacheInfo, currentCacheInfo) {
		return snapshot, errors.Join(fmt.Errorf("plugin cache %s changed while being snapshotted", cache), cacheErr)
	}
	return pluginCacheTopology{parentInfo: retainedParentInfo, parentChangeID: installFileChangeID(retainedParentInfo), entries: second}, nil
}

func walkPluginCacheTopology(root *os.Root, directory string, depth, pass int, inventory map[string]pluginCacheTopologyEntry, afterDirectory func(int, string)) error {
	if err := validateInstallTraversalDepth(depth, directory); err != nil {
		return err
	}
	info, err := root.Lstat(directory)
	if err != nil {
		return fmt.Errorf("inspect plugin cache topology member %s: %w", directory, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("plugin cache topology member %s is not a real directory", directory)
	}
	if len(inventory) >= pluginCacheMaxEntries {
		return fmt.Errorf("plugin cache topology exceeds %d-entry limit", pluginCacheMaxEntries)
	}
	dir, err := root.Open(directory)
	if err != nil {
		return fmt.Errorf("retain plugin cache topology member %s: %w", directory, err)
	}
	openedInfo, statErr := dir.Stat()
	entries, readErr := readInstallDirBounded(dir, pluginCacheMaxEntries-len(inventory)-1, "plugin cache topology")
	closeErr := dir.Close()
	if statErr != nil || !sameInstallTopologyEntry(info, openedInfo) {
		return errors.Join(fmt.Errorf("plugin cache topology member %s changed while being retained", directory), statErr, readErr, closeErr)
	}
	if readErr != nil || closeErr != nil {
		return errors.Join(fmt.Errorf("enumerate plugin cache topology member %s", directory), readErr, closeErr)
	}
	inventory[directory] = pluginCacheTopologyEntry{info: openedInfo, depth: depth, changeID: installFileChangeID(openedInfo)}
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		childInfo, err := root.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect plugin cache topology member %s: %w", path, err)
		}
		if childInfo.Mode()&os.ModeSymlink != 0 || !childInfo.IsDir() {
			return fmt.Errorf("plugin cache topology member %s is not a real directory", path)
		}
		childDepth := depth + 1
		if err := validateInstallTraversalDepth(childDepth, path); err != nil {
			return err
		}
		if childDepth < 3 {
			if err := walkPluginCacheTopology(root, path, childDepth, pass, inventory, afterDirectory); err != nil {
				return err
			}
			continue
		}
		inventory[path] = pluginCacheTopologyEntry{info: childInfo, depth: childDepth, changeID: installFileChangeID(childInfo)}
	}
	if afterDirectory != nil {
		afterDirectory(pass, directory)
	}
	return nil
}

func revalidatePluginCacheTopologyCensus(root *os.Root, inventory map[string]pluginCacheTopologyEntry) error {
	directChildren := make(map[string]map[string]pluginCacheTopologyEntry)
	for path, entry := range inventory {
		if path == "." {
			continue
		}
		parent := filepath.Dir(path)
		if directChildren[parent] == nil {
			directChildren[parent] = map[string]pluginCacheTopologyEntry{}
		}
		directChildren[parent][path] = entry
	}
	directories := make([]string, 0, len(inventory))
	for path, entry := range inventory {
		if entry.depth < 3 {
			directories = append(directories, path)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(directories)))
	for _, directory := range directories {
		expectedDirectory := inventory[directory]
		before, err := root.Lstat(directory)
		if err != nil {
			return fmt.Errorf("inspect topology directory %s before census: %w", directory, err)
		}
		if err := comparePluginCacheTopologyEntry(directory, expectedDirectory, pluginCacheTopologyEntry{
			info: before, depth: expectedDirectory.depth, changeID: installFileChangeID(before),
		}); err != nil {
			return err
		}
		dir, err := root.Open(directory)
		if err != nil {
			return fmt.Errorf("retain topology directory %s for census: %w", directory, err)
		}
		openedInfo, statErr := dir.Stat()
		entries, readErr := readInstallDirBounded(dir, len(directChildren[directory]), "plugin cache topology census")
		closeErr := dir.Close()
		if statErr != nil || readErr != nil || closeErr != nil {
			return errors.Join(fmt.Errorf("census topology directory %s", directory), statErr, readErr, closeErr)
		}
		if err := comparePluginCacheTopologyEntry(directory, expectedDirectory, pluginCacheTopologyEntry{
			info: openedInfo, depth: expectedDirectory.depth, changeID: installFileChangeID(openedInfo),
		}); err != nil {
			return err
		}
		expectedChildren := directChildren[directory]
		if len(entries) != len(expectedChildren) {
			return fmt.Errorf("topology directory %s child count changed from %d to %d", directory, len(expectedChildren), len(entries))
		}
		for _, entry := range entries {
			path := filepath.Join(directory, entry.Name())
			expected, ok := expectedChildren[path]
			if !ok {
				return fmt.Errorf("topology directory %s gained unexpected child %s", directory, path)
			}
			info, err := root.Lstat(path)
			if err != nil {
				return fmt.Errorf("inspect topology child %s during census: %w", path, err)
			}
			if err := comparePluginCacheTopologyEntry(path, expected, pluginCacheTopologyEntry{
				info: info, depth: expected.depth, changeID: installFileChangeID(info),
			}); err != nil {
				return err
			}
		}
		after, err := root.Lstat(directory)
		if err != nil {
			return fmt.Errorf("inspect topology directory %s after census: %w", directory, err)
		}
		if err := comparePluginCacheTopologyEntry(directory, expectedDirectory, pluginCacheTopologyEntry{
			info: after, depth: expectedDirectory.depth, changeID: installFileChangeID(after),
		}); err != nil {
			return err
		}
	}
	return nil
}

func comparePluginCacheTopologies(initial, final pluginCacheTopology) error {
	if !sameInstallTopologyEntry(initial.parentInfo, final.parentInfo) {
		return fmt.Errorf("plugin cache topology changed while being validated: cache parent was replaced or modified")
	}
	if initial.parentChangeID != "" && final.parentChangeID != "" && initial.parentChangeID != final.parentChangeID {
		return fmt.Errorf("plugin cache topology changed while being validated: cache parent change identity changed")
	}
	if err := comparePluginCacheTopologyEntries(initial.entries, final.entries); err != nil {
		return fmt.Errorf("plugin cache topology changed while being validated: %w", err)
	}
	return nil
}

func comparePluginCacheTopologyEntries(initial, final map[string]pluginCacheTopologyEntry) error {
	if len(initial) != len(final) {
		return fmt.Errorf("entry count changed from %d to %d", len(initial), len(final))
	}
	paths := make([]string, 0, len(initial))
	for path := range initial {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		before := initial[path]
		after, ok := final[path]
		if !ok {
			return fmt.Errorf("%s was removed", path)
		}
		if err := comparePluginCacheTopologyEntry(path, before, after); err != nil {
			return err
		}
	}
	return nil
}

func comparePluginCacheTopologyEntry(path string, before, after pluginCacheTopologyEntry) error {
	if before.depth != after.depth || !sameInstallTopologyEntry(before.info, after.info) {
		return fmt.Errorf("%s was replaced or modified", path)
	}
	if before.changeID != "" && after.changeID != "" && before.changeID != after.changeID {
		return fmt.Errorf("%s change identity changed", path)
	}
	return nil
}

func sameInstallTopologyEntry(before, after os.FileInfo) bool {
	return before != nil && after != nil && os.SameFile(before, after) && before.Mode() == after.Mode() &&
		before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}

func validateCachedMachineryPlugin(root string, validatedInventory *map[string]cachedPluginInventoryEntry) (retErr error) {
	rootInfo, err := installDiscoveryLstat(root)
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return fmt.Errorf("cached plugin root %s is not a real directory", root)
	}
	retained, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("retain cached plugin root: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, retained.Close()) }()
	retainedInfo, err := retained.Stat(".")
	if err != nil || !os.SameFile(rootInfo, retainedInfo) || retainedInfo.Mode() != rootInfo.Mode() {
		return errors.Join(fmt.Errorf("cached plugin root changed while being retained"), err)
	}
	for _, directory := range []string{
		".claude-plugin",
		"agents",
		"hooks",
		"skills",
		filepath.Join("skills", "machinery"),
		filepath.Join("skills", "machinery", "references"),
		filepath.Join("skills", "machinery", "tools"),
	} {
		info, err := retained.Lstat(directory)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("required member %s is not a real directory", directory)
		}
	}
	initialInventory, err := validateCachedPluginInventory(retained)
	if err != nil {
		return err
	}
	manifest := filepath.Join(".claude-plugin", "plugin.json")
	manifestRaw, err := readCachedPluginFileRoot(retained, manifest, 128<<10)
	if err != nil {
		return err
	}
	var identity map[string]json.RawMessage
	if err := decodePluginInventory(string(manifestRaw), &identity); err != nil {
		return fmt.Errorf("decode canonical plugin manifest: %w", err)
	}
	var name, pluginVersion string
	if raw, ok := identity["name"]; !ok || json.Unmarshal(raw, &name) != nil || name != "machinery" {
		return fmt.Errorf("plugin manifest name is not machinery")
	}
	if raw, ok := identity["version"]; !ok || json.Unmarshal(raw, &pluginVersion) != nil || !releaseTag.MatchString("v"+strings.TrimPrefix(pluginVersion, "v")) {
		return fmt.Errorf("plugin manifest version is missing or invalid")
	}
	if strings.TrimPrefix(pluginVersion, "v") != strings.TrimPrefix(machversion.Version, "v") {
		return fmt.Errorf("cached machinery plugin version %s does not match running machinery %s; run 'claude plugin update machinery@machinery'", pluginVersion, machversion.Version)
	}
	skill := filepath.Join("skills", "machinery", "SKILL.md")
	skillRaw, err := readCachedPluginFileRoot(retained, skill, 4<<20)
	if err != nil {
		return err
	}
	skillText := strings.ReplaceAll(string(skillRaw), "\r\n", "\n")
	if len(skillRaw) < 1024 || !strings.HasPrefix(skillText, "---\nname: machinery\nmetadata:\n") || !strings.Contains(skillText, "\ndescription: >\n") || !strings.Contains(skillText, "\n---\n\n# machinery\n") {
		return fmt.Errorf("cached SKILL.md is truncated or does not match the machinery skill schema")
	}
	wantSkillVersion := strings.TrimPrefix(machversion.Version, "v")
	if !strings.Contains(skillText, "\nmetadata:\n  version: \""+wantSkillVersion+"\"\n") {
		return fmt.Errorf("cached SKILL.md metadata version does not match running machinery %s", machversion.Version)
	}
	for _, relative := range []string{
		"references/archaeology-classification.md", "references/build-md-template.md", "references/c4-standalone.md",
		"references/execution-packets.md", "references/rebuild-guide.md", "references/surface-ledger.md",
		"references/target-surfaces.md", "references/verification-evidence.md", "references/xstate-format.md",
		"tools/README.md", "tools/tlc.sh", "tools/verify_formal.sh",
	} {
		if _, err := readCachedPluginFileRoot(retained, filepath.Join("skills", "machinery", relative), 4<<20); err != nil {
			return err
		}
	}
	for _, role := range RoleDocs {
		raw, err := readCachedPluginFileRoot(retained, filepath.Join(agentsRel, role), 4<<20)
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(role, filepath.Ext(role))
		text := strings.ReplaceAll(string(raw), "\r\n", "\n")
		if len(raw) < 1024 || !strings.HasPrefix(text, "---\nname: "+name+"\ndescription: >\n") || !strings.Contains(text, "\n---\n") {
			return fmt.Errorf("cached role %s is truncated or does not match its canonical schema", role)
		}
	}
	hooksRaw, err := readCachedPluginFileRoot(retained, filepath.Join("hooks", "hooks.json"), 128<<10)
	if err != nil {
		return err
	}
	if err := validateCachedHookManifest(hooksRaw); err != nil {
		return err
	}
	shimPath := filepath.Join("hooks", "machinery-hook.sh")
	shimRaw, err := readCachedPluginFileRoot(retained, shimPath, 128<<10)
	if err != nil {
		return err
	}
	shimInfo, err := retained.Lstat(shimPath)
	if err != nil {
		return err
	}
	if runtime.GOOS != "windows" && shimInfo.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("cached hook shim %s is not executable", shimPath)
	}
	shimText := strings.ReplaceAll(string(shimRaw), "\r\n", "\n")
	if len(shimRaw) < 1024 || !strings.HasPrefix(shimText, "#!/bin/sh\n# machinery Claude Code/Codex plugin:") || !strings.Contains(shimText, `"$bin" hook --root "$root"`) {
		return fmt.Errorf("cached hook shim is truncated or does not match its canonical schema")
	}
	if digest := fmt.Sprintf("%x", sha256.Sum256(shimRaw)); digest != canonicalCachedHookShimSHA256 {
		return fmt.Errorf("cached hook shim has non-canonical sha256 %s", digest)
	}
	cachedPluginBeforeFinalInventory(root)
	finalInventory, err := validateCachedPluginInventory(retained)
	if err != nil {
		return fmt.Errorf("revalidate cached plugin inventory: %w", err)
	}
	if err := compareCachedPluginInventories(initialInventory, finalInventory); err != nil {
		return err
	}
	currentRoot, err := installDiscoveryLstat(root)
	if err != nil || !os.SameFile(rootInfo, currentRoot) || currentRoot.Mode()&os.ModeSymlink != 0 || !currentRoot.IsDir() ||
		currentRoot.Mode() != rootInfo.Mode() || currentRoot.Size() != rootInfo.Size() || !currentRoot.ModTime().Equal(rootInfo.ModTime()) {
		return errors.Join(fmt.Errorf("cached plugin root changed while being validated"), err)
	}
	*validatedInventory = finalInventory
	return nil
}

type cachedPluginInventoryEntry struct {
	info      os.FileInfo
	digest    [sha256.Size]byte
	directory bool
	changeID  string
}

func validateCachedPluginInventory(root *os.Root) (map[string]cachedPluginInventoryEntry, error) {
	return validateCachedPluginInventoryWithHook(root, nil)
}

func validateCachedPluginInventoryWithHook(root *os.Root, afterMember func(string)) (map[string]cachedPluginInventoryEntry, error) {
	expected := map[string]bool{
		filepath.Join(".claude-plugin", "plugin.json"):   true,
		filepath.Join("hooks", "hooks.json"):             true,
		filepath.Join("hooks", "machinery-hook.sh"):      true,
		filepath.Join("skills", "machinery", "SKILL.md"): true,
	}
	for _, role := range RoleDocs {
		expected[filepath.Join(agentsRel, role)] = true
	}
	for _, rel := range []string{
		"references/archaeology-classification.md", "references/build-md-template.md", "references/c4-standalone.md",
		"references/execution-packets.md", "references/rebuild-guide.md", "references/surface-ledger.md",
		"references/target-surfaces.md", "references/verification-evidence.md", "references/xstate-format.md",
		"tools/README.md", "tools/tlc.sh", "tools/verify_formal.sh",
	} {
		expected[filepath.Join("skills", "machinery", filepath.FromSlash(rel))] = true
	}
	roots := []string{".claude-plugin", agentsRel, "hooks", filepath.Join("skills", "machinery")}
	expectedDirectories := map[string]bool{}
	for _, directory := range roots {
		expectedDirectories[directory] = true
	}
	for path := range expected {
		for parent := filepath.Dir(path); parent != "."; parent = filepath.Dir(parent) {
			expectedDirectories[parent] = true
		}
	}
	inventory := map[string]cachedPluginInventoryEntry{}
	seen := map[string]bool{}
	for _, directory := range roots {
		if err := walkCachedPluginInventory(root, directory, expected, expectedDirectories, seen, inventory, afterMember, installRelativeTraversalDepth(directory)); err != nil {
			return nil, err
		}
		for path := range expected {
			if (path == directory || strings.HasPrefix(path, directory+string(filepath.Separator))) && !seen[path] {
				return nil, fmt.Errorf("cached plugin inventory is missing %s", path)
			}
		}
	}
	return inventory, nil
}

func walkCachedPluginInventory(root *os.Root, directory string, expected, expectedDirectories, seen map[string]bool, inventory map[string]cachedPluginInventoryEntry, afterMember func(string), depth int) error {
	if err := validateInstallTraversalDepth(depth, directory); err != nil {
		return err
	}
	info, err := root.Lstat(directory)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("cached plugin inventory member %s is not a real directory", directory)
	}
	if !expectedDirectories[directory] {
		return fmt.Errorf("cached plugin inventory has unexpected directory %s", directory)
	}
	inventory[directory] = cachedPluginInventoryEntry{info: info, directory: true, changeID: installFileChangeID(info)}
	dir, err := root.Open(directory)
	if err != nil {
		return err
	}
	entries, readErr := readInstallDirBounded(dir, len(expected)+len(expectedDirectories)-len(inventory), "cached plugin inventory")
	closeErr := dir.Close()
	if readErr != nil || closeErr != nil {
		return errors.Join(readErr, closeErr)
	}
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		info, err := root.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("cached plugin inventory member %s is a symlink", path)
		}
		if info.IsDir() {
			if err := walkCachedPluginInventory(root, path, expected, expectedDirectories, seen, inventory, afterMember, depth+1); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("cached plugin inventory member %s is not regular", path)
		}
		if !expected[path] {
			return fmt.Errorf("cached plugin inventory has unexpected member %s", path)
		}
		raw, err := readCachedPluginFileRoot(root, path, 4<<20)
		if err != nil {
			return err
		}
		finalInfo, err := root.Lstat(path)
		if err != nil {
			return err
		}
		inventory[path] = cachedPluginInventoryEntry{info: finalInfo, digest: sha256.Sum256(raw), changeID: installFileChangeID(finalInfo)}
		seen[path] = true
		if afterMember != nil {
			afterMember(path)
		}
	}
	return nil
}

func compareCachedPluginInventories(initial, final map[string]cachedPluginInventoryEntry) error {
	if len(initial) != len(final) {
		return fmt.Errorf("cached plugin inventory changed while being validated: entry count changed from %d to %d", len(initial), len(final))
	}
	paths := make([]string, 0, len(initial))
	for path := range initial {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		before := initial[path]
		after, ok := final[path]
		if !ok {
			return fmt.Errorf("cached plugin inventory changed while being validated: %s was removed", path)
		}
		if before.directory != after.directory || !os.SameFile(before.info, after.info) || before.info.Mode() != after.info.Mode() ||
			before.info.Size() != after.info.Size() || !before.info.ModTime().Equal(after.info.ModTime()) || before.digest != after.digest {
			return fmt.Errorf("cached plugin inventory changed while being validated: %s was replaced or modified", path)
		}
		if before.changeID != "" && after.changeID != "" && before.changeID != after.changeID {
			return fmt.Errorf("cached plugin inventory changed while being validated: %s change identity changed", path)
		}
	}
	return nil
}

func validateCachedPluginSuccessWitness(root, cache string, expected map[string]cachedPluginInventoryEntry, expectedTopology pluginCacheTopology) (retErr error) {
	rootInfo, err := installDiscoveryLstat(root)
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return fmt.Errorf("cached plugin root %s is not a real directory", root)
	}
	retained, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("retain cached plugin root for success witness: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, retained.Close()) }()
	retainedInfo, err := retained.Stat(".")
	if err != nil || !sameInstallTopologyEntry(rootInfo, retainedInfo) {
		return errors.Join(fmt.Errorf("cached plugin root changed while retaining success witness"), err)
	}

	prior := expected
	const witnessPasses = 3
	for pass := 1; pass <= witnessPasses; pass++ {
		beforeTopology, err := capturePluginCacheTopology(cache)
		if err != nil {
			return fmt.Errorf("success witness pass %d topology before selected inventory: %w", pass, err)
		}
		if err := comparePluginCacheTopologies(expectedTopology, beforeTopology); err != nil {
			return fmt.Errorf("success witness pass %d topology before selected inventory: %w", pass, err)
		}
		current, err := validateCachedPluginInventoryWithHook(retained, func(path string) {
			cachedPluginAfterWitnessMember(pass, path)
		})
		if err != nil {
			return fmt.Errorf("success witness pass %d: %w", pass, err)
		}
		if err := compareCachedPluginInventories(expected, current); err != nil {
			return fmt.Errorf("success witness pass %d differs from validated inventory: %w", pass, err)
		}
		if err := compareCachedPluginInventories(prior, current); err != nil {
			return fmt.Errorf("success witness pass %d is not stable: %w", pass, err)
		}
		afterTopology, err := capturePluginCacheTopology(cache)
		if err != nil {
			return fmt.Errorf("success witness pass %d topology after selected inventory: %w", pass, err)
		}
		if err := comparePluginCacheTopologies(expectedTopology, afterTopology); err != nil {
			return fmt.Errorf("success witness pass %d topology after selected inventory: %w", pass, err)
		}
		if err := comparePluginCacheTopologies(beforeTopology, afterTopology); err != nil {
			return fmt.Errorf("success witness pass %d topology was not stable around selected inventory: %w", pass, err)
		}
		prior = current
	}
	if err := revalidateCachedPluginInventoryMetadata(retained, prior); err != nil {
		return fmt.Errorf("commit-bound inventory metadata witness: %w", err)
	}
	commitTopology, err := capturePluginCacheTopologyWithHook(cache, cachedPluginAfterCommitTopology)
	if err != nil {
		return fmt.Errorf("commit-bound topology witness: %w", err)
	}
	if err := comparePluginCacheTopologies(expectedTopology, commitTopology); err != nil {
		return fmt.Errorf("commit-bound topology witness: %w", err)
	}
	currentRoot, err := installDiscoveryLstat(root)
	if err != nil || !sameInstallTopologyEntry(rootInfo, currentRoot) {
		return errors.Join(fmt.Errorf("cached plugin root changed before success"), err)
	}
	return nil
}

func revalidateCachedPluginInventoryMetadata(root *os.Root, inventory map[string]cachedPluginInventoryEntry) error {
	paths := make([]string, 0, len(inventory))
	for path := range inventory {
		paths = append(paths, path)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	for _, path := range paths {
		expected := inventory[path]
		info, err := root.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect %s: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || expected.directory != info.IsDir() || !os.SameFile(expected.info, info) ||
			expected.info.Mode() != info.Mode() || expected.info.Size() != info.Size() || !expected.info.ModTime().Equal(info.ModTime()) {
			return fmt.Errorf("cached plugin inventory changed before success: %s was replaced or modified", path)
		}
		changeID := installFileChangeID(info)
		if expected.changeID != "" && changeID != "" && expected.changeID != changeID {
			return fmt.Errorf("cached plugin inventory changed before success: %s change identity changed", path)
		}
	}
	return nil
}

func installFileChangeID(info os.FileInfo) string {
	if info == nil || info.Sys() == nil {
		return ""
	}
	value := reflect.ValueOf(info.Sys())
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return ""
	}
	for _, name := range []string{"Ctim", "Ctimespec"} {
		field := value.FieldByName(name)
		if field.IsValid() && field.Kind() == reflect.Struct {
			sec := field.FieldByName("Sec")
			nsec := field.FieldByName("Nsec")
			if sec.IsValid() && nsec.IsValid() && sec.CanInt() && nsec.CanInt() {
				return fmt.Sprintf("%d:%d", sec.Int(), nsec.Int())
			}
		}
	}
	ctime := value.FieldByName("Ctime")
	ctimeNsec := value.FieldByName("Ctimensec")
	if ctime.IsValid() && ctimeNsec.IsValid() && ctime.CanInt() && ctimeNsec.CanInt() {
		return fmt.Sprintf("%d:%d", ctime.Int(), ctimeNsec.Int())
	}
	return ""
}

const canonicalCachedHookShimSHA256 = "3dc3e14ca878325a35125b85bc2db7a5d1a1cc792fdee12e8602bf1ab279e90b"

func validateCachedHookManifest(raw []byte) error {
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return fmt.Errorf("decode canonical hook manifest: %w", err)
	}
	var manifest cachedHookManifest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return fmt.Errorf("decode canonical hook manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode canonical hook manifest: trailing JSON content")
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
		return fmt.Errorf("cached hook inventory is incomplete or has unexpected events")
	}
	for _, want := range expected {
		bindings, ok := manifest.Hooks[want.event]
		if !ok || len(bindings) != 1 || bindings[0].Matcher != want.matcher || len(bindings[0].Hooks) != 1 {
			return fmt.Errorf("cached hook event %s does not have the canonical matcher and single command", want.event)
		}
		command := bindings[0].Hooks[0]
		if command.Type != "command" || command.Command != "${CLAUDE_PLUGIN_ROOT}/hooks/machinery-hook.sh" || command.Timeout != want.timeout {
			return fmt.Errorf("cached hook event %s does not wire the canonical shim contract", want.event)
		}
		if command.Async != nil {
			return fmt.Errorf("cached hook event %s must use the synchronous topology", want.event)
		}
	}
	return nil
}

type cachedHookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
	Async   *bool  `json:"async,omitempty"`
}

type cachedHookBinding struct {
	Matcher string              `json:"matcher"`
	Hooks   []cachedHookCommand `json:"hooks"`
}

type cachedHookManifest struct {
	Description string                         `json:"description"`
	Hooks       map[string][]cachedHookBinding `json:"hooks"`
}

func readCachedPluginFileRoot(root *os.Root, path string, limit int64) ([]byte, error) {
	info, err := root.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("required member %s is not a real regular file", path)
	}
	if info.Size() <= 0 || info.Size() > limit {
		return nil, fmt.Errorf("required member %s has invalid size %d", path, info.Size())
	}
	file, err := root.Open(path)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.Join(fmt.Errorf("required member %s changed before being read", path), err, closeInstallFile(file))
	}
	cachedPluginAfterOpen(path)
	raw, readErr := io.ReadAll(io.LimitReader(file, limit+1))
	after, statErr := file.Stat()
	closeErr := closeInstallFile(file)
	pathAfter, pathErr := root.Lstat(path)
	if readErr != nil || statErr != nil || closeErr != nil || pathErr != nil {
		return nil, errors.Join(readErr, statErr, closeErr, pathErr)
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("required member %s exceeds size limit %d", path, limit)
	}
	if int64(len(raw)) != info.Size() || !os.SameFile(info, after) || !os.SameFile(info, pathAfter) || after.Mode() != info.Mode() || !after.ModTime().Equal(info.ModTime()) {
		return nil, fmt.Errorf("required member %s changed while being read", path)
	}
	return raw, nil
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
		a = filepath.Clean(a)
		compare := a
		if resolved, rerr := evalInstallPath(a); rerr == nil {
			compare = resolved
		} else if !os.IsNotExist(rerr) {
			return nil, fmt.Errorf("resolve install home %s: %w", h, rerr)
		}
		for _, prior := range out {
			priorCompare, _ := evalInstallPath(prior)
			if priorCompare == "" {
				priorCompare = prior
			}
			if sameOrNestedPath(priorCompare, compare) {
				return nil, fmt.Errorf("install homes overlap or resolve to the same path: %s and %s", prior, a)
			}
		}
		out = append(out, a)
	}
	return out, nil
}

func evalInstallPath(p string) (string, error) {
	cur := filepath.Clean(p)
	var tail []string
	for {
		resolved, err := filepath.EvalSymlinks(cur)
		if err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return resolved, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", err
		}
		tail = append(tail, filepath.Base(cur))
		cur = parent
	}
}

func sameOrNestedPath(a, b string) bool {
	aID, aErr := installPathIdentity(a)
	bID, bErr := installPathIdentity(b)
	if aErr == nil && bErr == nil {
		a, b = aID, bID
	}
	rel, err := filepath.Rel(a, b)
	if err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))) {
		return true
	}
	rel, err = filepath.Rel(b, a)
	return err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))))
}

func installPathIdentity(path string) (string, error) {
	resolved, err := evalInstallPath(path)
	if err != nil {
		return "", err
	}
	return filelock.ScopeIdentity(resolved)
}

// installArtifactPathIdentity resolves aliases in the containing directory but
// deliberately does not follow the final path component. A managed symlink is
// an artifact in its own right; treating it as its referent would collapse the
// canonical file and link into one transaction target.
func installArtifactPathIdentity(path string) (string, error) {
	resolved, err := installArtifactResolvedPath(path)
	if err != nil {
		return "", err
	}
	// The deliberately nonexistent path segment prevents ScopeIdentity from
	// following a managed symlink in the final component while still applying
	// the platform's case/volume normalization.
	parent := filepath.Dir(resolved)
	identity, err := filelock.ScopeIdentity(filepath.Join(parent, ".machinery-no-follow-identity", filepath.Base(resolved)))
	if err != nil {
		return "", err
	}
	return identity, nil
}

func installArtifactResolvedPath(path string) (string, error) {
	resolved, err := installArtifactAccessPath(path)
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		resolved = strings.ToLower(resolved)
	}
	return resolved, nil
}

func installArtifactAccessPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent, err := evalInstallPath(filepath.Dir(abs))
	if err != nil {
		return "", err
	}
	resolved := filepath.Clean(filepath.Join(parent, filepath.Base(abs)))
	return resolved, nil
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
	if err := stageAndCommit(home, func(stage string) error { return buildRealStage(stage, src) }); err != nil {
		return err
	}
	fmt.Fprintf(out, "installed skill + agents -> %s\n", home)
	return nil
}

// placeLinks symlinks home's skill + role docs to the canonical copy.
func placeLinks(home, canon string, out io.Writer) error {
	if err := stageAndCommit(home, func(stage string) error { return buildLinkStage(stage, canon) }); err != nil {
		return err
	}
	fmt.Fprintf(out, "linked skill + agents -> %s (-> %s)\n", home, canon)
	return nil
}

func buildRealStage(stage, src string) error {
	if err := copyTree(filepath.Join(src, skillRel), filepath.Join(stage, "skills", "machinery")); err != nil {
		return err
	}
	for _, d := range RoleDocs {
		if err := copyFile(filepath.Join(src, agentsRel, d), filepath.Join(stage, "agents", d)); err != nil {
			return err
		}
	}
	return nil
}

func buildLinkStage(stage, canon string) error {
	dst := filepath.Join(stage, "skills", "machinery")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Symlink(filepath.Join(canon, "skills", "machinery"), dst); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(stage, "agents"), 0o755); err != nil {
		return err
	}
	for _, d := range RoleDocs {
		if err := os.Symlink(filepath.Join(canon, "agents", d), filepath.Join(stage, "agents", d)); err != nil {
			return err
		}
	}
	return nil
}

// stageAndCommit prepares the complete skill/role set before changing any
// installed path. Its scratch tree lives inside the durable outer journal, so
// a process death cannot strand stage or backup directories beside a home.
func stageAndCommit(home string, build func(stage string) error) (retErr error) {
	if err := durableMkdirAll(home); err != nil {
		return err
	}
	stage, cleanupStage, err := installScratchDir("home-stage")
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, cleanupStage()) }()
	if err := build(stage); err != nil {
		return err
	}
	rels := []string{filepath.Join("skills", "machinery")}
	for _, d := range RoleDocs {
		rels = append(rels, filepath.Join("agents", d))
	}
	for _, rel := range rels {
		dst := filepath.Join(home, rel)
		src := filepath.Join(stage, rel)
		if err := durableMkdirAll(filepath.Dir(dst)); err != nil {
			return err
		}
		if err := stageInstallEntryNoReplace(src, dst, installStagePublish); err != nil {
			return fmt.Errorf("commit install artifact %s: %w", rel, err)
		}
	}
	return nil
}

func copyTree(src, dst string) error {
	budget := &installArtifactBudget{}
	return walkInstallTreeBounded(src, installArtifactMaxEntries, func(path string, info os.FileInfo) error {
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			if budget.entries >= installArtifactMaxEntries {
				return fmt.Errorf("copy tree exceeds %d-entry limit", installArtifactMaxEntries)
			}
			budget.entries++
			return os.MkdirAll(target, 0o755)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("copy tree source %s contains unsupported entry %s", src, info.Mode().Type())
		}
		return copyFileWithBudget(path, target, budget)
	})
}

func copyFile(src, dst string) (retErr error) {
	return copyFileWithBudget(src, dst, &installArtifactBudget{})
}

func copyFileWithBudget(src, dst string, budget *installArtifactBudget) (retErr error) {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > installArtifactMaxFileBytes {
		return fmt.Errorf("copy source %s must be a bounded regular non-symlink file", src)
	}
	if budget.entries >= installArtifactMaxEntries || info.Size() > installArtifactMaxTotalBytes-budget.bytes {
		return fmt.Errorf("copy source %s exceeds tree bounds", src)
	}
	budget.entries++
	if err := durableMkdirAll(filepath.Dir(dst)); err != nil {
		return err
	}
	if fi, err := os.Lstat(dst); err == nil && fi.IsDir() {
		return fmt.Errorf("destination is a directory: %s", dst)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, closeInstallFile(in)) }()
	opened, err := in.Stat()
	if err != nil || !sameInstallArtifactInfo(info, opened) {
		return errors.Join(err, fmt.Errorf("copy source %s changed while opening", src))
	}
	out, cleanup, err := installScratchFile(filepath.Dir(dst), "copy")
	if err != nil {
		return err
	}
	tmp := out.Name()
	defer func() { retErr = errors.Join(retErr, cleanup()) }()
	written, copyErr := io.Copy(out, io.LimitReader(in, info.Size()+1))
	heldAfter, heldErr := in.Stat()
	liveAfter, liveErr := os.Lstat(src)
	if err := errors.Join(copyErr, heldErr, liveErr); err != nil {
		return errors.Join(err, closeInstallFile(out))
	}
	if written != info.Size() || !sameInstallArtifactInfo(info, heldAfter) || !sameInstallArtifactInfo(info, liveAfter) {
		return errors.Join(fmt.Errorf("copy source %s changed while copying", src), closeInstallFile(out))
	}
	budget.bytes += written
	if err := out.Chmod(info.Mode().Perm()); err != nil {
		return errors.Join(err, closeInstallFile(out))
	}
	if err := out.Sync(); err != nil {
		return errors.Join(err, closeInstallFile(out))
	}
	if err := closeInstallFile(out); err != nil {
		return err
	}
	return renameReplace(tmp, dst)
}

// fetchSource downloads and extracts the checksummed source release asset,
// returning the extracted top-level directory and a cleanup func. explicit
// says the version was requested by the user rather than defaulted from the
// binary (see resolveTag).
func fetchSource(repo, version string, explicit bool, out io.Writer) (string, func() error, error) {
	if out == nil {
		out = io.Discard
	}
	tag, err := resolveTag(repo, version, explicit)
	if err != nil {
		return "", nil, err
	}
	fmt.Fprintf(out, "fetching skill + agents from %s %s\n", repo, tag)
	tmp, cleanup, err := installScratchDir("source")
	if err != nil {
		return "", nil, err
	}
	const asset = "machinery-source.tar.gz"
	base := githubBase + "/" + repo + "/releases/download/" + tag
	archive := filepath.Join(tmp, asset)
	checksums := filepath.Join(tmp, "checksums-sha256.txt")
	if err := download(base+"/"+asset, archive, releaseSourceDownload); err != nil {
		return "", nil, errors.Join(fmt.Errorf("download checksummed source asset for %s: %w", tag, err), cleanup())
	}
	if err := download(base+"/checksums-sha256.txt", checksums, releaseChecksumsDownload); err != nil {
		return "", nil, errors.Join(fmt.Errorf("release %s has no checksums-sha256.txt for source verification: %w", tag, err), cleanup())
	}
	want, err := checksumForAsset(checksums, asset)
	if err != nil {
		return "", nil, errors.Join(err, cleanup())
	}
	got, err := hashDownloadedFile(archive, releaseSourceDownload)
	if err != nil {
		return "", nil, errors.Join(err, cleanup())
	}
	if got != want {
		return "", nil, errors.Join(fmt.Errorf("checksum mismatch for %s (want %s, got %s)", asset, want, got), cleanup())
	}
	dest := filepath.Join(tmp, "extracted")
	if err := extractTarGz(archive, dest); err != nil {
		return "", nil, errors.Join(err, cleanup())
	}
	top, err := singleChildDir(dest)
	if err != nil {
		return "", nil, errors.Join(err, cleanup())
	}
	return top, cleanup, nil
}

// resolveTag maps a requested version to a release tag. A well-formed release
// tag the user asked for explicitly is returned as-is: an explicit request is
// never substituted, so a missing release fails loudly downstream. The same
// tag arriving implicitly (the binary's own version, no --version flag) is
// used when its release exists and otherwise falls back to the newest release,
// so a locally built binary ahead of its tag still installs. Anything else
// (blank, "latest", a non-release string) resolves to the newest release.
func resolveTag(repo, version string, explicit bool) (tag string, retErr error) {
	v := strings.TrimSpace(version)
	if releaseTag.MatchString(v) {
		exists, err := releaseExists(repo, v)
		if err != nil {
			return "", err
		}
		if exists {
			return v, nil
		}
		if explicit {
			return "", fmt.Errorf("release %s does not exist in %s", v, repo)
		}
	}
	tmp, cleanup, err := installScratchFile(os.TempDir(), "release")
	if err != nil {
		return "", err
	}
	defer func() { retErr = errors.Join(retErr, cleanup()) }()
	if err := closeInstallFile(tmp); err != nil {
		return "", err
	}
	if err := download(apiBase+"/repos/"+repo+"/releases/latest", tmp.Name(), releaseAPIDownload); err != nil {
		return "", fmt.Errorf("resolve latest release for %s: %w", repo, err)
	}
	data, err := readDownloadedFile(tmp.Name(), releaseAPIDownload)
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

// releaseExists reports whether the repo has a published release for the tag.
// Only a 404 means absent; transport failures and other HTTP statuses remain
// errors so an outage can never be mistaken for an unpublished version.
func releaseExists(repo, tag string) (exists bool, retErr error) {
	tmp, cleanup, err := installScratchFile(os.TempDir(), "release-tag")
	if err != nil {
		return false, err
	}
	defer func() { retErr = errors.Join(retErr, cleanup()) }()
	if err := closeInstallFile(tmp); err != nil {
		return false, err
	}
	err = download(apiBase+"/repos/"+repo+"/releases/tags/"+tag, tmp.Name(), releaseAPIDownload)
	if err == nil {
		return true, nil
	}
	var statusErr *httpStatusError
	if errors.As(err, &statusErr) && statusErr.Code == http.StatusNotFound {
		return false, nil
	}
	return false, fmt.Errorf("resolve release %s for %s: %w", tag, repo, err)
}

type httpStatusError struct {
	URL    string
	Code   int
	Status string
}

func (e *httpStatusError) Error() string { return fmt.Sprintf("GET %s: %s", e.URL, e.Status) }

func validateDownloadContentLength(length int64, policy downloadPolicy) error {
	if policy.maxBytes <= 0 {
		return fmt.Errorf("%s has invalid configured byte bound %d", policy.label, policy.maxBytes)
	}
	if length == -1 {
		// HTTP/2 and HTTP/3 frame response bodies without Transfer-Encoding and
		// legitimately omit Content-Length. The read below remains bounded by
		// maxBytes+1 and by the request context deadline.
		return nil
	}
	if length < -1 {
		return fmt.Errorf("%s reports invalid negative Content-Length %d", policy.label, length)
	}
	if length > policy.maxBytes {
		return fmt.Errorf("%s Content-Length %d exceeds %d-byte bound", policy.label, length, policy.maxBytes)
	}
	return nil
}

func download(url, dst string, policy downloadPolicy) (retErr error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: request failed: %w", policy.label, err)
	}
	defer func() { retErr = errors.Join(retErr, resp.Body.Close()) }()
	if resp.StatusCode != http.StatusOK {
		return &httpStatusError{URL: url, Code: resp.StatusCode, Status: resp.Status}
	}
	if err := validateDownloadContentLength(resp.ContentLength, policy); err != nil {
		return err
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(f, io.LimitReader(resp.Body, policy.maxBytes+1))
	if copyErr != nil {
		return errors.Join(fmt.Errorf("%s stream read failed: %w", policy.label, copyErr), closeInstallFile(f))
	}
	if written > policy.maxBytes {
		return errors.Join(fmt.Errorf("%s stream exceeds %d-byte bound", policy.label, policy.maxBytes), closeInstallFile(f))
	}
	if resp.ContentLength >= 0 && written != resp.ContentLength {
		return errors.Join(fmt.Errorf("%s stream length %d does not match Content-Length %d", policy.label, written, resp.ContentLength), closeInstallFile(f))
	}
	if err := f.Sync(); err != nil {
		return errors.Join(err, closeInstallFile(f))
	}
	return closeInstallFile(f)
}

func openDownloadedFile(path string, policy downloadPolicy) (*os.File, os.FileInfo, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%s cache %s is not a real regular file", policy.label, path)
	}
	if before.Size() < 0 || before.Size() > policy.maxBytes {
		return nil, nil, fmt.Errorf("%s cache %s has invalid size %d (maximum %d)", policy.label, path, before.Size(), policy.maxBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	opened, statErr := file.Stat()
	if statErr != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) || opened.Mode() != before.Mode() || opened.Size() != before.Size() {
		return nil, nil, errors.Join(fmt.Errorf("%s cache %s changed while opening", policy.label, path), statErr, closeInstallFile(file))
	}
	return file, before, nil
}

func revalidateDownloadedFile(path string, file *os.File, before os.FileInfo, policy downloadPolicy) error {
	after, statErr := file.Stat()
	pathAfter, pathErr := os.Lstat(path)
	if statErr != nil || pathErr != nil || !os.SameFile(before, after) || !os.SameFile(before, pathAfter) ||
		after.Mode() != before.Mode() || pathAfter.Mode() != before.Mode() || after.Size() != before.Size() || pathAfter.Size() != before.Size() ||
		!after.ModTime().Equal(before.ModTime()) || !pathAfter.ModTime().Equal(before.ModTime()) ||
		installFileChangeID(after) != installFileChangeID(before) || installFileChangeID(pathAfter) != installFileChangeID(before) {
		return errors.Join(fmt.Errorf("%s cache %s changed while being read", policy.label, path), statErr, pathErr)
	}
	return nil
}

func readDownloadedFile(path string, policy downloadPolicy) (_ []byte, retErr error) {
	file, before, err := openDownloadedFile(path, policy)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, closeInstallFile(file)) }()
	raw, err := io.ReadAll(io.LimitReader(file, policy.maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > policy.maxBytes {
		return nil, fmt.Errorf("%s cache %s exceeds %d-byte read bound", policy.label, path, policy.maxBytes)
	}
	if int64(len(raw)) != before.Size() {
		return nil, fmt.Errorf("%s cache %s size changed while being read", policy.label, path)
	}
	if err := revalidateDownloadedFile(path, file, before, policy); err != nil {
		return nil, err
	}
	return raw, nil
}

func hashDownloadedFile(path string, policy downloadPolicy) (_ string, retErr error) {
	file, before, err := openDownloadedFile(path, policy)
	if err != nil {
		return "", err
	}
	defer func() { retErr = errors.Join(retErr, closeInstallFile(file)) }()
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, policy.maxBytes+1))
	if err != nil {
		return "", err
	}
	if written > policy.maxBytes {
		return "", fmt.Errorf("%s cache %s exceeds %d-byte hash bound", policy.label, path, policy.maxBytes)
	}
	if written != before.Size() {
		return "", fmt.Errorf("%s cache %s size changed while being hashed", policy.label, path)
	}
	if err := revalidateDownloadedFile(path, file, before, policy); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// extractTarGz unpacks a gzipped tar into dest, taking only regular files and
// directories and rejecting any entry that would escape dest.
func extractTarGz(archive, dest string) (retErr error) {
	f, archiveInfo, err := openDownloadedFile(archive, releaseSourceDownload)
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, revalidateDownloadedFile(archive, f, archiveInfo, releaseSourceDownload), closeInstallFile(f))
	}()
	// Validate and extract through the same retained archive description. A
	// pathname replacement between the two passes cannot substitute a different
	// archive after validation.
	if err := validateTarGzMembers(f); err != nil {
		return err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewind validated source archive: %w", err)
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, gz.Close()) }()
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	directoryModes := map[string]os.FileMode{filepath.Clean(dest): 0o755}
	root := filepath.Clean(dest) + string(os.PathSeparator)
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		member, err := canonicalTarMember(hdr)
		if err != nil {
			return err // prevalidation makes this unreachable without archive mutation
		}
		target := filepath.Join(dest, filepath.FromSlash(member))
		if target != filepath.Clean(dest) && !strings.HasPrefix(target, root) {
			return fmt.Errorf("archive member %q escapes extraction root", hdr.Name)
		}
		for parent := filepath.Dir(target); pathAtOrBelow(filepath.Clean(dest), parent); parent = filepath.Dir(parent) {
			if _, exists := directoryModes[parent]; !exists {
				directoryModes[parent] = 0o755
			}
			if parent == filepath.Clean(dest) {
				break
			}
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			mode := os.FileMode(hdr.Mode).Perm()
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			directoryModes[target] = mode
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(hdr.Mode).Perm()
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			written, copyErr := io.CopyN(out, tr, hdr.Size)
			if copyErr != nil || written != hdr.Size { //nolint:gosec // our own checksummed release tarball
				out.Close()
				return errors.Join(copyErr, fmt.Errorf("archive member %q yielded %d bytes, want %d", hdr.Name, written, hdr.Size))
			}
			if err := out.Close(); err != nil {
				return err
			}
			// Creation modes are filtered by the process umask. Reapply the
			// sanitized archive permission after close so extraction is identical
			// under 0022, 0077, or any other ambient mask.
			if err := os.Chmod(target, mode); err != nil {
				return err
			}
		}
	}
	// Implicit parent directories have canonical 0755; explicit directory
	// headers retain their sanitized permission. Apply after extraction so a
	// restrictive header cannot prevent creation of later children.
	directories := make([]string, 0, len(directoryModes))
	for path := range directoryModes {
		directories = append(directories, path)
	}
	sort.Slice(directories, func(i, j int) bool {
		leftDepth := strings.Count(filepath.Clean(directories[i]), string(os.PathSeparator))
		rightDepth := strings.Count(filepath.Clean(directories[j]), string(os.PathSeparator))
		if leftDepth != rightDepth {
			return leftDepth > rightDepth
		}
		return directories[i] < directories[j]
	})
	for _, path := range directories {
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return errors.Join(err, fmt.Errorf("extracted directory %s changed before permission finalization", path))
		}
		if err := os.Chmod(path, directoryModes[path]); err != nil {
			return err
		}
	}
	return nil
}

type tarMemberKind uint8

const (
	tarMemberDirectory tarMemberKind = iota
	tarMemberRegular
)

func reserveSourceArchiveMember(name string, size, total int64) (int64, error) {
	if size < 0 || size > releaseSourceMemberMaxBytes {
		return 0, fmt.Errorf("archive member %q has invalid size %d (maximum %d)", name, size, releaseSourceMemberMaxBytes)
	}
	if total < 0 || total > releaseSourceTreeMaxBytes {
		return 0, fmt.Errorf("source archive has invalid accumulated size %d", total)
	}
	remaining := releaseSourceTreeMaxBytes - total
	if size > remaining {
		return 0, fmt.Errorf("source archive exceeds extracted byte bound (%d-byte member, %d bytes remaining)", size, remaining)
	}
	return total + size, nil
}

func validateTarGzMembers(source io.Reader) error {
	gz, err := gzip.NewReader(source)
	if err != nil {
		return err
	}
	type member struct {
		path string
		kind tarMemberKind
	}
	var members []member
	exact := map[string]bool{}
	folded := map[string]string{}
	reader := tar.NewReader(gz)
	var totalBytes int64
	for {
		hdr, nextErr := reader.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return errors.Join(nextErr, gz.Close())
		}
		canonical, canonicalErr := canonicalTarMember(hdr)
		if canonicalErr != nil {
			return errors.Join(canonicalErr, gz.Close())
		}
		kind := tarMemberRegular
		if hdr.Typeflag == tar.TypeDir {
			kind = tarMemberDirectory
		} else {
			nextTotal, sizeErr := reserveSourceArchiveMember(hdr.Name, hdr.Size, totalBytes)
			if sizeErr != nil {
				return errors.Join(sizeErr, gz.Close())
			}
			totalBytes = nextTotal
		}
		if len(members)+1 > releaseSourceTreeMaxFiles {
			return errors.Join(fmt.Errorf("source archive exceeds member bound of %d", releaseSourceTreeMaxFiles), gz.Close())
		}
		if exact[canonical] {
			return errors.Join(fmt.Errorf("archive member %q repeats canonical path %q", hdr.Name, canonical), gz.Close())
		}
		key := strings.ToLower(canonical)
		if prior, ok := folded[key]; ok {
			return errors.Join(fmt.Errorf("archive member %q aliases prior portable path %q", hdr.Name, prior), gz.Close())
		}
		for _, prior := range members {
			if prior.kind == tarMemberRegular && strings.HasPrefix(canonical, prior.path+"/") {
				return errors.Join(fmt.Errorf("archive member %q is nested beneath regular member %q", hdr.Name, prior.path), gz.Close())
			}
			if kind == tarMemberRegular && strings.HasPrefix(prior.path, canonical+"/") {
				return errors.Join(fmt.Errorf("regular archive member %q contains prior member %q", hdr.Name, prior.path), gz.Close())
			}
		}
		exact[canonical] = true
		folded[key] = canonical
		members = append(members, member{path: canonical, kind: kind})
	}
	return gz.Close()
}

func canonicalTarMember(hdr *tar.Header) (string, error) {
	name := hdr.Name
	if hdr.Typeflag == tar.TypeDir {
		name = strings.TrimSuffix(name, "/")
	}
	if name == "" || archivepath.IsAbs(name) {
		return "", fmt.Errorf("archive member %q is not a portable canonical relative path", hdr.Name)
	}
	canonical := archivepath.Clean(name)
	if canonical != name || canonical == "." || canonical == ".." || strings.HasPrefix(canonical, "../") {
		return "", fmt.Errorf("archive member %q is not a portable canonical relative path", hdr.Name)
	}
	if err := portablepath.ValidateRelative(canonical); err != nil {
		return "", fmt.Errorf("archive member %q is not portable: %w", hdr.Name, err)
	}
	switch hdr.Typeflag {
	case tar.TypeDir, tar.TypeReg:
		return canonical, nil
	default:
		return "", fmt.Errorf("archive member %q has unsupported type %d", hdr.Name, hdr.Typeflag)
	}
}

func singleChildDir(dir string) (string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return "", err
	}
	entries, readErr := f.ReadDir(2)
	if errors.Is(readErr, io.EOF) {
		readErr = nil
	}
	closeErr := closeInstallFile(f)
	if err := errors.Join(readErr, closeErr); err != nil {
		return "", err
	}
	if len(entries) != 1 {
		return "", fmt.Errorf("source archive must contain exactly the machinery/ root (found %d top-level entries)", len(entries))
	}
	entry := entries[0]
	if entry.Name() != "machinery" || !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("source archive root must be exactly machinery/ (found %q)", entry.Name())
	}
	return filepath.Join(dir, "machinery"), nil
}
