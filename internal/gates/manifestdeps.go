package gates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type cargoManifestRecord struct {
	dependencies     []string
	inherited        []string
	workspaceDeps    map[string]bool
	workspaceRoot    bool
	packageWorkspace string
}

var (
	manifestModuleRe     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~/-]*\.[A-Za-z0-9._~/-]+$`)
	manifestVersionRe    = regexp.MustCompile(`^v[^\s]+$`)
	cargoKeyRe           = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	cargoArtifactRe      = regexp.MustCompile(`^(?:bin(?::[A-Za-z0-9][A-Za-z0-9_.-]*)?|cdylib|staticlib)$`)
	requirementRe        = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9_.-]*)(?:\[[A-Za-z0-9_,.-]+\])?(?:\s*@\s*\S+|\s*(?:===|==|!=|~=|<=|>=|<|>)\s*[^\s,;]+(?:\s*,\s*(?:===|==|!=|~=|<=|>=|<|>)\s*[^\s,;]+)*)?(?:\s*;\s*.+)?$`)
	mixDepsRe            = regexp.MustCompile(`(?s)defp?\s+deps\s+do\s*\[(.*?)\]\s*end`)
	mixDependencyEntryRe = regexp.MustCompile(`^\{\s*:([a-z][a-z0-9_]*)\s*,`)
)

func parsePackageManifest(body []byte) ([]string, error) {
	if err := validateJSONDocument(body); err != nil {
		return nil, err
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	if root == nil {
		return nil, fmt.Errorf("package.json root must be a non-null object")
	}
	var deps []string
	for _, group := range []string{"dependencies", "optionalDependencies", "peerDependencies"} {
		raw, ok := root[group]
		if !ok {
			continue
		}
		var values map[string]json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, fmt.Errorf("%s must be an object of package names to version strings", group)
		}
		if values == nil {
			return nil, fmt.Errorf("%s must be a non-null object of package names to version strings", group)
		}
		names := make([]string, 0, len(values))
		for name := range values {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			versionRaw := values[name]
			var version string
			if err := json.Unmarshal(versionRaw, &version); err != nil || strings.TrimSpace(version) == "" {
				return nil, fmt.Errorf("%s.%s must be a nonempty version string", group, name)
			}
			deps = append(deps, name)
		}
	}
	return deps, nil
}

func validateJSONDocument(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := validateJSONValue(dec, "root"); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value after root")
		}
		return err
	}
	return nil
}

func validateJSONValue(dec *json.Decoder, where string) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, composite := tok.(json.Delim)
	if !composite {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("%s key must be a string", where)
			}
			if seen[key] {
				return fmt.Errorf("duplicate JSON key %q in %s", key, where)
			}
			seen[key] = true
			if err := validateJSONValue(dec, where+"."+key); err != nil {
				return err
			}
		}
	case '[':
		for i := 0; dec.More(); i++ {
			if err := validateJSONValue(dec, fmt.Sprintf("%s[%d]", where, i)); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
	_, err = dec.Token()
	return err
}

func parseGoModManifest(body []byte) ([]string, error) {
	var deps []string
	block := ""
	for lineNo, raw := range strings.Split(string(body), "\n") {
		line := strings.TrimSpace(strings.SplitN(raw, "//", 2)[0])
		if line == "" {
			continue
		}
		if line == ")" {
			if block == "" {
				return nil, fmt.Errorf("line %d: unmatched )", lineNo+1)
			}
			block = ""
			continue
		}
		if block != "" {
			if block == "require" {
				dep, err := parseGoRequire(line, lineNo+1)
				if err != nil {
					return nil, err
				}
				deps = append(deps, dep)
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "(" {
			switch fields[0] {
			case "require", "replace", "exclude", "retract", "use", "tool", "godebug":
				block = fields[0]
				continue
			}
			return nil, fmt.Errorf("line %d: unsupported block directive %q", lineNo+1, fields[0])
		}
		switch fields[0] {
		case "require":
			dep, err := parseGoRequire(strings.TrimSpace(strings.TrimPrefix(line, "require")), lineNo+1)
			if err != nil {
				return nil, err
			}
			deps = append(deps, dep)
		case "module", "go", "toolchain", "replace", "exclude", "retract", "use", "tool", "godebug":
			if len(fields) < 2 {
				return nil, fmt.Errorf("line %d: incomplete %s directive", lineNo+1, fields[0])
			}
		default:
			return nil, fmt.Errorf("line %d: unknown or malformed go.mod directive %q", lineNo+1, fields[0])
		}
	}
	if block != "" {
		return nil, fmt.Errorf("unterminated %s block", block)
	}
	return deps, nil
}

func parseGoRequire(line string, lineNo int) (string, error) {
	fields := strings.Fields(line)
	if len(fields) != 2 || !manifestModuleRe.MatchString(fields[0]) || !manifestVersionRe.MatchString(fields[1]) {
		return "", fmt.Errorf("line %d: malformed require directive", lineNo)
	}
	return fields[0], nil
}

func parseCargoManifest(body []byte) ([]string, error) {
	record, err := parseCargoManifestRecord(body)
	if err != nil {
		return nil, err
	}
	return record.dependencies, nil
}

func parseCargoManifestRecord(body []byte) (*cargoManifestRecord, error) {
	var root map[string]any
	if err := toml.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	if root == nil {
		return nil, fmt.Errorf("cargo manifest root must be a table")
	}

	record := &cargoManifestRecord{workspaceDeps: map[string]bool{}}
	seen := map[string]bool{}
	addGroup := func(path string, value any, workspaceDefinition bool) error {
		group, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be a dependency table", path)
		}
		names := make([]string, 0, len(group))
		for name := range group {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if !cargoKeyRe.MatchString(name) {
				return fmt.Errorf("%s has invalid dependency name %q", path, name)
			}
			inherited, err := validateCargoDependencySpec(path+"."+name, group[name], workspaceDefinition)
			if err != nil {
				return err
			}
			if inherited {
				record.inherited = append(record.inherited, name)
			}
			if workspaceDefinition {
				record.workspaceDeps[name] = true
			}
			seen[name] = true
			seen[strings.ReplaceAll(name, "-", "_")] = true
		}
		return nil
	}
	addGroups := func(prefix string, table map[string]any, workspaceDefinition bool, groups ...string) error {
		for _, alias := range []string{"dev_dependencies", "build_dependencies"} {
			if _, present := table[alias]; present {
				path := alias
				if prefix != "" {
					path = prefix + "." + alias
				}
				return fmt.Errorf("%s uses an underscore dependency-table alias that Cargo warns about; use the canonical hyphenated table name", path)
			}
		}
		for _, group := range groups {
			if value, ok := table[group]; ok {
				path := group
				if prefix != "" {
					path = prefix + "." + group
				}
				if err := addGroup(path, value, workspaceDefinition); err != nil {
					return err
				}
			}
		}
		return nil
	}

	groups := []string{"dependencies", "dev-dependencies", "build-dependencies"}
	if workspace, ok := root["workspace"]; ok {
		record.workspaceRoot = true
		workspaceTable, tableOK := workspace.(map[string]any)
		if !tableOK {
			return nil, fmt.Errorf("workspace must be a table")
		}
		for _, unsupported := range []string{"build-dependencies", "dev-dependencies"} {
			if _, present := workspaceTable[unsupported]; present {
				return nil, fmt.Errorf("workspace.%s is not a Cargo dependency group; only workspace.dependencies is supported by Cargo", unsupported)
			}
		}
		if err := addGroups("workspace", workspaceTable, true, "dependencies"); err != nil {
			return nil, err
		}
	}
	if pkg, ok := root["package"].(map[string]any); ok {
		if value, present := pkg["workspace"]; present {
			workspacePath, stringOK := value.(string)
			if !stringOK || strings.TrimSpace(workspacePath) == "" {
				return nil, fmt.Errorf("package.workspace must be a non-empty string")
			}
			record.packageWorkspace = workspacePath
		}
	}
	if err := addGroups("", root, false, groups...); err != nil {
		return nil, err
	}
	if targets, ok := root["target"]; ok {
		targetTable, tableOK := targets.(map[string]any)
		if !tableOK {
			return nil, fmt.Errorf("target must be a table")
		}
		targetNames := make([]string, 0, len(targetTable))
		for name := range targetTable {
			targetNames = append(targetNames, name)
		}
		sort.Strings(targetNames)
		for _, name := range targetNames {
			entry, entryOK := targetTable[name].(map[string]any)
			if !entryOK {
				return nil, fmt.Errorf("target.%s must be a table", name)
			}
			if err := addGroups("target."+name, entry, false, groups...); err != nil {
				return nil, err
			}
		}
	}

	deps := make([]string, 0, len(seen))
	for name := range seen {
		deps = append(deps, name)
	}
	sort.Strings(deps)
	record.dependencies = deps
	sort.Strings(record.inherited)
	return record, nil
}

func validateCargoDependencySpec(path string, value any, workspaceDefinition bool) (bool, error) {
	switch spec := value.(type) {
	case string:
		if err := validateCargoVersionRequirement(spec); err != nil {
			return false, fmt.Errorf("%s has invalid version requirement: %w", path, err)
		}
		return false, nil
	case map[string]any:
		if len(spec) == 0 {
			return false, fmt.Errorf("%s must not be an empty dependency table", path)
		}
		keys := make([]string, 0, len(spec))
		for key := range spec {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		stringFields := map[string]bool{
			"branch": true, "git": true, "package": true,
			"path": true, "registry": true, "rev": true,
			"tag": true, "target": true, "version": true,
		}
		boolFields := map[string]bool{
			"default-features": true, "lib": true, "optional": true, "public": true,
			"workspace": true,
		}
		for _, key := range keys {
			field := spec[key]
			switch {
			case stringFields[key]:
				text, ok := field.(string)
				if !ok || strings.TrimSpace(text) == "" {
					return false, fmt.Errorf("%s.%s must be a non-empty string", path, key)
				}
				if key == "version" {
					if err := validateCargoVersionRequirement(text); err != nil {
						return false, fmt.Errorf("%s.version has invalid version requirement: %w", path, err)
					}
				}
				if key == "package" && !cargoKeyRe.MatchString(text) {
					return false, fmt.Errorf("%s.package has invalid Cargo package name %q", path, text)
				}
				if key == "git" && strings.Contains(text, "#") {
					return false, fmt.Errorf("%s.git must not contain a URL fragment; Cargo ignores it with a warning", path)
				}
			case boolFields[key]:
				flag, ok := field.(bool)
				if !ok {
					return false, fmt.Errorf("%s.%s must be a boolean", path, key)
				}
				_ = flag
			case key == "artifact":
				switch artifact := field.(type) {
				case string:
					if !cargoArtifactRe.MatchString(artifact) {
						return false, fmt.Errorf("%s.artifact has unsupported kind %q", path, artifact)
					}
				case []any:
					if len(artifact) == 0 {
						return false, fmt.Errorf("%s.artifact must be a non-empty string or string array", path)
					}
					seenKinds := map[string]bool{}
					hasAllBins, hasSelectedBin := false, false
					for i, kind := range artifact {
						text, ok := kind.(string)
						if !ok || !cargoArtifactRe.MatchString(text) {
							return false, fmt.Errorf("%s.artifact[%d] has unsupported kind", path, i)
						}
						if seenKinds[text] {
							return false, fmt.Errorf("%s.artifact repeats kind %q", path, text)
						}
						seenKinds[text] = true
						hasAllBins = hasAllBins || text == "bin"
						hasSelectedBin = hasSelectedBin || strings.HasPrefix(text, "bin:")
					}
					if hasAllBins && hasSelectedBin {
						return false, fmt.Errorf("%s.artifact cannot combine bin with selected bin:<name> kinds", path)
					}
				default:
					return false, fmt.Errorf("%s.artifact must be a non-empty string or string array", path)
				}
			case key == "features":
				features, ok := field.([]any)
				if !ok {
					return false, fmt.Errorf("%s.features must be a string array", path)
				}
				for i, feature := range features {
					text, ok := feature.(string)
					if !ok || strings.TrimSpace(text) == "" {
						return false, fmt.Errorf("%s.features[%d] must be a non-empty string", path, i)
					}
				}
			default:
				return false, fmt.Errorf("%s has unsupported dependency field %q", path, key)
			}
		}
		if workspaceDefinition {
			if _, ok := spec["workspace"]; ok {
				return false, fmt.Errorf("%s cannot inherit from workspace inside workspace.dependencies", path)
			}
			if _, ok := spec["optional"]; ok {
				return false, fmt.Errorf("%s cannot be optional inside workspace.dependencies", path)
			}
			if _, ok := spec["public"]; ok {
				return false, fmt.Errorf("%s cannot be public inside workspace.dependencies", path)
			}
		}

		workspace, inherited := spec["workspace"]
		if inherited {
			if workspace != true {
				return false, fmt.Errorf("%s.workspace must be true", path)
			}
			for _, conflicting := range []string{
				"artifact", "branch", "default-features", "git", "lib", "package", "path", "public",
				"registry", "rev", "tag", "target", "version",
			} {
				if _, ok := spec[conflicting]; ok {
					return false, fmt.Errorf("%s.workspace cannot be combined with %s", path, conflicting)
				}
			}
			return true, nil
		}

		hasSource := false
		for _, source := range []string{"git", "path", "version"} {
			if _, ok := spec[source]; ok {
				hasSource = true
			}
		}
		if !hasSource {
			return false, fmt.Errorf("%s must declare version, path, git, or workspace = true", path)
		}
		selectors := 0
		for _, selector := range []string{"branch", "rev", "tag"} {
			if _, ok := spec[selector]; ok {
				selectors++
				if _, hasGit := spec["git"]; !hasGit {
					return false, fmt.Errorf("%s.%s requires git", path, selector)
				}
			}
		}
		if selectors > 1 {
			return false, fmt.Errorf("%s may declare only one of branch, rev, or tag", path)
		}
		_, hasGit := spec["git"]
		_, hasPath := spec["path"]
		if hasGit && hasPath {
			return false, fmt.Errorf("%s cannot combine git and path sources", path)
		}
		_, hasArtifact := spec["artifact"]
		for _, artifactOnly := range []string{"lib", "target"} {
			if _, ok := spec[artifactOnly]; ok && !hasArtifact {
				return false, fmt.Errorf("%s.%s requires artifact", path, artifactOnly)
			}
		}
		if _, hasRegistry := spec["registry"]; hasRegistry {
			if _, hasVersion := spec["version"]; !hasVersion {
				return false, fmt.Errorf("%s.registry requires version", path)
			}
			if hasGit || hasPath {
				return false, fmt.Errorf("%s.registry cannot be combined with git or path", path)
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("%s must be a non-empty version string or dependency table", path)
	}
}

// validateCargoVersionRequirement implements Cargo's VersionReq grammar:
// star or comma-separated comparators, each holding an optional Cargo
// operator and a one-to-three-component partial SemVer.
func validateCargoVersionRequirement(requirement string) error {
	requirement = strings.TrimSpace(requirement)
	if requirement == "" {
		return fmt.Errorf("empty version requirement")
	}
	for _, raw := range strings.Split(requirement, ",") {
		part := strings.TrimSpace(raw)
		if part == "" {
			return fmt.Errorf("empty comparator")
		}
		op := ""
		for _, candidate := range []string{">=", "<=", "^", "~", ">", "<", "="} {
			if strings.HasPrefix(part, candidate) {
				op = candidate
				part = strings.TrimSpace(strings.TrimPrefix(part, candidate))
				break
			}
		}
		if part == "" || strings.ContainsAny(part, " \t\r\n") {
			return fmt.Errorf("malformed comparator %q", raw)
		}
		core := part
		if strings.Contains(core, "+") {
			return fmt.Errorf("build metadata in %q is ignored by Cargo and is forbidden by the zero-warning contract", raw)
		}
		if dash := strings.IndexByte(core, '-'); dash >= 0 {
			if !validCargoSemverIdentifiers(core[dash+1:]) {
				return fmt.Errorf("malformed prerelease in %q", raw)
			}
			core = core[:dash]
		}
		components := strings.Split(core, ".")
		if len(components) == 0 || len(components) > 3 {
			return fmt.Errorf("version in %q must have one to three components", raw)
		}
		wildcard := false
		for i, component := range components {
			if component == "*" || component == "x" || component == "X" {
				if op != "" || i != len(components)-1 || strings.ContainsAny(part, "-+") {
					return fmt.Errorf("wildcard is not valid in %q", raw)
				}
				wildcard = true
				continue
			}
			if wildcard || component == "" || (len(component) > 1 && component[0] == '0') {
				return fmt.Errorf("malformed numeric component in %q", raw)
			}
			if _, err := strconv.ParseUint(component, 10, 64); err != nil {
				return fmt.Errorf("malformed numeric component in %q", raw)
			}
		}
		if strings.ContainsAny(part, "-+") && len(components) != 3 {
			return fmt.Errorf("prerelease and build metadata require a full version in %q", raw)
		}
	}
	return nil
}

func validCargoSemverIdentifiers(value string) bool {
	if value == "" {
		return false
	}
	for _, identifier := range strings.Split(value, ".") {
		if identifier == "" {
			return false
		}
		allNumeric := true
		for _, r := range identifier {
			valid := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '-'
			if !valid {
				return false
			}
			if r < '0' || r > '9' {
				allNumeric = false
			}
		}
		if allNumeric && len(identifier) > 1 && identifier[0] == '0' {
			return false
		}
	}
	return true
}

func balancedManifestValue(value string) bool {
	stack := []byte{}
	quote, escaped := byte(0), false
	for i := 0; i < len(value); i++ {
		c := value[i]
		if escaped {
			escaped = false
			continue
		}
		if quote != 0 {
			if c == '\\' && quote == '"' {
				escaped = true
			} else if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		switch c {
		case '{', '[', '(':
			stack = append(stack, c)
		case '}', ']', ')':
			if len(stack) == 0 || !matchingManifestDelimiter(stack[len(stack)-1], c) {
				return false
			}
			stack = stack[:len(stack)-1]
		}
	}
	return quote == 0 && len(stack) == 0
}

func matchingManifestDelimiter(open, close byte) bool {
	return open == '{' && close == '}' || open == '[' && close == ']' || open == '(' && close == ')'
}

func parseRequirementsManifest(body []byte) ([]string, error) {
	var deps []string
	for lineNo, raw := range strings.Split(string(body), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, "\\") || strings.HasPrefix(line, "-") || strings.Contains(line, "#egg=") {
			return nil, fmt.Errorf("line %d: recursive/options/VCS requirement cannot be enumerated completely", lineNo+1)
		}
		if i := strings.Index(line, " #"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		match := requirementRe.FindStringSubmatch(line)
		if match == nil {
			return nil, fmt.Errorf("line %d: malformed requirement", lineNo+1)
		}
		deps = append(deps, match[1], strings.ReplaceAll(match[1], "-", "_"))
	}
	return deps, nil
}

func parseMixManifest(body []byte) ([]string, error) {
	text := string(body)
	if !balancedManifestValue(text) {
		return nil, fmt.Errorf("unbalanced Elixir delimiters or quoted string")
	}
	match := mixDepsRe.FindStringSubmatch(text)
	if match == nil {
		if strings.Contains(text, "deps:") || strings.Contains(text, "defp deps") || strings.Contains(text, "def deps") {
			return nil, fmt.Errorf("dependency declaration is dynamic or malformed; expected defp deps do [...] end")
		}
		return nil, nil
	}
	entries, err := splitManifestList(match[1])
	if err != nil {
		return nil, err
	}
	var deps []string
	for _, entry := range entries {
		m := mixDependencyEntryRe.FindStringSubmatch(strings.TrimSpace(entry))
		if m == nil || !strings.HasSuffix(strings.TrimSpace(entry), "}") {
			return nil, fmt.Errorf("unsupported or malformed Mix dependency entry %q", strings.TrimSpace(entry))
		}
		deps = append(deps, m[1])
	}
	return deps, nil
}

func splitManifestList(text string) ([]string, error) {
	var out []string
	start, depth := 0, 0
	quote, escaped := byte(0), false
	for i := 0; i < len(text); i++ {
		c := text[i]
		if escaped {
			escaped = false
			continue
		}
		if quote != 0 {
			switch c {
			case '\\':
				escaped = true
			default:
				if c == quote {
					quote = 0
				}
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		switch c {
		case '{', '[', '(':
			depth++
		case '}', ']', ')':
			depth--
		case ',':
			if depth == 0 {
				if entry := strings.TrimSpace(text[start:i]); entry != "" {
					out = append(out, entry)
				}
				start = i + 1
			}
		}
		if depth < 0 {
			return nil, fmt.Errorf("malformed dependency list")
		}
	}
	if entry := strings.TrimSpace(text[start:]); entry != "" {
		out = append(out, entry)
	}
	return out, nil
}
