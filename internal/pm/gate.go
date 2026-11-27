package pm

import (
	"sort"
	"strings"

	"github.com/RamXX/machinery/v2/internal/backlog"
	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/evidence"
	"github.com/RamXX/machinery/v2/internal/formal"
	"github.com/RamXX/machinery/v2/internal/gitx"
	"github.com/RamXX/machinery/v2/internal/report"
)

// StoryGate is stage story (section 15.6): the tier gate of one story at the
// current commit. issues is the backlog, read for a capstone (its ids and
// its siblings are the other stories of its intake); nil otherwise.
func StoryGate(d *design.Design, is *backlog.Issue, issues []backlog.Issue) (*report.Stage, error) {
	s := report.New("story")
	t := backlog.ParseTags(is)
	if t.Intake {
		s.Errorf("%s is an intake item: an intake item is planned, not built; run machinery intake status %s", is.ID, is.ID)
		return s, nil
	}
	ids := t.IDs
	var siblings []backlog.Issue
	if t.Capstone {
		siblings = CapstoneSiblings(is, issues)
		ids = CapstoneIDs(is, issues)
	} else if len(ids) == 0 {
		return s.NotDemonstrated("story names no design ids"), nil
	}
	p := d.P
	ev, err := evidence.Evaluate(p, d, false)
	if err != nil {
		return nil, err
	}
	s.Count(len(ids), "ids").Count(len(t.Proofs), "proofs").Count(len(t.Predictions), "predictions").Count(len(t.Verdicts), "verdicts")
	// the evidence is valid for a commit of a proof, a prediction or a verdict
	// when section 10.1 says so; a git failure stops the gate
	validAt := map[string]bool{}
	commits := []string{}
	for _, pr := range t.Proofs {
		commits = append(commits, pr.Commit)
	}
	for _, st := range append(append([]backlog.Stamp{}, t.Predictions...), t.Verdicts...) {
		commits = append(commits, st.Commit)
	}
	for _, c := range commits {
		if _, seen := validAt[c]; seen || c == "" {
			continue
		}
		ok, err := evidence.ValidFor(p, ev.Head, c)
		if err != nil {
			return nil, err
		}
		validAt[c] = ok
	}
	valid := func(commit string) bool { return validAt[commit] }
	short := gitx.Short(ev.Head)
	if ev.Dirty {
		s.Errorf("working tree has uncommitted changes")
	}
	// a receipts file that is not used may hold the failure of this story (section 10.2)
	s.Findings = append(s.Findings, ev.RefusedFindings()...)
	tier, _ := StoryTier(d, is, ids)
	if l := LabelTier(is); l != "" && l != tier {
		s.Warnf("label tier:%s disagrees with the ids: the story is %s", l, tier)
	}
	owed := d.OwedSet()
	for _, id := range ids {
		judgeID(s, d, ev, owed, id)
	}
	if t.Capstone {
		judgeCapstone(s, t, siblings, ids, ev.HasEvidence, short)
	}

	// routine: a proof for a commit the evidence is valid for
	builders := map[string]bool{}
	var proofs []backlog.Proof
	for _, pr := range t.Proofs {
		if valid(pr.Commit) {
			builders[pr.Actor] = true
			proofs = append(proofs, pr)
		}
	}
	switch {
	case len(builders) > 0:
	case len(t.Proofs) > 0:
		last := t.Proofs[len(t.Proofs)-1]
		s.Errorf("proof is for another commit: %s; post a proof at %s with machinery proof --story %s --post", gitx.Short(last.Commit), short, is.ID)
	default:
		s.Errorf("no proof for %s: the builder runs machinery proof --story %s --post", short, is.ID)
	}
	if tier == design.Routine {
		return s, nil
	}

	// consequential: a prediction by a fresh actor, then that actor's accept
	var preds []backlog.Stamp
	for _, pr := range t.Predictions {
		if valid(pr.Commit) {
			preds = append(preds, pr)
		}
	}
	var fresh []backlog.Stamp
	for _, pr := range preds {
		if builders[pr.Actor] {
			s.Errorf("verifier is the builder: %s", pr.Actor)
			continue
		}
		fresh = append(fresh, pr)
	}
	if len(preds) == 0 {
		s.Errorf("no prediction for %s: a verifier runs machinery predict before reading the proof", short)
	}
	judgeSessions(s, proofs, fresh)
	if len(fresh) > 0 {
		judgeVerdict(s, t, fresh, valid, short, is.ID)
	}
	if !design.Modeled(tier) {
		return s, nil
	}

	// critical and vital: the formal models of the story's areas and their replays
	areas := CriticalAreas(d, ids)
	proven := formal.ProvenAreas(d)
	specs, err := formal.Load(p)
	if err != nil {
		return nil, err
	}
	want := setOf(areas)
	for _, a := range areas {
		if !proven["fv:"+a] {
			s.Errorf("unproven: fv:%s", a)
		}
	}
	for _, sp := range specs {
		covers := false
		for _, a := range sp.Areas {
			covers = covers || want[a]
		}
		if !covers {
			continue
		}
		for _, tr := range d.Traces {
			if tr.Spec != sp.Spec {
				continue
			}
			id := "cx:" + sp.Spec + ":" + tr.Run
			if ev.State(id) != evidence.Proven {
				s.Errorf("unproven: %s", id)
			}
		}
	}
	if tier == design.Vital {
		judgeVital(s, d, ev, ids)
	}
	return s, nil
}

// DesignStoryGate is stage story for a design story (section 15.6): the
// design sound and generated at the current commit, a clean tree, and with
// reviewOn an approved design review of its intake for the current contract.
// designStages are the lint stages and stage gen-drift, run by the caller;
// intake is the story's intake item, nil when it names none.
func DesignStoryGate(d *design.Design, is *backlog.Issue, intake *Intake, designStages []*report.Stage, reviewOn bool) *report.Stage {
	s := report.New("story")
	for _, st := range designStages {
		if !st.OK() {
			s.Errorf("design: %s: %s", st.Name, OutcomeLine(st))
		}
	}
	switch dirty, err := gitx.Dirty(d.P.Root, d.P.Cfg.Receipts.Dir+"/"); {
	case err != nil:
		s.Errorf("%v", err)
	case dirty:
		s.Errorf("working tree has uncommitted changes")
	}
	reviews := 0
	if reviewOn {
		c := ContractOf(d)
		switch {
		case intake == nil:
			s.Errorf("%s names no intake: a design story is a story of one (machinery:intake <id>)", is.ID)
		default:
			var ok bool
			ok, reviews = DesignReviewApproved(intake, c.Hash)
			if !ok {
				s.Errorf("no approved design review of %s at %s: the checker runs machinery review design %s", intake.ID, c.Short(), intake.ID)
			}
		}
	}
	return s.Count(len(designStages), "stages").Count(reviews, "design reviews")
}

// judgeVital adds what the vital tier asks on top of the critical gate
// (section 15.6): the evidence of every assumption of the story's vital areas
// is not owed. Its proof obligations never block and are never proven.
func judgeVital(s *report.Stage, d *design.Design, ev *evidence.Evaluation, ids []string) {
	areas := d.VitalAreasOf(ids)
	for _, u := range d.Unevidenced(areas) {
		for _, id := range u.Owed {
			s.Errorf("%s: %s %s: %s is owed; a story of a vital area needs the evidence of every assumption proven", design.AssumedUnproven, u.Area, u.Assumption.Name, id)
		}
	}
	evidence.VitalOf(ev, areas, setOf(ids)).Warn(s, false)
}

// judgeSessions compares the sessions and agents of the proofs the evidence is
// valid for with those of the fresh predictions (section 15.6). Different
// sessions are no finding. In one session, equal agents are an error, different
// agents are no finding, and a missing agent or session is a warning.
func judgeSessions(s *report.Stage, proofs []backlog.Proof, fresh []backlog.Stamp) {
	proofWithout := false
	for _, pr := range proofs {
		if pr.Session == "" {
			proofWithout = true
		}
	}
	predWithout := false
	shared := map[string]bool{}  // "<session> <agent>"
	noAgent := map[string]bool{} // "<session> proof|prediction"
	for _, pd := range fresh {
		ss, pa := pd.Fields["session"], pd.Fields["agent"]
		if ss == "" {
			predWithout = true
			continue
		}
		for _, pr := range proofs {
			if pr.Session != ss {
				continue
			}
			if pr.Agent == "" {
				noAgent[ss+" proof"] = true
			}
			if pa == "" {
				noAgent[ss+" prediction"] = true
			}
			if pr.Agent != "" && pr.Agent == pa {
				shared[ss+" "+pa] = true
			}
		}
	}
	for _, k := range sortedKeys(shared) {
		s.Errorf("verifier shares a session and an agent with the builder: %s", k)
	}
	for _, k := range sortedKeys(noAgent) {
		ss, kind, _ := strings.Cut(k, " ")
		s.Warnf("verifier and builder share session %s and no agent is recorded for %s: independence rests on the actor name", ss, kind)
	}
	if proofWithout {
		s.Warnf("no session recorded for proof: independence rests on the actor name")
	}
	if predWithout {
		s.Warnf("no session recorded for prediction: independence rests on the actor name")
	}
}

// judgeCapstone adds what a capstone needs on top of its tier (section
// 15.6): demo lines, closed siblings, and evidence when it has no id.
func judgeCapstone(s *report.Stage, t backlog.Tags, siblings []backlog.Issue, ids []string, hasEvidence bool, short string) {
	s.Count(len(t.Demos), "demos").Count(len(siblings), "stories")
	if len(t.Demos) == 0 {
		s.Errorf("no demo: record each command that shows the intent on its own machinery:demo line")
	}
	for _, sib := range siblings {
		if sib.Status != "closed" {
			s.Errorf("open story of the intake: %s %s: the capstone closes last", sib.ID, sib.Title)
		}
	}
	if len(ids) == 0 && !hasEvidence {
		s.Errorf("no evidence at %s: run the tests through machinery run", short)
	}
}

func judgeID(s *report.Stage, d *design.Design, ev *evidence.Evaluation, owed map[string]design.OwedEntry, id string) {
	_, isOwed := owed[id]
	switch {
	case d.Status(id) == design.Proposed:
		s.Errorf("proposed: %s yields no obligation; commit it first", id)
		return
	case d.Status(id) == design.Retired:
		s.Errorf("retired: %s yields no obligation; take it out of the story", id)
		return
	case d.IsProof(id):
		s.Errorf("proof: %s awaits a machine-checked proof; a story does not discharge it: take it out of the story", id)
		return
	case !d.Known(id):
		s.Errorf("unknown id: %s", id)
		return
	case isOwed:
		s.Errorf("owed: %s", id)
	}
	if d.IsAct(id) {
		return
	}
	switch ev.State(id) {
	case evidence.Proven:
	case evidence.Failed:
		s.Errorf("failed: %s", id)
	case evidence.PartlyRun: // section 10.3: a claimant has no pass
		pr, _ := ev.Partly(id)
		s.Errorf("partly run: %s: %s: %s", id, pr.Text(), evidence.PartlyRemedy)
	default:
		if n, none := ev.NoClaimant(id); none {
			s.Errorf("unproven: %s; no claimant can run here: %d claimant(s) excused: %s", id, n, evidence.NoClaimantRemedy)
			return
		}
		if n, held := ev.AdapterNotCounted(id); held { // section 10.3
			s.Errorf("unproven: %s; adapter evidence does not count: %s", id, n.Why)
			return
		}
		s.Errorf("unproven: %s", id)
	}
}

// judgeVerdict requires an accept by a predicting actor, recorded after that
// actor's prediction; the actor's last verdict counts.
func judgeVerdict(s *report.Stage, t backlog.Tags, fresh []backlog.Stamp, valid func(string) bool, short, story string) {
	firstPred := map[string]int{}
	for _, pr := range fresh {
		if seq, ok := firstPred[pr.Actor]; !ok || pr.Seq < seq {
			firstPred[pr.Actor] = pr.Seq
		}
	}
	last := map[string]backlog.Stamp{}
	early := map[string]bool{}
	for _, v := range t.Verdicts {
		seq, predicted := firstPred[v.Actor]
		if !predicted || !valid(v.Commit) {
			continue
		}
		if v.Seq < seq {
			early[v.Actor] = true
			continue
		}
		if cur, ok := last[v.Actor]; !ok || v.Seq > cur.Seq {
			last[v.Actor] = v
		}
	}
	for _, v := range last {
		if v.Verdict == "accept" {
			return
		}
	}
	var actors []string
	for a := range last {
		actors = append(actors, a)
	}
	sort.Strings(actors)
	for _, a := range actors {
		v := last[a]
		s.Errorf("verdict %s by %s at %s: %s", v.Verdict, a, short, v.Fields["observed"])
	}
	if len(last) > 0 {
		return
	}
	var earlyActors []string
	for a := range early {
		earlyActors = append(earlyActors, a)
	}
	sort.Strings(earlyActors)
	for _, a := range earlyActors {
		s.Errorf("verdict before prediction: %s; the verifier predicts first, then runs machinery accept", a)
	}
	if len(early) == 0 {
		s.Errorf("no verdict for %s: the verifier runs machinery accept %s", short, story)
	}
}

// StoryVerify renders stage verify for the ids of one story (section 12):
// findings about other obligations are dropped, except failed ones and those
// about receipts files and receipts that are not used (section 10.2).
func StoryVerify(d *design.Design, ev *evidence.Evaluation, ids []string, partly bool) *report.Stage {
	s := report.New("verify")
	sel := setOf(ids)
	for _, f := range ev.Findings {
		if keepFinding(f.Msg, sel) {
			s.Findings = append(s.Findings, f)
		}
	}
	s.Findings = append(s.Findings, ev.PartlyFindings(sel, partly)...)
	s.Findings = append(s.Findings, ev.ExcusedFindings(partly)...)
	for _, id := range ids {
		if !d.Known(id) {
			s.Errorf("unknown id in story: %s", id)
		}
	}
	if !ev.HasEvidence {
		return s.NotDemonstrated(ev.NoEvidence())
	}
	owed := d.OwedSet()
	var n, proven, partial, partlyRun, nOwed, receipts int
	byFunction, byFile := ev.AdapterCounts(ObligationIDs(d, ids))
	for _, id := range ObligationIDs(d, ids) {
		n++
		switch ev.State(id) {
		case evidence.Proven:
			proven++
		case evidence.Partial:
			partial++
		case evidence.PartlyRun:
			partlyRun++
		}
		if _, ok := owed[id]; ok {
			nOwed++
		}
		receipts += len(ev.Receipts(id))
	}
	s.Count(n, "obligations").Count(proven, "proven")
	evidence.CountAdapter(s, byFunction, byFile)
	s.Count(partial, "partial")
	if partlyRun > 0 {
		s.Count(partlyRun, "partly run")
	}
	s.Count(nOwed, "owed").Count(receipts, "receipts")
	if areas := d.VitalAreasOf(ids); len(areas) > 0 {
		evidence.VitalOf(ev, areas, sel).Warn(s, true) // section 12
	}
	if n == 0 {
		s.NotDemonstrated("story names no obligation")
	}
	return s
}

// per-id findings of stage verify; failed ones always print
var idFindings = []string{"unproven: ", "stale owed entry: ", "unexpected effect: ", "adapter: ", "partly run: ", "no claimant can run here: ", "excused: ", "by its set: "}

func keepFinding(msg string, sel map[string]bool) bool {
	if id, ok := evidence.FindingID(msg, idFindings); ok {
		return sel[id]
	}
	return true
}
