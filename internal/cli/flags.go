package cli

import "flag"

// ParseMixed parses flags that may come before or after positional
// arguments. Everything after "--" is positional. ok is false on a usage error.
func (e *Env) ParseMixed(fs *flag.FlagSet, args []string) (pos []string, ok bool) {
	for {
		if err := fs.Parse(args); err != nil {
			return nil, false
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return pos, true
		}
		// flag.Parse stops at "--" (consumed) or at the first non-flag.
		consumed := len(args) - len(rest)
		if consumed > 0 && args[consumed-1] == "--" {
			return append(pos, rest...), true
		}
		pos = append(pos, rest[0])
		args = rest[1:]
	}
}
