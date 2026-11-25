package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/RamXX/machinery/examples/crm/internal/store"
)

// Roles of a user.
const (
	roleAdmin = "Admin"
	roleRep   = "Rep"
)

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !strings.Contains(in.Email, "@") || (in.Role != roleAdmin && in.Role != roleRep) {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "a user needs an email and the role Admin or Rep")
		return
	}
	u, err := s.store.CreateUser(r.Context(), in.Email, in.Role)
	if errors.Is(err, store.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "EMAIL_TAKEN", "a user with this email exists")
		return
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}
