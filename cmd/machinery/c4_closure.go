package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/RamXX/machinery/internal/designlock"
)

var structurizrRemoteReference = regexp.MustCompile(`(?i)(?:[a-z][a-z0-9+.-]*:)?//[^\s"']+`)

func withDesignWorkspaceSnapshot(design string, fn func(snapshot string) error) (retErr error) {
	lock, err := designlock.AcquireReader(design)
	if err != nil {
		return err
	}
	workspace, err := lock.MaterializeDesignWorkspace()
	if err != nil {
		return errors.Join(err, lock.Release())
	}
	snapshot := workspace.Path()
	defer func() {
		retErr = errors.Join(retErr, lock.CheckUnchanged(), workspace.Close(), lock.Release())
		retErr = remapSnapshotError(retErr, snapshot, design)
	}()
	designReaderAfterSnapshot()
	return fn(snapshot)
}

func resolveC4Include(scope, includingFile, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 && ((raw[0] == '"' && raw[len(raw)-1] == '"') || (raw[0] == '\'' && raw[len(raw)-1] == '\'')) {
		raw = raw[1 : len(raw)-1]
	}
	if raw == "" || strings.IndexByte(raw, 0) >= 0 {
		return "", fmt.Errorf("local !include path is empty or contains NUL")
	}
	if strings.ContainsAny(raw, "\r\n") {
		return "", fmt.Errorf("local !include path must stay on one line")
	}
	native := filepath.FromSlash(raw)
	if filepath.IsAbs(native) || filepath.VolumeName(native) != "" {
		return "", fmt.Errorf("local !include path %q must be relative", raw)
	}
	candidate := filepath.Clean(filepath.Join(filepath.Dir(includingFile), native))
	rel, err := filepath.Rel(scope, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("local !include path %q escapes the retained design workspace", raw)
	}
	return candidate, nil
}

func validateC4LocalDataPath(scope, includingFile, raw string) error {
	path, err := resolveC4Include(scope, includingFile, raw)
	if err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect local data path %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() && !info.IsDir() {
		return fmt.Errorf("local data path %s must be a real regular file or directory", path)
	}
	return nil
}

// structurizrSemanticLine removes comments without erasing a URL's //.
// Quoted content remains visible because remote references inside strings are
// exactly the network-capable forms this preflight must reject.
func structurizrSemanticLine(line string) string {
	var quote rune
	escaped := false
	runes := []rune(line)
	for i, r := range runes {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '"' || r == '\'' {
			quote = r
			continue
		}
		if r == '#' {
			return string(runes[:i])
		}
		if r == '/' && i+1 < len(runes) && runes[i+1] == '/' {
			prior := rune(0)
			for p := i - 1; p >= 0; p-- {
				if !unicode.IsSpace(runes[p]) {
					prior = runes[p]
					break
				}
			}
			if prior != ':' {
				return string(runes[:i])
			}
		}
	}
	return line
}

// validateStructurizrClosure accepts a deliberately small deterministic DSL
// subset: committed local regular-file !include directives within the retained
// workspace are recursively closed; executable extensions, workspace
// inheritance, custom implied-relationship strategies, and every remote URL
// are rejected before Java or Structurizr starts.
func validateStructurizrClosure(design, entry string) ([]string, error) {
	scope, err := designlock.RetainedWorkspaceScope(design)
	if err != nil {
		return nil, err
	}
	state := make(map[string]uint8)
	var closure []string
	var visit func(string) error
	visit = func(path string) error {
		path = filepath.Clean(path)
		switch state[path] {
		case 1:
			return fmt.Errorf("structurizr local include cycle reaches %s", path)
		case 2:
			return nil
		}
		state[path] = 1
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect structurizr input %s: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("structurizr input %s must be a regular non-symlink file", path)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read structurizr input %s: %w", path, err)
		}
		for lineNo, rawLine := range strings.Split(string(body), "\n") {
			line := strings.TrimSpace(structurizrSemanticLine(rawLine))
			if line == "" {
				continue
			}
			lower := strings.ToLower(line)
			where := fmt.Sprintf("%s:%d", path, lineNo+1)
			if strings.Contains(line, "${") {
				return fmt.Errorf("%s: environment or variable substitution is outside the deterministic local subset", where)
			}
			if remote := structurizrRemoteReference.FindString(line); remote != "" {
				return fmt.Errorf("%s: remote structurizr reference %q is outside the deterministic local subset", where, remote)
			}
			fields := strings.Fields(lower)
			// These constructs can dereference additional local paths at parse or
			// render time. Machinery's deterministic subset has no syntax-aware
			// way to bind those transitive inputs yet, so reject them rather than
			// silently executing against the mutable host filesystem.
			if len(fields) > 0 {
				switch fields[0] {
				case "theme", "themes", "icon", "logo", "image", "plantuml", "mermaid", "kroki":
					return fmt.Errorf("%s: %s can reference unbound local data and is outside the deterministic local subset", where, fields[0])
				}
			}
			if len(fields) >= 2 && fields[0] == "workspace" && fields[1] == "extends" {
				return fmt.Errorf("%s: workspace extends is outside the deterministic local subset", where)
			}
			for _, directive := range []string{"!script", "!plugin", "!components", "!const", "!var"} {
				if lower == directive || strings.HasPrefix(lower, directive+" ") || strings.HasPrefix(lower, directive+"\t") {
					return fmt.Errorf("%s: %s is executable or dynamically discovered and is outside the deterministic local subset", where, directive)
				}
			}
			if strings.HasPrefix(lower, "!impliedrelationships") {
				args := strings.TrimSpace(line[len("!impliedRelationships"):])
				if args != "" && !strings.EqualFold(args, "true") && !strings.EqualFold(args, "false") {
					return fmt.Errorf("%s: custom !impliedRelationships strategy %q is outside the deterministic local subset", where, args)
				}
			}
			if lower == "!docs" || strings.HasPrefix(lower, "!docs ") || strings.HasPrefix(lower, "!docs\t") {
				args := strings.Fields(strings.TrimSpace(line[len("!docs"):]))
				if len(args) != 1 {
					return fmt.Errorf("%s: !docs accepts exactly one local path; custom DocumentationImporter classes are outside the deterministic local subset", where)
				}
				if err := validateC4LocalDataPath(scope, path, args[0]); err != nil {
					return fmt.Errorf("%s: !docs: %w", where, err)
				}
			}
			if lower == "!adrs" || strings.HasPrefix(lower, "!adrs ") || strings.HasPrefix(lower, "!adrs\t") {
				args := strings.Fields(strings.TrimSpace(line[len("!adrs"):]))
				if len(args) < 1 || len(args) > 2 || (len(args) == 2 && !strings.EqualFold(args[1], "adr-tools")) {
					return fmt.Errorf("%s: !adrs accepts one local path and optional built-in adr-tools type; custom importer classes are outside the deterministic local subset", where)
				}
				if err := validateC4LocalDataPath(scope, path, args[0]); err != nil {
					return fmt.Errorf("%s: !adrs: %w", where, err)
				}
			}
			if lower == "!include" || strings.HasPrefix(lower, "!include ") || strings.HasPrefix(lower, "!include\t") {
				raw := strings.TrimSpace(line[len("!include"):])
				include, err := resolveC4Include(scope, path, raw)
				if err != nil {
					return fmt.Errorf("%s: %w", where, err)
				}
				if err := visit(include); err != nil {
					return err
				}
			}
		}
		state[path] = 2
		closure = append(closure, path)
		return nil
	}
	if err := visit(entry); err != nil {
		return nil, err
	}
	sort.Strings(closure)
	return closure, nil
}
