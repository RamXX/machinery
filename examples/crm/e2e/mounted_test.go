package e2e

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

// operation is one method and path of the contract.
type operation struct{ method, path string }

var (
	pathLine   = regexp.MustCompile(`^  (/[^:]*):\s*$`)
	methodLine = regexp.MustCompile(`^    (get|post|put|patch|delete):\s*$`)
)

// contractOperations reads the operations of contracts/openapi.yaml by
// indentation: paths at two spaces under paths:, methods at four.
func contractOperations(t *testing.T) []operation {
	t.Helper()
	f, err := os.Open("../contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var ops []operation
	in, path := false, ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "paths:":
			in = true
		case in && line != "" && !strings.HasPrefix(line, " "):
			in = false
		case in && pathLine.MatchString(line):
			path = pathLine.FindStringSubmatch(line)[1]
		case in && methodLine.MatchString(line):
			ops = append(ops, operation{strings.ToUpper(methodLine.FindStringSubmatch(line)[1]), path})
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(ops) == 0 {
		t.Fatal("no operations read from contracts/openapi.yaml")
	}
	return ops
}

// TestEveryOperationMounted sends one request per contract operation: the
// route must exist (a 404 is allowed only as the NOT_FOUND answer for an
// unknown record) and accept the method.
func TestEveryOperationMounted(t *testing.T) {
	user := createUser(t, "Rep")
	for _, op := range contractOperations(t) {
		path := strings.ReplaceAll(op.path, "{id}", "6b1f0c38-3a5d-4c39-9a4e-0000000000ff")
		var in any
		if op.method == "POST" {
			in = map[string]string{}
		}
		code, b := call(t, op.method, path, user, in)
		var e apiError
		_ = json.Unmarshal(b, &e)
		if code == http.StatusMethodNotAllowed || (code == http.StatusNotFound && e.Code != "NOT_FOUND") {
			t.Errorf("%s %s = %d %s: not mounted", op.method, op.path, code, b)
		}
	}
}
