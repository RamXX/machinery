package gates

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/dirscan"
	"github.com/RamXX/machinery/internal/ir"
)

const implementationDirectoryMaxEntries = 100_000
const implementationDirectoryMaxDepth = 64

// --- imports (G4-import) ---

// walkSourceFiles collects source files under one canonical source root. One
// aggregate entry ceiling and one portable depth ceiling cover the complete
// traversal, including entries that are inspected and then ignored.
// Symlink entries below that root are rejected: following them makes G4/Gt
// depend on mutable bytes outside the governed source inventory, while
// skipping a broken link silently makes the scan incomplete.
//
// Directories whose root-relative path matches a contract ignore glob are
// pruned BEFORE descent (symlinked or not): everything under them is
// excluded post-walk anyway, and descending a huge ignored tree (a
// node_modules tree wastes the walk at best and used to abort it at worst.
// A symlink is skipped only when its own walk path is ignored; a non-ignored
// alias is rejected even when its target would have been ignored.
//
// A directory that cannot be resolved or read is recorded in warns and its
// SIBLINGS are still walked: a partial walk must be loud (the caller reports
// every skipped subtree), never a silent truncation of everything sorting
// after the failing directory. Only a failure on root itself is fatal.
//
// pruned lists the root-relative directories the ignore globs pruned, so the
// caller can keep the skipped volume visible in its counts.
func walkSourceFiles(root string, ignore []string) (files, pruned, warns []string, err error) {
	inventory, pruned, warns, err := walkSourceFilesBounded(root, ignore, implementationDirectoryMaxEntries, implementationDirectoryMaxDepth)
	if inventory == nil {
		return nil, pruned, warns, err
	}
	files = inventory.Paths()
	return files, pruned, warns, errors.Join(err, inventory.Close())
}

type sourceFileInventory struct {
	root        *os.Root
	displayRoot string
	files       []string
	listed      map[string]bool
	witnesses   map[string]rootDirectoryWitness
	closed      bool
}

func (inventory *sourceFileInventory) Files() []string {
	return append([]string(nil), inventory.files...)
}

func (inventory *sourceFileInventory) Paths() []string {
	paths := make([]string, 0, len(inventory.files))
	for _, rel := range inventory.files {
		paths = append(paths, filepath.Join(inventory.displayRoot, rel))
	}
	return paths
}

func (inventory *sourceFileInventory) ReadFile(rel string) ([]byte, error) {
	if inventory == nil || inventory.closed {
		return nil, fmt.Errorf("source inventory authority is closed")
	}
	if !inventory.listed[rel] {
		return nil, fmt.Errorf("source path %s was not in the governed inventory", rel)
	}
	return readRootRegularFile(inventory.root, rel)
}

func (inventory *sourceFileInventory) Close() (retErr error) {
	if inventory == nil || inventory.closed {
		return nil
	}
	inventory.closed = true
	var dirs []string
	for rel := range inventory.witnesses {
		dirs = append(dirs, rel)
	}
	sort.Strings(dirs)
	for _, rel := range dirs {
		retErr = errors.Join(retErr, revalidateRootDirectory(inventory.root, rel, inventory.witnesses[rel]))
	}
	rootWitness := inventory.witnesses["."]
	publicInfo, pathErr := os.Lstat(inventory.displayRoot)
	if pathErr != nil || !sameInventoryInfo(rootWitness.info, publicInfo) {
		retErr = errors.Join(retErr, pathErr, fmt.Errorf("source inventory root %s changed identity during traversal", inventory.displayRoot))
	} else {
		publicDir, openErr := os.Open(inventory.displayRoot)
		if openErr != nil {
			retErr = errors.Join(retErr, openErr)
		} else {
			opened, statErr := publicDir.Stat()
			changeID, changeErr := dirscan.ChangeID(publicDir, opened)
			closeErr := publicDir.Close()
			if statErr != nil || changeErr != nil || !sameInventoryInfo(rootWitness.info, opened) || changeID != rootWitness.changeID {
				retErr = errors.Join(retErr, statErr, changeErr, fmt.Errorf("source inventory root %s changed during traversal", inventory.displayRoot))
			}
			retErr = errors.Join(retErr, closeErr)
		}
	}
	retErr = errors.Join(retErr, inventory.root.Close())
	return retErr
}

func walkSourceFilesBounded(root string, ignore []string, maxEntries, maxDepth int) (inventory *sourceFileInventory, pruned, warns []string, err error) {
	if maxEntries <= 0 {
		return nil, nil, nil, fmt.Errorf("source inventory entry limit must be positive")
	}
	if maxDepth < 0 {
		return nil, nil, nil, fmt.Errorf("source inventory depth limit must be non-negative")
	}
	rootAuthority, displayRoot, err := openRealRoot(root)
	if err != nil {
		return nil, nil, nil, err
	}
	inventory = &sourceFileInventory{
		root:        rootAuthority,
		displayRoot: displayRoot,
		listed:      map[string]bool{},
		witnesses:   map[string]rootDirectoryWitness{},
	}
	openedInventory := inventory
	valid := false
	defer func() {
		if !valid {
			err = errors.Join(err, openedInventory.root.Close())
			inventory = nil
		}
	}()
	entriesSeen := 0
	var walk func(relDir string, depth int, isRoot bool) error
	walk = func(relDir string, depth int, isRoot bool) error {
		displayDir := displayRoot
		if relDir != "." {
			displayDir = filepath.Join(displayRoot, relDir)
		}
		fail := func(e error) error {
			if isRoot {
				return e
			}
			warns = append(warns, displayDir+": "+e.Error())
			return nil
		}
		entries, witness, readErr := readRootDirectory(rootAuthority, relDir, maxEntries-entriesSeen)
		if readErr != nil {
			return fail(readErr)
		}
		inventory.witnesses[relDir] = witness
		entriesSeen += len(entries)
		for _, e := range entries {
			rel := e.Name()
			if relDir != "." {
				rel = filepath.Join(relDir, e.Name())
			}
			p := filepath.Join(displayRoot, rel)
			if depth >= maxDepth {
				return fmt.Errorf("source inventory exceeds %d-level depth limit at %s", maxDepth, p)
			}
			info, statErr := rootAuthority.Lstat(rel)
			if statErr != nil {
				warns = append(warns, p+": "+statErr.Error())
				continue
			}
			if info.Mode()&os.ModeSymlink != 0 {
				if sourcePathIgnored(rel, ignore) {
					continue
				}
				warns = append(warns, p+": symlink entries are rejected from the governed source inventory")
				continue
			}
			if info.IsDir() {
				if dirIgnored(rel, ignore) {
					pruned = append(pruned, rel)
					continue
				}
				if walkErr := walk(rel, depth+1, false); walkErr != nil {
					return walkErr
				}
				continue
			}
			if !info.Mode().IsRegular() {
				warns = append(warns, p+": non-regular entries are rejected from the governed source inventory")
				continue
			}
			if _, ok := langExts[filepath.Ext(rel)]; ok {
				inventory.files = append(inventory.files, rel)
				inventory.listed[rel] = true
			} else if isTestFile(e.Name()) {
				// *.test.mjs / *.test.cjs: test files in extensions langExts
				// never maps for import parsing; Gt still needs them walked
				inventory.files = append(inventory.files, rel)
				inventory.listed[rel] = true
			}
		}
		return revalidateRootDirectory(rootAuthority, relDir, witness)
	}
	if walkErr := walk(".", 0, true); walkErr != nil {
		return nil, pruned, warns, walkErr
	}
	valid = true
	return inventory, pruned, warns, nil
}

func sourcePathIgnored(rel string, ignore []string) bool {
	for _, pattern := range ignore {
		if matchGlob(rel, pattern) || dirIgnored(rel, []string{pattern}) {
			return true
		}
	}
	return false
}

// dirIgnored reports whether a directory (root-relative rel) is excluded by
// the contract's ignore globs, so the walk can prune it before descending.
// Semantics match the post-walk file filters (matchGlob): the glob
// "mock-ui/**" prunes the directory "mock-ui" (its static prefix), while
// "lib/tixx/**" does NOT prune "lib". File-level globs ("mix.exs") never
// match a directory prefix and stay a post-walk concern.
func dirIgnored(rel string, ignore []string) bool {
	for _, ig := range ignore {
		if matchGlob(rel, ig) || matchGlob(rel+"/", ig) {
			return true
		}
	}
	return false
}

// testFilePatterns, isTestFile, and rustSplitTests are the ONE test-file
// classifier, shared by G4 (which SKIPS test files, per its documented
// semantics) and Gt (which scans exactly the files G4 skips): the two gates
// can never disagree about what a test file is.
var testFilePatterns = []string{"*_test.go", "*_test.py", "test_*.py", "*.test.ts", "*.test.tsx",
	"*.test.js", "*.test.jsx", "*.test.mjs", "*.test.cjs", "*_test.exs", "*_spec.rb"}

func isTestFile(rel string) bool {
	base := filepath.Base(rel)
	for _, p := range testFilePatterns {
		if ok, _ := filepath.Match(p, base); ok {
			return true
		}
	}
	// a .rs file is test-only iff it lives under a tests/ or benches/
	// directory (Rust integration tests and benchmarks), at any depth;
	// every other .rs file is production code and rustSplitTests carves its
	// #[cfg(test)] spans out for Gt
	if strings.HasSuffix(base, ".rs") {
		for _, seg := range strings.Split(filepath.ToSlash(filepath.Dir(rel)), "/") {
			if seg == "tests" || seg == "benches" {
				return true
			}
		}
	}
	return false
}

// cfgTestAttr marks the Rust unit-test idiom: a #[cfg(test)] item (usually
// `mod tests { ... }`) inside a production .rs file.
const cfgTestAttr = "#[cfg(test)]"

// rustSplitTests carves a .rs file into its production text and its
// #[cfg(test)] item spans. A .rs file is test-only iff it lives under tests/
// or benches/ (isTestFile); every OTHER .rs file is production code: G4 scans
// the production portion (a wholesale skip made production imports invisible,
// NG-1) and Gt's corpus receives ONLY the cfg(test) spans, where Rust unit
// tests actually live (production text once wholesale-covered oracles, NG-7).
// The production text keeps its line structure: span bytes become spaces.
func rustSplitTests(text string) (string, []string) {
	var spans [][2]int
	for i := 0; i < len(text); {
		j := strings.Index(text[i:], cfgTestAttr)
		if j < 0 {
			break
		}
		start := i + j
		end := rustItemEnd(text, start+len(cfgTestAttr))
		spans = append(spans, [2]int{start, end})
		i = end
	}
	if len(spans) == 0 {
		return text, nil
	}
	prod := []byte(text)
	var tests []string
	for _, sp := range spans {
		tests = append(tests, text[sp[0]:sp[1]])
		for k := sp[0]; k < sp[1]; k++ {
			if prod[k] != '\n' {
				prod[k] = ' '
			}
		}
	}
	return string(prod), tests
}

// rustItemEnd returns the end offset of the item that follows a #[cfg(test)]
// attribute: the matching close brace of its first block, the terminating
// semicolon of a braceless item, or end of text. Brace counting is
// lint-grade (a brace inside a string literal would miscount), matching the
// regex-grade parsing of every other language here.
func rustItemEnd(text string, from int) int {
	// rustStripStrings is length preserving: braces in strings/comments become
	// spaces while structural braces stay in place. Counting on that lexical
	// view prevents one test literal from swallowing production code to EOF.
	text = rustStripStrings(text)
	depth := 0
	for i := from; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth <= 0 {
				return i + 1
			}
		case ';':
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(text)
}

func matchGlob(rel, pattern string) bool {
	pattern = strings.TrimSuffix(pattern, "/")
	if ok, _ := filepath.Match(pattern, rel); ok {
		return true
	}
	static := strings.ReplaceAll(pattern, "/**", "")
	static = strings.ReplaceAll(static, "/*", "")
	static = strings.TrimSuffix(static, "/")
	return rel == static || strings.HasPrefix(rel, static+"/")
}

// boundaryMatch returns the unique most-specific owner and, on an equal
// specificity tie across different boundaries, the sorted ambiguous owners.
func boundaryMatch(rel string, pkgmap [][2]string) (string, []string) {
	best := -1
	owners := map[string]bool{}
	for _, pm := range pkgmap {
		pattern, bid := pm[0], pm[1]
		if matchGlob(rel, pattern) {
			static := strings.ReplaceAll(pattern, "/**", "")
			static = strings.ReplaceAll(static, "/*", "")
			if len(static) > best {
				best = len(static)
				owners = map[string]bool{bid: true}
			} else if len(static) == best {
				owners[bid] = true
			}
		}
	}
	var bids []string
	for bid := range owners {
		bids = append(bids, bid)
	}
	sort.Strings(bids)
	if len(bids) == 1 {
		return bids[0], nil
	}
	if len(bids) > 1 {
		return "", bids
	}
	return "", nil
}

// goModule is one go.mod under impl: its module path and the directory
// (impl-relative, "." for the root) whose packages it names.
type goModule struct {
	path, dir string
}

var goModuleLineRe = regexp.MustCompile(`(?m)^module\s+(\S+)`)

// goModules discovers every go.mod under impl (a repo may hold several
// modules and no root one), skipping dot, vendor, and contract-ignored
// directories. Longer module paths sort first so a nested module wins over
// an enclosing one whose path is its prefix. A subtree that cannot be read
// is recorded in warns and the walk continues with its siblings; only a
// failure on impl itself is fatal.
func goModules(impl string, ignore []string) ([]goModule, error) {
	var mods []goModule
	err := walkTreeDirBounded(impl, implementationDirectoryMaxEntries, implementationDirectoryMaxDepth, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(impl, path)
		if d.Type()&os.ModeSymlink != 0 {
			if sourcePathIgnored(rel, ignore) {
				return nil
			}
			return fmt.Errorf("%s: symlink entries are rejected from Go module discovery", path)
		}
		if d.IsDir() {
			if rel == "." {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" {
				return filepath.SkipDir
			}
			if dirIgnored(rel, ignore) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		data, readErr := readRegularFile(path)
		if readErr != nil {
			return readErr
		}
		if m := goModuleLineRe.FindSubmatch(data); m != nil {
			mods = append(mods, goModule{path: string(m[1]), dir: filepath.Dir(rel)})
		}
		return nil
	})
	sort.SliceStable(mods, func(i, j int) bool { return len(mods[i].path) > len(mods[j].path) })
	return mods, err
}

// goModuleFor returns the module owning an import path and the impl-relative
// package directory, or ok=false when no discovered module names it.
func goModuleFor(mods []goModule, ref string) (mod goModule, rel string, ok bool) {
	for _, m := range mods {
		if ref == m.path || strings.HasPrefix(ref, m.path+"/") {
			return m, filepath.Join(m.dir, strings.TrimLeft(ref[len(m.path):], "/")), true
		}
	}
	return goModule{}, "", false
}

// tsPackage is one named package.json under impl: the workspace package name
// and the directory (impl-relative, "." for the root) it owns.
type tsPackage struct {
	name, dir string
}

// tsPackages discovers every named package.json under impl (a JS/TS
// workspace holds one per package), mirroring goModules: dot, vendor,
// node_modules, and contract-ignored directories are skipped, longer names
// sort first so a package whose name prefixes another's never shadows it,
// and an unreadable subtree is recorded in warns while the walk continues
// with its siblings; only a failure on impl itself is fatal. A package.json
// without a name (or unparseable) is skipped silently: it names nothing an
// import specifier could reference.
func tsPackages(impl string, ignore []string) ([]tsPackage, error) {
	var pkgs []tsPackage
	err := walkTreeDirBounded(impl, implementationDirectoryMaxEntries, implementationDirectoryMaxDepth, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(impl, path)
		if d.Type()&os.ModeSymlink != 0 {
			if sourcePathIgnored(rel, ignore) {
				return nil
			}
			return fmt.Errorf("%s: symlink entries are rejected from package discovery", path)
		}
		if d.IsDir() {
			if rel == "." {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			if dirIgnored(rel, ignore) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "package.json" {
			return nil
		}
		data, readErr := readRegularFile(path)
		if readErr != nil {
			return readErr
		}
		var pj struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(data, &pj) == nil && pj.Name != "" {
			pkgs = append(pkgs, tsPackage{name: pj.Name, dir: filepath.Dir(rel)})
		}
		return nil
	})
	sort.SliceStable(pkgs, func(i, j int) bool { return len(pkgs[i].name) > len(pkgs[j].name) })
	return pkgs, err
}

// tsPackageFor returns the impl-relative path an import specifier resolves
// to through the discovered workspace packages, or ok=false when no package
// names it. A bare-name import resolves to the package directory itself
// (entry-point resolution through main/exports is not modeled: the
// directory is enough to name the owning boundary).
func tsPackageFor(pkgs []tsPackage, ref string) (rel string, ok bool) {
	for _, p := range pkgs {
		if ref == p.name {
			return p.dir, true
		}
		if strings.HasPrefix(ref, p.name+"/") {
			return filepath.Join(p.dir, ref[len(p.name)+1:]), true
		}
	}
	return "", false
}

var (
	goBlockImportRe   = regexp.MustCompile(`(?ms)^import\s*\((.*?)\)`)
	goLineImportRe    = regexp.MustCompile(`(?m)^import\s+(?:[\w.]+\s+)?"([^"]+)"`)
	goBlockLineRe     = regexp.MustCompile(`(?:^|\s)(?:[\w.]+\s+)?"([^"]+)"`)
	pyImportRe        = regexp.MustCompile(`(?m)^\s*import\s+([\w.]+(?:\s+as\s+\w+)?(?:\s*,\s*[\w.]+(?:\s+as\s+\w+)?)*)`)
	pyFromRe          = regexp.MustCompile(`(?m)^\s*from\s+([\w.]+)\s+import\b`)
	pyDynamicImportRe = regexp.MustCompile(`(?:importlib\.import_module|__import__)\s*\(\s*['"]([^'"]+)['"]`)
	// from/import/dynamic import()/require() forms
	tsImportRe = regexp.MustCompile(`(?:from|import\s*\(|import|require\()\s*['"]([^'"]+)['"]`)
	exModRe    = regexp.MustCompile(`(?m)^\s*(?:alias|import|use|require)\s+([A-Z][\w.]*)`)
	// Fully-qualified inline references, which idiomatic Elixir uses without
	// an alias line: a qualified call with parens (Mod.Sub.fun(, Mod.Sub.fun!(),
	// a struct literal (%Mod.Sub{), and a function capture (&Mod.Sub.fun/1).
	// The module path must have at least two segments: a single-segment
	// reference (Enum.map() is almost always stdlib and never maps to a
	// boundary modules: prefix that carries a dot, so it is excluded on
	// purpose rather than resolved and dropped.
	exQualCallRe = regexp.MustCompile(`([A-Z][A-Za-z0-9_]*(?:\.[A-Z][A-Za-z0-9_]*)+)\.[a-z_][A-Za-z0-9_]*[!?]?\(`)
	exStructRe   = regexp.MustCompile(`%([A-Z][A-Za-z0-9_]*(?:\.[A-Z][A-Za-z0-9_]*)+)\{`)
	exCaptureRe  = regexp.MustCompile(`&([A-Z][A-Za-z0-9_]*(?:\.[A-Z][A-Za-z0-9_]*)+)\.`)
	rustUseRe    = regexp.MustCompile(`(?m)^\s*(?:pub(?:\([^)]*\))?\s+)?use\s+([^;]+);`)
	// Use-free fully-qualified Rust references in expression position: a
	// call path::to::item(, a macro call path::to::name!(, a struct literal
	// path::To::Type {, and a turbofish path::to::item::<T>. The head
	// segment is lowercase (crate and module names), the path has at least
	// one :: (mirroring the Elixir multi-segment rule), and the leading
	// guard keeps a match from starting mid-path. A path in pure type,
	// trait-bound, or bare-const position (let x: a::B = a::C;) is not
	// matched: telling that apart from noise needs a parser, a documented
	// limitation.
	rustQualRe    = regexp.MustCompile(`(?:^|[^:\w])([a-z_][a-z0-9_]*(?:::[A-Za-z_][A-Za-z0-9_]*)+)\s*(?:!?\(|\{|::<)`)
	exDefmoduleRe = regexp.MustCompile(`(?m)^\s*defmodule\s+([A-Z][\w.]*)`)
)

func goImports(text string) []string {
	var out []string
	for _, blk := range goBlockImportRe.FindAllStringSubmatch(text, -1) {
		for _, m := range goBlockLineRe.FindAllStringSubmatch(blk[1], -1) {
			out = append(out, m[1])
		}
	}
	for _, m := range goLineImportRe.FindAllStringSubmatch(text, -1) {
		out = append(out, m[1])
	}
	return out
}

// pyImports returns the imports of a Python file: line-anchored import and
// from lines, scanned on pyStripStrings output so a docstring that spells
// an import at line start never counts. Ordinary single-line strings could
// never host a line-anchored import (their opening quote precedes the
// keyword on the same line), and a # comment can never precede a
// line-anchored match either; both are stripped anyway for scan stability.
// Literal dynamic imports (importlib.import_module, __import__) are retained;
// computed arguments remain intentionally unresolved.
func pyImports(text, rel string) []string {
	text = pyStripStrings(text)
	var out []string
	for _, m := range pyImportRe.FindAllStringSubmatch(text, -1) {
		// "import a as x, b.c" imports every listed module, not just the first
		for _, part := range strings.Split(m[1], ",") {
			mod := strings.Fields(strings.TrimSpace(part))[0]
			out = append(out, strings.ReplaceAll(mod, ".", "/"))
		}
	}
	for _, m := range pyFromRe.FindAllStringSubmatch(text, -1) {
		mod := m[1]
		if strings.HasPrefix(mod, ".") {
			base := filepath.Dir(rel)
			for i := 1; i < len(mod)-len(strings.TrimLeft(mod, ".")); i++ {
				base = filepath.Dir(base)
			}
			mod = filepath.Join(base, strings.ReplaceAll(strings.TrimLeft(mod, "."), ".", "/"))
			mod = strings.Trim(mod, "/")
			out = append(out, mod)
		} else {
			out = append(out, strings.ReplaceAll(mod, ".", "/"))
		}
	}
	for _, m := range pyDynamicImportRe.FindAllStringSubmatch(text, -1) {
		out = append(out, strings.ReplaceAll(m[1], ".", "/"))
	}
	return out
}

// tsImports returns the module specifiers of a TS/JS file: static
// from/import forms plus dynamic import() and require() calls, scanned on
// tsStripStrings output (tsImportRe is not line-anchored, so unstripped
// comments and strings used to produce edges). Specifier strings of real
// import forms survive the strip verbatim (keepImportArg); a specifier
// spelled inside any other string, a comment, or a template literal never
// counts.
func tsImports(text, rel string) []string {
	text = tsStripStrings(text)
	var out []string
	for _, m := range tsImportRe.FindAllStringSubmatch(text, -1) {
		spec := m[1]
		if strings.HasPrefix(spec, ".") {
			joined, _ := filepath.Abs(filepath.Join(filepath.Dir(rel), spec))
			_ = joined
			out = append(out, filepath.Clean(filepath.Join(filepath.Dir(rel), spec)))
		} else {
			out = append(out, spec)
		}
	}
	return out
}

// exModules returns every module a file references, deduplicated in first-seen
// order: the alias/import/use/require lines plus fully-qualified inline
// references (see exQualCallRe). String-like literals (heredocs, strings,
// charlists, sigils) and line comments are stripped before the inline scan
// (see exStripStrings), so neither a prose mention nor a string that spells
// a module call becomes an edge. A module referenced only inside #{...}
// interpolation is dropped with its surrounding string: an accepted miss.
func exModules(text string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(mod string) {
		if !seen[mod] {
			seen[mod] = true
			out = append(out, mod)
		}
	}
	for _, m := range exModRe.FindAllStringSubmatch(text, -1) {
		add(m[1])
	}
	code := exStripStrings(text)
	for _, re := range []*regexp.Regexp{exQualCallRe, exStructRe, exCaptureRe} {
		for _, m := range re.FindAllStringSubmatch(code, -1) {
			add(m[1])
		}
	}
	return out
}

// exStripStrings blanks the contents of Elixir string-like literals and
// drops line comments in a single pass, so the inline-reference regexes
// never match inside either. Delimiters are kept and inner bytes become
// spaces (newlines survive), so line and offset positions stay stable.
// Handled forms:
//
//   - heredocs """...""" and ”'...”' (doc attributes included)
//   - double-quoted strings and single-quoted charlists, where a backslash
//     escapes the next byte (so an escaped quote does not terminate)
//   - sigils: ~ plus any ASCII letter plus a delimiter pair from (), [],
//     {}, <>, "", ”, ||, //, including the heredoc forms ~x"""...""" and
//     ~x”'...”'; lowercase sigils honor backslash escapes, uppercase do
//     not (they have no escapes in Elixir)
//
// Deliberate simplifications: the scan runs to the first unescaped closing
// delimiter with no nesting tracking (nesting is not needed to keep the
// reference regexes out of string contents), #{...} interpolation is blanked
// with its surrounding string (a module referenced only there is an accepted
// miss), and ? char literals are not special-cased (a bare ?" reads as a
// string opener). Comments are handled after strings by construction: a #
// reached in code state starts a comment that runs to end of line, so a #
// inside a string never truncates the line.
func exStripStrings(text string) string {
	src := []byte(text)
	out := make([]byte, 0, len(src))
	n := len(src)

	closerFor := map[byte]byte{
		'(': ')', '[': ']', '{': '}', '<': '>',
		'"': '"', '\'': '\'', '|': '|', '/': '/',
	}
	isLetter := func(b byte) bool {
		return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
	}
	blank := func(b byte) byte {
		if b == '\n' {
			return '\n'
		}
		return ' '
	}
	// blankUntil emits spaces (newlines kept) from i to the first unescaped
	// occurrence of closer, emits the closer verbatim, and returns the
	// position after it. An unterminated literal blanks to EOF.
	blankUntil := func(i int, closer string, escapes bool) int {
		for i < n {
			if escapes && src[i] == '\\' {
				out = append(out, ' ')
				i++
				if i < n {
					out = append(out, blank(src[i]))
					i++
				}
				continue
			}
			if src[i] == closer[0] && i+len(closer) <= n && string(src[i:i+len(closer)]) == closer {
				out = append(out, closer...)
				return i + len(closer)
			}
			out = append(out, blank(src[i]))
			i++
		}
		return i
	}

	for i := 0; i < n; {
		c := src[i]
		switch {
		case c == '#':
			// code state: comment to end of line (the newline itself is
			// emitted by the default case on the next iteration)
			for i < n && src[i] != '\n' {
				i++
			}
		case c == '~' && i+2 < n && isLetter(src[i+1]):
			closer, ok := closerFor[src[i+2]]
			if !ok {
				out = append(out, c)
				i++
				continue
			}
			escapes := src[i+1] >= 'a' && src[i+1] <= 'z'
			width := 1
			if (src[i+2] == '"' || src[i+2] == '\'') && i+4 < n && src[i+3] == src[i+2] && src[i+4] == src[i+2] {
				width = 3
			}
			out = append(out, src[i:i+2+width]...)
			i = blankUntil(i+2+width, strings.Repeat(string(closer), width), escapes)
		case c == '"' || c == '\'':
			width := 1
			if i+2 < n && src[i+1] == c && src[i+2] == c {
				width = 3
			}
			out = append(out, src[i:i+width]...)
			i = blankUntil(i+width, strings.Repeat(string(c), width), true)
		default:
			out = append(out, c)
			i++
		}
	}
	return string(out)
}

// rustImports returns the crate-level references of a .rs file: use lines
// plus use-free fully-qualified inline references (rustQualRe), both scanned
// on rustStripStrings output (the line-anchored use scan on raw text used to
// match a use line inside a block comment), deduplicated in first-seen
// order. crate:: paths map to src/-relative paths as before; the std, core,
// and alloc heads are excluded on purpose (they never map to a boundary,
// like the Elixir single-segment rule), and self/super are excluded because
// resolving them needs module context a regex does not have.
func rustImports(text string) []string {
	code := rustStripStrings(text)
	var out []string
	seen := map[string]bool{}
	add := func(path string) {
		var ref string
		switch strings.SplitN(path, "::", 2)[0] {
		case "std", "core", "alloc", "self", "super":
			return
		case "crate":
			rest := strings.TrimPrefix(path, "crate::")
			if rest == path { // a bare `crate` reference crosses nothing
				return
			}
			ref = "src/" + strings.ReplaceAll(rest, "::", "/")
		default:
			ref = strings.SplitN(path, "::", 2)[0]
		}
		if !seen[ref] {
			seen[ref] = true
			out = append(out, ref)
		}
	}
	for _, m := range rustUseRe.FindAllStringSubmatch(code, -1) {
		for _, path := range rustUsePaths(m[1]) {
			add(path)
		}
	}
	for _, m := range rustQualRe.FindAllStringSubmatch(code, -1) {
		add(m[1])
	}
	return out
}

// rustUsePaths expands the dependency-bearing heads of a Rust use tree. It
// intentionally does not model imported item aliases: boundary ownership is
// decided by the path before `as`.
func rustUsePaths(expr string) []string {
	expr = strings.TrimSpace(strings.SplitN(expr, " as ", 2)[0])
	open := strings.Index(expr, "{")
	close := strings.LastIndex(expr, "}")
	if open < 0 || close < open {
		return []string{strings.TrimSuffix(strings.TrimSpace(expr), "::")}
	}
	prefix := strings.TrimSuffix(strings.TrimSpace(expr[:open]), "::")
	var out []string
	for _, branch := range strings.Split(expr[open+1:close], ",") {
		branch = strings.TrimSpace(strings.SplitN(branch, " as ", 2)[0])
		if branch == "" || branch == "self" {
			if prefix != "" {
				out = append(out, prefix)
			}
			continue
		}
		if prefix != "" {
			out = append(out, prefix+"::"+branch)
		} else {
			out = append(out, branch)
		}
	}
	return out
}

// rustStripStrings blanks Rust comments and string-like literals in a single
// pass so the reference regexes never match inside either. Delimiters are
// kept and inner bytes become spaces (newlines survive), so line positions
// stay stable. Handled forms:
//
//   - line comments // to end of line, and block comments /* */, which NEST
//     per Rust rules
//   - string literals "..." and byte strings b"...", where a backslash
//     escapes the next byte
//   - raw strings r"..." and br"...", with any hash count (r#"..."#,
//     r##"..."##, ...); raw strings have no escapes
//   - char literals 'x' and '\x' (escape forms included); a lone ' is a
//     lifetime and passes through untouched
//
// Lint-grade like the other languages here: no macro or attribute awareness.
func rustStripStrings(text string) string {
	src := []byte(text)
	out := make([]byte, 0, len(src))
	n := len(src)
	blank := func(b byte) byte {
		if b == '\n' {
			return '\n'
		}
		return ' '
	}
	identByte := func(b byte) bool {
		return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
	}
	// blankQuoted blanks from i to the closing quote, honoring backslash
	// escapes when escapes is true; returns the position after the closer.
	blankQuoted := func(i int, quote byte, escapes bool) int {
		for i < n {
			if escapes && src[i] == '\\' {
				out = append(out, ' ')
				i++
				if i < n {
					out = append(out, blank(src[i]))
					i++
				}
				continue
			}
			if src[i] == quote {
				out = append(out, quote)
				return i + 1
			}
			out = append(out, blank(src[i]))
			i++
		}
		return i
	}
	for i := 0; i < n; {
		c := src[i]
		switch {
		case c == '/' && i+1 < n && src[i+1] == '/':
			for i < n && src[i] != '\n' {
				out = append(out, ' ')
				i++
			}
		case c == '/' && i+1 < n && src[i+1] == '*':
			depth := 1
			out = append(out, ' ', ' ')
			i += 2
			for i < n && depth > 0 {
				if src[i] == '/' && i+1 < n && src[i+1] == '*' {
					depth++
					out = append(out, ' ', ' ')
					i += 2
					continue
				}
				if src[i] == '*' && i+1 < n && src[i+1] == '/' {
					depth--
					out = append(out, ' ', ' ')
					i += 2
					continue
				}
				out = append(out, blank(src[i]))
				i++
			}
		case (c == 'r' || c == 'b') && (i == 0 || !identByte(src[i-1])):
			j := i + 1
			if c == 'b' && j < n && src[j] == 'r' {
				j++
			}
			if c == 'b' && j == i+1 { // b"...": byte string, escapes apply
				if j < n && src[j] == '"' {
					out = append(out, src[i:j+1]...)
					i = blankQuoted(j+1, '"', true)
					continue
				}
				out = append(out, c)
				i++
				continue
			}
			hashes := 0
			for j+hashes < n && src[j+hashes] == '#' {
				hashes++
			}
			if j+hashes < n && src[j+hashes] == '"' { // raw string, no escapes
				open := j + hashes + 1
				out = append(out, src[i:open]...)
				closer := `"` + strings.Repeat("#", hashes)
				k := open
				for k < n {
					if src[k] == '"' && k+len(closer) <= n && string(src[k:k+len(closer)]) == closer {
						out = append(out, closer...)
						k += len(closer)
						break
					}
					out = append(out, blank(src[k]))
					k++
				}
				i = k
				continue
			}
			out = append(out, c)
			i++
		case c == '"':
			out = append(out, c)
			i = blankQuoted(i+1, '"', true)
		case c == '\'':
			// a char literal iff '\... or 'x'; otherwise a lifetime
			if i+1 < n && src[i+1] == '\\' {
				out = append(out, c)
				i = blankQuoted(i+1, '\'', true)
			} else if i+2 < n && src[i+2] == '\'' && src[i+1] != '\'' {
				out = append(out, '\'', ' ', '\'')
				i += 3
			} else {
				out = append(out, c)
				i++
			}
		default:
			out = append(out, c)
			i++
		}
	}
	return string(out)
}

// pyStripStrings blanks Python triple-quoted strings, single-line strings,
// and # comments in a single pass. Delimiters are kept and inner bytes
// become spaces (newlines survive). Only a triple-quoted string can host a
// line-anchored import (a single-line string keeps its opening quote on the
// import's line), but singles are blanked too so a quote inside one never
// opens a phantom triple. A backslash escapes the next byte in every form:
// Python raw strings still treat a backslash-quote as non-terminating, so
// prefix letters (r, b, f, u, combinations) need no special handling and
// pass through as ordinary code bytes.
func pyStripStrings(text string) string {
	src := []byte(text)
	out := make([]byte, 0, len(src))
	n := len(src)
	blank := func(b byte) byte {
		if b == '\n' {
			return '\n'
		}
		return ' '
	}
	for i := 0; i < n; {
		c := src[i]
		switch c {
		case '#':
			for i < n && src[i] != '\n' {
				out = append(out, ' ')
				i++
			}
		case '"', '\'':
			keep := keepPyImportArg(out)
			width := 1
			if i+2 < n && src[i+1] == c && src[i+2] == c {
				width = 3
			}
			closer := strings.Repeat(string(c), width)
			out = append(out, src[i:i+width]...)
			i += width
			for i < n {
				if src[i] == '\\' {
					if keep {
						out = append(out, src[i])
					} else {
						out = append(out, ' ')
					}
					i++
					if i < n {
						if keep {
							out = append(out, src[i])
						} else {
							out = append(out, blank(src[i]))
						}
						i++
					}
					continue
				}
				if width == 1 && src[i] == '\n' {
					break // a single-line string cannot span lines; recover
				}
				if src[i] == c && i+width <= n && string(src[i:i+width]) == closer {
					out = append(out, closer...)
					i += width
					break
				}
				if keep {
					out = append(out, src[i])
				} else {
					out = append(out, blank(src[i]))
				}
				i++
			}
		default:
			out = append(out, c)
			i++
		}
	}
	return string(out)
}

func keepPyImportArg(out []byte) bool {
	s := strings.TrimSpace(string(out))
	return strings.HasSuffix(s, "importlib.import_module(") || strings.HasSuffix(s, "__import__(")
}

// tsStripStrings blanks TS/JS comments, string literals, and template
// literals in a single pass so tsImportRe (which is not line-anchored)
// never matches inside them. Delimiters are kept and inner bytes become
// spaces (newlines survive). A string that is the specifier of a real
// import form survives verbatim: when the code emitted so far ends with the
// from or import keyword, or with an import( / require( call opener, the
// string IS the reference (keepImportArg). Template literals are always
// blanked, ${...} included: a require inside one, or a require(`...`) call,
// produces no edge (documented choice). Regex literals are lexed separately
// so quotes and comment-like bytes inside them cannot desynchronize the scan.
func tsStripStrings(text string) string {
	src := []byte(text)
	out := make([]byte, 0, len(src))
	n := len(src)
	blank := func(b byte) byte {
		if b == '\n' {
			return '\n'
		}
		return ' '
	}
	for i := 0; i < n; {
		c := src[i]
		switch {
		case c == '/' && i+1 < n && src[i+1] == '/':
			for i < n && src[i] != '\n' {
				out = append(out, ' ')
				i++
			}
		case c == '/' && i+1 < n && src[i+1] == '*':
			out = append(out, ' ', ' ')
			i += 2
			for i < n {
				if src[i] == '*' && i+1 < n && src[i+1] == '/' {
					out = append(out, ' ', ' ')
					i += 2
					break
				}
				out = append(out, blank(src[i]))
				i++
			}
		case c == '/' && tsRegexStart(out):
			// Regex literals are lexical code, not comments. Blank their body so
			// quote bytes inside /.../ cannot open phantom JS strings.
			out = append(out, '/')
			i++
			inClass := false
			for i < n {
				if src[i] == '\\' {
					out = append(out, ' ')
					i++
					if i < n {
						out = append(out, blank(src[i]))
						i++
					}
					continue
				}
				if src[i] == '[' {
					inClass = true
				}
				if src[i] == ']' {
					inClass = false
				}
				if src[i] == '/' && !inClass {
					out = append(out, '/')
					i++
					for i < n && ((src[i] >= 'a' && src[i] <= 'z') || (src[i] >= 'A' && src[i] <= 'Z')) {
						out = append(out, src[i])
						i++
					}
					break
				}
				out = append(out, blank(src[i]))
				i++
			}
		case c == '\'' || c == '"' || c == '`':
			keep := c != '`' && keepImportArg(out)
			emit := func(b byte) {
				if keep {
					out = append(out, b)
				} else {
					out = append(out, blank(b))
				}
			}
			out = append(out, c)
			i++
			for i < n {
				if src[i] == '\\' {
					emit(src[i])
					i++
					if i < n {
						emit(src[i])
						i++
					}
					continue
				}
				if c != '`' && src[i] == '\n' {
					break // a quoted string cannot span lines; recover
				}
				if src[i] == c {
					out = append(out, c)
					i++
					break
				}
				emit(src[i])
				i++
			}
		default:
			out = append(out, c)
			i++
		}
	}
	return string(out)
}

func tsRegexStart(out []byte) bool {
	i := len(out) - 1
	for i >= 0 && (out[i] == ' ' || out[i] == '\t' || out[i] == '\r' || out[i] == '\n') {
		i--
	}
	if i < 0 {
		return true
	}
	return strings.ContainsRune("=([{,:;!?&|+-*%^~<>", rune(out[i]))
}

// keepImportArg reports whether the code emitted so far ends where an
// import specifier string begins: after the from or import keyword, or
// after the opening paren of an import(...) or require(...) call.
func keepImportArg(out []byte) bool {
	i := len(out)
	skipWS := func() {
		for i > 0 && (out[i-1] == ' ' || out[i-1] == '\t' || out[i-1] == '\n' || out[i-1] == '\r') {
			i--
		}
	}
	identByte := func(b byte) bool {
		return b == '_' || b == '$' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
	}
	word := func(w string) bool {
		if i < len(w) || string(out[i-len(w):i]) != w {
			return false
		}
		return i == len(w) || !identByte(out[i-len(w)-1])
	}
	skipWS()
	if i > 0 && out[i-1] == '(' {
		i--
		skipWS()
		return word("require") || word("import")
	}
	return word("from") || word("import")
}

// Per-language reference extraction (all lint-grade regex scans; none run
// a compiler):
//
//	go      goImports: import declarations only. Go has no import-free
//	        qualified reference form (every cross-package reference
//	        requires an import), so Go deliberately has no inline scan.
//	python  pyImports: line-anchored import/from lines plus literal dynamic
//	        imports on pyStripStrings output.
//	ts      tsImports: from/import/import()/require() specifiers on
//	        tsStripStrings output.
//	elixir  exModules: alias/import/use/require lines plus inline
//	        qualified references on exStripStrings output.
//	rust    rustImports: use lines plus inline qualified references on
//	        rustStripStrings output, production spans only
//	        (rustSplitTests).
var langExts = map[string]string{
	".go": "go", ".py": "python", ".ts": "ts", ".tsx": "ts", ".js": "ts",
	".jsx": "ts", ".ex": "elixir", ".exs": "elixir", ".rs": "rust",
}

// scanEdge is one observed cross-boundary edge with its offender files, as
// judged by the contract: "allowed", "denied", "undeclared", or "baselined".
type scanEdge struct {
	Src, Dst string
	Witness  string
	Files    []string // sorted
	Status   string
}

// importScan collects what `machinery baseline` needs from a G4 pass, so the
// generator and the gate share one discovery implementation and can never
// disagree about what the code contains.
type importScan struct {
	Edges         []scanEdge
	UnmappedFiles []string            // source files outside every boundary (rel)
	OrphanRefs    map[string][]string // module-internal import -> referencing files
	WalkWarns     []string            // subtrees the walk could not read (the scan is partial)
	Complete      bool                // the walk and judgment actually ran
}

// manifestDependencies closes the deterministically enumerable part of the
// external universe. It reads direct dependency declarations from supported
// language manifests; a literal import matching one of these names must bind
// to Architecture Contract externals rather than disappear as "not local".
func manifestDependencies(impl string, ignore []string) (map[string]bool, []string) {
	deps := map[string]bool{}
	var errs []string
	walkErr := walkTreeDirBounded(impl, implementationDirectoryMaxEntries, implementationDirectoryMaxDepth, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d == nil {
			return nil
		}
		rel, _ := filepath.Rel(impl, path)
		if d.Type()&os.ModeSymlink != 0 {
			if sourcePathIgnored(rel, ignore) {
				return nil
			}
			return fmt.Errorf("%s: symlink entries are rejected from dependency manifest discovery", path)
		}
		if d.IsDir() {
			if rel != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor" || d.Name() == "node_modules" || dirIgnored(rel, ignore)) {
				return filepath.SkipDir
			}
			return nil
		}
		body, readErr := readRegularFile(path)
		if readErr != nil {
			return readErr
		}
		switch d.Name() {
		case "package.json":
			found, parseErr := parsePackageManifest(body)
			if parseErr != nil {
				return fmt.Errorf("%s: invalid package.json: %w", path, parseErr)
			}
			for _, name := range found {
				deps[name] = true
			}
		case "go.mod":
			found, parseErr := parseGoModManifest(body)
			if parseErr != nil {
				return fmt.Errorf("%s: invalid go.mod: %w", path, parseErr)
			}
			for _, name := range found {
				deps[name] = true
			}
		case "Cargo.toml":
			found, parseErr := parseCargoManifest(body)
			if parseErr != nil {
				return fmt.Errorf("%s: invalid Cargo.toml: %w", path, parseErr)
			}
			for _, name := range found {
				deps[name] = true
			}
		case "requirements.txt", "requirements.in":
			found, parseErr := parseRequirementsManifest(body)
			if parseErr != nil {
				return fmt.Errorf("%s: invalid %s: %w", path, d.Name(), parseErr)
			}
			for _, name := range found {
				deps[name] = true
			}
		case "mix.exs":
			found, parseErr := parseMixManifest(body)
			if parseErr != nil {
				return fmt.Errorf("%s: invalid mix.exs: %w", path, parseErr)
			}
			for _, name := range found {
				deps[name] = true
			}
		}
		return nil
	})
	if walkErr != nil {
		errs = append(errs, walkErr.Error())
	}
	sort.Strings(errs)
	return deps, errs
}

func importMatchesManifest(ref string, deps map[string]bool) bool {
	for dep := range deps {
		if ref == dep || strings.HasPrefix(ref, dep+"/") || strings.HasPrefix(ref, dep+".") || strings.SplitN(ref, "::", 2)[0] == dep {
			return true
		}
	}
	return false
}

// CheckImports implements G4-import.
func CheckImports(design, impl string) *Gate {
	return checkImports(design, impl, nil)
}

// checkImports is CheckImports with an optional scan collector; scan may be
// nil (the plain gate) and collecting must never change the gate's findings.
func checkImports(design, impl string, scan *importScan) *Gate {
	g := NewGate("G4-import  code respects the contract")
	g.startOrder()
	if fi, err := os.Stat(impl); err != nil || !fi.IsDir() {
		g.Errs = append(g.Errs, fmt.Sprintf("--impl %s is not a directory", ir.Repr(impl)))
		return g
	}
	cg := NewGate("_")
	c := loadContract(design, filepath.Join(design, "ARCHITECTURE.md"), cg)
	if c == nil {
		g.Errs = append(g.Errs, cg.Errs...)
		if len(cg.Errs) == 0 {
			g.Errs = append(g.Errs, "no contract to check against")
		}
		return g
	}
	co := c.AsObject()
	var boundaries []*ir.Value
	for _, b := range objSlice(co.Get2("boundaries")) {
		if bo := b.AsObject(); bo != nil && bo.GetString("id") != "" {
			boundaries = append(boundaries, b)
		}
	}
	var externals []*ir.Value
	for _, x := range objSlice(co.Get2("externals")) {
		if xo := x.AsObject(); xo != nil && xo.GetString("id") != "" {
			externals = append(externals, x)
		}
	}
	var ignore []string
	for _, ig := range objSlice(co.Get2("ignore")) {
		ignore = append(ignore, ig.AsString())
	}
	var pkgmap [][2]string
	exposes := map[string][]string{}
	for _, b := range boundaries {
		bo := b.AsObject()
		for _, code := range objSlice(bo.Get2("code")) {
			pkgmap = append(pkgmap, [2]string{code.AsString(), bo.GetString("id")})
		}
		if exp := bo.Get2("exposes"); exp != nil {
			var es []string
			for _, e := range exp.AsArray() {
				es = append(es, e.AsString())
			}
			exposes[bo.GetString("id")] = es
		}
	}
	var extByPrefix, extModules, boundModules [][2]string
	for _, x := range externals {
		xo := x.AsObject()
		for _, p := range objSlice(xo.Get2("imports")) {
			extByPrefix = append(extByPrefix, [2]string{p.AsString(), xo.GetString("id")})
		}
		for _, mp := range objSlice(xo.Get2("modules")) {
			extModules = append(extModules, [2]string{mp.AsString(), xo.GetString("id")})
		}
	}
	for _, b := range boundaries {
		bo := b.AsObject()
		for _, mp := range objSlice(bo.Get2("modules")) {
			boundModules = append(boundModules, [2]string{mp.AsString(), bo.GetString("id")})
		}
	}
	rules := co.GetObject("dependency_rules")
	if rules == nil {
		rules = ir.NewObject()
	}
	allow := contractEdges(rules, "allow", nil)
	deny := contractEdges(rules, "deny", nil)
	baselineRules := contractEdges(rules, "baseline", nil)
	// a wildcard baseline rule would amnesty the whole edge space, so it
	// never matches here; G2 owns the hard ERROR on the rule itself (GATE-7),
	// and the edges it would have covered stay undeclared/denied below
	baselineRules = dropWildcardEdges(baselineRules)
	ratchet, ratchetErr := LoadRatchet(design)
	if ratchetErr != nil {
		g.Errs = append(g.Errs, ratchetErr.Error())
	}
	if ratchet != nil && ratchet.Date != "" {
		g.Notes = append(g.Notes, ratchetSnapshotNote(ratchet.Date))
	}

	// matchEdgeRule (dsledges.go) is the shared rule matcher: the drawn-edge
	// check in G2 judges a diagram edge with the same wildcard semantics
	matchRule := matchEdgeRule

	goMods, goModErr := goModules(impl, ignore)
	if goModErr != nil {
		g.Errs = append(g.Errs, "discovering go modules under "+impl+": "+goModErr.Error())
	}
	tsPkgs, tsPkgErr := tsPackages(impl, ignore)
	if tsPkgErr != nil {
		g.Errs = append(g.Errs, "discovering workspace packages under "+impl+": "+tsPkgErr.Error())
	}
	manifestDeps, manifestErrs := manifestDependencies(impl, ignore)
	for _, err := range manifestErrs {
		g.Errs = append(g.Errs, "dependency manifest discovery incomplete: "+err)
	}

	resolveBoundary := func(rel string) (string, []string) { return boundaryMatch(rel, pkgmap) }
	bestNamedTarget := func(ref, sep string, pairs [][2]string) (string, []string) {
		best := -1
		owners := map[string]bool{}
		for _, pair := range pairs {
			prefix := strings.TrimSuffix(pair[0], sep)
			if ref != pair[0] && !strings.HasPrefix(ref, prefix+sep) {
				continue
			}
			if len(prefix) > best {
				best, owners = len(prefix), map[string]bool{pair[1]: true}
			} else if len(prefix) == best {
				owners[pair[1]] = true
			}
		}
		var ids []string
		for id := range owners {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		if len(ids) == 1 {
			return ids[0], nil
		}
		if len(ids) > 1 {
			return "", ids
		}
		return "", nil
	}

	internalTarget := func(ref string) (string, string, []string) {
		if _, rel, ok := goModuleFor(goMods, ref); ok {
			bid, amb := resolveBoundary(rel)
			return bid, rel, amb
		}
		if rel, ok := tsPackageFor(tsPkgs, ref); ok {
			bid, amb := resolveBoundary(rel)
			return bid, rel, amb
		}
		if bid, amb := bestNamedTarget(ref, ".", boundModules); bid != "" || len(amb) > 0 {
			return bid, ref, amb
		}
		if b, amb := resolveBoundary(ref); b != "" || len(amb) > 0 {
			return b, ref, amb
		}
		for _, ext := range []string{"", ".py", ".ts", ".tsx", ".js", ".rs"} {
			if b, amb := resolveBoundary(ref + ext); b != "" || len(amb) > 0 {
				return b, ref + ext, amb
			}
		}
		return "", "", nil
	}

	externalTarget := func(ref string) (string, []string) {
		if bid, amb := bestNamedTarget(ref, "/", extByPrefix); bid != "" || len(amb) > 0 {
			return bid, amb
		}
		if bid, amb := bestNamedTarget(ref, ".", extModules); bid != "" || len(amb) > 0 {
			return bid, amb
		}
		return "", nil
	}

	// Each distinct cross-boundary edge is judged once, but every witness file
	// is counted so a violation's error names the real amount of work.
	type edgeRec struct {
		witness string
		files   map[string]bool
	}
	edgeHits := map[[2]string]*edgeRec{}
	var edgeOrder [][2]string
	inventory, walkPruned, walkWarns, walkErr := walkSourceFilesBounded(impl, ignore, implementationDirectoryMaxEntries, implementationDirectoryMaxDepth)
	if walkErr != nil {
		g.Errs = append(g.Errs, "walking "+impl+": "+walkErr.Error())
	}
	if inventory != nil {
		defer func() {
			if closeErr := inventory.Close(); closeErr != nil {
				g.Errs = append(g.Errs, "walking "+impl+": source inventory changed before traversal completed: "+closeErr.Error())
			}
		}()
	}
	for range walkPruned {
		g.Count("dirs pruned by contract ignore")
	}
	for _, w := range walkWarns {
		g.Errs = append(g.Errs, "walk incomplete, subtree skipped: "+w)
	}
	if scan != nil {
		scan.WalkWarns = append([]string{}, walkWarns...)
	}
	var files []string
	if inventory != nil {
		files = inventory.Files()
	}
	sort.Strings(files)

	for _, rel := range files {
		ignored := false
		for _, ig := range ignore {
			if matchGlob(rel, ig) {
				ignored = true
				break
			}
		}
		if ignored {
			g.Count("files ignored by contract")
			continue
		}
		if isTestFile(rel) {
			g.Count("test files skipped")
			continue
		}
		lang := langExts[filepath.Ext(rel)]
		srcB, srcAmb := resolveBoundary(rel)
		if len(srcAmb) > 0 {
			g.Errs = append(g.Errs, "source file "+rel+" matches equally specific code globs in multiple boundaries ("+strings.Join(srcAmb, ", ")+"); boundary ownership must be unique")
			continue
		}
		body, readErr := inventory.ReadFile(rel)
		if readErr != nil {
			g.Errs = append(g.Errs, filepath.Join(impl, rel)+" is unreadable: "+readErr.Error())
			continue
		}
		text := string(body)
		if lang == "rust" {
			// judge only the production portion: imports living inside a
			// #[cfg(test)] module are test wiring (Gt's corpus), and a file
			// carrying such a module is NOT thereby a test file (NG-1)
			text, _ = rustSplitTests(text)
		}
		if srcB == "" && lang == "elixir" {
			for _, mod := range exDefmoduleRe.FindAllStringSubmatch(text, -1) {
				bid, amb := bestNamedTarget(mod[1], ".", boundModules)
				if len(amb) > 0 {
					g.Errs = append(g.Errs, "source file "+rel+" module "+mod[1]+" resolves equally specifically to multiple boundaries ("+strings.Join(amb, ", ")+")")
					continue
				}
				if srcB != "" && bid != "" && bid != srcB {
					g.Errs = append(g.Errs, "source file "+rel+" declares modules owned by both "+srcB+" and "+bid+"; one file must have one boundary owner")
					continue
				}
				if bid != "" {
					srcB = bid
				}
			}
		}
		if srcB == "" {
			g.Errs = append(g.Errs, "source file "+rel+" maps to no contract boundary; add it to a boundary's code globs or to the contract ignore list")
			if scan != nil {
				scan.UnmappedFiles = append(scan.UnmappedFiles, rel)
			}
			continue
		}
		g.Count(lang + " files checked")

		var refs []string
		switch lang {
		case "go":
			refs = goImports(text)
		case "python":
			refs = pyImports(text, rel)
		case "ts":
			refs = tsImports(text, rel)
		case "elixir":
			refs = exModules(text)
		case "rust":
			refs = rustImports(text)
		}
		for _, ref := range refs {
			dstB, norm, amb := internalTarget(ref)
			if len(amb) > 0 {
				g.Errs = append(g.Errs, rel+": import "+ref+" resolves equally specifically to multiple boundaries ("+strings.Join(amb, ", ")+"); import ownership must be unique")
				continue
			}
			if dstB == "" {
				dstB, amb = externalTarget(ref)
				if len(amb) > 0 {
					g.Errs = append(g.Errs, rel+": import "+ref+" resolves equally specifically to multiple externals ("+strings.Join(amb, ", ")+"); external ownership must be unique")
					continue
				}
				norm = ref
				if dstB == "" {
					orphaned := false
					if mod, _, inModule := goModuleFor(goMods, ref); inModule && ref != mod.path {
						orphaned = true
					} else if _, inPkg := tsPackageFor(tsPkgs, ref); inPkg {
						// bare-name imports included: importing a discovered
						// workspace package whose directory no boundary owns
						// is code outside the contract
						orphaned = true
					}
					if orphaned {
						g.Errs = append(g.Errs, rel+": imports "+ref+", which maps to no contract boundary (code outside the contract)")
						if scan != nil {
							if scan.OrphanRefs == nil {
								scan.OrphanRefs = map[string][]string{}
							}
							scan.OrphanRefs[ref] = append(scan.OrphanRefs[ref], rel)
						}
					}
					if importMatchesManifest(ref, manifestDeps) {
						g.Errs = append(g.Errs, rel+": imports manifest-declared dependency "+ref+" without a matching Architecture Contract external; dependency declarations and import literals must reconcile")
					} else if lang == "go" && !orphaned {
						head := strings.SplitN(ref, "/", 2)[0]
						if strings.Contains(head, ".") {
							g.Errs = append(g.Errs, rel+": imports undeclared third-party module "+ref+"; every external dependency must be declared under Architecture Contract externals.imports")
						}
					}
					continue
				}
			}
			g.Count("imports resolved")
			if dstB == srcB {
				continue
			}
			exp := exposes[dstB]
			if exp != nil && norm != "" {
				exposedDirs := map[string]bool{}
				for _, e := range exp {
					if !strings.Contains(e, "*") {
						exposedDirs[filepath.Dir(e)] = true
					}
				}
				ok := exposedDirs[norm]
				if !ok {
					for _, e := range exp {
						for _, cand := range []string{norm, norm + ".py", norm + ".ts", norm + ".js", norm + ".rs"} {
							if m, _ := filepath.Match(e, cand); m {
								ok = true
								break
							}
						}
						if ok {
							break
						}
					}
				}
				if !ok {
					g.Errs = append(g.Errs, rel+": imports "+ref+", which is not in the exposes list of "+dstB)
				}
			}
			edge := [2]string{srcB, dstB}
			if rec, hit := edgeHits[edge]; hit {
				rec.files[rel] = true
				continue
			}
			edgeHits[edge] = &edgeRec{witness: rel, files: map[string]bool{rel: true}}
			edgeOrder = append(edgeOrder, edge)
		}
	}
	missingRatchetReported := false
	for _, edge := range edgeOrder {
		srcB, dstB := edge[0], edge[1]
		rec := edgeHits[edge]
		seen := "seen in " + rec.witness
		if extra := len(rec.files) - 1; extra == 1 {
			seen += " and 1 more file"
		} else if extra > 1 {
			seen += fmt.Sprintf(" and %d more files", extra)
		}
		denied := matchRule(deny, srcB, dstB)
		allowed := matchRule(allow, srcB, dstB)
		baselined := matchRule(baselineRules, srcB, dstB)
		status := ""
		switch {
		case baselined && allowed:
			// G2 reports the allow+baseline contradiction; judge as allowed
			// here so the finding is not duplicated per edge
			status = "allowed"
			g.Count("edges verified")
		case baselined:
			// baseline tolerates the edge (even a denied one: intent stays
			// written as deny while the debt is being burned down) but the
			// ratchet holds it to its snapshot: no new offender files
			status = "baselined"
			g.Count("baselined edges")
			key := srcB + " -> " + dstB
			if ratchet == nil {
				if !missingRatchetReported {
					g.Errs = append(g.Errs, "contract has baseline: rules but design has no "+RatchetFile+"; run 'machinery baseline <design> --impl <dir>' to record the snapshot")
					missingRatchetReported = true
				}
			} else if snap, ok := ratchet.Edges[key]; !ok {
				g.Errs = append(g.Errs, "baselined edge "+key+" has no "+RatchetFile+" entry ("+seen+"); rerun 'machinery baseline' to record it")
			} else {
				snapSet := setOf(snap)
				var grew []string
				for f := range rec.files {
					if !snapSet[f] {
						grew = append(grew, f)
					}
				}
				sort.Strings(grew)
				if len(grew) > 0 {
					show := strings.Join(grew, ", ")
					if len(grew) > 3 {
						show = strings.Join(grew[:3], ", ") + fmt.Sprintf(" and %d more", len(grew)-3)
					}
					g.Errs = append(g.Errs, fmt.Sprintf("ratchet: baselined edge %s grew by %d new offender file(s) (%s); fix the new code, or rerun 'machinery baseline' only if the growth is a deliberate decision", key, len(grew), show))
				} else {
					shrunk := 0
					for _, f := range snap {
						if !rec.files[f] {
							shrunk++
						}
					}
					if shrunk > 0 {
						g.Notes = append(g.Notes, fmt.Sprintf("ratchet can tighten: %s dropped %d offender file(s); rerun 'machinery baseline'", key, shrunk))
					}
					g.Count("ratcheted edges")
				}
			}
		case denied && !allowed:
			status = "denied"
			g.Errs = append(g.Errs, srcB+" -> "+dstB+" is denied by the contract ("+seen+"); either the code violates the boundary or the contract needs an explicit allow")
		case !allowed && !denied:
			status = "undeclared"
			g.Errs = append(g.Errs, "undeclared cross-boundary edge "+srcB+" -> "+dstB+" ("+seen+"); add an explicit allow or deny to the contract")
		default:
			status = "allowed"
			g.Count("edges verified")
		}
		if scan != nil {
			var files []string
			for f := range rec.files {
				files = append(files, f)
			}
			sort.Strings(files)
			scan.Edges = append(scan.Edges, scanEdge{Src: srcB, Dst: dstB, Witness: rec.witness, Files: files, Status: status})
		}
	}
	if ratchet != nil {
		observed := map[string]bool{}
		for _, e := range edgeOrder {
			observed[e[0]+" -> "+e[1]] = true
		}
		var stale []string
		for k := range ratchet.Edges {
			if !observed[k] {
				stale = append(stale, k)
			}
		}
		sort.Strings(stale)
		for _, k := range stale {
			g.Notes = append(g.Notes, "ratchet edge "+k+" is no longer observed; rerun 'machinery baseline' to retire it")
		}
	}
	if scan != nil {
		scan.Complete = true
	}

	anyChecked := false
	for k, v := range g.Counts {
		if strings.HasSuffix(k, "files checked") && v > 0 {
			anyChecked = true
			break
		}
	}
	if !anyChecked {
		g.Errs = append(g.Errs, "no source files under "+impl+" mapped to any contract boundary; the gate checked nothing")
	}
	g.RequireNonzero("imports resolved", "no imports were resolved against the contract")
	return g
}

// dropWildcardEdges removes rules with a wildcard on either side; used only
// for baseline: rules, which are an enumerated-edges ratchet.
func dropWildcardEdges(edges [][2]string) [][2]string {
	var out [][2]string
	for _, e := range edges {
		if strings.Contains(e[0], "*") || strings.Contains(e[1], "*") {
			continue
		}
		out = append(out, e)
	}
	return out
}

// ratchetSnapshotNote makes tolerated debt visible without consulting the
// wall clock. The same tree and arguments therefore produce byte-identical
// check output across midnight and on hosts with different clock settings.
func ratchetSnapshotNote(date string) string {
	for _, layout := range []string{"2006-01-02", "2006-01"} {
		if _, err := time.Parse(layout, date); err == nil {
			return "ratchet snapshot " + date
		}
	}
	return fmt.Sprintf("ratchet snapshot dated %s (not a YYYY-MM or YYYY-MM-DD date)", ir.Repr(date))
}
