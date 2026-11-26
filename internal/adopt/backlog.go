package adopt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Backlog is what adopt reads of the project's own bd backlog. It never writes.
type Backlog struct {
	Found    bool // a .beads directory exists
	Runnable bool // bd answered
	NotRead  bool // a rehearsal: bd is never run (section 17.3)
	Open     int
	Ready    int
	Closed   int
	Cycles   int
}

// Line is the backlog line of the report.
func (b Backlog) Line() string {
	switch {
	case !b.Found:
		return "backlog: none found"
	case b.NotRead:
		return "backlog: bd, not read in a rehearsal"
	case !b.Runnable:
		return "backlog: bd not runnable"
	}
	cycles := "none"
	if b.Cycles > 0 {
		cycles = fmt.Sprint(b.Cycles)
	}
	return fmt.Sprintf("backlog: bd, %d open, %d ready, %d closed, cycles: %s", b.Open, b.Ready, b.Closed, cycles)
}

func bd(exe, dir string, args ...string) ([]byte, error) {
	cmd := exec.Command(exe, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "BD_NON_INTERACTIVE=1")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

type rawIssue struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Dependencies []struct {
		DependsOnID string `json:"depends_on_id"`
		Type        string `json:"type"`
	} `json:"dependencies"`
}

// readBacklog runs bd in the project directory when a .beads directory exists.
// A missing or failing bd is reported in the result, never as an error.
func readBacklog(root, exe string, notRead bool) Backlog {
	var b Backlog
	if info, err := os.Stat(filepath.Join(root, ".beads")); err != nil || !info.IsDir() {
		return b
	}
	b.Found = true
	if notRead {
		b.NotRead = true
		return b
	}
	out, err := bd(exe, root, "list", "--all", "--json", "--limit", "0")
	if err != nil {
		return b
	}
	var issues []rawIssue
	if json.Unmarshal(out, &issues) != nil {
		return b
	}
	cyc, err := bd(exe, root, "dep", "cycles", "--json")
	if err != nil {
		return b
	}
	var cycles []json.RawMessage
	if json.Unmarshal(cyc, &cycles) != nil {
		return b
	}
	b.Runnable, b.Cycles = true, len(cycles)
	status := map[string]string{}
	for _, is := range issues {
		status[is.ID] = is.Status
	}
	for _, is := range issues {
		switch is.Status {
		case "closed":
			b.Closed++
			continue
		case "deferred":
			continue
		}
		b.Open++
		blocked := false
		for _, d := range is.Dependencies {
			if d.Type == "blocks" && status[d.DependsOnID] != "" && status[d.DependsOnID] != "closed" {
				blocked = true
			}
		}
		if is.Status == "open" && !blocked {
			b.Ready++
		}
	}
	return b
}
