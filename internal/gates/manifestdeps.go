package gates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

var (
	manifestModuleRe     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~/-]*\.[A-Za-z0-9._~/-]+$`)
	manifestVersionRe    = regexp.MustCompile(`^v[^\s]+$`)
	cargoKeyRe           = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
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
	var root map[string]any
	if err := toml.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	if root == nil {
		return nil, fmt.Errorf("cargo manifest root must be a table")
	}

	seen := map[string]bool{}
	addGroup := func(path string, value any) error {
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
			seen[name] = true
			seen[strings.ReplaceAll(name, "-", "_")] = true
		}
		return nil
	}
	addGroups := func(prefix string, table map[string]any, groups ...string) error {
		for _, group := range groups {
			if value, ok := table[group]; ok {
				path := group
				if prefix != "" {
					path = prefix + "." + group
				}
				if err := addGroup(path, value); err != nil {
					return err
				}
			}
		}
		return nil
	}

	groups := []string{"dependencies", "dev-dependencies", "build-dependencies"}
	if err := addGroups("", root, groups...); err != nil {
		return nil, err
	}
	if workspace, ok := root["workspace"]; ok {
		workspaceTable, tableOK := workspace.(map[string]any)
		if !tableOK {
			return nil, fmt.Errorf("workspace must be a table")
		}
		if err := addGroups("workspace", workspaceTable, groups...); err != nil {
			return nil, err
		}
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
			if err := addGroups("target."+name, entry, groups...); err != nil {
				return nil, err
			}
		}
	}

	deps := make([]string, 0, len(seen))
	for name := range seen {
		deps = append(deps, name)
	}
	sort.Strings(deps)
	return deps, nil
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
