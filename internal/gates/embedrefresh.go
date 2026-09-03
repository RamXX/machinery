// `machinery embed refresh`: the WRITE half of Ge-embed.
//
// Ge holds a declared copy byte-identical to its source and says which rows
// drifted. Fixing them was hand work: an author re-copied 743 rows across
// seven shards with an editor, which is exactly the labour the marker exists
// to make unnecessary and exactly the place a copy silently re-drifts. This
// command does the copy the gate checks.
//
// The contract is deliberately narrow, because a rewriter of hand-authored
// documents earns trust only by doing less than it could:
//
//   - a row is matched to its source row by KEY, not by position, so a
//     reordered or newly inserted source row still lands on the right copy;
//   - a row carrying '(shard-local: ...)' anywhere is left exactly as it is,
//     localization being the author's explicit statement that this row (or a
//     cell of it) is NOT the source's;
//   - a row with no source row is REPORTED and kept, never deleted: a rename
//     or a retirement is a judgment, and a tool that silently drops rows is
//     one an author cannot leave running;
//   - only a `complete` claim with a `where=` selector appends, and only the
//     selected source rows the copy lacks, because that is the one case where
//     the set of rows that BELONG here is mechanically decidable.
//
// Deterministic (files in sorted order, source rows in source order) and
// idempotent: the second run of a pair changes nothing.

package gates

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/designlock"
	"github.com/RamXX/machinery/internal/filelock"
	"github.com/RamXX/machinery/internal/ir"
)

func embedRefreshLockScope(design string) string {
	return filepath.Join(design, ".machinery-embed-refresh")
}

const (
	embedRefreshWriter   = "embed-refresh"
	embedRefreshRecovery = "rerun `machinery embed refresh` with the same arguments"
)

var embedRefreshAfterWorkspaceSnapshot = func() {}

// leadTokenRe matches a cell's leading backticked token, the key most design
// tables carry in their first column.
var leadTokenRe = regexp.MustCompile("^\\s*`([^`]+)`")

// bareTokenRe matches a cell's leading bare identifier, for a table whose key
// column is not backticked.
var bareTokenRe = regexp.MustCompile(`^\s*([A-Za-z0-9_.-]+)`)

// leadToken returns the token a cell is keyed by: its leading backticked
// token, else its leading bare identifier, else the whole trimmed cell.
func leadToken(cell string) string {
	if m := leadTokenRe.FindStringSubmatch(cell); m != nil {
		return m[1]
	}
	if m := bareTokenRe.FindStringSubmatch(cell); m != nil {
		return m[1]
	}
	return strings.TrimSpace(cell)
}

// keyPrefixLen bounds how much of the participant cell joins an event key.
const keyPrefixLen = 60

// embedRowKey computes the identity of a table row under the marker's
// declared columns. Most tables are unique on their first cell's leading
// token. An EVENT table is not: one event has a row per producer-consumer
// pair, and several rows can share an event and a consumer boundary (an SSE
// lane and a durable lane), so the key is the leading tokens of the first
// three columns plus the opening text of the third, which is what separates
// the lanes.
func embedRowKey(cells []string, cols []string) string {
	if len(cells) == 0 {
		return ""
	}
	if len(cols) > 0 && strings.EqualFold(strings.TrimSpace(cols[0]), "event") && len(cells) >= 3 {
		third := strings.TrimSpace(cells[2])
		if len(third) > keyPrefixLen {
			third = third[:keyPrefixLen]
		}
		return strings.Join([]string{leadToken(cells[0]), leadToken(cells[1]), leadToken(cells[2]), third}, "\x00")
	}
	return leadToken(cells[0])
}

// RefreshReport is one marker's outcome.
type RefreshReport struct {
	File      string   // design-relative path of the embedding document
	From      string   // the marker's source path, as written
	Table     string   // the marker's table selector
	Recopied  int      // rows whose text the source changed
	Appended  int      // selected source rows the copy lacked (complete + where)
	Kept      int      // rows left alone because they are localized
	Orphans   []string // keys with no source row: a rename or a retired row
	Ambiguous []string // keys several source rows share: which one is a judgment
	Problem   string   // why this marker was skipped ("" when it was refreshed)

	whereSel string // the marker's where= selector
	complete bool   // whether the marker claims completeness
}

// RefreshEmbeds re-copies every embed table under design from its source.
// With dryRun it computes and reports exactly the same thing and writes
// nothing. Returns one report per marker, in file order, plus the set of
// files it changed (empty under dryRun; the files it WOULD change are the
// ones with a non-zero Recopied or Appended).
func RefreshEmbeds(design string, dryRun bool) (reports []RefreshReport, changed []string, retErr error) {
	acquire := designlock.Acquire
	if dryRun {
		acquire = designlock.AcquireReader
	}
	snapshot, err := acquire(design)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err := snapshot.Release(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("release design snapshot lock: %w", err))
		}
		retErr = snapshot.LogicalError(retErr)
	}()
	if !dryRun {
		// The inner journal is deliberately removed before PublishExpected
		// performs its final validation and clears the outer sentinel. If the
		// process dies in that last interval, resume the persisted exact output
		// contract before discovery can mistake the completed outputs for a
		// no-op and strand the sentinel forever. Live inner residue is left for
		// the rooted transaction recovery below.
		if err := snapshot.ResumeExpected(embedRefreshWriter, embedRefreshRecovery); err != nil {
			return nil, nil, fmt.Errorf("resume finalized embed refresh publication: %w", err)
		}
	}
	lock, err := filelock.Acquire(embedRefreshLockScope(design))
	if err != nil {
		return nil, nil, fmt.Errorf("acquire design-wide embed refresh lock: %w", err)
	}
	defer func() {
		if err := lock.Release(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("release design-wide embed refresh lock: %w", err))
		}
	}()
	txRoot, err := snapshot.OpenRoot(design)
	if err != nil {
		return nil, nil, err
	}
	tx, err := openEmbedRootTransactionRetained(design, txRoot)
	if err != nil {
		_ = txRoot.Close()
		return nil, nil, err
	}
	defer func() {
		if err := tx.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("close rooted embed refresh transaction: %w", err))
		}
	}()
	if journal, pending, err := tx.pending(); err != nil {
		return nil, nil, err
	} else if pending {
		expected := make([]designlock.OutputExpectation, 0, len(journal.Items))
		for _, item := range journal.Items {
			target := filepath.Join(design, filepath.FromSlash(item.Path))
			if item.Deletion {
				expected = append(expected, designlock.ExpectAbsent(target))
			} else {
				expected = append(expected, designlock.ExpectFile(target, item.New, os.FileMode(item.NewMode)))
			}
			changed = append(changed, item.Path)
		}
		if dryRun {
			return nil, nil, fmt.Errorf("interrupted embed refresh transaction requires an exact non-dry-run retry before reads")
		}
		if err := snapshot.PublishExpectedRooted(embedRefreshWriter, embedRefreshRecovery, expected, func(outputs *designlock.OutputScope) error {
			return outputs.WithRoot(design, func(root *os.Root) error {
				return tx.withRoot(root, func() error { return tx.recover(journal) })
			})
		}); err != nil {
			return nil, nil, err
		}
		return nil, changed, nil
	}
	workspace, err := snapshot.MaterializeDesignWorkspace()
	if err != nil {
		return nil, nil, fmt.Errorf("materialize retained embed workspace: %w", err)
	}
	defer func() {
		if err := workspace.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("remove retained embed workspace: %w", err))
		}
	}()
	sourceDesign := workspace.Path()
	embedRefreshAfterWorkspaceSnapshot()
	files, err := markdownFiles(sourceDesign)
	if err != nil {
		return nil, nil, err
	}
	type plannedDoc struct {
		path string
		old  []byte
		new  []byte
		mode uint32
	}
	var planned []plannedDoc
	for _, path := range files {
		body, err := readDesignFile(sourceDesign, path)
		if err != nil {
			return nil, nil, err
		}
		rel, rerr := filepath.Rel(sourceDesign, path)
		if rerr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		out, rs := refreshOneDoc(sourceDesign, rel, path, string(body))
		reports = append(reports, rs...)
		targetPath := filepath.Join(design, filepath.FromSlash(rel))
		if out == string(body) {
			continue
		}
		info, statErr := os.Lstat(path)
		if statErr != nil {
			return nil, nil, statErr
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, nil, fmt.Errorf("embed target %s must be a regular file", rel)
		}
		changed = append(changed, rel)
		planned = append(planned, plannedDoc{path: targetPath, old: body, new: []byte(out), mode: uint32(info.Mode().Perm())})
	}
	if dryRun || len(planned) == 0 {
		if err := snapshot.CheckUnchanged(); err != nil {
			return nil, nil, err
		}
		return reports, changed, nil
	}

	items := make([]embedTxItem, 0, len(planned))
	expected := make([]designlock.OutputExpectation, 0, len(planned))
	for _, p := range planned {
		rel, err := filepath.Rel(design, p.path)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, embedTxItem{Path: filepath.ToSlash(rel), Old: p.old, New: p.new, OldMode: p.mode, NewMode: p.mode})
		expected = append(expected, designlock.ExpectFile(p.path, p.new, os.FileMode(p.mode)))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	journal, err := newEmbedTxJournal(tx.scope, items)
	if err != nil {
		return nil, nil, err
	}
	publishErr := snapshot.PublishExpectedRooted(embedRefreshWriter, embedRefreshRecovery, expected, func(outputs *designlock.OutputScope) error {
		return outputs.WithRoot(design, func(root *os.Root) error {
			return tx.withRoot(root, func() error { return tx.commit(journal) })
		})
	})
	if publishErr != nil {
		return nil, nil, publishErr
	}
	return reports, changed, nil
}

// refreshOneDoc rewrites every marked table in one document, returning the
// new text and one report per marker.
func refreshOneDoc(design, rel, path, text string) (string, []RefreshReport) {
	lines := strings.Split(text, "\n")
	var out []string
	var reports []RefreshReport
	for i := 0; i < len(lines); {
		m := embedMarker.FindStringSubmatch(lines[i])
		if m == nil {
			out = append(out, lines[i])
			i++
			continue
		}
		out = append(out, lines[i])
		i++
		rep := RefreshReport{File: rel}
		cols := parseMarkerAttrs(m[1], &rep)
		// pass through whatever sits between the marker and its table
		for i < len(lines) && !isTableLine(lines[i]) {
			out = append(out, lines[i])
			i++
		}
		start := i
		for i < len(lines) && isTableLine(lines[i]) {
			i++
		}
		block := lines[start:i]
		if rep.Problem == "" && len(block) < 2 {
			rep.Problem = "the marker marks no table"
		}
		if rep.Problem != "" {
			out = append(out, block...)
			reports = append(reports, rep)
			continue
		}
		refreshed := refreshOneTable(design, path, block, cols, &rep)
		out = append(out, refreshed...)
		reports = append(reports, rep)
	}
	return strings.Join(out, "\n"), reports
}

// parseMarkerAttrs reads a marker's attributes into the report and returns
// its declared column list.
func parseMarkerAttrs(raw string, rep *RefreshReport) []string {
	var where, claims string
	for _, a := range embedAttr.FindAllStringSubmatch(raw, -1) {
		switch a[1] {
		case "from":
			rep.From = a[2]
		case "table":
			rep.Table = a[2]
		case "where":
			where = a[2]
		case "claims":
			claims = a[2]
		}
	}
	rep.whereSel, rep.complete = where, strings.Contains(claims, "complete")
	if rep.From == "" || rep.Table == "" {
		rep.Problem = "the marker is missing from= or table=; Ge reports it"
	}
	var cols []string
	for _, c := range strings.Split(rep.Table, ",") {
		cols = append(cols, strings.TrimSpace(c))
	}
	return cols
}

// isTableLine reports whether a line belongs to a markdown table block, the
// same rule ir.ParseMdTables blocks on.
func isTableLine(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " \t"), "|")
}

// refreshOneTable rewrites one marked table block (header, separator, rows)
// against its source, recording the outcome in rep.
func refreshOneTable(design, docPath string, block, cols []string, rep *RefreshReport) []string {
	srcPath, resolveErr := resolveRetainedDesignReference(design, docPath, rep.From)
	if resolveErr != nil {
		rep.Problem = resolveErr.Error() + "; Ge reports it"
		return block
	}
	srcBody, err := readRegularFile(srcPath)
	if err != nil || len(srcBody) == 0 {
		rep.Problem = "source " + ir.Repr(rep.From) + " does not exist or is empty; Ge reports it"
		return block
	}
	srcText := string(srcBody)
	var matched []ir.MdTable
	for _, t := range ir.ParseMdTables(srcText) {
		if headerHasAll(t.Header, cols) {
			matched = append(matched, t)
		}
	}
	if len(matched) != 1 {
		rep.Problem = "the table selector resolves to " + plural(len(matched), "source table") + ", not one; Ge reports it"
		return block
	}
	src := matched[0]
	// the copy's own header must be the source's, or a row-for-row copy is
	// meaningless. Ge already errors on that; here it means DO NOT WRITE.
	embedTbls := ir.ParseMdTables(strings.Join(block, "\n"))
	if len(embedTbls) != 1 || !sameHeader(embedTbls[0].Header, src.Header) {
		rep.Problem = "the embedded table does not carry the source's columns; Ge reports it"
		return block
	}
	// A source row is CONSUMED once some copied row stands for it. Matching
	// runs in two passes so an ambiguous key can never silently overwrite a
	// row with its neighbour's text: exact copies claim their source row
	// first, and only what is left is matched by key.
	srcKeys := make([]string, len(src.Rows))
	for i, cells := range src.Rows {
		srcKeys[i] = embedRowKey(cells, cols)
	}
	consumed := make([]bool, len(src.Rows))
	// claimExact reports whether a copied row is ALREADY a byte copy of a
	// source row, consuming that source row when it is still unclaimed. A row
	// matching only source rows another copied row already claimed (a
	// duplicated row in the copy) is still a byte copy and is left alone:
	// de-duplicating a table is a judgment, not a re-copy.
	claimExact := func(row string) bool {
		exact := false
		for i, line := range src.RowLines {
			if line != row {
				continue
			}
			if !consumed[i] {
				consumed[i] = true
				return true
			}
			exact = true
		}
		return exact
	}
	claimKeyed := func(k string) []int {
		var out []int
		for i := range src.RowLines {
			if !consumed[i] && srcKeys[i] == k {
				out = append(out, i)
			}
		}
		return out
	}
	// header + separator pass through untouched
	head := 1
	if len(block) > 1 && isSeparatorRow(block[1]) {
		head = 2
	}
	rows := block[head:]
	out := append([]string{}, block[:head]...)
	// pass 1: rows that are already byte-identical to a source row, and rows
	// the author localized (which are never rewritten, but do stand for their
	// source row, so the completeness pass must not append it beside them).
	settled := make([]bool, len(rows))
	for ri, row := range rows {
		trimmed := strings.TrimSpace(row)
		if strings.Contains(row, "(shard-local:") {
			rep.Kept++
			settled[ri] = true
			if !claimExact(trimmed) {
				if c := claimKeyed(embedRowKey(splitTableRow(row), cols)); len(c) > 0 {
					consumed[c[0]] = true
				}
			}
			continue
		}
		if claimExact(trimmed) {
			settled[ri] = true
		}
	}
	// pass 2: everything else, matched by key against what is left
	result := make([]string, len(rows))
	copy(result, rows)
	for ri, row := range rows {
		if settled[ri] {
			continue
		}
		k := embedRowKey(splitTableRow(row), cols)
		switch c := claimKeyed(k); len(c) {
		case 1:
			consumed[c[0]] = true
			result[ri] = src.RowLines[c[0]]
			rep.Recopied++
		case 0:
			rep.Orphans = append(rep.Orphans, k)
		default:
			// several source rows carry this key and none of them is a byte
			// copy of this row: which one this row means is a judgment, and
			// guessing would write a neighbour's text over it.
			rep.Ambiguous = append(rep.Ambiguous, k)
		}
	}
	out = append(out, result...)
	if rep.complete && rep.whereSel != "" {
		picked, err := filterRowIdx(src, rep.whereSel)
		if err != "" {
			rep.Problem = err + "; Ge reports it"
			return block
		}
		for _, i := range picked {
			if consumed[i] {
				continue
			}
			consumed[i] = true
			rep.Appended++
			out = append(out, src.RowLines[i])
		}
	}
	return out
}

// sameHeader reports cell-for-cell header equality.
func sameHeader(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// isSeparatorRow reports whether a table line is the `|---|---|` separator.
func isSeparatorRow(line string) bool {
	body := strings.TrimSpace(line)
	if !strings.HasPrefix(body, "|") {
		return false
	}
	for _, c := range body {
		if c != '|' && c != '-' && c != ':' && c != ' ' {
			return false
		}
	}
	return true
}

// plural renders "1 source table" / "3 source tables" for a finding.
func plural(n int, noun string) string {
	s := noun
	if n != 1 {
		s += "s"
	}
	return strconv.Itoa(n) + " " + s
}

// SortReports orders reports by file then source, so a run's output is the
// same on every filesystem.
func SortReports(rs []RefreshReport) {
	sort.SliceStable(rs, func(i, j int) bool {
		if rs[i].File != rs[j].File {
			return rs[i].File < rs[j].File
		}
		return rs[i].From < rs[j].From
	})
}
