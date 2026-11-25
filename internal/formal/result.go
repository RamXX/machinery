package formal

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/RamXX/machinery/internal/gen"
	"github.com/RamXX/machinery/internal/project"
)

// RunResult is the recorded outcome of one run (section 16.9).
type RunResult struct {
	Name     string          `json:"name"`
	Outcome  string          `json:"outcome"` // passed | violated | error
	Violated string          `json:"violated"`
	Reach    map[string]bool `json:"reach"`
	Trace    string          `json:"trace,omitempty"`
}

// MutationResult is the recorded outcome of one mutation.
type MutationResult struct {
	ID      string `json:"id"`
	Outcome string `json:"outcome"` // detected | missed | inapplicable
}

// Result is <spec>.result.json.
type Result struct {
	Version     int              `json:"version"`
	Spec        string           `json:"spec"`
	Tool        string           `json:"tool"`
	ToolVersion string           `json:"tool_version"`
	InputSHA256 string           `json:"input_sha256"`
	Runs        []RunResult      `json:"runs"`
	Mutations   []MutationResult `json:"mutations"`
}

// Step is one step of a counterexample trace (section 16.10).
type Step struct {
	N        int               `json:"n"`
	Case     string            `json:"case"`
	Instance string            `json:"instance"`
	Caller   string            `json:"caller"`
	Step     string            `json:"step"`
	State    map[string]string `json:"state"`
}

// Trace is a counterexample trace file.
type Trace struct {
	Version  int                            `json:"version"`
	ID       string                         `json:"id"`
	Spec     string                         `json:"spec"`
	Run      string                         `json:"run"`
	Tool     string                         `json:"tool"`
	Violates string                         `json:"violates"`
	Expect   string                         `json:"expect"`
	Machine  string                         `json:"machine"`
	Steps    []Step                         `json:"steps"`
	Loop     int                            `json:"loop"`
	Instance map[string]map[string][]string `json:"instance"`
}

// LoadResult reads a spec's result file; ok is false when it does not exist
// or cannot be parsed.
func LoadResult(p *project.Project, s *Spec) (*Result, bool) {
	data, err := p.Read(s.ResultPath())
	if err != nil {
		return nil, false
	}
	var r Result
	if json.Unmarshal(data, &r) != nil {
		return nil, false
	}
	return &r, true
}

func (r *Result) run(name string) *RunResult {
	for i := range r.Runs {
		if r.Runs[i].Name == name {
			return &r.Runs[i]
		}
	}
	return nil
}

func (r *Result) mutation(id string) *MutationResult {
	for i := range r.Mutations {
		if r.Mutations[i].ID == id {
			return &r.Mutations[i]
		}
	}
	return nil
}

func writeJSON(p *project.Project, rel string, v any) error {
	data, err := gen.JSON(v)
	if err != nil {
		return err
	}
	return p.WriteAtomic(rel, data)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
