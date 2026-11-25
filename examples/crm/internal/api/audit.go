package api

import (
	"net/http"

	"github.com/RamXX/machinery/examples/crm/internal/store"
)

// listAudit answers the audit trail, oldest first.
func (s *Server) listAudit(w http.ResponseWriter, r *http.Request, _ store.User) {
	entries, err := s.store.ListAudit(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if entries == nil {
		entries = []store.AuditEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}
