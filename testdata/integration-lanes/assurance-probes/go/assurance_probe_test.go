package probe

import "testing"

func TestAssuranceRuntimeProbe(t *testing.T) {
	if 6*7 != 42 {
		t.Fatal("pinned go probe closure mismatch")
	}
}
