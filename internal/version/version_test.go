package version

import (
	"strings"
	"testing"
)

func TestStampLinesCarryMarkerAndVersion(t *testing.T) {
	for name, line := range map[string]string{
		"markdown": MarkdownStamp(),
		"tla":      TLAStamp(),
		"alloy":    AlloyStamp(),
	} {
		if !strings.Contains(line, "machinery-version: "+Version) {
			t.Errorf("%s stamp %q does not carry machinery-version: %s", name, line, Version)
		}
		if strings.Contains(line, "\n") {
			t.Errorf("%s stamp %q is not a single line", name, line)
		}
	}
}

func TestStampTLAModuleInsertsAfterModuleLine(t *testing.T) {
	doc := "---- MODULE X ----\nEXTENDS Naturals\n====\n"
	got := StampTLAModule(doc)
	want := "---- MODULE X ----\n" + TLAStamp() + "\nEXTENDS Naturals\n====\n"
	if got != want {
		t.Errorf("StampTLAModule:\n%q\nwant\n%q", got, want)
	}
	// a body without a module line gets the stamp prepended, never lost
	if got := StampTLAModule("EXTENDS Naturals\n"); !strings.HasPrefix(got, TLAStamp()+"\n") {
		t.Errorf("non-module body not prepended: %q", got)
	}
}

func TestStampCfgPrepends(t *testing.T) {
	got := StampCfg("SPECIFICATION Spec\n")
	if got != TLAStamp()+"\nSPECIFICATION Spec\n" {
		t.Errorf("StampCfg = %q", got)
	}
}

func TestStripRemovesStampLinesOnly(t *testing.T) {
	stamped := "---- MODULE X ----\n" + TLAStamp() + "\nEXTENDS Naturals\n====\n"
	if got := Strip(stamped); got != "---- MODULE X ----\nEXTENDS Naturals\n====\n" {
		t.Errorf("Strip = %q", got)
	}
	// stamps of OTHER versions strip too: that is the whole point
	old := "<!-- machinery-version: v0.0.1 -->\nbody\n"
	if got := Strip(old); got != "body\n" {
		t.Errorf("Strip(old version) = %q", got)
	}
	// text without a stamp is returned unchanged
	plain := "no stamp here\n"
	if got := Strip(plain); got != plain {
		t.Errorf("Strip(plain) = %q", got)
	}
}

func TestStampOf(t *testing.T) {
	cases := []struct{ doc, want string }{
		{"<!-- machinery-version: v0.1.2 -->\nbody\n", "v0.1.2"},
		{`\* machinery-version: v9.9.9` + "\nCONSTANT X = 1\n", "v9.9.9"},
		{"// machinery-version: v0.3.3-dev\nsig S {}\n", "v0.3.3-dev"},
		{"no stamp\n", ""},
	}
	for _, c := range cases {
		if got := StampOf(c.doc); got != c.want {
			t.Errorf("StampOf(%q) = %q, want %q", c.doc, got, c.want)
		}
	}
}

func TestVersionOnlySkewStripsEqual(t *testing.T) {
	a := "---- MODULE X ----\n" + `\* machinery-version: v0.1.0` + "\nbody\n====\n"
	b := "---- MODULE X ----\n" + `\* machinery-version: v0.2.0` + "\nbody\n====\n"
	if Strip(a) != Strip(b) {
		t.Error("version-only skew must strip equal")
	}
	c := "---- MODULE X ----\n" + `\* machinery-version: v0.2.0` + "\nDRIFTED body\n====\n"
	if Strip(a) == Strip(c) {
		t.Error("content drift must survive stripping")
	}
}
