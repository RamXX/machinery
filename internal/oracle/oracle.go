// Package oracle is the Go port of oracle_gen.py: generates the canonical
// transition oracle from a machine JSON, with content-derived stable ids.
package oracle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
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
//
//	sha256("{tag}|{source}|{trig}|{guard or ''}").hex()[:6]
//	return f"{tag}-{h}"
//
// Render extends the 6-hex prefix per colliding group when two DIFFERENT
// stimuli share it (a 24-bit collision would otherwise silently mislabel a
// distinct transition as a duplicate branch).
func StableID(tag, source, trig, guard string) string {
	return tag + "-" + stableHash(tag, source, trig, guard)[:6]
}

func stableHash(tag, source, trig, guard string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s", tag, source, trig, guard)))
	return hex.EncodeToString(h[:])
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
	if r := []rune(tag); len(r) > 4 {
		tag = string(r[:4]) // rune-safe: a multibyte id must not be split mid-rune
	}

	var L []string
	L = append(L, fmt.Sprintf("# Generated transition oracle: `%s`", mid))
	L = append(L, "")
	L = append(L, fmt.Sprintf("Generated from `%s` by `machinery oracle`. DO NOT EDIT BY HAND.", filepath.Base(sourceName)))
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
	// pass 1: collect rows and their full stimulus hashes
	type row struct {
		source, trig, guard, target, actions string
		hash                                 string
	}
	var rows []row
	for _, s := range states {
		for _, tr := range ir.TransitionsOf(s.Node, nil, s.Path) {
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
			rows = append(rows, row{source: s.Path, trig: trig, guard: guard, target: target,
				actions: fmtActions(nilIfEmptyList(tr.Actions)), hash: stableHash(tag, s.Path, trig, tr.Guard)})
		}
	}
	// pass 2: per 6-hex prefix group, extend the prefix until distinct stimuli
	// get distinct ids; identical stimuli (duplicate branches, a lint error)
	// keep the positional .2/.3 suffix
	distinctByPrefix := map[string]map[string]bool{}
	for _, r := range rows {
		p := r.hash[:6]
		if distinctByPrefix[p] == nil {
			distinctByPrefix[p] = map[string]bool{}
		}
		distinctByPrefix[p][r.hash] = true
	}
	idOf := func(r row) string {
		group := distinctByPrefix[r.hash[:6]]
		n := 6
		for n < len(r.hash) {
			unique := true
			for other := range group {
				if other != r.hash && other[:n] == r.hash[:n] {
					unique = false
					break
				}
			}
			if unique {
				break
			}
			n += 2
		}
		return tag + "-" + r.hash[:n]
	}
	i := 0
	seen := map[string]int{}
	for _, r := range rows {
		i++
		sid := idOf(r)
		if n, ok := seen[sid]; ok {
			seen[sid] = n + 1
			sid = fmt.Sprintf("%s.%d", sid, seen[sid])
		} else {
			seen[sid] = 1
		}
		L = append(L, fmt.Sprintf("| T-%s-%02d | %s | %s | %s | %s | %s | %s |",
			tag, i, sid, r.source, r.trig, r.guard, r.target, r.actions))
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
		return "", fmt.Errorf("oracle_gen: %w", err)
	}
	return Render(m, path), nil
}
