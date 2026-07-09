package install

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Target identifies a first-class agent host adapter. The shared Agent Skills
// installation is implicit for Codex and OpenCode because both discover
// ~/.agents/skills.
type Target string

const (
	TargetClaude   Target = "claude"
	TargetCodex    Target = "codex"
	TargetOpenCode Target = "opencode"
	TargetAll      Target = "all"
)

var targetOrder = []Target{TargetClaude, TargetCodex, TargetOpenCode}

// Artifact is one expected installed file or directory used by doctor.
type Artifact struct {
	Target string
	Label  string
	Path   string
}

type roleSpec struct {
	File        string
	Name        string
	Description string
}

var roleSpecs = []roleSpec{
	{
		File:        "machinery-fsm-author.md",
		Name:        "machinery-fsm-author",
		Description: "Author machinery Phase 3 state-machine contracts from the domain and architecture artifacts.",
	},
	{
		File:        "machinery-build-writer.md",
		Name:        "machinery-build-writer",
		Description: "Assemble machinery Phase 4 BUILD.md from the checked domain, architecture, machines, and oracles.",
	},
}

var openCodeCommands = []string{"design.md", "check.md", "init.md", "status.md"}

func parseTargets(names []string) (map[Target]bool, error) {
	set := map[Target]bool{}
	for _, raw := range names {
		name := Target(strings.ToLower(strings.TrimSpace(raw)))
		switch name {
		case TargetAll:
			for _, target := range targetOrder {
				set[target] = true
			}
		case TargetClaude, TargetCodex, TargetOpenCode:
			set[name] = true
		case "":
			continue
		default:
			return nil, fmt.Errorf("unknown install target %q (want claude, codex, opencode, or all)", raw)
		}
	}
	if len(set) == 0 {
		return nil, fmt.Errorf("at least one install target is required")
	}
	return set, nil
}

func installTargets(names []string, src string, copyAll bool, out io.Writer) error {
	set, err := parseTargets(names)
	if err != nil {
		return err
	}
	if err := validateTargetSource(src, set); err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	sharedHome := filepath.Join(home, ".agents")
	claudeHome := filepath.Join(home, ".claude")
	needShared := set[TargetCodex] || set[TargetOpenCode]
	if needShared {
		if err := placeReal(sharedHome, src, out); err != nil {
			return err
		}
	}
	if set[TargetClaude] {
		if needShared && !copyAll {
			if err := placeLinks(claudeHome, sharedHome, out); err != nil {
				return err
			}
		} else if err := placeReal(claudeHome, src, out); err != nil {
			return err
		}
	}
	if set[TargetCodex] {
		if err := installCodexAgents(home, src, out); err != nil {
			return err
		}
	}
	if set[TargetOpenCode] {
		if err := installOpenCodeAdapter(home, src, out); err != nil {
			return err
		}
	}
	return nil
}

// UninstallTargets removes the host-native assets selected by names. A
// complete selection (normally --target all) also removes the shared
// ~/.agents copy. A single Codex or OpenCode removal deliberately preserves
// that shared copy because the other host, or another Agent Skills runtime,
// may still consume it.
func UninstallTargets(names []string, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	set, err := parseTargets(names)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	var homes []string
	if set[TargetClaude] {
		homes = append(homes, filepath.Join(home, ".claude"))
	}
	if len(set) == len(targetOrder) {
		homes = append(homes, filepath.Join(home, ".agents"))
	}
	if len(homes) > 0 {
		if err := Uninstall(homes, out); err != nil {
			return err
		}
	}

	if set[TargetCodex] {
		for _, spec := range roleSpecs {
			if err := removeIfPresent(filepath.Join(home, ".codex", "agents", spec.Name+".toml")); err != nil {
				return err
			}
		}
		fmt.Fprintf(out, "removed Codex agents -> %s\n", filepath.Join(home, ".codex", "agents"))
	}
	if set[TargetOpenCode] {
		base := filepath.Join(home, ".config", "opencode")
		for _, spec := range roleSpecs {
			if err := removeIfPresent(filepath.Join(base, "agents", spec.Name+".md")); err != nil {
				return err
			}
		}
		for _, command := range openCodeCommands {
			if err := removeIfPresent(filepath.Join(base, "commands", command)); err != nil {
				return err
			}
		}
		if err := removeIfPresent(filepath.Join(base, "plugins", "machinery.js")); err != nil {
			return err
		}
		fmt.Fprintf(out, "removed OpenCode agents + commands + governance adapter -> %s\n", base)
	}
	return nil
}

func removeIfPresent(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func validateTargetSource(src string, targets map[Target]bool) error {
	if targets[TargetOpenCode] {
		for _, command := range openCodeCommands {
			if _, err := os.Stat(filepath.Join(src, "adapters", "opencode", "commands", command)); err != nil {
				return fmt.Errorf("source is missing OpenCode command adapter %s: %w", command, err)
			}
		}
		if _, err := os.Stat(filepath.Join(src, "adapters", "opencode", "plugins", "machinery.js")); err != nil {
			return fmt.Errorf("source is missing OpenCode governance adapter: %w", err)
		}
	}
	return nil
}

func installCodexAgents(home, src string, out io.Writer) error {
	dir := filepath.Join(home, ".codex", "agents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, spec := range roleSpecs {
		body, err := canonicalRoleBody(src, spec)
		if err != nil {
			return err
		}
		doc, err := renderCodexRole(spec, body)
		if err != nil {
			return err
		}
		if err := writeRendered(filepath.Join(dir, spec.Name+".toml"), doc); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "installed Codex agents -> %s\n", dir)
	return nil
}

func installOpenCodeAdapter(home, src string, out io.Writer) error {
	base := filepath.Join(home, ".config", "opencode")
	agentDir := filepath.Join(base, "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}
	for _, spec := range roleSpecs {
		body, err := canonicalRoleBody(src, spec)
		if err != nil {
			return err
		}
		if err := writeRendered(filepath.Join(agentDir, spec.Name+".md"), renderOpenCodeRole(spec, body)); err != nil {
			return err
		}
	}

	commandDir := filepath.Join(base, "commands")
	for _, command := range openCodeCommands {
		if err := copyFile(
			filepath.Join(src, "adapters", "opencode", "commands", command),
			filepath.Join(commandDir, command),
		); err != nil {
			return err
		}
	}
	pluginDir := filepath.Join(base, "plugins")
	if err := copyFile(
		filepath.Join(src, "adapters", "opencode", "plugins", "machinery.js"),
		filepath.Join(pluginDir, "machinery.js"),
	); err != nil {
		return err
	}
	fmt.Fprintf(out, "installed OpenCode agents + commands + governance adapter -> %s\n", base)
	return nil
}

func canonicalRoleBody(src string, spec roleSpec) (string, error) {
	raw, err := os.ReadFile(filepath.Join(src, agentsRel, spec.File))
	if err != nil {
		return "", err
	}
	doc := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(doc, "---\n") {
		return "", fmt.Errorf("role doc %s has no YAML frontmatter", spec.File)
	}
	end := strings.Index(doc[4:], "\n---\n")
	if end < 0 {
		return "", fmt.Errorf("role doc %s has unterminated YAML frontmatter", spec.File)
	}
	body := strings.TrimLeft(doc[4+end+5:], "\n")
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf("role doc %s has an empty canonical body", spec.File)
	}
	return strings.TrimRight(body, "\n") + "\n", nil
}

func renderCodexRole(spec roleSpec, body string) (string, error) {
	if strings.Contains(body, "'''") {
		return "", fmt.Errorf("role doc %s contains a TOML multiline-literal delimiter", spec.File)
	}
	return fmt.Sprintf("name = %q\ndescription = %q\ndeveloper_instructions = '''\n%s'''\n", spec.Name, spec.Description, body), nil
}

func renderOpenCodeRole(spec roleSpec, body string) string {
	return fmt.Sprintf(`---
description: %q
mode: subagent
permission:
  read: allow
  edit: allow
  glob: allow
  grep: allow
  list: allow
  bash: allow
  skill: allow
  question: allow
---

%s`, spec.Description, body)
}

func writeRendered(dst, content string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(dst, []byte(content), 0o644)
}

// TargetArtifacts returns the expected host-specific installation topology.
func TargetArtifacts(names []string) ([]Artifact, error) {
	set, err := parseTargets(names)
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	var artifacts []Artifact
	if set[TargetCodex] || set[TargetOpenCode] {
		shared := filepath.Join(home, ".agents")
		artifacts = append(artifacts, Artifact{Target: "shared", Label: "machinery skill", Path: filepath.Join(shared, "skills", "machinery")})
		for _, spec := range roleSpecs {
			artifacts = append(artifacts, Artifact{Target: "shared", Label: spec.Name + " role", Path: filepath.Join(shared, "agents", spec.File)})
		}
	}
	if set[TargetClaude] {
		base := filepath.Join(home, ".claude")
		artifacts = append(artifacts, Artifact{Target: string(TargetClaude), Label: "machinery skill", Path: filepath.Join(base, "skills", "machinery")})
		for _, spec := range roleSpecs {
			artifacts = append(artifacts, Artifact{Target: string(TargetClaude), Label: spec.Name + " agent", Path: filepath.Join(base, "agents", spec.File)})
		}
	}
	if set[TargetCodex] {
		for _, spec := range roleSpecs {
			artifacts = append(artifacts, Artifact{Target: string(TargetCodex), Label: spec.Name + " agent", Path: filepath.Join(home, ".codex", "agents", spec.Name+".toml")})
		}
	}
	if set[TargetOpenCode] {
		base := filepath.Join(home, ".config", "opencode")
		for _, spec := range roleSpecs {
			artifacts = append(artifacts, Artifact{Target: string(TargetOpenCode), Label: spec.Name + " agent", Path: filepath.Join(base, "agents", spec.Name+".md")})
		}
		for _, command := range openCodeCommands {
			artifacts = append(artifacts, Artifact{Target: string(TargetOpenCode), Label: "command " + strings.TrimSuffix(command, ".md"), Path: filepath.Join(base, "commands", command)})
		}
		artifacts = append(artifacts, Artifact{Target: string(TargetOpenCode), Label: "governance adapter", Path: filepath.Join(base, "plugins", "machinery.js")})
	}
	sort.SliceStable(artifacts, func(i, j int) bool {
		if artifacts[i].Target == artifacts[j].Target {
			return artifacts[i].Path < artifacts[j].Path
		}
		return artifacts[i].Target < artifacts[j].Target
	})
	return artifacts, nil
}
