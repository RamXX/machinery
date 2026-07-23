package main

import (
	"fmt"
	"path/filepath"
	"testing"
)

// TestGoldenCheckExternalChecker pins the Gk gate output for the pii-flow
// example. The committed projection and the Souffle-produced evidence let
// `check --gate gk` pass with no engine present, so this runs hermetically and
// byte-for-byte, the same way the rest of the corpus does. The engine phase
// (verify-checkers) needs souffle and is deliberately not part of the corpus,
// exactly as verify-formal (TLC, needs Java) is not.
func TestGoldenCheckExternalChecker(t *testing.T) {
	root := repoRootDir(t)
	out, errS, code := runBin(t, "check", filepath.Join(root, "examples", "pii-flow", "design"), "--gate", "gk")
	g := goldenDir(t, "check-pii-flow-gk")
	compareOrUpdate(t, filepath.Join(g, "stdout.txt"), out)
	compareOrUpdate(t, filepath.Join(g, "stderr.txt"), errS)
	compareOrUpdate(t, filepath.Join(g, "exitcode.txt"), fmt.Sprintf("%d\n", code))
}
