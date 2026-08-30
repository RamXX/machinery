package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/ir"
)

// SurfaceLedgerName is the optional legacy surface ledger: the capability
// disposition inventory authored by the opening and closing sweeps when a
// design run has a legacy system behind it.
const SurfaceLedgerName = "legacy/surface.yaml"

var (
	surfaceRootKeys  = stringSet("surface_version", "system", "classes", "as_of", "_comment")
	surfaceClassKeys = stringSet("none", "source", "items")
	surfaceItemKeys  = stringSet("name", "disposition", "via", "target", "rationale")
	// surfaceClasses is the fixed enumeration vocabulary: every class must be
	// inventoried or explicitly waived, so a forgotten class is an error, not
	// a silent pass.
	surfaceClasses = []string{"routes", "commands", "tables", "jobs", "events", "integrations"}
	// the three shapes an as_of anchor may take. surfaceAsOfDateRe pins the
	// date shape before the calendar check; surfaceAsOfRevRe is a git-style
	// object name; surfaceAsOfTagRe is any single revision-like token, wide
	// enough for the forms real VCSs produce (v1.4.2, release/2026-06,
	// legacy@a1b2c3, svn:41207, HEAD@{2}). What none of them admits is prose.
	surfaceAsOfDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	surfaceAsOfRevRe  = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)
	surfaceAsOfTagRe  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+/@:~^{}-]{0,63}$`)
)

// HasSurfaceLedger reports whether a design opted into legacy surface
// checking.
func HasSurfaceLedger(design string) bool {
	fi, err := os.Stat(filepath.Join(design, SurfaceLedgerName))
	return err == nil && !fi.IsDir()
}

// surfaceTargetModel is the slice of the Phase 1 target model that covered
// bindings resolve against: entities and their actions.
type surfaceTargetModel struct {
	actions map[string]map[string]bool
}

type surfaceValidator struct {
	design    string
	g         *Gate
	root      *ir.Object
	model     surfaceTargetModel
	dslEls    map[string]dslEl
	dslExists bool
	covered   int
	dropped   int
	deferred  int
	waived    int
}

// CheckSurface implements Gs-surface. The ledger anchors design coverage to
// the legacy system's mechanically enumerable surface: every route, command,
// table, job, event, and integration is mapped to a target design element or
// carries an explicit dropped/deferred disposition. Gm proves the declared
// legacy model is disposed; Gs proves the observable legacy system is, which
// is the hole a docs-first excavation leaves open.
func CheckSurface(design string) *Gate {
	g := NewGate("Gs-surface  legacy surface ledger")
	g.startOrder()
	path := filepath.Join(design, SurfaceLedgerName)
	if !HasSurfaceLedger(design) {
		g.Errs = append(g.Errs, "no "+SurfaceLedgerName+" in the design; the surface gate was requested but no legacy surface ledger was authored")
		return g
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}
	value, err := ir.LoadYAML(raw)
	if err != nil || value.AsObject() == nil {
		g.Errs = append(g.Errs, SurfaceLedgerName+" is not a yaml mapping")
		return g
	}
	v := &surfaceValidator{design: design, g: g, root: value.AsObject()}
	v.validateRoot()
	if len(g.Errs) != 0 {
		return g
	}
	v.model, err = readSurfaceTargetModel(filepath.Join(design, "domain.modelith.yaml"))
	if err != nil {
		v.errf("domain.modelith.yaml: %v; covered bindings resolve against the Phase 1 target model", err)
		return g
	}
	dslPath := filepath.Join(design, "workspace.dsl")
	if fi, statErr := os.Stat(dslPath); statErr == nil && !fi.IsDir() {
		v.dslExists = true
		v.dslEls = dslElements(dslPath, g)
	}
	v.validateClasses()
	v.g.Count("covered", v.covered)
	v.g.Count("dropped", v.dropped)
	v.g.Count("deferred", v.deferred)
	v.g.Count("waived classes", v.waived)
	v.g.Count("surface items", v.covered+v.dropped+v.deferred)
	if asOf := strings.TrimSpace(v.root.GetString("as_of")); asOf != "" {
		// the ledger's anchor: which legacy revision or date the enumeration
		// held for; surfacing it keeps a stale ledger reviewable (P-F3)
		g.CheckedExtra("as_of " + asOf)
	}
	g.RequireNonzero("surface items", "no legacy surface item was inventoried; a ledger of only waivers has nothing to hold the design to")
	return g
}

func (v *surfaceValidator) errf(format string, args ...any) {
	v.g.Errs = append(v.g.Errs, fmt.Sprintf(format, args...))
}

func (v *surfaceValidator) warnf(format string, args ...any) {
	v.g.Warns = append(v.g.Warns, fmt.Sprintf(format, args...))
}

// checkAsOfShape holds the ledger's anchor to a shape a reviewer can act on.
// This is a WARN, not an ERROR, and deliberately: docs/surface-ledger.md
// documents as_of as "the legacy commit or date the surface was enumerated
// against; non-empty string", which pins the MEANING and leaves the FORMAT
// open, so a design cannot be blocked for a spelling the documentation never
// asked for. What the warning catches is the anchor that quietly stopped being
// one. as_of exists so a stale ledger is visible in every gate run (P-F3), and
// a sentence of prose where a revision belongs is not something a reviewer can
// compare against the legacy system: the narrative belongs in _comment, the
// anchor here.
func (v *surfaceValidator) checkAsOfShape(asOf string) {
	switch {
	case surfaceAsOfDateRe.MatchString(asOf):
		if _, err := time.Parse("2006-01-02", asOf); err != nil {
			v.warnf("as_of %s has the shape of an ISO date but names no real calendar day", ir.Repr(asOf))
		}
	case surfaceAsOfRevRe.MatchString(asOf):
		// a VCS object name
	case surfaceAsOfTagRe.MatchString(asOf):
		// a tag or branch name: one whitespace-free token
	default:
		v.warnf("as_of %s is none of the shapes an anchor takes: an ISO date (YYYY-MM-DD), a VCS revision (7 to 40 hex characters), or a tag-like token; a reviewer compares this value against the legacy system, so keep it an anchor and put the narrative in _comment", ir.Repr(asOf))
	}
}

func (v *surfaceValidator) checkKeys(obj *ir.Object, allowed map[string]bool, where string) {
	for _, key := range obj.Keys() {
		if !allowed[key] {
			v.errf("unsupported key %q in %s (a typo here weakens the ledger)", key, where)
		}
	}
}

func (v *surfaceValidator) validateRoot() {
	v.checkKeys(v.root, surfaceRootKeys, SurfaceLedgerName)
	version := v.root.Get2("surface_version")
	n, err := version.AsNumber().Int64()
	if version == nil || version.Kind != ir.KindNumber || err != nil || n != 1 {
		v.errf("surface_version must be the integer 1")
	}
	if v.root.GetString("system") == "" {
		v.errf("system is required: one line naming the legacy system and its shape")
	}
	if asOf := v.root.Get2("as_of"); asOf != nil {
		if asOf.Kind != ir.KindString || strings.TrimSpace(asOf.AsString()) == "" {
			v.errf("as_of must be a non-empty string: the legacy revision or date the surface was enumerated against")
		} else {
			v.checkAsOfShape(strings.TrimSpace(asOf.AsString()))
		}
	}
	classes := v.root.GetObject("classes")
	if classes.Len() == 0 {
		v.errf("classes is required: inventory or waive all of %s", strings.Join(surfaceClasses, ", "))
		return
	}
	allowed := stringSet(surfaceClasses...)
	for _, key := range classes.Keys() {
		if !allowed[key] {
			v.errf("classes.%s is not a surface class (the vocabulary is %s)", key, strings.Join(surfaceClasses, ", "))
		}
	}
}

func (v *surfaceValidator) validateClasses() {
	classes := v.root.GetObject("classes")
	for _, kind := range surfaceClasses {
		node := classes.Get2(kind)
		if node == nil {
			v.errf("classes.%s is missing; inventory it or waive it with none: <reason>", kind)
			continue
		}
		obj := node.AsObject()
		if obj == nil {
			v.errf("classes.%s must be a mapping", kind)
			continue
		}
		v.checkKeys(obj, surfaceClassKeys, "classes."+kind)
		if none := obj.Get2("none"); none != nil {
			if obj.GetString("none") == "" {
				v.errf("classes.%s.none needs a reason; an unexplained waiver is an unanswered enumeration question", kind)
				continue
			}
			if obj.GetString("source") != "" || obj.Get2("items") != nil {
				v.errf("classes.%s mixes a waiver with an inventory; pick one", kind)
				continue
			}
			v.waived++
			continue
		}
		if obj.GetString("source") == "" {
			v.errf("classes.%s.source is required: name where the enumeration came from", kind)
		}
		items := migrationList(obj.Get2("items"))
		if len(items) == 0 {
			v.errf("classes.%s.items is empty; enumerate the surface or waive the class with none: <reason>", kind)
			continue
		}
		seen := map[string]bool{}
		for i, item := range items {
			v.validateItem(kind, i, item, seen)
		}
	}
}

func (v *surfaceValidator) validateItem(kind string, index int, item *ir.Value, seen map[string]bool) {
	where := fmt.Sprintf("classes.%s.items[%d]", kind, index)
	obj := item.AsObject()
	if obj == nil {
		v.errf("%s is not a mapping", where)
		return
	}
	v.checkKeys(obj, surfaceItemKeys, where)
	name := obj.GetString("name")
	if name == "" {
		v.errf("%s.name is required", where)
		return
	}
	if seen[name] {
		v.errf("classes.%s lists %q twice", kind, name)
		return
	}
	seen[name] = true
	disposition := obj.GetString("disposition")
	via, target, rationale := obj.GetString("via"), obj.GetString("target"), obj.GetString("rationale")
	switch disposition {
	case "covered":
		if via == "" || target == "" {
			v.errf("%s (%s) is covered but names no via/target design element", where, name)
			return
		}
		if !v.resolveBinding(where, name, via, target) {
			return
		}
		v.covered++
	case "dropped", "deferred":
		if rationale == "" {
			v.errf("%s (%s) is %s without a rationale; an unexplained gap is not a disposition", where, name, disposition)
			return
		}
		if via != "" || target != "" {
			v.errf("%s (%s) is %s but names a design element; a capability with a target is covered", where, name, disposition)
			return
		}
		if disposition == "dropped" {
			v.dropped++
		} else {
			v.deferred++
		}
	default:
		v.errf("%s.disposition must be covered, dropped, or deferred", where)
		return
	}
	v.g.Count(kind)
}

// resolveBinding checks a covered row's target against the design artifact
// its via names. Bindings resolve against the TARGET design, never the legacy
// model: the ledger's question is whether the new design accounts for the
// capability, and the legacy model is itself under suspicion of being
// incomplete.
func (v *surfaceValidator) resolveBinding(where, name, via, target string) bool {
	switch via {
	case "entity":
		if _, ok := v.model.actions[target]; !ok {
			v.errf("%s (%s) binds to unknown target entity %q", where, name, target)
			return false
		}
	case "action":
		parts := strings.Split(target, ".")
		if len(parts) != 2 {
			v.errf("%s (%s) via action needs an Entity.action target, got %q", where, name, target)
			return false
		}
		actions, ok := v.model.actions[parts[0]]
		if !ok {
			v.errf("%s (%s) binds to unknown target entity %q", where, name, parts[0])
			return false
		}
		if !actions[parts[1]] {
			v.errf("%s (%s) binds to unknown action %q on entity %s", where, name, parts[1], parts[0])
			return false
		}
	case "component":
		if !v.dslExists {
			v.errf("%s (%s) binds via component but workspace.dsl does not exist yet; bind via entity/action until Phase 2 lands", where, name)
			return false
		}
		if _, ok := v.dslEls[target]; !ok {
			v.errf("%s (%s) binds to %q, which is not a workspace.dsl element", where, name, target)
			return false
		}
	case "machine":
		if strings.ContainsAny(target, "/\\") || strings.Contains(target, "..") {
			v.errf("%s (%s) via machine needs a bare machine name, got %q", where, name, target)
			return false
		}
		machinePath := filepath.Join(v.design, "machines", target+".machine.json")
		if fi, err := os.Stat(machinePath); err != nil || fi.IsDir() {
			v.errf("%s (%s) binds to machines/%s.machine.json, which does not exist yet", where, name, target)
			return false
		}
	default:
		v.errf("%s (%s) via must be entity, action, component, or machine", where, name)
		return false
	}
	return true
}

func readSurfaceTargetModel(path string) (surfaceTargetModel, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return surfaceTargetModel{}, err
	}
	value, err := ir.LoadYAML(raw)
	if err != nil || value.AsObject() == nil {
		return surfaceTargetModel{}, fmt.Errorf("%s is not a yaml mapping", filepath.Base(path))
	}
	root := value.AsObject()
	if root.GetString("kind") != "DomainModel" || root.GetString("version") != "v1" {
		return surfaceTargetModel{}, fmt.Errorf("%s must be a Modelith DomainModel v1", filepath.Base(path))
	}
	entities := root.GetObject("entities")
	if entities.Len() == 0 {
		return surfaceTargetModel{}, fmt.Errorf("%s declares no entities", filepath.Base(path))
	}
	model := surfaceTargetModel{actions: map[string]map[string]bool{}}
	for _, name := range entities.Keys() {
		actions := map[string]bool{}
		for _, item := range migrationList(entities.GetObject(name).Get2("actions")) {
			if obj := item.AsObject(); obj != nil && obj.GetString("name") != "" {
				actions[obj.GetString("name")] = true
			}
		}
		model.actions[name] = actions
	}
	return model, nil
}
