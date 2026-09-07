// Package gates — RED safe-default skeleton for the authoritative assurance
// inventory. AssuranceInventory refuses with a deterministic
// not-implemented error so the frozen RED suite fails semantically.
package gates

import (
	"fmt"

	"github.com/RamXX/machinery/internal/tdd"
)

// AssuranceInventory derives the authoritative obligation inventory
// {oracle-row, guard-clause, invariant, runtime} plus the current root BUILD
// milestones from a held immutable design snapshot.
func AssuranceInventory(design string) (tdd.Inventory, error) {
	return tdd.Inventory{}, fmt.Errorf("MISSING_CONTRACT: RED STUB: AssuranceInventory not implemented")
}
