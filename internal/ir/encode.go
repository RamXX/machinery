package ir

import (
	"bytes"
	"encoding/json"
)

// Marshal renders a value as JSON with object members in their order,
// two-space indentation, no HTML escaping and a trailing newline: the way
// every generated JSON file is written.
func Marshal(v *Value) ([]byte, error) {
	var compact bytes.Buffer
	if err := encodeCompact(&compact, v); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, compact.Bytes(), "", "  "); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

// Compact renders a value as compact JSON, object members in their order.
func Compact(v *Value) string {
	var b bytes.Buffer
	if encodeCompact(&b, v) != nil {
		return ""
	}
	return b.String()
}

func encodeString(b *bytes.Buffer, s string) error {
	enc := json.NewEncoder(b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return err
	}
	b.Truncate(b.Len() - 1) // Encode ends with a newline
	return nil
}

func encodeCompact(b *bytes.Buffer, v *Value) error {
	if v == nil {
		b.WriteString("null")
		return nil
	}
	switch v.Kind {
	case KindObject:
		b.WriteByte('{')
		o := v.AsObject()
		for i, k := range o.Keys() {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := encodeString(b, k); err != nil {
				return err
			}
			b.WriteByte(':')
			if err := encodeCompact(b, o.Get2(k)); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	case KindArray:
		b.WriteByte('[')
		for i, e := range v.AsArray() {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := encodeCompact(b, e); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case KindString:
		return encodeString(b, v.AsString())
	case KindNumber:
		b.WriteString(string(v.AsNumber()))
	case KindBool:
		if x, _ := v.AsBool(); x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	default:
		b.WriteString("null")
	}
	return nil
}
