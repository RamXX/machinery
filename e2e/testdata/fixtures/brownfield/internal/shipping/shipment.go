// Package shipping holds the shipment lifecycle of the depot.
package shipping

import "fmt"

// Status is where a shipment is in its life.
type Status string

const (
	Created    Status = "created"
	Packed     Status = "packed"
	Dispatched Status = "dispatched"
	Delivered  Status = "delivered"
	Cancelled  Status = "cancelled"
)

// Shipment is one parcel moving through the depot.
type Shipment struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
}

// Error is a refused move. Code is stable and printed by the command line.
type Error struct {
	Code string
	Msg  string
}

func (e *Error) Error() string { return e.Code + ": " + e.Msg }

func refuse(code, format string, a ...any) error {
	return &Error{Code: code, Msg: fmt.Sprintf(format, a...)}
}

// New creates a shipment in the created status.
func New(id string) (*Shipment, error) {
	if id == "" {
		return nil, refuse("SHIPMENT_ID_REQUIRED", "a shipment needs an id")
	}
	return &Shipment{ID: id, Status: Created}, nil
}

// Pack moves a created shipment to packed.
func (s *Shipment) Pack() error {
	if s.Status != Created {
		return refuse("SHIPMENT_NOT_CREATED", "cannot pack shipment %s in status %s", s.ID, s.Status)
	}
	s.Status = Packed
	return nil
}

// Dispatch hands a packed shipment to the carrier.
func (s *Shipment) Dispatch() error {
	if s.Status != Packed {
		return refuse("SHIPMENT_NOT_PACKED", "cannot dispatch shipment %s in status %s", s.ID, s.Status)
	}
	s.Status = Dispatched
	return nil
}

// Deliver closes a dispatched shipment.
func (s *Shipment) Deliver() error {
	if s.Status != Dispatched {
		return refuse("SHIPMENT_NOT_DISPATCHED", "cannot deliver shipment %s in status %s", s.ID, s.Status)
	}
	s.Status = Delivered
	return nil
}

// Cancel abandons a shipment that has not left the depot.
func (s *Shipment) Cancel() error {
	if s.Status != Created && s.Status != Packed {
		return refuse("SHIPMENT_NOT_CANCELLABLE", "cannot cancel shipment %s in status %s", s.ID, s.Status)
	}
	s.Status = Cancelled
	return nil
}
