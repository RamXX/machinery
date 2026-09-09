---
id: MAC-uwbw
title: "update: Claude Code marketplace refresh rejected as non-canonical output (Refreshing marketplace cache line)"
status: in_progress
priority: 0
type: bug
labels: [install, update, claude-code, field-defect]
created_at: 2026-09-09T17:31:46Z
created_by: ramirosalas
updated_at: 2026-09-09T17:48:03Z
content_hash: "sha256:104ccccd23122269e4b69421cef22b38776e07ea2d099790f54b7f6a9fd53f15"
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
- 2026-09-09T17:36:55Z status: open -> in_progress

## Links


## Comments

### 2026-09-09T17:36:55Z ramirosalas
Root cause: internal/install/update.go claudeMarketplaceUpdateOutputRE pinned the exact marketplace-update lines ('Updating marketplace: machinery...', optional 'Validating local marketplace', '✔ Successfully updated marketplace: machinery'); Claude Code 2.1.266 prints 'Refreshing marketplace cache (timeout: 120s)…' glued to the banner, so a successful refresh (exit 0) was rejected as non-canonical. Fix on main at 1e97df0 (0.7.2 stamp): validateClaudeMarketplaceUpdateOutput requires the banner first and the exact success line last and rejects failure markers on intermediate lines; the 2.1.266 output is pinned as claudeMarketplaceSuccessOutput2_1_266 with five rejected shapes in TestPluginMutationSuccessOutputContracts. Field verification after the v0.7.2 release: machinery update --version v0.7.2 run twice on this host (first parent 0.7.1 still rejects, second parent 0.7.2 must refresh the plugin).

### 2026-09-09T17:48:03Z ramirosalas
Owner ruling 2026-09-09: no second external release on the day of 0.7.1. The fix and the 0.7.2 stamp are on main (1e97df0, hosted CI running) but v0.7.2 is not tagged; tag on a later day after the field verification (machinery update --version v0.7.2 twice on the real host). Until then the 0.7.1 workaround is: run 'claude plugin marketplace update machinery' and 'claude plugin update machinery@machinery --scope user' by hand after machinery update.
