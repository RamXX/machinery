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
	authorizationMarker     = regexp.MustCompile(`<!--\s*machinery:authorization-inventory\s*-->`)
	noAuthorization         = regexp.MustCompile(`\(no authorization:\s*([^)]*)\)`)
	authorizationCapability = regexp.MustCompile("^`([A-Za-z][A-Za-z0-9_]*(?:\\.[A-Za-z][A-Za-z0-9_]*)+)`$")
	authorizationSubject    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(?:\.[A-Za-z][A-Za-z0-9_]*)*$`)
	h2ResourceCell          = regexp.MustCompile("^\\s*`([A-Za-z][A-Za-z0-9_]*)`(?:\\s|$)")
	h2MachineWritten        = regexp.MustCompile(`\bMACHINE-WRITTEN\{([^}]*)\}`)
	h2MachineWrittenBy      = regexp.MustCompile(`\bMACHINE-WRITTEN-BY\{([^}]*)\}`)
	c4OwnerDeclaration      = regexp.MustCompile(`(?m)^\s*([A-Za-z][A-Za-z0-9_]*)\s*=\s*(?:softwareSystem|container|component)\b`)
)

type authorizationRow struct {
	subject, admission, where string
}

// h2AuthorizationInventory reads declarations only from the two tables that
// carry H2's compile-time inventory. Mentions in explanatory prose do not
// grant an action. A producer mark is an admission for its named producers,
// never a name-wide machine-written grant.
func h2AuthorizationInventory(g *Gate, design string) (map[string]bool, bool) {
	admitted := map[string]bool{}
	nameAdmitted := map[string]bool{}
	producerNarrowed := map[string]bool{}
	found := false
	paths, _ := strictSortedGlob(g, filepath.Join(design, "machines"), "*.matrix.md", "authorization matrix")
	for _, path := range paths {
		body, err := readDesignFile(design, path)
		if err != nil {
			continue
		}
		for _, table := range ir.ParseMdTables(string(body)) {
			ri := ir.FindCol(table.Header, "resource")
			if ri < 0 {
				continue
			}
			wi := ir.FindCol(table.Header, "machine-written actions")
			residual := wi < 0 && ir.FindCol(table.Header, "platform_admin") >= 0 && ir.FindCol(table.Header, "tenant_admin") >= 0 && ir.FindCol(table.Header, "every other preset") >= 0
			if wi < 0 && !residual {
				continue
			}
			for i, row := range table.Rows {
				resourceCell := cellAt(row, ri)
				match := h2ResourceCell.FindStringSubmatch(resourceCell)
				if match == nil {
					g.Errs = append(g.Errs, filepath.Base(path)+": resource row "+strconv.Itoa(i+1)+": inventory resource cell must start with one backticked resource")
					continue
				}
				resource := match[1]
				if wi >= 0 {
					groups := h2MachineWritten.FindAllStringSubmatch(cellAt(row, wi), -1)
					if len(groups) > 0 {
						found = true
					}
					if len(groups) != 1 {
						g.Errs = append(g.Errs, filepath.Base(path)+": resource "+resource+": inventory row needs exactly one MACHINE-WRITTEN mark")
					}
					for _, group := range groups {
						for _, raw := range strings.Split(group[1], ",") {
							action := strings.TrimSpace(raw)
							if !valueMember.MatchString(action) {
								g.Errs = append(g.Errs, filepath.Base(path)+": resource "+resource+": invalid MACHINE-WRITTEN action "+ir.Repr(action))
								continue
							}
							nameAdmitted[resource+"."+action] = true
						}
					}
				}
				if residual {
					byGroups := h2MachineWrittenBy.FindAllStringSubmatch(resourceCell, -1)
					if len(byGroups) > 0 {
						found = true
					}
					for _, group := range byGroups {
						tokens := strings.Split(group[1], ",")
						first := strings.Split(tokens[0], ":")
						if len(first) != 2 || !valueMember.MatchString(strings.TrimSpace(first[0])) || !valueMember.MatchString(strings.TrimSpace(first[1])) {
							g.Errs = append(g.Errs, filepath.Base(path)+": resource "+resource+": invalid MACHINE-WRITTEN-BY entry "+ir.Repr(tokens[0]))
							continue
						}
						valid := true
						for _, producer := range tokens[1:] {
							if strings.Contains(producer, ":") {
								g.Errs = append(g.Errs, filepath.Base(path)+": resource "+resource+": MACHINE-WRITTEN-BY names one action followed by producers")
								valid = false
							} else if !valueMember.MatchString(strings.TrimSpace(producer)) {
								g.Errs = append(g.Errs, filepath.Base(path)+": resource "+resource+": invalid MACHINE-WRITTEN-BY entry "+ir.Repr(producer))
								valid = false
							}
						}
						if valid {
							producerNarrowed[resource+"."+strings.TrimSpace(first[0])] = true
						}
					}
				}
			}
		}
	}
	for subject := range nameAdmitted {
		admitted[subject] = true
	}
	for subject := range producerNarrowed {
		if nameAdmitted[subject] {
			g.Errs = append(g.Errs, subject+": both name-admitted and producer-narrowed in H2 authorization inventory")
			continue
		}
		admitted[subject] = true
	}
	return admitted, found
}

type authorizationDocument struct {
	path, text string
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
			pi := ir.FindCol(table.Header, "producer")
			if pi < 0 || strings.Contains(strings.ToLower(ir.CleanCell(table.Header[pi])), "producer / consumer") ||
				(ir.FindCol(table.Header, "cascade") < 0 && ir.FindCol(table.Header, "consumer") < 0) {
				continue
			}
			for i, row := range table.Rows {
				if subject := ir.CleanCell(cellAt(row, pi)); subject != "" {
					if !authorizationSubject.MatchString(subject) {
						g.Errs = append(g.Errs, filepath.Base(path)+": producer row "+strconv.Itoa(i+1)+": producer cell must name one subject, got "+ir.Repr(subject))
						continue
					}
					out[subject] = "matrix producer"
				}
			}
		}
	}
	return out
}

func collectAuthorizationRows(g *Gate, documents []authorizationDocument) []authorizationRow {
	var out []authorizationRow
	for _, document := range documents {
		lines := strings.Split(document.text, "\n")
		lineAt := 0
		for _, table := range ir.ParseMdTables(document.text) {
			si := colContaining(table.Header, "authorization subject")
			ai := colContaining(table.Header, "admission")
			if si < 0 && ai < 0 {
				continue
			}
			if si < 0 || ai < 0 {
				g.Errs = append(g.Errs, document.path+": authorization inventory table must have authorization subject and admission columns")
				continue
			}
			for i, row := range table.Rows {
				for lineAt < len(lines) && strings.TrimSpace(lines[lineAt]) != strings.TrimSpace(table.RowLines[i]) {
					lineAt++
				}
				subject := ir.CleanCell(cellAt(row, si))
				out = append(out, authorizationRow{
					subject: subject, admission: strings.TrimSpace(cellAt(row, ai)),
					where: document.path + ":" + strconv.Itoa(lineAt+1) + " authorization row " + strconv.Itoa(i+1),
				})
				lineAt++
			}
		}
	}
	return out
}

func authorizationInventoryCorpus(g *Gate, design string) ([]authorizationDocument, int) {
	var documents []authorizationDocument
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
		documents = append(documents, authorizationDocument{path: filepath.ToSlash(rel), text: string(body)})
		return nil
	})
	if err != nil {
		g.Errs = append(g.Errs, "authorization inventory scan failed: "+err.Error())
	}
	return documents, markers
}

func checkAuthorizationInventory(g *Gate, design string, dm *ir.Value) {
	obligations := authorizationObligations(g, design, dm)
	if len(obligations) == 0 {
		return
	}
	documents, markers := authorizationInventoryCorpus(g, design)
	h2Admitted, h2Found := h2AuthorizationInventory(g, design)
	if markers == 0 && !h2Found {
		g.Errs = append(g.Errs, "System writes exist but the design has no <!-- machinery:authorization-inventory --> marker and closed authorization inventory")
	} else if markers > 1 {
		g.Errs = append(g.Errs, "authorization inventory marker appears "+strconv.Itoa(markers)+" times; one design has exactly one closed authorization inventory")
	}
	rows := collectAuthorizationRows(g, documents)
	c4Owners := map[string]bool{}
	if body, err := readDesignFile(design, filepath.Join(design, "workspace.dsl")); err == nil {
		for _, match := range c4OwnerDeclaration.FindAllStringSubmatch(string(body), -1) {
			c4Owners[match[1]] = true
		}
	}
	if markers > 0 && len(rows) == 0 {
		g.Errs = append(g.Errs, "authorization inventory marker has no table with authorization subject and admission columns")
	}
	seen := map[string]bool{}
	for subject := range h2Admitted {
		if _, ok := obligations[subject]; ok {
			seen[subject] = true
			g.Count("authorization obligations admitted")
		}
	}
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
		if !authorizationCapability.MatchString(row.admission) {
			g.Errs = append(g.Errs, row.where+": admission must name one capability as a backticked dotted identifier or waive with '(no authorization: <reason>)'")
			continue
		}
		owner := strings.SplitN(strings.Trim(row.admission, "`"), ".", 2)[0]
		if !c4Owners[owner] {
			g.Errs = append(g.Errs, row.where+": admission owner "+ir.Repr(owner)+" is not a C4 element in workspace.dsl")
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
