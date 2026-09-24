// Package rules embeds machinery's shipped methodology rules: the Datalog
// files under consistency/ that the Gy-rules gate evaluates over a design's
// projected facts (docs/consistency-layer-proposal.md, section 3.3). The
// files are data: each runs unchanged under Soufflé, and the parity test in
// internal/gates holds the in-process evaluator to Soufflé's output on every
// bundled example.
package rules

import "embed"

// Consistency holds consistency/*.dl, one file per rule family.
//
//go:embed consistency/*.dl
var Consistency embed.FS

// ConsistencyDir is the directory inside Consistency that holds the files.
const ConsistencyDir = "consistency"
