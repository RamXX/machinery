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
}

// New loads the tables from dir, binds their units and builds the handler.
// A table naming a unit the service does not implement is an error.
func New(st *store.Store, dir string) (*Server, error) {
	task, err := loadMachine(dir, "task", taskUnits())
	if err != nil {
		return nil, err
	}
	s := &Server{store: st, mux: http.NewServeMux(), task: task}
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
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "NO_ROUTE", r.Method+" "+r.URL.Path+" is not an operation of this service")
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
