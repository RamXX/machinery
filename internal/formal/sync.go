package formal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
)

// SyncClasses are the drift classifications of section 16.7.
var SyncClasses = map[string]bool{"no-effect": true, "model-drift": true, "promise-change": true, "possible-bug": true}

// Drift lists the commits after read_at that touch a source path, newest
// first. problem is non-empty when read_at cannot be used.
func Drift(p *project.Project, s *Spec) (commits []string, problem string) {
	switch repo, err := gitx.IsRepo(p.Root); {
	case err != nil:
		return nil, "git failed: " + err.Error()
	case !repo:
		return nil, "sources.read_at needs a git repository"
	}
	switch ok, err := gitx.Exists(p.Root, s.Sources.ReadAt); {
	case err != nil:
		return nil, "git failed: " + err.Error()
	case !ok:
		return nil, fmt.Sprintf("sources.read_at %s is not a commit", gitx.Short(s.Sources.ReadAt))
	}
	switch ok, err := gitx.IsAncestor(p.Root, s.Sources.ReadAt, "HEAD"); {
	case err != nil:
		return nil, "git failed: " + err.Error()
	case !ok:
		return nil, fmt.Sprintf("sources.read_at %s is not an ancestor of the current commit", gitx.Short(s.Sources.ReadAt))
	}
	var specs []string
	for _, sp := range s.Sources.Paths {
		specs = append(specs, ":(glob)"+sp)
	}
	for _, own := range s.OwnFiles() {
		specs = append(specs, ":(exclude)"+own)
	}
	specs = append(specs, ":(exclude,glob)"+s.Dir+"/traces/**", ":(exclude)"+BaselinePath(p))
	cs, err := gitx.CommitsTouching(p.Root, s.Sources.ReadAt, specs)
	if err != nil {
		return nil, "git rev-list failed: " + err.Error()
	}
	return cs, ""
}

// PendingBug returns the story of an open possible-bug classification, or "".
func (s *Spec) PendingBug() (story string, pending bool) {
	if n := len(s.Sources.Log); n > 0 && s.Sources.Log[n-1].Class == "possible-bug" {
		return s.Sources.Log[n-1].Story, true
	}
	return "", false
}

// changedSinceRead reports whether a file of the spec differs from its content at read_at.
func changedSinceRead(p *project.Project, s *Spec) (bool, error) {
	for _, f := range []string{s.SkeletonPath(), s.PropsPath()} {
		now, _ := p.Read(f)
		then, ok, err := gitx.Show(p.Root, s.Sources.ReadAt, f)
		if err != nil {
			return false, err
		}
		if !ok || !bytes.Equal(now, then) {
			return true, nil
		}
	}
	then, ok, err := gitx.Show(p.Root, s.Sources.ReadAt, s.Path)
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}
	var old Manifest
	if yaml.Unmarshal(then, &old) != nil {
		return true, nil
	}
	a, _ := json.Marshal(old)
	b, _ := json.Marshal(s.Manifest)
	return !bytes.Equal(a, b), nil
}

// Classify records a drift classification in the manifest (comments are kept).
func Classify(p *project.Project, s *Spec, class, evidence, story string) error {
	if !SyncClasses[class] {
		return project.Errf("unknown class %q: no-effect, model-drift, promise-change or possible-bug", class)
	}
	head, err := gitx.Head(p.Root)
	if err != nil {
		return err
	}
	switch class {
	case "model-drift":
		changed, err := changedSinceRead(p, s)
		if err != nil {
			return err
		}
		if !changed {
			return fmt.Errorf("%s: model-drift needs a change to the spec since %s; the model is updated from the code first", s.Spec, gitx.Short(s.Sources.ReadAt))
		}
	case "promise-change":
		if strings.TrimSpace(evidence) == "" {
			return project.Errf("promise-change needs --evidence naming the decision")
		}
	case "no-effect":
		if strings.TrimSpace(evidence) == "" {
			return project.Errf("no-effect needs --evidence saying why the change does not touch what the model describes")
		}
	case "possible-bug":
		if strings.TrimSpace(story) == "" {
			return project.Errf("possible-bug needs --story")
		}
	}
	data, err := p.Read(s.Path)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil || len(doc.Content) == 0 {
		return fmt.Errorf("%s: cannot parse", s.Path)
	}
	root := doc.Content[0]
	src := mapValue(root, "sources")
	if src == nil {
		return fmt.Errorf("%s: no sources block", s.Path)
	}
	from := s.Sources.ReadAt
	if class != "possible-bug" {
		if ra := mapValue(src, "read_at"); ra != nil {
			ra.SetString(head)
		}
	}
	entry := &yaml.Node{Kind: yaml.MappingNode, Style: yaml.FlowStyle}
	add := func(k, v string) {
		if v == "" {
			return
		}
		kn := &yaml.Node{Kind: yaml.ScalarNode, Value: k}
		vn := &yaml.Node{Kind: yaml.ScalarNode, Style: yaml.DoubleQuotedStyle}
		vn.SetString(v)
		vn.Style = yaml.DoubleQuotedStyle
		entry.Content = append(entry.Content, kn, vn)
	}
	add("at", head)
	add("from", from)
	add("class", class)
	add("evidence", evidence)
	add("story", story)
	log := mapValue(src, "log")
	if log == nil || log.Kind != yaml.SequenceNode {
		if log == nil {
			src.Content = append(src.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "log"}, &yaml.Node{Kind: yaml.SequenceNode})
			log = src.Content[len(src.Content)-1]
		} else {
			*log = yaml.Node{Kind: yaml.SequenceNode}
		}
	}
	log.Style = 0
	log.Content = append(log.Content, entry)
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return err
	}
	enc.Close()
	return p.WriteAtomic(s.Path, buf.Bytes())
}

func mapValue(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}
