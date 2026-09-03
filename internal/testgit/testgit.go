// Package testgit runs hermetic, bounded Git commands for test fixtures.
package testgit

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const commandTimeout = 10 * time.Second

// Run executes Git in repo with a closed configuration, signing, hook,
// prompt, repository-redirection, and locale environment.
func Run(parent context.Context, repo string, args ...string) ([]byte, error) {
	return RunInput(parent, repo, nil, args...)
}

// RunInput is Run with explicit standard input.
func RunInput(parent context.Context, repo string, stdin io.Reader, args ...string) ([]byte, error) {
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
	output, runErr := command.CombinedOutput()
	if ctx.Err() != nil {
		runErr = errors.Join(runErr, ctx.Err())
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
