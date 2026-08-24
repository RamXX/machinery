package gates

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestExModules(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{"alias line", "alias Foo.Bar\n", []string{"Foo.Bar"}},
		{"import/use/require lines", "import Foo.Bar\nuse Foo.Web, :controller\nrequire Foo.Log\n", []string{"Foo.Bar", "Foo.Web", "Foo.Log"}},
		{"qualified call", "def url, do: TixxWeb.Endpoint.url()\n", []string{"TixxWeb.Endpoint"}},
		{"qualified bang and question calls", "x = Foo.Bar.load!(1)\ny = Foo.Baz.ok?(x)\n", []string{"Foo.Bar", "Foo.Baz"}},
		{"struct literal", "%Foo.Bar{id: 1}\n", []string{"Foo.Bar"}},
		{"function capture", "Enum.map(xs, &Foo.Bar.baz/1)\n", []string{"Foo.Bar"}},
		{"comment mention excluded", "# see Foo.Bar.baz( for details\nx = 1\n", nil},
		{"interpolation is not a comment", "s = \"#{Foo.Bar.baz(1)}\"\n", []string{"Foo.Bar"}},
		{"moduledoc heredoc excluded", "@moduledoc \"\"\"\nCalls Foo.Bar.baz( and builds %Foo.Bar{}.\n\"\"\"\nx = 1\n", nil},
		{"doc heredoc single quotes excluded", "@doc '''\nCalls Foo.Bar.baz(.\n'''\nx = 1\n", nil},
		{"single-segment stdlib call excluded", "Enum.map(xs, fn x -> x end)\n", nil},
		// Plain strings are not stripped (documented on exModules): a
		// qualified call inside a string literal still counts.
		{"string containing a qualified call is included", "s = \"run Foo.Bar.baz(\"\n", []string{"Foo.Bar"}},
		{"mixed file dedupes", "defmodule A.B do\n  alias Foo.Bar\n  @doc \"\"\"\n  Ignore Zed.Doc.only(\n  \"\"\"\n  def f, do: Foo.Bar.baz() # Comment.Only.here(\n  def g, do: %Foo.Bar{} |> Baz.Qux.run!(&Foo.Bar.h/1)\nend\n", []string{"Foo.Bar", "Baz.Qux"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := exModules(c.src); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("exModules(%q) = %v, want %v", c.src, got, c.want)
			}
		})
	}
}

// writeElixirFixture lays down two Elixir boundaries (core, web) resolved by
// modules: prefixes; core references the web layer only through the given
// body (no alias line).
func writeElixirFixture(t *testing.T, rules, coreBody string) (design, impl string) {
	t.Helper()
	root := t.TempDir()
	design = filepath.Join(root, "design")
	impl = filepath.Join(root, "impl")
	if rules == "" {
		rules = "  allow: []\n  deny: []"
	}
	arch := "# Architecture\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: core\n    code: [\"lib/core/**\"]\n    modules: [\"Core\"]\n" +
		"  - id: web\n    code: [\"lib/web/**\"]\n    modules: [\"Web\"]\n" +
		"dependency_rules:\n" + rules + "\n```\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	mustWrite(t, filepath.Join(impl, "lib", "core", "descriptor.ex"),
		"defmodule Core.Descriptor do\n  @moduledoc \"\"\"\n  Builds the descriptor from Web.Endpoint.url().\n  \"\"\"\n"+coreBody+"end\n")
	mustWrite(t, filepath.Join(impl, "lib", "web", "endpoint.ex"), "defmodule Web.Endpoint do\n  def url, do: \"http://x\"\nend\n")
	return design, impl
}

func TestG4ElixirQualifiedCallIsAnEdge(t *testing.T) {
	body := "  def build, do: Web.Endpoint.url()\n"
	design, impl := writeElixirFixture(t, "", body)
	g := CheckImports(design, impl)
	if !hasErr(g, "undeclared cross-boundary edge core -> web") {
		t.Fatalf("a fully-qualified call must produce the core -> web edge, got %v", g.Errs)
	}

	design, impl = writeElixirFixture(t, "  deny: [\"core -> web\"]", body)
	if g := CheckImports(design, impl); !hasErr(g, "core -> web") {
		t.Fatalf("a denied qualified call must fail G4, got %v", g.Errs)
	}

	design, impl = writeElixirFixture(t, "  baseline: [\"core -> web\"]", body)
	if err := WriteRatchet(design, &Ratchet{Date: "2026-08-23", Edges: map[string][]string{"core -> web": {"lib/core/descriptor.ex"}}}); err != nil {
		t.Fatal(err)
	}
	g = CheckImports(design, impl)
	if len(g.Errs) != 0 {
		t.Fatalf("a baseline rule must tolerate the qualified-call edge, got %v", g.Errs)
	}
	if g.Counts["baselined edges"] != 1 {
		t.Fatalf("baselined edges = %d, want 1: %+v", g.Counts["baselined edges"], g.Counts)
	}
}

func TestG4ElixirDocMentionIsNotAnEdge(t *testing.T) {
	// the alias keeps the gate from tripping its "nothing checked" guard
	design, impl := writeElixirFixture(t, "  deny: [\"core -> web\"]", "  alias Core.Helper\n  def build, do: Core.Helper.ok()\n")
	g := CheckImports(design, impl)
	if hasErr(g, "core -> web") {
		t.Fatalf("a @moduledoc mention alone must not create an edge, got %v", g.Errs)
	}
	if g.Counts["imports resolved"] == 0 {
		t.Fatalf("the intra-boundary reference must resolve: %+v", g.Counts)
	}
}

// A repo with several Go modules and no root go.mod: every module's imports
// resolve through the module owning its directory, so cross-module edges are
// judged and module-internal orphans are still reported.
func TestG4MultiModuleGoResolvesEveryModule(t *testing.T) {
	root := t.TempDir()
	design, impl := filepath.Join(root, "design"), filepath.Join(root, "impl")
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: cli\n    code: [\"cmd/cli/**\"]\n  - id: mcp\n    code: [\"cmd/mcp/internal/**\"]\n" +
		"ignore: [\"scratch/**\"]\ndependency_rules:\n  allow: []\n  deny: [\"cli -> mcp\"]\n```\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	mustWrite(t, filepath.Join(impl, "cmd", "cli", "go.mod"), "module example.com/cli\n")
	mustWrite(t, filepath.Join(impl, "cmd", "mcp", "go.mod"), "module example.com/mcp\n")
	mustWrite(t, filepath.Join(impl, "scratch", "go.mod"), "module example.com/scratch\n")
	mustWrite(t, filepath.Join(impl, "cmd", "cli", "main.go"),
		"package main\n\nimport (\n\t\"example.com/cli/internal/x\"\n\t\"example.com/mcp/internal/server\"\n\t\"example.com/mcp/gone\"\n\t\"example.com/scratch/y\"\n)\n")
	mustWrite(t, filepath.Join(impl, "cmd", "cli", "internal", "x", "x.go"), "package x\n")
	mustWrite(t, filepath.Join(impl, "cmd", "mcp", "internal", "server", "s.go"), "package server\n")
	scan := &importScan{}
	g := checkImports(design, impl, scan)
	if !hasErr(g, "cli -> mcp") {
		t.Fatalf("the cross-module deny must be enforced, got %v", g.Errs)
	}
	if g.Counts["imports resolved"] != 2 {
		t.Fatalf("intra-module and cross-module imports must both resolve: %+v", g.Counts)
	}
	if hasErr(g, "example.com/scratch/y") {
		t.Fatalf("a go.mod under an ignored glob must not be discovered, got %v", g.Errs)
	}
	if len(scan.OrphanRefs["example.com/mcp/gone"]) != 1 {
		t.Fatalf("a module-internal import with no boundary is an orphan: %+v", scan.OrphanRefs)
	}
}

// Nested modules: the longest module path owns the import.
func TestGoModuleForPrefersLongestPath(t *testing.T) {
	mods := []goModule{{path: "example.com/m/sub", dir: "sub"}, {path: "example.com/m", dir: "."}}
	if _, rel, ok := goModuleFor(mods, "example.com/m/sub/pkg"); !ok || rel != "sub/pkg" {
		t.Fatalf("got %q %v, want sub/pkg", rel, ok)
	}
	if _, rel, ok := goModuleFor(mods, "example.com/m/other"); !ok || rel != "other" {
		t.Fatalf("got %q %v, want other", rel, ok)
	}
	if _, _, ok := goModuleFor(mods, "example.com/mx"); ok {
		t.Fatal("a module path is a path prefix, not a string prefix")
	}
}

// Snapshots stamp a full date; older month-only snapshots stay readable and
// age from the first of the month.
func TestRatchetAgeNoteBothFormats(t *testing.T) {
	now := time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC)
	if got := ratchetAgeNote("2026-08-23", now); got != "ratchet snapshot 2026-08-23, 0 day(s) old" {
		t.Fatalf("full date: %q", got)
	}
	if got := ratchetAgeNote("2026-08", now); got != "ratchet snapshot 2026-08, 22 day(s) old" {
		t.Fatalf("month form: %q", got)
	}
}
