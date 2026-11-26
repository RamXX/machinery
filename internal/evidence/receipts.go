// Package evidence joins the design to receipts from real test runs
// (docs/formats.md sections 10, 10.1 and 12). Every command that judges
// evidence (verify, owed sync, proof, status, probe) uses this package.
package evidence

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
)

// Outcomes and replay values of a receipt (section 10).
const (
	Pass       = "pass"
	Fail       = "fail"
	Skip       = "skip"
	Reproduced = "reproduced"
	Refuted    = "refuted"
)

// Receipt is one receipt line.
type Receipt struct {
	ID       string
	Outcome  string
	Test     string
	Effects  []string
	Observed string
	Replay   string
	// Source is "adapter" for a receipt a run adapter wrote, with its
	// Granularity (function or file) and the build Tags of the run
	// (section 10.3); "" for a receipt a test emitted.
	Source      string
	Granularity string
	Tags        any    // a list of build tags, or "unknown"
	File        string // project-relative receipts file
}

// RunInfo describes one complete receipts file that was used as evidence.
type RunInfo struct {
	File     string // project-relative
	Commit   string
	Command  []string
	Stack    string
	Exit     int
	Dirty    bool
	Receipts int
}

// line is any line of a receipts file.
type line struct {
	V        int      `json:"v"`
	Type     string   `json:"type"`
	Commit   string   `json:"commit"`
	Dirty    bool     `json:"dirty"`
	Command  []string `json:"command"`
	Stack    string   `json:"stack"`
	ID       string   `json:"id"`
	Outcome  string   `json:"outcome"`
	Test     string   `json:"test"`
	Effects  []string `json:"effects"`
	Observed string   `json:"observed"`
	Replay   string   `json:"replay"`
	Exit     *int     `json:"exit"`

	Source      string `json:"source"`
	Granularity string `json:"granularity"`
	Tags        any    `json:"tags"`
	// Adapter names the run adapter in a header; AdapterReceipts counts, on
	// the end line, the lines that adapter wrote right before it (section 10.3)
	Adapter         any  `json:"adapter"`
	AdapterReceipts *int `json:"adapter_receipts"`
}

// fileData is a parsed receipts file.
type fileData struct {
	Rel      string
	Header   *line // nil: not a receipts file
	End      *line // nil: incomplete run
	Receipts []Receipt
	Problems []string
	// Sum is the digest of the whole file, compared with the run ledger
	// (section 10.2).
	Sum string
}

func parseFile(abs, rel string) (*fileData, error) {
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	fd := &fileData{Rel: rel, Sum: digest(data)}
	type problem struct {
		n   int
		msg string
	}
	var problems []problem
	addProblem := func(n int, format string, a ...any) {
		problems = append(problems, problem{n, fmt.Sprintf("line %d: ", n) + fmt.Sprintf(format, a...)})
	}
	// receipts are kept until the end line says which lines the adapter wrote
	type parsed struct {
		n       int
		r       Receipt
		adapter bool // has source, granularity or tags
	}
	var recs []parsed
	var order []int // the lines after the header that are not blank
	endAt := 0
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 16*1024*1024)
	n := 0
	for sc.Scan() {
		n++
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			continue
		}
		if n > 1 {
			order = append(order, n)
		}
		var l line
		if err := json.Unmarshal([]byte(text), &l); err != nil {
			if n == 1 {
				return fd, nil
			}
			addProblem(n, "not a JSON object")
			continue
		}
		if n == 1 {
			if l.Type != "run" || l.V != 1 {
				return fd, nil
			}
			fd.Header = &l
			continue
		}
		if l.V != 1 {
			addProblem(n, "v must be 1")
			continue
		}
		switch l.Type {
		case "receipt":
			if msg := checkReceipt(&l); msg != "" {
				addProblem(n, "%s", msg)
				continue
			}
			recs = append(recs, parsed{n, Receipt{ID: l.ID, Outcome: l.Outcome, Test: l.Test,
				Effects: l.Effects, Observed: l.Observed, Replay: l.Replay,
				Source: l.Source, Granularity: l.Granularity, Tags: l.Tags, File: rel},
				l.Source != "" || l.Granularity != "" || l.Tags != nil})
		case "end":
			if l.Exit == nil {
				addProblem(n, "end line without exit")
				continue
			}
			end := l
			fd.End = &end
			endAt = n
		default:
			addProblem(n, "unknown type %q", l.Type)
		}
	}
	// the lines the adapter wrote: those its end line counts, right before it,
	// in a run whose header names the adapter (section 10.3)
	byAdapter := map[int]bool{}
	if fd.Header != nil && fd.Header.Adapter == GoTestJSON && fd.End != nil && fd.End.AdapterReceipts != nil {
		var before []int
		for _, k := range order {
			if k < endAt {
				before = append(before, k)
			}
		}
		m := min(max(*fd.End.AdapterReceipts, 0), len(before))
		for _, k := range before[len(before)-m:] {
			byAdapter[k] = true
		}
	}
	for _, p := range recs {
		if p.adapter && !byAdapter[p.n] {
			addProblem(p.n, "%s: source, granularity and tags are written only by the run adapter: a test that emits receipts leaves them out", p.r.ID)
			continue
		}
		fd.Receipts = append(fd.Receipts, p.r)
	}
	sort.SliceStable(problems, func(i, j int) bool { return problems[i].n < problems[j].n })
	for _, p := range problems {
		fd.Problems = append(fd.Problems, p.msg)
	}
	return fd, sc.Err()
}

func checkReceipt(l *line) string {
	switch {
	case l.ID == "":
		return "receipt without id"
	case l.Outcome != Pass && l.Outcome != Fail && l.Outcome != Skip:
		return fmt.Sprintf("%s: outcome must be pass, fail or skip, got %q", l.ID, l.Outcome)
	case l.Test == "":
		return fmt.Sprintf("%s: receipt without test: name the test file in test", l.ID)
	case l.Replay != "" && l.Replay != Reproduced && l.Replay != Refuted:
		return fmt.Sprintf("%s: replay must be reproduced or refuted, got %q", l.ID, l.Replay)
	case l.Source != "" && l.Source != SourceAdapter:
		return fmt.Sprintf("%s: source must be adapter or absent, got %q", l.ID, l.Source)
	case l.Source == SourceAdapter && l.Granularity != project.GranularityFunction && l.Granularity != project.GranularityFile:
		return fmt.Sprintf("%s: granularity must be function or file, got %q", l.ID, l.Granularity)
	case l.Source == "" && l.Granularity != "":
		return fmt.Sprintf("%s: granularity without source: adapter", l.ID)
	}
	return ""
}

// Dir returns the project-relative receipts directory without a trailing slash.
func Dir(p *project.Project) string { return strings.TrimSuffix(p.Cfg.Receipts.Dir, "/") }

// receiptFiles lists the *.jsonl files of the receipts directory, sorted by name.
func receiptFiles(p *project.Project) ([]string, error) {
	entries, err := os.ReadDir(p.Abs(Dir(p)))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var res []string
	for _, e := range entries {
		if e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".jsonl") {
			res = append(res, Dir(p)+"/"+e.Name())
		}
	}
	sort.Strings(res)
	return res, nil
}

// TreeDirty reports whether the working tree has changes outside the receipts
// directory and the owed file (section 10).
func TreeDirty(p *project.Project) (bool, error) {
	paths, err := gitx.StatusPaths(p.Root)
	if err != nil {
		return false, err
	}
	for _, rel := range paths {
		if !exempt(p, rel) {
			return true, nil
		}
	}
	return false, nil
}

// exempt reports whether a change to a path leaves evidence valid (sections
// 10 and 10.1): the owed file, a receipts file, or a property record of a
// vital area, whose statuses render the owed file (section 23.4).
func exempt(p *project.Project, rel string) bool {
	return rel == p.Cfg.Design.Owed || isReceipt(p, rel) || isVitalRecord(p, rel)
}

// isVitalRecord reports whether a path is a property record: a .json file
// directly in <gen>/vital.
func isVitalRecord(p *project.Project, rel string) bool {
	return strings.HasSuffix(rel, ".json") && path.Dir(rel) == strings.TrimSuffix(p.Cfg.Design.Gen, "/")+"/vital"
}

// isReceipt reports whether a project-relative path is a receipts file: a
// .jsonl file directly in the receipts directory. Nothing else in that
// directory is exempt from the dirty check or from the validity rule.
func isReceipt(p *project.Project, rel string) bool {
	return strings.HasSuffix(rel, ".jsonl") && path.Dir(rel) == Dir(p)
}

// validity decides, per commit, whether evidence from that commit is valid for
// the head (section 10.1).
type validity struct {
	p     *project.Project
	head  string
	cache map[string]bool
}

// ValidFor reports whether evidence from commit is valid for head (section 10.1).
func ValidFor(p *project.Project, head, commit string) bool {
	return newValidity(p, head).valid(commit)
}

func newValidity(p *project.Project, head string) *validity {
	return &validity{p: p, head: head, cache: map[string]bool{head: true}}
}

func (v *validity) valid(commit string) bool {
	if ok, seen := v.cache[commit]; seen {
		return ok
	}
	ok := false
	if commit != "" && gitx.Exists(v.p.Root, commit) {
		names, err := gitx.DiffNames(v.p.Root, commit, v.head)
		if err == nil {
			ok = true
			for _, n := range names {
				if !exempt(v.p, n) {
					ok = false
					break
				}
			}
		}
	}
	v.cache[commit] = ok
	return ok
}
