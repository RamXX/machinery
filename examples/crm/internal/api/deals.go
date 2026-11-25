package api

import (
	"net/http"
	"strings"

	"github.com/RamXX/machinery/examples/crm/internal/store"
)

func (s *Server) dealRecord() record {
	return record{
		machine: s.deal,
		lock: func(tx *store.Tx, id string, u *unitCtx) (string, error) {
			d, err := tx.LockDeal(id)
			u.deal = &d
			return d.Stage, err
		},
		save: func(tx *store.Tx, id, state, caseID string) error { return tx.SetDealStage(id, state, caseID) },
	}
}

// createDeal creates a deal in the machine's initial state, owned by the
// caller.
//
// machinery:serves act:Deal.create
func (s *Server) createDeal(w http.ResponseWriter, r *http.Request, caller store.User) {
	var in struct {
		Title       string `json:"title"`
		AmountCents int64  `json:"amountCents"`
	}
	if !decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Title) == "" || in.AmountCents < 0 {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "a deal needs a title and an amountCents of zero or more")
		return
	}
	d, err := s.store.CreateDeal(r.Context(), store.Deal{Title: in.Title, Stage: s.deal.Initial(), AmountCents: in.AmountCents, OwnerID: caller.ID})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	created(w, d.ID, d.Stage)
}

func (s *Server) listDeals(w http.ResponseWriter, r *http.Request, _ store.User) {
	deals, err := s.store.ListDeals(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, deals)
}

func (s *Server) getDeal(w http.ResponseWriter, r *http.Request, _ store.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	d, err := s.store.GetDeal(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}
