package formal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type alloyRunner struct {
	java string
	jar  string
}

type alloyCommand struct {
	Type     string `json:"type"`
	Solution []struct {
		Instances []struct {
			Values map[string]map[string][][]string `json:"values"`
		} `json:"instances"`
	} `json:"solution"`
}

type alloyReceipt struct {
	Commands map[string]alloyCommand `json:"commands"`
}

var alloyCommandRe = regexp.MustCompile(`(?m)^\s*(check|run)\s+\w`)

// exec runs the Alloy file (props text plus commands) in dir and returns the
// commands that found an instance. errLine is non-empty when Alloy could not
// check the file.
func (a *alloyRunner) exec(dir, fileName string, timeout time.Duration) (found map[string]map[string]map[string][]string, known map[string]bool, errLine string) {
	out := filepath.Join(dir, "out")
	os.RemoveAll(out)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, a.java, "-jar", a.jar, "exec", "-f", "-q", "-o", out, "-t", "json", fileName)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	runErr := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, nil, fmt.Sprintf("timed out after %s", timeout)
	}
	data, err := os.ReadFile(filepath.Join(out, "receipt.json"))
	if err != nil {
		line := "alloy did not finish"
		for _, l := range strings.Split(buf.String(), "\n") {
			if l = strings.TrimSpace(l); l != "" {
				line = l
				break
			}
		}
		if runErr != nil && line == "alloy did not finish" {
			line += ": " + runErr.Error()
		}
		return nil, nil, line
	}
	var rc alloyReceipt
	if err := json.Unmarshal(data, &rc); err != nil {
		return nil, nil, "alloy wrote an unreadable receipt"
	}
	found = map[string]map[string]map[string][]string{}
	known = map[string]bool{}
	for name, c := range rc.Commands {
		known[name] = true
		if len(c.Solution) == 0 || len(c.Solution[0].Instances) == 0 {
			continue
		}
		inst := map[string]map[string][]string{}
		for atom, fields := range c.Solution[0].Instances[0].Values {
			if !strings.Contains(atom, "$") {
				continue // integers and other built-in atoms
			}
			fm := map[string][]string{}
			for f, tuples := range fields {
				vals := []string{}
				for _, tup := range tuples {
					parts := make([]string, len(tup))
					for i, x := range tup {
						parts[i] = atomName(x)
					}
					vals = append(vals, strings.Join(parts, "->"))
				}
				sort.Strings(vals)
				fm[f] = vals
			}
			inst[atomName(atom)] = fm
		}
		found[name] = inst
	}
	return found, known, ""
}

// atomName strips the module prefix and the "$" of an Alloy atom ("shop/Order$0" is "Order0").
func atomName(atom string) string {
	if i := strings.LastIndex(atom, "/"); i >= 0 {
		atom = atom[i+1:]
	}
	return strings.ReplaceAll(atom, "$", "")
}

func alloyScope(r *Run) int {
	if v, ok := r.Constants["scope"]; ok {
		return intOf(v, 4)
	}
	return 4
}
