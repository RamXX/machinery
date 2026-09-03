package gates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
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
	section := ""
	seen := map[string]bool{}
	var deps []string
	for lineNo, raw := range strings.Split(string(body), "\n") {
		line, err := stripManifestComment(raw)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo+1, err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") || strings.HasPrefix(line, "[[") != strings.HasSuffix(line, "]]") {
				return nil, fmt.Errorf("line %d: malformed TOML table header", lineNo+1)
			}
			section = strings.Trim(line, "[] ")
			if section == "" {
				return nil, fmt.Errorf("line %d: empty TOML table header", lineNo+1)
			}
			if dep, ok := cargoDependencyTable(section); ok {
				deps = append(deps, dep...)
			}
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("line %d: malformed TOML assignment", lineNo+1)
		}
		key, value := strings.Trim(strings.TrimSpace(parts[0]), `"'`), strings.TrimSpace(parts[1])
		if !cargoKeyRe.MatchString(key) || value == "" || !balancedManifestValue(value) {
			return nil, fmt.Errorf("line %d: malformed TOML key/value", lineNo+1)
		}
		identity := section + "\x00" + key
		if seen[identity] {
			return nil, fmt.Errorf("line %d: duplicate TOML key %q in [%s]", lineNo+1, key, section)
		}
		seen[identity] = true
		if cargoDependencySection(section) {
			deps = append(deps, key, strings.ReplaceAll(key, "-", "_"))
		}
	}
	return deps, nil
}

func cargoDependencySection(section string) bool {
	return section == "dependencies" || section == "workspace.dependencies" || (strings.HasPrefix(section, "target.") && strings.HasSuffix(section, ".dependencies"))
}

func cargoDependencyTable(section string) ([]string, bool) {
	for _, prefix := range []string{"dependencies.", "workspace.dependencies."} {
		if strings.HasPrefix(section, prefix) {
			name := strings.TrimPrefix(section, prefix)
			return []string{name, strings.ReplaceAll(name, "-", "_")}, true
		}
	}
	return nil, false
}

func stripManifestComment(line string) (string, error) {
	quote, escaped := byte(0), false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if escaped {
			escaped = false
			continue
		}
		if quote != 0 && c == '\\' && quote == '"' {
			escaped = true
			continue
		}
		if c == quote {
			quote = 0
			continue
		}
		if quote == 0 && (c == '"' || c == '\'') {
			quote = c
			continue
		}
		if quote == 0 && c == '#' {
			return line[:i], nil
		}
	}
	if quote != 0 {
		return "", fmt.Errorf("unterminated quoted string")
	}
	return line, nil
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
