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
		// #{...} interpolation is blanked with its surrounding string: a
		// module referenced only there is a documented, accepted miss.
		{"interpolation dropped with its string", "s = \"#{Foo.Bar.baz(1)}\"\n", nil},
		{"moduledoc heredoc excluded", "@moduledoc \"\"\"\nCalls Foo.Bar.baz( and builds %Foo.Bar{}.\n\"\"\"\nx = 1\n", nil},
		{"doc heredoc single quotes excluded", "@doc '''\nCalls Foo.Bar.baz(.\n'''\nx = 1\n", nil},
		{"single-segment stdlib call excluded", "Enum.map(xs, fn x -> x end)\n", nil},
		// Strings, charlists, and sigils are stripped (documented on
		// exModules and exStripStrings): a qualified call spelled inside
		// one never counts.
		{"string spelling a qualified call excluded", "s = \"run Foo.Bar.baz(\"\n", nil},
		{"tuple string excluded (TIXX config.ex false-positive shape)", "cfg = {:vcs_github_module,\n  \"GitHub VCS adapter module (default: Tixx.Integrations.Github.Req.new())\"}\n", nil},
		{"plain heredoc excluded", "s = \"\"\"\nFoo.Bar.baz(1)\n\"\"\"\nx = 1\n", nil},
		{"charlist excluded", "c = 'Foo.Bar.baz('\n", nil},
		{"sigil with pipe delimiter excluded", "s = ~s|Foo.Bar.baz(|\n", nil},
		{"uppercase sigil heredoc excluded", "s = ~S\"\"\"\nFoo.Bar.baz(1)\n\"\"\"\nx = 1\n", nil},
		{"regex sigil excluded", "ok = Regex.match?(~r/Foo.Bar.baz(/, s)\n", nil},
		{"escaped quote then real call on the same line", "s = \"say \\\" it\"; Foo.Bar.baz(1)\n", []string{"Foo.Bar"}},
		{"hash inside a string does not truncate the line", "s = \"x # y\"; Foo.Bar.baz(1)\n", []string{"Foo.Bar"}},
		{"lowercase sigil honors escaped closer", "s = ~s(Foo.Bar.baz\\() ; x = 1\n", nil},
		{"uppercase sigil does not escape its closer", "s = ~S(a\\); Foo.Bar.baz(1)\n", []string{"Foo.Bar"}},
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

// --- extractor parity: Rust (H1 inline references, H2 string/comment strips) ---

func TestRustImports(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{"use line", "use foo::bar::Baz;\n", []string{"foo"}},
		{"use crate line", "use crate::repo::Item;\n", []string{"src/repo/Item"}},
		// H1: idiomatic use-free inline qualified references must count.
		{"inline crate call", "fn f() { crate::repo::save(); }\n", []string{"src/repo/save"}},
		{"inline foreign crate call", "fn f() { other_crate::api::call(1); }\n", []string{"other_crate"}},
		{"inline struct literal", "fn f() { let c = other_crate::config::Config { x: 1 }; }\n", []string{"other_crate"}},
		{"inline turbofish", "fn f() { helper::parse::<u32>(\"1\"); }\n", []string{"helper"}},
		// Deliberate exclusions, mirroring the Elixir single-segment rule:
		// std/core/alloc never map to a boundary, self/super need module
		// context a regex does not have.
		{"std core alloc excluded", "fn f() { std::mem::swap(&mut a, &mut b); core::hint::spin_loop(); alloc::vec::Vec::new(); }\n", nil},
		{"self and super excluded", "fn f() { self::helper::run(); super::util::go(); }\n", nil},
		// H2: the use-line scan was line-anchored on RAW text; a use line
		// inside a block comment was a real pre-existing false positive.
		{"use inside block comment excluded", "/*\nuse foo::bar;\n*/\nfn f() {}\n", nil},
		{"inline in line comment excluded", "// crate::repo::save(\nfn f() {}\n", nil},
		{"inline in string excluded", "fn f() { let s = \"crate::repo::save(\"; }\n", nil},
		{"inline in raw string excluded", "fn f() { let s = r#\"other_crate::api::call(\"#; }\n", nil},
		{"inline in raw byte string excluded", "fn f() { let s = br##\"helper::run(\"##; }\n", nil},
		{"nested block comment excluded", "/* outer /* use foo::bar; */ still comment crate::repo::save( */\nfn f() {}\n", nil},
		{"lifetime does not open a string", "fn f<'a>(x: &'a str) { crate::repo::save(x); }\n", []string{"src/repo/save"}},
		{"char literal with quote stripped", "fn f() { let c = '\"'; crate::repo::save(); }\n", []string{"src/repo/save"}},
		{"escaped char literal stripped", "fn f() { let c = '\\''; crate::repo::save(); }\n", []string{"src/repo/save"}},
		{"use and inline dedupe", "use foo::bar;\nfn f() { foo::bar::run(); }\n", []string{"foo"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := rustImports(c.src); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("rustImports(%q) = %v, want %v", c.src, got, c.want)
			}
		})
	}
}

// --- extractor parity: Python (H3 docstring false positives) ---

func TestPyImports(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{"plain import", "import foo.bar\n", []string{"foo/bar"}},
		{"from import", "from foo.bar import baz\n", []string{"foo/bar"}},
		{"relative from", "from .sib import x\n", []string{"pkg/sib"}},
		{"comma imports", "import a as x, b.c\n", []string{"a", "b/c"}},
		{"indented import kept", "def f():\n    import inner\n", []string{"inner"}},
		// H3: docstrings are line-start text; a line-anchored regex matched
		// an import spelled inside one.
		{"docstring import excluded", "\"\"\"\nimport secretdep\nfrom other import x\n\"\"\"\nimport real\n", []string{"real"}},
		{"single-quote docstring excluded", "'''\nimport hidden\n'''\n", nil},
		{"prefixed docstring excluded", "x = r\"\"\"\nimport hidden\n\"\"\"\n", nil},
		{"fstring docstring excluded", "x = f\"\"\"\nimport hidden\n\"\"\"\n", nil},
		// Pin: a # comment could never match the line-anchored regex.
		{"hash comment pin", "# import commented\nimport real\n", []string{"real"}},
		{"triple quote inside single-line string does not open", "s = '\"\"\"'\nimport real\n", []string{"real"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pyImports(c.src, "pkg/mod.py"); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("pyImports(%q) = %v, want %v", c.src, got, c.want)
			}
		})
	}
}

// --- extractor parity: TS/JS (H4 string/comment false positives) ---

func TestTsImports(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{"static import", "import x from 'y';\n", []string{"y"}},
		{"side-effect import", "import './setup';\n", []string{"src/setup"}},
		{"require call", "const l = require('lodash');\n", []string{"lodash"}},
		{"dynamic import", "const m = await import('mod');\n", []string{"mod"}},
		{"export from", "export { a } from './a';\n", []string{"src/a"}},
		{"import after comment still counts", "// setup\nimport x from 'y';\n", []string{"y"}},
		// H4: tsImportRe is not line-anchored; strings and comments matched.
		{"string spelling require excluded", "const s = \"require('lodash')\";\n", nil},
		{"single-quoted string spelling import excluded", "const s = 'import(\"x\")';\n", nil},
		{"line comment excluded", "// import('x')\nconst a = 1;\n", nil},
		{"block comment excluded", "/* require('y') */\nconst a = 1;\n", nil},
		// Documented choice: template literals are blanked whole, so an
		// embedded ${require('y')} does not produce an edge.
		{"template-embedded require dropped", "const s = `${require('y')}`;\n", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := tsImports(c.src, "src/app.ts"); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("tsImports(%q) = %v, want %v", c.src, got, c.want)
			}
		})
	}
}

// --- extractor parity: Rust inline edges reach the gate (H1 end to end) ---

func TestG4RustInlineQualifiedCallIsAnEdge(t *testing.T) {
	root := t.TempDir()
	design, impl := filepath.Join(root, "design"), filepath.Join(root, "impl")
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: core\n    code: [\"src/core/**\"]\n  - id: repo\n    code: [\"src/repo/**\"]\n" +
		"dependency_rules:\n  allow: []\n  deny: [\"core -> repo\"]\n```\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	// no use line: the reference is a use-free fully-qualified call
	mustWrite(t, filepath.Join(impl, "src", "core", "svc.rs"),
		"pub fn run() { crate::repo::store::save(1); let x = crate::core::util::id(); }\n")
	mustWrite(t, filepath.Join(impl, "src", "core", "util.rs"), "pub fn id() -> u32 { 1 }\n")
	mustWrite(t, filepath.Join(impl, "src", "repo", "store.rs"), "pub fn save(_x: u32) {}\n")
	g := CheckImports(design, impl)
	if !hasErr(g, "core -> repo") {
		t.Fatalf("a use-free qualified call must produce the denied core -> repo edge, got %v", g.Errs)
	}
}

// --- extractor parity: TS workspace package names resolve to boundaries (H5) ---

func TestG4TsWorkspacePackagesResolve(t *testing.T) {
	root := t.TempDir()
	design, impl := filepath.Join(root, "design"), filepath.Join(root, "impl")
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: pkga\n    code: [\"packages/a/**\"]\n  - id: pkgb\n    code: [\"packages/b/**\"]\n" +
		"ignore: [\"scratch/**\"]\ndependency_rules:\n  allow: []\n  deny: [\"pkga -> pkgb\"]\n```\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	mustWrite(t, filepath.Join(impl, "packages", "a", "package.json"), "{\"name\": \"@org/a\"}\n")
	mustWrite(t, filepath.Join(impl, "packages", "b", "package.json"), "{\"name\": \"@org/b\"}\n")
	mustWrite(t, filepath.Join(impl, "packages", "c", "package.json"), "{\"name\": \"@org/c\"}\n")
	mustWrite(t, filepath.Join(impl, "scratch", "package.json"), "{\"name\": \"@org/scratch\"}\n")
	mustWrite(t, filepath.Join(impl, "node_modules", "leftpad", "package.json"), "{\"name\": \"leftpad\"}\n")
	mustWrite(t, filepath.Join(impl, "packages", "a", "src", "index.ts"),
		"import { b } from '@org/b';\nimport { u } from '@org/b/src/util';\n"+
			"import { g } from '@org/c/lib';\nimport lp from 'leftpad';\nimport sc from '@org/scratch';\n")
	mustWrite(t, filepath.Join(impl, "packages", "b", "src", "index.ts"), "export const b = 1;\n")
	mustWrite(t, filepath.Join(impl, "packages", "b", "src", "util.ts"), "export const u = 1;\n")
	scan := &importScan{}
	g := checkImports(design, impl, scan)
	if !hasErr(g, "pkga -> pkgb") {
		t.Fatalf("a package-name import must resolve to the owning boundary and trip the deny, got %v", g.Errs)
	}
	if len(scan.OrphanRefs["@org/c/lib"]) != 1 {
		t.Fatalf("an import into a discovered package outside every boundary is an orphan: %+v", scan.OrphanRefs)
	}
	if hasErr(g, "leftpad") {
		t.Fatalf("a node_modules package.json must not be discovered, got %v", g.Errs)
	}
	if hasErr(g, "scratch") {
		t.Fatalf("a package.json under an ignored glob must not be discovered, got %v", g.Errs)
	}
}
