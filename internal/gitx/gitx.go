// Package gitx runs the few git commands machinery needs.
package gitx

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
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
