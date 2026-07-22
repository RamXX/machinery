package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/install"
)

const modelithVersion = "v0.4.0"

// preflightRun mirrors the Makefile `preflight` target: checks the same labels
// (ok/MISSING/optional/auto) so `machinery preflight` and `make preflight` agree.
func preflightRun() {
	out := stdoutW
	fmt.Fprintln(out, "machinery prerequisites:")

	// modelith
	if p, err := exec.LookPath("modelith"); err == nil {
		ver := strings.TrimSpace(strings.Join(strings.Fields(run(p, "--version")), " "))
		installed := lastWord(ver)
		if strings.TrimPrefix(installed, "v") == strings.TrimPrefix(modelithVersion, "v") {
			fmt.Fprintf(out, "  ok       modelith %s (pinned %s)\n", installed, modelithVersion)
		} else {
			fmt.Fprintf(out, "  WARN     modelith %s does not match the pin %s -- install: go install github.com/stacklok/modelith/cmd/modelith@%s (or make install-modelith)\n", installed, modelithVersion, modelithVersion)
		}
	} else {
		fmt.Fprintf(out, "  MISSING  modelith (Phase 1 domain model lint/render) -- install: go install github.com/stacklok/modelith/cmd/modelith@%s (or make install-modelith)\n", modelithVersion)
	}

	// the gate tools and generators are this binary itself: nothing else needed
	fmt.Fprintln(out, "  ok       machinery "+version+" (the deterministic gate tools and formal generators are this binary; no Python, no other runtime)")

	// java (optional, reported ok when present)
	if p, err := exec.LookPath("java"); err == nil {
		v := firstLine(strings.TrimSpace(runErr(p, "-version")))
		fmt.Fprintf(out, "  ok       java (%s)\n", v)
	} else {
		fmt.Fprintln(out, "  optional Java 11+ -- needed ONLY for 'make verify-formal' (TLC model-checks the TLA+ proofs). The design pipeline and every gate run without it; with it you also get the exhaustive proofs. https://adoptium.net/")
	}
	fmt.Fprintln(out, "  auto     'machinery verify-formal' downloads the TLA+ tools (tla2tools.jar) and, for designs with a relational annotation (policy, integrity, or isolation), the Alloy analyzer (org.alloytools.alloy.dist.jar) on first use, pinned and checksum-verified (that step needs Java)")

	// structurizr-cli (optional)
	if _, err := exec.LookPath("structurizr-cli"); err == nil {
		fmt.Fprintln(out, "  ok       structurizr-cli (C4 diagram export)")
	} else if _, err := exec.LookPath("structurizr"); err == nil {
		fmt.Fprintln(out, "  ok       structurizr-cli (C4 diagram export)")
	} else {
		fmt.Fprintln(out, "  optional structurizr-cli (C4 diagram EXPORT only; the DSL and every gate need only text) -- https://structurizr.com/cli")
	}

	// scorecard (optional)
	if p, err := exec.LookPath("scorecard"); err == nil {
		v := "present"
		for _, line := range strings.Split(run(p, "version"), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "GitVersion:") {
				v = lastWord(line)
				break
			}
		}
		fmt.Fprintf(out, "  ok       scorecard %s (OpenSSF Scorecard: Phase 2 adoption-closure risk evidence; needs GITHUB_AUTH_TOKEN at run time)\n", v)
	} else {
		fmt.Fprintln(out, "  optional scorecard (OpenSSF Scorecard, Phase 2 adoption-closure risk evidence; public GitHub repos need NO install: curl https://api.securityscorecards.dev/projects/github.com/<org>/<repo>) -- install: go install github.com/ossf/scorecard/v5@latest (needs GITHUB_AUTH_TOKEN at run time)")
	}
}

// doctorRun mirrors the Makefile `doctor` target: preflight + install status.
// With targets it checks each host's native adapter; without them it preserves
// the original ~/.claude + ~/.agents report.
func doctorRun(targets []string) error {
	if len(targets) > 0 {
		artifacts, err := install.TargetArtifacts(targets)
		if err != nil {
			return err
		}
		preflightRun()
		out := stdoutW
		fmt.Fprintln(out, "install status:")
		for _, artifact := range artifacts {
			if _, err := os.Stat(artifact.Path); err == nil {
				fmt.Fprintf(out, "  ok       [%s] %s at %s\n", artifact.Target, artifact.Label, artifact.Path)
			} else {
				fmt.Fprintf(out, "  MISSING  [%s] %s at %s -- run machinery install --target %s\n", artifact.Target, artifact.Label, artifact.Path, strings.Join(targets, " --target "))
			}
		}
		reportHookWiring(out)
		reportUpdateReceipt(out)
		return nil
	}

	preflightRun()
	out := stdoutW
	fmt.Fprintln(out, "install status:")
	homes := []string{os.Getenv("HOME") + "/.claude", os.Getenv("HOME") + "/.agents"}
	for _, home := range homes {
		if _, err := os.Stat(home + "/skills/machinery"); err == nil {
			fmt.Fprintf(out, "  ok       skill at %s/skills/machinery\n", home)
		} else {
			fmt.Fprintf(out, "  MISSING  skill at %s/skills/machinery -- run make install\n", home)
		}
		if _, err := os.Stat(home + "/agents/machinery-fsm-author.md"); err == nil {
			fmt.Fprintf(out, "  ok       fsm-author role at %s/agents\n", home)
		} else {
			fmt.Fprintf(out, "  MISSING  fsm-author role at %s/agents -- run make install\n", home)
		}
		if _, err := os.Stat(home + "/agents/machinery-build-writer.md"); err == nil {
			fmt.Fprintf(out, "  ok       build-writer role at %s/agents\n", home)
		} else {
			fmt.Fprintf(out, "  MISSING  build-writer role at %s/agents -- run make install\n", home)
		}
	}
	reportHookWiring(out)
	reportUpdateReceipt(out)
	return nil
}

// reportHookWiring checks the machinery plugin hook plumbing wherever a
// plugin layout exists (a .claude-plugin/plugin.json with a hooks/ sibling):
// hooks.json must be present and the shim executable, or every governance
// hook silently never fires (GATE-11 doctor check).
func reportHookWiring(out io.Writer) {
	roots := pluginRoots()
	if len(roots) == 0 {
		fmt.Fprintln(out, "  auto     no machinery plugin layout found (.claude-plugin/ + hooks/); governance hooks run only where the plugin is installed")
		return
	}
	for _, root := range roots {
		manifest := filepath.Join(root, "hooks", "hooks.json")
		if _, err := os.Stat(manifest); err == nil {
			fmt.Fprintf(out, "  ok       hook manifest at %s\n", manifest)
		} else {
			fmt.Fprintf(out, "  MISSING  hook manifest at %s -- the plugin layout exists but no governance hook will fire\n", manifest)
		}
		shim := filepath.Join(root, "hooks", "machinery-hook.sh")
		if fi, err := os.Stat(shim); err != nil {
			fmt.Fprintf(out, "  MISSING  hook shim at %s -- the hooks.json entries point at nothing\n", shim)
		} else if fi.Mode()&0o111 == 0 {
			fmt.Fprintf(out, "  WARN     hook shim at %s is not executable -- chmod +x it or every hook invocation fails silently\n", shim)
		} else {
			fmt.Fprintf(out, "  ok       hook shim at %s (executable)\n", shim)
		}
	}
}

// pluginRoots finds machinery plugin layouts: $CLAUDE_PLUGIN_ROOT, the
// current directory, and any ~/.claude/plugins entry that carries a
// machinery hook shim. A root qualifies when .claude-plugin/plugin.json and
// a hooks/ directory both exist.
func pluginRoots() []string {
	var roots []string
	seen := map[string]bool{}
	add := func(dir string) {
		if dir == "" || seen[dir] {
			return
		}
		if _, err := os.Stat(filepath.Join(dir, ".claude-plugin", "plugin.json")); err != nil {
			return
		}
		if fi, err := os.Stat(filepath.Join(dir, "hooks")); err != nil || !fi.IsDir() {
			return
		}
		seen[dir] = true
		roots = append(roots, dir)
	}
	add(os.Getenv("CLAUDE_PLUGIN_ROOT"))
	if wd, err := os.Getwd(); err == nil {
		add(wd)
	}
	plugins := filepath.Join(os.Getenv("HOME"), ".claude", "plugins")
	if entries, err := os.ReadDir(plugins); err == nil {
		for _, e := range entries {
			if e.IsDir() && strings.Contains(strings.ToLower(e.Name()), "machinery") {
				add(filepath.Join(plugins, e.Name()))
			}
		}
	}
	return roots
}

func reportUpdateReceipt(out io.Writer) {
	status, err := install.InstallationReceiptStatus()
	switch {
	case err != nil:
		fmt.Fprintf(out, "  WARN     update receipt at %s is unreadable: %v\n", status.Path, err)
	case status.Exists:
		fmt.Fprintf(out, "  ok       update receipt at %s (%d home group(s), %d native target(s))\n", status.Path, status.HomeInstalls, status.Targets)
	default:
		fmt.Fprintf(out, "  auto     no update receipt at %s; standard installed paths will be discovered on the first 'machinery update'\n", status.Path)
	}
}

func run(name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	b, _ := exec.CommandContext(ctx, name, args...).Output()
	return string(b)
}

func runErr(name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	b, _ := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(b)
}

func lastWord(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return s[:i]
	}
	return s
}
