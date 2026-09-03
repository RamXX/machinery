package install

import (
	"errors"
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

// ValidateArtifact verifies an installed adapter without following an
// unexpected final symlink. Existence alone is not health: the artifact must
// have the expected type, any allowed link must target the shared canonical
// asset exactly, and rendered files must carry their identity marker.
func ValidateArtifact(artifact Artifact) error {
	info, err := os.Lstat(artifact.Path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if artifact.Target != string(TargetClaude) {
			return fmt.Errorf("unexpected symlink")
		}
		home, err := userHomeDir()
		if err != nil {
			return err
		}
		shared := filepath.Join(home, ".agents")
		var want string
		if artifact.Label == "machinery skill" {
			want = filepath.Join(shared, "skills", "machinery")
		} else {
			want = filepath.Join(shared, "agents", filepath.Base(artifact.Path))
		}
		got, err := os.Readlink(artifact.Path)
		if err != nil {
			return err
		}
		if !filepath.IsAbs(got) {
			got = filepath.Join(filepath.Dir(artifact.Path), got)
		}
		if !sameInstallPath(got, want) {
			return fmt.Errorf("symlink target is %s, want %s", got, want)
		}
		_, err = validateReceiptArtifactDigest(artifact.Path)
		return err
	}
	if artifact.Label == "machinery skill" {
		if !info.IsDir() {
			return fmt.Errorf("expected directory, got %s", info.Mode().Type())
		}
		if governed, err := validateReceiptArtifactDigest(artifact.Path); governed || err != nil {
			return err
		}
		raw, err := os.ReadFile(filepath.Join(artifact.Path, "SKILL.md"))
		if err != nil {
			return fmt.Errorf("read SKILL.md: %w", err)
		}
		text := strings.ReplaceAll(string(raw), "\r\n", "\n")
		if len(raw) < 1024 || !strings.HasPrefix(text, "---\nname: machinery\nmetadata:\n") || !strings.Contains(text, "\ndescription: >\n") || !strings.Contains(text, "\n---\n\n# machinery\n") {
			return fmt.Errorf("SKILL.md is truncated or does not match the machinery skill schema")
		}
		for _, rel := range []string{
			"references/build-md-template.md", "references/c4-standalone.md", "references/rebuild-guide.md",
			"references/surface-ledger.md", "references/target-surfaces.md", "references/xstate-format.md",
			"tools/README.md", "tools/tlc.sh", "tools/verify_formal.sh",
		} {
			child, err := os.Lstat(filepath.Join(artifact.Path, rel))
			if err != nil || child.Mode()&os.ModeSymlink != 0 || !child.Mode().IsRegular() || child.Size() == 0 {
				return fmt.Errorf("skill inventory entry %s is missing, empty, or not a real file", rel)
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("expected regular file, got %s", info.Mode().Type())
	}
	if governed, err := validateReceiptArtifactDigest(artifact.Path); governed || err != nil {
		return err
	}
	raw, err := os.ReadFile(artifact.Path)
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return fmt.Errorf("artifact is empty")
	}
	text := string(raw)
	base := filepath.Base(artifact.Path)
	switch {
	case strings.HasSuffix(base, ".toml"):
		name := strings.TrimSuffix(base, ".toml")
		if len(raw) < 1024 || !strings.HasPrefix(text, "name = \""+name+"\"\ndescription = ") || !strings.Contains(text, "\ndeveloper_instructions = '''\n") || !strings.HasSuffix(text, "'''\n") {
			return fmt.Errorf("rendered Codex agent is truncated or does not match %s schema", name)
		}
	case strings.HasPrefix(base, "machinery-") && strings.HasSuffix(base, ".md"):
		identity := strings.TrimSuffix(base, ".md")
		if artifact.Target == string(TargetOpenCode) {
			expectedDescription := ""
			for _, spec := range roleSpecs {
				if spec.Name == identity {
					expectedDescription = spec.Description
					break
				}
			}
			if len(raw) < 1024 || expectedDescription == "" || !strings.HasPrefix(text, "---\ndescription: "+fmt.Sprintf("%q", expectedDescription)+"\nmode: subagent\npermission:\n") || !strings.Contains(text, "\n---\n\n") {
				return fmt.Errorf("rendered OpenCode role is truncated or does not match %s schema", base)
			}
			break
		}
		phaseMarker := ""
		if strings.Contains(base, "fsm-author") {
			phaseMarker = "Phase 3"
		} else if strings.Contains(base, "build-writer") {
			phaseMarker = "Phase 4"
		}
		if len(raw) < 1024 || !strings.HasPrefix(text, "---\nname: "+identity+"\n") || !strings.Contains(text, "\ndescription: >\n") || !strings.Contains(text, "\n---\n") || (phaseMarker != "" && !strings.Contains(text, phaseMarker)) {
			return fmt.Errorf("role document is truncated or does not match %s schema", base)
		}
	case base == "machinery.js":
		for _, required := range []string{"async function runMachinery", `"tool.execute.before"`, `"tool.execute.after"`, `event: async`, `"session.idle"`} {
			if len(raw) < 1024 || !strings.Contains(text, required) {
				return fmt.Errorf("OpenCode adapter is truncated or missing contract %q", required)
			}
		}
	case strings.HasSuffix(base, ".md"):
		if len(raw) < 128 || !strings.HasPrefix(text, "---\ndescription: ") || !strings.Contains(text, "\n---\n\n") || !strings.Contains(text, "`machinery") {
			return fmt.Errorf("command adapter is truncated or does not match the machinery command schema")
		}
	}
	return nil
}

func validateReceiptArtifactDigest(path string) (bool, error) {
	receipt, exists, err := loadReceipt()
	if err != nil {
		return false, err
	}
	if !exists {
		return true, fmt.Errorf("no schema-%d installation receipt governs this artifact; run machinery install or machinery update", receiptSchema)
	}
	if receipt.SchemaVersion < receiptSchema {
		return true, fmt.Errorf("artifact is governed by legacy receipt schema %d without a content digest; run machinery update", receipt.SchemaVersion)
	}
	wantPaths, err := receiptArtifactPaths(receipt)
	if err != nil {
		return false, err
	}
	identity, err := installArtifactPathIdentity(path)
	if err != nil {
		return false, err
	}
	governed := false
	for _, candidate := range wantPaths {
		candidateID, idErr := installArtifactPathIdentity(candidate)
		if idErr == nil && candidateID == identity {
			governed = true
			break
		}
	}
	if !governed {
		return true, fmt.Errorf("artifact is absent from the schema-%d installation receipt topology; run machinery install or machinery update", receiptSchema)
	}
	for _, recorded := range receipt.Artifacts {
		recordedID, idErr := installArtifactPathIdentity(recorded.Path)
		if idErr != nil || recordedID != identity {
			continue
		}
		digest, err := artifactTreeDigest(path)
		if err != nil {
			return true, err
		}
		if digest != recorded.Digest {
			return true, fmt.Errorf("artifact digest is %s, want receipt-bound %s", digest, recorded.Digest)
		}
		return true, nil
	}
	return true, fmt.Errorf("artifact is governed by the installation receipt but absent from its inventory")
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

func installTargets(names []string, src string, copyAll bool, out io.Writer, before func(string) error) error {
	set, err := parseTargets(names)
	if err != nil {
		return err
	}
	if err := validateTargetSource(src, set); err != nil {
		return err
	}
	home, err := userHomeDir()
	if err != nil {
		return err
	}

	sharedHome := filepath.Join(home, ".agents")
	claudeHome := filepath.Join(home, ".claude")
	needShared := set[TargetCodex] || set[TargetOpenCode]
	if needShared {
		if before != nil {
			if err := before(sharedHome); err != nil {
				return err
			}
		}
		if err := placeReal(sharedHome, src, out); err != nil {
			return err
		}
	}
	if set[TargetClaude] {
		if before != nil {
			if err := before(claudeHome); err != nil {
				return err
			}
		}
		if needShared && !copyAll {
			if err := placeLinks(claudeHome, sharedHome, out); err != nil {
				return err
			}
		} else if err := placeReal(claudeHome, src, out); err != nil {
			return err
		}
	}
	if set[TargetCodex] {
		if before != nil {
			if err := before(string(TargetCodex)); err != nil {
				return err
			}
		}
		if err := installCodexAgents(home, src, out); err != nil {
			return err
		}
	}
	if set[TargetOpenCode] {
		if before != nil {
			if err := before(string(TargetOpenCode)); err != nil {
				return err
			}
		}
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
func UninstallTargets(names []string, out io.Writer) (retErr error) {
	if out == nil {
		out = io.Discard
	}
	set, err := parseTargets(names)
	if err != nil {
		return err
	}
	home, err := userHomeDir()
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
	var paths []string
	if set[TargetCodex] {
		for _, spec := range roleSpecs {
			paths = append(paths, filepath.Join(home, ".codex", "agents", spec.Name+".toml"))
		}
	}
	if set[TargetOpenCode] {
		base := filepath.Join(home, ".config", "opencode")
		for _, spec := range roleSpecs {
			paths = append(paths, filepath.Join(base, "agents", spec.Name+".md"))
		}
		for _, command := range openCodeCommands {
			paths = append(paths, filepath.Join(base, "commands", command))
		}
		paths = append(paths, filepath.Join(base, "plugins", "machinery.js"))
	}

	operationLock, err := acquireInstallOperationLock()
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, operationLock.Release()) }()
	receiptPath, err := installationReceiptPath()
	if err != nil {
		return err
	}
	receipt, receiptExists, err := loadReceipt()
	if err != nil {
		return err
	}
	if receiptExists {
		homes = expandCanonicalHomeGroups(receipt, homes)
	}
	paths = append(paths, homeInstallArtifactPaths(homes)...)
	txPaths := append(append([]string(nil), paths...), receiptPath)
	tx, err := beginArtifactTransaction(txPaths)
	if err != nil {
		return fmt.Errorf("snapshot target uninstall transaction: %w", err)
	}
	for _, path := range paths {
		if err := removeInstallArtifact(path); err != nil {
			return rollbackUninstallTransaction(tx, fmt.Errorf("remove install artifact %s: %w", path, err))
		}
	}
	if receiptExists {
		changed := forgetTargetsFromReceipt(&receipt, set)
		changed = forgetHomesFromReceipt(&receipt, homes) || changed
		if changed {
			if err := saveReceipt(receipt); err != nil {
				return rollbackUninstallTransaction(tx, fmt.Errorf("update installation receipt: %w", err))
			}
		}
	}
	if err := tx.commit(); err != nil {
		return fmt.Errorf("commit target uninstall transaction: %w", err)
	}
	for _, targetHome := range homes {
		fmt.Fprintf(out, "removed machinery -> %s\n", targetHome)
	}
	if set[TargetCodex] {
		fmt.Fprintf(out, "removed Codex agents -> %s\n", filepath.Join(home, ".codex", "agents"))
	}
	if set[TargetOpenCode] {
		fmt.Fprintf(out, "removed OpenCode agents + commands + governance adapter -> %s\n", filepath.Join(home, ".config", "opencode"))
	}
	return nil
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
	if err := durableMkdirAll(dir); err != nil {
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
	if err := durableMkdirAll(agentDir); err != nil {
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

func writeRendered(dst, content string) (retErr error) {
	if err := durableMkdirAll(filepath.Dir(dst)); err != nil {
		return err
	}
	tmp, cleanupTmp, err := installScratchFile(filepath.Dir(dst), "render")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { retErr = errors.Join(retErr, cleanupTmp()) }()
	if err := tmp.Chmod(0o644); err != nil {
		return errors.Join(err, closeInstallFile(tmp))
	}
	if _, err := tmp.WriteString(content); err != nil {
		return errors.Join(err, closeInstallFile(tmp))
	}
	if err := tmp.Sync(); err != nil {
		return errors.Join(err, closeInstallFile(tmp))
	}
	if err := closeInstallFile(tmp); err != nil {
		return err
	}
	return renameReplace(tmpPath, dst)
}

// TargetArtifacts returns the expected host-specific installation topology.
func TargetArtifacts(names []string) ([]Artifact, error) {
	set, err := parseTargets(names)
	if err != nil {
		return nil, err
	}
	home, err := userHomeDir()
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
