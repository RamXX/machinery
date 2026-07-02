package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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
		fmt.Fprintf(out, "  ok       modelith %s (pinned %s)\n", lastWord(ver), modelithVersion)
	} else {
		fmt.Fprintf(out, "  MISSING  modelith (Phase 1 domain model lint/render) -- install: go install github.com/stacklok/modelith/cmd/modelith@%s\n", modelithVersion)
	}

	// python3 — after migration this is optional, but preflight keeps reporting it
	// to match the Makefile until the deletion gate flips it.
	if p, err := exec.LookPath("python3"); err == nil {
		v := strings.TrimSpace(run(p, "--version"))
		fmt.Fprintf(out, "  ok       python3 %s (the gate tools need 3.10+)\n", lastWord(v))
	} else {
		fmt.Fprintln(out, "  MISSING  python3 3.10+ (the deterministic gate tools)")
	}

	// PyYAML
	if ok, err := runCheck("python3", "-c", "import yaml"); err == nil && ok {
		fmt.Fprintln(out, "  ok       PyYAML")
	} else {
		fmt.Fprintln(out, "  MISSING  PyYAML (the gate tools parse YAML) -- install: python3 -m pip install pyyaml")
	}

	// java (optional, reported ok when present)
	if p, err := exec.LookPath("java"); err == nil {
		v := firstLine(strings.TrimSpace(runErr(p, "-version")))
		fmt.Fprintf(out, "  ok       java (%s)\n", v)
	} else {
		fmt.Fprintln(out, "  optional Java 11+ -- needed ONLY for 'make verify-formal' (TLC model-checks the TLA+ proofs). The design pipeline and every gate run without it; with it you also get the exhaustive proofs. https://adoptium.net/")
	}
	fmt.Fprintln(out, "  auto     'machinery verify-formal' downloads the TLA+ tools (tla2tools.jar) on first use, pinned and checksum-verified (that step needs Java)")

	// uv (optional)
	if _, err := exec.LookPath("uv"); err == nil {
		fmt.Fprintln(out, "  ok       uv")
	} else {
		fmt.Fprintln(out, "  optional uv (legacy Python test runner) -- https://docs.astral.sh/uv/")
	}

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
	cmd := exec.Command(name, args...)
	b, _ := cmd.Output()
	return string(b)
}

func runErr(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	b, _ := cmd.CombinedOutput()
	return string(b)
}

func runCheck(name string, args ...string) (bool, error) {
	cmd := exec.Command(name, args...)
	return cmd.Run() == nil, nil
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
