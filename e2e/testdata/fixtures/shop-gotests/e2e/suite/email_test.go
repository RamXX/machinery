package suite

import (
	"os"
	"testing"

	"example.test/shop/internal/domain"
)

// TestEmailUnique is steered by SHOP_EMAIL: "fail" fails it and "skip" skips it.
//
// machinery:verifies inv:customer-email-unique
func TestEmailUnique(t *testing.T) {
	switch os.Getenv("SHOP_EMAIL") {
	case "fail":
		t.Fatal("second customer with the same email accepted")
	case "skip":
		t.Skip("no customer store in this run")
	}
	if got := domain.Apply("register"); got != "register" {
		t.Fatalf("Apply(register) = %q", got)
	}
}
