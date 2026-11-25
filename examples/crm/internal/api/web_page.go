package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/RamXX/machinery/examples/crm/internal/store"
)

// pageData is what index.html renders.
type pageData struct {
	Users   []store.User
	As      *store.User
	Deals   []dealView
	Audit   []auditView
	Notices map[string]*notice // by record id or form name
	Error   *notice            // a read of the page that failed
	DealEvt []string
	TaskEvt []string
}

type dealView struct {
	store.Deal
	Owner string
	Tasks []taskView
}

type taskView struct {
	store.Task
	Assignee string
}

type auditView struct {
	store.AuditEntry
	Actor string
}

// page renders the UI as the user named by ?as=, with the notice the last
// answer left in the query.
func (s *Server) page(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	d := pageData{Notices: map[string]*notice{}, DealEvt: dealEvents, TaskEvt: taskEvents}
	if on := q.Get("on"); on != "" {
		n := &notice{Code: q.Get("code"), Message: q.Get("message")}
		if sim := q.Get("simulated"); sim != "" {
			n.Simulated = strings.Split(sim, ",")
		}
		d.Notices[on] = n
	}
	status := http.StatusOK
	if err := s.read(r, "/users", "", &d.Users); err != nil {
		d.Error, status = err, http.StatusServiceUnavailable
	} else if as := q.Get("as"); as != "" {
		d.As = userByID(d.Users, as)
		if d.As != nil {
			d.Error = s.readRecords(r, &d)
		}
		if d.Error != nil {
			status = http.StatusServiceUnavailable
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := pageTmpl.Execute(w, d); err != nil {
		log.Printf("crm: page: %v", err)
	}
}

// read decodes one API answer; a refusal comes back as the notice.
func (s *Server) read(r *http.Request, path, as string, out any) *notice {
	status, b := s.api(r, "GET", path, as, nil)
	if status != http.StatusOK {
		n := answered(status, b)
		return &n
	}
	if err := json.Unmarshal(b, out); err != nil {
		return &notice{Code: "INTERNAL", Message: path + ": " + err.Error()}
	}
	return nil
}

// readRecords fills the deals, their tasks and the audit trail as the
// selected user.
func (s *Server) readRecords(r *http.Request, d *pageData) *notice {
	var deals []store.Deal
	var tasks []store.Task
	var audit []store.AuditEntry
	for _, rd := range []struct {
		path string
		out  any
	}{{"/deals", &deals}, {"/tasks", &tasks}, {"/audit", &audit}} {
		if n := s.read(r, rd.path, d.As.ID, rd.out); n != nil {
			return n
		}
	}
	email := func(id string) string {
		if u := userByID(d.Users, id); u != nil {
			return u.Email
		}
		return id
	}
	byDeal := map[string][]taskView{}
	for _, t := range tasks {
		byDeal[t.DealID] = append(byDeal[t.DealID], taskView{t, email(t.AssigneeID)})
	}
	for _, dl := range deals {
		d.Deals = append(d.Deals, dealView{dl, email(dl.OwnerID), byDeal[dl.ID]})
	}
	for _, a := range audit {
		d.Audit = append(d.Audit, auditView{a, email(a.ActorID)})
	}
	return nil
}

func userByID(users []store.User, id string) *store.User {
	for i := range users {
		if users[i].ID == id {
			return &users[i]
		}
	}
	return nil
}
