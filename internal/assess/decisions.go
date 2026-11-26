package assess

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RamXX/machinery/internal/gitx"
)

// census counts the lines, the words and the lines a record expression
// matches.
func census(data []byte, record *regexp.Regexp) (lines, words, records int) {
	text := string(data)
	words = len(strings.Fields(text))
	if text == "" {
		return 0, words, 0
	}
	ls := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	for _, l := range ls {
		if record != nil && record.MatchString(strings.TrimSuffix(l, "\r")) {
			records++
		}
	}
	return len(ls), words, records
}

// instructionNames are the agent instruction files at the root (section 18.7).
var instructionNames = []string{"AGENTS.md", "CLAUDE.md"}

type instruction struct {
	path         string
	lines, words int
	linked       []string // the files that link to this one
}

// instructions counts the agent instruction files at root, once each: a
// file that is a symbolic link to the other, or a single line naming it, is
// counted under the file it points at.
func instructions(root string) []instruction {
	present := map[string]bool{}
	pointsTo := map[string]string{}
	for _, name := range instructionNames {
		if _, err := os.Lstat(filepath.Join(root, name)); err == nil {
			present[name] = true
		}
	}
	for _, name := range instructionNames {
		if !present[name] {
			continue
		}
		other := instructionNames[0]
		if other == name {
			other = instructionNames[1]
		}
		if !present[other] {
			continue
		}
		path := filepath.Join(root, name)
		info, _ := os.Lstat(path)
		if info.Mode()&os.ModeSymlink != 0 {
			if target, err := os.Readlink(path); err == nil && filepath.Clean(target) == other {
				pointsTo[name] = other
			}
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var nonEmpty []string
		for _, l := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(l) != "" {
				nonEmpty = append(nonEmpty, l)
			}
		}
		if len(nonEmpty) == 1 && strings.Contains(nonEmpty[0], other) {
			pointsTo[name] = other
		}
	}
	// two files that point at each other are both counted
	for name, other := range pointsTo {
		if pointsTo[other] == name {
			delete(pointsTo, name)
			delete(pointsTo, other)
		}
	}
	var res []instruction
	for _, name := range instructionNames {
		if !present[name] || pointsTo[name] != "" {
			continue
		}
		path := filepath.Join(root, name)
		real, err := filepath.EvalSymlinks(path)
		if err != nil || !gitx.Within(real, gitx.RealPath(root)) {
			continue
		}
		data, err := os.ReadFile(real)
		if err != nil {
			continue
		}
		in := instruction{path: name}
		in.lines, in.words, _ = census(data, nil)
		for _, from := range instructionNames {
			if pointsTo[from] == name {
				in.linked = append(in.linked, from)
			}
		}
		res = append(res, in)
	}
	return res
}

// decisions: the census of a project's decision records and of its agent
// instruction files.
func (a *assessment) decisions(s *section) {
	var files []string
	var re *regexp.Regexp
	dec := a.p.Cfg.Assess
	configured := dec != nil && dec.Decisions != nil
	if configured {
		var err error
		if re, err = dec.Decisions.RecordPattern(); err != nil {
			s.own.Errorf("assess.decisions.record: %v", err)
			return
		}
		if files, err = a.p.Glob(dec.Decisions.Files); err != nil {
			s.own.Errorf("cannot list files: %v", err)
			return
		}
	}
	inst := instructions(a.p.Root)
	if !configured && len(inst) == 0 {
		s.own.NotApplicable("no assess.decisions configured and no CLAUDE.md or AGENTS.md at the root")
		return
	}
	if configured && len(files) == 0 {
		s.own.Errorf("assess.decisions.files matches no file")
	}
	records := 0
	fileFacts := []any{}
	for _, f := range files {
		data, err := a.p.Read(f)
		if err != nil {
			s.own.Errorf("%s: cannot read", f)
			continue
		}
		l, w, r := census(data, re)
		records += r
		if r == 0 {
			s.own.Warnf("%s: no line matches assess.decisions.record", f)
		}
		s.lines = append(s.lines, fmt.Sprintf("%s: %d records, %d lines, %d words", f, r, l, w))
		fileFacts = append(fileFacts, map[string]any{"path": f, "records": r, "lines": l, "words": w})
	}
	instFacts := []any{}
	for _, in := range inst {
		line := fmt.Sprintf("%s: %d lines, %d words", in.path, in.lines, in.words)
		for _, from := range in.linked {
			line += " (" + from + " links to it)"
		}
		s.lines = append(s.lines, line)
		linked := in.linked
		if linked == nil {
			linked = []string{}
		}
		instFacts = append(instFacts, map[string]any{"path": in.path, "lines": in.lines, "words": in.words, "linked": linked})
	}
	s.own.Count(len(files), "decision files").Count(records, "records").Count(len(inst), "instruction files")
	s.facts = map[string]any{"files": fileFacts, "instructions": instFacts}
}
