// Ga-accept: the milestone acceptance gate. Gb-plan holds the SHAPE of the
// build plan; until now nothing held a milestone being DISCHARGED, so a
// milestone was closed by assertion and the assertion was checked by nobody.
// Ga closes that: a milestone marked "Status: closed" in the plan must have
// committed acceptance evidence, design/acceptance/M<n>.yaml, whose verdict
// is ACCEPTED, which names the commit the review ran on and lists every
// oracle id that milestone's DoD cites.
//
// The split is machinery's standing one. Deterministic here: the evidence
// exists, parses, binds to a declared milestone, carries every required
// field, covers the DoD's ids, and names the commit under review (the same
// binding discipline as Gk's input_hash and Gt's stable ids). Attested, like
// every other LLM-attested half: whether the reviewer judged well. What Ga
// proves is that an acceptance review happened, on this commit, against
// these obligations.
//
// One file per milestone: git history is the record of prior attempts, so a
// milestone never accumulates numbered rounds in the tree.

package gates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/dirscan"
	"github.com/RamXX/machinery/internal/gitcontrol"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/processcontrol"
)

// AcceptanceDirName is the committed acceptance-evidence directory under the
// design; Ga auto-activates on it.
const AcceptanceDirName = "acceptance"

// minCommitPrefix is the shortest commit prefix that binds. Git's own default
// abbreviation is 7 characters; anything shorter names too many commits to be
// evidence of anything.
const minCommitPrefix = 7

// gitHeadTimeout bounds the one subprocess the gate suite runs. A gate that
// can hang is a gate people disable.
const gitHeadTimeout = 5 * time.Second

// gitOutputLimit is enough for every read-only query below while bounding a
// corrupt or hostile git wrapper. processcontrol continues draining after the
// limit so truncation cannot deadlock the child.
const gitOutputLimit = 64 << 10

// gitCommandTimeout is a variable only so the timeout failure path can be
// exercised without making the test suite sleep for the full production
// bound. Production never changes it.
var gitCommandTimeout = gitHeadTimeout

// errReviewCommitAbsent identifies the two semantic absence cases where a
// staged acceptance run may explicitly report an unchecked binding: the
// design is outside Git, or it is inside a repository that has no commit yet.
// Operational failures must never be folded into this sentinel.
var errReviewCommitAbsent = errors.New("repository commit is semantically absent")

// commitProvenance records where the commit under review came from. It is
// printed on the checked: line, because a binding whose source the reader
// cannot trace is half an audit trail.
type commitProvenance int

const (
	commitAbsent commitProvenance = iota
	commitFromCaller
	commitFromGit
)

var (
	// acceptanceFileRe matches the one legal evidence file name.
	acceptanceFileRe = regexp.MustCompile(`^M(\d+)\.yaml$`)
	// acceptanceDateRe pins the date shape before the calendar check.
	acceptanceDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	// gitObjectNameRe pins what a derived HEAD may look like before it is
	// treated as a commit; git prints nothing else, and a gate must not bind
	// evidence to whatever a subprocess happened to write.
	gitObjectNameRe = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
	// acceptanceKeys pins the evidence schema; an unknown key is a typo that
	// would otherwise silently contribute nothing.
	acceptanceKeys = map[string]bool{
		"milestone": true, "commit": true, "verdict": true, "dod_ids": true,
		"attestations": true, "findings": true, "reviewer": true, "date": true,
		"_comment": true,
	}
	// acceptanceRequired is every field the evidence must carry, in schema
	// order. findings is required but may be empty: an empty list says the
	// reviewer found nothing, an absent key says nobody looked.
	acceptanceRequired = []string{"milestone", "commit", "verdict", "dod_ids", "attestations", "findings", "reviewer", "date"}
)

// HasAcceptanceDir reports whether the design carries the acceptance
// directory at all.
func HasAcceptanceDir(design string) bool {
	has, err := probeRealDir(design, AcceptanceDirName)
	return has || err != nil
}

// AcceptanceActive reports whether Ga auto-activates on this design: the
// acceptance directory exists, or some milestone carries the closed marker.
// Either alone is a claim that a milestone is being discharged, and a claim
// with nothing behind it is exactly what the gate is for. Read errors are
// dropped here (the gate run itself reports them on the real gate).
func AcceptanceActive(design string) bool {
	if HasAcceptanceDir(design) {
		return true
	}
	for _, doc := range planDocuments(design, NewGate("")) {
		for _, m := range doc.milestones {
			if m.closed() {
				return true
			}
		}
	}
	return false
}

// milestoneRef is one declared milestone plus the plan document declaring it.
type milestoneRef struct {
	doc string
	m   planMilestone
}

// acceptRecord is one parsed acceptance-evidence file.
type acceptRecord struct {
	label        string // "acceptance/M1.yaml", the path as the findings name it
	milestone    int    // the milestone: field
	commit       string
	verdict      string
	dodIDs       []string
	attestations []string
	findings     []string
	reviewer     string
	date         string
}

// CheckAcceptance implements Ga-accept. commit is the VCS commit under review
// (--commit or MACHINERY_COMMIT), and a supplied one binds by identity. When
// it is empty the gate defaults to the HEAD of the git repository the design
// sits in and binds by ancestry instead (see checkCommitBinding); only outside
// a repository does the binding degrade to a non-blocking note.
func CheckAcceptance(design, commit string) *Gate {
	return checkAcceptance(design, commit, false)
}

// checkAcceptance implements both the staged and final-handoff forms of Ga.
// Staged runs retain the historical, explicit non-check note when no commit
// can be derived. Final handoff is closed: it must receive a commit from the
// caller or derive a well-formed HEAD from the repository holding the design.
func checkAcceptance(design, commit string, requireCommit bool) *Gate {
	return checkAcceptanceWithGit(design, design, commit, requireCommit)
}

func checkAcceptanceWithGit(design, gitDesign, commit string, requireCommit bool) *Gate {
	g := NewGate("Ga-accept  milestone acceptance evidence")
	g.startOrder()
	reviewed, provenance := "", commitAbsent
	commitResolveFailed := false
	if requireCommit {
		var resolveErr error
		reviewed, provenance, resolveErr = resolveReviewCommitExact(gitDesign, commit)
		if resolveErr != nil {
			g.Errs = append(g.Errs, "final handoff requires a commit under review from --commit/MACHINERY_COMMIT or a readable repository HEAD: "+resolveErr.Error())
		}
	}
	if _, err := probeRealDir(design, AcceptanceDirName); err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}
	if !HasBuildDoc(design) {
		g.Errs = append(g.Errs, "no BUILD.md in the design; acceptance evidence is keyed to the build plan's milestones and this design declares none (author BUILD.md, or drop ga from the gate list)")
		return g
	}
	docs := planDocuments(design, g)
	byNum, closed := acceptanceMilestones(g, docs)
	hasDir := HasAcceptanceDir(design)
	if !hasDir && len(closed) == 0 {
		g.Errs = append(g.Errs, "no "+AcceptanceDirName+"/ directory and no milestone marked 'Status: closed' in the build plan; the acceptance gate was requested but this design has discharged no milestone (close a milestone with a 'Status: closed' line and commit "+AcceptanceDirName+"/M<n>.yaml, or drop ga from the gate list)")
		return g
	}
	g.Count("plan documents", len(docs))
	g.Count("declared milestones", len(byNum))
	g.Count("closed milestones", len(closed))

	present, records := scanAcceptanceDir(design, g)
	if hasDir && len(present) == 0 && len(closed) == 0 {
		g.Errs = append(g.Errs, AcceptanceDirName+"/ exists but holds no acceptance evidence (M<n>.yaml) and no milestone is marked 'Status: closed'; an empty check is a failure, not a pass")
		return g
	}
	g.Count("acceptance files", len(records))

	ids := acceptanceOracleIDs(design, g)
	for _, num := range sortedRecordNums(records) {
		rec := records[num]
		ref, ok := byNum[num]
		if !ok {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: names milestone M%d, which no build-plan document declares; acceptance evidence binds to a planned milestone", rec.label, num))
			continue
		}
		checkDoDCoverage(g, rec, ref, ids)
	}

	if len(closed) > 0 && !requireCommit {
		var resolveErr error
		reviewed, provenance, resolveErr = resolveReviewCommit(gitDesign, commit)
		if resolveErr != nil {
			commitResolveFailed = true
			g.Errs = append(g.Errs, "commit binding could not resolve the repository HEAD: "+resolveErr.Error())
		}
	}
	for _, num := range closed {
		ref := byNum[num]
		rec, ok := records[num]
		switch {
		case !ok:
			if present[num] {
				continue // the parse error above already blocks; do not double-report
			}
			g.Errs = append(g.Errs, fmt.Sprintf("%s: milestone M%s (%s) is marked closed but %s/M%d.yaml is not committed; closing a milestone takes committed acceptance evidence", ref.doc, ref.m.numRaw, ref.m.title, AcceptanceDirName, num))
		case rec.verdict == "REJECTED":
			g.Errs = append(g.Errs, fmt.Sprintf("%s: milestone M%s (%s) is marked closed but %s records verdict REJECTED; a rejected review does not close a milestone (reopen the milestone, or land the fixes and re-review)", ref.doc, ref.m.numRaw, ref.m.title, rec.label))
		case rec.verdict == "ACCEPTED":
			g.Count("closed milestones with accepted evidence")
			checkCommitBinding(g, gitDesign, rec, reviewed, provenance)
		}
	}
	// the provenance is stated, never the sha: the source is what a reader
	// cannot recover from the artifacts, while the sha is already in the
	// evidence file and in the mismatch finding when the binding fails. It
	// names the RULE, not the outcome; the outcome is the count beside it and
	// the findings above it.
	switch provenance {
	case commitFromCaller:
		g.CheckedExtra("commit under review supplied by --commit or MACHINERY_COMMIT; evidence commit bound by identity")
	case commitFromGit:
		g.CheckedExtra("commit under review derived from git HEAD of the repository holding the design; evidence commit bound by ancestry")
	case commitAbsent:
		if len(closed) > 0 && !requireCommit && !commitResolveFailed {
			g.Notes = append(g.Notes, "commit binding not checked: no --commit and no MACHINERY_COMMIT, and the design is not inside a git repository this binary can read; CI is expected to pass the reviewed commit")
		}
	}
	return g
}

// resolveReviewCommit resolves the commit accepted evidence binds to. It
// converts only semantic absence (outside a repository or an unborn HEAD) to
// commitAbsent; every operational/tool/output error remains blocking. An
// explicit --commit or MACHINERY_COMMIT always wins, because the caller knows
// which commit the review ran on and a local checkout may have moved since.
// Otherwise the commit defaults to the HEAD of the git repository the design
// sits in: a local run then binds as tightly as CI does, instead of printing a
// note and proving nothing on every developer machine.
func resolveReviewCommit(design, given string) (string, commitProvenance, error) {
	commit, provenance, err := resolveReviewCommitExact(design, given)
	if errors.Is(err, errReviewCommitAbsent) {
		return "", commitAbsent, nil
	}
	return commit, provenance, err
}

// resolveReviewCommitExact is the fail-closed resolver used by final handoff.
// Unlike the staged wrapper, it preserves every reason an implicit binding
// could not be established: missing tool, inaccessible directory, timeout,
// command failure, and malformed output are materially different from a
// successfully resolved commit and therefore cannot collapse to absence.
func resolveReviewCommitExact(design, given string) (string, commitProvenance, error) {
	if g := strings.TrimSpace(given); g != "" {
		return g, commitFromCaller, nil
	}
	head, err := gitHeadAtExact(design)
	if err != nil {
		return "", commitAbsent, err
	}
	return head, commitFromGit, nil
}

// The repository is resolved FROM DIR with `git -C`, never from the process
// working directory: `machinery check some/other/design` is routinely run from
// an unrelated repository, and binding a design's evidence to whatever
// repository the shell happened to sit in would be worse than not binding at
// all.
// runGitExact runs a read-only git query without erasing operational errors.
// Semantic callers inspect gitQueryError.exitCode explicitly; inability to
// run the proof never collapses to a negative repository answer.
func runGitExact(dir string, args ...string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve design directory: %w", err)
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("inspect design directory %s: %w", abs, err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("design path %s is not a directory", abs)
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", abs}, args...)...)
	// a gate reads; it never takes a lock, opens a pager, or runs a hook
	cmd.Env = gitcontrol.Environment(os.Environ())
	stdout, stderr := &acceptGitCapture{}, &acceptGitCapture{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err = processcontrol.Run(ctx, cmd)
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("git %s timed out after %s: %w", strings.Join(args, " "), gitCommandTimeout, ctx.Err())
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		return "", &gitQueryError{args: append([]string(nil), args...), exitCode: processExitCode(err), detail: detail, err: err}
	}
	if stdout.truncated || stderr.truncated {
		streams := make([]string, 0, 2)
		if stdout.truncated {
			streams = append(streams, "stdout")
		}
		if stderr.truncated {
			streams = append(streams, "stderr")
		}
		return "", fmt.Errorf("git %s exceeded the %d-byte success-output limit on %s", strings.Join(args, " "), gitOutputLimit, strings.Join(streams, " and "))
	}
	if detail := strings.TrimSpace(stderr.String()); detail != "" {
		return "", fmt.Errorf("git %s emitted stderr on success: %s", strings.Join(args, " "), detail)
	}
	return strings.TrimSpace(stdout.String()), nil
}

type gitQueryError struct {
	args     []string
	exitCode int
	detail   string
	err      error
}

type acceptGitCapture struct {
	data      []byte
	truncated bool
}

func (capture *acceptGitCapture) Write(p []byte) (int, error) {
	overflow := len(p) > gitOutputLimit-len(capture.data)
	if room := gitOutputLimit - len(capture.data); room > 0 {
		take := len(p)
		if take > room {
			take = room
		}
		capture.data = append(capture.data, p[:take]...)
	}
	capture.truncated = capture.truncated || overflow
	return len(p), nil
}

func (capture *acceptGitCapture) String() string {
	out := string(capture.data)
	if capture.truncated {
		out += fmt.Sprintf("\n[output truncated at %d bytes]\n", gitOutputLimit)
	}
	return out
}

func (err *gitQueryError) Error() string {
	prefix := "git " + strings.Join(err.args, " ") + " failed"
	if err.detail == "" {
		return prefix + ": " + err.err.Error()
	}
	return prefix + ": " + err.err.Error() + ": " + err.detail
}

func (err *gitQueryError) Unwrap() error { return err.err }

func processExitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

// gitHeadAt returns the HEAD commit of the git repository containing dir, or
// "" when dir is outside a repository, git is unavailable, or the repository
// carries no commit yet.
func gitHeadAt(dir string) string {
	head, err := gitHeadAtExact(dir)
	if err != nil {
		return ""
	}
	return head
}

func gitHeadAtExact(dir string) (string, error) {
	head, err := runGitExact(dir, "rev-parse", "HEAD")
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "not a git repository") ||
			(strings.Contains(lower, "ambiguous argument 'head'") && strings.Contains(lower, "unknown revision or path")) {
			return "", errors.Join(errReviewCommitAbsent, err)
		}
		return "", err
	}
	head = strings.ToLower(head)
	if !gitObjectNameRe.MatchString(head) {
		return "", fmt.Errorf("git rev-parse HEAD returned malformed commit %q", head)
	}
	return head, nil
}

// gitCommitOf resolves rev to a full commit object name in the repository
// containing dir, or "" when that repository holds no such commit. A leading
// dash is refused before the value reaches git: an evidence field is data, and
// data must never arrive as an option.
func gitCommitOf(dir, rev string) (string, error) {
	rev = strings.TrimSpace(rev)
	if rev == "" || strings.HasPrefix(rev, "-") {
		return "", nil
	}
	full, err := runGitExact(dir, "rev-parse", "--verify", "--quiet", rev+"^{commit}")
	if err != nil {
		var queryErr *gitQueryError
		if errors.As(err, &queryErr) && queryErr.exitCode == 1 {
			return "", nil
		}
		return "", err
	}
	full = strings.ToLower(full)
	if !gitObjectNameRe.MatchString(full) {
		return "", fmt.Errorf("git rev-parse returned malformed commit %q", full)
	}
	return full, nil
}

// gitIsAncestor reports whether ancestor is reachable from descendant, which
// includes the two being the same commit.
func gitIsAncestor(dir, ancestor, descendant string) (bool, error) {
	stdout, err := runGitExact(dir, "merge-base", "--is-ancestor", ancestor, descendant)
	if err == nil {
		if stdout != "" {
			return false, fmt.Errorf("git merge-base --is-ancestor emitted stdout on success: %s", stdout)
		}
		return true, nil
	}
	var queryErr *gitQueryError
	if errors.As(err, &queryErr) && queryErr.exitCode == 1 {
		return false, nil
	}
	return false, err
}

// acceptanceMilestones indexes every declared milestone by number and returns
// the closed ones in ascending order. Milestone numbers must be unique across
// ALL plan-bearing documents, not only within one: acceptance evidence is
// keyed by number alone, so a number declared twice makes the evidence
// ambiguous about which milestone it discharges. Gb owns duplicates inside a
// single document; this is the cross-document rule.
func acceptanceMilestones(g *Gate, docs []planDoc) (map[int]milestoneRef, []int) {
	byNum := map[int]milestoneRef{}
	var closed []int
	for _, doc := range docs {
		for _, m := range doc.milestones {
			if !m.numOK {
				continue
			}
			if prev, dup := byNum[m.num]; dup {
				if prev.doc != doc.name {
					g.Errs = append(g.Errs, fmt.Sprintf("milestone M%d is declared in both %s and %s; acceptance evidence is keyed by milestone number alone, so numbers must be unique across every plan-bearing document (renumber one of them)", m.num, prev.doc, doc.name))
				}
				continue // Gb reports the same-document duplicate
			}
			byNum[m.num] = milestoneRef{doc: doc.name, m: m}
			if m.closed() {
				closed = append(closed, m.num)
			}
		}
	}
	sort.Ints(closed)
	return byNum, closed
}

// scanAcceptanceDir reads design/acceptance/, returning which milestone
// numbers have an evidence FILE (parsed or not) and the records that parsed.
// Anything else in the directory is a finding: a gate that quietly ignores an
// unrecognized artifact teaches people to leave unrecognized artifacts.
func scanAcceptanceDir(design string, g *Gate) (map[int]bool, map[int]*acceptRecord) {
	present := map[int]bool{}
	records := map[int]*acceptRecord{}
	dir := filepath.Join(design, AcceptanceDirName)
	entries, err := dirscan.Read(dir, designInventoryMaxEntries)
	if err != nil {
		g.Errs = append(g.Errs, "cannot enumerate "+AcceptanceDirName+"/: "+err.Error())
		return present, records
	}
	indexFiles := 0
	for _, e := range entries {
		name := e.Name()
		label := AcceptanceDirName + "/" + name
		info, lerr := os.Lstat(filepath.Join(dir, name))
		if lerr != nil {
			g.Errs = append(g.Errs, label+" cannot be inspected: "+lerr.Error())
			continue
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()):
			g.Errs = append(g.Errs, label+" must be a regular evidence file; symlinks and special entries are rejected")
		case e.IsDir():
			g.Errs = append(g.Errs, label+" is a directory; acceptance evidence is one flat "+AcceptanceDirName+"/M<n>.yaml per milestone")
		case strings.EqualFold(name, "README.md"), strings.EqualFold(name, "index.md"):
			indexFiles++
		case acceptanceFileRe.MatchString(name):
			num, _ := strconv.Atoi(acceptanceFileRe.FindStringSubmatch(name)[1])
			if present[num] {
				g.Errs = append(g.Errs, fmt.Sprintf("%s duplicates milestone M%d acceptance evidence under another numeric spelling; use exactly one canonical M<n>.yaml file", label, num))
				continue
			}
			present[num] = true
			if rec := parseAcceptance(g, design, filepath.Join(dir, name), label, num); rec != nil {
				records[num] = rec
			}
		default:
			g.Errs = append(g.Errs, label+" is not acceptance evidence; the gate reads exactly "+AcceptanceDirName+"/M<n>.yaml (one per milestone; git history is the record of prior attempts)")
		}
	}
	if indexFiles > 0 {
		g.CheckedExtra(fmt.Sprintf("%d index files exempt", indexFiles))
	}
	return present, records
}

// parseAcceptance reads and validates one evidence file. nil means the file
// could not be trusted; every reason was recorded as an ERROR first.
func parseAcceptance(g *Gate, design, path, label string, fileNum int) *acceptRecord {
	data, err := readDesignFile(design, path)
	if err != nil {
		g.Errs = append(g.Errs, label+" is unreadable: "+err.Error())
		return nil
	}
	v, err := ir.LoadYAML(data)
	if err != nil {
		g.Errs = append(g.Errs, label+": invalid YAML: "+err.Error())
		return nil
	}
	root := v.AsObject()
	if root == nil {
		g.Errs = append(g.Errs, label+": not a yaml mapping (empty file?)")
		return nil
	}
	for _, k := range root.Keys() {
		if !acceptanceKeys[k] {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: unknown key %s (the evidence fields are %s)", label, ir.Repr(k), strings.Join(acceptanceRequired, ", ")))
		}
	}
	var missing []string
	for _, k := range acceptanceRequired {
		if !root.Has(k) {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: missing required field(s): %s; incomplete evidence is not evidence", label, strings.Join(missing, ", ")))
		return nil
	}

	rec := &acceptRecord{
		label:        label,
		commit:       strings.TrimSpace(root.GetString("commit")),
		verdict:      strings.TrimSpace(root.GetString("verdict")),
		reviewer:     strings.TrimSpace(root.GetString("reviewer")),
		date:         strings.TrimSpace(root.GetString("date")),
		dodIDs:       acceptStringList(g, label, root, "dod_ids"),
		attestations: acceptStringList(g, label, root, "attestations"),
		findings:     acceptStringList(g, label, root, "findings"),
	}
	ok := true
	num, numOK := acceptInt(root, "milestone")
	switch {
	case !numOK:
		g.Errs = append(g.Errs, label+": milestone must be an integer (the plan's M<n> number)")
		ok = false
	case num != fileNum:
		g.Errs = append(g.Errs, fmt.Sprintf("%s: milestone is %d but the file names M%d; the file name and the field must agree", label, num, fileNum))
		ok = false
	}
	rec.milestone = num
	switch rec.verdict {
	case "ACCEPTED", "REJECTED":
	default:
		g.Errs = append(g.Errs, fmt.Sprintf("%s: verdict is %s; it is exactly ACCEPTED or REJECTED (upper case)", label, ir.Repr(root.GetString("verdict"))))
		ok = false
	}
	if rec.commit == "" || strings.ContainsAny(rec.commit, " \t") {
		g.Errs = append(g.Errs, label+": commit must name the single VCS commit the review ran on (quote a purely numeric revision so it reads as a string)")
		ok = false
	}
	if rec.reviewer == "" {
		g.Errs = append(g.Errs, label+": reviewer must name who or what produced the review; evidence without provenance is anonymous")
		ok = false
	}
	if !acceptanceDateRe.MatchString(rec.date) {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: date %s is not YYYY-MM-DD", label, ir.Repr(rec.date)))
		ok = false
	} else if _, derr := time.Parse("2006-01-02", rec.date); derr != nil {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: date %s is not a real calendar date", label, ir.Repr(rec.date)))
		ok = false
	}
	if rec.verdict == "ACCEPTED" && len(rec.attestations) == 0 {
		g.Errs = append(g.Errs, label+": an ACCEPTED verdict with no attestations attests nothing; list what the review checked by judgment")
		ok = false
	}
	if !ok {
		return nil
	}
	return rec
}

// acceptStringList reads a list-of-strings field. An absent or null value is
// an empty list (the required-field check owns absence); a non-list, or an
// entry that is not a non-empty string, is a finding.
func acceptStringList(g *Gate, label string, o *ir.Object, key string) []string {
	v := o.Get2(key)
	if v == nil || v.Kind == ir.KindNull {
		return nil
	}
	arr := v.AsArray()
	if arr == nil {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: %s must be a list of strings", label, key))
		return nil
	}
	var out []string
	for i, e := range arr {
		s := ""
		if e != nil && e.Kind == ir.KindString {
			s = strings.TrimSpace(e.AsString())
		}
		if s == "" {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: %s[%d] is not a non-empty string", label, key, i))
			continue
		}
		out = append(out, s)
	}
	return out
}

// acceptInt reads an integer field.
func acceptInt(o *ir.Object, key string) (int, bool) {
	v := o.Get2(key)
	if v == nil || v.Kind != ir.KindNumber {
		return 0, false
	}
	n, ok := v.Data.(json.Number)
	if !ok {
		return 0, false
	}
	i, err := strconv.Atoi(n.String())
	if err != nil {
		return 0, false
	}
	return i, true
}

// acceptanceOracleIDs is the id corpus the DoD-coverage rule matches against:
// both id columns of every committed transition oracle plus the relational
// decision oracles Gt also holds (Policy, Isolation). The committed files are
// the source; G3, Gp, and Gn hold them fresh.
func acceptanceOracleIDs(design string, g *Gate) []string {
	ids := planOracleIDs(design, g)
	for _, name := range formalOracleNames {
		path := filepath.Join(design, "formal", name)
		if fi, err := os.Stat(path); err != nil || fi.IsDir() {
			continue // the relational layers are opt-in; Gp/Gn own their health
		}
		testIDs, stableIDs := oracleTableIDs(readDesignFileOrErr(design, path, g))
		ids = append(ids, testIDs...)
		ids = append(ids, stableIDs...)
	}
	return ids
}

// checkDoDCoverage holds the evidence to the obligations, in both directions:
// every committed oracle id the milestone's DoD cites whole-token must appear
// in dod_ids, and every dod_ids entry must resolve to a committed oracle id.
// That is the deterministic proof the review looked at the right rows, the
// same way Gk's input_hash proves a verdict covered the right design. The
// reverse direction is the half that was missing: a list that resolves to
// nothing reads as coverage and proves nothing.
func checkDoDCoverage(g *Gate, rec *acceptRecord, ref milestoneRef, ids []string) {
	listed := map[string]bool{}
	for _, id := range rec.dodIDs {
		listed[id] = true
	}
	committed := map[string]bool{}
	for _, id := range ids {
		committed[id] = true
	}
	for _, id := range rec.dodIDs {
		if committed[id] {
			continue
		}
		g.Errs = append(g.Errs, fmt.Sprintf("%s: dod_ids names %s, which no committed oracle declares (neither a test id nor a stable id of machines/*.oracle.md or the relational decision oracles); a typo, or an id a regeneration left behind, binds the evidence to no obligation at all", rec.label, ir.Repr(id)))
	}
	dod := ref.m.dodText()
	seen := map[string]bool{}
	var cited []string
	for _, id := range ids {
		if seen[id] || !idTokenIn(id, dod) {
			continue
		}
		seen[id] = true
		cited = append(cited, id)
	}
	bound := 0
	for _, id := range cited {
		if listed[id] {
			bound++
			continue
		}
		g.Errs = append(g.Errs, fmt.Sprintf("%s: dod_ids omits %s, which milestone M%d's DoD in %s cites; the evidence must list every committed oracle id its DoD names", rec.label, ir.Repr(id), rec.milestone, ref.doc))
	}
	if bound > 0 {
		g.Count("DoD ids bound", bound)
	}
}

// checkCommitBinding holds accepted evidence to the commit under review. The
// rule depends on where that commit came from, and the two modes answer two
// different questions.
//
// EXPLICIT (--commit or MACHINERY_COMMIT): identity. The caller named the one
// commit the review is being judged against, so the evidence must name it too.
// This is CI's contract and it does not move: on a pull request the commit
// under review is the head commit, and a merge commit that did not exist when
// the review ran does not bind, deliberately.
//
// DERIVED (git HEAD of the design's repository): ancestry. Nobody named a
// commit; the gate went looking for one, and the honest question it can ask of
// an arbitrary local checkout is not "was the review run on exactly this
// commit" (it never is: the commit that ADDS the evidence file already has a
// different sha than the commit the evidence names, so identity here would go
// red one commit later and stay red) but "is the reviewed commit part of this
// history at all". A sha that resolves to nothing, or to a commit on a branch
// this history never took, is caught on every local run and at stop time,
// which is the hole the note tier left open.
func checkCommitBinding(g *Gate, design string, rec *acceptRecord, commit string, prov commitProvenance) {
	if commit == "" {
		return
	}
	if prov == commitFromGit {
		checkCommitAncestry(g, design, rec, commit)
		return
	}
	if !commitBinds(rec.commit, commit) {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: commit %s does not name the commit under review (%s); the review was performed on a different commit (an exact match, or either value an unambiguous prefix of the other of at least %d characters, binds)", rec.label, ir.Repr(rec.commit), ir.Repr(commit), minCommitPrefix))
		return
	}
	g.Count("commit bindings verified")
}

// checkCommitAncestry is the derived mode's rule: the evidence commit must
// resolve to a commit this repository holds, and that commit must be reachable
// from HEAD (equal counts). Both failures are ERRORs, because both mean the
// evidence names something this tree cannot account for.
func checkCommitAncestry(g *Gate, design string, rec *acceptRecord, head string) {
	full, err := gitCommitOf(design, rec.commit)
	if err != nil {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: could not resolve evidence commit %s in the repository holding the design: %v", rec.label, ir.Repr(rec.commit), err))
		return
	}
	if full == "" {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: commit %s names no commit in the repository holding the design (HEAD is %s); a reviewed commit that the history does not contain is a typo, a fabrication, or evidence from another repository", rec.label, ir.Repr(rec.commit), ir.Repr(head)))
		return
	}
	isAncestor, err := gitIsAncestor(design, full, head)
	if err != nil {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: could not prove whether commit %s is an ancestor of HEAD %s: %v", rec.label, ir.Repr(rec.commit), ir.Repr(head), err))
		return
	}
	if !isAncestor {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: commit %s is not an ancestor of the commit under review (HEAD is %s); the review ran on a commit this history never took (an unmerged branch, or a rewritten one), so it says nothing about this tree (pass --commit to bind a specific commit by identity instead)", rec.label, ir.Repr(rec.commit), ir.Repr(head)))
		return
	}
	g.Count("commit bindings verified")
}

// commitBinds reports whether an evidence commit names the commit under
// review: an exact match, or either value an unambiguous prefix of the other.
// Prefixes shorter than minCommitPrefix never bind; they name too many
// commits to be evidence.
func commitBinds(evidence, given string) bool {
	e := strings.ToLower(strings.TrimSpace(evidence))
	got := strings.ToLower(strings.TrimSpace(given))
	switch {
	case e == "" || got == "":
		return false
	case e == got:
		return true
	case len(e) < minCommitPrefix || len(got) < minCommitPrefix:
		return false
	case len(e) < len(got):
		return strings.HasPrefix(got, e)
	default:
		return strings.HasPrefix(e, got)
	}
}

// sortedRecordNums returns the evidence milestone numbers in ascending
// order, so every finding order is deterministic.
func sortedRecordNums(m map[int]*acceptRecord) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
