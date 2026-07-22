package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/lint"
	"github.com/RamXX/machinery/internal/pack"
)

// Thresholds for the decomposition recommendation. Deliberately conservative
// defaults; they are advisory (the report recommends, the human decides), and
// they should be recalibrated as real runs accumulate evidence about where
// synthesis quality actually degrades.
const (
	scaleShardMachines  = 10      // SKILL's sharding rule
	scaleRecurseTokens  = 100_000 // estimated synthesis input beyond one context
	scaleRecurseEntWide = 25      // a domain model this wide rarely has one language
)

func newScaleCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "scale <design>",
		Short: "Measure a design's size and recommend sharding or recursive decomposition",
		Args:  cobra.ExactArgs(1),
	}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		design := args[0]
		if err := checkIsDir(design); err != nil {
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
			return nil
		}
		// refuse to measure a directory that is not a design: an empty dir
		// once produced a confident "single-run design" recommendation
		if !pack.LooksLikeDesignDir(design) {
			fmt.Fprintf(stderrW, "scale: %s contains no *.modelith.yaml, no machines/, and no decomposition.yaml; not a machinery design directory, nothing to measure\n", design)
			exitFunc(1)
			return nil
		}
		out := stdoutW
		fmt.Fprintln(out, "== scale  design size and decomposition signals ==")

		machines := 0
		states, transitions := 0, 0
		mdir := filepath.Join(design, "machines")
		if entries, err := os.ReadDir(mdir); err == nil {
			for _, e := range entries {
				if !strings.HasSuffix(e.Name(), ".machine.json") {
					continue
				}
				machines++
				if m, err := ir.LoadMachineJSON(filepath.Join(mdir, e.Name())); err == nil {
					_, _, _, counts := lint.LintMachine(m, e.Name())
					states += counts.States
					transitions += counts.Transitions
				}
			}
		}
		entities, invariants := 0, 0
		var modelithBytes int
		if entries, err := os.ReadDir(design); err == nil {
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".modelith.yaml") {
					data, _ := os.ReadFile(filepath.Join(design, e.Name()))
					modelithBytes += len(data)
					if v, err := ir.LoadYAML(data); err == nil && v.AsObject() != nil {
						ents := v.AsObject().GetObject("entities")
						entities += ents.Len()
						invariants += len(v.AsObject().Get2("invariants").AsArray())
						for _, en := range ents.Keys() {
							invariants += len(ents.Get2(en).AsObject().Get2("invariants").AsArray())
						}
					}
				}
			}
		}
		eventRows := len(pack.EventRows(design))
		inputBytes := modelithBytes
		for _, f := range []string{"ARCHITECTURE.md", "workspace.dsl"} {
			if data, err := os.ReadFile(filepath.Join(design, f)); err == nil {
				inputBytes += len(data)
			}
		}
		if entries, err := os.ReadDir(mdir); err == nil {
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".matrix.md") {
					if data, err := os.ReadFile(filepath.Join(mdir, e.Name())); err == nil {
						inputBytes += len(data)
					}
				}
			}
		}
		estTokens := inputBytes / 4

		fmt.Fprintf(out, "  machines: %d (states %d, transitions %d)\n", machines, states, transitions)
		fmt.Fprintf(out, "  entities: %d, invariants: %d, boundary event rows: %d\n", entities, invariants, eventRows)
		fmt.Fprintf(out, "  estimated synthesis input: ~%dk tokens (modelith + architecture + matrices)\n", estTokens/1000)
		if pack.HasDecomposition(design) {
			fmt.Fprintln(out, "  decomposition: PARENT (decomposition.yaml present)")
		}
		if pack.HasPack(design) {
			fmt.Fprintln(out, "  decomposition: CHILD (pack/ present)")
		}

		var recs []string
		if machines > scaleShardMachines {
			recs = append(recs, fmt.Sprintf("shard the synthesis: %d stateful components exceeds the ~%d-per-run rule (SKILL: Sharding large designs)", machines, scaleShardMachines))
		}
		if estTokens > scaleRecurseTokens {
			recs = append(recs, fmt.Sprintf("consider recursion: the synthesis input (~%dk tokens) exceeds a single-context budget; if the ubiquitous language forks across areas, split into subsystems with contract packs (machinery pack)", estTokens/1000))
		}
		if entities > scaleRecurseEntWide {
			recs = append(recs, fmt.Sprintf("consider recursion: %d entities rarely share one ubiquitous language; probe for context boundaries (the same word meaning different things is the split signal)", entities))
		}
		if len(recs) == 0 {
			fmt.Fprintln(out, "  recommendation: single-run design; neither sharding nor recursion is indicated")
		} else {
			for _, r := range recs {
				fmt.Fprintln(out, "  recommendation: "+r)
			}
			fmt.Fprintln(out, "  note: shard first (cheap: one design tree); recurse only when the domain model itself no longer fits one conversation. Recursion is the contract-pack protocol: machinery pack generate / pack refine / check --gate g5.")
		}
		return nil
	}
	return c
}
