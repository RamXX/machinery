// Declaration producer for the closed v1 executable-assurance contract
// (docs/test-assurance-contract.md sections 3-4, 7): closed plan/milestone
// decoders, the exact canonical review projection C, the typed design
// payload tree digest, and finalized Validate reconciliation. No process is
// launched and no evidence is issued here; capture/register/replay and the
// normal gates consume this producer.

package tdd

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/tdd/protocol"
)

// ---- closed scalar grammars ----

var (
	digestRe     = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	idRe         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)
	milestoneRe  = regexp.MustCompile(`^M(0|[1-9][0-9]*)$`)
	uuidRe       = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	intLiteralRe = regexp.MustCompile(`^-?[0-9]+$`)
	// windowsReserved are device basenames rejected in every path segment.
	windowsReserved = map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
		"COM6": true, "COM7": true, "COM8": true, "COM9": true,
		"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
		"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
	}
)

func validDigest(s string) bool      { return digestRe.MatchString(s) }
func validID(s string) bool          { return idRe.MatchString(s) }
func validMilestoneID(s string) bool { return milestoneRe.MatchString(s) }
func validUUID(s string) bool        { return uuidRe.MatchString(s) }

// validatePath enforces the closed Path grammar: a normalized /-relative
// path other than ".", printable ASCII only (non-ASCII is rejected outright
// so no Unicode aliasing survives), no absolute/drive/backslash forms, no
// traversal or dot segments, no reserved device basenames, no trailing
// dot/space segments.
func validatePath(p string) error {
	if p == "" || p == protocol.RepositoryRoot {
		return fmt.Errorf("path %q must be a nonempty relative path other than \".\"", p)
	}
	if strings.HasPrefix(p, "/") {
		return fmt.Errorf("absolute path %q is rejected; paths are repository-root relative", p)
	}
	if strings.Contains(p, "\\") {
		return fmt.Errorf("path %q contains a backslash; use \"/\" separators", p)
	}
	if len(p) >= 2 && p[1] == ':' {
		return fmt.Errorf("path %q looks like a drive-letter path", p)
	}
	if !utf8.ValidString(p) {
		return fmt.Errorf("path %q is not valid UTF-8", p)
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("path %q contains control characters", p)
		}
		if r > 0x7e {
			return fmt.Errorf("path %q contains non-ASCII characters; v1 control paths are ASCII-only so no Unicode alias survives", p)
		}
	}
	if c := path.Clean(p); c != p {
		return fmt.Errorf("path %q is not clean (canonical form %q)", p, c)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" {
			return fmt.Errorf("path %q contains an empty segment", p)
		}
		if seg == "." || seg == ".." {
			return fmt.Errorf("path %q must not contain %q segments", p, seg)
		}
		if strings.HasSuffix(seg, ".") || strings.HasSuffix(seg, " ") {
			return fmt.Errorf("path segment %q in %q ends with a dot or space (an OS alias)", seg, p)
		}
		if windowsReserved[strings.ToUpper(seg)] {
			return fmt.Errorf("path segment %q in %q uses a reserved device basename", seg, p)
		}
	}
	return nil
}

// validateRootPath accepts exactly "." or a valid Path; "." always names the
// explicitly supplied repository root, never an ambient working directory.
func validateRootPath(p string) error {
	if p == protocol.RepositoryRoot {
		return nil
	}
	return validatePath(p)
}

// rejectCaseAliases fails a set-semantics list whose members collide after
// case folding: an unsafe case alias must never pass as distinct evidence.
func rejectCaseAliases(items []string, where string) error {
	seen := map[string]string{}
	for _, p := range items {
		fold := strings.ToLower(p)
		if prev, ok := seen[fold]; ok && prev != p {
			return fmt.Errorf("INVALID_SCHEMA: %s: %q is an unsafe case alias of %q", where, p, prev)
		}
		seen[fold] = p
	}
	return nil
}

func rejectDuplicates(items []string, where string) error {
	seen := map[string]bool{}
	for _, p := range items {
		if seen[p] {
			return fmt.Errorf("INVALID_SCHEMA: %s: duplicate entry %q; lists with set semantics reject duplicates", where, p)
		}
		seen[p] = true
	}
	return nil
}

// ---- canonical encoding and the review projection C ----

// canonString emits the contract's canonical JSON string: unescaped UTF-8
// except quote, backslash and control characters (\n \r \t shorthands,
// \u00xx otherwise).
func canonString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"':
			b.WriteString(`\"`)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func canonInt(n int64) string { return strconv.FormatInt(n, 10) }

// canonObj emits an object with UTF-8-lexicographically sorted keys and no
// whitespace; arrays stay in their schema/authored order.
func canonObj(members map[string]string) string {
	keys := make([]string, 0, len(members))
	for k := range members {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(canonString(k))
		b.WriteByte(':')
		b.WriteString(members[k])
	}
	b.WriteByte('}')
	return b.String()
}

func canonArr(items []string) string {
	return "[" + strings.Join(items, ",") + "]"
}

func canonOptRef(ref string) string {
	if ref == "" {
		return "null"
	}
	return canonString(ref)
}

func canonTestRef(r TestRef) string {
	return canonObj(map[string]string{
		"design": canonString(r.Design), "milestone": canonString(r.Milestone),
		"suite": canonString(r.Suite), "test": canonString(r.Test),
	})
}

func canonTestRefs(rs []TestRef) string {
	items := make([]string, 0, len(rs))
	for _, r := range rs {
		items = append(items, canonTestRef(r))
	}
	return canonArr(items)
}

func canonStrs(xs []string) string {
	items := make([]string, 0, len(xs))
	for _, x := range xs {
		items = append(items, canonString(x))
	}
	return canonArr(items)
}

func canonReviewNull() string { return "null" }

// canonNative emits only the set members of the closed native-identity
// union in canonical key order.
func canonNative(n NativeID) string {
	members := map[string]string{}
	if n.Package != "" {
		members["package"] = canonString(n.Package)
	}
	if n.Test != "" {
		members["test"] = canonString(n.Test)
	}
	if n.Source != "" {
		members["source"] = canonString(n.Source)
	}
	if len(n.Path) > 0 {
		members["path"] = canonStrs(n.Path)
	}
	if n.Line != 0 {
		members["line"] = canonInt(n.Line)
	}
	if n.Column != 0 {
		members["column"] = canonInt(n.Column)
	}
	if n.Module != "" {
		members["module"] = canonString(n.Module)
	}
	if n.Class != "" {
		members["class"] = canonString(n.Class)
	}
	if n.Method != "" {
		members["method"] = canonString(n.Method)
	}
	if n.Name != "" {
		members["name"] = canonString(n.Name)
	}
	if n.File != "" {
		members["file"] = canonString(n.File)
	}
	return canonObj(members)
}

func canonExpectations(es []Expectation) string {
	items := make([]string, 0, len(es))
	for _, e := range es {
		items = append(items, canonObj(map[string]string{
			"assertions": canonStrs(e.Assertions), "outcome": canonString(e.Outcome),
			"test": canonTestRef(e.Test),
		}))
	}
	return canonArr(items)
}

// ReviewProjectionPlan returns projection C of the plan: canonical form
// with ONLY runtime_obligations[i].review replaced by null. There is no
// recursive ignore-by-key; every other review-shaped value survives.
func ReviewProjectionPlan(p Plan) []byte {
	ms := make([]string, 0, len(p.Milestones))
	for _, m := range p.Milestones {
		ms = append(ms, canonObj(map[string]string{"id": canonString(m.ID), "manifest": canonString(m.Manifest)}))
	}
	ros := make([]string, 0, len(p.RuntimeObligations))
	for _, ro := range p.RuntimeObligations {
		srs := make([]string, 0, len(ro.SourceRefs))
		for _, s := range ro.SourceRefs {
			srs = append(srs, canonObj(map[string]string{
				"anchor": canonString(s.Anchor), "digest": canonString(s.Digest), "path": canonString(s.Path),
			}))
		}
		ros = append(ros, canonObj(map[string]string{
			"category": canonString(ro.Category), "disposition": canonString(ro.Disposition),
			"id": canonString(ro.ID), "owner": canonString(ro.Owner), "reason": canonString(ro.Reason),
			"review": canonReviewNull(), "source_refs": canonArr(srs), "tests": canonTestRefs(ro.Tests),
		}))
	}
	return []byte(canonObj(map[string]string{
		"design": canonString(p.Design), "milestones": canonArr(ms),
		"project_id": canonString(p.ProjectID), "runtime_obligations": canonArr(ros),
		"schema": canonString(p.Schema),
	}))
}

// ReviewProjectionManifest returns projection C of the manifest: canonical
// form with ONLY the top-level review and every variants[i].review replaced
// by null.
func ReviewProjectionManifest(m Manifest) []byte {
	suites := make([]string, 0, len(m.Suites))
	for _, s := range m.Suites {
		tests := make([]string, 0, len(s.Tests))
		for _, tst := range s.Tests {
			as := make([]string, 0, len(tst.Assertions))
			for _, a := range tst.Assertions {
				as = append(as, canonObj(map[string]string{
					"helper": canonString(a.Helper), "id": canonString(a.ID),
					"line": canonInt(a.Line), "source": canonString(a.Source),
				}))
			}
			tests = append(tests, canonObj(map[string]string{
				"assertions": canonArr(as), "id": canonString(tst.ID),
				"native": canonNative(tst.Native), "role": canonString(tst.Role),
				"source": canonString(tst.Source),
			}))
		}
		env := make([]string, 0, len(s.Environment))
		for _, e := range s.Environment {
			env = append(env, canonObj(map[string]string{"name": canonString(e.Name), "value": canonString(e.Value)}))
		}
		suites = append(suites, canonObj(map[string]string{
			"adapter": canonString(s.Adapter), "dependency_roots": canonStrs(s.DependencyRoots),
			"environment": canonArr(env), "files": canonStrs(s.Files), "id": canonString(s.ID),
			"root": canonString(s.Root),
			"runtime": canonObj(map[string]string{
				"closure": canonString(s.Runtime.Closure), "platform": canonString(s.Runtime.Platform),
				"profile": canonString(s.Runtime.Profile), "version": canonString(s.Runtime.Version),
			}),
			"tests": canonArr(tests),
		}))
	}
	obls := make([]string, 0, len(m.Obligations))
	for _, o := range m.Obligations {
		obls = append(obls, canonObj(map[string]string{
			"key": canonObj(map[string]string{
				"design": canonString(o.Key.Design), "id": canonString(o.Key.ID),
				"kind": canonString(o.Key.Kind), "owner": canonString(o.Key.Owner),
			}),
			"negative": canonTestRefs(o.Negative), "positive": canonTestRefs(o.Positive),
		}))
	}
	variants := make([]string, 0, len(m.Variants))
	for _, v := range m.Variants {
		variants = append(variants, canonObj(map[string]string{
			"expected": canonExpectations(v.Expected), "id": canonString(v.ID),
			"kind": canonString(v.Kind), "pair": canonString(v.Pair),
			"review": canonReviewNull(), "source": canonOptRef(v.Source),
			"target_tests": canonTestRefs(v.TargetTests),
		}))
	}
	checks := make([]string, 0, len(m.Checks))
	for _, c := range m.Checks {
		checks = append(checks, canonObj(map[string]string{
			"id": canonString(c.ID), "inputs": canonStrs(c.Inputs),
			"kind": canonString(c.Kind), "profile": canonString(c.Profile),
		}))
	}
	controls := make([]string, 0, len(m.RedControls))
	for _, rc := range m.RedControls {
		controls = append(controls, canonObj(map[string]string{
			"assertion": canonString(rc.Assertion), "safe_variant": canonString(rc.SafeVariant),
			"test": canonTestRef(rc.Test),
		}))
	}
	subjects := make([]string, 0, len(m.SubjectEntries))
	for _, se := range m.SubjectEntries {
		subjects = append(subjects, canonObj(map[string]string{"kind": canonString(se.Kind), "path": canonString(se.Path)}))
	}
	return []byte(canonObj(map[string]string{
		"baseline": canonOptRef(m.Baseline), "checks": canonArr(checks),
		"frozen_roots": canonStrs(m.FrozenRoots), "id": canonString(m.ID),
		"implementation_roots": canonStrs(m.ImplementationRoots),
		"limits": canonObj(map[string]string{
			"wall_ms": canonInt(m.Limits.WallMS), "cleanup_ms": canonInt(m.Limits.CleanupMS),
			"stdout_bytes": canonInt(m.Limits.StdoutBytes), "stderr_bytes": canonInt(m.Limits.StderrBytes),
			"event_bytes": canonInt(m.Limits.EventBytes), "event_count": canonInt(m.Limits.EventCount),
			"jobs": canonInt(m.Limits.Jobs), "bundle_bytes": canonInt(m.Limits.BundleBytes),
			"entries": canonInt(m.Limits.Entries), "depth": canonInt(m.Limits.Depth),
		}),
		"obligations": canonArr(obls), "predecessor": canonOptRef(m.Predecessor),
		"red_controls": canonArr(controls), "red_expectations": canonExpectations(m.RedExpectations),
		"repository": canonString(m.Repository), "revision": canonInt(m.Revision),
		"review": canonReviewNull(), "schema": canonString(m.Schema),
		"subject_entries": canonArr(subjects), "suites": canonArr(suites),
		"variants": canonArr(variants),
	}))
}

// ReviewSubjectDigest derives review.subject_digest = SHA256 over the
// domain, the U64-big-endian length and bytes of C(projected control), then
// the raw 32 bytes of the authoritative design-payload digest.
func ReviewSubjectDigest(projection []byte, payloadDigest string) (string, error) {
	if !validDigest(payloadDigest) {
		return "", fmt.Errorf("INVALID_SCHEMA: design payload digest %q is not a digest", payloadDigest)
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(payloadDigest, "sha256:"))
	if err != nil || len(raw) != sha256.Size {
		return "", fmt.Errorf("INVALID_SCHEMA: design payload digest %q does not decode", payloadDigest)
	}
	h := sha256.New()
	h.Write([]byte(protocol.ReviewDomain))
	var lenBuf [8]byte
	binary.BigEndian.PutUint64(lenBuf[:], uint64(len(projection)))
	h.Write(lenBuf[:])
	h.Write(projection)
	h.Write(raw)
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// ---- typed design payload tree digest ----

type payloadEntry struct {
	rel  string
	dir  bool
	perm uint32
	size int64
	dig  [sha256.Size]byte
}

// DesignPayloadDigest computes the typed topology digest of the design
// payload: domain machinery.tdd.tree/v1, entries sorted by UTF-8 path, each
// encoded as kind byte, U64 path length + bytes, U32 portable permission
// bits, U64 logical size, 32 raw digest bytes (zero for directories) and
// the U64-length ASCII role "design". The exact assurance/ control
// namespace and VCS .git directories are excluded (they are separately
// captured control inputs, not payload); symlinks and special entries fail
// closed.
func DesignPayloadDigest(designDir string) (string, error) {
	fi, err := os.Lstat(designDir)
	if err != nil {
		return "", fmt.Errorf("MISSING_CONTRACT: design root %s: %w", designDir, err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("MISSING_CONTRACT: design root %s is not a directory", designDir)
	}
	var entries []payloadEntry
	root := payloadEntry{rel: ".", dir: true, perm: uint32(fi.Mode().Perm())}
	entries = append(entries, root)
	seen := 1
	maxDepth := int(protocol.LimitDepthMax)
	walkErr := filepath.WalkDir(designDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == designDir {
			return nil
		}
		rel, rerr := filepath.Rel(designDir, p)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		depth := strings.Count(rel, "/") + 1
		if depth > maxDepth {
			return fmt.Errorf("INVALID_SCHEMA: design payload exceeds depth %d at %s", maxDepth, rel)
		}
		if d.IsDir() {
			base := filepath.Base(p)
			if rel == protocol.ControlDirName || base == ".git" {
				return filepath.SkipDir
			}
			info, ierr := d.Info()
			if ierr != nil {
				return ierr
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("INVALID_SCHEMA: symlink directory %q in the design payload is rejected", rel)
			}
			entries = append(entries, payloadEntry{rel: rel, dir: true, perm: uint32(info.Mode().Perm())})
			seen++
		} else if d.Type().IsRegular() {
			info, ierr := d.Info()
			if ierr != nil {
				return ierr
			}
			data, rerr := os.ReadFile(p)
			if rerr != nil {
				return rerr
			}
			if int64(len(data)) != info.Size() {
				return fmt.Errorf("STALE_INPUT: file %q changed while hashing", rel)
			}
			entries = append(entries, payloadEntry{
				rel: rel, perm: uint32(info.Mode().Perm()), size: info.Size(),
				dig: sha256.Sum256(data),
			})
			seen++
		} else {
			return fmt.Errorf("INVALID_SCHEMA: special entry %q in the design payload is rejected (symlink/device/socket)", rel)
		}
		if seen > int(protocol.LimitEntriesMax) {
			return fmt.Errorf("OUTPUT_LIMIT: design payload exceeds %d entries", protocol.LimitEntriesMax)
		}
		return nil
	})
	if walkErr != nil {
		return "", walkErr
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].rel < entries[j].rel })
	h := sha256.New()
	h.Write([]byte(protocol.TreeDigestDomain))
	var b8 [8]byte
	var b4 [4]byte
	for _, e := range entries {
		if e.dir {
			h.Write([]byte{0})
		} else {
			h.Write([]byte{1})
		}
		binary.BigEndian.PutUint64(b8[:], uint64(len(e.rel)))
		h.Write(b8[:])
		h.Write([]byte(e.rel))
		binary.BigEndian.PutUint32(b4[:], e.perm)
		h.Write(b4[:])
		binary.BigEndian.PutUint64(b8[:], uint64(e.size))
		h.Write(b8[:])
		h.Write(e.dig[:])
		role := protocol.DesignPayloadRole
		binary.BigEndian.PutUint64(b8[:], uint64(len(role)))
		h.Write(b8[:])
		h.Write([]byte(role))
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// ---- closed JSON control reading and decode helpers ----

// readControlJSON reads one control document with the fixed 16 MiB
// read/parse cap, regular-file/symlink rejection and duplicate-key plus
// trailing-data rejection (the ir decoder preserves key order and rejects
// duplicates at parse time).
func readControlJSON(p string) ([]byte, *ir.Value, error) {
	before, err := os.Lstat(p)
	if err != nil {
		return nil, nil, fmt.Errorf("INVALID_SCHEMA: cannot read %s: %w", p, err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("INVALID_SCHEMA: %s must be a regular non-symlink control file", p)
	}
	if before.Size() > protocol.ControlInputMaxBytes {
		return nil, nil, fmt.Errorf("OUTPUT_LIMIT: %s exceeds the %d-byte control-input cap", p, protocol.ControlInputMaxBytes)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, nil, fmt.Errorf("INVALID_SCHEMA: cannot read %s: %w", p, err)
	}
	after, err := os.Lstat(p)
	if err != nil || !os.SameFile(before, after) || after.Size() != before.Size() {
		return nil, nil, fmt.Errorf("STALE_INPUT: %s changed while reading", p)
	}
	v, err := ir.LoadMachineJSONBytes(p, data)
	if err != nil {
		return nil, nil, fmt.Errorf("INVALID_SCHEMA: %s: %w", p, err)
	}
	return data, v, nil
}

func asObject(v *ir.Value, where string) (*ir.Object, error) {
	if v == nil || v.IsNull() {
		return nil, fmt.Errorf("INVALID_SCHEMA: %s is null; the closed schema forbids null here", where)
	}
	o := v.AsObject()
	if o == nil {
		return nil, fmt.Errorf("INVALID_SCHEMA: %s must be an object", where)
	}
	return o, nil
}

func exactKeys(o *ir.Object, where string, keys ...string) error {
	allowed := make(map[string]bool, len(keys))
	for _, k := range keys {
		allowed[k] = true
	}
	for _, k := range o.Keys() {
		if !allowed[k] {
			return fmt.Errorf("INVALID_SCHEMA: %s: unknown key %q (closed record)", where, k)
		}
	}
	for _, k := range keys {
		if !o.Has(k) {
			return fmt.Errorf("INVALID_SCHEMA: %s: required key %q is missing", where, k)
		}
	}
	return nil
}

func reqStr(o *ir.Object, key, where string) (string, error) {
	v := o.Get2(key)
	if v == nil {
		return "", fmt.Errorf("INVALID_SCHEMA: %s: required key %q is missing", where, key)
	}
	if v.IsNull() {
		return "", fmt.Errorf("INVALID_SCHEMA: %s: %s is null; the closed schema forbids null here", where, key)
	}
	if v.Kind != ir.KindString {
		return "", fmt.Errorf("INVALID_SCHEMA: %s: %s must be a string", where, key)
	}
	return v.AsString(), nil
}

// optStr reads a string-or-null field; "" encodes null.
func optStr(o *ir.Object, key, where string) (string, error) {
	v := o.Get2(key)
	if v == nil {
		return "", fmt.Errorf("INVALID_SCHEMA: %s: required key %q is missing", where, key)
	}
	if v.IsNull() {
		return "", nil
	}
	if v.Kind != ir.KindString {
		return "", fmt.Errorf("INVALID_SCHEMA: %s: %s must be a string or null", where, key)
	}
	return v.AsString(), nil
}

func reqInt(o *ir.Object, key, where string) (int64, error) {
	v := o.Get2(key)
	if v == nil {
		return 0, fmt.Errorf("INVALID_SCHEMA: %s: required key %q is missing", where, key)
	}
	if v.IsNull() {
		return 0, fmt.Errorf("INVALID_SCHEMA: %s: %s is null; the closed schema forbids null here", where, key)
	}
	if v.Kind != ir.KindNumber {
		return 0, fmt.Errorf("INVALID_SCHEMA: %s: %s must be an integer", where, key)
	}
	literal := string(v.AsNumber())
	if !intLiteralRe.MatchString(literal) {
		return 0, fmt.Errorf("INVALID_SCHEMA: %s: %s must be an integer literal (no fraction or exponent): %s", where, key, literal)
	}
	n, err := strconv.ParseInt(literal, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("INVALID_SCHEMA: %s: %s integer out of range: %s", where, key, literal)
	}
	return n, nil
}

func reqObjArray(o *ir.Object, key, where string) ([]*ir.Object, error) {
	v := o.Get2(key)
	if v == nil {
		return nil, fmt.Errorf("INVALID_SCHEMA: %s: required key %q is missing", where, key)
	}
	if v.IsNull() {
		return nil, fmt.Errorf("INVALID_SCHEMA: %s: %s is null; the closed schema forbids null here", where, key)
	}
	if v.Kind != ir.KindArray {
		return nil, fmt.Errorf("INVALID_SCHEMA: %s: %s must be an array", where, key)
	}
	out := make([]*ir.Object, 0, len(v.AsArray()))
	for i, item := range v.AsArray() {
		io, err := asObject(item, fmt.Sprintf("%s[%d]", where, i))
		if err != nil {
			return nil, err
		}
		out = append(out, io)
	}
	return out, nil
}

func reqStrArray(o *ir.Object, key, where string) ([]string, error) {
	v := o.Get2(key)
	if v == nil {
		return nil, fmt.Errorf("INVALID_SCHEMA: %s: required key %q is missing", where, key)
	}
	if v.IsNull() {
		return nil, fmt.Errorf("INVALID_SCHEMA: %s: %s is null; the closed schema forbids null here", where, key)
	}
	if v.Kind != ir.KindArray {
		return nil, fmt.Errorf("INVALID_SCHEMA: %s: %s must be an array", where, key)
	}
	out := make([]string, 0, len(v.AsArray()))
	for i, item := range v.AsArray() {
		if item == nil || item.Kind != ir.KindString {
			return nil, fmt.Errorf("INVALID_SCHEMA: %s.%s[%d] must be a string", where, key, i)
		}
		out = append(out, item.AsString())
	}
	return out, nil
}

func decodeTestRef(o *ir.Object, where string) (TestRef, error) {
	if err := exactKeys(o, where, "design", "milestone", "suite", "test"); err != nil {
		return TestRef{}, err
	}
	d, err := reqStr(o, "design", where+".design")
	if err != nil {
		return TestRef{}, err
	}
	if err := validateRootPath(d); err != nil {
		return TestRef{}, fmt.Errorf("INVALID_SCHEMA: %s.design: %w", where, err)
	}
	m, err := reqStr(o, "milestone", where+".milestone")
	if err != nil {
		return TestRef{}, err
	}
	if !validMilestoneID(m) {
		return TestRef{}, fmt.Errorf("INVALID_SCHEMA: %s.milestone: milestone id %q is not canonical", where, m)
	}
	s, err := reqStr(o, "suite", where+".suite")
	if err != nil {
		return TestRef{}, err
	}
	if !validID(s) {
		return TestRef{}, fmt.Errorf("INVALID_SCHEMA: %s.suite: %q is not an ID", where, s)
	}
	t, err := reqStr(o, "test", where+".test")
	if err != nil {
		return TestRef{}, err
	}
	if !validID(t) {
		return TestRef{}, fmt.Errorf("INVALID_SCHEMA: %s.test: %q is not an ID", where, t)
	}
	return TestRef{Design: d, Milestone: m, Suite: s, Test: t}, nil
}

func decodeTestRefs(o *ir.Object, key, where string) ([]TestRef, error) {
	arr, err := reqObjArray(o, key, where)
	if err != nil {
		return nil, err
	}
	out := make([]TestRef, 0, len(arr))
	for i, io := range arr {
		r, err := decodeTestRef(io, fmt.Sprintf("%s.%s[%d]", where, key, i))
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func decodeReview(o *ir.Object, where string) (Review, error) {
	if err := exactKeys(o, where, "reviewer", "rationale", "subject_digest"); err != nil {
		return Review{}, err
	}
	r, err := reqStr(o, "reviewer", where+".reviewer")
	if err != nil {
		return Review{}, err
	}
	if r == "" {
		return Review{}, fmt.Errorf("INVALID_SCHEMA: %s.reviewer must be nonempty; an anonymous judgment binds nothing", where)
	}
	ra, err := reqStr(o, "rationale", where+".rationale")
	if err != nil {
		return Review{}, err
	}
	if ra == "" {
		return Review{}, fmt.Errorf("INVALID_SCHEMA: %s.rationale must be nonempty; an unexplained judgment binds nothing", where)
	}
	d, err := reqStr(o, "subject_digest", where+".subject_digest")
	if err != nil {
		return Review{}, err
	}
	if !validDigest(d) {
		return Review{}, fmt.Errorf("INVALID_SCHEMA: %s.subject_digest %q is not a digest", where, d)
	}
	return Review{Reviewer: r, Rationale: ra, SubjectDigest: d}, nil
}

// ---- plan decoding ----

func decodePlan(doc *ir.Value, designDir, payloadDigest string, raw []byte) (Plan, error) {
	root, err := asObject(doc, "plan.json")
	if err != nil {
		return Plan{}, err
	}
	if err := exactKeys(root, "plan.json", "schema", "design", "milestones", "project_id", "runtime_obligations"); err != nil {
		return Plan{}, err
	}
	schema, _ := reqStr(root, "schema", "plan.json.schema")
	if schema != protocol.SchemaPlan {
		return Plan{}, fmt.Errorf("UNSUPPORTED_VERSION: plan schema %q is not %s", schema, protocol.SchemaPlan)
	}
	p := Plan{Schema: schema, raw: raw, payloadDigest: payloadDigest, designDir: designDir}
	design, err := reqStr(root, "design", "plan.json.design")
	if err != nil {
		return Plan{}, err
	}
	if err := validateRootPath(design); err != nil {
		return Plan{}, fmt.Errorf("INVALID_SCHEMA: plan.json.design: %w", err)
	}
	if design != protocol.RepositoryRoot {
		clean := filepath.ToSlash(filepath.Clean(designDir))
		if clean != design && !strings.HasSuffix(clean, "/"+design) {
			return Plan{}, fmt.Errorf("INVALID_SCHEMA: plan design %q does not match the loaded design directory %s", design, clean)
		}
	}
	p.Design = design
	msObjs, err := reqObjArray(root, "milestones", "plan.json.milestones")
	if err != nil {
		return Plan{}, err
	}
	seenM := map[string]bool{}
	for i, mo := range msObjs {
		where := fmt.Sprintf("plan.json.milestones[%d]", i)
		if err := exactKeys(mo, where, "id", "manifest"); err != nil {
			return Plan{}, err
		}
		id, err := reqStr(mo, "id", where+".id")
		if err != nil {
			return Plan{}, err
		}
		if !validMilestoneID(id) {
			return Plan{}, fmt.Errorf("INVALID_SCHEMA: %s: milestone id %q is not canonical (M plus an unsigned decimal, no leading zeroes except M0)", where, id)
		}
		if seenM[id] {
			return Plan{}, fmt.Errorf("INVALID_SCHEMA: %s: duplicate milestone %q; milestones are unique within a design", where, id)
		}
		seenM[id] = true
		manifest, err := reqStr(mo, "manifest", where+".manifest")
		if err != nil {
			return Plan{}, err
		}
		if err := validatePath(manifest); err != nil {
			return Plan{}, fmt.Errorf("INVALID_SCHEMA: %s.manifest: %w", where, err)
		}
		want := path.Join(design, protocol.ControlDirName, protocol.MilestonesDirName, id+".json")
		if manifest != want {
			return Plan{}, fmt.Errorf("INVALID_SCHEMA: %s.manifest: path %q must live in the design control namespace as %q", where, manifest, want)
		}
		rel := manifest
		if design != protocol.RepositoryRoot {
			rel = strings.TrimPrefix(manifest, design+"/")
		}
		mp := filepath.Join(designDir, filepath.FromSlash(rel))
		if fi, err := os.Lstat(mp); err != nil || fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
			return Plan{}, fmt.Errorf("MISSING_CONTRACT: %s.manifest: milestone %s manifest %q does not exist under the design root", where, id, manifest)
		}
		p.Milestones = append(p.Milestones, PlanMilestone{ID: id, Manifest: manifest})
	}
	pid, err := reqStr(root, "project_id", "plan.json.project_id")
	if err != nil {
		return Plan{}, err
	}
	if !validUUID(pid) {
		return Plan{}, fmt.Errorf("INVALID_SCHEMA: plan.json.project_id %q is not a UUID", pid)
	}
	p.ProjectID = pid
	roObjs, err := reqObjArray(root, "runtime_obligations", "plan.json.runtime_obligations")
	if err != nil {
		return Plan{}, err
	}
	seenRO := map[string]bool{}
	for i, ro := range roObjs {
		where := fmt.Sprintf("plan.json.runtime_obligations[%d]", i)
		if err := exactKeys(ro, where, "id", "category", "owner", "source_refs", "disposition", "reason", "review", "tests"); err != nil {
			return Plan{}, err
		}
		id, err := reqStr(ro, "id", where+".id")
		if err != nil {
			return Plan{}, err
		}
		if !validID(id) {
			return Plan{}, fmt.Errorf("INVALID_SCHEMA: %s.id %q is not an ID", where, id)
		}
		if seenRO[id] {
			return Plan{}, fmt.Errorf("INVALID_SCHEMA: %s: duplicate runtime obligation id %q", where, id)
		}
		seenRO[id] = true
		cat, err := reqStr(ro, "category", where+".category")
		if err != nil {
			return Plan{}, err
		}
		if !protocolCategory(cat) {
			return Plan{}, fmt.Errorf("INVALID_SCHEMA: %s.category: invalid category %q (closed set: %s)", where, cat, strings.Join(protocol.RuntimeCategories, ", "))
		}
		owner, err := reqStr(ro, "owner", where+".owner")
		if err != nil {
			return Plan{}, err
		}
		if owner == "" {
			return Plan{}, fmt.Errorf("INVALID_SCHEMA: %s.owner must be nonempty; an ownerless runtime obligation pools", where)
		}
		srObjs, err := reqObjArray(ro, "source_refs", where+".source_refs")
		if err != nil {
			return Plan{}, err
		}
		var srs []SourceRef
		for j, so := range srObjs {
			swhere := fmt.Sprintf("%s.source_refs[%d]", where, j)
			if err := exactKeys(so, swhere, "path", "anchor", "digest"); err != nil {
				return Plan{}, err
			}
			sp, err := reqStr(so, "path", swhere+".path")
			if err != nil {
				return Plan{}, err
			}
			if err := validatePath(sp); err != nil {
				return Plan{}, fmt.Errorf("INVALID_SCHEMA: %s.path: %w", swhere, err)
			}
			anchor, err := reqStr(so, "anchor", swhere+".anchor")
			if err != nil {
				return Plan{}, err
			}
			dig, err := reqStr(so, "digest", swhere+".digest")
			if err != nil {
				return Plan{}, err
			}
			if !validDigest(dig) {
				return Plan{}, fmt.Errorf("INVALID_SCHEMA: %s.digest %q is not a digest", swhere, dig)
			}
			srs = append(srs, SourceRef{Path: sp, Anchor: anchor, Digest: dig})
		}
		disp, err := reqStr(ro, "disposition", where+".disposition")
		if err != nil {
			return Plan{}, err
		}
		switch disp {
		case protocol.DispositionTest, protocol.DispositionNotApplicable, protocol.DispositionUnverified:
		default:
			return Plan{}, fmt.Errorf("INVALID_SCHEMA: %s.disposition: invalid disposition %q (test | not-applicable | unverified)", where, disp)
		}
		reason, err := reqStr(ro, "reason", where+".reason")
		if err != nil {
			return Plan{}, err
		}
		if disp == protocol.DispositionNotApplicable && reason == "" {
			return Plan{}, fmt.Errorf("INVALID_SCHEMA: %s: disposition not-applicable requires a nonempty rationale", where)
		}
		revObj, err := asObject(ro.Get2("review"), where+".review")
		if err != nil {
			return Plan{}, err
		}
		review, err := decodeReview(revObj, where+".review")
		if err != nil {
			return Plan{}, err
		}
		tests, err := decodeTestRefs(ro, "tests", where)
		if err != nil {
			return Plan{}, err
		}
		if disp == protocol.DispositionTest && len(tests) == 0 {
			return Plan{}, fmt.Errorf("MISSING_CONTRACT: %s: disposition test requires mapped tests", where)
		}
		p.RuntimeObligations = append(p.RuntimeObligations, RuntimeObligation{
			ID: id, Category: cat, Owner: owner, SourceRefs: srs,
			Disposition: disp, Reason: reason, Review: review, Tests: tests,
		})
	}
	// reviewed source anchors/digests bind to the held design bytes
	for i, ro := range p.RuntimeObligations {
		for j, sr := range ro.SourceRefs {
			sp := filepath.Join(designDir, filepath.FromSlash(sr.Path))
			data, err := readPayloadFile(sp)
			if err != nil {
				return Plan{}, fmt.Errorf("MISSING_CONTRACT: plan.json.runtime_obligations[%d].source_refs[%d]: %w", i, j, err)
			}
			sum := "sha256:" + hex.EncodeToString(sha256Sum(data))
			if sum != sr.Digest {
				return Plan{}, fmt.Errorf("STALE_INPUT: runtime obligation %s cites source %s with digest %s but the held design bytes hash to %s", ro.ID, sr.Path, sr.Digest, sum)
			}
			if sr.Anchor != "" && !bytesContain(data, sr.Anchor) {
				return Plan{}, fmt.Errorf("INVALID_SCHEMA: runtime obligation %s anchor %q is not present in cited source %s", ro.ID, sr.Anchor, sr.Path)
			}
		}
	}
	// every review on the control binds the entire projection and payload
	want, err := ReviewSubjectDigest(ReviewProjectionPlan(p), payloadDigest)
	if err != nil {
		return Plan{}, err
	}
	for i, ro := range p.RuntimeObligations {
		if ro.Review.SubjectDigest != want {
			return Plan{}, fmt.Errorf("STALE_INPUT: plan.json.runtime_obligations[%d].review.subject_digest does not match the projection of this control over the held design payload", i)
		}
	}
	return p, nil
}

func protocolCategory(c string) bool {
	for _, x := range protocol.RuntimeCategories {
		if x == c {
			return true
		}
	}
	return false
}

func sha256Sum(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[:]
}

func bytesContain(data []byte, needle string) bool {
	return len(needle) == 0 || strings.Contains(string(data), needle)
}

// readPayloadFile reads a design payload file for digest/anchor binding.
func readPayloadFile(p string) ([]byte, error) {
	fi, err := os.Lstat(p)
	if err != nil {
		return nil, fmt.Errorf("cited source %s does not exist under the design root", p)
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("cited source %s must be a regular non-symlink file", p)
	}
	if fi.Size() > protocol.ControlInputMaxBytes {
		return nil, fmt.Errorf("cited source %s exceeds the control-input cap", p)
	}
	return os.ReadFile(p)
}

// ValidateControlNamespace enforces the exact control namespace:
// assurance/ may hold only the authored plan.json and milestones/ of
// <M id>.json. Gates and hooks use it for cheap fail-closed discovery.
func ValidateControlNamespace(designDir string) error {
	dir := filepath.Join(designDir, protocol.ControlDirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("MISSING_CONTRACT: %s: no assurance/plan.json design declaration (%w)", designDir, err)
	}
	for _, e := range entries {
		if e.Name() == protocol.PlanFileName {
			if !e.Type().IsRegular() {
				return fmt.Errorf("INVALID_SCHEMA: control namespace assurance/%s must be a regular file", e.Name())
			}
			continue
		}
		if e.Name() == protocol.MilestonesDirName {
			if !e.IsDir() {
				return fmt.Errorf("INVALID_SCHEMA: control namespace assurance/%s must be a directory", e.Name())
			}
			ms, merr := os.ReadDir(filepath.Join(dir, e.Name()))
			if merr != nil {
				return fmt.Errorf("INVALID_SCHEMA: cannot enumerate assurance/milestones/: %w", merr)
			}
			for _, m := range ms {
				if !m.Type().IsRegular() || !milestoneRe.MatchString(strings.TrimSuffix(m.Name(), ".json")) || !strings.HasSuffix(m.Name(), ".json") {
					return fmt.Errorf("INVALID_SCHEMA: control namespace assurance/milestones/ holds unknown entry %q; only <M id>.json manifests are authored controls", m.Name())
				}
			}
			continue
		}
		return fmt.Errorf("INVALID_SCHEMA: control namespace assurance/ holds unknown entry %q (only plan.json and milestones/ are authored controls)", e.Name())
	}
	return nil
}

// LoadPlan loads and validates design/assurance/plan.json over the held
// immutable design snapshot directory: closed decoding, reviewed source
// anchors/digests against the current bytes, and review projection binding.
func LoadPlan(design string) (Plan, error) {
	fi, err := os.Stat(design)
	if err != nil || !fi.IsDir() {
		return Plan{}, fmt.Errorf("MISSING_CONTRACT: design root %s is not a readable directory", design)
	}
	if err := ValidateControlNamespace(design); err != nil {
		return Plan{}, err
	}
	raw, doc, err := readControlJSON(filepath.Join(design, protocol.ControlDirName, protocol.PlanFileName))
	if err != nil {
		return Plan{}, err
	}
	payload, err := DesignPayloadDigest(design)
	if err != nil {
		return Plan{}, err
	}
	return decodePlan(doc, design, payload, raw)
}

// ---- manifest decoding ----

var nativeUnionKeys = map[string]bool{
	"package": true, "test": true, "source": true, "path": true, "line": true,
	"column": true, "module": true, "class": true, "method": true, "name": true, "file": true,
}

func decodeNative(o *ir.Object, adapter, where string) (NativeID, error) {
	for _, k := range o.Keys() {
		if !nativeUnionKeys[k] {
			return NativeID{}, fmt.Errorf("INVALID_SCHEMA: %s: unknown native identity key %q; no arbitrary native-identity keys are accepted", where, k)
		}
	}
	var n NativeID
	getStr := func(k string) (string, error) {
		v := o.Get2(k)
		if v == nil {
			return "", nil
		}
		if v.IsNull() {
			return "", nil
		}
		if v.Kind != ir.KindString {
			return "", fmt.Errorf("INVALID_SCHEMA: %s.%s must be a string", where, k)
		}
		return v.AsString(), nil
	}
	getInt := func(k string) (int64, error) {
		v := o.Get2(k)
		if v == nil {
			return 0, nil
		}
		if v.IsNull() {
			return 0, nil
		}
		n, err := reqInt(o, k, where)
		return n, err
	}
	var err error
	if n.Package, err = getStr("package"); err != nil {
		return NativeID{}, err
	}
	if n.Test, err = getStr("test"); err != nil {
		return NativeID{}, err
	}
	if n.Source, err = getStr("source"); err != nil {
		return NativeID{}, err
	}
	if n.Module, err = getStr("module"); err != nil {
		return NativeID{}, err
	}
	if n.Class, err = getStr("class"); err != nil {
		return NativeID{}, err
	}
	if n.Method, err = getStr("method"); err != nil {
		return NativeID{}, err
	}
	if n.Name, err = getStr("name"); err != nil {
		return NativeID{}, err
	}
	if n.File, err = getStr("file"); err != nil {
		return NativeID{}, err
	}
	if n.Line, err = getInt("line"); err != nil {
		return NativeID{}, err
	}
	if n.Column, err = getInt("column"); err != nil {
		return NativeID{}, err
	}
	if pv := o.Get2("path"); pv != nil && !pv.IsNull() {
		if pv.Kind != ir.KindArray {
			return NativeID{}, fmt.Errorf("INVALID_SCHEMA: %s.path must be an array of strings", where)
		}
		for _, seg := range pv.AsArray() {
			if seg == nil || seg.Kind != ir.KindString {
				return NativeID{}, fmt.Errorf("INVALID_SCHEMA: %s.path must be an array of strings", where)
			}
			n.Path = append(n.Path, seg.AsString())
		}
	}
	set := func(bs ...bool) int {
		cnt := 0
		for _, b := range bs {
			if b {
				cnt++
			}
		}
		return cnt
	}
	check := func(ok bool, want string) error {
		if !ok {
			return fmt.Errorf("INVALID_SCHEMA: %s: native identity must set exactly %s for adapter %s", where, want, adapter)
		}
		return nil
	}
	switch adapter {
	case protocol.AdapterGoTesting:
		if err := check(n.Package != "" && n.Test != "" && set(n.Source != "", len(n.Path) > 0, n.Line != 0, n.Column != 0, n.Module != "", n.Class != "", n.Method != "", n.Name != "", n.File != "") == 0, "{package, test}"); err != nil {
			return NativeID{}, err
		}
	case protocol.AdapterNodeTestTS:
		if err := check(n.Source != "" && len(n.Path) > 0 && n.Line > 0 && n.Column > 0 && set(n.Package != "", n.Test != "", n.Module != "", n.Class != "", n.Method != "", n.Name != "", n.File != "") == 0, "{source, path, line, column}"); err != nil {
			return NativeID{}, err
		}
	case protocol.AdapterPythonUnittest:
		if err := check(n.Module != "" && n.Method != "" && set(n.Package != "", n.Test != "", n.Source != "", len(n.Path) > 0, n.Line != 0, n.Column != 0, n.Class != "", n.Name != "", n.File != "") == 0, "{module, [class], method}"); err != nil {
			return NativeID{}, err
		}
	case protocol.AdapterElixirExunit:
		if err := check(n.Module != "" && n.Name != "" && n.File != "" && n.Line > 0 && set(n.Package != "", n.Test != "", n.Source != "", len(n.Path) > 0, n.Column != 0, n.Class != "", n.Method != "") == 0, "{module, name, file, line}"); err != nil {
			return NativeID{}, err
		}
	default:
		return NativeID{}, fmt.Errorf("UNSUPPORTED_ADAPTER: %s", adapter)
	}
	return n, nil
}

func decodeExpectation(o *ir.Object, where string) (Expectation, error) {
	if err := exactKeys(o, where, "test", "outcome", "assertions"); err != nil {
		return Expectation{}, err
	}
	tr, err := decodeTestRefObj(o, where)
	if err != nil {
		return Expectation{}, err
	}
	outcome, err := reqStr(o, "outcome", where+".outcome")
	if err != nil {
		return Expectation{}, err
	}
	switch outcome {
	case protocol.OutcomePass, protocol.OutcomeAssertionFail:
	default:
		return Expectation{}, fmt.Errorf("INVALID_SCHEMA: %s.outcome: invalid outcome %q (pass | assertion-fail)", where, outcome)
	}
	assertions, err := reqStrArray(o, "assertions", where+".assertions")
	if err != nil {
		return Expectation{}, err
	}
	if outcome == protocol.OutcomeAssertionFail && len(assertions) == 0 {
		return Expectation{}, fmt.Errorf("INVALID_SCHEMA: %s: assertion-fail expectations must name the exact nonempty expected failing assertion set", where)
	}
	for _, a := range assertions {
		if !validID(a) {
			return Expectation{}, fmt.Errorf("INVALID_SCHEMA: %s.assertions: %q is not an ID", where, a)
		}
	}
	return Expectation{Test: tr, Outcome: outcome, Assertions: assertions}, nil
}

func decodeTestRefObj(o *ir.Object, where string) (TestRef, error) {
	tro, err := asObject(o.Get2("test"), where+".test")
	if err != nil {
		return TestRef{}, err
	}
	return decodeTestRef(tro, where+".test")
}

// decodeManifest performs full single-manifest validation over the parsed
// document; cross-file reconciliation belongs to Validate.
func decodeManifest(doc *ir.Value, msPath, designID, designDir, payloadDigest string, raw []byte) (Manifest, error) {
	root, err := asObject(doc, msPath)
	if err != nil {
		return Manifest{}, err
	}
	if err := exactKeys(root, msPath, "schema", "id", "revision", "predecessor", "repository",
		"implementation_roots", "frozen_roots", "subject_entries", "suites", "obligations",
		"baseline", "variants", "red_expectations", "checks", "red_controls", "limits", "review"); err != nil {
		return Manifest{}, err
	}
	schema, _ := reqStr(root, "schema", msPath+".schema")
	if schema != protocol.SchemaMilestone {
		return Manifest{}, fmt.Errorf("UNSUPPORTED_VERSION: milestone schema %q is not %s", schema, protocol.SchemaMilestone)
	}
	m := Manifest{Schema: schema, raw: raw, payloadDigest: payloadDigest, designID: designID, designDir: designDir}
	rel := filepath.ToSlash(filepath.Join(protocol.ControlDirName, protocol.MilestonesDirName, filepath.Base(msPath)))
	if designID != protocol.RepositoryRoot {
		rel = designID + "/" + rel
	}
	m.relPath = rel
	id, err := reqStr(root, "id", msPath+".id")
	if err != nil {
		return Manifest{}, err
	}
	if !validMilestoneID(id) {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.id: milestone id %q is not canonical (M plus an unsigned decimal, no leading zeroes except M0)", msPath, id)
	}
	if base := filepath.Base(msPath); base != id+".json" {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: manifest filename %q must be %q for milestone id %q; filenames cannot inherit the broader ID grammar", base, id+".json", id)
	}
	m.ID = id
	rev, err := reqInt(root, "revision", msPath+".revision")
	if err != nil {
		return Manifest{}, err
	}
	if rev <= 0 {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.revision must be a positive integer", msPath)
	}
	m.Revision = rev
	pred, err := optStr(root, "predecessor", msPath+".predecessor")
	if err != nil {
		return Manifest{}, err
	}
	if pred != "" && !validDigest(pred) {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.predecessor %q is not a digest", msPath, pred)
	}
	m.Predecessor = pred
	repo, err := reqStr(root, "repository", msPath+".repository")
	if err != nil {
		return Manifest{}, err
	}
	if repo != protocol.RepositoryRoot {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.repository must be exactly \".\" (the explicitly supplied repository root)", msPath)
	}
	m.Repository = repo
	decodeRootList := func(key string) ([]string, error) {
		vals, err := reqStrArray(root, key, msPath+"."+key)
		if err != nil {
			return nil, err
		}
		for i, v := range vals {
			if err := validateRootPath(v); err != nil {
				return nil, fmt.Errorf("INVALID_SCHEMA: %s.%s[%d]: %w", msPath, key, i, err)
			}
		}
		if err := rejectDuplicates(vals, msPath+"."+key); err != nil {
			return nil, err
		}
		if err := rejectCaseAliases(vals, msPath+"."+key); err != nil {
			return nil, err
		}
		return vals, nil
	}
	if m.ImplementationRoots, err = decodeRootList("implementation_roots"); err != nil {
		return Manifest{}, err
	}
	if m.FrozenRoots, err = decodeRootList("frozen_roots"); err != nil {
		return Manifest{}, err
	}
	seObjs, err := reqObjArray(root, "subject_entries", msPath+".subject_entries")
	if err != nil {
		return Manifest{}, err
	}
	var subjectPaths []string
	for i, so := range seObjs {
		where := fmt.Sprintf("%s.subject_entries[%d]", msPath, i)
		if err := exactKeys(so, where, "path", "kind"); err != nil {
			return Manifest{}, err
		}
		sp, err := reqStr(so, "path", where+".path")
		if err != nil {
			return Manifest{}, err
		}
		if err := validateRootPath(sp); err != nil {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.path: %w", where, err)
		}
		kind, err := reqStr(so, "kind", where+".kind")
		if err != nil {
			return Manifest{}, err
		}
		if kind != "file" && kind != "directory" {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.kind: invalid kind %q (file | directory)", where, kind)
		}
		if sp == protocol.RepositoryRoot && kind != "directory" {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s: subject entry \".\" is only valid with kind directory", where)
		}
		m.SubjectEntries = append(m.SubjectEntries, SubjectEntry{Path: sp, Kind: kind})
		subjectPaths = append(subjectPaths, sp)
	}
	if err := rejectDuplicates(subjectPaths, msPath+".subject_entries"); err != nil {
		return Manifest{}, err
	}
	if err := rejectCaseAliases(subjectPaths, msPath+".subject_entries"); err != nil {
		return Manifest{}, err
	}
	suObjs, err := reqObjArray(root, "suites", msPath+".suites")
	if err != nil {
		return Manifest{}, err
	}
	seenSuite := map[string]bool{}
	for i, so := range suObjs {
		where := fmt.Sprintf("%s.suites[%d]", msPath, i)
		if err := exactKeys(so, where, "id", "adapter", "runtime", "root", "files", "tests", "environment", "dependency_roots"); err != nil {
			return Manifest{}, err
		}
		var s Suite
		s.ID, err = reqStr(so, "id", where+".id")
		if err != nil {
			return Manifest{}, err
		}
		if !validID(s.ID) {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.id %q is not an ID", where, s.ID)
		}
		if seenSuite[s.ID] {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s: duplicate suite id %q; suites are unique within a milestone", where, s.ID)
		}
		seenSuite[s.ID] = true
		s.Adapter, err = reqStr(so, "adapter", where+".adapter")
		if err != nil {
			return Manifest{}, err
		}
		if !protocol.SupportedAdapters[s.Adapter] {
			return Manifest{}, fmt.Errorf("UNSUPPORTED_ADAPTER: %s.adapter %q is not one of the first-release closed adapters %s", where, s.Adapter, strings.Join([]string{protocol.AdapterGoTesting, protocol.AdapterNodeTestTS, protocol.AdapterPythonUnittest, protocol.AdapterElixirExunit}, ", "))
		}
		rtObj, err := asObject(so.Get2("runtime"), where+".runtime")
		if err != nil {
			return Manifest{}, err
		}
		if err := exactKeys(rtObj, where+".runtime", "profile", "version", "platform", "closure"); err != nil {
			return Manifest{}, err
		}
		s.Runtime.Profile, err = reqStr(rtObj, "profile", where+".runtime.profile")
		if err != nil {
			return Manifest{}, err
		}
		if !validID(s.Runtime.Profile) {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.runtime.profile %q is not an ID", where, s.Runtime.Profile)
		}
		s.Runtime.Version, err = reqStr(rtObj, "version", where+".runtime.version")
		if err != nil {
			return Manifest{}, err
		}
		if s.Runtime.Version == "" {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.runtime.version must be nonempty", where)
		}
		s.Runtime.Platform, err = reqStr(rtObj, "platform", where+".runtime.platform")
		if err != nil {
			return Manifest{}, err
		}
		if !protocol.SupportedPlatforms[s.Runtime.Platform] {
			return Manifest{}, fmt.Errorf("UNSUPPORTED_PLATFORM: %s.runtime.platform %q is not one of the supported native platforms darwin/arm64, linux/amd64", where, s.Runtime.Platform)
		}
		s.Runtime.Closure, err = reqStr(rtObj, "closure", where+".runtime.closure")
		if err != nil {
			return Manifest{}, err
		}
		if !validDigest(s.Runtime.Closure) {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.runtime.closure %q is not a digest", where, s.Runtime.Closure)
		}
		s.Root, err = reqStr(so, "root", where+".root")
		if err != nil {
			return Manifest{}, err
		}
		if err := validateRootPath(s.Root); err != nil {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.root: %w", where, err)
		}
		s.Files, err = reqStrArray(so, "files", where+".files")
		if err != nil {
			return Manifest{}, err
		}
		for j, f := range s.Files {
			if err := validatePath(f); err != nil {
				return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.files[%d]: %w", where, j, err)
			}
		}
		if err := rejectDuplicates(s.Files, where+".files"); err != nil {
			return Manifest{}, err
		}
		if err := rejectCaseAliases(s.Files, where+".files"); err != nil {
			return Manifest{}, err
		}
		deps, err := reqStrArray(so, "dependency_roots", where+".dependency_roots")
		if err != nil {
			return Manifest{}, err
		}
		for j, d := range deps {
			if err := validateRootPath(d); err != nil {
				return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.dependency_roots[%d]: %w", where, j, err)
			}
		}
		if err := rejectDuplicates(deps, where+".dependency_roots"); err != nil {
			return Manifest{}, err
		}
		s.DependencyRoots = deps
		envObjs, err := reqObjArray(so, "environment", where+".environment")
		if err != nil {
			return Manifest{}, err
		}
		seenEnv := map[string]bool{}
		for j, eo := range envObjs {
			ewhere := fmt.Sprintf("%s.environment[%d]", where, j)
			if err := exactKeys(eo, ewhere, "name", "value"); err != nil {
				return Manifest{}, err
			}
			name, err := reqStr(eo, "name", ewhere+".name")
			if err != nil {
				return Manifest{}, err
			}
			value, err := reqStr(eo, "value", ewhere+".value")
			if err != nil {
				return Manifest{}, err
			}
			if name == "" || strings.ContainsAny(name, "=\x00") {
				return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.name %q is not a portable environment variable name", ewhere, name)
			}
			if seenEnv[name] {
				return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s: duplicate environment entry %q", ewhere, name)
			}
			seenEnv[name] = true
			s.Environment = append(s.Environment, EnvironmentVar{Name: name, Value: value})
		}
		ttObjs, err := reqObjArray(so, "tests", where+".tests")
		if err != nil {
			return Manifest{}, err
		}
		if len(ttObjs) == 0 {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s must declare at least one test; a suite with no tests is missing suite controls", where)
		}
		seenTest := map[string]bool{}
		for j, to := range ttObjs {
			twhere := fmt.Sprintf("%s.tests[%d]", where, j)
			if err := exactKeys(to, twhere, "id", "native", "source", "role", "assertions"); err != nil {
				return Manifest{}, err
			}
			var tst Test
			tst.ID, err = reqStr(to, "id", twhere+".id")
			if err != nil {
				return Manifest{}, err
			}
			if !validID(tst.ID) {
				return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.id %q is not an ID", twhere, tst.ID)
			}
			if seenTest[tst.ID] {
				return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s: duplicate test id %q in suite %s; tests are unique within a suite", twhere, tst.ID, s.ID)
			}
			seenTest[tst.ID] = true
			noObj, err := asObject(to.Get2("native"), twhere+".native")
			if err != nil {
				return Manifest{}, err
			}
			if tst.Native, err = decodeNative(noObj, s.Adapter, twhere+".native"); err != nil {
				return Manifest{}, err
			}
			tst.Source, err = reqStr(to, "source", twhere+".source")
			if err != nil {
				return Manifest{}, err
			}
			if err := validatePath(tst.Source); err != nil {
				return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.source: %w", twhere, err)
			}
			tst.Role, err = reqStr(to, "role", twhere+".role")
			if err != nil {
				return Manifest{}, err
			}
			switch tst.Role {
			case protocol.RolePositive, protocol.RoleNegative, protocol.RoleControl, protocol.RoleRegression:
			default:
				return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.role: invalid role %q (positive | negative | control | regression)", twhere, tst.Role)
			}
			asObjs, err := reqObjArray(to, "assertions", twhere+".assertions")
			if err != nil {
				return Manifest{}, err
			}
			if len(asObjs) == 0 {
				return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s must declare at least one assertion; every leaf binds its assertions and aggregate parents are not substitute leaf coverage", twhere)
			}
			seenA := map[string]bool{}
			for k, ao := range asObjs {
				awhere := fmt.Sprintf("%s.assertions[%d]", twhere, k)
				if err := exactKeys(ao, awhere, "id", "source", "line", "helper"); err != nil {
					return Manifest{}, err
				}
				var a Assertion
				a.ID, err = reqStr(ao, "id", awhere+".id")
				if err != nil {
					return Manifest{}, err
				}
				if !validID(a.ID) {
					return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.id %q is not an ID", awhere, a.ID)
				}
				if seenA[a.ID] {
					return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s: duplicate assertion id %q within test %s", awhere, a.ID, tst.ID)
				}
				seenA[a.ID] = true
				a.Source, err = reqStr(ao, "source", awhere+".source")
				if err != nil {
					return Manifest{}, err
				}
				if err := validatePath(a.Source); err != nil {
					return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.source: %w", awhere, err)
				}
				a.Line, err = reqInt(ao, "line", awhere+".line")
				if err != nil {
					return Manifest{}, err
				}
				if a.Line <= 0 {
					return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.line must be a positive integer", awhere)
				}
				a.Helper, err = reqStr(ao, "helper", awhere+".helper")
				if err != nil {
					return Manifest{}, err
				}
				if a.Helper != protocol.AssertionHelperV1 {
					return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.helper must be %q", awhere, protocol.AssertionHelperV1)
				}
				tst.Assertions = append(tst.Assertions, a)
			}
			s.Tests = append(s.Tests, tst)
		}
		m.Suites = append(m.Suites, s)
	}
	obObjs, err := reqObjArray(root, "obligations", msPath+".obligations")
	if err != nil {
		return Manifest{}, err
	}
	seenKey := map[string]bool{}
	for i, oo := range obObjs {
		where := fmt.Sprintf("%s.obligations[%d]", msPath, i)
		if err := exactKeys(oo, where, "key", "positive", "negative"); err != nil {
			return Manifest{}, err
		}
		ko, err := asObject(oo.Get2("key"), where+".key")
		if err != nil {
			return Manifest{}, err
		}
		if err := exactKeys(ko, where+".key", "design", "kind", "owner", "id"); err != nil {
			return Manifest{}, err
		}
		var key ObligationKey
		key.Design, err = reqStr(ko, "design", where+".key.design")
		if err != nil {
			return Manifest{}, err
		}
		if err := validateRootPath(key.Design); err != nil {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.key.design: %w", where, err)
		}
		key.Kind, err = reqStr(ko, "kind", where+".key.kind")
		if err != nil {
			return Manifest{}, err
		}
		switch key.Kind {
		case protocol.KindOracleRow, protocol.KindGuardClause, protocol.KindInvariant, protocol.KindRuntime:
		default:
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.key.kind: invalid kind %q (oracle-row | guard-clause | invariant | runtime)", where, key.Kind)
		}
		key.Owner, err = reqStr(ko, "owner", where+".key.owner")
		if err != nil {
			return Manifest{}, err
		}
		if key.Owner == "" {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.key.owner must be nonempty", where)
		}
		key.ID, err = reqStr(ko, "id", where+".key.id")
		if err != nil {
			return Manifest{}, err
		}
		if !validID(key.ID) {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.key.id %q is not an ID", where, key.ID)
		}
		kstr := key.Design + "\x00" + key.Kind + "\x00" + key.Owner + "\x00" + key.ID
		if seenKey[kstr] {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.key: duplicate obligation key {%s %s %s %s}", where, key.Design, key.Kind, key.Owner, key.ID)
		}
		seenKey[kstr] = true
		positive, err := decodeTestRefs(oo, "positive", where)
		if err != nil {
			return Manifest{}, err
		}
		negative, err := decodeTestRefs(oo, "negative", where)
		if err != nil {
			return Manifest{}, err
		}
		m.Obligations = append(m.Obligations, Obligation{Key: key, Positive: positive, Negative: negative})
	}
	baseline, err := optStr(root, "baseline", msPath+".baseline")
	if err != nil {
		return Manifest{}, err
	}
	if baseline != "" && !validDigest(baseline) {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.baseline %q is not a content-addressed Ref digest", msPath, baseline)
	}
	m.Baseline = baseline
	vaObjs, err := reqObjArray(root, "variants", msPath+".variants")
	if err != nil {
		return Manifest{}, err
	}
	seenVariant := map[string]bool{}
	for i, vo := range vaObjs {
		where := fmt.Sprintf("%s.variants[%d]", msPath, i)
		if err := exactKeys(vo, where, "id", "kind", "source", "pair", "target_tests", "expected", "review"); err != nil {
			return Manifest{}, err
		}
		var v Variant
		v.ID, err = reqStr(vo, "id", where+".id")
		if err != nil {
			return Manifest{}, err
		}
		if !validID(v.ID) {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.id %q is not an ID", where, v.ID)
		}
		if seenVariant[v.ID] {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s: duplicate variant id %q", where, v.ID)
		}
		seenVariant[v.ID] = true
		v.Kind, err = reqStr(vo, "kind", where+".kind")
		if err != nil {
			return Manifest{}, err
		}
		switch v.Kind {
		case protocol.VariantSafeControl, protocol.VariantUnsafeChallenge:
		default:
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.kind: invalid kind %q (safe-control | unsafe-challenge)", where, v.Kind)
		}
		v.Source, err = optStr(vo, "source", where+".source")
		if err != nil {
			return Manifest{}, err
		}
		if v.Source != "" && !validDigest(v.Source) {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.source %q is not a content-addressed Ref digest", where, v.Source)
		}
		v.Pair, err = reqStr(vo, "pair", where+".pair")
		if err != nil {
			return Manifest{}, err
		}
		if v.Pair != "" && !validID(v.Pair) {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.pair %q is not an ID", where, v.Pair)
		}
		if v.TargetTests, err = decodeTestRefs(vo, "target_tests", where); err != nil {
			return Manifest{}, err
		}
		exObjs, err := reqObjArray(vo, "expected", where+".expected")
		if err != nil {
			return Manifest{}, err
		}
		for j, eo := range exObjs {
			exp, err := decodeExpectation(eo, fmt.Sprintf("%s.expected[%d]", where, j))
			if err != nil {
				return Manifest{}, err
			}
			v.Expected = append(v.Expected, exp)
		}
		revObj, err := asObject(vo.Get2("review"), where+".review")
		if err != nil {
			return Manifest{}, err
		}
		if v.Review, err = decodeReview(revObj, where+".review"); err != nil {
			return Manifest{}, err
		}
		m.Variants = append(m.Variants, v)
	}
	reObjs, err := reqObjArray(root, "red_expectations", msPath+".red_expectations")
	if err != nil {
		return Manifest{}, err
	}
	for i, eo := range reObjs {
		exp, err := decodeExpectation(eo, fmt.Sprintf("%s.red_expectations[%d]", msPath, i))
		if err != nil {
			return Manifest{}, err
		}
		m.RedExpectations = append(m.RedExpectations, exp)
	}
	ckObjs, err := reqObjArray(root, "checks", msPath+".checks")
	if err != nil {
		return Manifest{}, err
	}
	seenCheck := map[string]bool{}
	for i, co := range ckObjs {
		where := fmt.Sprintf("%s.checks[%d]", msPath, i)
		if err := exactKeys(co, where, "id", "kind", "profile", "inputs"); err != nil {
			return Manifest{}, err
		}
		var c Check
		c.ID, err = reqStr(co, "id", where+".id")
		if err != nil {
			return Manifest{}, err
		}
		if !validID(c.ID) {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.id %q is not an ID", where, c.ID)
		}
		if seenCheck[c.ID] {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s: duplicate check id %q", where, c.ID)
		}
		seenCheck[c.ID] = true
		c.Kind, err = reqStr(co, "kind", where+".kind")
		if err != nil {
			return Manifest{}, err
		}
		c.Profile, err = reqStr(co, "profile", where+".profile")
		if err != nil {
			return Manifest{}, err
		}
		wantKind, known := protocol.CheckProfiles[c.Profile]
		if !known {
			return Manifest{}, fmt.Errorf("UNSUPPORTED_FEATURE: %s.profile: unknown check profile %q; a user-written shell command saying exit 0 is not a checked precondition", where, c.Profile)
		}
		switch c.Kind {
		case protocol.CheckKindDesign, protocol.CheckKindArchitecture, protocol.CheckKindTypecheck, protocol.CheckKindFormat, protocol.CheckKindLint:
		default:
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.kind: invalid kind %q (design | architecture | typecheck | format | lint)", where, c.Kind)
		}
		if c.Kind != wantKind {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s: check kind %q does not match the closed profile %s which requires kind %s", where, c.Kind, c.Profile, wantKind)
		}
		c.Inputs, err = reqStrArray(co, "inputs", where+".inputs")
		if err != nil {
			return Manifest{}, err
		}
		for j, in := range c.Inputs {
			if err := validatePath(in); err != nil {
				return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.inputs[%d]: %w", where, j, err)
			}
		}
		m.Checks = append(m.Checks, c)
	}
	rcObjs, err := reqObjArray(root, "red_controls", msPath+".red_controls")
	if err != nil {
		return Manifest{}, err
	}
	for i, ro := range rcObjs {
		where := fmt.Sprintf("%s.red_controls[%d]", msPath, i)
		if err := exactKeys(ro, where, "test", "assertion", "safe_variant"); err != nil {
			return Manifest{}, err
		}
		tr, err := decodeTestRefObj(ro, where)
		if err != nil {
			return Manifest{}, err
		}
		assertion, err := reqStr(ro, "assertion", where+".assertion")
		if err != nil {
			return Manifest{}, err
		}
		if !validID(assertion) {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.assertion %q is not an ID", where, assertion)
		}
		sv, err := reqStr(ro, "safe_variant", where+".safe_variant")
		if err != nil {
			return Manifest{}, err
		}
		if !validID(sv) {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.safe_variant %q is not an ID", where, sv)
		}
		m.RedControls = append(m.RedControls, RedControl{Test: tr, Assertion: assertion, SafeVariant: sv})
	}
	limObj, err := asObject(root.Get2("limits"), msPath+".limits")
	if err != nil {
		return Manifest{}, err
	}
	if err := exactKeys(limObj, msPath+".limits", "wall_ms", "cleanup_ms", "stdout_bytes", "stderr_bytes", "event_bytes", "event_count", "jobs", "bundle_bytes", "entries", "depth"); err != nil {
		return Manifest{}, err
	}
	limitFields := []struct {
		key string
		dst *int64
		max int64
	}{
		{"wall_ms", &m.Limits.WallMS, protocol.LimitWallMaxMS},
		{"cleanup_ms", &m.Limits.CleanupMS, protocol.LimitCleanupMaxMS},
		{"stdout_bytes", &m.Limits.StdoutBytes, protocol.LimitStreamMaxBytes},
		{"stderr_bytes", &m.Limits.StderrBytes, protocol.LimitStreamMaxBytes},
		{"event_bytes", &m.Limits.EventBytes, protocol.LimitStreamMaxBytes},
		{"event_count", &m.Limits.EventCount, protocol.LimitEventMaxCount},
		{"jobs", &m.Limits.Jobs, protocol.LimitJobsMax},
		{"bundle_bytes", &m.Limits.BundleBytes, protocol.LimitBundleMaxBytes},
		{"entries", &m.Limits.Entries, protocol.LimitEntriesMax},
		{"depth", &m.Limits.Depth, protocol.LimitDepthMax},
	}
	for _, lf := range limitFields {
		n, err := reqInt(limObj, lf.key, msPath+".limits."+lf.key)
		if err != nil {
			return Manifest{}, err
		}
		if n <= 0 {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.limits.%s must be a positive integer", msPath, lf.key)
		}
		if n > lf.max {
			return Manifest{}, fmt.Errorf("INVALID_SCHEMA: %s.limits.%s %d exceeds the shipped absolute cap %d", msPath, lf.key, n, lf.max)
		}
		*lf.dst = n
	}
	revObj, err := asObject(root.Get2("review"), msPath+".review")
	if err != nil {
		return Manifest{}, err
	}
	if m.Review, err = decodeReview(revObj, msPath+".review"); err != nil {
		return Manifest{}, err
	}
	if err := validateManifestTopology(&m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// validateManifestTopology enforces the single-manifest source-topology
// rules: subject/frozen separation, suite containment, the reserved control
// namespace, and (when the design root is the repository root) existence of
// every referenced path.
func validateManifestTopology(m *Manifest) error {
	underSubject := func(p string) bool {
		for _, se := range m.SubjectEntries {
			if se.Kind != "directory" {
				if p == se.Path {
					return true
				}
				continue
			}
			if p == se.Path || strings.HasPrefix(p, se.Path+"/") {
				return true
			}
		}
		return false
	}
	var frozenish []string
	for _, s := range m.Suites {
		frozenish = append(frozenish, s.Root)
		frozenish = append(frozenish, s.Files...)
		frozenish = append(frozenish, s.DependencyRoots...)
		for _, t := range s.Tests {
			frozenish = append(frozenish, t.Source)
			for _, a := range t.Assertions {
				frozenish = append(frozenish, a.Source)
			}
		}
	}
	frozenish = append(frozenish, m.FrozenRoots...)
	for _, f := range frozenish {
		if f == protocol.RepositoryRoot {
			continue
		}
		if underSubject(f) {
			return fmt.Errorf("INVALID_SCHEMA: subject entry covers frozen/test input %q; a mutable subject must not contain or be an ancestor of frozen entries", f)
		}
	}
	for _, s := range m.Suites {
		contains := func(p string) bool {
			if s.Root == protocol.RepositoryRoot {
				return true
			}
			if p == s.Root || strings.HasPrefix(p, s.Root+"/") {
				return true
			}
			for _, d := range s.DependencyRoots {
				if d == protocol.RepositoryRoot || p == d || strings.HasPrefix(p, d+"/") {
					return true
				}
			}
			return false
		}
		for _, f := range s.Files {
			if !contains(f) {
				return fmt.Errorf("INVALID_SCHEMA: suite %s file %q must live under the suite root or a declared dependency root", s.ID, f)
			}
		}
		for _, t := range s.Tests {
			if !contains(t.Source) {
				return fmt.Errorf("INVALID_SCHEMA: suite %s test %s source %q must live under the suite root or a declared dependency root", s.ID, t.ID, t.Source)
			}
		}
	}
	// no runtime source path may live in the reserved control namespace
	var all []string
	all = append(all, frozenish...)
	for _, c := range m.Checks {
		all = append(all, c.Inputs...)
	}
	for _, se := range m.SubjectEntries {
		all = append(all, se.Path)
	}
	for _, p := range all {
		if p == protocol.RepositoryRoot {
			continue
		}
		if rel, err := filepath.Rel(filepath.Join(m.designDir, protocol.ControlDirName), filepath.Join(m.designDir, filepath.FromSlash(p))); err == nil && !strings.HasPrefix(rel, "..") {
			return fmt.Errorf("INVALID_SCHEMA: path %q must not live in the reserved control namespace %s/", p, protocol.ControlDirName)
		}
	}
	// existence is decidable when the design root is the repository root
	if m.designID == protocol.RepositoryRoot {
		exists := func(p string) error {
			full := filepath.Join(m.designDir, filepath.FromSlash(p))
			fi, err := os.Lstat(full)
			if err != nil || fi.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("MISSING_CONTRACT: %q does not exist under the repository root (design \".\")", p)
			}
			return nil
		}
		checks := append(append([]string{}, m.ImplementationRoots...), m.FrozenRoots...)
		for _, s := range m.Suites {
			checks = append(checks, s.Files...)
			checks = append(checks, s.DependencyRoots...)
			for _, t := range s.Tests {
				checks = append(checks, t.Source)
				for _, a := range t.Assertions {
					checks = append(checks, a.Source)
				}
			}
		}
		for _, se := range m.SubjectEntries {
			checks = append(checks, se.Path)
		}
		for _, c := range m.Checks {
			checks = append(checks, c.Inputs...)
		}
		for _, p := range checks {
			if p == protocol.RepositoryRoot {
				continue
			}
			if err := exists(p); err != nil {
				return err
			}
		}
	}
	return nil
}

// adjacentDesignID resolves the design identity from the sibling plan.json
// (design/assurance/plan.json, three levels above the manifest file); a
// missing plan means the design root is the repository root itself. A
// present-but-corrupt plan is corruption, not a default.
func adjacentDesignID(msPath string) (string, string, error) {
	designDir := filepath.Dir(filepath.Dir(filepath.Dir(msPath)))
	planPath := filepath.Join(designDir, protocol.ControlDirName, protocol.PlanFileName)
	if _, err := os.Stat(planPath); err != nil {
		if os.IsNotExist(err) {
			return protocol.RepositoryRoot, designDir, nil
		}
		return "", "", fmt.Errorf("INVALID_SCHEMA: cannot inspect %s: %w", planPath, err)
	}
	_, doc, err := readControlJSON(planPath)
	if err != nil {
		return "", "", err
	}
	root, err := asObject(doc, "plan.json")
	if err != nil {
		return "", "", err
	}
	v := root.Get2("design")
	if v == nil || v.Kind != ir.KindString {
		return "", "", fmt.Errorf("INVALID_SCHEMA: adjacent plan.json carries no string design root")
	}
	d := v.AsString()
	if err := validateRootPath(d); err != nil {
		return "", "", fmt.Errorf("INVALID_SCHEMA: adjacent plan.json design: %w", err)
	}
	return d, designDir, nil
}

// LoadManifest loads and validates one milestone manifest file, binding its
// reviews to the projection over the held design payload.
func LoadManifest(path string) (Manifest, error) {
	if filepath.Base(filepath.Dir(path)) != protocol.MilestonesDirName {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: manifest %s must live in the %s/ control namespace", path, protocol.MilestonesDirName)
	}
	raw, doc, err := readControlJSON(path)
	if err != nil {
		return Manifest{}, err
	}
	designID, designDir, err := adjacentDesignID(path)
	if err != nil {
		return Manifest{}, err
	}
	payload, err := DesignPayloadDigest(designDir)
	if err != nil {
		return Manifest{}, err
	}
	m, err := decodeManifest(doc, path, designID, designDir, payload, raw)
	if err != nil {
		return Manifest{}, err
	}
	want, err := ReviewSubjectDigest(ReviewProjectionManifest(m), m.payloadDigest)
	if err != nil {
		return Manifest{}, err
	}
	if m.Review.SubjectDigest != want {
		return Manifest{}, fmt.Errorf("STALE_INPUT: %s: review.subject_digest does not match the projection of this control over the held design payload", path)
	}
	for i, v := range m.Variants {
		if v.Review.SubjectDigest != want {
			return Manifest{}, fmt.Errorf("STALE_INPUT: %s: variants[%d].review.subject_digest does not match the projection of this control", path, i)
		}
	}
	return m, nil
}

// ---- finalized cross-file reconciliation ----

type testKey struct{ design, milestone, suite, test string }

func (k testKey) String() string {
	return k.design + "/" + k.milestone + "/" + k.suite + "/" + k.test
}

func refKey(r TestRef) testKey {
	return testKey{r.Design, r.Milestone, r.Suite, r.Test}
}

// Validate reconciles a loaded plan, its loaded manifests and the
// authoritative inventory as FINALIZED declarations: qualified test
// resolution, plan/manifest set equality, milestone currency, obligation
// mapping without owner pooling, the seven runtime categories, expectation
// completeness, the red_control bijection and complete safe/unsafe pairs.
// Hand-built records are accepted; review digest binding was verified at
// load time by LoadPlan/LoadManifest.
func Validate(plan Plan, manifests []Manifest, inventory Inventory) error {
	var errs []string
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}

	// index every declared leaf test with its registered assertions
	type testInfo struct {
		manifest   *Manifest
		assertions map[string]bool
	}
	tests := map[testKey]*testInfo{}
	for mi := range manifests {
		m := &manifests[mi]
		for _, s := range m.Suites {
			for _, t := range s.Tests {
				k := testKey{m.designID, m.ID, s.ID, t.ID}
				if _, dup := tests[k]; dup {
					add("DUPLICATE_TEST: test %s is declared more than once in the provided manifests", k)
					continue
				}
				aset := map[string]bool{}
				for _, a := range t.Assertions {
					aset[a.ID] = true
				}
				tests[k] = &testInfo{manifest: m, assertions: aset}
			}
		}
	}
	validateRefs := func(r TestRef, where string) {
		k := refKey(r)
		if _, ok := tests[k]; !ok {
			add("MISSING_TEST: %s references undeclared test %s (dangling, ambiguous or cross-project references fail)", where, k)
		}
	}

	// resolve every declared reference first
	for _, ro := range plan.RuntimeObligations {
		for _, r := range ro.Tests {
			validateRefs(r, "plan runtime obligation "+ro.ID)
		}
	}
	for mi := range manifests {
		m := &manifests[mi]
		for _, o := range m.Obligations {
			for _, r := range o.Positive {
				validateRefs(r, "manifest "+m.ID+" obligation positive")
			}
			for _, r := range o.Negative {
				validateRefs(r, "manifest "+m.ID+" obligation negative")
			}
		}
		for _, e := range m.RedExpectations {
			validateRefs(e.Test, "manifest "+m.ID+" red_expectations")
		}
		for _, v := range m.Variants {
			for _, r := range v.TargetTests {
				validateRefs(r, "variant "+v.ID+" target_tests")
			}
			for _, e := range v.Expected {
				validateRefs(e.Test, "variant "+v.ID+" expected")
			}
		}
		for _, rc := range m.RedControls {
			validateRefs(rc.Test, "manifest "+m.ID+" red_controls")
		}
	}

	// plan <-> manifest set equality for the plan's own design
	declared := map[string]*Manifest{}
	for mi := range manifests {
		m := &manifests[mi]
		if m.designID != plan.Design {
			continue
		}
		if prev, dup := declared[m.ID]; dup {
			add("DUPLICATE_TEST: milestone %s of design %s has two manifests (%s, %s)", m.ID, plan.Design, prev.relPath, m.relPath)
			continue
		}
		declared[m.ID] = m
	}
	for _, pm := range plan.Milestones {
		m, ok := declared[pm.ID]
		if !ok {
			add("MISSING_CONTRACT: plan declares milestone %s but no loaded manifest for design %s was provided", pm.ID, plan.Design)
			continue
		}
		if m.relPath != "" && m.relPath != pm.Manifest {
			add("INVALID_SCHEMA: manifest for milestone %s was loaded from %s but the plan declares %s", pm.ID, m.relPath, pm.Manifest)
		}
	}
	for id := range declared {
		found := false
		for _, pm := range plan.Milestones {
			if pm.ID == id {
				found = true
			}
		}
		if !found {
			add("MISSING_CONTRACT: manifest %s (design %s) is not declared in the plan's milestones", id, plan.Design)
		}
	}

	// milestone currency: a manifest of the plan's design must be a current
	// root BUILD milestone of the authoritative inventory
	current := map[string]bool{}
	for _, mk := range inventory.Milestones {
		if mk.Design == plan.Design {
			current[mk.Milestone] = true
		}
	}
	for mi := range manifests {
		m := &manifests[mi]
		if m.designID != plan.Design {
			continue
		}
		if !current[m.ID] {
			add("MISSING_CONTRACT: milestone %s of design %s is not current in the authoritative inventory; deleting or renaming a BUILD milestone cannot reduce requirements", m.ID, plan.Design)
		}
	}

	// obligation reconciliation: manifests map only their own design's
	// obligations; every authoritative oracle/guard/invariant obligation is
	// mapped by at least one current manifest; no invented or pooled key
	mapped := map[string]bool{}
	for mi := range manifests {
		m := &manifests[mi]
		for _, o := range m.Obligations {
			if o.Key.Design != m.designID {
				add("INVALID_SCHEMA: manifest for design %s cannot map obligation {%s %s %s %s} of another design", m.designID, o.Key.Design, o.Key.Kind, o.Key.Owner, o.Key.ID)
				continue
			}
			mapped[o.Key.Design+"\x00"+o.Key.Kind+"\x00"+o.Key.Owner+"\x00"+o.Key.ID] = true
		}
	}
	invSet := map[string]bool{}
	for _, o := range inventory.Obligations {
		invSet[o.Key.Design+"\x00"+o.Key.Kind+"\x00"+o.Key.Owner+"\x00"+o.Key.ID] = true
		if o.Key.Kind == protocol.KindRuntime {
			continue
		}
		if !mapped[o.Key.Design+"\x00"+o.Key.Kind+"\x00"+o.Key.Owner+"\x00"+o.Key.ID] {
			add("MISSING_CONTRACT: unmapped obligation {design=%s kind=%s owner=%s id=%s}: no current milestone's manifest covers it", o.Key.Design, o.Key.Kind, o.Key.Owner, o.Key.ID)
		}
	}
	for mi := range manifests {
		m := &manifests[mi]
		if m.designID != plan.Design {
			continue
		}
		for _, o := range m.Obligations {
			if !invSet[o.Key.Design+"\x00"+o.Key.Kind+"\x00"+o.Key.Owner+"\x00"+o.Key.ID] {
				add("MISSING_CONTRACT: obligation {design=%s kind=%s owner=%s id=%s} is not in the authoritative inventory for design %s (invented, stale or pooled key)", o.Key.Design, o.Key.Kind, o.Key.Owner, o.Key.ID, plan.Design)
			}
		}
	}

	// the seven runtime/NFR categories: explicit dispositions, unverified
	// blocks strict completion, test dispositions carry mapped tests
	present := map[string]bool{}
	for _, ro := range plan.RuntimeObligations {
		present[ro.Category] = true
		switch ro.Disposition {
		case protocol.DispositionTest:
			if len(ro.Tests) == 0 {
				add("MISSING_CONTRACT: runtime obligation %s declares disposition test without mapped tests", ro.ID)
			}
		case protocol.DispositionNotApplicable:
			if ro.Reason == "" {
				add("MISSING_CONTRACT: runtime obligation %s is not-applicable without a nonempty rationale", ro.ID)
			}
		case protocol.DispositionUnverified:
			add("MISSING_CONTRACT: runtime obligation %s (%s) is unverified: unverified blocks strict completion", ro.ID, ro.Category)
		}
	}
	for _, cat := range protocol.RuntimeCategories {
		if !present[cat] {
			add("MISSING_CONTRACT: missing runtime category %s; a generic runtime row cannot hide a missing category", cat)
		}
	}

	// per-manifest finalized reconciliation
	for mi := range manifests {
		m := &manifests[mi]
		own := map[testKey]bool{}
		for _, s := range m.Suites {
			for _, t := range s.Tests {
				own[testKey{m.designID, m.ID, s.ID, t.ID}] = true
			}
		}
		if m.Baseline == "" {
			add("MISSING_CONTRACT: finalized milestone %s must declare its baseline reference (drafts never reach Validate)", m.ID)
		}
		// expectation completeness: exactly one baseline expectation per own
		// leaf; pass expectations name the exact asserted set, failures a
		// nonempty subset of the registered assertions
		checkExp := func(e Expectation, where string) {
			info, ok := tests[refKey(e.Test)]
			if !ok {
				return
			}
			switch e.Outcome {
			case protocol.OutcomePass:
				if len(e.Assertions) != len(info.assertions) {
					add("ASSERTION_MISMATCH: %s: pass expectation for %s must name the exact asserted set (%d registered, %d named)", where, refKey(e.Test), len(info.assertions), len(e.Assertions))
					return
				}
				for _, a := range e.Assertions {
					if !info.assertions[a] {
						add("ASSERTION_MISMATCH: %s: pass expectation for %s names unregistered assertion %s", where, refKey(e.Test), a)
					}
				}
			case protocol.OutcomeAssertionFail:
				if len(e.Assertions) == 0 {
					add("ASSERTION_MISMATCH: %s: assertion-fail expectation for %s names an empty assertion set", where, refKey(e.Test))
					return
				}
				for _, a := range e.Assertions {
					if !info.assertions[a] {
						add("ASSERTION_MISMATCH: %s: assertion-fail expectation for %s names unregistered assertion %s", where, refKey(e.Test), a)
					}
				}
			}
		}
		baseline := map[testKey]Expectation{}
		for _, e := range m.RedExpectations {
			k := refKey(e.Test)
			if _, dup := baseline[k]; dup {
				add("ASSERTION_MISMATCH: manifest %s has duplicate baseline expectation for %s", m.ID, k)
				continue
			}
			baseline[k] = e
			if !own[k] {
				add("INVALID_SCHEMA: manifest %s red_expectations references %s outside this milestone's own suites", m.ID, k)
			}
			checkExp(e, "manifest "+m.ID+" red_expectations")
		}
		for k := range own {
			if _, ok := baseline[k]; !ok {
				add("ASSERTION_MISMATCH: manifest %s is missing the baseline expectation for own leaf %s; no partial silently omitted suite", m.ID, k)
			}
		}
		// variants: sources finalized, expected coverage of own leaves plus
		// declared targets
		for _, v := range m.Variants {
			if v.Source == "" {
				add("MISSING_CONTRACT: finalized variant %s must declare its source reference", v.ID)
			}
			cover := map[testKey]Expectation{}
			for _, e := range v.Expected {
				k := refKey(e.Test)
				if _, dup := cover[k]; dup {
					add("ASSERTION_MISMATCH: variant %s has duplicate expectation for %s", v.ID, k)
					continue
				}
				cover[k] = e
				checkExp(e, "variant "+v.ID)
			}
			need := map[testKey]bool{}
			for k := range own {
				need[k] = true
			}
			for _, r := range v.TargetTests {
				need[refKey(r)] = true
			}
			for k := range need {
				if _, ok := cover[k]; !ok {
					add("ASSERTION_MISMATCH: variant %s is missing the expectation for %s; every leaf has exactly one expectation in every run", v.ID, k)
				}
			}
			for k := range cover {
				if !need[k] {
					add("ASSERTION_MISMATCH: variant %s carries an expectation for %s which is neither an own leaf nor a declared target", v.ID, k)
				}
			}
		}
		// red_controls: exact bijection with the expected baseline failures
		type failKey struct {
			ref       testKey
			assertion string
		}
		fails := map[failKey]bool{}
		for _, e := range m.RedExpectations {
			if e.Outcome != protocol.OutcomeAssertionFail {
				continue
			}
			for _, a := range e.Assertions {
				fails[failKey{refKey(e.Test), a}] = true
			}
		}
		variantsByID := map[string]*Variant{}
		for vi := range m.Variants {
			variantsByID[m.Variants[vi].ID] = &m.Variants[vi]
		}
		controls := map[failKey]string{}
		for _, rc := range m.RedControls {
			k := failKey{refKey(rc.Test), rc.Assertion}
			if prev, dup := controls[k]; dup {
				add("ASSERTION_MISMATCH: red_control for %s/%s maps more than once (%s, %s); each baseline failure maps exactly once", k.ref, k.assertion, prev, rc.SafeVariant)
				continue
			}
			controls[k] = rc.SafeVariant
			if !fails[k] {
				add("ASSERTION_MISMATCH: red_control for %s/%s does not correspond to an expected baseline assertion failure", k.ref, k.assertion)
			}
			sv, ok := variantsByID[rc.SafeVariant]
			if !ok {
				add("MISSING_TEST: red_control for %s/%s cites undeclared variant %s", k.ref, k.assertion, rc.SafeVariant)
				continue
			}
			if sv.Kind != protocol.VariantSafeControl {
				add("INVALID_SCHEMA: red_control for %s/%s cites variant %s which is not a safe-control", k.ref, k.assertion, rc.SafeVariant)
				continue
			}
			passOK := false
			for _, e := range sv.Expected {
				if refKey(e.Test) == k.ref && e.Outcome == protocol.OutcomePass {
					for _, a := range e.Assertions {
						if a == k.assertion {
							passOK = true
						}
					}
				}
			}
			if !passOK {
				add("ASSERTION_MISMATCH: safe-control variant %s must expect %s to pass at assertion %s under the identical frozen closure", rc.SafeVariant, k.ref, k.assertion)
			}
		}
		for k := range fails {
			if _, ok := controls[k]; !ok {
				add("MISSING_CONTRACT: expected baseline assertion failure %s/%s has no red_control safe-control calibration", k.ref, k.assertion)
			}
		}
		// complete safe/unsafe pairs
		pairs := map[string][]*Variant{}
		for vi := range m.Variants {
			v := &m.Variants[vi]
			if v.Pair != "" {
				pairs[v.Pair] = append(pairs[v.Pair], v)
			}
		}
		targetsOf := func(v *Variant) map[testKey]bool {
			out := map[testKey]bool{}
			for _, r := range v.TargetTests {
				out[refKey(r)] = true
			}
			return out
		}
		sameTargets := func(a, b *Variant) bool {
			ta, tb := targetsOf(a), targetsOf(b)
			if len(ta) != len(tb) {
				return false
			}
			for k := range ta {
				if !tb[k] {
					return false
				}
			}
			return true
		}
		completePairTargets := map[testKey]bool{}
		for pid, members := range pairs {
			if len(members) != 2 || members[0].Kind == members[1].Kind {
				add("INVALID_SCHEMA: pair %s must have exactly one safe-control and one unsafe-challenge member", pid)
				continue
			}
			var safe, unsafe *Variant
			if members[0].Kind == protocol.VariantSafeControl {
				safe, unsafe = members[0], members[1]
			} else {
				safe, unsafe = members[1], members[0]
			}
			if !sameTargets(safe, unsafe) {
				add("INVALID_SCHEMA: pair %s members must share identical target sets", pid)
				continue
			}
			if safe.Source == unsafe.Source || safe.Source == "" || unsafe.Source == "" {
				add("INVALID_SCHEMA: pair %s members must have distinct subject digests", pid)
			}
			if unsafe.Source == m.Baseline && unsafe.Source != "" {
				add("INVALID_SCHEMA: pair %s unsafe challenge equals the baseline subject: a no-op challenge fails validation", pid)
			}
			for k := range targetsOf(safe) {
				completePairTargets[k] = true
				safeExp, uExp := expectationFor(safe, k), expectationFor(unsafe, k)
				if uExp == nil || uExp.Outcome != protocol.OutcomeAssertionFail || len(uExp.Assertions) == 0 {
					add("ASSERTION_MISMATCH: pair %s unsafe member must expect target %s to assertion-fail at a registered assertion", pid, k)
				}
				if safeExp == nil || safeExp.Outcome != protocol.OutcomePass {
					add("ASSERTION_MISMATCH: pair %s safe member must expect target %s to pass", pid, k)
				}
			}
			// non-target outcomes identical between members
			seenKeys := map[testKey]bool{}
			for _, e := range append(append([]Expectation{}, safe.Expected...), unsafe.Expected...) {
				seenKeys[refKey(e.Test)] = true
			}
			for k := range seenKeys {
				if targetsOf(safe)[k] {
					continue
				}
				se, ue := expectationFor(safe, k), expectationFor(unsafe, k)
				if se == nil || ue == nil || se.Outcome != ue.Outcome || !sameStrings(se.Assertions, ue.Assertions) {
					add("ASSERTION_MISMATCH: pair %s non-target expectation for %s differs between safe and unsafe members; additional differing behavior requires its own targeted obligation", pid, k)
				}
			}
		}
		// every negative test is targeted by at least one complete pair, and
		// every milestone has at least one public-boundary challenge
		for _, s := range m.Suites {
			for _, t := range s.Tests {
				if t.Role != protocol.RoleNegative {
					continue
				}
				k := testKey{m.designID, m.ID, s.ID, t.ID}
				if !completePairTargets[k] {
					add("MISSING_CONTRACT: negative test %s has no complete safe/unsafe pair demonstrating sensitivity", k)
				}
			}
		}
		if len(pairs) == 0 {
			add("MISSING_CONTRACT: milestone %s has no safe/unsafe pair challenging its public boundary", m.ID)
		}
		// every control variant is referenced: pair members by their pair,
		// pairless safe-controls by at least one red_control
		referenced := map[string]bool{}
		for pid := range pairs {
			for _, v := range pairs[pid] {
				referenced[v.ID] = true
			}
		}
		for _, rc := range m.RedControls {
			referenced[rc.SafeVariant] = true
		}
		for _, v := range m.Variants {
			if v.Kind == protocol.VariantUnsafeChallenge && v.Pair == "" {
				add("INVALID_SCHEMA: unsafe-challenge variant %s must belong to a pair", v.ID)
				continue
			}
			if v.Kind == protocol.VariantSafeControl && v.Pair == "" && !referenced[v.ID] {
				add("MISSING_CONTRACT: safe-control variant %s is unreferenced: every control variant must be referenced by a pair or a red_control", v.ID)
			}
		}
	}

	// acyclic design references: a design graph edge exists for every
	// reference crossing design boundaries; cycles fail
	edges := map[string]map[string]bool{}
	for mi := range manifests {
		m := &manifests[mi]
		if m.designID == "" {
			continue
		}
		cross := func(r TestRef) {
			if r.Design != m.designID {
				if edges[m.designID] == nil {
					edges[m.designID] = map[string]bool{}
				}
				edges[m.designID][r.Design] = true
			}
		}
		for _, o := range m.Obligations {
			for _, r := range o.Positive {
				cross(r)
			}
			for _, r := range o.Negative {
				cross(r)
			}
		}
		for _, e := range m.RedExpectations {
			cross(e.Test)
		}
		for _, v := range m.Variants {
			for _, r := range v.TargetTests {
				cross(r)
			}
			for _, e := range v.Expected {
				cross(e.Test)
			}
		}
		for _, rc := range m.RedControls {
			cross(rc.Test)
		}
	}
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var cycleAt string
	var visit func(n string) bool
	visit = func(n string) bool {
		color[n] = gray
		for next := range edges[n] {
			switch color[next] {
			case gray:
				cycleAt = next
				return true
			case white:
				if visit(next) {
					return true
				}
			}
		}
		color[n] = black
		return false
	}
	nodes := map[string]bool{}
	for from, tos := range edges {
		nodes[from] = true
		for to := range tos {
			nodes[to] = true
		}
	}
	var order []string
	for n := range nodes {
		order = append(order, n)
	}
	sort.Strings(order)
	for _, n := range order {
		if color[n] == white && visit(n) {
			add("INVALID_SCHEMA: design reference cycle detected involving %s; cyclic-child references fail", cycleAt)
			break
		}
	}

	if len(errs) == 0 {
		return nil
	}
	msgs := make([]string, len(errs))
	copy(msgs, errs)
	return errors.New(strings.Join(msgs, "\n"))
}

func expectationFor(v *Variant, k testKey) *Expectation {
	for i := range v.Expected {
		if refKey(v.Expected[i].Test) == k {
			return &v.Expected[i]
		}
	}
	return nil
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := map[string]int{}
	for _, x := range a {
		set[x]++
	}
	for _, x := range b {
		set[x]--
		if set[x] < 0 {
			return false
		}
	}
	return true
}
