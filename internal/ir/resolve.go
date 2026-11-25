package ir

import (
	"fmt"
	"regexp"
	"strings"
)

var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Resolver implements XState v5 target resolution over one machine:
//
//   - a bare-name target resolves only among siblings of the source state
//     (the source itself included, for explicit self-transitions);
//   - "#<machine id or state id>" resolves to that node, and "#<id>.<path>"
//     resolves every path segment strictly under that node;
//   - ".child" relative targets are outside the subset and rejected;
//   - anything else is dangling, never a fallback.
type Resolver struct {
	statesVal *Value
	PathSet   map[string]bool
	NodeOf    map[string]*Value
	IDs       map[string]string // custom state id -> full dotted path
	RootID    string
	Initial   string
	IDErrs    []string // id-registration findings
}

// NewResolver indexes a machine.
func NewResolver(m *Value) *Resolver {
	r := &Resolver{PathSet: map[string]bool{}, NodeOf: map[string]*Value{}, IDs: map[string]string{}}
	ro := m.AsObject()
	if ro == nil {
		return r
	}
	r.RootID = ro.GetString("id")
	r.Initial = ro.GetString("initial")
	r.statesVal = ro.Get2("states")
	for _, s := range WalkStates(r.statesVal, "") {
		r.PathSet[s.Path] = true
		r.NodeOf[s.Path] = s.Node
		if s.Node == nil || s.Node.Kind != KindObject {
			continue
		}
		idv := s.Node.AsObject().Get2("id")
		if idv == nil {
			continue
		}
		if idv.Kind != KindString || idv.AsString() == "" {
			r.IDErrs = append(r.IDErrs, fmt.Sprintf("state %s id must be a non-empty string", s.Path))
			continue
		}
		id := idv.AsString()
		if !identRe.MatchString(id) {
			r.IDErrs = append(r.IDErrs, fmt.Sprintf("state %s id %s is not a valid identifier (a dotted or symbolic id can never be referenced by #id.path)", s.Path, Repr(id)))
			continue
		}
		if r.RootID != "" && id == r.RootID {
			r.IDErrs = append(r.IDErrs, fmt.Sprintf("state %s id %s collides with the machine id", s.Path, Repr(id)))
			continue
		}
		if prev, dup := r.IDs[id]; dup {
			r.IDErrs = append(r.IDErrs, fmt.Sprintf("duplicate state id %s (declared on %s and %s)", Repr(id), prev, s.Path))
			continue
		}
		r.IDs[id] = s.Path
	}
	return r
}

// Resolve returns (destPath, "") on success or ("", why) on failure. An empty
// target is an internal transition and resolves to srcPath.
func (r *Resolver) Resolve(tgt, srcPath string) (string, string) {
	if tgt == "" {
		return srcPath, ""
	}
	if strings.HasPrefix(tgt, ".") {
		return "", fmt.Sprintf("relative target %s is outside the supported subset (use a sibling name or #id.path.to.state)", Repr(tgt))
	}
	if strings.HasPrefix(tgt, "#") {
		rest := strings.TrimPrefix(tgt, "#")
		if rest == "" {
			return "", fmt.Sprintf("dangling target %s (empty id reference)", Repr(tgt))
		}
		segs := strings.Split(rest, ".")
		var cur string // "" means the machine root
		head := segs[0]
		if p, ok := r.IDs[head]; ok {
			cur = p
		} else if r.RootID != "" && head == r.RootID {
			cur = ""
		} else {
			return "", fmt.Sprintf("dangling target %s (%s is neither the machine id nor a declared state id)", Repr(tgt), Repr(head))
		}
		for _, seg := range segs[1:] {
			var children *Value
			if cur == "" {
				children = r.statesVal
			} else if n := r.NodeOf[cur]; n != nil && n.Kind == KindObject {
				children = n.AsObject().Get2("states")
			}
			if children == nil || children.Kind != KindObject || !children.AsObject().Has(seg) {
				at := cur
				if at == "" {
					at = "the machine root"
				}
				return "", fmt.Sprintf("dangling target %s (%s has no child state %s)", Repr(tgt), at, Repr(seg))
			}
			if cur == "" {
				cur = seg
			} else {
				cur = cur + "." + seg
			}
		}
		if cur == "" {
			if r.Initial != "" && r.PathSet[r.Initial] {
				return r.Initial, ""
			}
			return "", ""
		}
		return cur, ""
	}
	if strings.Contains(tgt, ".") {
		return "", fmt.Sprintf("dangling target %s (dotted targets must use the #id.path.to.state form)", Repr(tgt))
	}
	cand := tgt
	if i := strings.LastIndex(srcPath, "."); i >= 0 {
		cand = srcPath[:i] + "." + tgt
	}
	if r.PathSet[cand] {
		return cand, ""
	}
	return "", fmt.Sprintf("dangling target %s (no sibling of %s is named %s)", Repr(tgt), srcPath, Repr(tgt))
}

// Leaf follows initial children from a state path down to a leaf state.
func (r *Resolver) Leaf(path string) string {
	for depth := 0; depth < 64; depth++ {
		n := r.NodeOf[path]
		if n == nil || n.Kind != KindObject {
			return path
		}
		o := n.AsObject()
		if o.Get2("states") == nil {
			return path
		}
		init := o.GetString("initial")
		if init == "" || !r.PathSet[path+"."+init] {
			return path
		}
		path = path + "." + init
	}
	return path
}
