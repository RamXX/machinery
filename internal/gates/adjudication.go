// Gj-adjudication: characterization-verdict evidence. Brownfield Stage 3
// adjudicates every failing characterization row with an explicit verdict
// (code-is-truth: the model was wrong archaeology, fix it; model-is-truth:
// the code has a defect, file it and quarantine the test), and until now the
// verdicts lived in PR prose, checked by nobody. This gate gives them the
// Ga-accept treatment: one committed evidence file per machine under
// design/adjudications/, each verdict bound to a committed oracle stable id.
// The gate proves the adjudication happened, on real rows, in a well-formed
// record; whether each verdict is RIGHT stays attested, exactly like Ga's
// review quality.

package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/ir"
)

// AdjudicationDirName is the evidence directory Gj activates on.
const AdjudicationDirName = "adjudications"

var (
	adjudicationRootKeys = stringSet("adjudication_version", "machine", "rows", "_comment")
	adjudicationRowKeys  = stringSet("id", "verdict", "date", "note", "defect", "_comment")
	adjudicationVerdicts = stringSet("code-is-truth", "model-is-truth")
)

// AdjudicationActive reports whether the design carries adjudication
// evidence.
func AdjudicationActive(design string) bool {
	fi, err := os.Stat(filepath.Join(design, AdjudicationDirName))
	return err == nil && fi.IsDir()
}

// CheckAdjudications implements Gj-adjudication.
func CheckAdjudications(design string) *Gate {
	g := NewGate("Gj-adjudication  characterization verdicts")
	g.startOrder()
	dir := filepath.Join(design, AdjudicationDirName)
	if !AdjudicationActive(design) {
		g.Errs = append(g.Errs, "no "+AdjudicationDirName+"/ in the design; the adjudication gate was requested but no verdict evidence was committed")
		return g
	}
	files := sortedGlobExt(dir, ".yaml")
	if len(files) == 0 {
		g.Errs = append(g.Errs, AdjudicationDirName+"/ exists but holds no *.yaml; an empty evidence directory is a failure, not a pass")
		return g
	}
	removed := loadRemovedIDs(design)
	for _, path := range files {
		checkAdjudicationFile(g, design, path, removed)
	}
	g.RequireNonzero("adjudicated rows", "no verdict row was checked")
	return g
}

// checkAdjudicationFile validates one machine's verdict file: the machine
// exists, every row is complete, every id resolves to that machine's
// committed oracle (or the removed-ids allowance), and no id is adjudicated
// twice.
func checkAdjudicationFile(g *Gate, design, path string, removed map[string]bool) {
	label := AdjudicationDirName + "/" + filepath.Base(path)
	raw, err := os.ReadFile(path)
	if err != nil {
		g.Errs = append(g.Errs, label+" is unreadable: "+err.Error())
		return
	}
	value, err := ir.LoadYAML(raw)
	if err != nil || value.AsObject() == nil {
		g.Errs = append(g.Errs, label+" is not a yaml mapping")
		return
	}
	root := value.AsObject()
	for _, key := range root.Keys() {
		if !adjudicationRootKeys[key] {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: unsupported key %q (a typo here weakens the evidence)", label, key))
		}
	}
	if ver := root.Get2("adjudication_version"); ver == nil || ver.Kind != ir.KindNumber || string(ver.AsNumber()) != "1" {
		g.Errs = append(g.Errs, label+": adjudication_version must be the integer 1")
		return
	}
	machine := strings.TrimSpace(root.GetString("machine"))
	base := strings.TrimSuffix(filepath.Base(path), ".yaml")
	if machine != base {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: machine %s does not match the file name (%s); the file name keys the evidence", label, ir.Repr(machine), ir.Repr(base)))
		return
	}
	oraclePath := filepath.Join(design, "machines", machine+".oracle.md")
	if fi, err := os.Stat(oraclePath); err != nil || fi.IsDir() {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: no committed oracle machines/%s.oracle.md; adjudication evidence binds to oracle rows, so the machine and its oracle must exist", label, machine))
		return
	}
	_, stableIDs := oracleTableIDs(readFileOrErr(oraclePath, g))
	minted := setOf(stableIDs)
	rows := root.Get2("rows")
	if rows == nil || rows.Kind != ir.KindArray || len(rows.AsArray()) == 0 {
		g.Errs = append(g.Errs, label+": rows must be a non-empty list of verdicts")
		return
	}
	seen := map[string]bool{}
	for i, item := range rows.AsArray() {
		where := fmt.Sprintf("%s rows[%d]", label, i)
		obj := item.AsObject()
		if obj == nil {
			g.Errs = append(g.Errs, where+" is not a mapping")
			continue
		}
		for _, key := range obj.Keys() {
			if !adjudicationRowKeys[key] {
				g.Errs = append(g.Errs, fmt.Sprintf("%s: unsupported key %q", where, key))
			}
		}
		id := strings.TrimSpace(obj.GetString("id"))
		verdict := strings.TrimSpace(obj.GetString("verdict"))
		date := strings.TrimSpace(obj.GetString("date"))
		note := strings.TrimSpace(obj.GetString("note"))
		if id == "" {
			g.Errs = append(g.Errs, where+".id is required: the oracle stable id the verdict adjudicates")
			continue
		}
		if seen[id] {
			g.Errs = append(g.Errs, fmt.Sprintf("%s adjudicates %s twice; one verdict per stable id (git history is the record of prior rounds)", label, id))
		}
		seen[id] = true
		if !minted[id] && !removed[id] {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: %s resolves to no stable id in machines/%s.oracle.md (and is not in removed-ids.txt); a verdict on a row that does not exist adjudicates nothing", where, id, machine))
		}
		if !adjudicationVerdicts[verdict] {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: verdict %s is not code-is-truth or model-is-truth", where, ir.Repr(verdict)))
		}
		if _, derr := time.Parse("2006-01-02", date); derr != nil {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: date %s is not a real YYYY-MM-DD date", where, ir.Repr(date)))
		}
		if note == "" {
			g.Errs = append(g.Errs, where+".note is required: what the adjudication found")
		}
		switch verdict {
		case "model-is-truth":
			if strings.TrimSpace(obj.GetString("defect")) == "" {
				g.Errs = append(g.Errs, where+": a model-is-truth verdict requires defect: the filed defect reference; a code defect nobody filed evaporates")
			}
			g.Count("model-is-truth verdicts")
		case "code-is-truth":
			g.Count("code-is-truth verdicts")
		}
		g.Count("adjudicated rows")
	}
}
