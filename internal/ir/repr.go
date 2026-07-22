package ir

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Repr is a Python-style repr() over arbitrary Go values, used to reproduce the
// exact lint error strings (machine_lint uses f"{x!r}" on keys, targets,
// state types, invoke sources, and the polymorphic transition/action values).
func Repr(v interface{}) string {
	if v == nil {
		return "None"
	}
	switch x := v.(type) {
	case *Value:
		return goRepr(x)
	case string:
		return pyReprStr(x)
	case bool:
		if x {
			return "True"
		}
		return "False"
	case int:
		return fmt.Sprintf("%d", x)
	case int64:
		return fmt.Sprintf("%d", x)
	case float64:
		// Python repr(1.0) == '1.0'; repr(1.5) == '1.5'. Match minimally.
		if x == float64(int64(x)) {
			return fmt.Sprintf("%.1f", x)
		}
		return fmt.Sprintf("%g", x)
	case []string:
		parts := make([]string, len(x))
		for i, s := range x {
			parts[i] = pyReprStr(s)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return fmt.Sprintf("%v", v)
}

// GetString reads o[key] as a string ("" if absent or non-string), mirroring
// Python's `o.get("key")` returning None when absent; callers distinguish via
// Has. For lint fields like `node.get("type")` Python returns None when absent;
// use GetStringOr for that (returns ("", true) only when present-and-string).
func (o *Object) GetString(key string) string {
	if v := o.Get2(key); v != nil && v.Kind == KindString {
		return v.AsString()
	}
	return ""
}

// GetOptString returns (value, true) when key is present and a string;
// ("", false) when absent; ("", false) when present-but-not-string (caller
// should treat non-string like Python None for repr purposes via Has).
func (o *Object) GetOptString(key string) (string, bool) {
	v, ok := o.Get(key)
	if !ok || v == nil || v.Kind != KindString {
		return "", false
	}
	return v.AsString(), true
}

// GetObject returns the child object value, or nil.
func (o *Object) GetObject(key string) *Object {
	if v := o.Get2(key); v != nil && v.Kind == KindObject {
		return v.AsObject()
	}
	return nil
}

// IsUpperFirst reports whether the first rune is uppercase (Python
// n[:1].isupper()). Unicode-aware: an ASCII-only check exempted non-ASCII
// state names from every uppercase-keyed rule.
func IsUpperFirst(s string) bool {
	r, size := utf8.DecodeRuneInString(s)
	if size == 0 || r == utf8.RuneError && size == 1 {
		return false
	}
	return unicode.IsUpper(r)
}
