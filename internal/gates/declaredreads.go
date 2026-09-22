package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/portablepath"
)

var declaredReadCommitRe = regexp.MustCompile(`^[0-9a-f]{40}$`)

type declaredRead struct {
	artifact string
	reader   string
	reviewed string
}

// HasDeclaredReads reports whether the Architecture Contract carries the
// artifact-activated reads list. Schema errors are reported by G2 and Gr.
func HasDeclaredReads(design string) bool {
	data, err := os.ReadFile(filepath.Join(design, "ARCHITECTURE.md"))
	if err != nil {
		return false
	}
	fence, ok := ir.ContractFence(string(data))
	if !ok {
		return false
	}
	value, err := ir.LoadYAML([]byte(fence))
	return err == nil && value.AsObject() != nil && value.AsObject().Get2("reads") != nil
}

// CheckDeclaredReads implements Gr-reads. Without an implementation it keeps
// the declaration visible as a warning. With an implementation it proves that
// every artifact-changing commit since the recorded review also touched the
// declared reader, including the current uncommitted change set.
func CheckDeclaredReads(design, impl, gitDesign, gitImpl string) *Gate {
	g := NewGate("Gr-reads  implementation design-file consumers")
	contract := loadContract(design, filepath.Join(design, "ARCHITECTURE.md"), g)
	if contract == nil {
		return g
	}
	reads := parseDeclaredReads(g, contract)
	g.Count("declared reads", len(reads))
	g.RequireNonzero("declared reads", "Architecture Contract reads declarations")
	seen := map[string]bool{}
	for _, row := range reads {
		key := row.artifact + "\x00" + row.reader
		if seen[key] {
			g.Errs = append(g.Errs, fmt.Sprintf("duplicate reads declaration for artifact %s and reader %s", ir.Repr(row.artifact), ir.Repr(row.reader)))
			continue
		}
		seen[key] = true
		if !validateDeclaredRead(g, design, row) {
			continue
		}
		g.Count("artifacts resolved")
		if impl == "" {
			g.Warns = append(g.Warns, fmt.Sprintf("%s: an implementation reads this file through %s; land any edit with its reader follow-up and run with --impl to verify the review history", row.artifact, row.reader))
			continue
		}
		readerPath := filepath.Join(impl, filepath.FromSlash(row.reader))
		if fi, err := os.Lstat(readerPath); err != nil || !fi.Mode().IsRegular() {
			g.Errs = append(g.Errs, fmt.Sprintf("reader %s does not resolve to a regular file under --impl", ir.Repr(row.reader)))
			continue
		}
		g.Count("readers resolved")
		if err := checkDeclaredReadHistory(gitDesign, gitImpl, row, g); err != nil {
			g.Errs = append(g.Errs, err.Error())
			continue
		}
		g.Count("review histories verified")
	}
	return g
}

func parseDeclaredReads(g *Gate, contract *ir.Value) []declaredRead {
	value := contract.AsObject().Get2("reads")
	if value == nil || value.Kind != ir.KindArray {
		return nil
	}
	var out []declaredRead
	for i, item := range value.AsArray() {
		o := item.AsObject()
		if o == nil {
			continue
		}
		row := declaredRead{artifact: o.GetString("artifact"), reader: o.GetString("reader"), reviewed: o.GetString("reviewed")}
		if row.artifact == "" || row.reader == "" || row.reviewed == "" {
			continue
		}
		if !declaredReadCommitRe.MatchString(row.reviewed) {
			g.Errs = append(g.Errs, fmt.Sprintf("Architecture Contract: reads[%d].reviewed must be a full 40-character lowercase Git commit", i))
			continue
		}
		out = append(out, row)
	}
	return out
}

func validateDeclaredRead(g *Gate, design string, row declaredRead) bool {
	valid := true
	for _, field := range []struct{ label, value string }{{"artifact", row.artifact}, {"reader", row.reader}} {
		label, value := field.label, field.value
		if err := portablepath.ValidateRelative(value); err != nil {
			g.Errs = append(g.Errs, fmt.Sprintf("reads %s %s is not a portable relative path: %v", label, ir.Repr(value), err))
			valid = false
		}
	}
	if !valid {
		return false
	}
	if fi, err := os.Lstat(filepath.Join(design, filepath.FromSlash(row.artifact))); err != nil || !fi.Mode().IsRegular() {
		g.Errs = append(g.Errs, fmt.Sprintf("declared-read artifact %s does not resolve to a regular design file", ir.Repr(row.artifact)))
		return false
	}
	return true
}

func checkDeclaredReadHistory(gitDesign, gitImpl string, row declaredRead, g *Gate) error {
	if gitDesign == "" || gitImpl == "" {
		return fmt.Errorf("declared-read history for %s cannot be verified without logical design and implementation paths", ir.Repr(row.artifact))
	}
	repo, err := runGitExact(gitDesign, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("declared-read history for %s is unavailable: %w", ir.Repr(row.artifact), err)
	}
	artifact, err := repoRelativePath(repo, filepath.Join(gitDesign, filepath.FromSlash(row.artifact)))
	if err != nil {
		return err
	}
	reader, err := repoRelativePath(repo, filepath.Join(gitImpl, filepath.FromSlash(row.reader)))
	if err != nil {
		return err
	}
	head, err := gitHeadAtExact(repo)
	if err != nil {
		return fmt.Errorf("resolve HEAD for declared-read history: %w", err)
	}
	reviewed, err := gitCommitOf(repo, row.reviewed)
	if err != nil {
		return fmt.Errorf("resolve reviewed commit %s: %w", row.reviewed, err)
	}
	if reviewed == "" {
		return fmt.Errorf("declared-read reviewed commit %s does not exist in this repository", row.reviewed)
	}
	ancestor, err := gitIsAncestor(repo, reviewed, head)
	if err != nil {
		return fmt.Errorf("verify declared-read review ancestry: %w", err)
	}
	if !ancestor {
		return fmt.Errorf("declared-read reviewed commit %s is not an ancestor of HEAD %s", reviewed, head)
	}
	commits, err := runGitExact(repo, "rev-list", "--reverse", reviewed+".."+head)
	if err != nil {
		return fmt.Errorf("enumerate declared-read commits: %w", err)
	}
	type parentDiff struct {
		commit  string
		changes []declaredReadFileChange
	}
	var diffs []parentDiff
	for _, commit := range strings.Fields(commits) {
		line, err := runGitExact(repo, "rev-list", "--parents", "-n", "1", commit)
		if err != nil {
			return fmt.Errorf("read parents of declared-read commit %s: %w", commit, err)
		}
		parents := strings.Fields(line)
		if len(parents) < 2 || parents[0] != commit {
			return fmt.Errorf("declared-read commit %s has no verifiable parent", commit)
		}
		for _, parent := range parents[1:] {
			output, err := runGitExact(repo, "diff-tree", "-r", "-M", "--name-status", "-z", parent, commit)
			if err != nil {
				return fmt.Errorf("read declared-read tree diff for %s against %s: %w", commit, parent, err)
			}
			changes, err := parseDeclaredReadDiff(output)
			if err != nil {
				return fmt.Errorf("parse declared-read tree diff for %s: %w", commit, err)
			}
			diffs = append(diffs, parentDiff{commit, changes})
		}
	}
	// Resolve the current reader's custody backwards through actual Git rename
	// edges. A path introduced only after the review cannot stand in for the
	// reviewed reader, even when it changed alongside the artifact.
	readerPaths := map[string]bool{reader: true}
	for changed := true; changed; {
		changed = false
		for _, diff := range diffs {
			for _, change := range diff.changes {
				if strings.HasPrefix(change.status, "R") && readerPaths[change.new] && !readerPaths[change.old] {
					readerPaths[change.old] = true
					changed = true
				}
			}
		}
	}
	if _, err := runGitExact(repo, "cat-file", "-e", reviewed+":"+artifact); err != nil {
		return fmt.Errorf("declared-read path %s was not tracked at reviewed commit %s", ir.Repr(artifact), reviewed)
	}
	readerReviewed := false
	for candidate := range readerPaths {
		if _, err := runGitExact(repo, "cat-file", "-e", reviewed+":"+candidate); err == nil {
			readerReviewed = true
			break
		}
	}
	if !readerReviewed {
		return fmt.Errorf("declared-read reader %s has no tracked custody at reviewed commit %s", ir.Repr(reader), reviewed)
	}
	pairedCommits := map[string]bool{}
	for _, diff := range diffs {
		artifactChanged, readerChanged := false, false
		for _, change := range diff.changes {
			artifactChanged = artifactChanged || change.old == artifact || change.new == artifact
			readerChanged = readerChanged || readerPaths[change.old] || readerPaths[change.new]
		}
		if artifactChanged && !readerChanged {
			return fmt.Errorf("declared-read artifact %s changed without reader %s in the same commit %s", row.artifact, row.reader, diff.commit)
		}
		if artifactChanged {
			pairedCommits[diff.commit] = true
		}
	}
	for range pairedCommits {
		g.Count("paired artifact changes")
	}
	diff, err := runGitExact(repo, "diff", "--name-only", "--no-renames", "HEAD", "--", artifact, reader)
	if err != nil {
		return fmt.Errorf("read declared-read working-tree changes: %w", err)
	}
	changed := map[string]bool{}
	for _, line := range strings.Split(diff, "\n") {
		if line != "" {
			changed[line] = true
		}
	}
	if changed[artifact] && !changed[reader] {
		return fmt.Errorf("declared-read artifact %s changed without reader %s in the same uncommitted change set", row.artifact, row.reader)
	}
	if changed[artifact] {
		g.Count("paired artifact changes")
	}
	return nil
}

type declaredReadFileChange struct{ status, old, new string }

func parseDeclaredReadDiff(output string) ([]declaredReadFileChange, error) {
	if output == "" {
		return nil, nil
	}
	fields := strings.Split(strings.TrimSuffix(output, "\x00"), "\x00")
	var changes []declaredReadFileChange
	for i := 0; i < len(fields); {
		status := fields[i]
		i++
		if status == "" || i >= len(fields) {
			return nil, fmt.Errorf("malformed name-status record")
		}
		old := fields[i]
		i++
		change := declaredReadFileChange{status: status, old: old, new: old}
		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			if i >= len(fields) {
				return nil, fmt.Errorf("incomplete rename record")
			}
			change.new = fields[i]
			i++
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func repoRelativePath(repo, target string) (string, error) {
	absRepo, err := filepath.Abs(repo)
	if err != nil {
		return "", err
	}
	absRepo, err = filepath.EvalSymlinks(absRepo)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	absTarget, err = filepath.EvalSymlinks(absTarget)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRepo, absTarget)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("declared-read path %s is outside repository %s", ir.Repr(absTarget), ir.Repr(absRepo))
	}
	rel = filepath.ToSlash(rel)
	if err := portablepath.ValidateRelative(rel); err != nil {
		return "", fmt.Errorf("declared-read repository path %s is not portable: %w", ir.Repr(rel), err)
	}
	return rel, nil
}
