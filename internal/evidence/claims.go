package evidence

import (
	"path"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/tags"
)

// claims answers whether the test of a receipt lies in a test file that
// statically claims the receipt's id (section 10.2).
type claims struct {
	p     *project.Project
	files []string                   // the files matched by trace.tests
	ids   map[string]map[string]bool // file -> the ids its tags and markers verify
	text  map[string]string          // file -> content, read on demand
	lies  map[string][]string        // test -> the files it lies in
}

// newClaims reads the verifies tags and markers of the test files exactly as
// stage trace does (sections 8 and 8.1).
func newClaims(p *project.Project, d *design.Design) (*claims, error) {
	c := &claims{p: p, ids: map[string]map[string]bool{}, text: map[string]string{}, lies: map[string][]string{}}
	files, err := tags.TestFiles(p)
	if err != nil {
		return nil, err
	}
	c.files = files
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

// claimed reports whether a file the receipt's test lies in claims its id.
func (c *claims) claimed(r Receipt) bool {
	files, seen := c.lies[r.Test]
	if !seen {
		files = c.filesOf(r.Test)
		c.lies[r.Test] = files
	}
	for _, f := range files {
		if c.ids[f][r.ID] {
			return true
		}
	}
	return false
}

// filesOf returns the test files a test lies in: the files it names by path
// or base name, and when it names none, the files whose text holds the name
// of the test (the text before the first slash) as a whole word.
func (c *claims) filesOf(test string) []string {
	var res []string
	for _, f := range c.files {
		if names(test, f) || names(test, path.Base(f)) {
			res = append(res, f)
		}
	}
	if len(res) > 0 {
		return res
	}
	name, _, _ := strings.Cut(test, "/")
	if name == "" {
		return nil
	}
	for _, f := range c.files {
		text, ok := c.text[f]
		if !ok {
			data, err := c.p.Read(f)
			if err != nil {
				continue
			}
			text = string(data)
			c.text[f] = text
		}
		if holdsWord(text, name) {
			res = append(res, f)
		}
	}
	return res
}

// names reports whether test is the file name, or starts with it followed by
// a colon, a slash, a number sign or a space.
func names(test, file string) bool {
	rest, ok := strings.CutPrefix(test, file)
	return ok && (rest == "" || strings.ContainsRune(":/# ", rune(rest[0])))
}

// holdsWord reports whether text holds word with no letter, digit or
// underscore directly before or after it.
func holdsWord(text, word string) bool {
	for from := 0; ; {
		i := strings.Index(text[from:], word)
		if i < 0 {
			return false
		}
		start := from + i
		end := start + len(word)
		if (start == 0 || !wordByte(text[start-1])) && (end == len(text) || !wordByte(text[end])) {
			return true
		}
		from = start + 1
	}
}

func wordByte(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= 0x80
}
