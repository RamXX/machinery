// Capture engine for the MAC-p9z1 retention surface: held-InputView
// validation, declared-root topology reconciliation, the real walk with
// link/special/alias/race rejection, deterministic role classification,
// immutable content-addressed blob publication and the separately bound
// control-plane and judgment-control inventories. No subprocesses.
package tdd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/tdd/protocol"
)

// validateCaptureRequest performs the request-side validation shared by
// capture and status: identity, declared-root grammars, alias rejection and
// the frozen-under-mutable-subject separation.
func validateCaptureRequest(req CaptureRequest) error {
	if req.Name != "" && !validID(req.Name) {
		return fmt.Errorf("INVALID_SCHEMA: capture name %q is not an ID", req.Name)
	}
	m := req.Manifest
	for _, list := range []struct {
		name  string
		paths []string
	}{
		{"implementation_roots", m.ImplementationRoots},
		{"frozen_roots", m.FrozenRoots},
	} {
		for i, p := range list.paths {
			if err := validateRootPath(p); err != nil {
				return fmt.Errorf("INVALID_SCHEMA: manifest.%s[%d]: %v", list.name, i, err)
			}
		}
		if err := rejectDuplicates(list.paths, "manifest."+list.name); err != nil {
			return err
		}
	}
	var declared []string
	declared = append(declared, m.ImplementationRoots...)
	declared = append(declared, m.FrozenRoots...)
	subjectDirs := map[string]bool{}
	subjectFiles := map[string]bool{}
	for i, se := range m.SubjectEntries {
		if err := validateRootPath(se.Path); err != nil {
			return fmt.Errorf("INVALID_SCHEMA: manifest.subject_entries[%d].path: %v", i, err)
		}
		if se.Kind != "file" && se.Kind != "directory" {
			return fmt.Errorf("INVALID_SCHEMA: manifest.subject_entries[%d].kind must be file or directory", i)
		}
		if se.Path == protocol.RepositoryRoot && se.Kind != "directory" {
			return fmt.Errorf("INVALID_SCHEMA: manifest.subject_entries[%d]: \".\" is only a directory subject", i)
		}
		if se.Kind == "directory" {
			subjectDirs[se.Path] = true
		} else {
			subjectFiles[se.Path] = true
		}
		declared = append(declared, se.Path)
	}
	for si, s := range m.Suites {
		if err := validateRootPath(s.Root); err != nil {
			return fmt.Errorf("INVALID_SCHEMA: manifest.suites[%d].root: %v", si, err)
		}
		declared = append(declared, s.Root)
		for fi, f := range s.Files {
			if err := validatePath(f); err != nil {
				return fmt.Errorf("INVALID_SCHEMA: manifest.suites[%d].files[%d]: %v", si, fi, err)
			}
			declared = append(declared, f)
		}
		for di, d := range s.DependencyRoots {
			if err := validateRootPath(d); err != nil {
				return fmt.Errorf("INVALID_SCHEMA: manifest.suites[%d].dependency_roots[%d]: %v", si, di, err)
			}
			declared = append(declared, d)
		}
		for ti, tst := range s.Tests {
			if err := validatePath(tst.Source); err != nil {
				return fmt.Errorf("INVALID_SCHEMA: manifest.suites[%d].tests[%d].source: %v", si, ti, err)
			}
			declared = append(declared, tst.Source)
			for ai, a := range tst.Assertions {
				if err := validatePath(a.Source); err != nil {
					return fmt.Errorf("INVALID_SCHEMA: manifest.suites[%d].tests[%d].assertions[%d].source: %v", si, ti, ai, err)
				}
				declared = append(declared, a.Source)
			}
		}
	}
	if err := rejectCaseAliases(declared, "manifest declared paths"); err != nil {
		return err
	}
	// a mutable subject directory must not contain or be an ancestor of
	// frozen/test/dependency declared entries
	underSubject := func(p string) bool {
		for d := range subjectDirs {
			if d == protocol.RepositoryRoot || p == d || strings.HasPrefix(p, d+"/") {
				return true
			}
		}
		return false
	}
	for _, p := range declared {
		if subjectFiles[p] || subjectDirs[p] {
			continue
		}
		if underSubject(p) {
			return fmt.Errorf("INVALID_SCHEMA: declared entry %q lies under a mutable subject directory; frozen descendants under mutable subjects are rejected", p)
		}
	}
	return nil
}

// validateHeldInputView validates the held InputView contract: nonnil
// revalidate/release callbacks, real non-symlink roots and closed relative
// paths.
func validateHeldInputView(v InputView) error {
	if v.SourceRoot == "" || v.ControlRoot == "" {
		return fmt.Errorf("INVALID_SCHEMA: InputView requires SourceRoot and ControlRoot")
	}
	if v.Revalidate == nil {
		return fmt.Errorf("INVALID_SCHEMA: InputView.Revalidate must be nonnil; a missing validator is rejected")
	}
	if v.Release == nil {
		return fmt.Errorf("INVALID_SCHEMA: InputView.Release must be nonnil; a missing release callback is rejected")
	}
	if err := validateRootPath(v.DesignPath); err != nil {
		return fmt.Errorf("INVALID_SCHEMA: InputView.DesignPath: %v", err)
	}
	for i, p := range v.ImplementationPaths {
		if err := validateRootPath(p); err != nil {
			return fmt.Errorf("INVALID_SCHEMA: InputView.ImplementationPaths[%d]: %v", i, err)
		}
	}
	for _, root := range []string{v.SourceRoot, v.ControlRoot} {
		fi, err := os.Lstat(root)
		if err != nil || !fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("INVALID_SCHEMA: InputView root %s must be a real non-symlink directory", root)
		}
	}
	return nil
}

// captureEngine performs one bounded capture (or read-only inspection)
// over the held view.
type captureEngine struct {
	ctx        context.Context
	req        CaptureRequest
	store      *storeView // nil for read-only inspection
	root       *os.Root
	entries    []bundleEntryData
	treeDigest string
	entryCap   int64
	depthCap   int64
	byteCap    int64
	totalBytes int64
	inodes      map[inodeIdentity]string
	visited     map[string]bool
	roleOf      func(rel string) string
}

func newCaptureEngine(ctx context.Context, req CaptureRequest, store *storeView) (*captureEngine, error) {
	if err := validateHeldInputView(req.Inputs); err != nil {
		return nil, err
	}
	if err := req.Inputs.Revalidate(); err != nil {
		return nil, fmt.Errorf("STALE_INPUT: the held InputView failed revalidation before capture: %v", err)
	}
	limits := req.Limits
	if limits.Entries <= 0 {
		limits.Entries = protocol.LimitEntriesDefault
	}
	if limits.Depth <= 0 {
		limits.Depth = protocol.LimitDepthDefault
	}
	if limits.BundleBytes <= 0 {
		limits.BundleBytes = protocol.LimitBundleDefault
	}
	if limits.Entries > protocol.LimitEntriesMax {
		limits.Entries = protocol.LimitEntriesMax
	}
	if limits.Depth > protocol.LimitDepthMax {
		limits.Depth = protocol.LimitDepthMax
	}
	if limits.BundleBytes > protocol.LimitBundleMaxBytes {
		limits.BundleBytes = protocol.LimitBundleMaxBytes
	}
	root, err := os.OpenRoot(req.Inputs.SourceRoot)
	if err != nil {
		return nil, fmt.Errorf("INVALID_SCHEMA: cannot hold the source root: %w", err)
	}
	e := &captureEngine{
		ctx: ctx, req: req, store: store, root: root,
		entryCap: limits.Entries, depthCap: limits.Depth, byteCap: limits.BundleBytes,
		inodes:  map[inodeIdentity]string{},
		visited: map[string]bool{},
	}
	e.roleOf = e.buildClassifier()
	return e, nil
}

// buildClassifier returns the deterministic role assignment: subject
// entries first, then suite dependency roots, then explicitly frozen inputs
// (frozen roots, suite files and assertion sources), then the design
// payload, then the frozen default. Every non-subject role is byte-immutable
// during replay.
func (e *captureEngine) buildClassifier() func(rel string) string {
	m := e.req.Manifest
	subject := newPrefixSet(false)
	for _, se := range m.SubjectEntries {
		subject.add(se.Path)
	}
	deps := newPrefixSet(false)
	for _, s := range m.Suites {
		for _, d := range s.DependencyRoots {
			deps.add(d)
		}
	}
	frozenRoots := newPrefixSet(false)
	for _, fr := range m.FrozenRoots {
		frozenRoots.add(fr)
	}
	explicitFrozen := newPrefixSet(true)
	for _, s := range m.Suites {
		for _, f := range s.Files {
			explicitFrozen.add(f)
		}
		for _, t := range s.Tests {
			explicitFrozen.add(t.Source)
			for _, a := range t.Assertions {
				explicitFrozen.add(a.Source)
			}
		}
	}
	design := newPrefixSet(false)
	design.add(e.req.Inputs.DesignPath)
	defaults := newPrefixSet(false)
	for _, r := range m.ImplementationRoots {
		defaults.add(r)
	}
	for _, s := range m.Suites {
		defaults.add(s.Root)
	}
	return func(rel string) string {
		switch {
		case subject.covers(rel):
			return RoleSubject
		case deps.covers(rel):
			return RoleDependency
		case frozenRoots.covers(rel):
			return RoleFrozen
		case explicitFrozen.covers(rel):
			return RoleFrozen
		case design.covers(rel):
			return RoleDesign
		default:
			return RoleFrozen
		}
	}
}

// prefixSet matches exact paths and, for declared directories, their
// subtrees; "." covers everything. With exactOnly, only exact matches hit.
type prefixSet struct {
	exactOnly bool
	exact     map[string]bool
	dirs      []string
	rootAll   bool
}

func newPrefixSet(exactOnly bool) *prefixSet {
	return &prefixSet{exactOnly: exactOnly, exact: map[string]bool{}}
}

func (s *prefixSet) add(p string) {
	if s.exactOnly {
		s.exact[p] = true
		return
	}
	if p == protocol.RepositoryRoot {
		s.rootAll = true
		return
	}
	s.exact[p] = true
	s.dirs = append(s.dirs, p)
}

func (s *prefixSet) covers(p string) bool {
	if s.exactOnly {
		return s.exact[p]
	}
	if s.rootAll || s.exact[p] {
		return true
	}
	for _, d := range s.dirs {
		if strings.HasPrefix(p, d+"/") {
			return true
		}
	}
	return false
}

// run walks, classifies, publishes blobs/bundle/controls and revalidates.
func (e *captureEngine) run() error {
	defer e.root.Close()
	if err := e.walk(); err != nil {
		return err
	}
	if err := checkCtx(e.ctx); err != nil {
		return err
	}
	e.treeDigest = treeDigestOfEntries(e.entries)
	for _, en := range e.entries {
		if en.Kind != "file" {
			continue
		}
		if err := checkCtx(e.ctx); err != nil {
			return err
		}
		data, err := e.readVerifiedFile(en.Path)
		if err != nil {
			return err
		}
		if err := publishImmutableFile(filepath.Join(e.store.root, storeBlobs, strings.TrimPrefix(en.Digest, "sha256:")), storeBlobMod, data); err != nil {
			return fmt.Errorf("CUSTODY_ERROR: publishing blob %s: %w", en.Digest, err)
		}
	}
	objDir := filepath.Join(e.store.root, storeObjects, strings.TrimPrefix(e.treeDigest, "sha256:"))
	if err := os.Mkdir(objDir, storeRootMod); err != nil && !os.IsExist(err) {
		return fmt.Errorf("CUSTODY_ERROR: creating the object directory: %w", err)
	}
	if err := publishImmutableFile(filepath.Join(objDir, "bundle.json"), storeFileMod, encodeBundleCanonical(e.entries, e.treeDigest)); err != nil {
		return fmt.Errorf("CUSTODY_ERROR: publishing bundle.json: %w", err)
	}
	if _, err := e.captureControls(true); err != nil {
		return err
	}
	if _, err := e.captureJudgment(true); err != nil {
		return err
	}
	if err := e.req.Inputs.Revalidate(); err != nil {
		return fmt.Errorf("STALE_INPUT: the held InputView changed during capture: %v", err)
	}
	return nil
}

// walk collects the complete retained inventory beneath the declared roots.
func (e *captureEngine) walk() error {
	m := e.req.Manifest
	roots := map[string]bool{}
	addRoot := func(p string) { roots[p] = true }
	for _, r := range m.ImplementationRoots {
		addRoot(r)
	}
	for _, r := range m.FrozenRoots {
		addRoot(r)
	}
	for _, s := range m.Suites {
		addRoot(s.Root)
		for _, d := range s.DependencyRoots {
			addRoot(d)
		}
	}
	for _, se := range m.SubjectEntries {
		fi, err := e.root.Lstat(se.Path)
		if err != nil {
			return fmt.Errorf("MISSING_CONTRACT: subject entry %q does not exist under the source root", se.Path)
		}
		if fi.IsDir() != (se.Kind == "directory") {
			return fmt.Errorf("MISSING_CONTRACT: subject entry %q kind %q does not match the filesystem", se.Path, se.Kind)
		}
		addRoot(se.Path)
	}
	controlNS := path.Join(e.req.Inputs.DesignPath, protocol.ControlDirName)
	judgmentPath := path.Join(e.req.Inputs.DesignPath, "attestations.yaml")
	rootStat, err := e.root.Stat(".")
	if err != nil {
		return err
	}
	e.record(bundleEntryData{Path: protocol.RepositoryRoot, Kind: "directory", Mode: uint32(rootStat.Mode().Perm()), Role: e.roleOf(protocol.RepositoryRoot)})
	ordered := make([]string, 0, len(roots))
	for r := range roots {
		ordered = append(ordered, r)
	}
	sort.Strings(ordered)
	for _, r := range ordered {
		if r == protocol.RepositoryRoot {
			if err := e.walkDir(".", controlNS, judgmentPath, 0); err != nil {
				return err
			}
			continue
		}
		if err := e.walkEntry(r, controlNS, judgmentPath, 0); err != nil {
			return err
		}
	}
	rels := make([]string, 0, len(e.entries))
	for _, en := range e.entries {
		rels = append(rels, en.Path)
	}
	if err := rejectCaseAliases(rels, "captured inventory"); err != nil {
		return err
	}
	if int64(len(e.entries)) > e.entryCap {
		return fmt.Errorf("OUTPUT_LIMIT: captured inventory exceeds the %d-entry bound", e.entryCap)
	}
	if e.totalBytes > e.byteCap {
		return fmt.Errorf("OUTPUT_LIMIT: captured inventory exceeds the %d-byte bundle bound", e.byteCap)
	}
	return nil
}

func (e *captureEngine) walkDir(rel, controlNS, judgmentPath string, depth int64) error {
	if depth > e.depthCap {
		return fmt.Errorf("OUTPUT_LIMIT: captured inventory exceeds the depth bound %d at %s", e.depthCap, rel)
	}
	f, err := e.root.Open(rel)
	if err != nil {
		return fmt.Errorf("INVALID_SCHEMA: cannot enumerate %q: %w", rel, err)
	}
	dirents, err := f.ReadDir(-1)
	f.Close()
	if err != nil {
		return fmt.Errorf("INVALID_SCHEMA: cannot enumerate %q: %w", rel, err)
	}
	names := make([]string, 0, len(dirents))
	for _, d := range dirents {
		names = append(names, d.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		if err := e.walkEntry(path.Join(rel, name), controlNS, judgmentPath, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (e *captureEngine) walkEntry(rel, controlNS, judgmentPath string, depth int64) error {
	// exact payload exclusions: the reserved control namespace, VCS .git
	// subtrees and the separately captured judgment control
	if rel == controlNS || rel == judgmentPath || path.Base(rel) == ".git" {
		return nil
	}
	if e.visited[rel] {
		return nil // declared roots may overlap; one entry is retained once
	}
	e.visited[rel] = true
	fi, err := e.root.Lstat(rel)
	if err != nil {
		return fmt.Errorf("INVALID_SCHEMA: cannot inspect %q: %w", rel, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("INVALID_SCHEMA: %q is a symlink; escaping or aliased links are rejected and dependencies must be resolved into a separate verified closure before capture", rel)
	}
	if fi.Mode()&(os.ModeDevice|os.ModeNamedPipe|os.ModeSocket) != 0 {
		return fmt.Errorf("INVALID_SCHEMA: %q is a special file (device/FIFO/socket); special files are rejected", rel)
	}
	if fi.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return fmt.Errorf("INVALID_SCHEMA: %q carries nonportable special mode bits; declared ACL/nonportable metadata dependencies are unsupported", rel)
	}
	if rel != protocol.RepositoryRoot {
		if err := validatePath(rel); err != nil {
			return fmt.Errorf("INVALID_SCHEMA: walked path %q: %v", rel, err)
		}
	}
	if fi.IsDir() {
		e.record(bundleEntryData{Path: rel, Kind: "directory", Mode: uint32(fi.Mode().Perm()), Role: e.roleOf(rel)})
		return e.walkDir(rel, controlNS, judgmentPath, depth)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("INVALID_SCHEMA: %q is not a regular file; irregular entries are rejected", rel)
	}
	if fi.Size() > e.byteCap-e.totalBytes {
		return fmt.Errorf("OUTPUT_LIMIT: file %q exceeds the remaining bundle byte bound", rel)
	}
	if key, ok := inodeKey(fi); ok {
		if prev, dup := e.inodes[key]; dup {
			return fmt.Errorf("INVALID_SCHEMA: %q and %q are the same inode hardlinked at two captured paths; alias substitution is rejected", rel, prev)
		}
		e.inodes[key] = rel
	}
	data, err := e.readFileBound(rel, fi)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	e.totalBytes += int64(len(data))
	e.record(bundleEntryData{
		Path: rel, Kind: "file", Mode: uint32(fi.Mode().Perm()),
		Size: int64(len(data)), Digest: "sha256:" + hex.EncodeToString(sum[:]), Role: e.roleOf(rel),
	})
	return nil
}

// readFileBound reads one file through the held root and binds the bytes to
// the pre-inspected identity, detecting substitution races.
func (e *captureEngine) readFileBound(rel string, before os.FileInfo) ([]byte, error) {
	f, err := e.root.Open(rel)
	if err != nil {
		return nil, fmt.Errorf("INVALID_SCHEMA: cannot open %q: %w", rel, err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Mode() != before.Mode() || st.Size() != before.Size() || !os.SameFile(st, before) {
		return nil, fmt.Errorf("STALE_INPUT: %q changed identity while being read (topology/mode race)", rel)
	}
	data := make([]byte, st.Size())
	if _, err := readFull(f, data); err != nil {
		return nil, fmt.Errorf("STALE_INPUT: reading %q: %v", rel, err)
	}
	st2, err := f.Stat()
	if err != nil || st2.Size() != st.Size() || !os.SameFile(st2, before) {
		return nil, fmt.Errorf("STALE_INPUT: %q changed while being read", rel)
	}
	return data, nil
}

// readVerifiedFile re-reads a captured file for blob publication, verifying
// it still hashes to the recorded digest.
func (e *captureEngine) readVerifiedFile(rel string) ([]byte, error) {
	fi, err := e.root.Lstat(rel)
	if err != nil {
		return nil, fmt.Errorf("STALE_INPUT: %q disappeared during capture: %v", rel, err)
	}
	data, err := e.readFileBound(rel, fi)
	if err != nil {
		return nil, err
	}
	for _, en := range e.entries {
		if en.Path == rel && digestOfBytes(data) != en.Digest {
			return nil, fmt.Errorf("STALE_INPUT: %q changed bytes during capture", rel)
		}
	}
	return data, nil
}

func (e *captureEngine) record(en bundleEntryData) {
	e.entries = append(e.entries, en)
}

// captureControls validates the separate control materialization, derives
// the typed control inventory digest (role control) and, when archiving,
// publishes the exact bytes into controls/.
func (e *captureEngine) captureControls(archive bool) (string, error) {
	ctlRoot := e.req.Inputs.ControlRoot
	rootFi, err := os.Lstat(ctlRoot)
	if err != nil {
		return "", fmt.Errorf("MISSING_CONTRACT: the separate control materialization %s is unavailable: %w", ctlRoot, err)
	}
	tree := []treeEntry{{rel: protocol.RepositoryRoot, dir: true, perm: uint32(rootFi.Mode().Perm()), role: RoleControlPlane}}
	entries, err := os.ReadDir(ctlRoot)
	if err != nil {
		return "", fmt.Errorf("MISSING_CONTRACT: the control materialization %s cannot be enumerated: %w", ctlRoot, err)
	}
	var files []string
	for _, en := range entries {
		switch en.Name() {
		case protocol.PlanFileName:
			if !en.Type().IsRegular() {
				return "", fmt.Errorf("INVALID_SCHEMA: control/%s must be a regular file", en.Name())
			}
			files = append(files, en.Name())
		case protocol.MilestonesDirName:
			if !en.IsDir() {
				return "", fmt.Errorf("INVALID_SCHEMA: control/%s must be a directory", en.Name())
			}
			tree = append(tree, treeEntry{rel: en.Name(), dir: true, perm: uint32(mustMode(filepath.Join(ctlRoot, en.Name()))), role: RoleControlPlane})
			ms, merr := os.ReadDir(filepath.Join(ctlRoot, en.Name()))
			if merr != nil {
				return "", merr
			}
			for _, m := range ms {
				base := m.Name()
				if !m.Type().IsRegular() || !strings.HasSuffix(base, ".json") || !validMilestoneID(strings.TrimSuffix(base, ".json")) {
					return "", fmt.Errorf("INVALID_SCHEMA: control namespace milestones/ holds unknown entry %q", base)
				}
				files = append(files, path.Join(protocol.MilestonesDirName, base))
			}
		default:
			return "", fmt.Errorf("INVALID_SCHEMA: control namespace holds unknown entry %q (only plan.json and milestones/ are authored controls)", en.Name())
		}
	}
	if _, err := os.Stat(filepath.Join(ctlRoot, protocol.PlanFileName)); err != nil {
		return "", fmt.Errorf("MISSING_CONTRACT: the control materialization carries no plan.json")
	}
	sort.Strings(files)
	for _, f := range files {
		p := filepath.Join(ctlRoot, filepath.FromSlash(f))
		data, cerr := readCappedControlBytes(p)
		if cerr != nil {
			return "", cerr
		}
		fi, ferr := os.Lstat(p)
		if ferr != nil {
			return "", ferr
		}
		sum := sha256.Sum256(data)
		tree = append(tree, treeEntry{rel: f, perm: uint32(fi.Mode().Perm()), size: int64(len(data)), dig: sum, role: RoleControlPlane})
		if archive {
			if perr := publishImmutableFile(filepath.Join(e.store.root, "controls", digestHexBytes(data)+".json"), storeFileMod, data); perr != nil {
				return "", fmt.Errorf("CUSTODY_ERROR: archiving control %s: %w", f, perr)
			}
		}
	}
	return encodeTreeDigest(tree), nil
}

// captureJudgment binds design/attestations.yaml when present; its exact
// bytes are archived and its typed inventory digest carries the
// judgment-control role. It is never part of the source bundle.
func (e *captureEngine) captureJudgment(archive bool) (string, error) {
	judgmentRel := path.Join(e.req.Inputs.DesignPath, "attestations.yaml")
	src := filepath.Join(e.req.Inputs.SourceRoot, filepath.FromSlash(judgmentRel))
	fi, err := os.Lstat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		return "", fmt.Errorf("INVALID_SCHEMA: the judgment control %s must be a regular non-symlink file", judgmentRel)
	}
	data, err := readCappedControlBytes(src)
	if err != nil {
		return "", err
	}
	if archive {
		if err := publishImmutableFile(filepath.Join(e.store.root, "controls", digestHexBytes(data)+".json"), storeFileMod, data); err != nil {
			return "", fmt.Errorf("CUSTODY_ERROR: archiving the judgment control: %w", err)
		}
	}
	sum := sha256.Sum256(data)
	tree := []treeEntry{
		{rel: protocol.RepositoryRoot, dir: true, perm: uint32(mustMode(filepath.Dir(src))), role: RoleJudgmentControl},
		{rel: "attestations.yaml", perm: uint32(fi.Mode().Perm()), size: int64(len(data)), dig: sum, role: RoleJudgmentControl},
	}
	return encodeTreeDigest(tree), nil
}

func mustMode(p string) os.FileMode {
	fi, err := os.Lstat(p)
	if err != nil {
		return 0o755
	}
	return fi.Mode().Perm()
}

// readCappedControlBytes reads one control-plane file with the fixed cap,
// symlink rejection and before/after identity binding.
func readCappedControlBytes(p string) ([]byte, error) {
	before, err := os.Lstat(p)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("INVALID_SCHEMA: %s must be a regular non-symlink control file", p)
	}
	if before.Size() > protocol.ControlInputMaxBytes {
		return nil, fmt.Errorf("OUTPUT_LIMIT: %s exceeds the control-input cap", p)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	after, err := os.Lstat(p)
	if err != nil || after.Size() != before.Size() || !os.SameFile(before, after) {
		return nil, fmt.Errorf("STALE_INPUT: %s changed while reading", p)
	}
	return data, nil
}

// readFull reads exactly len(buf) bytes or fails.
func readFull(f *os.File, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		m, err := f.Read(buf[n:])
		n += m
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
