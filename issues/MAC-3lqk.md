---
id: MAC-3lqk
title: "Pin the Claude Code marketplace to release tags so a push to main is not a plugin release"
status: open
priority: 1
type: task
labels: [release-process, claude-code]
created_at: 2026-09-09T18:28:12Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:49Z
content_hash: "sha256:c8bc1958fb145e91bd910fca03ebfd47d65d44dd89cfac170b3ea074e25b06a9"
---

## Description
## Why
On 2026-09-09 the 0.7.2 stamp pushed to main (untagged) was served to Claude Code users by the next marketplace refresh, because .claude-plugin/marketplace.json declares source './' on the default branch. A 0.7.1 binary beside a 0.7.2 plugin cache refuses its default-home install ('cached machinery plugin version 0.7.2 does not match running machinery v0.7.1'). The owner wants slower, grouped releases and pushes that are not releases.

## Ask
Make the marketplace serve the tagged release (a ref or a release branch that only moves at tag time), keep the plugin-cache version check consistent with that, document it in docs/claude-plugin.md, and add a repository contract test that the marketplace source cannot silently track main again.

## Acceptance Criteria


## Design


## Notes
Triage 2026-09-24: raised. marketplace.json source is ./; the 0.10.0 withdraw/revert on main (9c042684, aaac8403) showed the exposure is real.
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Serve the tagged release (ref or release branch moved only at tag time), keep the plugin-cache version check consistent, document in docs/claude-plugin.md, add contract test. Evidence: /.claude-plugin/marketplace.json still declares source './' on the default branch; no repository contract test pins it to a tag or release branch (grep of *_test.go for marketplace finds only install-cache tests). 0.10.0 withdraw/revert (9c042684, aaac8403) shows the exposure. Notes: Also interacts with release-cut policy (bump only in release commits). Fits with MAC-y8lj release workflow if a release branch is chosen.

## History


## Links


## Comments
