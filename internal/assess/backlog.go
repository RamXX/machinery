package assess

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/gitx"
)

// exportFile is the tracker's export, read as a file (section 18.8).
const exportFile = ".beads/issues.jsonl"

type exportIssue struct {
	Type      *string `json:"_type"`
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Status    string  `json:"status"`
	IssueType string  `json:"issue_type"`
	Desc      string  `json:"description"`
	Design    string  `json:"design"`
	Notes     string  `json:"notes"`
	Accept    string  `json:"acceptance_criteria"`
}

var tokenRe = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_.:-]*[A-Za-z0-9]`)

// notOpen are the statuses that do not count as open.
var notOpen = map[string]bool{"closed": true, "deferred": true, "tombstone": true}

func joinCounts(m map[string]int) string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s %d", k, m[k])
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// backlog reads the tracker export of the project work tree, never running
// the tracker, and lists the other local branches.
func (a *assessment) backlog(s *section) {
	var branches []string
	if all, err := gitx.Branches(a.root); err == nil {
		cur := gitx.Branch(a.root)
		for _, b := range all {
			if b != cur {
				branches = append(branches, b)
			}
		}
	}
	if branches == nil {
		branches = []string{}
	}
	branchLine := "branches: no other local branch"
	if len(branches) > 0 {
		branchLine = fmt.Sprintf("branches: %d other local branch(es): %s", len(branches), strings.Join(branches, ", "))
	}
	data, err := os.ReadFile(filepath.Join(a.root, filepath.FromSlash(exportFile)))
	if err != nil {
		s.own.NotApplicable("none found (no " + exportFile + ")")
		s.lines = []string{"backlog: none found", branchLine}
		s.facts = map[string]any{"export": "", "branches": branches}
		return
	}
	d := a.ctx.Design()
	ids := map[string]bool{}
	for _, o := range d.Obligations {
		ids[o.ID] = true
	}
	own := map[string]bool{}
	for _, fm := range d.FailureModes {
		if fm.Imported {
			own[strings.ToLower(fm.ID)] = true
		}
	}
	names := func(texts ...string) bool {
		for _, t := range texts {
			for _, tok := range tokenRe.FindAllString(t, -1) {
				if ids[tok] || own[strings.ToLower(tok)] {
					return true
				}
			}
		}
		return false
	}
	byStatus, byType := map[string]int{}, map[string]int{}
	var inProgress []exportIssue
	issues, open, naming := 0, 0, 0
	for n, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var is exportIssue
		if err := json.Unmarshal([]byte(line), &is); err != nil {
			s.own.Errorf("%s:%d: not a JSON object", exportFile, n+1)
			continue
		}
		if is.Type != nil && *is.Type != "issue" {
			continue
		}
		issues++
		byStatus[is.Status]++
		t := is.IssueType
		if t == "" {
			t = "(none)"
		}
		byType[t]++
		if is.Status == "in_progress" {
			inProgress = append(inProgress, is)
		}
		if !notOpen[is.Status] {
			open++
			if names(is.Title, is.Desc, is.Design, is.Notes, is.Accept) {
				naming++
			}
		}
	}
	sort.Slice(inProgress, func(i, j int) bool { return inProgress[i].ID < inProgress[j].ID })
	s.lines = append(s.lines,
		fmt.Sprintf("export: %s, %d issues", exportFile, issues),
		"by status: "+joinCounts(byStatus),
		"by type: "+joinCounts(byType),
		fmt.Sprintf("in progress: %d", len(inProgress)))
	progress := []any{}
	for _, is := range inProgress {
		s.lines = append(s.lines, fmt.Sprintf("  %s  %s", is.ID, is.Title))
		progress = append(progress, map[string]any{"id": is.ID, "title": is.Title})
	}
	s.lines = append(s.lines, fmt.Sprintf("open issues naming an obligation id: %d of %d", naming, open), branchLine)
	s.own.Count(issues, "issues").Count(open, "open").Count(naming, "naming an obligation")
	s.facts = map[string]any{"export": exportFile, "issues": issues, "by_status": byStatus, "by_type": byType,
		"in_progress": progress, "open": open, "open_naming_obligations": naming, "branches": branches}
}
