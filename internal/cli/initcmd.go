package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/v2/internal/project"
)

func init() {
	Register(&Command{Name: "init", Summary: "Scaffold a new project.", Run: runInit})
}

const initConfig = `# machinery project file. The contract for every key is docs/formats.md section 2.
# A stage whose key is absent is not applicable. Uncomment as the project grows.
version: 2

design:
  # models:
  #   - design/model/<name>.modelith.yaml
  # machines: design/machines
  # failure_modes: design/failure-modes.yaml
  # risk: design/risk.yaml
  owed: design/owed.yaml

# contracts:
#   - {kind: openapi, path: contracts/openapi.yaml}

# trace:
#   code: ["internal/**/*.go"]
#   tests: ["e2e/**"]

# tests:
#   e2e: ["e2e/**"]

# commands:
#   e2e: ["go", "test", "./e2e/...", "-count=1"]

# backlog:
#   tool: bd

# tools:
#   xstate: true   # compare generated tables with the real XState package
`

const initNext = `next:
  1. write design/model/<name>.modelith.yaml and list it under design.models
  2. write design/machines/<Entity>.machine.json and uncomment design.machines
  3. machinery lint, then machinery gen, then machinery owed sync after the first run
add to your Makefile:
  check:   ## Static gates
  	machinery check
  verify:  ## The end gate: check, the whole end-to-end suite, verify
  	machinery landing --base main
`

// runInit implements section 17.1. It never overwrites a file.
func runInit(e *Env, args []string) int {
	fs := e.Flags("init")
	if !e.Parse(fs, args) {
		return 2
	}
	root := e.ProjectFlag
	if root == "" {
		root = "."
	}
	cfg := filepath.Join(root, project.ConfigName)
	if _, err := os.Stat(cfg); err == nil {
		return e.Fail(project.Errf("%s exists; init never overwrites (use machinery adopt for an existing project)", project.ConfigName))
	}
	var created []string
	for _, dir := range []string{"design/model", "design/machines"} {
		abs := filepath.Join(root, filepath.FromSlash(dir))
		if _, err := os.Stat(abs); err != nil {
			if err := os.MkdirAll(abs, 0o755); err != nil {
				return e.Fail(err)
			}
			if err := os.WriteFile(filepath.Join(abs, ".gitkeep"), nil, 0o644); err != nil {
				return e.Fail(err)
			}
			created = append(created, dir+"/")
		}
	}
	owed := filepath.Join(root, "design", "owed.yaml")
	if _, err := os.Stat(owed); err != nil {
		if err := os.WriteFile(owed, []byte("version: 1\nowed: []\n"), 0o644); err != nil {
			return e.Fail(err)
		}
		created = append(created, "design/owed.yaml")
	}
	if err := os.WriteFile(cfg, []byte(initConfig), 0o644); err != nil {
		return e.Fail(err)
	}
	created = append(created, project.ConfigName)
	// receipts are never committed
	ignore := filepath.Join(root, ".gitignore")
	if data, _ := os.ReadFile(ignore); !strings.Contains("\n"+string(data), "\n.machinery/") {
		if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
			data = append(data, '\n')
		}
		if err := os.WriteFile(ignore, append(data, []byte(".machinery/\n")...), 0o644); err != nil {
			return e.Fail(err)
		}
		created = append(created, ".gitignore")
	}
	fmt.Fprintf(e.Stdout, "init: created %d path(s)\n", len(created))
	for _, c := range created {
		fmt.Fprintf(e.Stdout, "  %s\n", c)
	}
	fmt.Fprint(e.Stdout, initNext)
	return 0
}
