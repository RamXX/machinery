package api

import (
	"net/http"

	"github.com/RamXX/machinery/examples/crm/internal/fsm"
	"github.com/RamXX/machinery/examples/crm/internal/store"
)

// unitCtx is what the guards, actions and actors of a request see.
type unitCtx struct {
	tx     *store.Tx
	caller store.User
	task   *store.Task
	deal   *store.Deal
}

func unit(c fsm.Ctx) *unitCtx { return c.(*unitCtx) }

// record tells fire how to lock and save one kind of record.
type record struct {
	machine *fsm.Machine
	lock    func(tx *store.Tx, id string, u *unitCtx) (state string, err error)
	save    func(tx *store.Tx, id, state, caseID string) error
}

// fire answers an event on a record: load it FOR UPDATE, fire the event
// through the table, save the new state, commit, all in one transaction.
func (s *Server) fire(rec record, event string) userHandler {
	return func(w http.ResponseWriter, r *http.Request, caller store.User) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		var res fsm.Result
		err := s.store.InTx(r.Context(), func(tx *store.Tx) error {
			u := &unitCtx{tx: tx, caller: caller}
			state, err := rec.lock(tx, id, u)
			if err != nil {
				return err
			}
			if res, err = rec.machine.Fire(tx.Context(), state, event, u); err != nil {
				return engineError{err}
			}
			if res.State != state {
				return rec.save(tx, id, res.State, res.Cases[len(res.Cases)-1])
			}
			return nil
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeResult(w, id, event, res)
	}
}
