---
id: MAC-9zpf
title: "update: Claude Code plugin inventory rejected for unknown fields (enabled, installPath, installedAt, lastUpdated, mcpServers, version)"
status: open
priority: 0
type: bug
labels: [install, update, claude-code, field-defect]
created_at: 2026-09-09T00:49:00Z
created_by: ramirosalas
updated_at: 2026-09-09T00:49:00Z
content_hash: "sha256:2a7dfcb74bd2b365bca899f30972ea7f066ea3bb6ab3fde29bd8b7510539349b"
---

## Description
## Symptom
machinery update --version v0.7.0 on 2026-09-08 (from v0.6.11, Claude Code current) committed the direct update, then:

```
error: Claude Code plugin inventory was not understood: Claude plugin entry 0 has unknown fields ["enabled" "installPath" "installedAt" "lastUpdated" "mcpServers" "version"]
```

## Cause
The Claude Code plugin inventory reader treats the plugin entry as a closed record and rejects fields the current Claude Code writes (enabled, installPath, installedAt, lastUpdated, mcpServers, version). Same failure class as the 0.6.9 plugin-cache 'unexpected member' defect. Inventory files on this host: /Users/ramirosalas/.claude/plugins/blocklist.json /Users/ramirosalas/.claude/plugins/config.json /Users/ramirosalas/.claude/plugins/installed_plugins.json 

## Impact
Every Claude Code user running machinery update gets a returned error and a recorded refresh obligation; the plugin never refreshes until machinery accepts the host's schema. Field defect in 0.7.0; ship in 0.7.1.

## Ask
Read the inventory as an open record for the fields machinery does not consume (fail closed only on the fields it does consume being absent or malformed), pin a fixture captured from the current Claude Code, and add a contract test that a superset entry is accepted.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
