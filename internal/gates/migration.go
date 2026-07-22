package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// MigrationContractName is the optional hybrid/rebuild transition contract.
const MigrationContractName = "migration.yaml"

var (
	migrationRootKeys = stringSet(
		"contract_version", "mode", "legacy", "target", "dispositions",
		"new_entities", "assets", "data_mappings", "state_mappings", "phases",
		"cutover", "risks", "_comment",
	)
	migrationModelKeys       = stringSet("model")
	migrationDispositionKeys = stringSet("legacy", "target", "strategy", "rationale")
	migrationAssetKeys       = stringSet("name", "kind", "strategy", "target", "rationale", "verification")
	migrationDataKeys        = stringSet("source", "target", "transform", "validation", "rollback")
	migrationStateKeys       = stringSet("source", "target", "reason")
	migrationPhaseKeys       = stringSet(
		"id", "source_of_truth", "read_path", "write_path", "backfill",
		"entry_criteria", "exit_criteria", "rollback", "observability",
		"idempotency", "conflict_resolution", "reconciliation", "parity",
	)
	migrationCutoverKeys = stringSet("phase", "rollback_phase", "decision_criteria", "rollback_window", "max_data_loss")
	migrationRiskKeys    = stringSet("dependency", "detection", "mitigation", "residual", "owner")
	migrationAssetKinds  = stringSet("module", "service", "schema", "data", "test")
)

func stringSet(values ...string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

// HasMigrationContract reports whether a design opted into hybrid/rebuild
// transition checking.
func HasMigrationContract(design string) bool {
	fi, err := os.Stat(filepath.Join(design, MigrationContractName))
	return err == nil && !fi.IsDir()
}

type migrationEntity struct {
	attributes      map[string]string
	lifecycleAttr   string
	lifecycleValues []string
}

type migrationModel struct {
	entities map[string]migrationEntity
	order    []string
	enums    map[string][]string
}

type migrationDisposition struct {
	legacy   string
	target   string
	strategy string
}

type migrationPhase struct {
	id            string
	sourceOfTruth string
	readPath      string
	writePath     string
}

type migrationValidator struct {
	design       string
	g            *Gate
	root         *ir.Object
	legacy       migrationModel
	target       migrationModel
	dispositions map[string]migrationDisposition
	targetMapped map[string]bool
	newEntities  map[string]bool
	phases       map[string]migrationPhase
	phaseOrder   map[string]int
}

// CheckMigration implements Gm-transition. It reconciles two domain truths
// (legacy and target) plus the transition between them. The contract is a
// coverage gate: every legacy entity is disposed, every target entity is
// mapped or new, replace mappings cover data and lifecycle states, and the
// coexistence phases name their source of truth, rollback, observability, and
// transitional failure posture.
func CheckMigration(design string) *Gate {
	g := NewGate("Gm-transition  hybrid/rebuild migration contract")
	g.startOrder()
	path := filepath.Join(design, MigrationContractName)
	if !HasMigrationContract(design) {
		g.Errs = append(g.Errs, "no "+MigrationContractName+" in the design; the transition gate was requested but no hybrid/rebuild contract was authored")
		return g
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}
	value, err := ir.LoadYAML(raw)
	if err != nil || value.AsObject() == nil {
		g.Errs = append(g.Errs, MigrationContractName+" is not a yaml mapping")
		return g
	}
	v := &migrationValidator{
		design:       design,
		g:            g,
		root:         value.AsObject(),
		dispositions: map[string]migrationDisposition{},
		targetMapped: map[string]bool{},
		newEntities:  map[string]bool{},
		phases:       map[string]migrationPhase{},
		phaseOrder:   map[string]int{},
	}
	v.validateRoot()
	if len(g.Errs) != 0 {
		return g
	}
	v.loadModels()
	if len(g.Errs) != 0 {
		return g
	}
	v.validateDispositions()
	v.validateAssets()
	v.validateDataMappings()
	v.validateStateMappings()
	v.validatePhases()
	v.validateCutover()
	v.validateRisks()
	v.validateNarrativeBridges()
	g.RequireNonzero("dispositions", "no legacy entity disposition was checked")
	g.RequireNonzero("salvage decisions", "no reusable/replaceable implementation asset was inventoried")
	g.RequireNonzero("transition phases", "no coexistence/cutover phases were checked")
	g.RequireNonzero("transition risks", "no transitional dependency failure posture was checked")
	return g
}

func (v *migrationValidator) validateAssets() {
	seen := map[string]bool{}
	for i, item := range migrationList(v.root.Get2("assets")) {
		where := fmt.Sprintf("assets[%d]", i)
		obj := item.AsObject()
		if obj == nil {
			v.errf("%s is not a mapping", where)
			continue
		}
		v.checkKeys(obj, migrationAssetKeys, where)
		name, kind, strategy := obj.GetString("name"), obj.GetString("kind"), obj.GetString("strategy")
		if name == "" || obj.GetString("rationale") == "" || obj.GetString("verification") == "" {
			v.errf("%s needs name, rationale, and verification", where)
			continue
		}
		if seen[name] {
			v.errf("asset %q is listed twice", name)
			continue
		}
		seen[name] = true
		if !migrationAssetKinds[kind] {
			v.errf("%s.kind must be module, service, schema, data, or test", where)
			continue
		}
		switch strategy {
		case "reuse", "wrap", "replace":
			if obj.GetString("target") == "" {
				v.errf("%s strategy %s requires target", where, strategy)
				continue
			}
		case "retire":
			if obj.GetString("target") != "" {
				v.errf("%s strategy retire must not name a target", where)
				continue
			}
		default:
			v.errf("%s.strategy must be reuse, wrap, replace, or retire", where)
			continue
		}
		v.g.Count("salvage decisions")
	}
}

func (v *migrationValidator) errf(format string, args ...any) {
	v.g.Errs = append(v.g.Errs, fmt.Sprintf(format, args...))
}

func (v *migrationValidator) checkKeys(obj *ir.Object, allowed map[string]bool, where string) {
	for _, key := range obj.Keys() {
		if !allowed[key] {
			v.errf("unsupported key %q in %s (a typo here weakens the transition contract)", key, where)
		}
	}
}

func (v *migrationValidator) validateRoot() {
	v.checkKeys(v.root, migrationRootKeys, MigrationContractName)
	version := v.root.Get2("contract_version")
	n, err := version.AsNumber().Int64()
	if version == nil || version.Kind != ir.KindNumber || err != nil || n != 1 {
		v.errf("contract_version must be the integer 1")
	}
	mode := v.root.GetString("mode")
	if mode != "hybrid" && mode != "rebuild" {
		v.errf("mode must be 'hybrid' or 'rebuild'")
	}
	for _, key := range []string{"legacy", "target"} {
		obj := v.root.GetObject(key)
		if obj.Len() == 0 {
			v.errf("%s.model is required", key)
			continue
		}
		v.checkKeys(obj, migrationModelKeys, key)
		if obj.GetString("model") == "" {
			v.errf("%s.model is required", key)
		}
	}
}

func (v *migrationValidator) loadModels() {
	legacyPath, err := migrationDesignPath(v.design, v.root.GetObject("legacy").GetString("model"))
	if err != nil {
		v.errf("legacy.model: %v", err)
		return
	}
	targetPath, err := migrationDesignPath(v.design, v.root.GetObject("target").GetString("model"))
	if err != nil {
		v.errf("target.model: %v", err)
		return
	}
	if legacyPath == targetPath {
		v.errf("legacy.model and target.model must be different files; rebuild/hybrid mode keeps current and intended truth separate")
		return
	}
	v.legacy, err = readMigrationModel(legacyPath)
	if err != nil {
		v.errf("legacy.model: %v", err)
		return
	}
	v.target, err = readMigrationModel(targetPath)
	if err != nil {
		v.errf("target.model: %v", err)
		return
	}
	v.g.Count("legacy entities", len(v.legacy.order))
	v.g.Count("target entities", len(v.target.order))
}

func migrationDesignPath(design, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("path must be relative to the design directory")
	}
	joined := filepath.Clean(filepath.Join(design, name))
	rel, err := filepath.Rel(design, joined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes the design directory")
	}
	return joined, nil
}

func readMigrationModel(path string) (migrationModel, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return migrationModel{}, err
	}
	value, err := ir.LoadYAML(raw)
	if err != nil || value.AsObject() == nil {
		return migrationModel{}, fmt.Errorf("%s is not a yaml mapping", filepath.Base(path))
	}
	root := value.AsObject()
	if root.GetString("kind") != "DomainModel" || root.GetString("version") != "v1" {
		return migrationModel{}, fmt.Errorf("%s must be a Modelith DomainModel v1", filepath.Base(path))
	}
	model := migrationModel{entities: map[string]migrationEntity{}, enums: map[string][]string{}}
	for _, enumName := range root.GetObject("enums").Keys() {
		var values []string
		for _, item := range migrationList(root.GetObject("enums").GetObject(enumName).Get2("values")) {
			if item.AsObject() != nil && item.AsObject().GetString("name") != "" {
				values = append(values, item.AsObject().GetString("name"))
			}
		}
		model.enums[enumName] = values
	}
	entities := root.GetObject("entities")
	if entities.Len() == 0 {
		return migrationModel{}, fmt.Errorf("%s declares no entities", filepath.Base(path))
	}
	for _, name := range entities.Keys() {
		entity := migrationEntity{attributes: map[string]string{}}
		for _, item := range migrationList(entities.GetObject(name).Get2("attributes")) {
			obj := item.AsObject()
			if obj == nil {
				continue
			}
			attr, typ := obj.GetString("name"), obj.GetString("type")
			if attr == "" {
				continue
			}
			entity.attributes[attr] = typ
			if (attr == "status" || attr == "stage" || attr == "state") && len(model.enums[typ]) > 0 {
				entity.lifecycleAttr = attr
				entity.lifecycleValues = append([]string{}, model.enums[typ]...)
			}
		}
		model.order = append(model.order, name)
		model.entities[name] = entity
	}
	return model, nil
}

func migrationList(value *ir.Value) []*ir.Value {
	if value == nil || value.Kind != ir.KindArray {
		return nil
	}
	return value.AsArray()
}

func migrationStrings(value *ir.Value) ([]string, bool) {
	if value == nil || value.Kind != ir.KindArray {
		return nil, false
	}
	var out []string
	for _, item := range value.AsArray() {
		if item == nil || item.Kind != ir.KindString || item.AsString() == "" {
			return nil, false
		}
		out = append(out, item.AsString())
	}
	return out, true
}

func (v *migrationValidator) validateDispositions() {
	for i, item := range migrationList(v.root.Get2("dispositions")) {
		where := fmt.Sprintf("dispositions[%d]", i)
		obj := item.AsObject()
		if obj == nil {
			v.errf("%s is not a mapping", where)
			continue
		}
		v.checkKeys(obj, migrationDispositionKeys, where)
		d := migrationDisposition{legacy: obj.GetString("legacy"), target: obj.GetString("target"), strategy: obj.GetString("strategy")}
		if d.legacy == "" || obj.GetString("rationale") == "" {
			v.errf("%s needs legacy and rationale", where)
			continue
		}
		if _, ok := v.legacy.entities[d.legacy]; !ok {
			v.errf("%s names unknown legacy entity %q", where, d.legacy)
			continue
		}
		if _, dup := v.dispositions[d.legacy]; dup {
			v.errf("legacy entity %q has more than one disposition", d.legacy)
			continue
		}
		switch d.strategy {
		case "reuse", "wrap", "replace":
			if d.target == "" {
				v.errf("%s strategy %s requires target", where, d.strategy)
				continue
			}
			if _, ok := v.target.entities[d.target]; !ok {
				v.errf("%s names unknown target entity %q", where, d.target)
				continue
			}
			v.targetMapped[d.target] = true
		case "retire":
			if d.target != "" {
				v.errf("%s strategy retire must not name a target", where)
				continue
			}
		default:
			v.errf("%s strategy must be reuse, wrap, replace, or retire", where)
			continue
		}
		v.dispositions[d.legacy] = d
		v.g.Count("dispositions")
	}
	for _, name := range v.legacy.order {
		if _, ok := v.dispositions[name]; !ok {
			v.errf("legacy entity %q has no disposition", name)
		}
	}
	newEntities, ok := migrationStrings(v.root.Get2("new_entities"))
	if !ok {
		v.errf("new_entities must be a list of target entity names (use [] when none)")
	} else {
		for _, name := range newEntities {
			if _, exists := v.target.entities[name]; !exists {
				v.errf("new_entities names unknown target entity %q", name)
				continue
			}
			if v.targetMapped[name] {
				v.errf("target entity %q is both mapped and declared new", name)
				continue
			}
			if v.newEntities[name] {
				v.errf("new_entities lists %q twice", name)
				continue
			}
			v.newEntities[name] = true
			v.g.Count("new target entities")
		}
	}
	for _, name := range v.target.order {
		if !v.targetMapped[name] && !v.newEntities[name] {
			v.errf("target entity %q is neither mapped from legacy nor declared new", name)
		}
	}
}

func (v *migrationValidator) validateDataMappings() {
	sourceCovered, targetCovered := map[string]bool{}, map[string]bool{}
	for i, item := range migrationList(v.root.Get2("data_mappings")) {
		where := fmt.Sprintf("data_mappings[%d]", i)
		obj := item.AsObject()
		if obj == nil {
			v.errf("%s is not a mapping", where)
			continue
		}
		v.checkKeys(obj, migrationDataKeys, where)
		source, target := obj.GetString("source"), obj.GetString("target")
		if source == "" || target == "" || (source == "-" && target == "-") {
			v.errf("%s needs source and target; exactly one may be '-' for derive/drop", where)
			continue
		}
		if obj.GetString("transform") == "" || obj.GetString("validation") == "" || obj.GetString("rollback") == "" {
			v.errf("%s needs transform, validation, and rollback", where)
		}
		var sourceEntity, targetEntity string
		if source != "-" {
			entity, attr, ok := migrationAttributeRef(v.legacy, source)
			if !ok {
				v.errf("%s source %q is not a legacy Entity.attribute", where, source)
				continue
			}
			sourceEntity = entity
			sourceCovered[entity+"."+attr] = true
		}
		if target != "-" {
			entity, attr, ok := migrationAttributeRef(v.target, target)
			if !ok {
				v.errf("%s target %q is not a target Entity.attribute", where, target)
				continue
			}
			targetEntity = entity
			targetCovered[entity+"."+attr] = true
		}
		if sourceEntity != "" && targetEntity != "" {
			d, ok := v.dispositions[sourceEntity]
			if !ok || d.target != targetEntity || d.strategy == "retire" {
				v.errf("%s maps %s to %s outside its entity disposition", where, source, target)
				continue
			}
		}
		v.g.Count("data mappings")
	}
	for _, d := range v.dispositions {
		if d.strategy != "replace" {
			continue
		}
		for attr := range v.legacy.entities[d.legacy].attributes {
			if !sourceCovered[d.legacy+"."+attr] {
				v.errf("replace disposition %s -> %s does not map or drop legacy attribute %s.%s", d.legacy, d.target, d.legacy, attr)
			}
		}
		for attr := range v.target.entities[d.target].attributes {
			if !targetCovered[d.target+"."+attr] {
				v.errf("replace disposition %s -> %s does not map or derive target attribute %s.%s", d.legacy, d.target, d.target, attr)
			}
		}
	}
}

func migrationAttributeRef(model migrationModel, ref string) (string, string, bool) {
	parts := strings.Split(ref, ".")
	if len(parts) != 2 {
		return "", "", false
	}
	entity, ok := model.entities[parts[0]]
	if !ok {
		return "", "", false
	}
	if _, ok := entity.attributes[parts[1]]; !ok {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func (v *migrationValidator) validateStateMappings() {
	covered := map[string]bool{}
	for i, item := range migrationList(v.root.Get2("state_mappings")) {
		where := fmt.Sprintf("state_mappings[%d]", i)
		obj := item.AsObject()
		if obj == nil {
			v.errf("%s is not a mapping", where)
			continue
		}
		v.checkKeys(obj, migrationStateKeys, where)
		source, target := obj.GetString("source"), obj.GetString("target")
		if source == "" || target == "" || obj.GetString("reason") == "" {
			v.errf("%s needs source, target, and reason", where)
			continue
		}
		legacyEntity, legacyValue, ok := migrationStateRef(v.legacy, source)
		if !ok {
			v.errf("%s source %q is not a legacy lifecycle Entity.Value", where, source)
			continue
		}
		if covered[source] {
			v.errf("legacy state %q is mapped twice", source)
			continue
		}
		d, ok := v.dispositions[legacyEntity]
		if !ok || d.strategy == "retire" {
			v.errf("%s maps a lifecycle outside a non-retired disposition", where)
			continue
		}
		if target != "drain" {
			targetEntity, _, ok := migrationStateRef(v.target, target)
			if !ok || targetEntity != d.target {
				v.errf("%s target %q is not a lifecycle value on disposition target %s", where, target, d.target)
				continue
			}
		}
		covered[legacyEntity+"."+legacyValue] = true
		v.g.Count("state mappings")
	}
	for _, d := range v.dispositions {
		if d.strategy != "replace" {
			continue
		}
		for _, value := range v.legacy.entities[d.legacy].lifecycleValues {
			if !covered[d.legacy+"."+value] {
				v.errf("replace disposition %s -> %s does not map or drain legacy lifecycle value %s.%s", d.legacy, d.target, d.legacy, value)
			}
		}
	}
}

func migrationStateRef(model migrationModel, ref string) (string, string, bool) {
	parts := strings.Split(ref, ".")
	if len(parts) != 2 {
		return "", "", false
	}
	entity, ok := model.entities[parts[0]]
	if !ok || entity.lifecycleAttr == "" {
		return "", "", false
	}
	for _, value := range entity.lifecycleValues {
		if value == parts[1] {
			return parts[0], parts[1], true
		}
	}
	return "", "", false
}

func (v *migrationValidator) validatePhases() {
	items := migrationList(v.root.Get2("phases"))
	if len(items) < 2 {
		v.errf("phases must contain at least two ordered coexistence/cutover phases")
		return
	}
	seenTargetAuthority := false
	for i, item := range items {
		where := fmt.Sprintf("phases[%d]", i)
		obj := item.AsObject()
		if obj == nil {
			v.errf("%s is not a mapping", where)
			continue
		}
		v.checkKeys(obj, migrationPhaseKeys, where)
		phase := migrationPhase{id: obj.GetString("id"), sourceOfTruth: obj.GetString("source_of_truth"), readPath: obj.GetString("read_path"), writePath: obj.GetString("write_path")}
		if phase.id == "" {
			v.errf("%s.id is required", where)
			continue
		}
		if _, dup := v.phases[phase.id]; dup {
			v.errf("phase id %q is duplicated", phase.id)
			continue
		}
		if phase.sourceOfTruth != "legacy" && phase.sourceOfTruth != "target" {
			v.errf("%s.source_of_truth must be legacy or target", where)
		} else if phase.sourceOfTruth == "target" {
			seenTargetAuthority = true
		} else if seenTargetAuthority {
			v.errf("%s cannot return source_of_truth to legacy after target became authoritative; use cutover.rollback_phase for the rollback path", where)
		}
		if phase.readPath != "legacy" && phase.readPath != "target" && phase.readPath != "shadow" {
			v.errf("%s.read_path must be legacy, target, or shadow", where)
		}
		if phase.writePath != "legacy" && phase.writePath != "target" && phase.writePath != "dual" {
			v.errf("%s.write_path must be legacy, target, or dual", where)
		}
		for _, key := range []string{"backfill", "entry_criteria", "exit_criteria", "rollback"} {
			if obj.GetString(key) == "" {
				v.errf("%s.%s is required", where, key)
			}
		}
		observability, ok := migrationStrings(obj.Get2("observability"))
		if !ok || len(observability) == 0 {
			v.errf("%s.observability must list at least one signal", where)
		}
		if phase.writePath == "dual" {
			for _, key := range []string{"idempotency", "conflict_resolution", "reconciliation"} {
				if obj.GetString(key) == "" {
					v.errf("%s.%s is required when write_path is dual", where, key)
				}
			}
		}
		if phase.readPath == "shadow" && obj.GetString("parity") == "" {
			v.errf("%s.parity is required when read_path is shadow", where)
		}
		v.phases[phase.id] = phase
		v.phaseOrder[phase.id] = i
		v.g.Count("transition phases")
	}
	if first, ok := v.phases[v.phaseIDAt(0)]; ok && first.sourceOfTruth != "legacy" {
		v.errf("the first phase must keep legacy as source_of_truth")
	}
	if last, ok := v.phases[v.phaseIDAt(len(items)-1)]; ok && last.sourceOfTruth != "target" {
		v.errf("the final phase must make target the source_of_truth")
	}
}

func (v *migrationValidator) phaseIDAt(index int) string {
	for id, pos := range v.phaseOrder {
		if pos == index {
			return id
		}
	}
	return ""
}

func (v *migrationValidator) validateCutover() {
	obj := v.root.GetObject("cutover")
	if obj.Len() == 0 {
		v.errf("cutover is required")
		return
	}
	v.checkKeys(obj, migrationCutoverKeys, "cutover")
	phaseID, rollbackID := obj.GetString("phase"), obj.GetString("rollback_phase")
	phase, phaseOK := v.phases[phaseID]
	_, rollbackOK := v.phases[rollbackID]
	if !phaseOK {
		v.errf("cutover.phase %q is not a declared phase", phaseID)
	} else if phase.sourceOfTruth != "target" || phase.readPath != "target" || phase.writePath != "target" {
		v.errf("cutover.phase %q must use target for source_of_truth, read_path, and write_path", phaseID)
	}
	if !rollbackOK {
		v.errf("cutover.rollback_phase %q is not a declared phase", rollbackID)
	} else if phaseOK && v.phaseOrder[rollbackID] >= v.phaseOrder[phaseID] {
		v.errf("cutover.rollback_phase must precede cutover.phase")
	}
	for _, key := range []string{"decision_criteria", "rollback_window", "max_data_loss"} {
		if obj.GetString(key) == "" {
			v.errf("cutover.%s is required", key)
		}
	}
	if phaseOK && rollbackOK {
		v.g.Count("cutover contracts")
	}
}

func (v *migrationValidator) validateRisks() {
	items := migrationList(v.root.Get2("risks"))
	for i, item := range items {
		where := fmt.Sprintf("risks[%d]", i)
		obj := item.AsObject()
		if obj == nil {
			v.errf("%s is not a mapping", where)
			continue
		}
		v.checkKeys(obj, migrationRiskKeys, where)
		valid := true
		for _, key := range []string{"dependency", "detection", "mitigation", "residual", "owner"} {
			if obj.GetString(key) == "" {
				v.errf("%s.%s is required", where, key)
				valid = false
			}
		}
		if valid {
			v.g.Count("transition risks")
		}
	}
}

func (v *migrationValidator) validateNarrativeBridges() {
	arch := readFileOrErr(filepath.Join(v.design, "ARCHITECTURE.md"), v.g)
	if !headingContains(arch, "transition architecture") {
		v.errf("ARCHITECTURE.md needs a 'Transition architecture' heading describing the temporary coexistence topology and dependency failure posture")
	} else {
		v.g.Count("transition architecture sections")
	}
	build := readFileOrErr(filepath.Join(v.design, "BUILD.md"), v.g)
	if !headingContains(build, "migration implementation plan") {
		v.errf("BUILD.md needs a 'Migration implementation plan' heading that turns migration.yaml into build and test work")
	} else {
		v.g.Count("migration implementation plans")
	}
}
