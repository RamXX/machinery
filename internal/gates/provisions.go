package gates

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// Composability over declared capability keys (optional contract vocabulary):
// a boundary may declare `provides: [key, ...]` and `consumes: [key, ...]`.
// Two laws become checkable at design time, before any code exists:
//
//  1. Disjoint provisions: every key has at most one provider. Two providers
//     for one key is an ambiguous system: which one a consumer binds to is an
//     accident of wiring.
//  2. Satisfied consumption: every consumed key has a provider, and the
//     consumer holds a direct allow edge to that provider. Consuming a key
//     without the dependency edge is a contradiction between the capability
//     view and the dependency view of the same architecture.
//
// The remaining composability laws (activation ordering, a provider
// withdrawing a key only after every dependent has deactivated) are runtime
// lifecycle properties this static vocabulary deliberately does not model;
// they are NOT checked here, and no output of this gate should be read as
// covering them. Contracts without provides/consumes check exactly as before.
var regexpCapabilityKey = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func checkProvisions(boundaries []*ir.Value, declared map[string]bool, allow [][2]string, g *Gate) {
	providerOf := map[string][]string{}
	consumers := map[string][]string{}
	edge := map[[2]string]bool{}
	for _, e := range allow {
		edge[e] = true
	}

	keysOf := func(bo *ir.Object, field, id string) []string {
		v := bo.Get2(field)
		if v == nil {
			return nil
		}
		if v.Kind != ir.KindArray {
			g.Errs = append(g.Errs, fmt.Sprintf("boundary %s: %s must be a list of capability keys", ir.Repr(id), field))
			return nil
		}
		var out []string
		seen := map[string]bool{}
		for _, e := range v.AsArray() {
			if e == nil || e.Kind != ir.KindString || !regexpCapabilityKey.MatchString(e.AsString()) {
				g.Errs = append(g.Errs, fmt.Sprintf("boundary %s: %s entries must be kebab-case keys, got %s", ir.Repr(id), field, goReprValue(e)))
				continue
			}
			k := e.AsString()
			if seen[k] {
				g.Errs = append(g.Errs, fmt.Sprintf("boundary %s: duplicate %s key %s", ir.Repr(id), field, ir.Repr(k)))
				continue
			}
			seen[k] = true
			out = append(out, k)
		}
		return out
	}

	for _, b := range boundaries {
		bo := b.AsObject()
		if bo == nil {
			continue
		}
		id := bo.GetString("id")
		if id == "" || !declared[id] {
			continue
		}
		for _, k := range keysOf(bo, "provides", id) {
			providerOf[k] = append(providerOf[k], id)
			g.Count("provided keys")
		}
		for _, k := range keysOf(bo, "consumes", id) {
			consumers[k] = append(consumers[k], id)
			g.Count("consumed keys")
		}
	}

	for _, k := range sortedKeys(providerOf) {
		if len(providerOf[k]) > 1 {
			sort.Strings(providerOf[k])
			g.Errs = append(g.Errs, fmt.Sprintf(
				"capability key %s has %d providers (%s); provisions must be disjoint, one provider per key",
				ir.Repr(k), len(providerOf[k]), strings.Join(providerOf[k], ", ")))
		}
	}
	for _, k := range sortedKeys(consumers) {
		providers := providerOf[k]
		if len(providers) == 0 {
			sort.Strings(consumers[k])
			g.Errs = append(g.Errs, fmt.Sprintf(
				"capability key %s is consumed by %s but no boundary provides it",
				ir.Repr(k), strings.Join(consumers[k], ", ")))
			continue
		}
		if len(providers) != 1 {
			continue // already reported as non-disjoint above
		}
		provider := providers[0]
		for _, consumer := range consumers[k] {
			if consumer == provider {
				g.Errs = append(g.Errs, fmt.Sprintf(
					"boundary %s both provides and consumes key %s; a self-binding key declares nothing",
					ir.Repr(consumer), ir.Repr(k)))
				continue
			}
			if !edge[[2]string{consumer, provider}] {
				g.Errs = append(g.Errs, fmt.Sprintf(
					"boundary %s consumes key %s provided by %s, but dependency_rules.allow has no %s -> %s edge; the capability view and the dependency view disagree",
					ir.Repr(consumer), ir.Repr(k), ir.Repr(provider), consumer, provider))
			}
		}
	}
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
