package checker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeRegistry drops a registry file into a temp dir and returns its path.
func writeRegistry(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "checkers.local.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const validRegistry = `checkers:
  privacy:
    runtime:
      kind: oci
      engine: [docker]
      image: example.invalid/privacy@sha256:1111111111111111111111111111111111111111111111111111111111111111
      platform: linux/amd64
    run: ["privacy-checker", "run", "--projection", "{projection}", "--config", "{config}", "--out", "{out}"]
    verify: ["privacy-checker", "verify", "--trace", "{out}"]
    timeout: "45s"
  invariants:
    runtime:
      kind: oci
      engine: [podman]
      image: example.invalid/invariants@sha256:2222222222222222222222222222222222222222222222222222222222222222
      platform: linux/amd64
    run: ["inv-checker", "{manifest}", "{design}", "{out}"]
`

func TestLoadRegistryValidAndResolve(t *testing.T) {
	reg, err := LoadRegistry(writeRegistry(t, validRegistry))
	if err != nil {
		t.Fatal(err)
	}

	got := reg.IDs()
	if len(got) != 2 || got[0] != "invariants" || got[1] != "privacy" {
		t.Fatalf("IDs not sorted/complete: %v", got)
	}

	privacy, ok := reg.Resolve("privacy")
	if !ok {
		t.Fatal("privacy should resolve")
	}
	if privacy.Timeout != 45*time.Second {
		t.Fatalf("explicit timeout not parsed: %v", privacy.Timeout)
	}
	if len(privacy.Verify) == 0 {
		t.Fatal("privacy verify command not carried")
	}
	if privacy.Runtime.Platform != "linux/amd64" {
		t.Fatalf("runtime platform = %q, want linux/amd64", privacy.Runtime.Platform)
	}

	if _, ok := reg.Resolve("nonexistent"); ok {
		t.Fatal("a missing id must not resolve")
	}
}

func TestLoadRegistryDefaultTimeout(t *testing.T) {
	reg, err := LoadRegistry(writeRegistry(t, validRegistry))
	if err != nil {
		t.Fatal(err)
	}
	inv, ok := reg.Resolve("invariants")
	if !ok {
		t.Fatal("invariants should resolve")
	}
	if inv.Timeout != DefaultCheckerTimeout {
		t.Fatalf("absent timeout should default to %v, got %v", DefaultCheckerTimeout, inv.Timeout)
	}
	if len(inv.Verify) != 0 {
		t.Fatalf("absent verify should be empty, got %v", inv.Verify)
	}
}

func TestTokenSubstitutionReplacesAll(t *testing.T) {
	args := []string{"tool", "--projection", "{projection}", "--config", "{config}", "--manifest", "{manifest}", "--out", "{out}", "--design", "{design}"}
	tok := Tokens{
		Projection: "/d/proj.json",
		Config:     "/tmp/cfg.json",
		Manifest:   "/d/checkers/x.checker.yaml",
		Out:        "/tmp/out.json",
		Design:     "/d",
	}
	got := tok.Substitute(args)
	want := []string{"tool", "--projection", "/d/proj.json", "--config", "/tmp/cfg.json", "--manifest", "/d/checkers/x.checker.yaml", "--out", "/tmp/out.json", "--design", "/d"}
	if len(got) != len(want) {
		t.Fatalf("length mismatch: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg %d = %q, want %q", i, got[i], want[i])
		}
	}
	// The source args must be untouched (Substitute returns a fresh slice).
	if args[2] != "{projection}" {
		t.Fatal("Substitute mutated the input args")
	}
}

func TestTokenSubstitutionMultipleTokensPerArg(t *testing.T) {
	tok := Tokens{Projection: "P", Out: "O"}
	got := tok.Substitute([]string{"{projection}:{out}"})
	if got[0] != "P:O" {
		t.Fatalf("multiple tokens in one arg not both replaced: %q", got[0])
	}
}

func TestLoadRegistryMalformedYAMLIsError(t *testing.T) {
	if _, err := LoadRegistry(writeRegistry(t, "checkers: [this is: not, a: map\n")); err == nil {
		t.Fatal("malformed YAML must be an error")
	}
}

func TestLoadRegistryEmptyRunIsError(t *testing.T) {
	body := `checkers:
  broken:
    verify: ["something"]
`
	if _, err := LoadRegistry(writeRegistry(t, body)); err == nil {
		t.Fatal("an entry with an empty run command must be an error")
	}
}

func TestLoadRegistryInvalidTimeoutIsError(t *testing.T) {
	body := `checkers:
  bad:
    run: ["x"]
    timeout: "not-a-duration"
`
	if _, err := LoadRegistry(writeRegistry(t, body)); err == nil {
		t.Fatal("an invalid timeout string must be an error")
	}
}

func TestLoadRegistryMissingFileIsError(t *testing.T) {
	if _, err := LoadRegistry(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("a missing registry file must be an error")
	}
}

func TestLoadRegistryRejectsOpenOrAmbiguousYAML(t *testing.T) {
	for name, body := range map[string]string{
		"unknown field":       "checkers:\n  x:\n    run: [x]\n    timout: 1s\n",
		"duplicate field":     "checkers:\n  x:\n    run: [x]\n    run: [y]\n",
		"non scalar key":      "checkers:\n  ? [x]\n  : {run: [x]}\n",
		"multiple documents":  "checkers: {x: {run: [x]}}\n---\ncheckers: {y: {run: [y]}}\n",
		"case alias ids":      "checkers:\n  Privacy: {run: [x]}\n  privacy: {run: [y]}\n",
		"nonpositive timeout": "checkers: {x: {run: [x], timeout: 0s}}\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadRegistry(writeRegistry(t, body)); err == nil {
				t.Fatal("open or ambiguous registry was accepted")
			}
		})
	}
}

func TestLoadRegistryRequiresImmutableOCIRuntime(t *testing.T) {
	digest := strings.Repeat("1", 64)
	for name, runtimeBlock := range map[string]string{
		"missing runtime":       "",
		"ambient runtime":       "    runtime: {kind: host, engine: [docker], image: example.invalid/x@sha256:" + digest + "}\n",
		"missing engine":        "    runtime: {kind: oci, image: example.invalid/x@sha256:" + digest + "}\n",
		"tag only image":        "    runtime: {kind: oci, engine: [docker], image: example.invalid/x:latest}\n",
		"uppercase digest":      "    runtime: {kind: oci, engine: [docker], image: example.invalid/x@sha256:" + strings.Repeat("A", 64) + "}\n",
		"empty input source":    "    runtime: {kind: oci, engine: [docker], image: example.invalid/x@sha256:" + digest + ", inputs: [{source: '', mount: rules.dl}]}\n",
		"absolute input source": "    runtime: {kind: oci, engine: [docker], image: example.invalid/x@sha256:" + digest + ", inputs: [{source: /tmp/rules.dl, mount: rules.dl}]}\n",
		"nonportable mount":     "    runtime: {kind: oci, engine: [docker], image: example.invalid/x@sha256:" + digest + ", inputs: [{source: rules.dl, mount: ../rules.dl}]}\n",
		"case alias mounts":     "    runtime: {kind: oci, engine: [docker], image: example.invalid/x@sha256:" + digest + ", inputs: [{source: a, mount: Rules.dl}, {source: b, mount: rules.dl}]}\n",
		"unknown runtime key":   "    runtime: {kind: oci, engine: [docker], image: example.invalid/x@sha256:" + digest + ", network: host}\n",
	} {
		t.Run(name, func(t *testing.T) {
			body := "checkers:\n  x:\n" + runtimeBlock + "    run: [checker, '{out}']\n"
			if _, err := LoadRegistry(writeRegistry(t, body)); err == nil {
				t.Fatalf("unsafe runtime was accepted:\n%s", body)
			}
		})
	}
}

func TestLoadRegistryRequiresClosedOCIPlatform(t *testing.T) {
	digest := strings.Repeat("1", 64)
	for name, platform := range map[string]string{
		"missing":       "",
		"host inferred": "native",
		"wrong OS":      "darwin/amd64",
		"wrong case":    "linux/AMD64",
		"open variant":  "linux/arm64/v8",
	} {
		t.Run(name, func(t *testing.T) {
			body := "checkers:\n  x:\n    runtime: {kind: oci, engine: [docker], image: example.invalid/x@sha256:" + digest
			if platform != "" {
				body += ", platform: " + platform
			}
			body += "}\n    run: [checker, '{out}']\n"
			_, err := LoadRegistry(writeRegistry(t, body))
			if err == nil || !strings.Contains(err.Error(), "runtime.platform must be one of") {
				t.Fatalf("platform %q diagnostic = %v", platform, err)
			}
		})
	}
}

func TestRuntimeClosureDigestBindsCompleteDeclaredClosure(t *testing.T) {
	image := "sha256:" + strings.Repeat("1", 64)
	inputs := map[string]string{
		"adapter.py": "sha256:" + strings.Repeat("2", 64),
		"rules.dl":   "sha256:" + strings.Repeat("3", 64),
	}
	base, err := RuntimeClosureDigest(image, "linux/amd64", []string{"python3", "/checker/adapter.py", "{out}"}, []string{"python3", "--version"}, inputs)
	if err != nil {
		t.Fatal(err)
	}
	reordered, err := RuntimeClosureDigest(image, "linux/amd64", []string{"python3", "/checker/adapter.py", "{out}"}, []string{"python3", "--version"}, map[string]string{
		"rules.dl":   inputs["rules.dl"],
		"adapter.py": inputs["adapter.py"],
	})
	if err != nil || reordered != base {
		t.Fatalf("map insertion order changed closure: got %s (%v), want %s", reordered, err, base)
	}
	if _, err := RuntimeClosureDigest(image, "", []string{"python3"}, nil, nil); err == nil {
		t.Fatal("missing OCI platform was accepted by the closure digest")
	}
	mutations := []struct {
		name     string
		image    string
		platform string
		run      []string
		verify   []string
		inputs   map[string]string
	}{
		{"image", "sha256:" + strings.Repeat("4", 64), "linux/amd64", []string{"python3", "/checker/adapter.py", "{out}"}, []string{"python3", "--version"}, inputs},
		{"platform", image, "linux/arm64", []string{"python3", "/checker/adapter.py", "{out}"}, []string{"python3", "--version"}, inputs},
		{"run argv", image, "linux/amd64", []string{"python3", "/checker/other.py", "{out}"}, []string{"python3", "--version"}, inputs},
		{"verify argv", image, "linux/amd64", []string{"python3", "/checker/adapter.py", "{out}"}, []string{"python3", "-VV"}, inputs},
		{"missing input", image, "linux/amd64", []string{"python3", "/checker/adapter.py", "{out}"}, []string{"python3", "--version"}, map[string]string{"adapter.py": inputs["adapter.py"]}},
		{"extra input", image, "linux/amd64", []string{"python3", "/checker/adapter.py", "{out}"}, []string{"python3", "--version"}, map[string]string{"adapter.py": inputs["adapter.py"], "rules.dl": inputs["rules.dl"], "extra": "sha256:" + strings.Repeat("5", 64)}},
		{"input bytes", image, "linux/amd64", []string{"python3", "/checker/adapter.py", "{out}"}, []string{"python3", "--version"}, map[string]string{"adapter.py": "sha256:" + strings.Repeat("6", 64), "rules.dl": inputs["rules.dl"]}},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			got, err := RuntimeClosureDigest(mutation.image, mutation.platform, mutation.run, mutation.verify, mutation.inputs)
			if err != nil {
				t.Fatal(err)
			}
			if got == base {
				t.Fatalf("%s mutation did not change runtime closure", mutation.name)
			}
		})
	}
}
