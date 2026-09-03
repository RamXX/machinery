package checker

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultRegistryPath is where machinery looks for the local checker registry
// when no --registry is given. It is repo-root relative and git-ignored: the
// registry names the local OCI control plane and immutable runtime closure,
// which must never leak into a committed design artifact (the design stays
// tool-neutral; see docs/external-checkers.md).
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
	Runtime OCIRuntime
}

// OCIRuntime is the only supported external-checker execution boundary. The
// immutable image digest binds the complete userspace closure (interpreter,
// modules, native loader/libraries, checker engine, and rules) without trusting
// host PATH or host language search paths.
type OCIRuntime struct {
	Engine   []string
	Image    string
	Digest   string
	Platform string
	Inputs   []OCIInput
}

type OCIInput struct {
	Source string
	Mount  string
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
	Runtime struct {
		Kind     string   `yaml:"kind"`
		Engine   []string `yaml:"engine"`
		Image    string   `yaml:"image"`
		Platform string   `yaml:"platform"`
		Inputs   []struct {
			Source string `yaml:"source"`
			Mount  string `yaml:"mount"`
		} `yaml:"inputs"`
	} `yaml:"runtime"`
}

// LoadRegistry reads and validates the registry at path. Malformed YAML is an
// error, and an entry with an empty run command is an error: an unusable
// registry must fail loudly, never resolve to a silent no-op.
func LoadRegistry(path string) (*Registry, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: checker registry must be a regular, non-symlink file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	nodeDecoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := nodeDecoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("%s: empty YAML document", path)
	}
	if err := rejectInvalidYAMLMappingKeys(doc.Content[0], "$registry"); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var extra yaml.Node
	if err := nodeDecoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("%s: multiple YAML documents are not allowed", path)
		}
		return nil, fmt.Errorf("%s: trailing YAML data: %w", path, err)
	}
	var raw rawRegistry
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if raw.Checkers == nil {
		return nil, fmt.Errorf("%s: checkers mapping is required", path)
	}
	reg := &Registry{Path: path, entries: make(map[string]Entry, len(raw.Checkers))}
	portableIDs := map[string]string{}
	ids := make([]string, 0, len(raw.Checkers))
	for id := range raw.Checkers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		re := raw.Checkers[id]
		if err := validatePortableComponent(id); err != nil {
			return nil, fmt.Errorf("%s: checker id %q is not portable: %w", path, id, err)
		}
		folded := strings.ToLower(id)
		if prior, exists := portableIDs[folded]; exists {
			return nil, fmt.Errorf("%s: checker ids %q and %q collide on a case-insensitive filesystem", path, prior, id)
		}
		portableIDs[folded] = id
		if len(re.Run) == 0 {
			return nil, fmt.Errorf("%s: checker %q has an empty run command; a checker with nothing to run is not a checker", path, id)
		}
		if re.Runtime.Kind != "oci" {
			return nil, fmt.Errorf("%s: checker %q runtime.kind must be \"oci\"; ambient host runtimes cannot provide a complete checker closure", path, id)
		}
		if len(re.Runtime.Engine) == 0 {
			return nil, fmt.Errorf("%s: checker %q runtime.engine must name the local OCI engine command", path, id)
		}
		for i, arg := range re.Runtime.Engine {
			if strings.TrimSpace(arg) == "" {
				return nil, fmt.Errorf("%s: checker %q runtime.engine argument %d is empty", path, id, i)
			}
		}
		digest, err := OCIImageDigest(re.Runtime.Image)
		if err != nil {
			return nil, fmt.Errorf("%s: checker %q runtime.image: %w", path, id, err)
		}
		if !validOCIPlatform(re.Runtime.Platform) {
			return nil, fmt.Errorf("%s: checker %q runtime.platform must be one of \"linux/amd64\" or \"linux/arm64\"", path, id)
		}
		inputs := make([]OCIInput, 0, len(re.Runtime.Inputs))
		mounts := map[string]string{}
		for i, input := range re.Runtime.Inputs {
			if strings.TrimSpace(input.Source) == "" {
				return nil, fmt.Errorf("%s: checker %q runtime.inputs[%d].source is empty", path, id, i)
			}
			if err := validatePortableRelativePath(input.Source); err != nil {
				return nil, fmt.Errorf("%s: checker %q runtime.inputs[%d].source is not a portable registry-relative path: %w", path, id, i, err)
			}
			if err := validatePortableRelativePath(input.Mount); err != nil {
				return nil, fmt.Errorf("%s: checker %q runtime.inputs[%d].mount is not portable: %w", path, id, i, err)
			}
			folded := strings.ToLower(input.Mount)
			if prior, exists := mounts[folded]; exists {
				return nil, fmt.Errorf("%s: checker %q runtime input mounts %q and %q alias on a case-insensitive filesystem", path, id, prior, input.Mount)
			}
			mounts[folded] = input.Mount
			inputs = append(inputs, OCIInput{Source: input.Source, Mount: input.Mount})
		}
		for _, commandSpec := range []struct {
			name string
			args []string
		}{{"run", re.Run}, {"verify", re.Verify}} {
			commandName, command := commandSpec.name, commandSpec.args
			for i, arg := range command {
				if arg == "" {
					return nil, fmt.Errorf("%s: checker %q %s command argument %d is empty", path, id, commandName, i)
				}
			}
		}
		to := DefaultCheckerTimeout
		if strings.TrimSpace(re.Timeout) != "" {
			d, perr := time.ParseDuration(re.Timeout)
			if perr != nil {
				return nil, fmt.Errorf("%s: checker %q has an invalid timeout %q: %w", path, id, re.Timeout, perr)
			}
			if d <= 0 {
				return nil, fmt.Errorf("%s: checker %q timeout must be positive", path, id)
			}
			to = d
		}
		reg.entries[id] = Entry{
			ID: id, Run: re.Run, Verify: re.Verify, Timeout: to,
			Runtime: OCIRuntime{Engine: append([]string(nil), re.Runtime.Engine...), Image: re.Runtime.Image, Digest: digest, Platform: re.Runtime.Platform, Inputs: inputs},
		}
	}
	return reg, nil
}

// RuntimeClosureDigest binds the immutable OCI userspace, exact selected
// platform, checker command topology, and every read-only checker input mounted
// outside the image.
func RuntimeClosureDigest(imageDigest, platform string, run, verify []string, inputs map[string]string) (string, error) {
	if !validSHA256(imageDigest) {
		return "", fmt.Errorf("OCI image digest is malformed")
	}
	if !validOCIPlatform(platform) {
		return "", fmt.Errorf("OCI platform must be one of \"linux/amd64\" or \"linux/arm64\"")
	}
	h := sha256.New()
	write := func(kind, value string) {
		_, _ = fmt.Fprintf(h, "%d:%s:%d:%s\n", len(kind), kind, len(value), value)
	}
	write("image", imageDigest)
	write("platform", platform)
	for _, command := range []struct {
		name string
		args []string
	}{{"run", run}, {"verify", verify}} {
		for i, arg := range command.args {
			write(fmt.Sprintf("%s[%d]", command.name, i), arg)
		}
	}
	mounts := make([]string, 0, len(inputs))
	for mount := range inputs {
		mounts = append(mounts, mount)
	}
	sort.Strings(mounts)
	for _, mount := range mounts {
		if !validSHA256(inputs[mount]) {
			return "", fmt.Errorf("runtime input %s digest is malformed", mount)
		}
		write("input.mount", mount)
		write("input.digest", inputs[mount])
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

func validOCIPlatform(platform string) bool {
	return platform == "linux/amd64" || platform == "linux/arm64"
}

// OCIImageDigest validates an immutable OCI reference and returns its closure
// digest. Tags and short/uppercase digests are intentionally rejected.
func OCIImageDigest(image string) (string, error) {
	const marker = "@sha256:"
	if strings.Count(image, marker) != 1 {
		return "", fmt.Errorf("must be an immutable OCI reference ending in @sha256:<64 lowercase hex>")
	}
	parts := strings.SplitN(image, marker, 2)
	if strings.TrimSpace(parts[0]) == "" || !validSHA256("sha256:"+parts[1]) {
		return "", fmt.Errorf("must be an immutable OCI reference ending in @sha256:<64 lowercase hex>")
	}
	if strings.ContainsAny(parts[0], " \t\r\n") {
		return "", fmt.Errorf("repository name contains whitespace")
	}
	return "sha256:" + parts[1], nil
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
