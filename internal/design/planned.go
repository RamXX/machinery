package design

import (
	"regexp"
	"strings"

	"github.com/RamXX/machinery/internal/project"
)

// PlannedMarks reads the project's own "planned" marks (import.planned,
// section 2) and returns, for each id whose source carries one, the story
// the mark names. The first entry that matches an id decides.
func (d *Design) PlannedMarks() map[string]string {
	res := map[string]string{}
	for _, pl := range d.P.Cfg.ImportedPlanned() {
		re, err := pl.Regexp()
		if err != nil {
			continue // a config error, refused when the config is loaded
		}
		mark := func(id string, texts ...string) {
			if _, have := res[id]; have {
				return
			}
			for _, t := range texts {
				if story := storyOf(re, strings.TrimSpace(t)); story != "" {
					res[id] = story
					return
				}
			}
		}
		switch pl.In {
		case project.PlannedInStatement:
			for _, m := range d.Models {
				for _, inv := range m.Invariants {
					mark("inv:"+inv.ID, inv.Statement)
				}
				for _, e := range m.Entities {
					for _, inv := range e.Invariants {
						mark("inv:"+inv.ID, inv.Statement)
					}
				}
			}
		case project.PlannedInFailureRow:
			for _, fm := range d.FailureModes {
				if fm.Imported {
					mark("fm:"+fm.ID, fm.Cells...)
				}
			}
		}
	}
	return res
}

// storyOf returns what the group story matched in the text, or "".
func storyOf(re *regexp.Regexp, text string) string {
	m := re.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[re.SubexpIndex("story")])
}
