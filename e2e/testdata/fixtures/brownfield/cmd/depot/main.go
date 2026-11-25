// Command depot drives shipments and invoices from the command line.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"example.test/depot/internal/billing"
	"example.test/depot/internal/shipping"
)

const usage = "usage: depot [--db <file>] ship create|pack|dispatch|deliver|cancel|show <id>\n       depot [--db <file>] invoice <id> <cents>"

func main() {
	db := flag.String("db", "depot.json", "the JSON file that holds the shipments")
	flag.Parse()
	args := flag.Args()
	switch {
	case len(args) == 3 && args[0] == "ship":
		fail(ship(shipping.OpenStore(*db), args[1], args[2]))
	case len(args) == 3 && args[0] == "invoice":
		fail(invoice(args[1], args[2]))
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
}

func fail(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func ship(st *shipping.Store, verb, id string) error {
	if verb == "create" {
		s, err := shipping.New(id)
		if err != nil {
			return err
		}
		if err := st.Add(s); err != nil {
			return err
		}
		return show(s)
	}
	s, err := st.Get(id)
	if err != nil {
		return err
	}
	moves := map[string]func() error{"pack": s.Pack, "dispatch": s.Dispatch, "deliver": s.Deliver, "cancel": s.Cancel}
	if move, ok := moves[verb]; ok {
		if err := move(); err != nil {
			return err
		}
		if err := st.Update(s); err != nil {
			return err
		}
	} else if verb != "show" {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	return show(s)
}

func show(s *shipping.Shipment) error {
	fmt.Printf("shipment %s %s\n", s.ID, s.Status)
	return nil
}

func invoice(id, amount string) error {
	cents, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return billing.ErrAmount
	}
	inv, err := billing.Price(id, cents)
	if err != nil {
		return err
	}
	fmt.Printf("invoice %s subtotal=%d tax=%d total=%d\n", inv.ShipmentID, inv.Subtotal, inv.Tax, inv.Total)
	return nil
}
