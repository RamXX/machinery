package landing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/v2/internal/gitx"
	"github.com/RamXX/machinery/v2/internal/project"
)

// Red is <git common dir>/machinery/red.json: the integration branch is red
// since a landing failed at Commit (section 12.2).
type Red struct {
	Version int      `json:"version"`
	Commit  string   `json:"commit"`
	Base    string   `json:"base"`
	Stages  []string `json:"stages"`
}

// What names the red commit and its stages: "<short sha> (<stages>)".
func (r *Red) What() string {
	return fmt.Sprintf("%s (%s)", gitx.Short(r.Commit), strings.Join(r.Stages, ", "))
}

// Refusal is the line a command prints when it refuses to move work onto a
// red integration branch.
func (r *Red) Refusal() string {
	return "ERROR  the integration branch is red since " + r.What() + ": repair or revert first"
}

// RedPath returns the absolute path of the marker.
func RedPath(p *project.Project) (string, error) {
	dir, err := gitx.CommonDir(p.Root)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "machinery", "red.json"), nil
}

// LoadRed reads the marker; nil when the branch is not red.
func LoadRed(p *project.Project) (*Red, error) {
	path, err := RedPath(p)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var r Red
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	return &r, nil
}

// writeRed records a failed landing.
func writeRed(p *project.Project, r Red) error {
	path, err := RedPath(p)
	if err != nil {
		return err
	}
	r.Version = 1
	if r.Stages == nil {
		r.Stages = []string{}
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return project.WriteFileAtomic(path, append(data, '\n'))
}

// ClearRed removes the marker and returns what it held; nil when there was none.
func ClearRed(p *project.Project) (*Red, error) {
	r, err := LoadRed(p)
	if err != nil || r == nil {
		return nil, err
	}
	path, err := RedPath(p)
	if err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return r, nil
}
