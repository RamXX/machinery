package conformance

import (
	"testing"

	"machinery.test/conformance/machinerycheck"
)

// TestConformanceWitnessPass proves one registered passing assertion through
// the byte-pinned helper transport under the native runner.
func TestConformanceWitnessPass(t *testing.T) {
	answer := 6 * 7
	machinerycheck.Check(t, "conformance/witness-pass", answer == 42)
}

// TestConformanceSubtestIdentity proves fixed-name deterministic subtests:
// every declared child leaf executes under its complete native path.
func TestConformanceSubtestIdentity(t *testing.T) {
	t.Run("first", func(t *testing.T) {
		machinerycheck.Check(t, "conformance/subtest-first", 1+1 == 2)
	})
	t.Run("second", func(t *testing.T) {
		machinerycheck.Check(t, "conformance/subtest-second", 2+2 == 4)
	})
}

// TestConformanceMultipleAssertions proves two registered assertions reached
// sequentially within one native test.
func TestConformanceMultipleAssertions(t *testing.T) {
	machinerycheck.Check(t, "conformance/multi-a", len("machinery") == 9)
	machinerycheck.Check(t, "conformance/multi-b", len("assurance") == 9)
}
