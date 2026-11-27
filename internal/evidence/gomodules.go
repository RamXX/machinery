package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/gitx"
	"github.com/RamXX/machinery/v2/internal/project"
	"github.com/RamXX/machinery/v2/internal/tags"
)

// GoEnv is the go environment of a direct-mode adapter run, recorded in its
// header (section 10.3). It decides nothing.
type GoEnv struct {
	Version  string `json:"go_version"`
	Binary   string `json:"go_binary"`
	GOFLAGS  string `json:"GOFLAGS"`
	GOWORK   string `json:"GOWORK"`
	GOOS     string `json:"GOOS"`
	GOARCH   string `json:"GOARCH"`
	GoModSum string `json:"go_mod_sha256"`
	WorkSum  string `json:"go_work_sha256"`
}

// EnvLine is the text of the adapter environment line of a run (sections
// 10.3, 12 and 13); env is nil for a run through --streams.
func EnvLine(file string, env *GoEnv) string {
	if env == nil {
		return "adapter environment: " + file + ": not recorded: the command ran through --streams"
	}
	short := func(sum string) string {
		if sum == "" {
			return "none"
		}
		return sum[:min(12, len(sum))]
	}
	return fmt.Sprintf("adapter environment: %s: %s %s, GOOS/GOARCH %s/%s, GOFLAGS %q, GOWORK %q, go.mod %s, go.work %s",
		file, env.Version, env.Binary, env.GOOS, env.GOARCH, env.GOFLAGS, env.GOWORK, short(env.GoModSum), short(env.WorkSum))
}

// fileSum is the SHA-256 of a file, "" when it cannot be read.
func fileSum(abs string) string {
	data, err := os.ReadFile(abs)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ClaimingGoTests returns the claiming files of section 10.3 among the tags
// found: the _test.go files under trace.tests that claim an id of a kind the
// adapter joins, sorted.
func ClaimingGoTests(d *design.Design, found []tags.Tag) []string {
	ids, scenarios := d.DeclaredIDs(), d.ScenarioIDs()
	set := map[string]bool{}
	for _, t := range found {
		if t.Kind != tags.Verifies || !strings.HasSuffix(t.File, "_test.go") {
			continue
		}
		for _, tok := range t.IDs {
			for _, id := range tags.Resolve(t, tok, ids, scenarios) {
				if o := d.Obligation(id); o != nil && contains(project.AdapterKinds, o.Kind) {
					set[t.File] = true
				}
			}
		}
	}
	return sortedKeys(set)
}

// GoLayout returns the problems of the layout of the Go modules (section
// 10.3) as stage trace reports them, for a project that configures
// evidence.adapter: a module path declared by two go.mod files, and a module
// of a claiming file that the workspace does not use. The workspace is the
// file GOWORK names, or else the go.work of the project root; none with
// GOWORK=off.
func GoLayout(p *project.Project, d *design.Design, found []tags.Tag) ([]string, error) {
	if p.Cfg.Evidence == nil || p.Cfg.Evidence.Adapter == nil {
		return nil, nil
	}
	claiming := ClaimingGoTests(d, found)
	if len(claiming) == 0 {
		return nil, nil
	}
	res, err := duplicateModules(p)
	if err != nil {
		return nil, err
	}
	work := os.Getenv("GOWORK")
	switch work {
	case "off":
		return res, nil
	case "":
		work = p.Abs("go.work")
	}
	unused, err := unusedModules(p, claiming, work)
	return append(res, unused...), err
}

// countsAsModule reports a project-relative path of a go.mod or go.work file
// the layout rules read: not under testdata or vendor (section 10.3).
func countsAsModule(rel string) bool {
	for _, el := range strings.Split(rel, "/") {
		if el == "testdata" || el == "vendor" {
			return false
		}
	}
	return true
}

// duplicateModules returns, sorted, the finding for every module path two
// go.mod files of the project declare: those git tracks, outside testdata
// and vendor.
func duplicateModules(p *project.Project) ([]string, error) {
	files, err := gitx.Tracked(p.Root, "*go.mod")
	if err != nil {
		return nil, err
	}
	by := map[string][]string{}
	for _, rel := range files {
		if path.Base(rel) != "go.mod" || !countsAsModule(rel) {
			continue
		}
		data, err := p.Read(rel)
		if err != nil {
			continue // tracked and deleted: no module
		}
		if mod := modulePath(data); mod != "" {
			by[mod] = append(by[mod], rel)
		}
	}
	var res []string
	for _, mod := range sortedKeys(by) {
		if rels := by[mod]; len(rels) > 1 {
			sort.Strings(rels)
			res = append(res, fmt.Sprintf("module %s is declared by %s and %s: give each module its own path, or move the copy under a testdata directory", mod, strings.Join(rels[:len(rels)-1], ", "), rels[len(rels)-1]))
		}
	}
	return res, nil
}

// unusedModules returns the finding for every module of a claiming file that
// the workspace file does not use, naming its first claiming file. The
// workspace is read only when git tracks it in the project, outside testdata
// and vendor.
func unusedModules(p *project.Project, claiming []string, work string) ([]string, error) {
	rel, in := inside(p.Root, work)
	if !in || !countsAsModule(rel) {
		return nil, nil
	}
	if tracked, err := gitx.Tracked(p.Root, ":(literal)"+rel); err != nil || len(tracked) == 0 {
		return nil, err
	}
	uses := useDirs(work)
	var res []string
	seen := map[string]bool{}
	for _, file := range claiming {
		mod, ok := moduleFile(p.Abs(path.Dir(file)))
		if !ok || seen[mod] {
			continue
		}
		seen[mod] = true
		if !uses[filepath.Dir(mod)] {
			res = append(res, fmt.Sprintf("%s does not use the module of %s (%s): add the module to the workspace, or set GOWORK=off", rel, file, shown(p, mod)))
		}
	}
	return res, nil
}

// useDirs reads the use directives of a workspace file: the directories they
// name, absolute. A file that cannot be read uses nothing.
func useDirs(work string) map[string]bool {
	res := map[string]bool{}
	data, err := os.ReadFile(work)
	if err != nil {
		return res
	}
	block := false
	for _, line := range strings.Split(string(data), "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		dir := ""
		switch rest, isUse := strings.CutPrefix(line, "use"); {
		case block && line == ")":
			block = false
		case block:
			dir = line
		case isUse && strings.TrimSpace(rest) == "(":
			block = true
		case isUse && rest != "" && (rest[0] == ' ' || rest[0] == '\t'):
			dir = strings.TrimSpace(rest)
		}
		if dir = strings.Trim(dir, "\"`"); dir == "" {
			continue
		}
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(filepath.Dir(work), dir)
		}
		res[filepath.Clean(dir)] = true
	}
	return res
}

// moduleFile returns the nearest go.mod at or above a directory, absolute.
func moduleFile(abs string) (string, bool) {
	for dir := abs; ; {
		if f := filepath.Join(dir, "go.mod"); fileExists(f) {
			return f, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func fileExists(abs string) bool {
	info, err := os.Stat(abs)
	return err == nil && !info.IsDir()
}

// shown names a file by its project-relative path when it is in the
// project, and by its absolute path otherwise.
func shown(p *project.Project, abs string) string {
	if rel, inside := inside(p.Root, abs); inside {
		return rel
	}
	return abs
}

// inside reports whether an absolute path is inside root, and its
// root-relative form. Links are not followed.
func inside(root, abs string) (string, bool) {
	if !filepath.IsAbs(abs) {
		return "", false
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return filepath.ToSlash(rel), true
}
