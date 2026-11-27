package cli

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/v2/internal/backlog"
	"github.com/RamXX/machinery/v2/internal/delegate"
	"github.com/RamXX/machinery/v2/internal/pm"
)

func init() {
	Register(&Command{Name: "answer", Summary: "Record the owner's answer to a question, or one on the owner's behalf.", Run: runAnswer})
}

func runAnswer(e *Env, args []string) int {
	fs := e.Flags("answer")
	option := fs.Int("option", 0, "the number of the option chosen")
	reason := fs.String("reason", "", "why")
	onBehalf := fs.Bool("delegate", false, "answer on the owner's behalf, under a clause of the delegate policy")
	clause := fs.String("clause", "", "with --delegate: the clause of the policy")
	hold := fs.Bool("hold", false, "with --delegate: leave the question to the owner")
	refuse := fs.Bool("refuse", false, "with --delegate: refuse what the question asks")
	actor := fs.String("actor", "", "who answers (default BEADS_ACTOR, else git user.name)")
	pos, ok := parseMixed(e, fs, args)
	if !ok {
		return 2
	}
	set := visited(fs)
	kinds := 0
	for _, k := range []string{"option", "hold", "refuse"} {
		if set[k] {
			kinds++
		}
	}
	switch {
	case len(pos) != 1:
		return e.Usage("machinery answer takes one question id")
	case !*onBehalf && (set["hold"] || set["refuse"] || set["clause"]):
		return e.Usage("--hold, --refuse and --clause go with --delegate")
	case !*onBehalf && !set["option"]:
		return e.Usage("machinery answer needs --option <n>, or --delegate")
	case *onBehalf && strings.TrimSpace(*clause) == "":
		return e.Usage("--delegate needs --clause")
	case *onBehalf && (kinds != 1 || (set["hold"] && !*hold) || (set["refuse"] && !*refuse)):
		return e.Usage("--delegate needs one of --option, --hold and --refuse")
	case pm.OneLine(*reason) == "":
		return e.Usage("machinery answer needs --reason")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	c, err := backlog.New(p, *actor)
	if err != nil {
		return e.Fail(err)
	}
	if c.Actor == "" {
		return e.Usage("no actor: pass --actor or set BEADS_ACTOR")
	}
	is, code := showStory(e, c, pos[0])
	if is == nil {
		return code
	}
	t := backlog.ParseTags(is)
	if !t.Question {
		return e.Usage("%s is not a question: it has no machinery:question line", is.ID)
	}
	if set["option"] && (*option < 1 || *option > len(t.Options)) {
		return e.Usage("--option %d names no option of %s: give 1 to %d", *option, is.ID, len(t.Options))
	}
	if a := pm.OwnerAnswer(t); a != nil {
		return e.Usage("%s is answered already by %s", is.ID, a.Actor)
	}
	pol, err := delegate.Load(p)
	if err != nil {
		return e.Fail(err)
	}
	a := answer{e: e, c: c, is: is, t: t, reason: pm.OneLine(*reason)}
	if !*onBehalf {
		return a.owner(pol, *option)
	}
	if d := pm.DelegateDecision(t); d != nil {
		return e.Usage("%s is answered already by %s", is.ID, d.Actor)
	}
	who, err := backlog.WhoFromEnv()
	if err != nil {
		return e.Usage("%v", err)
	}
	a.who = who
	kind := "option"
	switch {
	case *hold:
		kind = delegate.Hold
	case *refuse:
		kind = delegate.Refuse
	}
	return a.onBehalf(pol, strings.TrimSpace(*clause), kind, *option)
}

// answer is one answer being recorded.
type answer struct {
	e      *Env
	c      *backlog.Client
	is     *backlog.Issue
	t      backlog.Tags
	reason string
	who    backlog.Who // the session and agent of an answer on the owner's behalf
}

func (a answer) refused(format string, args ...any) int {
	fmt.Fprintf(a.e.Stderr, "machinery: refused: "+format+"\n", args...)
	return 1
}

// owner records the owner's answer and settles an answer given on the
// owner's behalf.
func (a answer) owner(pol *delegate.Policy, option int) int {
	if pol != nil && a.c.Actor != pol.Owner {
		return a.refused("%s is not the owner the delegate policy names (%s); an agent answers with --delegate", a.c.Actor, pol.Owner)
	}
	note := fmt.Sprintf("machinery:answer %d actor=%s reason=%s", option, backlog.Value(a.c.Actor), backlog.Quote(a.reason))
	settled := ""
	if d := pm.DelegateDecision(a.t); d != nil {
		if d.Option() == option {
			note += "\nmachinery:confirmed " + a.c.Actor
			settled = "confirmed: " + a.is.ID
		} else {
			note += "\nmachinery:overruled " + a.c.Actor
			settled = "overruled: " + a.is.ID
		}
	}
	if err := a.c.Note(a.is.ID, note); err != nil {
		return a.e.Fail(err)
	}
	fmt.Fprintf(a.e.Stdout, "answered: %s option %d: %s\n", a.is.ID, option, a.t.Options[option-1])
	if settled != "" {
		fmt.Fprintln(a.e.Stdout, settled)
	}
	return a.close(fmt.Sprintf("answered by the owner: option %d", option))
}

// onBehalf records an answer on the owner's behalf when the clause permits it.
func (a answer) onBehalf(pol *delegate.Policy, clause, kind string, option int) int {
	// the answer comes from the delegate, never from the context that asked
	switch asked := a.t.AskedIn; {
	case a.who.Agent == "":
		return a.refused("MACHINERY_AGENT is not set: an answer on the owner's behalf comes from the delegate, whose agent ends in -delegate")
	case !strings.HasSuffix(a.who.Agent, "-delegate"):
		return a.refused("MACHINERY_AGENT %s does not end in -delegate: an answer on the owner's behalf comes from the delegate", a.who.Agent)
	case asked.Session != "" && asked.Agent != "" && asked == a.who:
		return a.refused("%s was asked in session %s by agent %s: the asker does not answer its own question", a.is.ID, asked.Session, asked.Agent)
	}
	if pol == nil {
		return a.refused("no delegate policy: delegate.policy is not set in machinery.yaml; only the owner answers %s", a.is.ID)
	}
	cl := pol.Clause(clause)
	if cl == nil {
		return a.refused("clause %s is not in the delegate policy", clause)
	}
	head := fmt.Sprintf("machinery:answer %s actor=%s clause=%s%s reason=%s", "%s", backlog.Value(a.c.Actor), backlog.Value(clause), a.who.Part(), backlog.Quote(a.reason))
	if kind == delegate.Hold {
		if err := a.c.Note(a.is.ID, fmt.Sprintf(head, delegate.Hold)); err != nil {
			return a.e.Fail(err)
		}
		fmt.Fprintf(a.e.Stdout, "held: %s (clause %s): %s\n", a.is.ID, clause, a.reason)
		return 0
	}
	q := delegate.Question{ID: a.is.ID, Topics: a.t.About, Amount: a.t.Amount, Unit: a.t.Unit}
	var spentErr error
	spent := func() *big.Rat {
		issues, err := a.c.List()
		if err != nil {
			spentErr = err
			return new(big.Rat)
		}
		return pm.Spent(issues, clause, a.t.Unit)
	}
	why := cl.Refusal(q, kind, spent)
	if spentErr != nil {
		return a.e.Fail(spentErr)
	}
	if why != "" {
		return a.refused("%s; answer --hold", why)
	}
	what, said := kind, "refuse"
	if kind == "option" {
		what = strconv.Itoa(option)
		said = fmt.Sprintf("option %d: %s", option, a.t.Options[option-1])
	}
	note := fmt.Sprintf(head, what) + "\nmachinery:decision " + a.c.Actor + "\nmachinery:vetoable"
	if err := a.c.Note(a.is.ID, note); err != nil {
		return a.e.Fail(err)
	}
	fmt.Fprintf(a.e.Stdout, "answered: %s %s (on the owner's behalf, clause %s)\n", a.is.ID, said, clause)
	if cl.Confirm {
		fmt.Fprintf(a.e.Stdout, "open: %s waits for the owner's confirmation (clause %s)\n", a.is.ID, clause)
		return 0
	}
	return a.close("answered on the owner's behalf under clause " + clause)
}

// close closes the question when it is open.
func (a answer) close(reason string) int {
	if a.is.Status == "closed" {
		return 0
	}
	if err := a.c.Close(a.is.ID, reason); err != nil {
		return a.e.Fail(err)
	}
	fmt.Fprintf(a.e.Stdout, "closed: %s\n", a.is.ID)
	return 0
}
