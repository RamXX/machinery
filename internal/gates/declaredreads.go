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
		row := declaredRead{artifact: strings.TrimSpace(o.GetString("artifact")), reader: strings.TrimSpace(o.GetString("reader")), reviewed: strings.TrimSpace(o.GetString("reviewed"))}
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
	for label, value := range map[string]string{"artifact": row.artifact, "reader": row.reader} {
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
	for _, path := range []string{artifact, reader} {
		listed, err := runGitExact(repo, "cat-file", "-e", reviewed+":"+path)
		if err != nil || listed != "" {
			return fmt.Errorf("declared-read path %s was not tracked at reviewed commit %s", ir.Repr(path), reviewed)
		}
	}
	log, err := runGitExact(repo, "log", "--format=commit:%H", "--name-only", "--no-renames", reviewed+".."+head, "--", artifact, reader)
	if err != nil {
		return fmt.Errorf("read declared-read change history: %w", err)
	}
	for _, change := range parseDeclaredReadLog(log) {
		if change.paths[artifact] && !change.paths[reader] {
			return fmt.Errorf("declared-read artifact %s changed without reader %s in the same commit %s", row.artifact, row.reader, change.commit)
		}
		if change.paths[artifact] {
			g.Count("paired artifact changes")
		}
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

type declaredReadChange struct {
	commit string
	paths  map[string]bool
}

func parseDeclaredReadLog(text string) []declaredReadChange {
	var out []declaredReadChange
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "commit:") {
			out = append(out, declaredReadChange{commit: strings.TrimPrefix(line, "commit:"), paths: map[string]bool{}})
		} else if line != "" && len(out) > 0 {
			out[len(out)-1].paths[line] = true
		}
	}
	return out
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
