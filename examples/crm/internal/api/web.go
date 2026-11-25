package api

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// The UI: pages rendered by the service itself, no script and no build
// step. Every button posts a form to a /ui route, which sends the same
// request a client would send to the API, as the selected user, through
// the same routes (s.mux), so a click takes the same transition path as
// an API call. The answer comes back to the page as a notice: the code and
// message of a refusal, or the effects the answer lists as simulated.

//go:embed web
var webFS embed.FS

var pageTmpl = template.Must(template.New("index.html").Funcs(template.FuncMap{
	"join": strings.Join,
	"wrap": func(on string, n *notice) noticeAt { return noticeAt{on, n} },
}).ParseFS(webFS, "web/index.html"))

// Events offered as buttons, in the order of the lifecycle.
var (
	dealEvents = []string{"qualify", "win", "lose", "reopen"}
	taskEvents = []string{"start", "complete", "cancel"}
)

func (s *Server) webRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /{$}", s.page)
	m.HandleFunc("GET /style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		b, _ := webFS.ReadFile("web/style.css")
		_, _ = w.Write(b)
	})
	m.HandleFunc("POST /ui/users", s.uiCreateUser)
	m.HandleFunc("POST /ui/deals", s.uiCreateDeal)
	m.HandleFunc("POST /ui/tasks", s.uiCreateTask)
	m.HandleFunc("POST /ui/{kind}/{id}/{event}", s.uiEvent)
}

// captured is the answer of an in-process API request.
type captured struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (c *captured) Header() http.Header {
	if c.header == nil {
		c.header = http.Header{}
	}
	return c.header
}

func (c *captured) WriteHeader(status int) {
	if c.status == 0 {
		c.status = status
	}
}

func (c *captured) Write(b []byte) (int, error) {
	c.WriteHeader(http.StatusOK)
	return c.body.Write(b)
}

// api sends one request through the routes of the API as the user as
// (empty: no X-User-Id) and returns the status and the JSON answer.
func (s *Server) api(r *http.Request, method, path, as string, in any) (int, []byte) {
	var body io.Reader = http.NoBody
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return http.StatusInternalServerError, nil
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(r.Context(), method, path, body)
	if err != nil {
		return http.StatusInternalServerError, nil
	}
	if as != "" {
		req.Header.Set("X-User-Id", as)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	var c captured
	s.mux.ServeHTTP(&c, req)
	return c.status, c.body.Bytes()
}

// notice is what the page shows next to a record (or a form) after an
// answer: a refusal, or the effects that are still simulated.
type notice struct {
	Code, Message string
	Simulated     []string
}

// noticeAt is a notice with the id of the record or form it belongs to.
type noticeAt struct {
	On string
	N  *notice
}

// back sends the browser to the page as the user as, with the notice for
// the record on when there is one.
func back(w http.ResponseWriter, r *http.Request, as, on string, n notice) {
	q := url.Values{}
	if as != "" {
		q.Set("as", as)
	}
	if on != "" && (n.Code != "" || len(n.Simulated) > 0) {
		q.Set("on", on)
		q.Set("code", n.Code)
		q.Set("message", n.Message)
		q.Set("simulated", strings.Join(n.Simulated, ","))
	}
	http.Redirect(w, r, "/?"+q.Encode(), http.StatusSeeOther)
}

// answered reads an API answer: a refusal for a status of 400 or more,
// else the simulated effects of a Change.
func answered(status int, b []byte) notice {
	if status >= http.StatusBadRequest {
		var e errorBody
		if json.Unmarshal(b, &e) != nil || e.Code == "" {
			return notice{Code: http.StatusText(status), Message: "the service answered " + strconv.Itoa(status)}
		}
		return notice{Code: e.Code, Message: e.Message}
	}
	var c changeBody
	_ = json.Unmarshal(b, &c)
	return notice{Simulated: c.Simulated}
}

func (s *Server) uiCreateUser(w http.ResponseWriter, r *http.Request) {
	as := r.FormValue("as")
	status, b := s.api(r, "POST", "/users", "", map[string]string{"email": r.FormValue("email"), "role": r.FormValue("role")})
	back(w, r, as, "user-create", answered(status, b))
}

func (s *Server) uiCreateDeal(w http.ResponseWriter, r *http.Request) {
	as := r.FormValue("as")
	amount, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("amount")), 10, 64)
	if err != nil {
		back(w, r, as, "deal-create", notice{Code: "BAD_REQUEST", Message: "the amount is a whole number of cents"})
		return
	}
	status, b := s.api(r, "POST", "/deals", as, map[string]any{"title": r.FormValue("title"), "amountCents": amount})
	back(w, r, as, "deal-create", answered(status, b))
}

func (s *Server) uiCreateTask(w http.ResponseWriter, r *http.Request) {
	as, deal := r.FormValue("as"), r.FormValue("dealId")
	status, b := s.api(r, "POST", "/tasks", as, map[string]string{"title": r.FormValue("title"), "dealId": deal})
	back(w, r, as, "task-create-"+deal, answered(status, b))
}

// uiEvent fires an event on a deal or a task through the API route of the
// event.
func (s *Server) uiEvent(w http.ResponseWriter, r *http.Request) {
	as, kind, id := r.FormValue("as"), r.PathValue("kind"), r.PathValue("id")
	if kind != "deals" && kind != "tasks" {
		writeError(w, http.StatusNotFound, "NO_ROUTE", r.Method+" "+r.URL.Path+" is not a page of this service")
		return
	}
	status, b := s.api(r, "POST", "/"+kind+"/"+id+"/"+r.PathValue("event"), as, nil)
	back(w, r, as, id, answered(status, b))
}
