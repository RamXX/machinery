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
// to root. Untracked files are listed one by one.
func StatusPaths(root string) ([]string, error) {
	out, err := run(root, "status", "--porcelain", "--untracked-files=all", "--", ".")
	if err != nil {
		return nil, err
	}
	top, err := run(root, "rev-parse", "--show-prefix")
	if err != nil {
		return nil, err
	}
	prefix := strings.TrimSpace(top)
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 4 {
			continue
		}
		p := line[3:]
		if i := strings.Index(p, " -> "); i >= 0 {
			p = p[i+4:]
		}
		p = strings.Trim(p, "\"")
		p = strings.TrimPrefix(p, prefix)
		paths = append(paths, p)
	}
	return paths, nil
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

// DiffNames lists project-relative paths that differ between two commits.
func DiffNames(root, a, b string) ([]string, error) {
	out, err := run(root, "diff", "--name-only", "--relative", a, b, "--", ".")
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

// ChangedSince lists project-relative paths that differ between a ref and the
// working tree, untracked files included.
func ChangedSince(root, ref string) ([]string, error) {
	out, err := run(root, "diff", "--name-only", "--relative", ref, "--", ".")
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, p := range lines(out) {
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

// IsAncestor reports whether a is an ancestor of b (or equal).
func IsAncestor(root, a, b string) bool {
	_, err := run(root, "merge-base", "--is-ancestor", a, b)
	return err == nil
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

// FilesOfCommit lists project-relative paths a commit changed.
func FilesOfCommit(root, sha string) ([]string, error) {
	out, err := run(root, "show", "--name-only", "--relative", "--format=", sha, "--", ".")
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
// to an extraction, with no worktree and no checkout. It returns the number
// of files and symbolic links written.
func Archive(root, dest string) (int, error) {
	cmd := exec.Command("git", "archive", "--format=tar", "HEAD")
	cmd.Dir = root
	var errb bytes.Buffer
	cmd.Stderr = &errb
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	n, xerr := extract(pipe, dest)
	io.Copy(io.Discard, pipe)
	werr := cmd.Wait()
	if werr != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = werr.Error()
		}
		return n, errors.New("git archive: " + msg)
	}
	return n, xerr
}

func extract(r io.Reader, dest string) (int, error) {
	tr := tar.NewReader(r)
	n := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return n, nil
		}
		if err != nil {
			return n, err
		}
		name := path.Clean(h.Name)
		if name == "." {
			continue
		}
		if path.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
			return n, fmt.Errorf("archive entry %q leaves the snapshot", h.Name)
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return n, err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return n, err
			}
			mode := os.FileMode(0o644)
			if h.Mode&0o111 != 0 {
				mode = 0o755
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				return n, err
			}
			_, cerr := io.Copy(f, tr)
			if err := f.Close(); cerr == nil {
				cerr = err
			}
			if cerr != nil {
				return n, cerr
			}
			n++
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return n, err
			}
			if err := os.Symlink(h.Linkname, target); err != nil {
				return n, err
			}
			n++
		}
	}
}

// InitCommit makes dir a repository of its own and commits everything in it,
// ignored files included, with hooks and signing off. The environment of a
// surrounding repository (GIT_DIR and the like) is not inherited.
func InitCommit(dir, message string) error {
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
	for _, args := range [][]string{
		{"init", "-q"},
		{"add", "-A", "-f"},
		{"commit", "-q", "--no-verify", "--allow-empty", "-m", message},
	} {
		cmd := exec.Command("git", append(append([]string{}, base...), args...)...)
		cmd.Dir = dir
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git %s in the snapshot: %s", args[0], strings.TrimSpace(string(out)))
		}
	}
	return nil
}
