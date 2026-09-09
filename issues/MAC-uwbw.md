---
id: MAC-uwbw
title: "update: Claude Code marketplace refresh rejected as non-canonical output (Refreshing marketplace cache line)"
status: open
priority: 0
type: bug
labels: [install, update, claude-code, field-defect]
created_at: 2026-09-09T17:31:46Z
created_by: ramirosalas
updated_at: 2026-09-09T17:31:46Z
content_hash: "sha256:bb37013522e1b4e84af8d536f41d88a5d669388d2cf2b9cc6533524756641c72"
---

## Description
## Symptom
machinery update --version v0.7.1 on 2026-09-09 (parent 0.7.1, Claude Code 2.1.266), after the MAC-9zpf inventory fix let the refresh proceed:

error: Claude Code machinery marketplace refresh failed (Updating marketplace: machinery...Refreshing marketplace cache (timeout: 120s)... Successfully updated marketplace: machinery): non-canonical Claude marketplace success output

The marketplace update itself succeeded (exit 0, final line '✔ Successfully updated marketplace: machinery'); machinery rejected the output because claudeMarketplaceUpdateOutputRE (internal/install/update.go) pins the exact lines and Claude Code now prints a 'Refreshing marketplace cache (timeout: 120s)…' progress line between the first line and the success line. The plugin update output ('Checking for updates ... ✔ Plugin "machinery" updated from 0.7.0 to 0.7.1 for scope user. Restart to apply changes.') still matches.

## Impact
Same field impact as MAC-9zpf: the Claude Code plugin never refreshes through machinery update, the update returns an error and records the obligation. Third closed-format defect in this reader class (0.6.9 plugin cache, 0.7.0 inventory, 0.7.1 marketplace output).

## Ask
Prove success by the final success line and the exit status, treat intermediate lines as Claude Code progress output (reject any line carrying a failure marker), pin the 2.1.266 output as a fixture, ship in 0.7.2.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
