// Package xstate runs the real XState package against machines and their
// generated tables (docs/formats.md section 6.8).
package xstate

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/v2/internal/machine"
	"github.com/RamXX/machinery/v2/internal/project"
)

//go:embed conformance.cjs
var script []byte

// Range is the supported version range, as printed in reasons.
const Range = "5.33.4 or newer, below 6"

const installHint = "npm install --save-exact xstate@5.33.4 in <tools dir>/xstate"

var semverRe = regexp.MustCompile(`^([0-9]+)\.([0-9]+)\.([0-9]+)`)

// Package returns the directory of the xstate package under the tools
// directory, or a reason it cannot be used.
func Package(p *project.Project) (dir string, reason string) {
	dir = filepath.Join(p.ToolsDir(), "xstate", "node_modules", "xstate")
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return "", fmt.Sprintf("the xstate package is not in the tools directory (%s)", installHint)
	}
	var doc struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(data, &doc) != nil || doc.Version == "" {
		return "", fmt.Sprintf("the xstate package in the tools directory has no version (%s)", installHint)
	}
	m := semverRe.FindStringSubmatch(doc.Version)
	if m == nil {
		return "", fmt.Sprintf("xstate %s is not a version in the range %s (%s)", doc.Version, Range, installHint)
	}
	var v [3]int
	for i := range v {
		v[i], _ = strconv.Atoi(m[i+1])
	}
	if v[0] != 5 || v[1] < 33 || (v[1] == 33 && v[2] < 4) {
		return "", fmt.Sprintf("xstate %s is outside the range %s (%s)", doc.Version, Range, installHint)
	}
	return dir, ""
}

// Node checks that tools.node runs. The reason is empty when it does.
func Node(p *project.Project) string {
	exe := p.Cfg.Tools.Node
	if err := exec.Command(exe, "--version").Run(); err != nil {
		return fmt.Sprintf("node not found (tools.node is %s)", exe)
	}
	return ""
}

// Result is what XState did for one case, or for a machine when Case is "".
type Result struct {
	Machine string   `json:"machine"`
	Case    string   `json:"case"`
	Initial string   `json:"initial"`
	State   string   `json:"state"`
	Actions []string `json:"actions"`
	Error   string   `json:"error"`
}

type caseIn struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Event  string `json:"event"`
	Guard  string `json:"guard"`
}

type machineIn struct {
	ID     string         `json:"id"`
	JSON   string         `json:"json"`
	Guards []string       `json:"guards"`
	Delays map[string]int `json:"delays"`
	Cases  []caseIn       `json:"cases"`
}

// Run asks XState about every machine and every transition row of kind on.
func Run(p *project.Project, pkg string, ms []*machine.Machine) ([]Result, error) {
	in := struct {
		Package  string      `json:"package"`
		Machines []machineIn `json:"machines"`
	}{Package: pkg, Machines: []machineIn{}}
	for _, m := range ms {
		mi := machineIn{ID: m.ID, JSON: string(m.Bytes), Guards: m.Table.Units.Guards, Delays: map[string]int{}, Cases: []caseIn{}}
		for _, tr := range m.Table.Transitions {
			switch tr.Kind {
			case "after":
				mi.Delays[tr.Event] = 1
			case "on":
				mi.Cases = append(mi.Cases, caseIn{tr.ID, tr.Source, tr.Event, tr.Guard})
			}
		}
		for k, v := range m.Table.Delays {
			if v > 0 {
				mi.Delays[k] = v
			}
		}
		in.Machines = append(in.Machines, mi)
	}
	payload, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	f, err := os.CreateTemp("", "machinery-xstate-*.cjs")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(script); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	cmd := exec.Command(p.Cfg.Tools.Node, f.Name())
	cmd.Dir = p.Root
	cmd.Stdin = bytes.NewReader(payload)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("the conformance script failed: %s", errorLine(msg))
	}
	var res []Result
	sc := bufio.NewScanner(&out)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		var r Result
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("the conformance script printed a line that is not JSON: %v", err)
		}
		res = append(res, r)
	}
	return res, sc.Err()
}

var errLineRe = regexp.MustCompile(`^[A-Za-z]*Error\b.*`)

// errorLine picks the error line out of a Node stack trace.
func errorLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if errLineRe.MatchString(strings.TrimSpace(l)) {
			return strings.TrimSpace(l)
		}
	}
	return strings.TrimSpace(strings.SplitN(s, "\n", 2)[0])
}
