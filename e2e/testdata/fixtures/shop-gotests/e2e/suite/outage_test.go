//go:build faults

package suite

// machinery:verifies fm:payment-provider-down

import (
	"testing"

	"example.test/shop/internal/domain"
)

func TestProviderDown(t *testing.T) {
	if got := domain.Apply("providerDown"); got != "providerDown" {
		t.Fatalf("Apply(providerDown) = %q", got)
	}
}
