package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
)

// machinery:verifies tc:DEAL-4a25db tc:DEAL-d09c4d tc:DEAL-701b01 tc:DEAL-81d0ab tc:DEAL-559572
// machinery:verifies tc:DEAL-062b75 tc:DEAL-2c0b2a tc:DEAL-776ffd tc:DEAL-fdd582 tc:DEAL-7bb594 tc:DEAL-fe513e
// machinery:verifies sc:rep-cannot-change-another-rep-s-deal sc:admin-reopens-a-lost-deal

// newDeal creates a deal over HTTP as user and checks it is a Lead.
func newDeal(t *testing.T, user string, amount int64) string {
	t.Helper()
	var c change
	code := callInto(t, "POST", "/deals", user, map[string]any{"title": unique("deal"), "amountCents": amount}, &c)
	if code != http.StatusCreated || c.ID == "" {
		t.Fatalf("POST /deals = %d %+v, want 201 with an id", code, c)
	}
	wantChange(t, c, "Lead", nil, nil, nil)
	wantRow(t, "deals", c.ID, "Lead")
	return c.ID
}

func qualify(t *testing.T, id, user string) {
	t.Helper()
	c := fire(t, "/deals/"+id+"/qualify", user)
	wantChange(t, c, "Qualified", []string{"tc:DEAL-4a25db"}, []string{"auditQualify"}, nil)
	wantRow(t, "deals", id, "Qualified")
}

func lose(t *testing.T, id, user string) {
	t.Helper()
	c := fire(t, "/deals/"+id+"/lose", user)
	wantChange(t, c, "Lost", []string{"tc:DEAL-776ffd"}, []string{"auditLose"}, nil)
	wantRow(t, "deals", id, "Lost")
}

func TestDealHappyPath(t *testing.T) {
	wants(t, "tc:DEAL-4a25db", "tc:DEAL-d09c4d", "tc:DEAL-701b01")
	rep := createUser(t, "Rep")
	id := newDeal(t, rep, 5000)
	qualify(t, id, rep)
	pass(t, "tc:DEAL-4a25db", "Lead qualify by the owner: 200, Qualified in the answer and the row")
	c := fire(t, "/deals/"+id+"/win", rep)
	wantChange(t, c, "Won", []string{"tc:DEAL-d09c4d", "tc:DEAL-701b01"}, []string{"stampWinAttempt", "auditWin"}, []string{"recordWin"})
	wantRow(t, "deals", id, "Won")
	pass(t, "tc:DEAL-d09c4d", "Qualified win by the owner with an amount: the walk entered closing")
	pass(t, "tc:DEAL-701b01", "closing: recordWin done, Won in the answer and the row")
}

// TestDealWinNeedsAmount makes only the amountIsPositive clause of
// canWinDeal false: the owner wins a deal of amount 0.
func TestDealWinNeedsAmount(t *testing.T) {
	wants(t, "tc:DEAL-81d0ab", "tc:DEAL-559572")
	rep := createUser(t, "Rep")
	id := newDeal(t, rep, 0)
	qualify(t, id, rep)
	e := refused(t, "/deals/"+id+"/win", rep, http.StatusForbidden)
	wantRefusal(t, e, "FORBIDDEN", "tc:DEAL-81d0ab")
	wantRow(t, "deals", id, "Qualified")
	pass(t, "tc:DEAL-81d0ab", "win by the owner with amount 0: 403 FORBIDDEN, the row stays Qualified")
	pass(t, "tc:DEAL-559572", "only amountIsPositive false: the guarded win branch did not fire")
}

func TestDealIgnoredEvents(t *testing.T) {
	wants(t, "tc:DEAL-062b75", "tc:DEAL-2c0b2a", "tc:DEAL-776ffd")
	rep := createUser(t, "Rep")
	lead := newDeal(t, rep, 100)
	e := refused(t, "/deals/"+lead+"/win", rep, http.StatusConflict)
	wantRefusal(t, e, "DEAL_NOT_QUALIFIED", "tc:DEAL-062b75")
	wantRow(t, "deals", lead, "Lead")
	pass(t, "tc:DEAL-062b75", "win on a Lead: 409 DEAL_NOT_QUALIFIED, the row stays Lead")
	lost := newDeal(t, rep, 100)
	lose(t, lost, rep)
	pass(t, "tc:DEAL-776ffd", "Lead lose by the owner: 200, Lost in the answer and the row")
	e = refused(t, "/deals/"+lost+"/qualify", rep, http.StatusConflict)
	wantRefusal(t, e, "DEAL_LOST", "tc:DEAL-2c0b2a")
	wantRow(t, "deals", lost, "Lost")
	pass(t, "tc:DEAL-2c0b2a", "qualify on a Lost deal: 409 DEAL_LOST, the row stays Lost")
}

// TestDealOwnership walks the scenario: a second Rep tries to qualify the
// first Rep's deal.
func TestDealOwnership(t *testing.T) {
	wants(t, "tc:DEAL-fdd582", "sc:rep-cannot-change-another-rep-s-deal")
	rep, other := createUser(t, "Rep"), createUser(t, "Rep")
	id := newDeal(t, rep, 100)
	e := refused(t, "/deals/"+id+"/qualify", other, http.StatusForbidden)
	wantRefusal(t, e, "FORBIDDEN", "tc:DEAL-fdd582")
	wantRow(t, "deals", id, "Lead")
	pass(t, "tc:DEAL-fdd582", "qualify by another Rep: 403 FORBIDDEN, the row stays Lead")
	pass(t, "sc:rep-cannot-change-another-rep-s-deal", "a second Rep's qualify was refused with 403 and the deal stayed a Lead")
}

// TestDealReopen walks the scenario: the Rep loses a deal, cannot reopen
// it, the Admin does.
func TestDealReopen(t *testing.T) {
	wants(t, "tc:DEAL-7bb594", "tc:DEAL-fe513e", "sc:admin-reopens-a-lost-deal")
	admin, rep := createUser(t, "Admin"), createUser(t, "Rep")
	id := newDeal(t, rep, 100)
	lose(t, id, rep)
	e := refused(t, "/deals/"+id+"/reopen", rep, http.StatusForbidden)
	wantRefusal(t, e, "FORBIDDEN", "tc:DEAL-7bb594")
	wantRow(t, "deals", id, "Lost")
	pass(t, "tc:DEAL-7bb594", "reopen by the owning Rep: 403 FORBIDDEN, the row stays Lost")
	c := fire(t, "/deals/"+id+"/reopen", admin)
	wantChange(t, c, "Lead", []string{"tc:DEAL-fe513e"}, []string{"auditReopen"}, nil)
	wantRow(t, "deals", id, "Lead")
	pass(t, "tc:DEAL-fe513e", "Lost reopen by an Admin: 200, Lead in the answer and the row")
	pass(t, "sc:admin-reopens-a-lost-deal", "lost by the Rep, reopen refused for the Rep, reopened by the Admin: a Lead again")
}

func TestDealWonIsFinal(t *testing.T) {
	rep := createUser(t, "Rep")
	id := newDeal(t, rep, 100)
	qualify(t, id, rep)
	fire(t, "/deals/"+id+"/win", rep)
	e := refused(t, "/deals/"+id+"/lose", rep, http.StatusConflict)
	wantRefusal(t, e, "NOT_ALLOWED_IN_STATE", "")
	wantRow(t, "deals", id, "Won")
}

func TestDealReads(t *testing.T) {
	rep := createUser(t, "Rep")
	id := newDeal(t, rep, 4200)
	var d struct {
		ID          string `json:"id"`
		Stage       string `json:"stage"`
		AmountCents int64  `json:"amountCents"`
		OwnerID     string `json:"ownerId"`
	}
	if code := callInto(t, "GET", "/deals/"+id, rep, nil, &d); code != http.StatusOK || d.ID != id || d.Stage != "Lead" || d.AmountCents != 4200 || d.OwnerID != rep {
		t.Fatalf("GET /deals/%s = %d %+v, want 200 the Lead of 4200 owned by %s", id, code, d, rep)
	}
	code, b := call(t, "GET", "/deals", rep, nil)
	var all []json.RawMessage
	if err := json.Unmarshal(b, &all); code != http.StatusOK || err != nil || len(all) == 0 {
		t.Fatalf("GET /deals = %d %s, want 200 with a list", code, b)
	}
}

// TestAuditList: GET /audit lists the row a qualify wrote.
func TestAuditList(t *testing.T) {
	rep := createUser(t, "Rep")
	id := newDeal(t, rep, 100)
	qualify(t, id, rep)
	var entries []struct {
		ActorID string `json:"actorId"`
		Action  string `json:"action"`
		Subject string `json:"subject"`
		Outcome string `json:"outcome"`
		At      string `json:"at"`
	}
	if code := callInto(t, "GET", "/audit", rep, nil, &entries); code != http.StatusOK {
		t.Fatalf("GET /audit = %d, want 200", code)
	}
	found := 0
	for _, e := range entries {
		if e.Subject == "deal:"+id {
			found++
			if e.ActorID != rep || e.Action != "qualify" || e.Outcome != "applied" || e.At == "" {
				t.Fatalf("GET /audit lists %+v for deal %s, want qualify applied by %s", e, id, rep)
			}
		}
	}
	if found != 1 {
		t.Fatalf("GET /audit lists %d rows for deal %s, want 1", found, id)
	}
}
