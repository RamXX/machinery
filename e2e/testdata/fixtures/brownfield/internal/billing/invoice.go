// Package billing holds the invoice arithmetic of the depot.
package billing

import "errors"

// TaxBasisPoints is the sales tax: 8.25 percent.
const TaxBasisPoints = 825

// ErrAmount is returned for an amount that cannot be invoiced.
var ErrAmount = errors.New("INVOICE_AMOUNT_INVALID: the amount must be a positive number of cents")

// Invoice is the priced amount of one shipment.
type Invoice struct {
	ShipmentID string
	Subtotal   int64
	Tax        int64
	Total      int64
}

// Tax returns the tax on an amount, rounded half up to a cent.
func Tax(cents int64) int64 {
	return (cents*TaxBasisPoints + 5000) / 10000
}

// Price builds the invoice of a shipment.
func Price(shipmentID string, cents int64) (Invoice, error) {
	if cents <= 0 {
		return Invoice{}, ErrAmount
	}
	t := Tax(cents)
	return Invoice{ShipmentID: shipmentID, Subtotal: cents, Tax: t, Total: cents + t}, nil
}
