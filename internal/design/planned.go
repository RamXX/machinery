package design

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/RamXX/machinery/internal/project"
)

// PlannedMarks reads the project's own "planned" marks (import.planned,
// section 2) and returns, for each id whose source carries one, the story
// the mark names. The first entry that matches an id decides. A mark whose
// story is not a story id is a problem, named by its source line, and marks
// nothing: what a pattern captured is never written as it is.
func (d *Design) PlannedMarks() (marks map[string]string, problems []string) {
	marks = map[string]string{}
	refused := map[string]bool{}
	for n, pl := range d.P.Cfg.ImportedPlanned() {
		re, err := pl.Regexp()
		if err != nil {
			continue // a config error, refused when the config is loaded
		}
		mark := func(id, file string, line int, texts ...string) {
			if marks[id] != "" || refused[id] {
				return
			}
			for _, t := range texts {
				story := storyOf(re, strings.TrimSpace(t))
				switch {
				case story == "":
					continue
				case !StoryID.MatchString(story):
					refused[id] = true
					problems = append(problems, fmt.Sprintf("%s:%d: %s: import.planned entry %d captured the story %q; a story id has only letters, digits, \".\", \"-\" and \"_\": tighten the pattern", file, line, id, n+1, story))
				default:
					marks[id] = story
				}
				return
			}
		}
		switch pl.In {
		case project.PlannedInStatement:
			for _, m := range d.Models {
				for _, inv := range m.Invariants {
					mark("inv:"+inv.ID, m.Path, inv.Line, inv.Statement)
				}
				for _, e := range m.Entities {
					for _, inv := range e.Invariants {
						mark("inv:"+inv.ID, m.Path, inv.Line, inv.Statement)
					}
				}
			}
		case project.PlannedInScenario:
			for _, m := range d.Models {
				for _, sc := range m.Scenarios {
					if sc.Description != "" {
						mark("sc:"+Slug(sc.Name), m.Path, sc.Line, sc.Description)
					}
				}
			}
		case project.PlannedInFailureRow:
			for _, fm := range d.FailureModes {
				if fm.Imported {
					mark("fm:"+fm.ID, fm.Source, fm.Line, fm.Cells...)
				}
			}
		}
	}
	return marks, problems
}

// storyOf returns what the group story matched in the text, or "".
func storyOf(re *regexp.Regexp, text string) string {
	m := re.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[re.SubexpIndex("story")])
}
