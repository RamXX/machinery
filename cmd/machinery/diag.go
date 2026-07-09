package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
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
}

// doctorRun mirrors the Makefile `doctor` target: preflight + install status.
func doctorRun() {
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
