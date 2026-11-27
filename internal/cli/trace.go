package cli

import (
	"fmt"

	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/stages"
)

func init() {
	Register(&Command{Name: "trace", Summary: "Check that code serves actions and tests claim obligations.", Run: func(e *Env, args []string) int {
		fs := e.Flags("trace")
		list := fs.Bool("non-action-serves", false, "list the ids that serves tags name and that are not actions, and run no check")
		if !e.Parse(fs, args) {
			return 2
		}
		if fs.NArg() > 0 {
			return e.Usage("machinery trace takes no arguments")
		}
		p, ok := e.Project()
		if !ok {
			return 2
		}
		if *list {
			// a report, never a gate (section 8)
			if p.Cfg.Trace == nil {
				return e.Fail(project.Errf("no trace configured: machinery trace --non-action-serves lists serves tags, and trace says where they are"))
			}
			list, links, unread := stages.NonActionServes(&stages.Context{P: p})
			if len(unread) > 0 {
				// no list is better than a part of one that reads as whole
				for _, why := range unread {
					fmt.Fprintf(e.Stderr, "machinery: trace --non-action-serves: %s; the list is not known\n", why)
				}
				return 1
			}
			for _, n := range list {
				fmt.Fprintf(e.Stdout, "%s:%d: %s\n", n.File, n.Line, n.ID)
			}
			// what is behind a link is never read, so the list says so
			for _, l := range links {
				fmt.Fprintf(e.Stderr, "machinery: trace --non-action-serves: %s is a symbolic link; tags behind it are not listed\n", l)
			}
			return 0
		}
		return e.RunStages("trace", &stages.Context{P: p}, []string{"trace"})
	}})
}
