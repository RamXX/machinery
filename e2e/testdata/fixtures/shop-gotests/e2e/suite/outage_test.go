//go:build faults

package suite

// machinery:verifies fm:payment-provider-down

import (
	"os"
	"testing"

	"example.test/shop/internal/domain"
)

// TestProviderDown is steered by SHOP_OUTAGE: "fail" fails it.
func TestProviderDown(t *testing.T) {
	if os.Getenv("SHOP_OUTAGE") == "fail" {
		t.Fatal("the order was charged while the provider was down")
	}
	if got := domain.Apply("providerDown"); got != "providerDown" {
		t.Fatalf("Apply(providerDown) = %q", got)
	}
}
