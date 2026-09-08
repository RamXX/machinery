---
id: MAC-86fr
title: "processscope writeFrame fails on frames larger than the unix socket send buffer (short control write)"
status: open
priority: 1
type: bug
labels: [custody, processscope]
created_at: 2026-09-08T07:58:35Z
created_by: ramirosalas
updated_at: 2026-09-08T07:58:35Z
content_hash: "sha256:9b3c9f2f22b5e91e9b300214335f8239055fc6823c9c36905bee98dea2bc65ca"
---

## Description
## Symptom
With a large inherited shell environment (measured: 7,597 bytes, 87 variables, PATH 3,157 bytes) every `TestContributorLaneExecutes*ConformanceFragment` test in `internal/tdd/adapters` fails deterministically on macOS, under race and non-race, at the pristine 0.7.0 tip (4bc4dad) and earlier (65183fc). Under `env -i PATH HOME TMPDIR USER SHELL LANG` the package passes under race (414 s).

## Root cause (diagnosed 2026-09-08 during the 0.7.0 release preparation)
The custody frame that carries `Env: os.Environ()` for the `go build` job exceeds macOS's 8,192-byte unix-stream send buffer. `WriteMsgUnix` returns a short write, `processscope.writeFrame` (`internal/processscope/guardian_unix.go:122`) reports "short control write" with no partial-write loop, and the scope close then times out at 45 s.

## Expected
Frames larger than the socket send buffer are written completely (loop on partial writes, or raise the send buffer with SO_SNDBUF, or chunk the frame), and the lane test helper launches with a closed minimal environment so the test does not depend on the contributor's shell.

## Impact
Any contributor or CI runner with a large environment cannot run the adapters package; production custody could hit the same short write for any guarded job with a large environment. Hosted CI environments are small, so CI is green today.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
