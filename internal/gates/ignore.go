// `.machineryignore`: paths under the design tree that are not design.
//
// `machinery check` reads every markdown and yaml file under the design tree,
// which is right: a design document nobody registered is exactly what the
// walkers exist to catch. It stops being right the moment the tree carries
// something that is not authored design at all. A spike project committed
// under design/experiments/ pulls its dependencies into a deps/ directory,
// and a vendored library's README is then scanned for house style and copied
// tables: on one private design that produced em-dash warnings and
// duplicate-table warnings against code nobody in the design wrote and nobody
// may edit. The alternative to an ignore file is worse: either the gate
// suite stays noisy, or the author moves real evidence out of the design
// tree to keep it quiet.
//
// The file is `.machineryignore` at the DESIGN ROOT, gitignore-shaped:
//
//	# fetched dependencies of the spike projects, not design
//	experiments/spikes/*/deps
//	_build
//
// One pattern per line, '#' comments and blank lines ignored, patterns
// relative to the design root. A pattern with no "/" matches any path SEGMENT
// (so `deps` ignores every deps directory at any depth); a pattern with a "/"
// (or a leading "/") matches the path from the design root, and matching a
// directory ignores everything under it. `*` matches within one segment, `**`
// crosses segments, `?` matches one character. It is deliberately NOT a full
// gitignore implementation: there are no negations and no re-inclusions,
// because an ignore list a reader cannot evaluate by eye is worse than none.

package gates

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// IgnoreFileName is the design-root ignore list.
const IgnoreFileName = ".machineryignore"

// designIgnore is one design's compiled ignore list.
type designIgnore struct {
	present bool
	pats    []ignorePattern
}

type ignorePattern struct {
	re      *regexp.Regexp
	rooted  bool // the pattern names a path, not a bare segment
	literal string
}

// ignoreCache memoizes the compiled list per design, keyed by the file's
// modification stamp so an edit is never served stale.
var (
	ignoreMu    sync.Mutex
	ignoreCache = map[string]cachedIgnore{}
)

type cachedIgnore struct {
	stamp string
	set   *designIgnore
}

// designIgnoreFor loads (and memoizes) the ignore list of a design.
func designIgnoreFor(design string) *designIgnore {
	path := filepath.Join(design, IgnoreFileName)
	stamp := "absent"
	body, readErr := readDesignFile(design, path)
	if readErr == nil {
		stamp = fmt.Sprintf("sha256:%x", sha256.Sum256(body))
	}
	key, err := filepath.Abs(design)
	if err != nil {
		key = design
	}
	ignoreMu.Lock()
	defer ignoreMu.Unlock()
	if c, ok := ignoreCache[key]; ok && c.stamp == stamp {
		return c.set
	}
	set := parseIgnore(string(body), readErr == nil)
	ignoreCache[key] = cachedIgnore{stamp: stamp, set: set}
	return set
}

// parseIgnore compiles an ignore file body.
func parseIgnore(body string, present bool) *designIgnore {
	out := &designIgnore{present: present}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rooted := strings.Contains(strings.Trim(line, "/"), "/") || strings.HasPrefix(line, "/")
		p := strings.Trim(line, "/")
		if p == "" {
			continue
		}
		out.pats = append(out.pats, ignorePattern{re: globRegexp(p), rooted: rooted, literal: p})
	}
	return out
}

// globRegexp compiles one glob into an anchored regexp: `**` crosses path
// separators, `*` and `?` do not, everything else is literal.
func globRegexp(pat string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString(`\A`)
	for i := 0; i < len(pat); i++ {
		switch c := pat[i]; c {
		case '*':
			if i+1 < len(pat) && pat[i+1] == '*' {
				b.WriteString(`.*`)
				i++
				continue
			}
			b.WriteString(`[^/]*`)
		case '?':
			b.WriteString(`[^/]`)
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString(`\z`)
	re, err := regexp.Compile(b.String())
	if err != nil {
		// unreachable: every branch above emits valid syntax. A pattern that
		// somehow does not compile matches nothing rather than everything.
		return regexp.MustCompile(`\A\z`)
	}
	return re
}

// skips reports whether a design-relative path is ignored. A nil list ignores
// nothing, so every caller can hold one unconditionally.
func (ig *designIgnore) skips(rel string) bool {
	if ig == nil || len(ig.pats) == 0 {
		return false
	}
	rel = filepath.ToSlash(rel)
	segs := strings.Split(rel, "/")
	for _, p := range ig.pats {
		if !p.rooted {
			for _, s := range segs {
				if p.re.MatchString(s) {
					return true
				}
			}
			continue
		}
		// a rooted pattern matches the path itself or any ancestor of it, so
		// naming a directory ignores everything under it
		for i := 1; i <= len(segs); i++ {
			if p.re.MatchString(strings.Join(segs[:i], "/")) {
				return true
			}
		}
	}
	return false
}

// DesignIgnores reports whether a design-relative path is listed in the
// design's `.machineryignore`. Exported for the walkers outside this package
// (`machinery sweep`), which must ignore exactly what the gates ignore.
func DesignIgnores(design, rel string) bool {
	return designIgnoreFor(design).skips(rel)
}

// ignoredHere is the walkers' shared skip test: a design-relative path that
// the ignore list names is not design and is not scanned.
func ignoredHere(design, rel string) bool {
	return designIgnoreFor(design).skips(rel)
}
