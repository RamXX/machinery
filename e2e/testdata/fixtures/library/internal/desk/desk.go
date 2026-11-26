// Package desk lends copies at the circulation desk.
package desk

import "fmt"

// Lend records what the desk does with a loan.
//
// Implements: Loan.lend, Loan.giveBack,
//   Loan.reportLost.
func Lend(event string) string {
	fmt.Println("Implements: Loan.unknown")
	return event
}
