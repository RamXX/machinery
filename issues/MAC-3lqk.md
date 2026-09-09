---
id: MAC-3lqk
title: "Pin the Claude Code marketplace to release tags so a push to main is not a plugin release"
status: open
priority: 2
type: task
labels: [release-process, claude-code]
created_at: 2026-09-09T18:28:12Z
created_by: ramirosalas
updated_at: 2026-09-09T18:28:12Z
content_hash: "sha256:9e4f77aecff6d62b1e5874d914838a9705c758f2d360ddbdb5efb03f61b30792"
---

## Description
## Why
On 2026-09-09 the 0.7.2 stamp pushed to main (untagged) was served to Claude Code users by the next marketplace refresh, because .claude-plugin/marketplace.json declares source './' on the default branch. A 0.7.1 binary beside a 0.7.2 plugin cache refuses its default-home install ('cached machinery plugin version 0.7.2 does not match running machinery v0.7.1'). The owner wants slower, grouped releases and pushes that are not releases.

## Ask
Make the marketplace serve the tagged release (a ref or a release branch that only moves at tag time), keep the plugin-cache version check consistent with that, document it in docs/claude-plugin.md, and add a repository contract test that the marketplace source cannot silently track main again.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
