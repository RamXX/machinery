package e2e

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/RamXX/machinery/v2/pkg/receipt"
)

// change is the Change schema of the contract.
type change struct {
	ID        string   `json:"id"`
	State     string   `json:"state"`
	Cases     []string `json:"cases"`
	Effects   []string `json:"effects"`
	Simulated []string `json:"simulated"`
}

// apiError is the Error schema of the contract.
type apiError struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Case    string   `json:"case"`
	Cases   []string `json:"cases"`
}

// call sends one request to the service under test. user is the X-User-Id
// header, omitted when empty; in is marshalled as the JSON body when not nil.
func call(t *testing.T, method, path, user string, in any) (int, []byte) {
	t.Helper()
	var rd io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL+path, rd)
	if err != nil {
		t.Fatal(err)
	}
	if user != "" {
		req.Header.Set("X-User-Id", user)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("%s %s: read body: %v", method, path, err)
	}
	return resp.StatusCode, out
}

// callInto sends a request and decodes the JSON answer into out.
func callInto(t *testing.T, method, path, user string, in, out any) int {
	t.Helper()
	code, b := call(t, method, path, user, in)
	if err := json.Unmarshal(b, out); err != nil {
		t.Fatalf("%s %s: %d answer is not the expected JSON: %v: %s", method, path, code, err, b)
	}
	return code
}

// fire posts an event and expects a 200 Change.
func fire(t *testing.T, path, user string) change {
	t.Helper()
	var c change
	if code := callInto(t, "POST", path, user, nil, &c); code != http.StatusOK {
		t.Fatalf("POST %s = %d %+v, want 200", path, code, c)
	}
	return c
}

// refused posts an event and expects a refusal with the given status.
func refused(t *testing.T, path, user string, status int) apiError {
	t.Helper()
	var e apiError
	if code := callInto(t, "POST", path, user, nil, &e); code != status {
		t.Fatalf("POST %s = %d %+v, want %d", path, code, e, status)
	}
	return e
}

func unique(prefix string) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + "-" + hex.EncodeToString(b)
}

// createUser creates a user over HTTP and returns its id.
func createUser(t *testing.T, role string) string {
	t.Helper()
	var u struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	}
	email := unique(role) + "@example.test"
	if code := callInto(t, "POST", "/users", "", map[string]string{"email": email, "role": role}, &u); code != http.StatusCreated || u.ID == "" || u.Role != role {
		t.Fatalf("POST /users %s = %d %+v, want 201 with an id", role, code, u)
	}
	return u.ID
}

// insertDeal writes a deal row with SQL through the admin pool.
func insertDeal(t *testing.T, owner, stage string, amount int64) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var id string
	err := adminPool.QueryRow(ctx, `INSERT INTO deals (title, stage, amount_cents, owner_id)
		VALUES ($1, $2, $3, $4::uuid) RETURNING id::text`, unique("deal"), stage, amount, owner).Scan(&id)
	if err != nil {
		t.Fatalf("insert deal: %v", err)
	}
	return id
}

// rowState reads the stored state of a task (status) or a deal (stage).
func rowState(t *testing.T, table, id string) string {
	t.Helper()
	col := map[string]string{"tasks": "status", "deals": "stage"}[table]
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var s string
	q := fmt.Sprintf("SELECT %s FROM %s WHERE id = $1::uuid", col, table)
	if err := adminPool.QueryRow(ctx, q, id).Scan(&s); err != nil {
		t.Fatalf("read %s %s: %v", table, id, err)
	}
	return s
}

func wantRow(t *testing.T, table, id, want string) {
	t.Helper()
	if got := rowState(t, table, id); got != want {
		t.Fatalf("%s row %s is %q, want %q", table, id, got, want)
	}
}

// sameList compares a decoded JSON array with want; a missing or null
// array never matches, because the contract requires the arrays.
func sameList(got, want []string) bool {
	if got == nil || len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func wantChange(t *testing.T, c change, state string, cases, effects, simulated []string) {
	t.Helper()
	if c.State != state || !sameList(c.Cases, cases) || !sameList(c.Effects, effects) || !sameList(c.Simulated, simulated) {
		t.Fatalf("change = %+v, want state %s cases %v effects %v simulated %v", c, state, cases, effects, simulated)
	}
}

func wantRefusal(t *testing.T, e apiError, code, caseID string) {
	t.Helper()
	if e.Code != code || e.Case != caseID || e.Message == "" {
		t.Fatalf("refusal = %+v, want code %s case %q and a message", e, code, caseID)
	}
}

// wants skips a test none of whose ids a targeted run asks for.
func wants(t *testing.T, ids ...string) {
	t.Helper()
	for _, id := range ids {
		if receipt.Wanted(id) {
			return
		}
	}
	t.Skipf("MACHINERY_ONLY does not ask for %v", ids)
}

// pass emits a pass receipt after the assertions held. The receipt names the
// file of the caller, which must be the test file that claims the id. The skeleton's
// effects are noops, so a test observes no effect yet and claims none.
func pass(t *testing.T, id, observed string) {
	t.Helper()
	if t.Failed() {
		return
	}
	if err := receipt.Pass(id, receipt.Test(1, t.Name()), observed); err != nil {
		t.Errorf("emit receipt %s: %v", id, err)
	}
}
