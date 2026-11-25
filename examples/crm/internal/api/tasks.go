package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/RamXX/machinery/examples/crm/internal/store"
)

func (s *Server) taskRecord() record {
	return record{
		machine: s.task,
		lock: func(tx *store.Tx, id string, u *unitCtx) (string, error) {
			t, err := tx.LockTask(id)
			u.task = &t
			return t.Status, err
		},
		save: func(tx *store.Tx, id, state string) error { return tx.SetTaskStatus(id, state) },
	}
}

// createTask creates a task in the machine's initial state, assigned to the
// caller.
//
// machinery:serves act:Task.create
func (s *Server) createTask(w http.ResponseWriter, r *http.Request, caller store.User) {
	var in struct {
		Title  string `json:"title"`
		DealID string `json:"dealId"`
	}
	if !decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Title) == "" || !uuidRE.MatchString(in.DealID) {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "a task needs a title and the dealId of an existing deal")
		return
	}
	t, err := s.store.CreateTask(r.Context(), store.Task{Title: in.Title, Status: s.task.Initial(), DealID: in.DealID, AssigneeID: caller.ID})
	if errors.Is(err, store.ErrBadReference) {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "dealId names no deal")
		return
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	created(w, t.ID, t.Status)
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request, _ store.User) {
	tasks, err := s.store.ListTasks(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request, _ store.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	t, err := s.store.GetTask(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}
