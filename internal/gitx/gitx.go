// Package gitx runs every git command machinery runs, through one function
// (git): plumbing where git has it, git status only as --porcelain=v2 -z,
// and every setting a configuration could use to change an answer fixed on
// the command line (docs/formats.md section 10.1, "How machinery reads git").
package gitx

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// neutral comes before the arguments of every git command. Optional locks
// are off, so a read never rewrites the index, and every setting that could
// change what a query returns is fixed. The configuration is otherwise kept:
// safe.directory (a repository another user owns), user.name, worktrees and
// the ignore rules work as they do for git.
var neutral = []string{
	"--no-optional-locks",
	"-c", "core.quotepath=off",
	"-c", "core.fsmonitor=false",
	"-c", "diff.ignoreSubmodules=none",
	"-c", "diff.renames=false",
	"-c", "status.renames=false",
	"-c", "color.ui=never",
	"-c", "submodule.recurse=false",
}

// compared are the flags of every comparison of trees, the index and the
// work tree: a submodule is compared by its pointer whatever
// submodule.<name>.ignore or diff.ignoreSubmodules say, and no rename or copy
// is detected, so a renamed file is its old path deleted and its new path
// added, both read, whatever threshold or setting there is.
var compared = []string{"--ignore-submodules=none", "--no-renames"}

// diffFlags are the flags of every diff-tree and diff-index: raw,
// NUL-separated, recursive, paths relative to the directory it runs in, and
// no external diff, text conversion or color.
var diffFlags = append([]string{"-r", "-z", "--raw", "--relative", "--no-ext-diff", "--no-textconv", "--no-color"}, compared...)

// Error is a git command that failed: its arguments, its exit code (-1 when
// it did not start) and what it said.
type Error struct {
	Args []string
	Code int
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

// isNo reports whether err is the exit code 1 by which a yes-or-no query
// (merge-base --is-ancestor, check-ignore, symbolic-ref --quiet,
// rev-parse --verify --quiet, config --get) says no.
func isNo(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == 1
}

// options of one git command: the environment (nil keeps machinery's),
// what it reads on stdin, and where its output goes (nil: returned).
type options struct {
	env    []string
	stdin  io.Reader
	stdout io.Writer
}

// git runs one git command in dir with the neutral settings and git's
// messages in the C locale, and returns its output. A command that fails
// returns an *Error with what git said. Every git command machinery runs goes
// through it.
func git(dir string, o options, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append(append([]string{}, neutral...), args...)...)
	cmd.Dir = dir
	env := o.env
	if env == nil {
		env = os.Environ()
	}
	cmd.Env = append(append([]string{}, env...), "LC_ALL=C")
	cmd.Stdin = o.stdin
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	if o.stdout != nil {
		cmd.Stdout = o.stdout
	}
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		e := &Error{Args: args, Code: -1, Msg: strings.TrimSpace(errb.String())}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			e.Code = exit.ExitCode()
		}
		if e.Msg == "" {
			e.Msg = "git " + verb(args) + ": " + err.Error()
		}
		return out.Bytes(), e
	}
	return out.Bytes(), nil
}

// verb is the git command among args: the first that is no option.
func verb(args []string) string {
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "-c":
			i++
		case !strings.HasPrefix(a, "-"):
			return a
		}
	}
	return ""
}

// run runs a git query in root and returns its output.
func run(root string, args ...string) (string, error) {
	out, err := git(root, options{}, args...)
	return string(out), err
}

// revs refuses a revision that git would read as an option.
func revs(names ...string) error {
	for _, n := range names {
		if n == "" || strings.HasPrefix(n, "-") {
			return fmt.Errorf("%q is not a git revision", n)
		}
	}
	return nil
}

// fields splits NUL-separated output into its entries; the empty entry after
// the last NUL is dropped.
func fields(out string) []string {
	entries := strings.Split(out, "\x00")
	if n := len(entries); n > 0 && entries[n-1] == "" {
		entries = entries[:n-1]
	}
	return entries
}

// Head returns the full sha of HEAD.
func Head(root string) (string, error) {
	out, err := run(root, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Short returns the first 7 characters of a sha.
func Short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// Commit returns the full sha of the commit a ref names. ok is false when it
// names none: no such ref or object, or one that is no commit.
func Commit(root, ref string) (sha string, ok bool, err error) {
	if revs(ref) != nil {
		return "", false, nil
	}
	out, err := run(root, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	if isNo(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return strings.TrimSpace(out), true, nil
}

// Resolve returns the full sha of the commit a ref names, an error when it
// names none.
func Resolve(root, ref string) (string, error) {
	sha, ok, err := Commit(root, ref)
	if err == nil && !ok {
		err = fmt.Errorf("%s names no commit", ref)
	}
	return sha, err
}

// Exists reports whether a commit exists.
func Exists(root, sha string) (bool, error) {
	_, ok, err := Commit(root, sha)
	return ok, err
}

// HeadIfAny returns the full sha of HEAD, or "" when root is outside a git
// work tree or its branch has no commit yet. Any other failure is an error.
func HeadIfAny(root string) (string, error) {
	repo, err := IsRepo(root)
	if err != nil || !repo {
		return "", err
	}
	sha, _, err := Commit(root, "HEAD")
	return sha, err
}

// Branch returns the current branch name, or "" when HEAD is detached.
func Branch(root string) (string, error) {
	out, err := run(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	if isNo(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// statusEntry is one entry of git status --porcelain=v2: its kind ('1'
// ordinary, '2' rename or copy, 'u' unmerged, '?' untracked) and its paths,
// relative to the top of the work tree.
type statusEntry struct {
	kind  byte
	paths []string
}

// statusFields is how many fields come before the path in an entry of git
// status --porcelain=v2, by its first character: 8 of an ordinary entry, 9 of
// a rename or a copy (whose original path is the next entry), 10 of an
// unmerged one, 1 of an untracked file. With --no-renames git writes no
// rename or copy entry; one is read all the same.
var statusFields = map[byte]int{'1': 8, '2': 9, 'u': 10, '?': 1}

// status runs git status below root: every change of the index and the work
// tree, untracked files one by one, ignored files left out, submodules
// compared, no rename detected. An entry machinery cannot read is an error.
func status(root string) ([]statusEntry, error) {
	out, err := run(root, append(append([]string{"status", "--porcelain=v2", "-z", "--untracked-files=all", "--ignored=no"}, compared...), "--", ".")...)
	if err != nil {
		return nil, err
	}
	var res []statusEntry
	entries := fields(out)
	for i := 0; i < len(entries); i++ {
		e := entries[i]
		if strings.HasPrefix(e, "# ") {
			continue // a header
		}
		n, known := statusFields[firstByte(e)]
		parts := strings.SplitN(e, " ", n+1)
		if !known || len(parts) != n+1 || parts[n] == "" {
			return nil, fmt.Errorf("git status: an entry machinery cannot read: %q", e)
		}
		se := statusEntry{kind: e[0], paths: []string{parts[n]}}
		if se.kind == '2' {
			if i++; i >= len(entries) {
				return nil, fmt.Errorf("git status: %q has no original path", e)
			}
			se.paths = append(se.paths, entries[i])
		}
		res = append(res, se)
	}
	return res, nil
}

func firstByte(s string) byte {
	if s == "" {
		return 0
	}
	return s[0]
}

// StatusPaths returns, sorted and each once, the paths git status reports,
// relative to root: both paths of a rename, a submodule whose checkout or
// files changed, untracked files one by one. Paths outside root are left out.
func StatusPaths(root string) ([]string, error) {
	entries, err := status(root)
	if err != nil {
		return nil, err
	}
	prefix, err := showPrefix(root)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, e := range entries {
		for _, p := range e.paths { // porcelain paths are relative to the top of the work tree
			if rel, ok := strings.CutPrefix(p, prefix); ok && rel != "" {
				set[rel] = true
			}
		}
	}
	return sorted(set), nil
}

// StatusCounts counts, below root, the entries of git status for tracked
// files (changed, added, deleted, unmerged: a rename is its two paths) and
// the untracked files that are not ignored. Optional locks are off, so git
// does not refresh the index: nothing in the repository is written.
func StatusCounts(root string) (changed, untracked int, err error) {
	entries, err := status(root)
	if err != nil {
		return 0, 0, err
	}
	for _, e := range entries {
		if e.kind == '?' {
			untracked++
		} else {
			changed += len(e.paths)
		}
	}
	return changed, untracked, nil
}

func showPrefix(root string) (string, error) {
	out, err := run(root, "rev-parse", "--show-prefix")
	return strings.TrimSpace(out), err
}

// Change is a project-relative path that differs between two trees, and
// whether it is a submodule pointer (a gitlink) on either side.
type Change struct {
	Path    string
	Gitlink bool
}

// GitlinkMode is the mode git records for a submodule pointer.
const GitlinkMode = "160000"

// diffRaw runs diff-tree or diff-index below root and returns every path it
// names, with whether it is a submodule pointer on either side.
func diffRaw(root, command string, args ...string) ([]Change, error) {
	full := append(append(append([]string{command}, diffFlags...), args...), "--", ".")
	out, err := run(root, full...)
	if err != nil {
		return nil, err
	}
	var res []Change
	entries := fields(out)
	for i := 0; i < len(entries); i++ {
		meta := entries[i]
		// :<src mode> <dst mode> <src sha> <dst sha> <status>
		f := strings.Fields(strings.TrimPrefix(meta, ":"))
		if !strings.HasPrefix(meta, ":") || len(f) < 5 {
			return nil, fmt.Errorf("git %s: an entry machinery cannot read: %q", command, meta)
		}
		link := f[0] == GitlinkMode || f[1] == GitlinkMode
		n := 1
		if s := f[4][0]; s == 'R' || s == 'C' {
			n = 2
		}
		for ; n > 0; n-- {
			if i++; i >= len(entries) || entries[i] == "" {
				return nil, fmt.Errorf("git %s: %q has no path", command, meta)
			}
			res = append(res, Change{Path: entries[i], Gitlink: link})
		}
	}
	return res, nil
}

// DiffChanges lists the project-relative paths that differ between two
// commits: both paths of a rename or a copy, each with whether it is a
// submodule pointer in either commit.
func DiffChanges(root, a, b string) ([]Change, error) {
	if err := revs(a, b); err != nil {
		return nil, err
	}
	return diffRaw(root, "diff-tree", a, b)
}

// StagedSince lists, sorted and each once, the project-relative paths that
// differ between a commit and the index: what is committed since it and
// what is staged, both paths of a rename or a copy.
func StagedSince(root, commit string) ([]string, error) {
	if err := revs(commit); err != nil {
		return nil, err
	}
	changes, err := diffRaw(root, "diff-index", "--cached", commit)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, c := range changes {
		set[c.Path] = true
	}
	return sorted(set), nil
}

// ChangedSince lists, sorted and each once, the project-relative paths that
// differ between a ref and the working tree: what is committed or staged
// since the ref, and what git status reports (untracked files and both paths
// of a rename or a copy included).
func ChangedSince(root, ref string) ([]string, error) {
	staged, err := StagedSince(root, ref)
	if err != nil {
		return nil, err
	}
	st, err := StatusPaths(root)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, p := range append(staged, st...) {
		set[p] = true
	}
	return sorted(set), nil
}

func sorted(set map[string]bool) []string {
	res := make([]string, 0, len(set))
	for p := range set {
		res = append(res, p)
	}
	sort.Strings(res)
	return res
}

// Gitlinks lists the project-relative paths the index records as submodule
// pointers.
func Gitlinks(root string) ([]string, error) {
	out, err := run(root, "ls-files", "--stage", "-z", "--", ".")
	if err != nil {
		return nil, err
	}
	var res []string
	for _, e := range fields(out) {
		// <mode> <sha> <stage>\t<path>
		meta, p, ok := strings.Cut(e, "\t")
		if !ok {
			return nil, fmt.Errorf("git ls-files: an entry machinery cannot read: %q", e)
		}
		if strings.HasPrefix(meta, GitlinkMode+" ") {
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
	if _, err := os.Lstat(file); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	out, err := run(root, "config", "--file", file, "--null", "--list")
	if err != nil {
		return nil, fmt.Errorf("git config --file .gitmodules: %v", err)
	}
	prefix, err := showPrefix(root)
	if err != nil {
		return nil, err
	}
	var res []string
	for _, e := range fields(out) {
		// <key>\n<value>
		key, p, ok := strings.Cut(e, "\n")
		if !ok || !strings.HasPrefix(key, "submodule.") || !strings.HasSuffix(key, ".path") {
			continue
		}
		if rel, in := strings.CutPrefix(path.Clean(p), prefix); in && rel != "" {
			res = append(res, rel)
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

// Show returns the content of a project-relative file at a ref, as it is
// committed. ok is false when the ref has no file at that path (a directory
// or a submodule there is none). A ref git cannot read is an error.
func Show(root, ref, rel string) (data []byte, ok bool, err error) {
	if err := revs(ref); err != nil {
		return nil, false, err
	}
	out, err := run(root, "ls-tree", "-z", ref, "--", rel)
	if err != nil {
		return nil, false, err
	}
	for _, e := range fields(out) {
		// <mode> <type> <sha>\t<path>
		meta, p, _ := strings.Cut(e, "\t")
		f := strings.Fields(meta)
		if p != rel || len(f) != 3 || f[1] != "blob" {
			continue
		}
		data, err := git(root, options{}, "cat-file", "blob", f[2])
		if err != nil {
			return nil, false, err
		}
		return data, true, nil
	}
	return nil, false, nil
}

// MergeBase returns the merge base of two refs, "" when they have no common
// commit.
func MergeBase(root, a, b string) (string, error) {
	if err := revs(a, b); err != nil {
		return "", err
	}
	out, err := run(root, "merge-base", a, b)
	if isNo(err) && strings.TrimSpace(out) == "" {
		return "", nil
	}
	return strings.TrimSpace(out), err
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
	if err := revs(a, b); err != nil {
		return 0, err
	}
	out, err := run(root, "rev-list", "--count", a+".."+b)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(out))
}

// IsAncestor reports whether commit a is an ancestor of b (or equal). A
// commit git does not have is an error.
func IsAncestor(root, a, b string) (bool, error) {
	if err := revs(a, b); err != nil {
		return false, err
	}
	_, err := run(root, "merge-base", "--is-ancestor", a, b)
	if isNo(err) {
		return false, nil
	}
	return err == nil, err
}

// Newest returns the newest of the commits by commit date, as git rev-list
// --no-walk orders them, or "" for none.
func Newest(root string, shas []string) (string, error) {
	if len(shas) == 0 {
		return "", nil
	}
	if err := revs(shas...); err != nil {
		return "", err
	}
	out, err := run(root, append([]string{"rev-list", "--no-walk"}, shas...)...)
	if err != nil {
		return "", err
	}
	if ls := lines(out); len(ls) > 0 {
		return ls[0], nil
	}
	return "", nil
}

// CommitsTouching lists commits in (from, HEAD] that touch any of the
// pathspecs, newest first.
func CommitsTouching(root, from string, paths []string) ([]string, error) {
	if err := revs(from); err != nil {
		return nil, err
	}
	out, err := run(root, append([]string{"rev-list", from + "..HEAD", "--"}, paths...)...)
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

// FilesOfCommit lists the project-relative paths a commit changed, both
// paths of a rename; of a merge, the paths that differ from every parent.
func FilesOfCommit(root, sha string) ([]string, error) {
	if err := revs(sha); err != nil {
		return nil, err
	}
	args := append(append([]string{"diff-tree", "-r", "-z", "--root", "--cc", "--no-commit-id", "--name-only", "--relative",
		"--no-ext-diff", "--no-textconv", "--no-color"}, compared...), sha, "--", ".")
	out, err := run(root, args...)
	if err != nil {
		return nil, err
	}
	return fields(out), nil
}

// UserName returns git config user.name, "" when it is not set.
func UserName(root string) (string, error) {
	out, err := run(root, "config", "--get", "user.name")
	if isNo(err) {
		return "", nil
	}
	return strings.TrimSpace(out), err
}

// Ignored reports whether git ignores a project-relative path.
func Ignored(root, rel string) (bool, error) {
	_, err := run(root, "check-ignore", "-q", "--", rel)
	if isNo(err) {
		return false, nil
	}
	return err == nil, err
}

// Tracked returns the files git tracks below root that match the
// pathspecs, root-relative.
func Tracked(root string, pathspecs ...string) ([]string, error) {
	out, err := run(root, append([]string{"ls-files", "-z", "--"}, pathspecs...)...)
	if err != nil {
		return nil, err
	}
	return fields(out), nil
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
	for _, f := range fields(out) {
		if !seen[f] {
			seen[f] = true
			res = append(res, f)
		}
	}
	return res, nil
}

// IsRepo reports whether root is inside a git work tree. Outside a
// repository it is false; any other failure (git missing, a repository git
// refuses to read) is an error.
func IsRepo(root string) (bool, error) {
	out, err := run(root, "rev-parse", "--is-inside-work-tree")
	var e *Error
	if errors.As(err, &e) && e.Code == 128 && strings.Contains(e.Msg, "not a git repository") {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "true", nil
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

// TopLevel returns the top of the git work tree that holds dir.
func TopLevel(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--show-toplevel")
	return strings.TrimSpace(out), err
}

// FirstAdd returns the first commit that adds a project-relative path, or ""
// when no commit does: the oldest commit of rev-list HEAD -- <path> whose
// diff-tree --diff-filter=A names it.
func FirstAdd(root, rel string) (string, error) {
	out, err := run(root, "rev-list", "--reverse", "HEAD", "--", rel)
	if err != nil {
		return "", err
	}
	for _, sha := range lines(out) {
		args := append(append([]string{"diff-tree", "-r", "-z", "--root", "--no-commit-id", "--name-only", "--diff-filter=A"}, compared...), sha, "--", rel)
		names, err := run(root, args...)
		if err != nil {
			return "", err
		}
		if names != "" {
			return sha, nil
		}
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

// treeEntry is one entry of git ls-tree -r: its mode, its type (blob or
// commit, a submodule pointer), its object and its path.
type treeEntry struct {
	mode, typ, oid, name string
}

// Archive writes the tree of HEAD below root into dest: git ls-tree -r for
// the entries and git cat-file --batch for their contents, with no worktree,
// no checkout and no attributes, so every file is written as it is
// committed and a submodule is an empty directory. Every write goes through
// an os.Root bound to dest, so nothing lands outside it. A symbolic link
// whose target is absolute, or resolves outside dest, is not created:
// outside lists those links, sorted, by path. An entry whose name is
// absolute or holds a ".." segment stops the extraction. files counts the
// files and links written.
func Archive(root, dest string) (files int, outside []string, err error) {
	out, err := run(root, "ls-tree", "-r", "-z", "HEAD")
	if err != nil {
		return 0, nil, err
	}
	var entries []treeEntry
	var want strings.Builder
	for _, e := range fields(out) {
		meta, name, ok := strings.Cut(e, "\t")
		f := strings.Fields(meta)
		if !ok || len(f) != 3 {
			return 0, nil, fmt.Errorf("git ls-tree: an entry machinery cannot read: %q", e)
		}
		entries = append(entries, treeEntry{mode: f[0], typ: f[1], oid: f[2], name: name})
		if f[1] == "blob" {
			want.WriteString(f[2] + "\n")
		}
	}
	pr, pw := io.Pipe()
	done := make(chan error, 1)
	go func() {
		_, err := git(root, options{stdin: strings.NewReader(want.String()), stdout: pw}, "cat-file", "--batch")
		pw.CloseWithError(err)
		done <- err
	}()
	br := bufio.NewReader(pr)
	files, outside, xerr := extract(entries, br, dest)
	io.Copy(io.Discard, br)
	if werr := <-done; werr != nil {
		return files, outside, fmt.Errorf("git cat-file: %v", werr)
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

// blob reads the next object of git cat-file --batch, which must be oid, and
// copies its content to w.
func blob(r *bufio.Reader, oid string, w io.Writer) error {
	head, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("git cat-file: %v", err)
	}
	f := strings.Fields(head)
	if len(f) != 3 || f[0] != oid || f[1] != "blob" {
		return fmt.Errorf("git cat-file: %q where blob %s was expected", strings.TrimSpace(head), oid)
	}
	size, err := strconv.ParseInt(f[2], 10, 64)
	if err != nil {
		return fmt.Errorf("git cat-file: %q: %v", strings.TrimSpace(head), err)
	}
	if _, err := io.CopyN(w, r, size); err != nil {
		return fmt.Errorf("git cat-file: %v", err)
	}
	if b, err := r.ReadByte(); err != nil || b != '\n' {
		return fmt.Errorf("git cat-file: blob %s does not end where its size says", oid)
	}
	return nil
}

func extract(entries []treeEntry, r *bufio.Reader, dest string) (int, []string, error) {
	rt, err := os.OpenRoot(dest)
	if err != nil {
		return 0, nil, err
	}
	defer rt.Close()
	n := 0
	var outside, links []string
	for _, e := range entries {
		raw := strings.TrimSuffix(e.name, "/")
		if path.IsAbs(raw) || filepath.IsAbs(raw) || containsDotDot(raw) {
			return n, outside, fmt.Errorf("tree entry %q is absolute or holds ..; nothing outside the snapshot is written", e.name)
		}
		name := path.Clean(raw)
		if name == "." || name == "" {
			return n, outside, fmt.Errorf("tree entry %q has no name", e.name)
		}
		parent := path.Dir(name)
		switch {
		case e.typ == "commit": // a submodule pointer: an empty directory
			if err := rt.MkdirAll(name, 0o755); err != nil {
				return n, outside, err
			}
		case e.typ == "blob" && e.mode == "120000":
			var target bytes.Buffer
			if err := blob(r, e.oid, &target); err != nil {
				return n, outside, err
			}
			if leaves(name, target.String()) {
				outside = append(outside, name)
				continue
			}
			if err := rt.MkdirAll(parent, 0o755); err != nil {
				return n, outside, err
			}
			if err := rt.Symlink(target.String(), name); err != nil {
				return n, outside, err
			}
			links = append(links, name)
			n++
		case e.typ == "blob":
			if err := rt.MkdirAll(parent, 0o755); err != nil {
				return n, outside, err
			}
			mode := os.FileMode(0o644)
			if e.mode == "100755" {
				mode = 0o755
			}
			f, err := rt.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				return n, outside, err
			}
			cerr := blob(r, e.oid, f)
			if err := f.Close(); cerr == nil {
				cerr = err
			}
			if cerr != nil {
				return n, outside, cerr
			}
			n++
		default:
			return n, outside, fmt.Errorf("tree entry %q: git ls-tree type %s machinery cannot write", e.name, e.typ)
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
		if _, err := git(dir, options{env: env}, append(append([]string{}, base...), args...)...); err != nil {
			return fmt.Errorf("git %s in the snapshot: %v", args[0], err)
		}
	}
	return nil
}
