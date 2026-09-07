// Package machinerycheck is Machinery's byte-pinned assertion helper of the
// closed machinery-check/v1 transport for the go-testing/v1 adapter
// (MAC-wi2u). These bytes are frozen: internal/tdd/adapters/go.go embeds
// them, pins their sha256, materializes them into every prepared suite and
// verifies them before and after native execution. Do not edit without a
// new reviewed revision; the adapter rejects any mismatched copy.
package machinerycheck

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Check evaluates one registered assertion at its exact frozen call site.
//
// It writes exactly one machine witness line to stdout,
//
//	machinery-check/v1 witness id=<ID> value=<true|false> test=<NAME> site=<FILE>:<LINE>
//
// which the native runner captures in the enclosing test's go test -json
// output events, and on a false condition reports the failure through the
// native framework with testing.T.Errorf while naming the caller's frozen
// source site. It never recovers, catches or converts any panic or error
// raised by surrounding test code.
func Check(t *testing.T, id string, condition bool) {
	if t == nil {
		panic("machinery-check/v1: Check requires the enclosing *testing.T")
	}
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		panic("machinery-check/v1: Check cannot resolve its frozen call site")
	}
	site := filepath.Base(file)
	fmt.Fprintf(os.Stdout, "machinery-check/v1 witness id=%s value=%t test=%s site=%s:%d\n", id, condition, t.Name(), site, line)
	if !condition {
		t.Errorf("machinery-check/v1 assertion %s evaluated false at %s:%d", id, site, line)
	}
}
