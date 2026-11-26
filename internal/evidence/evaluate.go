package evidence

import (
	"fmt"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
)

// Obligation states (section 12).
const (
	Proven   = "proven"
	Partial  = "partial"
	Failed   = "failed"
	Unproven = "unproven"
)

// Undetermined is the expect value of a trace no replay can prove.
const Undetermined = "undetermined"

// FormalProven reports which fv ids the formal stage proves (section 16.9).
// It is set by the formal package; nil means none is proven.
var FormalProven func(d *design.Design) map[string]bool

// FormalCounts gives the runs and detected mutations behind a proven fv id,
// for the proof block. It is set by the formal package; nil means 0 and 0.
var FormalCounts func(d *design.Design, id string) (runs, detected int)

// Evaluation is the judgment of the evidence at the current commit.
type Evaluation struct {
	Head        string
	Dirty       bool // the working tree has changes outside receipts and owed file
	HasEvidence bool // at least one complete, usable run valid for Head
	Runs        []RunInfo
	// Findings are those of stage verify (section 12), unsorted.
	Findings []report.Finding

	Obligations int
	NReceipts   int
	NProven     int
	NPartial    int
	NFailed     int
	NOwed       int // owed entries
	NUnowed     int // unproven or partial and not owed

	d        *design.Design
	states   map[string]string
	receipts map[string][]Receipt
}

// State returns the state of an obligation id, or "" when it is not one.
func (e *Evaluation) State(id string) string { return e.states[id] }

// Design returns the design the evaluation judged.
func (e *Evaluation) Design() *design.Design { return e.d }

// Receipts returns the receipts of an id, in file and line order.
func (e *Evaluation) Receipts(id string) []Receipt { return e.receipts[id] }

// NoEvidence is the reason of a stage that found no evidence.
func (e *Evaluation) NoEvidence() string { return "no evidence at " + gitx.Short(e.Head) }

// Evaluate reads the receipts files valid for the current commit and judges
// every obligation (sections 10.1 and 12).
func Evaluate(p *project.Project, d *design.Design, allowDirty bool) (*Evaluation, error) {
	head, err := gitx.Head(p.Root)
	if err != nil {
		return nil, fmt.Errorf("cannot read the current commit (evidence needs a git repository with a commit): %v", err)
	}
	dirty, err := TreeDirty(p)
	if err != nil {
		return nil, err
	}
	ev := &Evaluation{Head: head, Dirty: dirty, d: d}
	add := func(sev report.Severity, format string, a ...any) {
		ev.Findings = append(ev.Findings, report.Finding{Sev: sev, Msg: fmt.Sprintf(format, a...)})
	}
	if dirty && !allowDirty {
		add(report.Error, "working tree has uncommitted changes")
	}
	files, err := receiptFiles(p)
	if err != nil {
		return nil, err
	}
	v := newValidity(p, head)
	var recs []Receipt
	dirtyUsed := false
	for _, rel := range files {
		fd, err := parseFile(p.Abs(rel), rel)
		if err != nil {
			return nil, err
		}
		if fd.Header == nil || !v.valid(fd.Header.Commit) {
			continue
		}
		// a file machinery run did not write, or that changed since, proves nothing (section 10.2)
		state, err := binding(p, fd)
		if err != nil {
			return nil, err
		}
		if state == notRun || state == changed {
			add(report.Error, "receipts: %s: %s", rel, state)
			continue
		}
		for _, pr := range fd.Problems {
			add(report.Error, "malformed receipts file: %s: %s", rel, pr)
		}
		if fd.End == nil {
			add(report.Error, "incomplete run: %s", rel)
			continue
		}
		if fd.Header.Dirty {
			if !allowDirty {
				continue
			}
			dirtyUsed = true
		}
		if *fd.End.Exit != 0 {
			add(report.Warn, "run exited %d: %s", *fd.End.Exit, rel)
		}
		ev.Runs = append(ev.Runs, RunInfo{File: rel, Commit: fd.Header.Commit, Command: fd.Header.Command,
			Stack: fd.Header.Stack, Exit: *fd.End.Exit, Dirty: fd.Header.Dirty, Receipts: len(fd.Receipts)})
		recs = append(recs, fd.Receipts...)
	}
	if dirtyUsed {
		add(report.Warn, "dirty evidence: not valid for acceptance")
	}
	for _, pr := range d.OwedProblems {
		add(report.Error, "%s", pr)
	}
	owed := d.OwedSet()
	ev.NOwed = len(d.Owed)
	for _, msg := range d.OwedIssues() {
		add(report.Error, "%s", msg)
	}
	ev.HasEvidence = len(ev.Runs) > 0
	ev.NReceipts = len(recs)
	ev.Obligations = len(d.Obligations)
	counted, err := claimedOnly(p, d, recs, add)
	if err != nil {
		return nil, err
	}
	j := Judge(d, counted)
	ev.states, ev.receipts = j.States, j.Receipts
	if ev.HasEvidence {
		ev.Findings = append(ev.Findings, j.Findings...)
	}
	for _, o := range d.Obligations {
		st := ev.states[o.ID]
		_, isOwed := owed[o.ID]
		switch st {
		case Proven:
			ev.NProven++
			if isOwed && ev.HasEvidence {
				add(report.Error, "stale owed entry: %s is proven", o.ID)
			}
		case Partial:
			ev.NPartial++
		case Failed:
			ev.NFailed++
		}
		if (st == Unproven || st == Partial) && !isOwed {
			ev.NUnowed++
			if ev.HasEvidence {
				add(report.Error, "unproven: %s", o.ID)
			}
		}
	}
	return ev, nil
}

// claimedOnly keeps the receipts whose test lies in a test file that claims
// their id (section 10.2) and reports the others once per id and test. A
// receipt for an id that is no obligation is kept: Judge names it.
func claimedOnly(p *project.Project, d *design.Design, recs []Receipt, add func(report.Severity, string, ...any)) ([]Receipt, error) {
	if len(recs) == 0 {
		return nil, nil
	}
	cl, err := newClaims(p, d)
	if err != nil {
		return nil, err
	}
	var res []Receipt
	seen := map[string]bool{}
	for _, r := range recs {
		if d.Obligation(r.ID) != nil && !cl.claimed(r) {
			if key := r.ID + "\x00" + r.Test; !seen[key] {
				seen[key] = true
				add(report.Error, "receipt: %s: %s does not claim it", r.ID, r.Test)
			}
			continue
		}
		res = append(res, r)
	}
	return res, nil
}

// Judgment is the state of every obligation from one set of receipts.
type Judgment struct {
	States   map[string]string
	Receipts map[string][]Receipt
	Findings []report.Finding
}

// Judge computes the state of every obligation from the receipts alone
// (section 12), plus the findings the receipts themselves give.
func Judge(d *design.Design, recs []Receipt) *Judgment {
	j := &Judgment{States: map[string]string{}, Receipts: map[string][]Receipt{}}
	seen := map[string]bool{}
	errorf := func(format string, a ...any) {
		msg := fmt.Sprintf(format, a...)
		if !seen[msg] {
			seen[msg] = true
			j.Findings = append(j.Findings, report.Finding{Sev: report.Error, Msg: msg})
		}
	}
	for _, r := range recs {
		if d.Obligation(r.ID) == nil {
			if st := d.Status(r.ID); st != "" {
				errorf("%s id in receipt: %s; a %s id yields no obligation: drop the receipt, or take the id out of %s", st, r.ID, st, d.P.Cfg.Design.Status)
			} else if d.IsProof(r.ID) {
				errorf("proof id in receipt: %s; tests do not discharge a proof obligation: drop the receipt", r.ID)
			} else {
				errorf("unknown id in receipt: %s", r.ID)
			}
			continue
		}
		j.Receipts[r.ID] = append(j.Receipts[r.ID], r)
	}
	var fproven map[string]bool
	if FormalProven != nil {
		fproven = FormalProven(d)
	}
	expect := map[string]string{}
	for _, t := range d.Traces {
		expect[t.ID] = t.Expect
	}
	for _, o := range d.Obligations {
		rs := j.Receipts[o.ID]
		var st string
		switch o.Kind {
		case "fv":
			st = Unproven
			if fproven[o.ID] {
				st = Proven
			}
		case "cx":
			st = judgeCX(o.ID, expect[o.ID], rs, errorf)
		default:
			st = judgeReceipts(o, rs, errorf)
		}
		j.States[o.ID] = st
	}
	return j
}

func judgeCX(id, expect string, rs []Receipt, errorf func(string, ...any)) string {
	failed, proven := false, false
	for _, r := range rs {
		if r.Outcome == Skip {
			continue
		}
		if r.Outcome == Fail {
			failed = true
			errorf("failed: %s (%s)", id, r.Test)
		}
		switch {
		case r.Replay == "":
		case r.Replay == expect && expect != Undetermined:
			proven = true
		default:
			failed = true
			if r.Replay == Reproduced {
				errorf("counterexample reproduced on the real system: %s: a real bug; fix the code, or record the run as a finding with a story", id)
			} else {
				errorf("counterexample did not reproduce: %s: the model is presumed wrong; fix the model, not the property", id)
			}
		}
	}
	switch {
	case failed:
		return Failed
	case proven:
		return Proven
	}
	return Unproven
}

func judgeReceipts(o design.Obligation, rs []Receipt, errorf func(string, ...any)) string {
	isTC := o.Kind == "tc"
	want := setOf(o.Effects)
	failed, pass, full := false, false, false
	for _, r := range rs {
		if r.Outcome == Skip {
			continue
		}
		if isTC {
			for _, e := range r.Effects {
				if !want[e] {
					errorf("unexpected effect: %s observed %s", o.ID, e)
				}
			}
		}
		switch r.Outcome {
		case Fail:
			failed = true
			errorf("failed: %s (%s)", o.ID, r.Test)
		case Pass:
			pass = true
			if !isTC || sameSet(setOf(r.Effects), want) {
				full = true
			}
		}
	}
	switch {
	case failed:
		return Failed
	case full:
		return Proven
	case pass:
		return Partial
	}
	return Unproven
}

// FullyObserved reports whether a pass receipt proves its obligation alone.
func FullyObserved(o *design.Obligation, r Receipt) bool {
	return r.Outcome == Pass && (o.Kind != "tc" || sameSet(setOf(r.Effects), setOf(o.Effects)))
}

func setOf(names []string) map[string]bool {
	m := map[string]bool{}
	for _, n := range names {
		m[n] = true
	}
	return m
}

func sameSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// JudgeFile judges one receipts file alone, without the validity rules: the
// probe uses it for the run of one mutation. n is the number of receipts.
func JudgeFile(d *design.Design, abs string) (j *Judgment, n int, err error) {
	fd, err := parseFile(abs, abs)
	if err != nil {
		return nil, 0, err
	}
	return Judge(d, fd.Receipts), len(fd.Receipts), nil
}
