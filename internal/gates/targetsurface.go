package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// TargetSurfacesName is the target-side surface ledger: the mapping from every
// human act the domain model declares to the named interface that carries it.
// Gs anchors the design to the LEGACY system's observable surface; this is its
// forward twin, and it exists because a human act with no named interface has
// no artifact, so no gate and no closed review list. Eight admin-gated acts
// once survived ten deep reviews that way.
const TargetSurfacesName = "surfaces.yaml"

// targetSurfaceSystemActor is the one actor value that carries no surface
// obligation: an act the system performs on its own needs no screen, command,
// route, or knob for a person to reach it.
const targetSurfaceSystemActor = "System"

var (
	targetSurfacesRootKeys  = stringSet("surface_version", "sources", "acts", "deferrals", "_comment")
	targetSurfacesActKeys   = stringSet("act", "actor", "surface", "milestone", "_comment")
	targetSurfacesDeferKeys = stringSet("act", "reason", "_comment")
)

// HasTargetSurfaces reports whether a design authored a target surface ledger.
func HasTargetSurfaces(design string) bool {
	fi, err := os.Stat(filepath.Join(design, TargetSurfacesName))
	return err == nil && !fi.IsDir()
}

// HasHumanActions closes Gu's activation universe: the ledger is owed by a
// human act in the model, so deleting surfaces.yaml must activate a failure
// rather than delete the gate.
func HasHumanActions(design string) bool {
	model, err := readTargetActModel(design)
	return err == nil && len(model.obligated()) > 0
}

// targetAct is one action of the Phase 1 target model, with the actor that
// performs it ("" when the action declares none).
type targetAct struct {
	entity string
	action string
	actor  string
}

// key renders the act the way the ledger names it.
func (a targetAct) key() string { return a.entity + "." + a.action }

// targetActModel is the slice of the target model the ledger resolves against:
// every entity action in declaration order, plus the actor vocabulary.
type targetActModel struct {
	acts     []targetAct
	index    map[string]targetAct
	entities map[string]bool
	actors   map[string]bool
}

// obligated returns the acts a person performs, in model declaration order: an
// actor is declared and it is not the System. Those are the acts that owe a
// named surface.
func (m targetActModel) obligated() []targetAct {
	var out []targetAct
	for _, a := range m.acts {
		if a.actor != "" && a.actor != targetSurfaceSystemActor {
			out = append(out, a)
		}
	}
	return out
}

// actorless counts the actions that declare no actor at all. They carry no
// obligation, but the count is printed so partial actor adoption stays visible:
// a model where nobody filled the field in yet must not read as full coverage.
func (m targetActModel) actorless() int {
	n := 0
	for _, a := range m.acts {
		if a.actor == "" {
			n++
		}
	}
	return n
}

// planMilestoneIndex is the set of spellings a surfaces.yaml row may use to
// name a milestone the build plan declares, plus the declared milestones as a
// reader would list them. declared is false when the design has no BUILD.md,
// or its plan declares no milestone: the ledger is authored in Phase 2 and the
// plan in Phase 4, so a ledger that legitimately precedes the plan must not be
// failed for naming a milestone nothing has written down yet.
type planMilestoneIndex struct {
	declared bool
	tokens   map[string]bool // accepted spellings, lower-cased
	labels   []string        // "M<n> - <title>", in plan order, for the finding
}

// resolves reports whether name is one of the declared milestones.
func (p planMilestoneIndex) resolves(name string) bool {
	return p.tokens[strings.ToLower(strings.TrimSpace(name))]
}

type targetSurfaceValidator struct {
	g     *Gate
	root  *ir.Object
	model targetActModel
	plan  planMilestoneIndex
	// seen records every act value the ledger names, anywhere, so a value
	// written twice is an error even when the two rows disagree about what to
	// do with it.
	seen             map[string]bool
	covered          map[string]bool
	deferredActs     map[string]bool
	deferredPersonas map[string]bool
	knobRows         int
	deferralRows     int
	personaRows      int
	milestoneRefs    int
}

// CheckTargetSurfaces implements Gu-surfaces. The obligated set is closed and
// mechanically derived: every action in the target model whose actor is a
// person. Each one is either mapped to a named surface (a screen, an admin
// command, an API route, a config release) or explicitly deferred with a
// reason. An act that is neither is an ERROR naming it, which is the finding
// that has no other home: G2 holds boundaries, Gx holds placement, and neither
// asks how a human reaches an act.
func CheckTargetSurfaces(design string) *Gate {
	g := NewGate("Gu-surfaces  target surface ledger")
	g.startOrder()
	path := filepath.Join(design, TargetSurfacesName)
	if !HasTargetSurfaces(design) {
		g.Errs = append(g.Errs, "no "+TargetSurfacesName+" in the design; the target surface gate was requested but no target surface ledger was authored")
		return g
	}
	raw, err := readDesignFile(design, path)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}
	value, err := ir.LoadYAML(raw)
	if err != nil || value.AsObject() == nil {
		g.Errs = append(g.Errs, TargetSurfacesName+" is not a yaml mapping")
		return g
	}
	v := &targetSurfaceValidator{
		g:                g,
		root:             value.AsObject(),
		seen:             map[string]bool{},
		covered:          map[string]bool{},
		deferredActs:     map[string]bool{},
		deferredPersonas: map[string]bool{},
	}
	v.validateRoot()
	if len(g.Errs) != 0 {
		return g
	}
	v.model, err = readTargetActModel(design)
	if err != nil {
		v.errf("%v; the obligated act set is every target-model action performed by a person", err)
		return g
	}
	v.plan = readPlanMilestones(design, g)
	v.validateActs()
	v.validateDeferrals()
	obligated := v.model.obligated()
	v.checkCompleteness(obligated)
	covered := 0
	for _, a := range obligated {
		if v.covered[a.key()] {
			covered++
		}
	}
	// every segment is emitted verbatim, zeros included: a ledger that covers
	// nothing, or a model where no action names an actor, must be visible in
	// the run rather than vanishing with the zero-count suppression
	g.CheckedExtra(fmt.Sprintf("%d obligated actions", len(obligated)))
	g.CheckedExtra(fmt.Sprintf("%d covered", covered))
	g.CheckedExtra(fmt.Sprintf("%d deferred acts", v.deferralRows))
	g.CheckedExtra(fmt.Sprintf("%d deferred personas", v.personaRows))
	g.CheckedExtra(fmt.Sprintf("%d knob rows", v.knobRows))
	g.CheckedExtra(fmt.Sprintf("%d actorless actions", v.model.actorless()))
	if v.plan.declared {
		g.CheckedExtra(fmt.Sprintf("%d milestone references resolved", v.milestoneRefs))
	}
	if len(obligated) == 0 {
		g.Notes = append(g.Notes, "no target-model action names a human actor; the ledger holds nothing to a surface (add actor: to the model's actions to arm this gate)")
	}
	return g
}

func (v *targetSurfaceValidator) errf(format string, args ...any) {
	v.g.Errs = append(v.g.Errs, fmt.Sprintf(format, args...))
}

func (v *targetSurfaceValidator) checkKeys(obj *ir.Object, allowed map[string]bool, where string) {
	for _, key := range obj.Keys() {
		if !allowed[key] {
			v.errf("unsupported key %q in %s (a typo here weakens the ledger)", key, where)
		}
	}
}

// record registers an act value and reports whether it is the first time the
// ledger names it. The set spans acts and deferrals together: an act both
// mapped and deferred is a contradiction, not two independent statements.
func (v *targetSurfaceValidator) record(act, where string) bool {
	if v.seen[act] {
		v.errf("%s names act %q, which the ledger already names; every act is stated exactly once", where, act)
		return false
	}
	v.seen[act] = true
	return true
}

func (v *targetSurfaceValidator) validateRoot() {
	v.checkKeys(v.root, targetSurfacesRootKeys, TargetSurfacesName)
	version := v.root.Get2("surface_version")
	n, err := version.AsNumber().Int64()
	if version == nil || version.Kind != ir.KindNumber || err != nil || n != 1 {
		v.errf("surface_version must be the integer 1")
	}
	// sources is required, the same rule Gs holds the legacy ledger's classes
	// to: an enumeration with no named source is a completeness claim with no
	// evidence, and the persona walk is exactly a completeness claim.
	sources := v.root.Get2("sources")
	switch {
	case sources == nil:
		v.errf("sources is required: a list naming where the act list was enumerated from (the model's per-persona action walk, plus any route or command sweep)")
	case sources.Kind != ir.KindArray || len(sources.AsArray()) == 0:
		v.errf("sources must be a non-empty list of strings naming where the act list was enumerated from")
	default:
		for i, item := range sources.AsArray() {
			if item == nil || item.Kind != ir.KindString || strings.TrimSpace(item.AsString()) == "" {
				v.errf("sources[%d] must be a non-empty string naming where the act list was enumerated from", i)
			}
		}
	}
	acts := v.root.Get2("acts")
	if acts == nil {
		v.errf("acts is required: the list of acts and the surfaces that carry them")
		return
	}
	if acts.Kind != ir.KindArray {
		v.errf("acts must be a list")
	}
	if deferrals := v.root.Get2("deferrals"); deferrals != nil && deferrals.Kind != ir.KindArray {
		v.errf("deferrals must be a list")
	}
}

func (v *targetSurfaceValidator) validateActs() {
	for i, item := range migrationList(v.root.Get2("acts")) {
		where := fmt.Sprintf("acts[%d]", i)
		obj := item.AsObject()
		if obj == nil {
			v.errf("%s is not a mapping", where)
			continue
		}
		v.checkKeys(obj, targetSurfacesActKeys, where)
		act := strings.TrimSpace(obj.GetString("act"))
		actor := strings.TrimSpace(obj.GetString("actor"))
		surface := strings.TrimSpace(obj.GetString("surface"))
		if act == "" {
			v.errf("%s.act is required: an Entity.action, or knob:<key> for a configuration knob", where)
			continue
		}
		if actor == "" {
			v.errf("%s (%s) names no actor; the ledger states who performs the act", where, act)
		}
		if surface == "" {
			v.errf("%s (%s) names no surface; an act with no named interface is exactly the gap this ledger exists to catch", where, act)
		}
		if milestone := obj.Get2("milestone"); milestone != nil {
			v.checkMilestone(where, act, strings.TrimSpace(milestone.AsString()))
		}
		if !v.record(act, where) {
			continue
		}
		if strings.HasPrefix(act, "knob:") {
			if strings.TrimSpace(strings.TrimPrefix(act, "knob:")) == "" {
				v.errf("%s names a knob with no key; write knob:<key>", where)
				continue
			}
			// knob rows resolve against nothing: configuration is an open set
			// by design, so the gate holds their shape and uniqueness only
			v.knobRows++
			continue
		}
		v.resolveAct(where, act, actor)
	}
}

// checkMilestone holds an acts row's milestone to the build plan. A milestone
// name that resolves to nothing is exactly as useless as no milestone at all,
// and worse than none, because it reads like a commitment: "M2" surviving a
// replan that renumbered the plan tells a reader the surface lands in a
// milestone that no longer exists. The check arms itself only once the plan
// declares milestones; before Phase 4 the ledger legitimately runs ahead.
func (v *targetSurfaceValidator) checkMilestone(where, act, name string) {
	switch {
	case name == "":
		v.errf("%s (%s) has an empty milestone; drop the key or name the milestone", where, act)
	case !v.plan.declared:
		// no BUILD.md, or a plan that declares no milestone: nothing to resolve against
	case !v.plan.resolves(name):
		v.errf("%s (%s) names milestone %s, which the build plan does not declare; the declared milestones are %s", where, act, ir.Repr(name), strings.Join(v.plan.labels, "; "))
	default:
		v.milestoneRefs++
	}
}

// readPlanMilestones indexes the milestones of every plan-bearing document.
// It reads them through planDocuments, the parse Gb and Ga already share,
// rather than scanning BUILD.md a third way: a ledger resolving against a
// different reading of the plan than the plan gate uses could bind to a
// milestone Gb does not hold. A milestone answers to its number written any
// documented way (M2, m2, 2, and M02 as authored) or to its own title.
func readPlanMilestones(design string, g *Gate) planMilestoneIndex {
	idx := planMilestoneIndex{tokens: map[string]bool{}}
	if !HasBuildDoc(design) {
		return idx
	}
	seen := map[string]bool{}
	for _, doc := range planDocuments(design, g) {
		for _, m := range doc.milestones {
			label := "M" + m.numRaw + " - " + m.title
			if seen[label] {
				continue
			}
			seen[label] = true
			idx.labels = append(idx.labels, label)
			for _, tok := range []string{"m" + m.numRaw, m.numRaw, m.title} {
				if tok = strings.ToLower(strings.TrimSpace(tok)); tok != "" {
					idx.tokens[tok] = true
				}
			}
			if m.numOK {
				idx.tokens["m"+strconv.Itoa(m.num)] = true
				idx.tokens[strconv.Itoa(m.num)] = true
			}
		}
	}
	idx.declared = len(idx.labels) > 0
	return idx
}

// resolveAct holds an Entity.action row to the target model: the act must
// exist, and the actor the row names must be the actor the model declares. A
// ledger that misattributes an act is worse than no ledger: it closes the
// review question with the wrong answer.
func (v *targetSurfaceValidator) resolveAct(where, act, actor string) {
	parts := strings.Split(act, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		v.errf("%s act %q is neither an Entity.action nor a knob:<key>", where, act)
		return
	}
	modelAct, ok := v.model.index[act]
	if !ok {
		if !v.model.entities[parts[0]] {
			v.errf("%s names unknown entity %q; the ledger resolves against the target model", where, parts[0])
		} else {
			v.errf("%s names unknown action %q on entity %s; the ledger resolves against the target model", where, parts[1], parts[0])
		}
		return
	}
	if actor == "" {
		// the missing-actor error is already recorded; do not cascade
		return
	}
	if actor != modelAct.actor {
		if modelAct.actor == "" {
			v.errf("%s (%s) names actor %q but the target model declares no actor for that action; the ledger must not misattribute an act", where, act, actor)
		} else {
			v.errf("%s (%s) names actor %q but the target model declares %q; the ledger must not misattribute an act", where, act, actor, modelAct.actor)
		}
		return
	}
	v.covered[act] = true
}

func (v *targetSurfaceValidator) validateDeferrals() {
	for i, item := range migrationList(v.root.Get2("deferrals")) {
		where := fmt.Sprintf("deferrals[%d]", i)
		obj := item.AsObject()
		if obj == nil {
			v.errf("%s is not a mapping", where)
			continue
		}
		v.checkKeys(obj, targetSurfacesDeferKeys, where)
		act := strings.TrimSpace(obj.GetString("act"))
		reason := strings.TrimSpace(obj.GetString("reason"))
		if act == "" {
			v.errf("%s.act is required: an Entity.action, knob:<key>, or actor:<Name> to defer one persona wholesale", where)
			continue
		}
		if reason == "" {
			v.errf("%s (%s) is deferred without a reason; an unexplained gap is not a deferral", where, act)
		}
		if !v.record(act, where) {
			continue
		}
		if name, ok := strings.CutPrefix(act, "actor:"); ok {
			name = strings.TrimSpace(name)
			if name == "" {
				v.errf("%s defers a persona with no name; write actor:<Name>", where)
				continue
			}
			if !v.model.actors[name] {
				v.errf("%s defers actor %q, which no action in the target model declares", where, name)
				continue
			}
			v.deferredPersonas[name] = true
			v.personaRows++
			continue
		}
		if !strings.HasPrefix(act, "knob:") && strings.Count(act, ".") != 1 {
			v.errf("%s act %q is not an Entity.action, a knob:<key>, or an actor:<Name>", where, act)
			continue
		}
		v.deferredActs[act] = true
		v.deferralRows++
	}
}

// checkCompleteness is the gate's reason to exist: the obligated set is closed,
// so every member is either mapped or explicitly deferred, and anything else is
// named out loud.
func (v *targetSurfaceValidator) checkCompleteness(obligated []targetAct) {
	for _, a := range obligated {
		key := a.key()
		if v.covered[key] || v.deferredActs[key] || v.deferredPersonas[a.actor] {
			continue
		}
		v.errf("%s (actor %s) is named by no acts row and no deferral; every act a person performs needs a named surface or an explicit deferral", key, a.actor)
	}
}

// readTargetActModel loads every *.modelith.yaml at the design root, the same
// way the Gs target model is loaded, and additionally reads each action's
// actor. A design root carries the target model only: the legacy model lives
// under legacy/ and is not swept.
func readTargetActModel(design string) (targetActModel, error) {
	paths := sortedGlobExt(design, ".modelith.yaml")
	if len(paths) == 0 {
		return targetActModel{}, fmt.Errorf("no *.modelith.yaml in the design directory")
	}
	model := targetActModel{
		index:    map[string]targetAct{},
		entities: map[string]bool{},
		actors:   map[string]bool{},
	}
	for _, path := range paths {
		if err := model.readFile(design, path); err != nil {
			return targetActModel{}, err
		}
	}
	return model, nil
}

func (m *targetActModel) readFile(design, path string) error {
	base := filepath.Base(path)
	raw, err := readDesignFile(design, path)
	if err != nil {
		return err
	}
	value, err := ir.LoadYAML(raw)
	if err != nil || value.AsObject() == nil {
		return fmt.Errorf("%s is not a yaml mapping", base)
	}
	root := value.AsObject()
	if root.GetString("kind") != "DomainModel" || root.GetString("version") != "v1" {
		return fmt.Errorf("%s must be a Modelith DomainModel v1", base)
	}
	entities := root.GetObject("entities")
	if entities.Len() == 0 {
		return fmt.Errorf("%s declares no entities", base)
	}
	for _, name := range entities.Keys() {
		m.entities[name] = true
		for _, item := range migrationList(entities.GetObject(name).Get2("actions")) {
			obj := item.AsObject()
			if obj == nil || obj.GetString("name") == "" {
				continue
			}
			act := targetAct{
				entity: name,
				action: obj.GetString("name"),
				actor:  strings.TrimSpace(obj.GetString("actor")),
			}
			if _, dup := m.index[act.key()]; dup {
				continue
			}
			m.acts = append(m.acts, act)
			m.index[act.key()] = act
			if act.actor != "" {
				m.actors[act.actor] = true
			}
		}
	}
	// entity map order is the model's own; the act list is sorted so findings
	// and the checked line never vary between runs
	sort.SliceStable(m.acts, func(i, j int) bool { return m.acts[i].key() < m.acts[j].key() })
	return nil
}
