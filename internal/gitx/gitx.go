// Package gitx runs the few git commands machinery needs.
package gitx

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func run(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return out.String(), errors.New(msg)
	}
	return out.String(), nil
}

// Head returns the full sha of HEAD.
func Head(root string) (string, error) {
	out, err := run(root, "rev-parse", "HEAD")
	return strings.TrimSpace(out), err
}

// Short returns the first 7 characters of a sha.
func Short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// Resolve returns the full sha of a ref.
func Resolve(root, ref string) (string, error) {
	out, err := run(root, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	return strings.TrimSpace(out), err
}

// Branch returns the current branch name, or "" when detached.
func Branch(root string) string {
	out, err := run(root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	b := strings.TrimSpace(out)
	if b == "HEAD" {
		return ""
	}
	return b
}

// StatusPaths returns the project-relative paths git status reports, relative
// to root, both paths of a rename or a copy included. Untracked files are
// listed one by one. Paths outside root are left out.
func StatusPaths(root string) ([]string, error) {
	out, err := run(root, "status", "--porcelain=v2", "-z", "--untracked-files=all", "--", ".")
	if err != nil {
		return nil, err
	}
	top, err := run(root, "rev-parse", "--show-prefix")
	if err != nil {
		return nil, err
	}
	prefix := strings.TrimSpace(top)
	var paths []string
	keep := func(p string) { // porcelain paths are relative to the top of the work tree
		if rel, ok := strings.CutPrefix(p, prefix); ok && rel != "" {
			paths = append(paths, rel)
		}
	}
	entries := strings.Split(out, "\x00")
	for i := 0; i < len(entries); i++ {
		e := entries[i]
		fields := statusFields[firstByte(e)]
		if fields == 0 {
			continue // a header line, or the end
		}
		parts := strings.SplitN(e, " ", fields+1)
		if len(parts) != fields+1 {
			continue
		}
		keep(parts[fields])
		if e[0] == '2' && i+1 < len(entries) {
			i++
			keep(entries[i])
		}
	}
	return paths, nil
}

// statusFields is how many fields come before the path in an entry of git
// status --porcelain=v2, by its first character: 8 of an ordinary entry, 9 of
// a rename or a copy (whose original path is the next entry), 10 of an
// unmerged one, 1 of an untracked file.
var statusFields = map[byte]int{'1': 8, '2': 9, 'u': 10, '?': 1}

func firstByte(s string) byte {
	if s == "" {
		return 0
	}
	return s[0]
}

// Change is a project-relative path that differs between two trees, and
// whether it is a submodule pointer (a gitlink) on either side.
type Change struct {
	Path    string
	Gitlink bool
}

// GitlinkMode is the mode git records for a submodule pointer.
const GitlinkMode = "160000"

// diffChanges runs a git diff in raw form, renames and copies detected, and
// returns every path it names: both paths of a rename or a copy.
func diffChanges(root string, args ...string) ([]Change, error) {
	full := append([]string{"diff", "--raw", "-z", "-M", "-C", "--relative"}, args...)
	out, err := run(root, append(full, "--", ".")...)
	if err != nil {
		return nil, err
	}
	var res []Change
	entries := strings.Split(out, "\x00")
	for i := 0; i < len(entries); i++ {
		meta := entries[i]
		if !strings.HasPrefix(meta, ":") {
			continue
		}
		// :<src mode> <dst mode> <src sha> <dst sha> <status>
		f := strings.Fields(meta[1:])
		if len(f) < 5 {
			continue
		}
		link := f[0] == GitlinkMode || f[1] == GitlinkMode
		n := 1
		if s := f[4][0]; s == 'R' || s == 'C' {
			n = 2
		}
		for ; n > 0 && i+1 < len(entries); n-- {
			i++
			res = append(res, Change{Path: entries[i], Gitlink: link})
		}
	}
	return res, nil
}

// DiffChanges lists the project-relative paths that differ between two
// commits: both paths of a rename or a copy, each with whether it is a
// submodule pointer in either commit.
func DiffChanges(root, a, b string) ([]Change, error) { return diffChanges(root, a, b) }

// Gitlinks lists the project-relative paths the index records as submodule
// pointers.
func Gitlinks(root string) ([]string, error) {
	out, err := run(root, "ls-files", "-s", "-z", "--", ".")
	if err != nil {
		return nil, err
	}
	var res []string
	for _, e := range strings.Split(out, "\x00") {
		// <mode> <sha> <stage>\t<path>
		if meta, p, ok := strings.Cut(e, "\t"); ok && strings.HasPrefix(meta, GitlinkMode+" ") {
			res = append(res, p)
		}
	}
	return res, nil
}

// Submodules lists the paths of the submodules .gitmodules records at the
// top of the work tree, project-relative; those outside root are left out.
func Submodules(root string) ([]string, error) {
	top, err := TopLevel(root)
	if err != nil {
		return nil, err
	}
	file := filepath.Join(top, ".gitmodules")
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	cmd := exec.Command("git", "config", "--file", file, "--null", "--get-regexp", `^submodule\..*\.path$`)
	cmd.Dir = root
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return nil, nil // no submodule has a path
	}
	if err != nil {
		return nil, fmt.Errorf("git config --file .gitmodules: %v", err)
	}
	prefix, err := run(root, "rev-parse", "--show-prefix")
	if err != nil {
		return nil, err
	}
	var res []string
	for _, e := range strings.Split(string(out), "\x00") {
		// <key>\n<value>
		if _, p, ok := strings.Cut(e, "\n"); ok {
			if rel, in := strings.CutPrefix(path.Clean(p), strings.TrimSpace(prefix)); in && rel != "" {
				res = append(res, rel)
			}
		}
	}
	sort.Strings(res)
	return res, nil
}

// Dirty reports whether the tree has changes outside the ignored paths. An
// ignored entry ending in "/" is a directory prefix.
func Dirty(root string, ignore ...string) (bool, error) {
	paths, err := StatusPaths(root)
	if err != nil {
		return false, err
	}
	for _, p := range paths {
		if !ignored(p, ignore) {
			return true, nil
		}
	}
	return false, nil
}

func ignored(p string, ignore []string) bool {
	for _, ig := range ignore {
		if ig == "" {
			continue
		}
		if strings.HasSuffix(ig, "/") {
			if strings.HasPrefix(p, ig) {
				return true
			}
		} else if p == ig {
			return true
		}
	}
	return false
}

// Show returns the content of a project-relative path at a ref.
func Show(root, ref, rel string) ([]byte, bool) {
	prefix, err := run(root, "rev-parse", "--show-prefix")
	if err != nil {
		return nil, false
	}
	cmd := exec.Command("git", "show", ref+":"+strings.TrimSpace(prefix)+rel)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, false
	}
	return out, true
}

// MergeBase returns the merge base of two refs.
func MergeBase(root, a, b string) (string, error) {
	out, err := run(root, "merge-base", a, b)
	return strings.TrimSpace(out), err
}

// StagedSince lists the project-relative paths that differ between a commit
// and the index: what is committed since it and what is staged, both paths of
// a rename or a copy.
func StagedSince(root, commit string) ([]string, error) {
	changes, err := diffChanges(root, "--cached", commit)
	if err != nil {
		return nil, err
	}
	return changePaths(changes), nil
}

// changePaths returns the paths of changes, sorted, each once.
func changePaths(changes []Change) []string {
	set := map[string]bool{}
	for _, c := range changes {
		set[c.Path] = true
	}
	res := make([]string, 0, len(set))
	for p := range set {
		res = append(res, p)
	}
	sort.Strings(res)
	return res
}

// ChangedSince lists project-relative paths that differ between a ref and the
// working tree, untracked files and both paths of a rename or a copy
// included.
func ChangedSince(root, ref string) ([]string, error) {
	changes, err := diffChanges(root, ref)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, p := range changePaths(changes) {
		set[p] = true
	}
	st, err := StatusPaths(root)
	if err != nil {
		return nil, err
	}
	for _, p := range st {
		set[p] = true
	}
	var res []string
	for p := range set {
		res = append(res, p)
	}
	sortStrings(res)
	return res, nil
}

// CommonDir returns the absolute git common directory (shared by worktrees).
func CommonDir(root string) (string, error) {
	out, err := run(root, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	d := strings.TrimSpace(out)
	if !filepath.IsAbs(d) {
		d = filepath.Join(root, d)
	}
	return filepath.Clean(d), nil
}

// CountCommits returns how many commits are in a..b.
func CountCommits(root, a, b string) (int, error) {
	out, err := run(root, "rev-list", "--count", a+".."+b)
	if err != nil {
		return 0, err
	}
	n := 0
	_, err = fmt.Sscanf(strings.TrimSpace(out), "%d", &n)
	return n, err
}

// IsAncestor reports whether a is an ancestor of b (or equal).
func IsAncestor(root, a, b string) bool {
	_, err := run(root, "merge-base", "--is-ancestor", a, b)
	return err == nil
}

// Newest returns the newest of the commits by commit date, as git rev-list
// --no-walk orders them, or "" for none.
func Newest(root string, shas []string) string {
	if len(shas) == 0 {
		return ""
	}
	out, err := run(root, append([]string{"rev-list", "--no-walk"}, shas...)...)
	if err != nil {
		return ""
	}
	if ls := lines(out); len(ls) > 0 {
		return ls[0]
	}
	return ""
}

// Exists reports whether a commit exists.
func Exists(root, sha string) bool {
	_, err := run(root, "cat-file", "-e", sha+"^{commit}")
	return err == nil
}

// CommitsTouching lists commits in (from, HEAD] that touch any of the paths,
// newest first.
func CommitsTouching(root, from string, paths []string) ([]string, error) {
	args := append([]string{"log", "--format=%H", from + "..HEAD", "--"}, paths...)
	out, err := run(root, args...)
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

// FilesOfCommit lists project-relative paths a commit changed, both paths of
// a rename.
func FilesOfCommit(root, sha string) ([]string, error) {
	out, err := run(root, "show", "--name-only", "--relative", "--no-renames", "--format=", sha, "--", ".")
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

// UserName returns git config user.name.
func UserName(root string) string {
	out, _ := run(root, "config", "user.name")
	return strings.TrimSpace(out)
}

// Ignored reports whether git ignores a project-relative path.
func Ignored(root, rel string) bool {
	_, err := run(root, "check-ignore", "-q", rel)
	return err == nil
}

// Tracked returns the files git tracks below root that match the
// pathspecs, root-relative.
func Tracked(root string, pathspecs ...string) ([]string, error) {
	out, err := run(root, append([]string{"ls-files", "-z", "--"}, pathspecs...)...)
	if err != nil {
		return nil, err
	}
	var res []string
	for _, f := range strings.Split(out, "\x00") {
		if f != "" {
			res = append(res, f)
		}
	}
	return res, nil
}

// Listed returns the files below root that git tracks and the untracked
// files it does not ignore, root-relative.
func Listed(root string) ([]string, error) {
	out, err := run(root, "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", ".")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var res []string
	for _, f := range strings.Split(out, "\x00") {
		if f != "" && !seen[f] {
			seen[f] = true
			res = append(res, f)
		}
	}
	return res, nil
}

// IsRepo reports whether root is inside a git work tree.
func IsRepo(root string) bool {
	_, err := run(root, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

func lines(s string) []string {
	var res []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			res = append(res, l)
		}
	}
	return res
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// TopLevel returns the top of the git work tree that holds dir.
func TopLevel(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--show-toplevel")
	return strings.TrimSpace(out), err
}

// StatusCounts counts, below root, the tracked files that differ from HEAD
// and the untracked files that are not ignored. Optional locks are off, so
// git does not refresh the index: nothing in the repository is written.
func StatusCounts(root string) (changed, untracked int, err error) {
	out, err := run(root, "--no-optional-locks", "status", "--porcelain", "--untracked-files=all", "--", ".")
	if err != nil {
		return 0, 0, err
	}
	for _, line := range strings.Split(out, "\n") {
		switch {
		case len(line) < 4:
		case strings.HasPrefix(line, "?? "):
			untracked++
		default:
			changed++
		}
	}
	return changed, untracked, nil
}

// FirstAdd returns the first commit that adds a project-relative path, or ""
// when no commit does.
func FirstAdd(root, rel string) (string, error) {
	out, err := run(root, "log", "--diff-filter=A", "--reverse", "--format=%H", "--", rel)
	if err != nil {
		return "", err
	}
	if ls := lines(out); len(ls) > 0 {
		return ls[0], nil
	}
	return "", nil
}

// Branches lists the local branches, sorted.
func Branches(root string) ([]string, error) {
	out, err := run(root, "for-each-ref", "--format=%(refname:short)", "refs/heads/")
	if err != nil {
		return nil, err
	}
	res := lines(out)
	sort.Strings(res)
	return res, nil
}

// Archive extracts the tree of HEAD below root into dest: git archive piped
// to an extraction, with no worktree and no checkout. Every write goes
// through an os.Root bound to dest, so nothing lands outside it. A symbolic
// link whose target is absolute, or resolves outside dest, is not created:
// outside lists those links, sorted, by path. An entry whose name is
// absolute or holds a ".." segment stops the extraction. files counts the
// files and links written.
func Archive(root, dest string) (files int, outside []string, err error) {
	cmd := exec.Command("git", "archive", "--format=tar", "HEAD")
	cmd.Dir = root
	var errb bytes.Buffer
	cmd.Stderr = &errb
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return 0, nil, err
	}
	if err := cmd.Start(); err != nil {
		return 0, nil, err
	}
	files, outside, xerr := extract(pipe, dest)
	io.Copy(io.Discard, pipe)
	if werr := cmd.Wait(); werr != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = werr.Error()
		}
		return files, outside, errors.New("git archive: " + msg)
	}
	return files, outside, xerr
}

// leaves reports whether a link at name with this target points outside the
// tree by its text alone: an absolute target, or one that climbs above the
// root with "..".
func leaves(name, target string) bool {
	if target == "" || path.IsAbs(target) || filepath.IsAbs(target) {
		return true
	}
	j := path.Clean(path.Join(path.Dir(name), target))
	return j == ".." || strings.HasPrefix(j, "../")
}

func extract(r io.Reader, dest string) (int, []string, error) {
	rt, err := os.OpenRoot(dest)
	if err != nil {
		return 0, nil, err
	}
	defer rt.Close()
	tr := tar.NewReader(r)
	n := 0
	var outside, links []string
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return n, outside, err
		}
		if h.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		raw := strings.TrimSuffix(h.Name, "/")
		if path.IsAbs(raw) || filepath.IsAbs(raw) || containsDotDot(raw) {
			return n, outside, fmt.Errorf("archive entry %q is absolute or holds ..; nothing outside the snapshot is written", h.Name)
		}
		name := path.Clean(raw)
		if name == "." || name == "" {
			continue
		}
		parent := path.Dir(name)
		switch h.Typeflag {
		case tar.TypeDir:
			if err := rt.MkdirAll(name, 0o755); err != nil {
				return n, outside, err
			}
		case tar.TypeReg:
			if err := rt.MkdirAll(parent, 0o755); err != nil {
				return n, outside, err
			}
			mode := os.FileMode(0o644)
			if h.Mode&0o111 != 0 {
				mode = 0o755
			}
			f, err := rt.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				return n, outside, err
			}
			_, cerr := io.Copy(f, tr)
			if err := f.Close(); cerr == nil {
				cerr = err
			}
			if cerr != nil {
				return n, outside, cerr
			}
			n++
		case tar.TypeSymlink:
			if leaves(name, h.Linkname) {
				outside = append(outside, name)
				continue
			}
			if err := rt.MkdirAll(parent, 0o755); err != nil {
				return n, outside, err
			}
			if err := rt.Symlink(h.Linkname, name); err != nil {
				return n, outside, err
			}
			links = append(links, name)
			n++
		}
	}
	// a link that stays inside by its text may still leave through another
	// link: resolve each for real, and remove the ones that leave
	realDest := RealPath(dest)
	for changed := true; changed; {
		changed = false
		for i, l := range links {
			if l == "" || stays(dest, realDest, l) {
				continue
			}
			if err := rt.Remove(l); err != nil {
				return n, outside, err
			}
			outside = append(outside, l)
			links[i] = ""
			n--
			changed = true
		}
	}
	sort.Strings(outside)
	return n, outside, nil
}

func containsDotDot(name string) bool {
	for _, seg := range strings.Split(name, "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}

// stays reports whether the link at rel below dest resolves inside dest.
func stays(dest, realDest, rel string) bool {
	p := filepath.Join(dest, filepath.FromSlash(rel))
	target, err := os.Readlink(p)
	if err != nil || filepath.IsAbs(target) {
		return false
	}
	dir := RealPath(filepath.Dir(p))
	return Within(dir, realDest) && Within(RealPath(filepath.Join(dir, target)), realDest)
}

// RealPath returns an absolute path with the symbolic links of its longest
// existing prefix resolved.
func RealPath(p string) string {
	cur, rest := filepath.Clean(p), ""
	for {
		if r, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(r, rest)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return filepath.Clean(p)
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
}

// Within reports whether path is dir or below it.
func Within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// InitCommit makes dir a repository of its own and commits everything in it,
// ignored files included, with hooks and signing off. The environment of a
// surrounding repository (GIT_DIR and the like) is not inherited.
func InitCommit(dir, message string) error {
	return snapshotGit(dir, [][]string{
		{"init", "-q"},
		{"add", "-A", "-f"},
		{"commit", "-q", "--no-verify", "--allow-empty", "-m", message},
	})
}

// CommitAll commits everything in a snapshot made by InitCommit, hooks and
// signing off.
func CommitAll(dir, message string) error {
	return snapshotGit(dir, [][]string{
		{"add", "-A", "-f"},
		{"commit", "-q", "--no-verify", "--allow-empty", "-m", message},
	})
}

func snapshotGit(dir string, steps [][]string) error {
	var env []string
	for _, kv := range os.Environ() {
		switch k, _, _ := strings.Cut(kv, "="); k {
		case "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY", "GIT_COMMON_DIR", "GIT_CEILING_DIRECTORIES":
			continue
		}
		env = append(env, kv)
	}
	base := []string{"-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false",
		"-c", "user.name=machinery assess", "-c", "user.email=assess@machinery.invalid"}
	for _, args := range steps {
		cmd := exec.Command("git", append(append([]string{}, base...), args...)...)
		cmd.Dir = dir
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git %s in the snapshot: %s", args[0], strings.TrimSpace(string(out)))
		}
	}
	return nil
}
