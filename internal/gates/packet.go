// Gw-packet: per-slice packet projection. A manifest-mode design can exceed
// the context window of the executor that has to build one of its slices.
// The owner ruled that the sources are not split by hand and the executor is
// not swapped for a larger one: machinery projects, per slice, only what the
// slice cites. The binding is an AUTHORED, machine-readable slice map
// (design/slices.yaml); the projector reads ids and never prose. The gate
// holds the map to the design: every citation resolves, every packet fits its
// declared byte budget, and every obligation the milestone owes (the committed
// oracle ids its DoD cites, the same set Ga-accept binds evidence to) is
// claimed by exactly one slice or carries a recorded waiver. The projection is
// a pure function of the design bytes, so two runs produce identical packets.

package gates

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/portablepath"
	"github.com/RamXX/machinery/internal/version"
)

// PacketBytesPerToken is the fixed, documented divisor the size report uses
// to state a packet's byte count as token-equivalents. The gate is on bytes;
// the divisor only translates. Declare a budget as executor tokens times this.
const PacketBytesPerToken = 3

// SliceMapFile is the authored slice map, relative to the design root.
const SliceMapFile = "slices.yaml"

var (
	sliceIDRe        = regexp.MustCompile(`^(M(?:0|[1-9][0-9]*))-S([1-9][0-9]*)$`)
	sliceMilestoneRe = regexp.MustCompile(`^M(0|[1-9][0-9]*)$`)
	// packetCiteRe splits a typed citation into kind and target.
	packetCiteRe = regexp.MustCompile(`^([a-z]+):(.+)$`)
	// packetStableIDRe and packetTestIDRe are the bare oracle citations.
	packetStableIDRe = regexp.MustCompile(`^[A-Z][A-Z0-9]*-[0-9a-f]{6}$`)
	packetTestIDRe   = regexp.MustCompile(`^T-[A-Z][A-Z0-9]*-\d+$`)
	// packetTableSepRe matches a Markdown table separator line.
	packetTableSepRe = regexp.MustCompile(`^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)*\|?\s*$`)
	// packetHeadingNumRe is a heading's leading number token ("8.5", "11.").
	packetHeadingNumRe = regexp.MustCompile(`^(\d+(?:\.\d+)*)\.?(?:\s|$)`)
	// packetYAMLItemIDRe is a "- id: X" list item line, with its indent;
	// packetYAMLFlowItemIDRe is the one-line flow-mapping form the Modelith
	// renderer writes ("- {id: X, statement: ...}").
	packetYAMLItemIDRe     = regexp.MustCompile(`^(\s*)-\s+id:\s*(.+?)\s*$`)
	packetYAMLFlowItemIDRe = regexp.MustCompile(`^(\s*)-\s+\{\s*id:\s*([^,}]+?)\s*[,}]`)
	// packetYAMLKeyRe is a top-level or nested mapping key line.
	packetYAMLKeyRe = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*):\s*$`)
	// shardMilestoneRe is the marker shape a shard's own Build plan uses for
	// its per-milestone block: a list item opening with the bold milestone id
	// alone ("- **M1** (trust substrate). Core's slice: ..."). Root BUILD.md is
	// parsed with Gb's milestoneRe, never this one, so a milestone: citation
	// of the root reads exactly the block Gb and Ga read.
	shardMilestoneRe = regexp.MustCompile(`(?m)^[ \t]*(?:[-*][ \t]+|\d+\.[ \t]+)?\*\*M(\d+)(?:\s*[-:]\s*[^*]+?\.?)?\*\*`)
	// packetYAMLListRe is a scalar list item line.
	packetYAMLListRe = regexp.MustCompile(`^(\s*)-\s+(.+?)\s*$`)
	backtickSpanRe   = regexp.MustCompile("`([^`]+)`")
)

// HasSliceMap reports whether the design carries an authored slice map; Gw
// auto-activates on it.
func HasSliceMap(design string) bool {
	has, err := probeRegularFile(design, SliceMapFile)
	return has || err != nil
}

// Packet is one projected executor packet.
type Packet struct {
	Milestone string
	Slice     string
	Shard     string
	Budget    int
	Body      []byte
}

// TokenEquivalents states the packet size under the documented divisor,
// rounded up.
func (p Packet) TokenEquivalents() int {
	return (len(p.Body) + PacketBytesPerToken - 1) / PacketBytesPerToken
}

// SizeLine is the one-line size report the command prints per packet.
func (p Packet) SizeLine(dest string) string {
	return fmt.Sprintf("packet %s -> %s: %d bytes of %d budget (%d token-equivalents at %d bytes per token)",
		p.Slice, dest, len(p.Body), p.Budget, p.TokenEquivalents(), PacketBytesPerToken)
}

// CheckPackets implements Gw-packet: the whole slice map is projected in
// memory and held; nothing is written.
func CheckPackets(design string) *Gate {
	_, g := ProjectPackets(design, "", "")
	return g
}

// ProjectPackets projects the selected slices (every milestone when milestone
// is "", every slice of the milestone when slice is ""). The returned gate
// carries every finding of the whole map, because the coverage rule is a
// property of the milestone, and the packets are nil whenever the gate has an
// error: a packet that failed the gate is never handed to an executor.
func ProjectPackets(design, milestone, slice string) ([]Packet, *Gate) {
	g := NewGate("Gw-packet  per-slice packet projection")
	g.startOrder()
	if milestone != "" && !sliceMilestoneRe.MatchString(milestone) {
		g.Errs = append(g.Errs, "--milestone "+ir.Repr(milestone)+" is not a milestone id of the form M<n>")
		return nil, g
	}
	if slice != "" && !sliceIDRe.MatchString(slice) {
		g.Errs = append(g.Errs, "--slice "+ir.Repr(slice)+" is not a slice id of the form M<n>-S<k>")
		return nil, g
	}
	has, probeErr := probeRegularFile(design, SliceMapFile)
	if probeErr != nil {
		g.Errs = append(g.Errs, probeErr.Error())
		return nil, g
	}
	if !has {
		g.Errs = append(g.Errs, "no "+SliceMapFile+" in the design; the packet gate was requested but no slice map binds slices to what they cite (author "+SliceMapFile+", or drop gw from the gate list)")
		return nil, g
	}
	d := &packetDesign{design: design, g: g, files: map[string]*packetFile{}}
	sm := d.loadSliceMap()
	if sm == nil {
		return nil, g
	}
	fixtureUsers := sm.fixtureConsumers()
	fixtureBindings := 0
	for _, users := range fixtureUsers {
		fixtureBindings += len(users)
	}
	if fixtureBindings > 0 {
		g.Count("fixture modules", len(fixtureUsers))
		g.Count("fixture bindings", fixtureBindings)
	}
	plan := d.rootPlan()
	if plan == nil {
		return nil, g
	}
	d.indexOracles()
	var out []Packet
	selectedMilestone := false
	selectedSlice := false
	for _, ms := range sm.milestones {
		block, ok := plan.blocks[ms.num]
		if !ok {
			g.Errs = append(g.Errs, SliceMapFile+": milestone "+ms.id+" is not declared in the Build plan of BUILD.md; the slice map binds milestones the plan declares")
			continue
		}
		g.Count("milestones")
		obligations := d.milestoneObligations(block)
		ledger := d.claimLedger(ms, obligations)
		var projected []*packetProjection
		for i := range ms.slices {
			sl := &ms.slices[i]
			p := d.projectSlice(ms, sl, plan, block, obligations, ledger, fixtureUsers)
			projected = append(projected, p)
			g.Count("slices")
			if len(p.body) > sl.budget {
				g.Errs = append(g.Errs, fmt.Sprintf("%s: slice %s projects to %d bytes, over its declared budget of %d (%d token-equivalents at %d bytes per token); cite less, as subsections rather than sections, or split the slice and move its claims", SliceMapFile, sl.id, len(p.body), sl.budget, (len(p.body)+PacketBytesPerToken-1)/PacketBytesPerToken, PacketBytesPerToken))
			}
			g.CheckedExtra(fmt.Sprintf("%s %d/%d bytes", sl.id, len(p.body), sl.budget))
		}
		if milestone != "" && ms.id != milestone {
			continue
		}
		selectedMilestone = true
		for _, p := range projected {
			if slice != "" && p.slice.id != slice {
				continue
			}
			selectedSlice = true
			out = append(out, Packet{Milestone: ms.id, Slice: p.slice.id, Shard: p.slice.shard, Budget: p.slice.budget, Body: p.body})
		}
	}
	if milestone != "" && !selectedMilestone {
		g.Errs = append(g.Errs, SliceMapFile+": declares no slices for milestone "+milestone)
	}
	if slice != "" && selectedMilestone && !selectedSlice {
		g.Errs = append(g.Errs, SliceMapFile+": declares no slice "+slice+" under milestone "+milestone)
	}
	if len(g.Errs) > 0 {
		return nil, g
	}
	return out, g
}

// ---------------------------------------------------------------- slice map

type sliceMap struct {
	milestones []sliceMilestone
}

func (sm *sliceMap) fixtureConsumers() map[string][]string {
	users := map[string][]string{}
	for _, ms := range sm.milestones {
		for _, sl := range ms.slices {
			for _, fixture := range sl.fixtures {
				users[fixture] = append(users[fixture], sl.id)
			}
		}
	}
	for fixture := range users {
		sort.Strings(users[fixture])
	}
	return users
}

type sliceMilestone struct {
	id      string
	num     int
	slices  []sliceSpec
	waivers []sliceWaiver
}

type sliceSpec struct {
	id       string
	num      int
	shard    string
	budget   int
	fixtures []string
	cites    []string
}

type sliceWaiver struct {
	id     string
	reason string
}

var (
	sliceMapRootKeys      = map[string]bool{"schema": true, "milestones": true}
	sliceMapMilestoneKeys = map[string]bool{"id": true, "slices": true, "waivers": true}
	sliceMapSliceKeys     = map[string]bool{"id": true, "shard": true, "budget": true, "fixtures": true, "cites": true}
	sliceMapWaiverKeys    = map[string]bool{"id": true, "reason": true}
)

func (d *packetDesign) sliceMapErr(format string, args ...interface{}) {
	d.g.Errs = append(d.g.Errs, SliceMapFile+": "+fmt.Sprintf(format, args...))
}

func (d *packetDesign) unknownKeys(o *ir.Object, allowed map[string]bool, where string) {
	for _, k := range o.Keys() {
		if !allowed[k] {
			d.sliceMapErr("unsupported key %s in %s (the schema is closed; a typo must not remove an obligation)", ir.Repr(k), where)
		}
	}
}

func yamlString(o *ir.Object, key string) (string, bool) {
	v := o.Get2(key)
	if v == nil || v.Kind != ir.KindString {
		return "", false
	}
	s := strings.TrimSpace(v.AsString())
	return s, s != ""
}

func yamlInt(o *ir.Object, key string) (int, bool) {
	v := o.Get2(key)
	if v == nil || v.Kind != ir.KindNumber {
		return 0, false
	}
	n, err := strconv.Atoi(v.AsNumber().String())
	return n, err == nil
}

func (d *packetDesign) loadSliceMap() *sliceMap {
	f := d.file(SliceMapFile)
	if f == nil {
		return nil
	}
	root, err := ir.LoadYAML([]byte(f.text))
	if err != nil {
		d.sliceMapErr("is not valid YAML: %s", err.Error())
		return nil
	}
	o := root.AsObject()
	if o == nil {
		d.sliceMapErr("is not a mapping")
		return nil
	}
	d.unknownKeys(o, sliceMapRootKeys, "root")
	if n, ok := yamlInt(o, "schema"); !ok || n != 1 {
		d.sliceMapErr("schema must be the integer 1")
		return nil
	}
	mv := o.Get2("milestones")
	if mv == nil || mv.Kind != ir.KindArray || len(mv.AsArray()) == 0 {
		d.sliceMapErr("milestones must be a non-empty list of mappings")
		return nil
	}
	sm := &sliceMap{}
	seenMilestone := map[string]bool{}
	seenSlice := map[string]bool{}
	before := len(d.g.Errs)
	for i, item := range mv.AsArray() {
		where := fmt.Sprintf("milestones[%d]", i)
		mo := item.AsObject()
		if mo == nil {
			d.sliceMapErr("%s is not a mapping", where)
			continue
		}
		d.unknownKeys(mo, sliceMapMilestoneKeys, where)
		ms := sliceMilestone{}
		id, ok := yamlString(mo, "id")
		if !ok || !sliceMilestoneRe.MatchString(id) {
			d.sliceMapErr("%s.id must be a milestone id of the form M<n>", where)
			continue
		}
		if seenMilestone[id] {
			d.sliceMapErr("milestone %s is listed twice; one entry per milestone", id)
			continue
		}
		seenMilestone[id] = true
		ms.id = id
		ms.num, _ = strconv.Atoi(id[1:])
		where = "milestone " + id
		sv := mo.Get2("slices")
		if sv == nil || sv.Kind != ir.KindArray || len(sv.AsArray()) == 0 {
			d.sliceMapErr("%s: slices must be a non-empty list of mappings", where)
			continue
		}
		for j, sitem := range sv.AsArray() {
			swhere := fmt.Sprintf("%s slices[%d]", where, j)
			so := sitem.AsObject()
			if so == nil {
				d.sliceMapErr("%s is not a mapping", swhere)
				continue
			}
			d.unknownKeys(so, sliceMapSliceKeys, swhere)
			sl := sliceSpec{}
			sid, ok := yamlString(so, "id")
			if !ok {
				d.sliceMapErr("%s: id is required", swhere)
				continue
			}
			m := sliceIDRe.FindStringSubmatch(sid)
			if m == nil {
				d.sliceMapErr("%s: id %s is not of the form <milestone>-S<n>", swhere, ir.Repr(sid))
				continue
			}
			if m[1] != id {
				d.sliceMapErr("%s: slice id %s belongs to %s, not to %s", swhere, ir.Repr(sid), m[1], id)
				continue
			}
			if seenSlice[sid] {
				d.sliceMapErr("slice id %s is listed twice; slice ids are unique across the map", sid)
				continue
			}
			seenSlice[sid] = true
			sl.id = sid
			sl.num, _ = strconv.Atoi(m[2])
			swhere = "slice " + sid
			shard, ok := yamlString(so, "shard")
			if !ok {
				d.sliceMapErr("%s: shard is required and names exactly one design-relative .md file", swhere)
			} else if rel, ok := packetRelPath(shard); !ok || strings.ToLower(path.Ext(rel)) != ".md" {
				d.sliceMapErr("%s: shard %s is not a clean design-relative path to a .md file", swhere, ir.Repr(shard))
			} else {
				sl.shard = rel
				if has, perr := probeRegularFile(d.design, filepath.FromSlash(rel)); perr != nil {
					d.sliceMapErr("%s: shard %s cannot be inspected: %s", swhere, rel, perr.Error())
				} else if !has {
					d.sliceMapErr("%s: shard %s does not exist in the design as a regular file", swhere, rel)
				}
			}
			budget, ok := yamlInt(so, "budget")
			if !ok || budget <= 0 {
				d.sliceMapErr("%s: budget must be a positive integer number of bytes", swhere)
			}
			sl.budget = budget
			if fv := so.Get2("fixtures"); fv != nil {
				if fv.Kind != ir.KindArray || len(fv.AsArray()) == 0 {
					d.sliceMapErr("%s: fixtures must be a non-empty list of clean implementation-relative paths", swhere)
				} else {
					seenFixture := map[string]bool{}
					for k, fitem := range fv.AsArray() {
						if fitem == nil || fitem.Kind != ir.KindString {
							d.sliceMapErr("%s: fixtures[%d] is not a non-empty string", swhere, k)
							continue
						}
						fixture := fitem.AsString()
						if err := portablepath.ValidateRelative(fixture); err != nil {
							d.sliceMapErr("%s: fixture %s is not a clean implementation-relative path", swhere, ir.Repr(fixture))
							continue
						}
						if seenFixture[fixture] {
							d.sliceMapErr("%s: fixtures repeats %s; each fixture obligation is stated once per slice", swhere, ir.Repr(fixture))
							continue
						}
						seenFixture[fixture] = true
						sl.fixtures = append(sl.fixtures, fixture)
					}
				}
			}
			cv := so.Get2("cites")
			if cv == nil || cv.Kind != ir.KindArray || len(cv.AsArray()) == 0 {
				d.sliceMapErr("%s: cites must be a non-empty list of citations", swhere)
			} else {
				seenCite := map[string]bool{}
				for k, citem := range cv.AsArray() {
					if citem == nil || citem.Kind != ir.KindString || strings.TrimSpace(citem.AsString()) == "" {
						d.sliceMapErr("%s: cites[%d] is not a non-empty string", swhere, k)
						continue
					}
					c := strings.TrimSpace(citem.AsString())
					if seenCite[c] {
						d.sliceMapErr("%s: cites repeats %s; each citation is stated once", swhere, ir.Repr(c))
						continue
					}
					seenCite[c] = true
					sl.cites = append(sl.cites, c)
				}
			}
			ms.slices = append(ms.slices, sl)
		}
		if wv := mo.Get2("waivers"); wv != nil && wv.Kind != ir.KindNull {
			if wv.Kind != ir.KindArray {
				d.sliceMapErr("%s: waivers must be a list of mappings", where)
			} else {
				for j, witem := range wv.AsArray() {
					wwhere := fmt.Sprintf("%s waivers[%d]", where, j)
					wo := witem.AsObject()
					if wo == nil {
						d.sliceMapErr("%s is not a mapping", wwhere)
						continue
					}
					d.unknownKeys(wo, sliceMapWaiverKeys, wwhere)
					wid, ok := yamlString(wo, "id")
					if !ok {
						d.sliceMapErr("%s: id is required", wwhere)
						continue
					}
					reason, ok := yamlString(wo, "reason")
					if !ok {
						d.sliceMapErr("%s: reason is required; an unexplained waiver is an unanswered planning question", wwhere)
						continue
					}
					ms.waivers = append(ms.waivers, sliceWaiver{id: wid, reason: reason})
				}
			}
		}
		sm.milestones = append(sm.milestones, ms)
	}
	if len(d.g.Errs) > before {
		return nil
	}
	return sm
}

// packetRelPath accepts a clean, slash-separated, design-relative path.
func packetRelPath(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, "\\\x00") || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "#") {
		return "", false
	}
	if strings.Contains(raw, "://") {
		return "", false
	}
	if path.Clean(raw) != raw {
		return "", false
	}
	for _, seg := range strings.Split(raw, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", false
		}
	}
	return raw, true
}

// ------------------------------------------------------------ design reads

type packetDesign struct {
	design string
	g      *Gate
	files  map[string]*packetFile
	// oracle rows by test id and by stable id
	oracleByID map[string]*oracleRowRef
	oracleRows []*oracleRowRef
	// design files and line ranges the current packet drew from (reset per
	// slice); the packet's Sources table lists them
	touched map[string][]lineRange
}

// lineRange is one drawn 0-based inclusive line range of a source file.
type lineRange struct{ first, last int }

type packetFile struct {
	rel    string
	text   string
	lines  []string
	masked []string // fence-masked lines, index-aligned with lines
	sum    string
	// missing is set when the read failed; the finding is recorded once
	missing bool
}

// file reads one design file through the confinement reader, once.
func (d *packetDesign) file(rel string) *packetFile {
	if f, ok := d.files[rel]; ok {
		if f.missing {
			return nil
		}
		return f
	}
	f := &packetFile{rel: rel}
	d.files[rel] = f
	has, err := probeRegularFile(d.design, filepath.FromSlash(rel))
	if err != nil {
		d.g.Errs = append(d.g.Errs, rel+" cannot be inspected: "+err.Error())
		f.missing = true
		return nil
	}
	if !has {
		d.g.Errs = append(d.g.Errs, rel+" does not exist in the design as a regular file")
		f.missing = true
		return nil
	}
	body, err := readDesignFile(d.design, filepath.Join(d.design, filepath.FromSlash(rel)))
	if err != nil {
		d.g.Errs = append(d.g.Errs, rel+" is unreadable: "+err.Error())
		f.missing = true
		return nil
	}
	sum := sha256.Sum256(body)
	f.sum = hex.EncodeToString(sum[:])
	f.text = string(body)
	f.lines = strings.Split(f.text, "\n")
	f.masked = strings.Split(maskFences(f.text), "\n")
	return f
}

// touch records that the packet under construction drew lines [first,last]
// of rel.
func (d *packetDesign) touch(rel string, first, last int) {
	if d.touched != nil {
		d.touched[rel] = append(d.touched[rel], lineRange{first, last})
	}
}

// drawnRanges renders a file's drawn ranges, sorted and merged, 1-based.
func drawnRanges(rs []lineRange) string {
	sorted := append([]lineRange(nil), rs...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].first != sorted[j].first {
			return sorted[i].first < sorted[j].first
		}
		return sorted[i].last < sorted[j].last
	})
	var merged []lineRange
	for _, r := range sorted {
		if n := len(merged); n > 0 && r.first <= merged[n-1].last+1 {
			if r.last > merged[n-1].last {
				merged[n-1].last = r.last
			}
			continue
		}
		merged = append(merged, r)
	}
	var parts []string
	for _, r := range merged {
		if r.first == r.last {
			parts = append(parts, strconv.Itoa(r.first+1))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", r.first+1, r.last+1))
		}
	}
	return strings.Join(parts, ", ")
}

// -------------------------------------------------------------- root plan

type rootPlan struct {
	file   *packetFile
	blocks map[int]*planBlock
}

// planBlock is one milestone block located by line in its document.
type planBlock struct {
	rel    string
	num    int
	first  int // 0-based first line (the marker line)
	last   int // 0-based last line, inclusive, trailing blank lines trimmed
	text   string
	dod    int
	demo   int // 0-based line of the Demo: line; -1 when none
	status int // 0-based line of the Status: line; -1 when none
}

// planBlocksOf locates the milestone blocks of a document's Build plan
// section by line, with the same boundaries Gb uses (fence-masked, ending at
// the next marker or any heading).
func planBlocksOf(f *packetFile, marker *regexp.Regexp) (map[int]*planBlock, bool) {
	start, end := -1, len(f.masked)
	for i, line := range f.masked {
		level, title := headingText(line)
		if (level != 2 && level != 3) || !planHeadingRe.MatchString(title) {
			continue
		}
		start = i
		for j := i + 1; j < len(f.masked); j++ {
			if l, _ := headingText(f.masked[j]); l > 0 && l <= level {
				end = j
				break
			}
		}
		break
	}
	if start < 0 {
		return nil, false
	}
	body := strings.Join(f.masked[start+1:end], "\n")
	if first := firstNonBlankLine(body); strings.HasPrefix(strings.ToUpper(first), "N/A") {
		return nil, false
	}
	matches := marker.FindAllStringSubmatchIndex(body, -1)
	headings := headingOffsets(body)
	blocks := map[int]*planBlock{}
	for i, m := range matches {
		bend := len(body)
		if i+1 < len(matches) {
			bend = matches[i+1][0]
		}
		for _, h := range headings {
			if h > m[0] && h < bend {
				bend = h
				break
			}
		}
		num, err := strconv.Atoi(body[m[2]:m[3]])
		if err != nil {
			continue
		}
		firstLine := start + 1 + strings.Count(body[:m[0]], "\n")
		lastLine := start + 1 + strings.Count(body[:bend], "\n")
		if bend > 0 && body[bend-1] == '\n' {
			lastLine--
		}
		for lastLine > firstLine && strings.TrimSpace(f.lines[lastLine]) == "" {
			lastLine--
		}
		if _, dup := blocks[num]; dup {
			continue // Gb reports the duplicate
		}
		masked := strings.Join(f.masked[firstLine:lastLine+1], "\n")
		pb := &planBlock{rel: f.rel, num: num, first: firstLine, last: lastLine, text: masked, dod: dodIndex(masked), demo: -1, status: -1}
		for k := firstLine; k <= lastLine; k++ {
			if pb.demo < 0 && demoLineRe.MatchString(f.masked[k]) {
				pb.demo = k
			}
			if pb.status < 0 && milestoneStatusRe.MatchString(f.masked[k]) {
				pb.status = k
			}
		}
		blocks[num] = pb
	}
	return blocks, true
}

func (d *packetDesign) rootPlan() *rootPlan {
	f := d.file("BUILD.md")
	if f == nil {
		return nil
	}
	blocks, ok := planBlocksOf(f, milestoneRe)
	if !ok {
		d.g.Errs = append(d.g.Errs, "BUILD.md declares no Build plan section; the slice map binds milestones the plan declares")
		return nil
	}
	return &rootPlan{file: f, blocks: blocks}
}

// ----------------------------------------------------------- oracle index

type oracleRowRef struct {
	rel      string
	line     int // 0-based row line
	header   int // 0-based header line
	testID   string
	stableID string
}

func (d *packetDesign) indexOracles() {
	d.oracleByID = map[string]*oracleRowRef{}
	var rels []string
	for _, p := range sortedGlob(filepath.Join(d.design, "machines"), "*.oracle.md") {
		rels = append(rels, "machines/"+filepath.Base(p))
	}
	for _, name := range formalOracleNames {
		if has, err := probeRegularFile(d.design, filepath.Join("formal", name)); err == nil && has {
			rels = append(rels, "formal/"+name)
		}
	}
	dup := map[string]bool{}
	for _, rel := range rels {
		f := d.file(rel)
		if f == nil {
			continue
		}
		for _, tbl := range packetTables(f, 0, len(f.lines)-1) {
			ti := ir.FindCol(tbl.header, "test id")
			si := ir.FindCol(tbl.header, "stable id")
			if ti < 0 || si < 0 {
				continue
			}
			for _, r := range tbl.rows {
				ref := &oracleRowRef{rel: rel, line: r.line, header: tbl.headerLine}
				if ti < len(r.cells) {
					ref.testID = strings.TrimSpace(r.cells[ti])
				}
				if si < len(r.cells) {
					ref.stableID = strings.TrimSpace(r.cells[si])
				}
				if ref.stableID == "" || ref.stableID == "-" {
					continue
				}
				d.oracleRows = append(d.oracleRows, ref)
				for _, id := range []string{ref.testID, ref.stableID} {
					if id == "" || id == "-" {
						continue
					}
					if prev, ok := d.oracleByID[id]; ok && prev != ref && !dup[id] {
						dup[id] = true
						d.g.Errs = append(d.g.Errs, "oracle id "+id+" is declared in both "+prev.rel+" and "+rel+"; a citation of it would be ambiguous")
						continue
					}
					d.oracleByID[id] = ref
				}
			}
		}
	}
}

// ------------------------------------------------------------ obligations

// milestoneObligations returns the sorted stable ids of every committed
// oracle row the milestone's DoD cites whole-token, ORACLESET expanded: the
// set Ga-accept binds acceptance evidence to, normalized to stable ids.
func (d *packetDesign) milestoneObligations(block *planBlock) []string {
	if block.dod < 0 {
		return nil
	}
	dod := block.text[block.dod:]
	seen := map[string]bool{}
	var out []string
	for _, ref := range d.oracleRows {
		if seen[ref.stableID] {
			continue
		}
		if idTokenIn(ref.stableID, dod) || (ref.testID != "" && ref.testID != "-" && idTokenIn(ref.testID, dod)) {
			seen[ref.stableID] = true
			out = append(out, ref.stableID)
		}
	}
	for _, marker := range acceptanceOracleSetRe.FindAllStringSubmatch(dod, -1) {
		for _, ref := range d.oracleRows {
			if ref.rel == marker[1] && !seen[ref.stableID] {
				seen[ref.stableID] = true
				out = append(out, ref.stableID)
			}
		}
	}
	sort.Strings(out)
	return out
}

// claimLedger maps every obligation to the slice claiming it or the waiver
// releasing it, and reports the coverage findings: unclaimed, double-claimed,
// and claimed-and-waived obligations, and stale waivers.
func (d *packetDesign) claimLedger(ms sliceMilestone, obligations []string) map[string]string {
	owed := map[string]bool{}
	for _, id := range obligations {
		owed[id] = true
	}
	claims := map[string][]string{}
	for _, sl := range ms.slices {
		for _, id := range d.claimedRows(sl) {
			claims[id] = append(claims[id], sl.id)
		}
	}
	waived := map[string]string{}
	for _, w := range ms.waivers {
		ref, ok := d.oracleByID[w.id]
		if !ok {
			d.sliceMapErr("milestone %s waives %s, which no committed oracle declares", ms.id, ir.Repr(w.id))
			continue
		}
		if !owed[ref.stableID] {
			d.sliceMapErr("milestone %s waives %s, which its DoD does not cite; a waiver releases an obligation the milestone owes, and this one owes none by that id (stale waiver)", ms.id, w.id)
			continue
		}
		if _, dup := waived[ref.stableID]; dup {
			d.sliceMapErr("milestone %s waives %s twice", ms.id, ref.stableID)
			continue
		}
		waived[ref.stableID] = w.reason
	}
	ledger := map[string]string{}
	claimed, waivedCount := 0, 0
	for _, id := range obligations {
		by := claims[id]
		reason, isWaived := waived[id]
		switch {
		case len(by) > 1:
			sort.Strings(by)
			d.sliceMapErr("milestone %s obligation %s is claimed by %s; every obligation is claimed by exactly one slice", ms.id, id, strings.Join(by, " and "))
			ledger[id] = strings.Join(by, ", ")
		case len(by) == 1 && isWaived:
			d.sliceMapErr("milestone %s obligation %s is both claimed by %s and waived; drop one", ms.id, id, by[0])
			ledger[id] = by[0]
		case len(by) == 1:
			ledger[id] = by[0]
			claimed++
		case isWaived:
			ledger[id] = "waived: " + reason
			waivedCount++
		default:
			d.sliceMapErr("milestone %s obligation %s is claimed by no slice and carries no waiver; the projection would drop an obligation the milestone owes (cite it from one slice, or record a waiver with its reason)", ms.id, id)
			ledger[id] = "UNCLAIMED"
		}
	}
	d.g.Count("obligations owed", len(obligations))
	d.g.Count("obligations claimed", claimed)
	d.g.Count("obligations waived", waivedCount)
	return ledger
}

// claimedRows returns the stable ids a slice claims through bare id and
// oracleset citations, unique, in citation order. Unresolvable citations are
// reported by projectSlice; this pass only collects.
func (d *packetDesign) claimedRows(sl sliceSpec) []string {
	seen := map[string]bool{}
	var out []string
	add := func(id string) {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, c := range sl.cites {
		if packetStableIDRe.MatchString(c) || packetTestIDRe.MatchString(c) {
			if ref, ok := d.oracleByID[c]; ok {
				add(ref.stableID)
			}
			continue
		}
		if m := packetCiteRe.FindStringSubmatch(c); len(m) > 2 && m[1] == "oracleset" {
			for _, ref := range d.oracleRows {
				if ref.rel == m[2] {
					add(ref.stableID)
				}
			}
		}
	}
	return out
}

// --------------------------------------------------------------- excerpts

// excerpt is one verbatim copy of design lines under its citation.
type excerpt struct {
	cite   string
	rel    string
	first  int // 0-based
	last   int // 0-based inclusive
	lines  []string
	render string // "markdown" (marker-delimited), "table", or a fence language
	note   string // appended to the heading, e.g. the rule list name
	extra  []excerpt
}

func (e excerpt) heading() string {
	loc := fmt.Sprintf("%s:%d", e.rel, e.first+1)
	if e.last > e.first {
		loc = fmt.Sprintf("%s:%d-%d", e.rel, e.first+1, e.last+1)
	}
	if e.note != "" {
		return fmt.Sprintf("### %s (%s, %s)", e.cite, loc, e.note)
	}
	return fmt.Sprintf("### %s (%s)", e.cite, loc)
}

type packetProjection struct {
	slice *sliceSpec
	body  []byte
}

// sliceErr records a finding against one slice.
func (d *packetDesign) sliceErr(sl *sliceSpec, format string, args ...interface{}) {
	d.g.Errs = append(d.g.Errs, SliceMapFile+": slice "+sl.id+": "+fmt.Sprintf(format, args...))
}

// projectSlice resolves every citation of a slice and renders its packet.
func (d *packetDesign) projectSlice(ms sliceMilestone, sl *sliceSpec, plan *rootPlan, block *planBlock, obligations []string, ledger map[string]string, fixtureUsers map[string][]string) *packetProjection {
	d.touched = map[string][]lineRange{}
	d.touch(block.rel, block.first, block.first)
	if block.demo >= 0 {
		d.touch(block.rel, block.demo, block.demo)
	}
	if block.status >= 0 {
		d.touch(block.rel, block.status, block.status)
	}
	groups := map[string][]excerpt{}
	claimed := d.claimedRows(*sl)
	for _, c := range sl.cites {
		kind, ex, ok := d.resolve(sl, c, block)
		if !ok {
			continue
		}
		groups[kind] = append(groups[kind], ex...)
	}
	var b strings.Builder
	w := func(format string, args ...interface{}) { fmt.Fprintf(&b, format, args...) }
	w("# Execution packet %s\n\n", sl.id)
	w("Milestone: %s\nShard: %s\n", ms.id, sl.shard)
	w("Generated by `machinery packet` from %s (sha256 %s). DO NOT EDIT BY HAND.\n", SliceMapFile, d.files[SliceMapFile].sum)
	w("<!-- machinery-version: %s -->\n\n", version.Version)
	w("Every excerpt below is a verbatim copy of design bytes under its citation and its source\n")
	w("path:line (1-based, inclusive). The source is the authority; nothing here supersedes it.\n\n")

	// 1. milestone
	w("## 1. Milestone\n\n")
	w("- marker (%s:%d): %s\n", block.rel, block.first+1, strings.TrimSpace(plan.file.lines[block.first]))
	if block.demo >= 0 {
		w("- demo (%s:%d): %s\n", block.rel, block.demo+1, strings.TrimSpace(plan.file.lines[block.demo]))
	}
	if block.status >= 0 {
		w("- status (%s:%d): %s\n", block.rel, block.status+1, strings.TrimSpace(plan.file.lines[block.status]))
	} else {
		w("- status: open (the block states no Status line)\n")
	}
	w("\nObligations %s owes (the committed oracle ids its DoD cites, %s:%d-%d):\n\n", ms.id, block.rel, block.first+1, block.last+1)
	if len(obligations) == 0 {
		w("(none: the DoD cites no committed oracle id)\n")
	} else {
		w("| obligation | claimed by |\n|---|---|\n")
		for _, id := range obligations {
			w("| %s | %s |\n", id, ledger[id])
		}
	}
	w("\nThis slice claims %d of %d: %s\n\n", d.countOwned(claimed, obligations), len(obligations), joinOrNone(claimed))

	sections := []struct {
		title string
		kinds []string
	}{
		{"2. Sections and files", []string{"section", "milestone", "file"}},
		{"3. Oracle rows", []string{"oracle", "oracleset"}},
		{"4. Matrices", []string{"matrix"}},
		{"5. Architecture Contract", []string{"boundary", "external", "rule", "row"}},
		{"6. Invariants", []string{"invariant"}},
	}
	for _, s := range sections {
		w("## %s\n\n", s.title)
		n := 0
		for _, k := range s.kinds {
			for _, e := range groups[k] {
				d.renderExcerpt(&b, e)
				n++
			}
		}
		if n == 0 {
			w("(none cited)\n\n")
		}
	}

	acceptanceSection := 7
	if len(sl.fixtures) > 0 {
		w("## 7. Fixture obligations\n\n")
		for _, fixture := range sl.fixtures {
			w("- `%s` feeds suites of slices %s; run every consuming slice suite after changing it.\n", fixture, joinPacketList(fixtureUsers[fixture]))
		}
		w("\n")
		acceptanceSection++
	}

	// acceptance entry shape
	w("## %d. Acceptance entry shape\n\n", acceptanceSection)
	w("The milestone review commits `acceptance/%s.yaml` with this shape; `dod_ids` is the full\n", ms.id)
	w("obligation set of the milestone, every id exactly once, with the slice that claims it noted here.\n\n")
	w("```yaml\nmilestone: %d\ncommit: <the commit the review ran on>\nverdict: REJECTED\ndod_ids:\n", ms.num)
	for _, id := range obligations {
		w("  - %s  # %s\n", id, ledger[id])
	}
	if len(obligations) == 0 {
		w("  []\n")
	}
	w("attestations: []\nfindings: []\nreviewer: <who or what produced the review>\ndate: <YYYY-MM-DD>\n```\n\n")

	// sources: every file the packet drew from and the lines it drew. The
	// packet binds to the excerpted bytes, which it carries verbatim, and to
	// the slice map by digest; it does not bind to whole files, so an edit
	// elsewhere in a source file never changes a packet.
	w("## %d. Sources\n\n| path | lines drawn |\n|---|---|\n", acceptanceSection+1)
	var rels []string
	for rel := range d.touched {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		w("| %s | %s |\n", rel, drawnRanges(d.touched[rel]))
	}
	d.touched = nil
	return &packetProjection{slice: sl, body: []byte(b.String())}
}

func joinPacketList(items []string) string {
	switch len(items) {
	case 0:
		return "(none)"
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
	}
}

func (d *packetDesign) countOwned(claimed, obligations []string) int {
	owed := map[string]bool{}
	for _, id := range obligations {
		owed[id] = true
	}
	n := 0
	for _, id := range claimed {
		if owed[id] {
			n++
		}
	}
	return n
}

func joinOrNone(ids []string) string {
	if len(ids) == 0 {
		return "(no oracle rows cited)"
	}
	return strings.Join(ids, ", ")
}

// renderExcerpt writes one excerpt under its heading. Markdown excerpts are
// marker-delimited so their own headings and fences survive; everything else
// is fenced with a fence longer than any backtick run inside it.
func (d *packetDesign) renderExcerpt(b *strings.Builder, e excerpt) {
	b.WriteString(e.heading())
	b.WriteString("\n\n")
	body := strings.Join(e.lines, "\n")
	switch e.render {
	case "markdown":
		fmt.Fprintf(b, "<!-- begin %s -->\n%s\n<!-- end %s -->\n\n", e.cite, body, e.cite)
	case "table":
		b.WriteString(body)
		b.WriteString("\n\n")
	default:
		fence := "```"
		for run := 3; ; run++ {
			fence = strings.Repeat("`", run)
			if !strings.Contains(body, fence) {
				break
			}
		}
		fmt.Fprintf(b, "%s%s\n%s\n%s\n\n", fence, e.render, body, fence)
	}
	for _, x := range e.extra {
		d.renderExcerpt(b, x)
	}
}

// resolve turns one citation into excerpts, recording a finding when it does
// not resolve to exactly one. kind groups the excerpt in the packet.
func (d *packetDesign) resolve(sl *sliceSpec, c string, block *planBlock) (kind string, out []excerpt, ok bool) {
	if packetStableIDRe.MatchString(c) || packetTestIDRe.MatchString(c) {
		ref, found := d.oracleByID[c]
		if !found {
			d.sliceErr(sl, "cites oracle id %s, which no committed oracle declares (machines/*.oracle.md, formal/Policy.oracle.md, formal/Isolation.oracle.md)", c)
			return "", nil, false
		}
		return "oracle", []excerpt{d.oracleRowExcerpt(ref)}, true
	}
	m := packetCiteRe.FindStringSubmatch(c)
	if m == nil {
		d.sliceErr(sl, "citation %s has no recognized shape: a bare oracle id, or one of section:, milestone:, file:, oracleset:, matrix:, boundary:, external:, rule:, row:, invariant:", ir.Repr(c))
		return "", nil, false
	}
	kind, target := m[1], strings.TrimSpace(m[2])
	switch kind {
	case "section":
		rel, id, ok := splitHash2(target)
		if !ok {
			d.sliceErr(sl, "section citation %s is not <path>#<heading id>", ir.Repr(c))
			return "", nil, false
		}
		if !d.shardBound(sl, rel, c) {
			return "", nil, false
		}
		f := d.mdFile(sl, rel, c)
		if f == nil {
			return "", nil, false
		}
		first, last, why := sectionRange(f, id)
		if why != "" {
			d.sliceErr(sl, "%s: %s", c, why)
			return "", nil, false
		}
		d.touch(rel, first, last)
		return kind, []excerpt{{cite: c, rel: rel, first: first, last: last, lines: f.lines[first : last+1], render: "markdown"}}, true
	case "milestone":
		rel, ok := packetRelPath(target)
		if !ok || strings.ToLower(path.Ext(rel)) != ".md" {
			d.sliceErr(sl, "milestone citation %s does not name a design-relative .md file", ir.Repr(c))
			return "", nil, false
		}
		if !d.shardBound(sl, rel, c) {
			return "", nil, false
		}
		f := d.mdFile(sl, rel, c)
		if f == nil {
			return "", nil, false
		}
		marker := shardMilestoneRe
		if rel == "BUILD.md" {
			marker = milestoneRe
		}
		blocks, found := planBlocksOf(f, marker)
		if !found {
			d.sliceErr(sl, "%s: %s declares no Build plan section", c, rel)
			return "", nil, false
		}
		pb, found := blocks[block.num]
		if !found {
			d.sliceErr(sl, "%s: %s declares no milestone M%d block (a root block opens with **M%d - <title>**, a shard block with a list item opening **M%d**)", c, rel, block.num, block.num, block.num)
			return "", nil, false
		}
		d.touch(rel, pb.first, pb.last)
		return kind, []excerpt{{cite: c, rel: rel, first: pb.first, last: pb.last, lines: f.lines[pb.first : pb.last+1], render: "markdown"}}, true
	case "file":
		rel, ok := packetRelPath(target)
		if !ok {
			d.sliceErr(sl, "file citation %s is not a clean design-relative path", ir.Repr(c))
			return "", nil, false
		}
		if rel == SliceMapFile {
			d.sliceErr(sl, "%s: the slice map is the projection's input, not an excerpt", c)
			return "", nil, false
		}
		f := d.file(rel)
		if f == nil {
			d.sliceErr(sl, "%s does not resolve", c)
			return "", nil, false
		}
		render := fenceLanguage(rel)
		last := len(f.lines) - 1
		for last > 0 && f.lines[last] == "" {
			last--
		}
		d.touch(rel, 0, last)
		return kind, []excerpt{{cite: c, rel: rel, first: 0, last: last, lines: f.lines[:last+1], render: render}}, true
	case "oracleset":
		rel, ok := packetRelPath(target)
		if !ok {
			d.sliceErr(sl, "oracleset citation %s is not a clean design-relative path", ir.Repr(c))
			return "", nil, false
		}
		var refs []*oracleRowRef
		for _, ref := range d.oracleRows {
			if ref.rel == rel {
				refs = append(refs, ref)
			}
		}
		if len(refs) == 0 {
			d.sliceErr(sl, "%s resolves no oracle rows (the oracle sets are machines/<Machine>.oracle.md, formal/Policy.oracle.md and formal/Isolation.oracle.md)", c)
			return "", nil, false
		}
		f := d.file(rel)
		first, last := refs[0].header, refs[len(refs)-1].line
		d.touch(rel, first, last)
		return kind, []excerpt{{cite: c, rel: rel, first: first, last: last, lines: f.lines[first : last+1], render: "table"}}, true
	case "matrix":
		if target == "" || strings.ContainsAny(target, "/\\.#") {
			d.sliceErr(sl, "matrix citation %s does not name a machine", ir.Repr(c))
			return "", nil, false
		}
		rel := "machines/" + target + ".matrix.md"
		f := d.file(rel)
		if f == nil {
			d.sliceErr(sl, "%s does not resolve (no %s)", c, rel)
			return "", nil, false
		}
		last := len(f.lines) - 1
		for last > 0 && f.lines[last] == "" {
			last--
		}
		d.touch(rel, 0, last)
		return kind, []excerpt{{cite: c, rel: rel, first: 0, last: last, lines: f.lines[:last+1], render: "markdown"}}, true
	case "boundary", "external":
		ex, why := d.contractItem(kind, target, c)
		if why != "" {
			d.sliceErr(sl, "%s: %s", c, why)
			return "", nil, false
		}
		return kind, []excerpt{ex}, true
	case "rule":
		ex, why := d.contractRule(target, c)
		if why != "" {
			d.sliceErr(sl, "%s: %s", c, why)
			return "", nil, false
		}
		return kind, []excerpt{ex}, true
	case "row":
		rel, rest, ok := splitHash2(target)
		var sectionID, key string
		if ok {
			sectionID, key, ok = splitHash2(rest)
		}
		if !ok {
			d.sliceErr(sl, "row citation %s is not <path>#<section id>#<key>", ir.Repr(c))
			return "", nil, false
		}
		if !d.shardBound(sl, rel, c) {
			return "", nil, false
		}
		f := d.mdFile(sl, rel, c)
		if f == nil {
			return "", nil, false
		}
		first, last, why := sectionRange(f, sectionID)
		if why != "" {
			d.sliceErr(sl, "%s: %s", c, why)
			return "", nil, false
		}
		ex, why := tableRowExcerpt(f, first, last, key, c)
		if why != "" {
			d.sliceErr(sl, "%s: %s", c, why)
			return "", nil, false
		}
		d.touch(rel, ex.first, ex.last)
		return kind, []excerpt{ex}, true
	case "invariant":
		ex, why := d.invariantExcerpt(target, c)
		if why != "" {
			d.sliceErr(sl, "%s: %s", c, why)
			return "", nil, false
		}
		return kind, []excerpt{ex}, true
	}
	d.sliceErr(sl, "citation %s has unknown kind %s", ir.Repr(c), ir.Repr(kind))
	return "", nil, false
}

// shardBound holds a slice to its one shard: a BUILD/ file it cites must be
// that shard.
func (d *packetDesign) shardBound(sl *sliceSpec, rel, c string) bool {
	if strings.HasPrefix(rel, "BUILD/") && rel != sl.shard {
		d.sliceErr(sl, "%s cites %s, but the slice is bound to shard %s; a slice never pulls another shard's content", c, rel, sl.shard)
		return false
	}
	return true
}

// mdFile reads a cited Markdown file, reporting a bad path or a missing file
// against the slice.
func (d *packetDesign) mdFile(sl *sliceSpec, rel, c string) *packetFile {
	clean, ok := packetRelPath(rel)
	if !ok || clean != rel || strings.ToLower(path.Ext(rel)) != ".md" {
		d.sliceErr(sl, "%s does not name a clean design-relative .md file", c)
		return nil
	}
	f := d.file(rel)
	if f == nil {
		d.sliceErr(sl, "%s does not resolve", c)
	}
	return f
}

func splitHash2(s string) (string, string, bool) {
	i := strings.Index(s, "#")
	if i <= 0 || i == len(s)-1 {
		return "", "", false
	}
	return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:]), true
}

func fenceLanguage(rel string) string {
	switch strings.ToLower(path.Ext(rel)) {
	case ".md":
		return "markdown"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".dsl":
		return "dsl"
	case ".tla", ".cfg", ".als":
		return "text"
	default:
		return "text"
	}
}

// sectionRange resolves a heading id in a Markdown file to the 0-based line
// range of the heading and its subtree. why is non-empty when zero or several
// headings match.
func sectionRange(f *packetFile, id string) (first, last int, why string) {
	type hit struct{ line, level int }
	var hits []hit
	var candidates []string
	for i, line := range f.masked {
		level, text := headingText(line)
		if level == 0 {
			continue
		}
		num := ""
		if m := packetHeadingNumRe.FindStringSubmatch(text); m != nil {
			num = m[1]
		}
		if num == id || text == id {
			hits = append(hits, hit{i, level})
			candidates = append(candidates, fmt.Sprintf("line %d %s", i+1, ir.Repr(text)))
		}
	}
	switch len(hits) {
	case 0:
		return 0, 0, "no heading of " + f.rel + " has id " + ir.Repr(id) + " (a heading's ids are its leading number token and its full text)"
	case 1:
	default:
		return 0, 0, fmt.Sprintf("%d headings of %s match id %s (%s); cite the full heading text of the one you mean", len(hits), f.rel, ir.Repr(id), strings.Join(candidates, "; "))
	}
	first = hits[0].line
	last = len(f.lines) - 1
	for j := first + 1; j < len(f.masked); j++ {
		if l, _ := headingText(f.masked[j]); l > 0 && l <= hits[0].level {
			last = j - 1
			break
		}
	}
	for last > first && strings.TrimSpace(f.lines[last]) == "" {
		last--
	}
	return first, last, ""
}

// ------------------------------------------------------------- tables

type packetTable struct {
	headerLine int
	header     []string
	rows       []packetTableRow
}

type packetTableRow struct {
	line  int
	cells []string
}

func splitTableCells(line string) []string {
	t := strings.TrimSpace(line)
	t = strings.TrimPrefix(t, "|")
	t = strings.TrimSuffix(t, "|")
	parts := strings.Split(t, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// packetTables walks the fence-masked lines in [first,last] and returns every
// Markdown table (a header line, a separator line, then data rows) with
// source line numbers.
func packetTables(f *packetFile, first, last int) []packetTable {
	var out []packetTable
	isRow := func(i int) bool {
		return i >= 0 && i < len(f.masked) && strings.HasPrefix(strings.TrimSpace(f.masked[i]), "|")
	}
	for i := first; i+1 <= last; i++ {
		if !isRow(i) || !isRow(i+1) || !packetTableSepRe.MatchString(f.masked[i+1]) {
			continue
		}
		tbl := packetTable{headerLine: i, header: splitTableCells(f.lines[i])}
		j := i + 2
		for ; j <= last && isRow(j); j++ {
			tbl.rows = append(tbl.rows, packetTableRow{line: j, cells: splitTableCells(f.lines[j])})
		}
		out = append(out, tbl)
		i = j - 1
	}
	return out
}

// rowKey is the first cell's key: its first backticked span, else the cell.
func rowKey(cell string) string {
	if m := backtickSpanRe.FindStringSubmatch(cell); m != nil {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(cell)
}

// tableRowExcerpt finds the one row keyed key in the tables of [first,last].
func tableRowExcerpt(f *packetFile, first, last int, key, cite string) (excerpt, string) {
	var found []excerpt
	for _, tbl := range packetTables(f, first, last) {
		for _, r := range tbl.rows {
			if len(r.cells) == 0 || rowKey(r.cells[0]) != key {
				continue
			}
			found = append(found, excerpt{
				cite:   cite,
				rel:    f.rel,
				first:  r.line,
				last:   r.line,
				lines:  []string{f.lines[tbl.headerLine], f.lines[tbl.headerLine+1], f.lines[r.line]},
				render: "table",
			})
		}
	}
	switch len(found) {
	case 0:
		return excerpt{}, fmt.Sprintf("no table row under %s:%d-%d has first-cell key %s", f.rel, first+1, last+1, ir.Repr(key))
	case 1:
		return found[0], ""
	default:
		var lines []string
		for _, e := range found {
			lines = append(lines, strconv.Itoa(e.first+1))
		}
		return excerpt{}, fmt.Sprintf("%d table rows under %s:%d-%d have first-cell key %s (lines %s); a row citation must be unique", len(found), f.rel, first+1, last+1, ir.Repr(key), strings.Join(lines, ", "))
	}
}

func (d *packetDesign) oracleRowExcerpt(ref *oracleRowRef) excerpt {
	f := d.file(ref.rel)
	d.touch(ref.rel, ref.line, ref.line)
	return excerpt{
		cite:   ref.stableID,
		rel:    ref.rel,
		first:  ref.line,
		last:   ref.line,
		lines:  []string{f.lines[ref.header], f.lines[ref.header+1], f.lines[ref.line]},
		render: "table",
	}
}

// ------------------------------------------------- Architecture Contract

// contractFence locates the Architecture Contract fence body by line.
func (d *packetDesign) contractFence() (f *packetFile, first, last int, why string) {
	f = d.file("ARCHITECTURE.md")
	if f == nil {
		return nil, 0, 0, "ARCHITECTURE.md does not resolve"
	}
	body, ok := ir.ContractFence(f.text)
	if !ok {
		return nil, 0, 0, "ARCHITECTURE.md carries no Architecture Contract fence"
	}
	idx := strings.Index(f.text, body)
	if idx < 0 {
		return nil, 0, 0, "ARCHITECTURE.md contract fence could not be located by line"
	}
	first = strings.Count(f.text[:idx], "\n")
	last = first + strings.Count(body, "\n")
	return f, first, last, ""
}

func yamlIndent(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

// yamlItemID matches a list item declaring an id in block or flow form and
// returns its indent and id.
func yamlItemID(line string) (indent int, id string, ok bool) {
	if m := packetYAMLItemIDRe.FindStringSubmatch(line); m != nil {
		return len(m[1]), unquoteYAMLScalar(m[2]), true
	}
	if m := packetYAMLFlowItemIDRe.FindStringSubmatch(line); m != nil {
		return len(m[1]), unquoteYAMLScalar(m[2]), true
	}
	return 0, "", false
}

// unquoteYAMLScalar returns a plain or quoted scalar's value with any trailing
// YAML comment removed (" # ..." after an unquoted value, or after the closing
// quote of a quoted one).
func unquoteYAMLScalar(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') {
		if end := strings.IndexByte(s[1:], s[0]); end >= 0 {
			return s[1 : end+1]
		}
		return s
	}
	if i := strings.Index(s, " #"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if strings.HasPrefix(s, "#") {
		return ""
	}
	return s
}

// contractItem finds the boundaries[] or externals[] item with the given id.
func (d *packetDesign) contractItem(list, id, cite string) (excerpt, string) {
	f, first, last, why := d.contractFence()
	if why != "" {
		return excerpt{}, why
	}
	listKey := list == "boundary"
	want := "externals"
	if listKey {
		want = "boundaries"
	}
	inList := false
	for i := first; i <= last; i++ {
		line := f.lines[i]
		if m := packetYAMLKeyRe.FindStringSubmatch(line); len(m) > 2 && len(m[1]) == 0 {
			inList = m[2] == want
			continue
		}
		if !inList {
			continue
		}
		indent, got, ok := yamlItemID(line)
		if !ok || got != id {
			continue
		}
		end := i
		for j := i + 1; j <= last; j++ {
			t := strings.TrimSpace(f.lines[j])
			if t != "" && yamlIndent(f.lines[j]) <= indent {
				break
			}
			end = j
		}
		for end > i && strings.TrimSpace(f.lines[end]) == "" {
			end--
		}
		d.touch(f.rel, i, end)
		return excerpt{cite: cite, rel: f.rel, first: i, last: end, lines: f.lines[i : end+1], render: "yaml"}, ""
	}
	return excerpt{}, fmt.Sprintf("the Architecture Contract declares no %s item with id %s", want, ir.Repr(id))
}

func normalizeEdge(s string) (string, bool) {
	parts := strings.Split(s, "->")
	if len(parts) != 2 {
		return "", false
	}
	src, dst := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if src == "" || dst == "" {
		return "", false
	}
	return src + " -> " + dst, true
}

// contractRule finds one dependency_rules allow, deny, or baseline line.
func (d *packetDesign) contractRule(edge, cite string) (excerpt, string) {
	want, ok := normalizeEdge(edge)
	if !ok {
		return excerpt{}, "rule citation is not <src> -> <dst>"
	}
	f, first, last, why := d.contractFence()
	if why != "" {
		return excerpt{}, why
	}
	inRules := false
	sub := ""
	for i := first; i <= last; i++ {
		line := f.lines[i]
		if m := packetYAMLKeyRe.FindStringSubmatch(line); m != nil {
			if len(m[1]) == 0 {
				inRules = m[2] == "dependency_rules"
				sub = ""
			} else if inRules {
				sub = m[2]
			}
			continue
		}
		if !inRules || (sub != "allow" && sub != "deny" && sub != "baseline") {
			continue
		}
		m := packetYAMLListRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		got, ok := normalizeEdge(unquoteYAMLScalar(m[2]))
		if ok && got == want {
			d.touch(f.rel, i, i)
			return excerpt{cite: cite, rel: f.rel, first: i, last: i, lines: []string{line}, render: "yaml", note: "dependency_rules." + sub}, ""
		}
	}
	return excerpt{}, fmt.Sprintf("no dependency_rules allow, deny, or baseline entry reads %s", ir.Repr(want))
}

// ----------------------------------------------------------- invariants

// invariantExcerpt finds the domain model's invariant entry and, when the
// root BUILD.md carries a traceability matrix, the invariant's row of it.
func (d *packetDesign) invariantExcerpt(id, cite string) (excerpt, string) {
	f := d.file("domain.modelith.yaml")
	if f == nil {
		return excerpt{}, "domain.modelith.yaml does not resolve"
	}
	// Modelith declares invariants at the top level and under entities, so
	// any "invariants:" list at any nesting is searched; a list ends at the
	// first non-blank line at or above its key's indent.
	inList, listIndent := false, 0
	for i, line := range f.lines {
		if m := packetYAMLKeyRe.FindStringSubmatch(line); m != nil {
			inList = m[2] == "invariants"
			listIndent = len(m[1])
			continue
		}
		if inList && strings.TrimSpace(line) != "" && yamlIndent(line) <= listIndent {
			inList = false
		}
		if !inList {
			continue
		}
		indent, got, ok := yamlItemID(line)
		if !ok || got != id {
			continue
		}
		end := i
		for j := i + 1; j < len(f.lines); j++ {
			t := strings.TrimSpace(f.lines[j])
			if t != "" && yamlIndent(f.lines[j]) <= indent {
				break
			}
			end = j
		}
		for end > i && strings.TrimSpace(f.lines[end]) == "" {
			end--
		}
		d.touch(f.rel, i, end)
		ex := excerpt{cite: cite, rel: f.rel, first: i, last: end, lines: f.lines[i : end+1], render: "yaml"}
		if row, ok := d.traceabilityRow(id, cite); ok {
			ex.extra = append(ex.extra, row)
		}
		return ex, ""
	}
	return excerpt{}, fmt.Sprintf("domain.modelith.yaml declares no invariant with id %s", ir.Repr(id))
}

// traceabilityRow finds the invariant's row of the root traceability matrix:
// the first BUILD.md table whose header names an invariant column.
func (d *packetDesign) traceabilityRow(id, cite string) (excerpt, bool) {
	f := d.file("BUILD.md")
	if f == nil {
		return excerpt{}, false
	}
	for _, tbl := range packetTables(f, 0, len(f.lines)-1) {
		if colContaining(tbl.header, "invariant") < 0 {
			continue
		}
		for _, r := range tbl.rows {
			if len(r.cells) == 0 || rowKey(r.cells[0]) != id {
				continue
			}
			d.touch(f.rel, r.line, r.line)
			return excerpt{
				cite:   cite,
				rel:    f.rel,
				first:  r.line,
				last:   r.line,
				lines:  []string{f.lines[tbl.headerLine], f.lines[tbl.headerLine+1], f.lines[r.line]},
				render: "table",
				note:   "traceability row",
			}, true
		}
		return excerpt{}, false
	}
	return excerpt{}, false
}

// ------------------------------------------------------------- snapshot

// ProjectPackets runs the projection inside an acquired design snapshot, so a
// command that writes packets reads the design under the same custody every
// other reader uses.
func (s *Snapshot) ProjectPackets(milestone, slice string) ([]Packet, *Gate) {
	if err := validateDesignInventory(s.design); err != nil {
		return nil, &Gate{Title: "G0-snapshot", Errs: []string{"cannot inventory design artifacts: " + s.LogicalError(err).Error()}}
	}
	if err := validateActivationDiscovery(s.design); err != nil {
		return nil, &Gate{Title: "G0-snapshot", Errs: []string{s.LogicalError(err).Error()}}
	}
	return ProjectPackets(s.design, milestone, slice)
}
