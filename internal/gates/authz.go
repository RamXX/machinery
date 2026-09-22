// Package gates implements deterministic design checks. This file owns the
// Gx authorization-inventory join for System actions and matrix producers.
package gates

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

var (
	authorizationMarker = regexp.MustCompile(`<!--\s*machinery:authorization-inventory\s*-->`)
	noAuthorization     = regexp.MustCompile(`\(no authorization:\s*([^)]*)\)`)
)

type authorizationRow struct {
	subject, admission, where string
}

func authorizationObligations(g *Gate, design string, dm *ir.Value) map[string]string {
	out := map[string]string{}
	entities := dm.AsObject().GetObject("entities")
	for _, entity := range entities.Keys() {
		for _, action := range objSlice(entities.Get2(entity).AsObject().Get2("actions")) {
			if action.Kind != ir.KindObject || action.AsObject().GetString("actor") != "System" {
				continue
			}
			name := strings.TrimSpace(action.AsObject().GetString("name"))
			if name != "" {
				out[entity+"."+name] = "System action"
			}
		}
	}
	paths, _ := strictSortedGlob(g, filepath.Join(design, "machines"), "*.matrix.md", "authorization matrix")
	for _, path := range paths {
		body, err := readDesignFile(design, path)
		if err != nil {
			g.Errs = append(g.Errs, filepath.Base(path)+": unreadable authorization matrix: "+err.Error())
			continue
		}
		for _, table := range ir.ParseMdTables(string(body)) {
			headers := strings.ToLower(strings.Join(table.Header, " "))
			if !strings.Contains(headers, "producer") ||
				(!strings.Contains(headers, "cascade") && !strings.Contains(headers, "consumer")) {
				continue
			}
			pi := colContaining(table.Header, "producer")
			for _, row := range table.Rows {
				cell := cellAt(row, pi)
				var subjects []string
				for _, match := range payloadBacktick.FindAllStringSubmatch(cell, -1) {
					subjects = append(subjects, match[1])
				}
				if len(subjects) == 0 {
					if subject := ir.CleanCell(cell); subject != "" {
						subjects = append(subjects, subject)
					}
				}
				for _, subject := range subjects {
					out[subject] = "matrix producer"
				}
			}
		}
	}
	return out
}

func collectAuthorizationRows(g *Gate, archText string) []authorizationRow {
	var out []authorizationRow
	for _, table := range ir.ParseMdTables(archText) {
		si := colContaining(table.Header, "authorization subject")
		ai := colContaining(table.Header, "admission")
		if si < 0 && ai < 0 {
			continue
		}
		if si < 0 || ai < 0 {
			g.Errs = append(g.Errs, "authorization inventory table must have authorization subject and admission columns")
			continue
		}
		for i, row := range table.Rows {
			subject := ir.CleanCell(cellAt(row, si))
			out = append(out, authorizationRow{
				subject: subject, admission: strings.TrimSpace(cellAt(row, ai)),
				where: "authorization row " + strconv.Itoa(i+1),
			})
		}
	}
	return out
}

func authorizationInventoryCorpus(g *Gate, design string) (string, int) {
	var documents []string
	markers := 0
	err := walkTreeBounded(design, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(design, path)
		if rerr != nil {
			rel = path
		}
		if ignoredHere(design, rel) {
			if fi.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if fi.IsDir() || strings.ToLower(filepath.Ext(fi.Name())) != ".md" {
			return nil
		}
		body, readErr := readDesignFile(design, path)
		if readErr != nil {
			return readErr
		}
		found := authorizationMarker.FindAll(body, -1)
		if len(found) == 0 {
			return nil
		}
		markers += len(found)
		documents = append(documents, string(body))
		return nil
	})
	if err != nil {
		g.Errs = append(g.Errs, "authorization inventory scan failed: "+err.Error())
	}
	return strings.Join(documents, "\n"), markers
}

func checkAuthorizationInventory(g *Gate, design string, dm *ir.Value) {
	obligations := authorizationObligations(g, design, dm)
	if len(obligations) == 0 {
		return
	}
	corpus, markers := authorizationInventoryCorpus(g, design)
	if markers == 0 {
		g.Errs = append(g.Errs, "System writes exist but the design has no <!-- machinery:authorization-inventory --> marker and closed authorization inventory")
	} else if markers > 1 {
		g.Errs = append(g.Errs, "authorization inventory marker appears "+strconv.Itoa(markers)+" times; one design has exactly one closed authorization inventory")
	}
	rows := collectAuthorizationRows(g, corpus)
	if markers > 0 && len(rows) == 0 {
		g.Errs = append(g.Errs, "authorization inventory marker has no table with authorization subject and admission columns")
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if row.subject == "" {
			g.Errs = append(g.Errs, row.where+": authorization row names no subject")
			continue
		}
		if seen[row.subject] {
			g.Errs = append(g.Errs, row.where+": duplicate authorization row for "+ir.Repr(row.subject))
			continue
		}
		seen[row.subject] = true
		if _, ok := obligations[row.subject]; !ok {
			g.Errs = append(g.Errs, row.where+": "+ir.Repr(row.subject)+" names no System action or matrix producer; remove the stale authorization row")
			continue
		}
		if waiver := noAuthorization.FindStringSubmatch(row.admission); waiver != nil {
			if strings.TrimSpace(waiver[1]) == "" {
				g.Errs = append(g.Errs, row.where+": authorization waiver names no reason; write '(no authorization: <reason>)'")
			} else {
				g.Count("authorization obligations waived")
			}
			continue
		}
		if row.admission == "" {
			g.Errs = append(g.Errs, row.where+": authorization admission is empty; name the admitting capability or waive with '(no authorization: <reason>)'")
			continue
		}
		g.Count("authorization obligations admitted")
	}
	var subjects []string
	for subject := range obligations {
		subjects = append(subjects, subject)
	}
	sort.Strings(subjects)
	for _, subject := range subjects {
		if !seen[subject] {
			g.Errs = append(g.Errs, obligations[subject]+" "+ir.Repr(subject)+" has no authorization row; admit it by exact subject or waive with '(no authorization: <reason>)'")
		}
	}
}
