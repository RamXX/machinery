package suite

import (
	"os"
	"syscall"
	"testing"
	"time"

	"example.test/shop/internal/domain"
)

// TestEmailUnique is steered by SHOP_EMAIL: "fail" fails it and "skip" skips it; "hang" blocks
// it past any -timeout, "exit" ends the test binary with os.Exit, "panic" panics in a goroutine,
// and "kill" kills the go command that runs the test binary, so that its stream is cut short.
//
// machinery:verifies inv:customer-email-unique
func TestEmailUnique(t *testing.T) {
	switch os.Getenv("SHOP_EMAIL") {
	case "fail":
		t.Fatal("second customer with the same email accepted")
	case "skip":
		t.Skip("no customer store in this run")
	case "hang":
		time.Sleep(time.Hour)
	case "exit":
		os.Exit(3)
	case "panic":
		go func() { panic("customer store closed twice") }()
		time.Sleep(time.Hour)
	case "kill":
		syscall.Kill(os.Getppid(), syscall.SIGKILL)
		time.Sleep(100 * time.Millisecond)
		os.Exit(1)
	}
	if got := domain.Apply("register"); got != "register" {
		t.Fatalf("Apply(register) = %q", got)
	}
}
