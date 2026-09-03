// Command git-safe runs one Git command with a bounded process tree, a closed
// ambient Git environment, bounded output, and warning-free success semantics.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/RamXX/machinery/internal/gitcontrol"
	"github.com/RamXX/machinery/internal/processcontrol"
)

const defaultOutputLimit = 4 << 20

var outputLimit = defaultOutputLimit

func main() {
	os.Exit(run(os.Args[1:], os.Environ(), os.Stdout, os.Stderr))
}

func run(args, environ []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("git-safe", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "trusted repository root")
	timeout := flags.Duration("timeout", 30*time.Second, "Git command timeout")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	gitArgs := flags.Args()
	if *root == "" || len(gitArgs) == 0 || *timeout <= 0 || *timeout > 5*time.Minute {
		fmt.Fprintln(stderr, "git-safe: require -root, a Git command, and a timeout in (0,5m]")
		return 2
	}
	absRoot, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintf(stderr, "git-safe: resolve repository root: %v\n", err)
		return 1
	}
	info, err := os.Stat(absRoot)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(stderr, "git-safe: repository root is not a real directory: %s\n", absRoot)
		return 1
	}

	gitPath, err := exec.LookPath("git")
	if err != nil {
		fmt.Fprintf(stderr, "git-safe: resolve Git executable: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitPath, append([]string{"-C", absRoot}, gitArgs...)...)
	cmd.Dir = absRoot
	cmd.Env = gitcontrol.Environment(environ)
	commandStdout, commandStderr, runErr := processcontrol.RunCapturedStreams(ctx, cmd, outputLimit)
	if ctx.Err() != nil {
		fmt.Fprintf(stderr, "git-safe: Git command timed out after %s\n", timeout.String())
		return 124
	}
	if runErr != nil {
		_, _ = io.WriteString(stdout, commandStdout)
		_, _ = io.WriteString(stderr, commandStderr)
		fmt.Fprintf(stderr, "git-safe: Git command failed: %v\n", runErr)
		return 1
	}
	if commandStderr != "" {
		_, _ = io.WriteString(stderr, commandStderr)
		fmt.Fprintln(stderr, "git-safe: successful Git command emitted stderr")
		return 1
	}
	if _, err := io.WriteString(stdout, commandStdout); err != nil {
		fmt.Fprintf(stderr, "git-safe: write stdout: %v\n", err)
		return 1
	}
	return 0
}
