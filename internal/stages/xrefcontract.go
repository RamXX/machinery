package stages

import (
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/contract"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
)

func init() { Bind("xref-contract", XrefContract) }

var xrefContractRules = []string{"operation-id", "operation-action", "action-operation"}

// XrefContract is stage xref-contract (section 6.5).
func XrefContract(ctx *Context) *report.Stage {
	s := report.New("xref-contract")
	p := ctx.P
	if len(p.Cfg.Contracts) == 0 {
		return s.NotApplicable("no contracts configured")
	}
	d := ctx.Design()
	ops, problems := contract.Load(p)
	for _, pr := range problems {
		s.Errorf("%s", pr)
	}
	w := waiversFor(p, xrefContractRules...)
	bound, named := 0, map[string]bool{}
	for _, op := range ops {
		if op.ID == "" {
			w.find(s, "operation-id", op.Subject(), "%s has no operation name (%s); give the operation an operationId or the rpc a name", op.Contract, opKind(op))
		}
		if len(op.Actions) > 0 {
			bound++
		}
		for _, a := range op.Actions {
			id := "act:" + a
			named[id] = true
			if !d.IsAct(id) {
				w.find(s, "operation-action", id, "operation %s in %s names an action that is not in the model", op.Subject(), op.Contract)
			}
		}
	}
	for _, sv := range d.Served {
		if !named[sv.ID] {
			w.find(s, "action-operation", sv.ID, "no operation names this action; bind one with %s", bindingHint(p.Cfg.Contracts))
		}
	}
	w.stale(s)
	s.Count(len(p.Cfg.Contracts), "contracts").Count(len(ops), "operations").Count(bound, "bound operations").Count(len(d.Served), "actions")
	if len(ops) == 0 && len(d.Served) == 0 && len(problems) == 0 {
		s.NotDemonstrated("nothing to check")
	}
	return s
}

func opKind(op contract.Operation) string {
	if op.Method == "RPC" {
		return "rpc"
	}
	return "operation"
}

// bindingHint names the binding syntax of the configured contract kinds.
func bindingHint(cs []project.Contract) string {
	var kinds []string
	for _, c := range cs {
		switch c.Kind {
		case "openapi":
			kinds = append(kinds, "x-machinery-action")
		case "proto":
			kinds = append(kinds, "// machinery:action")
		}
	}
	return strings.Join(uniqueSorted(kinds), " or ")
}

func uniqueSorted(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range in {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}
