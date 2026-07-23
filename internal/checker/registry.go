package checker

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultRegistryPath is where machinery looks for the local checker registry
// when no --registry is given. It is repo-root relative and git-ignored: the
// registry names the actual binaries, which must never leak into a committed
// design artifact (the design stays tool-neutral; see docs/external-checkers.md).
const DefaultRegistryPath = ".machinery/checkers.local.yaml"

// DefaultCheckerTimeout bounds every checker invocation when the entry omits an
// explicit timeout. A checker that never returns is a verification failure, not
// a hang.
const DefaultCheckerTimeout = 120 * time.Second

// Entry is one checker's resolved local wiring: how to run its adapter (which
// produces fresh evidence at {out}), an optional replay/verify command, and the
// per-invocation timeout.
type Entry struct {
	ID      string
	Run     []string
	Verify  []string
	Timeout time.Duration
}

// Registry maps a checker id to its Entry. It is the resolution layer that
// keeps tool names out of the design: the manifest names an id, the registry
// (here, outside the design) says what that id runs.
type Registry struct {
	Path    string
	entries map[string]Entry
}

// rawRegistry / rawEntry are the on-disk YAML shape, kept separate from Entry so
// the timeout can be parsed from its string form once, at load time.
type rawRegistry struct {
	Checkers map[string]rawEntry `yaml:"checkers"`
}

type rawEntry struct {
	Run     []string `yaml:"run"`
	Verify  []string `yaml:"verify"`
	Timeout string   `yaml:"timeout"`
}

// LoadRegistry reads and validates the registry at path. Malformed YAML is an
// error, and an entry with an empty run command is an error: an unusable
// registry must fail loudly, never resolve to a silent no-op.
func LoadRegistry(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw rawRegistry
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	reg := &Registry{Path: path, entries: make(map[string]Entry, len(raw.Checkers))}
	for id, re := range raw.Checkers {
		if len(re.Run) == 0 {
			return nil, fmt.Errorf("%s: checker %q has an empty run command; a checker with nothing to run is not a checker", path, id)
		}
		to := DefaultCheckerTimeout
		if strings.TrimSpace(re.Timeout) != "" {
			d, perr := time.ParseDuration(re.Timeout)
			if perr != nil {
				return nil, fmt.Errorf("%s: checker %q has an invalid timeout %q: %w", path, id, re.Timeout, perr)
			}
			to = d
		}
		reg.entries[id] = Entry{ID: id, Run: re.Run, Verify: re.Verify, Timeout: to}
	}
	return reg, nil
}

// Resolve returns the entry for id and whether it was found.
func (r *Registry) Resolve(id string) (Entry, bool) {
	e, ok := r.entries[id]
	return e, ok
}

// IDs returns the configured checker ids in sorted order, so any report over
// the registry is stable.
func (r *Registry) IDs() []string {
	out := make([]string, 0, len(r.entries))
	for id := range r.entries {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Tokens carries the concrete paths that substitute into a registry command.
// They are resolved OUTSIDE the design so no path leaks into a committed
// artifact: {projection} and {manifest} are committed inputs, {config} and
// {out} are ephemeral temp files, {design} is the design directory.
type Tokens struct {
	Projection string
	Config     string
	Manifest   string
	Out        string
	Design     string
}

// Substitute replaces every supported token in each arg with its concrete
// path, returning a fresh slice (the entry's Run/Verify are never mutated).
func (t Tokens) Substitute(args []string) []string {
	repl := strings.NewReplacer(
		"{projection}", t.Projection,
		"{config}", t.Config,
		"{manifest}", t.Manifest,
		"{out}", t.Out,
		"{design}", t.Design,
	)
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = repl.Replace(a)
	}
	return out
}
