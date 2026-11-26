// Package suite is the Go suite of the fixture shop. Its tests are ordinary Go tests: none of
// them emits a receipt. Each says in a comment which design ids it verifies.
package suite

import (
	"os"
	"testing"

	"example.test/shop/internal/api"
	"example.test/shop/internal/domain"
)

// TestShipNeedsPayment drives the ship event through the domain layer.
//
// machinery:verifies inv:paid-before-shipped
func TestShipNeedsPayment(t *testing.T) {
	if got := domain.Apply("ship"); got != "ship" {
		t.Fatalf("Apply(ship) = %q", got)
	}
}

// TestCancelIsFinal has two subtests; go test gives the parent their verdict.
//
// machinery:verifies inv:terminal-is-final tc:ORDE-1ded14
func TestCancelIsFinal(t *testing.T) {
	for _, state := range []string{"shipped", "cancelled"} {
		t.Run(state, func(t *testing.T) {
			if got := api.Handle("cancel"); got != "cancel" {
				t.Fatalf("Handle(cancel) = %q", got)
			}
			if state == os.Getenv("SHOP_FAIL_SUBTEST") {
				t.Fatalf("%s order changed status", state)
			}
		})
	}
}

// TestRouting claims nothing. SHOP_ROUTING=fail makes it fail, so that its package does not
// end ok while every claiming function passes.
func TestRouting(t *testing.T) {
	if os.Getenv("SHOP_ROUTING") == "fail" {
		t.Fatal("routing table not loaded")
	}
}
