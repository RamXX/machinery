// Alloy orchestration for the opted-in static relational proof layers: fetch
// the pinned Alloy dist jar, run `exec` headless, and read the verdicts from
// its receipt. TLC checks machine behavior; Alloy checks admissible static
// configurations.

package formal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/alloy"
)

const (
	alloyVersion = "v6.2.0"
	alloySHA256  = "6b8c1cb5bc93bedfc7c61435c4e1ab6e688a242dc702a394628d9a9801edb78d"
)

// alloyJarPath resolves the pinned Alloy dist jar location (env override honored).
func alloyJarPath() string {
	if j := os.Getenv("ALLOY_TOOLS_JAR"); j != "" {
		return j
	}
	cache, _ := os.UserCacheDir()
	if cache == "" {
		cache = os.TempDir()
	}
	return filepath.Join(cache, "machinery", "alloy-dist-"+alloyVersion+".jar")
}

func ensureAlloyJar() (string, error) {
	return fetchJar(alloyJarPath(),
		"https://github.com/AlloyTools/org.alloytools.alloy/releases/download/"+alloyVersion+"/org.alloytools.alloy.dist.jar",
		"org.alloytools.alloy.dist.jar "+alloyVersion, alloySHA256)
}

// --- receipt.json (what `alloy exec` writes next to the solutions) ---

type alloyRelation [][]string

type alloyInstance struct {
	Skolems map[string]struct {
		Data alloyRelation `json:"data"`
	} `json:"skolems"`
	Values map[string]map[string]alloyRelation `json:"values"`
}

type alloySolution struct {
	Instances []alloyInstance `json:"instances"`
}

type alloyCommandResult struct {
	Type     string          `json:"type"` // check | run
	Solution []alloySolution `json:"solution"`
}

type alloyReceipt struct {
	Commands map[string]alloyCommandResult `json:"commands"`
}

func (r alloyCommandResult) sat() bool {
	for _, s := range r.Solution {
		if len(s.Instances) > 0 {
			return true
		}
	}
	return false
}

// AlloyVerdict is one command's outcome, in the model's command order.
type AlloyVerdict struct {
	Command alloy.Command
	Pass    bool
	Detail  string // counterexample or vacuity note on failure
}

// runAlloy executes every command in alsPath and maps the receipt back onto
// the generated command list (kind decides pass semantics: check passes on
// UNSAT, run passes on SAT). Solutions are written as text (-t text): the
// receipt's per-atom values are unreliable for inherited and total relations,
// while the text form carries every relation in full.
func runAlloy(alsPath string, commands []alloy.Command) ([]AlloyVerdict, error) {
	jar, err := ensureAlloyJar()
	if err != nil {
		return nil, err
	}
	outDir, err := os.MkdirTemp("", "machinery-alloy")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(outDir)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "java", "-jar", jar, "exec", "-f", "-t", "text", "-c", "*", "-o", outDir, filepath.Base(alsPath))
	cmd.Dir = filepath.Dir(alsPath)
	var buf strings.Builder
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("alloy exec failed on %s: %w\n%s", filepath.Base(alsPath), err, tail(buf.String(), 20))
	}
	raw, err := os.ReadFile(filepath.Join(outDir, "receipt.json"))
	if err != nil {
		return nil, fmt.Errorf("alloy exec wrote no receipt.json for %s: %w", filepath.Base(alsPath), err)
	}
	var receipt alloyReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return nil, fmt.Errorf("receipt.json for %s does not parse: %w", filepath.Base(alsPath), err)
	}
	detail := func(name string) string {
		sol, rerr := os.ReadFile(filepath.Join(outDir, name+"-solution-0.txt"))
		if rerr != nil {
			return ""
		}
		return renderSolutionText(string(sol))
	}
	return verdicts(receipt, commands, detail)
}

// verdicts maps the receipt onto the generated command list. A command the
// receipt does not mention is an error, never a silent pass. detail renders
// a counterexample summary for a failed check ("" is acceptable).
func verdicts(receipt alloyReceipt, commands []alloy.Command, detail func(name string) string) ([]AlloyVerdict, error) {
	var out []AlloyVerdict
	for _, c := range commands {
		res, ok := receipt.Commands[c.Name]
		if !ok {
			return nil, fmt.Errorf("receipt.json has no result for command %s; the model and the run disagree", c.Name)
		}
		v := AlloyVerdict{Command: c}
		switch c.Kind {
		case "check":
			v.Pass = !res.sat()
			if !v.Pass && detail != nil {
				v.Detail = detail(c.Name)
			}
		default: // run
			v.Pass = res.sat()
			if !v.Pass {
				v.Detail = "no instance within scope: the asserted possibility does not exist in any admissible world (vacuous or contradictory policy)"
			}
		}
		out = append(out, v)
	}
	return out, nil
}

// renderSolutionText compresses a `-t text` solution into one line: skolem
// witnesses first, then each atom with its fields ("(none)" for an atom a
// total-looking field skips, e.g. a teamless subject).
func renderSolutionText(raw string) string {
	type sigInfo struct {
		name   string
		atoms  []string
		fields []string
	}
	var sigs []*sigInfo
	byName := map[string]*sigInfo{}
	atomFields := map[string][]string{} // atom -> rendered "f=v" in field order
	var skolems []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "$") { // skolem witness: $X_u={User$5}
			skolems = append(skolems, strings.TrimPrefix(line, "$"))
			continue
		}
		if !strings.HasPrefix(line, "this/") {
			continue
		}
		lhs, rest, found := strings.Cut(strings.TrimPrefix(line, "this/"), "={")
		if !found {
			continue
		}
		set := strings.TrimSuffix(rest, "}")
		if sig, field, ok := strings.Cut(lhs, "<:"); ok {
			si := byName[sig]
			if si == nil {
				continue
			}
			si.fields = append(si.fields, field)
			seen := map[string]bool{}
			for _, tuple := range splitSet(set) {
				atom, val, ok := strings.Cut(tuple, "->")
				if !ok {
					continue
				}
				atomFields[atom] = append(atomFields[atom], field+"="+val)
				seen[atom] = true
			}
			for _, atom := range si.atoms {
				if !seen[atom] {
					atomFields[atom] = append(atomFields[atom], field+"=(none)")
				}
			}
			continue
		}
		si := &sigInfo{name: lhs, atoms: splitSet(set)}
		sigs = append(sigs, si)
		byName[lhs] = si
	}
	parts := append([]string{}, skolems...)
	const maxAtoms = 10
	total := 0
	for _, si := range sigs {
		if len(si.fields) == 0 {
			continue
		}
		for _, atom := range si.atoms {
			if total >= maxAtoms {
				parts = append(parts, "(...)")
				return "counterexample: " + strings.Join(parts, " ")
			}
			parts = append(parts, fmt.Sprintf("%s{%s}", atom, strings.Join(atomFields[atom], ", ")))
			total++
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "counterexample: " + strings.Join(parts, " ")
}

func splitSet(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, x := range strings.Split(s, ",") {
		out = append(out, strings.TrimSpace(x))
	}
	return out
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
