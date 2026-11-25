package arch

import (
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// langOf names the language of a source file by extension, or "".
func langOf(f string) string {
	switch strings.ToLower(path.Ext(f)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs":
		return "js"
	case ".ex", ".exs":
		return "elixir"
	case ".rs":
		return "rust"
	}
	return ""
}

// importsOf lists the project paths a non-Go file imports. Anything that does
// not resolve to a file of the project is external and left out.
func importsOf(lang, f, src string, files map[string]bool) []target {
	var paths []string
	switch lang {
	case "python":
		paths = pythonImports(f, src, files)
	case "js":
		paths = jsImports(f, src, files)
	case "elixir":
		paths = elixirImports(src, files)
	case "rust":
		paths = rustImports(f, src, files)
	}
	ts := make([]target, 0, len(paths))
	for _, p := range paths {
		ts = append(ts, target{path: p})
	}
	return ts
}

// firstExisting returns the first candidate that is a project file.
func firstExisting(files map[string]bool, cands ...string) string {
	for _, c := range cands {
		if files[c] {
			return c
		}
	}
	return ""
}

func addIf(res []string, p string) []string {
	if p != "" {
		res = append(res, p)
	}
	return res
}

// Python

var (
	pyFrom   = regexp.MustCompile(`(?m)^[ \t]*from[ \t]+(\.*)([A-Za-z_][\w.]*)?[ \t]+import[ \t]+(.+)$`)
	pyImport = regexp.MustCompile(`(?m)^[ \t]*import[ \t]+([A-Za-z_][\w.]*(?:[ \t]+as[ \t]+\w+)?(?:[ \t]*,[ \t]*[A-Za-z_][\w.]*(?:[ \t]+as[ \t]+\w+)?)*)`)
)

func pyModule(base string, parts []string, files map[string]bool) string {
	p := path.Join(append([]string{base}, parts...)...)
	if p == "." || p == "" {
		return firstExisting(files, "__init__.py")
	}
	return firstExisting(files, p+".py", p+"/__init__.py")
}

func pythonImports(f, src string, files map[string]bool) []string {
	var res []string
	for _, m := range pyFrom.FindAllStringSubmatch(src, -1) {
		dots, mod, names := m[1], m[2], m[3]
		bases := []string{"", "src"}
		if dots != "" {
			d := path.Dir(f)
			for i := 1; i < len(dots); i++ {
				d = path.Dir(d)
			}
			if d == "." {
				d = ""
			}
			bases = []string{d}
		}
		var parts []string
		if mod != "" {
			parts = strings.Split(mod, ".")
		}
		for _, b := range bases {
			found := false
			for _, n := range strings.Split(strings.SplitN(names, "#", 2)[0], ",") {
				n = strings.Trim(strings.TrimSpace(strings.SplitN(strings.TrimSpace(n), " ", 2)[0]), "()")
				if n == "" || n == "*" {
					continue
				}
				if t := pyModule(b, append(append([]string{}, parts...), n), files); t != "" {
					res, found = append(res, t), true
				}
			}
			if !found && len(parts) > 0 {
				if t := pyModule(b, parts, files); t != "" {
					res, found = append(res, t), true
				}
			}
			if found {
				break
			}
		}
	}
	for _, m := range pyImport.FindAllStringSubmatch(src, -1) {
		for _, item := range strings.Split(m[1], ",") {
			mod := strings.Fields(item)[0]
			for _, b := range []string{"", "src"} {
				if t := pyModule(b, strings.Split(mod, "."), files); t != "" {
					res = append(res, t)
					break
				}
			}
		}
	}
	return res
}

// TypeScript and JavaScript

var jsSpec = []*regexp.Regexp{
	regexp.MustCompile(`(?:import|export)\s[^'"\x60;]*?\sfrom\s*['"]([^'"]+)['"]`),
	regexp.MustCompile(`\bimport\s*['"]([^'"]+)['"]`),
	regexp.MustCompile(`\b(?:require|import)\s*\(\s*['"]([^'"]+)['"]\s*\)`),
}

var jsExts = []string{".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs"}

func jsImports(f, src string, files map[string]bool) []string {
	var res []string
	for _, re := range jsSpec {
		for _, m := range re.FindAllStringSubmatch(src, -1) {
			res = addIf(res, jsResolve(f, m[1], files))
		}
	}
	return res
}

func jsResolve(f, spec string, files map[string]bool) string {
	var base string
	switch {
	case strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../") || spec == "." || spec == "..":
		base = path.Join(path.Dir(f), spec)
	case strings.HasPrefix(spec, "/"):
		base = path.Clean(strings.TrimPrefix(spec, "/"))
	default:
		base = path.Clean(spec)
	}
	cands := []string{base}
	stem := base
	if e := path.Ext(base); e == ".js" || e == ".jsx" || e == ".mjs" || e == ".cjs" {
		stem = strings.TrimSuffix(base, e)
	}
	for _, e := range jsExts {
		cands = append(cands, base+e, stem+e)
	}
	for _, e := range jsExts {
		cands = append(cands, base+"/index"+e)
	}
	return firstExisting(files, cands...)
}

// Elixir

var exAlias = regexp.MustCompile(`(?m)^[ \t]*(?:alias|import|use)[ \t]+([A-Z][A-Za-z0-9_]*(?:\.[A-Z][A-Za-z0-9_]*)*)(?:\.\{([^}]*)\})?`)

func elixirImports(src string, files map[string]bool) []string {
	var exFiles []string
	for f := range files {
		if e := path.Ext(f); e == ".ex" || e == ".exs" {
			exFiles = append(exFiles, f)
		}
	}
	sort.Strings(exFiles)
	var res []string
	for _, m := range exAlias.FindAllStringSubmatch(src, -1) {
		mods := []string{m[1]}
		if m[2] != "" {
			mods = nil
			for _, c := range strings.Split(m[2], ",") {
				if c = strings.TrimSpace(c); c != "" {
					mods = append(mods, m[1]+"."+c)
				}
			}
		}
		for _, mod := range mods {
			want := elixirPath(mod)
			for _, f := range exFiles {
				stem := strings.TrimSuffix(f, path.Ext(f))
				if stem == want || strings.HasSuffix(stem, "/"+want) {
					res = append(res, f)
					break
				}
			}
		}
	}
	return res
}

// elixirPath turns Shop.Api.Handler into shop/api/handler.
func elixirPath(mod string) string {
	var segs []string
	for _, s := range strings.Split(mod, ".") {
		segs = append(segs, snake(s))
	}
	return strings.Join(segs, "/")
}

func snake(s string) string {
	rs := []rune(s)
	var b strings.Builder
	for i, r := range rs {
		if unicode.IsUpper(r) && i > 0 {
			prev := rs[i-1]
			nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				b.WriteByte('_')
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// Rust

var (
	rsUse = regexp.MustCompile(`(?m)^[ \t]*(?:pub(?:\([^)]*\))?[ \t]+)?use[ \t]+crate::([A-Za-z0-9_:]+)`)
	rsMod = regexp.MustCompile(`(?m)^[ \t]*(?:pub(?:\([^)]*\))?[ \t]+)?mod[ \t]+([A-Za-z0-9_]+)[ \t]*;`)
)

// rustRoot is the crate source directory of a file: the path up to and
// including its src component, or the project root.
func rustRoot(f string) string {
	if strings.HasPrefix(f, "src/") {
		return "src"
	}
	if i := strings.LastIndex(f, "/src/"); i >= 0 {
		return f[:i+4]
	}
	return ""
}

func rustImports(f, src string, files map[string]bool) []string {
	var res []string
	root := rustRoot(f)
	for _, m := range rsUse.FindAllStringSubmatch(src, -1) {
		segs := strings.Split(strings.Trim(m[1], ":"), "::")
		for n := len(segs); n >= 1; n-- {
			base := path.Join(root, path.Join(segs[:n]...))
			if t := firstExisting(files, base+".rs", base+"/mod.rs"); t != "" {
				res = append(res, t)
				break
			}
		}
	}
	dir := path.Dir(f)
	stem := strings.TrimSuffix(path.Base(f), ".rs")
	if stem != "lib" && stem != "main" && stem != "mod" {
		dir = path.Join(dir, stem)
	}
	for _, m := range rsMod.FindAllStringSubmatch(src, -1) {
		res = addIf(res, firstExisting(files, path.Join(dir, m[1]+".rs"), path.Join(dir, m[1], "mod.rs")))
	}
	return res
}
