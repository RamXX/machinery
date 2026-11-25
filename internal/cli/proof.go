package cli

import (
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/evidence"
)

func init() {
	Register(&Command{Name: "proof", Summary: "Print a proof block for the selected obligations.", Run: runProof})
}

func runProof(e *Env, args []string) int {
	fs := e.Flags("proof")
	var ids, prefixes StringList
	fs.Var(&ids, "ids", "comma-separated obligation ids (repeatable)")
	fs.Var(&prefixes, "prefix", "id prefix (repeatable)")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery proof takes no arguments")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	d := design.Load(p)
	sel, err := evidence.SelectIDs(d, splitIDs(ids), prefixes)
	if err != nil {
		return e.Usage("%v", err)
	}
	ev, err := evidence.Evaluate(p, d, false)
	if err != nil {
		return e.Fail(err)
	}
	return evidence.Proof(e.Stdout, d, ev, sel)
}

// splitIDs splits comma-separated flag values.
func splitIDs(vals []string) []string {
	var res []string
	for _, v := range vals {
		for _, id := range strings.Split(v, ",") {
			if id = strings.TrimSpace(id); id != "" {
				res = append(res, id)
			}
		}
	}
	return res
}
