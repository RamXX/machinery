// End-to-end tests of the depot command line. TestMain builds the real binary; each test
// drives it over a store of its own and reads what it prints. No test here emits a receipt:
// each names in its comment the design ids it checks.
package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var bin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "depot-e2e-")
	if err != nil {
		panic(err)
	}
	bin = filepath.Join(dir, "depot")
	out, err := exec.Command("go", "build", "-o", bin, "example.test/depot/cmd/depot").CombinedOutput()
	if err != nil {
		os.RemoveAll(dir)
		panic("building depot: " + err.Error() + "\n" + string(out))
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// depot runs the binary over the store db and returns its standard output, its standard
// error and whether it exited 0.
func depot(db string, args ...string) (string, string, bool) {
	cmd := exec.Command(bin, append([]string{"--db", db}, args...)...)
	var out, errText bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errText
	err := cmd.Run()
	return strings.TrimSpace(out.String()), errText.String(), err == nil
}

// moves runs one command and wants it to print the status line.
func moves(t *testing.T, db, line string, args ...string) {
	t.Helper()
	out, errText, ok := depot(db, args...)
	if !ok || out != line {
		t.Fatalf("depot %s: got %q (stderr %q), want %q", strings.Join(args, " "), out, errText, line)
	}
}

// refused runs one command and wants it refused with the code, the shipment unchanged.
func refused(t *testing.T, db, code, id string, args ...string) {
	t.Helper()
	before, _, _ := depot(db, "ship", "show", id)
	_, errText, ok := depot(db, args...)
	if ok || !strings.HasPrefix(errText, code) {
		t.Fatalf("depot %s: want a refusal %s, got exit 0=%v, stderr %q", strings.Join(args, " "), code, ok, errText)
	}
	if after, _, _ := depot(db, "ship", "show", id); after != before {
		t.Fatalf("depot %s changed %q into %q", strings.Join(args, " "), before, after)
	}
}

// TestDelivery packs, dispatches and delivers a shipment; an unpacked one cannot leave.
//
// machinery:verifies sc:shipment-is-packed-dispatched-and-delivered inv:dispatched-only-from-packed
func TestDelivery(t *testing.T) {
	db := filepath.Join(t.TempDir(), "depot.json")
	moves(t, db, "shipment s1 created", "ship", "create", "s1")
	refused(t, db, "SHIPMENT_NOT_PACKED", "s1", "ship", "dispatch", "s1")
	moves(t, db, "shipment s1 packed", "ship", "pack", "s1")
	moves(t, db, "shipment s1 dispatched", "ship", "dispatch", "s1")
	moves(t, db, "shipment s1 delivered", "ship", "deliver", "s1")
}

// TestFinal: a dispatched shipment refuses cancellation, and a delivered or cancelled one
// never changes again.
//
// machinery:verifies sc:dispatched-shipment-refuses-cancellation inv:terminal-is-final tc:SHIP-58b48e
func TestFinal(t *testing.T) {
	db := filepath.Join(t.TempDir(), "depot.json")
	moves(t, db, "shipment s1 created", "ship", "create", "s1")
	moves(t, db, "shipment s1 packed", "ship", "pack", "s1")
	moves(t, db, "shipment s1 dispatched", "ship", "dispatch", "s1")
	refused(t, db, "SHIPMENT_NOT_CANCELLABLE", "s1", "ship", "cancel", "s1")
	moves(t, db, "shipment s1 delivered", "ship", "deliver", "s1")
	moves(t, db, "shipment s2 created", "ship", "create", "s2")
	moves(t, db, "shipment s2 cancelled", "ship", "cancel", "s2")
	for _, id := range []string{"s1", "s2"} {
		refused(t, db, "SHIPMENT_NOT_CREATED", id, "ship", "pack", id)
		refused(t, db, "SHIPMENT_NOT_PACKED", id, "ship", "dispatch", id)
		refused(t, db, "SHIPMENT_NOT_DISPATCHED", id, "ship", "deliver", id)
		refused(t, db, "SHIPMENT_NOT_CANCELLABLE", id, "ship", "cancel", id)
	}
}
