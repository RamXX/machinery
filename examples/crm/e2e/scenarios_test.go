package e2e

import (
	"net/http"
	"testing"
)

// The scenarios of the model, walked over HTTP as their actors, checking
// the answers, the rows and the audit trail.
//
// machinery:verifies sc:rep-qualifies-and-wins-a-deal sc:rep-cannot-change-another-rep-s-deal
// machinery:verifies sc:admin-reopens-a-lost-deal sc:rep-works-a-task-to-completion

func TestScenarioRepQualifiesAndWins(t *testing.T) {
	wants(t, "sc:rep-qualifies-and-wins-a-deal")
	rep := createUser(t, "Rep")
	id := newDeal(t, rep, 5000)
	qualify(t, id, rep)
	if c := fire(t, "/deals/"+id+"/win", rep); c.State != "Won" {
		t.Fatalf("win = %+v, want Won", c)
	}
	wantRow(t, "deals", id, "Won")
	wantTrail(t, "deal:"+id, trailRow{rep, "qualify", "applied"}, trailRow{rep, "win-attempt", "applied"}, trailRow{rep, "win", "applied"})
	pass(t, "sc:rep-qualifies-and-wins-a-deal", "created a Lead of 5000, qualified, won: the row is Won and the trail has three applied rows by the Rep")
}

func TestScenarioRepCannotChangeAnotherRepsDeal(t *testing.T) {
	wants(t, "sc:rep-cannot-change-another-rep-s-deal")
	first, second := createUser(t, "Rep"), createUser(t, "Rep")
	id := newDeal(t, first, 100)
	e := refused(t, "/deals/"+id+"/qualify", second, http.StatusForbidden)
	wantRefusal(t, e, "FORBIDDEN", "tc:DEAL-fdd582")
	wantRow(t, "deals", id, "Lead")
	wantTrail(t, "deal:"+id, trailRow{second, "dealdenied", "refused"})
	pass(t, "sc:rep-cannot-change-another-rep-s-deal", "a second Rep's qualify: 403 FORBIDDEN, the deal stays a Lead, one refused audit row")
}

func TestScenarioAdminReopensALostDeal(t *testing.T) {
	wants(t, "sc:admin-reopens-a-lost-deal")
	admin, rep := createUser(t, "Admin"), createUser(t, "Rep")
	id := newDeal(t, rep, 100)
	lose(t, id, rep)
	e := refused(t, "/deals/"+id+"/reopen", rep, http.StatusForbidden)
	wantRefusal(t, e, "FORBIDDEN", "tc:DEAL-7bb594")
	wantRow(t, "deals", id, "Lost")
	if c := fire(t, "/deals/"+id+"/reopen", admin); c.State != "Lead" {
		t.Fatalf("reopen by the Admin = %+v, want Lead", c)
	}
	wantRow(t, "deals", id, "Lead")
	wantTrail(t, "deal:"+id, trailRow{rep, "lose", "applied"}, trailRow{rep, "dealdenied", "refused"}, trailRow{admin, "reopen", "applied"})
	pass(t, "sc:admin-reopens-a-lost-deal", "lost by the Rep, reopen refused for the Rep, reopened by the Admin: a Lead again, three audit rows")
}

func TestScenarioRepWorksATaskToCompletion(t *testing.T) {
	wants(t, "sc:rep-works-a-task-to-completion")
	rep := createUser(t, "Rep")
	id := newTask(t, rep)
	fire(t, "/tasks/"+id+"/start", rep)
	fire(t, "/tasks/"+id+"/complete", rep)
	e := refused(t, "/tasks/"+id+"/cancel", rep, http.StatusConflict)
	wantRefusal(t, e, "NOT_ALLOWED_IN_STATE", "")
	wantRow(t, "tasks", id, "Done")
	wantTrail(t, "task:"+id, trailRow{rep, "start", "applied"}, trailRow{rep, "complete", "applied"})
	pass(t, "sc:rep-works-a-task-to-completion", "created Open, started, completed; cancel on Done: 409, the row stays Done, two applied audit rows")
}
