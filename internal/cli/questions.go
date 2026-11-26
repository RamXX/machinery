package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/pm"
)

func init() {
	Register(&Command{Name: "questions", Summary: "List the open questions for the owner and what each waits for.", Run: runQuestions})
}

func runQuestions(e *Env, args []string) int {
	fs := e.Flags("questions")
	held := fs.Bool("held", false, "only the questions a hold left to the owner")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery questions takes no arguments")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	c, err := backlog.New(p, "")
	if err != nil {
		return e.Fail(err)
	}
	issues, err := c.List()
	if err != nil {
		return e.Fail(err)
	}
	open, nHeld := 0, 0
	for _, is := range pm.OpenQuestions(issues) {
		t := backlog.ParseTags(&is)
		open++
		if pm.Held(t) {
			nHeld++
		} else if *held {
			continue
		}
		printQuestion(e.Stdout, &is, t)
	}
	if *held {
		fmt.Fprintf(e.Stdout, "questions: %d held\n", nHeld)
	} else {
		fmt.Fprintf(e.Stdout, "questions: %d open, %d held\n", open, nHeld)
	}
	return 0
}

// printQuestion writes the three lines of one question (section 15.9).
func printQuestion(w io.Writer, is *backlog.Issue, t backlog.Tags) {
	title := is.Title
	if t.StoryOf != "" {
		title += " (story " + t.StoryOf + ")"
	}
	rec := ""
	if t.Recommend >= 1 && t.Recommend <= len(t.Options) {
		rec = t.Options[t.Recommend-1]
	}
	fmt.Fprintf(w, "%s  %s\n", is.ID, title)
	fmt.Fprintf(w, "  recommended: %s; default: %s\n", strings.TrimSpace(fmt.Sprintf("%d %s", t.Recommend, rec)), t.Default)
	fmt.Fprintf(w, "  waits for: %s\n", pm.WaitsFor(t))
}

// questionCount is the line status prints while a question is open, or "".
func questionCount(issues []backlog.Issue) string {
	open, held := 0, 0
	for _, is := range pm.OpenQuestions(issues) {
		open++
		if pm.Held(backlog.ParseTags(&is)) {
			held++
		}
	}
	if open == 0 {
		return ""
	}
	return fmt.Sprintf("questions: %d open, %d held", open, held)
}
