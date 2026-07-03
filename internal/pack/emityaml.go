package pack

// A minimal, deterministic Value->YAML emitter for the pack's domain slice.
// Emits in the Value's insertion order with plain scalars quoted only when
// needed, so regenerating a pack is byte-stable.

import (
	"fmt"
	"strings"

	"github.com/ramirosalas/machinery/internal/ir"
)

func yamlScalar(v *ir.Value) string {
	switch v.Kind {
	case ir.KindString:
		return yamlQuote(v.AsString())
	case ir.KindNumber:
		return string(v.AsNumber())
	case ir.KindBool:
		b, _ := v.AsBool()
		if b {
			return "true"
		}
		return "false"
	case ir.KindNull:
		return "null"
	}
	return ""
}

// yamlQuote double-quotes a string when a plain scalar would be ambiguous.
func yamlQuote(s string) string {
	if s == "" {
		return `""`
	}
	plain := true
	for _, r := range s {
		if strings.ContainsRune(":#{}[]&*!|>'\"%@`,\n\t", r) {
			plain = false
			break
		}
	}
	switch strings.ToLower(s) {
	case "true", "false", "null", "yes", "no", "on", "off", "~":
		plain = false
	}
	if plain && (s[0] == ' ' || s[len(s)-1] == ' ' || s[0] == '-' || s[0] == '?') {
		plain = false
	}
	if plain {
		return s
	}
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return `"` + s + `"`
}

func indent(n int) string { return strings.Repeat("  ", n) }

// emitYAML writes `key: <value>` at the given indent level, recursing into
// objects and arrays.
func emitYAML(b *strings.Builder, level int, key string, v *ir.Value) {
	pre := indent(level)
	switch {
	case v == nil:
		fmt.Fprintf(b, "%s%s: null\n", pre, yamlQuote(key))
	case v.Kind == ir.KindObject:
		o := v.AsObject()
		if o.Len() == 0 {
			fmt.Fprintf(b, "%s%s: {}\n", pre, yamlQuote(key))
			return
		}
		fmt.Fprintf(b, "%s%s:\n", pre, yamlQuote(key))
		for _, k := range o.Keys() {
			emitYAML(b, level+1, k, o.Get2(k))
		}
	case v.Kind == ir.KindArray:
		arr := v.AsArray()
		if len(arr) == 0 {
			fmt.Fprintf(b, "%s%s: []\n", pre, yamlQuote(key))
			return
		}
		fmt.Fprintf(b, "%s%s:\n", pre, yamlQuote(key))
		for _, e := range arr {
			emitYAMLListItem(b, level+1, e)
		}
	default:
		fmt.Fprintf(b, "%s%s: %s\n", pre, yamlQuote(key), yamlScalar(v))
	}
}

// emitYAMLListItem writes one `- ...` sequence entry at the given level.
func emitYAMLListItem(b *strings.Builder, level int, v *ir.Value) {
	pre := indent(level)
	switch {
	case v == nil:
		fmt.Fprintf(b, "%s- null\n", pre)
	case v.Kind == ir.KindObject:
		o := v.AsObject()
		if o.Len() == 0 {
			fmt.Fprintf(b, "%s- {}\n", pre)
			return
		}
		first := true
		for _, k := range o.Keys() {
			cv := o.Get2(k)
			lead := pre + "  "
			if first {
				lead = pre + "- "
				first = false
			}
			switch {
			case cv != nil && cv.Kind == ir.KindObject && cv.AsObject().Len() > 0:
				fmt.Fprintf(b, "%s%s:\n", lead, yamlQuote(k))
				for _, ck := range cv.AsObject().Keys() {
					emitYAML(b, level+2, ck, cv.AsObject().Get2(ck))
				}
			case cv != nil && cv.Kind == ir.KindArray && len(cv.AsArray()) > 0:
				fmt.Fprintf(b, "%s%s:\n", lead, yamlQuote(k))
				for _, e := range cv.AsArray() {
					emitYAMLListItem(b, level+2, e)
				}
			case cv != nil && cv.Kind == ir.KindObject:
				fmt.Fprintf(b, "%s%s: {}\n", lead, yamlQuote(k))
			case cv != nil && cv.Kind == ir.KindArray:
				fmt.Fprintf(b, "%s%s: []\n", lead, yamlQuote(k))
			default:
				fmt.Fprintf(b, "%s%s: %s\n", lead, yamlQuote(k), yamlScalar(cv))
			}
		}
	case v.Kind == ir.KindArray:
		fmt.Fprintf(b, "%s-\n", pre)
		for _, e := range v.AsArray() {
			emitYAMLListItem(b, level+1, e)
		}
	default:
		fmt.Fprintf(b, "%s- %s\n", pre, yamlScalar(v))
	}
}
