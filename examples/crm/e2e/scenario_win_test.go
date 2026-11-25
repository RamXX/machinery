package e2e

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// The scenarios walked in a real headless Chrome through the pages of the
// service. They prove the same scenarios as scenarios_test.go over HTTP, a
// second receipt each.
//
// machinery:verifies sc:rep-qualifies-and-wins-a-deal sc:rep-cannot-change-another-rep-s-deal

func TestScenarioRepQualifiesAndWinsInBrowser(t *testing.T) {
	wants(t, "sc:rep-qualifies-and-wins-a-deal")
	p := openBrowser(t)
	p.open("/")
	email := unique("rep") + "@example.test"
	p.fill("user-create-email", email)
	p.fill("user-create-role", "Rep")
	p.click("user-create-submit")
	rep := idBy(t, "users", "email", email)
	p.selectUser(rep, email)

	title := unique("deal")
	p.fill("deal-create-title", title)
	p.fill("deal-create-amount", "5000")
	p.click("deal-create-submit")
	id := idBy(t, "deals", "title", title)
	p.waitEqual("deal-"+id+"-stage", "Lead")
	p.click("deal-" + id + "-qualify")
	p.waitEqual("deal-"+id+"-stage", "Qualified")
	p.click("deal-" + id + "-win")
	p.waitEqual("deal-"+id+"-stage", "Won")
	p.absent("refusal-" + id)
	// Every effect of a win is real now, so the answer lists none as
	// simulated and the page shows no simulated note.
	p.absent("simulated-" + id)
	p.waitContains("audit", "deal:"+id)

	wantRow(t, "deals", id, "Won")
	wantTrail(t, "deal:"+id, trailRow{rep, "qualify", "applied"}, trailRow{rep, "win-attempt", "applied"}, trailRow{rep, "win", "applied"})
	pass(t, "sc:rep-qualifies-and-wins-a-deal", "in Chrome: created a Rep, selected it, created a deal of 5000, clicked qualify then win: the page shows Lead, Qualified, Won and no simulated note; the row is Won and the trail has three applied rows by the Rep")
}

func TestUIRefusalShown(t *testing.T) {
	wants(t, "sc:rep-cannot-change-another-rep-s-deal")
	first, second := createUser(t, "Rep"), createUser(t, "Rep")
	id := newDeal(t, first, 100)
	p := openBrowser(t)
	p.open("/")
	p.selectUser(second, emailOf(t, second))
	p.waitEqual("deal-"+id+"-stage", "Lead")
	p.click("deal-" + id + "-qualify")
	if got := p.waitContains("refusal-"+id, "FORBIDDEN"); !strings.HasPrefix(got, "FORBIDDEN") {
		t.Fatalf("refusal-%s = %q, want the code FORBIDDEN first", id, got)
	}
	p.waitEqual("deal-"+id+"-stage", "Lead")
	wantRow(t, "deals", id, "Lead")
	wantTrail(t, "deal:"+id, trailRow{second, "dealdenied", "refused"})
	pass(t, "sc:rep-cannot-change-another-rep-s-deal", "in Chrome: a second Rep clicked qualify on the first Rep's deal: the page shows FORBIDDEN next to the deal and the stage Lead; the row is a Lead, one refused audit row")
}

// TestUIServesNoExternalResources reads what the pages serve: nothing may
// load from another origin.
func TestUIServesNoExternalResources(t *testing.T) {
	rep := createUser(t, "Rep")
	newDeal(t, rep, 100)
	for _, path := range []string{"/", "/?as=" + rep, "/style.css"} {
		code, body := get(t, path)
		if code != http.StatusOK || len(body) == 0 {
			t.Fatalf("GET %s = %d, %d bytes; want 200 and a body", path, code, len(body))
		}
		for _, scheme := range []string{"http://", "https://"} {
			if i := strings.Index(body, scheme); i >= 0 {
				t.Errorf("GET %s serves an external URL: %q", path, body[i:min(len(body), i+60)])
			}
		}
	}
}

func get(t *testing.T, path string) (int, string) {
	t.Helper()
	resp, err := http.Get(baseURL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp.StatusCode, string(b)
}

// idBy waits for the row whose column has the value, which the form a
// click submitted creates, and returns its id.
func idBy(t *testing.T, table, column, value string) string {
	t.Helper()
	q := "SELECT id::text FROM " + table + " WHERE " + column + " = $1"
	var err error
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		var id string
		err = adminPool.QueryRow(ctx, q, value).Scan(&id)
		cancel()
		if err == nil {
			return id
		}
	}
	t.Fatalf("no %s row with %s %q within 10s: %v", table, column, value, err)
	return ""
}

func emailOf(t *testing.T, user string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var email string
	if err := adminPool.QueryRow(ctx, "SELECT email FROM users WHERE id = $1::uuid", user).Scan(&email); err != nil {
		t.Fatalf("read user %s: %v", user, err)
	}
	return email
}
