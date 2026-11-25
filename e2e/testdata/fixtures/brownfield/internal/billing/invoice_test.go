package billing

import "testing"

func TestTaxRoundsHalfUp(t *testing.T) {
	for _, c := range []struct{ cents, want int64 }{
		{1000, 83}, // 82.5 rounds up
		{100, 8},   // 8.25 rounds down
		{1, 0},
		{10000, 825},
	} {
		if got := Tax(c.cents); got != c.want {
			t.Errorf("Tax(%d) = %d, want %d", c.cents, got, c.want)
		}
	}
}

func TestPriceRefusesNonPositive(t *testing.T) {
	if _, err := Price("s1", 0); err != ErrAmount {
		t.Fatalf("got %v, want ErrAmount", err)
	}
}

func TestPriceTotals(t *testing.T) {
	inv, err := Price("s1", 2000)
	if err != nil || inv.Total != 2165 {
		t.Fatalf("got %+v, %v", inv, err)
	}
}
