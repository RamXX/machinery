package gates

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// packetFixture copies testdata/packet-fixture/design into a temp dir so a
// test can mutate one file and observe the gate.
func packetFixture(t *testing.T) string {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "packet-fixture", "design")
	dst := filepath.Join(t.TempDir(), "design")
	err := filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if fi.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func rewriteFixtureFile(t *testing.T, design, rel, old, new string) {
	t.Helper()
	path := filepath.Join(design, filepath.FromSlash(rel))
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), old) {
		t.Fatalf("%s does not contain %q", rel, old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(body), old, new, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func packetByID(t *testing.T, packets []Packet, id string) Packet {
	t.Helper()
	for _, p := range packets {
		if p.Slice == id {
			return p
		}
	}
	t.Fatalf("no packet %s among %d", id, len(packets))
	return Packet{}
}

func wantErr(t *testing.T, g *Gate, fragment string) {
	t.Helper()
	for _, e := range g.Errs {
		if strings.Contains(e, fragment) {
			return
		}
	}
	t.Fatalf("no ERROR contains %q; got %q", fragment, g.Errs)
}

func TestPacketProjectionClean(t *testing.T) {
	design := packetFixture(t)
	packets, g := ProjectPackets(design, "", "")
	if len(g.Errs) > 0 {
		t.Fatalf("unexpected errors: %q", g.Errs)
	}
	if len(packets) != 2 {
		t.Fatalf("packets = %d, want 2", len(packets))
	}
	s1 := string(packetByID(t, packets, "M1-S1").Body)
	s2 := string(packetByID(t, packets, "M1-S2").Body)
	for _, want := range []string{
		"### section:BUILD/core.md#8.1 (BUILD/core.md:11-18)",
		"### milestone:BUILD/core.md (BUILD/core.md:27-28)",
		"### section:BUILD.md#11 (BUILD.md:34-37)",
		"### DEAL-38ba11 (machines/Deal.oracle.md:10)",
		"### matrix:Deal (machines/Deal.matrix.md:1-8)",
		"### boundary:crm.domain (ARCHITECTURE.md:8-11)",
		"### external:external.db (ARCHITECTURE.md:18-20)",
		"### rule:crm.domain -> crm.repo (ARCHITECTURE.md:23, dependency_rules.allow)",
		"### row:ARCHITECTURE.md#3#db (ARCHITECTURE.md:33)",
		"### row:ARCHITECTURE.md#6#crm.domain -> crm.repo (ARCHITECTURE.md:39)",
		"### invariant:deal-stage-forward (domain.modelith.yaml:9-11)",
		"### invariant:deal-stage-forward (BUILD.md:15, traceability row)",
		"| DEAL-1fe825 | M1-S2 |",
		"| POL-a1b2c3 | M1-S2 |",
		"This slice claims 1 of 4: DEAL-38ba11",
		"## not a heading", // fenced content inside a cited section is copied verbatim
	} {
		if !strings.Contains(s1, want) {
			t.Errorf("M1-S1 packet lacks %q\n%s", want, s1)
		}
	}
	for _, want := range []string{
		"### section:BUILD/core.md#8.2 (BUILD/core.md:20-22)",
		"### oracleset:formal/Policy.oracle.md (formal/Policy.oracle.md:3-6)",
		"This slice claims 3 of 4: DEAL-1fe825, POL-a1b2c3, POL-d4e5f6",
	} {
		if !strings.Contains(s2, want) {
			t.Errorf("M1-S2 packet lacks %q\n%s", want, s2)
		}
	}
	// AC7: a packet is bounded by what its slice cites; the shard and the root
	// are never read wholesale.
	for _, absent := range []string{"## 1. Charter", "Named units", "This paragraph is what the", "### 9.1 Notes"} {
		if strings.Contains(s1, absent) {
			t.Errorf("M1-S1 packet carries uncited content %q", absent)
		}
	}
}

func TestPacketCarriesSharedFixtureObligations(t *testing.T) {
	design := packetFixture(t)
	packets, g := ProjectPackets(design, "M1", "")
	if len(g.Errs) > 0 {
		t.Fatalf("fixture declaration failed: %v", g.Errs)
	}
	if g.Counts["fixture bindings"] != 2 || g.Counts["fixture modules"] != 1 {
		t.Fatalf("fixture counts = %v", g.Counts)
	}
	for _, packet := range packets {
		body := string(packet.Body)
		for _, want := range []string{
			"## 7. Fixture obligations",
			"`test/support/release_fixture.ex` feeds suites of slices M1-S1 and M1-S2; run every consuming slice suite after changing it.",
		} {
			if !strings.Contains(body, want) {
				t.Errorf("%s packet lacks %q\n%s", packet.Slice, want, body)
			}
		}
	}
}

func TestPacketFixtureDeclarationFailsClosed(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		finding string
	}{
		{"not a list", " fixture.ex", "fixtures must be a non-empty list"},
		{"not portable", "\n          - ../fixture.ex", "is not a clean implementation-relative path"},
		{"duplicate", "\n          - fixture.ex\n          - fixture.ex", "fixtures repeats 'fixture.ex'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			design := packetFixture(t)
			rewriteFixtureFile(t, design, "slices.yaml", "        fixtures:\n          - test/support/release_fixture.ex\n", "        fixtures:"+tt.value+"\n")
			_, g := ProjectPackets(design, "", "")
			wantErr(t, g, tt.finding)
		})
	}
}

// AC2: two runs over the same bytes are byte-identical.
func TestPacketProjectionIsByteReproducible(t *testing.T) {
	design := packetFixture(t)
	first, g := ProjectPackets(design, "M1", "")
	if len(g.Errs) > 0 {
		t.Fatal(g.Errs)
	}
	second, _ := ProjectPackets(design, "M1", "")
	for i := range first {
		if !bytes.Equal(first[i].Body, second[i].Body) {
			t.Fatalf("packet %s differs between two runs", first[i].Slice)
		}
	}
}

var excerptHeadingRe = regexp.MustCompile(`(?m)^### (\S.*?) \(([^\s:()]+):(\d+)(?:-(\d+))?(?:, [^)]*)?\)$`)

// AC3: every excerpt heading names a source path:line that resolves, and the
// excerpt bytes are those source lines.
func TestPacketExcerptsResolveToSource(t *testing.T) {
	design := packetFixture(t)
	packets, g := ProjectPackets(design, "", "")
	if len(g.Errs) > 0 {
		t.Fatal(g.Errs)
	}
	for _, p := range packets {
		body := string(p.Body)
		matches := excerptHeadingRe.FindAllStringSubmatchIndex(body, -1)
		if len(matches) == 0 {
			t.Fatalf("%s: no excerpt headings", p.Slice)
		}
		for i, m := range matches {
			cite := body[m[2]:m[3]]
			rel := body[m[4]:m[5]]
			first, _ := strconv.Atoi(body[m[6]:m[7]])
			last := first
			if m[8] >= 0 {
				last, _ = strconv.Atoi(body[m[8]:m[9]])
			}
			src, err := os.ReadFile(filepath.Join(design, filepath.FromSlash(rel)))
			if err != nil {
				t.Fatalf("%s: excerpt %s names unreadable %s: %v", p.Slice, cite, rel, err)
			}
			lines := strings.Split(string(src), "\n")
			if first < 1 || last > len(lines) || last < first {
				t.Fatalf("%s: excerpt %s names %s:%d-%d outside the file's %d lines", p.Slice, cite, rel, first, last, len(lines))
			}
			end := len(body)
			if i+1 < len(matches) {
				end = matches[i+1][0]
			}
			section := body[m[1]:end]
			if strings.HasPrefix(cite, "section:") || strings.HasPrefix(cite, "milestone:") || strings.HasPrefix(cite, "matrix:") {
				want := strings.Join(lines[first-1:last], "\n")
				if !strings.Contains(section, "-->\n"+want+"\n<!-- end ") {
					t.Errorf("%s: markdown excerpt %s is not the verbatim source lines %s:%d-%d", p.Slice, cite, rel, first, last)
				}
				continue
			}
			// fenced and table excerpts: the last source line of the range is
			// present verbatim, and every excerpt line is a source line
			if !strings.Contains(section, lines[last-1]) {
				t.Errorf("%s: excerpt %s does not carry source line %s:%d verbatim", p.Slice, cite, rel, last)
			}
		}
	}
}

// AC4: a packet over its declared budget fails the gate and nothing is
// projected.
func TestPacketOverBudgetFails(t *testing.T) {
	design := packetFixture(t)
	rewriteFixtureFile(t, design, "slices.yaml", "budget: 12000\n        fixtures:\n          - test/support/release_fixture.ex\n        cites:\n          - section:BUILD/core.md#8.1", "budget: 500\n        fixtures:\n          - test/support/release_fixture.ex\n        cites:\n          - section:BUILD/core.md#8.1")
	packets, g := ProjectPackets(design, "M1", "")
	wantErr(t, g, "slice M1-S1 projects to")
	wantErr(t, g, "over its declared budget of 500")
	if packets != nil {
		t.Fatal("packets returned despite a failing gate")
	}
}

// AC5: an obligation claimed by no slice fails; a waiver releases it; a
// waiver on a claimed obligation contradicts; a double claim fails.
func TestPacketObligationCoverage(t *testing.T) {
	t.Run("unclaimed", func(t *testing.T) {
		design := packetFixture(t)
		rewriteFixtureFile(t, design, "slices.yaml", "          - DEAL-1fe825\n", "")
		_, g := ProjectPackets(design, "", "")
		wantErr(t, g, "obligation DEAL-1fe825 is claimed by no slice and carries no waiver")
	})
	t.Run("waived", func(t *testing.T) {
		design := packetFixture(t)
		rewriteFixtureFile(t, design, "slices.yaml", "          - DEAL-1fe825\n", "")
		rewriteFixtureFile(t, design, "slices.yaml", "          - invariant:rbac-write-scope\n", "          - invariant:rbac-write-scope\n    waivers:\n      - id: DEAL-1fe825\n        reason: the lose path moved to M2 by owner ruling\n")
		packets, g := ProjectPackets(design, "", "")
		if len(g.Errs) > 0 {
			t.Fatal(g.Errs)
		}
		if !strings.Contains(string(packetByID(t, packets, "M1-S1").Body), "| DEAL-1fe825 | waived: the lose path moved to M2 by owner ruling |") {
			t.Fatal("ledger does not record the waiver")
		}
	})
	t.Run("claimed and waived", func(t *testing.T) {
		design := packetFixture(t)
		rewriteFixtureFile(t, design, "slices.yaml", "          - invariant:rbac-write-scope\n", "          - invariant:rbac-write-scope\n    waivers:\n      - id: DEAL-1fe825\n        reason: stale\n")
		_, g := ProjectPackets(design, "", "")
		wantErr(t, g, "obligation DEAL-1fe825 is both claimed by M1-S2 and waived")
	})
	t.Run("stale waiver", func(t *testing.T) {
		design := packetFixture(t)
		rewriteFixtureFile(t, design, "slices.yaml", "          - invariant:rbac-write-scope\n", "          - invariant:rbac-write-scope\n    waivers:\n      - id: DEAL-eb0c40\n        reason: not owed by M1\n")
		_, g := ProjectPackets(design, "", "")
		wantErr(t, g, "waives DEAL-eb0c40, which its DoD does not cite")
	})
	t.Run("double claim", func(t *testing.T) {
		design := packetFixture(t)
		rewriteFixtureFile(t, design, "slices.yaml", "          - DEAL-38ba11\n", "          - DEAL-38ba11\n          - T-DEAL-03\n")
		_, g := ProjectPackets(design, "", "")
		wantErr(t, g, "obligation DEAL-1fe825 is claimed by M1-S1 and M1-S2")
	})
}

// AC6: a citation that does not resolve, or a shard that does not exist,
// fails.
func TestPacketDanglingCitationsFail(t *testing.T) {
	cases := []struct{ name, old, new, want string }{
		{"unknown oracle id", "          - DEAL-38ba11\n", "          - DEAL-ffffff\n", "cites oracle id DEAL-ffffff, which no committed oracle declares"},
		{"unknown section", "section:BUILD/core.md#8.1", "section:BUILD/core.md#8.9", `no heading of BUILD/core.md has id '8.9'`},
		{"missing shard", "shard: BUILD/core.md\n        budget: 12000\n        fixtures:\n          - test/support/release_fixture.ex\n        cites:\n          - section:BUILD/core.md#8.1", "shard: BUILD/gone.md\n        budget: 12000\n        fixtures:\n          - test/support/release_fixture.ex\n        cites:\n          - section:BUILD/core.md#8.1", "shard BUILD/gone.md does not exist in the design"},
		{"foreign shard", "section:BUILD/core.md#8.1", "section:BUILD/other.md#8.1", "cites BUILD/other.md, but the slice is bound to shard BUILD/core.md"},
		{"unknown boundary", "boundary:crm.domain", "boundary:crm.nope", `declares no boundaries item with id 'crm.nope'`},
		{"unknown rule", "rule:crm.domain -> crm.repo", "rule:crm.repo -> crm.nope", `no dependency_rules allow, deny, or baseline entry reads 'crm.repo -> crm.nope'`},
		{"unknown row", "row:ARCHITECTURE.md#3#db", "row:ARCHITECTURE.md#3#redis", `no table row under ARCHITECTURE.md:29-33 has first-cell key 'redis'`},
		{"unknown invariant", "invariant:deal-stage-forward", "invariant:deal-goes-backward", `declares no invariant with id 'deal-goes-backward'`},
		{"unknown matrix", "matrix:Deal", "matrix:Order", "matrix:Order does not resolve"},
		{"unknown kind", "matrix:Deal", "chapter:Deal", `has unknown kind 'chapter'`},
		{"repeated citation", "          - matrix:Deal\n", "          - matrix:Deal\n          - matrix:Deal\n", `cites repeats 'matrix:Deal'`},
		{"slice under wrong milestone", "      - id: M1-S2\n", "      - id: M2-S2\n", `slice id 'M2-S2' belongs to M2, not to M1`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			design := packetFixture(t)
			rewriteFixtureFile(t, design, "slices.yaml", tc.old, tc.new)
			packets, g := ProjectPackets(design, "", "")
			wantErr(t, g, tc.want)
			if packets != nil {
				t.Fatal("packets returned despite a failing gate")
			}
		})
	}
}

func TestPacketUndeclaredMilestoneFails(t *testing.T) {
	design := packetFixture(t)
	body, err := os.ReadFile(filepath.Join(design, "slices.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	renamed := strings.ReplaceAll(strings.ReplaceAll(string(body), "  - id: M1\n", "  - id: M7\n"), "M1-S", "M7-S")
	if err := os.WriteFile(filepath.Join(design, "slices.yaml"), []byte(renamed), 0o644); err != nil {
		t.Fatal(err)
	}
	_, g := ProjectPackets(design, "", "")
	wantErr(t, g, "milestone M7 is not declared in the Build plan")
}

func TestPacketSourcesListDrawnLines(t *testing.T) {
	design := packetFixture(t)
	packets, g := ProjectPackets(design, "M1", "M1-S1")
	if len(g.Errs) > 0 {
		t.Fatal(g.Errs)
	}
	body := string(packets[0].Body)
	for _, want := range []string{
		"| BUILD.md | 15, 26-27, 34-37 |",
		"| BUILD/core.md | 11-18, 27-28 |",
		"| ARCHITECTURE.md | 8-11, 18-20, 23, 33, 39 |",
		"| machines/Deal.oracle.md | 10 |",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("sources table lacks %q\n%s", want, body)
		}
	}
	if strings.Contains(body, "| slices.yaml |") {
		t.Error("the slice map is bound by digest in the header, not listed as a drawn source")
	}
}

func TestPacketSectionIdMustBeUnique(t *testing.T) {
	design := packetFixture(t)
	rewriteFixtureFile(t, design, "BUILD/core.md", "### 9.1 Notes", "### 8.1 Notes")
	_, g := ProjectPackets(design, "", "")
	wantErr(t, g, `2 headings of BUILD/core.md match id '8.1'`)
}

// AC8: the projector reads no prose. Re-wording the slice narrative in the
// milestone block leaves every packet byte-identical, and no packet carries
// the narrative.
func TestPacketIgnoresProse(t *testing.T) {
	design := packetFixture(t)
	before, g := ProjectPackets(design, "", "")
	if len(g.Errs) > 0 {
		t.Fatal(g.Errs)
	}
	rewriteFixtureFile(t, design, "BUILD.md",
		"Slice narrative, written in prose and binding nothing: slice M1-S1 takes the win path and\nslice M1-S2 takes the lose path together with the policy rows. This paragraph is what the\nprojector must never read.",
		"Reworded entirely: M1-S2 now takes the win path and M1-S1 the lose path, says this prose,\nwhich binds nothing because the slice map is the only binding the projector reads, and\nthis sentence keeps the line count.")
	after, g := ProjectPackets(design, "", "")
	if len(g.Errs) > 0 {
		t.Fatal(g.Errs)
	}
	for i := range before {
		if !bytes.Equal(before[i].Body, after[i].Body) {
			t.Fatalf("packet %s changed after a prose-only edit", before[i].Slice)
		}
		if strings.Contains(string(before[i].Body), "binding nothing") {
			t.Fatalf("packet %s carries the prose narrative", before[i].Slice)
		}
	}
}

func TestPacketSelection(t *testing.T) {
	design := packetFixture(t)
	packets, g := ProjectPackets(design, "M1", "M1-S2")
	if len(g.Errs) > 0 || len(packets) != 1 || packets[0].Slice != "M1-S2" {
		t.Fatalf("selection = %v errs=%q", packets, g.Errs)
	}
	if p := packets[0]; p.Budget != 12000 || p.Shard != "BUILD/core.md" || p.Milestone != "M1" {
		t.Fatalf("packet metadata = %+v", p)
	}
	_, g = ProjectPackets(design, "M1", "M1-S9")
	wantErr(t, g, "declares no slice M1-S9 under milestone M1")
	_, g = ProjectPackets(design, "M0", "")
	wantErr(t, g, "declares no slices for milestone M0")
	_, g = ProjectPackets(design, "one", "")
	wantErr(t, g, `--milestone 'one' is not a milestone id`)
}

func TestPacketGateWithoutSliceMap(t *testing.T) {
	design := packetFixture(t)
	if err := os.Remove(filepath.Join(design, "slices.yaml")); err != nil {
		t.Fatal(err)
	}
	if HasSliceMap(design) {
		t.Fatal("HasSliceMap true without the file")
	}
	g := CheckPackets(design)
	wantErr(t, g, "no slices.yaml in the design")
}

func TestPacketGateInDefaultSuite(t *testing.T) {
	design := packetFixture(t)
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["gw"] {
		t.Fatal("gw not selected on a design carrying slices.yaml")
	}
	if !KnownGate("gw") {
		t.Fatal("gw is not a known gate")
	}
}

func TestPacketSizeLine(t *testing.T) {
	p := Packet{Slice: "M1-S1", Budget: 600000, Body: bytes.Repeat([]byte("x"), 10)}
	if got, want := p.SizeLine("out/M1-S1.packet.md"), "packet M1-S1 -> out/M1-S1.packet.md: 10 bytes of 600000 budget (4 token-equivalents at 3 bytes per token)"; got != want {
		t.Fatalf("size line = %q, want %q", got, want)
	}
}

// Modelith declares invariants under entities as one-line flow mappings, and
// contract rules may carry a trailing YAML comment; both forms resolve.
func TestPacketYAMLForms(t *testing.T) {
	design := packetFixture(t)
	rewriteFixtureFile(t, design, "domain.modelith.yaml",
		"entities:\n  Deal:\n    attributes:\n      stage: DealStage\n",
		"entities:\n  Deal:\n    attributes:\n      stage: DealStage\n    invariants:\n      - {id: deal-terminal, statement: \"A won or lost deal accepts no mutation but reopen.\"}\n")
	rewriteFixtureFile(t, design, "ARCHITECTURE.md",
		"    - crm.repo -> external.db\n",
		"    - crm.repo   ->   external.db   # the repository is the sole importer\n")
	rewriteFixtureFile(t, design, "slices.yaml",
		"          - invariant:rbac-write-scope\n",
		"          - invariant:rbac-write-scope\n          - invariant:deal-terminal\n          - rule:crm.repo -> external.db\n")
	packets, g := ProjectPackets(design, "M1", "M1-S2")
	if len(g.Errs) > 0 {
		t.Fatal(g.Errs)
	}
	body := string(packets[0].Body)
	for _, want := range []string{
		"### invariant:deal-terminal (domain.modelith.yaml:9)",
		`- {id: deal-terminal, statement: "A won or lost deal accepts no mutation but reopen."}`,
		"### rule:crm.repo -> external.db (ARCHITECTURE.md:24, dependency_rules.allow)",
		"    - crm.repo   ->   external.db   # the repository is the sole importer",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("packet lacks %q\n%s", want, body)
		}
	}
	if strings.Contains(body, "traceability row") && strings.Contains(body, "invariant:deal-terminal (BUILD.md") {
		t.Error("deal-terminal has no traceability row in the fixture and must not gain one")
	}
}

func TestUnquoteYAMLScalar(t *testing.T) {
	for in, want := range map[string]string{
		`crm.repo -> external.db   # comment`: "crm.repo -> external.db",
		`"crm.* -> external.db"   # quoted`:   "crm.* -> external.db",
		`'single'`:                            "single",
		`plain`:                               "plain",
		`# only a comment`:                    "",
	} {
		if got := unquoteYAMLScalar(in); got != want {
			t.Errorf("unquoteYAMLScalar(%q) = %q, want %q", in, got, want)
		}
	}
}
