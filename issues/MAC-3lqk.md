---
id: MAC-3lqk
title: "Pin the Claude Code marketplace to release tags so a push to main is not a plugin release"
status: open
priority: 1
type: task
labels: [release-process, claude-code]
created_at: 2026-09-09T18:28:12Z
created_by: ramirosalas
updated_at: 2026-09-24T21:33:45Z
content_hash: "sha256:1aa3d7ec02d7b56fd2f405bcaa65f3a6b54493116a0a554bc885a101424b3339"
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

## History


## Links


## Comments
