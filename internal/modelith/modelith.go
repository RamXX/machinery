// Package modelith runs the real modelith executable (docs/formats.md sections 5.1 and 6.1).
package modelith

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/version"
)

// InstallHint is printed when the tool is missing or too old.
const InstallHint = "go install github.com/stacklok/modelith/cmd/modelith@v0.6.0"

var versionRe = regexp.MustCompile(`modelith version v?([0-9]+)\.([0-9]+)\.([0-9]+)`)

// Check runs "<modelith> --version" and requires 0.6.0 or newer. The returned
// reason is empty when the tool is usable.
func Check(p *project.Project) (ver string, reason string) {
	exe := p.Cfg.Tools.Modelith
	out, err := exec.Command(exe, "--version").CombinedOutput()
	notFound := fmt.Sprintf("modelith %s or newer not found (%s)", version.ModelithMin, InstallHint)
	if err != nil {
		return "", notFound
	}
	m := versionRe.FindStringSubmatch(string(out))
	if m == nil {
		return "", notFound
	}
	got := [3]int{}
	for i := 0; i < 3; i++ {
		got[i], _ = strconv.Atoi(m[i+1])
	}
	ver = fmt.Sprintf("%d.%d.%d", got[0], got[1], got[2])
	want := [3]int{0, 6, 0}
	for i := 0; i < 3; i++ {
		if got[i] > want[i] {
			return ver, ""
		}
		if got[i] < want[i] {
			return ver, fmt.Sprintf("modelith %s is older than %s (%s)", ver, version.ModelithMin, InstallHint)
		}
	}
	return ver, ""
}

// Finding is one modelith lint finding.
type Finding struct {
	Severity string
	Message  string
}

// Lint runs "modelith lint --format json <file>" and returns its findings.
func Lint(p *project.Project, rel string) ([]Finding, error) {
	cmd := exec.Command(p.Cfg.Tools.Modelith, "lint", "--format", "json", rel)
	cmd.Dir = p.Root
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	runErr := cmd.Run()
	var doc struct {
		Files []struct {
			File     string            `json:"file"`
			Findings []json.RawMessage `json:"findings"`
		} `json:"files"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" && runErr != nil {
			msg = runErr.Error()
		}
		return nil, fmt.Errorf("modelith lint did not print JSON: %s", firstLine(msg))
	}
	var res []Finding
	for _, f := range doc.Files {
		for _, raw := range f.Findings {
			res = append(res, parseFinding(raw))
		}
	}
	return res, nil
}

func parseFinding(raw json.RawMessage) Finding {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return Finding{"error", string(raw)}
	}
	get := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := m[k]; ok && v != nil {
				return strings.TrimSpace(fmt.Sprint(v))
			}
		}
		return ""
	}
	sev := strings.ToLower(get("severity", "level"))
	if sev != "warning" && sev != "warn" {
		sev = "error"
	} else {
		sev = "warn"
	}
	msg := get("message", "msg")
	if rule := get("rule", "code", "id"); rule != "" {
		msg = rule + ": " + msg
	}
	if path := get("path", "pointer", "location"); path != "" {
		msg = path + ": " + msg
	}
	if line := get("line"); line != "" && line != "0" {
		msg = "line " + line + ": " + msg
	}
	return Finding{sev, msg}
}

// Render returns the committed rendering of a model: "modelith render --stdout"
// with every em dash replaced by a hyphen-minus.
func Render(p *project.Project, rel string) ([]byte, error) {
	cmd := exec.Command(p.Cfg.Tools.Modelith, "render", "--stdout", rel)
	cmd.Dir = p.Root
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("modelith render failed: %s", firstLine(strings.TrimSpace(errb.String())))
	}
	return bytes.ReplaceAll(out.Bytes(), []byte("\u2014"), []byte("-")), nil
}

// RenderedPath is the path of the committed rendering beside a model.
func RenderedPath(modelRel string) string {
	for _, suf := range []string{".modelith.yaml", ".modelith.yml"} {
		if strings.HasSuffix(modelRel, suf) {
			return strings.TrimSuffix(modelRel, suf) + ".modelith.md"
		}
	}
	if i := strings.LastIndex(modelRel, "."); i > strings.LastIndex(modelRel, "/") {
		return modelRel[:i] + ".md"
	}
	return modelRel + ".md"
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.ReplaceAll(s, "\u2014", "-")
}
