package cli

import (
	"flag"

	"github.com/RamXX/machinery/v2/internal/backlog"
	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/pm"
	"github.com/RamXX/machinery/v2/internal/project"
	"github.com/RamXX/machinery/v2/internal/report"
	"github.com/RamXX/machinery/v2/internal/stages"
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
	s, err := gateOf(p, c, is)
	if err != nil {
		return e.Fail(err)
	}
	return report.PrintAll(e.Stdout, "story", []*report.Stage{s})
}

// gateOf runs stage story for one story: the design story of an intake is
// judged by its design and its review, every other story by its tier.
func gateOf(p *project.Project, c *backlog.Client, is *backlog.Issue) (*report.Stage, error) {
	if backlog.ParseTags(is).IntakeDesign {
		ctx := &stages.Context{P: p}
		checked := stages.Run(ctx, append(append([]string{}, stages.LintStages...), "gen-drift"))
		var in *pm.Intake
		if id := pm.IntakeOf(is); id != "" {
			if item, err := c.Show(id); err == nil {
				in, _ = pm.ParseIntake(item)
			}
		}
		return pm.DesignStoryGate(ctx.Design(), is, in, checked, p.Cfg.ReviewOn(pm.ReviewDesign)), nil
	}
	issues, err := capstoneBacklog(c, is)
	if err != nil {
		return nil, err
	}
	return pm.StoryGate(design.Load(p), is, issues)
}

// capstoneBacklog reads the backlog when a story is a capstone, whose ids and
// siblings are the other stories of its intake (section 15.6); nil otherwise.
func capstoneBacklog(c *backlog.Client, is *backlog.Issue) ([]backlog.Issue, error) {
	if !backlog.ParseTags(is).Capstone {
		return nil, nil
	}
	return c.List()
}

// storyIDs returns the design ids of a story: its machinery:ids, or for a
// capstone the ids of the other stories of its intake.
func storyIDs(c *backlog.Client, is *backlog.Issue) ([]string, bool, error) {
	issues, err := capstoneBacklog(c, is)
	if err != nil || issues == nil {
		return backlog.ParseTags(is).IDs, false, err
	}
	return pm.CapstoneIDs(is, issues), true, nil
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
