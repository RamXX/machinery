---
id: MAC-nvbd
title: "install: TestBootstrapReceiptCLI later-target-failure subtest flakes under heavy host load (passes in isolation)"
status: closed
priority: 3
type: bug
labels: [install, tests]
created_at: 2026-10-06T02:35:36Z
created_by: ramirosalas
updated_at: 2026-10-06T05:53:40Z
content_hash: "sha256:d5287bc03de5a6dbf545c70f2a5b0ad099868f88da5be9d911c7d5325267367e"
closed_at: 2026-10-06T05:53:40Z
close_reason: "Fixed in 0.11.1 (598f48cc RED, 8ea33bdd and 474a44c3 GREEN): bootstrap CLI fixtures use the package deadline instead of fixed 30 s bounds."
---

## Description
Fails on unchanged main (9c7263bd, v0.10.3) as well as on story/MAC-52sf, so it predates the threat-driven change. Subtest missing_prestate_later_target_failure: bootstrap_receipt_test.go:562 'did not reach intended later source failure' (the update to fixture v9.9.2 completed every home instead of failing on the later target), then :565 'later target failure did not restore old binary, all homes/native targets, and receipt' and :569 'original absence was not restored'. Package run took 1682s on a loaded host. Repro: go test -count=1 -timeout 60m -run TestBootstrapReceiptCLI ./internal/install/

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Flaky under heavy host load: failed twice on unchanged v0.10.3 and on 0.11.0 during full-suite runs (1660s+), passes in isolation. Not a product defect; a timing assumption in the test. Evidence: At HEAD 8c620d8f (v0.11.0) `go test -count=1 -v -run 'TestBootstrapReceiptCLI/missing_prestate_later' ./internal/install/` passes both subtests (missing_prestate_later_target_failure_rolls_back_bootstrap_false/true, 36s each, 95s total). internal/install changed since v0.10.3 only by f7c079a1 (install.go, plugin_skip_test.go); no commit names MAC-nvbd. The original report cited a 1682s package run on a loaded host, so a load or timing artifact is likely (cf. MAC-awrq, MAC-y5ho). Notes: Uncertainty: only the named subtests were run, not the whole package under load. Treat as cannot-reproduce rather than fixed-by-commit.

## History
- 2026-10-06T05:53:40Z status: open -> closed

## Links


## Comments
