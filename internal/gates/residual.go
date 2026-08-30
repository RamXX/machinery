// Residual-handler binding (Gx, opt-in by column presence). "Every C4
// dependency failure has its residual transition" has been a Gate 3 attested
// line: the mitigation table names what the FSM must handle and nothing bound
// those rows to the machines. The binding: the mitigation table may carry a
// "handled by" column naming, backticked, the machine (or the specific invoke
// actor) that carries each dependency's residual transitions. Both sides are
// then closed: the row names are held to the committed machines and their
// invoke actors, and a dependency whose cell is empty must waive with a
// reason. Whether the named machine's transitions are semantically adequate
// stays attested, as everywhere.

package gates

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// noResidualWaiverRe waives one mitigation row's handler: the residual needs
// no machine transition (a fatal-and-loud posture, say). Distinct from every
// other waiver token, as always.
var noResidualWaiverRe = regexp.MustCompile(`\(no residual:\s*([^)]*)\)`)

// machineInvokeSrcs collects the invoke src actors of every committed
// machine. Load failures are ignored here: G3 owns reporting them, and a
// machine that does not parse contributes no actors.
func machineInvokeSrcs(design string) map[string]bool {
	srcs := map[string]bool{}
	for _, path := range sortedGlob(filepath.Join(design, "machines"), "*.machine.json") {
		m, err := ir.LoadMachineJSON(path)
		if err != nil {
			continue
		}
		for _, s := range ir.WalkStates(m.AsObject().Get2("states"), "") {
			if s.Node == nil || s.Node.Kind != ir.KindObject {
				continue
			}
			for _, iv := range ir.InvokesOf(s.Node) {
				if ivo := iv.AsObject(); ivo != nil && ivo.GetString("src") != "" {
					srcs[ivo.GetString("src")] = true
				}
			}
		}
	}
	return srcs
}

// checkResidualHandling holds the mitigation table's "handled by" column,
// when a design adopts one, to the committed machines and their invoke
// actors.
func checkResidualHandling(g *Gate, design string, machineNames map[string]bool) {
	arch := filepath.Join(design, "ARCHITECTURE.md")
	text := readOrEmpty(arch) // read errors reported by the placement pass
	var srcs map[string]bool  // computed lazily: only an adopted column needs it
	for _, tbl := range ir.ParseMdTables(text) {
		hl := strings.ToLower(strings.Join(tbl.Header, " "))
		if !strings.Contains(hl, "failure") || !strings.Contains(hl, "mitigation") {
			continue
		}
		hi := colContaining(tbl.Header, "handled")
		if hi < 0 {
			continue // opt-in: a table without the column carries no obligation
		}
		if srcs == nil {
			srcs = machineInvokeSrcs(design)
		}
		for _, r := range tbl.Rows {
			if len(r) == 0 {
				continue
			}
			dep := "(unnamed dependency)"
			if m := mitTokRe.FindStringSubmatch(r[0]); m != nil {
				dep = "`" + m[1] + "`"
			}
			cell := ""
			if hi < len(r) {
				cell = r[hi]
			}
			if m := noResidualWaiverRe.FindStringSubmatch(cell); m != nil {
				if strings.TrimSpace(m[1]) == "" {
					g.Errs = append(g.Errs, "residual-handler waiver for "+dep+" names no reason; write '(no residual: <reason>)'")
				} else {
					g.Count("residual handlers waived")
				}
				continue
			}
			handlers := mitTokRe.FindAllStringSubmatch(parenAnnotationRe.ReplaceAllString(cell, " "), -1)
			if len(handlers) == 0 {
				g.Errs = append(g.Errs, "mitigation row "+dep+" names no residual handler in backticks and no '(no residual: <reason>)' waiver; name the machine or invoke actor that carries its residual transitions")
				continue
			}
			ok := true
			for _, m := range handlers {
				tok := m[1]
				if !machineNames[tok] && !srcs[tok] {
					g.Errs = append(g.Errs, "mitigation row "+dep+" residual handler `"+tok+"` is neither a committed machine nor an invoke actor of one; the handler column binds the row to the behavior layer")
					ok = false
				}
			}
			if ok {
				g.Count("mitigation rows with residual handlers")
			}
		}
	}
}
