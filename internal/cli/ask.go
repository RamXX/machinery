package cli

import (
	"flag"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/delegate"
	"github.com/RamXX/machinery/internal/pm"
)

func init() {
	Register(&Command{Name: "ask", Summary: "Record a question for the owner, with options, a recommendation and the default.", Run: runAsk})
}

var topicRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func runAsk(e *Env, args []string) int {
	fs := e.Flags("ask")
	var options, about StringList
	fs.Var(&options, "option", "an answer the owner may give (repeatable, at least two)")
	recommend := fs.Int("recommend", 0, "the number of the option you recommend")
	def := fs.String("default", "", "what happens if nobody answers")
	fs.Var(&about, "about", "a topic of the question (repeatable)")
	amount := fs.String("amount", "", "what the answer may spend or move, a number")
	unit := fs.String("unit", "", "the unit of --amount")
	story := fs.String("story", "", "the story the question holds up")
	actor := fs.String("actor", "", "who asks (default BEADS_ACTOR, else git user.name)")
	pos, ok := parseMixed(e, fs, args)
	if !ok {
		return 2
	}
	set := visited(fs)
	var topics []string
	seen := map[string]bool{}
	for _, t := range about {
		if !topicRe.MatchString(t) {
			return e.Usage("--about %q: a topic is one word of ^[a-z][a-z0-9-]*$", t)
		}
		if !seen[t] {
			seen[t] = true
			topics = append(topics, t)
		}
	}
	sort.Strings(topics)
	switch {
	case len(pos) != 1 || strings.TrimSpace(pos[0]) == "":
		return e.Usage("machinery ask takes one question")
	case len(options) < 2:
		return e.Usage("a question needs at least two --option")
	}
	for _, o := range options {
		if pm.OneLine(o) == "" {
			return e.Usage("--option: an option must not be empty")
		}
	}
	switch {
	case !set["recommend"]:
		return e.Usage("a question needs --recommend <n>: the number of the option you recommend")
	case *recommend < 1 || *recommend > len(options):
		return e.Usage("--recommend %d names no option: give 1 to %d", *recommend, len(options))
	case pm.OneLine(*def) == "":
		return e.Usage("a question needs --default: what happens if nobody answers")
	case (*amount == "") != (*unit == ""):
		return e.Usage("--amount and --unit go together")
	case *amount != "" && !delegate.ValidAmount(*amount):
		return e.Usage("--amount %q is not a number", *amount)
	case *unit != "" && len(strings.Fields(*unit)) != 1:
		return e.Usage("--unit %q is not one word", *unit)
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	c, err := backlog.New(p, *actor)
	if err != nil {
		return e.Fail(err)
	}
	if c.Actor == "" {
		return e.Usage("no actor: pass --actor or set BEADS_ACTOR")
	}
	if *story != "" {
		if is, code := showStory(e, c, *story); is == nil {
			return code
		}
	}
	desc := pm.QuestionDescription(c.Actor, options, *recommend, *def, topics, *amount, strings.TrimSpace(*unit), *story)
	id, err := c.CreateItem(backlog.Item{Title: pm.OneLine(pos[0]), Type: "task", Description: desc})
	if err != nil {
		return e.Fail(err)
	}
	fmt.Fprintf(e.Stdout, "question: %s\n", id)
	return 0
}

// visited returns the names of the flags given on the command line.
func visited(fs *flag.FlagSet) map[string]bool {
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	return set
}
