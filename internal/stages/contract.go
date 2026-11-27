package stages

import (
	"github.com/RamXX/machinery/v2/internal/pm"
	"github.com/RamXX/machinery/v2/internal/report"
)

func init() { Bind("contract", ContractStage) }

// ContractStage is stage contract (section 15.4).
func ContractStage(ctx *Context) *report.Stage {
	s := report.New("contract")
	w, ok, err := pm.LoadWave(ctx.P)
	if err != nil {
		s.Errorf("%v", err)
		return s
	}
	if !ok {
		return s.NotApplicable("no wave is frozen")
	}
	c := pm.ContractOf(ctx.Design())
	s.Count(len(c.Paths), "design files")
	for _, path := range pm.ChangedFiles(w, c) {
		s.Errorf("contract changed during fan-out: %s", path)
	}
	return s
}
