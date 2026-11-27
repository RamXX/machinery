package evidence

import (
	"fmt"
	"path"
	"strings"

	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/project"
	"github.com/RamXX/machinery/v2/internal/tags"
)

// claims answers whether the test of a receipt names a test file that
// statically claims the receipt's id (section 10.2).
type claims struct {
	set    bool                       // trace.tests is configured
	files  map[string]bool            // the files matched by trace.tests
	byBase map[string][]string        // base name -> those files
	ids    map[string]map[string]bool // file -> the ids its tags and markers verify
}

// newClaims reads the verifies tags and markers of the test files exactly as
// stage trace does (sections 8 and 8.1).
func newClaims(p *project.Project, d *design.Design) (*claims, error) {
	c := &claims{files: map[string]bool{}, byBase: map[string][]string{}, ids: map[string]map[string]bool{}}
	c.set = p.Cfg.Trace != nil && len(p.Cfg.Trace.Tests) > 0
	files, err := tags.TestFiles(p)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		c.files[f] = true
		c.byBase[path.Base(f)] = append(c.byBase[path.Base(f)], f)
	}
	found, err := tags.Scan(p, files)
	if err != nil {
		return nil, err
	}
	ids, scenarios := d.DeclaredIDs(), d.ScenarioIDs()
	for _, t := range found {
		if t.Kind != tags.Verifies {
			continue
		}
		for _, tok := range t.IDs {
			for _, id := range tags.Resolve(t, tok, ids, scenarios) {
				if c.ids[t.File] == nil {
					c.ids[t.File] = map[string]bool{}
				}
				c.ids[t.File][id] = true
			}
		}
	}
	return c, nil
}

// unclaimed returns "" when the test of the receipt names a test file that
// claims its id, and otherwise what is wrong and what to do about it.
func (c *claims) unclaimed(r Receipt) string {
	if !c.set {
		return "trace.tests is not set: list the test files there in machinery.yaml"
	}
	file, why := c.resolve(r.Test)
	if why != "" {
		return why
	}
	if !c.ids[file][r.ID] {
		return fmt.Sprintf("add machinery:verifies %s to %s, or emit the receipt from the test that claims it", r.ID, file)
	}
	return ""
}

// resolve returns the test file the test of a receipt names, or why it names
// none. The test is <file> or <file>::<name>; the file is a project-relative
// path, or a base name that exactly one test file has (section 10.2).
func (c *claims) resolve(test string) (file, why string) {
	name, _, _ := strings.Cut(test, "::")
	if c.files[name] {
		return name, ""
	}
	switch same := c.byBase[name]; len(same) {
	case 1:
		return same[0], ""
	case 0:
		return "", fmt.Sprintf("no file under trace.tests is named %s: write test as <path of the test file> or <path>::<name>", name)
	default:
		return "", fmt.Sprintf("%d files under trace.tests are named %s: write the path in test", len(same), name)
	}
}

// fileOf returns the test file the test of a receipt names by that rule, or
// "" when it names none. A nil c knows no file.
func (c *claims) fileOf(test string) string {
	if c == nil {
		return ""
	}
	file, _ := c.resolve(test)
	return file
}
