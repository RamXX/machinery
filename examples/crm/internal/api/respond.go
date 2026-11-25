package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"

	"github.com/RamXX/machinery/examples/crm/internal/fsm"
	"github.com/RamXX/machinery/examples/crm/internal/store"
)

// errorBody is the Error schema of the contract. Cases is added for
// answers in which more than one case fired.
type errorBody struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Case    string   `json:"case,omitempty"`
	Cases   []string `json:"cases,omitempty"`
}

// changeBody is the Change schema of the contract.
type changeBody struct {
	ID        string   `json:"id"`
	State     string   `json:"state"`
	Cases     []string `json:"cases"`
	Effects   []string `json:"effects"`
	Simulated []string `json:"simulated"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, errorBody{Code: code, Message: msg})
}

// created answers a new record in the machine's initial state.
func created(w http.ResponseWriter, id, state string) {
	writeJSON(w, http.StatusCreated, changeBody{ID: id, State: state, Cases: []string{}, Effects: []string{}, Simulated: []string{}})
}

// refusalCode reads the API code an ignore reason promises
// ("... the API refuses with DEAL_LOST").
var refusalCode = regexp.MustCompile(`refuses with ([A-Z][A-Z0-9_]*)`)

// writeResult answers what an event did: 409 when nothing fired, 403 when
// the branch that fired keeps the state (the denied branch), 503 when the
// invoked actor failed or timed out, 200 with the Change otherwise.
func writeResult(w http.ResponseWriter, id, event string, res fsm.Result) {
	switch {
	case res.Refused:
		code := "NOT_ALLOWED_IN_STATE"
		if m := refusalCode.FindStringSubmatch(res.Reason); res.CaseID != "" && m != nil {
			code = m[1]
		}
		writeJSON(w, http.StatusConflict, errorBody{Code: code, Message: res.Reason, Case: res.CaseID})
	case res.Failed:
		writeJSON(w, http.StatusServiceUnavailable, errorBody{Code: "STORE_UNAVAILABLE",
			Message: "the store did not record the change in time; the record is " + res.State,
			Case:    res.Cases[len(res.Cases)-1], Cases: res.Cases})
	case res.Internal:
		writeJSON(w, http.StatusForbidden, errorBody{Code: "FORBIDDEN", Message: "the caller may not " + event + " this record", Case: res.CaseID})
	default:
		writeJSON(w, http.StatusOK, changeBody{ID: id, State: res.State, Cases: res.Cases, Effects: res.Effects, Simulated: res.Simulated})
	}
}

// engineError marks a failure of the machine, not of the store.
type engineError struct{ err error }

func (e engineError) Error() string { return e.err.Error() }

// writeStoreError answers a failed request: 404 for an unknown record, 500
// for a machine failure, 503 for anything the store did.
func writeStoreError(w http.ResponseWriter, err error) {
	var ee engineError
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "no such record")
	case errors.As(err, &ee):
		log.Printf("crm: machine: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", err.Error())
	default:
		writeError(w, http.StatusServiceUnavailable, "STORE_UNAVAILABLE", err.Error())
	}
}

// decode reads a JSON body; a bad body is answered with 400.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "body is not the expected JSON: "+err.Error())
		return false
	}
	return true
}

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// pathID reads the {id} of the path; a malformed id is answered with 400.
func pathID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("id")
	if !uuidRE.MatchString(id) {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "id "+id+" is not a uuid")
		return "", false
	}
	return id, true
}
