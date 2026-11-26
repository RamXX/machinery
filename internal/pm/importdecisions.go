package pm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/tags"
)

// DecisionRecord is one record of an existing decision file (section 15.8).
type DecisionRecord struct {
	Line    int // the line of its heading, from 1
	Heading string
	Title   string
	Body    string
	IDs     []string
	Source  string // <path>#<12 hex>
}

var importToken = regexp.MustCompile(`[A-Za-z0-9_.:-]+`)

// DecisionRecords splits a decision file into its records: a line the
// expression matches starts one, and it runs to the next. The ids are the
// design ids its text names, resolved as a project's own trace markers are.
func DecisionRecords(text string, record *regexp.Regexp, path string, d *design.Design) []DecisionRecord {
	var res []DecisionRecord
	var body []string
	flush := func() {
		if len(res) == 0 {
			return
		}
		r := &res[len(res)-1]
		for len(body) > 0 && strings.TrimSpace(body[0]) == "" {
			body = body[1:]
		}
		for len(body) > 0 && strings.TrimSpace(body[len(body)-1]) == "" {
			body = body[:len(body)-1]
		}
		r.Body = strings.Join(body, "\n")
		r.IDs = namedIDs(r.Heading+"\n"+r.Body, d)
		body = nil
	}
	for i, line := range splitLines(text) {
		if record.MatchString(line) {
			flush()
			heading := strings.TrimRight(line, " \t")
			title := strings.TrimSpace(strings.TrimLeft(heading, "#"))
			if title == "" {
				title = strings.TrimSpace(heading)
			}
			sum := sha256.Sum256([]byte(heading))
			res = append(res, DecisionRecord{Line: i + 1, Heading: heading, Title: title,
				Source: path + "#" + hex.EncodeToString(sum[:])[:12]})
			continue
		}
		if len(res) > 0 {
			body = append(body, line)
		}
	}
	flush()
	return res
}

// splitLines splits a text into its lines, carriage returns dropped.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	ls := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	for i := range ls {
		ls[i] = strings.TrimSuffix(ls[i], "\r")
	}
	return ls
}

// CountLines counts the lines of a file as section 18.7 does.
func CountLines(text string) int { return len(splitLines(text)) }

// namedIDs lists the design ids a text names, sorted, once each.
func namedIDs(text string, d *design.Design) []string {
	ids := d.DeclaredIDs()
	set := map[string]bool{}
	marker := tags.Tag{Marker: true}
	for _, tok := range importToken.FindAllString(text, -1) {
		tok = strings.TrimRight(tok, ".:")
		if tok == "" {
			continue
		}
		for _, id := range tags.Resolve(marker, tok, ids, nil) {
			set[id] = true
		}
	}
	res := make([]string, 0, len(set))
	for id := range set {
		res = append(res, id)
	}
	sort.Strings(res)
	return res
}

// ImportDescription renders the description of an imported decision.
func ImportDescription(r DecisionRecord, actor string) string {
	body := strings.ReplaceAll(r.Body, "machinery:", "machinery :")
	if body == "" {
		body = r.Title
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Why:\n%s\n\nmachinery:decision %s\n", body, actor)
	if len(r.IDs) > 0 {
		fmt.Fprintf(&b, "machinery:ids %s\n", strings.Join(r.IDs, " "))
	}
	fmt.Fprintf(&b, "machinery:decision-source %s\n", r.Source)
	return b.String()
}

// ImportedSources maps each machinery:decision-source of the backlog to the
// item that carries it, the first by id.
func ImportedSources(issues []backlog.Issue) map[string]string {
	res := map[string]string{}
	for i := range issues {
		for _, s := range backlog.ParseTags(&issues[i]).Sources {
			if cur, ok := res[s]; !ok || issues[i].ID < cur {
				res[s] = issues[i].ID
			}
		}
	}
	return res
}
