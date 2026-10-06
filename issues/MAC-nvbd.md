---
id: MAC-nvbd
title: "install: TestBootstrapReceiptCLI fails on darwin/arm64 at v0.10.3 (later-target-failure rollback not restored)"
status: open
priority: 2
type: bug
labels: [install, tests]
created_at: 2026-10-06T02:35:36Z
created_by: ramirosalas
updated_at: 2026-10-06T02:35:36Z
content_hash: "sha256:f03f40dab14c17c7e71e3af107ff28946e98738358562a9d25d274c23a0b463b"
---

## Description
Fails on unchanged main (9c7263bd, v0.10.3) as well as on story/MAC-52sf, so it predates the threat-driven change. Subtest missing_prestate_later_target_failure: bootstrap_receipt_test.go:562 'did not reach intended later source failure' (the update to fixture v9.9.2 completed every home instead of failing on the later target), then :565 'later target failure did not restore old binary, all homes/native targets, and receipt' and :569 'original absence was not restored'. Package run took 1682s on a loaded host. Repro: go test -count=1 -timeout 60m -run TestBootstrapReceiptCLI ./internal/install/

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
