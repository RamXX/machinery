// Package api is the HTTP layer of the service.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/RamXX/machinery/examples/crm/internal/store"
)

// Server routes requests to the handlers.
type Server struct {
	store *store.Store
	mux   *http.ServeMux
}

// New builds the handler of the service.
func New(st *store.Store) *Server {
	s := &Server{store: st, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /healthz", s.health)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// errorBody is the body of every refusal.
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, errorBody{Code: code, Message: msg})
}
