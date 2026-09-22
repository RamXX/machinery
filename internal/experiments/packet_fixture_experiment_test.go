package experiments

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/gates"
)

// This corpus experiment makes the fixture obligation observable outside the
// projector's unit suite. Removing the declaration or its packet statement
// must make the experiment fail.
func TestPacketFixtureObligationSurvivesProjection(t *testing.T) {
	design := filepath.Join("..", "..", "testdata", "packet-fixture", "design")
	packets, gate := gates.ProjectPackets(design, "M1", "")
	if len(gate.Errs) > 0 {
		t.Fatalf("packet fixture design failed: %v", gate.Errs)
	}
	if gate.Counts["fixture modules"] != 1 || gate.Counts["fixture bindings"] != 2 {
		t.Fatalf("fixture obligation was not fully counted: %v", gate.Counts)
	}
	for _, packet := range packets {
		body := string(packet.Body)
		if !strings.Contains(body, "`test/support/release_fixture.ex` feeds suites of slices M1-S1 and M1-S2") {
			t.Fatalf("packet %s lost the shared fixture obligation", packet.Slice)
		}
	}
}
