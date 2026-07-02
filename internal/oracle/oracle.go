// Package oracle is the Go port of oracle_gen.py: generates the canonical
// transition oracle from a machine JSON, with content-derived stable ids.
package oracle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ramirosalas/machinery/internal/ir"
)

// fmtActions mirrors oracle_gen._fmt: ", ".join(names) or "-".
func fmtActions(v *ir.Value) string {
	names := ir.ActionNames(v, nil, "")
	if len(names) == 0 {
		return "-"
	}
	return strings.Join(names, ", ")
}

// StableID mirrors oracle_gen.stable_id:
//   sha256("{tag}|{source}|{trig}|{guard or ''}").hex()[:6]
//   return f"{tag}-{h}"
func StableID(tag, source, trig, guard string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s", tag, source, trig, guard)))
	return tag + "-" + hex.EncodeToString(h[:])[:6]
}

// Render mirrors oracle_gen.render(m, source_name). Pure.
// Returns the oracle markdown body (no trailing newline beyond what Python emits).
func Render(m *ir.Value, sourceName string) string {
	mid := m.AsObject().GetString("id")
	if mid == "" {
		base := filepath.Base(sourceName)
		mid = strings.TrimSuffix(base, filepath.Ext(base))
	}
	states := ir.WalkStates(m.AsObject().Get2("states"), "")
	tag := strings.ToUpper(mid)
	if len(tag) > 4 {
		tag = tag[:4]
	}

	var L []string
	L = append(L, fmt.Sprintf("# Generated transition oracle: `%s`", mid))
	L = append(L, "")
	L = append(L, fmt.Sprintf("Generated from `%s` by tools/oracle_gen.py. DO NOT EDIT BY HAND.", filepath.Base(sourceName)))
	L = append(L, "Single source of truth for the hard-TDD transition tests: one transition row is one")
	L = append(L, "test case. Key tests on the STABLE id, not the row number; row numbers renumber when")
	L = append(L, "the design changes, stable ids do not.")
	L = append(L, "")
	L = append(L, "## State entry / exit actions")
	L = append(L, "")
	L = append(L, "| state | kind | entry | exit |")
	L = append(L, "|---|---|---|---|")
	for _, s := range states {
		if strings.Contains(s.Path, ".") {
			continue
		}
		o := s.Node.AsObject()
		kind := o.GetString("type")
		if kind == "" {
			kind = "atomic"
		}
		L = append(L, fmt.Sprintf("| %s | %s | %s | %s |", s.Path, kind, fmtActions(o.Get2("entry")), fmtActions(o.Get2("exit"))))
	}
	L = append(L, "")
	L = append(L, "## Transitions")
	L = append(L, "")
	L = append(L, "| test id | stable id | source | trigger | guard | target | actions |")
	L = append(L, "|---|---|---|---|---|---|---|")
	i := 0
	seen := map[string]int{}
	for _, s := range states {
		for _, tr := range ir.TransitionsOf(s.Node, nil, s.Path) {
			i++
			var trig string
			if tr.Event != "" {
				trig = tr.Kind + ":" + tr.Event
			} else {
				trig = tr.Kind
			}
			guard := tr.Guard
			if guard == "" {
				guard = "-"
			}
			target := tr.Target
			if target == "" {
				target = "(internal)"
			}
			sid := StableID(tag, s.Path, trig, tr.Guard)
			if n, ok := seen[sid]; ok {
				seen[sid] = n + 1
				sid = fmt.Sprintf("%s.%d", sid, seen[sid])
			} else {
				seen[sid] = 1
			}
			L = append(L, fmt.Sprintf("| T-%s-%02d | %s | %s | %s | %s | %s | %s |",
				tag, i, sid, s.Path, trig, guard, target, fmtActions(nilIfEmptyList(tr.Actions))))
		}
	}
	L = append(L, "")
	L = append(L, fmt.Sprintf("Total transitions (test cases): %d", i))
	L = append(L, "")
	return strings.Join(L, "\n")
}

// nilIfEmptyList wraps a string slice into a *Value for fmtActions (which expects
// the raw actions value). We reconstruct as an array of strings to reuse fmtActions.
func nilIfEmptyList(acts []string) *ir.Value {
	if len(acts) == 0 {
		return nil // _fmt(None) -> "-"
	}
	arr := make([]*ir.Value, len(acts))
	for i, a := range acts {
		arr[i] = ir.StringValue(a)
	}
	return ir.ArrayValue(arr)
}

// Generate loads a machine file and renders its oracle. Mirrors oracle_gen.generate;
// returns ("oracle_gen: <err>", err) style via the error.
func Generate(path string) (string, error) {
	m, err := ir.LoadMachineJSON(path)
	if err != nil {
		return "", fmt.Errorf("oracle_gen: %s", err)
	}
	return Render(m, path), nil
}
