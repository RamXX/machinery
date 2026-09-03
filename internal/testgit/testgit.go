// Package testgit runs hermetic, bounded Git commands for test fixtures.
package testgit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/processcontrol"
)

const (
	commandTimeout       = 10 * time.Second
	commandStdoutMaxSize = 4 << 20
	commandStderrMaxSize = 1 << 20
)

// Run executes Git in repo with a closed configuration, signing, hook,
// prompt, repository-redirection, and locale environment.
func Run(parent context.Context, repo string, args ...string) ([]byte, error) {
	return RunInput(parent, repo, nil, args...)
}

// RunInput is Run with explicit standard input.
func RunInput(parent context.Context, repo string, stdin io.Reader, args ...string) ([]byte, error) {
	return runInputWithLimits(parent, repo, stdin, commandStdoutMaxSize, commandStderrMaxSize, args...)
}

func runInputWithLimits(parent context.Context, repo string, stdin io.Reader, stdoutLimit, stderrLimit int, args ...string) ([]byte, error) {
	if parent == nil {
		return nil, fmt.Errorf("run hermetic Git command: nil context")
	}
	hooks, err := os.MkdirTemp("", "machinery-test-git-hooks-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(hooks) }()
	ctx, cancel := context.WithTimeout(parent, commandTimeout)
	defer cancel()
	commandArgs := []string{
		"-c", "commit.gpgsign=false",
		"-c", "tag.gpgSign=false",
		"-c", "core.hooksPath=" + filepath.ToSlash(hooks),
	}
	if repo != "" {
		commandArgs = append(commandArgs, "-C", repo)
	}
	commandArgs = append(commandArgs, args...)
	command := exec.CommandContext(ctx, "git", commandArgs...)
	command.Stdin = stdin
	command.Env = closedEnvironment(os.Environ())
	stdout, stderr, runErr := processcontrol.RunCapturedStreamLimits(ctx, command, stdoutLimit, stderrLimit)
	if ctx.Err() != nil {
		runErr = errors.Join(runErr, ctx.Err())
	}
	output := []byte(stdout + stderr)
	if runErr == nil && strings.TrimSpace(stderr) != "" {
		runErr = fmt.Errorf("git %s emitted stderr on success: %s", strings.Join(args, " "), strings.TrimSpace(stderr))
	}
	return output, runErr
}

func closedEnvironment(environ []string) []string {
	result := make([]string, 0, len(environ)+8)
	for _, item := range environ {
		key, _, _ := strings.Cut(item, "=")
		if strings.HasPrefix(key, "GIT_") || key == "GCM_INTERACTIVE" || key == "SSH_ASKPASS" || key == "LC_ALL" || key == "LANG" || key == "LANGUAGE" || key == "TZ" {
			continue
		}
		result = append(result, item)
	}
	return append(result,
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_SYSTEM="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		"GCM_INTERACTIVE=Never",
		"SSH_ASKPASS=",
		"LC_ALL=C",
		"LANG=C",
		"LANGUAGE=C",
		"TZ=UTC",
	)
}
