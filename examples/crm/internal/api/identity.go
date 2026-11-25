package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/RamXX/machinery/examples/crm/internal/store"
)

// requestTimeout bounds the store work of one request.
const requestTimeout = 2500 * time.Millisecond

// userHandler is a handler that knows its caller.
type userHandler func(w http.ResponseWriter, r *http.Request, caller store.User)

// authed resolves the X-User-Id header to an existing user; a missing,
// malformed or unknown id is 401 UNAUTHENTICATED.
func (s *Server) authed(h userHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-User-Id")
		if !uuidRE.MatchString(id) {
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "X-User-Id must name an existing user")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()
		caller, err := s.store.GetUser(ctx, id)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "X-User-Id names no user")
			return
		}
		if err != nil {
			writeStoreError(w, err)
			return
		}
		h(w, r.WithContext(ctx), caller)
	}
}
