package checkers

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/RamXX/machinery/internal/project"
)

// DefaultRegistry is the machine-local registry, relative to the project.
const DefaultRegistry = ".machinery/checkers.local.yaml"

// RegistryEnv names the environment variable that points at a registry.
const RegistryEnv = "MACHINERY_CHECKERS_REGISTRY"

// DefaultTimeout is the timeout of a checker command, in seconds.
const DefaultTimeout = 300

// Entry is the registration of one checker.
type Entry struct {
	Command        []string `yaml:"command"`
	TimeoutSeconds int      `yaml:"timeout_seconds,omitempty"`
}

// Registry is the content of the registry file (section 22.6).
type Registry struct {
	Version  int              `yaml:"version"`
	Checkers map[string]Entry `yaml:"checkers"`

	// Path is how messages name the registry file.
	Path string `yaml:"-"`
}

// LoadRegistry reads the registry named by the flag, else by the environment,
// else the default file. A missing file is an empty registry; a malformed one
// is a config error.
func LoadRegistry(p *project.Project, flag string) (*Registry, error) {
	name, abs := DefaultRegistry, p.Abs(DefaultRegistry)
	switch {
	case flag != "":
		name = flag
	case os.Getenv(RegistryEnv) != "":
		name = os.Getenv(RegistryEnv)
	}
	if name != DefaultRegistry {
		a, err := filepath.Abs(name)
		if err != nil {
			return nil, project.Errf("%s: %v", name, err)
		}
		abs = a
	}
	reg := &Registry{Version: 1, Checkers: map[string]Entry{}, Path: name}
	data, err := os.ReadFile(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return reg, nil
	}
	if err != nil {
		return nil, project.Errf("%s: %v", name, lastPart(err))
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var doc Registry
	if err := dec.Decode(&doc); err != nil {
		return nil, project.Errf("%s: %s", name, cleanYAML(err))
	}
	if doc.Version != 1 {
		return nil, project.Errf("%s: version must be 1", name)
	}
	var ids []string
	for id := range doc.Checkers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		e := doc.Checkers[id]
		if len(e.Command) == 0 || e.Command[0] == "" {
			return nil, project.Errf("%s: checker %s: command is required", name, id)
		}
		if e.TimeoutSeconds < 0 {
			return nil, project.Errf("%s: checker %s: timeout_seconds must not be negative", name, id)
		}
	}
	if doc.Checkers == nil {
		doc.Checkers = map[string]Entry{}
	}
	doc.Path = name
	return &doc, nil
}

func cleanYAML(err error) string {
	msg := err.Error()
	for _, pre := range []string{"yaml: unmarshal errors:\n  ", "yaml: "} {
		if len(msg) >= len(pre) && msg[:len(pre)] == pre {
			msg = msg[len(pre):]
		}
	}
	return string(bytes.ReplaceAll([]byte(msg), []byte("\n  "), []byte("; ")))
}

// Command returns the registration of a checker, or a config error.
func (r *Registry) Command(id string) (Entry, error) {
	e, ok := r.Checkers[id]
	if !ok {
		return Entry{}, project.Errf("no command registered for checker %s in %s", id, r.Path)
	}
	return e, nil
}

// Registered reports whether the checker has a command.
func (r *Registry) Registered(id string) bool {
	_, ok := r.Checkers[id]
	return ok
}
