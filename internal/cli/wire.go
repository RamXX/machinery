package cli

import (
	"github.com/RamXX/machinery/v2/internal/evidence"
	"github.com/RamXX/machinery/v2/internal/formal"
)

// The evidence loop judges fv obligations through the formal stage. The two
// packages do not import each other; they are joined here.
func init() {
	evidence.FormalProven = formal.ProvenAreas
	evidence.FormalCounts = formal.Counts
}
