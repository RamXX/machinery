---
description: Mark this repository as machinery-managed
---

Load the `machinery` skill, configure `.machinery.json`, and run
`machinery preflight`. Preserve an existing configuration unless the user
explicitly changes it. Tell the user, in plain language, that generated
design files are now protected from direct edits here and that the same
checks run in CI; this command is operator-facing, so its own output may
use machinery vocabulary.

Design directory: $ARGUMENTS
