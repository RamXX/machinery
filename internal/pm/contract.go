// Package pm is the backlog bridge: plan, waves, the contract freeze, briefs
// and the tier gates of a story (docs/formats.md section 15).
package pm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
)

// Contract is the contract hash over the design sources (section 15.4).
type Contract struct {
	Hash  string            // sha256 hex
	Files map[string]string // path -> sha256 hex of the file
	Paths []string          // sorted
}

// Short is the first 12 hex of the hash.
func (c *Contract) Short() string { return c.Hash[:12] }

// missing is the file hash recorded for a design source that cannot be read.
const missing = "missing"

// ContractOf hashes the design sources of d.
func ContractOf(d *design.Design) *Contract {
	c := &Contract{Files: map[string]string{}, Paths: d.ContractFiles()}
	var b strings.Builder
	for _, rel := range c.Paths {
		h := missing
		if data, err := d.P.Read(rel); err == nil {
			sum := sha256.Sum256(data)
			h = hex.EncodeToString(sum[:])
		}
		c.Files[rel] = h
		fmt.Fprintf(&b, "%s %s\n", rel, h)
	}
	sum := sha256.Sum256([]byte(b.String()))
	c.Hash = hex.EncodeToString(sum[:])
	return c
}

// Wave is <git common dir>/machinery/wave.json.
type Wave struct {
	Version  int               `json:"version"`
	Contract string            `json:"contract"`
	Commit   string            `json:"commit"`
	Files    map[string]string `json:"files"`
	Stories  []string          `json:"stories"`
}

// ErrNoRepo is returned when the project is not in a git repository.
var ErrNoRepo = errors.New("the project is not in a git repository")

// WavePath returns the absolute path of the wave file.
func WavePath(p *project.Project) (string, error) {
	if !gitx.IsRepo(p.Root) {
		return "", ErrNoRepo
	}
	dir, err := gitx.CommonDir(p.Root)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "machinery", "wave.json"), nil
}

// LoadWave reads the frozen wave. ok is false when no wave is frozen or the
// project is not in a git repository.
func LoadWave(p *project.Project) (w *Wave, ok bool, err error) {
	path, err := WavePath(p)
	if errors.Is(err, ErrNoRepo) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	w = &Wave{}
	if err := json.Unmarshal(data, w); err != nil || w.Version != 1 {
		return nil, false, fmt.Errorf("wave file unreadable: run machinery waves --thaw, then --freeze again")
	}
	return w, true, nil
}

// WriteWave writes the wave file.
func WriteWave(p *project.Project, w *Wave) error {
	path, err := WavePath(p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Thaw removes the wave file; removed is false when none was frozen.
func Thaw(p *project.Project) (removed bool, err error) {
	path, err := WavePath(p)
	if err != nil {
		return false, err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// ChangedFiles lists the design sources whose hash differs from the frozen
// wave, added and removed ones included, sorted.
func ChangedFiles(w *Wave, c *Contract) []string {
	set := map[string]bool{}
	for path, h := range c.Files {
		if w.Files[path] != h {
			set[path] = true
		}
	}
	for path := range w.Files {
		if _, ok := c.Files[path]; !ok {
			set[path] = true
		}
	}
	var res []string
	for path := range set {
		res = append(res, path)
	}
	sort.Strings(res)
	return res
}
