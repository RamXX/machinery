package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/pack"
)

// CheckFinalHandoff closes the phase-artifact universe. Ordinary staged runs
// legitimately omit later artifacts; --complete is the point at which that
// omission stops being a skip and becomes a blocking handoff defect.
func CheckFinalHandoff(design string) *Gate {
	g := NewGate("G!-complete  final-handoff completeness")
	requireFile := func(name string) {
		fi, err := os.Stat(filepath.Join(design, name))
		if err != nil || fi.IsDir() {
			g.Errs = append(g.Errs, name+" is required for final handoff")
			return
		}
		g.Count("required phase artifacts")
	}
	if !HasModelith(design) {
		g.Errs = append(g.Errs, "a root *.modelith.yaml is required for final handoff")
	} else {
		g.Count("required phase artifacts")
	}
	modelSources, err := sortedGlobExt(design, ".modelith.yaml")
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
	}
	legacySources, err := sortedGlobExt(filepath.Join(design, "legacy"), ".modelith.yaml")
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
	}
	modelSources = append(modelSources, legacySources...)
	for _, source := range modelSources {
		render := strings.TrimSuffix(source, ".yaml") + ".md"
		fi, err := os.Stat(render)
		if err != nil || fi.IsDir() {
			rel, _ := filepath.Rel(design, render)
			g.Errs = append(g.Errs, filepath.ToSlash(rel)+" is required for final handoff; render the corresponding Modelith source and commit it")
		} else {
			g.Count("committed modelith renders")
		}
	}
	for _, name := range []string{"workspace.dsl", "ARCHITECTURE.md", "BUILD.md", AttestationsFileName} {
		requireFile(name)
	}
	if !HasMachines(design) && !pack.HasDecomposition(design) {
		g.Errs = append(g.Errs, "machines/*.machine.json is required for final handoff")
	} else {
		g.Count("required phase artifacts")
	}
	if HasHumanActions(design) {
		requireFile(TargetSurfacesName)
	}
	docs := planDocuments(design, g)
	milestones := 0
	closed := 0
	for _, doc := range docs {
		for _, m := range doc.milestones {
			if !m.numOK {
				continue
			}
			milestones++
			if !m.closed() {
				g.Errs = append(g.Errs, fmt.Sprintf("%s milestone M%d is not Status: closed; final handoff requires every declared milestone accepted", doc.name, m.num))
			} else {
				closed++
			}
		}
	}
	if milestones == 0 {
		g.Errs = append(g.Errs, "the Build plan declares no milestones; final handoff has no completed work units")
	}
	g.Count("milestones closed", closed)
	return g
}
