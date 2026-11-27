package suite

// The markers of this file stand above no function: they belong to every test function in it.
//
// machinery:verifies sc:customer-pays-and-the-order-ships sc:cancelled-order-refuses-payment

import (
	"os"
	"testing"

	"example.test/shop/internal/api"
)

func TestPayThenShip(t *testing.T) {
	for _, event := range []string{"pay", "ship"} {
		if got := api.Handle(event); got != event {
			t.Fatalf("Handle(%s) = %q", event, got)
		}
	}
}

// TestMain is not a test function: it has no verdict of its own, and the markers of the file
// do not wait for one.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestCancelledRefusesPay(t *testing.T) {
	if os.Getenv("SHOP_SCENARIO") == "fail" {
		t.Fatal("pay accepted on a cancelled order")
	}
	if os.Getenv("SHOP_SCENARIO") == "skip" {
		t.Skip("no cancelled order in this run")
	}
	if got := api.Handle("pay"); got != "pay" {
		t.Fatalf("Handle(pay) = %q", got)
	}
}
