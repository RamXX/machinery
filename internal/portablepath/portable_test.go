package portablepath

import "testing"

func TestRejectsCrossPlatformAliases(t *testing.T) {
	for _, value := range []string{"CON", "con.tla", "NUL.cfg", "COM1.pack", "name.", "naïve", `C:\\out`, `dir\\file`, "../out", "/out"} {
		if err := ValidateRelative(value); err == nil {
			t.Errorf("accepted non-portable path %q", value)
		}
	}
	for _, value := range []string{"Foo.tla", "checkers/pii-flow/projection.json", "orders.pack"} {
		if err := ValidateRelative(value); err != nil {
			t.Errorf("rejected portable path %q: %v", value, err)
		}
	}
}
