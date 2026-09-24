package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/RamXX/machinery/internal/runtimeclosure"
)

// pinSite is one place outside internal/runtimeclosure that spells a
// runtime version: every match of pattern in file must capture exactly want.
// A site that stops matching at all also fails, so a reworded file cannot
// silently drop out of the check.
type pinSite struct {
	file    string
	pattern string
	want    string
}

func pinSites() []pinSite {
	const (
		otp    = runtimeclosure.RequiredOTPVersion
		erts   = runtimeclosure.RequiredErtsVersion
		elixir = runtimeclosure.RequiredElixirVersion
		node   = runtimeclosure.RequiredNodeRelease
		ts     = runtimeclosure.RequiredTypeScriptVersion
		python = runtimeclosure.RequiredPythonVersion
		java   = runtimeclosure.PinnedJavaRuntimeVersion
		golang = runtimeclosure.RequiredGoVersion
	)
	action := ".github/actions/assurance-runtimes/action.yml"
	docker := "scripts/ci-linux.dockerfile"
	return []pinSite{
		{"go.mod", `(?m)^go (\S+)$`, golang},
		{"README.md", "`go.mod` pins (\\S+?)\\)", golang},
		{action, `go-version: "([^"]+)"`, golang},
		{action, `node-version: (\S+)`, node},
		{action, `python-version: (\S+)`, python},
		{action, `elixir-version: (\S+)`, elixir},
		{action, `otp-version: (\S+)`, otp},
		{action, `typescript@(\S+)`, ts},
		{action, `Install Go (\S+),`, golang},
		{action, `Node (\S+) with TypeScript`, node},
		{action, `on OTP (\S+)`, otp},
		{docker, `golang:(\d[^-\s]*)-`, golang},
		{docker, `node:(\d[^-\s]*)-`, node},
		{docker, `python:(\d[^-\s]*)-`, python},
		{docker, `hexpm/elixir:(\d[^-\s]*)-erlang`, elixir},
		{docker, `-erlang-(\d[^-\s]*)-`, otp},
		{docker, `typescript@(\S+)`, ts},
		{docker, `Node (\d\S*) with`, node},
		{docker, `on OTP (\d[\d.]*\d)`, otp},
		{"CONTRIBUTING.md", `Node (\d\S*) with TypeScript`, node},
		{"CONTRIBUTING.md", `on OTP\s+(\d[\d.]*\d)`, otp},
		{"CONTRIBUTING.md", `Go (\d[\d.]*\d),\s+Node`, golang},
		{"scripts/preflight.sh", `pins Node (\S+) \+`, node},
		{"scripts/preflight.sh", `ERTS (\d[\d.]*\d)`, erts},
		{"docs/test-assurance-contract.md", `Node (\S+) with TypeScript compiler`, node},
		{"docs/test-assurance-contract.md", `Erlang/OTP (\S+) \(ERTS`, otp},
		{"docs/test-assurance-contract.md", `\(ERTS (\d[\d.]*\d)\)`, erts},
		{"testdata/integration-lanes/assurance.CONTRACT.md", `\nNode (\S+) with TypeScript`, node},
		{"testdata/integration-lanes/assurance.CONTRACT.md", `ERTS (\d[\d.]*\d)`, erts},
		{"testdata/integration-lanes/assurance.CONTRACT.md", `erts-(\d[\d.]*\d)`, erts},
		{"testdata/integration-lanes/assurance.CONTRACT.md", "`node\\s+--version` \\(v(\\S+)\\)", node},
		{"internal/tdd/adapters/assets/elixir/README.md", `OTP (\d[\d.]*\d) \(ERTS`, otp},
		{"internal/tdd/adapters/assets/elixir/README.md", `\(ERTS (\d[\d.]*\d)\)`, erts},
		{"README.md", `Java\]\(https://adoptium.net/\) (\S+)\*\*`, java},
		{"README.md", `same Java (\S+) runtime`, java},
		{"skills/machinery/tools/README.md", `Temurin Java (\S+)`, java},
		{"skills/machinery/references/c4-standalone.md", `Temurin Java (\S+)`, java},
		{"testdata/integration-lanes/CONTRACT.md", `\n(\d[\d.]*\+\d+)-LTS`, java},
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// TestPinSitesAgree reads every file that spells a runtime version and
// holds each spelling to the one runtimeclosure pin, so a bump that misses
// a site fails here instead of in CI's runtime install.
func TestPinSitesAgree(t *testing.T) {
	root := repoRoot(t)
	for _, site := range pinSites() {
		body, err := os.ReadFile(filepath.Join(root, site.file))
		if err != nil {
			t.Errorf("%s: %v", site.file, err)
			continue
		}
		matches := regexp.MustCompile(site.pattern).FindAllStringSubmatch(string(body), -1)
		if len(matches) == 0 {
			t.Errorf("%s: pattern %q no longer matches; update the pin site list", site.file, site.pattern)
			continue
		}
		for _, match := range matches {
			if match[1] != site.want {
				t.Errorf("%s: %q spells %s, pin is %s", site.file, match[0], match[1], site.want)
			}
		}
	}
}

// TestFrozenPinFilesAgree decodes the JSON pin files the integration lane
// freezes and holds every version in them to the runtimeclosure pins.
func TestFrozenPinFilesAgree(t *testing.T) {
	root := repoRoot(t)
	var assurance struct {
		Adapters map[string]struct {
			Runtime    string `json:"runtime"`
			Version    string `json:"version"`
			TypeScript string `json:"typescript"`
			OTP        string `json:"otp"`
			Erts       string `json:"erts"`
			Mix        string `json:"mix"`
		} `json:"adapters"`
	}
	decode(t, filepath.Join(root, "testdata", "integration-lanes", "assurance-runtime-pins.json"), &assurance)
	type adapter = struct{ version, typescript, otp, erts, mix string }
	want := map[string]adapter{
		"go-testing/v1":           {runtimeclosure.RequiredGoVersion, "", "", "", ""},
		"node-test-typescript/v1": {runtimeclosure.RequiredNodeRelease, runtimeclosure.RequiredTypeScriptVersion, "", "", ""},
		"python-unittest/v1":      {runtimeclosure.RequiredPythonVersion, "", "", "", ""},
		"elixir-exunit/v1":        {runtimeclosure.RequiredElixirVersion, "", runtimeclosure.RequiredOTPMajor, runtimeclosure.RequiredErtsVersion, runtimeclosure.RequiredElixirVersion},
	}
	for id, w := range want {
		got, ok := assurance.Adapters[id]
		if !ok || (adapter{got.Version, got.TypeScript, got.OTP, got.Erts, got.Mix}) != w {
			t.Errorf("assurance-runtime-pins.json %s = %+v, want %+v", id, got, w)
		}
	}
	var lane struct {
		Java struct {
			Version string `json:"version"`
		} `json:"java"`
	}
	decode(t, filepath.Join(root, "testdata", "integration-lanes", "runtime-pins.json"), &lane)
	if lane.Java.Version != runtimeclosure.PinnedJavaRuntimeVersion {
		t.Errorf("runtime-pins.json java %s, pin is %s", lane.Java.Version, runtimeclosure.PinnedJavaRuntimeVersion)
	}
}

func decode(t *testing.T, path string, v any) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}
