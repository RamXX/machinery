package cli

import (
	"flag"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/pm"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
)

func init() {
	Register(&Command{Name: "story", Summary: "Stage story: the tier gate of one backlog story.", Run: runStory})
}

func runStory(e *Env, args []string) int {
	fs := e.Flags("story")
	pos, ok := parseMixed(e, fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		return e.Usage("machinery story takes one story id")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	c, err := backlog.New(p, "")
	if err != nil {
		return e.Fail(err)
	}
	return storyGate(e, p, c, pos[0])
}

// storyGate runs stage story for one story and prints it.
func storyGate(e *Env, p *project.Project, c *backlog.Client, id string) int {
	is, code := showStory(e, c, id)
	if is == nil {
		return code
	}
	s, err := pm.StoryGate(design.Load(p), is)
	if err != nil {
		return e.Fail(err)
	}
	return report.PrintAll(e.Stdout, "story", []*report.Stage{s})
}

// showStory reads one story; a story bd cannot show is a usage error.
func showStory(e *Env, c *backlog.Client, id string) (*backlog.Issue, int) {
	is, err := c.Show(id)
	if err != nil {
		return nil, e.Usage("no story %s in the backlog: %v", id, err)
	}
	return is, 0
}

// parseMixed parses flags that may come before or after positional arguments.
func parseMixed(e *Env, fs *flag.FlagSet, args []string) ([]string, bool) {
	var pos []string
	for {
		if !e.Parse(fs, args) {
			return nil, false
		}
		if fs.NArg() == 0 {
			return pos, true
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
}
