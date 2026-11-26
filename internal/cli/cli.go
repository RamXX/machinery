// Package cli is the command dispatcher. Every command lives in its own file
// and registers itself in an init function.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/stages"
)

// Env is what a command gets.
type Env struct {
	Stdout      io.Writer
	Stderr      io.Writer
	ProjectFlag string
	// exit, when set, is the exit code whatever the command returns.
	exit int
}

// Command is one entry of section 3.1. Name may hold a space ("owed list").
type Command struct {
	Name    string
	Summary string
	Run     func(e *Env, args []string) int
}

var commands = map[string]*Command{}

// Register adds a command. It panics on a duplicate, which is a build error.
func Register(c *Command) {
	if _, dup := commands[c.Name]; dup {
		panic("duplicate command " + c.Name)
	}
	commands[c.Name] = c
}

func names() []string {
	var ns []string
	for n := range commands {
		ns = append(ns, n)
	}
	sort.Strings(ns)
	return ns
}

func help(w io.Writer) {
	fmt.Fprintln(w, "machinery joins a project's design to evidence from its running system.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage: machinery [-C <project dir>] <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, n := range names() {
		fmt.Fprintf(w, "  %-18s %s\n", n, commands[n].Summary)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The contract for every format, output line and exit code is docs/formats.md.")
}

// Execute runs the command line and returns the exit code.
func Execute(args []string, stdout, stderr io.Writer) int {
	e := &Env{Stdout: stdout, Stderr: stderr}
	for len(args) > 0 {
		a := args[0]
		switch {
		case a == "-C" || a == "--project":
			if len(args) < 2 {
				fmt.Fprintf(stderr, "usage: %s needs a directory\n", a)
				return 2
			}
			e.ProjectFlag = args[1]
			args = args[2:]
			continue
		case strings.HasPrefix(a, "--project="):
			e.ProjectFlag = strings.TrimPrefix(a, "--project=")
			args = args[1:]
			continue
		}
		break
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		help(stdout)
		return 0
	}
	if len(args) >= 2 {
		if c, ok := commands[args[0]+" "+args[1]]; ok {
			return e.run(c, args[2:])
		}
	}
	if c, ok := commands[args[0]]; ok {
		return e.run(c, args[1:])
	}
	var subs []string
	for _, n := range names() {
		if strings.HasPrefix(n, args[0]+" ") {
			subs = append(subs, n)
		}
	}
	if len(subs) > 0 {
		fmt.Fprintf(stderr, "usage: machinery %s needs a subcommand: %s\n", args[0], strings.Join(subs, ", "))
		return 2
	}
	fmt.Fprintf(stderr, "usage: unknown command %q; run machinery for the list\n", args[0])
	return 2
}

// Flags returns a flag set that reports errors on stderr without exiting.
func (e *Env) Flags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet("machinery "+name, flag.ContinueOnError)
	fs.SetOutput(e.Stderr)
	return fs
}

// Parse parses flags and returns false (exit 2) on a usage error.
func (e *Env) Parse(fs *flag.FlagSet, args []string) bool {
	return fs.Parse(args) == nil
}

// Usage prints a usage error and returns 2.
func (e *Env) Usage(format string, a ...any) int {
	fmt.Fprintf(e.Stderr, "usage: "+format+"\n", a...)
	return 2
}

// Project loads the project or prints the config error. ok is false on failure.
func (e *Env) Project() (*project.Project, bool) {
	p, err := project.Load(e.ProjectFlag)
	if err != nil {
		e.Fail(err)
		return nil, false
	}
	// an owed list that does not parse has no entries to trust: no command
	// goes on as if it were empty (section 11.1)
	if lines := design.OwedRefusal(p); len(lines) > 0 {
		for _, l := range lines {
			fmt.Fprintf(e.Stderr, "ERROR  %s\n", l)
		}
		e.exit = 1
		return nil, false
	}
	return p, true
}

// run runs a command; a refusal that sets its own exit code keeps it.
func (e *Env) run(c *Command, args []string) int {
	code := c.Run(e, args)
	if e.exit != 0 {
		return e.exit
	}
	return code
}

// Fail prints an error and returns its exit code: 2 for a config error, 1 otherwise.
func (e *Env) Fail(err error) int {
	var ce *project.ConfigError
	if errors.As(err, &ce) {
		fmt.Fprintf(e.Stderr, "config: %s\n", ce.Msg)
		return 2
	}
	fmt.Fprintf(e.Stderr, "machinery: %v\n", err)
	return 1
}

// RunStages runs the named stages, prints them and returns the exit code.
func (e *Env) RunStages(command string, ctx *stages.Context, names []string) int {
	return report.PrintAll(e.Stdout, command, stages.Run(ctx, names))
}

// StringList is a repeatable string flag.
type StringList []string

func (s *StringList) String() string     { return strings.Join(*s, ",") }
func (s *StringList) Set(v string) error { *s = append(*s, v); return nil }
