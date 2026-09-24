package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAllowLine(t *testing.T) {
	e, err := parseAllowLine(`Gx-trace  warn  /path a\/b (x|y)/   the candidate reads the new row`)
	if err != nil {
		t.Fatal(err)
	}
	if e.Gate != "Gx-trace" || e.Severity != sevWarn || e.Pattern != `path a\/b (x|y)` || e.Reason != "the candidate reads the new row" {
		t.Fatalf("entry = %#v", e)
	}
	if !e.re.MatchString("path a/b y") {
		t.Fatal("an escaped slash must match a literal slash")
	}
	for _, line := range []string{"", "   ", "# a comment"} {
		if e, err := parseAllowLine(line); e != nil || err != nil {
			t.Fatalf("%q: got %v, %v; want nothing", line, e, err)
		}
	}
}

func TestParseAllowLineRejects(t *testing.T) {
	cases := map[string]string{
		"gx warn":                      "want `<gate> <severity>",
		"gx note /x/ notes never gate": "is not error, drift or warn",
		"gx warn x reason":             "must be written /<regexp>/",
		"gx warn /unterminated reason": "no closing slash",
		"gx warn // reason":            "pattern is empty",
		"gx warn /(/ reason":           "pattern /(/",
		"gx warn /fine/":               "reason is mandatory",
		"gx warn /fine/    ":           "reason is mandatory",
	}
	for line, want := range cases {
		_, err := parseAllowLine(line)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: err = %v, want it to mention %q", line, err, want)
		}
	}
}

func TestAllowEntryMatchesGateForms(t *testing.T) {
	f := finding{Gate: "Gx-trace", Severity: sevError, Message: "row 'ship': unknown declaration group"}
	for _, gate := range []string{"Gx-trace", "gx-TRACE", "gx", "GX", "*"} {
		e, err := parseAllowLine(gate + " error /unknown declaration/ reason")
		if err != nil {
			t.Fatal(err)
		}
		if !e.matches(f) {
			t.Errorf("gate %q should match %s", gate, f.Gate)
		}
	}
	for _, line := range []string{
		"gl error /unknown declaration/ other gate",
		"gx drift /unknown declaration/ other severity",
		"gx error /^unknown/ anchored elsewhere",
	} {
		e, err := parseAllowLine(line)
		if err != nil {
			t.Fatal(err)
		}
		if e.matches(f) {
			t.Errorf("%q must not match", line)
		}
	}
}

func TestParseAllowFileNamesTheLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "allow.txt")
	if err := os.WriteFile(path, []byte("# ok\ngx warn /a/ fine\ngx warn /b/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := parseAllowFile(path)
	if err == nil || !strings.Contains(err.Error(), "allow.txt:3") {
		t.Fatalf("err = %v, want it to name line 3", err)
	}
	if _, err := parseAllowFile(filepath.Join(t.TempDir(), "missing.txt")); err == nil {
		t.Fatal("a missing allow file must be an error")
	}
}
