package ir

import "strings"

// FailureWords are the name fragments of the name hints
// (docs/formats.md section 6.2).
var FailureWords = []string{"fail", "error", "abort", "crash", "timeout"}

// NoFailureWord says, for a message, that a name has none of them.
const NoFailureWord = "no fail, error, abort, crash or timeout in the name"

// LooksLikeFailure reports whether a name contains a failure word, in any
// case.
func LooksLikeFailure(name string) bool {
	l := strings.ToLower(name)
	for _, w := range FailureWords {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}
