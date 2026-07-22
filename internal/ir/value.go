// Package ir is the shared intermediate representation ported from machine_lint.py.
//
// The highest-fidelity concern: JSON object key order MUST survive parsing
// (states, and the on/after maps inside a state, are emitted in source order
// downstream). We therefore model JSON values as an ordered map type instead
// of bare map[string]any, which Go does not guarantee to order.
package ir

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Value is a JSON value whose object members preserve source order.
//
//	KindObject  -> *Object
//	KindArray   -> []*Value
//	KindString  -> string
//	KindNumber  -> json.Number (we decode with UseNumber to preserve int-vs-float)
//	KindBool    -> bool
//	KindNull    -> nil
type Value struct {
	Kind Kind
	Data any
}

// Kind tags a Value.
type Kind int

const (
	KindNull Kind = iota
	KindBool
	KindNumber
	KindString
	KindArray
	KindObject
)

// Object is an insertion-ordered string->Value map.
type Object struct {
	keys   []string
	values map[string]*Value
}

// NewObject builds an empty ordered object.
func NewObject() *Object { return &Object{values: map[string]*Value{}} }

// Set inserts or updates a key (appends on new key, preserves slot on update).
func (o *Object) Set(k string, v *Value) {
	if _, ok := o.values[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.values[k] = v
}

// Read methods below are nil-receiver-safe: AsObject() returns nil for any
// non-object Value (including JSON null), and chained reads on that nil must
// degrade to "absent", never SIGSEGV (malformed input is reported by the lint
// layer, not by a crash). Set stays panicking: mutating nil is a program bug.

// Get returns (value, true) or (nil, false).
func (o *Object) Get(k string) (*Value, bool) {
	if o == nil {
		return nil, false
	}
	v, ok := o.values[k]
	return v, ok
}

// Has reports key presence.
func (o *Object) Has(k string) bool { _, ok := o.Get(k); return ok }

// Delete removes a key (no-op if absent), preserving remaining order.
func (o *Object) Delete(k string) {
	if o == nil {
		return
	}
	if _, ok := o.values[k]; !ok {
		return
	}
	delete(o.values, k)
	for i, key := range o.keys {
		if key == k {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			return
		}
	}
}

// Keys returns the insertion-ordered keys.
func (o *Object) Keys() []string {
	if o == nil {
		return nil
	}
	return append([]string{}, o.keys...)
}

// Len is the number of members.
func (o *Object) Len() int {
	if o == nil {
		return 0
	}
	return len(o.keys)
}

// Iter calls fn(key,value) in source order; stop early by returning false.
func (o *Object) Iter(fn func(k string, v *Value) bool) {
	if o == nil {
		return
	}
	for _, k := range o.keys {
		if !fn(k, o.values[k]) {
			return
		}
	}
}

// AsObject unwraps an object Value, or returns nil.
func (v *Value) AsObject() *Object {
	if v != nil && v.Kind == KindObject {
		o, _ := v.Data.(*Object)
		return o
	}
	return nil
}

// AsArray unwraps an array Value, or returns nil.
func (v *Value) AsArray() []*Value {
	if v != nil && v.Kind == KindArray {
		a, _ := v.Data.([]*Value)
		return a
	}
	return nil
}

// AsString unwraps a string Value, or returns "".
func (v *Value) AsString() string {
	if v != nil && v.Kind == KindString {
		s, _ := v.Data.(string)
		return s
	}
	return ""
}

// AsNumber unwraps a number Value as its raw textual form.
func (v *Value) AsNumber() json.Number {
	if v != nil && v.Kind == KindNumber {
		n, _ := v.Data.(json.Number)
		return n
	}
	return ""
}

// AsBool unwraps a bool Value.
func (v *Value) AsBool() (bool, bool) {
	if v != nil && v.Kind == KindBool {
		b, _ := v.Data.(bool)
		return b, true
	}
	return false, false
}

// IsNull reports a null Value (also true for a nil *Value).
func (v *Value) IsNull() bool { return v == nil || v.Kind == KindNull }

// ObjectValue wraps an *Object into a Value.
func ObjectValue(o *Object) *Value { return &Value{Kind: KindObject, Data: o} }

// ArrayValue wraps a slice of *Value into a Value.
func ArrayValue(a []*Value) *Value { return &Value{Kind: KindArray, Data: a} }

// StringValue builds a string Value.
func StringValue(s string) *Value { return &Value{Kind: KindString, Data: s} }

// BoolValue builds a bool Value.
func BoolValue(b bool) *Value { return &Value{Kind: KindBool, Data: b} }

// NumberValue builds a number Value from a json.Number string.
func NumberValue(n json.Number) *Value { return &Value{Kind: KindNumber, Data: n} }

// NullValue is the null singleton.
func NullValue() *Value { return &Value{Kind: KindNull} }

// orderedDecode parses JSON preserving object key order, using json.Decoder
// token stream. Numbers stay as json.Number so int-vs-float formatting is
// preserved when re-emitted.
func orderedDecode(dec *json.Decoder) (*Value, error) {
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return decodeToken(dec, t)
}

func decodeToken(dec *json.Decoder, t json.Token) (*Value, error) {
	switch x := t.(type) {
	case json.Delim:
		switch x {
		case '{':
			o := NewObject()
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, _ := kt.(string)
				if o.Has(key) {
					// last-wins would silently drop the first definition; a
					// duplicated state or event key is a hard parse error
					return nil, fmt.Errorf("duplicate key %s in object", pyReprStr(key))
				}
				val, err := orderedDecode(dec)
				if err != nil {
					return nil, err
				}
				o.Set(key, val)
			}
			if _, err := dec.Token(); err != nil { // consume '}'
				return nil, err
			}
			return ObjectValue(o), nil
		case '[':
			var arr []*Value
			for dec.More() {
				val, err := orderedDecode(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, val)
			}
			if _, err := dec.Token(); err != nil { // consume ']'
				return nil, err
			}
			return ArrayValue(arr), nil
		}
	}
	// scalar token
	switch x := t.(type) {
	case string:
		return StringValue(x), nil
	case bool:
		return BoolValue(x), nil
	case json.Number:
		return NumberValue(x), nil
	case nil:
		return NullValue(), nil
	}
	return nil, fmt.Errorf("ir: unexpected token %v", t)
}

// LoadMachineJSON reads+parses a *.machine.json preserving key order.
// Returns (root, nil) on success or (nil, err) mirroring machine_lint.load_machine
// error strings ("invalid JSON in <path>: line N: <msg>" / "cannot read ...").
func LoadMachineJSON(path string) (*Value, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, derr := orderedDecode(dec)
	if derr != nil {
		// json.SyntaxError carries Offset; translate to a 1-based line like Python.
		var se *json.SyntaxError
		if errors.As(derr, &se) {
			line := 1 + bytes.Count(data[:se.Offset], []byte("\n"))
			return nil, fmt.Errorf("invalid JSON in %s: line %d: %s", path, line, se.Error())
		}
		return nil, fmt.Errorf("invalid JSON in %s: %w", path, derr)
	}
	// reject trailing content after the first value (json.load "Extra data")
	if t, err := dec.Token(); err == nil {
		return nil, fmt.Errorf("invalid JSON in %s: extra data after the machine object (%v)", path, t)
	} else if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("invalid JSON in %s: extra data after the machine object", path)
	}
	return v, nil
}

// SortedKeys returns o.Keys() sorted by code point (Python sorted() semantics).
func SortedKeys(o *Object) []string {
	ks := o.Keys()
	sort.Strings(ks)
	return ks
}

// SortStringsInPlace sorts s by code point (matches Python sorted()).
func SortStringsInPlace(s []string) { sort.Strings(s) }

// JoinNonEmpty is ", ".join(xs filtered to non-empty).
func JoinNonEmpty(xs []string) string {
	var out []string
	for _, x := range xs {
		if x != "" {
			out = append(out, x)
		}
	}
	return strings.Join(out, ", ")
}

// Indent1 reproduces json.dump(m, indent=1) two-space? No: Python indent=1 means
// 1-space-per-level. Not used for machine JSON output; machines are read, not
// re-written by the gate tools.
