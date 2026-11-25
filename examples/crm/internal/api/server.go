// Package api is the HTTP layer of the service.
package api

import (
	"net/http"

	"github.com/RamXX/machinery/examples/crm/internal/fsm"
	"github.com/RamXX/machinery/examples/crm/internal/store"
)

// Server routes requests to the handlers.
type Server struct {
	store *store.Store
	mux   *http.ServeMux
	task  *fsm.Machine
	deal  *fsm.Machine
}

// New loads the tables from dir, binds their units and builds the handler.
// A table naming a unit the service does not implement is an error.
func New(st *store.Store, dir string) (*Server, error) {
	task, err := loadMachine(dir, "task", taskUnits())
	if err != nil {
		return nil, err
	}
	deal, err := loadMachine(dir, "deal", dealUnits())
	if err != nil {
		return nil, err
	}
	s := &Server{store: st, mux: http.NewServeMux(), task: task, deal: deal}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	m := s.mux
	m.HandleFunc("GET /healthz", s.health)
	m.HandleFunc("GET /users", s.listUsers)
	m.HandleFunc("POST /users", s.createUser)
	m.HandleFunc("GET /tasks", s.authed(s.listTasks))
	m.HandleFunc("POST /tasks", s.authed(s.createTask))
	m.HandleFunc("GET /tasks/{id}", s.authed(s.getTask))
	tasks := s.taskRecord()
	m.HandleFunc("POST /tasks/{id}/start", s.authed(s.fire(tasks, "start")))       // machinery:serves act:Task.start
	m.HandleFunc("POST /tasks/{id}/complete", s.authed(s.fire(tasks, "complete"))) // machinery:serves act:Task.complete
	m.HandleFunc("POST /tasks/{id}/cancel", s.authed(s.fire(tasks, "cancel")))     // machinery:serves act:Task.cancel
	m.HandleFunc("GET /deals", s.authed(s.listDeals))
	m.HandleFunc("POST /deals", s.authed(s.createDeal))
	m.HandleFunc("GET /deals/{id}", s.authed(s.getDeal))
	deals := s.dealRecord()
	m.HandleFunc("POST /deals/{id}/qualify", s.authed(s.fire(deals, "qualify"))) // machinery:serves act:Deal.qualify
	m.HandleFunc("POST /deals/{id}/win", s.authed(s.fire(deals, "win")))         // machinery:serves act:Deal.win
	m.HandleFunc("POST /deals/{id}/lose", s.authed(s.fire(deals, "lose")))       // machinery:serves act:Deal.lose
	m.HandleFunc("POST /deals/{id}/reopen", s.authed(s.fire(deals, "reopen")))   // machinery:serves act:Deal.reopen
	m.HandleFunc("GET /audit", s.authed(s.listAudit))
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "NO_ROUTE", r.Method+" "+r.URL.Path+" is not an operation of this service")
	})
}

// Lifecycles is what the database must let the service do to the state
// columns: the initial state and the persisted edges of each loaded table.
func (s *Server) Lifecycles() []store.Lifecycle {
	var out []store.Lifecycle
	for _, m := range []*fsm.Machine{s.deal, s.task} {
		out = append(out, store.Lifecycle{Machine: m.Name(), Initial: m.Initial(), Edges: m.PersistedEdges()})
	}
	return out
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
