package dialect

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/project"
)

// wild reports whether a name matches a pattern in which "*" stands for any
// run of characters, the empty one too; every other character is itself.
func wild(pattern, name string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == name
	}
	if !strings.HasPrefix(name, parts[0]) {
		return false
	}
	name = name[len(parts[0]):]
	for _, mid := range parts[1 : len(parts)-1] {
		i := strings.Index(name, mid)
		if i < 0 {
			return false
		}
		name = name[i+len(mid):]
	}
	return strings.HasSuffix(name, parts[len(parts)-1])
}

// lastSegment is the name of a state: the last segment of its dotted path.
func lastSegment(p string) string { return p[strings.LastIndex(p, ".")+1:] }

// annotate adds what the entries of import.machines.annotate say about a
// machine (section 5.2.1): every entry whose key is the machine id or a glob
// that matches the source file applies, in key order. A pattern that
// matches no event or state of the machine is a problem that names it.
func (t *translator) annotate(out *ir.Object) {
	var keys []string
	for _, k := range t.im.AnnotateKeys() {
		if k == t.id || project.Match([]string{k}, t.src) {
			keys = append(keys, k)
			t.annotated[k] = true
		}
	}
	if len(keys) == 0 {
		return
	}
	states := ir.WalkStates(out.Get2("states"), "")
	events := map[string]bool{}
	for _, s := range states {
		if o := s.Node.AsObject(); o != nil {
			for _, k := range o.GetObject("on").Keys() {
				events[k] = true
			}
			for _, k := range o.GetObject("_ignores").Keys() {
				events[k] = true
			}
		}
	}
	names := make([]string, 0, len(events))
	for ev := range events {
		names = append(names, ev)
	}
	sort.Strings(names)

	external, failure := map[string]bool{}, map[string]bool{}
	aliases, aliasKey := map[string]string{}, map[string]string{}
	dismissed, dismissKey := ir.NewObject(), map[string]string{}
	t.listed = map[string]string{}
	for _, k := range keys {
		a := t.im.Annotate[k]
		for _, l := range []struct {
			list     string
			patterns []string
			into     map[string]bool
		}{{"external_events", a.ExternalEvents, external}, {"failure_events", a.FailureEvents, failure}} {
			for _, pat := range l.patterns {
				matched := false
				for _, ev := range names {
					if wild(pat, ev) {
						if !l.into[ev] {
							t.listed["_"+l.list+" "+ev] = fmt.Sprintf("import.machines.annotate %s lists it for %s (%s pattern %s)", k, t.src, l.list, pat)
						}
						l.into[ev] = true
						matched = true
					}
				}
				if !matched {
					t.fail("import.machines.annotate %s: %s pattern %s matches no event of the machine; correct the pattern or remove it", k, l.list, pat)
				}
			}
		}
		for _, pat := range a.FailureStates {
			matched := false
			for _, s := range states {
				src := t.sourceOf[s.Path]
				if wild(pat, s.Name) || wild(pat, s.Path) || wild(pat, lastSegment(src)) || wild(pat, src) {
					matched = true
					tagFailure(s.Node.AsObject())
				}
			}
			if !matched {
				t.fail("import.machines.annotate %s: failure_states pattern %s matches no state of the machine; correct the pattern or remove it", k, pat)
			}
		}
		t.dismiss(k, a.NotFailures, events, states, dismissed, dismissKey)
		evs := make([]string, 0, len(a.Aliases))
		for ev := range a.Aliases {
			evs = append(evs, ev)
		}
		sort.Strings(evs)
		for _, ev := range evs {
			act := a.Aliases[ev]
			switch have, dup := aliases[ev]; {
			case !events[ev]:
				t.fail("import.machines.annotate %s: aliases names event %s, which no state of the machine handles or ignores; correct the name or remove the alias", k, ev)
			case dup && have != act:
				t.fail("import.machines.annotate %s and %s give event %s different actions (%s, %s); keep one", aliasKey[ev], k, ev, have, act)
			default:
				aliases[ev], aliasKey[ev] = act, k
			}
		}
	}
	t.addNames(out, "_external_events", external)
	t.addNames(out, "_failure_events", failure)
	t.addAliases(out, aliases, aliasKey)
	t.addDismissed(out, dismissed, dismissKey)
}

// dismiss collects the not_failures of one entry: each names, exactly, an
// event of the machine, or a state by its name or dotted path as the source
// or the translation writes it. An event keeps its name and a state gets
// its translated dotted path.
func (t *translator) dismiss(k string, nf map[string]string, events map[string]bool, states []ir.StateEntry, into *ir.Object, from map[string]string) {
	names := make([]string, 0, len(nf))
	for n := range nf {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		var targets []string
		if events[n] {
			targets = append(targets, n)
		}
		for _, s := range states {
			src := t.sourceOf[s.Path]
			if (n == s.Name || n == s.Path || n == lastSegment(src) || n == src) && (len(targets) == 0 || targets[0] != s.Path) {
				targets = append(targets, s.Path)
			}
		}
		if len(targets) == 0 {
			t.fail("import.machines.annotate %s: not_failures names %s, which is no event and no state of the machine; correct the name or remove it", k, n)
		}
		for _, tg := range targets {
			if other, dup := from[tg]; dup {
				if other != k {
					t.fail("import.machines.annotate %s and %s both dismiss %s; keep one", other, k, n)
				}
				continue
			}
			from[tg] = k
			into.Set(tg, ir.StringValue(nf[n]))
		}
	}
}

// addDismissed adds the dismissals to the root _not_failures, after the
// ones the source has. A name the source dismisses already is a problem.
func (t *translator) addDismissed(out, dismissed *ir.Object, from map[string]string) {
	if dismissed.Len() == 0 {
		return
	}
	res := ir.NewObject()
	if v := out.Get2("_not_failures"); v != nil {
		if v.Kind != ir.KindObject {
			return
		}
		for _, n := range v.AsObject().Keys() {
			res.Set(n, v.AsObject().Get2(n))
		}
	}
	for _, n := range dismissed.Keys() {
		if res.Has(n) {
			t.fail("import.machines.annotate %s dismisses %s, which the source dismisses already; keep one", from[n], n)
			continue
		}
		res.Set(n, dismissed.Get2(n))
	}
	out.Set("_not_failures", ir.ObjectValue(res))
}

// tagFailure adds the tag failure to a state that does not have it. Tags of
// another shape are left to the lint.
func tagFailure(o *ir.Object) {
	if o == nil {
		return
	}
	tag := ir.StringValue("failure")
	switch v := o.Get2("tags"); {
	case v == nil:
		o.Set("tags", tag)
	case v.Kind == ir.KindString && v.AsString() != "failure":
		o.Set("tags", ir.ArrayValue([]*ir.Value{v, tag}))
	case v.Kind == ir.KindArray:
		for _, e := range v.AsArray() {
			if e != nil && e.Kind == ir.KindString && e.AsString() == "failure" {
				return
			}
		}
		o.Set("tags", ir.ArrayValue(append(append([]*ir.Value{}, v.AsArray()...), tag)))
	}
}

// addNames adds names, sorted, to a root list of event names, after the
// ones the source lists. A value of another shape is left to the lint.
func (t *translator) addNames(out *ir.Object, key string, names map[string]bool) {
	if len(names) == 0 {
		return
	}
	var list []*ir.Value
	if v := out.Get2(key); v != nil {
		if v.Kind != ir.KindArray {
			return
		}
		for _, e := range v.AsArray() {
			if e != nil && e.Kind == ir.KindString {
				delete(names, e.AsString())
				delete(t.listed, key+" "+e.AsString()) // the source lists it
			}
			list = append(list, e)
		}
	}
	added := make([]string, 0, len(names))
	for n := range names {
		added = append(added, n)
	}
	sort.Strings(added)
	for _, n := range added {
		list = append(list, ir.StringValue(n))
	}
	out.Set(key, ir.ArrayValue(list))
}

// addAliases adds the aliases, in event order, to the root _aliases, after
// the ones the source has. An event the source gives another action is a
// problem; a value of another shape is left to the lint.
func (t *translator) addAliases(out *ir.Object, aliases, from map[string]string) {
	if len(aliases) == 0 {
		return
	}
	res := ir.NewObject()
	if v := out.Get2("_aliases"); v != nil {
		if v.Kind != ir.KindObject {
			return
		}
		for _, ev := range v.AsObject().Keys() {
			res.Set(ev, v.AsObject().Get2(ev))
		}
	}
	evs := make([]string, 0, len(aliases))
	for ev := range aliases {
		evs = append(evs, ev)
	}
	sort.Strings(evs)
	for _, ev := range evs {
		if have := res.Get2(ev); have != nil {
			if have.Kind != ir.KindString || have.AsString() != aliases[ev] {
				t.fail("import.machines.annotate %s gives event %s the action %s, and the source gives it %s; keep one", from[ev], ev, aliases[ev], textOf(have))
			}
			continue
		}
		res.Set(ev, ir.StringValue(aliases[ev]))
	}
	out.Set("_aliases", ir.ObjectValue(res))
}
